/* L2 tripwire — "One definition per utility" (KERNEL.md law L2).
 *
 * The old frontend defined formatDate ~20 times. The kernel defines each
 * utility ONCE and every screen imports it. This test makes that mechanical:
 *
 *   - No screen <style> block may (re)define a kernel-owned `.k-*` class
 *     (k-field / k-input / k-field-label / k-field-wide / k-grow / …). Those
 *     live once in styles/kernel.css; screens USE them in markup, never redefine
 *     them locally.
 *   - No screen may re-implement a kernel formatter (formatDate / formatMoney /
 *     formatNumber). They must be imported from `$kernel/format`.
 *
 * EXCLUDED: Showcase.svelte (dev kitchen-sink). App shell chrome is not under
 * src/screens.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const SCREENS_DIR = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'screens')
const EXCLUDED = new Set(['Showcase.svelte'])
const KERNEL_FORMATTERS = ['formatDate', 'formatMoney', 'formatNumber']

function styleBlocks(source: string): string {
  const blocks: string[] = []
  const re = /<style[^>]*>([\s\S]*?)<\/style>/g
  let m: RegExpExecArray | null
  while ((m = re.exec(source))) blocks.push(m[1] ?? '')
  return blocks.join('\n')
}

function stripComments(css: string): string {
  return css.replace(/\/\*[\s\S]*?\*\//g, '')
}

/** Screen source with <style> blocks removed — used for the formatter scan so a
 *  `.k-…` mention or a `format*` word inside CSS never trips the script check. */
function withoutStyle(source: string): string {
  return source.replace(/<style[^>]*>[\s\S]*?<\/style>/g, '')
}

const screenFiles = readdirSync(SCREENS_DIR)
  .filter((f) => f.endsWith('.svelte') && !EXCLUDED.has(f))
  .sort()

describe('L2 — one definition per utility', () => {
  it('has screen files to scan', () => {
    expect(screenFiles.length).toBeGreaterThan(0)
  })

  for (const file of screenFiles) {
    const source = readFileSync(join(SCREENS_DIR, file), 'utf8')

    it(`${file} does not redefine a kernel .k-* class`, () => {
      const css = stripComments(styleBlocks(source))
      // A `.k-…` token in a screen <style> block can only be a (re)definition —
      // screens reference kernel classes from markup, never from their own CSS.
      const hits = [...css.matchAll(/\.k-[a-z][a-z0-9-]*/g)].map((m) => m[0] ?? '')
      expect(
        [...new Set(hits)],
        `${file} redefines kernel utility classes in <style>: ${[...new Set(hits)].join(', ')}`,
      ).toEqual([])
    })

    it(`${file} does not re-implement a kernel formatter`, () => {
      const script = withoutStyle(source)
      const offenders: string[] = []
      for (const fn of KERNEL_FORMATTERS) {
        // local definition: `function fmt`, `const fmt =`, `let fmt =`, `var fmt =`.
        // Import destructures (`import { formatMoney } from …`) have no such keyword.
        if (new RegExp(`\\b(?:function|const|let|var)\\s+${fn}\\b`).test(script)) offenders.push(fn)
      }
      expect(
        offenders,
        `${file} re-implements kernel formatter(s): ${offenders.join(', ')} — import from $kernel/format`,
      ).toEqual([])
    })
  }
})
