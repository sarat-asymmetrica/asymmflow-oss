// hello.mjs — Band 4 step 1: does the Bare runtime execute at all on this
// Windows machine? No Holepunch stack, no compat shims — just Bare itself.
console.log('bare hello-world', typeof Bare !== 'undefined' ? Bare.argv : process.argv)
