// Test for batch offers OCR processing
package ocr

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

// TestBatchOffersProcessing tests the batch offer processor
func TestBatchOffersProcessing(t *testing.T) {
	// Skip if offers folder not available
	offersFolder := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`
	if _, err := os.Stat(offersFolder); os.IsNotExist(err) {
		t.Skip("Offers folder not available, skipping test")
	}

	// Create temp database
	tmpDB := filepath.Join(t.TempDir(), "test_batch_offers.db")
	db, err := gorm.Open(sqlite.Open(tmpDB), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Auto-migrate OCRDocument table
	if err := db.AutoMigrate(&OCRDocument{}); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	// Create OCR engine
	config := &EngineConfig{
		EnableGPU:             false, // Disable GPU for testing
		MaxWorkers:            2,
		DefaultLanguage:       LangEnglish,
		EnablePreprocessing:   false,
		EnableVedicValidation: false,
		FallbackToAIMLAPI:     false,
		LogLevel:              "info",
	}

	engine, err := NewACEEngine(config)
	if err != nil {
		t.Fatalf("Failed to create OCR engine: %v", err)
	}
	defer engine.Close()

	// Create progress channel
	progressChan := make(chan BatchOfferProgress, 10)
	defer close(progressChan)

	// Start goroutine to consume progress
	progressDone := make(chan bool)
	go func() {
		for progress := range progressChan {
			t.Logf("Progress: %s (%d/%d = %.1f%%) - Offer: %s, Customer: %s, Stage: %s",
				progress.CurrentFile,
				progress.FilesProcessed,
				progress.TotalFiles,
				progress.Percentage,
				progress.OfferNumber,
				progress.CustomerName,
				progress.Stage,
			)
			if progress.Error != nil {
				t.Logf("  Error: %v", progress.Error)
			}
		}
		progressDone <- true
	}()

	// Create batch request (process only first offer for testing)
	req := &BatchOfferRequest{
		OffersFolder:      offersFolder,
		OfferNumberFilter: "01", // Process only offer #01
		MaxConcurrency:    2,
		EnableGPU:         false,
		StopOnError:       false,
		DB:                db,
		ProgressChan:      progressChan,
	}

	ctx := context.Background()
	result, err := engine.ProcessOffersBatch(ctx, req)
	if err != nil {
		t.Fatalf("Batch processing failed: %v", err)
	}

	// Wait for progress goroutine to finish
	<-progressDone

	// Verify results
	t.Logf("=== BATCH OFFER RESULTS ===")
	t.Logf("Total Offers: %d", result.TotalOffers)
	t.Logf("Total Files: %d", result.TotalFiles)
	t.Logf("Processed: %d", result.ProcessedFiles)
	t.Logf("Failed: %d", result.FailedFiles)
	t.Logf("Average Confidence: %.2f%%", result.AverageConfidence*100)
	t.Logf("Total Time: %v", result.TotalTime)
	t.Logf("Total Cost: $%.4f", result.TotalCostUSD)
	t.Logf("GPU Usage: %.1f%%", result.GPUUsagePercent)

	t.Logf("\nDocuments by Type:")
	for docType, count := range result.DocumentsByType {
		t.Logf("  %s: %d", docType, count)
	}

	t.Logf("\nPer-Offer Results:")
	for offerNum, offerRes := range result.OfferResults {
		t.Logf("  Offer %s (%s): %d processed, %d failed",
			offerNum,
			offerRes.CustomerName,
			offerRes.FilesProcessed,
			offerRes.FilesFailed,
		)
	}

	// Verify database persistence
	var dbCount int64
	db.Model(&OCRDocument{}).Count(&dbCount)
	t.Logf("\nOCR Documents in DB: %d", dbCount)

	if dbCount != int64(result.ProcessedFiles) {
		t.Errorf("Database count mismatch: got %d, want %d", dbCount, result.ProcessedFiles)
	}

	// Verify at least some files were processed
	if result.ProcessedFiles == 0 {
		t.Error("Expected some files to be processed, got 0")
	}
}

// TestOfferFolderScanning tests the folder scanning logic
func TestOfferFolderScanning(t *testing.T) {
	offersFolder := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`
	if _, err := os.Stat(offersFolder); os.IsNotExist(err) {
		t.Skip("Offers folder not available, skipping test")
	}

	// Create minimal engine for testing
	config := &EngineConfig{
		EnableGPU:  false,
		MaxWorkers: 1,
		LogLevel:   "error",
	}

	engine, err := NewACEEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Scan folders
	offers, err := engine.scanOfferFolders(offersFolder, "")
	if err != nil {
		t.Fatalf("Failed to scan folders: %v", err)
	}

	t.Logf("Found %d offer folders:", len(offers))
	for _, offer := range offers {
		t.Logf("  %s: Offer #%s, Customer: %s",
			offer.FullName,
			offer.OfferNumber,
			offer.CustomerName,
		)
	}

	if len(offers) == 0 {
		t.Error("Expected to find offer folders, got 0")
	}

	// Test filtering
	filteredOffers, err := engine.scanOfferFolders(offersFolder, "01")
	if err != nil {
		t.Fatalf("Failed to scan with filter: %v", err)
	}

	if len(filteredOffers) != 1 {
		t.Errorf("Expected 1 offer with filter '01', got %d", len(filteredOffers))
	}

	if len(filteredOffers) > 0 && filteredOffers[0].OfferNumber != "01" {
		t.Errorf("Expected offer number '01', got '%s'", filteredOffers[0].OfferNumber)
	}
}

// BenchmarkBatchOffersProcessing benchmarks the batch processor
func BenchmarkBatchOffersProcessing(b *testing.B) {
	offersFolder := `C:\Projects\ACE_Engine\data/ssot\Offers No 1-50 (2025)`
	if _, err := os.Stat(offersFolder); os.IsNotExist(err) {
		b.Skip("Offers folder not available, skipping benchmark")
	}

	// Create temp database
	tmpDB := filepath.Join(b.TempDir(), "bench_batch_offers.db")
	db, err := gorm.Open(sqlite.Open(tmpDB), &gorm.Config{})
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}

	if err := db.AutoMigrate(&OCRDocument{}); err != nil {
		b.Fatalf("Failed to migrate schema: %v", err)
	}

	// Create engine
	config := &EngineConfig{
		EnableGPU:             false,
		MaxWorkers:            4,
		DefaultLanguage:       LangEnglish,
		EnablePreprocessing:   false,
		EnableVedicValidation: false,
		FallbackToAIMLAPI:     false,
		LogLevel:              "error",
	}

	engine, err := NewACEEngine(config)
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &BatchOfferRequest{
			OffersFolder:      offersFolder,
			OfferNumberFilter: "01", // Benchmark single offer
			MaxConcurrency:    4,
			EnableGPU:         false,
			StopOnError:       false,
			DB:                db,
		}

		ctx := context.Background()
		_, err := engine.ProcessOffersBatch(ctx, req)
		if err != nil {
			b.Fatalf("Batch processing failed: %v", err)
		}
	}
}
