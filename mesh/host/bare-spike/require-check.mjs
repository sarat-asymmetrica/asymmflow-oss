// require-check.mjs — Band 4 step 2: attempt to load each Holepunch stack
// module the bridge depends on, ONE AT A TIME, under Bare. Report exact
// success/failure per module so blockers are characterized precisely
// instead of one opaque stack trace burying the real culprit.
const modules = [
  'corestore',
  'autobase',
  'hyperswarm',
  'hyperdht',
  'hyperbee',
  'hyperblobs',
  'hypercore',
  'hypercore-id-encoding',
  'blind-peer',
  'blind-peering',
  'protomux-wakeup',
]

for (const name of modules) {
  try {
    await import(name)
    console.log(`OK   ${name}`)
  } catch (err) {
    console.log(`FAIL ${name} :: ${err.code ?? err.constructor.name} :: ${err.message.split('\n')[0]}`)
  }
}
