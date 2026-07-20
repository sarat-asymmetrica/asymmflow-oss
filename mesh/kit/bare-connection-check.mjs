// bare-connection-check.mjs — SC-3b (Sealed Corridor campaign): wires
// bare-probe.mjs's real DHT/NAT checks into the sealed guide's menu [1]
// "Check the connection", in Reception-Grade client words. Read
// mesh/docs/bare-corridor/SC0_PORT_MAP.md §1d and §3.4 before touching this
// file — both are binding on the copy below.
//
// IN-PROCESS ONLY, not a design choice but a fact of the target shape: a
// bare-pack'd sealed kit has no `bare-probe.mjs` FILE on disk to spawn --
// everything is folded into one bundle at pack time. This module imports
// bare-probe.mjs's checks directly and calls them in the guide's own
// process. Consequence, unavoidable and confirmed empirically this session
// (`node kit/bare-probe.mjs --self-test` throws `require.addon is not a
// function` under plain Node -- bare-probe.mjs's own header says as much:
// its `bare-process` import is Bare-ONLY by construction): this file, and
// anything that imports it, is ALSO Bare-only. There is no dual-runtime
// path here, unlike bare-guide.mjs.
//
// WHAT IS RUN: bare-probe.mjs's check 1 (DHT bootstrap reachability),
// check 2 (NAT self-diagnosis), and check 3 (the CGNAT card, reusing
// `printCgnatCard` verbatim -- it is pure presentation over check 2's own
// output, no extra network cost).
//
// WHAT IS DELIBERATELY LEFT OUT, stated plainly rather than silently
// skipped: check 4, the hyperswarm punch test. It needs a second live
// machine and a pasted key -- SC0_PORT_MAP.md's own measurement (N=7: 6/7
// AMBER, 1/7 RED with an established negative control) shows a single punch
// can legitimately take up to PUNCH_WAIT_MS (45s) and can come back RED
// even when the corridor is fine. Menu [2] "Open the messenger" already
// performs a REAL two-sided connection through the actual room ceremony --
// that is a stronger and more representative test of "can I actually talk
// to the other computer" than a synthetic, unpaired punch fired from a menu
// item whose whole point is a FAST diagnostic. Running it here would either
// (a) block on `--listen` for up to 45s with nobody to dial it, which is a
// worse experience than today's honest stub, or (b) need its own
// invite/pairing plumbing, which is SC-3's job, not this one's. This
// module's own output says this explicitly to the user (see
// `runConnectionCheck`'s closing line) -- never a silent gap.
//
// THE COPY LAW (SC0_PORT_MAP.md §1d, binding on every string below): one
// probe run is not a definitive verdict about the network. RED here means
// "did not reach it just now," never "the corridor is broken." Every
// non-GREEN result says what to do next; a RED result offers a retry as
// the first step, bounded (see MAX_ATTEMPTS) so this menu item can never
// turn into an unbounded loop.
//
// PRESENTATION DUPLICATION, DECLARED: bare-guide.mjs owns
// `printVerdictLarge`/`reportError` and does not export them (single-writer
// file split -- this coder does not touch bare-guide.mjs). The same SHAPES
// (a framed verdict block, "Read this word to the person on the call:", an
// error fold-line) are re-derived below rather than imported. This is a
// deliberate, declared cost of the split, not an oversight -- see
// SC3B_REPORT.md.
//
// TEST SEAM, DECLARED: `runConnectionCheck({ write, ask })` is this
// module's entire public contract -- exactly the signature the guide wires
// in one line, unchanged. `_testHooks` (exported below) is a MUTABLE hooks
// object the real implementation reads from internally
// (`_testHooks.checkDht`, `_testHooks.checkNat`); the gate overwrites its
// properties to inject a deliberately-broken DHT check for the negative
// control (bare-corridor-entry.mjs/bare-guide.mjs never touch it, and never
// need to -- the default values are bare-probe.mjs's real, unmodified
// checks). This keeps the guide's wiring line pristine while still giving
// the gate a way to force a red result without touching bare-probe.mjs's
// own logic.
//
// TIME BUDGET / NEVER HANGS / NEVER THROWS / NEVER EXITS: every check
// attempt is raced against ATTEMPT_TIMEOUT_MS via bare-probe.mjs's own
// `withTimeout` (reused, not reinvented). Every failure path -- a caught
// error, a timeout, an unexpected throw from an injected test hook --
// resolves to a CORRIDOR RED result object; nothing in this file ever
// rejects out of `runConnectionCheck`, and nothing in this file calls
// `process.exit()`/`Bare.exit()` -- this runs inside a live guide session
// and killing the process would close the client's window mid-menu. The
// one INTENTIONALLY unbounded wait is `ask()` for a human's retry answer --
// consistent with every other `ask()` in bare-guide.mjs (e.g.
// `ensureFirewall`'s "Press Enter to continue"), which also waits
// indefinitely for a person, not a network call.
//
// NO RAW 64-HEX KEY: this module never has one to print -- check 4 (the
// only check that ever prints a z32 key) is not run here.
//
// No `node:` specifiers anywhere in this file (binding constraint, same as
// bare-probe.mjs and every other file in this campaign).

import { checkDht, checkNat, printCgnatCard, computeVerdict, withTimeout } from './bare-probe.mjs'

// Bounds ONE check attempt. bare-probe.mjs's own DHT_READY_MS (15000ms) is
// unexported and already bounds `checkDht` internally -- this is defense in
// depth, not a substitute: a truly wedged promise from an unexpected code
// path (an injected test hook, or a future change beneath us) must never
// leave the guide's menu loop stuck.
const ATTEMPT_TIMEOUT_MS = 20000

// At most one retry is ever offered automatically -- never an unbounded
// "try again" loop. A person who wants to try a third time can simply
// choose menu [1] again from the top.
const MAX_ATTEMPTS = 2

/** Mutable dependency seam for gate use only -- see file header "TEST SEAM,
 * DECLARED". The real guide never touches this; the default values ARE
 * bare-probe.mjs's real checks. */
export const _testHooks = { checkDht, checkNat }

function frameVerdict(write, verdict) {
  const line = '='.repeat(Math.max(40, verdict.length + 8))
  write('\n')
  write(line + '\n')
  write(`   ${verdict}\n`)
  write(line + '\n')
  write('\n')
  write(`Read this word to the person on the call: ${verdict}\n`)
}

/** Client-word explanation of a result -- the copy §1d governs. Never a
 * bare technical verdict alone; always what was observed and what to do
 * next. */
function clientWords({ verdict, firewalled }) {
  if (verdict === 'CORRIDOR GREEN') {
    return 'This computer can reach the internet meeting point right now, and this network\n' +
      'does not look blocked. That is a good sign -- but the real test is opening the\n' +
      'messenger (option 2) and actually connecting to the other computer.\n'
  }
  if (verdict === 'CORRIDOR AMBER') {
    return 'This computer can reach the internet meeting point, but this network reports as\n' +
      (firewalled ? 'firewalled or behind a shared/NAT connection.\n' : 'relayed rather than a direct line.\n') +
      'That usually still works fine. If the messenger does not connect, use the CGNAT\n' +
      'check above with your support contact.\n'
  }
  return 'This computer could NOT reach the internet meeting point just now.\n' +
    'One failed check like this is NOT proof the corridor is broken -- try again first.\n' +
    'If it fails again, check this computer\'s internet connection and ask your support\n' +
    'contact for help.\n'
}

async function attemptOnce(say) {
  let dht = null
  try {
    const dhtResult = await _testHooks.checkDht(say)
    dht = dhtResult.dht
    const natResult = _testHooks.checkNat(dht, say)
    printCgnatCard(say, natResult.publicHost)
    return { ok: true, state: { ...dhtResult, ...natResult } }
  } finally {
    // Same cleanup runProbe() itself performs in its own finally block --
    // this module bypasses runProbe (it needs client-word framing, not the
    // CLI's), so it is responsible for its own dht.destroy(). Guarded:
    // an injected test-hook dht stub may not have a real destroy() method.
    if (dht && typeof dht.destroy === 'function') {
      try { await dht.destroy({ force: true }) } catch { /* best-effort */ }
    }
  }
}

/**
 * runConnectionCheck({ write, ask }) -- see file header. `write(str)` is
 * the guide's output function. `ask(prompt)` is the guide's FIFO-queue
 * stdin asker; used ONLY to offer a bounded retry after a RED result (never
 * for a paste-a-key step -- check 4 is not run here, see header). Never
 * throws, never hangs past ATTEMPT_TIMEOUT_MS per attempt (plus however
 * long a human takes to answer a retry prompt, which is intentional and
 * unbounded, same as every other `ask()` in this guide), never calls
 * process.exit()/Bare.exit().
 *
 * Returns a plain object: { verdict, reason, dhtReachable, dhtTotal,
 * firewalled, attempts }. The verdict is always one of the three literal
 * words from bare-probe.mjs's computeVerdict -- never invented here.
 */
export async function runConnectionCheck({ write, ask } = {}) {
  const out = (s) => { try { write?.(s) } catch { /* a broken write must never wedge the check */ } }
  const say = (line) => out(line + '\n')

  out('\n')
  out('Checking whether this computer can reach the internet meeting point...\n')
  out('(this can take up to about 20 seconds)\n')

  let attempt = 0
  let last = null
  for (;;) {
    attempt++
    let attemptResult
    try {
      attemptResult = await withTimeout(attemptOnce(say), ATTEMPT_TIMEOUT_MS, 'connection check')
    } catch (err) {
      attemptResult = { ok: false, error: err }
    }

    if (!attemptResult?.ok) {
      const message = attemptResult?.error?.message ?? String(attemptResult?.error ?? 'the check did not finish')
      say(`FAIL: could not finish the connection check -- ${message}`)
      last = { verdict: 'CORRIDOR RED', reason: message, dhtTotal: 0, dhtReachable: 0, firewalled: null }
    } else {
      const state = { ...attemptResult.state, punchAttempted: false }
      const { verdict, reason } = computeVerdict(state)
      last = {
        verdict, reason,
        dhtTotal: state.dhtTotal ?? 0, dhtReachable: state.dhtReachable ?? 0,
        firewalled: state.firewalled ?? null,
      }
    }

    frameVerdict(out, last.verdict)
    out('\n')
    out(clientWords(last))

    // Retry offer: RED only (§1d exists precisely because a RED can be a
    // false negative), bounded by MAX_ATTEMPTS, and only if the caller gave
    // us an `ask` to use. GREEN/AMBER never prompt -- AMBER is a real,
    // stable property of the network (firewalled), not something a retry
    // is expected to change.
    if (last.verdict !== 'CORRIDOR RED' || attempt >= MAX_ATTEMPTS || !ask) break
    let answer
    try {
      answer = await ask('\nPress Enter to try again, or type skip and press Enter to go back to the menu.\n> ')
    } catch { answer = null }
    if (answer === null) break
    if (/^skip$/i.test(String(answer).trim())) break
  }

  out('\n')
  out('(This checks the internet path, not the other computer directly -- the\n')
  out(' messenger option is the real end-to-end test.)\n')

  return { verdict: last.verdict, reason: last.reason, dhtReachable: last.dhtReachable, dhtTotal: last.dhtTotal, firewalled: last.firewalled, attempts: attempt }
}
