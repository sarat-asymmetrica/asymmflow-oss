// room_domain.go — Messenger Wave 1 (Mission M1): the deterministic room fold.
//
// "A conversation and a ledger are the same object — a multi-writer signed log —
// at different levels of formality." (FABLE_CAMPAIGN_MESSENGER.md §0)
//
// A room is its OWN Autobase (campaign distinction #3) — chat volume never
// bloats the business linearizer — so it gets its OWN fold: ApplyRoom, over the
// SAME signed op envelope, guarded by the SAME capability plane (capabilityGate,
// per-room grant table), obeying the SAME four determinism landmines as the
// business reducer (see reducer.go header). One law engine, new vocabulary.
//
// THE taxonomy split (campaign distinction #1, invariant 2):
//   - chat content is CRDT-shaped: the fold ACCEPTS it; ordering resolves races.
//     Ops that fail their own chat rules (edit by non-author, react to unknown
//     msgId, stale read cursor) are SKIPPED with a typed reason — never rejected,
//     never a crash (poison-pill discipline).
//   - membership/capability is invariant-bound: unsigned/ungranted/stale-epoch
//     ops land in Rejected — the SAME vocabulary Missions C/D use. rejected[] is
//     reserved for capability & kernel law; skipped[] is the chat half.
//
// Graduation (campaign distinction #4): msg.draft-op carries a business op as
// an OPAQUE STRING. This fold never parses it, never executes it, and no code
// path forwards it to the business base — graduation is a separate, human-signed
// op appended THERE (M2+ scope). The draft is cargo, not law.
package reducer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
)

// RoomManifest is the room's declaration — first accepted room.manifest op wins
// (in canonical order); every later one is skipped (manifest-uniqueness).
type RoomManifest struct {
	Title      string `json:"title"`
	AnchorType string `json:"anchorType,omitempty"` // business-object anchor: "po", "invoice", …
	AnchorID   string `json:"anchorId,omitempty"`
	Observers  bool   `json:"observersAllowed"`
	Actor      string `json:"actor"` // who declared the room
	TS         int64  `json:"ts"`
	// PredecessorRoomKey (Constitution Art. II amendment, MSG-D20): set only
	// on a room re-issued after a revocation wave, carrying the PREVIOUS
	// epoch's Autobase base key. Stored exactly as signed — the fold records
	// this pointer, it does not validate or dereference it (an offline
	// verifier/peer cannot follow another base; navigation is a host
	// concern). Empty for a first-epoch room.
	PredecessorRoomKey string `json:"predecessorRoomKey,omitempty"`
}

// RoomMessage is one converged message. Deleted messages keep their id and
// author but blank their content (tombstone: append-only log ≠ erased bytes;
// the UX says "deleted", the log stays honest).
type RoomMessage struct {
	MsgID      string `json:"msgId"` // {actor}:{seq} — derived, never random
	Actor      string `json:"actor"`
	ActorType  string `json:"actorType,omitempty"` // "agent" marks Butler drafts for the UX
	DevicePub  string `json:"devicePub,omitempty"` // author device (set when enforcement is on)
	TS         int64  `json:"ts"`
	Body       string `json:"body,omitempty"`
	ReplyTo    string `json:"replyTo,omitempty"`
	Draft      string `json:"draft,omitempty"`      // INERT business-op JSON (msg.draft-op)
	Attachment string `json:"attachment,omitempty"` // INERT blob reference (M3 fills it)
	Edited     bool   `json:"edited,omitempty"`
	EditTS     int64  `json:"editTs,omitempty"` // op data, never a clock
	Deleted    bool   `json:"deleted,omitempty"`
	DeletedBy  string `json:"deletedBy,omitempty"`
	// Expectation is the sender-side tag (Constitution Art. III §3): ""
	// (default, treated as "whenever" by the UI), "whenever", "today", or
	// "urgent". Stored exactly as signed — the fold never normalizes "" to
	// "whenever" in state, only the UI does that presentational step. A
	// msg.edit never changes it (edit only carries body; expectation on an
	// edit op is ignored by the fold — see applyEdit).
	Expectation string `json:"expectation,omitempty"`
}

// RoomClaim is the current owner of an anchored room's work item
// (Constitution Art. VI): last claim in canonical order wins, and an empty
// Assignee is a release. A nil Claim means the room has never seen an
// accepted room.claim op — kept a pointer (the Manifest/Invites pattern) so
// claim-free rooms hash byte-identically; the room goldens regenerate this
// wave regardless (MSG-D16), but the discipline is kept for future waves
// that don't force a re-golden.
type RoomClaim struct {
	Assignee string `json:"assignee"` // "" = released/unassigned
	ByActor  string `json:"byActor"`  // who made this claim/release
	AtSeq    int64  `json:"atSeq"`    // the claiming op's seq (canonical-order provenance)
}

// Skip records a chat op the fold declined with a typed reason — the CRDT half
// of the taxonomy. Never fatal, never silent.
type Skip struct {
	Seq    int64  `json:"seq"`
	Actor  string `json:"actor"`
	Kind   string `json:"kind"`
	MsgID  string `json:"msgId,omitempty"`
	Reason string `json:"reason"`
}

// RoomState is the converged, deterministic result of replaying a room's op
// log. Digest is sha256 over the canonical map-free projection — two peers
// agree iff their digests match (same contract as the business State).
type RoomState struct {
	Manifest *RoomManifest `json:"manifest,omitempty"`
	Claim    *RoomClaim    `json:"claim,omitempty"` // Constitution Art. VI; nil until the first accepted room.claim
	Messages []RoomMessage `json:"messages"`
	// Reactions: msgId → emoji → set of actors currently toggled ON. A toggle
	// OFF removes the actor; empty sets are pruned (only live reactions hash).
	Reactions map[string]map[string]map[string]bool `json:"reactions"`
	// ReadCursors: reader → writer → highest seq read (per-writer monotonic).
	ReadCursors map[string]map[string]int64 `json:"readCursors"`
	Skipped     []Skip                      `json:"skipped"`
	Rejected    []Rejection                 `json:"rejected"` // capability plane only
	Applied     int                         `json:"applied"`
	Digest      string                      `json:"digest"`
	OpsHashed   int                         `json:"opsHashed"`

	// Per-room capability plane (Mission D machinery, unchanged). Non-nil only
	// when enforcement is on (Config.AuthorityPub set) — MESH-D12 pattern.
	CapEpoch int64                 `json:"capEpoch,omitempty"`
	Grants   map[string]GrantState `json:"grants,omitempty"`

	// Invite plane (Mission M2): signed grant OFFERS with fold-enforced expiry,
	// use-count, and revocation. Created lazily on the first invite op, so
	// rooms without invites (incl. the Wave-1 golden) hash byte-identically.
	Invites map[string]InviteState `json:"invites,omitempty"`

	msgIndex map[string]int // msgId → index into Messages (fold-internal, not hashed)
}

// InviteState is one converged invite offer. Uses counts successful
// redemptions; Revoked tombstones the offer (kept, never deleted — an
// append-only plane stays inspectable).
type InviteState struct {
	InvitePub string `json:"invitePub"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expiresAt,omitempty"` // 0 = never (explicit opt-in; creation defaults set 72h)
	MaxUses   int64  `json:"maxUses"`
	Uses      int64  `json:"uses"`
	Revoked   bool   `json:"revoked,omitempty"`
}

// ApplyRoom replays a room op log into the converged RoomState. Pure function
// of (cfg, ops): no I/O, clock, randomness, or map-order dependence. Business
// kinds do not fold here (skipped), exactly as room kinds do not fold in the
// business Apply (rejected there as unknown kinds) — rooms are separate bases.
func ApplyRoom(cfg Config, ops []Op) RoomState {
	sorted := make([]Op, len(ops))
	copy(sorted, ops)
	sort.SliceStable(sorted, func(i, j int) bool { return canonicalLess(sorted[i], sorted[j]) })

	enforce := cfg.AuthorityPub != ""
	rs := RoomState{
		Messages:    make([]RoomMessage, 0),
		Reactions:   make(map[string]map[string]map[string]bool),
		ReadCursors: make(map[string]map[string]int64),
		Skipped:     make([]Skip, 0),
		Rejected:    make([]Rejection, 0),
		msgIndex:    make(map[string]int),
	}
	if enforce {
		rs.Grants = make(map[string]GrantState)
	}

	for _, op := range sorted {
		var capReason string
		handled := false
		if enforce {
			if isInviteKind(op.Kind) {
				// The invite plane is capability law with its own bootstrap
				// rule: invite.redeem is HOW an ungranted device becomes
				// granted, so it bypasses the grant check (never the
				// signature or proof checks) — see applyInvite.
				handled = true
				capReason = applyInvite(&rs, cfg, op)
			} else {
				handled, capReason = capabilityGate(rs.Grants, &rs.CapEpoch, cfg, op)
				if !handled && capReason == "" && op.DevicePub != cfg.AuthorityPub {
					// Role floor (M2): an observer grant replicates and reads;
					// it writes NOTHING — not even read cursors (campaign M2:
					// "observer (read-only)"). The authority and writer-role
					// grants pass through untouched.
					if g, ok := rs.Grants[op.DevicePub]; ok && g.Role == "observer" {
						capReason = "capability: observer grant is read-only (device " + shortKey(op.DevicePub) + " may not write room ops)"
					}
				}
			}
		}
		if capReason != "" {
			// Invariant-bound half: capability refusals share the business
			// fold's Rejected vocabulary — a revoked device reads the same
			// kernel words in a room as in the ledger.
			rs.Rejected = append(rs.Rejected, Rejection{
				Seq: op.Seq, Actor: op.Actor, Kind: op.Kind, Reason: capReason,
			})
			continue
		}
		if handled { // cap.grant / cap.epoch / cap.revoke applied to the room's plane
			rs.Applied++
			continue
		}

		var skipReason string
		switch op.Kind {
		case "room.manifest":
			skipReason = applyManifest(&rs, cfg, enforce, op)
		case "msg.post", "msg.draft-op":
			skipReason = applyPost(&rs, op)
		case "msg.edit":
			skipReason = applyEdit(&rs, enforce, op)
		case "msg.delete":
			skipReason = applyDelete(&rs, cfg, enforce, op)
		case "msg.react":
			skipReason = applyReact(&rs, op)
		case "msg.read":
			skipReason = applyRead(&rs, op)
		case "room.claim":
			skipReason = applyClaim(&rs, cfg, enforce, op)
		case "cap.grant", "cap.epoch", "cap.revoke":
			skipReason = "capability op in a room with no authority configured"
		case "invite.offer", "invite.redeem", "invite.revoke":
			skipReason = "invite op in a room with no authority configured"
		default:
			// Business kinds and unknowns alike: not this base's law.
			skipReason = "not a room op: " + strconv.Quote(op.Kind)
		}
		if skipReason != "" {
			rs.Skipped = append(rs.Skipped, Skip{
				Seq: op.Seq, Actor: op.Actor, Kind: op.Kind, MsgID: op.MsgID, Reason: skipReason,
			})
			continue
		}
		rs.Applied++
	}

	rs.OpsHashed = len(sorted)
	rs.Digest = roomDigest(rs)
	return rs
}

// deriveMsgID is THE msgId rule: {actor}:{seq}. Deterministic, collision-free
// per writer (Hypercore seqs are per-writer monotonic), no uuid, no rand.
func deriveMsgID(op Op) string {
	return op.Actor + ":" + strconv.FormatInt(op.Seq, 10)
}

// applyInvite folds the M2 invite plane: offers, redemptions, revocations.
// Every failure is a capability-law REJECTION (invariant-bound half of the
// taxonomy). Returns "" on success. Determinism notes:
//   - expiry NEVER reads a clock: the redemption op's own TS (event data) is
//     compared against the offer's ExpiresAt (offer data) — one answer on
//     every peer at the op's canonical position (MESH-D13 discipline).
//   - inviteId = {actor}:{seq} of the offer, derived like msgIds.
func applyInvite(rs *RoomState, cfg Config, op Op) string {
	// Signature first — same floor as every enforced op.
	if op.DevicePub == "" || op.Sig == "" {
		return "capability: unsigned op (devicePub+sig required when an authority is configured)"
	}
	if !verifySig(op) {
		return "capability: signature verification failed for device " + shortKey(op.DevicePub)
	}
	isAuthority := op.DevicePub == cfg.AuthorityPub

	switch op.Kind {
	case "invite.offer":
		if !isAuthority {
			return "invite: offers must be signed by the room authority"
		}
		id := deriveMsgID(op)
		if op.InviteID != "" && op.InviteID != id {
			return "invite: inviteId must be {actor}:{seq} (got " + strconv.Quote(op.InviteID) + ", want " + strconv.Quote(id) + ")"
		}
		if len(op.InvitePub) != 64 {
			return "invite: offer requires a 32-byte hex invitePub"
		}
		if op.MaxUses < 1 {
			return "invite: maxUses must be >= 1 (one-time is the default, not zero)"
		}
		if rs.Invites == nil {
			rs.Invites = make(map[string]InviteState)
		}
		if _, dup := rs.Invites[id]; dup {
			return "invite: duplicate inviteId " + strconv.Quote(id)
		}
		role := op.Role
		if role == "" {
			role = "writer"
		}
		if role != "writer" && role != "observer" {
			return "invite: unknown role " + strconv.Quote(op.Role) + " (writer or observer)"
		}
		rs.Invites[id] = InviteState{
			InvitePub: op.InvitePub,
			Role:      role,
			ExpiresAt: op.ExpiresAt,
			MaxUses:   op.MaxUses,
		}
		return ""

	case "invite.redeem":
		if rs.Invites == nil {
			return "invite: unknown invite " + strconv.Quote(op.InviteID)
		}
		inv, ok := rs.Invites[op.InviteID]
		if !ok {
			return "invite: unknown invite " + strconv.Quote(op.InviteID)
		}
		if inv.Revoked {
			return "invite: invite " + strconv.Quote(op.InviteID) + " was revoked"
		}
		if inv.ExpiresAt > 0 && op.TS > inv.ExpiresAt {
			return "invite: invite " + strconv.Quote(op.InviteID) + " expired at " +
				strconv.FormatInt(inv.ExpiresAt, 10) + " (redeem ts " + strconv.FormatInt(op.TS, 10) + ")"
		}
		if inv.Uses >= inv.MaxUses {
			return "invite: invite " + strconv.Quote(op.InviteID) + " exhausted (" +
				strconv.FormatInt(inv.MaxUses, 10) + " use(s))"
		}
		if !verifyInviteProof(inv.InvitePub, op) {
			return "invite: invalid invite proof for device " + shortKey(op.DevicePub)
		}
		// A device holding a CURRENT grant gains nothing from redeeming — the
		// use would be wasted; refuse. A STALE-epoch holder MAY re-redeem a
		// multi-use invite to rejoin after a revocation wave (MSG-D12).
		if g, ok := rs.Grants[op.DevicePub]; ok && g.Epoch == rs.CapEpoch {
			return "invite: device " + shortKey(op.DevicePub) + " already holds a current grant"
		}
		inv.Uses++
		rs.Invites[op.InviteID] = inv
		rs.Grants[op.DevicePub] = GrantState{Role: inv.Role, Epoch: rs.CapEpoch}
		return ""

	case "invite.revoke":
		if !isAuthority {
			return "invite: revocations must be signed by the room authority"
		}
		if rs.Invites == nil {
			return "invite: unknown invite " + strconv.Quote(op.InviteID)
		}
		inv, ok := rs.Invites[op.InviteID]
		if !ok {
			return "invite: unknown invite " + strconv.Quote(op.InviteID)
		}
		if inv.Revoked {
			return "invite: invite " + strconv.Quote(op.InviteID) + " was revoked"
		}
		inv.Revoked = true
		rs.Invites[op.InviteID] = inv
		return ""
	}
	return "invite: unknown kind " + strconv.Quote(op.Kind)
}

// isRoomAuthor reports whether op comes from msg's author. With enforcement on,
// authorship is the DEVICE key (the signed fact); without, the actor string
// (unit-test / legacy mode — the host screens actors there).
func isRoomAuthor(enforce bool, msg RoomMessage, op Op) bool {
	if enforce {
		return op.DevicePub == msg.DevicePub
	}
	return op.Actor == msg.Actor
}

func applyManifest(rs *RoomState, cfg Config, enforce bool, op Op) string {
	if enforce && op.DevicePub != cfg.AuthorityPub {
		return "manifest must be signed by the room authority"
	}
	if op.Title == "" {
		return "manifest requires a title"
	}
	if rs.Manifest != nil {
		return "manifest already declared (first in canonical order wins)"
	}
	rs.Manifest = &RoomManifest{
		Title:              op.Title,
		AnchorType:         op.AnchorType,
		AnchorID:           op.AnchorID,
		Observers:          op.Observers,
		Actor:              op.Actor,
		TS:                 op.TS,
		PredecessorRoomKey: op.PredecessorRoomKey,
	}
	return ""
}

func applyPost(rs *RoomState, op Op) string {
	id := deriveMsgID(op)
	if op.MsgID != "" && op.MsgID != id {
		return "msgId must be {actor}:{seq} (got " + strconv.Quote(op.MsgID) + ", want " + strconv.Quote(id) + ")"
	}
	if _, dup := rs.msgIndex[id]; dup {
		return "duplicate msgId " + strconv.Quote(id)
	}
	if op.Kind == "msg.draft-op" && op.Draft == "" {
		return "draft-op requires a draft payload"
	}
	if op.Kind == "msg.post" && op.Body == "" && op.Attachment == "" {
		return "post requires a body or an attachment"
	}
	if op.Kind == "msg.post" {
		switch op.Expectation {
		case "", "whenever", "today", "urgent":
			// valid — "" is the default, treated as "whenever" by the UI
		default:
			return "unknown expectation tag"
		}
	}
	rs.msgIndex[id] = len(rs.Messages)
	rs.Messages = append(rs.Messages, RoomMessage{
		MsgID:       id,
		Actor:       op.Actor,
		ActorType:   op.ActorType,
		DevicePub:   op.DevicePub,
		TS:          op.TS,
		Body:        op.Body,
		ReplyTo:     op.ReplyTo, // dangling replies allowed: offline peers thread first, converge later
		Draft:       op.Draft,
		Attachment:  op.Attachment,
		Expectation: op.Expectation, // msg.draft-op carries it unvalidated (host convention: empty); msg.post is the only kind that enforces the vocabulary
	})
	return ""
}

func applyEdit(rs *RoomState, enforce bool, op Op) string {
	i, ok := rs.msgIndex[op.MsgID]
	if !ok {
		return "unknown msgId " + strconv.Quote(op.MsgID)
	}
	msg := &rs.Messages[i]
	if msg.Deleted {
		return "message is deleted"
	}
	if !isRoomAuthor(enforce, *msg, op) {
		return "edit by non-author (only the original author may edit)"
	}
	if op.Body == "" {
		return "edit requires a body"
	}
	msg.Body = op.Body
	msg.Edited = true
	msg.EditTS = op.TS // last edit in canonical order wins
	// op.Expectation is deliberately NOT read here: edit only carries body,
	// so the message's original expectation tag survives every edit.
	return ""
}

// applyClaim folds room.claim (Constitution Art. VI, MSG-D17): anchored rooms
// only ("claims are a work concept" in a social room), the room authority may
// assign or release anyone, a non-authority member may claim for themselves
// (op.Assignee == op.Actor) or RELEASE their own current claim (Assignee ""
// while the standing claim is theirs — gate ruling: you can drop work you
// picked up without asking the authority). Releasing someone else's claim, or
// a release when nothing is claimed, is skipped. State-dependence here is
// safe: the standing claim is itself canonical-order deterministic, so every
// peer evaluates the release against the same predecessor.
// Last claim in canonical order wins — reassignment is normal, not an error.
func applyClaim(rs *RoomState, cfg Config, enforce bool, op Op) string {
	if rs.Manifest == nil {
		return "claim requires a manifest"
	}
	if rs.Manifest.AnchorType == "" {
		return "claims are a work concept"
	}
	isAuthority := enforce && op.DevicePub == cfg.AuthorityPub
	if !isAuthority && op.Assignee != op.Actor {
		if op.Assignee != "" || rs.Claim == nil || rs.Claim.Assignee != op.Actor {
			if op.Assignee == "" {
				return "may only release own claim"
			}
			return "may only claim for self"
		}
	}
	rs.Claim = &RoomClaim{Assignee: op.Assignee, ByActor: op.Actor, AtSeq: op.Seq}
	return ""
}

func applyDelete(rs *RoomState, cfg Config, enforce bool, op Op) string {
	i, ok := rs.msgIndex[op.MsgID]
	if !ok {
		return "unknown msgId " + strconv.Quote(op.MsgID)
	}
	msg := &rs.Messages[i]
	if msg.Deleted {
		return "message is deleted"
	}
	isAuthority := enforce && op.DevicePub == cfg.AuthorityPub
	if !isRoomAuthor(enforce, *msg, op) && !isAuthority {
		return "delete by non-author (author or room authority only)"
	}
	// Tombstone: id, author, ts, and thread position survive; content blanks.
	msg.Body = ""
	msg.Draft = ""
	msg.Attachment = ""
	msg.Edited = false
	msg.EditTS = 0
	msg.Deleted = true
	msg.DeletedBy = op.Actor
	// Existing reactions stay (they are separate facts); NEW ones are skipped.
	return ""
}

func applyReact(rs *RoomState, op Op) string {
	if op.Emoji == "" {
		return "react requires an emoji"
	}
	i, ok := rs.msgIndex[op.MsgID]
	if !ok {
		return "unknown msgId " + strconv.Quote(op.MsgID)
	}
	if rs.Messages[i].Deleted {
		return "message is deleted"
	}
	// Toggle per (msgId, emoji, actor): last toggle in canonical order wins.
	if op.On {
		byEmoji := rs.Reactions[op.MsgID]
		if byEmoji == nil {
			byEmoji = make(map[string]map[string]bool)
			rs.Reactions[op.MsgID] = byEmoji
		}
		actors := byEmoji[op.Emoji]
		if actors == nil {
			actors = make(map[string]bool)
			byEmoji[op.Emoji] = actors
		}
		actors[op.Actor] = true
		return ""
	}
	// Toggle off: prune empty sets so only live reactions reach the digest.
	if byEmoji := rs.Reactions[op.MsgID]; byEmoji != nil {
		if actors := byEmoji[op.Emoji]; actors != nil {
			delete(actors, op.Actor)
			if len(actors) == 0 {
				delete(byEmoji, op.Emoji)
			}
		}
		if len(byEmoji) == 0 {
			delete(rs.Reactions, op.MsgID)
		}
	}
	return "" // off-toggle is idempotent: clearing an unset reaction is a no-op, not an error
}

func applyRead(rs *RoomState, op Op) string {
	if op.UpToActor == "" {
		return "read cursor requires upToActor"
	}
	if op.UpToSeq <= 0 {
		return "read cursor requires a positive upToSeq"
	}
	byWriter := rs.ReadCursors[op.Actor]
	if byWriter != nil && op.UpToSeq <= byWriter[op.UpToActor] {
		return "stale read cursor (cursors only advance)"
	}
	if byWriter == nil {
		byWriter = make(map[string]int64)
		rs.ReadCursors[op.Actor] = byWriter
	}
	byWriter[op.UpToActor] = op.UpToSeq
	return ""
}

// roomDigest hashes the canonical map-free projection of a RoomState — every
// map traversed in sorted key order (landmine #1), reaction actor-sets as
// sorted slices. Mirrors the business digest() discipline exactly.
func roomDigest(rs RoomState) string {
	type reactionKV struct {
		MsgID  string   `json:"msgId"`
		Emoji  string   `json:"emoji"`
		Actors []string `json:"actors"` // sorted
	}
	type cursorKV struct {
		Reader string `json:"reader"`
		Writer string `json:"writer"`
		Seq    int64  `json:"seq"`
	}
	type grantKV struct {
		Device string     `json:"device"`
		Grant  GrantState `json:"grant"`
	}
	type inviteKV struct {
		InviteID string      `json:"inviteId"`
		Invite   InviteState `json:"invite"`
	}
	proj := struct {
		Manifest    *RoomManifest `json:"manifest,omitempty"`
		Claim       *RoomClaim    `json:"claim,omitempty"`
		Messages    []RoomMessage `json:"messages"`
		Reactions   []reactionKV  `json:"reactions"`
		ReadCursors []cursorKV    `json:"readCursors"`
		Skipped     []Skip        `json:"skipped"`
		Rejected    []Rejection   `json:"rejected"`
		Applied     int           `json:"applied"`
		CapEpoch    int64         `json:"capEpoch,omitempty"`
		Grants      []grantKV     `json:"grants,omitempty"`
		// Appended ONLY when an invite op ever appeared — invite-free rooms
		// (incl. the Wave-1 golden) hash byte-identically (MSG-D11).
		Invites []inviteKV `json:"invites,omitempty"`
	}{
		Manifest:    rs.Manifest,
		Claim:       rs.Claim,
		Messages:    rs.Messages,
		Reactions:   make([]reactionKV, 0),
		ReadCursors: make([]cursorKV, 0),
		Skipped:     rs.Skipped,
		Rejected:    rs.Rejected,
		Applied:     rs.Applied,
	}
	for _, msgID := range sortedKeys(rs.Reactions) {
		byEmoji := rs.Reactions[msgID]
		for _, emoji := range sortedKeys(byEmoji) {
			actors := make([]string, 0, len(byEmoji[emoji]))
			for a := range byEmoji[emoji] {
				actors = append(actors, a)
			}
			sort.Strings(actors)
			proj.Reactions = append(proj.Reactions, reactionKV{MsgID: msgID, Emoji: emoji, Actors: actors})
		}
	}
	for _, reader := range sortedKeys(rs.ReadCursors) {
		byWriter := rs.ReadCursors[reader]
		for _, writer := range sortedKeys(byWriter) {
			proj.ReadCursors = append(proj.ReadCursors, cursorKV{Reader: reader, Writer: writer, Seq: byWriter[writer]})
		}
	}
	if rs.Grants != nil {
		proj.CapEpoch = rs.CapEpoch
		proj.Grants = make([]grantKV, 0, len(rs.Grants))
		for _, k := range sortedKeys(rs.Grants) {
			proj.Grants = append(proj.Grants, grantKV{Device: k, Grant: rs.Grants[k]})
		}
	}
	if rs.Invites != nil {
		proj.Invites = make([]inviteKV, 0, len(rs.Invites))
		for _, k := range sortedKeys(rs.Invites) {
			proj.Invites = append(proj.Invites, inviteKV{InviteID: k, Invite: rs.Invites[k]})
		}
	}
	b, _ := json.Marshal(proj)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
