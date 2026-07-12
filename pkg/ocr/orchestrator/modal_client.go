// Modal A10G Client - Heavy-duty cloud GPU compute
//
// Connects to Modal endpoints for:
// - Quaternion evolution (82B ops/sec on A10G!)
// - Batch OCR processing
// - VQC pattern detection
//
// Endpoints (from modal_benchmark/asymmetrica_api.py):
// - /quaternion_evolve - Quaternion evolution on S³
// - /sat_solve - SAT solving via quaternion origami
// - /vqc_compute - Vedic pattern recognition
// - /batch_quaternion - Batch quaternion processing
package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ModalClient handles communication with Modal A10G endpoints
type ModalClient struct {
	baseURL    string
	httpClient *http.Client
	stats      *ModalStats
	mu         sync.RWMutex
}

// ModalConfig configures the Modal client
type ModalConfig struct {
	BaseURL string
	Timeout time.Duration
}

// ModalStats tracks Modal usage statistics
type ModalStats struct {
	TotalRequests int
	SuccessCount  int
	ErrorCount    int
	TotalOps      int64
	TotalDuration time.Duration
	EstimatedCost float64 // A10G: ~$1.10/hr
}

// QuaternionEvolveRequest represents a quaternion evolution request
type QuaternionEvolveRequest struct {
	Quaternions [][]float32 `json:"quaternions"`
	Iterations  int         `json:"iterations"`
}

// QuaternionEvolveResponse represents the response
type QuaternionEvolveResponse struct {
	Status             string      `json:"status"`
	Quaternions        [][]float32 `json:"quaternions"`
	Iterations         int         `json:"iterations"`
	PhiAlignment       float64     `json:"phi_alignment"`
	AttractorProximity float64     `json:"attractor_proximity"`
	ElapsedMs          float64     `json:"elapsed_ms"`
	Mystery            string      `json:"mystery"`
}

// VQCComputeRequest represents a VQC compute request
type VQCComputeRequest struct {
	Values    []float32 `json:"values"`
	Operation string    `json:"operation"`
}

// VQCComputeResponse represents the response
type VQCComputeResponse struct {
	DigitalRoots    []int   `json:"digital_roots"`
	Distribution    []int   `json:"distribution"`
	ChiSquare       float64 `json:"chi_square"`
	EliminationRate float64 `json:"elimination_rate"`
	VedicAligned    bool    `json:"vedic_aligned"`
	PhiAlignment    float64 `json:"phi_alignment"`
	ElapsedMs       float64 `json:"elapsed_ms"`
}

// BatchQuaternionRequest represents a batch quaternion request
type BatchQuaternionRequest struct {
	Quaternions [][]float32 `json:"quaternions"`
	Targets     [][]float32 `json:"targets"`
	DT          float32     `json:"dt"`
	Strength    float32     `json:"strength"`
}

// BatchQuaternionResponse represents the response
type BatchQuaternionResponse struct {
	Status    string      `json:"status"`
	Results   [][]float32 `json:"results"`
	TotalOps  int64       `json:"total_ops"`
	OpsPerSec float64     `json:"ops_per_sec"`
	ElapsedMs float64     `json:"elapsed_ms"`
}

// DefaultModalConfig returns production defaults
func DefaultModalConfig() *ModalConfig {
	return &ModalConfig{
		// Modal endpoint - LIVE on the maintainer-asymmetrica
		// URL format: https://{workspace}--{app}-{function}.modal.run
		// App: ocr-gpu (deployed Dec 21, 2025)
		// Endpoints: health, preprocess-batch, quaternion-evolve, batch-benchmark
		BaseURL: "https://the maintainer-asymmetrica--ocr-gpu",
		Timeout: 60 * time.Second,
	}
}

// NewModalClient creates a new Modal client
func NewModalClient(config *ModalConfig) (*ModalClient, error) {
	if config == nil {
		config = DefaultModalConfig()
	}

	return &ModalClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		stats: &ModalStats{},
	}, nil
}

// QuaternionEvolve evolves quaternions on S³ using Modal A10G
func (c *ModalClient) QuaternionEvolve(ctx context.Context, quaternions [][]float32, iterations int) (*QuaternionEvolveResponse, error) {
	start := time.Now()

	req := QuaternionEvolveRequest{
		Quaternions: quaternions,
		Iterations:  iterations,
	}

	resp, err := c.post(ctx, "/quaternion_evolve", req)
	if err != nil {
		c.recordError()
		return nil, err
	}

	var result QuaternionEvolveResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	c.recordSuccess(int64(len(quaternions)*iterations*4), time.Since(start))

	return &result, nil
}

// VQCCompute performs Vedic pattern recognition on Modal
func (c *ModalClient) VQCCompute(ctx context.Context, values []float32, operation string) (*VQCComputeResponse, error) {
	start := time.Now()

	req := VQCComputeRequest{
		Values:    values,
		Operation: operation,
	}

	resp, err := c.post(ctx, "/vqc_compute", req)
	if err != nil {
		c.recordError()
		return nil, err
	}

	var result VQCComputeResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	c.recordSuccess(int64(len(values)), time.Since(start))

	return &result, nil
}

// BatchQuaternion processes a batch of quaternions on Modal A10G
func (c *ModalClient) BatchQuaternion(ctx context.Context, quaternions, targets [][]float32, dt, strength float32) (*BatchQuaternionResponse, error) {
	start := time.Now()

	req := BatchQuaternionRequest{
		Quaternions: quaternions,
		Targets:     targets,
		DT:          dt,
		Strength:    strength,
	}

	resp, err := c.post(ctx, "/batch_quaternion", req)
	if err != nil {
		c.recordError()
		return nil, err
	}

	var result BatchQuaternionResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	c.recordSuccess(result.TotalOps, time.Since(start))

	return &result, nil
}

// ProcessImageBatchOnGPU sends images to Modal for GPU preprocessing
// This is for heavy batches that exceed local GPU capacity
func (c *ModalClient) ProcessImageBatchOnGPU(ctx context.Context, imageQuaternions [][]float32) ([][]float32, error) {
	start := time.Now()

	// Create targets (neighborhood averages would be computed on Modal)
	targets := make([][]float32, len(imageQuaternions))
	for i := range targets {
		targets[i] = imageQuaternions[i] // Self-target for now
	}

	result, err := c.BatchQuaternion(ctx, imageQuaternions, targets, 0.1, 0.5)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.stats.TotalDuration += time.Since(start)
	c.mu.Unlock()

	return result.Results, nil
}

// post sends a POST request to Modal
func (c *ModalClient) post(ctx context.Context, endpoint string, body any) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Modal URL format: https://{workspace}--{app}-{function}.modal.run
	// endpoint comes as "/quaternion_evolve" -> need "quaternion-evolve"
	funcName := strings.TrimPrefix(endpoint, "/")
	funcName = strings.ReplaceAll(funcName, "_", "-")
	url := c.baseURL + "-" + funcName + ".modal.run"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// recordSuccess records a successful request
func (c *ModalClient) recordSuccess(ops int64, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.TotalRequests++
	c.stats.SuccessCount++
	c.stats.TotalOps += ops
	c.stats.TotalDuration += duration

	// Estimate cost: A10G is ~$1.10/hr = $0.000306/sec
	c.stats.EstimatedCost += duration.Seconds() * 0.000306
}

// recordError records a failed request
func (c *ModalClient) recordError() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.TotalRequests++
	c.stats.ErrorCount++
}

// GetStats returns Modal usage statistics
func (c *ModalClient) GetStats() *ModalStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &ModalStats{
		TotalRequests: c.stats.TotalRequests,
		SuccessCount:  c.stats.SuccessCount,
		ErrorCount:    c.stats.ErrorCount,
		TotalOps:      c.stats.TotalOps,
		TotalDuration: c.stats.TotalDuration,
		EstimatedCost: c.stats.EstimatedCost,
	}
}

// Summary returns a formatted summary
func (c *ModalClient) Summary() string {
	stats := c.GetStats()

	opsPerSec := float64(0)
	if stats.TotalDuration.Seconds() > 0 {
		opsPerSec = float64(stats.TotalOps) / stats.TotalDuration.Seconds()
	}

	return fmt.Sprintf(`
🚀 MODAL A10G SUMMARY
═══════════════════════════════════════════════════
Requests:    %d total, %d success, %d errors
Operations:  %d total (%.2f B ops/sec)
Duration:    %v
Cost:        $%.4f estimated
GPU:         NVIDIA A10G (24GB VRAM)
`,
		stats.TotalRequests, stats.SuccessCount, stats.ErrorCount,
		stats.TotalOps, opsPerSec/1e9,
		stats.TotalDuration,
		stats.EstimatedCost,
	)
}

// ========================================================================
// HELPER FUNCTIONS
// ========================================================================

// ImageToQuaternionBatch converts images to quaternion batch for Modal processing
func ImageToQuaternionBatch(images [][]Quaternion) [][]float32 {
	batch := make([][]float32, 0)

	for _, imgQuats := range images {
		for _, q := range imgQuats {
			batch = append(batch, []float32{q.W, q.X, q.Y, q.Z})
		}
	}

	return batch
}

// QuaternionBatchToImage converts quaternion batch back to image format
func QuaternionBatchToImage(batch [][]float32, width, height int) []Quaternion {
	result := make([]Quaternion, len(batch))

	for i, q := range batch {
		if len(q) >= 4 {
			result[i] = Quaternion{W: q[0], X: q[1], Y: q[2], Z: q[3]}
		}
	}

	return result
}
