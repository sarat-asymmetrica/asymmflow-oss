package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Wave 9.7 tight-ship-2 VAT parsing-bug fix: `if vatPercent == 0 { vatPercent
// = 10 }` (and the equivalent frontend `Number(x) || 10`) silently overwrote
// a genuine 0% (zero-rated/export) invoice with the 10% default, because 0
// is falsy in both Go's `== 0` short-circuit and JS's `||`. These tests
// cover the derived-document Go paths that used to re-coerce a posted
// invoice's explicit 0% VAT rate back up to 10%: credit notes (unit-tested
// end-to-end here) and the e-invoice XML rate (unit-tested via the
// generated XML). The PDF path (invoice_pdf_service.go) shares the exact
// same one-line fix and is covered by manual reasoning: it now reads
// invoice.VATPercent directly with no coercion, same as e-invoice.

func seedZeroRatedInvoice(t *testing.T, a *App, number string) Invoice {
	t.Helper()
	inv := Invoice{
		InvoiceNumber: number, Status: "Sent",
		InvoiceDate:  time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		CustomerID:   "cust-zero-rated",
		CustomerName: "Export Customer",
		VATPercent:   0, SubtotalBHD: 200, VATBHD: 0, GrandTotalBHD: 200, OutstandingBHD: 200,
	}
	require.NoError(t, a.db.Create(&inv).Error)
	return inv
}

// TestCreateCreditNote_PreservesZeroRatedVAT: a credit note issued against a
// genuinely zero-rated invoice must itself carry VATPercent=0 / VATBHD=0,
// not the 10% default (credit_note_service.go ~line 97).
func TestCreateCreditNote_PreservesZeroRatedVAT(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&CreditNote{}, &CreditNoteItem{}))

	inv := seedZeroRatedInvoice(t, a, "INV-26-9001")

	cn, err := a.CreateCreditNote(inv.ID, "Partial return of export shipment", []CreditNoteItemInput{
		{Description: "Returned unit", Quantity: 2, Rate: 50},
	})
	require.NoError(t, err)
	require.Equal(t, 100.0, cn.SubtotalBHD)
	require.Equal(t, 0.0, cn.VATPercent, "credit note must mirror the invoice's zero VAT rate")
	require.Equal(t, 0.0, cn.VATBHD, "zero-rated invoice's credit note must not gain 10% VAT")
	require.Equal(t, 100.0, cn.GrandTotalBHD)
}

// TestCreateCreditNote_NonZeroVATUnaffected: a normal (non-zero) VAT invoice
// still produces a credit note at the invoice's actual rate — this fix only
// changes the treatment of an explicit 0, nothing else.
func TestCreateCreditNote_NonZeroVATUnaffected(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&CreditNote{}, &CreditNoteItem{}))

	inv := Invoice{
		InvoiceNumber: "INV-26-9002", Status: "Sent",
		InvoiceDate:  time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		CustomerID:   "cust-standard",
		CustomerName: "Standard Customer",
		VATPercent:   10, SubtotalBHD: 200, VATBHD: 20, GrandTotalBHD: 220, OutstandingBHD: 220,
	}
	require.NoError(t, a.db.Create(&inv).Error)

	cn, err := a.CreateCreditNote(inv.ID, "Pricing correction", []CreditNoteItemInput{
		{Description: "Adjustment", Quantity: 1, Rate: 100},
	})
	require.NoError(t, err)
	require.Equal(t, 10.0, cn.VATPercent)
	require.Equal(t, 10.0, cn.VATBHD)
	require.Equal(t, 110.0, cn.GrandTotalBHD)
}

// TestCreateCreditNote_ReconstructsLegacyVATRate: a legacy invoice that stored
// a real VATBHD but no VATPercent (column default 0) must NOT be treated as
// zero-rated — the derived credit note reconstructs the applied rate from
// VATBHD/SubtotalBHD (here 20/200 = 10%), so removing the old blanket ||10
// coercion never under-reports VAT on real historical invoices.
func TestCreateCreditNote_ReconstructsLegacyVATRate(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&CreditNote{}, &CreditNoteItem{}))

	inv := Invoice{
		InvoiceNumber: "INV-26-9004", Status: "Sent",
		InvoiceDate:  time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		CustomerID:   "cust-legacy",
		CustomerName: "Legacy Customer",
		// Legacy shape: VATBHD present, VATPercent never populated.
		VATPercent: 0, SubtotalBHD: 200, VATBHD: 20, GrandTotalBHD: 220, OutstandingBHD: 220,
	}
	require.NoError(t, a.db.Create(&inv).Error)

	cn, err := a.CreateCreditNote(inv.ID, "Return against legacy invoice", []CreditNoteItemInput{
		{Description: "Returned unit", Quantity: 1, Rate: 100},
	})
	require.NoError(t, err)
	require.Equal(t, 10.0, cn.VATPercent, "legacy VATBHD-without-rate invoice must reconstruct to 10%%, not 0%%")
	require.Equal(t, 10.0, cn.VATBHD)
	require.Equal(t, 110.0, cn.GrandTotalBHD)
}

// TestGenerateEInvoiceXML_PreservesZeroRatedVAT: the ZATCA UBL XML must
// report the invoice's true (0%) rate, not the 10% default
// (einvoice_service.go ~line 76).
func TestGenerateEInvoiceXML_PreservesZeroRatedVAT(t *testing.T) {
	a := setupTestApp(t)

	inv := seedZeroRatedInvoice(t, a, "INV-26-9003")

	xmlPath, err := a.GenerateEInvoiceXML(inv.ID)
	require.NoError(t, err)
	xmlBytes, err := os.ReadFile(xmlPath)
	require.NoError(t, err)
	xml := string(xmlBytes)
	require.Contains(t, xml, "<cbc:Percent>0.0</cbc:Percent>", "e-invoice XML must report the true 0%% VAT rate, not the 10%% default")
	require.False(t, strings.Contains(xml, "<cbc:Percent>10.0</cbc:Percent>"), "zero-rated invoice must not be reported to ZATCA as 10%%")
}
