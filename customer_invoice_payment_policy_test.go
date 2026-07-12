package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRecordPayment_RejectsDraftInvoice pins the Mission G fix: a Draft (unsent)
// invoice is a closed workflow status and must not accept a payment. The prior
// inline non-payable map omitted "Draft".
func TestRecordPayment_RejectsDraftInvoice(t *testing.T) {
	app := setupPaymentTestApp(t)

	inv := &Invoice{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		InvoiceNumber:  "INV-DRAFT-1",
		InvoiceDate:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		GrandTotalBHD:  1000.000,
		OutstandingBHD: 1000.000,
		Status:         "Draft",
	}
	require.NoError(t, app.db.Create(inv).Error)

	pay, err := app.RecordPayment(inv.ID, 100.000, "Cash", "2026-01-15", "REF")
	require.Error(t, err)
	assert.Nil(t, pay)
	assert.Contains(t, err.Error(), "Draft")

	// No payment row created.
	var count int64
	app.db.Model(&Payment{}).Where("invoice_id = ?", inv.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// TestHydrateCustomerInvoicePaymentState_Overdue pins the read-hydration parity
// fix: a stored-"Sent" invoice past its due date surfaces as "Overdue" on read
// (this pure helper is what ListCustomerInvoices / GetCustomerInvoiceByID now
// apply before returning).
func TestHydrateCustomerInvoicePaymentState_Overdue(t *testing.T) {
	inv := Invoice{
		Base:           Base{ID: uuid.New().String()},
		InvoiceNumber:  "INV-OD-1",
		InvoiceDate:    time.Now().AddDate(0, 0, -60),
		DueDate:        time.Now().AddDate(0, 0, -30), // 30 days past due
		GrandTotalBHD:  1000.000,
		OutstandingBHD: 1000.000,
		Status:         "Sent", // stored stale — never transitioned to Overdue
	}

	hydrateCustomerInvoicePaymentState(&inv)
	assert.Equal(t, "Overdue", inv.Status, "past-due open invoice should hydrate to Overdue")

	// A Draft invoice is a closed workflow status: hydration must NOT flip it to a
	// settlement status regardless of dates.
	draft := Invoice{
		Base:           Base{ID: uuid.New().String()},
		DueDate:        time.Now().AddDate(0, 0, -30),
		GrandTotalBHD:  500.000,
		OutstandingBHD: 500.000,
		Status:         "Draft",
	}
	hydrateCustomerInvoicePaymentState(&draft)
	assert.Equal(t, "Draft", draft.Status)

	// A fully-settled invoice hydrates to Paid.
	paid := Invoice{
		Base:           Base{ID: uuid.New().String()},
		DueDate:        time.Now().AddDate(0, 0, -5),
		GrandTotalBHD:  200.000,
		OutstandingBHD: 0.000,
		Status:         "Sent",
	}
	hydrateCustomerInvoicePaymentState(&paid)
	assert.Equal(t, "Paid", paid.Status)
}
