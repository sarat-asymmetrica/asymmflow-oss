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

const DATA_DIR = './data'
const KEYS_DIR = `${DATA_DIR}/keys`
const SEED_FILE = `${KEYS_DIR}/bare-guide-device.seed`
const ROOM_STORAGE_DIR = `${DATA_DIR}/corestore/bare-guide-room`

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

async function checkConnection(io, write) {
  await ensureFirewall(io, write)
  write('\n')
  write('This step is not available yet in the Bare kit.\n')
  write('(no Bare-native connection check exists in this build -- this is an honest gap,\n')
  write(' not a fake result. Use the messenger option to confirm the mesh itself works.)\n')
}

let messengerCtx = null // lazily created, reused across menu visits in one session

async function ensureMessengerCore() {
  if (messengerCtx) return messengerCtx
  const seed = loadOrCreateSeed()
  const keys = deviceKeys(seed)
  const core = createBridgeCore({ rooms: new Map(), actor: 'guide', deviceKeys: keys, storageDir: ROOM_STORAGE_DIR })
  messengerCtx = { core, keys, nextId: 1 }
  return messengerCtx
}

async function openMessenger(io, write) {
  await ensureFirewall(io, write)
  const ctx = await ensureMessengerCore()
  const call = (method, params) => ctx.core.dispatch({ id: ctx.nextId++, method, params })

  write('\n')
  write('Opening the messenger now.\n')
  write('Type a message and press Enter to post it. Type /rooms to list rooms, /exit to leave.\n')
  write('\n')

  let listed = await call('listRooms', {})
  let roomKey = listed.result?.[0]?.roomKey
  if (!roomKey) {
    const created = await call('createSocialRoom', { title: 'kitchen table' })
    if (!created.ok) { write(`(could not open a room: ${created.error})\n`); return }
    roomKey = created.result.roomKey
    write('(created a new room for this kit -- "kitchen table")\n')
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
  const write = (s) => realIo.write(s)
  const guideIo = createGuideIO(realIo)
  write('Welcome. This will walk you through connecting to the other computer.\n')
  await menuLoop(guideIo, write)
  guideIo.close()
  write('\nGoodbye -- this window is safe to close.\n')
  if (messengerCtx) messengerCtx.core.close()
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
