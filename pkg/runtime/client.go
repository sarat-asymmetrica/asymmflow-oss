package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client represents HTTP client for Asymmetrica.Runtime
type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates runtime client pointing to localhost:5263
func NewClient() *Client {
	return &Client{
		baseURL: "http://localhost:5263",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
}

// HealthCheck verifies Runtime is running
func (c *Client) HealthCheck(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Runtime unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Runtime health check failed: %d", resp.StatusCode)
	}
	return nil
}

// DOCUMENT PROCESSING API

type ProcessDocumentRequest struct {
	FilePath     string `json:"file_path"`
	DocumentType string `json:"document_type"` // "invoice", "po", "rfq"
}

type ProcessDocumentResponse struct {
	DocumentID     string            `json:"document_id"`
	Text           string            `json:"text"`
	Entities       map[string]string `json:"entities"`
	Classification string            `json:"classification"`
	Confidence     float64           `json:"confidence"`
}

func (c *Client) ProcessDocument(ctx context.Context, req *ProcessDocumentRequest) (*ProcessDocumentResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/documents/process", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ProcessDocumentResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

// GRAPH DATABASE API

type GraphEntity struct {
	Type  string         `json:"type"` // "customer", "supplier", "product"
	ID    string         `json:"id"`
	Props map[string]any `json:"props"`
}

type GraphRelationship struct {
	FromID string         `json:"from_id"`
	ToID   string         `json:"to_id"`
	Type   string         `json:"type"` // "purchased", "supplied", "related"
	Props  map[string]any `json:"props"`
}

func (c *Client) CreateEntity(ctx context.Context, entity *GraphEntity) error {
	body, _ := json.Marshal(entity)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/graph/entities", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) QueryGraph(ctx context.Context, query string) ([]map[string]any, error) {
	type queryReq struct {
		Query string `json:"query"`
	}
	body, _ := json.Marshal(queryReq{Query: query})
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/graph/query", bytes.NewReader(body))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []map[string]any
	json.NewDecoder(resp.Body).Decode(&results)
	return results, nil
}

// GPU ACCELERATION API

type GPUComputeRequest struct {
	Operation string         `json:"operation"` // "slerp", "quaternion_rotate", "matrix_mult"
	Input     []float64      `json:"input"`
	Params    map[string]any `json:"params"`
}

type GPUComputeResponse struct {
	Output    []float64 `json:"output"`
	Duration  float64   `json:"duration_ms"`
	Bandwidth float64   `json:"bandwidth_gbps"`
}

func (c *Client) GPUCompute(ctx context.Context, req *GPUComputeRequest) (*GPUComputeResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/gpu/compute", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result GPUComputeResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

// AI INTELLIGENCE API

type AIExecuteRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model"` // "claude-3.5-sonnet", "gpt-4o"
}

type AIExecuteResponse struct {
	Result string `json:"result"`
	Usage  struct {
		Tokens int `json:"tokens"`
	} `json:"usage"`
}

func (c *Client) AIExecute(ctx context.Context, req *AIExecuteRequest) (*AIExecuteResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/ai/execute", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result AIExecuteResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

// ============================================================================
// SMART INBOX API - Document Processing Pipeline
// ============================================================================

type InboxProcessRequest struct {
	FilePath string  `json:"file_path"`
	RawText  *string `json:"raw_text,omitempty"` // Optional: pre-extracted text
}

type InboxDocument struct {
	ID                string            `json:"id"`
	FilePath          string            `json:"file_path"`
	FileName          string            `json:"file_name"`
	DocumentType      string            `json:"document_type"` // "rfq", "invoice", "po", "quote", etc.
	Confidence        float64           `json:"confidence"`
	ExtractedData     map[string]any    `json:"extracted_data"`
	Status            string            `json:"status"` // "ready", "needs_review", "processed"
	SuggestedActions  []SuggestedAction `json:"suggested_actions"`
	ProcessedAt       time.Time         `json:"processed_at"`
	ProcessedByUserAt *time.Time        `json:"processed_by_user_at,omitempty"`
	ActionTaken       *string           `json:"action_taken,omitempty"`
}

type SuggestedAction struct {
	ActionType  string         `json:"action_type"`  // "create_opportunity", "start_costing", etc.
	Label       string         `json:"label"`        // UI button text
	Description string         `json:"description"`  // What this action does
	Priority    int            `json:"priority"`     // 1 = highest
	AutoExecute bool           `json:"auto_execute"` // Can be executed automatically?
	Parameters  map[string]any `json:"parameters"`   // Action-specific data
}

type InboxProcessResult struct {
	Success                  bool              `json:"success"`
	Error                    *string           `json:"error,omitempty"`
	DocumentID               string            `json:"document_id"`
	DetectedType             string            `json:"detected_type"`
	ClassificationConfidence float64           `json:"classification_confidence"`
	OcrConfidence            float64           `json:"ocr_confidence"`
	NeedsReview              bool              `json:"needs_review"`
	ExtractedData            map[string]any    `json:"extracted_data"`
	SuggestedActions         []SuggestedAction `json:"suggested_actions"`
	ProcessingTimeMs         int               `json:"processing_time_ms"`
}

// ProcessInboxDocument - Process a document through Smart Inbox pipeline
func (c *Client) ProcessInboxDocument(ctx context.Context, req *InboxProcessRequest) (*InboxProcessResult, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/inbox/process", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("inbox process request failed: %w", err)
	}
	defer resp.Body.Close()

	var result InboxProcessResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode inbox result: %w", err)
	}
	return &result, nil
}

// GetInboxDocuments - Get all documents in inbox (optional filter by status)
func (c *Client) GetInboxDocuments(ctx context.Context, status *string) ([]InboxDocument, error) {
	url := c.baseURL + "/api/inbox/documents"
	if status != nil {
		url += "?status=" + *status
	}

	httpReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var docs []InboxDocument
	json.NewDecoder(resp.Body).Decode(&docs)
	return docs, nil
}

type InboxStats struct {
	TotalDocuments   int            `json:"total_documents"`
	ReadyCount       int            `json:"ready_count"`
	NeedsReviewCount int            `json:"needs_review_count"`
	ProcessedCount   int            `json:"processed_count"`
	ByType           map[string]int `json:"by_type"`
}

// GetInboxStats - Get inbox statistics
func (c *Client) GetInboxStats(ctx context.Context) (*InboxStats, error) {
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/inbox/stats", nil)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats InboxStats
	json.NewDecoder(resp.Body).Decode(&stats)
	return &stats, nil
}

// ============================================================================
// PRICING INTELLIGENCE API - Customer/Product Pricing Recommendations
// ============================================================================

type PricingRequest struct {
	Customer string               `json:"customer"`
	Items    []PricingRequestItem `json:"items"`
}

type PricingRequestItem struct {
	ProductCode string  `json:"product_code"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	CostPrice   float64 `json:"cost_price"`
}

type PricingRecommendation struct {
	Customer                string               `json:"customer"`
	RequestedAt             time.Time            `json:"requested_at"`
	CustomerRegime          string               `json:"customer_regime"` // "price_sensitive", "value_balanced", "premium"
	CustomerWinRate         float64              `json:"customer_win_rate"`
	TotalQuotesWithCustomer int                  `json:"total_quotes_with_customer"`
	SuggestedMargin         float64              `json:"suggested_margin"`
	MarginRange             [2]float64           `json:"margin_range"` // [min, max]
	Confidence              float64              `json:"confidence"`
	ItemRecommendations     []ItemRecommendation `json:"item_recommendations"`
	Insights                []PricingInsight     `json:"insights"`
}

type ItemRecommendation struct {
	ProductCode     string  `json:"product_code"`
	Description     string  `json:"description"`
	Quantity        int     `json:"quantity"`
	CostPrice       float64 `json:"cost_price"`
	SuggestedMargin float64 `json:"suggested_margin"`
	MarginFloor     float64 `json:"margin_floor"`
	MarginCeiling   float64 `json:"margin_ceiling"`
	SuggestedPrice  float64 `json:"suggested_price"`
	Confidence      float64 `json:"confidence"`
}

type PricingInsight struct {
	Type        string `json:"type"` // "info", "positive", "warning", "critical"
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // Actionable guidance
}

// GetPricingRecommendation - Get AI-powered pricing recommendation
func (c *Client) GetPricingRecommendation(ctx context.Context, req *PricingRequest) (*PricingRecommendation, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/pricing/recommend", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("pricing recommend request failed: %w", err)
	}
	defer resp.Body.Close()

	var result PricingRecommendation
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode pricing recommendation: %w", err)
	}
	return &result, nil
}

type QuoteRecord struct {
	QuoteNumber string          `json:"quote_number"`
	Customer    string          `json:"customer"`
	QuoteDate   time.Time       `json:"quote_date"`
	TotalValue  float64         `json:"total_value"`
	CostValue   float64         `json:"cost_value"`
	Outcome     string          `json:"outcome"` // "pending", "won", "lost", "expired"
	LossReason  *string         `json:"loss_reason,omitempty"`
	Items       []QuoteLineItem `json:"items"`
}

type QuoteLineItem struct {
	ProductCode string  `json:"product_code"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitCost    float64 `json:"unit_cost"`
	UnitPrice   float64 `json:"unit_price"`
}

// RecordQuote - Record a quote outcome for learning
func (c *Client) RecordQuote(ctx context.Context, record *QuoteRecord) error {
	body, _ := json.Marshal(record)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/pricing/record", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type CustomerAnalytics struct {
	Customer          string           `json:"customer"`
	TotalQuotes       int              `json:"total_quotes"`
	WonQuotes         int              `json:"won_quotes"`
	LostQuotes        int              `json:"lost_quotes"`
	PendingQuotes     int              `json:"pending_quotes"`
	WinRate           float64          `json:"win_rate"`
	AverageMargin     float64          `json:"average_margin"`
	TotalRevenue      float64          `json:"total_revenue"`
	Regime            string           `json:"regime"`
	RegimeDescription string           `json:"regime_description"`
	QuoteHistory      []QuoteRecord    `json:"quote_history"`
	MonthlyTrend      []MonthlyTrend   `json:"monthly_trend"`
	TopProducts       []ProductSummary `json:"top_products"`
}

type MonthlyTrend struct {
	Month         string  `json:"month"` // "2025-12"
	QuoteCount    int     `json:"quote_count"`
	WonCount      int     `json:"won_count"`
	TotalValue    float64 `json:"total_value"`
	AverageMargin float64 `json:"average_margin"`
}

type ProductSummary struct {
	ProductCode   string  `json:"product_code"`
	Description   string  `json:"description"`
	TotalQuantity int     `json:"total_quantity"`
	TotalRevenue  float64 `json:"total_revenue"`
}

// GetCustomerAnalytics - Get detailed analytics for a customer
func (c *Client) GetCustomerAnalytics(ctx context.Context, customer string) (*CustomerAnalytics, error) {
	url := fmt.Sprintf("%s/api/pricing/customer/%s", c.baseURL, customer)
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var analytics CustomerAnalytics
	json.NewDecoder(resp.Body).Decode(&analytics)
	return &analytics, nil
}

// ============================================================================
// QUOTATION GENERATION API
// ============================================================================

type QuotationFromExcelRequest struct {
	FilePath      string `json:"file_path"`
	SaveFormat    string `json:"save_format"` // "html", "pdf", "docx"
	SavePath      string `json:"save_path"`
	Language      string `json:"language"` // "en", "ar"
	RTLMode       bool   `json:"rtl_mode"`
	CustomerName  string `json:"customer_name"`
	QuotationDate string `json:"quotation_date"`
}

type QuotationFromExcelResult struct {
	Success        bool           `json:"success"`
	Error          *string        `json:"error,omitempty"`
	QuotationData  map[string]any `json:"quotation_data"`
	OutputPath     string         `json:"output_path"`
	ProcessingTime int            `json:"processing_time_ms"`
}

// GenerateQuotationFromExcel - Generate quotation from Excel costing sheet
func (c *Client) GenerateQuotationFromExcel(ctx context.Context, req *QuotationFromExcelRequest) (*QuotationFromExcelResult, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/quotation/from-excel", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result QuotationFromExcelResult
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

// ============================================================================
// OCR PROCESSING API
// ============================================================================

type OcrProcessAutoRequest struct {
	FilePath string  `json:"file_path"`
	RawText  *string `json:"raw_text,omitempty"`
}

type OcrProcessAutoResult struct {
	Success                  bool              `json:"success"`
	Error                    *string           `json:"error,omitempty"`
	DetectedType             string            `json:"detected_type"`
	ClassificationConfidence float64           `json:"classification_confidence"`
	StructuredData           map[string]any    `json:"structured_data"`
	Confidence               float64           `json:"confidence"`
	NeedsReview              bool              `json:"needs_review"`
	ValidationResult         *ValidationResult `json:"validation_result,omitempty"`
}

type ValidationResult struct {
	Confidence    float64            `json:"confidence"`
	NeedsReview   bool               `json:"needs_review"`
	ThreadScores  map[string]float64 `json:"thread_scores"`
	MissingFields []string           `json:"missing_fields"`
}

// ProcessOCRAuto - Auto-detect document type and process with OCR
func (c *Client) ProcessOCRAuto(ctx context.Context, req *OcrProcessAutoRequest) (*OcrProcessAutoResult, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/ocr/process-auto", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result OcrProcessAutoResult
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}
