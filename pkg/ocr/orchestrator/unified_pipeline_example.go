package orchestrator

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg" // Register JPEG decoder
	"image/png"
	_ "image/png" // Register PNG decoder
	"os"
	"path/filepath"
	"time"
)

// ExampleUnifiedPipeline_SinglePage demonstrates processing a single page
func ExampleUnifiedPipeline_SinglePage() {
	// Create pipeline with default configuration
	config := DefaultUnifiedPipelineConfig()
	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		panic(err)
	}

	// Load image from file
	img, err := loadImageExample("document.png")
	if err != nil {
		panic(err)
	}

	// Process the page
	ctx := context.Background()
	result, err := pipeline.ProcessPage(ctx, img, 1)
	if err != nil {
		panic(err)
	}

	// Display results
	fmt.Printf("Extracted Text: %s\n", result.Text)
	fmt.Printf("Confidence: %.2f\n", result.Confidence)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Cost: $%.6f\n", result.Cost)

	// Save DNA database for future speedups
	if err := pipeline.SaveDNA(); err != nil {
		fmt.Printf("Warning: Failed to save DNA: %v\n", err)
	}
}

// ExampleUnifiedPipeline_MultiPage demonstrates batch processing
func ExampleUnifiedPipeline_MultiPage() {
	config := DefaultUnifiedPipelineConfig()
	config.EnableParallel = true
	config.WorkerCount = 4 // 4 parallel workers

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		panic(err)
	}

	// Load multiple images
	imageFiles := []string{
		"page1.png",
		"page2.png",
		"page3.png",
		"page4.png",
	}

	var images []image.Image
	for _, file := range imageFiles {
		img, err := loadImageExample(file)
		if err != nil {
			panic(err)
		}
		images = append(images, img)
	}

	// Process batch
	ctx := context.Background()
	results, err := pipeline.ProcessBatch(ctx, images)
	if err != nil {
		panic(err)
	}

	// Display results
	for i, result := range results {
		fmt.Printf("\n=== Page %d ===\n", i+1)
		fmt.Printf("Text: %s\n", truncate(result.Text, 100))
		fmt.Printf("Confidence: %.2f\n", result.Confidence)
		fmt.Printf("Duration: %v\n", result.Duration)
		fmt.Printf("DNA Cache Hit: %v\n", result.DNACacheHit)
		fmt.Printf("Table Detected: %v\n", result.TableDetected)
	}

	// Print summary
	fmt.Println(pipeline.Summary())
}

// ExampleUnifiedPipeline_CustomConfig demonstrates custom configuration
func ExampleUnifiedPipeline_CustomConfig() {
	// Custom configuration for specific use case
	config := &UnifiedPipelineConfig{
		// DNA Fingerprinting - CRITICAL for recurring documents!
		EnableDNASampling:  true,
		DNADatabasePath:    "./invoices_dna.json", // Separate DB per document type
		RecurringThreshold: 2,                     // Lower threshold for invoice headers

		// Table Detection - Essential for financial documents
		EnableTableDetection: true,
		TableOrthogThreshold: 0.3, // More sensitive for light table lines
		TableMinCells:        6,   // Expect larger tables

		// Predator Vision - For scanned/faxed documents
		EnablePredatorVision: true,
		EnableUVChannel:      true,
		EnableSkewCorrection: true,
		UVBoostFactor:        2.0, // Stronger boost for very faded faxes

		// Octonion Color Processing - For color forms
		EnableOctonion:  true,
		InkSeparation:   true,
		DenoiseStrength: 0.4,  // More aggressive denoising
		ContrastEnhance: 1.5,  // Stronger contrast
		BlueInkBoost:    true, // Enhance blue signatures
		RedInkBoost:     true, // Enhance red stamps

		// Florence-2 OCR
		Florence2BaseURL: "https://the maintainer-asymmetrica--florence2-ocr",
		Florence2Timeout: 60 * time.Second,

		// Performance
		EnableParallel: true,
		WorkerCount:    8, // Maximize throughput for batch processing
	}

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Pipeline created with custom config: %+v\n", config)

	// Use pipeline...
	_ = pipeline
}

// ExampleUnifiedPipeline_DNARecycling demonstrates DNA-based recycling
func ExampleUnifiedPipeline_DNARecycling() {
	config := DefaultUnifiedPipelineConfig()
	config.DNADatabasePath = "./company_forms_dna.json"

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		panic(err)
	}

	// Process 100 company forms (many with identical headers/footers)
	var totalDNASaved time.Duration

	for i := 1; i <= 100; i++ {
		img, err := loadImageExample(fmt.Sprintf("form_%03d.png", i))
		if err != nil {
			continue
		}

		result, err := pipeline.ProcessPage(context.Background(), img, i)
		if err != nil {
			continue
		}

		if result.DNACacheHit {
			totalDNASaved += result.Duration
			fmt.Printf("Page %d: DNA cache hit! Saved %v\n", i, result.Duration)
		}
	}

	// Save DNA for next run
	if err := pipeline.SaveDNA(); err != nil {
		fmt.Printf("Warning: Failed to save DNA: %v\n", err)
	}

	stats := pipeline.GetStats()
	fmt.Printf("\n=== DNA Recycling Results ===\n")
	fmt.Printf("Total Pages: %d\n", stats.TotalPages)
	fmt.Printf("DNA Cache Hits: %d (%.1f%%)\n",
		stats.DNACacheHits,
		float64(stats.DNACacheHits)*100/float64(stats.TotalPages))
	fmt.Printf("Time Saved: %v\n", stats.DNATimeSaved)
	fmt.Printf("Speedup Factor: %.2fx\n", stats.DNASpeedupFactor)
}

// ExampleUnifiedPipeline_TableExtraction demonstrates table-focused processing
func ExampleUnifiedPipeline_TableExtraction() {
	config := DefaultUnifiedPipelineConfig()
	// Optimize for table detection
	config.EnableTableDetection = true
	config.TableOrthogThreshold = 0.4
	config.TableMinCells = 4

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		panic(err)
	}

	// Load spreadsheet/table image
	img, err := loadImageExample("financial_table.png")
	if err != nil {
		panic(err)
	}

	result, err := pipeline.ProcessPage(context.Background(), img, 1)
	if err != nil {
		panic(err)
	}

	if result.TableDetected {
		fmt.Println("✓ Table detected!")
		fmt.Printf("  Orthogonality: %.2f\n", result.TableFingerprint.Orthogonality)
		fmt.Printf("  Grid Cells: %d\n", result.TableFingerprint.GridCells)
		fmt.Printf("  Row Peaks: %v\n", result.TableFingerprint.RowPeaks)
		fmt.Printf("  Col Peaks: %v\n", result.TableFingerprint.ColPeaks)

		// Future: Route to specialized table OCR
		// For now, standard OCR is applied
		fmt.Printf("\nExtracted Text:\n%s\n", result.Text)
	} else {
		fmt.Println("No table detected - processed as regular document")
	}
}

// ExampleUnifiedPipeline_DegradedScans demonstrates predator vision for poor quality
func ExampleUnifiedPipeline_DegradedScans() {
	config := DefaultUnifiedPipelineConfig()
	// Optimize for degraded scans
	config.EnablePredatorVision = true
	config.EnableUVChannel = true
	config.EnableSkewCorrection = true
	config.UVBoostFactor = 2.0 // Aggressive enhancement

	config.EnableOctonion = true
	config.DenoiseStrength = 0.5 // Strong denoising
	config.ContrastEnhance = 1.8 // Strong contrast boost

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		panic(err)
	}

	// Load degraded scan
	img, err := loadImageExample("faded_fax.png")
	if err != nil {
		panic(err)
	}

	result, err := pipeline.ProcessPage(context.Background(), img, 1)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Degraded Scan Processing ===")
	fmt.Printf("Predator Vision: %v\n", result.PredatorUsed)
	fmt.Printf("Skew Corrected: %v (%.2f°)\n", result.SkewCorrected, result.SkewAngle)
	fmt.Printf("Octonion Processing: %v\n", result.OctonionUsed)
	fmt.Printf("Confidence: %.2f\n", result.Confidence)
	fmt.Printf("\nRecovered Text:\n%s\n", result.Text)

	// Save processed image to see enhancement
	if result.ProcessedImage != nil {
		saveImageExample(result.ProcessedImage, "faded_fax_enhanced.png")
		fmt.Println("\n✓ Enhanced image saved to faded_fax_enhanced.png")
	}
}

// ExampleUnifiedPipeline_ColorForms demonstrates color document processing
func ExampleUnifiedPipeline_ColorForms() {
	config := DefaultUnifiedPipelineConfig()
	// Optimize for color forms (signatures, stamps)
	config.EnableOctonion = true
	config.InkSeparation = true
	config.BlueInkBoost = true // Enhance blue signatures
	config.RedInkBoost = true  // Enhance red stamps
	config.ContrastEnhance = 1.3

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		panic(err)
	}

	// Load color form
	img, err := loadImageExample("signed_contract.png")
	if err != nil {
		panic(err)
	}

	result, err := pipeline.ProcessPage(context.Background(), img, 1)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Color Form Processing ===")
	fmt.Printf("Octonion Color Processing: %v\n", result.OctonionUsed)
	fmt.Printf("Confidence: %.2f\n", result.Confidence)
	fmt.Printf("\nExtracted Text (with ink enhancement):\n%s\n", result.Text)
}

// ExampleUnifiedPipeline_PerformanceBenchmark demonstrates performance tracking
func ExampleUnifiedPipeline_PerformanceBenchmark() {
	config := DefaultUnifiedPipelineConfig()
	config.EnableParallel = true
	config.WorkerCount = 8

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		panic(err)
	}

	// Process a large batch
	batchSize := 100
	images := make([]image.Image, batchSize)
	for i := 0; i < batchSize; i++ {
		// In real usage, load actual images
		images[i] = createMockImage(800, 600)
	}

	start := time.Now()
	results, err := pipeline.ProcessBatch(context.Background(), images)
	if err != nil {
		panic(err)
	}
	totalDuration := time.Since(start)

	// Analyze performance
	var totalChars int
	var totalCost float64
	for _, result := range results {
		totalChars += len(result.Text)
		totalCost += result.Cost
	}

	fmt.Printf("\n=== Performance Benchmark ===\n")
	fmt.Printf("Batch Size: %d pages\n", batchSize)
	fmt.Printf("Total Duration: %v\n", totalDuration)
	fmt.Printf("Pages/Second: %.2f\n", float64(batchSize)/totalDuration.Seconds())
	fmt.Printf("Avg Page Duration: %v\n", totalDuration/time.Duration(batchSize))
	fmt.Printf("Total Characters: %d\n", totalChars)
	fmt.Printf("Total Cost: $%.4f\n", totalCost)
	fmt.Printf("Cost per Page: $%.6f\n", totalCost/float64(batchSize))

	// Print full summary
	fmt.Println(pipeline.Summary())
}

// Helper functions

func loadImageExample(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func saveImageExample(img image.Image, path string) error {
	// Create output directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create file
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Encode as PNG
	return png.Encode(f, img)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func createMockImage(width, height int) image.Image {
	// Simple mock image for benchmarking
	return image.NewRGBA(image.Rect(0, 0, width, height))
}
