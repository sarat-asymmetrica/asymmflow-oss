package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

const legacyOfferShellAmountTolerance = 0.000001

const deploymentActiveInvoiceStatusWhere = "status NOT IN ('Cancelled', 'Void', 'Proforma', 'Draft')"

type DeploymentDataAudit struct {
	GeneratedAt                    string   `json:"generated_at"`
	DatabasePath                   string   `json:"database_path"`
	ExpectedRuntimeDatabasePath    string   `json:"expected_runtime_database_path"`
	PackagedDatabasePath           string   `json:"packaged_database_path"`
	UsingRuntimeAppData            bool     `json:"using_runtime_app_data"`
	RuntimeDatabaseExists          bool     `json:"runtime_database_exists"`
	PackagedDatabaseExists         bool     `json:"packaged_database_exists"`
	MissingTables                  []string `json:"missing_tables"`
	BlockingDataIssues             []string `json:"blocking_data_issues"`
	WarningDataIssues              []string `json:"warning_data_issues"`
	Blocking                       bool     `json:"blocking"`
	ActiveCustomers                int      `json:"active_customers"`
	ActiveInvoices                 int      `json:"active_invoices"`
	ActiveInvoicesWithoutItems     int      `json:"active_invoices_without_items"`
	ActiveOrders                   int      `json:"active_orders"`
	ActiveOrdersWithoutItems       int      `json:"active_orders_without_items"`
	ActiveZeroTotalOrders          int      `json:"active_zero_total_orders"`
	ActiveOffers                   int      `json:"active_offers"`
	ActiveOperationalOffers        int      `json:"active_operational_offers"`
	ActiveOperationalOffersNoItems int      `json:"active_operational_offers_without_items"`
	WonOffersWithoutItems          int      `json:"won_offers_without_items"`
	LegacyQuotedOfferShells        int      `json:"legacy_quoted_offer_shells"`
	LegacyRFQOfferShells           int      `json:"legacy_rfq_offer_shells"`
	Employees                      int      `json:"employees"`
	Notifications                  int      `json:"notifications"`
	TaskItems                      int      `json:"task_items"`
	ExpenseEntries                 int      `json:"expense_entries"`
	PayrollRuns                    int      `json:"payroll_runs"`
	LegacyFollowUpTasks            int      `json:"legacy_followup_tasks"`
	MigratedLegacyTasks            int      `json:"migrated_legacy_tasks"`
}

var criticalDeploymentTableNames = []string{
	"employees",
	"employee_access_links",
	"projects",
	"project_members",
	"notifications",
	"notification_receipts",
	"task_items",
	"task_comments",
	"task_activity",
	"collaborative_pending_operations",
	"expense_categories",
	"expense_vendors",
	"expense_entries",
	"expense_allocations",
	"recurring_expenses",
	"expense_attachments",
	"expense_approvals",
	"employee_compensation_profiles",
	"payroll_periods",
	"payroll_runs",
	"payroll_run_items",
	"payroll_components",
	"payroll_payouts",
	"user_activity_sessions",
	"user_activity_events",
	"user_activity_weekly_summaries",
	"opportunity_edit_conflicts",
	"chart_of_accounts",
	"account_mappings",
	// Bank-reconciliation + FX + VAT-return suite (Mission G fresh-provision fix).
	"bank_accounts",
	"bank_statements",
	"bank_statement_lines",
	"bank_statement_files",
	"statement_hashes",
	"book_bank_reconciliations",
	"deposits_in_transit",
	"cheque_registers",
	"outstanding_cheques",
	"bank_reconciliation_audit_logs",
	"bank_cash_balances",
	"bank_expense_entries",
	"fx_rates",
	"fx_revaluations",
	"vat_returns",
	// Mission H fresh-provision fix: read by live services (finance_reporting
	// period locks, customer_linkage dashboard) but in no boot migration set.
	"fiscal_periods",
	"customer_name_mappings",
}

func criticalDeploymentModels() []any {
	return []any{
		&LicenseKey{},
		&CompanyBankAccount{},
		&Setting{},
		&Employee{},
		&EmployeeDocument{}, // Wave 9.8 B4
		&EmployeeAccessLink{},
		&Project{},
		&ProjectMember{},
		&Notification{},
		&NotificationReceipt{},
		&TaskItem{},
		&TaskComment{},
		&TaskActivity{},
		&CollaborativePendingOperation{},
		&ExpenseCategory{},
		&ExpenseVendor{},
		&ExpenseEntry{},
		&ExpenseAllocation{},
		&RecurringExpense{},
		&ExpenseAttachment{},
		&ExpenseApproval{},
		&JournalEntry{},
		&JournalLine{},
		&EmployeeCompensationProfile{},
		&PayrollPeriod{},
		&PayrollRun{},
		&PayrollRunItem{},
		&PayrollComponent{},
		&PayrollPayout{},
		&UserActivitySession{},
		&UserActivityEvent{},
		&UserActivityWeeklySummary{},
		&OpportunityEditConflict{},
		// Posting spine: the expense foundation ensures accounts (6100â€¦) and
		// invoice/payment posting resolves account mappings â€” but nothing else
		// migrates these on a FRESH database file (mature DBs always had them,
		// so the gap was invisible until Mission E provisioned from scratch).
		&ChartOfAccount{},
		&AccountMapping{},
		// Bank-reconciliation + FX + VAT-return suite: these models are compiled
		// and actively read/written by live services (bank_integrity_service,
		// book_bank_reconciliation_service, cheque_register_service,
		// cashflow_evidence_service, ExportVATReturnData) but were in NO boot
		// migration set â€” so a from-zero OSS DB never created the tables and the
		// entire Finance-Hub reconciliation surface silent-no-op'd on a fresh
		// install (Mission G parity gap; models exist, only registration missing).
		// Registered here (unconditional) so mature DBs predating the module also
		// gain the tables. Data that populates them is Mission-H (data) deferred.
		&BankAccount{},
		&BankStatement{},
		&BankStatementLine{},
		&BankStatementFile{},
		&StatementHash{},
		&BookBankReconciliation{},
		&DepositInTransit{},
		&ChequeRegister{},
		&OutstandingCheque{},
		&BankReconciliationAuditLog{},
		&BankCashBalance{},
		&BankExpenseEntry{},
		&FXRate{},
		&FXRevaluation{},
		&VATReturn{},
		// Mission H: same gap class as the banking suite â€” models compiled and
		// read by live services (fiscal-period close locks in
		// finance_reporting_service, customer-linkage dashboard) but absent
		// from every boot migration set, so fresh installs (and the PH import
		// destination) never had the tables.
		&FiscalPeriod{},
		&CustomerNameMapping{},
	}
}

func (a *App) ensureCriticalDeploymentFoundations() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var failures []string
	for _, model := range criticalDeploymentModels() {
		if shouldSkipCriticalAutoMigrate(a.db, model) {
			continue
		}
		if err := a.db.AutoMigrate(model); err != nil {
			failures = append(failures, fmt.Sprintf("%T migration: %v", model, err))
		}
	}
	if err := a.ensureCollaborativeFoundationInternal(); err != nil {
		failures = append(failures, fmt.Sprintf("collaborative foundation: %v", err))
	}
	if err := a.EnsureExpenseFoundation(); err != nil {
		failures = append(failures, fmt.Sprintf("expense foundation: %v", err))
	}
	if err := a.ensurePayrollFoundationInternal(); err != nil {
		failures = append(failures, fmt.Sprintf("payroll foundation: %v", err))
	}
	if err := a.ensureUserActivityMonitoringFoundationInternal(); err != nil {
		failures = append(failures, fmt.Sprintf("activity monitoring foundation: %v", err))
	}
	if err := a.ensureOpportunityConflictFoundationInternal(); err != nil {
		failures = append(failures, fmt.Sprintf("opportunity conflict foundation: %v", err))
	}
	a.ensureCrossModuleSchemaExtensions()
	a.migrateInvoiceCheckConstraint()
	if err := a.ensurePhase7RolloutInternal(); err != nil {
		failures = append(failures, fmt.Sprintf("phase 7 rollout: %v", err))
	}
	if len(failures) > 0 {
		return fmt.Errorf("critical deployment foundations failed: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (a *App) shouldBlockStartupOnDeploymentAudit() bool {
	if strings.TrimSpace(os.Getenv("ASYMMFLOW_ALLOW_DEPLOYMENT_AUDIT_WARNINGS")) == "1" {
		return false
	}
	if strings.TrimSpace(os.Getenv("ASYMMFLOW_BLOCK_DEPLOYMENT_AUDIT")) == "1" {
		return true
	}
	if strings.TrimSpace(os.Getenv("WAILS_DEV_SERVER_URL")) != "" {
		return false
	}
	if exePath, err := os.Executable(); err == nil {
		if strings.HasSuffix(strings.ToLower(exePath), ".test") {
			return false
		}
	}
	return false
}

func (a *App) logDeploymentAuditResult(message string, audit DeploymentDataAudit) {
	payload := map[string]any{
		"db_path":                  audit.DatabasePath,
		"expected_runtime_db_path": audit.ExpectedRuntimeDatabasePath,
		"packaged_db_path":         audit.PackagedDatabasePath,
		"missing_tables":           audit.MissingTables,
		"blocking_issues":          audit.BlockingDataIssues,
		"warning_issues":           audit.WarningDataIssues,
		"task_items":               audit.TaskItems,
		"expense_entries":          audit.ExpenseEntries,
		"payroll_runs":             audit.PayrollRuns,
		"legacy_offer_shells":      audit.LegacyQuotedOfferShells + audit.LegacyRFQOfferShells,
	}
	if audit.Blocking {
		AppLogger.Error(message, fmt.Errorf("deployment audit found blocking issues"), payload)
		return
	}
	if len(audit.WarningDataIssues) > 0 {
		AppLogger.Warn(message, payload)
		return
	}
	AppLogger.Info(message, payload)
}

func (a *App) computeDeploymentDataAudit() (DeploymentDataAudit, error) {
	if a.db == nil {
		return DeploymentDataAudit{}, fmt.Errorf("database not initialized")
	}

	audit := DeploymentDataAudit{
		GeneratedAt:                 time.Now().UTC().Format(time.RFC3339),
		DatabasePath:                getDatabasePath(),
		ExpectedRuntimeDatabasePath: appDataDatabasePath(),
		PackagedDatabasePath:        packagedDatabasePath(),
		MissingTables:               []string{},
		BlockingDataIssues:          []string{},
		WarningDataIssues:           []string{},
	}
	if a.config != nil && strings.TrimSpace(a.config.Database.Path) != "" {
		audit.DatabasePath = a.config.Database.Path
	}
	audit.RuntimeDatabaseExists = deploymentAuditFileExists(audit.ExpectedRuntimeDatabasePath)
	audit.PackagedDatabaseExists = deploymentAuditFileExists(audit.PackagedDatabasePath)
	audit.UsingRuntimeAppData = samePath(audit.DatabasePath, audit.ExpectedRuntimeDatabasePath)

	for _, table := range criticalDeploymentTableNames {
		if !a.db.Migrator().HasTable(table) {
			audit.MissingTables = append(audit.MissingTables, table)
		}
	}

	var err error
	if audit.ActiveCustomers, err = a.auditCount(`SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL`); err != nil {
		return DeploymentDataAudit{}, err
	}
	if uuidLikeCount, countErr := a.auditCount(`
		SELECT COUNT(*)
		FROM customers
		WHERE deleted_at IS NULL
		  AND customer_id GLOB '????????-????-????-????-????????????'
	`); countErr != nil {
		return DeploymentDataAudit{}, countErr
	} else if uuidLikeCount > 0 {
		audit.BlockingDataIssues = append(audit.BlockingDataIssues, fmt.Sprintf("%d active customers still use UUID-style customer_ids", uuidLikeCount))
	}

	if audit.ActiveInvoices, err = a.auditCount(`SELECT COUNT(*) FROM invoices WHERE deleted_at IS NULL AND ` + deploymentActiveInvoiceStatusWhere); err != nil {
		return DeploymentDataAudit{}, err
	}
	if a.db.Migrator().HasTable("invoices") && a.db.Migrator().HasTable("invoice_items") {
		if audit.ActiveInvoicesWithoutItems, err = a.auditCount(`
			SELECT COUNT(*)
			FROM (
				SELECT i.id
				FROM invoices i
				LEFT JOIN invoice_items ii ON ii.invoice_id = i.id AND ii.deleted_at IS NULL
				WHERE i.deleted_at IS NULL
				  AND i.status NOT IN ('Cancelled', 'Void', 'Proforma', 'Draft')
				GROUP BY i.id
				HAVING COUNT(ii.id) = 0
			) AS hollow_invoices
		`); err != nil {
			return DeploymentDataAudit{}, err
		}
		if audit.ActiveInvoicesWithoutItems > 0 {
			audit.BlockingDataIssues = append(audit.BlockingDataIssues, fmt.Sprintf("%d active invoices have no invoice_items", audit.ActiveInvoicesWithoutItems))
		}
	}

	if audit.ActiveOrders, err = a.auditCount(`SELECT COUNT(*) FROM orders WHERE deleted_at IS NULL`); err != nil {
		return DeploymentDataAudit{}, err
	}
	if a.db.Migrator().HasTable("orders") && a.db.Migrator().HasTable("order_items") {
		if audit.ActiveOrdersWithoutItems, err = a.auditCount(`
			SELECT COUNT(*)
			FROM (
				SELECT o.id
				FROM orders o
				LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL
				WHERE o.deleted_at IS NULL
				GROUP BY o.id
				HAVING COUNT(oi.id) = 0
			) AS hollow_orders
		`); err != nil {
			return DeploymentDataAudit{}, err
		}
		if audit.ActiveOrdersWithoutItems > 0 {
			audit.BlockingDataIssues = append(audit.BlockingDataIssues, fmt.Sprintf("%d active orders have no order_items", audit.ActiveOrdersWithoutItems))
		}
		if audit.ActiveZeroTotalOrders, err = a.auditCount(`
			SELECT COUNT(*)
			FROM orders
			WHERE deleted_at IS NULL
			  AND ABS(COALESCE(total_value_bhd, 0)) < ?
			  AND ABS(COALESCE(grand_total_bhd, 0)) < ?
		`, legacyOfferShellAmountTolerance, legacyOfferShellAmountTolerance); err != nil {
			return DeploymentDataAudit{}, err
		}
		if audit.ActiveZeroTotalOrders > 0 {
			audit.BlockingDataIssues = append(audit.BlockingDataIssues, fmt.Sprintf("%d active orders still have zero totals", audit.ActiveZeroTotalOrders))
		}
	}

	if audit.ActiveOffers, err = a.auditCount(`SELECT COUNT(*) FROM offers WHERE deleted_at IS NULL`); err != nil {
		return DeploymentDataAudit{}, err
	}
	if a.db.Migrator().HasTable("offers") && a.db.Migrator().HasTable("offer_items") {
		if audit.LegacyQuotedOfferShells, err = a.auditCount(`
			SELECT COUNT(*)
			FROM (
				SELECT o.id
				FROM offers o
				LEFT JOIN offer_items oi ON oi.offer_id = o.id AND oi.deleted_at IS NULL
				WHERE o.deleted_at IS NULL
				GROUP BY o.id
				HAVING COUNT(oi.id) = 0
				   AND UPPER(TRIM(COALESCE(o.stage, ''))) = 'QUOTED'
				   AND ABS(COALESCE(o.total_value_bhd, 0)) < ?
			) AS legacy_quoted_shells
		`, legacyOfferShellAmountTolerance); err != nil {
			return DeploymentDataAudit{}, err
		}
		if audit.LegacyRFQOfferShells, err = a.auditCount(`
			SELECT COUNT(*)
			FROM (
				SELECT o.id
				FROM offers o
				LEFT JOIN offer_items oi ON oi.offer_id = o.id AND oi.deleted_at IS NULL
				WHERE o.deleted_at IS NULL
				GROUP BY o.id
				HAVING COUNT(oi.id) = 0
				   AND UPPER(TRIM(COALESCE(o.stage, ''))) = 'RFQ'
				   AND ABS(COALESCE(o.total_value_bhd, 0)) < ?
			) AS legacy_rfq_shells
		`, legacyOfferShellAmountTolerance); err != nil {
			return DeploymentDataAudit{}, err
		}
		if audit.WonOffersWithoutItems, err = a.auditCount(`
			SELECT COUNT(*)
			FROM (
				SELECT o.id
				FROM offers o
				LEFT JOIN offer_items oi ON oi.offer_id = o.id AND oi.deleted_at IS NULL
				WHERE o.deleted_at IS NULL
				GROUP BY o.id
				HAVING COUNT(oi.id) = 0
				   AND UPPER(TRIM(COALESCE(o.stage, ''))) = 'WON'
			) AS won_hollow_offers
		`); err != nil {
			return DeploymentDataAudit{}, err
		}
		var pricedQuotedOfferShells int
		if pricedQuotedOfferShells, err = a.auditCount(`
			SELECT COUNT(*)
			FROM (
				SELECT o.id
				FROM offers o
				LEFT JOIN offer_items oi ON oi.offer_id = o.id AND oi.deleted_at IS NULL
				WHERE o.deleted_at IS NULL
				GROUP BY o.id
				HAVING COUNT(oi.id) = 0
				   AND UPPER(TRIM(COALESCE(o.stage, ''))) IN ('QUOTED', 'RFQ')
				   AND ABS(COALESCE(o.total_value_bhd, 0)) >= ?
			) AS priced_quoted_shells
		`, legacyOfferShellAmountTolerance); err != nil {
			return DeploymentDataAudit{}, err
		}
		if audit.ActiveOperationalOffersNoItems, err = a.auditCount(`
			SELECT COUNT(*)
			FROM (
				SELECT o.id
				FROM offers o
				LEFT JOIN offer_items oi ON oi.offer_id = o.id AND oi.deleted_at IS NULL
				WHERE o.deleted_at IS NULL
				GROUP BY o.id
				HAVING COUNT(oi.id) = 0
				   AND UPPER(TRIM(COALESCE(o.stage, ''))) NOT IN ('QUOTED', 'RFQ')
			) AS operational_hollow_offers
		`); err != nil {
			return DeploymentDataAudit{}, err
		}
		audit.ActiveOperationalOffers = audit.ActiveOffers - audit.LegacyQuotedOfferShells - audit.LegacyRFQOfferShells
		if audit.ActiveOperationalOffers < 0 {
			audit.ActiveOperationalOffers = 0
		}
		if audit.ActiveOperationalOffersNoItems > 0 {
			audit.BlockingDataIssues = append(audit.BlockingDataIssues, fmt.Sprintf("%d active operational offers still have no offer_items", audit.ActiveOperationalOffersNoItems))
		}
		if audit.WonOffersWithoutItems > 0 {
			audit.BlockingDataIssues = append(audit.BlockingDataIssues, fmt.Sprintf("%d won offers still have no offer_items", audit.WonOffersWithoutItems))
		}
		if pricedQuotedOfferShells > 0 {
			audit.WarningDataIssues = append(audit.WarningDataIssues, fmt.Sprintf("%d priced quoted/RFQ offer shells have no offer_items and should be repaired from source documents", pricedQuotedOfferShells))
		}
	}

	if audit.Employees, err = a.countIfTableExists("employees", "deleted_at IS NULL"); err != nil {
		return DeploymentDataAudit{}, err
	}
	if audit.Notifications, err = a.countIfTableExists("notifications", "deleted_at IS NULL"); err != nil {
		return DeploymentDataAudit{}, err
	}
	if audit.TaskItems, err = a.countIfTableExists("task_items", "deleted_at IS NULL"); err != nil {
		return DeploymentDataAudit{}, err
	}
	if audit.ExpenseEntries, err = a.countIfTableExists("expense_entries", "deleted_at IS NULL"); err != nil {
		return DeploymentDataAudit{}, err
	}
	if audit.PayrollRuns, err = a.countIfTableExists("payroll_runs", "deleted_at IS NULL"); err != nil {
		return DeploymentDataAudit{}, err
	}
	if audit.LegacyFollowUpTasks, err = a.countIfTableExists("followup_tasks", "deleted_at IS NULL"); err != nil {
		return DeploymentDataAudit{}, err
	}
	if audit.MigratedLegacyTasks, err = a.countIfTableExists("task_items", "legacy_follow_up_id IS NOT NULL AND deleted_at IS NULL"); err != nil {
		return DeploymentDataAudit{}, err
	}

	if audit.LegacyQuotedOfferShells+audit.LegacyRFQOfferShells > 0 {
		audit.WarningDataIssues = append(
			audit.WarningDataIssues,
			fmt.Sprintf(
				"%d legacy quoted/RFQ offer shells are hidden from the default commercial list and should be reviewed from deployment audit before release",
				audit.LegacyQuotedOfferShells+audit.LegacyRFQOfferShells,
			),
		)
	}
	if audit.LegacyFollowUpTasks > 0 {
		if audit.MigratedLegacyTasks < audit.LegacyFollowUpTasks {
			audit.BlockingDataIssues = append(
				audit.BlockingDataIssues,
				fmt.Sprintf("%d legacy follow-up tasks exist but only %d migrated task_items were created", audit.LegacyFollowUpTasks, audit.MigratedLegacyTasks),
			)
		} else {
			audit.WarningDataIssues = append(
				audit.WarningDataIssues,
				fmt.Sprintf("%d legacy follow-up tasks remain as source rows; active workflow uses %d migrated task_items", audit.LegacyFollowUpTasks, audit.MigratedLegacyTasks),
			)
		}
	}

	audit.Blocking = len(audit.MissingTables) > 0 || len(audit.BlockingDataIssues) > 0
	return audit, nil
}

func (a *App) GetDeploymentDataAudit() (DeploymentDataAudit, error) {
	if !a.canAccessPhase7Rollout() {
		return DeploymentDataAudit{}, fmt.Errorf("access denied")
	}
	return a.computeDeploymentDataAudit()
}

func (a *App) auditCount(query string, args ...any) (int, error) {
	var count int64
	if err := a.db.Raw(query, args...).Scan(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (a *App) countIfTableExists(tableName string, condition string, args ...any) (int, error) {
	if !a.db.Migrator().HasTable(tableName) {
		return 0, nil
	}
	var count int64
	query := a.db.Table(tableName)
	if strings.TrimSpace(condition) != "" {
		query = query.Where(condition, args...)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func samePath(left, right string) bool {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return false
	}
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return left == right
	}
	return leftAbs == rightAbs
}

func deploymentAuditFileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func legacyOfferShellSubquery(db *gorm.DB) *gorm.DB {
	return db.Table("offers AS o").
		Select("o.id").
		Joins("LEFT JOIN offer_items oi ON oi.offer_id = o.id AND oi.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Group("o.id").
		Having(`
			COUNT(oi.id) = 0
			AND UPPER(TRIM(COALESCE(o.stage, ''))) IN ('QUOTED', 'RFQ')
			AND ABS(COALESCE(o.total_value_bhd, 0)) < ?
		`, legacyOfferShellAmountTolerance)
}

func (a *App) enforceCriticalDeploymentState(stage string, foundationErr error) {
	audit, auditErr := a.computeDeploymentDataAudit()
	if auditErr == nil {
		a.logDeploymentAuditResult(fmt.Sprintf("Deployment data audit (%s)", stage), audit)
	}

	if foundationErr == nil && auditErr == nil && !audit.Blocking {
		return
	}

	messageParts := []string{}
	if foundationErr != nil {
		messageParts = append(messageParts, foundationErr.Error())
	}
	if auditErr != nil {
		messageParts = append(messageParts, auditErr.Error())
	}
	if audit.Blocking {
		messageParts = append(messageParts, "deployment audit found blocking issues")
	}
	message := strings.Join(messageParts, "; ")
	if message == "" {
		message = "critical deployment verification failed"
	}

	log.Printf("ðŸš¨ %s", message)
	if auditErr == nil {
		log.Printf("ðŸš¨ Missing tables: %v", audit.MissingTables)
		log.Printf("ðŸš¨ Blocking issues: %v", audit.BlockingDataIssues)
	}
	hardFailure := foundationErr != nil || auditErr != nil || len(audit.MissingTables) > 0
	if !hardFailure && !a.shouldBlockStartupOnDeploymentAudit() {
		return
	}
	if a.logFile != nil {
		_ = a.logFile.Sync()
	}
	os.Exit(1)
}
