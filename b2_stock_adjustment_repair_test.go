package main

// B5 repair stage. See b2_stock_adjustment_diagnostic_test.go for the full
// runbook and safety guarantees. This file performs the ONLY mutating step:
// it snapshots the target DB, then posts one compensating StockMovement per
// doubled group via the real app.RecordStockMovement (the sole safe posting
// entry point — hand-inserting a stock_movements row is never done here).
//
// Article III (never delete history): a doubled movement is never rewritten
// or deleted. The repair adds a NEW, forward-dated compensating movement
// that nets the extra post to zero and carries provenance back to the
// movement it reverses (ReferenceType="StockMovementRepair",
// ReferenceID=<doubled movement's id>).

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

type b2StockMovementRow struct {
	ID              string
	InventoryItemID string
	Quantity        float64
	Direction       string
	UnitCost        float64
	CreatedAt       time.Time
}

// b2StockCopyFileForBackup copies src to dst (overwrites dst if present).
// Used to snapshot a DB before a mutating maintenance run.
func b2StockCopyFileForBackup(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// b2StockExistingRepairCount returns how many compensating repair movements
// already reference the given doubled-movement id. Used as the idempotency
// guard: a movement is never repaired twice.
func b2StockExistingRepairCount(db *gorm.DB, doubledMovementID string) (int64, error) {
	var n int64
	err := db.Raw(`
		SELECT COUNT(*) FROM stock_movements
		WHERE reference_type = 'StockMovementRepair'
		  AND reference_id = ?
		  AND deleted_at IS NULL
	`, doubledMovementID).Scan(&n).Error
	return n, err
}

// b2StockRepairAdminApp builds an App instance authorized to call
// RecordStockMovement (permission "inventory:create") for this maintenance
// run only. It mirrors the pattern used by other B-series manual repair
// scripts in this repo (e.g. manual_payroll_expense_backfill_test.go): a
// synthetic operator identity, not a real user session.
func b2StockRepairAdminApp(db *gorm.DB) *App {
	app := &App{
		db:            db,
		cache:         NewCache(),
		currentUserID: "b2-stock-repair",
		currentUser: &User{
			Base:     Base{ID: "b2-stock-repair"},
			Username: "b2-stock-repair",
			Role:     Role{Permissions: `["*"]`},
		},
	}
	return app
}

// TestB2StockRepair runs the compensating-movement repair against explicit,
// backed-up database(s). It is gated so it never runs in the normal suite.
// Run with, e.g.:
//
//	B2_STOCK_COMMIT=1 B2_STOCK_DB_PATH="ph_holdings.db" go test ./... -run TestB2StockRepair -v
//
// Set B2_STOCK_DRYRUN=1 alongside B2_STOCK_COMMIT=1 to log exactly what
// would be posted without posting anything (no snapshot is taken either,
// since a dry run performs no write to protect against).
//
// Target DBs default to the project-root working copy and the live AppData
// copy; override with a ';'-separated B2_STOCK_DB_PATH. Each real target is
// copied (plus -wal/-shm sidecars) to a timestamped
// *.PRE_B2_STOCK_REPAIR_<stamp>.db snapshot BEFORE any mutation.
func TestB2StockRepair(t *testing.T) {
	if os.Getenv("B2_STOCK_COMMIT") != "1" {
		t.Skip("set B2_STOCK_COMMIT=1 to run the mutating stock-adjustment double-post repair")
	}
	dryRun := os.Getenv("B2_STOCK_DRYRUN") == "1"

	targets := b2StockCandidateDBPaths()
	stamp := time.Now().Format("2006_01_02_150405")

	for _, dbPath := range targets {
		if dbPath == "" {
			continue
		}
		label := b2StockTargetLabel(dbPath)
		if !b2StockTargetPresent(dbPath) {
			t.Logf("---- %s : NOT PRESENT, skipping ----", label)
			continue
		}

		if !dryRun {
			// STEP 1: snapshot BEFORE any write. Never skipped for a real commit.
			if b2StockIsPostgres(dbPath) {
				// A PostgreSQL target (the cloud-sync layer) cannot be
				// file-copied. The operator must take a server-side backup
				// FIRST and acknowledge it explicitly — the repair refuses
				// to write otherwise.
				if os.Getenv("B2_STOCK_PG_BACKUP_ACK") != "1" {
					t.Logf("---- %s : SKIPPED — PostgreSQL target needs an external backup first.", label)
					t.Logf("     Take one with:  pg_dump --format=custom --file=pre_b2_stock_repair_%s.dump \"<dsn>\"", stamp)
					t.Logf("     Then re-run with B2_STOCK_PG_BACKUP_ACK=1 to confirm the backup exists.")
					continue
				}
				t.Logf("PostgreSQL target %s: proceeding on operator's B2_STOCK_PG_BACKUP_ACK=1 (external backup confirmed)", label)
			} else {
				backupPath := fmt.Sprintf("%s.PRE_B2_STOCK_REPAIR_%s.db", strings.TrimSuffix(dbPath, ".db"), stamp)
				if err := b2StockCopyFileForBackup(dbPath, backupPath); err != nil {
					t.Fatalf("backup %s -> %s failed: %v", dbPath, backupPath, err)
				}
				t.Logf("backup written: %s", backupPath)
				for _, ext := range []string{"-wal", "-shm"} {
					if _, err := os.Stat(dbPath + ext); err == nil {
						_ = b2StockCopyFileForBackup(dbPath+ext, backupPath+ext)
					}
				}
			}
		}

		db, err := b2StockOpen(dbPath, true)
		if err != nil {
			t.Fatalf("open %s: %v", label, err)
		}

		var groups []b2StockDoubleGroup
		if err := db.Raw(b2StockDoubleGroupSQL(dbPath)).Scan(&groups).Error; err != nil {
			t.Fatalf("group query %s: %v", label, err)
		}

		app := b2StockRepairAdminApp(db)
		t.Cleanup(app.cache.Stop)

		posted, skippedIdempotent := 0, 0
		t.Logf("==== %s ==== (%d doubled group(s), dry_run=%v)", label, len(groups), dryRun)

		for _, g := range groups {
			ids := strings.Split(g.IDs, ",")
			var rows []b2StockMovementRow
			if err := db.Raw(`
				SELECT id, inventory_item_id, quantity, direction, unit_cost, created_at
				FROM stock_movements
				WHERE id IN (?)
			`, ids).Scan(&rows).Error; err != nil {
				t.Fatalf("fetch group rows for item %s: %v", g.InventoryItemID, err)
			}
			if len(rows) < 2 {
				continue // heuristic drift between the two queries; nothing to reverse
			}

			// Keep the earliest-created row as the legitimate post; every
			// later row in the group is an "extra" to reverse exactly once.
			sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.Before(rows[j].CreatedAt) })
			extras := rows[1:]

			for _, extra := range extras {
				existing, err := b2StockExistingRepairCount(db, extra.ID)
				if err != nil {
					t.Fatalf("idempotency check for %s: %v", extra.ID, err)
				}
				if existing > 0 {
					skippedIdempotent++
					t.Logf("  SKIP (already repaired): movement %s (item=%s qty=%.4f dir=%s)",
						extra.ID, extra.InventoryItemID, extra.Quantity, extra.Direction)
					continue
				}

				opposite := "OUT"
				if extra.Direction == "OUT" {
					opposite = "IN"
				}
				compensating := StockMovement{
					InventoryItemID: extra.InventoryItemID,
					MovementType:    "Adjustment",
					MovementDate:    time.Now(), // never backdated (Article III)
					Quantity:        extra.Quantity,
					Direction:       opposite,
					UnitCost:        extra.UnitCost,
					ReferenceType:   "StockMovementRepair",
					ReferenceID:     extra.ID,
					Notes: fmt.Sprintf(
						"B5 repair: reversing pre-9.7 duplicate Adjustment post %s "+
							"(CreateStockAdjustment+ApproveStockAdjustment double-post bug, fixed Wave 9.7)",
						extra.ID),
				}

				if dryRun {
					t.Logf("  DRY-RUN would post: item=%s qty=%.4f dir=%s (reversing movement %s)",
						compensating.InventoryItemID, compensating.Quantity, compensating.Direction, extra.ID)
					continue
				}

				if _, err := app.RecordStockMovement(compensating); err != nil {
					t.Fatalf("repair post failed for movement %s: %v", extra.ID, err)
				}
				posted++
				t.Logf("  POSTED compensating movement: item=%s qty=%.4f dir=%s (reversing movement %s)",
					compensating.InventoryItemID, compensating.Quantity, compensating.Direction, extra.ID)
			}
		}

		t.Logf("==== %s ==== posted=%d skipped_idempotent=%d dry_run=%v", label, posted, skippedIdempotent, dryRun)

		if err := closeGormDB(db); err != nil {
			t.Logf("warning: close %s: %v", label, err)
		}
	}
}
