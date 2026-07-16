package main

// G3 known-technique wiring — Go oracles for the three mutations that needed a
// backing test (notifications review + cascade delete reuse the existing R2 /
// INTEG tests). Synthetic canon only.

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// UpdateSettings does a FULL settings.json overwrite, so the kernel bridge does
// fetch-merge-write. This proves the pattern's guarantee: overlaying a few fields
// onto the full GetSettings object and writing it back leaves unrelated keys intact.
func TestUpdateSettings_FetchMergeWritePreservesUnrelatedKeys(t *testing.T) {
	app := setupFullTestApp(t)
	app.config.Database.Path = filepath.Join(t.TempDir(), "test.db")

	// Seed a rich settings.json (non-default language/theme so a lost merge is visible).
	require.NoError(t, app.UpdateSettings(map[string]any{
		"companyName": "Old Co",
		"currency":    "BHD",
		"language":    "ar",
		"theme":       "dark",
		"business":    map[string]any{"default_margin": 20.0, "vat_rate": 10.0},
		"folders":     map[string]any{"customers_path": "C:/customers"},
	}))

	// Fetch-merge-write (the exact bridge realUpdate pattern): overlay only the
	// screen's fields onto the FULL GetSettings object.
	full, err := app.GetSettings()
	require.NoError(t, err)
	full["companyName"] = "New Co"
	full["currency"] = "USD"
	biz := full["business"].(map[string]any)
	biz["default_margin"] = 25.0
	biz["vat_rate"] = 5.0
	require.NoError(t, app.UpdateSettings(full))

	got, err := app.GetSettings()
	require.NoError(t, err)
	// The 5 owned fields changed.
	require.Equal(t, "New Co", got["companyName"])
	require.Equal(t, "USD", got["currency"])
	gotBiz := got["business"].(map[string]any)
	require.EqualValues(t, 25.0, gotBiz["default_margin"])
	require.EqualValues(t, 5.0, gotBiz["vat_rate"])
	// Unrelated keys survived (a narrow write would have reverted these to defaults).
	require.Equal(t, "ar", got["language"], "unrelated language key must survive")
	require.Equal(t, "dark", got["theme"], "unrelated theme key must survive")
	gotFolders := got["folders"].(map[string]any)
	require.Equal(t, "C:/customers", gotFolders["customers_path"], "folder paths survive")
}

// Customer status change goes through UpdateCustomer with the FULL record
// (fetch-merge-write): UpdateCustomer applies every editable field including
// blanks, so a sparse {id,status} would wipe name/city/email. The full round-trip
// changes only the status and leaves every other field intact.
func TestSetCustomerStatus_FullRecordPreservesOtherFields(t *testing.T) {
	app := setupTestApp(t)

	require.NoError(t, app.db.Create(&CustomerMaster{
		Base:           Base{ID: "cust-x"},
		BusinessName:   "Alpha Trading W.L.L.",
		City:           "Manama",
		PrimaryEmail:   "ops@alpha.example",
		Status:         "Active",
		CreditLimitBHD: 50000,
	}).Error)

	// Bridge pattern: fetch full record, override ONLY status, write it back.
	existing, err := app.GetCustomer("cust-x")
	require.NoError(t, err)
	existing.Status = "On Hold"
	_, err = app.UpdateCustomer(existing)
	require.NoError(t, err)

	var row CustomerMaster
	require.NoError(t, app.db.First(&row, "id = ?", "cust-x").Error)
	require.Equal(t, "On Hold", row.Status)
	require.Equal(t, "Alpha Trading W.L.L.", row.BusinessName)
	require.Equal(t, "Manama", row.City)
	require.Equal(t, "ops@alpha.example", row.PrimaryEmail)
	require.Equal(t, 50000.0, row.CreditLimitBHD)
}

// SyncCashflowEvidenceProposalReviews reconciles the review worklist — it is a
// review-row SYNC, NOT a GL posting. This pins that invariant (no journals) plus
// idempotency (a second sync over the same window does not duplicate or error).
func TestSyncCashflowEvidenceProposalReviews_SyncsNeverPosts(t *testing.T) {
	app := setupFullTestApp(t)
	// The command center's posting-coverage report + evidence gauges read these
	// finance tables; migrate them so the sync runs against an empty-but-valid DB.
	require.NoError(t, app.db.AutoMigrate(
		&JournalEntry{}, &JournalLine{},
		&Invoice{}, &Payment{}, &SupplierInvoice{}, &SupplierPayment{},
		&BankStatementLine{},
	))

	rows, err := app.SyncCashflowEvidenceProposalReviews(30)
	require.NoError(t, err)
	require.NotNil(t, rows)

	var journalCount int64
	require.NoError(t, app.db.Model(&JournalEntry{}).Count(&journalCount).Error)
	require.Equal(t, int64(0), journalCount, "a review sync must never post a journal entry")

	rows2, err := app.SyncCashflowEvidenceProposalReviews(30)
	require.NoError(t, err)
	require.Len(t, rows2, len(rows), "sync is idempotent over the same window")
}
