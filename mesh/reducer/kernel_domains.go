// kernel_domains.go — Mission C: the ACTUAL AsymmFlow kernel packages as the
// invariant law inside the mesh reducer. Each handler returns "" on success or
// a deterministic rejection reason. Every kernel call that needs time gets the
// OP's timestamp (time.UnixMilli(op.TS).UTC()) — never a clock (landmine #2).
package reducer

import (
	"time"

	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/approval"
	"ph_holdings_app/pkg/kernel/money"
	"ph_holdings_app/pkg/kernel/policy"
)

// opTime converts the op's event timestamp into the kernel's `now` parameter.
// Always UTC so no host timezone can leak into kernel-generated values.
func opTime(op Op) time.Time { return time.UnixMilli(op.TS).UTC() }

// buildActor reconstructs the acting kernel Actor from the op's claims.
// actor.New enforces the AI-authority boundary AT CONSTRUCTION: an op claiming
// an agent with approve authority is rejected before any domain logic runs.
func buildActor(op Op) (actor.Actor, string) {
	a, err := actor.New(actor.Input{
		ID:        op.Actor,
		Type:      actor.Type(op.ActorType),
		Authority: actor.Authority(op.Authority),
	})
	if err != nil {
		return actor.Actor{}, err.Error()
	}
	return a, ""
}

// applyAR enforces the credit-limit invariant with pkg/kernel/money integer
// arithmetic. Balance may go negative (overpayment); it may never exceed the
// limit. Currency mismatches are typed kernel errors, rejected deterministically.
func applyAR(st *State, op Op) string {
	acct, exists := st.AR[op.Customer]

	switch op.Kind {
	case "ar.limit":
		if op.LimitMinor < 0 {
			return "ar: credit limit must not be negative"
		}
		if op.Currency == "" {
			return "ar: limit requires a currency"
		}
		if exists && acct.Currency != op.Currency {
			return "ar: cannot change account currency from " + acct.Currency + " to " + op.Currency
		}
		acct.LimitMinor = op.LimitMinor
		acct.Currency = op.Currency
		st.AR[op.Customer] = acct
		return ""

	case "ar.charge", "ar.payment":
		if !exists {
			return "ar: no credit limit set for customer " + op.Customer
		}
		if op.AmountMinor <= 0 {
			return "ar: amount must be positive (kind picks the sign)"
		}
		// Integer money through the kernel (landmine #3): scale 3 = BHD fils.
		bal := money.FromMinor(acct.BalanceMinor, acct.Currency, 3)
		amt := money.FromMinor(op.AmountMinor, op.Currency, 3)
		var next money.Amount
		var err error
		if op.Kind == "ar.charge" {
			next, err = bal.Add(amt)
		} else {
			next, err = bal.Sub(amt)
		}
		if err != nil {
			return "ar: " + err.Error() // e.g. currency mismatch — a typed kernel error
		}
		if op.Kind == "ar.charge" && next.Minor() > acct.LimitMinor {
			return "ar: credit limit invariant: balance " + money.FromMinor(next.Minor(), acct.Currency, 3).Format() +
				" would exceed limit " + money.FromMinor(acct.LimitMinor, acct.Currency, 3).Format()
		}
		acct.BalanceMinor = next.Minor()
		st.AR[op.Customer] = acct
		return ""
	}
	return "ar: unknown kind " + op.Kind
}

// applyApproval drives the kernel approval state machine. Subjects begin
// implicitly at pending_review. The rules, ALL enforced by kernel primitives:
//   - actor.New rejects an agent claiming approve/admin authority (boundary #1)
//   - approval.NewRecord rejects an agent APPROVING regardless of claims (#2)
//   - terminal decisions (approved→rejected etc.) refuse via ValidTransition
//   - deciding approved/rejected/superseded additionally requires CanApprove;
//     needs_input/pending round-trips require CanPropose (SoD floor)
func applyApproval(st *State, op Op) string {
	a, reason := buildActor(op)
	if reason != "" {
		return reason
	}

	to := approval.Decision(op.Decision)
	current := approval.DecisionPending
	if prev, ok := st.Approvals[op.Subject]; ok {
		current = approval.Decision(prev.Decision)
	}
	if !approval.ValidTransition(current, to) {
		return "approval: invalid transition " + string(current) + " → " + string(to)
	}

	switch to {
	case approval.DecisionApproved, approval.DecisionRejected, approval.DecisionSuperseded:
		if !a.CanApprove() {
			return "approval: actor " + a.ID + " (" + string(a.Type) + "/" + a.Authority.String() +
				") cannot decide " + string(to) + " (AI-authority boundary / insufficient authority)"
		}
	default:
		if !a.CanPropose() {
			return "approval: actor " + a.ID + " cannot move a review to " + string(to)
		}
	}

	rec, err := approval.NewRecord(approval.RecordInput{
		SubjectKey:    op.Subject,
		SubjectType:   op.SubjectType,
		Decision:      to,
		Actor:         op.Actor,
		ActorType:     approval.ActorType(op.ActorType),
		Reason:        op.Reason,
		CorrelationID: op.CorrelationID,
	}, opTime(op))
	if err != nil {
		return err.Error() // incl. "agent actors cannot approve" — the kernel line itself
	}

	st.Approvals[op.Subject] = ApprovalState{
		Decision:      string(rec.Decision),
		Actor:         rec.Actor,
		ActorType:     string(rec.ActorType),
		Reason:        rec.Reason,
		CorrelationID: rec.CorrelationID,
		DecidedAtMS:   rec.Timestamp.UnixMilli(),
	}
	return ""
}

// applyPolicy records violations and lets ONLY an approver override them, with
// a mandatory reason — policy.Override is the kernel enforcement point.
func applyPolicy(st *State, op Op) string {
	switch op.Kind {
	case "policy.violation":
		if op.PolicyID == "" {
			return "policy: violation requires a policyId"
		}
		st.Policies[op.PolicyID] = PolicyState{Status: string(policy.StatusViolation)}
		return ""

	case "policy.override":
		ps, ok := st.Policies[op.PolicyID]
		if !ok || ps.Status != string(policy.StatusViolation) {
			return "policy: nothing to override for " + op.PolicyID + " (no standing violation)"
		}
		a, reason := buildActor(op)
		if reason != "" {
			return reason
		}
		p, err := policy.New(policy.Input{ID: op.PolicyID, Version: "mesh"})
		if err != nil {
			return err.Error()
		}
		ov, err := p.Override(a, op.Reason, opTime(op))
		if err != nil {
			return err.Error() // AI-authority boundary or missing reason — kernel words
		}
		st.Policies[op.PolicyID] = PolicyState{
			Status:       string(policy.StatusOverridden),
			OverriddenBy: ov.ActorID,
			Reason:       ov.Reason,
		}
		return ""
	}
	return "policy: unknown kind " + op.Kind
}
