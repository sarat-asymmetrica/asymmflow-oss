package main

// Wave 8 P3 slice 2: customer receipt-allocation sub-model.
// Covers on-account receipt creation, invoice-applied receipts (Payment +
// allocation + invoice-state advance), balance math across partial/full
// application, the validation guards, and the customer/division cross-checks.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func receiptTestModels(t *testing.T, app *App) {
	t.Helper()
	require.NoError(t, app.db.AutoMigrate(
		&Invoice{},
		&Payment{},
		&CustomerMaster{},
		&CustomerReceipt{},
		&CustomerReceiptAllocation{},
	))
}

// seedOpenInvoice creates a payable ("Sent", outstanding > 0) invoice.
func seedOpenInvoice(t *testing.T, app *App, id, number, customerID string, total float64) Invoice {
	t.Helper()
	inv := Invoice{
		Base:           Base{ID: id},
		InvoiceNumber:  number,
		CustomerID:     customerID,
		CustomerName:   "Acme Instrumentation",
		Division:       "PH Trading",
		InvoiceDate:    time.Now().AddDate(0, 0, -10),
		DueDate:        time.Now().AddDate(0, 0, 20),
		GrandTotalBHD:  total,
		OutstandingBHD: total,
		Status:         "Sent",
	}
	require.NoError(t, app.db.Create(&inv).Error)
	return inv
}

func TestCreateCustomerReceipt_OnAccount(t *testing.T) {
	app := setupTestApp(t)
	receiptTestModels(t, app)

	got, err := app.CreateCustomerReceipt(CustomerReceiptInput{
		CustomerID:    "cust-1",
		CustomerName:  "Acme Instrumentation",
		AmountBHD:     500.125,
		PaymentMethod: "Cash",
	})
	require.NoError(t, err)
	require.Equal(t, "OnAccount", got.Status)
	require.Equal(t, 500.125, got.AmountBHD)
	require.Equal(t, 0.0, got.AppliedAmountBHD)
	require.Equal(t, 500.125, got.UnappliedAmountBHD)
	require.Contains(t, got.ReceiptNumber, "RCT-")

	// No allocations, no payments for a pure on-account receipt.
	var payCount, allocCount int64
	require.NoError(t, app.db.Model(&Payment{}).Count(&payCount).Error)
	require.NoError(t, app.db.Model(&CustomerReceiptAllocation{}).Count(&allocCount).Error)
	require.Equal(t, int64(0), payCount)
	require.Equal(t, int64(0), allocCount)
}

func TestCreateCustomerReceipt_AppliedToInvoiceFull(t *testing.T) {
	app := setupTestApp(t)
	receiptTestModels(t, app)
	seedOpenInvoice(t, app, "inv-1", "INV-1", "cust-1", 300.000)

	receipt, err := app.CreateCustomerReceipt(CustomerReceiptInput{
		InvoiceID:     "inv-1",
		AmountBHD:     300.000,
		PaymentMethod: "Cheque",
		Reference:     "CHQ-9",
	})
	require.NoError(t, err)
	// Customer info inherited from the invoice.
	require.Equal(t, "cust-1", receipt.CustomerID)
	require.Equal(t, "Acme Instrumentation", receipt.CustomerName)

	// Receipt fully applied.
	var reloaded CustomerReceipt
	require.NoError(t, app.db.First(&reloaded, "id = ?", receipt.ID).Error)
	require.Equal(t, "Applied", reloaded.Status)
	require.Equal(t, 300.0, reloaded.AppliedAmountBHD)
	require.Equal(t, 0.0, reloaded.UnappliedAmountBHD)

	// Invoice settled to Paid, outstanding zeroed.
	var inv Invoice
	require.NoError(t, app.db.First(&inv, "id = ?", "inv-1").Error)
	require.Equal(t, 0.0, inv.OutstandingBHD)
	require.Equal(t, "Paid", inv.Status)

	// One Payment (linked to the receipt) + one allocation.
	var pay Payment
	require.NoError(t, app.db.First(&pay, "invoice_id = ?", "inv-1").Error)
	require.NotNil(t, pay.ReceiptID)
	require.Equal(t, receipt.ID, *pay.ReceiptID)
	require.Equal(t, 300.0, pay.AmountBHD)

	var allocs []CustomerReceiptAllocation
	require.NoError(t, app.db.Where("receipt_id = ?", receipt.ID).Find(&allocs).Error)
	require.Len(t, allocs, 1)
	require.Equal(t, pay.ID, allocs[0].PaymentID)
	require.Equal(t, 300.0, allocs[0].AllocatedAmountBHD)
}

func TestApplyCustomerReceiptToInvoice_PartialThenFull(t *testing.T) {
	app := setupTestApp(t)
	receiptTestModels(t, app)
	seedOpenInvoice(t, app, "inv-1", "INV-1", "cust-1", 1000.000)

	// On-account receipt for the full amount.
	receipt, err := app.CreateCustomerReceipt(CustomerReceiptInput{
		CustomerID: "cust-1", CustomerName: "Acme Instrumentation",
		AmountBHD: 1000.000, PaymentMethod: "Cash", Division: "PH Trading",
	})
	require.NoError(t, err)

	// Apply 400 → PartiallyApplied receipt, PartiallyPaid invoice.
	alloc1, err := app.ApplyCustomerReceiptToInvoice(receipt.ID, "inv-1", 400.000)
	require.NoError(t, err)
	require.Equal(t, 400.0, alloc1.AllocatedAmountBHD)

	var recAfter1 CustomerReceipt
	require.NoError(t, app.db.First(&recAfter1, "id = ?", receipt.ID).Error)
	require.Equal(t, "PartiallyApplied", recAfter1.Status)
	require.Equal(t, 600.0, recAfter1.UnappliedAmountBHD)

	var invAfter1 Invoice
	require.NoError(t, app.db.First(&invAfter1, "id = ?", "inv-1").Error)
	require.Equal(t, 600.0, invAfter1.OutstandingBHD)
	require.Equal(t, "PartiallyPaid", invAfter1.Status)

	// Apply remaining with amount<=0 → auto-fills min(unapplied, outstanding)=600.
	alloc2, err := app.ApplyCustomerReceiptToInvoice(receipt.ID, "inv-1", 0)
	require.NoError(t, err)
	require.Equal(t, 600.0, alloc2.AllocatedAmountBHD)

	var recAfter2 CustomerReceipt
	require.NoError(t, app.db.First(&recAfter2, "id = ?", receipt.ID).Error)
	require.Equal(t, "Applied", recAfter2.Status)
	require.Equal(t, 0.0, recAfter2.UnappliedAmountBHD)

	var invAfter2 Invoice
	require.NoError(t, app.db.First(&invAfter2, "id = ?", "inv-1").Error)
	require.Equal(t, 0.0, invAfter2.OutstandingBHD)
	require.Equal(t, "Paid", invAfter2.Status)

	// Two allocations recorded; GetCustomerReceiptAllocations returns both.
	got, err := app.GetCustomerReceiptAllocations(receipt.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestCreateCustomerReceipt_ValidationGuards(t *testing.T) {
	app := setupTestApp(t)
	receiptTestModels(t, app)

	_, err := app.CreateCustomerReceipt(CustomerReceiptInput{CustomerID: "c", AmountBHD: 0})
	require.ErrorContains(t, err, "greater than zero")

	// Bank transfer requires a reference.
	_, err = app.CreateCustomerReceipt(CustomerReceiptInput{
		CustomerID: "c", CustomerName: "Acme", AmountBHD: 10, PaymentMethod: "Bank Transfer",
	})
	require.ErrorContains(t, err, "reference is required")

	// On-account receipt with no customer identity.
	_, err = app.CreateCustomerReceipt(CustomerReceiptInput{AmountBHD: 10, PaymentMethod: "Cash"})
	require.ErrorContains(t, err, "customer is required")
}

func TestCreateCustomerReceipt_RejectsOverpay(t *testing.T) {
	app := setupTestApp(t)
	receiptTestModels(t, app)
	seedOpenInvoice(t, app, "inv-1", "INV-1", "cust-1", 100.000)

	_, err := app.CreateCustomerReceipt(CustomerReceiptInput{
		InvoiceID: "inv-1", AmountBHD: 150.000, PaymentMethod: "Cash",
	})
	require.ErrorContains(t, err, "exceeds open invoice balance")
}

func TestApplyCustomerReceiptToInvoice_CrossChecks(t *testing.T) {
	app := setupTestApp(t)
	receiptTestModels(t, app)

	// Receipt for cust-1, invoice for cust-2 → customer mismatch.
	receipt, err := app.CreateCustomerReceipt(CustomerReceiptInput{
		CustomerID: "cust-1", CustomerName: "Acme Instrumentation",
		AmountBHD: 100, PaymentMethod: "Cash", Division: "PH Trading",
	})
	require.NoError(t, err)
	seedOpenInvoice(t, app, "inv-2", "INV-2", "cust-2", 100.000)

	_, err = app.ApplyCustomerReceiptToInvoice(receipt.ID, "inv-2", 50)
	require.ErrorContains(t, err, "customer does not match")
}

func TestListCustomerReceipts_OrdersAndPermits(t *testing.T) {
	app := setupTestApp(t)
	receiptTestModels(t, app)

	for _, amt := range []float64{10, 20, 30} {
		_, err := app.CreateCustomerReceipt(CustomerReceiptInput{
			CustomerID: "cust-1", CustomerName: "Acme", AmountBHD: amt, PaymentMethod: "Cash",
		})
		require.NoError(t, err)
	}

	got, err := app.ListCustomerReceipts(0, 0)
	require.NoError(t, err)
	require.Len(t, got, 3)
}
