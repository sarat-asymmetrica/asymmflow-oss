package intake

import (
	"encoding/json"
	"os"
	"testing"
)

func TestFromInboxProcessResultNormalizesSupplierInvoice(t *testing.T) {
	var input InboxProcessResultInput
	readFixture(t, "testdata/inbox_process_result.json", &input)

	candidate := FromInboxProcessResult(input)

	if candidate.SourceKind != SourceKindPDF {
		t.Fatalf("source kind = %q, want %q", candidate.SourceKind, SourceKindPDF)
	}
	if candidate.BusinessObjectType != "supplier_invoice" {
		t.Fatalf("business object type = %q", candidate.BusinessObjectType)
	}
	if candidate.ReviewStatus != ReviewStatusNew {
		t.Fatalf("review status = %q, want new", candidate.ReviewStatus)
	}
	if candidate.Confidence != 0.91 {
		t.Fatalf("confidence = %v", candidate.Confidence)
	}
	assertField(t, candidate, "invoice_number", "INV-1042", FieldStatusExtracted)
	if len(candidate.SuggestedLinks) != 1 {
		t.Fatalf("suggested links = %d, want 1", len(candidate.SuggestedLinks))
	}
	if candidate.SuggestedLinks[0].RequiredDeterministicService != "finance.supplier_invoice.review" {
		t.Fatalf("deterministic service = %q", candidate.SuggestedLinks[0].RequiredDeterministicService)
	}
}

func TestFromInboxDocumentMapsReviewAndMissingFields(t *testing.T) {
	var input InboxDocumentInput
	readFixture(t, "testdata/inbox_document.json", &input)

	candidate := FromInboxDocument(input)

	if candidate.SourceKind != SourceKindEmail {
		t.Fatalf("source kind = %q, want email", candidate.SourceKind)
	}
	if candidate.BusinessObjectType != "opportunity" {
		t.Fatalf("business object type = %q", candidate.BusinessObjectType)
	}
	if candidate.ReviewStatus != ReviewStatusNeedsReview {
		t.Fatalf("review status = %q, want needs_review", candidate.ReviewStatus)
	}
	assertField(t, candidate, "bid_deadline", "", FieldStatusMissing)
	if len(candidate.Warnings) == 0 {
		t.Fatalf("expected warnings for low-confidence inbox row")
	}
}

func TestFromOCRExtractionSkipsRawTextAndMarksLowConfidence(t *testing.T) {
	var input OCRExtractionInput
	readFixture(t, "testdata/ocr_extraction.json", &input)

	candidate := FromOCRExtraction(input)

	if candidate.SourceKind != SourceKindExcel {
		t.Fatalf("source kind = %q, want excel", candidate.SourceKind)
	}
	if candidate.BusinessObjectType != "bank_statement" {
		t.Fatalf("business object type = %q", candidate.BusinessObjectType)
	}
	if candidate.ReviewStatus != ReviewStatusNeedsReview {
		t.Fatalf("review status = %q, want needs_review", candidate.ReviewStatus)
	}
	assertField(t, candidate, "account_number", "12345678", FieldStatusNeedsConfirmation)
	for _, field := range candidate.ExtractedFields {
		if field.Name == "raw_text" {
			t.Fatalf("raw_text should not become an extracted field")
		}
	}
}

func TestNormalizeExtractionMapHandlesNestedFieldStatusAndPercentConfidence(t *testing.T) {
	var input map[string]any
	readFixture(t, "testdata/ocr_extraction_map.json", &input)

	candidate := NormalizeExtractionMap(input)

	if candidate.SourceKind != SourceKindScreenshot {
		t.Fatalf("source kind = %q, want screenshot", candidate.SourceKind)
	}
	if candidate.BusinessObjectType != "supplier_invoice" {
		t.Fatalf("business object type = %q", candidate.BusinessObjectType)
	}
	if candidate.Confidence != 0.72 {
		t.Fatalf("confidence = %v, want 0.72", candidate.Confidence)
	}
	if candidate.ReviewStatus != ReviewStatusNeedsReview {
		t.Fatalf("review status = %q, want needs_review", candidate.ReviewStatus)
	}
	assertField(t, candidate, "invoice_number", "SI-8841", FieldStatusExtracted)
	assertField(t, candidate, "po_number", "PO-771", FieldStatusNeedsConfirmation)
	assertField(t, candidate, "vendor_vat_number", "", FieldStatusMissing)
	if len(candidate.Warnings) == 0 {
		t.Fatal("expected OCR map warning to be preserved")
	}
}

func TestReviewStatusFromInboxStatus(t *testing.T) {
	tests := map[string]ReviewStatus{
		"Processed":   ReviewStatusLinked,
		"Archived":    ReviewStatusArchived,
		"Rejected":    ReviewStatusRejected,
		"Corrected":   ReviewStatusCorrected,
		"NeedsReview": ReviewStatusNeedsReview,
		"Ready":       ReviewStatusNew,
	}

	for input, want := range tests {
		if got := ReviewStatusFromInboxStatus(input, 0.95); got != want {
			t.Fatalf("ReviewStatusFromInboxStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func readFixture(t *testing.T, path string, out any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
}

func assertField(t *testing.T, candidate Candidate, name string, value string, status FieldStatus) {
	t.Helper()
	for _, field := range candidate.ExtractedFields {
		if field.Name != name {
			continue
		}
		if field.Value != value || field.Status != status {
			t.Fatalf("field %s = (%q, %q), want (%q, %q)", name, field.Value, field.Status, value, status)
		}
		return
	}
	t.Fatalf("field %s not found in %#v", name, candidate.ExtractedFields)
}
