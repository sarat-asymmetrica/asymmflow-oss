# Phase 0 Notes — A (Bare Runtime + bare-* module family)

Coder: P0-A. Scope: Bare runtime itself and the bare-* module family, per the
"Sealed Ship" Bare-runtime campaign brief. Doctrine: zero assumptions, every
claim cited, doc wins over training-vintage impressions. Today is 2026-07-20;
all fetches below are dated to this session.

No code written. This file is the only artifact.

## 1. Environment / versions verified (this session, via WebFetch/WebSearch)

- Bare runtime: **v1.30.3**, released **July 3, 2026** — confirmed via
  github.com/holepunchto/bare README fetch. This matches the version already
  pinned in `mesh/package.json` (`"bare": "^1.30.3"`) and the version the
  2026-07-19 spike actually ran (`npx bare --version` → `v1.30.3`, per
  `BARE_SPIKE_REPORT.md` line 14). Consistent, not a new finding, but worth
  stating: the pinned version and the doc-current version are the same build.
- bare-fs: **v4.7.4**, released **July 7, 2026** — confirmed via
  github.com/holepunchto/bare-fs README fetch. `mesh/package.json` pins
  `"bare-fs": "^4.7.4"` — also current.
- I could not get exact version numbers for bare-process/bare-crypto/
  bare-events/bare-stream from the doc sites in this session (npmjs.com pages
  for those returned HTTP 403 to WebFetch — see §6). `mesh/package.json`'s
  pinned ranges (`bare-process ^4.5.1`, `bare-crypto ^1.15.3`, `bare-events
  ^2.9.1`, `bare-stream ^2.13.3`) are the only version evidence I have for
  those four; I did not independently re-verify them against a live registry
  this session.
- Windows platform tier: confirmed **Tier 1** for both x64 (Windows 10+) and
  arm64 (Windows 11+), with prebuilds available for both — github.com/
  holepunchto/bare README, fetched this session. This directly supports the
  spike's observed zero-build-step install on this machine (Windows 11 Pro
  10.0.26200, x64).

## 2. Findings per scope item

### 2.1 docs.pears.com — Bare guides, API reference, Node differences, module system, "bare" export condition, addons/prebuilds

- Root fetch of `docs.pears.com` (this session) returned a page listing the
  site's structure; the Bare-specific material lives under `/reference/`:
  - `/reference/bare/runtime/` — Bare Runtime API
  - `/reference/bare/cli/` — Bare CLI
  - `/reference/bare/bare-kit/` — Bare Kit (native embedding, out of our scope)
  - `/reference/modules/bare-modules/` — the bare-* module catalog
  - `/explanation/use-bare-standalone/` — "the runtime underneath Pear — on
    its own, as an embeddable, cross-platform JavaScript runtime without the
    peer-to-peer platform"
- A direct fetch of `https://docs.pears.com/guides/getting-started` **404'd**
  this session — that exact path does not exist on the current site. I did
  not find the correct guides-section URL for a Bare-vs-Node conceptual doc
  in the time available; flagging as unverified rather than guessing a path.
- `/reference/modules/bare-modules` (fetched this session) confirms: all
  catalog modules are marked **stable** ("unlikely to change or be removed in
  the foreseeable future"), and the catalog includes a **Node.js
  compatibility mapping table**. From a WebSearch snippet quoting that
  mapping directly: `child_process` → `bare-subprocess`, `fs` → `bare-fs`,
  `os` → `bare-os`, `path` → `bare-path`, `process` → `bare-process`. This is
  the authoritative "which Node core modules need a bare-* package" answer
  for the modules we care about — none of fs/os/path/process/child_process
  are built into the Bare runtime itself; each needs its bare-* package.
- **"bare" export condition** (WebSearch snippet of Bare/Pear docs, this
  session): package.json conditional `exports` can branch on a `"bare"`
  condition for "any Bare environment" and a `"node"` condition for "any
  Node.js environment," falling back to `"default"`. Full condition
  precedence order found: `import` → `require` → `require.asset()` →
  `require.addon()` → `bare` → `node` → platform-specific conditions matching
  `Bare.platform` (e.g. `win32`). This matters directly for the spike's §3
  finding: the Holepunch stack packages (corestore, autobase, hyperswarm,
  etc.) loaded under Bare with **no `"bare"` condition present** in their own
  package.json — meaning they fell through to `"default"`/plain `"main"`,
  and it was Bare's own `require()`/`import` resolver, not a per-package
  `"bare"` export branch, doing the work. I could not verify from the docs
  alone *why* that resolution succeeds (i.e., which of the 5 mapped Node
  builtins those specific packages import at load time) — that would need
  reading each package's actual source, out of scope for this doc pass.
- **Addons/prebuilds model** (github.com/holepunchto/bare README, fetched
  this session): native addons load via the global `Bare.Addon` namespace
  (`Addon.cache`, `Addon.host` — a target-triplet identifier, `Addon.resolve
  (specifier, parentURL)`, `Addon.load(url)`). Prebuilds are controlled by a
  `BARE_PREBUILDS` CMake option, default ON, and exist specifically so
  install doesn't require a local compile toolchain. This matches the
  spike's observed behavior exactly: `npm i -D bare` pulled
  `bare-runtime-win32-x64` — a prebuilt binary — with "no build step, no
  admin rights" (`BARE_SPIKE_REPORT.md` lines 15, 34-35).

### 2.2 github.com/holepunchto/bare — README + API shapes on Windows

Fetched this session. Key points not already covered in §1/§2.1:

- Bare is built on **libjs** (ABI-stable-ish C bindings to V8, "engine
  independent" per its own description) + **libuv** (async I/O). Three
  primary runtime features beyond bare JS execution: (a) a CommonJS/ESM
  module system with bidirectional interop, (b) the native addon system
  above, (c) lightweight `Bare.Thread`s with synchronous `.join()` and
  `SharedArrayBuffer` support.
- `Bare` global namespace confirmed: `Bare.platform` (`win32` on this
  machine), `Bare.arch`, `Bare.argv`, `Bare.pid`, `Bare.exitCode`,
  `Bare.version`, `Bare.versions`; methods `Bare.exit([code])`,
  `Bare.suspend([linger])`, `Bare.wakeup([deadline])`, `Bare.idle()`,
  `Bare.resume()`; events `uncaughtException`, `unhandledRejection`,
  `beforeExit`, `exit`, `suspend`, `wakeup`, `idle`, `resume`.
- No mention of WebAssembly/WASM/WASI anywhere in the README (explicitly
  checked). This is a **negative finding, not a contradiction** of the
  spike: WebAssembly still works (see §2.5) but it is not something Bare's
  own README documents as a feature — it is inherited for free from the
  underlying V8 engine via libjs, not a Bare-authored subsystem. See §2.5 for
  the reasoning chain.

### 2.3 The bare-* modules relevant to us

Per `/reference/modules/bare-modules` catalog (fetched this session) and the
individual bare-fs README fetch:

| Module | One-line (from catalog) | Pinned in mesh/package.json |
|---|---|---|
| bare-fs | "Native file system for JavaScript" | ^4.7.4 (confirmed current) |
| bare-process | "Node.js-compatible process control for Bare" | ^4.5.1 (not independently re-verified this session) |
| bare-stream | "Streaming data for JavaScript" | ^2.13.3 (not independently re-verified this session) |
| bare-os | "Operating system utilities for JavaScript" | not in package.json — not installed |
| bare-path | "Path manipulation library for JavaScript" | not in package.json — not installed |
| bare-subprocess | "Native process spawning for JavaScript" | not in package.json — not installed |
| bare-tcp | "Native TCP sockets for JavaScript" | not in package.json — not installed |
| bare-crypto | "Cryptographic primitives for JavaScript" | ^1.15.3 (not independently re-verified this session) |
| bare-events | "Event emitters for JavaScript" | ^2.9.1 (not independently re-verified this session) |

Correction to my own scope list: `mesh/package.json` (read this session,
line-for-line) does **not** have `bare-os`, `bare-path`, `bare-subprocess`,
or `bare-tcp` installed at all — only `bare`, `bare-crypto`, `bare-events`,
`bare-fs`, `bare-process`, `bare-stream` are devDependencies. If a future
WASI shim or fd-table needs `os`/`path`/`child_process` equivalents, those
four packages are **not yet in the tree** and would need adding.

### 2.4 WASI-for-Bare — re-verification of the 2026-07-19 spike's claim

**The spike's core claim holds: no working WASI preview1 host exists for
Bare today, either built-in or as an installable npm package.** But the
picture is more specific than "404, nothing found" — there is a **named,
structurally-present, but unimplemented placeholder**:

- github.com/holepunchto/bare-node (fetched this session) is a compatibility
  wrapper repo: "the 95% [of Node builtins] that everyone uses." It
  catalogs 50+ Node builtins in three buckets: **Fully Supported** (4:
  `os`, `querystring`, `string_decoder`, `timers`), **Partially Supported**
  (35+, including `fs`, `http`, `crypto`, `stream`, `buffer`, `path`,
  `events`), and **Unsupported/Deprecated** (11, including `wasi` itself,
  explicitly marked 🔴 Unsupported, alongside `async_hooks`, `cluster`,
  `http2`, `sea`, `test`).
- The wrapper pattern for every module in this repo is `npm i bare-[module]
  [module]@npm:bare-node-[module]`. WebSearch found a description of the
  `wasi` wrapper specifically: it "simply exports `require('bare-wasi')` at
  version `*`" — i.e. the wrapper package `bare-node-wasi` exists as
  *plumbing that points at a package named `bare-wasi`*, but that target
  package does not itself exist on the registry.
- Direct verification of that target: `https://www.npmjs.com/package/
  bare-wasi` returned **HTTP 403 to WebFetch** (npm's anti-bot wall, not
  conclusive either way) — but a `WebSearch` for the exact package name
  returned **no matching npm package** among real WASI-related hits (wasi-js,
  @wasmer/wasi, wasi-kernel, wasi, all unrelated), and explicitly stated "I
  cannot find a specific npm package called 'bare-wasi' on npmjs.com." This
  is consistent with, not contradicting, the spike's `npm view bare-wasi` →
  `404 Not Found` transcript (`BARE_SPIKE_REPORT.md` line 87).
- **Net verdict**: bare-wasi is a forward-declared, unimplemented gap in
  Holepunch's own compatibility-module bookkeeping — they know they don't
  have it (it's explicitly listed, not silently missing), but nothing ships.
  No Pear app, no community package, and no built-in Bare WASI support turned
  up in any search this session. The spike's "no official Holepunch WASI
  package exists on npm" claim is **confirmed, with the added texture that
  Holepunch has already reserved the name and marked it red in their own
  tracking table** — a shim we write would not collide with anything in
  flight, and there is no signal of an imminent official one landing either.

### 2.5 Bare's WebAssembly support specifics

- The spike found (transcript, `BARE_SPIKE_REPORT.md` §5) that
  `WebAssembly.compile()` on the real 3.96MB `reducer.wasm` succeeded under
  Bare, correctly identifying its one unsatisfied import namespace
  (`wasi_snapshot_preview1`, 18 import entries / 16 distinct syscalls). That
  transcript is primary evidence and I have no reason to doubt it.
- What the docs add: Bare's own README (§2.2) does not mention WebAssembly
  at all. The explanation is architectural, not a gap: Bare is "built on top
  of libjs, which provides low-level bindings to V8" (github.com/holepunchto/
  bare README + WebSearch corroboration of the same phrase from multiple
  sources), and V8 has a first-class, standards-compliant `WebAssembly`
  global (`WebAssembly.compile`, `.instantiate`, `.Module`, `.Instance`,
  `.Memory`, `.Table`, `.compileStreaming`/`.instantiateStreaming`) as part
  of the JS engine itself, not something a host runtime has to add. Since
  libjs's job is to bind Bare to V8, `WebAssembly` comes through as a
  standard global for free — this is *why* the spike's `WebAssembly typeof:
  object` check passed with zero shimming, and why no Bare doc needs to
  "document" WebAssembly support: it isn't Bare-authored functionality.
- **Caveat I could not resolve**: one search result stated Bare "was built to
  support multiple JavaScript engines — V8, JavaScriptCore, QuickJS... there
  are bindings for v8, jsc, quickjs already." I could not confirm from the
  docs which engine backs the specific `bare-runtime-win32-x64` prebuild the
  spike installed, nor whether QuickJS/JSC builds have equivalent
  `WebAssembly` global support (QuickJS in particular has historically had
  partial/no WASM support in some embeddings elsewhere in the ecosystem —
  general knowledge, not verified against Bare's own docs this session, so
  treat that specific concern as **flagged, not confirmed**). For our
  purposes this is moot as long as the Windows x64 prebuild is V8-backed,
  which the working `WebAssembly.compile` transcript strongly implies, but I
  did not find a doc page stating "the win32-x64 prebuild uses V8" in so
  many words.
- No documented limits on WebAssembly module size, memory, or table found in
  any Bare-specific doc this session (the 3.96MB module compiled fine, but
  that's one data point, not a documented ceiling).

## 3. bare-fs fd-level API surface table

Source: github.com/holepunchto/bare-fs README, fetched this session
(v4.7.4, released 2026-07-07). "Closely follows the Node.js `fs` module
API" per its own description.

| Syscall need (from spike's WASI import list) | bare-fs sync API | Signature (per README fetch) | Windows caveat |
|---|---|---|---|
| fd_read | `fs.readSync` | `readSync(fd, buffer[, offset[, len[, pos]]])` → bytes read | none documented |
| fd_write | `fs.writeSync` | `writeSync(fd, data[, offset[, len[, pos]]])` → bytes written | none documented |
| fd_close | `fs.closeSync` | `closeSync(fd)` | none documented |
| fd_fdstat_get / fd_fdstat_set_flags | `fs.fstatSync` (get); no direct "set flags" analog found | `fstatSync(fd)` → Stats object | none documented for fstat; no README evidence either way on a flags-setter — likely needs to be synthesized from open-time flags tracked by the shim itself, not from a bare-fs primitive |
| fd_prestat_get / fd_prestat_dir_name | no bare-fs equivalent — this is a WASI-specific concept (pre-opened directory handles) with no Node/bare-fs analog | n/a | n/a — must be implemented entirely in shim state, not backed by a bare-fs call |
| (open, for completeness — used internally even though not in the reducer's own import list) | `fs.openSync` | `openSync(filepath[, flags[, mode]])` → fd | none documented |
| (general) | `fs.statSync` | `statSync(filepath)` → Stats object | none documented |

Additional bare-fs surface confirmed present but not on the reducer's 16-item
list (context, in case the shim needs them for prestat bookkeeping or a
later `path_open`-bearing reducer build): `mkdirSync`, `readdirSync`,
`rmdirSync`, `chmodSync`, `utimesSync`, `renameSync`. One explicit Windows
gap found: **the `chown` family (`chown`, `lchown`, `fchown`) is not
implemented on Windows** — irrelevant to our 16-syscall surface (no chown in
the WASI import list) but worth knowing if the shim ever grows.

The README fetch also states bare-fs ships both callback and sync variants,
plus promise-based and stream variants, implemented as "59.4% JavaScript,
40.5% C" — i.e. native code under the hood, not a pure-JS shim over
something else, which matters for perf/behavior parity claims but I did not
independently verify the percentage split (it's a GitHub language-bar stat
surfaced by the fetch, self-reported by the repo, not something I confirmed
by reading source).

**What I could NOT get from the docs**: exact TypeScript-style signatures
with all parameter types (e.g., whether `writeSync`'s `data` accepts a
plain string or requires a Buffer/TypedArray on Bare specifically — Node
allows both; I found no bare-fs-specific confirmation either way this
session). A future coder implementing the actual shim should read
`node_modules/bare-fs/index.js` directly rather than trust this doc-only
pass for exact type signatures.

## 4. WASI-for-Bare search result (consolidated)

**Confirmed: no WASI preview1 host implementation for Bare exists today**,
via three independent angles this session:
1. Direct npm existence check for `bare-wasi` — no package found (§2.4).
2. The one place Holepunch's own tooling references it — the `bare-node-wasi`
   wrapper in `holepunchto/bare-node` — is explicitly tagged 🔴 Unsupported
   and its `require('bare-wasi')` target doesn't resolve (§2.4).
3. No community/Pear-app WASI-for-Bare package surfaced in general search
   (searches returned only Node-side or browser-side WASI packages —
   `wasi-js`, `@wasmer/wasi`, `wasi`, `@bjorn3/browser_wasi_shim`, etc. —
   none Bare-native).

This **corroborates** rather than corrects the 2026-07-19 spike's headline
finding. The one thing this session adds beyond the spike: the gap is
*tracked and named* by Holepunch (not merely absent), which is mild positive
signal that our own shim's namespacing/shape (a `bare-wasi` package) would
land in a slot Holepunch has already reserved conceptually, should they ever
want to adopt or supersede it — worth flagging to the gate as a possible
naming/API-shape consideration, not a blocker either way.

## 5. Corrections to the 2026-07-19 spike report's claims

**None of the spike's factual claims were contradicted.** Everything I could
independently check against current docs (Bare version, bare-fs version,
Windows Tier-1 status, the bare-wasi 404, the "no bare export condition
present in corestore/autobase/hyperswarm" observation) matched. The one
**addition/refinement**, not correction:

- The spike's shim-inventory table (`BARE_SPIKE_REPORT.md` line 167)
  proposes building the fd-level syscalls "on top of `bare-fs`" and
  args/env "from `Bare.argv`/`bare-process`". Docs confirm `bare-process`
  is the correct package for that (catalog: "Node.js-compatible process
  control for Bare"), but I could not confirm from docs whether
  `bare-process` exposes `process.env` reads suitable for `environ_get`
  (Node's `process.env` is a live object; whether Bare's compat layer
  supports enumeration the same way wasn't documented anywhere I could
  find). This isn't a contradiction — just an unverified assumption in the
  spike's own plan that a future implementer should confirm against
  `node_modules/bare-process` source before relying on it.
- The spike's plan also references `fd_fdstat_set_flags` as "near-trivial";
  per §3 above, bare-fs has no direct "set flags on an open fd" primitive
  in its documented API — this would need to be synthesized as shim-local
  state (tracked at `fd_fdstat_get`/open time) rather than backed by a real
  bare-fs call. Flagging this as slightly less trivial than the spike's
  phrasing suggests, though still small in absolute scope.

## 6. What I could NOT verify

- Exact current registry versions of bare-process, bare-crypto, bare-events,
  bare-stream (npmjs.com pages 403'd to WebFetch this session; only
  bare-fs and bare itself got through, both via their GitHub README pages,
  not npm).
- The exact `/guides/...` URL structure on docs.pears.com for a Bare-vs-Node
  conceptual comparison doc — the one path I guessed (`/guides/getting-
  started`) 404'd; I did not find the correct path in the time available.
  The `/reference/` tree (§2.1) is confirmed navigable; `/guides/` and
  `/explanation/` trees are not fully mapped.
- Full bare-fs TypeScript-precise signatures (parameter types, optional-arg
  defaults) — README fetch gave shapes, not exhaustive type detail.
- Whether `fd_fdstat_set_flags`/`fd_prestat_get`/`fd_prestat_dir_name` have
  *any* existing precedent in another Bare-ecosystem package (I did not
  search exhaustively for e.g. a Pear-app-internal WASI experiment beyond
  the general searches in §2.4 and §4 — a narrower search scoped to
  "site:github.com holepunchto" combined with "wasi_snapshot_preview1" was
  not run this session and might turn up more).
- Which JS engine (V8/JSC/QuickJS) backs the specific `bare-runtime-
  win32-x64` prebuild — inferred as V8 from the working WebAssembly
  transcript, not confirmed by a doc statement (§2.5 caveat).
- Whether Bare's WebAssembly global has any Bare-specific size/memory/table
  limits — no doc page found stating a ceiling either way.
