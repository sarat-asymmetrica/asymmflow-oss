// transcript-spike.mjs — Messenger Constitution Article V §5 gate: self-serve
// evidence export. "One gesture, from the target's OWN copy, no admin
// mediation" — a harassment target (or auditor) exports a signed transcript
// of a room they already hold and anyone can verify it OFFLINE, later,
// trusting nothing the bundle itself claims.
//
// Single writer, causally chained (GL-2: a view/state golden is valid on
// this shape — no concurrent fork here, the export/verify boundary is the
// thing under test, not the linearizer).
//
// Scenario, one enforced room ("hub" is authority, "desk"/"phone"/"guest"
// are members), all ops appended through ONE node:
//   manifest -> grants -> a reply thread -> an edit -> a tombstoned delete ->
//   a reaction -> a message from "guest" -> an epoch bump (revocation wave
//   that does NOT re-issue guest) -> guest's post-revocation message, which
//   the fold REJECTS but which still rides the honest log verbatim.
//
// Then a SEPARATE social room (Article II: unanchored, no authority — the
// human layer) with its own tiny export, proving Article V's evidence right
// applies there too, not just to work rooms.
//
// Run: npm run transcriptspike
import { createMeshNode } from './mesh-node.mjs'
import { deviceKeys, signOp, epochOp, signable } from './capability.mjs'
import { exportTranscript } from './export-transcript.mjs'
import { verifyTranscript } from './verify-transcript.mjs'
import { createHash } from 'node:crypto'
import hcrypto from 'hypercore-crypto'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDEN = join(__dirname, '..', 'goldens', 'transcript_autobase.json')
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
const GUEST = deviceKeys(Buffer.alloc(32, 0x88))
const FORGER = deviceKeys(Buffer.alloc(32, 0x99)) // never granted, anywhere

console.log('Messenger — transcript spike: Article V §5 self-serve evidence export\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-transcript-'))
const mk = (dir, b, extra = {}) =>
  createMeshNode({ storage: join(tmp, dir), primaryKey: PK(b), ...extra })

// ── Anchored, enforced room: the harassment-evidence scenario ──
const hub = await mk('hub', 0x1a, { authorityPub: AUTH.pubHex, mode: 'room' })

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
  epochOp({ seq: 12, actor: 'hub', ts: 1200, epoch: 1 }, AUTH), // revocation wave: guest NOT re-issued
  signOp({ seq: 13, actor: 'guest', ts: 1300, kind: 'msg.post', body: 'still trying to post after being let go' }, GUEST),
]

for (const op of OPS) await hub.append(op)
await hub.base.update()

const preState = await hub.state()
check('setup: the room folds as expected (1 rejection — guest post-revocation)',
  preState.rejected.length === 1 && preState.rejected[0].actor === 'guest' && preState.rejected[0].reason.includes('is stale'))
check('setup: applied count carries the honest math (13 ops hashed, 12 applied)',
  preState.opsHashed === OPS.length && preState.applied === OPS.length - 1)

// ── (1) export -> verify => VERIFIED ──
const bundle = await exportTranscript(hub, { exportedBy: 'desk (self-export, no admin involved)' })
check('export: format tag', bundle.format === 'asymm-transcript.v1')
check('export: roomKey carried', bundle.roomKey === hub.key)
check('export: authorityPub carried from node config', bundle.authorityPub === AUTH.pubHex)
check('export: ops carried VERBATIM (same length, same sig/devicePub bytes)',
  bundle.ops.length === OPS.length &&
  bundle.ops.every((op, i) => op.sig === OPS[i].sig && op.devicePub === OPS[i].devicePub))
check('export: the tombstone exports blanked, not omitted',
  bundle.ops.find((o) => o.kind === 'msg.delete' && o.msgId === 'desk:8') !== undefined)
check('export: the rejected post-revocation op still rides the log',
  bundle.ops.some((o) => o.actor === 'guest' && o.seq === 13))
check('export: digests match the node\'s own calls',
  bundle.stateDigest === preState.digest && bundle.viewDigest === (await hub.viewDigest()))

const verdict1 = verifyTranscript(bundle)
check('verify: VERIFIED', verdict1.verified === true)
check('verify: every op signature valid', verdict1.allSigsValid === true && verdict1.ops.every((v) => v.sigValid))
check('verify: digest matches', verdict1.digestMatches === true && verdict1.recomputedDigest === bundle.stateDigest)
check('verify: per-op verdict shape carries seq/actor/kind/sigValid',
  verdict1.ops.every((v) => 'seq' in v && 'actor' in v && 'kind' in v && 'sigValid' in v))

// input immutability: bundle must be untouched by verification
const bundleSnapshotJSON = JSON.stringify(bundle)

// ── (2) TAMPER a body byte -> that op's sig fails, overall not verified ──
const tampered = JSON.parse(JSON.stringify(bundle))
const tamperOp = tampered.ops.find((o) => o.msgId === 'desk:5' && o.kind === 'msg.edit')
tamperOp.body = tamperOp.body.slice(0, -1) + (tamperOp.body.slice(-1) === 'M' ? 'X' : 'M')
const verdict2 = verifyTranscript(tampered)
check('tamper: the mutated op\'s signature fails', verdict2.ops.find((v) => v.seq === tamperOp.seq)?.sigValid === false)
check('tamper: overall verdict is NOT verified', verdict2.verified === false)
check('tamper: original bundle untouched by verification', JSON.stringify(bundle) === bundleSnapshotJSON)

// ── (3) DROP an op -> digest mismatch reported ──
const dropped = JSON.parse(JSON.stringify(bundle))
dropped.ops = dropped.ops.filter((o) => !(o.seq === 10 && o.actor === 'phone')) // drop the reaction
const verdict3 = verifyTranscript(dropped)
check('drop: every remaining op still signature-valid', verdict3.allSigsValid === true)
check('drop: digest mismatch is what catches it', verdict3.digestMatches === false)
check('drop: overall verdict is NOT verified', verdict3.verified === false)
check('drop: original bundle untouched by verification', JSON.stringify(bundle) === bundleSnapshotJSON)

// ── (4) FORGE: tamper + re-sign with a device key NOT in the room's grant
// table -> self-consistent signature, but the refold (capability plane)
// diverges from the bundle's claimed digest. Sig-check alone would miss this. ──
const forged = JSON.parse(JSON.stringify(bundle))
const forgeIdx = forged.ops.findIndex((o) => o.seq === 6 && o.actor === 'phone')
const forgedOp = { ...forged.ops[forgeIdx], body: 'yes, Tuesday EVENING actually', devicePub: FORGER.pubHex }
delete forgedOp.sig
const forgedDigest = createHash('sha256').update(signable(forgedOp)).digest()
forgedOp.sig = hcrypto.sign(forgedDigest, FORGER.secretKey).toString('hex')
forged.ops[forgeIdx] = forgedOp

const verdict4 = verifyTranscript(forged)
check('forge: the forged op\'s OWN signature checks out (self-consistent attacker)',
  verdict4.ops.find((v) => v.seq === 6)?.sigValid === true)
check('forge: yet the refold diverges — not caught by sig-check alone', verdict4.digestMatches === false)
check('forge: overall verdict is NOT verified', verdict4.verified === false)
check('forge: original bundle untouched by verification', JSON.stringify(bundle) === bundleSnapshotJSON)

// ── (5) A social room (unanchored, no authority) also verifies — Article V
// applies to the human layer most of all. ──
const social = await mk('social', 0x2b, { mode: 'room' }) // no authorityPub: unenforced, no admin in the room
const SOCIAL_OPS = [
  signOp({ seq: 1, actor: 'desk', ts: 100, kind: 'room.manifest', title: 'just us' }, DESK), // anchorType '' -> social
  signOp({ seq: 2, actor: 'phone', ts: 200, kind: 'msg.post', body: 'hey, you free to vent for a sec' }, PHONE),
  signOp({ seq: 3, actor: 'desk', ts: 300, kind: 'msg.post', body: 'always', replyTo: 'phone:2' }, DESK),
]
for (const op of SOCIAL_OPS) await social.append(op)
await social.base.update()
const socialBundle = await exportTranscript(social, { exportedBy: 'phone (own copy, own gesture)' })
check('social: authorityPub is null — no admin in the room, none in the export',
  socialBundle.authorityPub === null)
const verdict5 = verifyTranscript(socialBundle)
check('social: VERIFIED with no authority configured (unenforced fold still recomputes)',
  verdict5.verified === true && verdict5.allSigsValid === true && verdict5.digestMatches === true)

// ── (6) golden: pin stateDigest + sha256 of the canonical bundle JSON ──
const canonicalJSON = JSON.stringify(bundle)
const bundleSha256 = createHash('sha256').update(canonicalJSON).digest('hex')

if (UPDATE_GOLDEN || !existsSync(GOLDEN)) {
  writeFileSync(GOLDEN, JSON.stringify({ stateDigest: bundle.stateDigest, viewDigest: bundle.viewDigest, bundleSha256 }, null, 2) + '\n')
  console.log(`\n  (golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(GOLDEN, 'utf8'))
  check('golden: transcript state digest matches pinned golden', golden.stateDigest === bundle.stateDigest)
  check('golden: transcript view digest matches pinned golden', golden.viewDigest === bundle.viewDigest)
  check('golden: canonical bundle JSON sha256 matches pinned golden', golden.bundleSha256 === bundleSha256)
}

console.log(`\nstate digest:   ${bundle.stateDigest}`)
console.log(`view digest:    ${bundle.viewDigest}`)
console.log(`bundle sha256:  ${bundleSha256}`)

await hub.close()
await social.close()
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nTRANSCRIPT SPIKE GREEN ✅' : `\nTRANSCRIPT SPIKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
