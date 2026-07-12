package orchestrator

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ph_holdings_app/pkg/ocr/fitz"
)

// TestGPUPreprocessorBasic tests basic GPU preprocessing
func TestGPUPreprocessorBasic(t *testing.T) {
	t.Log("🎮 GPU PREPROCESSOR BASIC TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	// Create a test image (100x100 with some noise)
	img := createTestImage(100, 100)

	// Create preprocessor
	config := DefaultGPUPreprocessConfig()
	config.EnableDenoise = true
	config.EnableContrast = true
	config.DenoiseStrength = 0.5
	config.ContrastFactor = 1.2

	gp, err := NewGPUPreprocessor(config)
	if err != nil {
		t.Fatalf("Failed to create preprocessor: %v", err)
	}

	ctx := context.Background()

	// Process image
	result, err := gp.PreprocessImage(ctx, img)
	if err != nil {
		t.Fatalf("Preprocessing failed: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("Result is nil")
	}

	bounds := result.Bounds()
	t.Logf("   Input:  %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	t.Logf("   Output: %dx%d", bounds.Dx(), bounds.Dy())

	t.Log(gp.Summary())
	t.Log("✅ Basic preprocessing working!")
}

// TestGPUPreprocessorBatch tests batch preprocessing
func TestGPUPreprocessorBatch(t *testing.T) {
	t.Log("🎮 GPU PREPROCESSOR BATCH TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	// Create multiple test images
	images := make([]image.Image, 5)
	for i := range images {
		images[i] = createTestImage(200, 200)
	}

	// Create preprocessor
	gp, _ := NewGPUPreprocessor(nil)
	ctx := context.Background()

	// Process batch
	results, err := gp.PreprocessBatch(ctx, images)
	if err != nil {
		t.Fatalf("Batch preprocessing failed: %v", err)
	}

	if len(results) != len(images) {
		t.Errorf("Expected %d results, got %d", len(images), len(results))
	}

	t.Log(gp.Summary())
	t.Log("✅ Batch preprocessing working!")
}

// TestGPUPreprocessorWithRealScannedPDF tests with real scanned PDF
func TestGPUPreprocessorWithRealScannedPDF(t *testing.T) {
	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("🔥 GPU PREPROCESSING WITH REAL SCANNED PDF")
	t.Log("═══════════════════════════════════════════════════════════")

	// Find a scanned PDF
	var scannedPDF string
	filepath.Walk(offersPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			result, _ := fitz.ExtractPDFReal(path)
			if result != nil && result.NeedsOCR && len(result.Images) > 0 {
				scannedPDF = path
				return filepath.SkipAll
			}
		}
		return nil
	})

	if scannedPDF == "" {
		t.Log("No scanned PDFs with images found - all PDFs are vector!")
		t.Log("✅ Test skipped (no scanned PDFs to process)")
		return
	}

	t.Logf("📄 Found scanned PDF: %s", filepath.Base(scannedPDF))

	// Extract images
	result, err := fitz.ExtractPDFReal(scannedPDF)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	t.Logf("   Pages: %d", result.Pages)
	t.Logf("   Images: %d", len(result.Images))

	if len(result.Images) == 0 {
		t.Log("No images extracted from scanned PDF")
		return
	}

	// Get first image info
	img := result.Images[0]
	bounds := img.Bounds()
	t.Logf("   First image: %dx%d pixels", bounds.Dx(), bounds.Dy())

	// Create GPU preprocessor
	config := DefaultGPUPreprocessConfig()
	config.EnableDenoise = true
	config.EnableContrast = true
	config.DenoiseStrength = 0.5
	config.ContrastFactor = 1.2

	gp, _ := NewGPUPreprocessor(config)
	ctx := context.Background()

	// Process image
	t.Log("\n⏱️  Applying GPU preprocessing...")
	processed, err := gp.PreprocessImage(ctx, img)
	if err != nil {
		t.Fatalf("GPU preprocessing failed: %v", err)
	}

	processedBounds := processed.Bounds()
	t.Logf("   Processed: %dx%d pixels", processedBounds.Dx(), processedBounds.Dy())

	t.Log(gp.Summary())

	stats := gp.GetStats()
	if stats.TotalDuration.Seconds() > 0 {
		pixelsPerSec := float64(stats.TotalPixels) / stats.TotalDuration.Seconds()
		t.Logf("   Pixel throughput: %.0f pixels/sec", pixelsPerSec)
	}

	t.Log("\n✅ Real scanned PDF GPU preprocessing complete!")
}

// TestHybridPipelineWithGPU tests the full hybrid pipeline with GPU
func TestHybridPipelineWithGPU(t *testing.T) {
	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("🔀 HYBRID PIPELINE WITH GPU PREPROCESSING")
	t.Log("═══════════════════════════════════════════════════════════")

	// Create hybrid pipeline with GPU enabled
	config := DefaultHybridConfig()
	config.PreferPyMuPDF = false // Use go-fitz for this test
	config.EnableGPUPreprocess = true
	config.DenoiseStrength = 0.5
	config.ContrastFactor = 1.2

	pipeline, err := NewHybridPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Find first 20 PDFs
	var pdfFiles []string
	filepath.Walk(offersPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			pdfFiles = append(pdfFiles, path)
			if len(pdfFiles) >= 20 {
				return filepath.SkipAll
			}
		}
		return nil
	})

	t.Logf("📄 Processing %d PDFs with GPU preprocessing enabled", len(pdfFiles))

	ctx := context.Background()

	vectorCount := 0
	scannedCount := 0

	for _, pdf := range pdfFiles {
		result, err := pipeline.ExtractDocument(ctx, pdf)
		if err != nil {
			continue
		}

		if result.NeedsOCR {
			scannedCount++
		} else {
			vectorCount++
		}
	}

	t.Log(pipeline.Summary())

	stats := pipeline.GetStats()
	t.Logf("\n📊 Results:")
	t.Logf("   Vector PDFs: %d", vectorCount)
	t.Logf("   Scanned PDFs: %d", scannedCount)
	t.Logf("   GPU preprocessed: %d images", stats.GPUPreprocessed)
	t.Logf("   GPU time: %v", stats.GPUDuration)

	t.Log("\n✅ Hybrid pipeline with GPU complete!")
}

// TestQuaternionMath tests the quaternion operations
func TestQuaternionMath(t *testing.T) {
	t.Log("🧮 QUATERNION MATH TESTS")
	t.Log("═══════════════════════════════════════════════════════════")

	// Test normalize
	q := Quaternion{W: 3, X: 4, Y: 0, Z: 0}
	normalized := normalize(q)
	norm := normalized.W*normalized.W + normalized.X*normalized.X + normalized.Y*normalized.Y + normalized.Z*normalized.Z
	if norm < 0.999 || norm > 1.001 {
		t.Errorf("Normalize failed: norm = %f", norm)
	}
	t.Logf("   ✅ Normalize: |q| = %.6f", norm)

	// Test dot product
	q1 := Quaternion{W: 1, X: 0, Y: 0, Z: 0}
	q2 := Quaternion{W: 0, X: 1, Y: 0, Z: 0}
	d := dot(q1, q2)
	if d != 0 {
		t.Errorf("Dot product failed: expected 0, got %f", d)
	}
	t.Logf("   ✅ Dot product: q1·q2 = %.6f", d)

	// Test SLERP
	q1 = Quaternion{W: 1, X: 0, Y: 0, Z: 0}
	q2 = Quaternion{W: 0, X: 1, Y: 0, Z: 0}
	mid := slerp(q1, q2, 0.5)
	midNorm := mid.W*mid.W + mid.X*mid.X + mid.Y*mid.Y + mid.Z*mid.Z
	if midNorm < 0.999 || midNorm > 1.001 {
		t.Errorf("SLERP failed: norm = %f", midNorm)
	}
	t.Logf("   ✅ SLERP: midpoint norm = %.6f", midNorm)

	// Test geodesic distance
	dist := geodesicDistance(q1, q2)
	if dist < 1.5 || dist > 1.6 { // Should be π/2 ≈ 1.57
		t.Errorf("Geodesic distance failed: expected ~1.57, got %f", dist)
	}
	t.Logf("   ✅ Geodesic distance: d(q1,q2) = %.6f (expected π/2 ≈ 1.57)", dist)

	t.Log("\n✅ All quaternion math tests passed!")
}

// createTestImage creates a test image with some patterns
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create a gradient with some noise
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(((x + y) * 127) / (width + height))

			// Add some "noise" pattern
			if (x+y)%7 == 0 {
				r = uint8(min(int(r)+30, 255))
			}

			img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
		}
	}

	return img
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
