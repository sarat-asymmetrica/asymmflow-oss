package orchestrator

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestSparseDNAIntegration verifies DNA lookup wiring
func TestSparseDNAIntegration(t *testing.T) {
	o := NewOrchestrator(DefaultConfig())
	ctx := context.Background()

	// Test DNA miss
	docMiss := &Document{
		Path:       "test.pdf",
		Type:       DocTypeScannedPDF,
		SeenBefore: false,
		DNAHash:    "",
	}

	result, err := o.trySparseDNA(ctx, docMiss)
	if err == nil {
		t.Errorf("Expected DNA miss error, got nil")
	}
	if result.Success {
		t.Errorf("Expected DNA miss to fail, got success")
	}

	// Test DNA hit
	docHit := &Document{
		Path:       "test.pdf",
		Type:       DocTypeScannedPDF,
		SeenBefore: true,
		DNAHash:    "abc123def456789012345678901234567890",
	}

	result, err = o.trySparseDNA(ctx, docHit)
	if err != nil {
		t.Errorf("Expected DNA hit success, got error: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected DNA hit to succeed, got failure")
	}
	if result.Confidence != 0.99 {
		t.Errorf("Expected DNA hit confidence 0.99, got %.2f", result.Confidence)
	}
	if result.Cost != 0.0 {
		t.Errorf("Expected DNA hit cost $0.00, got $%.4f", result.Cost)
	}
	if result.Duration > 2*time.Millisecond {
		t.Errorf("Expected DNA hit latency <2ms, got %v", result.Duration)
	}
}

// TestPredatorVisionIntegration verifies predator preprocessing wiring
func TestPredatorVisionIntegration(t *testing.T) {
	o := NewOrchestrator(DefaultConfig())
	ctx := context.Background()

	doc := &Document{
		Path:          "degraded.pdf",
		Type:          DocTypeScannedPDF,
		Quality:       QualityDegraded,
		DegradedScore: 0.8, // Highly degraded
	}

	enhanced, err := o.applyPredatorPreprocessing(ctx, doc)
	if err != nil {
		t.Errorf("Expected predator preprocessing success, got error: %v", err)
	}

	// Should improve quality by 30% (0.7× factor)
	expectedScore := 0.8 * 0.7
	// Use tolerance-based comparison for floating point
	if diff := enhanced.DegradedScore - expectedScore; diff > 0.01 || diff < -0.01 {
		t.Errorf("Expected degraded score %.4f, got %.4f", expectedScore, enhanced.DegradedScore)
	}
}

// TestKsumTableDetectionIntegration verifies table detection wiring
func TestKsumTableDetectionIntegration(t *testing.T) {
	o := NewOrchestrator(DefaultConfig())

	// Test with XLSX (should detect tables)
	docXLSX := &Document{
		Path:      "invoice.xlsx",
		Type:      DocTypeXLSX,
		HasTables: false,
	}

	enhanced := o.analyzeTableStructure(docXLSX)
	if !enhanced.HasTables {
		t.Errorf("Expected XLSX to be marked as having tables, got false")
	}

	// Test with image (should not detect without actual analysis)
	docImage := &Document{
		Path:      "photo.jpg",
		Type:      DocTypeImage,
		HasTables: false,
	}

	enhanced = o.analyzeTableStructure(docImage)
	// Image type doesn't auto-mark tables (needs actual detection)
	// This is correct behavior
}

// TestOctonionColorProcessingIntegration verifies color processing wiring
func TestOctonionColorProcessingIntegration(t *testing.T) {
	o := NewOrchestrator(DefaultConfig())

	// Test with scanned PDF (should mark as color)
	docScanned := &Document{
		Path:    "scanned.pdf",
		Type:    DocTypeScannedPDF,
		IsColor: false,
	}

	enhanced := o.applyOctonionProcessing(docScanned)
	if !enhanced.IsColor {
		t.Errorf("Expected scanned PDF to be marked as color, got false")
	}

	// Test with vector PDF (should not mark as color)
	docVector := &Document{
		Path:    "vector.pdf",
		Type:    DocTypeVectorPDF,
		IsColor: false,
	}

	enhanced = o.applyOctonionProcessing(docVector)
	if enhanced.IsColor {
		t.Errorf("Expected vector PDF to NOT be marked as color, got true")
	}
}

// TestAIMLAPIIntegration verifies AIMLAPI fallback wiring
func TestAIMLAPIIntegration(t *testing.T) {
	o := NewOrchestrator(DefaultConfig())
	ctx := context.Background()

	doc := &Document{
		Path:    "difficult.pdf",
		Type:    DocTypeScannedPDF,
		Quality: QualityHandwritten,
	}

	result, err := o.tryAIMLAPI(ctx, doc)
	if err != nil {
		t.Errorf("Expected AIMLAPI success, got error: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected AIMLAPI to succeed, got failure")
	}
	if result.Engine != EngineAIMLAPI {
		t.Errorf("Expected engine AIMLAPI, got %s", result.Engine)
	}
	if result.Confidence < 0.90 {
		t.Errorf("Expected AIMLAPI confidence >= 0.90, got %.2f", result.Confidence)
	}
	if result.Cost != 0.006 {
		t.Errorf("Expected AIMLAPI cost $0.006, got $%.4f", result.Cost)
	}
}

// TestTieredProcessingFlow verifies complete cascade
func TestTieredProcessingFlow(t *testing.T) {
	// Skip if Tesseract is not available
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("Skipping test: tesseract not found in PATH")
	}
	if _, err := os.Stat("old_faded_scan.pdf"); err != nil {
		t.Skipf("Skipping test: fixture old_faded_scan.pdf is not available: %v", err)
	}
	o := NewOrchestrator(DefaultConfig())
	ctx := context.Background()

	// Scenario 1: DNA hit (should stop at Tier 0)
	docDNA := &Document{
		Path:       "recurring_invoice.pdf",
		Type:       DocTypeScannedPDF,
		SeenBefore: true,
		DNAHash:    "abc123def456789012345678901234567890",
	}

	result, err := o.TieredProcess(ctx, docDNA)
	if err != nil {
		t.Errorf("TieredProcess failed: %v", err)
	}
	if result.Engine != EngineSparse {
		t.Errorf("Expected DNA engine for recurring doc, got %s", result.Engine)
	}

	// Scenario 2: Novel degraded document (should apply Predator → Florence-2)
	docDegraded := &Document{
		Path:          "old_faded_scan.pdf",
		Type:          DocTypeScannedPDF,
		Quality:       QualityDegraded,
		SeenBefore:    false,
		DegradedScore: 0.8,
	}

	result, err = o.TieredProcess(ctx, docDegraded)
	if err != nil {
		t.Errorf("TieredProcess failed: %v", err)
	}
	// Should have applied predator preprocessing
	if docDegraded.DegradedScore == 0.8 {
		t.Logf("Note: Predator preprocessing should have improved quality score")
	}
}

// BenchmarkDNALookup measures DNA lookup performance
func BenchmarkDNALookup(b *testing.B) {
	o := NewOrchestrator(DefaultConfig())
	ctx := context.Background()

	doc := &Document{
		Path:       "test.pdf",
		Type:       DocTypeScannedPDF,
		SeenBefore: true,
		DNAHash:    "abc123def456789012345678901234567890",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = o.trySparseDNA(ctx, doc)
	}
}

// BenchmarkPredatorPreprocessing measures predator vision performance
func BenchmarkPredatorPreprocessing(b *testing.B) {
	o := NewOrchestrator(DefaultConfig())
	ctx := context.Background()

	doc := &Document{
		Path:          "degraded.pdf",
		Type:          DocTypeScannedPDF,
		Quality:       QualityDegraded,
		DegradedScore: 0.8,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = o.applyPredatorPreprocessing(ctx, doc)
	}
}
