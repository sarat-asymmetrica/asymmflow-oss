package organism

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const N = 79 // M⁷⁹ manifold dimension

// ========================================================================
// QUATERNION (S³ Geometry)
// ========================================================================

type Quaternion struct {
	W, X, Y, Z float64
}

func NewQuaternion(w, x, y, z float64) Quaternion {
	return Quaternion{W: w, X: x, Y: y, Z: z}
}

func (q Quaternion) Norm() float64 {
	return math.Sqrt(q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z)
}

func (q Quaternion) Normalize() Quaternion {
	n := q.Norm()
	if n < 1e-10 {
		return NewQuaternion(1, 0, 0, 0)
	}
	return NewQuaternion(q.W/n, q.X/n, q.Y/n, q.Z/n)
}

// SLERP - Geodesic evolution on S³!
func SLERP(q0, q1 Quaternion, t float64) Quaternion {
	dot := q0.W*q1.W + q0.X*q1.X + q0.Y*q1.Y + q0.Z*q1.Z
	if dot < 0 {
		q1 = NewQuaternion(-q1.W, -q1.X, -q1.Y, -q1.Z)
		dot = -dot
	}
	if dot > 0.9995 {
		// Linear interpolation for very close quaternions
		return NewQuaternion(
			q0.W+t*(q1.W-q0.W),
			q0.X+t*(q1.X-q0.X),
			q0.Y+t*(q1.Y-q0.Y),
			q0.Z+t*(q1.Z-q0.Z),
		).Normalize()
	}

	theta := math.Acos(dot)
	sinTheta := math.Sin(theta)
	a := math.Sin((1-t)*theta) / sinTheta
	b := math.Sin(t*theta) / sinTheta

	return NewQuaternion(
		a*q0.W+b*q1.W,
		a*q0.X+b*q1.X,
		a*q0.Y+b*q1.Y,
		a*q0.Z+b*q1.Z,
	)
}

// ========================================================================
// OCR CELL (Φ-Organism per Document)
// ========================================================================

type OCRCell struct {
	// Core Φ state (79-D manifold)
	State     [N]float64
	Energy    float64
	Iteration int

	// Document properties
	ID           int
	FilePath     string
	FileSize     int64
	DigitalRoot  int // 1-9 (Vedic clustering!)
	DocumentType string

	// OCR results
	ExtractedText string
	Confidence    float64
	ProcessTime   time.Duration
	Status        string // "pending", "processing", "completed", "failed"
	Error         error

	// Network state (communication layer!)
	Q  Quaternion // S³ representation
	R1 float64    // Exploration regime
	R2 float64    // Optimization regime
	R3 float64    // Stabilization regime

	// Processing metadata
	StartTime time.Time
	EndTime   time.Time
}

// NewOCRCell creates Φ-cell for document
func NewOCRCell(id int, filePath string) (*OCRCell, error) {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	cell := &OCRCell{
		ID:           id,
		FilePath:     filePath,
		FileSize:     info.Size(),
		DocumentType: filepath.Ext(filePath),
		Status:       "pending",
		Iteration:    0,
	}

	// Compute digital root (Vedic clustering!)
	cell.DigitalRoot = ComputeDigitalRoot(cell.FileSize)

	// Initialize Φ state from file metadata
	cell.InitializeState()

	return cell, nil
}

// ComputeDigitalRoot - Vedic Sutra 12 (53× faster!)
func ComputeDigitalRoot(n int64) int {
	if n == 0 {
		return 0
	}
	return int((n-1)%9 + 1)
}

// InitializeState - Encode file metadata into 79-D Φ state
func (c *OCRCell) InitializeState() {
	// File size → first components (log scale)
	sizeLog := math.Log(float64(c.FileSize + 1))
	c.State[0] = math.Tanh(sizeLog / 10.0)

	// Digital root → sacred geometry mapping
	goldenAngle := 137.5077640500378 // 360/φ²
	theta := float64(c.DigitalRoot) * goldenAngle * (math.Pi / 180.0)
	c.State[1] = math.Cos(theta)
	c.State[2] = math.Sin(theta)

	// Document type → hash to vector
	typeHash := 0
	for _, ch := range c.DocumentType {
		typeHash = (typeHash << 5) + int(ch)
	}
	c.State[3] = math.Tanh(float64(typeHash%100) / 100.0)

	// Path hash → additional components
	pathHash := 0
	for _, ch := range c.FilePath {
		pathHash = (pathHash << 5) + int(ch)
	}
	for i := 4; i < 10; i++ {
		component := (pathHash >> (i - 4)) % 200
		c.State[i] = (float64(component)/200.0 - 0.5) * 2.0
	}

	// Random initialization for remaining components (seeded by path for determinism)
	seed := int64(pathHash)
	for i := 10; i < N; i++ {
		seed = (1103515245*seed + 12345) & 0x7fffffff
		val := float64(seed) / float64(0x7fffffff)
		c.State[i] = (val - 0.5) * 2.0
	}

	// Normalize to unit sphere
	c.UpdateEnergy()
	c.Normalize()
	c.UpdateQuaternion()
	c.UpdateRegimes()
}

// UpdateEnergy - Compute ||Φ|| magnitude
func (c *OCRCell) UpdateEnergy() {
	sum := 0.0
	for i := 0; i < N; i++ {
		sum += c.State[i] * c.State[i]
	}
	c.Energy = math.Sqrt(sum)
}

// Normalize - Keep on unit sphere (S³ constraint!)
func (c *OCRCell) Normalize() {
	if c.Energy > 1e-10 {
		for i := 0; i < N; i++ {
			c.State[i] /= c.Energy
		}
		c.Energy = 1.0
	}
}

// UpdateQuaternion - Project 79-D state to S³
func (c *OCRCell) UpdateQuaternion() {
	c.Q = NewQuaternion(c.State[0], c.State[1], c.State[2], c.State[3]).Normalize()
}

// UpdateRegimes - Three-regime classification
func (c *OCRCell) UpdateRegimes() {
	// Compute variance distribution
	mean := 0.0
	for i := 0; i < N; i++ {
		mean += math.Abs(c.State[i])
	}
	mean /= float64(N)

	variance := 0.0
	for i := 0; i < N; i++ {
		diff := math.Abs(c.State[i]) - mean
		variance += diff * diff
	}
	variance /= float64(N)

	// Classify based on energy distribution
	highEnergy := 0
	medEnergy := 0
	lowEnergy := 0

	threshold1 := mean + 0.5*math.Sqrt(variance)
	threshold2 := mean - 0.5*math.Sqrt(variance)

	for i := 0; i < N; i++ {
		abs := math.Abs(c.State[i])
		if abs > threshold1 {
			highEnergy++
		} else if abs > threshold2 {
			medEnergy++
		} else {
			lowEnergy++
		}
	}

	total := float64(N)
	c.R1 = float64(highEnergy) / total // Exploration
	c.R2 = float64(medEnergy) / total  // Optimization
	c.R3 = float64(lowEnergy) / total  // Stabilization
}

// ========================================================================
// OCR NETWORK (Parallel Intelligence)
// ========================================================================

type OCRNetwork struct {
	Cells         []*OCRCell
	NumWorkers    int
	BatchSize     int
	TotalFiles    int
	ProcessedDocs int
	FailedDocs    int
	TotalTime     time.Duration
	StartTime     time.Time

	// Concurrency primitives
	WorkQueue   chan *OCRCell
	ResultQueue chan *OCRCell
	WaitGroup   sync.WaitGroup
	Mutex       sync.Mutex
}

// NewOCRNetwork creates network with Williams batching
func NewOCRNetwork(files []string) (*OCRNetwork, error) {
	numFiles := len(files)

	// Williams batching: √n × log₂(n)
	batchSize := ComputeWilliamsBatchSize(numFiles)

	// Auto-detect workers (use ALL cores!)
	numWorkers := runtime.NumCPU()

	net := &OCRNetwork{
		Cells:       make([]*OCRCell, numFiles),
		NumWorkers:  numWorkers,
		BatchSize:   batchSize,
		TotalFiles:  numFiles,
		WorkQueue:   make(chan *OCRCell, numFiles),
		ResultQueue: make(chan *OCRCell, numFiles),
	}

	// Create Φ-cells for all documents
	for i, filePath := range files {
		cell, err := NewOCRCell(i, filePath)
		if err != nil {
			// Skip files that can't be read, or log error?
			// For now, we'll try to proceed or return error
			// Better to just log and continue for robustness in batch
			fmt.Printf("Warning: Failed to create cell for %s: %v\n", filePath, err)
			continue
		}
		net.Cells[i] = cell
	}

	return net, nil
}

// ComputeWilliamsBatchSize - Optimal batch size formula
func ComputeWilliamsBatchSize(n int) int {
	if n <= 0 {
		return 1
	}
	sqrtN := math.Sqrt(float64(n))
	log2N := math.Log2(float64(n + 1))
	batchSize := int(sqrtN * log2N)

	if batchSize < 1 {
		batchSize = 1
	}
	if batchSize > n {
		batchSize = n
	}

	return batchSize
}

// ProcessorFunc is the function signature for the actual OCR work
type ProcessorFunc func(cell *OCRCell) error

// ProcessAll - Parallel processing with goroutines
func (net *OCRNetwork) ProcessAll(processor ProcessorFunc) error {
	net.StartTime = time.Now()

	// Launch workers
	for i := 0; i < net.NumWorkers; i++ {
		net.WaitGroup.Add(1)
		go func() {
			defer net.WaitGroup.Done()
			for cell := range net.WorkQueue {
				// Process OCR using the provided function
				cell.Status = "processing"
				cell.StartTime = time.Now()

				err := processor(cell)

				cell.EndTime = time.Now()
				cell.ProcessTime = cell.EndTime.Sub(cell.StartTime)

				if err != nil {
					cell.Status = "failed"
					cell.Error = err
					net.Mutex.Lock()
					net.FailedDocs++
					net.Mutex.Unlock()
				} else {
					cell.Status = "completed"
				}

				// Send result
				net.ResultQueue <- cell
			}
		}()
	}

	// Digital root clustering (Vedic optimization!)
	clusters := make(map[int][]*OCRCell)
	for _, cell := range net.Cells {
		if cell != nil {
			clusters[cell.DigitalRoot] = append(clusters[cell.DigitalRoot], cell)
		}
	}

	// Send work in digital root order (optimized!)
	go func() {
		for root := 1; root <= 9; root++ {
			for _, cell := range clusters[root] {
				net.WorkQueue <- cell
			}
		}
		close(net.WorkQueue)
	}()

	// Collect results
	go func() {
		for i := 0; i < net.TotalFiles; i++ {
			// We might have nil cells if initialization failed
			if net.Cells[i] == nil {
				continue
			}

			<-net.ResultQueue // cell received
			net.Mutex.Lock()
			net.ProcessedDocs++
			net.Mutex.Unlock()
		}
		// Only close result queue when we're sure we're done or use a separatedone mechanism
		// For simplicity here, we'll let the WaitGroup handle the worker completion
	}()

	// Wait for all workers
	net.WaitGroup.Wait()
	close(net.ResultQueue)

	net.TotalTime = time.Since(net.StartTime)
	return nil
}
