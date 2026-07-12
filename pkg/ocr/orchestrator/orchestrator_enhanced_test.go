// Enhanced Orchestrator Tests - Mathematical Engine Routing
//
// Tests the new smart routing logic:
// 1. Sparse DNA lookup (FREE, instant)
// 2. Characteristic-based routing (tables, color, degraded)
// 3. Tiered processing cascade
//
// Built: December 22, 2025
package orchestrator

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestRoute_SparseDNA(t *testing.T) {
	config := DefaultConfig()
	config.EnableSparse = true
	o := NewOrchestrator(config)

	doc := &Document{
		Path:       "test.pdf",
		Type:       DocTypeScannedPDF,
		SeenBefore: true, // DNA hit!
		DNAHash:    "abc123",
	}

	engine := o.Route(doc)
	if engine != EngineSparse {
		t.Errorf("Expected EngineSparse for DNA hit, got %s", engine)
	}
}

func TestRoute_TableDetection(t *testing.T) {
	config := DefaultConfig()
	config.EnableUnified = false // Disable unified to test specific routing
	config.EnableKsum = true
	o := NewOrchestrator(config)

	doc := &Document{
		Path:      "invoice.pdf",
		Type:      DocTypeScannedPDF,
		HasTables: true, // Ksum detected tables!
	}

	engine := o.Route(doc)
	if engine != EngineKsum {
		t.Errorf("Expected EngineKsum for table document, got %s", engine)
	}
}

func TestRoute_ColorDocument(t *testing.T) {
	config := DefaultConfig()
	config.EnableUnified = false // Disable unified to test specific routing
	config.EnableOctonion = true
	o := NewOrchestrator(config)

	doc := &Document{
		Path:    "colored.pdf",
		Type:    DocTypeScannedPDF,
		IsColor: true, // Octonion 8D processing!
	}

	engine := o.Route(doc)
	if engine != EngineOctonion {
		t.Errorf("Expected EngineOctonion for color document, got %s", engine)
	}
}

func TestRoute_DegradedQuality(t *testing.T) {
	config := DefaultConfig()
	config.EnableUnified = false // Disable unified to test specific routing
	config.EnablePredator = true
	o := NewOrchestrator(config)

	doc := &Document{
		Path:    "faded.pdf",
		Type:    DocTypeScannedPDF,
		Quality: QualityDegraded, // Predator preprocessing!
	}

	engine := o.Route(doc)
	if engine != EnginePredator {
		t.Errorf("Expected EnginePredator for degraded document, got %s", engine)
	}
}

func TestRoute_UnifiedPipeline(t *testing.T) {
	config := DefaultConfig()
	config.EnableUnified = true // Full mathematical pipeline!
	o := NewOrchestrator(config)

	doc := &Document{
		Path: "complex.pdf",
		Type: DocTypeScannedPDF,
	}

	engine := o.Route(doc)
	if engine != EngineUnified {
		t.Errorf("Expected EngineUnified when enabled, got %s", engine)
	}
}

func TestRoute_Fallback(t *testing.T) {
	config := DefaultConfig()
	config.EnableUnified = false
	config.EnableSparse = false
	config.EnablePredator = false
	config.EnableKsum = false
	config.EnableOctonion = false
	config.EnableFlorence2 = true
	o := NewOrchestrator(config)

	doc := &Document{
		Path: "scan.pdf",
		Type: DocTypeScannedPDF,
	}

	engine := o.Route(doc)
	if engine != EngineFlorence2 {
		t.Errorf("Expected EngineFlorence2 as fallback, got %s", engine)
	}
}

func TestTieredProcess_DNAHit(t *testing.T) {
	config := DefaultConfig()
	config.EnableSparse = true
	config.DNAEnabled = true
	o := NewOrchestrator(config)

	doc := &Document{
		Path:       "recurring.pdf",
		Type:       DocTypeScannedPDF,
		SeenBefore: true,
		DNAHash:    "xyz789",
	}

	ctx := context.Background()
	result, err := o.TieredProcess(ctx, doc)

	if err != nil {
		t.Fatalf("TieredProcess failed: %v", err)
	}

	if result.Engine != EngineSparse {
		t.Errorf("Expected EngineSparse from DNA hit, got %s", result.Engine)
	}

	if result.Cost != 0.0 {
		t.Errorf("Expected zero cost for DNA hit, got %.4f", result.Cost)
	}

	// DNA hits have very high confidence (0.99 or 1.0)
	if result.Confidence < 0.99 {
		t.Errorf("Expected high confidence (>=0.99) for DNA hit, got %.2f", result.Confidence)
	}
}

func TestTieredProcess_Cascade(t *testing.T) {
	// Skip if Tesseract is not available
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("Skipping test: tesseract not found in PATH")
	}
	if _, err := os.Stat("new.pdf"); err != nil {
		t.Skipf("Skipping test: fixture new.pdf is not available: %v", err)
	}
	config := DefaultConfig()
	config.EnableSparse = true
	config.DNAEnabled = true
	config.EnableFlorence2 = true
	o := NewOrchestrator(config)

	doc := &Document{
		Path:       "new.pdf",
		Type:       DocTypeScannedPDF,
		SeenBefore: false, // DNA miss, will cascade to Florence-2
	}

	ctx := context.Background()
	result, err := o.TieredProcess(ctx, doc)

	if err != nil {
		t.Fatalf("TieredProcess failed: %v", err)
	}

	// Should cascade to Florence-2 after DNA miss
	if result.Engine != EngineFlorence2 {
		t.Errorf("Expected EngineFlorence2 after DNA miss, got %s", result.Engine)
	}

	if result.Cost == 0.0 {
		t.Errorf("Expected non-zero cost for Florence-2, got %.4f", result.Cost)
	}
}

func TestEngineCapabilities_MathematicalEngines(t *testing.T) {
	config := DefaultConfig()
	o := NewOrchestrator(config)

	// Verify all mathematical engines are registered
	mathEngines := []Engine{
		EngineSparse,
		EnginePredator,
		EngineKsum,
		EngineOctonion,
		EngineUnified,
	}

	for _, engine := range mathEngines {
		caps, ok := o.engines[engine]
		if !ok {
			t.Errorf("Mathematical engine %s not registered", engine)
			continue
		}

		// All math engines should be FREE (local processing)
		if caps.CostPerDoc != 0.0 {
			t.Errorf("Engine %s should be FREE, but costs %.4f", engine, caps.CostPerDoc)
		}

		// All math engines should be fast (< 1 second latency)
		if caps.Latency > time.Second {
			t.Errorf("Engine %s should be fast, but has latency %v", engine, caps.Latency)
		}
	}
}

func TestEngineCapabilities_SparseDNA(t *testing.T) {
	config := DefaultConfig()
	o := NewOrchestrator(config)

	caps := o.engines[EngineSparse]
	if caps == nil {
		t.Fatal("EngineSparse not registered")
	}

	// Sparse should be FASTEST (1ms latency)
	if caps.Latency != 1*time.Millisecond {
		t.Errorf("EngineSparse should have 1ms latency, got %v", caps.Latency)
	}

	// Sparse should have HIGHEST throughput (1000 docs/sec)
	if caps.ThroughputPerSec < 1000.0 {
		t.Errorf("EngineSparse should have 1000+ docs/sec, got %.1f", caps.ThroughputPerSec)
	}

	// Sparse should support ALL document types
	if len(caps.SupportsTypes) < 3 {
		t.Errorf("EngineSparse should support multiple types, got %d", len(caps.SupportsTypes))
	}
}

func TestDocumentFields_Enhanced(t *testing.T) {
	doc := &Document{
		Path:          "test.pdf",
		HasTables:     true,
		IsColor:       true,
		SeenBefore:    true,
		DNAHash:       "abc123",
		DegradedScore: 0.5,
	}

	// Verify all enhanced fields are accessible
	if !doc.HasTables {
		t.Error("HasTables field not set")
	}
	if !doc.IsColor {
		t.Error("IsColor field not set")
	}
	if !doc.SeenBefore {
		t.Error("SeenBefore field not set")
	}
	if doc.DNAHash == "" {
		t.Error("DNAHash field not set")
	}
	if doc.DegradedScore == 0.0 {
		t.Error("DegradedScore field not set")
	}
}

func BenchmarkRoute_SparseDNA(b *testing.B) {
	config := DefaultConfig()
	o := NewOrchestrator(config)

	doc := &Document{
		Path:       "test.pdf",
		Type:       DocTypeScannedPDF,
		SeenBefore: true,
		DNAHash:    "abc123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = o.Route(doc)
	}
}

func BenchmarkRoute_Traditional(b *testing.B) {
	config := DefaultConfig()
	config.EnableUnified = false
	config.EnableSparse = false
	o := NewOrchestrator(config)

	doc := &Document{
		Path: "test.pdf",
		Type: DocTypeScannedPDF,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = o.Route(doc)
	}
}
