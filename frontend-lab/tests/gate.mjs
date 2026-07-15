/* Layout-detector gate harness (KERNEL pillar 5, browser half). Drives the
 * running dev/preview server, clicks each product screen at 1440 then 420,
 * selects the first row (to open detail panels/profile slots), and runs
 * detectLayoutViolations at each width. Exits non-zero if any screen has an
 * overflow offender, a degenerate text column, or page horizontal scroll.
 *
 * Usage:
 *   node tests/gate.mjs                       # all product screens (Showcase excluded)
 *   node tests/gate.mjs "Accounting,Payroll"  # only these labels
 * Env: BASE_URL (default http://localhost:5175). Playwright resolved globally.
 */
import { createRequire } from 'module'
import { detectLayoutViolations } from './layout-detector.mjs'

const require = createRequire(import.meta.url)
const { chromium } = require('C:/Users/schan/AppData/Roaming/npm/node_modules/playwright')

const BASE_URL = process.env.BASE_URL || 'http://localhost:5175'
const only = (process.argv[2] || '').split(',').map((s) => s.trim()).filter(Boolean)
const EXCLUDE = new Set(['Showcase'])

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

async function detectAt(page, width) {
  await page.setViewportSize({ width, height: 900 })
  await sleep(450)
  return page.evaluate(detectLayoutViolations)
}

function bad(r) {
  return r.pageOverflow || r.offenders.length > 0 || r.degenerate.length > 0
}

const run = async () => {
  const browser = await chromium.launch()
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
  await page.goto(BASE_URL, { waitUntil: 'networkidle' })
  await page.waitForSelector('.lab-tab', { timeout: 15000 })

  const labels = await page.$$eval('.lab-tab', (els) => els.map((e) => e.textContent.trim()))
  const targets = labels.filter((l) => !EXCLUDE.has(l) && (only.length === 0 || only.includes(l)))

  const failures = []
  for (const label of targets) {
    await page.setViewportSize({ width: 1440, height: 900 })
    // Dispatch the nav click in-page (the dev harness's fixed "bridge" pill can
    // overlap the last nav tab and intercept Playwright's actionability check;
    // this is dev chrome, not the product screen, so a direct click is fine).
    await page.evaluate((lbl) => {
      const btn = [...document.querySelectorAll('.lab-tab')].find((b) => b.textContent.trim() === lbl)
      if (btn) btn.click()
    }, label)
    await page.waitForLoadState('networkidle').catch(() => {})
    await sleep(700) // mock fetch (~250ms) + render

    // Is the screen showing an error/empty state instead of data?
    const emptyish = await page.evaluate(() => {
      const t = document.querySelector('.lab-main')?.textContent || ''
      return /INTEG gap|Could not load|Failed to|Error:/.test(t)
    })

    const r1440 = await detectAt(page, 1440)

    // Select the first row to open detail panels, then re-check at 1440.
    let r1440row = null
    const rowSel = '.k-tr.clickable, .k-lie-row, .k-amp-cand'
    const hasRow = await page.$(rowSel)
    if (hasRow) {
      await hasRow.click().catch(() => {})
      await sleep(400)
      r1440row = await page.evaluate(detectLayoutViolations)
    }

    const r420 = await detectAt(page, 420)

    const results = { r1440, r1440row, r420 }
    const anyBad = [r1440, r1440row, r420].filter(Boolean).some(bad)
    const status = anyBad ? 'FAIL' : emptyish ? 'WARN(empty?)' : 'OK'
    console.log(`\n[${status}] ${label}`)
    for (const [k, r] of Object.entries(results)) {
      if (!r) continue
      if (bad(r)) {
        console.log(`  ${k}: pageOverflow=${r.pageOverflow} offenders=${JSON.stringify(r.offenders)} degenerate=${JSON.stringify(r.degenerate)}`)
      }
    }
    if (emptyish) console.log('  note: screen text matched an error/INTEG marker — verify it renders DATA, not an error state.')
    if (anyBad) failures.push(label)
  }

  await browser.close()
  console.log(`\n==== gate: ${targets.length - failures.length}/${targets.length} clean ====`)
  if (failures.length) {
    console.log('FAILURES:', failures.join(', '))
    process.exit(1)
  }
}

run().catch((e) => {
  console.error(e)
  process.exit(2)
})
