package evidence

import (
	"context"
	"fmt"
	"time"
)

// ProposalDispatcher routes approved action proposals to their
// required deterministic services. Each dispatcher handles one
// service type (e.g., finance_posting_service, bank_reconciliation_service).
type ProposalDispatcher interface {
	// ServiceName returns the deterministic service this dispatcher handles.
	ServiceName() string

	// CanDispatch reports whether this dispatcher can handle the given proposal.
	CanDispatch(proposal ActionProposal) bool

	// Dispatch executes the approved proposal's action through the deterministic
	// service. The approval must have been granted by a non-agent actor.
	// Returns a DispatchResult describing what happened.
	Dispatch(ctx context.Context, proposal ActionProposal, approval DispatchApproval) (DispatchResult, error)
}

// DispatchApproval carries the operator approval that authorizes dispatch.
// This is intentionally a separate struct from any kernel type so the
// evidence package remains self-contained.
type DispatchApproval struct {
	Actor         string
	ActorType     string // "operator" or "system" — never "agent"
	Reason        string
	CorrelationID string
	ApprovedAt    time.Time
	SourceType    string // optional: source document type for dispatch context (e.g. "posting_coverage")
	SourceID      string // optional: source document ID for dispatch context (e.g. a posting entry ID)
}

// DispatchResult describes the outcome of dispatching a proposal.
type DispatchResult struct {
	Executed    bool   // true if the deterministic service actually ran
	ServiceUsed string // which service handled it
	OutputRef   string // ID of created/modified entity (e.g., draft journal ID)
	Message     string // human-readable summary
}

// DispatchRouter holds a set of dispatchers and routes proposals to the right one.
type DispatchRouter struct {
	dispatchers []ProposalDispatcher
}

// NewDispatchRouter creates a router with the given dispatchers.
func NewDispatchRouter(dispatchers ...ProposalDispatcher) *DispatchRouter {
	return &DispatchRouter{dispatchers: dispatchers}
}

// Route finds the appropriate dispatcher for a proposal.
// Returns nil if no dispatcher can handle it.
func (r *DispatchRouter) Route(proposal ActionProposal) ProposalDispatcher {
	for _, d := range r.dispatchers {
		if d.CanDispatch(proposal) {
			return d
		}
	}
	return nil
}

// Dispatch routes and executes a proposal through the appropriate dispatcher.
// Returns error if:
// - the approval's ActorType is "agent"
// - no dispatcher can handle the proposal
// - the underlying dispatcher returns an error
func (r *DispatchRouter) Dispatch(ctx context.Context, proposal ActionProposal, approval DispatchApproval) (DispatchResult, error) {
	if approval.ActorType == "agent" {
		return DispatchResult{}, fmt.Errorf("agent actors cannot dispatch proposals")
	}
	d := r.Route(proposal)
	if d == nil {
		return DispatchResult{}, fmt.Errorf("no dispatcher registered for service %q", proposal.RequiredDeterministicService)
	}
	return d.Dispatch(ctx, proposal, approval)
}

// NoOpDispatcher is a test double that accepts all proposals for its service
// but never executes them.
type NoOpDispatcher struct {
	service string
}

// NewNoOpDispatcher creates a NoOpDispatcher for the given service name.
func NewNoOpDispatcher(service string) *NoOpDispatcher {
	return &NoOpDispatcher{service: service}
}

// ServiceName returns the deterministic service this dispatcher handles.
func (d *NoOpDispatcher) ServiceName() string {
	return d.service
}

// CanDispatch returns true when the proposal's RequiredDeterministicService
// matches this dispatcher's service name.
func (d *NoOpDispatcher) CanDispatch(proposal ActionProposal) bool {
	return proposal.RequiredDeterministicService == d.service
}

// Dispatch records the call but performs no real work, returning Executed=false.
func (d *NoOpDispatcher) Dispatch(_ context.Context, _ ActionProposal, _ DispatchApproval) (DispatchResult, error) {
	return DispatchResult{
		Executed:    false,
		ServiceUsed: d.service,
		Message:     "no-op dispatch for testing",
	}, nil
}
