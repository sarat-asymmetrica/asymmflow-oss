package main

// INTEG residue campaign — Wave R1.3 (expense GL hot-zone) validation.
// PostExpenseEntry posts a real general-ledger JOURNAL ENTRY (postExpenseJournal),
// not just a status flip — the frontend now wires it (owner-ratified: posting
// belongs where users act, behind a GL-naming ConfirmDialog). This drives the
// bound App method against a scratch SQLite: an approved entry posts, flipping
// its status to `posted`, linking a balanced journal entry sourced from it.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIntegExpense_PostExpenseEntry(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(
		&ExpenseEntry{}, &ExpenseCategory{}, &ExpenseApproval{},
		&JournalEntry{}, &JournalLine{}, &ChartOfAccount{},
	), "migrate expense + GL tables")

	// A category with no explicit GL account — the posting path self-provisions
	// the expense + credit accounts via ensureSupportingAccount.
	cat := ExpenseCategory{Name: "R1.3 Synthetic Posting Category", Code: "ZZ99"}
	require.NoError(t, app.db.Create(&cat).Error)

	// An APPROVED, unpaid entry — the only state PostExpenseEntry accepts.
	entry := ExpenseEntry{
		EntryNumber:   "EXP-2026-0001",
		ExpenseDate:   time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
		Description:   "Synthetic office supplies",
		CategoryID:    cat.ID,
		Currency:      "BHD",
		Amount:        100.000,
		VATAmount:     10.000,
		TotalAmount:   110.000,
		Status:        "approved",
		PaymentStatus: "unpaid",
	}
	require.NoError(t, app.db.Create(&entry).Error)

	posted, err := app.PostExpenseEntry(entry.ID)
	require.NoError(t, err, "an approved entry must post")
	require.Equal(t, "posted", posted.Status, "status flips to posted")
	require.NotNil(t, posted.JournalEntryID)
	require.NotEmpty(t, *posted.JournalEntryID, "a journal entry is linked")

	// --- The journal entry persists, sourced from this expense, and balances. ---
	var je JournalEntry
	require.NoError(t, app.db.Where("source_type = ? AND source_id = ?", "expense_entry", entry.ID).First(&je).Error)
	require.True(t, je.IsPosted, "the GL entry is posted")
	require.InDelta(t, 110.000, je.DebitTotal, 1e-6)
	require.InDelta(t, 110.000, je.CreditTotal, 1e-6, "debits == credits")

	var lineCount int64
	require.NoError(t, app.db.Model(&JournalLine{}).Where("entry_id = ?", je.ID).Count(&lineCount).Error)
	require.Equal(t, int64(2), lineCount, "a debit + a credit line")

	// --- Persisted entry reflects the post. ---
	var storedEntry ExpenseEntry
	require.NoError(t, app.db.Where("id = ?", entry.ID).First(&storedEntry).Error)
	require.Equal(t, "posted", storedEntry.Status)

	// --- Posting is idempotent: a second post returns the same journal, no dup. ---
	_, err = app.PostExpenseEntry(entry.ID)
	require.NoError(t, err, "re-posting a posted entry is a no-op, not an error")
	var jeCount int64
	require.NoError(t, app.db.Model(&JournalEntry{}).Where("source_id = ?", entry.ID).Count(&jeCount).Error)
	require.Equal(t, int64(1), jeCount, "no duplicate journal entry on re-post")
}
