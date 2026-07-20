// spawn-pipe-harness.mjs — the permanent fix for how this campaign tests the
// Bare stdio seam, not just what it found broken (PHASE0_NOTES_D2_FLUSH_RACE.md).
//
// THE METHOD BUG THIS FILE CLOSES: `mesh/host/bare-spike/stdio-check.mjs` was
// marked PASS on 2026-07-19 by piping it with a shell (`printf '...' | npx
// bare stdio-check.mjs`). Under the real production topology — a Node parent
// spawning the sidecar with `stdio: ['pipe','pipe','pipe']` and reading
// `child.stdout.on('data', …)`, exactly how a DP4 bridge would drive it — the
// SAME unmodified script drops its entire reply payload 10/10 (100%), exit
// code 0, no hang, no error. A shell pipe and a `child_process` pipe are
// different consumers of the child's stdout handle on Windows; only one of
// them is what our real bridge will ever be. Testing the wrong one for a day
// is what let two real bugs (PHASE0_NOTES_D2_FLUSH_RACE.md §3, Bug A and Bug
// B) hide behind a green spike. This harness makes "tested with a shell pipe"
// structurally impossible to repeat: every run it drives goes through a real
// `child_process.spawn` pipe, full stop.
//
// NODE-ONLY, DELIBERATELY (stated up front per campaign instruction): this
// file uses `node:child_process`/`node:fs`/`node:path`/`node:url` — it is the
// PARENT half of the seam, the thing that spawns and reads the sidecar, which
// is legitimately a Node-side concern (the sealed artifact's host process is
// Node, not Bare — Bare is only ever the spawned child). This file must NEVER
// be `import`ed by, or shipped inside, anything destined to run AS the Bare
// child itself (bare-bridge.mjs, apply-bare.mjs, wasi-preview1-lite.mjs, the
// sealed artifact). It has no relationship to and does not touch any of those
// files.
//
// DESIGN LAW — read before adding a new caller:
//   1. Assert on OUTPUT CONTENT, never on exit code. Bug B's own signature
//      (PHASE0_NOTES_D2_FLUSH_RACE.md §1) is "exit 0, zero bytes lost
//      forever" — an exit-code check would have called that a pass.
//   2. Report MEASURED fractions ("7/30"), never an estimated/assumed rate.
//   3. Distinguish FOUR outcomes per run, not two: full success, PARTIAL
//      (some output arrived but the success predicate failed — a truncation,
//      not a total loss), TOTAL LOSS (process exited cleanly, zero relevant
//      bytes ever arrived), and HANG (the process never closed within the
//      timeout and was killed — Bug B's other face). Collapsing HANG into
//      "failure" loses exactly the distinction that made Bug B legible.
//   4. A harness with no negative control is exactly the shape of thing that
//      created this problem (five probes in this campaign have now returned
//      only the answer they were built to give). selfTest() below is not
//      decoration — CI-equivalent gates that import this file should run it
//      before trusting anything else in the file.

import { spawn } from 'node:child_process'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

/**
 * runSpawnPipe(options) — drive one script N times through a real spawned
 * pipe and categorize every run.
 *
 * @param {string} options.exe - executable to run (e.g. a resolved bare.exe
 *   path, or `process.execPath` to drive a Node script instead — the harness
 *   itself is runtime-agnostic; it only cares that the CHILD's stdio is a
 *   real OS pipe, not what interprets the script).
 * @param {string[]} [options.args] - extra args before the script path is
 *   appended; usually `[]`.
 * @param {string} options.scriptPath - absolute path to the script to run.
 * @param {string} [options.cwd] - child's cwd (defaults to scriptPath's dir).
 * @param {number} [options.runs=30]
 * @param {number} [options.timeoutMs=15000] - a run still open after this
 *   long is killed (SIGKILL) and counted as a HANG, never left to block the
 *   batch forever (Bug B deadlocks rather than failing fast — PHASE0_NOTES_D2
 *   §2, variants f/h/j).
 * @param {string|null} [options.stdin=null] - if non-null, written to the
 *   child's stdin and the stdin stream is then ended. If null, stdin is
 *   'ignore'd (the child gets no stdin pipe at all).
 * @param {(stdout: string, stderr: string, code: number|null) => boolean}
 *   options.isSuccess - REQUIRED. Content predicate for a full pass. Never
 *   inspect `code` as the sole criterion (design law #1) — `code` is passed
 *   through only so a predicate MAY use it as one signal among several.
 * @param {(stdout: string) => boolean} [options.isPartial] - optional finer
 *   classifier for "some output, not success" vs "success". If omitted, any
 *   non-timeout run that fails `isSuccess` but produced non-empty stdout is
 *   classified PARTIAL, and one that produced empty stdout is TOTAL_LOSS.
 * @returns {Promise<{runs, ok, partial, totalLoss, hang, results}>}
 */
export async function runSpawnPipe(options) {
  const {
    exe,
    args = [],
    scriptPath,
    cwd = dirname(scriptPath),
    runs = 30,
    timeoutMs = 15000,
    stdin = null,
    isSuccess,
    isPartial,
  } = options

  if (typeof isSuccess !== 'function') {
    throw new TypeError('runSpawnPipe: options.isSuccess(stdout, stderr, code) is required (design law #1 — never assert on exit code alone)')
  }

  const results = []
  for (let i = 1; i <= runs; i++) {
    results.push(await runOnce({ exe, args, scriptPath, cwd, timeoutMs, stdin }))
  }

  let ok = 0
  let partial = 0
  let totalLoss = 0
  let hang = 0
  for (const r of results) {
    if (r.hang) { hang++; r.outcome = 'HANG'; continue }
    if (isSuccess(r.stdout, r.stderr, r.code)) { ok++; r.outcome = 'OK'; continue }
    const trulyEmpty = r.stdout.trim().length === 0
    const partialHit = isPartial ? isPartial(r.stdout) : !trulyEmpty
    if (partialHit) { partial++; r.outcome = 'PARTIAL' }
    else { totalLoss++; r.outcome = 'TOTAL_LOSS' }
  }

  return { runs, ok, partial, totalLoss, hang, results }
}

function runOnce({ exe, args, scriptPath, cwd, timeoutMs, stdin }) {
  return new Promise((resolve) => {
    const stdioSpec = stdin === null ? ['ignore', 'pipe', 'pipe'] : ['pipe', 'pipe', 'pipe']
    const child = spawn(exe, [...args, scriptPath], { cwd, stdio: stdioSpec })

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

    if (stdin !== null) {
      child.stdin.write(stdin)
      child.stdin.end()
    }
  })
}

/** Formats a runSpawnPipe() result as the one-line "measured fraction"
 * summary every caller in this campaign should print — never hand-roll a
 * different phrasing that could drift toward an eyeballed rate. */
export function formatResult(label, result) {
  const { runs, ok, partial, totalLoss, hang } = result
  return `${label}: OK=${ok}/${runs} PARTIAL=${partial}/${runs} TOTAL_LOSS=${totalLoss}/${runs} HANG=${hang}/${runs}`
}

// ── self-test: proves the harness can go RED, not just green ──────────────
//
// Three deliberately broken fixture scripts, written to a real temp
// directory (never committed — generated at self-test time), each engineered
// to trip exactly one non-OK outcome. If any of the three is misreported as
// OK, or the good-control script is misreported as anything but OK, the
// self-test fails LOUD (throws / non-zero exit) rather than silently.

const GOOD_SCRIPT = `
process.stdout.write('READY\\n')
process.exit(0)
`

const HANG_SCRIPT = `
// never writes, never exits — parks on an interval that keeps the loop alive
setInterval(() => {}, 1000)
`

const TOTAL_LOSS_SCRIPT = `
// exits immediately without ever writing anything to stdout
process.exit(0)
`

const PARTIAL_SCRIPT = `
// writes SOMETHING, but not the expected marker — a stand-in for a
// truncated/wrong-content run (the observable shape of Bug A)
process.stdout.write('WRONG\\n')
process.exit(0)
`

/**
 * selfTest() — the harness's own negative control. Runs each fixture through
 * runSpawnPipe() under plain Node (process.execPath), asserts every fixture
 * lands in the outcome bucket it was built to hit, and throws with a precise
 * message naming which bucket misfired if not. Call this before trusting any
 * other result from this file — a harness that has never been proven capable
 * of reporting failure is exactly the class of tool that let Bug A and Bug B
 * hide for a day (PHASE0_NOTES_D2_FLUSH_RACE.md, campaign gate note).
 *
 * @param {number} [runsPerFixture=5] - kept small by default since this is a
 *   correctness proof of the harness's classification logic, not a
 *   statistical measurement of a real bug's rate (that's what
 *   stdio-seam-spike.mjs / the D2 matrix are for).
 * @returns {Promise<{pass: boolean, detail: string[]}>}
 */
export async function selfTest(runsPerFixture = 5) {
  const dir = mkdtempSync(join(tmpdir(), 'spawn-pipe-harness-selftest-'))
  const detail = []
  let pass = true

  try {
    const goodPath = join(dir, 'good.mjs')
    const hangPath = join(dir, 'hang.mjs')
    const lossPath = join(dir, 'loss.mjs')
    const partialPath = join(dir, 'partial.mjs')
    writeFileSync(goodPath, GOOD_SCRIPT)
    writeFileSync(hangPath, HANG_SCRIPT)
    writeFileSync(lossPath, TOTAL_LOSS_SCRIPT)
    writeFileSync(partialPath, PARTIAL_SCRIPT)

    const isSuccess = (stdout) => stdout.includes('READY')

    const goodResult = await runSpawnPipe({ exe: process.execPath, scriptPath: goodPath, runs: runsPerFixture, timeoutMs: 5000, isSuccess })
    detail.push(formatResult('selfTest good-control', goodResult))
    if (goodResult.ok !== runsPerFixture) { pass = false; detail.push(`FAIL: good-control script should be OK=${runsPerFixture}/${runsPerFixture}, got ${goodResult.ok}`) }

    const hangResult = await runSpawnPipe({ exe: process.execPath, scriptPath: hangPath, runs: runsPerFixture, timeoutMs: 3000, isSuccess })
    detail.push(formatResult('selfTest hang-fixture', hangResult))
    if (hangResult.hang !== runsPerFixture) { pass = false; detail.push(`FAIL: hang-fixture should be HANG=${runsPerFixture}/${runsPerFixture}, got ${hangResult.hang} (harness failed to detect a real hang — this is the Bug-B blind spot)`) }

    const lossResult = await runSpawnPipe({ exe: process.execPath, scriptPath: lossPath, runs: runsPerFixture, timeoutMs: 5000, isSuccess })
    detail.push(formatResult('selfTest total-loss-fixture', lossResult))
    if (lossResult.totalLoss !== runsPerFixture) { pass = false; detail.push(`FAIL: total-loss-fixture should be TOTAL_LOSS=${runsPerFixture}/${runsPerFixture}, got ${lossResult.totalLoss} (harness failed to detect a real total loss — this is the Bug-B-clean-exit blind spot)`) }

    const partialResult = await runSpawnPipe({ exe: process.execPath, scriptPath: partialPath, runs: runsPerFixture, timeoutMs: 5000, isSuccess })
    detail.push(formatResult('selfTest partial-fixture', partialResult))
    if (partialResult.partial !== runsPerFixture) { pass = false; detail.push(`FAIL: partial-fixture should be PARTIAL=${runsPerFixture}/${runsPerFixture}, got ${partialResult.partial} (harness failed to distinguish partial/truncated output from total loss)`) }
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }

  return { pass, detail }
}

// ── CLI entry point: `node spawn-pipe-harness.mjs --selftest` ─────────────
const isMain = process.argv[1] && __filename === process.argv[1]
if (isMain) {
  if (process.argv.includes('--selftest')) {
    const { pass, detail } = await selfTest()
    for (const line of detail) console.log(line)
    if (!pass) {
      console.error('\nspawn-pipe-harness selfTest: FAIL — the harness cannot currently be trusted to report a broken script as broken.')
      process.exit(1)
    }
    console.log('\nspawn-pipe-harness selfTest: PASS — good=OK, hang=HANG, total-loss=TOTAL_LOSS, partial=PARTIAL, all correctly distinguished.')
    process.exit(0)
  } else {
    console.log('Usage: node spawn-pipe-harness.mjs --selftest')
    console.log('This file is primarily a library (import { runSpawnPipe, selfTest, formatResult }) — see stdio-seam-spike.mjs for a real consumer.')
  }
}
