// bare-registry-spike.mjs — SC-1 (Sealed Corridor campaign) gate: proves
// bare-guide.mjs actually reopens its room across a process restart instead
// of creating a fresh "kitchen table" every run (the exact gap
// PHASE3_GUIDE_REPORT.md §5 flagged). Modeled on bare-guide-spike.mjs's own
// layer 4 (the sealed, bare-pack'd kit, from a from-scratch directory) —
// same method, same harness, same "test the layer the client touches"
// discipline (CAMPAIGN_REPORT.md §4 Rule 3): this spike drives
// `bare.exe app.bundle` through a real spawned pipe, never an in-process
// call, never a shell pipe.
//
// USES mesh/host/spawn-pipe-harness.mjs, as instructed — no hand-rolled
// spawn logic. NODE-ONLY by construction (the harness itself is Node-only —
// see its own header): this spike is the PARENT half of the seam.
//
// THE REQUIRED GATE (FABLE_CAMPAIGN_SEALED_CORRIDOR.md §SC-1, verbatim):
//   "spike runs the sealed guide twice against one data dir: run 1 creates
//   + posts, run 2 must list the SAME room and read back run 1's message
//   (content-asserted). Negative control: point run 2 at a fresh data dir
//   -> must NOT find the room. N>=16 on the reopen leg (a persistence race
//   is plausible)."
//
// LAYERS:
//   1. Build the sealed kit once (kit/bare-guide-entry.mjs, same recipe as
//      bare-guide-spike.mjs's own layer 4) and sanity-check its manifest.
//   2. The reopen leg: N=16 independent cycles, EACH in its own fresh
//      hostile directory (mkdtempSync OUTSIDE the repo — hostile geography,
//      D5). Within one cycle, run 1 and run 2 share that SAME directory as
//      `cwd` (the guide's storage is CWD-relative — this is what "same
//      data dir" means for a process that never takes a --data flag; see
//      §4 below for how this was verified, not assumed).
//   3. Negative control A: the run-2-shaped script pointed at a FRESH
//      (never-run) directory must NOT find the room — must print "created
//      a new room" and must NOT contain run 1's message text.
//   4. Negative control B: spawn-pipe-harness.mjs's own selfTest().
//
// Run: node kit/bare-registry-spike.mjs   (or: npm run sc1spike)

import { runSpawnPipe, formatResult, selfTest } from '../host/spawn-pipe-harness.mjs'
import { mkdtempSync, rmSync, existsSync, cpSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'

const __dirname = dirname(fileURLToPath(import.meta.url))
const meshRoot = join(__dirname, '..')

let failures = 0
let checks = 0
function check(name, cond, detail = '') {
  checks++
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' -- ' + detail : ''}`) }
}

console.log('bare-registry-spike -- SC-1: rooms that survive the night, over a real spawned pipe\n')

function guideStdin(lines) { return lines.join('\n') + '\n' }

const DISTINCTIVE_MSG = 'sc1-registry-persistence-proof-3f8a1c'

// A run-1-shaped script: open the messenger, post the distinctive message,
// leave, close. Firewall offer answered 'skip' -- same convention as
// bare-guide-spike.mjs's own sealed-kit layer 4 (a bare Enter also works;
// 'skip' is the precedent this spike matches).
const RUN1_STDIN = guideStdin(['2', 'skip', DISTINCTIVE_MSG, '/exit', '5'])
// A run-2-shaped script: open the messenger (no post), list rooms (surfaces
// lastPreview, which is where a reopened room's last message shows up),
// leave, close.
const RUN2_STDIN = guideStdin(['2', 'skip', '/rooms', '/exit', '5'])

const RAW_HEX64 = /\b[0-9a-f]{64}\b/i

// NOTE: run 1's OWN stdout never echoes the posted message body back -- the
// guide only echoes a message via /rooms' lastPreview (see run2, below),
// and RUN1_STDIN deliberately does not call /rooms (that would make run 1
// indistinguishable from a script that tests /rooms rather than persistence
// across a restart). Run 1's success predicate is therefore: it actually
// posted (a real seq number), not that the text round-tripped -- the
// round-trip assertion belongs to run 2, which is the whole point of this
// gate.
function run1Success(stdout) {
  return stdout.includes('Welcome.')
    && stdout.includes('ASYMMFLOW MESH -- GUIDE (Bare)')
    && stdout.includes('(created a new room for this kit -- "kitchen table")')
    && /posted, seq \d+/.test(stdout)
    && stdout.includes('Goodbye -- this window is safe to close.')
}

// The positive reopen assertion: SAME room found again, run 1's message
// read back (via /rooms' lastPreview), and the create-path line must be
// ABSENT -- a run that silently re-created a fresh room would still print
// "Goodbye" and could still coincidentally contain leftover terminal noise,
// but it can never contain a message this process never posted.
function run2ReopenSuccess(stdout) {
  return stdout.includes('Welcome.')
    && stdout.includes('(found your earlier conversation again -- "kitchen table")')
    && stdout.includes(DISTINCTIVE_MSG)
    && !stdout.includes('(created a new room for this kit -- "kitchen table")')
    && !RAW_HEX64.test(stdout)
    && stdout.includes('Goodbye -- this window is safe to close.')
}

// The negative-control assertion: a run-2-shaped script against a directory
// that has never seen run 1 must NOT find a room, must NOT contain run 1's
// message, and must go through the create path instead.
function freshDirSuccess(stdout) {
  return stdout.includes('Welcome.')
    && stdout.includes('(created a new room for this kit -- "kitchen table")')
    && !stdout.includes('(found your earlier conversation again -- "kitchen table")')
    && !stdout.includes(DISTINCTIVE_MSG)
    && stdout.includes('Goodbye -- this window is safe to close.')
}

function assertNoHash(label, p) {
  check(`${label}: hostile dir path contains no "#" (merge-gate finding -- "#" breaks Bare addon resolution)`, !p.includes('#'), p)
}

// ── layer 1: build the sealed kit once ─────────────────────────────────────
console.log('-- layer 1: build the sealed kit (kit/bare-guide-entry.mjs) --')
let bundleDir = null
try {
  execFileSync(process.execPath, [join(meshRoot, 'kit', 'build-bare-kit.mjs'), '--entry=kit/bare-guide-entry.mjs'], { cwd: meshRoot, stdio: 'pipe' })
  bundleDir = join(meshRoot, 'kit', 'dist-bare')
  const requiredFiles = ['app.bundle', 'bare.exe']
  const built = requiredFiles.every((f) => existsSync(join(bundleDir, f)))
  check('build-bare-kit.mjs --entry=kit/bare-guide-entry.mjs produced app.bundle + bare.exe', built)
  check('dist/reducer.wasm is present in the sealed kit manifest (posting requires it)', existsSync(join(bundleDir, 'dist', 'reducer.wasm')))
} catch (err) {
  check('layer 1: sealed kit build did not throw', false, err?.message ?? String(err))
}

const activeDirs = []
function freshHostileDir(prefix) {
  const d = mkdtempSync(join(tmpdir(), prefix))
  activeDirs.push(d)
  return d
}
function cleanup(d) {
  try { rmSync(d, { recursive: true, force: true }) } catch { /* best-effort */ }
  const i = activeDirs.indexOf(d)
  if (i !== -1) activeDirs.splice(i, 1)
}

if (bundleDir && existsSync(join(bundleDir, 'app.bundle')) && existsSync(join(bundleDir, 'bare.exe'))) {
  // ── layer 2: the reopen leg — N>=16 independent cycles ──────────────────
  console.log('\n-- layer 2: the reopen leg (N=16 independent cycles, hostile geography) --')
  const N = 16
  const reopenAgg = { runs: 0, ok: 0, partial: 0, totalLoss: 0, hang: 0, results: [] }
  for (let i = 1; i <= N; i++) {
    const hostileDir = freshHostileDir(`sc1-cycle-${i}-`)
    assertNoHash(`cycle ${i}`, hostileDir)
    try {
      cpSync(bundleDir, hostileDir, { recursive: true })
      const exe = join(hostileDir, 'bare.exe')
      const scriptPath = join(hostileDir, 'app.bundle')

      const r1 = await runSpawnPipe({ exe, scriptPath, cwd: hostileDir, runs: 1, timeoutMs: 20000, stdin: RUN1_STDIN, isSuccess: run1Success })
      const r2 = await runSpawnPipe({ exe, scriptPath, cwd: hostileDir, runs: 1, timeoutMs: 20000, stdin: RUN2_STDIN, isSuccess: run2ReopenSuccess })

      reopenAgg.runs++
      const r1ok = r1.ok === 1
      const r2ok = r2.ok === 1
      const r1r = r1.results[0]
      const r2r = r2.results[0]
      if (r1r.hang || r2r.hang) {
        reopenAgg.hang++
        reopenAgg.results.push({ cycle: i, outcome: 'HANG', run1: r1r.outcome, run2: r2r.outcome })
      } else if (r1ok && r2ok) {
        reopenAgg.ok++
        reopenAgg.results.push({ cycle: i, outcome: 'OK' })
      } else {
        const anyOutput = (r1r.stdout || '').trim().length > 0 || (r2r.stdout || '').trim().length > 0
        if (anyOutput) reopenAgg.partial++
        else reopenAgg.totalLoss++
        reopenAgg.results.push({
          cycle: i,
          outcome: anyOutput ? 'PARTIAL' : 'TOTAL_LOSS',
          run1: r1r.outcome, run2: r2r.outcome,
          run1Tail: (r1r.stdout || '').slice(-300),
          run2Tail: (r2r.stdout || '').slice(-300),
        })
      }
    } finally {
      cleanup(hostileDir)
    }
  }
  console.log(`  ${formatResult('reopen leg (16 cycles)', reopenAgg)}`)
  check(`reopen leg: run 2 finds the SAME room and reads back run 1's message, all ${N} cycles`, reopenAgg.ok === N,
    JSON.stringify(reopenAgg.results.filter((r) => r.outcome !== 'OK')).slice(0, 800))

  // ── layer 3: negative control A — fresh dir must NOT find the room ──────
  console.log('\n-- layer 3: negative control A (run-2-shaped script against a FRESH dir) --')
  const NEG_N = 5
  const negAgg = { runs: 0, ok: 0, partial: 0, totalLoss: 0, hang: 0, results: [] }
  for (let i = 1; i <= NEG_N; i++) {
    const hostileDir = freshHostileDir(`sc1-negctrl-${i}-`)
    assertNoHash(`negctrl ${i}`, hostileDir)
    try {
      cpSync(bundleDir, hostileDir, { recursive: true })
      const exe = join(hostileDir, 'bare.exe')
      const scriptPath = join(hostileDir, 'app.bundle')
      const result = await runSpawnPipe({ exe, scriptPath, cwd: hostileDir, runs: 1, timeoutMs: 20000, stdin: RUN2_STDIN, isSuccess: freshDirSuccess })
      negAgg.runs++
      negAgg.ok += result.ok
      negAgg.partial += result.partial
      negAgg.totalLoss += result.totalLoss
      negAgg.hang += result.hang
      negAgg.results.push({ cycle: i, outcome: result.results[0].outcome })
    } finally {
      cleanup(hostileDir)
    }
  }
  console.log(`  ${formatResult('negative control A (fresh dir)', negAgg)}`)
  check(`negative control A: a run-2-shaped script against a directory that never saw run 1 does NOT find a room (must go red if reopen always claims success) -- ${NEG_N}/${NEG_N}`, negAgg.ok === NEG_N,
    JSON.stringify(negAgg.results.filter((r) => r.outcome !== 'OK')).slice(0, 800))
} else {
  check('layer 2/3 skipped: sealed kit was not built (see layer 1 failure above)', false)
}

// ── layer 4: negative control B — the harness's own selfTest() ────────────
console.log('\n-- layer 4: negative control B (spawn-pipe-harness.mjs selfTest()) --')
const selfTestResult = await selfTest()
for (const line of selfTestResult.detail) console.log(`  ${line}`)
check('spawn-pipe-harness.mjs selfTest(): the shipped harness correctly distinguishes OK/HANG/TOTAL_LOSS/PARTIAL', selfTestResult.pass)

// ── cleanup any directory a thrown error left behind ───────────────────────
for (const d of [...activeDirs]) cleanup(d)

console.log(`\n${checks} check(s), ${failures} failure(s).`)
console.log(failures === 0 ? '\nSC1 REGISTRY SPIKE GREEN' : `\nSC1 REGISTRY SPIKE RED (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
