# MESSENGER — Decisions (the mirror)

Campaign: `FABLE_CAMPAIGN_MESSENGER.md` (a mesh track; sits on Missions A/C/D).
Style per `MESH_DECISIONS.md`: numbered, honest, written when decided.

---

**MSG-D1 [Mirror] — Rooms are a separate fold, not a business domain.**
`ApplyRoom(cfg, ops) → RoomState` in `mesh/reducer/room_domain.go`, alongside —
never inside — the business `Apply`. A room is its own Autobase (campaign
distinction #3), so it gets its own state, digest, and gate. The two folds stay
strangers by test: room kinds are REJECTED by the business fold as unknown
kinds (its existing default arm — zero legacy behavior moved), business kinds
are SKIPPED by the room fold ("not a room op"). Chat volume can never bloat the
business linearizer, and a smuggled `msg.post` can never perturb stock.

**MSG-D2 [Mirror] — The signable payload is versioned BY KIND.**
Room kinds (`room.manifest`, `msg.*`) sign `"meshop.v2"` = the v1 netstring
field list + the room fields appended; every legacy kind keeps its exact
`"meshop.v1"` bytes. Chosen over one grown v1 payload because re-signing would
have invalidated every Mission A–D signature, test vector, and golden — the
regression floor stayed byte-identical instead. Consequence, accepted and
documented: room fields present on a NON-room op are unsigned — and ignored,
since no legacy handler reads them and the room fold only folds room kinds.
Mirrors: `signableV2()` (Go) ↔ `FIELDS_V2` (`capability.mjs`); `isRoomKind` is
character-for-character identical on both sides (incl. the `len > 4` guard).

**MSG-D3 [Mirror] — msgId is derived, never minted: `{actor}:{seq}`.**
The fold computes it; a provided msgId that disagrees is skipped ("msgId must
be {actor}:{seq}"), a duplicate is skipped. No uuid, no rand (landmine #2);
per-writer Hypercore seqs make it collision-free without coordination.

**MSG-D4 [Mirror] — The taxonomy split is structural: `skipped[]` vs `rejected[]`.**
Chat ops that fail their own rules (non-author edit, react to unknown/deleted
msgId, stale cursor, duplicate manifest…) are SKIPPED with a typed reason — the
CRDT half: chat has no oversell, so nothing is "refused", only declined with
honesty. `rejected[]` is reserved for the invariant-bound half: capability law
(unsigned / no grant / stale epoch), same vocabulary and kernel words as the
business fold. The room spike asserts the split (3 capability rejections, 1
chat skip in the enforced unit scenario; 1/0 in the mesh gate).

**MSG-D5 [Mirror] — Tombstones keep structure, blank content.**
`msg.delete` keeps msgId, author, ts, and thread position (replyTo chains stay
navigable); blanks body/draft/attachment; stamps `deletedBy`. Append-only log ≠
erased bytes — the UX says "deleted", the log stays honest. Pre-delete
reactions SURVIVE (they are separate facts about a message that existed);
post-delete edits/reacts/deletes are skipped ("message is deleted").

**MSG-D6 [Mirror] — Authorship is the DEVICE when enforcement is on.**
Edit requires the original author; delete requires author or room authority;
the manifest requires the authority itself. With `Config.AuthorityPub` set,
"author" means the signing device key (the proven fact); in unenforced/unit
mode it falls back to the actor string. An actor-string claim can never
impersonate across devices in a real (enforced) room.

**MSG-D7 [Mirror] — Read cursors are per-(reader, writer) and strictly advance.**
`msg.read {upToActor, upToSeq}`: reader R has read writer W's log up to seq S.
Equal-or-lower cursors skip as stale — monotonic, so a delayed old cursor can
never rewind "seen" state on any peer. (Flattened from the spec's nested
`upTo:{actor,seq}` for netstring determinism — same semantics, fewer bytes.)

**MSG-D8 [Mirror] — Draft and attachment are OPAQUE STRINGS, never parsed.**
`msg.draft-op` carries its business op as a JSON string the fold treats as
cargo: byte-identical across the Go/JS boundary (no re-marshalling ambiguity),
inert by construction (no code path executes or forwards it — graduation is a
separate human-signed op on the business base, M2+). `attachment` is reserved
in the v2 envelope NOW (same opacity rule) so M3's blob references won't force
a signable v3 bump.

**MSG-D9 [Mirror] — Reaction off-toggles prune; only live reactions hash.**
Toggle semantics per (msgId, emoji, actor), last in canonical order wins; a
final OFF removes the entry and empty sets are pruned, so two rooms that end in
the same visible reactions digest identically regardless of toggle history.
Off-toggling an unset reaction is an idempotent no-op, not an error.

**MSG-D10 [Mirror] — The room capability plane is Mission D's, verbatim.**
`capabilityGate()` extracted from `checkCapability` by pure code motion (the
Mission D unit tests + goldens prove no semantic drift) and pointed at the
RoomState's own grant table: each room Autobase carries its own per-room
grants/epochs — "membership is a capability grant", grantable and revocable
per-room, no second permission system (campaign invariant 1). Proven live:
revocation-mid-conversation (first message folds, the post-bump one is
rejected identically everywhere) and the rogue device bounced on all peers
while its bytes replicate fine.
