// bare-connection-check-spike.mjs — SC-3b gate: proves bare-connection-
// check.mjs (the in-process wrapper around bare-probe.mjs's DHT/NAT checks,
// wired into the sealed guide's "Check the connection" menu item) is safe
// to call from a live guide session -- never throws into the caller, never
// hangs, never exits the host process, never prints a raw key -- and that
// bare-probe.mjs's own self-test is unregressed by this file's additive
// exports.
//
// RULE 1 FIRST (verify the probe): the negative control below runs and is
// asserted BEFORE any green claim is printed. If this spike cannot make
// the wrapper report red, nothing else here counts.
//
// NODE-ONLY by construction, same as every other *-spike.mjs in this
// campaign: this is the PARENT half of the seam, spawning bare.exe as the
// child. bare-connection-check.mjs itself is Bare-ONLY (it imports
// bare-probe.mjs, whose `bare-process` import fails under plain Node --
// measured directly this session: `node kit/bare-probe.mjs --self-test`
// throws `TypeError: require.addon is not a function` from
// node_modules/bare-abort/binding.js). Every leg below that executes real
// wrapper code therefore spawns bare.exe on a small fixture script written
// to a temp dir; the fixture imports bare-connection-check.mjs via a
// `file:///` URL (the same manual URL construction bare-probe.mjs's own
// isMain check uses -- verified empirically this session, not assumed).
//
// A DELIBERATE DEPARTURE FROM `runSpawnPipe`'s SEQUENTIAL DESIGN, for
// N legs only: spawn-pipe-harness.mjs's runSpawnPipe() runs one process at
// a time by design -- correct for its existing callers, but this file has
// two legs bounded at ATTEMPT_TIMEOUT_MS (~20s, bare-connection-check.mjs's
// own constant) that Rule 5 requires at N>=16. Sequential N=16 at ~20s each
// is 5+ minutes per leg. `spawnBareParallel` below is the SAME real-
// spawned-pipe technique (never a shell pipe; a per-process timeout+kill
// classified as a hang; content-asserted, never exit-code-asserted) run
// concurrently via Promise.all instead of a for-loop, so the same N=16
// costs ~20-25s wall time instead of 5+ minutes. spawn-pipe-harness.mjs
// itself is untouched (not owned by this mission) -- this is a local,
// spike-only helper, used only where the explicit instruction to use
// `runSpawnPipe` does not apply. The ONE leg that instruction DOES name
// (the bare-probe.mjs --self-test unregression check) uses the real
// `runSpawnPipe`, per that instruction -- see "MANDATED LEG" below for the
// one wrinkle that required.
//
// Run: node kit/bare-connection-check-spike.mjs
// (also wired as `npm run sc3bspike`)

import { runSpawnPipe, formatResult, selfTest } from '../host/spawn-pipe-harness.mjs'
import { mkdtempSync, writeFileSync, rmSync, existsSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawn } from 'node:child_process'

const __dirname = dirname(fileURLToPath(import.meta.url))
const MESH = join(__dirname, '..')
const BARE_EXE = join(MESH, 'node_modules', 'bare-runtime-win32-x64', 'bin', 'bare.exe')
const CONNCHECK_PATH = join(__dirname, 'bare-connection-check.mjs')
const PROBE_PATH = join(__dirname, 'bare-probe.mjs')

function toFileUrl(p) { return 'file:///' + p.replace(/\\/g, '/') }
const CONNCHECK_URL = toFileUrl(CONNCHECK_PATH)

let checks = 0
let failures = 0
function check(name, cond, detail = '') {
  checks++
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' -- ' + detail : ''}`) }
}

console.log('bare-connection-check-spike -- SC-3b, over a real spawned Bare pipe\n')

if (!existsSync(BARE_EXE)) {
  check('bare.exe found at the expected node_modules path', false, `not found at ${BARE_EXE}`)
  console.log(`\n${checks} check(s), ${failures} failure(s).`)
  console.log(`\nSC3B CONNECTION CHECK SPIKE RED (${failures} failure(s))`)
  process.exit(1)
}

// ── local parallel spawn helper (see file header) ─────────────────────────
function spawnBareOnce(scriptPath, { timeoutMs = 30000 } = {}) {
  return new Promise((resolve) => {
    const child = spawn(BARE_EXE, [scriptPath], { cwd: MESH, stdio: ['ignore', 'pipe', 'pipe'] })
    let stdout = ''
    let stderr = ''
    let settled = false
    const t0 = Date.now()
    const timer = setTimeout(() => {
      if (settled) return
      settled = true
      child.kill('SIGKILL')
      resolve({ stdout, stderr, code: null, hang: true, ms: Date.now() - t0 })
    }, timeoutMs)
    child.stdout.on('data', (d) => { stdout += d.toString('utf8') })
    child.stderr.on('data', (d) => { stderr += d.toString('utf8') })
    child.on('close', (code) => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      resolve({ stdout, stderr, code, hang: false, ms: Date.now() - t0 })
    })
    child.on('error', (err) => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      resolve({ stdout, stderr: stderr + String(err), code: -1, hang: false, ms: Date.now() - t0 })
    })
  })
}
async function spawnBareParallel(scriptPath, { runs, timeoutMs }) {
  return Promise.all(Array.from({ length: runs }, () => spawnBareOnce(scriptPath, { timeoutMs })))
}
function fraction(results, pred) { return `${results.filter(pred).length}/${results.length}` }
// JSON.stringify(undefined) is `undefined` (not a string), so a bare
// `.slice()` on a not-found `.find()` result throws -- every failure-detail
// call site below goes through this guard instead.
function detailOf(x) { return x === undefined ? '(no failing run to show -- all runs matched)' : JSON.stringify(x).slice(0, 400) }

let workDir = null
function fixture(name, source) {
  if (!workDir) workDir = mkdtempSync(join(tmpdir(), 'sc3b-spike-'))
  const p = join(workDir, name)
  writeFileSync(p, source)
  return p
}

const HEX64 = /\b[0-9a-f]{64}\b/i
const allOutputs = [] // every fixture's stdout across every leg, swept once at the end

try {
  // ── LEG 1: THE NEGATIVE CONTROL, FIRST — Rule 1 ──────────────────────────
  // A synthetic, deterministic "DHT that cannot bootstrap" (per the
  // mission's own suggestion), injected via `_testHooks` -- the declared
  // test seam (see bare-connection-check.mjs's own header). Proves the
  // wrapper's render path produces the honest CORRIDOR RED / plain-word
  // copy, not a fake pass, BEFORE anything else here is trusted.
  console.log('-- leg 1: negative control (MUST come first) --')
  const negSrc = `
import { runConnectionCheck, _testHooks } from ${JSON.stringify(CONNCHECK_URL)}
_testHooks.checkDht = async (say) => {
  say('FAIL: DHT bootstrap unreachable -- synthetic negative control (no network reachable)')
  return { dht: { firewalled: true, remoteAddress: () => null }, dhtOk: false, dhtTotal: 3, dhtReachable: 0 }
}
const result = await runConnectionCheck({ write: (s) => console.log(String(s)), ask: async () => 'skip' })
console.log('RESULT_JSON:' + JSON.stringify(result))
if (typeof Bare !== 'undefined') Bare.exit(0)
`
  const negPath = fixture('neg-control.mjs', negSrc)
  const negResults = await spawnBareParallel(negPath, { runs: 16, timeoutMs: 15000 })
  for (const r of negResults) allOutputs.push(r.stdout)
  const negOk = (r) => !r.hang && r.code === 0 &&
    r.stdout.includes('could NOT reach the internet meeting point') &&
    r.stdout.includes('NOT proof the corridor is broken') &&
    /"verdict":"CORRIDOR RED"/.test(r.stdout)
  console.log(`  negative control: ${fraction(negResults, negOk)} correctly reported CORRIDOR RED with plain-word copy`)
  check('REQUIRED FIRST: negative control — the wrapper can and does report CORRIDOR RED in plain words for an unreachable DHT (N=16)', negResults.every(negOk),
    detailOf(negResults.find((r) => !negOk(r))))
  if (!negResults.every(negOk)) {
    console.log('\nHARNESS/WRAPPER UNTRUSTWORTHY: the negative control did not go red on every run -- nothing below this line is admissible.')
  }

  // ── LEG 2: never hangs — an eternally-unresolved dependency, N=16 ───────
  // `_testHooks.checkDht` returns a Promise that never settles: the closest
  // stand-in for "no network reachable at all". `ask` answers 'skip'
  // immediately so exactly one attempt happens; the wrapper's own
  // ATTEMPT_TIMEOUT_MS (20000ms, ../kit/bare-connection-check.mjs) must
  // still bound the whole call. N=16 (Rule 5 — a timing race in the
  // wrapper's own timeout plumbing is exactly the kind of thing that could
  // fire reliably only most of the time).
  console.log('\n-- leg 2: never hangs (eternally-unresolved dependency, N=16) --')
  const hangSrc = `
import { runConnectionCheck, _testHooks } from ${JSON.stringify(CONNCHECK_URL)}
_testHooks.checkDht = () => new Promise(() => {})
const t0 = Date.now()
const result = await runConnectionCheck({ write: (s) => console.log(String(s)), ask: async () => 'skip' })
console.log('ELAPSED_MS:' + (Date.now() - t0))
console.log('RESULT_JSON:' + JSON.stringify(result))
if (typeof Bare !== 'undefined') Bare.exit(0)
`
  const hangPath = fixture('hang.mjs', hangSrc)
  const hangResults = await spawnBareParallel(hangPath, { runs: 16, timeoutMs: 28000 })
  for (const r of hangResults) allOutputs.push(r.stdout)
  const hangOk = (r) => {
    if (r.hang || r.code !== 0) return false
    const m = r.stdout.match(/ELAPSED_MS:(\d+)/)
    const j = r.stdout.match(/RESULT_JSON:(\{.*\})/)
    if (!m || !j) return false
    const elapsed = Number(m[1])
    let parsed
    try { parsed = JSON.parse(j[1]) } catch { return false }
    return elapsed < 25000 && parsed.verdict === 'CORRIDOR RED' && parsed.attempts === 1
  }
  console.log(`  never-hangs (single attempt, ask=skip): ${fraction(hangResults, hangOk)} bounded under 25s, verdict RED, attempts=1`)
  check('never hangs: an eternally-unresolved dependency still resolves within ATTEMPT_TIMEOUT_MS (N=16)', hangResults.every(hangOk),
    detailOf(hangResults.find((r) => !hangOk(r))))
  check('never hangs: no run was killed by the spike\'s own outer timeout (would mean the wrapper\'s OWN bound failed)', hangResults.every((r) => !r.hang))

  // ── LEG 3: the retry offer is CAPPED, not unbounded — N=5 ───────────────
  // Same eternally-hung dependency, but `ask` always answers '' (try
  // again). Proves MAX_ATTEMPTS (2, bare-connection-check.mjs) caps the
  // total wait even when a person would keep saying "try again" forever.
  // Smaller N here — this is deterministic control-flow (does the loop
  // stop at 2), not a timing race, so N=16's purpose (catching an
  // occasional timing miss) does not apply the same way; N=5 is enough to
  // rule out a fluke.
  console.log('\n-- leg 3: retry offer is capped at MAX_ATTEMPTS, even under permanent failure (N=5) --')
  const hangRetrySrc = `
import { runConnectionCheck, _testHooks } from ${JSON.stringify(CONNCHECK_URL)}
_testHooks.checkDht = () => new Promise(() => {})
const t0 = Date.now()
const result = await runConnectionCheck({ write: (s) => console.log(String(s)), ask: async () => '' })
console.log('ELAPSED_MS:' + (Date.now() - t0))
console.log('RESULT_JSON:' + JSON.stringify(result))
if (typeof Bare !== 'undefined') Bare.exit(0)
`
  const hangRetryPath = fixture('hang-retry.mjs', hangRetrySrc)
  const hangRetryResults = await spawnBareParallel(hangRetryPath, { runs: 5, timeoutMs: 55000 })
  for (const r of hangRetryResults) allOutputs.push(r.stdout)
  const hangRetryOk = (r) => {
    if (r.hang || r.code !== 0) return false
    const m = r.stdout.match(/ELAPSED_MS:(\d+)/)
    const j = r.stdout.match(/RESULT_JSON:(\{.*\})/)
    if (!m || !j) return false
    const elapsed = Number(m[1])
    let parsed
    try { parsed = JSON.parse(j[1]) } catch { return false }
    return elapsed >= 38000 && elapsed < 48000 && parsed.verdict === 'CORRIDOR RED' && parsed.attempts === 2
  }
  console.log(`  retry cap (always says "try again"): ${fraction(hangRetryResults, hangRetryOk)} stopped at attempts=2, ~2x ATTEMPT_TIMEOUT_MS, never unbounded`)
  check('retry offer is bounded: MAX_ATTEMPTS caps the loop even if the human always asks to retry (N=5)', hangRetryResults.every(hangRetryOk),
    detailOf(hangRetryResults.find((r) => !hangRetryOk(r))))

  // ── LEG 4: never calls exit — proved by content, not asserted in prose ──
  // The fixture NEVER calls Bare.exit() except inside a setTimeout AFTER
  // runConnectionCheck has already returned. If runConnectionCheck (or
  // anything it calls) had exited the process internally, neither marker
  // below would ever print — this is a positive, content-based proof, not
  // an assumption.
  console.log('\n-- leg 4: never calls exit (proof by observed post-return liveness, N=5) --')
  const neverExitSrc = `
import { runConnectionCheck, _testHooks } from ${JSON.stringify(CONNCHECK_URL)}
_testHooks.checkDht = async (say) => {
  say('FAIL: DHT bootstrap unreachable -- synthetic negative control (never-exit probe)')
  return { dht: { firewalled: false, remoteAddress: () => null }, dhtOk: false, dhtTotal: 3, dhtReachable: 0 }
}
await runConnectionCheck({ write: (s) => console.log(String(s)), ask: async () => 'skip' })
console.log('STILL_ALIVE_AFTER_RETURN')
setTimeout(() => {
  console.log('STILL_ALIVE_AFTER_TIMEOUT')
  if (typeof Bare !== 'undefined') Bare.exit(0)
}, 500)
`
  const neverExitPath = fixture('never-exit.mjs', neverExitSrc)
  const neverExitResults = await spawnBareParallel(neverExitPath, { runs: 5, timeoutMs: 10000 })
  for (const r of neverExitResults) allOutputs.push(r.stdout)
  const neverExitOk = (r) => !r.hang && r.code === 0 && r.stdout.includes('STILL_ALIVE_AFTER_RETURN') && r.stdout.includes('STILL_ALIVE_AFTER_TIMEOUT')
  console.log(`  never-exit: ${fraction(neverExitResults, neverExitOk)} the host process survived past runConnectionCheck's return`)
  check('never calls exit: the process is provably still alive after runConnectionCheck() resolves (N=5)', neverExitResults.every(neverExitOk),
    detailOf(neverExitResults.find((r) => !neverExitOk(r))))

  // ── LEG 5: the REAL, live checks — no override, honest measurement ──────
  // Sequential (matches SC0_PORT_MAP.md's own two-process punch driver's
  // method and avoids several concurrent HyperDHT nodes on one machine
  // confounding the measurement). N=7 — matches SC0's own precedent for
  // this exact class of live-network leg, explicitly NOT claimed as a
  // rate (Rule 5: a rate needs N>=30). Measured honestly, not
  // retried-until-green, not silently lowered.
  console.log('\n-- leg 5: live DHT/NAT check, no override (N=7, measured not claimed) --')
  const liveSrc = `
import { runConnectionCheck } from ${JSON.stringify(CONNCHECK_URL)}
const result = await runConnectionCheck({ write: (s) => console.log(String(s)), ask: async () => 'skip' })
console.log('RESULT_JSON:' + JSON.stringify(result))
if (typeof Bare !== 'undefined') Bare.exit(0)
`
  const livePath = fixture('live.mjs', liveSrc)
  const liveIsSuccess = (stdout, stderr, code) => code === 0 && /"verdict":"CORRIDOR (GREEN|AMBER|RED)"/.test(stdout)
  const liveResult = await runSpawnPipe({ exe: BARE_EXE, scriptPath: livePath, cwd: MESH, runs: 7, timeoutMs: 30000, isSuccess: liveIsSuccess })
  console.log(`  ${formatResult('live check (harness-level: did it complete and return a well-formed result)', liveResult)}`)
  check('live leg: the wrapper completed and returned a well-formed result on every run (harness-level only — the CORRIDOR verdict itself is a measurement, not an assertion, below)', liveResult.ok === liveResult.runs,
    detailOf(liveResult.results.find((r) => r.outcome !== 'OK')))
  const verdictCounts = { GREEN: 0, AMBER: 0, RED: 0 }
  for (const r of liveResult.results) {
    const m = r.stdout.match(/"verdict":"CORRIDOR (GREEN|AMBER|RED)"/)
    if (m) verdictCounts[m[1]]++
    allOutputs.push(r.stdout)
  }
  console.log(`  MEASURED (N=7, too small to state a rate — Rule 5): CORRIDOR GREEN=${verdictCounts.GREEN} AMBER=${verdictCounts.AMBER} RED=${verdictCounts.RED} on this machine/network today`)

  // ── LEG 6: no raw 64-hex key anywhere in any output collected above ─────
  console.log('\n-- leg 6: no raw 64-hex key in any output collected so far --')
  const leaked = allOutputs.filter((s) => HEX64.test(s))
  check('no raw 64-hex key ever appears in the wrapper\'s output (same tripwire bare-probe.mjs\'s own self-test uses)', leaked.length === 0,
    leaked.length ? `${leaked.length} output(s) matched the hex64 pattern` : '')

  // ── MANDATED LEG: bare-probe.mjs --self-test, unregressed, via runSpawnPipe ──
  // Explicit instruction: use runSpawnPipe for this one. Wrinkle, measured
  // not assumed: bare.exe requires the entry SCRIPT before any flag meant
  // for that script -- `bare.exe kit/bare-probe.mjs --self-test` runs the
  // self-test; `bare.exe --self-test kit/bare-probe.mjs` prints "unknown
  // flag: self-test" and does nothing (checked directly this session).
  // runSpawnPipe always builds argv as `[...args, scriptPath]` (args
  // BEFORE scriptPath), so it cannot express "script, then a trailing
  // flag" on its own. Worked around with a tiny Node launcher — spawned BY
  // runSpawnPipe as its own "scriptPath" under `process.execPath`, which
  // immediately re-spawns the real target with `stdio: 'inherit'` so
  // runSpawnPipe's own pipe-reading listeners see the real bare-probe.mjs
  // output exactly as if they had spawned it directly. Still a real OS
  // pipe end to end; never a shell pipe. spawn-pipe-harness.mjs itself is
  // untouched (not this mission's file to change).
  console.log('\n-- MANDATED LEG: bare-probe.mjs --self-test unregressed, via runSpawnPipe (N=5) --')
  const launcherSrc = `
import { spawnSync } from 'node:child_process'
const r = spawnSync(${JSON.stringify(BARE_EXE)}, [${JSON.stringify(PROBE_PATH)}, '--self-test'], { stdio: 'inherit', cwd: ${JSON.stringify(MESH)} })
process.exit(r.status ?? 1)
`
  const launcherPath = fixture('selftest-launcher.mjs', launcherSrc)
  const selfTestIsSuccess = (stdout, stderr, code) => {
    const okCount = (stdout.match(/\[OK\]/g) || []).length
    return code === 0 && okCount === 15 && stdout.includes('SELF-TEST GREEN') && !/\[FAIL\]/.test(stdout)
  }
  const probeSelfTestResult = await runSpawnPipe({ exe: process.execPath, scriptPath: launcherPath, cwd: MESH, runs: 5, timeoutMs: 15000, isSuccess: selfTestIsSuccess })
  console.log(`  ${formatResult('bare-probe.mjs --self-test (via runSpawnPipe + launcher)', probeSelfTestResult)}`)
  check('bare-probe.mjs --self-test is unregressed by this file\'s additive `export` changes: 15/15 [OK] + SELF-TEST GREEN, every run (N=5)', probeSelfTestResult.ok === probeSelfTestResult.runs,
    detailOf(probeSelfTestResult.results.find((r) => r.outcome !== 'OK')))
  for (const r of probeSelfTestResult.results) allOutputs.push(r.stdout)
  const leakedInSelfTest = allOutputs.filter((s) => HEX64.test(s))
  check('no raw 64-hex key in bare-probe.mjs --self-test output either', leakedInSelfTest.length === 0)

  // ── spawn-pipe-harness.mjs's own selfTest() — required by the mission ───
  console.log('\n-- spawn-pipe-harness.mjs selfTest() --')
  const harnessSelfTest = await selfTest()
  for (const line of harnessSelfTest.detail) console.log(`  ${line}`)
  check('spawn-pipe-harness.mjs selfTest(): the shipped harness correctly distinguishes OK/HANG/TOTAL_LOSS/PARTIAL', harnessSelfTest.pass)
} finally {
  if (workDir) { try { rmSync(workDir, { recursive: true, force: true }) } catch { /* best-effort */ } }
}

console.log(`\n${checks} check(s), ${failures} failure(s).`)
console.log(failures === 0 ? '\nSC3B CONNECTION CHECK SPIKE GREEN' : `\nSC3B CONNECTION CHECK SPIKE RED (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
