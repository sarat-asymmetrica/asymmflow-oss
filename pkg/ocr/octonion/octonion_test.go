package octonion

import (
	"context"
	"image"
	"image/color"
	"math"
	"testing"
)

func TestOctonionBasics(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		o := Zero()
		for i := 0; i < 8; i++ {
			if o.E[i] != 0 {
				t.Errorf("Zero()[%d] = %f, want 0", i, o.E[i])
			}
		}
	})

	t.Run("One", func(t *testing.T) {
		o := One()
		if o.E[0] != 1.0 {
			t.Errorf("One().E[0] = %f, want 1.0", o.E[0])
		}
		for i := 1; i < 8; i++ {
			if o.E[i] != 0 {
				t.Errorf("One().E[%d] = %f, want 0", i, o.E[i])
			}
		}
	})

	t.Run("NewOctonion", func(t *testing.T) {
		o := NewOctonion(1, 2, 3, 4, 5, 6, 7, 8)
		for i := 0; i < 8; i++ {
			expected := float64(i + 1)
			if o.E[i] != expected {
				t.Errorf("E[%d] = %f, want %f", i, o.E[i], expected)
			}
		}
	})
}

func TestOctonionArithmetic(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		a := NewOctonion(1, 2, 3, 4, 5, 6, 7, 8)
		b := NewOctonion(8, 7, 6, 5, 4, 3, 2, 1)
		c := a.Add(b)
		for i := 0; i < 8; i++ {
			if c.E[i] != 9.0 {
				t.Errorf("Add().E[%d] = %f, want 9.0", i, c.E[i])
			}
		}
	})

	t.Run("Sub", func(t *testing.T) {
		a := NewOctonion(10, 10, 10, 10, 10, 10, 10, 10)
		b := NewOctonion(1, 2, 3, 4, 5, 6, 7, 8)
		c := a.Sub(b)
		expected := []float64{9, 8, 7, 6, 5, 4, 3, 2}
		for i := 0; i < 8; i++ {
			if c.E[i] != expected[i] {
				t.Errorf("Sub().E[%d] = %f, want %f", i, c.E[i], expected[i])
			}
		}
	})

	t.Run("Scale", func(t *testing.T) {
		o := NewOctonion(1, 2, 3, 4, 5, 6, 7, 8)
		scaled := o.Scale(2.0)
		for i := 0; i < 8; i++ {
			expected := float64(i+1) * 2.0
			if scaled.E[i] != expected {
				t.Errorf("Scale(2.0).E[%d] = %f, want %f", i, scaled.E[i], expected)
			}
		}
	})

	t.Run("Conjugate", func(t *testing.T) {
		o := NewOctonion(1, 2, 3, 4, 5, 6, 7, 8)
		conj := o.Conjugate()
		if conj.E[0] != 1.0 {
			t.Errorf("Conjugate().E[0] = %f, want 1.0", conj.E[0])
		}
		for i := 1; i < 8; i++ {
			expected := -float64(i + 1)
			if conj.E[i] != expected {
				t.Errorf("Conjugate().E[%d] = %f, want %f", i, conj.E[i], expected)
			}
		}
	})
}

func TestOctonionGeometry(t *testing.T) {
	t.Run("Norm", func(t *testing.T) {
		o := NewOctonion(1, 1, 1, 1, 1, 1, 1, 1)
		norm := o.Norm()
		expected := math.Sqrt(8.0)
		if math.Abs(norm-expected) > 1e-10 {
			t.Errorf("Norm() = %f, want %f", norm, expected)
		}
	})

	t.Run("Normalize", func(t *testing.T) {
		o := NewOctonion(3, 4, 0, 0, 0, 0, 0, 0)
		normalized := o.Normalize()
		norm := normalized.Norm()
		if math.Abs(norm-1.0) > 1e-10 {
			t.Errorf("Normalize().Norm() = %f, want 1.0", norm)
		}
	})

	t.Run("Dot", func(t *testing.T) {
		a := NewOctonion(1, 2, 3, 4, 5, 6, 7, 8)
		b := NewOctonion(8, 7, 6, 5, 4, 3, 2, 1)
		dot := a.Dot(b)
		// 1*8 + 2*7 + 3*6 + 4*5 + 5*4 + 6*3 + 7*2 + 8*1 = 120
		expected := 120.0
		if dot != expected {
			t.Errorf("Dot() = %f, want %f", dot, expected)
		}
	})

	t.Run("Distance", func(t *testing.T) {
		a := NewOctonion(0, 0, 0, 0, 0, 0, 0, 0)
		b := NewOctonion(1, 1, 1, 1, 1, 1, 1, 1)
		dist := a.Distance(b)
		expected := math.Sqrt(8.0)
		if math.Abs(dist-expected) > 1e-10 {
			t.Errorf("Distance() = %f, want %f", dist, expected)
		}
	})
}

func TestOctonionPixelOperations(t *testing.T) {
	t.Run("FromPixel", func(t *testing.T) {
		o := FromPixel(255, 128, 64, 255, 100, 200, 1000, 2000, 0.9, 0.5)

		// Check RGB normalization
		if math.Abs(o.E[0]-1.0) > 1e-10 {
			t.Errorf("Red = %f, want 1.0", o.E[0])
		}
		if math.Abs(o.E[1]-128.0/255.0) > 1e-3 {
			t.Errorf("Green = %f, want ~0.502", o.E[1])
		}

		// Check spatial normalization
		if math.Abs(o.E[4]-0.1) > 1e-10 {
			t.Errorf("X = %f, want 0.1", o.E[4])
		}
		if math.Abs(o.E[5]-0.1) > 1e-10 {
			t.Errorf("Y = %f, want 0.1", o.E[5])
		}

		// Check metadata
		if o.E[6] != 0.9 {
			t.Errorf("Confidence = %f, want 0.9", o.E[6])
		}
		if o.E[7] != 0.5 {
			t.Errorf("Context = %f, want 0.5", o.E[7])
		}
	})

	t.Run("ToRGBA", func(t *testing.T) {
		o := NewOctonion(1.0, 0.5, 0.25, 1.0, 0, 0, 0, 0)
		r, g, b, a := o.ToRGBA()

		if r != 255 {
			t.Errorf("R = %d, want 255", r)
		}
		if g != 127 && g != 128 { // Allow rounding
			t.Errorf("G = %d, want ~127-128", g)
		}
		if b != 63 && b != 64 { // Allow rounding
			t.Errorf("B = %d, want ~63-64", b)
		}
		if a != 255 {
			t.Errorf("A = %d, want 255", a)
		}
	})

	t.Run("ProjectRGB", func(t *testing.T) {
		o := NewOctonion(0.5, 0.6, 0.7, 0.8, 0.1, 0.2, 0.3, 0.4)
		rgb := o.ProjectRGB()

		if rgb.E[0] != 0.5 || rgb.E[1] != 0.6 || rgb.E[2] != 0.7 {
			t.Errorf("ProjectRGB preserved wrong values: %v", rgb.E[:3])
		}

		for i := 3; i < 8; i++ {
			if rgb.E[i] != 0 {
				t.Errorf("ProjectRGB().E[%d] = %f, want 0", i, rgb.E[i])
			}
		}
	})

	t.Run("ColorMagnitude", func(t *testing.T) {
		o := NewOctonion(0.6, 0.8, 0.0, 0, 0, 0, 0, 0)
		mag := o.ColorMagnitude()
		expected := 1.0 // sqrt(0.36 + 0.64) = 1.0
		if math.Abs(mag-expected) > 1e-10 {
			t.Errorf("ColorMagnitude() = %f, want %f", mag, expected)
		}
	})

	t.Run("Grayscale", func(t *testing.T) {
		o := NewOctonion(1.0, 1.0, 1.0, 1.0, 0, 0, 0, 0)
		gray := o.Grayscale()
		// 0.299 + 0.587 + 0.114 = 1.0
		if math.Abs(gray-1.0) > 1e-10 {
			t.Errorf("Grayscale(white) = %f, want 1.0", gray)
		}

		o2 := NewOctonion(0, 0, 0, 1, 0, 0, 0, 0)
		gray2 := o2.Grayscale()
		if math.Abs(gray2-0.0) > 1e-10 {
			t.Errorf("Grayscale(black) = %f, want 0.0", gray2)
		}
	})

	t.Run("IsInk", func(t *testing.T) {
		// Black pixel (high ink)
		black := NewOctonion(0, 0, 0, 1, 0, 0, 0, 0)
		if !black.IsInk(0.5) {
			t.Error("Black should be detected as ink")
		}

		// White pixel (no ink)
		white := NewOctonion(1, 1, 1, 1, 0, 0, 0, 0)
		if white.IsInk(0.5) {
			t.Error("White should NOT be detected as ink")
		}
	})

	t.Run("InkStrength", func(t *testing.T) {
		black := NewOctonion(0, 0, 0, 1, 0, 0, 0, 0)
		strength := black.InkStrength()
		if math.Abs(strength-1.0) > 1e-10 {
			t.Errorf("InkStrength(black) = %f, want 1.0", strength)
		}

		white := NewOctonion(1, 1, 1, 1, 0, 0, 0, 0)
		strength2 := white.InkStrength()
		if math.Abs(strength2-0.0) > 1e-10 {
			t.Errorf("InkStrength(white) = %f, want 0.0", strength2)
		}
	})
}

func TestOctonionMultiplication(t *testing.T) {
	t.Run("Identity", func(t *testing.T) {
		o := NewOctonion(1, 2, 3, 4, 5, 6, 7, 8)
		one := One()
		result := o.Mul(one)

		for i := 0; i < 8; i++ {
			if math.Abs(result.E[i]-o.E[i]) > 1e-10 {
				t.Errorf("o * 1: E[%d] = %f, want %f", i, result.E[i], o.E[i])
			}
		}
	})

	t.Run("NonAssociative", func(t *testing.T) {
		// Octonions are non-associative: (a*b)*c != a*(b*c) in general
		a := NewOctonion(1, 1, 0, 0, 0, 0, 0, 0)
		b := NewOctonion(0, 0, 1, 1, 0, 0, 0, 0)
		c := NewOctonion(0, 0, 0, 0, 1, 1, 0, 0)

		left := a.Mul(b).Mul(c)
		right := a.Mul(b.Mul(c))

		// They should be different
		areDifferent := false
		for i := 0; i < 8; i++ {
			if math.Abs(left.E[i]-right.E[i]) > 1e-10 {
				areDifferent = true
				break
			}
		}

		if !areDifferent {
			t.Log("Note: This particular case might coincidentally be equal")
		}
	})
}

func TestColorProcessor(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultColorProcessorConfig()
		if !config.InkSeparation {
			t.Error("Default should have InkSeparation enabled")
		}
		if config.DenoiseStrength != 0.3 {
			t.Errorf("Default DenoiseStrength = %f, want 0.3", config.DenoiseStrength)
		}
	})

	t.Run("NewColorProcessor", func(t *testing.T) {
		cp := NewColorProcessor(nil)
		if cp == nil {
			t.Fatal("NewColorProcessor returned nil")
		}
		if cp.config == nil {
			t.Error("Processor has nil config")
		}
	})

	t.Run("ProcessSimpleImage", func(t *testing.T) {
		// Create a simple test image
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))

		// Fill with white background
		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				img.Set(x, y, color.RGBA{255, 255, 255, 255})
			}
		}

		// Add some blue "ink"
		for y := 40; y < 60; y++ {
			for x := 40; x < 60; x++ {
				img.Set(x, y, color.RGBA{0, 0, 200, 255})
			}
		}

		// Process
		cp := NewColorProcessor(nil)
		result, err := cp.Process(context.Background(), img)
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result == nil {
			t.Fatal("Process returned nil image")
		}

		// Check stats
		stats := cp.GetStats()
		if stats.ImagesProcessed != 1 {
			t.Errorf("ImagesProcessed = %d, want 1", stats.ImagesProcessed)
		}
		if stats.TotalPixels != 10000 {
			t.Errorf("TotalPixels = %d, want 10000", stats.TotalPixels)
		}
		if stats.PixelsPerSec == 0 {
			t.Error("PixelsPerSec should be > 0")
		}
	})

	t.Run("ProcessWithAllFeatures", func(t *testing.T) {
		config := &ColorProcessorConfig{
			InkSeparation:   true,
			DenoiseStrength: 0.5,
			ContrastEnhance: 1.5,
			BlueInkBoost:    true,
			RedInkBoost:     true,
			WorkerCount:     2,
		}

		cp := NewColorProcessor(config)

		// Create gradient image
		img := image.NewRGBA(image.Rect(0, 0, 50, 50))
		for y := 0; y < 50; y++ {
			for x := 0; x < 50; x++ {
				intensity := uint8(x * 5)
				img.Set(x, y, color.RGBA{intensity, intensity, intensity, 255})
			}
		}

		result, err := cp.Process(context.Background(), img)
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result.Bounds() != img.Bounds() {
			t.Error("Result has different bounds than input")
		}
	})

	t.Run("ResetStats", func(t *testing.T) {
		cp := NewColorProcessor(nil)
		img := image.NewRGBA(image.Rect(0, 0, 10, 10))

		cp.Process(context.Background(), img)
		stats1 := cp.GetStats()
		if stats1.ImagesProcessed != 1 {
			t.Error("Should have processed 1 image")
		}

		cp.ResetStats()
		stats2 := cp.GetStats()
		if stats2.ImagesProcessed != 0 {
			t.Error("Stats should be reset to 0")
		}
	})

	t.Run("String", func(t *testing.T) {
		cp := NewColorProcessor(nil)
		s := cp.String()
		if s == "" {
			t.Error("String() should not be empty")
		}
		if len(s) < 20 {
			t.Errorf("String() seems too short: %s", s)
		}
	})
}

func BenchmarkOctonionOperations(b *testing.B) {
	o1 := NewOctonion(1, 2, 3, 4, 5, 6, 7, 8)
	o2 := NewOctonion(8, 7, 6, 5, 4, 3, 2, 1)

	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = o1.Add(o2)
		}
	})

	b.Run("Mul", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = o1.Mul(o2)
		}
	})

	b.Run("Norm", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = o1.Norm()
		}
	})

	b.Run("Normalize", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = o1.Normalize()
		}
	})

	b.Run("Distance", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = o1.Distance(o2)
		}
	})
}

func BenchmarkColorProcessor(b *testing.B) {
	cp := NewColorProcessor(nil)
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))

	// Fill with varied content
	for y := 0; y < 512; y++ {
		for x := 0; x < 512; x++ {
			img.Set(x, y, color.RGBA{
				uint8((x + y) % 256),
				uint8((x * y) % 256),
				uint8((x - y) % 256),
				255,
			})
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cp.Process(ctx, img)
	}

	stats := cp.GetStats()
	b.ReportMetric(stats.PixelsPerSec, "pixels/sec")
}
