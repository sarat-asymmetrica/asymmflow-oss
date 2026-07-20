// bare-net.mjs — SC-2 (Sealed Corridor campaign): a Bare-native port of
// kit-net.mjs's network model (MSG-D24 — hyperswarm primary + direct-TCP
// REQUIRED fallback), same exported shape, same replication primitive
// (`corestore.replicate(socket)` piped onto a raw duplex — the SAME
// primitive mesh-node.mjs's own `connect()` and kit-net.mjs's Node original
// use; this file invents no new wire protocol, per campaign law).
//
// PORT MAP against kit-net.mjs (verbatim vs substituted, named):
//   - `node:net`                -> `bare-tcp` (the one pre-approved new
//     devDependency, mesh/package.json). API shape is a near drop-in
//     (Server/Socket/connect/createServer, `.pipe()`-capable Duplex) but
//     TWO real behavioral differences were found porting this file, both
//     fixed here and documented at the fix site, not glossed over:
//       1. `server.listen(port, cb)` — bare-tcp's `host` parameter DEFAULTS
//          TO `'localhost'` when omitted; node:net's default is "all
//          interfaces" (0.0.0.0). kit-net.mjs's own `listenTcp` omits host
//          entirely, relying on Node's default — porting that verbatim
//          under Bare would silently make the LAN fallback deaf to every
//          machine except itself (exactly the corridor's real job is to
//          accept a connection FROM another machine). Fixed by passing
//          `'0.0.0.0'` explicitly below.
//       2. bare-tcp's Socket is a `bare-stream` Duplex, not a Node
//          `net.Socket` — `.pipe()` exists and is Node-shaped, but whether
//          it interoperates with `node.store.replicate()`'s own duplex
//          (built on `streamx`, the corestore/hypercore stack's stream
//          library) is NOT something either file's own docs assert. This
//          file ports the `.pipe()` wiring VERBATIM (same as kit-net.mjs)
//          and the SC-2 gate (bare-net-spike.mjs) is what actually proves
//          bytes cross both ways — see that file's report for the verdict;
//          this header does not claim success it hasn't measured.
//   - `hypercore-crypto`/`hyperswarm` — verbatim, ZERO shim. Both already
//     proven Bare-clean by bare-probe.mjs (Phase 3, this same campaign's own
//     prior art) and re-confirmed by this file's own gate.
//
// SCOPE carried over unchanged from kit-net.mjs: hyperswarm routes an
// inbound connection to every joined room whose topic matched (or, absent a
// match, to every currently joined room, best-effort); the TCP fallback
// replicates exactly the ONE room passed to `listenTcp`/`connectTcp` at call
// time. See kit-net.mjs's own header for the full reasoning — unchanged
// here, this file does not revisit that design.

import tcp from 'bare-tcp'
import hcrypto from 'hypercore-crypto'
import Hyperswarm from 'hyperswarm'

/** Same contract as kit-net.mjs's roomTopic: the room key's discoveryKey,
 * never the raw base key (standard hyperswarm/hypercore hygiene — the base
 * key is a capability, not something announced to the public DHT). */
export function roomTopic(roomKeyHex) {
  return hcrypto.discoveryKey(Buffer.from(roomKeyHex, 'hex'))
}

/**
 * createNetwork({ useHyperswarm }) -> same shape as kit-net.mjs's
 * createNetwork: { mode, peerCount, swarmPeerCount, lastPeerSeenAt,
 * joinHyperswarm, listenTcp, connectTcp, close }.
 *
 * `useHyperswarm: false` gives the `--no-hyperswarm`-equivalent
 * degradation for a no-internet LAN — `ensureSwarm()` below never
 * constructs a `Hyperswarm` (and therefore never touches the DHT) when this
 * is false, exactly as kit-net.mjs's own `useHyperswarm` guard does.
 */
export function createNetwork({ useHyperswarm = true } = {}) {
  let swarm = null
  let swarmFailed = false
  const joinedTopics = new Map() // topicHex -> { roomKey, node }
  const tcpServers = new Map()   // port -> tcp.Server
  const tcpSockets = new Set()
  const roomSockets = new Map()  // node.key -> Set<socket>, TCP only
  const swarmSockets = new Set() // hyperswarm connections, swarm-wide
  let lastPeerSeenAt = null

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

    swarmPeerCount() {
      return swarmSockets.size
    },

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

    /** listenTcp(port, node, onPeer?) -> Promise<boundPort>. Port 0 asks the
     * OS to assign one; the resolved value is the ACTUAL bound port — same
     * discipline kit-repl.mjs's ensureTcpListen relies on (never assume the
     * requested port is the bound one). Binds '0.0.0.0' EXPLICITLY — see
     * file header, deviation 1: bare-tcp's own default ('localhost') would
     * silently make this deaf to every other machine on the LAN/WAN. */
    listenTcp(port, node, onPeer) {
      return new Promise((resolve, reject) => {
        const server = tcp.createServer((socket) => {
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
        server.listen(port, '0.0.0.0', () => {
          server.off('error', reject)
          tcpServers.set(server.address().port, server)
          resolve(server.address().port)
        })
      })
    },

    connectTcp(host, port, node) {
      return new Promise((resolve, reject) => {
        const socket = tcp.connect(port, host)
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
      // Sockets FIRST — same reasoning as kit-net.mjs's own close(): a
      // Server#close() callback does not fire until every accepted
      // connection has ended, so awaiting it before destroying those same
      // connections deadlocks forever.
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
