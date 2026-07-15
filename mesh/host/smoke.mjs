// smoke.mjs — Sovereign Mesh determinism smoke over the real Go->WASM boundary.
//
// This is the SCAFFOLD-stage proof (not yet the full Mission A gate). It proves:
//   1. BOUNDARY: JS can drive the wasip1 Go reducer and marshal ops in / state out.
//   2. INVARIANT: the concurrent-offline oversell is deterministically rejected.
//   3. CONVERGENCE (reducer half): 3 "peers", each handed the op-log in a
//      DIFFERENT arrival order, produce byte-identical digests.
//   4. GOLDEN: the converged state matches the pinned golden exactly.
//
// What it does NOT yet prove (the remaining on-site Mission A work, see
// docs/MESH_PROGRESS.md): real Autobase linearization over real Hypercores
// replicated across real Holesail tunnels between ≥2 machines. Here the 3 peers
// are 3 in-process invocations with shuffled input — the transport/replication
// layer is stubbed. That is the honest line between "scaffold ready" and "gate green".
//
// Run: node mesh/host/smoke.mjs   (after building mesh/dist/reducer.wasm)

import { applyViaWasm } from './apply.mjs'
import { readFileSync, writeFileSync, existsSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDEN = join(__dirname, '..', 'goldens', 'inventory_basic.json')
const UPDATE_GOLDEN = process.argv.includes('--update-golden')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) {
    console.log(`  ✓ ${name}`)
  } else {
    failures++
    console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`)
  }
}

// The canonical scenario (mirrors reducer_test.go canonicalOps): two devices
// move two SKUs; dev-b's Seq-1 sale + dev-a's Seq-2 sale both draw TX-100 down,
// and the one that would breach the floor is deterministically rejected.
const OPS = [
  { seq: 1, actor: 'dev-a', sku: 'TX-100', delta: +10, ts: 100 },
  { seq: 2, actor: 'dev-a', sku: 'TX-100', delta: -6, ts: 200 },
  { seq: 1, actor: 'dev-b', sku: 'TX-100', delta: -6, ts: 150 },
  { seq: 1, actor: 'dev-a', sku: 'PH-200', delta: +3, ts: 120 },
  { seq: 2, actor: 'dev-b', sku: 'PH-200', delta: +4, ts: 220 },
]

// A deterministic shuffle (seeded LCG) so peer arrival orders vary but the run
// is itself reproducible — the TEST may be pseudo-random; the REDUCER may not.
function seededShuffle(arr, seed) {
  const a = arr.slice()
  let s = seed >>> 0
  for (let i = a.length - 1; i > 0; i--) {
    s = (1664525 * s + 1013904223) >>> 0
    const j = s % (i + 1)
    ;[a[i], a[j]] = [a[j], a[i]]
  }
  return a
}

console.log('Sovereign Mesh — determinism smoke (Go→WASM boundary)\n')

// 1. BOUNDARY + basic apply
const base = applyViaWasm(OPS)
check('boundary: reducer.wasm ran and returned JSON', !!base && typeof base.digest === 'string')
check('state: TX-100 == 4 (10 - 6, second -6 rejected)', base.stock['TX-100'] === 4,
  `got ${base.stock['TX-100']}`)
check('state: PH-200 == 7 (3 + 4)', base.stock['PH-200'] === 7, `got ${base.stock['PH-200']}`)

// 2. INVARIANT: exactly one deterministic oversell rejection
check('invariant: exactly 1 oversell rejected', base.rejected.length === 1,
  `got ${base.rejected.length}`)
check('invariant: the rejected op is dev-a / TX-100 / seq 2 (canonical loser)',
  base.rejected.length === 1 &&
  base.rejected[0].actor === 'dev-a' &&
  base.rejected[0].sku === 'TX-100' &&
  base.rejected[0].seq === 2,
  JSON.stringify(base.rejected[0]))
check('invariant: no SKU ever below 0',
  Object.values(base.stock).every((q) => q >= 0))

// 3. CONVERGENCE (reducer half): 3 peers, 3 different arrival orders, 1 digest
const peerA = applyViaWasm(seededShuffle(OPS, 0xA11CE))
const peerB = applyViaWasm(seededShuffle(OPS, 0xB0B))
const peerC = applyViaWasm(seededShuffle(OPS, 0xC0FFEE))
check('convergence: peer A digest == base digest', peerA.digest === base.digest)
check('convergence: peer B digest == base digest', peerB.digest === base.digest)
check('convergence: peer C digest == base digest', peerC.digest === base.digest)
check('convergence: all 3 peers byte-identical',
  peerA.digest === peerB.digest && peerB.digest === peerC.digest)

// 4. GOLDEN
if (UPDATE_GOLDEN || !existsSync(GOLDEN)) {
  writeFileSync(GOLDEN, JSON.stringify(base, null, 2) + '\n')
  console.log(`\n  (golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(GOLDEN, 'utf8'))
  check('golden: converged state matches pinned golden', golden.digest === base.digest,
    `golden ${golden.digest?.slice(0, 12)} vs got ${base.digest?.slice(0, 12)}`)
}

console.log(`\ndigest: ${base.digest}`)
console.log(failures === 0 ? '\nSMOKE GREEN ✅' : `\nSMOKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
