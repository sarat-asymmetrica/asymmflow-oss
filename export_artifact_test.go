package main

// G4 export tail — artifact-proven tests for the 5 export bindings that lacked
// coverage (the other 5 are covered: ExportVATReturnData by vat_return_division_test,
// ExportCostingToPDF by ahs_branding_smoke_test, both pilot exports by
// phase7_rollout / hybrid_feature_flow tests). Each test QUARANTINES the export
// dir (getExportDir → os.UserHomeDir()/Documents/AsymmFlow Exports) into a temp
// home, drives the real binding, asserts the returned path lands UNDER the
// quarantine, and spot-checks the artifact's content. OpenExportedFile is the one
// true OS side-effect and is deliberately NOT invoked here (G4 law).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// quarantineExports redirects the export root into a throwaway home dir so no
// test ever writes into the real ~/Documents.
func quarantineExports(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home) // Windows os.UserHomeDir
	t.Setenv("HOME", home)        // POSIX fallback
	return home
}

func requireUnderQuarantine(t *testing.T, path, home string) {
	t.Helper()
	require.NotEmpty(t, path, "export must return a real path")
	require.True(t, strings.HasPrefix(path, home), "export path %q must be under the quarantine %q", path, home)
	info, err := os.Stat(path)
	require.NoError(t, err, "the exported file must exist on disk")
	require.Greater(t, info.Size(), int64(0), "the exported file must not be empty")
}

func readExport(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestExportBalanceSheetCSV_Artifact(t *testing.T) {
	home := quarantineExports(t)
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(
		&ChartOfAccount{}, &JournalEntry{}, &JournalLine{}, &Invoice{}, &InventoryItem{},
		&TallyInvoiceImport{}, &TallyPurchaseImport{},
	))

	path, err := app.ExportBalanceSheetCSV(2026)
	require.NoError(t, err)
	requireUnderQuarantine(t, path, home)
	require.True(t, strings.HasSuffix(path, ".csv"))
	content := readExport(t, path)
	require.Contains(t, content, "Balance Sheet", "CSV carries its report header")
	require.Contains(t, content, "Category", "CSV carries its column header row")
}

func TestExportJournalCSV_Artifact(t *testing.T) {
	home := quarantineExports(t)
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&JournalEntry{}, &JournalLine{}, &ChartOfAccount{}))

	path, err := app.ExportJournalCSV(2026)
	require.NoError(t, err)
	requireUnderQuarantine(t, path, home)
	content := readExport(t, path)
	// The header row the CSV writer emits verbatim.
	require.Contains(t, content, "Entry Number", "CSV carries its column header row")
	require.Contains(t, content, "Debit (BHD)")
}

func TestExportGeneralLedgerCSV_Artifact(t *testing.T) {
	home := quarantineExports(t)
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&JournalEntry{}, &JournalLine{}, &ChartOfAccount{}))

	path, err := app.ExportGeneralLedgerCSV(2026)
	require.NoError(t, err)
	requireUnderQuarantine(t, path, home)
	require.True(t, strings.HasSuffix(path, ".csv"))
	require.NotEmpty(t, readExport(t, path))
}

func TestExportCashflowEvidencePack_Artifact(t *testing.T) {
	home := quarantineExports(t)
	app := setupFullTestApp(t)
	require.NoError(t, app.db.AutoMigrate(
		&JournalEntry{}, &JournalLine{},
		&Invoice{}, &Payment{}, &SupplierInvoice{}, &SupplierPayment{},
		&BankStatementLine{},
	))

	path, err := app.ExportCashflowEvidencePack(30)
	require.NoError(t, err)
	requireUnderQuarantine(t, path, home)
	// The pack is a JSON document — spot-check it parses as an object.
	content := strings.TrimSpace(readExport(t, path))
	require.True(t, strings.HasPrefix(content, "{"), "evidence pack is a JSON object")
}

func TestExportCostingToExcel_Artifact(t *testing.T) {
	home := quarantineExports(t)
	app := setupTestApp(t)

	// A minimal well-formed CostingExportData (same shape SaveCostingAsOffer uses).
	path, err := app.ExportCostingToExcel(CostingExportData{
		Division:     "Acme Instrumentation",
		Date:         "2026-07-16",
		PreparedBy:   "A. Yusuf",
		CustomerName: "Gulf Fabrication W.L.L.",
		QuoteType:    "Quotation",
		GrandTotal:   1000.000,
		LineItems: []CostingExportLineItem{
			{SlNo: 1, Equipment: "Flow Meter", Quantity: 2, SuggestedPrice: 300.000, TotalPrice: 600.000, TotalCost: 420.000},
		},
	})
	require.NoError(t, err)
	requireUnderQuarantine(t, path, home)
	// .xlsx is a zip archive — assert the PK magic bytes.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), 2)
	require.Equal(t, "PK", string(data[:2]), "xlsx must be a valid zip (PK magic)")
	require.True(t, strings.HasSuffix(path, ".xlsx"))
	_ = filepath.Base(path)
}
