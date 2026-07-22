package mistralocr

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient wires the Client at the given mock server, with a fast timeout so a hung
// request fails the test quickly rather than the default 60s.
func newTestClient(t *testing.T, srv *httptest.Server, cfgOverrides func(*Config)) *Client {
	t.Helper()
	cfg := Config{
		APIKey:  "test-key-synthetic",
		BaseURL: srv.URL,
		Timeout: 5 * time.Second,
	}
	if cfgOverrides != nil {
		cfgOverrides(&cfg)
	}
	return NewClient(cfg)
}

func TestProcess_HappyPath_PlainOCR(t *testing.T) {
	var capturedBody ocrRequest
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		raw, _ := decodeRawRequest(r)
		capturedBody = raw

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ocrResponse{
			Model: "mistral-ocr-4-0",
			Pages: []ocrPageObject{
				{Index: 0, Markdown: "# Invoice 001", AveragePageConfidenceScore: 0.97, MinimumPageConfidenceScore: 0.95},
				{Index: 1, Markdown: "Total: 100 BHD", AveragePageConfidenceScore: 0.92, MinimumPageConfidenceScore: 0.90},
			},
			UsageInfo: ocrUsageInfo{PagesProcessed: 2},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil)
	res, err := c.Process(context.Background(), DocumentInput{
		Data:     []byte("%PDF-1.4 fake pdf bytes"),
		MIMEType: "application/pdf",
	}, ProcessOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedAuth != "Bearer test-key-synthetic" {
		t.Errorf("Authorization header = %q, want Bearer test-key-synthetic", capturedAuth)
	}
	if capturedBody.Model != "mistral-ocr-4-0" {
		t.Errorf("request model = %q, want mistral-ocr-4-0", capturedBody.Model)
	}
	chunk, ok := capturedBody.Document.(map[string]any)
	if !ok {
		t.Fatalf("document chunk did not decode as object: %#v", capturedBody.Document)
	}
	if chunk["type"] != "document_url" {
		t.Errorf("document chunk type = %v, want document_url", chunk["type"])
	}
	docURL, _ := chunk["document_url"].(string)
	if !strings.HasPrefix(docURL, "data:application/pdf;base64,") {
		t.Errorf("document_url = %q, want data:application/pdf;base64,... prefix", docURL)
	}

	if res.ModelID != "mistral-ocr-4-0" {
		t.Errorf("ModelID = %q, want mistral-ocr-4-0", res.ModelID)
	}
	if len(res.Pages) != 2 || res.Pages[0] != "# Invoice 001" || res.Pages[1] != "Total: 100 BHD" {
		t.Errorf("Pages = %#v, want [\"# Invoice 001\" \"Total: 100 BHD\"]", res.Pages)
	}
	wantText := "# Invoice 001\n\nTotal: 100 BHD"
	if res.Text != wantText {
		t.Errorf("Text = %q, want %q", res.Text, wantText)
	}
	if len(res.Fields) != 0 {
		t.Errorf("Fields = %#v, want empty (no schema requested)", res.Fields)
	}
}

func TestProcess_HappyPath_WithSchemaAnnotation(t *testing.T) {
	schema := &DocumentSchema{
		Name:   "invoice_fields",
		Schema: map[string]any{"type": "object", "properties": map[string]any{"invoice_number": map[string]any{"type": "string"}}},
		Strict: true,
	}

	var capturedBody ocrRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := decodeRawRequest(r)
		capturedBody = raw

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ocrResponse{
			Model: "mistral-ocr-4-0",
			Pages: []ocrPageObject{
				{Index: 0, Markdown: "INV-42", AveragePageConfidenceScore: 0.99, MinimumPageConfidenceScore: 0.98},
			},
			DocumentAnnotation: map[string]any{"invoice_number": "INV-42"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil)
	res, err := c.Process(context.Background(), DocumentInput{URL: "https://example.test/invoice.pdf"}, ProcessOptions{
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedBody.DocumentAnnotationFormat == nil {
		t.Fatalf("request did not carry document_annotation_format")
	}
	if capturedBody.DocumentAnnotationFormat.JSONSchema.Name != "invoice_fields" {
		t.Errorf("schema name = %q, want invoice_fields", capturedBody.DocumentAnnotationFormat.JSONSchema.Name)
	}
	if !capturedBody.DocumentAnnotationFormat.JSONSchema.Strict {
		t.Errorf("schema Strict = false, want true")
	}
	if capturedBody.ConfidenceScoresGranularity != "page" {
		t.Errorf("confidence_scores_granularity = %q, want page", capturedBody.ConfidenceScoresGranularity)
	}

	fv, ok := res.Fields["invoice_number"]
	if !ok {
		t.Fatalf("Fields missing invoice_number: %#v", res.Fields)
	}
	if fv.Value != "INV-42" {
		t.Errorf("invoice_number value = %v, want INV-42", fv.Value)
	}
	if fv.NeedsReview {
		t.Errorf("invoice_number NeedsReview = true, want false (confidence 0.98 above default threshold)")
	}
	if fv.Confidence != 0.98 {
		t.Errorf("invoice_number confidence = %v, want 0.98 (minimum page confidence)", fv.Confidence)
	}
}

func TestProcess_LowConfidenceFlagging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ocrResponse{
			Model: "mistral-ocr-4-0",
			Pages: []ocrPageObject{
				{Index: 0, Markdown: "blurry scan", AveragePageConfidenceScore: 0.60, MinimumPageConfidenceScore: 0.40},
			},
			DocumentAnnotation: map[string]any{"po_number": "PO-99"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil)
	res, err := c.Process(context.Background(), DocumentInput{URL: "https://example.test/po.pdf"}, ProcessOptions{
		Schema: &DocumentSchema{Name: "po_fields", Schema: map[string]any{"type": "object"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fv, ok := res.Fields["po_number"]
	if !ok {
		t.Fatalf("Fields missing po_number: %#v", res.Fields)
	}
	if fv.Confidence != 0.40 {
		t.Errorf("po_number confidence = %v, want 0.40 (carried through, not dropped)", fv.Confidence)
	}
	if !fv.NeedsReview {
		t.Errorf("po_number NeedsReview = false, want true (0.40 below default threshold 0.85)")
	}
}

func TestProcess_MissingConfidenceSignal_RefusesToGuess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// No confidence fields at all in the page objects.
		_ = json.NewEncoder(w).Encode(ocrResponse{
			Model:              "mistral-ocr-4-0",
			Pages:              []ocrPageObject{{Index: 0, Markdown: "some text"}},
			DocumentAnnotation: map[string]any{"field_a": "value_a"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil)
	res, err := c.Process(context.Background(), DocumentInput{URL: "https://example.test/doc.pdf"}, ProcessOptions{
		Schema: &DocumentSchema{Name: "generic", Schema: map[string]any{"type": "object"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fv := res.Fields["field_a"]
	if fv.Confidence != 0 {
		t.Errorf("field_a confidence = %v, want 0 (no signal present)", fv.Confidence)
	}
	if !fv.NeedsReview {
		t.Errorf("field_a NeedsReview = false, want true when no confidence signal exists")
	}
}

func TestProcess_ImageInput_RequestShape(t *testing.T) {
	var capturedBody ocrRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := decodeRawRequest(r)
		capturedBody = raw
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ocrResponse{Model: "mistral-ocr-4-0", Pages: []ocrPageObject{{Index: 0, Markdown: "ok"}}})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil)
	_, err := c.Process(context.Background(), DocumentInput{
		Data:     []byte("fake png bytes"),
		MIMEType: "image/png",
		IsImage:  true,
	}, ProcessOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chunk, ok := capturedBody.Document.(map[string]any)
	if !ok {
		t.Fatalf("document chunk did not decode as object: %#v", capturedBody.Document)
	}
	if chunk["type"] != "image_url" {
		t.Errorf("document chunk type = %v, want image_url", chunk["type"])
	}
	imgURL, _ := chunk["image_url"].(string)
	if !strings.HasPrefix(imgURL, "data:image/png;base64,") {
		t.Errorf("image_url = %q, want data:image/png;base64,... prefix", imgURL)
	}
}

func TestProcess_PageCapEnforced_NoHTTPCall(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) { cfg.PageCap = 10 })
	_, err := c.Process(context.Background(), DocumentInput{URL: "https://example.test/big.pdf"}, ProcessOptions{
		Pages: "1-500",
	})
	if err == nil {
		t.Fatalf("expected an error for a 500-page request against a 10-page cap")
	}
	if callCount != 0 {
		t.Errorf("HTTP server was called %d times, want 0 (client-side rejection)", callCount)
	}
}

func TestProcess_Timeout_ReturnsError_NotHang(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // never responds until the test cleans up
	}))
	defer func() {
		close(block)
		srv.Close()
	}()

	c := newTestClient(t, srv, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Process(ctx, DocumentInput{URL: "https://example.test/slow.pdf"}, ProcessOptions{})
	if err == nil {
		t.Fatalf("expected a timeout error, got nil")
	}
}

func TestProcess_ErrorClasses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		envelope   errorEnvelope
		withSchema bool
		checkAs    func(err error) bool
	}{
		{
			name:       "auth 401",
			statusCode: http.StatusUnauthorized,
			envelope:   errorEnvelope{Object: "error", Type: "authentication_error", Message: "invalid API key", Code: "invalid_api_key"},
			checkAs:    func(err error) bool { var e *AuthError; return errors.As(err, &e) },
		},
		{
			name:       "auth 403",
			statusCode: http.StatusForbidden,
			envelope:   errorEnvelope{Object: "error", Type: "permission_error", Message: "forbidden"},
			checkAs:    func(err error) bool { var e *AuthError; return errors.As(err, &e) },
		},
		{
			name:       "quota 429",
			statusCode: http.StatusTooManyRequests,
			envelope:   errorEnvelope{Object: "error", Type: "rate_limit_error", Message: "rate limited", Code: "rate_limit_exceeded"},
			checkAs:    func(err error) bool { var e *QuotaError; return errors.As(err, &e) },
		},
		{
			name:       "too large 413",
			statusCode: http.StatusRequestEntityTooLarge,
			envelope:   errorEnvelope{Object: "error", Type: "invalid_request_error", Message: "file too large", Code: "file_too_large"},
			checkAs:    func(err error) bool { var e *TooLargeError; return errors.As(err, &e) },
		},
		{
			name:       "too large via 422 message",
			statusCode: http.StatusUnprocessableEntity,
			envelope:   errorEnvelope{Object: "error", Type: "invalid_request_error", Message: "document exceeds max_pages limit"},
			checkAs:    func(err error) bool { var e *TooLargeError; return errors.As(err, &e) },
		},
		{
			name:       "schema mismatch 422",
			statusCode: http.StatusUnprocessableEntity,
			envelope:   errorEnvelope{Object: "error", Type: "invalid_request_error", Message: "schema validation failed", Param: "document_annotation_format"},
			withSchema: true,
			checkAs:    func(err error) bool { var e *SchemaMismatchError; return errors.As(err, &e) },
		},
		{
			name:       "plain server error 500",
			statusCode: http.StatusInternalServerError,
			envelope:   errorEnvelope{Object: "error", Type: "server_error", Message: "internal error"},
			checkAs: func(err error) bool {
				var auth *AuthError
				var quota *QuotaError
				var large *TooLargeError
				var mismatch *SchemaMismatchError
				var api *APIError
				return !errors.As(err, &auth) && !errors.As(err, &quota) && !errors.As(err, &large) &&
					!errors.As(err, &mismatch) && errors.As(err, &api)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.envelope)
			}))
			defer srv.Close()

			c := newTestClient(t, srv, nil)
			opts := ProcessOptions{}
			if tt.withSchema {
				opts.Schema = &DocumentSchema{Name: "x", Schema: map[string]any{"type": "object"}}
			}
			_, err := c.Process(context.Background(), DocumentInput{URL: "https://example.test/doc.pdf"}, opts)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !tt.checkAs(err) {
				t.Errorf("error %v (%#v) did not match expected class %s", err, err, tt.name)
			}
		})
	}
}

func TestBuildDocumentChunk_RejectsAmbiguousOrEmptyInput(t *testing.T) {
	c := NewClient(Config{APIKey: "k"})

	if _, err := c.buildDocumentChunk(DocumentInput{}); err == nil {
		t.Errorf("expected error for empty DocumentInput, got nil")
	}
	if _, err := c.buildDocumentChunk(DocumentInput{URL: "https://x", Data: []byte("y"), MIMEType: "application/pdf"}); err == nil {
		t.Errorf("expected error when both URL and Data are set, got nil")
	}
	if _, err := c.buildDocumentChunk(DocumentInput{Data: []byte("y")}); err == nil {
		t.Errorf("expected error when Data is set without MIMEType, got nil")
	}
}

func TestParsePageCount(t *testing.T) {
	cases := []struct {
		in     string
		want   int
		wantOK bool
	}{
		{"", 0, false},
		{"1", 1, true},
		{"1,2,3", 3, true},
		{"1-5", 5, true},
		{"1-5,10", 6, true},
		{"5-1", 0, false},
		{"not-a-number", 0, false},
	}
	for _, tc := range cases {
		got, ok := parsePageCount(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("parsePageCount(%q) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

// decodeRawRequest decodes the request body into ocrRequest, but with Document left as a raw
// map[string]any (rather than a concrete chunk struct) so tests can assert on wire field names
// without depending on the client's own decoding.
func decodeRawRequest(r *http.Request) (ocrRequest, error) {
	defer r.Body.Close()
	var req ocrRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		return req, err
	}
	return req, nil
}
