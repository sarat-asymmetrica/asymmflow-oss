// peer.mjs — a standalone Sovereign Mesh peer process, driven over stdin/stdout
// with JSON lines. This is the Wave-1 stage-2 unit: real Corestore on disk,
// real Autobase, replicating over a REAL Holesail DHT tunnel.
//
// Transport doctrine (Mission D): the peer speaks raw Corestore replication
// over a plain TCP socket; Holesail carries that socket over the DHT.
// Transport-auth (the hs:// connector) stays strictly separate from
// capability-auth (addWriter grants through the linearizer).
//
// Roles:
//   node peer.mjs host --storage DIR [--tcp-port 49222]
//     Founds (or reopens) a base, listens for replication on 127.0.0.1:PORT,
//     shares it via a secure Holesail tunnel. Prints {event:'ready', url, baseKey, writerKey}.
//
//   node peer.mjs join --storage DIR --url hs://... --base-key HEX [--tcp-port 49223]
//     Connects the Holesail client to a local port, dials it, replicates.
//     Prints {event:'ready', baseKey, writerKey} then {event:'writable'} once granted.
//
// Stdin commands (one per line):
//   add-writer <hex>   grant a writer through the linearizer
//   append <json-op>   append one inventory op
//   digest             -> {event:'digest', viewLength, viewDigest, stateDigest, stock, rejected}
//   exit               close cleanly
//
// Two-machine run (the real Mission A finale): run `host` on machine 1, copy the
// printed url + baseKey to machine 2, run `join`, grant, append, compare digests.

import net from 'node:net'
import readline from 'node:readline'
import Holesail from 'holesail'
import { createMeshNode } from './mesh-node.mjs'

const [role, ...rest] = process.argv.slice(2)
const args = {}
for (let i = 0; i < rest.length; i += 2) args[rest[i].replace(/^--/, '')] = rest[i + 1]

const out = (obj) => process.stdout.write(JSON.stringify(obj) + '\n')

if (!['host', 'join'].includes(role) || !args.storage) {
  out({ event: 'error', error: 'usage: peer.mjs host|join --storage DIR [--tcp-port N] [--url HS] [--base-key HEX]' })
  process.exit(2)
}

const tcpPort = Number(args['tcp-port'] || (role === 'host' ? 49222 : 49223))
const sockets = new Set()
let node, tunnel, server

if (role === 'host') {
  node = await createMeshNode({ storage: args.storage })

  server = net.createServer((socket) => {
    sockets.add(socket)
    const rs = node.store.replicate(false)
    socket.pipe(rs).pipe(socket)
    const drop = () => { sockets.delete(socket); rs.destroy(); socket.destroy() }
    socket.on('error', drop); socket.on('close', drop); rs.on('error', drop)
  })
  await new Promise((resolve) => server.listen(tcpPort, '127.0.0.1', resolve))

  tunnel = new Holesail({ server: true, port: tcpPort, host: '127.0.0.1', secure: true })
  await tunnel.ready()

  out({ event: 'ready', role, url: tunnel.info.url, baseKey: node.key, writerKey: node.writerKey })
} else {
  tunnel = new Holesail({ client: true, key: args.url, port: tcpPort, host: '127.0.0.1' })
  await tunnel.ready()

  node = await createMeshNode({ storage: args.storage, bootstrap: args['base-key'] })

  const socket = net.connect(tcpPort, '127.0.0.1')
  sockets.add(socket)
  const rs = node.store.replicate(true)
  socket.pipe(rs).pipe(socket)
  socket.on('error', (e) => out({ event: 'transport-error', error: String(e) }))

  out({ event: 'ready', role, baseKey: node.key, writerKey: node.writerKey })
}

node.base.on('writable', () => out({ event: 'writable' }))
if (node.writable) out({ event: 'writable' })

const rl = readline.createInterface({ input: process.stdin })
rl.on('line', async (line) => {
  const [cmd, ...argParts] = line.trim().split(' ')
  const arg = argParts.join(' ')
  try {
    if (cmd === 'add-writer') {
      await node.addWriter(arg)
      out({ event: 'ok', cmd })
    } else if (cmd === 'append') {
      await node.append(JSON.parse(arg))
      out({ event: 'ok', cmd })
    } else if (cmd === 'digest') {
      const [viewDigest, state] = [await node.viewDigest(), await node.state()]
      out({
        event: 'digest',
        viewLength: (await node.ops()).length,
        viewDigest,
        stateDigest: state.digest,
        stock: state.stock,
        rejected: state.rejected,
      })
    } else if (cmd === 'exit') {
      for (const s of sockets) s.destroy()
      if (server) server.close()
      await tunnel.close()
      await node.close()
      process.exit(0)
    } else if (cmd) {
      out({ event: 'error', error: `unknown command: ${cmd}` })
    }
  } catch (e) {
    out({ event: 'error', cmd, error: String(e) })
  }
})
