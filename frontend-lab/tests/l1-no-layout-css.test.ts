/* L1 tripwire — "No raw layout CSS in screens" (KERNEL.md law L1).
 *
 * The whole campaign thesis: layout lives in kernel primitives, never in a
 * screen <style> block. Sprint 1/2 enforced this by hand across ~50 screens;
 * this test makes it mechanical so it can never regress.
 *
 * It scans every src/screens/*.svelte <style> block and FAILS on:
 *   - structural layout props (display / margin / float / flex-* / justify-* /
 *     align-* / grid-* / gap) — these belong to Stack/Row/Grid/FormGrid, never a screen;
 *   - sizing/spacing props (padding / width / height / min-height / top / left / …)
 *     carrying a raw `px` value — spacing is primitive-owned; `0` and token/%
 *     values are fine;
 *   - `min-width` other than `0` (the anti-overflow idiom is the only allowed use);
 *   - raw `#rrggbb` hex colors — color comes through the semantic token layer (L3).
 *
 * ALLOWED in a screen: font-*, color, letter-spacing, text-*, white-space,
 * overflow*, cursor, resize, border/border-radius/outline/background as skin
 * (token- or keyword-valued), transition, and `min-width: 0`.
 *
 * EXCLUDED: Showcase.svelte (intentional dev kitchen-sink with a 3000px overflow
 * demo — detector-exempt by design). App shell chrome (App.svelte, app/*) is not
 * under src/screens and is exempt like the old lab-shell.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const SCREENS_DIR = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'screens')

/** Detector-exempt by design (KERNEL.md + handoff). */
const EXCLUDED = new Set(['Showcase.svelte'])

/** Structural layout — belongs to a primitive, never a screen. Matched by exact
 *  name or by these prefixes. */
const FORBIDDEN_EXACT = new Set([
  'display',
  'float',
  'flex-direction',
  'flex-flow',
  'flex-wrap',
  'justify-content',
  'justify-items',
  'justify-self',
  'align-items',
  'align-content',
  'align-self',
  'place-items',
  'place-content',
  'place-self',
  'gap',
  'row-gap',
  'column-gap',
])
const FORBIDDEN_PREFIX = ['margin', 'grid-template', 'grid-column', 'grid-row', 'grid-area', 'grid-auto']

/** Sizing/spacing props that must not carry a raw px value. */
const PX_CHECKED = new Set([
  'padding',
  'padding-top',
  'padding-right',
  'padding-bottom',
  'padding-left',
  'width',
  'height',
  'min-height',
  'max-height',
  'max-width',
  'top',
  'left',
  'right',
  'bottom',
  'inset',
])

const RAW_PX = /\b\d*\.?\d+px\b/
const RAW_HEX = /#[0-9a-fA-F]{3,8}\b/

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

/** Return every `prop: value` declaration inside a rule body (brace-delimited),
 *  so selectors and pseudo-classes (`.foo:focus {`) are never mistaken for
 *  declarations. Non-nested `{…}` extraction handles @media (inner rules match
 *  individually; the media query itself sits outside any brace body). */
function declarations(css: string): Array<{ prop: string; value: string }> {
  const decls: Array<{ prop: string; value: string }> = []
  const bodies = css.match(/\{([^{}]*)\}/g) ?? []
  for (const body of bodies) {
    const inner = body.slice(1, -1)
    for (const chunk of inner.split(';')) {
      const idx = chunk.indexOf(':')
      if (idx === -1) continue
      const prop = chunk.slice(0, idx).trim().toLowerCase()
      const value = chunk.slice(idx + 1).trim()
      if (!prop || !value) continue
      decls.push({ prop, value })
    }
  }
  return decls
}

function violationsFor(prop: string, value: string): string | null {
  if (RAW_HEX.test(value)) return `raw hex color in \`${prop}: ${value}\``
  if (FORBIDDEN_EXACT.has(prop) || FORBIDDEN_PREFIX.some((p) => prop === p || prop.startsWith(p + '-')))
    return `structural layout prop \`${prop}\` (belongs in a kernel primitive)`
  if (prop === 'min-width') {
    if (value !== '0') return `\`min-width: ${value}\` — only \`min-width: 0\` is allowed`
    return null
  }
  if (PX_CHECKED.has(prop) && RAW_PX.test(value)) return `raw px in \`${prop}: ${value}\``
  return null
}

const screenFiles = readdirSync(SCREENS_DIR)
  .filter((f) => f.endsWith('.svelte') && !EXCLUDED.has(f))
  .sort()

describe('L1 — no raw layout CSS in screens', () => {
  it('has screen files to scan', () => {
    expect(screenFiles.length).toBeGreaterThan(0)
  })

  for (const file of screenFiles) {
    it(`${file} contains no raw layout CSS`, () => {
      const css = stripComments(styleBlocks(readFileSync(join(SCREENS_DIR, file), 'utf8')))
      const found: string[] = []
      for (const { prop, value } of declarations(css)) {
        const v = violationsFor(prop, value)
        if (v) found.push(v)
      }
      expect(found, `${file} L1 violations:\n  - ${found.join('\n  - ')}`).toEqual([])
    })
  }
})
