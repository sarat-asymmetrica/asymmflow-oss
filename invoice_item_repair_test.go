package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// PH convergence B1 (PH 3c5127b): the won-import path populates invoice line
// items inline via repairInvoiceItemsFromOrder, so an invoice created
// mid-session is never hollow until the next restart's backfill. The helper is
// shared with BackfillInvoiceItemsFromOrders — these tests pin its contract.
func TestRepairInvoiceItemsFromOrder_PopulatesFromOrderItems(t *testing.T) {
	a := setupTestApp(t)

	order := Order{OrderNumber: "ORD-26-4001", CustomerName: "Nimbus Controls"}
	require.NoError(t, a.db.Create(&order).Error)
	require.NoError(t, a.db.Create(&OrderItem{
		OrderID: order.ID, LineNumber: 1, Description: "Pressure transmitter",
		Quantity: 2, UnitPrice: 150, TotalPrice: 300, ProductCode: "PT-100",
	}).Error)
	require.NoError(t, a.db.Create(&OrderItem{
		OrderID: order.ID, LineNumber: 2, Description: "Impulse manifold",
		Quantity: 1, UnitPrice: 45, TotalPrice: 45,
	}).Error)

	hollow := Invoice{InvoiceNumber: "INV-26-4001", Status: "Draft", OrderID: order.ID, CustomerName: "Nimbus Controls", SubtotalBHD: 345, GrandTotalBHD: 379.5}
	require.NoError(t, a.db.Create(&hollow).Error)

	didRepair, invoiceNumber, err := a.repairInvoiceItemsFromOrder(hollow.ID)
	require.NoError(t, err)
	require.True(t, didRepair)
	require.Equal(t, "INV-26-4001", invoiceNumber)

	var items []DBInvoiceItem
	require.NoError(t, a.db.Where("invoice_id = ?", hollow.ID).Order("line_number").Find(&items).Error)
	require.Len(t, items, 2)
	require.Equal(t, "Pressure transmitter", items[0].Description)
	require.InDelta(t, 150.0, items[0].Rate, 0.0001)
	require.InDelta(t, 300.0, items[0].TotalBHD, 0.0001)

	// Idempotent on re-import: delete-then-create must not duplicate lines.
	didRepair, _, err = a.repairInvoiceItemsFromOrder(hollow.ID)
	require.NoError(t, err)
	require.True(t, didRepair)
	var count int64
	require.NoError(t, a.db.Model(&DBInvoiceItem{}).Where("invoice_id = ? AND deleted_at IS NULL", hollow.ID).Count(&count).Error)
	require.EqualValues(t, 2, count)
}

func TestRepairInvoiceItemsFromOrder_SyntheticLineWhenOrderHasNoItems(t *testing.T) {
	a := setupTestApp(t)

	order := Order{OrderNumber: "ORD-26-4002", CustomerName: "Atlas Traders", TotalValueBHD: 200}
	require.NoError(t, a.db.Create(&order).Error)
	inv := Invoice{InvoiceNumber: "INV-26-4002", Status: "Draft", OrderID: order.ID, CustomerName: "Atlas Traders", SubtotalBHD: 200}
	require.NoError(t, a.db.Create(&inv).Error)

	didRepair, _, err := a.repairInvoiceItemsFromOrder(inv.ID)
	require.NoError(t, err)
	require.True(t, didRepair)

	var items []DBInvoiceItem
	require.NoError(t, a.db.Where("invoice_id = ?", inv.ID).Find(&items).Error)
	require.Len(t, items, 1)
	require.Contains(t, items[0].Description, "Per Order ORD-26-4002")
	require.InDelta(t, 200.0, items[0].TotalBHD, 0.0001)
}

func TestRepairInvoiceItemsFromOrder_SkipsWithoutUsableOrder(t *testing.T) {
	a := setupTestApp(t)

	orphan := Invoice{InvoiceNumber: "INV-26-4003", Status: "Draft", OrderID: "no-such-order", CustomerName: "Meridian Instruments GmbH"}
	require.NoError(t, a.db.Create(&orphan).Error)

	didRepair, _, err := a.repairInvoiceItemsFromOrder(orphan.ID)
	require.NoError(t, err)
	require.False(t, didRepair, "an invoice with no loadable order is skipped, not an error")
}
