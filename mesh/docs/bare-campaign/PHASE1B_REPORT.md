# Phase 1b — the real reducer reactor (2026-07-20)

**Coder:** P0-C · **Campaign:** Bare-runtime, `feat/fable-bare-runtime` · **Ruling:** R1
(`mesh/docs/bare-campaign/OWNER_RULINGS.md`)

## Headline

**Real import count: 15 distinct `wasi_snapshot_preview1` functions — not 10.**
Only ONE import is genuinely eliminated by the reactor path (`fd_read`), not six.
The Phase 0 gate-verification's "10-syscall shim with no I/O at all" framing
(`PHASE0_GATE_C_VERIFICATION.md`'s "Consequence for Phase 1" section) does **not**
hold for the real reducer — it held only for the trivial `add()` probe function
used to verify the premise. This report corrects the record with the real
measurement, as that document itself flagged as still-open: *"the reactor build
tested here is a trivial `add()` function… Phase 1b must re-enumerate against the
REAL reducer and this document's 10 must not be treated as final until it does."*

**Byte-identity verdict: PASS, 13/13 scenarios, all 11 goldens, 0 divergent bytes.**
Owner ruling R1's condition 3 is satisfied. Conditions 1 and 2 also hold:
`mesh/reducer/**` was not edited (verified: no Edit/Write call touched it this
phase; `go test ./mesh/reducer/...` reports `ok … (cached)` — unchanged), and the
command module still builds clean and its own spike (`smoke.mjs`) is still green.

## Task 1b-1 — the real reactor

New file: `mesh/cmd/reducer-reactor/main.go` (`//go:build wasip1`). Three exports —
`malloc(size) -> ptr`, `apply(ptr,len) -> fatPtr` (packed `(outPtr<<32)|outLen`,
since `string` cannot be a `go:wasmexport` RESULT type — Phase 0 §3), `free(ptr,
size)` — plus the compiler-generated `_initialize`. `apply()` calls the identical
`reducer.ApplyWithConfig` / `reducer.ApplyRoom` functions the command module calls,
with the identical `input{Mode,Config,Ops}` struct, in the identical two-case mode
switch (mechanically comparable side-by-side against `mesh/cmd/reducer/main.go`).
The full ABI (why three exports, the malloc/pin/free lifecycle, the fat-pointer
return encoding) is documented in the file's own header comment, at the same
density as `../reducer/main.go`'s.

`go build -buildmode=c-shared` for `GOOS=wasip1 GOARCH=wasm` succeeded cleanly, no
warnings. `go build ./mesh/...` (whole package tree) also stays green.

## Task 1b-2 — the decisive measurement

```
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o mesh/dist/reducer-reactor.wasm ./mesh/cmd/reducer-reactor
```

Enumerated with `WebAssembly.Module.imports()` under Node v22.17.0, deduplicated to
distinct names:

```
REDUCER-REACTOR (go:wasmexport, real reducer): 17 total import entries, 15 DISTINCT
  wasi_snapshot_preview1:
    args_get, args_sizes_get, clock_time_get, environ_get, environ_sizes_get,
    fd_close, fd_fdstat_get, fd_fdstat_set_flags, fd_prestat_dir_name,
    fd_prestat_get, fd_write, poll_oneoff, proc_exit, random_get, sched_yield
  exports: _initialize, malloc, free, apply, memory

COMMAND MODULE (existing rollback path, unmodified): 18 total import entries, 16 DISTINCT
  wasi_snapshot_preview1:
    args_get, args_sizes_get, clock_time_get, environ_get, environ_sizes_get,
    fd_close, fd_fdstat_get, fd_fdstat_set_flags, fd_prestat_dir_name,
    fd_prestat_get, fd_read, fd_write, poll_oneoff, proc_exit, random_get,
    sched_yield
  exports: _start, memory
```

**Diff: exactly one function, `fd_read`.** Every other name is shared. The reactor
does NOT eliminate `fd_close`, `fd_fdstat_get`, `fd_fdstat_set_flags`,
`fd_prestat_get`, `fd_prestat_dir_name` — the five P0-A flagged as having "no
bare-fs backing" and the earlier gate-verification's decision-matrix framing
assumed the reactor would shed. It sheds only `fd_read`.

### Why — traced, not guessed

`mesh/reducer` itself imports no filesystem/OS packages directly (`grep` over
`mesh/reducer/*.go` confirms: `capability.go` imports `crypto/ed25519`,
`crypto/sha256`, `encoding/hex`, `strconv`; `kernel_domains.go` imports `time`;
`reducer.go`/`room_domain.go` have their own small stdlib sets — none of them
`os`, `fmt`, or `log`). But `go list -deps` on `GOOS=wasip1 GOARCH=wasm` shows
`os` (and `internal/poll`, `syscall`) as a **transitive** dependency of the
reducer's own imports:

```
encoding/json  -> os? YES
crypto/ed25519 -> os? YES
crypto/sha256  -> os? YES
encoding/hex   -> os? YES
strconv        -> os? no
time           -> os? no
sync           -> os? no
unsafe         -> os? no
```

Once any dependency in the build graph pulls in `os`, the linker keeps `os`'s
package-level init — which on wasip1 unconditionally sets up the `os.Stdin` /
`os.Stdout` / `os.Stderr` file table (`fd_fdstat_get` to probe each fd's type,
`fd_prestat_get`/`fd_prestat_dir_name` as part of the preopen scan, `fd_close` in
the table's lifecycle) — regardless of whether the program ever calls
`os.Open`/reads a file. This is why the trivial `add(a,b int32) int32` probe
(Phase 0, and the orchestrator's independent gate re-check) measured only 10: it
imports nothing that pulls in `os` at all, so that whole init path is dead-code-
eliminated. The real reducer imports `encoding/json` (for `Marshal`/`Unmarshal`)
and `crypto/ed25519`/`crypto/sha256`/`encoding/hex` (Mission D's capability
verification) — every one of which transitively drags `os` in — so that init path
survives linking and its five WASI calls appear in the binary regardless of
`go:wasmexport` vs. the command module's `func main`.

`fd_read` is the one call that genuinely depends on the ENTRY-POINT CHOICE rather
than the dependency graph: the command module explicitly calls
`io.ReadAll(os.Stdin)` (`mesh/cmd/reducer/main.go:36`) to get its input bytes; the
reactor's `apply(ptr,len)` never reads stdin at all — its input arrives via the
malloc/host-write channel instead. That is the one real, mechanism-level
elimination this packaging change buys.

### Consequence for the campaign's decision matrix

`PHASE0_GATE_C_VERIFICATION.md`'s reframing — "16-syscall shim with a real fd
layer (command module) vs. 10-syscall shim with no I/O at all (reactor)" — is
**superseded** by this measurement. The real comparison is:

> **16-import shim (command module)** vs. **15-import shim (reactor)** — a
> reduction of ONE function (`fd_read`), not six. The reactor still needs every
> fd-table/preopen call P0-A found "no bare-fs backing" for.

This does not mean the reactor path is worthless — it still buys the warm-instance
call pattern (no per-call process/memory teardown), drops the temp-file-per-call
channel `apply.mjs` currently uses, and is one real import lighter — but it is NOT
the "shim touches no filesystem whatsoever" result the earlier document projected.
Whoever owns the Bare-side WASI shim (P0-A/B) needs to budget for **15** WASI
functions on the reactor path, including all five of the ones flagged as having no
`bare-fs` equivalent, not 10.

## Task 1b-3 — host channel + byte-identity proof

New file `mesh/host/apply-reactor.mjs`: instantiates `reducer-reactor.wasm` once,
calls Node's `wasi.initialize()` (the documented reactor counterpart of
`wasi.start()` for command modules) to run `_initialize`, then drives
malloc/apply/free directly per call — no temp files, no `node:fs` writes, no
`node:os` tmpdir. Exports `applyViaWasm()` (parsed object, signature-compatible
with `apply.mjs`'s own export) and `applyViaWasmRaw()` (undecoded output `Buffer`,
needed for true byte comparison — see below).

New file `mesh/host/reactor-parity-spike.mjs`. **Scope, stated honestly up front
in the file's own header**: it drives the reducer boundary directly — the same
`(ops, config, mode)` triple `mesh-node.mjs:160`'s `state()` hands to
`applyViaWasm` — through both channels and diffs the raw bytes. It does **not**
re-run the real spikes' Autobase/Hyperswarm/BlindPeer replication machinery. That
scoping rests on a verified fact, not a shortcut: `mesh/reducer/reducer.go:249`
(`sort.SliceStable(sorted, canonicalLess)`) sorts every op into canonical order
**inside** the reducer before folding, so the reducer's output depends only on the
SET of ops it receives, never the order the host handed them in — a property
`room-spike.mjs`'s own comments already rely on ("canonical order sorts primarily
by Seq, actor only breaks ties"). This was spot-verified, not just cited: for the
reissue scenario, a throwaway debug run reproduced the REAL replicated flow
(separate desk node, `successor.connect(deskNode)`, `addWriter`, actual P2P
replication) and diffed its op array byte-for-byte against this spike's simplified
single-node reconstruction (desk's op appended directly onto the successor's own
core, no second node) — **identical**, confirming that which physical writer core
carries an op never matters to reducer output, only the op's own signed content
does.

Every op set in the spike is transcribed from the corresponding source file
(`smoke.mjs`, `missionc-mesh.mjs`, `missiond-mesh.mjs`, `room-spike.mjs`,
`invite-spike.mjs`, `attach-spike.mjs`, `mirror-spike.mjs`, `reissue-spike.mjs`,
`social-spike.mjs`, `transcript-spike.mjs`) — pinned device seeds, op literals, and
helper calls (`capability.mjs`, `invite-code.mjs`, `reissue-room.mjs`,
`social-room.mjs`, `attachments.mjs`) copied, not re-derived. Where a golden pins a
`stateDigest`/`digest`, the spike separately checks the command module's own output
against that pinned value — independent proof the reconstruction is the right op
set, not just that the two wasm modules happen to agree with each other on
arbitrary input.

### Result (13 scenarios, covering all 11 golden files)

```
inventory_basic                              ✓ byte-identical  ✓ golden sanity
missionc_autobase                            ✓ byte-identical  ✓ golden sanity
missiond_autobase                            ✓ byte-identical  ✓ golden sanity
room_autobase                                ✓ byte-identical  ✓ golden sanity
invite_autobase                              ✓ byte-identical  ✓ golden sanity
attach_autobase                              ✓ byte-identical  ✓ golden sanity
mirror_autobase                              ✓ byte-identical  ✓ golden sanity
reissue_autobase (epoch1)                    ✓ byte-identical  ✓ golden sanity
reissue_autobase (successor, at desk join)   ✓ byte-identical  ✓ golden sanity
reissue_autobase (successor, at forgery)     ✓ byte-identical  (no golden pinned at this point — see below)
social_autobase (pre-block)                  ✓ byte-identical  ✓ golden sanity
social_autobase (post-block)                 ✓ byte-identical  ✓ golden sanity (opsHashed/applied/skipped/rejected)
transcript_autobase (hub)                    ✓ byte-identical  ✓ golden sanity

13 scenario(s) run, 0 check failure(s) total.
REACTOR PARITY GREEN ✅
```

`reissue_autobase (successor, at forgery)` has no golden-digest check because
`reissue-spike.mjs` itself never pins a digest at that point (rogue's forgery is
asserted only by rejection behavior, not a digest) — command-vs-reactor
byte-identity was still checked and passed.

### One reconciliation, reported honestly (not papered over)

The first run of this spike had **one** failure — but it was a golden-sanity
failure in this spike's own reconstruction, not a command-vs-reactor byte
divergence (command and reactor already agreed with each other, byte-for-byte, on
that run too). Root cause: `reissue-spike.mjs`'s own `runScenario()` returns
`successorStateAfterDesk.digest` (3 ops: manifest, grant, desk's message) as the
golden's `successorStateDigest` — captured BEFORE rogue's forgery op is appended,
even though the forgery is appended earlier in the script for its own
rejection-behavior assertions. This spike's first draft fed the golden check the
4-op (post-forgery) state instead of the 3-op (at-desk-join) state. Fixed by
splitting into two scenarios (documented above): `(successor, at desk join)` now
checked against the golden, `(successor, at forgery)` checked command-vs-reactor
only. Confirmed via `node mesh/host/reissue-spike.mjs` (the real, unmodified
spike) that the 3-op digest is what's actually pinned, and via the byte-for-byte
op-array diff described above that this spike's op reconstruction was correct all
along — the bug was in which fold-point this spike compared, not in the ops
reconstructed or in either wasm module.

## Owner ruling R1 — condition-by-condition

1. **`mesh/reducer/**` untouched**: confirmed — no `Edit`/`Write` call touched any
   file under it this phase; `go test ./mesh/reducer/...` reports cached/unchanged.
2. **Command module keeps building, its spike stays green**: confirmed —
   `GOOS=wasip1 GOARCH=wasm go build -o mesh/dist/reducer.wasm ./mesh/cmd/reducer`
   exits 0; `node mesh/host/smoke.mjs` — `SMOKE GREEN ✅`, same digest as before
   (`6c8c35eff1e2c04d…`).
3. **Every golden vector folds byte-identical through the new channel**:
   confirmed — 13/13 scenarios, 0 divergent bytes, across all 11 golden files (see
   above).

**All three conditions hold. Owner ruling R1 is satisfied for Phase 1b.**

## Files touched this phase

- `mesh/cmd/reducer-reactor/main.go` (new)
- `mesh/host/apply-reactor.mjs` (new)
- `mesh/host/reactor-parity-spike.mjs` (new)
- `mesh/docs/bare-campaign/PHASE1B_REPORT.md` (this file, new)
- `mesh/dist/reducer-reactor.wasm`, `mesh/dist/reducer.wasm` — generated build
  artifacts (not committed; same status as the existing `mesh/dist/reducer.wasm`
  convention)
- No changes to `mesh/package.json` were needed — the spike runs directly via
  `node mesh/host/reactor-parity-spike.mjs`, matching every other `*-spike.mjs`'s
  own npm-script convention if the orchestrator wants one added later, but this
  phase didn't require it to prove the three ruling conditions.
- Every debug/throwaway file created while diagnosing the reconciliation above
  (`mesh/host/_debug_*_tmp.mjs`, `mesh/_debug_successor_ops_*.json`) was deleted
  before this report was written — none remain in the tree.

## Not verified (honest limits)

- **Bare's own WASI implementation** covering all 15 of these functions is
  unproven here — this phase is entirely Node-side (`node:wasi`), as scoped
  (Bare comes in Phase 1a/2). P0-A/B's dependency to confirm, now against **15**,
  not 10.
- **`poll_oneoff` as a no-op**: this phase ran real folds successfully under
  Node's real `poll_oneoff` implementation, which necessarily works, but that
  says nothing about whether a stub/no-op `poll_oneoff` on Bare's side (P0-A/B's
  plan, per `PHASE0_GATE_C_VERIFICATION.md`) would also work — not tested against
  a stub here, still open.
- **Performance**: this phase did not benchmark reactor-vs-command call latency
  or the warm-instance advantage under Node. `attach-spike.mjs`'s existing REFOLD
  BENCH (10k/5k/1k ops through the command-module `applyViaWasm`) was not re-run
  against the reactor channel.
- **Multi-writer/real replication timing effects**: the reissue-scenario
  reconciliation confirmed op-array equality between the real replicated flow and
  this spike's simplified single-node reconstruction for THAT ONE scenario;
  it was not re-verified for every other scenario that also simplifies away
  replication (room/invite/mirror/social/missionc/missiond) — those rest on the
  same general argument (reducer sorts canonically, ops are self-contained signed
  values) rather than an individual diff-test each, though every one of them DID
  independently pass its own golden-digest sanity check, which is strong
  corroborating evidence the same argument holds for all of them.
