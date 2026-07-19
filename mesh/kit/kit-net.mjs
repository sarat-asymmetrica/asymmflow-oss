// kit-net.mjs — Mission U1.5 "The Kitchen Table Kit": REAL networking between
// two machines (MSG-D24 — first packets leave localhost; content stays
// encrypted end-to-end regardless of transport).
//
// Two paths, both wired to the SAME primitive every spike in this codebase
// already uses — `corestore.replicate(socket)` piped onto a raw duplex
// stream (mesh-node.mjs `connect()`, peer.mjs's host/join sockets):
//
//   1. hyperswarm (primary, best-effort): DHT-based discovery by the room's
//      OWN discoveryKey (never the raw base key — standard hyperswarm
//      hygiene, verified against the installed source: hyperswarm/index.js
//      `join(topic, opts)` at line 501 takes a 32-byte topic and
//      `PeerInfo.topics` (hyperswarm/lib/peer-info.js:33/101) records which
//      topic(s) matched a connection, so a single swarm instance can safely
//      host multiple rooms and route each incoming socket to the room(s) it
//      was discovered for).
//   2. direct TCP (REQUIRED fallback): `--listen <port>` / connect-to-ip,
//      exactly peer.mjs's own host/join shape — no DHT dependency at all.
//      This is the path the kitchen-table test must not depend on internet
//      DHT health for (two machines on the same LAN, or connected by cable).
//
// SCOPE, declared: hyperswarm routes an incoming connection to EVERY joined
// room whose topic matched (or, if none matched — a peer that dialed us
// directly rather than through discovery — to every currently joined room,
// best-effort). The TCP fallback is simpler and stricter: one listener
// replicates ONE room (whichever the caller passes at `listenTcp` time) — a
// kitchen-table kit's real ceremony is a founder and one joiner building ONE
// shared room, so this is not a limitation that bites the actual use case;
// a future multi-room LAN fallback would need a room-selection handshake
// this module deliberately doesn't invent.
//
// Determinism note (kitspike, GL-2/GL-5 canon): kit-spike.mjs drives ONLY
// the TCP fallback — hermetic, localhost, no DHT bootstrap dependency. The
// hyperswarm path is real production infrastructure for the actual
// two-machine test and is exercised by hand at the kitchen table, not by
// the gate (see kit-spike.mjs header for the full honest-scope note).

import net from 'node:net'
import hcrypto from 'hypercore-crypto'
import Hyperswarm from 'hyperswarm'

/** The DHT discovery topic for a room — its key's discoveryKey, never the
 * raw base key (standard hyperswarm/hypercore hygiene: the base key itself
 * must stay a capability, not something announced to the public DHT). */
export function roomTopic(roomKeyHex) {
  return hcrypto.discoveryKey(Buffer.from(roomKeyHex, 'hex'))
}

/**
 * createNetwork({ useHyperswarm }) -> {
 *   mode,                          // 'hyperswarm' | 'tcp' | 'none' (best current guess)
 *   joinHyperswarm(roomKey, node) -> boolean   (true if DHT join attempted)
 *   listenTcp(port, node, onPeer?) -> Promise<boundPort>
 *   connectTcp(host, port, node) -> Promise<socket>
 *   peerCount(roomKey) -> number   (U1.6 /status: live TCP replication sockets for this room)
 *   close() -> Promise<void>
 * }
 *
 * useHyperswarm defaults true (primary path); the kit always ALSO wires TCP
 * on request — the two paths are independent and both may be active for the
 * same room (redundant replication is harmless; Autobase/Hypercore dedupe).
 *
 * U1.6: `listenTcp`'s optional `onPeer(remoteAddress)` fires once per
 * ACCEPTED inbound socket — kit-repl.mjs's ensureTcpListen wires this to
 * kit-registry.mjs's updateRoomRegistryPeer (the F2 auto-reconnect fix).
 * `peerCount` backs the new /status command; it counts TCP sockets only
 * (per room, via node.key) — hyperswarm's own per-topic connection count
 * isn't tracked here (multiple rooms can share one swarm connection, so a
 * per-room count would be a fiction; /status reports hyperswarm reachability
 * as the general `mode` instead, honestly, not a fabricated per-room number).
 */
export function createNetwork({ useHyperswarm = true } = {}) {
  let swarm = null
  let swarmFailed = false
  const joinedTopics = new Map() // topicHex -> { roomKey, node }
  const tcpServers = new Map()   // port -> net.Server
  const tcpSockets = new Set()
  const roomSockets = new Map()  // node.key -> Set<socket>, TCP only (see peerCount doc above)
  const swarmSockets = new Set() // hyperswarm connections, swarm-wide (Mission A2 Band 3 addition)
  let lastPeerSeenAt = null      // ms epoch of the most recent connect (either transport) — Mission A2 Band 3 addition, backs kit-repl.mjs's /status "last-seen peer" line

  function trackRoomSocket(node, socket) {
    if (!roomSockets.has(node.key)) roomSockets.set(node.key, new Set())
    roomSockets.get(node.key).add(socket)
    return () => { roomSockets.get(node.key)?.delete(socket) }
  }

  function ensureSwarm() {
    if (swarm || swarmFailed || !useHyperswarm) return swarm
    try {
      swarm = new Hyperswarm()
      swarm.on('connection', (socket, peerInfo) => {
        swarmSockets.add(socket)
        lastPeerSeenAt = Date.now()
        socket.once('close', () => swarmSockets.delete(socket))
        const topicHexes = (peerInfo.topics || []).map((t) => t.toString('hex'))
        const matched = topicHexes.map((h) => joinedTopics.get(h)).filter(Boolean)
        // A peer that dialed us directly (joinPeer, or a topic mismatch we
        // can't yet see) still deserves a replication attempt — best effort,
        // documented above.
        const targets = matched.length ? matched : [...joinedTopics.values()]
        for (const { node } of targets) {
          try { node.store.replicate(socket) } catch { /* best-effort: TCP fallback carries the kit */ }
        }
      })
      swarm.on('error', () => { /* DHT hiccups are expected off-network; TCP fallback is REQUIRED for exactly this reason */ })
    } catch {
      swarm = null
      swarmFailed = true
    }
    return swarm
  }

  return {
    get mode() {
      if (swarm && joinedTopics.size) return 'hyperswarm'
      if (tcpServers.size || tcpSockets.size) return 'tcp'
      return 'none'
    },

    peerCount(roomKey) {
      return roomSockets.get(roomKey)?.size ?? 0
    },

    /** Mission A2 Band 3 addition: hyperswarm connections are swarm-wide,
     * not per-room (same reasoning as peerCount's own doc above — a
     * connection isn't attributable to one room without screen-scraping
     * peerInfo.topics, which the mission's own doctrine treats as an
     * honesty risk not worth taking for a support/heartbeat number). */
    swarmPeerCount() {
      return swarmSockets.size
    },

    /** Mission A2 Band 3 addition: ms epoch of the most recent accepted
     * connection over EITHER transport, or null if none yet. Backs
     * kit-repl.mjs's /status "last-seen peer" line and anchor.mjs's
     * heartbeat. */
    get lastPeerSeenAt() {
      return lastPeerSeenAt
    },

    joinHyperswarm(roomKey, node) {
      const s = ensureSwarm()
      if (!s) return false
      const topic = roomTopic(roomKey)
      const hex = topic.toString('hex')
      if (joinedTopics.has(hex)) return true
      joinedTopics.set(hex, { roomKey, node })
      try {
        s.join(topic, { server: true, client: true })
        return true
      } catch {
        joinedTopics.delete(hex)
        return false
      }
    },

    listenTcp(port, node, onPeer) {
      return new Promise((resolve, reject) => {
        const server = net.createServer((socket) => {
          tcpSockets.add(socket)
          lastPeerSeenAt = Date.now()
          const untrack = trackRoomSocket(node, socket)
          const rs = node.store.replicate(false)
          socket.pipe(rs).pipe(socket)
          const drop = () => { tcpSockets.delete(socket); untrack(); rs.destroy(); socket.destroy() }
          socket.on('close', drop); socket.on('error', drop); rs.on('error', drop)
          if (onPeer) { try { onPeer(socket.remoteAddress) } catch { /* onPeer failures never break replication */ } }
        })
        server.once('error', reject)
        server.listen(port, () => {
          server.off('error', reject)
          tcpServers.set(server.address().port, server)
          resolve(server.address().port)
        })
      })
    },

    connectTcp(host, port, node) {
      return new Promise((resolve, reject) => {
        const socket = net.connect(port, host)
        socket.once('error', reject)
        socket.once('connect', () => {
          socket.off('error', reject)
          tcpSockets.add(socket)
          lastPeerSeenAt = Date.now()
          const untrack = trackRoomSocket(node, socket)
          const rs = node.store.replicate(true)
          socket.pipe(rs).pipe(socket)
          const drop = () => { tcpSockets.delete(socket); untrack(); rs.destroy() }
          socket.on('close', drop); socket.on('error', drop); rs.on('error', drop)
          resolve(socket)
        })
      })
    },

    async close() {
      // Sockets FIRST: net.Server#close()'s callback does not fire until
      // every accepted connection has ended — awaiting it before destroying
      // those same connections deadlocks close() forever.
      for (const socket of tcpSockets) socket.destroy()
      tcpSockets.clear()
      roomSockets.clear()
      for (const server of tcpServers.values()) await new Promise((r) => server.close(r))
      tcpServers.clear()
      joinedTopics.clear()
      swarmSockets.clear()
      if (swarm) { try { await swarm.destroy() } catch { /* best-effort teardown */ } }
      swarm = null
    },
  }
}
