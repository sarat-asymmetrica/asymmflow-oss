// invite-spike.mjs — Mission M2 gate: "click a code, you're in the PO room."
//
// Invites are signed grant OFFERS enforced by the FOLD — expiry (op-data time,
// never a clock), use-count, revocability, proof-of-possession bound to the
// joining device — everything upstream blind-pairing leaves advisory. The
// capability plane stays Mission D's; an invite is just a lawful way IN.
//
// Three real peers, one genuine offline fork:
//   peer A ("hub")   — room authority: manifest + THREE invite offers
//                      (one-time writer / multi-use observer / short-lived)
//   peer B ("desk")  — joins via the ONE-TIME code (full round-trip through
//                      the pasteable asymm-room1.… string), then speaks
//   peer C ("phone") — tries the exhausted code (dead), the expired code
//                      (dead, by op-data time), then joins as OBSERVER via
//                      the multi-use code — and finds read-only means it:
//                      its messages reject while its REPLICATION still works
//
// Run: npm run invitespike
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { deviceKeys, signOp, inviteKeys, inviteOfferOp, inviteRedeemOp } from './capability.mjs'
import { encodeInviteCode, decodeInviteCode } from './invite-code.mjs'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDEN = join(__dirname, '..', 'goldens', 'invite_autobase.json')
const UPDATE_GOLDEN = process.argv.includes('--update-golden')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

const PK = (b) => Buffer.alloc(32, b)

// Pinned identities → goldenable run.
const AUTH = deviceKeys(Buffer.alloc(32, 0xd4))
const DESK = deviceKeys(Buffer.alloc(32, 0xe5))
const PHONE = deviceKeys(Buffer.alloc(32, 0xf6))
const SEED_W = Buffer.alloc(32, 0x41) // one-time writer invite
const SEED_O = Buffer.alloc(32, 0x42) // multi-use observer invite
const SEED_X = Buffer.alloc(32, 0x43) // short-lived invite (will expire)
const INV_W = inviteKeys(SEED_W)
const INV_O = inviteKeys(SEED_O)
const INV_X = inviteKeys(SEED_X)

console.log('Messenger M2 — invite gate: a code mints capability, the fold is the bouncer\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-invite-'))
const mk = (dir, b, extra = {}) =>
  createMeshNode({ storage: join(tmp, dir), primaryKey: PK(b), authorityPub: AUTH.pubHex, mode: 'room', ...extra })
const a = await mk('a', 0x1a)
const b = await mk('b', 0x2b, { bootstrap: a.key })
const c = await mk('c', 0x3c, { bootstrap: a.key })

// The room opens: manifest + the three offers (all pre-fork, on the authority).
const A_OPS = [
  signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201' }, AUTH),
  inviteOfferOp({ seq: 2, actor: 'hub', ts: 200, invitePub: INV_W.pubHex, expiresAt: 100000, maxUses: 1 }, AUTH),
  inviteOfferOp({ seq: 3, actor: 'hub', ts: 300, invitePub: INV_O.pubHex, role: 'observer', expiresAt: 0, maxUses: 2 }, AUTH),
  inviteOfferOp({ seq: 4, actor: 'hub', ts: 400, invitePub: INV_X.pubHex, expiresAt: 450, maxUses: 1 }, AUTH),
]

// ── The pasteable codes (full encode→decode round-trip, as a human would) ──
const codeW = encodeInviteCode({ baseKey: a.key, authorityPub: AUTH.pubHex, inviteSeed: SEED_W, inviteId: 'hub:2' })
const codeO = encodeInviteCode({ baseKey: a.key, authorityPub: AUTH.pubHex, inviteSeed: SEED_O, inviteId: 'hub:3' })
const codeX = encodeInviteCode({ baseKey: a.key, authorityPub: AUTH.pubHex, inviteSeed: SEED_X, inviteId: 'hub:4' })
console.log(`  invite code (one-time writer): ${codeW.slice(0, 60)}…\n`)

const decW = decodeInviteCode(codeW)
const decO = decodeInviteCode(codeO)
const decX = decodeInviteCode(codeX)
check('code round-trip: baseKey + authorityPub + inviteId + seed survive z32',
  decW.baseKey === a.key && decW.authorityPub === AUTH.pubHex &&
  decW.inviteId === 'hub:2' && decW.inviteSeed.equals(SEED_W) &&
  decO.inviteId === 'hub:3' && decX.inviteId === 'hub:4')

// Redeem ops are built ONLY from the decoded codes — the string is the truth.
const redeemW = inviteRedeemOp({ seq: 5, actor: 'desk', ts: 500, inviteId: decW.inviteId }, inviteKeys(decW.inviteSeed), DESK)
const B_OPS = [
  redeemW,
  signOp({ seq: 6, actor: 'desk', ts: 600, kind: 'msg.post', body: 'in via the code — sovereignty by paste' }, DESK),
]
const C_OPS = [
  inviteRedeemOp({ seq: 7, actor: 'phone', ts: 700, inviteId: decW.inviteId }, inviteKeys(decW.inviteSeed), PHONE),  // exhausted
  inviteRedeemOp({ seq: 8, actor: 'phone', ts: 800, inviteId: decX.inviteId }, inviteKeys(decX.inviteSeed), PHONE),  // expired (ts 800 > 450)
  inviteRedeemOp({ seq: 9, actor: 'phone', ts: 900, inviteId: decO.inviteId }, inviteKeys(decO.inviteSeed), PHONE),  // observer — lands
  signOp({ seq: 10, actor: 'phone', ts: 1000, kind: 'msg.post', body: 'observer speaking!' }, PHONE),                // read-only → rejected
  signOp({ seq: 11, actor: 'phone', ts: 1100, kind: 'msg.read', upToActor: 'desk', upToSeq: 6 }, PHONE),             // even cursors → rejected
]
const TOTAL = A_OPS.length + B_OPS.length + C_OPS.length

let wires = [a.connect(b), a.connect(c), b.connect(c)]
const disconnect = () => { for (const cut of wires) cut(); wires = [] }
const reconnect = () => { wires = [a.connect(b), a.connect(c), b.connect(c)] }

await a.addWriter(b.writerKey)
await a.addWriter(c.writerKey)
await waitFor(async () => { await b.base.update(); return b.writable }, { label: 'B writable' })
await waitFor(async () => { await c.base.update(); return c.writable }, { label: 'C writable' })

for (const op of A_OPS) await a.append(op)
// Genuine offline fork: both joiners act blind, the linearizer sorts it out.
disconnect()
for (const op of B_OPS) await b.append(op)
for (const op of C_OPS) await c.append(op)
reconnect()

await waitFor(async () => {
  const [va, vb, vc] = await Promise.all([a.ops(), b.ops(), c.ops()])
  return va.length === TOTAL && vb.length === TOTAL && vc.length === TOTAL
}, { label: `all peers to see ${TOTAL} linearized ops`, timeout: 30000 })

const [da, db, dc] = await Promise.all([a.viewDigest(), b.viewDigest(), c.viewDigest()])
check('convergence: 3 peers, views byte-identical after the fork', da === db && db === dc)
const [sa, sb, sc] = await Promise.all([a.state(), b.state(), c.state()])
check('room state digests: byte-identical on all peers', sa.digest === sb.digest && sb.digest === sc.digest)

// ── The one-time writer path ──
check('desk: redemption granted writer at epoch 0, message folds',
  sa.grants?.[DESK.pubHex]?.role === 'writer' &&
  sa.messages.some((m) => m.msgId === 'desk:6' && m.body.includes('sovereignty by paste')))
check('one-time: the same code is EXHAUSTED for the second device, everywhere',
  [sa, sb, sc].every((s) => s.rejected.some((r) => r.actor === 'phone' && r.reason.includes('exhausted'))))
check('uses ledger: hub:2 consumed exactly once', sa.invites?.['hub:2']?.uses === 1 && sa.invites?.['hub:2']?.maxUses === 1)

// ── Expiry by OP-DATA time (no clock anywhere) ──
check('expiry: the short-lived code rejects by the REDEEM op’s own ts',
  [sa, sb, sc].every((s) => s.rejected.some((r) => r.actor === 'phone' && r.reason.includes('expired'))))

// ── The observer path ──
check('observer: multi-use code grants role observer', sa.grants?.[PHONE.pubHex]?.role === 'observer')
check('read-only means read-only: BOTH observer writes reject on every peer',
  [sa, sb, sc].every((s) => s.rejected.filter((r) => r.reason.includes('observer grant is read-only')).length === 2))
check('…while the observer still holds the full replicated view (pipe open)',
  dc === da && (await c.ops()).length === TOTAL)
check('taxonomy: all invite/role failures are REJECTIONS; zero chat skips',
  [sa, sb, sc].every((s) => s.rejected.length === 4 && s.skipped.length === 0))

if (UPDATE_GOLDEN || !existsSync(GOLDEN)) {
  writeFileSync(GOLDEN, JSON.stringify({ viewLength: TOTAL, viewDigest: da, stateDigest: sa.digest, state: sa }, null, 2) + '\n')
  console.log(`\n  (golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(GOLDEN, 'utf8'))
  check('golden: converged view matches pinned golden', golden.viewDigest === da)
  check('golden: invite state digest matches pinned golden', golden.stateDigest === sa.digest)
}

console.log(`\nview digest:  ${da}`)
console.log(`state digest: ${sa.digest}`)

disconnect()
await Promise.all([a.close(), b.close(), c.close()])
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nINVITE SPIKE GREEN ✅' : `\nINVITE SPIKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
