// bare-net-spike.mjs — SC-2 gate: proves bare-net.mjs's replication wire
// works end-to-end between TWO real, SEALED, bare-pack'd kit processes,
// each in its own from-scratch hostile directory, over a real spawned pipe
// (never a shell pipe — RULE 4, mesh/docs/bare-campaign's own lesson).
//
// WHY THIS FILE DOES NOT USE spawn-pipe-harness.mjs's runSpawnPipe() FOR THE
// CEREMONY ITSELF (declared, not silently deviated): runSpawnPipe's shape is
// ONE fixed stdin string fed to ONE child, then the whole run is judged on
// the FINAL stdout. The corridor ceremony needs the OPPOSITE of that: kit B
// mints a random Ed25519 writer key at JOIN time that kit A must read off
// B's live stdout and feed back into A's own stdin (`ADDWRITER <key>`)
// before either side can proceed — a real bidirectional relay between TWO
// concurrently running children, not a batch. There is no way to
// precompute that stdin script; the harness IS the human who reads the
// pairing code aloud in a real corridor ceremony (kit-spike.mjs's own
// in-process equivalent — cmdsA.addWriter(pairingCode) — read that file
// before touching this one). What IS reused: `selfTest()` from
// spawn-pipe-harness.mjs (required below, verbatim), its design laws
// (assert on CONTENT never exit code, categorize outcomes, measure
// fractions honestly), and `child_process.spawn` itself — the same
// primitive runSpawnPipe uses internally, not a fresh spawn technique.
//
// STRUCTURE:
//   0. build the sealed kit from kit/bare-corridor-entry.mjs
//   1. NEGATIVE CONTROLS FIRST (binding order, campaign law): wrong-key pair
//      and no-path pair must both fail to replicate before any green result
//      below counts for anything.
//   2. spawn-pipe-harness.mjs's own selfTest() (required)
//   3. positive (a): TCP fallback, hyperswarm disabled, N>=16
//   4. positive (b): hyperswarm path (live DHT), N>=16, reported honestly
//
// Run: npm run sc2spike   (equivalently: node kit/bare-net-spike.mjs)

import { spawn } from 'node:child_process'
import { mkdtempSync, rmSync, existsSync, cpSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'
import { randomBytes } from 'node:crypto'
import { selfTest } from '../host/spawn-pipe-harness.mjs'

const __dirname = dirname(fileURLToPath(import.meta.url))
const meshRoot = join(__dirname, '..')
// PRIVATE output directory (--out=kit/.sc2-dist), not the shared
// kit/dist-bare default — build-bare-kit.mjs unconditionally wipes its
// target (its own §1 comment), and this campaign runs several concurrent
// gates that each build a DIFFERENT entry. A shared target means one
// gate's rmSync can delete another gate's kit mid-copy — a failure that
// looks like a defect in the kit and is actually a defect in the harness
// (observed directly in this mission's own first run: a negative-control
// round failed at CREATE with a bare MODULE-not-quite-right timeout while
// an unrelated build was running concurrently on this machine). Building
// into a directory only this spike ever writes to removes the race at the
// source, structurally, rather than papering over it with a private copy
// taken after the fact.
const bundleDir = join(__dirname, '.sc2-dist')
const BARE_EXE_NAME = 'bare.exe'
const APP_BUNDLE_NAME = 'app.bundle'
// This entry's import graph (mesh-node.mjs + reducer/WASI shim + bare-net.
// mjs) — declared so build-bare-kit.mjs's §2c hard gate refuses a kit that
// silently cannot reach the network, at BUILD time rather than ceremony
// time. Measured via a real build (§8, SC2_REPORT.md) before being relied
// on here — not guessed.
const REQUIRED_ADDONS = 'bare-tcp,udx-native,sodium-native,bare-dns'

let checks = 0
let failures = 0
function check(name, cond, detail = '') {
  checks++
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' -- ' + detail : ''}`) }
}

function assertNoHash(p, label) {
  check(`path hygiene: ${label} contains no '#' (Bare addon resolution breaks otherwise)`, !p.includes('#'), p)
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

// ── the interactive line-reader over a real child's real stdout pipe ──────
function makeLineReader(child) {
  let buf = ''
  const queue = []
  const waiters = []
  let ended = false
  let stderrBuf = ''
  child.stdout.on('data', (d) => {
    buf += d.toString('utf8')
    let idx
    while ((idx = buf.indexOf('\n')) !== -1) {
      const line = buf.slice(0, idx).replace(/\r$/, '')
      buf = buf.slice(idx + 1)
      if (waiters.length) waiters.shift()(line)
      else queue.push(line)
    }
  })
  child.stderr.on('data', (d) => { stderrBuf += d.toString('utf8') })
  child.on('close', () => { ended = true; while (waiters.length) waiters.shift()(null) })
  return {
    // Returns a line, null (process ended, no more lines ever), or
    // undefined (timed out — distinct from EOF, per design law #3:
    // HANG/TOTAL_LOSS/PARTIAL/OK are four different things, never collapsed).
    nextLine(timeoutMs) {
      if (queue.length) return Promise.resolve(queue.shift())
      if (ended) return Promise.resolve(null)
      return new Promise((resolve) => {
        const timer = setTimeout(() => {
          const i = waiters.indexOf(onLine)
          if (i !== -1) waiters.splice(i, 1)
          resolve(undefined)
        }, timeoutMs)
        function onLine(line) { clearTimeout(timer); resolve(line) }
        waiters.push(onLine)
      })
    },
    get stderr() { return stderrBuf },
  }
}

/** expectLine(reader, predicate, timeoutMs) -> { ok, line, reason, seen } */
async function expectLine(reader, predicate, timeoutMs, label = '') {
  const deadline = Date.now() + timeoutMs
  const seen = []
  for (;;) {
    const remaining = deadline - Date.now()
    if (remaining <= 0) return { ok: false, reason: `timeout waiting for ${label}`, seen }
    const line = await reader.nextLine(remaining)
    if (line === null) return { ok: false, reason: `process closed while waiting for ${label} (stderr: ${reader.stderr.slice(0, 300)})`, seen }
    if (line === undefined) return { ok: false, reason: `timeout waiting for ${label}`, seen }
    seen.push(line)
    if (predicate(line)) return { ok: true, line, seen }
  }
}

/** collectUntil(reader, endMarker, timeoutMs) -> lines (best-effort partial on timeout) */
async function collectUntil(reader, endMarker, timeoutMs) {
  const deadline = Date.now() + timeoutMs
  const out = []
  for (;;) {
    const remaining = deadline - Date.now()
    if (remaining <= 0) return out
    const line = await reader.nextLine(remaining)
    if (line == null) return out
    out.push(line)
    if (line === endMarker) return out
  }
}

function send(child, line) { child.stdin.write(line + '\n') }

// The exe AND the bundle must both be resolved from the SAME hostile `cwd`
// (each mkdtempSync copy carries its own bare.exe, per cpSync(bundleDir,
// dir) above) — never the original kit/dist-bare location. An earlier draft
// spawned the ORIGINAL kit/dist-bare/bare.exe with only `cwd` overridden to
// the hostile dir and a RELATIVE 'app.bundle' arg: that is not actually a
// sealed run (the exe never left the repo) and, empirically, Bare's module
// resolution for a relative bundle arg does not follow spawn's `cwd` option
// the way a shell would — `MODULE_NOT_FOUND` on every round. Both the exe
// path and the bundle arg are ABSOLUTE, matching bare-guide-spike.mjs's own
// proven layer 4 pattern (`exe: join(hostileDir,'bare.exe'), scriptPath:
// join(hostileDir,'app.bundle')`) exactly.
function spawnKit(cwd) {
  const exe = join(cwd, BARE_EXE_NAME)
  const bundle = join(cwd, APP_BUNDLE_NAME)
  const child = spawn(exe, [bundle], { cwd, stdio: ['pipe', 'pipe', 'pipe'] })
  return { child, reader: makeLineReader(child) }
}

async function quitAndClose(procs) {
  for (const p of procs) { try { send(p.child, 'QUIT') } catch { /* best-effort */ } }
  await Promise.race([
    Promise.all(procs.map((p) => new Promise((r) => p.child.once('close', r)))),
    sleep(3000),
  ])
  for (const p of procs) { try { p.child.kill('SIGKILL') } catch { /* already dead */ } }
}

// `bundleDir` (kit/.sc2-dist) is now private to this spike — see its own
// declaration comment above — so every hostile-dir copy can read directly
// from it with no risk of a concurrent rebuild by another gate landing
// mid-`cpSync`. Build ONCE (step 0), copy from it N times; never rebuild
// per cycle.
function makeHostileDirs(prefix) {
  const dirA = mkdtempSync(join(tmpdir(), `${prefix}-a-`))
  const dirB = mkdtempSync(join(tmpdir(), `${prefix}-b-`))
  assertNoHash(dirA, `${prefix} dirA`)
  assertNoHash(dirB, `${prefix} dirB`)
  cpSync(bundleDir, dirA, { recursive: true })
  cpSync(bundleDir, dirB, { recursive: true })
  return { dirA, dirB }
}

function cleanupDirs(...dirs) {
  for (const d of dirs) { try { rmSync(d, { recursive: true, force: true }) } catch { /* best-effort */ } }
}

// ── ONE round of the real ceremony: A creates + posts, B joins + reads,
// B posts, A reads. Content-asserted both directions. `mode`: 'tcp' or
// 'swarm'. Returns { ok, reason }. ─────────────────────────────────────
async function runPositiveRound(dirA, dirB, mode, tag) {
  const A = spawnKit(dirA)
  const B = spawnKit(dirB)
  try {
    const readyA = await expectLine(A.reader, (l) => l === 'CORRIDOR READY', 20000, 'A ready')
    const readyB = await expectLine(B.reader, (l) => l === 'CORRIDOR READY', 20000, 'B ready')
    if (!readyA.ok) return { ok: false, reason: `A never became ready: ${readyA.reason}` }
    if (!readyB.ok) return { ok: false, reason: `B never became ready: ${readyB.reason}` }

    const useSwarm = mode === 'swarm' ? '1' : '0'
    send(A.child, 'ACTOR ana')
    await expectLine(A.reader, (l) => l === 'ACTOR ana', 12000, 'A actor ack')
    send(A.child, `NET ${useSwarm}`)
    await expectLine(A.reader, (l) => l.startsWith('NET '), 12000, 'A net ack')
    send(B.child, 'ACTOR sam')
    await expectLine(B.reader, (l) => l === 'ACTOR sam', 12000, 'B actor ack')
    send(B.child, `NET ${useSwarm}`)
    await expectLine(B.reader, (l) => l.startsWith('NET '), 12000, 'B net ack')

    send(A.child, 'CREATE')
    const rk = await expectLine(A.reader, (l) => l.startsWith('ROOMKEY '), 20000, 'A roomkey')
    if (!rk.ok) return { ok: false, reason: `A never printed ROOMKEY: ${rk.reason}` }
    const roomKey = rk.line.slice('ROOMKEY '.length)

    let port
    if (mode === 'tcp') {
      send(A.child, 'LISTEN 0')
      const listening = await expectLine(A.reader, (l) => l.startsWith('LISTENING '), 20000, 'A listening')
      if (!listening.ok) return { ok: false, reason: `A never listened: ${listening.reason}` }
      port = listening.line.slice('LISTENING '.length)
    } else {
      send(A.child, 'JOINSWARM')
      const joined = await expectLine(A.reader, (l) => l === 'JOINEDSWARM' || l === 'JOINSWARMFAILED', 20000, 'A joinswarm')
      if (joined.line !== 'JOINEDSWARM') return { ok: false, reason: `A could not join hyperswarm: ${joined.line}` }
    }

    send(B.child, `JOIN ${roomKey}`)
    const wk = await expectLine(B.reader, (l) => l.startsWith('WRITERKEY '), 20000, 'B writerkey')
    if (!wk.ok) return { ok: false, reason: `B never printed WRITERKEY: ${wk.reason}` }
    const writerKey = wk.line.slice('WRITERKEY '.length)

    if (mode === 'tcp') {
      send(B.child, `CONNECT 127.0.0.1 ${port}`)
      const connected = await expectLine(B.reader, (l) => l === 'CONNECTED' || l.startsWith('CONNECTFAILED') || l.startsWith('ERROR'), 20000, 'B connect')
      if (connected.line !== 'CONNECTED') return { ok: false, reason: `B could not connect: ${connected.line}` }
    } else {
      send(B.child, 'JOINSWARM')
      const joined = await expectLine(B.reader, (l) => l === 'JOINEDSWARM' || l === 'JOINSWARMFAILED', 20000, 'B joinswarm')
      if (joined.line !== 'JOINEDSWARM') return { ok: false, reason: `B could not join hyperswarm: ${joined.line}` }
    }

    send(A.child, `ADDWRITER ${writerKey}`)
    const added = await expectLine(A.reader, (l) => l === 'ADDEDWRITER' || l.startsWith('ADDWRITERFAILED') || l.startsWith('ERROR'), 20000, 'A addwriter')
    if (added.line !== 'ADDEDWRITER') return { ok: false, reason: `A could not add writer: ${added.line}` }

    const writableTimeout = mode === 'swarm' ? 40000 : 15000
    send(B.child, `WAITWRITABLE ${writableTimeout}`)
    const writable = await expectLine(B.reader, (l) => l === 'WRITABLE' || l.startsWith('NOTWRITABLE'), writableTimeout + 5000, 'B writable')
    if (writable.line !== 'WRITABLE') return { ok: false, reason: `B never became writable: ${writable.line}` }

    const msgA = `hello-from-ana-${tag}-${Date.now()}`
    send(A.child, `POST ${msgA}`)
    const postedA = await expectLine(A.reader, (l) => l.startsWith('POSTED '), 20000, 'A posted')
    if (!postedA.ok) return { ok: false, reason: `A could not post: ${postedA.reason}` }

    const seeTimeout = mode === 'swarm' ? 40000 : 15000
    const deadline1 = Date.now() + seeTimeout
    let sawA = false
    while (Date.now() < deadline1 && !sawA) {
      send(B.child, 'LIST')
      const lines = await collectUntil(B.reader, 'MSGEND', 5000)
      sawA = lines.some((l) => l === `MSG|1|ana|${msgA}` || l.endsWith(`|${msgA}`))
      if (!sawA) await sleep(400)
    }
    if (!sawA) return { ok: false, reason: 'B never saw A\'s exact message text' }

    const msgB = `hi-ana-from-sam-${tag}-${Date.now()}`
    send(B.child, `POST ${msgB}`)
    const postedB = await expectLine(B.reader, (l) => l.startsWith('POSTED '), 20000, 'B posted')
    if (!postedB.ok) return { ok: false, reason: `B could not post: ${postedB.reason}` }

    const deadline2 = Date.now() + seeTimeout
    let sawB = false
    while (Date.now() < deadline2 && !sawB) {
      send(A.child, 'LIST')
      const lines = await collectUntil(A.reader, 'MSGEND', 5000)
      sawB = lines.some((l) => l.endsWith(`|${msgB}`))
      if (!sawB) await sleep(400)
    }
    if (!sawB) return { ok: false, reason: 'A never saw B\'s exact message text' }

    return { ok: true, reason: '' }
  } finally {
    await quitAndClose([A, B])
  }
}

// ── negative control (i): wrong key — B joins a room key nobody founded ───
async function runWrongKeyRound(dirA, dirB) {
  const A = spawnKit(dirA)
  const B = spawnKit(dirB)
  try {
    await expectLine(A.reader, (l) => l === 'CORRIDOR READY', 20000, 'A ready')
    await expectLine(B.reader, (l) => l === 'CORRIDOR READY', 20000, 'B ready')
    send(A.child, 'ACTOR ana'); await expectLine(A.reader, (l) => l === 'ACTOR ana', 12000, '')
    send(A.child, 'NET 0'); await expectLine(A.reader, (l) => l.startsWith('NET '), 12000, '')
    send(B.child, 'ACTOR sam'); await expectLine(B.reader, (l) => l === 'ACTOR sam', 12000, '')
    send(B.child, 'NET 0'); await expectLine(B.reader, (l) => l.startsWith('NET '), 12000, '')

    send(A.child, 'CREATE')
    const rk = await expectLine(A.reader, (l) => l.startsWith('ROOMKEY '), 20000, 'A roomkey')
    if (!rk.ok) return { ok: false, replicated: null, reason: `setup failed: ${rk.reason}` }

    send(A.child, 'LISTEN 0')
    const listening = await expectLine(A.reader, (l) => l.startsWith('LISTENING '), 20000, 'A listening')
    if (!listening.ok) return { ok: false, replicated: null, reason: `setup failed: ${listening.reason}` }
    const port = listening.line.slice('LISTENING '.length)

    const wrongKey = randomBytes(32).toString('hex')
    send(B.child, `JOIN ${wrongKey}`)
    await expectLine(B.reader, (l) => l.startsWith('WRITERKEY ') || l.startsWith('ERROR'), 20000, 'B join wrong key')

    send(B.child, `CONNECT 127.0.0.1 ${port}`)
    await expectLine(B.reader, (l) => l === 'CONNECTED' || l.startsWith('CONNECTFAILED') || l.startsWith('ERROR'), 20000, 'B connect')

    const msgA = `wrongkey-control-${Date.now()}`
    send(A.child, `POST ${msgA}`)
    await expectLine(A.reader, (l) => l.startsWith('POSTED '), 20000, 'A posted')

    // give it a real window to (wrongly) replicate if the harness were broken
    await sleep(4000)
    send(B.child, 'LIST')
    const lines = await collectUntil(B.reader, 'MSGEND', 5000)
    const replicated = lines.some((l) => l.endsWith(`|${msgA}`))
    return { ok: !replicated, replicated, reason: replicated ? 'B saw A\'s message despite a mismatched room key' : '' }
  } finally {
    await quitAndClose([A, B])
  }
}

// ── negative control (ii): no path — A and B never connect at all ─────────
async function runNoPathRound(dirA, dirB) {
  const A = spawnKit(dirA)
  const B = spawnKit(dirB)
  try {
    await expectLine(A.reader, (l) => l === 'CORRIDOR READY', 20000, 'A ready')
    await expectLine(B.reader, (l) => l === 'CORRIDOR READY', 20000, 'B ready')
    send(A.child, 'ACTOR ana'); await expectLine(A.reader, (l) => l === 'ACTOR ana', 12000, '')
    send(A.child, 'NET 0'); await expectLine(A.reader, (l) => l.startsWith('NET '), 12000, '')
    send(B.child, 'ACTOR sam'); await expectLine(B.reader, (l) => l === 'ACTOR sam', 12000, '')
    send(B.child, 'NET 0'); await expectLine(B.reader, (l) => l.startsWith('NET '), 12000, '')

    send(A.child, 'CREATE')
    const rk = await expectLine(A.reader, (l) => l.startsWith('ROOMKEY '), 20000, 'A roomkey')
    if (!rk.ok) return { ok: false, replicated: null, reason: `setup failed: ${rk.reason}` }
    const roomKey = rk.line.slice('ROOMKEY '.length)

    // deliberately: NO LISTEN on A, NO CONNECT/JOINSWARM on B — no path at all.
    send(B.child, `JOIN ${roomKey}`)
    const wk = await expectLine(B.reader, (l) => l.startsWith('WRITERKEY '), 20000, 'B writerkey')
    if (!wk.ok) return { ok: false, replicated: null, reason: `setup failed: ${wk.reason}` }
    const writerKey = wk.line.slice('WRITERKEY '.length)

    send(A.child, `ADDWRITER ${writerKey}`)
    await expectLine(A.reader, (l) => l === 'ADDEDWRITER' || l.startsWith('ERROR'), 20000, 'A addwriter')

    const msgA = `nopath-control-${Date.now()}`
    send(A.child, `POST ${msgA}`)
    await expectLine(A.reader, (l) => l.startsWith('POSTED '), 20000, 'A posted')

    send(B.child, 'WAITWRITABLE 5000')
    const writable = await expectLine(B.reader, (l) => l === 'WRITABLE' || l.startsWith('NOTWRITABLE'), 20000, 'B writable')
    const becameWritable = writable.line === 'WRITABLE'

    send(B.child, 'LIST')
    const lines = await collectUntil(B.reader, 'MSGEND', 5000)
    const replicated = lines.some((l) => l.endsWith(`|${msgA}`))
    return {
      ok: !replicated && !becameWritable,
      replicated,
      reason: replicated ? 'B saw A\'s message with NO network path at all'
        : becameWritable ? 'B became writable with NO network path at all (the addWriter op should be unreachable)' : '',
    }
  } finally {
    await quitAndClose([A, B])
  }
}

// ═══════════════════════════════════════════════════════════════════════
console.log('bare-net-spike -- SC-2: the network leg under Bare, two real sealed processes\n')

// ── 0. build the sealed corridor kit ───────────────────────────────────
console.log('-- step 0: build the sealed kit (kit/bare-corridor-entry.mjs) --')
let built = false
try {
  execFileSync(process.execPath, [
    join(meshRoot, 'kit', 'build-bare-kit.mjs'),
    '--entry=kit/bare-corridor-entry.mjs',
    '--out=kit/.sc2-dist',
    `--require-addons=${REQUIRED_ADDONS}`,
  ], { cwd: meshRoot, stdio: 'pipe' })
  built = existsSync(join(bundleDir, APP_BUNDLE_NAME)) && existsSync(join(bundleDir, BARE_EXE_NAME))
} catch (err) {
  console.log(`  build FAILED: ${err.message}`)
}
check('step 0: build-bare-kit.mjs --entry=kit/bare-corridor-entry.mjs --out=kit/.sc2-dist produced app.bundle + bare.exe', built)
check('step 0: dist/reducer.wasm offloaded into the sealed kit', existsSync(join(bundleDir, 'dist', 'reducer.wasm')))
check(`step 0: required native addons present (${REQUIRED_ADDONS}) -- build-bare-kit.mjs's own hard gate would have thrown otherwise`, built)

if (!built) {
  console.log('\ncannot proceed without a built kit.')
  console.log(`\n${checks} check(s), ${failures} failure(s).`)
  console.log('\nSC2 NET SPIKE RED (build failed)')
  process.exit(1)
}

// ── 1. NEGATIVE CONTROLS FIRST — binding order ─────────────────────────
console.log('\n-- step 1: negative controls (run FIRST, per campaign law) --')

const NEGATIVE_RUNS = 3
let wrongKeyResults = []
for (let i = 1; i <= NEGATIVE_RUNS; i++) {
  const { dirA, dirB } = makeHostileDirs('sc2-wrongkey')
  try {
    const r = await runWrongKeyRound(dirA, dirB)
    wrongKeyResults.push(r)
    console.log(`  wrong-key run ${i}/${NEGATIVE_RUNS}: ${r.ok ? 'correctly did NOT replicate' : 'FAIL -- ' + r.reason}`)
  } finally {
    cleanupDirs(dirA, dirB)
  }
}
check(`negative control (i) wrong-key: B never replicates A's room across ${NEGATIVE_RUNS} run(s)`,
  wrongKeyResults.every((r) => r.ok), JSON.stringify(wrongKeyResults.filter((r) => !r.ok)))

let noPathResults = []
for (let i = 1; i <= NEGATIVE_RUNS; i++) {
  const { dirA, dirB } = makeHostileDirs('sc2-nopath')
  try {
    const r = await runNoPathRound(dirA, dirB)
    noPathResults.push(r)
    console.log(`  no-path run ${i}/${NEGATIVE_RUNS}: ${r.ok ? 'correctly did NOT replicate' : 'FAIL -- ' + r.reason}`)
  } finally {
    cleanupDirs(dirA, dirB)
  }
}
check(`negative control (ii) no-path: B never replicates A's room across ${NEGATIVE_RUNS} run(s) (no connect, no hyperswarm)`,
  noPathResults.every((r) => r.ok), JSON.stringify(noPathResults.filter((r) => !r.ok)))

const negativeControlsPassed = wrongKeyResults.every((r) => r.ok) && noPathResults.every((r) => r.ok)
if (!negativeControlsPassed) {
  console.log('\nNEGATIVE CONTROLS FAILED -- this harness cannot be trusted to detect non-replication.')
  console.log('Per campaign law, no positive result below counts. Stopping here.')
  console.log(`\n${checks} check(s), ${failures} failure(s).`)
  console.log(`\nSC2 NET SPIKE RED (${failures} failure(s), negative controls untrustworthy)`)
  process.exit(1)
}

// ── 2. spawn-pipe-harness.mjs's own selfTest() ─────────────────────────
console.log('\n-- step 2: spawn-pipe-harness.mjs selfTest() --')
const selfTestResult = await selfTest()
for (const line of selfTestResult.detail) console.log(`  ${line}`)
check('spawn-pipe-harness.mjs selfTest(): correctly distinguishes OK/HANG/TOTAL_LOSS/PARTIAL', selfTestResult.pass)

// ── 3. positive (a): TCP fallback, hyperswarm disabled, N>=16 ─────────
console.log('\n-- step 3: positive (a) -- TCP fallback, DHT disabled, N=16 --')
const TCP_N = 16
let tcpResults = []
for (let i = 1; i <= TCP_N; i++) {
  const { dirA, dirB } = makeHostileDirs('sc2-tcp')
  try {
    const r = await runPositiveRound(dirA, dirB, 'tcp', `tcp${i}`)
    tcpResults.push(r)
    console.log(`  TCP round ${i}/${TCP_N}: ${r.ok ? 'OK' : 'FAIL -- ' + r.reason}`)
  } finally {
    cleanupDirs(dirA, dirB)
  }
}
const tcpOk = tcpResults.filter((r) => r.ok).length
console.log(`  TCP fallback: OK=${tcpOk}/${TCP_N}`)
check(`positive (a): TCP fallback replicates both directions, content-asserted, ${TCP_N}/${TCP_N}`, tcpOk === TCP_N,
  JSON.stringify(tcpResults.filter((r) => !r.ok).map((r) => r.reason)))

// ── 4. positive (b): hyperswarm path, N>=16, reported honestly ────────
console.log('\n-- step 4: positive (b) -- hyperswarm (live DHT), N=16 --')
const SWARM_N = 16
let swarmResults = []
for (let i = 1; i <= SWARM_N; i++) {
  const { dirA, dirB } = makeHostileDirs('sc2-swarm')
  try {
    const r = await runPositiveRound(dirA, dirB, 'swarm', `swarm${i}`)
    swarmResults.push(r)
    console.log(`  hyperswarm round ${i}/${SWARM_N}: ${r.ok ? 'OK' : 'FAIL -- ' + r.reason}`)
  } finally {
    cleanupDirs(dirA, dirB)
  }
}
const swarmOk = swarmResults.filter((r) => r.ok).length
console.log(`  hyperswarm: MEASURED OK=${swarmOk}/${SWARM_N} (live DHT -- environment-dependent, reported honestly, not retried)`)
// This path is explicitly allowed to be flaky (campaign brief); we still
// record a check so the report's own pass/fail tally is honest, but the
// REPORT's prose is what carries the measured fraction, not just this line.
check(`positive (b): hyperswarm path measured fraction recorded (OK=${swarmOk}/${SWARM_N}, not gated pass/fail -- see SC2_REPORT.md)`, true)

console.log(`\n${checks} check(s), ${failures} failure(s).`)
console.log(`TCP: OK=${tcpOk}/${TCP_N}   hyperswarm: OK=${swarmOk}/${SWARM_N}`)
console.log(failures === 0 ? '\nSC2 NET SPIKE GREEN' : `\nSC2 NET SPIKE RED (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
