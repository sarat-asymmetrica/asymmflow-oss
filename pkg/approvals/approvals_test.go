package approvals

import (
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/approval"
)

func mustActor(t *testing.T, typ actor.Type, auth actor.Authority) actor.Actor {
	t.Helper()
	a, err := actor.New(actor.Input{ID: "t-" + string(typ), DisplayName: "T", Type: typ, Authority: auth})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAssessment_CleanSubjectAutoApproves(t *testing.T) {
	a := NewAssessment("cost-1", "costing")
	a.Add(Finding{Code: "large_order", Message: "Large order - verify credit"}) // warning only
	if a.NeedsApproval() {
		t.Fatal("warning-only findings must not require approval")
	}
	if a.Decision() != approval.DecisionApproved {
		t.Fatalf("decision = %s, want auto-approved", a.Decision())
	}
	if got := a.Recommendation("proceed", "needs manager"); got != "proceed" {
		t.Fatalf("recommendation = %q", got)
	}
	if len(a.Warnings()) != 1 {
		t.Fatalf("warnings = %v", a.Warnings())
	}
}

func TestAssessment_ApprovalDemandRoutesPending(t *testing.T) {
	a := NewAssessment("cost-2", "costing")
	a.Add(Finding{Code: "low_margin", Message: "Margin below threshold", RequiresApproval: true})
	a.Add(Finding{Code: "grade_d", Message: "HIGH RISK", RequiresApproval: true, Recommendation: "Require full payment upfront or decline"})
	if a.Decision() != approval.DecisionPending {
		t.Fatalf("decision = %s, want pending", a.Decision())
	}
	// The last recommendation-carrying finding wins (sequential rule order).
	if got := a.Recommendation("proceed", "needs manager"); got != "Require full payment upfront or decline" {
		t.Fatalf("recommendation = %q", got)
	}
}

func TestTransition_OperatorApproves(t *testing.T) {
	mgr := mustActor(t, actor.TypeOperator, actor.AuthorityApprove)
	rec, err := Transition("req-1", "delete_request", approval.DecisionPending, approval.DecisionApproved, mgr, "ok", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if rec.Decision != approval.DecisionApproved || rec.ActorType != approval.ActorOperator {
		t.Fatalf("record = %+v", rec)
	}
}

// The AI-authority boundary: an agent can NEVER approve or reject, whatever
// it asks for — and the kernel additionally refuses to even construct an
// agent actor with approve authority.
func TestTransition_AgentRefusedBothDirections(t *testing.T) {
	agent := mustActor(t, actor.TypeAgent, actor.AuthorityPropose)
	for _, to := range []approval.Decision{approval.DecisionApproved, approval.DecisionRejected} {
		_, err := Transition("req-2", "delete_request", approval.DecisionPending, to, agent, "please", time.Now().UTC())
		if err == nil {
			t.Fatalf("agent was allowed to decide %s", to)
		}
		if !strings.Contains(err.Error(), "AI-authority boundary") {
			t.Fatalf("refusal must name the boundary, got: %v", err)
		}
	}
	if _, err := actor.New(actor.Input{ID: "x", DisplayName: "X", Type: actor.TypeAgent, Authority: actor.AuthorityApprove}); err == nil {
		t.Fatal("kernel constructed an agent with approve authority")
	}
}

func TestTransition_IllegalMovesRefused(t *testing.T) {
	mgr := mustActor(t, actor.TypeOperator, actor.AuthorityApprove)
	cases := []struct{ from, to approval.Decision }{
		{approval.DecisionApproved, approval.DecisionApproved}, // re-deciding
		{approval.DecisionApproved, approval.DecisionRejected}, // reversal
		{approval.DecisionSuperseded, approval.DecisionApproved},
	}
	for _, c := range cases {
		if _, err := Transition("req-3", "delete_request", c.from, c.to, mgr, "", time.Now().UTC()); err == nil {
			t.Fatalf("illegal transition %s → %s allowed", c.from, c.to)
		}
	}
}

func TestTransition_ObserverCannotDecide(t *testing.T) {
	obs := mustActor(t, actor.TypeOperator, actor.AuthorityObserve)
	if _, err := Transition("req-4", "delete_request", approval.DecisionPending, approval.DecisionApproved, obs, "", time.Now().UTC()); err == nil {
		t.Fatal("observer approved")
	}
}

func TestDecisionFromStatus(t *testing.T) {
	cases := map[string]approval.Decision{
		"pending": approval.DecisionPending, "PENDING": approval.DecisionPending,
		"needs_approval": approval.DecisionPending, "pending_review": approval.DecisionPending,
		"draft":    approval.DecisionPending,
		"approved": approval.DecisionApproved, "auto_approved": approval.DecisionApproved,
		"rejected": approval.DecisionRejected, "needs_input": approval.DecisionNeedsInput,
		"superseded": approval.DecisionSuperseded,
	}
	for in, want := range cases {
		got, err := DecisionFromStatus(in)
		if err != nil || got != want {
			t.Fatalf("DecisionFromStatus(%q) = %s, %v; want %s", in, got, err, want)
		}
	}
	if _, err := DecisionFromStatus("wat"); err == nil {
		t.Fatal("unknown status must error, not guess")
	}
}
