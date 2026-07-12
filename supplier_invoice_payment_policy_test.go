package main

// Wave 8 P1: MarkSupplierInvoicePaid must write a reconciling SupplierPayment for
// the remaining balance so the payment ledger sums to the invoice total, and the
// paid/outstanding state is derived from that ledger (PH parity) — not a bare flag.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarkSupplierInvoicePaid_WritesReconcilingLedgerRow(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&SupplierInvoice{}, &SupplierPayment{}))

	inv := SupplierInvoice{
		Base:          Base{ID: "si-1"},
		InvoiceNumber: "SINV-001",
		SupplierID:    "sup-1",
		SupplierName:  "Acme Instrumentation",
		Status:        "Approved",
		Currency:      "BHD",
		ExchangeRate:  1,
		TotalBHD:      100.000,
		TotalForeign:  100.000,
	}
	require.NoError(t, app.db.Create(&inv).Error)

	require.NoError(t, app.MarkSupplierInvoicePaid("si-1", "REF-123", "Bank Transfer"))

	// Exactly one reconciling payment for the full balance must exist.
	var payments []SupplierPayment
	require.NoError(t, app.db.Where("supplier_invoice_id = ?", "si-1").Find(&payments).Error)
	require.Len(t, payments, 1)
	require.InDelta(t, 100.0, payments[0].AmountBHD, 0.0001,
		"the ledger must reconcile: SUM(payments) == TotalBHD")

	// State derived from the ledger: outstanding zero → Paid.
	var reloaded SupplierInvoice
	require.NoError(t, app.db.First(&reloaded, "id = ?", "si-1").Error)
	require.Equal(t, "Paid", reloaded.Status)
	require.Equal(t, "Paid", reloaded.PaymentStatus)
}

func TestMarkSupplierInvoicePaid_PartialPaymentTopsUpToTotal(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&SupplierInvoice{}, &SupplierPayment{}))

	inv := SupplierInvoice{
		Base:          Base{ID: "si-2"},
		InvoiceNumber: "SINV-002",
		SupplierID:    "sup-2",
		Status:        "Approved",
		Currency:      "BHD",
		ExchangeRate:  1,
		TotalBHD:      100.000,
		TotalForeign:  100.000,
	}
	require.NoError(t, app.db.Create(&inv).Error)

	// A prior partial payment of 40 already exists.
	require.NoError(t, app.db.Create(&SupplierPayment{
		Base:              Base{ID: "sp-prior"},
		SupplierInvoiceID: "si-2",
		AmountBHD:         40.000,
		AmountForeign:     40.000,
		Currency:          "BHD",
		ExchangeRate:      1,
		PaymentMethod:     "Cash",
	}).Error)

	require.NoError(t, app.MarkSupplierInvoicePaid("si-2", "REF-TOPUP", "Cheque"))

	// Only the remaining 60 should be added — total ledger == 100, not 140.
	var total float64
	require.NoError(t, app.db.Model(&SupplierPayment{}).
		Where("supplier_invoice_id = ?", "si-2").
		Select("COALESCE(SUM(amount_bhd),0)").Scan(&total).Error)
	require.InDelta(t, 100.0, total, 0.0001,
		"mark-as-paid must top up to the total, not double-pay")
}
