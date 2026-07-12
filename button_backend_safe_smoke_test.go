package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestButtonBackendSafeSmokeHarness exercises read-only or export-only Wails
// methods used by major UI buttons and dashboards. It deliberately avoids
// destructive create/update/delete flows and uses the in-memory test database.
func TestButtonBackendSafeSmokeHarness(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(
		&TallyInvoiceImport{},
		&TallyPurchaseImport{},
		&InventoryItem{},
		&SupplierInvoice{},
		&SupplierInvoiceItem{},
		&SupplierPayment{},
		&PurchaseOrder{},
		&PurchaseOrderItem{},
		&GoodsReceivedNote{},
		&GRNItem{},
		&DeliveryNote{},
		&DeliveryNoteItem{},
		&SerialNumber{},
		&CreditNote{},
		&CreditNoteItem{},
		&RFQData{},
		&CostingSheetData{},
		&CostingLineItem{},
		&OfferData{},
		&OfferNote{},
		&Contract{},
		&ContractTemplate{},
		&CurrencyExchangeRate{},
		&FXRevaluation{},
		&Warehouse{},
		&StockMovement{},
		&OCRDocument{},
		&QuickCapture{},
		&AuditLog{},
		&PredictionRecord{},
	))
	require.NoError(t, app.SeedDefaultRoles())
	app.config = &Config{
		Database: DatabaseConfig{Path: "file:button-smoke?mode=memory&cache=shared"},
		App:      AppConfig{AllowedOrigins: "http://localhost:5173"},
	}

	currentYear := time.Now().Year()
	start := time.Date(currentYear, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 3, 0).AddDate(0, 0, -1)
	posted := false

	checks := []struct {
		name string
		call func() error
	}{
		{name: "auth current user role", call: func() error {
			require.NotEmpty(t, app.GetCurrentUserRole())
			return nil
		}},
		{name: "user management list users", call: func() error {
			_, err := app.ListUsers()
			return err
		}},
		{name: "user management list roles", call: func() error {
			_, err := app.ListRoles()
			return err
		}},
		{name: "user management role permissions", call: func() error {
			_, err := app.GetRolePermissionsList("admin")
			return err
		}},
		{name: "user management audit logs", call: func() error {
			_, err := app.GetAuditLogs(10, "all", "")
			return err
		}},
		{name: "dashboard stats", call: func() error {
			_, err := app.GetDashboardStats()
			return err
		}},
		{name: "dashboard events", call: func() error {
			_, err := app.GetDashboardEvents(10)
			return err
		}},
		{name: "sales pipeline opportunities", call: func() error {
			_, err := app.GetPipelineOpportunities(10, 0)
			return err
		}},
		{name: "sales rfqs", call: func() error {
			_, err := app.GetRFQs(10, 0)
			return err
		}},
		{name: "costing sheets", call: func() error {
			_, err := app.GetCostingSheets(10)
			return err
		}},
		{name: "offers list", call: func() error {
			_, err := app.GetOffers(10)
			return err
		}},
		{name: "orders list", call: func() error {
			_, err := app.ListOrders(10, 0)
			return err
		}},
		{name: "customer invoices list", call: func() error {
			_, err := app.ListCustomerInvoices(10, 0)
			return err
		}},
		{name: "customer payments list", call: func() error {
			_, err := app.GetAllPayments(10, 0)
			return err
		}},
		{name: "supplier payments list", call: func() error {
			_, err := app.GetAllSupplierPayments()
			return err
		}},
		{name: "customers list", call: func() error {
			_, err := app.ListCustomers(10, 0)
			return err
		}},
		{name: "suppliers list", call: func() error {
			_, err := app.ListSuppliers(10, 0)
			return err
		}},
		{name: "crm customer dashboard", call: func() error {
			_ = app.GetCRMCustomerDashboard()
			return nil
		}},
		{name: "crm supplier dashboard", call: func() error {
			_ = app.GetCRMSupplierDashboard()
			return nil
		}},
		{name: "purchase orders list", call: func() error {
			_, err := app.GetPurchaseOrders()
			return err
		}},
		{name: "supplier invoices list", call: func() error {
			_, err := app.GetSupplierInvoices()
			return err
		}},
		{name: "delivery notes list", call: func() error {
			_, err := app.GetDeliveryNotes()
			return err
		}},
		{name: "grn list", call: func() error {
			_, err := app.ListGRNs(10, 0, "")
			return err
		}},
		{name: "accounting chart of accounts", call: func() error {
			_, err := app.GetChartOfAccounts("All")
			return err
		}},
		{name: "accounting journal entries", call: func() error {
			_, err := app.GetJournalEntries(currentYear, 0, &posted, 10)
			return err
		}},
		{name: "accounting generate profit and loss", call: func() error {
			_, err := app.GenerateProfitAndLoss(currentYear)
			return err
		}},
		{name: "accounting generate balance sheet", call: func() error {
			_, err := app.GenerateBalanceSheet(currentYear)
			return err
		}},
		{name: "accounting export vat return", call: func() error {
			_, err := app.ExportVATReturnData(currentYear, 1)
			return err
		}},
		{name: "finance dashboard", call: func() error {
			_ = app.GetFinancialDashboard()
			return nil
		}},
		{name: "finance dynamic dashboard", call: func() error {
			_, err := app.GetDynamicFinancialDashboard(currentYear)
			return err
		}},
		{name: "finance available years", call: func() error {
			_, err := app.GetAvailableFinancialYears()
			return err
		}},
		{name: "finance cash flow projection", call: func() error {
			_, err := app.GetCashFlowProjection(30)
			return err
		}},
		{name: "finance vat reconciliation", call: func() error {
			_, err := app.GetVATReconciliation(start.Format("2006-01-02"), end.Format("2006-01-02"))
			return err
		}},
		{name: "finance ar aging", call: func() error {
			_, err := app.GetARAgingReport()
			return err
		}},
		{name: "finance ap aging", call: func() error {
			_, err := app.GetAPAgingReport()
			return err
		}},
		{name: "expenses categories", call: func() error {
			_, err := app.ListExpenseCategories(true)
			return err
		}},
		{name: "expenses entries", call: func() error {
			_, err := app.ListExpenseEntries("", true)
			return err
		}},
		{name: "expenses dashboard summary", call: func() error {
			_, err := app.ListExpenseDashboardSummary()
			return err
		}},
		{name: "people employee profiles", call: func() error {
			_, err := app.ListEmployeeProfiles(true)
			return err
		}},
		{name: "people access links", call: func() error {
			_, err := app.ListEmployeeAccessLinks()
			return err
		}},
		{name: "work projects", call: func() error {
			_, err := app.ListCollaborativeProjects(true)
			return err
		}},
		{name: "work team tasks", call: func() error {
			_, err := app.ListCollaborativeTeamTasks(true)
			return err
		}},
		{name: "notifications count", call: func() error {
			_, err := app.GetUnreadNotificationsCount()
			return err
		}},
		{name: "notifications feed", call: func() error {
			_, err := app.ListNotificationFeed(10, false)
			return err
		}},
		{name: "payroll compensation profiles", call: func() error {
			_, err := app.ListEmployeeCompensationProfiles(true)
			return err
		}},
		{name: "payroll periods", call: func() error {
			_, err := app.ListPayrollPeriods(true)
			return err
		}},
		{name: "payroll dashboard summary", call: func() error {
			_, err := app.ListPayrollDashboardSummary()
			return err
		}},
		{name: "settings map", call: func() error {
			_, err := app.GetSettings()
			return err
		}},
		{name: "settings supported currencies", call: func() error {
			_, err := app.GetSupportedCurrencies()
			return err
		}},
		{name: "settings active currency rates", call: func() error {
			_, err := app.GetActiveCurrencyRates()
			return err
		}},
		{name: "deployment data audit", call: func() error {
			_, err := app.GetDeploymentDataAudit()
			return err
		}},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				require.NoError(t, check.call())
			})
		})
	}
}
