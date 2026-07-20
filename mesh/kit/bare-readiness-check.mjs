// bare-readiness-check.mjs — Phase 3 guide-entry readiness probe (P0-B).
//
// NOT the guide (mesh/kit/bare-guide.mjs, P1A-wasi-shim's, in flight — not
// touched or read by this file). This is a throwaway-shaped, but real and
// packaged, trivial entry proving the TWO packages bare-guide.mjs will need
// beyond bare-entry.mjs — bare-readline (interactive input) and
// bare-subprocess (spawning a child process) — actually offload and RUN
// correctly through bare-pack, sealed, in hostile geography. If this file
// packs and runs clean, the guide-entry swap is de-risked to the
// `--entry=` one-line change build-bare-kit.mjs was already built to take
// (PHASE3_KIT_REPORT.md §1).
//
// Exercises REAL behavior, not just import resolution:
//   - bare-readline: reads one real line off stdin via Readline's actual
//     'data' event (matches Node's readline shape closely enough per its
//     own index.d.ts) — proves the guide's own interactive-question loop
//     has a working primitive under a sealed bundle.
//   - bare-subprocess: spawnSync's the SEALED bare.exe itself (Bare.argv[0]
//     — the real path this process was launched with, no assumption about
//     where bare.exe sits beyond "wherever we were actually invoked from")
//     with a trivial -e script and captures its stdout — proves the
//     guide's anchor/probe-shell-out primitive works under a sealed bundle.
//     Synchronous spawnSync, matching this campaign's stated preference for
//     synchronous forms over async ones wherever a choice exists
//     (PHASE0_GATE_D2_FLUSH_RACE.md's WebAssembly.compile() lesson — this
//     is a different API, but the same "prefer sync, it's simpler to
//     reason about and this campaign has been burned by async surprises
//     under Bare more than once" instinct applies).
import * as Readline from 'bare-readline'
import { spawnSync } from 'bare-subprocess'
import process from 'bare-process'

const rl = Readline.createInterface({ input: process.stdin, output: process.stdout })

rl.on('data', (line) => {
  console.log('READLINE_OK line=' + JSON.stringify(line.trim()))
  rl.close()

  const bareExePath = Bare.argv[0]
  const result = spawnSync(bareExePath, ['-e', "console.log('SUBPROCESS_CHILD_OK')"])
  const childStdout = (result.stdout ?? Buffer.alloc(0)).toString('utf8').trim()

  if (childStdout.includes('SUBPROCESS_CHILD_OK')) {
    console.log('SUBPROCESS_OK stdout=' + JSON.stringify(childStdout))
  } else {
    console.log('SUBPROCESS_FAIL stdout=' + JSON.stringify(childStdout) + ' status=' + result.status)
  }

  console.log('READINESS_CHECK_DONE')
})
