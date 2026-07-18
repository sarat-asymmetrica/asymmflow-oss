// missiond-mesh.mjs — Mission D gate: the Ed25519 grant-with-epochs capability
// layer, proven through real Autobase machinery.
//
// THE claim under test (campaign §Mission D): transport-auth ≠ capability-auth.
// All three peers below are FULL members of the replication plane — they are in
// the Autobase writer set, their cores replicate, their ops arrive on every
// peer ("the pipe still opens"). But delivery grants NOTHING:
//   peer A ("sarat-hub")  — holds the mesh AUTHORITY key; issues grants/epochs
//   peer B ("laptop")     — granted at epoch 0, revoked by the epoch-1 bump,
//                           re-granted at epoch 1 (the wife's-laptop role :))
//   peer C ("rogue")      — in the writer set the whole time, NEVER granted;
//                           also tries to grant ITSELF (not the authority → dead)
// Every op is Ed25519-signed in JS and verified inside the Go/WASM reducer —
// this run is the standing cross-runtime proof that both sides build identical
// signable bytes AND derive identical keys from the same seeds.
//
// Run: npm run missiond
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { deviceKeys, signOp, grantOp, epochOp } from './capability.mjs'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDEN = join(__dirname, '..', 'goldens', 'missiond_autobase.json')
const UPDATE_GOLDEN = process.argv.includes('--update-golden')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

const PK = (b) => Buffer.alloc(32, b)

// Device identities from pinned seeds — the SAME seeds missiond_test.go uses,
// so the reducer state here must byte-match the Go unit-test expectations.
const AUTH = deviceKeys(Buffer.alloc(32, 0xa1))
const LAPTOP = deviceKeys(Buffer.alloc(32, 0xb2))
const ROGUE = deviceKeys(Buffer.alloc(32, 0xc3))

// The canonical Mission D op set (mirror of missionDOps in Go), split by writer.
const A_OPS = [
  grantOp({ seq: 1, actor: 'sarat-hub', ts: 100, device: LAPTOP.pubHex, epoch: 0 }, AUTH),
  signOp({ seq: 3, actor: 'sarat-hub', ts: 300, sku: 'TX-100', delta: 5 }, AUTH),
  epochOp({ seq: 6, actor: 'sarat-hub', ts: 600, epoch: 1 }, AUTH), // revocation wave
  grantOp({ seq: 8, actor: 'sarat-hub', ts: 800, device: LAPTOP.pubHex, epoch: 1 }, AUTH), // re-issue
]
const B_OPS = [
  signOp({ seq: 2, actor: 'laptop', ts: 200, sku: 'TX-100', delta: 10 }, LAPTOP),
  signOp({ seq: 7, actor: 'laptop', ts: 700, sku: 'TX-100', delta: -4 }, LAPTOP), // stale-epoch victim
  signOp({ seq: 9, actor: 'laptop', ts: 900, sku: 'TX-100', delta: -2 }, LAPTOP), // post-re-grant
]
const C_OPS = [
  signOp({ seq: 4, actor: 'rogue', ts: 400, sku: 'TX-100', delta: -3 }, ROGUE), // never granted
  grantOp({ seq: 5, actor: 'rogue', ts: 500, device: ROGUE.pubHex, epoch: 0 }, ROGUE), // self-grant
]
const TOTAL = A_OPS.length + B_OPS.length + C_OPS.length

console.log('Sovereign Mesh — Mission D gate: grants-with-epochs above the pipe\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-missiond-'))
const mk = (dir, b, extra = {}) =>
  createMeshNode({ storage: join(tmp, dir), primaryKey: PK(b), authorityPub: AUTH.pubHex, ...extra })
const a = await mk('a', 0x4a)
const b = await mk('b', 0x5b, { bootstrap: a.key })
const c = await mk('c', 0x6c, { bootstrap: a.key })

let wires = [a.connect(b), a.connect(c), b.connect(c)]
const disconnect = () => { for (const cut of wires) cut(); wires = [] }
const reconnect = () => { wires = [a.connect(b), a.connect(c), b.connect(c)] }

// The transport/writer plane is opened to EVERYONE — including the rogue.
// That is the point: writer-set membership must confer zero capability.
await a.addWriter(b.writerKey)
await a.addWriter(c.writerKey)
await waitFor(async () => { await b.base.update(); return b.writable }, { label: 'B writable' })
await waitFor(async () => { await c.base.update(); return c.writable }, { label: 'C writable' })

// Genuine offline fork: all three write blind, then the wires come back.
disconnect()
for (const op of A_OPS) await a.append(op)
for (const op of B_OPS) await b.append(op)
for (const op of C_OPS) await c.append(op)
reconnect()

await waitFor(async () => {
  const [va, vb, vc] = await Promise.all([a.ops(), b.ops(), c.ops()])
  return va.length === TOTAL && vb.length === TOTAL && vc.length === TOTAL
}, { label: `all peers to see ${TOTAL} linearized ops`, timeout: 30000 })

const [da, db, dc] = await Promise.all([a.viewDigest(), b.viewDigest(), c.viewDigest()])
check('convergence: 3 peers, views byte-identical', da === db && db === dc)

const [sa, sb, sc] = await Promise.all([a.state(), b.state(), c.state()])
check('state digests: byte-identical on all peers', sa.digest === sb.digest && sb.digest === sc.digest)

// ── The pipe is OPEN… (transport-auth: the rogue replicates like anyone)
check('transport: the ROGUE peer holds the full converged view (pipe open)',
  dc === da && (await c.ops()).length === TOTAL)

// ── …and the capability is DEAD (capability-auth: its ops never count)
const rejects = (s, who) => s.rejected.filter((r) => r.actor === who)
check('capability: rogue inventory op rejected on ALL peers (writer set ≠ grant)',
  [sa, sb, sc].every((s) => rejects(s, 'rogue').some((r) => r.reason.includes('no grant for device'))))
check('capability: rogue SELF-grant rejected everywhere (only the authority grants)',
  [sa, sb, sc].every((s) => rejects(s, 'rogue').some((r) => r.reason.includes('must be signed by the mesh authority'))))

// ── Revocation by epoch: the laptop's mid-life op is stale, its re-granted op lands
check('revocation: laptop op under the OLD epoch rejected as stale on all peers',
  [sa, sb, sc].every((s) => rejects(s, 'laptop').some((r) => r.reason.includes('is stale'))))
check('re-issue: laptop writes again after the epoch-1 re-grant',
  sa.grants?.[LAPTOP.pubHex]?.epoch === 1)
check('kernel arithmetic: TX-100 == 13 (+10 +5 -2; rejected ops NEVER touch stock)',
  sa.stock['TX-100'] === 13)
check('epoch: converged capEpoch == 1 on all peers',
  [sa, sb, sc].every((s) => s.capEpoch === 1))
check('rejections: exactly 3, identical everywhere',
  [sa, sb, sc].every((s) => s.rejected.length === 3))
check('rogue never appears in the grant table',
  [sa, sb, sc].every((s) => !(ROGUE.pubHex in (s.grants ?? {}))))

if (UPDATE_GOLDEN || !existsSync(GOLDEN)) {
  writeFileSync(GOLDEN, JSON.stringify({ viewLength: TOTAL, viewDigest: da, stateDigest: sa.digest, state: sa }, null, 2) + '\n')
  console.log(`\n  (golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(GOLDEN, 'utf8'))
  check('golden: converged view matches pinned golden', golden.viewDigest === da)
  check('golden: capability state digest matches pinned golden', golden.stateDigest === sa.digest)
}

console.log(`\nview digest:  ${da}`)
console.log(`state digest: ${sa.digest}`)

disconnect()
await Promise.all([a.close(), b.close(), c.close()])
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nMISSION D GREEN ✅' : `\nMISSION D RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
