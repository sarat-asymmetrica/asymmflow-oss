// peer.mjs — a standalone Sovereign Mesh peer process, driven over stdin/stdout
// with JSON lines. Wave-1 stage-2 unit (real Corestore on disk, real Autobase,
// real Holesail DHT tunnel), extended for Mission D: persistent Ed25519 device
// identities, an authority mode, and signed ops with grant/epoch/revoke.
//
// Transport doctrine (Mission D): the peer speaks raw Corestore replication
// over a plain TCP socket; Holesail carries that socket over the DHT.
// Transport-auth (the hs:// connector) and the Autobase writer set are the
// REPLICATION plane. Capability-auth is the Ed25519 grant-with-epochs layer
// enforced inside the reducer — a peer can be fully replicating and still have
// every op rejected on every honest peer ("the pipe is open, the grant is dead").
//
// Roles:
//   node peer.mjs host --storage DIR [--tcp-port 49222] [--authority]
//     Founds (or reopens) a base, listens for replication on 127.0.0.1:PORT,
//     shares it via a secure Holesail tunnel.
//     --authority: this peer holds the MESH AUTHORITY keypair (persisted in
//     DIR-keys/authority-seed.hex — a SIBLING dir; Corestore wipes foreign
//     files inside DIR) and capability enforcement is ON. The printed
//     `authorityPub` must be handed to joiners (like url + baseKey).
//     Prints {event:'ready', url, baseKey, writerKey, devicePub, authorityPub?}.
//
//   node peer.mjs join --storage DIR --url hs://... --base-key HEX
//                      [--authority-pub HEX] [--tcp-port 49223]
//     Connects the Holesail client to a local port, dials it, replicates.
//     --authority-pub: the host's authority key — REQUIRED to fold a
//     capability-enforced mesh (it is mesh-genesis config, not discovered).
//     Prints {event:'ready', baseKey, writerKey, devicePub} then
//     {event:'writable'} once granted into the writer set.
//
// Every peer keeps a persistent device identity in DIR-keys/device-seed.hex
// (created on first run; the authority peer's device IS the authority key).
//
// Stdin commands (one per line):
//   add-writer <hex>    admit a writer to the REPLICATION plane (linearizer)
//   grant <pub> [epoch] authority only: capability-grant a device (default: current epoch)
//   epoch <n>           authority only: bump the grant epoch (revocation wave —
//                       devices not re-granted at n go stale on every peer)
//   revoke <pub>        authority only: targeted capability revocation
//   append <json-op>    append one op, AUTO-SIGNED with this peer's device key
//                       when the mesh is capability-enforced
//   append-raw <json>   append WITHOUT signing (ceremony demo: watch the
//                       reducer reject it everywhere as "unsigned op")
//   whoami              -> {event:'whoami', devicePub, writerKey, authority}
//   digest              -> {event:'digest', viewLength, viewDigest, stateDigest,
//                           stock, rejected, capEpoch, grants}
//   exit                close cleanly
//
// Two-machine Mission D ceremony (box 1 = authority hub, box 2 = the laptop):
//   box 1:  npm run missiond:host           → copy url, baseKey, authorityPub
//   box 2:  npm run missiond:join -- --url <hs://…> --base-key <hex> --authority-pub <hex>
//   box 1:  add-writer <box2 writerKey>     (replication plane)
//   box 2:  append {"actor":"laptop","sku":"TX-100","delta":5}
//           → digest on both: REJECTED ("no grant for device") — pipe open, no capability
//   box 1:  grant <box2 devicePub>          (capability plane)
//   box 2:  append {"actor":"laptop","sku":"TX-100","delta":5}   → applied on both
//   box 1:  epoch 1                         → box 2's next append: stale-rejected everywhere
//   box 1:  grant <box2 devicePub> 1        → re-issued; box 2 writes again
// Ceremony seq/ts: omit them — the peer fills seq = ts = Date.now() at CREATION
// time (event data; the reducer never reads a clock). The reducer folds in
// canonical (seq, actor, …) order, so wall-clock-millis seqs make fold order
// match the real-time order of a human-paced ceremony (clock skew between the
// boxes only matters within the skew window — irrelevant at human pace).
// Ceremony-grade, not goldenable; goldened runs pin explicit seq/ts.

import net from 'node:net'
import readline from 'node:readline'
import { randomBytes } from 'node:crypto'
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs'
import { join as joinPath } from 'node:path'
import Holesail from 'holesail'
import { createMeshNode } from './mesh-node.mjs'
import { deviceKeys, signOp, grantOp, epochOp, revokeOp } from './capability.mjs'

const [role, ...rest] = process.argv.slice(2)
const args = {}
for (let i = 0; i < rest.length; i += 2) {
  const key = rest[i]?.replace(/^--/, '')
  const val = rest[i + 1]
  if (val === undefined || val.startsWith('--')) { args[key] = true; i -= 1 } // bare flag
  else args[key] = val
}

const out = (obj) => process.stdout.write(JSON.stringify(obj) + '\n')

if (!['host', 'join'].includes(role) || !args.storage || typeof args.storage !== 'string') {
  out({ event: 'error', error: 'usage: peer.mjs host|join --storage DIR [--tcp-port N] [--authority] [--url HS] [--base-key HEX] [--authority-pub HEX]' })
  process.exit(2)
}

/** Load-or-create a persistent 32-byte seed for this peer's keys. */
function persistentSeed(file) {
  if (existsSync(file)) return Buffer.from(readFileSync(file, 'utf8').trim(), 'hex')
  const seed = randomBytes(32)
  writeFileSync(file, seed.toString('hex') + '\n')
  return seed
}

// Key material lives in a SIBLING dir, never inside the Corestore dir:
// Corestore owns its storage tree and DELETES foreign files on init
// (verified against corestore 7.x — a seed stored inside is silently wiped).
const keysDir = args.storage.replace(/[\\/]+$/, '') + '-keys'
mkdirSync(keysDir, { recursive: true })
const isAuthority = role === 'host' && !!args.authority
const keys = isAuthority
  ? deviceKeys(persistentSeed(joinPath(keysDir, 'authority-seed.hex')))
  : deviceKeys(persistentSeed(joinPath(keysDir, 'device-seed.hex')))
const authorityPub = isAuthority ? keys.pubHex : (args['authority-pub'] || null)
const capability = !!authorityPub // capability enforcement on for this mesh?

const tcpPort = Number(args['tcp-port'] || (role === 'host' ? 49222 : 49223))
const sockets = new Set()
let node, tunnel, server
// Ceremony ops carry seq = ts = Date.now() when not given explicitly: event
// data stamped at CREATION (never read inside the fold), and wall-clock-millis
// seqs make canonical fold order track the real-time order of a live ceremony.
const stamp = (op) => {
  const now = Date.now()
  return { seq: now, ts: now, ...op }
}

if (role === 'host') {
  node = await createMeshNode({ storage: args.storage, authorityPub })

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

  out({
    event: 'ready', role, url: tunnel.info.url, baseKey: node.key,
    writerKey: node.writerKey, devicePub: keys.pubHex,
    ...(isAuthority ? { authorityPub } : {}),
  })
} else {
  tunnel = new Holesail({ client: true, key: args.url, port: tcpPort, host: '127.0.0.1' })
  await tunnel.ready()

  node = await createMeshNode({ storage: args.storage, bootstrap: args['base-key'], authorityPub })

  const socket = net.connect(tcpPort, '127.0.0.1')
  sockets.add(socket)
  const rs = node.store.replicate(true)
  socket.pipe(rs).pipe(socket)
  socket.on('error', (e) => out({ event: 'transport-error', error: String(e) }))

  out({ event: 'ready', role, baseKey: node.key, writerKey: node.writerKey, devicePub: keys.pubHex })
}

node.base.on('writable', () => out({ event: 'writable' }))
if (node.writable) out({ event: 'writable' })

const requireAuthority = (cmd) => {
  if (!isAuthority) throw new Error(`${cmd}: this peer does not hold the mesh authority key`)
}
const actorName = () => (isAuthority ? 'authority' : 'device')

// Appending needs WRITER-SET membership (Autobase-level, distinct from the
// capability grant). "Not writable" raw from Autobase reads like a crash to a
// first-timer — turn it into instructions they can act on without help.
const requireWritable = () => {
  if (node.writable) return
  throw new Error(
    `this peer is not in the writer set yet — wait for {"event":"writable"}. ` +
    `On the HOST, run: add-writer ${node.writerKey} ` +
    `(this peer's CURRENT writerKey — it changes whenever the storage folder is recreated, ` +
    `so re-run add-writer after any fresh start). If it never arrives, the url/baseKey pasted at ` +
    `join may be from an older host session — restart JOIN with the values the host is printing NOW.`,
  )
}

// Humans paste keys wrapped in <>, quotes, or with stray whitespace — strip
// the decoration, then insist on bare 64-hex with a message a human can act on.
const cleanKey = (cmd, raw) => {
  const key = String(raw ?? '').replace(/[<>"'`\s]/g, '')
  if (!/^[0-9a-fA-F]{64}$/.test(key)) {
    throw new Error(`${cmd}: expected a 64-hex-char key, got ${JSON.stringify(raw)} — paste the bare hex only`)
  }
  return key.toLowerCase()
}

const rl = readline.createInterface({ input: process.stdin })
rl.on('line', async (line) => {
  const [cmd, ...argParts] = line.trim().split(' ')
  const arg = argParts.join(' ')
  try {
    if (cmd === 'add-writer') {
      await node.addWriter(cleanKey(cmd, arg))
      out({ event: 'ok', cmd })
    } else if (cmd === 'grant') {
      requireAuthority(cmd)
      const [deviceRaw, epochStr] = argParts
      const device = cleanKey(cmd, deviceRaw)
      const epoch = epochStr !== undefined ? Number(epochStr) : (await node.state()).capEpoch ?? 0
      await node.append(grantOp(stamp({ actor: actorName(), device, epoch }), keys))
      out({ event: 'ok', cmd, device, epoch })
    } else if (cmd === 'epoch') {
      requireAuthority(cmd)
      await node.append(epochOp(stamp({ actor: actorName(), epoch: Number(arg) }), keys))
      out({ event: 'ok', cmd, epoch: Number(arg) })
    } else if (cmd === 'revoke') {
      requireAuthority(cmd)
      const device = cleanKey(cmd, arg)
      await node.append(revokeOp(stamp({ actor: actorName(), device }), keys))
      out({ event: 'ok', cmd, device })
    } else if (cmd === 'append') {
      requireWritable()
      const op = stamp(JSON.parse(arg))
      await node.append(capability ? signOp(op, keys) : op)
      out({ event: 'ok', cmd, signed: capability, seq: op.seq })
    } else if (cmd === 'append-raw') {
      requireWritable()
      const op = stamp(JSON.parse(arg)) // deliberately unsigned — the reducer will reject it
      await node.append(op)
      out({ event: 'ok', cmd, signed: false, seq: op.seq })
    } else if (cmd === 'whoami') {
      out({ event: 'whoami', devicePub: keys.pubHex, writerKey: node.writerKey, authority: isAuthority, capability })
    } else if (cmd === 'digest') {
      const [viewDigest, state] = [await node.viewDigest(), await node.state()]
      out({
        event: 'digest',
        viewLength: (await node.ops()).length,
        viewDigest,
        stateDigest: state.digest,
        stock: state.stock,
        rejected: state.rejected,
        ...(capability ? { capEpoch: state.capEpoch ?? 0, grants: state.grants ?? {} } : {}),
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
