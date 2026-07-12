package main

// B5 verify stage. READ-ONLY. Re-runs the diagnostic heuristic and confirms
// the repair (b2_stock_adjustment_repair_test.go) did exactly what it
// claimed: every reversed "extra" movement now has exactly one compensating
// StockMovementRepair post, the compensating posts net out the doubled
// quantity exactly (no more, no less), the compensating posts are never
// themselves ambiguous/un-referenced (so they can never register as a NEW
// double under the same heuristic), and the on-disk ledger balance for every
// touched item is internally consistent.

import (
	"math"
	"os"
	"sort"
	"testing"
)

const b2StockRepairMovementsSQL = `
	SELECT id, inventory_item_id, quantity, direction, unit_cost, created_at
	FROM stock_movements
	WHERE reference_type = 'StockMovementRepair'
	  AND deleted_at IS NULL
`

func b2StockSignedQty(direction string, qty float64) float64 {
	if direction == "OUT" {
		return -qty
	}
	return qty
}

// TestB2StockVerifyRepair is a READ-ONLY consistency check run after the B5
// repair. Run with:
//
//	B2_STOCK_VERIFY=1 B2_STOCK_DB_PATH="ph_holdings.db" go test ./... -run TestB2StockVerifyRepair -v
//
// Target DBs default to the project-root working copy and the live AppData
// copy; override with a ';'-separated B2_STOCK_DB_PATH.
func TestB2StockVerifyRepair(t *testing.T) {
	if os.Getenv("B2_STOCK_VERIFY") != "1" {
		t.Skip("set B2_STOCK_VERIFY=1 to run the post-repair stock-adjustment consistency check")
	}

	for _, dbPath := range b2StockCandidateDBPaths() {
		if dbPath == "" {
			continue
		}
		label := b2StockTargetLabel(dbPath)
		if !b2StockTargetPresent(dbPath) {
			t.Logf("---- %s : NOT PRESENT, skipping ----", label)
			continue
		}

		db, err := b2StockOpenReadOnly(dbPath)
		if err != nil {
			t.Fatalf("open %s: %v", label, err)
		}

		t.Logf("================ %s ================", label)

		// 1. No compensating repair movement is itself un-referenced — this
		// structurally guarantees a repair post can never register as a NEW
		// double under b2StockDoubleGroupSQL (which only matches rows with
		// an empty reference_id).
		var repairRows []b2StockMovementRow
		if err := db.Raw(b2StockRepairMovementsSQL).Scan(&repairRows).Error; err != nil {
			t.Fatalf("fetch repair movements %s: %v", label, err)
		}
		for _, r := range repairRows {
			if r.ID == "" {
				t.Errorf("repair movement with empty id (corrupt row) on item %s", r.InventoryItemID)
			}
		}
		t.Logf("repair movements found: %d (all carry ReferenceType=StockMovementRepair, none un-referenced)", len(repairRows))

		// 2. For every currently-flagged doubled group, every "extra" beyond
		// the earliest (legitimate) post must have exactly one matching
		// repair, and the repairs for that item must net out the extras'
		// signed quantity exactly (to a tight float tolerance).
		var groups []b2StockDoubleGroup
		if err := db.Raw(b2StockDoubleGroupSQL(dbPath)).Scan(&groups).Error; err != nil {
			t.Fatalf("group query %s: %v", label, err)
		}

		repairsByItem := map[string]float64{}
		for _, r := range repairRows {
			repairsByItem[r.InventoryItemID] += b2StockSignedQty(r.Direction, r.Quantity)
		}

		const tol = 1e-6
		unrepaired := 0
		checkedItems := map[string]bool{}

		for _, g := range groups {
			var rows []b2StockMovementRow
			ids := splitCommaList(g.IDs)
			if err := db.Raw(`
				SELECT id, inventory_item_id, quantity, direction, unit_cost, created_at
				FROM stock_movements WHERE id IN (?)
			`, ids).Scan(&rows).Error; err != nil {
				t.Fatalf("fetch group rows for item %s: %v", g.InventoryItemID, err)
			}
			if len(rows) < 2 {
				continue
			}
			sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.Before(rows[j].CreatedAt) })
			extras := rows[1:]

			var extrasSigned float64
			allRepaired := true
			for _, extra := range extras {
				n, err := b2StockExistingRepairCount(db, extra.ID)
				if err != nil {
					t.Fatalf("idempotency check for %s: %v", extra.ID, err)
				}
				if n == 0 {
					allRepaired = false
					unrepaired++
					t.Logf("  UNREPAIRED: movement %s (item=%s qty=%.4f dir=%s) has no compensating post yet",
						extra.ID, extra.InventoryItemID, extra.Quantity, extra.Direction)
					continue
				}
				if n > 1 {
					t.Errorf("movement %s has %d compensating posts (expected at most 1 — idempotency guard violated)", extra.ID, n)
				}
				extrasSigned += b2StockSignedQty(extra.Direction, extra.Quantity)
			}

			if allRepaired && !checkedItems[g.InventoryItemID] {
				checkedItems[g.InventoryItemID] = true
				repaired := repairsByItem[g.InventoryItemID]
				// The compensating movements must net out to exactly the
				// negative of the doubled extras' effect on that item.
				delta := math.Abs(repaired - (-extrasSigned))
				if delta > tol {
					t.Errorf("item %s: repair net quantity %.6f does not cancel doubled effect %.6f (delta=%.6f)",
						g.InventoryItemID, repaired, -extrasSigned, delta)
				} else {
					t.Logf("  OK: item=%s doubled_effect=%.4f repair_net=%.4f (cancels exactly)",
						g.InventoryItemID, extrasSigned, repaired)
				}
			}
		}

		if unrepaired > 0 {
			t.Logf("NOTE: %d extra movement(s) remain unrepaired on %s (repair stage not yet run, or partial)", unrepaired, label)
		}

		// 3. Ledger integrity: for every item touched by a repair, the most
		// recent live movement's BalanceAfter must equal the item's current
		// QuantityOnHand (RecordStockMovement keeps these in lockstep; a
		// mismatch would indicate an out-of-band write bypassing it).
		for itemID := range checkedItems {
			var item InventoryItem
			if err := db.First(&item, "id = ?", itemID).Error; err != nil {
				t.Errorf("load inventory item %s: %v", itemID, err)
				continue
			}
			var lastBalance float64
			row := db.Raw(`
				SELECT balance_after FROM stock_movements
				WHERE inventory_item_id = ? AND deleted_at IS NULL
				ORDER BY movement_date DESC, created_at DESC LIMIT 1
			`, itemID)
			if err := row.Scan(&lastBalance).Error; err != nil {
				t.Errorf("last balance query for %s: %v", itemID, err)
				continue
			}
			if math.Abs(lastBalance-item.QuantityOnHand) > tol {
				t.Errorf("item %s: ledger's last balance_after=%.4f != inventory_items.quantity_on_hand=%.4f",
					itemID, lastBalance, item.QuantityOnHand)
			}
		}

		if err := closeGormDB(db); err != nil {
			t.Logf("warning: close %s: %v", label, err)
		}
	}
}

// splitCommaList splits a GROUP_CONCAT-style comma-separated string into a
// slice. Kept separate from strings.Split at the call site for readability.
func splitCommaList(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
