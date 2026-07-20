// bare-bridge-spike.mjs — Phase 2 gate: the same 30-ish assertions as
// bridge-server.mjs's own bridge-spike.mjs (Node/TCP line), ported to
// bare-bridge.mjs's dispatch-core + stdio-transport shape. Two runtimes
// (`node` / `npx bare`), same file, same assertions, same fold verdicts.
//
// THREE LAYERS OF PROOF IN THIS FILE, each testing something different:
//   1. `wireClient(core)` — every request/response round-trips through REAL
//      JSON.stringify/JSON.parse (simulating the wire encode/decode
//      protocol v0 actually uses) but calls `core.dispatch()` directly —
//      no real stdio. This proves METHOD BEHAVIOR + WIRE SHAPE (30
//      scenario checks below) without needing a second OS process.
//   2. `frameLoopCheck()` — feeds literal newline-delimited byte strings
//      (including a deliberately malformed line, a request split across
//      two `onData` chunks, and a zero-length line) through the REAL
//      `attachStdioTransport` ndjson loop, proving the actual buffering/
//      framing code path bare-bridge.mjs ships, not just the dispatch core.
//   3. `--stdio-worker` mode (see bottom) — when invoked with that flag,
//      this file becomes a REAL stdio worker (`runStdioWorker`, real
//      process.stdin/stdout), for driving from OUTSIDE this process via
//      real OS shell pipes — this is how the P0-D flush-race risk gets
//      empirically exercised (see PHASE2_BRIDGE_REPORT.md for the
//      transcript; child-process spawning from INSIDE this file would need
//      a `node:child_process`/Bare-subprocess import this file isn't
//      allowed to carry, so that exercise is driven from the shell, not
//      from JS).
//
// Scenario data (hub/desk identities, PO room, social room, invite
// ceremony) is the SAME canon bridge-spike.mjs uses — same pinned keys,
// same op bodies — so a diff between the two spikes' outcomes would be
// meaningful, not an artifact of different fixtures.
//
// Run: node host/bare-bridge-spike.mjs
//      npx bare host/bare-bridge-spike.mjs
//      (both, from OUTSIDE the repo tree too, per D5)

import { createBridgeCore, attachStdioTransport, runStdioWorker } from './bare-bridge.mjs'
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { deviceKeys, signOp, grantOp } from './capability.mjs'
import fsForFixture from '#fs'
// verify-transcript.mjs is NOT on this campaign's migration list and still
// imports `node:crypto` + `./apply.mjs` directly -- a SECOND cross-cutting
// blocker outside this phase's file fence (see PHASE2_BRIDGE_REPORT.md).
// Imported dynamically, guarded, so its own load failure under Bare
// degrades that ONE check to "skipped", rather than taking down every
// other scenario in this file at module-load time.
let verifyTranscript = null
try {
  ({ verifyTranscript } = await import('./verify-transcript.mjs'))
} catch (err) {
  // stderr, deliberately: in --stdio-worker mode, stdout IS the protocol
  // frame stream -- a diagnostic line here would corrupt it (found by
  // running the worker mode transcript for this very report and noticing
  // this line mixed into the ndjson output).
  console.error(`(verify-transcript.mjs unavailable under this runtime -- ${err.code ?? err.constructor.name} -- exportTranscript's verification check will be skipped)`)
}

const isBare = typeof Bare !== 'undefined'

// ── --stdio-worker mode: a real stdio worker, driven from the shell ──────
const argv = isBare ? Bare.argv : process.argv
if (argv.includes('--stdio-worker')) {
  const PK = (b) => Buffer.alloc(32, b)
  const HUB = deviceKeys(PK(0xd4))
  const worker = await runStdioWorker({
    rooms: new Map(), actor: 'hub', deviceKeys: HUB, storageDir: './.bare-bridge-worker-storage',
  })
  // Stays alive on the stdin listener already registered inside
  // runStdioWorker/getRealStdio — no explicit "block forever" needed, and
  // deliberately no proc.exit()/Bare.exit() call anywhere in this branch
  // (see bare-bridge.mjs's flush-race mitigation note).
  void worker
} else {
  await mainSpike()
}

async function mainSpike() {
  let failures = 0
  let checks = 0
  function check(name, cond, detail = '') {
    checks++
    if (cond) console.log(`  ✓ ${name}`)
    else { failures++; console.log(`  ✗ ${name}${detail ? ' -- ' + detail : ''}`) }
  }

  function waitForEvent(client, event, { timeout = 15000 } = {}) {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => { unsub(); reject(new Error(`timeout waiting for event ${event}`)) }, timeout)
      const unsub = client.on(event, (params) => { clearTimeout(timer); unsub(); resolve(params) })
    })
  }

  // A thin in-process "wire client": every call round-trips through REAL
  // JSON.stringify/parse (the actual wire encoding), then calls
  // core.dispatch() directly -- see file header layer 1.
  function wireClient(core) {
    let id = 0
    const listeners = new Map()
    core.onEvent((event, params) => {
      for (const cb of (listeners.get(event) ?? [])) cb(params)
    })
    return {
      async request(method, params) {
        id++
        const wireIn = JSON.parse(JSON.stringify({ id, method, params }))
        const res = JSON.parse(JSON.stringify(await core.dispatch(wireIn)))
        if (!res.ok) { const e = new Error(res.error); e.bridgeError = res.error; throw e }
        return res.result
      },
      on(event, cb) {
        if (!listeners.has(event)) listeners.set(event, new Set())
        listeners.get(event).add(cb)
        return () => listeners.get(event).delete(cb)
      },
    }
  }

  const PK = (b) => Buffer.alloc(32, b)
  const HUB = deviceKeys(PK(0xd4))
  const DESK = deviceKeys(PK(0xe5))
  const SOCIAL_KEY = Buffer.alloc(32, 0x77)
  const INVITE_SEED = Buffer.alloc(32, 0x5d)

  console.log(`bare-bridge-spike -- protocol v0 over the Bare stdio transport [runtime: ${isBare ? 'Bare' : 'Node'}]\n`)

  const tmp = `.bare-bridge-spike-${Date.now()}-${Math.floor(Math.random() * 1e6)}`

  // ── Anchored PO room, opened directly (not a v0 method), exactly like
  // bridge-spike.mjs's own setup ──
  const hubPo = await createMeshNode({ storage: `${tmp}/hub-po`, primaryKey: PK(0x1a), authorityPub: HUB.pubHex, mode: 'room' })
  const deskPo = await createMeshNode({ storage: `${tmp}/desk-po`, primaryKey: PK(0x2b), bootstrap: hubPo.key, authorityPub: HUB.pubHex, mode: 'room' })
  const poWire = hubPo.connect(deskPo)
  await hubPo.addWriter(deskPo.writerKey)
  await waitFor(async () => { await deskPo.base.update(); return deskPo.writable }, { label: 'desk writable in PO room' })

  await hubPo.append(signOp({ seq: 1, actor: 'hub', ts: 100, kind: 'room.manifest', title: 'PO-9001 -- Steel Coils', anchorType: 'po', anchorId: 'PO-9001' }, HUB))
  await hubPo.append(grantOp({ seq: 2, actor: 'hub', ts: 200, device: DESK.pubHex, epoch: 0 }, HUB))
  await waitFor(async () => (await deskPo.ops()).length >= 2, { label: 'desk sees manifest + grant' })

  // ── Two bridge cores, one per device ──
  let hubCore = createBridgeCore({ rooms: new Map([[hubPo.key, hubPo]]), actor: 'hub', deviceKeys: HUB, storageDir: `${tmp}/hub-store` })
  let deskCore = createBridgeCore({ rooms: new Map([[deskPo.key, deskPo]]), actor: 'desk', deviceKeys: DESK, storageDir: `${tmp}/desk-store` })
  let hubClient = wireClient(hubCore)
  let deskClient = wireClient(deskCore)

  // ── hello() ──
  const hubHello = await hubClient.request('hello')
  const deskHello = await deskClient.request('hello')
  check('hello: hub identity', hubHello.devicePub === HUB.pubHex && hubHello.actor === 'hub' && hubHello.version === 'v0')
  check('hello: desk identity', deskHello.devicePub === DESK.pubHex && deskHello.actor === 'desk' && deskHello.version === 'v0')

  // ── malformed frame resilience, via the REAL ndjson loop (layer 2's
  // `frameLoopCheck` covers the transport in isolation; this checks the
  // dispatch()-level equivalent: an unrecognized method / bad shape never
  // throws out of dispatch, always returns {ok:false}) ──
  const malformed = await hubCore.dispatch({ notAValid: 'frame' })
  check('malformed frame: dispatch() returns {ok:false} without throwing', malformed.ok === false)
  const stillAlive = await hubClient.request('hello')
  check('malformed frame: core survives and keeps serving valid requests', stillAlive.devicePub === HUB.pubHex)

  // ── listRooms() ──
  const hubRoomsInitial = await hubClient.request('listRooms')
  check('listRooms: hub sees the PO room as anchored', hubRoomsInitial.some((r) => r.roomKey === hubPo.key && r.kind === 'anchored' && r.anchorId === 'PO-9001'))

  // ── roomState() ──
  const poState0 = await hubClient.request('roomState', { roomKey: hubPo.key })
  check('roomState: manifest title', poState0.manifest?.title === 'PO-9001 -- Steel Coils')
  check('roomState: desk is a current member (writer, epoch 0)',
    poState0.members.some((m) => m.devicePub === DESK.pubHex && m.role === 'writer' && m.epoch === 0))
  check('roomState: capEpoch reported', poState0.capEpoch === 0)

  // ── post(): invalid expectation surfaces the fold's own skip reason ──
  const badTag = await hubClient.request('post', { roomKey: hubPo.key, body: 'oops', expectation: 'someday', ts: 300 }).catch((e) => e)
  check('post: an invalid expectation tag is a bridge ERROR carrying the fold\'s own skip reason',
    badTag instanceof Error && badTag.bridgeError === 'unknown expectation tag')

  // ── post(): valid tags, mixed urgency, from both devices ──
  await hubClient.request('post', { roomKey: hubPo.key, body: 'kickoff -- anything blocking?', expectation: 'whenever', ts: 400 })
  await deskClient.request('post', { roomKey: hubPo.key, body: 'need the spec sheet', expectation: 'today', ts: 500 })
  await waitFor(async () => (await hubPo.ops()).length >= 4, { label: 'hub sees desk\'s post' })

  let hubRooms = await hubClient.request('listRooms')
  check('listRooms: topExpectation is "today" before anything urgent lands',
    hubRooms.find((r) => r.roomKey === hubPo.key)?.topExpectation === 'today')

  const urgentPost = await deskClient.request('post', { roomKey: hubPo.key, body: 'stock arrives Thursday, need approval NOW', expectation: 'urgent', ts: 600 })
  check('post: urgent message accepted', Number.isInteger(urgentPost.seq))
  await waitFor(async () => (await hubPo.ops()).length >= 5, { label: 'hub sees the urgent post' })
  hubRooms = await hubClient.request('listRooms')
  const deskRooms = await deskClient.request('listRooms')
  check('listRooms: topExpectation floats to "urgent" on hub\'s core', hubRooms.find((r) => r.roomKey === hubPo.key)?.topExpectation === 'urgent')
  check('listRooms: topExpectation floats to "urgent" on desk\'s core too (same converged fold)', deskRooms.find((r) => r.roomKey === hubPo.key)?.topExpectation === 'urgent')

  // ── claimRoom() / releaseClaim() ──
  const claim = await deskClient.request('claimRoom', { roomKey: hubPo.key, assignee: 'desk', ts: 700 })
  check('claimRoom: desk self-claims the anchored room', Number.isInteger(claim.seq))
  const poState1 = await deskClient.request('roomState', { roomKey: hubPo.key })
  check('roomState: claim reflects desk as assignee', poState1.claim?.assignee === 'desk')
  const release = await deskClient.request('releaseClaim', { roomKey: hubPo.key, ts: 800 })
  check('releaseClaim: desk releases her own claim', Number.isInteger(release.seq))

  // ── attach() -> fetchAttachment(): sha256 verified end-to-end ──
  const DOC_BYTES = Buffer.alloc(4096)
  for (let i = 0; i < DOC_BYTES.length; i++) DOC_BYTES[i] = (i * 31 + 7) & 0xff
  const docPath = `${tmp}/PO-9001-spec.bin`
  fsForFixture.mkdirSync(tmp, { recursive: true })
  fsForFixture.writeFileSync(docPath, DOC_BYTES)
  const attachResult = await hubClient.request('attach', {
    roomKey: hubPo.key, filePath: docPath, body: 'spec sheet attached', expectation: 'today', ts: 900,
  })
  check('attach: returns seq, ref, sha256', Number.isInteger(attachResult.seq) && typeof attachResult.ref === 'string' && attachResult.sha256.length === 64)
  await waitFor(async () => (await deskPo.ops()).length >= 8, { label: 'desk sees the attach op' })
  const fetchedPath = `${tmp}/fetched-spec.bin`
  const fetched = await deskClient.request('fetchAttachment', { roomKey: hubPo.key, ref: attachResult.ref, savePath: fetchedPath })
  check('fetchAttachment: sha256 verified end-to-end', fetched.verified === true && fetched.sha256 === attachResult.sha256)
  check('fetchAttachment: bytes on disk are byte-identical to the original', fsForFixture.readFileSync(fetchedPath).equals(DOC_BYTES))
  const poStateWithAttachment = await hubClient.request('roomState', { roomKey: hubPo.key })
  const attachedMsg = poStateWithAttachment.messages.find((m) => m.attachment?.ref === attachResult.ref)
  check('roomState: attachment surfaces as {name,size,sha256,ref}',
    attachedMsg?.attachment?.name === 'PO-9001-spec.bin' && attachedMsg.attachment.size === DOC_BYTES.length && attachedMsg.attachment.sha256 === attachResult.sha256)

  // ── exportTranscript(): verified via the real verify-transcript machinery ──
  const bundle = await deskClient.request('exportTranscript', { roomKey: hubPo.key, exportedBy: 'desk (own copy, bare-bridge export)' })
  check('exportTranscript: format + roomKey', bundle.format === 'asymm-transcript.v1' && bundle.roomKey === hubPo.key)
  if (verifyTranscript) {
    const verdict = verifyTranscript(bundle)
    check('exportTranscript: VERIFIED end-to-end through verify-transcript.mjs', verdict.verified === true && verdict.allSigsValid === true && verdict.digestMatches === true)
  } else {
    console.log('  (skip) exportTranscript: VERIFIED end-to-end through verify-transcript.mjs -- module unavailable under this runtime')
  }

  // ── room-updated events: self-observed AND replication-delivered ──
  const hubSelfUpdate = waitForEvent(hubClient, 'room-updated')
  await hubClient.request('post', { roomKey: hubPo.key, body: 'self-observed update check', expectation: '', ts: 1000 })
  const selfEvt = await hubSelfUpdate
  check('event: hub\'s OWN post emits room-updated on hub\'s own core', selfEvt.roomKey === hubPo.key)

  const deskReplicatedUpdate = waitForEvent(deskClient, 'room-updated')
  await hubClient.request('post', { roomKey: hubPo.key, body: 'replicated update check', expectation: '', ts: 1100 })
  const replEvt = await deskReplicatedUpdate
  check('event: hub\'s post, once replicated, emits room-updated on desk\'s core too', replEvt.roomKey === hubPo.key)

  // ── GL-5: seq continues across a bridge-core restart, never resets to 1 ──
  const beforeRestartOps = await hubPo.ops()
  const maxSeqBefore = Math.max(...beforeRestartOps.map((o) => o.seq))
  hubCore.close()
  hubCore = createBridgeCore({ rooms: new Map([[hubPo.key, hubPo]]), actor: 'hub', deviceKeys: HUB, storageDir: `${tmp}/hub-store` })
  hubClient = wireClient(hubCore)
  const afterRestart = await hubClient.request('post', { roomKey: hubPo.key, body: 'after a bridge-core restart', expectation: '', ts: 1200 })
  check('GL-5: seq continues after a bridge-core restart (never restarts at 1)', afterRestart.seq > maxSeqBefore)

  // ── Social room: createSocialRoom + a REAL invite redeem under encryption ──
  const socialCreated = await hubClient.request('createSocialRoom', { title: 'water cooler', encryptionKey: SOCIAL_KEY.toString('hex'), ts: 1300 })
  const socialRoomKey = socialCreated.roomKey
  check('createSocialRoom: room created, returns roomKey', typeof socialRoomKey === 'string' && socialRoomKey.length > 0)
  const socialListed = await hubClient.request('listRooms')
  check('listRooms: the social room reports kind "social" (no anchor)', socialListed.find((r) => r.roomKey === socialRoomKey)?.kind === 'social')

  const dmInvite = await hubClient.request('openDmInvite', { roomKey: socialRoomKey, ts: 1400, inviteSeed: INVITE_SEED.toString('hex') })
  check('openDmInvite: code is versioned asymm-room2 (encrypted room, key rides the invite)', dmInvite.inviteCode.startsWith('asymm-room2.'))

  const hubSocial = hubCore.rooms.get(socialRoomKey)
  const deskSocial = await createMeshNode({
    storage: `${tmp}/desk-social`, primaryKey: PK(0x9d),
    bootstrap: hubSocial.key, authorityPub: hubSocial.authorityPub, mode: 'room',
    encryptionKey: hubSocial.encryptionKey,
  })
  deskCore.registerRoom(deskSocial.key, deskSocial)
  const socialWire = hubSocial.connect(deskSocial)
  await hubSocial.addWriter(deskSocial.writerKey)
  await waitFor(async () => { await deskSocial.base.update(); return deskSocial.writable }, { label: 'desk writable in social room' })

  const redeemed = await deskClient.request('redeemInvite', { inviteCode: dmInvite.inviteCode, actor: 'desk', ts: 1500 })
  check('redeemInvite: the REAL invite.redeem lands, room reachable by key', redeemed.roomKey === socialRoomKey)

  await deskClient.request('post', { roomKey: socialRoomKey, body: 'joined via a real invite, over the bare bridge', expectation: 'whenever', ts: 1600 })
  await waitFor(async () => (await hubSocial.ops()).length >= 4, { label: 'hub sees desk\'s social-room post' })
  const socialStateOnHub = await hubClient.request('roomState', { roomKey: socialRoomKey })
  check('roomState: desk\'s message converged onto hub\'s core for the social room',
    socialStateOnHub.messages.some((m) => m.body === 'joined via a real invite, over the bare bridge'))

  // ── The claim-skip proof in a social room (Art. VI / MSG-D17) ──
  const socialClaim = await deskClient.request('claimRoom', { roomKey: socialRoomKey, assignee: 'desk', ts: 1700 }).catch((e) => e)
  check('claimRoom in a social room: the fold\'s "claims are a work concept" skip surfaces through the core verbatim',
    socialClaim instanceof Error && socialClaim.bridgeError === 'claims are a work concept')

  // ── Layer 2: the REAL ndjson transport loop, driven with literal bytes ──
  await frameLoopCheck(check)

  // ── Layer 3: the REAL production topology -- child_process.spawn, parent
  // reading stdout.on('data') -- not a shell pipe, not in-process. This is
  // the leg that actually matters (PHASE0_NOTES_D2_FLUSH_RACE.md: the shell
  // pipe this file used for its first flush-race investigation HID both
  // BUG A and BUG B for a full day of this campaign; only the real spawn
  // topology surfaced them). See spawnPipeCheck's own header for why this
  // uses `node:child_process` despite the "no node: specifier" rule.
  await spawnPipeCheck(check)

  console.log('\nAll scenarios exercised through the Bare-native protocol v0 dispatch core.')

  socialWire()
  poWire()
  hubCore.close()
  deskCore.close()
  await Promise.all([hubPo.close(), deskPo.close(), hubSocial.close(), deskSocial.close()])
  try { fsForFixture.rmSync(tmp, { recursive: true, force: true }) } catch { /* best-effort */ }

  console.log(`\n${checks} check(s), ${failures} failure(s).`)
  console.log(failures === 0 ? '\nBARE BRIDGE SPIKE GREEN' : `\nBARE BRIDGE SPIKE RED (${failures} failure(s))`)
  const exitCode = failures === 0 ? 0 : 1
  if (isBare) Bare.exit(exitCode)
  else process.exit(exitCode)
}

// Layer 2 (file header): the REAL attachStdioTransport ndjson loop, fed
// literal byte chunks -- including a request split mid-line across two
// `onData` calls (the exact shape a real OS pipe delivers under
// backpressure) and one deliberately malformed line -- proving the
// TRANSPORT code, not just the dispatch core.
async function frameLoopCheck(check) {
  const PK = (b) => Buffer.alloc(32, b)
  const keys = { pubHex: 'aa'.repeat(32) }
  const core = createBridgeCore({ rooms: new Map(), actor: 'frametest', deviceKeys: keys })
  const written = []
  const io = {
    onData(cb) { io._cb = cb; return () => { io._cb = null } },
    write(str) { written.push(str); return true },
  }
  const transport = attachStdioTransport({ core, io })

  // A request split across two chunks -- proves the buffer correctly
  // waits for the newline before parsing.
  const fullLine = JSON.stringify({ id: 1, method: 'hello', params: {} }) + '\n'
  const splitAt = Math.floor(fullLine.length / 2)
  io._cb(fullLine.slice(0, splitAt))
  check('frame loop: no response yet from a split (incomplete) line', written.length === 0)
  io._cb(fullLine.slice(splitAt))
  await new Promise((r) => setTimeout(r, 20)) // dispatch() is async
  check('frame loop: split line completes and dispatches once joined', written.length === 1)
  const parsed1 = JSON.parse(written[0])
  check('frame loop: response carries the request id and devicePub', parsed1.id === 1 && parsed1.result?.devicePub === keys.pubHex)

  // A malformed line -- must not crash the loop, must respond {ok:false}.
  io._cb('this is not json at all\n')
  await new Promise((r) => setTimeout(r, 20))
  check('frame loop: malformed line gets {ok:false}, loop survives', written.length === 2 && JSON.parse(written[1]).ok === false)

  // A zero-length line (bare "\n") -- must be silently skipped, not error.
  io._cb('\n')
  await new Promise((r) => setTimeout(r, 20))
  check('frame loop: a bare blank line is silently skipped (no extra response)', written.length === 2)

  // Two complete requests arriving in ONE chunk -- both must dispatch.
  const two = JSON.stringify({ id: 2, method: 'hello', params: {} }) + '\n' + JSON.stringify({ id: 3, method: 'hello', params: {} }) + '\n'
  io._cb(two)
  await new Promise((r) => setTimeout(r, 20))
  check('frame loop: two requests in one chunk both dispatch', written.length === 4)

  check('frame loop: no partial bytes left buffered after all complete lines processed', transport.pendingPartial() === '')

  transport.close()
  core.close()
}

// Layer 3 (file header): the REAL spawn+pipe topology. Deliberately
// `node:child_process` -- the ONE exception to this campaign's "no node:
// specifier in your files" rule, scoped and justified: this function only
// ever runs from the Node-side invocation of a file that is never packed
// into the sealed artifact (spike/test tooling, exactly like every other
// *-spike.mjs in mesh/host -- none of them ship either). bare-bridge.mjs
// itself (the file that DOES ship) has no such import and never will.
// Requested explicitly by the campaign lead after the shell-pipe method
// this file's earlier flush-race work used was shown to hide both real
// bugs (PHASE0_NOTES_D2_FLUSH_RACE.md) -- a spawn+pipe leg is the only
// topology that actually proves anything about production behavior.
async function spawnPipeCheck(check) {
  if (isBare) {
    console.log('  (skip) spawn-pipe leg only runs from the Node-side invocation (child_process is Node-only, by design -- see this function\'s header)')
    return
  }
  const { spawn } = await import('node:child_process')
  // Absolute paths + an explicit cwd, deliberately -- this leg must survive
  // being invoked from a hostile CWD (D5: bare-bridge-spike.mjs itself is
  // run by absolute path from outside the repo tree in this campaign's
  // gates), and `npx`'s own `bare` resolution needs mesh/'s node_modules,
  // which only a correct `cwd` guarantees regardless of the parent's own
  // working directory.
  const meshDir = new URL('..', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
  const selfScript = new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
  const stdioCheckScript = new URL('./bare-spike/stdio-check.mjs', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')

  function runWorker(cmd, args, requests, { timeoutMs = 20000 } = {}) {
    return new Promise((resolvePromise) => {
      // shell:true is needed on win32 to resolve `npx` (a .cmd shim), but
      // it BREAKS a direct `process.execPath` spawn -- cmd.exe re-splits an
      // unquoted path on spaces ("C:\Program Files\nodejs\node.exe" becomes
      // several tokens), found by running this exact check. Only shell
      // when the command isn't already a resolved absolute executable path.
      const needsShell = process.platform === 'win32' && cmd !== process.execPath
      const child = spawn(cmd, args, { stdio: ['pipe', 'pipe', 'pipe'], shell: needsShell, cwd: meshDir })
      let out = ''
      let err = ''
      let timedOut = false
      const timer = setTimeout(() => { timedOut = true; child.kill() }, timeoutMs)
      child.stdout.on('data', (d) => { out += d.toString('utf8') })
      child.stderr.on('data', (d) => { err += d.toString('utf8') })
      child.on('exit', (code) => { clearTimeout(timer); resolvePromise({ exitCode: code, out, err, timedOut }) })
      child.on('error', (spawnErr) => { clearTimeout(timer); resolvePromise({ exitCode: null, out, err: `${err}\nspawn error: ${spawnErr.message}`, timedOut }) })
      for (const req of requests) child.stdin.write(JSON.stringify(req) + '\n')
      child.stdin.end()
    })
  }

  const requests = [
    { id: 1, method: 'hello', params: {} },
    { id: 2, method: 'listRooms', params: {} },
    { id: 3, method: 'createSocialRoom', params: { title: 'spawn-check' } },
  ]

  // ── Positive: our own worker, both runtimes, real spawn+pipe ──
  const nodeResult = await runWorker(process.execPath, [selfScript, '--stdio-worker'], requests)
  const nodeLines = nodeResult.out.split('\n').filter((l) => l.trim())
  check('spawn(node): worker responded to all 3 requests over a real spawned pipe', nodeLines.length === requests.length,
    `got ${nodeLines.length} line(s): ${JSON.stringify(nodeLines)} stderr=${nodeResult.err.slice(0, 300)}`)
  check('spawn(node): every response id matches a sent request, none missing',
    requests.every((r) => nodeLines.some((l) => { try { return JSON.parse(l).id === r.id } catch { return false } })))
  check('spawn(node): worker exited cleanly on stdin end (RULE 3\'s drain-then-exit), no hang', nodeResult.exitCode === 0 && !nodeResult.timedOut)

  const bareResult = await runWorker('npx', ['bare', selfScript, '--stdio-worker'], requests)
  const bareLines = bareResult.out.split('\n').filter((l) => l.trim())
  check('spawn(bare): worker responded to all 3 requests over a real spawned pipe', bareLines.length === requests.length,
    `got ${bareLines.length} line(s): ${JSON.stringify(bareLines)} stderr=${bareResult.err.slice(0, 300)}`)
  check('spawn(bare): every response id matches a sent request, none missing',
    requests.every((r) => bareLines.some((l) => { try { return JSON.parse(l).id === r.id } catch { return false } })))
  check('spawn(bare): worker exited cleanly on stdin end (RULE 3\'s drain-then-exit), no hang', bareResult.exitCode === 0 && !bareResult.timedOut)

  // ── Negative control: the campaign's OWN known-broken script
  // (host/bare-spike/stdio-check.mjs, Bug B -- writes via bare-process's
  // `process.stdout.write()`) -- proves THIS harness can go RED on a real
  // failure, not just report green (the lead's explicit ask, after
  // stdio-check.mjs itself passed for a day under the wrong test method). ──
  const brokenResult = await runWorker('npx', ['bare', stdioCheckScript], [{ id: 99, method: 'hello', params: {} }], { timeoutMs: 10000 })
  const brokenLines = brokenResult.out.split('\n').filter((l) => l.trim())
  // Expected healthy shape: 2 lines ({"event":"ready"} + the echoed line).
  // Bug B's signature (per the lead's report): the ready line arrives, the
  // echo does not -- so 1 line, not 2, with a clean exit. A hang (timedOut)
  // is also an acceptable "caught it" outcome, in case this run's timing
  // hits the deadlock shape instead of the silent-drop shape.
  const caughtTheKnownBug = brokenLines.length < 2 || brokenResult.timedOut
  check('negative control: this harness correctly flags host/bare-spike/stdio-check.mjs as broken on a real spawn',
    caughtTheKnownBug,
    `got ${brokenLines.length} line(s) (expected 2 if healthy), timedOut=${brokenResult.timedOut}: ${JSON.stringify(brokenLines)}`)
  console.log(`  (negative control detail: stdio-check.mjs produced ${JSON.stringify(brokenLines)}, exitCode=${brokenResult.exitCode}, timedOut=${brokenResult.timedOut})`)
}
