// social-spike.mjs — "the vent": a full DM story on a hermetic testnet WITH
// the blind mirror (Constitution Art. II social rooms, Art. XI DM-mirror
// law, Art. III §6 / MSG-D21 read-cursor ban, Art. V safety law).
//
// ana creates a DM (participant authority — HER device key, not an org key;
// unanchored manifest), opens a real one-time M2 invite carrying the room's
// content key, and goes fully offline. bela — who was NEVER online at the
// same time as ana — wakes later, pulls the room through the always-on
// blind mirror, redeems the invite for real, and vents into the room alone.
// The office machine (the mirror) carries every byte of that vent and can
// read none of it (Art. XI). The next morning ana reconnects (her OWN
// storage reopened — the same device waking up, not a new identity) and
// converges to the exact same state bela already had, having never talked
// to bela directly either — both peers only ever spoke to the mirror.
//
// Then: self-serve evidence export from bela's own copy (Art. V §5),
// blocking (Art. V §2 — silence is structural, no op kind says "blocked"),
// and true deletion (Art. V §6 — physics, not an API: both sides forget).
//
// Run: npm run socialspike
import createTestnet from 'hyperdht/testnet.js'
import HyperDHT from 'hyperdht'
import BlindPeer from 'blind-peer'
import BlindPeering from 'blind-peering'
import Wakeup from 'protomux-wakeup'
import Corestore from 'corestore'
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { deviceKeys, signOp, inviteKeys, inviteRedeemOp } from './capability.mjs'
import { decodeInviteCode } from './invite-code.mjs'
import { createSocialRoom, openDmInvite, blockDevice } from './social-room.mjs'
import { exportTranscript } from './export-transcript.mjs'
import { verifyTranscript } from './verify-transcript.mjs'
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const GOLDEN = join(__dirname, '..', 'goldens', 'social_autobase.json')
const UPDATE_GOLDEN = process.argv.includes('--update-golden')

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  ✓ ${name}`)
  else { failures++; console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`) }
}

function withTimeout(promise, ms, label) {
  let timer
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error(`timeout: ${label}`)), ms)
  })
  return Promise.race([promise, timeout]).finally(() => clearTimeout(timer))
}

/** A genuinely independent raw-bytes pull off the mirror — bare Corestore,
 * no encryption option, forces a real block-0 request over a fresh
 * replication stream. GL-5 discipline: this is what proves bytes actually
 * crossed the wire, so the opacity check that follows can't be dismissed as
 * "never even connected" (the same pattern mirror-spike.mjs and
 * reissue-spike.mjs already established). */
async function rawProbe(tmp, mirror, coreKeyHex, tag) {
  const store = new Corestore(join(tmp, `probe-${tag}`), { primaryKey: Buffer.alloc(32, 0x4d), unsafe: true })
  const core = store.get(Buffer.from(coreKeyHex, 'hex'))
  await core.ready()
  const s1 = mirror.store.replicate(true)
  const s2 = store.replicate(false)
  s1.pipe(s2).pipe(s1)
  const block = await withTimeout(core.get(0), 15000, `raw probe ${tag} fetching block 0`).catch(() => null)
  s1.destroy(); s2.destroy()
  await store.close()
  return block
}

const PK = (b) => Buffer.alloc(32, b)

// Pinned identities → goldenable run.
const ANA = deviceKeys(Buffer.alloc(32, 0xa1))
const BELA = deviceKeys(Buffer.alloc(32, 0xb2))
const ROGUE = deviceKeys(Buffer.alloc(32, 0x99)) // canon rogue seed, reused per spike convention

// Synthetic fixture content key (canon rule, MSG-D13/MSG-D18 precedent).
const ROOM_KEY_BYTES = Buffer.alloc(32, 0x77)
const INVITE_SEED = Buffer.alloc(32, 0x5d)

console.log('Messenger — social spike: the vent, offline, through the blind mirror\n')

const tmp = mkdtempSync(join(tmpdir(), 'mesh-social-'))
const testnet = await createTestnet(3)
const bootstrap = testnet.bootstrap

// ── The always-on mirror (the office machine) ──
const mirror = new BlindPeer(join(tmp, 'mirror-rocks'), { bootstrap })
await mirror.ready()
await mirror.listen()
const MIRROR_KEY = mirror.publicKey
console.log(`  mirror up: ${MIRROR_KEY.toString('hex').slice(0, 16)}… (local testnet)\n`)

// ── Step 1: ana creates the DM room (participant authority, unanchored, encrypted) ──
const ana = await createSocialRoom({
  creatorKeys: ANA, storage: join(tmp, 'ana'), primaryKey: PK(0x1a),
  title: 'chai break ☕', encryptionKey: ROOM_KEY_BYTES, ts: 100, actor: 'ana',
})
const ROOM_KEY = ana.key
check('social room: authority is a PARTICIPANT key, not an org key', ana.authorityPub === ANA.pubHex)
const anaStateAfterManifest = await ana.state()
check('social room: manifest folds with NO anchor (Art. II — privacy is topology)',
  anaStateAfterManifest.manifest?.title === 'chai break ☕' && !anaStateAfterManifest.manifest?.anchorType)

const dmCode = await openDmInvite(ana, { creatorKeys: ANA, inviteSeed: INVITE_SEED, ts: 200, seq: 2, actor: 'ana' })
check('dm invite: code is versioned asymm-room2 (encrypted room, key rides the invite)', dmCode.startsWith('asymm-room2.'))
const decoded = decodeInviteCode(dmCode)
check('dm invite: decode recovers the room key, ana\'s device pub, and the content key',
  decoded.baseKey === ROOM_KEY && decoded.authorityPub === ANA.pubHex &&
  Buffer.compare(decoded.encryptionKey, ROOM_KEY_BYTES) === 0)

// bela's node object exists LOCALLY now — pure local Corestore/key derivation,
// zero network I/O — purely so ana can register bela's writer key below
// (Autobase plumbing: a device must be a registered WRITER before it can
// append anything at all, including its own invite.redeem — the same
// precondition every existing spike, room-spike through reissue-spike,
// already accepts). This is NOT "bela online": no replication, no DHT, no
// socket exists yet. bela stays fully offline until her own BlindPeering
// connects to the mirror, well after ana is closed below.
const bela = await createMeshNode({
  storage: join(tmp, 'bela'), primaryKey: PK(0x2b),
  bootstrap: decoded.baseKey, authorityPub: decoded.authorityPub, mode: 'room',
  encryptionKey: decoded.encryptionKey,
})
await ana.addWriter(bela.writerKey)

// ana's own opening line, before she goes to sleep.
const anaPost = signOp({ seq: 3, actor: 'ana', ts: 300, kind: 'msg.post', body: 'you up? been a day 🍃', expectation: 'whenever' }, ANA)
await ana.append(anaPost)
const ANA_RAW_BLOCKS = 4 // manifest, invite.offer, addWriter(pseudo), her post

// ── ana pushes to the mirror, then goes FULLY offline ──
const anaWakeup = new Wakeup()
const anaDht = new HyperDHT({ bootstrap })
const anaBlind = new BlindPeering(anaDht, ana.store, { wakeup: anaWakeup, keys: [MIRROR_KEY] })
await anaBlind.addAutobase(ana.base, { announce: true })

await waitFor(async () => {
  const core = mirror.store.get(Buffer.from(ROOM_KEY, 'hex'))
  await core.ready()
  const have = core.contiguousLength
  return have >= ANA_RAW_BLOCKS ? have : null
}, { label: 'mirror to hold ana\'s segment', timeout: 60000, interval: 250 })
check('push: the mirror holds ana\'s segment while she is still online', true)

await anaBlind.close()
await ana.close()
await anaDht.destroy()
check('the midnight vent: ana is fully offline (store, autobase, dht all closed)', true)

// ── GL-5, ana's segment: arrival AND opacity, asserted separately ──
const anaRaw = await rawProbe(tmp, mirror, ROOM_KEY, 'ana')
check('crypto probe (ana segment): the raw block genuinely arrived (bytes crossed the wire)', anaRaw !== null)
check('crypto probe (ana segment): the raw block does NOT contain her known plaintext (real ciphertext)',
  anaRaw !== null && !anaRaw.includes(Buffer.from('been a day')))

// ── Step 2/3: bela wakes — NEVER online at the same time as ana — pulls via
// the mirror, redeems FOR REAL, and vents ──
const belaWakeup = new Wakeup()
const belaDht = new HyperDHT({ bootstrap })
const belaBlind = new BlindPeering(belaDht, bela.store, { wakeup: belaWakeup, keys: [MIRROR_KEY] })
await belaBlind.addAutobase(bela.base, { announce: true })

await waitFor(async () => {
  const ops = await bela.ops()
  return ops.length >= 3 ? ops : null
}, { label: 'bela to receive ana\'s 3 ops via the mirror', timeout: 60000, interval: 250 })
await waitFor(async () => { await bela.base.update(); return bela.writable }, { label: 'bela writable (addWriter arrived via the mirror)' })
check('offline delivery: bela received ana\'s segment through the mirror alone (never connected to ana)', true)

const redeemReal = inviteRedeemOp({ seq: 4, actor: 'bela', ts: 10_000, inviteId: decoded.inviteId }, inviteKeys(decoded.inviteSeed), BELA)
await bela.append(redeemReal)
// A rogue, riding BELA's own writer core (any writer can append any signed
// VALUE — the capability/invite plane is a FOLD-time law, not a
// replication-time one; same pattern reissue-spike.mjs's smuggled forgery
// proves), tries the SAME already-consumed one-time invite.
const redeemExhausted = inviteRedeemOp({ seq: 5, actor: 'rogue', ts: 10_100, inviteId: decoded.inviteId }, inviteKeys(decoded.inviteSeed), ROGUE)
await bela.append(redeemExhausted)
const belaReply = signOp({ seq: 6, actor: 'bela', ts: 10_200, kind: 'msg.post', body: 'up now. long story, got the offer today', replyTo: 'ana:3', expectation: 'whenever' }, BELA)
await bela.append(belaReply)
// The voluntary wave (Art. III §2): the recipient's lightweight ack, built
// on msg.react — a signal given, never taken.
const belaWave = signOp({ seq: 7, actor: 'bela', ts: 10_300, kind: 'msg.react', msgId: 'ana:3', emoji: '👋', on: true }, BELA)
await bela.append(belaWave)
// Deliberately built to prove the fold refuses it (Art. III §6 / MSG-D21).
const belaReadAttempt = signOp({ seq: 8, actor: 'bela', ts: 10_400, kind: 'msg.read', upToActor: 'ana', upToSeq: 3 }, BELA)
await bela.append(belaReadAttempt)
// Deliberately built to prove the fold refuses it (Art. VI / MSG-D17: "claims
// are a work concept", never built by social-room.mjs itself).
const belaClaimAttempt = signOp({ seq: 9, actor: 'bela', ts: 10_500, kind: 'room.claim', assignee: 'bela' }, BELA)
await bela.append(belaClaimAttempt)
const BELA_RAW_BLOCKS = 6 // redeem, exhausted-redeem, reply, wave, read-attempt, claim-attempt

await waitFor(async () => {
  const core = mirror.store.get(Buffer.from(bela.writerKey, 'hex'))
  await core.ready()
  const have = core.contiguousLength
  return have >= BELA_RAW_BLOCKS ? have : null
}, { label: 'mirror to hold bela\'s segment', timeout: 60000, interval: 250 })

const belaState = await bela.state()
const belaView = await bela.viewDigest()
check('invite: bela\'s real redemption grants a writer role at the current epoch',
  belaState.grants?.[BELA.pubHex]?.role === 'writer')
check('invite: one-time exhaustion — the second (rogue) redeem of the SAME code is refused',
  belaState.rejected.some((r) => r.actor === 'rogue' && r.reason.includes('exhausted')))
check('uses ledger: the DM invite was consumed exactly once', belaState.invites?.['ana:2']?.uses === 1)
check('the voluntary wave lands', belaState.reactions?.['ana:3']?.['👋']?.bela === true)
check('Art. III §6 / MSG-D21: the attempted read cursor is SKIPPED with the fold\'s own words',
  belaState.skipped.some((s) => s.kind === 'msg.read' && s.reason === 'read cursors are not emitted in social rooms'))
check('MSG-D17: the attempted claim is SKIPPED — "claims are a work concept" — untouched by this mission',
  belaState.skipped.some((s) => s.kind === 'room.claim' && s.reason === 'claims are a work concept'))
check('the conversation itself folds (ana\'s opener + bela\'s reply, threaded)',
  belaState.messages.some((m) => m.msgId === 'ana:3') &&
  belaState.messages.some((m) => m.msgId === 'bela:6' && m.replyTo === 'ana:3'))

// ── GL-5, bela's segment: arrival AND opacity, asserted separately ──
const belaRaw = await rawProbe(tmp, mirror, bela.writerKey, 'bela')
check('crypto probe (bela segment): the raw block genuinely arrived (bytes crossed the wire)', belaRaw !== null)
check('crypto probe (bela segment): the raw block does NOT contain her known plaintext (real ciphertext)',
  belaRaw !== null && !belaRaw.includes(Buffer.from('long story')))

// ── Step 4: the next morning — ana's OWN storage reopens (the same device
// waking up, not a new identity), converges through the mirror alone, having
// never talked to bela directly either. ──
const anaMorningWakeup = new Wakeup()
const anaMorningDht = new HyperDHT({ bootstrap })
const anaMorning = await createMeshNode({
  storage: join(tmp, 'ana'), primaryKey: PK(0x1a), // SAME storage dir + key: this IS ana, reopened
  authorityPub: ANA.pubHex, mode: 'room', wakeup: anaMorningWakeup, encryptionKey: ROOM_KEY_BYTES,
})
const anaMorningBlind = new BlindPeering(anaMorningDht, anaMorning.store, { wakeup: anaMorningWakeup, keys: [MIRROR_KEY] })
await anaMorningBlind.addAutobase(anaMorning.base, { announce: true })

await waitFor(async () => {
  const ops = await anaMorning.ops()
  return ops.length >= 9 ? ops : null
}, { label: 'ana (morning) to converge on all 9 ops via the mirror', timeout: 60000, interval: 250 })

const anaMorningState = await anaMorning.state()
const anaMorningView = await anaMorning.viewDigest()
check('convergence: ana (never directly connected to bela) matches bela\'s STATE, byte-identical, mirror-mediated only',
  anaMorningState.digest === belaState.digest)
check('convergence: view digests agree too (this segment is a sequential handoff, not a concurrent fork)',
  anaMorningView === belaView)
check('ana reads the vent she slept through', anaMorningState.messages.some((m) => m.body.includes('long story')))

// ── Step 5: self-serve evidence export, from bela's OWN copy (Art. V §5) ──
const bundle = await exportTranscript(bela, { exportedBy: 'bela (own copy, no admin involved)' })
check('export: authorityPub in the bundle IS ana\'s device key — participant authority, no org anywhere',
  bundle.authorityPub === ANA.pubHex)
const verdict = verifyTranscript(bundle)
check('verify: VERIFIED — a social room verifies exactly like an anchored one (Art. V is not special-cased away)',
  verdict.verified === true && verdict.allSigsValid === true && verdict.digestMatches === true)

// ── Step 6: blocking (Art. V §2 — silent, structural) ──
await blockDevice(anaMorning, { authorityKeys: ANA, devicePub: BELA.pubHex, ts: 20_000, seq: 10, actor: 'ana' })
const stateAfterBlock = await anaMorning.state()
check('block: epoch bumped; the authority needs no self-grant (implicitly granted, capability.go)',
  stateAfterBlock.capEpoch === 1)
check('block: bela\'s grant is now stale at the new epoch (nobody re-issued her)',
  stateAfterBlock.grants?.[BELA.pubHex]?.epoch === 0)

const wire = bela.connect(anaMorning)
const belaPostBlockAttempt = signOp({ seq: 11, actor: 'bela', ts: 20_100, kind: 'msg.post', body: 'hello? anyone?' }, BELA)
await bela.append(belaPostBlockAttempt)
await waitFor(async () => (await anaMorning.ops()).length >= 11, { label: 'bela\'s post-block attempt to reach ana (morning)' })
const [belaAfterBlock, anaAfterBlock] = await Promise.all([bela.state(), anaMorning.state()])
check('block: bela\'s subsequent op REJECTS everywhere — on her OWN refold too, not just ana\'s',
  belaAfterBlock.rejected.some((r) => r.actor === 'bela' && r.seq === 11 && r.reason.includes('is stale')) &&
  anaAfterBlock.rejected.some((r) => r.actor === 'bela' && r.seq === 11 && r.reason.includes('is stale')))

// Silence is structural: scan every op kind that ever appeared. Nothing
// resembling a "blocked" signal exists anywhere in the vocabulary — the fold
// has no such op kind, and nothing here invents one.
const allOps = await anaMorning.ops()
const KNOWN_KINDS = new Set(['room.manifest', 'invite.offer', 'invite.redeem', 'msg.post', 'msg.react', 'msg.read', 'room.claim', 'cap.epoch', 'cap.grant'])
check('silence is structural: every op kind in the log is ordinary vocabulary, none of it says "blocked"',
  allOps.every((op) => KNOWN_KINDS.has(op.kind) && !/block/i.test(op.kind)))
wire()

// ── Step 7: true deletion — physics, not an API (Art. V §6) ──
const anaDir = join(tmp, 'ana')
const belaDir = join(tmp, 'bela')
await anaMorningBlind.close()
await anaMorning.close()
await anaMorningDht.destroy()
await belaBlind.close()
await bela.close()
await belaDht.destroy()
rmSync(anaDir, { recursive: true, force: true })
rmSync(belaDir, { recursive: true, force: true })
check('true deletion: ana\'s own copy is gone — she discarded it, nobody reached across the wire to do it for her',
  !existsSync(anaDir))
check('true deletion: bela\'s own copy is gone — same physics, her own gesture',
  !existsSync(belaDir))
// The mirror's copy is untouched by either deletion — it holds unreadable
// ciphertext forever now that both keyholders have forgotten K (MSG-D15: the
// mirror never held a grant or a key of its own, so it has no lever to act
// on its copy even if it wanted to "delete" anything).
const mirrorStillHasIt = mirror.store.get(Buffer.from(ROOM_KEY, 'hex'))
await mirrorStillHasIt.ready()
check('honest physics: the mirror\'s copy outlives both owners — unreachable, unreadable garbage, not deleted',
  mirrorStillHasIt.contiguousLength >= ANA_RAW_BLOCKS)

if (UPDATE_GOLDEN || !existsSync(GOLDEN)) {
  writeFileSync(GOLDEN, JSON.stringify({
    stateDigest: belaState.digest, opsHashed: anaAfterBlock.opsHashed, applied: anaAfterBlock.applied,
    skipped: anaAfterBlock.skipped.length, rejected: anaAfterBlock.rejected.length,
  }, null, 2) + '\n')
  console.log(`\n  (golden ${UPDATE_GOLDEN ? 'updated' : 'created'}: ${GOLDEN})`)
} else {
  const golden = JSON.parse(readFileSync(GOLDEN, 'utf8'))
  check('golden: converged STATE digest (pre-block) matches pinned golden', golden.stateDigest === belaState.digest)
  check('golden: final opsHashed/applied/skipped/rejected counts match pinned golden',
    golden.opsHashed === anaAfterBlock.opsHashed && golden.applied === anaAfterBlock.applied &&
    golden.skipped === anaAfterBlock.skipped.length && golden.rejected === anaAfterBlock.rejected.length)
}

console.log(`\nconverged state digest (pre-block): ${belaState.digest}`)
console.log(`final opsHashed=${anaAfterBlock.opsHashed} applied=${anaAfterBlock.applied} skipped=${anaAfterBlock.skipped.length} rejected=${anaAfterBlock.rejected.length}`)

await mirror.close()
await testnet.destroy()
try { rmSync(tmp, { recursive: true, force: true }) } catch {}

console.log(failures === 0 ? '\nSOCIAL SPIKE GREEN ✅' : `\nSOCIAL SPIKE RED ❌ (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
