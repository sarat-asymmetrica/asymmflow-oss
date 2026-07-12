// AIMLAPI client for advanced OCR.
// σ: AIMLAPI | ρ: pkg/ocr | γ: Integration | κ: O(1) per request
//
// AIMLAPI provides state-of-the-art vision models for tough OCR cases:
// - Low confidence local OCR → escalate to AIMLAPI
// - Multi-language documents
// - Handwritten text
// - Degraded/low quality images
//
// Cost: ~$0.001-0.005 per document (very cheap!)
package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ========================================================================
// AIMLAPI CLIENT
// ========================================================================

// AIMLAPIClient handles communication with AIMLAPI
type AIMLAPIClient struct {
	apiKey     string
	endpoint   string
	httpClient *http.Client
	model      string
}

// AIMLAPIConfig configures the AIMLAPI client
type AIMLAPIConfig struct {
	APIKey   string
	Endpoint string
	Model    string
	Timeout  time.Duration
}

// NewAIMLAPIClient creates a new AIMLAPI client
func NewAIMLAPIClient(apiKey string, endpoint string) *AIMLAPIClient {
	if endpoint == "" {
		endpoint = "https://api.aimlapi.com/v1"
	}

	return &AIMLAPIClient{
		apiKey:   apiKey,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		model: "gpt-4o-mini", // Cost-effective vision model
	}
}

// NewAIMLAPIClientWithConfig creates a client with full configuration
func NewAIMLAPIClientWithConfig(config *AIMLAPIConfig) *AIMLAPIClient {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://api.aimlapi.com/v1"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	model := config.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &AIMLAPIClient{
		apiKey:   config.APIKey,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		model: model,
	}
}

// ========================================================================
// OCR PROCESSING
// ========================================================================

// Process sends an image to AIMLAPI for OCR processing
func (c *AIMLAPIClient) Process(ctx context.Context, imageData []byte, req *ProcessRequest) (*ProcessResponse, error) {
	startTime := time.Now()

	// Encode image to base64
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// Determine image type from magic bytes
	imageType := c.detectImageType(imageData)

	// Build the prompt based on document type
	prompt := c.buildPrompt(req.DocumentType, req.Language)

	// Build request body
	requestBody := map[string]any{
		"model": c.model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": prompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", imageType, imageBase64),
						},
					},
				},
			},
		},
		"max_tokens":  2000,
		"temperature": 0.1, // Low temperature for consistent extraction
	}

	// Marshal request
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AIMLAPI returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var aiResponse AIMLAPIResponse
	if err := json.Unmarshal(respBody, &aiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from response
	if len(aiResponse.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := aiResponse.Choices[0].Message.Content

	// Parse extracted fields from the content
	fields, confidence := c.parseExtractedContent(content, req.DocumentType)

	return &ProcessResponse{
		Text:             content,
		Fields:           fields,
		Confidence:       confidence,
		DocumentType:     req.DocumentType,
		Tier:             TierAIMLAPI,
		ProcessingTime:   time.Since(startTime),
		EstimatedCostUSD: c.estimateCost(len(imageData), len(content)),
	}, nil
}

// ========================================================================
// HELPER METHODS
// ========================================================================

// AIMLAPIResponse represents the API response structure
type AIMLAPIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// detectImageType detects image MIME type from magic bytes
func (c *AIMLAPIClient) detectImageType(data []byte) string {
	if len(data) < 4 {
		return "image/png"
	}

	// PNG
	if bytes.HasPrefix(data, []byte("\x89PNG")) {
		return "image/png"
	}
	// JPEG
	if bytes.HasPrefix(data, []byte("\xff\xd8\xff")) {
		return "image/jpeg"
	}
	// GIF
	if bytes.HasPrefix(data, []byte("GIF")) {
		return "image/gif"
	}
	// WebP
	if len(data) >= 12 && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	// Default to PNG
	return "image/png"
}

// buildPrompt creates the extraction prompt based on document type
func (c *AIMLAPIClient) buildPrompt(docType DocumentType, lang Language) string {
	basePrompt := `You are an expert OCR system. Extract all text and structured fields from this document image.

Return your response in the following JSON format:
{
  "raw_text": "full extracted text here",
  "confidence": 0.95,
  "fields": {
    "field_name": "field_value"
  }
}

Be precise and accurate. If you cannot read something clearly, indicate low confidence.
`

	// Add document-specific instructions
	switch docType {
	case DocTypeInvoice:
		basePrompt += `
This is an INVOICE document. Extract these fields if present:
- invoice_number
- date
- customer_name
- items (as array)
- subtotal
- vat_amount
- total
- currency
- payment_terms

For Arabic text, ensure proper RTL handling.`

	case DocTypePassport:
		basePrompt += `
This is a PASSPORT document. Extract these fields:
- surname
- given_names
- date_of_birth
- place_of_birth
- nationality
- passport_number
- date_of_issue
- date_of_expiry
- sex
- mrz_line1
- mrz_line2`

	case DocTypeContract:
		basePrompt += `
This is a CONTRACT/EMPLOYMENT document. Extract:
- employee_name
- employer_name
- job_title
- start_date
- salary
- currency
- contract_duration
- benefits`

	case DocTypeDiploma:
		basePrompt += `
This is an EDUCATIONAL document. Extract:
- student_name
- institution_name
- degree_title
- date_of_graduation
- grade_or_gpa
- certificate_number`

	case DocTypeBOQ:
		basePrompt += `
This is a BILL OF QUANTITIES document. Extract:
- project_name
- items (array with description, quantity, unit, rate, amount)
- subtotal
- vat
- total`

	default:
		basePrompt += `
Extract all visible text and any structured data you can identify.`
	}

	// Add language hint
	if lang != "" && lang != LangAuto {
		basePrompt += fmt.Sprintf("\n\nThe document is primarily in %s language.", c.languageToName(lang))
	}

	return basePrompt
}

// languageToName converts language code to readable name
func (c *AIMLAPIClient) languageToName(lang Language) string {
	names := map[Language]string{
		LangEnglish:    "English",
		LangArabic:     "Arabic",
		LangHindi:      "Hindi",
		LangChinese:    "Chinese",
		LangJapanese:   "Japanese",
		LangKorean:     "Korean",
		LangRussian:    "Russian",
		LangFrench:     "French",
		LangSpanish:    "Spanish",
		LangGerman:     "German",
		LangDutch:      "Dutch",
		LangPortuguese: "Portuguese",
	}

	if name, ok := names[lang]; ok {
		return name
	}
	return "Unknown"
}

// parseExtractedContent parses the AI response into fields
func (c *AIMLAPIClient) parseExtractedContent(content string, docType DocumentType) (map[string]string, float64) {
	fields := make(map[string]string)
	confidence := 0.85 // Default confidence for AIMLAPI

	// Try to parse as JSON
	var parsed struct {
		RawText    string         `json:"raw_text"`
		Confidence float64        `json:"confidence"`
		Fields     map[string]any `json:"fields"`
	}

	// Find JSON in the response (it might be wrapped in markdown code blocks)
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")

	if jsonStart >= 0 && jsonEnd > jsonStart {
		jsonStr := content[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
			// Successfully parsed JSON
			fields["raw_text"] = parsed.RawText
			if parsed.Confidence > 0 {
				confidence = parsed.Confidence
			}

			// Add all extracted fields
			for k, v := range parsed.Fields {
				switch val := v.(type) {
				case string:
					fields[k] = val
				case float64:
					fields[k] = fmt.Sprintf("%.2f", val)
				case int:
					fields[k] = fmt.Sprintf("%d", val)
				default:
					// Convert complex types to JSON string
					if jsonBytes, err := json.Marshal(val); err == nil {
						fields[k] = string(jsonBytes)
					}
				}
			}

			return fields, confidence
		}
	}

	// Fallback: store raw content
	fields["raw_text"] = content
	return fields, confidence
}

// estimateCost estimates the API call cost
func (c *AIMLAPIClient) estimateCost(imageBytes int, responseTokens int) float64 {
	// AIMLAPI pricing (approximate):
	// - Image input: ~$0.00025 per 1000x1000 pixels
	// - Text output: ~$0.0002 per 1000 tokens

	// Estimate image tokens (rough: 1 token per 100 bytes)
	imageTokens := imageBytes / 100

	// Total tokens (not used currently, but kept for future reference)
	_ = imageTokens + responseTokens

	// Cost calculation (gpt-4o-mini pricing)
	// Input: $0.00015 per 1K tokens
	// Output: $0.0006 per 1K tokens
	inputCost := float64(imageTokens) / 1000.0 * 0.00015
	outputCost := float64(responseTokens) / 1000.0 * 0.0006

	return inputCost + outputCost
}

// ========================================================================
// BATCH PROCESSING
// ========================================================================

// ProcessBatch processes multiple images concurrently
func (c *AIMLAPIClient) ProcessBatch(ctx context.Context, images [][]byte, req *ProcessRequest) ([]*ProcessResponse, error) {
	responses := make([]*ProcessResponse, len(images))
	errors := make([]error, len(images))

	// Simple sequential processing for now
	// TODO: Implement concurrent processing with rate limiting
	for i, imageData := range images {
		resp, err := c.Process(ctx, imageData, req)
		if err != nil {
			errors[i] = err
		} else {
			responses[i] = resp
		}
	}

	return responses, nil
}

// ========================================================================
// HEALTH CHECK
// ========================================================================

// HealthCheck verifies API connectivity
func (c *AIMLAPIClient) HealthCheck(ctx context.Context) error {
	// Simple models list request to verify API key works
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint+"/models", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("AIMLAPI health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AIMLAPI returned status %d", resp.StatusCode)
	}

	return nil
}
