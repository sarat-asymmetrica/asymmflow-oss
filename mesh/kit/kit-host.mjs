// kit-host.mjs — Mission U1.5 "The Kitchen Table Kit": boots ONE device for
// the portable field kit (mesh/docs/MESSENGER_UI_CAMPAIGN.md §2 W-UI-1.5,
// MSG-D24). This is the entry point run_mesh.cmd launches on each machine.
//
// Boots, in order:
//   1. a persistent device identity in `data/keys/` — a SIBLING of
//      `data/corestore/`, never inside it (the lesson peer.mjs already paid
//      for: Corestore deletes foreign files inside its own storage tree).
//   2. `rooms` Map + bridge-server.mjs over localhost (protocol v0, MESSENGER_
//      UI_CAMPAIGN.md §1) — the kit's REPL talks to its OWN host through the
//      REAL seam, same shape a future frontend/DP4 sidecar will use.
//   3. kit-net.mjs (hyperswarm primary + REQUIRED direct-TCP fallback) for
//      REAL replication between the two machines (MSG-D24: first packets
//      leave localhost; content is encrypted end-to-end regardless).
//   4. hands off to kit-repl.mjs's startRepl(ctx) — same process, not a
//      second one: the cold-peer onboarding ceremony (GL-9's flagged gap)
//      needs `/addwriter` to reach the founder's raw MeshNode via
//      `server.rooms` directly, which only works in-process (protocol v0
//      deliberately has no wire method for becoming a writer — bridge-
//      server.mjs deviation #2, and MESSENGER_UI_CAMPAIGN.md §2 says this
//      mission solves the ceremony, not the protocol).
//
// createKitHost() is exported separately (headless, no REPL) so kit-spike.mjs
// can drive two instances programmatically through the SAME command layer
// the REPL uses (factored in kit-repl.mjs), never screen-scraping readline.
//
// CLI (interactive use — run_mesh.cmd calls this with no flags, --data is
// the only one a packaged kit sets):
//   node kit-host.mjs [--data DIR] [--actor NAME] [--bridge-port N]
//                      [--no-hyperswarm]
//   --data DIR         data root (default ./data). Creates DIR/keys and
//                       DIR/corestore as siblings.
//   --actor NAME        this device's actor label. First run only — once
//                       data/keys/actor.txt exists it wins (a device's actor
//                       name is chosen ONCE, at first run, per MSG-D24).
//                       Omit both this flag and a prior run to be prompted.
//   --bridge-port N     bridge-server TCP port (default 0 = OS-assigned).
//   --no-hyperswarm     disable the DHT path; TCP fallback only (useful on
//                       a machine with no internet — the kit still works).

import readline from 'node:readline'
import { randomBytes } from 'node:crypto'
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { deviceKeys } from '../host/capability.mjs'
import { createMeshNode } from '../host/mesh-node.mjs'
import { createBridgeServer } from '../host/bridge-server.mjs'
import { createBridgeClient } from '../host/bridge-client.mjs'
import { createNetwork } from './kit-net.mjs'
import { startRepl } from './kit-repl.mjs'
import { loadRoomRegistry } from './kit-registry.mjs'

/** "ip:port" -> [host, port] | null. Defensive: a hand-edited or malformed
 * registry entry must never crash boot (kit-registry.mjs's own standard). */
function parsePeerAddr(addr) {
  if (typeof addr !== 'string' || !addr.includes(':')) return null
  const idx = addr.lastIndexOf(':')
  const host = addr.slice(0, idx)
  const port = Number(addr.slice(idx + 1))
  if (!host || !Number.isInteger(port) || port <= 0) return null
  return [host, port]
}

/** Load-or-create a persistent 32-byte seed. Same shape as peer.mjs's own
 * persistentSeed — proven precedent, not reinvented here. */
function persistentSeed(file) {
  if (existsSync(file)) return Buffer.from(readFileSync(file, 'utf8').trim(), 'hex')
  const seed = randomBytes(32)
  writeFileSync(file, seed.toString('hex') + '\n')
  return seed
}

/** Load-or-create this device's actor label. First run wins; the file is
 * the source of truth from then on (a re-run with a different --actor does
 * NOT silently rename the device out from under its own room history). */
async function persistentActor(file, requested) {
  if (existsSync(file)) return readFileSync(file, 'utf8').trim()
  if (requested) {
    writeFileSync(file, requested + '\n')
    return requested
  }
  if (!process.stdin.isTTY) {
    throw new Error(`no actor name on file at ${file} and no --actor given, and stdin is not interactive — pass --actor NAME`)
  }
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout })
  const name = await new Promise((resolve) => {
    rl.question('This is a NEW device — choose a synthetic actor name (e.g. "ana", "sam"): ', resolve)
  })
  rl.close()
  const trimmed = name.trim() || `device-${Date.now()}`
  writeFileSync(file, trimmed + '\n')
  return trimmed
}

/**
 * createKitHost({ dataDir, actor, bridgePort, useHyperswarm, log }) -> ctx
 *
 * Headless bootstrap — no REPL. Returns the same `ctx` shape kit-repl.mjs's
 * command layer expects, so kit-spike.mjs can drive the ceremony
 * programmatically and run_mesh.cmd's interactive path (below) can hand the
 * identical object to startRepl().
 */
export async function createKitHost({
  dataDir, actor: requestedActor, bridgePort = 0, tcpPort, useHyperswarm = true, log = (m) => console.log(m),
} = {}) {
  if (!dataDir) throw new Error('createKitHost requires dataDir')
  const keysDir = join(dataDir, 'keys')
  const corestoreDir = join(dataDir, 'corestore')
  mkdirSync(keysDir, { recursive: true })
  mkdirSync(corestoreDir, { recursive: true })

  const keys = deviceKeys(persistentSeed(join(keysDir, 'device-seed.hex')))
  const actor = await persistentActor(join(keysDir, 'actor.txt'), requestedActor)

  const server = await createBridgeServer({ actor, deviceKeys: keys, storageDir: corestoreDir, port: bridgePort })
  const client = await createBridgeClient({ port: server.port })
  const net = createNetwork({ useHyperswarm })

  // GL-5 reopen discipline: every room this device previously created or
  // joined comes back automatically — a restart must not orphan a room the
  // human already has open on the other machine. See kit-registry.mjs.
  //
  // F2 (MSG-D25) auto-reconnect: each reopened room also (a) best-effort
  // rejoins hyperswarm on its own topic, and (b) if the registry has a
  // lastPeer, tries dialing it over the TCP fallback RIGHT NOW — a
  // restarted device should not need a human to re-run /connect just
  // because the process bounced. Either outcome is printed; a failed
  // auto-reconnect is not an error (the peer may not be up yet — kit-repl's
  // /connect and hyperswarm both keep trying independently).
  const reopened = []
  for (const entry of loadRoomRegistry(keysDir)) {
    try {
      const node = await createMeshNode({
        storage: join(corestoreDir, entry.storage),
        bootstrap: entry.bootstrap || null,
        authorityPub: entry.authorityPub,
        mode: 'room',
        encryptionKey: entry.encryptionKey ? Buffer.from(entry.encryptionKey, 'hex') : undefined,
      })
      server.registerRoom(node.key, node)
      log(`reopened room "${entry.title ?? ''}" — ${node.key.slice(0, 16)}…`)
      reopened.push(node.key)
      net.joinHyperswarm(node.key, node) // best-effort DHT rejoin; never throws
      const parsed = parsePeerAddr(entry.lastPeer)
      if (parsed) {
        const [host, port] = parsed
        try {
          await net.connectTcp(host, port, node)
          log(`  auto-reconnected to last-known peer ${entry.lastPeer}`)
        } catch (err) {
          log(`  could not auto-reconnect to ${entry.lastPeer} yet (${err.message}) — /connect ${entry.lastPeer} to retry by hand, or wait for hyperswarm`)
        }
      }
    } catch (err) {
      log(`could not reopen a registered room (${entry.roomKey?.slice(0, 16)}…): ${err.message}`)
    }
  }

  const hello = await client.request('hello')
  log(`device ready — actor "${actor}", devicePub ${hello.devicePub.slice(0, 16)}…, bridge on 127.0.0.1:${server.port}`)

  return {
    dataDir, keysDir, corestoreDir, actor, keys, tcpPort,
    server, client, net, log,
    // Kitchen-table UX (deliverable 4): the kit's real use case is ONE
    // shared room per device, so if exactly one came back at boot, open it
    // automatically — a receptionist who restarts mid-conversation should
    // see their conversation, not an empty prompt demanding /open. Multiple
    // rooms stay unopened (ambiguous which one "current" means); zero rooms
    // stays null (nothing to open yet).
    currentRoomKey: reopened.length === 1 ? reopened[0] : null,
    async close() {
      await net.close()
      await client.close()
      await server.close()
      for (const node of server.rooms.values()) await node.close().catch(() => {})
    },
  }
}

// ── CLI entry point ──────────────────────────────────────────────────────
const isMain = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]
if (isMain) {
  const args = {}
  const rest = process.argv.slice(2)
  for (let i = 0; i < rest.length; i++) {
    const a = rest[i]
    if (!a.startsWith('--')) continue
    const key = a.replace(/^--/, '')
    const next = rest[i + 1]
    if (next === undefined || next.startsWith('--')) { args[key] = true } else { args[key] = next; i++ }
  }

  const ctx = await createKitHost({
    dataDir: args.data || './data',
    actor: typeof args.actor === 'string' ? args.actor : undefined,
    bridgePort: args['bridge-port'] ? Number(args['bridge-port']) : 0,
    tcpPort: args['tcp-port'] ? Number(args['tcp-port']) : undefined,
    useHyperswarm: !args['no-hyperswarm'],
  })

  process.on('SIGINT', async () => { await ctx.close(); process.exit(0) })
  await startRepl(ctx)
  await ctx.close()
}
