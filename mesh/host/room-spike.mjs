// room-spike.mjs — Messenger Wave 1 gate (Mission M1): the room fold through
// the REAL machinery — the same bar Missions A/C/D passed.
//
// A room is its OWN Autobase (campaign distinction #3). Three real peers:
//   peer A ("hub")   — room authority: declares the manifest, issues grants
//   peer B ("desk")  — procurement desk, granted member
//   peer C ("phone") — supplier phone, granted member
// Plus two DEVICE identities that ride other peers' writer cores — proving
// again that writer-core-auth ≠ device-auth (the op's signature is the fact):
//   BUTLER — an actorType:'agent' device, granted: its msg.draft-op folds as
//            INERT cargo (the graduation seam carries, never executes)
//   ROGUE  — in nobody's grant table: its message is rejected on every peer
//            even though the bytes replicate fine (pipe open, capability dead)
//
// The conversation crosses a GENUINE offline fork (wires cut, both sides write
// blind, wires return) and must reconverge byte-identically: posts, a threaded
// reply, an edit, a tombstone delete, reaction toggle on AND off, read cursors,
// the agent draft. Golden: goldens/room_autobase.json.
//
// Run: npm run roomspike
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { deviceKeys, signOp } from './capability.mjs'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDEN = join(__dirname, '..', 'goldens', 'room_autobase.json')
const UPDATE_GOLDEN = process.argv.includes('--update-golden')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

const PK = (b) => Buffer.alloc(32, b)

// Pinned device seeds → reproducible identities → goldenable run.
const AUTH = deviceKeys(Buffer.alloc(32, 0xd4))
const DESK = deviceKeys(Buffer.alloc(32, 0xe5))
const PHONE = deviceKeys(Buffer.alloc(32, 0xf6))
const BUTLER = deviceKeys(Buffer.alloc(32, 0x77))
const ROGUE = deviceKeys(Buffer.alloc(32, 0x99))

const DRAFT = JSON.stringify({
  kind: 'approval.decide', subject: 'posting:PO-2201',
  subjectType: 'posting_draft', decision: 'approved',
})

// ── The room's op script (seq/ts are pinned event data, never a live clock) ──
// Pre-fork: the authority declares the room and grants the members.
const A_OPS = [
  signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201', observersAllowed: true }, AUTH),
  signOp({ seq: 2, actor: 'hub', ts: 200, kind: 'cap.grant', device: DESK.pubHex, epoch: 0 }, AUTH),
  signOp({ seq: 3, actor: 'hub', ts: 300, kind: 'cap.grant', device: PHONE.pubHex, epoch: 0 }, AUTH),
  signOp({ seq: 4, actor: 'hub', ts: 400, kind: 'cap.grant', device: BUTLER.pubHex, epoch: 0 }, AUTH),
  // authority claims the room for phone first — desk's later self-claim (B_OPS,
  // seq 17) reassigns it. Seq 5 ties with desk's first post; actor 'desk' <
  // 'hub' breaks the tie, so this still lands after the manifest/grants and
  // well before the seq-17 reassignment.
  signOp({ seq: 5, actor: 'hub', ts: 1800, kind: 'room.claim', assignee: 'phone' }, AUTH),
]
// Written BLIND on peer B during the fork (desk's side of the conversation;
// the butler's draft rides desk's writer core with its OWN device signature).
const B_OPS = [
  signOp({ seq: 5, actor: 'desk', ts: 500, kind: 'msg.post', body: 'Can we ship the coils Thursday?', expectation: 'urgent' }, DESK),
  signOp({ seq: 8, actor: 'desk', ts: 800, kind: 'msg.react', msgId: 'phone:6', emoji: '👍', on: true }, DESK),
  signOp({ seq: 10, actor: 'butler', actorType: 'agent', ts: 1000, kind: 'msg.draft-op', body: 'Drafted the PO-2201 approval — needs a human decision', draft: DRAFT }, BUTLER),
  signOp({ seq: 11, actor: 'desk', ts: 1100, kind: 'msg.read', upToActor: 'phone', upToSeq: 7 }, DESK),
  signOp({ seq: 12, actor: 'desk', ts: 1200, kind: 'msg.post', body: 'typo — ignore this' }, DESK),
  signOp({ seq: 13, actor: 'desk', ts: 1300, kind: 'msg.delete', msgId: 'desk:12' }, DESK),
  // desk self-claims, reassigning off the authority's earlier assignment (below)
  signOp({ seq: 17, actor: 'desk', ts: 1700, kind: 'room.claim', assignee: 'desk' }, DESK),
]
// Written BLIND on peer C during the same fork (supplier's side; the rogue's
// knock rides phone's writer core and must die on every peer).
const C_OPS = [
  signOp({ seq: 6, actor: 'phone', ts: 600, kind: 'msg.post', body: 'Thursday morning works', replyTo: 'desk:5' }, PHONE),
  signOp({ seq: 7, actor: 'phone', ts: 700, kind: 'msg.edit', msgId: 'phone:6', body: 'Thursday afternoon works better' }, PHONE),
  signOp({ seq: 9, actor: 'phone', ts: 900, kind: 'msg.react', msgId: 'desk:5', emoji: '🔥', on: true }, PHONE),
  signOp({ seq: 14, actor: 'phone', ts: 1400, kind: 'msg.react', msgId: 'desk:5', emoji: '🔥', on: false }, PHONE),
  signOp({ seq: 15, actor: 'rogue', ts: 1500, kind: 'msg.post', body: 'let me into this deal' }, ROGUE),
  signOp({ seq: 16, actor: 'phone', ts: 1600, kind: 'msg.read', upToActor: 'desk', upToSeq: 5 }, PHONE),
]
const TOTAL = A_OPS.length + B_OPS.length + C_OPS.length

console.log('Messenger Wave 1 — room-fold gate: what was said joins what was done\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-room-'))
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

// The room opens; then a GENUINE offline fork: both sides talk blind.
for (const op of A_OPS) await a.append(op)
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

// ── The room itself ──
check('manifest: anchored to the business object (po/PO-2201)',
  sa.manifest?.title === 'PO-2201 — Steel Coils' && sa.manifest?.anchorType === 'po' && sa.manifest?.anchorId === 'PO-2201')

const msg = (id) => sa.messages.find((m) => m.msgId === id)
check('threading: the reply anchors to the question', msg('phone:6')?.replyTo === 'desk:5')
check('edit: last authored edit wins on every peer',
  msg('phone:6')?.body === 'Thursday afternoon works better' && msg('phone:6')?.edited === true)
check('tombstone: deleted message keeps its id, blanks its content',
  msg('desk:12')?.deleted === true && !msg('desk:12')?.body)
check('reactions: 👍 lands; the toggled-off 🔥 is pruned',
  sa.reactions?.['phone:6']?.['👍']?.desk === true && !sa.reactions?.['desk:5'])
check('read cursors: per-writer, both directions',
  sa.readCursors?.desk?.phone === 7 && sa.readCursors?.phone?.desk === 5)

// ── Expectation tags (Constitution Art. III §3, MSG-D16) ──
check('expectation: the sender-side tag rides the message',
  msg('desk:5')?.expectation === 'urgent')

// ── Claim/assign (Constitution Art. VI, MSG-D17): authority assigns phone
// first (seq 5), desk's later self-claim (seq 17) reassigns — last in
// canonical order wins, on every peer ──
check('claim: desk\'s self-claim reassignment wins over the authority\'s earlier assignment',
  sa.claim?.assignee === 'desk' && sa.claim?.byActor === 'desk' && sa.claim?.atSeq === 17)
check('claim: converges identically on all peers',
  sb.claim?.assignee === 'desk' && sc.claim?.assignee === 'desk')

// ── The graduation seam (campaign distinction #4) ──
check('draft-op: the agent draft folds as INERT cargo, marked actorType agent',
  msg('butler:10')?.draft === DRAFT && msg('butler:10')?.actorType === 'agent')
check('graduation border: the room state has NO business surface (no approvals key)',
  !('approvals' in sa) && !('stock' in sa))

// ── The capability plane, live (pipe open, capability dead) ──
check('rogue: rejected on ALL peers with the kernel words',
  [sa, sb, sc].every((s) => s.rejected.length === 1 &&
    s.rejected[0].actor === 'rogue' && s.rejected[0].reason.includes('no grant for device')))
check('taxonomy: zero chat-rule skips in an honest conversation',
  [sa, sb, sc].every((s) => s.skipped.length === 0))
check(`applied: ${TOTAL - 1} of ${TOTAL} ops fold (the rogue is the remainder)`,
  sa.applied === TOTAL - 1 && sa.opsHashed === TOTAL)

if (UPDATE_GOLDEN || !existsSync(GOLDEN)) {
  writeFileSync(GOLDEN, JSON.stringify({ viewLength: TOTAL, viewDigest: da, stateDigest: sa.digest, state: sa }, null, 2) + '\n')
  console.log(`\n  (golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(GOLDEN, 'utf8'))
  check('golden: converged view matches pinned golden', golden.viewDigest === da)
  check('golden: room state digest matches pinned golden', golden.stateDigest === sa.digest)
}

console.log(`\nview digest:  ${da}`)
console.log(`state digest: ${sa.digest}`)

disconnect()
await Promise.all([a.close(), b.close(), c.close()])
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nROOM SPIKE GREEN ✅' : `\nROOM SPIKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
