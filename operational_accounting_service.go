package main

// Wave 8 P3 slice 4 (Bucket B): operational financial statements.
// Ported verbatim from the frozen PH reference (operational_accounting_service.go).
// These are read-only reporting endpoints that aggregate over already-migrated
// transaction tables (customer invoices, customer receipts, invoice payments,
// supplier invoices, supplier payments, expense entries) — no new schema.
//
//   - GetOperationalLedger — unified debit/credit ledger across every money
//     source, with a running balance; the customer/supplier/expense ledgers are
//     thin filtered views over it.
//   - GetOperationalProfitLoss — revenue/COGS/expense P&L with expense breakdown.
//   - GetOperationalBalanceSheet — cash / AR / customer-credits / AP / expense-
//     liability net position as of a date.
//
// Every type and helper it needs (Invoice/CustomerReceipt/Payment/SupplierInvoice/
// SupplierPayment/ExpenseEntry/AccountBalance, roundBHD/normalizeDivisionName/
// firstNonEmptyString/parseDate/resolveSupplierInvoiceDivision) already exists on
// the substrate, so this is a name-for-name port. Wires the FinanceHub Accounting
// screen (P4). Read gate: finance:view.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type OperationalLedgerFilter struct {
	Division   string `json:"division"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	CustomerID string `json:"customer_id"`
	SupplierID string `json:"supplier_id"`
	VendorID   string `json:"vendor_id"`
	SourceType string `json:"source_type"`
	Status     string `json:"status"`
}

type OperationalLedgerEntry struct {
	ID           string  `json:"id"`
	SourceType   string  `json:"source_type"`
	SourceID     string  `json:"source_id"`
	SourceNumber string  `json:"source_number"`
	EntryDate    string  `json:"entry_date"`
	PartyID      string  `json:"party_id"`
	PartyName    string  `json:"party_name"`
	PartyType    string  `json:"party_type"`
	Description  string  `json:"description"`
	DebitBHD     float64 `json:"debit_bhd"`
	CreditBHD    float64 `json:"credit_bhd"`
	BalanceBHD   float64 `json:"balance_bhd"`
	Status       string  `json:"status"`
	Division     string  `json:"division"`
}

type OperationalProfitLossReport struct {
	StartDate          string           `json:"start_date"`
	EndDate            string           `json:"end_date"`
	Division           string           `json:"division"`
	RevenueBHD         float64          `json:"revenue_bhd"`
	COGSBHD            float64          `json:"cogs_bhd"`
	GrossProfitBHD     float64          `json:"gross_profit_bhd"`
	GrossMarginPercent float64          `json:"gross_margin_percent"`
	ExpensesBHD        float64          `json:"expenses_bhd"`
	NetIncomeBHD       float64          `json:"net_income_bhd"`
	NetMarginPercent   float64          `json:"net_margin_percent"`
	InvoiceCount       int              `json:"invoice_count"`
	ExpenseCount       int              `json:"expense_count"`
	ExpenseBreakdown   []AccountBalance `json:"expense_breakdown"`
}

type OperationalBalanceSheetReport struct {
	AsOfDate              string  `json:"as_of_date"`
	Division              string  `json:"division"`
	CashBHD               float64 `json:"cash_bhd"`
	AccountsReceivableBHD float64 `json:"accounts_receivable_bhd"`
	CustomerCreditsBHD    float64 `json:"customer_credits_bhd"`
	AccountsPayableBHD    float64 `json:"accounts_payable_bhd"`
	ExpenseLiabilityBHD   float64 `json:"expense_liability_bhd"`
	NetPositionBHD        float64 `json:"net_position_bhd"`
}

func (a *App) GetOperationalLedger(filter OperationalLedgerFilter) ([]OperationalLedgerEntry, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	start, end := operationalDateRange(filter.StartDate, filter.EndDate)
	entries := make([]OperationalLedgerEntry, 0, 128)

	if operationalSourceIncluded(filter.SourceType, "CUSTOMER_INVOICE") {
		var invoices []Invoice
		query := a.db.Where("invoice_date BETWEEN ? AND ? AND status IN ?", start, end, operationalInvoiceStatuses())
		query = applyDivisionFilter(query, filter.Division)
		if filter.CustomerID != "" {
			query = query.Where("customer_id = ?", filter.CustomerID)
		}
		if filter.Status != "" {
			query = query.Where("LOWER(status) = ?", strings.ToLower(filter.Status))
		}
		if err := query.Find(&invoices).Error; err != nil {
			return nil, fmt.Errorf("failed to load customer invoices: %w", err)
		}
		for _, invoice := range invoices {
			entries = append(entries, OperationalLedgerEntry{
				ID:           "invoice:" + invoice.ID,
				SourceType:   "CUSTOMER_INVOICE",
				SourceID:     invoice.ID,
				SourceNumber: invoice.InvoiceNumber,
				EntryDate:    formatOperationalDate(invoice.InvoiceDate),
				PartyID:      invoice.CustomerID,
				PartyName:    invoice.CustomerName,
				PartyType:    "Customer",
				Description:  "Customer invoice",
				DebitBHD:     roundBHD(invoice.GrandTotalBHD),
				Status:       invoice.Status,
				Division:     normalizeDivisionName(invoice.Division),
			})
		}
	}

	if operationalSourceIncluded(filter.SourceType, "CUSTOMER_RECEIPT") {
		var receipts []CustomerReceipt
		query := a.db.Where("receipt_date BETWEEN ? AND ?", start, end)
		query = applyDivisionFilter(query, filter.Division)
		if filter.CustomerID != "" {
			query = query.Where("customer_id = ?", filter.CustomerID)
		}
		if filter.Status != "" {
			query = query.Where("LOWER(status) = ?", strings.ToLower(filter.Status))
		}
		if err := query.Find(&receipts).Error; err != nil {
			return nil, fmt.Errorf("failed to load customer receipts: %w", err)
		}
		for _, receipt := range receipts {
			description := "Customer receipt"
			if receipt.UnappliedAmountBHD > 0 {
				description = "Customer receipt - on account"
			}
			entries = append(entries, OperationalLedgerEntry{
				ID:           "receipt:" + receipt.ID,
				SourceType:   "CUSTOMER_RECEIPT",
				SourceID:     receipt.ID,
				SourceNumber: receipt.ReceiptNumber,
				EntryDate:    formatOperationalDate(receipt.ReceiptDate),
				PartyID:      receipt.CustomerID,
				PartyName:    receipt.CustomerName,
				PartyType:    "Customer",
				Description:  description,
				CreditBHD:    roundBHD(receipt.AmountBHD),
				Status:       receipt.Status,
				Division:     normalizeDivisionName(receipt.Division),
			})
		}
	}

	if operationalSourceIncluded(filter.SourceType, "CUSTOMER_PAYMENT") {
		type paymentLedgerRow struct {
			ID              string
			InvoiceID       string
			InvoiceNumber   string
			CustomerID      string
			CustomerName    string
			PaymentDate     time.Time
			AmountBHD       float64
			PaymentMethod   string
			Reference       string
			PaymentDivision string
			InvoiceDivision string
		}

		var payments []paymentLedgerRow
		query := a.db.Table("payments AS p").
			Select(`p.id, p.invoice_id, i.invoice_number, i.customer_id, i.customer_name, p.payment_date, p.amount_bhd,
				p.payment_method, p.reference, p.division AS payment_division, i.division AS invoice_division`).
			Joins("LEFT JOIN invoices AS i ON i.id = p.invoice_id").
			Where("p.payment_date BETWEEN ? AND ?", start, end).
			Where("(p.receipt_id IS NULL OR TRIM(p.receipt_id) = '')")
		if strings.TrimSpace(filter.Division) != "" {
			// Effective division = payment's own division, else the linked
			// invoice's, else blank; normalizeDivisionSQL maps blank → the
			// overlay default. (PH hardcoded 'PH Trading' as the fallback,
			// which is stale on the sovereign default division.)
			query = query.Where(
				normalizeDivisionSQL("COALESCE(NULLIF(TRIM(p.division), ''), NULLIF(TRIM(i.division), ''), '')")+" = ?",
				normalizeDivisionName(filter.Division),
			)
		}
		if filter.CustomerID != "" {
			query = query.Where("i.customer_id = ?", filter.CustomerID)
		}
		if filter.Status != "" && !strings.EqualFold(strings.TrimSpace(filter.Status), "received") {
			query = query.Where("1 = 0")
		}
		if err := query.Scan(&payments).Error; err != nil {
			return nil, fmt.Errorf("failed to load customer payments: %w", err)
		}
		for _, payment := range payments {
			division := firstNonEmptyString(payment.PaymentDivision, payment.InvoiceDivision)
			entries = append(entries, OperationalLedgerEntry{
				ID:           "payment:" + payment.ID,
				SourceType:   "CUSTOMER_PAYMENT",
				SourceID:     payment.ID,
				SourceNumber: firstNonEmptyString(payment.Reference, payment.InvoiceNumber),
				EntryDate:    formatOperationalDate(payment.PaymentDate),
				PartyID:      payment.CustomerID,
				PartyName:    payment.CustomerName,
				PartyType:    "Customer",
				Description:  "Customer payment",
				CreditBHD:    roundBHD(payment.AmountBHD),
				Status:       "Received",
				Division:     normalizeDivisionName(division),
			})
		}
	}

	if operationalSourceIncluded(filter.SourceType, "SUPPLIER_INVOICE") {
		var invoices []SupplierInvoice
		query := a.db.Where("invoice_date BETWEEN ? AND ?", start, end)
		query = applyDivisionFilter(query, filter.Division)
		if filter.SupplierID != "" {
			query = query.Where("supplier_id = ?", filter.SupplierID)
		}
		if filter.Status != "" {
			query = query.Where("LOWER(status) = ?", strings.ToLower(filter.Status))
		}
		if err := query.Find(&invoices).Error; err != nil {
			return nil, fmt.Errorf("failed to load supplier invoices: %w", err)
		}
		for _, invoice := range invoices {
			entries = append(entries, OperationalLedgerEntry{
				ID:           "supplier_invoice:" + invoice.ID,
				SourceType:   "SUPPLIER_INVOICE",
				SourceID:     invoice.ID,
				SourceNumber: invoice.InvoiceNumber,
				EntryDate:    formatOperationalDate(invoice.InvoiceDate),
				PartyID:      invoice.SupplierID,
				PartyName:    invoice.SupplierName,
				PartyType:    "Supplier",
				Description:  "Supplier invoice",
				CreditBHD:    roundBHD(invoice.TotalBHD),
				Status:       firstNonEmptyString(invoice.PaymentStatus, invoice.Status),
				Division:     a.resolveSupplierInvoiceDivision(invoice),
			})
		}
	}

	if operationalSourceIncluded(filter.SourceType, "SUPPLIER_PAYMENT") {
		var payments []SupplierPayment
		query := a.db.Where("payment_date BETWEEN ? AND ?", start, end)
		query = applyDivisionFilter(query, filter.Division)
		if filter.SupplierID != "" {
			query = query.Where("supplier_id = ?", filter.SupplierID)
		}
		if err := query.Find(&payments).Error; err != nil {
			return nil, fmt.Errorf("failed to load supplier payments: %w", err)
		}
		for _, payment := range payments {
			entries = append(entries, OperationalLedgerEntry{
				ID:           "supplier_payment:" + payment.ID,
				SourceType:   "SUPPLIER_PAYMENT",
				SourceID:     payment.ID,
				SourceNumber: payment.Reference,
				EntryDate:    formatOperationalDate(payment.PaymentDate),
				PartyID:      payment.SupplierID,
				PartyName:    payment.SupplierName,
				PartyType:    "Supplier",
				Description:  "Supplier settlement",
				DebitBHD:     roundBHD(payment.AmountBHD),
				Status:       "Paid",
				Division:     normalizeDivisionName(payment.Division),
			})
		}
	}

	if operationalSourceIncluded(filter.SourceType, "EXPENSE") {
		var expenses []ExpenseEntry
		query := a.db.Where("expense_date BETWEEN ? AND ? AND LOWER(status) IN ?", start, end, []string{"approved", "posted", "paid"})
		query = applyDivisionFilter(query, filter.Division)
		if filter.VendorID != "" {
			query = query.Where("vendor_id = ?", filter.VendorID)
		}
		if filter.Status != "" {
			query = query.Where("LOWER(status) = ? OR LOWER(payment_status) = ?", strings.ToLower(filter.Status), strings.ToLower(filter.Status))
		}
		if err := query.Find(&expenses).Error; err != nil {
			return nil, fmt.Errorf("failed to load expense entries: %w", err)
		}
		for _, expense := range expenses {
			partyName := firstNonEmptyString(expense.VendorName, expense.CategoryName, "Expense")
			partyID := expense.CategoryID
			if expense.VendorID != nil && strings.TrimSpace(*expense.VendorID) != "" {
				partyID = *expense.VendorID
			}
			entries = append(entries, OperationalLedgerEntry{
				ID:           "expense:" + expense.ID,
				SourceType:   "EXPENSE",
				SourceID:     expense.ID,
				SourceNumber: expense.EntryNumber,
				EntryDate:    formatOperationalDate(expense.ExpenseDate),
				PartyID:      partyID,
				PartyName:    partyName,
				PartyType:    "Expense",
				Description:  expense.Description,
				DebitBHD:     roundBHD(expense.TotalAmount),
				Status:       firstNonEmptyString(expense.PaymentStatus, expense.Status),
				Division:     normalizeDivisionName(expense.Division),
			})
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].EntryDate == entries[j].EntryDate {
			return entries[i].SourceNumber < entries[j].SourceNumber
		}
		return entries[i].EntryDate < entries[j].EntryDate
	})
	running := 0.0
	for i := range entries {
		running = roundBHD(running + entries[i].DebitBHD - entries[i].CreditBHD)
		entries[i].BalanceBHD = running
	}
	return entries, nil
}

func (a *App) GetCustomerLedger(customerID, startDate, endDate string) ([]OperationalLedgerEntry, error) {
	return a.GetOperationalLedger(OperationalLedgerFilter{
		CustomerID: customerID,
		StartDate:  startDate,
		EndDate:    endDate,
	})
}

func (a *App) GetSupplierLedger(supplierID, startDate, endDate string) ([]OperationalLedgerEntry, error) {
	return a.GetOperationalLedger(OperationalLedgerFilter{
		SupplierID: supplierID,
		StartDate:  startDate,
		EndDate:    endDate,
	})
}

func (a *App) GetExpenseLedger(filter OperationalLedgerFilter) ([]OperationalLedgerEntry, error) {
	filter.SourceType = "EXPENSE"
	return a.GetOperationalLedger(filter)
}

func (a *App) GetOperationalProfitLoss(filter OperationalLedgerFilter) (*OperationalProfitLossReport, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	start, end := operationalDateRange(filter.StartDate, filter.EndDate)
	report := &OperationalProfitLossReport{
		StartDate: formatOperationalDate(start),
		EndDate:   formatOperationalDate(end),
		Division:  normalizeDivisionName(filter.Division),
	}

	var invoices []Invoice
	invoiceQuery := a.db.Where("invoice_date BETWEEN ? AND ? AND status IN ?", start, end, operationalInvoiceStatuses())
	invoiceQuery = applyDivisionFilter(invoiceQuery, filter.Division)
	if err := invoiceQuery.Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to load P&L invoices: %w", err)
	}
	report.InvoiceCount = len(invoices)
	for _, invoice := range invoices {
		revenue := invoice.SubtotalBHD
		if revenue <= 0 {
			revenue = invoice.GrandTotalBHD - invoice.VATBHD
		}
		report.RevenueBHD = roundBHD(report.RevenueBHD + revenue)
		report.COGSBHD = roundBHD(report.COGSBHD + invoice.TotalSupplierCostBHD)
	}

	var expenses []ExpenseEntry
	expenseQuery := a.db.Where("expense_date BETWEEN ? AND ? AND LOWER(status) IN ?", start, end, []string{"approved", "posted", "paid"})
	expenseQuery = applyDivisionFilter(expenseQuery, filter.Division)
	if err := expenseQuery.Find(&expenses).Error; err != nil {
		return nil, fmt.Errorf("failed to load P&L expenses: %w", err)
	}
	report.ExpenseCount = len(expenses)
	breakdown := map[string]float64{}
	for _, expense := range expenses {
		amount := roundBHD(expense.TotalAmount)
		report.ExpensesBHD = roundBHD(report.ExpensesBHD + amount)
		key := firstNonEmptyString(expense.CategoryName, "Uncategorized")
		breakdown[key] = roundBHD(breakdown[key] + amount)
	}
	for name, amount := range breakdown {
		report.ExpenseBreakdown = append(report.ExpenseBreakdown, AccountBalance{
			AccountName: name,
			Balance:     amount,
		})
	}
	sort.Slice(report.ExpenseBreakdown, func(i, j int) bool {
		return report.ExpenseBreakdown[i].Balance > report.ExpenseBreakdown[j].Balance
	})

	report.GrossProfitBHD = roundBHD(report.RevenueBHD - report.COGSBHD)
	report.NetIncomeBHD = roundBHD(report.GrossProfitBHD - report.ExpensesBHD)
	if report.RevenueBHD > 0 {
		report.GrossMarginPercent = roundBHD((report.GrossProfitBHD / report.RevenueBHD) * 100)
		report.NetMarginPercent = roundBHD((report.NetIncomeBHD / report.RevenueBHD) * 100)
	}
	return report, nil
}

func (a *App) GetOperationalBalanceSheet(asOfDate, division string) (*OperationalBalanceSheetReport, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	_, asOf := operationalDateRange("", asOfDate)
	report := &OperationalBalanceSheetReport{
		AsOfDate: formatOperationalDate(asOf),
		Division: normalizeDivisionName(division),
	}

	invoiceQuery := a.db.Model(&Invoice{}).
		Where("invoice_date <= ? AND status IN ?", asOf, operationalInvoiceStatuses())
	invoiceQuery = applyDivisionFilter(invoiceQuery, division)
	invoiceQuery.Select("COALESCE(SUM(outstanding_bhd), 0)").Scan(&report.AccountsReceivableBHD)

	receiptQuery := a.db.Model(&CustomerReceipt{}).Where("receipt_date <= ?", asOf)
	receiptQuery = applyDivisionFilter(receiptQuery, division)
	receiptQuery.Select("COALESCE(SUM(amount_bhd), 0)").Scan(&report.CashBHD)
	receiptQuery.Select("COALESCE(SUM(unapplied_amount_bhd), 0)").Scan(&report.CustomerCreditsBHD)

	var legacyCustomerPayments float64
	legacyPaymentQuery := a.db.Model(&Payment{}).
		Where("payment_date <= ?", asOf).
		Where("(receipt_id IS NULL OR TRIM(receipt_id) = '')")
	legacyPaymentQuery = applyDivisionFilter(legacyPaymentQuery, division)
	legacyPaymentQuery.Select("COALESCE(SUM(amount_bhd), 0)").Scan(&legacyCustomerPayments)
	report.CashBHD = roundBHD(report.CashBHD + legacyCustomerPayments)

	supplierQuery := a.db.Model(&SupplierInvoice{}).
		Where("invoice_date <= ? AND LOWER(payment_status) <> ?", asOf, "paid")
	supplierQuery = applyDivisionFilter(supplierQuery, division)
	supplierQuery.Select("COALESCE(SUM(total_bhd), 0)").Scan(&report.AccountsPayableBHD)

	expenseQuery := a.db.Model(&ExpenseEntry{}).
		Where("expense_date <= ? AND LOWER(status) IN ? AND LOWER(payment_status) <> ?", asOf, []string{"approved", "posted"}, "paid")
	expenseQuery = applyDivisionFilter(expenseQuery, division)
	expenseQuery.Select("COALESCE(SUM(total_amount), 0)").Scan(&report.ExpenseLiabilityBHD)

	var supplierPaid float64
	supplierPaymentQuery := a.db.Model(&SupplierPayment{}).Where("payment_date <= ?", asOf)
	supplierPaymentQuery = applyDivisionFilter(supplierPaymentQuery, division)
	supplierPaymentQuery.Select("COALESCE(SUM(amount_bhd), 0)").Scan(&supplierPaid)

	var paidExpenses float64
	paidExpenseQuery := a.db.Model(&ExpenseEntry{}).Where("paid_at <= ? AND LOWER(payment_status) = ?", asOf, "paid")
	paidExpenseQuery = applyDivisionFilter(paidExpenseQuery, division)
	paidExpenseQuery.Select("COALESCE(SUM(total_amount), 0)").Scan(&paidExpenses)

	report.CashBHD = roundBHD(report.CashBHD - supplierPaid - paidExpenses)
	report.AccountsReceivableBHD = roundBHD(report.AccountsReceivableBHD)
	report.CustomerCreditsBHD = roundBHD(report.CustomerCreditsBHD)
	report.AccountsPayableBHD = roundBHD(report.AccountsPayableBHD)
	report.ExpenseLiabilityBHD = roundBHD(report.ExpenseLiabilityBHD)
	report.NetPositionBHD = roundBHD(report.CashBHD + report.AccountsReceivableBHD - report.CustomerCreditsBHD - report.AccountsPayableBHD - report.ExpenseLiabilityBHD)
	return report, nil
}

func operationalDateRange(startRaw, endRaw string) (time.Time, time.Time) {
	now := time.Now()
	start := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	if parsed, ok := parseOperationalDate(startRaw, false); ok {
		start = parsed
	}
	if parsed, ok := parseOperationalDate(endRaw, true); ok {
		end = parsed
	}
	if end.Before(start) {
		return end, start
	}
	return start, end
}

func parseOperationalDate(raw string, endOfDay bool) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		parsed = parseDate(raw)
		if parsed.IsZero() {
			return time.Time{}, false
		}
	}
	if endOfDay {
		parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 0, parsed.Location())
	}
	return parsed, true
}

func formatOperationalDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func operationalInvoiceStatuses() []string {
	return []string{"Sent", "Paid", "PartiallyPaid", "Overdue"}
}

func operationalSourceIncluded(filterSource, source string) bool {
	filterSource = strings.ToUpper(strings.TrimSpace(filterSource))
	return filterSource == "" || filterSource == strings.ToUpper(source)
}

// applyDivisionFilter scopes a single-table query to a division. PH hardcoded
// 'PH Trading' as both the normaliser default and the COALESCE fallback; on the
// sovereign substrate the division set + default are OVERLAY CONFIG (default
// "Acme Instrumentation"), so we reuse the config-driven normalizeDivisionSQL
// CASE — the same expression the dashboard/backfill queries use. This maps a
// row's raw/blank/aliased division to its canonical key (blank → the overlay
// default) so blank-division rows scope to the default division instead of
// silently vanishing under a stale hardcoded literal.
//
// normalizeDivisionName("") returns the overlay default key (never ""), so the
// empty-request path still scopes to the default division, matching PH intent.
func applyDivisionFilter(query *gorm.DB, division string) *gorm.DB {
	normalized := normalizeDivisionName(division)
	if normalized == "" {
		return query
	}
	return query.Where(normalizeDivisionSQL("division")+" = ?", normalized)
}
