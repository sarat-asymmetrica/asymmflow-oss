package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestUpdateSupplierInvoice_CannotBypassPaymentLifecycle proves the B1 fix:
// the plain descriptive UpdateSupplierInvoice endpoint can no longer be used
// as an off-ledger shortcut to mark an invoice Paid/Approved. Before the fix,
// passing PaymentStatus="Paid" (the same payload the old Edit-modal "Save
// Changes" button sent) flipped Status to "Paid" and stamped PaymentDate
// without ever going through PerformThreeWayMatch -> ApproveSupplierInvoice
// -> MarkSupplierInvoicePaid, and without creating the reconciling
// SupplierPayment ledger entry. This test drives the same bypass payload and
// asserts it is now a no-op on the lifecycle fields, while descriptive fields
// (invoice number) still update normally.
func TestUpdateSupplierInvoice_CannotBypassPaymentLifecycle(t *testing.T) {
	app := setupPostingPreviewTestApp(t)

	now := time.Now()
	invoiceID := uuid.New().String()
	original := SupplierInvoice{
		Base:            Base{ID: invoiceID, CreatedAt: now, UpdatedAt: now, CreatedBy: "creator-user"},
		SupplierID:      "supplier-1",
		InvoiceNumber:   "SINV-BYPASS-001",
		InvoiceDate:     now,
		DueDate:         now.AddDate(0, 0, 30),
		Currency:        "BHD",
		ExchangeRate:    1,
		SubtotalForeign: 100,
		SubtotalBHD:     100,
		TotalForeign:    100,
		TotalBHD:        100,
		Status:          "Pending",
		PaymentStatus:   "Unpaid",
		MatchStatus:     "Pending",
	}
	if err := app.db.Create(&original).Error; err != nil {
		t.Fatalf("seed invoice: %v", err)
	}

	// Same shape of payload the old Edit-modal bypass sent: PaymentStatus
	// "Paid" (which used to force Status to "Paid" too), plus a descriptive
	// change (invoice number) to prove non-lifecycle edits still work.
	bypassPayload := original
	bypassPayload.InvoiceNumber = "SINV-BYPASS-001-EDITED"
	bypassPayload.Status = "Paid"
	bypassPayload.PaymentStatus = "Paid"
	stampedDate := now.Add(24 * time.Hour)
	bypassPayload.PaymentDate = &stampedDate
	bypassPayload.PaymentRef = "FAKE-REF"
	bypassPayload.PaymentMethod = "Bank Transfer"

	updated, err := app.UpdateSupplierInvoice(bypassPayload)
	if err != nil {
		t.Fatalf("UpdateSupplierInvoice: %v", err)
	}

	if updated.Status != "Pending" {
		t.Fatalf("expected Status to stay 'Pending' (masked), got %q", updated.Status)
	}
	if updated.PaymentStatus != "Unpaid" {
		t.Fatalf("expected PaymentStatus to stay 'Unpaid' (masked), got %q", updated.PaymentStatus)
	}
	if updated.PaymentDate != nil {
		t.Fatalf("expected PaymentDate to stay nil (masked), got %v", updated.PaymentDate)
	}
	if updated.PaymentRef != "" {
		t.Fatalf("expected PaymentRef to stay empty (masked), got %q", updated.PaymentRef)
	}
	if updated.PaymentMethod != "" {
		t.Fatalf("expected PaymentMethod to stay empty (masked), got %q", updated.PaymentMethod)
	}
	if updated.InvoiceNumber != "SINV-BYPASS-001-EDITED" {
		t.Fatalf("expected descriptive field (invoice number) to update, got %q", updated.InvoiceNumber)
	}

	// Reload from the DB — not just the in-memory return value — to make
	// sure nothing slipped through Save() into the persisted row.
	var persisted SupplierInvoice
	if err := app.db.Where("id = ?", invoiceID).First(&persisted).Error; err != nil {
		t.Fatalf("reload invoice: %v", err)
	}
	if persisted.Status != "Pending" {
		t.Fatalf("persisted Status bypassed the mask: got %q", persisted.Status)
	}
	if persisted.PaymentStatus != "Unpaid" {
		t.Fatalf("persisted PaymentStatus bypassed the mask: got %q", persisted.PaymentStatus)
	}
	if persisted.PaymentDate != nil {
		t.Fatalf("persisted PaymentDate bypassed the mask: got %v", persisted.PaymentDate)
	}

	// No reconciling SupplierPayment should have been created — this
	// endpoint has no business writing to the payment ledger at all.
	var paymentCount int64
	app.db.Model(&SupplierPayment{}).Where("supplier_invoice_id = ?", invoiceID).Count(&paymentCount)
	if paymentCount != 0 {
		t.Fatalf("expected no SupplierPayment ledger entries, got %d", paymentCount)
	}
}
