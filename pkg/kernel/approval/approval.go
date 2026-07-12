// Package approval provides a typed, canonical approval state machine for use
// across the ph_holdings_app codebase. It generalises the raw-string status
// constants in cashflow_evidence_review.go and the ReviewRecord pattern in
// pkg/documents/intake/review_queue.go into a single, stdlib-only primitive.
//
// Design principles:
//   - All state is expressed through the typed Decision enum; no raw strings.
//   - ValidTransition is the single source of truth for legal state changes.
//   - NewRecord validates every field before construction; no zero-value traps.
//   - Zero dependencies beyond stdlib.
package approval

import (
	"fmt"
	"strings"
	"time"
)

// Decision represents the outcome of an approval review.
type Decision string

const (
	DecisionPending    Decision = "pending_review"
	DecisionApproved   Decision = "approved"
	DecisionRejected   Decision = "rejected"
	DecisionNeedsInput Decision = "needs_input"
	DecisionSuperseded Decision = "superseded"
)

// knownDecisions is the authoritative set used by IsKnown and NewRecord validation.
var knownDecisions = map[Decision]bool{
	DecisionPending:    true,
	DecisionApproved:   true,
	DecisionRejected:   true,
	DecisionNeedsInput: true,
	DecisionSuperseded: true,
}

// allowedTransitions is the canonical transition table for all approval-like
// state machines in this codebase. It mirrors — and types — the raw-string map
// in validCashflowProposalTransition (cashflow_evidence_review.go).
//
// Transition table:
//
//	pending_review → approved, rejected, needs_input, superseded
//	needs_input    → pending_review, approved, rejected, superseded
//	approved       → superseded (no reversal)
//	rejected       → pending_review, superseded
//	superseded     → (terminal, no exits)
var allowedTransitions = map[Decision]map[Decision]bool{
	DecisionPending: {
		DecisionApproved:   true,
		DecisionRejected:   true,
		DecisionNeedsInput: true,
		DecisionSuperseded: true,
	},
	DecisionNeedsInput: {
		DecisionPending:    true,
		DecisionApproved:   true,
		DecisionRejected:   true,
		DecisionSuperseded: true,
	},
	DecisionApproved: {
		DecisionSuperseded: true,
	},
	DecisionRejected: {
		DecisionPending:    true,
		DecisionSuperseded: true,
	},
	// DecisionSuperseded is terminal: it is not a key in this map.
}

// ActorType distinguishes human operators from system/agent actors.
type ActorType string

const (
	ActorOperator ActorType = "operator"
	ActorSystem   ActorType = "system"
	ActorAgent    ActorType = "agent"
)

// Record captures a single approval decision with full audit context.
type Record struct {
	ID            string
	SubjectKey    string // stable identity of the thing being approved (e.g., proposal review key)
	SubjectType   string // "cashflow_proposal", "intake_candidate", "posting_draft", etc.
	Decision      Decision
	Actor         string // who made the decision
	ActorType     ActorType
	Reason        string
	CorrelationID string
	Timestamp     time.Time
}

// RecordInput is the input for creating a new Record.
type RecordInput struct {
	SubjectKey    string
	SubjectType   string
	Decision      Decision
	Actor         string
	ActorType     ActorType
	Reason        string
	CorrelationID string
}

// NewRecord creates a validated approval Record.
//
// Validation rules (in order):
//  1. SubjectKey must not be empty (trimmed).
//  2. Decision must be one of the five known Decision constants.
//  3. If ActorType == ActorAgent and Decision == DecisionApproved, agents cannot approve.
//  4. CorrelationID must not be empty (trimmed).
//
// The Record ID is generated as fmt.Sprintf("apr_%d", now.UnixNano()), which is
// collision-resistant within a single process and requires no external dependency.
func NewRecord(input RecordInput, now time.Time) (Record, error) {
	if strings.TrimSpace(input.SubjectKey) == "" {
		return Record{}, fmt.Errorf("approval: SubjectKey must not be empty")
	}
	if !knownDecisions[input.Decision] {
		return Record{}, fmt.Errorf("approval: unknown decision %q", input.Decision)
	}
	if input.ActorType == ActorAgent && input.Decision == DecisionApproved {
		return Record{}, fmt.Errorf("approval: agent actors cannot approve")
	}
	if strings.TrimSpace(input.CorrelationID) == "" {
		return Record{}, fmt.Errorf("approval: CorrelationID must not be empty")
	}
	return Record{
		ID:            fmt.Sprintf("apr_%d", now.UnixNano()),
		SubjectKey:    input.SubjectKey,
		SubjectType:   input.SubjectType,
		Decision:      input.Decision,
		Actor:         input.Actor,
		ActorType:     input.ActorType,
		Reason:        input.Reason,
		CorrelationID: input.CorrelationID,
		Timestamp:     now,
	}, nil
}

// ValidTransition reports whether moving from one Decision to another is allowed.
// Self-transitions and unknown states return false.
func ValidTransition(from, to Decision) bool {
	exits, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return exits[to]
}

// IsTerminal reports whether a decision is a terminal state (no further transitions).
// Only DecisionSuperseded is terminal.
func IsTerminal(d Decision) bool {
	return d == DecisionSuperseded
}

// IsKnown reports whether a decision is one of the recognised constants.
func IsKnown(d Decision) bool {
	return knownDecisions[d]
}
