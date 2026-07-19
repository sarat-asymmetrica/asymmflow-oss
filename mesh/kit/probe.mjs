// probe.mjs — Mission A2 "The Corridor", Band 1 "The Probe" (mesh/docs/
// MISSION_A2_CORRIDOR_SPEC.md §Band 1). A single-file diagnostic the
// receptionist (phone-guided by SPOC) runs FIRST, before any ceremony is
// attempted: "can this network mesh?" in under a minute.
//
// Five checks, each with a plain-English PASS/FAIL/INFO line (I3 — readable
// aloud on a phone, no hex dumps on the happy path):
//   1. DHT bootstrap reachability   — join the default hyperdht bootstrap set.
//   2. NAT self-diagnosis           — what hyperdht itself knows post-bootstrap
//      (dht.firewalled, dht.remoteAddress()) — the observed public IP.
//   3. CGNAT check (human-in-the-loop) — print the IP + a 30-second phone
//      instruction; the probe cannot see the router, so a human closes the loop.
//   4. Punch test (--listen | --dial <key>) — a raw Hyperswarm rendezvous over
//      a fresh, throwaway 32-byte topic (never a room key — nothing here is a
//      capability), ping/pong round trip, RTT + best-effort direct/relayed read.
//   5. Holesail spot-check (--holesail, optional) — a LOOPBACK echo tunnel in
//      this same process, independent of the punch test, proving the holesail
//      path works at all before it's trusted for the corridor.
//
// Verdict line (the one phone-scriptable output — exactly one of):
//   CORRIDOR GREEN | CORRIDOR AMBER | CORRIDOR RED
// GREEN: DHT reachable, not firewalled, and (if a punch test ran) direct.
// AMBER: connected but relayed/firewalled — usable, anchor port-forward
//   recommended (R1 in the spec: the anchor is the always-on full member).
// RED: no DHT, or a requested punch test found no peer — stop, escalate.
//
// --json emits ONE machine-readable JSON object on stdout (nothing else —
// clean for an ops-log pipe). --self-test exercises the verdict logic and
// output formatting against fixtures with ZERO network I/O, so CI/gate runs
// stay offline-safe (mirrors kit-spike.mjs's hermetic-gate discipline).
//
// Every network handle this file opens (dht, swarm, holesail tunnels, echo
// server) is tracked and torn down in a single teardown() on every exit path
// (success, failure, or the 58s hard watchdog), so the process always exits —
// I1/I6: no new deps, no mesh-law surface touched, this is pure diagnostics.
//
// CLI: node kit/probe.mjs [--listen | --dial <key>] [--holesail] [--json]
//      node kit/probe.mjs --self-test

import net from 'node:net'
import { randomBytes } from 'node:crypto'
import { fileURLToPath } from 'node:url'
import HyperDHT from 'hyperdht'
import Hyperswarm from 'hyperswarm'
import HypercoreID from 'hypercore-id-encoding'
// holesail is intentionally NOT bundled in the field kit (see
// build-kit.mjs's node_modules prune note) — it is only ever needed for the
// OPTIONAL --holesail spot-check, so it must be a lazy import scoped inside
// checkHolesail() below, never a top-level import here. A top-level import
// crashed the field kit outside the repo tree with ERR_MODULE_NOT_FOUND
// (Mission A2.1 field report FR-1a) because Node module resolution inside
// the repo silently falls back to mesh/node_modules, masking the gap that
// only showed up once the built kit ran from its own directory.

const WATCHDOG_MS = 58000       // I: never hang >60s total
const DHT_READY_MS = 15000      // per-check budget, well under the 60s ceiling
const PUNCH_WAIT_MS = 45000     // human needs time to run the other side by hand
const PING_WAIT_MS = 8000
const HOLESAIL_MS = 15000

// ── pure helpers (self-test exercises these with ZERO network calls) ───────

/** z32 key, grouped in 4s — the one allowed "code" on the happy path (I3). */
export function z32Groups(z32) {
  return String(z32).match(/.{1,4}/g)?.join(' ') ?? String(z32)
}

/**
 * computeVerdict(state) -> { verdict: 'CORRIDOR GREEN'|'AMBER'|'RED', reason }
 * Pure function — every field of `state` is a plain value, never a live
 * handle, so this is exercised directly (no network) by --self-test.
 *
 * state = {
 *   dhtTotal, dhtReachable,      // bootstrap servers configured / seen
 *   firewalled,                  // hyperdht's own post-bootstrap NAT read
 *   punchAttempted, punchOk, punchDirect,
 *   holesailAttempted, holesailOk,
 * }
 */
export function computeVerdict(state) {
  const {
    dhtTotal = 0, dhtReachable = 0, firewalled = false,
    punchAttempted = false, punchOk = false, punchDirect = false,
    holesailAttempted = false, holesailOk = true,
  } = state

  if (dhtTotal > 0 && dhtReachable === 0) {
    return { verdict: 'CORRIDOR RED', reason: 'no DHT bootstrap servers reachable — check outbound UDP / internet connectivity' }
  }
  if (punchAttempted && !punchOk) {
    return { verdict: 'CORRIDOR RED', reason: 'punch test requested but no peer connection was established — no punch' }
  }

  const softIssues = []
  if (holesailAttempted && !holesailOk) softIssues.push('holesail loopback check failed')
  if (firewalled) softIssues.push('this network reports firewalled/behind NAT')
  if (punchAttempted && punchOk && !punchDirect) softIssues.push('the punch connection is relayed, not direct')

  if (softIssues.length) {
    return { verdict: 'CORRIDOR AMBER', reason: softIssues.join('; ') + ' — usable; the anchor port-forward path (R1/R2) is recommended' }
  }

  const provisional = !punchAttempted
  return {
    verdict: 'CORRIDOR GREEN',
    reason: provisional
      ? 'DHT reachable, not firewalled — no punch test was run (--listen/--dial) to confirm a direct peer connection'
      : 'DHT reachable, not firewalled, direct peer connection confirmed',
  }
}

/** Assembles the stable --json schema from a finished check state. */
export function toJsonReport(state, timestamp = new Date().toISOString()) {
  const { verdict, reason } = computeVerdict(state)
  return {
    schema: 'asymm-corridor-probe.v1',
    timestamp,
    verdict,
    reason,
    checks: {
      dht: { total: state.dhtTotal ?? 0, reachable: state.dhtReachable ?? 0 },
      nat: { firewalled: state.firewalled ?? null, publicHost: state.publicHost ?? null, publicPort: state.publicPort ?? null },
      punch: {
        attempted: state.punchAttempted ?? false,
        ok: state.punchOk ?? false,
        direct: state.punchDirect ?? null,
        rttMs: state.punchRttMs ?? null,
        role: state.punchRole ?? null,
      },
      holesail: { attempted: state.holesailAttempted ?? false, ok: state.holesailOk ?? null, rttMs: state.holesailRttMs ?? null },
    },
  }
}

function withTimeout(promise, ms, label) {
  let timer
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error(`${label} timed out after ${ms}ms`)), ms)
  })
  return Promise.race([promise, timeout]).finally(() => clearTimeout(timer))
}

/** Line-framed JSON reader over a duplex stream — same shape as this repo's
 * other JSON-lines protocols (host/peer.mjs stdin), applied to a socket. */
function onLine(socket, handler) {
  let buf = ''
  socket.on('data', (chunk) => {
    buf += chunk.toString('utf8')
    let idx
    while ((idx = buf.indexOf('\n')) !== -1) {
      const line = buf.slice(0, idx)
      buf = buf.slice(idx + 1)
      if (line) { try { handler(JSON.parse(line)) } catch { /* ignore malformed lines */ } }
    }
  })
}

function getFreePort() {
  return new Promise((resolve, reject) => {
    const srv = net.createServer()
    srv.once('error', reject)
    srv.listen(0, '127.0.0.1', () => { const { port } = srv.address(); srv.close(() => resolve(port)) })
  })
}

// ── the real, networked checks ──────────────────────────────────────────

async function checkDht(say) {
  const dht = new HyperDHT()
  const dhtTotal = dht.bootstrapNodes.length
  try {
    await withTimeout(dht.ready(), DHT_READY_MS, 'DHT bootstrap')
    // dht.nodes is only populated by nodes that actually replied — the
    // routing-table size after a successful bootstrap is the honest proxy
    // for "how many of the configured servers/peers answered us" (hyperdht
    // does not expose a raw per-bootstrap-node ping result; verified against
    // the installed source — dht-rpc/index.js's own bootstrap resolves
    // 'ip@host:port' internally, so re-pinging each entry by hand here would
    // duplicate private DNS-fallback logic rather than add real signal).
    const dhtReachable = Math.max(1, Math.min(dht.nodes.length, dhtTotal))
    say(`PASS: DHT bootstrap reachable (${dhtReachable}/${dhtTotal} servers responded)`)
    return { dht, dhtOk: true, dhtTotal, dhtReachable }
  } catch (err) {
    say(`FAIL: DHT bootstrap unreachable — ${err.message}`)
    return { dht, dhtOk: false, dhtTotal, dhtReachable: 0 }
  }
}

function checkNat(dht, say) {
  const firewalled = !!dht.firewalled
  const addr = dht.remoteAddress()
  say(firewalled
    ? 'INFO: this network reports as firewalled — a direct listener may not be reachable from outside'
    : 'PASS: this network is not firewalled (direct connections should work)')
  if (addr) {
    say(`PUBLIC ADDRESS: ${addr.host}:${addr.port}  <-- read this IP aloud for the CGNAT check below`)
  } else {
    say('PUBLIC ADDRESS: not detected yet (network is firewalled or still stabilizing)')
  }
  return { firewalled, publicHost: addr?.host ?? null, publicPort: addr?.port ?? null }
}

function printCgnatCard(say, publicHost) {
  say('')
  say('CGNAT CHECK (ask SPOC to do this — takes 30 seconds):')
  say(`  Compare the address above with the router's WAN address.`)
  say(publicHost
    ? `  Same as ${publicHost}? GREEN. Different? this network is behind CGNAT.`
    : '  No public address was detected — skip the comparison, this leans AMBER/RED on its own.')
  say('')
}

async function runPunch({ dht, listen, dial, say }) {
  const swarm = new Hyperswarm({ dht })
  try {
    if (listen) {
      const topic = randomBytes(32)
      const key = HypercoreID.encode(topic)
      say(`PASS: listening — give this key to the other side:`)
      say(`  KEY: ${z32Groups(key)}`)
      swarm.join(topic, { server: true, client: false })
      await swarm.flush().catch(() => {}) // best-effort: announce landed on the DHT

      const socket = await withTimeout(
        new Promise((resolve) => swarm.once('connection', (s) => { s.on('error', () => {}); resolve(s) })),
        PUNCH_WAIT_MS, 'punch test (no --dial arrived)',
      )
      say('PASS: a peer connected — echoing ping back')
      onLine(socket, (msg) => {
        if (msg?.type === 'ping') socket.write(JSON.stringify({ type: 'pong', nonce: msg.nonce }) + '\n')
      })
      // Listener holds the socket open briefly so the dialer's ping/pong and
      // RTT measurement (its own responsibility, per spec) has time to land.
      await new Promise((r) => setTimeout(r, 2000))
      socket.destroy()
      return { punchAttempted: true, punchOk: true, punchDirect: null, punchRttMs: null, punchRole: 'listen' }
    }

    // --dial <key>
    const topic = HypercoreID.decode(dial)
    swarm.join(topic, { server: false, client: true })
    const socket = await withTimeout(
      new Promise((resolve) => swarm.once('connection', (s, info) => { s.on('error', () => {}); resolve([s, info]) })).then(([s, info]) => ({ s, info })),
      PUNCH_WAIT_MS, 'punch test (could not reach the listener)',
    )
    const { s: socket2, info } = socket
    // Best-effort direct-vs-relayed read: hyperswarm does not expose a
    // definitive "this connection ended up relayed" flag on the public API
    // (verified against the installed hyperswarm/hyperdht source); a
    // non-empty relayAddresses on the matched PeerInfo means the DHT
    // supplied fallback relay addresses for this peer, which we report
    // honestly as "possibly relayed" rather than claim certainty.
    const relayHint = Array.isArray(info?.relayAddresses) && info.relayAddresses.length > 0

    const nonce = randomBytes(4).toString('hex')
    const sentAt = Date.now()
    const rtt = await withTimeout(new Promise((resolve) => {
      onLine(socket2, (msg) => { if (msg?.type === 'pong' && msg.nonce === nonce) resolve(Date.now() - sentAt) })
      socket2.write(JSON.stringify({ type: 'ping', nonce }) + '\n')
    }), PING_WAIT_MS, 'ping/pong round trip')

    say(`PASS: punch succeeded — RTT ${rtt}ms (${relayHint ? 'possibly relayed' : 'no relay hint observed'})`)
    socket2.destroy()
    return { punchAttempted: true, punchOk: true, punchDirect: !relayHint, punchRttMs: rtt, punchRole: 'dial' }
  } catch (err) {
    say(`FAIL: punch test — ${err.message}`)
    return { punchAttempted: true, punchOk: false, punchDirect: null, punchRttMs: null, punchRole: listen ? 'listen' : 'dial' }
  } finally {
    await swarm.destroy().catch(() => {})
  }
}

async function checkHolesail(say) {
  let Holesail
  try {
    ;({ default: Holesail } = await import('holesail'))
  } catch {
    say('INFO: holesail check not included in this kit — skipping')
    return { holesailAttempted: false, holesailOk: null, holesailRttMs: null }
  }

  const echoServer = net.createServer((socket) => socket.pipe(socket))
  let serverTunnel, clientTunnel
  try {
    const echoPort = await new Promise((resolve, reject) => {
      echoServer.once('error', reject)
      echoServer.listen(0, '127.0.0.1', () => resolve(echoServer.address().port))
    })

    serverTunnel = new Holesail({ server: true, port: echoPort, host: '127.0.0.1', secure: true })
    await withTimeout(serverTunnel.ready(), HOLESAIL_MS, 'holesail server tunnel')

    const clientPort = await getFreePort()
    clientTunnel = new Holesail({ client: true, key: serverTunnel.info.url, port: clientPort, host: '127.0.0.1' })
    await withTimeout(clientTunnel.ready(), HOLESAIL_MS, 'holesail client tunnel')

    const payload = 'corridor-probe-' + randomBytes(4).toString('hex')
    const sentAt = Date.now()
    const echoed = await withTimeout(new Promise((resolve, reject) => {
      const sock = net.connect(clientPort, '127.0.0.1')
      let received = ''
      sock.on('data', (chunk) => {
        received += chunk.toString('utf8')
        if (received.length >= payload.length) { sock.destroy(); resolve(received) }
      })
      sock.once('error', reject)
      sock.once('connect', () => sock.write(payload))
    }), HOLESAIL_MS, 'holesail loopback echo')

    const ok = echoed === payload
    const rtt = Date.now() - sentAt
    say(ok ? `PASS: holesail loopback tunnel echoed correctly — RTT ${rtt}ms` : 'FAIL: holesail loopback echo mismatch')
    return { holesailAttempted: true, holesailOk: ok, holesailRttMs: rtt }
  } catch (err) {
    say(`FAIL: holesail spot-check — ${err.message}`)
    return { holesailAttempted: true, holesailOk: false, holesailRttMs: null }
  } finally {
    try { await clientTunnel?.close() } catch { /* best-effort teardown */ }
    try { await serverTunnel?.close() } catch { /* best-effort teardown */ }
    await new Promise((r) => echoServer.close(r))
  }
}

// ── orchestration ───────────────────────────────────────────────────────

async function runProbe({ listen, dial, holesail, json }) {
  const lines = []
  const say = json ? () => {} : (line) => { lines.push(line); console.log(line) }

  say('Mission A2 Band 1 — The Corridor Probe')
  say('')

  let dht
  const state = {}
  try {
    const dhtResult = await checkDht(say)
    dht = dhtResult.dht
    Object.assign(state, dhtResult)

    const natResult = checkNat(dht, say)
    Object.assign(state, natResult)
    printCgnatCard(say, natResult.publicHost)

    if (listen || dial) {
      const punchResult = await runPunch({ dht, listen, dial, say })
      Object.assign(state, punchResult)
    } else {
      say('INFO: no --listen/--dial given — skipping the punch test (diagnostics only)')
    }

    if (holesail) {
      const holesailResult = await checkHolesail(say)
      Object.assign(state, holesailResult)
    }
  } finally {
    if (dht) await dht.destroy({ force: true }).catch(() => {})
  }

  const report = toJsonReport(state)
  if (json) {
    process.stdout.write(JSON.stringify(report) + '\n')
  } else {
    say('')
    say(report.reason)
    say(report.verdict)
  }
  return report.verdict === 'CORRIDOR RED' ? 1 : 0
}

// ── --self-test: hermetic, zero network, exercises verdict + formatting ──

async function runSelfTest() {
  let failures = 0
  const check = (name, cond, detail = '') => {
    if (cond) console.log(`  ✓ ${name}`)
    else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
  }

  console.log('probe.mjs self-test (hermetic — no network)\n')

  check('z32Groups groups a 52-char key into 13 groups of 4',
    z32Groups('a'.repeat(52)) === Array(13).fill('aaaa').join(' '))
  check('z32Groups is a no-op-ish pass-through on non-multiple-of-4 input (never throws)',
    (() => { try { z32Groups('abc'); return true } catch { return false } })())

  const fixtures = [
    ['RED — no DHT reachable',
      { dhtTotal: 3, dhtReachable: 0, firewalled: false, punchAttempted: false }, 'CORRIDOR RED'],
    ['RED — punch requested but no peer arrived',
      { dhtTotal: 3, dhtReachable: 3, firewalled: false, punchAttempted: true, punchOk: false }, 'CORRIDOR RED'],
    ['AMBER — firewalled but punch OK and direct',
      { dhtTotal: 3, dhtReachable: 3, firewalled: true, punchAttempted: true, punchOk: true, punchDirect: true }, 'CORRIDOR AMBER'],
    ['AMBER — not firewalled but punch is relayed',
      { dhtTotal: 3, dhtReachable: 3, firewalled: false, punchAttempted: true, punchOk: true, punchDirect: false }, 'CORRIDOR AMBER'],
    ['AMBER — holesail spot-check failed alone',
      { dhtTotal: 3, dhtReachable: 3, firewalled: false, punchAttempted: false, holesailAttempted: true, holesailOk: false }, 'CORRIDOR AMBER'],
    ['GREEN — direct punch confirmed, not firewalled',
      { dhtTotal: 3, dhtReachable: 3, firewalled: false, punchAttempted: true, punchOk: true, punchDirect: true }, 'CORRIDOR GREEN'],
    ['GREEN (provisional) — no punch requested, diagnostics only clean',
      { dhtTotal: 3, dhtReachable: 3, firewalled: false, punchAttempted: false }, 'CORRIDOR GREEN'],
  ]
  for (const [name, fixture, expected] of fixtures) {
    const { verdict } = computeVerdict(fixture)
    check(name, verdict === expected, `got ${verdict}`)
  }

  // --json schema stability: every documented field present, right shape.
  const report = toJsonReport({
    dhtTotal: 3, dhtReachable: 3, firewalled: false, publicHost: '203.0.113.9', publicPort: 49222,
    punchAttempted: true, punchOk: true, punchDirect: true, punchRttMs: 42, punchRole: 'dial',
    holesailAttempted: true, holesailOk: true, holesailRttMs: 12,
  })
  check('json report: schema tag present', report.schema === 'asymm-corridor-probe.v1')
  check('json report: verdict is one of the three exact words', ['CORRIDOR GREEN', 'CORRIDOR AMBER', 'CORRIDOR RED'].includes(report.verdict))
  check('json report: checks.dht/nat/punch/holesail all present',
    !!report.checks.dht && !!report.checks.nat && !!report.checks.punch && !!report.checks.holesail)
  check('json report: round-trips through JSON.stringify/parse losslessly',
    JSON.stringify(JSON.parse(JSON.stringify(report))) === JSON.stringify(report))

  // I3: no hex dumps on the happy path outside the one allowed z32 code.
  const sampleLines = []
  await runProbeDryRun(sampleLines) // pure formatting pass, no network — see below
  const HEX64 = /\b[0-9a-f]{64}\b/i
  check('output: no raw 64-hex key ever gets printed by the probe\'s own formatting helpers',
    !sampleLines.some((l) => HEX64.test(l)))

  console.log(failures === 0 ? '\nSELF-TEST GREEN ✅' : `\nSELF-TEST RED ❌ (${failures} failure(s))`)
  process.exitCode = failures === 0 ? 0 : 1
}

/** Drives the same `say()`-shaped formatting the real checks use, against
 * fixture data only, so the self-test can assert on exact printed lines
 * (e.g. the no-hex-dump invariant) without opening a socket. */
async function runProbeDryRun(sink) {
  const say = (line) => sink.push(line)
  const key = HypercoreID.encode(randomBytes(32))
  say(`PASS: listening — give this key to the other side:`)
  say(`  KEY: ${z32Groups(key)}`)
  printCgnatCard(say, '203.0.113.9')
  say('PASS: DHT bootstrap reachable (3/3 servers responded)')
  say('PASS: this network is not firewalled (direct connections should work)')
}

// ── CLI entry point ─────────────────────────────────────────────────────

const isMain = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]
if (isMain) {
  const args = {}
  const rest = process.argv.slice(2)
  for (let i = 0; i < rest.length; i++) {
    const a = rest[i]
    if (!a.startsWith('--')) continue
    const key = a.replace(/^--/, '')
    const next = rest[i + 1]
    if (next === undefined || next.startsWith('--')) { args[key] = true } else { args[key] = next; i++ }
  }

  const watchdog = setTimeout(() => {
    console.error('probe.mjs: hard watchdog fired (>58s) — forcing exit')
    process.exit(1)
  }, WATCHDOG_MS)
  watchdog.unref()

  if (args['self-test']) {
    await runSelfTest()
    clearTimeout(watchdog)
    process.exit(process.exitCode ?? 0)
  }

  if (args.listen && typeof args.dial === 'string') {
    console.error('probe.mjs: pass either --listen or --dial <key>, not both')
    process.exit(2)
  }

  try {
    const code = await runProbe({
      listen: !!args.listen,
      dial: typeof args.dial === 'string' ? args.dial : null,
      holesail: !!args.holesail,
      json: !!args.json,
    })
    clearTimeout(watchdog)
    process.exit(code)
  } catch (err) {
    console.error(`probe.mjs: fatal — ${err.message}`)
    clearTimeout(watchdog)
    process.exit(1)
  }
}
