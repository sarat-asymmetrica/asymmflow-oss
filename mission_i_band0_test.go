package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MISSION I — BAND 0 GOLDEN TESTS (I-01, I-02, I-04)
// =============================================================================
// Each test fails without its fix. The fixes route every customer-invoice
// status/outstanding mutation and read through the settlement policy in
// customer_invoice_payment_policy.go, matching deployed PH.
// =============================================================================

func makeMissionIInvoice(t *testing.T, app *App, number, status string, grand, outstanding float64, due time.Time) *Invoice {
	t.Helper()
	require.NoError(t, app.db.AutoMigrate(&DBInvoiceItem{}))
	inv := &Invoice{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		InvoiceNumber:  number,
		CustomerID:     "CUST-1",
		InvoiceDate:    due.AddDate(0, -1, 0),
		DueDate:        due,
		SubtotalBHD:    grand,
		GrandTotalBHD:  grand,
		OutstandingBHD: outstanding,
		Status:         status,
	}
	require.NoError(t, app.db.Create(inv).Error)
	return inv
}

// I-01: UpdatePayment must reject edits that land a payment on a Draft invoice.
// The old inline map ({Cancelled, Void, Proforma}) omitted "Draft".
func TestUpdatePayment_RejectsDraftInvoice(t *testing.T) {
	app := setupPaymentTestApp(t)
	inv := makeMissionIInvoice(t, app, "INV-I01-UP", "Draft", 1000.000, 1000.000, time.Now().AddDate(0, 1, 0))

	// Seed a payment row directly (bypassing RecordPayment's own gate) to
	// isolate the update-path check.
	pay := &Payment{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		InvoiceID:     inv.ID,
		InvoiceNumber: inv.InvoiceNumber,
		AmountBHD:     100.000,
		PaymentDate:   time.Now(),
		PaymentMethod: "Cash",
	}
	require.NoError(t, app.db.Create(pay).Error)

	updated := *pay
	updated.AmountBHD = 200.000
	res, err := app.UpdatePayment(pay.ID, updated)
	require.Error(t, err, "editing a payment onto a Draft invoice must be refused")
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "Draft")
}

// I-01: RecordPartialPayment carried its own stale inline map missing "Draft".
func TestRecordPartialPayment_RejectsDraftInvoice(t *testing.T) {
	app := setupPaymentTestApp(t)
	inv := makeMissionIInvoice(t, app, "INV-I01-PP", "Draft", 1000.000, 1000.000, time.Now().AddDate(0, 1, 0))

	err := app.RecordPartialPayment(inv.ID, 100.000, time.Now(), "REF-PP")
	require.Error(t, err, "partial payment on a Draft invoice must be refused")

	var count int64
	app.db.Model(&Payment{}).Where("invoice_id = ?", inv.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// I-02: list read paths must hydrate settlement state — a stored-"Sent" invoice
// past its due date surfaces as "Overdue" from GetInvoicesByCustomer.
func TestGetInvoicesByCustomer_HydratesOverdue(t *testing.T) {
	app := setupPaymentTestApp(t)
	makeMissionIInvoice(t, app, "INV-I02-BC", "Sent", 1000.000, 1000.000, time.Now().AddDate(0, 0, -30))

	invoices, err := app.GetInvoicesByCustomer("CUST-1")
	require.NoError(t, err)
	require.Len(t, invoices, 1)
	assert.Equal(t, "Overdue", invoices[0].Status, "past-due open invoice must display as Overdue")
}

// I-02: GetInvoicesByStatus is one of the previously-unwired read paths.
func TestGetInvoicesByStatus_HydratesOverdue(t *testing.T) {
	app := setupPaymentTestApp(t)
	makeMissionIInvoice(t, app, "INV-I02-ST", "Sent", 1000.000, 1000.000, time.Now().AddDate(0, 0, -30))

	invoices, err := app.GetInvoicesByStatus("Sent")
	require.NoError(t, err)
	require.Len(t, invoices, 1)
	assert.Equal(t, "Overdue", invoices[0].Status)
}

// I-04: MarkCustomerInvoicePaid is a payment EVENT — it must create a Payment
// row for the audit trail, not silently zero the balance. The old inline
// version left money disappearing with no payment record.
func TestMarkCustomerInvoicePaid_CreatesPaymentRecord(t *testing.T) {
	app := setupPaymentTestApp(t)
	inv := makeMissionIInvoice(t, app, "INV-I04-MP", "Sent", 750.000, 750.000, time.Now().AddDate(0, 1, 0))

	require.NoError(t, app.MarkCustomerInvoicePaid(inv.ID, time.Now(), "REF-MP"))

	var reloaded Invoice
	require.NoError(t, app.db.First(&reloaded, "id = ?", inv.ID).Error)
	assert.Equal(t, "Paid", reloaded.Status)
	assert.Equal(t, 0.0, reloaded.OutstandingBHD)

	var payments []Payment
	require.NoError(t, app.db.Where("invoice_id = ?", inv.ID).Find(&payments).Error)
	require.Len(t, payments, 1, "mark-paid must leave a payment audit record")
	assert.InDelta(t, 750.000, payments[0].AmountBHD, 0.0005)
}

// I-04: MarkCustomerInvoiceOverdue must DERIVE overdue from the settlement
// policy — the old version stamped "Overdue" even when the due date had not
// passed.
func TestMarkCustomerInvoiceOverdue_DerivedNotAsserted(t *testing.T) {
	app := setupPaymentTestApp(t)

	// Not yet due — must refuse.
	fresh := makeMissionIInvoice(t, app, "INV-I04-OD1", "Sent", 500.000, 500.000, time.Now().AddDate(0, 1, 0))
	err := app.MarkCustomerInvoiceOverdue(fresh.ID)
	require.Error(t, err, "an invoice that is not past due must not be markable Overdue")

	var reloaded Invoice
	require.NoError(t, app.db.First(&reloaded, "id = ?", fresh.ID).Error)
	assert.Equal(t, "Sent", reloaded.Status)

	// Genuinely past due — must persist Overdue.
	late := makeMissionIInvoice(t, app, "INV-I04-OD2", "Sent", 500.000, 500.000, time.Now().AddDate(0, 0, -10))
	require.NoError(t, app.MarkCustomerInvoiceOverdue(late.ID))
	var lateReloaded Invoice
	require.NoError(t, app.db.First(&lateReloaded, "id = ?", late.ID).Error)
	assert.Equal(t, "Overdue", lateReloaded.Status)
}

// I-08 guard (ported with I-02 wiring): UpdateCustomerInvoice must refuse a
// client payload that edits the payment-driven Outstanding balance or hand-sets
// a settlement status.
func TestUpdateCustomerInvoice_PaymentDrivenFieldsLocked(t *testing.T) {
	app := setupPaymentTestApp(t)
	inv := makeMissionIInvoice(t, app, "INV-I08-UP", "Sent", 900.000, 900.000, time.Now().AddDate(0, 1, 0))

	// Client tries to wipe the balance.
	tampered := *inv
	tampered.OutstandingBHD = 0
	_, err := app.UpdateCustomerInvoice(tampered)
	require.Error(t, err, "outstanding balance must not be editable")
	assert.Contains(t, err.Error(), "payment-driven")

	// Client tries to hand-set a settlement status.
	tampered = *inv
	tampered.Status = "PartiallyPaid"
	_, err = app.UpdateCustomerInvoice(tampered)
	require.Error(t, err, "settlement statuses must be derived, not set")

	var reloaded Invoice
	require.NoError(t, app.db.First(&reloaded, "id = ?", inv.ID).Error)
	assert.Equal(t, "Sent", reloaded.Status)
	assert.InDelta(t, 900.000, reloaded.OutstandingBHD, 0.0005)
}
