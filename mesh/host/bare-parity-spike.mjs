// bare-parity-spike.mjs — Phase 1a gate (Bare-runtime campaign): does the
// UNMODIFIED mesh/dist/reducer.wasm fold byte-identically through the
// Bare-native WASI shim (wasi-preview1-lite.mjs + apply-bare.mjs) versus the
// real Node WASI host (node:wasi, apply.mjs's own channel), over EVERY
// golden vector in mesh/goldens/*.json?
//
// DUAL-RUNTIME, ASYMMETRIC VERIFICATION (stated up front, honestly):
// This file itself has no node:/bare-* imports at its own top level, so it
// loads under both `node host/bare-parity-spike.mjs` and
// `npx bare host/bare-parity-spike.mjs` — but the two runs do NOT prove the
// identical thing, because a live Node-vs-Bare comparison inside ONE process
// is impossible (that's the entire premise of this campaign: node:wasi does
// not exist under Bare, and mesh-node.mjs/capability.mjs/reissue-room.mjs/
// social-room.mjs/attachments.mjs — everything that CONSTRUCTS these
// scenarios' op logs — import node:crypto/node:wasi transitively, so even
// the *scenario-building* code cannot run under Bare, only the fold itself):
//
//   Under NODE: this spike builds every scenario's exact (ops, config, mode)
//   FRESH (dynamic-importing mesh-node.mjs/capability.mjs/invite-code.mjs/
//   reissue-room.mjs/social-room.mjs/attachments.mjs — all Node-only), runs
//   BOTH the real Node WASI host (a local raw-bytes twin of apply.mjs's own
//   channel — see nodeHostApplyRaw below, which duplicates apply.mjs's logic
//   read-only, never imports or edits it) and the Bare shim host
//   (apply-bare.mjs) in the SAME process, and diffs the RAW bytes directly.
//   This is the real, live, byte-for-byte proof. It also refreshes
//   bare-parity-fixtures.json (ops/config/mode + the Node host's raw output,
//   base64) so the Bare-only run below has something to check against.
//
//   Under BARE: scenario construction is unavailable (see above), so this
//   run LOADS bare-parity-fixtures.json (written by the most recent Node
//   run) and compares the Bare shim's raw output against the PRE-RECORDED
//   Node-host bytes pinned in that file. This is still a genuine
//   byte-identity check of the shim under the real Bare runtime — it is
//   just not a live dual-host run in the same process, and this file says so
//   instead of hiding it.
//
// Every scenario's op log is transcribed from the same source spike files
// used across this campaign (smoke.mjs, missionc-mesh.mjs, missiond-mesh.mjs,
// room-spike.mjs, invite-spike.mjs, attach-spike.mjs, mirror-spike.mjs,
// reissue-spike.mjs, social-spike.mjs, transcript-spike.mjs) — the exact same
// reconstruction approach (and, for 8/10 scenarios, the exact same literals)
// already used by host/reactor-parity-spike.mjs (Phase 1b, a sibling file
// this spike does NOT import from or modify — campaign brief: that file and
// mesh/host/apply-reactor.mjs belong to another coder). Reusing an
// independently-already-reconstructed, in-repo op set is a legitimate
// shortcut, not a shortcut on the ABI/shim work this spike actually gates —
// what's under test here is whether the Bare host reproduces the SAME bytes
// as the Node host for a given op set, not whether the op set itself is
// novel.
//
// Run: node host/bare-parity-spike.mjs        (builds fixtures + live check)
//      npx bare host/bare-parity-spike.mjs     (checks the shim against them)
//      (from OUTSIDE the repo tree too, per D5 — see PHASE1A_REPORT.md)

const isBare = typeof Bare !== 'undefined'
const fsMod = isBare ? await import('bare-fs') : await import('node:fs')

const FIXTURES_URL = new URL('./bare-parity-fixtures.json', import.meta.url)
const GOLDENS_DIR_URL = new URL('../goldens/', import.meta.url)

import { applyViaWasmRaw as bareApplyRaw } from './apply-bare.mjs'

let failures = 0
let scenarioCount = 0

function report(name, expectedBytes, gotBytes) {
  scenarioCount++
  const identical = Buffer.compare(expectedBytes, gotBytes) === 0
  console.log(`\n-- ${name} --`)
  console.log(`   expected (Node host): ${expectedBytes.length}B`)
  console.log(`   got (Bare shim):      ${gotBytes.length}B`)
  if (identical) {
    console.log('   PASS - byte-identical')
  } else {
    failures++
    console.log('   FAIL - NOT byte-identical')
    const minLen = Math.min(expectedBytes.length, gotBytes.length)
    let firstDiff = -1
    for (let i = 0; i < minLen; i++) { if (expectedBytes[i] !== gotBytes[i]) { firstDiff = i; break } }
    if (firstDiff === -1) firstDiff = minLen
    const ctx = (b, at) => b.subarray(Math.max(0, at - 30), at + 30).toString('utf8')
    console.log(`   first divergence at byte offset ${firstDiff}`)
    console.log(`     expected ...${ctx(expectedBytes, firstDiff)}...`)
    console.log(`     got      ...${ctx(gotBytes, firstDiff)}...`)
  }
  return identical
}

function digestOf(bytes) {
  try { return JSON.parse(bytes.toString('utf8'))?.digest ?? null } catch { return null }
}

// ── Node-only: a raw-bytes twin of apply.mjs's own channel ────────────────
// apply.mjs (frozen, not touched or imported here) JSON.parses its result
// before returning, which would mask a key-ordering divergence — exactly
// what this spike exists to catch. This reads mesh/dist/reducer.wasm and
// drives it via node:wasi + temp-file fds, verbatim to apply.mjs's own
// design (see that file's header for why temp files, not pipes), stopping
// one step earlier: Buffer out, not JSON.parse(Buffer out).
async function nodeHostApplyRaw(ops, config, mode) {
  const { WASI } = await import('node:wasi')
  const { readFileSync, writeFileSync, closeSync, openSync, rmSync } = await import('node:fs')
  const { tmpdir } = await import('node:os')
  const wasmPath = new URL('../dist/reducer.wasm', import.meta.url)
  if (!nodeHostApplyRaw._mod) {
    nodeHostApplyRaw._mod = new WebAssembly.Module(readFileSync(wasmPath))
  }
  const id = `bareparity-${process.pid}-${(nodeHostApplyRaw._n = (nodeHostApplyRaw._n ?? 0) + 1)}`
  const inFile = `${tmpdir()}/mesh-bareparity-in-${id}.json`
  const outFile = `${tmpdir()}/mesh-bareparity-out-${id}.json`
  writeFileSync(inFile, JSON.stringify({ ...(mode ? { mode } : {}), ...(config ? { config } : {}), ops }))
  writeFileSync(outFile, '')
  const inFd = openSync(inFile, 'r')
  const outFd = openSync(outFile, 'w')
  try {
    const wasi = new WASI({ version: 'preview1', args: ['reducer'], env: {}, stdin: inFd, stdout: outFd, stderr: 2, returnOnExit: true })
    const instance = new WebAssembly.Instance(nodeHostApplyRaw._mod, wasi.getImportObject())
    const code = wasi.start(instance)
    if (code !== 0) throw new Error(`reducer.wasm exited with code ${code}`)
    return readFileSync(outFile)
  } finally {
    closeSync(inFd); closeSync(outFd)
    try { rmSync(inFile) } catch {}
    try { rmSync(outFile) } catch {}
  }
}

// ── Node-only: build every scenario's exact (ops, config, mode) ───────────
async function buildScenarios() {
  const { createMeshNode } = await import('./mesh-node.mjs')
  const { deviceKeys, signOp, grantOp, epochOp, inviteKeys, inviteOfferOp, inviteRedeemOp } = await import('./capability.mjs')
  const { encodeInviteCode, decodeInviteCode } = await import('./invite-code.mjs')
  const { reissueRoom } = await import('./reissue-room.mjs')
  const { createSocialRoom, openDmInvite, blockDevice } = await import('./social-room.mjs')
  const { openBlobStore, putAttachment } = await import('./attachments.mjs')
  const { default: Corestore } = await import('corestore')
  const { mkdtempSync, rmSync } = await import('node:fs')
  const { tmpdir } = await import('node:os')
  const { join } = await import('node:path')

  const PK = (b) => Buffer.alloc(32, b)
  const tmp = mkdtempSync(join(tmpdir(), 'mesh-bareparity-'))
  const scenarios = {}
  const add = (name, ops, config, mode) => { scenarios[name] = { ops, config, mode: mode ?? '' } }

  // 1. inventory_basic (smoke.mjs)
  add('inventory_basic', [
    { seq: 1, actor: 'dev-a', sku: 'TX-100', delta: +10, ts: 100 },
    { seq: 2, actor: 'dev-a', sku: 'TX-100', delta: -6, ts: 200 },
    { seq: 1, actor: 'dev-b', sku: 'TX-100', delta: -6, ts: 150 },
    { seq: 1, actor: 'dev-a', sku: 'PH-200', delta: +3, ts: 120 },
    { seq: 2, actor: 'dev-b', sku: 'PH-200', delta: +4, ts: 220 },
  ], undefined, '')

  // 2. missionc_autobase (missionc-mesh.mjs)
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
    add('missionc_autobase', [...PEER_A_OPS, ...PEER_B_OPS, ...PEER_C_OPS], undefined, '')
  }

  // 3. missiond_autobase (missiond-mesh.mjs)
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
    add('missiond_autobase', [...A_OPS, ...B_OPS, ...C_OPS], { authorityPub: AUTH.pubHex }, '')
  }

  // 4. room_autobase (room-spike.mjs)
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
      signOp({ seq: 8, actor: 'desk', ts: 800, kind: 'msg.react', msgId: 'phone:6', emoji: '\u{1F44D}', on: true }, DESK),
      signOp({ seq: 10, actor: 'butler', actorType: 'agent', ts: 1000, kind: 'msg.draft-op', body: 'Drafted the PO-2201 approval — needs a human decision', draft: DRAFT }, BUTLER),
      signOp({ seq: 11, actor: 'desk', ts: 1100, kind: 'msg.read', upToActor: 'phone', upToSeq: 7 }, DESK),
      signOp({ seq: 12, actor: 'desk', ts: 1200, kind: 'msg.post', body: 'typo — ignore this' }, DESK),
      signOp({ seq: 13, actor: 'desk', ts: 1300, kind: 'msg.delete', msgId: 'desk:12' }, DESK),
      signOp({ seq: 17, actor: 'desk', ts: 1700, kind: 'room.claim', assignee: 'desk' }, DESK),
    ]
    const C_OPS = [
      signOp({ seq: 6, actor: 'phone', ts: 600, kind: 'msg.post', body: 'Thursday morning works', replyTo: 'desk:5' }, PHONE),
      signOp({ seq: 7, actor: 'phone', ts: 700, kind: 'msg.edit', msgId: 'phone:6', body: 'Thursday afternoon works better' }, PHONE),
      signOp({ seq: 9, actor: 'phone', ts: 900, kind: 'msg.react', msgId: 'desk:5', emoji: '\u{1F525}', on: true }, PHONE),
      signOp({ seq: 14, actor: 'phone', ts: 1400, kind: 'msg.react', msgId: 'desk:5', emoji: '\u{1F525}', on: false }, PHONE),
      signOp({ seq: 15, actor: 'rogue', ts: 1500, kind: 'msg.post', body: 'let me into this deal' }, ROGUE),
      signOp({ seq: 16, actor: 'phone', ts: 1600, kind: 'msg.read', upToActor: 'desk', upToSeq: 5 }, PHONE),
    ]
    add('room_autobase', [...A_OPS, ...B_OPS, ...C_OPS], { authorityPub: AUTH.pubHex }, 'room')
  }

  // 5. invite_autobase (invite-spike.mjs)
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
    add('invite_autobase', [...A_OPS, ...B_OPS, ...C_OPS], { authorityPub: AUTH.pubHex }, 'room')
  }

  // 6. attach_autobase (attach-spike.mjs)
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
    add('attach_autobase', ops, { authorityPub: AUTH.pubHex }, 'room')
  }

  // 7. mirror_autobase (mirror-spike.mjs)
  {
    const AUTH = deviceKeys(PK(0xd4)), DESK = deviceKeys(PK(0xe5))
    const DESK_OPS = [
      signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201' }, AUTH),
      grantOp({ seq: 2, actor: 'hub', ts: 200, device: DESK.pubHex, epoch: 0 }, AUTH),
      signOp({ seq: 3, actor: 'desk', ts: 300, kind: 'msg.post', body: 'Shipment cleared customs at 16:40.' }, DESK),
      signOp({ seq: 4, actor: 'desk', ts: 400, kind: 'msg.post', body: 'Original BL is with the forwarder — collecting tomorrow.', replyTo: 'desk:3' }, DESK),
      signOp({ seq: 5, actor: 'desk', ts: 500, kind: 'msg.read', upToActor: 'desk', upToSeq: 4 }, DESK),
    ]
    add('mirror_autobase', DESK_OPS, { authorityPub: AUTH.pubHex }, 'room')
  }

  // 8. reissue_autobase (reissue-spike.mjs) -- two folds: epoch1, successor
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
    add('reissue_autobase_epoch1', EPOCH1_OPS, { authorityPub: AUTH.pubHex }, 'room')

    const epoch1 = await createMeshNode({ storage: join(tmp, 'reissue-epoch1'), primaryKey: PK(0x1a), authorityPub: AUTH.pubHex, mode: 'room', encryptionKey: K1 })
    for (const op of EPOCH1_OPS) await epoch1.append(op)
    const { node: successor } = await reissueRoom({
      predecessorNode: epoch1, storage: join(tmp, 'reissue-successor'), primaryKey: PK(0x2b),
      authorityKeys: AUTH, newEncryptionKey: K2, surviving: [DESK.pubHex], ts: 1000, actor: 'hub',
    })
    await successor.append(signOp({ seq: 3, actor: 'desk', ts: 1500, kind: 'msg.post', body: 'Confirmed — moved to the new room.' }, DESK))
    // reissue-spike.mjs captures its `successorStateAfterDesk` (the state that
    // becomes the pinned successorStateDigest golden) BEFORE the rogue-forgery
    // op below is appended — that op exists in the source spike purely to
    // prove the capability plane rejects it, not to be part of the golden
    // snapshot. Matching that exact capture point here.
    const successorOpsPreForgery = await successor.ops()
    add('reissue_autobase_successor', successorOpsPreForgery, { authorityPub: AUTH.pubHex }, 'room')
    await successor.append(signOp({ seq: 4, actor: 'rogue', ts: 1600, kind: 'msg.post', body: 'let me back in' }, ROGUE))
    const successorOpsWithForgery = await successor.ops()
    await epoch1.close(); await successor.close()
    // Extra byte-identity coverage (not tied to a golden field): the forgery
    // exercises the reducer's capability-rejection path.
    add('reissue_autobase_successor_with_forgery', successorOpsWithForgery, { authorityPub: AUTH.pubHex }, 'room')
  }

  // 9. social_autobase (social-spike.mjs) -- pre-block + post-block folds
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
    await ana.append(signOp({ seq: 7, actor: 'bela', ts: 10_300, kind: 'msg.react', msgId: 'ana:3', emoji: '\u{1F44B}', on: true }, BELA))
    await ana.append(signOp({ seq: 8, actor: 'bela', ts: 10_400, kind: 'msg.read', upToActor: 'ana', upToSeq: 3 }, BELA))
    await ana.append(signOp({ seq: 9, actor: 'bela', ts: 10_500, kind: 'room.claim', assignee: 'bela' }, BELA))
    const opsAt9 = await ana.ops()
    add('social_autobase_preblock', opsAt9, { authorityPub: ANA.pubHex }, 'room')

    await blockDevice(ana, { authorityKeys: ANA, devicePub: BELA.pubHex, ts: 20_000, seq: 10, actor: 'ana' })
    await ana.append(signOp({ seq: 11, actor: 'bela', ts: 20_100, kind: 'msg.post', body: 'hello? anyone?' }, BELA))
    const opsFinal = await ana.ops()
    await ana.close()
    add('social_autobase_postblock', opsFinal, { authorityPub: ANA.pubHex }, 'room')
  }

  // 10. transcript_autobase (transcript-spike.mjs, hub scenario)
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
      signOp({ seq: 10, actor: 'phone', ts: 1000, kind: 'msg.react', msgId: 'desk:5', emoji: '\u{1F44D}', on: true }, PHONE),
      signOp({ seq: 11, actor: 'guest', ts: 1100, kind: 'msg.post', body: 'guest checking in before I leave the account' }, GUEST),
      epochOp({ seq: 12, actor: 'hub', ts: 1200, epoch: 1 }, AUTH),
      signOp({ seq: 13, actor: 'guest', ts: 1300, kind: 'msg.post', body: 'still trying to post after being let go' }, GUEST),
    ]
    add('transcript_autobase', OPS, { authorityPub: AUTH.pubHex }, 'room')
  }

  try { rmSync(tmp, { recursive: true, force: true }) } catch {}
  return scenarios
}

// name -> [golden filename (no .json), digest field] — every scenario this
// spike runs has a home golden to cross-check against, an independent proof
// that the reconstructed op set is the RIGHT one, not just that the two
// hosts happen to agree with each other on arbitrary input.
const GOLDEN_OF = {
  inventory_basic: ['inventory_basic', 'digest'],
  missionc_autobase: ['missionc_autobase', 'stateDigest'],
  missiond_autobase: ['missiond_autobase', 'stateDigest'],
  room_autobase: ['room_autobase', 'stateDigest'],
  invite_autobase: ['invite_autobase', 'stateDigest'],
  attach_autobase: ['attach_autobase', 'stateDigest'],
  mirror_autobase: ['mirror_autobase', 'stateDigest'],
  reissue_autobase_epoch1: ['reissue_autobase', 'epoch1StateDigest'],
  reissue_autobase_successor: ['reissue_autobase', 'successorStateDigest'],
  social_autobase_preblock: ['social_autobase', 'stateDigest'],
  transcript_autobase: ['transcript_autobase', 'stateDigest'],
  // social_autobase_postblock and reissue_autobase's own top-level fields
  // (opsHashed/applied/skipped/rejected counts, not a single digest) aren't
  // in this map — the byte-identity check above already covers them; this
  // map exists for the digest-based sanity cross-check only.
}

function checkGolden(name, bytes) {
  const entry = GOLDEN_OF[name]
  if (!entry) return
  const [file, key] = entry
  let golden
  try { golden = JSON.parse(fsMod.readFileSync(new URL(`${file}.json`, GOLDENS_DIR_URL), 'utf8')) } catch { console.log(`   (golden check skipped for ${name}: no goldens/${file}.json)`); return }
  const expected = golden[key]
  if (!expected) { console.log(`   (golden check skipped for ${name}: goldens/${file}.json has no ${key})`); return }
  const got = digestOf(bytes)
  if (got === expected) {
    console.log(`   golden check: matches goldens/${file}.json's ${key} (${expected.slice(0, 16)}...)`)
  } else {
    failures++
    console.log(`   golden check FAILED: got ${got?.slice(0, 16)}..., goldens/${file}.json wants ${expected?.slice(0, 16)}... -- the reconstructed op set is WRONG, the byte-identity result above is not trustworthy`)
  }
}

console.log(`bare-parity-spike -- Node WASI host vs Bare shim host, byte-for-byte, every golden [runtime: ${isBare ? 'Bare' : 'Node'}]\n`)

if (!isBare) {
  const scenarios = await buildScenarios()
  const fixtures = {}
  for (const [name, { ops, config, mode }] of Object.entries(scenarios)) {
    const nodeBytes = await nodeHostApplyRaw(ops, config, mode)
    const bareBytes = bareApplyRaw(ops, config, mode)
    report(name, nodeBytes, bareBytes)
    checkGolden(name, nodeBytes) // sanity-check the REFERENCE bytes, not the shim's — this is "is the op set right", independent of the shim entirely
    fixtures[name] = { ops, config: config ?? null, mode, expectedRawBase64: nodeBytes.toString('base64') }
  }
  fsMod.writeFileSync(FIXTURES_URL, JSON.stringify(fixtures, null, 2) + '\n')
  console.log(`\n(fixtures written for the Bare-only run: ${FIXTURES_URL.pathname})`)
} else {
  if (!fsMod.existsSync(FIXTURES_URL)) {
    console.log('FIXTURES MISSING -- run `node host/bare-parity-spike.mjs` first to generate bare-parity-fixtures.json')
    Bare.exit(1)
  }
  const fixtures = JSON.parse(fsMod.readFileSync(FIXTURES_URL, 'utf8'))
  for (const [name, { ops, config, mode, expectedRawBase64 }] of Object.entries(fixtures)) {
    const expected = Buffer.from(expectedRawBase64, 'base64')
    const bareBytes = bareApplyRaw(ops, config ?? undefined, mode)
    report(name, expected, bareBytes)
    checkGolden(name, bareBytes) // here it IS the shim's own bytes -- no live Node host to fall back on under Bare
  }
}

console.log(`\n${scenarioCount} scenario(s) run, ${failures} failure(s) total.`)
console.log(failures === 0
  ? '\nBARE PARITY SPIKE GREEN -- the unmodified reducer folds byte-identically under Bare'
  : `\nBARE PARITY SPIKE RED (${failures} failure(s)) -- see above`)

if (isBare) Bare.exit(failures === 0 ? 0 : 1)
else process.exit(failures === 0 ? 0 : 1)
