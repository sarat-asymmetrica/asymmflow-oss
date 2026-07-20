// bare-guide-entry.mjs — the SEALED-KIT entry point. This is what
// build-bare-kit.mjs packs, NOT bare-guide.mjs directly. Same shape as
// host/bare-entry.mjs (the reducer's own Bare-only pack entry) — this
// file exists to close the same class of problem that one closes,
// discovered live during kit integration (2026-07-20, the lead's own
// root-cause, recorded here rather than paraphrased):
//
// bare-guide.mjs's own `isMain` guard —
//   const argv = typeof Bare !== 'undefined' ? Bare.argv : process.argv
//   const isMain = argv[1] && new URL(import.meta.url).pathname... === argv[1]...
//   if (isMain) await runGuide()
// — is CORRECT for `bare kit/bare-guide.mjs` (a real script invocation:
// argv[1] and import.meta.url both name the same real file on disk) and
// for bare-guide-spike.mjs's own spawn-pipe gate (same reason). It is
// STRUCTURALLY FALSE the moment bare-guide.mjs is bare-pack'd: inside a
// bundle, `argv[1]` is the BUNDLE's own path (e.g. `.../app.bundle`) while
// `import.meta.url` resolves to the VIRTUAL path of the module *inside*
// that bundle (e.g. `/kit/bare-guide.mjs`) — the two can never compare
// equal, so `isMain` is false every time, `runGuide()` is never called,
// and the sealed kit produces zero bytes on stdout/stderr with exit code
// 0 (silent no-op, not a crash — nothing surfaced it until this was
// bisected by packing progressively smaller entries and watching for
// where the printed trail stopped).
//
// THE FIX IS NOT "make the guard bundle-aware" — detecting "am I the
// bundle's own main module" from inside a bundle is fragile and
// topology-dependent, exactly the class of assumption this campaign keeps
// finding broken in a new way each time it's tried. The fix is this file:
// a thin, unconditional entry that always calls `runGuide()`, with no
// guard at all, because an entry file's only job IS to be the entry — it
// is never imported for its exports (unlike bare-guide.mjs, which
// bare-guide-spike.mjs DOES import directly, for the pure
// normalizeCode/groupInFours unit checks — that is exactly why
// bare-guide.mjs needs its own guard and this file must not have one).
//
// Do not "simplify" this away by pointing build-bare-kit.mjs back at
// bare-guide.mjs directly — that is the exact regression this file exists
// to prevent, and the failure mode (silent, exit 0) would not be caught
// by a casual smoke test.
// ── WASM ASSET INJECTION — do not delete, and do not move into apply-bare.mjs ──
//
// apply-bare.mjs locates reducer.wasm by default with
// `new URL('../dist/reducer.wasm', import.meta.url)`. That is correct on disk and
// WRONG in two independent ways once packed:
//
//   1. bare-pack's static asset detector does not recognise that form — only
//      `import.meta.asset()` — so reducer.wasm is never offloaded into the kit
//      at all (verified: the guide-entry manifest had no dist/reducer.wasm).
//   2. inside a bundle `import.meta.url` is a VIRTUAL path, so even a present
//      file would not be found there.
//
// The observable symptom is nasty precisely because it is NOT a crash: the whole
// ceremony renders, the room is created, and only posting fails with
// "(not posted -- ENOENT: ...app.bundle\dist\reducer.wasm)". A casual look at a
// kit that boots, draws its menu and says Goodbye reads as working.
//
// `import.meta.asset()` is a bare-pack/Bare lexer feature with no Node
// equivalent, which is exactly why it belongs HERE, in a Bare-only entry, and
// never in apply-bare.mjs — that file must keep running under BOTH runtimes,
// since the Node leg of bare-parity-spike.mjs is what makes the byte-identity
// proof meaningful. Same separation host/bare-entry.mjs already uses, and the
// direct `bare-fs` import (rather than `#fs`) is safe here for the same reason:
// this file never runs under Node.
import * as fs from 'bare-fs'
import { setWasmSource } from '../host/apply-bare.mjs'
import { runGuide } from './bare-guide.mjs'

const wasmAssetPath = import.meta.asset('../dist/reducer.wasm')
setWasmSource(fs.readFileSync(new URL(wasmAssetPath)))

await runGuide()
