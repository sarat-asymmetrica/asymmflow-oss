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

**MSG-D19 [Mirror] — Self-serve evidence export: the envelope IS the
evidence; verification trusts nothing the bundle claims.**
(Constitution Art. V §5, `export-transcript.mjs` / `verify-transcript.mjs`.)
**Export law:** `exportTranscript(node, { exportedBy })` reads the room's
linearized ops off the caller's OWN node (`node.ops()`) and carries them
VERBATIM into a plain JSON-able bundle — no re-signing, no filtering, no
normalization, tombstones export as the tombstone (MSG-D5's blanked-not-
erased structure survives the export unchanged). This is what makes the
export self-serve and admin-free (Art. V §5's "no IT/admin mediation" —
the export function takes a node, not a network call; it reads whatever
copy of the room the caller already holds, same physics as Art. V §3's "no
unsend"). `mesh-node.mjs` grew one read-only passthrough,
`node.authorityPub`, so the bundle can carry the room's authority key
without the exporter tracking it out-of-band — additive, no other spike's
behavior changes (proven by the untouched gates below passing unmodified).
**Verify law:** `verifyTranscript(bundle)` is the trust-nothing counterpart
— offline, no Autobase, no network, no storage. It recomputes, per op, the
versioned signable BY KIND via `capability.mjs`'s own `signable()` (the
same function the reducer's Go mirror agrees with — no duplicated field
list, per the standing standard) and checks the Ed25519 signature against
the op's own `devicePub`; then it re-folds the verbatim ops through the
REAL wasm room reducer (`applyViaWasm(ops, config, 'room')`, `authorityPub`
read from the bundle since an offline verifier has no other source) and
compares the recomputed state digest to the bundle's claimed one. VERIFIED
requires both: all signatures valid AND the digest matches. **The
forged-key case, and why it needs the refold, not just the sig-check:** an
attacker who tampers a body and re-signs with a device key that was never
granted in the room produces an op whose OWN signature checks out
perfectly (the attacker controls both the payload and the key it's checked
against) — sig-verification alone is blind to this. The refold is what
catches it: the kernel's capability plane rejects the ungranted device
(`capability.go`'s "no grant for device" law, unmodified), so the
recomputed digest diverges from what the bundle claims. `transcript-spike.mjs`
asserts this split explicitly (case 4: `sigValid === true` for the forged
op, `digestMatches === false` overall) so the property is proven, not
assumed. Dropped-op tampering is caught the same way (digest mismatch,
case 3); a tampered-but-not-re-signed op is caught by the sig-check alone
(case 1/2) — three different tamper shapes, three different mechanisms,
one bundle format. **Honest limitations, stated plainly (the seatbelt
sentence's sibling):** a VERIFIED verdict proves every op in the bundle was
lawfully signed by the device it claims AND that the exported state is
exactly what those ops fold to — it does NOT prove the transcript is
*complete* beyond the exported range (a verifier only ever sees what's
inside the bundle; a truncated-but-internally-consistent export verifies
cleanly, same as a truncated bank statement balances internally), and it
does NOT prove exporter identity — `exportedBy` is a plain string field,
unsigned, a claim of intent for the reader's benefit, not a cryptographic
attestation (nothing today has the exporting human sign anything; that
would be a real feature, not a documentation gap, if ever wanted). Social
rooms (Art. II, unanchored, no authority in the room at all) export and
verify identically — `authorityPub: null` in the bundle, `config: undefined`
into the refold, same reducer, same law: Art. V's evidence right was
written for the human layer first, and the code doesn't special-case it
away. Scenario is single-writer/causally-chained (GL-2 lets a state+view
golden stand on that shape); gate: `npm run build`, `go test ./mesh/...`
green, `transcriptspike:update-golden` then three plain `transcriptspike`
runs byte-identical, all seven pre-existing spikes (`roomspike`,
`invitespike`, `attachspike`, `mirrorspike`, `smoke`, `wave1`, `missionc`,
`missiond`) green unmodified, `git diff --stat mesh/goldens/` showing only
the new `transcript_autobase.json` addition.

**MSG-D20 [Mirror] — Rotation is re-issue, made real: room identity is a
CHAIN of Autobases, the fold records the link and never follows it.**
(Constitution Art. II amendment 2026-07-18; executes the MSG-D18 GATE
RULING's doctrine — option (1), room re-issue — which had been ruled but
not built.) `Op` gains `PredecessorRoomKey` (`mesh/reducer/reducer.go`),
appended at the END of the v2 field list — MSG-D16's field-list pattern,
THIRD growth — and in the same relative slot in v3 (right before the
invite fields, since v3 is defined as "v2 + invite fields"). `RoomManifest`
carries it exactly as signed; `applyManifest` (`room_domain.go`) copies the
field with **zero additional fold semantics**: no validation beyond what
already exists (an empty or garbage pointer is just the authority's own
signed statement — the fold is not a registry and cannot verify a claim
about a base it has never seen), and critically **the fold never
dereferences it**. Following the chain — opening the base a manifest
points at — is entirely a HOST concern (`reissue-spike.mjs` step 5 does
exactly this, reading the pointer off decoded state rather than ever
hardcoding a predecessor key). This mirrors MSG-D8's opaque-cargo
discipline (`draft`) applied to a different field: the fold's job is to
carry the fact forward untouched, not to act on it.

**The ceremony (`mesh/host/reissue-room.mjs`, `reissueRoom(...)`).** A
revocation wave MINTS A SUCCESSOR: a fresh Autobase (new bootstrap key, new
`encryptionKey`), whose manifest is authority-signed, copies the
predecessor's title/anchorType/anchorId, and stamps the predecessor's OWN
base key as `predecessorRoomKey`; then every surviving device gets a
direct `cap.grant` at epoch 0 of the NEW room. The predecessor is **never
touched** — no tombstone, no close, no write of any kind — it stays a
fully readable historical container for whoever already holds its key
(Art. V's append-only-history-is-owed-to-the-room's-own-past honesty
class); a second "superseded" op on the OLD base would just be an
unrequested mutation of history nobody asked for, and the successor's
manifest pointer is the only linkage the design needs. The function
returns one fresh `asymm-room2.` code per survivor (MSG-D18's key-rides-
the-invite envelope, reused verbatim). **Honest framing, load-bearing:**
the capability grant is ALREADY direct (`cap.grant`, appended before any
code is built) — the invite plane's `inviteSeed`/`inviteId` fields riding
inside each code are NOT a redeemable offer (there is no `invite.offer`
anywhere in a re-issued room; nobody ever calls `invite.redeem` against
them). They are the KEY-TRANSPORT envelope only — "here is the new room
and the new content key" for a device whose capability already lives in
the op log — reusing the one proven bearer-secret string shape rather than
inventing a second one.

**The gate ruling's stated boundary, verified live, both directions
(`reissue-spike.mjs` step 4, two independent probes).** Probe A bypasses
Autobase entirely — a bare Corestore with no encryption option forces an
explicit block-0 request over a real replication stream — proving bytes
genuinely cross the wire (so a downstream failure can't be dismissed as
"never even connected", the same discipline MSG-D18's mirror-spike
third-node probe established) and that the raw block never contains the
known plaintext. Probe B bootstraps a full room-mode `MeshNode` on the
successor's base key with the OLD key (K1) as its `encryptionKey` and
reports the observed failure shape honestly: in this run, Autobase's own
linearizer never materializes a view for a wrong-keyed peer at all (no
thrown error, no timeout — `ops()` resolves fast with an empty view) —
arguably a STRONGER protection than "returns garbage", stated as an
observation, not a guaranteed API contract. The OTHER direction is
equally asserted: a probe bootstrapped on the PREDECESSOR's key with K1 —
reached by literally reading `predecessorRoomKey` off the successor's own
manifest, never hardcoded — reads the old conversation cleanly, because
K1 is honestly still good for the container it was actually issued for
(desk was a genuine epoch-1 member; Constitution Art. V's physics-stated-
plainly class, same sentence class as "the fired employee keeps their
local copy"). **What this does NOT claim:** forward secrecy against a
device that already held the OLD key against the OLD container — that was
never on the table (MSG-D18's stated boundary stands, unchanged) — what
re-issue buys is that the revoked device holds no grant AND no usable key
for anything NEW.

**Rogue holds no grant in the successor — proven the strongest honest
way available.** Any writer can append any signed VALUE past Autobase's
own `apply()` (it only screens the envelope shape); the capability plane
is a FOLD-time law, not a replication-time one (the same distinction
MSG-D10/room-spike.mjs already prove for a rogue riding another peer's
writer core). `reissue-spike.mjs` step 3 therefore SMUGGLES a rogue-signed
op straight into the successor's raw log and watches the refold reject it
("no grant for device") on top of the grant table's plain absence — this
is the strongest available proof precisely because it doesn't rely on
anything ever being physically prevented from being written; it proves
the rejection is a property of the law re-run on every peer.

**Field-list growth, re-golden discipline.** Exactly as MSG-D16 predicted,
every kind that signs v2 or v3 now produces different signature BYTES
even when the new field is unset, so the four room-family view-pinning
goldens regenerate this wave (`room_autobase.json`, `invite_autobase.json`,
`mirror_autobase.json`, `transcript_autobase.json`) — verified: each
regenerates via `--update-golden`, then reproduces byte-identically on a
plain re-run (transcript, the shape GL-2 flags for real per-node
causal-chain proof, x3). `attach_autobase.json` shows a genuine ZERO diff,
explained rather than silently accepted: its golden was scoped (GL-4c) to
state digest + deep state projection only, never a raw view digest, so a
signature-byte-only change is structurally invisible to it — the same
"absent diff, explained" discipline GL-3(b) established for the mirror
golden under encryption. `stateDigest` is byte-identical on every
regenerated golden too, confirming the new field carries zero fold
semantics in every PRE-EXISTING scenario (none of them ever set
`predecessorRoomKey`). The four legacy gates (`smoke`, `wave1`,
`missionc`, `missiond`) pass unmodified — no legacy kind ever selects v2/
v3. New spike: `reissue-spike.mjs` / `npm run reissuespike`, golden
`reissue_autobase.json` — single-authority-writer-per-node causal
chaining throughout (desk only acts on the successor after decoding a
code that provably didn't exist until the ceremony ran — a sequential
handoff, not a concurrent fork), 3 plain reproducibility runs, digests
identical every time.

**MSG-D21 [Mirror] — Social rooms are law, not a UI mode: participant
authority, a fold-enforced read-cursor ban, real invites under encryption,
and blocking whose silence is structural.**
(Constitution Art. II social rooms/DMs, Art. III §6, Art. V §2/§6, Art. XI —
executed 2026-07-18.) The room fold already had everything a social room
needs (Mission D's capability plane, M2's invite plane, M4's encryption);
this wave spends none of it on new mechanism and all of it on WHO holds the
pen and WHAT the manifest points at.

**Participant authority — privacy by topology, not policy.** A social room
is an ordinary ENFORCED room (`createMeshNode({ authorityPub, ... })`,
`mesh/host/social-room.mjs`'s `createSocialRoom`) whose `AuthorityPub` is a
PARTICIPANT'S OWN device key — the creator's — and whose signed manifest
never sets `anchorType`/`anchorId`. There is no second admin identity and no
org key anywhere in the log: `AuthorityPub` is fixed at `createMeshNode`
time and nothing in this vocabulary lets one be added later. A DM is the
two-person degenerate case, opened via the real M2 ceremony
(`openDmInvite`: `inviteOfferOp` + an `asymm-room2.` code carrying the
room's `encryptionKey`, MSG-D18) — the exact machinery reissue-room.mjs
already trusted for org rooms, now founded by a person instead of an org.

**Fold-enforced read-cursor ban, defense-in-depth (Art. III §6).**
`applyRead` (`room_domain.go`) gained two checks ahead of its existing
monotonicity law, in this order: (1) no manifest has folded yet → skip
"read cursor requires a manifest" (deterministic because canonical order
fixes what "before" means — mirrors `applyClaim`'s own manifest-check
style); (2) the manifest is present but unanchored (`AnchorType == ""`) →
skip "read cursors are not emitted in social rooms", the Constitution's own
words, same skip-reason-as-law pattern MSG-D17 established for
`room.claim`. Anchored-room read-cursor behavior is byte-for-byte
UNCHANGED below those two checks. The ban is DOUBLE: the fold refuses to
fold one, AND `social-room.mjs` never builds one in the first place — a
future UI bug that tries anyway still can't make it into the log.

**Flagged, not silently fixed: `TestReadCursorMonotonicity` now fails.**
The mission brief anticipated this exact risk and its instruction was
followed literally: that pre-existing test emits five `msg.read` ops with
NO `room.manifest` op anywhere in its scenario — an untested gap the new
law exposes, not a regression the new law causes. It was left completely
untouched (not edited to add a manifest, not deleted) so the break is
visible rather than papered over; `TestReadCursorAnchoredRoomUnchanged` was
added alongside it, re-proving the EXACT same assertions with a manifest
present, confirming the underlying monotonicity law itself is intact. `go
test ./mesh/...` is therefore RED on this one pre-existing test until a
future commit adds a manifest line to its fixture — a one-line fix,
deliberately not made here per the brief's explicit stop-and-report
instruction (GL-1's pattern: build the letter, flag the consequence, let
the gate rule). Five new tests cover the new law directly:
`TestReadSkippedInSocialRoom`, `TestReadBeforeManifestSkipped`,
`TestReadCursorAnchoredRoomUnchanged`,
`TestReadBeforeManifestCanonicalOrderDeterministic` (a `msg.read` and the
`room.manifest` share the same Seq; actor sort order alone decides which
folds first, in both physical delivery orders, with identical digests
either way), and `TestRoomConvergence500PermutationsSocial` (seed 2206,
500 permutations of a social-room scenario — posts, an expectation tag, a
reaction, an attempted read, an attempted claim — both attempts skip,
every permutation converges).

**Blocking — the participant-authority primitive, and its honest
boundary.** `blockDevice` (`social-room.mjs`) is an epoch bump
(`epochOp`) followed by re-grants (`grantOp`) for every OTHER device
currently holding a grant at the live epoch, read directly off the room's
own state — no survivor list for the caller to hand-maintain. The room's
authority needs no self-grant: `capabilityGate` (`capability.go`) already
treats the authority as implicitly granted, so blocking a DM's only other
member costs exactly one epoch-bump op and zero re-grants. Stated plainly
in the module's own comments, because the function is easy to misread as
more powerful than it is: (1) this only works for the room's OWN
authority — call it with any other device's keys and every op it builds
is simply rejected by the fold on every peer, it does nothing; (2) a
NON-authority participant has NO fold-level block available (membership
law is the same capability plane Mission D built for org rooms, and only
the authority holds that pen) — their "block" is CLIENT-SIDE ONLY: discard
the local copy of the room's base/key, refuse future invites from it. The
fold cannot help a participant block someone in a room they don't govern;
that is the honest shape of "participant authority," not a bug to route
around. A future multi-party social room with rotating/shared authority is
a different design. (3) NO op kind anywhere in the vocabulary signals "you
were blocked" — the blocked device's ops simply start rejecting on every
peer's refold with the ordinary capability wording ("is stale" / "no grant
for device"), indistinguishable from the room having gone quiet. Silence
is structural (Art. V §2), proven in `social-spike.mjs` by scanning every
op kind that ever appears in the full log against a whitelist and
asserting none of it matches `/block/i`.

**True deletion is physics, not a function (Art. V §6).** `social-room.mjs`
deliberately ships no `deleteRoom()`. "Delete the room" is each
participant independently running `rmSync` on their own Corestore
directory and forgetting the room's key — nothing wraps that call. A named
wrapper here would misrepresent what happens: its name would suggest an
operation reaching the ROOM (deleting it for the other party too), when
what genuinely occurs is each owner discarding THEIR OWN copy — forced
forgetting must not exist. `social-spike.mjs` step 7 demonstrates the real
physics directly (both sides `rmSync` their own storage, directories
verified gone) and step 7's final check demonstrates the corollary
honestly: the mirror's copy of the room OUTLIVES both owners' deletions,
unreadable ciphertext forever, because the mirror never held a grant or a
key of its own (MSG-D15) — there is nothing for either owner's deletion to
reach across the wire and act upon, and nothing the mirror itself could do
about it even if asked.

**Art. XI, restated for the standing record:** even encrypted, the mirror
sees WHICH room keys it carries and WHEN — metadata, never content. This
wave's spike (`social-spike.mjs`) exercises exactly that boundary twice,
independently, per GL-5 discipline (arrival and opacity asserted
separately, not inferred from each other): once for ana's segment (a bare,
unkeyed Corestore forces a real block-0 request off the mirror over a
fresh replication stream, and the returned ciphertext is checked against
her known plaintext) and once for bela's segment (same two-part proof
against her reply). The owner's standing veto right over DM-mirroring
(Art. XI, "the owner may veto DM-mirroring entirely at the stage-2 gate")
is unchanged and unexercised by this wave — this entry is the record that
the metadata-visibility boundary was demonstrated honestly, not waived.

**The DM story, hermetic, on `social-spike.mjs` / `npm run socialspike`:**
ana creates the room (participant authority, unanchored, `K` fixture bytes)
and opens a one-time `asymm-room2.` DM invite; pushes to the blind mirror;
goes fully offline (store, autobase, dht all closed). bela — who is NEVER
online at the same time as ana anywhere in this scenario — wakes later,
pulls ana's segment through the mirror alone, redeems the invite FOR REAL
(a rogue's attempt to reuse the same one-time code is refused
"exhausted"), and vents: a threaded reply with an expectation tag, a
voluntary-wave reaction, a deliberately-built `msg.read` attempt (skipped
per the new law) and a deliberately-built `room.claim` attempt (skipped,
MSG-D17, untouched by this mission). The next morning ana's OWN storage
reopens (same device, same key — not a new identity) and converges to
bela's exact state and view digest having never talked to bela directly
either — both peers only ever spoke to the mirror. bela self-serve-exports
a transcript from her own copy (`authorityPub` in the bundle equals ana's
device pub, proving participant authority end-to-end through the evidence
plane too); it verifies. ana then blocks bela; bela's next post rejects on
both peers' refolds with no signal anywhere in the log. Both sides
`rmSync` their storage; the mirror's copy outlives them, unreadable.
Golden (`social_autobase.json`) pins the pre-block converged STATE digest
plus the final `opsHashed`/`applied`/`skipped`/`rejected` counts — NOT a
view digest, per GL-2's letter (the scenario's async segments are the
point being proven, so the golden doesn't lean on a causal-chaining
argument to justify pinning view order); a runtime cross-peer view-digest
equality check (bela vs. ana-reopened) still runs, unpinned. 3 plain
reproducibility runs, identical every time. Gate: `npm run build`, `go
test ./mesh/... -count=1` (one pre-existing failure, flagged above, not
caused by silent editing), `socialspike:update-golden` then 3× plain, all
nine pre-existing spikes (`roomspike`, `invitespike`, `attachspike`,
`mirrorspike`, `transcriptspike`, `reissuespike`, `smoke`, `wave1`,
`missionc`, `missiond`) green UNMODIFIED — `git diff --stat
mesh/goldens/` is empty; `social_autobase.json` is a new, untracked
addition, not a diff to an existing golden.
