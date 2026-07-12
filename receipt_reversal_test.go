package main

// Wave 9 Spec-03 (C3): receipt reversal — fully-unapplied receipts only.
// Applied/posted receipt reversal is stop-and-report and is deliberately
// NOT implemented; these tests pin the guard that rejects it.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReverseCustomerReceipt_ZeroApplicationSucceeds(t *testing.T) {
	app := setupTestApp(t)
	receiptTestModels(t, app)

	receipt, err := app.CreateCustomerReceipt(CustomerReceiptInput{
		CustomerID: "cust-1", CustomerName: "Acme Instrumentation",
		AmountBHD: 250.000, PaymentMethod: "Cash", Division: "PH Trading",
	})
	require.NoError(t, err)
	require.Equal(t, "OnAccount", receipt.Status)

	reversed, err := app.ReverseCustomerReceipt(receipt.ID, "duplicate entry")
	require.NoError(t, err)
	require.Equal(t, "Reversed", reversed.Status)
	require.Equal(t, 0.0, reversed.UnappliedAmountBHD)
	require.Contains(t, reversed.Notes, "duplicate entry")

	var reloaded CustomerReceipt
	require.NoError(t, app.db.First(&reloaded, "id = ?", receipt.ID).Error)
	require.Equal(t, "Reversed", reloaded.Status)
	require.Equal(t, 0.0, reloaded.UnappliedAmountBHD)
	require.Equal(t, 0.0, reloaded.AppliedAmountBHD) // was never applied — reversal doesn't touch it
	require.Equal(t, 250.0, reloaded.AmountBHD)      // original face amount untouched
}

func TestReverseCustomerReceipt_RejectsAppliedAllocation(t *testing.T) {
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
	require.Equal(t, "Applied", receipt.Status)

	_, err = app.ReverseCustomerReceipt(receipt.ID, "changed my mind")
	require.ErrorContains(t, err, "cannot reverse a receipt with applied allocations")

	// Receipt left unchanged.
	var reloaded CustomerReceipt
	require.NoError(t, app.db.First(&reloaded, "id = ?", receipt.ID).Error)
	require.Equal(t, "Applied", reloaded.Status)
	require.Equal(t, 300.0, reloaded.AppliedAmountBHD)
	require.Equal(t, 0.0, reloaded.UnappliedAmountBHD)
}

func TestReverseCustomerReceipt_RejectsAlreadyReversed(t *testing.T) {
	app := setupTestApp(t)
	receiptTestModels(t, app)

	receipt, err := app.CreateCustomerReceipt(CustomerReceiptInput{
		CustomerID: "cust-1", CustomerName: "Acme Instrumentation",
		AmountBHD: 100.000, PaymentMethod: "Cash", Division: "PH Trading",
	})
	require.NoError(t, err)

	_, err = app.ReverseCustomerReceipt(receipt.ID, "first reversal")
	require.NoError(t, err)

	_, err = app.ReverseCustomerReceipt(receipt.ID, "second reversal")
	require.ErrorContains(t, err, "already reversed")
}
