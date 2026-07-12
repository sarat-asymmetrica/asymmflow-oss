package intake

import (
	"testing"
	"time"
)

func TestNewReviewRecordMapsDecisionToReviewStatus(t *testing.T) {
	candidate := Candidate{
		ID: "cand-review-1",
		Source: SourceRef{
			ID:    "doc-1",
			Label: "Supplier invoice",
			Kind:  SourceKindPDF,
		},
		SourceKind:         SourceKindPDF,
		BusinessObjectType: "supplier_invoice",
		SuggestedLinks: []SuggestedLink{
			{RequiredDeterministicService: "finance.supplier_invoice.review"},
		},
	}

	record, err := NewReviewRecord(ReviewRecordInput{
		Candidate: candidate,
		Decision:  ReviewDecisionAcceptProposal,
		Actor:     "tester",
		Reason:    "fields checked",
		Now:       time.Date(2026, 5, 14, 16, 40, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewReviewRecord returned error: %v", err)
	}

	if record.ReviewStatus != ReviewStatusLinked {
		t.Fatalf("review status = %q, want linked", record.ReviewStatus)
	}
	if record.ProposedDeterministicService != "finance.supplier_invoice.review" {
		t.Fatalf("service = %q", record.ProposedDeterministicService)
	}
	if record.CorrelationID == "" {
		t.Fatal("correlation id should be generated")
	}
}

func TestReviewQueueSaveIsIdempotent(t *testing.T) {
	queue := NewReviewQueue()
	record := ReviewRecord{
		ID:                           "review-1",
		CandidateID:                  "cand-1",
		Decision:                     ReviewDecisionNeedsInput,
		ReviewStatus:                 ReviewStatusNeedsReview,
		ProposedDeterministicService: "documents.intake.review",
		CorrelationID:                "corr-1",
		CreatedAt:                    time.Date(2026, 5, 14, 16, 40, 0, 0, time.UTC),
	}
	duplicate := record
	duplicate.ID = "review-duplicate"
	duplicate.CreatedAt = duplicate.CreatedAt.Add(time.Minute)

	first := queue.Save(record)
	second := queue.Save(duplicate)

	if first.ID != "review-1" || second.ID != "review-1" {
		t.Fatalf("idempotent save returned %#v then %#v", first, second)
	}
	records := queue.ListByCandidate("cand-1")
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
}

func TestReviewStatusForDecisionRejectsUnsupportedDecision(t *testing.T) {
	if _, err := ReviewStatusForDecision("approve"); err == nil {
		t.Fatal("unsupported authority-like decision should fail")
	}
}
