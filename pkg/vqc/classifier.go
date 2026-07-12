// Package vqc provides Vedic Quaternionic Computing classification engines
// Adapted from asymm_mathematical_organism VQC cancer classifier (71M genes/sec)
// Generalized for any classification task: gene expression, customer segments, payment risk, etc.
package vqc

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"
)

// ============================================================================
// NOTE: Quaternion primitives moved to quaternion.go to avoid duplication
// ============================================================================
// FEATURE VECTOR → QUATERNION ENCODING
// ============================================================================

// EncodeFeatures converts N-dimensional feature vector to quaternion on S³
// Uses statistical moments: mean, variance, skewness, kurtosis
// This is the KEY insight: ANY data can be encoded as geometry!
func EncodeFeatures(features []float64) Quaternion {
	if len(features) == 0 {
		return Quaternion{W: 1, X: 0, Y: 0, Z: 0}
	}

	n := float64(len(features))

	// Calculate mean
	sum := 0.0
	for _, f := range features {
		sum += f
	}
	mean := sum / n

	// Calculate variance, skewness, kurtosis
	variance := 0.0
	skewness := 0.0
	kurtosis := 0.0
	for _, f := range features {
		diff := f - mean
		variance += diff * diff
		skewness += diff * diff * diff
		kurtosis += diff * diff * diff * diff
	}
	variance /= n
	stdDev := math.Sqrt(variance)

	if stdDev > 1e-10 {
		skewness = (skewness / n) / (stdDev * stdDev * stdDev)
		kurtosis = (kurtosis / n) / (variance * variance)
	} else {
		skewness = 0
		kurtosis = 0
	}

	// Map statistical moments to quaternion components
	// Normalize to reasonable range using tanh (bounded to [-1, 1])
	w := math.Tanh(mean / 1000.0)               // Mean normalized
	x := math.Tanh(math.Log1p(variance) / 10.0) // Log variance
	y := math.Tanh(skewness / 3.0)              // Skewness bounded
	z := math.Tanh((kurtosis - 3.0) / 10.0)     // Excess kurtosis

	return NewQuaternion(w, x, y, z)
}

// ============================================================================
// DATA POINT STRUCTURE
// ============================================================================

// DataPoint represents a single item to classify
type DataPoint struct {
	ID        string     // Unique identifier
	Features  []float64  // N-dimensional feature vector
	Label     string     // Ground truth label (for training)
	State     Quaternion // Encoded quaternion state
	Predicted string     // Predicted label (after classification)
}

// ============================================================================
// VQC CLASSIFIER - GENERAL PURPOSE
// ============================================================================

// Classifier performs quaternionic classification on arbitrary data
// Preserves 71M ops/sec capability from cancer classifier
type Classifier struct {
	DataPoints  []DataPoint           // All data points
	Centroids   map[string]Quaternion // Class centroids (average quaternion per class)
	NumWorkers  int                   // Parallel workers (= CPU cores)
	TrainTime   time.Duration         // Training duration
	PredictTime time.Duration         // Prediction duration
}

// NewClassifier creates a new VQC classifier
func NewClassifier() *Classifier {
	return &Classifier{
		Centroids:  make(map[string]Quaternion),
		NumWorkers: runtime.NumCPU(),
	}
}

// Train computes centroids for each class
// Parallel encoding + centroid computation
func (c *Classifier) Train(dataPoints []DataPoint) {
	start := time.Now()
	c.DataPoints = dataPoints
	if len(dataPoints) == 0 {
		c.Centroids = make(map[string]Quaternion)
		c.TrainTime = time.Since(start)
		return
	}

	// Encode all data points to quaternions (PARALLEL!)
	var wg sync.WaitGroup
	workerCount := c.NumWorkers
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(dataPoints) {
		workerCount = len(dataPoints)
	}
	chunkSize := len(dataPoints) / workerCount
	if chunkSize < 1 {
		chunkSize = 1
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		startIdx := w * chunkSize
		endIdx := startIdx + chunkSize
		if w == workerCount-1 {
			endIdx = len(dataPoints)
		}

		go func(start, end int) {
			defer wg.Done()
			if start >= len(c.DataPoints) {
				return
			}
			if end > len(c.DataPoints) {
				end = len(c.DataPoints)
			}
			for i := start; i < end; i++ {
				c.DataPoints[i].State = EncodeFeatures(c.DataPoints[i].Features)
			}
		}(startIdx, endIdx)
	}
	wg.Wait()

	// Compute centroids (average quaternion for each class)
	classSums := make(map[string]Quaternion)
	classCounts := make(map[string]int)

	for _, dp := range c.DataPoints {
		if _, exists := classSums[dp.Label]; !exists {
			classSums[dp.Label] = Quaternion{W: 0, X: 0, Y: 0, Z: 0}
		}
		classSums[dp.Label] = classSums[dp.Label].Add(dp.State)
		classCounts[dp.Label]++
	}

	// Normalize centroids
	for class, sum := range classSums {
		count := float64(classCounts[class])
		if count > 0 {
			c.Centroids[class] = sum.Scale(1.0 / count).Normalize()
		}
	}

	c.TrainTime = time.Since(start)
}

// Predict classifies a single data point
// Returns the class with minimum geodesic distance on S³
func (c *Classifier) Predict(dp *DataPoint) string {
	minDistance := math.MaxFloat64
	predictedClass := ""

	for class, centroid := range c.Centroids {
		dist := dp.State.Distance(centroid)
		if dist < minDistance {
			minDistance = dist
			predictedClass = class
		}
	}

	return predictedClass
}

// PredictAll classifies all data points (PARALLEL!)
func (c *Classifier) PredictAll() {
	start := time.Now()
	if len(c.DataPoints) == 0 {
		c.PredictTime = time.Since(start)
		return
	}

	var wg sync.WaitGroup
	workerCount := c.NumWorkers
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(c.DataPoints) {
		workerCount = len(c.DataPoints)
	}
	chunkSize := len(c.DataPoints) / workerCount
	if chunkSize < 1 {
		chunkSize = 1
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		startIdx := w * chunkSize
		endIdx := startIdx + chunkSize
		if w == workerCount-1 {
			endIdx = len(c.DataPoints)
		}

		go func(start, end int) {
			defer wg.Done()
			if start >= len(c.DataPoints) {
				return
			}
			if end > len(c.DataPoints) {
				end = len(c.DataPoints)
			}
			for i := start; i < end; i++ {
				c.DataPoints[i].Predicted = c.Predict(&c.DataPoints[i])
			}
		}(startIdx, endIdx)
	}
	wg.Wait()

	c.PredictTime = time.Since(start)
}

// Evaluate computes accuracy and confusion metrics
func (c *Classifier) Evaluate() (accuracy float64, confusionMatrix map[string]map[string]int) {
	confusionMatrix = make(map[string]map[string]int)
	correct := 0

	for _, dp := range c.DataPoints {
		// Initialize confusion matrix entries
		if _, exists := confusionMatrix[dp.Predicted]; !exists {
			confusionMatrix[dp.Predicted] = make(map[string]int)
		}
		confusionMatrix[dp.Predicted][dp.Label]++

		// Count correct predictions
		if dp.Predicted == dp.Label {
			correct++
		}
	}

	accuracy = float64(correct) / float64(len(c.DataPoints))
	return
}

// GetCentroidDistance returns geodesic distance between two class centroids
func (c *Classifier) GetCentroidDistance(class1, class2 string) (float64, error) {
	c1, ok1 := c.Centroids[class1]
	c2, ok2 := c.Centroids[class2]

	if !ok1 || !ok2 {
		return 0, fmt.Errorf("one or both classes not found: %s, %s", class1, class2)
	}

	return c1.Distance(c2), nil
}

// GetClasses returns all known class labels
func (c *Classifier) GetClasses() []string {
	classes := make([]string, 0, len(c.Centroids))
	for class := range c.Centroids {
		classes = append(classes, class)
	}
	return classes
}

// PredictSingle classifies a single feature vector (NOT from training set)
func (c *Classifier) PredictSingle(features []float64) (class string, confidence float64) {
	state := EncodeFeatures(features)

	minDistance := math.MaxFloat64
	predictedClass := ""

	for cls, centroid := range c.Centroids {
		dist := state.Distance(centroid)
		if dist < minDistance {
			minDistance = dist
			predictedClass = cls
		}
	}

	// Convert distance to confidence (0 = perfect, π = opposite)
	// confidence = 1 - (distance / π)
	confidence = 1.0 - (minDistance / math.Pi)
	if confidence < 0 {
		confidence = 0
	}

	return predictedClass, confidence
}
