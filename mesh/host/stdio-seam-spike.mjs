// stdio-seam-spike.mjs — the permanent regression test for the DP4 stdio
// seam, built on spawn-pipe-harness.mjs. This is the test
// `mesh/host/bare-spike/stdio-check.mjs` should have been from day one: it
// gates on the REAL production topology (a Node parent, a `child_process`
// pipe, `stdout.on('data', …)`), asserts on OUTPUT CONTENT never exit code,
// and — this is the part that makes it a regression test rather than a
// one-off spike — it PROVES both binding-rule violations actually trip it,
// rather than asserting in prose that they would.
//
// Context: PHASE0_NOTES_D2_FLUSH_RACE.md found that a Bare sidecar script
// following two rules is safe over a real spawn pipe, and unsafe without
// them:
//   RULE (sync WASM)  — `new WebAssembly.Module(bytes)`, never
//                        `await WebAssembly.compile(bytes)`.
//   RULE (console writes) — write frames via Bare's native `console.log`,
//                        never `bare-process`'s `process.stdout.write()`.
// (A third campaign rule — keep the explicit `process.exit(0)` on stdin
// `'end'` — is also exercised by every fixture below; the GOOD fixture
// keeps it, and D2 already showed removing it makes things strictly worse,
// not better, so it is not re-litigated here as a fourth fixture.)
//
// Three fixtures, one real reducer.wasm compile each, run through the SAME
// harness, SAME topology, SAME real 3.96MB `mesh/dist/reducer.wasm`:
//   GOOD                — follows both rules.        Gate requires 0 loss.
//   BAD (async compile) — violates the sync-WASM rule. Gate requires this
//                          to actually fail some runs, not just exist.
//   BAD (bare-process)  — violates the console-writes rule. Gate requires
//                          this to actually fail (hang) some runs.
// If either BAD fixture stopped failing (e.g. a future Bare release fixes
// the underlying bug), this file's own assertions about them would need
// updating — that is by design: a regression test whose "bad" fixtures
// silently start passing is telling you the underlying defect it exists to
// catch may no longer be present, which is itself worth surfacing loudly
// rather than quietly going stale.
//
// NODE-ONLY (drives the child; imports spawn-pipe-harness.mjs, itself
// Node-only per its own header). Never part of the sealed artifact.
//
// Usage: npm run stdioseam   (builds reducer.wasm first, same convention as
// bareparity/kitspike/etc.), or directly: node host/stdio-seam-spike.mjs

import { mkdtempSync, rmSync, writeFileSync, existsSync, mkdirSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { runSpawnPipe, formatResult, selfTest } from './spawn-pipe-harness.mjs'

const __dirname = dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = join(__dirname, '..', '..') // module root (has go.mod), mirrors build-reducer.mjs
const WASM_PATH = join(REPO_ROOT, 'mesh', 'dist', 'reducer.wasm')
const BARE_EXE = join(__dirname, '..', 'node_modules', 'bare-runtime-win32-x64', 'bin', 'bare.exe')

const RUNS = 30
const STDIN_FRAME = '{"id":1,"method":"hello","params":{}}\n'

function jsonPath(p) {
  // embeds an absolute Windows path as a safe JS string literal inside a
  // generated fixture file (backslashes are the hazard; JSON.stringify
  // handles this correctly for any path, no manual escaping needed)
  return JSON.stringify(p)
}

const GOOD_FIXTURE = `
// GOOD: follows both binding rules from PHASE0_NOTES_D2_FLUSH_RACE.md.
import fs from 'bare-fs'
import process from 'bare-process'

const bytes = fs.readFileSync(${jsonPath(WASM_PATH)})
const mod = new WebAssembly.Module(bytes) // RULE: sync, never WebAssembly.compile()
console.log(JSON.stringify({ event: 'ready', importCount: WebAssembly.Module.imports(mod).length }))

let buf = ''
process.stdin.on('data', (chunk) => {
  buf += chunk.toString('utf8')
  let idx
  while ((idx = buf.indexOf('\\n')) !== -1) {
    const line = buf.slice(0, idx)
    buf = buf.slice(idx + 1)
    if (!line.trim()) continue
    console.log(JSON.stringify({ echoed: line })) // RULE: console.log, never process.stdout.write()
  }
})
process.stdin.on('end', () => process.exit(0)) // campaign rule: explicit exit, do not "clean this up"
`

const BAD_ASYNC_COMPILE_FIXTURE = `
// BAD: violates the sync-WASM rule (Bug A, PHASE0_NOTES_D2_FLUSH_RACE.md §3).
import fs from 'bare-fs'
import process from 'bare-process'

const bytes = fs.readFileSync(${jsonPath(WASM_PATH)})
const mod = await WebAssembly.compile(bytes) // VIOLATION: async compile
console.log(JSON.stringify({ event: 'ready', importCount: WebAssembly.Module.imports(mod).length }))

let buf = ''
process.stdin.on('data', (chunk) => {
  buf += chunk.toString('utf8')
  let idx
  while ((idx = buf.indexOf('\\n')) !== -1) {
    const line = buf.slice(0, idx)
    buf = buf.slice(idx + 1)
    if (!line.trim()) continue
    console.log(JSON.stringify({ echoed: line }))
  }
})
process.stdin.on('end', () => process.exit(0))
`

const BAD_BAREPROCESS_WRITE_FIXTURE = `
// BAD: violates the console-writes rule (Bug B, PHASE0_NOTES_D2_FLUSH_RACE.md §3).
import fs from 'bare-fs'
import process from 'bare-process'

const bytes = fs.readFileSync(${jsonPath(WASM_PATH)})
const mod = new WebAssembly.Module(bytes) // sync compile kept correct here —
                                           // isolates the OTHER rule's violation
process.stdout.write(JSON.stringify({ event: 'ready', importCount: WebAssembly.Module.imports(mod).length }) + '\\n') // VIOLATION

let buf = ''
process.stdin.on('data', (chunk) => {
  buf += chunk.toString('utf8')
  let idx
  while ((idx = buf.indexOf('\\n')) !== -1) {
    const line = buf.slice(0, idx)
    buf = buf.slice(idx + 1)
    if (!line.trim()) continue
    process.stdout.write(JSON.stringify({ echoed: line }) + '\\n') // VIOLATION
  }
})
process.stdin.on('end', () => process.exit(0))
`

function isSuccessEcho(stdout) {
  return stdout.includes('"event":"ready"') && stdout.includes('"echoed"')
}

async function runFixture(dir, name, source) {
  const path = join(dir, name)
  writeFileSync(path, source)
  return runSpawnPipe({
    exe: BARE_EXE,
    scriptPath: path,
    runs: RUNS,
    timeoutMs: 12000,
    stdin: STDIN_FRAME,
    isSuccess: isSuccessEcho,
  })
}

async function main() {
  console.log('=== stdio-seam-spike: harness self-test first (design law: prove RED before trusting GREEN) ===')
  const self = await selfTest()
  for (const line of self.detail) console.log('  ' + line)
  if (!self.pass) {
    console.error('\nFATAL: spawn-pipe-harness selfTest failed — refusing to trust any result below.')
    process.exit(1)
  }
  console.log('harness selfTest: PASS\n')

  if (!existsSync(BARE_EXE)) {
    console.error(`FATAL: bare.exe not found at ${BARE_EXE} — run "npm install" in mesh/ first.`)
    process.exit(1)
  }
  if (!existsSync(WASM_PATH)) {
    console.error(`FATAL: ${WASM_PATH} not found — run "npm run build" first (this file's own npm script does this for you: "npm run stdioseam").`)
    process.exit(1)
  }

  // Fixtures MUST live under mesh/host/, not a real OS tmpdir. This is a
  // real, load-bearing finding from this campaign's own Phase 0
  // hostile-geography testing: Bare's module resolution walks UPWARD from
  // the script's own directory looking for an ancestor node_modules (it
  // does not consult the process's cwd, and there is no such thing as a
  // "global" bare-fs/bare-process to fall back to) — a fixture written to a
  // real OS tmpdir has no ancestor node_modules at all, so `import fs from
  // 'bare-fs'` crashes immediately with MODULE_NOT_FOUND before the fixture
  // does anything. The first version of this file used `os.tmpdir()` and
  // every fixture (GOOD included) came back TOTAL_LOSS for that reason —
  // caught by the GOOD fixture's own gate, which is exactly what design law
  // #1 (assert on content, never exit code) is for. A directory under
  // mesh/host/ resolves mesh/node_modules by walking up two levels, exactly
  // like the "complete kit" case in PHASE0_NOTES_D_REVERIFY.md §4 pass (b).
  const dir = join(__dirname, '.stdio-seam-tmp')
  rmSync(dir, { recursive: true, force: true }) // clean any stray leftover from a prior crashed run
  mkdirSync(dir, { recursive: true })
  let allPass = true

  try {
    console.log('=== GOOD fixture (both binding rules followed) — gate requires 0 loss over 30 runs ===')
    const good = await runFixture(dir, 'good.mjs', GOOD_FIXTURE)
    console.log(formatResult('GOOD', good))
    const goodClean = good.ok === RUNS
    if (!goodClean) {
      allPass = false
      console.error(`FAIL: GOOD fixture must be OK=${RUNS}/${RUNS} (0 loss) — this is the regression this file exists to catch. Got OK=${good.ok}/${RUNS}.`)
      for (const r of good.results) if (r.outcome !== 'OK') console.error('  anomaly:', JSON.stringify({ outcome: r.outcome, ms: r.ms, stdout: r.stdout.slice(0, 200) }))
    } else {
      console.log('PASS: GOOD fixture round-tripped ndjson frames through a real spawn pipe with ZERO loss over 30 runs.')
    }

    console.log('\n=== BAD fixture: async WebAssembly.compile() (Bug A reintroduced) — gate requires this to actually fail some runs ===')
    const badAsync = await runFixture(dir, 'bad-async-compile.mjs', BAD_ASYNC_COMPILE_FIXTURE)
    console.log(formatResult('BAD (async compile)', badAsync))
    const badAsyncCaughtDefect = (badAsync.partial + badAsync.totalLoss + badAsync.hang) > 0
    if (!badAsyncCaughtDefect) {
      allPass = false
      console.error(`FAIL: the async-compile violation fixture produced 0 non-OK runs (OK=${badAsync.ok}/${RUNS}) — either the underlying Bare defect no longer reproduces (update this file's assumptions and PHASE0_NOTES_D2_FLUSH_RACE.md) or this fixture stopped exercising it. Either way this needs a human, not a silent pass.`)
    } else {
      console.log(`PASS (as a negative-control demonstration): the async-compile violation DID trip real failures (${RUNS - badAsync.ok}/${RUNS} non-OK) — the rule is proven load-bearing, not asserted in prose.`)
    }

    console.log('\n=== BAD fixture: bare-process process.stdout.write() (Bug B reintroduced) — gate requires this to actually fail some runs ===')
    const badWrite = await runFixture(dir, 'bad-bareprocess-write.mjs', BAD_BAREPROCESS_WRITE_FIXTURE)
    console.log(formatResult('BAD (bare-process write)', badWrite))
    const badWriteCaughtDefect = (badWrite.partial + badWrite.totalLoss + badWrite.hang) > 0
    if (!badWriteCaughtDefect) {
      allPass = false
      console.error(`FAIL: the bare-process-write violation fixture produced 0 non-OK runs (OK=${badWrite.ok}/${RUNS}) — either the underlying Bare defect no longer reproduces (update this file's assumptions and PHASE0_NOTES_D2_FLUSH_RACE.md) or this fixture stopped exercising it. Either way this needs a human, not a silent pass.`)
    } else {
      console.log(`PASS (as a negative-control demonstration): the bare-process-write violation DID trip real failures (${RUNS - badWrite.ok}/${RUNS} non-OK, HANG=${badWrite.hang}/${RUNS}) — the rule is proven load-bearing, not asserted in prose.`)
    }
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }

  console.log('')
  if (allPass) {
    console.log('stdio-seam-spike: ALL GATES PASS — the binding rules hold under the real production pipe topology, and both violations are proven (not assumed) to break the seam without them.')
    process.exit(0)
  } else {
    console.error('stdio-seam-spike: GATE FAILURE — see FAIL lines above.')
    process.exit(1)
  }
}

await main()
