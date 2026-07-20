# Bare Runtime Campaign — Final Report ("The Sealed Ship")

**Date:** 2026-07-20 · **Branch:** `feat/fable-bare-runtime` (not pushed)
**Orchestrator/gate:** Opus 4.8 · **Coders:** Sonnet 5 ×4 · **Owner:** the Commander
**Authorized scope:** Phases 0–3 (owner ruling R4). **Phase 4 is owner-reserved and untouched.**

---

## 1. Verdict

**The charter is met.** A sealed kit extracted to a from-scratch directory — no Node, no
npm, no source, no `node_modules` of its own — runs the complete client ceremony through
the real double-click launcher, folding a message through `reducer.wasm` inside the artifact:

```
  ASYMMFLOW MESH -- GUIDE (Bare)
(created a new room for this kit -- "kitchen table")
  (posted, seq 2)
Goodbye -- this window is safe to close.
```

The **FR-1 disease class is structurally unrepresentable** in that folder: there is no
`node_modules` to miss, no PATH consulted, no module resolution outside the sealed bundle.
Not by policy — by construction.

## 2. What was proven, with evidence

| claim | evidence |
|---|---|
| The unmodified reducer runs under Bare | 13/13 goldens byte-identical, both runtimes |
| A `go:wasmexport` reactor is viable | 13/13 byte-identical, `go vet` clean |
| Protocol v0 works over Bare stdio | 45 checks under Node, 37 under Bare, real spawn pipe |
| The kit ships sealed | 24 files, 62.8 MB, runs in hostile geography |
| The Guided Path ports | 17/17, ceremony verified through the real `.cmd` |
| The probe ports | 15/15 self-test, live DHT bootstrap, two-process punch (RTT 8 ms) |
| The anchor's shape ports | spike green, real spawn-pipe CLI refusal 5/5 |

`mesh/reducer/**` was never touched. The Node line stayed green throughout as the rollback path.

## 3. Two Bare defects found, characterized, mitigated

Both appear to be **new findings** — no matching upstream issue located.

- **Bug A — `await WebAssembly.compile()` loses stdout.** 53% loss on a real pipe, 27% on a
  file redirect, exit code 0. Mitigation: synchronous `new WebAssembly.Module()`, 80/80 clean.
- **Bug B — `bare-process`'s `process.stdout.write()` deadlocks on a pipe.** 30/30 hang, with
  no WASM involved. Mitigation: `console.log`, 30/30 clean.

Closest upstream corroboration: `bare-kit#92` (maintainers abandoned Windows pipes for an
in-process queue) and `bare` PR #160 (async-close vs stopped-loop race). Neither is the same bug.

**Four binding rules** now govern the codebase, and they are **executable, not prose**:
`mesh/host/stdio-seam-spike.mjs` *runs* both violations and fails loud if either ever stops
failing — so an upstream fix gets noticed rather than silently rotting the test.

## 4. The campaign's real lesson

**Seven times, a probe measured something other than what its author believed. Three were the
orchestrator's own.**

The most consequential: `mesh/host/bare-spike/stdio-check.mjs` — the literal DP4
ndjson-over-stdio proof-of-concept, marked PASS on 2026-07-19 — **silently drops 100% of its
reply payloads** when driven through a real `spawn` pipe. It passed only because it was
verified with a *shell* pipe. Our proof was not proving what we believed.

Others: a 10-syscall measurement taken against a probe that dead-code-eliminated the relevant
path (real answer: 15); a `node:fs` "does not resolve" finding from a probe that could not
report success for *any* input; a condition-map recipe invalid under the runtime it was
written for, verified three ways on the branch that worked and zero ways on the branch that
did not; a guide "defect" that was the gate's own malformed input; three false kit-launcher
failures that were Git Bash mangling `cmd.exe /c`; and a diagnosis that compared
bundled+spawn-pipe against unbundled+shell-pipe, changing two variables at once.

Distilled:

1. **A probe that cannot report success proves nothing by failing** — and vice versa. Verify
   the harness before trusting either colour.
2. **Vary one axis at a time.** Ruling out candidate causes is not the same as isolating the variable.
3. **Test the layer the client touches**, not the one beneath it.
4. **Silent success is the worst failure mode.** Both Bare bugs, both integration defects, and
   the launcher all failed while reporting success.

## 5. Exit codes are worthless here — three independent layers

Bare exits 0 on total stdout loss. Bare exits 0 while throwing. The launcher flattened
everything to 0 (now fixed to propagate). Even propagated, **a zero exit proves nothing**: a
kit that folds a *wrong digest* and never throws exits 0 by construction.

**Direct Phase 4 consequence:** an anchor scheduled task treating exit code as health would
report success on a completely broken kit, forever. Task Scheduler's retry logic and "last
run result" are exit-code driven. **Anchor health must assert on content.**

## 6. Integration is a distinct gate

Every component gate was green — guide 16/16, bridge 45/45, probe 15/15, anchor green, parity
13/13 both runtimes, smoke, reactor — **and the composition failed three times**:

1. `isMain` cannot fire inside a bundle (`argv[1]` is the bundle; `import.meta.url` is virtual)
   → `runGuide()` never called, exit 0, zero bytes.
2. `reducer.wasm` never offloaded (`new URL()` is invisible to `bare-pack`'s asset detector)
   → full ceremony renders, room created, **only posting fails**.
3. `runGuide()` never called `exit()`, relying on natural termination at stdin EOF
   → **intermittent hang, ~1 in 4 runs.**

The second is the more dangerous shape: a kit that boots, draws its menu and says Goodbye
reads as working. Both entry files now carry header comments explaining why the fixes must
not be "simplified" away.

### The third one is a failure of THIS GATE, and is recorded as such

Defect 3 is **RULE 3 violated in a second file** — the exact "let the loop drain naturally"
shape the rules already name as producing a 10/10 hang in the bridge worker. The coder found
it by stress-testing before reporting green, diagnosed it as the same bug class, and stated
plainly that it happened because the Phase 2 fix had not been generalized into a rule applied
on sight. That is the right instinct and it caught a defect nobody else had.

**The gate certified Phase 3 as PASSED without catching it.** The orchestrator's samples were
2 and 5 runs. At a ~25% failure rate, clean runs at that sample size are unremarkable — the
verdict was luck, not evidence. Re-verified after the fix at **16/16**, a sample size that
would make missing a 1-in-4 defect statistically negligible.

The lesson generalizes past this campaign and belongs with the four probe rules in §4:

5. **An intermittent defect is invisible at small N.** A pass on 2–5 samples cannot
   distinguish "works" from "fails 25% of the time". Every rate in this campaign that turned
   out to matter was measured over ≥30 runs; the one verdict taken on a handful of samples is
   the one that was wrong. Where a hang or a race is even plausible, sample size IS the test.

A green result from a harness that can go red is still only as strong as the number of times
it was asked.

## 7. Corrections to the campaign spec itself

- **§2's premise is false.** `go:wasmexport` does *not* eliminate the WASI import table (15 vs
  16 — only `fd_read` goes). A shim was mandatory on both paths; Phase 1 was never a fork.
- **The reactor's justification changed.** Not a cheaper shim (worth one syscall) but the
  warm-instance call pattern. Gate P1 shipped path 1a on the unmodified command module.
- **No single-`.exe` mechanism exists.** `pear-appling` is bootstrap-only and fetches the app
  from the swarm on first launch — disqualified for an offline field kit.

## 8. Owner-reserved (Phase 4, untouched)

1. **File the two Bare bugs upstream.** Outbound public action; evidence is ready.
2. **Anchor scheduled-task migration.** No task on this machine was created, modified or
   removed. The anchor's room-hosting payload still needs wiring to the Bare bridge.
3. **A genuinely clean VM.** This machine has Node and npm installed globally — "no npm tree in
   the directory" is **not** the same guarantee as a clean machine. Stated plainly rather than
   overclaimed.
4. **The corridor field ceremony** (India↔Bahrain) and real cross-machine testing.
5. **`bare-tcp` as an explicit dependency**, if the probe's `--holesail` loopback check is wanted.

## 9. Known gaps, stated plainly

- Three menu items are **honest stubs**: "Check the connection", and the firewall and anchor
  system mutations. The guide says so to the user and gives a manual path.
- `verify-transcript.mjs` is un-migrated and cannot load under Bare (one skipped check).
- `poll_oneoff`'s never-blocks simplification is **unexercised** — never hit by any scenario.
- No soak, concurrency, or backpressure testing. Longest observation is a handful of frames.
- `bare-readline` **hangs under a spawned pipe** (0/10). Not on our path — the guide hand-rolls
  its stdin — but it is a live landmine for anyone who reaches for it.
- The kit is not byte-reproducible (`portable.flag` carries a timestamp); `app.bundle` **is**.

## 10. Deliverables

`mesh/host/`: `wasi-preview1-lite.mjs`, `apply-bare.mjs`, `bare-bridge.mjs`,
`spawn-pipe-harness.mjs`, `stdio-seam-spike.mjs`, `bare-entry.mjs`, + spikes
`mesh/cmd/reducer-reactor/`, `mesh/kit/`: `bare-guide.mjs`, `bare-guide-entry.mjs`,
`bare-probe.mjs`, `bare-anchor.mjs`, `build-bare-kit.mjs`, + spikes
`mesh/docs/bare-campaign/`: 24 documents — every phase, every gate, every retraction.

**Every wrong turn is still visible in those documents, including the orchestrator's.** That
was deliberate: the corrections are the most transferable part of the record.

*Read everything, assume nothing, seal the ship.* 🐻
