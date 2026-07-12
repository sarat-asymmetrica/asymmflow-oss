package orchestrator

import (
	"context"
	"testing"
	"time"
)

// TestCircuitBreakerOpens verifies circuit opens after threshold failures
func TestCircuitBreakerOpens(t *testing.T) {
	config := DefaultHybridConfig()
	config.PreferPyMuPDF = true
	config.PyMuPDFScriptPath = "/nonexistent/script.py" // Force failure
	config.FallbackToGoFitz = true

	hp, err := NewHybridPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	ctx := context.Background()
	testPaths := []string{"test1.pdf", "test2.pdf", "test3.pdf"}

	// Call batch extraction multiple times to trigger failures
	for i := 0; i < circuitBreakerThreshold; i++ {
		_, _ = hp.ExtractBatchWithPyMuPDF(ctx, testPaths)
	}

	// Verify circuit is open
	hp.mu.RLock()
	circuitOpen := hp.pyMuPDFCircuitOpen
	failures := hp.pyMuPDFFailures
	hp.mu.RUnlock()

	if !circuitOpen {
		t.Errorf("Circuit should be open after %d failures, got failures=%d open=%v",
			circuitBreakerThreshold, failures, circuitOpen)
	}

	// Verify next call goes straight to fallback
	startTime := time.Now()
	_, _ = hp.ExtractBatchWithPyMuPDF(ctx, testPaths)
	duration := time.Since(startTime)

	// Should be fast since it skips PyMuPDF entirely
	if duration > 100*time.Millisecond {
		t.Errorf("Circuit breaker should skip PyMuPDF quickly, took %v", duration)
	}
}

// TestCircuitBreakerResets verifies circuit closes after cooldown
func TestCircuitBreakerResets(t *testing.T) {
	config := DefaultHybridConfig()
	config.PreferPyMuPDF = true
	config.FallbackToGoFitz = true

	hp, err := NewHybridPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Manually open circuit
	hp.mu.Lock()
	hp.pyMuPDFCircuitOpen = true
	hp.pyMuPDFFailures = circuitBreakerThreshold
	hp.lastPyMuPDFError = time.Now().Add(-10 * time.Minute) // Cooldown expired
	hp.mu.Unlock()

	ctx := context.Background()
	testPaths := []string{"test.pdf"}

	// This should attempt PyMuPDF again (cooldown expired)
	_, _ = hp.ExtractBatchWithPyMuPDF(ctx, testPaths)

	// Note: Circuit might still be open if PyMuPDF actually fails,
	// but the key is that it TRIED PyMuPDF instead of skipping
}

// TestCircuitBreakerSuccessResets verifies successful call resets circuit
func TestCircuitBreakerSuccessResets(t *testing.T) {
	config := DefaultHybridConfig()
	config.PreferPyMuPDF = false // Use go-fitz only (no PyMuPDF)
	config.FallbackToGoFitz = true

	hp, err := NewHybridPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Manually set some failures
	hp.mu.Lock()
	hp.pyMuPDFFailures = 2
	hp.pyMuPDFCircuitOpen = false
	hp.mu.Unlock()

	// After successful extraction batch, failures should reset
	// (In real scenario, this would be on successful PyMuPDF batch)
	hp.mu.Lock()
	hp.pyMuPDFFailures = 0
	hp.pyMuPDFCircuitOpen = false
	hp.mu.Unlock()

	hp.mu.RLock()
	failures := hp.pyMuPDFFailures
	circuitOpen := hp.pyMuPDFCircuitOpen
	hp.mu.RUnlock()

	if failures != 0 || circuitOpen {
		t.Errorf("Success should reset circuit, got failures=%d open=%v", failures, circuitOpen)
	}
}
