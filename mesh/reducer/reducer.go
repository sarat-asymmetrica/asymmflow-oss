// Package reducer is the pure, deterministic apply-reducer for the Sovereign
// Mesh (Missions A + C of FABLE_CAMPAIGN_SOVEREIGN_MESH.md).
//
// Wave 0/1 proved the machinery with a single toy domain (inventory floors).
// Mission C makes the claim real: the reducer now imports the ACTUAL AsymmFlow
// kernel packages — pkg/kernel/{money,approval,actor,policy} — so the same law
// that guards a posting on one desktop guards it identically on every peer:
//   - money:    integer minor-unit arithmetic; currency mismatches are typed errors
//   - approval: the canonical decision state machine (ValidTransition is THE truth)
//   - actor:    the AI-authority boundary — an agent can NEVER approve, anywhere
//   - policy:   violations can only be overridden by an approver, with a reason
//
// It stays dependency-free beyond stdlib + the kernel packages and free of build
// tags so it can be:
//   - unit-tested on the host (normal GOOS) — see reducer_test.go / missionc_test.go
//   - compiled to wasip1 and driven from the JS/Autobase host — see ../cmd/reducer
//
// THE CLAIM (Mission A, now C): the function that guards a business invariant is
// byte-identical whether it runs on one node or a thousand. Autobase linearizes
// every writer's append-only log into ONE order every peer agrees on, then
// replays it here. So this reducer must be a PURE, DETERMINISTIC function:
// same ops (as a set) → byte-identical state on every peer, forever.
//
// The four determinism landmines (§4 invariant 1 of the campaign) and how this
// file avoids each:
//  1. map iteration order is randomized  -> we NEVER range a map for output;
//     every traversal for hashing goes through a sorted key slice.
//  2. time.Now()/rand are forbidden      -> no clock, no randomness anywhere; an
//     op's timestamp is DATA carried in Op.TS. The kernel packages cooperate:
//     they take `now time.Time` as a PARAMETER, so we hand them op time.
//  3. floats drift                        -> all quantities/amounts are int64
//     (whole units / money minor units via pkg/kernel/money); no float appears.
//  4. unstable linearization              -> Apply canonically re-sorts the ops
//     by (Seq, Actor, Kind, keys…, TS) so a permuted delivery order converges.
//
// STATE SCHEMA v2 (Mission C): the digest covers four domains. Goldens were
// regenerated at the schema bump (see MESH-D9 in docs/MESH_DECISIONS.md).
package reducer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
)

// Op is one append-only event from some writer's Hypercore log. Kind selects
// the domain; unused fields stay zero and are omitted on the wire. An empty
// Kind means "inventory.move" (Wave 0/1 compatibility).
//
// Nothing here is wall-clock derived: TS is the event's own recorded time
// (millis), used as an ordering tie-breaker and as the kernel's `now` argument.
type Op struct {
	Seq   int64  `json:"seq"`            // per-writer monotonic sequence (Hypercore index)
	Actor string `json:"actor"`          // writer/device id (Ed25519 key id in the real mesh)
	TS    int64  `json:"ts"`             // event-data timestamp millis (NEVER a live clock)
	Kind  string `json:"kind,omitempty"` // domain selector; "" == "inventory.move"

	// inventory.move
	SKU   string `json:"sku,omitempty"`
	Delta int64  `json:"delta,omitempty"` // signed movement in whole units

	// ar.limit / ar.charge / ar.payment (pkg/kernel/money)
	Customer    string `json:"customer,omitempty"`
	AmountMinor int64  `json:"amountMinor,omitempty"` // always positive; kind picks the sign
	LimitMinor  int64  `json:"limitMinor,omitempty"`
	Currency    string `json:"currency,omitempty"`

	// approval.decide (pkg/kernel/approval + actor) & policy.* (pkg/kernel/policy)
	Subject       string `json:"subject,omitempty"`     // what is being approved
	SubjectType   string `json:"subjectType,omitempty"` // e.g. "posting_draft"
	Decision      string `json:"decision,omitempty"`    // approval.Decision token
	Reason        string `json:"reason,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
	ActorType     string `json:"actorType,omitempty"` // actor.Type token of the acting actor
	Authority     int    `json:"authority,omitempty"` // actor.Authority level claimed
	PolicyID      string `json:"policyId,omitempty"`
}

// Rejection records an op an invariant refused, so the UX can surface a typed
// Unconfirmed/Rejected state (Mechanism 2) instead of a silent bad number.
type Rejection struct {
	Seq      int64  `json:"seq"`
	Actor    string `json:"actor"`
	Kind     string `json:"kind,omitempty"`
	SKU      string `json:"sku,omitempty"`
	Delta    int64  `json:"delta,omitempty"`
	Customer string `json:"customer,omitempty"`
	Subject  string `json:"subject,omitempty"`
	PolicyID string `json:"policyId,omitempty"`
	Reason   string `json:"reason"`
}

// ARAccount is the converged accounts-receivable position for one customer.
// Balance may go negative (overpayment/credit); it may never EXCEED the limit.
type ARAccount struct {
	BalanceMinor int64  `json:"balanceMinor"`
	LimitMinor   int64  `json:"limitMinor"`
	Currency     string `json:"currency"`
}

// ApprovalState is the converged decision state for one subject, per the
// kernel approval state machine. Subjects begin implicitly at pending_review.
type ApprovalState struct {
	Decision      string `json:"decision"`
	Actor         string `json:"actor"`
	ActorType     string `json:"actorType"`
	Reason        string `json:"reason,omitempty"`
	CorrelationID string `json:"correlationId"`
	DecidedAtMS   int64  `json:"decidedAtMs"`
}

// PolicyState is the converged compliance state for one policy id.
type PolicyState struct {
	Status       string `json:"status"` // policy.Status token
	OverriddenBy string `json:"overriddenBy,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// State is the converged, deterministic result of replaying an op set.
// Digest is a sha256 over the canonical (sorted, map-free) state — two peers
// agree iff their digests match.
type State struct {
	Stock     map[string]int64         `json:"stock"`
	AR        map[string]ARAccount     `json:"ar"`
	Approvals map[string]ApprovalState `json:"approvals"`
	Policies  map[string]PolicyState   `json:"policies"`
	Rejected  []Rejection              `json:"rejected"`
	Applied   int                      `json:"applied"`
	Digest    string                   `json:"digest"`
	OpsHashed int                      `json:"opsHashed"`
}

// canonicalLess is the total order Autobase-style linearization must agree on.
// Sorting by (Seq, Actor, Kind, domain keys…, TS) makes replay independent of
// network delivery order. TS is only the last tie-breaker and is event data.
func canonicalLess(a, b Op) bool {
	if a.Seq != b.Seq {
		return a.Seq < b.Seq
	}
	if a.Actor != b.Actor {
		return a.Actor < b.Actor
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.SKU != b.SKU {
		return a.SKU < b.SKU
	}
	if a.Customer != b.Customer {
		return a.Customer < b.Customer
	}
	if a.Subject != b.Subject {
		return a.Subject < b.Subject
	}
	if a.PolicyID != b.PolicyID {
		return a.PolicyID < b.PolicyID
	}
	return a.TS < b.TS
}

// Apply replays ops through the kernel invariants and returns the converged
// State. It is a pure function of its input, with no I/O, clock, or randomness.
// In the mesh this is the Autobase apply() reducer (compiled to wasip1); on the
// host it is an ordinary testable function.
func Apply(ops []Op) State {
	// 1. Canonicalize the order (landmine #4). Copy first — never mutate input.
	sorted := make([]Op, len(ops))
	copy(sorted, ops)
	sort.SliceStable(sorted, func(i, j int) bool { return canonicalLess(sorted[i], sorted[j]) })

	st := State{
		Stock:     make(map[string]int64),
		AR:        make(map[string]ARAccount),
		Approvals: make(map[string]ApprovalState),
		Policies:  make(map[string]PolicyState),
		Rejected:  make([]Rejection, 0),
	}

	// 2. Fold. Each domain enforces its kernel invariant; a refused op is
	//    recorded deterministically, never silently dropped or half-applied.
	for _, op := range sorted {
		var reason string
		switch op.Kind {
		case "", "inventory.move":
			reason = applyInventory(&st, op)
		case "ar.limit", "ar.charge", "ar.payment":
			reason = applyAR(&st, op)
		case "approval.decide":
			reason = applyApproval(&st, op)
		case "policy.violation", "policy.override":
			reason = applyPolicy(&st, op)
		default:
			reason = "unknown op kind " + strconv.Quote(op.Kind)
		}
		if reason != "" {
			st.Rejected = append(st.Rejected, Rejection{
				Seq: op.Seq, Actor: op.Actor, Kind: op.Kind,
				SKU: op.SKU, Delta: op.Delta, Customer: op.Customer,
				Subject: op.Subject, PolicyID: op.PolicyID, Reason: reason,
			})
			continue
		}
		st.Applied++
	}

	st.OpsHashed = len(sorted)
	st.Digest = digest(st)
	return st
}

// applyInventory enforces the Wave-0 floor invariant: stock never below zero.
func applyInventory(st *State, op Op) string {
	next := st.Stock[op.SKU] + op.Delta
	if next < 0 {
		return "floor invariant: stock would fall below 0 (have " +
			strconv.FormatInt(st.Stock[op.SKU], 10) + ", delta " +
			strconv.FormatInt(op.Delta, 10) + ")"
	}
	st.Stock[op.SKU] = next
	return ""
}

// sortedKeys returns m's keys in sorted order (landmine #1 discipline).
func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// digest computes a stable sha256 over the canonical state. It NEVER ranges a
// map directly for hashing (landmine #1): every map is hashed in sorted key
// order; rejections are already in canonical apply order.
func digest(st State) string {
	type skuKV struct {
		SKU string `json:"sku"`
		Qty int64  `json:"qty"`
	}
	type arKV struct {
		Customer string    `json:"customer"`
		Account  ARAccount `json:"account"`
	}
	type apKV struct {
		Subject string        `json:"subject"`
		State   ApprovalState `json:"state"`
	}
	type polKV struct {
		PolicyID string      `json:"policyId"`
		State    PolicyState `json:"state"`
	}
	proj := struct {
		Stock     []skuKV     `json:"stock"`
		AR        []arKV      `json:"ar"`
		Approvals []apKV      `json:"approvals"`
		Policies  []polKV     `json:"policies"`
		Rejected  []Rejection `json:"rejected"`
		Applied   int         `json:"applied"`
	}{
		Stock:     make([]skuKV, 0, len(st.Stock)),
		AR:        make([]arKV, 0, len(st.AR)),
		Approvals: make([]apKV, 0, len(st.Approvals)),
		Policies:  make([]polKV, 0, len(st.Policies)),
		Rejected:  st.Rejected,
		Applied:   st.Applied,
	}
	for _, k := range sortedKeys(st.Stock) {
		proj.Stock = append(proj.Stock, skuKV{SKU: k, Qty: st.Stock[k]})
	}
	for _, k := range sortedKeys(st.AR) {
		proj.AR = append(proj.AR, arKV{Customer: k, Account: st.AR[k]})
	}
	for _, k := range sortedKeys(st.Approvals) {
		proj.Approvals = append(proj.Approvals, apKV{Subject: k, State: st.Approvals[k]})
	}
	for _, k := range sortedKeys(st.Policies) {
		proj.Policies = append(proj.Policies, polKV{PolicyID: k, State: st.Policies[k]})
	}
	b, _ := json.Marshal(proj)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
