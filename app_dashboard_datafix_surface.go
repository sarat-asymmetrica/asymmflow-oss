package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

func (a *App) GetFinancialDashboard() FinancialDashboard {
	if err := a.requirePermission("finance:view"); err != nil {
		log.Printf("⚠️ GetFinancialDashboard: permission denied")
		return FinancialDashboard{}
	}
	// Initialize with audited FS2024 data from Demo Auditors audit
	dashboard := FinancialDashboard{
		Period:    "FY2024",
		PriorYear: "FY2023",
		AsOfDate:  "2024-12-31",
		Source:    "Demo Auditors FS2024 (Audited)",

		// P&L - From audited financials
		Revenue:      1605794.0,
		RevenueYoY:   -69.3,
		COGS:         1302630.0,
		GrossProfit:  289998.0,
		GrossMargin:  18.2,
		OpEx:         255857.0, // Staff 178,757 + Director 24,000 + G&A 32,281 + Depreciation 20,819
		EBITDA:       54960.0,  // Gross Profit + Depreciation - Other OpEx
		EBITDAMargin: 3.4,
		NetProfit:    20715.0,
		NetMargin:    1.3,

		// Balance Sheet
		TotalAssets:      1127325.0,
		CurrentAssets:    626518.0,
		NonCurrentAssets: 500807.0,
		TotalLiabilities: 165642.0,
		CurrentLiab:      133256.0,
		TotalEquity:      961683.0,

		// Cash Position
		CashAndEquiv:   413295.0,
		FixedDeposits:  50000.0,
		TotalLiquidity: 463295.0,

		// Working Capital
		TradeReceivables: 123393.0,
		Inventory:        5104.0,
		TradePayables:    123882.0,
		WorkingCapital:   493262.0,

		// Liquidity Ratios
		CurrentRatio: 4.7,
		QuickRatio:   4.7,
		CashRatio:    3.1,

		// Solvency Ratios
		DebtToEquity: 0.04,
		EquityRatio:  85.3,

		// Efficiency Ratios
		DSO:             17.0,  // Excellent - fast collection
		DIO:             1.4,   // Minimal inventory holding
		DPO:             34.7,  // Calculated from payables/daily purchases
		CashConvCycle:   -16.3, // Negative = very efficient
		AssetTurnover:   1.41,
		ReceivablesTurn: 13.9,

		// Profitability Ratios (2024 anomaly year)
		ROA: 1.8,
		ROE: 2.2,

		// AR Aging (from payment analysis)
		ARCurrent:    28848.0,
		AR30_60:      26533.0,
		AR60_90:      7090.0,
		AROver90:     53231.0,
		AROverdue:    86854.0,
		AROverduePct: 35.0,

		// Prior Year (2023) comparisons
		PY_Revenue:     5237432.0,
		PY_GrossProfit: 883580.0,
		PY_NetProfit:   672908.0,
		PY_TotalAssets: 1232384.0,
	}

	// Try to get live data from database if available
	if a.db != nil {
		a.enrichDashboardFromDB(&dashboard)
	}

	log.Printf("📊 Financial Dashboard: Revenue %.0f, Gross Margin %.1f%%, Net Profit %.0f, Current Ratio %.1f",
		dashboard.Revenue, dashboard.GrossMargin, dashboard.NetProfit, dashboard.CurrentRatio)

	return dashboard
}

// GetFinancialDashboardForYear returns financial metrics for a specific year
// Uses verified FS2024 data for 2023/2024, calculates live from Tally for other years
// Cross-verified against Demo Auditors audited financials (2026-01-28)
func (a *App) GetFinancialDashboardForYear(year int) (FinancialDashboard, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return FinancialDashboard{}, err
	}
	log.Printf("📊 Loading Financial Dashboard for year %d", year)

	// Use the dynamic financial service with verified data
	dashboard, err := a.GetDynamicFinancialDashboard(year)
	if err != nil {
		log.Printf("⚠️ Error from dynamic dashboard, using fallback: %v", err)
		// Return a minimal fallback dashboard
		return FinancialDashboard{
			Period:    fmt.Sprintf("FY%d", year),
			PriorYear: fmt.Sprintf("FY%d", year-1),
			AsOfDate:  fmt.Sprintf("%d-12-31", year),
		}, err
	}

	// Enrich with live database data (AR aging, etc.) for current/recent years
	// Log data source
	verifiedYears := map[int]bool{2023: true, 2024: true}
	if verifiedYears[year] {
		log.Printf("📊 Financial Dashboard for FY%d: Using Demo Auditors audited data (FS2024)", year)
	} else {
		log.Printf("📊 Financial Dashboard for FY%d: Using Tally import data (unaudited)", year)
	}

	return dashboard, nil
}

// enrichDashboardFromDB updates dashboard with live database values
func (a *App) enrichDashboardFromDB(d *FinancialDashboard) {
	yearStart := time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	// YTD Revenue from invoices (using grand_total_bhd - correct column name)
	var ytdRevenue float64
	if err := a.db.Model(&Invoice{}).
		Where("invoice_date >= ? AND status NOT IN ('draft', 'cancelled', 'Draft', 'Cancelled')", yearStart).
		Select("COALESCE(SUM(grand_total_bhd), 0)").
		Scan(&ytdRevenue).Error; err == nil && ytdRevenue > 0 {
		d.Revenue = ytdRevenue
	}

	// Current AR from open invoices (using outstanding_bhd - correct column name)
	var currentAR float64
	if err := a.db.Model(&Invoice{}).
		Where("status IN ('sent', 'overdue', 'partially_paid', 'Sent', 'Overdue')").
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&currentAR).Error; err == nil && currentAR > 0 {
		d.TradeReceivables = currentAR
	}

	// AR Aging buckets (using outstanding_bhd - correct column name)
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	sixtyDaysAgo := now.AddDate(0, 0, -60)
	ninetyDaysAgo := now.AddDate(0, 0, -90)

	// Current (0-30 days)
	a.db.Model(&Invoice{}).
		Where("status IN ('sent', 'overdue', 'partially_paid', 'Sent', 'Overdue') AND invoice_date >= ?", thirtyDaysAgo).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&d.ARCurrent)

	// 31-60 days
	a.db.Model(&Invoice{}).
		Where("status IN ('sent', 'overdue', 'partially_paid', 'Sent', 'Overdue') AND invoice_date < ? AND invoice_date >= ?", thirtyDaysAgo, sixtyDaysAgo).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&d.AR30_60)

	// 61-90 days
	a.db.Model(&Invoice{}).
		Where("status IN ('sent', 'overdue', 'partially_paid', 'Sent', 'Overdue') AND invoice_date < ? AND invoice_date >= ?", sixtyDaysAgo, ninetyDaysAgo).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&d.AR60_90)

	// >90 days
	a.db.Model(&Invoice{}).
		Where("status IN ('sent', 'overdue', 'partially_paid', 'Sent', 'Overdue') AND invoice_date < ?", ninetyDaysAgo).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&d.AROver90)

	// Calculate overdue totals
	d.AROverdue = d.AR30_60 + d.AR60_90 + d.AROver90
	totalAR := d.ARCurrent + d.AROverdue
	if totalAR > 0 {
		d.AROverduePct = (d.AROverdue / totalAR) * 100
	}

	// Recalculate DSO if we have revenue
	if d.Revenue > 0 && d.TradeReceivables > 0 {
		d.DSO = (d.TradeReceivables / d.Revenue) * 365
	}
}

// ============================================================================
// CRM CUSTOMER DASHBOARD API
// ============================================================================

// CRMCustomerDashboard represents McKinsey-style customer command center metrics
type CRMCustomerDashboard struct {
	// KPIs
	TotalCustomers   int     `json:"total_customers"`
	ActiveCustomers  int     `json:"active_customers"`  // Ordered in last 12 months
	TotalRevenue     float64 `json:"total_revenue"`     // YTD
	RevenueYoY       float64 `json:"revenue_yoy"`       // % change
	TotalOutstanding float64 `json:"total_outstanding"` // Open exposure: invoice AR + uninvoiced booked orders
	OverdueAmount    float64 `json:"overdue_amount"`    // Invoice AR >30 days
	OverduePct       float64 `json:"overdue_pct"`

	// Top Customers
	TopCustomers []CustomerMetricCard `json:"top_customers"`

	// Grade Distribution
	GradeACount   int     `json:"grade_a_count"`
	GradeARevenue float64 `json:"grade_a_revenue"`
	GradeBCount   int     `json:"grade_b_count"`
	GradeBRevenue float64 `json:"grade_b_revenue"`
	GradeCCount   int     `json:"grade_c_count"`
	GradeCRevenue float64 `json:"grade_c_revenue"`
	GradeDCount   int     `json:"grade_d_count"`
	GradeDRevenue float64 `json:"grade_d_revenue"`

	// Concentration Risk
	Top3RevenuePct  float64 `json:"top3_revenue_pct"`
	Top5RevenuePct  float64 `json:"top5_revenue_pct"`
	Top10RevenuePct float64 `json:"top10_revenue_pct"`

	// All Customers (for cards)
	Customers []CustomerMetricCard `json:"customers"`
}

type CustomerMetricCard struct {
	ID             string     `json:"id"`
	BusinessName   string     `json:"business_name"`
	CustomerType   string     `json:"customer_type"`
	PaymentGrade   string     `json:"payment_grade"`
	TotalRevenue   float64    `json:"total_revenue"`
	ActiveInvoices int        `json:"active_invoices"`
	OutstandingBHD float64    `json:"outstanding_bhd"`
	OverdueBHD     float64    `json:"overdue_bhd"`
	LastOrderDate  *time.Time `json:"last_order_date"`
	City           string     `json:"city"`
}

func (a *App) latestCRMActivityYear() int {
	var latestInvoiceYear int
	a.db.Model(&Invoice{}).Select("COALESCE(MAX(CAST(strftime('%Y', invoice_date) AS INTEGER)), 0)").Scan(&latestInvoiceYear)

	var latestOrderYear int
	a.db.Model(&Order{}).Select("COALESCE(MAX(CAST(strftime('%Y', order_date) AS INTEGER)), 0)").Scan(&latestOrderYear)

	var latestOfferYear int
	a.db.Model(&Offer{}).Select("COALESCE(MAX(CAST(strftime('%Y', quotation_date) AS INTEGER)), 0)").Scan(&latestOfferYear)

	var latestOpportunityYear int
	a.db.Model(&Opportunity{}).Select("COALESCE(MAX(year), 0)").Scan(&latestOpportunityYear)

	year := max(latestInvoiceYear, max(latestOrderYear, max(latestOfferYear, latestOpportunityYear)))
	if year == 0 {
		return time.Now().Year()
	}
	return year
}

// dashboardYTDWindow returns the dashboard's activity year plus its [start,
// end) window. For the current calendar year the window is year-to-date
// (through end of today); for a historical activity year it is the full year.
func (a *App) dashboardYTDWindow() (int, time.Time, time.Time) {
	chartYear := a.latestCRMActivityYear()
	now := time.Now()
	loc := now.Location()
	start := time.Date(chartYear, time.January, 1, 0, 0, 0, 0, loc)
	end := time.Date(chartYear+1, time.January, 1, 0, 0, 0, 0, loc)

	if now.Year() == chartYear {
		end = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	} else if now.Year() < chartYear {
		end = start
	}

	return chartYear, start, end
}

// opportunityInDashboardYTD reports whether an opportunity belongs to the
// dashboard YTD window: matched on Year (falling back to OfferDate's year),
// then bounded by OfferDate when one exists.
func opportunityInDashboardYTD(opp Opportunity, chartYear int, start time.Time, end time.Time) bool {
	year := opp.Year
	if year == 0 && !opp.OfferDate.IsZero() {
		year = opp.OfferDate.Year()
	}
	if year != chartYear {
		return false
	}

	if !opp.OfferDate.IsZero() {
		return !opp.OfferDate.Before(start) && opp.OfferDate.Before(end)
	}
	return true
}

func (a *App) buildCRMCustomerDashboardForRange(start, end, activeStart time.Time, yearLabel int) CRMCustomerDashboard {
	dashboard := CRMCustomerDashboard{}
	if a.db == nil {
		log.Printf("⚠️ CRM customer dashboard: database not initialized")
		return dashboard
	}

	linkIndex := a.buildCustomerLinkIndex()
	customers := linkIndex.customers
	dashboard.TotalCustomers = len(customers)

	customerRevenue := make(map[string]float64)
	customerActiveInvoices := make(map[string]int)
	customerOutstanding := make(map[string]float64)
	customerOverdue := make(map[string]float64)
	customerLastOrder := make(map[string]time.Time)
	activeCustomerSet := make(map[string]bool)
	invoicedByOrderID := make(map[string]float64)
	var totalCommercialValue float64
	commercialEvents := newCustomerCommercialEventCollector()

	var invoices []Invoice
	if err := a.db.Where("invoice_date >= ? AND invoice_date < ?", start, end).Find(&invoices).Error; err != nil {
		log.Printf("⚠️ CRM customer dashboard invoice aggregation failed: %v", err)
	}
	now := time.Now()
	for _, inv := range invoices {
		if !invoicePostedStatus(inv.Status) {
			continue
		}
		customer, ok := linkIndex.resolve(inv.CustomerID, inv.CustomerName)
		if !ok {
			continue
		}
		if strings.TrimSpace(inv.OrderID) != "" {
			invoicedByOrderID[inv.OrderID] += inv.GrandTotalBHD
		}

		if invoiceOpenStatus(inv.Status) {
			outstanding := invoiceOutstandingValue(inv)
			customerActiveInvoices[customer.ID]++
			customerOutstanding[customer.ID] += outstanding
			if daysOverdueFrom(now, inv.DueDate) > 30 {
				customerOverdue[customer.ID] += outstanding
			}
		}
	}

	var orders []Order
	if err := a.db.Where("order_date >= ? AND order_date < ?", start, end).Find(&orders).Error; err != nil {
		log.Printf("⚠️ CRM customer dashboard order aggregation failed: %v", err)
	}
	for _, order := range orders {
		if !commercialOrderStatus(order.Status) {
			continue
		}
		customer, ok := linkIndex.resolve(order.CustomerID, order.CustomerName)
		if !ok {
			continue
		}
		orderValue := customerCommercialOrderValue(order)
		commercialEvents.add(customerCommercialEvent{
			CustomerID: customer.ID,
			OfferID:    order.OfferID,
			Ref:        order.OrderNumber,
			Project:    firstNonEmpty(order.CustomerReference, order.OrderNumber),
			Status:     order.Status,
			Source:     "order",
			Date:       order.OrderDate,
			Value:      orderValue,
		},
			"order:"+order.ID,
			"order_number:"+order.OrderNumber,
			"offer:"+order.OfferID,
			"rfq:"+order.RFQID,
		)

		remainder := orderValue - invoicedByOrderID[order.ID]
		if remainder > 0.001 {
			customerOutstanding[customer.ID] += remainder
		}
		activeCustomerSet[customer.ID] = true
		if existing, ok := customerLastOrder[customer.ID]; !ok || order.OrderDate.After(existing) {
			customerLastOrder[customer.ID] = order.OrderDate
		}
	}

	for _, inv := range invoices {
		if !invoicePostedStatus(inv.Status) {
			continue
		}
		customer, ok := linkIndex.resolve(inv.CustomerID, inv.CustomerName)
		if !ok {
			continue
		}
		commercialEvents.add(customerCommercialEvent{
			CustomerID: customer.ID,
			OfferID:    inv.OfferID,
			Ref:        inv.InvoiceNumber,
			Project:    firstNonEmpty(inv.CustomerReference, inv.InvoiceNumber),
			Status:     inv.Status,
			Source:     "invoice",
			Date:       inv.InvoiceDate,
			Value:      inv.GrandTotalBHD,
		},
			"invoice:"+inv.ID,
			"invoice_number:"+inv.InvoiceNumber,
			"order:"+inv.OrderID,
			"offer:"+inv.OfferID,
			"rfq:"+inv.RfqID,
		)
	}

	var activeOrders []Order
	if err := a.db.Where("order_date >= ? AND order_date < ?", activeStart, end).Find(&activeOrders).Error; err == nil {
		for _, order := range activeOrders {
			if !commercialOrderStatus(order.Status) {
				continue
			}
			if customer, ok := linkIndex.resolve(order.CustomerID, order.CustomerName); ok {
				activeCustomerSet[customer.ID] = true
			}
		}
	}

	var offers []Offer
	if err := a.db.Where("quotation_date >= ? AND quotation_date < ?", start, end).Find(&offers).Error; err != nil {
		log.Printf("⚠️ CRM customer dashboard offer aggregation failed: %v", err)
	}
	for _, offer := range offers {
		if !commercialOfferStage(offer.Stage) || offer.TotalValueBHD <= 0 {
			continue
		}
		customer, ok := linkIndex.resolve(offer.CustomerID, offer.CustomerName)
		if !ok {
			continue
		}
		commercialEvents.add(customerCommercialEvent{
			CustomerID: customer.ID,
			OfferID:    offer.ID,
			Ref:        offer.OfferNumber,
			Project:    firstNonEmpty(offer.CustomerReference, offer.ProjectName, offer.OfferNumber),
			Status:     offer.Stage,
			Source:     "offer",
			Date:       offer.QuotationDate,
			Value:      offer.TotalValueBHD,
		},
			"offer:"+offer.ID,
			"offer_number:"+offer.OfferNumber,
			"rfq:"+offer.RFQID,
		)
	}

	var opportunities []Opportunity
	if err := a.db.Where("year = ? OR (offer_date >= ? AND offer_date < ?)", yearLabel, start, end).Find(&opportunities).Error; err != nil {
		log.Printf("⚠️ CRM customer dashboard opportunity aggregation failed: %v", err)
	}
	for _, opp := range opportunities {
		if !commercialOfferStage(opp.Stage) || opp.RevenueBHD <= 0 {
			continue
		}
		customer, ok := linkIndex.resolve(opp.CustomerID, opp.CustomerName)
		if !ok {
			continue
		}
		eventDate := opp.OfferDate
		if eventDate.IsZero() {
			eventDate = opp.CreatedAt
		}
		if eventDate.IsZero() && opp.Year > 0 {
			eventDate = time.Date(opp.Year, time.January, 1, 0, 0, 0, 0, time.UTC)
		}
		commercialEvents.add(customerCommercialEvent{
			CustomerID: customer.ID,
			OfferID:    opp.OfferID,
			Ref:        opp.FolderNumber,
			Project:    firstNonEmpty(opp.Title, opp.FolderName, opp.FolderNumber),
			Status:     opp.Stage,
			Source:     "opportunity",
			Date:       eventDate,
			Value:      opp.RevenueBHD,
		},
			"opportunity:"+opp.ID,
			"offer:"+opp.OfferID,
			"folder:"+opp.FolderNumber,
		)
	}

	for _, event := range commercialEvents.events {
		if event.Value <= 0 {
			continue
		}
		customerRevenue[event.CustomerID] += event.Value
		totalCommercialValue += event.Value
		activeCustomerSet[event.CustomerID] = true
	}

	dashboard.TotalRevenue = roundTo3(totalCommercialValue)
	dashboard.ActiveCustomers = len(activeCustomerSet)

	for _, amount := range customerOutstanding {
		dashboard.TotalOutstanding += amount
	}
	for _, amount := range customerOverdue {
		dashboard.OverdueAmount += amount
	}
	dashboard.TotalOutstanding = roundTo3(dashboard.TotalOutstanding)
	dashboard.OverdueAmount = roundTo3(dashboard.OverdueAmount)
	if dashboard.TotalOutstanding > 0 {
		dashboard.OverduePct = (dashboard.OverdueAmount / dashboard.TotalOutstanding) * 100
	}

	cards := make([]CustomerMetricCard, 0, len(customers))
	for _, c := range customers {
		lastOrderDate := c.LastOrderDate
		if dynamicLast, ok := customerLastOrder[c.ID]; ok {
			if lastOrderDate == nil || dynamicLast.After(*lastOrderDate) {
				last := dynamicLast
				lastOrderDate = &last
			}
		}
		cards = append(cards, CustomerMetricCard{
			ID:             c.ID,
			BusinessName:   c.BusinessName,
			CustomerType:   c.CustomerType,
			PaymentGrade:   c.PaymentGrade,
			TotalRevenue:   roundTo3(customerRevenue[c.ID]),
			ActiveInvoices: customerActiveInvoices[c.ID],
			OutstandingBHD: roundTo3(customerOutstanding[c.ID]),
			OverdueBHD:     roundTo3(customerOverdue[c.ID]),
			LastOrderDate:  lastOrderDate,
			City:           c.City,
		})
	}

	sort.Slice(cards, func(i, j int) bool {
		return cards[i].TotalRevenue > cards[j].TotalRevenue
	})
	dashboard.Customers = cards
	if len(cards) > 10 {
		dashboard.TopCustomers = cards[:10]
	} else {
		dashboard.TopCustomers = cards
	}

	if dashboard.TotalRevenue > 0 {
		var top3, top5, top10 float64
		for i, card := range cards {
			if i < 3 {
				top3 += card.TotalRevenue
			}
			if i < 5 {
				top5 += card.TotalRevenue
			}
			if i < 10 {
				top10 += card.TotalRevenue
			}
		}
		dashboard.Top3RevenuePct = (top3 / dashboard.TotalRevenue) * 100
		dashboard.Top5RevenuePct = (top5 / dashboard.TotalRevenue) * 100
		dashboard.Top10RevenuePct = (top10 / dashboard.TotalRevenue) * 100
	}

	for _, c := range customers {
		rev := customerRevenue[c.ID]
		switch c.PaymentGrade {
		case "A":
			dashboard.GradeACount++
			dashboard.GradeARevenue += rev
		case "B":
			dashboard.GradeBCount++
			dashboard.GradeBRevenue += rev
		case "C":
			dashboard.GradeCCount++
			dashboard.GradeCRevenue += rev
		case "D":
			dashboard.GradeDCount++
			dashboard.GradeDRevenue += rev
		}
	}
	dashboard.GradeARevenue = roundTo3(dashboard.GradeARevenue)
	dashboard.GradeBRevenue = roundTo3(dashboard.GradeBRevenue)
	dashboard.GradeCRevenue = roundTo3(dashboard.GradeCRevenue)
	dashboard.GradeDRevenue = roundTo3(dashboard.GradeDRevenue)

	log.Printf("📊 CRM Customer Dashboard (year=%d): %d customers, %.0f BHD commercial value, %.0f BHD outstanding",
		yearLabel, dashboard.TotalCustomers, dashboard.TotalRevenue, dashboard.TotalOutstanding)

	return dashboard
}

// GetCRMCustomerDashboard returns McKinsey-style customer command center metrics.
func (a *App) GetCRMCustomerDashboard() CRMCustomerDashboard {
	if err := a.requirePermission("customers:view"); err != nil {
		log.Printf("⚠️ GetCRMCustomerDashboard: permission denied")
		return CRMCustomerDashboard{}
	}
	if a.db == nil {
		log.Printf("⚠️ GetCRMCustomerDashboard: Database not initialized")
		return CRMCustomerDashboard{}
	}

	activityYear := a.latestCRMActivityYear()
	start := time.Date(activityYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(activityYear+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	activeStart := time.Date(activityYear-1, time.January, 1, 0, 0, 0, 0, time.UTC)
	return a.buildCRMCustomerDashboardForRange(start, end, activeStart, activityYear)
}

// GetCRMCustomerDashboardByYear returns the customer CRM dashboard filtered to a specific year.
func (a *App) GetCRMCustomerDashboardByYear(year int) CRMCustomerDashboard {
	if year == 0 {
		return a.GetCRMCustomerDashboard()
	}
	if err := a.requirePermission("customers:view"); err != nil {
		return CRMCustomerDashboard{}
	}
	if year < 2020 || year > time.Now().Year()+1 {
		return CRMCustomerDashboard{}
	}
	if a.db == nil {
		log.Printf("⚠️ GetCRMCustomerDashboardByYear: Database not initialized")
		return CRMCustomerDashboard{}
	}

	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	activeStart := time.Date(year-1, time.January, 1, 0, 0, 0, 0, time.UTC)
	return a.buildCRMCustomerDashboardForRange(start, end, activeStart, year)
}

// ============================================================================
// SETUP & CONFIGURATION API (Phase 4)
// ============================================================================

// ============================================================================
// DATA FIX UTILITIES
// ============================================================================

// FixDatabaseDates corrects dates that are 1 year off (2026 -> 2025)
// This is a one-time fix for data imported with incorrect years
func (a *App) FixDatabaseDates() (map[string]int, error) {
	// SECURITY: Admin-only permission for database manipulation
	if err := a.requirePermission("*"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database not initialized", "")
	}

	results := make(map[string]int)
	log.Println("🔧 Starting database date fix (subtracting 1 year from dates >= 2026)...")

	// Fix Purchase Orders dates
	poResult := a.db.Exec(`
		UPDATE purchase_orders
		SET po_date = date(po_date, '-1 year'),
		    expected_delivery = date(expected_delivery, '-1 year'),
		    updated_at = datetime('now')
		WHERE strftime('%Y', po_date) >= '2026'
	`)
	if poResult.Error != nil {
		log.Printf("⚠️ Error fixing PO dates: %v", poResult.Error)
	} else {
		results["purchase_orders"] = int(poResult.RowsAffected)
		log.Printf("✅ Fixed %d purchase order dates", poResult.RowsAffected)
	}

	// Fix Invoices dates
	invResult := a.db.Exec(`
		UPDATE invoices
		SET invoice_date = date(invoice_date, '-1 year'),
		    due_date = date(due_date, '-1 year'),
		    updated_at = datetime('now')
		WHERE strftime('%Y', invoice_date) >= '2026'
	`)
	if invResult.Error != nil {
		log.Printf("⚠️ Error fixing invoice dates: %v", invResult.Error)
	} else {
		results["invoices"] = int(invResult.RowsAffected)
		log.Printf("✅ Fixed %d invoice dates", invResult.RowsAffected)
	}

	// Fix Supplier Invoices dates
	siResult := a.db.Exec(`
		UPDATE supplier_invoices
		SET invoice_date = date(invoice_date, '-1 year'),
		    due_date = date(due_date, '-1 year'),
		    updated_at = datetime('now')
		WHERE strftime('%Y', invoice_date) >= '2026'
	`)
	if siResult.Error != nil {
		log.Printf("⚠️ Error fixing supplier invoice dates: %v", siResult.Error)
	} else {
		results["supplier_invoices"] = int(siResult.RowsAffected)
		log.Printf("✅ Fixed %d supplier invoice dates", siResult.RowsAffected)
	}

	// Fix Orders dates
	orderResult := a.db.Exec(`
		UPDATE orders
		SET order_date = date(order_date, '-1 year'),
		    required_date = date(required_date, '-1 year'),
		    updated_at = datetime('now')
		WHERE strftime('%Y', order_date) >= '2026'
	`)
	if orderResult.Error != nil {
		log.Printf("⚠️ Error fixing order dates: %v", orderResult.Error)
	} else {
		results["orders"] = int(orderResult.RowsAffected)
		log.Printf("✅ Fixed %d order dates", orderResult.RowsAffected)
	}

	// Fix Payments dates
	payResult := a.db.Exec(`
		UPDATE payments
		SET payment_date = date(payment_date, '-1 year'),
		    updated_at = datetime('now')
		WHERE strftime('%Y', payment_date) >= '2026'
	`)
	if payResult.Error != nil {
		log.Printf("⚠️ Error fixing payment dates: %v", payResult.Error)
	} else {
		results["payments"] = int(payResult.RowsAffected)
		log.Printf("✅ Fixed %d payment dates", payResult.RowsAffected)
	}

	// Fix Supplier Payments dates
	spResult := a.db.Exec(`
		UPDATE supplier_payments
		SET payment_date = date(payment_date, '-1 year'),
		    updated_at = datetime('now')
		WHERE strftime('%Y', payment_date) >= '2026'
	`)
	if spResult.Error != nil {
		log.Printf("⚠️ Error fixing supplier payment dates: %v", spResult.Error)
	} else {
		results["supplier_payments"] = int(spResult.RowsAffected)
		log.Printf("✅ Fixed %d supplier payment dates", spResult.RowsAffected)
	}

	// Fix Delivery Notes dates
	dnResult := a.db.Exec(`
		UPDATE delivery_notes
		SET delivery_date = date(delivery_date, '-1 year'),
		    updated_at = datetime('now')
		WHERE strftime('%Y', delivery_date) >= '2026'
	`)
	if dnResult.Error != nil {
		log.Printf("⚠️ Error fixing delivery note dates: %v", dnResult.Error)
	} else {
		results["delivery_notes"] = int(dnResult.RowsAffected)
		log.Printf("✅ Fixed %d delivery note dates", dnResult.RowsAffected)
	}

	// Fix GRN dates
	grnResult := a.db.Exec(`
		UPDATE goods_received_notes
		SET received_date = date(received_date, '-1 year'),
		    updated_at = datetime('now')
		WHERE strftime('%Y', received_date) >= '2026'
	`)
	if grnResult.Error != nil {
		log.Printf("⚠️ Error fixing GRN dates: %v", grnResult.Error)
	} else {
		results["goods_received_notes"] = int(grnResult.RowsAffected)
		log.Printf("✅ Fixed %d GRN dates", grnResult.RowsAffected)
	}

	// Fix RFQ dates
	rfqResult := a.db.Exec(`
		UPDATE rfqs
		SET rfq_date = date(rfq_date, '-1 year'),
		    due_date = date(due_date, '-1 year'),
		    updated_at = datetime('now')
		WHERE strftime('%Y', rfq_date) >= '2026'
	`)
	if rfqResult.Error != nil {
		log.Printf("⚠️ Error fixing RFQ dates: %v", rfqResult.Error)
	} else {
		results["rfqs"] = int(rfqResult.RowsAffected)
		log.Printf("✅ Fixed %d RFQ dates", rfqResult.RowsAffected)
	}

	// Fix Offers dates
	offerResult := a.db.Exec(`
		UPDATE offers
		SET offer_date = date(offer_date, '-1 year'),
		    valid_until = date(valid_until, '-1 year'),
		    updated_at = datetime('now')
		WHERE strftime('%Y', offer_date) >= '2026'
	`)
	if offerResult.Error != nil {
		log.Printf("⚠️ Error fixing offer dates: %v", offerResult.Error)
	} else {
		results["offers"] = int(offerResult.RowsAffected)
		log.Printf("✅ Fixed %d offer dates", offerResult.RowsAffected)
	}

	log.Println("🔧 Database date fix complete!")
	return results, nil
}

// FixPurchaseOrderSupplierNames populates supplier_name in all POs that have a valid supplier_id
func (a *App) FixPurchaseOrderSupplierNames() (int, error) {
	// SECURITY: Admin-only permission for database manipulation
	if err := a.requirePermission("*"); err != nil {
		return 0, err
	}
	if a.db == nil {
		return 0, newError("DB_NOT_INITIALIZED", "Database not initialized", "")
	}

	log.Println("🔧 Starting supplier name fix for purchase orders...")

	// Get all POs with supplier IDs but no/empty supplier names
	var pos []PurchaseOrder
	if err := a.db.Where("supplier_id IS NOT NULL AND supplier_id != '' AND (supplier_name IS NULL OR supplier_name = '')").Find(&pos).Error; err != nil {
		return 0, newError("DB_QUERY_FAILED", "Failed to get POs for fixing", err.Error())
	}

	if len(pos) == 0 {
		log.Println("✅ No POs need supplier name fixing")
		return 0, nil
	}

	// Collect unique supplier IDs
	supplierIDs := make([]string, 0)
	seenIDs := make(map[string]bool)
	for _, po := range pos {
		if po.SupplierID != "" && !seenIDs[po.SupplierID] {
			supplierIDs = append(supplierIDs, po.SupplierID)
			seenIDs[po.SupplierID] = true
		}
	}

	// Fetch suppliers
	var suppliers []SupplierMaster
	if err := a.db.Where("id IN ?", supplierIDs).Find(&suppliers).Error; err != nil {
		return 0, newError("DB_QUERY_FAILED", "Failed to get suppliers", err.Error())
	}

	// Build map
	supplierMap := make(map[string]string)
	for _, s := range suppliers {
		supplierMap[s.ID] = s.SupplierName
	}

	// Update each PO
	fixed := 0
	for _, po := range pos {
		if name, ok := supplierMap[po.SupplierID]; ok && name != "" {
			if err := a.db.Model(&PurchaseOrder{}).Where("id = ?", po.ID).Update("supplier_name", name).Error; err != nil {
				log.Printf("⚠️ Failed to update PO %s: %v", po.PONumber, err)
				continue
			}
			fixed++
		}
	}

	log.Printf("✅ Fixed %d purchase order supplier names", fixed)
	return fixed, nil
}

// RunAllDataFixes runs safe data fix utilities at once.
// Date rewriting is intentionally excluded because valid 2026+ production data exists.
func (a *App) RunAllDataFixes() (map[string]any, error) {
	// SECURITY: Admin-only permission for database manipulation
	if err := a.requirePermission("*"); err != nil {
		return nil, err
	}
	results := make(map[string]any)

	// Fix supplier names
	supplierFixed, err := a.FixPurchaseOrderSupplierNames()
	if err != nil {
		log.Printf("⚠️ Supplier name fix error: %v", err)
	}
	results["supplier_names_fixed"] = supplierFixed

	// Backfill RFQ document tracking
	rfqFixed, err := a.BackfillRFQDocumentTracking()
	if err != nil {
		log.Printf("⚠️ RFQ document tracking backfill error: %v", err)
	}
	results["rfq_document_tracking_fixed"] = rfqFixed
	results["date_fixes_skipped"] = "legacy date rewrite disabled for production safety"

	log.Println("🔧 All data fixes complete!")
	return results, nil
}

// BackfillRFQDocumentTracking generates document_hash and sets source_doc_path for existing RFQs
// Provides duplicate detection even for manually entered RFQs
func (a *App) BackfillRFQDocumentTracking() (int, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return 0, err
	}
	if a.db == nil {
		return 0, newError("DB_NOT_INITIALIZED", "Database not initialized", "")
	}

	log.Println("🔧 Starting RFQ document tracking backfill...")

	// Get all RFQs where document_hash is NULL or empty
	var rfqs []RFQData
	if err := a.db.Where("document_hash IS NULL OR document_hash = ''").Find(&rfqs).Error; err != nil {
		return 0, newError("DB_QUERY_FAILED", "Failed to get RFQs for backfill", err.Error())
	}

	if len(rfqs) == 0 {
		log.Println("✅ No RFQs need document tracking backfill")
		return 0, nil
	}

	log.Printf("📋 Found %d RFQs needing document tracking", len(rfqs))

	// Process each RFQ
	fixed := 0
	for _, rfq := range rfqs {
		// Generate document_hash from content (client + project + value)
		// This provides duplicate detection even for manually entered RFQs
		hashInput := fmt.Sprintf("%s|%s|%.3f", rfq.Client, rfq.Project, rfq.Value)
		hashBytes := sha256.Sum256([]byte(hashInput))
		hash := fmt.Sprintf("%x", hashBytes)

		// Set source_doc_path to "manual_entry" if empty
		sourcePath := rfq.SourceDocPath
		if sourcePath == "" {
			sourcePath = "manual_entry"
		}

		// Update both fields
		if err := a.db.Model(&RFQData{}).Where("id = ?", rfq.ID).Updates(map[string]any{
			"document_hash":   hash,
			"source_doc_path": sourcePath,
		}).Error; err != nil {
			log.Printf("⚠️ Failed to update RFQ #%d: %v", rfq.ID, err)
			continue
		}

		fixed++
		if fixed%10 == 0 {
			log.Printf("✅ Progress: %d/%d RFQs updated", fixed, len(rfqs))
		}
	}

	log.Printf("✅ Backfilled document tracking for %d RFQs", fixed)
	return fixed, nil
}

// BackfillOfferItemCostBreakdown populates extended cost breakdown fields in offer_items
// from matching costing_line_items data. Runs once at startup.
func (a *App) BackfillOfferItemCostBreakdown() (map[string]any, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return nil, err
	}
	log.Println("🔧 Starting offer_items cost breakdown backfill...")

	// First, ensure the columns exist (they should from database.go migration)
	// But SQLite may need explicit ALTER TABLE if the schema changed
	log.Println("📊 Checking offer_items schema...")

	// Add missing columns if they don't exist
	columns := []struct {
		name     string
		datatype string
	}{
		{"exchange_rate", "REAL DEFAULT 0"},
		{"fob_bhd", "REAL DEFAULT 0"},
		{"freight_bhd", "REAL DEFAULT 0"},
		{"insurance", "REAL DEFAULT 0"},
		{"customs_percent", "REAL DEFAULT 0"},
		{"customs_bhd", "REAL DEFAULT 0"},
		{"handling_percent", "REAL DEFAULT 0"},
		{"handling_bhd", "REAL DEFAULT 0"},
		{"finance_percent", "REAL DEFAULT 0"},
		{"finance_bhd", "REAL DEFAULT 0"},
		{"other_costs", "REAL DEFAULT 0"},
		{"user_price", "REAL DEFAULT 0"},
		{"user_price_set", "INTEGER DEFAULT 0"},
	}

	for _, col := range columns {
		// SECURITY FIX: Validate column name to prevent SQL injection
		if !isValidSQLIdentifier(col.name) {
			log.Printf("  ⚠️ Invalid column name rejected: %s", col.name)
			continue
		}

		// Try to add column (will fail silently if exists)
		alterSQL := fmt.Sprintf("ALTER TABLE offer_items ADD COLUMN %s %s", col.name, col.datatype)
		if err := a.db.Exec(alterSQL).Error; err != nil {
			// Column likely exists, ignore error
			log.Printf("  Column %s: already exists or error (%v)", col.name, err)
		} else {
			log.Printf("  ✅ Added column: %s", col.name)
		}
	}

	// Get all offer_items
	var offerItems []OfferItem
	if err := a.db.Find(&offerItems).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch offer_items: %w", err)
	}

	log.Printf("📦 Found %d offer_items to process", len(offerItems))

	// Get all costing_line_items for matching
	type CostingLineItemRaw struct {
		ID                string  `gorm:"column:id"`
		CostingSheetID    int     `gorm:"column:costing_sheet_id"`
		ProductNumber     int     `gorm:"column:product_number"`
		Equipment         string  `gorm:"column:equipment"`
		Model             string  `gorm:"column:model"`
		Specification     string  `gorm:"column:specification"`
		Supplier          string  `gorm:"column:supplier"`
		Quantity          float64 `gorm:"column:quantity"`
		FobEUR            float64 `gorm:"column:fob_eur"`
		ExchangeRate      float64 `gorm:"column:exchange_rate"`
		TotalCostBHD      float64 `gorm:"column:total_cost_bhd"`
		MarkupPercent     float64 `gorm:"column:markup_percent"`
		SellingPriceBHD   float64 `gorm:"column:selling_price_bhd"`
		TotalSuggestedBHD float64 `gorm:"column:total_suggested_bhd"`
	}

	var costingItems []CostingLineItemRaw
	if err := a.db.Table("costing_line_items").Find(&costingItems).Error; err != nil {
		log.Printf("⚠️ Failed to fetch costing_line_items: %v", err)
		// Continue with defaults
	} else {
		log.Printf("📊 Found %d costing_line_items for matching", len(costingItems))
	}

	// Build quick lookup map by equipment name and model
	costingMap := make(map[string]*CostingLineItemRaw)
	for i := range costingItems {
		item := &costingItems[i]
		// Index by equipment name
		if item.Equipment != "" {
			key := strings.ToLower(strings.TrimSpace(item.Equipment))
			if _, exists := costingMap[key]; !exists {
				costingMap[key] = item
			}
		}
		// Also index by model
		if item.Model != "" {
			key := "model:" + strings.ToLower(strings.TrimSpace(item.Model))
			if _, exists := costingMap[key]; !exists {
				costingMap[key] = item
			}
		}
	}

	log.Printf("🗺️ Built lookup map with %d keys", len(costingMap))

	// Process each offer_item
	updated := 0
	matched := 0
	defaulted := 0

	for i := range offerItems {
		item := &offerItems[i]

		// Skip if already has exchange_rate populated
		if item.ExchangeRate > 0 {
			continue
		}

		// Try to find matching costing item
		var costingMatch *CostingLineItemRaw

		// Match by equipment name
		if item.Equipment != "" {
			key := strings.ToLower(strings.TrimSpace(item.Equipment))
			if match, ok := costingMap[key]; ok {
				costingMatch = match
			}
		}

		// If no match, try by model
		if costingMatch == nil && item.Model != "" {
			key := "model:" + strings.ToLower(strings.TrimSpace(item.Model))
			if match, ok := costingMap[key]; ok {
				costingMatch = match
			}
		}

		// If no match, try by product_code
		if costingMatch == nil && item.ProductCode != "" {
			key := "model:" + strings.ToLower(strings.TrimSpace(item.ProductCode))
			if match, ok := costingMap[key]; ok {
				costingMatch = match
			}
		}

		// If match found, copy data
		if costingMatch != nil {
			matched++

			// Copy exchange rate
			if costingMatch.ExchangeRate > 0 {
				item.ExchangeRate = normalizeExchangeRateToBHD("EUR", costingMatch.ExchangeRate)
			} else {
				item.ExchangeRate = activeOverlay.ExchangeRateToBase("EUR")
			}

			// Calculate FOB in BHD
			if costingMatch.FobEUR > 0 {
				item.FobBHD = costingMatch.FobEUR * item.ExchangeRate
			}

			// If we have total_cost_bhd, derive other costs
			if costingMatch.TotalCostBHD > 0 && item.FobBHD > 0 {
				landedCost := item.FobBHD

				// Assume standard percentages (Bahrain typical)
				item.CustomsPercent = 5.0
				item.CustomsBHD = landedCost * 0.05

				item.HandlingPercent = 4.0
				item.HandlingBHD = (landedCost + item.CustomsBHD) * 0.04

				item.FinancePercent = 1.0
				item.FinanceBHD = (landedCost + item.CustomsBHD + item.HandlingBHD) * 0.01

				// Calculate freight as difference
				totalDerived := landedCost + item.CustomsBHD + item.HandlingBHD + item.FinanceBHD
				if costingMatch.TotalCostBHD > totalDerived {
					item.FreightBHD = costingMatch.TotalCostBHD - totalDerived
				}

				// Set total_cost to match
				item.TotalCost = costingMatch.TotalCostBHD
			}

			// Copy currency if available
			if item.Currency == "" {
				item.Currency = "EUR" // Most costing is in EUR
			}

			log.Printf("  ✅ Matched: %s (model: %s) -> costing item", item.Equipment, item.Model)

		} else {
			// No match - use reasonable defaults
			defaulted++

			item.ExchangeRate = activeOverlay.ExchangeRateToBase("EUR")

			// If FOB exists, calculate FOB in BHD
			if item.FOB > 0 {
				item.FobBHD = item.FOB * item.ExchangeRate

				// Apply standard Bahrain cost structure
				landedCost := item.FobBHD

				// Freight (estimate 8% of FOB)
				if item.Freight > 0 {
					item.FreightBHD = item.Freight * item.ExchangeRate
				} else {
					item.FreightBHD = landedCost * 0.08
				}

				// Insurance (0.5% of CIF)
				item.Insurance = (landedCost + item.FreightBHD) * 0.005

				// Customs (5%)
				item.CustomsPercent = 5.0
				item.CustomsBHD = (landedCost + item.FreightBHD + item.Insurance) * 0.05

				// Handling (4%)
				item.HandlingPercent = 4.0
				cifPlusDuty := landedCost + item.FreightBHD + item.Insurance + item.CustomsBHD
				item.HandlingBHD = cifPlusDuty * 0.04

				// Finance (1%)
				item.FinancePercent = 1.0
				item.FinanceBHD = (cifPlusDuty + item.HandlingBHD) * 0.01

				// Total cost
				item.TotalCost = cifPlusDuty + item.HandlingBHD + item.FinanceBHD

				log.Printf("  🔧 Default: %s (FOB: %.3f EUR -> %.3f BHD)", item.Equipment, item.FOB, item.FobBHD)
			} else {
				log.Printf("  ⚠️ No data: %s (no FOB, no match)", item.Equipment)
			}

			if item.Currency == "" {
				item.Currency = "EUR"
			}
		}

		// Save updated item
		if err := a.db.Save(&item).Error; err != nil {
			log.Printf("⚠️ Failed to save offer_item %s: %v", item.ID, err)
		} else {
			updated++
		}
	}

	log.Printf("✅ Backfill complete: %d updated, %d matched, %d defaulted", updated, matched, defaulted)

	return map[string]any{
		"total_items":    len(offerItems),
		"updated":        updated,
		"matched":        matched,
		"defaulted":      defaulted,
		"already_filled": len(offerItems) - updated,
	}, nil
}

// RecalculateInvoiceItemCosts fixes invoice_items where cost data causes negative margins
// It recalculates costs based on selling price using a standard 20% margin assumption
func (a *App) RecalculateInvoiceItemCosts() (map[string]any, error) {
	if err := a.requirePermission("finance:update"); err != nil {
		return nil, err
	}
	log.Println("=== RecalculateInvoiceItemCosts: Fixing negative margin data ===")

	// Target margin assumption (Acme Instrumentation typical gross margin from FS2024 is ~18-20%)
	const targetMarginPercent = 20.0
	eurToBhdRate := activeOverlay.ExchangeRateToBase("EUR")

	var invoiceItems []DBInvoiceItem
	if err := a.db.Find(&invoiceItems).Error; err != nil {
		return nil, fmt.Errorf("failed to load invoice items: %w", err)
	}

	log.Printf("Processing %d invoice_items...", len(invoiceItems))

	var updated, fixed, skipped int
	for _, item := range invoiceItems {
		// Skip items with no selling price
		if item.Rate <= 0 {
			skipped++
			continue
		}

		// Calculate cost from selling price: cost = selling / (1 + margin%)
		sellingPrice := item.Rate * item.Quantity
		estimatedCost := sellingPrice / (1 + targetMarginPercent/100)

		// Check if current data has negative margin (cost > selling)
		needsFix := item.TotalCost > sellingPrice || item.TotalCost <= 0

		if needsFix {
			// Fix the cost data
			item.TotalCost = estimatedCost
			item.FOB = estimatedCost / eurToBhdRate // Estimate FOB in EUR
			item.MarginPercent = targetMarginPercent
			item.TotalPrice = sellingPrice

			if err := a.db.Model(&DBInvoiceItem{}).Where("id = ?", item.ID).Updates(map[string]any{
				"total_cost":     item.TotalCost,
				"fob":            item.FOB,
				"margin_percent": item.MarginPercent,
				"total_price":    item.TotalPrice,
			}).Error; err != nil {
				log.Printf("⚠️ Failed to update item %s: %v", item.ID, err)
			} else {
				fixed++
			}
		} else {
			// Margin is valid, just ensure margin_percent is set correctly
			if item.TotalCost > 0 {
				actualMargin := (sellingPrice - item.TotalCost) / sellingPrice * 100
				if item.MarginPercent != actualMargin {
					a.db.Model(&DBInvoiceItem{}).Where("id = ?", item.ID).Update("margin_percent", actualMargin)
				}
			}
		}
		updated++
	}

	log.Printf("✅ Invoice items: %d processed, %d fixed (negative margins), %d skipped (no rate)", updated, fixed, skipped)

	// Now recalculate invoice header totals
	log.Println("Recalculating invoice header totals...")

	var invoices []Invoice
	if err := a.db.Preload("Items").Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to load invoices: %w", err)
	}

	var invoicesUpdated int
	for _, inv := range invoices {
		var totalCost float64
		var subtotal float64

		for _, item := range inv.Items {
			subtotal += item.Rate * item.Quantity
			totalCost += item.TotalCost
		}

		grossMargin := subtotal - totalCost
		grossMarginPercent := 0.0
		if subtotal > 0 {
			grossMarginPercent = (grossMargin / subtotal) * 100
		}

		if err := a.db.Model(&Invoice{}).Where("id = ?", inv.ID).Updates(map[string]any{
			"total_supplier_cost_bhd": totalCost,
			"gross_margin_bhd":        grossMargin,
			"gross_margin_percent":    grossMarginPercent,
		}).Error; err != nil {
			log.Printf("⚠️ Failed to update invoice %s: %v", inv.InvoiceNumber, err)
		} else {
			invoicesUpdated++
		}
	}

	log.Printf("✅ Updated %d invoice headers with recalculated margins", invoicesUpdated)

	return map[string]any{
		"items_processed":  updated,
		"items_fixed":      fixed,
		"items_skipped":    skipped,
		"invoices_updated": invoicesUpdated,
		"target_margin":    targetMarginPercent,
	}, nil
}
