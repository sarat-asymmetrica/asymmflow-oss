package engines

import (
	"math"
	"testing"
)

// TestWilliamsFormula verifies the core Williams formula: √n × log₂(n)
func TestWilliamsFormula(t *testing.T) {
	optimizer := NewWilliamsOptimizer()

	tests := []struct {
		name          string
		n             int
		expectedBatch int // Approximate, due to rounding
	}{
		{"Small dataset", 10, 11},       // √10 × log₂(10) ≈ 10.5 → 11
		{"Medium dataset", 1000, 315},   // √1000 × log₂(1000) ≈ 315.8 → 315
		{"Large dataset", 108000, 5500}, // √108000 × log₂(108000) ≈ 5,495
		{"Single item", 1, 1},           // Edge case
		{"Empty", 0, 0},                 // Edge case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batchSize := optimizer.CalculateOptimalBatchSize(tt.n)

			// Allow 10% tolerance for rounding
			tolerance := float64(tt.expectedBatch) * 0.1
			diff := math.Abs(float64(batchSize - tt.expectedBatch))

			if diff > tolerance {
				t.Errorf("CalculateOptimalBatchSize(%d) = %d, expected ~%d (±%.0f)",
					tt.n, batchSize, tt.expectedBatch, tolerance)
			}
		})
	}
}

// TestWilliamsMetrics verifies complete metrics calculation
func TestWilliamsMetrics(t *testing.T) {
	optimizer := NewWilliamsOptimizer()

	// Test with Acme Instrumentation realistic scale: 108,000 customers
	n := 108000
	metrics := optimizer.CalculateMetrics(n)

	// Verify fields are populated
	if metrics.TotalItems != n {
		t.Errorf("TotalItems = %d, want %d", metrics.TotalItems, n)
	}

	if metrics.OptimalBatchSize <= 0 {
		t.Error("OptimalBatchSize should be positive")
	}

	if metrics.MemoryMB <= 0 {
		t.Error("MemoryMB should be positive")
	}

	if metrics.Efficiency <= 1.0 {
		t.Error("Efficiency should be > 1.0 (Williams is better than linear)")
	}

	if metrics.MemorySavingsPercent <= 0 {
		t.Error("MemorySavingsPercent should be positive")
	}

	if metrics.SpaceComplexity != "O(√n × log₂n)" {
		t.Errorf("SpaceComplexity = %s, want O(√n × log₂n)", metrics.SpaceComplexity)
	}

	if metrics.ProofURL == "" {
		t.Error("ProofURL should not be empty")
	}

	t.Logf("Williams Metrics for n=%d:", n)
	t.Logf("  Batch Size: %d", metrics.OptimalBatchSize)
	t.Logf("  Memory: %.1f MB", metrics.MemoryMB)
	t.Logf("  Savings: %.1f MB (%.1f%%)", metrics.MemorySavingsMB, metrics.MemorySavingsPercent)
	t.Logf("  Efficiency: %.1fx better than linear", metrics.Efficiency)
}

// TestWilliamsWorkerCalculation verifies worker pool sizing
func TestWilliamsWorkerCalculation(t *testing.T) {
	optimizer := NewWilliamsOptimizer()

	tests := []struct {
		name       string
		totalFiles int
		maxWorkers int
		expected   int
	}{
		{"Small batch", 10, 100, 11},     // Williams formula: √10 × log₂(10) ≈ 11
		{"Medium batch", 1000, 100, 100}, // Capped by maxWorkers
		{"Large batch", 10000, 200, 200}, // Capped by maxWorkers
		{"Zero files", 0, 100, 1},        // Minimum 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workers := optimizer.CalculateOptimalWorkers(tt.totalFiles, tt.maxWorkers)

			// Allow tolerance for uncapped cases
			if tt.totalFiles > 0 && tt.expected < tt.maxWorkers {
				tolerance := float64(tt.expected) * 0.1
				diff := math.Abs(float64(workers - tt.expected))
				if diff > tolerance {
					t.Errorf("CalculateOptimalWorkers(%d, %d) = %d, expected ~%d",
						tt.totalFiles, tt.maxWorkers, workers, tt.expected)
				}
			} else {
				if workers != tt.expected {
					t.Errorf("CalculateOptimalWorkers(%d, %d) = %d, want %d",
						tt.totalFiles, tt.maxWorkers, workers, tt.expected)
				}
			}
		})
	}
}

// TestWilliamsComparison verifies comparison metrics
func TestWilliamsComparison(t *testing.T) {
	optimizer := NewWilliamsOptimizer()
	n := 10000

	comparison := optimizer.CompareWithLinear(n)

	// Verify structure
	williams, ok := comparison["williams"].(map[string]any)
	if !ok {
		t.Fatal("Williams metrics not found in comparison")
	}

	linear, ok := comparison["linear"].(map[string]any)
	if !ok {
		t.Fatal("Linear metrics not found in comparison")
	}

	improvement, ok := comparison["improvement"].(map[string]any)
	if !ok {
		t.Fatal("Improvement metrics not found in comparison")
	}

	// Verify Williams is better
	williamsBatch := williams["batch_size"].(int)
	linearBatch := linear["batch_size"].(int)

	if williamsBatch >= linearBatch {
		t.Errorf("Williams batch (%d) should be < linear batch (%d)", williamsBatch, linearBatch)
	}

	williamsMemory := williams["memory_mb"].(float64)
	linearMemory := linear["memory_mb"].(float64)

	if williamsMemory >= linearMemory {
		t.Errorf("Williams memory (%.1f MB) should be < linear memory (%.1f MB)",
			williamsMemory, linearMemory)
	}

	efficiency := improvement["efficiency"].(float64)
	if efficiency <= 1.0 {
		t.Errorf("Efficiency (%.1fx) should be > 1.0", efficiency)
	}

	t.Logf("Comparison for n=%d:", n)
	t.Logf("  Williams: batch=%d, memory=%.1f MB", williamsBatch, williamsMemory)
	t.Logf("  Linear: batch=%d, memory=%.1f MB", linearBatch, linearMemory)
	t.Logf("  Efficiency: %.1fx better", efficiency)
}

// TestWilliamsMemoryEstimation verifies memory calculations are realistic
func TestWilliamsMemoryEstimation(t *testing.T) {
	optimizer := NewWilliamsOptimizer()

	// For 108,000 customers (Acme Instrumentation scale)
	metrics := optimizer.CalculateMetrics(108000)

	// Memory should be in reasonable range (not MB for 100K items!)
	if metrics.MemoryMB > 100 {
		t.Errorf("Memory estimate too high: %.1f MB for Williams batching", metrics.MemoryMB)
	}

	if metrics.MemoryMB < 5 {
		t.Errorf("Memory estimate suspiciously low: %.1f MB", metrics.MemoryMB)
	}

	// Savings should be substantial
	if metrics.MemorySavingsPercent < 50 {
		t.Errorf("Memory savings should be >50%%, got %.1f%%", metrics.MemorySavingsPercent)
	}

	t.Logf("Memory estimation for 108K items:")
	t.Logf("  Williams: %.1f MB", metrics.MemoryMB)
	t.Logf("  Savings: %.1f%% reduction", metrics.MemorySavingsPercent)
}

// BenchmarkWilliamsCalculation benchmarks the formula calculation
func BenchmarkWilliamsCalculation(b *testing.B) {
	optimizer := NewWilliamsOptimizer()
	n := 108000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.CalculateOptimalBatchSize(n)
	}
}

// BenchmarkWilliamsMetrics benchmarks full metrics generation
func BenchmarkWilliamsMetrics(b *testing.B) {
	optimizer := NewWilliamsOptimizer()
	n := 108000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.CalculateMetrics(n)
	}
}
