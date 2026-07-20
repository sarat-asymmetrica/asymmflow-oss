// bare-guide.mjs — Phase 3 (Bare-runtime campaign): the Guided Path, ported
// to Bare. Successor to mesh/kit/guide.mjs (untouched, stays the Node
// kit's rollback entry point — campaign doctrine: the old line receives
// ZERO further investment but stays warm and green).
//
// THE UX LAW THIS FILE PORTS AS-IS, NOT AS A SUGGESTION (per
// PHASE0_NOTES_D_REVERIFY.md §6, the implementable spec this file was built
// against — read that section before touching anything here):
//   - the plain-question menu style and its exact copy
//   - normalizeCode/groupInFours whitespace handling (byte-for-byte the
//     same regex as guide.mjs's own — WhatsApp-paste tolerance is
//     field-critical, not decoration)
//   - the three literal verdict words, printed large, framed, with the
//     exact "Read this word to the person on the call:" instruction
//   - the once-per-session conversational firewall OFFER's exact copy
//     (the ACTION behind it is an honest stub here — see §"NOT PORTED"
//     below; the COPY is law, the mutation is not yet implemented)
//   - the error fold-line convention (one plain sentence, a fold rule,
//     the raw detail)
//   - the FIFO-queue stdin discipline (P0-D's §6: sequential
//     rl.question()-shaped code drops a line that arrives before the next
//     ask() call — reproduced in guide.mjs's own history, "100% of the
//     time" per its comment). This file's I/O layer is NOT built on
//     `readline` at all (see "Import discipline" below) — it re-derives
//     the SAME FIFO-queue-of-lines guarantee directly over raw stdio data
//     chunks, which is the exact technique bare-bridge.mjs's ndjson framer
//     already proved correct in Phase 2 (frameLoopCheck's split-line case)
//     — reused here, not reinvented.
//
// WHAT IS DELIBERATELY NOT PORTED, and why (honesty law, per this phase's
// brief: "report it rather than approximating it"):
//   - Menu [1] "Check the connection": no Bare-native connectivity probe
//     exists anywhere in this tree yet. probe.mjs (the Node original) is
//     npm-package-based (hyperdht/hyperswarm/hypercore-id-encoding — all
//     independently proven Bare-clean, BARE_SPIKE_REPORT.md's 11/11) but
//     building a Bare-native probe is not in this phase's file ownership
//     (mesh/kit/bare-guide.mjs, mesh/kit/bare-guide-spike.mjs only) and was
//     not assigned. This menu item is a clearly-labeled stub, not a faked
//     CORRIDOR verdict.
//   - Menu [3]/[4] "anchor"/"status": the real actions here install/remove
//     a Windows Scheduled Task and query firewall state — hard-to-reverse,
//     outward-facing system mutation. Building and testing that live in
//     this sandbox, without a chance for the machine's owner to review the
//     exact schtasks/netsh invocation first, is exactly the kind of action
//     this session's own operating guidance says to confirm before taking.
//     The menu entries, prompts, and copy are ported verbatim; the actual
//     mutation is an honest stub that says so plainly rather than either
//     silently doing nothing or pretending to succeed.
//   - The firewall step's ACTUAL elevation/rule-creation action (its COPY
//     is ported, per the UX law above).
//
// WHAT IS REAL, not a stub: Menu [2] "Open the messenger" wires to
// bare-bridge.mjs's `createBridgeCore` IN-PROCESS (Phase 2's own
// deliverable, reused here exactly per I1 — no reimplementation) and
// drives a small text REPL over it (post a message, list rooms) — a
// genuine, working, Bare-native, end-to-end path from this guide through
// the reducer.
//
// SC-3a (Sealed Corridor campaign) EXTENDS menu [2] beyond a single-device
// demo: it now mints/redeems real invite codes and drives bare-net.mjs's
// real hyperswarm+TCP network, so two SEPARATE kits can join the SAME
// room — founder side in startAsFounder(), joiner side in joinAsJoiner(),
// both below, reached via one plain question (A2.1's own "did someone send
// you a code?" wording, Mission A2 Band 6). The single-device "kitchen
// table" behavior this paragraph already describes is still exactly what
// happens when nobody chooses to connect (the Enter-default "just open what
// I have" shortcut, and Path A's own reopen-or-create first step) — this is
// additive, not a rewrite.
//
// Import discipline (binding, unchanged from Phase 1/2): NO `node:`
// specifier anywhere, static or dynamic. `fs` via the `#fs` condition map.
// `bare-subprocess`/`bare-readline` were flagged as possibly-needed in the
// brief; NEITHER is used here — `bare-subprocess` is deliberately unused
// because the actions that would need it (firewall/anchor mutation) are
// the honest stubs above, and `bare-readline` is unused because the
// FIFO-queue I/O layer is hand-rolled over `bare-bridge.mjs`'s own
// `getRealStdio()` (raw data/end events), not `readline`'s line-splitting
// — one less dependency, and it reuses code already proven correct.
//
// LAYOUT: unlike guide.mjs's `resolveKitPath` (which exists because the
// OLD kit's sibling files could live at one of two relative depths — see
// that file's own header), a bare-pack-sealed kit resolves its wasm/native
// assets at PACK TIME via `import.meta.asset()` (PHASE2_ASSET_LOCATION.md,
// proven end-to-end by `host/bare-entry.mjs`) — the whole "which depth is
// probe.mjs at" problem class does not exist here. This file's only
// runtime-resolved paths are its OWN mutable state (a device identity, a
// storage directory for the demo room), and those are deliberately
// CWD-relative (`./data/...`), matching DP1's established `data\keys\` /
// `data\corestore\` sibling convention for a launched kit — never
// `import.meta.url`-relative, which the condition-map gate's own "bonus
// finding" (PHASE0_GATE_B3_CONDITION_MAP.md) proved resolves to a VIRTUAL
// path inside a bare-pack `.bundle` file, not a real directory.

import fs from '#fs'
import hcrypto from 'hypercore-crypto'
import { getRealStdio, createBridgeCore } from '../host/bare-bridge.mjs'
import { deviceKeys } from '../host/capability.mjs'
// SC-1 (Sealed Corridor campaign) additions — see `ensureMessengerCore`/
// `openMessenger` below for how these are used. `createMeshNode` reopens a
// previously-created room's Autobase; `createSocialRoomNode` (aliased,
// same alias kit-repl.mjs's own create() uses) founds a fresh one with a
// STABLE storage dir this file chooses, not the bridge core's own
// `createSocialRoom` wire method's unpredictable `social-<hex>` pick.
// `bare-registry.mjs` is the Bare-native port of kit-registry.mjs — see
// that file's own header for the precedent this reuses.
import { createMeshNode, waitFor } from '../host/mesh-node.mjs'
import { createSocialRoom as createSocialRoomNode } from '../host/social-room.mjs'
import { loadRoomRegistry, saveRoomRegistryEntry } from './bare-registry.mjs'
// SC-3a (Sealed Corridor campaign) addition — see startAsFounder/
// joinAsJoiner below. decodeInviteCode is the SAME decoder bare-bridge.mjs's
// own redeemInvite uses; this file never re-derives the invite envelope
// format. It runs under BOTH runtimes (pure JS + hypercore-id-encoding,
// already proven Bare-clean) so it is a plain static import, unlike bare-net
// below.
import { decodeInviteCode } from '../host/invite-code.mjs'
// SC-3b's connection check is imported DYNAMICALLY, inside checkConnection()
// — see that function. It is deliberately NOT a static import here, and the
// reason is measured, not stylistic: see the comment at its use site.
//
// SC-3a's bare-net.mjs (the real corridor network) is ALSO imported
// dynamically, inside ensureNetwork() below — same reason, independently
// re-measured for this file (`node -e "import('./kit/bare-net.mjs')"` throws
// `require.addon is not a function`, same failure bare-connection-check.mjs's
// own import produces): see ensureNetwork()'s own comment for the full
// consequence.

const DATA_DIR = './data'
const KEYS_DIR = `${DATA_DIR}/keys`
const SEED_FILE = `${KEYS_DIR}/bare-guide-device.seed`
// SC-1 deviation 3, gate-reviewed and confirmed harmless (documented per the
// reviewer's ask, not left implicit): this used to be ROOM_STORAGE_DIR
// (`${DATA_DIR}/corestore/bare-guide-room`), a single fixed subdirectory.
// It is now the PARENT corestore dir, matching kit-host.mjs's own
// `corestoreDir` convention -- required so `${CORESTORE_DIR}/${entry.
// storage}` (the reopen loop, below) and the create path's own
// `${CORESTORE_DIR}/${dirName}` land in the SAME place a stable per-room
// subdirectory can be chosen under. This also changes what `createBridgeCore`'s
// OWN `storageDir` option means for its wire methods (`createSocialRoom`/
// `redeemInvite` in bare-bridge.mjs, both join `storageDir` with their own
// subdir) -- rooms created through THOSE wire methods would now land at
// `data/corestore/social-<hex>` instead of `data/corestore/bare-guide-room/
// social-<hex>`. Confirmed harmless: grepping this file, `openMessenger`'s
// REPL only ever dispatches `listRooms` and `post` (see `call('...')`
// below) -- `createSocialRoom`/`redeemInvite` are never reached from this
// guide today (this file's own create path bypasses them entirely, per the
// DEVIATION comment at the create call site), and no prior version of this
// guide ever persisted a room, so there is no existing on-disk layout under
// the old `ROOM_STORAGE_DIR` to migrate. A future mission that wires `/join`
// (SC-3's ceremony) will reach `redeemInvite` for the first time and should
// re-confirm this note is still true before relying on it.
//
// RE-CONFIRMED, SC-3a: `redeemInvite` IS now reached, from `joinAsJoiner`
// below — the note above is still true in the way that matters. `redeemInvite`
// only falls back to ITS OWN unstable `joined-<prefix>-<randomHex>` storage
// path when `rooms.get(decoded.baseKey)` finds nothing (bare-bridge.mjs's own
// `if (!node)` guard); `joinAsJoiner` always creates the node with THIS
// file's own stable `joined-<prefix>` dir and `registerRoom`s it FIRST, so
// that branch is never taken and the on-disk layout stays exactly what
// SC-1/kit-registry.mjs's reopen loop expects.
const CORESTORE_DIR = `${DATA_DIR}/corestore`

// SC-3a: bounds joinAsJoiner's wait for `node.writable` after redeeming an
// invite. "Never hang" is binding law (see this file's own menu-[1] doctrine,
// restated for this ceremony) — a wait that can block forever is a defect,
// not patience. Generous enough for a real two-human phone ceremony
// (SC0_PORT_MAP.md §1 measured a single DHT punch taking up to 45s even on a
// WORKING corridor) and short enough that a gate can run this leg's negative
// control ("unreachable network") to natural completion repeatedly — see
// bare-corridor-spike.mjs's own report for the measured tradeoff.
const JOIN_WRITABLE_TIMEOUT_MS = 90 * 1000

// ── pure helpers — byte-for-byte the same regex/logic as guide.mjs's own
// (guide-spike.mjs unit-tests these directly there; bare-guide-spike.mjs
// does the same here) ──────────────────────────────────────────────────

export function normalizeCode(input) {
  return String(input ?? '').replace(new RegExp('[\\s\\u00A0\\u200B]+', 'g'), '').trim()
}

export function groupInFours(code) {
  return String(code).match(/.{1,4}/g)?.join(' ') ?? String(code)
}

// ── FIFO-queue stdin I/O — the same guarantee as guide.mjs's
// createGuideIO, re-derived over raw stdio events instead of `readline`
// (see file header). Unconditional listening from the start, own queue —
// no line is ever lost regardless of arrival-vs-ask() timing. ──────────

export function createGuideIO(io) {
  let buf = ''
  const queue = []
  const waiters = []
  let closed = false
  const offData = io.onData((chunk) => {
    buf += chunk
    let idx
    while ((idx = buf.indexOf('\n')) !== -1) {
      const line = buf.slice(0, idx)
      buf = buf.slice(idx + 1)
      if (waiters.length) waiters.shift()(line)
      else queue.push(line)
    }
  })
  const offEnd = io.onEnd(() => {
    closed = true
    while (waiters.length) waiters.shift()(null) // null = stdin ended, no more input ever
  })
  function nextLine() {
    if (queue.length) return Promise.resolve(queue.shift())
    if (closed) return Promise.resolve(null)
    return new Promise((resolve) => waiters.push(resolve))
  }
  return {
    async ask(prompt) {
      io.write(prompt)
      return nextLine()
    },
    close() { offData(); offEnd() },
  }
}

// ── device identity — persisted once per kit install, `./data/keys/`
// sibling convention (DP1). Random on first run, reused thereafter so a
// receptionist's messenger identity is stable across guide sessions. ────

function loadOrCreateSeed() {
  try {
    if (fs.existsSync(SEED_FILE)) {
      const seed = fs.readFileSync(SEED_FILE)
      if (seed.length === 32) return seed
    }
  } catch { /* fall through to (re)create */ }
  const seed = hcrypto.randomBytes(32)
  try {
    fs.mkdirSync(KEYS_DIR, { recursive: true })
    fs.writeFileSync(SEED_FILE, seed)
  } catch { /* best-effort persistence; an in-memory-only identity for this
    session is still correct, just not stable across restarts */ }
  return seed
}

// ── verdict / error presentation — exact copy, exact shape ─────────────

function printVerdictLarge(write, verdict) {
  const line = '='.repeat(Math.max(40, verdict.length + 8))
  write('\n')
  write(line + '\n')
  write(`   ${verdict}\n`)
  write(line + '\n')
  write('\n')
  write(`Read this word to the person on the call: ${verdict}\n`)
}

function reportError(write, err) {
  write('\n')
  write('Something went wrong -- read this line to the person on the phone:\n')
  write(`  ${err?.message || String(err)}\n`)
  write('--- details for support ---\n')
  write(`${err?.stack || String(err)}\n`)
}

// ── firewall offer -- COPY is law, action is an honest stub (see header) ──

let firewallOffered = false

async function ensureFirewall(io, write) {
  if (firewallOffered) return
  firewallOffered = true

  write('\n')
  write('Before we connect, this computer needs one quick permission.\n')
  write('Windows will ask "Do you want to allow this app to make changes?" -- click Yes.\n')
  write('That is the ONE popup in this whole process; nothing else on this computer changes.\n')
  const answer = await io.ask('Press Enter to continue, or type skip and press Enter to skip this for now.\n> ')
  if (answer === null) return
  if (/^skip$/i.test(answer.trim())) {
    write('Skipping for now -- the connection might not work until this step is done.\n')
    return
  }
  // Honest stub -- see file header "WHAT IS DELIBERATELY NOT PORTED".
  write('(this automatic step is not available yet in the Bare kit -- if Windows asks for\n')
  write(' permission when you connect, click Yes; nothing else needs to be done manually.)\n')
}

// ── menu actions ─────────────────────────────────────────────────────────

// SC-3b (Sealed Corridor): this stopped being an honest stub. The real DHT
// bootstrap + NAT checks now run in-process from bare-connection-check.mjs,
// which wraps bare-probe.mjs's OWN checks (exported, never reimplemented, so
// the diagnostic vocabulary a field contact reads aloud stays byte-identical
// to the probe's). In-process is not a preference: a bare-pack'd sealed kit
// has no bare-probe.mjs FILE on disk to spawn.
//
// WHY THIS DOES NOT RUN THE PUNCH TEST (check 4), stated here as well as in
// that module: the punch needs a second live machine and a pasted key, and
// SC0_PORT_MAP.md §1 measured a single punch taking up to 45s and coming back
// RED 1 time in 7 even with a verified negative control. A "quick diagnostic"
// menu item that blocks for 45s with nobody to dial it would be worse than
// the stub it replaces. Menu [2] performs a REAL two-sided connection, which
// is the stronger test — and the module's own closing line says exactly that
// to the person at the keyboard rather than leaving the gap silent.
//
// `runConnectionCheck` is contractually safe to call from inside this menu
// loop: it never throws, never hangs past its own bound, and never calls
// exit() (proven by that mission's gate, not asserted). `io.ask` is passed
// through unchanged so its bounded retry offer uses the guide's own
// FIFO-queue stdin, never a second reader.
//
// WHY THE IMPORT IS DYNAMIC AND MUST STAY DYNAMIC — measured, not preferred.
// bare-connection-check.mjs imports bare-probe.mjs, which top-level-imports
// `bare-process`, whose graph reaches `bare-abort` — a native addon that
// needs `require.addon`, a Bare-only API. A STATIC import here therefore
// makes this whole file un-loadable under Node: verified directly, `node
// kit/bare-guide.mjs` dies in `bare-abort/index.js:1` before printing a
// single byte. That is not hypothetical breakage — bare-guide-spike.mjs
// drives this guide under BOTH `process.execPath` and `bare.exe` (its
// layer-2 "spawn(node)" legs), so a static import silently deletes half the
// existing regression suite.
//
// Deferring it to the call site keeps every other path dual-runtime and
// costs only that menu [1] is Bare-only — which is honest, because the check
// it performs is Bare-only anyway.
//
// THE TRADE THIS CREATES, and how it is covered: bare-pack's static
// traverser is what decides which native addons get offloaded into the
// sealed kit, so a dynamic specifier risks the campaign's signature
// broken-but-green shape (the kit renders its whole ceremony and silently
// cannot do the thing). That risk is NOT left to inference — the corridor
// build passes `--require-addons` to build-bare-kit.mjs, which refuses to
// produce a kit missing them. If bare-pack ever stops resolving this form,
// the BUILD fails loudly instead of the field failing quietly.
async function checkConnection(io, write) {
  await ensureFirewall(io, write)
  const { runConnectionCheck } = await import('./bare-connection-check.mjs')
  await runConnectionCheck({ write, ask: (prompt) => io.ask(prompt) })
}

let messengerCtx = null // lazily created, reused across menu visits in one session

// SC-1 (Sealed Corridor campaign): PHASE3_GUIDE_REPORT.md §5 flagged this
// exact gap -- "each fresh run prints 'created a new room' again... not a
// regression... but a real limitation." This function now ports kit-host.mjs's
// own reopen loop (GL-5 discipline: state must survive a real process
// restart, not just an in-memory reconnect) so a device's room comes back
// automatically instead of vanishing every run. `write` is the guide's own
// output function (see runGuide) so a per-room reopen failure can be
// reported to the person at the keyboard the same plain way kit-host.mjs's
// `log(...)` callback does -- never silently, and never by crashing the
// other rooms' reopen or the guide itself.
// SC-3a bug, found live at this mission's own gate (2026-07-20): every
// device used the SAME literal actor string 'guide', which is silently
// correct for a single device talking to itself (the original "kitchen
// table" demo this file started as) and silently WRONG the moment a real
// corridor joins two separate machines into the same room -- msgId is
// `{actor}:{seq}` (bare-bridge.mjs's own header), and `takeSeq` computes
// each device's next seq from ITS OWN local view, so two devices sharing
// one actor name can independently compute the SAME seq for two DIFFERENT
// messages. Reproduced directly at this gate: a founder's second post was
// silently rejected with `(not posted -- duplicate msgId "guide:4")` after
// the joiner's own post happened to land on the same seq. Deriving the
// actor from this device's OWN public key makes collision structurally
// impossible (two devices never share a keypair) while staying STABLE
// across restarts (the seed persists, so does this). This was flagged as
// out of scope for SC-1 and left for "SC-3 if it is wanted at all"
// (SC0_PORT_MAP.md §3.1) -- it is wanted, because SC-3a is the first
// mission that puts two real devices in one room.
async function ensureMessengerCore(write) {
  if (messengerCtx) return messengerCtx
  const seed = loadOrCreateSeed()
  const keys = deviceKeys(seed)
  const actor = `guide-${keys.pubHex.slice(0, 8)}`
  const core = createBridgeCore({ rooms: new Map(), actor, deviceKeys: keys, storageDir: CORESTORE_DIR })
  messengerCtx = { core, keys, actor, nextId: 1 }

  // Reopen every room this device previously created -- same discipline as
  // kit-host.mjs's own boot loop, ported verbatim in spirit: createMeshNode
  // against the SAME storage dir + same authorityPub/encryptionKey/bootstrap
  // it was opened with originally is mesh-node.mjs's own reopen contract
  // (kit-registry.mjs's header: "same storage dir + same keys IS the same
  // device waking up, not a new identity"). A single entry that fails to
  // reopen must not abort the others or crash the guide -- it is logged as
  // one plain sentence and skipped, exactly like kit-host.mjs's own catch.
  for (const entry of loadRoomRegistry(KEYS_DIR)) {
    const storagePath = `${CORESTORE_DIR}/${entry.storage}`
    // Gate finding (SC1_REPORT.md §5b): createMeshNode/Corestore does NOT
    // throw when a registered room's storage folder is missing (checked
    // directly against Corestore's real behavior, not assumed) -- Corestore
    // silently CREATES a fresh empty store at that path instead, and since
    // a founder's own room always has `bootstrap: null`, Autobase then
    // founds a brand-new base there with a DIFFERENT key than the
    // registry's roomKey. Left unguarded that is worse than a caught
    // exception: it fabricates a phantom, empty "room" this guide would
    // then greet as "found your earlier conversation again" -- silently
    // wrong, not silently safe. The fix is a plain existence check BEFORE
    // ever calling createMeshNode, so no phantom store is written to disk
    // as a side effect of merely trying to reopen a folder that is gone
    // (someone deleted it by hand is the realistic field case).
    if (!fs.existsSync(storagePath)) {
      write?.('(could not reopen a saved room -- its storage folder is missing)\n')
      continue
    }
    try {
      const node = await createMeshNode({
        storage: storagePath,
        bootstrap: entry.bootstrap || null,
        authorityPub: entry.authorityPub,
        mode: 'room',
        encryptionKey: entry.encryptionKey ? Buffer.from(entry.encryptionKey, 'hex') : undefined,
      })
      core.registerRoom(node.key, node)
    } catch (err) {
      write?.(`(could not reopen a saved room -- ${err.message})\n`)
    }
  }

  return messengerCtx
}

// Shared by the "just open what I have" shortcut and Path A's own first
// step (both need the identical reopen-or-create logic) -- extracted
// unchanged from what used to be openMessenger's own inline body (SC-3a).
async function reopenOrCreateRoom(write, ctx, call) {
  const listed = await call('listRooms', {})
  let roomKey = listed.result?.[0]?.roomKey
  if (!roomKey) {
    // DEVIATION, declared (same one kit-repl.mjs's own create() declares,
    // ported here verbatim in spirit): bypass the bridge core's OWN
    // `createSocialRoom` wire method, which picks an unpredictable
    // `social-<hex>` storage dir every call -- GL-5 reopen discipline needs
    // a STABLE directory name to reopen against after a restart, and
    // Corestore doesn't reliably expose the path it was opened with back
    // out. Only one social room is ever auto-created by this guide (the
    // "kitchen table" kit, one shared room per device -- kit-host.mjs's own
    // kitchen-table UX doc), so a single fixed dir name is correct here and
    // needs no random/unique id generation at all -- deliberately simpler
    // than kit-repl.mjs's own `room-${randomUUID()}`, which exists there to
    // support MULTIPLE named rooms via repeated /create, a case this guide
    // does not have. The room is registered into the SAME core `rooms` Map
    // either way, so every dispatch method behaves identically afterward --
    // only the creation path differs.
    const dirName = 'room-guide'
    const encryptionKey = hcrypto.randomBytes(32)
    const node = await createSocialRoomNode({
      creatorKeys: ctx.keys, storage: `${CORESTORE_DIR}/${dirName}`,
      title: 'kitchen table', encryptionKey, ts: Date.now(), actor: ctx.actor,
    })
    ctx.core.registerRoom(node.key, node)
    saveRoomRegistryEntry(KEYS_DIR, {
      roomKey: node.key, storage: dirName, authorityPub: ctx.keys.pubHex,
      encryptionKey: encryptionKey.toString('hex'), bootstrap: null, title: 'kitchen table',
    })
    roomKey = node.key
    write('(created a new room for this kit -- "kitchen table")\n')
  } else {
    // Reception-Grade voice (A2.1): plain, short, no hex on the happy path
    // -- the earlier conversation was found again, not a new one. Never
    // print the raw 64-hex room key to the client here.
    const title = listed.result[0].title || 'kitchen table'
    write(`(found your earlier conversation again -- "${title}")\n`)
  }
  return roomKey
}

// The actual post/rooms/exit loop -- unchanged from what used to be
// openMessenger's own inline body, now shared by every path that reaches a
// usable room (the reopen shortcut, Path A after its ceremony, Path B after
// its ceremony, and both sides' already-joined short-circuits) so there is
// exactly one place that owns this REPL's behavior (SC-3a).
// SC-3a gate finding, 2026-07-20 — A REAL GAP, not a test artifact.
//
// The messenger could POST but had no way to READ. `/rooms` printed one
// summary line per room ("last: <preview>"), and `lastPreview` is the LAST
// message in canonical (Seq, Actor) order — NOT the most recently received
// one. Two consequences, and the second is the serious one:
//
//   1. A message that arrives with a LOWER seq than one already present is
//      structurally invisible there. That is not a rare edge case: each
//      device seeds its own seq counter independently from the ops it has
//      SEEN (bridge-server.mjs's own documented GL-9 lesson), so two people
//      typing at roughly the same time routinely produce exactly this. It is
//      reproducible — the corridor gate hit it every round: the founder
//      posted seq 3 while the joiner had already posted seq 4, so the
//      founder's message replicated correctly and simply never appeared in
//      the joiner's one-line preview.
//   2. Therefore two people on a corridor COULD NOT READ EACH OTHER. The
//      entire point of the mission is a two-way conversation, and the
//      client-facing surface offered no way to see one.
//
// Found by driving the guide the way a person uses it and asserting on what
// the SCREEN shows, rather than on internal state. An earlier draft of the
// gate read this as "A's message never reached B" — a replication failure
// that was not happening. The messages were always there; nothing ever
// displayed them.
//
// The fix ports kit-repl.mjs's own `fmtMessage` shape and its "show the
// recent history on open" behavior (that REPL has had `/open` printing the
// last 10 messages since Mission U1.5 — this guide simply never got it).
const RECENT_MESSAGE_COUNT = 10

async function showConversation(write, call, roomKey) {
  const state = await call('roomState', { roomKey })
  if (!state.ok) { write(`  (could not read the conversation -- ${state.error})\n`); return }
  const msgs = (state.result.messages ?? []).filter((m) => !m.deleted)
  if (!msgs.length) { write('  (no messages yet -- type one and press Enter)\n'); return }
  for (const m of msgs.slice(-RECENT_MESSAGE_COUNT)) write(`  ${m.actor}: ${m.body}\n`)
}

async function messengerRepl(io, write, ctx, call, roomKey) {
  write('\n')
  write('Type a message and press Enter to post it.\n')
  write('Type /read to see the conversation, /rooms to list rooms, /exit to leave.\n')
  write('\n')
  // Show whatever is already there on entry -- a person who joins an existing
  // conversation must not be greeted by an empty screen.
  await showConversation(write, call, roomKey)
  write('\n')
  for (;;) {
    const raw = await io.ask('> ')
    if (raw === null) return
    const line = raw.trim()
    if (line === '') continue
    if (line === '/exit') return
    if (line === '/read') { await showConversation(write, call, roomKey); continue }
    if (line === '/rooms') {
      const rooms = await call('listRooms', {})
      for (const r of rooms.result) write(`  ${r.title || '(untitled)'} -- ${r.kind} -- last: ${r.lastPreview || '(no messages yet)'}\n`)
      continue
    }
    const res = await call('post', { roomKey, body: line, expectation: '' })
    if (!res.ok) write(`  (not posted -- ${res.error})\n`)
    else write(`  (posted, seq ${res.result.seq})\n`)
  }
}

let corridorNet = null

// SC-3a (Sealed Corridor campaign): the real network leg, wired to the
// founder/joiner ceremony below. DYNAMIC IMPORT, and it must STAY dynamic
// for the SAME measured reason checkConnection's own import of
// bare-connection-check.mjs must (see that function's comment, above):
// bare-net.mjs top-level-imports `bare-tcp`, a native addon that needs
// `require.addon` -- a Bare-only API. Verified directly, same finding as
// checkConnection's own: `node -e "import('./kit/bare-net.mjs')"` throws
// `require.addon is not a function` under plain Node. A STATIC import here
// would make this whole file un-loadable under Node exactly like a static
// bare-connection-check.mjs import would -- see checkConnection's own
// comment for the full consequence (bare-guide-spike.mjs's "spawn(node)"
// legs, half the existing regression suite).
//
// CONSEQUENCE FOR MENU [2]: the founder/joiner NETWORK step is therefore
// Bare-only, the same way menu [1] is -- but unlike menu [1], menu [2] has a
// real LOCAL-ONLY fallback (posting to a room nobody else has joined yet
// still works with zero network, always has). startAsFounder() below treats
// a failure HERE as NON-FATAL for exactly that reason: a founder running
// under Node can still open their own messenger and post, they just cannot
// be joined until this is fixed (or the kit is run under Bare, its real
// target runtime). joinAsJoiner() has no such fallback -- joining WITHOUT a
// network is not a real state, so it lets a failure here propagate to the
// menu loop's own reportError(), same fold-line convention as a bad invite
// code.
async function ensureNetwork() {
  if (corridorNet) return corridorNet
  const { createNetwork } = await import('./bare-net.mjs')
  corridorNet = createNetwork()
  return corridorNet
}

// SC-3a Path A ("I am starting it here"): reopen-or-create, mint an invite,
// start the network, and accept the OTHER machine's pairing code IN-PROCESS
// via `node.addWriter()` -- deviation #2 (bare-bridge.mjs's own header):
// protocol v0 deliberately has no become-a-writer wire method. Ported from
// kit-repl.mjs's own `/invite` + `/addwriter` pair, folded into one guided
// flow per A2.1's "one question at a time" law -- this file's own
// `checkConnection` doctrine ("one probe run is not a definitive verdict")
// does not apply here: this IS the real two-sided connection, the stronger
// test that same doctrine points to.
async function startAsFounder(io, write, ctx, call) {
  const roomKey = await reopenOrCreateRoom(write, ctx, call)
  const node = ctx.core.rooms.get(roomKey)

  const inviteRes = await call('openDmInvite', { roomKey })
  if (!inviteRes.ok) throw new Error(inviteRes.error || 'could not create a code for this room')

  write('\n')
  write('Here is the code for the OTHER computer. It is long, so send it however\n')
  write('you like (WhatsApp, email, a message) -- or read it aloud in groups of four\n')
  write('if you have to.\n')
  write('\n')
  write(groupInFours(inviteRes.result.inviteCode) + '\n')
  write('\n')

  // Best-effort, deliberately NOT let-it-throw -- see ensureNetwork()'s own
  // comment for why this is the ONE call site in this ceremony that is
  // non-fatal on a network failure.
  let net = null
  try {
    net = await ensureNetwork()
    net.joinHyperswarm(roomKey, node)
  } catch (err) {
    write(`(could not start the network on this computer right now -- ${err.message})\n`)
    write('(you can still use the messenger below -- the other computer just cannot join until this is fixed)\n')
  }
  let tcpPort = null
  if (net) {
    try { tcpPort = await net.listenTcp(0, node) } catch { /* LAN fallback is best-effort; hyperswarm can still carry the ceremony */ }
    write('Waiting for the other computer to connect over the internet...\n')
    if (tcpPort) write(`(On the same office network? They can also connect directly to this computer -- port ${tcpPort}.)\n`)
  }

  write('\n')
  write('When the OTHER computer shows ITS code (a short one), paste it here and press Enter.\n')
  const pairingRaw = await io.ask('If you want to open the messenger now and do this later, type skip and press Enter.\n> ')
  if (pairingRaw === null) return
  const pairingAnswer = pairingRaw.trim()
  if (pairingAnswer !== '' && !/^skip$/i.test(pairingAnswer)) {
    // Sanitisation ported VERBATIM from kit-repl.mjs's own addWriter --
    // WhatsApp/voice-paste tolerance is field-critical (SC0_PORT_MAP.md
    // §3.3), not decoration.
    const pairingCode = pairingAnswer.replace(/[<>"'`\s]/g, '')
    try {
      await node.addWriter(pairingCode)
      write('(done -- the other computer can finish joining now)\n')
    } catch (err) {
      write('That did not look like a valid code from the other computer -- read this line to the person on the phone:\n')
      write(`  ${err.message}\n`)
      write('(you can open the messenger and try again from the top menu once you have the right one)\n')
    }
  }

  return messengerRepl(io, write, ctx, call, roomKey)
}

// SC-3a Path B ("someone sent me a code"): paste-and-Enter redeem, ported
// from kit-repl.mjs's own `/join` -- including F2's already-joined
// short-circuit (MSG-D25, a real field failure: a person who did nothing
// wrong must NEVER be told their invite is "exhausted"). Storage dir is
// STABLE (`joined-<prefix>`, matching kit-repl.mjs's own convention) and
// the registry entry is saved BEFORE the ceremony completes (F2(a), see the
// RE-CONFIRMED note at CORESTORE_DIR above) -- a restart mid-ceremony still
// finds this room on the next boot instead of orphaning it.
async function joinAsJoiner(io, write, ctx, call, code) {
  // decodeInviteCode's own errors are already plain language (see that
  // file) -- letting this throw and land in menuLoop's own reportError() is
  // the SAME fold-line convention every other error in this file uses, not
  // a special case. This is negative control (i)'s exact path (a malformed
  // code): a plain sentence, never a stack trace, on the client's own line.
  const decoded = decodeInviteCode(code)

  const existing = ctx.core.rooms.get(decoded.baseKey)
  if (existing && existing.writable) {
    write('(you already joined this room -- opening it)\n')
    return messengerRepl(io, write, ctx, call, existing.key)
  }

  let node = existing
  if (!node) {
    const dirName = `joined-${decoded.baseKey.slice(0, 16)}`
    node = await createMeshNode({
      storage: `${CORESTORE_DIR}/${dirName}`,
      bootstrap: decoded.baseKey, authorityPub: decoded.authorityPub, mode: 'room',
      encryptionKey: decoded.encryptionKey,
    })
    ctx.core.registerRoom(node.key, node)
    saveRoomRegistryEntry(KEYS_DIR, {
      roomKey: node.key, storage: dirName, authorityPub: decoded.authorityPub,
      encryptionKey: decoded.encryptionKey ? decoded.encryptionKey.toString('hex') : null,
      bootstrap: decoded.baseKey, title: '(joined room)',
    })
  }

  write('\n')
  write('Your code for the OTHER computer -- read it out, or have them type it in:\n')
  write('\n')
  write(groupInFours(node.writerKey) + '\n')
  write('\n')
  write('Tell them: paste this in when their screen asks for it.\n')

  // Network failure here is NOT recoverable -- joining without a network
  // path is not a real state -- so this is the one place in the ceremony
  // that is deliberately allowed to throw naturally into reportError(),
  // same fold-line convention as a bad code above. This IS "unreachable
  // network", gate negative control (iii).
  const net = await ensureNetwork()
  net.joinHyperswarm(node.key, node)

  write('\n')
  const lanRaw = await io.ask(
    'Same office network, and you have the other computer\'s address? Type it now as\n' +
    'address:port and press Enter. Otherwise just press Enter.\n> '
  )
  if (lanRaw === null) return
  const lanAnswer = lanRaw.trim()
  if (lanAnswer) {
    const [host, portStr] = lanAnswer.split(':')
    const port = Number(portStr)
    if (host && Number.isInteger(port) && port > 0) {
      try { await net.connectTcp(host, port, node) } catch { /* best-effort -- hyperswarm can still carry the ceremony */ }
    }
  }

  write('Waiting for the other computer to let you in (this can take a little while)...\n')
  // Bounded -- "never hang" is binding law (JOIN_WRITABLE_TIMEOUT_MS's own
  // comment, above). On expiry this throws a plain "timed out waiting for
  // the other computer letting you in" -- deliberately NOT the raw 64-hex
  // room key kit-repl.mjs's own equivalent label carries (never a raw key
  // on the happy path) -- caught by menuLoop's own reportError(), same
  // convention as everywhere else in this file. This is the honest-failure-
  // copy half of negative control (iii): the guide returns to its menu,
  // never hangs.
  await waitFor(async () => { await node.base.update(); return node.writable }, {
    label: 'the other computer letting you in', timeout: JOIN_WRITABLE_TIMEOUT_MS,
  })

  const redeemRes = await call('redeemInvite', { inviteCode: code, actor: ctx.actor })
  if (!redeemRes.ok) {
    const reason = redeemRes.error ?? ''
    // Same already-joined short-circuit as the top of this function --
    // reachable here when THIS device redeemed successfully in a PRIOR run
    // of the ceremony and is now retrying (MSG-D25).
    if (/exhausted|already holds a current grant/i.test(reason)) {
      write('(you already joined this room -- opening it)\n')
      return messengerRepl(io, write, ctx, call, node.key)
    }
    throw new Error(reason || 'could not finish joining this room')
  }

  write('(joined -- you can post now)\n')
  return messengerRepl(io, write, ctx, call, node.key)
}

async function openMessenger(io, write) {
  await ensureFirewall(io, write)
  const ctx = await ensureMessengerCore(write)
  const call = (method, params) => ctx.core.dispatch({ id: ctx.nextId++, method, params })

  write('\n')
  write('Opening the messenger now.\n')

  // SC-3a: the corridor fork. THE SAME QUESTION IS ASKED IN BOTH STATES --
  // whether or not this computer already has a conversation -- and Enter is
  // always the safe answer. That uniformity is load-bearing for two separate
  // reasons, both found at the gate (2026-07-20):
  //
  //  1. CLIENT UX (D3). An earlier draft asked a DIFFERENT question when no
  //     room existed ("did someone send you a code?"), so pressing Enter on a
  //     brand-new kit walked the person through minting an invite and
  //     starting hyperswarm+TCP -- a whole corridor ceremony -- when all they
  //     wanted was to send a test message. A receptionist's job is "post a
  //     message, see it worked, close"; making that the hard path and the
  //     corridor the default is backwards.
  //
  //  2. THE SHIPPED VERIFIER DEPENDS ON IT. verify-clean-machine.ps1 runs the
  //     ceremony 16x against the SAME kit directory with ONE fixed stdin
  //     script. Run 1 has no room; runs 2-16 do. With two different questions
  //     there is no single script that drives both: the short one makes run 1
  //     eat the message as a pairing-code answer and post nothing, the long
  //     one posts a stray literal "skip" on every later run. A prompt that
  //     shifts by one line does not fail loudly -- it silently consumes the
  //     NEXT line as its answer, which is the exact shape that has already
  //     cost this codebase real debugging time (see bare-guide-spike.mjs's
  //     layer-4 comment).
  //
  // So: Enter -> reopen-or-create and go straight to the messenger, in BOTH
  // states, no invite, no network. "connect" -> the A2.1 code question below.
  // Menu [2] path A ("start a room here" + mint an invite) is still exactly
  // what the campaign spec asks for; it is simply reached deliberately rather
  // than by default.
  const hasRoom = ctx.core.rooms.size > 0
  const raw = await io.ask(
    (hasRoom
      ? 'You already have a conversation on this computer.\nPress Enter to open it'
      : 'Press Enter to start a conversation on this computer') +
    ', or type connect and press Enter\nto link up with a different computer.\n> '
  )
  if (raw === null) return
  if (!/^connect$/i.test(raw.trim())) {
    const roomKey = await reopenOrCreateRoom(write, ctx, call)
    return messengerRepl(io, write, ctx, call, roomKey)
  }

  // The A2.1 Guided Path question (Mission A2 Band 6's own wording, reused
  // here in spirit -- "Did the other person send you a code? PASTE it here
  // and press Enter. If YOU are starting, just press Enter."): one question
  // distinguishes founder from joiner, no menu number, no jargon.
  write('\n')
  const codeRaw = await io.ask(
    'Did someone send you a code to connect? Paste it here and press Enter.\n' +
    'If you are the one starting this, just press Enter.\n> '
  )
  if (codeRaw === null) return
  const pasted = normalizeCode(codeRaw)
  if (pasted) return joinAsJoiner(io, write, ctx, call, pasted)
  return startAsFounder(io, write, ctx, call)
}

async function anchorOption(io, write) {
  write('\n')
  write('This makes this computer stay on and keep messaging automatically,\n')
  write('even when nobody is sitting at it.\n')
  const raw = await io.ask('Press Enter to set this up, type undo to remove it, or type cancel to go back.\n> ')
  const answer = (raw ?? 'cancel').trim().toLowerCase()
  if (answer === 'cancel' || answer === 'c') return
  // Honest stub -- see file header. No scheduled-task mutation happens here.
  write('(this step is not available yet in the Bare kit -- the always-on anchor has not\n')
  write(' been ported to Bare in this phase; nothing on this computer was changed.)\n')
}

async function statusOption(write) {
  write('\n')
  write('This machine is not the always-on anchor (that step is not available yet in the\n')
  write('Bare kit). Open the messenger (option 2) to confirm the mesh itself is working.\n')
}

// ── menu loop ────────────────────────────────────────────────────────────

function printMenu(write) {
  write('\n')
  write('====================================\n')
  write('  ASYMMFLOW MESH -- GUIDE (Bare)\n')
  write('====================================\n')
  write('[1] Check the connection\n')
  write('[2] Open the messenger\n')
  write('[3] Make this machine the always-on anchor\n')
  write('[4] Show status\n')
  write('[5] Close\n')
  write('\n')
}

async function menuLoop(io, write) {
  for (;;) {
    printMenu(write)
    const raw = await io.ask('> ')
    if (raw === null) return
    const choice = raw.trim()
    try {
      if (choice === '1') await checkConnection(io, write)
      else if (choice === '2') await openMessenger(io, write)
      else if (choice === '3') await anchorOption(io, write)
      else if (choice === '4') await statusOption(write)
      else if (choice === '5') return
      else write('Please type a number from 1 to 5.\n')
    } catch (err) {
      reportError(write, err)
    }
  }
}

/**
 * runGuide({ io }) -- io: an optional injected { onData, onEnd, write }
 * (bare-bridge.mjs's `getRealStdio()` shape). Defaults to the REAL stdio
 * via `getRealStdio()` when omitted -- the shape every prior phase's real
 * entry point uses (apply-bare.mjs/bare-bridge.mjs), so this file needs no
 * runtime detection of its own.
 */
export async function runGuide({ io } = {}) {
  const realIo = io ?? await getRealStdio()
  // DOUBLE-SPACING FIX (Sealed Corridor, found at the SC-3b integration gate
  // by actually LOOKING at a sealed run's output instead of only asserting
  // substrings on it).
  //
  // The bug: `getRealStdio()`'s `write` is `console.log(str)` — and it MUST
  // stay `console.log`, because RULE 2 (PHASE0_NOTES_D2_FLUSH_RACE.md)
  // measured `bare-process`'s `stdout.write()` HANGING 30/30 on a real
  // spawned pipe. But `console.log` appends its own newline, and essentially
  // every string this guide writes already ends in one. Result: every single
  // client-facing line has been rendering with a blank line after it since
  // Phase 2 — the menu, the questions, the verdict, all of it double-spaced.
  //
  // Why nobody caught it: every gate in this campaign asserts with
  // `.includes(...)` on substrings, which is completely blind to the blank
  // lines between them. The suite was green and the client-facing surface
  // looked broken. That is a small, cosmetic instance of exactly the failure
  // shape this campaign keeps naming — a green proof that was not proving
  // what we believed.
  //
  // The fix strips ONE trailing newline before handing the string to
  // console.log, which re-adds it. Net effect per call is unchanged bytes for
  // the common `write('...\n')` case, and `write('\n')` still yields exactly
  // one blank line. RULE 2 is untouched — this is still console.log.
  //
  // Prompts (`write('> ')`, no trailing newline) still land on their own line,
  // because console.log always terminates. That is unchanged behavior, not a
  // regression, and it cannot be fixed without the banned proc.stdout.write.
  const write = (s) => realIo.write(typeof s === 'string' && s.endsWith('\n') ? s.slice(0, -1) : s)
  const guideIo = createGuideIO(realIo)
  write('Welcome. This will walk you through connecting to the other computer.\n')
  await menuLoop(guideIo, write)
  guideIo.close()
  write('\nGoodbye -- this window is safe to close.\n')
  if (messengerCtx) messengerCtx.core.close()
  // SC-3a: the corridor network (if the ceremony ever started one) — same
  // best-effort teardown discipline as every other close() in this file;
  // a teardown failure here must never block the exit below.
  if (corridorNet) { try { await corridorNet.close() } catch { /* best-effort */ } }
  // RULE 3 (PHASE0_GATE_D2_FLUSH_RACE.md): an explicit exit call is
  // LOAD-BEARING, never relied-upon-natural-drain -- found live in this
  // file, not just cited from the doc: the sealed kit driven from a
  // hostile from-scratch directory HUNG intermittently (1 hang in 4
  // sampled spawn-pipe runs) before this fix. Root cause: nothing in this
  // function ever called `process.exit()`/`Bare.exit()`; it relied on the
  // process exiting naturally once stdin reached EOF and no other work was
  // pending -- exactly the "let the loop drain naturally" shape RULE 3
  // warns produces a 10/10 hang in bare-bridge.mjs's own worker. Only
  // guarded by `!io` so a caller that injects its own `io` (a future
  // direct-import test with a fake stdio, not a real spawned process)
  // never has ITS OWN process killed out from under it -- the real exit
  // only fires when this function owns the real stdio it was given.
  if (!io) {
    if (typeof Bare !== 'undefined') Bare.exit(0)
    else process.exit(0)
  }
}

// ── entry point ──────────────────────────────────────────────────────────
// Guarded, NOT unconditional -- an earlier draft of this comment claimed
// Bare has no argv[1]-script-path convention; that was an unverified
// assumption and it was wrong (checked empirically: `Bare.argv[1]` IS the
// invoked script's path, exactly like Node's `process.argv[1]`). Left here
// rather than silently corrected, per this campaign's own transparency
// norm. The guard matters for real behavior, not just tidiness:
// bare-guide-spike.mjs imports `normalizeCode`/`groupInFours` directly for
// pure unit tests (matching guide-spike.mjs's own convention, per the file
// header) -- an unconditional `await runGuide()` would hijack real stdio
// and hang that import.
const argv = typeof Bare !== 'undefined' ? Bare.argv : process.argv
const isMain = argv[1] && new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1') === argv[1].replace(/\\/g, '/')
if (isMain) await runGuide()
