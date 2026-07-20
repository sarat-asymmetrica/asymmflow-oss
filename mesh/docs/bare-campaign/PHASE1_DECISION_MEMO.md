# Phase 1 — Decision memo (Gate P1)

**Date:** 2026-07-20 · **Decided by:** orchestrator (Opus 4.8), per campaign §3 Phase 1
**Escalation check:** the spec reserves STOP-AND-ASK for "both paths fail" or "the winner
requires changing reducer source semantics". **Neither applies** — both paths succeeded,
and `mesh/reducer/**` is untouched by both (empty diff vs `main`). The orchestrator decides
and records the ruling. Owner ruling R1 pre-authorized the reactor path on exactly these terms.

## The fork was not a fork

The campaign spec framed Phase 1 as **WASI-shim vs `go:wasmexport`** — two alternatives,
pick one. That framing rested on spec §2's claim that a `wasmexport` build "would need NO
WASI import table at all". **That claim is false** (verified twice, independently, with
real builds — see `PHASE0_GATE_C_VERIFICATION.md` and its retraction note):

| build | distinct `wasi_snapshot_preview1` imports |
|---|---|
| command module (`mesh/cmd/reducer`) | **16** |
| reactor (`mesh/cmd/reducer-reactor`) | **15** — only `fd_read` eliminated |

Stock Go's runtime pulls `os` transitively through `encoding/json` and the crypto packages,
and `os`'s package init builds the stdio fd table regardless of entry-point style.

**Therefore the shim was never optional.** It is a prerequisite of running *any* stock-Go
wasip1 module under Bare. The two paths are not alternatives; one is the floor and the
other is an option built on top of it. Both were built. Both are green.

## Results

| | Phase 1a (shim + command module) | Phase 1b (shim + reactor) |
|---|---|---|
| byte-identity vs Node WASI host | **13/13, 0 divergent bytes** | **13/13, 0 divergent bytes** |
| runs under Bare | **yes**, verified | not yet (Node-side proven) |
| `mesh/reducer/**` touched | no | no |
| syscalls required | 16 | 15 (subset — same shim serves both) |
| new Go artifact | none | one (`reducer-reactor.wasm`, 3.94 MB) |
| filesystem at runtime | **none** — in-memory Buffer fds | none |
| instance lifetime | one-shot per apply | warm across calls |
| enables incremental fold | no | yes (future) |

Note the reactor's originally-projected advantage largely evaporated: the temp-file-per-call
design it was meant to retire was *already* retired by `apply-bare.mjs`'s in-memory fds, and
the syscall saving is one call, not six.

## RULING

**Phase 2 and Phase 3 ship path 1a: the WASI shim driving the EXISTING, UNMODIFIED command
module.** The reactor is **retained, committed and green**, as the proven forward path for
incremental folding — not shipped in the sealed kit for now.

Reasoning, in priority order:

1. **Fewest moving parts in the artifact that goes to a client.** 1a adds no new Go build
   output to the sealed folder and no second ABI to keep correct. The reactor's malloc/
   fat-pointer/pin-registry contract is careful work (and gate-hardened), but it is
   additional surface that Phase 3 gets no benefit from.
2. **The reducer stays literally untouched.** 1a runs the same `reducer.wasm` the Node line
   ships today. That keeps the rollback path (owner ruling R1 condition 2) not merely warm
   but byte-identical to production.
3. **The reactor's benefits are real but not yet needed.** Warm instances and incremental
   fold matter when replay cost dominates; Phase 3's goal is ceremony parity on a field kit.
   Adopting an optimization before its problem is measured is the ladder-building D4 warns
   against.
4. **Nothing is thrown away.** The reactor is committed, byte-identical and vet-clean. If
   Phase 4 or a later wave measures replay cost as a real constraint, the channel swap is a
   host-side change with a proven parity harness already in the tree.

**This ruling is reversible on evidence.** The trigger to revisit: a measured fold-latency
problem, or an incremental-apply requirement from the Autobase integration. Neither exists today.

## Consequences for Phase 2 (binding)

- `mesh/host/wasi-preview1-lite.mjs` is the reducer's runtime under Bare. It is dual-runtime
  by construction (zero imports) and must **stay** that way — no `node:` and no `bare-*`
  import may be added to it.
- `mesh/host/apply-bare.mjs` has a **known pack-time blocker**: it selects fs via
  `isBare ? import('bare-fs') : import('node:fs')`, and `bare-pack`'s static traverser
  cannot walk past a `node:` specifier. Fix with the verified `bare` condition import map
  (`PHASE0_GATE_B3_CONDITION_MAP.md`) before Phase 3 packaging.
- All 36 non-aliasing call sites (`node:crypto` ×11, `node:os` ×18, `node:readline` ×7)
  must be migrated. `node:fs`/`path`/`url`/`net` alias at runtime but still break the pack,
  so they migrate too.

## Honest limits of this decision

- **No performance measurement was taken on either path.** The ruling deliberately does not
  rest on a perf claim, because none was made. If perf turns out to favour the reactor, this
  ruling changes.
- The reactor has **not** been run under Bare (its parity was proven under Node). Should it
  ever be promoted, that gate must be run first.
- `poll_oneoff`'s never-blocks simplification is **unexercised** — instrumented traffic shows
  it is never hit by any of the 13 scenarios. It is correct for this reducer's shape today
  and an open risk for any future reducer that legitimately sleeps.
- API equivalence of the `bare-*` packages against their Node counterparts is unestablished
  (`bare-fs` 95 exports vs 107; `bare-crypto` 21 vs 68). Resolution is proven; behaviour is not.
