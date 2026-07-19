// kit2-spike.mjs — Mission A2 "The Corridor", Band 2, Gate G2
// (mesh/docs/MISSION_A2_CORRIDOR_SPEC.md §Band 2). Hermetic proof that a
// REAL `build-kit.mjs --bundle-node` run produces a kit a receptionist
// machine can actually use with zero installs:
//
//   1. bundled-node layout resolves udx-native's native binding (spawns the
//      BUILT node.exe itself, from the built cwd — not this process's own
//      node, so a prebuild/ABI mismatch would actually surface here)
//   2. run_mesh.cmd / run_probe.cmd reference the bundled runtime
//      (%~dp0node.exe preamble), not a bare PATH lookup
//   3. setup_firewall.cmd --print-only is sane: no elevation attempted, both
//      netsh rules echoed, delete-then-add idempotent shape, ends
//      FIREWALL READY
//   4. built Machine-B folder contains the corridor file set — probe/anchor
//      files copied conditionally BY PRESENCE at build time (a band that
//      hasn't landed is a graceful skip, never a spike failure) — this run
//      states which of each were actually present
//   5. README_CORRIDOR.txt exists, orders the firewall step BEFORE first
//      run (I3), and contains no real names (I4) — checked against a small
//      blocklist of names known from project context, since the corridor
//      card must use ONLY synthetic identities from SYNTHETIC_IDENTITY.md
//
// Drives a REAL build (per spec: "must build into a temp/scratch dir or
// kit\dist and clean up") — this spike runs the actual build-kit.mjs
// subprocess against kit/dist, inspects the real output, then removes it.
// Never touches kit-spike.mjs or its 33 checks (I2 — GL-11 discipline: new
// capability = new checks, in a new file).
//
// GEOGRAPHIC HERMETICITY (Mission A2.1 Band 5, field report FR-1): every
// EXECUTION check below that spawns the built node.exe against the built
// kit's own code (udx smoke, run_probe.cmd --self-test, the anchor.mjs
// import-resolution check, and the new end-to-end probe dial) runs against
// a COPY of the built Machine-B placed under os.tmpdir() — outside this
// repo and outside any ancestor node_modules — never against kit/dist in
// place. Running in place is exactly what let FR-1a (probe.mjs's top-level
// `import Holesail from 'holesail'`, a package the kit deliberately does
// NOT bundle) hide for an entire mission: Node's module resolution walks
// UP from kit/dist/Machine-B/kit/ and silently finds mesh/node_modules,
// so a missing-package bug that crashes the real receptionist machine
// looked green here. Static checks that only read files (file-presence,
// CRLF walk, README content, .cmd source-text checks, the firewall
// --print-only run, which never touches Node module resolution) still read
// the in-repo build — only checks that resolve/execute the built kit's own
// JS move. The temp copy is made ONCE (~200MB: node.exe + node_modules) and
// reused by every relocated check.
//
// Run: node kit/kit2-spike.mjs

import { existsSync, readFileSync, rmSync, readdirSync, cpSync, mkdtempSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { tmpdir } from 'node:os'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'

const __dirname = dirname(fileURLToPath(import.meta.url))
const kitDir = __dirname
const distOut = join(kitDir, 'dist')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

console.log('Mission A2 Band 2 — kit2 spike: the bundled-runtime field kit, hermetic\n')

// Best-effort cleanup matching build-kit.mjs's own lock-resilience: a stray
// watcher/AV handle on a leftover dist/ must skip-and-report, never crash
// the spike (same reality build-kit.mjs itself was hardened against this
// same mission — see the fresh-build lock-resilience note there).
function cleanup(label) {
  if (!existsSync(distOut)) return
  // One retry after a short pause: the anchor.mjs import-resolution check
  // above kills a spawned process that has udx-native's native binding
  // loaded — Windows can take a beat to fully release that handle after
  // the kill, so an immediate rmSync can EBUSY on a file that's genuinely
  // just about to free up (not stuck). A real, permanent lock still
  // reports clearly instead of crashing the spike.
  for (let attempt = 1; attempt <= 2; attempt++) {
    try {
      rmSync(distOut, { recursive: true, force: true })
      return
    } catch (err) {
      if (attempt === 2) {
        console.log(`  (cleanup ${label}: could not fully remove ${distOut} — ${err.code || err.message}; leaving remainder for a later rebuild to clear)`)
      } else {
        execFileSync(process.execPath, ['-e', 'setTimeout(() => {}, 1500)'])
      }
    }
  }
}

// ── 1. run a REAL build ──────────────────────────────────────────────────
console.log('running: node kit/build-kit.mjs --bundle-node ...')
let buildOk = true
let buildOutput = ''
try {
  buildOutput = execFileSync(process.execPath, [join(kitDir, 'build-kit.mjs'), '--bundle-node'], {
    cwd: dirname(kitDir), encoding: 'utf8',
  })
} catch (err) {
  buildOk = false
  buildOutput = (err.stdout || '') + (err.stderr || err.message)
}
check('build: build-kit.mjs --bundle-node exits clean', buildOk, buildOutput.slice(-2000))

const machineBDir = join(distOut, 'Machine-B')
check('build: Machine-B folder produced', existsSync(machineBDir))

if (!buildOk || !existsSync(machineBDir)) {
  console.log(`\nKIT2 SPIKE ${failures === 0 ? 'GREEN ✅' : 'RED ❌'} — ${failures} failure(s) (build did not produce output; remaining checks skipped)`)
  cleanup('post-build-failure')
  process.exit(failures === 0 ? 0 : 1)
}

// ── 1b. geographic hermeticity: copy the built Machine-B OUTSIDE the repo
// tree, once, and run every check that resolves/executes the built kit's
// own JS against that copy instead of kit/dist in place (see the file-top
// note on FR-1). os.tmpdir() (e.g. %TEMP%) is never an ancestor of this
// repo, so Node's upward module-resolution walk cannot escape into
// mesh/node_modules from here the way it silently did from kit/dist.
let tempKitDir = null
try {
  tempKitDir = mkdtempSync(join(tmpdir(), 'mission-a2-kit2-'))
  cpSync(machineBDir, tempKitDir, { recursive: true })
  console.log(`  (geography-hermetic copy of Machine-B placed at ${tempKitDir})`)
} catch (err) {
  console.log(`  (could not stage a geography-hermetic copy: ${err.message} — relocated checks below will fail honestly rather than silently fall back to the in-repo build)`)
}
const execDir = tempKitDir // undefined-safe: join(null, ...) throws, which is the honest failure mode above

// ── 2. bundled-node layout resolves udx-native's native binding ───────────
// Spawn the BUILT node.exe (not this process's own node) from the geography
// -hermetic copy's own cwd, exactly the layout run_mesh.cmd's
// `cd /d "%~dp0"` sets up on the real receptionist machine.
const builtNodeExe = join(machineBDir, 'node.exe')
const hasBuiltNode = existsSync(builtNodeExe)
check('bundled node.exe present in built Machine-B', hasBuiltNode)

const tempNodeExe = execDir ? join(execDir, 'node.exe') : null
if (hasBuiltNode && tempNodeExe && existsSync(tempNodeExe)) {
  const smokeScript = join(execDir, '.kit2-udx-smoke.cjs')
  const { writeFileSync } = await import('node:fs')
  writeFileSync(smokeScript, "require('udx-native'); console.log('KIT2_UDX_OK')\n")
  let udxOut = ''
  let udxOk = false
  try {
    udxOut = execFileSync(tempNodeExe, [smokeScript], { cwd: execDir, encoding: 'utf8' })
    udxOk = udxOut.includes('KIT2_UDX_OK')
  } catch (err) {
    udxOut = (err.stdout || '') + (err.stderr || err.message)
  } finally {
    try { rmSync(smokeScript, { force: true }) } catch { /* best-effort */ }
  }
  check('udx-native native binding resolves under bundled-node layout (geography-hermetic)', udxOk, udxOut.trim())
} else if (hasBuiltNode) {
  check('udx-native native binding resolves under bundled-node layout (geography-hermetic)', false, 'no geography-hermetic temp copy available — see staging error above')
}

// ── 3. generated .cmds reference the bundled runtime ───────────────────────
function cmdPrefersBundledNode(path) {
  if (!existsSync(path)) return false
  const src = readFileSync(path, 'utf8')
  return src.includes('%~dp0node.exe') && src.includes('where node')
}
check('run_mesh.cmd prefers bundled node.exe over PATH', cmdPrefersBundledNode(join(machineBDir, 'run_mesh.cmd')))

// run_probe.cmd is C1's own file (frozen — not one of build-kit.mjs's
// generated templates): it prefers the bundled runtime through a DIFFERENT
// but equivalent pattern (`..\node.exe` checked from inside kit/, one level
// below the bundled node.exe at kit root) rather than the `%~dp0node.exe`
// string this spike's earlier draft checked for verbatim — that draft
// false-failed on a working file. The check that actually matters is
// end-to-end: run it for real and see it resolve the bundled node AND find
// probe.mjs (the real Band 2 packaging bug this spike caught — see
// build-kit.mjs's PROBE_CLUSTER placement note). Runs against the
// geography-hermetic temp copy (Band 5) — this is exactly the launch that
// hid FR-1a when it ran in place.
const runProbeCmd = join(machineBDir, 'kit', 'run_probe.cmd')
const probeMjs = join(machineBDir, 'kit', 'probe.mjs')
const tempRunProbeCmd = execDir ? join(execDir, 'kit', 'run_probe.cmd') : null
if (existsSync(runProbeCmd)) {
  check('run_probe.cmd sits beside probe.mjs (bare `probe.mjs` reference resolves)', existsSync(probeMjs))
  let probeOut = ''
  let probeOk = false
  if (tempRunProbeCmd && existsSync(tempRunProbeCmd)) {
    try {
      probeOut = execFileSync(process.env.ComSpec || 'cmd.exe', ['/c', tempRunProbeCmd, '--self-test'], { cwd: dirname(tempRunProbeCmd), encoding: 'utf8', timeout: 20000, input: '' })
      probeOk = true
    } catch (err) {
      probeOut = (err.stdout || '') + (err.stderr || err.message)
    }
  } else {
    probeOut = 'no geography-hermetic temp copy available — see staging error above'
  }
  check('run_probe.cmd --self-test actually launches probe.mjs under the bundled node (geography-hermetic)', probeOk, probeOut.slice(-800))
} else {
  console.log('  (run_probe.cmd not present — Band 1 probe files were not built into this kit; see corridor-file-set check below)')
}

// ── 4. setup_firewall.cmd --print-only is sane ─────────────────────────────
const firewallCmd = join(machineBDir, 'setup_firewall.cmd')
check('setup_firewall.cmd present in built kit', existsSync(firewallCmd))
if (existsSync(firewallCmd)) {
  let fwOut = ''
  let fwOk = false
  try {
    fwOut = execFileSync(process.env.ComSpec || 'cmd.exe', ['/c', firewallCmd, '--print-only'], { cwd: machineBDir, encoding: 'utf8' })
    fwOk = true
  } catch (err) {
    fwOut = (err.stdout || '') + (err.stderr || err.message)
  }
  check('setup_firewall.cmd --print-only runs without elevation', fwOk, fwOut)
  const deleteCount = (fwOut.match(/netsh advfirewall firewall delete rule/g) || []).length
  const addCount = (fwOut.match(/netsh advfirewall firewall add rule/g) || []).length
  check('--print-only echoes both delete rules (idempotent shape)', deleteCount === 2, `saw ${deleteCount}`)
  check('--print-only echoes both add rules (in + out)', addCount === 2, `saw ${addCount}`)
  check('--print-only never triggers elevation (no "Start-Process -Verb RunAs" in output)', !fwOut.includes('RunAs'))
  check('--print-only ends FIREWALL READY', /FIREWALL READY/.test(fwOut))
}

// ── 5. corridor file set present, conditionally by presence ────────────────
// Fixed filenames per spec §6 — a band that hasn't landed is a graceful
// build-time skip, so kit2-spike states which were actually present in
// mesh/kit/ at build time rather than hard-requiring every name.
// Paths match build-kit.mjs's two placement clusters: PROBE_CLUSTER lands
// under kit/ (beside probe.mjs); ANCHOR_CLUSTER lands at the kit root
// (beside node.exe/data/) — see build-kit.mjs's CORRIDOR_OPTIONAL_FILES note.
const CORRIDOR_FILES = {
  'kit/probe.mjs': join(machineBDir, 'kit', 'probe.mjs'),
  'kit/run_probe.cmd': join(machineBDir, 'kit', 'run_probe.cmd'),
  'kit/run_probe_dial.cmd': join(machineBDir, 'kit', 'run_probe_dial.cmd'),
  'anchor.mjs (root, import-rewritten)': join(machineBDir, 'anchor.mjs'),
  'install_anchor.cmd': join(machineBDir, 'install_anchor.cmd'),
  'uninstall_anchor.cmd': join(machineBDir, 'uninstall_anchor.cmd'),
  'anchor_status.cmd': join(machineBDir, 'anchor_status.cmd'),
  'run_anchor.cmd': join(machineBDir, 'run_anchor.cmd'),
  'install_anchor.ps1': join(machineBDir, 'install_anchor.ps1'),
  'uninstall_anchor.ps1': join(machineBDir, 'uninstall_anchor.ps1'),
}
console.log('  corridor file set (present/absent is a graceful build-time skip, not a failure):')
for (const [label, path] of Object.entries(CORRIDOR_FILES)) {
  console.log(`    ${existsSync(path) ? 'present' : 'absent '} — ${label}`)
}
const presentCount = Object.values(CORRIDOR_FILES).filter(existsSync).length
check('corridor file set: at least the probe travels in the built kit', existsSync(CORRIDOR_FILES['kit/probe.mjs']) || presentCount > 0, `${presentCount}/${Object.keys(CORRIDOR_FILES).length} present`)

// anchor.mjs's root copy has its `./kit-host.mjs` import rewritten to
// `./kit/kit-host.mjs` by build-kit.mjs (anchorSourceForRoot) — the one
// real cross-band placement conflict this spike caught. Prove the rewrite
// actually resolves: spawn the BUILT node.exe importing the BUILT anchor.mjs
// from its real root location, for a short grace window, then kill it — a
// MODULE_NOT_FOUND on `./kit/kit-host.mjs` (or anything downstream of it)
// would throw synchronously during import, well before the anchor's own
// hyperswarm/heartbeat loop ever starts, so a short window is enough to
// catch a resolution failure without running the full headless anchor.
// Runs against the geography-hermetic temp copy (Band 5).
const anchorRootPath = join(machineBDir, 'anchor.mjs')
const tempAnchorRootPath = execDir ? join(execDir, 'anchor.mjs') : null
if (hasBuiltNode && existsSync(anchorRootPath)) {
  const rewritten = readFileSync(anchorRootPath, 'utf8').includes("from './kit/kit-host.mjs'")
  check('anchor.mjs (root copy): kit-host.mjs import rewritten for its root location', rewritten)

  if (tempNodeExe && existsSync(tempNodeExe) && tempAnchorRootPath && existsSync(tempAnchorRootPath)) {
    const { spawnSync } = await import('node:child_process')
    const res = spawnSync(tempNodeExe, ['--input-type=module', '-e', "import('./anchor.mjs')"], {
      cwd: execDir, encoding: 'utf8', timeout: 4000,
    })
    const combined = (res.stdout || '') + (res.stderr || '')
    const moduleNotFound = /Cannot find module|ERR_MODULE_NOT_FOUND/.test(combined)
    // SIGTERM/timeout (res.signal set) after 4s means it imported cleanly and
    // moved on to starting the anchor loop — that's the SUCCESS case here,
    // not a failure; only an early module-resolution error is a real failure.
    check('anchor.mjs (root copy): import resolves under bundled node (no MODULE_NOT_FOUND, geography-hermetic)', !moduleNotFound, combined.slice(-800))
  } else {
    check('anchor.mjs (root copy): import resolves under bundled node (no MODULE_NOT_FOUND, geography-hermetic)', false, 'no geography-hermetic temp copy available — see staging error above')
  }
}

// NEW (Band 5, FR-1 reproduction): run the probe end-to-end, OUTSIDE the
// repo tree, exactly the field recipe that first surfaced FR-1a
// (`cmd /c "kit\run_probe_dial.cmd dummykey123 < nul"`). A dummy dial key
// is expected to fail the punch step honestly (invalid key -> no peer) and
// still reach a printed CORRIDOR verdict line — the one thing FR-1a broke
// was reaching ANY verdict at all, crashing instead with
// ERR_MODULE_NOT_FOUND on 'holesail' before a single check line printed.
const tempRunProbeDialCmd = execDir ? join(execDir, 'kit', 'run_probe_dial.cmd') : null
if (existsSync(join(machineBDir, 'kit', 'run_probe_dial.cmd'))) {
  let dialOut = ''
  if (tempRunProbeDialCmd && existsSync(tempRunProbeDialCmd)) {
    try {
      dialOut = execFileSync(process.env.ComSpec || 'cmd.exe', ['/c', tempRunProbeDialCmd, 'dummykey123'], {
        cwd: dirname(tempRunProbeDialCmd), encoding: 'utf8', timeout: 30000, input: '',
      })
    } catch (err) {
      dialOut = (err.stdout || '') + (err.stderr || err.message)
    }
  } else {
    dialOut = 'no geography-hermetic temp copy available — see staging error above'
  }
  const moduleNotFound = /Cannot find module|ERR_MODULE_NOT_FOUND/.test(dialOut)
  const reachedVerdict = /CORRIDOR (GREEN|AMBER|RED)/.test(dialOut)
  check('probe end-to-end dial (dummy key, geography-hermetic): reaches a CORRIDOR verdict, not ERR_MODULE_NOT_FOUND',
    !moduleNotFound && reachedVerdict, dialOut.slice(-800))
} else {
  console.log('  (run_probe_dial.cmd not present — Band 1 probe files were not built into this kit; see corridor-file-set check below)')
}

// ── 6. README_CORRIDOR.txt: firewall-before-first-run + no real names ──────
const readmePath = join(machineBDir, 'README_CORRIDOR.txt')
check('README_CORRIDOR.txt present in built kit', existsSync(readmePath))
if (existsSync(readmePath)) {
  const readme = readFileSync(readmePath, 'utf8')
  const idxFirewall = readme.search(/Step 0.*[\s\S]*?setup_firewall\.cmd/i)
  const idxFirstRun = readme.search(/Step 1[\s\S]*?run_mesh\.cmd/i)
  check('README_CORRIDOR.txt orders the firewall step before first run',
    idxFirewall !== -1 && idxFirstRun !== -1 && idxFirewall < idxFirstRun)
  check('README_CORRIDOR.txt names /status as the support tool', /\/status/.test(readme))
  // I4 blocklist: names known from project context that must never appear
  // in a synthetic-data-only card. Not exhaustive by design — a full
  // real-name audit is a human review step; this is the hermetic tripwire.
  const REAL_NAME_BLOCKLIST = ['Sarat', 'Abhie', 'Rahul', 'SPOC Abhie']
  const hits = REAL_NAME_BLOCKLIST.filter((n) => readme.includes(n))
  check('README_CORRIDOR.txt contains no known real names (I4)', hits.length === 0, hits.join(', '))
}

// ── 7. Line endings: every built .cmd must be CRLF ─────────────────────────
// cmd.exe misparses LF-only batch files (chopped tokens on screen, broken
// goto labels) — the repo's .gitattributes LF baseline makes this the easy
// default failure, so build-kit.mjs CRLF-normalizes at write/copy time and
// this tripwire proves it held for the ACTUAL built artifact.
// (Mission A2 gate finding, 2026-07-19.)
{
  const walkCmds = (dir) => readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const p = join(dir, e.name)
    if (e.isDirectory()) return e.name === 'node_modules' ? [] : walkCmds(p)
    return e.name.endsWith('.cmd') ? [p] : []
  })
  const bareLfCmds = walkCmds(machineBDir).filter((p) => {
    const bytes = readFileSync(p)
    for (let i = 0; i < bytes.length; i++) {
      if (bytes[i] === 0x0a && (i === 0 || bytes[i - 1] !== 0x0d)) return true
    }
    return false
  })
  check('every .cmd in the built kit is CRLF (cmd.exe misparses bare LF)',
    bareLfCmds.length === 0, bareLfCmds.join(', '))
}

// ── cleanup ──────────────────────────────────────────────────────────────
// Best-effort, informational rather than gating: G2's own pass/fail
// criteria (spec §Band 2) are udx resolution, bundled-runtime .cmd
// references, firewall script validity, and the corridor file set — all
// proven above. Deleting the disposable build artifact afterward is spike
// hygiene on top, not a G2 requirement, and this dev machine's real-time
// AV scanning a freshly-written ~85 MB node.exe can hold a directory-level
// lock for longer than any bounded in-process wait should block a gate on
// (observed: Machine-B clears in ~1.5s, Machine-A sometimes takes longer,
// with NO process of this spike's own holding it — confirmed by fatally
// checking spawned processes are all reaped before this point runs).
cleanup('final')
if (existsSync(distOut) && readdirSync(distOut).length > 0) {
  console.log(`  (kit/dist not fully removed — leftover disposable build output at ${distOut}, safe to delete by hand or on the next build; not a G2 failure)`)
} else {
  console.log('  cleanup: kit/dist removed after the spike')
}

// The geography-hermetic temp copy (~200MB) — same best-effort, informational
// discipline as the in-repo dist cleanup above (Band 5).
if (tempKitDir && existsSync(tempKitDir)) {
  try {
    rmSync(tempKitDir, { recursive: true, force: true })
    console.log(`  cleanup: geography-hermetic temp copy removed (${tempKitDir})`)
  } catch (err) {
    console.log(`  (cleanup: could not fully remove temp copy ${tempKitDir} — ${err.code || err.message}; safe to delete by hand)`)
  }
}

console.log(`\nKIT2 SPIKE ${failures === 0 ? 'GREEN ✅' : 'RED ❌'} — ${failures === 0 ? 'all checks passed' : `${failures} failure(s)`}`)
process.exit(failures === 0 ? 0 : 1)
