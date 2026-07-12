package main

import (
	"fmt"
	"log"
	"time"
)

// SurvivalIntelligence handles critical business survival metrics
// Based on PH_TRADING_BUSINESS_REALITY_DOC.md
type SurvivalIntelligence struct {
	app *App
}

// SurvivalMetrics represents the core survival dashboard data
type SurvivalMetrics struct {
	// Backend canonical fields
	CashRunwayDays       float64 `json:"cash_runway_days"`
	MonthlyBurnRate      float64 `json:"monthly_burn_rate"`
	CashOnHand           float64 `json:"cash_on_hand"`
	ReceivablesTotal     float64 `json:"receivables_total"`
	PayablesTotal        float64 `json:"payables_total"`
	CriticalAlerts       int     `json:"critical_alerts"`
	CollectionEfficiency float64 `json:"collection_efficiency"` // 0-100%

	// Frontend-expected fields (aliased for compatibility)
	LastUpdated           time.Time          `json:"last_updated"`
	RunwayStatus          string             `json:"runway_status"`  // "safe", "warning", "critical"
	DaysOfRunway          float64            `json:"days_of_runway"` // alias of CashRunwayDays
	CashBalance           float64            `json:"cash_balance"`   // alias of CashOnHand
	MonthlyBurn           float64            `json:"monthly_burn"`   // alias of MonthlyBurnRate
	WeekCollectionsActual float64            `json:"week_collections_actual"`
	WeekCollectionsTarget float64            `json:"week_collections_target"`
	OverdueByGrade        map[string]float64 `json:"overdue_by_grade"` // {"A": 1234.5, "B": 2345.6, ...}
}

// AlertSummary represents alert counts for the dashboard
type AlertSummary struct {
	ActiveCritical int     `json:"active_critical"`
	ActiveWarning  int     `json:"active_warning"`
	ActiveInfo     int     `json:"active_info"`
	TotalActive    int     `json:"total_active"`
	TopAlerts      []Alert `json:"top_alerts"`
}

// NewSurvivalIntelligence creates a new survival intelligence engine
func NewSurvivalIntelligence(app *App) *SurvivalIntelligence {
	return &SurvivalIntelligence{app: app}
}

// CheckCashRunway analyzes cash runway and generates alerts
func (s *SurvivalIntelligence) CheckCashRunway() (float64, error) {
	if s.app.db == nil {
		return 0, nil
	}

	// 1. Get Cash on Hand (Account 1000 + 1100)
	var cashOnHand float64
	s.app.db.Model(&ChartOfAccount{}).
		Where("account_code IN ?", []string{"1000", "1100"}).
		Select("COALESCE(SUM(balance), 0)").
		Scan(&cashOnHand)

	// 2. Calculate Burn Rate (Expenses - COGS) / 3 months
	var totalExpenses float64
	// threeMonthsAgo := time.Now().AddDate(0, -3, 0)

	// Simplified: Get all expenses from journal lines in last 3 months
	// In a real implementation, we'd filter by account type "Expense" and exclude COGS
	// For now, using a fixed estimate fallback if data is insufficient
	if totalExpenses == 0 {
		totalExpenses = 30000.0 // Fallback: 10k/month burn
	}

	monthlyBurn := totalExpenses / 3.0
	if monthlyBurn <= 0 {
		monthlyBurn = 10000.0 // Safety floor
	}

	// 3. Calculate Runway
	daysRunway := (cashOnHand / monthlyBurn) * 30.0

	// 4. Alerting
	threshold := 60.0 // 2 months
	if daysRunway < threshold {
		s.createRunwayAlert(daysRunway, threshold)
	}

	return daysRunway, nil
}

// createRunwayAlert generates a critical alert for low runway
func (s *SurvivalIntelligence) createRunwayAlert(current, threshold float64) {
	alertType := "cash_runway_critical"

	// Check if active alert exists
	var count int64
	s.app.db.Model(&Alert{}).Where("alert_type = ? AND is_active = ?", alertType, true).Count(&count)
	if count > 0 {
		return // Alert already exists
	}

	alert := Alert{
		AlertType: alertType,
		Severity:  "critical",
		Message:   fmt.Sprintf("Cash runway is %.1f days (below %.0f day threshold). Immediate action required.", current, threshold),
		IsActive:  true,
	}

	s.app.db.Create(&alert)
	log.Printf("🚨 CRITICAL ALERT CREATED: Cash Runway %.1f Days", current)
}

// CheckCollectionEfficiency monitors if customers are paying on time
func (s *SurvivalIntelligence) CheckCollectionEfficiency() error {
	if s.app.db == nil {
		return nil
	}

	// Find overdue invoices > 90 days
	cutoffDate := time.Now().AddDate(0, 0, -90)
	var overdueInvoices []Invoice

	if err := s.app.db.Where("status != ? AND invoice_date < ?", "Paid", cutoffDate).Find(&overdueInvoices).Error; err != nil {
		return err
	}

	for _, inv := range overdueInvoices {
		s.createOverdueAlert(inv)
	}

	return nil
}

// createOverdueAlert generates an alert for a specific overdue invoice
func (s *SurvivalIntelligence) createOverdueAlert(inv Invoice) {
	alertType := "invoice_overdue_90"

	// Check if active alert exists for this invoice
	// Note: We scan message for invoice number since we don't have direct link in simplified Alert struct
	var count int64
	s.app.db.Model(&Alert{}).Where("alert_type = ? AND is_active = ? AND message LIKE ?",
		alertType, true, "%"+inv.InvoiceNumber+"%").Count(&count)

	if count > 0 {
		return
	}

	alert := Alert{
		AlertType: alertType,
		Severity:  "warning",
		Message:   fmt.Sprintf("Invoice %s for %s is >90 days overdue. Stop service advised.", inv.InvoiceNumber, inv.CustomerName),
		IsActive:  true,
	}

	s.app.db.Create(&alert)
}

// GetSurvivalMetrics returns the current survival metrics for the dashboard
// This wraps GetSurvivalMetricsOptimized for API compatibility
func (a *App) GetSurvivalMetrics() (SurvivalMetrics, error) {
	return a.GetSurvivalMetricsOptimized()
}

// GetAlertSummary returns alert counts by severity for the dashboard
func (a *App) GetAlertSummary() (AlertSummary, error) {
	summary := AlertSummary{
		TopAlerts: []Alert{}, // Initialize empty array
	}

	if a.db == nil {
		return summary, nil
	}

	// Count alerts by severity
	var criticalCount, warningCount, infoCount int64
	a.db.Model(&Alert{}).Where("is_active = ? AND severity = ?", true, "critical").Count(&criticalCount)
	a.db.Model(&Alert{}).Where("is_active = ? AND severity = ?", true, "warning").Count(&warningCount)
	a.db.Model(&Alert{}).Where("is_active = ? AND severity = ?", true, "info").Count(&infoCount)

	summary.ActiveCritical = int(criticalCount)
	summary.ActiveWarning = int(warningCount)
	summary.ActiveInfo = int(infoCount)
	summary.TotalActive = summary.ActiveCritical + summary.ActiveWarning + summary.ActiveInfo

	// Get top alerts (most recent active, max 10)
	a.db.Where("is_active = ?", true).
		Order("CASE severity WHEN 'critical' THEN 1 WHEN 'warning' THEN 2 ELSE 3 END").
		Order("created_at DESC").
		Limit(10).
		Find(&summary.TopAlerts)

	return summary, nil
}

// AcknowledgeAlert marks an alert as acknowledged but still active
func (a *App) AcknowledgeAlert(alertID int) error {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return err
	}
	if a.db == nil {
		return nil
	}

	return a.db.Model(&Alert{}).Where("id = ?", alertID).Update("acknowledged", true).Error
}

// DismissAlert marks an alert as inactive (dismissed)
func (a *App) DismissAlert(alertID int) error {
	if err := a.requirePermission("finance:view"); err != nil {
		return err
	}
	if a.db == nil {
		return nil
	}

	return a.db.Model(&Alert{}).Where("id = ?", alertID).Update("is_active", false).Error
}
