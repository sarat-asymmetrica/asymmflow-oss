# Phase 0 Notes — Packaging (Coder B)

**Scope:** the load-bearing question of the campaign — how does a Bare app ship to a
Windows machine as one sealed artifact with all JS and native prebuilds embedded, no
module resolution outside the artifact, no PATH lookup, no separate `node_modules`.

**Method:** every claim below is either a doc quote (URL cited), an `npm view`
transcript run today (2026-07-20), or a live `npx bare --help` transcript. Where a
page 404'd or a claim could not be independently confirmed, it is marked so in §6.
No prior training-data assumption about these tools is used un-cited.

---

## 1. Direct answer

**There is no single-`.exe` mechanism. The achievable sealed artifact is a two-file
sealed folder: the unmodified `bare.exe` prebuild binary, sitting next to ONE
`app.bundle.cjs` file produced by `bare-pack`, invoked as `bare.exe app.bundle.cjs`.**

- `bare.exe` comes from the `bare-runtime-win32-x64` npm package (verified
  `1.30.3`, `npm view bare-runtime-win32-x64` — see §2 table) — a single prebuilt
  binary, `bin: bare`, 46.6 MB unpacked, no companion DLLs listed in its package
  metadata. This is the runtime; it is copied as-is, not rebuilt.
- `app.bundle.cjs` comes from running `bare-pack --host win32-x64 -o app.bundle.cjs
  <entry.mjs>` (flags verified against the live README, §2). By default `bare-pack`
  **embeds** addon and asset imports into the bundle rather than referencing files on
  disk — this is the mechanism that eliminates `node_modules` and PATH lookup. The
  `.cjs`/`.mjs` "wrapper" output format is real, self-mounting JavaScript (not a
  binary blob), so `bare app.bundle.cjs` is a normal `bare <filename>` invocation per
  Bare's own CLI contract (confirmed live, §2).
- Command to run: `bare.exe app.bundle.cjs [...args]` — no `bare-pack`, no npm, no
  Node needed on the target machine. This is the "extract a zip, double-click one
  thing" shape once wrapped in a `.bat`/`.cmd`/shortcut that invokes `bare.exe
  app.bundle.cjs`.

**Why not one true `.exe`:** the only Holepunch-documented path to a literal single
compiled executable is `pear-appling` (§3), and it is a *bootstrap-only* binary that
fetches your actual app code from the P2P swarm on first run over the network — the
opposite of what a sealed, offline field artifact needs (see §3 for the exact
evidence). No other tool in the family (`bare-pack`, `bare-link`, `bare-make`,
`bare-bundle`) compiles JS + runtime into one native binary; `bare-make`/`bare-link`
operate on native *addons*, not on producing an app executable.

---

## 2. Tooling inventory

| Tool | npm version (verified 2026-07-20) | What it produces | Windows support | Evidence |
|---|---|---|---|---|
| `bare-pack` | `2.2.1` (published ~1 week ago) | Traverses the module graph from an entry file and emits a `bare-bundle` artifact in one of 4 forms: raw `.bundle`, `.bundle.cjs`, `.bundle.mjs`, `.bundle.json`. Embeds addon/asset imports by default; `--offload` externalizes them to disk instead; `--linked` resolves addons to `linked:` specifiers (for ahead-of-time-linked addons, mainly mobile). | Yes — `--host win32-x64` targets Windows explicitly; with `--linked` on Windows, addons resolve to `linked:addon-<version>.dll`. | `npm view bare-pack` (deps: bare-bundle ^1.8.3, bare-fs, bare-module-traverse, bare-path, paparam, promaphore); README fetched via WebFetch (github.com/holepunchto/bare-pack#readme) — flags `--out/-o`, `--base`, `--offload`, `--linked`, `--host`, `--format/-f`, `--encoding/-e` all confirmed present. |
| `bare-bundle` | `1.10.0` | Not a CLI tool — the **archive format** `bare-pack` writes into, "inspired by [electron/asar](https://github.com/electron/asar)". Structure: optional hashbang, header-length int, JSON header (version, entry point, import map, byte offsets/lengths), then embedded files. Header has explicit `addons` and `assets` arrays. | N/A (format, not a platform-specific tool) | `npm view bare-bundle`; README fetched (github.com/holepunchto/bare-bundle). |
| `bare-runtime-win32-x64` | `1.30.3` (published 2 weeks ago) | The actual Windows x64 `bare.exe` prebuilt binary (`bin: bare`). 46.6 MB unpacked. One npm dependency: `require-asset@^1.0.2`. | Is the Windows prebuild itself. | `npm view bare-runtime-win32-x64` — full metadata pulled; matches C4's spike report which resolved this exact package (`bare-runtime-win32-x64`) with zero build step on Windows 11 (`BARE_SPIKE_REPORT.md` line 13). |
| `bare-link` | `3.3.0` | **Native-addon linker**, not an app/executable linker. Ahead-of-time links native addon prebuilds for specific `--host` targets (e.g. `darwin-arm64`, `ios-arm64`) so `bare-pack --linked` can reference them as `linked:` specifiers instead of embedding raw files. Has a programmatic API (`require('bare-link')`) and CLI. Supports Windows code-signing flags (`--subject`, `--subject-name`, `--thumbprint`). | Partial — signing flags exist for Windows, but its stated primary use case in the README/search evidence is mobile (where ahead-of-time linking is "essential"), not desktop. | `npm view bare-link`; README fetched (github.com/holepunchto/bare-link#readme). |
| `bare-make` | `1.8.0` | An **opinionated build-system generator on top of CMake** ("generates build files for Ninja using Clang... across all supported systems") — for compiling native addons (or Pear applings, §3) from source. Not involved unless you're building a native addon from scratch rather than using a prebuild. | Documented generically as cross-platform via CMake/Ninja/Clang; README fetched had no Windows-specific caveats visible. | `npm view bare-make`; README fetched (github.com/holepunchto/bare-make#readme). Also the exact tool `pear-appling`'s build instructions invoke (`npm i -g bare-make && bare-make generate && bare-make build`) — confirmed via the `distribute-as-binary` how-to page, §3. |
| `bare-addon` | — | **Does not exist on npm.** `npm view bare-addon` → `404 Not Found`. Not part of the real toolchain; do not reference it. | — | `npm view bare-addon` transcript, this session. |
| `bare-kit` | — (npm 404 for package name `bare-kit`; GitHub repo exists) | GitHub repo `holepunchto/bare-kit` — "Bare for native application development," a web-worker-style API for host apps to spawn managed Bare "worklets" with IPC bindings. **Not a packager.** Explicitly scoped to **iOS and Android only** in its README; a `win32/` folder exists in the repo tree but no Windows docs/examples were surfaced. Not usable for this campaign's desktop target. | Repo has a `win32/` folder but no documented Windows story. | `npm view bare-kit` → 404; README fetched via WebFetch (github.com/holepunchto/bare-kit#readme). |
| `pear` (CLI) | `3.0.0` | The full Pear P2P runtime/CLI (`pear stage`, `pear release`, `pear run`) — a different application model, see §3. Not itself a Windows-exe packager for Bare-as-library apps. | Yes, it's the primary supported OS target across the docs. | `npm view pear`. |

---

## 3. Bare-as-library vs Pear-as-app-model — framed as what the end user must do

This is the stop-and-ask trigger #3 in the campaign doc (§4.3 of
`FABLE_CAMPAIGN_BARE_RUNTIME.md`) if the sealed-artifact mechanism required adopting
full Pear. **It does not** — the sealed-folder answer in §1 uses Bare as a library we
ship ourselves, not the Pear app model. Evidence for why the two are genuinely
different, not just cosmetically:

### Bare-as-library (what we're using)
- Quote (docs.pears.com, `/explanation/use-bare-standalone/`, fetched today): *"Bare
  is a general-purpose JavaScript runtime in its own right, and you can adopt it on
  its own, with no peer-to-peer or Pear machinery at all."* And explicitly: *"A single
  standalone executable. Compile a CLI, service, or daemon into one binary with no
  Node.js, Bare, or Pear install required on the user's machine."* — but the same
  page, when asked for the concrete packaging path, did **not** surface a documented
  literal single-binary-compile command; it only pointed at `bare-pack`/bundling
  how-tos (which is the bundle+runtime-binary sealed-folder path in §1, not a true
  single compiled exe). Treat the "single standalone executable" language as
  aspirational/marketing framing not yet backed by a turnkey CLI command for the
  bundle+runtime desktop case — see §6.
- **End-user requirement: none.** They receive our folder (`bare.exe` +
  `app.bundle.cjs` + a launcher), no install step, no network fetch, no account, no
  P2P key. This matches D3 ("clients are never handed a command line") and the FR-1
  requirement (no missing-module class of failure) once wrapped in a double-clickable
  launcher.

### Pear-as-app-model (`pear stage` / `pear release` / `pear run`)
- Pear's own framing (WebSearch of docs.pears.com, fetched today): *"Pear is an
  installable peer-to-peer runtime, development, and deployment platform to build,
  share, and extend unstoppable, zero-infrastructure P2P apps."* `pear stage <channel>`
  derives an application key from the channel + package.json; `pear release` runs a
  stage → provision → multisig → release pipeline; `pear run <link>` loads the
  released version at runtime — i.e. apps are addressed by **P2P key**, not by a file
  you hand someone.
- The one literal single-executable artifact Pear's docs describe is **`pear-appling`**
  (`docs.pears.com/how-to/operate-an-app/build-and-package/distribute-as-binary/`,
  fetched today): *"one small bootstrap binary (typically a few MB)... does not
  require end users to have Pear or Node installed."* Built via `git clone
  pear-appling`, edit `CMakeLists.txt` (verified: the real template requires an `ID
  "<z32-key>"` field — a P2P application key — plus `NAME`, `VERSION`, platform
  signing fields; fetched raw from `raw.githubusercontent.com/holepunchto/pear-
  appling/main/CMakeLists.txt` today), then `npm i -g bare-make && npm i && bare-make
  generate && bare-make build`.
- **Critical distinction, stated plainly by the fetched page itself:** *"The binary is
  bootstrap-only: it doesn't ship the application's code or assets, only the
  references needed to fetch them. On first launch, it downloads the Pear platform
  and your application from the swarm."* — i.e. **`pear-appling`'s single-exe result
  requires network access to the Holepunch swarm on first run**, and the `ID` field is
  a P2P key, confirming this is fundamentally a *fetch-from-network* app model, not an
  embed-everything-offline model.
- **End-user requirement for Pear apps generally (not appling):** install the `pear`
  CLI/runtime (from npm or an installer), then either `pear run pear://<key>` or run a
  staged/released app by key — i.e., they need *something* installed and a working
  network path to the swarm (or a seeded peer) at least once.

### The verdict this campaign needs
Full Pear (`pear run`/`pear://` links) is the wrong app model for a sealed, offline,
zero-network field artifact — it is architected around P2P-key addressing and
over-the-air fetch, which is a feature for *update distribution* but a liability for
*first-contact sealed delivery* to a machine with no prior network path. `pear-appling`
inherits that same network-fetch dependency even as a single exe. Bare-as-library (§1)
is therefore the correct choice already implied by the campaign's D3/FR-1 constraints,
and this is not a Question-3 stop-and-ask — it resolves without owner input.

---

## 4. Native prebuild embedding story

- `bare-pack` **embeds native addon binaries into the bundle by default** — this is
  explicit in the fetched README ("By default, addons and assets are embedded into
  the bundle"). `--offload`/`--offload-addons` is the opt-out (writes them to disk
  next to the bundle instead — NOT what we want for a sealed single-file JS layer).
  `--linked` (paired with `bare-link`) is the ahead-of-time-linking alternative,
  described as "essential for mobile platforms" — not required for our Windows
  desktop target, where default embedding should suffice.
- **Confirmed real native-addon deps in the mesh dependency tree** (traced via `npm
  view <pkg> dependencies`, live today, not guessed):
  - `sodium-universal` (`^5.0.1`) ← depended on by `corestore`, `hypercore`,
    `hyperdht`, `@hyperswarm/secret-stream`, `dht-rpc` — itself depends on
    **`sodium-native`** (native libsodium bindings), confirmed `npm view sodium-
    universal dependencies` → `{ 'sodium-native': '^5.0.1' }`.
  - **`udx-native`** (`^1.5.3`) ← depended on directly by `dht-rpc` (which underlies
    `hyperdht`/`hyperswarm`) — confirmed `npm view dht-rpc dependencies`.
  - `quickbit-native` and `b4a` were named in the brief as candidates; `b4a` (buffer
    utility, `^1.8.1` latest) is a plain JS-with-optional-native-fastpath dependency
    present throughout the tree (corestore, hyperdht, dht-rpc, secret-stream all
    depend on it) but is not confirmed here to ship a native addon — not verified
    either way this session (see §6). `quickbit-native` was not found as a direct or
    one-hop transitive dependency of any of the 11 packages the spike report
    enumerated; not chased further (see §6).
  - This matches — and gives a concrete name to — the C4 spike's blanket "all 11
    packages import clean under Bare, no shims" result (`BARE_SPIKE_REPORT.md`
    §3): resolution succeeding under `bare require-check.mjs` for these packages is
    only possible if Bare-compatible prebuilds for `sodium-native`/`udx-native`
    already exist and were what `require()` resolved to on Windows. The spike
    confirmed *import* succeeds; it did not exercise a native call, so full runtime
    correctness of those Bare prebuilds is still pending Phase 1/2 spikes, not this
    packaging question.
- **Practical implication for Phase 3 (the sealed kit):** `bare-pack --host win32-x64
  -o app.bundle.cjs <entry>` run against the mesh's real entry point should, by
  default behavior, embed the `sodium-native`/`udx-native` Windows prebuild binaries
  directly into `app.bundle.cjs` as bytes (per the bundle format's `addons` array +
  base64/hex encoding option), needing no `--linked`/`bare-link` step for this
  target. This should be verified with an actual `bare-pack` run against
  `mesh/host/*.mjs` in Phase 3 — not run this session (Phase 0 is docs/notes only,
  no code, per the brief).

---

## 5. Risks / unknowns

1. **No literal single-`.exe`.** The sealed artifact is a folder with (at minimum)
   `bare.exe` + `app.bundle.cjs`. If "one file" is a hard requirement rather than
   "one sealed folder," the only documented route to a literal single binary
   (`pear-appling`) carries a mandatory first-run network fetch from the P2P swarm —
   incompatible with the sealed/offline requirement (§3). This should be flagged to
   the campaign gate explicitly: §1's two-file folder is the recommended target,
   not a single exe.
2. **`bare app.bundle.cjs` end-to-end has not been run.** The README-derived claim
   that the `.cjs` wrapper is self-mounting, addon-embedding JS that `bare
   <filename>` can execute directly is inferred from the bundle format spec + the
   bare-pack README's described output formats, not from an actual `bare-pack` build
   + `bare` execution transcript. This is squarely a Phase 1/2 spike task, not
   verified here.
3. **Native addon prebuild availability for Bare on win32-x64 specifically for
   `sodium-native` and `udx-native`** was inferred from the spike's successful
   `require()` resolution, not from inspecting the actual prebuild binaries or
   confirming a Bare ABI-tagged prebuild exists in those packages' npm tarballs.
   Should be checked directly (e.g. `npm view sodium-native` → look for a `prebuilds/`
   file list with a `bare` or matching ABI tag) before Phase 3 build-out.
4. **`bare-link`'s Windows fit is unclear.** Its README emphasizes mobile as the
   "essential" use case; Windows support is evidenced only via code-signing flags,
   not via a demonstrated desktop `--linked` addon workflow. If default embedding
   (§4) turns out insufficient for any Windows-specific addon shape, `bare-link`'s
   actual desktop behavior needs a real test, not just a README read.
5. **`bare-make`'s role is a fallback, not a requirement** — only needed if a native
   addon must be compiled from source (no prebuild available) or if a future
   decision embraces `pear-appling`-style native compilation despite §3's network
   caveat. Not needed for the sealed-folder plan in §1 as currently scoped.

---

## 6. What could NOT be verified this session

- `docs.pears.com/getting-started/` and `docs.pears.com/guides/releasing-a-pear-app`
  both returned HTTP 404 when fetched directly by URL guess; the content summarized
  in §3 for `pear stage`/`pear release`/`pear run` came from WebSearch result
  snippets (attributed to docs.pears.com by the search engine), not a direct page
  fetch — treat as lower-confidence than the directly-fetched pages (which are cited
  inline above).
- `docs.pears.com/pear-runtime/faq` fetched successfully but contained no populated
  FAQ entries in the fetched excerpt (page appeared to be a redirect/index stub) —
  could not confirm or deny additional standalone-distribution FAQ content beyond
  the "Distribute as a binary" link it surfaced (which was then fetched directly).
- The "Bundle a Bare app" how-to page referenced by `/explanation/use-bare-
  standalone/`'s link list could not be located at a working URL this session (guessed
  path 404'd; WebSearch surfaced only the mobile-specific
  `guides/making-a-bare-mobile-app` variant, which documents `bare-pack`/`--linked`
  for iOS+Android, not Windows desktop specifically — used as supporting evidence for
  `bare-pack`'s general behavior in §1/§2, but not a Windows-specific walkthrough).
- Did not run an actual `bare-pack` build against any real entry file this session
  (Phase 0 is notes-only per the brief) — the sealed-folder mechanism in §1 is
  therefore a well-evidenced design, not yet a proven transcript. That proof belongs
  to Phase 1/2/3.
- `quickbit-native`'s presence (or absence) in the mesh's real dependency graph was
  not exhaustively traced (only one hop below the 11 packages named in the spike
  report was checked) — flagged in §4, not resolved.
- Did not inspect `mesh/node_modules/bare-runtime-win32-x64` locally as the brief
  suggested — `mesh/node_modules` does not exist in this checkout (dependencies are
  not currently installed; confirmed via `ls`). All `bare-runtime-win32-x64` facts in
  §2 come from `npm view`, not local package inspection.
- `mesh/kit/build-kit.mjs` (922 lines) and `mesh/docs/FABLE_CAMPAIGN_BARE_RUNTIME.md`
  were read for context per the brief; the Node-kit pruning logic in `build-kit.mjs`
  was read but not deeply analyzed line-by-line — sufficient to confirm it is the
  Node-line predecessor being retired (per the campaign doc's charter), not to
  extract further packaging lessons from it. If Phase 3 wants specific pruning
  patterns ported from it, that needs a dedicated re-read.
