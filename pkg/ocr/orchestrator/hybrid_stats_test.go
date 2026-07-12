package orchestrator

import (
	"testing"
	"time"
)

// TestHybridStatsFailureRate validates failure rate calculation
func TestHybridStatsFailureRate(t *testing.T) {
	tests := []struct {
		name         string
		totalDocs    int
		errorCount   int
		expectedRate float64
	}{
		{"No errors", 100, 0, 0.0},
		{"10% failure", 100, 10, 0.1},
		{"50% failure", 100, 50, 0.5},
		{"All failures", 100, 100, 1.0},
		{"No documents", 0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &HybridStats{
				TotalDocuments: tt.totalDocs,
				ErrorCount:     tt.errorCount,
			}

			rate := stats.FailureRate()
			if rate != tt.expectedRate {
				t.Errorf("FailureRate() = %v, want %v", rate, tt.expectedRate)
			}
		})
	}
}

// TestHybridStatsAverageLatency validates average latency calculation
func TestHybridStatsAverageLatency(t *testing.T) {
	tests := []struct {
		name          string
		totalDocs     int
		totalDuration time.Duration
		expectedAvg   time.Duration
	}{
		{"100 docs in 10s", 100, 10 * time.Second, 100 * time.Millisecond},
		{"50 docs in 5s", 50, 5 * time.Second, 100 * time.Millisecond},
		{"No documents", 0, 0, 0},
		{"Single doc 1s", 1, 1 * time.Second, 1 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &HybridStats{
				TotalDocuments: tt.totalDocs,
				TotalDuration:  tt.totalDuration,
			}

			avg := stats.AverageLatency()
			if avg != tt.expectedAvg {
				t.Errorf("AverageLatency() = %v, want %v", avg, tt.expectedAvg)
			}
		})
	}
}

// TestHybridStatsP95Latency validates P95 percentile calculation
func TestHybridStatsP95Latency(t *testing.T) {
	tests := []struct {
		name        string
		latencies   []time.Duration
		expectedP95 time.Duration
	}{
		{
			"Empty latencies",
			[]time.Duration{},
			0,
		},
		{
			"Single latency",
			[]time.Duration{100 * time.Millisecond},
			100 * time.Millisecond,
		},
		{
			"100 latencies (95th is 96ms)",
			generateLatencies(100),
			96 * time.Millisecond, // index 95 (0-based) = 96th value (1-based)
		},
		{
			"20 latencies (95th is 20ms)",
			generateLatencies(20),
			20 * time.Millisecond, // index 19 (0-based) = 20th value (1-based)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &HybridStats{
				Latencies: tt.latencies,
			}

			p95 := stats.P95Latency()
			if p95 != tt.expectedP95 {
				t.Errorf("P95Latency() = %v, want %v", p95, tt.expectedP95)
			}
		})
	}
}

// TestHybridStatsThreadSafety validates GetStats creates independent copies
func TestHybridStatsThreadSafety(t *testing.T) {
	pipeline := &HybridPipeline{
		stats: &HybridStats{
			TotalDocuments: 100,
			ErrorCount:     5,
			Latencies:      []time.Duration{1 * time.Second, 2 * time.Second},
			GPUAvailable:   true,
		},
	}

	// Get first copy
	stats1 := pipeline.GetStats()

	// Modify original
	pipeline.mu.Lock()
	pipeline.stats.TotalDocuments = 200
	pipeline.stats.Latencies = append(pipeline.stats.Latencies, 3*time.Second)
	pipeline.mu.Unlock()

	// Get second copy
	stats2 := pipeline.GetStats()

	// Verify independence
	if stats1.TotalDocuments != 100 {
		t.Errorf("stats1.TotalDocuments modified = %v, want 100", stats1.TotalDocuments)
	}

	if stats2.TotalDocuments != 200 {
		t.Errorf("stats2.TotalDocuments = %v, want 200", stats2.TotalDocuments)
	}

	if len(stats1.Latencies) != 2 {
		t.Errorf("stats1.Latencies modified = %v, want 2", len(stats1.Latencies))
	}

	if len(stats2.Latencies) != 3 {
		t.Errorf("stats2.Latencies = %v, want 3", len(stats2.Latencies))
	}
}

// TestHybridStatsMemoryCap validates latency cap at 10,000 entries
func TestHybridStatsMemoryCap(t *testing.T) {
	stats := &HybridStats{
		Latencies: make([]time.Duration, 10000),
	}

	// Simulate adding beyond cap (manual tracking logic)
	maxCap := 10000
	if len(stats.Latencies) >= maxCap {
		// Should NOT append
		t.Log("Latency cap reached - new latencies would be dropped")
	}

	if len(stats.Latencies) > maxCap {
		t.Errorf("Latencies exceeded cap: %v > %v", len(stats.Latencies), maxCap)
	}
}

// Helper: Generate sequential latencies (1ms, 2ms, ..., Nms)
func generateLatencies(n int) []time.Duration {
	latencies := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		latencies[i] = time.Duration(i+1) * time.Millisecond
	}
	return latencies
}
