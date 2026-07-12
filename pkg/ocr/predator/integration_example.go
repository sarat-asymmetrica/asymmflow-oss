package predator

import (
	"context"
	"fmt"
	"image"
	"time"
)

// Example demonstrates how to integrate Predator Vision with ACE Engine OCR

// ProcessWithPredatorVision shows basic integration
func ProcessWithPredatorVision(ctx context.Context, img image.Image) (image.Image, error) {
	// Create processor with production defaults
	pv := NewPredatorVision(nil)

	// Process image
	result, err := pv.Process(ctx, img)
	if err != nil {
		return nil, fmt.Errorf("predator vision failed: %w", err)
	}

	// Log results
	fmt.Printf("Predator Vision:\n")
	fmt.Printf("  Processing time: %.2f ms\n", result.ProcessingMs)
	fmt.Printf("  Skew detected: %.2f degrees\n", result.SkewAngle)
	fmt.Printf("  Focus regions: %d\n", len(result.FocusRegions))
	fmt.Printf("  Saliency samples: %d\n", len(result.SaliencyMap))

	return result.Image, nil
}

// ProcessWithCustomConfig shows advanced configuration
func ProcessWithCustomConfig(ctx context.Context, img image.Image, mode string) (image.Image, error) {
	var config *PredatorConfig

	switch mode {
	case "fast":
		// Minimal processing for speed
		config = &PredatorConfig{
			EnableUVChannel:     true,
			EnableSaliency:      false,
			EnableOpticalFlow:   false,
			EnableAdaptiveFocus: false,
			UVBoostFactor:       1.3,
		}

	case "quality":
		// Maximum quality for difficult documents
		config = &PredatorConfig{
			EnableUVChannel:     true,
			EnableSaliency:      true,
			EnableOpticalFlow:   true,
			EnableAdaptiveFocus: true,
			UVBoostFactor:       2.0,
			SaliencyThreshold:   0.2, // Lower threshold = more regions
		}

	case "skew-only":
		// Just fix rotation
		config = &PredatorConfig{
			EnableUVChannel:     false,
			EnableSaliency:      false,
			EnableOpticalFlow:   true,
			EnableAdaptiveFocus: false,
		}

	default:
		config = DefaultPredatorConfig()
	}

	pv := NewPredatorVision(config)
	result, err := pv.Process(ctx, img)
	if err != nil {
		return nil, err
	}

	return result.Image, nil
}

// BatchProcessWithStats demonstrates batch processing with statistics
func BatchProcessWithStats(ctx context.Context, images []image.Image) ([]image.Image, error) {
	pv := NewPredatorVision(nil)
	results := make([]image.Image, 0, len(images))

	startTime := time.Now()

	for i, img := range images {
		result, err := pv.Process(ctx, img)
		if err != nil {
			return nil, fmt.Errorf("failed to process image %d: %w", i, err)
		}
		results = append(results, result.Image)
	}

	duration := time.Since(startTime)
	stats := pv.GetStats()

	// Print batch statistics
	fmt.Printf("\nBatch Processing Statistics:\n")
	fmt.Printf("  Images processed: %d\n", stats.ImagesProcessed)
	fmt.Printf("  Total pixels: %d\n", stats.TotalPixels)
	fmt.Printf("  Skew corrected: %d (%.1f%%)\n",
		stats.SkewCorrected,
		100.0*float64(stats.SkewCorrected)/float64(stats.ImagesProcessed))
	fmt.Printf("  Total duration: %v\n", duration)
	fmt.Printf("  Avg per image: %v\n", duration/time.Duration(len(images)))
	fmt.Printf("  Throughput: %.1f images/sec\n",
		float64(len(images))/duration.Seconds())

	return results, nil
}

// ProcessWithFallback demonstrates graceful degradation
func ProcessWithFallback(ctx context.Context, img image.Image, timeout time.Duration) (image.Image, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pv := NewPredatorVision(nil)

	// Channel for result
	type result struct {
		img image.Image
		err error
	}
	resultChan := make(chan result, 1)

	// Process in goroutine
	go func() {
		processed, err := pv.Process(ctx, img)
		if err != nil {
			resultChan <- result{nil, err}
			return
		}
		resultChan <- result{processed.Image, nil}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultChan:
		if res.err != nil {
			// Fall back to original image
			fmt.Printf("Warning: Predator vision failed, using original: %v\n", res.err)
			return img, nil
		}
		return res.img, nil

	case <-ctx.Done():
		// Timeout - use original image
		fmt.Printf("Warning: Predator vision timeout, using original\n")
		return img, nil
	}
}

// ProcessWithMetrics demonstrates metrics collection
func ProcessWithMetrics(ctx context.Context, img image.Image, metrics MetricsCollector) (image.Image, error) {
	startTime := time.Now()

	pv := NewPredatorVision(nil)
	result, err := pv.Process(ctx, img)

	// Collect metrics
	metrics.RecordDuration("predator_vision_process", time.Since(startTime))

	if err != nil {
		metrics.IncrementCounter("predator_vision_errors")
		return nil, err
	}

	metrics.IncrementCounter("predator_vision_success")
	metrics.RecordGauge("predator_vision_skew_angle", result.SkewAngle)
	metrics.RecordGauge("predator_vision_focus_regions", float64(len(result.FocusRegions)))

	if len(result.SaliencyMap) > 0 {
		// Calculate average saliency
		var avgSaliency float64
		for _, s := range result.SaliencyMap {
			avgSaliency += s
		}
		avgSaliency /= float64(len(result.SaliencyMap))
		metrics.RecordGauge("predator_vision_avg_saliency", avgSaliency)
	}

	return result.Image, nil
}

// MetricsCollector interface for metrics integration
type MetricsCollector interface {
	IncrementCounter(name string)
	RecordDuration(name string, duration time.Duration)
	RecordGauge(name string, value float64)
}

// AdaptiveProcessor demonstrates adaptive processing based on image characteristics
type AdaptiveProcessor struct {
	pv *PredatorVision
}

// NewAdaptiveProcessor creates an adaptive processor
func NewAdaptiveProcessor() *AdaptiveProcessor {
	return &AdaptiveProcessor{
		pv: NewPredatorVision(nil),
	}
}

// Process adaptively processes based on image analysis
func (ap *AdaptiveProcessor) Process(ctx context.Context, img image.Image) (image.Image, error) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	pixelCount := width * height

	// Analyze image to determine processing strategy
	var config *PredatorConfig

	if pixelCount > 1000000 { // Large image (>1MP)
		// Fast processing to avoid timeout
		config = &PredatorConfig{
			EnableUVChannel:     true,
			EnableSaliency:      false, // Skip for speed
			EnableOpticalFlow:   true,
			EnableAdaptiveFocus: true,
			UVBoostFactor:       1.5,
		}
	} else if pixelCount < 100000 { // Small image (<0.1MP)
		// Maximum quality for thumbnails/previews
		config = &PredatorConfig{
			EnableUVChannel:     true,
			EnableSaliency:      true,
			EnableOpticalFlow:   true,
			EnableAdaptiveFocus: true,
			UVBoostFactor:       2.0,
			SaliencyThreshold:   0.2,
		}
	} else {
		// Balanced processing
		config = DefaultPredatorConfig()
	}

	pv := NewPredatorVision(config)
	result, err := pv.Process(ctx, img)
	if err != nil {
		return nil, err
	}

	return result.Image, nil
}

// Example OCR Engine Integration
type OCREngine struct {
	pv      *PredatorVision
	enabled bool
}

// NewOCREngine creates an OCR engine with Predator Vision
func NewOCREngine(enablePredatorVision bool) *OCREngine {
	var pv *PredatorVision
	if enablePredatorVision {
		pv = NewPredatorVision(nil)
	}
	return &OCREngine{
		pv:      pv,
		enabled: enablePredatorVision,
	}
}

// Preprocess applies all preprocessing including Predator Vision
func (e *OCREngine) Preprocess(ctx context.Context, img image.Image) (image.Image, error) {
	// Step 1: Predator Vision (if enabled)
	if e.enabled && e.pv != nil {
		result, err := e.pv.Process(ctx, img)
		if err != nil {
			// Log error but continue with original
			fmt.Printf("Warning: Predator Vision failed: %v\n", err)
		} else {
			img = result.Image
		}
	}

	// Step 2: Other preprocessing (quaternion denoise, contrast, etc.)
	// ... additional steps ...

	return img, nil
}

// GetPredatorStats returns Predator Vision statistics
func (e *OCREngine) GetPredatorStats() *PredatorStats {
	if e.pv != nil {
		return e.pv.GetStats()
	}
	return nil
}
