# SC-1 Report — Rooms that survive the night (`kit-registry` port)

**Coder:** Sonnet 5 · **Date:** 2026-07-20 · **Branch:** `feat/fable-sealed-corridor`

## 1. Verdict

**YES — the sealed guide now reopens its room across a process restart**,
closing the exact gap `PHASE3_GUIDE_REPORT.md` §5 flagged: "each fresh
`bare.exe app.bundle` run prints 'created a new room' again rather than
finding the room from the previous run." `bare-guide.mjs`'s
`ensureMessengerCore()` now reopens every room this device has previously
created (ported from `kit-host.mjs`'s own boot loop) before `openMessenger`
ever checks `listRooms`, and room creation now goes through
`social-room.mjs`'s `createSocialRoom` directly with a **stable** storage
directory name, mirroring `kit-repl.mjs`'s declared `create()` deviation,
instead of the bridge core's own `createSocialRoom` wire method (which picks
an unpredictable `social-<hex>` dir every call and would give the reopen
loop nothing stable to reopen against).

Verified over a real spawned pipe, driving the actual `bare.exe app.bundle`
sealed artifact from hostile geography (temp directories outside the repo),
never in-process, never a shell pipe.

## 2. What was built

- **`mesh/kit/bare-registry.mjs`** (new) — a Bare-native port of
  `kit-registry.mjs`. Same JSON shape (`rooms.json` in `<keysDir>`), same
  idempotent-by-`roomKey` semantics, same "a corrupt registry must never
  crash boot" contract, same no-op `updateRoomRegistryPeer` when the room
  isn't registered. Two changes from the precedent, both forced by the Bare
  import discipline: `fs` via `#fs` instead of `node:fs`, and a hand-rolled
  `joinPath` (same shape as `bare-bridge.mjs`'s own) instead of `node:path`.
  `updateRoomRegistryPeer` is exported but not called from `bare-guide.mjs`
  this mission — SC-2 (the network leg) is a separate mission and is the
  natural caller once a real `/connect` exists under Bare.

- **`mesh/kit/bare-guide.mjs`** (edited) — only `ensureMessengerCore` /
  `openMessenger` and their imports/constants, per the file-ownership rule.
  `ensureMessengerCore(write)` now loads `bare-registry.mjs`'s registry and,
  for each entry, calls `createMeshNode` against the same storage dir +
  `authorityPub`/`encryptionKey`/`bootstrap` it was opened with originally,
  then `core.registerRoom(node.key, node)` — `kit-host.mjs`'s own reopen
  contract, ported. A single entry that fails to reopen is caught, reported
  to the person at the keyboard as one plain sentence via the guide's own
  `write`, and does **not** abort the rest of the loop or the guide itself.
  `openMessenger` then calls `listRooms` as before; if a room already came
  back from the reopen loop it is found there and **no new room is
  created** — the create branch (with its own stable `'room-guide'`
  directory name and a fresh `hypercore-crypto` `randomBytes(32)`
  encryption key) only runs the very first time. The two client-facing
  lines:
  - reopened: `(found your earlier conversation again -- "kitchen table")`
  - created: `(created a new room for this kit -- "kitchen table")` (unchanged copy)

  Neither line, nor anything else in this file, prints the raw 64-hex room
  key to the client (verified by the spike's own `RAW_HEX64` regex check on
  every reopen-leg run, not just read-and-compared against the source).

- **`mesh/kit/bare-registry-spike.mjs`** (new) — the gate.

- **`mesh/package.json`** — one line added: `"sc1spike": "npm run build &&
  node kit/bare-registry-spike.mjs"`, appended at the end of the `scripts`
  block per the fencing brief (minimize conflict with the concurrent
  `bare-net.mjs`/`build-bare-kit.mjs` coder — confirmed no conflict; that
  coder's own `sc2spike` line landed alongside mine, both present).

## 3. Buffer availability under Bare — verified empirically, not assumed

The brief asked me to check this rather than trust `bare-bridge.mjs`'s
precedent blindly. Ran directly against `bare.exe`:

```
typeof Buffer: function
Buffer.from works, length: 4 deadbeef
```

`Buffer` is a real Bare global (same as `bare-bridge.mjs`'s own
`createSocialRoom`/`redeemInvite` already rely on) — `Buffer.from(entry.
encryptionKey, 'hex')` in the reopen loop is safe. `crypto` (the global) is
**not** present under Bare (`typeof crypto === 'undefined'`) — irrelevant
here since this file uses `hypercore-crypto`'s `randomBytes`, not the
`crypto` global, matching the brief's explicit instruction to avoid
`node:crypto`/`randomUUID`.

## 4. Gate results — exact command: `npm run sc1spike` (or `node
kit/bare-registry-spike.mjs` after `npm run build`)

26 checks, 0 failures, `SC1 REGISTRY SPIKE GREEN`.

| Check | Measured |
|---|---|
| Sealed kit build (`build-bare-kit.mjs --entry=kit/bare-guide-entry.mjs`) produced `app.bundle`+`bare.exe`, and `dist/reducer.wasm` is in the manifest | PASS |
| Hostile temp dir path contains no `#` (merge-gate finding — "#" breaks Bare addon resolution), all 21 directories created (16 reopen cycles + 5 negative-control) | 21/21 PASS |
| **Reopen leg — run 1 creates+posts, run 2 (same `cwd`) finds the SAME room and reads back run 1's distinctive message, create-line absent, no raw 64-hex key printed** | **16/16 (OK=16/16 PARTIAL=0/16 TOTAL_LOSS=0/16 HANG=0/16)** |
| **Negative control A — a run-2-shaped script pointed at a FRESH (never-run) directory must NOT find the room, must print the create line, must NOT contain run 1's message** | **5/5 (OK=5/5 PARTIAL=0/5 TOTAL_LOSS=0/5 HANG=0/5)** |
| Negative control B — `spawn-pipe-harness.mjs`'s own `selfTest()` (good/hang/total-loss/partial fixtures all correctly classified) | PASS |

The N=16 reopen leg is not a single `runSpawnPipe({runs:16})` call, because
each cycle is a **two-step** scenario (run 1, then run 2, same directory) —
`runSpawnPipe` drives one script N times, not a two-script sequence. I
called `runSpawnPipe` with `runs: 1` twice per cycle (still the shipped
harness, no hand-rolled spawn logic), 16 independent cycles each in its own
fresh `mkdtempSync` directory outside the repo, and aggregated the results
into a `runSpawnPipe`-shaped object so `formatResult` could still print the
one-line measured-fraction summary the campaign's method rules require.

### What this run actually proves, concretely

Layer 2's positive assertion is content-based, not a mock: run 1 spawns the
real `bare.exe app.bundle`, types `2` (open messenger), `skip` (past the
firewall offer), a distinctive message body
(`sc1-registry-persistence-proof-3f8a1c`), `/exit`, `5` — and the process
then **exits**. A brand-new `bare.exe app.bundle` process is spawned
against the **same directory** as `cwd`, types `2`, `skip`, `/rooms`,
`/exit`, `5` — and its real stdout contains
`(found your earlier conversation again -- "kitchen table")` and the exact
distinctive string from the FIRST process's run, with the create-path line
and any 64-hex room key **absent**. This is the genuine "does state survive
a real process restart" proof, not an in-memory reconnect — two entirely
separate OS processes, driven the way a real client's double-click launcher
is driven (per `run_bare_mesh.cmd`'s own shape).

### One predicate bug found and fixed during this work (documented per honesty norm)

My first draft of `run1Success` required the distinctive message text to
appear in **run 1's own stdout**. It does not — the guide only echoes a
posted message body back through `/rooms`' `lastPreview`, and `RUN1_STDIN`
deliberately never calls `/rooms` (calling it would make run 1
indistinguishable from a test of `/rooms` itself rather than of
persistence). First execution of the gate was RED on all 16 reopen cycles
with `run1: "PARTIAL"` for exactly this reason — caught before reporting
green, not after. Fixed by dropping that clause from `run1Success` (the
`posted, seq N` regex already proves the post happened); run 2's predicate
still requires the distinctive text, which is where the real round-trip
assertion belongs. Re-ran clean: 16/16.

## 5. What was verified vs. NOT verified

**Verified:**
- Room persistence across a real process restart, same directory, N=16,
  0 failures.
- The negative control actually goes red under the right conditions (a
  fresh directory does not falsely claim reopening), N=5, 0 failures —
  this is what makes the positive 16/16 trustworthy rather than a harness
  that always says yes.
- `Buffer` availability under Bare, empirically (§3).
- No raw 64-hex room key is ever printed by this guide (checked
  programmatically on every one of the 16 positive runs, not just by
  reading the source).
- "Same data dir" = same `cwd` (the guide's storage is CWD-relative,
  `./data/...`) — proven differentially: identical `RUN2_STDIN` script
  produces the reopen line when `cwd` matches run 1's directory (16/16) and
  the create line when it does not (5/5 negative-control dirs). This *is*
  the verification the brief asked for, not an assumption carried over from
  reading the code.
- `bare-guide-spike.mjs`'s existing 17-check regression suite re-run after
  this change: still 17/17 green (its layer-2 "full flow" scenario reuses
  the SAME `mkdtempSync` `cwd` across 3 sequential runs via `runs: 3`, so it
  now exercises the reopen path on runs 2 and 3 without ever having been
  designed to — its predicate only requires the posted-message text and
  `Goodbye`, both still present regardless of create-vs-reopen wording, so
  it passed without needing changes).

**NOT verified:**
- **A literal machine restart** (process killed by a real reboot, not just
  a new spawn) — every "restart" here is a fresh `spawn()` of a fresh OS
  process against a directory left on disk by a prior one, which is the
  correct proxy for what a client double-clicking `run_bare_mesh.cmd` again
  tomorrow experiences, but was not tested across an actual Windows
  restart/hibernate cycle.
- **Concurrent/racing invocations** — two `bare.exe app.bundle` processes
  running against the SAME directory at the same time. Not in scope for
  this mission (SC-1 is about one device's own state surviving a restart,
  not about two writers); flagged for whoever builds the corridor's
  concurrent-write story (SC-2/SC-3).
- **`updateRoomRegistryPeer`** — ported and exported, never called from
  `bare-guide.mjs` (no `/connect` exists in the guide yet — that's SC-2's
  network leg). Untested beyond `bare-registry.mjs`'s own straightforward
  port of `kit-registry.mjs`'s logic.
- **A hand-edited / corrupted `rooms.json` on a real field machine** — the
  "never crash boot" contract is inherited verbatim from `kit-registry.mjs`
  (`try { JSON.parse(...) } catch { return [] }`) but this spike never
  constructs a deliberately malformed registry file and drives the guide
  against it. Flagged as a gap, not fixed here (out of this mission's
  required gate, which is silent on this case) — a reasonable follow-up
  negative control for SC-4's regression pass.
- **A second, differently-titled room** — this guide only ever
  auto-creates one room ("kitchen table"); the registry format supports
  multiple entries (kit-host.mjs's own multi-room reopen loop is the
  precedent), and `bare-guide.mjs`'s reopen loop iterates ALL entries, but
  nothing in this guide's current menu ever produces a second entry to
  reopen — untested because there is no code path that creates one yet.

## 5a. A transient failure observed, root-caused, not a defect in this
mission's own code

Re-running `npm run sc1spike` a second time (to confirm before committing)
came back RED at layer 1 — `build-bare-kit.mjs` failed to produce
`app.bundle`/`bare.exe`. A third run, seconds later, was clean (26/26)
again. Root cause: the SC-2 coder is concurrently running `build-bare-kit.
mjs` (their own `sc2spike`/`buildbarekit`) against the SAME shared output
directory (`kit/dist-bare`) at the same time — that builder unconditionally
`rmSync`s and rebuilds its target directory (`build-bare-kit.mjs`'s own §1
comment: "a full wipe-and-rebuild is unconditionally correct here"), which
is safe for one agent running it alone but races when two processes hit it
concurrently. Not a bug in `bare-registry.mjs`/`bare-guide.mjs`/this spike
— flagged here because it is exactly the kind of intermittent-looking
failure this campaign's own method rules (Rule 5, sample-size-is-the-test)
warn against dismissing without a root cause. Re-run clean immediately
after; no code change was needed. Whoever runs SC-4's full regression pass
should serialize kit builds across coders, or each mission's spike should
build into its own private copy of `dist-bare` — flagged for that mission,
not fixed here (outside this mission's file ownership).

## 6. Deviations from the brief, and why

1. **Fixed stable directory name (`'room-guide'`) instead of an id-derived
   one.** The brief's precedent (`kit-repl.mjs`'s `create()`) uses
   `room-${randomUUID()}` because that REPL supports founding multiple
   named rooms via repeated `/create <title>`. `bare-guide.mjs` only ever
   auto-creates a single room (the "kitchen table" kit, one shared room per
   device — `kit-host.mjs`'s own kitchen-table UX doc), and that creation
   path can only ever run once per device (once a room exists, `listRooms`
   always finds it first). A fixed name is therefore correct here and
   avoids needing any id generation at all for this call site. Documented
   in `bare-guide.mjs`'s own comment at the call site, not silently done.
2. **`ensureMessengerCore` takes a `write` parameter it didn't have
   before.** Needed so a per-room reopen failure can be reported to the
   person at the keyboard (matching `kit-host.mjs`'s own `log(...)`
   callback contract) rather than silently swallowed or thrown through the
   guide's own `try/catch`-per-menu-action wrapper (which would misreport a
   partial reopen failure as a generic "something went wrong" for the
   WHOLE messenger, not the one bad entry). Both existing call sites
   (`openMessenger`) already have a `write` in scope, so this is not a
   breaking change to any caller.

No stop-and-report triggers were hit: no new npm dependency, no
protocol-v0 change, no touch to `mesh/reducer/**` or capability/invite
semantics, no Holesail.

## 7. Files

- `mesh/kit/bare-registry.mjs` (new)
- `mesh/kit/bare-guide.mjs` (edited — `ensureMessengerCore`/`openMessenger`
  and their imports/constants only)
- `mesh/kit/bare-registry-spike.mjs` (new, the gate)
- `mesh/package.json` (one line: `sc1spike` script)
- `mesh/docs/bare-corridor/SC1_REPORT.md` (this file)

Not touched: `mesh/kit/bare-net.mjs`, `mesh/kit/build-bare-kit.mjs` (the
concurrent coder's files), `mesh/reducer/**`, `mesh/host/**`,
`mesh/kit/kit-host.mjs`/`kit-registry.mjs`/`kit-repl.mjs` (the Node kit,
untouched precedent), `mesh/kit/bare-guide-entry.mjs`,
`mesh/kit/bare-guide-spike.mjs`, `mesh/kit/bare-bridge.mjs`.

## 8. Gate command to re-run

```
npm run sc1spike
```

equivalently: `npm run build && node kit/bare-registry-spike.mjs`
