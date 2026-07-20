// apply-reactor.mjs — the JS host side of the go:wasmexport REACTOR channel
// (Bare-runtime campaign Phase 1b). Sibling of apply.mjs (the WASI *command*
// module's host), NOT a replacement for it — apply.mjs and mesh-node.mjs are
// untouched by this phase (owner ruling R1: the command module is the
// rollback path and must stay green).
//
// Channel: mesh/cmd/reducer-reactor/main.go exports three wasm functions —
// malloc(size)->ptr, apply(ptr,len)->fatPtr (packed (outPtr<<32)|outLen,
// since `string` cannot be a go:wasmexport RESULT type — see that file's
// header and docs/bare-campaign/PHASE0_NOTES_C_WASMEXPORT.md §3), and
// free(ptr,size). This module: instantiates the module ONCE (a reactor stays
// warm — unlike apply.mjs's one-shot-per-call command module, see that
// file's own header comment), calls Node's `wasi.initialize()` (the
// documented reactor counterpart of `wasi.start()` for command modules) to
// run the wasm's `_initialize` export exactly once, then drives
// malloc/apply/free directly per call — no temp files, no node:fs, no
// node:os tmpdir at all (this channel touches no filesystem beyond reading
// the .wasm bytes once at load time).
//
// applyViaWasm() mirrors apply.mjs's exact exported signature (parsed JS
// object out) so a future caller (e.g. a reactor-mode mesh-node.mjs) can
// swap the import with no call-site change. applyViaWasmRaw() is the lower-
// level twin returning the undecoded output Buffer — reactor-parity-spike.mjs
// needs the RAW bytes (not a reparsed object) to prove true byte-identity
// against the command module's own stdout bytes; JSON.parse+reserialize
// would risk masking exactly the key-ordering divergence that comparison
// exists to catch.

import { WASI } from 'node:wasi'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const WASM_PATH = join(__dirname, '..', 'dist', 'reducer-reactor.wasm')

// Instantiate + _initialize once; the instance then stays warm for the life
// of the process (this IS the reactor's whole point vs. the command module's
// "memory is consumed by its run" — apply.mjs:24-26).
let _instance = null

function loadInstance() {
  if (_instance) return _instance
  const bytes = readFileSync(WASM_PATH)
  const mod = new WebAssembly.Module(bytes)
  // No stdin/stdout/args/env needed for THIS module's actual work (malloc/
  // apply/free never touch them) — but the wasi_snapshot_preview1 import
  // table still has to be satisfied at instantiation time regardless (Phase
  // 0/1b's own measurement: the real reducer pulls in 15 distinct WASI
  // imports transitively via encoding/json + crypto/ed25519 + crypto/sha256
  // + encoding/hex all importing "os" on wasip1 — see PHASE1B_REPORT.md).
  // node:wasi's default stdio (inherited fds) is a safe, unused-in-practice
  // choice here since nothing this program does ever calls fd_write for
  // anything but a Go runtime panic, which we want visible if it happens.
  const wasi = new WASI({ version: 'preview1', args: ['reducer-reactor'], env: {} })
  const instance = new WebAssembly.Instance(mod, wasi.getImportObject())
  // The reactor counterpart of wasi.start() — runs _initialize exactly once
  // and does NOT try to invoke a (nonexistent, for a reactor) _start / main.
  wasi.initialize(instance)
  _instance = instance
  return instance
}

/**
 * applyViaWasmRaw(ops, config?, mode?) -> Buffer of the reducer's raw output
 * JSON bytes, undecoded. Throws if the reducer itself reported an error
 * (main.go's `emit({"error": ...})` path) — detected on the PARSED form
 * internally, but the returned Buffer on success is exactly what apply()
 * wrote into wasm linear memory, byte for byte.
 */
export function applyViaWasmRaw(ops, config = undefined, mode = '') {
  const instance = loadInstance()
  const { malloc, free, apply, memory } = instance.exports

  const inputBytes = Buffer.from(JSON.stringify({
    ...(mode ? { mode } : {}),
    ...(config ? { config } : {}),
    ops,
  }), 'utf8')

  const inPtr = malloc(inputBytes.length)
  // memory.buffer is read AFRESH here (never cached across a call) — malloc
  // can grow the wasm memory, which detaches any previously-held ArrayBuffer
  // reference. Every access below follows the same discipline.
  new Uint8Array(memory.buffer, inPtr, inputBytes.length).set(inputBytes)

  const fat = apply(inPtr, inputBytes.length) // i64 result -> JS BigInt
  free(inPtr, inputBytes.length)

  const outPtr = Number(fat >> 32n)
  const outLen = Number(fat & 0xffffffffn)
  // Copy out before freeing — the copy must happen while the pointer is
  // still pinned Go-side (main.go's `pinned` registry, released only by the
  // matching free() call below).
  const outBytes = Buffer.from(new Uint8Array(memory.buffer, outPtr, outLen))
  free(outPtr, outLen)

  // Cheap, conservative error-shape sniff: main.go's ONLY single-key-"error"
  // JSON object is the reducer's own error path (parse/marshal failure or
  // unknown mode) — a real State object always carries multiple domain keys
  // (stock/ar/... or manifest/messages/...) so this can't false-positive on
  // real output.
  let maybeErr
  try { maybeErr = JSON.parse(outBytes.toString('utf8')) } catch { maybeErr = null }
  if (maybeErr && typeof maybeErr === 'object' && !Array.isArray(maybeErr) &&
      Object.keys(maybeErr).length === 1 && typeof maybeErr.error === 'string') {
    throw new Error(`reducer-reactor.wasm: ${maybeErr.error}`)
  }

  return outBytes
}

/**
 * applyViaWasm(ops, config?, mode?) -> converged State object (parsed).
 * Signature-compatible with apply.mjs's export of the same name.
 */
export function applyViaWasm(ops, config = undefined, mode = '') {
  return JSON.parse(applyViaWasmRaw(ops, config, mode).toString('utf8'))
}
