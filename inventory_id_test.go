package main

// Wave 8 P1: InventoryItem/StockMovement/StockAdjustment use string UUID PKs
// (Base), but GetInventoryItem(s)/UpdateInventoryItem/GetStockMovements/
// ApproveStockAdjustment took uint IDs, so GORM lookups never matched a row
// (silent "not found"). This proves the string-ID round-trip now works.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInventoryStringIDRoundTrip(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&InventoryItem{}, &StockMovement{}, &StockAdjustment{}))

	// Create an inventory item — its ID is a string UUID assigned by Base.
	created, err := app.CreateInventoryItem(InventoryItem{
		ProductCode: "TEST-001",
		WarehouseID: "wh-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	// GetInventoryItem by string ID must find it (pre-fix: a uint arg matched nothing).
	got, err := app.GetInventoryItem(created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)

	// UpdateInventoryItem by string ID persists an allowed column.
	require.NoError(t, app.UpdateInventoryItem(created.ID, map[string]any{"reorder_point": 5.0}))
	reloaded, err := app.GetInventoryItem(created.ID)
	require.NoError(t, err)
	require.Equal(t, 5.0, reloaded.ReorderPoint)

	// GetInventoryItems filtered by warehouse string ID.
	wh := "wh-1"
	items, err := app.GetInventoryItems(&wh, "All", false)
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Record a movement, then fetch movements filtered by the item's string ID.
	_, err = app.RecordStockMovement(StockMovement{
		InventoryItemID: created.ID,
		Quantity:        10,
		Direction:       "IN",
	})
	require.NoError(t, err)
	itemID := created.ID
	movements, err := app.GetStockMovements(&itemID, "All", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	require.Len(t, movements, 1)

	// Approve a Pending adjustment by string ID (pre-fix: a uint arg matched nothing,
	// so this always returned ADJUSTMENT_NOT_FOUND).
	adj := StockAdjustment{
		Base:             Base{ID: "adj-string-1"},
		InventoryItemID:  created.ID,
		Reason:           "cycle count",
		Status:           "Pending",
		Variance:         2,
		AdjustmentNumber: "ADJ-TEST-1",
		AdjustmentDate:   time.Now(),
	}
	require.NoError(t, app.db.Create(&adj).Error)
	require.NoError(t, app.ApproveStockAdjustment(adj.ID))

	var approved StockAdjustment
	require.NoError(t, app.db.First(&approved, "id = ?", adj.ID).Error)
	require.Equal(t, "Approved", approved.Status)
}
