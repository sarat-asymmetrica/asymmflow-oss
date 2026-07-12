package intake

import (
	"context"
	"testing"
	"time"
)

func TestMemoryReviewRecordRepositorySaveListGetAndIdempotency(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryReviewRecordRepository()
	record := ReviewRecord{
		ID:                           "review-1",
		CandidateID:                  "candidate-1",
		SourceID:                     "source-1",
		Decision:                     ReviewDecisionAcceptProposal,
		ReviewStatus:                 ReviewStatusLinked,
		ProposedDeterministicService: "finance.invoice.review",
		Actor:                        "operator",
		CorrelationID:                "corr-1",
		CreatedAt:                    time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC),
	}

	saved, err := repo.Save(ctx, record)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	duplicate := record
	duplicate.ID = "review-duplicate"
	duplicate.CreatedAt = duplicate.CreatedAt.Add(time.Hour)
	savedAgain, err := repo.Save(ctx, duplicate)
	if err != nil {
		t.Fatalf("duplicate Save returned error: %v", err)
	}
	if savedAgain.ID != saved.ID {
		t.Fatalf("duplicate save should return first record id %q, got %q", saved.ID, savedAgain.ID)
	}

	got, ok, err := repo.Get(ctx, "review-1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !ok || got.ID != record.ID {
		t.Fatalf("Get should find review-1, got ok=%v record=%+v", ok, got)
	}

	list, err := repo.ListByCandidate(ctx, "candidate-1")
	if err != nil {
		t.Fatalf("ListByCandidate returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one idempotent record, got %d", len(list))
	}
}

func TestValidateReviewRecordRejectsInvalidDecision(t *testing.T) {
	err := ValidateReviewRecord(ReviewRecord{
		ID:            "review-1",
		CandidateID:   "candidate-1",
		Decision:      ReviewDecision("mutate_authority"),
		ReviewStatus:  ReviewStatusLinked,
		Actor:         "operator",
		CorrelationID: "corr-1",
	})
	if err == nil {
		t.Fatal("expected invalid decision to be rejected")
	}
}
