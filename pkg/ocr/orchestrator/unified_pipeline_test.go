package orchestrator

import (
	"context"
	"image"
	"image/color"
	"os"
	"testing"
	"time"
)

// skipIfNoOCRAPI skips tests that require external OCR APIs
func skipIfNoOCRAPI(t *testing.T) {
	t.Helper()
	// These tests hit real external OCR services and should only run when
	// explicitly enabled in an environment with network access.
	if os.Getenv("RUN_EXTERNAL_OCR_TESTS") != "1" {
		t.Skip("Skipping test: set RUN_EXTERNAL_OCR_TESTS=1 to run external OCR integration tests")
	}
	if os.Getenv("FLY_OCR_URL") == "" && os.Getenv("AIMLAPI_KEY") == "" {
		t.Skip("Skipping test: OCR API not configured (set FLY_OCR_URL or AIMLAPI_KEY)")
	}
}

// TestUnifiedPipeline_Basic tests basic pipeline functionality
func TestUnifiedPipeline_Basic(t *testing.T) {
	skipIfNoOCRAPI(t)
	config := DefaultUnifiedPipelineConfig()
	config.EnableParallel = false // Sequential for predictable testing

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Create test image
	img := createTestImageUnified(800, 600)

	// Process single page
	ctx := context.Background()
	result, err := pipeline.ProcessPage(ctx, img, 1)
	if err != nil {
		t.Fatalf("ProcessPage failed: %v", err)
	}

	// Validate result
	if result.PageNumber != 1 {
		t.Errorf("Expected page number 1, got %d", result.PageNumber)
	}

	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	// Check stats
	stats := pipeline.GetStats()
	if stats.TotalPages != 1 {
		t.Errorf("Expected 1 page processed, got %d", stats.TotalPages)
	}

	t.Logf("Result: %+v", result)
	t.Logf("Stats: %+v", stats)
}

// TestUnifiedPipeline_DNACaching tests DNA fingerprinting and caching
func TestUnifiedPipeline_DNACaching(t *testing.T) {
	skipIfNoOCRAPI(t)
	config := DefaultUnifiedPipelineConfig()
	config.EnableDNASampling = true
	config.DNADatabasePath = ""   // In-memory only for testing
	config.RecurringThreshold = 2 // Lower threshold for testing

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Create identical test images
	img := createTestImageUnified(400, 300)

	ctx := context.Background()

	// Process same image multiple times
	for i := 0; i < 3; i++ {
		result, err := pipeline.ProcessPage(ctx, img, i+1)
		if err != nil {
			t.Fatalf("ProcessPage %d failed: %v", i+1, err)
		}

		t.Logf("Page %d: DNACacheHit=%v, Duration=%v",
			i+1, result.DNACacheHit, result.Duration)
	}

	// Check stats
	stats := pipeline.GetStats()
	t.Logf("Total Pages: %d", stats.TotalPages)
	t.Logf("DNA Cache Hits: %d", stats.DNACacheHits)
	t.Logf("DNA Speedup: %.2fx", stats.DNASpeedupFactor)

	// After processing same image 3 times, we should see some caching
	// (though DNA caching works at region level, not full page)
	if stats.TotalPages != 3 {
		t.Errorf("Expected 3 pages, got %d", stats.TotalPages)
	}
}

// TestUnifiedPipeline_TableDetection tests table detection
func TestUnifiedPipeline_TableDetection(t *testing.T) {
	skipIfNoOCRAPI(t)
	config := DefaultUnifiedPipelineConfig()
	config.EnableTableDetection = true
	config.TableOrthogThreshold = 0.4
	config.TableMinCells = 4

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Create image with table-like structure
	img := createTableImage(800, 600, 4, 5) // 4 rows × 5 columns

	ctx := context.Background()
	result, err := pipeline.ProcessPage(ctx, img, 1)
	if err != nil {
		t.Fatalf("ProcessPage failed: %v", err)
	}

	// Check if table was detected
	if result.TableFingerprint == nil {
		t.Error("Expected table fingerprint")
	}

	if result.TableFingerprint != nil {
		t.Logf("Table detected: orthogonality=%.2f, cells=%d",
			result.TableFingerprint.Orthogonality,
			result.TableFingerprint.GridCells)

		if result.TableDetected {
			t.Logf("✓ Table classification successful")
		}
	}

	stats := pipeline.GetStats()
	t.Logf("Tables detected: %d", stats.TablesDetected)
}

// TestUnifiedPipeline_PredatorVision tests predator vision preprocessing
func TestUnifiedPipeline_PredatorVision(t *testing.T) {
	skipIfNoOCRAPI(t)
	config := DefaultUnifiedPipelineConfig()
	config.EnablePredatorVision = true
	config.EnableUVChannel = true
	config.EnableSkewCorrection = true
	config.UVBoostFactor = 1.5

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Create faded/degraded test image
	img := createFadedImage(600, 400)

	ctx := context.Background()
	result, err := pipeline.ProcessPage(ctx, img, 1)
	if err != nil {
		t.Fatalf("ProcessPage failed: %v", err)
	}

	if !result.PredatorUsed {
		t.Error("Expected predator vision to be used")
	}

	if result.ProcessedImage == nil {
		t.Error("Expected processed image")
	}

	t.Logf("Predator vision applied: skew=%.2f°", result.SkewAngle)

	stats := pipeline.GetStats()
	t.Logf("Skews corrected: %d", stats.SkewsCorrected)
}

// TestUnifiedPipeline_OctonionProcessing tests octonion color processing
func TestUnifiedPipeline_OctonionProcessing(t *testing.T) {
	skipIfNoOCRAPI(t)
	config := DefaultUnifiedPipelineConfig()
	config.EnableOctonion = true
	config.InkSeparation = true
	config.DenoiseStrength = 0.3
	config.ContrastEnhance = 1.2
	config.BlueInkBoost = true
	config.RedInkBoost = true

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Create color test image
	img := createColorImage(500, 500)

	ctx := context.Background()
	result, err := pipeline.ProcessPage(ctx, img, 1)
	if err != nil {
		t.Fatalf("ProcessPage failed: %v", err)
	}

	if !result.OctonionUsed {
		t.Error("Expected octonion processing to be used")
	}

	if result.ProcessedImage == nil {
		t.Error("Expected processed image")
	}

	stats := pipeline.GetStats()
	t.Logf("Color processed: %d images", stats.ColorProcessed)
}

// TestUnifiedPipeline_BatchProcessing tests parallel batch processing
func TestUnifiedPipeline_BatchProcessing(t *testing.T) {
	skipIfNoOCRAPI(t)
	config := DefaultUnifiedPipelineConfig()
	config.EnableParallel = true
	config.WorkerCount = 2

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Create batch of test images
	images := make([]image.Image, 5)
	for i := range images {
		images[i] = createTestImageUnified(400+i*100, 300)
	}

	ctx := context.Background()
	start := time.Now()

	results, err := pipeline.ProcessBatch(ctx, images)
	if err != nil {
		t.Fatalf("ProcessBatch failed: %v", err)
	}

	duration := time.Since(start)

	if len(results) != len(images) {
		t.Errorf("Expected %d results, got %d", len(images), len(results))
	}

	for i, result := range results {
		if result.PageNumber != i+1 {
			t.Errorf("Page %d: expected page number %d, got %d",
				i, i+1, result.PageNumber)
		}
	}

	stats := pipeline.GetStats()
	t.Logf("Batch processing: %d pages in %v (%.2f pages/sec)",
		len(images), duration, stats.PagesPerSecond)
}

// TestUnifiedPipeline_AllEngines tests all engines working together
func TestUnifiedPipeline_AllEngines(t *testing.T) {
	skipIfNoOCRAPI(t)
	config := DefaultUnifiedPipelineConfig()
	// Enable everything!
	config.EnableDNASampling = true
	config.EnableTableDetection = true
	config.EnablePredatorVision = true
	config.EnableOctonion = true

	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Create diverse test images
	images := []image.Image{
		createTestImageUnified(600, 400), // Regular document
		createTableImage(800, 600, 5, 4), // Table document
		createFadedImage(500, 500),       // Degraded scan
		createColorImage(600, 600),       // Color document
	}

	ctx := context.Background()
	results, err := pipeline.ProcessBatch(ctx, images)
	if err != nil {
		t.Fatalf("ProcessBatch failed: %v", err)
	}

	// Verify results
	for i, result := range results {
		t.Logf("Page %d:", i+1)
		t.Logf("  DNA Cache Hit: %v", result.DNACacheHit)
		t.Logf("  Table Detected: %v", result.TableDetected)
		t.Logf("  Predator Used: %v", result.PredatorUsed)
		t.Logf("  Octonion Used: %v", result.OctonionUsed)
		t.Logf("  Duration: %v", result.Duration)
		t.Logf("  Cost: $%.6f", result.Cost)
	}

	// Print comprehensive summary
	t.Logf("\n%s", pipeline.Summary())
}

// TestUnifiedPipeline_Summary tests statistics summary
func TestUnifiedPipeline_Summary(t *testing.T) {
	skipIfNoOCRAPI(t)
	config := DefaultUnifiedPipelineConfig()
	pipeline, err := NewUnifiedPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Process a few images
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		img := createTestImageUnified(500+i*100, 400)
		_, err := pipeline.ProcessPage(ctx, img, i+1)
		if err != nil {
			t.Fatalf("ProcessPage %d failed: %v", i+1, err)
		}
	}

	// Get summary
	summary := pipeline.Summary()
	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	t.Logf("\n%s", summary)
}

// Helper functions to create test images

func createTestImageUnified(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with white background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Add some black text-like patterns
	for y := height / 4; y < height*3/4; y += 20 {
		for x := width / 4; x < width*3/4; x++ {
			if x%5 < 3 { // Simple pattern
				img.Set(x, y, color.Black)
			}
		}
	}

	return img
}

func createTableImage(width, height, rows, cols int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with white background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Draw grid lines
	rowSpacing := height / rows
	colSpacing := width / cols

	// Horizontal lines
	for r := 0; r <= rows; r++ {
		y := r * rowSpacing
		if y < height {
			for x := 0; x < width; x++ {
				img.Set(x, y, color.Black)
				if y+1 < height {
					img.Set(x, y+1, color.Black) // Thicker lines
				}
			}
		}
	}

	// Vertical lines
	for c := 0; c <= cols; c++ {
		x := c * colSpacing
		if x < width {
			for y := 0; y < height; y++ {
				img.Set(x, y, color.Black)
				if x+1 < width {
					img.Set(x+1, y, color.Black) // Thicker lines
				}
			}
		}
	}

	return img
}

func createFadedImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with light gray background (faded paper)
	bgColor := color.RGBA{220, 220, 220, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, bgColor)
		}
	}

	// Add very faint text patterns (low contrast)
	textColor := color.RGBA{180, 180, 180, 255}
	for y := height / 4; y < height*3/4; y += 15 {
		for x := width / 4; x < width*3/4; x++ {
			if x%7 < 4 {
				img.Set(x, y, textColor)
			}
		}
	}

	return img
}

func createColorImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with white background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Add blue ink patterns
	blueInk := color.RGBA{0, 0, 200, 255}
	for y := height / 4; y < height/2; y += 10 {
		for x := width / 4; x < width*3/4; x++ {
			if x%6 < 4 {
				img.Set(x, y, blueInk)
			}
		}
	}

	// Add red ink patterns
	redInk := color.RGBA{200, 0, 0, 255}
	for y := height / 2; y < height*3/4; y += 10 {
		for x := width / 4; x < width*3/4; x++ {
			if x%6 < 4 {
				img.Set(x, y, redInk)
			}
		}
	}

	return img
}
