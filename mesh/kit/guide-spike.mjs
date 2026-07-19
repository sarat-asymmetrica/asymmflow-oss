// guide-spike.mjs — Mission A2.1 "Reception Grade", Band 6, Gate G7
// (mesh/docs/MISSION_A2_CORRIDOR_SPEC.md addendum §Band 6). Hermetic proof
// that the guided path (guide.mjs + START_HERE.cmd + the restructured
// README_CORRIDOR.txt) actually works for a receptionist who never sees a
// command line (owner ruling R6):
//
//   0. pure-function checks: normalizeCode()'s WhatsApp-paste tolerance
//      (spaces from reading in groups of four, newlines from a chat-bubble
//      copy-paste) — no process, no network.
//   1. a REAL `build-kit.mjs --bundle-node` run (same as kit2-spike.mjs).
//   2. GEOGRAPHY-HERMETIC relocation: the built Machine-B is copied to a
//      fresh os.tmpdir() directory OUTSIDE this repo before any of it is
//      EXECUTED — this is the FR-1b lesson (mesh/docs/MISSION_A2_CORRIDOR_
//      SPEC.md's Mission A2.1 addendum): every earlier spike ran the built
//      kit from inside the repo, where Node's module resolution silently
//      escapes upward into mesh/node_modules and masks a genuinely missing
//      package. A leak here fails for real, the way it did in the field.
//   3. guide.mjs launched under the bundled node with SCRIPTED stdin: the
//      menu appears and option 5 exits cleanly.
//   4. the connection-check path (option 1) reaches a real probe.mjs launch
//      with a dummy dial code — CORRIDOR RED on a dummy key IS success here
//      (it proves the plumbing: menu -> prompt -> spawn -> probe runs ->
//      verdict flows back and gets re-printed large), never a fabricated
//      probe result.
//   5. README_CORRIDOR.txt leads with START_HERE (Band 6 item 6) and the
//      old ceremony survives intact in its appendix.
//   6. START_HERE.cmd is CRLF (cmd.exe misparses bare LF) and references
//      the bundled node.exe.
//   7. I4 real-name blocklist over guide.mjs's own source (its only source
//      of user-visible strings — nothing here is templated with a name at
//      runtime, unlike README_CORRIDOR_TEXT).
//
// Never touches kit-spike.mjs, kit2-spike.mjs, or probe.mjs (I2/single-writer
// — those are C1/C3's own gates and files this wave). Keeps the in-repo
// build step (kit/dist), matching build-kit.mjs's normal output location;
// only EXECUTION relocates outside the repo, per the FR-1b fix's own scope.
//
// Run: node kit/guide-spike.mjs

import { existsSync, readFileSync, rmSync, mkdtempSync, cpSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync, spawnSync } from 'node:child_process'

const __dirname = dirname(fileURLToPath(import.meta.url))
const kitDir = __dirname
const meshRoot = join(kitDir, '..')
const distOut = join(kitDir, 'dist')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

console.log('Mission A2.1 Band 6 — guide spike: the guided path, geography-hermetic\n')

// ── 0. pure helpers — no process, no network (guide.mjs has zero npm
//    imports, see its own file header, so importing it here is safe and
//    side-effect-free: `isMain` is false when it's the import target rather
//    than process.argv[1], so runGuide() never fires just from this import) ──
const { normalizeCode, groupInFours } = await import('./guide.mjs')

check('normalizeCode: strips ordinary spaces from a groups-of-four read-aloud code',
  normalizeCode('a1b2 c3d4 e5f6') === 'a1b2c3d4e5f6')
check('normalizeCode: strips newlines from a multi-line/chat-bubble paste',
  normalizeCode('a1b2c3d4\ne5f6\n') === 'a1b2c3d4e5f6')
check('normalizeCode: strips tabs and leading/trailing whitespace',
  normalizeCode('  a1b2\tc3d4  ') === 'a1b2c3d4')
check('normalizeCode: strips a non-breaking space (U+00A0) and a zero-width space (U+200B)',
  normalizeCode('a1b2 c3d4\u200Be5f6') === 'a1b2c3d4e5f6')
check('normalizeCode: never throws on empty/undefined input, returns ""',
  normalizeCode(undefined) === '' && normalizeCode('') === '' && normalizeCode(null) === '')
check('groupInFours: re-groups a normalized code back into fours for reading aloud',
  groupInFours('a1b2c3d4e5f6') === 'a1b2 c3d4 e5f6')

// ── 1. a REAL build ─────────────────────────────────────────────────────
console.log('\nrunning: node kit/build-kit.mjs --bundle-node ...')
let buildOk = true
let buildOutput = ''
try {
  buildOutput = execFileSync(process.execPath, [join(kitDir, 'build-kit.mjs'), '--bundle-node'], {
    cwd: meshRoot, encoding: 'utf8',
  })
} catch (err) {
  buildOk = false
  buildOutput = (err.stdout || '') + (err.stderr || err.message)
}
check('build: build-kit.mjs --bundle-node exits clean', buildOk, buildOutput.slice(-2000))

const builtMachineB = join(distOut, 'Machine-B')
check('build: Machine-B folder produced', existsSync(builtMachineB))

if (!buildOk || !existsSync(builtMachineB)) {
  console.log(`\nGUIDE SPIKE ${failures === 0 ? 'GREEN ✅' : 'RED ❌'} — ${failures} failure(s) (build did not produce output; remaining checks skipped)`)
  process.exit(failures === 0 ? 0 : 1)
}

// Same lock-resilience posture as kit2-spike.mjs's own cleanup: best-effort,
// never crashes the spike on a stray AV/watcher handle.
function rmBestEffort(path, label) {
  try { rmSync(path, { recursive: true, force: true }) } catch (err) {
    console.log(`  (cleanup ${label}: could not fully remove ${path} — ${err.code || err.message}; safe to delete by hand later)`)
  }
}

// ── 2. geography-hermetic relocation (FR-1b fix) ────────────────────────
const tmpRoot = mkdtempSync(join(tmpdir(), 'asymm-guide-spike-'))
const machineB = join(tmpRoot, 'Machine-B')
cpSync(builtMachineB, machineB, { recursive: true })
check('geography: built kit copied to a location outside this repo', existsSync(machineB))
check('geography: copy location is genuinely outside the repo tree (not a subpath of it)',
  !machineB.toLowerCase().startsWith(meshRoot.toLowerCase()))

// The in-repo build output has done its job (its own contents are now
// copied out) — remove it now so nothing below this line can accidentally
// execute the in-repo copy and mask a real leak the way FR-1a did.
rmBestEffort(distOut, 'in-repo dist/ (post-copy)')

const builtNodeExe = join(machineB, 'node.exe')
const hasBuiltNode = existsSync(builtNodeExe)
check('bundled node.exe present in the relocated built kit', hasBuiltNode)

const guideScript = join(machineB, 'kit', 'guide.mjs')
const hasGuideScript = existsSync(guideScript)
check('guide.mjs present in the relocated built kit (kit/ cluster, beside probe.mjs)', hasGuideScript)

/** Spawns the RELOCATED, bundled guide.mjs with scripted stdin — the exact
 * shape a receptionist's paste-then-Enter produces, fed all at once (this
 * is also what caught the readline `question()`-vs-piped-input race this
 * file's own development hit and fixed in guide.mjs's createGuideIO()). */
function runGuideScripted(scriptedInput, { timeout = 45000 } = {}) {
  if (!hasBuiltNode || !hasGuideScript) return { status: null, output: 'bundled node.exe or guide.mjs missing — see the checks above' }
  const res = spawnSync(builtNodeExe, [guideScript], { cwd: machineB, input: scriptedInput, encoding: 'utf8', timeout })
  const output = `${res.stdout || ''}${res.stderr || ''}${res.error ? `\n[spawn error] ${res.error.message}` : ''}`
  return { status: res.status, output }
}

// ── 3. menu appears, option 5 exits cleanly ─────────────────────────────
const exitRun = runGuideScripted('5\n', { timeout: 15000 })
check('guide.mjs: menu lists all five plain-word options under scripted stdin',
  ['[1] Check the connection', '[2] Open the messenger', '[3] Make this machine the always-on anchor', '[4] Show status', '[5] Close']
    .every((line) => exitRun.output.includes(line)),
  exitRun.output.slice(-500))
check('guide.mjs: option 5 exits cleanly (exit code 0)', exitRun.status === 0, `status=${exitRun.status}\n${exitRun.output.slice(-400)}`)
check('guide.mjs: prints the goodbye line on a clean exit', exitRun.output.includes('Goodbye'))

// ── 4. connection-check reaches a real probe launch ─────────────────────
const connRun = runGuideScripted('1\ndead beef\n5\n', { timeout: 65000 }) // above probe.mjs's own 58s watchdog ceiling
check('guide.mjs: connection-check option reaches probe.mjs (its own banner line appears)',
  connRun.output.includes('Mission A2 Band 1 — The Corridor Probe'), connRun.output.slice(-1500))
check('guide.mjs: a dummy dial code still reaches a verdict — CORRIDOR RED here is SUCCESS (proves the plumbing, not a fabricated result)',
  /CORRIDOR (GREEN|AMBER|RED)/.test(connRun.output), connRun.output.slice(-800))
check('guide.mjs: the verdict is re-printed large with the phone-read sentence',
  /Read this word to the person on the call: CORRIDOR (GREEN|AMBER|RED)/.test(connRun.output))
check('guide.mjs: WhatsApp-paste tolerance held for a spaced code typed at the real prompt ("dead beef" reads as one code, not two args)',
  connRun.output.includes('Trying to reach the other machine using code: dead beef'))
check('guide.mjs: the whole scripted session (menu -> check -> probe -> menu -> close) exits cleanly',
  connRun.status === 0, `status=${connRun.status}`)

// ── 5. README_CORRIDOR.txt leads with START_HERE ────────────────────────
const readmePath = join(machineB, 'README_CORRIDOR.txt')
const hasReadme = existsSync(readmePath)
check('README_CORRIDOR.txt present in the relocated built kit', hasReadme)
let readme = ''
if (hasReadme) {
  readme = readFileSync(readmePath, 'utf8')
  const idxStartHere = readme.indexOf('START HERE')
  const idxWhosWho = readme.indexOf("Who's who")
  const idxAppendix = readme.indexOf('APPENDIX')
  check('README_CORRIDOR.txt: leads with "Double-click START_HERE.cmd and follow the questions"',
    /Double-click START_HERE\.cmd and follow the questions/.test(readme))
  check('README_CORRIDOR.txt: the START HERE section appears before the old step-by-step content',
    idxStartHere !== -1 && idxWhosWho !== -1 && idxStartHere < idxWhosWho)
  check('README_CORRIDOR.txt: an appendix heading marks where the old ceremony was moved',
    idxAppendix !== -1 && idxStartHere < idxAppendix && idxAppendix < idxWhosWho)
  check('README_CORRIDOR.txt: the old ceremony survives INTACT in the appendix (Step 0 / setup_firewall.cmd still present)',
    idxAppendix !== -1 && /Step 0[\s\S]*?setup_firewall\.cmd/.test(readme.slice(idxAppendix)))
  check('README_CORRIDOR.txt: /status is still named as the universal support tool',
    /\/status/.test(readme))
}

// ── 6. START_HERE.cmd: CRLF, references the bundled node ────────────────
const startHerePath = join(machineB, 'START_HERE.cmd')
const hasStartHere = existsSync(startHerePath)
check('START_HERE.cmd present at the built kit root', hasStartHere)
let startHereText = ''
if (hasStartHere) {
  const bytes = readFileSync(startHerePath)
  let bareLf = false
  for (let i = 0; i < bytes.length; i++) {
    if (bytes[i] === 0x0a && (i === 0 || bytes[i - 1] !== 0x0d)) { bareLf = true; break }
  }
  check('START_HERE.cmd: every line ending is CRLF (cmd.exe misparses bare LF)', !bareLf)
  startHereText = bytes.toString('utf8')
  check('START_HERE.cmd: prefers the bundled node.exe over a bare PATH lookup', startHereText.includes('%~dp0node.exe') && startHereText.includes('where node'))
  check('START_HERE.cmd: launches kit\\guide.mjs (guide.mjs\'s own GUIDE_CLUSTER location)', /kit\\guide\.mjs/.test(startHereText))
}

// ── 7. I4 real-name blocklist ────────────────────────────────────────────
// guide.mjs never templates a name into its output at runtime (unlike
// README_CORRIDOR_TEXT's Jordan/Sam placeholders) — its own source is the
// complete, exhaustive source of every string it can ever print, so
// checking the source file covers every runtime output by construction.
// The two scripted runs' captured stdout are checked too, as a second,
// independent pass over what actually printed (belt-and-braces, not
// redundant — a future edit could add a templated string this source
// check alone wouldn't catch if it were built dynamically).
const REAL_NAME_BLOCKLIST = ['Sarat', 'Abhie', 'Rahul', 'SPOC Abhie']
const guideSource = readFileSync(guideScript, 'utf8')
const sourceHits = REAL_NAME_BLOCKLIST.filter((n) => guideSource.includes(n))
check('guide.mjs source contains no known real names (I4)', sourceHits.length === 0, sourceHits.join(', '))
const runtimeHits = REAL_NAME_BLOCKLIST.filter((n) => exitRun.output.includes(n) || connRun.output.includes(n))
check('guide.mjs scripted-run output contains no known real names (I4)', runtimeHits.length === 0, runtimeHits.join(', '))

// ── cleanup ──────────────────────────────────────────────────────────────
rmBestEffort(tmpRoot, 'geography-hermetic tmp copy')

console.log(`\nGUIDE SPIKE ${failures === 0 ? 'GREEN ✅' : 'RED ❌'} — ${failures === 0 ? 'all checks passed' : `${failures} failure(s)`}`)
process.exit(failures === 0 ? 0 : 1)
