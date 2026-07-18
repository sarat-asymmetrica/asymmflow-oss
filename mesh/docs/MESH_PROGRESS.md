# MESH — Progress (honest status per wave)

Campaign: `FABLE_CAMPAIGN_SOVEREIGN_MESH.md`. Branch: `exp/sovereign-mesh`
(worktree `C:\Projects\asymmflow\asymmflow-mesh`). LOCAL-ONLY — never pushed.
Orchestrator = Opus 4.8. "The spike is the gate; the ground wins; the mirror records why."

---

## Wave 0 — SCAFFOLD (2026-07-15) · ✅ done, boundary proven

Goal: stand up the mesh track and **price the two-runtime boundary** (the #1
risk Mission A exists to price) before the Commander is back to run the full gate.

**Built:**
- `mesh/reducer/` — the pure, deterministic inventory apply-reducer (Go, no build
  tags, no I/O/clock/rand). Enforces the floor invariant `qty ≥ 0`; rejects the
  concurrent-offline oversell deterministically. Host unit tests incl. a
  **500-permutation convergence** proof + no-input-mutation + empty-stable.
- `mesh/cmd/reducer/` — the wasip1 packaging (`//go:build wasip1`), a WASI command
  module (stdin JSON → stdout JSON). **Compiles clean to `GOOS=wasip1 GOARCH=wasm`.**
- `mesh/host/apply.mjs` — the JS host driving the wasm via `node:wasi` (real
  file-descriptor stdin/stdout, no pipe deadlock).
- `mesh/host/smoke.mjs` — the determinism smoke over the **real Go→WASM→JS boundary**.
- `mesh/goldens/inventory_basic.json` — pinned converged state.
- `mesh/scripts/build-reducer.mjs` — cross-platform wasm build.
- Holepunch/Pear data stack + Holesail installed (`mesh/package.json`).

**Measured results (Wave-0 smoke, `npm run smoke`):**
- ✅ Boundary: JS drives the Go/WASM reducer; ops marshal in, state marshals out.
- ✅ Invariant: exactly one oversell rejected; no SKU ever < 0.
- ✅ Convergence (reducer half): 3 "peers" fed 3 different arrival orders →
  **byte-identical digest** `aa5fa416…`. Reproducible across separate processes.
- ✅ Golden: converged state matches the pinned golden.

**What Wave 0 deliberately does NOT yet prove (the honest line):**
the 3 peers here are 3 in-process invocations with shuffled input — the
**real Autobase linearization, real Hypercore replication, and real Holesail
transport across ≥2 machines are stubbed.** Wave 0 proves the *reducer + boundary*
halves of Mission A; it does not yet stand up the *replication* half. That is the
first on-site task below.

---

## Wave 1 — MISSION A, the real gate (2026-07-15) · ✅ stages 1+2 GREEN

The stubbed 3-peer smoke is now the real thing. Built (Fable-driven, per the
owner's parallel-tracks call — Opus 4.8 runs frontend INTEG concurrently):

- `mesh/host/mesh-node.mjs` — the mesh peer: Corestore + Autobase whose **view is
  the linearized op log**; `apply()` handles only writer grants + op appends
  (never external state); **state is materialized OUTSIDE apply** by folding the
  whole view through the wasm reducer. Convergence checks: raw **view digest**
  (stronger) + reducer **state digest**.
- `mesh/host/wave1-local.mjs` (`npm run wave1`) — **stage 1**: 3 real Autobase
  writers, 3 on-disk Corestores, real `replicate()` streams, and a GENUINE
  concurrent-offline causal fork (wires cut → dev-a and dev-b write blind →
  wires reconnect → linearizer merges).
- `mesh/host/peer.mjs` — a standalone peer process (host/join roles, JSON-line
  REPL: `add-writer` / `append` / `digest` / `exit`). Transport doctrine held:
  raw Corestore replication over TCP; **Holesail carries the socket** (secure
  `hs://` connector); transport-auth ≠ capability-auth.
- `mesh/host/wave1-holesail.mjs` (`npm run wave1:holesail`) — **stage 2**: two
  separate OS processes replicating through a **real Holesail tunnel over the
  real DHT/UDX stack** (same code path two machines use).

**Measured results (both stages):**
- ✅ Writer grants flow through the linearizer (`addWriter` ops) and replicate.
- ✅ Fork merge: all peers converge to a **byte-identical view**
  (`5962c1f9…`, pinned in `goldens/inventory_autobase.json`; reproducible across
  runs under the test's pinned primary keys).
- ✅ Invariant through the REAL machinery: exactly one oversell rejected, same
  canonical loser (dev-a seq 2) on every peer; no SKU < 0.
- ✅ **State digest == the Wave-0 reducer golden (`aa5fa416…`)** — same ops, same
  reducer, now arriving via real Autobase + real transport. The reducer is
  provably transport-indifferent.
- ✅ Stage 2 over Holesail: grant, appends, convergence, golden — all through the
  tunnel between two processes with independent on-disk stores.

**The honest line (what remains for the full Mission A finale):** stage 2 runs
both processes on ONE machine — the bytes traverse the real DHT/UDX transport,
but not two physical NICs/networks. The ≥2-machine run is the identical
commands on two boxes: machine 1 `npm run wave1:host`, copy the printed `hs://`
url + baseKey, machine 2 `npm run wave1:join -- --url <hs> --base-key <hex>`,
then `add-writer`/`append`/`digest` per `peer.mjs`'s header. Ceremony, not
machinery — the machinery is proven.

**Gate verdict (campaign §6):** peers converge byte-identical ✅ · oversell
deterministically rejected on all peers ✅ · goldens pin the state ✅ — **GREEN**
(with the two-physical-boxes ceremony left for when the Commander has both
machines at hand).

## Wave 2 — MISSION C, kernel-as-reducer (2026-07-16) · ✅ GREEN

The reducer now imports the **REAL kernel packages** —
`pkg/kernel/{money,approval,actor,policy}` — compiled clean to wasip1 (3.7MB
wasm) and proven through the real Autobase machinery:

- **Reducer v2** (`mesh/reducer/reducer.go` + `kernel_domains.go`): typed op
  envelope (`kind` selects the domain; `""` stays Wave-0-compatible), four
  domains folded through kernel law:
  - `inventory.move` — the Wave-0 floor invariant, unchanged.
  - `ar.limit/charge/payment` — **kernel money** integer minor-unit arithmetic;
    credit-limit invariant; currency mismatches are typed kernel errors.
  - `approval.decide` — **kernel approval + actor**: subjects start at
    pending_review; `ValidTransition` is the single truth (approved→rejected
    refused); approve/reject/supersede requires `CanApprove()`;
    needs_input/pending requires `CanPropose()`.
  - `policy.violation/override` — **kernel policy**: only an approver may
    override a standing violation, with a mandatory reason.
- **The AI-authority boundary is DISTRIBUTED LAW:** the agent is stopped at
  THREE kernel layers — `actor.New` (an agent can't even be CONSTRUCTED with
  approve authority), `approval.NewRecord` ("agent actors cannot approve"),
  `policy.Override` (CanApprove gate) — and the Mission C mesh gate proves the
  rejection lands **identically on every peer, even from the agent's own
  writer core**.
- **Determinism audit:** the kernel packages take `now time.Time` as a
  PARAMETER everywhere (no clock reads, no rand, no map-order output —
  audited by grep + the 500-permutation test over the mixed-domain op set).
  Reducer hands them `time.UnixMilli(op.TS).UTC()` — op data, never a clock.
- **Tests:** `missionc_test.go` — kernel-law invariants, agent rejection with
  kernel words, forged-authority agent, propose-level human refused,
  500-permutation convergence, input-immutability. Plus the Wave-0 suite,
  which passes UNCHANGED (envelope is backward compatible).
- **Gates:** `npm run missionc` — 3 peers (one of them the agent's), genuine
  offline fork, byte-identical views + kernel state on all peers, goldened
  (`goldens/missionc_autobase.json`, reproducible).
- **STATE SCHEMA v2:** the state digest now covers stock/ar/approvals/policies —
  Wave-0/1 STATE goldens regenerated (MESH-D9). The Wave-1 VIEW digest was
  untouched (`5962c1f9…`) — the op log didn't change, only the projection.

## Wave 3 — MISSION D, grants-with-epochs (2026-07-16) · ✅ GREEN

The capability layer now sits ABOVE the pipe, enforced INSIDE the reducer
(MESH-D11..D13). `npm run missiond` is the gate.

- **`mesh/reducer/capability.go`** — every op carries `devicePub` + an Ed25519
  signature over sha256 of a version-prefixed NETSTRING payload (byte-identical
  builder mirrored in `mesh/host/capability.mjs`; JSON deliberately avoided).
  Grant plane = three op kinds folded by the same reducer: `cap.grant`
  (authority-signed, role + epoch), `cap.epoch` (strictly-increasing bump =
  revocation wave; grants not re-issued go STALE), `cap.revoke` (targeted).
  Enforcement is opt-in via `Config.AuthorityPub` (mesh-genesis data);
  **zero legacy goldens moved** — smoke/wave1/missionc passed unchanged.
- **`mesh/host/capability.mjs`** — device keypairs (libsodium via
  hypercore-crypto), `signOp`/`grantOp`/`epochOp`/`revokeOp`. Same seeds →
  same keys as Go `ed25519.NewKeyFromSeed` (RFC 8032), cross-proven by the gate.
- **Gate (`missiond-mesh.mjs`)**: 3 peers, ALL in the Autobase writer set,
  genuine offline fork. THE campaign sentence proven mechanically:
  **the pipe still opens — the rogue peer replicates the full converged view —
  while its ops are rejected on every peer** ("no grant for device"). Its
  self-grant dies too (not authority-signed). The laptop peer is granted at
  epoch 0, revoked by the epoch-1 bump mid-fork, re-granted at epoch 1: its
  stale-epoch op is rejected identically everywhere, its re-granted op lands.
  Byte-identical views + state, goldened (`goldens/missiond_autobase.json`,
  reproducible across runs).
- **Tests (`missiond_test.go`)**: grant/epoch lifecycle, unsigned + forged-key +
  tampered-payload ops rejected, epoch replay/rollback refused, targeted revoke,
  kernel-law-still-holds-above-capability (a granted device cannot smuggle an
  agent approval), legacy-mode byte-stability, 500-permutation convergence,
  input immutability.
- **Determinism ruling (MESH-D13):** grant validity is evaluated at the op's
  position in the CANONICAL order — revocation is never retroactive; whether an
  op beats a revocation has exactly one answer on every peer.

### Mission D stage 2 — the ceremony REPL + real-DHT dress rehearsal · ✅ GREEN

`peer.mjs` extended for the two-physical-box ceremony (backward compatible —
no flags = exact Wave-1 behavior, `wave1:holesail` re-verified):

- **Persistent device identities** — each peer keeps an Ed25519 seed in
  `DIR-keys/` (a SIBLING dir: **Corestore 7.x silently DELETES foreign files
  inside its storage dir on init** — found the hard way, seed survived restarts
  only after moving out). Verified stable across restarts.
- **Authority mode** — `host --authority` holds the mesh authority keypair and
  turns enforcement on; joiners pass `--authority-pub` (mesh-genesis config,
  like url + baseKey). New commands: `grant <pub> [epoch]` / `epoch <n>` /
  `revoke <pub>` / `whoami`; `append` AUTO-SIGNS with the peer's device key;
  `append-raw` deliberately doesn't (demo: watch the mesh bounce it).
- **Ceremony seq/ts** — omitted fields are stamped seq = ts = Date.now() at
  CREATION (event data, never read in the fold); wall-clock-millis seqs make
  canonical fold order track live-ceremony real time.
- **Gate (`npm run missiond:holesail`)**: two OS processes over the REAL DHT
  run the exact human ceremony — writer-set admit → pre-grant signed op
  REJECTED both sides ("no grant") → unsigned op REJECTED ("unsigned") →
  grant → op lands → epoch bump → stale-rejected → re-grant → op lands.
  Byte-identical views + state; TX-100 == 8; capEpoch == 1 on both.

### 🏁 THE TWO-PHYSICAL-BOX CEREMONY — ✅ COMPLETE (2026-07-16, hand-run)

Owner's desktop (authority) + the household laptop (device), two real machines
over the public DHT, driven by human hands from the ceremony kit
(`HOST.cmd`/`JOIN.cmd`). Verified from both machines' logs:

- **Convergence:** THREE checkpoints byte-identical on both boxes —
  viewLength 5 (`73631357…`/`b0451550…`), 7 (`6f4cb9e3…`/`9327e88f…`),
  and final 9 (`871c45545…`/`77ca3b0a…`).
- **Mission D live:** `epoch 1` on the desktop stale-rejected the laptop's
  next signed append on BOTH machines with the identical kernel reason;
  pre-revocation stock untouched; laptop kept replicating throughout (pipe
  open, capability dead); `grant <dev> 1` re-issued and its next write landed
  (final stock TX-100 == 23 everywhere).
- Missions A and D finales both closed in the same run. Ceremony lessons fed
  back as hardening: poison-pill addWriter guard, pasted-key sanitizing, the
  writable fix-it error (commits `8f3d620`, `357df27`).

Next ladder rung when convergence cutover approaches: the PH office machine
as the always-on mesh peer (owner ruling 2026-07-15).

## Messenger Wave 1 — MISSION M1, the room fold (2026-07-18) · ✅ GREEN

The Messenger campaign's first build (`FABLE_CAMPAIGN_MESSENGER.md` §6):
"a conversation and a ledger are the same object." Chat is now a domain of
the same law engine — decisions mirrored in `MESSENGER_DECISIONS.md`
(MSG-D1..D10).

- **`mesh/reducer/room_domain.go`** — `ApplyRoom`: the deterministic room
  fold. Vocabulary: `room.manifest` (authority-signed, anchors the room to a
  business object), `msg.post/edit/delete/react/read`, `msg.draft-op` (the
  graduation seam — an agent's business-op draft carried as INERT opaque
  cargo; nothing flows room→business without a human-signed op THERE).
  Taxonomy split is structural: chat-rule failures → `skipped[]` (typed,
  never fatal); capability/kernel law → `rejected[]` (Mission D vocabulary).
  msgId = `{actor}:{seq}` derived, tombstones keep structure + blank content,
  reactions toggle-and-prune, read cursors per-(reader,writer) monotonic.
- **Envelope**: room fields ride the existing signed op; the signable payload
  is now VERSIONED BY KIND (`meshop.v2` for room kinds, exact v1 bytes for
  everything else) — zero legacy signatures/goldens moved (MSG-D2).
  `capabilityGate()` extracted (pure motion) so each room Autobase carries
  its OWN per-room grant plane — membership IS a Mission D grant (MSG-D10).
- **Tests** (`room_test.go`): schema round-trip, manifest-uniqueness (+
  authority-required when enforced), msgId law, edit-authorship/last-wins,
  tombstone, react-toggle, cursor monotonicity, draft inertness, the two
  folds staying strangers both directions, skipped-vs-rejected taxonomy,
  **revocation-mid-conversation** (epoch bump between two messages: first
  folds, second rejected, everywhere), 2×500-permutation convergence
  (plain + enforced), input immutability, no internal index on the wire.
- **Gate (`npm run roomspike`)**: 3 real peers (authority hub + desk +
  phone), genuine offline fork, full conversation (posts, threaded reply,
  edit, tombstone, react on+off, read cursors, the Butler's `msg.draft-op`
  via a granted agent DEVICE riding desk's writer core, a rogue device
  bounced on every peer while its bytes replicate) → byte-identical views
  + room states, goldened (`goldens/room_autobase.json`, reproducible).
  13/13 checks green on first run; cross-runtime v2 signing proven (JS
  signs → Go/WASM verifies).
- **Regression floor**: `go test ./mesh/...` + smoke + wave1 + missionc +
  missiond + BOTH holesail stages — all green, legacy goldens untouched.

**The honest line (what Wave 1 does NOT prove):** invites/blind-pairing (M2),
attachments beyond the reserved envelope field (M3), mirrors/push/mobile
(M4–M6), any UI (owner placement call still open), any graduation EXECUTION
into the business base, delivery states beyond replication, and no two-machine
room ceremony yet (the machinery is the ceremony-proven Mission D stack; the
room-specific dress rehearsal is a rung for later). Refold-per-read is still
O(n) — fine for the spike, priced at M3/M4 per campaign §8.

## Messenger Wave 2 — MISSION M2, invites (2026-07-18) · ✅ GREEN

"Click a code, you're in the PO room." Invites fused with the capability
plane per the campaign — owner rulings ratified same-day (one-time + 72h
defaults, kernel-screen UI direction, M2→M4 autonomous scope). Decisions:
MSG-D11 (fold-enforced offers, v3-by-kind signable, clock-free expiry,
code format) + MSG-D12 (current-epoch grants, stale re-redemption,
observer read-only).

- **Reducer**: `applyInvite` in room_domain.go — offer/redeem/revoke as
  capability LAW (all failures → `rejected[]`); proof-of-possession bound
  to the joining device (`verifyInviteProof`); observer role floor (every
  room op from an observer device rejects; replication untouched). Invite
  plane + digest projection materialize lazily → invite-free rooms (incl.
  the Wave-1 golden) hash byte-identically.
- **Host**: `invite-code.mjs` (pasteable `asymm-room1.…` code via
  hypercore-id-encoding z32: baseKey + authorityPub + invite seed +
  inviteId; transport rendezvous deliberately excluded) + capability.mjs
  FIELDS_V3 mirror, inviteKeys/proof/offer/redeem/revoke builders with the
  owner defaults at creation.
- **Tests** (`invite_test.go`): offer law (authority-only, derived id,
  budget/role/pub validation), redeem law (proof binding incl.
  captured-proof replay, wrong-key, expiry BY OP-DATA TIME, exhaustion,
  current-grant refusal, stale re-redemption across an epoch bump),
  revocation, observer read-only, 500-permutation convergence.
- **Gate (`npm run invitespike`)**: 3 peers, genuine offline fork, FULL
  code round-trip (redeem ops built only from decoded strings): one-time
  writer join lands + speaks; same code exhausted for the second device;
  short-lived code expired by the redeem op's own ts; observer joins via
  multi-use code, both its writes reject on every peer while it holds the
  full replicated view. 11/11 first run; goldened
  (`goldens/invite_autobase.json`), reproducible.
- **Regression floor**: roomspike (W1 golden UNTOUCHED) + smoke + missionc
  + missiond + full go suite — green.

**The honest line (M2 scope):** the spike proves the invite LAW + code over
the existing transport; blind-pairing's asynchronous pairing (host offline at
join time) and the REPL `invite`/`join-code` ceremony commands are the M2
stage-2 rung, not yet built. UI (kernel screen direction ratified) untouched
by design.

## Messenger Wave 3 — MISSION M3, attachments + voice (2026-07-18) · ✅ GREEN

Files and voice notes, content-addressed over Hyperblobs — the cheapest media
win on the recon ladder, culturally the highest-value one in Gulf trade.
ZERO reducer changes: the Wave-1 `attachment` reservation paid off exactly as
designed. Decisions: MSG-D13 (ref-is-the-promise pipeline) + MSG-D14
(reactor measured-not-built: 376ms/10k-op refold, no bottleneck yet).

- **`mesh/host/attachments.mjs`**: one blob core per writer; put → ref
  (blobKey + locator + name + contentType + byteLength + sha256); get →
  P2P stream + END-TO-END sha verification (the ref pins the sender's
  promise, independent of transport integrity).
- **Gate (`npm run attachspike`)**: 3 peers, offline fork; desk attaches a
  48KB document, phone a 96KB audio/webm voice note (synthetic fixtures);
  every peer — including the hub that sent nothing — streams BOTH blobs and
  verifies byte-identity; the room log stays 2.4KB (bytes never inlined);
  flipped-byte and forged-ref tampering both die loudly. 9/9 + goldened
  (`goldens/attach_autobase.json`), reproducible. Refold bench rode along
  (1k/5k/10k = 129/196/376ms via wasm).
- **Regression floor**: roomspike + invitespike + smoke + wave1 + missionc
  + missiond + full go suite — all green, all prior goldens untouched.

**The honest line (M3 scope):** no progress-rollup events (put/get are
awaited wholes — streaming progress is a UI-wave concern), no thumbnails/MIME
sniffing, no blob garbage collection (append-only stores grow; pruning policy
is an M4+/ops question), voice CAPTURE untested (MediaRecorder is UI-side —
the mesh half is proven with fixture bytes).

## Messenger Wave 4 — MISSION M4 stage 1, the blind mirror (2026-07-18) · ✅ GREEN

The receptionist machine: offline delivery through blind-peer, proven on a
hermetic local testnet DHT. Decision: MSG-D15 (mirror = delivery infra, not
a member; plaintext honesty; blindness-by-encryption = stage 2, Commander
doctrine first).

- **Gate (`npm run mirrorspike`)**: desk writes a capability-enforced room
  (manifest + grant + threaded messages + cursor), pushes it to a blind-peer
  mirror via blind-peering (shared wakeup instance across Autobase +
  BlindPeering — the wiring lesson of the mission), then closes STORE, BASE,
  AND DHT. Phone wakes after, knows only the room key + mirror key, and
  converges BYTE-IDENTICALLY (view + state digests) through the mirror —
  grant table, epoch, reply threading all intact. The gate also asserts the
  honest limit: the stage-1 mirror holds PLAINTEXT blocks. 7/7 first run;
  goldened (`goldens/mirror_autobase.json`), reproducible.
- **Deps**: blind-peer 3.12 + blind-peering + protomux-wakeup + hyperdht
  (testnet helper). All Apache-2.0/MIT-family, license posture clean.
- **Regression floor**: ALL gates green (mirror + attach + invite + room +
  smoke + wave1 + missionc + missiond + go suite), all goldens stable.

**The honest line (M4 remaining):** encryption-at-rest on the mirror
(Autobase encryptionKey + room-key distribution doctrine — STOP-AND-ASK),
mailbox-vs-pure-mirror for one-shot message drops (blind-peer-encodings
Mailbox — read before choosing), ops packaging for the office machine
(service wrapper, disk budgets, blind-peer-cli vs embedded), gc/pruning
policy for blob cores, and the public-DHT dress rehearsal. Push (M6) stays
its own workstream; POC-grade upstream untouched.

## Wave 4+ — Mission E (next)

- **E** — per-device ZATCA Hypercore chains (`ICV = core.length`).
- The `//go:wasmexport` incremental reactor (per MESH-D6) when marshalling
  volume warrants it.

## Messenger wave: "The Blind Mirror Earns Its Name" (2026-07-18)

The first orchestrated wave on this track: Fable 5 technical lead + Sonnet 5
coder agents, each mission gated by the lead before the next launched
(`AGENT_GATE_LEDGER.md` GL-1..GL-4 — the quality findings now bind every
future agent). Preceded by the DESIGN CONSTITUTION
(`MESSENGER_DESIGN_CONSTITUTION.md`, 12 articles, four-agent research pass,
owner-ratified) — the wave enacted its Articles III, VI, XI, and V §5.

- **Mission 1 — constitution vocabulary** (commit a1375d0): expectation tags
  on `msg.post` (`whenever/today/urgent`, signed, unknown = typed skip) +
  `room.claim` (anchored-only, authority-or-self, SELF-RELEASE per gate
  ruling, last-canonical-wins). v2 signable grew (MSG-D16, owner-ratified
  re-golden of the four room-family goldens; legacy v1 byte-frozen). 14 new
  Go tests incl. 500-perm seed 2204. Gate also fixed a latent M3-era flake:
  attach-spike pinned a view digest over a genuinely concurrent fork (GL-2 —
  state-pinned now, view asserted converged).
- **Mission 2 — the mirror goes blind** (16dce36): Autobase's own
  `encryptionKey` threaded through `createMeshNode` (source-verified: oplog
  AND named view cores encrypt); `asymm-room2.` invite codes carry the
  content key (one paste = capability + key; room1 path byte-identical);
  mirror-spike honesty check FLIPPED — mirror holds ciphertext, an
  independent keyless third node pulling the same bytes over its own
  replication stream reads nothing, the keyed phone reads fine. Rotation
  investigated NOT built: no content-key rotation API exists (MSG-D18);
  gate ruling = room re-issue as the crypto-epoch boundary; Constitution
  Art. II amended (owner-approved, 7b2e540): room identity = a chain of
  crypto-epoch containers linked by `predecessorRoomKey`.
- **Mission 3 — the evidence export** (this commit): Article V §5 real —
  `exportTranscript` (verbatim signed ops off the target's OWN node, no
  admin mediation, tombstones stay tombstones) + `verifyTranscript`
  (offline, trust-nothing: every signature recomputed via the shared
  versioned signable, whole transcript refolded through the real wasm
  reducer). transcript-spike: 25 checks — tamper (sig fails), drop (digest
  catches), FORGE (self-consistent attacker signature passes sig-check but
  the capability plane's refold diverges — the kernel is the lie detector),
  social-room export (no authority, still verifies), bundle immutability,
  golden 3× reproducible.
- **Regression floor**: every gate green after every mission (9 npm gates +
  go suite), legacy goldens byte-identical throughout.

**The honest line:** rotation/room re-issue is ruled doctrine but NOT
implemented (future wave: successor manifest + predecessorRoomKey + re-issue
ceremony); room2's interim boundary = confidentiality against the mirror and
new joiners, NOT forward secrecy against a device that already held the key;
`exportedBy` on a transcript is a claim, not an attestation; expectation
tags/claims/evidence export have no UI yet (kernel-screen wave pending,
owner placement ruling stands); DMs remain P2P-only until the social layer
ships on this encrypted road (Constitution Art. XI).

---

## Residue / notes for the next session

- `mesh/dist/reducer.wasm` is git-ignored (build output); `npm run build` regenerates
  it. `npm run smoke` builds then runs.
- The reducer digest is a sha256 over a canonical, map-free projection — safe to use
  as the cross-peer convergence check.
- Determinism landmines are documented inline in `mesh/reducer/reducer.go`; keep that
  discipline when the reducer grows to import the real kernel packages (Mission C).
