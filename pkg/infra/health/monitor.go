package health

import (
	"sync"
	"time"

	"ph_holdings_app/pkg/math/trident"
)

// SystemHealth represents the current system health assessment.
type SystemHealth struct {
	Regime       trident.Regime `json:"regime"`
	RegimeName   string         `json:"regime_name"`
	Score        float64        `json:"score"`
	ActiveUsers  int            `json:"active_users"`
	ErrorRate    float64        `json:"error_rate"`
	AvgLatencyMs float64        `json:"avg_latency_ms"`
	Uptime       time.Duration  `json:"uptime"`
	CheckedAt    time.Time      `json:"checked_at"`
}

// Monitor tracks system health metrics and classifies them into regimes.
type Monitor struct {
	mu           sync.RWMutex
	errorCount   int64
	requestCount int64
	latencySum   float64
	startTime    time.Time
}

// NewMonitor creates a health monitor.
func NewMonitor() *Monitor {
	return &Monitor{startTime: time.Now()}
}

// RecordRequest records a completed request.
func (m *Monitor) RecordRequest(latencyMs float64, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if latencyMs < 0 {
		latencyMs = 0
	}
	m.requestCount++
	m.latencySum += latencyMs
	if isError {
		m.errorCount++
	}
}

// Health returns the current health assessment with regime classification.
func (m *Monitor) Health() SystemHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	uptime := now.Sub(m.startTime)
	requests := m.requestCount

	var avgLatency float64
	var errorRatio float64
	if requests > 0 {
		avgLatency = m.latencySum / float64(requests)
		errorRatio = float64(m.errorCount) / float64(requests)
	}

	errorRate := 0.0
	if uptime > 0 {
		errorRate = float64(m.errorCount) / uptime.Minutes()
	}

	regime := classify(errorRatio, avgLatency)
	score := healthScore(errorRatio, avgLatency)
	return SystemHealth{
		Regime:       regime,
		RegimeName:   regime.String(),
		Score:        score,
		ErrorRate:    errorRate,
		AvgLatencyMs: avgLatency,
		Uptime:       uptime,
		CheckedAt:    now,
	}
}

// Reset clears accumulated metrics.
func (m *Monitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errorCount = 0
	m.requestCount = 0
	m.latencySum = 0
	m.startTime = time.Now()
}

func classify(errorRatio, avgLatencyMs float64) trident.Regime {
	switch {
	case errorRatio >= 0.20 || avgLatencyMs >= 1000:
		return trident.RegimeExploration
	case errorRatio >= 0.05 || avgLatencyMs >= 250:
		return trident.RegimeOptimization
	default:
		return trident.RegimeStabilization
	}
}

func healthScore(errorRatio, avgLatencyMs float64) float64 {
	latencyPenalty := avgLatencyMs / 1000
	if latencyPenalty > 1 {
		latencyPenalty = 1
	}
	score := 1 - errorRatio - latencyPenalty*0.3
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
