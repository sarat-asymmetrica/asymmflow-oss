// sealed-corridor-gate.mjs — the Sealed Corridor campaign's INDEPENDENT final
// gate (SC-5). Written by the orchestrator, not by any mission's coder, and
// deliberately NOT built on any mission's spike: SC-5's charter is "re-verify
// independently (own driver, own negative controls, honest N)". A gate that
// re-runs the same spike the coder wrote proves the coder ran it, not that the
// thing works.
//
// WHAT MAKES THIS DIFFERENT FROM EVERY MISSION SPIKE, and why it is not
// duplication:
//
//   RULE 3 — "test the layer the client touches" (CAMPAIGN_REPORT.md §4).
//   The campaign spec states it explicitly for this wave: "final gates drive
//   `run_bare_mesh.cmd` through a real spawned pipe, never `bare.exe
//   app.bundle`, never a shell pipe." EVERY spike in the tree today —
//   bare-guide-spike.mjs layer 4, bare-registry-spike.mjs, bare-net-spike.mjs
//   — drives `bare.exe app.bundle` directly. That is one layer BELOW what a
//   client double-clicks. The launcher is not a formality: it does `cd /d
//   "%~dp0"` (which is what makes the kit's CWD-relative `./data/...` storage
//   land inside the kit folder rather than wherever the shell happened to be),
//   it captures and propagates the exit code around `pause`, and it is the
//   file that has historically been mis-driven — three false kit-launcher
//   failures in the Sealed Ship campaign were Git Bash mangling `cmd.exe /c`.
//   This file closes that gap by spawning the .cmd itself.
//
//   The launcher's own `pause` is skipped via ASYMMFLOW_KIT_NONINTERACTIVE —
//   a DOCUMENTED opt-in on the same file, same code path (build-bare-kit.mjs's
//   own comment explains why that beats substituting a different command).
//
// NODE-ONLY by construction, like spawn-pipe-harness.mjs: this is the PARENT
// half of the seam. It is never imported by, or shipped inside, anything that
// runs AS the Bare child.
//
// NEGATIVE CONTROLS ARE RUN FIRST, ALWAYS. Rule 1: a probe that cannot report
// the opposite result proves nothing. If any control in `provingGroundsRed()`
// fails to go red, this file refuses to report ANY positive result at all —
// it does not merely note it and continue.
//
// Run: node kit/sealed-corridor-gate.mjs [--runs N] [--keep]

import { spawn } from 'node:child_process'
import { mkdtempSync, rmSync, existsSync, writeFileSync, mkdirSync, cpSync, readdirSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'

const __dirname = dirname(fileURLToPath(import.meta.url))
const meshRoot = join(__dirname, '..')
const OUT_DIR = 'kit/.sc4-dist'          // own output dir — never contends with a mission spike
const BUILT = join(meshRoot, OUT_DIR)

const argv = process.argv.slice(2)
const RUNS = (() => {
  const i = argv.indexOf('--runs')
  return i !== -1 && argv[i + 1] ? Number(argv[i + 1]) : 16
})()
const KEEP = argv.includes('--keep')

let checks = 0
let failures = 0
function check(name, cond, detail = '') {
  checks++
  if (cond) console.log(`  OK   ${name}`)
  else { failures++; console.log(`  FAIL ${name}${detail ? ' -- ' + detail : ''}`) }
  return cond
}

// ── driving the REAL launcher ─────────────────────────────────────────────
//
// `run_bare_mesh.cmd` is a batch file: it must be run by cmd.exe, and the
// PHASE3_KIT_REPORT.md §5d finding is that invoking it from a POSIX shell
// mangles the arguments. We therefore spawn `cmd.exe /c <abs path>` directly
// from Node with `shell: false` — no shell interprets anything, and the child
// gets a real OS pipe for stdio (never a shell pipe, which is the exact
// distinction that hid two real Bare bugs for a day).
function runLauncher({ kitDir, stdin, timeoutMs = 90000 }) {
  return new Promise((resolve) => {
    const comspec = process.env.ComSpec || 'C:\\Windows\\System32\\cmd.exe'
    const child = spawn(comspec, ['/c', join(kitDir, 'run_bare_mesh.cmd')], {
      cwd: kitDir,
      stdio: ['pipe', 'pipe', 'pipe'],
      shell: false,
      env: { ...process.env, ASYMMFLOW_KIT_NONINTERACTIVE: '1' },
    })
    let stdout = ''
    let stderr = ''
    let settled = false
    const t0 = Date.now()
    const timer = setTimeout(() => {
      if (settled) return
      settled = true
      try { child.kill('SIGKILL') } catch { /* already gone */ }
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
    child.stdin.write(stdin)
    child.stdin.end()
  })
}

const tmpDirs = []
function hostileDir(tag) {
  // The '#'-in-path defect (merge-gate finding 2026-07-20) breaks Bare addon
  // resolution. mkdtemp's suffix is random, so this is asserted rather than
  // assumed — a generated '#' would silently invalidate the whole run.
  const d = mkdtempSync(join(tmpdir(), `asymm-corridor-${tag}-`))
  if (d.includes('#')) throw new Error(`hostile dir contains '#', which breaks Bare addon resolution: ${d}`)
  tmpDirs.push(d)
  return d
}
// Copying a ~63 MB kit repeatedly, on a machine that may also be running other
// sealed-kit gates, hits transient Windows filesystem errors (EINPROGRESS,
// EBUSY, EPERM — antivirus and the OS both hold handles briefly). Observed
// live 2026-07-20: one `EINPROGRESS` aborted a whole gate run mid-leg.
//
// This retry is HARNESS ROBUSTNESS ONLY and it is important to be precise
// about what that means: it must never mask a failure of the thing under
// test. It does not, because it only ever retries SETUP (creating and
// populating a fresh directory) and never retries a ceremony run or
// reinterprets a ceremony result. A setup that cannot be completed after
// three attempts still throws.
function freshKit(tag) {
  let lastErr = null
  for (let attempt = 1; attempt <= 3; attempt++) {
    const d = hostileDir(tag)
    try {
      cpSync(BUILT, d, { recursive: true })
      return d
    } catch (err) {
      lastErr = err
      console.log(`  (setup retry ${attempt}/3 for ${tag}: ${err?.code ?? ''} ${err?.message ?? err})`)
    }
  }
  throw new Error(`could not stage a fresh kit for ${tag} after 3 attempts: ${lastErr?.message ?? lastErr}`)
}
function releaseKit(d) {
  // Free each staged kit AS SOON AS its run is done, not at the end of the
  // gate. Found by measurement, 2026-07-20: the first version accumulated
  // every staged copy until the final cleanup, and each copy is ~63 MB. By
  // the fourth leg-C fixture that is well over a gigabyte of live temp data,
  // and `cpSync` began failing with a reproducible `EINPROGRESS` on a QUIET
  // machine — reproducible at the same point across all three retries, which
  // is what distinguished it from the load-related flakiness above it.
  //
  // Worth stating precisely because the two failures looked similar and were
  // not: the earlier HANGs were caused by other processes loading the
  // machine (they vanished when the machine was quiet), while this one was
  // caused by THIS harness's own disk appetite (it survived the machine
  // going quiet). Same red, two different causes, isolated one at a time.
  if (KEEP) return
  try { rmSync(d, { recursive: true, force: true }) } catch { /* best-effort */ }
  const i = tmpDirs.indexOf(d)
  if (i !== -1) tmpDirs.splice(i, 1)
}
function cleanup() {
  if (KEEP) { console.log(`\n(--keep: left ${tmpDirs.length} temp dir(s) in place)`); return }
  for (const d of [...tmpDirs]) releaseKit(d)
}

const stdinScript = (lines) => lines.join('\r\n') + '\r\n'

// The message bodies are synthetic by construction (campaign invariant I4 —
// synthetic identities only, everywhere).
const MARK = 'sc5-independent-gate-marker'

// ── PROVING GROUNDS: the controls run FIRST and gate everything else ───────
async function provingGroundsRed() {
  console.log('\n== proving grounds: can this driver report a FAILURE? ==')
  let allRed = true

  // Control 1 — a kit whose app.bundle is corrupt. The launcher still runs,
  // bare.exe still starts, and the ceremony cannot happen. If this driver
  // calls that a pass, nothing it says afterwards means anything.
  {
    const kit = freshKit('ctl-corrupt')
    writeFileSync(join(kit, 'app.bundle'), 'this is not a bundle')
    const r = await runLauncher({ kitDir: kit, stdin: stdinScript(['2', 'skip', MARK, '/exit', '5']), timeoutMs: 45000 })
    const looksHealthy = r.stdout.includes('Goodbye') && /posted, seq \d+/.test(r.stdout)
    allRed = check('control 1: a corrupted app.bundle is NOT reported as a healthy ceremony', !looksHealthy,
      `stdout was ${JSON.stringify(r.stdout.slice(0, 200))}`) && allRed
    releaseKit(kit)
  }

  // Control 2 — the success predicate itself. A run that never posts must not
  // satisfy it. This is the predicate, not the kit, under test: the campaign's
  // most expensive lesson was a proof that passed because it asserted the
  // wrong thing (CAMPAIGN_REPORT.md §4).
  {
    const kit = freshKit('ctl-nopost')
    const r = await runLauncher({ kitDir: kit, stdin: stdinScript(['5']), timeoutMs: 45000 })
    const reachedMenu = r.stdout.includes('ASYMMFLOW MESH -- GUIDE (Bare)')
    const claimsPosted = /posted, seq \d+/.test(r.stdout) && r.stdout.includes(MARK)
    allRed = check('control 2: a run that closes without posting is NOT counted as a post', !claimsPosted) && allRed
    // Positive sub-assertion: the control kit is otherwise healthy, so a red
    // above is attributable to "did not post" and not to "kit is broken".
    // Without this, control 2 would pass for the wrong reason (Rule 2 —
    // vary one axis at a time).
    check('control 2: the same kit DOES still reach the menu (so the red above is about posting, not a broken kit)', reachedMenu)
    releaseKit(kit)
  }

  // Control 3 — the launcher itself is the thing under test in this file, so
  // prove a MISSING launcher is detected rather than silently skipped.
  {
    const kit = freshKit('ctl-nolauncher')
    rmSync(join(kit, 'run_bare_mesh.cmd'), { force: true })
    const r = await runLauncher({ kitDir: kit, stdin: stdinScript(['5']), timeoutMs: 30000 })
    const healthy = r.stdout.includes('Goodbye')
    allRed = check('control 3: a kit with no run_bare_mesh.cmd is NOT reported as healthy', !healthy) && allRed
    releaseKit(kit)
  }

  return allRed
}

// ── POSITIVE LEG A: the ceremony through the REAL launcher ────────────────
async function ceremonyThroughLauncher(runs) {
  console.log(`\n== leg A: the full ceremony through run_bare_mesh.cmd, N=${runs} ==`)
  let ok = 0, partial = 0, totalLoss = 0, hang = 0
  let firstBad = null
  for (let i = 1; i <= runs; i++) {
    const kit = freshKit(`legA-${i}`)
    const r = await runLauncher({ kitDir: kit, stdin: stdinScript(['2', 'skip', `${MARK}-${i}`, '/rooms', '/exit', '5']) })
    const good = r.stdout.includes('ASYMMFLOW MESH -- GUIDE (Bare)')
      && /posted, seq \d+/.test(r.stdout)
      && r.stdout.includes(`${MARK}-${i}`)
      && r.stdout.includes('Goodbye -- this window is safe to close.')
    if (r.hang) { hang++; if (!firstBad) firstBad = { i, why: 'HANG' } }
    else if (good) ok++
    else if (r.stdout.trim().length) { partial++; if (!firstBad) firstBad = { i, why: 'PARTIAL', out: r.stdout.slice(0, 300) } }
    else { totalLoss++; if (!firstBad) firstBad = { i, why: 'TOTAL_LOSS', err: r.stderr.slice(0, 300) } }
    releaseKit(kit)
  }
  console.log(`  leg A: OK=${ok}/${runs} PARTIAL=${partial}/${runs} TOTAL_LOSS=${totalLoss}/${runs} HANG=${hang}/${runs}`)
  check(`leg A: the sealed kit runs its full ceremony through the REAL launcher, ${runs}/${runs}`,
    ok === runs, firstBad ? JSON.stringify(firstBad) : '')
  return { ok, runs }
}

// ── POSITIVE LEG B: persistence across a restart, through the REAL launcher ─
// SC-1's own spike proved this through `bare.exe app.bundle`. This re-proves
// it one layer up, where the client actually lives — and the launcher's
// `cd /d "%~dp0"` is precisely what makes the CWD-relative data dir land in
// the kit folder, so this leg tests something the lower-layer spike could not.
async function persistenceThroughLauncher(runs) {
  console.log(`\n== leg B: room persistence across a restart, through run_bare_mesh.cmd, N=${runs} ==`)
  let ok = 0
  let firstBad = null
  for (let i = 1; i <= runs; i++) {
    const kit = freshKit(`legB-${i}`)
    const body = `${MARK}-persist-${i}`
    const r1 = await runLauncher({ kitDir: kit, stdin: stdinScript(['2', 'skip', body, '/exit', '5']) })
    const posted = /posted, seq \d+/.test(r1.stdout)
    // Second, entirely separate process, same kit folder.
    const r2 = await runLauncher({ kitDir: kit, stdin: stdinScript(['2', 'skip', '/rooms', '/exit', '5']) })
    const readBack = r2.stdout.includes(body)
    const didNotRecreate = !r2.stdout.includes('created a new room for this kit')
    // The data really landed inside the kit folder — this is the launcher's
    // `cd /d "%~dp0"` doing its job, and it is why this leg exists.
    const dataInKit = existsSync(join(kit, 'data', 'keys')) && existsSync(join(kit, 'data', 'corestore'))
    if (posted && readBack && didNotRecreate && dataInKit) ok++
    else if (!firstBad) firstBad = { i, posted, readBack, didNotRecreate, dataInKit, r2: r2.stdout.slice(0, 300) }
    releaseKit(kit)
  }
  console.log(`  leg B: OK=${ok}/${runs}`)
  check(`leg B: run 2 reopens run 1's room and reads its message back, ${runs}/${runs}`,
    ok === runs, firstBad ? JSON.stringify(firstBad) : '')
  return { ok, runs }
}

// ── POSITIVE LEG C: the registry cannot brick the kit ─────────────────────
// Independent of SC-1's own spike, deliberately: SC-5's charter is its own
// controls, and "a corrupt registry must never crash boot" is inherited
// verbatim from kit-registry.mjs as LAW. Law that is never executed is prose.
//
// Each fixture is written into a real kit's data/keys/rooms.json and the kit
// is then driven through the REAL launcher. The bar is deliberately low and
// absolute: whatever the registry says, the client must still reach the menu
// and still get a clean Goodbye. A kit that dies on a malformed JSON file is
// a kit that a field machine can brick by half-writing one file during a
// power cut.
const REGISTRY_FIXTURES = [
  ['invalid JSON', '{not json at all'],
  ['well-formed JSON that is not an array', '{"rooms":[]}'],
  ['an array of nonsense', '[1,2,3]'],
  ['an entry pointing at a storage folder that does not exist', JSON.stringify([{
    roomKey: 'a'.repeat(64), storage: 'room-that-was-deleted', authorityPub: 'b'.repeat(64),
    encryptionKey: null, bootstrap: null, title: 'gone',
  }], null, 2)],
  ['an entry with a malformed encryptionKey', JSON.stringify([{
    roomKey: 'c'.repeat(64), storage: 'room-guide', authorityPub: 'd'.repeat(64),
    encryptionKey: 'not-hex-at-all', bootstrap: null, title: 'bad key',
  }], null, 2)],
  ['an empty file', ''],
]

async function registryRobustness(runsPer) {
  console.log(`\n== leg C: a broken registry must not brick the kit, N=${runsPer} per fixture ==`)
  for (const [label, contents] of REGISTRY_FIXTURES) {
    let ok = 0
    let firstBad = null
    for (let i = 1; i <= runsPer; i++) {
      const kit = freshKit('legC')
      mkdirSync(join(kit, 'data', 'keys'), { recursive: true })
      writeFileSync(join(kit, 'data', 'keys', 'rooms.json'), contents)
      // 150s, not 60s. The first version of this leg used 60s and produced
      // two HANGs with COMPLETELY EMPTY stdout — which is NOT the signature
      // of a kit wedged mid-ceremony (that leaves partial output); it is the
      // signature of a process that had not yet produced its first byte.
      // That run was made while three other sealed-kit gates were saturating
      // the same machine, so TWO axes had changed at once (Rule 2) and the
      // red was not attributable. The timeout is raised here so the leg
      // measures the kit rather than the machine's load; a genuine wedge
      // still fails, it just is not confused with a slow cold start.
      const r = await runLauncher({ kitDir: kit, stdin: stdinScript(['2', 'skip', '/exit', '5']), timeoutMs: 150000 })
      const survived = r.stdout.includes('ASYMMFLOW MESH -- GUIDE (Bare)')
        && r.stdout.includes('Goodbye -- this window is safe to close.')
        && !r.hang
      if (survived) ok++
      else if (!firstBad) {
        firstBad = {
          i, hang: r.hang, ms: r.ms,
          // An empty stdout on a hang is diagnostically different from a
          // truncated one — surfaced explicitly so a future reader is not
          // left guessing which shape they are looking at.
          stdoutEmpty: r.stdout.trim().length === 0,
          out: r.stdout.slice(0, 300), err: r.stderr.slice(0, 200),
        }
      }
      releaseKit(kit)
    }
    check(`leg C: ${label} -- kit still reaches the menu and closes cleanly (${ok}/${runsPer})`,
      ok === runsPer, firstBad ? JSON.stringify(firstBad) : '')
  }
}

// ── main ──────────────────────────────────────────────────────────────────
console.log('sealed-corridor-gate -- SC-5 independent verification (own driver, own controls)\n')

try {
  console.log(`building the sealed kit into ${OUT_DIR} (entry: kit/bare-guide-entry.mjs)...`)
  execFileSync(process.execPath, [
    join(meshRoot, 'kit', 'build-bare-kit.mjs'),
    '--entry=kit/bare-guide-entry.mjs',
    `--out=${OUT_DIR}`,
    // The guide reaches these through menu [1]'s DYNAMIC import of
    // bare-connection-check.mjs (dynamic because a static one makes the guide
    // un-loadable under Node — see bare-guide.mjs's checkConnection for the
    // measurement). bare-pack DOES follow that dynamic specifier today,
    // verified 2026-07-20. This flag is what stops that from being a silent
    // assumption: if bare-pack ever stops resolving it, the BUILD fails here
    // rather than the field failing quietly with a kit that renders its whole
    // ceremony and can never reach the network.
    '--require-addons=udx-native,sodium-native,bare-tcp,bare-dns',
  ], { cwd: meshRoot, stdio: 'pipe' })

  check('build: app.bundle produced', existsSync(join(BUILT, 'app.bundle')))
  check('build: bare.exe copied into the kit', existsSync(join(BUILT, 'bare.exe')))
  check('build: run_bare_mesh.cmd produced (the layer this gate exists to drive)', existsSync(join(BUILT, 'run_bare_mesh.cmd')))
  check('build: dist/reducer.wasm offloaded (without it the kit renders everything and silently cannot post)',
    existsSync(join(BUILT, 'dist', 'reducer.wasm')))

  const red = await provingGroundsRed()
  if (!red) {
    console.log('\nREFUSING TO REPORT POSITIVE RESULTS: at least one negative control did not go red.')
    console.log('A driver that cannot report failure proves nothing by succeeding (Rule 1).')
    console.log(`\n${checks} check(s), ${failures} failure(s).`)
    console.log('\nSEALED CORRIDOR GATE RED (controls)')
    cleanup()
    process.exit(1)
  }
  console.log('  (all controls went red as required -- positive results below are admissible)')

  await ceremonyThroughLauncher(RUNS)
  await persistenceThroughLauncher(Math.min(RUNS, 16))
  // N=3 per fixture, deliberately: these are CORRECTNESS proofs (does a
  // malformed file crash the kit — yes or no), not rate measurements, so
  // Rule 5's N>=16 does not apply. Six fixtures at N=3 is eighteen launcher
  // runs; stated here so the smaller N is a declared choice rather than an
  // unexplained shortfall.
  await registryRobustness(3)
} catch (err) {
  check('gate did not throw', false, err?.message ?? String(err))
} finally {
  cleanup()
}

console.log(`\n${checks} check(s), ${failures} failure(s).`)
console.log(failures === 0 ? '\nSEALED CORRIDOR GATE GREEN' : `\nSEALED CORRIDOR GATE RED (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
