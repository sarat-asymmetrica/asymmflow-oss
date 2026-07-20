// bare-probe.mjs — Phase 3 "Probe and Anchor Roles" (bare-runtime campaign):
// the Bare-native port of kit/probe.mjs (Mission A2 "The Corridor", Band 1 —
// mesh/docs/MISSION_A2_CORRIDOR_SPEC.md §Band 1). Same five checks, same
// verdict logic, same PLAIN-LANGUAGE VOCABULARY, byte-identical where the
// underlying check is portable — the campaign charter names the diagnostic
// vocabulary a REQUIREMENT on the successor, not a suggestion, because it is
// what a non-technical field contact reads aloud over the phone when
// something is wrong. Every PASS:/FAIL:/INFO: line, the CGNAT card wording,
// z32Groups()'s grouping, and the three verdict words (CORRIDOR GREEN/AMBER/
// RED) are copied from probe.mjs unchanged, not paraphrased.
//
// WHAT PORTS TODAY, WHAT DOESN'T (stated up front, honestly — see
// PHASE3_PROBE_ANCHOR_REPORT.md §2/§5 for the full accounting):
//   Checks 1-4 (DHT bootstrap, NAT self-diagnosis, CGNAT card, hyperswarm
//   punch test) port COMPLETELY — hyperdht, hyperswarm, and
//   hypercore-id-encoding all resolve and run under Bare with zero shims
//   (confirmed, PHASE0_NOTES_D_REVERIFY.md §2.3's require-check.mjs, 11/11
//   OK). `#crypto`'s `bare` condition (mesh/package.json) already maps to
//   `bare-crypto`, which exports `randomBytes` — no substitute needed there.
//   Check 5 (the OPTIONAL --holesail loopback spot-check) does NOT port
//   today: it needs a TCP loopback pair (`net.createServer`/`net.connect` in
//   the original) to prove the tunnel echoes correctly, and there is no
//   `node:net` alias for Bare (unlike `#fs`/`#crypto`/`#apply`) — `bare-tcp`
//   exists in this tree ONLY as an undeclared transitive dependency (via
//   holesail's own dependency chain and bare's bare-subprocess), not as an
//   explicit `mesh/package.json` dependency. Importing an undeclared
//   transitive package directly would be exactly the fragility this
//   campaign's own binding rules exist to avoid — so this file does not do
//   it. If `bare-tcp` is added as an explicit devDependency (owner/gate
//   decision, not this coder's to make unilaterally), this check can port in
//   full; until then it degrades gracefully to an INFO skip line, using the
//   SAME code shape probe.mjs already uses for "holesail package itself is
//   absent" (checkHolesail's existing try/catch), extended one case further.
//
// FR-1a DISCIPLINE (this file's own hostile-geography finding, applied to
// itself): every optional/environment-dependent import is scoped inside a
// try/catch at the point of use, never a top-level `import`. A probe whose
// entire job is diagnosing a broken environment must never itself be the
// thing that crashes hard in that environment — PHASE0_NOTES_D_REVERIFY.md
// §4 pass (a) showed the require-check/wasi-check per-import try/catch
// pattern degrades to a readable FAIL report with zero dependencies present,
// while an unguarded top-level import crashes with a full stack trace
// (exit 127) — the ORIGINAL probe.mjs's own `holesail` import was exactly
// this failure class (FR-1a). This file has NO top-level imports of
// anything that is not proven to resolve under Bare in this repo today.
//
// No `node:` specifiers anywhere in this file (binding constraint, Phase 3
// dispatch). `#crypto` resolves to `bare-crypto` under Bare (mesh/package.json
// imports map). Path handling avoids `node:path`/`node:url` entirely — see
// the CLI entry point at the bottom for the manual argv/URL comparison this
// forces, mirroring apply-bare.mjs's own `new URL(..., import.meta.url)`
// pattern rather than `fileURLToPath`.
//
// CLI: npx bare kit/bare-probe.mjs [--listen | --dial <key>] [--holesail] [--json]
//      npx bare kit/bare-probe.mjs --self-test

import HyperDHT from 'hyperdht'
import Hyperswarm from 'hyperswarm'
import HypercoreID from 'hypercore-id-encoding'
import { randomBytes } from '#crypto'

const WATCHDOG_MS = 58000       // I: never hang >60s total — identical to probe.mjs
const DHT_READY_MS = 15000
const PUNCH_WAIT_MS = 45000
const PING_WAIT_MS = 8000

// ── pure helpers (self-test exercises these with ZERO network calls) ───────
// Every line below is COPIED from probe.mjs, not reworded — the diagnostic
// vocabulary is the contract (see file header).

/** z32 key, grouped in 4s — the one allowed "code" on the happy path (I3). */
export function z32Groups(z32) {
  return String(z32).match(/.{1,4}/g)?.join(' ') ?? String(z32)
}

/**
 * computeVerdict(state) -> { verdict: 'CORRIDOR GREEN'|'AMBER'|'RED', reason }
 * Pure function, byte-identical to probe.mjs's computeVerdict — same
 * fields, same thresholds, same reason strings. A field contact reading a
 * verdict off a Bare-hosted probe must see EXACTLY what they would have seen
 * off the Node one; a divergent wording here would itself be a field-support
 * hazard (I3), not merely an inconsistency.
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

/** Assembles the stable --json schema from a finished check state. Schema
 * tag deliberately distinct from probe.mjs's ('asymm-corridor-probe-bare.v1'
 * vs 'asymm-corridor-probe.v1') so a downstream ops-log consumer can tell
 * which runtime produced a given report — every OTHER field is identical
 * shape, so a consumer that doesn't care can treat them the same. */
export function toJsonReport(state, timestamp = new Date().toISOString()) {
  const { verdict, reason } = computeVerdict(state)
  return {
    schema: 'asymm-corridor-probe-bare.v1',
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

/** Line-framed JSON reader over a duplex stream — identical shape to
 * probe.mjs's onLine, reused for the punch test's socket (a hyperswarm
 * connection, which is a Node-shaped duplex stream under Bare too — proven
 * by require-check.mjs and this file's own gate). */
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

// ── the real, networked checks (1-4 identical logic to probe.mjs) ─────────

async function checkDht(say) {
  const dht = new HyperDHT()
  const dhtTotal = dht.bootstrapNodes.length
  try {
    await withTimeout(dht.ready(), DHT_READY_MS, 'DHT bootstrap')
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
      await swarm.flush().catch(() => {})

      const socket = await withTimeout(
        new Promise((resolve) => swarm.once('connection', (s) => { s.on('error', () => {}); resolve(s) })),
        PUNCH_WAIT_MS, 'punch test (no --dial arrived)',
      )
      say('PASS: a peer connected — echoing ping back')
      onLine(socket, (msg) => {
        if (msg?.type === 'ping') socket.write(JSON.stringify({ type: 'pong', nonce: msg.nonce }) + '\n')
      })
      await new Promise((r) => setTimeout(r, 2000))
      socket.destroy()
      return { punchAttempted: true, punchOk: true, punchDirect: null, punchRttMs: null, punchRole: 'listen' }
    }

    const topic = HypercoreID.decode(dial)
    swarm.join(topic, { server: false, client: true })
    const socket = await withTimeout(
      new Promise((resolve) => swarm.once('connection', (s, info) => { s.on('error', () => {}); resolve([s, info]) })).then(([s, info]) => ({ s, info })),
      PUNCH_WAIT_MS, 'punch test (could not reach the listener)',
    )
    const { s: socket2, info } = socket
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

/** Check 5, degraded (see file header §"WHAT PORTS TODAY"). Two DISTINCT
 * skip reasons, both reported honestly rather than collapsed into one vague
 * line — a field contact or support engineer reading `--json` output should
 * be able to tell "holesail isn't in this kit" apart from "holesail IS here
 * but this runtime can't loop it back yet" without guessing:
 *   - holesail itself unavailable (mirrors probe.mjs's existing behavior)
 *   - holesail available, but no TCP loopback primitive to prove it with
 *     (Bare-specific gap, not present in the Node original) */
async function checkHolesail(say) {
  try {
    await import('holesail')
  } catch {
    say('INFO: holesail check not included in this kit — skipping')
    return { holesailAttempted: false, holesailOk: null, holesailRttMs: null }
  }
  say('INFO: holesail loopback check needs a TCP loopback primitive not yet available under Bare in this kit (no node:net equivalent wired — see PHASE3_PROBE_ANCHOR_REPORT.md) — skipping')
  return { holesailAttempted: false, holesailOk: null, holesailRttMs: null }
}

// ── orchestration (identical shape to probe.mjs's runProbe) ───────────────

async function runProbe({ listen, dial, holesail, json }) {
  const lines = []
  const say = json ? () => {} : (line) => { lines.push(line); console.log(line) }

  say('Mission A2 Band 1 — The Corridor Probe (Bare runtime)')
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
    console.log(JSON.stringify(report))
  } else {
    say('')
    say(report.reason)
    say(report.verdict)
  }
  return report.verdict === 'CORRIDOR RED' ? 1 : 0
}

// ── --self-test: hermetic, zero network — identical assertions to probe.mjs
// (minus the holesail-specific dry-run line, which no longer applies) ──────

async function runSelfTest() {
  let failures = 0
  const check = (name, cond, detail = '') => {
    if (cond) console.log(`  [OK] ${name}`)
    else { failures++; console.log(`  [FAIL] ${name}${detail ? ' — ' + detail : ''}`) }
  }

  console.log('bare-probe.mjs self-test (hermetic — no network)\n')

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
    ['GREEN — holesail unattempted (skipped, either reason) never drags the verdict down',
      { dhtTotal: 3, dhtReachable: 3, firewalled: false, punchAttempted: false, holesailAttempted: false }, 'CORRIDOR GREEN'],
  ]
  for (const [name, fixture, expected] of fixtures) {
    const { verdict } = computeVerdict(fixture)
    check(name, verdict === expected, `got ${verdict}`)
  }

  const report = toJsonReport({
    dhtTotal: 3, dhtReachable: 3, firewalled: false, publicHost: '203.0.113.9', publicPort: 49222,
    punchAttempted: true, punchOk: true, punchDirect: true, punchRttMs: 42, punchRole: 'dial',
    holesailAttempted: false, holesailOk: null, holesailRttMs: null,
  })
  check('json report: bare-specific schema tag present', report.schema === 'asymm-corridor-probe-bare.v1')
  check('json report: verdict is one of the three exact words', ['CORRIDOR GREEN', 'CORRIDOR AMBER', 'CORRIDOR RED'].includes(report.verdict))
  check('json report: checks.dht/nat/punch/holesail all present',
    !!report.checks.dht && !!report.checks.nat && !!report.checks.punch && !!report.checks.holesail)
  check('json report: round-trips through JSON.stringify/parse losslessly',
    JSON.stringify(JSON.parse(JSON.stringify(report))) === JSON.stringify(report))

  const sampleLines = []
  const say = (line) => sampleLines.push(line)
  const key = HypercoreID.encode(randomBytes(32))
  say(`PASS: listening — give this key to the other side:`)
  say(`  KEY: ${z32Groups(key)}`)
  printCgnatCard(say, '203.0.113.9')
  say('PASS: DHT bootstrap reachable (3/3 servers responded)')
  say('PASS: this network is not firewalled (direct connections should work)')
  const HEX64 = /\b[0-9a-f]{64}\b/i
  check('output: no raw 64-hex key ever gets printed by the probe\'s own formatting helpers',
    !sampleLines.some((l) => HEX64.test(l)))

  console.log(failures === 0 ? '\nSELF-TEST GREEN' : `\nSELF-TEST RED (${failures} failure(s))`)
  return failures === 0 ? 0 : 1
}

// ── CLI entry point ─────────────────────────────────────────────────────
// No node:url/fileURLToPath — `process.argv[1]` (bare-process, an explicit
// mesh/package.json devDependency, confirmed identical in argv SHAPE to
// Node's — verified directly under a real bare.exe this session) is compared
// against `import.meta.url` by hand-converting the path to a file:// URL
// string (backslash -> forward-slash, `file:///` prefix) — the same
// information `fileURLToPath(import.meta.url) === process.argv[1]` would
// have compared, without importing a node: builtin to get it. `bare-process`
// itself is a plain top-level import here (not behind `#crypto`-style
// condition-map indirection) because this file is Bare-ONLY by construction
// (like host/bare-entry.mjs) — there is no Node line to stay compatible
// with, unlike apply-bare.mjs's dual-runtime `#fs`.
import process from 'bare-process'

const isMain = process.argv[1] && import.meta.url === 'file:///' + process.argv[1].replace(/\\/g, '/')
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
    console.error('bare-probe.mjs: hard watchdog fired (>58s) — forcing exit')
    process.exit(1)
  }, WATCHDOG_MS)
  watchdog.unref?.()

  if (args['self-test']) {
    const code = await runSelfTest()
    clearTimeout(watchdog)
    process.exit(code)
  } else if (args.listen && typeof args.dial === 'string') {
    console.error('bare-probe.mjs: pass either --listen or --dial <key>, not both')
    process.exit(2)
  } else {
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
      console.error(`bare-probe.mjs: fatal — ${err.message}`)
      clearTimeout(watchdog)
      process.exit(1)
    }
  }
}
