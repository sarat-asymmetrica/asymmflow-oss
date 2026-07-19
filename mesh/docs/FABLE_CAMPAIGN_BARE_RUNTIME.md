# Fable Campaign — Bare Runtime Integration ("The Sealed Ship")

**Status:** RATIFIED 2026-07-19 · **Orchestrator:** Opus 4.8 (long-running autonomous wave)
**Coders:** Sonnet 5 · **Spec author & final gate:** Fable · **Owner:** the Commander
**Prior art (MANDATORY reading):** `mesh/host/bare-spike/BARE_SPIKE_REPORT.md` (C4's findings),
`mesh/docs/MISSION_A2_CORRIDOR_SPEC.md` (incl. the A2.1 addendum & field report FR-1),
`mesh/docs/MESSENGER_UI_CAMPAIGN.md` §1 (protocol v0 + the DP4 stdio seam).

---

## 0. Charter

Replace the Node.js sidecar and the bundled-node field kit with the **Bare runtime**,
delivered as a **sealed artifact** whose failure modes cannot include missing-module
resolution, PATH lookups, or any of the Node-distribution disease class that produced
field report FR-1. The protocol-v0 bridge seam is the fixed migration boundary; the
mesh law (reducer, capabilities, invites, rooms) does not change.

The Node kit (Mission A2/A2.1 line) is CLOSED as legacy learning. It remains in the
tree, its spikes remain green, and it is the rollback path — but it receives ZERO
further investment. Its transferable assets — the ceremony design, the Guided Path UX
(START_HERE, plain questions, paste-and-Enter, verdict words), the probe's diagnostic
vocabulary, the anchor role, the geography-hermetic gate discipline — are REQUIREMENTS
on the successor, not suggestions.

## 1. Doctrine (binding on the orchestrator and every coder)

- **D1 — Zero assumptions; documentation first.** Before ANY design or code, the
  orchestrator runs a full documentation-immersion phase (§3 Phase 0) reading the
  CURRENT docs and source of Bare and the Holepunch stack. Training-vintage
  impressions of what is "complex," "immature," or "risky" are inadmissible evidence.
  Every claim about what Bare can/cannot do must cite a doc page, a repo file, or a
  spike transcript produced in this campaign.
- **D2 — "Simpler" is a human-developer illusion.** Paths are evaluated by
  destination-fitness, never by familiarity. An LLM's cost for the "hard" path is the
  same tokens minus the rework. (Owner doctrine, 2026-07-19.)
- **D3 — The complexity is ours; the simplicity is for the end user.** Internal
  build/dev friction is acceptable cost. Client-facing friction is never acceptable.
  Ruling R6 stands: clients are never handed a command line.
- **D4 — Do it right, do it once.** Interim scaffolding is permitted ONLY to retire a
  genuine unknown, and each rung must name the unknown it retires. A rung that will be
  discarded and retires no unknown is skipped.
- **D5 — Honest gates, hostile geography.** Every gate that executes a built artifact
  runs it from a location outside the repo tree (FR-1b lesson). Every report states
  what was NOT verified. A findings report that says "blocked, here is exactly why"
  is a successful deliverable (C4 precedent).
- **D6 — Mesh law is frozen.** `mesh/reducer/**`, capability semantics, invite
  semantics, protocol-v0 method semantics: any change = stop and report to the owner.

## 2. The known landscape (verified 2026-07-19 — re-verify, do not trust)

From C4's spike (Bare 1.30.3, Windows 11, x64):
- Bare runtime executes on Windows via npm prebuilds, zero build step.
- All 11 Holepunch packages used by the mesh (`autobase`, `corestore`, `hypercore`,
  `hyperbee`, `hyperblobs`, `hyperswarm`, `hyperdht`, `hypercore-id-encoding`,
  `blind-peer`, `blind-peering`, `protomux-wakeup`) import clean under Bare, no shims.
- `bare-process` stdio behaves stream-like; the ndjson framing of protocol v0 ports
  verbatim from TCP to stdio.
- `WebAssembly.compile()` succeeds on the real `reducer.wasm` (3.96 MB).
- **THE GAP:** no WASI preview1 host exists for Bare (`bare-wasi` is a 404). The
  reducer's import table needs exactly 16 `wasi_snapshot_preview1` syscalls
  (enumerated in the report): sched_yield, proc_exit, args_get, args_sizes_get,
  clock_time_get, environ_get, environ_sizes_get, fd_write, random_get, poll_oneoff,
  fd_close, fd_read, fd_fdstat_get, fd_fdstat_set_flags, fd_prestat_get,
  fd_prestat_dir_name. No path_open, no dir walking.
- **THE ALTERNATIVE:** `host/apply.mjs`'s own header flags a future
  `//go:wasmexport apply()` build (Go 1.24+ wasip1 reactor exports / js-less export
  surface) that would need NO WASI import table at all.

## 3. Phases

### Phase 0 — Documentation immersion & environment truth (no code)
Read thoroughly, with notes committed as `mesh/docs/bare-campaign/PHASE0_NOTES.md`:
- docs.pears.com in full (Bare guides, Pear runtime, packaging/distribution model).
- github.com/holepunchto/bare + the bare-* module family actually needed (bare-fs,
  bare-process, bare-stream, bare-subprocess, bare-os, bare-path, bare-tcp if used).
  Read READMEs AND enough source to confirm API shapes on Windows.
- Bare packaging/distribution tooling as it exists NOW: `bare-pack`/`pear stage`/
  linked binaries — whatever the current mechanism is for producing a single sealed
  artifact with prebuilds embedded. THIS IS THE LOAD-BEARING QUESTION of the campaign:
  how does a Bare app ship to a Windows machine as one file or one sealed folder?
- Holesail docs + source (docs.holesail.io, github.com/holesail) — as a consumer
  (optional transport), and as UX prior art (connection string / QR conventions).
- Go's wasip1 story CURRENT state: `go:wasmexport`, reactor mode, wasmexport
  restrictions, tinygo alternative if stock Go blocks.
- Re-run C4's spike scripts against current package versions; record deltas.
Gate P0: notes doc exists, every §2 claim re-verified or corrected with citations,
and a written answer to the sealed-artifact question with the exact tooling named.

### Phase 1 — The runtime-gap decision spike (the fork: WASI shim vs wasmexport)
Build BOTH candidate bridges as minimal spikes, measure, decide by evidence:
- **1a WASI-shim path:** a Bare-native `wasi-preview1-lite.mjs` implementing exactly
  the 16 syscalls against bare-fs/bare-process; run the UNMODIFIED reducer.wasm
  through the existing apply.mjs contract under Bare; golden-vector parity against
  the Node WASI host (same ops in → byte-identical state out).
- **1b wasmexport path:** a parallel reducer build (`cmd/reducer` variant) exposing
  `apply` via `go:wasmexport` with a linear-memory I/O convention; a thin Bare host
  calling it; the same golden-vector parity.
- Decision matrix: correctness parity, artifact size, call overhead, build-chain
  complexity, upstream-contribution value (a working `bare-wasi` lite is a
  contributable package — weigh it), maintenance surface.
Gate P1: both spikes' honest transcripts + a decision memo. STOP-AND-ASK the owner
only if both paths fail or the winner requires changing reducer source semantics (D6).
Otherwise the orchestrator decides and records the ruling.

### Phase 2 — The Bare bridge (DP4 executed)
Port `bridge-server.mjs` to run under Bare with stdio ndjson framing per the
MESSENGER_UI_CAMPAIGN §1 seam (TCP mode retained for dev). The bridge-spike's 30
checks run under Bare (adapted runner, same assertions, same fold verdicts).
Gate P2: bridge-spike parity under Bare, geography-hermetic; deviations enumerated.

### Phase 3 — The sealed kit ("Machine-B, final form")
Successor to the field kit, built with the Phase-0-identified packaging mechanism:
- ONE sealed artifact (single binary or sealed folder with embedded prebuilds — per
  Phase 0's answer), containing runtime + mesh host + reducer + guided path.
- The Guided Path UX ports as law: START_HERE double-click, the menu, plain
  questions, paste normalization, verdict words, conversational firewall step,
  error fold-lines. (Port `guide.mjs` semantics; reimplementation of the UI layer is
  expected and permitted — it is presentation, not mesh law.)
- Probe and anchor roles port: same diagnostic vocabulary, same anchor ceremony
  (scheduled task now launches the sealed artifact).
- The FR-1 disease class must be UNREPRESENTABLE: no module resolution at runtime
  outside the sealed artifact, no PATH consultation, no separate node_modules.
Gate P3: full kit-spike-equivalent + guide-spike-equivalent suites, geography-
hermetic, PLUS a "hostile machine" rehearsal: fresh Windows VM or cleaned directory
with no Node, no npm, no dev tooling — extract, double-click, full ceremony.

### Phase 4 — Corridor re-gate & cutover
- Two-machine corridor rehearsal on the sealed artifact (LAN + real DHT), then the
  India↔Bahrain field ceremony re-run when the field contact is available.
- Anchor migration: uninstall Node anchor task, install sealed-artifact anchor,
  verify heartbeat + convergence.
- Ledger closure: field results recorded; Node kit formally marked legacy in
  mesh/README.md; campaign retrospective doc (what the docs said vs what was true —
  feed corrections upstream as issues/PRs where valuable, per the give-freely ethos).
Gate P4 (FINAL, Fable + owner): sealed artifact carries the corridor end-to-end with
zero client-facing friction beyond: extract, double-click, answer questions.

## 4. Stop-and-ask triggers (owner decisions, everything else is orchestrator's)
1. Any change to reducer source semantics or mesh law (D6).
2. Both Phase-1 paths blocked.
3. Sealed-artifact mechanism requires adopting the full Pear runtime as the app model
   (vs Bare-as-library) — that's an architecture identity question.
4. Any new heavyweight dependency outside the Holepunch/bare-* family.
5. Anything requiring changes to the OSS ERP (this campaign is mesh-side only).

## 5. Working agreements
- Branch: `feat/fable-bare-runtime` off main; commits per phase-band; no push until
  Fable's final gate unless the owner rules otherwise mid-campaign.
- Single-writer per file per band; parallel coders coordinate by filename contracts
  named in the phase briefs (A2 discipline).
- Every phase produces an honest report: verified / not-verified / deviations.
  Reports are appended under `mesh/docs/bare-campaign/`.
- Existing Node-line spikes stay green throughout (the rollback path stays warm).
- Synthetic identities only, everywhere (I4 / GL-12).

*The ladder is over. Read everything, assume nothing, seal the ship.* 🐻
