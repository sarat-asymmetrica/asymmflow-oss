package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/ocr/fitz"
)

// TestFullArchiveBenchmark runs the complete pipeline on the full PH archive
func TestFullArchiveBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full archive benchmark in short mode")
	}

	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("🏰 DIGITIZATION KINGDOM - FULL ARCHIVE BENCHMARK")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Log("Pipeline: PyMuPDF batch → go-fitz fallback → GPU preprocess → AIMLAPI OCR")
	t.Log("")

	// Collect ALL files
	var allFiles []string
	supportedExts := map[string]bool{
		".pdf": true, ".docx": true, ".xlsx": true,
		".jpg": true, ".jpeg": true, ".png": true,
	}

	filepath.Walk(offersPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if supportedExts[ext] {
				allFiles = append(allFiles, path)
			}
		}
		return nil
	})

	t.Logf("📁 Total files: %d", len(allFiles))

	// Categorize files
	var pdfFiles, docxFiles, xlsxFiles, imageFiles []string
	for _, f := range allFiles {
		ext := strings.ToLower(filepath.Ext(f))
		switch ext {
		case ".pdf":
			pdfFiles = append(pdfFiles, f)
		case ".docx":
			docxFiles = append(docxFiles, f)
		case ".xlsx":
			xlsxFiles = append(xlsxFiles, f)
		case ".jpg", ".jpeg", ".png":
			imageFiles = append(imageFiles, f)
		}
	}

	t.Logf("   PDFs: %d", len(pdfFiles))
	t.Logf("   DOCX: %d", len(docxFiles))
	t.Logf("   XLSX: %d", len(xlsxFiles))
	t.Logf("   Images: %d", len(imageFiles))

	// Create pipeline components
	hybridConfig := DefaultHybridConfig()
	hybridConfig.PreferPyMuPDF = false // Use go-fitz for this test
	hybridConfig.EnableGPUPreprocess = true

	pipeline, err := NewHybridPipeline(hybridConfig)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	gpuPreprocessor, _ := NewGPUPreprocessor(nil)

	// AIMLAPI client (for scanned PDFs)
	aimlClient, _ := NewAIMLAPIOCRClient(nil)

	ctx := context.Background()
	startTime := time.Now()

	// Results tracking
	vectorPDFs := 0
	scannedPDFs := 0
	ocrSuccess := 0
	totalChars := 0
	totalCost := 0.0

	// Process PDFs
	t.Log("\n📄 Processing PDFs...")
	pdfStart := time.Now()

	for i, pdfPath := range pdfFiles {
		result, err := pipeline.ExtractDocument(ctx, pdfPath)
		if err != nil {
			continue
		}

		if !result.NeedsOCR {
			vectorPDFs++
			totalChars += result.Characters
		} else {
			scannedPDFs++

			// GPU preprocess + AIMLAPI OCR for first 5 scanned PDFs (to save cost)
			if len(result.Images) > 0 && ocrSuccess < 5 {
				// GPU preprocess
				processedImages, _ := gpuPreprocessor.PreprocessBatch(ctx, result.Images)

				// AIMLAPI OCR
				if len(processedImages) > 0 {
					ocrResult, err := aimlClient.OCRImage(ctx, processedImages[0])
					if err == nil && ocrResult.Success {
						ocrSuccess++
						totalChars += ocrResult.Characters
						totalCost += ocrResult.EstimatedCost
					}
				}
			}
		}

		// Progress every 100 files
		if (i+1)%100 == 0 {
			t.Logf("   Progress: %d/%d PDFs...", i+1, len(pdfFiles))
		}
	}

	pdfDuration := time.Since(pdfStart)

	// Summary
	totalDuration := time.Since(startTime)

	t.Log("\n" + strings.Repeat("═", 70))
	t.Log("🏆 FULL ARCHIVE BENCHMARK RESULTS")
	t.Log(strings.Repeat("═", 70))

	t.Logf("\n📊 FILE STATISTICS:")
	t.Logf("   Total files: %d", len(allFiles))
	t.Logf("   PDFs processed: %d", len(pdfFiles))
	t.Logf("   Vector PDFs (FREE): %d (%.1f%%)", vectorPDFs, float64(vectorPDFs)/float64(len(pdfFiles))*100)
	t.Logf("   Scanned PDFs: %d (%.1f%%)", scannedPDFs, float64(scannedPDFs)/float64(len(pdfFiles))*100)

	t.Logf("\n📊 EXTRACTION STATISTICS:")
	t.Logf("   Total characters: %d", totalChars)
	t.Logf("   OCR successful: %d scanned pages", ocrSuccess)

	t.Logf("\n⏱️ TIMING:")
	t.Logf("   PDF processing: %v", pdfDuration)
	t.Logf("   Total time: %v", totalDuration)
	if pdfDuration.Seconds() > 0 {
		t.Logf("   PDF throughput: %.1f files/sec", float64(len(pdfFiles))/pdfDuration.Seconds())
	}

	t.Logf("\n💰 COST:")
	t.Logf("   Vector PDFs: $0.00 (FREE)")
	t.Logf("   AIMLAPI OCR: $%.4f (%d pages)", totalCost, ocrSuccess)
	t.Logf("   Total: $%.4f", totalCost)

	// Pipeline summaries
	t.Log("\n" + pipeline.Summary())
	t.Log(gpuPreprocessor.Summary())
	t.Log(aimlClient.Summary())

	t.Log("\n✅ Full archive benchmark complete!")
}

// TestQuickBenchmark runs a quick benchmark on 50 PDFs
func TestQuickBenchmark(t *testing.T) {
	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("⚡ QUICK BENCHMARK - 50 PDFs")
	t.Log("═══════════════════════════════════════════════════════════════")

	// Collect first 50 PDFs
	var pdfFiles []string
	filepath.Walk(offersPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			pdfFiles = append(pdfFiles, path)
			if len(pdfFiles) >= 50 {
				return filepath.SkipAll
			}
		}
		return nil
	})

	t.Logf("📄 Processing %d PDFs", len(pdfFiles))

	// Use go-fitz directly for speed
	stats := fitz.NewPipelineStats(len(pdfFiles))
	startTime := time.Now()

	for i, pdfPath := range pdfFiles {
		result, err := fitz.ExtractPDFReal(pdfPath)
		if err != nil {
			result = &fitz.ExtractionResult{Success: false, Method: "error", Error: err}
		}
		stats.Update(result)

		if (i+1)%10 == 0 {
			t.Logf("   Progress: %d/%d", i+1, len(pdfFiles))
		}
	}

	stats.TotalDuration = time.Since(startTime)

	t.Log(stats.Summary())
	t.Log("\n✅ Quick benchmark complete!")
}

// TestPipelineComparison compares different pipeline configurations
func TestPipelineComparison(t *testing.T) {
	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("📊 PIPELINE COMPARISON")
	t.Log("═══════════════════════════════════════════════════════════════")

	// Collect 30 PDFs for comparison
	var pdfFiles []string
	filepath.Walk(offersPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			pdfFiles = append(pdfFiles, path)
			if len(pdfFiles) >= 30 {
				return filepath.SkipAll
			}
		}
		return nil
	})

	ctx := context.Background()

	// Test 1: go-fitz only
	t.Log("\n🔷 Test 1: go-fitz only")
	goFitzStart := time.Now()
	goFitzChars := 0
	for _, pdf := range pdfFiles {
		result, _ := fitz.ExtractPDFReal(pdf)
		if result != nil && result.Success {
			goFitzChars += result.Characters
		}
	}
	goFitzDuration := time.Since(goFitzStart)
	t.Logf("   Duration: %v", goFitzDuration)
	t.Logf("   Throughput: %.1f files/sec", float64(len(pdfFiles))/goFitzDuration.Seconds())
	t.Logf("   Characters: %d", goFitzChars)

	// Test 2: Hybrid pipeline (go-fitz + GPU preprocess)
	t.Log("\n🔀 Test 2: Hybrid pipeline (go-fitz + GPU)")
	hybridConfig := DefaultHybridConfig()
	hybridConfig.PreferPyMuPDF = false
	hybridConfig.EnableGPUPreprocess = true
	pipeline, _ := NewHybridPipeline(hybridConfig)

	hybridStart := time.Now()
	hybridChars := 0
	for _, pdf := range pdfFiles {
		result, _ := pipeline.ExtractDocument(ctx, pdf)
		if result != nil && result.Success {
			hybridChars += result.Characters
		}
	}
	hybridDuration := time.Since(hybridStart)
	t.Logf("   Duration: %v", hybridDuration)
	t.Logf("   Throughput: %.1f files/sec", float64(len(pdfFiles))/hybridDuration.Seconds())
	t.Logf("   Characters: %d", hybridChars)

	t.Log(pipeline.Summary())

	t.Log("\n✅ Pipeline comparison complete!")
}
