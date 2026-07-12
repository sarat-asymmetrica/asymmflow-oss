package intake

import (
	"context"
	"testing"
	"time"
)

func TestReviewServiceRecordsOperatorDecisionIdempotently(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 14, 13, 0, 0, 0, time.UTC)
	service, err := NewReviewService(NewMemoryReviewRecordRepository(), Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}

	command := ReviewCommand{
		Candidate:                    serviceCandidate(),
		Decision:                     ReviewDecisionAcceptProposal,
		Actor:                        "casey",
		ActorType:                    ReviewActorOperator,
		Reason:                       "source verified",
		ProposedDeterministicService: "finance.invoice.review",
		CorrelationID:                "corr-1",
	}
	record, err := service.RecordDecision(ctx, command)
	if err != nil {
		t.Fatalf("RecordDecision returned error: %v", err)
	}
	duplicate, err := service.RecordDecision(ctx, command)
	if err != nil {
		t.Fatalf("duplicate RecordDecision returned error: %v", err)
	}
	if duplicate.ID != record.ID {
		t.Fatalf("duplicate decision should return existing record id %q, got %q", record.ID, duplicate.ID)
	}
	if record.Actor != "casey" || record.CorrelationID != "corr-1" {
		t.Fatalf("record did not preserve actor/correlation: %+v", record)
	}
	if !record.CreatedAt.Equal(now) {
		t.Fatalf("record time = %s, want %s", record.CreatedAt, now)
	}
}

func TestReviewServiceRejectsAgentMutation(t *testing.T) {
	service, err := NewReviewService(NewMemoryReviewRecordRepository())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	_, err = service.RecordDecision(context.Background(), ReviewCommand{
		Candidate:     serviceCandidate(),
		Decision:      ReviewDecisionAcceptProposal,
		Actor:         "butler",
		ActorType:     ReviewActorAgent,
		CorrelationID: "corr-agent",
	})
	if err == nil {
		t.Fatal("expected agent-originated mutation to be rejected")
	}
}

func TestReviewServiceRequiresActorCorrelationAndSource(t *testing.T) {
	service, err := NewReviewService(NewMemoryReviewRecordRepository())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	candidate := serviceCandidate()
	candidate.Source.ID = ""

	_, err = service.RecordDecision(context.Background(), ReviewCommand{
		Candidate: candidate,
		Decision:  ReviewDecisionNeedsInput,
		Actor:     "operator",
	})
	if err == nil {
		t.Fatal("expected missing correlation/source validation error")
	}
}

func TestReviewServiceBuildsQueueStateAndContextPack(t *testing.T) {
	ctx := context.Background()
	service, err := NewReviewService(NewMemoryReviewRecordRepository())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	candidate := serviceCandidate()
	if _, err := service.RecordDecision(ctx, ReviewCommand{
		Candidate:     candidate,
		Decision:      ReviewDecisionNeedsInput,
		Actor:         "operator",
		CorrelationID: "corr-state",
	}); err != nil {
		t.Fatalf("RecordDecision returned error: %v", err)
	}

	state, err := service.BuildQueueState(ctx, candidate)
	if err != nil {
		t.Fatalf("BuildQueueState returned error: %v", err)
	}
	if len(state.Records) != 1 || state.LastReview == nil {
		t.Fatalf("expected review state with one last review, got %+v", state)
	}
	if state.ContextPack.CandidateID != candidate.ID {
		t.Fatalf("context pack candidate id = %q, want %q", state.ContextPack.CandidateID, candidate.ID)
	}
}

func TestReviewServiceBuildsQueueStateWithSourceRegistryProvenance(t *testing.T) {
	ctx := context.Background()
	service, err := NewReviewService(NewMemoryReviewRecordRepository())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	sourceRepo := NewMemorySourceAssetRepository()
	asset, err := NewSourceAsset(SourceAssetInput{
		Kind:             SourceKindPDF,
		Path:             "C:\\Inbox\\invoice.pdf",
		Label:            "invoice.pdf",
		PrivacyClass:     SourcePrivacyConfidential,
		ProcessingStatus: SourceStatusReviewed,
		CandidateIDs:     []string{"candidate-service-1"},
		AuditRefs:        []AuditRef{{Type: "inbox", SourceID: "source-service-1", Summary: "stored source"}},
		SeenAt:           time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewSourceAsset returned error: %v", err)
	}
	asset.ID = "source-service-1"
	if _, _, err := sourceRepo.Upsert(ctx, asset); err != nil {
		t.Fatalf("source Upsert returned error: %v", err)
	}

	state, err := service.BuildQueueStateWithSources(ctx, serviceCandidate(), sourceRepo)
	if err != nil {
		t.Fatalf("BuildQueueStateWithSources returned error: %v", err)
	}
	if len(state.SourceAssets) != 1 || state.SourceAssets[0].ID != "source-service-1" {
		t.Fatalf("source assets = %+v", state.SourceAssets)
	}
}

func serviceCandidate() Candidate {
	return Candidate{
		ID:                 "candidate-service-1",
		Source:             SourceRef{ID: "source-service-1", Label: "invoice.pdf", Kind: SourceKindPDF},
		SourceKind:         SourceKindPDF,
		BusinessObjectType: "customer_invoice",
		Classification:     Classification{Type: "invoice", Confidence: 0.92},
		ExtractedFields: []ExtractedField{
			{Name: "invoice_number", Label: "Invoice Number", Value: "INV-1", Status: FieldStatusExtracted, Confidence: 0.95, Source: "ocr"},
		},
		SuggestedLinks: []SuggestedLink{
			{ID: "link-service-1", Label: "Review invoice", Reason: "invoice candidate", BusinessObjectType: "customer_invoice", RequiredDeterministicService: "finance.invoice.review"},
		},
		ReviewStatus: ReviewStatusNeedsReview,
		Confidence:   0.92,
	}
}
