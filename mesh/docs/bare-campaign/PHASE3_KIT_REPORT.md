# Phase 3 — The Sealed Kit Builder (Coder P0-B)

**Verdict: YES — an extracted kit runs from hostile geography.** 10/10 real
spawned-pipe runs of the sealed artifact, in a from-scratch directory with
no npm tree and no source, produced correct real-fold output every time;
the actual double-clickable launcher does the same; a negative control
(the identical kit with `reducer.wasm` deleted) was correctly detected as
broken 5/5, proving the rehearsal can report failure and isn't just
confirming its own bias.

---

## 1. What was proved with which entry point

Every gate below used **`mesh/host/bare-entry.mjs`** — the file this coder
already proved end-to-end in isolation during Phase 2 (`PHASE2_ASSET_
LOCATION.md`) — as the sealed artifact's entry point. `mesh/kit/bare-
guide.mjs` (P1A-wasi-shim's client-facing entry) had not landed at build
time and was not blocked on: the builder takes the entry as a parameter
(`--entry=<path>`, default `host/bare-entry.mjs`), so pointing it at the
guide once it exists is a one-line CLI change, not a rewrite. This is
stated here loudly because it matters for reading the rest of this report
correctly: **the kit built and rehearsed in this report demonstrates a real
fold, not the actual client-facing guided-path ceremony** — that swap and
its own rehearsal is follow-on work once `bare-guide.mjs` lands.

## 2. The build command

```
$ npm run buildbarekit
> npm run build && node kit/build-bare-kit.mjs
```

Equivalently, with an explicit entry override for later use:
```
$ node kit/build-bare-kit.mjs --entry=kit/bare-guide.mjs
```

Internally, the builder's one load-bearing command (the verified recipe
from `PHASE0_NOTES_B2_PACKAGING_SPIKE.md`/`PHASE2_ASSET_LOCATION.md`, run
for real, not simulated):

```
bare-pack --host win32-x64 --offload -o kit/dist-bare/app.bundle host/bare-entry.mjs
```

**No hand-pruning anywhere in this builder.** `build-kit.mjs` (the Node
kit) has to hand-walk its own import graph and its own `package.json`
dependency graph because Node has no packaging tool that does this for it —
its own header explains why a wrong guess there is dangerous. `bare-pack`
performs the real static resolution itself; this builder hands it the
entry point and trusts the result. There is no equivalent guessing surface
to get wrong here, and that is a structural property of using `bare-pack`,
not a claim this coder is making about its own carefulness.

## 3. Sealed-folder manifest (real byte sizes, from an actual build)

```
      0.24 MB  app.bundle
     45.14 MB  bare.exe
      3.96 MB  dist/reducer.wasm
      0.16 MB  node_modules/bare-fs/prebuilds/win32-x64/bare-fs.bare
      0.11 MB  node_modules/bare-path/prebuilds/win32-x64/bare-path.bare
      0.17 MB  node_modules/bare-url/prebuilds/win32-x64/bare-url.bare
      0.00 MB  portable.flag
      0.00 MB  run_bare_mesh.cmd

total: 8 file(s), 49.8 MB
```

`bare.exe` (the unmodified runtime prebuild) is ~90% of the total —
consistent with `PHASE0_NOTES_B2_PACKAGING_SPIKE.md` §6's earlier measurement
on a different payload (there ~48 MB total, same shape: bare.exe dominates).
Re-ran the build twice in a row from a clean tree; the manifest was
byte-count-identical both times (same 8 files, same sizes) — deterministic
and re-runnable, per requirement 1.

Every `.bare` addon file above was **offloaded**, not embedded — confirmed
necessary, not a stylistic choice: default embedding is a documented
defect in this `bare-pack`/`bare` version pairing on Windows
(`PHASE0_NOTES_B2_PACKAGING_SPIKE.md` §3a — "specified module could not be
found," reproduces even outside hostile geography). `dist/reducer.wasm` is
likewise a real offloaded sibling file, not embedded — an embedded asset
resolves to a virtual in-bundle path `bare-fs` cannot read (§4b's landmine);
`bare-entry.mjs`'s `import.meta.asset()` + this builder's `--offload` is
the verified-working combination.

## 4. The launcher — CRLF, `%~dp0`-relative, ASCII-only

```bat
@echo off
setlocal
cd /d "%~dp0"

"%~dp0bare.exe" "%~dp0app.bundle"

echo.
echo (the kit has stopped - press any key to close this window)
pause >nul
```

Written via the same `toCrlf()` guarantee `build-kit.mjs` uses (its own
Mission A2 finding: cmd.exe misparses LF-only batch files, and the repo's
LF `.gitattributes` baseline means any `.cmd` that passes through git comes
out LF — the guarantee has to live in the builder, at write time). Verified
on the actual built file, not assumed:

```
$ file run_bare_mesh.cmd
run_bare_mesh.cmd: DOS batch file, ASCII text, with CRLF line terminators
```

Every path in the template is `%~dp0`-relative (the launcher's own
directory); there is no `where`/PATH lookup anywhere in this file, unlike
`build-kit.mjs`'s Node launcher which has to fall back to `where node` when
no bundled `node.exe` is present — the Bare launcher has no such fallback
because `bare.exe` is always present in the sealed folder by construction.
ASCII-only, matching `build-kit.mjs`'s own field-tested rule (a non-ASCII
byte inside a parenthesized `cmd.exe` block was found to corrupt the batch
parser under a non-UTF-8 codepage in an earlier campaign band).

## 5. Rehearsal — hostile geography, real spawned pipes, content assertions

Copied the built kit (`mesh/kit/dist-bare/*`, all 8 files) to a from-scratch
directory outside the repo, with **no `package.json`, no `node_modules`, no
source file, nothing but the 8 kit files** — confirmed by listing the
directory before running anything.

Driven through **`mesh/host/spawn-pipe-harness.mjs`** (P0-D's, not
reimplemented) per the explicit instruction to reuse it rather than roll a
new one — its `runSpawnPipe()` spawns a real OS pipe (the production
topology, not a shell pipe — the exact distinction that hid two real bugs
earlier in this campaign per its own header) and its `isSuccess` predicate
is **content-based**: `stdout.includes('BARE_ENTRY_FOLD_OK
digest=6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410')`
— the exact digest `smoke.mjs`'s own Node-line run of the identical op set
produces (established in Phase 2). Exit code is logged but never the pass
criterion, per the harness's own design law and the instruction repeated
in this task.

### 5a. Harness self-test (run first, per the harness's own design law)

```
selfTest good-control:        OK=5/5 PARTIAL=0/5 TOTAL_LOSS=0/5 HANG=0/5
selfTest hang-fixture:        OK=0/5 PARTIAL=0/5 TOTAL_LOSS=0/5 HANG=5/5
selfTest total-loss-fixture:  OK=0/5 PARTIAL=0/5 TOTAL_LOSS=5/5 HANG=0/5
selfTest partial-fixture:     OK=0/5 PARTIAL=5/5 TOTAL_LOSS=0/5 HANG=0/5
harness self-test: PASS
```

### 5b. The good kit — 10 real spawned-pipe runs

```
good kit: OK=10/10 PARTIAL=0/10 TOTAL_LOSS=0/10 HANG=0/10
  run outcome=OK stdout="BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410" code=0
  (×10, identical)
```

**10/10, zero flush-race flakiness, identical digest every time.**

### 5c. Negative control — the SAME kit with `reducer.wasm` deleted

```
negative control (broken kit): OK=0/5 PARTIAL=0/5 TOTAL_LOSS=5/5 HANG=0/5
  run outcome=TOTAL_LOSS stdout="" stderr="Uncaught ModuleError: ASSET_NOT_FOUND:
    Cannot find asset '../dist/reducer.wasm' imported from
    'file:///.../app.bundle/host/bare-entry.mjs'..." code=3221226505
  (×5, identical)
```

**Correctly detected as broken every time — 0/5 false-positive OK.** This
is the rehearsal's own negative control: a harness that could not
distinguish this deliberately-broken kit from the working one would be
exactly the kind of tool this campaign has already caught twice today
producing a false "green."

### 5d. The actual double-click artifact — first pass, then hardened after gate pre-flight

`runSpawnPipe` (§5b/5c) proves the `bare.exe app.bundle` pipeline directly.
It does not by itself prove the `.cmd` a human actually double-clicks
behaves the same, because `run_bare_mesh.cmd` ends in `pause >nul`, which
blocks forever waiting for a keystroke — a real difference from a
fire-and-forget script. First verification, one-shot, piped a keypress:

```
$ "x" | cmd.exe /c ".\run_bare_mesh.cmd"
BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410

(the kit has stopped - press any key to close this window)
EXITCODE=0
```

**The team lead independently gate-tested this same kit** (before this
addendum) and raised exactly the right concern back: `pause >nul` is
correct and must stay — it's what lets a client actually read the outcome
instead of the window vanishing — but a rehearsal that only ever fires one
piped keystroke isn't the same as proving the real launcher survives
**repeated** automated invocation the way a statistical rehearsal needs.
They also hit real Git-Bash-specific invocation traps (`cmd.exe /c` gets
MSYS-mangled; a bare relative filename isn't found without `cygpath -w`)
and, critically, demonstrated their own standing rule in the process: their
first three attempts LOOKED like a failing kit and were entirely their own
broken invocation — exactly the "can this rehearsal go RED for the right
reason" question this document already applies to the harness itself (§5a),
now applied to the invocation layer too.

**Response: added a documented, explicit non-interactive switch to the
launcher itself** (`ASYMMFLOW_KIT_NONINTERACTIVE`), rather than have the
rehearsal quietly substitute `bare.exe app.bundle` for the real `.cmd`:

```bat
"%~dp0bare.exe" "%~dp0app.bundle"

if defined ASYMMFLOW_KIT_NONINTERACTIVE goto skippause
echo.
echo (the kit has stopped - press any key to close this window)
pause >nul
:skippause
```

Unset (every human double-click, always): identical behavior to before,
verified explicitly, no env var:

```
$ "y" | cmd.exe /c ".\run_bare_mesh.cmd"
BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410
(the kit has stopped - press any key to close this window)
```

Set (rehearsal only): re-ran the FULL rehearsal through the **actual
launcher file**, not the underlying binary, avoiding the Git-Bash traps by
driving it from PowerShell (native `cmd.exe /c`, no MSYS path mangling):

```
$env:ASYMMFLOW_KIT_NONINTERACTIVE = "1"
10x: & cmd.exe /c ".\run_bare_mesh.cmd"   →  all 10 matched the exact expected digest, exitcode=0

NEGATIVE CONTROL (same kit, reducer.wasm deleted), same real launcher, 5x:
run 1..5: matched=False, exitcode=0, output contains
  "Uncaught ModuleError: ASSET_NOT_FOUND: Cannot find asset '../dist/reducer.wasm'..."
```

**10/10 real launcher invocations correct; 5/5 negative-control launcher
invocations correctly detected as broken.** One finding worth its own line:
**through the `.cmd` wrapper, `cmd.exe`'s own reported exit code was 0 in
both the passing AND the deliberately-broken runs** — `run_bare_mesh.cmd`
never propagates `bare.exe`'s own exit code (`if defined
ASYMMFLOW_KIT_NONINTERACTIVE goto skippause` falls through to the end of
the script regardless of what came before). This is a live, concrete
demonstration of why every assertion in this report is content-based, never
exit-code-based — at the launcher layer, exit code carries **zero**
information about success either way.

## 6. What this rehearsal does NOT prove

**This machine has Node and npm installed globally.** A directory with no
`package.json`/`node_modules`/source in it is not the same guarantee as a
machine that has never had Node installed at all — `bare.exe` doesn't
consult Node's install, but this is not proof that nothing on this specific
Windows installation could be an unaccounted-for dependency (a shared DLL
some other installed program also happens to provide, for instance). A
genuinely clean VM with no dev tooling ever installed is explicitly Phase 4
(campaign §3, owner-reserved) and was not attempted here — stated plainly
per the instruction, not overclaimed.

Also not covered by this report:
- **macOS/Linux** — Windows `win32-x64` only, consistent with every prior
  gate in this campaign.
- **The actual client-facing ceremony** (§1) — `bare-entry.mjs` proves the
  packaging pipeline and the reducer-fold boundary; it is not the Guided
  Path UX (`START_HERE`, plain questions, the messenger ceremony) that
  `build-kit.mjs`'s Node kit ships and that `bare-guide.mjs` is meant to
  port. Swapping the entry is designed to be a one-line change (§1) but
  was not exercised because that file does not exist yet.
- **A persistent `data/` directory** and the corresponding rebuild-safety
  guards `build-kit.mjs` has (`hasRealData`/`trySync`/lock-resilience for a
  kit that might be running mid-rebuild) — `bare-entry.mjs` has no
  persistent state, so this builder does an unconditional clean rebuild
  every time. This is CORRECT for the current entry point and INCOMPLETE
  for a future stateful one; flagged explicitly in the builder's own
  header comment (§1 of `build-bare-kit.mjs`) so it isn't rediscovered
  cold when `bare-guide.mjs` lands.
- **The firewall/anchor/probe ceremony** `build-kit.mjs`'s Node kit ships
  (`setup_firewall.cmd`, the anchor cluster, the probe cluster, the
  README/phone-script text) — none of that is reproduced in this sealed
  kit. This report's scope was the packaging mechanism (campaign §0's
  charter: no missing-module class of failure), not the full ceremony
  parity with the Node kit's UX. That parity work is a separate,
  follow-on task once the entry point is the real guide.
- **Two-machine corridor testing** — this rehearsal is one machine
  producing a local fold; it says nothing about the P2P/networking layer
  the eventual real kit needs (that's `bare-bridge.mjs`, P1A-wasi-shim's
  file, untouched and unread beyond its filename in this task).

## 7. `mesh/package.json` — what was appended

Per the instruction (append-only, serialized writes owned by the team
lead): added one script and one devDependency.

```diff
     "harnessselftest": "node host/spawn-pipe-harness.mjs --selftest",
-    "stdioseam": "npm run build && node host/stdio-seam-spike.mjs"
+    "stdioseam": "npm run build && node host/stdio-seam-spike.mjs",
+    "buildbarekit": "npm run build && node kit/build-bare-kit.mjs"
   },
   ...
   "devDependencies": {
     "bare": "^1.30.3",
     "bare-crypto": "^1.15.3",
     "bare-events": "^2.9.1",
     "bare-fs": "^4.7.4",
+    "bare-pack": "^2.2.1",
     "bare-process": "^4.5.1",
     "bare-stream": "^2.13.3"
   }
```

**Honesty note on the append-only instruction:** I added `"bare-pack":
"^2.2.1"` at the end of the object in my edit, but the subsequent `npm i -D
bare-pack` **re-sorted `devDependencies` alphabetically** as an npm side
effect of writing the lockfile-adjacent `package.json` update — it now
sits between `bare-fs` and `bare-process` rather than at the end. This was
npm's own behavior, not an intentional reorder on my part, but it does
technically violate the letter of "never reorder." Flagging it plainly
rather than passing it off as compliant; the *keys and values* are
unchanged (nothing removed, nothing altered), only their order. Let me know
if you want it manually re-ordered to strictly append-only shape.

`bare-pack` was not previously installed in the real `mesh/` tree (only
`bare`/`bare-crypto`/`bare-events`/`bare-fs`/`bare-process`/`bare-stream`
were) — required for this builder to run at all, so it was added.

**Addendum (§10):** two more devDependencies added the same way —
`bare-readline@^1.3.1`, `bare-subprocess@^6.1.0` — for the guide-entry
readiness check. Same npm alphabetical-resort side effect applies (team
lead already ruled this fine: "npm did it, nothing removed or altered, no
action"). Current full `devDependencies` list: `bare`, `bare-crypto`,
`bare-events`, `bare-fs`, `bare-pack`, `bare-process`, `bare-readline`,
`bare-stream`, `bare-subprocess` — all Holepunch/`bare-*` family, build-
time verification of packaging behavior, owner ruling R3.

## 8. `WebAssembly.compile()`/`instantiate()` compliance

The builder itself makes no `WebAssembly.*` call. Its packed entry
(`bare-entry.mjs` → `apply-bare.mjs` → `new WebAssembly.Module()`/`new
WebAssembly.Instance()`, both synchronous) was already verified compliant
in Phase 2 and reconfirmed by this rehearsal's own flakiness check: 10/10
clean runs with zero truncation is itself an empirical check against the
async-compile flush-race class of failure this binding rule exists to
prevent (`PHASE0_GATE_D2_FLUSH_RACE.md`).

## 9. What is NOT verified

1. Everything in §6, restated for completeness: no genuinely clean VM, no
   macOS/Linux, no real client ceremony, no persistent-data rebuild
   discipline, no firewall/anchor/probe parity, no two-machine corridor.
2. `bare-guide.mjs` as the entry — landed in the tree partway through this
   task (`mesh/kit/bare-guide.mjs`, P1A-wasi-shim's, seen as an untracked
   file appearing mid-session) but was NOT read, touched, or run through
   this builder — `--entry=kit/bare-guide.mjs` remains an untested (though
   designed-for) path. Confirming it packs and rehearses clean is follow-on
   work, explicitly not claimed here.
3. The `devDependencies` alphabetical-reorder side effect noted in §7 —
   not reverted; flagged for the team lead's call.
4. `kit/dist-bare/`'s `.gitignore` entry — confirmed by the team lead
   (commit `b993936`) to be their own addition, made because the root
   `.gitignore`'s `dist/` pattern does not match `dist-bare/` and this
   builder's 45 MB `bare.exe` would otherwise have landed in the repo on
   the next broad `git add`. Not this coder's doing; noted for the record.
5. The `ASYMMFLOW_KIT_NONINTERACTIVE` switch (§5d) is new in this addendum
   and has only been exercised by this coder's own rehearsal — it has not
   been reviewed or used by anyone else yet.

---

## 10. README + guide-entry readiness (addendum)

### 10a. The README

Added `README_BARE_KIT.txt` to the builder's output (§4b of
`build-bare-kit.mjs`) — ASCII-only (0 non-ASCII bytes, verified), matching
`README_KITCHEN_TABLE.txt`'s safety-box voice and plain-language discipline
(owner ruling R6: never a command line; D3: "the complexity is ours, the
simplicity is for the end user").

**Written in LF, not CRLF, deliberately** — this looks like it contradicts
the instruction ("ASCII-only and CRLF, per build-kit.mjs's field-tested
rules"), so the reasoning is stated plainly rather than silently deviating:
`build-kit.mjs`'s own `toCrlf()` guard is applied ONLY to files `cmd.exe`
parses (every `.cmd`), and its own header note says exactly why —
"README_*.txt files are plain text, never parsed by cmd.exe — safe." Its
own `README_TEXT`/`README_CORRIDOR_TEXT` constants are written with a bare
`writeFileSync`, no `toCrlf()`. This builder matches that file's actual
practice, not a literal-but-broader reading of the instruction. If this
judgment call is wrong, it's a one-line fix (wrap the `writeFileSync` call
in `toCrlf()`).

**Content is scoped to what THIS kit is actually proven to do** — the
honest framing this whole report has held to throughout. `bare-entry.mjs`
is a technical proof (a real fold, sealed, hostile-geography verified),
not the messenger ceremony. The README says so directly rather than
describing a chat feature this build cannot perform: it tells the reader
what the single printed line means, what to do if it's wrong, and that
there's nothing to type in. Verified end-to-end in hostile geography after
adding it (full manifest now 9 files, 49.8 MB, README included) — the kit
still runs clean:

```
$ ./bare.exe app.bundle
BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410
exit=0
```

**This README will need real revision once `bare-guide.mjs` is the entry**
— it should then describe the actual ceremony, the way
`README_KITCHEN_TABLE.txt` does for the Node kit. That is follow-on work
for whoever sequences the guide-entry swap, flagged here so it isn't
missed, not attempted in this pass (the guide's actual UX is P1A's file,
unread by this coder per the fence).

### 10b. Guide-entry readiness — bare-readline and bare-subprocess

**Added as devDependencies** (append-only, per the fence):
`bare-readline@^1.3.1`, `bare-subprocess@^6.1.0`. Neither was previously
installed. `bare-subprocess` transitively pulls in real native addons:
`bare-pipe`, `bare-tcp`, `bare-os` (each with a `win32-x64` `.bare`
prebuild, confirmed by direct inspection of `node_modules/*/prebuilds/`
before writing any test) — `bare-readline` itself is pure JS (depends on
`bare-stream`, already present, and `bare-ansi-escapes`, also pure JS).

Built a trivial real entry, `mesh/kit/bare-readiness-check.mjs` (this
coder's file, under `kit/`, not `host/` — the fence moved `host/**`
off-limits for this task), exercising REAL behavior, not just import
resolution: a real `bare-readline` interface reading one real line off
stdin, then a real `bare-subprocess.spawnSync()` of the sealed `bare.exe`
itself (via `Bare.argv[0]`, not an assumed path) running a trivial script
and capturing its stdout.

**Packing: clean.** `bare-pack --host win32-x64 --offload -o app.bundle
kit/bare-readiness-check.mjs` succeeds, offloading 14 addon prebuilds
(the two direct packages plus every native-bearing transitive dependency
of `bare-process`/`bare-subprocess`'s own chain — `bare-abort`,
`bare-buffer`, `bare-dns`, `bare-hrtime`, `bare-os`, `bare-pipe`,
`bare-signals`, `bare-stdio`, `bare-structured-clone`, `bare-subprocess`,
`bare-tcp`, `bare-tty`, `bare-type`, plus `bare-fs`/`bare-path`/`bare-url`
from before). `--entry=` proved itself generically here, not just against
`bare-entry.mjs` — confirms the parameterization is real, not hand-tuned
for one file.

**Sanity, unbundled, under Bare directly:** both packages work.

```
$ printf 'hello-readiness\n' | npx bare kit/bare-readiness-check.mjs
READLINE_OK line="hello-readiness"
SUBPROCESS_OK stdout="SUBPROCESS_CHILD_OK"
READINESS_CHECK_DONE
```

**Then the real rehearsal — through a genuine spawned pipe, not a shell
pipe — surfaced a real defect, not a clean pass.** Packed, copied to
hostile geography, driven through `spawn-pipe-harness.mjs` with real
stdin:

```
readiness kit: OK=0/10 PARTIAL=0/10 TOTAL_LOSS=0/10 HANG=10/10
```

**`bare-readline` hangs 10/10 under a real spawned pipe, despite working
cleanly under a shell pipe one line above.** This is exactly the
shell-pipe-vs-spawned-pipe distinction `spawn-pipe-harness.mjs`'s own
header exists to catch (`PHASE0_GATE_D2_FLUSH_RACE.md`'s lesson) — a
shell-pipe-only test would have reported this as working. Isolated the
cause with two follow-up throwaway-shaped diagnostic entries (kept in the
tree, `kit/bare-readiness-check-subprocess-only.mjs` and
`kit/bare-readiness-check-readline-only.mjs`, both this coder's, both
small and self-explanatory):

```
subprocess-only (no readline): OK=5/5 PARTIAL=0/5 TOTAL_LOSS=0/5 HANG=0/5   ← clean
readline-only (no subprocess): OK=0/5 PARTIAL=0/5 TOTAL_LOSS=0/5 HANG=5/5   ← hangs
```

**`bare-subprocess` is clean. The hang is isolated to `bare-readline`
specifically**, under this exact topology: a real OS pipe fed by
`child_process.spawn`'s `stdin.write()`+`.end()`, not a shell-level pipe
built before the process starts. Root cause not diagnosed further (out of
scope for a readiness check) — plausible candidates not investigated:
timing between `stdin.write()` and `bare-readline`'s internal setup, a
raw-mode/TTY assumption that a piped (non-console) stdin doesn't satisfy,
or a difference in how Bare's own pipe primitive delivers data versus how
`bare-readline` expects to consume it.

**What this means for `bare-guide.mjs`, stated plainly:** the guide's
actual production topology is a HUMAN double-clicking the launcher, where
`bare.exe`'s stdin is a real console handle — a third topology, distinct
from both tested here (shell pipe: clean; spawned pipe: hangs), and NOT
independently verified by this readiness check. If the guide (or any
future automated gate for it) is ever driven by spawning `bare.exe` with
piped stdin — which is exactly how a CI-style rehearsal, or a
parent-process-driven guide, would naturally be built — it will hit this
hang. **This is the headline finding of this readiness check, not a
footnote:** `bare-subprocess` is ready; `bare-readline`, as used in the
shape this test exercised it, is NOT proven safe under the automation
topology this campaign's own gates rely on, and needs either a fix, a
different consumption pattern, or an upstream report before `bare-
guide.mjs`'s own gate can trust a spawned-pipe rehearsal of it.

### 10c. Verdict

- **Builder readiness:** YES — `--entry=` genuinely parameterizes, proven
  against a second, different, real entry point (not just re-running the
  same one), and every native addon the guide will pull in via
  `bare-subprocess` offloads and packs cleanly.
- **`bare-subprocess` in hostile geography, real spawned pipe:** YES,
  5/5 clean.
- **`bare-readline` in hostile geography, real spawned pipe:** NO — 0/5,
  reproducible hang, isolated to this package specifically. Works fine
  under a shell pipe and unbundled, which is precisely why this needed a
  real spawned-pipe rehearsal to catch, not a shortcut.

The full readiness entry (`bare-readiness-check.mjs`, exercising both
packages together) is therefore currently blocked by the readline hang
under the spawned-pipe topology — reported honestly per the campaign's
standing law rather than reported as a pass on the strength of the
shell-pipe/unbundled results alone.

---

## 11. Exit-code propagation (for Phase 4's anchor)

**Why this matters, restated from the brief:** the human double-click path
never reads an exit code off a window it closed — but a Windows Scheduled
Task (the Phase-4 anchor's own mechanism, owner ruling R4, not touched by
this coder) drives its "last run result" and retry logic off exit code.
Before this fix the launcher swallowed `bare.exe`'s exit code unconditionally
(`pause` resets `%errorlevel%`, and nothing captured it beforehand) — a
scheduled anchor would have reported healthy forever, on a completely
broken kit, silently.

### 11a. The fix

```bat
"%~dp0bare.exe" "%~dp0app.bundle"
set RC=%errorlevel%

if defined ASYMMFLOW_KIT_NONINTERACTIVE goto skippause
echo.
echo (the kit has stopped - press any key to close this window)
pause >nul
:skippause

exit /b %RC%
```

`RC` is captured immediately after `bare.exe` returns, before `pause` (or
the branch that skips it) can touch `%errorlevel%`; `exit /b %RC%`
propagates the captured value regardless of which branch ran. The
non-interactive switch (§5d) is unchanged and still gates the pause only.

### 11b. Verified — three real breakages, real launcher, real exit codes

Rebuilt, copied to three separate from-scratch hostile directories, drove
the ACTUAL `.cmd` (not the underlying `bare.exe app.bundle`) via PowerShell
(no Git Bash `cmd.exe` mangling), captured `$LASTEXITCODE` and content
together:

```
HEALTHY                          : exitcode=0            contentOK=True
NO-WASM  (reducer.wasm deleted)  : exitcode=-1073740791   contentOK=False
NO-BUNDLE (app.bundle deleted)   : exitcode=-1073740791   contentOK=False
WRONG-DIGEST (runs, wrong answer, never throws): exitcode=0   contentOK=False
```

**Which failure modes now propagate non-zero, and which still report 0 —
stated plainly, as asked:**

- **Now non-zero (propagation actually helps):** deleting `reducer.wasm`
  (an uncaught `ModuleError: ASSET_NOT_FOUND` at import time) and deleting
  `app.bundle` itself (an uncaught `ModuleError: MODULE_NOT_FOUND` — the
  runtime can't even load its own entry). Both are genuine crashes inside
  `bare.exe`, and `bare.exe` itself already exits non-zero for both
  (`-1073740791` signed / `3221226505` unsigned, a Windows
  unhandled-exception status code) — the launcher was simply discarding
  that non-zero value before this fix. **This corrects a specific
  prediction in the brief**: deleting `reducer.wasm` was expected to
  "probably still" yield 0 (reasoning: "that failure happens inside a Bare
  process that exits 0"), but measured directly, it does not — an uncaught
  `ModuleError` at import time crashes the process with a real non-zero
  code, distinct from the silent-0-exit classes this campaign has
  documented elsewhere (the async-`WebAssembly.compile()` flush-race;
  observed-but-undiagnosed throw-yet-exit-0 cases). Worth having the
  prediction corrected with evidence rather than left standing unverified.
- **Still 0 despite propagation, and always will be, by construction:**
  a kit that runs to completion, produces a WRONG answer, and never
  throws at all. Built and tested this directly — a diagnostic entry
  (`kit/bare-readiness-check-wrongdigest.mjs`, this coder's, a copy of
  `bare-entry.mjs`'s own logic with a deliberately different, but valid
  and non-crashing, op set) folds cleanly, prints
  `BARE_ENTRY_FOLD_MISMATCH got=... want=...`, and exits 0 — because
  nothing went wrong from `bare.exe`'s own point of view; only the
  business answer differs from what was expected. **No exit-code fix, in
  the launcher or anywhere else, can turn this case non-zero** — the
  process genuinely, correctly-by-its-own-lights exited clean. This is
  also the most realistic tamper/corruption shape for a real anchor to
  worry about (a subtly wrong `reducer.wasm`, a bit-flipped asset that
  still parses and runs) — silent wrong-answer, not crash.

### 11c. The guidance for whoever wires up anchor health (stated explicitly, per the brief)

**Exit code is now a necessary signal but never a sufficient one.**
Propagation fixes the launcher's own lie (silently flattening every
outcome to 0); it does nothing about `bare.exe`'s own remaining ways to
report false success (§11b's wrong-digest case, and the previously
documented async-compile flush-race and throw-yet-exit-0 classes this
campaign has hit twice already, per the team lead's own count). **An
anchor health check must assert on printed CONTENT — the exact expected
line/digest, or whatever the guide's own real health marker turns out to
be — never on exit code alone.** Exit code is now useful as a fast,
cheap FIRST filter (a non-zero code is unambiguously bad, skip the content
check and flag immediately) but a zero code proves nothing on its own.

Cleanup: rebuilt with the default entry afterward so `kit/dist-bare/`
reflects the primary artifact, not a diagnostic build; all four hostile
test directories removed. `kit/bare-readiness-check-wrongdigest.mjs`
kept in the tree (this coder's fence, `kit/`) alongside the other two
readiness diagnostics, for the same reproducibility reason.

---

## 12. Integration: `--entry=kit/bare-guide.mjs` — the real guided ceremony

**Verdict, stated up front: NO — the sealed kit does NOT run the guided
ceremony from hostile geography under the real production topology.** It
packs. It runs correctly unbundled, and under a shell pipe. Under a real
spawned pipe — the topology this campaign's own binding rules exist
because of, and the one a scripted rehearsal or any future automated gate
would use — it produces **zero bytes of output on stdout or stderr and
exits 0**, reproduced through both the raw `bare.exe app.bundle` pipeline
and the actual `.cmd` launcher. This is reported as a defect to route, per
the explicit instruction, not fixed — `bare-guide.mjs` and
`bare-bridge.mjs` (the likely site of the actual bug, via
`getRealStdio()`) are P1A's files and were not touched.

### 12a. Dependency reality check (item 1 of the brief)

Read `bare-guide.mjs` in full before testing anything (not touched,
read-only). Confirmed directly from its own header and imports: **neither
`bare-readline` nor `bare-subprocess` is used.** The guide hand-rolls its
own FIFO-queue stdin discipline directly over `bare-bridge.mjs`'s
`getRealStdio()` raw data/end events (`createGuideIO`), and every menu
action that would need `bare-subprocess` (firewall elevation, scheduled-
task install/remove) is an honest, clearly-labeled stub — no child process
is ever spawned. **§10's offload verification of those two packages is
therefore moot for this entry** — it remains valid evidence for a
*different* future file that does use them, but it does not describe what
`bare-guide.mjs` actually needs.

**What the guide's real dependency closure looks like, from an actual
build** (`bare-pack --host win32-x64 --offload -o app.bundle
kit/bare-guide.mjs`, real, not simulated):

```
      1.81 MB  app.bundle
     45.14 MB  bare.exe
      0.11 MB  node_modules/bare-abort/prebuilds/win32-x64/bare-abort.bare
      1.30 MB  node_modules/bare-crypto/prebuilds/win32-x64/bare-crypto.bare
      0.16 MB  node_modules/bare-fs/prebuilds/win32-x64/bare-fs.bare
      0.11 MB  node_modules/bare-hrtime/prebuilds/win32-x64/bare-hrtime.bare
      0.11 MB  node_modules/bare-inspect/prebuilds/win32-x64/bare-inspect.bare
      0.17 MB  node_modules/bare-os/prebuilds/win32-x64/bare-os.bare
      0.11 MB  node_modules/bare-path/prebuilds/win32-x64/bare-path.bare
      0.12 MB  node_modules/bare-pipe/prebuilds/win32-x64/bare-pipe.bare
      0.12 MB  node_modules/bare-signals/prebuilds/win32-x64/bare-signals.bare
      0.11 MB  node_modules/bare-stdio/prebuilds/win32-x64/bare-stdio.bare
      0.12 MB  node_modules/bare-tty/prebuilds/win32-x64/bare-tty.bare
      0.12 MB  node_modules/bare-type/prebuilds/win32-x64/bare-type.bare
      0.17 MB  node_modules/bare-url/prebuilds/win32-x64/bare-url.bare
      0.14 MB  node_modules/fs-native-extensions/prebuilds/win32-x64/fs-native-extensions.bare
      0.15 MB  node_modules/quickbit-native/prebuilds/win32-x64/quickbit-native.bare
      7.81 MB  node_modules/rocksdb-native/prebuilds/win32-x64/rocksdb-native.bare
      0.22 MB  node_modules/simdle-native/prebuilds/win32-x64/simdle-native.bare
      0.78 MB  node_modules/sodium-native/prebuilds/win32-x64/sodium-native.bare
      0.00 MB  portable.flag
      0.00 MB  README_BARE_KIT.txt
      0.00 MB  run_bare_mesh.cmd

total: 23 file(s), 58.9 MB
```

**Packs cleanly, no `MODULE_NOT_FOUND`, no manual pruning needed** — the
same structural property every prior build in this report has had.
`rocksdb-native` (7.81 MB, the single largest addon) confirms the guide's
messenger path really does reach corestore's storage layer, consistent
with `createBridgeCore`'s real `storageDir` option.

**One thing conspicuously ABSENT from this manifest and worth flagging
loudly: no `dist/reducer.wasm`.** Confirmed by listing the copied hostile
directory before running anything — no `dist/` folder at all. This traces
to `apply-bare.mjs`'s DEFAULT self-locating path (`new URL('../dist/
reducer.wasm', import.meta.url)`, unchanged since Phase 2 — this coder's
own file, but out of fence for this task, not edited) — `bare-pack`'s
static asset detector only recognizes the literal `import.meta.asset()`/
`require.asset()` syntax (established in Phase 0, §4c), not a dynamic
`new URL(..., import.meta.url)` construction, so it is never offloaded.
**This is a second, independent gap from the total-loss finding below** —
see §12d for why it wasn't reached by this rehearsal's actual script, but
it WILL surface the moment `openMessenger`'s reducer-backed `post`/
`createSocialRoom` calls actually execute inside a *working* packed guide,
via `mesh-node.mjs` → `#apply` → `apply-bare.mjs`'s default `loadModule()`
hitting a virtual bundle path with no real file behind it. Reporting this
as a second finding, not silently folding it into §12c's headline one —
they are different bugs with different fixes (one is a stdio/output
defect, the other is an asset-resolution gap in a file this coder
authored and would normally fix, but is fenced off from touching in this
task).

### 12b. Sanity — unbundled and shell-piped: WORKS

Scripted the exact ceremony from the brief (menu → messenger → post a
real message → `/exit` → close), unbundled, driven by a shell pipe:

```
$ printf '2\n\nhello from rehearsal\n/exit\n5\n' | npx bare kit/bare-guide.mjs
Welcome. This will walk you through connecting to the other computer.
====================================
  ASYMMFLOW MESH -- GUIDE (Bare)
====================================
[1] Check the connection
[2] Open the messenger
...
Before we connect, this computer needs one quick permission.
...
Opening the messenger now.
...
(created a new room for this kit -- "kitchen table")
>
  (posted, seq 2)
>
====================================
  ASYMMFLOW MESH -- GUIDE (Bare)
====================================
...
> 
Goodbye -- this window is safe to close.
```

**Full ceremony, correct, matches the team lead's own independent
verification.** This corroborates the "14/14 gate pass" — it is real, for
the topology it was tested under.

### 12c. THE FINDING — real spawned pipe, packed, hostile geography: TOTAL SILENT LOSS

Packed (§12a), copied to a from-scratch hostile directory (no npm tree, no
source — confirmed by listing it), then driven through the SAME topology
this campaign's own binding rules were written for: a real
`child_process.spawn` pipe, `stdin.write()`+`.end()` after spawn, a
parent reading `stdout`/`stderr` via `'data'` events — via
`spawn-pipe-harness.mjs`, self-tested first (harness self-test PASS, same
as every prior rehearsal in this report):

```
guide ceremony: OK=0/5 PARTIAL=0/5 TOTAL_LOSS=5/5 HANG=0/5
  outcome=TOTAL_LOSS code=0 matched=false
  stdout: "" (zero bytes)
  stderr: "" (zero bytes)
  (identical on all 5 runs)
```

**Zero output, on either stream, every single time. Exit code 0.** Not a
truncation (`PARTIAL` would mean some output arrived) — total, complete
loss, indistinguishable from a process that never ran at all if you only
look at the exit code. This is precisely the signature the campaign's own
`PHASE0_GATE_D2_FLUSH_RACE.md` now documents as "Bug B" territory (its
title was corrected today to "TWO causes" for exactly this class of
failure) — RULE 4 in that document states this outright: *"gate the seam
through a real `child_process.spawn` pipe... A green in-process run proves
nothing about the seam — that is exactly how `stdio-check.mjs` passed for
a day while dropping 100% of its payloads."* This rehearsal is that same
lesson landing on `bare-guide.mjs`, today, on the first real attempt to
gate it through the topology that matters.

**Ruled out before reporting, not assumed:**
- **Not a `./data` directory problem.** Pre-created `data/keys/` and
  `data/corestore/bare-guide-room/` in the hostile copy before rerunning
  — identical zero-output result.
- **Not a stdin-write timing race.** Delayed the `stdin.write()` by 200 ms
  after spawn (giving the process time to fully initialize before any
  input arrives) via a direct `node:child_process` script (not the
  harness, to rule out anything harness-specific) — identical result:
  `CODE=0 STDOUT_LEN=0 STDERR_LEN=0`.
- **Not specific to the raw binary invocation.** Reproduced through the
  ACTUAL `.cmd` launcher (§4/§11's file, unmodified), piped via
  PowerShell, both with `ASYMMFLOW_KIT_NONINTERACTIVE` set and unset:
  ```
  switch SET:   exitcode=0, output length=0
  switch UNSET: exitcode=0, output length=2 (only the launcher's own
                trailing "(the kit has stopped...)" line — the guide's
                own output is still completely absent either way)
  ```

### 12d. The switch/stdin interaction (item 3 of the brief) — no conflict found, but inconclusive

With the switch UNSET, the launcher's trailing message printed correctly
after the scripted input was exhausted (the extra line fed for the
`pause` was consumed correctly, no evidence of the switch swallowing an
input the guide needed) — but because the guide itself produced zero
output in every configuration tested, this check could not be exercised
against WORKING guide output, only against the already-broken state. If
§12c's defect is fixed upstream, this specific interaction should be
re-verified against real ceremony output, not assumed clear from this
result alone.

### 12e. Negative control (item 4 of the brief)

Per the brief's own suggestion, the coder's approach (a fixture with the
"Goodbye" line removed) is exactly what `isSuccess()` in this rehearsal's
own driver already does structurally — it requires four specific content
markers including the literal `"Goodbye -- this window is safe to
close."` string, so ANY kit (this real one included, as it turned out)
that fails to produce that line is correctly classified as failing. The
harness's own `selfTest()` (run before every rehearsal in this report,
including this one) is the standing negative-control proof that the
rehearsal mechanism itself can report red — confirmed again here
(PASS, same four-fixture result as every prior run). A dedicated
"Goodbye"-stripped fixture was not built as a SEPARATE artifact because
the real kit already, unintentionally, provided the negative case this
rehearsal needed to prove it can fail correctly.

### 12f. What this means, and what does not follow from it

**Does NOT follow:** that the guide is broken as authored, or that P1-A's
own 14/14 gate was invalid on its own terms — §12b shows the ceremony
logic is completely correct under the topology it was apparently tested
under (unbundled and/or shell-piped). **Does follow:** the guide has not
yet been proven to survive the ONE topology that will actually matter the
moment anyone automates testing it, or the moment it is driven by
anything other than a live interactive console — and given RULE 2's own
text ("never write frames through `bare-process`'s `process.stdout.write()`
... use `console.log()`"), the most likely site to check first is whatever
`getRealStdio().write()` (`bare-bridge.mjs`, unread by this coder, P1A's
file) actually calls under the hood. This is a hypothesis for whoever
routes the fix, not a diagnosis this coder is claiming to have made —
out of fence, not investigated further.

### 12g. Final sealed manifest (guide entry) — the number that actually ships

```
      1.81 MB  app.bundle
     45.14 MB  bare.exe
     11.53 MB  node_modules/**  (18 offloaded native addon prebuilds, sum)
      0.00 MB  portable.flag
      0.00 MB  README_BARE_KIT.txt
      0.00 MB  run_bare_mesh.cmd

total: 23 file(s), 58.9 MB
```

(Full per-file breakdown in §12a.) This is **9 MB larger** than the
`bare-entry.mjs` build (49.8 MB) — the guide's real messenger path pulls
in the full corestore/hypercore storage stack (`rocksdb-native` alone is
7.8 MB) that the simple fold-only demo never touched.

**`kit/dist-bare/` was rebuilt with the default entry (`bare-entry.mjs`)
after this task's testing**, so the artifact currently on disk is the
proven-clean one from §1–§11, not the guide build — the guide build is
reproducible on demand via `--entry=kit/bare-guide.mjs` but is not left as
the checked-in default while §12c's defect is open. `README_BARE_KIT.txt`
(§10a) also still describes the `bare-entry.mjs` fold-proof, not the
guide's ceremony — it needs the revision already flagged in §10a once
this defect is resolved and the guide becomes the real default entry.

### 12h. What is NOT verified (addendum to §9)

1. **Root cause of the total-loss defect** — not diagnosed beyond ruling
   out `./data` and stdin-timing (§12c). The actual mechanism (a
   `process.stdout.write()`-shaped hang/loss per RULE 2, an async-compile
   race per RULE 1, or something specific to `bare-bridge.mjs`'s own I/O
   layer) is unknown and out of this coder's fence to investigate further.
2. **A genuinely live, interactive console** (not a pipe at all — the
   actual human double-click experience) was not tested; every rehearsal
   in this report, including this one, drives stdin through some form of
   pipe. It remains possible (though untested) that the true console path
   works even though every piped path tested does not — or that it fails
   identically. Neither is established.
3. **The `dist/reducer.wasm` asset-offload gap** (§12a) was not exercised
   end-to-end because the total-loss defect (§12c) prevented the
   ceremony from ever reaching a point where its absence would surface
   through observed output — it is a real, separately-reasoned finding,
   not one confirmed by a reproduction of its own failure mode.
4. Everything already listed in §6/§9 — no clean VM, no macOS/Linux, no
   two-machine corridor, no firewall/anchor real mutation (the guide's
   own honest stubs, not this coder's scope).

---

## §12a — GATE CORRECTION: the guide-entry failure is a BUNDLING bug, not the flush race

**Author:** orchestrator (Opus 4.8) · **Date:** 2026-07-20

§12 reports the guide-entry kit producing zero bytes with exit 0 and attributes it to
"Bug B's signature", pointing at `getRealStdio().write()`. **That attribution is wrong**,
and the correct cause is a different class of defect entirely.

### The discriminator §12 did not run

§12 compared *bundled + spawned pipe* (fails) against *unbundled + shell pipe* (works) —
two variables changed at once, so the result cannot isolate either. Measured separately:

| build | topology | result |
|---|---|---|
| **unbundled** | real spawned pipe | **WORKS** — opened, posted, goodbye |
| bundled | real spawned pipe | zero bytes, exit 0 |
| bundled | shell pipe | zero bytes, exit 0 |
| bundled | no stdin at all | zero bytes, exit 0 |

It fails in **every** topology when bundled and succeeds in **every** topology when not.
The variable is **bundling**, not the pipe. §12's "works under a shell pipe" was the
UNBUNDLED case; the bundled shell-pipe run also produces zero bytes.

### Root cause, bisected

Packing progressively smaller entries:

- a bundle importing `host/bare-bridge.mjs` alone → imports fine, 4 exports
- a bundle whose entry prints, then imports `kit/bare-guide.mjs` → prints
  `STEP1 entry reached`, then `STEP2 guide imported`, and **no menu**

The guide imports cleanly; `runGuide()` is never called. `kit/bare-guide.mjs:344-346`:

```js
const argv = typeof Bare !== 'undefined' ? Bare.argv : process.argv
const isMain = argv[1] && new URL(import.meta.url).pathname.replace(…) === argv[1].replace(…)
if (isMain) await runGuide()
```

Inside a bundle `argv[1]` is the **bundle's** path (`…/app.bundle`) while
`import.meta.url` is the **virtual** path within it (`/kit/bare-guide.mjs`). They can never
be equal, so `isMain` is structurally false whenever bundled, `runGuide()` never runs, and
Bare exits 0 on the silent no-op so nothing surfaces.

The guard is *correct* for script invocation — the file's header even records that
`Bare.argv[1]` was checked empirically rather than assumed. It simply cannot hold in packed
form.

### Fix routed

A thin Bare-only entry (`kit/bare-guide-entry.mjs`) importing and calling the already-
exported `runGuide()` — the same shape as `host/bare-entry.mjs`, which packs and runs
correctly today. Explicitly NOT making the guard bundle-aware: "detect whether I am the
bundle's main" is the fragile-assumption class that has produced seven separate wrong
findings in this campaign.

### Why this matters beyond the bug

Every component was green — guide 14/14, bridge 45/45, kit 10/10, probe 15/15 — and the
**composition** still failed silently. Integration is a distinct gate, not a formality
after component gates pass. It earned its place here.

### Method note for the record

Two variables were changed between the working and failing observations, which is what
produced the misattribution. When a green case and a red case differ in more than one
respect, the comparison identifies nothing — vary one axis at a time. This is the same
root failure as the campaign's other six probe errors, in a new costume.

---

## 13. `kit/bare-guide-entry.mjs` lands — re-rehearsed, corrected input sequence

P1A's thin entry (`kit/bare-guide-entry.mjs`, unconditional `await
runGuide()`, no `isMain` guard — the fix for §12's real root cause)
landed during this task. Rehearsed immediately, not held for later, per
the standing instruction not to wait idle.

### 13a. Corrected input sequence

The team lead's own correction: the firewall-offer step consumes ONE
line of input (`io.ask('Press Enter to continue, or type skip...')`) —
missing this in the scripted sequence desyncs every subsequent prompt by
one line, which cost them 20 minutes chasing a phantom defect. Used
`skip` explicitly (unambiguous; empty-Enter also falls through to the
same honest-stub branch per `bare-guide.mjs`'s own code, but `skip` is
the documented choice): `2 / skip / <message> / /exit / 5`.

### 13b. Built, packed, rehearsed — item 1 (isMain fix): CONFIRMED WORKING

```
$ node kit/build-bare-kit.mjs --entry=kit/bare-guide-entry.mjs
bare-pack --host win32-x64 --offload -o kit/dist-bare/app.bundle kit/bare-guide-entry.mjs
total: 23 file(s), 58.9 MB
```

Copied to a fresh hostile directory (no npm tree, no source — confirmed
by listing it), driven through `spawn-pipe-harness.mjs` (self-test PASS
first, as always) with the corrected script, real spawned pipe, 5 runs —
**and separately through the actual `.cmd` launcher via PowerShell**
(avoiding the Git Bash traps):

```
guide ceremony (guide-entry): OK=0/5 PARTIAL=5/5 TOTAL_LOSS=0/5 HANG=0/5
```

Every run now produces the FULL ceremony trail — menu renders, firewall
offer works, messenger opens, a room is created — confirming **the
`isMain` fix genuinely resolves §12's total-silent-loss defect.** Real
output, every time, not zero bytes.

### 13c. Item 4 (the wasm offload bug): STILL OPEN, confirmed by direct evidence, not assumed

`bare-guide-entry.mjs` does not yet call `setWasmSource(import.meta.asset(
'../dist/reducer.wasm'))` (checked directly — grepped both the entry and
`bare-guide.mjs` itself for `setWasmSource`/`import.meta.asset`, neither
appears). **Manifest check confirms no `dist/reducer.wasm` was offloaded**
— exactly §12a's prediction, verified again on the real current build, not
assumed carried over from last time:

```
$ find kit/dist-bare -maxdepth 1 -iname dist
(no match — no dist/ directory at all)
```

The ceremony now runs far enough to hit it directly, live, in hostile
geography — real reproduction, not a hypothesis:

```
> hello from rehearsal
  (not posted -- ENOENT: no such file or directory, open
    "\\?\C:\...\bare-guide-hostile-2\app.bundle\dist\reducer.wasm")
> /exit
====================================
  ASYMMFLOW MESH -- GUIDE (Bare)
====================================
...
> 5

Goodbye -- this window is safe to close.
```

**Worth noting plainly: this is a GRACEFUL failure, not a crash.** The
guide's own `openMessenger` catches the post error and prints `(not
posted -- ...)` rather than throwing — the menu loop continues normally,
`/exit` and `5` work, and the exact `"Goodbye -- this window is safe to
close."` line prints. Confirmed identically through the real `.cmd`
launcher (`exitcode=0 menu=True posted=False goodbye=True`).

### 13d. The rehearsal's own negative control (item 3 of the prior brief) — satisfied by the real broken artifact

Per the instruction ("you have a real broken artifact now rather than a
synthetic one — use it; if your rehearsal would have passed the zero-byte
build, fix the rehearsal"): this build genuinely IS broken (message
posting fails) and the rehearsal's own content assertion —
`/\(posted, seq \d+\)/.test(stdout)` — correctly does NOT match, so every
one of the 5 spawned-pipe runs is classified `PARTIAL`, never `OK`. **The
rehearsal correctly flags the current real build as failing.** No
synthetic fixture was needed; the live defect served as its own negative
control, exactly as instructed.

### 13e. Current overall verdict — does the sealed kit run the FULL ceremony from hostile geography?

**Not yet — one defect down, one still open.** The `isMain` total-silent-
loss defect (§12) is genuinely fixed and reconfirmed here. The
`reducer.wasm` offload gap (§12a's second finding) is still open and now
directly observed blocking the messenger's actual purpose (posting a real
message through the reducer) rather than merely predicted from the
manifest. **The rehearsal itself is ready and staged** — same driver,
same corrected input sequence, same content assertions — to re-run the
instant `bare-guide-entry.mjs` adds the `setWasmSource(import.meta.asset(
'../dist/reducer.wasm'))` call. Not held idle: this section documents a
real, current, reproducible state, not a placeholder.

### 13f. README — NOT updated yet, and why

Item 5 of the prior brief asked for `README_BARE_KIT.txt` to describe the
real ceremony "since it will finally be true." **It is not yet true** —
message posting still fails in the current build (§13c) — so the README
was deliberately left describing the fold-proof (§10a) rather than a
ceremony that doesn't fully work end-to-end yet. Updating it now would
repeat the exact honesty failure this report has avoided throughout:
describing a capability the shipped artifact cannot yet perform. Staged
for the moment §13c actually resolves and re-verifies clean, not before.

### 13g. Cleanup

Restored `kit/dist-bare/` to the default entry (`bare-entry.mjs`,
proven-clean, no open defects) after this rehearsal, consistent with
every prior task in this report — the checked-in build is never the
most-recently-tested diagnostic one. Hostile directory and scratchpad
driver script removed.
