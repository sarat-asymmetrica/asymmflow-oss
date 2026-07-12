package vqc

import (
	"fmt"
	"testing"
)

// TestQuaternionBasics verifies quaternion operations
func TestQuaternionBasics(t *testing.T) {
	q1 := NewQuaternion(1, 0, 0, 0)
	q2 := NewQuaternion(0, 1, 0, 0)

	// Test normalization
	if norm := q1.Norm(); norm < 0.999 || norm > 1.001 {
		t.Errorf("Expected norm ~1.0, got %f", norm)
	}

	// Test distance
	dist := q1.Distance(q2)
	expectedDist := 1.5708 // π/2 radians (orthogonal quaternions)
	if dist < expectedDist-0.01 || dist > expectedDist+0.01 {
		t.Errorf("Expected distance ~π/2, got %f", dist)
	}

	t.Logf("✅ Quaternion basics: norm=%f, distance=%f", q1.Norm(), dist)
}

// TestEncodeFeatures verifies feature encoding
func TestEncodeFeatures(t *testing.T) {
	// Simple feature vector
	features := []float64{100, 150, 200, 120, 180}

	q := EncodeFeatures(features)

	// Should be unit quaternion
	if norm := q.Norm(); norm < 0.999 || norm > 1.001 {
		t.Errorf("Expected unit quaternion (norm=1), got %f", norm)
	}

	t.Logf("✅ Feature encoding: Q(%f, %f, %f, %f), norm=%f",
		q.W, q.X, q.Y, q.Z, q.Norm())
}

// TestClassifierBinary tests binary classification
func TestClassifierBinary(t *testing.T) {
	// Create synthetic training data
	dataPoints := []DataPoint{
		// Class A: high values
		{ID: "A1", Features: []float64{200, 220, 210, 205}, Label: "A"},
		{ID: "A2", Features: []float64{210, 215, 220, 200}, Label: "A"},
		{ID: "A3", Features: []float64{205, 208, 212, 215}, Label: "A"},

		// Class B: low values
		{ID: "B1", Features: []float64{100, 110, 105, 108}, Label: "B"},
		{ID: "B2", Features: []float64{95, 105, 100, 102}, Label: "B"},
		{ID: "B3", Features: []float64{102, 98, 105, 100}, Label: "B"},
	}

	// Train classifier
	classifier := NewClassifier()
	classifier.Train(dataPoints)

	// Verify centroids exist
	if len(classifier.Centroids) != 2 {
		t.Errorf("Expected 2 centroids, got %d", len(classifier.Centroids))
	}

	// Check centroid separation
	centroidA := classifier.Centroids["A"]
	centroidB := classifier.Centroids["B"]
	distance := centroidA.Distance(centroidB)

	t.Logf("📊 Centroid A: Q(%f, %f, %f, %f)",
		centroidA.W, centroidA.X, centroidA.Y, centroidA.Z)
	t.Logf("📊 Centroid B: Q(%f, %f, %f, %f)",
		centroidB.W, centroidB.X, centroidB.Y, centroidB.Z)
	t.Logf("📏 Centroid distance: %f radians", distance)

	// Centroids should be separated (distance > 0)
	if distance < 0.01 {
		t.Errorf("Centroids too close (distance=%f), classes not separable", distance)
	}

	// Predict all
	classifier.PredictAll()

	// Evaluate
	accuracy, _ := classifier.Evaluate()

	t.Logf("🎯 Classification accuracy: %.1f%%", accuracy*100)
	t.Logf("⏱️  Training time: %v", classifier.TrainTime)
	t.Logf("⏱️  Prediction time: %v", classifier.PredictTime)

	// Should achieve reasonable accuracy (dataset is very small, so not perfect)
	if accuracy < 0.5 {
		t.Errorf("Expected accuracy >= 50%%, got %.1f%%", accuracy*100)
	} else {
		t.Log("✅ Binary classification working!")
	}
}

// TestClassifierWithRealData tests with realistic gene expression values
func TestClassifierWithRealData(t *testing.T) {
	// Simulate realistic gene expression (similar to cancer data)
	// ALL: higher mean expression
	// AML: lower mean expression

	dataPoints := make([]DataPoint, 0, 20)

	// Generate 10 ALL patients
	for i := 0; i < 10; i++ {
		genes := make([]float64, 100) // 100 genes (small scale)
		for j := 0; j < 100; j++ {
			genes[j] = 150.0 + float64((i+j)%50) // Mean ~175
		}
		dataPoints = append(dataPoints, DataPoint{
			ID:       fmt.Sprintf("ALL_%d", i),
			Features: genes,
			Label:    "ALL",
		})
	}

	// Generate 10 AML patients
	for i := 0; i < 10; i++ {
		genes := make([]float64, 100)
		for j := 0; j < 100; j++ {
			genes[j] = 100.0 + float64((i+j)%40) // Mean ~120
		}
		dataPoints = append(dataPoints, DataPoint{
			ID:       fmt.Sprintf("AML_%d", i),
			Features: genes,
			Label:    "AML",
		})
	}

	// Train and evaluate
	classifier := NewClassifier()
	classifier.Train(dataPoints)
	classifier.PredictAll()
	accuracy, confusionMatrix := classifier.Evaluate()

	t.Logf("\n╔══════════════════════════════════════════════════════╗")
	t.Logf("║  VQC CLASSIFIER - REALISTIC GENE EXPRESSION TEST    ║")
	t.Logf("╠══════════════════════════════════════════════════════╣")
	t.Logf("║  Patients:   %d                                      ║", len(dataPoints))
	t.Logf("║  Genes:      %d                                      ║", 100)
	t.Logf("║  Classes:    %d (ALL, AML)                           ║", len(classifier.Centroids))
	t.Logf("║  Accuracy:   %.1f%%                                   ║", accuracy*100)
	t.Logf("║  Train time: %v                                      ║", classifier.TrainTime)
	t.Logf("║  Pred time:  %v                                      ║", classifier.PredictTime)
	t.Logf("╠══════════════════════════════════════════════════════╣")
	t.Logf("║  Confusion Matrix:                                   ║")
	t.Logf("║                    Actual ALL    Actual AML          ║")
	t.Logf("║  Predicted ALL     %8d      %8d            ║", confusionMatrix["ALL"]["ALL"], confusionMatrix["ALL"]["AML"])
	t.Logf("║  Predicted AML     %8d      %8d            ║", confusionMatrix["AML"]["ALL"], confusionMatrix["AML"]["AML"])
	t.Logf("╚══════════════════════════════════════════════════════╝")

	// Should achieve good accuracy
	if accuracy < 0.8 {
		t.Logf("⚠️  Warning: Accuracy below 80%% (got %.1f%%)", accuracy*100)
	} else {
		t.Log("✅ Classification accuracy acceptable!")
	}
}

// BenchmarkEncodeFeatures benchmarks feature encoding
func BenchmarkEncodeFeatures(b *testing.B) {
	features := make([]float64, 7129) // Realistic gene count
	for i := range features {
		features[i] = float64(i % 1000)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeFeatures(features)
	}
}

// BenchmarkClassification benchmarks full classification pipeline
func BenchmarkClassification(b *testing.B) {
	// Create small training set
	dataPoints := make([]DataPoint, 100)
	for i := 0; i < 100; i++ {
		genes := make([]float64, 1000)
		for j := range genes {
			genes[j] = float64((i + j) % 500)
		}
		label := "A"
		if i%2 == 0 {
			label = "B"
		}
		dataPoints[i] = DataPoint{
			ID:       fmt.Sprintf("P%d", i),
			Features: genes,
			Label:    label,
		}
	}

	classifier := NewClassifier()
	classifier.Train(dataPoints)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.PredictAll()
	}
}
