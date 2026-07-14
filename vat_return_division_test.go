package main

// vat_return_division_test.go
//
// Wave 12.5 (owner-sanctioned tax-behavior change): ExportVATReturnData now
// emits ONE CSV per configured division, each stamped with that division's own
// TRN and covering only that division's supplies. Each division is a distinct
// VAT-registered legal entity (distinct TRN) that files its own NBR return —
// filing a Beacon sale under Acme's TRN would mis-report. This test proves the
// per-TRN partitioning: a Beacon invoice's VAT lands ONLY in Beacon's return
// (Beacon TRN), never in Acme's, and vice-versa.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExportVATReturnData_PartitionsPerDivisionTRN(t *testing.T) {
	withSyntheticOverlay(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Invoice{}, &CreditNote{}, &DBInvoiceItem{}, &CustomerMaster{}))

	// One Acme invoice (100/10) and two Beacon invoices (200/20 total) in the
	// current quarter (buildEInvoiceTestInvoice dates them time.Now()).
	buildEInvoiceTestInvoice(t, app, "Acme Instrumentation", "VAT-ACME-001")
	buildEInvoiceTestInvoice(t, app, "Beacon Controls", "VAT-BEACON-001")
	buildEInvoiceTestInvoice(t, app, "Beacon Controls", "VAT-BEACON-002")

	now := time.Now()
	quarter := int((now.Month()-1)/3) + 1

	exportDir, err := app.ExportVATReturnData(now.Year(), quarter)
	require.NoError(t, err, "VAT return export must succeed")
	require.DirExists(t, exportDir, "export must return a directory holding per-division CSVs")

	acmeProfile := companyDocumentProfile("Acme Instrumentation")
	beaconProfile := companyDocumentProfile("Beacon Controls")

	readReturn := func(division string) string {
		profile := companyDocumentProfile(division)
		// Mirror the production filename exactly.
		name := fmt.Sprintf("VAT_Return_Q%d_%d_%s.csv", quarter, now.Year(), sanitizeFilename(profile.Division))
		path := filepath.Join(exportDir, name)
		b, readErr := os.ReadFile(path)
		require.NoError(t, readErr, "per-division VAT return CSV must exist: %s", name)
		return string(b)
	}

	acmeCSV := readReturn("Acme Instrumentation")
	beaconCSV := readReturn("Beacon Controls")

	// --- Each return stamps its OWN TRN, never the other's ---
	require.Contains(t, acmeCSV, acmeProfile.VATNumber, "Acme return must carry Acme's TRN")
	require.NotContains(t, acmeCSV, beaconProfile.VATNumber, "Acme return must NOT carry Beacon's TRN — this is the bug this fix closes")
	require.Contains(t, beaconCSV, beaconProfile.VATNumber, "Beacon return must carry Beacon's TRN")
	require.NotContains(t, beaconCSV, acmeProfile.VATNumber, "Beacon return must NOT carry Acme's TRN")

	// --- Each return reflects only its OWN supplies ---
	// Acme: 1 invoice, 100.000 supply / 10.000 VAT.
	require.Contains(t, acmeCSV, "100.000", "Acme return supply total must be its own 100.000")
	require.Contains(t, acmeCSV, "1 invoices", "Acme return must count only Acme's invoice")
	// Beacon: 2 invoices, 200.000 supply / 20.000 VAT.
	require.Contains(t, beaconCSV, "200.000", "Beacon return supply total must be its own 200.000")
	require.Contains(t, beaconCSV, "2 invoices", "Beacon return must count only Beacon's invoices")

	// The division identity row must name the correct entity.
	require.Contains(t, acmeCSV, acmeProfile.LegalName)
	require.Contains(t, beaconCSV, beaconProfile.LegalName)
	require.True(t, strings.Contains(acmeCSV, "Division"), "CSV should carry a Division identity row")
}
