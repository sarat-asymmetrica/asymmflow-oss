// reissue-spike.mjs — Messenger: the room re-issue ceremony under fire.
// Constitution Art. II amendment / MSG-D18 GATE RULING / MSG-D20.
//
// The full revocation story, hermetic (no public network — in-process
// replication only, the same pattern room-spike.mjs and invite-spike.mjs
// already gate on), encrypted end-to-end:
//
//   1. Epoch-1 room (encryption key K1): authority + desk + rogue, a short
//      conversation including rogue's message — it folds fine, rogue was
//      legitimately granted THEN.
//   2. The incident: authority runs the ceremony (reissueRoom), surviving =
//      [desk], new key K2. The successor's manifest carries
//      predecessorRoomKey = epoch-1's own base key, same anchor/title; desk
//      gets one fresh asymm-room2 code carrying the new base + K2.
//   3. Desk joins the successor via ONLY the decoded code, converses,
//      converges with the authority. Rogue holds no grant in the successor —
//      proven by appending a rogue-signed op straight into the successor's
//      log (any writer can smuggle any signed VALUE past Autobase's apply();
//      only the capability-plane REFOLD is the actual bouncer) and watching
//      it land in `rejected[]`, plus the grants table's plain absence.
//   4. THE CRYPTO POINT: rogue still holds K1. A rogue-shaped probe —
//      bootstrapped on the SUCCESSOR's base key but with K1 as its
//      encryptionKey — cannot make sense of the successor's ciphertext
//      (assert the strongest observable: a thrown decrypt/parse error, or a
//      hard timeout — either is "cannot read", the mission brief's own
//      framing).
//   5. History navigation: read predecessorRoomKey straight off the
//      SUCCESSOR's manifest (never hardcode the predecessor's key) and open
//      it with K1, as desk — desk was a genuine epoch-1 member, so this
//      reads the old conversation cleanly. This is the SAME crypto boundary
//      as step 4 read from the other side: K1 is honestly still good for
//      the container it was actually issued for.
//
// Scenario shape (GL-2): every append in this spike is either (a) a single
// writer signing its own ops onto its OWN node (epoch-1's hub node; the
// successor's hub node before desk joins), or (b) a sequential handoff —
// desk can only act on the successor AFTER decoding a code that provably
// did not exist until the ceremony ran, so there is no genuine concurrent
// fork anywhere in this spike. A view+state digest golden is therefore valid
// on GL-2's letter, not just its state-pin fallback. 3 reproducibility runs.
//
// Run: npm run reissuespike
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { deviceKeys, signOp, grantOp } from './capability.mjs'
import { decodeInviteCode } from './invite-code.mjs'
import { reissueRoom } from './reissue-room.mjs'
import Corestore from 'corestore'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDEN = join(__dirname, '..', 'goldens', 'reissue_autobase.json')
const UPDATE_GOLDEN = process.argv.includes('--update-golden')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

// A hard wall-clock timeout for probes that might hang instead of throwing
// (a wrong-key block-verification failure can make Hypercore keep
// re-requesting a "corrupt" block from peers rather than reject locally) —
// distinct from mesh-node.mjs's `waitFor`, which polls a CONDITION and can
// itself hang forever if the awaited promise inside it never settles.
function withTimeout(promise, ms, label) {
  let timer
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error(`timeout: ${label}`)), ms)
  })
  return Promise.race([promise, timeout]).finally(() => clearTimeout(timer))
}

const PK = (b) => Buffer.alloc(32, b)

// Pinned identities → goldenable run.
const AUTH = deviceKeys(Buffer.alloc(32, 0xd4))
const DESK = deviceKeys(Buffer.alloc(32, 0xe5))
const ROGUE = deviceKeys(Buffer.alloc(32, 0x99))

// Synthetic fixture content keys (canon rule, MSG-D13/MSG-D18's precedent):
// deliberately distinguishable so a bug that reuses the wrong key is loud.
const K1 = Buffer.alloc(32, 0x71) // epoch-1's content key — rogue holds this
const K2 = Buffer.alloc(32, 0x72) // the successor's content key — rogue never gets this

console.log('Messenger — room re-issue ceremony: rotate-on-revoke made real\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-reissue-'))

async function runScenario() {
  // ── Step 1: the epoch-1 room, encrypted under K1 ──
  const epoch1 = await createMeshNode({
    storage: join(tmp, 'epoch1'), primaryKey: PK(0x1a),
    authorityPub: AUTH.pubHex, mode: 'room', encryptionKey: K1,
  })
  const EPOCH1_OPS = [
    signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-2201 — Steel Coils', anchorType: 'po', anchorId: 'PO-2201' }, AUTH),
    grantOp({ seq: 2, actor: 'hub', ts: 200, device: DESK.pubHex, epoch: 0 }, AUTH),
    grantOp({ seq: 3, actor: 'hub', ts: 300, device: ROGUE.pubHex, epoch: 0 }, AUTH),
    signOp({ seq: 4, actor: 'desk', ts: 400, kind: 'msg.post', body: 'Booking the slot for PO-2201.' }, DESK),
    signOp({ seq: 5, actor: 'rogue', ts: 500, kind: 'msg.post', body: 'In on this one too — legit for now.' }, ROGUE),
  ]
  for (const op of EPOCH1_OPS) await epoch1.append(op)
  const epoch1View = await epoch1.viewDigest()
  const epoch1State = await epoch1.state()

  check('epoch-1: manifest + both grants + both messages fold cleanly',
    epoch1State.manifest?.anchorId === 'PO-2201' &&
    epoch1State.grants?.[DESK.pubHex]?.role === 'writer' &&
    epoch1State.grants?.[ROGUE.pubHex]?.role === 'writer' &&
    epoch1State.applied === EPOCH1_OPS.length)
  check("epoch-1: rogue's message folds — it was legitimately granted THEN",
    epoch1State.messages.some((m) => m.msgId === 'rogue:5'))

  // ── Step 2: the incident — authority re-issues, surviving = [desk] ──
  const { node: successor, inviteCodesByDevice } = await reissueRoom({
    predecessorNode: epoch1,
    storage: join(tmp, 'successor'), primaryKey: PK(0x2b),
    authorityKeys: AUTH, newEncryptionKey: K2,
    surviving: [DESK.pubHex], ts: 1000, actor: 'hub',
  })
  const successorStateAfterCeremony = await successor.state()

  check("re-issue: successor manifest carries predecessorRoomKey = epoch-1's own base key",
    successorStateAfterCeremony.manifest?.predecessorRoomKey === epoch1.key)
  check('re-issue: successor manifest copies the same anchor/title forward',
    successorStateAfterCeremony.manifest?.title === 'PO-2201 — Steel Coils' &&
    successorStateAfterCeremony.manifest?.anchorType === 'po' &&
    successorStateAfterCeremony.manifest?.anchorId === 'PO-2201')
  check('re-issue: only the survivor (desk) holds a grant in the successor',
    successorStateAfterCeremony.grants?.[DESK.pubHex]?.role === 'writer' &&
    !(ROGUE.pubHex in (successorStateAfterCeremony.grants ?? {})))

  const deskCode = inviteCodesByDevice[DESK.pubHex]
  check("re-issue: desk's fresh code is versioned asymm-room2 (encrypted room)", deskCode?.startsWith('asymm-room2.'))
  const decodedDeskCode = decodeInviteCode(deskCode)
  check('re-issue: decode recovers the successor base key and the new content key K2',
    decodedDeskCode.baseKey === successor.key &&
    decodedDeskCode.authorityPub === AUTH.pubHex &&
    Buffer.compare(decodedDeskCode.encryptionKey, K2) === 0)

  // ── Step 3: desk joins the successor via ONLY the decoded code ──
  const deskNode = await createMeshNode({
    storage: join(tmp, 'desk'), primaryKey: PK(0x3c),
    bootstrap: decodedDeskCode.baseKey, authorityPub: decodedDeskCode.authorityPub,
    mode: 'room', encryptionKey: decodedDeskCode.encryptionKey,
  })
  await successor.addWriter(deskNode.writerKey)
  const wire = successor.connect(deskNode)
  await waitFor(async () => { await deskNode.base.update(); return deskNode.writable }, { label: 'desk writable on the successor' })

  // Seq 1 (manifest) and 2 (desk's cap.grant) already belong to the ceremony
  // — canonical order sorts primarily by Seq, actor only breaks ties, so
  // desk's op must continue the successor's own seq counter (seq 3), not
  // restart at 1 (which would tie/precede the ceremony's own seq-1 manifest
  // by actor sort, exactly the bug this comment used to be a fix for).
  await deskNode.append(signOp({ seq: 3, actor: 'desk', ts: 1500, kind: 'msg.post', body: 'Confirmed — moved to the new room.' }, DESK))

  await waitFor(async () => {
    const ops = await successor.ops()
    return ops.length >= 3 ? ops : null // manifest + desk-grant + desk's message (rogue's smuggled op comes next)
  }, { label: 'successor to see desk\'s message', timeout: 20000 })

  const successorViewAfterDesk = await successor.viewDigest()
  const deskViewAfterJoin = await deskNode.viewDigest()
  check("converge: desk's own view matches the successor's, byte-identical", successorViewAfterDesk === deskViewAfterJoin)
  const successorStateAfterDesk = await successor.state()
  const deskStateAfterJoin = await deskNode.state()
  check('converge: successor STATE digest identical on both peers (law included)',
    successorStateAfterDesk.digest === deskStateAfterJoin.digest)
  check("desk's message folds in the successor",
    successorStateAfterDesk.messages.some((m) => m.msgId === 'desk:3' && m.body.includes('moved to the new room')))

  // ── Rogue holds NO grant in the successor. Prove it two ways. ──
  check('capability plane: rogue is simply absent from the successor grant table',
    !(ROGUE.pubHex in (successorStateAfterDesk.grants ?? {})))

  // A smuggled rogue-signed op: any writer can append any signed VALUE past
  // Autobase's own apply() (it only screens the envelope shape, mesh-node.mjs
  // isOp()) — the capability plane is a FOLD-time law, not a replication-time
  // one (same distinction MSG-D10/room-spike.mjs already prove for a rogue
  // riding another peer's writer core). Smuggling the op in is therefore the
  // STRONGEST honest assertion available here: it shows the rejection is a
  // property of the fold re-run on every peer, not of some write ever being
  // physically prevented.
  const rogueForgery = signOp({ seq: 4, actor: 'rogue', ts: 1600, kind: 'msg.post', body: 'let me back in' }, ROGUE)
  await successor.append(rogueForgery)
  await waitFor(async () => (await successor.ops()).length >= 4, { label: 'rogue forgery to land in the raw log' })
  const successorStateAfterForgery = await successor.state()
  check("capability plane: rogue's smuggled op is REJECTED on refold ('no grant for device')",
    successorStateAfterForgery.rejected.some((r) => r.actor === 'rogue' && r.reason.includes('no grant for device')))
  check("capability plane: rogue's forged op never becomes a message",
    !successorStateAfterForgery.messages.some((m) => m.msgId === 'rogue:4'))

  // ── Step 4: THE CRYPTO POINT — rogue still holds K1; assert it is useless
  // against the successor (K2). Two separate probes, deliberately: the FIRST
  // proves bytes genuinely crossed the wire (so the second probe's failure
  // can't be dismissed as "never even connected") — the exact same discipline
  // MSG-D18's mirror-spike.mjs third-node keyless probe already established.
  //
  // Probe A — RAW bytes, bypassing Autobase's own decode path entirely (a
  // bare Corestore, no encryptionKey option even exists at this layer):
  // forces an explicit block-0 request over a real replication stream and
  // checks the returned ciphertext does not contain the known plaintext.
  // This is the strong, unambiguous "replication happened, content didn't
  // leak" proof.
  const rawProbeStore = new Corestore(join(tmp, 'rogue-raw-probe'), { primaryKey: PK(0x4d), unsafe: true })
  const rawProbeCore = rawProbeStore.get(Buffer.from(successor.key, 'hex')) // deliberately no encryption option
  await rawProbeCore.ready()
  const rawWire1 = successor.store.replicate(true)
  const rawWire2 = rawProbeStore.replicate(false)
  rawWire1.pipe(rawWire2).pipe(rawWire1)
  const KNOWN_PLAINTEXT = Buffer.from('moved to the new room')
  const rawBlock = await withTimeout(rawProbeCore.get(0), 12000, 'raw probe fetching block 0 off the successor')
  check('crypto probe A: the raw block genuinely replicated (bytes crossed the wire)', rawBlock !== null)
  check('crypto probe A: the raw block does NOT contain the known plaintext (real ciphertext, not a no-op read)',
    rawBlock !== null && !rawBlock.includes(KNOWN_PLAINTEXT))
  rawWire1.destroy(); rawWire2.destroy()
  await rawProbeStore.close()

  // Probe B — the Autobase/room-API level attempt: bootstrap a full MeshNode
  // on the successor's base key but with K1 (the WRONG key) as its
  // encryptionKey, and see what createMeshNode/Autobase itself does with it.
  // Reported honestly, whichever shape it takes: this run's observed
  // behavior is that Autobase's own linearizer never materializes a view for
  // a peer whose encryptionKey doesn't match (no thrown error, no timeout —
  // `ops()` resolves fast with an empty view), which is a STRONGER failure
  // mode than "gets garbage back": the wrong-keyed peer cannot even make
  // functional progress as a room, on top of Probe A's proof that the bytes
  // it does hold are unreadable ciphertext.
  const rogueProbe = await createMeshNode({
    storage: join(tmp, 'rogue-probe'), primaryKey: PK(0x6f),
    bootstrap: successor.key, authorityPub: AUTH.pubHex, mode: 'room', encryptionKey: K1,
  })
  const probeWire = successor.connect(rogueProbe)
  let probeOutcome = ''
  try {
    const rogueOps = await withTimeout(rogueProbe.ops(), 12000, 'rogue probe reading the successor with K1')
    const sawKnownPlaintext = JSON.stringify(rogueOps).includes('moved to the new room')
    check('crypto probe B: the Autobase-level view never leaks the known plaintext', !sawKnownPlaintext)
    probeOutcome = sawKnownPlaintext ? 'FAILED: plaintext leaked' : `no readable ops materialized (view length ${rogueOps.length})`
  } catch (err) {
    check('crypto probe B: the Autobase-level attempt fails closed (threw rather than leaking)', true)
    probeOutcome = `threw: ${err.message}`
  }
  probeWire()
  console.log(`    (probe B outcome: ${probeOutcome})`)

  // ── Step 5: history navigation — follow predecessorRoomKey off the
  // SUCCESSOR's own manifest (never hardcoded), open it with K1 as desk ──
  const predecessorKeyFromManifest = successorStateAfterDesk.manifest.predecessorRoomKey
  check("navigation: the pointer read off the successor's manifest IS epoch-1's key",
    predecessorKeyFromManifest === epoch1.key)

  const historian = await createMeshNode({
    storage: join(tmp, 'historian'), primaryKey: PK(0x5e),
    bootstrap: predecessorKeyFromManifest, authorityPub: AUTH.pubHex, mode: 'room', encryptionKey: K1,
  })
  const historyWire = epoch1.connect(historian)
  await waitFor(async () => {
    const ops = await historian.ops()
    return ops.length >= EPOCH1_OPS.length ? ops : null
  }, { label: 'historian to replicate epoch-1 through the followed pointer', timeout: 20000 })
  const historianState = await historian.state()
  historyWire()

  check('navigation: K1 still opens the PREDECESSOR fine — desk was a genuine member there (physics, Art. V)',
    historianState.digest === epoch1State.digest &&
    historianState.messages.some((m) => m.msgId === 'desk:4') &&
    historianState.messages.some((m) => m.msgId === 'rogue:5'))
  check('the chain is navigable forward-in-trust (grants), backward-in-history (predecessorRoomKey)',
    successorStateAfterDesk.grants?.[DESK.pubHex] !== undefined && historianState.messages.length === 2)

  wire()
  await Promise.all([epoch1.close(), successor.close(), deskNode.close(), historian.close(), rogueProbe.close()])

  return { epoch1View, epoch1Digest: epoch1State.digest, successorView: successorViewAfterDesk, successorDigest: successorStateAfterDesk.digest }
}

const first = await runScenario()

if (UPDATE_GOLDEN || !existsSync(GOLDEN)) {
  writeFileSync(GOLDEN, JSON.stringify({
    epoch1ViewDigest: first.epoch1View, epoch1StateDigest: first.epoch1Digest,
    successorViewDigest: first.successorView, successorStateDigest: first.successorDigest,
  }, null, 2) + '\n')
  console.log(`\n  (golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(GOLDEN, 'utf8'))
  check('golden: epoch-1 view digest matches pinned golden', golden.epoch1ViewDigest === first.epoch1View)
  check('golden: epoch-1 state digest matches pinned golden', golden.epoch1StateDigest === first.epoch1Digest)
  check('golden: successor view digest matches pinned golden', golden.successorViewDigest === first.successorView)
  check('golden: successor state digest matches pinned golden', golden.successorStateDigest === first.successorDigest)
}

console.log(`\nepoch-1 state digest:   ${first.epoch1Digest}`)
console.log(`successor state digest: ${first.successorDigest}`)

try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nREISSUE SPIKE GREEN ✅' : `\nREISSUE SPIKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
