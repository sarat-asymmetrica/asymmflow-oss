package evidence

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNoOpDispatcherMatchesService(t *testing.T) {
	d := NewNoOpDispatcher("finance_posting_service")
	proposal := ActionProposal{
		Action:                       "finance.requestDraftJournal",
		Label:                        "Review draft posting",
		RequiredDeterministicService: "finance_posting_service",
	}

	if !d.CanDispatch(proposal) {
		t.Fatalf("CanDispatch = false, want true for matching service %q", proposal.RequiredDeterministicService)
	}
}

func TestNoOpDispatcherRejectsMismatchedService(t *testing.T) {
	d := NewNoOpDispatcher("finance_posting_service")
	proposal := ActionProposal{
		Action:                       "banking.resolveMatch",
		Label:                        "Resolve bank matches",
		RequiredDeterministicService: "bank_reconciliation_service",
	}

	if d.CanDispatch(proposal) {
		t.Fatalf("CanDispatch = true, want false for mismatched service %q", proposal.RequiredDeterministicService)
	}
}

func TestNoOpDispatcherDoesNotExecute(t *testing.T) {
	d := NewNoOpDispatcher("finance_posting_service")
	proposal := ActionProposal{
		Action:                       "finance.requestDraftJournal",
		RequiredDeterministicService: "finance_posting_service",
	}
	approval := DispatchApproval{
		Actor:      "operator@example.com",
		ActorType:  "operator",
		ApprovedAt: time.Now(),
	}

	result, err := d.Dispatch(context.Background(), proposal, approval)
	if err != nil {
		t.Fatalf("Dispatch returned unexpected error: %v", err)
	}
	if result.Executed {
		t.Fatal("NoOpDispatcher must not set Executed=true")
	}
	if result.ServiceUsed != "finance_posting_service" {
		t.Fatalf("ServiceUsed = %q, want %q", result.ServiceUsed, "finance_posting_service")
	}
}

func TestDispatchRouterFindsCorrectDispatcher(t *testing.T) {
	financeDisp := NewNoOpDispatcher("finance_posting_service")
	bankDisp := NewNoOpDispatcher("bank_reconciliation_service")
	router := NewDispatchRouter(financeDisp, bankDisp)

	tests := []struct {
		service string
		want    ProposalDispatcher
	}{
		{"finance_posting_service", financeDisp},
		{"bank_reconciliation_service", bankDisp},
	}

	for _, tt := range tests {
		proposal := ActionProposal{RequiredDeterministicService: tt.service}
		got := router.Route(proposal)
		if got != tt.want {
			t.Errorf("Route(%q): got dispatcher for %q, want %q", tt.service, got.ServiceName(), tt.want.ServiceName())
		}
	}
}

func TestDispatchRouterRejectsAgentActor(t *testing.T) {
	router := NewDispatchRouter(NewNoOpDispatcher("finance_posting_service"))
	proposal := ActionProposal{
		Action:                       "finance.requestDraftJournal",
		RequiredDeterministicService: "finance_posting_service",
	}
	approval := DispatchApproval{
		Actor:     "ai-agent",
		ActorType: "agent",
	}

	_, err := router.Dispatch(context.Background(), proposal, approval)
	if err == nil {
		t.Fatal("expected error when ActorType is agent, got nil")
	}
	if !strings.Contains(err.Error(), "agent") {
		t.Fatalf("error message should mention agent, got: %q", err.Error())
	}
}

func TestDispatchRouterRejectsUnknownService(t *testing.T) {
	router := NewDispatchRouter(NewNoOpDispatcher("finance_posting_service"))
	proposal := ActionProposal{
		Action:                       "cashflowEvidence.draftFollowUp",
		RequiredDeterministicService: "task_or_collections_service",
	}
	approval := DispatchApproval{
		Actor:     "operator@example.com",
		ActorType: "operator",
	}

	_, err := router.Dispatch(context.Background(), proposal, approval)
	if err == nil {
		t.Fatal("expected error for unregistered service, got nil")
	}
	if !strings.Contains(err.Error(), "task_or_collections_service") {
		t.Fatalf("error should name the unregistered service, got: %q", err.Error())
	}
}

func TestDispatchRouterDispatchesSuccessfully(t *testing.T) {
	router := NewDispatchRouter(
		NewNoOpDispatcher("finance_posting_service"),
		NewNoOpDispatcher("bank_reconciliation_service"),
	)
	proposal := ActionProposal{
		Action:                       "finance.requestDraftJournal",
		Label:                        "Review draft posting",
		Reason:                       "Missing journal links detected.",
		Priority:                     PriorityHigh,
		RequiredDeterministicService: "finance_posting_service",
		MutatesState:                 false,
	}
	approval := DispatchApproval{
		Actor:         "finance@example.com",
		ActorType:     "operator",
		Reason:        "Reviewed and approved by finance lead.",
		CorrelationID: "corr-001",
		ApprovedAt:    time.Now(),
	}

	result, err := router.Dispatch(context.Background(), proposal, approval)
	if err != nil {
		t.Fatalf("Dispatch returned unexpected error: %v", err)
	}
	if result.ServiceUsed != "finance_posting_service" {
		t.Fatalf("ServiceUsed = %q, want %q", result.ServiceUsed, "finance_posting_service")
	}
	if result.Message == "" {
		t.Fatal("expected non-empty result message")
	}
}
