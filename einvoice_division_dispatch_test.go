package main

// einvoice_division_dispatch_test.go
//
// Tests that GenerateEInvoiceXML dispatches the correct supplier TRN, legal
// name, and address based on Invoice.Division — the canonical Mission A
// e-invoice bug fix.
//
// Bug (before fix): ALL invoices emitted Acme Instrumentation's TRN and name
// via hardcoded phTradingTRN/phTradingName constants, regardless of the
// invoice's Division field.
//
// Fix: supplier identity is now resolved through companyDocumentProfile(
// invoice.Division) which reads from activeOverlay, so a Beacon Controls
// invoice correctly emits Beacon's TRN and a Acme invoice emits Acme's TRN.

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// buildEInvoiceTestInvoice creates a minimal invoice record in the test DB
// and returns the ID. The caller controls Division.
func buildEInvoiceTestInvoice(t *testing.T, app *App, division, invoiceNo string) string {
	t.Helper()

	customer := CustomerMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now(), CreatedBy: "test"},
		BusinessName: "Test Customer Corp",
		CustomerCode: "EINV-CUST-" + invoiceNo,
		CustomerID:   "EINV-CUST-" + invoiceNo,
		AddressLine1: "Test Street, Test City",
		Country:      "BH",
		Status:       "Active",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	now := time.Now()
	inv := Invoice{
		Base:          Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test"},
		InvoiceNumber: invoiceNo,
		InvoiceDate:   now,
		DueDate:       now.AddDate(0, 0, 30),
		CustomerID:    customer.ID,
		CustomerName:  customer.BusinessName,
		SubtotalBHD:   100.000,
		VATBHD:        10.000,
		VATPercent:    10.0,
		GrandTotalBHD: 110.000,
		Status:        "Sent",
		Division:      division,
	}
	require.NoError(t, app.db.Create(&inv).Error)

	item := DBInvoiceItem{
		Base:        Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test"},
		InvoiceID:   inv.ID,
		LineNumber:  1,
		Description: "Test Item",
		Quantity:    1,
		Rate:        100,
		TotalBHD:    100,
	}
	require.NoError(t, app.db.Create(&item).Error)

	return inv.ID
}

// TestEInvoiceXMLDispatchesSupplierByDivision is the canonical regression test
// for the per-division TRN dispatch bug.
//
//   - An Acme Instrumentation division invoice must include Acme's TRN
//     (990000000000000) and legal name (ACME INSTRUMENTATION W.L.L).
//   - A Beacon Controls division invoice must include Beacon's TRN
//     (990000000000001) and legal name (BEACON CONTROLS W.L.L.).
//   - Crucially, the Beacon invoice must NOT contain Acme's TRN — this is
//     the exact symptom of the original bug.
func TestEInvoiceXMLDispatchesSupplierByDivision(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	app := setupTestApp(t)

	// Resolve expected values from the same overlay that the production code
	// uses — this keeps the test robust to future overlay changes.
	acmeProfile := companyDocumentProfile("Acme Instrumentation")
	beaconProfile := companyDocumentProfile("Beacon Controls")

	// --- Acme invoice ---
	acmeID := buildEInvoiceTestInvoice(t, app, "Acme Instrumentation", "EINV-ACME-001")
	acmePath, err := app.GenerateEInvoiceXML(acmeID)
	require.NoError(t, err, "Acme e-invoice generation must succeed")
	t.Cleanup(func() { _ = os.Remove(acmePath) })

	acmeXML, err := os.ReadFile(acmePath)
	require.NoError(t, err)
	acmeXMLStr := string(acmeXML)

	require.Contains(t, acmeXMLStr, acmeProfile.VATNumber,
		"Acme invoice XML must contain Acme's TRN (%s)", acmeProfile.VATNumber)
	require.Contains(t, acmeXMLStr, acmeProfile.LegalName,
		"Acme invoice XML must contain Acme's legal name")

	// Acme invoice must NOT contain Beacon's TRN (sanity check)
	require.NotContains(t, acmeXMLStr, beaconProfile.VATNumber,
		"Acme invoice XML must NOT contain Beacon's TRN — division dispatch must be clean")

	// --- Beacon Controls invoice ---
	beaconID := buildEInvoiceTestInvoice(t, app, "Beacon Controls", "EINV-BEACON-001")
	beaconPath, err := app.GenerateEInvoiceXML(beaconID)
	require.NoError(t, err, "Beacon e-invoice generation must succeed")
	t.Cleanup(func() { _ = os.Remove(beaconPath) })

	beaconXML, err := os.ReadFile(beaconPath)
	require.NoError(t, err)
	beaconXMLStr := string(beaconXML)

	// This is THE regression assertion — before the fix, a Beacon Controls
	// invoice emitted Acme's TRN (990000000000000) here instead of Beacon's.
	require.Contains(t, beaconXMLStr, beaconProfile.VATNumber,
		"Beacon invoice XML must contain Beacon's TRN (%s) — this was the bug", beaconProfile.VATNumber)
	require.Contains(t, beaconXMLStr, beaconProfile.LegalName,
		"Beacon invoice XML must contain Beacon's legal name")

	// Beacon invoice must NOT leak Acme's TRN
	require.NotContains(t, beaconXMLStr, acmeProfile.VATNumber,
		"Beacon invoice XML must NOT contain Acme's TRN (was the bug before fix)")

	// Both XMLs must be well-formed UBL 2.1 documents
	require.True(t, strings.HasPrefix(strings.TrimSpace(acmeXMLStr), `<?xml`),
		"Acme XML must start with XML declaration")
	require.True(t, strings.HasPrefix(strings.TrimSpace(beaconXMLStr), `<?xml`),
		"Beacon XML must start with XML declaration")
	require.Contains(t, acmeXMLStr, "<cac:AccountingSupplierParty>",
		"Acme XML must contain supplier party element")
	require.Contains(t, beaconXMLStr, "<cac:AccountingSupplierParty>",
		"Beacon XML must contain supplier party element")
}

// TestEInvoiceXMLSupplierAddressFromOverlay verifies that the supplier address
// lines in the generated XML come from the overlay (not from hardcoded strings
// that existed before the fix).
func TestEInvoiceXMLSupplierAddressFromOverlay(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	app := setupTestApp(t)

	beaconProfile := companyDocumentProfile("Beacon Controls")
	require.NotEmpty(t, beaconProfile.AddressLines, "Beacon overlay must have address lines")

	beaconID := buildEInvoiceTestInvoice(t, app, "Beacon Controls", "EINV-BEACON-ADDR-001")
	beaconPath, err := app.GenerateEInvoiceXML(beaconID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(beaconPath) })

	beaconXML, err := os.ReadFile(beaconPath)
	require.NoError(t, err)
	beaconXMLStr := string(beaconXML)

	// The first address line from the Beacon overlay must appear in the XML
	require.Contains(t, beaconXMLStr, beaconProfile.AddressLines[0],
		"Beacon invoice XML must contain Beacon's first address line from the overlay")

	// Acme's old hardcoded address must NOT appear in a Beacon invoice
	acmeProfile := companyDocumentProfile("Acme Instrumentation")
	if len(acmeProfile.AddressLines) > 0 {
		// Only check if the Acme address is actually different from Beacon's
		if acmeProfile.AddressLines[0] != beaconProfile.AddressLines[0] {
			require.NotContains(t, beaconXMLStr, acmeProfile.AddressLines[0],
				"Beacon invoice XML must NOT contain Acme's address line")
		}
	}
}
