package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// PH convergence Band-2 rows 17-18 (PH 5937958): completing a GRN turns
// accepted goods into valued stock — inventory upsert, a GRN Receipt
// movement, and weighted-average cost from the PO reference cost.
func TestCompleteGRN_CreatesValuedStockAndWeightedAverages(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &Warehouse{}, &InventoryItem{}, &StockMovement{},
		&PurchaseOrder{}, &PurchaseOrderItem{}, &GoodsReceivedNote{}, &GRNItem{}))

	supplier := SupplierMaster{SupplierCode: "SRVX", SupplierName: "Oxan Analytics"}
	require.NoError(t, a.db.Create(&supplier).Error)
	product := ProductMaster{ProductCode: "PT-100", ProductName: "Pressure transmitter", StandardCostBHD: 10}
	require.NoError(t, a.db.Create(&product).Error)

	makeReceipt := func(poNumber, grnNumber string, unitPriceBHD, qty float64) {
		po := PurchaseOrder{PONumber: poNumber, SupplierID: supplier.ID, SupplierName: supplier.SupplierName, Currency: "BHD", ExchangeRate: 1, Status: "Sent", PODate: time.Now()}
		require.NoError(t, a.db.Create(&po).Error)
		poItem := PurchaseOrderItem{PurchaseOrderID: po.ID, ProductID: product.ID, ProductCode: product.ProductCode, Description: "Pressure transmitter", Quantity: qty, UnitPriceForeign: unitPriceBHD, UnitPriceBHD: unitPriceBHD, TotalBHD: qty * unitPriceBHD}
		require.NoError(t, a.db.Create(&poItem).Error)
		grn := GoodsReceivedNote{GRNNumber: grnNumber, PurchaseOrderID: po.ID, QCStatus: "Pending", ReceivedDate: time.Now()}
		require.NoError(t, a.db.Create(&grn).Error)
		require.NoError(t, a.db.Create(&GRNItem{GRNID: grn.ID, POItemID: poItem.ID, ProductID: product.ID, QuantityOrdered: qty, QuantityReceived: qty}).Error)
		require.NoError(t, a.CompleteGRN(grn.ID))
	}

	// GOLDEN receipt 1: 5 units @ 18 BHD (PO unit price beats standard cost).
	makeReceipt("PO-26-7001", "GRN-26-7001", 18, 5)

	var item InventoryItem
	require.NoError(t, a.db.First(&item, "product_id = ?", product.ID).Error)
	require.InDelta(t, 5.0, item.QuantityOnHand, 0.0001)
	require.InDelta(t, 18.0, item.UnitCost, 0.0001)
	require.InDelta(t, 18.0, item.LastPurchaseCost, 0.0001)
	require.InDelta(t, 90.0, item.TotalValue, 0.0001)
	require.Equal(t, "InStock", item.StockStatus)

	var movement StockMovement
	require.NoError(t, a.db.First(&movement, "inventory_item_id = ?", item.ID).Error)
	require.Equal(t, "GRN Receipt", movement.MovementType)
	require.Equal(t, "goods_received_note", movement.ReferenceType)
	require.Equal(t, "GRN-26-7001", movement.ReferenceNumber)
	require.InDelta(t, 0.0, movement.BalanceBefore, 0.0001)
	require.InDelta(t, 5.0, movement.BalanceAfter, 0.0001)
	require.InDelta(t, 18.0, movement.UnitCost, 0.0001)
	require.InDelta(t, 90.0, movement.TotalValue, 0.0001)

	// GOLDEN receipt 2: 5 more @ 24 → weighted average (18·5 + 24·5)/10 = 21.
	makeReceipt("PO-26-7002", "GRN-26-7002", 24, 5)

	item = InventoryItem{}
	require.NoError(t, a.db.First(&item, "product_id = ?", product.ID).Error)
	require.InDelta(t, 10.0, item.QuantityOnHand, 0.0001)
	require.InDelta(t, 21.0, item.UnitCost, 0.0001)
	require.InDelta(t, 24.0, item.LastPurchaseCost, 0.0001)
	require.InDelta(t, 210.0, item.TotalValue, 0.0001)

	var count int64
	require.NoError(t, a.db.Model(&InventoryItem{}).Where("product_id = ?", product.ID).Count(&count).Error)
	require.EqualValues(t, 1, count, "receipts into the same warehouse reuse one inventory row")
}

// Band-2: a GRN discrepancy's cost impact comes from the PO/product reference
// cost, not the old rejectedQty×100 placeholder.
func TestRaiseGRNDiscrepancy_CostImpactFromReferenceCost(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{},
		&PurchaseOrder{}, &PurchaseOrderItem{}, &GoodsReceivedNote{}, &GRNItem{}, &SupplierIssue{}))

	supplier := SupplierMaster{SupplierCode: "EH", SupplierName: "Rhine Instruments"}
	require.NoError(t, a.db.Create(&supplier).Error)
	po := PurchaseOrder{PONumber: "PO-26-7003", SupplierID: supplier.ID, SupplierName: supplier.SupplierName, Currency: "BHD", ExchangeRate: 1, Status: "Sent", PODate: time.Now()}
	require.NoError(t, a.db.Create(&po).Error)
	poItem := PurchaseOrderItem{PurchaseOrderID: po.ID, ProductCode: "FMR51-1", Description: "Radar level sensor", Quantity: 4, UnitPriceForeign: 850, UnitPriceBHD: 850, TotalBHD: 3400}
	require.NoError(t, a.db.Create(&poItem).Error)
	grn := GoodsReceivedNote{GRNNumber: "GRN-26-7003", PurchaseOrderID: po.ID, QCStatus: "Pending", ReceivedDate: time.Now()}
	require.NoError(t, a.db.Create(&grn).Error)
	grnItem := GRNItem{GRNID: grn.ID, POItemID: poItem.ID, QuantityOrdered: 4, QuantityReceived: 4}
	require.NoError(t, a.db.Create(&grnItem).Error)

	require.NoError(t, a.RaiseGRNDiscrepancy(grn.ID, grnItem.ID, "Two units damaged in transit", "damage", 2))

	var issue SupplierIssue
	require.NoError(t, a.db.First(&issue, "supplier_id = ?", supplier.ID).Error)
	require.InDelta(t, 1700.0, issue.CostBHD, 0.0001, "2 rejected × 850 BHD PO price")
}

// Wave 9.8 B1: a serialized-product line submitted with zero serials must be
// rejected — both through the plain ReceiveAndCompletePO path (which has no
// way to carry serials at all) and through ReceiveAgainstPOWithSerials itself
// (which previously only validated serial count when serials were present,
// letting an empty list sail through as an unserialized receive).
func TestReceive_SerializedProductWithoutSerials_Rejected(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &Warehouse{}, &InventoryItem{}, &StockMovement{},
		&PurchaseOrder{}, &PurchaseOrderItem{}, &GoodsReceivedNote{}, &GRNItem{}, &SerialNumber{}))

	supplier := SupplierMaster{SupplierCode: "SRLZ", SupplierName: "Serialized Supplies Co"}
	require.NoError(t, a.db.Create(&supplier).Error)
	product := ProductMaster{ProductCode: "SN-100", ProductName: "Serialized widget", StandardCostBHD: 10, RequiresSerialTracking: true}
	require.NoError(t, a.db.Create(&product).Error)

	po := PurchaseOrder{PONumber: "PO-26-7100", SupplierID: supplier.ID, SupplierName: supplier.SupplierName, Currency: "BHD", ExchangeRate: 1, Status: "Sent", PODate: time.Now()}
	require.NoError(t, a.db.Create(&po).Error)
	poItem := PurchaseOrderItem{PurchaseOrderID: po.ID, ProductID: product.ID, ProductCode: product.ProductCode, Description: "Serialized widget", Quantity: 3, UnitPriceForeign: 10, UnitPriceBHD: 10, TotalBHD: 30}
	require.NoError(t, a.db.Create(&poItem).Error)

	// Plain path: GRNItem has no SerialNumbers field at all — must be refused
	// outright rather than silently posted unserialized.
	_, err := a.ReceiveAndCompletePO(po.ID, []GRNItem{
		{POItemID: poItem.ID, ProductID: product.ID, QuantityOrdered: 3, QuantityReceived: 3},
	})
	require.Error(t, err, "receiving a serialized product with no serial capture must fail")

	// Serial-aware path with an empty serials list: previously bypassed the
	// len(serials)==qty check entirely because that check only fired when
	// serials were present at all.
	_, err = a.ReceiveAgainstPOWithSerials(po.ID, []GRNItemWithSerials{
		{
			GRNItem:       GRNItem{POItemID: poItem.ID, ProductID: product.ID, QuantityOrdered: 3, QuantityReceived: 3},
			SerialNumbers: []string{},
		},
	})
	require.Error(t, err, "empty serials for a serialized-product line must be rejected, not silently accepted")

	// Sanity: no GRN was ever created against this PO out of either attempt.
	var grnCount int64
	require.NoError(t, a.db.Model(&GoodsReceivedNote{}).Where("purchase_order_id = ?", po.ID).Count(&grnCount).Error)
	require.EqualValues(t, 0, grnCount, "both rejected receive attempts must leave no stranded GRN")
}
