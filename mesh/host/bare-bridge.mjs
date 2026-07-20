// bare-bridge.mjs — Phase 2 (Bare-runtime campaign): protocol v0
// (mesh/docs/MESSENGER_UI_CAMPAIGN.md §1, the DP4 seam) over ndjson-framed
// STDIO, Bare-native. This is the sidecar shape bridge-server.mjs's own
// header names as the eventual successor to its dev-mode TCP transport —
// same frames (`{id,method,params}` in, `{id,ok,result|error}` out,
// `{event,params}` unsolicited), same methods, same fold-verdict-verbatim
// discipline. bridge-server.mjs itself is frozen (Node-only, `node:net`
// et al) and is NOT imported here — it cannot be, under Bare, without
// breaking on load. This file re-derives its dispatch logic from the SAME
// underlying host modules bridge-server.mjs wraps (mesh-node.mjs,
// capability.mjs, social-room.mjs, invite-code.mjs, attachments.mjs,
// export-transcript.mjs) — reused, not reimplemented; chat law (expectation
// validation, claim rules, capability rejections) is the fold's alone, this
// file only appends signed ops and reads back skipped[]/rejected[] verbatim,
// exactly like bridge-server.mjs's own header mandates.
//
// THE FOUR RATIFIED DEVIATIONS from MESSENGER_UI_CAMPAIGN.md §1's literal
// text (bridge-server.mjs's own header, GL-1 pattern) are preserved exactly,
// unchanged, here:
//   1. `rooms` is a live Map<roomKey, MeshNode>, not a single `node` param.
//   2. Becoming an Autobase WRITER is host-side setup (`node.addWriter(...)`
//      via `rooms.get(roomKey)`), not a wire method.
//   3. Every op-building method accepts an optional `ts` (and
//      `openDmInvite` an optional `inviteSeed`) for deterministic spike
//      drive, defaulting to `Date.now()`.
//   4. `createSocialRoom({title})` -> `{roomKey}`; `openDmInvite(roomKey)`
//      -> `{inviteCode}` — minting an invite is a distinct gesture.
// GL-5 seq discipline is preserved verbatim too: `takeSeq` seeds from the
// GLOBAL max Seq across every actor's ops in the room, never a per-actor
// counter, never restarted at 1 — see bridge-server.mjs's header for the
// bug this discipline fixes; re-introducing a per-actor counter here would
// be a real regression, not a style choice.
//
// STRUCTURE — dispatch core, transport, and stdio are three separate
// layers on purpose (bridge-server.mjs fuses core+transport inside
// `net.createServer`'s callback; this file can't, both because the
// transport differs and because DI needs the core testable without any
// I/O at all):
//   createBridgeCore({...})   -> { rooms, registerRoom, dispatch(msg),
//                                  onEvent(cb), close() }  -- zero I/O,
//                                  pure function calls, testable in-process.
//   attachStdioTransport({core, io}) -> { close() } -- ndjson framing over
//                                  an INJECTED io ({onData(cb)->unsub,
//                                  write(str)}), decoupled from where those
//                                  bytes actually come from.
//   getRealStdio()            -> the real io: `process` (Node's ambient
//                                  global, present with NO import) or
//                                  `bare-process` (dynamically imported —
//                                  a real npm package specifier, NOT
//                                  `node:`-prefixed, so bare-pack's static
//                                  traverser has something real to resolve
//                                  either way it looks).
//
// Import discipline (binding, per the campaign's Phase 2 brief): NO `node:`
// specifier anywhere in this file, static or dynamic. `fs` comes through
// the `#fs` package.json condition map (`{"bare":"bare-fs","default":"fs"}`,
// landed by the concurrent packaging migration — PHASE0_GATE_B3_CONDITION_
// MAP.md's recipe). Randomness (storage-dir uniqueness, invite seeds) comes
// from `hypercore-crypto`'s `randomBytes` — already a direct dependency of
// capability.mjs, already proven Bare-clean (BARE_SPIKE_REPORT.md's 11/11),
// not a `node:crypto` substitute that needed inventing. Path joining is
// hand-rolled (three trivial string functions below) rather than pulling
// in a `#path` alias for something this small.
//
// THE STDIO HAZARDS (PHASE0_NOTES_D2_FLUSH_RACE.md — the load-bearing doc
// for this whole file; read it before touching anything below). TWO
// DISTINCT bugs were root-caused on the REAL production topology (a
// spawned child, parent reading `stdout.on('data')` — not a shell pipe,
// which hid both of these for a full day of this campaign):
//   BUG A: `await WebAssembly.compile()`/`instantiate()` silently drops
//     stdout (measured 27-53% loss depending on exact pipe shape), exit
//     code 0, no error. RULE 1: only the SYNCHRONOUS `new
//     WebAssembly.Module()`/`new WebAssembly.Instance()` forms are used
//     anywhere in the reducer channel this bridge calls into
//     (apply-bare.mjs already complies, and documents it).
//   BUG B: `bare-process`'s `process.stdout.write()` HANGS on a real
//     spawned pipe (measured 30/30), with NO wasm involved at all —
//     independent of Bug A. RULE 2: this file's real stdio writer
//     (`getRealStdio`) uses `console.log`, never `proc.stdout.write()`.
// RULE 3: an explicit exit call on stdin 'end' is load-bearing (removing
// it hangs 10/10) — see `runStdioWorker`.
// See `runStdioWorker`'s own comment for the full rule-by-rule account and
// PHASE2_BRIDGE_REPORT.md for what was independently re-verified here
// against THIS bridge's real workload (a real `child_process.spawn` leg in
// bare-bridge-spike.mjs, not just an in-process or shell-pipe proxy for
// one — see that file's own header for why the distinction matters).

import { createMeshNode, waitFor } from './mesh-node.mjs'
import { signOp, inviteKeys, inviteOfferOp, inviteRedeemOp } from './capability.mjs'
import { createSocialRoom as createSocialRoomNode } from './social-room.mjs'
import { decodeInviteCode, encodeInviteCode } from './invite-code.mjs'
import { openBlobStore, putAttachment, getAttachment } from './attachments.mjs'
import { exportTranscript as exportTranscriptBundle } from './export-transcript.mjs'
import fs from '#fs'
import hcrypto from 'hypercore-crypto'

const PROTOCOL_VERSION = 'v0'
const URGENCY_RANK = { urgent: 2, today: 1, whenever: 0, '': 0 }

function seqOfMsgId(msgId) {
  const i = typeof msgId === 'string' ? msgId.lastIndexOf(':') : -1
  return i === -1 ? 0 : Number(msgId.slice(i + 1))
}

function attachmentView(refString) {
  if (!refString) return null
  try {
    const ref = JSON.parse(refString)
    return { name: ref.name, size: ref.byteLength, sha256: ref.sha256, ref: refString }
  } catch {
    return null
  }
}

function randomHex(n) { return hcrypto.randomBytes(n).toString('hex') }

// Minimal, hand-rolled path helpers — the only three operations this file
// needs (build a storage sub-path, find a path's directory, find its
// leaf name). Not a general-purpose `path` replacement; documented
// simplifications, not a claim of full node:path parity.
function joinPath(...parts) { return parts.filter((p) => p !== '' && p != null).join('/').replace(/\/{2,}/g, '/') }
function dirnameOf(p) { const i = Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\')); return i <= 0 ? '.' : p.slice(0, i) }
function baseNameOf(p) { return p.split(/[\\/]/).pop() }
function looksLikeDirPath(p) { return p.endsWith('/') || p.endsWith('\\') }

/**
 * createBridgeCore({ rooms, actor, deviceKeys, storageDir })
 * Same parameter shape as bridge-server.mjs's createBridgeServer, minus
 * `port`/`socket` (transport concerns, not this layer's job).
 * Returns { rooms, registerRoom, dispatch(msg) -> response, onEvent(cb) ->
 * unsub, close() }.
 */
export function createBridgeCore({ rooms = new Map(), actor, deviceKeys: keys, storageDir } = {}) {
  if (!actor) throw new Error('createBridgeCore requires actor')
  if (!keys || !keys.pubHex) throw new Error('createBridgeCore requires deviceKeys (capability.mjs deviceKeys() shape)')

  const seqCounters = new Map()
  const roomUnsubs = new Map()
  const eventListeners = new Set()

  async function nextSeqFor(node) {
    const ops = await node.ops()
    let max = 0
    for (const op of ops) {
      if (Number.isInteger(op.seq) && op.seq > max) max = op.seq
    }
    return max + 1
  }

  async function takeSeq(node) {
    const key = node.key
    if (!seqCounters.has(key)) seqCounters.set(key, await nextSeqFor(node))
    const seq = seqCounters.get(key)
    seqCounters.set(key, seq + 1)
    return seq
  }

  function emit(event, params) {
    for (const cb of eventListeners) {
      try { cb(event, params) } catch { /* a listener's own failure never breaks the others */ }
    }
  }

  function watchRoom(roomKey, node) {
    if (roomUnsubs.has(roomKey)) return
    const onUpdate = () => emit('room-updated', { roomKey })
    node.base.on('update', onUpdate)
    roomUnsubs.set(roomKey, () => node.base.off('update', onUpdate))
  }

  function registerRoom(roomKey, node) {
    rooms.set(roomKey, node)
    watchRoom(roomKey, node)
  }
  for (const [roomKey, node] of rooms) watchRoom(roomKey, node)

  function getRoom(roomKey) {
    const node = rooms.get(roomKey)
    if (!node) throw new Error(`unknown room ${JSON.stringify(roomKey)}`)
    return node
  }

  async function appendAndVerify(node, op) {
    await node.append(op)
    const state = await node.state()
    const skip = (state.skipped ?? []).find((s) => s.seq === op.seq && s.actor === op.actor && s.kind === op.kind)
    if (skip) return { ok: false, error: skip.reason }
    const rej = (state.rejected ?? []).find((r) => r.seq === op.seq && r.actor === op.actor && (!r.kind || r.kind === op.kind))
    if (rej) return { ok: false, error: rej.reason }
    return { ok: true, result: { seq: op.seq } }
  }

  function membersOf(state) {
    if (!state.grants) return []
    const epoch = state.capEpoch ?? 0
    return Object.entries(state.grants)
      .filter(([, g]) => g.epoch === epoch)
      .map(([devicePub, g]) => ({ devicePub, role: g.role, epoch: g.epoch }))
      .sort((a, b) => a.devicePub.localeCompare(b.devicePub))
  }

  async function roomSummary(roomKey, node) {
    const state = await node.state()
    const manifest = state.manifest ?? null
    const kind = manifest?.anchorType ? 'anchored' : 'social'
    const live = (state.messages ?? []).filter((m) => !m.deleted)
    const last = live[live.length - 1] ?? null

    let topExpectation = ''
    let topRank = -1
    for (const m of live) {
      const r = URGENCY_RANK[m.expectation ?? ''] ?? 0
      if (r > topRank) { topRank = r; topExpectation = m.expectation ?? '' }
    }

    return {
      roomKey,
      title: manifest?.title ?? '',
      kind,
      anchorType: manifest?.anchorType ?? '',
      anchorId: manifest?.anchorId ?? '',
      lastSeq: last ? seqOfMsgId(last.msgId) : 0,
      lastTs: last ? last.ts : (manifest?.ts ?? 0),
      lastPreview: last ? (last.body ?? '') : '',
      topExpectation,
    }
  }

  const methods = {
    async hello() {
      return { ok: true, result: { devicePub: keys.pubHex, actor, version: PROTOCOL_VERSION } }
    },

    async listRooms() {
      const out = []
      for (const [roomKey, node] of rooms) out.push(await roomSummary(roomKey, node))
      return { ok: true, result: out }
    },

    async roomState({ roomKey } = {}) {
      const node = getRoom(roomKey)
      const state = await node.state()
      return {
        ok: true,
        result: {
          manifest: state.manifest ?? null,
          members: membersOf(state),
          claim: state.claim ?? null,
          capEpoch: state.capEpoch ?? 0,
          messages: (state.messages ?? []).map((m) => ({
            seq: seqOfMsgId(m.msgId),
            msgId: m.msgId,
            actor: m.actor,
            ts: m.ts,
            body: m.deleted ? '' : (m.body ?? ''),
            replyTo: m.replyTo ?? '',
            expectation: m.expectation ?? '',
            attachment: m.deleted ? null : attachmentView(m.attachment),
            deleted: !!m.deleted,
            edited: !!m.edited,
          })),
          skippedCount: (state.skipped ?? []).length,
          rejectedCount: (state.rejected ?? []).length,
        },
      }
    },

    async post({ roomKey, body, expectation, ts } = {}) {
      const node = getRoom(roomKey)
      const seq = await takeSeq(node)
      const op = signOp({ seq, actor, ts: ts ?? Date.now(), kind: 'msg.post', body: body ?? '', expectation: expectation ?? '' }, keys)
      return appendAndVerify(node, op)
    },

    async claimRoom({ roomKey, assignee, ts } = {}) {
      const node = getRoom(roomKey)
      const seq = await takeSeq(node)
      const op = signOp({ seq, actor, ts: ts ?? Date.now(), kind: 'room.claim', assignee: assignee ?? actor }, keys)
      return appendAndVerify(node, op)
    },

    async releaseClaim({ roomKey, ts } = {}) {
      const node = getRoom(roomKey)
      const seq = await takeSeq(node)
      const op = signOp({ seq, actor, ts: ts ?? Date.now(), kind: 'room.claim', assignee: '' }, keys)
      return appendAndVerify(node, op)
    },

    async attach({ roomKey, filePath, body, contentType, expectation, ts } = {}) {
      const node = getRoom(roomKey)
      const bytes = fs.readFileSync(filePath)
      const blobStore = await openBlobStore(node.store)
      const name = baseNameOf(filePath)
      const ref = await putAttachment(blobStore, { name, contentType: contentType ?? 'application/octet-stream', bytes })
      const seq = await takeSeq(node)
      const op = signOp({ seq, actor, ts: ts ?? Date.now(), kind: 'msg.post', body: body ?? '', expectation: expectation ?? '', attachment: ref }, keys)
      const verdict = await appendAndVerify(node, op)
      if (!verdict.ok) return verdict
      const refObj = JSON.parse(ref)
      return { ok: true, result: { seq, ref, sha256: refObj.sha256 } }
    },

    async fetchAttachment({ roomKey, ref, savePath } = {}) {
      if (!savePath) throw new Error('fetchAttachment requires savePath')
      const node = getRoom(roomKey)
      const { bytes, ref: parsedRef } = await getAttachment(node.store, ref)
      let target = savePath
      const looksLikeDir = looksLikeDirPath(target) || (fs.existsSync(target) && fs.statSync(target).isDirectory())
      if (looksLikeDir) target = joinPath(target, parsedRef.name || 'attachment')
      fs.mkdirSync(dirnameOf(target), { recursive: true })
      fs.writeFileSync(target, bytes)
      // Deviation from bridge-server.mjs, declared: `path.resolve(target)`
      // is not available without a `#path` alias (not requested — too small
      // a need to justify one more runtime-primitive surface). Returns
      // `target` as given/joined, absolute already in every caller this
      // file's own spike exercises. Flagged in PHASE2_BRIDGE_REPORT.md.
      return { ok: true, result: { path: target, sha256: parsedRef.sha256, verified: true } }
    },

    async createSocialRoom({ title, encryptionKey, ts } = {}) {
      if (!storageDir) throw new Error('createSocialRoom requires storageDir on createBridgeCore')
      const node = await createSocialRoomNode({
        creatorKeys: keys,
        storage: joinPath(storageDir, `social-${randomHex(16)}`),
        title: title ?? '',
        encryptionKey: encryptionKey ? Buffer.from(encryptionKey, 'hex') : undefined,
        ts: ts ?? Date.now(),
        actor,
      })
      registerRoom(node.key, node)
      return { ok: true, result: { roomKey: node.key } }
    },

    async openDmInvite({ roomKey, ts, inviteSeed } = {}) {
      const node = getRoom(roomKey)
      const seq = await takeSeq(node)
      const seedBuf = inviteSeed ? Buffer.from(inviteSeed, 'hex') : hcrypto.randomBytes(32)
      const invite = inviteKeys(seedBuf)
      const offer = inviteOfferOp({ seq, actor, ts: ts ?? Date.now(), invitePub: invite.pubHex }, keys)
      const verdict = await appendAndVerify(node, offer)
      if (!verdict.ok) return verdict
      const code = encodeInviteCode({
        baseKey: node.key,
        authorityPub: node.authorityPub,
        inviteSeed: seedBuf,
        inviteId: `${actor}:${seq}`,
        encryptionKey: node.encryptionKey,
      })
      return { ok: true, result: { inviteCode: code } }
    },

    async redeemInvite({ inviteCode, actor: redeemActor, ts } = {}) {
      const decoded = decodeInviteCode(inviteCode)
      const joinActor = redeemActor || actor
      let node = rooms.get(decoded.baseKey)
      if (!node) {
        if (!storageDir) throw new Error('redeemInvite requires storageDir on createBridgeCore to join a not-yet-open room')
        node = await createMeshNode({
          storage: joinPath(storageDir, `joined-${decoded.baseKey.slice(0, 16)}-${randomHex(8)}`),
          bootstrap: decoded.baseKey, authorityPub: decoded.authorityPub, mode: 'room',
          encryptionKey: decoded.encryptionKey,
        })
      }
      registerRoom(node.key, node)
      await waitFor(async () => { await node.base.update(); return node.writable }, {
        label: `writer registration for ${joinActor} in room ${node.key}`, timeout: 20000,
      })
      const seq = await takeSeq(node)
      const redeemOp = inviteRedeemOp({ seq, actor: joinActor, ts: ts ?? Date.now(), inviteId: decoded.inviteId }, inviteKeys(decoded.inviteSeed), keys)
      const verdict = await appendAndVerify(node, redeemOp)
      if (!verdict.ok) return verdict
      return { ok: true, result: { roomKey: node.key } }
    },

    async exportTranscript({ roomKey, exportedBy } = {}) {
      const node = getRoom(roomKey)
      const bundle = await exportTranscriptBundle(node, { exportedBy })
      return { ok: true, result: bundle }
    },
  }

  async function dispatch(msg) {
    if (!msg || typeof msg !== 'object' || !Number.isInteger(msg.id) || typeof msg.method !== 'string') {
      return { ok: false, error: 'malformed frame: expected {id:number, method:string, params?:object}' }
    }
    const fn = methods[msg.method]
    if (!fn) return { id: msg.id, ok: false, error: `unknown method ${JSON.stringify(msg.method)}` }
    try {
      const outcome = await fn(msg.params ?? {})
      return { id: msg.id, ...outcome }
    } catch (err) {
      return { id: msg.id, ok: false, error: err?.message ?? String(err) }
    }
  }

  return {
    rooms,
    registerRoom,
    dispatch,
    /** onEvent(cb) — cb(event, params). Returns unsub(). */
    onEvent(cb) { eventListeners.add(cb); return () => eventListeners.delete(cb) },
    close() {
      for (const unsub of roomUnsubs.values()) unsub()
      roomUnsubs.clear()
      eventListeners.clear()
    },
  }
}

/**
 * attachStdioTransport({ core, io }) — the ndjson framing loop, decoupled
 * from where the bytes come from. `io`: { onData(cb) -> unsub, write(str) }.
 * Malformed frames get `{ok:false, error}` back, never crash the loop
 * (§1's own reliability requirement, same as bridge-server.mjs's per-socket
 * handler).
 *
 * `io.write` receives the frame's JSON string WITHOUT a trailing newline —
 * the newline is `io`'s own responsibility (RULE 2, PHASE0_NOTES_D2_FLUSH_
 * RACE.md: the real stdio `io` writes via `console.log`, which appends
 * exactly the newline ndjson framing wants; a test `io` fake is free to add
 * its own or not, since JSON.parse tolerates trailing whitespace either
 * way).
 */
export function attachStdioTransport({ core, io }) {
  let buf = ''
  const inFlight = new Set()
  function writeFrame(obj) {
    return io.write(JSON.stringify(obj))
  }
  const offEvent = core.onEvent((event, params) => writeFrame({ event, params }))
  const offData = io.onData((chunk) => {
    buf += chunk
    let idx
    while ((idx = buf.indexOf('\n')) !== -1) {
      const line = buf.slice(0, idx)
      buf = buf.slice(idx + 1)
      if (!line.trim()) continue
      let msg
      try {
        msg = JSON.parse(line)
      } catch {
        writeFrame({ ok: false, error: 'malformed frame: not valid JSON' })
        continue
      }
      // Tracked in `inFlight` so a caller (runStdioWorker, on stdin 'end')
      // can DRAIN before exiting rather than truncating a response that
      // was still being computed when the writer closed its end of the
      // pipe — see runStdioWorker's header for why this matters on top of
      // RULE 3's explicit-exit requirement, not instead of it.
      const p = core.dispatch(msg).then(writeFrame, (err) => {
        writeFrame({ id: msg.id, ok: false, error: err?.message ?? String(err) })
      })
      inFlight.add(p)
      p.finally(() => inFlight.delete(p))
    }
  })
  return {
    /** Bytes currently buffered but not yet a complete line — a non-empty
     * value here at close() time means a partial frame was in flight
     * (either a genuine short read, or exactly the truncation this layer
     * exists to make visible). */
    pendingPartial() { return buf },
    /** Resolves once every dispatch() started so far has written its
     * response. Safe to call repeatedly; new dispatches started WHILE
     * draining are also waited for (loops until the set is empty). */
    async waitIdle() {
      while (inFlight.size > 0) await Promise.allSettled([...inFlight])
    },
    close() { offData(); offEvent() },
  }
}

/**
 * getRealStdio() -> { onData(cb)->unsub, onEnd(cb)->unsub, write(str)->bool },
 * backed by the REAL process stdin/stdout.
 *
 * `process` is a Node global (present with NO import at all); under Bare it
 * does not exist as a global (verified, PHASE1A_REPORT.md), so this
 * dynamically imports `bare-process` — a real npm package specifier, never
 * `node:`-prefixed, and only reached on the branch where Node's ambient
 * `process` is absent.
 *
 * RULE 2 (PHASE0_NOTES_D2_FLUSH_RACE.md §2, binding): `write` uses
 * `console.log`, NEVER `proc.stdout.write()` directly. P0-D measured
 * `bare-process`'s `stdout.write()` HANGING 30/30 on a real spawned pipe —
 * completely independent of the wasm-compile race (Bug A) — while
 * `console.log` measured 30/30 clean on the same topology. This is not a
 * style preference; the two are not interchangeable here.
 */
export async function getRealStdio() {
  const proc = typeof process !== 'undefined' ? process : (await import('bare-process')).default
  const dataListeners = new Set()
  const endListeners = new Set()
  proc.stdin.on('data', (chunk) => {
    const text = typeof chunk === 'string' ? chunk : chunk.toString('utf8')
    for (const cb of dataListeners) cb(text)
  })
  proc.stdin.on('end', () => { for (const cb of endListeners) cb() })
  return {
    onData(cb) { dataListeners.add(cb); return () => dataListeners.delete(cb) },
    onEnd(cb) { endListeners.add(cb); return () => endListeners.delete(cb) },
    write(str) { console.log(str); return true },
  }
}

/**
 * runStdioWorker({ rooms, actor, deviceKeys, storageDir }) — the real DP4
 * entry point: one bridge-core, real stdio, stays alive until the writer
 * closes its end of the pipe (stdin 'end'), then drains and exits.
 *
 * FLUSH-RACE MITIGATION, current understanding (PHASE0_NOTES_D2_FLUSH_
 * RACE.md — this superseded an earlier, partly-wrong draft of this
 * comment; corrected here rather than silently rewritten, per the
 * campaign's own transparency norm):
 *   RULE 1 — never `WebAssembly.compile()`/`instantiate()` anywhere in the
 *     reducer channel (apply-bare.mjs already complies; not this file's
 *     own concern, but load-bearing for this worker's correctness since
 *     `mesh-node.mjs`'s `state()` calls into it).
 *   RULE 2 — `getRealStdio()`'s `write` uses `console.log`, never
 *     `proc.stdout.write()` (see that function's own comment).
 *   RULE 3 — an EXPLICIT exit call on stdin 'end' is load-bearing, not
 *     cleanup: P0-D measured 10/10 hangs when a version of this worker
 *     tried to let the loop drain naturally instead. This function adds
 *     ONE thing beyond P0-D's own note: it drains in-flight dispatches
 *     (`transport.waitIdle()`) BEFORE exiting, so a request that arrived
 *     just before the writer closed stdin still gets its response written
 *     before the process exits, rather than being truncated by an
 *     exit-immediately policy. This drain step is this file's own
 *     addition, not verified against a real flush-race reproduction
 *     (P0-D's own repro scripts don't have a request/response cycle to
 *     drain) — flagged honestly in PHASE2_BRIDGE_REPORT.md.
 *   Frame-level detectability (each response echoes its request's `id`,
 *   `pendingPartial()` exposes any dangling bytes) remains as designed —
 *   a dropped frame is still OBSERVABLE even though RULE 2/3 make it far
 *   less likely to happen in the first place.
 */
export async function runStdioWorker({ rooms, actor, deviceKeys, storageDir }) {
  const core = createBridgeCore({ rooms, actor, deviceKeys, storageDir })
  const io = await getRealStdio()
  const transport = attachStdioTransport({ core, io })
  io.onEnd(async () => {
    await transport.waitIdle()
    if (typeof Bare !== 'undefined') Bare.exit(0)
    else process.exit(0)
  })
  return {
    core,
    close() { transport.close(); core.close() },
  }
}
