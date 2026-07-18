// kit-repl.mjs — Mission U1.5: the field kit's terminal REPL.
//
// Two halves, deliberately separated per the mission brief ("factor commands
// so the spike calls them; do not screen-scrape readline"):
//   - createCommandLayer(ctx) — plain async functions, one per ceremony step.
//     kit-spike.mjs drives these DIRECTLY (same code path production uses).
//   - startRepl(ctx) — node:readline wiring that parses a typed line into a
//     command-layer call and prints its result. Pretty but plain: no color
//     libraries, no extra deps.
//
// Commands NOT in the sidecar protocol v0 wire contract (mesh/docs/
// MESSENGER_UI_CAMPAIGN.md §1) — /join's pairing code and /addwriter — are
// the GL-9 cold-peer onboarding gap this mission was asked to solve at the
// ceremony level. They reach `ctx.server.rooms` directly (an in-process Map,
// the same surface bridge-spike.mjs's own host-side setup already uses for
// `addWriter`), not a new bridge wire method — kit-host.mjs and kit-repl.mjs
// run in the SAME process for exactly this reason (see kit-host.mjs header).
//
// No read receipts, no typing indicators, no delivery ticks — anywhere
// (Constitution Art. III/IV, restated in MESSENGER_UI_CAMPAIGN.md §3). The
// REPL surfaces the fold's own skip/reject reasons verbatim; it never
// re-implements chat law.

import readline from 'node:readline'
import { randomUUID, randomBytes } from 'node:crypto'
import { writeFileSync, mkdirSync, existsSync } from 'node:fs'
import { join, extname } from 'node:path'
import { createMeshNode, waitFor } from '../host/mesh-node.mjs'
import { createSocialRoom as createSocialRoomNode } from '../host/social-room.mjs'
import { decodeInviteCode } from '../host/invite-code.mjs'
import { loadRoomRegistry, saveRoomRegistryEntry, updateRoomRegistryPeer } from './kit-registry.mjs'

const DEFAULT_TCP_PORT = 4300

/** F1 (MSG-D25) default download path, with collision-avoidance: a bare
 * `/fetch <seq>` (no savePath) lands in data\downloads\<original-name>,
 * creating the dir if needed and never clobbering an existing file — a
 * second fetch of the same filename gets `-1`, `-2`, ... appended before
 * the extension. */
function uniqueDownloadPath(downloadsDir, name) {
  mkdirSync(downloadsDir, { recursive: true })
  const safeName = (name || 'attachment').replace(/[<>:"|?*\x00-\x1f]/g, '_')
  const ext = extname(safeName)
  const stem = ext ? safeName.slice(0, -ext.length) : safeName
  let candidate = join(downloadsDir, safeName)
  for (let n = 1; existsSync(candidate); n++) candidate = join(downloadsDir, `${stem}-${n}${ext}`)
  return candidate
}

function fmtRoom(r) {
  const key = r.roomKey.slice(0, 12) + '…'
  const title = r.title || '(untitled)'
  const preview = (r.lastPreview || '').slice(0, 40)
  return `  [${r.kind}] ${key}  "${title}"  last: "${preview}"  top: ${r.topExpectation || '-'}`
}

/** Shared with /open, /join's already-joined path, and the live stream
 * (deliverable 8) — one message line format everywhere, never duplicated. */
function fmtMessage(m) {
  return `  #${m.seq} [${m.expectation || '-'}] ${m.actor}: ${m.body}${m.attachment ? ` 📎 ${m.attachment.name}  (get it: /fetch ${m.seq})` : ''}`
}

/** Print a line WITHOUT corrupting a partially-typed command (deliverable
 * 8's readline-safety requirement). readline owns the current terminal row
 * (prompt + whatever the human has typed so far); writing straight through
 * it would interleave a live message into the middle of their keystrokes.
 * The fix: clear the current row, print above it, then ask readline to
 * redraw itself — `rl.prompt(true)` re-renders `prompt + rl.line` with the
 * cursor preserved at its prior offset, so an in-flight command survives
 * byte-for-byte. Falls back to a plain log when there's no live readline
 * (kit-spike.mjs's headless ctx, or before startRepl wires `ctx._rl`). */
function printLive(ctx, text) {
  if (ctx._rl) {
    readline.clearLine(process.stdout, 0)
    readline.cursorTo(process.stdout, 0)
    ctx.log(text)
    ctx._rl.prompt(true)
  } else {
    ctx.log(text)
  }
}

const LIVE_NOTICE_THROTTLE_MS = 4000

/**
 * createLiveStream(ctx, { onMessages } = {}) -> { stop() }
 *
 * Deliverable 8 (owner field finding): a message posted by anyone previously
 * only appeared after a manual /open — the conversation was invisible while
 * it happened. This subscribes to the bridge's own 'room-updated' event
 * (bridge-server.mjs broadcasts it on every Autobase update, already wired,
 * already used by bridge-spike.mjs) and, on each tick, diffs the room's
 * current message SET against the set of msgIds already accounted for
 * (`ctx._seenMsgIds`, seeded by create/open/join/_openAlreadyJoined right
 * after they print their own initial dump — so a live tick only ever
 * reports messages genuinely NEW since this device last displayed the
 * room, never a replay of history).
 *
 * IDENTITY-based diff, not positional: a first draft diffed by ARRAY INDEX
 * (`messages.slice(seenCount)`), which broke under a real, reproducible
 * race — `roomState`'s `messages` array is the canonically-ordered
 * (Seq, Actor) view (MSG-D3), and each bridge-server's `takeSeq` caches its
 * OWN next-seq counter in-memory per device (bridge-server.mjs's own
 * documented GL-9 lesson: seq is a GLOBAL max, but each device's cache is
 * seeded independently and never resyncs against ops it only learns about
 * via replication). Two devices posting close together can legitimately
 * land on canonical positions that don't match arrival order once a third
 * op's seq sorts in BETWEEN two already-seen ones — a positional slice then
 * either replays an already-shown message as "fresh" (reproduced live: a
 * kitspike run showed sam's OWN earlier reply re-appearing as new the
 * moment a third message arrived) or, worse, could silently skip a
 * genuinely new one that landed below the old cutoff. `msgId` (MSG-D3:
 * derived `{actor}:{seq}`, globally unique, stable regardless of position)
 * makes the diff immune to this by construction — membership, not offset.
 *
 * `onMessages(roomKey, freshMessages, { current, title })` fires for EVERY
 * room that updates, not just the open one — kit-spike.mjs hooks this SAME
 * callback (never screen-scraping stdout) to prove live delivery; the REPL
 * itself passes no callback and just prints: full lines (same format as
 * /open, via fmtMessage — including the poster's OWN echo, deliberately
 * NOT suppressed, since that echo is the proof the op actually folded, not
 * just that the keystroke happened) for the CURRENTLY open room, or a
 * throttled one-line notice for any other room (never per-message spam).
 */
export function createLiveStream(ctx, { onMessages } = {}) {
  ctx._seenMsgIds = ctx._seenMsgIds || new Map() // roomKey -> Set<msgId> already accounted for
  const lastNotice = new Map() // roomKey -> ts of the last background notice
  const unsub = ctx.client.on('room-updated', async ({ roomKey }) => {
    let state
    try {
      state = await ctx.client.request('roomState', { roomKey })
    } catch {
      return // transient — the NEXT update event retries; never crash the stream
    }
    let seen = ctx._seenMsgIds.get(roomKey)
    if (!seen) {
      // First-ever observation of this room via the stream: baseline
      // silently to whatever's there right now — no history replay.
      ctx._seenMsgIds.set(roomKey, new Set(state.messages.map((m) => m.msgId)))
      return
    }
    const fresh = state.messages.filter((m) => !seen.has(m.msgId))
    for (const m of fresh) seen.add(m.msgId)
    if (!fresh.length) return
    const isCurrent = roomKey === ctx.currentRoomKey
    if (onMessages) { try { onMessages(roomKey, fresh, { current: isCurrent, title: state.manifest?.title ?? '' }) } catch { /* caller's problem, never ours */ } }
    if (isCurrent) {
      for (const m of fresh) printLive(ctx, fmtMessage(m))
    } else {
      const now = Date.now()
      if (now - (lastNotice.get(roomKey) ?? 0) > LIVE_NOTICE_THROTTLE_MS) {
        lastNotice.set(roomKey, now)
        printLive(ctx, `  (${fresh.length} new in "${state.manifest?.title || '(untitled)'}" — /open to switch)`)
      }
    }
  })
  return { stop: unsub }
}

async function ensureTcpListen(ctx, node) {
  ctx._tcpListening = ctx._tcpListening || new Set()
  ctx.tcpPorts = ctx.tcpPorts || new Map() // roomKey -> bound port, read by kit-spike.mjs (no log-scraping)
  if (ctx._tcpListening.has(node.key)) return ctx.tcpPorts.get(node.key)
  try {
    // F2 auto-reconnect (MSG-D25): remember whoever connects TO us, best
    // effort — see kit-registry.mjs's lastPeer doc for the honest limits
    // (we know their IP, not their listening port; we assume our own
    // ACTUAL bound port, the README's own convention that both machines
    // listen on the same port). `boundPort` is read, not captured, at
    // callback time — it's assigned the line right after `listenTcp`
    // resolves, strictly before any queued 'connection' event can fire
    // (nothing was listening yet to accept one). A first draft used the
    // REQUESTED port (`ctx.tcpPort ?? DEFAULT_TCP_PORT`) instead, which is
    // wrong the moment the requested port is 0 (OS-assigned) — exactly
    // kitspike's own setup — and would have persisted "peer:0" into the
    // registry, silently poisoning auto-reconnect.
    let boundPort
    const port = await ctx.net.listenTcp(ctx.tcpPort ?? DEFAULT_TCP_PORT, node, (remoteAddress) => {
      updateRoomRegistryPeer(ctx.keysDir, node.key, `${remoteAddress}:${boundPort}`)
    })
    boundPort = port
    ctx._tcpListening.add(node.key)
    ctx.tcpPorts.set(node.key, port)
    ctx.log(`  (LAN fallback listening on port ${port} for this room — give the other machine this IP:${port})`)
    return port
  } catch (err) {
    ctx.log(`  (LAN fallback listen skipped: ${err.message} — /connect from this side, or /listen <port> to retry)`)
    return undefined
  }
}

/**
 * createCommandLayer(ctx) -> { rooms, create, open, post, claim, release,
 *   attach, fetch, invite, join, addWriter, connect, listen, exportTranscript }
 *
 * Every method is called AS `cmds.method(...)` (relies on `this` for
 * requireCurrent()) — never destructure the returned object.
 */
export function createCommandLayer(ctx) {
  function setCurrent(roomKey, node) {
    ctx.currentRoomKey = roomKey
    ctx.net.joinHyperswarm(roomKey, node) // best-effort DHT join; never throws
  }

  return {
    async rooms() {
      const list = await ctx.client.request('listRooms')
      if (!list.length) ctx.log('  (no rooms yet — /create <title> to found one, or /join <invite-code>)')
      for (const r of list) ctx.log(fmtRoom(r))
      return list
    },

    // DEVIATION, declared: this bypasses bridge-server.mjs's `createSocialRoom`
    // wire method (which picks an unpredictable `social-<uuid>` storage dir)
    // and calls social-room.mjs directly with a storage path THIS module
    // chooses and remembers — GL-5 reopen discipline (kit-registry.mjs)
    // needs a stable directory name to reopen against after a restart, and
    // Corestore doesn't reliably expose the path it was opened with back out.
    // The room is registered into the SAME bridge `rooms` Map either way, so
    // every protocol v0 method (post, roomState, attach, …) behaves
    // identically afterward — only the creation path differs.
    async create(title) {
      const dirName = `room-${randomUUID()}`
      const encryptionKey = randomBytes(32)
      const node = await createSocialRoomNode({
        creatorKeys: ctx.keys, storage: join(ctx.corestoreDir, dirName),
        title: title || '(untitled)', encryptionKey, ts: Date.now(), actor: ctx.actor,
      })
      ctx.server.registerRoom(node.key, node)
      saveRoomRegistryEntry(ctx.keysDir, {
        roomKey: node.key, storage: dirName, authorityPub: ctx.keys.pubHex,
        encryptionKey: encryptionKey.toString('hex'), bootstrap: null, title: title || '(untitled)',
      })
      setCurrent(node.key, node)
      await ensureTcpListen(ctx, node)
      ctx._seenMsgIds = ctx._seenMsgIds || new Map() // deliverable 8: fresh room, empty baseline — even the founder's own first post streams live
      ctx._seenMsgIds.set(node.key, new Set())
      ctx.log(`created + opened "${title || '(untitled)'}" — ${node.key.slice(0, 16)}…`)
      return { roomKey: node.key }
    },

    async open(roomKeyPrefix) {
      if (!roomKeyPrefix) throw new Error('usage: /open <roomKey-or-prefix> (see /rooms)')
      const list = await ctx.client.request('listRooms')
      const match = list.find((r) => r.roomKey === roomKeyPrefix || r.roomKey.startsWith(roomKeyPrefix))
      if (!match) throw new Error(`no room matches ${JSON.stringify(roomKeyPrefix)} — /rooms to list`)
      const node = ctx.server.rooms.get(match.roomKey)
      setCurrent(match.roomKey, node)
      await ensureTcpListen(ctx, node)
      const state = await ctx.client.request('roomState', { roomKey: match.roomKey })
      ctx.log(`opened "${state.manifest?.title ?? ''}" — ${state.messages.length} message(s)`)
      for (const m of state.messages.slice(-10)) ctx.log(fmtMessage(m))
      // Deliverable 8: seed the live-stream baseline to what we JUST showed —
      // everything from here on is genuinely new, never a history replay.
      ctx._seenMsgIds = ctx._seenMsgIds || new Map()
      ctx._seenMsgIds.set(match.roomKey, new Set(state.messages.map((m) => m.msgId)))
      return { roomKey: match.roomKey, state }
    },

    requireCurrent() {
      if (!ctx.currentRoomKey) throw new Error('no room open — /rooms then /open <key>, or /create <title>')
      return ctx.currentRoomKey
    },

    async post(body, expectation = '') {
      const roomKey = this.requireCurrent()
      const res = await ctx.client.request('post', { roomKey, body, expectation })
      ctx.log(`posted (seq ${res.seq}${expectation ? `, expect: ${expectation}` : ''})`)
      return res
    },

    async claim(assignee) {
      const roomKey = this.requireCurrent()
      const res = await ctx.client.request('claimRoom', { roomKey, assignee: assignee || ctx.actor })
      ctx.log(`claimed (seq ${res.seq})`)
      return res
    },

    async release() {
      const roomKey = this.requireCurrent()
      const res = await ctx.client.request('releaseClaim', { roomKey })
      ctx.log(`released (seq ${res.seq})`)
      return res
    },

    async attach(filePath, body = '') {
      if (!filePath) throw new Error('usage: /attach <path> [message text]')
      const roomKey = this.requireCurrent()
      const res = await ctx.client.request('attach', { roomKey, filePath, body })
      ctx.log(`attached ${filePath} (seq ${res.seq}) — sha256 ${res.sha256}`)
      return res
    },

    async fetch(refOrSeq, savePath) {
      if (!refOrSeq) throw new Error('usage: /fetch <seq> [savePath]   (the #N shown next to the 📎 line; savePath is optional — defaults to data\\downloads\\<original-name>)')
      const roomKey = this.requireCurrent()
      let ref = refOrSeq
      let attachmentName
      // Kitchen-table UX: the ref is an opaque JSON locator (hostile to type,
      // full of quotes) — so the human-facing handle is the message SEQ shown
      // in /open. A plain integer resolves to that message's attachment ref;
      // anything else is treated as a raw ref (kit-spike still drives that).
      if (/^\d+$/.test(String(refOrSeq).trim())) {
        const seq = Number(refOrSeq)
        const state = await ctx.client.request('roomState', { roomKey })
        const msg = state.messages.find((m) => m.seq === seq && m.attachment)
        if (!msg) throw new Error(`no attachment on message #${seq} — /open to see the 📎 lines`)
        ref = msg.attachment.ref
        attachmentName = msg.attachment.name
      } else {
        try { attachmentName = JSON.parse(ref).name } catch { /* raw/malformed ref — bridge will surface its own error */ }
      }
      // F1 (MSG-D25): no savePath -> data\downloads\<name>, collision-safe.
      const target = savePath || uniqueDownloadPath(join(ctx.dataDir, 'downloads'), attachmentName)
      const res = await ctx.client.request('fetchAttachment', { roomKey, ref, savePath: target })
      ctx.log(`fetched -> ${res.path}\n  sha256 ${res.sha256}  verified: ${res.verified}`)
      if (res.verified) ctx.log('FILE VERIFIED END-TO-END ✅')
      return res
    },

    async invite() {
      const roomKey = this.requireCurrent()
      const res = await ctx.client.request('openDmInvite', { roomKey })
      ctx.log('invite code (read it to the other person, or send it however you like):')
      ctx.log(`  ${res.inviteCode}`)
      return res
    },

    /** The GL-9 cold-peer ceremony, joiner side. Prints the PAIRING CODE
     * (this device's writer key) immediately, then blocks until the founder
     * runs /addwriter AND a network path exists (hyperswarm and/or
     * /connect), then completes the real invite.redeem automatically. */
    // { onPairingCode } lets a caller (kit-spike.mjs) observe the pairing
    // code the moment it's minted, WITHOUT screen-scraping the log line —
    // the REPL itself never passes this option, it just reads the log.
    // F2 (MSG-D25): opening an already-joined room must NEVER go through
    // invite.redeem again — surfaces the recent history like /open, no
    // error tone at all.
    async _openAlreadyJoined(node, note) {
      setCurrent(node.key, node)
      await ensureTcpListen(ctx, node)
      const state = await ctx.client.request('roomState', { roomKey: node.key })
      ctx.log(`${note} ("${state.manifest?.title ?? ''}", ${state.messages.length} message(s))`)
      for (const m of state.messages.slice(-10)) ctx.log(fmtMessage(m))
      ctx._seenMsgIds = ctx._seenMsgIds || new Map() // deliverable 8: same baseline-seeding as /open
      ctx._seenMsgIds.set(node.key, new Set(state.messages.map((m) => m.msgId)))
      return { roomKey: node.key, alreadyJoined: true, state }
    },

    async join(inviteCode, { onPairingCode } = {}) {
      if (!inviteCode) throw new Error('usage: /join <invite-code>')
      const decoded = decodeInviteCode(inviteCode)
      let node = ctx.server.rooms.get(decoded.baseKey)
      // The MOST common real-world /join: a room this device has already
      // joined (this session, or auto-reopened at boot) and is already a
      // writer in — the one-time invite is correctly exhausted, so there is
      // nothing left to redeem. Short-circuit BEFORE touching the invite
      // ceremony at all: zero risk of the hostile "exhausted" wording ever
      // reaching a person who did nothing wrong (MSG-D25, the restart-
      // rejoin field failure). MSG-D11's one-time-invite LAW is untouched —
      // this only changes what happens when redemption was never needed.
      const preExisting = !!node
      if (preExisting && node.writable) {
        return this._openAlreadyJoined(node, 'you already joined this room — opening it')
      }
      if (!node) {
        const dirName = `joined-${decoded.baseKey.slice(0, 16)}`
        node = await createMeshNode({
          storage: join(ctx.corestoreDir, dirName),
          bootstrap: decoded.baseKey, authorityPub: decoded.authorityPub, mode: 'room',
          encryptionKey: decoded.encryptionKey,
        })
        ctx.server.registerRoom(node.key, node)
        // F2(a): persisted IMMEDIATELY on node creation — before the
        // ceremony even completes — so a restart mid-ceremony still finds
        // this room on the next boot instead of orphaning it (GL-5).
        saveRoomRegistryEntry(ctx.keysDir, {
          roomKey: node.key, storage: dirName, authorityPub: decoded.authorityPub,
          encryptionKey: decoded.encryptionKey ? decoded.encryptionKey.toString('hex') : null,
          bootstrap: decoded.baseKey, title: '(joined room)',
        })
      }
      setCurrent(node.key, node)
      await ensureTcpListen(ctx, node)
      ctx.log('your PAIRING CODE — read it OUT to the founder, or have them type it in:')
      ctx.log(`  ${node.writerKey}`)
      ctx.log('waiting for the founder to run /addwriter with this code, and for a network path to connect (hyperswarm and/or /connect <ip:port> on either side)…')
      if (onPairingCode) onPairingCode(node.writerKey)
      await waitFor(async () => { await node.base.update(); return node.writable }, {
        label: `writer registration in room ${node.key}`, timeout: 10 * 60 * 1000,
      })
      try {
        const res = await ctx.client.request('redeemInvite', { inviteCode, actor: ctx.actor })
        // Deliverable 8: seed the live-stream baseline the instant we're
        // writable — the joiner's OWN first post (and anything folded while
        // the ceremony ran) streams live from here, never silently missed.
        const seedState = await ctx.client.request('roomState', { roomKey: node.key })
        ctx._seenMsgIds = ctx._seenMsgIds || new Map()
        ctx._seenMsgIds.set(node.key, new Set(seedState.messages.map((m) => m.msgId)))
        ctx.log(`joined "${node.key.slice(0, 16)}…" — you can post now`)
        return res
      } catch (err) {
        const reason = err.bridgeError ?? err.message ?? ''
        // Only reachable for a room THIS DEVICE already had resident before
        // this call (preExisting) — never for a genuinely fresh node, where
        // "exhausted" honestly means someone ELSE consumed the invite and
        // claiming "you already joined" would be a lie.
        if (preExisting && /exhausted|already holds a current grant/i.test(reason)) {
          return this._openAlreadyJoined(node, 'you already joined this room — opening it')
        }
        throw err
      }
    },

    /** The GL-9 cold-peer ceremony, founder side. */
    async addWriter(pairingCode) {
      if (!pairingCode) throw new Error('usage: /addwriter <pairing-code>')
      const roomKey = this.requireCurrent()
      const node = ctx.server.rooms.get(roomKey)
      const key = String(pairingCode).replace(/[<>"'`\s]/g, '')
      await node.addWriter(key)
      ctx.log(`writer added to ${roomKey.slice(0, 16)}… — the joiner's /join will finish on its own once the network path is up`)
      return { roomKey, writerKey: key }
    },

    async connect(ipPort) {
      if (!ipPort) throw new Error('usage: /connect <ip:port>')
      const roomKey = this.requireCurrent()
      const node = ctx.server.rooms.get(roomKey)
      const [host, portStr] = String(ipPort).split(':')
      const port = Number(portStr) || DEFAULT_TCP_PORT
      await ctx.net.connectTcp(host, port, node)
      // F2 auto-reconnect (MSG-D25): this is the exact, known-good address —
      // remembered so a future restart can reconnect without a human typing
      // /connect again.
      updateRoomRegistryPeer(ctx.keysDir, roomKey, `${host}:${port}`)
      ctx.log(`connected (LAN TCP) to ${host}:${port} — replicating this room`)
      return { host, port }
    },

    async listen(port) {
      const roomKey = this.requireCurrent()
      const node = ctx.server.rooms.get(roomKey)
      const bound = await ctx.net.listenTcp(Number(port) || DEFAULT_TCP_PORT, node)
      ctx._tcpListening = ctx._tcpListening || new Set()
      ctx.tcpPorts = ctx.tcpPorts || new Map()
      ctx._tcpListening.add(node.key)
      ctx.tcpPorts.set(node.key, bound)
      ctx.log(`listening on port ${bound} for this room`)
      return { port: bound }
    },

    async exportTranscript(savePath) {
      const roomKey = this.requireCurrent()
      const bundle = await ctx.client.request('exportTranscript', { roomKey, exportedBy: ctx.actor })
      const out = savePath || join(ctx.dataDir, `transcript-${roomKey.slice(0, 12)}-${Date.now()}.json`)
      writeFileSync(out, JSON.stringify(bundle, null, 2))
      ctx.log(`exported -> ${out}`)
      return { path: out }
    },

    /** The phone-support command (deliverable 3, MSG-D25/BAR): "read me
     * what /status says" — everything a remote-directing owner needs to
     * diagnose a stuck kit over a voice call, in plain words. */
    async status() {
      const list = await ctx.client.request('listRooms')
      ctx.log(`actor: ${ctx.actor}   device: ${ctx.keys.pubHex.slice(0, 16)}…`)
      ctx.log(`network: ${ctx.net.mode}   downloads: ${join(ctx.dataDir, 'downloads')}`)
      if (!list.length) {
        ctx.log('  (no rooms yet — /create <title> to found one, or /join <invite-code>)')
        return { rooms: [] }
      }
      const registry = loadRoomRegistry(ctx.keysDir)
      const out = []
      for (const r of list) {
        const state = await ctx.client.request('roomState', { roomKey: r.roomKey })
        const entry = registry.find((e) => e.roomKey === r.roomKey)
        const port = ctx.tcpPorts?.get(r.roomKey)
        const peers = ctx.net.peerCount(r.roomKey)
        const current = r.roomKey === ctx.currentRoomKey ? ' (OPEN)' : ''
        const count = state.messages.length
        ctx.log(`  "${r.title || '(untitled)'}"${current}  key: ${r.roomKey.slice(0, 12)}…  messages: ${count}  listening: ${port ?? 'no'}  peers: ${peers}  last-peer: ${entry?.lastPeer ?? '(none yet)'}`)
        out.push({ roomKey: r.roomKey, title: r.title, current: !!current, messages: count, listeningPort: port ?? null, peers, lastPeer: entry?.lastPeer ?? null })
      }
      return { rooms: out }
    },
  }
}

const HELP = `
commands:
  <text>                    post <text> to the open room (no expectation tag)
  /expect <tag> <text>      post with an expectation tag (whenever|today|urgent)
  /rooms                    list rooms
  /create <title>           found a new room, opens it
  /open <roomKey-or-prefix> open an existing room, shows recent messages
  /claim [assignee]         claim the open room (defaults to you)
  /release                  release your claim
  /attach <path> [text]     attach a real file to the open room
  /fetch <seq> [path]       fetch the 📎 on message #seq, verify + save
                            (no path -> saved to data\downloads\ automatically)
  /invite                   mint a DM invite code for the open room
  /join <invite-code>       redeem an invite; prints YOUR pairing code
                            (already joined this room? just reopens it — safe to retype)
  /addwriter <pairing-code> (founder) admit a joiner's device into the room
  /connect <ip:port>        LAN fallback: dial the other machine directly
  /listen <port>            LAN fallback: (re)start listening on a port
  /status                   show what this device is doing — the phone-support command
  /export [path]            export this room's transcript to a file
  /help                     this text
  /exit                     quit

new messages in the OPEN room appear automatically as they arrive — nothing
to type. Other rooms get a quiet "(N new in ... — /open to switch)" note.
`.trim()

/** Deliverable 4 (MSG-D25/BAR): on boot, tell a non-technical human the
 * ONE most likely next thing to type — plain words, 1-3 lines, no jargon.
 * Never guesses beyond what's locally knowable (no network probe here). */
function bootBanner(ctx) {
  const roomCount = ctx.server.rooms.size
  if (roomCount === 0) {
    return [
      'No room yet.',
      '  starting fresh? -> /create <a title>',
      '  someone sent you a code? -> /join <invite-code>',
    ]
  }
  if (!ctx.currentRoomKey) {
    return [`${roomCount} room(s) on this device, none open.`, '  -> /rooms to see them, then /open <key>']
  }
  const peers = ctx.net.peerCount(ctx.currentRoomKey)
  if (peers === 0) {
    return [
      'Room open, nobody connected yet.',
      '  -> /connect <ip:port> if you have it, otherwise just wait — hyperswarm and auto-reconnect keep trying',
    ]
  }
  return [`Connected — ${peers} peer(s) in this room.`, '  -> just type your message and press Enter']
}

/** startRepl(ctx) -> Promise<void>, resolves when the user exits. */
export async function startRepl(ctx) {
  const cmds = createCommandLayer(ctx)
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout, prompt: `${ctx.actor}> ` })
  ctx._rl = rl // deliverable 8: printLive()'s readline-safe redraw needs this

  // Deliverable 8: live thread display for the whole session, not just
  // while a command is in flight — a message that arrives between two
  // keystrokes must still show up.
  const liveStream = createLiveStream(ctx)

  ctx.log(`\nWelcome, ${ctx.actor}. /help for the full command list, /status any time you're unsure what's going on.`)
  for (const line of bootBanner(ctx)) ctx.log(line)
  ctx.log('')
  rl.prompt()

  rl.on('line', async (raw) => {
    const line = raw.trim()
    if (!line) { rl.prompt(); return }
    if (line === '/exit' || line === '/quit') { rl.close(); return }
    if (line === '/help') { ctx.log(HELP); rl.prompt(); return }
    try {
      if (!line.startsWith('/')) {
        await cmds.post(line)
      } else {
        // Quote-aware tokenizer: Windows humans paste paths as "C:\...\file.txt"
        // (often with spaces inside). Double-quoted spans become ONE token with
        // the quotes stripped; everything else splits on whitespace. Found at
        // the kitchen table: the naive split(' ') glued the quoted path onto
        // the CWD and ENOENT'd the very first real file transfer.
        const tokens = [...line.slice(1).matchAll(/"([^"]*)"|(\S+)/g)].map((m) => m[1] ?? m[2])
        const [cmd, ...rest] = tokens
        const arg = rest.join(' ').trim()
        switch (cmd) {
          case 'rooms': await cmds.rooms(); break
          case 'create': await cmds.create(arg); break
          case 'open': await cmds.open(arg); break
          case 'expect': {
            const [tag, ...msg] = rest
            await cmds.post(msg.join(' '), tag)
            break
          }
          case 'claim': await cmds.claim(arg || undefined); break
          case 'release': await cmds.release(); break
          case 'attach': {
            const [filePath, ...msg] = rest
            await cmds.attach(filePath, msg.join(' '))
            break
          }
          case 'fetch': {
            const [ref, savePath] = rest
            await cmds.fetch(ref, savePath)
            break
          }
          case 'invite': await cmds.invite(); break
          case 'join': await cmds.join(arg); break
          case 'addwriter': await cmds.addWriter(arg); break
          case 'connect': await cmds.connect(arg); break
          case 'listen': await cmds.listen(arg); break
          case 'status': await cmds.status(); break
          case 'export': await cmds.exportTranscript(arg || undefined); break
          default: ctx.log(`unknown command /${cmd} — /help for the list`)
        }
      }
    } catch (err) {
      ctx.log(`error: ${err.bridgeError ?? err.message}`)
    }
    rl.prompt()
  })

  await new Promise((resolve) => rl.on('close', resolve))
  liveStream.stop()
}
