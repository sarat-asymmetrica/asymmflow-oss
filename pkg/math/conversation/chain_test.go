package conversation

import (
	"testing"

	"ph_holdings_app/pkg/math/quaternion"
)

func TestNewChainStartsAtIdentity(t *testing.T) {
	chain := NewConversationChain()
	if !chain.State().EqualsRotation(quaternion.Identity(), 1e-12) {
		t.Fatalf("new chain state = %+v, want identity", chain.State())
	}
}

func TestAddMessageUpdatesState(t *testing.T) {
	chain := NewConversationChain()
	before := chain.State()
	chain.AddMessage("calculate pump efficiency for 42 readings")
	if chain.State().EqualsRotation(before, 1e-12) {
		t.Fatalf("state did not change after AddMessage")
	}
}

func TestCoherenceScoreMaxForSingleMessage(t *testing.T) {
	chain := NewConversationChain()
	chain.AddMessage("one message")
	if got := chain.CoherenceScore(); got != 1.0 {
		t.Fatalf("coherence for one message = %f, want 1", got)
	}
}

func TestStateAlwaysUnit(t *testing.T) {
	chain := NewConversationChain()
	for _, prompt := range []string{
		"imagine a story",
		"calculate 2+2",
		"what is Go",
		"design a dashboard",
		"verify the ledger",
		"explain delivery notes",
		"create an offer",
		"solve the issue",
		"describe the flow",
		"brainstorm options",
	} {
		chain.AddMessage(prompt)
	}
	if !chain.StateVerified() {
		t.Fatalf("chain state should remain unit")
	}
}

func TestMomentumZeroForFirstMessage(t *testing.T) {
	chain := NewConversationChain()
	chain.AddMessage("first message")
	if got := chain.Momentum(); got != 0 {
		t.Fatalf("momentum after first message = %f, want 0", got)
	}
}

func TestRegimeDriftDetection(t *testing.T) {
	chain := NewConversationChain()
	chain.AddMessage("imagine a story")
	chain.AddMessage("calculate 2+2")
	current, drifted, previous := chain.RegimeDrift()
	if !drifted {
		t.Fatalf("expected regime drift")
	}
	if current == previous {
		t.Fatalf("current and previous regimes should differ")
	}
}
