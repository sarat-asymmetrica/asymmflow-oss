// Package actor provides the kernel Actor primitive: a human, service, agent,
// device, or organization that can observe, propose, approve, execute, or audit
// an action.
//
// Actor is the home of the AI-authority boundary (CLAUDE.md invariant #4 and the
// Kernel Constitution invariant "agents cannot directly record authority
// transitions"): an agent actor can never approve/execute/reverse, regardless of
// any authority level it is otherwise granted. CanApprove enforces this at the
// kernel layer so every overlay inherits it.
//
// Design principles (mirroring pkg/kernel/approval):
//   - Typed enums; no raw strings at call sites.
//   - New validates every field; no zero-value traps.
//   - Zero dependencies beyond stdlib.
//
// Note: pkg/kernel/approval carries a narrower inline ActorType (operator/system/
// agent) that predates this package. Consolidating that into actor.Type is a
// documented migration follow-up, not a giant rename (Kernel Constitution
// "Migration Principle").
package actor

import (
	"fmt"
	"strings"
)

// Type distinguishes the kinds of actor the kernel recognises.
type Type string

const (
	TypeOperator     Type = "operator"     // a human operator
	TypeSystem       Type = "system"       // a deterministic, non-AI service
	TypeAgent        Type = "agent"        // an AI agent (inspect/draft/recommend only)
	TypeDevice       Type = "device"       // a device or sensor
	TypeOrganization Type = "organization" // an organization acting as a principal
)

// knownTypes is the authoritative set used by IsKnownType and New validation.
var knownTypes = map[Type]bool{
	TypeOperator:     true,
	TypeSystem:       true,
	TypeAgent:        true,
	TypeDevice:       true,
	TypeOrganization: true,
}

// Authority is the level of action an actor is permitted to take. It is ordered:
// a higher level implies the capabilities of the lower ones — EXCEPT that the
// AI-authority boundary in CanApprove overrides Authority for agent actors.
type Authority int

const (
	// AuthorityObserve can read/inspect only.
	AuthorityObserve Authority = iota
	// AuthorityPropose can draft, recommend, and propose (the ceiling for agents).
	AuthorityPropose
	// AuthorityApprove can approve, post, persist, and execute.
	AuthorityApprove
	// AuthorityAdmin can do everything, including administrative overrides.
	AuthorityAdmin
)

// authorityNames backs String and the (de)serialisation round-trip.
var authorityNames = map[Authority]string{
	AuthorityObserve: "observe",
	AuthorityPropose: "propose",
	AuthorityApprove: "approve",
	AuthorityAdmin:   "admin",
}

// String renders an Authority as its canonical token.
func (a Authority) String() string {
	if name, ok := authorityNames[a]; ok {
		return name
	}
	return fmt.Sprintf("authority(%d)", int(a))
}

// Actor is a validated kernel actor.
type Actor struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Type        Type      `json:"type"`
	Authority   Authority `json:"authority"`
	Permissions []string  `json:"permissions"` // permission claims (e.g. "invoices:create")
}

// Input is the input for New.
type Input struct {
	ID          string
	DisplayName string
	Type        Type
	Authority   Authority
	Permissions []string
}

// New constructs a validated Actor.
//
// Validation rules (in order):
//  1. ID must not be empty (trimmed).
//  2. Type must be one of the known Type constants.
//  3. Authority must be one of the four defined levels.
//  4. An agent actor may not be granted AuthorityApprove or AuthorityAdmin — the
//     AI-authority boundary forbids it at construction time, so an agent can
//     never be minted with approve power in the first place.
func New(in Input) (Actor, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Actor{}, fmt.Errorf("actor: ID must not be empty")
	}
	if !knownTypes[in.Type] {
		return Actor{}, fmt.Errorf("actor: unknown type %q", in.Type)
	}
	if _, ok := authorityNames[in.Authority]; !ok {
		return Actor{}, fmt.Errorf("actor: unknown authority level %d", int(in.Authority))
	}
	if in.Type == TypeAgent && in.Authority >= AuthorityApprove {
		return Actor{}, fmt.Errorf("actor: agent actors may not hold %s authority (AI-authority boundary)", in.Authority)
	}
	return Actor{
		ID:          in.ID,
		DisplayName: in.DisplayName,
		Type:        in.Type,
		Authority:   in.Authority,
		Permissions: append([]string(nil), in.Permissions...),
	}, nil
}

// IsAgent reports whether the actor is an AI agent.
func (a Actor) IsAgent() bool { return a.Type == TypeAgent }

// CanApprove reports whether the actor may approve, post, persist, execute, or
// reverse. This is the kernel AI-authority boundary: an agent can NEVER approve,
// no matter its Authority. A non-agent needs at least AuthorityApprove.
func (a Actor) CanApprove() bool {
	if a.IsAgent() {
		return false
	}
	return a.Authority >= AuthorityApprove
}

// CanPropose reports whether the actor may draft, recommend, or propose. Every
// actor at or above AuthorityPropose can propose (agents included — proposing is
// exactly what agents are for).
func (a Actor) CanPropose() bool {
	return a.Authority >= AuthorityPropose
}

// HasPermission reports whether the actor holds the named permission claim.
func (a Actor) HasPermission(permission string) bool {
	for _, p := range a.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// IsKnownType reports whether t is a recognised actor type.
func IsKnownType(t Type) bool { return knownTypes[t] }
