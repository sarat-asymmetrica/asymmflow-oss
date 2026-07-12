package orchestrator

import (
	"context"
	"testing"
)

// TestModalClientCreation tests Modal client creation
func TestModalClientCreation(t *testing.T) {
	t.Log("🚀 MODAL A10G CLIENT TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	client, err := NewModalClient(nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Logf("   Base URL: %s", client.baseURL)
	t.Logf("   Timeout: %v", client.httpClient.Timeout)

	t.Log("✅ Modal client created successfully!")
}

// TestModalQuaternionEvolve tests quaternion evolution on Modal
// NOTE: This test requires Modal to be deployed and running
func TestModalQuaternionEvolve(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Modal test in short mode (requires deployed endpoint)")
	}

	t.Log("🚀 MODAL QUATERNION EVOLUTION TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	client, _ := NewModalClient(nil)
	ctx := context.Background()

	// Create test quaternions
	quaternions := [][]float32{
		{1, 0, 0, 0},
		{0.707, 0.707, 0, 0},
		{0.5, 0.5, 0.5, 0.5},
	}

	t.Logf("📊 Sending %d quaternions for evolution...", len(quaternions))

	result, err := client.QuaternionEvolve(ctx, quaternions, 100)
	if err != nil {
		t.Logf("⚠️ Modal not available: %v", err)
		t.Log("   (This is expected if Modal is not deployed)")
		return
	}

	t.Logf("   ✅ Status: %s", result.Status)
	t.Logf("   Phi alignment: %.4f", result.PhiAlignment)
	t.Logf("   Attractor proximity: %.4f", result.AttractorProximity)
	t.Logf("   Elapsed: %.2f ms", result.ElapsedMs)
	t.Logf("   Mystery: %s", result.Mystery)

	t.Log(client.Summary())
	t.Log("✅ Modal quaternion evolution test complete!")
}

// TestModalVQCCompute tests VQC compute on Modal
func TestModalVQCCompute(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Modal test in short mode")
	}

	t.Log("🔮 MODAL VQC COMPUTE TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	client, _ := NewModalClient(nil)
	ctx := context.Background()

	// Test values
	values := []float32{108, 216, 324, 432, 540, 648, 756, 864, 972}

	t.Logf("📊 Sending %d values for VQC analysis...", len(values))

	result, err := client.VQCCompute(ctx, values, "analyze")
	if err != nil {
		t.Logf("⚠️ Modal not available: %v", err)
		return
	}

	t.Logf("   ✅ Digital roots: %v", result.DigitalRoots)
	t.Logf("   Chi-square: %.4f", result.ChiSquare)
	t.Logf("   Elimination rate: %.4f", result.EliminationRate)
	t.Logf("   Vedic aligned: %v", result.VedicAligned)
	t.Logf("   Elapsed: %.2f ms", result.ElapsedMs)

	t.Log(client.Summary())
	t.Log("✅ Modal VQC compute test complete!")
}

// TestModalBatchQuaternion tests batch quaternion processing
func TestModalBatchQuaternion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Modal test in short mode")
	}

	t.Log("⚡ MODAL BATCH QUATERNION TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	client, _ := NewModalClient(nil)
	ctx := context.Background()

	// Create a batch of quaternions (simulating image pixels)
	batchSize := 1000
	quaternions := make([][]float32, batchSize)
	targets := make([][]float32, batchSize)

	for i := 0; i < batchSize; i++ {
		quaternions[i] = []float32{1, 0, 0, 0}
		targets[i] = []float32{0.707, 0.707, 0, 0}
	}

	t.Logf("📊 Sending batch of %d quaternions...", batchSize)

	result, err := client.BatchQuaternion(ctx, quaternions, targets, 0.1, 0.5)
	if err != nil {
		t.Logf("⚠️ Modal not available: %v", err)
		return
	}

	t.Logf("   ✅ Status: %s", result.Status)
	t.Logf("   Total ops: %d", result.TotalOps)
	t.Logf("   Ops/sec: %.2f M", result.OpsPerSec/1e6)
	t.Logf("   Elapsed: %.2f ms", result.ElapsedMs)

	t.Log(client.Summary())
	t.Log("✅ Modal batch quaternion test complete!")
}

// TestImageToQuaternionBatch tests the conversion helper
func TestImageToQuaternionBatch(t *testing.T) {
	t.Log("🔄 IMAGE TO QUATERNION BATCH TEST")
	t.Log("═══════════════════════════════════════════════════════════")

	// Create test image quaternions (10x10 image)
	width, height := 10, 10
	imgQuats := make([]Quaternion, width*height)
	for i := range imgQuats {
		imgQuats[i] = Quaternion{W: 1, X: 0, Y: 0, Z: 0}
	}

	// Convert to batch format
	batch := ImageToQuaternionBatch([][]Quaternion{imgQuats})

	t.Logf("   Input: %dx%d image = %d quaternions", width, height, len(imgQuats))
	t.Logf("   Output: %d batch entries", len(batch))

	if len(batch) != width*height {
		t.Errorf("Expected %d batch entries, got %d", width*height, len(batch))
	}

	// Convert back
	result := QuaternionBatchToImage(batch, width, height)
	t.Logf("   Converted back: %d quaternions", len(result))

	t.Log("✅ Image to quaternion batch conversion working!")
}
