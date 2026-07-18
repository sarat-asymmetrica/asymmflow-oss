// attach-spike.mjs — Mission M3 gate: files + voice notes, content-addressed.
//
// "A message references a blob by key+hash; the fold never inlines file
// bytes into the room log" (campaign invariant 4). Three real peers:
//   desk  — attaches the PO-2201 spec sheet (a document)
//   phone — attaches a voice note (audio/webm bytes — same pipeline, zero
//           live-media stack; capture is a UI concern, not a mesh one)
//   hub   — sent nothing: it must still stream BOTH blobs P2P and verify
//           them end-to-end against the refs the room log carries
// Tamper checks close the gate: a flipped byte and a forged ref must both
// die loudly. Plus the REFOLD BENCH (owner reactor pre-authorization: build
// only on measured need) — 10k-op room refold timed through the real wasm.
//
// Run: npm run attachspike
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { deviceKeys, signOp, grantOp } from './capability.mjs'
import { openBlobStore, putAttachment, getAttachment, verifyAttachmentBytes } from './attachments.mjs'
import { applyViaWasm } from './apply.mjs'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDEN = join(__dirname, '..', 'goldens', 'attach_autobase.json')
const UPDATE_GOLDEN = process.argv.includes('--update-golden')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

const PK = (b) => Buffer.alloc(32, b)
const AUTH = deviceKeys(Buffer.alloc(32, 0xd4))
const DESK = deviceKeys(Buffer.alloc(32, 0xe5))
const PHONE = deviceKeys(Buffer.alloc(32, 0xf6))

// Deterministic fixtures (synthetic canon — no real documents, no real audio).
const DOC_BYTES = Buffer.alloc(48 * 1024)
for (let i = 0; i < DOC_BYTES.length; i++) DOC_BYTES[i] = (i * 31 + 7) & 0xff
const VOICE_BYTES = Buffer.concat([
  Buffer.from([0x1a, 0x45, 0xdf, 0xa3]), // EBML magic — "this is webm-shaped"
  Buffer.alloc(96 * 1024).map((_, i) => (Math.floor(127 + 96 * Math.sin(i / 16))) & 0xff),
])

console.log('Messenger M3 — attachment gate: bytes travel P2P, the ref is the promise\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-attach-'))
const mk = (dir, b, extra = {}) =>
  createMeshNode({ storage: join(tmp, dir), primaryKey: PK(b), authorityPub: AUTH.pubHex, mode: 'room', ...extra })
const a = await mk('a', 0x1a)
const b = await mk('b', 0x2b, { bootstrap: a.key })
const c = await mk('c', 0x3c, { bootstrap: a.key })

let wires = [a.connect(b), a.connect(c), b.connect(c)]
const disconnect = () => { for (const cut of wires) cut(); wires = [] }
const reconnect = () => { wires = [a.connect(b), a.connect(c), b.connect(c)] }

await a.addWriter(b.writerKey)
await a.addWriter(c.writerKey)
await waitFor(async () => { await b.base.update(); return b.writable }, { label: 'B writable' })
await waitFor(async () => { await c.base.update(); return c.writable }, { label: 'C writable' })

// Room opens; grants issued.
for (const op of [
  signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201' }, AUTH),
  grantOp({ seq: 2, actor: 'hub', ts: 200, device: DESK.pubHex, epoch: 0 }, AUTH),
  grantOp({ seq: 3, actor: 'hub', ts: 300, device: PHONE.pubHex, epoch: 0 }, AUTH),
]) await a.append(op)

// OFFLINE: each side writes its blob locally and posts the REF, not the bytes.
disconnect()
const deskBlobs = await openBlobStore(b.store)
const docRef = await putAttachment(deskBlobs, { name: 'PO-2201-spec-sheet.bin', contentType: 'application/octet-stream', bytes: DOC_BYTES })
await b.append(signOp({ seq: 4, actor: 'desk', ts: 400, kind: 'msg.post', body: 'Spec sheet attached — please review before Thursday.', attachment: docRef }, DESK))

const phoneBlobs = await openBlobStore(c.store)
const voiceRef = await putAttachment(phoneBlobs, { name: 'site-update.webm', contentType: 'audio/webm', bytes: VOICE_BYTES })
await c.append(signOp({ seq: 5, actor: 'phone', ts: 500, kind: 'msg.post', body: '', attachment: voiceRef }, PHONE))
reconnect()

const TOTAL = 5
await waitFor(async () => {
  const [va, vb, vc] = await Promise.all([a.ops(), b.ops(), c.ops()])
  return va.length === TOTAL && vb.length === TOTAL && vc.length === TOTAL
}, { label: `all peers to see ${TOTAL} linearized ops`, timeout: 30000 })

const [da, db, dc] = await Promise.all([a.viewDigest(), b.viewDigest(), c.viewDigest()])
check('convergence: 3 peers, views byte-identical', da === db && db === dc)
const [sa, sb, sc] = await Promise.all([a.state(), b.state(), c.state()])
check('room state digests: byte-identical on all peers', sa.digest === sb.digest && sb.digest === sc.digest)

// ── The refs live in the log; the bytes do not ──
const docMsg = sa.messages.find((m) => m.msgId === 'desk:4')
const voiceMsg = sa.messages.find((m) => m.msgId === 'phone:5')
check('refs: both messages carry their attachment refs verbatim (inert cargo)',
  docMsg?.attachment === docRef && voiceMsg?.attachment === voiceRef)
const viewBytes = Buffer.byteLength(JSON.stringify(await a.ops()))
check(`log stays lean: ${viewBytes}B of log vs ${DOC_BYTES.length + VOICE_BYTES.length}B of blobs (never inlined)`,
  viewBytes < 8 * 1024)
check('voice note is just an attachment: audio/webm contentType in the ref',
  JSON.parse(voiceMsg.attachment).contentType === 'audio/webm')

// ── Every peer streams BOTH blobs and verifies end-to-end ──
const got = { doc: [], voice: [] }
for (const [label, node] of [['hub', a], ['desk', b], ['phone', c]]) {
  const doc = await getAttachment(node.store, docMsg.attachment)
  const voice = await getAttachment(node.store, voiceMsg.attachment)
  got.doc.push([label, doc.bytes])
  got.voice.push([label, voice.bytes])
}
check('P2P retrieval: all 3 peers hold byte-identical documents (sha-verified)',
  got.doc.every(([, bytes]) => bytes.equals(DOC_BYTES)))
check('P2P retrieval: all 3 peers hold the byte-identical voice note',
  got.voice.every(([, bytes]) => bytes.equals(VOICE_BYTES)))

// ── Tampering dies loudly ──
const flipped = Buffer.from(DOC_BYTES); flipped[1234] ^= 0xff
check('tamper: a single flipped byte fails ref verification',
  verifyAttachmentBytes(docRef, flipped) === 'sha256 mismatch')
const forged = JSON.stringify({ ...JSON.parse(docRef), sha256: '0'.repeat(64) })
let forgedDied = false
try { await getAttachment(a.store, forged) } catch (e) { forgedDied = /sha256 mismatch/.test(e.message) }
check('tamper: a forged ref (wrong content address) throws on retrieval', forgedDied)

// ── REFOLD BENCH (reactor decision: measure, don't assume) ──
const benchOps = []
for (let i = 1; i <= 10_000; i++) {
  benchOps.push({ seq: i, actor: `w${i % 5}`, ts: i * 10, kind: 'msg.post', body: `message number ${i} with a plausible sentence of chat traffic` })
}
for (const n of [1_000, 5_000, 10_000]) {
  const slice = benchOps.slice(0, n)
  const t0 = process.hrtime.bigint()
  const st = applyViaWasm(slice, undefined, 'room')
  const ms = Number(process.hrtime.bigint() - t0) / 1e6
  console.log(`  ⏱ refold ${n} ops through the wasm boundary: ${ms.toFixed(1)}ms (applied ${st.applied})`)
}

// GATE FIX (GL-2): this scenario forks desk and phone CONCURRENTLY (both
// append while disconnected), so Autobase may legitimately linearize the two
// heads in either order — the VIEW digest is not run-deterministic here and
// pinning it made this gate flaky. The STATE digest is order-independent by
// construction (the canonical fold is the whole point), so the golden pins
// state only; the view is asserted as CONVERGED (byte-identical across
// peers) above, which is the actual guarantee. A view golden is only valid
// in a spike whose appends are causally chained (single writer or barriered).
if (UPDATE_GOLDEN || !existsSync(GOLDEN)) {
  writeFileSync(GOLDEN, JSON.stringify({ viewLength: TOTAL, stateDigest: sa.digest, state: sa }, null, 2) + '\n')
  console.log(`\n  (golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(GOLDEN, 'utf8'))
  check('golden: room state digest matches pinned golden', golden.stateDigest === sa.digest)
  check('golden: state projection matches pinned golden (deep)', JSON.stringify(golden.state) === JSON.stringify(sa))
}

console.log(`\nview digest:  ${da}`)
console.log(`state digest: ${sa.digest}`)

disconnect()
await Promise.all([a.close(), b.close(), c.close()])
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nATTACH SPIKE GREEN ✅' : `\nATTACH SPIKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
