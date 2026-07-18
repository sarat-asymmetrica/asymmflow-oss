// wave1-local.mjs — Mission A, replication half, stage 1 (single process).
//
// Turns Wave 0's stubbed 3-peer smoke into REAL Autobase linearization over
// REAL Corestores with REAL replication streams — including a genuine
// concurrent-offline causal fork:
//
//   1. Peer A founds the base; B and C join by key over replication streams.
//   2. A grants B and C write access through the linearizer (addWriter ops).
//   3. The wires are CUT. dev-a (A) and dev-b (B) append their ops fully
//      offline — a real causal fork, not a shuffled array.
//   4. The wires reconnect; Autobase linearizes the fork; all three peers must
//      converge to a BYTE-IDENTICAL view (raw view digest equality).
//   5. The kernel reducer materializes state from each peer's view: the
//      oversell is rejected deterministically (same loser on every peer) and
//      the state digest must equal Wave 0's pinned golden — the same 5 ops
//      through the same reducer, now arriving via the real machinery.
//
// Run: npm run wave1   (builds the wasm, then this)

import { createMeshNode, waitFor } from './mesh-node.mjs'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const REDUCER_GOLDEN = join(__dirname, '..', 'goldens', 'inventory_basic.json')
const VIEW_GOLDEN = join(__dirname, '..', 'goldens', 'inventory_autobase.json')
const UPDATE_GOLDEN = process.argv.includes('--update-golden')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

// Fixed primary keys -> deterministic core keys -> a goldenable converged view.
// (The TEST may pin identities; the REDUCER never depends on them.)
const PK = (hexByte) => Buffer.alloc(32, hexByte)

// The canonical Wave-0 scenario, now split across two genuinely-offline writers.
const DEV_A_OPS = [
  { seq: 1, actor: 'dev-a', sku: 'TX-100', delta: +10, ts: 100 },
  { seq: 2, actor: 'dev-a', sku: 'TX-100', delta: -6, ts: 200 },
  { seq: 1, actor: 'dev-a', sku: 'PH-200', delta: +3, ts: 120 },
]
const DEV_B_OPS = [
  { seq: 1, actor: 'dev-b', sku: 'TX-100', delta: -6, ts: 150 },
  { seq: 2, actor: 'dev-b', sku: 'PH-200', delta: +4, ts: 220 },
]
const TOTAL_OPS = DEV_A_OPS.length + DEV_B_OPS.length

console.log('Sovereign Mesh — Wave 1 gate, stage 1: real Autobase over real Corestore replication\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-wave1-'))
const dirs = { a: join(tmp, 'peer-a'), b: join(tmp, 'peer-b'), c: join(tmp, 'peer-c') }

const a = await createMeshNode({ storage: dirs.a, primaryKey: PK(0xaa) })
const b = await createMeshNode({ storage: dirs.b, bootstrap: a.key, primaryKey: PK(0xbb) })
const c = await createMeshNode({ storage: dirs.c, bootstrap: a.key, primaryKey: PK(0xcc) })

check('bootstrap: B and C joined A\'s base by key', b.key === a.key && c.key === a.key)

// Full-mesh wires.
let wires = [a.connect(b), a.connect(c), b.connect(c)]
const disconnect = () => { for (const cut of wires) cut(); wires = [] }
const reconnect = () => { wires = [a.connect(b), a.connect(c), b.connect(c)] }

// A grants write access to B and C through the linearizer.
await a.addWriter(b.writerKey)
await a.addWriter(c.writerKey)
await waitFor(async () => { await b.base.update(); return b.writable }, { label: 'B writable' })
await waitFor(async () => { await c.base.update(); return c.writable }, { label: 'C writable' })
check('writers: B and C granted write access via addWriter ops', b.writable && c.writable)

// ── THE FORK: cut every wire, write offline on both sides ──────────────────
disconnect()
for (const op of DEV_A_OPS) await a.append(op)
for (const op of DEV_B_OPS) await b.append(op)

const aAlone = await a.ops()
const bAlone = await b.ops()
check('fork: offline peers hold only their own writes',
  aAlone.length === DEV_A_OPS.length && bAlone.length === DEV_B_OPS.length,
  `A sees ${aAlone.length}, B sees ${bAlone.length}`)

// ── RECONNECT: the linearizer must merge the fork on every peer ────────────
reconnect()
await waitFor(async () => {
  const [va, vb, vc] = await Promise.all([a.ops(), b.ops(), c.ops()])
  return va.length === TOTAL_OPS && vb.length === TOTAL_OPS && vc.length === TOTAL_OPS
}, { label: `all peers to see ${TOTAL_OPS} linearized ops`, timeout: 30000 })

const [digestA, digestB, digestC] = await Promise.all([a.viewDigest(), b.viewDigest(), c.viewDigest()])
check('convergence: A and B view digests byte-identical', digestA === digestB)
check('convergence: A and C view digests byte-identical', digestA === digestC)

// ── STATE: the kernel reducer over each peer's REAL linearized view ────────
const [sa, sb, sc] = await Promise.all([a.state(), b.state(), c.state()])
check('state: TX-100 == 4 on all peers',
  sa.stock['TX-100'] === 4 && sb.stock['TX-100'] === 4 && sc.stock['TX-100'] === 4)
check('state: PH-200 == 7 on all peers',
  sa.stock['PH-200'] === 7 && sb.stock['PH-200'] === 7 && sc.stock['PH-200'] === 7)
check('invariant: exactly 1 oversell rejected, same loser on every peer (dev-a seq 2)',
  [sa, sb, sc].every((s) => s.rejected.length === 1 &&
    s.rejected[0].actor === 'dev-a' && s.rejected[0].seq === 2 && s.rejected[0].sku === 'TX-100'))
check('invariant: no SKU below 0 on any peer',
  [sa, sb, sc].every((s) => Object.values(s.stock).every((q) => q >= 0)))
check('state digests: all peers byte-identical', sa.digest === sb.digest && sb.digest === sc.digest)

// ── GOLDENS ────────────────────────────────────────────────────────────────
// (1) The reducer state must equal Wave 0's pinned golden — same ops, same
//     reducer, now via the real replication machinery.
const reducerGolden = JSON.parse(readFileSync(REDUCER_GOLDEN, 'utf8'))
check('golden: state digest equals Wave 0 reducer golden', sa.digest === reducerGolden.digest,
  `golden ${reducerGolden.digest?.slice(0, 12)} vs got ${sa.digest?.slice(0, 12)}`)

// (2) The CONVERGED AUTOBASE VIEW itself (deterministic under the pinned
//     primary keys): linearized op order + view digest.
const viewSnapshot = { viewLength: TOTAL_OPS, viewDigest: digestA, linearized: await a.ops() }
if (UPDATE_GOLDEN || !existsSync(VIEW_GOLDEN)) {
  writeFileSync(VIEW_GOLDEN, JSON.stringify(viewSnapshot, null, 2) + '\n')
  console.log(`\n  (view golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${VIEW_GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(VIEW_GOLDEN, 'utf8'))
  check('golden: converged Autobase view matches pinned golden', golden.viewDigest === digestA,
    `golden ${golden.viewDigest?.slice(0, 12)} vs got ${digestA?.slice(0, 12)}`)
}

console.log(`\nview digest:  ${digestA}`)
console.log(`state digest: ${sa.digest}`)

disconnect()
await Promise.all([a.close(), b.close(), c.close()])
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nWAVE 1 STAGE 1 GREEN ✅' : `\nWAVE 1 STAGE 1 RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
