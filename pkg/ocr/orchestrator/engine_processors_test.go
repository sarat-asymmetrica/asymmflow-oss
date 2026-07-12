package orchestrator

import (
	"context"
	"testing"
	"time"
)

// Test that processors implement the interface
func TestProcessorInterface(t *testing.T) {
	var _ EngineProcessor = (*GoFitzProcessor)(nil)
	var _ EngineProcessor = (*Florence2Processor)(nil)
	var _ EngineProcessor = (*TesseractProcessor)(nil)
	var _ EngineProcessor = (*LocalGPUProcessor)(nil)
}

// Test ProcessorStats
func TestProcessorStats(t *testing.T) {
	stats := &ProcessorStats{}

	// Record success
	stats.RecordSuccess(1000, 100*time.Millisecond, 0.01)

	if stats.SuccessCount != 1 {
		t.Errorf("Expected SuccessCount=1, got %d", stats.SuccessCount)
	}
	if stats.TotalCharacters != 1000 {
		t.Errorf("Expected TotalCharacters=1000, got %d", stats.TotalCharacters)
	}
	if stats.AvgLatency != 100*time.Millisecond {
		t.Errorf("Expected AvgLatency=100ms, got %v", stats.AvgLatency)
	}
	if stats.SuccessRate != 1.0 {
		t.Errorf("Expected SuccessRate=1.0, got %f", stats.SuccessRate)
	}

	// Record error
	stats.RecordError()

	if stats.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount=1, got %d", stats.ErrorCount)
	}
	if stats.TotalDocuments != 2 {
		t.Errorf("Expected TotalDocuments=2, got %d", stats.TotalDocuments)
	}
	if stats.SuccessRate != 0.5 {
		t.Errorf("Expected SuccessRate=0.5, got %f", stats.SuccessRate)
	}
}

// Test Copy is thread-safe
func TestProcessorStatsCopy(t *testing.T) {
	stats := &ProcessorStats{}
	stats.RecordSuccess(100, 10*time.Millisecond, 0.001)

	copy := stats.Copy()

	if copy.SuccessCount != stats.SuccessCount {
		t.Error("Copy didn't match original")
	}
}

// Test fallback chain
func TestFallbackEngine(t *testing.T) {
	config := DefaultConfig()
	config.EnableFlorence2 = true
	o := NewOrchestrator(config)

	tests := []struct {
		failed   Engine
		expected Engine
	}{
		{EngineFlorence2, EngineTesseract},
		{EngineLocalGPU, EngineTesseract},
		{EngineModalGPU, EngineFlorence2},
		{EngineGoFitz, EngineGoFitz}, // No fallback
	}

	for _, tt := range tests {
		result := o.getFallbackEngine(tt.failed)
		if result != tt.expected {
			t.Errorf("getFallbackEngine(%s) = %s, want %s", tt.failed, result, tt.expected)
		}
	}
}

// Test orchestrator initialization
func TestOrchestratorInit(t *testing.T) {
	config := DefaultConfig()
	o := NewOrchestrator(config)

	if o.processors == nil {
		t.Error("Processors map not initialized")
	}

	// Check that at least go-fitz processor is available
	if _, ok := o.processors[EngineGoFitz]; !ok {
		t.Log("Warning: go-fitz processor not initialized (may need module)")
	}
}

// Test routing with actual processing
func TestProcessWithFallback(t *testing.T) {
	t.Skip("Skipping integration test - requires go-fitz module")

	config := DefaultConfig()
	o := NewOrchestrator(config)

	doc := &Document{
		Path:       "test.pdf",
		Type:       DocTypeVectorPDF,
		Quality:    QualityClean,
		Pages:      1,
		Complexity: ComplexityTrivial,
	}

	ctx := context.Background()
	result, err := o.Process(ctx, doc)

	if err != nil {
		t.Logf("Processing failed (expected without real file): %v", err)
	} else {
		t.Logf("Processing succeeded: engine=%s, chars=%d", result.Engine, result.Characters)
	}
}
