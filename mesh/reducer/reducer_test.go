package reducer

import (
	"math/rand"
	"testing"
)

// canonicalOps is the spike's fixed scenario: two devices (dev-a, dev-b) move
// two SKUs concurrently, including an oversell that MUST be rejected on merge.
// SKU "TX-100" starts with a +10 receipt then two sales of 6 and 6: the second
// 6 would take stock to -2 and is the deterministic rejection.
func canonicalOps() []Op {
	return []Op{
		{Seq: 1, Actor: "dev-a", SKU: "TX-100", Delta: +10, TS: 100},
		{Seq: 2, Actor: "dev-a", SKU: "TX-100", Delta: -6, TS: 200},
		{Seq: 1, Actor: "dev-b", SKU: "TX-100", Delta: -6, TS: 150}, // oversell -> reject
		{Seq: 1, Actor: "dev-a", SKU: "PH-200", Delta: +3, TS: 120},
		{Seq: 2, Actor: "dev-b", SKU: "PH-200", Delta: +4, TS: 220},
	}
}

func TestApply_FloorInvariantRejectsOversell(t *testing.T) {
	st := Apply(canonicalOps())

	if got := st.Stock["TX-100"]; got != 4 {
		t.Fatalf("TX-100 = %d, want 4 (10 - 6, second -6 rejected)", got)
	}
	if got := st.Stock["PH-200"]; got != 7 {
		t.Fatalf("PH-200 = %d, want 7 (3 + 4)", got)
	}
	if len(st.Rejected) != 1 {
		t.Fatalf("rejected = %d, want 1", len(st.Rejected))
	}
	// Determinism detail worth internalizing: the floor invariant guarantees the
	// STATE (TX-100 = 4) and the reject COUNT (1) identically on every peer — but
	// *which* conflicting write loses is itself fixed by the canonical order, not
	// by wall-clock or arrival. Canonical sort (Seq, Actor, SKU, TS) applies
	// dev-b's Seq-1 sale before dev-a's Seq-2 sale, so dev-a's op is the one that
	// would breach the floor and is deterministically rejected on all peers.
	if st.Rejected[0].Actor != "dev-a" || st.Rejected[0].SKU != "TX-100" || st.Rejected[0].Seq != 2 {
		t.Fatalf("wrong rejection: %+v", st.Rejected[0])
	}
	if st.Applied != 4 {
		t.Fatalf("applied = %d, want 4", st.Applied)
	}
}

// TestApply_ConvergesUnderPermutation is the heart of the determinism claim:
// no matter what order the network delivers the ops, the canonical re-sort makes
// the converged digest identical. This is the single-process analogue of the
// "3 peers converge byte-identical" gate.
func TestApply_ConvergesUnderPermutation(t *testing.T) {
	base := Apply(canonicalOps())
	rng := rand.New(rand.NewSource(20260715)) // seeded: the TEST may be random; the REDUCER may not.

	for i := 0; i < 500; i++ {
		ops := canonicalOps()
		rng.Shuffle(len(ops), func(a, b int) { ops[a], ops[b] = ops[b], ops[a] })
		got := Apply(ops)
		if got.Digest != base.Digest {
			t.Fatalf("permutation %d diverged: %s != %s", i, got.Digest, base.Digest)
		}
	}
}

// TestApply_DoesNotMutateInput guards the "copy first, never mutate input"
// discipline — a shared op slice replayed by many views must be untouched.
func TestApply_DoesNotMutateInput(t *testing.T) {
	ops := canonicalOps()
	first := ops[0]
	_ = Apply(ops)
	if ops[0] != first {
		t.Fatalf("Apply mutated its input: %+v != %+v", ops[0], first)
	}
}

// TestApply_EmptyIsStable — an empty log has a stable, non-panicking digest
// (the genesis state every fresh peer starts from).
func TestApply_EmptyIsStable(t *testing.T) {
	a := Apply(nil)
	b := Apply([]Op{})
	if a.Digest != b.Digest {
		t.Fatalf("empty digests differ: %s != %s", a.Digest, b.Digest)
	}
	if a.Applied != 0 || len(a.Rejected) != 0 {
		t.Fatalf("empty state not clean: %+v", a)
	}
}
