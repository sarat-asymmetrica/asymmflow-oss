// bridge-client.mjs — minimal client for the sidecar protocol v0 seam
// (bridge-server.mjs / mesh/docs/MESSENGER_UI_CAMPAIGN.md §1). Used by
// bridge-spike.mjs today; doubles as the reference shape for the frontend's
// future real transport (W-UI-2 wires `bridge/mesh.ts` against this same
// request/event contract, DP4 swaps localhost TCP for sidecar stdio without
// changing the shape).
//
// ndjson framing, mirroring the server: one JSON object per line.
//   Request:  {"id": n, "method": "...", "params": {...}}
//   Response: {"id": n, "ok": true, "result": ...} | {"id": n, "ok": false, "error": "..."}
//   Event:    {"event": "...", "params": {...}}   (no id; server-initiated)

import net from 'node:net'
import { EventEmitter } from 'node:events'

/**
 * createBridgeClient({ port, host, socket }) -> Promise<{
 *   request(method, params) -> Promise<result>,   // rejects with .bridgeError on {ok:false}
 *   on(event, cb) -> unsubscribe(),
 *   close() -> Promise<void>,
 *   socket,                                        // raw net.Socket, for host-side probes
 * }>
 */
export function createBridgeClient({ port, host = '127.0.0.1', socket } = {}) {
  return new Promise((resolve, reject) => {
    const sock = socket ? net.createConnection(socket) : net.createConnection(port, host)
    const emitter = new EventEmitter()
    let buf = ''
    let nextId = 1
    const pending = new Map() // id -> {resolve, reject}

    sock.on('data', (chunk) => {
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
          continue // malformed frame from the server side would be a server bug; client just drops it
        }
        if (msg && msg.event) {
          emitter.emit(msg.event, msg.params)
          continue
        }
        if (msg && Number.isInteger(msg.id) && pending.has(msg.id)) {
          const { resolve: res, reject: rej } = pending.get(msg.id)
          pending.delete(msg.id)
          if (msg.ok) res(msg.result)
          else rej(Object.assign(new Error(msg.error ?? 'bridge error'), { bridgeError: msg.error }))
        }
      }
    })

    function onConnectError(err) { reject(err) }
    sock.once('error', onConnectError)
    sock.once('connect', () => {
      sock.off('error', onConnectError)
      sock.on('error', (err) => emitter.emit('error', err))
      resolve({
        request(method, params = {}) {
          return new Promise((res, rej) => {
            const id = nextId++
            pending.set(id, { resolve: res, reject: rej })
            sock.write(JSON.stringify({ id, method, params }) + '\n')
          })
        },
        on(event, cb) {
          emitter.on(event, cb)
          return () => emitter.off(event, cb)
        },
        close() {
          return new Promise((res) => sock.end(res))
        },
        socket: sock,
      })
    })
  })
}
