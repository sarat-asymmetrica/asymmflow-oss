// bridge-server.mjs — Mission U1 (W-UI-1): the sidecar protocol v0 seam
// (mesh/docs/MESSENGER_UI_CAMPAIGN.md §1), served TODAY over localhost TCP by
// the Node host so the UI can ship against the REAL wasm fold before DP4's
// Bare sidecar exists. Transport (dev): ndjson over node:net, ZERO new deps.
//
// THIS MODULE IS A THIN ADAPTER. It never re-implements chat law (expectation
// validation, claim rules, social-room skips, capability rejections) — every
// mutating method appends the signed op and then reads back the FOLD's own
// verdict (skipped[]/rejected[], MSG-D4's taxonomy) and surfaces that reason
// VERBATIM. It wraps mesh-node.mjs / capability.mjs / social-room.mjs /
// invite-code.mjs / attachments.mjs / export-transcript.mjs — none of those
// modules are reimplemented here, only called.
//
// PROTOCOL DEVIATIONS FROM THE LITERAL BRIEF, DECLARED (GL-1 pattern — build,
// flag, let the gate rule):
//
//   1. Signature is `createBridgeServer({ rooms, actor, deviceKeys, storageDir,
//      port, socket })`, not the brief's literal single `{ node, ... }`. The
//      protocol's own method list (listRooms/roomState/post/... all take a
//      roomKey) requires serving MANY rooms concurrently, not one — a single
//      `node` param cannot satisfy the contract it's meant to serve. `rooms`
//      is a live `Map<roomKey, MeshNode>` the caller pre-populates with
//      already-open rooms (e.g. an anchored PO room opened by whatever
//      ceremony created it — room CREATION for anchored rooms is not a v0
//      method) and the bridge grows via `createSocialRoom`/`redeemInvite`.
//      The returned server object exposes that same `rooms` Map live, so a
//      host-side caller (the spike, a future real ceremony) can register
//      pre-opened rooms or reach into a MeshNode directly for host-level
//      concerns the wire protocol deliberately doesn't cover (see #2).
//
//   2. Becoming an Autobase WRITER is NOT a v0 method, and isn't invented
//      here. Every existing spike (invite-spike, social-spike, attach-spike)
//      treats `node.addWriter(writerKey)` as host-side setup that precedes
//      the fold-level ceremony (invite redemption, capability grants) — it is
//      a separate Autobase-level primitive from room/capability law. This
//      module does not add a wire method for it; `redeemInvite` waits for
//      writability (mirroring `waitFor(... return node.writable)` in every
//      prior spike) but something else — a ceremony this mission's scope
//      doesn't own — must have already called `addWriter` for the joining
//      device's writer key. bridge-spike.mjs demonstrates the realistic
//      shape of that "something else": the founder's own bridge exposes its
//      raw MeshNode via `server.rooms.get(roomKey)`, and host code calls
//      `.addWriter()` on it directly, exactly like every prior spike's setup
//      phase.
//
//   3. Every op-building method (`post`, `claimRoom`, `releaseClaim`,
//      `attach`, `openDmInvite`, `redeemInvite`) accepts an OPTIONAL `ts`
//      (and `openDmInvite` an optional `inviteSeed` hex string) beyond what
//      §1 lists. The protocol's op fields need a timestamp and §1 gives the
//      bridge no way to receive one — omitting it, the server uses
//      `Date.now()` (this is LIVE production infrastructure serving a UI;
//      "no live clocks in op data" is spike/golden canon, not a rule that
//      can apply to a server whose whole job is timestamping real messages).
//      The optional override exists ONLY so bridge-spike.mjs can drive the
//      protocol with deterministic op data (spike canon) through the SAME
//      code path production uses, rather than bypassing the bridge for
//      determinism's sake.
//
//   4. `createSocialRoom({ title })` returns `{ roomKey }`, not `{inviteCode}`
//      as §1's compressed notation ("createSocialRoom(...) / openDmInvite(...)
//      → {inviteCode}") could be read literally. Minting an invite is a
//      DISTINCT, separately-billable gesture (openDmInvite) in social-room.mjs
//      — forcing every room creation to also mint a one-time consumable
//      invite would be wasteful and semantically odd, and doesn't match the
//      underlying host functions' own shapes (`createSocialRoom` returns a
//      node; only `openDmInvite` returns a code). `{roomKey}` also lets the
//      room appear in `listRooms()` immediately, matching how anchored rooms
//      already work. `openDmInvite(roomKey)` returns `{inviteCode}` exactly
//      as documented.
//
// Session state / GL-5 seq discipline: seq is a ROOM-WIDE monotonic counter,
// not a per-actor one — canonicalLess sorts by (Seq, Actor, ...) GLOBALLY, so
// an op whose Seq ties with or trails another actor's op it causally depends
// on (e.g. "my own capability grant") can sort BEFORE that dependency and be
// rejected for it (found the hard way: desk's first post picked Seq 1 from
// an actor-scoped scan, tied hub's Seq-1 manifest, lost the tie ('desk' <
// 'hub'), and evaluated before its OWN grant at Seq 2 had folded —
// `room-spike.mjs`'s own op script confirms the convention: B_OPS/C_OPS
// continue the SAME shared seq space hub's A_OPS left off at, never restart
// a private counter at 1). The server therefore seeds its per-room counter
// from the GLOBAL max Seq across every actor's ops (never restarts at 1 on a
// reconnect/redeploy either — GL-5's reissue-spike precedent).

import net from 'node:net'
import { randomUUID, randomBytes } from 'node:crypto'
import { readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { createMeshNode, waitFor } from './mesh-node.mjs'
import { signOp, inviteKeys, inviteOfferOp, inviteRedeemOp } from './capability.mjs'
import { createSocialRoom as createSocialRoomNode } from './social-room.mjs'
import { decodeInviteCode, encodeInviteCode } from './invite-code.mjs'
import { openBlobStore, putAttachment, getAttachment } from './attachments.mjs'
import { exportTranscript as exportTranscriptBundle } from './export-transcript.mjs'

const PROTOCOL_VERSION = 'v0'

// Expectation urgency (Constitution Art. III §3): "" and "whenever" are the
// same default weight; "today" is quieter-tint; "urgent" floats to top.
const URGENCY_RANK = { urgent: 2, today: 1, whenever: 0, '': 0 }

function seqOfMsgId(msgId) {
  const i = typeof msgId === 'string' ? msgId.lastIndexOf(':') : -1
  return i === -1 ? 0 : Number(msgId.slice(i + 1))
}

/** attachment ref STRING (opaque cargo, MSG-D8/M3) -> the protocol's
 * {name, size, sha256, ref} shape, or null. Never throws — a malformed ref
 * (should never happen; the fold never validates this field) degrades to
 * null rather than crashing the bridge. */
function attachmentView(refString) {
  if (!refString) return null
  try {
    const ref = JSON.parse(refString)
    return { name: ref.name, size: ref.byteLength, sha256: ref.sha256, ref: refString }
  } catch {
    return null
  }
}

/**
 * createBridgeServer({ rooms, actor, deviceKeys, storageDir, port, socket })
 *   rooms      — optional Map<roomKey, MeshNode> of already-open rooms
 *                (default: new empty Map, grows via createSocialRoom/redeemInvite).
 *   actor      — this device's actor string (op-signing identity's label).
 *   deviceKeys — capability.mjs deviceKeys() shape; this device's Ed25519 keypair.
 *   storageDir — corestore storage root for rooms this server itself CREATES
 *                (createSocialRoom, redeemInvite joining a room not already in
 *                `rooms`). Required only if those methods are used.
 *   port       — TCP port to listen on (0 = OS-assigned ephemeral; the actual
 *                bound port is returned).
 *   socket     — alternative: a named pipe / unix socket path to listen on
 *                instead of TCP (mutually exclusive with port).
 *
 * Returns { server, port, rooms, close() }. `rooms` is the SAME live Map
 * passed in (or created) — the caller may register more rooms into it any
 * time, including via `.set(roomKey, node)` directly for pre-opened anchored
 * rooms this module never creates.
 */
export async function createBridgeServer({
  rooms = new Map(), actor, deviceKeys: keys, storageDir, port, socket,
} = {}) {
  if (!actor) throw new Error('createBridgeServer requires actor')
  if (!keys || !keys.pubHex) throw new Error('createBridgeServer requires deviceKeys (capability.mjs deviceKeys() shape)')

  const seqCounters = new Map() // roomKey -> next seq for THIS device's actor
  const roomUnsubs = new Map()  // roomKey -> () => void (base 'update' listener teardown)
  const clients = new Set()     // connected sockets, for event broadcast

  async function nextSeqFor(node) {
    // GLOBAL max across every actor's ops, not just this device's own — see
    // the header note above. Scoping this to `op.actor === actor` is exactly
    // the bug that let a fresh device's first op sort before its own grant.
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

  function broadcast(event, params) {
    const frame = JSON.stringify({ event, params }) + '\n'
    for (const sock of clients) {
      try { sock.write(frame) } catch { /* dead socket, dropped on its own 'close' */ }
    }
  }

  function watchRoom(roomKey, node) {
    if (roomUnsubs.has(roomKey)) return
    const onUpdate = () => broadcast('room-updated', { roomKey })
    // Autobase emits 'update' on any state change (base changes) — verified
    // in the installed source: autobase/index.js:839 and :2060. Coarse on
    // purpose per §1: the client refetches roomState, no incremental diffs.
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

  /** Append op, then read back the FOLD's own verdict. Never pre-validate —
   * §1's law: "the bridge never pre-filters beyond the composer's own UI." */
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

    // topExpectation (§1): "highest-urgency expectation tag on a message
    // addressed at/after my cursor position". DEVIATION, declared: v0
    // explicitly does NOT wire read cursors (§1 "Explicitly NOT in v0" —
    // "read cursors (anchored-only UI comes with the chrome wave)"), so
    // there is no cursor position to filter by yet. This computes the
    // highest-urgency tag across ALL live messages in the room instead —
    // the honest simplification available given cursors are out of scope,
    // not a silent narrowing of the spec's intent. Revisit when the chrome
    // wave wires per-reader cursors through roomState.
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

  // ── Protocol v0 methods (mesh/docs/MESSENGER_UI_CAMPAIGN.md §1) ──────────
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
      const bytes = readFileSync(filePath)
      const blobStore = await openBlobStore(node.store)
      const name = filePath.split(/[\\/]/).pop()
      const ref = await putAttachment(blobStore, { name, contentType: contentType ?? 'application/octet-stream', bytes })
      const seq = await takeSeq(node)
      const op = signOp({ seq, actor, ts: ts ?? Date.now(), kind: 'msg.post', body: body ?? '', expectation: expectation ?? '', attachment: ref }, keys)
      const verdict = await appendAndVerify(node, op)
      if (!verdict.ok) return verdict
      const refObj = JSON.parse(ref)
      return { ok: true, result: { seq, ref, sha256: refObj.sha256 } }
    },

    async fetchAttachment({ roomKey, ref, savePath } = {}) {
      const node = getRoom(roomKey)
      const { bytes, ref: parsedRef } = await getAttachment(node.store, ref)
      writeFileSync(savePath, bytes)
      return { ok: true, result: { path: savePath, sha256: parsedRef.sha256, verified: true } }
    },

    async createSocialRoom({ title, encryptionKey, ts } = {}) {
      if (!storageDir) throw new Error('createSocialRoom requires storageDir on createBridgeServer')
      const node = await createSocialRoomNode({
        creatorKeys: keys,
        storage: join(storageDir, `social-${randomUUID()}`),
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
      const seedBuf = inviteSeed ? Buffer.from(inviteSeed, 'hex') : randomBytes(32)
      const invite = inviteKeys(seedBuf)
      const offer = inviteOfferOp({ seq, actor, ts: ts ?? Date.now(), invitePub: invite.pubHex }, keys)
      const verdict = await appendAndVerify(node, offer)
      if (!verdict.ok) return verdict
      const code = encodeInviteCode({
        baseKey: node.key,
        authorityPub: node.authorityPub, // this room's OWN authority, not necessarily `keys` — mirrors social-room.mjs's own doc note
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
        if (!storageDir) throw new Error('redeemInvite requires storageDir on createBridgeServer to join a not-yet-open room')
        node = await createMeshNode({
          storage: join(storageDir, `joined-${decoded.baseKey.slice(0, 16)}-${randomUUID()}`),
          bootstrap: decoded.baseKey, authorityPub: decoded.authorityPub, mode: 'room',
          encryptionKey: decoded.encryptionKey,
        })
      }
      registerRoom(node.key, node)
      // Writer registration is a SEPARATE, pre-existing Autobase precondition
      // this method does not perform — see deviation #2 above.
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

  const server = net.createServer((sock) => {
    clients.add(sock)
    let buf = ''
    sock.on('data', async (chunk) => {
      buf += chunk.toString('utf8')
      let idx
      while ((idx = buf.indexOf('\n')) !== -1) {
        const line = buf.slice(0, idx)
        buf = buf.slice(idx + 1)
        if (!line.trim()) continue
        let msg
        try {
          msg = JSON.parse(line)
        } catch {
          // Malformed frame: {ok:false} without crashing the server (§1's
          // own reliability requirement). No id to correlate — best effort.
          try { sock.write(JSON.stringify({ ok: false, error: 'malformed frame: not valid JSON' }) + '\n') } catch {}
          continue
        }
        const response = await dispatch(msg)
        try { sock.write(JSON.stringify(response) + '\n') } catch { /* socket gone */ }
      }
    })
    sock.on('close', () => clients.delete(sock))
    sock.on('error', () => clients.delete(sock))
  })

  await new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(socket ?? port ?? 0, () => { server.off('error', reject); resolve() })
  })
  const boundPort = socket ? null : server.address().port

  return {
    server,
    port: boundPort,
    socket: socket ?? null,
    rooms,
    /** Register an already-open MeshNode (e.g. an anchored room opened by a
     * ceremony outside this module, or a room a host-side caller connected
     * and addWriter'd directly — deviation #2) so it gets `room-updated`
     * event wiring too. Plain `rooms.set()` would skip that wiring. */
    registerRoom,
    async close() {
      for (const unsub of roomUnsubs.values()) unsub()
      roomUnsubs.clear()
      for (const sock of clients) sock.destroy()
      clients.clear()
      await new Promise((resolve) => server.close(resolve))
    },
  }
}
