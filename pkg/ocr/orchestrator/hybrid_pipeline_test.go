package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHybridPipelinePyMuPDF tests PyMuPDF as primary engine
func TestHybridPipelinePyMuPDF(t *testing.T) {
	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("🔀 HYBRID PIPELINE TEST - PyMuPDF Primary + go-fitz Fallback")
	t.Log("═══════════════════════════════════════════════════════════")

	// Create hybrid pipeline
	config := DefaultHybridConfig()
	config.PreferPyMuPDF = true
	config.FallbackToGoFitz = true
	config.EnableGPUPreprocess = true
	config.MaxConcurrent = 4

	pipeline, err := NewHybridPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Collect first 20 PDFs for quick test
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

	t.Logf("📄 Testing with %d PDFs", len(pdfFiles))

	ctx := context.Background()

	// Process each PDF
	vectorCount := 0
	scannedCount := 0
	totalChars := 0

	for i, pdfPath := range pdfFiles {
		result, err := pipeline.ExtractDocument(ctx, pdfPath)
		if err != nil {
			t.Logf("   ❌ %d. Error: %v", i+1, err)
			continue
		}

		if result.NeedsOCR {
			scannedCount++
		} else {
			vectorCount++
			totalChars += result.Characters
		}

		// Show first few results
		if i < 5 {
			preview := result.Text
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			preview = strings.ReplaceAll(preview, "\n", " ")
			t.Logf("   ✅ %d. %s (%s, %d chars)", i+1, filepath.Base(pdfPath), result.Method, result.Characters)
		}
	}

	t.Log(pipeline.Summary())

	stats := pipeline.GetStats()
	t.Logf("\n📊 Results:")
	t.Logf("   Vector PDFs: %d", vectorCount)
	t.Logf("   Scanned PDFs: %d", scannedCount)
	t.Logf("   Total chars: %d", totalChars)
	t.Logf("   PyMuPDF success: %d", stats.PyMuPDFSuccess)
	t.Logf("   go-fitz fallback: %d", stats.GoFitzFallback)

	t.Log("\n✅ Hybrid pipeline test complete!")
}

// TestHybridPipelineFullArchive tests the full PH archive
func TestHybridPipelineFullArchive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full archive test in short mode")
	}

	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("🔥 FULL HYBRID PIPELINE TEST - All PDFs")
	t.Log("═══════════════════════════════════════════════════════════")

	// Create hybrid pipeline
	config := DefaultHybridConfig()
	config.PreferPyMuPDF = true
	config.FallbackToGoFitz = true
	config.EnableGPUPreprocess = true
	config.MaxConcurrent = 8

	pipeline, err := NewHybridPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Collect ALL PDFs
	var pdfFiles []string
	filepath.Walk(offersPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			pdfFiles = append(pdfFiles, path)
		}
		return nil
	})

	t.Logf("📄 Processing %d PDFs with hybrid pipeline", len(pdfFiles))

	// Calculate Williams optimal batch
	optimalBatch := WilliamsBatchSize(len(pdfFiles))
	t.Logf("🔢 Williams optimal batch: %d", optimalBatch)

	ctx := context.Background()

	// Process in batches
	results, err := pipeline.ExtractBatch(ctx, pdfFiles)
	if err != nil {
		t.Fatalf("Batch extraction failed: %v", err)
	}

	// Analyze results
	vectorCount := 0
	scannedCount := 0
	errorCount := 0
	totalChars := 0

	for _, result := range results {
		if result == nil || !result.Success {
			errorCount++
			continue
		}

		if result.NeedsOCR {
			scannedCount++
		} else {
			vectorCount++
			totalChars += result.Characters
		}
	}

	t.Log(pipeline.Summary())

	t.Log("\n" + strings.Repeat("═", 60))
	t.Log("📊 FULL ARCHIVE HYBRID RESULTS")
	t.Log(strings.Repeat("═", 60))
	t.Logf("   Total PDFs: %d", len(pdfFiles))
	t.Logf("   Vector (FREE): %d (%.1f%%)", vectorCount, float64(vectorCount)/float64(len(pdfFiles))*100)
	t.Logf("   Scanned (need OCR): %d (%.1f%%)", scannedCount, float64(scannedCount)/float64(len(pdfFiles))*100)
	t.Logf("   Errors: %d", errorCount)
	t.Logf("   Total characters: %d", totalChars)

	stats := pipeline.GetStats()
	if stats.TotalDuration.Seconds() > 0 {
		t.Logf("   Throughput: %.1f files/sec", float64(stats.TotalDocuments)/stats.TotalDuration.Seconds())
	}

	t.Log("\n✅ Full hybrid pipeline test complete!")
}

// TestPyMuPDFvGoFitz compares PyMuPDF and go-fitz performance
func TestPyMuPDFvGoFitz(t *testing.T) {
	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("⚔️ PyMuPDF vs go-fitz COMPARISON")
	t.Log("═══════════════════════════════════════════════════════════")

	// Collect 50 PDFs for comparison
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

	t.Logf("📄 Comparing on %d PDFs", len(pdfFiles))

	ctx := context.Background()

	// Test PyMuPDF
	t.Log("\n🐍 Testing PyMuPDF...")
	pyConfig := DefaultHybridConfig()
	pyConfig.PreferPyMuPDF = true
	pyConfig.FallbackToGoFitz = false // No fallback for fair comparison

	pyPipeline, err := NewHybridPipeline(pyConfig)
	if err != nil {
		t.Logf("PyMuPDF not available: %v", err)
	} else {
		for _, pdf := range pdfFiles {
			_, _ = pyPipeline.ExtractDocument(ctx, pdf)
		}
		pyStats := pyPipeline.GetStats()
		t.Logf("   Documents: %d", pyStats.TotalDocuments)
		t.Logf("   Duration: %v", pyStats.PyMuPDFDuration)
		if pyStats.PyMuPDFDuration.Seconds() > 0 {
			t.Logf("   Throughput: %.1f files/sec", float64(pyStats.TotalDocuments)/pyStats.PyMuPDFDuration.Seconds())
		}
	}

	// Test go-fitz
	t.Log("\n🔷 Testing go-fitz...")
	goConfig := DefaultHybridConfig()
	goConfig.PreferPyMuPDF = false // Use go-fitz only

	goPipeline, _ := NewHybridPipeline(goConfig)
	for _, pdf := range pdfFiles {
		_, _ = goPipeline.ExtractDocument(ctx, pdf)
	}
	goStats := goPipeline.GetStats()
	t.Logf("   Documents: %d", goStats.TotalDocuments)
	t.Logf("   Duration: %v", goStats.TotalDuration)
	if goStats.TotalDuration.Seconds() > 0 {
		t.Logf("   Throughput: %.1f files/sec", float64(goStats.TotalDocuments)/goStats.TotalDuration.Seconds())
	}

	t.Log("\n✅ Comparison complete!")
}
