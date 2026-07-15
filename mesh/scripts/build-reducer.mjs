// build-reducer.mjs — cross-platform build of the wasip1 reducer.
// Equivalent to: GOOS=wasip1 GOARCH=wasm go build -o mesh/dist/reducer.wasm ./mesh/cmd/reducer
// Works on Windows/macOS/Linux without shell-specific env syntax.

import { spawnSync } from 'node:child_process'
import { mkdirSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const scriptDir = dirname(fileURLToPath(import.meta.url)) // mesh/scripts
const repoRoot = join(scriptDir, '..', '..') // module root (has go.mod)
const outPath = join(repoRoot, 'mesh', 'dist', 'reducer.wasm')

mkdirSync(join(repoRoot, 'mesh', 'dist'), { recursive: true })

const res = spawnSync(
  'go',
  ['build', '-o', outPath, './mesh/cmd/reducer'],
  {
    cwd: repoRoot,
    env: { ...process.env, GOOS: 'wasip1', GOARCH: 'wasm' },
    stdio: 'inherit',
    shell: process.platform === 'win32', // resolve go.exe via PATH on Windows
  },
)

if (res.status !== 0) {
  console.error(`\nbuild-reducer: go build failed (exit ${res.status})`)
  process.exit(res.status ?? 1)
}
console.log(`built ${outPath}`)
