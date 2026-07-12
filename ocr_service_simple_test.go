package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
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

// Test SimpleOCRService creation
func TestNewSimpleOCRService(t *testing.T) {
	service, err := NewSimpleOCRService()
	if err != nil {
		t.Fatalf("NewSimpleOCRService() failed: %v", err)
	}

	if service == nil {
		t.Fatal("NewSimpleOCRService() returned nil service")
	}

	if service.flyEndpoint == "" {
		t.Error("flyEndpoint should not be empty")
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

	// Verify default endpoint
	expectedEndpoint := "https://asymmetrica-runtime.fly.dev"
	if service.flyEndpoint != expectedEndpoint {
		t.Errorf("Expected endpoint %s, got %s", expectedEndpoint, service.flyEndpoint)
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

	if result.Engine != "pymupdf" && result.Engine != "fly-runtime" {
		t.Errorf("Unexpected engine: %s", result.Engine)
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
