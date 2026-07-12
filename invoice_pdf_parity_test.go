package main

import (
	"os"
	"testing"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/stretchr/testify/require"
)

// PH convergence Row C (PC-D2): the customer invoice PDF renders with the
// attention-block buyer fallback (C-b) and a deliberately long reference
// (C-c) without error at the trimmed 40mm top margin (C-a).
func TestGenerateInvoicePDF_AttentionFallbackAndLongRef(t *testing.T) {
	a := setupTestApp(t)

	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)

	// Customer with NO address — exercises the C-b fallback to the invoice
	// attention block.
	customer := CustomerMaster{BusinessName: "Nimbus Controls", CustomerCode: "NC-01", CustomerID: "NC-01", Status: "Active"}
	require.NoError(t, a.db.Create(&customer).Error)

	invoice := Invoice{
		InvoiceNumber: "INV-26-6001", InvoiceDate: now, DueDate: now.AddDate(0, 0, 30),
		CustomerID: customer.ID, CustomerName: customer.BusinessName,
		Status: "Draft", SubtotalBHD: 3780, VATPercent: 10, VATBHD: 378,
		GrandTotalBHD: 4158, OutstandingBHD: 4158,
		CustomerReference: "RFQ-PR-1220031144-NIMBUS-CPS11D-EXTRA-LONG-REFERENCE",
		AttentionCompany:  "Nimbus Procurement Dept",
		AttentionAddress:  "Building 12, Road 4567, Manama, Kingdom of Bahrain",
	}
	require.NoError(t, a.db.Create(&invoice).Error)
	require.NoError(t, a.db.Create(&DBInvoiceItem{
		InvoiceID: invoice.ID, LineNumber: 1,
		Description: "Conductivity sensor", Quantity: 12, Rate: 315, TotalBHD: 3780,
	}).Error)

	path, err := a.GenerateInvoicePDF(invoice.ID)
	require.NoError(t, err)
	require.NotEmpty(t, path)
	t.Cleanup(func() { _ = os.Remove(path) })
}

// C-c helper: truncation keeps text inside the column at the current font and
// leaves short text untouched.
func TestTruncatePDFTextToWidth(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.AddPage()

	require.Equal(t, "SHORT", truncatePDFTextToWidth(pdf, "SHORT", 60))

	long := "RFQ-PR-1220031144-NIMBUS-CPS11D-EXTRA-LONG-REFERENCE-THAT-OVERFLOWS"
	got := truncatePDFTextToWidth(pdf, long, 40)
	require.NotEqual(t, long, got)
	require.Contains(t, got, "...")
	require.LessOrEqual(t, pdf.GetStringWidth(got), 40.0)
}
