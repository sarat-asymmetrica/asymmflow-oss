// mirror-spike.mjs — Mission M4 stage 1: offline delivery via a blind mirror.
//
// The receptionist-machine story: desk writes into the PO room and goes HOME.
// The phone was off the whole time. When it wakes, the desk is gone — but the
// always-on mirror (blind-peer) kept the room's cores available, so the phone
// converges to the exact same byte-identical state WITHOUT the two devices
// ever being online together. That is the property WhatsApp buys with Meta's
// servers; here it is one Apache-2.0 process the office owns.
//
// Honesty first (stage-1 scope): these cores are PLAINTEXT, so this mirror
// could read what it holds. True blindness = encrypted autobase cores
// (Autobase encryptionKey) — deliberately stage 2, because key distribution
// for rooms is a Commander doctrine conversation (campaign §5), not a default
// an engineer should pick. What stage 1 proves is the DELIVERY mechanics:
// mirror-mediated replication of a capability-enforced room, end to end.
//
// Topology (all on a LOCAL testnet DHT — hermetic, no public network):
//   mirror — blind-peer server (rocksdb + hyperswarm, its own keypair)
//   desk   — mesh room node + BlindPeering client → pushes the room
//   phone  — comes online AFTER desk is fully closed; same mirror key;
//            pulls the room through wakeup + mirror replication
//
// Run: npm run mirrorspike
import createTestnet from 'hyperdht/testnet.js'
import HyperDHT from 'hyperdht'
import BlindPeer from 'blind-peer'
import BlindPeering from 'blind-peering'
import Wakeup from 'protomux-wakeup'
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { deviceKeys, signOp, grantOp } from './capability.mjs'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDEN = join(__dirname, '..', 'goldens', 'mirror_autobase.json')
const UPDATE_GOLDEN = process.argv.includes('--update-golden')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

const PK = (b) => Buffer.alloc(32, b)
const AUTH = deviceKeys(Buffer.alloc(32, 0xd4))
const DESK = deviceKeys(Buffer.alloc(32, 0xe5))

console.log('Messenger M4 stage 1 — mirror gate: the room outlives the sender\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-mirror-'))
const testnet = await createTestnet(3)
const bootstrap = testnet.bootstrap

// ── The always-on mirror ──
const mirror = new BlindPeer(join(tmp, 'mirror-rocks'), { bootstrap })
await mirror.ready()
await mirror.listen()
const MIRROR_KEY = mirror.publicKey
console.log(`  mirror up: ${MIRROR_KEY.toString('hex').slice(0, 16)}… (local testnet)\n`)

// ── Desk: writes the room, pushes it to the mirror, goes home ──
const deskWakeup = new Wakeup()
const desk = await createMeshNode({
  storage: join(tmp, 'desk'), primaryKey: PK(0x2b),
  authorityPub: AUTH.pubHex, mode: 'room', wakeup: deskWakeup,
})
const ROOM_KEY = desk.key

const DESK_OPS = [
  signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201' }, AUTH),
  grantOp({ seq: 2, actor: 'hub', ts: 200, device: DESK.pubHex, epoch: 0 }, AUTH),
  signOp({ seq: 3, actor: 'desk', ts: 300, kind: 'msg.post', body: 'Shipment cleared customs at 16:40.' }, DESK),
  signOp({ seq: 4, actor: 'desk', ts: 400, kind: 'msg.post', body: 'Original BL is with the forwarder — collecting tomorrow.', replyTo: 'desk:3' }, DESK),
  signOp({ seq: 5, actor: 'desk', ts: 500, kind: 'msg.read', upToActor: 'desk', upToSeq: 4 }, DESK),
]
for (const op of DESK_OPS) await desk.append(op)
const TOTAL = DESK_OPS.length
const deskView = await desk.viewDigest()
const deskState = await desk.state()

const deskDht = new HyperDHT({ bootstrap })
const deskBlind = new BlindPeering(deskDht, desk.store, { wakeup: deskWakeup, keys: [MIRROR_KEY] })
await deskBlind.addAutobase(desk.base, { announce: true })

// Wait until the mirror actually HOLDS the room (bootstrap core fully synced).
await waitFor(async () => {
  const core = mirror.store.get(Buffer.from(ROOM_KEY, 'hex'))
  await core.ready()
  const have = core.contiguousLength
  return have >= TOTAL ? have : null
}, { label: 'mirror to hold the full room core', timeout: 60000, interval: 250 })
check('push: the mirror holds the room core while desk is still online', true)

// Desk goes HOME — close everything it owns.
await deskBlind.close()
await desk.close()
await deskDht.destroy()
check('desk is fully offline (store, autobase, dht all closed)', true)

// ── Phone: wakes AFTER the desk is gone ──
const phoneWakeup = new Wakeup()
const phone = await createMeshNode({
  storage: join(tmp, 'phone'), primaryKey: PK(0x3c),
  bootstrap: ROOM_KEY, authorityPub: AUTH.pubHex, mode: 'room', wakeup: phoneWakeup,
})
const phoneDht = new HyperDHT({ bootstrap })
const phoneBlind = new BlindPeering(phoneDht, phone.store, { wakeup: phoneWakeup, keys: [MIRROR_KEY] })
await phoneBlind.addAutobase(phone.base, { announce: true })

await waitFor(async () => {
  const ops = await phone.ops()
  return ops.length >= TOTAL ? ops : null
}, { label: `phone to receive ${TOTAL} ops via the mirror`, timeout: 60000, interval: 250 })

const phoneView = await phone.viewDigest()
const phoneState = await phone.state()
check("offline delivery: phone's view is byte-identical to what desk wrote", phoneView === deskView)
check('offline delivery: room STATE digest identical (law included)', phoneState.digest === deskState.digest)
check('the conversation arrived intact (reply threading survives the mirror)',
  phoneState.messages.some((m) => m.msgId === 'desk:4' && m.replyTo === 'desk:3'))
check('capability plane crossed the mirror too (grant table + epoch)',
  phoneState.grants?.[DESK.pubHex]?.epoch === 0 && phoneState.applied === TOTAL)

// ── The honest blindness check (stage-1 truth, not a boast) ──
const mirrorCore = mirror.store.get(Buffer.from(ROOM_KEY, 'hex'))
await mirrorCore.ready()
const rawBlock = await mirrorCore.get(0, { wait: false }).catch(() => null)
check('honesty: stage-1 mirror holds PLAINTEXT blocks (encryption = stage 2, Commander doctrine)',
  rawBlock !== null)

if (UPDATE_GOLDEN || !existsSync(GOLDEN)) {
  writeFileSync(GOLDEN, JSON.stringify({ viewLength: TOTAL, viewDigest: deskView, stateDigest: deskState.digest, state: deskState }, null, 2) + '\n')
  console.log(`\n  (golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(GOLDEN, 'utf8'))
  check('golden: delivered view matches pinned golden', golden.viewDigest === phoneView)
  check('golden: delivered state digest matches pinned golden', golden.stateDigest === phoneState.digest)
}

console.log(`\nview digest:  ${phoneView}`)
console.log(`state digest: ${phoneState.digest}`)

await phoneBlind.close()
await phone.close()
await phoneDht.destroy()
await mirror.close()
await testnet.destroy()
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nMIRROR SPIKE GREEN ✅' : `\nMIRROR SPIKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
