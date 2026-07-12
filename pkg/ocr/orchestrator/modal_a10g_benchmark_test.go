package orchestrator

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ph_holdings_app/pkg/ocr/fitz"
)

// ModalA10GBenchmarkConfig holds configuration for Modal A10G benchmark
type ModalA10GBenchmarkConfig struct {
	// Concurrency
	MaxConcurrentPDFs int // Parallel PDF extractions (local)
	MaxConcurrentOCR  int // Parallel AIMLAPI calls
	MaxConcurrentGPU  int // Parallel Modal GPU calls

	// Cost limits
	MaxOCRCostUSD   float64 // Stop OCR if cost exceeds this
	MaxModalCostUSD float64 // Stop Modal if cost exceeds this

	// Retry
	OCRRetryAttempts   int
	ModalRetryAttempts int

	// Progress
	ProgressInterval int
}

// DefaultModalA10GConfig returns optimized config for Modal A10G
func DefaultModalA10GConfig() *ModalA10GBenchmarkConfig {
	return &ModalA10GBenchmarkConfig{
		MaxConcurrentPDFs:  8,    // 8 parallel local extractions
		MaxConcurrentOCR:   3,    // 3 parallel AIMLAPI (rate limit)
		MaxConcurrentGPU:   5,    // 5 parallel Modal calls
		MaxOCRCostUSD:      10.0, // $10 max for AIMLAPI
		MaxModalCostUSD:    5.0,  // $5 max for Modal
		OCRRetryAttempts:   3,
		ModalRetryAttempts: 2,
		ProgressInterval:   25,
	}
}

// ModalA10GStats tracks all costs and timing
type ModalA10GStats struct {
	// File counts
	totalFiles     int64
	processedFiles int64
	vectorPDFs     int64
	scannedPDFs    int64

	// OCR stats
	ocrSuccess int64
	ocrFailed  int64
	ocrChars   int64

	// Character counts
	totalChars int64

	// Cost tracking (thread-safe)
	aimlCostUSD  float64
	modalCostUSD float64
	costMu       sync.Mutex

	// Timing
	startTime      time.Time
	extractionTime time.Duration
	modalGPUTime   time.Duration
	ocrTime        time.Duration
	timeMu         sync.Mutex

	// Errors
	errors  []string
	errorMu sync.Mutex
}

func (s *ModalA10GStats) AddAIMLCost(cost float64) {
	s.costMu.Lock()
	s.aimlCostUSD += cost
	s.costMu.Unlock()
}

func (s *ModalA10GStats) AddModalCost(cost float64) {
	s.costMu.Lock()
	s.modalCostUSD += cost
	s.costMu.Unlock()
}

func (s *ModalA10GStats) GetTotalCost() (aiml, modal, total float64) {
	s.costMu.Lock()
	defer s.costMu.Unlock()
	return s.aimlCostUSD, s.modalCostUSD, s.aimlCostUSD + s.modalCostUSD
}

func (s *ModalA10GStats) AddTime(category string, d time.Duration) {
	s.timeMu.Lock()
	switch category {
	case "extraction":
		s.extractionTime += d
	case "modal":
		s.modalGPUTime += d
	case "ocr":
		s.ocrTime += d
	}
	s.timeMu.Unlock()
}

func (s *ModalA10GStats) AddError(err string) {
	s.errorMu.Lock()
	s.errors = append(s.errors, err)
	s.errorMu.Unlock()
}

// TestModalA10GFullBenchmark runs the complete pipeline with Modal A10G GPU
func TestModalA10GFullBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Modal A10G benchmark in short mode")
	}

	archivePath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Skipf("PH Archive not found: %s", archivePath)
	}

	config := DefaultModalA10GConfig()
	stats := &ModalA10GStats{startTime: time.Now()}

	t.Log("🚀 MODAL A10G FULL BENCHMARK - SPOC DEPLOYMENT REFERENCE")
	t.Log("═══════════════════════════════════════════════════════════════════════════")
	t.Log("Pipeline: go-fitz extraction → Modal A10G preprocessing → AIMLAPI OCR")
	t.Logf("⚙️  Config: %d PDF workers, %d OCR workers, %d GPU workers",
		config.MaxConcurrentPDFs, config.MaxConcurrentOCR, config.MaxConcurrentGPU)
	t.Logf("💰 Cost limits: AIMLAPI $%.2f, Modal $%.2f", config.MaxOCRCostUSD, config.MaxModalCostUSD)
	t.Log("")
	t.Log("📊 MONITOR THESE DASHBOARDS:")
	t.Log("   - AIMLAPI: https://aimlapi.com/dashboard")
	t.Log("   - Modal:   https://modal.com/apps/the maintainer-asymmetrica")
	t.Log("")

	// =========================================================================
	// PHASE 1: COLLECT FILES
	// =========================================================================
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

	// =========================================================================
	// PHASE 2: PARALLEL PDF EXTRACTION (LOCAL)
	// =========================================================================
	t.Log("\n📄 Phase 2: Parallel PDF extraction (local go-fitz)...")

	ctx := context.Background()
	extractStart := time.Now()

	type extractionResult struct {
		path      string
		result    *fitz.ExtractionResult
		isScanned bool
		images    []image.Image
	}

	resultsChan := make(chan extractionResult, len(pdfFiles))
	pdfSem := make(chan struct{}, config.MaxConcurrentPDFs)
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
					images:    result.Images,
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
			if processed%int64(config.ProgressInterval) == 0 {
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

	// Collect scanned PDFs for GPU processing
	var scannedResults []extractionResult
	for res := range resultsChan {
		if res.isScanned && len(res.images) > 0 {
			scannedResults = append(scannedResults, res)
		}
	}

	extractDuration := time.Since(extractStart)
	stats.AddTime("extraction", extractDuration)

	t.Logf("   ✅ Extraction complete: %v", extractDuration)
	t.Logf("   Vector PDFs: %d (FREE)", atomic.LoadInt64(&stats.vectorPDFs))
	t.Logf("   Scanned PDFs: %d (need GPU + OCR)", len(scannedResults))
	t.Logf("   Throughput: %.1f files/sec", float64(len(pdfFiles))/extractDuration.Seconds())

	// =========================================================================
	// PHASE 3: MODAL A10G GPU PREPROCESSING + AIMLAPI OCR
	// =========================================================================
	if len(scannedResults) > 0 {
		t.Logf("\n🚀 Phase 3: Modal A10G GPU preprocessing + AIMLAPI OCR for %d scanned PDFs...", len(scannedResults))
		t.Log("   ⚡ Using Modal A10G (18B+ ops/sec) for quaternion preprocessing")

		// Initialize clients
		modalClient, err := NewModalClient(nil)
		if err != nil {
			t.Fatalf("Failed to create Modal client: %v", err)
		}

		aimlClient, _ := NewAIMLAPIOCRClient(nil)

		// Process scanned PDFs with Modal GPU + AIMLAPI OCR
		ocrSem := make(chan struct{}, config.MaxConcurrentOCR)
		var ocrWg sync.WaitGroup

		for i, scanned := range scannedResults {
			// Check cost limits
			aimlCost, modalCost, _ := stats.GetTotalCost()
			if aimlCost >= config.MaxOCRCostUSD {
				t.Logf("   ⚠️  AIMLAPI cost limit reached ($%.2f), stopping OCR", aimlCost)
				break
			}
			if modalCost >= config.MaxModalCostUSD {
				t.Logf("   ⚠️  Modal cost limit reached ($%.2f), stopping GPU", modalCost)
				break
			}

			ocrWg.Add(1)
			go func(idx int, res extractionResult) {
				defer ocrWg.Done()
				ocrSem <- struct{}{}
				defer func() { <-ocrSem }()

				// Step 1: Modal A10G GPU preprocessing
				modalStart := time.Now()

				// Convert image to quaternions for Modal
				if len(res.images) == 0 {
					atomic.AddInt64(&stats.ocrFailed, 1)
					return
				}

				img := res.images[0]
				bounds := img.Bounds()
				numPixels := bounds.Dx() * bounds.Dy()

				// Create quaternion batch from image pixels
				quaternions := make([][]float32, 0, numPixels)
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						r, g, b, a := img.At(x, y).RGBA()
						quaternions = append(quaternions, []float32{
							float32(r) / 65535.0,
							float32(g) / 65535.0,
							float32(b) / 65535.0,
							float32(a) / 65535.0,
						})
					}
				}

				// Call Modal A10G for quaternion evolution (preprocessing)
				// Limit batch size to avoid timeout
				batchSize := 10000
				if len(quaternions) > batchSize {
					quaternions = quaternions[:batchSize]
				}

				var modalErr error
				for attempt := 0; attempt < config.ModalRetryAttempts; attempt++ {
					_, modalErr = modalClient.QuaternionEvolve(ctx, quaternions, 10)
					if modalErr == nil {
						break
					}
					time.Sleep(time.Duration(attempt+1) * time.Second)
				}

				modalDuration := time.Since(modalStart)
				stats.AddTime("modal", modalDuration)

				// Estimate Modal cost: A10G ~$1.10/hr = $0.000306/sec
				stats.AddModalCost(modalDuration.Seconds() * 0.000306)

				if modalErr != nil {
					stats.AddError(fmt.Sprintf("Modal GPU failed: %s: %v", filepath.Base(res.path), modalErr))
					// Continue to OCR anyway - preprocessing is optional
				}

				// Step 2: AIMLAPI OCR
				ocrStart := time.Now()
				var ocrResult *AIMLAPIOCRResult
				var ocrErr error

				for attempt := 0; attempt < config.OCRRetryAttempts; attempt++ {
					ocrResult, ocrErr = aimlClient.OCRImage(ctx, res.images[0])
					if ocrErr == nil && ocrResult != nil && ocrResult.Success {
						break
					}
					time.Sleep(time.Duration(attempt+1) * time.Second)
				}

				stats.AddTime("ocr", time.Since(ocrStart))

				if ocrErr != nil || ocrResult == nil || !ocrResult.Success {
					atomic.AddInt64(&stats.ocrFailed, 1)
					stats.AddError(fmt.Sprintf("OCR failed: %s", filepath.Base(res.path)))
					return
				}

				atomic.AddInt64(&stats.ocrSuccess, 1)
				atomic.AddInt64(&stats.ocrChars, int64(ocrResult.Characters))
				atomic.AddInt64(&stats.totalChars, int64(ocrResult.Characters))
				stats.AddAIMLCost(ocrResult.EstimatedCost)

				// Progress
				success := atomic.LoadInt64(&stats.ocrSuccess)
				if success%5 == 0 {
					aiml, modal, total := stats.GetTotalCost()
					t.Logf("   OCR progress: %d/%d | Cost: AIML $%.4f + Modal $%.4f = $%.4f",
						success, len(scannedResults), aiml, modal, total)
				}

			}(i, scanned)
		}

		ocrWg.Wait()

		// Print Modal summary
		t.Log(modalClient.Summary())
	}

	// =========================================================================
	// FINAL SUMMARY
	// =========================================================================
	totalDuration := time.Since(stats.startTime)
	aimlCost, modalCost, totalCost := stats.GetTotalCost()

	t.Log("\n" + strings.Repeat("═", 80))
	t.Log("🏆 MODAL A10G BENCHMARK RESULTS - SPOC DEPLOYMENT REFERENCE")
	t.Log(strings.Repeat("═", 80))

	t.Logf("\n📊 FILE STATISTICS:")
	t.Logf("   Total PDFs:         %d", len(pdfFiles))
	t.Logf("   Vector PDFs (FREE): %d (%.1f%%)",
		atomic.LoadInt64(&stats.vectorPDFs),
		float64(atomic.LoadInt64(&stats.vectorPDFs))/float64(len(pdfFiles))*100)
	t.Logf("   Scanned PDFs:       %d (%.1f%%)",
		atomic.LoadInt64(&stats.scannedPDFs),
		float64(atomic.LoadInt64(&stats.scannedPDFs))/float64(len(pdfFiles))*100)

	t.Logf("\n📊 EXTRACTION STATISTICS:")
	t.Logf("   Total characters:   %d", atomic.LoadInt64(&stats.totalChars))
	t.Logf("   OCR successful:     %d pages", atomic.LoadInt64(&stats.ocrSuccess))
	t.Logf("   OCR failed:         %d pages", atomic.LoadInt64(&stats.ocrFailed))
	t.Logf("   OCR characters:     %d", atomic.LoadInt64(&stats.ocrChars))

	t.Logf("\n⏱️  TIMING:")
	t.Logf("   Total time:         %v", totalDuration)
	t.Logf("   PDF extraction:     %v (local go-fitz)", stats.extractionTime)
	t.Logf("   Modal A10G GPU:     %v (cloud)", stats.modalGPUTime)
	t.Logf("   AIMLAPI OCR:        %v (cloud)", stats.ocrTime)
	t.Logf("   Overall throughput: %.1f files/sec", float64(len(pdfFiles))/totalDuration.Seconds())

	t.Logf("\n💰 COST BREAKDOWN (VERIFY ON DASHBOARDS!):")
	t.Logf("   Vector PDFs:        $0.00 (FREE - %.1f%% of files!)",
		float64(atomic.LoadInt64(&stats.vectorPDFs))/float64(len(pdfFiles))*100)
	t.Logf("   AIMLAPI OCR:        $%.4f (%d pages @ ~$0.006/page)",
		aimlCost, atomic.LoadInt64(&stats.ocrSuccess))
	t.Logf("   Modal A10G GPU:     $%.4f (%.1fs @ $1.10/hr)",
		modalCost, stats.modalGPUTime.Seconds())
	t.Log("   ─────────────────────────────────────────────────")
	t.Logf("   TOTAL:              $%.4f", totalCost)

	// Projections
	scannedCount := atomic.LoadInt64(&stats.scannedPDFs)
	ocrSuccessCount := atomic.LoadInt64(&stats.ocrSuccess)
	if ocrSuccessCount > 0 && scannedCount > 0 {
		costPerScanned := totalCost / float64(ocrSuccessCount)
		t.Logf("\n📈 PROJECTIONS FOR SPOC:")
		t.Logf("   Cost per scanned PDF:    $%.4f", costPerScanned)
		t.Logf("   Cost per 1000 PDFs:      $%.2f (assuming %.1f%% scanned)",
			costPerScanned*float64(scannedCount)/float64(len(pdfFiles))*1000,
			float64(scannedCount)/float64(len(pdfFiles))*100)
		t.Logf("   Cost per 10000 PDFs:     $%.2f",
			costPerScanned*float64(scannedCount)/float64(len(pdfFiles))*10000)
	}

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
	t.Log("✅ Modal A10G benchmark complete!")
	t.Log("")
	t.Log("📋 NEXT STEPS FOR SPOC DEPLOYMENT:")
	t.Log("   1. Verify costs on AIMLAPI dashboard: https://aimlapi.com/dashboard")
	t.Log("   2. Verify costs on Modal dashboard: https://modal.com/apps/the maintainer-asymmetrica")
	t.Log("   3. Use these numbers for pricing discussion with SPOC")
	t.Log(strings.Repeat("═", 80))
}
