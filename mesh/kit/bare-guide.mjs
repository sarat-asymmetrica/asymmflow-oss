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
import { createMeshNode } from '../host/mesh-node.mjs'
import { createSocialRoom as createSocialRoomNode } from '../host/social-room.mjs'
import { loadRoomRegistry, saveRoomRegistryEntry } from './bare-registry.mjs'
// SC-3b's connection check is imported DYNAMICALLY, inside checkConnection()
// — see that function. It is deliberately NOT a static import here, and the
// reason is measured, not stylistic: see the comment at its use site.

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
const CORESTORE_DIR = `${DATA_DIR}/corestore`

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
async function ensureMessengerCore(write) {
  if (messengerCtx) return messengerCtx
  const seed = loadOrCreateSeed()
  const keys = deviceKeys(seed)
  const core = createBridgeCore({ rooms: new Map(), actor: 'guide', deviceKeys: keys, storageDir: CORESTORE_DIR })
  messengerCtx = { core, keys, nextId: 1 }

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

async function openMessenger(io, write) {
  await ensureFirewall(io, write)
  const ctx = await ensureMessengerCore(write)
  const call = (method, params) => ctx.core.dispatch({ id: ctx.nextId++, method, params })

  write('\n')
  write('Opening the messenger now.\n')
  write('Type a message and press Enter to post it. Type /rooms to list rooms, /exit to leave.\n')
  write('\n')

  let listed = await call('listRooms', {})
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
      title: 'kitchen table', encryptionKey, ts: Date.now(), actor: 'guide',
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

  for (;;) {
    const raw = await io.ask('> ')
    if (raw === null) return
    const line = raw.trim()
    if (line === '') continue
    if (line === '/exit') return
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
