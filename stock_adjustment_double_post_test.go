package main

// B2: CreateStockAdjustment and ApproveStockAdjustment both used to call
// a.RecordStockMovement(...), so a create→approve sequence posted TWO
// StockMovements and applied the variance to on-hand quantity twice, even
// though the adjustment was created with Status="Pending" (implying nothing
// had been applied yet). Article III: posting happens at the authorization
// moment. These tests prove CreateStockAdjustment no longer posts a movement,
// and ApproveStockAdjustment posts exactly one, stamped with provenance back
// to the adjustment.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreateStockAdjustment_DoesNotPostMovement(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&InventoryItem{}, &StockMovement{}, &StockAdjustment{}))

	item, err := app.CreateInventoryItem(InventoryItem{
		ProductCode: "TEST-B2-001",
		WarehouseID: "wh-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, item.ID)

	adjustment := StockAdjustment{
		InventoryItemID:  item.ID,
		Reason:           "cycle count variance",
		Variance:         5,
		AdjustmentNumber: "ADJ-B2-1",
	}
	require.NoError(t, app.CreateStockAdjustment(adjustment))

	itemID := item.ID
	movements, err := app.GetStockMovements(&itemID, "All", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	require.Len(t, movements, 0, "CreateStockAdjustment must not post a StockMovement")

	var stored StockAdjustment
	require.NoError(t, app.db.First(&stored, "inventory_item_id = ? AND adjustment_number = ?", item.ID, "ADJ-B2-1").Error)
	require.Equal(t, "Pending", stored.Status)
}

func TestApproveStockAdjustment_PostsExactlyOneMovement(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&InventoryItem{}, &StockMovement{}, &StockAdjustment{}))

	item, err := app.CreateInventoryItem(InventoryItem{
		ProductCode: "TEST-B2-002",
		WarehouseID: "wh-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, item.ID)
	require.Equal(t, 0.0, item.QuantityOnHand)

	adjustment := StockAdjustment{
		InventoryItemID:  item.ID,
		Reason:           "cycle count variance",
		Variance:         7,
		AdjustmentNumber: "ADJ-B2-2",
	}
	require.NoError(t, app.CreateStockAdjustment(adjustment))

	var stored StockAdjustment
	require.NoError(t, app.db.First(&stored, "inventory_item_id = ? AND adjustment_number = ?", item.ID, "ADJ-B2-2").Error)

	require.NoError(t, app.ApproveStockAdjustment(stored.ID))

	itemID := item.ID
	movements, err := app.GetStockMovements(&itemID, "All", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	require.Len(t, movements, 1, "ApproveStockAdjustment must post exactly one StockMovement")
	require.Equal(t, "StockAdjustment", movements[0].ReferenceType)
	require.Equal(t, stored.ID, movements[0].ReferenceID)

	var approved StockAdjustment
	require.NoError(t, app.db.First(&approved, "id = ?", stored.ID).Error)
	require.Equal(t, "Approved", approved.Status)

	reloaded, err := app.GetInventoryItem(item.ID)
	require.NoError(t, err)
	require.Equal(t, 7.0, reloaded.QuantityOnHand, "variance must be applied exactly once")
}
