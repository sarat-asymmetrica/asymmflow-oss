// build-kit.mjs — Mission U1.5: assembles the portable field kit (MSG-D24,
// mesh/docs/MESSENGER_UI_CAMPAIGN.md §2 W-UI-1.5) into TWO ready folders,
// `dist/Machine-A/` and `dist/Machine-B/` — unzip one on each machine, run
// `run_mesh.cmd`, follow README_KITCHEN_TABLE.txt.
//
// Layout per machine (DP1 plane doctrine): `portable.flag`, `host/`, `kit/`,
// `dist/reducer.wasm`, `node_modules/`, `data/corestore/` + `data/keys/`
// SIBLINGS, `run_mesh.cmd`, `README_KITCHEN_TABLE.txt`.
//
// PRUNING, done for real rather than guessed (a wrong guess here ships a
// kit that fails at the actual kitchen table with a missing-module error —
// worse than a bigger zip):
//   1. SOURCE: a real import-graph walk from the kit's four entry files
//      (kit-host/kit-repl/kit-net/kit-registry.mjs) through relative
//      imports — only the host/*.mjs files actually reachable are copied
//      (naturally excludes every *-spike.mjs, peer.mjs, missiond-*.mjs —
//      none of them are imported by the kit's own code).
//   2. node_modules: a real package.json dependency-graph walk (deps +
//      optionalDependencies, recursively) seeded from the bare import
//      specifiers that walk (1) actually found, NOT from mesh/package.json's
//      top-level list — verified against the installed source first
//      (per AGENT_GATE_LEDGER.md's standing standard): `autobase`'s own
//      package.json lists `hyperbee`, `hyperschema`, and `protomux-wakeup`
//      as ITS dependencies, so a naive "grep our own code for imports"
//      prune would have deleted packages autobase needs internally and
//      broken the kit silently. The packages this walk DOES drop —
//      blind-peer, blind-peering, holesail(+ its holesail-client/-server/
//      -logger), hyperdb, hyperbee2, hyperswarm-capability,
//      hyper-cmd-lib-keys, ip-ban-list, and their own leaf-only deps — were
//      checked to have NO reverse-dependency from anything the kit actually
//      needs (grep across every node_modules/*/package.json).
//
// Run: npm run buildkit

import { existsSync, mkdirSync, readFileSync, writeFileSync, cpSync, rmSync, readdirSync, statSync } from 'node:fs'

// cmd.exe misparses LF-only batch files (chopped tokens, broken goto labels),
// and the repo's .gitattributes LF baseline means any .cmd that ever passes
// through git comes out LF. Built kits are not git checkouts, so the field
// guarantee lives HERE: every .cmd is CRLF-normalized at write/copy time.
// (Mission A2 gate finding, 2026-07-19.)
const toCrlf = (s) => s.replace(/\r?\n/g, '\r\n')
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'

const __dirname = dirname(fileURLToPath(import.meta.url))
const meshRoot = join(__dirname, '..')
const kitDir = __dirname
const hostDir = join(meshRoot, 'host')
const distOut = join(kitDir, 'dist')

// Mission A2 "The Corridor", Band 2 (mesh/docs/MISSION_A2_CORRIDOR_SPEC.md):
// `--bundle-node` copies THIS running node.exe (pinned v22 LTS, I5) into
// each machine folder so the receptionist machine needs zero installs.
const BUNDLE_NODE = process.argv.includes('--bundle-node')

// Filenames FIXED by the spec (§6) so this band can consume Band 1 (probe)
// and Band 3 (anchor) output by name alone, built in parallel, with no
// coordination beyond this list. Each is copied into the built kit ONLY if
// present at build time — a band that hasn't landed yet is a graceful skip,
// never a build failure (every band is independently testable this way).
//
// Split into two placement clusters — found the hard way running kit2-spike
// (Mission A2, Band 2 audit): a blanket ".mjs -> kit/, else -> root" rule
// (the original shape of this list) silently breaks BOTH bands, because
// each band's own frozen .cmd/.ps1 files hardcode where their .mjs sits
// relative to themselves, and the two bands hardcode OPPOSITE answers:
//   - PROBE_CLUSTER: run_probe.cmd/run_probe_dial.cmd (C1, frozen) call a
//     BARE `probe.mjs` at their own %~dp0, and separately check `..\node.exe`
//     one level up — i.e. C1 built these assuming the whole probe trio
//     lives together under kit/ (one level below the bundled node.exe),
//     matching README_CORRIDOR.txt's own "kit/probe.mjs" phrasing. All
//     three copy verbatim into target/kit/.
//   - ANCHOR_CLUSTER: run_anchor.cmd / install_anchor.cmd / uninstall_anchor.cmd
//     / anchor_status.cmd / install_anchor.ps1 / uninstall_anchor.ps1 (C3,
//     frozen) all resolve node.exe, data/, and each other via THEIR OWN
//     %~dp0 / $PSScriptRoot — i.e. C3 built the whole cluster assuming it
//     sits at the kit ROOT, beside node.exe and data/ (install_anchor.ps1
//     literally does `Join-Path $kitDir 'anchor.mjs'` where $kitDir is its
//     own script directory). These copy verbatim into target root.
//   - The one real conflict: anchor.mjs ALSO does `import ... from
//     './kit-host.mjs'` — which only resolves if anchor.mjs sits next to
//     kit-host.mjs, i.e. target/kit/, the OPPOSITE of where its own cmd/ps1
//     siblings need it. Rather than edit anchor.mjs (frozen — C3's file,
//     out of Band 2's authority) or duplicate the entire host/ closure a
//     second time just to satisfy one import, this build script places a
//     TRANSFORMED COPY of anchor.mjs at target root with that one import
//     specifier rewritten to `./kit/kit-host.mjs` (see anchorSourceForRoot
//     below) — a build-time packaging decision on the OUTPUT artifact, not
//     an edit to the source file in mesh/kit/.
const PROBE_CLUSTER_FILES = ['probe.mjs', 'run_probe.cmd', 'run_probe_dial.cmd']
const ANCHOR_CLUSTER_FILES = [
  'anchor.mjs', 'install_anchor.cmd', 'uninstall_anchor.cmd', 'anchor_status.cmd', 'run_anchor.cmd',
  'install_anchor.ps1', 'uninstall_anchor.ps1',
]
// Mission A2.1, Band 6 (mesh/docs/MISSION_A2_CORRIDOR_SPEC.md addendum):
// guide.mjs imports ONLY node: built-ins (see its own file header) and
// resolves every other kit file at either of two relative depths — so, like
// PROBE_CLUSTER, it travels under target/kit/ (beside probe.mjs) with no
// import-rewrite needed (unlike anchor.mjs's one load-bearing rewrite
// above). START_HERE.cmd itself is NOT in this list — it is an embedded
// template (see START_HERE_CMD below), written unconditionally to the
// built kit ROOT the same way RUN_MESH_CMD is, never copied from a source
// file (the dev-tree mesh/kit/START_HERE.cmd is a different, same-directory
// variant for running straight out of the repo — see its own header).
const GUIDE_CLUSTER_FILES = ['guide.mjs']
const CORRIDOR_OPTIONAL_FILES = [...PROBE_CLUSTER_FILES, ...ANCHOR_CLUSTER_FILES, ...GUIDE_CLUSTER_FILES]

// The one load-bearing rewrite described above. Guarded: throws (fails the
// build loudly) rather than silently shipping a broken anchor.mjs if a
// future edit to the source file changes this import line's exact text —
// a quiet mismatch here would only surface as a cryptic MODULE_NOT_FOUND
// on the receptionist's machine, the worst possible place to find it.
function anchorSourceForRoot(src) {
  const needle = "from './kit-host.mjs'"
  const replacement = "from './kit/kit-host.mjs'"
  if (!src.includes(needle)) {
    throw new Error(`anchor.mjs no longer contains ${JSON.stringify(needle)} — build-kit.mjs's root-copy rewrite (Mission A2 Band 2) is stale and must be updated to match anchor.mjs's real import`)
  }
  return src.replace(needle, replacement)
}

// ── 0. make sure the reducer is fresh ──────────────────────────────────────
console.log('building reducer.wasm...')
execFileSync(process.execPath, [join(meshRoot, 'scripts', 'build-reducer.mjs')], { stdio: 'inherit', cwd: meshRoot })
const wasmPath = join(meshRoot, 'dist', 'reducer.wasm')
if (!existsSync(wasmPath)) throw new Error(`expected ${wasmPath} after build — did build-reducer.mjs move?`)

// ── 1. local source closure: walk relative imports from the kit's own
//    entry files, collecting every reachable host/*.mjs and kit/*.mjs file
//    PLUS every bare (npm package) specifier encountered anywhere in that
//    reachable set. ─────────────────────────────────────────────────────
const KIT_ENTRY_FILES = ['kit-host.mjs', 'kit-repl.mjs', 'kit-net.mjs', 'kit-registry.mjs'].map((f) => join(kitDir, f))

function walkLocalClosure(entryFiles) {
  const visited = new Set()
  const bareSpecs = new Set()
  const queue = [...entryFiles]
  // Real ES module import syntax ONLY (anchored to the statement keyword),
  // never a bare `from(` / `import(...)` substring match — an earlier draft
  // matched `Buffer.from('meshinvite.v1:')` inside capability.mjs and other
  // in-code string literals that merely contain the word "from".
  const importRes = [
    /^\s*import\s+[\s\S]*?\bfrom\s+['"]([^'"]+)['"]/gm, // import X from '...'
    /^\s*import\s+['"]([^'"]+)['"]/gm,                  // bare side-effect import '...'
    /\bimport\(\s*['"]([^'"]+)['"]\s*\)/g,              // dynamic import('...')
  ]
  while (queue.length) {
    const file = queue.shift()
    if (visited.has(file)) continue
    if (!existsSync(file)) throw new Error(`kit source closure: missing ${file}`)
    visited.add(file)
    const src = readFileSync(file, 'utf8')
    const specs = []
    for (const re of importRes) {
      re.lastIndex = 0
      let m
      while ((m = re.exec(src))) specs.push(m[1])
    }
    for (const spec of specs) {
      if (spec.startsWith('.')) {
        const resolved = join(dirname(file), spec)
        if (existsSync(resolved)) queue.push(resolved)
      } else if (!spec.startsWith('node:')) {
        bareSpecs.add(spec.startsWith('@') ? spec.split('/').slice(0, 2).join('/') : spec.split('/')[0])
      }
    }
  }
  return { files: visited, packages: bareSpecs }
}

const { files: localFiles, packages: seedPackages } = walkLocalClosure(KIT_ENTRY_FILES)
console.log(`local source closure: ${localFiles.size} file(s), seed packages: ${[...seedPackages].sort().join(', ')}`)

// ── 2. node_modules closure: BFS package.json dependencies +
//    optionalDependencies (native prebuilds often ride optional) from the
//    seed packages found above. ───────────────────────────────────────────
function walkPackageClosure(rootNodeModules, seedNames) {
  const visited = new Set()
  const queue = [...seedNames]
  while (queue.length) {
    const name = queue.shift()
    if (visited.has(name)) continue
    const pkgDir = join(rootNodeModules, ...name.split('/'))
    const pkgJsonPath = join(pkgDir, 'package.json')
    if (!existsSync(pkgJsonPath)) continue // not installed for this platform — nothing to copy, nothing to recurse into
    visited.add(name)
    let pkgJson
    try { pkgJson = JSON.parse(readFileSync(pkgJsonPath, 'utf8')) } catch { continue }
    const deps = { ...(pkgJson.dependencies ?? {}), ...(pkgJson.optionalDependencies ?? {}) }
    for (const dep of Object.keys(deps)) queue.push(dep)
  }
  return visited
}

const rootNodeModules = join(meshRoot, 'node_modules')
const packageClosure = walkPackageClosure(rootNodeModules, seedPackages)
console.log(`node_modules closure: ${packageClosure.size} package(s) (of ${(() => { try { return readFileSync(join(rootNodeModules, '.package-lock.json'), 'utf8').match(/"node_modules\//g)?.length ?? '?' } catch { return '?' } })()} installed)`)

// ── templates (declared before use — assembleMachine below references them) ─
// Mission A2, Band 2, item 2: every generated .cmd entrypoint prefers the
// kit's OWN bundled node.exe (sitting next to this .cmd, at %~dp0node.exe —
// present when built with --bundle-node) over whatever "node" is on PATH.
// A receptionist machine with nothing installed still runs the kit; a
// developer machine with node already on PATH is unaffected either way.
// Shared by every generated launcher .cmd (run_mesh.cmd here; kept as one
// function so a future addition can't drift from this preference order).
function nodeLaunchPreamble() {
  return `set "NODE_EXE=%~dp0node.exe"
if exist "%NODE_EXE%" goto haveNode

where node >nul 2>nul
if errorlevel 1 (
  echo.
  echo Node.js was not found on this computer.
  echo Install it from https://nodejs.org/  ^(the LTS installer, default options are fine^),
  echo then double-click this file again.
  echo.
  pause
  exit /b 1
)
set "NODE_EXE=node"

:haveNode`
}

const RUN_MESH_CMD = `@echo off
setlocal
cd /d "%~dp0"

${nodeLaunchPreamble()}
"%NODE_EXE%" kit\\kit-host.mjs --data data

echo.
echo (the kit has stopped - press any key to close this window)
pause >nul
`

// Mission A2.1, Band 6, item 1 (mesh/docs/MISSION_A2_CORRIDOR_SPEC.md
// addendum): the ONE entry point a receptionist ever double-clicks (owner
// ruling R6 — never a command line, never a typed argument). Sits at the
// BUILT KIT ROOT, beside run_mesh.cmd and node.exe, and launches
// kit\guide.mjs (guide.mjs's own GUIDE_CLUSTER placement, alongside
// probe.mjs) under the bundled runtime via the SAME nodeLaunchPreamble()
// every other generated launcher in this file uses — never a second,
// drifted copy of the node-resolution logic.
const START_HERE_CMD = `@echo off
setlocal
cd /d "%~dp0"

echo ==================================================
echo   ASYMMFLOW MESH - START HERE
echo ==================================================
echo.

${nodeLaunchPreamble()}
"%NODE_EXE%" kit\\guide.mjs

echo.
echo (the guide has stopped - press any key to close this window)
pause >nul
`

const README_TEXT = `ASYMMFLOW MESH — THE KITCHEN TABLE KIT (v2)
==============================================

*******************************************************************
*  READ THIS BOX FIRST                                             *
*                                                                   *
*  - This kit is ONE self-contained folder. It touches NOTHING     *
*    outside itself — no other program, no other folder on this    *
*    computer, no company system.                                  *
*  - It contains ONLY made-up, synthetic demo data. Never type in  *
*    a real customer, a real document, or real business details.   *
*  - If this computer already runs other business software (an     *
*    office PC, for example), that software's data is NEVER read   *
*    or written by this kit. They cannot see each other.           *
*  - Deleting this folder deletes EVERYTHING this kit ever did on  *
*    this computer. There is no undo, no company backup, no cloud  *
*    copy anywhere.                                                *
*******************************************************************

What this is
-------------
A small, self-contained messenger + real file-transfer test between TWO
computers. No account, no server, no company involved — just this folder,
copied once to each machine.

SYNTHETIC DATA ONLY. Do not use real names, real documents, or real
business information with this kit. Pick made-up names when asked (e.g.
"ana", "sam") and use a throwaway test file for the attachment step.

PHONE SCRIPT — for a remote-directed test
-------------------------------------------
Use this if someone is walking you through this on a phone call and you
have never done it before. Read each "SAY" line to yourself, then do the
"THEY TYPE" line exactly as written (case doesn't matter, but spelling
does). If anything on screen doesn't look like what's described, STOP and
say what you actually see — don't guess and keep going.

  SAY: "I'm double-clicking run_mesh.cmd now."
  DO:  double-click run_mesh.cmd. A black window opens.

  SAY: "It's asking me to choose a name."          (first run only)
  TYPE: ana                                         (or whatever name you're told)
  DO:  press Enter.

  SAY: "It printed a few lines. I'll read them: ..." then read whatever
       appears out loud, word for word — it's designed to be readable.

  If you get stuck at ANY point, TYPE: /status
  SAY: "Reading /status: ..." then read every line out loud. This tells
       the person on the phone exactly what your machine sees — your
       room, whether anyone's connected, and where files are saved.

  If you typed /join before and it says something about the invite being
  used already — that's fine, it means you're already in. TYPE: /join
  again with the SAME code; the kit now just reopens the room quietly.

  When in doubt: TYPE: /help    for the full list of what you can type.

What actually happens (read this before you start)
-----------------------------------------------------
- The two computers talk to each other DIRECTLY — no company server sees
  your messages. First packets can leave your home network in two ways:
    1. hyperswarm — a public P2P discovery network (like BitTorrent's DHT)
       that helps the two computers FIND each other's address. It is tried
       automatically if it works on your network.
    2. Direct LAN connection — the REQUIRED fallback (/connect <ip:port>,
       see below). This works even if hyperswarm can't reach the DHT
       (e.g. a locked-down home router) — both computers just need to be
       on the same Wi-Fi/network, or connected by a cable.
  Either way, EVERY message and file is encrypted before it ever leaves
  your computer. What actually crosses the network is unreadable
  ciphertext plus routing metadata (which computer is talking to which) —
  never plaintext content.
- Deleting this folder is REAL, permanent deletion of THIS computer's copy
  — there is no "undo," no cloud backup, no company copy. If both
  computers delete their folders, the room is gone everywhere.

What you need
---------------
- Two Windows computers on the same Wi-Fi network (easiest), OR any
  network path between them (a direct LAN connection also works).
- Node.js installed on BOTH computers — https://nodejs.org/ (the LTS
  installer; accept the defaults). run_mesh.cmd will tell you if it's
  missing.
- A throwaway test file on ONE of the computers, for the file-transfer step
  (any small file — a photo, a text file — nothing sensitive).

Setup
------
1. Copy this WHOLE folder to Machine A. Copy the matching "Machine-B"
   folder to Machine B (they are DIFFERENT folders — do not mix them up
   or run the same one twice).
2. On EACH machine: double-click run_mesh.cmd. A black window opens.
3. The FIRST time it runs, it asks you to choose a synthetic actor name
   (e.g. "ana" on Machine A, "sam" on Machine B). Type one word, press
   Enter. This name is saved — you won't be asked again on this machine.

The ceremony (do this together, one step at a time)
------------------------------------------------------
On Machine A (the "founder"):
  4.  Type:  /create kitchen table test
      This founds a new, encrypted room and starts listening for a direct
      connection. It will print a line like:
        (LAN fallback listening on port 4300 for this room — give the
        other machine this IP:4300)
      Find Machine A's IP address (Windows: open a Command Prompt and run
      "ipconfig", look for "IPv4 Address") and write down IP:PORT.
  5.  Type:  /invite
      This prints a long code starting with "asymm-room2." — this is the
      invite. Read/type/send it to whoever is at Machine B.

On Machine B (the "joiner"):
  6.  Type:  /join <paste the invite code here>
      It prints YOUR PAIRING CODE (a long string of letters/numbers) and
      then waits.

Back on Machine A:
  7.  Type:  /addwriter <paste Machine B's pairing code here>
      This admits Machine B into the room.

On EITHER machine (whichever one doesn't already have a connection):
  8.  Type:  /connect <Machine A's IP:PORT from step 4>
      (Only needed if hyperswarm didn't already connect the two of you —
      it doesn't hurt to run it either way.)

Back on Machine B:
  9.  Within a few seconds, Machine B prints "joined ... — you can post
      now" on its own — nothing more to type for this step.

Now talk and test file transfer
----------------------------------
 10. On either machine, just type a message and press Enter to post it.
     Try tagging urgency: /expect urgent your message here
 11. File transfer test — on the machine WITH the test file:
       /attach C:\\path\\to\\your\\test\\file.txt  a note about this file
     It prints a sha256 (a fingerprint of the file's exact bytes) and a
     seq number (the #N shown next to the message).
 12. On the OTHER machine, look at the message with the 📎 next to it —
     it tells you exactly what to type, e.g.:
       /fetch 3
     That's it — NO path needed. It saves the file automatically into
     this kit's own data\\downloads\\ folder and prints:
       fetched -> C:\\...\\data\\downloads\\yourfile.txt
       sha256 ...  verified: true
       FILE VERIFIED END-TO-END ✅
     That checkmark line means the file arrived byte-for-byte intact —
     that's the whole point of this test. (If you'd rather choose where
     it saves, you can still type /fetch 3 "C:\\some\\path\\file.txt".)
 13. Try it in the other direction too, from the other machine.

If something restarts or disconnects
----------------------------------------
 - Closing the window (or the computer restarting) is SAFE. Just run
   run_mesh.cmd again — your room, your messages, and your identity are
   all still there. The kit also tries to reconnect to the other machine
   automatically; give it a few seconds.
 - If you type /join again with the SAME code you already used, you will
   NOT get an error — the kit recognizes you already joined and just
   reopens the room. There's nothing wrong; nothing to fix.
 - Not sure what's going on? Type /status and read what it says (see the
   PHONE SCRIPT section above).

Wrapping up
------------
 14. /export saves this room's full history to a file in the data\\
     folder — that is your own permanent, offline copy of everything said.
 15. Type /exit on each machine to close the kit. Your device identity,
     the room, and its messages are all saved in the data\\ folder and
     will be there again next time you run run_mesh.cmd.

Command reference
-------------------
  <text>                     post a message (no expectation tag)
  /expect <tag> <text>       post with a tag: whenever | today | urgent
  /rooms                     list rooms this device knows about
  /open <roomKey-or-prefix>  reopen a room and show recent messages
  /claim [name]              claim this room (defaults to you)
  /release                   release your claim
  /attach <path> [note]      attach a real file
  /fetch <seq> [savePath]    download + verify an attachment (no savePath
                             -> saved automatically to data\\downloads\\)
  /invite                    mint an invite code for the open room
  /join <invite-code>        redeem an invite code (safe to retype — see
                             "If something restarts or disconnects" above)
  /addwriter <pairing-code>  (founder) admit a joiner
  /connect <ip:port>         connect directly over your local network
  /listen <port>             (re)start listening for a direct connection
  /status                    show what this device is doing right now —
                             the command to read aloud on a support call
  /export [path]             save this room's transcript to a file
  /help                      show this list inside the kit
  /exit                      quit
`

// ASCII-only inside every .cmd template in this file, deliberately: a real
// field bisection (Mission A2, Band 2) found that a single em-dash (U+2014)
// byte sequence sitting inside a parenthesized `if (...)` block corrupts
// cmd.exe's batch parser under a non-UTF-8 active codepage — every line
// BEFORE the offending one starts throwing "'xyz' is not recognized" once
// cmd pre-scans the file for a GOTO label. A Bahrain field machine is not
// guaranteed to be running codepage 65001; plain hyphens and quotes only.
// (README_*.txt files are plain text, never parsed by cmd.exe — safe.)

// Mission A2, Band 2, item 3: setup_firewall.cmd — run-once, self-elevating,
// idempotent (delete-then-add so a re-run never duplicates a rule), targets
// the KIT'S OWN bundled node.exe by full path (never a blanket "allow node"
// rule — I7-adjacent: no broader hole than this one program needs). The
// ceremony card (README_CORRIDOR.txt) orders this BEFORE first run so the
// receptionist never sees a Windows Firewall popup mid-ceremony.
//
// `--print-only`: echoes the exact netsh commands with NO elevation attempt
// and NO actual firewall mutation — this is what kit2-spike.mjs drives (a
// CI/gate process must never trigger a UAC prompt).
const SETUP_FIREWALL_CMD = `@echo off
setlocal
cd /d "%~dp0"

set "NODE_EXE=%~dp0node.exe"
set "RULE_IN=AsymmFlow Mesh Kit (in)"
set "RULE_OUT=AsymmFlow Mesh Kit (out)"

if /I "%~1"=="--print-only" goto printOnly

net session >nul 2>nul
if not "%errorlevel%"=="0" (
  echo This step needs administrator rights - one popup, then it finishes on its own.
  powershell -NoProfile -Command "Start-Process -FilePath '%~f0' -Verb RunAs" >nul 2>nul
  exit /b 0
)

if not exist "%NODE_EXE%" (
  echo.
  echo node.exe was not found next to this file - this kit was not built with
  echo --bundle-node, or the file was moved out of the kit folder. Nothing to do.
  echo.
  pause
  exit /b 1
)

rem idempotent: delete any existing rule with this name first, then add fresh
netsh advfirewall firewall delete rule name="%RULE_IN%" >nul 2>nul
netsh advfirewall firewall delete rule name="%RULE_OUT%" >nul 2>nul
netsh advfirewall firewall add rule name="%RULE_IN%" dir=in action=allow program="%NODE_EXE%" enable=yes profile=any
netsh advfirewall firewall add rule name="%RULE_OUT%" dir=out action=allow program="%NODE_EXE%" enable=yes profile=any

echo.
echo FIREWALL READY
echo.
pause
exit /b 0

:printOnly
echo netsh advfirewall firewall delete rule name="%RULE_IN%"
echo netsh advfirewall firewall delete rule name="%RULE_OUT%"
echo netsh advfirewall firewall add rule name="%RULE_IN%" dir=in action=allow program="%NODE_EXE%" enable=yes profile=any
echo netsh advfirewall firewall add rule name="%RULE_OUT%" dir=out action=allow program="%NODE_EXE%" enable=yes profile=any
echo FIREWALL READY
exit /b 0
`

// Mission A2, Band 2, item 3: the remote ceremony card. Rewritten for
// phone/WhatsApp delivery (I3) — synthetic names only, from SYNTHETIC_
// IDENTITY.md (I4): Jordan (founder, India) and Sam (receptionist, Bahrain).
//
// Mission A2.1, Band 6, item 6: restructured to LEAD with the guided path
// (owner ruling R6 — a receptionist is never handed a command line). The
// old step-by-step ceremony is kept FULLY INTACT below, moved into an
// appendix — it is still the ground truth for exactly what to type once the
// messenger window is open (menu option [2] just gets you there), and it is
// still what a support call falls back to if the guide itself can't run.
const README_CORRIDOR_TEXT = `ASYMMFLOW MESH — THE CORRIDOR (remote field kit)
====================================================

This is the SAME kit as the kitchen-table test, set up for two machines
that are NOT in the same room — one in India, one in the Bahrain office.
Everything is delivered by phone call or WhatsApp: no email, no install,
no IT ticket.

*******************************************************************
*  READ THIS BOX FIRST                                             *
*  - This kit is ONE self-contained folder. It touches nothing else *
*    on this computer.                                              *
*  - SYNTHETIC DATA ONLY — made-up names, a throwaway test file.    *
*    Never type in a real customer or real business detail.        *
*  - This copy already includes its OWN copy of Node.js (node.exe   *
*    sitting right next to this file) — nothing to install.        *
*******************************************************************

START HERE
-------------
Double-click START_HERE.cmd and follow the questions on screen. It is the
ONLY thing you ever run by hand — everything else is a numbered menu:

  [1] Check the connection   - proves the two computers can reach each
                                other, and reads out one word at the end:
                                CORRIDOR GREEN, AMBER, or RED. Read that
                                word to whoever's on the call.
  [2] Open the messenger     - opens the same black window described in the
                                appendix below; use it to found the room,
                                invite the other machine, and talk.
  [3] Make this machine the always-on anchor  - Bahrain office machine
                                only; keeps it connected even when nobody
                                is using it.
  [4] Show status            - the "what do I read to support" option.
  [5] Close

On the Bahrain machine, the FIRST time you pick [1] or [2], it may ask for
one Windows permission (a single popup) — click Yes when it appears. That
only happens once, ever, on that machine.

Everything from here down is the SAME ceremony, written out in full — this
is what support reads you through if the guide can't run for some reason,
or if you'd rather type the steps yourself.

APPENDIX — if support asks you to do a step by hand
========================================================

Who's who in this card
-------------------------
  Jordan  = the founder, on a laptop in India (the "listen" side).
  Sam     = the receptionist, on the office PC in Bahrain (the "dial" side).
Use whatever two synthetic names you're actually given on the call —
these are just the placeholders in the steps below.

Reading codes over the phone or WhatsApp
--------------------------------------------
Invite codes and pairing codes are long strings of letters and numbers.
  - On a PHONE CALL: read them in GROUPS OF FOUR characters, with a pause
    between groups — e.g. "asymm dash room two dot... a1b2, c3d4, e5f6..."
    Whoever is typing reads them back once, before pressing Enter.
  - On WHATSAPP: it is much easier and less error-prone to just COPY and
    PASTE the whole code as a message. Prefer this over reading aloud
    whenever both people have WhatsApp open.

Step 0 — BEFORE the first run, on the Bahrain machine only
---------------------------------------------------------------
 0a. Double-click setup_firewall.cmd.
 0b. Windows will ask "Do you want to allow this app to make changes?" —
     click Yes. (This is the ONE popup in this whole test. It only lets
     THIS kit's own node.exe talk on the network — nothing else changes.)
 0c. A window prints FIREWALL READY, then closes. That's it — done once,
     never needed again on this machine.
 (Skip this step on Jordan's machine unless told otherwise on the call.)

Step 1 — start both machines
--------------------------------
 1. On EACH machine: double-click run_mesh.cmd. A black window opens.
 2. First run only: it asks for a synthetic actor name. Type one word
    (e.g. jordan or sam), press Enter. Saved from then on.
 3. If you ever get lost, TYPE: /status  and read every line out loud —
    that's the ONE command support needs to see your machine's state:
    your network transport, how many peers are connected, and when a
    peer was last seen.

Step 2 — Jordan founds the room (India)
-------------------------------------------
 4. TYPE:  /create corridor test
    Prints a LAN fallback line (ignore it for a remote test — hyperswarm
    is what carries this corridor) plus the room is now open.
 5. TYPE:  /invite
    Prints a code starting with "asymm-room2." — send this to Sam over
    WhatsApp (copy-paste) or read it in groups of four on the call.

Step 3 — Sam joins (Bahrain)
--------------------------------
 6. TYPE:  /join <paste Jordan's invite code here>
    Prints Sam's PAIRING CODE — send this back to Jordan the same way.

Step 4 — Jordan admits Sam
-------------------------------
 7. TYPE:  /addwriter <paste Sam's pairing code here>
    A few seconds later, Sam's window prints "joined ... — you can post
    now" on its own — nothing else to type for this step.

If hyperswarm can't find a path (rare, but the reason this card exists)
-----------------------------------------------------------------------
 - Run kit/probe.mjs (or its shortcuts run_probe.cmd / run_probe_dial.cmd,
   if this kit includes them) FIRST, before step 2, to check the corridor:
   it prints one word — CORRIDOR GREEN, CORRIDOR AMBER, or CORRIDOR RED —
   read that word to whoever's on the call.
 - If the Bahrain office has a port-forward + DuckDNS name set up (ask
   SPOC), /connect <that-name>:PORT works from Jordan's side as a
   DHT-free fallback.

Now talk and test file transfer
-----------------------------------
 8. Type a message, press Enter, on either machine — it should appear on
    the other side within a few seconds.
 9. File test — on the machine with a test file:
      /attach C:\\path\\to\\your\\test\\file.txt  a note about this file
    It prints a sha256 fingerprint and a #N seq number.
10. On the OTHER machine, look for the message with 📎, then TYPE:
      /fetch <that #N>
    It saves the file automatically and prints:
      FILE VERIFIED END-TO-END ✅
    Read that checkmark line out loud — it means the bytes crossed the
    corridor (India <-> Bahrain) perfectly intact.

/status — the universal support tool
-----------------------------------------
Whenever anything looks stuck, wrong, or just unfamiliar, TYPE: /status
and read every line out loud, top to bottom. It tells the person on the
phone: your actor name, your network transport (hyperswarm or tcp), how
many swarm connections you have right now, when a peer was last seen,
and which node.exe you're running (the kit's own bundled copy, or one
already on this computer). There is nothing else to check by hand.

Wrapping up
--------------
11. /export saves the full transcript to a file in data\\ — your own
    offline copy.
12. /exit on each machine closes the kit. Nothing is lost — your device
    identity, the room, and its messages are all saved in data\\ and
    come back automatically next time you run run_mesh.cmd.
`

// ── 3. assemble both machine folders ───────────────────────────────────────
//
// U1.6 preserve-guard (MSG-D25): the owner's real Machine-A has LIVE
// field-test data in dist/Machine-A/data/ RIGHT NOW. A rebuild must refresh
// code (host/, kit/, dist/reducer.wasm, node_modules/, run_mesh.cmd,
// README) WITHOUT ever touching an existing data/ that has content — the
// old blanket `rmSync(target, {recursive:true})` deleted the WHOLE machine
// folder, data included. "Has content" is judged the honest way: anything
// beyond the two placeholder `.keep` files this same script writes into a
// brand-new machine (a device-seed.hex, an actor.txt, a rooms.json, a
// corestore, ...) counts as real data worth keeping.
function hasRealData(dataDir) {
  if (!existsSync(dataDir)) return false
  const stack = [dataDir]
  while (stack.length) {
    const dir = stack.pop()
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      if (entry.name === '.keep') continue
      const full = join(dir, entry.name)
      if (entry.isDirectory()) stack.push(full)
      else return true
    }
  }
  return false
}

// U1.6 lock-resilience (found the hard way THIS wave: the owner's real
// Machine-A kit was RUNNING — a live `node kit/kit-host.mjs` process — while
// this script ran, and Windows holds an exclusive lock on any native addon
// (.node file, e.g. fs-native-extensions's prebuild) that process has
// loaded. The FIRST version of this guard called a bare `rmSync` on the
// whole node_modules tree and crashed with an uncaught EPERM partway
// through — deleting host/, kit/, dist/, run_mesh.cmd first (fine, those
// aren't locked) and then dying on node_modules, leaving the machine folder
// BROKEN (no host/, no kit/, no run_mesh.cmd) while data/ sat untouched.
// "A rebuild must be safe" means safe even with the kit mid-flight: every
// mutation below is wrapped so ONE locked file is skipped-and-reported,
// never a crash that half-finishes the whole folder.
function trySync(fn, label, skipped) {
  try {
    fn()
    return true
  } catch (err) {
    skipped.push(`${label}: ${err.code || err.message}`)
    return false
  }
}

function assembleMachine(name) {
  const target = join(distOut, name)
  const dataDir = join(target, 'data')
  const preserveData = hasRealData(dataDir)
  const skipped = []
  if (preserveData) {
    console.log(`  preserving existing data/ for ${name} (contains real content — not touched)`)
    // Remove every top-level entry EXCEPT data/ (the owner's live kit).
    // Best-effort per entry: a locked file (e.g. this machine's kit is
    // currently RUNNING) skips that one entry rather than aborting the
    // whole rebuild — see the lock-resilience note above.
    if (existsSync(target)) {
      for (const entry of readdirSync(target, { withFileTypes: true })) {
        if (entry.name === 'data') continue
        trySync(() => rmSync(join(target, entry.name), { recursive: true, force: true }), `remove ${entry.name}`, skipped)
      }
    }
  } else if (existsSync(target)) {
    // Same lock-resilience as the preserveData branch above, extended to the
    // fresh-build case (found live, Mission A2 Band 2: a leftover dist/
    // folder from an interrupted prior build can have a directory-level
    // handle held by an unrelated watcher/AV process even though every FILE
    // inside is gone — a bare `rmSync(target, {recursive:true})` throws
    // EBUSY on the top-level rmdir and crashes the whole build). Try the
    // fast path first; on failure, best-effort per top-level entry (same
    // as preserveData) so one stuck subtree never blocks the other, then
    // leave the (now near-empty) target dir in place — mkdirSync below is
    // idempotent and does not care whether target already existed.
    try {
      rmSync(target, { recursive: true, force: true })
    } catch (err) {
      skipped.push(`remove ${name}/ (top-level): ${err.code || err.message}`)
      for (const entry of readdirSync(target, { withFileTypes: true })) {
        trySync(() => rmSync(join(target, entry.name), { recursive: true, force: true }), `remove ${entry.name}`, skipped)
      }
    }
  }
  mkdirSync(target, { recursive: true })

  // portable.flag — presence alone is the marker; DP1 plane convention.
  writeFileSync(join(target, 'portable.flag'), `asymmflow-mesh kitchen-table kit\nbuilt ${new Date().toISOString()}\n`)

  // host/ + kit/ — only the reachable files, preserving relative layout.
  for (const file of localFiles) {
    const rel = file.startsWith(hostDir) ? join('host', file.slice(hostDir.length + 1))
      : file.startsWith(kitDir) ? join('kit', file.slice(kitDir.length + 1))
      : (() => { throw new Error(`unexpected source file outside host/ or kit/: ${file}`) })()
    const dest = join(target, rel)
    trySync(() => { mkdirSync(dirname(dest), { recursive: true }); cpSync(file, dest) }, `copy ${rel}`, skipped)
  }

  // dist/reducer.wasm
  mkdirSync(join(target, 'dist'), { recursive: true })
  trySync(() => cpSync(wasmPath, join(target, 'dist', 'reducer.wasm')), 'copy dist/reducer.wasm', skipped)

  // node_modules — the real closure, whole package directories (native
  // prebuilds live inside these, e.g. */prebuilds/**, and are NOT
  // individually filtered — a package is copied intact or not at all).
  // Per-package try/catch is the load-bearing fix: a package whose native
  // addon is loaded into a RUNNING kit process on this machine fails to
  // overwrite, skips (the old copy — unchanged content, since dependencies
  // didn't move this wave — stays exactly as valid as it was), and every
  // OTHER package still refreshes normally.
  mkdirSync(join(target, 'node_modules'), { recursive: true })
  for (const pkgName of packageClosure) {
    const src = join(rootNodeModules, ...pkgName.split('/'))
    const dest = join(target, 'node_modules', ...pkgName.split('/'))
    trySync(() => { mkdirSync(dirname(dest), { recursive: true }); cpSync(src, dest, { recursive: true, force: true }) }, `copy node_modules/${pkgName}`, skipped)
  }

  // data/corestore + data/keys — empty, SIBLING dirs (kit-host.mjs also
  // recreates these on first run; present here so the zip's shape matches
  // the doctrine even before the kit is ever launched).
  mkdirSync(join(target, 'data', 'corestore'), { recursive: true })
  mkdirSync(join(target, 'data', 'keys'), { recursive: true })
  writeFileSync(join(target, 'data', 'corestore', '.keep'), '')
  writeFileSync(join(target, 'data', 'keys', '.keep'), '')

  // minimal package.json — not required for module resolution (.mjs files
  // are ESM regardless of a "type" field), included for clarity only.
  writeFileSync(join(target, 'package.json'), JSON.stringify({
    name: `asymmflow-mesh-kit-${name.toLowerCase()}`, private: true, type: 'module',
  }, null, 2))

  writeFileSync(join(target, 'run_mesh.cmd'), toCrlf(RUN_MESH_CMD))
  writeFileSync(join(target, 'START_HERE.cmd'), toCrlf(START_HERE_CMD))
  writeFileSync(join(target, 'README_KITCHEN_TABLE.txt'), README_TEXT)

  // ── Mission A2, Band 2 ──────────────────────────────────────────────────
  // item 3: the remote ceremony card + firewall pre-authorization script —
  // unconditional, every built kit gets these regardless of --bundle-node
  // (setup_firewall.cmd itself checks for node.exe at runtime and says so
  // plainly if it's missing).
  writeFileSync(join(target, 'README_CORRIDOR.txt'), README_CORRIDOR_TEXT)
  writeFileSync(join(target, 'setup_firewall.cmd'), toCrlf(SETUP_FIREWALL_CMD))

  // item 4: probe (Band 1) + item's anchor files (Band 3) — travel inside
  // the same zip IF the parallel coder has delivered them by build time.
  // Graceful: warn-and-skip, never fail the build. `spec` fixes these exact
  // filenames so C1/C3's output is consumable without coordination beyond
  // the filenames themselves (MISSION_A2_CORRIDOR_SPEC.md §6).
  const corridorPresence = {}
  for (const f of CORRIDOR_OPTIONAL_FILES) {
    const src = join(kitDir, f)
    // PROBE_CLUSTER and GUIDE_CLUSTER travel together under kit/ (beside
    // probe.mjs / guide.mjs); ANCHOR_CLUSTER travels together at the kit
    // root (beside node.exe, data/) — see the placement note above
    // CORRIDOR_OPTIONAL_FILES.
    const dest = (PROBE_CLUSTER_FILES.includes(f) || GUIDE_CLUSTER_FILES.includes(f)) ? join(target, 'kit', f) : join(target, f)
    if (existsSync(src)) {
      trySync(() => {
        mkdirSync(dirname(dest), { recursive: true })
        if (f === 'anchor.mjs') {
          // The one transformed copy (see anchorSourceForRoot) — every
          // other file in both clusters copies verbatim (modulo the .cmd
          // CRLF normalization below).
          writeFileSync(dest, anchorSourceForRoot(readFileSync(src, 'utf8')))
        } else if (f.endsWith('.cmd')) {
          writeFileSync(dest, toCrlf(readFileSync(src, 'utf8')))
        } else {
          cpSync(src, dest)
        }
      }, `copy ${f}`, skipped)
      corridorPresence[f] = true
    } else {
      console.log(`  ⚠ ${name}: ${f} not present at build time (parallel band not yet delivered) — skipped, not an error`)
      corridorPresence[f] = false
    }
  }

  // item 1: bundle THIS running node.exe (pinned v22 LTS) into the kit
  // root — every generated .cmd (nodeLaunchPreamble above) prefers it over
  // PATH. Only on Windows / when process.execPath actually looks like a
  // node.exe; a non-Windows dev box building the kit just skips this with
  // a clear notice rather than copying a useless binary.
  let bundledNodeInfo = null
  if (BUNDLE_NODE) {
    if (process.platform !== 'win32') {
      console.log(`  ⚠ ${name}: --bundle-node requested on ${process.platform}, not win32 — the kit ships for Windows machines; skipping the copy (build on Windows to produce a real bundle)`)
    } else {
      const destNode = join(target, 'node.exe')
      const ok = trySync(() => cpSync(process.execPath, destNode), 'copy node.exe', skipped)
      if (ok) {
        bundledNodeInfo = { path: destNode, version: process.version }
        console.log(`  ${name}: bundled node.exe (${process.version}, ${(statSync(destNode).size / 1e6).toFixed(1)} MB)`)
      }
    }
  }

  if (skipped.length) {
    console.log(`  ⚠ ${name}: ${skipped.length} item(s) skipped (likely a running kit process holding a file open) — rebuild again after closing it for a full refresh:`)
    for (const s of skipped) console.log(`      - ${s}`)
  }

  return { target, bundledNodeInfo, corridorPresence }
}

// Mission A2, Band 2, item 1: verify udx-native's native prebuild resolves
// from UNDER the bundled layout — a copied node.exe, run from the kit's
// own root, resolving node_modules/udx-native the normal Node way (CWD +
// module-parent walk), exactly what run_mesh.cmd's `cd /d "%~dp0"` sets up.
// Spawns the bundled exe itself (not this build process's own node) so a
// prebuild/ABI mismatch would actually surface here, not be silently
// papered over by "well MY node can load it".
function verifyBundledUdx(target, bundledNodeInfo) {
  if (!bundledNodeInfo) return { skipped: true }
  const probeScript = join(target, '.udx-smoke.cjs')
  writeFileSync(probeScript, "require('udx-native'); console.log('UDX_BUNDLE_OK')\n")
  try {
    const out = execFileSync(bundledNodeInfo.path, [probeScript], { cwd: target, encoding: 'utf8' })
    return { skipped: false, ok: out.includes('UDX_BUNDLE_OK'), output: out.trim() }
  } catch (err) {
    return { skipped: false, ok: false, output: (err.stdout || '') + (err.stderr || err.message) }
  } finally {
    try { rmSync(probeScript, { force: true }) } catch { /* best-effort cleanup */ }
  }
}

const machineA = assembleMachine('Machine-A')
const machineB = assembleMachine('Machine-B')

console.log('\nbuilt:')
for (const { target } of [machineA, machineB]) {
  console.log(`  ${target}`)
}

if (BUNDLE_NODE) {
  console.log('\nudx-native bundled-layout smoke check:')
  for (const [name, { target, bundledNodeInfo }] of [['Machine-A', machineA], ['Machine-B', machineB]]) {
    const res = verifyBundledUdx(target, bundledNodeInfo)
    if (res.skipped) console.log(`  ${name}: skipped (no bundled node.exe — see notice above)`)
    else console.log(`  ${name}: ${res.ok ? 'OK ✅' : 'FAILED ❌'} — ${res.output}`)
  }
}
