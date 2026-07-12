// Package policy provides the kernel Policy primitive: a versioned rule that
// constrains actions over an effective period, optionally requiring evidence,
// and whose violations may only be overridden by an authorised actor.
//
// Policy is sector-agnostic: it carries a Scope string (e.g. a jurisdiction or
// domain tag) but embeds no tax model or industry vocabulary — those live in
// overlays (the Kernel Constitution forbids GSTReturn/VATInvoice in the kernel).
//
// It composes pkg/kernel/actor for the AI-authority boundary: overriding a
// policy violation is an authority transition, so an agent can never do it.
// Zero dependencies beyond stdlib + pkg/kernel/actor.
package policy

import (
	"fmt"
	"strings"
	"time"

	"ph_holdings_app/pkg/kernel/actor"
)

// Status is the outcome of evaluating a situation against a policy.
type Status string

const (
	StatusCompliant  Status = "compliant"
	StatusViolation  Status = "violation"
	StatusOverridden Status = "overridden"
)

// Policy is a validated, versioned constraint.
type Policy struct {
	ID            string    `json:"id"`
	Version       string    `json:"version"`
	Scope         string    `json:"scope"`          // jurisdiction/domain tag, or "global"
	EffectiveFrom time.Time `json:"effective_from"` // zero = always-effective lower bound
	EffectiveTo   time.Time `json:"effective_to"`   // zero = open-ended upper bound
	// RequiresEvidence makes a situation non-compliant unless backing evidence
	// is present, regardless of whether the rule itself is otherwise satisfied.
	RequiresEvidence bool `json:"requires_evidence"`
}

// Input is the input for New.
type Input struct {
	ID               string
	Version          string
	Scope            string
	EffectiveFrom    time.Time
	EffectiveTo      time.Time
	RequiresEvidence bool
}

// New constructs a validated Policy.
//
// Validation rules:
//  1. ID must not be empty (trimmed).
//  2. Version must not be empty (trimmed).
//  3. If both EffectiveFrom and EffectiveTo are set, To must not precede From.
func New(in Input) (Policy, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Policy{}, fmt.Errorf("policy: ID must not be empty")
	}
	if strings.TrimSpace(in.Version) == "" {
		return Policy{}, fmt.Errorf("policy: Version must not be empty")
	}
	if !in.EffectiveFrom.IsZero() && !in.EffectiveTo.IsZero() && in.EffectiveTo.Before(in.EffectiveFrom) {
		return Policy{}, fmt.Errorf("policy: EffectiveTo %s precedes EffectiveFrom %s", in.EffectiveTo, in.EffectiveFrom)
	}
	return Policy(in), nil
}

// IsEffective reports whether the policy is in force at the given instant.
// A zero EffectiveFrom means "no lower bound"; a zero EffectiveTo means
// "open-ended". Bounds are inclusive.
func (p Policy) IsEffective(at time.Time) bool {
	if !p.EffectiveFrom.IsZero() && at.Before(p.EffectiveFrom) {
		return false
	}
	if !p.EffectiveTo.IsZero() && at.After(p.EffectiveTo) {
		return false
	}
	return true
}

// Evaluate classifies a situation against the policy. A situation is compliant
// only when the rule is satisfied AND (if the policy requires evidence) evidence
// is present. Anything else is a violation.
func (p Policy) Evaluate(satisfied, hasEvidence bool) Status {
	if p.RequiresEvidence && !hasEvidence {
		return StatusViolation
	}
	if !satisfied {
		return StatusViolation
	}
	return StatusCompliant
}

// Override is a recorded, authorised override of a policy violation.
type Override struct {
	PolicyID string    `json:"policy_id"`
	ActorID  string    `json:"actor_id"`
	Reason   string    `json:"reason"`
	At       time.Time `json:"at"`
}

// Override records an authorised override of a violation of this policy.
//
// It fails if:
//  1. the acting actor cannot approve (the AI-authority boundary — an agent can
//     never override a policy violation), or
//  2. the reason is empty (an override must be justified).
func (p Policy) Override(by actor.Actor, reason string, at time.Time) (Override, error) {
	if !by.CanApprove() {
		return Override{}, fmt.Errorf("policy: actor %q cannot override policy %q (AI-authority boundary)", by.ID, p.ID)
	}
	if strings.TrimSpace(reason) == "" {
		return Override{}, fmt.Errorf("policy: override of %q requires a reason", p.ID)
	}
	return Override{
		PolicyID: p.ID,
		ActorID:  by.ID,
		Reason:   reason,
		At:       at,
	}, nil
}

// IsKnownStatus reports whether s is a recognised policy status.
func IsKnownStatus(s Status) bool {
	switch s {
	case StatusCompliant, StatusViolation, StatusOverridden:
		return true
	default:
		return false
	}
}
