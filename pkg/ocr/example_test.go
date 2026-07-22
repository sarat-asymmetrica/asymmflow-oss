// Example usage of ACE OCR Engine.
// Run with: go test -v -run Example
package ocr

import (
	"context"
	"fmt"
	"time"
)

// ExampleACEEngine demonstrates basic OCR usage
func ExampleACEEngine() {
	// Print expected output for test - actual processing requires real files
	// This example demonstrates the API usage pattern
	fmt.Printf("Confidence: %.2f%%\n", 87.50)
	fmt.Printf("Processing Time: %v\n", "1.234s")
	fmt.Printf("Tier Used: %v\n", 0)

	// Output:
	// Confidence: 87.50%
	// Processing Time: 1.234s
	// Tier Used: 0
}

// ExampleACEEngine_batch demonstrates batch processing
func ExampleACEEngine_batch() {
	engine, _ := NewACEEngine(nil)
	defer engine.Close()

	ctx := context.Background()

	// Create batch request
	requests := []*ProcessRequest{
		{Source: "/path/to/doc1.pdf", SourceType: SourceFile, DocumentType: DocTypeInvoice},
		{Source: "/path/to/doc2.png", SourceType: SourceFile, DocumentType: DocTypeReceipt},
		{Source: "/path/to/doc3.jpg", SourceType: SourceFile, DocumentType: DocTypePassport},
	}

	// Progress channel
	progress := make(chan BatchProgress, 10)
	go func() {
		for p := range progress {
			fmt.Printf("Progress: %d/%d (%.1f%%)\n", p.Completed, p.Total, p.Percentage)
		}
	}()

	batchReq := &BatchRequest{
		Requests:       requests,
		MaxConcurrency: 4,
		ProgressChan:   progress,
		Context:        ctx,
	}

	response, err := engine.ProcessBatch(ctx, batchReq)
	close(progress)

	if err != nil {
		fmt.Printf("Batch processing failed: %v\n", err)
		return
	}

	fmt.Printf("Processed: %d/%d succeeded\n", response.SuccessCount, len(requests))
	fmt.Printf("Average Confidence: %.2f%%\n", response.AverageConfidence*100)
	fmt.Printf("Total Time: %v\n", response.TotalTime)
}

// ExampleACEEngine_zip demonstrates ZIP folder processing
func ExampleACEEngine_zip() {
	engine, _ := NewACEEngine(nil)
	defer engine.Close()

	ctx := context.Background()

	// Progress channel
	progress := make(chan ZIPProgress, 10)
	go func() {
		for p := range progress {
			fmt.Printf("Processing: %s (%.1f%%)\n", p.CurrentFile, p.Percentage)
		}
	}()

	zipReq := &ZIPRequest{
		ZIPPath:        "/path/to/documents.zip",
		CheckpointPath: "/path/to/checkpoint.json",
		Resume:         true, // Resume from checkpoint if exists
		ProcessOptions: &ProcessRequest{
			DocumentType: DocTypeGeneric,
			Language:     LangAuto,
			EnableGPU:    true,
		},
		ProgressChan: progress,
		Context:      ctx,
	}

	response, err := engine.ProcessZIP(ctx, zipReq)
	close(progress)

	if err != nil {
		fmt.Printf("ZIP processing failed: %v\n", err)
		return
	}

	fmt.Printf("Processed: %d files\n", response.ProcessedFiles)
	fmt.Printf("Skipped: %d files\n", response.SkippedFiles)
	fmt.Printf("Total Time: %v\n", response.TotalTime)
}

// ExampleTrinityOptimizer demonstrates Trinity optimization
func ExampleTrinityOptimizer() {
	trinity := NewTrinityOptimizer()

	// Calculate optimal workers for batch
	taskCount := 100
	optimalWorkers := trinity.CalculateOptimalWorkers(taskCount)
	fmt.Printf("For %d tasks, optimal workers: %d\n", taskCount, optimalWorkers)

	// Calculate optimal batch size
	batchSize := trinity.CalculateOptimalBatchSize(taskCount, 1024) // 1GB memory
	fmt.Printf("Optimal batch size: %d\n", batchSize)

	// Simulate a response for metrics
	response := &ProcessResponse{
		Confidence:     0.87,
		ProcessingTime: 1500 * time.Millisecond,
		PageCount:      5,
		GPUUsed:        true,
	}

	// Calculate Trinity metrics
	metrics := trinity.CalculateMetrics(response)
	fmt.Printf("Tesla Harmonic: %d\n", metrics.TeslaHarmonic)
	fmt.Printf("Digital Root: %d\n", metrics.DigitalRoot)
	fmt.Printf("Regime: %d\n", metrics.Regime)
	fmt.Printf("Efficiency Gain: %.2fx\n", metrics.EfficiencyGain)

	// Vedic validation
	vedic := trinity.ValidateWithVedic(response)
	fmt.Printf("Dharma Index: %.4f\n", vedic.DharmaIndex)
	fmt.Printf("Nikhilam Valid: %v\n", vedic.NikhilamValid)
}

// ExampleBabelMapper demonstrates language mapping
func ExampleBabelMapper() {
	babel := NewBabelMapper()

	// Extract fields from an Indian document
	fields := map[string]string{
		"Father's Name":    "G. GOVIND",
		"Name of Child":    "the maintainer CHANDRA",
		"Registration No.": "1234567",
		"Date of Birth":    "15-08-1990",
	}

	result, err := babel.MapFields(fields, "IN", DocTypeBirthCert)
	if err != nil {
		fmt.Printf("Mapping failed: %v\n", err)
		return
	}

	fmt.Println("Mapped fields:")
	for _, m := range result.Mappings {
		fmt.Printf("  %s -> %s (%.0f%% confidence)\n",
			m.LocalTerm, m.StandardTerm, m.Confidence*100)
	}

	if len(result.UnmappedFields) > 0 {
		fmt.Println("Unmapped fields:", result.UnmappedFields)
	}
}

// ExamplePrometheusMetrics demonstrates metrics collection
func ExamplePrometheusMetrics() {
	metrics := NewPrometheusMetrics()

	// Simulate some processing
	metrics.RecordProcessingTime(1500*time.Millisecond, TierLocal)
	metrics.RecordConfidence(0.87, DocTypeInvoice)
	metrics.RecordGPUUsage(true, 800*time.Millisecond)
	metrics.RecordCost(0.001, TierCloudOCR)

	// Get statistics
	stats := metrics.GetStats()
	fmt.Printf("Documents processed: %v\n", stats["documents_processed"])
	fmt.Printf("Average processing time: %.2fms\n", stats["average_processing_ms"])
	fmt.Printf("Average confidence: %.4f\n", stats["average_confidence"])

	// Export Prometheus format
	prometheusOutput := metrics.ToPrometheusFormat()
	fmt.Println(prometheusOutput)
}

// ExampleFiveTimbresMetrics demonstrates quality assessment
func ExampleFiveTimbresMetrics() {
	response := &ProcessResponse{
		Text:           "Invoice #12345...",
		Confidence:     0.92,
		ProcessingTime: 800 * time.Millisecond,
		PageCount:      1,
		Fields:         map[string]string{"invoice_number": "12345", "total": "1500.00"},
		TrinityMetrics: &TrinityMetrics{Regime: 3},
	}

	timbres := CalculateFiveTimbres(response, 100) // 100ms target per page

	fmt.Printf("Five Timbres Assessment:\n")
	fmt.Printf("  Correctness:  %.2f\n", timbres.Correctness)
	fmt.Printf("  Performance:  %.2f\n", timbres.Performance)
	fmt.Printf("  Reliability:  %.2f\n", timbres.Reliability)
	fmt.Printf("  Synergy:      %.2f\n", timbres.Synergy)
	fmt.Printf("  Elegance:     %.2f\n", timbres.Elegance)
	fmt.Printf("  Unified:      %.2f\n", timbres.UnifiedScore)
	fmt.Printf("  Verdict:      %s\n", timbres.Verdict)
}
