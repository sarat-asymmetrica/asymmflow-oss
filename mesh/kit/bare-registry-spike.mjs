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
import { mkdtempSync, rmSync, existsSync, cpSync, mkdirSync, writeFileSync } from 'node:fs'
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

// Robustness finding (this coder, this file, gate-review round): a plain
// `cpSync(bundleDir, hostileDir, {recursive:true})` threw an uncaught
// `EINPROGRESS`/`EBUSY` on this machine under heavy concurrent `bare.exe`
// load (multiple coding agents spawning `bare.exe` at once -- see
// SC1_REPORT.md §5b) and crashed the ENTIRE spike process mid-run, losing
// every check result gathered so far. That is the exact "one bad cycle
// takes down everything else" shape this campaign's own rules forbid
// (kit-registry.mjs's "a corrupt registry must never crash boot", applied
// here to this GATE's own robustness, not just the guide's). A few retries
// with a short backoff is enough for a TRANSIENT Windows filesystem lock to
// clear; if it never clears, the caller still gets a real thrown error to
// catch and record as one failed cycle, not a crashed process.
async function robustCopy(src, dest, attempts = 4) {
  let lastErr
  for (let i = 1; i <= attempts; i++) {
    try { cpSync(src, dest, { recursive: true }); return } catch (err) {
      lastErr = err
      if (i < attempts) await new Promise((r) => setTimeout(r, 400 * i))
    }
  }
  throw lastErr
}

// ── layer 1: build the sealed kit once, into a PRIVATE output dir ─────────
// `--out=kit/.sc1-dist` (build-bare-kit.mjs's Sealed Corridor addition, the
// orchestrator's own fix): the default `kit/dist-bare` is a SHARED target —
// build-bare-kit.mjs unconditionally `rmSync`s its output dir before
// writing, so two coders building concurrently into the default dir can
// delete each other's kit mid-copy (observed once, see SC1_REPORT.md §5a).
// Built ONCE here, then copied (cheap) into each of the N hostile dirs below
// — the build itself (18 offloaded native addons) is not cheap enough to
// repeat 16+ times. `kit/.sc1-dist/` is already covered by .gitignore.
console.log('-- layer 1: build the sealed kit (kit/bare-guide-entry.mjs) --')
let bundleDir = null
try {
  execFileSync(process.execPath, [
    join(meshRoot, 'kit', 'build-bare-kit.mjs'),
    '--entry=kit/bare-guide-entry.mjs',
    '--out=kit/.sc1-dist',
  ], { cwd: meshRoot, stdio: 'pipe' })
  bundleDir = join(meshRoot, 'kit', '.sc1-dist')
  const requiredFiles = ['app.bundle', 'bare.exe']
  const built = requiredFiles.every((f) => existsSync(join(bundleDir, f)))
  check('build-bare-kit.mjs --entry=kit/bare-guide-entry.mjs --out=kit/.sc1-dist produced app.bundle + bare.exe', built)
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
      await robustCopy(bundleDir, hostileDir)
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
    } catch (err) {
      // A cycle that throws (e.g. a filesystem-level error that outlasted
      // robustCopy's retries) is recorded as ONE failed cycle, never a
      // crashed process -- see the robustCopy comment above for what this
      // is defending against, measured on this exact machine this round.
      reopenAgg.runs++
      reopenAgg.totalLoss++
      reopenAgg.results.push({ cycle: i, outcome: 'ERROR', error: err?.message ?? String(err) })
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
      await robustCopy(bundleDir, hostileDir)
      const exe = join(hostileDir, 'bare.exe')
      const scriptPath = join(hostileDir, 'app.bundle')
      const result = await runSpawnPipe({ exe, scriptPath, cwd: hostileDir, runs: 1, timeoutMs: 20000, stdin: RUN2_STDIN, isSuccess: freshDirSuccess })
      negAgg.runs++
      negAgg.ok += result.ok
      negAgg.partial += result.partial
      negAgg.totalLoss += result.totalLoss
      negAgg.hang += result.hang
      negAgg.results.push({ cycle: i, outcome: result.results[0].outcome })
    } catch (err) {
      negAgg.runs++
      negAgg.totalLoss++
      negAgg.results.push({ cycle: i, outcome: 'ERROR', error: err?.message ?? String(err) })
    } finally {
      cleanup(hostileDir)
    }
  }
  console.log(`  ${formatResult('negative control A (fresh dir)', negAgg)}`)
  check(`negative control A: a run-2-shaped script against a directory that never saw run 1 does NOT find a room (must go red if reopen always claims success) -- ${NEG_N}/${NEG_N}`, negAgg.ok === NEG_N,
    JSON.stringify(negAgg.results.filter((r) => r.outcome !== 'OK')).slice(0, 800))

  // ── layer 4: negative control C — a corrupt/dangling registry must never
  //    crash boot (kit-registry.mjs's own standard, ported by bare-registry.
  //    mjs -- exercised here for real, not just read-and-compared against
  //    the source). Three shapes, per the gate reviewer's explicit ask:
  //    malformed JSON, well-formed-but-not-an-array JSON, and a well-formed
  //    entry whose storage folder does not exist on disk (the realistic
  //    field case: someone deleted the folder by hand). The third one is
  //    the ONE this coder found does NOT already behave safely by accident
  //    -- see bare-guide.mjs's own comment at the fs.existsSync guard for
  //    what was actually observed (Corestore silently fabricates a phantom
  //    empty store instead of throwing) before this guard was added. ──────
  console.log('\n-- layer 4: negative control C (malformed / dangling rooms.json must never crash boot) --')
  const REGISTRY_N = 3
  const NOT_CRASHED_STDIN = guideStdin(['2', 'skip', '/exit', '5'])
  function notCrashedSuccess(stdout) {
    return stdout.includes('Welcome.')
      && stdout.includes('ASYMMFLOW MESH -- GUIDE (Bare)')
      && stdout.includes('Goodbye -- this window is safe to close.')
  }

  async function runRegistryScenario(label, prefix, seedRegistryText, n, extraCheck) {
    const agg = { runs: 0, ok: 0, partial: 0, totalLoss: 0, hang: 0, results: [] }
    for (let i = 1; i <= n; i++) {
      const hostileDir = freshHostileDir(`${prefix}-${i}-`)
      assertNoHash(`${label} ${i}`, hostileDir)
      try {
        await robustCopy(bundleDir, hostileDir)
        const keysDir = join(hostileDir, 'data', 'keys')
        mkdirSync(keysDir, { recursive: true })
        writeFileSync(join(keysDir, 'rooms.json'), seedRegistryText)
        const exe = join(hostileDir, 'bare.exe')
        const scriptPath = join(hostileDir, 'app.bundle')
        const result = await runSpawnPipe({ exe, scriptPath, cwd: hostileDir, runs: 1, timeoutMs: 20000, stdin: NOT_CRASHED_STDIN, isSuccess: notCrashedSuccess })
        agg.runs++
        agg.ok += result.ok
        agg.partial += result.partial
        agg.totalLoss += result.totalLoss
        agg.hang += result.hang
        const r = result.results[0]
        agg.results.push({ cycle: i, outcome: r.outcome, tail: r.outcome !== 'OK' ? (r.stdout || '').slice(-300) : undefined })
        if (extraCheck) extraCheck(r.stdout, i)
      } catch (err) {
        agg.runs++
        agg.totalLoss++
        agg.results.push({ cycle: i, outcome: 'ERROR', error: err?.message ?? String(err) })
      } finally {
        cleanup(hostileDir)
      }
    }
    console.log(`  ${formatResult(label, agg)}`)
    check(`${label}: guide reaches the menu and says Goodbye without crashing, ${n}/${n}`, agg.ok === n,
      JSON.stringify(agg.results.filter((r) => r.outcome !== 'OK')).slice(0, 800))
  }

  await runRegistryScenario('malformed JSON rooms.json', 'sc1-reg-badjson', '{ this is not valid json', REGISTRY_N)
  await runRegistryScenario('well-formed but non-array rooms.json', 'sc1-reg-nonarray', JSON.stringify({ not: 'an array' }), REGISTRY_N)
  await runRegistryScenario(
    'dangling entry (storage folder missing)', 'sc1-reg-dangling',
    JSON.stringify([{
      roomKey: 'a'.repeat(64), storage: 'nonexistent-storage-dir', authorityPub: 'b'.repeat(64),
      encryptionKey: null, bootstrap: null, title: 'ghost room',
    }]),
    REGISTRY_N,
    (stdout, i) => check(
      `dangling entry ${i}: guide reports the missing-storage-folder sentence instead of silently fabricating a phantom room`,
      stdout.includes('(could not reopen a saved room -- its storage folder is missing)'),
    ),
  )
} else {
  check('layer 2/3/4 skipped: sealed kit was not built (see layer 1 failure above)', false)
}

// ── layer 5: negative control D — the harness's own selfTest() ────────────
console.log('\n-- layer 5: negative control D (spawn-pipe-harness.mjs selfTest()) --')
const selfTestResult = await selfTest()
for (const line of selfTestResult.detail) console.log(`  ${line}`)
check('spawn-pipe-harness.mjs selfTest(): the shipped harness correctly distinguishes OK/HANG/TOTAL_LOSS/PARTIAL', selfTestResult.pass)

// ── cleanup any directory a thrown error left behind ───────────────────────
for (const d of [...activeDirs]) cleanup(d)

console.log(`\n${checks} check(s), ${failures} failure(s).`)
console.log(failures === 0 ? '\nSC1 REGISTRY SPIKE GREEN' : `\nSC1 REGISTRY SPIKE RED (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
