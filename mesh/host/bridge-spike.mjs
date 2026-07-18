// bridge-spike.mjs — Mission U1 gate: the sidecar protocol v0 seam driven
// end-to-end over the REAL wire (mesh/docs/MESSENGER_UI_CAMPAIGN.md §1), two
// in-process devices, each behind its OWN bridge-server.mjs + talking through
// bridge-client.mjs — exactly the shape the future frontend and the DP4
// sidecar will use, just localhost TCP instead of stdio.
//
// hub  — authority of an anchored PO room (opened directly, host-side — room
//        CREATION for anchored rooms is not a v0 method) AND founder of an
//        encrypted social room (created over the wire, createSocialRoom).
// desk — granted into the PO room directly (host-side cap.grant, same
//        precedent every prior spike uses), and joins the social room via a
//        REAL invite.offer/invite.redeem ceremony under encryption, driven
//        entirely through redeemInvite() on the wire.
//
// Proves: hello/listRooms/roomState, post (valid tags + the fold-skip
// surfacing through the bridge for an invalid one), claimRoom in the
// anchored room + the claim-skip proof in the social room, an
// attach()->fetchAttachment() round-trip verified end-to-end, exportTranscript
// verified via verify-transcript.mjs, topExpectation float-to-top logic in
// listRooms, room-updated events (self-observed AND replication-delivered),
// malformed-frame resilience, and GL-5 seq-continuation across a bridge
// restart on the same room.
//
// No new golden: this scenario is causally chained throughout (every op
// awaited before the next; no genuine concurrent fork), so GL-2 would permit
// one, but the room fold's own convergence/digest properties are already
// proven by room_autobase.json / social_autobase.json — a bridgespike golden
// would duplicate that coverage without proving anything new about the
// ADAPTER, which is what this mission actually gates (GL-4c: golden
// minimalism). Runtime asserts carry the whole proof instead.
//
// Run: npm run bridgespike
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { deviceKeys, signOp, grantOp } from './capability.mjs'
import { createBridgeServer } from './bridge-server.mjs'
import { createBridgeClient } from './bridge-client.mjs'
import { verifyTranscript } from './verify-transcript.mjs'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

function waitForEvent(client, event, { timeout = 15000 } = {}) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => { unsub(); reject(new Error(`timeout waiting for event ${event}`)) }, timeout)
    const unsub = client.on(event, (params) => { clearTimeout(timer); unsub(); resolve(params) })
  })
}

const PK = (b) => Buffer.alloc(32, b)

// Pinned identities + fixture keys → deterministic run (spike canon).
const HUB = deviceKeys(PK(0xd4))
const DESK = deviceKeys(PK(0xe5))
const SOCIAL_KEY = Buffer.alloc(32, 0x77) // MSG-D13/D18 synthetic fixture-key precedent
const INVITE_SEED = Buffer.alloc(32, 0x5d)

console.log('Messenger U1 — bridge spike: protocol v0 over the real wire\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-bridge-'))

// ── Anchored PO room, opened directly (not a v0 method) ──────────────────
const hubPo = await createMeshNode({ storage: join(tmp, 'hub-po'), primaryKey: PK(0x1a), authorityPub: HUB.pubHex, mode: 'room' })
const deskPo = await createMeshNode({ storage: join(tmp, 'desk-po'), primaryKey: PK(0x2b), bootstrap: hubPo.key, authorityPub: HUB.pubHex, mode: 'room' })
const poWire = hubPo.connect(deskPo)
await hubPo.addWriter(deskPo.writerKey)
await waitFor(async () => { await deskPo.base.update(); return deskPo.writable }, { label: 'desk writable in PO room' })

await hubPo.append(signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-9001 — Steel Coils', anchorType: 'po', anchorId: 'PO-9001' }, HUB))
await hubPo.append(grantOp({ seq: 2, actor: 'hub', ts: 200, device: DESK.pubHex, epoch: 0 }, HUB))
await waitFor(async () => (await deskPo.ops()).length >= 2, { label: 'desk sees manifest + grant' })

// ── Bridge servers, one per device ────────────────────────────────────────
const hubStoreDir = join(tmp, 'hub-store')
const deskStoreDir = join(tmp, 'desk-store')
let hubBridge = await createBridgeServer({ rooms: new Map([[hubPo.key, hubPo]]), actor: 'hub', deviceKeys: HUB, storageDir: hubStoreDir, port: 0 })
let deskBridge = await createBridgeServer({ rooms: new Map([[deskPo.key, deskPo]]), actor: 'desk', deviceKeys: DESK, storageDir: deskStoreDir, port: 0 })
let hubClient = await createBridgeClient({ port: hubBridge.port })
let deskClient = await createBridgeClient({ port: deskBridge.port })

// ── hello() ────────────────────────────────────────────────────────────────
const hubHello = await hubClient.request('hello')
const deskHello = await deskClient.request('hello')
check('hello: hub identity', hubHello.devicePub === HUB.pubHex && hubHello.actor === 'hub' && hubHello.version === 'v0')
check('hello: desk identity', deskHello.devicePub === DESK.pubHex && deskHello.actor === 'desk' && deskHello.version === 'v0')

// ── malformed frame resilience (§1: {ok:false}, never crashes the server) ──
hubClient.socket.write('this is not json at all\n')
await new Promise((r) => setTimeout(r, 200))
const stillAlive = await hubClient.request('hello')
check('malformed frame: server survives and keeps serving valid requests', stillAlive.devicePub === HUB.pubHex)

// ── listRooms() — anchored room visible on both sides ─────────────────────
const hubRoomsInitial = await hubClient.request('listRooms')
check('listRooms: hub sees the PO room as anchored', hubRoomsInitial.some((r) => r.roomKey === hubPo.key && r.kind === 'anchored' && r.anchorId === 'PO-9001'))

// ── roomState() — manifest + members + capEpoch ────────────────────────────
const poState0 = await hubClient.request('roomState', { roomKey: hubPo.key })
check('roomState: manifest title', poState0.manifest?.title === 'PO-9001 — Steel Coils')
check('roomState: desk is a current member (writer, epoch 0)',
  poState0.members.some((m) => m.devicePub === DESK.pubHex && m.role === 'writer' && m.epoch === 0))
check('roomState: capEpoch reported', poState0.capEpoch === 0)

// ── post(): the fold-skip surfaces through the bridge, verbatim ───────────
const badTag = await hubClient.request('post', { roomKey: hubPo.key, body: 'oops', expectation: 'someday', ts: 300 }).catch((e) => e)
check('post: an invalid expectation tag is a bridge ERROR carrying the fold\'s own skip reason',
  badTag instanceof Error && badTag.bridgeError === 'unknown expectation tag')

// ── post(): valid tags, mixed urgency, from both devices ──────────────────
await hubClient.request('post', { roomKey: hubPo.key, body: 'kickoff — anything blocking?', expectation: 'whenever', ts: 400 })
await deskClient.request('post', { roomKey: hubPo.key, body: 'need the spec sheet', expectation: 'today', ts: 500 })
await waitFor(async () => (await hubPo.ops()).length >= 4, { label: 'hub sees desk\'s post' })

// ── topExpectation float-to-top: still "today" (highest so far) ───────────
let hubRooms = await hubClient.request('listRooms')
check('listRooms: topExpectation is "today" before anything urgent lands',
  hubRooms.find((r) => r.roomKey === hubPo.key)?.topExpectation === 'today')

// hub escalates — urgent should now float to the top, from EITHER peer's view
const urgentPost = await deskClient.request('post', { roomKey: hubPo.key, body: 'stock arrives Thursday, need approval NOW', expectation: 'urgent', ts: 600 })
check('post: urgent message accepted', Number.isInteger(urgentPost.seq))
await waitFor(async () => (await hubPo.ops()).length >= 5, { label: 'hub sees the urgent post' })
hubRooms = await hubClient.request('listRooms')
const deskRooms = await deskClient.request('listRooms')
check('listRooms: topExpectation floats to "urgent" on hub\'s bridge', hubRooms.find((r) => r.roomKey === hubPo.key)?.topExpectation === 'urgent')
check('listRooms: topExpectation floats to "urgent" on desk\'s bridge too (same converged fold)', deskRooms.find((r) => r.roomKey === hubPo.key)?.topExpectation === 'urgent')

// ── claimRoom(): anchored room, the affirmative path ───────────────────────
const claim = await deskClient.request('claimRoom', { roomKey: hubPo.key, assignee: 'desk', ts: 700 })
check('claimRoom: desk self-claims the anchored room', Number.isInteger(claim.seq))
const poState1 = await deskClient.request('roomState', { roomKey: hubPo.key })
check('roomState: claim reflects desk as assignee', poState1.claim?.assignee === 'desk')
const release = await deskClient.request('releaseClaim', { roomKey: hubPo.key, ts: 800 })
check('releaseClaim: desk releases her own claim', Number.isInteger(release.seq))

// ── attach() -> fetchAttachment(): sha256 verified end-to-end ─────────────
const DOC_BYTES = Buffer.alloc(4096)
for (let i = 0; i < DOC_BYTES.length; i++) DOC_BYTES[i] = (i * 31 + 7) & 0xff
const docPath = join(tmp, 'PO-9001-spec.bin')
writeFileSync(docPath, DOC_BYTES)
const attachResult = await hubClient.request('attach', {
  roomKey: hubPo.key, filePath: docPath, body: 'spec sheet attached', expectation: 'today', ts: 900,
})
check('attach: returns seq, ref, sha256', Number.isInteger(attachResult.seq) && typeof attachResult.ref === 'string' && attachResult.sha256.length === 64)
await waitFor(async () => (await deskPo.ops()).length >= 8, { label: 'desk sees the attach op' })
const fetchedPath = join(tmp, 'fetched-spec.bin')
const fetched = await deskClient.request('fetchAttachment', { roomKey: hubPo.key, ref: attachResult.ref, savePath: fetchedPath })
check('fetchAttachment: sha256 verified end-to-end', fetched.verified === true && fetched.sha256 === attachResult.sha256)
check('fetchAttachment: bytes on disk are byte-identical to the original', readFileSync(fetchedPath).equals(DOC_BYTES))
const poStateWithAttachment = await hubClient.request('roomState', { roomKey: hubPo.key })
const attachedMsg = poStateWithAttachment.messages.find((m) => m.attachment?.ref === attachResult.ref)
check('roomState: attachment surfaces as {name,size,sha256,ref}',
  attachedMsg?.attachment?.name === 'PO-9001-spec.bin' && attachedMsg.attachment.size === DOC_BYTES.length && attachedMsg.attachment.sha256 === attachResult.sha256)

// ── exportTranscript(): verified via the real verify-transcript machinery ─
const bundle = await deskClient.request('exportTranscript', { roomKey: hubPo.key, exportedBy: 'desk (own copy, bridge export)' })
check('exportTranscript: format + roomKey', bundle.format === 'asymm-transcript.v1' && bundle.roomKey === hubPo.key)
const verdict = verifyTranscript(bundle)
check('exportTranscript: VERIFIED end-to-end through verify-transcript.mjs', verdict.verified === true && verdict.allSigsValid === true && verdict.digestMatches === true)

// ── room-updated events: self-observed AND replication-delivered ─────────
const hubSelfUpdate = waitForEvent(hubClient, 'room-updated')
await hubClient.request('post', { roomKey: hubPo.key, body: 'self-observed update check', expectation: '', ts: 1000 })
const selfEvt = await hubSelfUpdate
check('event: hub\'s OWN post emits room-updated on hub\'s own bridge', selfEvt.roomKey === hubPo.key)

const deskReplicatedUpdate = waitForEvent(deskClient, 'room-updated')
await hubClient.request('post', { roomKey: hubPo.key, body: 'replicated update check', expectation: '', ts: 1100 })
const replEvt = await deskReplicatedUpdate
check('event: hub\'s post, once replicated, emits room-updated on desk\'s bridge too', replEvt.roomKey === hubPo.key)

// ── GL-5: seq continues across a bridge restart, never resets to 1 ────────
const beforeRestartOps = await hubPo.ops()
const maxSeqBefore = Math.max(...beforeRestartOps.map((o) => o.seq)) // room-wide counter, not per-actor (see bridge-server.mjs)
await hubBridge.close()
hubBridge = await createBridgeServer({ rooms: new Map([[hubPo.key, hubPo]]), actor: 'hub', deviceKeys: HUB, storageDir: hubStoreDir, port: 0 })
hubClient = await createBridgeClient({ port: hubBridge.port })
const afterRestart = await hubClient.request('post', { roomKey: hubPo.key, body: 'after a bridge restart', expectation: '', ts: 1200 })
check('GL-5: seq continues after a bridge restart (never restarts at 1)', afterRestart.seq > maxSeqBefore)

// ── Social room: createSocialRoom + a REAL invite redeem under encryption ─
const socialCreated = await hubClient.request('createSocialRoom', { title: 'water cooler', encryptionKey: SOCIAL_KEY.toString('hex'), ts: 1300 })
const socialRoomKey = socialCreated.roomKey
check('createSocialRoom: room created, returns roomKey', typeof socialRoomKey === 'string' && socialRoomKey.length > 0)
const socialListed = await hubClient.request('listRooms')
check('listRooms: the social room reports kind "social" (no anchor)', socialListed.find((r) => r.roomKey === socialRoomKey)?.kind === 'social')

const dmInvite = await hubClient.request('openDmInvite', { roomKey: socialRoomKey, ts: 1400, inviteSeed: INVITE_SEED.toString('hex') })
check('openDmInvite: code is versioned asymm-room2 (encrypted room, key rides the invite)', dmInvite.inviteCode.startsWith('asymm-room2.'))

// Writer registration is a separate, pre-existing Autobase precondition
// (deviation #2, bridge-server.mjs header) — desk's own social-room node is
// opened and connected directly, exactly like every prior spike's setup
// phase, THEN addWriter'd by hub's raw node (reached via hubBridge.rooms).
const hubSocial = hubBridge.rooms.get(socialRoomKey)
const deskSocial = await createMeshNode({
  storage: join(tmp, 'desk-social'), primaryKey: PK(0x9d),
  bootstrap: hubSocial.key, authorityPub: hubSocial.authorityPub, mode: 'room',
  encryptionKey: hubSocial.encryptionKey,
})
deskBridge.registerRoom(deskSocial.key, deskSocial)
const socialWire = hubSocial.connect(deskSocial)
await hubSocial.addWriter(deskSocial.writerKey)
await waitFor(async () => { await deskSocial.base.update(); return deskSocial.writable }, { label: 'desk writable in social room' })

const redeemed = await deskClient.request('redeemInvite', { inviteCode: dmInvite.inviteCode, actor: 'desk', ts: 1500 })
check('redeemInvite: the REAL invite.redeem lands, room reachable by key', redeemed.roomKey === socialRoomKey)

await deskClient.request('post', { roomKey: socialRoomKey, body: 'joined via a real invite, over the wire', expectation: 'whenever', ts: 1600 })
await waitFor(async () => (await hubSocial.ops()).length >= 4, { label: 'hub sees desk\'s social-room post' }) // manifest, offer, redeem, post
const socialStateOnHub = await hubClient.request('roomState', { roomKey: socialRoomKey })
check('roomState: desk\'s message converged onto hub\'s bridge for the social room',
  socialStateOnHub.messages.some((m) => m.body === 'joined via a real invite, over the wire'))

// ── The claim-skip proof in a social room (Art. VI / MSG-D17, untouched) ──
const socialClaim = await deskClient.request('claimRoom', { roomKey: socialRoomKey, assignee: 'desk', ts: 1700 }).catch((e) => e)
check('claimRoom in a social room: the fold\'s "claims are a work concept" skip surfaces through the bridge verbatim',
  socialClaim instanceof Error && socialClaim.bridgeError === 'claims are a work concept')

console.log('\nAll scenarios exercised over the real protocol v0 wire.')

socialWire()
poWire()
await hubClient.close()
await deskClient.close()
await hubBridge.close()
await deskBridge.close()
await Promise.all([hubPo.close(), deskPo.close(), hubSocial.close(), deskSocial.close()])
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nBRIDGE SPIKE GREEN ✅' : `\nBRIDGE SPIKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
