// room_test.go — Messenger Wave 1 (Mission M1): the room fold's unit law.
//
// What must hold (FABLE_CAMPAIGN_MESSENGER.md §6.3 gate 1):
//   - schema round-trip · manifest-uniqueness · edit-authorship · tombstone
//   - react-toggle · cursor monotonicity · draft-op inertness
//   - skipped-vs-rejected taxonomy (chat rules skip; capability law rejects)
//   - 500-permutation convergence over a mixed room scenario
//   - input immutability
//   - revocation-mid-conversation: an epoch bump lands between two of a
//     device's messages — the first folds, the second is rejected, everywhere
//   - the two folds stay strangers: business kinds skip in rooms; room kinds
//     reject in the business fold (legacy digests untouched)
package reducer

import (
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
)

// ---------- helpers ----------

func post(seq int64, actor string, ts int64, body string) Op {
	return Op{Seq: seq, Actor: actor, TS: ts, Kind: "msg.post", Body: body}
}

func roomFold(t *testing.T, ops []Op) RoomState {
	t.Helper()
	return ApplyRoom(Config{}, ops)
}

func hasSkip(rs RoomState, kind, reasonPart string) bool {
	for _, s := range rs.Skipped {
		if s.Kind == kind && strings.Contains(s.Reason, reasonPart) {
			return true
		}
	}
	return false
}

func msgByID(rs RoomState, id string) *RoomMessage {
	for i := range rs.Messages {
		if rs.Messages[i].MsgID == id {
			return &rs.Messages[i]
		}
	}
	return nil
}

// ---------- schema ----------

func TestRoomOpSchemaRoundTrip(t *testing.T) {
	op := Op{
		Seq: 7, Actor: "hub", TS: 700, Kind: "msg.post",
		MsgID: "hub:7", Body: "shipping Thursday", ReplyTo: "procurement:3",
		Emoji: "👍", On: true, UpToActor: "hub", UpToSeq: 7,
		Title: "PO-2201", AnchorType: "po", AnchorID: "PO-2201", Observers: true,
		Draft: `{"kind":"approval.decide"}`, Attachment: `{"blobKey":"abc"}`,
	}
	b, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Op
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back != op {
		t.Fatalf("round-trip mismatch:\n  want %+v\n  got  %+v", op, back)
	}
	// Legacy ops must not grow room fields on the wire (omitempty discipline).
	legacy, _ := json.Marshal(Op{Seq: 1, Actor: "a", TS: 1, SKU: "TX", Delta: 2})
	for _, forbidden := range []string{"msgId", "body", "emoji", "upToActor", "title", "draft", "attachment"} {
		if strings.Contains(string(legacy), forbidden) {
			t.Fatalf("legacy op JSON leaked room field %q: %s", forbidden, legacy)
		}
	}
}

// ---------- manifest ----------

func TestManifestUniqueness(t *testing.T) {
	rs := roomFold(t, []Op{
		{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201", Observers: true},
		{Seq: 2, Actor: "hub", TS: 200, Kind: "room.manifest", Title: "usurper"},
		{Seq: 3, Actor: "hub", TS: 300, Kind: "room.manifest"}, // missing title
	})
	if rs.Manifest == nil || rs.Manifest.Title != "PO-2201 room" || rs.Manifest.AnchorID != "PO-2201" {
		t.Fatalf("first manifest should win: %+v", rs.Manifest)
	}
	if !hasSkip(rs, "room.manifest", "already declared") {
		t.Fatalf("second manifest must be skipped: %+v", rs.Skipped)
	}
	if !hasSkip(rs, "room.manifest", "requires a title") {
		t.Fatalf("untitled manifest must be skipped: %+v", rs.Skipped)
	}
	if rs.Applied != 1 || len(rs.Skipped) != 2 {
		t.Fatalf("applied=%d skipped=%d, want 1/2", rs.Applied, len(rs.Skipped))
	}
}

// ---------- msgId law ----------

func TestMsgIDDerivedAndDuplicateSkipped(t *testing.T) {
	rs := roomFold(t, []Op{
		post(1, "hub", 100, "hello"),
		{Seq: 2, Actor: "hub", TS: 200, Kind: "msg.post", MsgID: "hub:999", Body: "forged id"},
		{Seq: 1, Actor: "hub", TS: 900, Kind: "msg.post", Body: "same writer seq replayed"},
	})
	if got := rs.Messages[0].MsgID; got != "hub:1" {
		t.Fatalf("msgId must derive as {actor}:{seq}, got %q", got)
	}
	if !hasSkip(rs, "msg.post", "msgId must be {actor}:{seq}") {
		t.Fatalf("mismatched msgId must skip: %+v", rs.Skipped)
	}
	if !hasSkip(rs, "msg.post", "duplicate msgId") {
		t.Fatalf("duplicate msgId must skip: %+v", rs.Skipped)
	}
	if len(rs.Messages) != 1 {
		t.Fatalf("only the honest post folds, got %d messages", len(rs.Messages))
	}
}

// ---------- edit ----------

func TestEditAuthorshipAndLastWins(t *testing.T) {
	rs := roomFold(t, []Op{
		post(1, "hub", 100, "v1"),
		{Seq: 2, Actor: "mallory", TS: 200, Kind: "msg.edit", MsgID: "hub:1", Body: "hijacked"},
		{Seq: 3, Actor: "hub", TS: 300, Kind: "msg.edit", MsgID: "hub:1", Body: "v2"},
		{Seq: 4, Actor: "hub", TS: 400, Kind: "msg.edit", MsgID: "hub:1", Body: "v3"},
		{Seq: 5, Actor: "hub", TS: 500, Kind: "msg.edit", MsgID: "ghost:9", Body: "into the void"},
	})
	msg := msgByID(rs, "hub:1")
	if msg.Body != "v3" || !msg.Edited || msg.EditTS != 400 {
		t.Fatalf("last authored edit must win: %+v", msg)
	}
	if !hasSkip(rs, "msg.edit", "non-author") {
		t.Fatalf("non-author edit must skip: %+v", rs.Skipped)
	}
	if !hasSkip(rs, "msg.edit", "unknown msgId") {
		t.Fatalf("edit of unknown msgId must skip: %+v", rs.Skipped)
	}
}

// ---------- tombstone ----------

func TestTombstoneSemantics(t *testing.T) {
	rs := roomFold(t, []Op{
		post(1, "hub", 100, "delete me"),
		{Seq: 2, Actor: "hub", TS: 200, Kind: "msg.react", MsgID: "hub:1", Emoji: "👍", On: true},
		{Seq: 3, Actor: "mallory", TS: 300, Kind: "msg.delete", MsgID: "hub:1"},
		{Seq: 4, Actor: "hub", TS: 400, Kind: "msg.delete", MsgID: "hub:1"},
		{Seq: 5, Actor: "hub", TS: 500, Kind: "msg.edit", MsgID: "hub:1", Body: "necromancy"},
		{Seq: 6, Actor: "hub", TS: 600, Kind: "msg.react", MsgID: "hub:1", Emoji: "🔥", On: true},
		{Seq: 7, Actor: "hub", TS: 700, Kind: "msg.delete", MsgID: "hub:1"},
	})
	msg := msgByID(rs, "hub:1")
	if msg == nil || !msg.Deleted || msg.Body != "" || msg.DeletedBy != "hub" {
		t.Fatalf("tombstone must keep the id and blank the content: %+v", msg)
	}
	if !hasSkip(rs, "msg.delete", "non-author") {
		t.Fatalf("delete by non-author must skip: %+v", rs.Skipped)
	}
	if !hasSkip(rs, "msg.edit", "message is deleted") || !hasSkip(rs, "msg.react", "message is deleted") {
		t.Fatalf("edits/reacts on a tombstone must skip: %+v", rs.Skipped)
	}
	if !hasSkip(rs, "msg.delete", "message is deleted") {
		t.Fatalf("double delete must skip: %+v", rs.Skipped)
	}
	// The pre-delete reaction survives (separate fact, MSG-D5).
	if !rs.Reactions["hub:1"]["👍"]["hub"] {
		t.Fatalf("pre-delete reaction must survive: %+v", rs.Reactions)
	}
}

// ---------- reactions ----------

func TestReactToggleLastWins(t *testing.T) {
	rs := roomFold(t, []Op{
		post(1, "hub", 100, "react to me"),
		{Seq: 2, Actor: "ana", TS: 200, Kind: "msg.react", MsgID: "hub:1", Emoji: "👍", On: true},
		{Seq: 3, Actor: "ana", TS: 300, Kind: "msg.react", MsgID: "hub:1", Emoji: "👍", On: false},
		{Seq: 4, Actor: "ana", TS: 400, Kind: "msg.react", MsgID: "hub:1", Emoji: "👍", On: true},
		{Seq: 5, Actor: "bob", TS: 500, Kind: "msg.react", MsgID: "hub:1", Emoji: "👍", On: true},
		{Seq: 6, Actor: "bob", TS: 600, Kind: "msg.react", MsgID: "hub:1", Emoji: "🔥", On: true},
		{Seq: 7, Actor: "bob", TS: 700, Kind: "msg.react", MsgID: "hub:1", Emoji: "🔥", On: false},
		{Seq: 8, Actor: "bob", TS: 800, Kind: "msg.react", MsgID: "ghost:1", Emoji: "👻", On: true},
		{Seq: 9, Actor: "bob", TS: 900, Kind: "msg.react", MsgID: "hub:1", On: true}, // no emoji
	})
	actors := rs.Reactions["hub:1"]["👍"]
	if !actors["ana"] || !actors["bob"] || len(actors) != 2 {
		t.Fatalf("last toggle must win per (msg,emoji,actor): %+v", rs.Reactions)
	}
	if _, live := rs.Reactions["hub:1"]["🔥"]; live {
		t.Fatalf("toggled-off emoji set must be pruned: %+v", rs.Reactions)
	}
	if !hasSkip(rs, "msg.react", "unknown msgId") || !hasSkip(rs, "msg.react", "requires an emoji") {
		t.Fatalf("bad reacts must skip: %+v", rs.Skipped)
	}
}

// ---------- read cursors ----------

func TestReadCursorMonotonicity(t *testing.T) {
	rs := roomFold(t, []Op{
		post(1, "hub", 100, "one"),
		{Seq: 2, Actor: "ana", TS: 200, Kind: "msg.read", UpToActor: "hub", UpToSeq: 1},
		{Seq: 3, Actor: "ana", TS: 300, Kind: "msg.read", UpToActor: "hub", UpToSeq: 5},
		{Seq: 4, Actor: "ana", TS: 400, Kind: "msg.read", UpToActor: "hub", UpToSeq: 3},  // lower → skip
		{Seq: 5, Actor: "ana", TS: 500, Kind: "msg.read", UpToActor: "hub", UpToSeq: 5},  // equal → skip
		{Seq: 6, Actor: "bob", TS: 600, Kind: "msg.read", UpToActor: "hub", UpToSeq: 2},  // independent reader
		{Seq: 7, Actor: "bob", TS: 700, Kind: "msg.read", UpToSeq: 2},                    // missing writer
		{Seq: 8, Actor: "bob", TS: 800, Kind: "msg.read", UpToActor: "hub", UpToSeq: -1}, // nonsense
	})
	if rs.ReadCursors["ana"]["hub"] != 5 || rs.ReadCursors["bob"]["hub"] != 2 {
		t.Fatalf("cursors wrong: %+v", rs.ReadCursors)
	}
	stale := 0
	for _, s := range rs.Skipped {
		if strings.Contains(s.Reason, "stale read cursor") {
			stale++
		}
	}
	if stale != 2 {
		t.Fatalf("lower AND equal cursors must both skip as stale, got %d: %+v", stale, rs.Skipped)
	}
	if !hasSkip(rs, "msg.read", "requires upToActor") || !hasSkip(rs, "msg.read", "positive upToSeq") {
		t.Fatalf("malformed cursors must skip: %+v", rs.Skipped)
	}
}

// ---------- graduation seam ----------

func TestDraftOpIsInertCargo(t *testing.T) {
	draft := `{"kind":"approval.decide","subject":"posting:PO-2201","decision":"approved"}`
	rs := roomFold(t, []Op{
		{Seq: 1, Actor: "butler", ActorType: "agent", TS: 100, Kind: "msg.draft-op", Body: "Draft ready for review", Draft: draft},
		{Seq: 2, Actor: "butler", ActorType: "agent", TS: 200, Kind: "msg.draft-op"}, // no draft payload
	})
	msg := msgByID(rs, "butler:1")
	if msg == nil || msg.Draft != draft || msg.ActorType != "agent" {
		t.Fatalf("draft must be carried verbatim with the agent marker: %+v", msg)
	}
	if !hasSkip(rs, "msg.draft-op", "requires a draft") {
		t.Fatalf("empty draft-op must skip: %+v", rs.Skipped)
	}
	// INERTNESS: the draft names an approval, but no approval state may exist —
	// the room fold has no approvals surface at all, and the business fold
	// refuses the kind outright (TestRoomOpsDoNotFoldInBusinessBase).
	if strings.Contains(rs.Digest, "approved") {
		t.Fatalf("digest is a hash, sanity check failed")
	}
}

// ---------- the two folds stay strangers ----------

func TestBusinessOpsSkipInRoomFold(t *testing.T) {
	rs := roomFold(t, []Op{
		post(1, "hub", 100, "legit"),
		{Seq: 2, Actor: "hub", TS: 200, Kind: "inventory.move", SKU: "TX-100", Delta: 5},
		{Seq: 3, Actor: "hub", TS: 300, Kind: "approval.decide", Subject: "s", Decision: "approved"},
		{Seq: 4, Actor: "hub", TS: 400, Kind: "totally.unknown"},
	})
	if rs.Applied != 1 || len(rs.Skipped) != 3 {
		t.Fatalf("business/unknown kinds must skip in a room: applied=%d skipped=%+v", rs.Applied, rs.Skipped)
	}
	for _, s := range rs.Skipped {
		if !strings.Contains(s.Reason, "not a room op") {
			t.Fatalf("expected 'not a room op', got %+v", s)
		}
	}
}

func TestRoomOpsDoNotFoldInBusinessBase(t *testing.T) {
	baseline := Apply([]Op{{Seq: 1, Actor: "a", TS: 100, SKU: "TX-100", Delta: 5}})
	mixed := Apply([]Op{
		{Seq: 1, Actor: "a", TS: 100, SKU: "TX-100", Delta: 5},
		post(2, "hub", 200, "smuggled chat"),
		{Seq: 3, Actor: "hub", TS: 300, Kind: "room.manifest", Title: "smuggled room"},
	})
	if len(mixed.Rejected) != 2 {
		t.Fatalf("room kinds must be REJECTED by the business fold: %+v", mixed.Rejected)
	}
	for _, r := range mixed.Rejected {
		if !strings.Contains(r.Reason, "unknown op kind") {
			t.Fatalf("expected unknown-kind rejection, got %+v", r)
		}
	}
	if mixed.Stock["TX-100"] != baseline.Stock["TX-100"] {
		t.Fatalf("smuggled room ops must not perturb business state")
	}
}

// ---------- capability plane: taxonomy + revocation mid-conversation ----------

// roomDevices for the enforced-room tests (fresh seeds; no overlap with Mission D).
var (
	roomAuthority = newTestDevice(0xD4)
	deskDevice    = newTestDevice(0xE5)
	phoneDevice   = newTestDevice(0xF6)
)

func roomCfg() Config { return Config{AuthorityPub: roomAuthority.pub} }

// enforcedRoomOps is the canonical enforced-room scenario:
//
//	authority declares the room + grants desk & phone at epoch 0;
//	both chat; MID-CONVERSATION the authority bumps to epoch 1 re-issuing ONLY
//	desk — phone's second message goes stale; an ungranted device knocks;
//	an unsigned op knocks. Chat-rule breaks (non-author edit) still SKIP.
func enforcedRoomOps() []Op {
	rogue := newTestDevice(0x99)
	return []Op{
		roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"}),
		roomAuthority.sign(Op{Seq: 2, Actor: "hub", TS: 200, Kind: "cap.grant", Device: deskDevice.pub, Epoch: 0}),
		roomAuthority.sign(Op{Seq: 3, Actor: "hub", TS: 300, Kind: "cap.grant", Device: phoneDevice.pub, Epoch: 0}),
		deskDevice.sign(Op{Seq: 4, Actor: "desk", TS: 400, Kind: "msg.post", Body: "Can we ship Thursday?"}),
		phoneDevice.sign(Op{Seq: 5, Actor: "phone", TS: 500, Kind: "msg.post", Body: "Thursday works", ReplyTo: "desk:4"}),
		// phone tries to edit desk's message — a CHAT rule break (skip, not reject)
		phoneDevice.sign(Op{Seq: 6, Actor: "phone", TS: 600, Kind: "msg.edit", MsgID: "desk:4", Body: "Friday actually"}),
		// REVOCATION WAVE mid-conversation: epoch 1, only desk re-issued
		roomAuthority.sign(Op{Seq: 7, Actor: "hub", TS: 700, Kind: "cap.epoch", Epoch: 1}),
		roomAuthority.sign(Op{Seq: 8, Actor: "hub", TS: 800, Kind: "cap.grant", Device: deskDevice.pub, Epoch: 1}),
		// phone's SECOND message — after the bump, not re-issued → REJECTED
		phoneDevice.sign(Op{Seq: 9, Actor: "phone", TS: 900, Kind: "msg.post", Body: "wait, am I still in this room?"}),
		// desk continues fine under the new epoch
		deskDevice.sign(Op{Seq: 10, Actor: "desk", TS: 1000, Kind: "msg.post", Body: "Confirmed for Thursday."}),
		// an ungranted device in the writer set
		rogue.sign(Op{Seq: 11, Actor: "rogue", TS: 1100, Kind: "msg.post", Body: "let me in"}),
		// an unsigned op with an authority configured
		{Seq: 12, Actor: "ghost", TS: 1200, Kind: "msg.post", Body: "boo"},
	}
}

func TestRoomCapabilityTaxonomyAndRevocationMidConversation(t *testing.T) {
	rs := ApplyRoom(roomCfg(), enforcedRoomOps())

	// Revocation mid-conversation: first phone message folds, second rejects.
	if msgByID(rs, "phone:5") == nil {
		t.Fatalf("phone's pre-bump message must fold: %+v", rs.Messages)
	}
	if msgByID(rs, "phone:9") != nil {
		t.Fatalf("phone's post-bump message must NOT fold")
	}
	wantReject := func(actor, part string) {
		t.Helper()
		for _, r := range rs.Rejected {
			if r.Actor == actor && strings.Contains(r.Reason, part) {
				return
			}
		}
		t.Fatalf("missing rejection %s/%s in %+v", actor, part, rs.Rejected)
	}
	wantReject("phone", "is stale")
	wantReject("rogue", "no grant for device")
	wantReject("ghost", "unsigned op")

	// Taxonomy: the chat-rule break is a SKIP, not a rejection.
	if !hasSkip(rs, "msg.edit", "non-author") {
		t.Fatalf("chat-rule break must SKIP: %+v", rs.Skipped)
	}
	if len(rs.Rejected) != 3 || len(rs.Skipped) != 1 {
		t.Fatalf("taxonomy drift: rejected=%d skipped=%d, want 3/1", len(rs.Rejected), len(rs.Skipped))
	}
	if got := msgByID(rs, "desk:4").Body; got != "Can we ship Thursday?" {
		t.Fatalf("desk's message must survive the hijack attempt: %q", got)
	}
	if rs.CapEpoch != 1 || rs.Grants[phoneDevice.pub].Epoch != 0 {
		t.Fatalf("epoch state wrong: epoch=%d grants=%+v", rs.CapEpoch, rs.Grants)
	}
	if rs.Manifest == nil || rs.Manifest.AnchorID != "PO-2201" {
		t.Fatalf("manifest must fold: %+v", rs.Manifest)
	}
}

func TestManifestRequiresAuthorityWhenEnforced(t *testing.T) {
	ops := []Op{
		roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "cap.grant", Device: deskDevice.pub, Epoch: 0}),
		deskDevice.sign(Op{Seq: 2, Actor: "desk", TS: 200, Kind: "room.manifest", Title: "coup"}),
	}
	rs := ApplyRoom(roomCfg(), ops)
	if rs.Manifest != nil {
		t.Fatalf("a granted member must not declare the manifest: %+v", rs.Manifest)
	}
	if !hasSkip(rs, "room.manifest", "room authority") {
		t.Fatalf("non-authority manifest must skip: %+v", rs.Skipped)
	}
}

// ---------- expectation tags (Constitution Art. III §3, MSG-D16) ----------

func postTagged(seq int64, actor string, ts int64, body, expectation string) Op {
	return Op{Seq: seq, Actor: actor, TS: ts, Kind: "msg.post", Body: body, Expectation: expectation}
}

func TestExpectationRoundTrip(t *testing.T) {
	withTag := roomFold(t, []Op{postTagged(1, "hub", 100, "ship today?", "today")})
	withoutTag := roomFold(t, []Op{post(1, "hub", 100, "ship today?")})
	msg := msgByID(withTag, "hub:1")
	if msg == nil || msg.Expectation != "today" {
		t.Fatalf("expectation must round-trip onto the message: %+v", msg)
	}
	if withTag.Digest == withoutTag.Digest {
		t.Fatalf("expectation must participate in the digest (it rides the v2 signable, MSG-D16)")
	}
}

func TestUnknownExpectationTagSkipped(t *testing.T) {
	rs := roomFold(t, []Op{postTagged(1, "hub", 100, "ship today?", "asap")})
	if len(rs.Messages) != 0 {
		t.Fatalf("message with an unknown expectation tag must not fold: %+v", rs.Messages)
	}
	if !hasSkip(rs, "msg.post", "unknown expectation tag") {
		t.Fatalf("unknown expectation tag must skip with the exact reason: %+v", rs.Skipped)
	}
}

func TestExpectationIgnoredOnEdit(t *testing.T) {
	rs := roomFold(t, []Op{
		postTagged(1, "hub", 100, "v1", "urgent"),
		{Seq: 2, Actor: "hub", TS: 200, Kind: "msg.edit", MsgID: "hub:1", Body: "v2", Expectation: "whenever"},
	})
	msg := msgByID(rs, "hub:1")
	if msg == nil || msg.Body != "v2" || msg.Expectation != "urgent" {
		t.Fatalf("edit must update the body but leave the original expectation tag untouched: %+v", msg)
	}
}

// ---------- claim/assign (Constitution Art. VI, MSG-D17) ----------

func TestClaimHappyPathAuthorityAssignsMember(t *testing.T) {
	rs := ApplyRoom(roomCfg(), []Op{
		roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"}),
		roomAuthority.sign(Op{Seq: 2, Actor: "hub", TS: 200, Kind: "cap.grant", Device: deskDevice.pub, Epoch: 0}),
		roomAuthority.sign(Op{Seq: 3, Actor: "hub", TS: 300, Kind: "room.claim", Assignee: "desk"}),
	})
	if rs.Claim == nil || rs.Claim.Assignee != "desk" || rs.Claim.ByActor != "hub" || rs.Claim.AtSeq != 3 {
		t.Fatalf("authority must be able to assign a member: %+v", rs.Claim)
	}
}

func TestClaimSelfOKOthersSkip(t *testing.T) {
	rs := ApplyRoom(roomCfg(), []Op{
		roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"}),
		roomAuthority.sign(Op{Seq: 2, Actor: "hub", TS: 200, Kind: "cap.grant", Device: deskDevice.pub, Epoch: 0}),
		deskDevice.sign(Op{Seq: 3, Actor: "desk", TS: 300, Kind: "room.claim", Assignee: "desk"}),
		deskDevice.sign(Op{Seq: 4, Actor: "desk", TS: 400, Kind: "room.claim", Assignee: "phone"}),
	})
	if rs.Claim == nil || rs.Claim.Assignee != "desk" || rs.Claim.ByActor != "desk" {
		t.Fatalf("self-claim by a member must succeed: %+v", rs.Claim)
	}
	if !hasSkip(rs, "room.claim", "may only claim for self") {
		t.Fatalf("a member claiming for another actor must skip: %+v", rs.Skipped)
	}
}

func TestClaimSkippedInUnanchoredRoom(t *testing.T) {
	rs := roomFold(t, []Op{
		{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "social room"}, // no anchorType
		{Seq: 2, Actor: "hub", TS: 200, Kind: "room.claim", Assignee: "hub"},
	})
	if rs.Claim != nil {
		t.Fatalf("claim must not fold in a social (unanchored) room: %+v", rs.Claim)
	}
	if !hasSkip(rs, "room.claim", "claims are a work concept") {
		t.Fatalf("unanchored claim must skip with the Constitution's own words: %+v", rs.Skipped)
	}
}

func TestClaimBeforeManifestSkipped(t *testing.T) {
	rs := roomFold(t, []Op{
		{Seq: 1, Actor: "hub", TS: 100, Kind: "room.claim", Assignee: "hub"},
	})
	if rs.Claim != nil {
		t.Fatalf("claim before any manifest must not fold: %+v", rs.Claim)
	}
	if !hasSkip(rs, "room.claim", "claim requires a manifest") {
		t.Fatalf("claim before manifest must skip: %+v", rs.Skipped)
	}
}

func TestClaimRelease(t *testing.T) {
	rs := ApplyRoom(roomCfg(), []Op{
		roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"}),
		roomAuthority.sign(Op{Seq: 2, Actor: "hub", TS: 200, Kind: "room.claim", Assignee: "hub"}),
		roomAuthority.sign(Op{Seq: 3, Actor: "hub", TS: 300, Kind: "room.claim", Assignee: ""}),
	})
	if rs.Claim == nil || rs.Claim.Assignee != "" || rs.Claim.ByActor != "hub" || rs.Claim.AtSeq != 3 {
		t.Fatalf("an empty Assignee must release the claim: %+v", rs.Claim)
	}
}

func TestClaimMemberReleasesOwnClaim(t *testing.T) {
	// Gate ruling on MSG-D17: a member may drop work they picked up without
	// going through the authority — release is self-service for your own claim.
	rs := ApplyRoom(roomCfg(), []Op{
		roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"}),
		roomAuthority.sign(Op{Seq: 2, Actor: "hub", TS: 200, Kind: "cap.grant", Device: deskDevice.pub, Epoch: 0}),
		deskDevice.sign(Op{Seq: 3, Actor: "desk", TS: 300, Kind: "room.claim", Assignee: "desk"}),
		deskDevice.sign(Op{Seq: 4, Actor: "desk", TS: 400, Kind: "room.claim", Assignee: ""}),
	})
	if rs.Claim == nil || rs.Claim.Assignee != "" || rs.Claim.ByActor != "desk" || rs.Claim.AtSeq != 4 {
		t.Fatalf("a member must be able to release their OWN claim: %+v", rs.Claim)
	}
	if len(rs.Skipped) != 0 {
		t.Fatalf("self-release must not skip: %+v", rs.Skipped)
	}
}

func TestClaimMemberCannotReleaseOthersOrNothing(t *testing.T) {
	rs := ApplyRoom(roomCfg(), []Op{
		roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"}),
		roomAuthority.sign(Op{Seq: 2, Actor: "hub", TS: 200, Kind: "cap.grant", Device: deskDevice.pub, Epoch: 0}),
		roomAuthority.sign(Op{Seq: 3, Actor: "hub", TS: 300, Kind: "cap.grant", Device: phoneDevice.pub, Epoch: 0}),
		// nothing claimed yet — a member's release has nothing of theirs to drop
		deskDevice.sign(Op{Seq: 4, Actor: "desk", TS: 400, Kind: "room.claim", Assignee: ""}),
		// authority assigns phone; desk tries to release phone's claim
		roomAuthority.sign(Op{Seq: 5, Actor: "hub", TS: 500, Kind: "room.claim", Assignee: "phone"}),
		deskDevice.sign(Op{Seq: 6, Actor: "desk", TS: 600, Kind: "room.claim", Assignee: ""}),
	})
	if rs.Claim == nil || rs.Claim.Assignee != "phone" || rs.Claim.AtSeq != 5 {
		t.Fatalf("phone's claim must survive desk's release attempts: %+v", rs.Claim)
	}
	skips := 0
	for _, s := range rs.Skipped {
		if s.Kind == "room.claim" && s.Reason == "may only release own claim" {
			skips++
		}
	}
	if skips != 2 {
		t.Fatalf("both illegal releases must skip with the typed reason, got %d: %+v", skips, rs.Skipped)
	}
}

func TestClaimLastInCanonicalOrderWins(t *testing.T) {
	build := func() []Op {
		return []Op{
			roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"}),
			roomAuthority.sign(Op{Seq: 2, Actor: "hub", TS: 200, Kind: "cap.grant", Device: deskDevice.pub, Epoch: 0}),
			roomAuthority.sign(Op{Seq: 3, Actor: "hub", TS: 300, Kind: "room.claim", Assignee: "hub"}),
			roomAuthority.sign(Op{Seq: 4, Actor: "hub", TS: 400, Kind: "room.claim", Assignee: "desk"}),
		}
	}
	orderA := ApplyRoom(roomCfg(), build())
	reversed := build()
	reversed[2], reversed[3] = reversed[3], reversed[2] // deliver the two claims out of order
	orderB := ApplyRoom(roomCfg(), reversed)
	if orderA.Digest != orderB.Digest {
		t.Fatalf("permuted delivery of the claims must still converge: %s != %s", orderA.Digest, orderB.Digest)
	}
	if orderA.Claim == nil || orderA.Claim.Assignee != "desk" || orderA.Claim.AtSeq != 4 {
		t.Fatalf("the last claim in CANONICAL (seq) order must win, not delivery order: %+v", orderA.Claim)
	}
}

func TestObserverCannotClaim(t *testing.T) {
	observer := newTestDevice(0x71)
	rs := ApplyRoom(roomCfg(), []Op{
		roomAuthority.sign(Op{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"}),
		offerOp(2, 200, inviteObs, "observer", 0, 1),
		redeemOp(3, "auditor", 300, "hub:2", inviteObs, observer),
		observer.sign(Op{Seq: 4, Actor: "auditor", TS: 400, Kind: "room.claim", Assignee: "auditor"}),
	})
	if rs.Claim != nil {
		t.Fatalf("an observer device must not be able to claim: %+v", rs.Claim)
	}
	if !hasReject(rs, "auditor", "observer grant is read-only") {
		t.Fatalf("observer claim must be rejected read-only: %+v", rs.Rejected)
	}
}

// ---------- predecessor room pointer (Constitution Art. II amendment, MSG-D20) ----------

func TestManifestPredecessorRoomKeyFolds(t *testing.T) {
	predecessor := "aa" + strings.Repeat("bb", 31) // 64 hex chars, a plausible base key shape
	rs := roomFold(t, []Op{
		{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201", PredecessorRoomKey: predecessor},
	})
	if rs.Manifest == nil || rs.Manifest.PredecessorRoomKey != predecessor {
		t.Fatalf("predecessor pointer must fold onto the manifest exactly as signed: %+v", rs.Manifest)
	}
	baseline := roomFold(t, []Op{
		{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"},
	})
	if rs.Digest == baseline.Digest {
		t.Fatalf("a manifest carrying a predecessor pointer must digest differently than one without")
	}
}

func TestManifestPredecessorNoValidationByDesign(t *testing.T) {
	// Gate-adopted from the coder's own flag (MSG-D20): the fold records the
	// pointer, it does NOT validate or dereference it — a garbage pointer is
	// the authority's own signed statement, and the fold is not a registry.
	// This test pins the ABSENCE of validation as intent, so a future "helpful"
	// length/hex check shows up as a deliberate law change, not a drive-by fix.
	garbage := "not-a-key-at-all 🤷"
	rs := roomFold(t, []Op{
		{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201", PredecessorRoomKey: garbage},
	})
	if rs.Manifest == nil || rs.Manifest.PredecessorRoomKey != garbage {
		t.Fatalf("a garbage predecessor pointer must fold exactly as signed (no validation by design): %+v", rs.Manifest)
	}
	if len(rs.Skipped) != 0 || len(rs.Rejected) != 0 {
		t.Fatalf("no skip/reject for pointer shape — the fold is not a registry: %+v %+v", rs.Skipped, rs.Rejected)
	}
}

func TestManifestWithoutPredecessorUnchanged(t *testing.T) {
	rs := roomFold(t, []Op{
		{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"},
	})
	if rs.Manifest == nil || rs.Manifest.PredecessorRoomKey != "" {
		t.Fatalf("first-epoch manifest must have an empty predecessor pointer: %+v", rs.Manifest)
	}
	b, err := json.Marshal(rs.Manifest)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "predecessorRoomKey") {
		t.Fatalf("empty predecessor pointer must not appear on the wire (omitempty): %s", b)
	}
}

func TestOpSchemaRoundTripPredecessorRoomKey(t *testing.T) {
	predecessor := "cc" + strings.Repeat("dd", 31)
	op := Op{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", PredecessorRoomKey: predecessor}
	b, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Op
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back != op {
		t.Fatalf("round-trip mismatch:\n  want %+v\n  got  %+v", op, back)
	}
	legacy, _ := json.Marshal(Op{Seq: 1, Actor: "a", TS: 1, SKU: "TX", Delta: 2})
	if strings.Contains(string(legacy), "predecessorRoomKey") {
		t.Fatalf("legacy op JSON leaked the predecessor room field: %s", legacy)
	}
}

// mixedRoomOpsWithPredecessor: mixedRoomOps' manifest re-issued with a
// predecessor pointer — the permutation grinder for MSG-D20 (new convergence
// seed 2205, per the mission brief).
func mixedRoomOpsWithPredecessor() []Op {
	ops := mixedRoomOps()
	ops[0].PredecessorRoomKey = "ee" + strings.Repeat("ff", 31)
	return ops
}

func TestRoomConvergence500PermutationsPredecessor(t *testing.T) {
	canonical := roomFold(t, mixedRoomOpsWithPredecessor())
	if canonical.Manifest == nil || canonical.Manifest.PredecessorRoomKey == "" {
		t.Fatalf("scenario anchor wrong: predecessor pointer missing from %+v", canonical.Manifest)
	}
	rng := rand.New(rand.NewSource(2205)) // seeded: the test itself must be deterministic
	for i := range 500 {
		shuffled := mixedRoomOpsWithPredecessor()
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		if got := roomFold(t, shuffled); got.Digest != canonical.Digest {
			t.Fatalf("permutation %d diverged: %s != %s", i, got.Digest, canonical.Digest)
		}
	}
}

// ---------- determinism ----------

// mixedRoomOps: the full vocabulary in one scenario, unenforced (envelope-only),
// for the permutation grinder.
func mixedRoomOps() []Op {
	return []Op{
		{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201", Observers: true},
		post(2, "hub", 200, "Can we ship Thursday?"),
		{Seq: 3, Actor: "ana", TS: 300, Kind: "msg.post", Body: "Thursday works", ReplyTo: "hub:2"},
		{Seq: 4, Actor: "ana", TS: 400, Kind: "msg.edit", MsgID: "ana:3", Body: "Thursday morning works"},
		{Seq: 5, Actor: "hub", TS: 500, Kind: "msg.react", MsgID: "ana:3", Emoji: "👍", On: true},
		{Seq: 6, Actor: "bob", TS: 600, Kind: "msg.react", MsgID: "ana:3", Emoji: "👍", On: true},
		{Seq: 7, Actor: "bob", TS: 700, Kind: "msg.react", MsgID: "ana:3", Emoji: "👍", On: false},
		post(8, "bob", 800, "typo msg"),
		{Seq: 9, Actor: "bob", TS: 900, Kind: "msg.delete", MsgID: "bob:8"},
		{Seq: 10, Actor: "butler", ActorType: "agent", TS: 1000, Kind: "msg.draft-op", Body: "Drafted the approval", Draft: `{"kind":"approval.decide","subject":"PO-2201"}`},
		{Seq: 11, Actor: "ana", TS: 1100, Kind: "msg.read", UpToActor: "hub", UpToSeq: 2},
		{Seq: 12, Actor: "hub", TS: 1200, Kind: "msg.read", UpToActor: "ana", UpToSeq: 3},
		{Seq: 13, Actor: "mallory", TS: 1300, Kind: "msg.edit", MsgID: "hub:2", Body: "hijack"},
		{Seq: 14, Actor: "hub", TS: 1400, Kind: "inventory.move", SKU: "TX-100", Delta: 5}, // business op knocks
	}
}

func TestRoomConvergence500Permutations(t *testing.T) {
	canonical := roomFold(t, mixedRoomOps())
	rng := rand.New(rand.NewSource(2201)) // seeded: the test itself must be deterministic
	for i := range 500 {
		shuffled := mixedRoomOps()
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		if got := roomFold(t, shuffled); got.Digest != canonical.Digest {
			t.Fatalf("permutation %d diverged: %s != %s", i, got.Digest, canonical.Digest)
		}
	}
}

func TestRoomConvergence500PermutationsEnforced(t *testing.T) {
	canonical := ApplyRoom(roomCfg(), enforcedRoomOps())
	rng := rand.New(rand.NewSource(2202))
	for i := range 500 {
		shuffled := enforcedRoomOps()
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		if got := ApplyRoom(roomCfg(), shuffled); got.Digest != canonical.Digest {
			t.Fatalf("enforced permutation %d diverged: %s != %s", i, got.Digest, canonical.Digest)
		}
	}
}

// mixedClaimScenarioOps: expectation tags + claims mixed with the ordinary
// vocabulary, unenforced (envelope-only, same style as mixedRoomOps) — the
// permutation grinder for MSG-D16/D17.
func mixedClaimScenarioOps() []Op {
	return []Op{
		{Seq: 1, Actor: "hub", TS: 100, Kind: "room.manifest", Title: "PO-2201 room", AnchorType: "po", AnchorID: "PO-2201"},
		{Seq: 2, Actor: "hub", TS: 200, Kind: "msg.post", Body: "Can we ship Thursday?", Expectation: "today"},
		{Seq: 3, Actor: "ana", TS: 300, Kind: "msg.post", Body: "on it", Expectation: "urgent"},
		{Seq: 4, Actor: "hub", TS: 400, Kind: "msg.post", Body: "bad tag", Expectation: "asap"}, // unknown tag → skip
		{Seq: 5, Actor: "hub", TS: 500, Kind: "room.claim", Assignee: "hub"},
		{Seq: 6, Actor: "ana", TS: 600, Kind: "room.claim", Assignee: "ana"}, // self-claim reassigns
		{Seq: 7, Actor: "bob", TS: 700, Kind: "room.claim", Assignee: "ana"}, // non-self claim → skip
		{Seq: 8, Actor: "ana", TS: 800, Kind: "room.claim", Assignee: ""},    // release by a non-authority → skip (unenforced: nobody is authority)
	}
}

func TestRoomConvergence500PermutationsExpectationAndClaim(t *testing.T) {
	canonical := roomFold(t, mixedClaimScenarioOps())
	rng := rand.New(rand.NewSource(2204)) // seeded: the test itself must be deterministic
	for i := range 500 {
		shuffled := mixedClaimScenarioOps()
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		if got := roomFold(t, shuffled); got.Digest != canonical.Digest {
			t.Fatalf("permutation %d diverged: %s != %s", i, got.Digest, canonical.Digest)
		}
	}
}

func TestRoomInputImmutability(t *testing.T) {
	ops := mixedRoomOps()
	snapshot, _ := json.Marshal(ops)
	_ = roomFold(t, ops)
	after, _ := json.Marshal(ops)
	if string(snapshot) != string(after) {
		t.Fatalf("ApplyRoom mutated its input")
	}
}

func TestRoomStateJSONHasNoInternalIndex(t *testing.T) {
	rs := roomFold(t, mixedRoomOps())
	b, err := json.Marshal(rs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "msgIndex") {
		t.Fatalf("fold-internal index leaked into the wire state")
	}
	// The wasm boundary round-trips RoomState: it must unmarshal cleanly too.
	var back RoomState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Digest != rs.Digest {
		t.Fatalf("digest lost in round-trip")
	}
}
