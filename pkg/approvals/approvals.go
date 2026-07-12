// Package approvals is the approval-routing engine: it expresses "does this
// action need a human decision, and who may make it" on the kernel's
// vocabulary (pkg/kernel/approval decisions + pkg/kernel/actor authority).
//
// Two shapes cover every approval flow in the app (Wave 3 B.1):
//
//   - Assessment — the ROUTING half. Domain rules (margin floors, customer
//     grades, entity guards…) stay in the vertical and emit Findings; the
//     Assessment folds them into a kernel decision: DecisionApproved
//     (auto-approved, proceed) or DecisionPending (a human must decide).
//
//   - Transition — the DECIDING half. Every movement of an approval subject
//     between kernel decisions passes through here: the kernel transition
//     table says whether the move is legal, and the actor's authority says
//     whether THIS actor may make it. Agents can never approve or reject —
//     that is the AI-authority boundary, enforced at this seam and pinned by
//     tests in both consuming flows.
//
// The package is persistence-free: verticals own their storage rows and map
// their status strings through DecisionFromStatus/StatusFromDecision.
package approvals

import (
	"fmt"
	"strings"
	"time"

	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/approval"
)

// Finding is one routing rule's outcome: a warning for the operator and,
// optionally, a demand that the subject enter approval.
type Finding struct {
	// Code identifies the rule that fired (e.g. "margin_below_threshold").
	Code string
	// Message is the operator-facing warning text.
	Message string
	// RequiresApproval routes the subject to DecisionPending when true.
	RequiresApproval bool
	// Recommendation, when non-empty, overrides the assessment's recommended
	// action (the LAST finding with a recommendation wins, matching the
	// sequential rule evaluation the trading flow always had).
	Recommendation string
}

// Assessment accumulates rule findings for one subject and folds them into a
// kernel routing decision.
type Assessment struct {
	SubjectKey  string
	SubjectType string
	findings    []Finding
}

// NewAssessment starts an assessment for a subject.
func NewAssessment(subjectKey, subjectType string) *Assessment {
	return &Assessment{SubjectKey: subjectKey, SubjectType: subjectType}
}

// Add records a finding.
func (a *Assessment) Add(f Finding) { a.findings = append(a.findings, f) }

// Findings returns the recorded findings in evaluation order.
func (a *Assessment) Findings() []Finding { return a.findings }

// Warnings returns the operator-facing messages in evaluation order.
func (a *Assessment) Warnings() []string {
	out := make([]string, 0, len(a.findings))
	for _, f := range a.findings {
		if f.Message != "" {
			out = append(out, f.Message)
		}
	}
	return out
}

// NeedsApproval reports whether any finding demanded approval.
func (a *Assessment) NeedsApproval() bool {
	for _, f := range a.findings {
		if f.RequiresApproval {
			return true
		}
	}
	return false
}

// Decision folds the findings into the kernel routing decision:
// DecisionPending when any finding requires approval, DecisionApproved
// (auto-approved) otherwise.
func (a *Assessment) Decision() approval.Decision {
	if a.NeedsApproval() {
		return approval.DecisionPending
	}
	return approval.DecisionApproved
}

// Recommendation resolves the recommended action: the last finding carrying
// one wins; with none, needsApprovalDefault applies when approval is needed,
// else autoDefault.
func (a *Assessment) Recommendation(autoDefault, needsApprovalDefault string) string {
	rec := ""
	for _, f := range a.findings {
		if f.Recommendation != "" {
			rec = f.Recommendation
		}
	}
	if rec != "" {
		return rec
	}
	if a.NeedsApproval() {
		return needsApprovalDefault
	}
	return autoDefault
}

// Transition moves an approval subject from one kernel decision to another,
// enforcing (in order):
//
//  1. the kernel transition table (approval.ValidTransition) — illegal moves
//     (approving an already-rejected request, re-deciding a decided one) fail;
//  2. the actor-authority boundary — DecisionApproved and DecisionRejected
//     may only be issued by an actor whose CanApprove() is true. Agent actors
//     can never satisfy that (kernel guarantee), whatever authority they claim.
//
// On success it returns the validated kernel Record of the new decision,
// which the caller persists in its own storage alongside its domain row.
func Transition(subjectKey, subjectType string, from, to approval.Decision, by actor.Actor, reason string, now time.Time) (approval.Record, error) {
	if !approval.ValidTransition(from, to) {
		return approval.Record{}, fmt.Errorf("approvals: illegal transition %s → %s for %s %q", from, to, subjectType, subjectKey)
	}
	if (to == approval.DecisionApproved || to == approval.DecisionRejected) && !by.CanApprove() {
		return approval.Record{}, fmt.Errorf("approvals: actor %q (type %s, authority %s) cannot decide %s %q (AI-authority boundary)",
			by.ID, by.Type, by.Authority, subjectType, subjectKey)
	}
	return approval.NewRecord(approval.RecordInput{
		SubjectKey:    subjectKey,
		SubjectType:   subjectType,
		Decision:      to,
		Actor:         by.ID,
		ActorType:     kernelActorType(by.Type),
		Reason:        reason,
		CorrelationID: fmt.Sprintf("%s:%s", subjectType, subjectKey),
	}, now)
}

func kernelActorType(t actor.Type) approval.ActorType {
	switch t {
	case actor.TypeAgent:
		return approval.ActorAgent
	case actor.TypeOperator:
		return approval.ActorOperator
	default:
		return approval.ActorSystem
	}
}

// DecisionFromStatus maps a vertical's stored status string onto the kernel
// decision vocabulary. Recognised (case/space-insensitive): pending,
// pending_review, needs_approval, draft → DecisionPending; approved,
// auto_approved → DecisionApproved; rejected → DecisionRejected; needs_input →
// DecisionNeedsInput; superseded → DecisionSuperseded. Unknown strings return
// an error rather than a guess — status vocabulary drift must surface, not
// route.
func DecisionFromStatus(status string) (approval.Decision, error) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "pending_review", "needs_approval", "draft":
		return approval.DecisionPending, nil
	case "approved", "auto_approved":
		return approval.DecisionApproved, nil
	case "rejected":
		return approval.DecisionRejected, nil
	case "needs_input":
		return approval.DecisionNeedsInput, nil
	case "superseded":
		return approval.DecisionSuperseded, nil
	default:
		return "", fmt.Errorf("approvals: unrecognised approval status %q", status)
	}
}
