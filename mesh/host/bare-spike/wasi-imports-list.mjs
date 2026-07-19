// wasi-imports-list.mjs — Band 4 step 3c: enumerate the EXACT 18
// wasi_snapshot_preview1 syscalls the Go reducer needs, so the shim
// inventory in the findings report is precise, not a guess.
import fs from 'bare-fs'

const bytes = fs.readFileSync(new URL('../../dist/reducer.wasm', import.meta.url))
const mod = await WebAssembly.compile(bytes)
const imports = WebAssembly.Module.imports(mod)
for (const i of imports) console.log(`${i.module}.${i.name}  (${i.kind})`)
