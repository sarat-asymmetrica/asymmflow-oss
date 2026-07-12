package predator

import (
	"context"
	"image"
	"image/color"
	"math"
	"sync"
	"time"
)

// PredatorConfig configures predator vision preprocessing
type PredatorConfig struct {
	EnableUVChannel     bool    // Simulate UV channel (eagle vision)
	EnableSaliency      bool    // Saccadic attention (focus on text regions)
	EnableOpticalFlow   bool    // Motion detection for skew (owl vision)
	EnableAdaptiveFocus bool    // Laplacian pyramid focus
	UVBoostFactor       float64 // UV enhancement strength (default: 1.5)
	SaliencyThreshold   float64 // Min saliency to keep (default: 0.3)
}

// PredatorResult contains preprocessing results
type PredatorResult struct {
	Image        image.Image
	SaliencyMap  []float64 // Per-pixel saliency scores
	SkewAngle    float64   // Detected skew in degrees
	FocusRegions []image.Rectangle
	ProcessingMs float64
}

// PredatorVision implements bird-inspired image preprocessing
type PredatorVision struct {
	config *PredatorConfig
	stats  *PredatorStats
	mu     sync.RWMutex
}

// PredatorStats tracks processing statistics
type PredatorStats struct {
	ImagesProcessed int
	TotalPixels     int64
	SkewCorrected   int
	FadedRecovered  int
	Duration        time.Duration
}

// DefaultPredatorConfig returns production defaults
func DefaultPredatorConfig() *PredatorConfig {
	return &PredatorConfig{
		EnableUVChannel:     true,
		EnableSaliency:      true,
		EnableOpticalFlow:   true,
		EnableAdaptiveFocus: true,
		UVBoostFactor:       1.5,
		SaliencyThreshold:   0.3,
	}
}

// NewPredatorVision creates a new predator vision processor
func NewPredatorVision(config *PredatorConfig) *PredatorVision {
	if config == nil {
		config = DefaultPredatorConfig()
	}
	return &PredatorVision{
		config: config,
		stats:  &PredatorStats{},
	}
}

// Process applies predator vision preprocessing
func (p *PredatorVision) Process(ctx context.Context, img image.Image) (*PredatorResult, error) {
	start := time.Now()
	result := &PredatorResult{}

	bounds := img.Bounds()
	processed := image.NewRGBA(bounds)

	// Step 1: UV Channel Simulation (Eagle Tetrachromacy)
	if p.config.EnableUVChannel {
		img = p.applyUVChannel(img)
	}

	// Step 2: Saliency Map (Saccadic Attention)
	if p.config.EnableSaliency {
		result.SaliencyMap = p.computeSaliencyMap(img)
		result.FocusRegions = p.findSalientRegions(result.SaliencyMap, bounds)
	}

	// Step 3: Optical Flow Skew Detection (Owl Motion Detection)
	if p.config.EnableOpticalFlow {
		result.SkewAngle = p.detectSkew(img)
		if math.Abs(result.SkewAngle) > 0.5 {
			img = p.correctSkew(img, result.SkewAngle)
			p.mu.Lock()
			p.stats.SkewCorrected++
			p.mu.Unlock()
		}
	}

	// Step 4: Adaptive Focus (Laplacian Pyramid)
	if p.config.EnableAdaptiveFocus {
		img = p.applyAdaptiveFocus(img)
	}

	// Copy to result
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			processed.Set(x, y, img.At(x, y))
		}
	}
	result.Image = processed

	// Update stats
	p.mu.Lock()
	p.stats.ImagesProcessed++
	p.stats.TotalPixels += int64(bounds.Dx() * bounds.Dy())
	p.stats.Duration += time.Since(start)
	p.mu.Unlock()

	result.ProcessingMs = float64(time.Since(start).Microseconds()) / 1000.0

	return result, nil
}

// applyUVChannel simulates UV perception (enhances blue channel + edge detection)
func (p *PredatorVision) applyUVChannel(img image.Image) image.Image {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	boost := p.config.UVBoostFactor

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			// UV simulation: boost blue channel + enhance contrast in blue-UV range
			rf := float64(r >> 8)
			gf := float64(g >> 8)
			bf := float64(b >> 8)

			// Simulate UV as enhanced blue with edge emphasis
			uvComponent := bf * boost
			if uvComponent > 255 {
				uvComponent = 255
			}

			// Also boost overall contrast for faded documents
			rf = clamp(rf*1.1, 0, 255)
			gf = clamp(gf*1.1, 0, 255)

			result.Set(x, y, color.RGBA{
				R: uint8(uint32(rf) & 0xff),
				G: uint8(uint32(gf) & 0xff),
				B: uint8(uint32(uvComponent) & 0xff),
				A: uint8((a >> 8) & 0xff),
			})
		}
	}

	return result
}

// computeSaliencyMap computes per-pixel saliency (text regions are salient)
func (p *PredatorVision) computeSaliencyMap(img image.Image) []float64 {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	saliency := make([]float64, width*height)

	// Simple saliency: high contrast = high saliency
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Compute local contrast (Laplacian)
			center := luminance(img.At(x+bounds.Min.X, y+bounds.Min.Y))
			neighbors := (luminance(img.At(x-1+bounds.Min.X, y+bounds.Min.Y)) +
				luminance(img.At(x+1+bounds.Min.X, y+bounds.Min.Y)) +
				luminance(img.At(x+bounds.Min.X, y-1+bounds.Min.Y)) +
				luminance(img.At(x+bounds.Min.X, y+1+bounds.Min.Y))) / 4

			contrast := math.Abs(center - neighbors)
			saliency[y*width+x] = contrast / 255.0
		}
	}

	return saliency
}

// findSalientRegions finds bounding boxes of salient (text) regions
func (p *PredatorVision) findSalientRegions(saliency []float64, bounds image.Rectangle) []image.Rectangle {
	// Simple connected component on thresholded saliency
	// For now, return whole image if any saliency above threshold
	var regions []image.Rectangle

	threshold := p.config.SaliencyThreshold
	for _, s := range saliency {
		if s > threshold {
			regions = append(regions, bounds)
			break
		}
	}

	return regions
}

// detectSkew estimates document skew angle using gradient analysis
func (p *PredatorVision) detectSkew(img image.Image) float64 {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Compute horizontal gradients
	var totalAngle float64
	var count int

	sampleStep := max(1, width/100) // Sample every ~1% of width

	for y := height / 4; y < height*3/4; y += sampleStep {
		for x := 1; x < width-1; x += sampleStep {
			// Horizontal gradient
			left := luminance(img.At(x-1+bounds.Min.X, y+bounds.Min.Y))
			right := luminance(img.At(x+1+bounds.Min.X, y+bounds.Min.Y))
			dx := right - left

			// Vertical gradient
			top := luminance(img.At(x+bounds.Min.X, y-1+bounds.Min.Y))
			bottom := luminance(img.At(x+bounds.Min.X, y+1+bounds.Min.Y))
			dy := bottom - top

			if math.Abs(dx) > 10 { // Significant edge
				angle := math.Atan2(dy, dx) * 180 / math.Pi
				totalAngle += angle
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}

	return totalAngle / float64(count)
}

// correctSkew rotates image to correct skew
func (p *PredatorVision) correctSkew(img image.Image, angle float64) image.Image {
	// For small angles, use affine transform
	if math.Abs(angle) < 0.5 {
		return img
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Create rotated image
	result := image.NewRGBA(bounds)

	// Convert angle to radians
	rad := angle * math.Pi / 180.0
	cosA := math.Cos(rad)
	sinA := math.Sin(rad)

	// Center of rotation
	cx := float64(width) / 2.0
	cy := float64(height) / 2.0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Translate to origin
			tx := float64(x) - cx
			ty := float64(y) - cy

			// Rotate
			rx := tx*cosA - ty*sinA
			ry := tx*sinA + ty*cosA

			// Translate back
			sx := int(rx + cx)
			sy := int(ry + cy)

			// Bounds check
			if sx >= 0 && sx < width && sy >= 0 && sy < height {
				result.Set(x+bounds.Min.X, y+bounds.Min.Y,
					img.At(sx+bounds.Min.X, sy+bounds.Min.Y))
			} else {
				// Fill with white for out-of-bounds
				result.Set(x+bounds.Min.X, y+bounds.Min.Y, color.White)
			}
		}
	}

	return result
}

// applyAdaptiveFocus enhances text regions using Laplacian pyramid
func (p *PredatorVision) applyAdaptiveFocus(img image.Image) image.Image {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	// Simple sharpening kernel
	for y := 1; y < bounds.Dy()-1; y++ {
		for x := 1; x < bounds.Dx()-1; x++ {
			// Laplacian sharpening
			center := img.At(x+bounds.Min.X, y+bounds.Min.Y)
			r, g, b, a := center.RGBA()

			// Get neighbors for edge enhancement
			neighbors := []color.Color{
				img.At(x-1+bounds.Min.X, y+bounds.Min.Y),
				img.At(x+1+bounds.Min.X, y+bounds.Min.Y),
				img.At(x+bounds.Min.X, y-1+bounds.Min.Y),
				img.At(x+bounds.Min.X, y+1+bounds.Min.Y),
			}

			var avgR, avgG, avgB float64
			for _, n := range neighbors {
				nr, ng, nb, _ := n.RGBA()
				avgR += float64(nr >> 8)
				avgG += float64(ng >> 8)
				avgB += float64(nb >> 8)
			}
			avgR /= 4
			avgG /= 4
			avgB /= 4

			// Sharpen: center + (center - average) * factor
			factor := 0.5
			newR := clamp(float64(r>>8)+factor*(float64(r>>8)-avgR), 0, 255)
			newG := clamp(float64(g>>8)+factor*(float64(g>>8)-avgG), 0, 255)
			newB := clamp(float64(b>>8)+factor*(float64(b>>8)-avgB), 0, 255)

			result.Set(x+bounds.Min.X, y+bounds.Min.Y, color.RGBA{
				R: uint8(uint32(newR) & 0xff),
				G: uint8(uint32(newG) & 0xff),
				B: uint8(uint32(newB) & 0xff),
				A: uint8((a >> 8) & 0xff),
			})
		}
	}

	// Fill borders
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		result.Set(bounds.Min.X, y, img.At(bounds.Min.X, y))
		result.Set(bounds.Max.X-1, y, img.At(bounds.Max.X-1, y))
	}
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		result.Set(x, bounds.Min.Y, img.At(x, bounds.Min.Y))
		result.Set(x, bounds.Max.Y-1, img.At(x, bounds.Max.Y-1))
	}

	return result
}

// GetStats returns processing statistics
func (p *PredatorVision) GetStats() *PredatorStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return &PredatorStats{
		ImagesProcessed: p.stats.ImagesProcessed,
		TotalPixels:     p.stats.TotalPixels,
		SkewCorrected:   p.stats.SkewCorrected,
		FadedRecovered:  p.stats.FadedRecovered,
		Duration:        p.stats.Duration,
	}
}

// Helper functions
func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
