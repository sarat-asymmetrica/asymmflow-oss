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
