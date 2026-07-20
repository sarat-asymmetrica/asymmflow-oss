# Phase 2 Report — the Bare bridge (DP4 stdio seam)

**Coder:** P1-A · **Date:** 2026-07-20 · **Branch:** `feat/fable-bare-runtime`

## 1. Verdict

**YES.** Protocol v0 (`mesh/docs/MESSENGER_UI_CAMPAIGN.md` §1) works over
ndjson-framed stdio under Bare — a new dispatch core (`bare-bridge.mjs`)
built by re-deriving bridge-server.mjs's own logic from the SAME underlying
host modules (mesh-node.mjs, capability.mjs, social-room.mjs,
invite-code.mjs, attachments.mjs, export-transcript.mjs — reused, never
reimplemented), fed through a real ndjson stdio transport, passes the same
30-scenario behavioral suite bridge-spike.mjs gates on (38 checks here, one
extra layer added — see §2) under **both** `node` and `npx bare`, including
from outside the repo tree. The Node bridge line (`bridge-server.mjs` /
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

## 2. 30-check results, both runtimes

`bare-bridge-spike.mjs` runs a superset of bridge-spike.mjs's 30 checks:
the same scenario (anchored PO room, hub+desk, malformed-frame resilience,
listRooms/roomState, post + expectation validation + urgency float,
claimRoom/releaseClaim, attach→fetchAttachment sha256 round-trip,
exportTranscript verified via verify-transcript.mjs, room-updated events
(self + replicated), GL-5 seq-continuation across a bridge-core restart,
social room + real invite redeem under encryption, the claim-skip proof in
a social room) **plus** a 7-check "frame loop" layer (§3, layer 2) that
exercises the real ndjson buffering code path directly (split lines,
malformed lines, blank lines, multiple frames in one chunk) — something
bridge-spike.mjs's TCP-socket version doesn't need a separate layer for,
because `net.Socket`'s own buffering already gets exercised implicitly by
real localhost traffic; this file's stdio transport is new code, so it gets
its own explicit proof.

| Run | Checks | Failures | Result |
|---|---:|---:|---|
| `node host/bare-bridge-spike.mjs` | 38 | 0 | **GREEN** |
| `npx bare host/bare-bridge-spike.mjs` | 37 | 0 | **GREEN** (1 check downgraded to a logged skip — see §3) |
| `node host/bridge-spike.mjs` (frozen Node/TCP line) | 30 | 0 | **GREEN**, unchanged |
| `npm run bareparity` (Phase 1a suite) | 13 scenarios | 0 | **GREEN** |
| `npm run bareparity:bare` | 13 scenarios | 0 | **GREEN** |
| `npm run smoke` | — | 0 | **GREEN** |
| `node host/reactor-parity-spike.mjs` (Phase 1b suite, another coder's file, read-only here) | 13 scenarios | 0 | **GREEN** |

Both bare-bridge-spike.mjs runs also succeeded from
`C:\Users\schan\AppData\Local\Temp\claude\...\scratchpad\p1a` (outside the
repo tree, absolute script path, hostile CWD) — see §5.

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
   that one check (of 38) is skipped with a clear stderr note rather than
   crashing the whole spike at module load. `exportTranscript` itself
   (the bridge method) works fine under Bare — only the SEPARATE,
   independent-of-this-phase verification helper does not load.
7. **`Number`/`String`/`Buffer`-only op literals**, same canon as Phase 1a
   — no behavioral deviation, just consistency with the established
   dual-runtime discipline.

## 4. The flush race (P0-D's finding) — what was done, what was observed

**Design mitigation (in `bare-bridge.mjs`, see its header comment for the
full reasoning), three concrete choices:**

1. `runStdioWorker` never calls `proc.exit()`/`Bare.exit()` — the worker
   stays referenced on the event loop via its own stdin listener until the
   caller tears it down, the structural opposite of P0-D's flaky repro
   shape (compile → synchronous log loop → fall off the end of the script).
2. The wasm module compiles **lazily**, on the first `apply()` call inside
   `mesh-node.mjs`'s `state()` — not eagerly at worker boot — pushing the
   compile-then-write race window past process startup, into a point where
   the process is provably still running an active event loop.
3. `attachStdioTransport`'s `pendingPartial()` plus per-response `id`
   echoing make a dropped/truncated frame **detectable** from the reader
   side (a request whose id never gets a matching response is a signal),
   even though protocol v0 has no retry/ack layer to recover one — adding
   one would be a protocol change, out of this phase's authority (D6).

**Empirical investigation (today, this machine, `bare 1.30.3`):**

- Corrected re-run of P0-D's own flaky script
  (`host/bare-spike/wasm-compile-check.mjs`), 10 back-to-back runs under
  `npx bare`: **0/10 truncated** (full 4-line output every time). Note: my
  FIRST attempt at this check used a wrong grep pattern and misreported
  10/10 "incomplete" — caught and corrected before writing this section;
  recorded honestly rather than silently fixed, per the campaign's own
  transparency norm (see `PHASE0_GATE_B3_CONDITION_MAP.md`'s "the control
  you skip is the one that was hiding the defect").
- `--stdio-worker` mode, one request per invocation, piped over a real OS
  pipe (`echo '{...}' | npx bare host/bare-bridge-spike.mjs --stdio-worker`),
  **20/20 clean, single-line, valid-JSON responses.**
- `--stdio-worker` mode, three requests per invocation (including
  `createSocialRoom`, which touches the reducer via `state()` after
  appending), **15/15 clean.**
- **Net: 45 sampled worker runs + 10 corrected re-samples of P0-D's own
  script, 0 truncations observed today**, against P0-D's originally
  reported ~25-30% rate on `wasm-compile-check.mjs`/`wasi-imports-list.mjs`
  specifically.

**What this does NOT establish:** P0-D's finding is not retracted by this —
a 0/N sample does not prove a ~25% intermittent race is absent (P0-D's own
script, tested with the SAME corrected method, also showed 0/10 today,
which is itself notable — either genuine day-to-day environment variance,
or the race's trigger window is narrower/different than the original
sample suggested). This phase's own worker was **never observed to
truncate**, but "never observed in 45 runs" is evidence of absence at a
specific confidence level, not proof of absence — a client integrating this
bridge for real (Phase 3+) should still build the request/response `id`
timeout-and-retry the DP4 sidecar contract will eventually need, treating
this as an open risk, not a closed one.

## 5. Hostile-geography result (D5)

From `C:\Users\schan\AppData\Local\Temp\claude\...\scratchpad\p1a`
(outside the repo tree):

| command | result |
|---|---|
| `node <absolute-path>/bare-bridge-spike.mjs` | **PASS**, 38/38 |
| `npx --prefix <mesh-dir> bare <absolute-path>/bare-bridge-spike.mjs` | **PASS**, 37/37 |

No storage directories or temp files were left behind in the hostile
directory after either run (spike's own cleanup ran; confirmed by listing
the directory post-run). Same underlying property as Phase 1a: every path
in `bare-bridge.mjs`/`bare-bridge-spike.mjs` is either relative-to-CWD by
explicit design (the spike's own `tmp` storage dirs — intentionally, so a
real DP4 sidecar's working directory is where its data lands, not a
hardcoded repo-relative path) or resolved via `import.meta.url` for the
module's own asset needs (none, in this phase — no wasm/goldens touched
directly by these two files).

## 6. What was NOT verified

- **A genuinely clean machine** (no prior `npm install` anywhere, no warm
  npx cache) — same caveat as Phase 1a; not this phase's gate (Phase 3's
  hostile-machine rehearsal).
- **Real two-process stdio** (an actual parent process spawning a bare
  worker over real OS pipes with the PARENT also being one of this
  phase's own files) — not achievable without a `node:child_process`
  or Bare-only `bare-subprocess` import inside "my files" (explicitly
  forbidden). The flush-race investigation (§4) instead drove the worker
  from the SHELL (this coder's gate-running process, not committed
  source) via real pipes — genuine OS-level bytes, just not spawned from
  JS. A real two-JS-process proof is Phase 3/4 territory (the sealed kit
  actually launches a bare worker as a child of the wails app).
  This means bare-bridge.mjs's own end-to-end behavior when driven by a
  REAL wails-side stdio client is inferred from the frame-loop layer
  (§2, `frameLoopCheck`) plus the shell-piped worker-mode smoke tests
  (§4), not from a single committed automated test that does both at
  once.
- **`fetchAttachment`'s returned `path` field's correctness for
  non-absolute inputs** (§3 deviation #5) — every path in this phase's own
  spike is already workable as given; a caller passing a bare relative
  filename with different CWD assumptions than bridge-server.mjs's
  `path.resolve()`-normalized behavior is untested.
- **`verify-transcript.mjs` under Bare** (§3 deviation #6) — this
  dependency's own migration status is outside this phase's file fence;
  whether it needs to move to `#crypto`/`#apply` or gets a different fix is
  a question for whoever owns it next.
- **Load/concurrency behavior** — no stress test beyond the 45-run
  flush-race sampling (§4); no measurement of many rooms, many concurrent
  requests queued on one stdio channel, or large attachment payloads over
  stdio-bridged `attach`.
- **Whether the `--stdio-worker` process, left running indefinitely with
  NO stdin activity at all (not even an EOF), stays alive correctly or
  leaks anything** — only ever tested with a bounded pipe-then-EOF
  lifecycle in this phase.

## 7. Files

- `mesh/host/bare-bridge.mjs` — the dispatch core + stdio transport +
  real-stdio worker entry (new).
- `mesh/host/bare-bridge-spike.mjs` — the 38-check gate, `--stdio-worker`
  mode, and the frame-loop transport proof (new).
- `mesh/docs/bare-campaign/PHASE2_BRIDGE_REPORT.md` — this file.

Not touched: `mesh/host/bridge-server.mjs`, `mesh/host/bridge-spike.mjs`,
`mesh/host/apply.mjs`, `mesh/reducer/**`, `mesh/cmd/**`, `mesh/goldens/**`,
`mesh/package.json` (owned this phase by the concurrent packaging
migration — two aliases requested via message, both landed by that
migration, not edited by this coder).
