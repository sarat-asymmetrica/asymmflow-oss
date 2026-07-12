package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"ph_holdings_app/integration"
)

func (a *App) PredictPayment(customer Customer) (PaymentPrediction, error) {
	if err := a.requirePermission("predictions:create"); err != nil {
		return PaymentPrediction{}, err
	}
	// Validate customer data
	validationResult := a.ValidateCustomer(customer)
	if !validationResult.Valid {
		errMsg := fmt.Sprintf("Invalid customer data: %v", validationResult.Errors)
		return PaymentPrediction{}, newError("VALIDATION_FAILED", "Customer validation failed", errMsg)
	}

	if a.db == nil {
		return PaymentPrediction{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	log.Printf("Predicting payment for customer: %s", customer.BusinessName)

	// Perform prediction
	predictor := NewPaymentPredictor(&customer)
	prediction := predictor.Predict(&customer)

	// Save to database
	record := PredictionRecord{
		CustomerID:    prediction.CustomerID,
		CustomerName:  prediction.CustomerName,
		Grade:         prediction.Grade,
		PredictedDays: prediction.PredictedDays,
		Confidence:    prediction.Confidence,
		R1:            prediction.ThreeRegimes.R1,
		R2:            prediction.ThreeRegimes.R2,
		R3:            prediction.ThreeRegimes.R3,
	}

	result := a.db.Create(&record)
	if result.Error != nil {
		log.Printf("Warning: Failed to save prediction: %v", result.Error)
		// Don't fail the entire request - prediction was successful, just saving failed
		return prediction, newError("DB_SAVE_WARNING", "Prediction completed but failed to save to database", result.Error.Error())
	}

	log.Printf("Prediction saved with ID: %s", record.ID)
	return prediction, nil
}

// BatchPredict processes multiple customers
func (a *App) BatchPredict(customers []Customer) (BatchResult, error) {
	if err := a.requirePermission("predictions:create"); err != nil {
		return BatchResult{}, err
	}
	if len(customers) == 0 {
		return BatchResult{}, newError("INVALID_INPUT", "No customers provided for batch prediction", "")
	}

	if a.db == nil {
		return BatchResult{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	log.Printf("Batch predicting %d customers", len(customers))

	// Convert to pointer slice for batch processor
	customerPtrs := make([]*Customer, len(customers))
	for i := range customers {
		customerPtrs[i] = &customers[i]
	}

	// Perform batch prediction
	predictions := BatchPredictCustomers(customerPtrs)
	summary := SummarizeBatch(customerPtrs, predictions)

	// Save all predictions to database
	savedCount := 0
	for _, pred := range predictions {
		record := PredictionRecord{
			CustomerID:    pred.CustomerID,
			CustomerName:  pred.CustomerName,
			Grade:         pred.Grade,
			PredictedDays: pred.PredictedDays,
			Confidence:    pred.Confidence,
			R1:            pred.ThreeRegimes.R1,
			R2:            pred.ThreeRegimes.R2,
			R3:            pred.ThreeRegimes.R3,
		}
		if err := a.db.Create(&record).Error; err != nil {
			log.Printf("Warning: Failed to save prediction for %s: %v", pred.CustomerName, err)
		} else {
			savedCount++
		}
	}

	log.Printf("Batch prediction complete: %d/%d customers processed and saved", savedCount, len(predictions))

	return BatchResult{
		Summary:     summary,
		Predictions: predictions,
	}, nil
}

// ============================================================================
// Server-initiated payment-prediction generation (PH G5, Wave 8 Bucket G tail)
// ============================================================================
// The payment predictor is a deterministic geometric model that reads ONLY the
// in-struct Customer fields — it does not query the DB itself. Its #1 signal is
// PaymentHistory (per-invoice days-to-pay); with an empty history it encodes as
// "all stable" and produces a misleadingly optimistic Grade A. So these helpers
// build the Customer from REAL DB history and refuse to fabricate a prediction
// when there is no payment signal.

// generatePredictionForResolvedCustomer builds the predictor input from a
// customer's real payment/order history and persists a PredictionRecord.
// Internal (no RBAC) so it can be called from read paths without the
// predictions:create permission. Returns (nil, nil) when the customer has no
// payment history — the caller should leave the Insights tab empty rather
// than show a guessed grade.
func (a *App) generatePredictionForResolvedCustomer(idx *customerLinkIndex, customer CustomerMaster) (*PredictionRecord, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Real payment-days history (the dominant predictor input).
	history := a.linkedPaymentHistoryForCustomer(idx, customer, 200)
	paymentDays := make([]int, 0, len(history))
	for _, h := range history {
		if h.DaysToPayment > 0 {
			paymentDays = append(paymentDays, h.DaysToPayment)
		}
	}

	// QUALITY GATE: payment behavior cannot be honestly predicted without payment
	// history. Skip (empty tab) rather than surface an optimistic default grade.
	if len(paymentDays) == 0 {
		return nil, nil
	}

	// Order history (values) for the order-variability encoding.
	orders := a.linkedOrdersForCustomer(idx, customer)
	orderValues := make([]float64, 0, len(orders))
	for _, o := range orders {
		if commercialOrderStatus(o.Status) {
			orderValues = append(orderValues, customerCommercialOrderValue(o))
		}
	}

	input := Customer{
		ID:             customer.ID,
		BusinessName:   customer.BusinessName,
		OrderHistory:   orderValues,
		PaymentHistory: paymentDays,
		RelationYears:  customer.RelationYears,
		Industry:       customer.Industry,
		Country:        customer.Country,
		IsEmergency:    boolToInt(customer.IsEmergencyOnly),
		HasABB:         boolToInt(customer.HasABBCompetition),
		DisputeCount:   customer.DisputeCount,
	}
	if len(orderValues) > 0 {
		input.OrderValue = orderValues[len(orderValues)-1] // most recent commercial order
	}

	prediction := NewPaymentPredictor(&input).Predict(&input)
	record := PredictionRecord{
		CustomerID:    customer.ID,
		CustomerName:  customer.BusinessName,
		Grade:         prediction.Grade,
		PredictedDays: prediction.PredictedDays,
		Confidence:    prediction.Confidence,
		R1:            prediction.ThreeRegimes.R1,
		R2:            prediction.ThreeRegimes.R2,
		R3:            prediction.ThreeRegimes.R3,
	}
	if err := a.db.Create(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to save prediction for %s: %w", customer.BusinessName, err)
	}
	log.Printf("🔮 Generated prediction for %s: Grade %s (%d days, %.0f%% conf, %d payment samples)",
		customer.BusinessName, record.Grade, record.PredictedDays, record.Confidence*100, len(paymentDays))
	return &record, nil
}

// generatePredictionForCustomer resolves a customer by id/customer_id and
// generates a prediction. Internal (no RBAC); used by the RBAC'd
// RecomputeCustomerPrediction endpoint.
func (a *App) generatePredictionForCustomer(customerID string) (*PredictionRecord, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	idx := a.buildCustomerLinkIndex()
	customer, ok := idx.resolve(customerID, customerID)
	if !ok {
		return nil, fmt.Errorf("customer not found: %s", customerID)
	}
	return a.generatePredictionForResolvedCustomer(idx, customer)
}

// RecomputeCustomerPrediction forces a fresh prediction for a customer. This is
// the explicit, user-initiated recompute path and requires predictions:create.
// Returns (nil, nil) when the customer has no payment history to predict from.
func (a *App) RecomputeCustomerPrediction(customerID string) (*PredictionRecord, error) {
	if err := a.requirePermission("predictions:create"); err != nil {
		return nil, err
	}
	if customerID == "" {
		return nil, newError("INVALID_INPUT", "Customer ID is required", "")
	}
	return a.generatePredictionForCustomer(customerID)
}

// GetHistory retrieves recent prediction history
func (a *App) GetHistory(limit int) ([]PredictionRecord, error) {
	if err := a.requirePermission("predictions:view"); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}

	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var records []PredictionRecord
	result := a.db.Order("created_at desc").Limit(limit).Find(&records)

	if result.Error != nil {
		log.Printf("Error fetching history: %v", result.Error)
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve history", result.Error.Error())
	}

	log.Printf("Retrieved %d prediction records", len(records))
	return records, nil
}

// GetCustomerHistory retrieves predictions for a specific customer
func (a *App) GetCustomerHistory(customerID string) ([]PredictionRecord, error) {
	if err := a.requirePermission("predictions:view"); err != nil {
		return nil, err
	}
	if customerID == "" {
		return nil, newError("INVALID_INPUT", "Customer ID is required", "")
	}

	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var records []PredictionRecord
	result := a.db.Where("customer_id = ?", customerID).
		Order("created_at desc").
		Find(&records)

	if result.Error != nil {
		log.Printf("Error fetching customer history: %v", result.Error)
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve customer history", result.Error.Error())
	}

	log.Printf("Retrieved %d records for customer %s", len(records), customerID)
	return records, nil
}

// GetStatistics retrieves summary statistics
func (a *App) GetStatistics() (Statistics, error) {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return Statistics{}, err
	}
	if a.db == nil {
		return Statistics{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var stats Statistics

	// Total predictions
	if err := a.db.Model(&PredictionRecord{}).Count(&stats.TotalPredictions).Error; err != nil {
		return Statistics{}, newError("DB_QUERY_FAILED", "Failed to count predictions", err.Error())
	}

	// Grade distribution
	var gradeResults []struct {
		Grade string
		Count int64
	}

	if err := a.db.Model(&PredictionRecord{}).
		Select("grade, count(*) as count").
		Group("grade").
		Scan(&gradeResults).Error; err != nil {
		return Statistics{}, newError("DB_QUERY_FAILED", "Failed to get grade distribution", err.Error())
	}

	stats.GradeDistribution = make(map[string]int)
	for _, gr := range gradeResults {
		stats.GradeDistribution[gr.Grade] = int(gr.Count)
	}

	// Average confidence
	if err := a.db.Model(&PredictionRecord{}).
		Select("AVG(confidence) as avg_conf").
		Scan(&stats.AvgConfidence).Error; err != nil {
		return Statistics{}, newError("DB_QUERY_FAILED", "Failed to calculate average confidence", err.Error())
	}

	// Average predicted days
	if err := a.db.Model(&PredictionRecord{}).
		Select("AVG(predicted_days) as avg_days").
		Scan(&stats.AvgPredictedDays).Error; err != nil {
		return Statistics{}, newError("DB_QUERY_FAILED", "Failed to calculate average days", err.Error())
	}

	log.Printf("Statistics: %d total predictions, avg confidence: %.2f",
		stats.TotalPredictions, stats.AvgConfidence)

	return stats, nil
}

// DashboardStats represents dashboard/sales statistics for UI
type DashboardStats struct {
	ActiveRFQs       int     `json:"active_rfqs"`
	ActiveOrders     int     `json:"active_orders"`
	PendingReview    int     `json:"pending_review"`
	UrgentCount      int     `json:"urgent_count"`
	AvgVelocityDays  float64 `json:"avg_velocity_days"`
	WinRate          float64 `json:"win_rate"`
	TotalRevenue     float64 `json:"total_revenue"`
	MonthGrowth      float64 `json:"month_growth"`
	SystemHealth     string  `json:"system_health"`
	Runway           float64 `json:"runway_months"`
	OutstandingAR    float64 `json:"outstanding_ar"`   // Accounts receivable (unpaid invoices)
	ARDaysOverdue    int     `json:"ar_days_overdue"`  // Average days overdue
	PendingInvoices  int     `json:"pending_invoices"` // Count of unpaid invoices
	ActiveCustomers  int     `json:"active_customers"` // Total customers in database
	RevenueMeta      string  `json:"revenue_meta"`
	ActivityYear     int     `json:"activity_year"`
	PipelineValueBHD float64 `json:"pipeline_value_bhd"` // Open opportunity value
	CollectionRate   float64 `json:"collection_rate"`    // Collected value / collectible value
	CashBalanceBHD   float64 `json:"cash_balance_bhd"`   // Latest bank statement closing balances
	CashPositionNote string  `json:"cash_position_note"`
	FreshStartDate   string  `json:"fresh_start_date"`
}

type orderReceivableExposureRow struct {
	OrderValueBHD float64 `gorm:"column:order_value_bhd"`
	InvoicedBHD   float64 `gorm:"column:invoiced_bhd"`
}

func (a *App) calculateUninvoicedOrderExposure(start, end time.Time) (float64, int, error) {
	if a.db == nil {
		return 0, 0, fmt.Errorf("database connection not available")
	}

	var rows []orderReceivableExposureRow
	err := a.db.Model(&Order{}).
		Select(`CASE
				WHEN COALESCE(orders.grand_total_bhd, 0) > 0 THEN orders.grand_total_bhd
				ELSE COALESCE(orders.total_value_bhd, 0)
			END AS order_value_bhd,
			COALESCE((
				SELECT SUM(COALESCE(invoices.grand_total_bhd, 0))
				FROM invoices
				WHERE invoices.order_id = orders.id
					AND invoices.deleted_at IS NULL
					AND invoices.status NOT IN ('Draft', 'Cancelled', 'Void', 'Proforma')
			), 0) AS invoiced_bhd`).
		Where("orders.status NOT IN ?", []string{"Draft", "Cancelled", "Canceled", "Void"}).
		Where("orders.order_date >= ? AND orders.order_date < ?", start, end).
		Find(&rows).Error
	if err != nil {
		return 0, 0, err
	}

	var exposure float64
	var pendingOrders int
	for _, row := range rows {
		remainder := row.OrderValueBHD - row.InvoicedBHD
		if remainder > 0.001 {
			exposure += remainder
			pendingOrders++
		}
	}

	return roundTo3(exposure), pendingOrders, nil
}

// GetDashboardStats retrieves dashboard/sales statistics for UI
func (a *App) GetDashboardStats() (DashboardStats, error) {
	// P0 FIX: Add permission check - dashboard:view required
	if err := a.requirePermission("dashboard:view"); err != nil {
		return DashboardStats{}, err
	}
	if a.db == nil {
		return DashboardStats{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var stats DashboardStats

	var allOpportunities []Opportunity
	if err := a.db.Find(&allOpportunities).Error; err != nil {
		log.Printf("Warning: Failed to load opportunities for dashboard stats: %v", err)
	}

	closedKeys := make(map[string]bool)
	dedupedOpportunities := make(map[string]Opportunity)
	for _, opp := range allOpportunities {
		normalized := normalizeOpportunityForList(opp)
		if shouldSuppressSyntheticOCR(normalized) {
			continue
		}
		key := canonicalOpportunityKey(normalized)
		if strings.TrimSpace(key) == "" {
			key = normalized.ID
		}
		existing, exists := dedupedOpportunities[key]
		if !exists || shouldPreferOpportunity(normalized, existing) {
			dedupedOpportunities[key] = normalized
		}
		if normalized.Stage == "Won" || normalized.Stage == "Lost" {
			closedKeys[key] = true
		}
	}

	// 1. Active RFQs (Pipeline)
	activePipeline := 0
	for key, opp := range dedupedOpportunities {
		if opp.Stage == "Won" || opp.Stage == "Lost" || opp.Stage == "Expired" {
			continue
		}
		if closedKeys[key] {
			continue
		}
		activePipeline++
	}
	stats.ActiveRFQs = activePipeline

	// 2. Active Orders (NEW)
	var orderCount int64
	if err := a.db.Model(&Order{}).
		Where("status NOT IN ?", []string{"Delivered", "Completed", "Closed", "Cancelled"}).
		Count(&orderCount).Error; err != nil {
		log.Printf("Warning: Failed to count active Orders: %v", err)
	}
	stats.ActiveOrders = int(orderCount)

	// 3. Determine latest business activity year from invoices, orders, and opportunities.
	var latestInvoiceYear int
	a.db.Model(&Invoice{}).Select("COALESCE(MAX(CAST(strftime('%Y', invoice_date) AS INTEGER)), 0)").Scan(&latestInvoiceYear)

	var latestOrderYear int
	a.db.Model(&Order{}).Select("COALESCE(MAX(CAST(strftime('%Y', order_date) AS INTEGER)), 0)").Scan(&latestOrderYear)

	latestOpportunityYear := 0
	for _, opp := range dedupedOpportunities {
		if opp.Year > latestOpportunityYear {
			latestOpportunityYear = opp.Year
		}
	}

	activityYear := max(latestInvoiceYear, max(latestOrderYear, latestOpportunityYear))
	if activityYear == 0 {
		activityYear = time.Now().Year()
	}
	stats.ActivityYear = activityYear

	yearStart := time.Date(activityYear, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(activityYear+1, 1, 1, 0, 0, 0, 0, time.UTC)
	operationalStart := operationalMetricStartForYear(activityYear)
	if operationalStart.Before(yearStart) {
		operationalStart = yearStart
	}
	if usesOperationalFreshStart(activityYear) {
		stats.FreshStartDate = operationalStart.Format("2006-01-02")
	}
	log.Printf("Dashboard using business activity year: %d (invoice=%d order=%d opportunity=%d)", activityYear, latestInvoiceYear, latestOrderYear, latestOpportunityYear)

	activePipeline = 0
	pipelineValue := 0.0
	for key, opp := range dedupedOpportunities {
		if opp.Year != 0 && opp.Year != activityYear {
			continue
		}
		if opp.Stage == "Won" || opp.Stage == "Lost" || opp.Stage == "Expired" {
			continue
		}
		if closedKeys[key] {
			continue
		}
		activePipeline++
		pipelineValue += opp.RevenueBHD
	}
	stats.ActiveRFQs = activePipeline
	stats.PipelineValueBHD = roundTo3(pipelineValue)

	if err := a.db.Model(&Order{}).
		Where("status NOT IN ?", []string{"Delivered", "Completed", "Closed", "Cancelled"}).
		Where("order_date >= ? AND order_date < ?", operationalStart, yearEnd).
		Count(&orderCount).Error; err != nil {
		log.Printf("Warning: Failed to count YTD active Orders: %v", err)
	}
	stats.ActiveOrders = int(orderCount)

	postedInvoiceStatuses := []string{"Sent", "Paid", "PartiallyPaid", "Overdue"}

	var invoiceRevenue float64
	if err := a.db.Model(&Invoice{}).Where("status IN ? AND invoice_date >= ? AND invoice_date < ?", postedInvoiceStatuses, operationalStart, yearEnd).Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&invoiceRevenue).Error; err != nil {
		log.Printf("Warning: Failed to calculate revenue: %v", err)
	}

	var bookedOrderValue float64
	if err := a.db.Model(&Order{}).Where("order_date >= ? AND order_date < ? AND status NOT IN ?", operationalStart, yearEnd, []string{"Cancelled", "Canceled", "Void"}).
		Select("COALESCE(SUM(CASE WHEN grand_total_bhd > 0 THEN grand_total_bhd ELSE total_value_bhd END), 0)").Scan(&bookedOrderValue).Error; err != nil {
		log.Printf("Warning: Failed to calculate booked order value: %v", err)
	}

	stats.TotalRevenue = invoiceRevenue
	stats.RevenueMeta = fmt.Sprintf("Invoiced FY%d", activityYear)
	if usesOperationalFreshStart(activityYear) {
		stats.RevenueMeta = fmt.Sprintf("Fresh start from %s", operationalStart.Format("2 Jan 2006"))
	}
	if bookedOrderValue > stats.TotalRevenue {
		stats.TotalRevenue = bookedOrderValue
		stats.RevenueMeta = fmt.Sprintf("Confirmed orders FY%d", activityYear)
		if usesOperationalFreshStart(activityYear) {
			stats.RevenueMeta = fmt.Sprintf("Fresh orders from %s", operationalStart.Format("2 Jan 2006"))
		}
	}

	// 4. Win Rate
	var totalClosed, wonCount int
	for _, opp := range dedupedOpportunities {
		if opp.Year != 0 && opp.Year != activityYear {
			continue
		}
		if opp.Stage == "Won" || opp.Stage == "Lost" {
			totalClosed++
			if opp.Stage == "Won" {
				wonCount++
			}
		}
	}

	if totalClosed > 0 {
		stats.WinRate = (float64(wonCount) / float64(totalClosed)) * 100
	}

	// 5. Velocity (Avg days from Quote to Order)
	// For now, simpler: Just use PredictedDays from PredictionRecord as a proxy for "Sales Cycle"
	// or calculate from Won offers if we linked them properly.
	// Let's use the prediction records avg predicted days as it's already calculated per customer
	var avgDays float64
	if err := a.db.Model(&PredictionRecord{}).Select("AVG(predicted_days)").Scan(&avgDays).Error; err == nil {
		stats.AvgVelocityDays = avgDays
	}

	// 6. Month Growth (Revenue this month vs last month)
	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	currentPeriodStart := currentMonthStart
	if usesOperationalFreshStart(activityYear) && operationalFreshStartDate.After(currentPeriodStart) {
		currentPeriodStart = operationalFreshStartDate
	}
	lastMonthStart := currentMonthStart.AddDate(0, -1, 0)

	var currentMonthRev, lastMonthRev float64
	a.db.Model(&Invoice{}).Where("status IN ? AND invoice_date >= ?", postedInvoiceStatuses, currentPeriodStart).Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&currentMonthRev)
	a.db.Model(&Invoice{}).Where("status IN ? AND invoice_date >= ? AND invoice_date < ?", postedInvoiceStatuses, lastMonthStart, currentMonthStart).Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&lastMonthRev)

	if currentMonthRev == 0 && lastMonthRev == 0 {
		a.db.Model(&Order{}).Where("status NOT IN ? AND order_date >= ?", []string{"Cancelled", "Canceled", "Void"}, currentPeriodStart).Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&currentMonthRev)
		a.db.Model(&Order{}).Where("status NOT IN ? AND order_date >= ? AND order_date < ?", []string{"Cancelled", "Canceled", "Void"}, lastMonthStart, currentMonthStart).Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&lastMonthRev)
	}

	if lastMonthRev > 0 {
		stats.MonthGrowth = ((currentMonthRev - lastMonthRev) / lastMonthRev) * 100
	} else if currentMonthRev > 0 {
		stats.MonthGrowth = 100 // Infinite growth
	}

	// 7. Outstanding AR (Accounts Receivable - Unpaid Invoices - Same year as revenue)
	var outstandingAR float64
	var pendingCount int64
	openInvoiceStatuses := []string{"Sent", "PartiallyPaid", "Overdue"}
	if err := a.db.Model(&Invoice{}).Where("status IN ? AND invoice_date >= ? AND invoice_date < ?", openInvoiceStatuses, operationalStart, yearEnd).
		Select("COALESCE(SUM(outstanding_bhd), 0)").Scan(&outstandingAR).Error; err != nil {
		log.Printf("Warning: Failed to calculate outstanding AR: %v", err)
	}
	stats.OutstandingAR = outstandingAR

	if err := a.db.Model(&Invoice{}).Where("status IN ? AND invoice_date >= ? AND invoice_date < ?", openInvoiceStatuses, operationalStart, yearEnd).Count(&pendingCount).Error; err != nil {
		log.Printf("Warning: Failed to count pending invoices: %v", err)
	}
	stats.PendingInvoices = int(pendingCount)

	if outstandingAR <= 0 {
		orderReceivableFallback, ordersPendingInvoice, err := a.calculateUninvoicedOrderExposure(operationalStart, yearEnd)
		if err != nil {
			log.Printf("Warning: Failed to derive receivable exposure from active orders: %v", err)
		} else {
			if orderReceivableFallback > 0 {
				stats.OutstandingAR = orderReceivableFallback
				if stats.PendingInvoices == 0 {
					stats.PendingInvoices = ordersPendingInvoice
				}
			}
		}
	}
	collectibleValue := invoiceRevenue + stats.OutstandingAR
	if collectibleValue > 0 {
		stats.CollectionRate = math.Round((invoiceRevenue/collectibleValue)*1000) / 10
	}

	// 8. AR Days Overdue (Average days overdue for unpaid invoices)
	type OverdueResult struct {
		AvgDaysOverdue float64
	}
	var overdueResult OverdueResult
	if err := a.db.Model(&Invoice{}).
		Where("status IN ? AND invoice_date >= ? AND invoice_date < ? AND due_date IS NOT NULL AND due_date < ?", openInvoiceStatuses, operationalStart, yearEnd, time.Now()).
		Select("COALESCE(AVG(JULIANDAY('now') - JULIANDAY(due_date)), 0) as avg_days_overdue").
		Scan(&overdueResult).Error; err != nil {
		log.Printf("Warning: Failed to calculate AR days overdue: %v", err)
	}
	stats.ARDaysOverdue = int(overdueResult.AvgDaysOverdue)

	// 9. Active Customers
	var customerCount int64
	if err := a.db.Raw(`
		SELECT COUNT(DISTINCT customer_id)
		FROM (
			SELECT customer_id FROM orders
			WHERE deleted_at IS NULL
				AND customer_id <> ''
				AND order_date >= ? AND order_date < ?
				AND status NOT IN ('Cancelled', 'Void')
			UNION
			SELECT customer_id FROM invoices
			WHERE deleted_at IS NULL
				AND customer_id <> ''
				AND invoice_date >= ? AND invoice_date < ?
				AND status NOT IN ('Cancelled', 'Void', 'Proforma', 'Draft')
			UNION
			SELECT customer_id FROM opportunities
			WHERE deleted_at IS NULL
				AND customer_id <> ''
				AND year = ?
		)
	`, operationalStart, yearEnd, operationalStart, yearEnd, activityYear).Scan(&customerCount).Error; err != nil {
		log.Printf("Warning: Failed to count YTD active customers: %v", err)
	}
	if customerCount == 0 {
		if err := a.db.Model(&CustomerMaster{}).
			Where("deleted_at IS NULL").
			Where("COALESCE(status, 'Active') = ?", "Active").
			Count(&customerCount).Error; err != nil {
			log.Printf("Warning: Failed to count customers: %v", err)
		}
	}
	stats.ActiveCustomers = int(customerCount)

	// Mock/Derived fields
	stats.PendingReview = stats.ActiveRFQs
	stats.UrgentCount = stats.PendingInvoices
	stats.Runway = 12.5 // Calculated from Survival Garden usually

	if cashSnapshot, err := computeCashPositionSnapshot(a); err == nil {
		stats.CashBalanceBHD = cashSnapshot.CashBalanceBHD
		if len(cashSnapshot.Notices) > 0 {
			stats.CashPositionNote = strings.Join(cashSnapshot.Notices, " ")
		}
	} else {
		log.Printf("Warning: Failed to calculate dashboard cash position: %v", err)
	}

	// System health
	if stats.WinRate > 30 {
		stats.SystemHealth = "Robust"
	} else if stats.WinRate > 15 {
		stats.SystemHealth = "Healthy"
	} else {
		stats.SystemHealth = "Needs Attention"
	}

	if !a.currentSessionCanViewFinanceDashboard() {
		stats.TotalRevenue = 0
		stats.MonthGrowth = 0
		stats.OutstandingAR = 0
		stats.ARDaysOverdue = 0
		stats.PendingInvoices = 0
		stats.CollectionRate = 0
		stats.CashBalanceBHD = 0
		stats.CashPositionNote = ""
		stats.RevenueMeta = "Sales view"
		stats.UrgentCount = stats.ActiveRFQs
	}

	log.Printf("Dashboard stats: %d active RFQs, %d Active Orders, %.2f BHD Revenue, %.1f%% Win Rate, %d Customers, %.2f AR",
		stats.ActiveRFQs, stats.ActiveOrders, stats.TotalRevenue, stats.WinRate, stats.ActiveCustomers, stats.OutstandingAR)

	return stats, nil
}

// CustomerRevenueData represents revenue aggregated by customer for charts
type CustomerRevenueData struct {
	CustomerID   string  `json:"customer_id"`
	CustomerName string  `json:"customer_name"`
	Revenue      float64 `json:"revenue"`
	InvoiceCount int     `json:"invoice_count"`
}

// GetMonthlyRevenueByCustomer retrieves revenue data grouped by customer for dashboard charts (Current Year Only)
func (a *App) GetMonthlyRevenueByCustomer() ([]CustomerRevenueData, error) {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var results []CustomerRevenueData

	// Use latest invoice year (same approach as dashboard stats)
	var latestYear int
	if err := a.db.Model(&Invoice{}).Select("COALESCE(MAX(strftime('%Y', invoice_date)), 0)").Scan(&latestYear).Error; err != nil || latestYear == 0 {
		latestYear = time.Now().Year()
	}

	// Query to aggregate revenue by customer (latest year with data)
	// Join invoices with customers to get customer names
	query := `
		SELECT
			c.id as customer_id,
			c.business_name as customer_name,
			COALESCE(SUM(i.grand_total_bhd), 0) as revenue,
			COUNT(i.id) as invoice_count
		FROM customers c
		LEFT JOIN invoices i ON i.customer_id = c.id AND i.status NOT IN ('Cancelled', 'Void')
			AND strftime('%Y', i.invoice_date) = ?
		GROUP BY c.id, c.business_name
		HAVING revenue > 0
		ORDER BY revenue DESC
	`

	if err := a.db.Raw(query, fmt.Sprintf("%d", latestYear)).Scan(&results).Error; err != nil {
		log.Printf("Error fetching customer revenue data: %v", err)
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve customer revenue data", err.Error())
	}

	log.Printf("Retrieved revenue data for %d customers (year %d)", len(results), latestYear)
	return results, nil
}

// DashboardEvent represents a business event for the activity feed

type DashboardEvent struct {
	ID      int       `json:"id"`
	Type    string    `json:"type"` // "RFQ", "Win", "Alert", "Msg"
	Title   string    `json:"title"`
	Time    time.Time `json:"time"`
	TimeAgo string    `json:"time_ago"` // Human readable: "2m ago"
	Color   string    `json:"color"`    // Hex color for UI
}

// GetDashboardEvents retrieves recent business events for activity feed
func (a *App) GetDashboardEvents(limit int) ([]DashboardEvent, error) {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}

	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var records []PredictionRecord
	if err := a.db.Order("created_at DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve dashboard events", err.Error())
	}

	events := make([]DashboardEvent, 0, len(records))

	for i, record := range records {
		event := DashboardEvent{
			ID:    i + 1,
			Title: record.CustomerName,
			Time:  record.CreatedAt,
		}

		// Calculate time ago
		duration := time.Since(record.CreatedAt)
		if duration < time.Minute {
			event.TimeAgo = "just now"
		} else if duration < time.Hour {
			minutes := int(duration.Minutes())
			event.TimeAgo = fmt.Sprintf("%dm ago", minutes)
		} else if duration < 24*time.Hour {
			hours := int(duration.Hours())
			event.TimeAgo = fmt.Sprintf("%dh ago", hours)
		} else {
			days := int(duration.Hours() / 24)
			event.TimeAgo = fmt.Sprintf("%dd ago", days)
		}

		// Categorize by grade
		switch record.Grade {
		case "A":
			event.Type = "Win"
			event.Color = "#c5a059" // Gold
		case "B":
			event.Type = "RFQ"
			event.Color = "#6b7c6b" // Moss
		case "C":
			event.Type = "Msg"
			event.Color = "#7d8ca3" // Stone
		case "D":
			event.Type = "Alert"
			event.Color = "#8c5e58" // Clay
		default:
			event.Type = "RFQ"
			event.Color = "#6b7c6b"
		}

		events = append(events, event)
	}

	log.Printf("Retrieved %d recent events", len(events))
	return events, nil
}

// ClearHistory clears all prediction records
func (a *App) ClearHistory() error {
	if err := a.requirePermission("predictions:delete"); err != nil {
		return err
	}
	result := a.db.Exec("DELETE FROM prediction_records")
	if result.Error != nil {
		return result.Error
	}

	log.Printf("Cleared %d prediction records", result.RowsAffected)
	return nil
}

// ExportCustomerTemplate generates a sample customer JSON for import
func (a *App) ExportCustomerTemplate() Customer {
	return Customer{
		ID:             "CUST001",
		BusinessName:   "Acme Industries Ltd.",
		OrderValue:     15000.0,
		OrderHistory:   []float64{12000, 14000, 13500},
		PaymentHistory: []int{45, 60, 52},
		RelationYears:  3,
		Industry:       "Manufacturing",
		Country:        "Bahrain",
		IsEmergency:    0,
		HasABB:         0,
		DisputeCount:   0,
	}
}

// Greet returns a greeting for the user
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, welcome to %s - Asymmetrica Intelligence!", name, activeOverlay.CompanyDisplayName)
}

// ValidateCustomer checks if customer data is valid
func (a *App) ValidateCustomer(customer Customer) ValidationResult {
	result := ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	if customer.ID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Customer ID is required")
	}

	if customer.BusinessName == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Business name is required")
	}

	if customer.OrderValue <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Order value must be positive")
	}

	if customer.RelationYears < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Relation years cannot be negative")
	}

	if customer.DisputeCount < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Dispute count cannot be negative")
	}

	return result
}

// Statistics represents summary statistics
type Statistics struct {
	TotalPredictions  int64          `json:"total_predictions"`
	GradeDistribution map[string]int `json:"grade_distribution"`
	AvgConfidence     float64        `json:"avg_confidence"`
	AvgPredictedDays  float64        `json:"avg_predicted_days"`
}

// ValidationResult represents validation outcome
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

// AppError represents structured error responses for frontend
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// newError creates a new AppError
func newError(code, message, details string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// ============================================================================
// DATABASE HELPERS - RETRY LOGIC & TIMEOUT
// ============================================================================

// withRetry executes a database operation with retry logic
// Retries up to 3 times on transient errors with exponential backoff
func withRetry(operation func() error) error {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable (database locked, connection lost, etc.)
		errStr := err.Error()
		isRetryable := strings.Contains(errStr, "database is locked") ||
			strings.Contains(errStr, "connection") ||
			strings.Contains(errStr, "timeout")

		if !isRetryable {
			return err // Don't retry non-transient errors
		}

		if attempt < maxRetries-1 {
			// Exponential backoff: 100ms, 200ms, 400ms
			backoff := time.Duration(100*(1<<attempt)) * time.Millisecond
			log.Printf("⚠️ Database operation failed (attempt %d/%d), retrying in %v: %v", attempt+1, maxRetries, backoff, err)
			time.Sleep(backoff)
		}
	}

	log.Printf("❌ Database operation failed after %d retries: %v", maxRetries, lastErr)
	return lastErr
}

// withTimeout executes a database operation with timeout (10 seconds default)
func withTimeout(ctx context.Context, operation func() error) error {
	// Create context with 10 second timeout if no context provided
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}

	// Channel to receive operation result
	errChan := make(chan error, 1)

	// Run operation in goroutine
	go func() {
		errChan <- operation()
	}()

	// Wait for completion or timeout
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("database operation timed out: %w", ctx.Err())
	}
}

// BatchResult wraps batch prediction results
type BatchResult struct {
	Summary     BatchSummary        `json:"summary"`
	Predictions []PaymentPrediction `json:"predictions"`
}

// ═══════════════════════════════════════════════════════════════════════════
// EXTERNAL TOOLS VALIDATION
// ═══════════════════════════════════════════════════════════════════════════

// GetToolsStatus returns current status of all external tools
func (a *App) GetToolsStatus() *integration.ToolsReport {
	if err := a.requirePermission("settings:view"); err != nil {
		return nil
	}

	if a.toolsValidator == nil {
		a.toolsValidator = integration.NewToolsValidator()
	}

	report := a.toolsValidator.ValidateAllTools()
	log.Printf("Tools status requested: %s", report.Summary)

	return report
}

// RefreshToolsStatus forces fresh validation of all tools
func (a *App) RefreshToolsStatus() *integration.ToolsReport {
	if err := a.requirePermission("settings:view"); err != nil {
		return nil
	}
	if a.toolsValidator == nil {
		a.toolsValidator = integration.NewToolsValidator()
	}

	a.toolsValidator.InvalidateCache()
	report := a.toolsValidator.ValidateAllTools()
	log.Printf("Tools status refreshed: %s", report.Summary)

	return report
}

// GetToolInstallInstructions returns formatted installation guide
func (a *App) GetToolInstallInstructions() string {
	if a.toolsValidator == nil {
		a.toolsValidator = integration.NewToolsValidator()
		a.toolsValidator.ValidateAllTools()
	}

	return a.toolsValidator.GetInstallInstructions()
}

// ═══════════════════════════════════════════════════════════════════════════
// CONFIGURATION MANAGEMENT
// ═══════════════════════════════════════════════════════════════════════════

// GetConfig returns current configuration (safe for frontend - secrets masked)
func (a *App) GetConfig() (map[string]any, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return nil, err
	}

	if a.config == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	cfg := a.config

	return map[string]any{
		"onedrive": map[string]any{
			"rfq_path":      cfg.OneDrive.RFQPath,
			"eh_path":       cfg.OneDrive.EHPath,
			"offers_path":   cfg.OneDrive.OffersPath,
			"invoices_path": cfg.OneDrive.InvoicesPath,
		},
		"database": map[string]any{
			"path": cfg.Database.Path,
		},
		"azure": map[string]any{
			"enabled":       cfg.Azure.Enabled,
			"tenant_id":     maskSecret(cfg.Azure.TenantID),
			"client_id":     maskSecret(cfg.Azure.ClientID),
			"client_secret": "****", // Never expose secret
		},
		"tools": map[string]any{
			"pandoc":    cfg.Tools.PandocPath,
			"ffmpeg":    cfg.Tools.FFmpegPath,
			"tesseract": cfg.Tools.TesseractPath,
		},
		"app": map[string]any{
			"log_level":              cfg.App.LogLevel,
			"debug_mode":             cfg.App.DebugMode,
			"watcher_debounce_ms":    cfg.App.WatcherDebounceMS,
			"watcher_queue_size":     cfg.App.WatcherQueueSize,
			"enable_file_watcher":    cfg.App.EnableFileWatcher,
			"enable_geometry_bridge": cfg.App.EnableGeometryBridge,
			"enable_auto_backup":     cfg.App.EnableAutoBackup,
			"backup_retention_days":  cfg.App.BackupRetentionDays,
		},
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// CUSTOMER 360 VIEW - COMPLETE CUSTOMER PROFILE
// ═══════════════════════════════════════════════════════════════════════════

// ReceivablesAgingSummary represents outstanding invoices by aging bucket
type ReceivablesAgingSummary struct {
	Current          float64 `json:"current"`       // 0-30 days
	Days30_60        float64 `json:"days_30_60"`    // 30-60 days overdue
	Days60_90        float64 `json:"days_60_90"`    // 60-90 days overdue
	Days90_120       float64 `json:"days_90_120"`   // 90-120 days overdue
	Days120Plus      float64 `json:"days_120_plus"` // 120+ days overdue
	TotalOutstanding float64 `json:"total_outstanding"`
}

// PaymentHistoryEntry represents a single payment record
type PaymentHistoryEntry struct {
	PaymentDate   time.Time `json:"payment_date"`
	AmountBHD     float64   `json:"amount_bhd"`
	InvoiceNumber string    `json:"invoice_number"`
	DaysToPayment int       `json:"days_to_payment"` // Actual days from invoice date
	PaymentMethod string    `json:"payment_method"`
}

// OpportunitySummary represents an open opportunity/RFQ
type OpportunitySummary struct {
	ID        uint      `json:"id"`
	Project   string    `json:"project"`
	Value     float64   `json:"value"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// OrderSummary represents recent order history
type OrderSummary struct {
	OrderNumber   string    `json:"order_number"`
	OrderDate     time.Time `json:"order_date"`
	TotalValueBHD float64   `json:"total_value_bhd"`
	Status        string    `json:"status"`
}

// Customer360Data represents complete customer profile with all relationships
type Customer360Data struct {
	// Customer Info
	CustomerID    string `json:"customer_id"`
	BusinessName  string `json:"business_name"`
	CustomerType  string `json:"customer_type"`
	Industry      string `json:"industry"`
	City          string `json:"city"`
	Country       string `json:"country"`
	RelationYears int    `json:"relation_years"`

	// Payment Profile
	CurrentGrade       string  `json:"current_grade"`
	PaymentTermsDays   int     `json:"payment_terms_days"`
	AvgPaymentDays     float64 `json:"avg_payment_days"`
	DisputeCount       int     `json:"dispute_count"`
	IsCreditBlocked    bool    `json:"is_credit_blocked"`
	RequiresPrepayment bool    `json:"requires_prepayment"`

	// Three-Regime Dynamics (from most recent prediction)
	R1 float64 `json:"r1"` // Exploration regime
	R2 float64 `json:"r2"` // Optimization regime
	R3 float64 `json:"r3"` // Stabilization regime

	// Financial Metrics
	TotalOrdersValue float64    `json:"total_orders_value"`
	TotalOrdersCount int        `json:"total_orders_count"`
	AvgOrderValue    float64    `json:"avg_order_value"`
	LastOrderDate    *time.Time `json:"last_order_date"`

	// Risk Flags
	HasABBCompetition bool `json:"has_abb_competition"`
	IsEmergencyOnly   bool `json:"is_emergency_only"`

	// Recent Activity
	RecentPredictions []PredictionRecord `json:"recent_predictions"`

	// WAVE 2 AGENT 4: DEEP CUSTOMER JOINS
	// Receivables aging analysis (outstanding invoices by bucket)
	ReceivablesAging ReceivablesAgingSummary `json:"receivables_aging"`

	// Payment history (last 6 payments with timing)
	PaymentHistory []PaymentHistoryEntry `json:"payment_history"`

	// Open opportunities (pending RFQs)
	OpenOpportunities []OpportunitySummary `json:"open_opportunities"`

	// Recent orders (last 5 orders)
	RecentOrders []OrderSummary `json:"recent_orders"`

	// Customer Lifetime Value (total revenue to date)
	CustomerLifetimeValue float64 `json:"customer_lifetime_value"`
}

// GetCustomer360View retrieves complete customer profile with relationships
