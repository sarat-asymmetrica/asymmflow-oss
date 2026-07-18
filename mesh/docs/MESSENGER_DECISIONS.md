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

**MSG-D16 [Mirror] — The v2 signable field list GROWS again; only the room-family goldens re-golden.**
(Owner-ratified 2026-07-18.) `Expectation` (Constitution Art. III §3) and
`Assignee` (Art. VI) are appended at the END of the v2 field list — every
earlier field, in both v1 and v2, keeps its exact byte position, so this is
the MSG-D2 pattern's second growth. `signableV3` carries the same two fields
in the same relative slot (right before the invite fields) since v3 is
defined as "v2 + invite fields". Consequence, exactly as MSG-D2 predicted:
every kind that signs v2 or v3 — `room.manifest`, `msg.*`, `invite.*` — now
produces different signature bytes even when neither new field is set, so
the four room-family goldens regenerate this wave (`room_autobase.json`,
`invite_autobase.json`, `attach_autobase.json`, `mirror_autobase.json`); the
legacy v1 goldens (smoke, wave1, missionc, missiond) do not, because no
business kind ever selects v2/v3. Verified: all four regenerate via
`--update-golden`, then reproduce byte-identically re-run without it; all
four legacy gates pass unmodified. `msg.post` validates the tag vocabulary
(`""`/`"whenever"`/`"today"`/`"urgent"`, unknown = skip "unknown expectation
tag" — chat-domain law, MSG-D4's skipped[] half); every other kind carries
the field unvalidated, its presence in the signable alone prevents unsigned
drift. `msg.edit` never reads `Expectation` — a message's tag is set once,
at post time, and survives every edit untouched.

**MSG-D17 [Mirror] — `room.claim`: anchored-only, authority-or-self, self-release, last-wins.**
(Constitution Art. VI, owner-ratified 2026-07-18; release semantics = gate
ruling same day.) A new room kind, `isRoomKind`-extended identically on both
mirrors. Law, checked in order: (1) no manifest yet → skip "claim requires a
manifest" (there is nothing to own); (2) `Manifest.AnchorType == ""` (a
social room) → skip "claims are a work concept", the Constitution's own
words — ownership is a work concept that does not exist in the human layer;
(3) the room AUTHORITY (mirrors `applyManifest`/`applyDelete`'s
`enforce && op.DevicePub == cfg.AuthorityPub` pattern exactly — unenforced
rooms therefore have no authority at all, so only self-claims/releases ever
land there) may assign or release anyone; every other device may claim only
for itself (`op.Assignee == op.Actor`, else skip "may only claim for self")
— OR release its OWN standing claim (`Assignee == ""` while
`Claim.Assignee == Actor`): the gate ruled that a member who picked up work
may drop it without authority mediation. A release of someone else's claim,
or of nothing, skips "may only release own claim". The state-dependence of
self-release is deterministic because the standing claim is itself
canonical-order-resolved — every peer evaluates the release against the same
predecessor. (4) Last claim in canonical order wins, exactly like
`msg.edit` — no special merge logic. `RoomState.Claim` is a
`*RoomClaim{Assignee, ByActor, AtSeq}`, nil until the first accepted claim
(the `Manifest`/`Invites` pointer-when-materialized pattern). Observer
devices cannot claim: no new code — the existing role-floor check in
`ApplyRoom` runs before the kind switch, proven by a dedicated test rather
than assumed.

**MSG-D18 [Mirror] — The mirror goes blind: Autobase's own `encryptionKey`, the key rides the invite, rotation is NOT free.**
(M4 stage 2, owner-ratified doctrine 2026-07-18: "key rides the invite +
rotate-on-revoke".) Verified in the INSTALLED `autobase@7.11.x` source
before writing a line: the constructor accepts `handlers.encryptionKey` (a
32-byte Buffer) — `autobase/index.js:341-368` (`_runPreOpen`) threads it
into `boot()` (`autobase/lib/boot.js:104-157`), which persists it in the
local/bootstrap core's userData and turns on an `EncryptionView` for the
local writer AND the primary bootstrap core (`index.js:365-369`). Critically,
`ViewStore.getEncryption()` (`lib/store.js:246-252`) applies the SAME base
encryption to every NAMED view core, not just the oplog — so `createMeshNode`
(`mesh-node.mjs`)'s `encryptionKey` option encrypts the linearized view
(`inventory-ops` core) as well as the writer oplogs, end to end. No
Hypercore-level per-core `encryption:{key}` option was needed; Autobase's
own handler is the real, complete mechanism — no workaround required. A
node that omits the option never sets `this.encrypted`; it neither asserts
nor decrypts, it just can't make sense of the ciphertext it replicates.

**Key rides the invite (room2).** `invite-code.mjs` gained a second wire
version: `asymm-room2.` (6 dot-separated z32 parts, the room's
`encryptionKey` appended as the 6th) alongside the untouched `asymm-room1.`
(5 parts). `encodeInviteCode` picks room2 only when an `encryptionKey` is
passed; `decodeInviteCode` accepts both and always returns an
`encryptionKey` field (Buffer for room2, `null` for room1). The existing
invite-spike scenario (unencrypted rooms) never passes the option and stays
on the room1 path byte-for-byte — proven by the untouched gate passing
unmodified. One code is now capability AND content key together; the module
takes no position on how the string travels (MSG-D11's transport-≠-capability
stance, unchanged by encryption riding along) — that stays the human's
choice, documented in the module's own header comment as a bearer-secret
warning.

**The mirror goes honestly blind (mirror-spike.mjs stage 2).** Desk creates
the room with a synthetic fixture key (`Buffer.alloc(32, 0x77)` — canon rule,
MSG-D13's precedent), encodes a real `asymm-room2.` code, and phone joins
using ONLY the decoded fields (`baseKey`, `authorityPub`, `encryptionKey`) —
the ceremony is the proof, not a hand-copied constant. The stage-1 assertion
("the mirror holds plaintext") FLIPS: the gate now asserts the mirror's raw
block does NOT contain the known plaintext substring, AND adds a genuinely
independent keyless probe — a THIRD node, fresh storage, no encryption
option, pulling the identical bytes directly off the mirror over its own
replication stream — which also can't see the plaintext. The phone, holding
the decoded key, reads it fine. Delivery mechanics are unchanged (the mirror
still demonstrably holds the block — MSG-D15's contiguousLength check
stays). Golden note, honestly recorded: the pinned `viewDigest`/`stateDigest`
in `mirror_autobase.json` are computed over DECODED op values
(`node.ops()`/`node.state()`), and encryption is transparent to a peer that
holds the key — so regenerating the golden under encryption produced BYTES
IDENTICAL to the pre-existing stage-1 golden (`git diff` on
`mesh/goldens/` is empty across the whole gate run, mirror included).
Encryption is a storage/transport property, not a value-shape property; the
golden correctly can't see it, and that is the right invariant, not a
missed regeneration. Reproducibility: 3 plain runs post-update, digests
identical every time (GL-2 standard; this scenario is single-writer/
causally-chained, so a view-digest golden is valid per GL-2's letter).

**Rotation — investigated, NOT implemented (stop-and-report, per the
brief).** The installed source has no API to change an Autobase's content
`encryptionKey` mid-life: `boot.js:104-113` reads the key back from the
local core's OWN userData on every subsequent boot and reuses it
unconditionally, regardless of what's passed to the constructor; the only
"rotation" that exists in this version is `blindEncryption`'s re-wrap flow
(`boot.js:128-141`, `index.js:452-461`), which re-encrypts the STORED
encryptionKey blob under a new wrapping/envelope key — it never changes the
actual content-encryption key devices need to read messages. Grep for
`rotat` across `autobase/index.js` and `autobase/lib/*.js` turns up nothing
else; `_rotateLocalWriter` is an unrelated writer-core-swap mechanism. The
consequence, undecorated: today, a revoked device that already holds the
room's `encryptionKey` can still decrypt every message that ever crosses
the mirror in that room, forever — the existing capability-epoch bump
(Mission D) revokes WRITE law but not READ, because read law here is a
symmetric key, not a signature check. Options for the lead's ruling, in
order of how much this mission's constraints (no reducer changes, no new
deps) would tolerate: (1) **Room re-issue as the epoch boundary** — a
revocation wave means minting a NEW Autobase (new bootstrap key, new
`encryptionKey`), carrying a `predecessorRoomKey` pointer in
`room.manifest` so history is discoverable but the live room is a fresh
crypto container; every surviving grant gets re-issued via a fresh
`asymm-room2.` code. This fits the owner's own phrase
("rotate-on-revoke = epoch bump = new key redistributed with re-issued
grants") most literally, costs no reducer change to BUILD (the manifest
field is additive, deferred to whichever wave actually implements it), but
means "the room" is no longer one eternal Autobase — a topology change the
Constitution doesn't currently anticipate. (2) **Application-layer
per-message envelope encryption**, independent of Autobase's own
`encryptionKey` (which would then just be defense-in-depth, left constant)
— message bodies get their own rotatable symmetric key distributed via
re-issued grants, so revocation truly forward-secures new content. This is
the only option that gives real forward secrecy without a new base, but it
is a real new mechanism touching the signable/reducer boundary (new
encrypted-body fields need to exist in the v2/v3 payload) — explicitly the
kind of reducer change this mission was told to stop and flag rather than
build. (3) **Accept the gap as documented, not solved** — ship room2 as
"confidentiality against the mirror and new joiners," explicitly NOT
"forward secrecy against a device that already had the key," and let the
Constitution (Article XI/XII) record that as the stated boundary until a
future wave picks (1) or (2). No implementation attempted for any of these;
this entry exists so the next agent inherits the finding instead of
rediscovering it.

**GATE RULING (technical lead, 2026-07-18): option (1) is doctrine; option
(3) is the honest interim truth.** Rotation = ROOM RE-ISSUE: a revocation
wave mints a new Autobase (new bootstrap key + new `encryptionKey`), the
successor's `room.manifest` carries a `predecessorRoomKey` pointer, and
surviving grants are re-issued via fresh `asymm-room2.` codes — the room
becomes a sequence of crypto-epoch containers, which is the owner's ratified
phrase ("epoch bump = new key redistributed with re-issued grants") taken at
its word. Option (2) (application-layer envelope crypto) is REJECTED for
now: it invents a second encryption mechanism and touches the signable
boundary while (1) satisfies the doctrine with machinery we already trust.
Until re-issue is implemented (future wave), room2's stated boundary is
option (3)'s sentence verbatim: confidentiality against the mirror and new
joiners, NOT forward secrecy against a device that already held the key —
same honesty class as "the fired employee keeps their local copy of
history" (Constitution Art. V: physics, stated plainly). The room-as-
sequence topology touches Constitution Art. II's "one eternal room" framing
— flagged to the owner for an Art. XII amendment; engineering proposes,
only the owner amends.
