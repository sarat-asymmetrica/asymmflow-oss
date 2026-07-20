# Phase 3 Report — the Guided Path, ported to Bare

**Coder:** P1A-wasi-shim · **Date:** 2026-07-20 · **Branch:** `feat/fable-bare-runtime`

## 1. Verdict

**YES — the guided path runs end-to-end under Bare**, with one honestly
scoped exception. `mesh/kit/bare-guide.mjs` reproduces guide.mjs's exact
menu, prompts, paste-normalization, verdict-word presentation, and
error-fold convention, driven by a hand-rolled FIFO-queue stdin reader
(reusing `bare-bridge.mjs`'s `getRealStdio()`, not `readline`), and it
carries ONE genuinely working, real, Bare-native integration: opening the
messenger posts and lists messages through `bare-bridge.mjs`'s dispatch
core, in-process, all the way down to `reducer.wasm` — verified over a
real spawned pipe under both `node` and `bare.exe`, not just in-process.

The exception, stated plainly rather than approximated (per this phase's
own honesty law): **"Check the connection" (menu [1]) and "anchor"/"status"
(menu [3]/[4]) are honest stubs, not ported functionality.** No Bare-native
connectivity probe exists anywhere in this tree, and I chose not to build
one (out of this phase's file ownership) or to mutate real Windows
Scheduled-Task/firewall state from an automated test run without the
machine owner reviewing the exact invocation first (see §2 for the full
reasoning per item). The copy and menu shape for all four items are ported
verbatim regardless — the UX law's WORDS are law even where the ACTION
behind one isn't wired yet.

Old `mesh/kit/guide.mjs`/`guide-spike.mjs` (Node line) untouched, still
green — the rollback path stays warm, exactly as required.

**SEALED KIT: YES — a message actually posts from the sealed kit in
hostile geography, and `dist/reducer.wasm` is in the manifest.** Three
real defects surfaced and were fixed across this phase and one follow-up
round (§1a-§1c); all three are now closed and independently re-verified.
A `bare.exe app.bundle` built from `kit/bare-guide-entry.mjs`, copied whole
to a from-scratch temp directory, driven over a real spawned pipe: renders
the menu, answers the firewall offer, opens the messenger, POSTS a real
message (`posted, seq 2` — a real seq number, not a mock), lists it back
via `/rooms`, and prints the exact "Goodbye" line — then the process
actually exits. 17/17 spike checks green, both the manifest content and
the message-posting content are asserted, not merely reported.

### 1a. Root-caused defect, found and fixed this round

**Symptom** (P0-B, kit integration): the sealed kit built with
`kit/bare-guide.mjs` as entry produced ZERO bytes on stdout/stderr, exit
code 0, every run — indistinguishable from success by exit status alone.

**Root cause** (the campaign lead, bisected by packing progressively
smaller entries): `bare-guide.mjs`'s own `isMain` guard —

```js
const argv = typeof Bare !== 'undefined' ? Bare.argv : process.argv
const isMain = argv[1] && new URL(import.meta.url).pathname... === argv[1]...
if (isMain) await runGuide()
```

— is correct for a real script invocation (`bare kit/bare-guide.mjs`:
`argv[1]` and `import.meta.url` both name the same real file) but
STRUCTURALLY FALSE once bundled: inside a `bare-pack` bundle, `argv[1]` is
the bundle's own path (`.../app.bundle`) while `import.meta.url` resolves
to the module's VIRTUAL path *inside* the bundle (`/kit/bare-guide.mjs`).
The two can never compare equal, so `isMain` is false every time,
`runGuide()` is never called, and Bare exits 0 on the silent no-op.

**The fix** (per the lead's explicit direction, matching the precedent
already proven by `host/bare-entry.mjs`): a new file,
`kit/bare-guide-entry.mjs` — a thin, UNCONDITIONAL entry
(`import { runGuide } from './bare-guide.mjs'; await runGuide()`, no
guard at all) that `build-bare-kit.mjs --entry=kit/bare-guide-entry.mjs`
now packs, instead of `bare-guide.mjs` directly. `bare-guide.mjs`'s own
`isMain` guard is UNCHANGED and remains correct for its own two real
callers: `bare kit/bare-guide.mjs` (a real script invocation) and
`bare-guide-spike.mjs`'s pure-helper imports (`normalizeCode`/
`groupInFours`), which must NOT trigger a live guide session as a side
effect of being imported for their exports.

**Gated properly, per the lead's explicit ask** — a green unbundled run
proves nothing about the bundled artifact (the exact RULE 4 lesson,
recurring at a new layer): §3's layer 4 builds the real sealed kit, copies
it to a from-scratch directory, and drives it through a real spawned pipe,
asserting on content — never on exit code, which was 0 in both the broken
and the working case.

### 1b. Second defect, found and fixed the same round: the wasm asset was
never offloaded, so the sealed kit could open the messenger but never post

Directly observed once §1a's fix landed (P0-B had predicted it; the lead
confirmed it live): the ceremony now reached the reducer and reported
`(not posted -- ENOENT: ...app.bundle\dist\reducer.wasm)`. Cause:
`apply-bare.mjs`'s DEFAULT self-location (`new URL('../dist/reducer.wasm',
import.meta.url)`) is invisible to `bare-pack`'s static asset detector —
only the literal `import.meta.asset(...)` form is recognised — so the wasm
was never offloaded, and even a present file at that `new URL(...)` path
would resolve to a virtual, unreadable location inside a bundle anyway.

**The fix, in `kit/bare-guide-entry.mjs` only** (matching
`host/bare-entry.mjs`'s already-proven form exactly, per the lead's
direction — `apply-bare.mjs` itself was NOT touched, its self-locating
default stays exactly as it is for every existing spike and the whole Node
line):

```js
import * as fs from 'bare-fs'
import { setWasmSource } from '../host/apply-bare.mjs'
import { runGuide } from './bare-guide.mjs'

const wasmAssetPath = import.meta.asset('../dist/reducer.wasm')
setWasmSource(fs.readFileSync(new URL(wasmAssetPath)))
await runGuide()
```

**Verified, not assumed**: rebuilding the sealed kit now shows
`3.96 MB  dist\reducer.wasm` in `build-bare-kit.mjs`'s own manifest
output, and a message posted through the sealed kit from a from-scratch
hostile directory returns `(posted, seq 2)` and shows up in that same
process's own `/rooms` listing. `bare-guide-spike.mjs`'s layer 4 now
ASSERTS both of these (promoted from "reported" to a real gate) — see §3.

### 1c. Third defect, found by this coder's OWN stress-testing of the
fix, before reporting: an intermittent hang, root-caused and fixed

Running layer 4 repeatedly from hostile geography (not just once) surfaced
a real bug the single successful run had hidden: **1 hang in 4 sampled
sealed-kit runs** (`spawn-pipe-harness.mjs` correctly caught it as `HANG`,
not a false green). Root cause, found by reading `runGuide()` again with
the hang in hand rather than accepting the first success: nothing in
`bare-guide.mjs` ever called `process.exit()`/`Bare.exit()`. The function
relied on the process exiting naturally once stdin reached EOF and no
other work was pending — precisely the "let the loop drain naturally"
shape `PHASE0_GATE_D2_FLUSH_RACE.md`'s RULE 3 already names as producing a
10/10 hang in a DIFFERENT file (`bare-bridge.mjs`'s worker) — the same
bug, in a second file, because the fix for the first one was never
generalized as a rule this coder applied on sight elsewhere.

**The fix**, in `bare-guide.mjs`'s `runGuide()`: an explicit
`Bare.exit(0)`/`process.exit(0)` call after the "Goodbye" line, guarded by
`!io` so a future direct-import test that injects its own fake `io` (not
a real spawned process) never has its own test-runner process killed out
from under it.

**Re-verified after the fix**: 3 full spike re-runs from hostile
geography (6 additional sealed-kit spawn-pipe samples) — 0 hangs, 17/17
every time. Not proof a hang can never recur (a handful of samples is not
a statistical guarantee — same honest limit as every prior phase's
flush-race sampling), but a real, understood, fixed root cause, not a
suppressed symptom.

## 2. UX-law conformance table (against `PHASE0_NOTES_D_REVERIFY.md` §6)

| UX-law item | Status | Detail |
|---|---|---|
| START_HERE double-click entry, zero-argument | **ADAPTED** | The sealed Bare kit's entry point IS `bare-guide.mjs` itself (per `build-bare-kit.mjs`'s own header, already written by another coder to expect this exact file — see §4). No separate `START_HERE.cmd` was built this phase (not in file ownership); the packaging coder's `run_bare_mesh.cmd` is the analogous zero-argument launcher for the Bare kit. |
| Menu structure + plain-question style | **PORTED** | Same 5 items, same numbering, same "type a number" framing. Verbatim except the header line reads "(Bare)" — an honest label, not a content change. |
| `normalizeCode`/`groupInFours` whitespace handling | **PORTED, byte-for-byte** | Same character class (`\s`, U+00A0, U+200B), same `.{1,4}` grouping regex, same never-throws-on-null/undefined contract. Unit-tested directly (8 checks, layer 1) — including a case guide-spike.mjs's own suite doesn't enumerate explicitly (odd-length grouping), added for extra coverage. |
| Three literal verdict words + framing | **NOT EXERCISED, code present but unreachable** | `printVerdictLarge` is ported verbatim (same `=`-rule framing, same "Read this word to the person on the call:" line) — but nothing in this build ever calls it, because the only caller in the original (`checkConnection`) is the honest stub (see below). The FUNCTION is correct; the PATH to it is not wired. |
| Once-per-session firewall offer, exact copy | **COPY PORTED, action stubbed** | The three-line explanation and the exact "Press Enter to continue, or type skip..." prompt are verbatim. The action behind "continue" is an honest one-line stub (no `netsh`/`setup_firewall.cmd` invocation) rather than a real elevation ceremony — see reasoning below. |
| Error fold-line convention | **PORTED** | Same shape: one plain sentence, `--- details for support ---`, raw `err.stack`. Exercised indirectly (every menu action is wrapped the same way `menuLoop` does in the original) but not directly forced to fire in this phase's spike — flagged in §5. |
| FIFO-queue stdin discipline | **PORTED, re-derived, not copy-pasted** | Same guarantee (no line lost regardless of arrival-vs-`ask()` timing), same technique in spirit — but built directly over `bare-bridge.mjs`'s `getRealStdio()` raw `onData`/`onEnd` events instead of `readline`, because `bare-readline` is not installed and the raw-buffer technique is IDENTICAL to what `bare-bridge.mjs`'s ndjson framer already proved correct in Phase 2 (same class of bug, same fix, reused rather than re-derived from scratch). Verified under the exact failure mode P0-D's spec calls out: piped/scripted stdin delivering multiple lines before the guide gets around to asking for the second one (every spawn-pipe scenario in this spike's layer 2 feeds multi-line stdin in one shot). |
| Probe's diagnostic vocabulary (CORRIDOR GREEN/AMBER/RED) | **NOT PORTED** | See "Check the connection" below. |
| Anchor ceremony's shape (install/undo/status) | **MENU SHAPE PORTED, mutation NOT PORTED** | Same three-way prompt (`Press Enter to set this up, type undo..., or type cancel...`), same reasoning as firewall — see below. |

**Why "Check the connection" and the firewall/anchor mutations are honest
stubs, not best-effort approximations:**

1. **No Bare-native probe exists.** Building one (a Bare port of
   `probe.mjs`'s hyperdht/hyperswarm dial/listen logic) is real,
   substantial work outside `mesh/kit/bare-guide.mjs`/`bare-guide-spike.mjs`
   — my file ownership this phase — and was not assigned. Faking a
   CORRIDOR verdict from a probe that doesn't exist would be exactly the
   "approximating it" this phase's brief explicitly forbids.
2. **Firewall rule creation and Scheduled Task install/uninstall are real,
   hard-to-reverse, outward-facing mutations of the actual machine this
   agent is running on.** My own operating guidance is explicit that such
   actions get confirmed before taking, not defaulted into by an automated
   phase. I chose not to build and self-test a real `netsh`/`schtasks`
   invocation via `bare-subprocess` in this sandbox without the chance for
   a human to review the exact command first — a stub that says so plainly
   is the honest choice here, not a corner cut for convenience.

## 3. Spike results (layer-by-layer; full transcript is reproducible via
`node kit/bare-guide-spike.mjs`)

Uses `mesh/host/spawn-pipe-harness.mjs` directly, as instructed — no
hand-rolled spawn logic (Phase 2's own bridge spike predates this harness
and rolled its own; this phase does not repeat that).

| Layer | Checks | Result |
|---|---:|---|
| 1 — pure helpers (`normalizeCode`/`groupInFours`, no process) | 8 | **PASS**, all 8 |
| 2 — real spawn-pipe, full menu flow (open messenger, post, `/rooms`, `/exit`, close), 3 runs each target | 2 | **PASS**, `spawn(node)` OK=3/3, `spawn(bare)` OK=3/3 |
| 2 — real spawn-pipe, out-of-range menu choice handled gracefully, 2 runs each target | 2 | **PASS**, both targets OK=2/2 |
| 3 — negative control A: `spawn-pipe-harness.mjs`'s own shipped `selfTest()` | 1 | **PASS** — correctly distinguishes OK/HANG/TOTAL_LOSS/PARTIAL on its own synthetic fixtures |
| 3 — negative control B: a fixture copy of `bare-guide.mjs` with its closing "Goodbye" line deleted, driven through the SAME real spawn-pipe path as layer 2 | 1 | **PASS** — 3/3 runs correctly flagged as `TOTAL_LOSS` (not OK), proving THIS spike's own success predicate (not just the harness's generic one) can detect a broken guide |
| 4 — the SEALED kit (`build-bare-kit.mjs --entry=kit/bare-guide-entry.mjs`), copied to a from-scratch temp directory, driven via `bare.exe app.bundle` over a real spawned pipe, 2 runs | 3 | **PASS** — build produced `app.bundle`+`bare.exe`; `dist/reducer.wasm` ASSERTED present in the manifest; OK=2/2 on menu-rendered/message-ACTUALLY-posted/room-listed/Goodbye, all four now gated in one predicate |

**17 checks total, 0 failures**, after §1c's exit-hang fix (was 16 before
layer 4 gained the manifest check; briefly RED at 1 failure/17 when the
hang was first caught — see §1c). Re-run 3 full spike runs from hostile
geography after the fix (6 additional sealed-kit spawn-pipe samples, all
clean) plus 2 more from the dev tree — 17/17 every time since the fix
landed. Also run from
`C:\Users\schan\AppData\Local\Temp\claude\...\scratchpad\p1a` (outside the
repo tree, absolute script path) — no leftover temp directories (every
spawn-pipe scenario, including layer 4's sealed-kit copy, uses its own
`mkdtempSync` cwd, cleaned up in a `finally` block; confirmed by listing
both the hostile directory and the OS tmpdir post-run).

**The real end-to-end proof, stated plainly**: layer 2's "full menu flow"
scenario is not a mock — it spawns the actual `bare-guide.mjs` over a real
OS pipe, types `2` (open messenger) `Enter` (past the firewall offer) a
message body `/rooms` `/exit` `5`, and asserts the spawned process's REAL
stdout contains a `posted, seq N` line and the posted message text echoed
back through `/rooms`' `lastPreview` — i.e. the message genuinely
round-tripped through `createBridgeCore.dispatch()` → `mesh-node.mjs` →
`apply-bare.mjs`'s synchronous `WebAssembly.Instance` → `reducer.wasm` and
back, under the real Bare runtime, driven the way a real client process
would drive it.

## 4. Packages needed

**None beyond what was already installed.** `bare-readline` and
`bare-subprocess` were both flagged in the brief as possibly-needed;
neither is imported by `bare-guide.mjs`:
- `bare-readline` — avoided by design (the FIFO-queue I/O layer is
  hand-rolled over `bare-bridge.mjs`'s raw stdio events, see §2's table).
- `bare-subprocess` — would only be needed by the firewall/anchor
  mutation actions, which are honest stubs this phase (§2's reasoning) —
  confirmed present in `node_modules` already (a transitive dependency of
  something else, version `6.1.0`) but never imported, since nothing in
  `bare-guide.mjs` calls it.

`mesh/package.json` was not touched.

## 5. What was NOT verified

- **The firewall/anchor mutation actions themselves** — deliberately not
  built or tested this phase (§2). Whoever picks this up next will need
  `bare-subprocess`'s actual `spawn`/`spawnSync` API confirmed against a
  real `netsh`/`schtasks` invocation (only its export surface was checked
  this phase — `Subprocess.spawn`/`spawnSync` exist, per a one-line probe
  under `npx bare`, but no live process was ever spawned through them).
- **The error fold-line convention's live path** — `reportError` is ported
  and structurally identical to guide.mjs's own, but this spike's scenarios
  never deliberately trigger a menu action's exception, so the fold-line
  format was never observed printing for real in this phase's own runs
  (only read-and-compared against the source).
- **A genuinely clean machine** — same standing caveat as every prior
  phase; Phase 4+'s hostile-machine rehearsal is the real gate for that.
- **Device identity is NOT reused across separate sealed-kit invocations
  from the same directory** — noticed while stress-testing §1c's fix: each
  fresh `bare.exe app.bundle` run prints "created a new room" again rather
  than finding the room from the previous run, even though
  `./data/keys/bare-guide-device.seed` persists correctly on disk.
  `createBridgeCore`'s `rooms` Map starts empty every process start, and
  nothing in `bare-guide.mjs` re-discovers a previously-created room's
  corestore from `ROOM_STORAGE_DIR` — `createSocialRoom` always mints a new
  `social-<random>` subdirectory. Not a regression from this round's fixes
  (same behavior before them), not blocking (each session's messenger
  still works correctly end-to-end), but a real limitation for anything
  beyond a single-session demo. Flagged for whoever picks up the messenger
  UX next, not fixed here (out of this round's scope).
- **Device-identity persistence across a REAL machine restart** — the
  `./data/keys/bare-guide-device.seed` file is written and re-read
  correctly within a single spike run (each spawn-pipe scenario uses a
  fresh temp cwd, so persistence-across-runs within one cwd was checked
  implicitly by the 3-repetition layer-2 scenario reusing the same seed
  file across its 3 spawns) but never tested across a literal process
  restart with a deliberately preserved, non-temp directory.
- **Multi-device / real replication through the messenger** — this
  phase's messenger integration proves ONE device's local post/list
  round-trip through the real reducer; it does not test a second device
  actually joining the same room (that machinery exists in
  `bare-bridge.mjs`'s `createSocialRoom`/`redeemInvite` methods, already
  gated in Phase 2, but wasn't re-exercised through the guide specifically).

## 6. Files

- `mesh/kit/bare-guide.mjs` — the ported guide (new).
- `mesh/kit/bare-guide-entry.mjs` — the thin, unconditional sealed-kit
  entry point (new, added this round to fix §1a's defect).
- `mesh/kit/bare-guide-spike.mjs` — the 16-check gate: pure helpers, real
  spawn-pipe scenarios, two negative controls, and the sealed-kit
  from-scratch ceremony (new; layer 4 added this round).
- `mesh/docs/bare-campaign/PHASE3_GUIDE_REPORT.md` — this file.

Not touched: `mesh/kit/guide.mjs`, `mesh/kit/guide-spike.mjs`, the rest of
the Node kit, `mesh/host/**`, `mesh/host/apply.mjs`, `mesh/reducer/**`,
`mesh/cmd/**`, `mesh/goldens/**`, `mesh/package.json`.
