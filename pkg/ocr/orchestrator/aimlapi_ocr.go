// AIMLAPI OCR Integration - Cloud OCR for scanned documents
//
// Uses AIMLAPI's GPT-4o-mini vision model for high-quality OCR.
// This is the accuracy backstop for degraded/handwritten documents.
//
// API Key: Set via AIMLAPI_KEY environment variable
// Endpoint: https://api.aimlapi.com/v1/chat/completions
// Model: gpt-4o-mini (fast, cheap, good quality)
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
	"os"
	"sync"
	"time"
)

// AIMLAPIOCRClient handles OCR via AIMLAPI
type AIMLAPIOCRClient struct {
	apiKey     string
	endpoint   string
	model      string
	httpClient *http.Client
	stats      *AIMLAPIOCRStats
	mu         sync.RWMutex
}

// AIMLAPIOCRConfig configures the AIMLAPI client
type AIMLAPIOCRConfig struct {
	APIKey   string
	Endpoint string
	Model    string
	Timeout  time.Duration
}

// AIMLAPIOCRStats tracks OCR statistics
type AIMLAPIOCRStats struct {
	TotalRequests   int
	SuccessCount    int
	ErrorCount      int
	TotalCharacters int
	TotalDuration   time.Duration
	TotalCost       float64 // Estimated cost in USD
}

// AIMLAPIOCRResult represents OCR result
type AIMLAPIOCRResult struct {
	Success       bool
	Text          string
	Characters    int
	Duration      time.Duration
	Model         string
	Error         string
	EstimatedCost float64
}

// DefaultAIMLAPIOCRConfig returns production defaults
func DefaultAIMLAPIOCRConfig() *AIMLAPIOCRConfig {
	return &AIMLAPIOCRConfig{
		APIKey:   os.Getenv("AIMLAPI_KEY"),
		Endpoint: "https://api.aimlapi.com/v1/chat/completions",
		Model:    "gpt-4o-mini",
		Timeout:  30 * time.Second,
	}
}

// NewAIMLAPIOCRClient creates a new AIMLAPI OCR client
func NewAIMLAPIOCRClient(config *AIMLAPIOCRConfig) (*AIMLAPIOCRClient, error) {
	if config == nil {
		config = DefaultAIMLAPIOCRConfig()
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("AIMLAPI key is required")
	}

	return &AIMLAPIOCRClient{
		apiKey:   config.APIKey,
		endpoint: config.Endpoint,
		model:    config.Model,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		stats: &AIMLAPIOCRStats{},
	}, nil
}

// OCRImage performs OCR on an image
func (c *AIMLAPIOCRClient) OCRImage(ctx context.Context, img image.Image) (*AIMLAPIOCRResult, error) {
	start := time.Now()

	result := &AIMLAPIOCRResult{
		Model: c.model,
	}

	// Convert image to base64 JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		result.Error = fmt.Sprintf("failed to encode image: %v", err)
		return result, fmt.Errorf("failed to encode image: %w", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Build request
	requestBody := map[string]any{
		"model": c.model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": "Extract ALL text from this image. This is a business document (invoice, quotation, or similar). Return only the extracted text, preserving the layout as much as possible. No explanations.",
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
						},
					},
				},
			},
		},
		"max_tokens": 4096,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		result.Error = fmt.Sprintf("failed to marshal request: %v", err)
		return result, err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		c.mu.Lock()
		c.stats.TotalRequests++
		c.stats.ErrorCount++
		c.mu.Unlock()
		return result, err
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read response: %v", err)
		return result, err
	}

	result.Duration = time.Since(start)

	// Parse response
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("API error %d: %s", resp.StatusCode, string(respBody))
		c.mu.Lock()
		c.stats.TotalRequests++
		c.stats.ErrorCount++
		c.stats.TotalDuration += result.Duration
		c.mu.Unlock()
		return result, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &apiResponse); err != nil {
		result.Error = fmt.Sprintf("failed to parse response: %v", err)
		return result, err
	}

	if len(apiResponse.Choices) == 0 {
		result.Error = "no OCR results returned"
		return result, fmt.Errorf("no OCR results")
	}

	// Extract text
	result.Success = true
	result.Text = apiResponse.Choices[0].Message.Content
	result.Characters = len(result.Text)

	// Estimate cost (GPT-4o-mini: ~$0.00015/1K input tokens, ~$0.0006/1K output tokens)
	inputCost := float64(apiResponse.Usage.PromptTokens) * 0.00015 / 1000
	outputCost := float64(apiResponse.Usage.CompletionTokens) * 0.0006 / 1000
	result.EstimatedCost = inputCost + outputCost

	// Update stats
	c.mu.Lock()
	c.stats.TotalRequests++
	c.stats.SuccessCount++
	c.stats.TotalCharacters += result.Characters
	c.stats.TotalDuration += result.Duration
	c.stats.TotalCost += result.EstimatedCost
	c.mu.Unlock()

	return result, nil
}

// OCRImageBytes performs OCR on raw image bytes
func (c *AIMLAPIOCRClient) OCRImageBytes(ctx context.Context, imageData []byte, mimeType string) (*AIMLAPIOCRResult, error) {
	start := time.Now()

	result := &AIMLAPIOCRResult{
		Model: c.model,
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Build request
	requestBody := map[string]any{
		"model": c.model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": "Extract ALL text from this image. This is a business document. Return only the extracted text, no explanations.",
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image),
						},
					},
				},
			},
		},
		"max_tokens": 4096,
	}

	jsonBody, _ := json.Marshal(requestBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	result.Duration = time.Since(start)

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("API error %d", resp.StatusCode)
		return result, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	json.Unmarshal(respBody, &apiResponse)

	if len(apiResponse.Choices) > 0 {
		result.Success = true
		result.Text = apiResponse.Choices[0].Message.Content
		result.Characters = len(result.Text)
	}

	c.mu.Lock()
	c.stats.TotalRequests++
	if result.Success {
		c.stats.SuccessCount++
		c.stats.TotalCharacters += result.Characters
	} else {
		c.stats.ErrorCount++
	}
	c.stats.TotalDuration += result.Duration
	c.mu.Unlock()

	return result, nil
}

// OCRBatch performs OCR on multiple images
func (c *AIMLAPIOCRClient) OCRBatch(ctx context.Context, images []image.Image) ([]*AIMLAPIOCRResult, error) {
	results := make([]*AIMLAPIOCRResult, len(images))

	// Process sequentially to avoid rate limits
	// TODO: Add parallel processing with rate limiting
	for i, img := range images {
		result, err := c.OCRImage(ctx, img)
		if err != nil {
			results[i] = &AIMLAPIOCRResult{
				Success: false,
				Error:   err.Error(),
			}
		} else {
			results[i] = result
		}
	}

	return results, nil
}

// GetStats returns OCR statistics
func (c *AIMLAPIOCRClient) GetStats() *AIMLAPIOCRStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &AIMLAPIOCRStats{
		TotalRequests:   c.stats.TotalRequests,
		SuccessCount:    c.stats.SuccessCount,
		ErrorCount:      c.stats.ErrorCount,
		TotalCharacters: c.stats.TotalCharacters,
		TotalDuration:   c.stats.TotalDuration,
		TotalCost:       c.stats.TotalCost,
	}
}

// Summary returns a formatted summary
func (c *AIMLAPIOCRClient) Summary() string {
	stats := c.GetStats()

	avgTime := time.Duration(0)
	if stats.TotalRequests > 0 {
		avgTime = stats.TotalDuration / time.Duration(stats.TotalRequests)
	}

	return fmt.Sprintf(`
☁️ AIMLAPI OCR SUMMARY
═══════════════════════════════════════════════════
Requests:    %d total, %d success, %d errors
Characters:  %d extracted
Duration:    %v total, %v avg/request
Cost:        $%.4f estimated
Model:       %s
`,
		stats.TotalRequests, stats.SuccessCount, stats.ErrorCount,
		stats.TotalCharacters,
		stats.TotalDuration, avgTime,
		stats.TotalCost,
		c.model,
	)
}
