package ksum

import (
	"fmt"
	"image"
)

// Example: Pre-filter documents for table regions
func ExamplePrefilter() {
	// Suppose we have a document page image
	var pageImage image.Image // = loadPageImage(...)

	// Create detector
	detector := NewTableDetector(nil)

	// Detect tables
	tables := detector.DetectTables(pageImage)

	if len(tables) > 0 {
		fmt.Printf("Found %d tables on this page\n", len(tables))

		for i, table := range tables {
			fmt.Printf("  Table %d: %dx%d cells at %v (confidence: %.2f)\n",
				i, table.Rows, table.Cols, table.Bounds, table.Confidence)

			// Extract table region for specialized OCR
			// tableRegion := extractSubImage(pageImage, table.Bounds)
			// result := processTableOCR(tableRegion, table.Rows, table.Cols)
		}
	} else {
		fmt.Println("No tables detected - use standard OCR")
		// result := processStandardOCR(pageImage)
	}
}

// Example: Adaptive OCR strategy based on table confidence
func ExampleAdaptiveStrategy() {
	var pageImage image.Image // = loadPageImage(...)

	detector := NewTableDetector(nil)
	tables := detector.DetectTables(pageImage)

	// Determine OCR strategy
	maxConfidence := 0.0
	for _, t := range tables {
		if t.Confidence > maxConfidence {
			maxConfidence = t.Confidence
		}
	}

	if maxConfidence > 0.7 {
		fmt.Println("High-confidence table detected - using grid-optimized OCR")
		// config.TableOptimization = true
	} else if maxConfidence > 0.4 {
		fmt.Println("Medium-confidence table - using hybrid approach")
		// config.HybridMode = true
	} else {
		fmt.Println("No clear tables - using standard text OCR")
		// config.StandardMode = true
	}
}

// Example: Template matching for recurring table structures
func ExampleTemplateMatching() {
	var templateImage image.Image  // = loadTemplateImage(...)
	var candidateImage image.Image // = loadCandidateImage(...)

	config := DefaultKsumConfig()

	// Compute template fingerprint once
	templateFP := ComputeFingerprint(templateImage, config)

	// Compare against candidate
	candidateFP := ComputeFingerprint(candidateImage, config)
	similarity := templateFP.Similarity(candidateFP)

	if similarity > 0.8 {
		fmt.Printf("High similarity (%.2f) - likely same table structure\n", similarity)
		// Use template-based extraction
	} else {
		fmt.Printf("Low similarity (%.2f) - different structure\n", similarity)
		// Use general table extraction
	}
}

// Example: Quick mode for large batch processing
func ExampleQuickMode() {
	// For processing 1000s of pages, use quick mode
	detector := NewTableDetector(nil)

	// Process many pages
	pageImages := []image.Image{} // = loadManyPages(...)

	for i, img := range pageImages {
		// Quick detection (single scale, no overlap)
		tables := detector.DetectTablesQuick(img)

		if len(tables) > 0 {
			fmt.Printf("Page %d: %d tables detected\n", i, len(tables))
		}
	}
}

// Example: Region-aware processing
func ExampleRegionAware() {
	var pageImage image.Image // = loadPageImage(...)

	detector := NewTableDetector(nil)
	tables := detector.DetectTables(pageImage)

	// Separate processing for tables and text
	for _, table := range tables {
		fmt.Printf("Processing table region at %v with specialized OCR\n", table.Bounds)
		// tableImg := extractSubImage(pageImage, table.Bounds)
		// tableResult := processTableOCR(tableImg, table.Rows, table.Cols)
	}

	// Process non-table regions
	// textRegions := maskOutTables(pageImage, tables)
	// for _, region := range textRegions {
	//     textResult := processTextOCR(region)
	// }
}

// Example: Custom configuration for financial documents
func ExampleFinancialDocuments() {
	config := &KsumConfig{
		K:               10,   // Detect 10 strongest lines
		LineThreshold:   20.0, // Clear printed lines
		OrthogThreshold: 0.4,  // Accept some variation
		MinGridCells:    6,    // Financial tables typically 3x2 or larger
	}

	detector := NewTableDetector(config)

	// Process invoice/statement
	var invoiceImage image.Image // = loadInvoice(...)
	tables := detector.DetectTables(invoiceImage)

	for i, table := range tables {
		if table.Rows >= 3 && table.Cols >= 2 {
			fmt.Printf("Table %d: Likely financial table (itemized list)\n", i)
			// Extract and process
		}
	}
}

// Example: Deduplication across pages
func ExampleDeduplication() {
	detector := NewTableDetector(nil)

	// Track unique table structures
	seenHashes := make(map[[32]byte]bool)

	pageImages := []image.Image{} // = loadPages(...)

	for pageNum, img := range pageImages {
		tables := detector.DetectTables(img)

		for _, table := range tables {
			fp := table.Fingerprint
			hash := fp.Hash

			if seenHashes[hash] {
				fmt.Printf("Page %d: Duplicate table structure (seen before)\n", pageNum)
				// Use cached extraction strategy
			} else {
				fmt.Printf("Page %d: New table structure\n", pageNum)
				seenHashes[hash] = true
				// Analyze structure and cache strategy
			}
		}
	}
}

// Example: Performance comparison
func ExamplePerformanceComparison() {
	var pageImage image.Image // = loadPageImage(...)

	// Method 1: K-sum (fast pre-filter)
	detector := NewTableDetector(nil)
	tables := detector.DetectTablesQuick(pageImage) // ~90ms

	if len(tables) > 0 {
		fmt.Printf("K-sum detected %d tables in ~90ms\n", len(tables))
		// Only run expensive OCR on table regions
		// Total time: 90ms + (table_area_OCR)
	} else {
		fmt.Println("No tables - skip expensive table OCR entirely")
		// Total time: 90ms (saved all table OCR time!)
	}

	// Method 2: Traditional (always run full OCR)
	// Total time: full_page_OCR (~500ms) + post-processing
}
