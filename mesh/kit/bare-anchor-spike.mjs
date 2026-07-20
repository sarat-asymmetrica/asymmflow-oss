// bare-anchor-spike.mjs — Phase 3 gate for bare-anchor.mjs.
//
// ARCHITECTURE NOTE (a real mistake caught building this, kept visible
// rather than silently fixed): bare-anchor.mjs is Bare-ONLY by construction
// (top-level `import fs from 'bare-fs'`/`import process from 'bare-process'`
// — native-binding packages that do not resolve under plain Node, confirmed
// this session: `node -e "require('bare-crypto')"` throws `TypeError:
// require.addon is not a function`, the Bare-only native-hook bare-fs/
// bare-process/bare-crypto all share). spawn-pipe-harness.mjs (Phase 2, this
// coder's own file) is Node-ONLY by construction (`node:child_process`
// etc.), because it is the PARENT half of a spawn pipe. A single file cannot
// both `import { anchorMain } from './bare-anchor.mjs'` directly AND
// `import { runSpawnPipe } from '../host/spawn-pipe-harness.mjs'` — one
// would always fail to resolve under whichever runtime actually ran this
// file. The fix, and the reason this file is a NODE file: generate small
// Bare FIXTURE scripts (functions below, written to a real file under
// mesh/kit/ so Bare's module resolution finds mesh/node_modules — the same
// lesson stdio-seam-spike.mjs already paid for, PHASE0_NOTES_D_REVERIFY.md
// §4), each importing bare-anchor.mjs's anchorMain and exercising it for
// real, printing ONE `RESULT:<name>:<PASS|FAIL>[:<detail>]` line per check
// via `console.log` (RULE 2) before an explicit `process.exit()` (RULE 3).
// This spike then spawns each fixture through spawn-pipe-harness.mjs's
// runSpawnPipe — a REAL OS pipe (RULE 4) — and asserts on the RESULT lines
// it reads back, never on exit code alone (spawn-pipe-harness.mjs's own
// design law #1). This mirrors stdio-seam-spike.mjs's own fixture-generation
// pattern exactly, applied to a different module under test.
//
// WHAT THIS PROVES vs WHAT IT DOES NOT (see bare-anchor.mjs's own header):
// bare-anchor.mjs's actual room-hosting `boot()` is not wired anywhere yet —
// that needs host/bare-bridge.mjs, another coder's reserved, in-flight file.
// This spike therefore proves the LOOP CONTRACT (boot, heartbeat format
// I7-identical to anchor.mjs, boot-failure retry+recovery, tick() wiring,
// explicit-shutdown-only discipline, real close()) against a REAL (not
// trivial) fixture boot function that does genuine async work and file I/O
// — and separately proves the REAL, UNMODIFIED bare-anchor.mjs CLI entry
// point's own honest "no boot wired yet" refusal, spawned for real. Full
// room-hosting itself remains unproven and is explicitly NOT claimed here.
//
// NEGATIVE CONTROL (explicit gate requirement, mirrors this coder's own
// spawn-pipe-harness.mjs selfTest()): the `negative` fixture below uses a
// boot() that NEVER succeeds and asserts the SAME "heartbeat appeared"
// check used by the positive fixture correctly reports it did NOT — proving
// this spike's checks can detect a broken anchor, not only confirm a
// healthy one.
//
// Run: node kit/bare-anchor-spike.mjs

import { mkdirSync, rmSync, writeFileSync, existsSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { runSpawnPipe, formatResult } from '../host/spawn-pipe-harness.mjs'

const __dirname = dirname(fileURLToPath(import.meta.url))
const BARE_EXE = join(__dirname, '..', 'node_modules', 'bare-runtime-win32-x64', 'bin', 'bare.exe')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  [OK] ${name}`)
  else { failures++; console.log(`  [FAIL] ${name}${detail ? ' — ' + detail : ''}`) }
}

console.log('Phase 3 — bare-anchor spike: loop contract (real spawn pipe) + negative control + real CLI check\n')

/** Runs each RESULT:<name>:<PASS|FAIL>[:<detail>] line from a fixture's
 * stdout through check(), so the Node-side spike's pass/fail ledger is
 * driven by what the Bare fixture actually observed, not by this file's
 * assumptions about it. Returns the number of RESULT lines seen (0 is
 * itself worth flagging — a fixture that printed nothing parseable). */
function absorbResults(stdout) {
  let seen = 0
  for (const rawLine of stdout.split('\n')) {
    // Bare's stdout on win32 is CRLF — a bug caught building this spike:
    // an early regex anchored `$` right after PASS/FAIL/detail with no `\r`
    // tolerance matched ZERO lines against real output, misreporting a
    // fully-successful run as "zero parseable RESULT lines" and, combined
    // with a debug print that only showed stdout.slice(0, 500), LOOKED
    // exactly like a genuine Bug-A-class truncation. It was not — the full
    // output was always present; only this parser's `$` anchor was wrong.
    // Keeping this comment rather than silently fixing it, per the
    // honesty law: a promising-looking "new flush-race finding" that turns
    // out to be a `\r` bug in the OBSERVER is itself worth recording.
    const line = rawLine.replace(/\r$/, '')
    const m = line.match(/^RESULT:(.+?):(PASS|FAIL)(?::(.*))?$/)
    if (!m) continue
    seen++
    check(m[1], m[2] === 'PASS', m[3] || '')
  }
  return seen
}

const FIXTURE_PRELUDE = `
import fs from 'bare-fs'
import process from 'bare-process'
import { anchorMain } from '../bare-anchor.mjs'

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)) }
async function raced(donePromise, ms) { return Promise.race([donePromise.then(() => true), sleep(ms).then(() => false)]) }
function result(name, pass, detail) { console.log('RESULT:' + name + ':' + (pass ? 'PASS' : 'FAIL') + (detail ? ':' + detail : '')) }

function makeFixtureBoot({ failFirstNBoots = 0 } = {}) {
  let bootAttempts = 0
  let closeCount = 0
  let peers = 0
  const boot = async (dataDir, actor, log) => {
    bootAttempts++
    if (bootAttempts <= failFirstNBoots) throw new Error('fixture boot failure #' + bootAttempts + ' (deliberate)')
    await sleep(20) // genuine async boundary — not a synchronous stub
    fs.mkdirSync(dataDir + '/keys', { recursive: true })
    fs.writeFileSync(dataDir + '/keys/fixture-boot-marker.txt', 'booted attempt #' + bootAttempts + '\\n')
    log('fixture boot #' + bootAttempts + ' complete')
    return {
      roomCount: 1,
      mode: 'fixture',
      totalPeers: () => peers,
      tick: () => { peers++ },
      close: async () => { closeCount++ },
    }
  }
  return { boot, bootAttempts: () => bootAttempts, closeCount: () => closeCount }
}
`

// dataDir is embedded as a JS string literal at generation time, not read
// from argv — spawn-pipe-harness.mjs's runOnce() places `args` BEFORE the
// script path (spawn(exe, [...args, scriptPath], ...)), which is the right
// shape for interpreter flags but means there is no argv slot after the
// script path for the fixture's own arguments; embedding avoids fighting
// that shape for a one-off value that never needs to vary at spawn time.
const positiveFixture = (dataDir) => FIXTURE_PRELUDE + `
const dataDir = ${JSON.stringify(dataDir)}
const fixture = makeFixtureBoot()
let liveSession
const { done, requestShutdown } = anchorMain({
  dataDir, actor: 'anchor', heartbeatMs: 200, boot: fixture.boot,
  onBoot: (s) => { liveSession = s },
  log: () => {},
})

const heartbeatPath = dataDir + '/keys/anchor.log'
const d1 = Date.now() + 5000
while (!fs.existsSync(heartbeatPath) && Date.now() < d1) await sleep(50)
result('anchor booted and wrote a heartbeat file', fs.existsSync(heartbeatPath))

const firstLines = fs.existsSync(heartbeatPath) ? fs.readFileSync(heartbeatPath, 'utf8').trim().split('\\n').filter(Boolean) : []
result('heartbeat line format is timestamp+peers+rooms+mode, I7-identical to anchor.mjs',
  firstLines.length >= 1 && /^\\d{4}-\\d{2}-\\d{2}T.*Z peers=\\d+ rooms=\\d+ mode=fixture$/.test(firstLines[0]))

result('anchor never exited after boot (loop is alive)', !(await raced(done, 100)))

const d2 = Date.now() + 3000
while (fs.readFileSync(heartbeatPath, 'utf8').trim().split('\\n').filter(Boolean).length <= firstLines.length && Date.now() < d2) await sleep(50)
const secondLines = fs.readFileSync(heartbeatPath, 'utf8').trim().split('\\n').filter(Boolean)
result('heartbeat file advances on its own over time', secondLines.length > firstLines.length)

result('tick() was actually invoked by the heartbeat cadence (peers counter advanced)', liveSession && liveSession.totalPeers() > 0)

requestShutdown()
const shutdownOk = await raced(done, 3000)
result('anchor shuts down cleanly ONLY on an explicit requestShutdown() call', shutdownOk)
result('the fixture session was actually close()d on shutdown', fixture.closeCount() >= 1)
console.log('FIXTURE_DONE')
process.exit(0)
`

const retryFixture = (dataDir) => FIXTURE_PRELUDE + `
const dataDir = ${JSON.stringify(dataDir)}
const fixture = makeFixtureBoot({ failFirstNBoots: 2 })
const logs = []
const { done, requestShutdown } = anchorMain({
  dataDir, actor: 'anchor', heartbeatMs: 200, boot: fixture.boot,
  log: (m) => logs.push(m),
})

const dFail = Date.now() + 2000
while (!logs.some((l) => l.includes('fixture boot failure #1')) && Date.now() < dFail) await sleep(20)
result('a boot-time failure is logged (not swallowed) and does not crash the process', logs.some((l) => l.includes('boot/serve error')))

const heartbeatPath = dataDir + '/keys/anchor.log'
const dRecover = Date.now() + 20000 // covers anchorMain's own 5s + 10s backoff steps
while (!fs.existsSync(heartbeatPath) && Date.now() < dRecover) await sleep(200)
result('anchor RECOVERS after retrying past deliberate boot failures (heartbeat eventually appears)', fs.existsSync(heartbeatPath))
result('exactly 3 boot attempts occurred (2 deliberate failures + 1 success)', fixture.bootAttempts() === 3)

requestShutdown()
await raced(done, 3000)
console.log('FIXTURE_DONE')
process.exit(0)
`

const negativeFixture = (dataDir) => FIXTURE_PRELUDE + `
const dataDir = ${JSON.stringify(dataDir)}
const fixture = makeFixtureBoot({ failFirstNBoots: 999 })
const { done, requestShutdown } = anchorMain({
  dataDir, actor: 'anchor', heartbeatMs: 200, boot: fixture.boot,
  log: () => {},
})

const heartbeatPath = dataDir + '/keys/anchor.log'
const d = Date.now() + 3000 // well short of the 5s initial backoff — boot can never succeed in this window
while (!fs.existsSync(heartbeatPath) && Date.now() < d) await sleep(50)
const heartbeatNeverAppeared = !fs.existsSync(heartbeatPath)
result('NEGATIVE CONTROL: a permanently-broken boot() correctly produces NO heartbeat file',
  heartbeatNeverAppeared,
  heartbeatNeverAppeared ? '' : 'the heartbeat check reported healthy against a fixture that can never boot')
result('NEGATIVE CONTROL: the anchor is still alive (retrying) rather than crashing on permanent boot failure', !(await raced(done, 50)))
result('NEGATIVE CONTROL: at least one boot attempt was made and failed (fixture actually exercised)', fixture.bootAttempts() >= 1)

requestShutdown()
await raced(done, 3000)
console.log('FIXTURE_DONE')
process.exit(0)
`

async function runFixture(dir, name, source, timeoutMs) {
  const fixturePath = join(dir, name)
  writeFileSync(fixturePath, source)
  const result = await runSpawnPipe({
    exe: BARE_EXE,
    scriptPath: fixturePath,
    runs: 1,
    timeoutMs,
    isSuccess: (stdout) => stdout.includes('FIXTURE_DONE'),
  })
  // isSuccess only gates spawn-pipe-harness's own OK/PARTIAL/HANG bucket —
  // the actual pass/fail ledger comes from absorbResults() below, per
  // design law #1 (content, never exit code, and here not even a coarse
  // "did it look done" bucket either — the individual RESULT lines are what
  // matters).
  return result.results[0]
}

async function main() {
  if (!existsSync(BARE_EXE)) {
    console.error(`FATAL: bare.exe not found at ${BARE_EXE} — run "npm install" in mesh/ first.`)
    process.exit(1)
  }

  const dir = join(__dirname, '.bare-anchor-spike-tmp')
  rmSync(dir, { recursive: true, force: true })
  mkdirSync(dir, { recursive: true })

  try {
    console.log('=== positive: real fixture boot, full loop contract ===')
    const posDataDir = join(dir, 'data-positive').replace(/\\/g, '/')
    const pos = await runFixture(dir, 'positive.mjs', positiveFixture(posDataDir), 15000)
    const posSeen = absorbResults(pos.stdout)
    if (posSeen === 0) { failures++; console.log('  [FAIL] positive fixture produced zero parseable RESULT lines'); console.log('  raw stdout:', pos.stdout.slice(0, 500)); console.log('  raw stderr:', pos.stderr.slice(0, 500)) }

    console.log('\n=== retry: boot-time failure retried with backoff, then recovers ===')
    const retryDataDir = join(dir, 'data-retry').replace(/\\/g, '/')
    const retry = await runFixture(dir, 'retry.mjs', retryFixture(retryDataDir), 30000)
    const retrySeen = absorbResults(retry.stdout)
    if (retrySeen === 0) { failures++; console.log('  [FAIL] retry fixture produced zero parseable RESULT lines'); console.log('  raw stdout:', retry.stdout.slice(0, 500)); console.log('  raw stderr:', retry.stderr.slice(0, 500)) }

    console.log('\n=== negative control: permanently-broken boot, spike must detect it ===')
    const negDataDir = join(dir, 'data-negative').replace(/\\/g, '/')
    const neg = await runFixture(dir, 'negative.mjs', negativeFixture(negDataDir), 10000)
    const negSeen = absorbResults(neg.stdout)
    if (negSeen === 0) { failures++; console.log('  [FAIL] negative fixture produced zero parseable RESULT lines'); console.log('  raw stdout:', neg.stdout.slice(0, 500)); console.log('  raw stderr:', neg.stderr.slice(0, 500)) }

    console.log('\n=== §4 — RULE 4: the REAL, unmodified bare-anchor.mjs CLI entry point, through a real spawn pipe ===')
    const cliScript = join(__dirname, 'bare-anchor.mjs')
    const cliResult = await runSpawnPipe({
      exe: BARE_EXE,
      scriptPath: cliScript,
      runs: 5,
      timeoutMs: 8000,
      isSuccess: (stdout, stderr) => /no boot\(\) is wired into the CLI yet/.test(stderr) || /no boot\(\) is wired into the CLI yet/.test(stdout),
    })
    console.log('  ' + formatResult('bare-anchor.mjs CLI via real spawn pipe (no args)', cliResult))
    check('§4: the real CLI entry point, spawned through a real OS pipe, refuses with its documented message every run', cliResult.ok === 5)
  } finally {
    try { rmSync(dir, { recursive: true, force: true }) } catch { /* best-effort cleanup */ }
  }

  console.log(failures === 0 ? '\nBARE-ANCHOR SPIKE GREEN' : `\nBARE-ANCHOR SPIKE RED (${failures} failure(s))`)
  process.exit(failures === 0 ? 0 : 1)
}

await main()
