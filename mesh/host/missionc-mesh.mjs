// missionc-mesh.mjs — Mission C gate: the REAL kernel packages
// (money/approval/actor/policy), compiled to wasip1, as the law of the mesh —
// proven through real Autobase linearization over a genuine offline fork.
//
// The scenario splits the Mission C op set across three writers:
//   peer A ("dev-a"/sarat's box)  — inventory receipts + AR limit/charges + the human approval
//   peer B ("dev-b")              — the oversell, an over-limit charge, the policy violation
//   peer C ("butler-ai")          — an AI agent peer: tries to APPROVE a posting and
//                                   OVERRIDE a policy violation. The kernel boundary
//                                   must reject BOTH — identically on every peer.
//
// All three write OFFLINE (wires cut), then the fork merges. Gate: byte-identical
// views, kernel-identical state, the agent rejected everywhere with kernel words.
//
// Run: npm run missionc

import { createMeshNode, waitFor } from './mesh-node.mjs'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDEN = join(__dirname, '..', 'goldens', 'missionc_autobase.json')
const UPDATE_GOLDEN = process.argv.includes('--update-golden')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

const PK = (b) => Buffer.alloc(32, b)

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
const PEER_C_OPS = [ // the agent's own peer — the boundary must hold from ANY writer
  { seq: 4, actor: 'butler-ai', ts: 400, kind: 'approval.decide', subject: 'posting-77', subjectType: 'posting_draft', decision: 'approved', actorType: 'agent', authority: 1, correlationId: 'c-1' },
  { seq: 6, actor: 'butler-ai', ts: 510, kind: 'policy.override', policyId: 'VAT-DEADLINE', reason: 'agent says it is fine', actorType: 'agent', authority: 1 },
]
const TOTAL = PEER_A_OPS.length + PEER_B_OPS.length + PEER_C_OPS.length

console.log('Sovereign Mesh — Mission C gate: real kernel law over real Autobase\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-missionc-'))
const a = await createMeshNode({ storage: join(tmp, 'a'), primaryKey: PK(0x1a) })
const b = await createMeshNode({ storage: join(tmp, 'b'), bootstrap: a.key, primaryKey: PK(0x2b) })
const c = await createMeshNode({ storage: join(tmp, 'c'), bootstrap: a.key, primaryKey: PK(0x3c) })

let wires = [a.connect(b), a.connect(c), b.connect(c)]
const disconnect = () => { for (const cut of wires) cut(); wires = [] }
const reconnect = () => { wires = [a.connect(b), a.connect(c), b.connect(c)] }

await a.addWriter(b.writerKey)
await a.addWriter(c.writerKey)
await waitFor(async () => { await b.base.update(); return b.writable }, { label: 'B writable' })
await waitFor(async () => { await c.base.update(); return c.writable }, { label: 'C writable' })

// The fork: all three writers blind, including the agent's peer.
disconnect()
for (const op of PEER_A_OPS) await a.append(op)
for (const op of PEER_B_OPS) await b.append(op)
for (const op of PEER_C_OPS) await c.append(op)
reconnect()

await waitFor(async () => {
  const [va, vb, vc] = await Promise.all([a.ops(), b.ops(), c.ops()])
  return va.length === TOTAL && vb.length === TOTAL && vc.length === TOTAL
}, { label: `all peers to see ${TOTAL} linearized ops`, timeout: 30000 })

const [da, db, dc] = await Promise.all([a.viewDigest(), b.viewDigest(), c.viewDigest()])
check('convergence: 3 peers, views byte-identical', da === db && db === dc)

const [sa, sb, sc] = await Promise.all([a.state(), b.state(), c.state()])
check('state digests: byte-identical on all peers', sa.digest === sb.digest && sb.digest === sc.digest)

// Kernel law, domain by domain (checked on peer A; digests prove the rest):
check('inventory: TX-100 == 4, oversell rejected', sa.stock['TX-100'] === 4)
check('money: AR balance 250000/500000 BHD (over-limit charge rejected)',
  sa.ar['CUST-01']?.balanceMinor === 250000 && sa.ar['CUST-01']?.limitMinor === 500000)
check('approval: posting-77 approved by the HUMAN operator',
  sa.approvals['posting-77']?.decision === 'approved' && sa.approvals['posting-77']?.actor === 'sarat')
check('policy: VAT-DEADLINE overridden by the HUMAN with a reason',
  sa.policies['VAT-DEADLINE']?.status === 'overridden' && sa.policies['VAT-DEADLINE']?.overriddenBy === 'sarat')

// The flagship: the agent is rejected on EVERY peer, with the kernel's words.
const agentRejects = (s) => s.rejected.filter((r) => r.actor === 'butler-ai')
check('AI-authority boundary: agent approve + override rejected on ALL peers',
  [sa, sb, sc].every((s) => agentRejects(s).length === 2))
check('boundary reasons carry the kernel language',
  agentRejects(sa).every((r) => /agent|AI-authority/.test(r.reason)),
  JSON.stringify(agentRejects(sa).map((r) => r.reason)))
check('rejections: exactly 4 total (oversell, over-limit, agent×2), identical everywhere',
  [sa, sb, sc].every((s) => s.rejected.length === 4))

if (UPDATE_GOLDEN || !existsSync(GOLDEN)) {
  writeFileSync(GOLDEN, JSON.stringify({ viewLength: TOTAL, viewDigest: da, stateDigest: sa.digest, state: sa }, null, 2) + '\n')
  console.log(`\n  (golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(GOLDEN, 'utf8'))
  check('golden: converged view matches pinned golden', golden.viewDigest === da)
  check('golden: kernel state digest matches pinned golden', golden.stateDigest === sa.digest)
}

console.log(`\nview digest:  ${da}`)
console.log(`state digest: ${sa.digest}`)

disconnect()
await Promise.all([a.close(), b.close(), c.close()])
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nMISSION C GREEN ✅' : `\nMISSION C RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
