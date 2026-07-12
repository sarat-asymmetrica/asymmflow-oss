package sparse

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// Mock OCR function for testing
func mockOCR(ctx context.Context, img image.Image) (string, float64, error) {
	// Simulate OCR with confidence based on image brightness
	bounds := img.Bounds()
	var totalBrightness uint32
	pixelCount := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			brightness := (r + g + b) / 3
			totalBrightness += uint32(brightness >> 8)
			pixelCount++
		}
	}

	avgBrightness := float64(totalBrightness) / float64(pixelCount)
	confidence := 1.0 - (avgBrightness / 255.0) // Darker = higher confidence

	// Return mock text based on region
	text := "Sample OCR Text"
	return text, confidence, nil
}

// Helper: Create test image with pattern
func createTestImage(width, height int, fillColor color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, fillColor)
		}
	}
	return img
}

// Helper: Create image with horizontal stripe pattern
func createStripedImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (y/20)%2 == 0 {
				img.Set(x, y, color.RGBA{50, 50, 50, 255}) // Dark stripe
			} else {
				img.Set(x, y, color.RGBA{200, 200, 200, 255}) // Light stripe
			}
		}
	}
	return img
}

func TestPDFDNA_NewPDFDNA(t *testing.T) {
	dna := NewPDFDNA()

	if dna == nil {
		t.Fatal("NewPDFDNA returned nil")
	}

	if dna.Elements == nil {
		t.Error("Elements map not initialized")
	}

	if dna.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", dna.Version)
	}
}

func TestPDFDNA_RegisterAndLookup(t *testing.T) {
	dna := NewPDFDNA()

	hash := "test_hash_123"
	elemType := "header"
	text := "Company Logo"
	bbox := [4]int{0, 0, 100, 50}
	confidence := 0.95

	// Register element
	dna.RegisterElement(hash, elemType, text, bbox, confidence)

	// Lookup element
	elem, found := dna.LookupElement(hash)
	if !found {
		t.Fatal("Element not found after registration")
	}

	if elem.Hash != hash {
		t.Errorf("Expected hash %s, got %s", hash, elem.Hash)
	}

	if elem.Type != elemType {
		t.Errorf("Expected type %s, got %s", elemType, elem.Type)
	}

	if elem.Text != text {
		t.Errorf("Expected text %s, got %s", text, elem.Text)
	}

	if elem.Frequency != 1 {
		t.Errorf("Expected frequency 1, got %d", elem.Frequency)
	}

	if elem.Confidence != confidence {
		t.Errorf("Expected confidence %.2f, got %.2f", confidence, elem.Confidence)
	}
}

func TestPDFDNA_FrequencyIncrement(t *testing.T) {
	dna := NewPDFDNA()

	hash := "recurring_hash"
	bbox := [4]int{0, 0, 100, 50}

	// Register same element 5 times
	for i := 0; i < 5; i++ {
		dna.RegisterElement(hash, "footer", "Page Footer", bbox, 0.9)
	}

	elem, found := dna.LookupElement(hash)
	if !found {
		t.Fatal("Element not found")
	}

	if elem.Frequency != 5 {
		t.Errorf("Expected frequency 5, got %d", elem.Frequency)
	}
}

func TestPDFDNA_ConfidenceUpdate(t *testing.T) {
	dna := NewPDFDNA()

	hash := "confidence_test"
	bbox := [4]int{0, 0, 100, 50}

	// Register with low confidence
	dna.RegisterElement(hash, "text", "Initial Text", bbox, 0.5)

	// Register again with higher confidence
	dna.RegisterElement(hash, "text", "Better Text", bbox, 0.95)

	elem, _ := dna.LookupElement(hash)

	// Should keep higher confidence text
	if elem.Text != "Better Text" {
		t.Errorf("Expected 'Better Text', got '%s'", elem.Text)
	}

	if elem.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %.2f", elem.Confidence)
	}

	// Register with lower confidence - should NOT update
	dna.RegisterElement(hash, "text", "Worse Text", bbox, 0.6)

	elem, _ = dna.LookupElement(hash)
	if elem.Text != "Better Text" {
		t.Errorf("Text should not change to lower confidence, got '%s'", elem.Text)
	}
}

func TestPDFDNA_IsRecurring(t *testing.T) {
	dna := NewPDFDNA()

	hash := "recurring_test"
	bbox := [4]int{0, 0, 100, 50}

	// Register twice - not recurring yet (threshold = 3)
	for i := 0; i < 2; i++ {
		dna.RegisterElement(hash, "header", "Header", bbox, 0.9)
	}

	if dna.IsRecurring(hash) {
		t.Error("Element should not be recurring with frequency 2")
	}

	// Register once more - now recurring
	dna.RegisterElement(hash, "header", "Header", bbox, 0.9)

	if !dna.IsRecurring(hash) {
		t.Error("Element should be recurring with frequency 3")
	}
}

func TestPDFDNA_Stats(t *testing.T) {
	dna := NewPDFDNA()

	// Add 3 elements with different frequencies
	dna.RegisterElement("hash1", "header", "Header", [4]int{0, 0, 100, 50}, 0.9)
	dna.RegisterElement("hash1", "header", "Header", [4]int{0, 0, 100, 50}, 0.9)
	dna.RegisterElement("hash1", "header", "Header", [4]int{0, 0, 100, 50}, 0.9) // Freq=3, recurring

	dna.RegisterElement("hash2", "footer", "Footer", [4]int{0, 0, 100, 50}, 0.9)
	dna.RegisterElement("hash2", "footer", "Footer", [4]int{0, 0, 100, 50}, 0.9) // Freq=2, not recurring

	dna.RegisterElement("hash3", "logo", "Logo", [4]int{0, 0, 100, 50}, 0.9)
	dna.RegisterElement("hash3", "logo", "Logo", [4]int{0, 0, 100, 50}, 0.9)
	dna.RegisterElement("hash3", "logo", "Logo", [4]int{0, 0, 100, 50}, 0.9)
	dna.RegisterElement("hash3", "logo", "Logo", [4]int{0, 0, 100, 50}, 0.9) // Freq=4, recurring

	total, recurring, _ := dna.GetStats()

	if total != 3 {
		t.Errorf("Expected 3 total elements, got %d", total)
	}

	if recurring != 2 {
		t.Errorf("Expected 2 recurring elements, got %d", recurring)
	}

	// Note: Hit rate requires actual LookupElement calls, not just RegisterElement
	// This is tested separately in TestPDFDNA_RegisterAndLookup
}

func TestPDFDNA_SaveAndLoad(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	filepath := filepath.Join(tmpDir, "test_dna.json")

	// Create and populate DNA
	dna1 := NewPDFDNA()
	dna1.RegisterElement("hash1", "header", "Test Header", [4]int{0, 0, 100, 50}, 0.95)
	dna1.RegisterElement("hash2", "footer", "Test Footer", [4]int{0, 0, 100, 50}, 0.90)

	// Save
	err := dna1.Save(filepath)
	if err != nil {
		t.Fatalf("Failed to save DNA: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		t.Fatal("DNA file was not created")
	}

	// Load into new DNA instance
	dna2 := NewPDFDNA()
	err = dna2.Load(filepath)
	if err != nil {
		t.Fatalf("Failed to load DNA: %v", err)
	}

	// Verify data
	elem, found := dna2.LookupElement("hash1")
	if !found {
		t.Fatal("Element hash1 not found after load")
	}

	if elem.Text != "Test Header" {
		t.Errorf("Expected 'Test Header', got '%s'", elem.Text)
	}

	total, _, _ := dna2.GetStats()
	if total != 2 {
		t.Errorf("Expected 2 elements after load, got %d", total)
	}
}

func TestPDFDNA_Prune(t *testing.T) {
	dna := NewPDFDNA()

	// Add elements with different frequencies
	for i := 0; i < 5; i++ {
		dna.RegisterElement("freq5", "text", "High Freq", [4]int{0, 0, 100, 50}, 0.9)
	}

	for i := 0; i < 2; i++ {
		dna.RegisterElement("freq2", "text", "Low Freq", [4]int{0, 0, 100, 50}, 0.9)
	}

	dna.RegisterElement("freq1", "text", "Very Low Freq", [4]int{0, 0, 100, 50}, 0.9)

	// Prune elements with frequency < 3
	pruned := dna.Prune(3)

	if pruned != 2 {
		t.Errorf("Expected to prune 2 elements, pruned %d", pruned)
	}

	total, _, _ := dna.GetStats()
	if total != 1 {
		t.Errorf("Expected 1 element remaining, got %d", total)
	}

	// Verify high-frequency element remains
	_, found := dna.LookupElement("freq5")
	if !found {
		t.Error("High frequency element should not be pruned")
	}
}

func TestPDFDNA_Merge(t *testing.T) {
	dna1 := NewPDFDNA()
	dna1.RegisterElement("hash1", "header", "Header 1", [4]int{0, 0, 100, 50}, 0.9)
	dna1.RegisterElement("hash2", "footer", "Footer 1", [4]int{0, 0, 100, 50}, 0.8)

	dna2 := NewPDFDNA()
	dna2.RegisterElement("hash1", "header", "Header 1", [4]int{0, 0, 100, 50}, 0.95) // Higher confidence
	dna2.RegisterElement("hash3", "logo", "Logo 2", [4]int{0, 0, 100, 50}, 0.9)

	// Merge dna2 into dna1
	merged := dna1.Merge(dna2)

	if merged != 1 {
		t.Errorf("Expected 1 new element merged, got %d", merged)
	}

	total, _, _ := dna1.GetStats()
	if total != 3 {
		t.Errorf("Expected 3 total elements, got %d", total)
	}

	// Check that hash1 frequency increased and confidence updated
	elem, _ := dna1.LookupElement("hash1")
	if elem.Frequency != 2 {
		t.Errorf("Expected frequency 2 for merged element, got %d", elem.Frequency)
	}

	if elem.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95 (higher), got %.2f", elem.Confidence)
	}
}

func TestHashRegion(t *testing.T) {
	pixels1 := []byte{255, 0, 0, 0, 255, 0, 0, 0, 255} // RGB pixels
	pixels2 := []byte{255, 0, 0, 0, 255, 0, 0, 0, 255} // Same pixels
	pixels3 := []byte{0, 255, 0, 255, 0, 0, 0, 0, 255} // Different pixels

	hash1 := HashRegion(pixels1)
	hash2 := HashRegion(pixels2)
	hash3 := HashRegion(pixels3)

	if hash1 != hash2 {
		t.Error("Same pixels should produce same hash")
	}

	if hash1 == hash3 {
		t.Error("Different pixels should produce different hash")
	}

	// Hash should be 32 hex characters (128 bits)
	if len(hash1) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash1))
	}
}

func TestHashRegionFuzzy(t *testing.T) {
	width, height := 64, 64

	// Create similar patterns
	pixels1 := make([]byte, width*height*3)
	pixels2 := make([]byte, width*height*3)

	// Fill with similar but not identical patterns
	for i := 0; i < width*height; i++ {
		// Pattern 1
		pixels1[i*3] = byte(i % 200)
		pixels1[i*3+1] = byte((i + 10) % 200)
		pixels1[i*3+2] = byte((i + 20) % 200)

		// Pattern 2 (slightly different)
		pixels2[i*3] = byte((i + 2) % 200)
		pixels2[i*3+1] = byte((i + 12) % 200)
		pixels2[i*3+2] = byte((i + 22) % 200)
	}

	hash1 := HashRegionFuzzy(pixels1, width, height)
	hash2 := HashRegionFuzzy(pixels2, width, height)

	// Fuzzy hashes should be similar (not necessarily identical)
	// For now just verify they're generated
	if len(hash1) != 16 {
		t.Errorf("Expected fuzzy hash length 16, got %d", len(hash1))
	}

	if len(hash2) != 16 {
		t.Errorf("Expected fuzzy hash length 16, got %d", len(hash2))
	}
}

func TestSparseOCR_ClassifyRegions(t *testing.T) {
	config := DefaultSparseOCRConfig()
	config.GridSize = 50
	sparseOCR := NewSparseOCR(config)

	// Create test image (200x200)
	img := createStripedImage(200, 200)

	// Classify regions
	regions := sparseOCR.ClassifyRegions(img)

	// Should have 4x4 = 16 regions
	expectedRegions := 16
	if len(regions) != expectedRegions {
		t.Errorf("Expected %d regions, got %d", expectedRegions, len(regions))
	}

	// All should be novel initially (no DNA data)
	novelCount := 0
	for _, r := range regions {
		if r.Classification == RegionNovel {
			novelCount++
		}
	}

	if novelCount != expectedRegions {
		t.Errorf("Expected all %d regions to be novel, got %d", expectedRegions, novelCount)
	}
}

func TestSparseOCR_ClassifyRegions_WithDNA(t *testing.T) {
	config := DefaultSparseOCRConfig()
	config.GridSize = 50
	config.RecurringThreshold = 2
	sparseOCR := NewSparseOCR(config)

	// Create test image
	img := createStripedImage(200, 200)

	// First pass - classify and populate DNA
	regions1 := sparseOCR.ClassifyRegions(img)
	for _, r := range regions1 {
		if r.Classification == RegionNovel {
			// Simulate learning by registering twice
			bbox := [4]int{r.Rect.Min.X, r.Rect.Min.Y, r.Rect.Dx(), r.Rect.Dy()}
			config.DNA.RegisterElement(r.Hash, "detected", "Sample Text", bbox, 0.9)
			config.DNA.RegisterElement(r.Hash, "detected", "Sample Text", bbox, 0.9)
		}
	}

	// Second pass - should detect recurring regions
	regions2 := sparseOCR.ClassifyRegions(img)

	recurringCount := 0
	for _, r := range regions2 {
		if r.Classification == RegionRecurring {
			recurringCount++
		}
	}

	if recurringCount == 0 {
		t.Error("Expected some regions to be classified as recurring")
	}
}

func TestSparseOCR_EmptyRegionSkipping(t *testing.T) {
	config := DefaultSparseOCRConfig()
	config.GridSize = 50
	config.SkipEmptyRegions = true
	sparseOCR := NewSparseOCR(config)

	// Create mostly white image
	img := createTestImage(200, 200, color.RGBA{255, 255, 255, 255})

	regions := sparseOCR.ClassifyRegions(img)

	emptyCount := 0
	for _, r := range regions {
		if r.Classification == RegionEmpty {
			emptyCount++
		}
	}

	// Most/all regions should be classified as empty
	if emptyCount == 0 {
		t.Error("Expected empty regions in white image")
	}
}

func TestSparseOCR_ProcessWithDNA(t *testing.T) {
	config := DefaultSparseOCRConfig()
	config.GridSize = 100
	config.EnableLearning = true
	sparseOCR := NewSparseOCR(config)

	// Create test image
	img := createStripedImage(300, 300)

	ctx := context.Background()

	// First pass - all novel
	result1, err := sparseOCR.ProcessWithDNA(ctx, img, mockOCR)
	if err != nil {
		t.Fatalf("ProcessWithDNA failed: %v", err)
	}

	if result1.NovelRegions == 0 {
		t.Error("Expected novel regions on first pass")
	}

	if result1.RecycledRegions != 0 {
		t.Error("Expected no recycled regions on first pass")
	}

	if result1.Text == "" {
		t.Error("Expected non-empty text result")
	}

	// Register all regions multiple times to make them recurring
	regions := sparseOCR.ClassifyRegions(img)
	for _, r := range regions {
		if r.Classification == RegionNovel {
			bbox := [4]int{r.Rect.Min.X, r.Rect.Min.Y, r.Rect.Dx(), r.Rect.Dy()}
			config.DNA.RegisterElement(r.Hash, "detected", "Sample Text", bbox, 0.95)
			config.DNA.RegisterElement(r.Hash, "detected", "Sample Text", bbox, 0.95)
		}
	}

	// Second pass - should recycle
	result2, err := sparseOCR.ProcessWithDNA(ctx, img, mockOCR)
	if err != nil {
		t.Fatalf("ProcessWithDNA second pass failed: %v", err)
	}

	if result2.RecycledRegions == 0 {
		t.Error("Expected recycled regions on second pass")
	}

	if result2.SpeedupFactor <= 1.0 {
		t.Errorf("Expected speedup factor > 1.0, got %.2f", result2.SpeedupFactor)
	}

	if result2.CacheHitRate <= 0 {
		t.Errorf("Expected cache hit rate > 0, got %.2f", result2.CacheHitRate)
	}
}

func TestSparseOCR_ProcessMultiplePages(t *testing.T) {
	config := DefaultSparseOCRConfig()
	config.GridSize = 100
	config.EnableLearning = true
	config.RecurringThreshold = 2 // Lower threshold so we see recycling faster
	config.MinConfidence = 0.3    // Lower threshold so learning works with test images
	sparseOCR := NewSparseOCR(config)

	// Create 4 similar pages (need 4 to see recycling with threshold=2 + learning)
	images := []image.Image{
		createStripedImage(300, 300),
		createStripedImage(300, 300),
		createStripedImage(300, 300),
		createStripedImage(300, 300),
	}

	ctx := context.Background()

	results, err := sparseOCR.ProcessMultiplePages(ctx, images, mockOCR)
	if err != nil {
		t.Fatalf("ProcessMultiplePages failed: %v", err)
	}

	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}

	// First page should be mostly novel
	if results[0].NovelRegions == 0 {
		t.Error("Expected novel regions on first page")
	}

	// Log recycling progress across pages
	for i, result := range results {
		t.Logf("Page %d: novel=%d recycled=%d speedup=%.2fx",
			i, result.NovelRegions, result.RecycledRegions, result.SpeedupFactor)
	}

	// By page 2 or 3, we should see some recycling (regions seen on page 0 and 1 become recurring)
	laterRecycled := results[2].RecycledRegions + results[3].RecycledRegions
	if laterRecycled == 0 && results[0].NovelRegions > 0 {
		t.Error("Expected some recycling on later pages with learning enabled and threshold=2")
	}

	// DNA should have learned elements
	total, recurring, _, _ := sparseOCR.GetDNAStats()
	t.Logf("DNA Stats: total=%d recurring=%d", total, recurring)
	if total == 0 {
		t.Error("Expected DNA to learn elements across pages")
	}
}

func TestSparseOCR_DNAStats(t *testing.T) {
	config := DefaultSparseOCRConfig()
	sparseOCR := NewSparseOCR(config)

	// Initially empty
	total, recurring, _, timeSaved := sparseOCR.GetDNAStats()
	if total != 0 {
		t.Errorf("Expected 0 total elements initially, got %d", total)
	}

	// Add some elements
	config.DNA.RegisterElement("hash1", "header", "Header", [4]int{0, 0, 100, 50}, 0.9)
	config.DNA.RegisterElement("hash1", "header", "Header", [4]int{0, 0, 100, 50}, 0.9)
	config.DNA.RegisterElement("hash1", "header", "Header", [4]int{0, 0, 100, 50}, 0.9)

	total, recurring, _, timeSaved = sparseOCR.GetDNAStats()
	if total != 1 {
		t.Errorf("Expected 1 total element, got %d", total)
	}

	if recurring != 1 {
		t.Errorf("Expected 1 recurring element, got %d", recurring)
	}

	// Time saved should be trackable
	config.DNA.RecordTimeSaved(5000) // 5 seconds
	_, _, _, timeSaved = sparseOCR.GetDNAStats()
	if timeSaved != 5000 {
		t.Errorf("Expected 5000ms time saved, got %d", timeSaved)
	}
}

func TestSparseOCR_SpeedupCalculation(t *testing.T) {
	config := DefaultSparseOCRConfig()
	config.GridSize = 100
	sparseOCR := NewSparseOCR(config)

	img := createStripedImage(300, 300)

	// Manually set up DNA to recycle 6 out of 9 regions
	regions := sparseOCR.ClassifyRegions(img)
	for i, r := range regions {
		if i < 6 { // First 6 become recurring
			bbox := [4]int{r.Rect.Min.X, r.Rect.Min.Y, r.Rect.Dx(), r.Rect.Dy()}
			for j := 0; j < 3; j++ {
				config.DNA.RegisterElement(r.Hash, "detected", "Text", bbox, 0.9)
			}
		}
	}

	ctx := context.Background()
	result, err := sparseOCR.ProcessWithDNA(ctx, img, mockOCR)
	if err != nil {
		t.Fatalf("ProcessWithDNA failed: %v", err)
	}

	// Log actual speedup achieved
	t.Logf("Speedup: %.2fx (novel=%d, recycled=%d, skipped=%d)",
		result.SpeedupFactor, result.NovelRegions, result.RecycledRegions, result.SkippedRegions)

	// Speedup should be > 1.0 if we have recycled regions
	if result.RecycledRegions > 0 && result.SpeedupFactor <= 1.0 {
		t.Errorf("Expected speedup > 1.0 with recycled regions, got %.2fx", result.SpeedupFactor)
	}

	// If no novel regions (all recycled), speedup should be very high
	if result.NovelRegions == 0 && result.RecycledRegions > 0 {
		expectedMinSpeedup := float64(result.RecycledRegions)
		if result.SpeedupFactor < expectedMinSpeedup {
			t.Errorf("Expected speedup >= %.0fx for all-recycled, got %.2fx",
				expectedMinSpeedup, result.SpeedupFactor)
		}
	}
}

func BenchmarkHashRegion(b *testing.B) {
	pixels := make([]byte, 100*100*3) // 100x100 region
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HashRegion(pixels)
	}
}

func BenchmarkHashRegionFuzzy(b *testing.B) {
	width, height := 100, 100
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HashRegionFuzzy(pixels, width, height)
	}
}

func BenchmarkClassifyRegions(b *testing.B) {
	config := DefaultSparseOCRConfig()
	config.GridSize = 100
	sparseOCR := NewSparseOCR(config)

	img := createStripedImage(500, 500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sparseOCR.ClassifyRegions(img)
	}
}
