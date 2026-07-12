// Package workflow provides the kernel Workflow primitive: a typed progression
// of states for an operational object (proposed → approved → executed, with
// rejected and reversed terminal/closing states).
//
// It composes pkg/kernel/actor to enforce the AI-authority boundary at the
// kernel layer: an agent may CREATE a proposal (proposing is what agents are
// for), but only an actor that CanApprove may ADVANCE a workflow into an
// authority-bearing state. This encodes the Kernel Constitution invariants
// "workflows distinguish proposed, approved, executed, rejected, and reversed
// states" and "agents cannot directly record authority transitions".
//
// Advance is pure: it returns a new Workflow value rather than mutating the
// receiver. Zero dependencies beyond stdlib + pkg/kernel/actor.
package workflow

import (
	"fmt"
	"strings"

	"ph_holdings_app/pkg/kernel/actor"
)

// State is a workflow lifecycle state.
type State string

const (
	StateProposed State = "proposed" // created/recommended, awaiting an authority decision
	StateApproved State = "approved" // an authority allowed it
	StateExecuted State = "executed" // the approved action was carried out (posted/persisted)
	StateRejected State = "rejected" // an authority declined it (terminal)
	StateReversed State = "reversed" // a prior approval/execution was undone (terminal)
)

var knownStates = map[State]bool{
	StateProposed: true,
	StateApproved: true,
	StateExecuted: true,
	StateRejected: true,
	StateReversed: true,
}

// allowedTransitions is the canonical workflow transition table.
//
//	proposed → approved, rejected
//	approved → executed, reversed
//	executed → reversed
//	rejected → (terminal)
//	reversed → (terminal)
var allowedTransitions = map[State]map[State]bool{
	StateProposed: {
		StateApproved: true,
		StateRejected: true,
	},
	StateApproved: {
		StateExecuted: true,
		StateReversed: true,
	},
	StateExecuted: {
		StateReversed: true,
	},
	// StateRejected and StateReversed are terminal: not keys in this map.
}

// Workflow is a validated workflow record. It is a value type; Advance returns a
// new Workflow rather than mutating in place.
type Workflow struct {
	ID          string `json:"id"`
	SubjectKey  string `json:"subject_key"`  // stable identity of the object the workflow governs
	SubjectType string `json:"subject_type"` // e.g. "costing_approval", "posting_draft"
	State       State  `json:"state"`
	CreatedByID string `json:"created_by_id"`
}

// Input is the input for New.
type Input struct {
	ID          string
	SubjectKey  string
	SubjectType string
	CreatedBy   actor.Actor
}

// New constructs a Workflow in StateProposed.
//
// Validation rules (in order):
//  1. SubjectKey must not be empty (trimmed).
//  2. The creator must be able to propose (CanPropose). An observe-only actor
//     cannot open a workflow.
func New(in Input) (Workflow, error) {
	if strings.TrimSpace(in.SubjectKey) == "" {
		return Workflow{}, fmt.Errorf("workflow: SubjectKey must not be empty")
	}
	if !in.CreatedBy.CanPropose() {
		return Workflow{}, fmt.Errorf("workflow: actor %q lacks authority to propose", in.CreatedBy.ID)
	}
	return Workflow{
		ID:          in.ID,
		SubjectKey:  in.SubjectKey,
		SubjectType: in.SubjectType,
		State:       StateProposed,
		CreatedByID: in.CreatedBy.ID,
	}, nil
}

// Advance returns a copy of the workflow moved to the target state.
//
// It fails if:
//  1. the transition is not allowed by the transition table, or
//  2. the acting actor cannot approve (the AI-authority boundary — an agent can
//     never record an authority transition, no matter its granted authority).
func (w Workflow) Advance(by actor.Actor, to State) (Workflow, error) {
	if !ValidTransition(w.State, to) {
		return w, fmt.Errorf("workflow: illegal transition %s → %s", w.State, to)
	}
	if !by.CanApprove() {
		return w, fmt.Errorf("workflow: actor %q cannot record the authority transition %s → %s (AI-authority boundary)", by.ID, w.State, to)
	}
	next := w
	next.State = to
	return next, nil
}

// ValidTransition reports whether moving from one state to another is allowed.
// Self-transitions and unknown states return false.
func ValidTransition(from, to State) bool {
	exits, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return exits[to]
}

// IsTerminal reports whether a state has no outgoing transitions.
func IsTerminal(s State) bool {
	_, hasExits := allowedTransitions[s]
	return knownStates[s] && !hasExits
}

// IsKnownState reports whether s is a recognised workflow state.
func IsKnownState(s State) bool { return knownStates[s] }
