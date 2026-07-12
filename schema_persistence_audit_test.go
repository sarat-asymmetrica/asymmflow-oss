package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaAudit_NoteAndDivisionPersistenceColumnsAreBackedBySQLite(t *testing.T) {
	srcPath := filepath.Join(".", "ph_holdings.db")
	if !fileExists(srcPath) {
		t.Skip("repo deployment database not present")
	}
	tempPath := copyDeploymentAuditDBToTemp(t, srcPath)
	db := openDeploymentAuditTestDB(t, tempPath)
	app := newDeploymentAuditAppForDB(t, db, tempPath)

	require.NoError(t, app.ensureCriticalDeploymentFoundations())

	checks := []struct {
		table  string
		column string
	}{
		{table: "entity_notes", column: "content"},
		{table: "rfq_data", column: "notes"},
		{table: "rfq_comments", column: "comment"},
		{table: "opportunities", column: "comment"},
		{table: "opportunities", column: "owner_notes"},
		{table: "opportunities", column: "division"},
		{table: "opportunity_comments", column: "comment"},
		{table: "offer_notes", column: "content"},
		{table: "offers", column: "terms_and_conditions"},
		{table: "task_items", column: "description"},
		{table: "task_items", column: "blocked_reason"},
		{table: "task_comments", column: "body"},
		{table: "employees", column: "notes"},
		{table: "suppliers", column: "notes"},
		{table: "expense_entries", column: "notes"},
		{table: "expense_entries", column: "division"},
		{table: "bank_statements", column: "notes"},
		{table: "bank_statements", column: "division"},
		{table: "bank_statement_lines", column: "notes"},
		{table: "book_bank_reconciliations", column: "notes"},
		{table: "employee_compensation_profiles", column: "notes"},
		{table: "employee_compensation_profiles", column: "division"},
		{table: "payroll_periods", column: "notes"},
		{table: "payroll_periods", column: "division"},
		{table: "payroll_runs", column: "notes"},
		{table: "payroll_runs", column: "division"},
	}

	for _, check := range checks {
		require.Truef(
			t,
			db.Migrator().HasColumn(check.table, check.column),
			"expected %s.%s to exist after schema repair audit",
			check.table,
			check.column,
		)
	}
}
