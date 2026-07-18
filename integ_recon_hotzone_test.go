package main

// INTEG campaign — Wave I3 (recon batch) persistence validation.
// FinalizeReconciliation + DeleteBankStatement had no direct coverage (the
// surrounding split/unmatch/reopen machinery IS covered by
// bank_reconciliation_service_test.go). These drive the bound App methods
// against a scratch SQLite: a balanced, fully-matched statement finalizes to
// Reconciled (and refuses a second finalize), and a statement can be deleted.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestIntegRecon_FinalizeAndDeleteBankStatement(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(
		&BankStatement{}, &BankStatementLine{}, &BankLinePaymentAllocation{}, &BankReconciliationAuditLog{},
	), "migrate bank recon tables")

	// A balanced statement (opening == closing, no lines → 0 discrepancy,
	// 0 unmatched) is finalizable.
	require.NoError(t, app.db.Create(&BankStatement{
		Base:            Base{ID: "stmt-1"},
		StatementNumber: "STMT-T-0001",
		BankAccountID:   "acc-1",
		OpeningBalance:  1000.000,
		ClosingBalance:  1000.000,
		Currency:        "BHD",
		Status:          "Imported",
		PeriodStart:     time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:       time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
	}).Error)

	require.NoError(t, app.FinalizeReconciliation("stmt-1", "admin"))

	var stmt BankStatement
	require.NoError(t, app.db.Where("id = ?", "stmt-1").First(&stmt).Error)
	require.Equal(t, "Reconciled", stmt.Status, "a balanced, fully-matched statement finalizes")

	// Guard: a Reconciled statement refuses a second finalize.
	require.Error(t, app.FinalizeReconciliation("stmt-1", "admin"), "already-reconciled must be refused")

	// --- DeleteBankStatement removes a statement. ---
	require.NoError(t, app.db.Create(&BankStatement{
		Base:            Base{ID: "stmt-2"},
		StatementNumber: "STMT-T-0002",
		BankAccountID:   "acc-1",
		OpeningBalance:  0,
		ClosingBalance:  0,
		Currency:        "BHD",
		Status:          "Imported",
	}).Error)
	require.NoError(t, app.DeleteBankStatement("stmt-2"))

	var gone BankStatement
	require.ErrorIs(t, app.db.Where("id = ?", "stmt-2").First(&gone).Error, gorm.ErrRecordNotFound,
		"the statement must be gone after delete")
}
