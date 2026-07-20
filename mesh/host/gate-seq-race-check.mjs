// gate-seq-race-check.mjs — executable regression guard (merge-gate finding,
// 2026-07-20): posts dispatched CONCURRENTLY on a COLD seq counter must
// receive distinct seqs. bare-bridge.mjs's original takeSeq had an await
// between its has() check and its set(), so pipelined frames on the single
// stdio transport could both read max+1 and share a seq — a silent message
// drop under the fold's seq discipline. Verified red against the pre-fix
// takeSeq (FAIL) and green against the promise-chained fix (PASS) — the
// probe can report both colours. Run: npm run seqrace
import { createMeshNode } from './mesh-node.mjs'
import { signOp, grantOp } from './capability.mjs'
import { createBridgeCore } from './bare-bridge.mjs'
import hcrypto from 'hypercore-crypto'
import { rmSync } from '#fs'

function deviceKeys(seed) {
  const kp = hcrypto.keyPair(seed)
  return { pubHex: kp.publicKey.toString('hex'), secretKey: kp.secretKey, publicKey: kp.publicKey }
}

const PK = (b) => Buffer.alloc(32, b)
const HUB = deviceKeys(PK(0xd4))
const tmp = `.gate-seq-race-${Date.now()}`

const hubPo = await createMeshNode({ storage: `${tmp}/hub-po`, primaryKey: PK(0x1a), authorityPub: HUB.pubHex, mode: 'room' })
await hubPo.append(signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'race check', anchorType: 'po', anchorId: 'PO-RACE' }, HUB))
await hubPo.append(grantOp({ seq: 2, actor: 'hub', ts: 200, device: HUB.pubHex, epoch: 0 }, HUB))

const core = createBridgeCore({ rooms: new Map([[hubPo.key, hubPo]]), actor: 'hub', deviceKeys: HUB, storageDir: `${tmp}/store` })

// COLD counter, concurrent dispatch — exactly the pipelined-stdio shape.
const [a, b, c] = await Promise.all([
  core.dispatch({ id: 1, method: 'post', params: { roomKey: hubPo.key, body: 'first', ts: 300 } }),
  core.dispatch({ id: 2, method: 'post', params: { roomKey: hubPo.key, body: 'second', ts: 301 } }),
  core.dispatch({ id: 3, method: 'post', params: { roomKey: hubPo.key, body: 'third', ts: 302 } }),
])
const seqs = [a, b, c].map((r) => r?.result?.seq)
const allOk = [a, b, c].every((r) => r?.ok)
const distinct = new Set(seqs).size === 3
console.log(`ok=${allOk} seqs=${JSON.stringify(seqs)} distinct=${distinct}`)
console.log(allOk && distinct ? 'SEQ RACE CHECK PASS' : 'SEQ RACE CHECK FAIL')
try { rmSync(tmp, { recursive: true, force: true }) } catch {}
process.exit(allOk && distinct ? 0 : 1)
