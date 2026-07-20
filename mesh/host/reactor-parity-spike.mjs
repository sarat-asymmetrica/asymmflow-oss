// reactor-parity-spike.mjs — Bare-runtime campaign Phase 1b, owner ruling R1
// condition 3: "every existing golden vector folds byte-identical through the
// new channel — same ops in, byte-for-byte identical state out, versus the
// Node WASI host."
//
// SCOPE (stated up front, honestly): this spike drives the REDUCER BOUNDARY
// directly — the same (ops, config, mode) triple that mesh-node.mjs's
// state() hands to applyViaWasm() (mesh-node.mjs:160) — through BOTH the
// command module (apply.mjs) and the reactor (apply-reactor.mjs), and
// compares the RAW output bytes. It does NOT re-run the real spikes'
// Autobase/Hyperswarm/BlindPeer replication machinery (room-spike.mjs,
// mirror-spike.mjs, social-spike.mjs, etc. — those exercise P2P delivery,
// which is orthogonal to what changed in this phase). That scoping choice
// rests on one verified fact, not a shortcut: mesh/reducer/reducer.go:249
// (`sort.SliceStable(sorted, canonicalLess)`) sorts EVERY op by its own
// canonical (Seq, Actor, ...) order INSIDE the reducer before folding — so
// the reducer's output depends only on the SET of ops it receives, never on
// what order the host handed them in. Every existing spike script already
// relies on exactly this property (room-spike.mjs's own comment: "canonical
// order sorts primarily by Seq, actor only breaks ties"). Therefore the
// question "does the reactor fold this golden's ops byte-identically to the
// command module" can be answered by reconstructing each golden's exact op
// SET directly (reusing the same host helpers — capability.mjs,
// invite-code.mjs, reissue-room.mjs, social-room.mjs, attachments.mjs — the
// real spikes use to build those ops) and skipping the network/replication
// wiring entirely; a single local node's own `.ops()`/`.append()` already
// gives the exact op values without any peer ever being involved.
//
// Every op set below is transcribed from the corresponding *-spike.mjs /
// smoke.mjs / *-mesh.mjs file (op literals, pinned device seeds, and helper
// calls copied verbatim) — not re-derived or approximated. Where a golden
// pins a `stateDigest` (or plain `digest`), this spike ALSO checks the
// command module's own output against that pinned value, as an independent
// proof the reconstruction is the right op set — not just that two modules
// happen to agree with each other on arbitrary input.
//
// Run: node mesh/host/reactor-parity-spike.mjs   (after building BOTH
//   mesh/dist/reducer.wasm and mesh/dist/reducer-reactor.wasm)

import { WASI } from 'node:wasi'
import { readFileSync, writeFileSync, closeSync, openSync, rmSync, mkdtempSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'
import Corestore from 'corestore'

import { applyViaWasmRaw as reactorApplyRaw } from './apply-reactor.mjs'
import { createMeshNode } from './mesh-node.mjs'
import { deviceKeys, signOp, grantOp, epochOp, inviteKeys, inviteOfferOp, inviteRedeemOp } from './capability.mjs'
import { encodeInviteCode, decodeInviteCode } from './invite-code.mjs'
import { reissueRoom } from './reissue-room.mjs'
import { createSocialRoom, openDmInvite, blockDevice } from './social-room.mjs'
import { openBlobStore, putAttachment } from './attachments.mjs'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDENS_DIR = join(__dirname, '..', 'goldens')

// ── A RAW-bytes twin of apply.mjs's own command-module channel ─────────────
// apply.mjs (owner-ruling-frozen, not in this phase's file list) already
// does exactly this, but JSON.parses its result before returning — which
// would silently mask a key-ordering divergence, the exact thing this spike
// exists to catch ("compare bytes, not parsed objects"). This duplicates
// apply.mjs's own channel (temp-file stdin/stdout over node:wasi's command
// mode — see apply.mjs's header for why temp files, not pipes) verbatim,
// stopping one step earlier: Buffer out, not JSON.parse(Buffer out).
let _cmdModule = null
let _cmdCounter = 0
function commandApplyRaw(ops, config = undefined, mode = '') {
  if (!_cmdModule) {
    _cmdModule = new WebAssembly.Module(readFileSync(join(__dirname, '..', 'dist', 'reducer.wasm')))
  }
  const id = `parity-${process.pid}-${_cmdCounter++}`
  const inPath = join(tmpdir(), `mesh-reducer-in-${id}.json`)
  const outPath = join(tmpdir(), `mesh-reducer-out-${id}.json`)
  writeFileSync(inPath, JSON.stringify({ ...(mode ? { mode } : {}), ...(config ? { config } : {}), ops }))
  writeFileSync(outPath, '')
  const inFd = openSync(inPath, 'r')
  const outFd = openSync(outPath, 'w')
  try {
    const wasi = new WASI({ version: 'preview1', args: ['reducer'], env: {}, stdin: inFd, stdout: outFd, stderr: 2, returnOnExit: true })
    const instance = new WebAssembly.Instance(_cmdModule, wasi.getImportObject())
    const code = wasi.start(instance)
    if (code !== 0) throw new Error(`reducer.wasm exited with code ${code}`)
    return readFileSync(outPath) // Buffer, NOT JSON.parse'd
  } finally {
    closeSync(inFd); closeSync(outFd)
    try { rmSync(inPath) } catch {}
    try { rmSync(outPath) } catch {}
  }
}

let failures = 0
let scenarioCount = 0
function report(name, cmdBytes, reactorBytes, goldenCheck) {
  scenarioCount++
  const identical = Buffer.compare(cmdBytes, reactorBytes) === 0
  console.log(`\n── ${name} ──`)
  console.log(`   command bytes:  ${cmdBytes.length}B`)
  console.log(`   reactor bytes:  ${reactorBytes.length}B`)
  if (identical) {
    console.log(`   ✓ PASS — byte-identical`)
  } else {
    failures++
    console.log(`   ✗ FAIL — NOT byte-identical`)
    // Find and print the first differing byte/offset for diagnosis, honestly.
    const minLen = Math.min(cmdBytes.length, reactorBytes.length)
    let firstDiff = -1
    for (let i = 0; i < minLen; i++) { if (cmdBytes[i] !== reactorBytes[i]) { firstDiff = i; break } }
    if (firstDiff === -1) firstDiff = minLen
    const ctx = (buf, at) => buf.slice(Math.max(0, at - 30), at + 30).toString('utf8')
    console.log(`   first divergence at byte offset ${firstDiff}`)
    console.log(`     command  …${ctx(cmdBytes, firstDiff)}…`)
    console.log(`     reactor  …${ctx(reactorBytes, firstDiff)}…`)
  }
  if (goldenCheck) {
    const { ok, expected, got } = goldenCheck
    if (ok) {
      console.log(`   ✓ golden sanity: reconstructed op set matches the pinned golden (${expected.slice(0, 16)}…)`)
    } else {
      failures++
      console.log(`   ✗ golden sanity FAILED: expected ${expected?.slice(0, 16)}…, command module produced ${got?.slice(0, 16)}… — the reconstructed op set is WRONG, this scenario's PASS/FAIL above is not trustworthy`)
    }
  }
  return identical
}

function digestOf(buf) {
  try { return JSON.parse(buf.toString('utf8'))?.digest ?? null } catch { return null }
}

function runScenario(name, ops, config, mode, goldenPath, goldenDigestKey) {
  const cmdBytes = commandApplyRaw(ops, config, mode)
  const reactorBytes = reactorApplyRaw(ops, config, mode)
  let goldenCheck = null
  if (goldenPath && goldenDigestKey) {
    const golden = JSON.parse(readFileSync(goldenPath, 'utf8'))
    const expected = goldenDigestKey.split('.').reduce((o, k) => o?.[k], golden)
    const got = digestOf(cmdBytes)
    goldenCheck = { ok: expected === got, expected, got }
  }
  return report(name, cmdBytes, reactorBytes, goldenCheck)
}

console.log('reactor-parity-spike — command module vs go:wasmexport reactor, byte-for-byte, over every golden\n')
console.log('(scope note: reducer-boundary reconstruction, not a re-run of the P2P spikes — see file header)')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-parity-'))
const PK = (b) => Buffer.alloc(32, b)

let ok = true

// ── 1. inventory_basic (smoke.mjs) ──────────────────────────────────────────
{
  const OPS = [
    { seq: 1, actor: 'dev-a', sku: 'TX-100', delta: +10, ts: 100 },
    { seq: 2, actor: 'dev-a', sku: 'TX-100', delta: -6, ts: 200 },
    { seq: 1, actor: 'dev-b', sku: 'TX-100', delta: -6, ts: 150 },
    { seq: 1, actor: 'dev-a', sku: 'PH-200', delta: +3, ts: 120 },
    { seq: 2, actor: 'dev-b', sku: 'PH-200', delta: +4, ts: 220 },
  ]
  ok = runScenario('inventory_basic', OPS, undefined, '', join(GOLDENS_DIR, 'inventory_basic.json'), 'digest') && ok
}

// ── 2. missionc_autobase (missionc-mesh.mjs) ────────────────────────────────
{
  const PEER_A_OPS = [
    { seq: 1, actor: 'dev-a', ts: 100, sku: 'TX-100', delta: 10 },
    { seq: 2, actor: 'dev-a', ts: 200, sku: 'TX-100', delta: -6 },
    { seq: 3, actor: 'dev-a', ts: 300, kind: 'ar.limit', customer: 'CUST-01', limitMinor: 500000, currency: 'BHD' },
    { seq: 4, actor: 'dev-a', ts: 310, kind: 'ar.charge', customer: 'CUST-01', amountMinor: 400000, currency: 'BHD' },
    { seq: 6, actor: 'sarat', ts: 410, kind: 'approval.decide', subject: 'posting-77', subjectType: 'posting_draft', decision: 'approved', actorType: 'operator', authority: 2, correlationId: 'c-2' },
    { seq: 8, actor: 'sarat', ts: 520, kind: 'policy.override', policyId: 'VAT-DEADLINE', reason: 'filed via portal, receipt attached', actorType: 'operator', authority: 3 },
  ]
  const PEER_B_OPS = [
    { seq: 1, actor: 'dev-b', ts: 150, sku: 'TX-100', delta: -6 },
    { seq: 2, actor: 'dev-b', ts: 320, kind: 'ar.charge', customer: 'CUST-01', amountMinor: 200000, currency: 'BHD' },
    { seq: 3, actor: 'dev-b', ts: 330, kind: 'ar.payment', customer: 'CUST-01', amountMinor: 150000, currency: 'BHD' },
    { seq: 5, actor: 'dev-b', ts: 500, kind: 'policy.violation', policyId: 'VAT-DEADLINE' },
  ]
  const PEER_C_OPS = [
    { seq: 4, actor: 'butler-ai', ts: 400, kind: 'approval.decide', subject: 'posting-77', subjectType: 'posting_draft', decision: 'approved', actorType: 'agent', authority: 1, correlationId: 'c-1' },
    { seq: 6, actor: 'butler-ai', ts: 510, kind: 'policy.override', policyId: 'VAT-DEADLINE', reason: 'agent says it is fine', actorType: 'agent', authority: 1 },
  ]
  const ops = [...PEER_A_OPS, ...PEER_B_OPS, ...PEER_C_OPS]
  ok = runScenario('missionc_autobase', ops, undefined, '', join(GOLDENS_DIR, 'missionc_autobase.json'), 'stateDigest') && ok
}

// ── 3. missiond_autobase (missiond-mesh.mjs) ────────────────────────────────
{
  const AUTH = deviceKeys(PK(0xa1)), LAPTOP = deviceKeys(PK(0xb2)), ROGUE = deviceKeys(PK(0xc3))
  const A_OPS = [
    grantOp({ seq: 1, actor: 'sarat-hub', ts: 100, device: LAPTOP.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 3, actor: 'sarat-hub', ts: 300, sku: 'TX-100', delta: 5 }, AUTH),
    epochOp({ seq: 6, actor: 'sarat-hub', ts: 600, epoch: 1 }, AUTH),
    grantOp({ seq: 8, actor: 'sarat-hub', ts: 800, device: LAPTOP.pubHex, epoch: 1 }, AUTH),
  ]
  const B_OPS = [
    signOp({ seq: 2, actor: 'laptop', ts: 200, sku: 'TX-100', delta: 10 }, LAPTOP),
    signOp({ seq: 7, actor: 'laptop', ts: 700, sku: 'TX-100', delta: -4 }, LAPTOP),
    signOp({ seq: 9, actor: 'laptop', ts: 900, sku: 'TX-100', delta: -2 }, LAPTOP),
  ]
  const C_OPS = [
    signOp({ seq: 4, actor: 'rogue', ts: 400, sku: 'TX-100', delta: -3 }, ROGUE),
    grantOp({ seq: 5, actor: 'rogue', ts: 500, device: ROGUE.pubHex, epoch: 0 }, ROGUE),
  ]
  const ops = [...A_OPS, ...B_OPS, ...C_OPS]
  ok = runScenario('missiond_autobase', ops, { authorityPub: AUTH.pubHex }, '', join(GOLDENS_DIR, 'missiond_autobase.json'), 'stateDigest') && ok
}

// ── 4. room_autobase (room-spike.mjs) ───────────────────────────────────────
{
  const AUTH = deviceKeys(PK(0xd4)), DESK = deviceKeys(PK(0xe5)), PHONE = deviceKeys(PK(0xf6)), BUTLER = deviceKeys(PK(0x77))
  const ROGUE = deviceKeys(PK(0x99))
  const DRAFT = JSON.stringify({ kind: 'approval.decide', subject: 'posting:PO-2201', subjectType: 'posting_draft', decision: 'approved' })
  const A_OPS = [
    signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201', observersAllowed: true }, AUTH),
    signOp({ seq: 2, actor: 'hub', ts: 200, kind: 'cap.grant', device: DESK.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 3, actor: 'hub', ts: 300, kind: 'cap.grant', device: PHONE.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 4, actor: 'hub', ts: 400, kind: 'cap.grant', device: BUTLER.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 5, actor: 'hub', ts: 1800, kind: 'room.claim', assignee: 'phone' }, AUTH),
  ]
  const B_OPS = [
    signOp({ seq: 5, actor: 'desk', ts: 500, kind: 'msg.post', body: 'Can we ship the coils Thursday?', expectation: 'urgent' }, DESK),
    signOp({ seq: 8, actor: 'desk', ts: 800, kind: 'msg.react', msgId: 'phone:6', emoji: '👍', on: true }, DESK),
    signOp({ seq: 10, actor: 'butler', actorType: 'agent', ts: 1000, kind: 'msg.draft-op', body: 'Drafted the PO-2201 approval — needs a human decision', draft: DRAFT }, BUTLER),
    signOp({ seq: 11, actor: 'desk', ts: 1100, kind: 'msg.read', upToActor: 'phone', upToSeq: 7 }, DESK),
    signOp({ seq: 12, actor: 'desk', ts: 1200, kind: 'msg.post', body: 'typo — ignore this' }, DESK),
    signOp({ seq: 13, actor: 'desk', ts: 1300, kind: 'msg.delete', msgId: 'desk:12' }, DESK),
    signOp({ seq: 17, actor: 'desk', ts: 1700, kind: 'room.claim', assignee: 'desk' }, DESK),
  ]
  const C_OPS = [
    signOp({ seq: 6, actor: 'phone', ts: 600, kind: 'msg.post', body: 'Thursday morning works', replyTo: 'desk:5' }, PHONE),
    signOp({ seq: 7, actor: 'phone', ts: 700, kind: 'msg.edit', msgId: 'phone:6', body: 'Thursday afternoon works better' }, PHONE),
    signOp({ seq: 9, actor: 'phone', ts: 900, kind: 'msg.react', msgId: 'desk:5', emoji: '🔥', on: true }, PHONE),
    signOp({ seq: 14, actor: 'phone', ts: 1400, kind: 'msg.react', msgId: 'desk:5', emoji: '🔥', on: false }, PHONE),
    signOp({ seq: 15, actor: 'rogue', ts: 1500, kind: 'msg.post', body: 'let me into this deal' }, ROGUE),
    signOp({ seq: 16, actor: 'phone', ts: 1600, kind: 'msg.read', upToActor: 'desk', upToSeq: 5 }, PHONE),
  ]
  const ops = [...A_OPS, ...B_OPS, ...C_OPS]
  ok = runScenario('room_autobase', ops, { authorityPub: AUTH.pubHex }, 'room', join(GOLDENS_DIR, 'room_autobase.json'), 'stateDigest') && ok
}

// ── 5. invite_autobase (invite-spike.mjs) ───────────────────────────────────
{
  const AUTH = deviceKeys(PK(0xd4)), DESK = deviceKeys(PK(0xe5)), PHONE = deviceKeys(PK(0xf6))
  const SEED_W = PK(0x41), SEED_O = PK(0x42), SEED_X = PK(0x43)
  const INV_W = inviteKeys(SEED_W), INV_O = inviteKeys(SEED_O), INV_X = inviteKeys(SEED_X)
  const A_OPS = [
    signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201' }, AUTH),
    inviteOfferOp({ seq: 2, actor: 'hub', ts: 200, invitePub: INV_W.pubHex, expiresAt: 100000, maxUses: 1 }, AUTH),
    inviteOfferOp({ seq: 3, actor: 'hub', ts: 300, invitePub: INV_O.pubHex, role: 'observer', expiresAt: 0, maxUses: 2 }, AUTH),
    inviteOfferOp({ seq: 4, actor: 'hub', ts: 400, invitePub: INV_X.pubHex, expiresAt: 450, maxUses: 1 }, AUTH),
  ]
  // The pasteable codes need a base key — invite-spike.mjs uses the real
  // room's `a.key`; any deterministic 32-byte value serves identically here
  // since the reducer never inspects baseKey (host-only routing metadata).
  const roomBaseKey = 'a1'.repeat(32)
  const codeW = encodeInviteCode({ baseKey: roomBaseKey, authorityPub: AUTH.pubHex, inviteSeed: SEED_W, inviteId: 'hub:2' })
  const codeO = encodeInviteCode({ baseKey: roomBaseKey, authorityPub: AUTH.pubHex, inviteSeed: SEED_O, inviteId: 'hub:3' })
  const codeX = encodeInviteCode({ baseKey: roomBaseKey, authorityPub: AUTH.pubHex, inviteSeed: SEED_X, inviteId: 'hub:4' })
  const decW = decodeInviteCode(codeW), decO = decodeInviteCode(codeO), decX = decodeInviteCode(codeX)
  const redeemW = inviteRedeemOp({ seq: 5, actor: 'desk', ts: 500, inviteId: decW.inviteId }, inviteKeys(decW.inviteSeed), DESK)
  const B_OPS = [
    redeemW,
    signOp({ seq: 6, actor: 'desk', ts: 600, kind: 'msg.post', body: 'in via the code — sovereignty by paste' }, DESK),
  ]
  const C_OPS = [
    inviteRedeemOp({ seq: 7, actor: 'phone', ts: 700, inviteId: decW.inviteId }, inviteKeys(decW.inviteSeed), PHONE),
    inviteRedeemOp({ seq: 8, actor: 'phone', ts: 800, inviteId: decX.inviteId }, inviteKeys(decX.inviteSeed), PHONE),
    inviteRedeemOp({ seq: 9, actor: 'phone', ts: 900, inviteId: decO.inviteId }, inviteKeys(decO.inviteSeed), PHONE),
    signOp({ seq: 10, actor: 'phone', ts: 1000, kind: 'msg.post', body: 'observer speaking!' }, PHONE),
    signOp({ seq: 11, actor: 'phone', ts: 1100, kind: 'msg.read', upToActor: 'desk', upToSeq: 6 }, PHONE),
  ]
  const ops = [...A_OPS, ...B_OPS, ...C_OPS]
  ok = runScenario('invite_autobase', ops, { authorityPub: AUTH.pubHex }, 'room', join(GOLDENS_DIR, 'invite_autobase.json'), 'stateDigest') && ok
}

// ── 6. attach_autobase (attach-spike.mjs) ───────────────────────────────────
{
  const AUTH = deviceKeys(PK(0xd4)), DESK = deviceKeys(PK(0xe5)), PHONE = deviceKeys(PK(0xf6))
  const DOC_BYTES = Buffer.alloc(48 * 1024)
  for (let i = 0; i < DOC_BYTES.length; i++) DOC_BYTES[i] = (i * 31 + 7) & 0xff
  const VOICE_BYTES = Buffer.concat([
    Buffer.from([0x1a, 0x45, 0xdf, 0xa3]),
    Buffer.alloc(96 * 1024).map((_, i) => (Math.floor(127 + 96 * Math.sin(i / 16))) & 0xff),
  ])
  const deskStore = new Corestore(join(tmp, 'attach-desk'), { primaryKey: PK(0x2b), unsafe: true })
  const phoneStore = new Corestore(join(tmp, 'attach-phone'), { primaryKey: PK(0x3c), unsafe: true })
  await deskStore.ready(); await phoneStore.ready()
  const deskBlobs = await openBlobStore(deskStore)
  const phoneBlobs = await openBlobStore(phoneStore)
  const docRef = await putAttachment(deskBlobs, { name: 'PO-2201-spec-sheet.bin', contentType: 'application/octet-stream', bytes: DOC_BYTES })
  const voiceRef = await putAttachment(phoneBlobs, { name: 'site-update.webm', contentType: 'audio/webm', bytes: VOICE_BYTES })
  await deskStore.close(); await phoneStore.close()
  const ops = [
    signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201' }, AUTH),
    grantOp({ seq: 2, actor: 'hub', ts: 200, device: DESK.pubHex, epoch: 0 }, AUTH),
    grantOp({ seq: 3, actor: 'hub', ts: 300, device: PHONE.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 4, actor: 'desk', ts: 400, kind: 'msg.post', body: 'Spec sheet attached — please review before Thursday.', attachment: docRef }, DESK),
    signOp({ seq: 5, actor: 'phone', ts: 500, kind: 'msg.post', body: '', attachment: voiceRef }, PHONE),
  ]
  ok = runScenario('attach_autobase', ops, { authorityPub: AUTH.pubHex }, 'room', join(GOLDENS_DIR, 'attach_autobase.json'), 'stateDigest') && ok
}

// ── 7. mirror_autobase (mirror-spike.mjs) ───────────────────────────────────
{
  const AUTH = deviceKeys(PK(0xd4)), DESK = deviceKeys(PK(0xe5))
  const DESK_OPS = [
    signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201' }, AUTH),
    grantOp({ seq: 2, actor: 'hub', ts: 200, device: DESK.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 3, actor: 'desk', ts: 300, kind: 'msg.post', body: 'Shipment cleared customs at 16:40.' }, DESK),
    signOp({ seq: 4, actor: 'desk', ts: 400, kind: 'msg.post', body: 'Original BL is with the forwarder — collecting tomorrow.', replyTo: 'desk:3' }, DESK),
    signOp({ seq: 5, actor: 'desk', ts: 500, kind: 'msg.read', upToActor: 'desk', upToSeq: 4 }, DESK),
  ]
  ok = runScenario('mirror_autobase', DESK_OPS, { authorityPub: AUTH.pubHex }, 'room', join(GOLDENS_DIR, 'mirror_autobase.json'), 'stateDigest') && ok
}

// ── 8. reissue_autobase (reissue-spike.mjs) — two folds, epoch-1 + successor ─
{
  const AUTH = deviceKeys(PK(0xd4)), DESK = deviceKeys(PK(0xe5)), ROGUE = deviceKeys(PK(0x99))
  const K1 = PK(0x71), K2 = PK(0x72)
  const EPOCH1_OPS = [
    signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201' }, AUTH),
    grantOp({ seq: 2, actor: 'hub', ts: 200, device: DESK.pubHex, epoch: 0 }, AUTH),
    grantOp({ seq: 3, actor: 'hub', ts: 300, device: ROGUE.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 4, actor: 'desk', ts: 400, kind: 'msg.post', body: 'Booking the slot for PO-2201.' }, DESK),
    signOp({ seq: 5, actor: 'rogue', ts: 500, kind: 'msg.post', body: 'In on this one too — legit for now.' }, ROGUE),
  ]
  ok = runScenario('reissue_autobase (epoch1)', EPOCH1_OPS, { authorityPub: AUTH.pubHex }, 'room', join(GOLDENS_DIR, 'reissue_autobase.json'), 'epoch1StateDigest') && ok

  const epoch1 = await createMeshNode({ storage: join(tmp, 'reissue-epoch1'), primaryKey: PK(0x1a), authorityPub: AUTH.pubHex, mode: 'room', encryptionKey: K1 })
  for (const op of EPOCH1_OPS) await epoch1.append(op)
  const { node: successor } = await reissueRoom({
    predecessorNode: epoch1, storage: join(tmp, 'reissue-successor'), primaryKey: PK(0x2b),
    authorityKeys: AUTH, newEncryptionKey: K2, surviving: [DESK.pubHex], ts: 1000, actor: 'hub',
  })
  // desk's post-join message and rogue's forgery are self-contained signed
  // ops — appendable directly onto the successor node's own local core, with
  // no second node/replication needed (mesh-node.mjs's isOp()/apply() never
  // check WHICH writer core an op physically arrived on; only the fold's
  // capability check, downstream, does — and that's exactly what this
  // scenario is proving). VERIFIED against the real replicated flow
  // (deskNode + successor.connect(deskNode) + addWriter, per the actual
  // reissue-spike.mjs) via a throwaway debug diff during this spike's
  // development: byte-for-byte identical op arrays either way — see
  // PHASE1B_REPORT.md's reconciliation note.
  await successor.append(signOp({ seq: 3, actor: 'desk', ts: 1500, kind: 'msg.post', body: 'Confirmed — moved to the new room.' }, DESK))
  // The golden's successorStateDigest is pinned from `successorStateAfterDesk`
  // in reissue-spike.mjs — i.e. AFTER desk joins but BEFORE rogue's forgery
  // (reissue-spike.mjs's own `return` statement at the bottom of
  // runScenario() uses successorStateAfterDesk.digest, not
  // successorStateAfterForgery.digest, even though the forgery is appended
  // earlier in the script for its rejection-behavior assertions). This 3-op
  // snapshot is what must byte-match the golden.
  const successorOpsAtDesk = await successor.ops()
  ok = runScenario('reissue_autobase (successor, at desk join)', successorOpsAtDesk, { authorityPub: AUTH.pubHex }, 'room', join(GOLDENS_DIR, 'reissue_autobase.json'), 'successorStateDigest') && ok

  // The post-forgery state (4 ops) has no pinned golden digest of its own —
  // reissue-spike.mjs only asserts rejection behavior on it, never a digest —
  // so this scenario checks command-vs-reactor byte-identity only, no golden
  // sanity check possible or expected.
  await successor.append(signOp({ seq: 4, actor: 'rogue', ts: 1600, kind: 'msg.post', body: 'let me back in' }, ROGUE))
  const successorOpsAtForgery = await successor.ops()
  await epoch1.close(); await successor.close()
  ok = runScenario('reissue_autobase (successor, at forgery)', successorOpsAtForgery, { authorityPub: AUTH.pubHex }, 'room', null, null) && ok
}

// ── 9. social_autobase (social-spike.mjs) — pre-block + post-block folds ────
{
  const ANA = deviceKeys(PK(0xa1)), BELA = deviceKeys(PK(0xb2)), ROGUE = deviceKeys(PK(0x99))
  const ROOM_KEY_BYTES = PK(0x77), INVITE_SEED = PK(0x5d)
  const ana = await createSocialRoom({ creatorKeys: ANA, storage: join(tmp, 'social-ana'), primaryKey: PK(0x1a), title: 'chai break ☕', encryptionKey: ROOM_KEY_BYTES, ts: 100, actor: 'ana' })
  const dmCode = await openDmInvite(ana, { creatorKeys: ANA, inviteSeed: INVITE_SEED, ts: 200, seq: 2, actor: 'ana' })
  await ana.append(signOp({ seq: 3, actor: 'ana', ts: 300, kind: 'msg.post', body: 'you up? been a day 🍃', expectation: 'whenever' }, ANA))
  const decoded = decodeInviteCode(dmCode)
  await ana.append(inviteRedeemOp({ seq: 4, actor: 'bela', ts: 10_000, inviteId: decoded.inviteId }, inviteKeys(decoded.inviteSeed), BELA))
  await ana.append(inviteRedeemOp({ seq: 5, actor: 'rogue', ts: 10_100, inviteId: decoded.inviteId }, inviteKeys(decoded.inviteSeed), ROGUE))
  await ana.append(signOp({ seq: 6, actor: 'bela', ts: 10_200, kind: 'msg.post', body: 'up now. long story, got the offer today', replyTo: 'ana:3', expectation: 'whenever' }, BELA))
  await ana.append(signOp({ seq: 7, actor: 'bela', ts: 10_300, kind: 'msg.react', msgId: 'ana:3', emoji: '👋', on: true }, BELA))
  await ana.append(signOp({ seq: 8, actor: 'bela', ts: 10_400, kind: 'msg.read', upToActor: 'ana', upToSeq: 3 }, BELA))
  await ana.append(signOp({ seq: 9, actor: 'bela', ts: 10_500, kind: 'room.claim', assignee: 'bela' }, BELA))
  const opsAt9 = await ana.ops()
  ok = runScenario('social_autobase (pre-block)', opsAt9, { authorityPub: ANA.pubHex }, 'room', join(GOLDENS_DIR, 'social_autobase.json'), 'stateDigest') && ok

  await blockDevice(ana, { authorityKeys: ANA, devicePub: BELA.pubHex, ts: 20_000, seq: 10, actor: 'ana' })
  await ana.append(signOp({ seq: 11, actor: 'bela', ts: 20_100, kind: 'msg.post', body: 'hello? anyone?' }, BELA))
  const opsFinal = await ana.ops()
  await ana.close()
  const cmdFinalBytes = commandApplyRaw(opsFinal, { authorityPub: ANA.pubHex }, 'room')
  const reactorFinalBytes = reactorApplyRaw(opsFinal, { authorityPub: ANA.pubHex }, 'room')
  const finalIdentical = report('social_autobase (post-block)', cmdFinalBytes, reactorFinalBytes, null)
  const finalState = JSON.parse(cmdFinalBytes.toString('utf8'))
  const golden = JSON.parse(readFileSync(join(GOLDENS_DIR, 'social_autobase.json'), 'utf8'))
  const countsMatch = golden.opsHashed === finalState.opsHashed && golden.applied === finalState.applied &&
    golden.skipped === finalState.skipped.length && golden.rejected === finalState.rejected.length
  if (countsMatch) {
    console.log(`   ✓ golden sanity: post-block opsHashed/applied/skipped/rejected counts match the pinned golden`)
  } else {
    failures++
    console.log(`   ✗ golden sanity FAILED: post-block counts diverge from the pinned golden (got opsHashed=${finalState.opsHashed} applied=${finalState.applied} skipped=${finalState.skipped.length} rejected=${finalState.rejected.length}, golden wants opsHashed=${golden.opsHashed} applied=${golden.applied} skipped=${golden.skipped} rejected=${golden.rejected})`)
  }
  ok = finalIdentical && countsMatch && ok
}

// ── 10. transcript_autobase (transcript-spike.mjs, hub scenario only) ───────
{
  const AUTH = deviceKeys(PK(0xd4)), DESK = deviceKeys(PK(0xe5)), PHONE = deviceKeys(PK(0xf6)), GUEST = deviceKeys(PK(0x88))
  const OPS = [
    signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-9001 — Compressor Spares', anchorType: 'po', anchorId: 'PO-9001' }, AUTH),
    signOp({ seq: 2, actor: 'hub', ts: 200, kind: 'cap.grant', device: DESK.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 3, actor: 'hub', ts: 300, kind: 'cap.grant', device: PHONE.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 4, actor: 'hub', ts: 400, kind: 'cap.grant', device: GUEST.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 5, actor: 'desk', ts: 500, kind: 'msg.post', body: 'can you confirm the spares ship Tuesday' }, DESK),
    signOp({ seq: 6, actor: 'phone', ts: 600, kind: 'msg.post', body: 'yes, Tuesday morning', replyTo: 'desk:5' }, PHONE),
    signOp({ seq: 7, actor: 'desk', ts: 700, kind: 'msg.edit', msgId: 'desk:5', body: 'can you confirm the spares ship Tuesday AM' }, DESK),
    signOp({ seq: 8, actor: 'desk', ts: 800, kind: 'msg.post', body: 'ignore this, wrong thread' }, DESK),
    signOp({ seq: 9, actor: 'desk', ts: 900, kind: 'msg.delete', msgId: 'desk:8' }, DESK),
    signOp({ seq: 10, actor: 'phone', ts: 1000, kind: 'msg.react', msgId: 'desk:5', emoji: '👍', on: true }, PHONE),
    signOp({ seq: 11, actor: 'guest', ts: 1100, kind: 'msg.post', body: 'guest checking in before I leave the account' }, GUEST),
    epochOp({ seq: 12, actor: 'hub', ts: 1200, epoch: 1 }, AUTH),
    signOp({ seq: 13, actor: 'guest', ts: 1300, kind: 'msg.post', body: 'still trying to post after being let go' }, GUEST),
  ]
  ok = runScenario('transcript_autobase (hub)', OPS, { authorityPub: AUTH.pubHex }, 'room', join(GOLDENS_DIR, 'transcript_autobase.json'), 'stateDigest') && ok
}

try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(`\n${scenarioCount} scenario(s) run, ${failures} check failure(s) total.`)
console.log(failures === 0 ? '\nREACTOR PARITY GREEN ✅ — every golden folds byte-identically through the reactor' : '\nREACTOR PARITY RED ❌ — see failures above; owner ruling R1 is NOT satisfied')
process.exit(failures === 0 ? 0 : 1)
