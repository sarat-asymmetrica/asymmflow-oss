// kit-spike.mjs — Mission U1.5 gate: the full kitchen-table ceremony,
// driven end-to-end through the SAME command-layer functions kit-repl.mjs's
// readline loop calls (createCommandLayer — never screen-scraped), between
// TWO real kit-host.mjs instances with separate data directories, connected
// over the DIRECT TCP fallback path on localhost.
//
// HONEST SCOPE NOTE (mission brief's own permission: "an honest reduced
// scope beats a pretend-green"): this gate exercises the TCP fallback path
// ONLY, with hyperswarm disabled (`useHyperswarm: false`). hyperswarm/DHT
// replication is real, load-bearing production code in kit-net.mjs (wired,
// documented, ready for the actual two-machine kitchen-table run) but is
// NOT exercised here — a live DHT join needs real bootstrap nodes reachable
// over the internet, which is neither hermetic nor deterministic for a CI
// gate (and may simply be unavailable in a sandboxed run environment). The
// TCP fallback is the REQUIRED, LAN-only path per MSG-D24/the mission brief
// specifically because the kitchen-table test must not depend on internet
// DHT health — proving IT works hermetically is the correct gate; proving
// hyperswarm works is a manual smoke test the owner runs on the two real
// machines (README_KITCHEN_TABLE.txt says so plainly).
//
// Proves: device boot + persistent identity, room creation + a REAL
// asymm-room2 invite, the GL-9 cold-peer pairing ceremony (pairing code
// surfaces BEFORE the ceremony completes, addWriter is host-side via
// `server.rooms`, redeemInvite finishes once writable), converged posts
// both directions, a REAL file attach->fetch round trip with sha256
// verified end-to-end, transcript export to a real file, and — GL-5 reopen
// discipline — a genuine process-level restart of one device (close +
// recreate the SAME data dir) that recovers its room via kit-registry.mjs
// with full history intact, no in-memory state carried across.
//
// Run: npm run kitspike
import { mkdtempSync, rmSync, writeFileSync, readFileSync, existsSync, mkdirSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { createKitHost } from './kit-host.mjs'
import { createCommandLayer, createLiveStream } from './kit-repl.mjs'
import { loadRoomRegistry } from './kit-registry.mjs'
import { waitFor } from '../host/mesh-node.mjs'

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

console.log('Messenger U1.5 — kit spike: the kitchen-table ceremony, hermetic (TCP fallback path)\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-kit-'))
const dataA = join(tmp, 'machineA')
const dataB = join(tmp, 'machineB')

let hostA = await createKitHost({ dataDir: dataA, actor: 'ana', tcpPort: 0, useHyperswarm: false, log: () => {} })
let hostB = await createKitHost({ dataDir: dataB, actor: 'sam', tcpPort: 0, useHyperswarm: false, log: () => {} })
let cmdsA = createCommandLayer(hostA)
let cmdsB = createCommandLayer(hostB)

check('boot: ana has a device identity', /^[0-9a-f]{64}$/.test(hostA.keys.pubHex))
check('boot: sam has a device identity', /^[0-9a-f]{64}$/.test(hostB.keys.pubHex))
check('boot: distinct actor labels', hostA.actor === 'ana' && hostB.actor === 'sam')

// ── founder creates the room, TCP fallback listener comes up automatically ──
await cmdsA.create('kitchen table test')
const roomKey = hostA.currentRoomKey
check('create: room open on ana\'s device', typeof roomKey === 'string' && roomKey.length > 0)
const portA = hostA.tcpPorts.get(roomKey)
check('create: TCP fallback listening on an ephemeral port', Number.isInteger(portA) && portA > 0)

// ── founder mints a REAL, encrypted invite ──
const { inviteCode } = await cmdsA.invite()
check('invite: asymm-room2 code minted (this room is encrypted)', inviteCode.startsWith('asymm-room2.'))

// ── GL-9 cold-peer ceremony: pairing code observed BEFORE completion ──
let resolvePairing
const pairingCodePromise = new Promise((res) => { resolvePairing = res })
const joinPromise = cmdsB.join(inviteCode, { onPairingCode: (code) => resolvePairing(code) })
const pairingCode = await pairingCodePromise
check('join: pairing code (sam\'s writer key) surfaced before the ceremony completes', /^[0-9a-f]{64}$/.test(pairingCode))
check('join: sam\'s local room registered pre-writable', hostB.currentRoomKey === roomKey)

await cmdsB.connect(`127.0.0.1:${portA}`)
check('connect: sam dialed ana\'s TCP fallback listener directly', hostB.net.mode === 'tcp')

// ── U1.6 / F2 regression guard: ana's INBOUND-accepted-connection registry
// entry must remember her REAL bound port, never the requested "0" (OS-
// assigned) — a first draft of the onPeer wiring captured the wrong
// variable and would have poisoned auto-reconnect with "peer:0". ──
{
  const anaEntry = loadRoomRegistry(hostA.keysDir).find((r) => r.roomKey === roomKey)
  check('inbound peer-tracking: ana recorded a real port for sam, not the requested "0"',
    !!anaEntry?.lastPeer && !anaEntry.lastPeer.endsWith(':0') && anaEntry.lastPeer.endsWith(`:${portA}`))
}

await cmdsA.addWriter(pairingCode) // founder-side, host-only surface (server.rooms) — not a wire method
await joinPromise
check('join: redeemInvite landed — the real invite.offer/invite.redeem ceremony completed', true)

// ── converged posts, both directions ──
await cmdsA.post('hello from ana', 'whenever')
await waitFor(async () => (await hostB.client.request('roomState', { roomKey })).messages.some((m) => m.body === 'hello from ana'),
  { label: 'sam sees ana\'s post', timeout: 15000 })
check('converge: sam sees ana\'s post', true)

await cmdsB.post('hi ana — got it', 'today')
await waitFor(async () => (await hostA.client.request('roomState', { roomKey })).messages.some((m) => m.body === 'hi ana — got it'),
  { label: 'ana sees sam\'s reply', timeout: 15000 })
check('converge: ana sees sam\'s reply', true)

// ── U1.6 deliverable 8 (live thread display): post from ana while sam has
// the room open, and prove SAM'S OWN command-layer live-stream callback
// observes it with ZERO manual /open — the exact factored hook (never
// screen-scraped stdout) the REPL itself wires into startRepl(). ──
{
  // sam is "caught up" as of right now (mirrors what /open's own baseline
  // seeding does) — isolates this check to messages that arrive AFTER the
  // stream starts, not a replay of the two converged posts above.
  const caughtUp = await hostB.client.request('roomState', { roomKey })
  hostB._seenMsgIds = hostB._seenMsgIds || new Map()
  hostB._seenMsgIds.set(roomKey, new Set(caughtUp.messages.map((m) => m.msgId)))

  const seenLive = []
  const liveStreamB = createLiveStream(hostB, {
    onMessages: (rk, fresh, info) => { if (rk === roomKey) seenLive.push({ fresh, info }) },
  })
  await cmdsA.post('live thread test — can you see this without /open?', '')
  await waitFor(async () => seenLive.some((e) => e.fresh.some((m) => m.body === 'live thread test — can you see this without /open?')),
    { label: 'sam\'s live-stream callback observes ana\'s post with no manual /open', timeout: 15000 })
  const hit = seenLive.find((e) => e.fresh.some((m) => m.body === 'live thread test — can you see this without /open?'))
  check('live stream: sam\'s onMessages callback fired for the OPEN room (current: true)', hit?.info.current === true)
  check('live stream: the message body arrived intact through the stream, not just a count bump',
    hit.fresh.some((m) => m.actor === 'ana' && m.body === 'live thread test — can you see this without /open?'))
  liveStreamB.stop()
}

// ── real file attach -> fetch round trip, sha256 verified end-to-end ──
const fileBytes = Buffer.alloc(8192)
for (let i = 0; i < fileBytes.length; i++) fileBytes[i] = (i * 17 + 3) & 0xff
const srcPath = join(tmp, 'kitchen-table-photo.bin')
writeFileSync(srcPath, fileBytes)
const attachRes = await cmdsA.attach(srcPath, 'photo of the kitchen table')
check('attach: returns seq + ref + sha256', Number.isInteger(attachRes.seq) && attachRes.sha256.length === 64)

await waitFor(async () => (await hostB.client.request('roomState', { roomKey })).messages.some((m) => m.attachment?.ref === attachRes.ref),
  { label: 'sam sees the attach op', timeout: 15000 })

const savePath = join(tmp, 'fetched-photo.bin')
const fetchRes = await cmdsB.fetch(attachRes.ref, savePath)
check('fetch: sha256 verified end-to-end', fetchRes.verified === true && fetchRes.sha256 === attachRes.sha256)
check('fetch: bytes on disk are byte-identical to the original', readFileSync(savePath).equals(fileBytes))

// ── transcript export, a real file ──
const exportRes = await cmdsB.exportTranscript()
check('export: transcript file written to disk', existsSync(exportRes.path))
const bundle = JSON.parse(readFileSync(exportRes.path, 'utf8'))
check('export: bundle format + roomKey correct', bundle.format === 'asymm-transcript.v1' && bundle.roomKey === roomKey)

// ── U1.6 / F1 (MSG-D25): fetch robustness — every path the field failure
// could have taken, driven through the SAME cmds.fetch the REPL calls. ──
const defaultFetch1 = await cmdsB.fetch(attachRes.seq) // no savePath at all
check('fetch (F1): zero-argument fetch lands in data\\downloads\\<name>',
  defaultFetch1.path.startsWith(join(dataB, 'downloads')) && defaultFetch1.path.endsWith('kitchen-table-photo.bin'))
check('fetch (F1): zero-argument fetch is byte-identical', readFileSync(defaultFetch1.path).equals(fileBytes))

const defaultFetch2 = await cmdsB.fetch(attachRes.seq) // same attachment, second time — must NOT clobber
check('fetch (F1): collision-avoidance appends -1 on a repeat default fetch',
  defaultFetch2.path !== defaultFetch1.path && existsSync(defaultFetch1.path) && /-1\.bin$/.test(defaultFetch2.path))

const nestedPath = join(tmp, 'deep', 'nested', 'does-not-exist-yet', 'out.bin')
const nestedFetch = await cmdsB.fetch(attachRes.seq, nestedPath)
check('fetch (F1): nonexistent nested parent dir is created', nestedFetch.path === resolve(nestedPath) && readFileSync(nestedPath).equals(fileBytes))

const dirTarget = join(tmp, 'existing-download-dir')
mkdirSync(dirTarget, { recursive: true })
const dirFetch = await cmdsB.fetch(attachRes.seq, dirTarget)
check('fetch (F1): an existing directory as savePath resolves to dir/<original-name>',
  dirFetch.path === resolve(join(dirTarget, 'kitchen-table-photo.bin')) && readFileSync(dirFetch.path).equals(fileBytes))

// ── U1.6 / F2 (MSG-D25): restart sam's device — a REAL process-level
// close + recreate — and prove auto-reconnect happens with ZERO manual
// /connect (ana is still up on her ORIGINAL tcp port for this proof; her
// own restart, below, deliberately comes LAST so it doesn't invalidate the
// lastPeer address sam's registry just proved it can reconnect to). ──
await hostB.close()
hostB = await createKitHost({ dataDir: dataB, actor: 'sam', tcpPort: 0, useHyperswarm: false, log: () => {} })
cmdsB = createCommandLayer(hostB)
check('restart (F2): sam\'s room comes back via kit-registry.mjs', hostB.server.rooms.has(roomKey))
check('restart (F2): auto-reconnected to ana\'s last-known peer with NO /connect call', hostB.net.peerCount(roomKey) > 0)

await cmdsA.post('still here after sam restarted', '')
await waitFor(async () => (await hostB.client.request('roomState', { roomKey })).messages.some((m) => m.body === 'still here after sam restarted'),
  { label: 'sam (post-restart, auto-reconnected) sees a fresh post from ana', timeout: 15000 })
check('restart (F2): fresh replication resumed automatically after restart', true)

// ── U1.6 / F2: re-running /join with the SAME (now-exhausted) one-time
// invite must return the friendly already-joined path, never throw. ──
let rejoinThrew = false
let rejoinRes
try { rejoinRes = await cmdsB.join(inviteCode) } catch { rejoinThrew = true }
check('rejoin (F2): exhausted-invite retry does NOT throw', !rejoinThrew)
check('rejoin (F2): returns the friendly already-joined path', rejoinRes?.alreadyJoined === true && rejoinRes?.roomKey === roomKey)

// ── GL-5: restart ana's device — a REAL process-level close + recreate,
// not an in-memory reconnect — and prove the room + full history survive.
// (Deliberately LAST: this changes ana's TCP port, which is fine now that
// nothing above still depends on the original one.) ──
await hostA.close()
hostA = await createKitHost({ dataDir: dataA, actor: 'ana', tcpPort: 0, useHyperswarm: false, log: () => {} })
cmdsA = createCommandLayer(hostA)
check('reopen: the room comes back automatically via kit-registry.mjs', hostA.server.rooms.has(roomKey))
const reopened = await hostA.client.request('roomState', { roomKey })
check('reopen: manifest title survives', reopened.manifest?.title === 'kitchen table test')
check('reopen: both posts survive', reopened.messages.some((m) => m.body === 'hello from ana') && reopened.messages.some((m) => m.body === 'hi ana — got it'))
check('reopen: the attachment op survives (same sha256)', reopened.messages.some((m) => m.attachment?.sha256 === attachRes.sha256))

console.log('\nAll scenarios exercised over the real TCP fallback wire, including genuine process restarts (both devices) and auto-reconnect.')

await hostA.close()
await hostB.close()
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nKIT SPIKE GREEN ✅' : `\nKIT SPIKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
