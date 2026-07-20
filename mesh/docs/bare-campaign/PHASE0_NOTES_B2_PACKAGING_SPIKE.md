# Phase 0 Notes — Packaging Feasibility Spike (Coder B, addendum)

**Scope:** retire the "never actually ran bare-pack" risk flagged as #1 in
`PHASE0_NOTES_B_PACKAGING.md` §5. All work done in the scratchpad
(`p0b-pack/`), not the repo tree, per the brief. Every claim below is a real
transcript from this session — no doc-inference. **Go/no-go verdict up front:
bare-pack IS the sealing mechanism, but its DEFAULT invocation is broken for
this codebase in two separate, load-bearing ways. Both have a working
workaround verified end-to-end in hostile geography. See §0.**

---

## 0. Verdict summary

| # | Unknown | Verdict | Working recipe |
|---|---|---|---|
| 1 | Does bare-pack run at all | **YES** | `bare-pack --host win32-x64 -o x.bundle entry.mjs` |
| 2 | Survives hostile geography | **YES** (for the raw `.bundle` format) | copy `bare.exe` + `.bundle` anywhere, run |
| 3 | Native addons (sodium-native/udx-native/etc.) | **DEFAULT EMBEDDING IS BROKEN.** `--offload-addons` WORKS, verified in hostile geo. | `bare-pack --offload-addons ...` |
| 4a | `import.meta.url` self-location inside a bundle | **RESOLVES TO A VIRTUAL PATH INSIDE THE BUNDLE, NOT A REAL DIRECTORY.** Every one of the 23 `fileURLToPath(import.meta.url)` call sites in mesh/host + mesh/kit will break if bundled as-is. | must switch to bundle-aware asset loading (§4) |
| 4b | Loading `reducer.wasm` (a non-JS 3.96MB binary) | Default embedding produces an unreadable virtual path (`bare-fs.readFileSync` can't read inside a `.bundle` file). **`--offload`/`--offload-assets` WORKS**, verified in hostile geo. | `bare-pack --offload ...` + `import.meta.asset()` |
| 4c | `node:fs`/`node:url`/`node:path` (Node-style specifiers, what `apply.mjs` actually uses) | **CORRECTED by the gate — see §8/§4 note below.** The claim "neither `node:fs` nor plain `fs` resolves" was a probe artifact (`bare -e` evaluates as ESM, where `require` doesn't exist, so it fails for every specifier including good ones — no negative control was run). Re-measured with real `.mjs` files + `await import()`: `node:fs`/`node:path`/`node:url`/`node:net` DO alias onto `bare-fs`/etc when run from inside `node_modules` context; `node:crypto`/`node:os`/`node:readline` do not. In hostile geography (no `node_modules`) ALL of them fail, `node:fs` and `bare-fs` alike — Bare has no true builtins, `node:fs` is a resolved alias, not a polyfill. Full re-measurement: `mesh/docs/bare-campaign/PHASE0_GATE_B2_RESOLUTION.md`. **The recommended action is unchanged**: no `node:` specifier in anything destined for the sealed artifact — rewrite to `bare-fs`/`bare-path`/`bare-url`/`bare-crypto`/`bare-os` — because the alias table is undocumented internal policy, doesn't cover the full Node surface (crypto/os/readline have no alias), and pack-time behavior (§8) makes the difference concrete regardless of the runtime alias question. | source-level import rewrite required (now for determinism + full-surface-coverage reasons, not because `node:fs` is unconditionally broken) |
| 5 | Double-click `.cmd` launcher, CRLF, no PATH | **YES, works end-to-end.** | see §5, CRLF-verified, `%~dp0`-relative |
| 6 | Sealed folder size | ~48 MB total (bare.exe dominates at 45.1 MB) for a corestore+hypercore+hyperswarm+wasm payload | see §6 |

**Bottom line for the campaign:** the sealed-folder mechanism from
`PHASE0_NOTES_B_PACKAGING.md` §1 is CORRECT in shape (bare.exe + a
bare-pack output + a small self-contained side-tree, zero PATH/npm
dependency) but WRONG in the specific flags assumed. The right invocation is
`--offload` (both addons and assets), not default embedding, and every
`node:fs`/`node:path`/`node:url` import in the mesh codebase needs to become
`bare-fs`/`bare-path`/`bare-url` before Phase 2/3 bundling — this is a real
source change, not a packaging-flag change, and is the single biggest
Phase-1/2 risk this spike surfaces.

---

## 1. Unknown #1 — does bare-pack run at all

Scratchpad setup: `npm init -y && npm i -D bare-pack@2.2.1 bare@1.30.3` in
`p0b-pack/` (fresh dir, no relation to `mesh/`). `bare@1.30.3` pulled in
`bare-runtime-win32-x64@1.30.3` automatically, giving `bare.exe` at
`node_modules/bare-runtime-win32-x64/bin/bare.exe`.

```
$ ./node_modules/.bin/bare-pack --host win32-x64 -o hello.bundle.cjs hello.mjs
(exit 0, 914 bytes written)
```

**Correction to Phase 0's central assumption:** the `.cjs`/`.mjs` wrapper
formats are **not self-executing**. Inspecting the output:

```js
module.exports = "336\n{\"version\":0,...}\nconsole.log('hello...')..."
```

This is `bin.js`'s own source (read directly,
`node_modules/bare-pack/bin.js` lines 100-116): the `.cjs`/`.mjs`/`.json`
"formats" literally wrap the raw bundle bytes as a **string literal** inside
a `module.exports`/`export default` statement — a data carrier for some other
loader to `require()` and mount, not something `bare <file>` can run
directly. Running it does nothing (exits 0, no output) because it just
assigns a string and exits.

**The format that works is raw `--format bundle` (or just an output path with
no `.cjs`/`.mjs`/`.json` suffix, e.g. `-o hello.bundle`):**

```
$ ./node_modules/.bin/bare-pack --host win32-x64 -o hello.bundle hello.mjs
$ ./node_modules/bare-runtime-win32-x64/bin/bare.exe hello.bundle
hello from bare bundle
argv: [ '...bare.exe', '...hello.bundle' ]
```

`bare.exe` natively recognizes and mounts `.bundle` files as an entry point —
confirmed by the runtime's own module system (`Module._extensions..bundle`
appears in every subsequent stack trace in this report). **This corrects
`PHASE0_NOTES_B_PACKAGING.md` §1: the sealed artifact's second file must be a
raw `.bundle`, not a `.cjs` "wrapper."**

---

## 2. Unknown #2 — hostile geography (no bundling)

```
$ mkdir /c/.../hostile-geo-test-1
$ cp node_modules/bare-runtime-win32-x64/bin/bare.exe  hostile-geo-test-1/
$ cp hello.bundle                                       hostile-geo-test-1/
$ cd hostile-geo-test-1 && ls
bare.exe  hello.bundle          # nothing else — no node_modules, no package.json
$ ./bare.exe hello.bundle
hello from bare bundle
argv: [
  'C:\\...\\hostile-geo-test-1\\bare.exe',
  'C:\\...\\hostile-geo-test-1\\hello.bundle'
]
exit=0
```

**PASS.** `bare.exe` (45,142,016 bytes) + a `.bundle` file is a genuinely
portable two-file unit for JS-only code with no addons/assets. This is the
FR-1 disease class the whole campaign targets, and it does not reproduce here.

---

## 3. Unknown #3 — native addons (the real test)

Installed the actual Holepunch stack into the scratchpad:
`npm i corestore@^7.4.0 hypercore@^11.10.0 hyperswarm@^4.11.0
sodium-universal@^5.0.1`.

Wrote `addon-test.mjs` (kept in scratchpad, not repo) that: generates real
random bytes and a real hash via `sodium.randombytes_buf`/
`sodium.crypto_generichash` (exercising `sodium-native`'s actual native
call, not just import resolution), then opens a real `Corestore`, gets a
`Hypercore`, `append()`s a block, and `get(0)`s it back — a genuine
storage-engine round trip. Verified it passes under plain Node first (sanity
check, `node addon-test.mjs` → `ADDON_TEST_PASS`).

### 3a. Default bare-pack invocation (addons embedded) — FAILS

```
$ bare-pack --host win32-x64 -o addon-test.bundle addon-test.mjs
(exit 0, produced a 10,917,880-byte bundle — addons WERE embedded as bytes)

$ ./node_modules/bare-runtime-win32-x64/bin/bare.exe addon-test.bundle
Uncaught Error: The specified module could not be found.
    at Addon.load (bare:/bare.js:3570:16)
    at require.addon (bare:/bare.bundle/node_modules/bare-module/index.js:822:30)
    at .../addon-test.bundle/node_modules/bare-type/binding.js:1:26
    ...
exit=127
```

This reproduces **identically in the dev directory** (not just hostile
geography) — it is not a portability bug, it is a default-embedding bug.
Isolated the cause: `bare-type` (an internal dependency, unrelated to our
own addons) loads fine **unbundled** (`bare.exe -e "require('bare-type')"` →
`[function type]`, exit 0), so the failure is specific to loading a native
addon **that was embedded as bytes inside a `.bundle` file** — Windows'
"specified module could not be found" is `LoadLibrary`-speak, consistent
with Bare's `Addon.load` failing to extract/load the embedded binary from
inside the bundle. Root cause not fully diagnosed (out of scope for a
feasibility spike) — reported as a real, reproducible defect in bare-pack
2.2.1 / bare 1.30.3's default addon-embedding path on Windows, not worked
around.

### 3b. `--offload-addons` — WORKS

```
$ bare-pack --host win32-x64 --offload-addons -o offload-out/addon-test.bundle addon-test.mjs
(exit 0)
$ find offload-out -type f
offload-out/addon-test.bundle
offload-out/node_modules/bare-fs/prebuilds/win32-x64/bare-fs.bare
offload-out/node_modules/bare-inspect/prebuilds/win32-x64/bare-inspect.bare
offload-out/node_modules/bare-path/prebuilds/win32-x64/bare-path.bare
offload-out/node_modules/bare-type/prebuilds/win32-x64/bare-type.bare
offload-out/node_modules/bare-url/prebuilds/win32-x64/bare-url.bare
offload-out/node_modules/fs-native-extensions/prebuilds/win32-x64/fs-native-extensions.bare
offload-out/node_modules/quickbit-native/prebuilds/win32-x64/quickbit-native.bare
offload-out/node_modules/rocksdb-native/prebuilds/win32-x64/rocksdb-native.bare
offload-out/node_modules/simdle-native/prebuilds/win32-x64/simdle-native.bare
offload-out/node_modules/sodium-native/prebuilds/win32-x64/sodium-native.bare

$ cd offload-out && ../node_modules/bare-runtime-win32-x64/bin/bare.exe addon-test.bundle
sodium randombytes_buf OK, bytes: 32 nonzero: true
sodium crypto_generichash OK: 68d3a24748aa0797...
hypercore ready, discoveryKey: 9fb2c1edb67116b6...
hypercore append+get roundtrip OK: hello from a real hypercore append under bare
ADDON_TEST_PASS
exit=0
```

**Confirmed this list is the real, complete transitive native-addon set** —
correcting `PHASE0_NOTES_B_PACKAGING.md` §4's uncertainty about
`quickbit-native`: it IS present (pulled in via `hypercore-storage`'s
`rocksdb-native`/`quickbit-native` storage layer in hypercore v11), and
`rocksdb-native` and `simdle-native` and `fs-native-extensions` are three
*more* native addons the Phase 0 notes did not name at all (transitive
through `corestore`'s storage stack, not obvious from the top-level 11
packages the C4 spike enumerated).

### 3c. Hostile geography, offloaded variant — PASS

```
$ cp -r offload-out/. hostile-geo-test-3/
$ cp bare.exe hostile-geo-test-3/
$ cd hostile-geo-test-3 && ./bare.exe addon-test.bundle
sodium randombytes_buf OK, bytes: 32 nonzero: true
sodium crypto_generichash OK: 68d3a24748aa0797...
hypercore ready, discoveryKey: 7c1bbf9dbf43f245...
hypercore append+get roundtrip OK: hello from a real hypercore append under bare
ADDON_TEST_PASS
exit=0
```

(First attempt in hostile geo threw `Invalid device file, was moved
unsafely` from `rocksdb-native`'s own device-file safety check — that was
caused by *my* test copying a stale `addon-test-storage/` directory left
over from an earlier run into the hostile dir; it is rocksdb-native
correctly detecting relocated storage, not a bare-pack bug. Deleting the
stale storage and re-running with fresh storage passed cleanly, as shown
above.)

**Verdict: native addon embedding must use `--offload-addons` (or
`--offload`). Default embedding is broken and must not be used.** The
resulting artifact is not literally "one bundle file" for addon-bearing code
— it is the bundle plus a small, purely-relative `node_modules/<pkg>/
prebuilds/<host>/*.bare` tree that ships alongside it, entirely
self-contained (no reference to any system npm tree, no PATH lookup). This
still satisfies FR-1/D5: nothing is resolved outside the shipped folder.

---

## 4. Unknown #4 — self-location and the reducer.wasm asset (widest blast radius)

Read `mesh/host/apply.mjs` lines 16-22 directly (read-only) to replicate its
*exact* pattern:

```js
import { readFileSync, ... } from 'node:fs'
import { fileURLToPath } from 'node:url'
const __dirname = dirname(fileURLToPath(import.meta.url))
const WASM_PATH = join(__dirname, '..', 'dist', 'reducer.wasm')
```

### 4a. `node:fs`/`node:url`/`node:path` do not exist under Bare at all

> **CORRECTION (post-gate):** the runtime claim in this subsection — "neither
> `node:fs` nor plain `fs` resolves to anything under Bare, zero candidates" —
> was measured with a flawed probe (`bare.exe -e "require(...)"`, which
> evaluates as ESM where `require` doesn't exist at all, so it fails
> identically for every specifier including working ones; no negative
> control was run to catch this). The gate re-measured with real `.mjs`
> files and `await import()`: `node:fs`/`node:path`/`node:url`/`node:net`
> **do** alias onto `bare-fs`/etc when `node_modules` is present, though
> `node:crypto`/`node:os`/`node:readline` do not. In hostile geography (no
> `node_modules`) all of them fail alike, `node:fs` and `bare-fs` included —
> Bare has no true builtins; `node:fs` is a resolved alias from disk, not a
> polyfill. Full transcript: `PHASE0_GATE_B2_RESOLUTION.md`. **The pack-time
> failure quoted immediately below is unaffected by this correction** — it
> was reproduced again with a genuine negative control in §8, and stands.
> The recommended fix (rewrite to `bare-fs`/`bare-path`/`bare-url`/etc.) is
> unchanged; see the corrected row in §0.


First bare-pack attempt using this exact code failed to even **pack**:

```
Bail: ModuleTraverseError: MODULE_NOT_FOUND: Cannot find module 'node:url'
  imported from '...selfloc-test.mjs'
```

Declaring `node:fs`/`fs`/etc. via `--builtins` made packing *succeed* but
the resulting bundle failed at **runtime**:

```
Uncaught TypeError: Cannot read properties of null (reading 'fs')
```

Isolated definitively with a direct, unbundled runtime check (not a
bare-pack question at all — a Bare-runtime question):

```
$ ./bare.exe -e "require('node:fs')"
Uncaught ModuleError: MODULE_NOT_FOUND ... candidates: []

$ ./bare.exe -e "require('fs')"
Uncaught ModuleError: MODULE_NOT_FOUND ... candidates: []
```

**Neither `node:fs` nor plain `fs` resolves to anything under Bare — zero
candidates, not "resolves to the wrong thing."** This means every one of
the (per the team lead's own count) 23 `fileURLToPath(import.meta.url)`
call sites across `mesh/host/**` and `mesh/kit/**` — and any other file
using `node:fs`/`node:path`/`node:url`/etc. — **will fail to even pack**,
let alone run, under Bare as currently written. This is not a packaging
nuance; it is a source-level porting requirement.

**The fix, confirmed working:** import the real npm packages by name —
`bare-fs`, `bare-path`, `bare-url` (already transitive dependencies of the
Holepunch stack, confirmed present in every bundle built this session):

```js
import { readFileSync } from 'bare-fs'
import { fileURLToPath } from 'bare-url'
import { dirname, join } from 'bare-path'
```

```
$ ./bare.exe selfloc-test.mjs        # UNBUNDLED, direct file run
import.meta.url raw: file:///.../p0b-pack/selfloc-test.mjs
__dirname resolved: C:\...\p0b-pack
WASM_PATH: C:\...\p0b-pack\reducer.wasm
readFileSync via self-location OK, bytes: 3963665
WebAssembly.Module compile OK
SELFLOC_TEST_DONE
exit=0
```

This mirrors the C4 spike's own devDependency list (`bare-fs`, `bare-crypto`,
`bare-events`, `bare-process`, `bare-stream`) — those packages were not
random probes, they are the *mandatory replacements* for Node's `node:*`
built-ins under Bare. Phase 0's framing that "Bare implements enough of
Node's core module surface... natively" (`BARE_SPIKE_REPORT.md` line 72-73)
is about `require()`-ing *third-party* Holepunch packages that internally
already use `bare-fs`/etc — it does **not** mean `node:fs`-style imports in
**our own code** resolve. This is worth flagging back to the C4 spike's
framing as a clarification, not a contradiction.

### 4b. `import.meta.url` inside a bundle resolves to a VIRTUAL path — confirmed hazard

Bundled the corrected (`bare-fs`/`bare-path`/`bare-url`) version with
`--offload-addons` and ran it **without** `reducer.wasm` sitting next to the
bundle (the file existed only in the original dev directory that produced
the bundle, not copied into the run location):

```
$ ./bare.exe selfloc-test.bundle
import.meta.url raw: file:///.../selfloc-test.bundle/selfloc-test.mjs
__dirname resolved: C:\...\selfloc-test.bundle
WASM_PATH: C:\...\selfloc-test.bundle\reducer.wasm
SELF_LOCATION_READ_FAILED: ENOENT: no such file or directory, open
  "\\?\C:\...\selfloc-test.bundle\reducer.wasm"
ASSET_URL_READ_FAILED: ENOENT: ... (same path)
exit=0   (test script caught the error itself — a real app would crash here)
```

**This is the finding the team lead flagged as widest blast radius, and it
is real:** inside a bundle, `import.meta.url` for the entry module resolves
to `file:///<...>/<bundlename>.bundle/<original-relative-path>` — i.e.
`dirname(fileURLToPath(import.meta.url))` becomes a path **inside the
bundle's own virtual mount namespace**, which is not a real directory on
disk (the bundle is a single file). `apply.mjs`'s exact pattern —
`join(__dirname, '..', 'dist', 'reducer.wasm')` — will silently resolve to
a nonexistent path once `apply.mjs` itself is bundled, and fail with ENOENT
at runtime, not at pack time. **Every path-relative asset load anywhere in
the 23 flagged call sites has this exact failure mode once its containing
file is the bundle's entry point or a bundled module trying to reach a
non-bundled sibling file.**

### 4c. The correct asset-loading API — `import.meta.asset()`, plus `--offload`

Traced the fix through source, not guesswork: `bare-module-lexer`
(`node_modules/bare-module-lexer/index.js` line 38) documents the real
mechanism verbatim: *"CommonJS `require.asset()` if `REQUIRE` is set, or ES
module `import.meta.asset()` if `IMPORT` is set."* This is a **static,
lexer-recognized syntax** bare-pack's traversal specifically watches for
(confirmed in `bare-module-traverse/index.js` — a dynamic `new URL('./x',
import.meta.url)` pattern is NOT detected as an asset; the bundle header's
`"assets":[]` stayed empty when I tried that pattern first — only the
literal `import.meta.asset('./reducer.wasm')` call form works).

```js
const wasmPath = import.meta.asset('./reducer.wasm')   // returns a URL string
const wasmBuf = readFileSync(new URL(wasmPath))          // bare-fs needs a URL, not a bare string
```

Default (embedded) bundling of this **does** embed the 3.96 MB wasm into
the bundle (confirmed: bundle size grew from ~2 KB to 4,170,804 bytes) but
`import.meta.asset()`'s returned path still points *inside* the bundle's
virtual namespace, and `bare-fs.readFileSync` cannot read from inside a
`.bundle` file directly — same ENOENT failure as §4b, now for an asset that
IS embedded but not consumable via plain `fs` calls. **Not resolved this
session** whether there's a bundle-aware read API (`bare-bundle`'s own
`.read(key)` method looks like the right candidate, per its `index.d.ts`)
that would make true embedding work — ran out of scope/time to chase it
(see §7).

**The workaround that DOES work, verified in hostile geography:**
`--offload` (both `--offload-addons` and `--offload-assets`) writes
`reducer.wasm` out as a **real sibling file** next to the bundle, and
`import.meta.asset()` then returns a real, readable path:

```
$ bare-pack --host win32-x64 --offload -o asset-out4/asset-test2.bundle asset-import-test2.mjs
$ find asset-out4 -type f
asset-out4/asset-test2.bundle
asset-out4/node_modules/bare-fs/prebuilds/win32-x64/bare-fs.bare
asset-out4/node_modules/bare-path/prebuilds/win32-x64/bare-path.bare
asset-out4/node_modules/bare-url/prebuilds/win32-x64/bare-url.bare
asset-out4/reducer.wasm

$ cp -r asset-out4/. hostile-geo-test-5/  &&  cp bare.exe hostile-geo-test-5/
$ cd hostile-geo-test-5 && ./bare.exe asset-test2.bundle
asset path: file:///.../hostile-geo-test-5/reducer.wasm typeof: string
read bytes: 3963665
WebAssembly.Module compile from import.meta.asset: OK
ASSET_IMPORT_TEST2_DONE
exit=0
```

**PASS, in true hostile geography.** The sealed folder for anything touching
`reducer.wasm` is therefore: `bare.exe` + `<name>.bundle` + `reducer.wasm`
(a real sibling file, referenced via `import.meta.asset()`, not a bare
`new URL(..., import.meta.url)` guess) + the `node_modules/<pkg>/prebuilds/
<host>/*.bare` addon tree from §3. Still zero PATH lookup, zero reliance on
any system npm tree — everything is relative to the folder.

---

## 5. Unknown #5 — the double-click launcher

Wrote `run_mesh.cmd` with explicit CRLF line endings (the lesson documented
in `mesh/kit/build-kit.mjs` lines 38-40: *"the repo's .gitattributes LF
baseline means any .cmd that ever passes through git... every .cmd is
CRLF-normalized at write/copy time"*):

```bat
@echo off
cd /d "%~dp0"
"%~dp0bare.exe" "%~dp0app.bundle"
```

Verified CRLF with `file`:
```
$ file run_mesh.cmd
run_mesh.cmd: DOS batch file, ASCII text, with CRLF line terminators
```

`%~dp0` is the launcher's own directory (a batch built-in, no PATH lookup,
no assumption about the caller's CWD) — same pattern `build-kit.mjs` already
uses for `node.exe` (`%~dp0node.exe`), ported directly to `bare.exe`.

End-to-end test in hostile geography (`hostile-geo-test-6/`: `app.bundle` +
`bare.exe` + `reducer.wasm` + the addon prebuilds tree + `run_mesh.cmd`,
invoked via PowerShell's `cmd.exe /c .\run_mesh.cmd` to simulate a
double-click without an interactive session):

```
asset path: file:///.../hostile-geo-test-6/reducer.wasm typeof: string
read bytes: 3963665
WebAssembly.Module compile from import.meta.asset: OK
ASSET_IMPORT_TEST2_DONE
EXITCODE=0
```

**PASS.** No console-window weirdness, no PATH dependency, no working-
directory assumption beyond the launcher's own folder.

---

## 6. Unknown #6 — sealed folder size

| Component | Size |
|---|---|
| `bare.exe` (bare-runtime-win32-x64@1.30.3) | 45,142,016 bytes (45.1 MB) — dominates the total |
| A JS-only bundle (`hello.bundle`) | 779 bytes |
| Corestore+hypercore+hyperswarm+sodium bundle, addons embedded (broken) | 10,917,880 bytes |
| Same, with `--offload-addons` (bundle only) | 1,149,380 bytes |
| Addon prebuilds tree (`node_modules/*/prebuilds/win32-x64/*.bare`, 10 files) | ~9.3 MB combined (`rocksdb-native.bare` alone is 7.8 MB — the single largest addon) |
| `reducer.wasm` (offloaded as a sibling file) | 3,963,665 bytes (3.96 MB, unchanged from source) |
| **Full sealed folder** (`bare.exe` + bundle + wasm + addon tree + launcher, hostile-geo-test-6) | **~48 MB total** |

`bare.exe` itself is roughly 4x the size of everything else combined. This
is the fixed cost of shipping the runtime; it does not grow with the mesh's
own code.

---

## 7. What is still NOT verified

1. **Root cause of the default-embedding addon failure (§3a)** — reproduced
   and worked around (`--offload-addons`), but *why* `Addon.load` fails to
   extract/load a bundle-embedded addon on Windows was not diagnosed. If
   Phase 3 wants embedded (not offloaded) addons for a stricter "fewer
   visible files" shape, this needs real investigation, possibly a
   holepunchto issue report (per the give-freely ethos already established
   for this campaign).
2. **Whether embedded (non-offloaded) assets are consumable via some other
   API** (§4c) — `bare-bundle`'s own `Bundle.read(key)`/`.mount()` methods
   look like plausible candidates for reading an embedded asset's bytes
   directly from the mounted bundle, but this was not tried. Not needed for
   Phase 3 since `--offload` already works end-to-end, but worth a footnote
   if a stricter single-file goal resurfaces later.
3. **The exact scope of the `node:fs`/`node:path`/`node:url` → `bare-fs`/
   `bare-path`/`bare-url` rewrite** across the real mesh codebase — this
   spike confirmed the *pattern* and the *fix* on a synthetic replica of
   `apply.mjs`'s exact lines, but did not enumerate or touch any real
   `mesh/host/**`/`mesh/kit/**` file (out of scope: "do not touch mesh
   source," per the brief). The team lead's count of 23
   `fileURLToPath(import.meta.url)` sites is the right starting point for
   Phase 2's actual porting work.
4. **`bare-link`/ahead-of-time linking** was not exercised this session —
   `--offload-addons` sidesteps the need for it entirely on this platform,
   so its Windows fit (flagged as unclear in the prior notes doc) remains
   unverified but is now lower-priority since a working alternative exists.
5. **`quickbit-native`, `rocksdb-native`, `simdle-native`, and
   `fs-native-extensions`** — confirmed present and confirmed to load/work
   via `--offload-addons`, but their *individual* native calls were not
   directly exercised (the round-trip test exercises them transitively
   through `corestore`'s storage layer, which is a real but indirect
   check).
6. **macOS/Linux hosts** — everything in this spike targeted `win32-x64`
   only, per the campaign's Windows-first scope. Not attempted for other
   hosts.
7. Did not touch any file under `mesh/` other than reading
   `mesh/host/apply.mjs` (read-only, to replicate its exact pattern) and
   copying `mesh/dist/reducer.wasm` (read-only copy into the scratchpad).
   No mesh source was modified, per the brief's constraint.

---

## 9. `apply-bare.mjs`'s dynamic-import ternary — confirmed broken, condition-map fix verified

**Trigger:** `mesh/host/apply-bare.mjs` (P1-A's file, read-only, not edited)
selects its fs module at runtime via
`const fsMod = isBare ? await import('bare-fs') : await import('node:fs')`.
The team lead's hypothesis: `bare-pack`'s traverser is static and (per §8)
treats `node:fs` identically to a nonexistent package, so this file — never
run through `bare-pack` before, only run directly — would be a Phase 3
blocker hiding in code that currently looks green.

### 9a. Does the static traverser follow a dynamic `await import()` inside a ternary?

Copied `apply-bare.mjs` and `wasi-preview1-lite.mjs` unmodified into
`p0b-pack/apply-bare-test/host/`, with `mesh/dist/reducer.wasm` copied to a
matching relative `dist/` sibling (read-only copies; neither original was
edited). Sanity-checked first: a minimal `entry.mjs` calling
`applyViaWasm([])` runs correctly under both plain Node and unbundled
`bare.exe` (both branches of the ternary work fine un-packed — confirms
these two files are functionally sound; this spike is purely about
packaging).

```
$ bare-pack --host win32-x64 --offload -o packed-out/entry.bundle entry.mjs
Bail: ModuleTraverseError: MODULE_NOT_FOUND: Cannot find module 'node:fs'
  imported from '.../host/apply-bare.mjs'
bare-pack exit=1
```

**Confirmed: the traverser DOES walk into the dynamic `await import()` inside
the ternary and resolves both branches statically**, exactly the same
`node:fs` failure mode as §8's negative-control-backed finding. This is
actually the **better** of the two possible outcomes the team lead named —
a traverser that ignored dynamic imports would have packed successfully and
then thrown at runtime the first time the `node:fs` branch was reached
(never, since `isBare` is always true inside the sealed artifact — a latent
landmine). Instead it fails loud, at build time, before a bundle is ever
produced. Still a real Phase 3 blocker as the file is currently written:
`apply-bare.mjs` cannot be packed today.

### 9b. The `imports` condition-map fix — VERIFIED end-to-end in hostile geography

Built a scratchpad copy, `apply-bare-condmap.mjs`, replacing the ternary with
a static import of a subpath key:

```js
import * as fsMod from '#fs'
```

alongside a `package.json` in the same scratchpad root:

```json
{
  "name": "condmap-test",
  "type": "module",
  "imports": {
    "#fs": {
      "bare": "bare-fs",
      "default": "fs"
    }
  }
}
```

**One correction found along the way:** the team lead's sketch used
`"default": "node:fs"`. Under plain Node this throws
`ERR_INVALID_PACKAGE_TARGET: Invalid "imports" target "node:fs"` —
Node rejects a `node:`-prefixed target string in a package `imports` map on
this Node version (v22.17.0). Changing the default target to the legacy
bare specifier `"fs"` (no `node:` prefix) resolves cleanly under Node. Use
`"default": "fs"` in the real recipe, not `"node:fs"`.

Pack-time result:

```
$ bare-pack --host win32-x64 --offload -o packed-out/entry.bundle entry.mjs
bare-pack exit=0
```

**Clean pack, no `MODULE_NOT_FOUND`.** Inspected the bundle header to confirm
*which* branch was actually resolved — not just that packing succeeded:

```
"#fs":"/../node_modules/bare-fs/index.js"
```

and confirmed the `default`/`fs` branch was never even touched — no `fs`
resolution or embedded `fs.js` anywhere in the header. **The `bare` condition
is selected and the `default` branch is discarded entirely at pack time**,
which is exactly the property needed: one source file, dual-runtime by
construction, packable.

Combined this fix with the already-verified `import.meta.asset()` pattern
from §4c for the reducer.wasm self-location hazard (the realistic composed
Phase 2 recipe, not a second untested mechanism), and ran the full chain in
**true hostile geography**:

```
$ cp -r packed-out/. hostile-geo-test-7/  &&  cp bare.exe hostile-geo-test-7/
$ cd hostile-geo-test-7 && find . -type f
./bare.exe
./dist/reducer.wasm
./entry.bundle
./node_modules/bare-fs/prebuilds/win32-x64/bare-fs.bare
./node_modules/bare-path/prebuilds/win32-x64/bare-path.bare
./node_modules/bare-url/prebuilds/win32-x64/bare-url.bare

$ ./bare.exe entry.bundle
applyViaWasm ran, result type: object
CONDMAP_ENTRY_DONE
exit=0
```

**PASS.** Real `bare-fs` resolution via the condition map, real
`import.meta.asset()`-located `reducer.wasm` read, real WASI instantiation,
real `applyViaWasm()` call, zero PATH/npm-tree dependency — not just a
clean pack exit code.

### 9c. The copy-pasteable recipe

**`package.json`** (proposed addition — NOT applied; the team lead will
sequence this edit):

```json
{
  "imports": {
    "#fs": { "bare": "bare-fs", "default": "fs" }
  }
}
```

**Consumer file** (e.g. `apply-bare.mjs`), replace the ternary with:

```js
import * as fs from '#fs'
// use fs.readFileSync(...) etc. exactly as before — same API shape on both
// bare-fs and Node's fs for the calls this file makes.
```

For any file that also needs to locate a sibling binary asset by its own
position (the `reducer.wasm` case), pair this with the already-verified
`import.meta.asset('./relative/path')` + `--offload`/`--offload-assets`
recipe from `PHASE0_NOTES_B2_PACKAGING_SPIKE.md` §4c — the two fixes are
independent and compose cleanly, as demonstrated above.

**Generalizes beyond `#fs`:** the same `imports` map can carry `#path`,
`#url`, `#crypto`, `#os`, etc. subpath keys for every other `node:`-prefixed
module a dual-runtime file needs, each with its own `{bare: ..., default:
...}` pair (`bare-path`/`path`, `bare-url`/`url`, `bare-crypto`/`crypto`,
`bare-os`/`os`) — one `package.json` stanza serves every dual-runtime file
in Phase 2, not just this one.

**When NOT to use this pattern:** `wasi-preview1-lite.mjs`'s existing
approach — zero imports, all I/O injected by the caller — remains strictly
better where it applies (no condition map needed at all, packs trivially,
one shim body serves both runtimes with no static-resolution surface). The
condition-map pattern is for files that must directly touch a runtime
primitive with no possible injection seam (like `apply-bare.mjs`'s need to
`readFileSync` the wasm module itself) — use dependency injection first,
condition maps only when injection isn't available.

## 8. Pack-time resolution of `node:`-prefixed specifiers (post-gate follow-up)

**Question:** does `bare-pack` resolve `node:`-prefixed specifiers at pack
time — separate from the runtime-alias question the gate corrected in §4a's
note — and is that a hard blocker or just hygiene? Includes a genuine
negative control this time (an import of a package name that cannot possibly
exist), so a real resolution failure is distinguishable from a probe
artifact.

Three tiny entry files, packed from inside `p0b-pack/` (where `node_modules`
is present, i.e. the *best case* for resolution):

```js
// pt-nodefs.mjs
import { readFileSync } from 'node:fs'

// pt-barefs.mjs
import { readFileSync } from 'bare-fs'

// pt-nonexistent.mjs  (negative control — this package cannot exist)
import { thing } from 'this-package-definitely-does-not-exist-xyz123'
```

```
$ bare-pack --host win32-x64 -o pt-nodefs.bundle pt-nodefs.mjs
Bail: ModuleTraverseError: MODULE_NOT_FOUND: Cannot find module 'node:fs'
  imported from '...pt-nodefs.mjs'
pack exit=1

$ bare-pack --host win32-x64 -o pt-barefs.bundle pt-barefs.mjs
pack exit=0

$ bare-pack --host win32-x64 -o pt-nonexistent.bundle pt-nonexistent.mjs
Bail: ModuleTraverseError: MODULE_NOT_FOUND: Cannot find module
  'this-package-definitely-does-not-exist-xyz123' imported from
  '...pt-nonexistent.mjs'
pack exit=1
```

**`node:fs`'s pack-time failure is byte-for-byte the same error shape,
`ModuleTraverseError: MODULE_NOT_FOUND`, same exit code, as the genuine
negative control.** `bare-pack`'s static traverser cannot tell `node:fs`
apart from a package that does not exist on npm at all — it has no special
case for `node:`-prefixed specifiers, unlike the Bare *runtime* (§4a's
correction), which does resolve some of them via an alias table when
`node_modules` is present. Completed the round trip on the one that packed:

```
$ bare-pack --host win32-x64 --offload-addons -o pt-barefs-out/pt-barefs.bundle pt-barefs.mjs
pack exit=0
$ cd pt-barefs-out && bare.exe pt-barefs.bundle
bare-fs OK, typeof readFileSync: function
run exit=0
```

**Verdict: this is a hard blocker at pack time, not merely hygiene.**
Whatever the Bare runtime's alias table does or doesn't cover (§4a), it is
irrelevant to Phase 2/3 — `bare-pack` itself refuses to traverse past a
`node:`-prefixed import and the build fails before a bundle is even
produced. Every `node:fs`/`node:path`/`node:url`/`node:crypto`/`node:os`/
etc. specifier in any file that will be reachable from the sealed artifact's
entry point **must** be rewritten to its `bare-*` equivalent before that file
can be bundled at all. This confirms the team lead's binding rule (no
`node:` specifier in anything destined for the sealed artifact) is not just
good practice — it is required for `bare-pack` to run.
