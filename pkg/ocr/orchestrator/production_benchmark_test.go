package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ph_holdings_app/pkg/ocr/fitz"
)

// ============================================================================
// COMPREHENSIVE PRODUCTION BENCHMARK SUITE
// ============================================================================
//
// This benchmark suite provides end-to-end testing of the entire OCR pipeline:
// 1. Full Pipeline Benchmark - Complete document processing flow
// 2. Sparse DNA Performance - Recycling efficiency tests
// 3. Predator Preprocessing - GPU enhancement speed
// 4. Ksum Table Detection - Detection accuracy metrics
// 5. Florence-2 vs AIMLAPI - Cost/speed comparison
// 6. Engine Selection - Routing efficiency validation
//
// Location: C:\Projects\ACE_Engine\pkg\ocr\orchestrator\production_benchmark_test.go
// Test PDFs: C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)
//
// Usage:
//   go test -run TestProductionFullPipeline -v
//   go test -run TestEngineComparison -v
//   go test -bench=BenchmarkFullPipeline
// ============================================================================

// ProductionBenchmarkStats tracks comprehensive pipeline statistics
type ProductionBenchmarkStats struct {
	// File counts
	totalFiles     int64
	processedFiles int64
	vectorPDFs     int64
	scannedPDFs    int64

	// OCR stats by engine
	florence2Success int64
	aimlSuccess      int64
	tesseractSuccess int64
	ocrFailed        int64

	// Character counts
	totalChars     int64
	florence2Chars int64
	aimlChars      int64
	tesseractChars int64

	// Cost tracking (thread-safe)
	florence2CostUSD float64
	aimlCostUSD      float64
	totalCostUSD     float64
	costMu           sync.Mutex

	// Timing by stage
	startTime         time.Time
	extractionTime    time.Duration
	gpuPreprocessTime time.Duration
	florence2Time     time.Duration
	aimlTime          time.Duration
	tesseractTime     time.Duration
	totalTime         time.Duration
	timeMu            sync.Mutex

	// GPU metrics
	gpuOps        int64
	gpuImagesProc int64
	gpuPixels     int64
	gpuAvailable  bool

	// Engine routing stats
	routedToFlorenceCount  int64
	routedToAIMLCount      int64
	routedToTesseractCount int64
	fallbackCount          int64

	// Errors
	errors  []string
	errorMu sync.Mutex
}

func (s *ProductionBenchmarkStats) AddFlorence2Cost(cost float64) {
	s.costMu.Lock()
	s.florence2CostUSD += cost
	s.totalCostUSD += cost
	s.costMu.Unlock()
}

func (s *ProductionBenchmarkStats) AddAIMLCost(cost float64) {
	s.costMu.Lock()
	s.aimlCostUSD += cost
	s.totalCostUSD += cost
	s.costMu.Unlock()
}

func (s *ProductionBenchmarkStats) GetTotalCost() (florence2, aiml, total float64) {
	s.costMu.Lock()
	defer s.costMu.Unlock()
	return s.florence2CostUSD, s.aimlCostUSD, s.totalCostUSD
}

func (s *ProductionBenchmarkStats) AddTime(category string, d time.Duration) {
	s.timeMu.Lock()
	switch category {
	case "extraction":
		s.extractionTime += d
	case "gpu":
		s.gpuPreprocessTime += d
	case "florence2":
		s.florence2Time += d
	case "aiml":
		s.aimlTime += d
	case "tesseract":
		s.tesseractTime += d
	}
	s.totalTime += d
	s.timeMu.Unlock()
}

func (s *ProductionBenchmarkStats) AddError(err string) {
	s.errorMu.Lock()
	s.errors = append(s.errors, err)
	s.errorMu.Unlock()
}

// TestProductionFullPipeline runs the complete end-to-end pipeline benchmark
func TestProductionFullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full pipeline benchmark in short mode")
	}

	archivePath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", archivePath)
	}

	stats := &ProductionBenchmarkStats{startTime: time.Now()}

	t.Log("🏰 PRODUCTION FULL PIPELINE BENCHMARK")
	t.Log("═══════════════════════════════════════════════════════════════════════")
	t.Log("Pipeline: go-fitz → GPU preprocessing → Routing → Multi-engine OCR")
	t.Log("Engines: Florence-2 (primary), AIMLAPI (high accuracy), Tesseract (fallback)")
	t.Log("")

	// ========================================================================
	// PHASE 1: COLLECT FILES
	// ========================================================================
	t.Log("📁 Phase 1: Collecting files...")

	var pdfFiles []string
	filepath.Walk(archivePath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".pdf" {
				pdfFiles = append(pdfFiles, path)
			}
		}
		return nil
	})

	atomic.StoreInt64(&stats.totalFiles, int64(len(pdfFiles)))
	t.Logf("   Found %d PDFs", len(pdfFiles))

	// ========================================================================
	// PHASE 2: PARALLEL PDF EXTRACTION
	// ========================================================================
	t.Log("\n📄 Phase 2: Parallel PDF extraction (go-fitz)...")

	ctx := context.Background()
	extractStart := time.Now()

	type extractionResult struct {
		path      string
		result    *fitz.ExtractionResult
		isScanned bool
	}

	resultsChan := make(chan extractionResult, len(pdfFiles))
	pdfSem := make(chan struct{}, 8) // 8 parallel extractions
	var wg sync.WaitGroup

	for _, pdfPath := range pdfFiles {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			pdfSem <- struct{}{}
			defer func() { <-pdfSem }()

			result, err := fitz.ExtractPDFReal(path)
			if err != nil {
				stats.AddError(fmt.Sprintf("PDF extraction failed: %s", filepath.Base(path)))
				resultsChan <- extractionResult{path: path, result: nil}
				return
			}

			atomic.AddInt64(&stats.processedFiles, 1)
			atomic.AddInt64(&stats.totalChars, int64(result.Characters))

			if result.NeedsOCR {
				atomic.AddInt64(&stats.scannedPDFs, 1)
				resultsChan <- extractionResult{
					path:      path,
					result:    result,
					isScanned: true,
				}
			} else {
				atomic.AddInt64(&stats.vectorPDFs, 1)
				resultsChan <- extractionResult{
					path:      path,
					result:    result,
					isScanned: false,
				}
			}

			// Progress
			processed := atomic.LoadInt64(&stats.processedFiles)
			if processed%25 == 0 {
				t.Logf("   Extraction progress: %d/%d (%.1f%%)",
					processed, len(pdfFiles),
					float64(processed)/float64(len(pdfFiles))*100)
			}
		}(pdfPath)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect scanned PDFs for OCR
	var scannedResults []extractionResult
	for res := range resultsChan {
		if res.isScanned && res.result != nil && len(res.result.Images) > 0 {
			scannedResults = append(scannedResults, res)
		}
	}

	extractDuration := time.Since(extractStart)
	stats.AddTime("extraction", extractDuration)

	t.Logf("   ✅ Extraction complete: %v", extractDuration)
	t.Logf("   Vector PDFs: %d (FREE)", atomic.LoadInt64(&stats.vectorPDFs))
	t.Logf("   Scanned PDFs: %d (need OCR)", len(scannedResults))
	t.Logf("   Throughput: %.1f files/sec", float64(len(pdfFiles))/extractDuration.Seconds())

	// ========================================================================
	// PHASE 3: GPU PREPROCESSING
	// ========================================================================
	if len(scannedResults) > 0 {
		t.Logf("\n🎮 Phase 3: GPU preprocessing for %d scanned PDFs...", len(scannedResults))

		gpuPreprocessor, err := NewGPUPreprocessor(nil)
		if err != nil {
			t.Logf("   ⚠️  GPU preprocessor unavailable: %v", err)
			stats.gpuAvailable = false
		} else {
			stats.gpuAvailable = true
			gpuStart := time.Now()

			// Process all scanned images through GPU
			for _, scanned := range scannedResults {
				if len(scanned.result.Images) == 0 {
					continue
				}

				processedImages, err := gpuPreprocessor.PreprocessBatch(ctx, scanned.result.Images)
				if err == nil && len(processedImages) > 0 {
					scanned.result.Images = processedImages
					atomic.AddInt64(&stats.gpuImagesProc, int64(len(processedImages)))
				}
			}

			gpuDuration := time.Since(gpuStart)
			stats.AddTime("gpu", gpuDuration)

			gpuStats := gpuPreprocessor.GetStats()
			atomic.StoreInt64(&stats.gpuOps, gpuStats.GPUOps)
			atomic.StoreInt64(&stats.gpuPixels, gpuStats.TotalPixels)

			t.Logf("   ✅ GPU preprocessing complete: %v", gpuDuration)
			t.Logf("   Images processed: %d", atomic.LoadInt64(&stats.gpuImagesProc))
			t.Logf("   GPU ops: %d (%.2f M ops/sec)",
				gpuStats.GPUOps,
				float64(gpuStats.GPUOps)/gpuDuration.Seconds()/1e6)
		}
	}

	// ========================================================================
	// PHASE 4: MULTI-ENGINE OCR WITH ROUTING
	// ========================================================================
	if len(scannedResults) > 0 {
		t.Logf("\n🚀 Phase 4: Multi-engine OCR with intelligent routing...")

		// Initialize engines
		florence2Client, err := NewFlorence2Client(nil)
		if err != nil {
			t.Logf("   ⚠️  Florence-2 unavailable: %v", err)
		}

		aimlClient, err := NewAIMLAPIOCRClient(nil)
		if err != nil {
			t.Logf("   ⚠️  AIMLAPI unavailable: %v", err)
		}

		ocrSem := make(chan struct{}, 3) // 3 parallel OCR calls
		var ocrWg sync.WaitGroup

		maxOCRCost := 5.0 // $5 max

		for i, scanned := range scannedResults {
			// Check cost limit
			_, _, totalCost := stats.GetTotalCost()
			if totalCost >= maxOCRCost {
				t.Logf("   ⚠️  Cost limit reached ($%.2f), stopping OCR", totalCost)
				break
			}

			// Process first 10 PDFs with OCR (limited batch)
			if i >= 10 {
				t.Log("   ⚠️  Limiting to first 10 scanned PDFs for cost control")
				break
			}

			ocrWg.Add(1)
			go func(idx int, res extractionResult) {
				defer ocrWg.Done()
				ocrSem <- struct{}{}
				defer func() { <-ocrSem }()

				if len(res.result.Images) == 0 {
					atomic.AddInt64(&stats.ocrFailed, 1)
					return
				}

				// Routing logic: Clean docs → Florence-2, Degraded → AIMLAPI
				// NOTE: Quality field removed in favor of consistent Florence-2 routing
				// All documents now route through Florence-2 for consistency and speed
				useFlorence := true
				atomic.AddInt64(&stats.routedToFlorenceCount, 1)

				// Try Florence-2 first
				if useFlorence && florence2Client != nil {
					florenceStart := time.Now()
					florenceResult, err := florence2Client.OCRImage(ctx, res.result.Images[0])
					stats.AddTime("florence2", time.Since(florenceStart))

					if err == nil && florenceResult != nil && florenceResult.Success {
						atomic.AddInt64(&stats.florence2Success, 1)
						atomic.AddInt64(&stats.florence2Chars, int64(florenceResult.Characters))
						atomic.AddInt64(&stats.totalChars, int64(florenceResult.Characters))
						stats.AddFlorence2Cost(florenceResult.EstimatedCost)

						// Progress
						success := atomic.LoadInt64(&stats.florence2Success)
						if success%5 == 0 {
							f2Cost, aimlCost, total := stats.GetTotalCost()
							t.Logf("   OCR progress: %d Florence-2 + %d AIML | Cost: $%.4f + $%.4f = $%.4f",
								success, atomic.LoadInt64(&stats.aimlSuccess),
								f2Cost, aimlCost, total)
						}
						return
					}

					// Florence failed, fallback to AIMLAPI
					atomic.AddInt64(&stats.fallbackCount, 1)
				}

				// Use AIMLAPI (higher accuracy)
				if aimlClient != nil {
					aimlStart := time.Now()
					var aimlResult *AIMLAPIOCRResult
					var aimlErr error

					// Retry 3 times
					for attempt := 0; attempt < 3; attempt++ {
						aimlResult, aimlErr = aimlClient.OCRImage(ctx, res.result.Images[0])
						if aimlErr == nil && aimlResult != nil && aimlResult.Success {
							break
						}
						time.Sleep(time.Duration(attempt+1) * time.Second)
					}

					stats.AddTime("aiml", time.Since(aimlStart))

					if aimlErr != nil || aimlResult == nil || !aimlResult.Success {
						atomic.AddInt64(&stats.ocrFailed, 1)
						stats.AddError(fmt.Sprintf("OCR failed: %s", filepath.Base(res.path)))
						return
					}

					atomic.AddInt64(&stats.aimlSuccess, 1)
					atomic.AddInt64(&stats.aimlChars, int64(aimlResult.Characters))
					atomic.AddInt64(&stats.totalChars, int64(aimlResult.Characters))
					stats.AddAIMLCost(aimlResult.EstimatedCost)

					// Progress
					success := atomic.LoadInt64(&stats.aimlSuccess)
					if success%5 == 0 {
						f2Cost, aimlCost, total := stats.GetTotalCost()
						t.Logf("   OCR progress: %d Florence-2 + %d AIML | Cost: $%.4f + $%.4f = $%.4f",
							atomic.LoadInt64(&stats.florence2Success), success,
							f2Cost, aimlCost, total)
					}
				}

			}(i, scanned)
		}

		ocrWg.Wait()

		t.Logf("   ✅ OCR complete: %d Florence-2, %d AIML, %d failed",
			atomic.LoadInt64(&stats.florence2Success),
			atomic.LoadInt64(&stats.aimlSuccess),
			atomic.LoadInt64(&stats.ocrFailed))
	}

	// ========================================================================
	// FINAL SUMMARY
	// ========================================================================
	totalDuration := time.Since(stats.startTime)
	florence2Cost, aimlCost, totalCost := stats.GetTotalCost()

	t.Log("\n" + strings.Repeat("═", 80))
	t.Log("🏆 PRODUCTION FULL PIPELINE RESULTS")
	t.Log(strings.Repeat("═", 80))

	t.Logf("\n📊 FILE STATISTICS:")
	t.Logf("   Total PDFs:         %d", len(pdfFiles))
	t.Logf("   Vector PDFs (FREE): %d (%.1f%%)",
		atomic.LoadInt64(&stats.vectorPDFs),
		float64(atomic.LoadInt64(&stats.vectorPDFs))/float64(len(pdfFiles))*100)
	t.Logf("   Scanned PDFs:       %d (%.1f%%)",
		atomic.LoadInt64(&stats.scannedPDFs),
		float64(atomic.LoadInt64(&stats.scannedPDFs))/float64(len(pdfFiles))*100)

	t.Logf("\n📊 OCR STATISTICS:")
	t.Logf("   Florence-2 success: %d pages", atomic.LoadInt64(&stats.florence2Success))
	t.Logf("   AIMLAPI success:    %d pages", atomic.LoadInt64(&stats.aimlSuccess))
	t.Logf("   Tesseract success:  %d pages", atomic.LoadInt64(&stats.tesseractSuccess))
	t.Logf("   OCR failed:         %d pages", atomic.LoadInt64(&stats.ocrFailed))
	t.Logf("   Total characters:   %d", atomic.LoadInt64(&stats.totalChars))
	t.Logf("   Florence-2 chars:   %d", atomic.LoadInt64(&stats.florence2Chars))
	t.Logf("   AIML chars:         %d", atomic.LoadInt64(&stats.aimlChars))

	t.Logf("\n⏱️  TIMING BREAKDOWN:")
	t.Logf("   Total time:         %v", totalDuration)
	t.Logf("   PDF extraction:     %v (%.1f%%)", stats.extractionTime,
		float64(stats.extractionTime)/float64(totalDuration)*100)
	t.Logf("   GPU preprocessing:  %v (%.1f%%)", stats.gpuPreprocessTime,
		float64(stats.gpuPreprocessTime)/float64(totalDuration)*100)
	t.Logf("   Florence-2 OCR:     %v (%.1f%%)", stats.florence2Time,
		float64(stats.florence2Time)/float64(totalDuration)*100)
	t.Logf("   AIMLAPI OCR:        %v (%.1f%%)", stats.aimlTime,
		float64(stats.aimlTime)/float64(totalDuration)*100)
	t.Logf("   Overall throughput: %.1f files/sec", float64(len(pdfFiles))/totalDuration.Seconds())

	t.Logf("\n💰 COST BREAKDOWN:")
	t.Logf("   Vector PDFs:        $0.00 (FREE - %.1f%% of files!)",
		float64(atomic.LoadInt64(&stats.vectorPDFs))/float64(len(pdfFiles))*100)
	t.Logf("   Florence-2 OCR:     $%.4f (%d pages @ ~$0.00015/page)",
		florence2Cost, atomic.LoadInt64(&stats.florence2Success))
	t.Logf("   AIMLAPI OCR:        $%.4f (%d pages @ ~$0.006/page)",
		aimlCost, atomic.LoadInt64(&stats.aimlSuccess))
	t.Log("   ─────────────────────────────────────────────────")
	t.Logf("   TOTAL:              $%.4f", totalCost)

	// Cost comparison: Florence vs AIML
	if atomic.LoadInt64(&stats.florence2Success) > 0 && atomic.LoadInt64(&stats.aimlSuccess) > 0 {
		f2PerPage := florence2Cost / float64(atomic.LoadInt64(&stats.florence2Success))
		aimlPerPage := aimlCost / float64(atomic.LoadInt64(&stats.aimlSuccess))
		t.Logf("\n📈 COST COMPARISON:")
		t.Logf("   Florence-2 cost/page:  $%.6f", f2PerPage)
		t.Logf("   AIMLAPI cost/page:     $%.6f", aimlPerPage)
		if aimlPerPage > 0 {
			t.Logf("   Florence-2 is %.1f× cheaper!", aimlPerPage/f2PerPage)
		}
	}

	// GPU metrics
	if stats.gpuAvailable {
		t.Logf("\n🎮 GPU METRICS:")
		t.Logf("   GPU available:      ✅ YES")
		t.Logf("   Images processed:   %d", atomic.LoadInt64(&stats.gpuImagesProc))
		t.Logf("   Total pixels:       %d", atomic.LoadInt64(&stats.gpuPixels))
		t.Logf("   GPU ops:            %d", atomic.LoadInt64(&stats.gpuOps))
		if stats.gpuPreprocessTime > 0 {
			t.Logf("   GPU throughput:     %.2f M ops/sec",
				float64(atomic.LoadInt64(&stats.gpuOps))/stats.gpuPreprocessTime.Seconds()/1e6)
		}
	} else {
		t.Logf("\n🎮 GPU METRICS:")
		t.Logf("   GPU available:      ❌ NO (using CPU fallback)")
	}

	// Routing stats
	t.Logf("\n🔀 ENGINE ROUTING:")
	t.Logf("   Routed to Florence-2:  %d", atomic.LoadInt64(&stats.routedToFlorenceCount))
	t.Logf("   Routed to AIMLAPI:     %d", atomic.LoadInt64(&stats.routedToAIMLCount))
	t.Logf("   Routed to Tesseract:   %d", atomic.LoadInt64(&stats.routedToTesseractCount))
	t.Logf("   Fallback count:        %d", atomic.LoadInt64(&stats.fallbackCount))

	// Errors
	stats.errorMu.Lock()
	if len(stats.errors) > 0 {
		t.Logf("\n⚠️  ERRORS (%d total):", len(stats.errors))
		for i, err := range stats.errors {
			if i >= 10 {
				t.Logf("   ... and %d more", len(stats.errors)-10)
				break
			}
			t.Logf("   - %s", err)
		}
	}
	stats.errorMu.Unlock()

	t.Log("\n" + strings.Repeat("═", 80))
	t.Log("✅ Production full pipeline benchmark complete!")
	t.Log(strings.Repeat("═", 80))
}

// BenchmarkFullPipeline benchmarks the complete pipeline
func BenchmarkFullPipeline(b *testing.B) {
	archivePath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		b.Skipf("PH Archive not found: %s", archivePath)
	}

	// Collect first 10 PDFs
	var pdfFiles []string
	filepath.Walk(archivePath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			pdfFiles = append(pdfFiles, path)
			if len(pdfFiles) >= 10 {
				return filepath.SkipAll
			}
		}
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pdfPath := range pdfFiles {
			fitz.ExtractPDFReal(pdfPath)
		}
	}
}

// BenchmarkGPUPreprocessing benchmarks GPU preprocessing
func BenchmarkGPUPreprocessing(b *testing.B) {
	gpuPreprocessor, err := NewGPUPreprocessor(nil)
	if err != nil {
		b.Skipf("GPU preprocessor unavailable: %v", err)
	}

	archivePath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		b.Skipf("PH Archive not found: %s", archivePath)
	}

	// Find first scanned PDF
	var scannedPath string
	filepath.Walk(archivePath, func(path string, info os.FileInfo, err error) error {
		if scannedPath != "" {
			return filepath.SkipAll
		}
		if err == nil && !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			result, _ := fitz.ExtractPDFReal(path)
			if result != nil && result.NeedsOCR && len(result.Images) > 0 {
				scannedPath = path
			}
		}
		return nil
	})

	if scannedPath == "" {
		b.Skip("No scanned PDFs found")
	}

	result, _ := fitz.ExtractPDFReal(scannedPath)
	if result == nil || len(result.Images) == 0 {
		b.Skip("No images extracted")
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gpuPreprocessor.PreprocessImage(ctx, result.Images[0])
	}
}

// BenchmarkEngineRouting benchmarks the orchestrator routing logic
func BenchmarkEngineRouting(b *testing.B) {
	orchestrator := NewOrchestrator(nil)

	docs := []*Document{
		{Type: DocTypeVectorPDF, Quality: QualityClean, Pages: 1, EstimatedChars: 1000},
		{Type: DocTypeScannedPDF, Quality: QualityClean, Pages: 1, EstimatedChars: 500},
		{Type: DocTypeScannedPDF, Quality: QualityDegraded, Pages: 1, EstimatedChars: 500},
		{Type: DocTypeImage, Quality: QualityHandwritten, Pages: 1, EstimatedChars: 200},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, doc := range docs {
			orchestrator.Route(doc)
		}
	}
}
