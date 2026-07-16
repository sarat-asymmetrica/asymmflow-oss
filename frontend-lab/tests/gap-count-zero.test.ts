/* Gap tripwire — "no survivors, no asterisks" (Gap-Close campaign G5).
 *
 * The INTEG + Residue + Gap-Close campaigns drove the bridge's honest
 * `INTEG gap:` throws from ~160 → 0. Every real Wails binding a screen needs is
 * now wired-and-verified (or the affordance retired), so NO bridge function may
 * throw an `INTEG gap` any more. This test pins the count at ZERO mechanically:
 * a future edit that reintroduces a gap throw fails here, the same way the mesh
 * gate rejects an agent trying to approve.
 *
 * Scope: comments are stripped first, so a `// … INTEG gap …` note (docs) never
 * trips the check — only a live `throw new Error('INTEG gap …')` in CODE does.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const BRIDGE_DIR = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'bridge')

/** Remove line + block comments so a documentation mention of "INTEG gap" in a
 *  comment is ignored; only executable code is scanned. (String-literal false
 *  positives are impossible here — the phrase only ever lived inside throws.) */
function stripComments(ts: string): string {
  return ts.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/[^\n]*/g, '')
}

function bridgeFiles(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    if (statSync(full).isDirectory()) out.push(...bridgeFiles(full))
    else if (entry.endsWith('.ts')) out.push(full)
  }
  return out
}

const files = bridgeFiles(BRIDGE_DIR).sort()

describe('Gap tripwire — INTEG gap count is ZERO', () => {
  it('finds bridge files to scan', () => {
    expect(files.length).toBeGreaterThan(20)
  })

  for (const file of files) {
    const name = file.slice(file.indexOf('bridge'))
    it(`${name} has no INTEG-gap throw`, () => {
      const code = stripComments(readFileSync(file, 'utf8'))
      expect(code).not.toContain('INTEG gap')
    })
  }
})
