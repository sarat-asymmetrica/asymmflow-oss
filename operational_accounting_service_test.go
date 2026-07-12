package main

// Wave 8 P3 slice 4: operational financial statements.
// Covers the unified ledger (running balance + chronological sort), the thin
// customer/supplier/expense ledger views, source/date/division filtering, the
// P&L aggregation (revenue/COGS/expenses/margins), the balance-sheet net
// position across all six money surfaces, and the finance:view gate.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func operationalTestModels(t *testing.T, app *App) {
	t.Helper()
	require.NoError(t, app.db.AutoMigrate(
		&Invoice{},
		&CustomerReceipt{},
		&Payment{},
		&SupplierInvoice{},
		&SupplierPayment{},
		&ExpenseEntry{},
	))
}

// d2025 returns a fixed in-range instant so ledger/report windows are
// deterministic regardless of when the suite runs.
func d2025(month, day int) time.Time {
	return time.Date(2025, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func fy2025() OperationalLedgerFilter {
	return OperationalLedgerFilter{StartDate: "2025-01-01", EndDate: "2025-12-31"}
}

func TestGetOperationalLedger_RunningBalanceAndSort(t *testing.T) {
	app := setupTestApp(t)
	operationalTestModels(t, app)

	// Seed out of chronological order to prove the stable sort.
	require.NoError(t, app.db.Create(&SupplierPayment{
		SupplierID: "sup-1", SupplierName: "Endress", Reference: "SP-1",
		AmountBHD: 100.000, PaymentDate: d2025(6, 3), PaymentMethod: "Bank Transfer", Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-1", CustomerID: "cust-1", CustomerName: "Acme",
		InvoiceDate: d2025(6, 1), GrandTotalBHD: 1000.000, OutstandingBHD: 1000.000,
		Status: "Sent", Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&CustomerReceipt{
		ReceiptNumber: "RCT-1", CustomerID: "cust-1", CustomerName: "Acme",
		ReceiptDate: d2025(6, 2), AmountBHD: 400.000, UnappliedAmountBHD: 400.000,
		Status: "OnAccount", Division: "Acme Instrumentation",
	}).Error)

	entries, err := app.GetOperationalLedger(fy2025())
	require.NoError(t, err)
	require.Len(t, entries, 3)

	// Chronological: invoice (debit) → receipt (credit) → supplier payment (debit).
	require.Equal(t, "CUSTOMER_INVOICE", entries[0].SourceType)
	require.Equal(t, 1000.0, entries[0].DebitBHD)
	require.Equal(t, 1000.0, entries[0].BalanceBHD)

	require.Equal(t, "CUSTOMER_RECEIPT", entries[1].SourceType)
	require.Equal(t, 400.0, entries[1].CreditBHD)
	require.Equal(t, 600.0, entries[1].BalanceBHD)

	require.Equal(t, "SUPPLIER_PAYMENT", entries[2].SourceType)
	require.Equal(t, 100.0, entries[2].DebitBHD)
	require.Equal(t, 700.0, entries[2].BalanceBHD)
}

func TestGetOperationalLedger_SourceAndDateFilter(t *testing.T) {
	app := setupTestApp(t)
	operationalTestModels(t, app)

	// In-window invoice and an out-of-window invoice (2024).
	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-IN", CustomerID: "cust-1", CustomerName: "Acme",
		InvoiceDate: d2025(3, 10), GrandTotalBHD: 500.000, OutstandingBHD: 500.000,
		Status: "Sent", Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-OLD", CustomerID: "cust-1", CustomerName: "Acme",
		InvoiceDate:   time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
		GrandTotalBHD: 999.000, OutstandingBHD: 999.000, Status: "Sent", Division: "Acme Instrumentation",
	}).Error)
	// A receipt that a CUSTOMER_INVOICE source filter must exclude.
	require.NoError(t, app.db.Create(&CustomerReceipt{
		ReceiptNumber: "RCT-1", CustomerID: "cust-1", CustomerName: "Acme",
		ReceiptDate: d2025(3, 11), AmountBHD: 200.000, UnappliedAmountBHD: 200.000,
		Status: "OnAccount", Division: "Acme Instrumentation",
	}).Error)

	filter := fy2025()
	filter.SourceType = "CUSTOMER_INVOICE"
	entries, err := app.GetOperationalLedger(filter)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "INV-IN", entries[0].SourceNumber)
}

func TestGetCustomerSupplierExpenseLedger_ThinViews(t *testing.T) {
	app := setupTestApp(t)
	operationalTestModels(t, app)

	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-1", CustomerID: "cust-1", CustomerName: "Acme",
		InvoiceDate: d2025(4, 1), GrandTotalBHD: 300.000, OutstandingBHD: 300.000,
		Status: "Sent", Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-2", CustomerID: "cust-2", CustomerName: "Beacon",
		InvoiceDate: d2025(4, 2), GrandTotalBHD: 700.000, OutstandingBHD: 700.000,
		Status: "Sent", Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&SupplierInvoice{
		InvoiceNumber: "SINV-1", SupplierID: "sup-9", SupplierName: "Servomex",
		InvoiceDate: d2025(4, 3), TotalBHD: 250.000, Status: "Verified", PaymentStatus: "unpaid",
		Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&ExpenseEntry{
		EntryNumber: "EXP-1", ExpenseDate: d2025(4, 4), Description: "Rent",
		TotalAmount: 80.000, Status: "approved", PaymentStatus: "unpaid", Division: "Acme Instrumentation",
	}).Error)

	bySource := func(entries []OperationalLedgerEntry, source string) []OperationalLedgerEntry {
		out := make([]OperationalLedgerEntry, 0, len(entries))
		for _, e := range entries {
			if e.SourceType == source {
				out = append(out, e)
			}
		}
		return out
	}

	// Customer ledger pins the CustomerID filter: only cust-1's invoice appears
	// among the customer-typed rows (cust-2's INV-2 is excluded). Non-customer
	// sources remain in the running ledger — faithful to PH's GetOperationalLedger.
	cust, err := app.GetCustomerLedger("cust-1", "2025-01-01", "2025-12-31")
	require.NoError(t, err)
	custInvoices := bySource(cust, "CUSTOMER_INVOICE")
	require.Len(t, custInvoices, 1)
	require.Equal(t, "INV-1", custInvoices[0].SourceNumber)

	// Supplier ledger pins the SupplierID filter on the supplier-typed rows.
	sup, err := app.GetSupplierLedger("sup-9", "2025-01-01", "2025-12-31")
	require.NoError(t, err)
	supInvoices := bySource(sup, "SUPPLIER_INVOICE")
	require.Len(t, supInvoices, 1)
	require.Equal(t, "SINV-1", supInvoices[0].SourceNumber)

	// Expense ledger pins SourceType=EXPENSE, so ONLY expense rows come back.
	exp, err := app.GetExpenseLedger(fy2025())
	require.NoError(t, err)
	require.Len(t, exp, 1)
	require.Equal(t, "EXPENSE", exp[0].SourceType)
	require.Equal(t, 80.0, exp[0].DebitBHD)
}

func TestGetOperationalProfitLoss_Aggregates(t *testing.T) {
	app := setupTestApp(t)
	operationalTestModels(t, app)

	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-1", CustomerID: "cust-1", CustomerName: "Acme",
		InvoiceDate: d2025(5, 1), SubtotalBHD: 1000.000, VATBHD: 100.000, GrandTotalBHD: 1100.000,
		TotalSupplierCostBHD: 700.000, OutstandingBHD: 0, Status: "Paid", Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-2", CustomerID: "cust-2", CustomerName: "Beacon",
		InvoiceDate: d2025(5, 2), SubtotalBHD: 500.000, VATBHD: 50.000, GrandTotalBHD: 550.000,
		TotalSupplierCostBHD: 300.000, OutstandingBHD: 550.000, Status: "Sent", Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&ExpenseEntry{
		EntryNumber: "EXP-1", ExpenseDate: d2025(5, 3), Description: "Rent",
		TotalAmount: 100.000, Status: "approved", PaymentStatus: "unpaid", Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&ExpenseEntry{
		EntryNumber: "EXP-2", ExpenseDate: d2025(5, 4), Description: "Utilities",
		TotalAmount: 50.000, Status: "paid", PaymentStatus: "paid", Division: "Acme Instrumentation",
	}).Error)

	report, err := app.GetOperationalProfitLoss(fy2025())
	require.NoError(t, err)
	require.Equal(t, 2, report.InvoiceCount)
	require.Equal(t, 2, report.ExpenseCount)
	require.Equal(t, 1500.0, report.RevenueBHD)
	require.Equal(t, 1000.0, report.COGSBHD)
	require.Equal(t, 500.0, report.GrossProfitBHD)
	require.Equal(t, 150.0, report.ExpensesBHD)
	require.Equal(t, 350.0, report.NetIncomeBHD)
	require.InDelta(t, 33.333, report.GrossMarginPercent, 0.001)
	require.InDelta(t, 23.333, report.NetMarginPercent, 0.001)
	// CategoryName is a non-persisted column, so a fresh load groups all under Uncategorized.
	require.Len(t, report.ExpenseBreakdown, 1)
	require.Equal(t, "Uncategorized", report.ExpenseBreakdown[0].AccountName)
	require.Equal(t, 150.0, report.ExpenseBreakdown[0].Balance)
}

func TestGetOperationalBalanceSheet_NetPosition(t *testing.T) {
	app := setupTestApp(t)
	operationalTestModels(t, app)

	// AR: one open invoice, outstanding 300.
	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-1", CustomerID: "cust-1", CustomerName: "Acme",
		InvoiceDate: d2025(2, 1), GrandTotalBHD: 300.000, OutstandingBHD: 300.000,
		Status: "Sent", Division: "Acme Instrumentation",
	}).Error)
	// Cash: receipt 200 (unapplied 50 → customer credits) + legacy payment 100.
	require.NoError(t, app.db.Create(&CustomerReceipt{
		ReceiptNumber: "RCT-1", CustomerID: "cust-1", CustomerName: "Acme",
		ReceiptDate: d2025(2, 2), AmountBHD: 200.000, AppliedAmountBHD: 150.000,
		UnappliedAmountBHD: 50.000, Status: "PartiallyApplied", Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&Payment{
		InvoiceID: "inv-legacy", InvoiceNumber: "INV-L", AmountBHD: 100.000,
		PaymentDate: d2025(2, 3), PaymentMethod: "Cash", Division: "Acme Instrumentation",
	}).Error)
	// AP: unpaid supplier invoice 150.
	require.NoError(t, app.db.Create(&SupplierInvoice{
		InvoiceNumber: "SINV-1", SupplierID: "sup-1", SupplierName: "Endress",
		InvoiceDate: d2025(2, 4), TotalBHD: 150.000, Status: "Verified", PaymentStatus: "unpaid",
		Division: "Acme Instrumentation",
	}).Error)
	// Expense liability: approved+unpaid 80.
	require.NoError(t, app.db.Create(&ExpenseEntry{
		EntryNumber: "EXP-1", ExpenseDate: d2025(2, 5), Description: "Rent",
		TotalAmount: 80.000, Status: "approved", PaymentStatus: "unpaid", Division: "Acme Instrumentation",
	}).Error)
	// Cash outflow: supplier payment 40.
	require.NoError(t, app.db.Create(&SupplierPayment{
		SupplierID: "sup-1", SupplierName: "Endress", Reference: "SP-1",
		AmountBHD: 40.000, PaymentDate: d2025(2, 6), PaymentMethod: "Bank Transfer", Division: "Acme Instrumentation",
	}).Error)
	// Cash outflow: paid expense 30 (excluded from liability, subtracted from cash).
	paidAt := d2025(2, 7)
	require.NoError(t, app.db.Create(&ExpenseEntry{
		EntryNumber: "EXP-2", ExpenseDate: d2025(2, 7), Description: "Fuel",
		TotalAmount: 30.000, Status: "paid", PaymentStatus: "paid", PaidAt: &paidAt, Division: "Acme Instrumentation",
	}).Error)

	report, err := app.GetOperationalBalanceSheet("2025-12-31", "")
	require.NoError(t, err)
	require.Equal(t, 300.0, report.AccountsReceivableBHD)
	require.Equal(t, 50.0, report.CustomerCreditsBHD)
	require.Equal(t, 150.0, report.AccountsPayableBHD)
	require.Equal(t, 80.0, report.ExpenseLiabilityBHD)
	// Cash = 200 (receipt) + 100 (legacy pay) - 40 (supplier pay) - 30 (paid expense) = 230.
	require.Equal(t, 230.0, report.CashBHD)
	// Net = 230 + 300 - 50 - 150 - 80 = 250.
	require.Equal(t, 250.0, report.NetPositionBHD)
}

func TestGetOperationalLedger_DivisionFilter(t *testing.T) {
	app := setupTestApp(t)
	operationalTestModels(t, app)

	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-PH", CustomerID: "cust-1", CustomerName: "Acme",
		InvoiceDate: d2025(6, 1), GrandTotalBHD: 100.000, OutstandingBHD: 100.000,
		Status: "Sent", Division: "Acme Instrumentation",
	}).Error)
	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-BC", CustomerID: "cust-2", CustomerName: "Beacon",
		InvoiceDate: d2025(6, 2), GrandTotalBHD: 200.000, OutstandingBHD: 200.000,
		Status: "Sent", Division: "Beacon Controls",
	}).Error)

	filter := fy2025()
	filter.Division = "Beacon Controls"
	entries, err := app.GetOperationalLedger(filter)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "INV-BC", entries[0].SourceNumber)
}

// Guards the sovereign division-scoping fix: a row with a BLANK division must
// scope to the overlay DEFAULT division (Acme Instrumentation), not vanish. A
// verbatim PH port hardcodes 'PH Trading' as the COALESCE fallback, which is
// stale on the sovereign default and would silently drop this row from every
// unfiltered report.
func TestGetOperationalLedger_BlankDivisionScopesToDefault(t *testing.T) {
	app := setupTestApp(t)
	operationalTestModels(t, app)

	// Blank-division invoice → belongs to the default division.
	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-BLANK", CustomerID: "cust-1", CustomerName: "Acme",
		InvoiceDate: d2025(7, 1), GrandTotalBHD: 111.000, OutstandingBHD: 111.000,
		Status: "Sent", Division: "",
	}).Error)
	// A different division's invoice must NOT appear under the default scope.
	require.NoError(t, app.db.Create(&Invoice{
		InvoiceNumber: "INV-BC", CustomerID: "cust-2", CustomerName: "Beacon",
		InvoiceDate: d2025(7, 2), GrandTotalBHD: 222.000, OutstandingBHD: 222.000,
		Status: "Sent", Division: "Beacon Controls",
	}).Error)

	// Empty filter.Division → default division (Acme Instrumentation).
	entries, err := app.GetOperationalLedger(fy2025())
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "INV-BLANK", entries[0].SourceNumber)

	// The balance sheet's AR must include the blank-division row under default scope.
	bs, err := app.GetOperationalBalanceSheet("2025-12-31", "")
	require.NoError(t, err)
	require.Equal(t, 111.0, bs.AccountsReceivableBHD)
}

func TestOperationalStatements_PermissionGate(t *testing.T) {
	app := setupTestApp(t)
	operationalTestModels(t, app)

	// Strip the wildcard grant → finance:view must be denied on every endpoint.
	app.currentUser.Role.Permissions = `["dashboard:view"]`

	_, err := app.GetOperationalLedger(fy2025())
	require.Error(t, err)
	_, err = app.GetOperationalProfitLoss(fy2025())
	require.Error(t, err)
	_, err = app.GetOperationalBalanceSheet("2025-12-31", "")
	require.Error(t, err)
}
