# Gate result: the Bare stdout loss is real, silent, and NARROW

**Date:** 2026-07-20 · **Author:** orchestrator (Opus 4.8), run personally
**Status:** root-caused to a single trigger; **our shipped code is not exposed**
**Relation to P0-D:** P0-D's independent root-cause mission was running in parallel; this is
the orchestrator's own measurement of the decisive question. Both should be read together.

## Why this was the campaign's top risk

P0-D reported ~25–30% of Bare runs producing truncated/zero output with exit code 0 after
`WebAssembly.compile()`. Our sidecar transport (the DP4 seam) is **ndjson frames over
stdout** and our core workload is **folding a WASM reducer**. If output can vanish silently
on the production path, the mesh loses business records while reporting success — strictly
worse than the FR-1 field failure this campaign exists to eliminate, because FR-1 at least
failed loudly. That would have threatened the Bare bet itself.

## Confirmed — and worse than reported, on the case that matters

Measured on **piped** stdout (our production case: a parent process reads the sidecar's
output), 30 runs of a script that reads the real 3.96 MB `reducer.wasm` and
`await WebAssembly.compile()`s it, then prints 20 lines plus a sentinel:

```
PIPED (stdout captured): pass=20/30  truncated=10/30
```

**33% loss, on pipes.** And it is not partial truncation — every failing run produced
**zero lines**, total loss of the entire buffer:

```
LOSS run=9  lines=0 exit=0
LOSS run=11 lines=0 exit=0
LOSS run=12 lines=0 exit=0
LOSS run=14 lines=0 exit=0
LOSS run=18 lines=0 exit=0
```

**Exit code 0 on every loss.** Silent, complete, and indistinguishable from success to any
caller that trusts exit status.

## Root cause: `await WebAssembly.compile()` specifically — not async in general

The trigger is far narrower than "async output races exit". Isolated by matrix, 30 runs each:

| variant | result |
|---|---|
| `await WebAssembly.compile(reducer.wasm)` then print | **20/30** ← the defect |
| `new WebAssembly.Module(reducer.wasm)` (synchronous) then print | **30/30** |
| top-level `await Promise.resolve()`, no wasm | 30/30 |
| no await at all, no wasm | 30/30 |
| `await` inside an async function (not top-level) | 30/30 |
| `setTimeout` callback prints | 30/30 |
| async `fs.readFile` callback prints | 30/30 |
| top-level await over async `fs.readFile` | 30/30 |

Top-level await is **not** the trigger — `await Promise.resolve()` is a microtask that
resolves in-tick and never exposes the race. Async I/O is **not** the trigger either;
timers and fs callbacks are clean. The fault is specific to the genuinely off-thread
**async WASM compile**, whose continuation appears to race process teardown in a way that
discards the pending stdout buffer entirely.

## THE MITIGATION — and we already had it by accident

**Use the synchronous `new WebAssembly.Module(bytes)`, never `await WebAssembly.compile(bytes)`.**
Proven 30/30 against the identical workload.

Every host in the tree already does this:

```
host/apply-bare.mjs:45     _module = new WebAssembly.Module(fsMod.readFileSync(WASM_URL))
host/apply-bare.mjs:113    const instance = new WebAssembly.Instance(mod, imports)
host/apply-reactor.mjs:45  const mod = new WebAssembly.Module(bytes)
host/apply.mjs:32          _module = new WebAssembly.Module(readFileSync(WASM_PATH))
```

**No shipped code path uses `WebAssembly.compile()`.** The only files that did were the two
Phase-0 diagnostic scripts P0-D found flaking (`wasm-compile-check.mjs`,
`wasi-imports-list.mjs`) — throwaway probes, never part of the artifact. This also explains
why the parity spikes ran clean across dozens of executions today while the diagnostics
flaked: the production path was never exposed.

## BINDING RULE (campaign-wide, effective immediately)

> No file destined for the sealed artifact may use `WebAssembly.compile()` or
> `WebAssembly.instantiate()` (the async forms). Use `new WebAssembly.Module()` and
> `new WebAssembly.Instance()`. A reviewer must reject the async forms on sight.

The rule matters more than the current clean state, because the defect's signature is
**silent success**: nobody would notice a regression here from a green test run. A future
coder reaching for `await WebAssembly.compile()` because it "looks more modern" would
reintroduce a 33% silent data-loss rate with every suite still passing.

## Consequences for the DP4 seam

**The stdio transport is NOT inherently unsafe under Bare.** The frame channel itself is
clean — timers, fs callbacks, and async continuations all deliver reliably. The risk was
real but is confined to one API we do not use and now may not use.

Phase 2's bridge should still treat frame delivery as verifiable rather than assumed
(P1-A was briefed to make dropped frames detectable rather than silent), but this removes
the possibility that the seam is fundamentally lossy.

## What this does NOT establish

- **Not root-caused at the runtime level.** WHY the async compile's continuation loses the
  stdout buffer is undiagnosed — this characterizes the trigger and a reliable mitigation,
  not the mechanism inside Bare. Worth an upstream issue (give-freely ethos); P0-D was
  tasked with the upstream search.
- Only `win32-x64`, only Bare 1.30.3, only this machine.
- Only tested with the real 3.96 MB module; whether small modules compile fast enough to
  hide the race is unknown and irrelevant to the rule.
- The interaction with `--offload`/bundled assets is untested.
- Exit code 0 on total output loss is itself alarming and is NOT explained here. Treat Bare
  exit codes as unreliable evidence of success generally; a second instance was observed
  independently during the condition-map gate, where a throwing process still exited 0.
