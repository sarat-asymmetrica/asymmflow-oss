/* The layout-truth detector — KERNEL.md pillar 5's browser-side half.
 * Evaluated in-page (Playwright page.evaluate) at multiple viewport widths.
 *
 * Three rules, each earned by a live bug during kernel development:
 *   1. No element overflows horizontally UNLESS it is a declared scroll
 *      region (k-scroll, k-page-content, k-table-wrap).
 *   2. Declared truncation (text-overflow: ellipsis + nowrap) is policy,
 *      not overflow — whitelisted.
 *   3. No degenerate text column: multi-word text squeezed narrower than
 *      ~4ch (the "one letter per line" collapse min-width:0 permits).
 *
 * Returns { pageOverflow, offenders, degenerate } — all must be empty/false.
 */
export function detectLayoutViolations() {
  const offenders = []
  const degenerate = []
  const scrollOK = new Set(['k-scroll', 'k-page-content', 'k-table-wrap'])
  for (const el of document.querySelectorAll('*')) {
    const declared = [...el.classList].some((c) => scrollOK.has(c))
    const cs = getComputedStyle(el)
    const truncates = cs.textOverflow === 'ellipsis' && cs.whiteSpace === 'nowrap'
    if (
      !declared &&
      !truncates &&
      el.scrollWidth > el.clientWidth + 1 &&
      cs.overflowX !== 'visible' &&
      cs.overflowX !== 'clip'
    ) {
      offenders.push(`${el.className} sw=${el.scrollWidth} cw=${el.clientWidth}`)
    }
    if (el.children.length === 0 && el.textContent && el.textContent.trim().split(/\s+/).length > 2) {
      const r = el.getBoundingClientRect()
      if (r.width > 0 && r.width < 34 && r.height > 60) {
        degenerate.push(`${el.className} w=${Math.round(r.width)}`)
      }
    }
  }
  return {
    pageOverflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    offenders,
    degenerate,
  }
}
