package sparse

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"time"
)

// SparseOCRConfig configures sparse OCR
type SparseOCRConfig struct {
	DNA                *PDFDNA
	RecurringThreshold int     // Min frequency to skip OCR (default: 3)
	GridSize           int     // Region grid size in pixels (default: 100)
	EnableLearning     bool    // Auto-learn new patterns (default: true)
	UseFuzzyHash       bool    // Use perceptual hashing for minor variations (default: false)
	MinConfidence      float64 // Minimum OCR confidence to store in DNA (default: 0.7)
	SkipEmptyRegions   bool    // Skip regions with < 5% non-white pixels (default: true)
}

// DefaultSparseOCRConfig returns sensible defaults
func DefaultSparseOCRConfig() *SparseOCRConfig {
	return &SparseOCRConfig{
		DNA:                NewPDFDNA(),
		RecurringThreshold: 3,
		GridSize:           100,
		EnableLearning:     true,
		UseFuzzyHash:       false,
		MinConfidence:      0.7,
		SkipEmptyRegions:   true,
	}
}

// SparseOCRResult contains sparse OCR results
type SparseOCRResult struct {
	Text            string
	NovelRegions    int           // Regions that needed OCR
	RecycledRegions int           // Regions retrieved from DNA
	SkippedRegions  int           // Empty regions skipped
	TimeSaved       time.Duration // Estimated time saved
	SpeedupFactor   float64       // Total regions / novel regions
	CacheHitRate    float64       // Recycled / (recycled + novel)
	DNASize         int           // Current DNA database size
}

// RegionClassification categorizes a region
type RegionClassification string

const (
	RegionNovel     RegionClassification = "novel"     // Needs OCR
	RegionRecurring RegionClassification = "recurring" // In DNA, skip OCR
	RegionEmpty     RegionClassification = "empty"     // Blank, skip entirely
)

// ClassifiedRegion represents a classified image region
type ClassifiedRegion struct {
	Rect           image.Rectangle
	Classification RegionClassification
	Hash           string
	Text           string  // For recurring regions from DNA
	Confidence     float64 // For recurring regions from DNA
}

// SparseOCR performs DNA-aware OCR
type SparseOCR struct {
	config *SparseOCRConfig
}

// NewSparseOCR creates a new sparse OCR processor
func NewSparseOCR(config *SparseOCRConfig) *SparseOCR {
	if config == nil {
		config = DefaultSparseOCRConfig()
	}
	return &SparseOCR{config: config}
}

// ClassifyRegions splits image into grid and classifies each region
func (s *SparseOCR) ClassifyRegions(img image.Image) []ClassifiedRegion {
	bounds := img.Bounds()
	gridSize := s.config.GridSize

	var regions []ClassifiedRegion

	for y := bounds.Min.Y; y < bounds.Max.Y; y += gridSize {
		for x := bounds.Min.X; x < bounds.Max.X; x += gridSize {
			// Define region
			rect := image.Rect(x, y,
				min(x+gridSize, bounds.Max.X),
				min(y+gridSize, bounds.Max.Y))

			// Extract region pixels
			regionPixels, width, height := extractRegionPixels(img, rect)

			// Check if empty (skip blank regions)
			if s.config.SkipEmptyRegions && isRegionEmpty(regionPixels) {
				regions = append(regions, ClassifiedRegion{
					Rect:           rect,
					Classification: RegionEmpty,
				})
				continue
			}

			// Hash region
			var hash string
			if s.config.UseFuzzyHash {
				hash = HashRegionFuzzy(regionPixels, width, height)
			} else {
				hash = HashRegion(regionPixels)
			}

			// Classify based on DNA
			if elem, found := s.config.DNA.LookupElement(hash); found && elem.Frequency >= s.config.RecurringThreshold {
				// Recurring - retrieve from DNA
				regions = append(regions, ClassifiedRegion{
					Rect:           rect,
					Classification: RegionRecurring,
					Hash:           hash,
					Text:           elem.Text,
					Confidence:     elem.Confidence,
				})
			} else {
				// Novel - needs OCR
				regions = append(regions, ClassifiedRegion{
					Rect:           rect,
					Classification: RegionNovel,
					Hash:           hash,
				})
			}
		}
	}

	return regions
}

// ProcessWithDNA performs sparse OCR using DNA database
func (s *SparseOCR) ProcessWithDNA(ctx context.Context, img image.Image, ocrFunc func(context.Context, image.Image) (string, float64, error)) (*SparseOCRResult, error) {
	_ = time.Now() // For future timing metrics

	// Classify all regions
	regions := s.ClassifyRegions(img)

	result := &SparseOCRResult{}

	// Count classifications
	var novelRegions, recycledRegions, skippedRegions int
	var fullText string

	// Process regions in order (top-to-bottom, left-to-right for reading order)
	for _, region := range regions {
		switch region.Classification {
		case RegionRecurring:
			// Use DNA text
			fullText += region.Text + " "
			recycledRegions++

		case RegionNovel:
			// Perform OCR
			subImg := extractSubImage(img, region.Rect)
			text, confidence, err := ocrFunc(ctx, subImg)

			if err == nil && text != "" {
				fullText += text + " "

				// Learn this region if confidence is high enough
				if s.config.EnableLearning && confidence >= s.config.MinConfidence {
					bbox := [4]int{region.Rect.Min.X, region.Rect.Min.Y, region.Rect.Dx(), region.Rect.Dy()}
					s.config.DNA.RegisterElement(region.Hash, "detected", text, bbox, confidence)
				}
			}
			novelRegions++

		case RegionEmpty:
			// Skip entirely
			skippedRegions++
		}
	}

	// Calculate metrics
	result.Text = fullText
	result.NovelRegions = novelRegions
	result.RecycledRegions = recycledRegions
	result.SkippedRegions = skippedRegions

	// Estimate time saved (100ms per recycled region + 50ms per skipped)
	timeSavedMs := int64(recycledRegions*100 + skippedRegions*50)
	result.TimeSaved = time.Duration(timeSavedMs) * time.Millisecond
	s.config.DNA.RecordTimeSaved(timeSavedMs)

	// Calculate speedup
	totalProcessedRegions := float64(novelRegions + recycledRegions)
	if novelRegions > 0 {
		result.SpeedupFactor = totalProcessedRegions / float64(novelRegions)
	} else if totalProcessedRegions > 0 {
		result.SpeedupFactor = totalProcessedRegions // All recycled!
	} else {
		result.SpeedupFactor = 1.0
	}

	// Cache hit rate
	if totalProcessedRegions > 0 {
		result.CacheHitRate = float64(recycledRegions) / totalProcessedRegions
	}

	// DNA size
	total, _, _ := s.config.DNA.GetStats()
	result.DNASize = total

	return result, nil
}

// ProcessMultiplePages processes multiple pages with shared DNA learning
func (s *SparseOCR) ProcessMultiplePages(ctx context.Context, images []image.Image, ocrFunc func(context.Context, image.Image) (string, float64, error)) ([]*SparseOCRResult, error) {
	results := make([]*SparseOCRResult, len(images))

	for i, img := range images {
		result, err := s.ProcessWithDNA(ctx, img, ocrFunc)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", i+1, err)
		}
		results[i] = result
	}

	return results, nil
}

// GetDNAStats returns current DNA database statistics
func (s *SparseOCR) GetDNAStats() (total, recurring int, hitRate float64, timeSavedMs int64) {
	total, recurring, hitRate = s.config.DNA.GetStats()
	timeSavedMs = s.config.DNA.GetTimeSaved()
	return
}

// SaveDNA persists DNA database to disk
func (s *SparseOCR) SaveDNA(filepath string) error {
	return s.config.DNA.Save(filepath)
}

// LoadDNA restores DNA database from disk
func (s *SparseOCR) LoadDNA(filepath string) error {
	return s.config.DNA.Load(filepath)
}

// GetDNA returns the DNA database for direct access
func (s *SparseOCR) GetDNA() *PDFDNA {
	return s.config.DNA
}

// Helper functions

// extractRegionPixels extracts RGB pixel data from image region
func extractRegionPixels(img image.Image, rect image.Rectangle) ([]byte, int, int) {
	width := rect.Dx()
	height := rect.Dy()
	pixels := make([]byte, width*height*3)

	idx := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			pixels[idx] = rgbaToByte(r >> 8)
			pixels[idx+1] = rgbaToByte(g >> 8)
			pixels[idx+2] = rgbaToByte(b >> 8)
			idx += 3
		}
	}

	return pixels, width, height
}

// extractSubImage creates a sub-image from rectangle
func extractSubImage(img image.Image, rect image.Rectangle) image.Image {
	// If img supports SubImage, use it for efficiency
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	if si, ok := img.(subImager); ok {
		return si.SubImage(rect)
	}

	// Otherwise, create a new image with copied pixels
	subImg := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			subImg.Set(x-rect.Min.X, y-rect.Min.Y, img.At(x, y))
		}
	}
	return subImg
}

// isRegionEmpty checks if region is mostly white/blank (< 5% non-white pixels)
func isRegionEmpty(pixels []byte) bool {
	if len(pixels) == 0 {
		return true
	}

	nonWhitePixels := 0
	totalPixels := len(pixels) / 3

	for i := 0; i < len(pixels); i += 3 {
		r, g, b := pixels[i], pixels[i+1], pixels[i+2]
		// Consider pixel non-white if any channel < 240
		if r < 240 || g < 240 || b < 240 {
			nonWhitePixels++
		}
	}

	nonWhiteRatio := float64(nonWhitePixels) / float64(totalPixels)
	return nonWhiteRatio < 0.05 // Less than 5% content
}

// isColorNearWhite checks if color is close to white
func isColorNearWhite(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	// Consider "near white" if all channels > 240
	return (r>>8) > 240 && (g>>8) > 240 && (b>>8) > 240
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func rgbaToByte(v uint32) byte {
	if v > 0xFF {
		return 0xFF
	}
	return byte(v)
}
