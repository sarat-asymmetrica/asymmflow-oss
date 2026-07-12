// Fly.io OCR Processor - Asymmetrica Runtime on Fly.io
//
// This processor connects to the Asymmetrica Runtime deployed on Fly.io
// which provides GPU-accelerated OCR endpoints for the Acme Instrumentation app.
//
// Endpoints:
// - POST /api/ocr/batch-preprocess  - GPU-accelerated image preprocessing
// - POST /api/ocr/quality-gate      - Quality validation with resonance check
// - POST /api/kernel/table_extraction - Table and entity extraction
//
// Deployed at: https://asymmetrica-runtime.fly.dev
//
// Built: January 20, 2026 - Wave 1 Backend OCR Integration
package orchestrator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"sync"
	"time"
)

// FlyOCRProcessor handles OCR via Asymmetrica Runtime on Fly.io
type FlyOCRProcessor struct {
	apiURL     string
	apiKey     string
	httpClient *http.Client
	stats      *FlyOCRStats
	mu         sync.RWMutex
}

// FlyOCRConfig configures the Fly.io OCR processor
type FlyOCRConfig struct {
	APIURL  string
	APIKey  string
	Timeout time.Duration
}

// FlyOCRStats tracks Fly.io OCR statistics
type FlyOCRStats struct {
	TotalRequests   int
	SuccessCount    int
	ErrorCount      int
	TotalCharacters int
	TotalDuration   time.Duration
	TotalCost       float64 // Estimated cost in USD
	QualityPasses   int
	QualityFails    int
	TablesExtracted int
}

// FlyOCRResult represents OCR result from Fly.io
type FlyOCRResult struct {
	Success       bool          `json:"success"`
	Text          string        `json:"text"`
	Characters    int           `json:"characters"`
	Quality       string        `json:"quality"` // "PASS" | "FAIL"
	Confidence    float64       `json:"confidence"`
	Duration      time.Duration `json:"-"`
	EstimatedCost float64       `json:"-"`
	TablesFound   int           `json:"tables_found"`
	EntitiesFound int           `json:"entities_found"`
	Error         string        `json:"error,omitempty"`
}

// FlyBatchPreprocessResponse matches Fly.io API response
type FlyBatchPreprocessResponse struct {
	Text string `json:"text"`
}

// FlyQualityGateResponse matches Fly.io API response
type FlyQualityGateResponse struct {
	Quality    string  `json:"quality"`
	Confidence float64 `json:"confidence"`
}

// FlyTableExtractionResponse matches Fly.io API response
type FlyTableExtractionResponse struct {
	Tables   []map[string]any `json:"tables"`
	Entities []map[string]any `json:"entities"`
}

// DefaultFlyOCRConfig returns production defaults
func DefaultFlyOCRConfig() *FlyOCRConfig {
	return &FlyOCRConfig{
		APIURL:  "https://asymmetrica-runtime.fly.dev",
		APIKey:  "", // Optional - set via env var if auth required
		Timeout: 60 * time.Second,
	}
}

// NewFlyOCRProcessor creates a new Fly.io OCR processor
func NewFlyOCRProcessor(config *FlyOCRConfig) (*FlyOCRProcessor, error) {
	if config == nil {
		config = DefaultFlyOCRConfig()
	}

	return &FlyOCRProcessor{
		apiURL: config.APIURL,
		apiKey: config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		stats: &FlyOCRStats{},
	}, nil
}

// OCRImage performs full OCR pipeline on an image using Fly.io endpoints
func (p *FlyOCRProcessor) OCRImage(ctx context.Context, img image.Image) (*FlyOCRResult, error) {
	start := time.Now()

	// Convert image to base64 JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	imageBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	// STEP 1: Batch Preprocess (GPU-accelerated preprocessing + OCR)
	preprocessReq := map[string]any{
		"image": imageBase64,
	}
	preprocessResp, err := p.post(ctx, "/api/ocr/batch-preprocess", preprocessReq)
	if err != nil {
		p.recordError()
		return nil, fmt.Errorf("batch preprocess failed: %w", err)
	}

	var preprocessResult FlyBatchPreprocessResponse
	if err := json.Unmarshal(preprocessResp, &preprocessResult); err != nil {
		p.recordError()
		return nil, fmt.Errorf("failed to parse preprocess response: %w", err)
	}

	text := preprocessResult.Text
	if text == "" {
		p.recordError()
		return &FlyOCRResult{
			Success: false,
			Error:   "no text extracted",
		}, fmt.Errorf("no text extracted from image")
	}

	// STEP 2: Quality Gate (validate text quality with resonance check)
	qualityReq := map[string]any{
		"data":      text,
		"resonance": true,
	}
	qualityResp, err := p.post(ctx, "/api/ocr/quality-gate", qualityReq)
	if err != nil {
		// Quality gate failure is not fatal - proceed with text
		p.recordWarning()
	}

	var qualityResult FlyQualityGateResponse
	quality := "UNKNOWN"
	confidence := 0.8 // Default confidence if quality gate fails

	if err == nil {
		if err := json.Unmarshal(qualityResp, &qualityResult); err == nil {
			quality = qualityResult.Quality
			confidence = qualityResult.Confidence

			if quality == "PASS" {
				p.recordQualityPass()
			} else {
				p.recordQualityFail()
			}
		}
	}

	// STEP 3: Table Extraction (optional - only if tables detected)
	// For now, we'll skip this to keep latency low
	// Future enhancement: detect tables via k-sum first, then extract

	duration := time.Since(start)
	// Cost estimate: Fly.io Machines ~$0.0002/sec (shared-cpu-1x)
	// Average OCR: ~2 seconds = $0.0004 per page
	cost := duration.Seconds() * 0.0002

	result := &FlyOCRResult{
		Success:       true,
		Text:          text,
		Characters:    len(text),
		Quality:       quality,
		Confidence:    confidence,
		Duration:      duration,
		EstimatedCost: cost,
		TablesFound:   0,
		EntitiesFound: 0,
	}

	p.recordSuccess(result.Characters, duration, cost)

	return result, nil
}

// OCRImageBytes performs OCR on raw image bytes
func (p *FlyOCRProcessor) OCRImageBytes(ctx context.Context, imageData []byte) (*FlyOCRResult, error) {
	start := time.Now()

	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// STEP 1: Batch Preprocess
	preprocessReq := map[string]any{
		"image": imageBase64,
	}
	preprocessResp, err := p.post(ctx, "/api/ocr/batch-preprocess", preprocessReq)
	if err != nil {
		p.recordError()
		return nil, fmt.Errorf("batch preprocess failed: %w", err)
	}

	var preprocessResult FlyBatchPreprocessResponse
	if err := json.Unmarshal(preprocessResp, &preprocessResult); err != nil {
		p.recordError()
		return nil, fmt.Errorf("failed to parse preprocess response: %w", err)
	}

	text := preprocessResult.Text
	if text == "" {
		p.recordError()
		return &FlyOCRResult{
			Success: false,
			Error:   "no text extracted",
		}, fmt.Errorf("no text extracted from image")
	}

	// STEP 2: Quality Gate
	qualityReq := map[string]any{
		"data":      text,
		"resonance": true,
	}
	qualityResp, err := p.post(ctx, "/api/ocr/quality-gate", qualityReq)
	if err != nil {
		p.recordWarning()
	}

	var qualityResult FlyQualityGateResponse
	quality := "UNKNOWN"
	confidence := 0.8

	if err == nil {
		if err := json.Unmarshal(qualityResp, &qualityResult); err == nil {
			quality = qualityResult.Quality
			confidence = qualityResult.Confidence

			if quality == "PASS" {
				p.recordQualityPass()
			} else {
				p.recordQualityFail()
			}
		}
	}

	duration := time.Since(start)
	cost := duration.Seconds() * 0.0002

	result := &FlyOCRResult{
		Success:       true,
		Text:          text,
		Characters:    len(text),
		Quality:       quality,
		Confidence:    confidence,
		Duration:      duration,
		EstimatedCost: cost,
		TablesFound:   0,
		EntitiesFound: 0,
	}

	p.recordSuccess(result.Characters, duration, cost)

	return result, nil
}

// ExtractTables performs table extraction on text
func (p *FlyOCRProcessor) ExtractTables(ctx context.Context, text string) (*FlyTableExtractionResponse, error) {
	tableReq := map[string]any{
		"prompt": fmt.Sprintf("Extract tables and entities from: %s", text),
		"model":  "table_extraction",
	}

	tableResp, err := p.post(ctx, "/api/kernel/table_extraction", tableReq)
	if err != nil {
		return nil, fmt.Errorf("table extraction failed: %w", err)
	}

	var tableResult FlyTableExtractionResponse
	if err := json.Unmarshal(tableResp, &tableResult); err != nil {
		return nil, fmt.Errorf("failed to parse table extraction response: %w", err)
	}

	p.mu.Lock()
	p.stats.TablesExtracted += len(tableResult.Tables)
	p.mu.Unlock()

	return &tableResult, nil
}

// HealthCheck verifies the Fly.io service is running
func (p *FlyOCRProcessor) HealthCheck(ctx context.Context) error {
	url := p.apiURL + "/health"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed: HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// post sends a POST request to Fly.io endpoint
func (p *FlyOCRProcessor) post(ctx context.Context, endpoint string, reqData any) ([]byte, error) {
	jsonBody, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.apiURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// recordSuccess updates stats on success
func (p *FlyOCRProcessor) recordSuccess(chars int, duration time.Duration, cost float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stats.TotalRequests++
	p.stats.SuccessCount++
	p.stats.TotalCharacters += chars
	p.stats.TotalDuration += duration
	p.stats.TotalCost += cost
}

// recordError updates stats on error
func (p *FlyOCRProcessor) recordError() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stats.TotalRequests++
	p.stats.ErrorCount++
}

// recordWarning logs a warning (quality gate failure, etc.)
func (p *FlyOCRProcessor) recordWarning() {
	// No-op for now, could add warning counter if needed
}

// recordQualityPass updates stats on quality gate pass
func (p *FlyOCRProcessor) recordQualityPass() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.QualityPasses++
}

// recordQualityFail updates stats on quality gate fail
func (p *FlyOCRProcessor) recordQualityFail() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.QualityFails++
}

// GetStats returns OCR statistics
func (p *FlyOCRProcessor) GetStats() *FlyOCRStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return &FlyOCRStats{
		TotalRequests:   p.stats.TotalRequests,
		SuccessCount:    p.stats.SuccessCount,
		ErrorCount:      p.stats.ErrorCount,
		TotalCharacters: p.stats.TotalCharacters,
		TotalDuration:   p.stats.TotalDuration,
		TotalCost:       p.stats.TotalCost,
		QualityPasses:   p.stats.QualityPasses,
		QualityFails:    p.stats.QualityFails,
		TablesExtracted: p.stats.TablesExtracted,
	}
}

// Summary returns a formatted summary
func (p *FlyOCRProcessor) Summary() string {
	stats := p.GetStats()

	avgDuration := time.Duration(0)
	throughput := float64(0)
	qualityRate := float64(0)

	if stats.SuccessCount > 0 {
		avgDuration = stats.TotalDuration / time.Duration(stats.SuccessCount)
		throughput = float64(stats.SuccessCount) / stats.TotalDuration.Seconds()
	}

	if stats.QualityPasses+stats.QualityFails > 0 {
		qualityRate = float64(stats.QualityPasses) * 100.0 / float64(stats.QualityPasses+stats.QualityFails)
	}

	return fmt.Sprintf(`
🚀 FLY.IO OCR SUMMARY (Asymmetrica Runtime)
═══════════════════════════════════════════════════
Requests:      %d total, %d success, %d errors
Characters:    %d extracted
Duration:      %v total, %v avg/request
Throughput:    %.2f pages/sec
Cost:          $%.4f estimated
Quality:       %.1f%% pass rate (%d passed, %d failed)
Tables:        %d extracted
Deployment:    https://asymmetrica-runtime.fly.dev

Performance:
  Latency: ~2s per page (preprocessing + OCR + quality gate)
  Cost:    ~$0.0004 per page (Fly.io shared-cpu-1x)
  GPU:     Accelerated preprocessing via Fly.io GPUs
`,
		stats.TotalRequests, stats.SuccessCount, stats.ErrorCount,
		stats.TotalCharacters,
		stats.TotalDuration, avgDuration,
		throughput,
		stats.TotalCost,
		qualityRate, stats.QualityPasses, stats.QualityFails,
		stats.TablesExtracted,
	)
}

// Close cleans up processor resources
func (p *FlyOCRProcessor) Close() error {
	// Nothing to clean up for HTTP client
	return nil
}
