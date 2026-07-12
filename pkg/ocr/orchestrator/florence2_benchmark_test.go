package orchestrator

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/ocr/fitz"
)

// TestFlorence2VsAIMLAPI compares Florence-2 and AIMLAPI on real scanned PDFs
func TestFlorence2VsAIMLAPI(t *testing.T) {
	t.Log("🌸 FLORENCE-2 vs AIMLAPI BENCHMARK")
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("Goal: Prove Florence-2 is 33× faster and 40× cheaper!")
	t.Log("")

	// Find scanned PDFs in test archive
	archivePath := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Skip("Test archive not available")
	}

	// Find scanned PDFs (they need OCR)
	t.Log("📁 Phase 1: Finding scanned PDFs...")
	var scannedPDFs []string
	err := filepath.Walk(archivePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".pdf") {
			// Quick check if scanned
			result, err := fitz.ExtractPDFReal(path)
			if err == nil && result.NeedsOCR {
				scannedPDFs = append(scannedPDFs, path)
			}
		}
		return nil
	})

	if err != nil || len(scannedPDFs) == 0 {
		t.Skip("No scanned PDFs found")
	}

	// Limit to 5 for benchmark (to control costs)
	testCount := 5
	if len(scannedPDFs) > testCount {
		scannedPDFs = scannedPDFs[:testCount]
	}

	t.Logf("   Found %d scanned PDFs for benchmark", len(scannedPDFs))
	t.Log("")

	// Initialize clients
	t.Log("🔧 Phase 2: Initializing clients...")
	florence2Client, err := NewFlorence2Client(nil)
	if err != nil {
		t.Fatalf("Failed to create Florence-2 client: %v", err)
	}

	aimlClient, err := NewAIMLAPIOCRClient(nil)
	if err != nil {
		t.Fatalf("Failed to create AIMLAPI client: %v", err)
	}

	ctx := context.Background()

	// Results tracking
	type BenchResult struct {
		Provider   string
		File       string
		Duration   time.Duration
		Characters int
		Cost       float64
		Success    bool
		Error      string
	}

	var florence2Results []BenchResult
	var aimlResults []BenchResult

	// Run benchmark
	t.Log("")
	t.Log("🏁 Phase 3: Running OCR benchmark...")
	t.Log("")

	for i, pdfPath := range scannedPDFs {
		filename := filepath.Base(pdfPath)
		t.Logf("📄 PDF %d/%d: %s", i+1, len(scannedPDFs), filename)

		// Extract first page as image
		result, err := fitz.ExtractPDFReal(pdfPath)
		if err != nil || len(result.Images) == 0 {
			t.Logf("   ⚠️  Could not extract images: %v", err)
			continue
		}

		img := result.Images[0]

		// Test Florence-2
		t.Log("   🌸 Testing Florence-2...")
		f2Start := time.Now()
		f2Result, f2Err := florence2Client.OCRImage(ctx, img)
		f2Duration := time.Since(f2Start)

		if f2Err != nil {
			t.Logf("   ❌ Florence-2 error: %v", f2Err)
			florence2Results = append(florence2Results, BenchResult{
				Provider: "Florence-2",
				File:     filename,
				Duration: f2Duration,
				Success:  false,
				Error:    f2Err.Error(),
			})
		} else {
			t.Logf("   ✅ Florence-2: %d chars in %v (cost: $%.6f)",
				f2Result.Characters, f2Duration, f2Result.EstimatedCost)
			florence2Results = append(florence2Results, BenchResult{
				Provider:   "Florence-2",
				File:       filename,
				Duration:   f2Duration,
				Characters: f2Result.Characters,
				Cost:       f2Result.EstimatedCost,
				Success:    true,
			})
		}

		// Test AIMLAPI
		t.Log("   ☁️  Testing AIMLAPI...")
		aiStart := time.Now()
		aiResult, aiErr := aimlClient.OCRImage(ctx, img)
		aiDuration := time.Since(aiStart)

		if aiErr != nil {
			t.Logf("   ❌ AIMLAPI error: %v", aiErr)
			aimlResults = append(aimlResults, BenchResult{
				Provider: "AIMLAPI",
				File:     filename,
				Duration: aiDuration,
				Success:  false,
				Error:    aiErr.Error(),
			})
		} else {
			t.Logf("   ✅ AIMLAPI: %d chars in %v (cost: $%.6f)",
				aiResult.Characters, aiDuration, aiResult.EstimatedCost)
			aimlResults = append(aimlResults, BenchResult{
				Provider:   "AIMLAPI",
				File:       filename,
				Duration:   aiDuration,
				Characters: aiResult.Characters,
				Cost:       aiResult.EstimatedCost,
				Success:    true,
			})
		}

		t.Log("")
	}

	// Compute summary
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("📊 BENCHMARK RESULTS")
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("")

	// Florence-2 summary
	var f2TotalDuration time.Duration
	var f2TotalChars, f2Success int
	var f2TotalCost float64
	for _, r := range florence2Results {
		if r.Success {
			f2TotalDuration += r.Duration
			f2TotalChars += r.Characters
			f2TotalCost += r.Cost
			f2Success++
		}
	}

	// AIMLAPI summary
	var aiTotalDuration time.Duration
	var aiTotalChars, aiSuccess int
	var aiTotalCost float64
	for _, r := range aimlResults {
		if r.Success {
			aiTotalDuration += r.Duration
			aiTotalChars += r.Characters
			aiTotalCost += r.Cost
			aiSuccess++
		}
	}

	t.Log("🌸 FLORENCE-2:")
	t.Logf("   Success:      %d/%d (%.1f%%)", f2Success, len(florence2Results),
		float64(f2Success)/float64(max(len(florence2Results), 1))*100)
	t.Logf("   Total Time:   %v", f2TotalDuration)
	if f2Success > 0 {
		t.Logf("   Avg Time:     %v per page", f2TotalDuration/time.Duration(f2Success))
	}
	t.Logf("   Characters:   %d total", f2TotalChars)
	t.Logf("   Total Cost:   $%.6f", f2TotalCost)
	if f2Success > 0 {
		t.Logf("   Cost/Page:    $%.6f", f2TotalCost/float64(f2Success))
		t.Logf("   Cost/1K:      $%.2f", f2TotalCost/float64(f2Success)*1000)
	}

	t.Log("")
	t.Log("☁️  AIMLAPI:")
	t.Logf("   Success:      %d/%d (%.1f%%)", aiSuccess, len(aimlResults),
		float64(aiSuccess)/float64(max(len(aimlResults), 1))*100)
	t.Logf("   Total Time:   %v", aiTotalDuration)
	if aiSuccess > 0 {
		t.Logf("   Avg Time:     %v per page", aiTotalDuration/time.Duration(aiSuccess))
	}
	t.Logf("   Characters:   %d total", aiTotalChars)
	t.Logf("   Total Cost:   $%.6f", aiTotalCost)
	if aiSuccess > 0 {
		t.Logf("   Cost/Page:    $%.6f", aiTotalCost/float64(aiSuccess))
		t.Logf("   Cost/1K:      $%.2f", aiTotalCost/float64(aiSuccess)*1000)
	}

	// Comparison
	t.Log("")
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("📈 COMPARATIVE ANALYSIS")
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("")

	if f2Success > 0 && aiSuccess > 0 {
		f2Avg := f2TotalDuration / time.Duration(f2Success)
		aiAvg := aiTotalDuration / time.Duration(aiSuccess)

		speedup := float64(aiAvg) / float64(f2Avg)
		costRatio := aiTotalCost / f2TotalCost

		t.Log("⚡ SPEED COMPARISON:")
		t.Logf("   Florence-2:   %v per page", f2Avg)
		t.Logf("   AIMLAPI:      %v per page", aiAvg)
		t.Logf("   🚀 Florence-2 is %.1f× FASTER", speedup)

		if speedup >= 10 {
			t.Log("   ✅ Speed target MET (>10× faster)")
		} else {
			t.Logf("   ⚠️  Speed target not met (%.1f× < 10×)", speedup)
		}

		t.Log("")
		t.Log("💰 COST COMPARISON:")
		t.Logf("   Florence-2:   $%.6f per page", f2TotalCost/float64(f2Success))
		t.Logf("   AIMLAPI:      $%.6f per page", aiTotalCost/float64(aiSuccess))
		t.Logf("   💵 Florence-2 is %.1f× CHEAPER", costRatio)

		if costRatio >= 10 {
			t.Log("   ✅ Cost target MET (>10× cheaper)")
		} else {
			t.Logf("   ⚠️  Cost target not met (%.1f× < 10×)", costRatio)
		}

		// Calculate savings
		t.Log("")
		t.Log("💸 PROJECTED SAVINGS:")
		aimlCostPer1K := (aiTotalCost / float64(aiSuccess)) * 1000
		f2CostPer1K := (f2TotalCost / float64(f2Success)) * 1000
		savings1K := aimlCostPer1K - f2CostPer1K
		t.Logf("   Per 1,000 pages:    $%.2f savings (AIMLAPI: $%.2f, Florence-2: $%.2f)",
			savings1K, aimlCostPer1K, f2CostPer1K)
		t.Logf("   Per 10,000 pages:   $%.2f savings", savings1K*10)
		t.Logf("   Per 100,000 pages:  $%.2f savings", savings1K*100)

		// Character extraction comparison
		charDiff := float64(aiTotalChars-f2TotalChars) / float64(max(aiTotalChars, 1)) * 100
		t.Log("")
		t.Log("📝 TEXT EXTRACTION QUALITY:")
		t.Logf("   Florence-2:   %d chars", f2TotalChars)
		t.Logf("   AIMLAPI:      %d chars", aiTotalChars)
		if charDiff > 0 {
			t.Logf("   📊 AIMLAPI extracted %.1f%% more characters", charDiff)
		} else if charDiff < 0 {
			t.Logf("   📊 Florence-2 extracted %.1f%% more characters", -charDiff)
		} else {
			t.Log("   📊 Character counts are equivalent!")
		}
	}

	t.Log("")
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("🏆 RECOMMENDATION")
	t.Log("═══════════════════════════════════════════════════════════")

	if f2Success > 0 && aiSuccess > 0 {
		f2Avg := f2TotalDuration / time.Duration(f2Success)
		aiAvg := aiTotalDuration / time.Duration(aiSuccess)
		speedup := float64(aiAvg) / float64(f2Avg)
		costRatio := aiTotalCost / f2TotalCost

		if speedup >= 10 && costRatio >= 10 {
			t.Log("")
			t.Log("✅ Florence-2 MEETS ALL TARGETS!")
			t.Logf("   - Speed: %.1f× faster (target: 10×)", speedup)
			t.Logf("   - Cost: %.1f× cheaper (target: 10×)", costRatio)
			t.Log("")
			t.Log("💡 RECOMMENDED ACTION:")
			t.Log("   Switch primary OCR provider from AIMLAPI to Florence-2")
			t.Log("   Keep AIMLAPI as fallback for difficult documents")
		} else {
			t.Log("")
			t.Log("⚠️  Florence-2 performance varies from targets:")
			if speedup < 10 {
				t.Logf("   - Speed: %.1f× vs 10× target", speedup)
			}
			if costRatio < 10 {
				t.Logf("   - Cost: %.1f× vs 10× target", costRatio)
			}
			t.Log("")
			t.Log("💡 RECOMMENDED ACTION:")
			t.Log("   Run extended benchmark with more samples")
			t.Log("   Test on different document types")
		}
	}

	t.Log("")
	t.Log("═══════════════════════════════════════════════════════════")
}

// BenchmarkFlorence2OCR benchmarks Florence-2 throughput
func BenchmarkFlorence2OCR(b *testing.B) {
	client, err := NewFlorence2Client(nil)
	if err != nil {
		b.Skip("Florence-2 client not available")
	}

	// Create test image
	img := createFlorence2TestImage(500, 700)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.OCRImage(ctx, img)
		if err != nil {
			b.Logf("OCR error: %v", err)
		}
	}
}

// BenchmarkAIMLAPIOCR benchmarks AIMLAPI throughput
func BenchmarkAIMLAPIOCR(b *testing.B) {
	client, err := NewAIMLAPIOCRClient(nil)
	if err != nil {
		b.Skip("AIMLAPI client not available")
	}

	// Create test image
	img := createFlorence2TestImage(500, 700)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.OCRImage(ctx, img)
		if err != nil {
			b.Logf("OCR error: %v", err)
		}
	}
}

// BenchmarkFlorence2VsAIMLAPIParallel compares parallel throughput
func BenchmarkFlorence2VsAIMLAPIParallel(b *testing.B) {
	f2Client, _ := NewFlorence2Client(nil)
	aiClient, _ := NewAIMLAPIOCRClient(nil)
	ctx := context.Background()

	img := createFlorence2TestImage(500, 700)

	b.Run("Florence2-Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				f2Client.OCRImage(ctx, img)
			}
		})
	})

	b.Run("AIMLAPI-Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				aiClient.OCRImage(ctx, img)
			}
		})
	})
}

// createFlorence2TestImage creates a simple test image with text
func createFlorence2TestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with white background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Add some "text-like" patterns (simple rectangles)
	// This simulates a document layout
	black := color.Black

	// Title area
	for y := 20; y < 40; y++ {
		for x := 50; x < 450; x++ {
			img.Set(x, y, black)
		}
	}

	// Body text lines
	for line := 0; line < 10; line++ {
		y := 80 + line*40
		for yy := y; yy < y+15; yy++ {
			for x := 50; x < 400; x++ {
				if x%10 < 8 { // Create "word" gaps
					img.Set(x, yy, black)
				}
			}
		}
	}

	return img
}
