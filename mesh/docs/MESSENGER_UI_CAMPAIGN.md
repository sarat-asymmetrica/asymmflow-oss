# MESSENGER UI CAMPAIGN — "The Correspondence Desk"

**Status:** RATIFIED 2026-07-18 (joint owner/lead design session; rulings MSG-D23/D24).
**Branch:** `exp/sovereign-mesh` (worktree `C:\Projects\asymmflow\asymmflow-mesh`, LOCAL-ONLY).
**Governing law:** `MESSENGER_DESIGN_CONSTITUTION.md` (all 12 articles),
`MESSENGER_DECISIONS.md` (MSG-D1..D24), `AGENT_GATE_LEDGER.md` (GL-1..6 — BINDING on
every coder agent; read before touching anything).

## 0. Thesis

The messenger's UI is not a chat app bolted onto the ERP — it is the same room fold
rendered in two places: a Correspondence screen (the mini-email morning-pass surface)
and an `embedded` per-object conversation hosted inside existing detail screens.
The mesh cannot live inside the wails app until DP4 (Bare sidecar), so this campaign
builds the UI against **the seam**: a protocol shaped exactly like the future sidecar's,
served today by the Node host. UI ships real (actual wasm fold underneath via the dev
bridge); transport swaps at DP4 without a redesign.

## 1. The seam — sidecar protocol v0 (THE contract, both missions build against this)

Transport (dev): newline-delimited JSON over localhost TCP (`node:net`, ZERO new deps —
Bare-portable). Transport (DP4): sidecar stdio, same frames. One frame = one JSON
object per line.

```
Request:  {"id": n, "method": "...", "params": {...}}
Response: {"id": n, "ok": true, "result": ...} | {"id": n, "ok": false, "error": "..."}
Event:    {"event": "...", "params": {...}}            (no id; server-initiated)
```

Methods v0:

- `hello()` → `{devicePub, actor, version}` — identity of this node's device/actor.
- `listRooms()` → `[{roomKey, title, kind: "anchored"|"social", anchorType, anchorId,
  lastSeq, lastTs, lastPreview, topExpectation}]` — topExpectation = highest-urgency
  expectation tag on a message addressed at/after my cursor position ("" | "whenever"
  | "today" | "urgent"); drives the inbox float-to-top sort.
- `roomState(roomKey)` → `{manifest, members, claim, capEpoch, messages: [{seq, actor,
  ts, body, expectation, attachment: {name, size, sha256, ref} | null}], skippedCount,
  rejectedCount}` — messages in canonical fold order, verbatim from the room state.
- `post(roomKey, {body, expectation})` → `{seq}` — expectation ∈ the Art. III
  vocabulary; invalid values are the FOLD's job to skip, the bridge never pre-filters
  beyond the composer's own UI (fold law is not re-implemented client-side).
- `claimRoom(roomKey, {assignee})` / `releaseClaim(roomKey)` → `{seq}` — anchored only
  (the fold skips it in social rooms; the bridge passes the skip reason through).
- `attach(roomKey, {filePath, body, expectation})` → `{seq, ref, sha256}` — Hyperblobs
  put + msg.post with attachment locator (M3 machinery, untouched).
- `fetchAttachment(roomKey, {ref, savePath})` → `{path, sha256, verified}` — end-to-end
  sha256 verify on fetch, always.
- `createSocialRoom({title})` / `openDmInvite(roomKey)` → `{inviteCode}` (asymm-room2).
- `redeemInvite({inviteCode, actor})` → `{roomKey}` — the M2 ceremony.
- `exportTranscript(roomKey)` → the asymm-transcript.v1 bundle (Art. V §5, self-serve).

Events v0: `{"event": "room-updated", "params": {"roomKey"}}` on any fold change the
host observes. Coarse on purpose — the client refetches roomState; no incremental
diffs in v0.

Explicitly NOT in v0: wave/kettle's-on (needs `msg.wave` reducer kind — W-UI-2
mini-mission), read cursors (anchored-only UI comes with the chrome wave), quiet
hours, graduation gesture, ERP anchor-summary resolution (DESK pane, W-UI-2+).

## 2. Waves

- **W-UI-1 "The Seam & the REPL"** (this wave, two parallel missions):
  - **Mission U1 (mesh/host):** `bridge-server.mjs` serving protocol v0 over localhost
    TCP around the existing host modules (mesh-node/social-room/invite/attachments/
    export-transcript — REUSE, don't reimplement); a spike (`bridgespike`) driving two
    in-process nodes end-to-end through the protocol INCLUDING an attach/fetch
    round-trip; honest runtime asserts, golden discipline per GL-2 (state pins only
    where concurrency exists).
  - **Mission U2 (frontend-lab):** `bridge/mesh.ts` (mock-first per the repo's own
    `pick()` discipline; mock fixtures shaped EXACTLY as protocol-v0 frames),
    `Correspondence.svelte` bespoke screen + `correspondence-vm.svelte.ts` registered
    in `registry.ts`; flat inbox + FilterChips (All | Rooms | People) with
    expectation float-to-top; thread rendered via ActivityFeed (REPL-first — message
    cards come at W-UI-2); composer = plain input with `/`-command support
    (`/expect today|urgent|whenever`, `/claim`, `/release`, `/invite`, `/attach`) plus
    an expectation chip row; urgent = left-border + float, today = quiet tint, no
    other signal (MSG-D23 #5). `svelte-check` 0/0 required; mock-safe tripwires
    respected.
- **W-UI-1.5 "The Kitchen Table Kit"** (MSG-D24): packaging script emitting the
  portable field kit (DP1 plane layout: `portable.flag`, `host\`, `ui\`,
  `data\corestore\` + `data\keys\` SIBLING, `run_mesh.cmd`,
  `README_KITCHEN_TABLE.txt` two-machine ceremony walkthrough). Gated on U1+U2 green.
  Synthetic canon only; real-DHT first-packet fact stated in the README.
- **W-UI-2 "Three Panes & the Wave"**: full chrome (message cards, DESK pane with
  claim/members/anchor summary/evidence export), `msg.wave` reducer mission (v2
  signable growth + authorized re-golden), real-transport wiring of `bridge/mesh.ts`.
- **W-UI-3 "The Embedded Room"**: `embedded` hosting on WorkHub project detail +
  Customer360; graduation gesture via `setHandoff`; "also assign the task" seam
  (MSG-D23 #8).
- **W-UI-4 "First Listener"**: live push — the kernel's first real EventsOn-pattern
  consumer (dev: bridge events; DP4: sidecar events, same handler).

## 3. Law reminders for coders (non-exhaustive — the ledger is the law)

- The fold is the only rule engine. UI/bridge NEVER re-implement chat law
  (expectation validation, claim rules, social-room skips) — they surface the fold's
  skip/reject reasons verbatim.
- No read receipts, no typing indicators, no delivery ticks — anywhere, ever (Art.
  III/IV). Do not add "helpful" presence.
- Social rooms render with NO org furniture (no claim UI, no cursors, no DESK).
- Determinism canon in anything host-side: no live clocks in op data, deterministic
  seeds in spikes, canonical-order awareness (GL-5 seq discipline).
- Coder agents never commit; the lead gates and commits. Deliver your report BEFORE
  idling (GL-3).
