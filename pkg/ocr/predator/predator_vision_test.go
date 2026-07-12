package predator

import (
	"context"
	"image"
	"image/color"
	"math"
	"testing"
	"time"
)

func TestPredatorVision_DefaultConfig(t *testing.T) {
	config := DefaultPredatorConfig()

	if !config.EnableUVChannel {
		t.Error("Expected UV channel enabled by default")
	}
	if !config.EnableSaliency {
		t.Error("Expected saliency enabled by default")
	}
	if !config.EnableOpticalFlow {
		t.Error("Expected optical flow enabled by default")
	}
	if !config.EnableAdaptiveFocus {
		t.Error("Expected adaptive focus enabled by default")
	}
	if config.UVBoostFactor != 1.5 {
		t.Errorf("Expected UV boost factor 1.5, got %f", config.UVBoostFactor)
	}
	if config.SaliencyThreshold != 0.3 {
		t.Errorf("Expected saliency threshold 0.3, got %f", config.SaliencyThreshold)
	}
}

func TestPredatorVision_NewPredatorVision(t *testing.T) {
	// Test with nil config
	pv := NewPredatorVision(nil)
	if pv.config == nil {
		t.Error("Expected default config when nil provided")
	}

	// Test with custom config
	customConfig := &PredatorConfig{
		EnableUVChannel:   false,
		EnableSaliency:    true,
		UVBoostFactor:     2.0,
		SaliencyThreshold: 0.5,
	}
	pv = NewPredatorVision(customConfig)
	if pv.config.UVBoostFactor != 2.0 {
		t.Errorf("Expected UV boost factor 2.0, got %f", pv.config.UVBoostFactor)
	}
}

func TestPredatorVision_Process_BasicImage(t *testing.T) {
	pv := NewPredatorVision(nil)

	// Create simple test image (100x100 white with black text-like region)
	img := createTestImage(100, 100)

	ctx := context.Background()
	result, err := pv.Process(ctx, img)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Image == nil {
		t.Error("Expected processed image")
	}

	if result.ProcessingMs <= 0 {
		t.Error("Expected positive processing time")
	}

	// Verify stats updated
	stats := pv.GetStats()
	if stats.ImagesProcessed != 1 {
		t.Errorf("Expected 1 image processed, got %d", stats.ImagesProcessed)
	}
	if stats.TotalPixels != 100*100 {
		t.Errorf("Expected 10000 pixels, got %d", stats.TotalPixels)
	}
}

func TestPredatorVision_UVChannel(t *testing.T) {
	config := DefaultPredatorConfig()
	config.EnableSaliency = false
	config.EnableOpticalFlow = false
	config.EnableAdaptiveFocus = false
	config.UVBoostFactor = 2.0

	pv := NewPredatorVision(config)

	// Create image with blue content (should be enhanced)
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{R: 50, G: 50, B: 150, A: 255})
		}
	}

	ctx := context.Background()
	result, err := pv.Process(ctx, img)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Check that blue channel was enhanced
	_, _, b, _ := result.Image.At(25, 25).RGBA()
	blueValue := float64(b >> 8)

	// With boost factor 2.0, blue should be significantly enhanced
	if blueValue <= 150 {
		t.Errorf("Expected enhanced blue channel, got %f", blueValue)
	}
}

func TestPredatorVision_SaliencyMap(t *testing.T) {
	config := DefaultPredatorConfig()
	config.EnableUVChannel = false
	config.EnableOpticalFlow = false
	config.EnableAdaptiveFocus = false

	pv := NewPredatorVision(config)

	// Create image with high contrast region (text-like)
	img := createTextLikeImage(100, 100)

	ctx := context.Background()
	result, err := pv.Process(ctx, img)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if len(result.SaliencyMap) != 100*100 {
		t.Errorf("Expected saliency map of size %d, got %d", 100*100, len(result.SaliencyMap))
	}

	// Check that saliency values are reasonable (0-1 range)
	for i, s := range result.SaliencyMap {
		if s < 0 || s > 1 {
			t.Errorf("Saliency value at %d out of range: %f", i, s)
		}
	}

	// Should find at least one focus region
	if len(result.FocusRegions) == 0 {
		t.Error("Expected at least one focus region")
	}
}

func TestPredatorVision_SkewDetection(t *testing.T) {
	config := DefaultPredatorConfig()
	config.EnableUVChannel = false
	config.EnableSaliency = false
	config.EnableAdaptiveFocus = false

	pv := NewPredatorVision(config)

	// Create straight text-like image (should have minimal skew)
	img := createTextLikeImage(200, 100)

	ctx := context.Background()
	result, err := pv.Process(ctx, img)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Skew should be small for straight image
	if math.Abs(result.SkewAngle) > 10 {
		t.Logf("Warning: Large skew angle detected for straight image: %f degrees", result.SkewAngle)
	}
}

func TestPredatorVision_AdaptiveFocus(t *testing.T) {
	config := DefaultPredatorConfig()
	config.EnableUVChannel = false
	config.EnableSaliency = false
	config.EnableOpticalFlow = false

	pv := NewPredatorVision(config)

	img := createBlurryImage(100, 100)

	ctx := context.Background()
	result, err := pv.Process(ctx, img)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Result image should exist
	if result.Image == nil {
		t.Error("Expected processed image")
	}

	// Bounds should match
	if result.Image.Bounds() != img.Bounds() {
		t.Error("Image bounds mismatch after processing")
	}
}

func TestPredatorVision_Stats(t *testing.T) {
	pv := NewPredatorVision(nil)
	img1 := createTestImage(100, 100)
	img2 := createTestImage(50, 50)

	ctx := context.Background()

	// Process first image
	_, err := pv.Process(ctx, img1)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	stats1 := pv.GetStats()
	if stats1.ImagesProcessed != 1 {
		t.Errorf("Expected 1 image, got %d", stats1.ImagesProcessed)
	}
	if stats1.TotalPixels != 10000 {
		t.Errorf("Expected 10000 pixels, got %d", stats1.TotalPixels)
	}

	// Process second image
	_, err = pv.Process(ctx, img2)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	stats2 := pv.GetStats()
	if stats2.ImagesProcessed != 2 {
		t.Errorf("Expected 2 images, got %d", stats2.ImagesProcessed)
	}
	if stats2.TotalPixels != 12500 { // 10000 + 2500
		t.Errorf("Expected 12500 pixels, got %d", stats2.TotalPixels)
	}
	// Duration should increase or stay the same (fast processing may have ~0 increase)
	if stats2.Duration < stats1.Duration {
		t.Errorf("Expected cumulative duration to be >= previous, got %v -> %v", stats1.Duration, stats2.Duration)
	}
}

func TestPredatorVision_FullPipeline(t *testing.T) {
	pv := NewPredatorVision(nil)

	// Create realistic test image
	img := createRealisticDocument(300, 400)

	ctx := context.Background()
	start := time.Now()
	result, err := pv.Process(ctx, img)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	t.Logf("Processing time: %v", duration)
	t.Logf("Processing ms (reported): %f", result.ProcessingMs)
	t.Logf("Skew angle: %f degrees", result.SkewAngle)
	t.Logf("Focus regions: %d", len(result.FocusRegions))
	t.Logf("Saliency map size: %d", len(result.SaliencyMap))

	// Verify all components ran
	if result.Image == nil {
		t.Error("Expected processed image")
	}
	if len(result.SaliencyMap) != 300*400 {
		t.Error("Expected saliency map")
	}
}

func TestPredatorVision_SkewCorrection(t *testing.T) {
	config := DefaultPredatorConfig()
	pv := NewPredatorVision(config)

	// Create image that should trigger skew correction
	img := createSkewedImage(200, 200, 2.0)

	ctx := context.Background()
	result, err := pv.Process(ctx, img)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// If skew > 0.5 degrees detected, it should be corrected
	stats := pv.GetStats()
	if math.Abs(result.SkewAngle) > 0.5 {
		if stats.SkewCorrected != 1 {
			t.Errorf("Expected skew correction, but SkewCorrected = %d", stats.SkewCorrected)
		}
	}
}

func TestLuminance(t *testing.T) {
	tests := []struct {
		color    color.Color
		expected float64
	}{
		{color.RGBA{R: 0, G: 0, B: 0, A: 255}, 0},
		{color.RGBA{R: 255, G: 255, B: 255, A: 255}, 255},
		{color.RGBA{R: 255, G: 0, B: 0, A: 255}, 76.245},  // 0.299 * 255
		{color.RGBA{R: 0, G: 255, B: 0, A: 255}, 149.685}, // 0.587 * 255
		{color.RGBA{R: 0, G: 0, B: 255, A: 255}, 29.07},   // 0.114 * 255
	}

	for _, tt := range tests {
		result := luminance(tt.color)
		if math.Abs(result-tt.expected) > 0.01 {
			t.Errorf("luminance(%v) = %f, expected %f", tt.color, result, tt.expected)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		v, min, max, expected float64
	}{
		{5, 0, 10, 5},
		{-5, 0, 10, 0},
		{15, 0, 10, 10},
		{7.5, 0, 255, 7.5},
		{300, 0, 255, 255},
	}

	for _, tt := range tests {
		result := clamp(tt.v, tt.min, tt.max)
		if result != tt.expected {
			t.Errorf("clamp(%f, %f, %f) = %f, expected %f",
				tt.v, tt.min, tt.max, result, tt.expected)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{5, 10, 10},
		{10, 5, 10},
		{0, 0, 0},
		{-5, 5, 5},
	}

	for _, tt := range tests {
		result := max(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("max(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// Benchmark tests
func BenchmarkPredatorVision_Process_Small(b *testing.B) {
	pv := NewPredatorVision(nil)
	img := createTestImage(100, 100)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pv.Process(ctx, img)
	}
}

func BenchmarkPredatorVision_Process_Medium(b *testing.B) {
	pv := NewPredatorVision(nil)
	img := createTestImage(500, 500)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pv.Process(ctx, img)
	}
}

func BenchmarkPredatorVision_Process_Large(b *testing.B) {
	pv := NewPredatorVision(nil)
	img := createTestImage(1000, 1000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pv.Process(ctx, img)
	}
}

func BenchmarkPredatorVision_UVChannel(b *testing.B) {
	config := DefaultPredatorConfig()
	config.EnableSaliency = false
	config.EnableOpticalFlow = false
	config.EnableAdaptiveFocus = false
	pv := NewPredatorVision(config)
	img := createTestImage(500, 500)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pv.Process(ctx, img)
	}
}

func BenchmarkPredatorVision_Saliency(b *testing.B) {
	config := DefaultPredatorConfig()
	config.EnableUVChannel = false
	config.EnableOpticalFlow = false
	config.EnableAdaptiveFocus = false
	pv := NewPredatorVision(config)
	img := createTestImage(500, 500)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pv.Process(ctx, img)
	}
}

// Helper functions for creating test images

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// White background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Black rectangle (text-like)
	for y := height / 4; y < height*3/4; y++ {
		for x := width / 4; x < width*3/4; x++ {
			img.Set(x, y, color.Black)
		}
	}

	return img
}

func createTextLikeImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// White background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Horizontal bars (text lines)
	for line := 0; line < 5; line++ {
		y := (line + 1) * height / 6
		for x := width / 10; x < width*9/10; x++ {
			for dy := -2; dy <= 2; dy++ {
				if y+dy >= 0 && y+dy < height {
					img.Set(x, y+dy, color.Black)
				}
			}
		}
	}

	return img
}

func createBlurryImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Gradient (blurry-like)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grayValue := uint8((x + y) % 256)
			img.Set(x, y, color.RGBA{R: grayValue, G: grayValue, B: grayValue, A: 255})
		}
	}

	return img
}

func createRealisticDocument(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Slightly off-white background (aged paper)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 240, G: 238, B: 230, A: 255})
		}
	}

	// Text lines
	lineHeight := height / 20
	for line := 0; line < 15; line++ {
		y := (line + 2) * lineHeight
		for x := width / 10; x < width*9/10; x += 2 {
			// Faded text
			grayValue := uint8(50 + (x%100)/2)
			for dy := -1; dy <= 1; dy++ {
				if y+dy >= 0 && y+dy < height {
					img.Set(x, y+dy, color.RGBA{R: grayValue, G: grayValue, B: grayValue, A: 255})
				}
			}
		}
	}

	// Some noise (paper texture)
	for i := 0; i < width*height/100; i++ {
		x := i % width
		y := (i * 7) % height
		img.Set(x, y, color.RGBA{R: 220, G: 215, B: 205, A: 255})
	}

	return img
}

func createSkewedImage(width, height int, skewAngle float64) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// White background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Skewed horizontal line
	rad := skewAngle * math.Pi / 180.0
	for x := 0; x < width; x++ {
		y := height/2 + int(float64(x)*math.Tan(rad))
		if y >= 0 && y < height {
			for dy := -2; dy <= 2; dy++ {
				if y+dy >= 0 && y+dy < height {
					img.Set(x, y+dy, color.Black)
				}
			}
		}
	}

	return img
}
