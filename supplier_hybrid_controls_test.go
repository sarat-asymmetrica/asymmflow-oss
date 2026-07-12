package main

// Wave 8 P5-1 (user-ratified hybrid): supplier-invoice controls — OSS status
// vocabulary retained, PH's segregation-of-duties restored. Match verifies the
// paperwork (Status="Verified"), a DIFFERENT human approves the disbursement
// (ApproveSupplierInvoice → "Approved"), and only then does cash leave. Every
// payment settles invoice.Status through the payment-state policy.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSupplierHybrid_VerifyApprovePaySettles(t *testing.T) {
	app := setupSupplierGateTestApp(t)
	inv := makeSupplierInvoice(t, app, "SI-HYB-1", "Verified", "Matched", 500.000)

	// Verified alone cannot be paid — the approval step is mandatory.
	_, err := app.RecordSupplierPayment(inv.ID, 200.000, "BHD", "Bank Transfer", "2026-01-10", "HYB-0", 1.0)
	require.Error(t, err)

	// Segregated approval (creator "creator-user", approver differs).
	require.NoError(t, app.ApproveSupplierInvoice(inv.ID, "approver-user"))

	// Partial payment: ledger row created, invoice settles to Approved/Partial.
	pay, err := app.RecordSupplierPayment(inv.ID, 200.000, "BHD", "Bank Transfer", "2026-01-15", "HYB-1", 1.0)
	require.NoError(t, err)
	require.NotNil(t, pay)

	var afterPartial SupplierInvoice
	require.NoError(t, app.db.First(&afterPartial, "id = ?", inv.ID).Error)
	require.Equal(t, "Partial", afterPartial.PaymentStatus)
	require.Equal(t, "Approved", afterPartial.Status, "a partially paid invoice stays Approved, not Paid")

	// Remainder: invoice.Status itself settles to Paid (the settlement-policy
	// port — the audit found OSS used to flip payment_status only).
	_, err = app.RecordSupplierPayment(inv.ID, 300.000, "BHD", "Bank Transfer", "2026-01-20", "HYB-2", 1.0)
	require.NoError(t, err)

	var afterFull SupplierInvoice
	require.NoError(t, app.db.First(&afterFull, "id = ?", inv.ID).Error)
	require.Equal(t, "Paid", afterFull.PaymentStatus)
	require.Equal(t, "Paid", afterFull.Status, "full settlement must reach invoice.Status")

	// Ledger reconciles: SUM(payments) == TotalBHD.
	var totalPaid float64
	require.NoError(t, app.db.Model(&SupplierPayment{}).
		Where("supplier_invoice_id = ?", inv.ID).
		Select("COALESCE(SUM(amount_bhd), 0)").Scan(&totalPaid).Error)
	require.InDelta(t, 500.000, totalPaid, 0.001)
}

func TestSupplierInvoiceNonPaymentStatus_PreservesOSSVocabulary(t *testing.T) {
	// The hydrate/settlement policy must not collapse OSS workflow states:
	// "Verified" (clean match awaiting approval) and "Disputed" survive.
	verified := SupplierInvoice{Status: "Verified"}
	require.Equal(t, "Verified", supplierInvoiceNonPaymentStatus(verified))

	now := time.Now()
	verifiedApproved := SupplierInvoice{Status: "Verified", ApprovedAt: &now}
	require.Equal(t, "Approved", supplierInvoiceNonPaymentStatus(verifiedApproved))

	disputed := SupplierInvoice{Status: "Disputed"}
	require.Equal(t, "Disputed", supplierInvoiceNonPaymentStatus(disputed))

	legacyDispute := SupplierInvoice{Status: "Dispute"}
	require.Equal(t, "Dispute", supplierInvoiceNonPaymentStatus(legacyDispute))

	pending := SupplierInvoice{Status: "Pending"}
	require.Equal(t, "Pending", supplierInvoiceNonPaymentStatus(pending))
}
