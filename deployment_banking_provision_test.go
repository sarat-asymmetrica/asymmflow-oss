package main

import (
	"fmt"
	"testing"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestCriticalDeploymentProvisionsBankingSuite pins the Mission G fresh-provision
// fix: the bank-reconciliation + FX + VAT-return models are compiled and wired
// into live services but were in no boot migration set, so a from-zero DB never
// created their tables. They now ride criticalDeploymentModels() (unconditional,
// so mature DBs are repaired too). This verifies every one is created.
func TestCriticalDeploymentProvisionsBankingSuite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	for _, m := range criticalDeploymentModels() {
		require.NoError(t, db.AutoMigrate(m), "AutoMigrate %T", m)
	}

	// The 15 tables that were previously unmigrated on a fresh DB.
	wantTables := []string{
		"bank_accounts", "bank_statements", "bank_statement_lines", "bank_statement_files",
		"statement_hashes", "book_bank_reconciliations", "deposits_in_transit", "cheque_registers",
		"outstanding_cheques", "bank_reconciliation_audit_logs", "bank_cash_balances",
		"bank_expense_entries", "fx_rates", "fx_revaluations", "vat_returns",
		// Mission H additions (same gap class, found by the full-surface import).
		"fiscal_periods", "customer_name_mappings",
	}
	for _, tbl := range wantTables {
		require.Truef(t, db.Migrator().HasTable(tbl), "fresh provision must create %q", tbl)
	}
}
