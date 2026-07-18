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

**MSG-D11 [Mirror] — Invites are FOLD-ENFORCED grant offers; signable v3 by kind.**
(M2, owner invite law ratified 2026-07-18.) `invite.offer` (authority-signed:
invitePub, role writer|observer, expiresAt, maxUses≥1) / `invite.redeem`
(joining-device-signed, carrying an Ed25519 proof by the INVITE key over the
joining devicePub — a captured proof admits nobody else) / `invite.revoke`
(authority tombstone). Everything upstream blind-pairing leaves advisory —
expiry, use-count, revocation — is law here. EXPIRY NEVER READS A CLOCK: the
redeem op's own TS is compared against the offer's expiresAt at the op's
canonical position — one answer on every peer (MESH-D13 discipline). Defaults
live at CREATION (host `inviteOfferOp`: one-time, 72h TTL); the fold enforces
whatever the offer says (expiresAt 0 = never, explicit opt-in). Invite fields
ride a `meshop.v3` signable selected ONLY by `invite.*` kinds — v1/v2 bytes
and all prior goldens untouched (the MSG-D2 pattern, third generation). The
invite plane materializes lazily: invite-free rooms hash byte-identically.
The shareable code (`asymm-room1.…`, invite-code.mjs) carries baseKey +
authorityPub + invite SEED + inviteId via hypercore-id-encoding z32; the
transport rendezvous is deliberately NOT in the code (transport ≠ capability).

**MSG-D12 [Mirror] — Redemption grants at the CURRENT epoch; stale holders may re-redeem.**
A device holding a CURRENT-epoch grant is refused ("already holds a current
grant" — a use is never wasted); a STALE-epoch device may re-redeem a
still-open multi-use invite to rejoin after a revocation wave, consuming a
use and granting at the new epoch. Observer-role grants are READ-ONLY in
full: every room op from an observer device rejects (not even read cursors —
campaign M2's "observer (read-only)" taken literally); replication is
untouched (pipe open, capability scoped).

**MSG-D13 [Mirror] — Attachments: the ref is the promise; the bytes never enter the log.**
(M3.) The reducer was never touched — `attachment` stays the opaque string
reserved at Wave 1 (MSG-D8). The pipeline is host-side (`attachments.mjs`):
each writer owns ONE blob core (`room-blobs`) in its corestore; a message
carries a REF `{blobKey, id(locator), name, contentType, byteLength, sha256}`;
receivers stream P2P (Hypercore Merkle proofs guard transport) and verify the
ref's sha256 END-TO-END before the bytes reach anyone — the ref pins what the
SENDER promised, so a forged ref or flipped byte dies loudly regardless of
transport honesty. A voice note is just an attachment with `audio/webm` —
zero live-media stack; capture is a UI concern. Fixtures are synthetic
patterns (canon rule). Gate proof includes the leanness invariant: 2.4KB of
room log carrying 147KB of blobs. (Lesson paid: never pass a key array as a
JSON.stringify replacer — it filters recursively and gutted the nested blob
locator on the first run.)

**MSG-D14 [Mirror] — The incremental reactor stays UNBUILT: measured, not assumed.**
(Owner pre-authorization 2026-07-18: build only on measured need, gated by
byte-equivalence.) Bench through the real wasm boundary (attach-spike):
refold of 1k/5k/10k-op rooms = ~129/196/376ms. State materialization runs
per convergence-check, not per keystroke — no current caller is bottlenecked.
Verdict: numbers on the record, reactor deferred until a real surface (M4
mirror requiring frequent refolds, or the kernel-screen UI) crosses ~1s.

**MSG-D15 [Mirror] — The mirror is delivery infrastructure, not a member; blindness is stage 2.**
(M4 stage 1.) blind-peer (Apache-2.0) is the receptionist machine: clients
push the room's Autobase via blind-peering (`addAutobase`, shared
protomux-wakeup instance threaded through BOTH the Autobase handlers and the
BlindPeering client — one wakeup, two consumers), the mirror holds + serves
cores, and a later-waking peer converges byte-identically without ever
sharing an online moment with the sender. The mirror holds NO device key, NO
grant, NO authority — it cannot write a lawful op; it is transport that
happens to have a disk. STAGE-1 HONESTY, asserted in the gate itself: cores
are PLAINTEXT, so this mirror COULD read what it holds. True blindness =
Autobase encryptionKey, deliberately deferred — room-key distribution is a
Commander doctrine conversation (campaign §5), not an engineering default.
Gate runs on a hermetic LOCAL testnet DHT (hyperdht/testnet) — no public
network in CI-shaped runs; the public-DHT ceremony is the Mission D pattern
when the office machine takes the role.

**MSG-D10 [Mirror] — The room capability plane is Mission D's, verbatim.**
`capabilityGate()` extracted from `checkCapability` by pure code motion (the
Mission D unit tests + goldens prove no semantic drift) and pointed at the
RoomState's own grant table: each room Autobase carries its own per-room
grants/epochs — "membership is a capability grant", grantable and revocable
per-room, no second permission system (campaign invariant 1). Proven live:
revocation-mid-conversation (first message folds, the post-bump one is
rejected identically everywhere) and the rogue device bounced on all peers
while its bytes replicate fine.
