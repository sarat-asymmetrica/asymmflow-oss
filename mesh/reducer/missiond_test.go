// missiond_test.go — Mission D: the Ed25519 grant-with-epochs capability layer.
//
// What must hold (campaign §Mission D):
//   - transport-auth ≠ capability-auth: being able to DELIVER ops (writer set,
//     pipe) grants nothing; only a current-epoch grant signed by the mesh
//     authority makes a device's ops count.
//   - revocation = epoch bump + re-issue: grants not re-issued go stale at the
//     app layer even though the pipe still opens (proven at mesh level by
//     host/missiond-mesh.mjs; here we prove the reducer law itself).
//   - all of it deterministic: signatures are pure math; permuted delivery
//     converges to one digest.
package reducer

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"strings"
	"testing"
)

// testDevice derives a deterministic Ed25519 identity from a seed byte.
type testDevice struct {
	priv ed25519.PrivateKey
	pub  string // hex
}

func newTestDevice(seedByte byte) testDevice {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = seedByte
	}
	priv := ed25519.NewKeyFromSeed(seed)
	return testDevice{priv: priv, pub: hex.EncodeToString(priv.Public().(ed25519.PublicKey))}
}

// sign returns op with DevicePub+Sig filled — the Go mirror of the JS signOp.
func (d testDevice) sign(op Op) Op {
	op.DevicePub = d.pub
	digest := sha256.Sum256(signable(op))
	op.Sig = hex.EncodeToString(ed25519.Sign(d.priv, digest[:]))
	return op
}

var (
	authority = newTestDevice(0xA1) // the mesh authority (owner root key)
	laptop    = newTestDevice(0xB2) // granted device
	rogue     = newTestDevice(0xC3) // in the writer set, NEVER granted
)

func cfg() Config { return Config{AuthorityPub: authority.pub} }

// missionDOps builds the canonical Mission D scenario:
//   epoch 0: authority grants laptop; laptop + authority write (accepted);
//            rogue writes without a grant (rejected); rogue tries to grant
//            itself (rejected — not the authority)
//   epoch 1: authority bumps the epoch WITHOUT re-issuing laptop
//            (revocation); laptop's later op is stale-rejected
//   epoch 1: authority re-grants laptop at epoch 1; its final op is accepted
func missionDOps() []Op {
	return []Op{
		authority.sign(Op{Seq: 1, Actor: "sarat-hub", TS: 100, Kind: "cap.grant", Device: laptop.pub, Role: "writer", Epoch: 0}),
		laptop.sign(Op{Seq: 2, Actor: "laptop", TS: 200, SKU: "TX-100", Delta: 10}),
		authority.sign(Op{Seq: 3, Actor: "sarat-hub", TS: 300, SKU: "TX-100", Delta: 5}),
		rogue.sign(Op{Seq: 4, Actor: "rogue", TS: 400, SKU: "TX-100", Delta: -3}),
		rogue.sign(Op{Seq: 5, Actor: "rogue", TS: 500, Kind: "cap.grant", Device: rogue.pub, Role: "writer", Epoch: 0}),
		authority.sign(Op{Seq: 6, Actor: "sarat-hub", TS: 600, Kind: "cap.epoch", Epoch: 1}),
		laptop.sign(Op{Seq: 7, Actor: "laptop", TS: 700, SKU: "TX-100", Delta: -4}),
		authority.sign(Op{Seq: 8, Actor: "sarat-hub", TS: 800, Kind: "cap.grant", Device: laptop.pub, Role: "writer", Epoch: 1}),
		laptop.sign(Op{Seq: 9, Actor: "laptop", TS: 900, SKU: "TX-100", Delta: -2}),
	}
}

func TestMissionD_GrantEpochLifecycle(t *testing.T) {
	st := ApplyWithConfig(cfg(), missionDOps())

	// Accepted: grant, laptop@e0, authority, bump, re-grant, laptop@e1 = 6.
	if st.Applied != 6 {
		t.Fatalf("applied = %d, want 6 (rejected: %+v)", st.Applied, st.Rejected)
	}
	// Stock: +10 (laptop) +5 (authority) -2 (laptop after re-grant) = 13;
	// the rogue's -3 and the laptop's post-revocation -4 never landed.
	if st.Stock["TX-100"] != 13 {
		t.Fatalf("stock = %d, want 13", st.Stock["TX-100"])
	}
	if st.CapEpoch != 1 {
		t.Fatalf("capEpoch = %d, want 1", st.CapEpoch)
	}
	if g := st.Grants[laptop.pub]; g.Epoch != 1 || g.Role != "writer" {
		t.Fatalf("laptop grant = %+v, want epoch 1 writer", g)
	}
	if _, ok := st.Grants[rogue.pub]; ok {
		t.Fatal("rogue must never obtain a grant")
	}

	if len(st.Rejected) != 3 {
		t.Fatalf("rejected = %d, want 3: %+v", len(st.Rejected), st.Rejected)
	}
	wantReasons := map[int64]string{
		4: "no grant for device",
		5: "grants must be signed by the mesh authority",
		7: "is stale",
	}
	for _, r := range st.Rejected {
		want, ok := wantReasons[r.Seq]
		if !ok {
			t.Fatalf("unexpected rejection seq %d: %s", r.Seq, r.Reason)
		}
		if !strings.Contains(r.Reason, want) {
			t.Fatalf("seq %d reason %q must contain %q", r.Seq, r.Reason, want)
		}
	}
}

func TestMissionD_UnsignedAndForgedOpsRejected(t *testing.T) {
	grant := authority.sign(Op{Seq: 1, Actor: "sarat-hub", TS: 100, Kind: "cap.grant", Device: laptop.pub, Epoch: 0})

	unsigned := Op{Seq: 2, Actor: "laptop", TS: 200, SKU: "TX-100", Delta: 10}

	// Forgery 1: rogue signs but claims the laptop's public key.
	forged := rogue.sign(Op{Seq: 3, Actor: "laptop", TS: 300, SKU: "TX-100", Delta: 10})
	forged.DevicePub = laptop.pub

	// Forgery 2: a validly-signed laptop op, TAMPERED after signing.
	tampered := laptop.sign(Op{Seq: 4, Actor: "laptop", TS: 400, SKU: "TX-100", Delta: 1})
	tampered.Delta = 1000

	st := ApplyWithConfig(cfg(), []Op{grant, unsigned, forged, tampered})
	if st.Applied != 1 { // only the grant
		t.Fatalf("applied = %d, want 1 (rejected: %+v)", st.Applied, st.Rejected)
	}
	if st.Stock["TX-100"] != 0 {
		t.Fatalf("no forged/tampered stock may land, got %d", st.Stock["TX-100"])
	}
	for _, r := range st.Rejected {
		if !strings.Contains(r.Reason, "capability:") {
			t.Fatalf("rejection must be a capability rejection: %s", r.Reason)
		}
	}
}

func TestMissionD_EpochMustIncrease(t *testing.T) {
	ops := []Op{
		authority.sign(Op{Seq: 1, Actor: "sarat-hub", TS: 100, Kind: "cap.epoch", Epoch: 2}),
		authority.sign(Op{Seq: 2, Actor: "sarat-hub", TS: 200, Kind: "cap.epoch", Epoch: 2}), // replay
		authority.sign(Op{Seq: 3, Actor: "sarat-hub", TS: 300, Kind: "cap.epoch", Epoch: 1}), // rollback
	}
	st := ApplyWithConfig(cfg(), ops)
	if st.CapEpoch != 2 || st.Applied != 1 || len(st.Rejected) != 2 {
		t.Fatalf("epoch=%d applied=%d rejected=%d, want 2/1/2", st.CapEpoch, st.Applied, len(st.Rejected))
	}
}

func TestMissionD_TargetedRevoke(t *testing.T) {
	ops := []Op{
		authority.sign(Op{Seq: 1, Actor: "sarat-hub", TS: 100, Kind: "cap.grant", Device: laptop.pub, Epoch: 0}),
		laptop.sign(Op{Seq: 2, Actor: "laptop", TS: 200, SKU: "TX-100", Delta: 10}),
		authority.sign(Op{Seq: 3, Actor: "sarat-hub", TS: 300, Kind: "cap.revoke", Device: laptop.pub}),
		laptop.sign(Op{Seq: 4, Actor: "laptop", TS: 400, SKU: "TX-100", Delta: 10}),
	}
	st := ApplyWithConfig(cfg(), ops)
	if st.Stock["TX-100"] != 10 || len(st.Rejected) != 1 ||
		!strings.Contains(st.Rejected[0].Reason, "no grant for device") {
		t.Fatalf("targeted revoke failed: stock=%d rejected=%+v", st.Stock["TX-100"], st.Rejected)
	}
}

// The Mission C domains still run UNDER the capability plane: a granted
// device's agent-authored approval is still refused by the kernel actor
// boundary — capability says the DEVICE may write; the kernel says what the
// claimed ACTOR may do. Two independent laws, both enforced.
func TestMissionD_KernelLawStillHoldsAboveCapability(t *testing.T) {
	ops := []Op{
		authority.sign(Op{Seq: 1, Actor: "sarat-hub", TS: 100, Kind: "cap.grant", Device: laptop.pub, Epoch: 0}),
		laptop.sign(Op{Seq: 2, Actor: "butler-ai", TS: 200, Kind: "approval.decide",
			Subject: "posting-9", SubjectType: "posting_draft", Decision: "approved",
			ActorType: "agent", Authority: 1, CorrelationID: "c-9"}),
	}
	st := ApplyWithConfig(cfg(), ops)
	if len(st.Rejected) != 1 || !strings.Contains(st.Rejected[0].Reason, "agent") {
		t.Fatalf("granted device must NOT smuggle an agent approval past the kernel: %+v", st.Rejected)
	}
}

func TestMissionD_LegacyModeUntouched(t *testing.T) {
	// Without an authority, Mission C ops produce the same digest as Apply
	// (capability fields absent, goldens byte-stable — MESH-D12).
	ops := missionCOps()
	legacy := Apply(ops)
	viaCfg := ApplyWithConfig(Config{}, ops)
	if legacy.Digest != viaCfg.Digest {
		t.Fatal("empty config must be byte-identical to legacy Apply")
	}
	if legacy.Grants != nil || legacy.CapEpoch != 0 {
		t.Fatal("legacy mode must not grow a capability plane")
	}
	// And cap ops in a no-authority mesh are refused, not silently applied.
	st := Apply([]Op{authority.sign(Op{Seq: 1, Actor: "x", TS: 1, Kind: "cap.grant", Device: laptop.pub})})
	if st.Applied != 0 || len(st.Rejected) != 1 {
		t.Fatalf("cap op without authority must be rejected: %+v", st)
	}
}

func TestMissionD_500PermutationConvergence(t *testing.T) {
	ops := missionDOps()
	want := ApplyWithConfig(cfg(), ops)
	rng := rand.New(rand.NewSource(0xD))
	for i := 0; i < 500; i++ {
		shuffled := make([]Op, len(ops))
		copy(shuffled, ops)
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		got := ApplyWithConfig(cfg(), shuffled)
		if got.Digest != want.Digest {
			t.Fatalf("permutation %d diverged: %s != %s", i, got.Digest, want.Digest)
		}
	}
}

func TestMissionD_DoesNotMutateInput(t *testing.T) {
	ops := missionDOps()
	snapshot := make([]Op, len(ops))
	copy(snapshot, ops)
	_ = ApplyWithConfig(cfg(), ops)
	for i := range ops {
		if ops[i] != snapshot[i] {
			t.Fatalf("input op %d mutated", i)
		}
	}
}
