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

**14 checks total, 0 failures.** Re-run 3× back-to-back, 14/14 every time
— stable, no flake observed. Also run from
`C:\Users\schan\AppData\Local\Temp\claude\...\scratchpad\p1a` (outside the
repo tree, absolute script path) — same result, no leftover temp
directories (every spawn-pipe scenario uses its own `mkdtempSync` cwd,
cleaned up in a `finally` block; confirmed by listing both the hostile
directory and the OS tmpdir post-run).

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
- **The sealed-kit packaging path with `bare-guide.mjs` as the actual
  entry** — `build-bare-kit.mjs` (another coder's file, read-only here)
  already names `kit/bare-guide.mjs` as its intended eventual `--entry`,
  but this phase did not run `npm run buildbarekit --entry=kit/bare-guide.mjs`
  or verify the packed bundle launches correctly — that is a packaging-side
  gate, not this phase's.
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
- `mesh/kit/bare-guide-spike.mjs` — the 14-check gate, both runtimes, two
  negative controls (new).
- `mesh/docs/bare-campaign/PHASE3_GUIDE_REPORT.md` — this file.

Not touched: `mesh/kit/guide.mjs`, `mesh/kit/guide-spike.mjs`, the rest of
the Node kit, `mesh/host/**`, `mesh/host/apply.mjs`, `mesh/reducer/**`,
`mesh/cmd/**`, `mesh/goldens/**`, `mesh/package.json`.
