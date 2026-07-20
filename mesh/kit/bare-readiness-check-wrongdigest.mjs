// Diagnostic (P0-B, PHASE3_KIT_REPORT.md §11): a copy of bare-entry.mjs's
// own logic with a DELIBERATELY WRONG op set, to test the failure mode that
// matters most for exit-code propagation: a kit that runs successfully,
// produces WRONG content, and never throws at all. This is the shape of
// failure exit-code propagation CANNOT catch, by construction — the process
// exits 0 because nothing went wrong from bare.exe's own point of view, only
// the business answer is wrong. Kept in kit/ (this coder's fence).
import * as fs from 'bare-fs'
import { applyViaWasm, setWasmSource } from '../host/apply-bare.mjs'

const wasmAssetPath = import.meta.asset('../dist/reducer.wasm')
setWasmSource(fs.readFileSync(new URL(wasmAssetPath)))

// Deliberately DIFFERENT from bare-entry.mjs's canonical OPS (dropped the
// PH-200 moves) — a valid op set the reducer folds cleanly and normally,
// producing a real, different, non-throwing digest.
const OPS = [
  { seq: 1, actor: 'dev-a', sku: 'TX-100', delta: +10, ts: 100 },
  { seq: 2, actor: 'dev-a', sku: 'TX-100', delta: -6, ts: 200 },
  { seq: 1, actor: 'dev-b', sku: 'TX-100', delta: -6, ts: 150 },
]

const EXPECTED_DIGEST = '6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410' // bare-entry.mjs's real expectation

const state = applyViaWasm(OPS)

if (state.digest === EXPECTED_DIGEST) {
  console.log('BARE_ENTRY_FOLD_OK digest=' + state.digest)
} else {
  console.log('BARE_ENTRY_FOLD_MISMATCH got=' + state.digest + ' want=' + EXPECTED_DIGEST)
}
