package main

// B5 — Historical Stock-Movement Repair Toolkit (DRAFT + SCRIPTS ONLY, Wave 9.8).
//
// Bug (see stock_adjustment_double_post_test.go): before the Wave 9.7 fix,
// CreateStockAdjustment AND ApproveStockAdjustment each posted a StockMovement
// for the same adjustment, double-applying the variance to on-hand quantity.
// Both legacy posts carry MovementType="Adjustment" and an EMPTY reference_id
// (the 9.7 fix is the first version that stamps ReferenceType="StockAdjustment",
// ReferenceID=adjustment.ID — see ApproveStockAdjustment in
// app_accounting_inventory.go). This toolkit finds pre-9.7 doubled movements,
// reverses the EXTRA post(s) with a compensating movement (never rewrites or
// deletes history — Article III), and verifies the reversal is exact.
//
// This is a four-stage operator runbook. Every stage is gated behind an env
// var and skips instantly (no DB access, no writes) when that var is unset,
// so a bare `go test ./...` is always a no-op for these files.
//
//   1. DIAGNOSE (this file)      B2_STOCK_DIAGNOSE=1
//        Read-only. Lists candidate doubled-movement groups per DB.
//   2. DRY-RUN                   B2_STOCK_COMMIT=1 B2_STOCK_DRYRUN=1 B2_STOCK_DB_PATH=...
//        Logs exactly what the repair WOULD post, writes nothing.
//   3. SNAPSHOT + REPAIR         B2_STOCK_COMMIT=1 B2_STOCK_DB_PATH=...
//        Copies the target DB (+ -wal/-shm sidecars) to a timestamped
//        *.PRE_B2_STOCK_REPAIR_<stamp>.db snapshot, THEN posts one
//        compensating StockMovement per doubled group via the real
//        app.RecordStockMovement (the only safe posting entry point — it
//        recomputes balances, valuation, and stock status atomically).
//        Idempotent: a group whose doubled movement already has a
//        ReferenceType="StockMovementRepair" counter-post is never repaired
//        twice.
//   4. VERIFY                    B2_STOCK_VERIFY=1 B2_STOCK_DB_PATH=...
//        Read-only. Re-runs the heuristic and asserts the repaired items'
//        net QuantityOnHand delta from the repair movements equals exactly
//        the removed duplicate's effect, and that no new un-referenced
//        doubles exist.
//   5. CHECKPOINT                B2_STOCK_CHECKPOINT=1 B2_STOCK_DB_PATH=...
//        Read-only from a data standpoint (folds WAL into the main .db file
//        so the on-disk copy reflects the repair) via
//        PRAGMA wal_checkpoint(TRUNCATE), then force-closes the handle.
//
// Safety guarantees:
//   - No stage runs against any DB unless its env flag is explicitly set.
//   - The repair stage ALWAYS snapshots before it writes, and only ever adds
//     compensating movements — it never UPDATEs or DELETEs an existing
//     stock_movements row (Article III: never delete history).
//   - Compensating movements are dated "now" (MovementDate=time.Now()) —
//     they are never backdated to disguise the timing of the fix.
//   - Only RecordStockMovement (app_accounting_inventory.go) is used to post;
//     hand-inserting a stock_movements row is never done here, so balances,
//     StockStatus, and weighted-average valuation stay consistent.
//   - The repair is idempotent: re-running it against an already-repaired DB
//     posts nothing new for groups that already carry a
//     ReferenceType="StockMovementRepair" counter-post.
//   - This wave (B5) delivers the four scripts only. Nobody has pointed them
//     at a real database yet; that is a separate, explicitly authorized step.

import (
	"net/url"
	"os"
	"strings"
	"testing"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// b2StockDoubleGroup is one candidate group of doubled, un-referenced
// "Adjustment" stock movements: same item, same quantity, same direction,
// within the same minute (the closeness window that keeps genuinely
// different-day repeated adjustments from being flagged).
type b2StockDoubleGroup struct {
	InventoryItemID string
	Quantity        float64
	Direction       string
	MinuteBucket    string
	Count           int64
	IDs             string // comma-separated stock_movements.id list (GROUP_CONCAT)
}

// b2StockDoubleGroupSQL returns the tightened detection heuristic for the
// target's SQL dialect. It groups by (item, quantity, direction,
// movement_date truncated to the minute) rather than by (item, quantity,
// direction) alone, so two legitimately-repeated adjustments made on
// genuinely different dates are NOT flagged as a double. The PostgreSQL
// variant exists because the production deployment's cloud-sync layer is
// PostgreSQL (dual-DB: local SQLite + remote PG) — same shape, dialect
// spellings only (strftime→to_char, GROUP_CONCAT→string_agg).
func b2StockDoubleGroupSQL(target string) string {
	minuteBucket := `strftime('%Y-%m-%d %H:%M', movement_date)`
	idList := `GROUP_CONCAT(id)`
	if b2StockIsPostgres(target) {
		minuteBucket = `to_char(movement_date, 'YYYY-MM-DD HH24:MI')`
		idList = `string_agg(id::text, ',')`
	}
	return `
	SELECT inventory_item_id,
	       quantity,
	       direction,
	       ` + minuteBucket + ` AS minute_bucket,
	       COUNT(*) AS count,
	       ` + idList + ` AS ids
	FROM stock_movements
	WHERE movement_type = 'Adjustment'
	  AND deleted_at IS NULL
	  AND (reference_id IS NULL OR reference_id = '')
	GROUP BY inventory_item_id, quantity, direction, minute_bucket
	HAVING COUNT(*) > 1
	ORDER BY inventory_item_id, minute_bucket
`
}

// b2StockIsPostgres reports whether a B2_STOCK_DB_PATH target is a PostgreSQL
// DSN (the production cloud-sync layer) rather than a SQLite file path.
func b2StockIsPostgres(target string) bool {
	return strings.HasPrefix(target, "postgres://") || strings.HasPrefix(target, "postgresql://")
}

// b2StockTargetLabel returns a log-safe name for a target: file paths pass
// through; PostgreSQL DSNs get their password redacted so credentials never
// land in test output.
func b2StockTargetLabel(target string) string {
	if !b2StockIsPostgres(target) {
		return target
	}
	if u, err := url.Parse(target); err == nil {
		if u.User != nil {
			u.User = url.User(u.User.Username())
		}
		return u.String()
	}
	return "postgres://<unparseable-dsn-redacted>"
}

// b2StockTargetPresent reports whether a target is worth opening: SQLite
// files must exist on disk; a PostgreSQL DSN is always attempted (its
// reachability is only knowable by connecting).
func b2StockTargetPresent(target string) bool {
	if b2StockIsPostgres(target) {
		return true
	}
	_, err := os.Stat(target)
	return err == nil
}

func b2StockCandidateDBPaths() []string {
	if raw := strings.TrimSpace(os.Getenv("B2_STOCK_DB_PATH")); raw != "" {
		var paths []string
		for _, p := range strings.Split(raw, ";") {
			if tp := strings.TrimSpace(p); tp != "" {
				paths = append(paths, tp)
			}
		}
		return paths
	}
	return []string{
		"ph_holdings.db", // project-root working copy
		appDataDatabasePath(),
	}
}

func b2StockOpenReadOnly(target string) (*gorm.DB, error) {
	return b2StockOpen(target, false)
}

// b2StockOpen opens a toolkit target with the right GORM driver: PostgreSQL
// for DSNs (the cloud-sync layer), SQLite for file paths. forWrite adds the
// SQLite busy-timeout/foreign-key options the repair stage needs; PostgreSQL
// needs no equivalent (MVCC + server-side FK enforcement).
func b2StockOpen(target string, forWrite bool) (*gorm.DB, error) {
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	if b2StockIsPostgres(target) {
		return gorm.Open(postgres.Open(target), cfg)
	}
	if forWrite {
		target += "?_foreign_keys=ON&_busy_timeout=5000"
	}
	return gorm.Open(sqlite.Open(target), cfg)
}

// TestB2StockDiagnose is a READ-ONLY inventory of candidate doubled-post
// stock adjustment movements. It mutates nothing. Run with:
//
//	B2_STOCK_DIAGNOSE=1 go test ./... -run TestB2StockDiagnose -v
//
// Target DBs default to the project-root working copy and the live AppData
// copy; override with a ';'-separated B2_STOCK_DB_PATH.
func TestB2StockDiagnose(t *testing.T) {
	if os.Getenv("B2_STOCK_DIAGNOSE") != "1" {
		t.Skip("set B2_STOCK_DIAGNOSE=1 to run the read-only stock-movement double-post diagnostic")
	}

	for _, dbPath := range b2StockCandidateDBPaths() {
		if dbPath == "" {
			continue
		}
		label := b2StockTargetLabel(dbPath)
		if !b2StockTargetPresent(dbPath) {
			t.Logf("---- %s : NOT PRESENT ----", label)
			continue
		}

		db, err := b2StockOpenReadOnly(dbPath)
		if err != nil {
			t.Logf("---- %s : OPEN FAILED (%v) ----", label, err)
			continue
		}

		t.Logf("================ %s ================", label)

		var groups []b2StockDoubleGroup
		if err := db.Raw(b2StockDoubleGroupSQL(dbPath)).Scan(&groups).Error; err != nil {
			t.Logf("query failed for %s: %v", label, err)
			continue
		}

		totalExtra := int64(0)
		for _, g := range groups {
			extra := g.Count - 1
			totalExtra += extra
			t.Logf("  item=%s qty=%.4f dir=%s minute=%s count=%d extra_to_reverse=%d ids=[%s]",
				g.InventoryItemID, g.Quantity, g.Direction, g.MinuteBucket, g.Count, extra, g.IDs)
		}
		t.Logf("PROJECTION: %s -> %d doubled group(s), %d extra movement(s) to reverse",
			label, len(groups), totalExtra)

		if err := closeGormDB(db); err != nil {
			t.Logf("warning: close %s: %v", label, err)
		}
	}
}

// closeGormDB releases the pooled sql.DB handle so a read-only diagnostic
// pass never holds a lingering connection open against a live database file.
func closeGormDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
