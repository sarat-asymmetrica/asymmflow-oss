// guide.mjs — Mission A2.1 "Reception Grade", Band 6 "The Guided Path"
// (mesh/docs/MISSION_A2_CORRIDOR_SPEC.md §Band 6). The ONE entry point a
// receptionist ever touches: START_HERE.cmd launches this file under the
// kit's bundled node, and from here on it is plain questions, paste-and-
// Enter, and menu numbers — never a command line, never an argument typed
// by hand (owner ruling R6, field report FR-1c).
//
// DESIGN CONSTRAINT, load-bearing: this file imports ONLY node: built-ins.
// FR-1a was a top-level `import 'holesail'` in probe.mjs that crashed the
// built kit outside the repo tree, because build-kit.mjs's node_modules
// closure walk (see build-kit.mjs §1-2) is seeded from the kit's FOUR entry
// files (kit-host/kit-repl/kit-net/kit-registry.mjs) — never from probe.mjs,
// and never from this file either. Anything this file imported from npm
// (or from a local module that itself imports npm packages — e.g. probe.mjs
// pulls in hyperdht/hyperswarm/hypercore-id-encoding) would silently work
// in the repo (Node's module resolution escapes upward into mesh/node_modules)
// and silently break on the receptionist's machine, the exact failure mode
// this mission exists to close. Every other kit surface this file needs
// (the connection probe, the messenger, the anchor) is reached ONLY by
// spawning the existing .mjs/.cmd entry points as separate child processes
// (I1: reuse, never reimplement) — never by importing their code.
//
// LAYOUT, load-bearing: this file ships at TWO different relative depths
// depending on where it's run from —
//   - dev tree (mesh/kit/guide.mjs): every kit file is a SIBLING in this
//     same directory (probe.mjs, install_anchor.cmd, anchor_status.cmd, ...).
//   - a BUILT kit (kit\guide.mjs under the kit root): probe.mjs travels
//     alongside it in kit\ (build-kit.mjs's PROBE_CLUSTER placement), but
//     node.exe, run_mesh.cmd, and the whole anchor cluster sit ONE LEVEL UP
//     at the kit root (build-kit.mjs's ANCHOR_CLUSTER placement) — see
//     build-kit.mjs's CORRIDOR_OPTIONAL_FILES comment for why the two
//     clusters disagree about where they live.
// resolveKitPath() below checks both depths for any name so this file
// behaves correctly in either layout without caring which one it's in.

import readline from 'node:readline'
import { existsSync } from 'node:fs'
import { spawn, spawnSync } from 'node:child_process'
import { join, dirname, basename } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))

// Must match build-kit.mjs's SETUP_FIREWALL_CMD template's RULE_IN exactly —
// that template is the one thing that actually creates this rule, so this
// name is a cross-file contract, not a free choice. (Single-writer note:
// build-kit.mjs is also this coder's file this wave, so both sides are kept
// in sync by hand; guide-spike.mjs's I4/name checks do not cover this, a
// human diff of the two literals is the real guard.)
const FIREWALL_RULE_IN = 'AsymmFlow Mesh Kit (in)'

// ── pure helpers (guide-spike.mjs unit-tests these directly, no process) ──

/** WhatsApp-paste tolerance: a code read aloud in groups of four picks up
 * ordinary spaces; a code copy-pasted from a chat bubble can pick up a
 * trailing newline or a stray non-breaking/zero-width space (U+00A0,
 * U+200B). Strip every kind of whitespace this way — never throws on odd
 * input, including empty/undefined. */
export function normalizeCode(input) {
  return String(input ?? '').replace(/[\s\u00A0\u200B]+/g, '').trim()
}

/** Groups a code into 4-character chunks for reading aloud — same
 * convention as probe.mjs's own z32Groups(), duplicated here rather than
 * imported (see the file header: importing from probe.mjs would drag its
 * npm-package imports into this file's module graph). */
export function groupInFours(code) {
  return String(code).match(/.{1,4}/g)?.join(' ') ?? String(code)
}

/** Finds the first existing path for `parts` at either of this kit's two
 * possible depths (see the file header's LAYOUT note). Returns the full
 * path, or null if neither depth has it — every caller treats null as a
 * plain "not available here" rather than a crash. */
export function resolveKitPath(...parts) {
  for (const base of [join(__dirname, '..'), __dirname]) {
    const candidate = join(base, ...parts)
    if (existsSync(candidate)) return candidate
  }
  return null
}

function resolveNodeExe() {
  return resolveKitPath('node.exe') || 'node'
}

// ── child-process plumbing ─────────────────────────────────────────────

/** Runs a .cmd file with a fully inherited console — the right shape for
 * anything the receptionist should watch happen live (the messenger REPL,
 * an elevation popup, a scheduled-task install). Resolves with the exit
 * code; a spawn failure (missing cmd.exe, permissions) is reported the same
 * way rather than throwing, so callers never need a second error path. */
function runCmdInherited(cmdPath, args = []) {
  return new Promise((resolve) => {
    const child = spawn(process.env.ComSpec || 'cmd.exe', ['/c', cmdPath, ...args], {
      cwd: dirname(cmdPath), stdio: 'inherit',
    })
    child.on('close', (code) => resolve({ code }))
    child.on('error', (err) => { console.log(`could not start ${basename(cmdPath)}: ${err.message}`); resolve({ code: 1 }) })
  })
}

/** Runs probe.mjs under the resolved node runtime, streaming its output
 * live (so a human watching sees the same PASS/FAIL lines probe.mjs always
 * prints) while ALSO buffering it, so the verdict line can be pulled back
 * out and re-printed large below. Never throws — a spawn failure comes back
 * as a RED-shaped result the caller's normal error path already handles. */
function runProbe(args) {
  return new Promise((resolve) => {
    const probeScript = resolveKitPath('probe.mjs')
    if (!probeScript) { resolve({ code: 1, output: 'probe.mjs was not found in this kit — this guide only works inside a complete kit folder' }); return }
    // stdin is 'ignore', not 'inherit': probe.mjs never reads stdin (it is a
    // fixed-timeout batch check, no prompts of its own) — sharing this
    // guide's already-open readline stdin with a second consumer caused a
    // real hang/deadlock here during testing (Node's "unsettled top-level
    // await" watchdog eventually fired). stdout/stderr stay piped so the
    // output can be streamed live AND buffered for the verdict re-print.
    const child = spawn(resolveNodeExe(), [probeScript, ...args], {
      cwd: dirname(probeScript), stdio: ['ignore', 'pipe', 'pipe'],
    })
    let output = ''
    child.stdout.on('data', (d) => { process.stdout.write(d); output += d.toString('utf8') })
    child.stderr.on('data', (d) => { process.stderr.write(d); output += d.toString('utf8') })
    child.on('close', (code) => resolve({ code, output }))
    child.on('error', (err) => resolve({ code: 1, output: `could not start probe: ${err.message}` }))
  })
}

/**
 * createGuideIO(input, output) -> { ask(prompt), pause(), resume(), close() }
 *
 * NOT built on sequential `rl.question()` calls — that was the first draft
 * here, and it has a real, reproducible bug with piped/scripted stdin (the
 * exact shape guide-spike.mjs's gate needs, and arguably closer to how a
 * receptionist's paste-then-Enter behaves than slow interactive typing):
 * `rl.question()` only starts listening for the NEXT 'line' event at the
 * moment it's called. If the input stream already has more than one line
 * buffered (e.g. a script did `printf '1\ncode\n5\n' | node guide.mjs`, or a
 * human pastes a whole code-plus-Enter in one paste event), the SECOND
 * line's 'line' event can fire before this file's code gets around to
 * calling `rl.question()` a second time — readline does not queue it for a
 * future question(), so that answer is silently dropped and the next
 * question() hangs forever waiting for a line that already went by
 * (reproduced in isolation while building this file: a two-question
 * script over piped stdin hung on the second question 100% of the time).
 * The fix: listen to 'line' unconditionally from the start and maintain our
 * own FIFO queue — `ask()` drains the queue if a line is already waiting,
 * otherwise registers to receive the next one. No line is ever lost
 * regardless of the timing between when it arrives and when it's asked for.
 */
function createGuideIO(input, output) {
  const rl = readline.createInterface({ input, output })
  const queue = []
  const waiters = []
  let closed = false
  rl.on('line', (line) => {
    if (waiters.length) waiters.shift()(line)
    else queue.push(line)
  })
  rl.on('close', () => {
    closed = true
    while (waiters.length) waiters.shift()(null) // null = stdin ended, no more input ever
  })
  function nextLine() {
    if (queue.length) return Promise.resolve(queue.shift())
    if (closed) return Promise.resolve(null)
    return new Promise((resolve) => waiters.push(resolve))
  }
  return {
    // Returns the typed line, or null if stdin closed before one arrived
    // (a scripted/hermetic run out of input, or a closed pipe) — every
    // caller below treats null as "nothing more to do here", never a crash.
    async ask(prompt) {
      output.write(prompt)
      return nextLine()
    },
    pause: () => rl.pause(),
    resume: () => rl.resume(),
    close: () => rl.close(),
  }
}

// ── firewall pre-step (item 3 of the band) ─────────────────────────────

/** True if the kit's own firewall rule is already registered. Read-only —
 * `netsh advfirewall firewall show rule` never elevates or mutates
 * anything, safe to call every time this guide starts. Non-Windows / a
 * netsh that errors out is treated as "not present" — the caller's offer
 * step then degrades to a skip-with-warning, never a crash. */
function firewallRulePresent() {
  if (process.platform !== 'win32') return false
  try {
    const res = spawnSync('netsh', ['advfirewall', 'firewall', 'show', 'rule', `name=${FIREWALL_RULE_IN}`], { encoding: 'utf8' })
    const out = `${res.stdout || ''}${res.stderr || ''}`
    return res.status === 0 && !/No rules match the specified criteria/i.test(out)
  } catch {
    return false
  }
}

let firewallOffered = false // asked at most once per guide session (spec item 3: "first-run")

/** Offers the one-time firewall step, once, before the first connection
 * check or messenger open. Declining or a failed run is a warning, never a
 * blocker — spec item 3: "continue with a warning, don't block." */
async function ensureFirewall(io) {
  if (firewallOffered) return
  firewallOffered = true

  if (!resolveKitPath('node.exe')) {
    // A dev-tree run (no bundled node.exe) — setup_firewall.cmd itself
    // requires a bundled node.exe to target and refuses without one, so
    // there is nothing useful to offer here.
    return
  }
  if (firewallRulePresent()) return

  console.log('')
  console.log('Before we connect, this computer needs one quick permission.')
  console.log('Windows will ask "Do you want to allow this app to make changes?" — click Yes.')
  console.log('That is the ONE popup in this whole process; nothing else on this computer changes.')
  const answer = await io.ask('Press Enter to continue, or type skip and press Enter to skip this for now.\n> ')
  if (answer === null) return // stdin closed — nothing more we can ask; move on quietly
  if (/^skip$/i.test(answer.trim())) {
    console.log('Skipping for now — the connection might not work until this step is done.')
    return
  }
  const setupCmd = resolveKitPath('setup_firewall.cmd')
  if (!setupCmd) {
    console.log('(setup_firewall.cmd was not found next to this kit — skipping)')
    return
  }
  const { code } = await runCmdInherited(setupCmd)
  if (code !== 0) console.log('That did not finish cleanly — continuing anyway; the connection might still work.')
}

// ── menu actions ─────────────────────────────────────────────────────────

function printVerdictLarge(verdict) {
  const line = '='.repeat(Math.max(40, verdict.length + 8))
  console.log('')
  console.log(line)
  console.log(`   ${verdict}`)
  console.log(line)
  console.log('')
  console.log(`Read this word to the person on the call: ${verdict}`)
}

async function checkConnection(io) {
  await ensureFirewall(io)
  console.log('')
  const raw = await io.ask('Did the other person send you a code? PASTE it here and press Enter.\nIf YOU are starting, just press Enter.\n> ')
  const code = normalizeCode(raw) // raw === null (stdin closed) normalizes to '' — same as a bare Enter, listen mode
  console.log('')

  let result
  if (!code) {
    console.log('Starting up and waiting for the other machine to connect...')
    result = await runProbe(['--listen'])
  } else {
    console.log(`Trying to reach the other machine using code: ${groupInFours(code)}`)
    result = await runProbe(['--dial', code])
  }

  const match = result.output.match(/CORRIDOR (GREEN|AMBER|RED)/)
  if (match) {
    printVerdictLarge(match[0])
  } else {
    console.log('Could not read a clear result from that check — see the lines above for what happened.')
  }
}

async function openMessenger(io) {
  await ensureFirewall(io)
  const runMesh = resolveKitPath('run_mesh.cmd')
  if (!runMesh) {
    console.log('run_mesh.cmd was not found — this guide only works inside a complete kit folder.')
    return
  }
  console.log('')
  console.log('Opening the messenger now. Type /exit there when you want to come back to this menu.')
  console.log('')
  // The messenger owns the console fully while it runs (its own readline
  // prompt) — pause this guide's own reader so the two don't fight over
  // stdin, then hand the terminal to run_mesh.cmd with stdio inherited.
  io.pause()
  await runCmdInherited(runMesh)
  io.resume()
  console.log('')
  console.log('Back at the main menu.')
}

async function anchorOption(io) {
  console.log('')
  console.log('This makes this computer stay on and keep messaging automatically,')
  console.log('even when nobody is sitting at it.')
  const raw = await io.ask('Press Enter to set this up, type undo to remove it, or type cancel to go back.\n> ')
  const answer = (raw ?? 'cancel').trim().toLowerCase() // stdin closed -> treat like "cancel", never act unattended

  if (answer === 'cancel' || answer === 'c') return

  if (answer === 'undo') {
    const uninstallCmd = resolveKitPath('uninstall_anchor.cmd')
    if (!uninstallCmd) { console.log('uninstall_anchor.cmd was not found — this guide only works inside a complete kit folder.'); return }
    await runCmdInherited(uninstallCmd)
    return
  }

  const installCmd = resolveKitPath('install_anchor.cmd')
  if (!installCmd) { console.log('install_anchor.cmd was not found — this guide only works inside a complete kit folder.'); return }
  await runCmdInherited(installCmd)

  const statusCmd = resolveKitPath('anchor_status.cmd')
  if (statusCmd) {
    console.log('')
    console.log('Checking on it now:')
    console.log('')
    await runCmdInherited(statusCmd)
  }
}

async function statusOption() {
  console.log('')
  const statusCmd = resolveKitPath('anchor_status.cmd')
  const anchorLog = resolveKitPath('data', 'keys', 'anchor.log')
  if (statusCmd && anchorLog) {
    console.log('Reading the anchor status now — read every line out loud if support asks:')
    console.log('')
    await runCmdInherited(statusCmd)
    return
  }
  console.log('This machine is not the always-on anchor yet, so there is no anchor status to show.')
  console.log('Open the messenger (option 2) and type /status there — read every line out loud to support.')
}

// ── menu loop ────────────────────────────────────────────────────────────

function printMenu() {
  console.log('')
  console.log('====================================')
  console.log('  ASYMMFLOW MESH — GUIDE')
  console.log('====================================')
  console.log('[1] Check the connection')
  console.log('[2] Open the messenger')
  console.log('[3] Make this machine the always-on anchor')
  console.log('[4] Show status')
  console.log('[5] Close')
  console.log('')
}

/** Every menu action's failure lands here — ONE plain sentence for the
 * phone call, then a fold line, then the raw detail for support to read
 * off-call. Spec item 5. Never lets an action's error escape and crash the
 * whole guide — the menu always comes back. */
function reportError(err) {
  console.log('')
  console.log('Something went wrong — read this line to the person on the phone:')
  console.log(`  ${err?.message || String(err)}`)
  console.log('--- details for support ---')
  console.log(err?.stack || String(err))
}

async function menuLoop(io) {
  for (;;) {
    printMenu()
    const raw = await io.ask('> ')
    if (raw === null) return // stdin closed (headless run out of scripted input) — same as choosing Close
    const choice = raw.trim()
    try {
      if (choice === '1') await checkConnection(io)
      else if (choice === '2') await openMessenger(io)
      else if (choice === '3') await anchorOption(io)
      else if (choice === '4') await statusOption()
      else if (choice === '5') return
      else console.log('Please type a number from 1 to 5.')
    } catch (err) {
      reportError(err)
    }
  }
}

export async function runGuide({ input = process.stdin, output = process.stdout } = {}) {
  const io = createGuideIO(input, output)
  output.write('Welcome. This will walk you through connecting to the other computer.\n')
  await menuLoop(io)
  io.close()
  output.write('\nGoodbye — this window is safe to close.\n')
}

// ── CLI entry point ─────────────────────────────────────────────────────
const isMain = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]
if (isMain) {
  await runGuide()
}
