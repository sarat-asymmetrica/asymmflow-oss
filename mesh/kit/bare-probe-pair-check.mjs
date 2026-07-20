// SC-0 evidence: re-run the Sealed Ship's two-process DHT punch claim on
// TODAY's tree, with the timing variable controlled.
//
// The first attempt at this (orchestrator, 2026-07-20) reported CORRIDOR RED.
// That verdict was a HARNESS artifact, not a network result: the --listen
// probe carries its own 58s hard watchdog, and the manual dial was launched
// after it had already fired. Recorded rather than quietly discarded (Rule 1
// / verify the probe — the orchestrator's own third wrong probe of the
// campaign line). This driver removes the gap: the dial is spawned the
// instant the listener prints its KEY.
//
// NEGATIVE CONTROL: a second pair dials a well-formed but WRONG key (a fresh
// random z32 nobody is listening on). If that also came back GREEN the whole
// measurement would be meaningless.
// NODE-ONLY by construction (like spawn-pipe-harness.mjs): this is the PARENT
// half of the seam — the thing that spawns two Bare probes and reads them.
// It is never imported by, or shipped inside, anything that runs AS the Bare
// child, so its `node:` specifiers are legitimate here.
//
// Run: node kit/bare-probe-pair-check.mjs
import { spawn } from 'node:child_process'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const MESH = join(dirname(fileURLToPath(import.meta.url)), '..')
const BARE = join(MESH, 'node_modules', 'bare-runtime-win32-x64', 'bin', 'bare.exe')
const PROBE = join(MESH, 'kit', 'bare-probe.mjs')

function spawnProbe(args) {
  const child = spawn(BARE, [PROBE, ...args], { cwd: MESH, stdio: ['ignore', 'pipe', 'pipe'] })
  const state = { out: '', err: '', code: null, child }
  child.stdout.on('data', (d) => { state.out += d.toString() })
  child.stderr.on('data', (d) => { state.err += d.toString() })
  state.done = new Promise((r) => child.on('close', (c) => { state.code = c; r() }))
  return state
}

async function pair(label, dialKeyFrom) {
  const listener = spawnProbe(['--listen'])
  // wait for the KEY line (or the listener dying)
  const key = await new Promise((resolve) => {
    const t = setInterval(() => {
      const m = listener.out.match(/KEY:\s*([0-9a-z ]+)/)
      if (m) { clearInterval(t); resolve(m[1].replace(/\s+/g, '')) }
      if (listener.code !== null) { clearInterval(t); resolve(null) }
    }, 200)
    setTimeout(() => { clearInterval(t); resolve(null) }, 40000)
  })
  if (!key) {
    listener.child.kill()
    console.log(`${label}: LISTENER NEVER PRINTED A KEY -- inconclusive`)
    return { label, verdict: null }
  }
  const dialKey = dialKeyFrom ? dialKeyFrom(key) : key
  const t0 = Date.now()
  const dialer = spawnProbe(['--dial', dialKey])
  await dialer.done
  const elapsed = Date.now() - t0
  listener.child.kill()
  await listener.done.catch(() => {})
  const verdict = (dialer.out.match(/CORRIDOR (GREEN|AMBER|RED)/) || [])[1] ?? '(none)'
  const rtt = (dialer.out.match(/RTT[^\n]*/) || [])[0] ?? ''
  const punch = (dialer.out.match(/(PASS|FAIL): punch test[^\n]*/) || [])[0] ?? '(no punch line)'
  console.log(`${label}: verdict=${verdict}  ${punch}  ${rtt}  [dial took ${(elapsed / 1000).toFixed(1)}s]`)
  return { label, verdict, punch }
}

// A well-formed z32 key of the right length that nobody is listening on.
function wrongKey(realKey) {
  const alphabet = 'ybndrfg8ejkmcpqxot1uwisza345h769'
  let out = ''
  for (let i = 0; i < realKey.length; i++) {
    // deterministically rotate every character -> a valid-shaped, wrong key
    const idx = alphabet.indexOf(realKey[i])
    out += idx === -1 ? realKey[i] : alphabet[(idx + 7) % alphabet.length]
  }
  return out
}

console.log('SC-0 probe re-verification: two-process DHT punch, today\'s tree\n')
const positive = await pair('POSITIVE (dial the real key)', null)
const negative = await pair('NEGATIVE CONTROL (dial a wrong key)', wrongKey)

console.log('')
if (negative.verdict === 'GREEN') {
  console.log('HARNESS UNTRUSTWORTHY: the negative control also went GREEN.')
  process.exit(1)
}
console.log(`harness can report the opposite result: negative control = ${negative.verdict} (not GREEN) -- OK`)
console.log(`positive result stands as measured: ${positive.verdict}`)
