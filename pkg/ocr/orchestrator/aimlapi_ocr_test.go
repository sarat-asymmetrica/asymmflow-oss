package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ph_holdings_app/pkg/ocr/fitz"
)

// TestAIMLAPIOCRClient tests the AIMLAPI OCR client
func TestAIMLAPIOCRClient(t *testing.T) {
	t.Log("☁️ AIMLAPI OCR CLIENT TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	// Create client
	client, err := NewAIMLAPIOCRClient(nil) // Uses default config with API key
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Logf("   API Key: %s...%s", client.apiKey[:8], client.apiKey[len(client.apiKey)-4:])
	t.Logf("   Model: %s", client.model)
	t.Logf("   Endpoint: %s", client.endpoint)

	t.Log("✅ AIMLAPI client created successfully!")
}

// TestAIMLAPIOCRWithRealScannedPDF tests OCR on a real scanned PDF
func TestAIMLAPIOCRWithRealScannedPDF(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping AIMLAPI test in short mode (costs money)")
	}

	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("☁️ AIMLAPI OCR WITH REAL SCANNED PDF")
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
		t.Log("No scanned PDFs found - all PDFs are vector!")
		return
	}

	t.Logf("📄 Found scanned PDF: %s", filepath.Base(scannedPDF))

	// Extract image
	result, err := fitz.ExtractPDFReal(scannedPDF)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	if len(result.Images) == 0 {
		t.Fatal("No images extracted")
	}

	img := result.Images[0]
	bounds := img.Bounds()
	t.Logf("   Image size: %dx%d pixels", bounds.Dx(), bounds.Dy())

	// Create AIMLAPI client
	client, err := NewAIMLAPIOCRClient(nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Perform OCR
	t.Log("\n⏱️  Calling AIMLAPI OCR...")
	ocrResult, err := client.OCRImage(ctx, img)
	if err != nil {
		t.Fatalf("OCR failed: %v", err)
	}

	if !ocrResult.Success {
		t.Fatalf("OCR unsuccessful: %s", ocrResult.Error)
	}

	t.Logf("   ✅ OCR successful!")
	t.Logf("   Characters: %d", ocrResult.Characters)
	t.Logf("   Duration: %v", ocrResult.Duration)
	t.Logf("   Cost: $%.6f", ocrResult.EstimatedCost)

	// Show preview of extracted text
	preview := ocrResult.Text
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	t.Logf("\n📝 Extracted text preview:\n%s", preview)

	t.Log(client.Summary())
	t.Log("\n✅ AIMLAPI OCR test complete!")
}

// TestFullPipelineWithAIMLAPI tests the complete pipeline with AIMLAPI OCR
func TestFullPipelineWithAIMLAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full pipeline test in short mode")
	}

	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("🔥 FULL PIPELINE WITH AIMLAPI OCR")
	t.Log("═══════════════════════════════════════════════════════════")

	// Create hybrid pipeline
	config := DefaultHybridConfig()
	config.PreferPyMuPDF = false // Use go-fitz for this test
	config.EnableGPUPreprocess = true

	pipeline, err := NewHybridPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Create AIMLAPI client
	aimlClient, err := NewAIMLAPIOCRClient(nil)
	if err != nil {
		t.Fatalf("Failed to create AIMLAPI client: %v", err)
	}

	// Create GPU preprocessor
	gpuPreprocessor, _ := NewGPUPreprocessor(nil)

	// Find first 10 PDFs (mix of vector and scanned)
	var pdfFiles []string
	filepath.Walk(offersPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			pdfFiles = append(pdfFiles, path)
			if len(pdfFiles) >= 10 {
				return filepath.SkipAll
			}
		}
		return nil
	})

	t.Logf("📄 Processing %d PDFs through full pipeline", len(pdfFiles))

	ctx := context.Background()

	vectorCount := 0
	scannedCount := 0
	ocrCount := 0
	totalChars := 0

	for i, pdfPath := range pdfFiles {
		// Step 1: Extract with go-fitz
		result, err := pipeline.ExtractDocument(ctx, pdfPath)
		if err != nil {
			t.Logf("   ❌ %d. Error: %v", i+1, err)
			continue
		}

		if !result.NeedsOCR {
			// Vector PDF - already have text
			vectorCount++
			totalChars += result.Characters
			t.Logf("   ✅ %d. %s (vector, %d chars)", i+1, filepath.Base(pdfPath), result.Characters)
		} else {
			// Scanned PDF - need OCR
			scannedCount++

			if len(result.Images) > 0 {
				// GPU preprocess
				processedImages, _ := gpuPreprocessor.PreprocessBatch(ctx, result.Images)

				// AIMLAPI OCR (only do first image to save cost)
				if len(processedImages) > 0 {
					ocrResult, err := aimlClient.OCRImage(ctx, processedImages[0])
					if err == nil && ocrResult.Success {
						ocrCount++
						totalChars += ocrResult.Characters
						t.Logf("   ✅ %d. %s (scanned→OCR, %d chars, $%.6f)",
							i+1, filepath.Base(pdfPath), ocrResult.Characters, ocrResult.EstimatedCost)
					} else {
						t.Logf("   ⚠️ %d. %s (scanned, OCR failed)", i+1, filepath.Base(pdfPath))
					}
				}
			}
		}
	}

	t.Log("\n" + strings.Repeat("═", 60))
	t.Log("📊 FULL PIPELINE RESULTS")
	t.Log(strings.Repeat("═", 60))
	t.Logf("   Total PDFs: %d", len(pdfFiles))
	t.Logf("   Vector (free): %d", vectorCount)
	t.Logf("   Scanned: %d", scannedCount)
	t.Logf("   OCR successful: %d", ocrCount)
	t.Logf("   Total characters: %d", totalChars)

	t.Log(pipeline.Summary())
	t.Log(gpuPreprocessor.Summary())
	t.Log(aimlClient.Summary())

	t.Log("\n✅ Full pipeline with AIMLAPI complete!")
}
