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

import { existsSync, mkdirSync, readFileSync, writeFileSync, cpSync, rmSync, readdirSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'

const __dirname = dirname(fileURLToPath(import.meta.url))
const meshRoot = join(__dirname, '..')
const kitDir = __dirname
const hostDir = join(meshRoot, 'host')
const distOut = join(kitDir, 'dist')

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
const RUN_MESH_CMD = `@echo off
setlocal
cd /d "%~dp0"

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

node kit\\kit-host.mjs --data data

echo.
echo (the kit has stopped — press any key to close this window)
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
  } else {
    rmSync(target, { recursive: true, force: true })
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

  writeFileSync(join(target, 'run_mesh.cmd'), RUN_MESH_CMD)
  writeFileSync(join(target, 'README_KITCHEN_TABLE.txt'), README_TEXT)

  if (skipped.length) {
    console.log(`  ⚠ ${name}: ${skipped.length} item(s) skipped (likely a running kit process holding a file open) — rebuild again after closing it for a full refresh:`)
    for (const s of skipped) console.log(`      - ${s}`)
  }

  return target
}

const machineA = assembleMachine('Machine-A')
const machineB = assembleMachine('Machine-B')

console.log('\nbuilt:')
for (const dir of [machineA, machineB]) {
  console.log(`  ${dir}`)
}
