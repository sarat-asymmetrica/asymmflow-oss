// anchor.mjs — Mission A2 "The Corridor", Band 3 "The Anchor" (mesh/docs/
// MISSION_A2_CORRIDOR_SPEC.md §Band 3). The receptionist machine's second
// role: a headless, always-on full peer so the mesh converges whenever the
// founder comes online, regardless of who slept when (R1: anchor over VPS).
//
// Reuses kit-host.mjs's createKitHost() UNCHANGED — that function already
// does exactly what item 1 of the band asks for: load every room from the
// kit registry (kit-registry.mjs), best-effort join hyperswarm per room,
// best-effort auto-reconnect over TCP to each room's last-known peer. This
// module adds the pieces createKitHost() deliberately does NOT do on its
// own: no REPL (headless), a resilient outer loop that survives boot/serve
// failures forever, a heartbeat file, and an optional multi-room TCP
// listener (kit-net.mjs's own `listenTcp` primitive, one call per room —
// see the --listen section below for why "one fixed port for every room"
// isn't literally what gets bound).
//
// RESILIENCE DESIGN (3 sentences, restated in the mission report): ordinary
// peer churn (a TCP socket dropping, hyperswarm losing and regaining the
// DHT) is already handled at the kit-net.mjs layer — sockets clean up their
// own tracking on 'close'/'error' and the TCP listener keeps accepting new
// connections, hyperswarm's own join() keeps retrying internally — so the
// anchor does NOT tear anything down for routine disconnects. What the
// anchor DOES guard against is BOOT-time or top-level failure (createKitHost
// throwing, an uncaught exception/rejection anywhere in the process): an
// outer loop retries the whole boot+serve cycle with capped exponential
// backoff, resetting the backoff after any clean boot, and only stops
// retrying on SIGINT/SIGTERM (clean teardown via ctx.close()). Every
// heartbeat tick (60s, plus immediately on an accepted --listen connection)
// also re-attempts hyperswarm join for any room not yet joined — cheap and
// idempotent (kit-net.mjs's joinHyperswarm no-ops if already joined) — so a
// room whose join failed transiently at boot gets more chances without a
// full restart.
//
// I7 (never log message plaintext or keys): the heartbeat line is exactly
// timestamp + counts + room COUNT (never room keys, titles, or message
// bodies) — see writeHeartbeat() below.
//
// CLI: node anchor.mjs [--data DIR] [--actor NAME] [--listen PORT]
//                       [--no-hyperswarm] [--heartbeat-interval MS]
//   --data DIR              data root (default ./data), same shape as
//                           kit-host.mjs (data/keys, data/corestore).
//   --actor NAME            device actor label (default "anchor" — a role
//                           label, not a person's name; first-run-wins per
//                           kit-host.mjs's own persistentActor()).
//   --listen PORT           ALSO listen with kit-net's TCP transport,
//                           starting at PORT, one bound listener per
//                           currently-loaded room (PORT, PORT+1, PORT+2, ...
//                           — see the deviation note below). Off by default.
//   --no-hyperswarm         disable the DHT path (hermetic/offline anchor).
//   --heartbeat-interval MS override the 60s heartbeat period (tests only).

import { appendFileSync, existsSync, mkdirSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createKitHost } from './kit-host.mjs'

const INITIAL_BACKOFF_MS = 5000
const MAX_BACKOFF_MS = 5 * 60 * 1000
const DEFAULT_HEARTBEAT_MS = 60 * 1000

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)) }
function nowIso() { return new Date().toISOString() }

/** Append one heartbeat line. I7: counts only — never a room key, title,
 * peer address, or message body. Creates the parent dir defensively (a
 * hand-deleted data/keys dir must not crash the anchor). */
function writeHeartbeat(logPath, { peers, roomCount, mode }) {
  try {
    mkdirSync(dirname(logPath), { recursive: true })
    appendFileSync(logPath, `${nowIso()} peers=${peers} rooms=${roomCount} mode=${mode}\n`)
  } catch { /* a heartbeat write failure must never crash the anchor — best effort */ }
}

/** Sum of TCP-tracked replication sockets across every loaded room, plus
 * the swarm-wide hyperswarm connection count (kit-net.mjs's own honesty
 * doctrine: hyperswarm connections aren't attributable to one room, so they
 * are reported as one swarm-wide number, never fabricated per room — see
 * kit-net.mjs's peerCount() doc and the new swarmPeerCount() this mission
 * added alongside it). */
function totalPeers(ctx) {
  let n = ctx.net.swarmPeerCount ? ctx.net.swarmPeerCount() : 0
  for (const roomKey of ctx.server.rooms.keys()) n += ctx.net.peerCount(roomKey)
  return n
}

/**
 * Bind kit-net's TCP transport for every currently-loaded room, starting at
 * `basePort`.
 *
 * DEVIATION from the spec's literal "listen on a fixed TCP port ... for all
 * rooms" (one port, many rooms): kit-net.mjs's listenTcp() is architected
 * one-listener-per-room BY DESIGN (its own header comment: "one listener
 * replicates ONE room" — a deliberate, documented scope limit, not an
 * oversight). Mission A2's invariant I1/I6 forbids reimplementing transport
 * primitives, and this band's dispatch explicitly says "reuse kit-net.mjs,
 * do not reimplement" — so rather than hand-rolling a second TCP server
 * that fans a socket out to N rooms (duplicating kit-net's own hyperswarm
 * multi-room dispatch logic outside kit-net.mjs, a worse violation of
 * "don't reimplement"), this binds ONE kit-net listener per room on
 * consecutive ports (basePort, basePort+1, ...). The corridor's real field
 * use is a SINGLE shared room (mesh/docs/MISSION_A2_CORRIDOR_SPEC.md §0),
 * so in practice this is indistinguishable from "the fixed port" — the
 * founder's `/connect ph-office.duckdns.org:PORT` still works unchanged.
 * Flagged here and in the final report for the gate to rule on if multi-room
 * anchors become real. */
async function bindListeners(ctx, basePort, log) {
  const bound = []
  let port = basePort
  for (const node of ctx.server.rooms.values()) {
    try {
      // eslint-disable-next-line no-await-in-loop
      const actual = await ctx.net.listenTcp(port, node, () => {
        writeHeartbeat(ctx._anchorHeartbeatPath, { peers: totalPeers(ctx), roomCount: ctx.server.rooms.size, mode: ctx.net.mode })
      })
      bound.push(actual)
      log(`listening (TCP) on port ${actual} for room ${node.key.slice(0, 12)}…`)
      port = actual + 1
    } catch (err) {
      log(`--listen: could not bind a port for a room (${err.message}) — that room stays reachable via hyperswarm only`)
    }
  }
  return bound
}

/** One full boot+serve cycle. Resolves when shutdown is requested; throws
 * on a boot-time failure (caller retries with backoff). `onBoot(ctx)` is a
 * test-only seam (mirrors kit-repl.mjs's own `onPairingCode` pattern) —
 * anchor-spike.mjs uses it to query the live headless ctx (room state,
 * peer counts) without exposing ctx from the public anchorMain() API, which
 * production callers (the CLI) have no legitimate use for. */
async function runOnce({ dataDir, actor, listenPort, useHyperswarm, heartbeatMs, log, isShuttingDown, onBoot }) {
  const ctx = await createKitHost({ dataDir, actor, useHyperswarm, tcpPort: 0, log })
  const heartbeatPath = join(ctx.keysDir, 'anchor.log')
  ctx._anchorHeartbeatPath = heartbeatPath

  log(`anchor ready — actor "${ctx.actor}", ${ctx.server.rooms.size} room(s) loaded, transport ${ctx.net.mode}`)
  writeHeartbeat(heartbeatPath, { peers: totalPeers(ctx), roomCount: ctx.server.rooms.size, mode: ctx.net.mode })

  const listeners = listenPort ? await bindListeners(ctx, listenPort, log) : []
  if (onBoot) { try { onBoot(ctx) } catch { /* test hook errors never affect the anchor */ } }

  // Poll for shutdown on a short tick (responsive SIGINT/SIGTERM) while
  // only WRITING a heartbeat / re-attempting joins on the slower
  // heartbeatMs cadence — a bare `sleep(heartbeatMs)` would make
  // requestShutdown() take up to a full heartbeat period to take effect,
  // which is fine for the default 60s in the field but far too slow for
  // anchor-spike.mjs's short-interval hermetic run.
  const pollMs = Math.min(500, heartbeatMs)
  try {
    let nextHeartbeatAt = Date.now() + heartbeatMs
    while (!isShuttingDown()) {
      // eslint-disable-next-line no-await-in-loop
      await sleep(pollMs)
      if (isShuttingDown()) break
      if (Date.now() < nextHeartbeatAt) continue
      nextHeartbeatAt = Date.now() + heartbeatMs
      // Defensive top-up: re-attempt hyperswarm join for any room that
      // isn't joined yet (idempotent — see kit-net.mjs's joinHyperswarm).
      // Covers a room whose join failed transiently at boot (DHT hiccup)
      // without needing a full anchor restart.
      for (const [roomKey, node] of ctx.server.rooms) ctx.net.joinHyperswarm(roomKey, node)
      writeHeartbeat(heartbeatPath, { peers: totalPeers(ctx), roomCount: ctx.server.rooms.size, mode: ctx.net.mode })
    }
  } finally {
    void listeners // listeners themselves are torn down by ctx.close() (net.close() closes every tcpServer)
    await ctx.close()
  }
}

/**
 * anchorMain(opts) -> { done, requestShutdown() }
 *
 * `done` resolves only after a clean shutdown. `requestShutdown()` is the
 * ONE way to stop the loop — the CLI wires it to SIGINT/SIGTERM;
 * anchor-spike.mjs calls it directly (no real signals needed) to prove
 * clean teardown hermetically. Returning the trigger from the SAME call
 * (rather than only accepting a pre-built AbortSignal) keeps both callers
 * symmetric: neither has to construct its own AbortController.
 */
export function anchorMain({
  dataDir, actor = 'anchor', listenPort, useHyperswarm = true,
  heartbeatMs = DEFAULT_HEARTBEAT_MS, log = (m) => console.log(`[anchor] ${m}`),
  onBoot, // test-only seam — see runOnce()'s doc; the CLI never passes this
} = {}) {
  if (!dataDir) throw new Error('anchorMain requires dataDir')
  mkdirSync(dataDir, { recursive: true })

  let shuttingDown = false
  const requestShutdown = () => { shuttingDown = true }
  const isShuttingDown = () => shuttingDown

  const done = (async () => {
    let backoff = INITIAL_BACKOFF_MS
    while (!shuttingDown) {
      try {
        await runOnce({ dataDir, actor, listenPort, useHyperswarm, heartbeatMs, log, isShuttingDown, onBoot })
        backoff = INITIAL_BACKOFF_MS // clean boot+serve cycle — reset backoff
      } catch (err) {
        log(`boot/serve error: ${err.message} — retrying in ${backoff}ms`)
      }
      if (shuttingDown) break
      // eslint-disable-next-line no-await-in-loop
      await sleep(backoff)
      backoff = Math.min(backoff * 2, MAX_BACKOFF_MS)
    }
    log('shutdown complete')
  })()

  return { done, requestShutdown }
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

  const dataDir = args.data || './data'
  if (!existsSync(dataDir)) mkdirSync(dataDir, { recursive: true })

  const { done, requestShutdown } = anchorMain({
    dataDir,
    actor: typeof args.actor === 'string' ? args.actor : 'anchor',
    listenPort: args.listen ? Number(args.listen) : undefined,
    useHyperswarm: !args['no-hyperswarm'],
    heartbeatMs: args['heartbeat-interval'] ? Number(args['heartbeat-interval']) : DEFAULT_HEARTBEAT_MS,
  })

  const armSignal = (sig) => process.on(sig, () => {
    console.log(`[anchor] ${sig} received — shutting down cleanly (this may take a moment)`)
    requestShutdown()
  })
  armSignal('SIGINT')
  armSignal('SIGTERM')

  // Top-level containment: this is a background service — an uncaught
  // exception or rejection anywhere below must be logged, never silently
  // exit the process (that would defeat the entire resilient-loop design).
  process.on('uncaughtException', (err) => console.error('[anchor] uncaughtException (contained):', err))
  process.on('unhandledRejection', (err) => console.error('[anchor] unhandledRejection (contained):', err))

  await done
  process.exit(0)
}
