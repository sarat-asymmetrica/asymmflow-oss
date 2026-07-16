// missiond-holesail.mjs — Mission D, stage 2: the SAME grant/epoch ceremony the
// two-physical-box run performs, driven automatically between two OS processes
// over a REAL Holesail DHT tunnel. This is the dress rehearsal for the
// household-laptop ceremony: every command below is exactly what the humans
// type (peer.mjs header), in the same order.
//
// Beats:
//   1. join is admitted to the WRITER SET (pipe + linearizer open)…
//   2. …but its signed op is rejected on BOTH peers: no capability grant.
//   3. an UNSIGNED op (append-raw) is rejected on both: signature required.
//   4. authority grants the join device → its next op lands.
//   5. authority bumps the epoch → join's next op is stale-rejected everywhere.
//   6. authority re-grants at epoch 1 → join writes again.
// Convergence: byte-identical views + state on both processes at the end.
//
// Run: npm run missiond:holesail   (DHT bootstrap can take ~10-30s)

import { spawn } from 'node:child_process'
import readline from 'node:readline'
import { mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const PEER = join(__dirname, 'peer.mjs')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

/** Wrap a spawned peer with a JSON-line event queue + waiters. */
function wrapPeer(name, argv) {
  const child = spawn(process.execPath, argv, { stdio: ['pipe', 'pipe', 'pipe'] })
  const events = []
  const waiters = []
  readline.createInterface({ input: child.stdout }).on('line', (line) => {
    let ev
    try { ev = JSON.parse(line) } catch { return }
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
    waitNext(match, timeout = 60000, label = 'event') {
      return new Promise((resolve, reject) => {
        const t = setTimeout(() => reject(new Error(`[${name}] timed out waiting for ${label}`)), timeout)
        waiters.push({ match, resolve: (ev) => { clearTimeout(t); resolve(ev) } })
      })
    },
    /** Send a command and await its ok/error (sequences the ceremony beats). */
    async cmd(line, label = line) {
      const next = this.waitNext((e) => e.event === 'ok' || e.event === 'error', 30000, `ack of ${label}`)
      this.send(line)
      const ev = await next
      if (ev.event === 'error') throw new Error(`[${this.name}] ${label} failed: ${ev.error}`)
      return ev
    },
    kill: () => { try { child.kill() } catch {} },
  }
}

console.log('Sovereign Mesh — Mission D stage 2: the grant ceremony over a REAL Holesail tunnel\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-missiond-hs-'))
const host = wrapPeer('host', [PEER, 'host', '--storage', join(tmp, 'host'), '--tcp-port', '49232', '--authority'])
let joiner = null

try {
  const hostReady = await host.wait((e) => e.event === 'ready', 60000, 'host ready (Holesail server up)')
  check('authority: host publishes its authorityPub at ready', typeof hostReady.authorityPub === 'string')
  console.log(`  · host up — tunnel ${hostReady.url.slice(0, 24)}…`)

  joiner = wrapPeer('join', [
    PEER, 'join',
    '--storage', join(tmp, 'join'),
    '--url', hostReady.url,
    '--base-key', hostReady.baseKey,
    '--authority-pub', hostReady.authorityPub,
    '--tcp-port', '49233',
  ])
  const joinReady = await joiner.wait((e) => e.event === 'ready', 90000, 'join ready (DHT connect)')
  check('transport: join peer connected through the Holesail tunnel', joinReady.baseKey === hostReady.baseKey)
  const LAPTOP = joinReady.devicePub

  // Beat 1 — replication plane opens fully.
  host.send(`add-writer ${joinReady.writerKey}`)
  await joiner.wait((e) => e.event === 'writable', 60000, 'join admitted to the writer set')
  check('writer set: join peer admitted (pipe + linearizer open)', true)

  // Beat 2 — signed but UNGRANTED. Beat 3 — unsigned entirely.
  await joiner.cmd('append {"actor":"laptop","sku":"TX-100","delta":5}', 'pre-grant append')
  await joiner.cmd('append-raw {"actor":"intruder","sku":"TX-100","delta":7}', 'unsigned append')

  // Beat 4 — the capability grant, then a lawful write.
  await host.cmd(`grant ${LAPTOP}`, 'grant laptop@current-epoch')
  await joiner.cmd('append {"actor":"laptop","sku":"TX-100","delta":10}', 'post-grant append')

  // Beat 5 — revocation wave. Beat 6 — re-issue at the new epoch.
  await host.cmd('epoch 1', 'epoch bump')
  await joiner.cmd('append {"actor":"laptop","sku":"TX-100","delta":-4}', 'stale-epoch append')
  await host.cmd(`grant ${LAPTOP} 1`, 're-grant at epoch 1')
  await joiner.cmd('append {"actor":"laptop","sku":"TX-100","delta":-2}', 'post-re-grant append')

  const TOTAL = 8 // 4 laptop/intruder ops + 2 grants + 1 epoch + 0 (add-writer is not a view op)

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
  check('convergence: capability state digests byte-identical',
    hostDigest.stateDigest === joinDigest.stateDigest)

  const both = [hostDigest, joinDigest]
  check('pipe open, capability dead: pre-grant op rejected on BOTH ("no grant for device")',
    both.every((d) => d.rejected.some((r) => r.actor === 'laptop' && r.reason.includes('no grant for device'))))
  check('signature law: unsigned op rejected on BOTH ("unsigned op")',
    both.every((d) => d.rejected.some((r) => r.actor === 'intruder' && r.reason.includes('unsigned op'))))
  check('revocation: stale-epoch op rejected on BOTH ("is stale")',
    both.every((d) => d.rejected.some((r) => r.actor === 'laptop' && r.reason.includes('is stale'))))
  check('exactly 3 rejections, identical on both peers',
    both.every((d) => d.rejected.length === 3))
  check('lawful writes landed: TX-100 == 8 (+10 −2) on both',
    both.every((d) => d.stock['TX-100'] === 8))
  check('epoch converged at 1; laptop grant re-issued at epoch 1',
    both.every((d) => d.capEpoch === 1 && d.grants?.[LAPTOP]?.epoch === 1))

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

console.log(failures === 0 ? '\nMISSION D STAGE 2 GREEN ✅' : `\nMISSION D STAGE 2 RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
