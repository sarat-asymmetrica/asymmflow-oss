package workflow

import (
	"encoding/json"
	"testing"

	"ph_holdings_app/pkg/kernel/actor"
)

func mustActor(t *testing.T, in actor.Input) actor.Actor {
	t.Helper()
	a, err := actor.New(in)
	if err != nil {
		t.Fatalf("actor.New(%+v): %v", in, err)
	}
	return a
}

func TestNew_StartsProposed(t *testing.T) {
	creator := mustActor(t, actor.Input{ID: "op", Type: actor.TypeOperator, Authority: actor.AuthorityApprove})
	w, err := New(Input{SubjectKey: "costing-1", SubjectType: "costing_approval", CreatedBy: creator})
	if err != nil {
		t.Fatal(err)
	}
	if w.State != StateProposed {
		t.Errorf("new workflow should start proposed, got %s", w.State)
	}
	if w.CreatedByID != "op" {
		t.Errorf("CreatedByID = %q, want op", w.CreatedByID)
	}
}

func TestNew_Rejects(t *testing.T) {
	approver := mustActor(t, actor.Input{ID: "op", Type: actor.TypeOperator, Authority: actor.AuthorityApprove})
	observer := mustActor(t, actor.Input{ID: "ob", Type: actor.TypeOperator, Authority: actor.AuthorityObserve})

	if _, err := New(Input{SubjectKey: "  ", CreatedBy: approver}); err == nil {
		t.Error("empty subject key should be rejected")
	}
	if _, err := New(Input{SubjectKey: "x", CreatedBy: observer}); err == nil {
		t.Error("observe-only actor should not be able to propose a workflow")
	}
}

// TestAgentCanProposeButNotAdvance is the core AI-authority boundary test for
// workflows: an agent may open a proposal but may not advance it.
func TestAgentCanProposeButNotAdvance(t *testing.T) {
	agent := mustActor(t, actor.Input{ID: "bot", Type: actor.TypeAgent, Authority: actor.AuthorityPropose})

	w, err := New(Input{SubjectKey: "draft-1", SubjectType: "posting_draft", CreatedBy: agent})
	if err != nil {
		t.Fatalf("agent should be able to propose: %v", err)
	}

	// The agent cannot approve its own (or any) proposal.
	if _, err := w.Advance(agent, StateApproved); err == nil {
		t.Fatal("agent must not be able to advance a workflow to approved")
	}
	// ...nor reject, nor any authority transition.
	if _, err := w.Advance(agent, StateRejected); err == nil {
		t.Fatal("agent must not be able to record a rejection either")
	}
}

func TestAdvance_HappyPath(t *testing.T) {
	op := mustActor(t, actor.Input{ID: "op", Type: actor.TypeOperator, Authority: actor.AuthorityApprove})
	w, err := New(Input{SubjectKey: "s", CreatedBy: op})
	if err != nil {
		t.Fatal(err)
	}

	approved, err := w.Advance(op, StateApproved)
	if err != nil {
		t.Fatalf("operator approve: %v", err)
	}
	if approved.State != StateApproved {
		t.Errorf("expected approved, got %s", approved.State)
	}
	// Advance is pure: the original is unchanged.
	if w.State != StateProposed {
		t.Errorf("Advance must not mutate the receiver; got %s", w.State)
	}

	executed, err := approved.Advance(op, StateExecuted)
	if err != nil {
		t.Fatalf("operator execute: %v", err)
	}
	reversed, err := executed.Advance(op, StateReversed)
	if err != nil {
		t.Fatalf("operator reverse: %v", err)
	}
	if !IsTerminal(reversed.State) {
		t.Error("reversed should be terminal")
	}
}

func TestAdvance_IllegalTransition(t *testing.T) {
	op := mustActor(t, actor.Input{ID: "op", Type: actor.TypeOperator, Authority: actor.AuthorityApprove})
	w, _ := New(Input{SubjectKey: "s", CreatedBy: op})

	// proposed → executed is illegal (must be approved first).
	if _, err := w.Advance(op, StateExecuted); err == nil {
		t.Error("proposed → executed should be illegal")
	}
	// proposed → proposed (self) is illegal.
	if _, err := w.Advance(op, StateProposed); err == nil {
		t.Error("self-transition should be illegal")
	}
}

func TestTransitionTableAndTerminals(t *testing.T) {
	if !ValidTransition(StateProposed, StateApproved) || !ValidTransition(StateApproved, StateExecuted) ||
		!ValidTransition(StateExecuted, StateReversed) || !ValidTransition(StateProposed, StateRejected) {
		t.Error("expected legal transitions reported illegal")
	}
	if ValidTransition(StateRejected, StateApproved) || ValidTransition(StateReversed, StateExecuted) {
		t.Error("terminal states must have no exits")
	}
	if !IsTerminal(StateRejected) || !IsTerminal(StateReversed) {
		t.Error("rejected and reversed must be terminal")
	}
	if IsTerminal(StateProposed) || IsTerminal(StateApproved) || IsTerminal(StateExecuted) {
		t.Error("non-terminal states reported terminal")
	}
	if IsTerminal(State("bogus")) {
		t.Error("unknown state should not be terminal")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	op := mustActor(t, actor.Input{ID: "op", Type: actor.TypeOperator, Authority: actor.AuthorityApprove})
	w, _ := New(Input{ID: "wf-1", SubjectKey: "s", SubjectType: "t", CreatedBy: op})
	approved, _ := w.Advance(op, StateApproved)

	data, err := json.Marshal(approved)
	if err != nil {
		t.Fatal(err)
	}
	var restored Workflow
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored != approved {
		t.Errorf("round-trip mismatch: %+v vs %+v", restored, approved)
	}
}
