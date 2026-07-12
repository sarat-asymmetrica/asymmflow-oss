package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Wave 9 B3: CompleteGRN now stamps GoodsReceivedNote.CompletedAt inside the
// same row-locked transaction as the PO-quantity update, and the idempotency
// guard is `alreadyApplied || lockedGRN.CompletedAt != nil`. This closes a
// known gap in the older grnHasPostedMovement-only guard: an ALL-REJECTED GRN
// (QuantityAccepted <= 0 on every line) never posts a StockMovement even on a
// legitimate first completion, so that signal alone couldn't tell "never
// completed" apart from "completed with nothing accepted". These tests cover
// that edge case plus the ordinary accepted-quantity path, and the
// GRNResponse.IsCompleted derivation off the new flag.

// grnCompletionFixture creates one supplier + product + PO/POItem/GRN/GRNItem
// ready for CompleteGRN, with the given received/rejected quantities.
func grnCompletionFixture(t *testing.T, a *App, poNumber, grnNumber string, qtyReceived, qtyRejected float64) (GoodsReceivedNote, PurchaseOrderItem) {
	t.Helper()

	supplier := SupplierMaster{SupplierCode: "SUP-" + grnNumber, SupplierName: "Vantage Flow Instruments"}
	require.NoError(t, a.db.Create(&supplier).Error)
	product := ProductMaster{ProductCode: "PC-" + grnNumber, ProductName: "Flow transmitter", StandardCostBHD: 5}
	require.NoError(t, a.db.Create(&product).Error)

	po := PurchaseOrder{
		PONumber: poNumber, SupplierID: supplier.ID, SupplierName: supplier.SupplierName,
		Currency: "BHD", ExchangeRate: 1, Status: "Sent", PODate: time.Now(),
	}
	require.NoError(t, a.db.Create(&po).Error)

	poItem := PurchaseOrderItem{
		PurchaseOrderID: po.ID, ProductID: product.ID, ProductCode: product.ProductCode,
		Description: "Flow transmitter", Quantity: qtyReceived, UnitPriceForeign: 20, UnitPriceBHD: 20,
		TotalBHD: qtyReceived * 20,
	}
	require.NoError(t, a.db.Create(&poItem).Error)

	grn := GoodsReceivedNote{GRNNumber: grnNumber, PurchaseOrderID: po.ID, QCStatus: "Pending", ReceivedDate: time.Now()}
	require.NoError(t, a.db.Create(&grn).Error)

	grnItem := GRNItem{
		GRNID: grn.ID, POItemID: poItem.ID, ProductID: product.ID,
		QuantityOrdered: qtyReceived, QuantityReceived: qtyReceived, QuantityRejected: qtyRejected,
	}
	require.NoError(t, a.db.Create(&grnItem).Error)

	return grn, poItem
}

func TestCompleteGRN_AllRejected_SetsCompletedAtAndIsIdempotentWithoutDoubleCounting(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &Warehouse{}, &InventoryItem{}, &StockMovement{},
		&PurchaseOrder{}, &PurchaseOrderItem{}, &GoodsReceivedNote{}, &GRNItem{}))

	grn, poItem := grnCompletionFixture(t, a, "PO-26-9101", "GRN-26-9101", 4, 4) // fully rejected

	require.NoError(t, a.CompleteGRN(grn.ID))

	var completed GoodsReceivedNote
	require.NoError(t, a.db.First(&completed, "id = ?", grn.ID).Error)
	require.NotNil(t, completed.CompletedAt, "all-rejected GRN must still record a completion flag")

	// No stock movement should have posted — nothing was accepted.
	var movementCount int64
	require.NoError(t, a.db.Model(&StockMovement{}).Where("reference_id = ?", grn.ID).Count(&movementCount).Error)
	require.EqualValues(t, 0, movementCount, "all-rejected GRN posts no StockMovement")

	var afterFirst PurchaseOrderItem
	require.NoError(t, a.db.First(&afterFirst, "id = ?", poItem.ID).Error)
	require.InDelta(t, 4.0, afterFirst.QuantityReceived, 0.0001)

	// Second CompleteGRN call must be a silent no-op: no error, no double-apply.
	require.NoError(t, a.CompleteGRN(grn.ID))

	var afterSecond PurchaseOrderItem
	require.NoError(t, a.db.First(&afterSecond, "id = ?", poItem.ID).Error)
	require.InDelta(t, 4.0, afterSecond.QuantityReceived, 0.0001, "second completion must not double-count PO received qty")

	var stillCompleted GoodsReceivedNote
	require.NoError(t, a.db.First(&stillCompleted, "id = ?", grn.ID).Error)
	require.NotNil(t, stillCompleted.CompletedAt)
}

func TestCompleteGRN_WithAcceptedQty_SetsCompletedAtAndStaysIdempotent(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &Warehouse{}, &InventoryItem{}, &StockMovement{},
		&PurchaseOrder{}, &PurchaseOrderItem{}, &GoodsReceivedNote{}, &GRNItem{}))

	grn, poItem := grnCompletionFixture(t, a, "PO-26-9102", "GRN-26-9102", 6, 0) // fully accepted

	require.NoError(t, a.CompleteGRN(grn.ID))

	var completed GoodsReceivedNote
	require.NoError(t, a.db.First(&completed, "id = ?", grn.ID).Error)
	require.NotNil(t, completed.CompletedAt)

	var movementCount int64
	require.NoError(t, a.db.Model(&StockMovement{}).Where("reference_id = ?", grn.ID).Count(&movementCount).Error)
	require.EqualValues(t, 1, movementCount, "accepted GRN posts exactly one StockMovement")

	// Second call: still a no-op (belt-and-suspenders — the pre-existing
	// grnHasPostedMovement guard AND the new CompletedAt guard both fire).
	require.NoError(t, a.CompleteGRN(grn.ID))

	require.NoError(t, a.db.Model(&StockMovement{}).Where("reference_id = ?", grn.ID).Count(&movementCount).Error)
	require.EqualValues(t, 1, movementCount, "second completion must not post a duplicate StockMovement")

	var afterSecond PurchaseOrderItem
	require.NoError(t, a.db.First(&afterSecond, "id = ?", poItem.ID).Error)
	require.InDelta(t, 6.0, afterSecond.QuantityReceived, 0.0001, "second completion must not double-count PO received qty")
}

func TestGRNResponse_IsCompletedDerivesFromCompletedAtFlag(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &Warehouse{}, &InventoryItem{}, &StockMovement{},
		&PurchaseOrder{}, &PurchaseOrderItem{}, &GoodsReceivedNote{}, &GRNItem{}))

	grn, _ := grnCompletionFixture(t, a, "PO-26-9103", "GRN-26-9103", 3, 0)

	// Before completion: IsCompleted must be false on both single-fetch and list paths.
	before, err := a.GetGRN(grn.ID)
	require.NoError(t, err)
	require.False(t, before.IsCompleted)

	listBefore, err := a.ListGRNs(100, 0, "")
	require.NoError(t, err)
	beforeFromList := findGRNResponse(listBefore, grn.ID)
	require.NotNil(t, beforeFromList)
	require.False(t, beforeFromList.IsCompleted)

	require.NoError(t, a.CompleteGRN(grn.ID))

	after, err := a.GetGRN(grn.ID)
	require.NoError(t, err)
	require.True(t, after.IsCompleted, "GetGRN must report is_completed=true once CompletedAt is set")

	listAfter, err := a.ListGRNs(100, 0, "")
	require.NoError(t, err)
	found := findGRNResponse(listAfter, grn.ID)
	require.NotNil(t, found)
	require.True(t, found.IsCompleted, "ListGRNs must report is_completed=true once CompletedAt is set")
}

func findGRNResponse(list []GRNResponse, id string) *GRNResponse {
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}
