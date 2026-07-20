// bare-anchor.mjs — Phase 3 "Probe and Anchor Roles" (bare-runtime campaign):
// the Bare-native port of kit/anchor.mjs (Mission A2 "The Corridor", Band 3
// — mesh/docs/MISSION_A2_CORRIDOR_SPEC.md §Band 3). SAME ROLE, SAME
// LIFECYCLE (Phase 3 dispatch): a headless, always-on peer process with a
// resilient outer boot/serve loop (capped exponential backoff, reset on a
// clean boot), a heartbeat file in the EXACT I7-compliant format anchor.mjs
// already uses (`<iso-timestamp> peers=<n> rooms=<n> mode=<mode>` — counts
// only, never a room key/title/peer address/message body), and shutdown
// ONLY via an explicit requestShutdown() call — never an inferred idle-exit.
//
// WHAT DOES NOT PORT TODAY, AND EXACTLY WHY (stated up front — full
// accounting in PHASE3_PROBE_ANCHOR_REPORT.md §2/§5): anchor.mjs's actual
// room-hosting payload is `kit-host.mjs`'s `createKitHost()`, which
// transitively imports `bridge-server.mjs`/`bridge-client.mjs` — and THOSE
// files hardcode `node:net`/`node:crypto` for their OWN internal localhost
// RPC bridge (the seam kit-repl.mjs's command layer talks over), not merely
// for kit-net.mjs's optional TCP fallback. There is no Bare-native
// replacement for that bridge seam in this tree yet — building one is
// `mesh/host/bare-bridge.mjs`, explicitly reserved to another coder (P1A) and
// explicitly in flight, per this campaign's own file fence. mesh-node.mjs
// ITSELF is already Bare-portable (verified: it resolves entirely through
// `#apply`/`#crypto` condition maps, and `host/bare-entry.mjs` already proves
// a real reducer fold under Bare with a byte-matched digest) — the reducer
// fold is not the blocker. The bridge is. Reimplementing or bypassing that
// bridge here would step directly on P1A's reserved file and duplicate work
// this campaign has already assigned elsewhere — not attempted.
//
// WHAT THIS FILE ACTUALLY PORTS, THEN: the anchor's OUTER SHAPE — the
// resilience loop, heartbeat discipline, and shutdown contract — as a
// runtime- and payload-agnostic module. `anchorMain()` below takes a
// REQUIRED `boot(dataDir, actor, log)` function (no default is bundled,
// deliberately — inventing a stand-in "boot" that does something other than
// real room-hosting risks being mistaken for the real thing, or for the
// spec's separately-reserved "blind-peer" role, R5). Once
// `host/bare-bridge.mjs` lands, wiring it in as this file's `boot` is a
// small, additive change — the loop/heartbeat/shutdown code below does not
// change at all. `bare-anchor-spike.mjs` proves this loop contract
// exhaustively today against a real (not trivial) fixture boot function that
// does genuine async work, without pretending that fixture is room-hosting.
//
// BINDING RULES (PHASE0_GATE_D2_FLUSH_RACE.md) — all four apply, and this
// file has no code path that violates any of them: no WebAssembly compile at
// all (RULE 1 n/a — this file never touches the reducer); every write goes
// through `console.log` via the same `log()` seam anchor.mjs already uses,
// never `bare-process`'s `process.stdout.write()` (RULE 2); the CLI exits
// via an explicit `process.exit(0)` after `done` resolves, never an inferred
// natural-drain exit (RULE 3, and this file's own boot loop never lets a
// hung boot block shutdown — see requestShutdown()'s poll-based wakeup,
// identical shape to anchor.mjs's own pollMs discipline); `bare-anchor-spike.mjs`
// gates this file's CLI entry point through a REAL spawn pipe via
// `spawn-pipe-harness.mjs` (this coder's own Phase 2 harness), not merely an
// in-process function call (RULE 4).
//
// No `node:` specifiers anywhere in this file. Heartbeat/file I/O uses
// `bare-fs` directly (top-level import — this file is Bare-ONLY by
// construction, like host/bare-entry.mjs, so there is no dual-runtime `#fs`
// condition-map need). Path construction avoids `node:path` entirely by
// building plain forward-slash strings from the caller's `dataDir` (Node's
// own fs APIs and bare-fs both accept forward slashes on Windows) — the same
// choice apply-bare.mjs makes with `new URL(..., import.meta.url)` for its
// own path needs, adapted here since `dataDir` is a caller-supplied plain
// string, not a same-file-relative asset.
//
// CLI: npx bare kit/bare-anchor.mjs [--data DIR] [--actor NAME]
//                                   [--heartbeat-interval MS]
//   (No --listen/--no-hyperswarm here, unlike anchor.mjs — those flags
//   configure kit-net.mjs's transport, which this file does not own; a
//   caller wiring a real `boot` function is responsible for its own
//   transport flags, this CLI only owns dataDir/actor/heartbeat cadence.)

import fs from 'bare-fs'
import process from 'bare-process'

const INITIAL_BACKOFF_MS = 5000
const MAX_BACKOFF_MS = 5 * 60 * 1000
const DEFAULT_HEARTBEAT_MS = 60 * 1000

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)) }
function nowIso() { return new Date().toISOString() }

/** Append one heartbeat line — BYTE-IDENTICAL format to anchor.mjs's
 * writeHeartbeat (I7: counts only, never a room key/title/peer address/
 * message body). A heartbeat write failure must never crash the anchor —
 * best effort, matching the original exactly. */
function writeHeartbeat(logPath, dataDirKeys, { peers, roomCount, mode }) {
  try {
    fs.mkdirSync(dataDirKeys, { recursive: true })
    fs.appendFileSync(logPath, `${nowIso()} peers=${peers} rooms=${roomCount} mode=${mode}\n`)
  } catch { /* best-effort — identical contract to anchor.mjs */ }
}

/**
 * One full boot+serve cycle. Resolves when shutdown is requested; throws on
 * a boot-time failure (caller retries with backoff) — identical control flow
 * to anchor.mjs's runOnce(), generalized over `boot` instead of a hardcoded
 * createKitHost() call.
 *
 * `boot(dataDir, actor, log)` must return:
 *   { roomCount, mode, totalPeers(): number, tick()?: void, close(): Promise<void> }
 * `tick()` is optional — called once per heartbeat cadence (mirrors
 * anchor.mjs's own defensive per-heartbeat hyperswarm-rejoin top-up); a
 * `boot` result with no `tick` simply skips that step.
 */
async function runOnce({ dataDir, actor, heartbeatMs, log, isShuttingDown, boot, onBoot }) {
  const keysDir = `${dataDir}/keys`
  const heartbeatPath = `${keysDir}/anchor.log`
  const session = await boot(dataDir, actor, log)

  log(`anchor ready — actor "${actor}", ${session.roomCount} room(s) loaded, transport ${session.mode}`)
  writeHeartbeat(heartbeatPath, keysDir, { peers: session.totalPeers(), roomCount: session.roomCount, mode: session.mode })

  if (onBoot) { try { onBoot(session) } catch { /* test hook errors never affect the anchor */ } }

  // Same responsive-shutdown-vs-slower-heartbeat split as anchor.mjs: poll
  // frequently for a shutdown request, only touch the heartbeat/tick on the
  // slower cadence.
  const pollMs = Math.min(500, heartbeatMs)
  try {
    let nextHeartbeatAt = Date.now() + heartbeatMs
    while (!isShuttingDown()) {
      // eslint-disable-next-line no-await-in-loop
      await sleep(pollMs)
      if (isShuttingDown()) break
      if (Date.now() < nextHeartbeatAt) continue
      nextHeartbeatAt = Date.now() + heartbeatMs
      if (session.tick) { try { session.tick() } catch { /* a tick failure must never crash the anchor */ } }
      writeHeartbeat(heartbeatPath, keysDir, { peers: session.totalPeers(), roomCount: session.roomCount, mode: session.mode })
    }
  } finally {
    await session.close()
  }
}

/**
 * anchorMain(opts) -> { done, requestShutdown() }
 * Identical contract to anchor.mjs's anchorMain — same return shape, same
 * "requestShutdown() is the ONE way to stop the loop" discipline (RULE 3).
 *
 * @param {string} opts.dataDir
 * @param {string} [opts.actor='anchor']
 * @param {number} [opts.heartbeatMs]
 * @param {(m: string) => void} [opts.log] — defaults to console.log (RULE 2:
 *   never bare-process's process.stdout.write()).
 * @param {(dataDir, actor, log) => Promise<Session>} opts.boot — REQUIRED,
 *   see runOnce()'s doc. No default — see file header for why.
 * @param {(session) => void} [opts.onBoot] — test-only seam, anchor-spike.mjs
 *   uses this to reach the live session without exposing it from the public
 *   API (mirrors anchor.mjs's own onBoot).
 */
export function anchorMain({
  dataDir, actor = 'anchor', heartbeatMs = DEFAULT_HEARTBEAT_MS,
  log = (m) => console.log(`[anchor] ${m}`),
  boot, onBoot,
} = {}) {
  if (!dataDir) throw new Error('anchorMain requires dataDir')
  if (typeof boot !== 'function') throw new Error('anchorMain requires a boot(dataDir, actor, log) function — see file header: no default is bundled')
  fs.mkdirSync(dataDir, { recursive: true })

  let shuttingDown = false
  const requestShutdown = () => { shuttingDown = true }
  const isShuttingDown = () => shuttingDown

  const done = (async () => {
    let backoff = INITIAL_BACKOFF_MS
    while (!shuttingDown) {
      try {
        await runOnce({ dataDir, actor, heartbeatMs, log, isShuttingDown, boot, onBoot })
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

// ── CLI entry point ─────────────────────────────────────────────────────
// No default `boot` is wired here either (see header) — the CLI entry point
// currently has NOTHING it can honestly boot without touching the reserved
// bridge file, so it refuses clearly rather than pretending. This is
// intentional and documented, not an oversight: a future coder wiring
// host/bare-bridge.mjs in adds exactly one line here (a real `boot` import +
// pass-through) and nothing else in this file changes.
const isMain = process.argv[1] && import.meta.url === 'file:///' + process.argv[1].replace(/\\/g, '/')
if (isMain) {
  console.error('bare-anchor.mjs: no boot() is wired into the CLI yet — the Bare-native room-hosting bridge (host/bare-bridge.mjs) is a separate, in-flight piece of this campaign (see this file\'s header). anchorMain() is usable programmatically today with your own boot() function; there is no standalone CLI anchor to run until the bridge lands.')
  process.exit(1)
}
