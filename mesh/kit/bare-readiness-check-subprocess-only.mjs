// Isolation test (P0-B): bare-subprocess ONLY, no bare-readline, to determine
// whether the hang found under a real spawned pipe (vs a shell pipe) is
// caused by bare-readline specifically. Throwaway diagnostic, kept in kit/
// (this coder's fence) alongside bare-readiness-check.mjs.
import { spawnSync } from 'bare-subprocess'

const bareExePath = Bare.argv[0]
const result = spawnSync(bareExePath, ['-e', "console.log('SUBPROCESS_CHILD_OK')"])
const childStdout = (result.stdout ?? Buffer.alloc(0)).toString('utf8').trim()
console.log(childStdout.includes('SUBPROCESS_CHILD_OK') ? 'SUBPROCESS_ONLY_OK' : 'SUBPROCESS_ONLY_FAIL')
