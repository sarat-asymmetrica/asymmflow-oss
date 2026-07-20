// Isolation test (P0-B): bare-readline ONLY, no bare-subprocess, to confirm
// the spawned-pipe hang found in bare-readiness-check.mjs is caused by
// bare-readline specifically and not an interaction between the two.
// Throwaway diagnostic, kept in kit/ (this coder's fence).
import * as Readline from 'bare-readline'
import process from 'bare-process'

const rl = Readline.createInterface({ input: process.stdin, output: process.stdout })
rl.on('data', (line) => {
  console.log('READLINE_ONLY_OK line=' + JSON.stringify(line.trim()))
  rl.close()
})
