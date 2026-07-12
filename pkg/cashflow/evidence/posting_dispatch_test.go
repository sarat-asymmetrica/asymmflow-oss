package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// mockCreateDraft returns a CreateDraftJournalFunc that validates its inputs
// and returns a fixed journal ID.
func mockCreateDraft(wantType, wantID string) CreateDraftJournalFunc {
	return func(sourceType, sourceID string) (string, error) {
		if sourceType != wantType || sourceID != wantID {
			return "", fmt.Errorf("unexpected source: got %s/%s, want %s/%s", sourceType, sourceID, wantType, wantID)
		}
		return "journal_draft_001", nil
	}
}

func TestPostingDispatcherCanDispatch(t *testing.T) {
	d := NewPostingDispatcher(mockCreateDraft("posting_coverage", "src-1"))

	matching := ActionProposal{
		Action:                       "finance.requestDraftJournal",
		RequiredDeterministicService: "finance_posting_service",
	}
	if !d.CanDispatch(matching) {
		t.Fatal("CanDispatch = false, want true for finance_posting_service")
	}

	other := ActionProposal{
		Action:                       "banking.resolveMatch",
		RequiredDeterministicService: "bank_reconciliation_service",
	}
	if d.CanDispatch(other) {
		t.Fatal("CanDispatch = true, want false for bank_reconciliation_service")
	}
}

func TestPostingDispatcherRejectsAgentActor(t *testing.T) {
	d := NewPostingDispatcher(mockCreateDraft("posting_coverage", "src-1"))
	proposal := ActionProposal{
		Action:                       "finance.requestDraftJournal",
		RequiredDeterministicService: "finance_posting_service",
	}
	approval := DispatchApproval{
		Actor:      "ai-agent",
		ActorType:  "agent",
		SourceType: "posting_coverage",
		SourceID:   "src-1",
	}

	_, err := d.Dispatch(context.Background(), proposal, approval)
	if err == nil {
		t.Fatal("expected error when ActorType is agent, got nil")
	}
	if !strings.Contains(err.Error(), "agent") {
		t.Fatalf("error should mention agent, got: %q", err.Error())
	}
}

func TestPostingDispatcherRequiresSourceInfo(t *testing.T) {
	d := NewPostingDispatcher(mockCreateDraft("posting_coverage", "src-1"))
	proposal := ActionProposal{
		Action:                       "finance.requestDraftJournal",
		RequiredDeterministicService: "finance_posting_service",
	}
	base := DispatchApproval{
		Actor:      "operator@example.com",
		ActorType:  "operator",
		ApprovedAt: time.Now(),
	}

	cases := []struct {
		name       string
		sourceType string
		sourceID   string
	}{
		{"both empty", "", ""},
		{"sourceType missing", "", "src-1"},
		{"sourceID missing", "posting_coverage", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			approval := base
			approval.SourceType = tc.sourceType
			approval.SourceID = tc.sourceID
			_, err := d.Dispatch(context.Background(), proposal, approval)
			if err == nil {
				t.Fatal("expected error for missing source info, got nil")
			}
		})
	}
}

func TestPostingDispatcherCallsCreateDraft(t *testing.T) {
	d := NewPostingDispatcher(mockCreateDraft("posting_coverage", "src-42"))
	proposal := ActionProposal{
		Action:                       "finance.requestDraftJournal",
		Label:                        "Review draft posting",
		RequiredDeterministicService: "finance_posting_service",
	}
	approval := DispatchApproval{
		Actor:      "finance@example.com",
		ActorType:  "operator",
		SourceType: "posting_coverage",
		SourceID:   "src-42",
		ApprovedAt: time.Now(),
	}

	result, err := d.Dispatch(context.Background(), proposal, approval)
	if err != nil {
		t.Fatalf("Dispatch returned unexpected error: %v", err)
	}
	if !result.Executed {
		t.Fatal("result.Executed = false, want true")
	}
	if result.OutputRef != "journal_draft_001" {
		t.Fatalf("OutputRef = %q, want %q", result.OutputRef, "journal_draft_001")
	}
	if result.ServiceUsed != "finance_posting_service" {
		t.Fatalf("ServiceUsed = %q, want %q", result.ServiceUsed, "finance_posting_service")
	}
	if !strings.Contains(result.Message, "journal_draft_001") {
		t.Fatalf("Message should mention journal ID, got: %q", result.Message)
	}
}

func TestPostingDispatcherPropagatesCreateDraftError(t *testing.T) {
	wantErr := errors.New("downstream journal service unavailable")
	failing := func(sourceType, sourceID string) (string, error) {
		return "", wantErr
	}
	d := NewPostingDispatcher(failing)
	proposal := ActionProposal{
		Action:                       "finance.requestDraftJournal",
		RequiredDeterministicService: "finance_posting_service",
	}
	approval := DispatchApproval{
		Actor:      "finance@example.com",
		ActorType:  "operator",
		SourceType: "posting_coverage",
		SourceID:   "src-1",
		ApprovedAt: time.Now(),
	}

	_, err := d.Dispatch(context.Background(), proposal, approval)
	if err == nil {
		t.Fatal("expected error from createDraft, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("error chain should contain original error: %v", err)
	}
}

func TestPostingDispatcherIntegrationWithRouter(t *testing.T) {
	var calledWith struct{ sourceType, sourceID string }
	createDraft := func(sourceType, sourceID string) (string, error) {
		calledWith.sourceType = sourceType
		calledWith.sourceID = sourceID
		return "journal_draft_router_001", nil
	}

	router := NewDispatchRouter(
		NewPostingDispatcher(createDraft),
		NewNoOpDispatcher("bank_reconciliation_service"),
	)

	proposal := ActionProposal{
		Action:                       "finance.requestDraftJournal",
		Label:                        "Review draft posting",
		Reason:                       "Missing journal links detected.",
		Priority:                     PriorityHigh,
		SourceType:                   "posting_coverage",
		RequiredDeterministicService: "finance_posting_service",
		MutatesState:                 false,
	}
	approval := DispatchApproval{
		Actor:         "finance@example.com",
		ActorType:     "operator",
		Reason:        "Reviewed and approved.",
		CorrelationID: "corr-router-001",
		ApprovedAt:    time.Now(),
		SourceType:    "posting_coverage",
		SourceID:      "entry-99",
	}

	result, err := router.Dispatch(context.Background(), proposal, approval)
	if err != nil {
		t.Fatalf("router.Dispatch returned unexpected error: %v", err)
	}
	if !result.Executed {
		t.Fatal("result.Executed = false, want true")
	}
	if result.OutputRef != "journal_draft_router_001" {
		t.Fatalf("OutputRef = %q, want %q", result.OutputRef, "journal_draft_router_001")
	}
	if calledWith.sourceType != "posting_coverage" || calledWith.sourceID != "entry-99" {
		t.Fatalf("createDraft called with (%q, %q), want (%q, %q)",
			calledWith.sourceType, calledWith.sourceID, "posting_coverage", "entry-99")
	}
}
