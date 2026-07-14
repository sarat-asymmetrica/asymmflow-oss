package main

import (
	"fmt"
	"log"
	"math"
	"time"
)

var operationalFreshStartDate = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func usesOperationalFreshStart(year int) bool {
	return !operationalFreshStartDate.IsZero() && year >= operationalFreshStartDate.Year()
}

func operationalMetricStartForYear(year int) time.Time {
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	if usesOperationalFreshStart(year) && operationalFreshStartDate.After(yearStart) {
		return operationalFreshStartDate
	}
	return yearStart
}

// =============================================================================
// FINANCIAL YEAR SERVICE - Dynamic Finance with Demo Data
// =============================================================================
//
// Data Sources:
// - 2023: fabricated demo P&L (clearly non-real reference figures)
// - 2024: fabricated demo P&L (clearly non-real reference figures)
// - 2025: Live calculation from imported transactions (unaudited)
//
// All figures below are synthetic placeholders for the open-source demo build.
// =============================================================================

// FinancialYearSummary stores comprehensive financial data for a fiscal year
type FinancialYearSummary struct {
	Year     int    `json:"year"`
	IsAudit  bool   `json:"is_audited"`
	Source   string `json:"source"` // "Demo financial data", "Live Import"
	AsOfDate string `json:"as_of_date"`

	// Income Statement
	Revenue       float64 `json:"revenue"`
	CostOfSales   float64 `json:"cost_of_sales"`
	GrossProfit   float64 `json:"gross_profit"`
	OtherIncome   float64 `json:"other_income"`
	StaffCosts    float64 `json:"staff_costs"`
	AdminExpenses float64 `json:"admin_expenses"`
	Depreciation  float64 `json:"depreciation"`
	FinanceCosts  float64 `json:"finance_costs"`
	NetProfit     float64 `json:"net_profit"`

	// Balance Sheet - Assets
	PlantEquipment   float64 `json:"plant_equipment"`
	RightOfUse       float64 `json:"right_of_use"`
	InvestmentProp   float64 `json:"investment_property"`
	Inventories      float64 `json:"inventories"`
	TradeReceivables float64 `json:"trade_receivables"`
	FixedDeposits    float64 `json:"fixed_deposits"`
	RelatedPartyRecv float64 `json:"related_party_receivable"`
	CashEquivalents  float64 `json:"cash_equivalents"`
	TotalAssets      float64 `json:"total_assets"`
	CurrentAssets    float64 `json:"current_assets"`
	NonCurrentAssets float64 `json:"non_current_assets"`

	// Balance Sheet - Liabilities & Equity
	ShareCapital       float64 `json:"share_capital"`
	StatutoryReserve   float64 `json:"statutory_reserve"`
	OwnerCurrentAcct   float64 `json:"owner_current_account"`
	TotalEquity        float64 `json:"total_equity"`
	LeaseNonCurrent    float64 `json:"lease_non_current"`
	LeaseCurrent       float64 `json:"lease_current"`
	TradePayables      float64 `json:"trade_payables"`
	TotalLiabilities   float64 `json:"total_liabilities"`
	CurrentLiabilities float64 `json:"current_liabilities"`
}

// getVerifiedFS2024Data returns fabricated demo financial data for the OSS build.
// These are synthetic placeholder figures only — not real audited financials.
// Targets: 22% gross margin, ~2.10x current ratio.
func getVerifiedFS2024Data() map[int]FinancialYearSummary {
	return map[int]FinancialYearSummary{
		// FY2023 - demo comparative figures
		2023: {
			Year:     2023,
			IsAudit:  true,
			Source:   "Demo financial data (Comparative)",
			AsOfDate: "2023-12-31",

			// Income Statement - demo figures
			Revenue:       3100000.0, // demo: 3.10M BHD
			CostOfSales:   2418000.0, // 78% of revenue (22% gross margin)
			GrossProfit:   682000.0,  // 3,100,000 - 2,418,000
			OtherIncome:   30000.0,
			StaffCosts:    300000.0,
			AdminExpenses: 110000.0,
			Depreciation:  22000.0,
			FinanceCosts:  30000.0,
			NetProfit:     250000.0, // demo: 250K BHD

			// Balance Sheet - demo figures (current ratio ~2.10x)
			PlantEquipment:   45000.0,
			RightOfUse:       50000.0,
			InvestmentProp:   420000.0,
			Inventories:      8000.0,
			TradeReceivables: 117000.0,
			FixedDeposits:    50000.0,
			RelatedPartyRecv: 35000.0,
			CashEquivalents:  443000.0,
			TotalAssets:      1168000.0,
			CurrentAssets:    653000.0,
			NonCurrentAssets: 515000.0,

			ShareCapital:       300000.0,
			StatutoryReserve:   143000.0,
			OwnerCurrentAcct:   414000.0,
			TotalEquity:        857000.0,
			LeaseNonCurrent:    0.0,
			LeaseCurrent:       11000.0,
			TradePayables:      300000.0,
			TotalLiabilities:   311000.0,
			CurrentLiabilities: 311000.0,
		},

		// FY2024 - demo current year figures
		2024: {
			Year:     2024,
			IsAudit:  true,
			Source:   "Demo financial data (Audited)",
			AsOfDate: "2024-12-31",

			// Income Statement - demo figures
			Revenue:       2400000.0, // demo: 2.40M BHD
			CostOfSales:   1872000.0, // 78% of revenue (22% gross margin)
			GrossProfit:   528000.0,  // 2,400,000 - 1,872,000
			OtherIncome:   5000.0,
			StaffCosts:    230000.0,
			AdminExpenses: 80000.0,
			Depreciation:  21000.0,
			FinanceCosts:  22000.0,
			NetProfit:     180000.0, // demo: 180K BHD

			// Balance Sheet - demo figures (current ratio ~2.10x)
			PlantEquipment:   40000.0,
			RightOfUse:       40000.0,
			InvestmentProp:   420000.0,
			Inventories:      5000.0,
			TradeReceivables: 123000.0,
			FixedDeposits:    50000.0,
			RelatedPartyRecv: 35000.0,
			CashEquivalents:  267000.0,
			TotalAssets:      980000.0,
			CurrentAssets:    480000.0,
			NonCurrentAssets: 500000.0,

			ShareCapital:       300000.0,
			StatutoryReserve:   145000.0,
			OwnerCurrentAcct:   306000.0,
			TotalEquity:        751000.0,
			LeaseNonCurrent:    0.0,
			LeaseCurrent:       9000.0,
			TradePayables:      220000.0,
			TotalLiabilities:   229000.0,
			CurrentLiabilities: 229000.0,
		},
	}
}

// GetFinancialYearData returns verified financial data for a specific year
func (a *App) GetFinancialYearData(year int) (*FinancialYearSummary, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	log.Printf("📊 Fetching financial data for year %d", year)

	// Check for audited data first (2023, 2024)
	verifiedData := getVerifiedFS2024Data()
	if data, ok := verifiedData[year]; ok {
		log.Printf("✅ Returning audited data from FS2024 for year %d", year)
		return &data, nil
	}

	// For other years, calculate from Tally imports
	log.Printf("📈 Calculating live data from Tally imports for year %d", year)
	return a.calculateFinancialYearFromTally(year)
}

// calculateFinancialYearFromTally computes financial metrics from imported data
func (a *App) calculateFinancialYearFromTally(year int) (*FinancialYearSummary, error) {
	summary := &FinancialYearSummary{
		Year:     year,
		IsAudit:  false,
		Source:   "Live Data (Unaudited)",
		AsOfDate: fmt.Sprintf("%d-12-31", year),
	}

	if usesOperationalFreshStart(year) {
		return a.calculateFreshStartFinancialYear(year, summary)
	}

	// Get revenue from customer invoices (using SQLite strftime)
	var totalRevenue float64
	err := a.db.Model(&Invoice{}).
		Where("strftime('%Y', invoice_date) = ? AND status NOT IN ('Cancelled', 'Void', 'Proforma', 'Draft')", fmt.Sprintf("%d", year)).
		Select("COALESCE(SUM(grand_total_bhd), 0)").
		Scan(&totalRevenue).Error
	if err != nil {
		log.Printf("⚠️ Error fetching invoice total for year %d: %v", year, err)
	}
	summary.Revenue = totalRevenue
	log.Printf("📊 Year %d revenue from invoices: %.2f BHD", year, totalRevenue)

	var bookedOrderValue float64
	err = a.db.Model(&Order{}).
		Where("strftime('%Y', order_date) = ? AND status NOT IN ?", fmt.Sprintf("%d", year), []string{"Cancelled", "Canceled", "Void"}).
		Select("COALESCE(SUM(CASE WHEN grand_total_bhd > 0 THEN grand_total_bhd ELSE total_value_bhd END), 0)").
		Scan(&bookedOrderValue).Error
	if err != nil {
		log.Printf("⚠️ Error fetching booked order value for year %d: %v", year, err)
	} else if bookedOrderValue > summary.Revenue {
		summary.Revenue = bookedOrderValue
		summary.Source = "Live Data (Unaudited - booked orders)"
		log.Printf("📊 Year %d using booked order value for current live revenue: %.2f BHD", year, bookedOrderValue)
	}

	// For COGS, we need to estimate based on historical gross margin
	// Raw supplier invoice totals != COGS (proper COGS = Opening Inv + Purchases - Closing Inv)
	// Without inventory tracking, we estimate COGS using 2024's gross margin percentage
	fs2024 := getVerifiedFS2024Data()[2024]
	grossMarginPct2024 := (fs2024.Revenue - fs2024.CostOfSales) / fs2024.Revenue // ~18.2%

	// Apply historical gross margin to estimate COGS
	// This is more accurate than using raw purchases which don't account for inventory changes
	summary.CostOfSales = summary.Revenue * (1 - grossMarginPct2024)
	summary.GrossProfit = summary.Revenue - summary.CostOfSales

	log.Printf("📊 Year %d estimated COGS: %.2f BHD (using %.1f%% gross margin from 2024)",
		year, summary.CostOfSales, grossMarginPct2024*100)

	// For unaudited years, estimate other values based on 2024 ratios
	// This is a reasonable approach until the year is audited
	if summary.Revenue > 0 {
		// fs2024 already loaded above for gross margin calculation

		// Apply 2024 ratios to estimate other items
		revenueRatio := summary.Revenue / fs2024.Revenue

		// Estimate operating expenses proportionally
		summary.StaffCosts = fs2024.StaffCosts * revenueRatio * 0.9 // Assume some economies
		summary.AdminExpenses = fs2024.AdminExpenses * revenueRatio
		summary.Depreciation = fs2024.Depreciation        // Fixed assets don't change much
		summary.FinanceCosts = fs2024.FinanceCosts * 0.95 // Slight decrease assumed
		summary.OtherIncome = fs2024.OtherIncome * revenueRatio

		// Estimate net profit (simplified)
		totalOpex := summary.StaffCosts + summary.AdminExpenses + summary.Depreciation + summary.FinanceCosts
		summary.NetProfit = summary.GrossProfit + summary.OtherIncome - totalOpex

		// Balance sheet estimates (carry forward from 2024 with adjustments)
		summary.PlantEquipment = fs2024.PlantEquipment - fs2024.Depreciation/2
		summary.RightOfUse = fs2024.RightOfUse - fs2024.Depreciation/2
		summary.InvestmentProp = fs2024.InvestmentProp
		summary.Inventories = fs2024.Inventories * revenueRatio
		summary.FixedDeposits = fs2024.FixedDeposits
		summary.ShareCapital = fs2024.ShareCapital
		summary.StatutoryReserve = fs2024.StatutoryReserve
	}

	// Calculate receivables from live invoice data (using SQLite strftime)
	// Invoice uses outstanding_bhd (pre-calculated outstanding amount)
	var outstandingAR float64
	err = a.db.Model(&Invoice{}).
		Where("strftime('%Y', invoice_date) = ? AND status NOT IN ('Paid', 'Cancelled', 'Void', 'Proforma', 'Draft')", fmt.Sprintf("%d", year)).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&outstandingAR).Error
	if err == nil && outstandingAR > 0 {
		summary.TradeReceivables = outstandingAR
		log.Printf("📊 Year %d AR from invoices: %.2f BHD", year, outstandingAR)
	} else {
		start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)
		orderExposure, pendingOrders, exposureErr := a.calculateUninvoicedOrderExposure(start, end)
		if exposureErr == nil && orderExposure > 0 {
			summary.TradeReceivables = orderExposure
			log.Printf("📊 Year %d AR fallback from %d uninvoiced confirmed orders: %.2f BHD", year, pendingOrders, orderExposure)
		} else {
			// Estimate from revenue
			summary.TradeReceivables = summary.Revenue * 0.08 // ~30 days DSO
		}
	}

	// Calculate payables from live supplier invoice data (using SQLite strftime)
	// SupplierInvoice uses total_bhd, status tracks payment state
	var outstandingAP float64
	err = a.db.Model(&SupplierInvoice{}).
		Where("strftime('%Y', invoice_date) = ? AND status NOT IN ('Paid')", fmt.Sprintf("%d", year)).
		Select("COALESCE(SUM(total_bhd), 0)").
		Scan(&outstandingAP).Error
	if err == nil && outstandingAP > 0 {
		summary.TradePayables = outstandingAP
		log.Printf("📊 Year %d AP from supplier invoices: %.2f BHD", year, outstandingAP)
	} else {
		// Estimate from purchases
		summary.TradePayables = summary.CostOfSales * 0.10 // ~35 days DPO
	}

	// Cash position - try to get from actual bank statements first
	summary.RelatedPartyRecv = fs2024.RelatedPartyRecv

	// Query actual bank balances from latest bank statements
	var totalBankBalance float64
	var bankBalanceFound bool
	type BalanceResult struct {
		Balance float64
	}
	var balResult BalanceResult
	err = a.db.Model(&BankStatement{}).
		Where("strftime('%Y', period_end) = ?", fmt.Sprintf("%d", year)).
		Select("COALESCE(SUM(closing_balance), 0) as balance").
		Scan(&balResult).Error
	if err == nil && balResult.Balance > 0 {
		totalBankBalance = balResult.Balance
		bankBalanceFound = true
		log.Printf("📊 Year %d cash from bank statements: %.3f BHD", year, totalBankBalance)
	}

	if bankBalanceFound {
		summary.CashEquivalents = totalBankBalance
	} else {
		// Fallback: use 2024 value as starting point
		summary.CashEquivalents = fs2024.CashEquivalents
		if summary.NetProfit > 0 {
			summary.CashEquivalents += summary.NetProfit * 0.7
		}
		if summary.CashEquivalents < 0 {
			summary.CashEquivalents = fs2024.CashEquivalents * 0.8
		}
	}

	// Calculate totals - use actual AR/AP from database queries (most reliable)
	summary.CurrentAssets = summary.Inventories + summary.TradeReceivables +
		summary.FixedDeposits + summary.RelatedPartyRecv + summary.CashEquivalents
	summary.NonCurrentAssets = summary.PlantEquipment + summary.RightOfUse + summary.InvestmentProp
	summary.TotalAssets = summary.CurrentAssets + summary.NonCurrentAssets

	// For equity, use 2024 as base - don't cascade losses into negative equity
	summary.OwnerCurrentAcct = fs2024.OwnerCurrentAcct
	if summary.NetProfit > 0 {
		summary.OwnerCurrentAcct += summary.NetProfit
	} else if summary.NetProfit < 0 {
		// For losses, reduce but don't go below a minimum (company still exists)
		reduction := math.Min(math.Abs(summary.NetProfit), fs2024.OwnerCurrentAcct*0.3)
		summary.OwnerCurrentAcct -= reduction
	}
	summary.TotalEquity = summary.ShareCapital + summary.StatutoryReserve + summary.OwnerCurrentAcct

	// Liabilities - these come from actual AP data
	summary.LeaseNonCurrent = fs2024.LeaseNonCurrent - 10000 // Approximate lease payment
	if summary.LeaseNonCurrent < 0 {
		summary.LeaseNonCurrent = 0
	}
	summary.LeaseCurrent = fs2024.LeaseCurrent
	summary.CurrentLiabilities = summary.TradePayables + summary.LeaseCurrent
	summary.TotalLiabilities = summary.CurrentLiabilities + summary.LeaseNonCurrent

	if summary.Revenue == 0 {
		summary.Source = "Live Data (Unaudited - no posted invoices)"
	}

	log.Printf("✅ Calculated financial data for year %d: Revenue=%.2f, GrossProfit=%.2f",
		year, summary.Revenue, summary.GrossProfit)

	return summary, nil
}

func (a *App) calculateFreshStartFinancialYear(year int, summary *FinancialYearSummary) (*FinancialYearSummary, error) {
	start := operationalMetricStartForYear(year)
	end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)

	sourceBasis := "posted invoices"
	summary.Source = fmt.Sprintf("Fresh Start Live Data (from %s)", start.Format("2 Jan 2006"))
	summary.AsOfDate = time.Now().Format("2006-01-02")

	postedInvoiceStatuses := []string{"Sent", "Paid", "PartiallyPaid", "Overdue"}
	if err := a.db.Model(&Invoice{}).
		Where("status IN ? AND invoice_date >= ? AND invoice_date < ?", postedInvoiceStatuses, start, end).
		Select("COALESCE(SUM(grand_total_bhd), 0)").
		Scan(&summary.Revenue).Error; err != nil {
		log.Printf("⚠️ Error fetching fresh-start invoice revenue for year %d: %v", year, err)
	}

	var bookedOrderValue float64
	if err := a.db.Model(&Order{}).
		Where("status NOT IN ? AND order_date >= ? AND order_date < ?", []string{"Draft", "Cancelled", "Canceled", "Void"}, start, end).
		Select("COALESCE(SUM(CASE WHEN grand_total_bhd > 0 THEN grand_total_bhd ELSE total_value_bhd END), 0)").
		Scan(&bookedOrderValue).Error; err != nil {
		log.Printf("⚠️ Error fetching fresh-start booked orders for year %d: %v", year, err)
	} else if bookedOrderValue > summary.Revenue {
		summary.Revenue = bookedOrderValue
		sourceBasis = "posted invoices plus booked orders"
		log.Printf("📊 Year %d using booked order value for live revenue exposure: %.2f BHD", year, bookedOrderValue)
	}

	if err := a.db.Model(&SupplierInvoice{}).
		Where("invoice_date >= ? AND invoice_date < ? AND status NOT IN ?", start, end, []string{"Rejected", "Dispute"}).
		Select("COALESCE(SUM(total_bhd), 0)").
		Scan(&summary.CostOfSales).Error; err != nil {
		log.Printf("⚠️ Error fetching fresh-start supplier costs for year %d: %v", year, err)
	}

	openInvoiceStatuses := []string{"Sent", "PartiallyPaid", "Overdue"}
	var openInvoiceAR float64
	if err := a.db.Model(&Invoice{}).
		Where("status IN ? AND invoice_date >= ? AND invoice_date < ?", openInvoiceStatuses, start, end).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&openInvoiceAR).Error; err != nil {
		log.Printf("⚠️ Error fetching fresh-start AR for year %d: %v", year, err)
	}
	summary.TradeReceivables = openInvoiceAR

	orderExposure, pendingOrders, exposureErr := a.calculateUninvoicedOrderExposure(start, end)
	if exposureErr != nil {
		log.Printf("⚠️ Error fetching fresh-start uninvoiced order exposure for year %d: %v", year, exposureErr)
	} else if orderExposure > 0 {
		summary.TradeReceivables += orderExposure
		log.Printf("📊 Year %d added %d uninvoiced orders to AR exposure: %.2f BHD", year, pendingOrders, orderExposure)
	}

	if err := a.db.Model(&SupplierInvoice{}).
		Where("invoice_date >= ? AND invoice_date < ? AND status NOT IN ?", start, end, []string{"Paid", "Rejected", "Dispute"}).
		Select("COALESCE(SUM(total_bhd), 0)").
		Scan(&summary.TradePayables).Error; err != nil {
		log.Printf("⚠️ Error fetching fresh-start AP for year %d: %v", year, err)
	}

	summary.Source = fmt.Sprintf("Fresh Start Live Data (from %s; %s)", start.Format("2 Jan 2006"), sourceBasis)
	applyFreshStartProfitabilityEstimate(summary)

	if cash, err := computeCashPositionSnapshot(a); err == nil {
		summary.CashEquivalents = cash.CashBalanceBHD
	} else {
		log.Printf("⚠️ Error fetching fresh-start cash position: %v", err)
	}

	summary.CurrentAssets = summary.TradeReceivables + summary.CashEquivalents
	summary.TotalAssets = summary.CurrentAssets
	summary.CurrentLiabilities = summary.TradePayables
	summary.TotalLiabilities = summary.CurrentLiabilities
	summary.TotalEquity = summary.TotalAssets - summary.TotalLiabilities
	summary.OwnerCurrentAcct = summary.TotalEquity

	log.Printf("✅ Fresh-start financial data for year %d from %s: Revenue=%.2f, Cash=%.2f, AR=%.2f",
		year, start.Format("2006-01-02"), summary.Revenue, summary.CashEquivalents, summary.TradeReceivables)

	return summary, nil
}

func applyFreshStartProfitabilityEstimate(summary *FinancialYearSummary) {
	if summary == nil || summary.Revenue <= 0 {
		return
	}

	fs2024 := getVerifiedFS2024Data()[2024]
	if summary.CostOfSales <= 0 && fs2024.Revenue > 0 {
		grossMarginPct2024 := (fs2024.Revenue - fs2024.CostOfSales) / fs2024.Revenue
		summary.CostOfSales = summary.Revenue * (1 - grossMarginPct2024)
	}
	summary.GrossProfit = summary.Revenue - summary.CostOfSales

	revenueRatio := summary.Revenue / fs2024.Revenue
	yearStart := operationalMetricStartForYear(summary.Year)
	yearEnd := time.Date(summary.Year+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	asOf := time.Now()
	if asOf.Before(yearStart) {
		asOf = yearStart
	}
	if asOf.After(yearEnd) {
		asOf = yearEnd
	}
	elapsedYearRatio := asOf.Sub(yearStart).Hours() / yearEnd.Sub(yearStart).Hours()
	if elapsedYearRatio <= 0 || elapsedYearRatio > 1 {
		elapsedYearRatio = 1
	}

	summary.StaffCosts = fs2024.StaffCosts * revenueRatio * 0.9
	summary.AdminExpenses = fs2024.AdminExpenses * revenueRatio
	summary.Depreciation = fs2024.Depreciation * elapsedYearRatio
	summary.FinanceCosts = fs2024.FinanceCosts * 0.95 * elapsedYearRatio
	summary.OtherIncome = fs2024.OtherIncome * revenueRatio

	totalOpex := summary.StaffCosts + summary.AdminExpenses + summary.Depreciation + summary.FinanceCosts
	summary.NetProfit = summary.GrossProfit + summary.OtherIncome - totalOpex
}

// GetDynamicFinancialDashboard returns a complete dashboard with calculated ratios
func (a *App) GetDynamicFinancialDashboard(year int) (FinancialDashboard, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return FinancialDashboard{}, err
	}
	log.Printf("📊 Building dynamic financial dashboard for year %d", year)

	// Get current year data
	currentYear, err := a.GetFinancialYearData(year)
	if err != nil {
		return FinancialDashboard{}, fmt.Errorf("failed to get data for year %d: %w", year, err)
	}

	// Get prior year data for YoY comparison
	priorYear, err := a.GetFinancialYearData(year - 1)
	if err != nil {
		log.Printf("⚠️ Could not get prior year data: %v", err)
		priorYear = &FinancialYearSummary{Year: year - 1}
	}

	// Build dashboard with calculated ratios
	dashboard := FinancialDashboard{
		Period:    fmt.Sprintf("FY%d", year),
		PriorYear: fmt.Sprintf("FY%d", year-1),
		AsOfDate:  currentYear.AsOfDate,
		Source:    currentYear.Source,

		// P&L Summary
		Revenue:     currentYear.Revenue,
		COGS:        currentYear.CostOfSales,
		GrossProfit: currentYear.GrossProfit,
		NetProfit:   currentYear.NetProfit,

		// Balance Sheet
		TotalAssets:      currentYear.TotalAssets,
		CurrentAssets:    currentYear.CurrentAssets,
		NonCurrentAssets: currentYear.NonCurrentAssets,
		TotalLiabilities: currentYear.TotalLiabilities,
		CurrentLiab:      currentYear.CurrentLiabilities,
		TotalEquity:      currentYear.TotalEquity,

		// Cash Position
		CashAndEquiv:   currentYear.CashEquivalents,
		FixedDeposits:  currentYear.FixedDeposits,
		TotalLiquidity: currentYear.CashEquivalents + currentYear.FixedDeposits,

		// Working Capital
		TradeReceivables: currentYear.TradeReceivables,
		Inventory:        currentYear.Inventories,
		TradePayables:    currentYear.TradePayables,
		WorkingCapital:   currentYear.CurrentAssets - currentYear.CurrentLiabilities,

		// Prior Year for YoY
		PY_Revenue:     priorYear.Revenue,
		PY_GrossProfit: priorYear.GrossProfit,
		PY_NetProfit:   priorYear.NetProfit,
		PY_TotalAssets: priorYear.TotalAssets,
	}

	// Calculate OpEx and EBITDA
	totalOpex := currentYear.StaffCosts + currentYear.AdminExpenses
	dashboard.OpEx = totalOpex
	dashboard.EBITDA = currentYear.GrossProfit + currentYear.OtherIncome - totalOpex + currentYear.Depreciation

	// Calculate Margins (%)
	if currentYear.Revenue > 0 {
		dashboard.GrossMargin = (currentYear.GrossProfit / currentYear.Revenue) * 100
		dashboard.NetMargin = (currentYear.NetProfit / currentYear.Revenue) * 100
		dashboard.EBITDAMargin = (dashboard.EBITDA / currentYear.Revenue) * 100
	}

	// Calculate YoY Changes (%)
	if priorYear.Revenue > 0 {
		dashboard.RevenueYoY = ((currentYear.Revenue - priorYear.Revenue) / priorYear.Revenue) * 100
	}

	// Calculate Liquidity Ratios
	if currentYear.CurrentLiabilities > 0 {
		dashboard.CurrentRatio = currentYear.CurrentAssets / currentYear.CurrentLiabilities
		dashboard.QuickRatio = (currentYear.CurrentAssets - currentYear.Inventories) / currentYear.CurrentLiabilities
		dashboard.CashRatio = currentYear.CashEquivalents / currentYear.CurrentLiabilities
	}

	// Calculate Solvency Ratios
	if currentYear.TotalEquity > 0 {
		dashboard.DebtToEquity = currentYear.TotalLiabilities / currentYear.TotalEquity
		dashboard.ROE = (currentYear.NetProfit / currentYear.TotalEquity) * 100
	}
	if currentYear.TotalAssets > 0 {
		dashboard.EquityRatio = (currentYear.TotalEquity / currentYear.TotalAssets) * 100
		dashboard.ROA = (currentYear.NetProfit / currentYear.TotalAssets) * 100
		dashboard.AssetTurnover = currentYear.Revenue / currentYear.TotalAssets
	}

	// Calculate Efficiency Ratios (Days)
	if currentYear.Revenue > 0 {
		// DSO = (Trade Receivables / Revenue) * 365
		dashboard.DSO = (currentYear.TradeReceivables / currentYear.Revenue) * 365
		// Receivables Turnover = Revenue / Trade Receivables
		if currentYear.TradeReceivables > 0 {
			dashboard.ReceivablesTurn = currentYear.Revenue / currentYear.TradeReceivables
		}
	}
	if currentYear.CostOfSales > 0 {
		// DIO = (Inventory / COGS) * 365
		dashboard.DIO = (currentYear.Inventories / currentYear.CostOfSales) * 365
		// DPO = (Trade Payables / COGS) * 365
		dashboard.DPO = (currentYear.TradePayables / currentYear.CostOfSales) * 365
	}
	// Cash Conversion Cycle = DSO + DIO - DPO
	dashboard.CashConvCycle = dashboard.DSO + dashboard.DIO - dashboard.DPO

	// Get AR Aging from invoices
	arAging := a.calculateARAging(year)
	dashboard.ARCurrent = arAging.Current
	dashboard.AR30_60 = arAging.Days30_60
	dashboard.AR60_90 = arAging.Days60_90
	dashboard.AROver90 = arAging.Over90
	dashboard.AROverdue = arAging.Days30_60 + arAging.Days60_90 + arAging.Over90
	totalAR := arAging.Current + arAging.Days30_60 + arAging.Days60_90 + arAging.Over90
	if currentYear.TradeReceivables > totalAR {
		dashboard.ARCurrent += currentYear.TradeReceivables - totalAR
		totalAR = currentYear.TradeReceivables
	}
	if totalAR > 0 {
		dashboard.AROverduePct = (dashboard.AROverdue / totalAR) * 100
	}

	// Round all values to 3 decimal places (BHD standard)
	dashboard = roundDashboardValues(dashboard)

	log.Printf("✅ Dashboard built: Revenue=%.3f, GrossMargin=%.1f%%, CurrentRatio=%.2f",
		dashboard.Revenue, dashboard.GrossMargin, dashboard.CurrentRatio)

	return dashboard, nil
}

// ARAging represents accounts receivable aging buckets
type ARAging struct {
	Current   float64 `json:"current"`    // 0-30 days
	Days30_60 float64 `json:"days_30_60"` // 31-60 days
	Days60_90 float64 `json:"days_60_90"` // 61-90 days
	Over90    float64 `json:"over_90"`    // 90+ days
}

// calculateARAging computes AR aging buckets from invoice data
func (a *App) calculateARAging(year int) ARAging {
	aging := ARAging{}
	now := time.Now()

	// Get unpaid invoices for the year (SQLite strftime)
	var invoices []Invoice
	query := a.db.Where("status NOT IN ?", []string{"Paid", "Cancelled", "Void", "Proforma", "Draft"})
	if usesOperationalFreshStart(year) {
		query = query.Where("invoice_date >= ? AND invoice_date < ?", operationalMetricStartForYear(year), time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC))
	} else {
		query = query.Where("strftime('%Y', invoice_date) = ?", fmt.Sprintf("%d", year))
	}
	err := query.Find(&invoices).Error
	if err != nil {
		log.Printf("⚠️ Error fetching invoices for AR aging: %v", err)
		return aging
	}

	for _, inv := range invoices {
		// OutstandingBHD is pre-calculated as GrandTotalBHD - payments received
		outstanding := inv.OutstandingBHD
		if outstanding <= 0 {
			continue
		}

		daysPastDue := int(now.Sub(inv.DueDate).Hours() / 24)
		if daysPastDue < 0 {
			daysPastDue = 0 // Not yet due, count as current
		}

		switch {
		case daysPastDue <= 30:
			aging.Current += outstanding
		case daysPastDue <= 60:
			aging.Days30_60 += outstanding
		case daysPastDue <= 90:
			aging.Days60_90 += outstanding
		default:
			aging.Over90 += outstanding
		}
	}

	return aging
}

// roundDashboardValues rounds all numeric values to appropriate precision
func roundDashboardValues(d FinancialDashboard) FinancialDashboard {
	round3 := func(v float64) float64 { return math.Round(v*1000) / 1000 }
	round1 := func(v float64) float64 { return math.Round(v*10) / 10 }
	round2 := func(v float64) float64 { return math.Round(v*100) / 100 }

	// Currency values (3 decimals for BHD)
	d.Revenue = round3(d.Revenue)
	d.COGS = round3(d.COGS)
	d.GrossProfit = round3(d.GrossProfit)
	d.OpEx = round3(d.OpEx)
	d.EBITDA = round3(d.EBITDA)
	d.NetProfit = round3(d.NetProfit)
	d.TotalAssets = round3(d.TotalAssets)
	d.CurrentAssets = round3(d.CurrentAssets)
	d.NonCurrentAssets = round3(d.NonCurrentAssets)
	d.TotalLiabilities = round3(d.TotalLiabilities)
	d.CurrentLiab = round3(d.CurrentLiab)
	d.TotalEquity = round3(d.TotalEquity)
	d.CashAndEquiv = round3(d.CashAndEquiv)
	d.FixedDeposits = round3(d.FixedDeposits)
	d.TotalLiquidity = round3(d.TotalLiquidity)
	d.TradeReceivables = round3(d.TradeReceivables)
	d.Inventory = round3(d.Inventory)
	d.TradePayables = round3(d.TradePayables)
	d.WorkingCapital = round3(d.WorkingCapital)
	d.PY_Revenue = round3(d.PY_Revenue)
	d.PY_GrossProfit = round3(d.PY_GrossProfit)
	d.PY_NetProfit = round3(d.PY_NetProfit)
	d.PY_TotalAssets = round3(d.PY_TotalAssets)
	d.ARCurrent = round3(d.ARCurrent)
	d.AR30_60 = round3(d.AR30_60)
	d.AR60_90 = round3(d.AR60_90)
	d.AROver90 = round3(d.AROver90)
	d.AROverdue = round3(d.AROverdue)

	// Percentages (1 decimal)
	d.RevenueYoY = round1(d.RevenueYoY)
	d.GrossMargin = round1(d.GrossMargin)
	d.EBITDAMargin = round1(d.EBITDAMargin)
	d.NetMargin = round1(d.NetMargin)
	d.EquityRatio = round1(d.EquityRatio)
	d.ROA = round1(d.ROA)
	d.ROE = round1(d.ROE)
	d.AROverduePct = round1(d.AROverduePct)

	// Ratios (2 decimals)
	d.CurrentRatio = round2(d.CurrentRatio)
	d.QuickRatio = round2(d.QuickRatio)
	d.CashRatio = round2(d.CashRatio)
	d.DebtToEquity = round2(d.DebtToEquity)
	d.AssetTurnover = round2(d.AssetTurnover)
	d.ReceivablesTurn = round2(d.ReceivablesTurn)

	// Days (1 decimal)
	d.DSO = round1(d.DSO)
	d.DIO = round1(d.DIO)
	d.DPO = round1(d.DPO)
	d.CashConvCycle = round1(d.CashConvCycle)

	return d
}

// GetAvailableFinancialYears returns years with available financial data
func (a *App) GetAvailableFinancialYears() ([]int, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	yearsMap := make(map[int]bool)

	// Always include audited years
	yearsMap[2024] = true
	yearsMap[2023] = true

	// Add years from live invoices (using SQLite strftime)
	var invoiceYears []string
	err := a.db.Model(&Invoice{}).
		Select("DISTINCT strftime('%Y', invoice_date) as year").
		Where("invoice_date IS NOT NULL AND deleted_at IS NULL").
		Pluck("year", &invoiceYears).Error
	if err == nil {
		for _, yearStr := range invoiceYears {
			if yearStr != "" {
				var year int
				fmt.Sscanf(yearStr, "%d", &year)
				if year > 0 && year != 2023 && year != 2024 {
					yearsMap[year] = true
				}
			}
		}
	}

	// Add years from booked orders so fresh-start years with draft/unposted invoices
	// still appear in Finance Hub.
	var orderYears []string
	err = a.db.Model(&Order{}).
		Select("DISTINCT strftime('%Y', order_date) as year").
		Where("order_date IS NOT NULL AND deleted_at IS NULL AND status NOT IN ?", []string{"Draft", "Cancelled", "Canceled", "Void"}).
		Pluck("year", &orderYears).Error
	if err == nil {
		for _, yearStr := range orderYears {
			if yearStr != "" {
				var year int
				fmt.Sscanf(yearStr, "%d", &year)
				if year > 0 && year != 2023 && year != 2024 {
					yearsMap[year] = true
				}
			}
		}
	}

	// Add years from bank statements
	var bankYears []string
	err = a.db.Model(&BankStatement{}).
		Select("DISTINCT strftime('%Y', period_end) as year").
		Where("period_end IS NOT NULL").
		Pluck("year", &bankYears).Error
	if err == nil {
		for _, yearStr := range bankYears {
			if yearStr != "" {
				var year int
				fmt.Sscanf(yearStr, "%d", &year)
				if year > 0 {
					yearsMap[year] = true
				}
			}
		}
	}

	// Convert map to sorted slice (descending)
	years := make([]int, 0, len(yearsMap))
	for year := range yearsMap {
		years = append(years, year)
	}

	// Sort descending (newest first)
	for i := 0; i < len(years)-1; i++ {
		for j := i + 1; j < len(years); j++ {
			if years[j] > years[i] {
				years[i], years[j] = years[j], years[i]
			}
		}
	}

	log.Printf("📅 Available financial years: %v", years)
	return years, nil
}

// =============================================================================
// E2: DIVISION-FILTERED FINANCIAL DASHBOARD (Beacon Controls Support)
// =============================================================================

// DivisionFinancialSummary represents financial summary for a specific division
type DivisionFinancialSummary struct {
	Division         string  `json:"division"`
	Year             int     `json:"year"`
	Revenue          float64 `json:"revenue"`
	InvoiceCount     int     `json:"invoice_count"`
	OrderCount       int     `json:"order_count"`
	OutstandingAR    float64 `json:"outstanding_ar"`
	PaidAmount       float64 `json:"paid_amount"`
	OverdueAmount    float64 `json:"overdue_amount"`
	OverdueCount     int     `json:"overdue_count"`
	AvgInvoiceSize   float64 `json:"avg_invoice_size"`
	Source           string  `json:"source"`
	IsAudited        bool    `json:"is_audited"`
	CostOfSales      float64 `json:"cost_of_sales"`
	GrossProfit      float64 `json:"gross_profit"`
	StaffCosts       float64 `json:"staff_costs"`
	AdminExpenses    float64 `json:"admin_expenses"`
	NetProfit        float64 `json:"net_profit"`
	CashEquivalents  float64 `json:"cash_equivalents"`
	TradeReceivables float64 `json:"trade_receivables"`
	TotalAssets      float64 `json:"total_assets"`
	TotalLiabilities float64 `json:"total_liabilities"`
	TotalEquity      float64 `json:"total_equity"`
	HasData          bool    `json:"has_data"`
}

// getAHSAuditedDivisionData returns the synthetic audited-financials demo
// numbers for the division whose overlay DashboardVariant is "ahs". The
// numbers themselves are demo data (canon) and stay literal; only the
// Division field is parameterized on the caller's (already normalized)
// division key so this no longer hardcodes "Beacon Controls" as a literal
// vocabulary comparison.
func getAHSAuditedDivisionData(divisionKey string) map[int]DivisionFinancialSummary {
	return map[int]DivisionFinancialSummary{
		2023: {
			Division:         divisionKey,
			Year:             2023,
			Source:           "Demo financial data (Audited)",
			IsAudited:        true,
			Revenue:          0,
			CostOfSales:      0,
			GrossProfit:      0,
			StaffCosts:       0,
			AdminExpenses:    1962,
			NetProfit:        -1962,
			CashEquivalents:  427,
			TradeReceivables: 100,
			OutstandingAR:    100,
			TotalAssets:      44893,
			TotalLiabilities: 4538,
			TotalEquity:      40355,
			HasData:          true,
		},
		2024: {
			Division:         divisionKey,
			Year:             2024,
			Source:           "Demo financial data (Audited)",
			IsAudited:        true,
			Revenue:          63570,
			CostOfSales:      60542,
			GrossProfit:      3028,
			StaffCosts:       3722,
			AdminExpenses:    2693,
			NetProfit:        -3387,
			CashEquivalents:  2520,
			TradeReceivables: 70027,
			OutstandingAR:    70027,
			TotalAssets:      72547,
			TotalLiabilities: 35579,
			TotalEquity:      36968,
			HasData:          true,
		},
	}
}

// GetFinancialDashboardByDivision returns financial metrics filtered by division (Acme Instrumentation or Beacon Controls)
func (a *App) GetFinancialDashboardByDivision(year int, division string) (DivisionFinancialSummary, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return DivisionFinancialSummary{}, err
	}
	if a.db == nil {
		return DivisionFinancialSummary{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	// Validate division
	if !activeOverlay.IsKnownDivision(division) {
		return DivisionFinancialSummary{}, fmt.Errorf("invalid division: %s", division)
	}
	// Validate year bounds
	if year < 2020 || year > time.Now().Year()+1 {
		return DivisionFinancialSummary{}, fmt.Errorf("year must be between 2020 and %d", time.Now().Year()+1)
	}

	// Normalize to the canonical registry key so downstream comparisons and
	// SQL filters compare canonicalized values (Spec-07 canonicalization law).
	// For the synthetic overlay's exact keys this is identity — byte-identical
	// behavior for the default deployment.
	division = activeOverlay.NormalizeDivisionName(division)

	summary := DivisionFinancialSummary{
		Division: division,
		Year:     year,
	}

	if activeOverlay.Profile(division).DashboardVariant == "ahs" {
		if audited, ok := getAHSAuditedDivisionData(division)[year]; ok {
			return audited, nil
		}
	}

	yearStr := fmt.Sprintf("%d", year)

	// Get invoices for this division and year
	var invoices []Invoice
	err := a.db.Where("division = ? AND strftime('%Y', invoice_date) = ? AND status != 'Cancelled'", division, yearStr).
		Find(&invoices).Error
	if err != nil {
		log.Printf("Warning: Error fetching invoices for division %s, year %d: %v", division, year, err)
		return summary, nil
	}

	// Calculate revenue metrics from invoices
	for _, inv := range invoices {
		summary.Revenue += inv.GrandTotalBHD
		summary.InvoiceCount++
		summary.OutstandingAR += inv.OutstandingBHD
		summary.PaidAmount += (inv.GrandTotalBHD - inv.OutstandingBHD)

		if inv.Status == "Overdue" {
			summary.OverdueAmount += inv.OutstandingBHD
			summary.OverdueCount++
		}
	}

	if summary.InvoiceCount > 0 {
		summary.AvgInvoiceSize = summary.Revenue / float64(summary.InvoiceCount)
		summary.HasData = true
	}

	// Get order count for this division and year
	var orderCount int64
	a.db.Model(&Order{}).Where("division = ? AND strftime('%Y', order_date) = ? AND status != 'Cancelled'", division, yearStr).
		Count(&orderCount)
	summary.OrderCount = int(orderCount)

	if summary.OrderCount > 0 {
		summary.HasData = true
	}

	// Round to 3 decimal places (BHD)
	round3 := func(v float64) float64 { return math.Round(v*1000) / 1000 }
	summary.Revenue = round3(summary.Revenue)
	summary.OutstandingAR = round3(summary.OutstandingAR)
	summary.PaidAmount = round3(summary.PaidAmount)
	summary.OverdueAmount = round3(summary.OverdueAmount)
	summary.AvgInvoiceSize = round3(summary.AvgInvoiceSize)

	log.Printf("Division %s FY%d: Revenue=%.3f, Invoices=%d, Orders=%d, Outstanding=%.3f",
		division, year, summary.Revenue, summary.InvoiceCount, summary.OrderCount, summary.OutstandingAR)
	return summary, nil
}
