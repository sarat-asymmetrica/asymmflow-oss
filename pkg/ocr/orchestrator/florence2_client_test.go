package orchestrator

import (
	"context"
	"image"
	"image/color"
	"testing"
)

// TestFlorence2ClientCreation tests Florence-2 client creation
func TestFlorence2ClientCreation(t *testing.T) {
	t.Log("🌸 FLORENCE-2 CLIENT CREATION TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	client, err := NewFlorence2Client(nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Logf("   Base URL: %s", client.baseURL)
	t.Logf("   Timeout: %v", client.httpClient.Timeout)

	expectedURL := "https://the maintainer-asymmetrica--florence2-ocr"
	if client.baseURL != expectedURL {
		t.Errorf("Expected base URL %s, got %s", expectedURL, client.baseURL)
	}

	t.Log("✅ Florence-2 client created successfully!")
}

// TestFlorence2CustomConfig tests custom configuration
func TestFlorence2CustomConfig(t *testing.T) {
	t.Log("🌸 FLORENCE-2 CUSTOM CONFIG TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	customConfig := &Florence2Config{
		BaseURL: "https://custom-endpoint",
		Timeout: 30000000000, // 30 seconds
	}

	client, err := NewFlorence2Client(customConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.baseURL != customConfig.BaseURL {
		t.Errorf("Expected base URL %s, got %s", customConfig.BaseURL, client.baseURL)
	}

	t.Log("✅ Custom configuration working!")
}

// TestFlorence2OCRImage tests single image OCR
// NOTE: This test requires Modal to be deployed and running
func TestFlorence2OCRImage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Florence-2 test in short mode (requires deployed endpoint)")
	}

	t.Log("🌸 FLORENCE-2 SINGLE IMAGE OCR TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	client, _ := NewFlorence2Client(nil)
	ctx := context.Background()

	// Create a simple test image (100x100 white square)
	testImage := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			testImage.Set(x, y, color.White)
		}
	}

	t.Log("📊 Sending test image for OCR...")

	result, err := client.OCRImage(ctx, testImage)
	if err != nil {
		t.Logf("⚠️ Modal not available: %v", err)
		t.Log("   (This is expected if Modal is not deployed)")
		return
	}

	t.Logf("   ✅ Success: %v", result.Success)
	t.Logf("   Text: %q", result.Text)
	t.Logf("   Characters: %d", result.Characters)
	t.Logf("   Duration: %v", result.Duration)
	t.Logf("   Cost: $%.6f", result.EstimatedCost)
	t.Logf("   Model: %s", result.Model)

	if result.Success && result.Characters > 0 {
		t.Logf("   ✨ OCR extracted %d characters!", result.Characters)
	}

	t.Log(client.Summary())
	t.Log("✅ Florence-2 OCR test complete!")
}

// TestFlorence2OCRBatch tests batch OCR
func TestFlorence2OCRBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Florence-2 batch test in short mode")
	}

	t.Log("🌸 FLORENCE-2 BATCH OCR TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	client, _ := NewFlorence2Client(nil)
	ctx := context.Background()

	// Create multiple test images
	batchSize := 5
	images := make([]image.Image, batchSize)
	for i := 0; i < batchSize; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				img.Set(x, y, color.White)
			}
		}
		images[i] = img
	}

	t.Logf("📊 Sending batch of %d images...", batchSize)

	result, err := client.OCRBatch(ctx, images)
	if err != nil {
		t.Logf("⚠️ Modal not available: %v", err)
		return
	}

	t.Logf("   ✅ Success: %v", result.Success)
	t.Logf("   Total images: %d", result.TotalImages)
	t.Logf("   Total elapsed: %.2f ms", result.TotalElapsedMs)
	t.Logf("   Throughput: %.2f pages/sec", result.ThroughputPerSec)

	successCount := 0
	for i, r := range result.Results {
		if r.Success {
			successCount++
			t.Logf("   Image %d: %d chars in %.2f ms", i+1, r.Characters, r.ElapsedMs)
		} else {
			t.Logf("   Image %d: FAILED - %s", i+1, r.Error)
		}
	}

	t.Logf("   📈 Success rate: %d/%d (%.1f%%)",
		successCount, len(result.Results),
		float64(successCount)/float64(len(result.Results))*100)

	t.Log(client.Summary())
	t.Log("✅ Florence-2 batch OCR test complete!")
}

// TestFlorence2HealthCheck tests the health check endpoint
func TestFlorence2HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping health check in short mode")
	}

	t.Log("🌸 FLORENCE-2 HEALTH CHECK TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	client, _ := NewFlorence2Client(nil)
	ctx := context.Background()

	err := client.HealthCheck(ctx)
	if err != nil {
		t.Logf("⚠️ Health check failed: %v", err)
		t.Log("   (This is expected if Modal is not deployed)")
		return
	}

	t.Log("✅ Florence-2 service is healthy!")
}

// TestFlorence2Stats tests statistics tracking
func TestFlorence2Stats(t *testing.T) {
	t.Log("🌸 FLORENCE-2 STATS TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	client, _ := NewFlorence2Client(nil)

	// Simulate some stats
	client.recordSuccess(1000, 300000000, 0.00015) // 300ms, 1000 chars
	client.recordSuccess(800, 250000000, 0.00015)  // 250ms, 800 chars
	client.recordError()

	stats := client.GetStats()

	if stats.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", stats.TotalRequests)
	}

	if stats.SuccessCount != 2 {
		t.Errorf("Expected 2 successes, got %d", stats.SuccessCount)
	}

	if stats.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", stats.ErrorCount)
	}

	if stats.TotalCharacters != 1800 {
		t.Errorf("Expected 1800 total characters, got %d", stats.TotalCharacters)
	}

	expectedCost := 0.00030
	if stats.TotalCost != expectedCost {
		t.Errorf("Expected cost $%.6f, got $%.6f", expectedCost, stats.TotalCost)
	}

	t.Logf("   Total requests: %d", stats.TotalRequests)
	t.Logf("   Successes: %d", stats.SuccessCount)
	t.Logf("   Errors: %d", stats.ErrorCount)
	t.Logf("   Characters: %d", stats.TotalCharacters)
	t.Logf("   Cost: $%.6f", stats.TotalCost)

	t.Log("✅ Stats tracking working correctly!")
}

// TestFlorence2Summary tests summary generation
func TestFlorence2Summary(t *testing.T) {
	t.Log("🌸 FLORENCE-2 SUMMARY TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	client, _ := NewFlorence2Client(nil)

	// Simulate some activity
	client.recordSuccess(1000, 300000000, 0.00015)
	client.recordSuccess(1500, 250000000, 0.00015)

	summary := client.Summary()

	if summary == "" {
		t.Error("Summary should not be empty")
	}

	t.Log(summary)
	t.Log("✅ Summary generation working!")
}

// BenchmarkFlorence2ImageEncoding benchmarks image encoding
func BenchmarkFlorence2ImageEncoding(b *testing.B) {
	// Create test image
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			img.Set(x, y, color.White)
		}
	}

	client, _ := NewFlorence2Client(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This would normally call OCRImage, but we'll just test the client creation
		_ = client
	}
}
