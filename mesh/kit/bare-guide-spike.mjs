// bare-guide-spike.mjs — Phase 3 gate: proves bare-guide.mjs's UX-law port
// end-to-end, driven the way a real client machine drives it (a real
// spawned process, real stdin/stdout pipes — never in-process function
// calls, per RULE 4/PHASE0_GATE_D2_FLUSH_RACE.md's own lesson: a green
// in-process run proves nothing about the seam).
//
// USES mesh/host/spawn-pipe-harness.mjs, as instructed — does not roll its
// own spawn logic (Phase 2's own bare-bridge-spike.mjs did, before this
// harness existed; this file is what "use it rather than rolling your own"
// looks like). NODE-ONLY by construction (the harness itself is — see its
// own header): this spike is the PARENT half of the seam, spawning
// bare-guide.mjs (a Bare-only file, see that file's header) as the child,
// under BOTH `process.execPath` (proving the guide ALSO runs under Node,
// useful for dev iteration even though the sealed kit only ever launches
// it under Bare) and the real `bare.exe` binary.
//
// THREE LAYERS:
//   1. Pure unit checks — `normalizeCode`/`groupInFours` imported directly
//      (matching guide-spike.mjs's own "no process" convention for these).
//   2. Real spawn-pipe scenarios — the full menu flow (messenger post +
//      list + exit + close) driven over a real pipe, both runtimes,
//      asserting on OUTPUT CONTENT per the harness's own design law #1
//      (never on exit code alone).
//   3. Negative controls — TWO of them: `spawn-pipe-harness.mjs`'s own
//      `selfTest()` (the shipped, proven "can this harness go red" check),
//      PLUS a guide-specific one: a deliberately broken FIXTURE COPY of
//      bare-guide.mjs (its closing "Goodbye" line removed, simulating an
//      incomplete/crashed run) driven through the SAME real spawn-pipe
//      path, asserting this spike's own success predicate correctly
//      flags it as NOT ok — the same standard Phase 2's bridge spike met
//      against `stdio-check.mjs`.
//
// Run: node kit/bare-guide-spike.mjs   (drives BOTH targets; this file
//      itself does not run under Bare — see header — Node is the only
//      runtime that needs to invoke it)

import { runSpawnPipe, formatResult, selfTest } from '../host/spawn-pipe-harness.mjs'
import { normalizeCode, groupInFours } from './bare-guide.mjs'
import { readFileSync, writeFileSync, mkdtempSync, rmSync, existsSync, cpSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'

const __dirname = dirname(fileURLToPath(import.meta.url))
const meshRoot = join(__dirname, '..')
const GUIDE_SCRIPT = join(__dirname, 'bare-guide.mjs')
const BARE_EXE = join(__dirname, '..', 'node_modules', 'bare-runtime-win32-x64', 'bin', 'bare.exe')

let failures = 0
let checks = 0
function check(name, cond, detail = '') {
  checks++
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' -- ' + detail : ''}`) }
}

console.log('bare-guide-spike -- the Guided Path, over a real spawned pipe, both runtimes\n')

// ── Layer 1: pure helpers, byte-for-byte the same as guide.mjs's own ──────
console.log('-- layer 1: pure helpers --')
check('normalizeCode: strips plain spaces', normalizeCode('a b c') === 'abc')
check('normalizeCode: strips non-breaking space (U+00A0)', normalizeCode('a b') === 'ab')
check('normalizeCode: strips zero-width space (U+200B)', normalizeCode('a​b') === 'ab')
check('normalizeCode: strips newlines/tabs (chat-paste artifacts)', normalizeCode('a\n\tb') === 'ab')
check('normalizeCode: never throws on null/undefined, normalizes to empty string', normalizeCode(null) === '' && normalizeCode(undefined) === '')
check('groupInFours: chunks into 4-character groups', groupInFours('abcdefgh') === 'abcd efgh')
check('groupInFours: a short code (< 4 chars) is left whole', groupInFours('ab') === 'ab')
check('groupInFours: an odd-length code groups the remainder', groupInFours('abcdefghi') === 'abcd efgh i')

// ── Layer 2: real spawn-pipe scenarios ─────────────────────────────────────
console.log('\n-- layer 2: real spawn-pipe scenarios --')

function guideStdin(lines) { return lines.join('\n') + '\n' }

// Full flow: open messenger (Enter past the firewall offer), post one
// message, list rooms, leave the messenger, close the guide.
const FULL_FLOW_STDIN = guideStdin(['2', '', 'a real spawned message', '/rooms', '/exit', '5'])
const fullFlowSuccess = (stdout) =>
  stdout.includes('Welcome.') &&
  stdout.includes('ASYMMFLOW MESH -- GUIDE (Bare)') &&
  /posted, seq \d+/.test(stdout) &&
  stdout.includes('a real spawned message') && // echoed back via /rooms' lastPreview
  stdout.includes('Goodbye -- this window is safe to close.')

async function runFullFlow(label, exe) {
  const cwd = mkdtempSync(join(tmpdir(), 'bare-guide-spike-'))
  try {
    const result = await runSpawnPipe({
      exe, scriptPath: GUIDE_SCRIPT, cwd, runs: 3, timeoutMs: 20000,
      stdin: FULL_FLOW_STDIN, isSuccess: fullFlowSuccess,
    })
    console.log(`  ${formatResult(label, result)}`)
    check(`${label}: full menu flow (open messenger, post, list, exit, close) -- all runs OK`, result.ok === result.runs,
      result.results.find((r) => r.outcome !== 'OK') ? `first non-OK: ${JSON.stringify(result.results.find((r) => r.outcome !== 'OK'))}`.slice(0, 400) : '')
  } finally {
    try { rmSync(cwd, { recursive: true, force: true }) } catch { /* best-effort */ }
  }
}

await runFullFlow('spawn(node)', process.execPath)
if (existsSync(BARE_EXE)) {
  await runFullFlow('spawn(bare)', BARE_EXE)
} else {
  check('spawn(bare): bare.exe found at the expected node_modules path', false, `not found at ${BARE_EXE}`)
}

// Menu resilience: an out-of-range choice must be handled gracefully (the
// menu comes back, the guide does not crash) -- exercises the same
// try/catch-per-action discipline guide.mjs's own reportError() proves.
const BAD_CHOICE_STDIN = guideStdin(['99', '5'])
const badChoiceSuccess = (stdout) => stdout.includes('Please type a number from 1 to 5.') && stdout.includes('Goodbye')
async function runBadChoice(label, exe) {
  const cwd = mkdtempSync(join(tmpdir(), 'bare-guide-spike-badchoice-'))
  try {
    const result = await runSpawnPipe({ exe, scriptPath: GUIDE_SCRIPT, cwd, runs: 2, timeoutMs: 15000, stdin: BAD_CHOICE_STDIN, isSuccess: badChoiceSuccess })
    console.log(`  ${formatResult(label, result)}`)
    check(`${label}: an out-of-range menu choice is handled gracefully, guide does not crash`, result.ok === result.runs)
  } finally {
    try { rmSync(cwd, { recursive: true, force: true }) } catch { /* best-effort */ }
  }
}
await runBadChoice('spawn(node) bad-choice', process.execPath)
if (existsSync(BARE_EXE)) await runBadChoice('spawn(bare) bad-choice', BARE_EXE)

// ── Layer 3: negative controls -- proves this harness can go RED ──────────
console.log('\n-- layer 3: negative controls --')

const selfTestResult = await selfTest()
for (const line of selfTestResult.detail) console.log(`  ${line}`)
check('spawn-pipe-harness.mjs selfTest(): the shipped harness correctly distinguishes OK/HANG/TOTAL_LOSS/PARTIAL', selfTestResult.pass)

// Guide-specific negative control: a fixture copy of bare-guide.mjs with
// its closing "Goodbye" line deleted -- simulates a run that never reaches
// clean completion (a stand-in for a hang or a crash mid-shutdown). If
// this spike's OWN success predicate (which requires "Goodbye" in stdout)
// cannot tell this apart from a healthy run, the spike's earlier GREEN
// results are not trustworthy -- same standard Phase 2's bridge spike met
// by driving the campaign's own known-broken stdio-check.mjs.
{
  const brokenDir = mkdtempSync(join(tmpdir(), 'bare-guide-spike-broken-'))
  try {
    const guideSrc = readFileSync(GUIDE_SCRIPT, 'utf8')
    const brokenSrc = guideSrc.replace(
      "write('\\nGoodbye -- this window is safe to close.\\n')",
      "/* negative-control fixture: goodbye line deliberately removed */",
    )
    if (brokenSrc === guideSrc) {
      check('negative control setup: the Goodbye line pattern was found and removable', false, 'replace() found no match -- fixture would not actually be broken; check the literal string against bare-guide.mjs')
    } else {
      const brokenScript = join(brokenDir, 'bare-guide-broken.mjs')
      writeFileSync(brokenScript, brokenSrc)
      const cwd = mkdtempSync(join(tmpdir(), 'bare-guide-spike-broken-run-'))
      try {
        const result = await runSpawnPipe({
          exe: process.execPath, scriptPath: brokenScript, cwd, runs: 3, timeoutMs: 15000,
          stdin: guideStdin(['5']), isSuccess: fullFlowSuccess,
        })
        console.log(`  ${formatResult('negative control (broken fixture)', result)}`)
        check('negative control: this spike correctly flags the broken fixture as NOT ok (missing "Goodbye")', result.ok === 0 && (result.partial + result.totalLoss) === result.runs)
      } finally {
        try { rmSync(cwd, { recursive: true, force: true }) } catch { /* best-effort */ }
      }
    }
  } finally {
    try { rmSync(brokenDir, { recursive: true, force: true }) } catch { /* best-effort */ }
  }
}

// ── Layer 4: the SEALED, bare-pack'd kit, from a from-scratch directory ───
// A green run of bare-guide.mjs as a raw script (layers 1-3) proves
// NOTHING about the bundled artifact a real client actually receives --
// the exact lesson RULE 4/the flush-race investigation already taught this
// campaign once. This layer builds the REAL sealed kit
// (kit/bare-guide-entry.mjs, per that file's own header -- the thin,
// unconditional entry, NOT bare-guide.mjs's own isMain-guarded form, which
// is structurally unreachable once bundled), copies it whole to a
// from-scratch temp directory, and drives `bare.exe app.bundle` through
// the SAME real-spawned-pipe method as every other layer here. Sequence
// answers the firewall offer with `skip` -- omitting it consumes the next
// line as the firewall answer instead and silently shifts everything
// after it (lost 20 minutes to exactly this before landing on the right
// sequence).
console.log('\n-- layer 4: the sealed, bare-pack\'d kit, from a from-scratch directory --')
{
  let bundleDir = null
  let hostileDir = null
  try {
    console.log('  building the sealed kit (kit/bare-guide-entry.mjs)...')
    execFileSync(process.execPath, [join(meshRoot, 'kit', 'build-bare-kit.mjs'), '--entry=kit/bare-guide-entry.mjs'], { cwd: meshRoot, stdio: 'pipe' })
    bundleDir = join(meshRoot, 'kit', 'dist-bare')
    const requiredFiles = ['app.bundle', 'bare.exe']
    const built = requiredFiles.every((f) => existsSync(join(bundleDir, f)))
    check('layer 4: build-bare-kit.mjs --entry=kit/bare-guide-entry.mjs produced app.bundle + bare.exe', built)
    // GATED (promoted from "reported" once kit/bare-guide-entry.mjs's
    // setWasmSource(import.meta.asset(...)) injection landed): the manifest
    // must contain the offloaded reducer.wasm, or posting is guaranteed to
    // fail downstream with no chance of passing the content assertion below.
    check('layer 4: dist/reducer.wasm is present in the sealed kit manifest (the asset-offload fix)', existsSync(join(bundleDir, 'dist', 'reducer.wasm')))
    if (built) {
      hostileDir = mkdtempSync(join(tmpdir(), 'bare-guide-sealed-'))
      cpSync(bundleDir, hostileDir, { recursive: true })

      const CEREMONY_STDIN = guideStdin(['2', 'skip', 'a real sealed-kit message', '/rooms', '/exit', '5'])
      const result = await runSpawnPipe({
        exe: join(hostileDir, 'bare.exe'), scriptPath: join(hostileDir, 'app.bundle'), cwd: hostileDir,
        runs: 2, timeoutMs: 20000, stdin: CEREMONY_STDIN,
        // GATED, per the lead's explicit ask: the full ceremony, including
        // an ACTUAL post reaching the reducer ("posted, seq" marker) --
        // not just the menu/close shell. Both defects this phase closed
        // (the isMain guard AND the wasm asset injection) are covered by
        // one predicate now that both are fixed.
        isSuccess: (stdout) =>
          stdout.includes('Welcome.') &&
          stdout.includes('ASYMMFLOW MESH -- GUIDE (Bare)') &&
          stdout.includes('kitchen table') &&
          /posted, seq \d+/.test(stdout) &&
          stdout.includes('a real sealed-kit message') &&
          stdout.includes('Goodbye -- this window is safe to close.'),
      })
      console.log(`  ${formatResult('sealed kit ceremony', result)}`)
      check('layer 4: the SEALED kit renders the menu, posts a REAL message through the reducer, and closes cleanly', result.ok === result.runs,
        result.results.find((r) => r.outcome !== 'OK') ? JSON.stringify(result.results.find((r) => r.outcome !== 'OK')).slice(0, 400) : '')
    }
  } catch (err) {
    check('layer 4: sealed kit build/run did not throw', false, err?.message ?? String(err))
  } finally {
    if (hostileDir) { try { rmSync(hostileDir, { recursive: true, force: true }) } catch { /* best-effort */ } }
  }
}

console.log(`\n${checks} check(s), ${failures} failure(s).`)
console.log(failures === 0 ? '\nBARE GUIDE SPIKE GREEN' : `\nBARE GUIDE SPIKE RED (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
