package main

// INTEG campaign — Wave I3 (AR batch) persistence validation for the frontend
// hot-zone mutations wired in this wave.
//
// ReverseCustomerReceipt is already covered end-to-end by receipt_reversal_test.go
// (zero-application succeeds, rejects an applied allocation, rejects an already-
// reversed receipt) — the frontend adapter is a thin pass-through to that binding,
// so no new test is added here for it. ApplyCreditNote had NO existing coverage;
// this validates that the bound App method persists the AR reduction + CN status
// and honors its guards, against a scratch SQLite.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIntegAR_ApplyCreditNote(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CreditNote{}, &CreditNoteItem{}), "migrate credit notes")

	// Always read into a FRESH Invoice — a reused dest whose primary key is still
	// set makes GORM AND that stale id into the WHERE clause (id='x' AND id='y').
	getInvoice := func(id string) Invoice {
		var v Invoice
		require.NoError(t, app.db.Where("id = ?", id).First(&v).Error)
		return v
	}

	mkInvoice := func(id, number string, outstanding float64, status string) {
		require.NoError(t, app.db.Create(&Invoice{
			Base:           Base{ID: id},
			InvoiceNumber:  number,
			CustomerName:   "Synthetic Customer W.L.L.",
			Status:         status,
			OutstandingBHD: outstanding,
		}).Error)
	}
	mkCreditNote := func(id, number, invoiceID string, total float64, status string) {
		require.NoError(t, app.db.Create(&CreditNote{
			Base:             Base{ID: id},
			CreditNoteNumber: number,
			CreditNoteDate:   time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
			InvoiceID:        invoiceID,
			InvoiceNumber:    "linked",
			Status:           status,
			GrandTotalBHD:    total,
		}).Error)
	}

	// --- Partial application: reduces outstanding, invoice stays open. ---
	mkInvoice("inv-1", "INV-T-0001", 1000.000, "Sent")
	mkCreditNote("cn-1", "CN-T-0001", "inv-1", 300.000, "Issued")

	require.NoError(t, app.ApplyCreditNote("cn-1"))

	var cn CreditNote
	require.NoError(t, app.db.Where("id = ?", "cn-1").First(&cn).Error)
	require.Equal(t, "Applied", cn.Status, "CN must be marked Applied")
	require.NotNil(t, cn.AppliedAt, "applied_at timestamp must be set")

	inv := getInvoice("inv-1")
	require.InDelta(t, 700.000, inv.OutstandingBHD, 1e-6, "outstanding reduced by the CN total")
	require.Equal(t, "Sent", inv.Status, "still open — outstanding > 0")

	// --- Full application: outstanding hits 0 → invoice auto-marked Paid. ---
	mkCreditNote("cn-2", "CN-T-0002", "inv-1", 700.000, "Issued")
	require.NoError(t, app.ApplyCreditNote("cn-2"))
	inv = getInvoice("inv-1")
	require.InDelta(t, 0.0, inv.OutstandingBHD, 1e-6)
	require.Equal(t, "Paid", inv.Status, "zero outstanding auto-marks the invoice Paid")

	// --- Guard: re-applying an already-Applied CN is rejected. ---
	require.Error(t, app.ApplyCreditNote("cn-1"), "already-applied CN must be refused")

	// --- Guard: a Draft (not-yet-issued) CN cannot be applied. ---
	mkInvoice("inv-2", "INV-T-0002", 500.000, "Sent")
	mkCreditNote("cn-draft", "CN-T-0003", "inv-2", 100.000, "Draft")
	require.Error(t, app.ApplyCreditNote("cn-draft"), "a Draft CN must be issued before it can be applied")

	// The refused draft must not have touched the invoice.
	require.InDelta(t, 500.000, getInvoice("inv-2").OutstandingBHD, 1e-6, "a refused apply leaves outstanding untouched")
}
