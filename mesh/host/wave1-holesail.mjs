// wave1-holesail.mjs — Mission A, replication half, stage 2 (real transport).
//
// Two SEPARATE OS processes, each with its own on-disk Corestore, replicating
// through a REAL Holesail tunnel over the Hyperswarm DHT (even on one machine,
// the bytes go through the real DHT/UDX stack — this is the same path two
// machines use; the two-machine finale is the identical commands on two boxes,
// documented in peer.mjs's header).
//
//   host peer ──TCP──> holesail-server ──DHT/UDX──> holesail-client <──TCP── join peer
//
// Scenario: host writes dev-a's ops, join writes dev-b's ops (both live), the
// linearizer merges, both peers must converge byte-identical and the reducer
// state must equal the Wave-0 golden.
//
// Run: npm run wave1:holesail   (DHT bootstrap can take ~10-30s)

import { spawn } from 'node:child_process'
import readline from 'node:readline'
import { mkdtempSync, rmSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const PEER = join(__dirname, 'peer.mjs')
const REDUCER_GOLDEN = join(__dirname, '..', 'goldens', 'inventory_basic.json')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

const DEV_A_OPS = [
  { seq: 1, actor: 'dev-a', sku: 'TX-100', delta: 10, ts: 100 },
  { seq: 2, actor: 'dev-a', sku: 'TX-100', delta: -6, ts: 200 },
  { seq: 1, actor: 'dev-a', sku: 'PH-200', delta: 3, ts: 120 },
]
const DEV_B_OPS = [
  { seq: 1, actor: 'dev-b', sku: 'TX-100', delta: -6, ts: 150 },
  { seq: 2, actor: 'dev-b', sku: 'PH-200', delta: 4, ts: 220 },
]
const TOTAL = DEV_A_OPS.length + DEV_B_OPS.length

/** Wrap a spawned peer with a JSON-line event queue + waiters. */
function wrapPeer(name, argv) {
  const child = spawn(process.execPath, argv, { stdio: ['pipe', 'pipe', 'pipe'] })
  const events = []
  const waiters = []
  readline.createInterface({ input: child.stdout }).on('line', (line) => {
    let ev
    try { ev = JSON.parse(line) } catch { return } // ignore non-JSON noise
    events.push(ev)
    for (let i = waiters.length - 1; i >= 0; i--) {
      if (waiters[i].match(ev)) waiters.splice(i, 1)[0].resolve(ev)
    }
  })
  child.stderr.on('data', (d) => {
    const s = String(d)
    if (!s.includes('ExperimentalWarning') && !s.includes('trace-warnings')) {
      process.stderr.write(`[${name}] ${s}`)
    }
  })
  return {
    name,
    child,
    send: (line) => child.stdin.write(line + '\n'),
    wait(match, timeout = 60000, label = 'event') {
      const past = events.find(match)
      if (past) return Promise.resolve(past)
      return this.waitNext(match, timeout, label)
    },
    /** Like wait() but only matches events that arrive AFTER this call. */
    waitNext(match, timeout = 60000, label = 'event') {
      return new Promise((resolve, reject) => {
        const t = setTimeout(() => reject(new Error(`[${name}] timed out waiting for ${label}`)), timeout)
        waiters.push({ match, resolve: (ev) => { clearTimeout(t); resolve(ev) } })
      })
    },
    kill: () => { try { child.kill() } catch {} },
  }
}

console.log('Sovereign Mesh — Wave 1 gate, stage 2: two processes over a REAL Holesail tunnel\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-wave1-hs-'))
const host = wrapPeer('host', [PEER, 'host', '--storage', join(tmp, 'host'), '--tcp-port', '49222'])
let joiner = null

try {
  const hostReady = await host.wait((e) => e.event === 'ready', 60000, 'host ready (Holesail server up)')
  console.log(`  · host up — tunnel ${hostReady.url.slice(0, 24)}…`)

  joiner = wrapPeer('join', [
    PEER, 'join',
    '--storage', join(tmp, 'join'),
    '--url', hostReady.url,
    '--base-key', hostReady.baseKey,
    '--tcp-port', '49223',
  ])
  const joinReady = await joiner.wait((e) => e.event === 'ready', 90000, 'join ready (DHT connect)')
  check('transport: join peer connected through the Holesail tunnel', joinReady.baseKey === hostReady.baseKey)

  host.send(`add-writer ${joinReady.writerKey}`)
  await joiner.wait((e) => e.event === 'writable', 60000, 'join peer granted write access')
  check('capability: writer grant replicated through the tunnel (join is writable)', true)

  for (const op of DEV_A_OPS) host.send(`append ${JSON.stringify(op)}`)
  for (const op of DEV_B_OPS) joiner.send(`append ${JSON.stringify(op)}`)

  // Poll both peers until both linearize all ops with identical digests.
  // waitNext (fresh events only) so each round reads THIS round's digests.
  let hostDigest, joinDigest
  const deadline = Date.now() + 90000
  for (;;) {
    const nextHost = host.waitNext((e) => e.event === 'digest', 30000, 'host digest')
    const nextJoin = joiner.waitNext((e) => e.event === 'digest', 30000, 'join digest')
    host.send('digest'); joiner.send('digest')
    ;[hostDigest, joinDigest] = await Promise.all([nextHost, nextJoin])
    if (hostDigest.viewLength === TOTAL && joinDigest.viewLength === TOTAL &&
        hostDigest.viewDigest === joinDigest.viewDigest) break
    if (Date.now() > deadline) break
    await new Promise((r) => setTimeout(r, 500))
  }

  check(`convergence: both peers linearized all ${TOTAL} ops`,
    hostDigest.viewLength === TOTAL && joinDigest.viewLength === TOTAL,
    `host ${hostDigest.viewLength}, join ${joinDigest.viewLength}`)
  check('convergence: view digests byte-identical across processes',
    hostDigest.viewDigest === joinDigest.viewDigest)
  check('state: TX-100 == 4 and PH-200 == 7 on both peers',
    hostDigest.stock['TX-100'] === 4 && hostDigest.stock['PH-200'] === 7 &&
    joinDigest.stock['TX-100'] === 4 && joinDigest.stock['PH-200'] === 7)
  check('invariant: exactly 1 oversell rejected, same loser on both (dev-a seq 2)',
    [hostDigest, joinDigest].every((d) => d.rejected.length === 1 &&
      d.rejected[0].actor === 'dev-a' && d.rejected[0].seq === 2))

  const reducerGolden = JSON.parse(readFileSync(REDUCER_GOLDEN, 'utf8'))
  check('golden: state digest equals the Wave 0 reducer golden',
    hostDigest.stateDigest === reducerGolden.digest && joinDigest.stateDigest === reducerGolden.digest)

  console.log(`\nview digest:  ${hostDigest.viewDigest}`)
  console.log(`state digest: ${hostDigest.stateDigest}`)

  host.send('exit'); joiner.send('exit')
  await new Promise((r) => setTimeout(r, 1500))
} catch (e) {
  failures++
  console.log(`  ✗ ${e.message}`)
} finally {
  host.kill(); if (joiner) joiner.kill()
  try { rmSync(tmp, { recursive: true, force: true }) } catch {}
}

console.log(failures === 0 ? '\nWAVE 1 STAGE 2 GREEN ✅' : `\nWAVE 1 STAGE 2 RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
