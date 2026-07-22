// Client for Mistral's dedicated OCR endpoint (OCR 4 / mistral-ocr-4-0).
// σ: MistralOCR-Client | ρ: pkg/ocr/mistralocr | γ: Production | κ: O(1) per request
//
// Submits PDFs and images natively (no page-render-to-PNG loop). Supports whole-document
// structured extraction via a caller-supplied JSON schema (Document AI annotations). The API
// key is always caller-injected — this client never reads env vars, settings.json, or a
// database; callers should route through the app's existing key resolver.
package mistralocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultModel is mistral-ocr-4-0 per the OCR 4 model card (docs.mistral.ai/models/model-cards/ocr-4-0).
	DefaultModel = "mistral-ocr-4-0"
	// DefaultBaseURL is Mistral's direct API host.
	DefaultBaseURL = "https://api.mistral.ai"
	// DefaultPageCap is well under the documented ~1000-page ceiling; callers process large
	// documents in batches rather than relying on the server-side maximum.
	DefaultPageCap = 200
	// DefaultTimeout for a single OCR call.
	DefaultTimeout = 60 * time.Second
	// DefaultConfidenceThreshold: fields below this are marked NeedsReview.
	DefaultConfidenceThreshold = 0.85

	ocrPath = "/v1/ocr"
)

// Config configures a Client. All fields have sane defaults applied by NewClient — model IDs,
// endpoints, thresholds, and page caps are data, never literals sprinkled at call sites.
type Config struct {
	APIKey              string // required; caller-injected, never read from env/DB by this package
	BaseURL             string
	Model               string
	PageCap             int
	Timeout             time.Duration
	ConfidenceThreshold float64

	// HTTPClient allows callers (and tests) to inject a transport. If nil, a client with
	// Timeout is constructed.
	HTTPClient *http.Client
}

// Client talks to the Mistral OCR endpoint.
type Client struct {
	apiKey              string
	baseURL             string
	model               string
	pageCap             int
	confidenceThreshold float64
	httpClient          *http.Client
}

// NewClient builds a Client, applying defaults for any zero-valued Config fields.
func NewClient(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}
	pageCap := cfg.PageCap
	if pageCap <= 0 {
		pageCap = DefaultPageCap
	}
	threshold := cfg.ConfidenceThreshold
	if threshold <= 0 {
		threshold = DefaultConfidenceThreshold
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Client{
		apiKey:              cfg.APIKey,
		baseURL:             strings.TrimRight(baseURL, "/"),
		model:               model,
		pageCap:             pageCap,
		confidenceThreshold: threshold,
		httpClient:          httpClient,
	}
}

// Process submits one document to the OCR endpoint and returns a decoded Result.
func (c *Client) Process(ctx context.Context, doc DocumentInput, opts ProcessOptions) (*Result, error) {
	if requested, ok := parsePageCount(opts.Pages); ok && requested > c.pageCap {
		return nil, fmt.Errorf("mistralocr: requested %d pages exceeds page cap %d", requested, c.pageCap)
	}

	docChunk, err := c.buildDocumentChunk(doc)
	if err != nil {
		return nil, err
	}

	req := ocrRequest{
		Model:         c.model,
		Document:      docChunk,
		Pages:         opts.Pages,
		IncludeBlocks: opts.IncludeBlocks,
	}

	hadSchema := opts.Schema != nil
	if hadSchema {
		// Requesting a schema always asks for page-level confidence so the client can derive
		// an honest per-field confidence signal (see Result.Fields docs).
		req.ConfidenceScoresGranularity = "page"
		req.DocumentAnnotationFormat = &responseFormat{
			Type: "json_schema",
			JSONSchema: jsonSchemaEnvelope{
				Name:   opts.Schema.Name,
				Schema: opts.Schema.Schema,
				Strict: opts.Schema.Strict,
			},
		}
	} else {
		req.ConfidenceScoresGranularity = "page"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mistralocr: failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ocrPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mistralocr: failed to build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mistralocr: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mistralocr: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var env errorEnvelope
		_ = json.Unmarshal(respBody, &env)
		if env.Message == "" {
			env.Message = string(respBody)
		}
		return nil, classifyError(resp.StatusCode, env, hadSchema)
	}

	var wire ocrResponse
	if err := json.Unmarshal(respBody, &wire); err != nil {
		return nil, fmt.Errorf("mistralocr: failed to decode response: %w", err)
	}

	return c.decodeResult(wire, opts), nil
}

// buildDocumentChunk turns a DocumentInput into the document_url/image_url chunk the API
// expects. Local bytes are sent as a data: URI in the same string field — the OCRRequest wire
// shape has no separate raw-base64 field (see FABLE_WAVE13_REPORT.md A0 section).
func (c *Client) buildDocumentChunk(doc DocumentInput) (any, error) {
	hasURL := doc.URL != ""
	hasData := len(doc.Data) > 0

	if hasURL == hasData {
		return nil, fmt.Errorf("mistralocr: DocumentInput must set exactly one of URL or Data")
	}

	value := doc.URL
	if hasData {
		if doc.MIMEType == "" {
			return nil, fmt.Errorf("mistralocr: MIMEType is required when Data is set")
		}
		value = fmt.Sprintf("data:%s;base64,%s", doc.MIMEType, base64.StdEncoding.EncodeToString(doc.Data))
	}

	if doc.IsImage {
		return imageURLChunk{Type: "image_url", ImageURL: value}, nil
	}
	return documentURLChunk{Type: "document_url", DocumentURL: value}, nil
}

// decodeResult converts the wire response into the typed Result, deriving per-field confidence
// from page-level confidence scores per the rule documented in the wave report: the minimum
// page confidence across the document, or 0 (NeedsReview) if no confidence signal was returned.
func (c *Client) decodeResult(wire ocrResponse, opts ProcessOptions) *Result {
	result := &Result{
		ModelID: wire.Model,
		Pages:   make([]string, 0, len(wire.Pages)),
	}

	var texts []string
	minPageConfidence := -1.0
	sawConfidence := false

	for _, p := range wire.Pages {
		result.Pages = append(result.Pages, p.Markdown)
		texts = append(texts, p.Markdown)

		if p.MinimumPageConfidenceScore > 0 || p.AveragePageConfidenceScore > 0 {
			sawConfidence = true
			pageMin := p.MinimumPageConfidenceScore
			if pageMin == 0 {
				pageMin = p.AveragePageConfidenceScore
			}
			if minPageConfidence < 0 || pageMin < minPageConfidence {
				minPageConfidence = pageMin
			}
		}

		for _, b := range p.Blocks {
			block := Block{
				PageIndex:  p.Index,
				Type:       b.Type,
				Text:       b.Text,
				Confidence: b.Confidence,
			}
			if b.BBox != nil {
				block.BBox = &BoundingBox{X0: b.BBox.X0, Y0: b.BBox.Y0, X1: b.BBox.X1, Y1: b.BBox.Y1}
			}
			result.Blocks = append(result.Blocks, block)
		}
	}
	result.Text = strings.Join(texts, "\n\n")

	if opts.Schema != nil && wire.DocumentAnnotation != nil {
		confidence := 0.0
		if sawConfidence {
			confidence = minPageConfidence
		}
		result.Fields = make(map[string]FieldValue, len(wire.DocumentAnnotation))
		for k, v := range wire.DocumentAnnotation {
			result.Fields[k] = FieldValue{
				Value:       v,
				Confidence:  confidence,
				NeedsReview: confidence < c.confidenceThreshold,
			}
		}
	}

	return result
}

// parsePageCount parses the API's "pages" syntax (comma-separated integers and a-b ranges) and
// returns the total page count it would request, or ok=false if pages is empty/unparseable
// (in which case the client does not enforce a client-side cap — the server's own limit applies).
func parsePageCount(pages string) (int, bool) {
	pages = strings.TrimSpace(pages)
	if pages == "" {
		return 0, false
	}

	total := 0
	for _, part := range strings.Split(pages, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.Index(part, "-"); idx > 0 {
			lo, err1 := strconv.Atoi(strings.TrimSpace(part[:idx]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(part[idx+1:]))
			if err1 != nil || err2 != nil || hi < lo {
				return 0, false
			}
			total += hi - lo + 1
			continue
		}
		if _, err := strconv.Atoi(part); err != nil {
			return 0, false
		}
		total++
	}
	if total == 0 {
		return 0, false
	}
	return total, true
}
