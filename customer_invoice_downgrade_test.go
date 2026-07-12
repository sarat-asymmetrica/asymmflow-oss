package main

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestUpdateCustomerInvoice_CannotDowngradePostedToDraft proves the Wave 9.6
// AR1 fix: a Sent (posted) customer invoice can no longer be reverted to
// Draft via UpdateCustomerInvoice, and its line items can no longer be
// rewritten in the same call. Before the fix, `existing.Status` was
// reassigned to the client-sent value BEFORE the item-replacement guard
// checked it, so sending Status="Draft" alongside replacement Items both
// downgraded the workflow status AND rewrote a numbered, aged/VAT-scoped
// invoice's line items in one shot — pulling it out of the books. This
// mirrors the supplier-side lifecycle protection proven by
// TestUpdateSupplierInvoice_CannotBypassPaymentLifecycle.
func TestUpdateCustomerInvoice_CannotDowngradePostedToDraft(t *testing.T) {
	app := setupTestApp(t)

	customerID := seedTestCustomer(t, app.db, "Downgrade Test Customer")
	now := time.Now()

	invoiceID := uuid.New().String()
	original := Invoice{
		Base:           Base{ID: invoiceID, CreatedAt: now, UpdatedAt: now},
		InvoiceNumber:  "INV-DOWNGRADE-001",
		CustomerID:     customerID,
		CustomerName:   "Downgrade Test Customer",
		InvoiceDate:    now,
		DueDate:        now.AddDate(0, 0, 30),
		Status:         "Sent",
		SubtotalBHD:    100,
		VATBHD:         10,
		GrandTotalBHD:  110,
		OutstandingBHD: 110,
		Items: []DBInvoiceItem{{
			Base:        Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
			LineNumber:  1,
			Description: "Original Widget",
			Quantity:    10,
			Rate:        10,
			TotalBHD:    100,
			TotalPrice:  100,
		}},
	}
	require.NoError(t, app.db.Create(&original).Error)

	// Same shape of payload the Edit modal could send before the fix:
	// Status downgraded to Draft, plus replacement line items.
	bypassPayload := original
	bypassPayload.Status = "Draft"
	bypassPayload.Items = []DBInvoiceItem{{
		Base:        Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceID:   invoiceID,
		LineNumber:  1,
		Description: "HACKED replacement item",
		Quantity:    1,
		Rate:        1,
		TotalBHD:    1,
		TotalPrice:  1,
	}}

	_, err := app.UpdateCustomerInvoice(bypassPayload)
	if err == nil {
		t.Fatalf("expected UpdateCustomerInvoice to reject Sent->Draft downgrade, got nil error")
	}
	if !strings.Contains(err.Error(), "cannot be reverted") {
		t.Fatalf("expected downgrade-rejection error, got: %v", err)
	}

	// Reload from the DB — not just the in-memory return value — to make
	// sure nothing slipped through Save() into the persisted row.
	var persisted Invoice
	require.NoError(t, app.db.Preload("Items").Where("id = ?", invoiceID).First(&persisted).Error)
	if persisted.Status != "Sent" {
		t.Fatalf("expected persisted Status to stay 'Sent', got %q", persisted.Status)
	}
	require.Len(t, persisted.Items, 1)
	if persisted.Items[0].Description != "Original Widget" {
		t.Fatalf("expected original line item to survive, got %q", persisted.Items[0].Description)
	}

	// Regression guard: a legitimate Draft -> Sent transition (existing is
	// NOT yet posted) must still succeed.
	draftID := uuid.New().String()
	draftInvoice := Invoice{
		Base:           Base{ID: draftID, CreatedAt: now, UpdatedAt: now},
		InvoiceNumber:  "INV-DOWNGRADE-002",
		CustomerID:     customerID,
		CustomerName:   "Downgrade Test Customer",
		InvoiceDate:    now,
		DueDate:        now.AddDate(0, 0, 30),
		Status:         "Draft",
		SubtotalBHD:    100,
		VATBHD:         10,
		GrandTotalBHD:  110,
		OutstandingBHD: 110,
		Items: []DBInvoiceItem{{
			Base:        Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
			LineNumber:  1,
			Description: "Draft Widget",
			Quantity:    10,
			Rate:        10,
			TotalBHD:    100,
			TotalPrice:  100,
		}},
	}
	require.NoError(t, app.db.Create(&draftInvoice).Error)

	promote := draftInvoice
	promote.Status = "Sent"

	updated, err := app.UpdateCustomerInvoice(promote)
	require.NoError(t, err, "legitimate Draft->Sent transition must still succeed")
	if updated.Status != "Sent" {
		t.Fatalf("expected Status to become 'Sent', got %q", updated.Status)
	}
}
