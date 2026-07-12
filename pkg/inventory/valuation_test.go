package inventory

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/ncruces/go-sqlite3/embed"
	"ph_holdings_app/pkg/crm"
)

func valuationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "valuation.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&crm.ProductMaster{}, &crm.InventoryItem{}, &crm.StockMovement{}, &crm.PurchaseOrderItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func inDelta(t *testing.T, want, got float64, what string) {
	t.Helper()
	if math.Abs(want-got) > 0.0001 {
		t.Fatalf("%s: want %v, got %v", what, want, got)
	}
}

// GOLDEN (PH product_valuation_policy_test.go semantics): weighted average on
// an IN movement — 5 units on hand @10, 5 more received @18 → 10 units @14,
// LastPurchaseCost 18. Averages are raw float64, no intermediate rounding.
func TestApplyMovementValuation_WeightedAverageIn(t *testing.T) {
	item := crm.InventoryItem{UnitCost: 10, LastPurchaseCost: 10, QuantityOnHand: 10}
	movement := crm.StockMovement{Direction: "IN", Quantity: 5, BalanceBefore: 5, UnitCost: 18}

	ApplyMovementValuation(&item, &movement, ReferenceCost{UnitCostBHD: 10, Source: "inventory_unit_cost"})

	inDelta(t, 14, item.UnitCost, "weighted average unit cost")
	inDelta(t, 18, item.LastPurchaseCost, "last purchase cost")
	inDelta(t, 140, item.TotalValue, "item total value (10 × 14)")
	inDelta(t, 90, movement.TotalValue, "movement value (5 × 18)")
}

// A zero-cost item must average against its FALLBACK cost, not zero — the
// exact defect of the pre-convergence inline version.
func TestApplyMovementValuation_FallbackAwareExistingCost(t *testing.T) {
	item := crm.InventoryItem{UnitCost: 0, LastPurchaseCost: 12, QuantityOnHand: 10}
	movement := crm.StockMovement{Direction: "IN", Quantity: 6, BalanceBefore: 4, UnitCost: 0}

	// Chain resolves 12 via last purchase cost; incoming cost also backfills.
	ApplyMovementValuation(&item, &movement, ReferenceCost{UnitCostBHD: 12, Source: "inventory_last_purchase_cost"})

	inDelta(t, 12, item.UnitCost, "backfilled average")
	inDelta(t, 12, movement.UnitCost, "movement cost backfilled from chain")
	inDelta(t, 72, movement.TotalValue, "movement value (6 × 12)")
	inDelta(t, 120, item.TotalValue, "item total value (10 × 12)")
}

func TestApplyMovementValuation_FirstReceiptAndOutBackfill(t *testing.T) {
	// First receipt into an empty item takes the incoming cost outright.
	item := crm.InventoryItem{QuantityOnHand: 7}
	movement := crm.StockMovement{Direction: "IN", Quantity: 7, BalanceBefore: 0, UnitCost: 9}
	ApplyMovementValuation(&item, &movement, ReferenceCost{})
	inDelta(t, 9, item.UnitCost, "first-receipt unit cost")
	inDelta(t, 63, item.TotalValue, "first-receipt total value")

	// OUT with a zero stored cost backfills from the chain (e.g. standard cost).
	out := crm.InventoryItem{QuantityOnHand: 3}
	outMove := crm.StockMovement{Direction: "OUT", Quantity: 2, BalanceBefore: 5}
	ApplyMovementValuation(&out, &outMove, ReferenceCost{UnitCostBHD: 4.5, Source: "product_standard_cost"})
	inDelta(t, 4.5, out.UnitCost, "OUT backfilled unit cost")
	inDelta(t, 9, outMove.TotalValue, "OUT movement value (2 × 4.5)")
	inDelta(t, 13.5, out.TotalValue, "OUT item total value (3 × 4.5)")
}

func TestResolveInventoryItemUnitCost_ChainOrder(t *testing.T) {
	db := valuationTestDB(t)
	product := crm.ProductMaster{ProductCode: "PT-100", ProductName: "Pressure transmitter", StandardCostBHD: 7.5}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}

	got := ResolveInventoryItemUnitCost(db, crm.InventoryItem{UnitCost: 3, LastPurchaseCost: 5, ProductID: product.ID})
	if got.Source != "inventory_unit_cost" {
		t.Fatalf("stored unit cost must win, got %+v", got)
	}
	got = ResolveInventoryItemUnitCost(db, crm.InventoryItem{LastPurchaseCost: 5, ProductID: product.ID})
	if got.Source != "inventory_last_purchase_cost" || got.UnitCostBHD != 5 {
		t.Fatalf("last purchase cost is second, got %+v", got)
	}
	got = ResolveInventoryItemUnitCost(db, crm.InventoryItem{ProductID: product.ID})
	if got.Source != "product_standard_cost" || got.UnitCostBHD != 7.5 {
		t.Fatalf("standard cost is third, got %+v", got)
	}
	got = ResolveInventoryItemUnitCost(db, crm.InventoryItem{})
	if got.UnitCostBHD != 0 || got.Source != "" {
		t.Fatalf("empty chain resolves to zero, got %+v", got)
	}
}

func TestResolvePurchaseOrderItemReferenceCost(t *testing.T) {
	db := valuationTestDB(t)
	product := crm.ProductMaster{ProductCode: "GA-1900", ProductName: "Gas analyzer", StandardCostBHD: 3200}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}

	got := ResolvePurchaseOrderItemReferenceCost(db, crm.PurchaseOrderItem{UnitPriceBHD: 3150.250, ProductID: product.ID})
	if got.Source != "po_unit_price_bhd" {
		t.Fatalf("PO unit price wins, got %+v", got)
	}
	inDelta(t, 3150.250, got.UnitCostBHD, "PO reference cost")

	got = ResolvePurchaseOrderItemReferenceCost(db, crm.PurchaseOrderItem{ProductID: product.ID})
	if got.Source != "product_standard_cost" {
		t.Fatalf("falls back to standard cost, got %+v", got)
	}
	inDelta(t, 3200, got.UnitCostBHD, "standard-cost fallback")
}

func TestSupplierInvoiceItemUnitPriceBHD(t *testing.T) {
	inDelta(t, 100, SupplierInvoiceItemUnitPriceBHD("BHD", 1, 100), "BHD passthrough")
	inDelta(t, 100, SupplierInvoiceItemUnitPriceBHD("", 0, 100), "blank currency passthrough")
	inDelta(t, 42, SupplierInvoiceItemUnitPriceBHD("EUR", 0.42, 100), "EUR conversion at 0.42")
	inDelta(t, 100, SupplierInvoiceItemUnitPriceBHD("EUR", 0, 100), "non-positive rate passthrough")
	inDelta(t, 0, SupplierInvoiceItemUnitPriceBHD("EUR", 0.42, 0), "zero price stays zero")
}

func TestResolveInventoryItemTotalValue_Fallback(t *testing.T) {
	db := valuationTestDB(t)

	inDelta(t, 55.5, ResolveInventoryItemTotalValue(db, crm.InventoryItem{TotalValue: 55.5}), "stored total wins")
	inDelta(t, 24, ResolveInventoryItemTotalValue(db, crm.InventoryItem{QuantityOnHand: 8, LastPurchaseCost: 3}), "quantity × chain cost")
	inDelta(t, 0, ResolveInventoryItemTotalValue(db, crm.InventoryItem{QuantityOnHand: 8}), "no cost source → zero")
}
