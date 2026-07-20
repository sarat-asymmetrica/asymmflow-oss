# Phase 2 Report — the Bare bridge (DP4 stdio seam)

**Coder:** P1-A · **Date:** 2026-07-20 · **Branch:** `feat/fable-bare-runtime`

## 1. Verdict

**YES.** Protocol v0 (`mesh/docs/MESSENGER_UI_CAMPAIGN.md` §1) works over
ndjson-framed stdio under Bare — a new dispatch core (`bare-bridge.mjs`)
built by re-deriving bridge-server.mjs's own logic from the SAME underlying
host modules (mesh-node.mjs, capability.mjs, social-room.mjs,
invite-code.mjs, attachments.mjs, export-transcript.mjs — reused, never
reimplemented), fed through a real ndjson stdio transport, passes the same
30-scenario behavioral suite bridge-spike.mjs gates on **plus a real
`child_process.spawn`-driven leg** (45 checks total under Node, 37 under
Bare — see §2) under **both** `node` and `npx bare`, including from outside
the repo tree. The Node bridge line (`bridge-server.mjs` /
`bridge-spike.mjs`) is untouched and still green — the rollback path stays
warm.

Two cross-cutting blockers surfaced and were resolved **during this phase,
live, by the concurrent P0-B-packaging migration** (not by this coder,
outside this coder's file fence) — flagged to the lead as they were found
(msg_id `39a3f47d`, `deec0a42`) and confirmed fixed before the final gate
run: `mesh-node.mjs`'s reducer channel (`./apply.mjs` → `#apply` condition
map) and the `#fs`/`#crypto` condition-map entries this file's `attach`/
`fetchAttachment` needed. A third, smaller one (`verify-transcript.mjs`
still imports `node:crypto` + `./apply.mjs` directly, unmigrated) remains
open — see §3.

**A THIRD round of correction happened mid-phase, and it mattered more than
either blocker above.** This file's own FIRST flush-race investigation
(shell pipes, §4's earlier draft) reported "0 truncations in 45 runs" —
which was true, and also not evidence of much: the campaign lead
independently re-tested with the REAL production topology
(`child_process.spawn`, parent reading `stdout.on('data')`) and found TWO
real bugs the shell-pipe method could not see (`PHASE0_NOTES_D2_FLUSH_
RACE.md`). `bare-bridge.mjs` was fixed for both (RULE 1/2/3 below) and
`bare-bridge-spike.mjs` gained a real spawn-driven leg plus a negative
control that independently reproduces one of the two bugs against the
campaign's own known-broken script. This is the accurate, current account;
§4 below is written against it, not the earlier draft.

## 2. Check results, both runtimes

`bare-bridge-spike.mjs` runs THREE layers of proof (its own header explains
each in full):

1. **Layer 1 — the behavioral suite.** Bridge-spike.mjs's own scenario
   (anchored PO room, hub+desk, malformed-frame resilience, listRooms/
   roomState, post + expectation validation + urgency float, claimRoom/
   releaseClaim, attach→fetchAttachment sha256 round-trip, exportTranscript
   verified via verify-transcript.mjs where available, room-updated events
   (self + replicated), GL-5 seq-continuation across a bridge-core restart,
   social room + real invite redeem under encryption, the claim-skip proof
   in a social room), driven through `createBridgeCore.dispatch()` via a
   thin in-process "wire client" that still round-trips every call through
   real `JSON.stringify`/`parse`.
2. **Layer 2 — the frame loop.** 7 checks feeding literal ndjson byte
   strings (a request split across two `onData` chunks, a malformed line,
   a blank line, two frames in one chunk) through the REAL
   `attachStdioTransport` buffering code, in-process.
3. **Layer 3 — the real spawn topology.** `node:child_process.spawn`
   (Node-only, guarded — see §3 for why this is the one sanctioned
   exception to "no node: specifier"), parent reading `child.stdout.on
   ('data', ...)`, matching EXACTLY the shape the campaign lead's
   `PHASE0_NOTES_D2_FLUSH_RACE.md` used to find the two real bugs a shell
   pipe hid. Spawns the worker under `node` AND under `npx bare`, sends 3
   requests including one that touches the reducer
   (`createSocialRoom`), asserts all 3 responses arrive with matching ids
   and the process exits cleanly (no hang) — **plus a negative control**
   that spawns the campaign's own known-broken script
   (`host/bare-spike/stdio-check.mjs`) and asserts THIS harness correctly
   flags it as broken, not green (see §4).

| Run | Checks | Failures | Result |
|---|---:|---:|---|
| `node host/bare-bridge-spike.mjs` | 45 | 0 | **GREEN** (includes layer 3, both spawn targets + the negative control) |
| `npx bare host/bare-bridge-spike.mjs` | 37 | 0 | **GREEN** (layer 3 self-skips under Bare — `node:child_process` is Node-only by design; 1 layer-1 check downgraded to a logged skip — see §3 deviation #6) |
| `node host/bridge-spike.mjs` (frozen Node/TCP line) | 30 | 0 | **GREEN**, unchanged |
| `npm run bareparity` (Phase 1a suite) | 13 scenarios | 0 | **GREEN** |
| `npm run bareparity:bare` | 13 scenarios | 0 | **GREEN** |
| `npm run smoke` | — | 0 | **GREEN** |
| `node host/reactor-parity-spike.mjs` (Phase 1b suite, another coder's file, read-only here) | 13 scenarios | 0 | **GREEN** |

Re-run 3 back-to-back times under Node (45/45 every time — layer 3 is the
newest, timing-sensitive code, worth the extra stability check) and once
more under Bare, all green. Both bare-bridge-spike.mjs runs also succeeded
from `C:\Users\schan\AppData\Local\Temp\claude\...\scratchpad\p1a` (outside
the repo tree, absolute script path, hostile CWD) — see §5.

## 3. Deviations from the Node bridge, enumerated

**Preserved exactly** (bridge-server.mjs's own 4 ratified deviations from
the literal §1 text): the `rooms` Map (not a single `node`), `addWriter`
staying host-side (not a wire method), the optional `ts`/`inviteSeed` for
deterministic drive, `createSocialRoom` → `{roomKey}` / `openDmInvite` →
`{inviteCode}`. GL-5 seq discipline (global-max-seq counter, never
per-actor, never restarted) is preserved verbatim — re-checked against the
scenario's own restart step (§2's GL-5 check), still green.

**New deviations, this phase's own, all declared:**

1. **Dispatch core / transport split.** bridge-server.mjs fuses method
   dispatch and `net.createServer`'s socket handling in one function.
   `bare-bridge.mjs` splits these into `createBridgeCore` (pure dispatch,
   zero I/O, directly unit-testable) and `attachStdioTransport` (the ndjson
   framing loop, injected `io`). Necessary, not cosmetic: the transport
   genuinely differs (stdio vs. multi-socket TCP), and DI-testability was a
   binding constraint (dependency injection preferred over runtime
   ternaries per `PHASE0_GATE_B3_CONDITION_MAP.md`).
2. **One stdio channel, not many sockets.** The DP4 shape is one sidecar
   process per device, talking to its own parent (the wails app) over its
   own stdin/stdout — there is no "many concurrent clients" concept at this
   layer the way `bridge-server.mjs`'s TCP listener has. `rooms`/multi-room
   serving is unchanged; it's the client-multiplicity assumption that
   drops, because it was never true for a sidecar.
3. **`fs` via the `#fs` condition map**, not a runtime `isBare` ternary
   importing `node:fs` — the exact pattern `PHASE1_DECISION_MEMO.md`
   flagged as `apply-bare.mjs`'s own known pack-time blocker. This file
   never repeats it.
4. **Random storage-dir/invite-seed bytes via `hypercore-crypto`'s
   `randomBytes`** (already a direct dependency, proven Bare-clean),
   not `node:crypto`'s `randomUUID`/`randomBytes`. IDs are random hex
   strings, not RFC-4122 UUIDs — functionally equivalent for "make a
   unique storage path", never parsed as a UUID anywhere downstream.
5. **Hand-rolled path helpers** (`joinPath`/`dirnameOf`/`baseNameOf`,
   three trivial string functions) instead of a `#path` alias — judged too
   small a need to justify one more runtime-primitive surface. Consequence:
   `fetchAttachment`'s returned `path` is NOT run through `path.resolve()`
   the way bridge-server.mjs's is — it returns the joined/given path
   as-is. Every caller in this file's own spike passes an already-workable
   path, so this was never exercised as a real gap, but it IS a behavioral
   difference from the Node bridge, declared here rather than silently
   matched by accident.
6. **`verify-transcript.mjs` is unavailable under Bare** — it directly
   imports `node:crypto` and `./apply.mjs`, and is not on P0-B-packaging's
   migration list (confirmed by reading its current top-of-file imports).
   This is NOT a change in this phase's own files; it's an unplanned third
   cross-cutting blocker discovered while wiring `exportTranscript`'s
   verification check. Handled with a guarded dynamic import: under Bare,
   that one check is skipped with a clear stderr note rather than
   crashing the whole spike at module load. `exportTranscript` itself
   (the bridge method) works fine under Bare — only the SEPARATE,
   independent-of-this-phase verification helper does not load.
7. **`Number`/`String`/`Buffer`-only op literals**, same canon as Phase 1a
   — no behavioral deviation, just consistency with the established
   dual-runtime discipline.
8. **`node:child_process` in `bare-bridge-spike.mjs` only, scoped and
   guarded.** The one sanctioned exception to "no node: specifier in your
   files" — see §4's layer 3. Never appears in `bare-bridge.mjs` (the file
   that ships); the spike is dev/test tooling, like every other
   `*-spike.mjs` in `mesh/host`, none of which are packed into the sealed
   artifact.

## 4. The stdio hazards — corrected account (this section was rewritten
mid-phase; see §1's "third round of correction" note for why)

**The bugs, root-caused by the campaign lead against the REAL production
topology** (`child_process.spawn`, parent reading `stdout.on('data')` —
full detail in `PHASE0_NOTES_D2_FLUSH_RACE.md`, not reproduced here in
full):

- **Bug A**: `await WebAssembly.compile()`/`instantiate()` silently drops
  stdout (27-53% loss depending on exact pipe shape), exit code 0, no
  error.
- **Bug B**: `bare-process`'s `process.stdout.write()` HANGS on a real
  spawned pipe (measured 30/30) — independent of Bug A, no wasm involved
  at all.
- **Why this campaign's own DP4 proof-of-concept
  (`host/bare-spike/stdio-check.mjs`) passed for a full day while
  exhibiting Bug B**: it was verified with a shell pipe, which hides both
  bugs. This is the same "the control you skip is the one hiding the
  defect" lesson as `PHASE0_GATE_B3_CONDITION_MAP.md`'s finding, recurring
  at a different layer.

**What `bare-bridge.mjs` does about it (three rules, all now verified
against the real spawn topology by this phase's own layer-3 gate, not just
asserted):**

- **RULE 1** — the reducer channel this bridge calls into
  (`mesh-node.mjs` → `apply-bare.mjs`) already used only the synchronous
  `new WebAssembly.Module()`/`new WebAssembly.Instance()` forms since
  Phase 1a; confirmed still true (`grep -n "WebAssembly\."` across both
  files).
- **RULE 2** — `getRealStdio()`'s `write` uses `console.log`, never
  `proc.stdout.write()` directly (fixed this phase; the original draft of
  this file used `proc.stdout.write()`, exactly Bug B's shape, and would
  have hung on a real spawn — caught before shipping, not after).
- **RULE 3** — `runStdioWorker` calls an explicit `Bare.exit(0)`/
  `process.exit(0)` on stdin `'end'` (added this phase; the original draft
  had NO end handler at all — an omission, not a deliberate "let it drain
  naturally" choice, but the same failure mode P0-D's own removal
  experiment hit: 10/10 hangs). This file's own addition beyond the rule
  as written: `attachStdioTransport.waitIdle()` drains any in-flight
  `dispatch()` promise BEFORE exiting, so a request that arrives just
  before the writer closes stdin still gets its response written rather
  than truncated by an immediate exit. Not verified against a real
  reproduction of THAT specific race (no existing repro script has a
  request/response cycle to drain) — flagged in §6.

**Layer-3 gate results (the real proof — spawn+pipe, not shell pipe, not
in-process):**

| Check | Result |
|---|---|
| `spawn(node)`: worker answers 3 requests (incl. one touching the reducer) over a real spawned pipe | **PASS**, 3/3 responses, correct ids |
| `spawn(node)`: worker exits cleanly on stdin end, no hang | **PASS** |
| `spawn(bare)`: same, spawned via `npx bare` | **PASS**, 3/3 responses, correct ids |
| `spawn(bare)`: worker exits cleanly on stdin end, no hang | **PASS** |
| **Negative control**: spawn the campaign's own known-broken `host/bare-spike/stdio-check.mjs` under `npx bare`, same real topology | **CORRECTLY FLAGGED AS BROKEN** — produced only `{"event":"ready"}`, the echo line never arrived, exactly Bug B's signature, independently reproduced by this harness, not just cited from the lead's report |

Re-run 3× under Node (45/45 every time) plus once more under Bare and once
more each from outside the repo tree (§5) — stable, no flake observed in
this phase's own sampling.

**One bug found and fixed by this exact gate, worth stating plainly**: the
first version of the layer-3 leg failed `spawn(node)` with
`'C:\Program' is not recognized as an internal or external command` —
`shell:true` (needed for `npx` to resolve on win32) was ALSO being applied
to the direct `process.execPath` spawn, and cmd.exe re-splits an unquoted
path containing spaces ("C:\Program Files\nodejs\node.exe"). Fixed by only
shelling the `npx`-based spawns. Recorded because it is a second instance,
inside this same report, of exactly the pattern this section is about:
a test that can only ever pass would have hidden it.

**What this does NOT establish** — see §6.

## 5. Hostile-geography result (D5)

From `C:\Users\schan\AppData\Local\Temp\claude\...\scratchpad\p1a`
(outside the repo tree):

| command | result |
|---|---|
| `node <absolute-path>/bare-bridge-spike.mjs` (full 45-check run, including layer 3) | **PASS**, 45/45 |
| `npx --prefix <mesh-dir> bare <absolute-path>/bare-bridge-spike.mjs` | **PASS**, 37/37 |

No storage directories or temp files were left behind in the hostile
directory after either run, INCLUDING the `--stdio-worker` storage the
layer-3 spawn leg creates (confirmed by listing the hostile directory AND
`mesh/` itself post-run — the worker's storage always lands in `mesh/`, by
design, since a real DP4 sidecar's data should live with the kit, not
wherever the parent happened to be launched from; the spike wipes it before
and after its own run since ITS invocations are disposable test data, not
a persistent worker's real state). One genuine cleanup gap was caught and
fixed here: the first version of the layer-3 leg left
`mesh/.bare-bridge-worker-storage` behind after a run because nothing
removed it — fixed by wiping it both before (in case a prior run crashed
mid-test) and after the leg completes.

## 6. What was NOT verified

- **A genuinely clean machine** (no prior `npm install` anywhere, no warm
  npx cache) — same caveat as Phase 1a; not this phase's gate (Phase 3's
  hostile-machine rehearsal).
- **The exact race `runStdioWorker`'s `waitIdle()` drain exists for** (a
  request arriving immediately before stdin closes, still in flight when
  'end' fires) — added defensively, not verified against a reproduction;
  no existing repro script has a request/response cycle to exercise it.
- **A truly two-JS-process, wails-launched worker** — this phase's layer 3
  proves the spawn+pipe topology from a Node-based TEST parent; the actual
  Wails/Go parent process (Phase 3/4) is a different language runtime
  entirely and was not simulated here.
- **`fetchAttachment`'s returned `path` field's correctness for
  non-absolute inputs** (§3 deviation #5) — every path in this phase's own
  spike is already workable as given; a caller passing a bare relative
  filename with different CWD assumptions than bridge-server.mjs's
  `path.resolve()`-normalized behavior is untested.
- **`verify-transcript.mjs` under Bare** (§3 deviation #6) — this
  dependency's own migration status is outside this phase's file fence;
  whether it needs to move to `#crypto`/`#apply` or gets a different fix is
  a question for whoever owns it next.
- **Load/concurrency behavior** — no stress test beyond 3 requests per
  spawn leg and 3 back-to-back full-suite re-runs (§4); no measurement of
  many rooms, many concurrent requests queued on one stdio channel, or
  large attachment payloads over stdio-bridged `attach`.
- **Whether the `--stdio-worker` process, left running indefinitely with
  NO stdin activity at all (not even an EOF), stays alive correctly or
  leaks anything** — only ever tested with a bounded pipe-then-EOF
  lifecycle in this phase.

## 7. Files

- `mesh/host/bare-bridge.mjs` — the dispatch core + stdio transport +
  real-stdio worker entry (new).
- `mesh/host/bare-bridge-spike.mjs` — the 45-check (Node) / 37-check (Bare)
  gate, `--stdio-worker` mode, the frame-loop transport proof, and the
  real spawn+pipe leg with its negative control (new).
- `mesh/docs/bare-campaign/PHASE2_BRIDGE_REPORT.md` — this file.

Not touched: `mesh/host/bridge-server.mjs`, `mesh/host/bridge-spike.mjs`,
`mesh/host/apply.mjs`, `mesh/reducer/**`, `mesh/cmd/**`, `mesh/goldens/**`,
`mesh/package.json` (owned this phase by the concurrent packaging
migration — two aliases requested via message, both landed by that
migration, not edited by this coder).
