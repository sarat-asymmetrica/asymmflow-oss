package main

// INTEG campaign — Wave I3 (accounting batch) persistence validation.
// CreateJournalEntry is the double-entry posting path with no prior test
// coverage. This drives the bound App method against a scratch SQLite: a
// balanced entry persists (with its lines + a generated number + computed
// totals), and unbalanced / empty entries are refused.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIntegAccounting_CreateJournalEntry(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&JournalEntry{}, &JournalLine{}), "migrate journal")

	entryDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)

	// --- Balanced entry persists with computed totals + a generated number. ---
	balanced := JournalEntry{
		EntryDate:   entryDate,
		Description: "Synthetic manual voucher",
		Lines: []JournalLine{
			{AccountID: "acc-cash", AccountName: "Cash", Debit: 500.000, Credit: 0},
			{AccountID: "acc-rev", AccountName: "Revenue", Debit: 0, Credit: 500.000},
		},
	}
	created, err := app.CreateJournalEntry(balanced)
	require.NoError(t, err, "a balanced entry must post")
	require.NotNil(t, created)
	require.NotEmpty(t, created.EntryNumber, "an entry number is generated when absent")
	require.InDelta(t, 500.000, created.DebitTotal, 1e-6)
	require.InDelta(t, 500.000, created.CreditTotal, 1e-6)
	require.False(t, created.IsPosted, "a fresh entry is unposted (draft)")
	require.Equal(t, 2026, created.FiscalYear, "fiscal year derived from entry date")

	// Persisted: the entry + both lines are in the DB.
	var stored JournalEntry
	require.NoError(t, app.db.Preload("Lines").Where("id = ?", created.ID).First(&stored).Error)
	require.Len(t, stored.Lines, 2, "both journal lines persist")
	require.InDelta(t, 500.000, stored.DebitTotal, 1e-6)

	// --- Unbalanced entry is refused (debits != credits). ---
	_, err = app.CreateJournalEntry(JournalEntry{
		EntryDate:   entryDate,
		Description: "Unbalanced",
		Lines: []JournalLine{
			{AccountID: "a", Debit: 500.000},
			{AccountID: "b", Credit: 400.000},
		},
	})
	require.Error(t, err, "debits must equal credits")

	// --- Empty entry is refused. ---
	_, err = app.CreateJournalEntry(JournalEntry{EntryDate: entryDate, Description: "empty"})
	require.Error(t, err, "an entry needs at least one line")

	// Exactly one entry persisted (the balanced one); the two refusals wrote nothing.
	var count int64
	require.NoError(t, app.db.Model(&JournalEntry{}).Count(&count).Error)
	require.Equal(t, int64(1), count, "refused entries must not persist")
}
