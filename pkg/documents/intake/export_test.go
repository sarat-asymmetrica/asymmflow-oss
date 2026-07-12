package intake

import (
	"strings"
	"testing"
	"time"
)

func TestReviewExportBundleJSONReplay(t *testing.T) {
	exportedAt := time.Date(2026, 5, 14, 18, 0, 0, 0, time.UTC)
	bundle, err := NewReviewExportBundle(exportCandidate(), []ReviewRecord{exportReviewRecord()}, exportedAt)
	if err != nil {
		t.Fatalf("NewReviewExportBundle returned error: %v", err)
	}

	payload, err := ExportReviewBundleJSON(bundle)
	if err != nil {
		t.Fatalf("ExportReviewBundleJSON returned error: %v", err)
	}
	replayed, err := ReplayReviewBundleJSON(payload)
	if err != nil {
		t.Fatalf("ReplayReviewBundleJSON returned error: %v", err)
	}

	if replayed.SchemaVersion != ReviewExportSchemaVersion {
		t.Fatalf("schema = %q", replayed.SchemaVersion)
	}
	if replayed.Candidate.ID != "export-candidate" {
		t.Fatalf("candidate id = %q", replayed.Candidate.ID)
	}
	if len(replayed.ReviewRecords) != 1 || replayed.ReviewRecords[0].CorrelationID != "corr-export" {
		t.Fatalf("review records = %+v", replayed.ReviewRecords)
	}
	if replayed.ContextPack.CandidateID != "export-candidate" {
		t.Fatalf("context pack candidate id = %q", replayed.ContextPack.CandidateID)
	}
}

func TestReviewExportBundleTOONIncludesAgentBoundaries(t *testing.T) {
	bundle, err := NewReviewExportBundle(exportCandidate(), []ReviewRecord{exportReviewRecord()}, time.Date(2026, 5, 14, 18, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewReviewExportBundle returned error: %v", err)
	}
	text := ExportReviewBundleTOON(bundle)
	required := []string{
		"business_memory_review_bundle:",
		"schema_version: business-memory-review-bundle/v1",
		"candidate_id: export-candidate",
		"allowed_agent_actions:",
		"forbidden_agent_actions:",
		"decision: accept_proposal",
		"correlation_id: corr-export",
	}
	for _, snippet := range required {
		if !strings.Contains(text, snippet) {
			t.Fatalf("TOON export missing %q:\n%s", snippet, text)
		}
	}
}

func TestReplayReviewBundleJSONRejectsSchemaMismatch(t *testing.T) {
	_, err := ReplayReviewBundleJSON([]byte(`{"schema_version":"unknown","candidate":{"id":"candidate"}}`))
	if err == nil {
		t.Fatal("expected schema mismatch to be rejected")
	}
}

func TestReviewExportBundleIncludesSourceRegistryProvenance(t *testing.T) {
	sourceAsset := exportSourceAsset()
	bundle, err := NewReviewExportBundleWithSources(exportCandidate(), []ReviewRecord{exportReviewRecord()}, []SourceAsset{sourceAsset}, time.Date(2026, 5, 14, 18, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewReviewExportBundleWithSources returned error: %v", err)
	}
	payload, err := ExportReviewBundleJSON(bundle)
	if err != nil {
		t.Fatalf("ExportReviewBundleJSON returned error: %v", err)
	}
	replayed, err := ReplayReviewBundleJSON(payload)
	if err != nil {
		t.Fatalf("ReplayReviewBundleJSON returned error: %v", err)
	}
	if len(replayed.SourceAssets) != 1 || replayed.SourceAssets[0].ID != sourceAsset.ID {
		t.Fatalf("source assets = %+v", replayed.SourceAssets)
	}

	text := ExportReviewBundleTOON(replayed)
	required := []string{
		"source_assets:",
		"id: source-export",
		"privacy_class: confidential",
		"processing_status: reviewed",
		"candidate_ids: export-candidate",
	}
	for _, snippet := range required {
		if !strings.Contains(text, snippet) {
			t.Fatalf("TOON export missing source registry provenance %q:\n%s", snippet, text)
		}
	}
}

func exportCandidate() Candidate {
	return Candidate{
		ID:                 "export-candidate",
		Source:             SourceRef{ID: "source-export", Label: "invoice.pdf", Kind: SourceKindPDF},
		SourceKind:         SourceKindPDF,
		BusinessObjectType: "customer_invoice",
		Classification:     Classification{Type: "invoice", Confidence: 0.92},
		ExtractedFields: []ExtractedField{
			{Name: "invoice_number", Label: "Invoice Number", Value: "INV-EXPORT", Status: FieldStatusExtracted, Confidence: 0.93, Source: "ocr"},
		},
		SuggestedLinks: []SuggestedLink{{
			ID:                           "export-link",
			Label:                        "Review invoice",
			Reason:                       "invoice candidate",
			BusinessObjectType:           "customer_invoice",
			RequiredDeterministicService: "finance.invoice.review",
		}},
		ReviewStatus: ReviewStatusNeedsReview,
		Confidence:   0.92,
	}
}

func exportSourceAsset() SourceAsset {
	return SourceAsset{
		ID:               "source-export",
		Kind:             SourceKindPDF,
		Path:             "C:\\Inbox\\invoice.pdf",
		Label:            "invoice.pdf",
		Hash:             "sha256:export",
		ImportBatchID:    "batch-export",
		PrivacyClass:     SourcePrivacyConfidential,
		ProcessingStatus: SourceStatusReviewed,
		CandidateIDs:     []string{"export-candidate"},
		AuditRefs:        []AuditRef{{Type: "review", SourceID: "review-export", Summary: "operator reviewed source"}},
		FirstSeenAt:      time.Date(2026, 5, 14, 16, 0, 0, 0, time.UTC),
		LastSeenAt:       time.Date(2026, 5, 14, 17, 0, 0, 0, time.UTC),
	}
}

func exportReviewRecord() ReviewRecord {
	return ReviewRecord{
		ID:                           "review-export",
		CandidateID:                  "export-candidate",
		SourceID:                     "source-export",
		Decision:                     ReviewDecisionAcceptProposal,
		ReviewStatus:                 ReviewStatusLinked,
		ProposedDeterministicService: "finance.invoice.review",
		Actor:                        "operator",
		Reason:                       "verified for export",
		CorrelationID:                "corr-export",
		CreatedAt:                    time.Date(2026, 5, 14, 17, 59, 0, 0, time.UTC),
	}
}
