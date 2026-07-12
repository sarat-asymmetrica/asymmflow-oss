package policy

import (
	"encoding/json"
	"testing"
	"time"

	"ph_holdings_app/pkg/kernel/actor"
)

func mustActor(t *testing.T, in actor.Input) actor.Actor {
	t.Helper()
	a, err := actor.New(in)
	if err != nil {
		t.Fatalf("actor.New: %v", err)
	}
	return a
}

func TestNew_Valid(t *testing.T) {
	p, err := New(Input{ID: "min-margin", Version: "1", Scope: "global", RequiresEvidence: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "min-margin" || p.Version != "1" || !p.RequiresEvidence {
		t.Errorf("unexpected policy %+v", p)
	}
}

func TestNew_Rejects(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		in   Input
	}{
		{"empty id", Input{ID: " ", Version: "1"}},
		{"empty version", Input{ID: "p", Version: ""}},
		{"inverted period", Input{ID: "p", Version: "1", EffectiveFrom: now, EffectiveTo: now.Add(-time.Hour)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := New(c.in); err == nil {
				t.Errorf("expected error for %s", c.name)
			}
		})
	}
}

func TestIsEffective(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p, _ := New(Input{ID: "p", Version: "1", EffectiveFrom: base, EffectiveTo: base.AddDate(0, 0, 10)})

	if p.IsEffective(base.AddDate(0, 0, -1)) {
		t.Error("before window should not be effective")
	}
	if !p.IsEffective(base) || !p.IsEffective(base.AddDate(0, 0, 5)) || !p.IsEffective(base.AddDate(0, 0, 10)) {
		t.Error("inside window (inclusive) should be effective")
	}
	if p.IsEffective(base.AddDate(0, 0, 11)) {
		t.Error("after window should not be effective")
	}

	openEnded, _ := New(Input{ID: "p", Version: "1"})
	if !openEnded.IsEffective(base) || !openEnded.IsEffective(base.AddDate(10, 0, 0)) {
		t.Error("open-ended policy should always be effective")
	}
}

func TestEvaluate(t *testing.T) {
	noEvidence, _ := New(Input{ID: "p", Version: "1"})
	if got := noEvidence.Evaluate(true, false); got != StatusCompliant {
		t.Errorf("satisfied + no evidence requirement = %s, want compliant", got)
	}
	if got := noEvidence.Evaluate(false, true); got != StatusViolation {
		t.Errorf("unsatisfied = %s, want violation", got)
	}

	needsEvidence, _ := New(Input{ID: "p", Version: "1", RequiresEvidence: true})
	if got := needsEvidence.Evaluate(true, false); got != StatusViolation {
		t.Errorf("satisfied but missing required evidence = %s, want violation", got)
	}
	if got := needsEvidence.Evaluate(true, true); got != StatusCompliant {
		t.Errorf("satisfied + evidence = %s, want compliant", got)
	}
}

// TestOverride_AIAuthorityBoundary is the core denial test: an agent can never
// override a policy violation; an authorised non-agent can, with a reason.
func TestOverride_AIAuthorityBoundary(t *testing.T) {
	p, _ := New(Input{ID: "min-margin", Version: "1"})
	at := time.Now()

	agent := mustActor(t, actor.Input{ID: "bot", Type: actor.TypeAgent, Authority: actor.AuthorityPropose})
	if _, err := p.Override(agent, "looks fine to me", at); err == nil {
		t.Fatal("agent must not be able to override a policy")
	}

	observer := mustActor(t, actor.Input{ID: "ob", Type: actor.TypeOperator, Authority: actor.AuthorityObserve})
	if _, err := p.Override(observer, "rubber stamp", at); err == nil {
		t.Fatal("observe-only actor must not be able to override a policy")
	}

	manager := mustActor(t, actor.Input{ID: "mgr", Type: actor.TypeOperator, Authority: actor.AuthorityApprove})
	if _, err := p.Override(manager, "", at); err == nil {
		t.Error("override without a reason should be rejected")
	}
	override, err := p.Override(manager, "strategic loss-leader, approved by CFO", at)
	if err != nil {
		t.Fatalf("authorised manager override should succeed: %v", err)
	}
	if override.ActorID != "mgr" || override.PolicyID != "min-margin" || override.Reason == "" {
		t.Errorf("unexpected override record %+v", override)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	base := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	p, _ := New(Input{ID: "p-1", Version: "2", Scope: "BH-VAT", EffectiveFrom: base, RequiresEvidence: true})
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var restored Policy
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.ID != p.ID || restored.Version != p.Version || restored.Scope != p.Scope ||
		!restored.EffectiveFrom.Equal(p.EffectiveFrom) || restored.RequiresEvidence != p.RequiresEvidence {
		t.Errorf("round-trip mismatch: %+v vs %+v", restored, p)
	}
}

func TestIsKnownStatus(t *testing.T) {
	if !IsKnownStatus(StatusCompliant) || !IsKnownStatus(StatusViolation) || !IsKnownStatus(StatusOverridden) {
		t.Error("known statuses misreported")
	}
	if IsKnownStatus(Status("bogus")) {
		t.Error("unknown status reported known")
	}
}
