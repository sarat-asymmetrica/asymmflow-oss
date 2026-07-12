// Package finance contains display-ready ViewModels for finance screens.
package finance

import (
	vm "ph_holdings_app/internal/viewmodel"
	"ph_holdings_app/internal/viewmodel/shared"
)

// InvoiceListVM is the display contract for the customer invoices screen.
type InvoiceListVM struct {
	Table   shared.TableVM    `json:"table"`
	Summary InvoiceSummaryVM  `json:"summary"`
	Filters InvoiceFiltersVM  `json:"filters"`
	Actions []vm.ActionButton `json:"actions"`
}

// InvoiceSummaryVM contains display-ready invoice KPIs.
type InvoiceSummaryVM struct {
	TotalOutstanding   string `json:"totalOutstanding"`
	OverdueCount       int    `json:"overdueCount"`
	OverdueAmount      string `json:"overdueAmount"`
	PaidThisMonth      string `json:"paidThisMonth"`
	AveragePaymentDays int    `json:"averagePaymentDays"`
}

// InvoiceFiltersVM describes visible invoice filter state.
type InvoiceFiltersVM struct {
	StatusOptions []vm.Option `json:"statusOptions"`
	DateRange     string      `json:"dateRange,omitempty"`
	Search        string      `json:"search,omitempty"`
}

// InvoiceDetailVM is the display contract for a single invoice.
type InvoiceDetailVM struct {
	ID              string               `json:"id"`
	InvoiceNumber   string               `json:"invoiceNumber"`
	CustomerName    string               `json:"customerName"`
	InvoiceDate     string               `json:"invoiceDate"`
	DueDate         string               `json:"dueDate"`
	Status          shared.StatusBadgeVM `json:"status"`
	Items           []InvoiceItemVM      `json:"items"`
	SubtotalDisplay string               `json:"subtotalDisplay"`
	VATDisplay      string               `json:"vatDisplay"`
	TotalDisplay    string               `json:"totalDisplay"`
	PaymentHistory  []PaymentRowVM       `json:"paymentHistory"`
	Actions         []vm.ActionButton    `json:"actions"`
	Breadcrumbs     []vm.BreadcrumbItem  `json:"breadcrumbs"`
}

// InvoiceItemVM is a display-ready invoice line.
type InvoiceItemVM struct {
	ID           string `json:"id"`
	LineNumber   int    `json:"lineNumber"`
	Description  string `json:"description"`
	Quantity     string `json:"quantity"`
	RateDisplay  string `json:"rateDisplay"`
	TotalDisplay string `json:"totalDisplay"`
	ProductCode  string `json:"productCode,omitempty"`
}

// PaymentRowVM is a display-ready payment history row.
type PaymentRowVM struct {
	ID            string `json:"id"`
	PaymentDate   string `json:"paymentDate"`
	AmountDisplay string `json:"amountDisplay"`
	Method        string `json:"method"`
	Reference     string `json:"reference,omitempty"`
	DaysToPayment int    `json:"daysToPayment"`
}

// BankReconciliationVM is the display contract for bank reconciliation.
type BankReconciliationVM struct {
	Statement        StatementHeaderVM       `json:"statement"`
	UnmatchedLines   []BankLineVM            `json:"unmatchedLines"`
	MatchedLines     []BankLineVM            `json:"matchedLines"`
	MatchSuggestions []MatchSuggestionVM     `json:"matchSuggestions"`
	Summary          ReconciliationSummaryVM `json:"summary"`
	Actions          []vm.ActionButton       `json:"actions"`
}

// StatementHeaderVM summarizes an imported bank statement.
type StatementHeaderVM struct {
	ID                    string               `json:"id"`
	StatementNumber       string               `json:"statementNumber"`
	StatementDate         string               `json:"statementDate"`
	PeriodDisplay         string               `json:"periodDisplay"`
	OpeningBalanceDisplay string               `json:"openingBalanceDisplay"`
	ClosingBalanceDisplay string               `json:"closingBalanceDisplay"`
	Status                shared.StatusBadgeVM `json:"status"`
}

// BankLineVM is a display-ready bank statement line.
type BankLineVM struct {
	ID             string               `json:"id"`
	Date           string               `json:"date"`
	Description    string               `json:"description"`
	Reference      string               `json:"reference,omitempty"`
	DebitDisplay   string               `json:"debitDisplay,omitempty"`
	CreditDisplay  string               `json:"creditDisplay,omitempty"`
	BalanceDisplay string               `json:"balanceDisplay"`
	Status         shared.StatusBadgeVM `json:"status"`
}

// MatchSuggestionVM displays a reconciliation candidate.
type MatchSuggestionVM struct {
	LineID            string  `json:"lineId"`
	TargetID          string  `json:"targetId"`
	TargetLabel       string  `json:"targetLabel"`
	ConfidenceDisplay string  `json:"confidenceDisplay"`
	Confidence        float64 `json:"confidence"`
}

// ReconciliationSummaryVM displays reconciliation totals.
type ReconciliationSummaryVM struct {
	TotalLines        int    `json:"totalLines"`
	MatchedLines      int    `json:"matchedLines"`
	UnmatchedLines    int    `json:"unmatchedLines"`
	MatchedAmount     string `json:"matchedAmount"`
	UnmatchedAmount   string `json:"unmatchedAmount"`
	DiscrepancyAmount string `json:"discrepancyAmount"`
}

// CashPositionVM is the display contract for cash position widgets.
type CashPositionVM struct {
	TotalCashDisplay string             `json:"totalCashDisplay"`
	Accounts         []AccountBalanceVM `json:"accounts"`
	Trend            string             `json:"trend"`
}

// AccountBalanceVM is a display-ready bank account balance.
type AccountBalanceVM struct {
	ID             string `json:"id"`
	BankName       string `json:"bankName"`
	AccountName    string `json:"accountName"`
	Currency       string `json:"currency"`
	BalanceDisplay string `json:"balanceDisplay"`
	Status         string `json:"status"`
}

// ExpenseDashboardVM is the display contract for expense dashboards.
type ExpenseDashboardVM struct {
	Dashboard             shared.DashboardVM `json:"dashboard"`
	MonthToDateSpend      string             `json:"monthToDateSpend"`
	UpcomingCommitments   string             `json:"upcomingCommitments"`
	ApprovalQueueCount    int                `json:"approvalQueueCount"`
	RecurringExpenseCount int                `json:"recurringExpenseCount"`
}

// PayrollSummaryVM is the display contract for payroll summaries.
type PayrollSummaryVM struct {
	ActiveProfiles           int    `json:"activeProfiles"`
	OpenPeriods              int    `json:"openPeriods"`
	DraftRuns                int    `json:"draftRuns"`
	ApprovedUnpaidRuns       int    `json:"approvedUnpaidRuns"`
	MonthToDateNetPayroll    string `json:"monthToDateNetPayroll"`
	UpcomingPayrollLiability string `json:"upcomingPayrollLiability"`
}

// FinancialDashboardVM is the display contract for the main finance hub.
type FinancialDashboardVM struct {
	Dashboard    shared.DashboardVM `json:"dashboard"`
	CashPosition CashPositionVM     `json:"cashPosition"`
	ARAgingChart []AgingBucketVM    `json:"arAgingChart"`
	APAgingChart []AgingBucketVM    `json:"apAgingChart"`
	RevenueChart []MonthlyDataVM    `json:"revenueChart"`
}

// AgingBucketVM is a display-ready AR/AP aging bucket.
type AgingBucketVM struct {
	Label         string `json:"label"`
	AmountDisplay string `json:"amountDisplay"`
	Count         int    `json:"count"`
	Color         string `json:"color"`
}

// MonthlyDataVM is a display-ready monthly chart point.
type MonthlyDataVM struct {
	Month        string  `json:"month"`
	ValueDisplay string  `json:"valueDisplay"`
	Value        float64 `json:"value"`
}
