package evidence

import (
	"context"
	"fmt"
)

// CreateDraftJournalFunc is the signature for creating a draft journal entry.
// In production, this wraps App.CreateDraftJournalFromPosting.
// In tests, it can be a simple function that records calls.
type CreateDraftJournalFunc func(sourceType, sourceID string) (journalID string, err error)

// PostingDispatcher routes approved posting proposals to the journal creation
// service. It uses function injection so it has no dependency on *App.
type PostingDispatcher struct {
	createDraft CreateDraftJournalFunc
}

// NewPostingDispatcher creates a dispatcher with the given journal creation function.
func NewPostingDispatcher(createDraft CreateDraftJournalFunc) *PostingDispatcher {
	return &PostingDispatcher{createDraft: createDraft}
}

// ServiceName returns the deterministic service this dispatcher handles.
func (d *PostingDispatcher) ServiceName() string { return "finance_posting_service" }

// CanDispatch reports whether this dispatcher can handle the given proposal.
func (d *PostingDispatcher) CanDispatch(proposal ActionProposal) bool {
	return proposal.RequiredDeterministicService == "finance_posting_service"
}

// Dispatch executes an approved posting proposal by calling the injected
// createDraft function. The approval must carry SourceType and SourceID
// identifying the specific document to journal.
//
// Rules:
//   - ActorType "agent" is always rejected.
//   - SourceType and SourceID must be non-empty.
//   - Any error from createDraft is wrapped and returned.
func (d *PostingDispatcher) Dispatch(ctx context.Context, proposal ActionProposal, approval DispatchApproval) (DispatchResult, error) {
	if approval.ActorType == "agent" {
		return DispatchResult{}, fmt.Errorf("agent actors cannot dispatch posting proposals")
	}
	if d.createDraft == nil {
		return DispatchResult{}, fmt.Errorf("posting dispatch: createDraft function not configured")
	}
	if approval.SourceType == "" || approval.SourceID == "" {
		return DispatchResult{}, fmt.Errorf("posting dispatch requires SourceType and SourceID in approval")
	}
	journalID, err := d.createDraft(approval.SourceType, approval.SourceID)
	if err != nil {
		return DispatchResult{}, fmt.Errorf("posting dispatch: %w", err)
	}
	return DispatchResult{
		Executed:    true,
		ServiceUsed: "finance_posting_service",
		OutputRef:   journalID,
		Message:     fmt.Sprintf("created draft journal %s for %s %s", journalID, approval.SourceType, approval.SourceID),
	}, nil
}
