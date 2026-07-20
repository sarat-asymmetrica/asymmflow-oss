# SC-3b — "Check the connection" stops being a stub

**Campaign:** The Sealed Corridor · **Mission:** SC-3b (menu [1], client-word wiring)
**Date:** 2026-07-20 · **Branch:** `feat/fable-sealed-corridor`
**Files owned:** `mesh/kit/bare-probe.mjs` (additive only), `mesh/kit/bare-connection-check.mjs`
(new), `mesh/kit/bare-connection-check-spike.mjs` (new, gate), this report.

---

## 1. What was built

`bare-probe.mjs` gained four `export` keywords and nothing else: `withTimeout`,
`checkDht`, `checkNat`, `printCgnatCard`. `git diff` confirms the entire change is those
four lines — no logic, threshold, verdict word, printed string, or the CLI entry point
was touched. `bare-probe.mjs --self-test` is re-verified unregressed below, through a
real spawned pipe.

`bare-connection-check.mjs` is a new, single-export module:
`runConnectionCheck({ write, ask } = {})`. It runs bare-probe.mjs's check 1 (DHT
bootstrap reachability), check 2 (NAT self-diagnosis), and check 3 (the CGNAT card,
reusing `printCgnatCard` verbatim), then renders the result in Reception-Grade client
words, bounded and safe to call from inside a live guide session. It never throws,
never hangs past its own bound, never calls `process.exit()`/`Bare.exit()`, and never
has a raw 64-hex key to print.

## 2. The one-line wire-in for `bare-guide.mjs`

In `checkConnection(io, write)` (currently the honest stub at
`mesh/kit/bare-guide.mjs:219-225`), replace the stub body with:

```js
async function checkConnection(io, write) {
  await ensureFirewall(io, write)
  const { runConnectionCheck } = await import('./bare-connection-check.mjs')
  await runConnectionCheck({ write, ask: io.ask })
}
```

(A static top-level import works equally well if the lead prefers it — the dynamic
form above just keeps the diff to `checkConnection`'s body alone.) `io.ask` is
`createGuideIO`'s own `ask(prompt)` method — same object, same FIFO-queue guarantee,
passed through unchanged.

## 3. What was deliberately left out, and why

**Check 4, the hyperswarm punch test, is NOT run by this module.** It needs a second
live machine and a pasted key. SC0_PORT_MAP.md's own measurement (N=7: 6/7 AMBER, 1/7
RED, negative control 7/7 RED) shows a single punch can legitimately take up to
`PUNCH_WAIT_MS` (45s) and can come back RED even when the corridor itself is fine.
Running it from a "quick diagnostic" menu item would mean either blocking on `--listen`
for up to 45s with nobody to dial it, or building invite/pairing plumbing that is SC-3's
job, not this one's. `runConnectionCheck`'s own closing line says this explicitly to the
user: *"This checks the internet path, not the other computer directly — the messenger
option is the real end-to-end test."* Never a silent gap — menu [2] already performs a
real, two-sided connection through the actual room ceremony, which is a stronger test of
"can I reach the other computer" than a synthetic, unpaired punch would be here.

**The `ask` parameter is used for exactly one thing:** offering a bounded retry after a
RED result (see §4). It is never used for a paste-a-key step, since check 4 is not run.

## 4. The copy — how it satisfies SC0_PORT_MAP.md §1d

Every non-GREEN result states what was actually observed, in plain words, and what to do
next:

- **CORRIDOR RED:** *"This computer could NOT reach the internet meeting point just now.
  One failed check like this is NOT proof the corridor is broken — try again first. If it
  fails again, check this computer's internet connection and ask your support contact for
  help."* A retry is then offered via `ask` — but bounded: at most one automatic retry
  (`MAX_ATTEMPTS = 2`), never an unbounded loop. AMBER and GREEN never prompt for a retry
  — AMBER (firewalled) is a real, stable property of the network that a retry would not be
  expected to change; prompting there would just be friction.
- **CORRIDOR AMBER:** *"...this network reports as firewalled or behind a shared/NAT
  connection. That usually still works fine... use the CGNAT check above with your
  support contact."*
- **CORRIDOR GREEN:** *"...That is a good sign — but the real test is opening the
  messenger (option 2)..."* — even a clean pass is not oversold as a guarantee.

The three verdict words themselves are never invented — every one is `computeVerdict`'s
own output from bare-probe.mjs, reused unchanged (`export`ed, not reimplemented).

## 5. Presentation duplication — declared, not accidental

`bare-guide.mjs` owns `printVerdictLarge`/`reportError` and does not export them
(single-writer split; I do not touch that file). `bare-connection-check.mjs` re-derives
the same shapes locally (`frameVerdict`, matching the framed block + "Read this word to
the person on the call:" convention). This is a deliberate, declared cost of the file
split, exactly as the brief anticipated — not an oversight.

## 6. The test seam — declared

`runConnectionCheck({ write, ask })` is the entire public contract, unchanged from the
required signature. `_testHooks` (an exported **mutable object**, `{ checkDht, checkNat
}`) is the dependency-injection seam the gate uses to force a red result without touching
bare-probe.mjs's own logic — the real implementation reads `_testHooks.checkDht`/
`_testHooks.checkNat` internally, and the real guide's one-line wire-in (§2) never
touches `_testHooks` at all, so the defaults (bare-probe.mjs's real, unmodified checks)
are what ships. This keeps the wiring line pristine while giving the gate a way to
inject a deliberately-broken dependency (an object property mutation on an imported
binding, not a reassignment of the binding itself — the standard JS pattern for this).

## 7. Gate — command, results, and the negative control run FIRST

```
npm run sc3bspike
```
(wired in `package.json`, appended as the last script; also runs `npm run build` first)

**Result: `SC3B CONNECTION CHECK SPIKE GREEN` — 10 checks, 0 failures.**

| leg | N | result |
|---|---|---|
| 1. Negative control (synthetic unreachable DHT, injected via `_testHooks`) — **run and asserted FIRST, before any green claim** | 16 | **16/16** correctly reported `CORRIDOR RED` with the plain-word copy above |
| 2. Never hangs (eternally-unresolved `checkDht`, single attempt) | 16 | **16/16** resolved in <25s (bounded by `ATTEMPT_TIMEOUT_MS`=20000ms), verdict RED, attempts=1 |
| 3. Retry offer is capped, not unbounded (eternally-unresolved `checkDht`, `ask` always answers "try again") | 5 | **5/5** stopped at attempts=2, elapsed ~38-48s (≈2× the per-attempt bound), never ran a third time |
| 4. Never calls exit (content-proof: markers printed *after* `runConnectionCheck()` returns) | 5 | **5/5** the host process was observably still alive after the call returned |
| 5. Live DHT/NAT check, no override — harness-level completion | 7 | **7/7** OK (completed, well-formed result) |
| 5b. Live DHT/NAT check — **measured, not claimed** corridor verdict on this machine/network today | 7 | GREEN=0 AMBER=7 RED=0 — **N=7 is too small to state a rate (Rule 5)**; recorded as evidence, matching SC0's own precedent for this exact leg |
| 6. No raw 64-hex key in any collected output | — | 0 leaks across all of the above |
| Mandated: `bare-probe.mjs --self-test` unregressed, via `runSpawnPipe` | 5 | **5/5** OK — 15/15 `[OK]`, `SELF-TEST GREEN`, every run |
| `spawn-pipe-harness.mjs`'s own `selfTest()` | 5 per fixture | PASS — good=OK 5/5, hang=HANG 5/5, total-loss=TOTAL_LOSS 5/5, partial=PARTIAL 5/5 |

**The negative control ran and was asserted before leg 2 even started** — the gate's own
code structure enforces this (leg 1 is first in file order and the script prints its own
"HARNESS/WRAPPER UNTRUSTWORTHY" line and would have continued to report the failure if
it had not gone red 16/16). It did: **16/16 RED**, so everything below it is admissible.

## 8. A `runSpawnPipe` API-shape note (not a Bare defect, not a file I'm allowed to touch)

`bare.exe` requires the entry script **before** any flag meant for that script — measured
directly, not assumed:

```
bare.exe kit/bare-probe.mjs --self-test   → runs the self-test correctly
bare.exe --self-test kit/bare-probe.mjs   → "unknown flag: self-test", does nothing
```

`spawn-pipe-harness.mjs`'s `runSpawnPipe` always builds its child's argv as
`[...args, scriptPath]` — args **before** scriptPath — so it cannot express "script, then
a trailing flag" directly. Worked around with a small Node launcher (`spawnSync(...,
{ stdio: 'inherit' })`), spawned by `runSpawnPipe` as its own `scriptPath` under
`process.execPath`, which immediately re-spawns the real target in the correct order.
`stdio: 'inherit'` passes the real pipe file descriptors straight through, so
`runSpawnPipe`'s own `child.stdout`/`child.stderr` listeners see the actual
`bare-probe.mjs --self-test` output exactly as if they had spawned it directly — still a
real OS pipe end to end, never a shell pipe. `spawn-pipe-harness.mjs` itself is untouched
(not owned by this mission); this is flagged here for the consolidated upstream filing,
not acted on beyond the local workaround.

## 9. A deliberate departure from `runSpawnPipe`'s sequential execution, for two legs only

Legs 2 and 3 are bounded at `ATTEMPT_TIMEOUT_MS` (~20s) or `2×ATTEMPT_TIMEOUT_MS` (~40s)
per run. Rule 5 requires N≥16 wherever a hang/race is plausible, and both legs are
exactly that class of check (proving the wrapper's *own* timeout plumbing fires
reliably, not just usually). Sequential N=16 at ~20-25s each would cost 5+ minutes for
leg 2 alone. `spawnBareParallel` (local to the spike file, `spawn-pipe-harness.mjs`
untouched) is the same real-spawned-pipe technique — never a shell pipe, a per-process
timeout+kill classified as a hang, content-asserted never exit-code-asserted — run
concurrently via `Promise.all` instead of a for-loop, cutting the same N=16 to ~20-25s
wall time. `runSpawnPipe` itself is used, per instruction, for the one leg that
explicitly names it (§8).

## 10. What this did NOT verify

- **No claim about a clean machine.** This machine has Node, npm, and the dev toolchain
  installed. Not re-checked here — out of scope for this mission (SC-4's job).
- **No claim about the sealed, bare-pack'd bundle.** Every leg here drives
  `bare-connection-check.mjs` as a raw file under `bare.exe`, never through
  `build-bare-kit.mjs`'s packed `app.bundle`. SC-4's own gate (following
  `bare-guide-spike.mjs`'s layer 4 precedent) is where the packed-bundle question belongs
  — this mission's file ownership does not include `build-bare-kit.mjs`.
- **No claim about the wiring actually landed in `bare-guide.mjs`.** §2 above is a
  recommendation for the lead to apply; this report does not certify the guide's own
  behavior post-wire-in, since I do not own or edit that file.
- **The live-network leg (N=7) is evidence, not a rate.** On this machine today the
  network is firewalled (AMBER, 7/7) — consistent with SC0's own finding on the same
  machine. No claim is made about what a receptionist's machine, or this same machine
  tomorrow, will report.
- **The retry-offer path with an `ask` that answers "try again" and then the check
  actually succeeding** (a transient RED followed by a real GREEN on retry) was not
  separately exercised — legs 2/3 use an eternally-hung dependency, which proves the
  *bound* holds but not the "retry recovers a transient failure" happy path. This is a
  reasonable inference from the code (`for (;;)` re-runs `attemptOnce` identically each
  time) but is recorded here as unverified, not asserted.
- **No Bare defect was found in this mission** — the one wrinkle (§8) is a
  `runSpawnPipe` API-shape limitation combined with `bare.exe`'s real (and reasonable)
  argv convention, not a Bare bug.

---

*Port the proven, seal the port, prove the seal.* 🐻
