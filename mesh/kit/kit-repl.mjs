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
import { writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { createMeshNode, waitFor } from '../host/mesh-node.mjs'
import { createSocialRoom as createSocialRoomNode } from '../host/social-room.mjs'
import { decodeInviteCode } from '../host/invite-code.mjs'
import { saveRoomRegistryEntry } from './kit-registry.mjs'

const DEFAULT_TCP_PORT = 4300

function fmtRoom(r) {
  const key = r.roomKey.slice(0, 12) + '…'
  const title = r.title || '(untitled)'
  const preview = (r.lastPreview || '').slice(0, 40)
  return `  [${r.kind}] ${key}  "${title}"  last: "${preview}"  top: ${r.topExpectation || '-'}`
}

async function ensureTcpListen(ctx, node) {
  ctx._tcpListening = ctx._tcpListening || new Set()
  ctx.tcpPorts = ctx.tcpPorts || new Map() // roomKey -> bound port, read by kit-spike.mjs (no log-scraping)
  if (ctx._tcpListening.has(node.key)) return ctx.tcpPorts.get(node.key)
  try {
    const port = await ctx.net.listenTcp(ctx.tcpPort ?? DEFAULT_TCP_PORT, node)
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
      for (const m of state.messages.slice(-10)) {
        ctx.log(`  [${m.expectation || '-'}] ${m.actor}: ${m.body}${m.attachment ? ` 📎 ${m.attachment.name}` : ''}`)
      }
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

    async fetch(ref, savePath) {
      if (!ref || !savePath) throw new Error('usage: /fetch <ref-json> <savePath>')
      const roomKey = this.requireCurrent()
      const res = await ctx.client.request('fetchAttachment', { roomKey, ref, savePath })
      ctx.log(`fetched -> ${res.path}\n  sha256 ${res.sha256}  verified: ${res.verified}`)
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
    async join(inviteCode, { onPairingCode } = {}) {
      if (!inviteCode) throw new Error('usage: /join <invite-code>')
      const decoded = decodeInviteCode(inviteCode)
      let node = ctx.server.rooms.get(decoded.baseKey)
      if (!node) {
        const dirName = `joined-${decoded.baseKey.slice(0, 16)}`
        node = await createMeshNode({
          storage: join(ctx.corestoreDir, dirName),
          bootstrap: decoded.baseKey, authorityPub: decoded.authorityPub, mode: 'room',
          encryptionKey: decoded.encryptionKey,
        })
        ctx.server.registerRoom(node.key, node)
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
      const res = await ctx.client.request('redeemInvite', { inviteCode, actor: ctx.actor })
      ctx.log(`joined "${node.key.slice(0, 16)}…" — you can post now`)
      return res
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
  /fetch <ref-json> <path>  fetch an attachment by its ref, verify + save
  /invite                   mint a DM invite code for the open room
  /join <invite-code>       redeem an invite; prints YOUR pairing code
  /addwriter <pairing-code> (founder) admit a joiner's device into the room
  /connect <ip:port>        LAN fallback: dial the other machine directly
  /listen <port>            LAN fallback: (re)start listening on a port
  /export [path]            export this room's transcript to a file
  /help                     this text
  /exit                     quit
`.trim()

/** startRepl(ctx) -> Promise<void>, resolves when the user exits. */
export async function startRepl(ctx) {
  const cmds = createCommandLayer(ctx)
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout, prompt: `${ctx.actor}> ` })

  ctx.log(`\nWelcome, ${ctx.actor}. /help for commands. /rooms to see what's here, /create <title> to found one.\n`)
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
}
