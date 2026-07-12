package actor

import (
	"encoding/json"
	"testing"
)

func TestNew_Valid(t *testing.T) {
	a, err := New(Input{
		ID:          "u-1",
		DisplayName: "Alice",
		Type:        TypeOperator,
		Authority:   AuthorityApprove,
		Permissions: []string{"invoices:create"},
	})
	if err != nil {
		t.Fatalf("New valid operator: unexpected error %v", err)
	}
	if a.ID != "u-1" || a.Type != TypeOperator || a.Authority != AuthorityApprove {
		t.Errorf("unexpected actor: %+v", a)
	}
	if !a.HasPermission("invoices:create") {
		t.Error("expected permission claim to be present")
	}
}

func TestNew_RejectsInvalid(t *testing.T) {
	cases := []struct {
		name string
		in   Input
	}{
		{"empty id", Input{ID: "   ", Type: TypeOperator, Authority: AuthorityObserve}},
		{"unknown type", Input{ID: "x", Type: Type("alien"), Authority: AuthorityObserve}},
		{"unknown authority", Input{ID: "x", Type: TypeOperator, Authority: Authority(99)}},
		{"negative authority", Input{ID: "x", Type: TypeOperator, Authority: Authority(-1)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := New(c.in); err == nil {
				t.Errorf("expected error for %s, got nil", c.name)
			}
		})
	}
}

// TestNew_AgentCannotBeMintedWithApprovePower is an AI-authority boundary test:
// an agent actor cannot even be CONSTRUCTED with approve/admin authority.
func TestNew_AgentCannotBeMintedWithApprovePower(t *testing.T) {
	for _, auth := range []Authority{AuthorityApprove, AuthorityAdmin} {
		if _, err := New(Input{ID: "bot", Type: TypeAgent, Authority: auth}); err == nil {
			t.Errorf("expected agent with %s authority to be rejected at construction", auth)
		}
	}
	// An agent at propose/observe is fine.
	if _, err := New(Input{ID: "bot", Type: TypeAgent, Authority: AuthorityPropose}); err != nil {
		t.Errorf("agent at propose authority should be allowed: %v", err)
	}
}

// TestCanApprove_AIAuthorityBoundary is the core denial test: agents can never
// approve; non-agents need AuthorityApprove.
func TestCanApprove_AIAuthorityBoundary(t *testing.T) {
	mustNew := func(in Input) Actor {
		a, err := New(in)
		if err != nil {
			t.Fatalf("New(%+v): %v", in, err)
		}
		return a
	}

	// Agent at the highest authority it can legally hold still cannot approve.
	agent := mustNew(Input{ID: "bot", Type: TypeAgent, Authority: AuthorityPropose})
	if agent.CanApprove() {
		t.Error("agent must never be able to approve")
	}
	if !agent.CanPropose() {
		t.Error("agent at propose authority should be able to propose")
	}

	operator := mustNew(Input{ID: "op", Type: TypeOperator, Authority: AuthorityApprove})
	if !operator.CanApprove() {
		t.Error("operator with approve authority should be able to approve")
	}

	system := mustNew(Input{ID: "svc", Type: TypeSystem, Authority: AuthorityAdmin})
	if !system.CanApprove() {
		t.Error("system service with admin authority should be able to approve")
	}

	observer := mustNew(Input{ID: "ob", Type: TypeOperator, Authority: AuthorityObserve})
	if observer.CanApprove() || observer.CanPropose() {
		t.Error("observe-only operator should neither approve nor propose")
	}
}

// TestForgedAgentStructCannotApprove proves the boundary holds even if a caller
// bypasses New and hand-forges a struct claiming admin authority.
func TestForgedAgentStructCannotApprove(t *testing.T) {
	forged := Actor{ID: "evil", Type: TypeAgent, Authority: AuthorityAdmin}
	if forged.CanApprove() {
		t.Fatal("a forged agent struct must still be denied approval by CanApprove")
	}
	if !forged.IsAgent() {
		t.Error("forged agent should report IsAgent")
	}
}

func TestPermissions_AreCopied(t *testing.T) {
	src := []string{"a", "b"}
	a, err := New(Input{ID: "x", Type: TypeOperator, Authority: AuthorityObserve, Permissions: src})
	if err != nil {
		t.Fatal(err)
	}
	src[0] = "mutated"
	if a.HasPermission("mutated") || !a.HasPermission("a") {
		t.Error("New must defensively copy the Permissions slice")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	original, err := New(Input{
		ID:          "u-9",
		DisplayName: "Bob",
		Type:        TypeSystem,
		Authority:   AuthorityApprove,
		Permissions: []string{"payments:create"},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var restored Actor
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if restored.ID != original.ID || restored.Type != original.Type ||
		restored.Authority != original.Authority || !restored.HasPermission("payments:create") {
		t.Errorf("round-trip mismatch: %+v vs %+v", restored, original)
	}
	if !restored.CanApprove() {
		t.Error("restored system actor should retain approve capability")
	}
}

func TestAuthorityString(t *testing.T) {
	cases := map[Authority]string{
		AuthorityObserve: "observe",
		AuthorityPropose: "propose",
		AuthorityApprove: "approve",
		AuthorityAdmin:   "admin",
	}
	for auth, want := range cases {
		if got := auth.String(); got != want {
			t.Errorf("Authority(%d).String() = %q, want %q", int(auth), got, want)
		}
	}
}
