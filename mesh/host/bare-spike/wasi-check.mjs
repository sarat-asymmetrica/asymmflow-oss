// wasi-check.mjs — Band 4 step 3: the reducer boundary. apply.mjs
// (mesh/host/apply.mjs) drives mesh/dist/reducer.wasm (a Go wasip1 command
// module) via node:wasi + file-descriptor stdin/stdout. Does Bare have
// node:wasi, or WebAssembly at all?
console.log('WebAssembly typeof:', typeof WebAssembly)

try {
  const wasi = await import('node:wasi')
  console.log('OK   node:wasi ::', Object.keys(wasi))
} catch (err) {
  console.log(`FAIL node:wasi :: ${err.code ?? err.constructor.name} :: ${err.message.split('\n')[0]}`)
}

try {
  const wasi2 = await import('wasi')
  console.log('OK   wasi (bare builtin?) ::', Object.keys(wasi2))
} catch (err) {
  console.log(`FAIL wasi :: ${err.code ?? err.constructor.name} :: ${err.message.split('\n')[0]}`)
}

try {
  const bareWasi = await import('bare-wasi')
  console.log('OK   bare-wasi ::', Object.keys(bareWasi))
} catch (err) {
  console.log(`FAIL bare-wasi :: ${err.code ?? err.constructor.name} :: ${err.message.split('\n')[0]}`)
}
