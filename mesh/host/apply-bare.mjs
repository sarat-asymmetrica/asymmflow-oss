// apply-bare.mjs — the Bare-runtime counterpart to apply.mjs (mesh/host/
// apply.mjs), same channel contract, same exported signature
// (`applyViaWasm(ops, config?, mode?)`), swapping node:wasi + temp-file fds
// for wasi-preview1-lite.mjs (mesh/host/wasi-preview1-lite.mjs) + in-memory
// buffers. Runs the UNMODIFIED mesh/dist/reducer.wasm — no reducer changes,
// no cmd/reducer changes (D6/R1 condition 1).
//
// Why in-memory buffers instead of apply.mjs's temp-file-per-call design:
// the brief flagged this as "strictly better (no filesystem, no temp-file-
// per-call)" IF it can be proven to work — it does (see PHASE1A_REPORT.md).
// stdin's whole payload is already a JS string/Buffer before this function
// is ever called, and the reducer's own contract (mesh/cmd/reducer/main.go)
// is `io.ReadAll(os.Stdin)` then one `os.Stdout.Write` — a full-buffer
// read-then-write, never streamed — so there is no correctness reason to
// touch a real filesystem at all, on either runtime.
//
// Runtime feature-detection: this file DOES run under both Node and Bare
// (the parity spike drives it from both), so unlike wasi-preview1-lite.mjs
// it needs *some* way to read the wasm file, which differs by runtime
// (node:fs vs bare-fs — no bare-path/bare-os were needed; `new URL(...,
// import.meta.url)` plus each fs module's Node-shaped `readFileSync(url)`
// was enough, mirroring the exact pattern already proven in
// host/bare-spike/wasi-imports-list.mjs). `Bare` is a global namespace that
// only exists under the Bare runtime (verified: PHASE0_NOTES_A_BARE.md §2.2
// lists `Bare.platform`/`Bare.argv`/etc as real Bare-only globals) — that is
// the feature-detection switch used below, at module load time, once.
import { createWASI } from './wasi-preview1-lite.mjs'

const isBare = typeof Bare !== 'undefined'
const fsMod = isBare ? await import('bare-fs') : await import('node:fs')

const WASM_URL = new URL('../dist/reducer.wasm', import.meta.url)

// Compile once, instantiate fresh per apply — same reasoning as apply.mjs's
// header: a command module's memory is consumed by its one run.
let _module = null
function loadModule() {
  if (_module) return _module
  _module = new WebAssembly.Module(fsMod.readFileSync(WASM_URL))
  return _module
}

// A read-only view over an in-memory input buffer — fd 0.
function makeStdinHandle(input) {
  let pos = 0
  return {
    read(dst) {
      const remaining = input.length - pos
      if (remaining <= 0) return 0
      const n = Math.min(dst.length, remaining)
      input.copy(dst, 0, pos, pos + n)
      pos += n
      return n
    },
  }
}

// An append-only in-memory sink — fd 1 (stdout, the channel's real payload).
function makeCaptureHandle() {
  const chunks = []
  return {
    write(src) { chunks.push(Buffer.from(src)); return src.length },
    result() { return Buffer.concat(chunks) },
  }
}

// fd 2 (stderr): the reducer only writes here on a hard failure path
// (mesh/cmd/reducer/main.go's `fmt.Fprintln(os.Stderr, ...)` before a
// non-zero exit) — surfacing it to the host's own console is the honest
// equivalent of apply.mjs's `stderr: 2` passthrough (real Node stderr fd),
// which this shim has no fd-2-is-the-terminal concept to hand off to.
function makeStderrHandle() {
  return {
    write(src) {
      try { console.error(src.toString('utf8')) } catch { /* best-effort */ }
      return src.length
    },
  }
}

/**
 * applyViaWasmRaw(ops, config?, mode?) -> raw stdout Buffer (NOT JSON.parsed).
 * The byte-identity primitive: bare-parity-spike.mjs compares these bytes
 * directly against the Node WASI host's own raw bytes, because parsing both
 * sides first would silently mask a key-ordering divergence.
 */
export function applyViaWasmRaw(ops, config = undefined, mode = '') {
  const mod = loadModule()

  const input = Buffer.from(JSON.stringify({
    ...(mode ? { mode } : {}),
    ...(config ? { config } : {}),
    ops,
  }), 'utf8')

  const stdout = makeCaptureHandle()
  const { imports, setMemory, WASIExit } = createWASI({
    args: ['reducer'],
    env: {},
    fds: {
      0: makeStdinHandle(input),
      1: stdout,
      2: makeStderrHandle(),
    },
  })

  const instance = new WebAssembly.Instance(mod, imports)
  setMemory(instance.exports.memory)

  let code = 0
  try {
    if (typeof instance.exports._start !== 'function') {
      throw new Error('reducer.wasm: no _start export — not a WASI command module')
    }
    instance.exports._start()
  } catch (err) {
    if (err instanceof WASIExit) code = err.code
    else throw err
  }
  if (code !== 0) {
    throw new Error(`reducer.wasm exited with code ${code}`)
  }

  return stdout.result()
}

/**
 * applyViaWasm(ops, config?, mode?) -> converged State object.
 * Same contract as apply.mjs's applyViaWasm — see that file's header.
 */
export function applyViaWasm(ops, config = undefined, mode = '') {
  return JSON.parse(applyViaWasmRaw(ops, config, mode).toString('utf8'))
}
