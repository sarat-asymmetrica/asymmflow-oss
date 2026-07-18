package main

// INTEG campaign — Wave I3 (AP batch) persistence validation.
// DeleteSupplierPayment had no prior coverage. As an admin session it deletes
// directly (a non-admin would instead raise a delete-approval request via the
// guard); this drives the bound App method against a scratch SQLite and asserts
// the payment is removed and a missing id is refused.
//
// PerformThreeWayMatch / Approve / MarkPaid are covered by supplier_ap_gate_test.go.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestIntegAP_DeleteSupplierPayment(t *testing.T) {
	app := setupTestApp(t)
	// The delete rolls back the linked supplier invoice's paid/outstanding — both
	// tables must exist.
	require.NoError(t, app.db.AutoMigrate(&SupplierPayment{}, &SupplierInvoice{}), "migrate AP tables")

	// A supplier invoice partly paid down by the payment we'll delete. The delete
	// re-derives payment_status from the payments that remain (no stored paid col).
	require.NoError(t, app.db.Create(&SupplierInvoice{
		Base:          Base{ID: "si-1"},
		InvoiceNumber: "SI-T-0001",
		TotalBHD:      2000.000,
		PaymentStatus: "Partial",
	}).Error)

	require.NoError(t, app.db.Create(&SupplierPayment{
		Base:              Base{ID: "sp-1"},
		SupplierInvoiceID: "si-1",
		PaymentNumber:     "SP-T-0001",
		AmountBHD:         1250.500,
		Currency:          "BHD",
		ExchangeRate:      1,
		PaymentDate:       time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		PaymentMethod:     "Cash",
		Reference:         "REF-T-1",
	}).Error)

	// Admin session → direct delete (no approval request).
	require.NoError(t, app.DeleteSupplierPayment("sp-1"))

	var got SupplierPayment
	err := app.db.Where("id = ?", "sp-1").First(&got).Error
	require.ErrorIs(t, err, gorm.ErrRecordNotFound, "the payment must be gone after delete")

	// The linked invoice's payment_status was rolled back (no payments remain → Unpaid).
	var si SupplierInvoice
	require.NoError(t, app.db.Where("id = ?", "si-1").First(&si).Error)
	require.Equal(t, "Unpaid", si.PaymentStatus, "deleting the only payment re-derives status to Unpaid")

	// Deleting a non-existent payment is refused, not a silent success.
	require.Error(t, app.DeleteSupplierPayment("sp-does-not-exist"))
}
