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

### 5d. The actual double-click artifact, not just the underlying binary

`runSpawnPipe` proves the `bare.exe app.bundle` pipeline; it does not by
itself prove the `.cmd` a human actually double-clicks behaves the same,
because `run_bare_mesh.cmd` ends in `pause >nul`, which blocks forever
waiting for a keystroke — a real difference from a fire-and-forget script.
Verified this separately, invoking the launcher the way a human would
(via `cmd.exe`, with a keypress fed to satisfy the trailing `pause`):

```
$ "x" | cmd.exe /c ".\run_bare_mesh.cmd"
BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410

(the kit has stopped - press any key to close this window)
EXITCODE=0
```

The real launcher, in hostile geography, produces the correct digest and
the expected closing prompt.

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
2. `bare-guide.mjs` as the entry — does not exist yet; this builder's
   `--entry=` parameter is designed for it but has not been run against it.
3. The `devDependencies` alphabetical-reorder side effect noted in §7 —
   not reverted; flagged for the team lead's call.
4. Whether `kit/dist-bare/`'s existing `.gitignore` entry
   (`mesh/.gitignore` already lists `kit/dist-bare/` — confirmed present
   before this task started) was added by this coder or pre-existed from
   anticipation of this work; not investigated further, not load-bearing
   either way since the effect (build output never committed) is correct
   regardless of who added it.
