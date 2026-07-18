// invite_test.go — Mission M2: invites are signed grant OFFERS, enforced by
// the fold — expiry, use-count, revocation, proof-of-possession — everything
// upstream blind-pairing deliberately leaves advisory.
//
// What must hold (FABLE_CAMPAIGN_MESSENGER.md §M2 + owner rulings 2026-07-18):
//   - offers: authority-signed, derived inviteId, maxUses >= 1 (one-time is
//     the DEFAULT, set at creation — the fold refuses a zero budget)
//   - redemption: possession proof bound to the joining device; expiry decided
//     by OP-DATA time (redeem.ts vs offer.expiresAt — no clocks, MESH-D13);
//     grants materialize at the CURRENT epoch with the OFFERED role
//   - one-time invites die after one use; revoked invites die immediately;
//     a device with a current grant cannot waste a use; a STALE-epoch device
//     may re-redeem a multi-use invite to rejoin (MSG-D12)
//   - observer-role grants are read-only: every room write rejects
//   - all of it converges under permutation
package reducer

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"strings"
	"testing"
)

// inviteKey derives a deterministic invite keypair (same shape as testDevice).
type inviteKey struct {
	priv ed25519.PrivateKey
	pub  string
}

func newInviteKey(seedByte byte) inviteKey {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = seedByte
	}
	priv := ed25519.NewKeyFromSeed(seed)
	return inviteKey{priv: priv, pub: hex.EncodeToString(priv.Public().(ed25519.PublicKey))}
}

// proveFor signs the invite proof binding the invite secret to a device.
func (k inviteKey) proveFor(devicePub string) string {
	digest := sha256.Sum256(inviteProofPayload(devicePub))
	return hex.EncodeToString(ed25519.Sign(k.priv, digest[:]))
}

var (
	inviteA   = newInviteKey(0x11) // the shareable one-time invite
	inviteB   = newInviteKey(0x22) // a multi-use invite
	inviteObs = newInviteKey(0x33) // an observer-role invite
)

func offerOp(seq int64, ts int64, ik inviteKey, role string, expiresAt, maxUses int64) Op {
	return roomAuthority.sign(Op{
		Seq: seq, Actor: "hub", TS: ts, Kind: "invite.offer",
		InvitePub: ik.pub, Role: role, ExpiresAt: expiresAt, MaxUses: maxUses,
	})
}

func redeemOp(seq int64, actor string, ts int64, inviteID string, ik inviteKey, d testDevice) Op {
	return d.sign(Op{
		Seq: seq, Actor: actor, TS: ts, Kind: "invite.redeem",
		InviteID: inviteID, InviteProof: ik.proveFor(d.pub),
	})
}

func hasReject(rs RoomState, actor, part string) bool {
	for _, r := range rs.Rejected {
		if r.Actor == actor && strings.Contains(r.Reason, part) {
			return true
		}
	}
	return false
}

// ---------- offers ----------

func TestInviteOfferLaw(t *testing.T) {
	member := newTestDevice(0x55)
	rs := ApplyRoom(roomCfg(), []Op{
		roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "cap.grant", Device: member.pub, Epoch: 0}),
		offerOp(2, 200, inviteA, "", 100_000, 1), // happy: default role writer
		// a granted MEMBER may not mint invites
		member.sign(Op{Seq: 3, Actor: "desk", TS: 300, Kind: "invite.offer", InvitePub: inviteB.pub, MaxUses: 1}),
		// zero-budget, bad-role, bad-pub, forged-id offers all die
		roomAuthority.sign(Op{Seq: 4, Actor: "hub", TS: 400, Kind: "invite.offer", InvitePub: inviteB.pub, MaxUses: 0}),
		roomAuthority.sign(Op{Seq: 5, Actor: "hub", TS: 500, Kind: "invite.offer", InvitePub: inviteB.pub, MaxUses: 1, Role: "emperor"}),
		roomAuthority.sign(Op{Seq: 6, Actor: "hub", TS: 600, Kind: "invite.offer", InvitePub: "deadbeef", MaxUses: 1}),
		roomAuthority.sign(Op{Seq: 7, Actor: "hub", TS: 700, Kind: "invite.offer", InvitePub: inviteB.pub, MaxUses: 1, InviteID: "hub:999"}),
	})
	inv, ok := rs.Invites["hub:2"]
	if !ok || inv.Role != "writer" || inv.MaxUses != 1 || inv.ExpiresAt != 100_000 {
		t.Fatalf("happy offer must fold with derived id + writer default: %+v", rs.Invites)
	}
	for actor, part := range map[string]string{"desk": "must be signed by the room authority"} {
		if !hasReject(rs, actor, part) {
			t.Fatalf("missing rejection %s/%s: %+v", actor, part, rs.Rejected)
		}
	}
	for _, part := range []string{"maxUses must be >= 1", "unknown role", "32-byte hex invitePub", "inviteId must be {actor}:{seq}"} {
		if !hasReject(rs, "hub", part) {
			t.Fatalf("missing offer rejection %q: %+v", part, rs.Rejected)
		}
	}
	if len(rs.Invites) != 1 {
		t.Fatalf("only the lawful offer survives, got %+v", rs.Invites)
	}
}

// ---------- redemption ----------

func TestInviteRedeemHappyPathAndOneTime(t *testing.T) {
	joiner := newTestDevice(0x66)
	second := newTestDevice(0x67)
	rs := ApplyRoom(roomCfg(), []Op{
		offerOp(1, 100, inviteA, "", 100_000, 1),
		redeemOp(2, "joiner", 200, "hub:1", inviteA, joiner),
		// the joiner can now actually WRITE
		joiner.sign(Op{Seq: 3, Actor: "joiner", TS: 300, Kind: "msg.post", Body: "hello from the invite path"}),
		// one-time: a second device on the same invite is refused
		redeemOp(4, "second", 400, "hub:1", inviteA, second),
	})
	if g, ok := rs.Grants[joiner.pub]; !ok || g.Role != "writer" || g.Epoch != 0 {
		t.Fatalf("redemption must grant at current epoch: %+v", rs.Grants)
	}
	if msgByID(rs, "joiner:3") == nil {
		t.Fatalf("the redeemed device's message must fold: %+v", rs.Messages)
	}
	if !hasReject(rs, "second", "exhausted") {
		t.Fatalf("one-time invite must exhaust: %+v", rs.Rejected)
	}
	if rs.Invites["hub:1"].Uses != 1 {
		t.Fatalf("uses must count: %+v", rs.Invites)
	}
}

func TestInviteRedeemExpiryIsOpDataTime(t *testing.T) {
	late := newTestDevice(0x68)
	early := newTestDevice(0x69)
	rs := ApplyRoom(roomCfg(), []Op{
		offerOp(1, 100, inviteB, "", 1_000, 5), // expires at ts=1000
		redeemOp(2, "early", 900, "hub:1", inviteB, early),
		redeemOp(3, "late", 1_001, "hub:1", inviteB, late),
	})
	if _, ok := rs.Grants[early.pub]; !ok {
		t.Fatalf("pre-deadline redemption must land: %+v", rs.Grants)
	}
	if _, ok := rs.Grants[late.pub]; ok || !hasReject(rs, "late", "expired") {
		t.Fatalf("post-deadline redemption must reject by OP-DATA time: %+v / %+v", rs.Grants, rs.Rejected)
	}
}

func TestInviteProofBindsTheDevice(t *testing.T) {
	honest := newTestDevice(0x6a)
	thief := newTestDevice(0x6b)
	honestProof := inviteA.proveFor(honest.pub)
	rs := ApplyRoom(roomCfg(), []Op{
		offerOp(1, 100, inviteA, "", 0, 5),
		// the thief captured honest's proof off the wire and replays it
		thief.sign(Op{Seq: 2, Actor: "thief", TS: 200, Kind: "invite.redeem", InviteID: "hub:1", InviteProof: honestProof}),
		// a proof from the WRONG invite key
		redeemOp(3, "wrongkey", 300, "hub:1", inviteB, honest),
		// no proof at all
		honest.sign(Op{Seq: 4, Actor: "noproof", TS: 400, Kind: "invite.redeem", InviteID: "hub:1"}),
		// unknown invite
		redeemOp(5, "ghost", 500, "hub:99", inviteA, honest),
	})
	if len(rs.Grants) != 0 {
		t.Fatalf("no redemption above is lawful: %+v", rs.Grants)
	}
	for actor, part := range map[string]string{
		"thief": "invalid invite proof", "wrongkey": "invalid invite proof",
		"noproof": "invalid invite proof", "ghost": "unknown invite",
	} {
		if !hasReject(rs, actor, part) {
			t.Fatalf("missing rejection %s/%s: %+v", actor, part, rs.Rejected)
		}
	}
	if rs.Invites["hub:1"].Uses != 0 {
		t.Fatalf("failed redemptions must not consume uses: %+v", rs.Invites)
	}
}

func TestInviteRevokeAndCurrentGrantRefusal(t *testing.T) {
	joiner := newTestDevice(0x6c)
	member := newTestDevice(0x6d)
	rs := ApplyRoom(roomCfg(), []Op{
		offerOp(1, 100, inviteB, "", 0, 5),
		roomAuthority.sign(Op{Seq: 2, Actor: "hub", TS: 200, Kind: "cap.grant", Device: member.pub, Epoch: 0}),
		// a member with a CURRENT grant cannot waste a use
		redeemOp(3, "member", 300, "hub:1", inviteB, member),
		// a member may not revoke; the authority may; double-revoke dies
		member.sign(Op{Seq: 4, Actor: "member", TS: 400, Kind: "invite.revoke", InviteID: "hub:1"}),
		roomAuthority.sign(Op{Seq: 5, Actor: "hub", TS: 500, Kind: "invite.revoke", InviteID: "hub:1"}),
		roomAuthority.sign(Op{Seq: 6, Actor: "hub", TS: 600, Kind: "invite.revoke", InviteID: "hub:1"}),
		// post-revocation redemption dies even with a perfect proof
		redeemOp(7, "joiner", 700, "hub:1", inviteB, joiner),
	})
	if !rs.Invites["hub:1"].Revoked {
		t.Fatalf("authority revocation must land: %+v", rs.Invites)
	}
	if !hasReject(rs, "member", "already holds a current grant") ||
		!hasReject(rs, "member", "must be signed by the room authority") ||
		!hasReject(rs, "hub", "was revoked") ||
		!hasReject(rs, "joiner", "was revoked") {
		t.Fatalf("revoke law rejections missing: %+v", rs.Rejected)
	}
	if _, ok := rs.Grants[joiner.pub]; ok {
		t.Fatalf("revoked invite must not grant")
	}
}

func TestStaleGrantMayReRedeemMultiUse(t *testing.T) {
	device := newTestDevice(0x6e)
	rs := ApplyRoom(roomCfg(), []Op{
		offerOp(1, 100, inviteB, "", 0, 5),
		redeemOp(2, "dev", 200, "hub:1", inviteB, device),
		device.sign(Op{Seq: 3, Actor: "dev", TS: 300, Kind: "msg.post", Body: "first life"}),
		// revocation wave: epoch bump, dev NOT re-issued
		roomAuthority.sign(Op{Seq: 4, Actor: "hub", TS: 400, Kind: "cap.epoch", Epoch: 1}),
		device.sign(Op{Seq: 5, Actor: "dev", TS: 500, Kind: "msg.post", Body: "stale life"}),
		// MSG-D12: the stale device re-redeems the still-open multi-use invite
		redeemOp(6, "dev", 600, "hub:1", inviteB, device),
		device.sign(Op{Seq: 7, Actor: "dev", TS: 700, Kind: "msg.post", Body: "second life"}),
	})
	if msgByID(rs, "dev:3") == nil || msgByID(rs, "dev:7") == nil {
		t.Fatalf("pre-bump and re-redeemed messages must fold: %+v", rs.Messages)
	}
	if msgByID(rs, "dev:5") != nil || !hasReject(rs, "dev", "is stale") {
		t.Fatalf("the stale-epoch message must reject: %+v", rs.Rejected)
	}
	if g := rs.Grants[device.pub]; g.Epoch != 1 {
		t.Fatalf("re-redemption must grant at the CURRENT epoch: %+v", g)
	}
	if rs.Invites["hub:1"].Uses != 2 {
		t.Fatalf("re-redemption consumes a use: %+v", rs.Invites)
	}
}

// ---------- observer role ----------

func TestObserverGrantIsReadOnly(t *testing.T) {
	observer := newTestDevice(0x6f)
	rs := ApplyRoom(roomCfg(), []Op{
		offerOp(1, 100, inviteObs, "observer", 0, 1),
		redeemOp(2, "auditor", 200, "hub:1", inviteObs, observer),
		observer.sign(Op{Seq: 3, Actor: "auditor", TS: 300, Kind: "msg.post", Body: "just noting"}),
		observer.sign(Op{Seq: 4, Actor: "auditor", TS: 400, Kind: "msg.read", UpToActor: "hub", UpToSeq: 1}),
	})
	if g := rs.Grants[observer.pub]; g.Role != "observer" {
		t.Fatalf("observer grant must carry the role: %+v", rs.Grants)
	}
	if len(rs.Messages) != 0 || len(rs.ReadCursors) != 0 {
		t.Fatalf("observer writes must not fold: %+v / %+v", rs.Messages, rs.ReadCursors)
	}
	rejects := 0
	for _, r := range rs.Rejected {
		if strings.Contains(r.Reason, "observer grant is read-only") {
			rejects++
		}
	}
	if rejects != 2 {
		t.Fatalf("both observer writes must reject read-only, got %d: %+v", rejects, rs.Rejected)
	}
}

// ---------- determinism ----------

func inviteScenarioOps() []Op {
	joiner := newTestDevice(0x66)
	second := newTestDevice(0x67)
	observer := newTestDevice(0x6f)
	return []Op{
		roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"}),
		offerOp(2, 200, inviteA, "", 100_000, 1),
		offerOp(3, 300, inviteObs, "observer", 0, 2),
		redeemOp(4, "joiner", 400, "hub:2", inviteA, joiner),
		joiner.sign(Op{Seq: 5, Actor: "joiner", TS: 500, Kind: "msg.post", Body: "in via code"}),
		redeemOp(6, "second", 600, "hub:2", inviteA, second), // exhausted
		redeemOp(7, "auditor", 700, "hub:3", inviteObs, observer),
		observer.sign(Op{Seq: 8, Actor: "auditor", TS: 800, Kind: "msg.post", Body: "read-only means me"}),
		roomAuthority.sign(Op{Seq: 9, Actor: "hub", TS: 900, Kind: "invite.revoke", InviteID: "hub:3"}),
		redeemOp(10, "late-obs", 1000, "hub:3", inviteObs, newTestDevice(0x70)),
	}
}

func TestInviteConvergence500Permutations(t *testing.T) {
	canonical := ApplyRoom(roomCfg(), inviteScenarioOps())
	// Sanity anchors before grinding.
	if canonical.Invites["hub:2"].Uses != 1 || canonical.Invites["hub:3"].Uses != 1 {
		t.Fatalf("scenario anchor wrong: %+v", canonical.Invites)
	}
	if len(canonical.Messages) != 1 {
		t.Fatalf("only the joiner's message folds: %+v", canonical.Messages)
	}
	rng := rand.New(rand.NewSource(2203))
	for i := range 500 {
		shuffled := inviteScenarioOps()
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		if got := ApplyRoom(roomCfg(), shuffled); got.Digest != canonical.Digest {
			t.Fatalf("invite permutation %d diverged: %s != %s", i, got.Digest, canonical.Digest)
		}
	}
}

func TestInviteFreeRoomDigestHasNoInvitesSurface(t *testing.T) {
	// The Wave-1 golden's protection: rooms that never saw an invite op must
	// not grow an invites projection (nil map → omitted → digest unchanged).
	rs := ApplyRoom(roomCfg(), enforcedRoomOps())
	if rs.Invites != nil {
		t.Fatalf("invite-free room must keep a nil invite plane")
	}
}
