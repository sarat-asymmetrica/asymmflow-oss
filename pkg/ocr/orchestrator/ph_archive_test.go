package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/ocr/fitz"
)

// TestFullPHArchive tests the orchestrator on the complete Acme Instrumentation archive
func TestFullPHArchive(t *testing.T) {
	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("🏰 DIGITIZATION KINGDOM - FULL PH ARCHIVE TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	// Collect all files
	var allFiles []string
	supportedExts := map[string]bool{
		".pdf": true, ".docx": true, ".xlsx": true,
		".rtf": true, ".msg": true,
		".jpg": true, ".jpeg": true, ".png": true,
	}

	err := filepath.Walk(offersPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if supportedExts[ext] {
				allFiles = append(allFiles, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk directory: %v", err)
	}

	t.Logf("📁 Total files found: %d", len(allFiles))

	// Classify files by type
	filesByType := make(map[string][]string)
	for _, f := range allFiles {
		ext := strings.ToLower(filepath.Ext(f))
		filesByType[ext] = append(filesByType[ext], f)
	}

	t.Log("\n📊 File Distribution:")
	for ext, files := range filesByType {
		t.Logf("   %s: %d files", ext, len(files))
	}

	// Create orchestrator
	config := DefaultConfig()
	config.EnableLocalGPU = true
	config.EnableModalGPU = false // Don't use cloud for this test
	config.EnableAIMLAPI = false  // Don't use cloud for this test
	config.WilliamsBatching = true

	orch := NewOrchestrator(config)

	// Calculate Williams optimal batch
	optimalBatch := WilliamsBatchSize(len(allFiles))
	t.Logf("\n🔢 Williams Optimal Batch: %d", optimalBatch)

	// Convert files to Documents with classification
	docs := make([]*Document, 0, len(allFiles))

	t.Log("\n⏱️  Classifying documents...")
	classifyStart := time.Now()

	for _, path := range allFiles {
		ext := strings.ToLower(filepath.Ext(path))

		doc := &Document{
			Path:    path,
			Quality: QualityClean, // Assume clean unless proven otherwise
		}

		// Classify by extension
		switch ext {
		case ".pdf":
			// Need to check if vector or scanned
			doc.Type = DocTypeVectorPDF // Will be updated during processing
		case ".docx":
			doc.Type = DocTypeDOCX
		case ".xlsx":
			doc.Type = DocTypeXLSX
		case ".rtf":
			doc.Type = DocTypeRTF
		case ".msg":
			doc.Type = DocTypeMSG
		case ".jpg", ".jpeg", ".png":
			doc.Type = DocTypeImage
			doc.Quality = QualityClean // Images need OCR
		default:
			doc.Type = DocTypeUnknown
		}

		docs = append(docs, doc)
	}

	classifyDuration := time.Since(classifyStart)
	t.Logf("   Classification complete in %v", classifyDuration)

	// Route documents
	t.Log("\n🛤️  Routing documents to engines...")
	routing := orch.RouteBatch(docs)

	t.Log("\n📊 Routing Results:")
	for engine, engineDocs := range routing {
		t.Logf("   %s: %d documents", engine, len(engineDocs))
	}

	// Process PDFs with go-fitz to get accurate classification
	t.Log("\n📄 Processing PDFs with go-fitz...")

	pdfDocs := routing[EngineGoFitz]
	if len(pdfDocs) == 0 {
		// All PDFs should route to go-fitz initially
		for _, doc := range docs {
			if doc.Type == DocTypeVectorPDF || doc.Type == DocTypeDOCX || doc.Type == DocTypeXLSX {
				pdfDocs = append(pdfDocs, doc)
			}
		}
	}

	vectorCount := 0
	scannedCount := 0
	totalChars := 0
	totalPages := 0
	var processingDuration time.Duration

	processStart := time.Now()

	for i, doc := range pdfDocs {
		if doc.Type != DocTypeVectorPDF {
			continue // Skip non-PDFs for now
		}

		result, err := fitz.ExtractPDFReal(doc.Path)
		if err != nil {
			continue
		}

		if result.NeedsOCR {
			scannedCount++
			doc.Type = DocTypeScannedPDF
		} else {
			vectorCount++
			totalChars += result.Characters
			totalPages += result.Pages
		}

		// Progress every 100 files
		if (i+1)%100 == 0 {
			t.Logf("   Progress: %d/%d PDFs processed...", i+1, len(pdfDocs))
		}
	}

	processingDuration = time.Since(processStart)

	// Re-route with accurate classification
	t.Log("\n🔄 Re-routing with accurate PDF classification...")
	routing = orch.RouteBatch(docs)

	t.Log("\n📊 Final Routing:")
	for engine, engineDocs := range routing {
		t.Logf("   %s: %d documents", engine, len(engineDocs))
	}

	// Summary
	t.Log("\n" + strings.Repeat("═", 60))
	t.Log("📊 FULL PH ARCHIVE RESULTS")
	t.Log(strings.Repeat("═", 60))
	t.Logf("   Total files: %d", len(allFiles))
	t.Logf("   PDFs processed: %d", vectorCount+scannedCount)
	t.Logf("   Vector PDFs (FREE): %d (%.1f%%)", vectorCount, float64(vectorCount)/float64(vectorCount+scannedCount)*100)
	t.Logf("   Scanned PDFs (need OCR): %d (%.1f%%)", scannedCount, float64(scannedCount)/float64(vectorCount+scannedCount)*100)
	t.Logf("   Total characters: %d", totalChars)
	t.Logf("   Total pages: %d", totalPages)
	t.Logf("   Processing time: %v", processingDuration)

	if processingDuration.Seconds() > 0 {
		t.Logf("   Throughput: %.1f files/sec", float64(vectorCount+scannedCount)/processingDuration.Seconds())
		t.Logf("   Char throughput: %.0f chars/sec", float64(totalChars)/processingDuration.Seconds())
	}

	// Cost estimation
	t.Log("\n💰 COST ESTIMATION:")

	// go-fitz handles vector PDFs for FREE
	goFitzCost := 0.0
	goFitzDocs := vectorCount

	// Scanned PDFs need OCR
	// Option 1: Local GPU (FREE)
	localGPUCost := 0.0
	localGPUTime := float64(scannedCount) * 2.0 // ~2s per scanned PDF

	// Option 2: Modal A10G (~$0.01 per 1000)
	modalCost := float64(scannedCount) * 0.00001
	modalTime := float64(scannedCount) * 0.1 // ~0.1s per PDF at scale

	// Option 3: AIMLAPI (~$0.001 per page)
	aimlCost := float64(scannedCount) * 0.001 // Assume 1 page average
	aimlTime := float64(scannedCount) * 2.0   // ~2s per page

	t.Logf("   go-fitz (vector PDFs): %d docs, $%.4f, already done!", goFitzDocs, goFitzCost)
	t.Logf("   Local GPU (scanned): %d docs, $%.4f, ~%.0fs", scannedCount, localGPUCost, localGPUTime)
	t.Logf("   Modal A10G (scanned): %d docs, $%.4f, ~%.0fs", scannedCount, modalCost, modalTime)
	t.Logf("   AIMLAPI (scanned): %d docs, $%.4f, ~%.0fs", scannedCount, aimlCost, aimlTime)

	// Recommendation
	t.Log("\n🎯 RECOMMENDATION:")
	if scannedCount < 10 {
		t.Log("   → Use Local GPU for scanned PDFs (FREE, fast enough)")
	} else if scannedCount < 100 {
		t.Log("   → Use Local GPU or Modal A10G (both good options)")
	} else {
		t.Log("   → Use Modal A10G for scanned PDFs (faster at scale)")
	}

	t.Log("\n✅ Full PH Archive test complete!")
}

// TestPHArchiveWithRealExtraction runs full extraction on a subset
func TestPHArchiveWithRealExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full extraction in short mode")
	}

	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", offersPath)
	}

	t.Log("🔥 FULL EXTRACTION TEST - First 100 PDFs")
	t.Log("═══════════════════════════════════════════════════════════")

	// Collect first 100 PDFs
	var pdfFiles []string
	filepath.Walk(offersPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			pdfFiles = append(pdfFiles, path)
			if len(pdfFiles) >= 100 {
				return filepath.SkipAll
			}
		}
		return nil
	})

	t.Logf("📄 Processing %d PDFs", len(pdfFiles))

	// Process with go-fitz
	stats := fitz.NewPipelineStats(len(pdfFiles))
	startTime := time.Now()

	for i, pdfPath := range pdfFiles {
		result, err := fitz.ExtractPDFReal(pdfPath)
		if err != nil {
			result = &fitz.ExtractionResult{
				Success: false,
				Error:   err,
				Method:  "error",
			}
		}
		stats.Update(result)

		if (i+1)%25 == 0 {
			t.Logf("   Progress: %d/%d", i+1, len(pdfFiles))
		}
	}

	totalTime := time.Since(startTime)
	stats.TotalDuration = totalTime

	t.Log(stats.Summary())
	t.Log("✅ Full extraction test complete!")
}

// TestOrchestrationSimulation simulates full orchestration
func TestOrchestrationSimulation(t *testing.T) {
	t.Log("🎮 ORCHESTRATION SIMULATION")
	t.Log("═══════════════════════════════════════════════════════════")

	// Simulate Acme Instrumentation document mix based on real data
	// From benchmark: 556 PDFs (490 vector, 66 scanned), 165 DOCX, 115 XLSX, etc.

	docs := make([]*Document, 0)

	// Add vector PDFs (88%)
	for i := 0; i < 490; i++ {
		docs = append(docs, &Document{
			Path:           fmt.Sprintf("vector_%d.pdf", i),
			Type:           DocTypeVectorPDF,
			Quality:        QualityClean,
			Pages:          5,
			EstimatedChars: 15000,
		})
	}

	// Add scanned PDFs (12%)
	for i := 0; i < 66; i++ {
		docs = append(docs, &Document{
			Path:           fmt.Sprintf("scanned_%d.pdf", i),
			Type:           DocTypeScannedPDF,
			Quality:        QualityClean,
			Pages:          2,
			EstimatedChars: 1000,
		})
	}

	// Add DOCX
	for i := 0; i < 165; i++ {
		docs = append(docs, &Document{
			Path:           fmt.Sprintf("doc_%d.docx", i),
			Type:           DocTypeDOCX,
			Quality:        QualityClean,
			Pages:          3,
			EstimatedChars: 5000,
		})
	}

	// Add XLSX
	for i := 0; i < 115; i++ {
		docs = append(docs, &Document{
			Path:           fmt.Sprintf("sheet_%d.xlsx", i),
			Type:           DocTypeXLSX,
			Quality:        QualityClean,
			Pages:          1,
			EstimatedChars: 10000,
		})
	}

	// Add images
	for i := 0; i < 68; i++ {
		docs = append(docs, &Document{
			Path:           fmt.Sprintf("image_%d.jpg", i),
			Type:           DocTypeImage,
			Quality:        QualityClean,
			Pages:          1,
			EstimatedChars: 500,
		})
	}

	t.Logf("📊 Simulated %d documents (Acme Instrumentation mix)", len(docs))

	// Create orchestrator
	config := DefaultConfig()
	config.BatchThresholdForModal = 10
	orch := NewOrchestrator(config)

	// Route
	routing := orch.RouteBatch(docs)

	t.Log("\n🛤️ Routing Results:")
	totalCost := 0.0
	totalTime := 0.0

	for engine, engineDocs := range routing {
		cap := orch.engines[engine]
		engineTime := float64(len(engineDocs)) / cap.ThroughputPerSec
		engineCost := float64(len(engineDocs)) * cap.CostPerDoc

		totalCost += engineCost
		totalTime += engineTime

		t.Logf("   %s: %d docs, ~%.1fs, $%.4f", engine, len(engineDocs), engineTime, engineCost)
	}

	t.Log("\n📊 SIMULATION SUMMARY:")
	t.Logf("   Total documents: %d", len(docs))
	t.Logf("   Estimated time: %.1f seconds (%.1f minutes)", totalTime, totalTime/60)
	t.Logf("   Estimated cost: $%.4f", totalCost)
	t.Logf("   Throughput: %.1f docs/sec", float64(len(docs))/totalTime)

	// Williams batch analysis
	optimalBatch := WilliamsBatchSize(len(docs))
	t.Logf("\n🔢 Williams Analysis:")
	t.Logf("   Optimal batch size: %d", optimalBatch)
	t.Logf("   Number of batches: %d", (len(docs)+optimalBatch-1)/optimalBatch)

	t.Log("\n✅ Orchestration simulation complete!")
}

// BenchmarkFullPHArchive benchmarks the full archive processing
func BenchmarkFullPHArchive(b *testing.B) {
	offersPath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(offersPath); os.IsNotExist(err) {
		b.Skipf("PH Archive not found: %s", offersPath)
	}

	// Collect first 50 PDFs for benchmark
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pdf := range pdfFiles {
			_, _ = fitz.ExtractPDFReal(pdf)
		}
	}
}
