package main

// B5 seed stage (test-rig only — NEVER part of the production runbook).
// Provisions a SCRATCH database with the real app schema (AutoMigrate on the
// actual models, so the schema can never drift from the app) and plants a
// known synthetic double-post so the diagnose → dry-run → repair → verify
// runbook can be proven end-to-end on a given engine — in particular
// PostgreSQL, the production deployment's cloud-sync layer.
//
// SAFETY: this stage refuses to run unless the target path/DSN contains the
// substring "scratch". It exists to build throwaway proof databases, and the
// guard makes pointing it at a real database a hard error, not a foot-gun.
//
// Run with, e.g.:
//
//	B2_STOCK_SEED=1 B2_STOCK_DB_PATH="postgres://user:pass@localhost:5432/b2_stock_scratch" \
//	  go test ./... -run TestB2StockSeedScratch -v

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestB2StockSeedScratch(t *testing.T) {
	if os.Getenv("B2_STOCK_SEED") != "1" {
		t.Skip("set B2_STOCK_SEED=1 to seed a synthetic double-post into a SCRATCH database")
	}

	for _, dbPath := range b2StockCandidateDBPaths() {
		if dbPath == "" {
			continue
		}
		label := b2StockTargetLabel(dbPath)
		if !strings.Contains(strings.ToLower(dbPath), "scratch") {
			t.Fatalf("REFUSED: %s — the seed stage only runs against a target containing \"scratch\" (it plants synthetic data)", label)
		}

		db, err := b2StockOpen(dbPath, true)
		if err != nil {
			t.Fatalf("open %s: %v", label, err)
		}

		// Real app models — the scratch schema is generated from the same
		// structs the application migrates, so it cannot drift.
		if err := db.AutoMigrate(&ProductMaster{}, &InventoryItem{}, &StockMovement{}); err != nil {
			t.Fatalf("automigrate %s: %v", label, err)
		}

		item := InventoryItem{
			ProductCode:       "SCRATCH-B5-001",
			QuantityOnHand:    13,
			QuantityAvailable: 13,
			UnitCost:          2.5,
			IsActive:          true,
		}
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("seed inventory item: %v", err)
		}

		// The double: two identical un-referenced Adjustment INs in the same
		// minute (the exact fingerprint of the pre-9.7 create→approve bug),
		// plus one legitimate adjustment in a different minute that must NOT
		// be flagged. Balances mirror what the double-post bug really left
		// behind: the ledger and quantity_on_hand agree (13) — the ledger is
		// internally consistent, it's just 5 units too high vs reality.
		base := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.UTC)
		seedMovements := []StockMovement{
			{InventoryItemID: item.ID, MovementType: "Adjustment", Quantity: 5, Direction: "IN",
				MovementDate: base.Add(10 * time.Second), UnitCost: 2.5, BalanceBefore: 0, BalanceAfter: 5},
			{InventoryItemID: item.ID, MovementType: "Adjustment", Quantity: 5, Direction: "IN",
				MovementDate: base.Add(40 * time.Second), UnitCost: 2.5, BalanceBefore: 5, BalanceAfter: 10},
			{InventoryItemID: item.ID, MovementType: "Adjustment", Quantity: 3, Direction: "IN",
				MovementDate: base.Add(5 * time.Minute), UnitCost: 2.5, BalanceBefore: 10, BalanceAfter: 13},
		}
		for i := range seedMovements {
			if err := db.Create(&seedMovements[i]).Error; err != nil {
				t.Fatalf("seed movement %d: %v", i, err)
			}
			// db.Create stamps CreatedAt=now for every row; the repair keeps
			// the EARLIEST-created row as legitimate, so give the rows
			// strictly increasing created_at matching their movement order.
			if err := db.Model(&StockMovement{}).Where("id = ?", seedMovements[i].ID).
				Update("created_at", base.Add(time.Duration(i)*time.Second)).Error; err != nil {
				t.Fatalf("stamp created_at for movement %d: %v", i, err)
			}
		}

		t.Logf("seeded %s: item %s (qty_on_hand=13) + 1 doubled group (2×IN 5 same minute) + 1 legit adjustment (IN 3)",
			label, item.ID)
		t.Logf("expected downstream: DIAGNOSE=1 group/1 extra · REPAIR posts 1 compensating OUT 5 · VERIFY cancels exactly, final qty 8")

		if err := closeGormDB(db); err != nil {
			t.Logf("warning: close %s: %v", label, err)
		}
	}
}
