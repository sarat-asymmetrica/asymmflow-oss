// build-bare-kit.mjs — Phase 3: assembles the SEALED Bare kit, the successor
// to build-kit.mjs's Node kit. Same charter (campaign §0): a folder a client
// unzips and double-clicks, whose failure modes CANNOT include missing-
// module resolution, PATH lookups, or a separate node_modules.
//
// THE KEY DIFFERENCE FROM build-kit.mjs: that builder hand-walks its own
// import graph and its own package.json dependency graph (real walks, not
// guesses — see its own header) because Node has no packaging tool that does
// this for it. Bare does: `bare-pack` performs the REAL static module
// resolution and embeds/offloads exactly what the entry point's import graph
// actually reaches — this builder does NOT re-implement that walk, it hands
// the entry point to bare-pack and trusts its resolution, per the verified
// recipe (PHASE0_NOTES_B2_PACKAGING_SPIKE.md, PHASE2_ASSET_LOCATION.md).
// There is no hand-pruning here to get wrong.
//
// Sealed-folder shape (the verified recipe, run for real in this file):
//   bare.exe                          — the unmodified prebuilt runtime
//   app.bundle                        — bare-pack --host win32-x64 --offload
//   dist/reducer.wasm                 — offloaded asset (real sibling file,
//                                        NOT embedded — PHASE0_NOTES_B2 §4b's
//                                        import.meta.url-inside-a-bundle
//                                        landmine is why: an embedded asset
//                                        resolves to a virtual path bare-fs
//                                        can't read; --offload writes a real
//                                        file next to the bundle instead)
//   node_modules/<pkg>/prebuilds/win32-x64/<pkg>.bare  — offloaded native
//                                        addons (bare-fs/bare-path/bare-url;
//                                        default EMBEDDING is broken on this
//                                        bare-pack/bare version — §3a of the
//                                        same doc — --offload is not optional)
//   run_bare_mesh.cmd                 — CRLF, %~dp0-relative, no PATH lookup
//   portable.flag                     — presence-alone marker, DP1 convention
//
// ENTRY POINT is a parameter, not a hardcode (--entry=<path relative to mesh
// root, or absolute>). Defaults to host/bare-entry.mjs — the file this
// coder already proved end-to-end in hostile geography (PHASE2_ASSET_
// LOCATION.md). P1A-wasi-shim's client-facing mesh/kit/bare-guide.mjs is the
// intended eventual entry; this builder does not block on it — swapping it
// in is `--entry=kit/bare-guide.mjs` once that file lands, no code change
// here. See PHASE3_KIT_REPORT.md for which entry this was actually proved
// against in this run.
//
// WebAssembly.compile()/instantiate() (async forms) are BANNED anywhere in
// the reachable graph (PHASE0_GATE_D2_FLUSH_RACE.md — silently drops ~33%
// of stdout under Bare, exit code 0). This builder does not call either; the
// entry points it packs (apply-bare.mjs transitively) use the sync
// `new WebAssembly.Module()`/`new WebAssembly.Instance()` forms only —
// verified again in this run's rehearsal (§ below), not merely asserted.
//
// Run: npm run buildbarekit  (equivalently: node kit/build-bare-kit.mjs)

import { existsSync, mkdirSync, writeFileSync, cpSync, rmSync, readdirSync, statSync } from 'node:fs'
import { join, dirname, resolve, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'

// cmd.exe misparses LF-only batch files (build-kit.mjs's own hard-won
// finding, Mission A2, restated here because a sealed kit built from a git
// checkout inherits the same LF .gitattributes baseline) — every .cmd this
// builder writes goes through this at write time, no exceptions.
const toCrlf = (s) => s.replace(/\r?\n/g, '\r\n')

const __dirname = dirname(fileURLToPath(import.meta.url))
const kitDir = __dirname
const meshRoot = join(kitDir, '..')
// Deliberately a DIFFERENT output directory than build-kit.mjs's dist/ —
// the two builders must never contend for the same target (the Node kit
// stays the rollback path, untouched by this file, per the fencing brief).
const distOut = join(kitDir, 'dist-bare')

// ── entry point parameter ──────────────────────────────────────────────────
const entryArg = process.argv.find((a) => a.startsWith('--entry='))
const ENTRY_FILE = entryArg
  ? resolve(meshRoot, entryArg.slice('--entry='.length))
  : join(meshRoot, 'host', 'bare-entry.mjs')
if (!existsSync(ENTRY_FILE)) {
  throw new Error(`entry file not found: ${ENTRY_FILE} (pass --entry=<path relative to mesh/> to point at a different one)`)
}
console.log(`entry point: ${relative(meshRoot, ENTRY_FILE)}`)

const BARE_EXE_SRC = join(meshRoot, 'node_modules', 'bare-runtime-win32-x64', 'bin', 'bare.exe')
if (!existsSync(BARE_EXE_SRC)) {
  throw new Error(`bare.exe not found at ${BARE_EXE_SRC} — is 'bare' installed as a devDependency in mesh/package.json?`)
}

const BARE_PACK_BIN = join(meshRoot, 'node_modules', 'bare-pack', 'bin.js')
if (!existsSync(BARE_PACK_BIN)) {
  throw new Error(`bare-pack not found at ${BARE_PACK_BIN} — run 'npm i' in mesh/ (bare-pack is a devDependency)`)
}

// ── 0. make sure the reducer is fresh (same discipline as build-kit.mjs) ───
console.log('building reducer.wasm...')
execFileSync(process.execPath, [join(meshRoot, 'scripts', 'build-reducer.mjs')], { stdio: 'inherit', cwd: meshRoot })
const wasmPath = join(meshRoot, 'dist', 'reducer.wasm')
if (!existsSync(wasmPath)) throw new Error(`expected ${wasmPath} after build — did build-reducer.mjs move?`)

// ── 1. clean target — deterministic, re-runnable, no stale-output
//    contamination. Unlike build-kit.mjs's Machine-A/B, this builder has no
//    live client data/ directory concept yet (bare-entry.mjs is a fold
//    demonstration, not a stateful kit-host) — a full wipe-and-rebuild is
//    unconditionally correct here. If/when bare-guide.mjs becomes the entry
//    and gains its own persistent data/, this builder's data-preservation
//    story needs the same guard build-kit.mjs has (hasRealData/trySync) —
//    flagged in PHASE3_KIT_REPORT.md as NOT yet needed, not forgotten. ──────
if (existsSync(distOut)) rmSync(distOut, { recursive: true, force: true })
mkdirSync(distOut, { recursive: true })

// ── 2. bare-pack — the REAL resolution, run for real, not a hand walk ──────
// --host win32-x64: this campaign's target. --offload: BOTH addons and
// assets written as real sibling files (verified recipe — default embedding
// is a confirmed defect for addons, PHASE0_NOTES_B2_PACKAGING_SPIKE.md §3a;
// embedded assets resolve to an unreadable virtual path, §4b/4c).
const bundlePath = join(distOut, 'app.bundle')
console.log(`bare-pack --host win32-x64 --offload -o ${relative(meshRoot, bundlePath)} ${relative(meshRoot, ENTRY_FILE)}`)
execFileSync(process.execPath, [
  BARE_PACK_BIN,
  '--host', 'win32-x64',
  '--offload',
  '-o', bundlePath,
  ENTRY_FILE,
], { stdio: 'inherit', cwd: meshRoot })
if (!existsSync(bundlePath)) throw new Error('bare-pack reported success but app.bundle was not produced')

// ── 3. the runtime binary — copied verbatim, never rebuilt ─────────────────
cpSync(BARE_EXE_SRC, join(distOut, 'bare.exe'))

// ── 4. the launcher — CRLF at write time, %~dp0-relative, no PATH lookup,
//    ASCII-only (build-kit.mjs's own field finding: a non-ASCII byte inside
//    a parenthesized cmd.exe block corrupts the batch parser under a
//    non-UTF-8 codepage — plain hyphens and quotes only, same discipline).
//
// `pause >nul` at the end is CORRECT and stays for the human double-click
// case — it keeps the window open so a client can read the outcome instead
// of it vanishing. But it makes the REAL launcher un-runnable by an
// automated rehearsal without piped stdin, and this campaign has already
// been burned once by a gate that passed because it drove a seam
// differently than production does (PHASE0_NOTES_D2_FLUSH_RACE.md's shell-
// pipe-vs-spawned-pipe lesson) — a rehearsal that silently substitutes
// "bare.exe app.bundle" for "the actual .cmd a human clicks" risks the same
// class of gap. ASYMMFLOW_KIT_NONINTERACTIVE is a DOCUMENTED, explicit
// opt-in: unset (the default, what a client's double-click always sees),
// the pause behaves exactly as before; set, the SAME launcher — same file,
// same code path up to this point — skips the pause so a rehearsal can
// exercise the real .cmd repeatedly without relying on fragile piped-stdin
// invocation quirks (team lead's own cmd.exe //c + cygpath -w findings,
// PHASE3_KIT_REPORT.md §5d).
const RUN_BARE_MESH_CMD = `@echo off
setlocal
cd /d "%~dp0"

"%~dp0bare.exe" "%~dp0app.bundle"

if defined ASYMMFLOW_KIT_NONINTERACTIVE goto skippause
echo.
echo (the kit has stopped - press any key to close this window)
pause >nul
:skippause
`
writeFileSync(join(distOut, 'run_bare_mesh.cmd'), toCrlf(RUN_BARE_MESH_CMD))

// ── 5. portable.flag — presence-alone marker, DP1 plane convention
//    (build-kit.mjs's own marker file, same shape here). ───────────────────
writeFileSync(join(distOut, 'portable.flag'), [
  'asymmflow-mesh SEALED bare kit',
  `entry: ${relative(meshRoot, ENTRY_FILE)}`,
  `built ${new Date().toISOString()}`,
  '',
].join('\n'))

// ── 6. manifest — real byte sizes, every file, for the report ──────────────
function walkManifest(dir, base = dir) {
  const out = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) out.push(...walkManifest(full, base))
    else out.push({ rel: relative(base, full), bytes: statSync(full).size })
  }
  return out
}

const manifest = walkManifest(distOut)
const totalBytes = manifest.reduce((s, f) => s + f.bytes, 0)

console.log('\nsealed kit manifest:')
for (const f of manifest.sort((a, b) => a.rel.localeCompare(b.rel))) {
  console.log(`  ${(f.bytes / 1e6).toFixed(2).padStart(8)} MB  ${f.rel}`)
}
console.log(`\ntotal: ${manifest.length} file(s), ${(totalBytes / 1e6).toFixed(1)} MB`)
console.log(`\nbuilt: ${distOut}`)
