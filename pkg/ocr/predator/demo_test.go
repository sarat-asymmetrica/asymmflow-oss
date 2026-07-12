package predator

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"
)

// TestDemo_GenerateComparison generates before/after comparison images
// Run with: go test -run TestDemo_GenerateComparison
func TestDemo_GenerateComparison(t *testing.T) {
	// Create realistic faded document
	original := createFadedHistoricalDocument(800, 1000)

	// Process with Predator Vision
	pv := NewPredatorVision(nil)
	ctx := context.Background()
	result, err := pv.Process(ctx, original)
	if err != nil {
		t.Fatalf("Processing failed: %v", err)
	}

	// Save original
	saveImage(t, original, "demo_original.png")

	// Save processed
	saveImage(t, result.Image, "demo_processed.png")

	// Save saliency visualization
	saliencyImg := visualizeSaliency(result.SaliencyMap, original.Bounds())
	saveImage(t, saliencyImg, "demo_saliency.png")

	// Print statistics
	t.Logf("Demo images generated!")
	t.Logf("Processing time: %.2f ms", result.ProcessingMs)
	t.Logf("Skew detected: %.2f degrees", result.SkewAngle)
	t.Logf("Focus regions: %d", len(result.FocusRegions))
	t.Logf("\nFiles created:")
	t.Logf("  - demo_original.png (faded historical document)")
	t.Logf("  - demo_processed.png (after Predator Vision)")
	t.Logf("  - demo_saliency.png (saliency map visualization)")
}

// TestDemo_CompareConfigurations compares different processing modes
func TestDemo_CompareConfigurations(t *testing.T) {
	original := createFadedHistoricalDocument(600, 800)
	ctx := context.Background()

	configs := map[string]*PredatorConfig{
		"fast": {
			EnableUVChannel:     true,
			EnableSaliency:      false,
			EnableOpticalFlow:   false,
			EnableAdaptiveFocus: false,
			UVBoostFactor:       1.3,
		},
		"quality": {
			EnableUVChannel:     true,
			EnableSaliency:      true,
			EnableOpticalFlow:   true,
			EnableAdaptiveFocus: true,
			UVBoostFactor:       2.0,
			SaliencyThreshold:   0.2,
		},
		"uv_only": {
			EnableUVChannel:     true,
			EnableSaliency:      false,
			EnableOpticalFlow:   false,
			EnableAdaptiveFocus: false,
			UVBoostFactor:       2.0,
		},
		"focus_only": {
			EnableUVChannel:     false,
			EnableSaliency:      false,
			EnableOpticalFlow:   false,
			EnableAdaptiveFocus: true,
		},
	}

	for name, config := range configs {
		pv := NewPredatorVision(config)
		result, err := pv.Process(ctx, original)
		if err != nil {
			t.Errorf("Config %s failed: %v", name, err)
			continue
		}

		filename := fmt.Sprintf("demo_config_%s.png", name)
		saveImage(t, result.Image, filename)

		t.Logf("\nConfig: %s", name)
		t.Logf("  Time: %.2f ms", result.ProcessingMs)
		t.Logf("  Skew: %.2f degrees", result.SkewAngle)
		t.Logf("  File: %s", filename)
	}
}

// TestDemo_ProgressiveEnhancement shows step-by-step processing
func TestDemo_ProgressiveEnhancement(t *testing.T) {
	original := createFadedHistoricalDocument(600, 800)
	ctx := context.Background()

	// Step 1: Original
	saveImage(t, original, "demo_step1_original.png")

	// Step 2: UV channel only
	config2 := &PredatorConfig{
		EnableUVChannel:     true,
		EnableSaliency:      false,
		EnableOpticalFlow:   false,
		EnableAdaptiveFocus: false,
		UVBoostFactor:       1.5,
	}
	pv2 := NewPredatorVision(config2)
	result2, _ := pv2.Process(ctx, original)
	saveImage(t, result2.Image, "demo_step2_uv.png")

	// Step 3: UV + Skew correction
	config3 := &PredatorConfig{
		EnableUVChannel:     true,
		EnableSaliency:      false,
		EnableOpticalFlow:   true,
		EnableAdaptiveFocus: false,
		UVBoostFactor:       1.5,
	}
	pv3 := NewPredatorVision(config3)
	result3, _ := pv3.Process(ctx, original)
	saveImage(t, result3.Image, "demo_step3_uv_skew.png")

	// Step 4: UV + Skew + Focus
	config4 := &PredatorConfig{
		EnableUVChannel:     true,
		EnableSaliency:      false,
		EnableOpticalFlow:   true,
		EnableAdaptiveFocus: true,
		UVBoostFactor:       1.5,
	}
	pv4 := NewPredatorVision(config4)
	result4, _ := pv4.Process(ctx, original)
	saveImage(t, result4.Image, "demo_step4_uv_skew_focus.png")

	// Step 5: Full pipeline
	pv5 := NewPredatorVision(nil)
	result5, _ := pv5.Process(ctx, original)
	saveImage(t, result5.Image, "demo_step5_full.png")

	t.Logf("Progressive enhancement images created!")
	t.Logf("  1. demo_step1_original.png")
	t.Logf("  2. demo_step2_uv.png")
	t.Logf("  3. demo_step3_uv_skew.png")
	t.Logf("  4. demo_step4_uv_skew_focus.png")
	t.Logf("  5. demo_step5_full.png")
}

// TestDemo_UVChannelEffect demonstrates UV enhancement on different ink colors
func TestDemo_UVChannelEffect(t *testing.T) {
	// Create document with different ink colors
	img := image.NewRGBA(image.Rect(0, 0, 600, 400))

	// Background: aged paper
	for y := 0; y < 400; y++ {
		for x := 0; x < 600; x++ {
			img.Set(x, y, color.RGBA{R: 235, G: 230, B: 220, A: 255})
		}
	}

	// Black ink (top section)
	drawText(img, 50, 50, "BLACK INK (faded)", color.RGBA{R: 60, G: 60, B: 60, A: 255})

	// Blue ink (middle section)
	drawText(img, 50, 150, "BLUE INK (faded)", color.RGBA{R: 80, G: 80, B: 120, A: 255})

	// Brown ink (bottom section)
	drawText(img, 50, 250, "BROWN INK (faded)", color.RGBA{R: 100, G: 70, B: 50, A: 255})

	// Save original
	saveImage(t, img, "demo_uv_original.png")

	// Process with strong UV boost
	config := &PredatorConfig{
		EnableUVChannel:     true,
		EnableSaliency:      false,
		EnableOpticalFlow:   false,
		EnableAdaptiveFocus: false,
		UVBoostFactor:       2.5, // Strong boost
	}
	pv := NewPredatorVision(config)
	result, _ := pv.Process(context.Background(), img)
	saveImage(t, result.Image, "demo_uv_enhanced.png")

	t.Logf("UV effect demonstration created!")
	t.Logf("  - demo_uv_original.png (faded inks)")
	t.Logf("  - demo_uv_enhanced.png (UV boost = 2.5)")
	t.Logf("\nNotice: Blue ink shows most improvement (UV simulation)")
}

// Helper functions

func createFadedHistoricalDocument(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Aged, yellowed paper with slight texture
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Base color: aged paper
			noise := float64((x*7+y*13)%20) - 10 // -10 to +10
			r := clamp(235+noise, 220, 250)
			g := clamp(225+noise, 210, 240)
			b := clamp(210+noise, 195, 225)
			img.Set(x, y, color.RGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: 255,
			})
		}
	}

	// Add faded text (very low contrast)
	lineHeight := height / 20
	for line := 0; line < 15; line++ {
		y := (line + 2) * lineHeight

		// Add slight skew (simulate warped page)
		skewOffset := int(float64(line) * 0.5)

		for x := width/10 + skewOffset; x < width*9/10; x += 3 {
			// Very faded ink (gray, low contrast)
			grayValue := uint8(130 + (x%50)/2) // 130-155 range (faded!)

			// Add some variation to simulate different ink densities
			if (x/20)%3 == 0 {
				grayValue += 10 // Slightly darker
			}

			for dy := -1; dy <= 1; dy++ {
				if y+dy >= 0 && y+dy < height {
					img.Set(x, y+dy, color.RGBA{
						R: grayValue,
						G: grayValue,
						B: uint8(float64(grayValue) * 1.05), // Slight blue tint
						A: 255,
					})
				}
			}
		}
	}

	// Add some age spots and stains
	for i := 0; i < 50; i++ {
		cx := (i * 127) % width
		cy := (i * 193) % height
		radius := 10 + (i%10)*2

		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if dx*dx+dy*dy <= radius*radius {
					x, y := cx+dx, cy+dy
					if x >= 0 && x < width && y >= 0 && y < height {
						// Brown stain
						current := img.At(x, y)
						r, g, b, _ := current.RGBA()
						img.Set(x, y, color.RGBA{
							R: uint8((r >> 8) * 95 / 100),
							G: uint8((g >> 8) * 90 / 100),
							B: uint8((b >> 8) * 85 / 100),
							A: 255,
						})
					}
				}
			}
		}
	}

	return img
}

func visualizeSaliency(saliency []float64, bounds image.Rectangle) image.Image {
	width := bounds.Dx()
	height := bounds.Dy()
	img := image.NewRGBA(bounds)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if idx < len(saliency) {
				s := saliency[idx]

				// Heatmap: blue (low) -> green -> yellow -> red (high)
				var r, g, b uint8
				if s < 0.25 {
					// Blue to cyan
					t := s / 0.25
					r = 0
					g = uint8(t * 255)
					b = 255
				} else if s < 0.5 {
					// Cyan to green
					t := (s - 0.25) / 0.25
					r = 0
					g = 255
					b = uint8((1 - t) * 255)
				} else if s < 0.75 {
					// Green to yellow
					t := (s - 0.5) / 0.25
					r = uint8(t * 255)
					g = 255
					b = 0
				} else {
					// Yellow to red
					t := (s - 0.75) / 0.25
					r = 255
					g = uint8((1 - t) * 255)
					b = 0
				}

				img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}

	return img
}

func drawText(img *image.RGBA, x, y int, text string, col color.RGBA) {
	// Simple block letters (very basic, just for demo)
	for i, ch := range text {
		drawChar(img, x+i*12, y, ch, col)
	}
}

func drawChar(img *image.RGBA, x, y int, ch rune, col color.RGBA) {
	// Draw a simple 8x12 block character
	// Just draw rectangles for demo purposes
	for dy := 0; dy < 12; dy++ {
		for dx := 0; dx < 8; dx++ {
			// Simple pattern based on character
			if (dx+dy+int(ch))%3 == 0 {
				img.Set(x+dx, y+dy, col)
			}
		}
	}
}

func saveImage(t *testing.T, img image.Image, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		t.Logf("Warning: Could not create %s: %v", filename, err)
		return
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		t.Logf("Warning: Could not encode %s: %v", filename, err)
	}
}

// TestDemo_SkewCorrection demonstrates skew detection and correction
func TestDemo_SkewCorrection(t *testing.T) {
	angles := []float64{0, 2, 5, 10, -3}

	for _, angle := range angles {
		// Create skewed document
		img := createSkewedDocument(600, 800, angle)

		// Save original
		filename := fmt.Sprintf("demo_skew_%.0f_original.png", angle)
		saveImage(t, img, filename)

		// Process
		config := &PredatorConfig{
			EnableUVChannel:     false,
			EnableSaliency:      false,
			EnableOpticalFlow:   true, // Skew detection
			EnableAdaptiveFocus: false,
		}
		pv := NewPredatorVision(config)
		result, _ := pv.Process(context.Background(), img)

		// Save corrected
		correctedFilename := fmt.Sprintf("demo_skew_%.0f_corrected.png", angle)
		saveImage(t, result.Image, correctedFilename)

		t.Logf("Skew %.0f°: detected %.2f°", angle, result.SkewAngle)
	}

	t.Logf("\nSkew correction demo created!")
	t.Logf("Files: demo_skew_*_original.png and demo_skew_*_corrected.png")
}

func createSkewedDocument(width, height int, skewAngle float64) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// White background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Draw skewed horizontal lines
	rad := skewAngle * math.Pi / 180.0
	numLines := 15

	for line := 0; line < numLines; line++ {
		baseY := (line + 2) * height / (numLines + 3)

		for x := width / 10; x < width*9/10; x++ {
			y := baseY + int(float64(x-width/2)*math.Tan(rad))

			if y >= 0 && y < height {
				for dy := -2; dy <= 2; dy++ {
					if y+dy >= 0 && y+dy < height {
						img.Set(x, y+dy, color.Black)
					}
				}
			}
		}
	}

	return img
}
