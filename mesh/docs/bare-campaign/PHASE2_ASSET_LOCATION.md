# Phase 2 — Asset-location fix via dependency injection (Coder P0-B)

**Question:** does a real fold run from a sealed hostile directory?
**Answer: YES.**

---

## 0. The fix, in shape

`apply-bare.mjs` still doesn't know anything about how it is packaged. All
packaging knowledge (`import.meta.asset()`, a bare-pack/Bare-only lexer
feature with no Node equivalent) concentrates in ONE new Bare-only file,
`mesh/host/bare-entry.mjs`, which injects the wasm bytes into
`apply-bare.mjs` via a new optional `setWasmSource()` — dependency
injection, not a runtime branch, per the campaign's stated preference
(`PHASE0_GATE_B3_CONDITION_MAP.md`: "prefer dependency injection where it
applies... condition maps only where a file must touch a runtime primitive
directly"). This mirrors `wasi-preview1-lite.mjs`'s own zero-import,
injected-I/O shape.

Files touched: `mesh/host/apply-bare.mjs` (added `setWasmSource()`, the
default self-locating path is byte-for-byte unchanged as the fallback) and
the new `mesh/host/bare-entry.mjs` (Bare-only, never runs under Node — by
design and confirmed empirically, §3). Nothing else in the fenced set was
touched; `bare-bridge.mjs`/`bare-bridge-spike.mjs` (P1A-wasi-shim's,
untracked in the tree during this session) were not read or touched.

---

## 1. `apply-bare.mjs`'s new DI surface

```js
let _wasmSourceOverride = null

export function setWasmSource(bytes) {
  _wasmSourceOverride = bytes
  _module = null   // force recompile on next apply if already cached
}

function loadModule() {
  if (_module) return _module
  const bytes = _wasmSourceOverride ?? fsMod.readFileSync(WASM_URL)
  _module = new WebAssembly.Module(bytes)   // sync form ONLY — see §4
  return _module
}
```

`setWasmSource()` is never called by anything except `bare-entry.mjs`. Every
existing consumer — `mesh-node.mjs` via `#apply`, `bare-parity-spike.mjs`
under both runtimes, `reactor-parity-spike.mjs`, every Node-line spike —
keeps using the unmodified self-locating default with **zero call-site
changes**. This is why §2's gates reproduce identical digests to before the
change: the default code path is byte-identical to what it replaced, only
additive.

## 2. Gates — unaffected by the DI addition (byte-identical to prior runs)

```
$ node host/bare-parity-spike.mjs       → 13/13, BARE PARITY SPIKE GREEN
$ npx bare host/bare-parity-spike.mjs   → 13/13, BARE PARITY SPIKE GREEN
$ node host/smoke.mjs                   → SMOKE GREEN ✅  digest: 6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410  (unchanged)
$ node host/reactor-parity-spike.mjs    → 13/13, REACTOR PARITY GREEN ✅
$ npm run invitespike       → INVITE SPIKE GREEN ✅       state digest a3d679e2...  (unchanged)
$ npm run roomspike         → ROOM SPIKE GREEN ✅         state digest 74523010...  (unchanged)
$ npm run socialspike       → SOCIAL SPIKE GREEN ✅       opsHashed=11 applied=7... (unchanged)
$ npm run attachspike       → ATTACH SPIKE GREEN ✅       state digest ec23eb92...  (unchanged)
$ npm run transcriptspike   → TRANSCRIPT SPIKE GREEN ✅   bundle sha256 62a501f4... (unchanged)
$ npm run reissuespike      → REISSUE SPIKE GREEN ✅      successor digest 7da5d00c... (unchanged)
$ npm run missionc          → MISSION C GREEN ✅          state digest 79432ed8...  (unchanged)
```

Every digest above is identical, character-for-character, to the pre-change
runs recorded in `PHASE2_IMPORT_MIGRATION.md`. The DI addition changed
nothing observable on any existing path.

## 3. `bare-entry.mjs` — the Bare-only concentration point

```js
import * as fs from 'bare-fs'                         // direct import is
import { applyViaWasm, setWasmSource } from './apply-bare.mjs'   // correct HERE ONLY —
                                                                    // this file is Bare-only
const wasmAssetPath = import.meta.asset('../dist/reducer.wasm')
setWasmSource(fs.readFileSync(new URL(wasmAssetPath)))

const OPS = [ /* the exact canonical scenario smoke.mjs uses */ ]
const state = applyViaWasm(OPS)
console.log(state.digest === EXPECTED_DIGEST
  ? 'BARE_ENTRY_FOLD_OK digest=' + state.digest
  : 'BARE_ENTRY_FOLD_MISMATCH got=' + state.digest + ' want=' + EXPECTED_DIGEST)
```

**Confirmed Bare-only by construction, not just convention** — ran it under
Node directly to check the boundary actually holds:

```
$ node host/bare-entry.mjs
ReferenceError: Bare is not defined
    at .../node_modules/bare-path/index.js:5   (bare-fs's own internal use of the Bare global)
```

It fails immediately and loudly if anyone ever tries to run it under Node —
the runtime itself enforces the boundary, not just a comment.

Sanity-checked unbundled under Bare in the real tree first:

```
$ npx bare host/bare-entry.mjs
BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410
```

Matches `smoke.mjs`'s own digest for the identical `OPS` set exactly — the
real reducer, the real WASI shim, real injected wasm bytes, real fold.

## 4. `WebAssembly.compile()` compliance (`PHASE0_GATE_D2_FLUSH_RACE.md`)

Read the binding rule before writing anything, as instructed. `apply-
bare.mjs`'s `loadModule()` already used `new WebAssembly.Module()`
(synchronous) before this change and still does — the DI addition only
changed *where the bytes come from*, not how they're compiled.
`bare-entry.mjs` never calls any `WebAssembly.*` API directly (that stays
inside `apply-bare.mjs`), so there is no new async-compile surface to
comply with. §5's 5-repetition hostile-geography run is also an empirical
check against the flush-race: zero truncation across 5 runs.

## 5. THE PROOF — real fold, sealed artifact, true hostile geography

```
$ bare-pack --host win32-x64 --offload -o bare-entry-pack/entry.bundle host/bare-entry.mjs
pack exit=0

$ find bare-entry-pack -type f
bare-entry-pack/dist/reducer.wasm
bare-entry-pack/entry.bundle
bare-entry-pack/node_modules/bare-fs/prebuilds/win32-x64/bare-fs.bare
bare-entry-pack/node_modules/bare-path/prebuilds/win32-x64/bare-path.bare
bare-entry-pack/node_modules/bare-url/prebuilds/win32-x64/bare-url.bare
```

Copied to a from-scratch directory (`hostile-geo-test-9/`) with **no npm
tree, no source files, nothing but the artifact and a copy of `bare.exe`**:

```
$ find hostile-geo-test-9 -type f
hostile-geo-test-9/bare.exe
hostile-geo-test-9/dist/reducer.wasm
hostile-geo-test-9/entry.bundle
hostile-geo-test-9/node_modules/bare-fs/prebuilds/win32-x64/bare-fs.bare
hostile-geo-test-9/node_modules/bare-path/prebuilds/win32-x64/bare-path.bare
hostile-geo-test-9/node_modules/bare-url/prebuilds/win32-x64/bare-url.bare

$ cd hostile-geo-test-9 && ./bare.exe entry.bundle
BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410
exit=0
```

**Asserted on printed CONTENT, not exit code** — per the explicit
instruction (Bare has already been shown twice today to exit 0 on failure).
The output string was grep-matched against the exact expected digest, and
the assertion is real: had the fold failed, thrown, or produced wrong
output, the printed line would read `BARE_ENTRY_FOLD_MISMATCH` or the
process would produce no matching line at all — there is no path that
prints the success string without the digest actually being correct.

**Repeated 5 times** to rule out the flush-race class of failure
specifically (the concern that motivated the binding rule in §4):

```
run 1: BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c...
run 2: BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c...
run 3: BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c...
run 4: BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c...
run 5: BARE_ENTRY_FOLD_OK digest=6c8c35eff1e2c...
```

**5/5, zero flakiness, identical digest every time.**

---

## 6. What is NOT verified

1. Only `win32-x64`, only this machine, consistent with every prior gate in
   this campaign.
2. `bare-entry.mjs` is a standalone demonstration/proof file, not wired into
   any build script or the sealed-kit assembly process — that wiring is
   Phase 3's job (packaging the actual field kit), not this task's.
3. Did not test the interaction of this recipe with `mesh-node.mjs`/the
   full Bare bridge (`bare-bridge.mjs`, P1A-wasi-shim's file, not read) —
   only the `apply-bare.mjs` boundary directly, which is what this task
   scoped.
4. The `git status`/`git diff` commands in this session showed the source
   edits as already matching `HEAD` (some auto-commit process appears
   active in this shared team environment, evidenced by commits
   `9c8827e`/`061fb0f` from other activity) — this coder did not run `git
   commit` and has no visibility into what committed what; content on disk
   was verified directly via `grep`, not inferred from git state.
