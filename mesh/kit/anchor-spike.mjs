// anchor-spike.mjs — Mission A2 "The Corridor" gate G3 (mesh/docs/
// MISSION_A2_CORRIDOR_SPEC.md §Band 3): the anchor started with a fixture
// room registry, replicating over local TCP with a counterpart process,
// surviving that counterpart being killed and restarted, heartbeat file
// advancing, --listen accepting a kit-net TCP connect, and install/uninstall
// scripts validated via --print-only. Hermetic — NO real DHT (useHyperswarm:
// false throughout, same honest-scope doctrine as kit-spike.mjs).
//
// Setup mirrors kit-spike.mjs's own ceremony for the first few steps
// (founder creates a room, mints an invite) because that IS how a real
// anchor gets its first room: a human runs the normal /join ceremony ONCE
// on the receptionist machine, then that SAME data dir is handed to
// anchor.mjs to run headless from then on. This spike proves exactly that
// handoff — a fixture data dir is populated via the ordinary kit-repl
// command layer, closed, then re-opened by anchorMain(), never by
// anchor.mjs improvising its own room creation.
//
// Run: node kit/anchor-spike.mjs
import { mkdtempSync, rmSync, readFileSync, existsSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'
import { createKitHost } from './kit-host.mjs'
import { createCommandLayer } from './kit-repl.mjs'
import { waitFor } from '../host/mesh-node.mjs'
import { anchorMain } from './anchor.mjs'

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

console.log('Mission A2 Band 3 — anchor spike: headless anchor over local TCP, hermetic\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-anchor-'))
const dataFounder = join(tmp, 'founder')
const dataAnchor = join(tmp, 'anchor')
const logs = []
const log = (m) => logs.push(m)

// ── founder creates the room, mints an invite (ordinary kit-repl ceremony) ──
const founder = await createKitHost({ dataDir: dataFounder, actor: 'founder', tcpPort: 0, useHyperswarm: false, log })
const cmdsFounder = createCommandLayer(founder)
await cmdsFounder.create('corridor spike room')
const roomKey = founder.currentRoomKey
const founderPort = founder.tcpPorts.get(roomKey)
check('setup: founder room created + TCP listener up', typeof roomKey === 'string' && Number.isInteger(founderPort))
const { inviteCode } = await cmdsFounder.invite()

// ── the anchor-to-be joins ONCE via the ordinary /join ceremony, then its
// process is closed — this is the fixture registry anchorMain() will load
// headlessly, exactly like a real anchor's first-ever human setup step ──
{
  const anchorJoiner = await createKitHost({ dataDir: dataAnchor, actor: 'anchor', tcpPort: 0, useHyperswarm: false, log })
  const cmdsAnchorJoiner = createCommandLayer(anchorJoiner)
  let resolvePairing
  const pairingCodePromise = new Promise((res) => { resolvePairing = res })
  const joinPromise = cmdsAnchorJoiner.join(inviteCode, { onPairingCode: (c) => resolvePairing(c) })
  const pairingCode = await pairingCodePromise
  await cmdsAnchorJoiner.connect(`127.0.0.1:${founderPort}`)
  await cmdsFounder.addWriter(pairingCode)
  await joinPromise
  check('setup: fixture anchor device joined the room via the ordinary ceremony', anchorJoiner.server.rooms.has(roomKey))
  await anchorJoiner.close() // release the corestore lock — anchorMain() reopens the SAME data dir next
}

// ── boot the REAL anchor.mjs headlessly against the fixture data dir ──
const heartbeatPath = join(dataAnchor, 'keys', 'anchor.log')
const listenPort = founderPort + 1000 // arbitrary, distinct from the founder's own listener
let liveCtx
const { done, requestShutdown } = anchorMain({
  dataDir: dataAnchor, actor: 'anchor', useHyperswarm: false, listenPort,
  heartbeatMs: 300, // short — this is a hermetic gate, not a 60s field run
  log,
  onBoot: (ctx) => { liveCtx = ctx },
})

await waitFor(async () => !!liveCtx, { label: 'anchor boots and exposes its live ctx', timeout: 10000 })
check('band 3.1: anchor loaded the room from the registry HEADLESSLY (no REPL, no re-join)', liveCtx.server.rooms.has(roomKey))
check('band 3.1: anchor never exited after boot', !(await Promise.race([done.then(() => true), sleep(50).then(() => false)])))

// ── heartbeat file: created at boot, next to the registry ──
await waitFor(() => existsSync(heartbeatPath), { label: 'heartbeat file created', timeout: 5000 })
const linesAtBoot = readFileSync(heartbeatPath, 'utf8').trim().split('\n').filter(Boolean)
check('band 3.1: heartbeat file lives next to the registry (data/keys/anchor.log)', linesAtBoot.length >= 1)
check('band 3.1: heartbeat line format is timestamp + peers + rooms, no plaintext/keys (I7)',
  /^\d{4}-\d{2}-\d{2}T.*Z peers=\d+ rooms=\d+ mode=\w+$/.test(linesAtBoot[0]) && !linesAtBoot[0].includes(roomKey) && !linesAtBoot[0].includes('corridor spike room'))

await waitFor(() => {
  const lines = readFileSync(heartbeatPath, 'utf8').trim().split('\n').filter(Boolean)
  return lines.length > linesAtBoot.length
}, { label: 'heartbeat advances on its own (60s field cadence, 300ms here)', timeout: 5000 })
check('band 3.1: heartbeat file advances over time without any external trigger', true)

// ── --listen: founder dials the anchor's TCP listener directly (the
// Era-1-style DuckDNS+port-forward path from spec item 3) ──
await cmdsFounder.connect(`127.0.0.1:${listenPort}`)
check('band 3.3: --listen accepted a kit-net TCP /connect from the founder', founder.net.mode === 'tcp')

await cmdsFounder.post('hello anchor, first message')
await waitFor(async () => {
  const state = await liveCtx.client.request('roomState', { roomKey })
  return state.messages.some((m) => m.body === 'hello anchor, first message')
}, { label: 'anchor replicates a message posted over --listen', timeout: 15000 })
check('band 3.1/3.3: message replicated into the running anchor over --listen, with anchor still up', true)

// ── counterpart (the founder) is KILLED and RESTARTED — the anchor must
// not exit and must resume replication once the founder reconnects ──
await founder.close()
const founder2 = await createKitHost({ dataDir: dataFounder, actor: 'founder', tcpPort: 0, useHyperswarm: false, log })
const cmdsFounder2 = createCommandLayer(founder2)
check('restart: founder counterpart came back with its room via kit-registry.mjs', founder2.server.rooms.has(roomKey))
check('restart: anchor did NOT exit while its counterpart was down/restarting', !(await Promise.race([done.then(() => true), sleep(50).then(() => false)])))

await cmdsFounder2.open(roomKey)
await cmdsFounder2.connect(`127.0.0.1:${listenPort}`)
await cmdsFounder2.post('hello again after restart')
await waitFor(async () => {
  const state = await liveCtx.client.request('roomState', { roomKey })
  return state.messages.some((m) => m.body === 'hello again after restart')
}, { label: 'anchor replicates a post-restart message', timeout: 15000 })
check('band 3.1: replication RESUMED after the counterpart was killed and restarted', true)
check('band 3.1: anchor is STILL up after full replication resumed (never exited)', !(await Promise.race([done.then(() => true), sleep(50).then(() => false)])))

await founder2.close()

// ── clean shutdown: SIGINT/SIGTERM-equivalent, only via requestShutdown() ──
requestShutdown()
await waitFor(async () => (await Promise.race([done.then(() => true), sleep(50).then(() => false)])), { label: 'anchor shuts down cleanly on request', timeout: 5000 })
check('band 3.1: anchor exits ONLY on an explicit shutdown request, and does so cleanly', true)

// ── after shutdown, the SAME data dir reopens with a plain kit-host — proves
// the anchor's corestore was closed cleanly (no stale lock) and both
// messages (pre- and post-restart) survived, i.e. real replication, not an
// in-memory illusion ──
{
  const reopened = await createKitHost({ dataDir: dataAnchor, actor: 'anchor', tcpPort: 0, useHyperswarm: false, log })
  check('teardown: anchor data dir reopens with no stale corestore lock', reopened.server.rooms.has(roomKey))
  const state = await reopened.client.request('roomState', { roomKey })
  check('teardown: both messages the anchor replicated while running are on disk',
    state.messages.some((m) => m.body === 'hello anchor, first message') &&
    state.messages.some((m) => m.body === 'hello again after restart'))
  await reopened.close()
}

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)) }

// ── install/uninstall scripts: --print-only validation (no elevation, no
// real Task Scheduler mutation — this must be safe to run in any CI) ──
if (process.platform === 'win32') {
  const kitDir = dirname(fileURLToPath(import.meta.url))
  try {
    const installOut = execFileSync('cmd.exe', ['/c', join(kitDir, 'install_anchor.cmd'), '--print-only'], { encoding: 'utf8', timeout: 20000 })
    check('band 3.2: install_anchor.cmd --print-only runs cleanly and prints a schtasks command', /schtasks/i.test(installOut))
    check('band 3.2: install_anchor.cmd --print-only never claims elevation was requested', /no elevation requested/i.test(installOut))
  } catch (err) {
    check('band 3.2: install_anchor.cmd --print-only runs cleanly', false, err.message)
  }
  try {
    const uninstallOut = execFileSync('cmd.exe', ['/c', join(kitDir, 'uninstall_anchor.cmd'), '--print-only'], { encoding: 'utf8', timeout: 20000 })
    check('band 3.2: uninstall_anchor.cmd --print-only runs cleanly and prints a schtasks command', /schtasks/i.test(uninstallOut))
  } catch (err) {
    check('band 3.2: uninstall_anchor.cmd --print-only runs cleanly', false, err.message)
  }
} else {
  console.log('  (skipping install/uninstall --print-only checks — not on win32)')
}

try { rmSync(tmp, { recursive: true, force: true }) } catch {}
try { rmSync(join(dirname(fileURLToPath(import.meta.url)), 'anchor_task.xml'), { force: true }) } catch {}

console.log(failures === 0 ? '\nANCHOR SPIKE GREEN ✅' : `\nANCHOR SPIKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
