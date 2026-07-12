package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Wave 9.7 tight-ship-2: ReceiveAgainstPO/ReceiveAgainstPOWithSerials only
// ever created a PENDING GRN — no stock posted, no PO status moved — which
// reproduced the cosmetic PO-status-flip bug once the GRNScreen receiving UI
// was retired (the only caller left, UpdatePOStatus(poId,'Received'), just
// flips the status column with nothing behind it). ReceiveAndCompletePO[
// WithSerials] chain create -> CompleteGRN atomically so a single call both
// posts stock (via reconcileInventoryReceipt, inside CompleteGRN's FOR
// UPDATE-guarded transaction) and advances the PO's real status (via
// updatePOStatus). These tests exercise that chain end to end.

// receiveAndCompleteFixture builds a supplier + serial-trackable product +
// Sent PO with a single line of qtyOrdered units, ready for
// ReceiveAndCompletePO[WithSerials].
func receiveAndCompleteFixture(t *testing.T, a *App, poNumber string, qtyOrdered float64) (PurchaseOrder, PurchaseOrderItem, ProductMaster) {
	t.Helper()

	supplier := SupplierMaster{SupplierCode: "SUP-" + poNumber, SupplierName: "Meridian Flow Instruments"}
	require.NoError(t, a.db.Create(&supplier).Error)
	product := ProductMaster{ProductCode: "PC-" + poNumber, ProductName: "Flow transmitter", StandardCostBHD: 5, RequiresSerialTracking: true}
	require.NoError(t, a.db.Create(&product).Error)

	po := PurchaseOrder{
		PONumber: poNumber, SupplierID: supplier.ID, SupplierName: supplier.SupplierName,
		Currency: "BHD", ExchangeRate: 1, Status: "Sent", PODate: time.Now(),
	}
	require.NoError(t, a.db.Create(&po).Error)

	poItem := PurchaseOrderItem{
		PurchaseOrderID: po.ID, ProductID: product.ID, ProductCode: product.ProductCode,
		Description: "Flow transmitter", Quantity: qtyOrdered, UnitPriceForeign: 20, UnitPriceBHD: 20,
		TotalBHD: qtyOrdered * 20,
	}
	require.NoError(t, a.db.Create(&poItem).Error)

	return po, poItem, product
}

func TestReceiveAndCompletePO_PostsStockAndUpdatesPO(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &Warehouse{}, &InventoryItem{}, &StockMovement{},
		&PurchaseOrder{}, &PurchaseOrderItem{}, &GoodsReceivedNote{}, &GRNItem{}))

	po, poItem, _ := receiveAndCompleteFixture(t, a, "PO-26-9201", 10)

	// Partial receive: 6 of 10.
	resp, err := a.ReceiveAndCompletePO(po.ID, []GRNItem{
		{POItemID: poItem.ID, QuantityReceived: 6},
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.ID)
	require.NotNil(t, resp.CompletedAt, "wrapper must complete the GRN, not just create it")
	require.True(t, resp.IsCompleted)

	var movementCount int64
	require.NoError(t, a.db.Model(&StockMovement{}).Where("reference_id = ?", resp.ID).Count(&movementCount).Error)
	require.EqualValues(t, 1, movementCount, "accepted receipt must post exactly one StockMovement via reconcileInventoryReceipt")

	var afterPartial PurchaseOrderItem
	require.NoError(t, a.db.First(&afterPartial, "id = ?", poItem.ID).Error)
	require.InDelta(t, 6.0, afterPartial.QuantityReceived, 0.0001)

	var poAfterPartial PurchaseOrder
	require.NoError(t, a.db.First(&poAfterPartial, "id = ?", po.ID).Error)
	require.Equal(t, "Partially Received", poAfterPartial.Status)

	// Receive the remainder: 4 more.
	resp2, err := a.ReceiveAndCompletePO(po.ID, []GRNItem{
		{POItemID: poItem.ID, QuantityReceived: 4},
	})
	require.NoError(t, err)
	require.NotNil(t, resp2.CompletedAt)

	var afterFull PurchaseOrderItem
	require.NoError(t, a.db.First(&afterFull, "id = ?", poItem.ID).Error)
	require.InDelta(t, 10.0, afterFull.QuantityReceived, 0.0001)

	var poAfterFull PurchaseOrder
	require.NoError(t, a.db.First(&poAfterFull, "id = ?", po.ID).Error)
	require.Equal(t, "Received", poAfterFull.Status)

	// Two separate GRNs (one per call), two separate movements.
	require.NoError(t, a.db.Model(&StockMovement{}).Where("reference_type = ?", "goods_received_note").Count(&movementCount).Error)
	require.EqualValues(t, 2, movementCount)
}

// TestReceiveAndCompletePO_Idempotent asserts that completing the SAME GRN
// twice (bypassing the wrapper the second time, exactly like a retried
// CompleteGRN call would) never double-posts stock — CompleteGRN's own
// FOR UPDATE + CompletedAt guard, which ReceiveAndCompletePO must not
// weaken or bypass.
func TestReceiveAndCompletePO_Idempotent(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &Warehouse{}, &InventoryItem{}, &StockMovement{},
		&PurchaseOrder{}, &PurchaseOrderItem{}, &GoodsReceivedNote{}, &GRNItem{}))

	po, poItem, _ := receiveAndCompleteFixture(t, a, "PO-26-9202", 5)

	resp, err := a.ReceiveAndCompletePO(po.ID, []GRNItem{
		{POItemID: poItem.ID, QuantityReceived: 5},
	})
	require.NoError(t, err)

	var movementCount int64
	require.NoError(t, a.db.Model(&StockMovement{}).Where("reference_id = ?", resp.ID).Count(&movementCount).Error)
	require.EqualValues(t, 1, movementCount)

	// A second CompleteGRN on the SAME GRN (e.g. a retried call) must be a
	// silent no-op per the existing idempotency guard.
	require.NoError(t, a.CompleteGRN(resp.ID))

	require.NoError(t, a.db.Model(&StockMovement{}).Where("reference_id = ?", resp.ID).Count(&movementCount).Error)
	require.EqualValues(t, 1, movementCount, "second completion must not double-post stock")

	var afterSecond PurchaseOrderItem
	require.NoError(t, a.db.First(&afterSecond, "id = ?", poItem.ID).Error)
	require.InDelta(t, 5.0, afterSecond.QuantityReceived, 0.0001, "second completion must not double-count PO received qty")
}

func TestReceiveAndCompletePOWithSerials_MintsSerials(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &Warehouse{}, &InventoryItem{}, &StockMovement{},
		&PurchaseOrder{}, &PurchaseOrderItem{}, &GoodsReceivedNote{}, &GRNItem{}, &SerialNumber{}))

	po, poItem, product := receiveAndCompleteFixture(t, a, "PO-26-9203", 3)

	resp, err := a.ReceiveAndCompletePOWithSerials(po.ID, []GRNItemWithSerials{
		{
			GRNItem:       GRNItem{POItemID: poItem.ID, QuantityReceived: 3},
			SerialNumbers: []string{"SN-9203-1", "SN-9203-2", "SN-9203-3"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.CompletedAt, "wrapper must complete the GRN so the mint is paired with a real receipt")

	var serials []SerialNumber
	require.NoError(t, a.db.Where("product_id = ?", product.ID).Find(&serials).Error)
	require.Len(t, serials, 3)
	for _, s := range serials {
		require.Equal(t, "Available", s.Status)
		require.Equal(t, resp.GRNNumber, s.GRNNumber)
		require.Equal(t, po.PONumber, s.PONumber)
	}

	var movementCount int64
	require.NoError(t, a.db.Model(&StockMovement{}).Where("reference_id = ?", resp.ID).Count(&movementCount).Error)
	require.EqualValues(t, 1, movementCount)
}
