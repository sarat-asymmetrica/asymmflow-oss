// mirror-spike.mjs — Mission M4 stage 2: the mirror goes BLIND.
//
// The receptionist-machine story: desk writes into the PO room and goes HOME.
// The phone was off the whole time. When it wakes, the desk is gone — but the
// always-on mirror (blind-peer) kept the room's cores available, so the phone
// converges to the exact same byte-identical state WITHOUT the two devices
// ever being online together. That is the property WhatsApp buys with Meta's
// servers; here it is one Apache-2.0 process the office owns.
//
// Stage 2 (owner-ratified doctrine, 2026-07-18: "key rides the invite,
// rotate-on-revoke"): the room's Autobase carries an `encryptionKey`
// (mesh-node.mjs; verified against the installed autobase source —
// autobase/index.js:341-368, autobase/lib/store.js:246-252 — that this
// encrypts the oplog AND every named view core end-to-end, not just
// transport). The key travels with the room key inside a SINGLE
// `asymm-room2.` invite code (invite-code.mjs) — one paste, capability AND
// content key together. The mirror now holds real ciphertext: it still
// delivers the room (blind-peering never needed to read it — that was
// always the point of "blind" peering), but it can no longer make sense of
// what it's carrying. What stage 1 proved was the DELIVERY mechanics;
// stage 2 proves the mirror is honestly blind while still doing its job.
//
// Topology (all on a LOCAL testnet DHT — hermetic, no public network):
//   mirror — blind-peer server (rocksdb + hyperswarm, its own keypair)
//   desk   — mesh room node (encrypted) + BlindPeering client → pushes the room
//   phone  — comes online AFTER desk is fully closed; same mirror key AND
//            the same room2 invite code (decoded, not hand-copied);
//            pulls the room through wakeup + mirror replication
//
// Run: npm run mirrorspike
import createTestnet from 'hyperdht/testnet.js'
import HyperDHT from 'hyperdht'
import BlindPeer from 'blind-peer'
import BlindPeering from 'blind-peering'
import Wakeup from 'protomux-wakeup'
import Corestore from 'corestore'
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { deviceKeys, signOp, grantOp } from './capability.mjs'
import { encodeInviteCode, decodeInviteCode } from './invite-code.mjs'
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

console.log('Messenger M4 stage 2 — the mirror goes blind\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-mirror-'))
const testnet = await createTestnet(3)
const bootstrap = testnet.bootstrap

// Synthetic fixture bytes for the room's content encryption key (canon rule,
// MSG-D13's precedent: fixtures are synthetic patterns, not real secrets).
// Deterministic on purpose — this spike proves the CEREMONY (encode on desk,
// decode on phone, key rides the invite), not key-generation randomness.
const ROOM_ENCRYPTION_KEY = Buffer.alloc(32, 0x77)

// ── The always-on mirror ──
const mirror = new BlindPeer(join(tmp, 'mirror-rocks'), { bootstrap })
await mirror.ready()
await mirror.listen()
const MIRROR_KEY = mirror.publicKey
console.log(`  mirror up: ${MIRROR_KEY.toString('hex').slice(0, 16)}… (local testnet)\n`)

// ── Desk: writes the room (ENCRYPTED), pushes it to the mirror, goes home ──
const deskWakeup = new Wakeup()
const desk = await createMeshNode({
  storage: join(tmp, 'desk'), primaryKey: PK(0x2b),
  authorityPub: AUTH.pubHex, mode: 'room', wakeup: deskWakeup,
  encryptionKey: ROOM_ENCRYPTION_KEY,
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

// ── The key rides the invite: desk encodes ONE asymm-room2 code carrying
// the room key, the authority, AND the content encryption key. The invite
// plane's own fields (inviteSeed/inviteId) are along for the ride here —
// this spike proves the KEY ceremony, not grant redemption (invite-spike.mjs
// owns that). ──
const INVITE_SEED = Buffer.alloc(32, 0x9a) // synthetic fixture, unused beyond the code envelope here
const roomInviteCode = encodeInviteCode({
  baseKey: ROOM_KEY, authorityPub: AUTH.pubHex, inviteSeed: INVITE_SEED, inviteId: 'hub:99',
  encryptionKey: ROOM_ENCRYPTION_KEY,
})
check('key-rides-the-invite: the code is versioned asymm-room2 (encrypted room)', roomInviteCode.startsWith('asymm-room2.'))

const decodedInvite = decodeInviteCode(roomInviteCode)
check('key-rides-the-invite: decode recovers the room key + authority unchanged',
  decodedInvite.baseKey === ROOM_KEY && decodedInvite.authorityPub === AUTH.pubHex)
check('key-rides-the-invite: decode recovers the exact content encryption key',
  Buffer.compare(decodedInvite.encryptionKey, ROOM_ENCRYPTION_KEY) === 0)

// ── Phone: wakes AFTER the desk is gone, joins using ONLY the decoded invite ──
const phoneWakeup = new Wakeup()
const phone = await createMeshNode({
  storage: join(tmp, 'phone'), primaryKey: PK(0x3c),
  bootstrap: decodedInvite.baseKey, authorityPub: decodedInvite.authorityPub, mode: 'room', wakeup: phoneWakeup,
  encryptionKey: decodedInvite.encryptionKey,
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

// ── The honest blindness check (stage-2 truth — this is what flips) ──
// Stage 1 asserted the mirror's raw blocks WERE plaintext. Stage 2 must prove
// the opposite: the mirror holds real ciphertext, and even a fresh peer that
// pulls those exact bytes without the key gets nothing readable.
const KNOWN_PLAINTEXT = Buffer.from('Shipment cleared')

const mirrorCore = mirror.store.get(Buffer.from(ROOM_KEY, 'hex'))
await mirrorCore.ready()
const rawBlock = await mirrorCore.get(0, { wait: false }).catch(() => null)
check('honesty: the mirror still HOLDS the block (delivery mechanics unchanged, MSG-D15)', rawBlock !== null)
check("honesty: the mirror's raw block does NOT contain the known plaintext (real blindness, not stage-1's boast)",
  rawBlock !== null && !rawBlock.includes(KNOWN_PLAINTEXT))

// A THIRD node — no encryptionKey at all — pulls the SAME bytes directly off
// the mirror's store via plain in-process replication (the mesh-node.mjs
// connect() pattern) and still cannot see the plaintext. This is the
// strongest "keyless probe" the installed APIs allow: it is not reading
// mirror.store's cache, it independently downloads the block over the wire.
const probeStore = new Corestore(join(tmp, 'probe'), { primaryKey: PK(0x4d), unsafe: true })
const probeCore = probeStore.get(Buffer.from(ROOM_KEY, 'hex')) // deliberately NO encryption option
await probeCore.ready()
const probeS1 = mirror.store.replicate(true)
const probeS2 = probeStore.replicate(false)
probeS1.pipe(probeS2).pipe(probeS1)
// wait:true (default) actively requests block 0 over the fresh replication
// stream — contiguousLength alone never moves without a real block request.
const probeTimeout = new Promise((_, reject) =>
  setTimeout(() => reject(new Error('keyless probe: block 0 never arrived')), 15000))
const probeBlock = await Promise.race([probeCore.get(0), probeTimeout])
check('honesty: a keyless THIRD node, independently pulling the same room, also cannot see the plaintext',
  probeBlock !== null && !probeBlock.includes(KNOWN_PLAINTEXT))
probeS1.destroy(); probeS2.destroy()
await probeStore.close()

// And the peer that DOES hold the key reads it fine — encryption is real,
// not merely "the mirror never bothered to decode."
check('the phone (holding the decoded key) reads the known plaintext just fine',
  phoneState.messages.some((m) => m.body === 'Shipment cleared customs at 16:40.'))

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
