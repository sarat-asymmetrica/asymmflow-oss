// bare-entry.mjs — Bare-ONLY sealed-artifact entry point. NEVER runs under
// Node (unlike apply-bare.mjs, which must). This is the one place in the
// tree allowed to know about bare-pack packaging mechanics — everything else
// (apply-bare.mjs, wasi-preview1-lite.mjs) stays runtime-agnostic by
// construction, per the Phase 2 DI ruling (PHASE2_ASSET_LOCATION.md).
//
// Demonstrates + proves the real recipe for a sealed artifact touching the
// reducer: `import.meta.asset()` (a bare-pack/Bare lexer feature with no
// Node equivalent — PHASE0_NOTES_B2_PACKAGING_SPIKE.md §4c) resolves
// reducer.wasm's bytes at pack time, embedded via `--offload`/
// `--offload-assets` as a real sibling file (not a virtual in-bundle path —
// §4b's landmine), and `setWasmSource()` injects them into apply-bare.mjs
// without that file ever importing `bare-fs` directly or branching on
// `Bare`.
//
// Direct `bare-fs` import (not `#fs`) is correct and safe HERE ONLY, because
// this file is Bare-only by construction — the dual-runtime constraint that
// makes apply-bare.mjs need the `#fs` condition map does not apply to a file
// that is never reached from the Node line.
import * as fs from 'bare-fs'
import { applyViaWasm, setWasmSource } from './apply-bare.mjs'

const wasmAssetPath = import.meta.asset('../dist/reducer.wasm')
setWasmSource(fs.readFileSync(new URL(wasmAssetPath)))

// The canonical scenario (mirrors smoke.mjs's OPS / reducer_test.go
// canonicalOps) — a REAL fold, not an empty-ops smoke check: two devices
// move two SKUs; dev-a's Seq-2 sale is the one that would breach the TX-100
// floor and must be deterministically rejected.
const OPS = [
  { seq: 1, actor: 'dev-a', sku: 'TX-100', delta: +10, ts: 100 },
  { seq: 2, actor: 'dev-a', sku: 'TX-100', delta: -6, ts: 200 },
  { seq: 1, actor: 'dev-b', sku: 'TX-100', delta: -6, ts: 150 },
  { seq: 1, actor: 'dev-a', sku: 'PH-200', delta: +3, ts: 120 },
  { seq: 2, actor: 'dev-b', sku: 'PH-200', delta: +4, ts: 220 },
]

// The exact digest smoke.mjs's own run of this same OPS set produces via
// apply.mjs (the Node host) — the real, content-level assertion the gate
// brief asked for, not merely "it printed something and exited 0"
// (PHASE0_GATE_D2_FLUSH_RACE.md: Bare exit codes are not trustworthy
// evidence of success on their own).
const EXPECTED_DIGEST = '6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410'

const state = applyViaWasm(OPS)

if (state.digest === EXPECTED_DIGEST) {
  console.log('BARE_ENTRY_FOLD_OK digest=' + state.digest)
} else {
  console.log('BARE_ENTRY_FOLD_MISMATCH got=' + state.digest + ' want=' + EXPECTED_DIGEST)
}
