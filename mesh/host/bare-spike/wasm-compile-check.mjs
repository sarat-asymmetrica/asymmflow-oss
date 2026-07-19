// wasm-compile-check.mjs — Band 4 step 3b: WebAssembly.compile itself works
// under Bare (typeof WebAssembly === 'object' from wasi-check.mjs); does it
// actually parse the real Go wasip1 reducer.wasm binary? This isolates
// "can Bare load OUR module at all" from "does Bare have a WASI runtime to
// satisfy its imports" (the latter is the confirmed blocker).
import fs from 'bare-fs'

const bytes = fs.readFileSync(new URL('../../dist/reducer.wasm', import.meta.url))
console.log('reducer.wasm bytes:', bytes.length)

try {
  const mod = await WebAssembly.compile(bytes)
  console.log('OK   WebAssembly.compile(reducer.wasm) succeeded')
  const imports = WebAssembly.Module.imports(mod)
  const importModules = [...new Set(imports.map((i) => i.module))]
  console.log('required import namespaces:', importModules)
  console.log('import count:', imports.length)
} catch (err) {
  console.log(`FAIL WebAssembly.compile :: ${err.constructor.name} :: ${err.message.split('\n')[0]}`)
}
