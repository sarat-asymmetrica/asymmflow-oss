# Phase 0 Notes — D: Bare Spike Re-Verification

**Coder:** P0-D · **Branch:** `feat/fable-bare-runtime` · **Date re-run:** 2026-07-20
**Doctrine:** zero assumptions; "verified 2026-07-19" is not verification. Every claim below
is either a transcript I produced today or a doc URL fetched today. Sole file written by this
coder: this one.

---

## 1. Versions table

| Package | Report (2026-07-19) claimed | Installed today (2026-07-20) | Match? |
|---|---|---|---|
| `bare` (CLI/runtime) | `1.30.3` | `1.30.3` (`npx bare --version` → `v1.30.3`) | yes |
| `bare-runtime-win32-x64` prebuild | resolved, no build step | same — `npx bare` auto-installed `bare@1.30.3` as an npx-cache package the first time it ran in this session; the local devDependency resolved to the same `1.30.3` | yes |
| `bare-fs` | not versioned in report | `4.7.4` | (matches `mesh/package.json` `^4.7.4`) |
| `bare-process` | not versioned in report | `4.5.1` | (matches `^4.5.1`) |
| `bare-stream` | not versioned in report | `2.13.3` | (matches `^2.13.3`) |
| `bare-crypto` | not versioned in report | `1.15.3` | (matches `^1.15.3`) |
| `bare-events` | not versioned in report | `2.9.1` | (matches `^2.9.1`) |
| `corestore` | not versioned in report | `7.11.1` |  |
| `autobase` | not versioned in report | `7.28.1` |  |
| `hyperswarm` | not versioned in report | `4.17.0` |  |
| `hyperdht` | not versioned in report | `6.33.0` |  |
| `hyperbee` | not versioned in report | `2.27.3` |  |
| `hyperblobs` | not versioned in report | `2.12.1` |  |
| `hypercore` | not versioned in report | `11.34.0` |  |
| `hypercore-id-encoding` | not versioned in report | `1.3.0` |  |
| `blind-peer` | not versioned in report | `3.12.2` |  |
| `blind-peering` | not versioned in report | `2.4.1` |  |
| `protomux-wakeup` | not versioned in report | `2.9.0` |  |
| `holesail` | not exercised by this spike per report | `2.4.1` (present as a `mesh/package.json` dep, separately from the bare-spike scripts) |  |

Note: `mesh/node_modules` was **not present** at the start of this run — I ran `npm install`
in `mesh/` first (181 packages, 0 vulnerabilities, ~7s). This itself is a delta worth flagging:
the 2026-07-19 report implies a working tree with `node_modules` already resolved; a fresh
clone/checkout of this branch does not have it until `npm install` runs. Not a defect, just a
prerequisite the report didn't state explicitly.

Go: `go1.25.3 windows/amd64` (used to rebuild `reducer.wasm`, see §3 delta below).

---

## 2. Re-run transcripts verbatim

All run from `mesh/` via `npx bare host/bare-spike/<script>.mjs`, same as the report's own
invocation. Environment: Windows 11 Pro 10.0.26200, same machine class as the report.

### 2.1 `hello.mjs`

```
$ npx bare host/bare-spike/hello.mjs
bare hello-world [
  'C:\\Projects\\asymmflow\\asymmflow-oss\\mesh\\node_modules\\bare-runtime-win32-x64\\bin\\bare.exe',
  'C:\\Projects\\asymmflow\\asymmflow-oss\\mesh\\host\\bare-spike\\hello.mjs'
]
```
PASS. Matches report (path differs only because the repo folder is `asymmflow-oss` now, not
`asymmflow-mesh` — a rename between 2026-07-19 and today, not a functional change).

### 2.2 `stdio-check.mjs`

```
$ printf '{"id":1,"method":"hello","params":{}}\n' | npx bare host/bare-spike/stdio-check.mjs
{"event":"ready"}
{"echoed":"{\"id\":1,\"method\":\"hello\",\"params\":{}}"}
```
PASS. Identical to report.

### 2.3 `require-check.mjs`

```
$ npx bare host/bare-spike/require-check.mjs
OK   corestore
OK   autobase
OK   hyperswarm
OK   hyperdht
OK   hyperbee
OK   hyperblobs
OK   hypercore
OK   hypercore-id-encoding
OK   blind-peer
OK   blind-peering
OK   protomux-wakeup
```
PASS, 11/11, identical to report. Re-ran 15 times back-to-back (stress test, see §3) — 15/15
consistent, no flake.

### 2.4 `wasi-check.mjs`

```
$ npx bare host/bare-spike/wasi-check.mjs
WebAssembly typeof: object
FAIL node:wasi :: MODULE_NOT_FOUND :: MODULE_NOT_FOUND: Cannot find module 'node:wasi' imported from 'file:///C:/Projects/asymmflow/asymmflow-oss/mesh/host/bare-spike/wasi-check.mjs'
FAIL wasi :: MODULE_NOT_FOUND :: MODULE_NOT_FOUND: Cannot find module 'wasi' imported from 'file:///C:/Projects/asymmflow/asymmflow-oss/mesh/host/bare-spike/wasi-check.mjs'
FAIL bare-wasi :: MODULE_NOT_FOUND :: MODULE_NOT_FOUND: Cannot find module 'bare-wasi' imported from 'file:///C:/Projects/asymmflow/asymmflow-oss/mesh/host/bare-spike/wasi-check.mjs'
```
Same conclusion as report (no WASI host under Bare), full error text is more verbose than the
report's truncated one-liners but semantically identical.

### 2.5 `wasm-compile-check.mjs`

**Prerequisite delta**: `mesh/dist/reducer.wasm` did **not exist** in the checked-out tree —
I had to run `npm run build` (→ `node scripts/build-reducer.mjs` → `GOOS=wasip1 GOARCH=wasm go
build`) to produce it before this script could run at all. See §3 for why this matters.

```
$ npm run build
> node scripts/build-reducer.mjs
built C:\Projects\asymmflow\asymmflow-oss\mesh\dist\reducer.wasm

$ npx bare host/bare-spike/wasm-compile-check.mjs
reducer.wasm bytes: 3963665
OK   WebAssembly.compile(reducer.wasm) succeeded
required import namespaces: [ 'wasi_snapshot_preview1' ]
import count: 18
```
PASS on this run — but this script is **flaky** (see §3, new finding). `3963665` bytes vs the
report's `3963498` — a 167-byte difference, expected: rebuilt today from the same source with
today's Go toolchain (`go1.25.3`), not a byte-identical artifact from 2026-07-19's build. Same
import namespace/count either way.

### 2.6 `wasi-imports-list.mjs`

```
$ npx bare host/bare-spike/wasi-imports-list.mjs
wasi_snapshot_preview1.sched_yield  (function)
wasi_snapshot_preview1.proc_exit  (function)
wasi_snapshot_preview1.args_get  (function)
wasi_snapshot_preview1.args_sizes_get  (function)
wasi_snapshot_preview1.clock_time_get  (function)
wasi_snapshot_preview1.environ_get  (function)
wasi_snapshot_preview1.environ_sizes_get  (function)
wasi_snapshot_preview1.fd_write  (function)
wasi_snapshot_preview1.random_get  (function)
wasi_snapshot_preview1.poll_oneoff  (function)
wasi_snapshot_preview1.fd_close  (function)
wasi_snapshot_preview1.fd_read  (function)
wasi_snapshot_preview1.fd_write  (function)
wasi_snapshot_preview1.random_get  (function)
wasi_snapshot_preview1.fd_fdstat_get  (function)
wasi_snapshot_preview1.fd_fdstat_set_flags  (function)
wasi_snapshot_preview1.fd_prestat_get  (function)
wasi_snapshot_preview1.fd_prestat_dir_name  (function)
```
Identical 18-entry/16-distinct-syscall list to the report — **when it produces output at all**
(see §3, same flake class as 2.5).

---

## 3. Deltas vs the 2026-07-19 report

1. **`mesh/dist/reducer.wasm` is not a committed artifact.** It is `.gitignore`d build output
   (`npm run build` regenerates it via `scripts/build-reducer.mjs`). The report's transcripts
   read as if the file was simply present. Anyone re-running this spike from a fresh clone
   needs `go` on PATH and must run `npm run build` first, or scripts 2.5/2.6 fail with
   `ENOENT`. Worth stating explicitly in the report next time, since it's a silent prerequisite.

2. **NEW FINDING — `wasm-compile-check.mjs` and `wasi-imports-list.mjs` are flaky under Bare
   on this machine.** Not mentioned in the 2026-07-19 report, which shows one clean run of
   each and draws its conclusions from that single pass. Re-running each script repeatedly,
   back-to-back, in the same working directory with no code changes:

   - `wasi-imports-list.mjs`: 15 runs → **4 runs (27%) produced zero output** (empty stdout
     AND empty stderr, exit code 0 — not a crash, just silently nothing printed).
   - `wasm-compile-check.mjs`: 8 runs → 2 runs (25%) printed only the first line
     (`reducer.wasm bytes: N`) and then silently stopped — `WebAssembly.compile` and
     everything after it produced no output, exit code 0 either way.
   - By contrast, `require-check.mjs` (also does `await import()` in a loop, also has
     top-level await) was run 15 times with **zero anomalies** — 11/11 `OK` lines every time.
   - `hello.mjs` and `stdio-check.mjs` have no `WebAssembly.compile` call and were not
     observed to flake in any run this session.

   The common factor across the two flaky scripts is `WebAssembly.compile()` immediately
   followed by synchronous `console.log()` calls in a `for` loop, with the process then
   reaching end-of-script — it looks like a stdout-flush race on process exit specific to the
   turn after a `WebAssembly.compile` await resolves, not a resolution/import issue. I did not
   have time this session to bisect further (e.g. does `queueMicrotask`-delaying the exit, or
   an explicit flush call, fix it) — flagging as a **stability risk for any future Bare-hosted
   reducer driver**: a production `bare-bridge.mjs` that calls `WebAssembly.compile` on every
   `apply()` (or even once at startup) needs to know its stdout writes are not guaranteed to
   land before the process is considered "done." This is new information the DP4 push should
   account for; it does not change the WASI-host blocker conclusion (§4 below), but it is a
   second, independent Bare rough edge worth tracking.

3. **No other functional deltas.** Every other transcript (§2.1–2.4) matches the report
   exactly in outcome. `require-check.mjs`'s 11/11 pass, the WASI-host absence, and the
   16-syscall/18-import inventory all reproduce cleanly and repeatably.

4. **Report's Environment section says Node v22.17.0 "used here only to run npm."** Confirmed
   still true — `npx bare` itself invokes the `bare.exe` prebuild directly; Node is only in
   the loop for `npm install`/`npm run build`.

---

## 4. Hostile-geography failure map (campaign doctrine D5)

Copied `mesh/host/bare-spike/*.mjs` to a directory outside the repo tree
(`…/scratchpad/hostile/`), preserving the `host/bare-spike/` relative depth so the scripts'
own relative-path logic (`../../dist/reducer.wasm`) stays meaningful. Two passes: (a) bare
copy, no `node_modules`, no `dist/`; (b) same copy plus a **self-contained** `npm install bare
bare-fs bare-process` and a copied `dist/reducer.wasm`, to test the "complete kit" scenario
distinct from the "nothing provisioned" scenario.

### Pass (a) — nothing provisioned

| Script | Result | Exit | Failure mode |
|---|---|---|---|
| `hello.mjs` | **PASS** | 0 | No imports beyond Bare globals — works anywhere. |
| `require-check.mjs` | **PASS (as designed)** | 0 | Every one of the 11 modules reports `FAIL … MODULE_NOT_FOUND` individually — the script's own try/catch-per-module pattern means "everything missing" is reported cleanly, not a crash. This is the *good* pattern. |
| `stdio-check.mjs` | **HARD CRASH** | 127 | Top-level `import process from 'bare-process'` is unguarded — `Uncaught ModuleError: MODULE_NOT_FOUND`, full stack trace to stderr, no graceful message. |
| `wasi-check.mjs` | **PASS (as designed)** | 0 | Same per-import try/catch pattern as require-check — three `FAIL` lines, no crash. |
| `wasm-compile-check.mjs` | **HARD CRASH** | 127 | Top-level `import fs from 'bare-fs'` unguarded — same `Uncaught ModuleError` shape as stdio-check. |
| `wasi-imports-list.mjs` | **HARD CRASH** | 127 | Same as above (`bare-fs`). |

**Pattern**: scripts that wrap each external import in its own `try { await import(x) }
catch` (require-check, wasi-check) degrade gracefully to a readable FAIL report even with zero
dependencies present. Scripts with an unguarded top-level `import x from 'y'` (stdio-check,
wasm-compile-check, wasi-imports-list) crash hard with a full stack trace. This is **exactly**
the FR-1a disease class from the corridor field report (`probe.mjs`'s top-level `import
'holesail'`) — reproduced here in miniature, in a different script, under a different runtime.
The fix pattern that FR-1a Band 5 already established (lazy `await import()` only inside the
branch that needs it, with a plain one-line skip message) generalizes directly: any future
Bare host script should use the require-check/wasi-check per-import try/catch shape, never a
bare top-level `import`.

### Pass (b) — self-contained kit (own `node_modules` + own `dist/reducer.wasm`)

| Script | Result | Notes |
|---|---|---|
| `hello.mjs` | PASS | Runtime resolved from `hostile/node_modules/bare-runtime-win32-x64`. |
| `stdio-check.mjs` | PASS | Identical ndjson echo as the in-repo run. |
| `wasm-compile-check.mjs` | PASS, but **same flake** as §3 finding 2 (2/8 runs in this location truncated after line 1, one run in a stress loop hung past a 2-minute timeout — see below) | Confirms the flake is a Bare runtime property, not an artifact of running inside this specific repo checkout. |

One run in the hostile-geography stress loop **did not return within 2 minutes** and the
command was killed by timeout (the other 7 in that batch completed normally, 2 of those 7
truncated per the flake pattern). I did not get a chance to isolate whether the hang is the
same root cause as the truncation (a stuck-forever version of the same race) or a distinct
issue — flagging as **not verified**, worth a dedicated follow-up.

### Positive finding: Bare's module resolution is naturally hermetic, but not absent

Testing pass (a) vs (b) clarifies something the FR-1b writeup (about Node's upward
`node_modules` resolution silently escaping the repo tree) doesn't establish for Bare: Bare
**does** walk upward through ancestor `node_modules` directories exactly like Node does
*within a self-contained copied tree* (pass (b)'s `hostile/node_modules` was found correctly
from a script three directories below it, `hostile/host/bare-spike/*.mjs`). What it does
**not** do is escape *past* the root of whatever tree you actually copied — in pass (a), with
no `node_modules` anywhere under `hostile/`, nothing leaked in from the real repo's
`mesh/node_modules` even though that directory exists on the same machine. This is a
meaningfully different risk profile than the Node/FR-1b case: a Bare-hosted kit that is
copied whole (including its own `node_modules`) is geography-hermetic by construction, in a
way the current Node-based kit had to be made hermetic by policy (Band 5's relocated
`kit2-spike.mjs` checks). Worth remembering as a real advantage if DP4 ever ships.

---

## 5. Holesail findings

Studied `docs.holesail.io` and `github.com/holesail/holesail` (fetched today, 2026-07-20).

- **Current state**: actively maintained P2P TCP/UDP tunnel/reverse-proxy, AGPL-3.0,
  `supersuryaansh`/Holepunch-ecosystem project. README: "a truly peer-to-peer network
  tunneling and reverse proxy software that supports both TCP and UDP protocols," no port
  forwarding or static IP required.
- **Runs under Bare/Pear today, not just Node.** The docs site states it "Works on Linux,
  macOS, Windows, iOS, and Android using Bare modules and the Pear runtime." Confirmed
  independently in `mesh/node_modules/holesail/package.json`: the package ships explicit
  `"imports"` conditional-export maps for `util`/`fs`/`process`/`crypto` that swap to
  `bare-utils`/`bare-fs`/`bare-process`/`bare-crypto` under the `"bare"` condition and fall
  back to Node's builtins under `"default"`. This is the opposite situation from the
  reducer's WASI gap — holesail (and its `holesail-client`/`holesail-server`/`hyper-cmd`
  dependency chain) is **already Bare-native by design**, not something that merely happens
  to load. `holesail`'s own `package.json` `scripts.build` block also references
  `bare-make` (`npx bare-make generate/build/install`), confirming the project's own release
  process targets Bare/Pear directly.
- **Native addons**: yes. `mesh/node_modules/udx-native` (the UDP transport underneath
  `hyperdht`/`hyperswarm`/holesail) ships prebuilt `.node` binaries for
  win32-x64/win32-arm64/darwin-x64/darwin-arm64/linux-x64/linux-arm64/android-\* — a real
  native addon, not pure JS. GitHub's own language breakdown for `holesail/holesail` lists
  "JavaScript 89.3%, C 6.1%, CMake 4.6%," consistent with a native build step
  (`bare-make`/CMake) producing those prebuilds. `holesail-client`/`holesail-server` did not
  show any `.node` files directly under their own package directories in this tree (the
  native surface lives in `udx-native`, one layer down) — not exhaustively audited beyond
  that one layer, flagged as not fully verified.
- **Can it be consumed as a library under Bare?** Not directly tested this session (no
  network corridor available in this sandbox to prove a live tunnel) — but structurally yes,
  based on the package.json evidence above; this is an inference from static inspection, not
  a runtime proof. **Not verified: an actual `import('holesail')` + tunnel round-trip under
  `npx bare` was not attempted.**
- **UX prior art — the ceremony vocabulary**:
  - **Connection-string format**: `hs://` URL scheme. Secure servers use an `hs://s000…`
    prefix, insecure servers `hs://0000…`; the client-side parser auto-detects and accepts
    either, and a full `hs://` URL can be passed straight into the client constructor or CLI.
  - **CLI shape**: `holesail --live <port>` (server/share side) and `holesail
    <connection-string>` (client/connect side) — a single positional argument each side, no
    flags required for the basic case.
  - **QR codes**: the docs site's own framing of the ceremony is *"Run a single command, scan
    a QR code, and connect"* — QR is presented as a first-class alternative to typing/pasting
    the connection string, explicitly for the case where typing a long string is impractical
    (phone-to-desktop pairing). The GitHub README (as fetched) did not itself surface QR
    code details — this appears to live in the docs site / a client UI layer, not the base
    CLI package. **Not verified**: which exact holesail package/flag renders the QR (not
    found in the base `holesail` npm package's own README this session).
  - **Pairing UX framing**: a user testimonial quoted on the docs site: *"Generate key, paste
    key, done."* — three-step mental model, no accounts/usernames/passwords. This is the
    same shape our own `guide.mjs`/`probe.mjs` ceremony already converged on independently
    (§6 below) — worth citing as external validation of the design, not a new requirement.

---

## 6. Guided Path UX law (implementable spec, from `guide.mjs` + `MISSION_A2_CORRIDOR_SPEC.md` §Band 6)

This is binding on any successor that touches the guided ceremony. Quoting the actual
behavior/strings from `mesh/kit/guide.mjs` (read today), not paraphrasing.

**Entry point contract**: `START_HERE.cmd` at kit root launches `kit/guide.mjs` under the
kit's bundled `node.exe`. Zero arguments, ever — every subsequent input is a menu number or a
pasted code answered through prompts.

**Import discipline (load-bearing, do not relax)**: `guide.mjs` imports **only `node:`
built-ins** (`readline`, `fs.existsSync`, `child_process.spawn`/`spawnSync`, `path`, `url`).
It never imports `probe.mjs`, `anchor.mjs`, or anything that itself imports an npm package
(hyperdht, hyperswarm, holesail, etc.) — those are reached exclusively by **spawning them as
separate child processes** (`runCmdInherited`, `runProbe`). The file's own header comment
states why: this is the exact FR-1a failure class (a top-level npm import silently working
inside the repo via upward `node_modules` resolution, then crashing on the receptionist's
machine) closed structurally by construction, not by testing.

**Menu** (`printMenu()`, verbatim):
```
====================================
  ASYMMFLOW MESH — GUIDE
====================================
[1] Check the connection
[2] Open the messenger
[3] Make this machine the always-on anchor
[4] Show status
[5] Close
```

**Layout independence**: `resolveKitPath(...parts)` checks two depths for any filename — the
kit root (one level up from `guide.mjs`) and `guide.mjs`'s own directory — because the built
kit's `PROBE_CLUSTER` (probe.mjs, alongside guide.mjs) and `ANCHOR_CLUSTER` (node.exe,
run_mesh.cmd, install_anchor.cmd, etc.) live at different relative depths by design. Any new
file this guide needs to shell out to must be resolvable through this same helper, not a
hardcoded relative path.

**Connection-check flow** (`checkConnection`), exact prompt text: *"Did the other person send
you a code? PASTE it here and press Enter.\nIf YOU are starting, just press Enter."* Empty
input → listen mode (`probe.mjs --listen`); non-empty → normalized and dialed
(`probe.mjs --dial <code>`). The verdict is pulled back out of the buffered probe output via
`/CORRIDOR (GREEN|AMBER|RED)/` and reprinted **large**, framed in `=` rule lines, followed by
the exact instruction: *"Read this word to the person on the call: \<VERDICT\>"*.

**Paste normalization** (`normalizeCode`): strips *all* whitespace classes — plain spaces,
` ` (non-breaking space), `​` (zero-width space) — because a code read aloud in
groups of four picks up ordinary spaces, while a WhatsApp/chat-bubble paste can carry a
trailing newline or an invisible Unicode space character; the function must never throw on
`null`/`undefined`/empty input.

**Grouping for reading aloud** (`groupInFours`): chunks any code string into 4-character
groups joined by a single space — the same convention `probe.mjs`'s own `z32Groups()` uses,
deliberately duplicated (not imported) here per the import-discipline rule above.

**Conversational firewall step** (`ensureFirewall`, item 3 of the band spec): offered **at
most once per session**, only if (a) a bundled `node.exe` is present (i.e. this is a real
built kit, not a dev-tree run) and (b) the firewall rule (`AsymmFlow Mesh Kit (in)` — a
cross-file literal contract with `build-kit.mjs`'s `SETUP_FIREWALL_CMD` template) is not
already registered. Exact framing:
```
Before we connect, this computer needs one quick permission.
Windows will ask "Do you want to allow this app to make changes?" — click Yes.
That is the ONE popup in this whole process; nothing else on this computer changes.
Press Enter to continue, or type skip and press Enter to skip this for now.
```
Declining (`skip`) or a non-zero exit from `setup_firewall.cmd` is a **warning, never a
blocker** — the guide continues either way. This is the single instance in the whole flow
where UAC elevation happens, and the copy exists specifically to pre-explain the popup so it
is never a surprise.

**Error fold-line convention** (`reportError`, spec item 5): every menu action's uncaught
error is caught at the menu-loop level and rendered as: one plain sentence (*"Something went
wrong — read this line to the person on the phone:"* + `err.message`), then a literal
`--- details for support ---` fold line, then the raw `err.stack`. The menu **always returns**
after any error — no action's failure is allowed to crash the whole guide.

**Stdin queuing discipline (load-bearing implementation detail, not just UX)**:
`createGuideIO` explicitly rejects sequential `rl.question()` calls in favor of an
unconditional `'line'` listener feeding a FIFO queue, because `rl.question()` only starts
listening at the moment it's called — if scripted/piped stdin (or a fast human paste) already
has more than one line buffered, the second line's event can fire before the code calls
`question()` again, silently dropping that answer and hanging the next prompt forever. The
comment in the file states this was reproduced 100% of the time in isolation before the fix.
Any successor reimplementing prompt/answer plumbing must preserve this queue-based shape, not
regress to sequential `question()` calls — this is exactly the shape a hermetic gate
(scripted stdin) and a real human paste-then-Enter both exercise.

**Verdict words**: exactly three, always uppercase, always exactly `CORRIDOR GREEN` /
`CORRIDOR AMBER` / `CORRIDOR RED` — sourced from `probe.mjs`'s own output and never
reworded/paraphrased by the guide layer.

**Reuse-never-reimplement (I1)**: the messenger, anchor install/uninstall, and status are
never reimplemented in `guide.mjs` — each menu option shells out to the existing
`run_mesh.cmd` / `install_anchor.cmd` / `uninstall_anchor.cmd` / `anchor_status.cmd` with
console control explicitly paused/resumed (`io.pause()`/`io.resume()`) around the messenger's
own REPL so the two readline consumers never fight over stdin.

---

## 7. Not verified

- Whether the `wasm-compile-check.mjs`/`wasi-imports-list.mjs` flake (§3 finding 2, §4 pass
  (b)) has a root cause beyond "stdout flush race after `WebAssembly.compile`" — not bisected
  this session (no `queueMicrotask`/explicit-flush experiment run).
- The one stress-loop run that hung past a 2-minute timeout in hostile geography (§4 pass
  (b)) — not confirmed same-or-different root cause from the truncation flake.
- A live `holesail` tunnel round-trip under `npx bare` (no network corridor available in this
  sandbox) — the "consumable as a library under Bare" conclusion is from static
  `package.json`/prebuild inspection only, not a runtime proof.
- Which exact holesail surface (base CLI vs. a client UI layer) renders the QR code mentioned
  on the docs site — not found in the base `holesail` npm package's own files this session.
- Whether `holesail-client`/`holesail-server` carry native addons of their own beyond the
  `udx-native` dependency (only one dependency layer was checked for `.node` files).
- ~~Full `npm view bare-wasi` registry re-check~~ — **done**: `npm view bare-wasi` → `404 Not
  Found` as of 2026-07-20, same result as the original report. Still no `bare-wasi` package
  on the registry.
