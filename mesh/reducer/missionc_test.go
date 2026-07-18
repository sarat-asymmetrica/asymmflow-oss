package reducer

// Mission C determinism + invariant tests: the REAL kernel packages
// (money/approval/actor/policy) as reducer law. Mirrors reducer_test.go's
// discipline: the TEST may use seeded pseudo-randomness for permutations;
// the REDUCER may not use randomness at all.

import (
	"math/rand"
	"testing"
)

// missionCOps is the canonical mixed-domain scenario. Two writers, four
// domains, and one deliberate violation per kernel invariant:
//   - inventory: dev-b's oversell (canonical loser as in Wave 0)
//   - ar/money:  a charge that breaches the credit limit + a USD op on a BHD account
//   - approval:  an AI agent trying to APPROVE (the boundary), then a human
//                approving, then an illegal approved→rejected transition
//   - policy:    an agent trying to override a violation, then a human override
func missionCOps() []Op {
	return []Op{
		// inventory (Wave-0 heritage)
		{Seq: 1, Actor: "dev-a", TS: 100, SKU: "TX-100", Delta: 10},
		{Seq: 2, Actor: "dev-a", TS: 200, SKU: "TX-100", Delta: -6},
		{Seq: 1, Actor: "dev-b", TS: 150, SKU: "TX-100", Delta: -6}, // rejected: floor

		// ar / kernel money (BHD, scale 3)
		{Seq: 3, Actor: "dev-a", TS: 300, Kind: "ar.limit", Customer: "CUST-01", LimitMinor: 500_000, Currency: "BHD"},
		{Seq: 4, Actor: "dev-a", TS: 310, Kind: "ar.charge", Customer: "CUST-01", AmountMinor: 400_000, Currency: "BHD"},
		{Seq: 2, Actor: "dev-b", TS: 320, Kind: "ar.charge", Customer: "CUST-01", AmountMinor: 200_000, Currency: "BHD"}, // rejected: limit
		{Seq: 3, Actor: "dev-b", TS: 330, Kind: "ar.payment", Customer: "CUST-01", AmountMinor: 150_000, Currency: "BHD"},
		{Seq: 5, Actor: "dev-a", TS: 340, Kind: "ar.charge", Customer: "CUST-01", AmountMinor: 100_000, Currency: "USD"}, // rejected: currency

		// approval / kernel actor+approval
		{Seq: 4, Actor: "butler-ai", TS: 400, Kind: "approval.decide", Subject: "posting-77", SubjectType: "posting_draft",
			Decision: "approved", ActorType: "agent", Authority: 1, CorrelationID: "c-1"}, // rejected: AI boundary
		{Seq: 6, Actor: "sarat", TS: 410, Kind: "approval.decide", Subject: "posting-77", SubjectType: "posting_draft",
			Decision: "approved", ActorType: "operator", Authority: 2, CorrelationID: "c-2"},
		{Seq: 7, Actor: "sarat", TS: 420, Kind: "approval.decide", Subject: "posting-77", SubjectType: "posting_draft",
			Decision: "rejected", ActorType: "operator", Authority: 2, CorrelationID: "c-3"}, // rejected: approved→rejected illegal

		// policy / kernel policy
		{Seq: 5, Actor: "dev-b", TS: 500, Kind: "policy.violation", PolicyID: "VAT-DEADLINE"},
		{Seq: 6, Actor: "butler-ai", TS: 510, Kind: "policy.override", PolicyID: "VAT-DEADLINE",
			Reason: "agent says it is fine", ActorType: "agent", Authority: 1}, // rejected: AI boundary
		{Seq: 8, Actor: "sarat", TS: 520, Kind: "policy.override", PolicyID: "VAT-DEADLINE",
			Reason: "filed via portal, receipt attached", ActorType: "operator", Authority: 3},
	}
}

func TestMissionC_InvariantsThroughKernelLaw(t *testing.T) {
	st := Apply(missionCOps())

	// inventory
	if st.Stock["TX-100"] != 4 {
		t.Fatalf("TX-100 = %d, want 4", st.Stock["TX-100"])
	}
	// ar: 400000 - 150000 = 250000; the 200000 charge and USD charge rejected
	acct := st.AR["CUST-01"]
	if acct.BalanceMinor != 250_000 || acct.LimitMinor != 500_000 || acct.Currency != "BHD" {
		t.Fatalf("AR account wrong: %+v", acct)
	}
	// approval: approved by the human, agent + illegal transition rejected
	ap := st.Approvals["posting-77"]
	if ap.Decision != "approved" || ap.Actor != "sarat" || ap.ActorType != "operator" {
		t.Fatalf("approval state wrong: %+v", ap)
	}
	// policy: overridden by the human with a reason
	pol := st.Policies["VAT-DEADLINE"]
	if pol.Status != "overridden" || pol.OverriddenBy != "sarat" || pol.Reason == "" {
		t.Fatalf("policy state wrong: %+v", pol)
	}
	// exactly 6 rejections: floor, limit, currency, agent-approve, illegal transition, agent-override
	if len(st.Rejected) != 6 {
		for _, r := range st.Rejected {
			t.Logf("rejected: %+v", r)
		}
		t.Fatalf("rejected = %d, want 6", len(st.Rejected))
	}
	if st.Applied != len(missionCOps())-6 {
		t.Fatalf("applied = %d, want %d", st.Applied, len(missionCOps())-6)
	}
}

func TestMissionC_AgentRejectionsCarryKernelWords(t *testing.T) {
	st := Apply(missionCOps())
	var agentApprove, agentOverride bool
	for _, r := range st.Rejected {
		if r.Actor == "butler-ai" && r.Kind == "approval.decide" {
			agentApprove = true
			if r.Reason == "" {
				t.Fatalf("agent approval rejection has no reason")
			}
		}
		if r.Actor == "butler-ai" && r.Kind == "policy.override" {
			agentOverride = true
		}
	}
	if !agentApprove || !agentOverride {
		t.Fatalf("expected both agent rejections (approve=%v override=%v)", agentApprove, agentOverride)
	}
}

func TestMissionC_AgentWithForgedAuthorityStillRejected(t *testing.T) {
	// An op CLAIMING an agent holds approve authority must fail at actor.New —
	// the boundary is enforced at construction, before any domain logic.
	ops := []Op{{
		Seq: 1, Actor: "rogue-ai", TS: 1, Kind: "approval.decide", Subject: "s", SubjectType: "t",
		Decision: "approved", ActorType: "agent", Authority: 3, CorrelationID: "c",
	}}
	st := Apply(ops)
	if len(st.Rejected) != 1 || st.Applied != 0 {
		t.Fatalf("forged-authority agent op must be rejected: %+v", st)
	}
}

func TestMissionC_ProposeAuthorityCannotApprove(t *testing.T) {
	// A HUMAN with only propose authority also cannot approve (SoD floor).
	ops := []Op{{
		Seq: 1, Actor: "junior", TS: 1, Kind: "approval.decide", Subject: "s", SubjectType: "t",
		Decision: "approved", ActorType: "operator", Authority: 1, CorrelationID: "c",
	}}
	st := Apply(ops)
	if len(st.Rejected) != 1 {
		t.Fatalf("propose-level human approval must be rejected: %+v", st)
	}
}

func TestMissionC_500PermutationConvergence(t *testing.T) {
	base := Apply(missionCOps())
	rng := rand.New(rand.NewSource(0xC0FFEE)) // TEST-side seed; reducer stays pure
	for i := 0; i < 500; i++ {
		ops := missionCOps()
		rng.Shuffle(len(ops), func(a, b int) { ops[a], ops[b] = ops[b], ops[a] })
		if got := Apply(ops); got.Digest != base.Digest {
			t.Fatalf("permutation %d diverged: %s != %s", i, got.Digest, base.Digest)
		}
	}
}

func TestMissionC_DoesNotMutateInput(t *testing.T) {
	ops := missionCOps()
	snapshot := make([]Op, len(ops))
	copy(snapshot, ops)
	_ = Apply(ops)
	for i := range ops {
		if ops[i] != snapshot[i] {
			t.Fatalf("Apply mutated its input at %d", i)
		}
	}
}
