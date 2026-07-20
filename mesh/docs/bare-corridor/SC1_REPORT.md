# SC-1 Report — Rooms that survive the night (`kit-registry` port)

**Coder:** Sonnet 5 · **Date:** 2026-07-20 · **Branch:** `feat/fable-sealed-corridor`

## 0. Gate-review addendum (read this first)

This report was updated after gate review. Two items were closed
(§6 deviation 3 answers the `storageDir` question; §4/layer 4 adds the two
required negative controls: malformed/non-array `rooms.json`, and a
dangling registry entry whose storage folder is missing — the latter
surfaced and fixed a REAL bug, see §5b). A **third item was found during
that work and is not closed**: an intermittent `HANG` outcome, roughly
3-10% of individual `bare.exe` spawns across four full gate runs, that
occurs on code paths unrelated to this mission's own reopen logic (see
§5b). This is reported honestly rather than re-run until a clean pass
appeared — the campaign's own method rules exist for exactly this
situation. **The gate verdict on this specific finding is the reviewer's
call, not mine; I have not marked this mission done pending that call.**

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

The kit is now built ONCE per gate run into a **private** output directory
(`--out=kit/.sc1-dist`, the orchestrator's own `build-bare-kit.mjs` fix —
see §5a: the default `kit/dist-bare` is a shared target two coders'
concurrent builds can stomp on), then copied cheaply into each hostile
temp dir rather than rebuilt per cycle.

**41 checks total.** Four full gate runs were executed this round (one
before the gate-review additions, three after). Results were NOT uniform —
see §5b for the honest reason why, which is a real intermittent `HANG`
finding, not noise dismissed without investigation. The table below reports
the LAST run (all 41 green) alongside the aggregate across all four, per
this campaign's own "measured fraction, never a single lucky sample"
discipline:

| Check | Last run | Aggregate across 4 runs |
|---|---|---|
| Sealed kit build (`--out=kit/.sc1-dist`) produced `app.bundle`+`bare.exe`, `dist/reducer.wasm` present | PASS | PASS, 4/4 |
| Hostile temp dir path contains no `#` | PASS, every dir every run | PASS, every dir every run |
| **Reopen leg (16 cycles/run) — run 1 creates+posts, run 2 finds the SAME room, reads back the message, create-line absent, no raw 64-hex key** | **16/16** | **A:15/16 (1 hang) · B:15/16 (1 hang) · C: not fully captured (see §5b) · D:16/16** |
| **Negative control A — fresh dir must NOT find the room** | **5/5** | **5/5 every run (20/20)** |
| Negative control C — malformed JSON `rooms.json` (3/run) | 3/3 | 4/4 runs clean where captured |
| Negative control C — well-formed non-array `rooms.json` (3/run) | 3/3 | A:3/3 · B:2/3 (1 hang) · C:3/3 · D:3/3 |
| Negative control C — dangling entry, storage folder missing (3/run) | 3/3 | A:3/3 · B:3/3 · C:2/3 (1 hang) · D:3/3 |
| Negative control D — `spawn-pipe-harness.mjs`'s own `selfTest()` | PASS | PASS, 4/4 |

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
- ~~**A hand-edited / corrupted `rooms.json` on a real field machine.**~~
  **RETRACTED by the gate reviewer, 2026-07-20 — this entry contradicted
  §4 of this same report and is left visible rather than deleted, per the
  campaign's own retraction norm.** It was written before the gate reviewer
  asked for the control, and was not updated when the control landed. §4's
  table now records three malformed-registry negative controls (invalid
  JSON, well-formed non-array, and a dangling entry whose storage folder is
  missing), so the contract is executed, not merely inherited by reading.

  **Independently re-verified at a HIGHER layer** by the orchestrator's own
  `kit/sealed-corridor-gate.mjs` leg C, which drives the REAL
  `run_bare_mesh.cmd` launcher (this spike drives `bare.exe app.bundle`, one
  layer below what a client double-clicks) against **six** fixtures —
  invalid JSON, well-formed non-array, an array of nonsense, a dangling
  storage folder, a malformed `encryptionKey`, and an empty file — at
  **3/3 each on a quiet machine**.

  What remains genuinely unverified is narrower and is stated as such: a
  registry corrupted in a way nobody has thought to imitate (a torn write
  from a real power cut mid-`writeFileSync`, for instance, rather than a
  hand-authored malformed file).
- **A second, differently-titled room** — this guide only ever
  auto-creates one room ("kitchen table"); the registry format supports
  multiple entries (kit-host.mjs's own multi-room reopen loop is the
  precedent), and `bare-guide.mjs`'s reopen loop iterates ALL entries, but
  nothing in this guide's current menu ever produces a second entry to
  reopen — untested because there is no code path that creates one yet.
- ~~**Whether §5b's HANG finding is a genuine infinite hang or a spawn slow
  enough to exceed a fixed 20s timeout under concurrent load.**~~
  **ANSWERED by the gate reviewer, 2026-07-20 — it is load, not a hang.**
  The original text is kept visible because asking the question was the
  right call and the answer arrived from a different harness.

  The orchestrator's `kit/sealed-corridor-gate.mjs` hit the identical
  signature independently: two fixtures at 2/3, each failure a HANG whose
  stdout was **completely empty**. Re-run with **no code change at all** on
  a machine measured quiet (`0` live `bare.exe`, versus `5 bare.exe + 2
  node.exe` during the loaded run): **3/3, every fixture, every leg.**

  The empty stdout is the discriminator, and it is worth writing down as a
  reusable tell: a kit wedged mid-ceremony leaves PARTIAL output, because
  it printed its menu before wedging. A process that never got scheduled
  leaves NONE. An empty-stdout HANG is therefore evidence about the
  MACHINE, not about the kit — and should be re-run before it is reported.

  A separate red in that same gate run — a reproducible `EINPROGRESS`
  staging a temp dir — did NOT go away on a quiet machine, which is exactly
  what distinguished it as a second, unrelated cause (the harness was
  retaining ~63 MB per staged kit until the end of the run). Two reds, two
  causes, isolated one axis at a time.

  Still genuinely unverified: the hang rate on a quiet machine at N≥30.
  Both quiet-machine measurements were N=3 per fixture, which is a
  correctness proof, not a rate.

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

## 5b. Two gate-review findings

### Finding 1 — a real bug: a dangling registry entry does not fail safely, it fabricates a phantom room (FIXED)

The gate reviewer asked for a negative control proving a registry entry
whose storage folder is missing (the realistic field case: a human deleted
it) reports one plain sentence and continues, "which is the behavior you
already wrote but have not proven." I did not assume that was true — I
checked it directly first, against `mesh-node.mjs`'s real behavior, outside
Bare:

```
storage path exists before call? false
createMeshNode did NOT throw. node.key = 96be998...
storage path exists after call? true
```

**`createMeshNode` does not throw when the storage directory is missing.**
Corestore silently creates a fresh empty store there instead, and because a
founder's own room always has `bootstrap: null`, Autobase then founds a
brand-new base at that path with a **different key** than the registry's
`roomKey`. Left as originally written (a bare `try { createMeshNode(...) }
catch`), this is worse than a caught exception: `core.registerRoom` would
register the phantom under its new key, `listRooms` would return it, and
`openMessenger` would greet the human with `"found your earlier
conversation again"` for a room that is actually empty and unrelated to
anything they posted — silently WRONG, not silently safe.

**Fix**, in `bare-guide.mjs`'s reopen loop: an explicit `fs.existsSync`
check on the entry's storage path BEFORE ever calling `createMeshNode`. If
the folder is missing, the guide logs `(could not reopen a saved room --
its storage folder is missing)` and `continue`s, exactly the behavior
originally claimed but not previously true. No phantom store is ever
written to disk as a side effect of trying. New negative control
(`bare-registry-spike.mjs` layer 4, "dangling entry"): seeds a
well-formed-but-dangling entry into a hostile dir's `data/keys/rooms.json`
and asserts the exact sentence appears — 3/3, 3/3, 2/3 (1 hang, see below),
3/3 across the four runs; the sentence itself was present in 12/12 attempts
including the one that later hung (the hang happened AFTER the sentence
printed, during message-posting — see Finding 2).

Also added: malformed-JSON and well-formed-non-array `rooms.json`
scenarios (`bare-registry.mjs`'s own inherited `kit-registry.mjs` contract,
now exercised for real instead of read-and-compared) — both behave exactly
as documented, guide reaches the menu and says Goodbye every time it
wasn't hit by Finding 2's unrelated hang.

### Finding 2 — an intermittent HANG, ~3-10% of spawns, root cause NOT fully determined, likely NOT this mission's own code

Adding the new negative controls surfaced something this mission's original
16/16-and-5/5 result never hit: across four full gate runs, individual
`bare.exe app.bundle` spawns occasionally never complete within the 20s
timeout and get killed as `HANG` by `spawn-pipe-harness.mjs`.

**Measured, not estimated:**

| Run | Reopen leg (32 spawns) | Non-array control (3 spawns) | Dangling control (3 spawns) | Hangs this run |
|---|---|---|---|---|
| A | 15/16 cycles OK — cycle 13: **both** run 1 and run 2 hung | 3/3 | 3/3 | 2 |
| B | 15/16 cycles OK — cycle 14: run 1 hung, run 2 was PARTIAL (not hang) | 2/3 — cycle 2 hung | 3/3 | 2 |
| C | not fully captured (output truncated by my own `tail`) | 3/3 | 2/3 — cycle 3 hung | ≥1 |
| D | 16/16 clean | 3/3 | 3/3 | 0 |

A supplementary, tighter investigation (outside the committed spike, `N=30`
run1→run2 cycles against the already-built kit, no other scenarios mixed
in) hit the same class of failure at cycle 19 — **both run 1 and run 2
hung simultaneously**, and the killed process left its temp directory
`EBUSY` (rmdir failed: "resource busy or locked") even after `SIGKILL`,
meaning a file handle stayed open past the kill. A second, independent
attempt at the same investigation hit a hang on cycle 1 and then a
`cpSync` failure (`EINPROGRESS`) on cycle 3's directory copy — a different
Windows-level symptom of the same underlying contention.

**Why I believe this is very unlikely to be a defect in SC-1's own code,
though I cannot rule it out with certainty:**
1. It hit **run 1**, which never touches the reopen loop (a fresh directory
   has an empty registry — the `for` loop over `loadRoomRegistry()` has
   nothing to iterate).
2. It hit the **malformed-JSON** and **non-array** negative controls, whose
   registry never has a valid entry either (`loadRoomRegistry` returns `[]`
   before ever reaching a `createMeshNode` call).
3. `tasklist` showed **16 simultaneous `bare.exe` processes** running on
   this machine at one point during this investigation — far more than my
   own sequential script (which runs one `bare.exe` at a time) could ever
   produce on its own. The task list at that time showed `SC-2` and
   `SC-3b` missions active concurrently. This is a shared machine with
   multiple coding agents each spawning `bare.exe` processes at the same
   time; a fixed 20s timeout under that load is a very plausible source of
   an occasional real hang OR a spawn so slow it exceeds the timeout
   without ever being a genuine infinite hang (I did not distinguish these
   two cases — see "not verified" below).

**What I did NOT do, and why:** I did not keep re-running the gate hoping
for a clean pass to report instead of this finding — that is exactly the
"fishing for green" this campaign's own rules warn against. I also did not
attempt to fix, mitigate, or characterize this further (e.g. raising the
harness timeout, isolating a quiet machine, bisecting whether it predates
this mission's changes) because: (a) it is very likely NOT in this
mission's file ownership (`spawn-pipe-harness.mjs`, `bare-bridge.mjs`'s
stdio layer, or the shared-machine contention itself, none of which are
SC-1's files), and (b) the campaign's stop-and-report discipline says an
ambiguous finding like this belongs in front of the gate reviewer, not
patched over unilaterally.

**What I recommend, but do not decide:** re-run this spike in isolation
(no other coder's `bare.exe` concurrently active) before treating its
result as final, and/or consider whether `spawn-pipe-harness.mjs`'s fixed
20s timeout needs to scale with observed concurrent load in a multi-agent
session — a question for whoever owns that file, not a change I made
unilaterally.

**What this does NOT undermine:** the reopen mechanism itself. Every HANG
observed is indistinguishable from "this process was starved of CPU/IO
long enough to exceed a fixed timeout," not "this process completed and
gave a WRONG answer" — no run in any of the four gate executions produced
a `PARTIAL` or `TOTAL_LOSS` outcome with content that contradicted the
reopen contract (one `PARTIAL` occurred, run B cycle 14's run 2, and its
tail shows it was still mid-ceremony, not a wrong answer already given).
The positive claim ("when it completes, it completes correctly") is
supported by every completed run across all four executions, with zero
counterexamples.

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
3. **`createBridgeCore`'s `storageDir` changed from `ROOM_STORAGE_DIR`
   (`./data/corestore/bare-guide-room`, a single fixed subdirectory) to
   `CORESTORE_DIR` (`./data/corestore`, the parent).** Required so the
   reopen loop's `${CORESTORE_DIR}/${entry.storage}` and the create path's
   `${CORESTORE_DIR}/${dirName}` land in the same place a stable per-room
   subdirectory can be chosen under — `ROOM_STORAGE_DIR` was itself a
   fixed leaf directory, leaving nowhere to put a second name under it.
   **Gate-reviewer question, answered directly, not assumed:** does any
   code path still reach `bare-bridge.mjs`'s own `createSocialRoom`/
   `redeemInvite` wire methods (both join `storageDir` with their own
   subdirectory, so they'd now land at `data/corestore/social-<hex>`
   instead of the old `data/corestore/bare-guide-room/social-<hex>`)? I
   grepped `bare-guide.mjs` for every `call('...')` dispatch: only
   `listRooms` and `post` are ever reached from `openMessenger`'s REPL.
   Neither `createSocialRoom` nor `redeemInvite` is dispatched anywhere in
   this file today — this guide's own create path bypasses the wire method
   entirely (deviation 1, kit-repl.mjs's precedent), and no `/join` exists
   in the menu yet (that's SC-3). And since no prior version of this guide
   ever persisted a room, there is no existing on-disk layout under the old
   `ROOM_STORAGE_DIR` for a real device to migrate. **Confirmed harmless
   today; documented as a landmine for SC-3's `/join`/`redeemInvite` wiring
   in a comment at the `CORESTORE_DIR` constant itself, not just here.**

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
