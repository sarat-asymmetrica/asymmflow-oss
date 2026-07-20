# Phase 1a Report — the WASI-shim path (Bare-runtime campaign)

**Coder:** P1-A · **Date:** 2026-07-20 · **Branch:** `feat/fable-bare-runtime`

## 1. Verdict

**YES.** The unmodified `mesh/dist/reducer.wasm` (built from unmodified
`mesh/cmd/reducer/main.go` + `mesh/reducer/**` — nothing in either was
touched) runs correctly under the Bare runtime via a new Bare-native WASI
preview1 shim, and its output is **byte-for-byte identical** to the real
Node WASI host (`node:wasi`) across every golden scenario in
`mesh/goldens/*.json` — **13/13 scenarios, 0 divergent bytes**, verified live
under Node and verified again under Bare itself (`npx bare`), including from
a directory outside the repo tree.

## 2. Byte-identity results

Run: `npm run bareparity` (Node, live dual-host) then `npm run bareparity:bare`
(Bare, shim vs. pinned Node-host bytes — see §7 on why the two runs check
different things).

| scenario | bytes | Node host vs Bare shim | golden digest check |
|---|---:|---|---|
| inventory_basic | 309 | identical | matches `digest` |
| missionc_autobase | 1100 | identical | matches `stateDigest` |
| missiond_autobase | 696 | identical | matches `stateDigest` |
| room_autobase | 1731 | identical | matches `stateDigest` |
| invite_autobase | 1613 | identical | matches `stateDigest` |
| attach_autobase | 1526 | identical | matches `stateDigest` |
| mirror_autobase | 807 | identical | matches `stateDigest` |
| reissue_autobase (epoch1) | 842 | identical | matches `epoch1StateDigest` |
| reissue_autobase (successor) | 673 | identical | matches `successorStateDigest` |
| reissue_autobase (successor + forgery)¹ | 771 | identical | n/a |
| social_autobase (pre-block) | 1236 | identical | matches `stateDigest` |
| social_autobase (post-block)¹ | 1399 | identical | n/a |
| transcript_autobase | 1517 | identical | matches `stateDigest` |

Byte counts are identical across both runs (the Node run's own live
dual-host comparison, and the separate Bare run checked against the fixture
pinned by that same Node run) — reconfirmed by re-running both back to back
immediately before writing this report (§8.1/§8.2 are the unedited
transcripts of that final pair of runs).

¹ Not tied to a single pinned golden digest field (see §6 on why); included
for extra byte-identity coverage of the capability-rejection path
(reissue's forgery) and the post-block-device fold (social's second stage).

**Both runs: 13 scenario(s) run, 0 failure(s) total. GREEN.**

## 3. Syscall implementation table

All 16 in `mesh/host/wasi-preview1-lite.mjs`. "Real traffic" columns are
measured, not assumed — instrumented by wrapping each import function with a
call counter and running all 13 real scenarios through the shim (transcript
in §8.3).

| syscall | backing | hit in real traffic (13 scenarios) | honesty note |
|---|---|---:|---|
| `args_get` | injected `args: ['reducer']`, encoded via `Buffer` | 13 | — |
| `args_sizes_get` | same | 13 | — |
| `environ_get` | injected `env: {}` | **0** | never called — Go's runtime apparently skips the data call when `environ_sizes_get` reports zero environ entries; implemented per spec but unexercised by this suite |
| `environ_sizes_get` | same | 13 | — |
| `clock_time_get` | `Date.now()` | 100 | `id`/`precision` ignored — every clock this process could ask for collapses to wall time (see file header); called ~8×/run, not observably load-bearing to the fold's output (no clock value appears in any state) |
| `fd_close` | shim-local fd-table deletion | **0** | never called — the reducer's `io.ReadAll`/`os.Stdout.Write` contract (mesh/cmd/reducer/main.go) never explicitly closes its fds before `proc_exit` |
| `fd_fdstat_get` | shim-local (filetype=character_device, flags from shim state) | 78 | — |
| `fd_fdstat_set_flags` | **no bare-fs equivalent** — pure synthesized shim state (P0-A §5, confirmed) | 39 | real traffic, not hypothetical — Go's runtime does call this on fd 0/1 during setup; the shim's answer (accept, store, never fail) was sufficient for every real run |
| `fd_prestat_dir_name` | **no bare-fs equivalent** — WASI-only preopen concept (P0-A §5) | **0** | never called — `fd_prestat_get`'s EBADF on the first probed fd (below) stops the probe before this would ever run |
| `fd_prestat_get` | **no bare-fs equivalent** — WASI-only preopen concept (P0-A §5); honest EBADF for every fd (no preopens exist) | 13 | called exactly once per run, on fd 3 — confirms the "probe until EBADF, then stop" behavior is real, not assumed |
| `fd_read` | injected `read(dst)` closure over an in-memory input Buffer | 77 | — |
| `fd_write` | injected `write(src)` closure over an in-memory output Buffer | 13 | — |
| `poll_oneoff` | **honest simplification: never blocks** — every subscription resolves as "ready" on the same call (clock: timeout already elapsed; fd: ≥1 byte available) — see file header for the full reasoning | **0** | never called by any of the 13 real scenarios, including the largest (1.7KB output, 13 signed ops with Ed25519 verification). P0-D flagged this "TO VERIFY" — verified: not exercised at all at this scale. **Not proven safe at larger scale or under real concurrency/threads** — flagging honestly per §6 |
| `proc_exit` | throws a `WASIExit` sentinel the host catches (same unwind pattern node:wasi's own C++ binding uses one layer down) | 13 | every run ends by hitting this (exit code 0) |
| `random_get` | `Math.random()` by default (injectable `random(n)` override) — see file header for why Go's map-hash-seed use makes cryptographic strength irrelevant to the fold's byte-identity, verified by grep that no non-test reducer file imports `crypto/rand`/`math/rand` | 13 | — |
| `sched_yield` | no-op, returns success | **0** | never called — no wasm threads/goroutine contention in this single-shot batch shape |

## 4. Hostile-geography map (D5)

Ran from `C:\Users\schan\AppData\Local\Temp\claude\...\scratchpad\p1a`
(outside the repo tree, no `node_modules` anywhere in that directory's
ancestor chain) against the real script by absolute path:

| command | result | why |
|---|---|---|
| `node <absolute-path>/bare-parity-spike.mjs` | **PASS**, 13/13 green | Node resolves `node_modules` relative to the SCRIPT's own directory tree (mesh/node_modules), not CWD — the FR-1 disease class (PATH/CWD-relative resolution) never triggers here because every path in wasi-preview1-lite.mjs/apply-bare.mjs/bare-parity-spike.mjs is built from `import.meta.url`, never `process.cwd()` |
| `npx --prefix <mesh-dir> bare <absolute-path>/bare-parity-spike.mjs` | **PASS**, 13/13 green | `--prefix` explicitly points npx at mesh's own `node_modules/.bin/bare` — the "polite" hostile-geography case |
| `npx bare <absolute-path>/bare-parity-spike.mjs` (no `--prefix`, run from the hostile CWD) | **PASS**, 13/13 green — unexpected, noted as a positive finding | npx apparently resolves `bare` via its own global npx package cache rather than requiring a `node_modules/bare` in the CWD's ancestor chain. **Not fully characterized**: this depends on npm's local cache already being warm on this machine (from earlier `npm i -D bare` in mesh/) — a genuinely clean machine (Phase 3's hostile-machine rehearsal, not this phase's job) might behave differently. Flagging as unverified, not claiming it as a packaging answer. |
| plain `bare` / `which bare` from the hostile CWD | **FAIL** (`command not found`) | expected — `bare` is a project devDependency, never installed globally; this is the exact failure a truly bare (pun intended) end-user machine would hit, which is precisely why Phase 3 exists (a sealed artifact, not "run npx and hope") |

No filesystem writes, temp-file races, or PATH lookups occur anywhere in the
shim, the Bare host, or the parity spike itself — everything is either
`import.meta.url`-relative or fully in-memory (see §5).

## 5. Design notes worth the gate's attention

- **No filesystem at all, on either side of the channel.** `apply-bare.mjs`
  backs stdin/stdout with in-memory `Buffer`s, not temp files — strictly
  better than `apply.mjs`'s own temp-file-per-call design (which exists
  there only because `node:wasi`'s public API wants real fds). Proven
  working, not assumed.
- **`Buffer` is the one universal global.** Verified empirically (not
  assumed) that `Buffer` is present, unimported, under both Node and Bare,
  while `crypto`/`TextEncoder` are Node-only and undefined under Bare
  (`npx bare -e "typeof crypto"` → `undefined`). The shim is built entirely
  on `Buffer`, which is what makes one shim body correct under both
  runtimes without any `node:`/`bare-*` import inside
  `wasi-preview1-lite.mjs` itself.
- **`WebAssembly.Instance`/`WebAssembly.Module` (sync constructors), not
  `WebAssembly.instantiate`/`compile` (async).** This keeps
  `apply-bare.mjs`'s `applyViaWasm`/`applyViaWasmRaw` fully synchronous,
  matching `apply.mjs`'s own exported signature exactly (no Promise
  wrapping) — important because `mesh-node.mjs`'s `state()` and other
  callers invoke `applyViaWasm` without `await`.

## 6. What this does NOT establish (honest limits)

- **`poll_oneoff`'s "never blocks" simplification is unexercised at scale**
  (§3) — it was never called even once across the 13 real scenarios,
  including the largest. A future reducer shape that legitimately needs to
  sleep or wait on real I/O readiness would need this revisited; nothing in
  the current suite proves or disproves that case.
- **`environ_get`, `fd_close`, `fd_prestat_dir_name`, `sched_yield` are
  implemented but never hit by real traffic** in this suite (§3) — correct
  per spec reasoning, not battle-tested.
- **The `npx bare` hostile-CWD success (§4) is not characterized against a
  genuinely clean machine** — Phase 3's hostile-machine rehearsal is the
  actual gate for that; this phase only proves the shim's OWN code has no
  CWD-relative resolution, not that the Bare *toolchain* is available
  without any prior `npm install` anywhere on the machine.
- **No performance/overhead measurement** between the Node WASI host and
  the Bare shim host — out of this report's scope; the Phase 1 decision
  memo (comparing this path against Phase 1b's `go:wasmexport` reactor
  path) is a separate deliverable.
- **The two golden-scenario reconstructions that needed a second look
  (`reissue_autobase` successor, `social_autobase` pre-block) required
  correcting my own transcription** (an em-dash flattened to `--`, a
  dropped ☕/🍃 emoji, and capturing the reissue successor's state snapshot
  BEFORE the rogue-forgery op instead of after, matching exactly where
  `reissue-spike.mjs` itself takes that snapshot) — documented here in the
  interest of showing the actual debugging path, not just the clean
  result. Verified against a from-scratch reproduction script (diffed
  op-for-op against a faithful copy of `reissue-spike.mjs`'s own
  `runScenario`) before concluding the fix, not guessed.
- **Only one Windows machine, one Bare version (1.30.3), one Node version
  (22.17.0)** — no cross-version or cross-OS matrix was run.

## 7. Packages added

**None.** Everything needed (`bare`, `bare-fs`, `bare-crypto`, `bare-events`,
`bare-process`, `bare-stream`) was already a devDependency from P0-A's
spike. `bare-os`/`bare-path`/`bare-subprocess` — flagged by P0-A as
possibly-needed — turned out unnecessary: `Buffer` + `new URL(...,
import.meta.url)` covered every path/byte need `wasi-preview1-lite.mjs` and
`apply-bare.mjs` had. `package.json` gained two script entries only
(`bareparity`, `bareparity:bare`), appended, nothing reordered or removed.

## 8. Files

- `mesh/host/wasi-preview1-lite.mjs` — the shim (dual-runtime, zero
  `node:`/`bare-*` imports, `Buffer`-only).
- `mesh/host/apply-bare.mjs` — the Bare-runtime host, same channel contract
  as `apply.mjs`, in-memory fds, exports both `applyViaWasm` (parsed) and
  `applyViaWasmRaw` (raw bytes, for byte-identity comparison).
- `mesh/host/bare-parity-spike.mjs` — the proof: 13 real scenarios (op sets
  transcribed from the campaign's own spike files), live dual-host
  byte-identity under Node, fixture-checked byte-identity under Bare,
  plus a golden-digest cross-check.
- `mesh/host/bare-parity-fixtures.json` — generated artifact (by the Node
  run), consumed by the Bare-only run; not hand-written.
- `mesh/package.json` — `+2` script entries (`bareparity`,
  `bareparity:bare`), appended only.
- `mesh/docs/bare-campaign/PHASE1A_REPORT.md` — this file.

Not touched: `mesh/reducer/**`, `mesh/cmd/reducer/**`, `mesh/host/apply.mjs`,
`mesh/goldens/**`, `mesh/host/apply-reactor.mjs`,
`mesh/host/reactor-parity-spike.mjs`, `mesh/cmd/reducer-reactor/**`.

### 8.1 Raw transcript — Node run (`node host/bare-parity-spike.mjs`, the
final re-run immediately before writing this report; ExperimentalWarning
noise stripped, nothing else edited)

```
bare-parity-spike -- Node WASI host vs Bare shim host, byte-for-byte, every golden [runtime: Node]


-- inventory_basic --
   expected (Node host): 309B
   got (Bare shim):      309B
   PASS - byte-identical
   golden check: matches goldens/inventory_basic.json's digest (6c8c35eff1e2c04d...)

-- missionc_autobase --
   expected (Node host): 1100B
   got (Bare shim):      1100B
   PASS - byte-identical
   golden check: matches goldens/missionc_autobase.json's stateDigest (79432ed8c16c9898...)

-- missiond_autobase --
   expected (Node host): 696B
   got (Bare shim):      696B
   PASS - byte-identical
   golden check: matches goldens/missiond_autobase.json's stateDigest (06d37b35844e5085...)

-- room_autobase --
   expected (Node host): 1731B
   got (Bare shim):      1731B
   PASS - byte-identical
   golden check: matches goldens/room_autobase.json's stateDigest (74523010325c2be8...)

-- invite_autobase --
   expected (Node host): 1613B
   got (Bare shim):      1613B
   PASS - byte-identical
   golden check: matches goldens/invite_autobase.json's stateDigest (a3d679e22cd5393b...)

-- attach_autobase --
   expected (Node host): 1526B
   got (Bare shim):      1526B
   PASS - byte-identical
   golden check: matches goldens/attach_autobase.json's stateDigest (ec23eb92ca445a11...)

-- mirror_autobase --
   expected (Node host): 807B
   got (Bare shim):      807B
   PASS - byte-identical
   golden check: matches goldens/mirror_autobase.json's stateDigest (550fbb75660a86e9...)

-- reissue_autobase_epoch1 --
   expected (Node host): 842B
   got (Bare shim):      842B
   PASS - byte-identical
   golden check: matches goldens/reissue_autobase.json's epoch1StateDigest (e3b1a32b52ec90e8...)

-- reissue_autobase_successor --
   expected (Node host): 673B
   got (Bare shim):      673B
   PASS - byte-identical
   golden check: matches goldens/reissue_autobase.json's successorStateDigest (7da5d00c4817204b...)

-- reissue_autobase_successor_with_forgery --
   expected (Node host): 771B
   got (Bare shim):      771B
   PASS - byte-identical

-- social_autobase_preblock --
   expected (Node host): 1236B
   got (Bare shim):      1236B
   PASS - byte-identical
   golden check: matches goldens/social_autobase.json's stateDigest (17209aecd430c151...)

-- social_autobase_postblock --
   expected (Node host): 1399B
   got (Bare shim):      1399B
   PASS - byte-identical

-- transcript_autobase --
   expected (Node host): 1517B
   got (Bare shim):      1517B
   PASS - byte-identical
   golden check: matches goldens/transcript_autobase.json's stateDigest (1a1aed0082f0113b...)

(fixtures written for the Bare-only run: /C:/Projects/asymmflow/asymmflow-oss/mesh/host/bare-parity-fixtures.json)

13 scenario(s) run, 0 failure(s) total.

BARE PARITY SPIKE GREEN -- the unmodified reducer folds byte-identically under Bare
```

### 8.2 Raw transcript — Bare run (`npx bare host/bare-parity-spike.mjs`,
same final re-run pass, checked against the fixture the Node run above just
wrote)

```
bare-parity-spike -- Node WASI host vs Bare shim host, byte-for-byte, every golden [runtime: Bare]

[... identical per-scenario PASS + golden-check lines as §8.1, byte-for-byte
the same sizes and digest prefixes for every one of the 13 scenarios —
omitted here to avoid duplicating the block verbatim ...]

13 scenario(s) run, 0 failure(s) total.

BARE PARITY SPIKE GREEN -- the unmodified reducer folds byte-identically under Bare
```

The hostile-geography runs (§4) produced this exact same tail
(`13 scenario(s) run, 0 failure(s) total.` / `GREEN`) from both
`node <absolute-path>` and `npx bare <absolute-path>` invoked from
`C:\Users\schan\AppData\Local\Temp\claude\...\scratchpad\p1a`.

### 8.3 Syscall hit-count instrumentation (source for §3's counts)

Ad hoc counter wrapping every `wasi_snapshot_preview1` import function,
run once over all 13 fixture scenarios:

```
aggregate syscall hit counts across all 13 scenarios: {
  "clock_time_get": 100, "random_get": 13, "args_sizes_get": 13,
  "args_get": 13, "environ_sizes_get": 13, "fd_fdstat_get": 78,
  "fd_fdstat_set_flags": 39, "fd_prestat_get": 13, "fd_read": 77,
  "fd_write": 13, "proc_exit": 13
}
```
(`environ_get`, `fd_close`, `fd_prestat_dir_name`, `poll_oneoff`,
`sched_yield` absent from this object — zero calls, confirmed by the
counter never incrementing them, not by their absence from a hand-written
list.)
