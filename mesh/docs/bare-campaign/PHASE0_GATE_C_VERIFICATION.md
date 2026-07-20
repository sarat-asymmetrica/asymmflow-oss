# Phase 0 — Gate verification of P0-C's verdict (orchestrator, independent re-run)

> **⚠️ SUPERSEDED IN PART — READ THIS FIRST (appended 2026-07-20, after Phase 1b).**
>
> This document's headline verdict — that `go:wasmexport` does not eliminate the WASI
> import table, so a shim is mandatory on both paths — **STANDS and was confirmed.**
>
> Its *secondary* finding is **WRONG and is retracted**: the claim that the reactor
> eliminates six syscalls and needs "a 10-syscall shim with no I/O at all". The real
> reducer's reactor build imports **15**, not 10. **Only `fd_read` is eliminated.** All
> five no-`bare-fs`-backing syscalls (`fd_close`, `fd_fdstat_get`, `fd_fdstat_set_flags`,
> `fd_prestat_get`, `fd_prestat_dir_name`) are still required.
>
> **Why this document was wrong, and it is my own error, not the coder's:** the 10 was
> measured against a trivial `add()` probe — a limitation this document itself flagged in
> its "What this does NOT establish" section, which then went ahead and used the number in
> its conclusion anyway. The probe imported nothing that pulls in Go's `os` package, so
> `os`'s stdio-fd-table init was dead-code-eliminated. The real reducer reaches `os`
> transitively through `encoding/json`, `crypto/ed25519`, `crypto/sha256` and
> `encoding/hex`, and that init survives linking regardless of entry-point style.
>
> Coder P0-C measured the real build and corrected the orchestrator. That is the doctrine
> working as intended (D1: every claim cites evidence; a measurement beats an inference).
>
> **Authoritative numbers: `PHASE1B_REPORT.md`. Shim authors must budget for 15 syscalls,
> including all five with no `bare-fs` backing.** The consequence: the reactor's value is
> the warm-instance call pattern and the retirement of the temp-file channel — *not* a
> cheaper shim. The Phase 1 decision matrix below must be read with that correction.

**Date:** 2026-07-20 · **Verifier:** orchestrator (Opus 4.8), independently of coder P0-C
**Why re-verified personally:** P0-C's verdict REFUTES a load-bearing premise written into
the campaign spec itself (`FABLE_CAMPAIGN_BARE_RUNTIME.md` §2: *"a future `//go:wasmexport
apply()` build … would need NO WASI import table at all"*). A finding that overturns the
spec's own stated alternative does not get accepted on one agent's transcript (D1).

## Method

Fresh throwaway module in the scratchpad (outside the repo tree, D5), Go 1.25.3:

```go
module gatecheck
go 1.25
```
```go
package main

//go:wasmexport apply
func apply(a int32, b int32) int32 { return a + b }

func main() {}
```
```
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o reactor.wasm .
```

Import tables enumerated with `WebAssembly.Module.imports()` under Node v22, deduplicated
to DISTINCT function names, and compared against the repo's real
`mesh/dist/reducer.wasm` (the existing wasip1 command module).

## Result — P0-C's verdict CONFIRMED

```
reactor.wasm: 10 distinct wasi imports; other namespaces: []
args_get, args_sizes_get, clock_time_get, environ_get, environ_sizes_get,
fd_write, poll_oneoff, proc_exit, random_get, sched_yield
exports: _initialize, apply, memory

cmd.wasm (repo reducer.wasm): 16 distinct wasi imports; other namespaces: []
args_get, args_sizes_get, clock_time_get, environ_get, environ_sizes_get,
fd_close, fd_fdstat_get, fd_fdstat_set_flags, fd_prestat_dir_name,
fd_prestat_get, fd_read, fd_write, poll_oneoff, proc_exit, random_get,
sched_yield
exports: memory
```

**The spec's §2 premise is FALSE.** A `go:wasmexport` reactor still imports 10
`wasi_snapshot_preview1` functions — the stock Go runtime's own baseline (scheduler,
GC, panic output, env/args introspection). No `GOOS=wasip1` Go binary instantiates
without them. A WASI shim is therefore **MANDATORY ON BOTH PHASE-1 PATHS**; Phase 1 is
not a shim-vs-no-shim fork, and the campaign text should be read with this correction.

Corollary confirmed: `_initialize` and `apply` are real exports on the reactor, and the
command module exports only `memory` — i.e. the reactor's warm-instance call pattern is
available as documented.

## The finding that actually matters: WHICH 6 disappear

The reactor's saving is not "6 fewer functions", it is **which** 6:

| eliminated by the reactor | what it costs to shim |
|---|---|
| `fd_read` | real fd table + backing file/stream reads |
| `fd_close` | fd table lifecycle |
| `fd_fdstat_get` | fd metadata |
| `fd_fdstat_set_flags` | **no `bare-fs` equivalent** — pure synthesized shim state (P0-A §5) |
| `fd_prestat_get` | **no `bare-fs` equivalent** — WASI-only preopen concept (P0-A §5) |
| `fd_prestat_dir_name` | **no `bare-fs` equivalent** — WASI-only preopen concept (P0-A §5) |

Every syscall P0-A identified as having *no `bare-fs` backing* is in the eliminated set.
The 10 survivors are all satisfiable with no filesystem at all:

| survivor | shim implementation |
|---|---|
| `args_get` / `args_sizes_get` | return an empty argv |
| `environ_get` / `environ_sizes_get` | return an empty environment |
| `clock_time_get` | `Date.now()` / monotonic source |
| `random_get` | `bare-crypto` random bytes |
| `sched_yield` | no-op, return 0 |
| `proc_exit` | throw a sentinel the host catches |
| `poll_oneoff` | no-op / `ENOSYS` (Go's runtime tolerates this in our fold path — TO VERIFY in Phase 1) |
| `fd_write` | route fd 1/2 to console; nothing else is ever written |

**On the reactor path the shim touches no filesystem whatsoever** — no `bare-fs`, no fd
table, no temp files, no preopen fiction. It also retires `apply.mjs`'s current
temp-file-per-call channel along with its `node:os`/`node:fs` usage, shrinking both the
port surface (orchestrator port map) and the sealed artifact.

## What this does NOT establish (honest limits)

- The reactor build tested here is a **trivial `add()` function**, not the real reducer.
  A real reducer reactor may pull additional runtime paths (JSON/reflect/GC pressure)
  and could import more than 10. **Phase 1b must re-enumerate against the REAL reducer**
  and this document's 10 must not be treated as final until it does.
- `poll_oneoff` as a no-op is an assumption pending a real fold run.
- No byte-identity has been demonstrated yet. R1's gate (goldens byte-identical,
  `mesh/reducer/**` untouched, command module still green) is untouched by this note.
- Bare's ability to satisfy these 10 in practice is P0-A/P0-B's dependency, unproven here.

## Consequence for Phase 1 (orchestrator ruling, within R1)

Phase 1 proceeds as specified — BOTH spikes get built and measured — but the decision
matrix is re-framed. The real question is no longer "shim or no shim" but:

> **16-syscall shim with a real fd layer (command module)** vs
> **10-syscall shim with no I/O at all + a warm instance (reactor)**.

Both must fold every existing golden vector byte-identically before either can win.
