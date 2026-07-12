package octonion

import (
	"context"
	"fmt"
	"image"
	"time"
)

// IntegrationExample demonstrates how to use octonion processing in ACE Engine
type IntegrationExample struct {
	processor *ColorProcessor
	stats     ProcessingStats
}

// ProcessingStats tracks overall pipeline statistics
type ProcessingStats struct {
	TotalDocuments      int
	ColorDocuments      int
	BWDocuments         int
	OctonionProcessTime time.Duration
	AccuracyImprovement float64 // Percentage improvement
}

// NewIntegrationExample creates a new example processor
func NewIntegrationExample() *IntegrationExample {
	return &IntegrationExample{
		processor: NewColorProcessor(DefaultColorProcessorConfig()),
		stats:     ProcessingStats{},
	}
}

// ProcessDocument decides whether to use octonion processing
func (ie *IntegrationExample) ProcessDocument(ctx context.Context, img image.Image) (image.Image, error) {
	start := time.Now()
	ie.stats.TotalDocuments++

	// Analyze if document needs color processing
	needsColor := ie.isColorDocument(img)

	if needsColor {
		ie.stats.ColorDocuments++

		// Apply octonion processing
		enhanced, err := ie.processor.Process(ctx, img)
		if err != nil {
			return nil, fmt.Errorf("octonion processing failed: %w", err)
		}

		ie.stats.OctonionProcessTime += time.Since(start)
		return enhanced, nil
	}

	// Black and white - skip octonion processing
	ie.stats.BWDocuments++
	return img, nil
}

// isColorDocument determines if image has significant color information
func (ie *IntegrationExample) isColorDocument(img image.Image) bool {
	bounds := img.Bounds()

	// Sample every 10th pixel for speed
	colorPixels := 0
	totalSamples := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 10 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 10 {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			// Check if RGB channels differ significantly
			maxDiff := max(abs(int(r8)-int(g8)), abs(int(r8)-int(b8)), abs(int(g8)-int(b8)))
			if maxDiff > 15 {
				colorPixels++
			}
			totalSamples++
		}
	}

	// If >5% of pixels have color variation, process as color
	colorRatio := float64(colorPixels) / float64(totalSamples)
	return colorRatio > 0.05
}

// GetStats returns processing statistics
func (ie *IntegrationExample) GetStats() ProcessingStats {
	return ie.stats
}

// ResetStats resets statistics
func (ie *IntegrationExample) ResetStats() {
	ie.stats = ProcessingStats{}
}

// PrintReport prints a summary report
func (ie *IntegrationExample) PrintReport() {
	fmt.Println("\n=== Octonion Processing Report ===")
	fmt.Printf("Total Documents: %d\n", ie.stats.TotalDocuments)
	fmt.Printf("  Color:         %d (%.1f%%)\n", ie.stats.ColorDocuments,
		100.0*float64(ie.stats.ColorDocuments)/float64(ie.stats.TotalDocuments))
	fmt.Printf("  B&W:           %d (%.1f%%)\n", ie.stats.BWDocuments,
		100.0*float64(ie.stats.BWDocuments)/float64(ie.stats.TotalDocuments))
	fmt.Printf("Processing Time: %v\n", ie.stats.OctonionProcessTime)

	if ie.stats.ColorDocuments > 0 {
		avgTime := ie.stats.OctonionProcessTime / time.Duration(ie.stats.ColorDocuments)
		fmt.Printf("Avg Time/Doc:    %v\n", avgTime)
	}

	if ie.stats.AccuracyImprovement > 0 {
		fmt.Printf("Accuracy Boost:  +%.1f%%\n", ie.stats.AccuracyImprovement)
	}
	fmt.Println("==================================")
}

// Example usage patterns

// ExampleBlueInkCheck demonstrates processing blue ink checks
func ExampleBlueInkCheck() {
	config := &ColorProcessorConfig{
		InkSeparation:   true,
		DenoiseStrength: 0.4,  // Moderate denoising
		ContrastEnhance: 1.3,  // Boost contrast
		BlueInkBoost:    true, // Critical for blue ink!
		RedInkBoost:     false,
		WorkerCount:     4,
	}

	processor := NewColorProcessor(config)
	fmt.Printf("Blue ink processor configured: %s\n", processor)
}

// ExampleRedInkAnnotations demonstrates processing red ink annotations
func ExampleRedInkAnnotations() {
	config := &ColorProcessorConfig{
		InkSeparation:   true,
		DenoiseStrength: 0.3, // Light denoising
		ContrastEnhance: 1.2, // Slight contrast
		BlueInkBoost:    false,
		RedInkBoost:     true, // Critical for red ink!
		WorkerCount:     4,
	}

	processor := NewColorProcessor(config)
	fmt.Printf("Red ink processor configured: %s\n", processor)
}

// ExampleFadedDocument demonstrates processing faded color documents
func ExampleFadedDocument() {
	config := &ColorProcessorConfig{
		InkSeparation:   true,
		DenoiseStrength: 0.5,  // Aggressive denoising
		ContrastEnhance: 1.5,  // Strong contrast boost
		BlueInkBoost:    true, // Boost all ink colors
		RedInkBoost:     true,
		WorkerCount:     4,
	}

	processor := NewColorProcessor(config)
	fmt.Printf("Faded document processor configured: %s\n", processor)
}

// ExampleMinimalProcessing demonstrates light-touch processing
func ExampleMinimalProcessing() {
	config := &ColorProcessorConfig{
		InkSeparation:   true,
		DenoiseStrength: 0.1,  // Minimal denoising
		ContrastEnhance: 1.05, // Very slight contrast
		BlueInkBoost:    false,
		RedInkBoost:     false,
		WorkerCount:     2,
	}

	processor := NewColorProcessor(config)
	fmt.Printf("Minimal processor configured: %s\n", processor)
}

// Helper functions

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// ExamplePipelineIntegration shows how to integrate with ACE Engine pipeline
func ExamplePipelineIntegration() {
	ctx := context.Background()

	// Create integration example
	pipeline := NewIntegrationExample()

	// Simulate processing documents
	// In real usage, these would be actual images
	var documents []image.Image // Loaded from PDFs/images

	for _, doc := range documents {
		// Process with automatic color detection
		enhanced, err := pipeline.ProcessDocument(ctx, doc)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Enhanced image ready for OCR
		_ = enhanced

		// Continue with standard OCR pipeline...
	}

	// Print summary
	pipeline.PrintReport()
}

// ExampleBatchProcessing demonstrates batch processing with stats
func ExampleBatchProcessing() {
	config := DefaultColorProcessorConfig()
	config.DenoiseStrength = 0.4
	config.ContrastEnhance = 1.3

	processor := NewColorProcessor(config)

	// Process multiple images
	var images []image.Image // Loaded from files

	for i, img := range images {
		enhanced, err := processor.Process(context.Background(), img)
		if err != nil {
			fmt.Printf("Image %d failed: %v\n", i, err)
			continue
		}
		_ = enhanced
	}

	// Get overall statistics
	stats := processor.GetStats()
	fmt.Printf("Processed %d images\n", stats.ImagesProcessed)
	fmt.Printf("Total pixels: %d\n", stats.TotalPixels)
	fmt.Printf("Total time: %v\n", stats.Duration)
	fmt.Printf("Throughput: %.2f Mpx/sec\n", stats.PixelsPerSec/1e6)
}
