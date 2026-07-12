package main

import (
	"log"
	"sort"
	"strings"
	"time"
)

// ============================================================================
// P2 FINANCE REPORTING SERVICE (Medium Priority - 2026-01-27)
// ============================================================================
// This service provides advanced finance reporting capabilities:
// 1. Payment Aging Report - AR aging with customer-level breakdown
// 2. Cash Flow Projection - Forecasting based on due dates and payment history
// 3. Profit Margin Analysis - Customer and product profitability
// 4. VAT Reconciliation - Output VAT vs Input VAT calculation
// 5. Financial Period Close - Prevent edits to closed periods
// ============================================================================

// PaymentAgingBucket represents aging analysis for a customer
type PaymentAgingBucket struct {
	CustomerID       string  `json:"customer_id"`
	CustomerName     string  `json:"customer_name"`
	Current          float64 `json:"current"`       // 0-30 days
	Days1To30        float64 `json:"days_1_to_30"`  // 1-30 days overdue
	Days31To60       float64 `json:"days_31_to_60"` // 31-60 days overdue
	Days61To90       float64 `json:"days_61_to_90"` // 61-90 days overdue
	Over90Days       float64 `json:"over_90_days"`  // 90+ days overdue
	TotalOutstanding float64 `json:"total_outstanding"`
	AvgDaysOverdue   float64 `json:"avg_days_overdue"` // Weighted average
}

// PaymentAgingReport represents complete aging report
type PaymentAgingReport struct {
	ReportDate      time.Time            `json:"report_date"`
	TotalCurrent    float64              `json:"total_current"`
	TotalDays1To30  float64              `json:"total_days_1_to_30"`
	TotalDays31To60 float64              `json:"total_days_31_to_60"`
	TotalDays61To90 float64              `json:"total_days_61_to_90"`
	TotalOver90Days float64              `json:"total_over_90_days"`
	GrandTotal      float64              `json:"grand_total"`
	AvgDaysOverdue  float64              `json:"avg_days_overdue"`
	CustomerBuckets []PaymentAgingBucket `json:"customer_buckets"`
}

// agingBucketForDueDate returns the canonical AR aging bucket key for an invoice
// due date relative to `now`. It is the single source of truth for AR bucketing,
// shared by GetPaymentAgingReport (customer-level aggregate) and
// GetInvoicesByAgingBucket (invoice-level drill-through) so the two can never
// drift. Boundaries (days overdue, DueDate-based):
//
//	<= 0  -> "current"   (not yet due)
//	1..30 -> "1_30"
//	31..60 -> "31_60"
//	61..90 -> "61_90"
//	> 90  -> "over_90"
func agingBucketForDueDate(due time.Time, now time.Time) string {
	daysOverdue := int(now.Sub(due).Hours() / 24)
	switch {
	case daysOverdue <= 0:
		return "current"
	case daysOverdue <= 30:
		return "1_30"
	case daysOverdue <= 60:
		return "31_60"
	case daysOverdue <= 90:
		return "61_90"
	default:
		return "over_90"
	}
}

// GetPaymentAgingReport generates AR aging report with customer-level breakdown
func (a *App) GetPaymentAgingReport() (PaymentAgingReport, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return PaymentAgingReport{}, err
	}
	if a.db == nil {
		return PaymentAgingReport{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	report := PaymentAgingReport{
		ReportDate:      time.Now(),
		CustomerBuckets: []PaymentAgingBucket{},
	}

	// Get all open-balance invoices and normalize collectible/open state in-memory.
	// OSS invoices are payment-driven: outstanding_bhd and status are derived, so
	// aging must reflect the settlement policy (customer_invoice_payment_policy.go)
	// rather than the stored status column (Mission I Band-0 convention).
	var invoices []Invoice
	if err := a.db.Where("outstanding_bhd > 0").Find(&invoices).Error; err != nil {
		return report, newError("DB_QUERY_FAILED", "Failed to retrieve invoices", err.Error())
	}

	// Group by customer
	customerMap := make(map[string]*PaymentAgingBucket)
	now := time.Now()

	for _, inv := range invoices {
		state := customerInvoicePaymentStateFromInvoice(inv, now)
		if !state.IsCollectible {
			continue
		}

		// Calculate days overdue (negative = not yet due)
		daysOverdue := int(now.Sub(inv.DueDate).Hours() / 24)
		outstanding := state.OutstandingBHD

		// Get or create customer bucket
		bucket, exists := customerMap[inv.CustomerID]
		if !exists {
			bucket = &PaymentAgingBucket{
				CustomerID:   inv.CustomerID,
				CustomerName: inv.CustomerName,
			}
			customerMap[inv.CustomerID] = bucket
		}

		// Categorize into aging buckets (shared helper — keeps this aggregate and
		// the GetInvoicesByAgingBucket drill-through using identical boundaries).
		switch agingBucketForDueDate(inv.DueDate, now) {
		case "current":
			bucket.Current += outstanding
		case "1_30":
			bucket.Days1To30 += outstanding
		case "31_60":
			bucket.Days31To60 += outstanding
		case "61_90":
			bucket.Days61To90 += outstanding
		default:
			bucket.Over90Days += outstanding
		}

		bucket.TotalOutstanding += outstanding

		// Calculate weighted average days overdue
		if daysOverdue > 0 {
			bucket.AvgDaysOverdue += float64(daysOverdue) * outstanding
		}
	}

	// Finalize buckets and calculate totals
	var totalWeightedDays float64
	for _, bucket := range customerMap {
		// Finalize weighted average
		if bucket.TotalOutstanding > 0 {
			bucket.AvgDaysOverdue = bucket.AvgDaysOverdue / bucket.TotalOutstanding
		}

		// Add to report totals
		report.TotalCurrent += bucket.Current
		report.TotalDays1To30 += bucket.Days1To30
		report.TotalDays31To60 += bucket.Days31To60
		report.TotalDays61To90 += bucket.Days61To90
		report.TotalOver90Days += bucket.Over90Days
		report.GrandTotal += bucket.TotalOutstanding

		totalWeightedDays += bucket.AvgDaysOverdue * bucket.TotalOutstanding

		report.CustomerBuckets = append(report.CustomerBuckets, *bucket)
	}

	// Calculate global weighted average
	if report.GrandTotal > 0 {
		report.AvgDaysOverdue = totalWeightedDays / report.GrandTotal
	}

	// Sort by total outstanding (descending)
	sort.Slice(report.CustomerBuckets, func(i, j int) bool {
		return report.CustomerBuckets[i].TotalOutstanding > report.CustomerBuckets[j].TotalOutstanding
	})

	log.Printf("💳 Payment Aging Report: %.3f BHD outstanding, %.1f avg days overdue, %d customers",
		report.GrandTotal, report.AvgDaysOverdue, len(report.CustomerBuckets))

	return report, nil
}

// AgingBucketInvoices is the paginated result of an aging-bucket drill-through.
type AgingBucketInvoices struct {
	Bucket   string    `json:"bucket"`
	Total    int       `json:"total"`     // total invoices in the bucket (pre-pagination)
	Limit    int       `json:"limit"`     // effective page size applied
	Offset   int       `json:"offset"`    // effective offset applied
	TotalBHD float64   `json:"total_bhd"` // sum of outstanding across the WHOLE bucket
	Invoices []Invoice `json:"invoices"`  // this page of invoices
}

// GetInvoicesByAgingBucket returns the individual open invoices that fall into a
// given AR aging bucket ("all","current","1_30","31_60","61_90","over_90"),
// paginated. Buckets are computed from due_date through the shared
// agingBucketForDueDate helper and collectibility through the same
// customerInvoicePaymentStateFromInvoice normalization GetPaymentAgingReport
// uses — so this drill-through list always agrees with the aggregate report's
// totals. Unlike ListCustomerInvoices (capped at 200) it scans the full set of
// open invoices, so the result is complete beyond that cap. (I-23)
func (a *App) GetInvoicesByAgingBucket(bucket string, limit, offset int) (AgingBucketInvoices, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return AgingBucketInvoices{}, err
	}
	if a.db == nil {
		return AgingBucketInvoices{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	bucket = strings.ToLower(strings.TrimSpace(bucket))
	if bucket == "" {
		bucket = "all"
	}
	// "0_30" is a composite (current + 1_30) so the Financial Dashboard's single
	// "0-30d" receivables card can drill to one accurate due_date-based list.
	validBuckets := map[string]bool{
		"all": true, "current": true, "1_30": true, "0_30": true,
		"31_60": true, "61_90": true, "over_90": true,
	}
	if !validBuckets[bucket] {
		return AgingBucketInvoices{}, newError("INVALID_BUCKET",
			"Invalid aging bucket: "+bucket, "valid: all, current, 1_30, 0_30, 31_60, 61_90, over_90")
	}
	bucketMatches := func(actual string) bool {
		switch bucket {
		case "all":
			return true
		case "0_30":
			return actual == "current" || actual == "1_30"
		default:
			return actual == bucket
		}
	}

	// Pagination bounds (mirror sibling finance listing caps).
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	// Only open-balance invoices participate in AR aging.
	var candidates []Invoice
	if err := a.db.
		Where("outstanding_bhd > 0").
		Order("due_date ASC").
		Find(&candidates).Error; err != nil {
		return AgingBucketInvoices{}, newError("DB_QUERY_FAILED", "Failed to retrieve invoices", err.Error())
	}

	now := time.Now()
	matched := make([]Invoice, 0, len(candidates))
	var bucketTotalBHD float64
	for _, inv := range candidates {
		state := customerInvoicePaymentStateFromInvoice(inv, now)
		if !state.IsCollectible {
			continue
		}
		if !bucketMatches(agingBucketForDueDate(inv.DueDate, now)) {
			continue
		}
		// Reflect the normalized collectible outstanding/status so the page totals
		// match the aggregate report exactly.
		inv.OutstandingBHD = state.OutstandingBHD
		inv.Status = state.Status
		matched = append(matched, inv)
		bucketTotalBHD += state.OutstandingBHD
	}

	result := AgingBucketInvoices{
		Bucket:   bucket,
		Total:    len(matched),
		Limit:    limit,
		Offset:   offset,
		TotalBHD: roundTo3(bucketTotalBHD),
		Invoices: []Invoice{},
	}

	if offset >= len(matched) {
		return result, nil
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}
	if end < len(matched) {
		log.Printf("📄 GetInvoicesByAgingBucket(%s): page %d-%d of %d matched (limit=%d)",
			bucket, offset, end, len(matched), limit)
	}
	result.Invoices = matched[offset:end]
	return result, nil
}

// CashFlowProjectionDay represents projected cash for a single day
type CashFlowProjectionDay struct {
	Date             time.Time `json:"date"`
	ExpectedInflows  float64   `json:"expected_inflows"`  // Customer payments
	ExpectedOutflows float64   `json:"expected_outflows"` // Supplier payments
	NetCashFlow      float64   `json:"net_cash_flow"`
	CumulativeCash   float64   `json:"cumulative_cash"`
	OpeningBalance   float64   `json:"opening_balance"`
	ClosingBalance   float64   `json:"closing_balance"`
}

// CashFlowProjection represents cash flow forecast
type CashFlowProjection struct {
	StartDate        time.Time               `json:"start_date"`
	EndDate          time.Time               `json:"end_date"`
	OpeningCash      float64                 `json:"opening_cash"`
	TotalInflows     float64                 `json:"total_inflows"`
	TotalOutflows    float64                 `json:"total_outflows"`
	ProjectedCash    float64                 `json:"projected_cash"`
	DailyProjections []CashFlowProjectionDay `json:"daily_projections"`
}

// GetCashFlowProjection generates cash flow forecast based on due dates and payment history
func (a *App) GetCashFlowProjection(days int) (CashFlowProjection, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return CashFlowProjection{}, err
	}
	if a.db == nil {
		return CashFlowProjection{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Clamp days to a reasonable range to prevent memory exhaustion
	if days < 1 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	now := time.Now()
	endDate := now.AddDate(0, 0, days)

	// Query current bank account balances for opening cash
	var openingCash float64
	var bankAccounts []BankAccount
	if err := a.db.Find(&bankAccounts).Error; err == nil {
		for _, account := range bankAccounts {
			if account.Currency == "BHD" {
				openingCash += account.CurrentBalance
			}
			// NOTE: For non-BHD accounts, should convert using exchange rates.
			// Currently summing BHD accounts only for simplicity.
		}
	}

	projection := CashFlowProjection{
		StartDate:        now,
		EndDate:          endDate,
		OpeningCash:      openingCash,
		DailyProjections: []CashFlowProjectionDay{},
	}

	// Create daily buckets
	dailyMap := make(map[string]*CashFlowProjectionDay)
	for d := 0; d <= days; d++ {
		date := now.AddDate(0, 0, d)
		dateKey := date.Format("2006-01-02")
		dailyMap[dateKey] = &CashFlowProjectionDay{
			Date: date,
		}
	}

	// 1. Project INFLOWS from customer invoices (based on due dates + payment history)
	// Only include collectible statuses — exclude Cancelled, Void, Proforma, Draft
	collectibleStatuses := []string{"Sent", "Overdue", "PartiallyPaid"}
	var unpaidInvoices []Invoice
	if err := a.db.Where("status IN ? AND due_date BETWEEN ? AND ?", collectibleStatuses, now, endDate).
		Find(&unpaidInvoices).Error; err != nil {
		return projection, newError("DB_QUERY_FAILED", "Failed to retrieve invoices", err.Error())
	}

	for _, inv := range unpaidInvoices {
		// Skip if already fully collected (status may lag a payment update)
		if inv.OutstandingBHD <= 0 {
			continue
		}

		// Get customer's average payment delay
		var customer CustomerMaster
		a.db.First(&customer, "id = ?", inv.CustomerID)

		// Adjust due date by average payment days
		expectedDate := inv.DueDate
		if customer.AvgPaymentDays > 0 {
			expectedDate = inv.DueDate.AddDate(0, 0, int(customer.AvgPaymentDays))
		}

		// Check if expected date is within projection window
		if expectedDate.After(now) && expectedDate.Before(endDate) {
			dateKey := expectedDate.Format("2006-01-02")
			if day, exists := dailyMap[dateKey]; exists {
				day.ExpectedInflows += inv.OutstandingBHD
				projection.TotalInflows += inv.OutstandingBHD
			}
		}
	}

	// 2. Project OUTFLOWS from supplier invoices (based on due dates)
	// Exclude Pending invoices — they are unverified and may be rejected, so they
	// should not inflate the projected outflow commitment.
	var unpaidSupplierInvoices []SupplierInvoice
	if err := a.db.Where("payment_status != ? AND (status IS NULL OR status NOT IN ?) AND due_date BETWEEN ? AND ?",
		"Paid", []string{"Pending", "Rejected", "Dispute"}, now, endDate).
		Find(&unpaidSupplierInvoices).Error; err != nil {
		log.Printf("⚠️ Failed to retrieve supplier invoices: %v", err)
		// Continue with customer inflows only
	} else {
		for _, inv := range unpaidSupplierInvoices {
			dateKey := inv.DueDate.Format("2006-01-02")
			if day, exists := dailyMap[dateKey]; exists {
				day.ExpectedOutflows += inv.TotalBHD
				projection.TotalOutflows += inv.TotalBHD
			}
		}
	}

	// 2b. Project OUTFLOWS from approved/posted expense entries
	var unpaidExpenses []ExpenseEntry
	if err := a.db.Where(
		"status IN ? AND payment_status != ? AND ((due_date BETWEEN ? AND ?) OR (due_date IS NULL AND expense_date BETWEEN ? AND ?))",
		[]string{"approved", "posted"},
		"paid",
		now, endDate,
		now, endDate,
	).Find(&unpaidExpenses).Error; err != nil {
		log.Printf("⚠️ Failed to retrieve expense entries for cashflow: %v", err)
	} else {
		for _, expense := range unpaidExpenses {
			postDate := expense.ExpenseDate
			if expense.DueDate != nil {
				postDate = *expense.DueDate
			}
			dateKey := postDate.Format("2006-01-02")
			if day, exists := dailyMap[dateKey]; exists {
				day.ExpectedOutflows += expense.TotalAmount
				projection.TotalOutflows += expense.TotalAmount
			}
		}
	}

	// 2c. Project OUTFLOWS from recurring expenses not yet generated in the window
	var recurringExpenses []RecurringExpense
	if err := a.db.Where("is_active = ? AND next_run_date <= ?", true, endDate).Find(&recurringExpenses).Error; err != nil {
		log.Printf("⚠️ Failed to retrieve recurring expenses for cashflow: %v", err)
	} else {
		for _, recurring := range recurringExpenses {
			for runAt := recurring.NextRunDate; !runAt.After(endDate); runAt = nextRecurringDate(runAt, recurring.Frequency, recurring.IntervalValue) {
				if runAt.Before(now) {
					continue
				}
				dateKey := runAt.Format("2006-01-02")
				if day, exists := dailyMap[dateKey]; exists {
					amount := recurring.DefaultAmount + recurring.DefaultVATAmount
					day.ExpectedOutflows += amount
					projection.TotalOutflows += amount
				}
			}
		}
	}

	// 2d. Project OUTFLOWS from approved/posted payroll liabilities
	var payrollRuns []PayrollRun
	if err := a.db.Where("status IN ?", []string{"approved", "posted"}).Find(&payrollRuns).Error; err != nil {
		log.Printf("⚠️ Failed to retrieve payroll runs for cashflow: %v", err)
	} else if len(payrollRuns) > 0 {
		periodIDs := make([]string, 0, len(payrollRuns))
		for _, run := range payrollRuns {
			if strings.TrimSpace(run.PayrollPeriodID) != "" {
				periodIDs = append(periodIDs, run.PayrollPeriodID)
			}
		}

		periodMap := map[string]PayrollPeriod{}
		var payrollPeriods []PayrollPeriod
		if len(periodIDs) > 0 && a.db.Where("id IN ?", periodIDs).Find(&payrollPeriods).Error == nil {
			for _, period := range payrollPeriods {
				periodMap[period.ID] = period
			}
		}

		for _, run := range payrollRuns {
			period, ok := periodMap[run.PayrollPeriodID]
			if !ok {
				continue
			}
			dueOn := period.PeriodEnd
			if period.PaymentDate != nil {
				dueOn = *period.PaymentDate
			}
			if dueOn.Before(now) || dueOn.After(endDate) {
				continue
			}
			dateKey := dueOn.Format("2006-01-02")
			if day, exists := dailyMap[dateKey]; exists {
				liability := run.NetTotal + run.DeductionsTotal + run.EmployerCostTotal
				day.ExpectedOutflows += liability
				projection.TotalOutflows += liability
			}
		}
	}

	// 3. Calculate cumulative cash
	cumulativeCash := projection.OpeningCash
	for d := 0; d <= days; d++ {
		date := now.AddDate(0, 0, d)
		dateKey := date.Format("2006-01-02")
		day := dailyMap[dateKey]

		day.OpeningBalance = cumulativeCash
		day.NetCashFlow = day.ExpectedInflows - day.ExpectedOutflows
		cumulativeCash += day.NetCashFlow
		day.ClosingBalance = cumulativeCash
		day.CumulativeCash = cumulativeCash

		projection.DailyProjections = append(projection.DailyProjections, *day)
	}

	projection.ProjectedCash = cumulativeCash

	log.Printf("💰 Cash Flow Projection (%d days): +%.3f BHD inflows, -%.3f BHD outflows, %.3f BHD projected",
		days, projection.TotalInflows, projection.TotalOutflows, projection.ProjectedCash)

	return projection, nil
}

// MarginAnalysisByCustomer represents margin breakdown per customer
type MarginAnalysisByCustomer struct {
	CustomerID    string  `json:"customer_id"`
	CustomerName  string  `json:"customer_name"`
	TotalRevenue  float64 `json:"total_revenue"`
	TotalCost     float64 `json:"total_cost"`
	GrossMargin   float64 `json:"gross_margin"`
	MarginPercent float64 `json:"margin_percent"`
	OrderCount    int     `json:"order_count"`
	AvgMarginPct  float64 `json:"avg_margin_pct"`
}

// MarginAnalysisByProduct represents margin breakdown per product category
type MarginAnalysisByProduct struct {
	ProductCategory string  `json:"product_category"`
	TotalRevenue    float64 `json:"total_revenue"`
	TotalCost       float64 `json:"total_cost"`
	GrossMargin     float64 `json:"gross_margin"`
	MarginPercent   float64 `json:"margin_percent"`
	OrderCount      int     `json:"order_count"`
}

// GetMarginAnalysisByCustomer returns profitability per customer
func (a *App) GetMarginAnalysisByCustomer() ([]MarginAnalysisByCustomer, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Only include finalized invoices — Cancelled and Void invoices never generated real margin.
	// Including them inflates per-customer revenue and distorts profitability rankings.
	finalizedStatuses := []string{"Sent", "Overdue", "PartiallyPaid", "Paid"}
	var invoices []Invoice
	if err := a.db.Preload("Items").Where("status IN ?", finalizedStatuses).Find(&invoices).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve invoices", err.Error())
	}

	// Group by customer
	customerMap := make(map[string]*MarginAnalysisByCustomer)

	for _, inv := range invoices {
		analysis, exists := customerMap[inv.CustomerID]
		if !exists {
			analysis = &MarginAnalysisByCustomer{
				CustomerID:   inv.CustomerID,
				CustomerName: inv.CustomerName,
			}
			customerMap[inv.CustomerID] = analysis
		}

		analysis.TotalRevenue += inv.GrandTotalBHD
		analysis.TotalCost += inv.TotalSupplierCostBHD
		analysis.OrderCount++
	}

	// Calculate margins
	results := []MarginAnalysisByCustomer{}
	for _, analysis := range customerMap {
		analysis.GrossMargin = analysis.TotalRevenue - analysis.TotalCost
		if analysis.TotalRevenue > 0 {
			analysis.MarginPercent = (analysis.GrossMargin / analysis.TotalRevenue) * 100
			analysis.AvgMarginPct = analysis.MarginPercent
		}
		results = append(results, *analysis)
	}

	// Sort by margin (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].GrossMargin > results[j].GrossMargin
	})

	log.Printf("📊 Margin Analysis by Customer: %d customers analyzed", len(results))

	return results, nil
}

// GetMarginAnalysisByProduct returns profitability per product category
func (a *App) GetMarginAnalysisByProduct() ([]MarginAnalysisByProduct, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var items []DBInvoiceItem
	if err := a.db.Find(&items).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve invoice items", err.Error())
	}

	// Get product details for category mapping
	productMap := make(map[string]string) // productID -> category
	var products []ProductMaster
	a.db.Find(&products)
	for _, p := range products {
		productMap[p.ID] = p.ProductCategory
	}

	// Group by product category
	categoryMap := make(map[string]*MarginAnalysisByProduct)

	for _, item := range items {
		category := productMap[item.ProductID]
		if category == "" {
			category = "Uncategorized"
		}

		analysis, exists := categoryMap[category]
		if !exists {
			analysis = &MarginAnalysisByProduct{
				ProductCategory: category,
			}
			categoryMap[category] = analysis
		}

		revenue := item.TotalPrice
		cost := item.TotalCost * item.Quantity

		analysis.TotalRevenue += revenue
		analysis.TotalCost += cost
		analysis.OrderCount++
	}

	// Calculate margins
	results := []MarginAnalysisByProduct{}
	for _, analysis := range categoryMap {
		analysis.GrossMargin = analysis.TotalRevenue - analysis.TotalCost
		if analysis.TotalRevenue > 0 {
			analysis.MarginPercent = (analysis.GrossMargin / analysis.TotalRevenue) * 100
		}
		results = append(results, *analysis)
	}

	// Sort by margin (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].GrossMargin > results[j].GrossMargin
	})

	log.Printf("📊 Margin Analysis by Product: %d categories analyzed", len(results))

	return results, nil
}

// VATReconciliation represents VAT summary for a period
type VATReconciliation struct {
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	OutputVAT        float64   `json:"output_vat"`        // VAT collected from customers
	InputVAT         float64   `json:"input_vat"`         // VAT paid to suppliers
	NetVAT           float64   `json:"net_vat"`           // Payable (positive) or Receivable (negative)
	CustomerInvoices int       `json:"customer_invoices"` // Count of customer invoices
	SupplierInvoices int       `json:"supplier_invoices"` // Count of supplier invoices
}

// GetVATReconciliation calculates VAT payable/receivable for a date range
func (a *App) GetVATReconciliation(startDateStr, endDateStr string) (VATReconciliation, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return VATReconciliation{}, err
	}
	if a.db == nil {
		return VATReconciliation{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return VATReconciliation{}, newError("INVALID_DATE", "Invalid start date format, expected YYYY-MM-DD", err.Error())
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return VATReconciliation{}, newError("INVALID_DATE", "Invalid end date format, expected YYYY-MM-DD", err.Error())
	}

	reconciliation := VATReconciliation{
		StartDate: startDate,
		EndDate:   endDate,
	}

	// 1. Sum OUTPUT VAT from customer invoices (VAT collected)
	// Exclude Cancelled, Void, Proforma, Draft — only real invoices count for VAT
	excludedStatuses := []string{"Cancelled", "Void", "Proforma", "Draft"}
	var customerInvoices []Invoice
	if err := a.db.Where("invoice_date BETWEEN ? AND ? AND status NOT IN ?", startDate, endDate, excludedStatuses).
		Find(&customerInvoices).Error; err != nil {
		return reconciliation, newError("DB_QUERY_FAILED", "Failed to retrieve customer invoices", err.Error())
	}

	for _, inv := range customerInvoices {
		reconciliation.OutputVAT += inv.VATBHD
	}
	reconciliation.CustomerInvoices = len(customerInvoices)

	// 2. Sum INPUT VAT from supplier invoices (VAT paid)
	// Exclude Rejected, Dispute, and Pending — Pending invoices are unverified and VAT is not yet claimable.
	supplierExcludedStatuses := []string{"Rejected", "Dispute", "Pending"}
	var supplierInvoices []SupplierInvoice
	if err := a.db.Where("invoice_date BETWEEN ? AND ? AND (status IS NULL OR status NOT IN ?)", startDate, endDate, supplierExcludedStatuses).
		Find(&supplierInvoices).Error; err != nil {
		log.Printf("⚠️ Failed to retrieve supplier invoices: %v", err)
		// Continue with output VAT only
	} else {
		for _, inv := range supplierInvoices {
			reconciliation.InputVAT += inv.VATBHD
		}
		reconciliation.SupplierInvoices = len(supplierInvoices)
	}

	// 3. Calculate net VAT (positive = payable, negative = receivable)
	reconciliation.NetVAT = reconciliation.OutputVAT - reconciliation.InputVAT

	log.Printf("💼 VAT Reconciliation (%s to %s): Output %.3f BHD - Input %.3f BHD = Net %.3f BHD",
		startDateStr, endDateStr, reconciliation.OutputVAT, reconciliation.InputVAT, reconciliation.NetVAT)

	return reconciliation, nil
}

// ClosePeriod closes a fiscal period to prevent edits
func (a *App) ClosePeriod(periodID string) error {
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Check permission (requires finance:manage permission)
	// Note: Add PERM_CLOSE_PERIOD = "finance:close_period" to app.go permission constants if needed
	const PERM_CLOSE_PERIOD = "finance:manage"
	if err := a.requirePermission(PERM_CLOSE_PERIOD); err != nil {
		return err
	}

	// Get period
	var period FiscalPeriod
	if err := a.db.First(&period, "id = ?", periodID).Error; err != nil {
		return newError("PERIOD_NOT_FOUND", "Fiscal period not found", err.Error())
	}

	// Check if already closed
	if period.Status == "Closed" || period.Status == "Locked" {
		return newError("PERIOD_ALREADY_CLOSED", "Period is already closed", "")
	}

	// Update status
	now := time.Now()
	period.Status = "Closed"
	period.ClosedAt = &now
	period.ClosedBy = a.getCurrentUserID()

	if err := a.db.Save(&period).Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to close period", err.Error())
	}

	log.Printf("🔒 Period closed: %d/%d by %s", period.FiscalYear, period.Period, period.ClosedBy)

	return nil
}

// IsPeriodClosed checks if a date falls within a closed period
func (a *App) IsPeriodClosed(dateStr string) (bool, error) {
	if a.db == nil {
		return false, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Parse date
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return false, newError("INVALID_DATE", "Invalid date format, expected YYYY-MM-DD", err.Error())
	}

	// Find period containing this date
	var period FiscalPeriod
	if err := a.db.Where("period_start <= ? AND period_end >= ?", date, date).
		First(&period).Error; err != nil {
		// No period found = not closed
		return false, nil
	}

	// Check if period is closed or locked
	isClosed := period.Status == "Closed" || period.Status == "Locked"

	return isClosed, nil
}
