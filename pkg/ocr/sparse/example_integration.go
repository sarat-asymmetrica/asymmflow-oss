package sparse

import (
	"context"
	"fmt"
	"image"
	"log"
	"time"
)

// ExampleIntegration demonstrates how to integrate Sparse OCR with ACE Engine
func ExampleIntegration() {
	// Configuration for invoice processing (typical use case)
	config := &SparseOCRConfig{
		DNA:                NewPDFDNA(),
		RecurringThreshold: 3,     // Headers/footers seen 3× become recurring
		GridSize:           100,   // 100×100 pixel regions
		EnableLearning:     true,  // Auto-learn patterns
		UseFuzzyHash:       false, // Exact matching (invoices are scanned consistently)
		MinConfidence:      0.7,   // Only store high-quality OCR
		SkipEmptyRegions:   true,  // Skip white space
	}

	sparseOCR := NewSparseOCR(config)

	// Optional: Load existing DNA database from previous sessions
	// This gives instant speedup even on first document!
	if err := sparseOCR.LoadDNA("invoice_templates.json"); err != nil {
		log.Printf("No existing DNA database found (cold start): %v", err)
	} else {
		total, recurring, _, _ := sparseOCR.GetDNAStats()
		log.Printf("Loaded DNA database: %d total elements, %d recurring", total, recurring)
	}

	// Simulate processing 100 invoices
	ctx := context.Background()
	var totalSpeedup float64
	var totalTimeSaved time.Duration

	for i := 0; i < 100; i++ {
		// In real usage, load actual invoice image
		invoiceImage := loadInvoiceImage(i)

		// Process with DNA-aware OCR
		result, err := sparseOCR.ProcessWithDNA(ctx, invoiceImage, mockAIMLAPICall)
		if err != nil {
			log.Printf("Invoice %d failed: %v", i, err)
			continue
		}

		// Accumulate metrics
		totalSpeedup += result.SpeedupFactor
		totalTimeSaved += result.TimeSaved

		// Log progress every 10 invoices
		if (i+1)%10 == 0 {
			avgSpeedup := totalSpeedup / float64(i+1)
			fmt.Printf("Processed %d invoices - Avg speedup: %.2f×, Total time saved: %.1fs\n",
				i+1, avgSpeedup, totalTimeSaved.Seconds())
		}
	}

	// Final statistics
	avgSpeedup := totalSpeedup / 100.0
	fmt.Printf("\n=== Final Results ===\n")
	fmt.Printf("Average speedup: %.2f×\n", avgSpeedup)
	fmt.Printf("Total time saved: %.1f seconds\n", totalTimeSaved.Seconds())

	// DNA statistics
	total, recurring, hitRate, timeSaved := sparseOCR.GetDNAStats()
	fmt.Printf("\nDNA Database:\n")
	fmt.Printf("  Total elements: %d\n", total)
	fmt.Printf("  Recurring: %d\n", recurring)
	fmt.Printf("  Cache hit rate: %.1f%%\n", hitRate*100)
	fmt.Printf("  Cumulative time saved: %.1f seconds\n", float64(timeSaved)/1000.0)

	// Save DNA database for next session
	if err := sparseOCR.SaveDNA("invoice_templates.json"); err != nil {
		log.Printf("Failed to save DNA database: %v", err)
	} else {
		fmt.Printf("\nDNA database saved - next session will start HOT!\n")
	}

	// Prune low-frequency elements (optional cleanup)
	pruned := config.DNA.Prune(2) // Remove elements seen < 2 times
	fmt.Printf("Pruned %d low-frequency elements\n", pruned)
}

// mockAIMLAPICall simulates AIMLAPI OCR call
// In real integration, replace with actual AIMLAPI client
func mockAIMLAPICall(ctx context.Context, img image.Image) (string, float64, error) {
	// Simulate OCR latency (100ms per region)
	time.Sleep(100 * time.Millisecond)

	// In real code:
	// return aimlapi.OCR(ctx, img)

	return "Sample Invoice Text\nTotal: $1,234.56", 0.95, nil
}

// loadInvoiceImage simulates loading invoice image
// In real integration, load from file system or API
func loadInvoiceImage(index int) image.Image {
	// In real code:
	// return loadImageFromFile(fmt.Sprintf("invoices/invoice_%d.png", index))

	// For demo, return synthetic image
	return image.NewRGBA(image.Rect(0, 0, 800, 1000))
}

// ExampleMultiVendorProcessing shows how to handle multiple vendors
func ExampleMultiVendorProcessing() {
	// Create separate DNA databases per vendor (better accuracy)
	vendorDNAs := map[string]*PDFDNA{
		"vendor_a": NewPDFDNA(),
		"vendor_b": NewPDFDNA(),
		"vendor_c": NewPDFDNA(),
	}

	// Load existing templates
	for vendor, dna := range vendorDNAs {
		filename := fmt.Sprintf("templates_%s.json", vendor)
		if err := dna.Load(filename); err == nil {
			total, recurring, _ := dna.GetStats()
			log.Printf("Vendor %s: %d templates, %d recurring", vendor, total, recurring)
		}
	}

	// Create sparse OCR config (will swap DNA per vendor)
	config := DefaultSparseOCRConfig()
	sparseOCR := NewSparseOCR(config)

	// Process invoices by vendor
	invoices := []struct {
		vendor string
		image  image.Image
	}{
		{"vendor_a", loadInvoiceImage(0)},
		{"vendor_b", loadInvoiceImage(1)},
		{"vendor_a", loadInvoiceImage(2)}, // Same vendor → high recycling!
		// ...
	}

	ctx := context.Background()
	for _, inv := range invoices {
		// Swap DNA database for this vendor
		config.DNA = vendorDNAs[inv.vendor]

		// Process
		result, err := sparseOCR.ProcessWithDNA(ctx, inv.image, mockAIMLAPICall)
		if err != nil {
			log.Printf("Failed: %v", err)
			continue
		}

		fmt.Printf("Vendor %s: %.2f× speedup (%d recycled)\n",
			inv.vendor, result.SpeedupFactor, result.RecycledRegions)
	}

	// Save all vendor templates
	for vendor, dna := range vendorDNAs {
		dna.Save(fmt.Sprintf("templates_%s.json", vendor))
	}
}

// ExampleBatchProcessing shows optimized batch processing
func ExampleBatchProcessing() {
	config := DefaultSparseOCRConfig()
	config.RecurringThreshold = 2 // Lower threshold for batch (see patterns faster)
	sparseOCR := NewSparseOCR(config)

	// Load batch of documents
	documents := []image.Image{
		loadInvoiceImage(0),
		loadInvoiceImage(1),
		loadInvoiceImage(2),
		// ... 100 more
	}

	ctx := context.Background()

	// Batch process (automatic cross-document learning)
	results, err := sparseOCR.ProcessMultiplePages(ctx, documents, mockAIMLAPICall)
	if err != nil {
		log.Fatalf("Batch processing failed: %v", err)
	}

	// Analyze results
	fmt.Printf("Batch Processing Results:\n")
	fmt.Printf("%-6s %-8s %-10s %-10s %-10s\n", "Page", "Speedup", "Novel", "Recycled", "Skipped")
	fmt.Println("--------------------------------------------------------")

	var totalNovel, totalRecycled, totalSkipped int
	for i, result := range results {
		fmt.Printf("%-6d %-8.2fx %-10d %-10d %-10d\n",
			i+1, result.SpeedupFactor, result.NovelRegions,
			result.RecycledRegions, result.SkippedRegions)

		totalNovel += result.NovelRegions
		totalRecycled += result.RecycledRegions
		totalSkipped += result.SkippedRegions
	}

	fmt.Println("--------------------------------------------------------")
	fmt.Printf("TOTAL: Novel=%d, Recycled=%d, Skipped=%d\n",
		totalNovel, totalRecycled, totalSkipped)

	recyclingRate := float64(totalRecycled) / float64(totalNovel+totalRecycled)
	fmt.Printf("Recycling rate: %.1f%%\n", recyclingRate*100)
}

// ExampleFuzzyMatching shows perceptual hashing for scan variations
func ExampleFuzzyMatching() {
	config := DefaultSparseOCRConfig()
	config.UseFuzzyHash = true // Enable perceptual hashing
	sparseOCR := NewSparseOCR(config)

	// Process scanned documents (may have slight rotation, brightness variations)
	ctx := context.Background()

	// First scan (reference)
	result1, _ := sparseOCR.ProcessWithDNA(ctx, loadInvoiceImage(0), mockAIMLAPICall)
	fmt.Printf("Scan 1: %.2f× speedup (reference)\n", result1.SpeedupFactor)

	// Second scan (same document, but slightly different angle/brightness)
	result2, _ := sparseOCR.ProcessWithDNA(ctx, loadInvoiceImage(0), mockAIMLAPICall)
	fmt.Printf("Scan 2: %.2f× speedup (fuzzy matching!)\n", result2.SpeedupFactor)

	// Fuzzy hashing allows recycling even with minor variations
	if result2.RecycledRegions > 0 {
		fmt.Println("✓ Fuzzy matching successfully recycled regions despite variations!")
	}
}

// ExamplePersistentLearning shows cross-session learning
func ExamplePersistentLearning() {
	dnaFile := "persistent_templates.json"

	// Session 1: Process first batch
	fmt.Println("=== Session 1: Cold Start ===")
	config1 := DefaultSparseOCRConfig()
	sparseOCR1 := NewSparseOCR(config1)

	ctx := context.Background()
	for i := 0; i < 10; i++ {
		result, _ := sparseOCR1.ProcessWithDNA(ctx, loadInvoiceImage(i), mockAIMLAPICall)
		fmt.Printf("Doc %d: %.2f× speedup\n", i+1, result.SpeedupFactor)
	}

	// Save learned patterns
	sparseOCR1.SaveDNA(dnaFile)
	total, recurring, _, _ := sparseOCR1.GetDNAStats()
	fmt.Printf("Saved %d templates (%d recurring)\n\n", total, recurring)

	// Session 2: Process second batch (hot start!)
	fmt.Println("=== Session 2: Hot Start ===")
	config2 := DefaultSparseOCRConfig()
	sparseOCR2 := NewSparseOCR(config2)

	// Load previous learning
	sparseOCR2.LoadDNA(dnaFile)
	fmt.Println("Loaded existing templates")

	for i := 10; i < 20; i++ {
		result, _ := sparseOCR2.ProcessWithDNA(ctx, loadInvoiceImage(i), mockAIMLAPICall)
		fmt.Printf("Doc %d: %.2f× speedup (immediate benefit!)\n", i+1, result.SpeedupFactor)
	}
}
