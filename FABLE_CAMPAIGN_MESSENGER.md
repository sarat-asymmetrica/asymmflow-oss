# FABLE CAMPAIGN — MESSENGER: "What Was Said Joins What Was Done"

Written 2026-07-16 by the Fable 5 senior architect, from the Commander's
(Sarat's) ecosystem strategy session, for the instance that will run this
campaign. Grounded in `C:\Projects\asymmflow\MESSENGER_RECON.md` (the
9-agent ecosystem recon, orchestrator-verified) and the integration
brainstorm (`asymmflow-integration-discussion-summary.md`). This campaign
sits ON TOP of the Sovereign Mesh campaign — Missions A/C/D are its
foundation and are DONE (2-physical-box ceremony completed 2026-07-16).

Commander's call (2026-07-16): the messenger layer outranks the ephemeral
process contract in build order, because it IS the process contract's UX —
a multi-party order core is a room where some messages are structured
milestones. Build the room first; the contract inherits it.

**This campaign is horizon work. It never blocks, delays, or shares a
branch with the PH convergence/deployment track.** The Commander is
available. Ask when a decision is his; do not ask when this document
already answers it.

---

## 0. The thesis this campaign proves

Every business runs two parallel records: **what was said** (WhatsApp,
email — unstructured, on rented infrastructure, legally murky) and **what
was done** (the ERP — structured, auditable). The entire pain of business
coordination lives in the gap between them.

> **A conversation and a ledger are the same object — a multi-writer
> signed log — at different levels of formality. Fold them with one law
> engine and the gap disappears.**

Concretely: a room is an Autobase; a message is a signed op; membership is
a capability grant (Mission D machinery, unchanged); a file is a blob
reference; and a message can **graduate** into a kernel-checked business op
— "approved, ship it" becomes `approval.decide`, checked by the same
`CanApprove()` that guards it everywhere else. The AI (Butler) sits in any
room as a participant who drafts and never decides. The chat log is
evidence-grade: signed, replicated, independently held by every party — and
it never touches Meta's or Salesforce's servers.

## 1. Why this is worth it (the payoff)

- **It completes the sovereignty thesis.** "Own your data" is incomplete
  while the conversations ABOUT the data live on rented infrastructure.
- **Threads anchor to business objects.** A conversation *on* PO-2201, not
  a chat app beside an ERP. Context is structural (same mesh, same keys),
  not a hyperlink.
- **The dispute story.** In GCC trade, "as you said on WhatsApp" is a
  screenshot war. Here both parties hold the same signed log.
- **The recon says the ground is firm.** ~60-70% of a Keet-grade messenger
  exists as maintained, permissively-licensed open source (verdict + license
  table in MESSENGER_RECON.md §1/§3, license posture CLEAN after
  verification). What's missing — the chat data model — is exactly the
  kind of artifact this codebase is best at: a typed op vocabulary + a
  deterministic fold.
- **The wedge is viral.** External parties (suppliers, forwarders,
  accountants) join rooms via invite-code microtools without running
  AsymmFlow. They experience it, then ask for it. (Recon: Bare mobile is
  stable; blind-pairing is the module Keet itself uses.)

## 2. The load-bearing distinctions you must internalize FIRST

1. **Chat is CRDT-shaped; membership is invariant-bound.** Per the
   conflict-shape taxonomy: message posts/edits/reactions/read-cursors are
   commutative — the fold ALWAYS accepts them (ordering resolves races;
   nothing is "oversold"). Membership/roles/invite-consumption are
   invariant-bound — they ride the EXISTING capability plane (cap.grant /
   cap.epoch / cap.revoke), which already enforces what blind-pairing
   deliberately does not (expiry, revocation, roles). Do not invent a
   second permission system. There is one: Mission D's.
2. **One law engine, new vocabulary.** The room fold is the SAME wasip1
   reducer discipline (canonical order, no clock/rand/map-iteration,
   goldens) with a `msg.*` op family. Do NOT fork a second reducer style.
   Whether it compiles into the same .wasm or a sibling module is an
   implementation detail; the determinism law is not.
3. **Rooms are separate Autobases, not the business base.** One Autobase
   per room (Keet's shape). Chat volume must never bloat the business
   log's linearizer, and room membership must be grantable per-room.
   The room's `manifest` op (first entry) declares its anchor:
   `{kind:'room.manifest', anchor:{type:'po', id:'PO-2201'}, title, rules}`.
4. **Graduation is a border crossing, not a bridge.** A message may CARRY
   a draft business op (Butler's or a human's). Confirmation is a separate,
   human-signed op appended to the BUSINESS base, where the kernel checks
   actor authority. Nothing flows from room-log to business-log
   automatically. The AI-authority boundary applies on both sides of the
   border (agents: draft-only in rooms, rejected as deciders in the
   business fold — already proven in Mission C/D gates).
5. **Delivery honesty.** An offline-first messenger must never fake
   immediacy. Checkmark states are typed and honest: `local` → `replicated
   (n peers)` → `mirrored` → `read (per-writer cursor)`. "Delivered to
   mirror" and "delivered to device" are different facts; show them as
   different facts. (Kernel Mechanism 2, applied to chat.)

## 3. Missions (value/effort order, per the recon ladder)

**M0 — Foundation reuse. DONE.** Autobase rooms machinery, Ed25519
grant-with-epochs capability plane, wasip1 reducer discipline, Holesail
transport, ceremony-proven 2-box replication. Every mission below composes
on this; none re-solves it.

**M1 — The chat data model (THE build; Wave 1 below).** The `msg.*` op
vocabulary + deterministic room fold: post, edit, delete, react, read
cursors, reply threading, business-object anchoring. This is the piece
nobody has open-sourced (recon §4 row 1). Everything downstream consumes it.

**M2 — Membership + invites.** blind-pairing/blind-pairing-core for the
invite handshake + `hypercore-id-encoding` for shareable codes, FUSED with
our capability plane for enforcement: invite = a signed grant offer;
acceptance = cap.grant into the room's Autobase; expiry/one-time-use =
tombstone ops checked by the fold (upstream defines `expires` but does not
enforce it — we do). Roles: `member` (msg.*), `observer` (read-only, no
writer core), `authority` (grants). Deliverable includes the
"click-a-code, you're in the PO room" flow in the peer REPL first, UI second.

**M3 — Files + voice notes.** Hyperdrive/Hyperblobs attachment pipeline:
`msg.post` carries `{attachment:{blobKey, byteLength, contentType, name,
sha256}}`; receiver streams P2P, integrity by Merkle proof. Voice note =
MediaRecorder capture → same pipeline (`audio/webm`), zero live-media
stack. Progress rollup + MIME/thumbnail convention are ours to write
(recon: raw events exist, no helper). Cheapest media win on the ladder —
and culturally the highest-value one in Gulf trade.

**M4 — Offline delivery / the always-on peer.** blind-peer/blind-peering
as the receptionist-machine role: an encrypted mirror that CANNOT read
what it holds. Ops packaging is ours (no upstream Docker/systemd). Read
blind-peer-encodings' Mailbox source before choosing mailbox vs pure-mirror
(docs are thin — recon §6). Overlaps infrastructure the business mesh
needs regardless of the messenger.

**M5 — Mobile read-only companion.** Bare + bare-kit + react-native/expo
templates (stable, recon §7.4): a 2-screen companion (rooms list, room
view) that replicates and reads. Ships BEFORE push — a foreground/pull
companion has standalone value. App-store treatment of Bare binaries is an
unknown: pre-flight check before any submission (stop-and-ask).

**M6 — Push / background wake. Own workstream; never on a critical path.**
blind-push/blind-push-gateway are self-labeled POC. Self-hosted FCM/APNS
gateway = our ops + our credentials. Prototype iOS background-wake before
promising any UX. Budget separately from M5.

**M7 — Live voice/video. Last, gated, honestly speculative.** Zero open
reference exists (recon: no WebRTC/RTP/codec repo in the org; community
attempts abandoned). Path: SDP/ICE signaling over the room's swarm +
browser-native WebRTC for media — which may need STUN/TURN, and a TURN
relay reintroduces a server. That tension with the sovereignty thesis is a
Commander design conversation BEFORE any M7 work starts, not after.

**M8 — The mirror.** `mesh/docs/MESSENGER_DECISIONS.md` (MSG-D1…,
`[Mirror]` style) + progress in `mesh/docs/MESH_PROGRESS.md` (the messenger
is a mesh track). Written when decided, per wave, honest.

## 4. Invariants (inherit Sovereign Mesh §4; these are added)

1. **One capability plane.** Room permissions are Mission D grants. No
   role flags in message payloads, no second ACL system, ever.
2. **The fold accepts chat, verifies everything.** Every room op is
   signed (devicePub+sig, existing envelope) and capability-checked; but
   `msg.*` kinds never REJECT on content — chat has no oversell. Malformed
   ops are skipped (poison-pill discipline, hardened 2026-07-16), never
   crash the fold.
3. **Graduation is human-signed.** No path exists from room log to
   business log without a human-actor op. Butler drafts carry
   `actorType:'agent'` and would be rejected as deciders by the kernel —
   keep it that way and tripwire it (the frontend butler boundary test is
   the pattern).
4. **Attachments are content-addressed.** A message references a blob by
   key+hash; the fold never inlines file bytes into the room log.
5. **Delivery states are typed and honest.** No fake "delivered". The
   UX vocabulary is `local | replicated | mirrored | read`, derived from
   observable facts (peer acks, mirror acks, read-cursor ops).
6. **No real client data in the repo.** Synthetic canon only — chat
   fixtures included (no real names, numbers, or business content).

## 5. Stop-and-ask registry (Commander calls)

- **Where the messenger surfaces in the product** (a screen in AsymmFlow's
  kernel frontend? a sibling window? standalone app?) — UI placement is an
  owner call at M1's end; the data model does not depend on it.
- Anything M7 (calls): the TURN/STUN centralization tension, codec
  choices, per-platform webview media spikes.
- App-store submission strategy for the M5 companion (Bare binary review
  risk — recon §6).
- Hosting/operating any shared infrastructure (blind mirrors for clients,
  push gateway) — a service-tier business decision, not an eng default.
- Any interop with actual Keet/WhatsApp bridges (scope trap; default NO).
- Upstream license nudge: filing the corestore LICENSE-file issue
  (cosmetic; npm declares MIT — verified) — file it, but ask before any
  vendoring beyond package pinning.

## 6. WAVE 1 SPEC — Mission M1: the room fold (the spike is the gate)

**Goal:** prove the chat data model end-to-end through the REAL machinery —
the same bar Missions A/C/D passed: 3 peers, genuine offline fork,
byte-identical convergence, goldens, and the capability plane active.

**Branch discipline:** `exp/sovereign-mesh` worktree
(`C:\Projects\asymmflow\asymmflow-mesh`), new `mesh/rooms/` +
`mesh/reducer` extensions. LOCAL-ONLY, never pushed. Zero contact with
`frontend-lab/`, the PH fork, or `wails.json`. All existing mesh gates
(smoke, wave1, missionc, missiond, holesail stages) must stay green —
they are the regression floor.

### 6.1 The op vocabulary (design frozen at wave start, mirrored at wave end)

All ops ride the EXISTING signed envelope (seq, actor, ts, kind,
devicePub, sig — capability-checked by the existing plane). New kinds:

- `room.manifest` — first op, authority-signed: `{title, anchor:{type,id},
  rules:{observersAllowed}}`. Exactly one; later manifests are skipped.
- `msg.post` — `{msgId, body, replyTo?, attachment?}`. msgId =
  `{actor}:{seq}` (deterministic, no uuids/rand).
- `msg.edit` — `{msgId, body}`. Only the original author's devicePub may
  edit; last edit in canonical order wins.
- `msg.delete` — `{msgId}`. Author or room authority. Tombstone: the fold
  keeps the id, blanks the content (append-only log ≠ erased bytes; the
  UX says "deleted", the log stays honest).
- `msg.react` — `{msgId, emoji, on:true|false}`. Toggle semantics per
  (msgId, emoji, actor); last toggle in canonical order wins.
- `msg.read` — `{upTo:{actor,seq}}`. Per-writer monotonic read cursor
  (only advances; a lower cursor is skipped, not rejected).
- `msg.draft-op` — `{msgId, draft:{...business op...}}`. The graduation
  seam: carried as INERT DATA in the room (the room fold never executes
  it). Wave 1 proves it folds and displays; actual graduation into the
  business base is M2+ scope with a human-signed confirm.

### 6.2 The fold (Go, wasip1, same law)

`mesh/reducer` grows a `rooms` domain (package `reducer`, file
`room_domain.go` + `room_test.go`): RoomState = `{manifest, messages
(ordered), reactions, readCursors, applied, skipped, digest}` — digest over
the canonical map-free projection, exactly the Mission C pattern. Determinism
landmines: msgIds are derived (no rand); edit/react resolution is
canonical-order-based (never wall-clock); tombstones deterministic. The
capability plane wraps it unchanged: a revoked device's messages stop
folding at the epoch bump, identically on every peer (Mission D semantics,
now with chat as the payload).

**Skipped-not-rejected:** chat ops that fail their own rules (edit by
non-author, react to unknown msgId, stale read cursor) are SKIPPED with a
typed reason in `skipped[]` — the CRDT half of the taxonomy; `rejected[]`
stays reserved for capability/kernel law.

### 6.3 The gates (all must be GREEN to call Wave 1 done)

1. **Unit law** (`room_test.go`): schema round-trip, edit-authorship,
   tombstone, react-toggle, cursor monotonicity, manifest-uniqueness,
   500-permutation convergence over a mixed room scenario, input
   immutability, and **revocation-mid-conversation** (epoch bump between
   two of a device's messages — first folds, second doesn't, everywhere).
2. **`npm run roomspike`** (`mesh/host/room-spike.mjs`): 3 real peers
   (authority + two members), genuine offline fork, a full conversation
   (posts, replies, an edit, a delete, reactions, read cursors, one
   `msg.draft-op` from an `actorType:'agent'` writer), reconverge →
   byte-identical views + room state on all peers; golden
   (`goldens/room_autobase.json`), reproducible.
3. **Regression floor:** every existing mesh gate green, existing goldens
   UNTOUCHED (room kinds must not perturb legacy folds — the MESH-D12
   opt-in pattern shows how).
4. **Mirror:** MSG-D1..Dn decisions + MESH_PROGRESS wave entry, honest
   line included (what Wave 1 does NOT prove: invites, blobs, UI, mobile,
   delivery states beyond replication).

### 6.4 Explicitly OUT of Wave 1

Invites/blind-pairing (M2) · attachments (M3) · any UI (owner placement
call pending) · mirrors/push/mobile (M4-M6) · calls (M7) · any business-base
graduation execution · any touch of the PH track.

## 7. Reference docs

- `C:\Projects\asymmflow\MESSENGER_RECON.md` — THE ground truth: solved-vs-
  gap matrix, license table (verified), risks, per-dimension appendix.
- `FABLE_CAMPAIGN_SOVEREIGN_MESH.md` + `mesh/docs/MESH_DECISIONS.md`
  (MESH-D1..D13) + `mesh/docs/MESH_PROGRESS.md` — the foundation and its law.
- `mesh/host/{mesh-node,capability,peer}.mjs`, `mesh/reducer/*` — the
  running code every mission composes on.
- blind-pairing: https://github.com/holepunchto/blind-pairing (M2) ·
  Hyperdrive/Hyperblobs: https://docs.pears.com/ (M3) ·
  blind-peer: https://github.com/holepunchto/blind-peer (M4) ·
  bare-kit: https://github.com/holepunchto/bare-kit (M5).
- `asymmflow-integration-discussion-summary.md` — the stakeholder-mesh
  vision this serves (Layers 2-4, ephemeral contracts, meta-pattern).

## 8. Honest risks

- **Scope gravity.** A messenger invites infinite feature surface
  (presence, typing indicators, search, pins…). The wave discipline is the
  defense: every rung ships alone or not at all.
- **Chat-volume performance.** Refold-per-read (MESH-D7) is O(n) — fine
  for a spike, wrong for a 10k-message room. The `//go:wasmexport`
  incremental reactor (MESH-D6's deferred endpoint) likely graduates from
  "optimization" to "required" during M3/M4. Price it then, not now.
- **Two UX cultures.** ERP users tolerate honesty; chat users expect
  WhatsApp's lies (instant everything). The typed delivery states will need
  real UX craft to feel calm instead of pedantic.
- **Upstream POC dependencies at the top of the ladder** (blind-push,
  Mailbox). Mitigated by ladder order: they're M4+/M6, after the core is
  proven on stable ground.
- **The PH track owns the calendar.** This campaign yields the road the
  moment convergence work needs hands. Horizon, not critical path.

Build → Test → Ship. Measure, don't estimate. The spike is the gate; the
ground wins; the mirror records why. 🌊
