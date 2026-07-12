package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// EXPENSE SEGREGATION-OF-DUTIES GUARD TESTS — B3 (Wave 9 Spec-07)
// =============================================================================
// Mirrors the supplier-invoice SoD precedent (ApproveSupplierInvoice,
// supplier_invoice_service.go:406-411 / supplier_ap_gate_test.go): an expense
// creator must not be able to approve their own expense entry. The guard in
// transitionExpenseEntry (expense_service.go) fires ONLY on the approve
// transition — submit and reject must remain unaffected.
// =============================================================================

// makeExpenseSoDEntry inserts an ExpenseEntry directly (bypassing
// CreateExpenseEntry, which would stamp CreatedBy from the acting identity)
// so the test controls CreatedBy independently of the resolved actor.
func makeExpenseSoDEntry(t *testing.T, app *App, categoryID, createdBy, status string) *ExpenseEntry {
	t.Helper()
	entry := &ExpenseEntry{
		Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now(), CreatedBy: createdBy},
		EntryNumber: "EXP-SOD-" + uuid.New().String()[:8],
		Description: "SoD guard probe expense",
		CategoryID:  categoryID,
		Currency:    "BHD",
		Amount:      100,
		VATAmount:   10,
		TotalAmount: 110,
		ExpenseDate: time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC),
		Status:      status,
	}
	require.NoError(t, app.db.Create(entry).Error)
	return entry
}

// resolvedTestActor returns the identity transitionExpenseEntry will resolve
// as the acting user under setupTestApp's default fixture (no
// EmployeeAccessLink / activated LicenseKey wired up), so tests can set
// CreatedBy to match or differ from it deliberately.
func resolvedTestActor(t *testing.T, app *App) string {
	t.Helper()
	actor := getExpenseActorID(app)
	require.NotEmpty(t, actor)
	return actor
}

func TestApproveExpense_SelfApprovalRejected(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	category, err := app.CreateExpenseCategory(ExpenseCategory{
		Name: "SoD Self-Approval Category",
		Code: "SOD_SELF_APPROVE",
	})
	require.NoError(t, err)

	actor := resolvedTestActor(t, app)
	entry := makeExpenseSoDEntry(t, app, category.ID, actor, "submitted")

	_, err = app.ApproveExpenseEntry(entry.ID, "trying to approve my own expense")
	require.Error(t, err)
	require.Contains(t, err.Error(), "segregation of duties")
	require.Contains(t, err.Error(), "cannot approve their own expense")

	// Status must not have moved.
	var reloaded ExpenseEntry
	require.NoError(t, app.db.First(&reloaded, "id = ?", entry.ID).Error)
	require.Equal(t, "submitted", reloaded.Status)
}

func TestApproveExpense_DistinctApproverSucceeds(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	category, err := app.CreateExpenseCategory(ExpenseCategory{
		Name: "SoD Distinct Approver Category",
		Code: "SOD_DISTINCT_APPROVE",
	})
	require.NoError(t, err)

	actor := resolvedTestActor(t, app)
	require.NotEqual(t, "other-creator-user", actor)
	entry := makeExpenseSoDEntry(t, app, category.ID, "other-creator-user", "submitted")

	approved, err := app.ApproveExpenseEntry(entry.ID, "approved by a distinct reviewer")
	require.NoError(t, err)
	require.Equal(t, "approved", approved.Status)

	var reloaded ExpenseEntry
	require.NoError(t, app.db.First(&reloaded, "id = ?", entry.ID).Error)
	require.Equal(t, "approved", reloaded.Status)
	require.Equal(t, actor, reloaded.ApprovedBy)
}

func TestApproveExpense_SelfCreatedEntryCanStillBeSubmittedAndRejected(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	category, err := app.CreateExpenseCategory(ExpenseCategory{
		Name: "SoD Submit Reject Category",
		Code: "SOD_SUBMIT_REJECT",
	})
	require.NoError(t, err)

	actor := resolvedTestActor(t, app)

	// Submit: a self-created draft must still be submittable — the guard is
	// keyed strictly on the approve transition.
	submitEntry := makeExpenseSoDEntry(t, app, category.ID, actor, "draft")
	submitted, err := app.SubmitExpenseEntry(submitEntry.ID)
	require.NoError(t, err, "self-created expense must still be submittable — SoD guard must not block submit")
	require.Equal(t, "submitted", submitted.Status)

	// Reject: a self-created submitted entry must still be rejectable by
	// anyone (including the creator) — the guard must not block reject.
	rejectEntry := makeExpenseSoDEntry(t, app, category.ID, actor, "submitted")
	rejected, err := app.RejectExpenseEntry(rejectEntry.ID, "missing receipt")
	require.NoError(t, err, "self-created expense must still be rejectable — SoD guard must not block reject")
	require.Equal(t, "rejected", rejected.Status)
}
