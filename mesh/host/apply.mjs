// apply.mjs — the JS host side of the Sovereign Mesh determinism spike.
//
// It runs the wasip1 reducer (mesh/dist/reducer.wasm) over an op-log and returns
// the converged state. This is the two-runtime boundary Mission A exists to
// price: JS (which will host Autobase/Hypercore/Holesail) calling the Go kernel
// reducer compiled to WASM. Prove this is clean + deterministic and the whole
// "kernel as distributed apply() reducer" bet is de-risked.
//
// Channel: the reducer is a WASI *command* module reading JSON from stdin and
// writing JSON to stdout. We feed stdin / capture stdout through real temp-file
// file descriptors (not pipes) so there is no buffering deadlock and the run is
// fully deterministic. Next step (docs/MESH_PROGRESS.md) swaps this for an
// incremental //go:wasmexport apply() wired straight into Autobase's apply().

import { WASI } from 'node:wasi'
import { readFileSync, openSync, closeSync, writeFileSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const WASM_PATH = join(__dirname, '..', 'dist', 'reducer.wasm')

// Compile the module once; instantiate fresh per apply (a command module's
// memory is consumed by its run). No wall-clock / randomness here either — a
// pid + counter names temp files so a resumed/replayed run is reproducible.
let _module = null
let _counter = 0

function loadModule() {
  if (_module) return _module
  _module = new WebAssembly.Module(readFileSync(WASM_PATH))
  return _module
}

/**
 * applyViaWasm(ops) -> converged State object.
 * ops: [{ seq, actor, sku, delta, ts }]
 * Returns: { stock, rejected, applied, digest, opsHashed }
 */
export function applyViaWasm(ops) {
  const mod = loadModule()

  const id = `${process.pid}-${_counter++}`
  const inPath = join(tmpdir(), `mesh-reducer-in-${id}.json`)
  const outPath = join(tmpdir(), `mesh-reducer-out-${id}.json`)

  writeFileSync(inPath, JSON.stringify({ ops }))
  // Pre-create the output file so we can hand WASI a writable fd for stdout.
  writeFileSync(outPath, '')

  const inFd = openSync(inPath, 'r')
  const outFd = openSync(outPath, 'w')

  try {
    const wasi = new WASI({
      version: 'preview1',
      args: ['reducer'],
      env: {},
      stdin: inFd,
      stdout: outFd,
      stderr: 2,
      returnOnExit: true, // return the exit code instead of killing the host process
    })

    const instance = new WebAssembly.Instance(mod, wasi.getImportObject())
    const code = wasi.start(instance)
    if (code !== 0) {
      throw new Error(`reducer.wasm exited with code ${code}`)
    }

    const out = readFileSync(outPath, 'utf8')
    return JSON.parse(out)
  } finally {
    closeSync(inFd)
    closeSync(outFd)
    // Best-effort cleanup; ignore if already gone.
    try { rmSync(inPath) } catch {}
    try { rmSync(outPath) } catch {}
  }
}
