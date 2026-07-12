package main

import (
	"sync"
	"time"
)

// PerformanceOptimizations - Professional Edition (UUID-Aware)

// CachedMetrics holds survival metrics with TTL
type CachedMetrics struct {
	data      *SurvivalMetrics
	timestamp time.Time
	mu        sync.RWMutex
}

var metricsCache = &CachedMetrics{}

// OverdueGradeBreakdown struct for reporting
type OverdueGradeBreakdown struct {
	Grade        string  `json:"grade"`
	Overdue30    float64 `json:"overdue_30"`
	Overdue60    float64 `json:"overdue_60"`
	Overdue120   float64 `json:"overdue_120"`
	TotalOverdue float64 `json:"total_overdue"`
	InvoiceCount int     `json:"invoice_count"`
}

// GetSurvivalMetricsOptimized returns cached metrics if fresh
func (a *App) GetSurvivalMetricsOptimized() (SurvivalMetrics, error) {
	metricsCache.mu.RLock()
	if metricsCache.data != nil && time.Since(metricsCache.timestamp) < 30*time.Second {
		cached := *metricsCache.data
		metricsCache.mu.RUnlock()
		return cached, nil
	}
	metricsCache.mu.RUnlock()

	metricsCache.mu.Lock()
	defer metricsCache.mu.Unlock()

	// Recompute
	survivalEngine := NewSurvivalIntelligence(a)
	runway, _ := survivalEngine.CheckCashRunway()

	// Get cash on hand from chart of accounts
	var cashOnHand float64
	if a.db != nil {
		a.db.Model(&ChartOfAccount{}).
			Where("account_code IN ?", []string{"1000", "1100"}).
			Select("COALESCE(SUM(balance), 0)").
			Scan(&cashOnHand)
	}

	// Estimate monthly burn (simplified - should come from real expense data)
	monthlyBurn := 10000.0 // Default 10k BHD/month

	// Determine runway status
	var runwayStatus string
	switch {
	case runway < 30:
		runwayStatus = "critical"
	case runway < 60:
		runwayStatus = "warning"
	default:
		runwayStatus = "safe"
	}

	// Count critical alerts
	var criticalAlerts int64
	if a.db != nil {
		a.db.Model(&Alert{}).Where("is_active = ? AND severity = ?", true, "critical").Count(&criticalAlerts)
	}

	// Get receivables and payables totals
	var receivablesTotal float64
	var payablesTotal float64 = 0.0 // No PurchaseOrder table yet - default to 0
	if a.db != nil {
		a.db.Model(&Invoice{}).Select("COALESCE(SUM(outstanding_bhd), 0)").Scan(&receivablesTotal)
	}

	// Build overdue by grade map
	overdueByGrade := make(map[string]float64)
	if a.db != nil {
		var gradeResults []struct {
			Grade        string  `gorm:"column:grade"`
			TotalOverdue float64 `gorm:"column:total_overdue"`
		}
		a.db.Raw(`
			SELECT c.payment_grade as grade, COALESCE(SUM(i.outstanding_bhd), 0) as total_overdue
			FROM invoices i
			JOIN customers c ON i.customer_id = c.id
			WHERE i.status = 'Overdue'
			GROUP BY c.payment_grade
		`).Scan(&gradeResults)
		for _, gr := range gradeResults {
			overdueByGrade[gr.Grade] = gr.TotalOverdue
		}
	}

	metrics := SurvivalMetrics{
		// Core fields
		CashRunwayDays:       runway,
		MonthlyBurnRate:      monthlyBurn,
		CashOnHand:           cashOnHand,
		ReceivablesTotal:     receivablesTotal,
		PayablesTotal:        payablesTotal,
		CriticalAlerts:       int(criticalAlerts),
		CollectionEfficiency: 0.75, // Default - should be calculated from actual data

		// Frontend-expected aliases
		LastUpdated:           time.Now(),
		RunwayStatus:          runwayStatus,
		DaysOfRunway:          runway,
		CashBalance:           cashOnHand,
		MonthlyBurn:           monthlyBurn,
		WeekCollectionsActual: receivablesTotal * 0.1, // Simplified estimate
		WeekCollectionsTarget: receivablesTotal * 0.15,
		OverdueByGrade:        overdueByGrade,
	}

	metricsCache.data = &metrics
	metricsCache.timestamp = time.Now()

	return metrics, nil
}

// InvalidateMetricsCache forces cache refresh
func InvalidateMetricsCache() {
	metricsCache.mu.Lock()
	metricsCache.timestamp = time.Time{}
	metricsCache.mu.Unlock()
}

// GetCollectionPerformance analyzes collection efficiency
func (a *App) GetCollectionPerformance() (map[string]any, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// 1. Get Survival Metrics
	survival, err := a.GetSurvivalMetricsOptimized()
	if err != nil {
		return nil, err
	}

	// 2. Analyze Collection Efficiency
	var gradePerformance []OverdueGradeBreakdown

	// Optimized query using UUID joins
	query := `
		SELECT 
			c.payment_grade as grade,
			COUNT(i.id) as invoice_count,
			SUM(i.outstanding_bhd) as total_overdue
		FROM invoices i
		JOIN customers c ON i.customer_id = c.id
		WHERE i.status = 'Overdue'
		GROUP BY c.payment_grade
	`

	if err := a.db.Raw(query).Scan(&gradePerformance).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to analyze collection performance", err.Error())
	}

	// 3. Score
	totalOverdue := 0.0
	for _, gp := range gradePerformance {
		totalOverdue += gp.TotalOverdue
	}

	return map[string]any{
		"survival_metrics": survival,
		"grade_breakdown":  gradePerformance,
		"total_overdue":    totalOverdue,
	}, nil
}
