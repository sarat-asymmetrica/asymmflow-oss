# Phase 2 — Gate: Rule 4 (real spawn-pipe) run against the Bare bridge

**Date:** 2026-07-20 · **Author:** orchestrator (Opus 4.8), run personally
**Verdict: the bridge PASSES.** 10/10 under Bare, 5/5 under Node, all frames returned,
with a negative control proving the harness can fail.

## Why the gate had to be run separately

`bare-bridge-spike.mjs` is green at 37 checks, and its coverage is honestly declared: it
injects the `io` object rather than spawning a child process (stated in its own header).
That is a legitimate design for exercising dispatch logic — but it means those 37 checks say
**nothing about the real stdio transport**. That gap is exactly how `stdio-check.mjs` passed
for a day while dropping 100% of its payloads, which is why Rule 4 exists.

## Method

A temporary entry (`host/.gate-worker-entry.mjs`, deleted after the run — deliberately not
committed) invoked the REAL `runStdioWorker()` on real stdio. A Node parent drove it with
`spawn(bin, [script], { stdio: ['pipe','pipe','pipe'] })`, wrote three ndjson requests
(`listRooms`, `listRooms`, and a deliberately invalid method to force an error frame),
closed stdin, and collected `stdout` via `'data'` events, matching responses by `id`.
Assertions are on **frame content**, never exit code.

## Results

| run | result |
|---|---|
| real bridge worker under **Bare**, 10 runs | **allFramesBack=10, partial=0, hang=0** |
| real bridge worker under **Node**, 5 runs | **allFramesBack=5, partial=0, hang=0** |
| **NEGATIVE CONTROL** — same driver, worker writing via `proc.stdout.write` | **allFramesBack=0, hang=4/4** |
| positive control re-run at the negative control's shorter timeout | **allFramesBack=4/4** |

The negative control is the part that makes the pass meaningful: the identical harness
reports total failure against a Rule-2-violating worker and full success against the real
one, at the same timeout. It can distinguish them.

## Rule compliance audit of `bare-bridge.mjs`

| rule | status |
|---|---|
| 1 — no async `WebAssembly.compile()`/`instantiate()` | **PASS** (none present; the reducer channel is `apply-bare.mjs`, already compliant) |
| 2 — frames via `console.log`, never `proc.stdout.write` | **PASS** — `getRealStdio()`: `write(str) { console.log(str); return true }` |
| 3 — explicit exit on stdin `end` | **PASS**, and improved on: it `await transport.waitIdle()`s in-flight dispatches BEFORE exiting, so a request arriving just before stdin closed still gets its response written rather than truncated |
| 4 — gated through a real spawn pipe | **PASS as of this document** (was not covered by the spike) |

The drain-before-exit refinement is the bridge's own addition beyond P0-D's note and is a
genuine improvement: Rule 3 alone prevents the hang, but does not by itself guarantee a
late-arriving request gets answered.

## Correction to a claim in `PHASE2_BRIDGE_REPORT.md`

That report states the flush race "did not reproduce today" — 0/10 on a *corrected* copy of
P0-D's repro script, versus P0-D's original ~25–30%, and interprets this as the hazard
possibly being absent on this machine today.

**That inference is wrong, and the hazard is fully live.** Re-running the UNMODIFIED
`mesh/host/bare-spike/stdio-check.mjs` through a real spawn pipe, minutes after that report:

```
runs=10  ready=10/10  echoed(payload)=0/10  exits={"0":10}
runs=20  ready=20/20  echoed(payload)=0/20  exits={"0":20}
```

**30/30, 100% payload loss, right now.** The 0/10 result came from a script that had been
*corrected into compliance* — which removes the bug. Measuring a fixed script and concluding
the defect no longer reproduces is measuring the fix, not the defect.

This is the sixth instance in this campaign of a probe that could only return the answer it
gave, and at least two of the previous five were the orchestrator's own. The recurring shape:
**the thing under test was silently not the thing we believed we were testing.**

The report's practical recommendation — build request-id timeout/retry as if the race is
real — stands, and is now backed by evidence rather than caution. Its frame-level
detectability design (`id` echo, `pendingPartial()`) is exactly right.

## Not verified

- Long-run/soak behaviour: the longest observation here is three frames per process.
  Sustained multi-hour worker operation is untested.
- Concurrency: one client, sequential frames. Multiple simultaneous in-flight requests, and
  interleaved large attachment payloads, are untested.
- `verify-transcript.mjs` remains un-migrated (`node:crypto` + `./apply.mjs`), so it cannot
  load under Bare; the spike guards it and skips one check (37/38). Carried as a known gap.
- Backpressure: `console.log` return value is ignored (`write` always returns `true`).
  Behaviour when the parent reads slowly, or stops reading, is uncharacterized.
