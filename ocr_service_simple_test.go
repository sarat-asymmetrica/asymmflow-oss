package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
	"ph_holdings_app/pkg/ocr/mistralocr"
)

// Test vector PDF detection
func TestIsVectorPDF(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Non-PDF file", "document.txt", false},
		{"Image file", "scan.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVectorPDF(tt.path)
			if result != tt.expected {
				t.Errorf("isVectorPDF(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

// Test SimpleOCRService creation (Wave 13: Fly.io retired — no endpoint to check; the
// service now carries a local offline-fallback ACEEngine instead).
func TestNewSimpleOCRService(t *testing.T) {
	service, err := NewSimpleOCRService()
	if err != nil {
		t.Fatalf("NewSimpleOCRService() failed: %v", err)
	}

	if service == nil {
		t.Fatal("NewSimpleOCRService() returned nil service")
	}

	if service.httpClient == nil {
		t.Error("httpClient should not be nil")
	}

	if service.maxPages <= 0 {
		t.Error("maxPages should be > 0")
	}

	if service.dpi <= 0 {
		t.Error("dpi should be > 0")
	}

	if service.localEngine == nil {
		t.Error("localEngine (offline tesseract fallback) should not be nil")
	}
}

// Test OCRResultSimple structure
func TestOCRResultSimple(t *testing.T) {
	result := &OCRResultSimple{
		Success:        true,
		Text:           "Sample extracted text",
		Confidence:     0.95,
		DocumentType:   "invoice",
		ExtractedData:  map[string]any{"raw_text": "Sample extracted text"},
		ProcessingTime: 1500,
		Engine:         "pymupdf",
	}

	if !result.Success {
		t.Error("Result should be successful")
	}

	if result.Text == "" {
		t.Error("Text should not be empty")
	}

	if result.Confidence <= 0 || result.Confidence > 1.0 {
		t.Errorf("Confidence should be between 0 and 1, got: %f", result.Confidence)
	}

	if result.Engine != "pymupdf" && result.Engine != "mistral-ocr-4" && result.Engine != "mistral-ocr-4-image" && result.Engine != "tesseract-local" {
		t.Errorf("Unexpected engine: %s", result.Engine)
	}
}

// TestLoadMistralOCREnvDefaults verifies config-not-constant defaults resolve when no env vars
// are set, and that env overrides are honored (verify-the-probe: asserts concrete values, not
// just "no error").
func TestLoadMistralOCREnvDefaults(t *testing.T) {
	for _, key := range []string{"MISTRAL_OCR_MODEL", "MISTRAL_OCR_BASE_URL", "MISTRAL_OCR_PAGE_CAP", "MISTRAL_OCR_TIMEOUT_SECONDS", "MISTRAL_OCR_CONFIDENCE_THRESHOLD", "OCR_SCANT_TEXT_THRESHOLD"} {
		t.Setenv(key, "")
	}

	cfg := loadMistralOCREnv()
	if cfg.Model != mistralocr.DefaultModel {
		t.Errorf("expected default model %q, got %q", mistralocr.DefaultModel, cfg.Model)
	}
	if cfg.ScantTextThreshold != 50 {
		t.Errorf("expected default scant-text threshold 50, got %d", cfg.ScantTextThreshold)
	}
	if cfg.ConfidenceThreshold != mistralocr.DefaultConfidenceThreshold {
		t.Errorf("expected default confidence threshold %v, got %v", mistralocr.DefaultConfidenceThreshold, cfg.ConfidenceThreshold)
	}

	t.Setenv("MISTRAL_OCR_MODEL", "mistral-ocr-test-override")
	t.Setenv("OCR_SCANT_TEXT_THRESHOLD", "120")
	overridden := loadMistralOCREnv()
	if overridden.Model != "mistral-ocr-test-override" {
		t.Errorf("expected env override for model, got %q", overridden.Model)
	}
	if overridden.ScantTextThreshold != 120 {
		t.Errorf("expected env override for scant-text threshold, got %d", overridden.ScantTextThreshold)
	}
}

// TestSchemaForDocType verifies each normalized document type resolves to its intended schema
// (RFQ/invoice/PO/bank-statement/generic per the wave 13 spec) — a wrong mapping here would
// silently ask Mistral to extract the wrong fields, so this asserts concrete schema names.
func TestSchemaForDocType(t *testing.T) {
	tests := []struct {
		docType      string
		expectedName string
	}{
		{"invoice", "invoice_extraction"},
		{"supplier_invoice", "invoice_extraction"},
		{"rfq", "rfq_extraction"},
		{"quotation", "rfq_extraction"},
		{"purchase_order", "purchase_order_extraction"},
		{"po", "purchase_order_extraction"},
		{"bank_statement", "bank_statement_extraction"},
		{"delivery_note", "generic_document_extraction"},
		{"auto", "generic_document_extraction"},
		{"", "generic_document_extraction"},
	}

	for _, tt := range tests {
		t.Run(tt.docType, func(t *testing.T) {
			schema := schemaForDocType(tt.docType)
			if schema == nil {
				t.Fatalf("schemaForDocType(%q) returned nil", tt.docType)
			}
			if schema.Name != tt.expectedName {
				t.Errorf("schemaForDocType(%q).Name = %q, expected %q", tt.docType, schema.Name, tt.expectedName)
			}
			props, ok := schema.Schema["properties"].(map[string]any)
			if !ok || len(props) == 0 {
				t.Errorf("schemaForDocType(%q) has no properties", tt.docType)
			}
		})
	}
}

// TestMimeTypeForOCRFile verifies MIME detection used when submitting to Mistral OCR 4.
func TestMimeTypeForOCRFile(t *testing.T) {
	tests := []struct {
		path     string
		isImage  bool
		expected string
	}{
		{"doc.pdf", false, "application/pdf"},
		{"scan.png", true, "image/png"},
		{"scan.jpg", true, "image/jpeg"},
		{"scan.jpeg", true, "image/jpeg"},
		{"scan.bmp", true, "image/bmp"},
		{"scan.tiff", true, "image/tiff"},
		{"scan.webp", true, "image/webp"},
	}
	for _, tt := range tests {
		if got := mimeTypeForOCRFile(tt.path, tt.isImage); got != tt.expected {
			t.Errorf("mimeTypeForOCRFile(%q, %v) = %q, expected %q", tt.path, tt.isImage, got, tt.expected)
		}
	}
}

// TestOCRWithLocalEngineOfflineFallback proves the offline-first hard boundary: with no
// Mistral key configured (the default in this test environment) and a scanned-looking PDF,
// the local engine path still returns a result rather than erroring the whole document out.
func TestOCRWithLocalEngineOfflineFallback(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "scan.pdf")
	// Not a real PDF — the point is to prove ocrWithLocalEngine degrades gracefully to a
	// low-confidence, NeedsReview result instead of blocking the caller, even when the local
	// tesseract/pandoc tools can't make sense of the bytes.
	if err := os.WriteFile(path, []byte("%PDF-1.4\nnot a real scanned document"), 0644); err != nil {
		t.Fatal(err)
	}

	service, err := NewSimpleOCRService()
	if err != nil {
		t.Fatalf("NewSimpleOCRService() failed: %v", err)
	}

	result, err := service.ocrWithLocalEngine(path, "generic", "partial go-fitz text")
	if err != nil {
		t.Fatalf("ocrWithLocalEngine should never hard-fail on a readable file, got: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true even in degraded offline mode")
	}
	if !result.NeedsReview {
		t.Error("offline/degraded-mode results must always be marked NeedsReview")
	}
	if result.Engine != "tesseract-local" {
		t.Errorf("expected engine=tesseract-local, got %q", result.Engine)
	}
}

func TestSimpleOCRServiceProcessesExcelLocally(t *testing.T) {
	tmpDir := t.TempDir()
	workbookPath := filepath.Join(tmpDir, "rfq_fixture.xlsx")

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	if err := f.SetCellValue(sheet, "A1", "Request for Quotation"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue(sheet, "A2", "RFQ Number"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue(sheet, "B2", "RFQ-2026-LOCAL"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue(sheet, "A3", "Please quote analyzer spares"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(workbookPath); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	service, err := NewSimpleOCRService()
	if err != nil {
		t.Fatalf("NewSimpleOCRService() failed: %v", err)
	}

	result, err := service.ProcessDocument(workbookPath, "rfq")
	if err != nil {
		t.Fatalf("ProcessDocument() failed: %v", err)
	}
	if !result.Success {
		t.Fatal("Excel OCR result should be successful")
	}
	if result.Engine != "excelize" {
		t.Fatalf("expected excelize engine, got %s", result.Engine)
	}
	if result.DocumentType != "rfq" {
		t.Fatalf("expected rfq document type, got %s", result.DocumentType)
	}
	if !strings.Contains(result.Text, "Request for Quotation") {
		t.Fatalf("expected extracted workbook text, got %q", result.Text)
	}
	if result.ExtractedData["source_type"] != "excel" {
		t.Fatalf("expected source_type=excel, got %#v", result.ExtractedData["source_type"])
	}
}

// Benchmark vector PDF detection (should be fast!)
func BenchmarkIsVectorPDF(b *testing.B) {
	// Test with a non-PDF file (fast path)
	path := "document.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isVectorPDF(path)
	}
}
