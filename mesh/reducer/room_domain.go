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

	msgIndex map[string]int // msgId → index into Messages (fold-internal, not hashed)
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
			handled, capReason = capabilityGate(rs.Grants, &rs.CapEpoch, cfg, op)
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
		case "cap.grant", "cap.epoch", "cap.revoke":
			skipReason = "capability op in a room with no authority configured"
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
		Title:      op.Title,
		AnchorType: op.AnchorType,
		AnchorID:   op.AnchorID,
		Observers:  op.Observers,
		Actor:      op.Actor,
		TS:         op.TS,
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
	rs.msgIndex[id] = len(rs.Messages)
	rs.Messages = append(rs.Messages, RoomMessage{
		MsgID:      id,
		Actor:      op.Actor,
		ActorType:  op.ActorType,
		DevicePub:  op.DevicePub,
		TS:         op.TS,
		Body:       op.Body,
		ReplyTo:    op.ReplyTo, // dangling replies allowed: offline peers thread first, converge later
		Draft:      op.Draft,
		Attachment: op.Attachment,
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
	proj := struct {
		Manifest    *RoomManifest `json:"manifest,omitempty"`
		Messages    []RoomMessage `json:"messages"`
		Reactions   []reactionKV  `json:"reactions"`
		ReadCursors []cursorKV    `json:"readCursors"`
		Skipped     []Skip        `json:"skipped"`
		Rejected    []Rejection   `json:"rejected"`
		Applied     int           `json:"applied"`
		CapEpoch    int64         `json:"capEpoch,omitempty"`
		Grants      []grantKV     `json:"grants,omitempty"`
	}{
		Manifest:    rs.Manifest,
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
	b, _ := json.Marshal(proj)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
