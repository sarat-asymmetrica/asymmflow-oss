package main

import (
	"testing"

	"ph_holdings_app/pkg/documents/intake"
)

func TestParseBusinessMemoryReviewDecision(t *testing.T) {
	cases := map[string]intake.ReviewDecision{
		"accept proposal": intake.ReviewDecisionAcceptProposal,
		"approved":        intake.ReviewDecisionAcceptProposal,
		"needs-input":     intake.ReviewDecisionNeedsInput,
		"corrected":       intake.ReviewDecisionCorrectField,
		"reject":          intake.ReviewDecisionRejectCandidate,
		"archived":        intake.ReviewDecisionArchive,
	}
	for input, want := range cases {
		got, err := parseBusinessMemoryReviewDecision(input)
		if err != nil {
			t.Fatalf("parseBusinessMemoryReviewDecision(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("parseBusinessMemoryReviewDecision(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseBusinessMemoryReviewDecisionRejectsAuthorityMutation(t *testing.T) {
	if _, err := parseBusinessMemoryReviewDecision("post journal"); err == nil {
		t.Fatal("expected unsupported authority mutation to be rejected")
	}
}

func TestInboxDocumentToBusinessMemoryCandidate(t *testing.T) {
	candidate := inboxDocumentToBusinessMemoryCandidate(InboxDocument{
		DocumentID:   "doc-1",
		FileName:     "invoice.pdf",
		FilePath:     "C:\\Inbox\\invoice.pdf",
		DocumentType: "invoice",
		Status:       "NeedsReview",
		Confidence:   0.74,
		ExtractedData: map[string]string{
			"invoice_number": "INV-1",
		},
		SuggestedActions: []string{"Review invoice"},
	})

	if candidate.ID != "doc-1" {
		t.Fatalf("candidate id = %q, want doc-1", candidate.ID)
	}
	if candidate.BusinessObjectType != "customer_invoice" {
		t.Fatalf("business object type = %q, want customer_invoice", candidate.BusinessObjectType)
	}
	if firstBusinessMemoryDeterministicTarget(candidate) != "finance.invoice.review" {
		t.Fatalf("unexpected deterministic target: %+v", candidate.SuggestedLinks)
	}
}

func TestCandidateToBusinessMemorySourceAsset(t *testing.T) {
	candidate := inboxDocumentToBusinessMemoryCandidate(InboxDocument{
		DocumentID:   "doc-1",
		FileName:     "invoice.pdf",
		FilePath:     "C:\\Inbox\\invoice.pdf",
		DocumentType: "invoice",
		Status:       "NeedsReview",
		Confidence:   0.74,
	})

	asset := candidateToBusinessMemorySourceAsset(candidate)
	if asset.ID != candidate.Source.ID {
		t.Fatalf("source asset id = %q, want %q", asset.ID, candidate.Source.ID)
	}
	if asset.Kind != intake.SourceKindPDF {
		t.Fatalf("source asset kind = %q", asset.Kind)
	}
	if asset.ProcessingStatus != intake.SourceStatusCandidateGenerated {
		t.Fatalf("source asset status = %q", asset.ProcessingStatus)
	}
	if len(asset.CandidateIDs) != 1 || asset.CandidateIDs[0] != candidate.ID {
		t.Fatalf("candidate ids = %+v", asset.CandidateIDs)
	}
}
