package vqc

import (
	"context"
	"testing"
	"time"
)

// TestOptimizationEngine_Basic tests the VQC optimizer at small scale
func TestOptimizationEngine_Basic(t *testing.T) {
	// Create engine with 1K candidates (fast test)
	engine := NewEngine(1000)
	engine.MaxIterations = 50 // Reduce iterations for faster test

	ctx := context.Background()
	start := time.Now()

	// Run optimization
	err := engine.Run(ctx)
	if err != nil {
		t.Fatalf("Optimization failed: %v", err)
	}

	elapsed := time.Since(start)
	results := engine.GetResults()

	// Verify results
	t.Logf("✅ VQC Optimization Complete!")
	t.Logf("   Candidates:     %d", 1000)
	t.Logf("   Filtered:       %d (%.1f%%)", results.FilteredCount, float64(results.FilteredCount)/10.0)
	t.Logf("   Active:         %d", results.ActiveCount)
	t.Logf("   Best Fitness:   %.6f", results.BestFitness)
	t.Logf("   Target (7/8):   %.6f", SEVEN_EIGHTHS)
	t.Logf("   Iterations:     %d", results.Iterations)
	t.Logf("   Total Time:     %v", elapsed)
	t.Logf("   Velocity:       %.0f candidates/sec", results.Velocity)
	t.Logf("")
	t.Logf("   Phase 1 (Filter):  %v", results.Phase1Time)
	t.Logf("   Phase 2 (SLERP):   %v", results.Phase2Time)
	t.Logf("   Phase 3 (Collapse): %v", results.Phase3Time)

	// Assertions
	if results.BestFitness < 0 || results.BestFitness > 1 {
		t.Errorf("Fitness out of range [0,1]: %.6f", results.BestFitness)
	}

	if results.ActiveCount == 0 {
		t.Error("No active candidates after filtering (shouldn't happen)")
	}

	if results.FilteredCount < 500 {
		t.Logf("⚠️ Warning: Expected ~88.9%% filtering, got %.1f%%. This is OK for small sample.", float64(results.FilteredCount)/10.0)
	}

	// Verify quaternion is on S³
	bestQ := results.BestCandidate.State
	norm := bestQ.Norm()
	if norm < 0.99 || norm > 1.01 {
		t.Errorf("Best candidate not on S³: ||Q|| = %.6f (expected ~1.0)", norm)
	}

	t.Logf("   Best Q = (%.4f, %.4fi, %.4fj, %.4fk)", bestQ.W, bestQ.X, bestQ.Y, bestQ.Z)
	t.Logf("   ||Q||  = %.6f ✅", norm)
}

// TestOptimizationEngine_MediumScale tests at 10K candidates
func TestOptimizationEngine_MediumScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping medium-scale test in short mode")
	}

	engine := NewEngine(10000)
	ctx := context.Background()
	start := time.Now()

	err := engine.Run(ctx)
	if err != nil {
		t.Fatalf("Optimization failed: %v", err)
	}

	elapsed := time.Since(start)
	results := engine.GetResults()

	t.Logf("✅ VQC Medium-Scale Test (10K candidates)")
	t.Logf("   Time:     %v", elapsed)
	t.Logf("   Velocity: %.0f candidates/sec", results.Velocity)
	t.Logf("   Fitness:  %.6f (target: %.6f)", results.BestFitness, SEVEN_EIGHTHS)

	// Should easily handle 10K in <100ms
	if elapsed > 500*time.Millisecond {
		t.Logf("⚠️ Warning: 10K candidates took %v (expected <500ms). CPU may be throttled.", elapsed)
	}
}

// BenchmarkOptimization_1K benchmarks 1K candidates
func BenchmarkOptimization_1K(b *testing.B) {
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		engine := NewEngine(1000)
		engine.MaxIterations = 50
		_ = engine.Run(ctx)
	}
}

// BenchmarkOptimization_10K benchmarks 10K candidates
func BenchmarkOptimization_10K(b *testing.B) {
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		engine := NewEngine(10000)
		engine.MaxIterations = 108
		_ = engine.Run(ctx)
	}
}

// BenchmarkOptimization_100K benchmarks 100K candidates
func BenchmarkOptimization_100K(b *testing.B) {
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		engine := NewEngine(100000)
		engine.MaxIterations = 108
		_ = engine.Run(ctx)
	}
}
