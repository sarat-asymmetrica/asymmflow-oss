/* Business Settings bridge module — self-contained: types + mock + real +
 * switch (pricing.ts/suppliers.ts pattern). K4 SettingsScreen split: this is
 * the "general + business rules" consolidation (see
 * screens/parity/Settings.parity.md) — app prefs + default margin/VAT/
 * currency/company-name/fiscal-year-start, the one bespoke form piece of
 * the old 10-tab screen.
 *
 * Real bindings confirmed on App (wailsjs/go/main/App.d.ts):
 * GetSettings/UpdateSettings — but BOTH are generic `Record<string, any>`
 * (no typed Go model exists for "settings"), so the key names below
 * (company_name, base_currency, default_margin_percent, vat_rate_percent,
 * fiscal_year_start_month) are this bridge's ASSUMPTION, not a verified
 * schema — the real Go handler's key vocabulary was not confirmed against
 * this lab. Fetch maps those keys defensively (blank/zero on a miss, same
 * doctrine as suppliers.ts' profile-only fields); UpdateSettings stays
 * INTEG-gapped so a wrong key name can never silently write the wrong
 * field into a FINANCIAL hot-zone record. Synthetic-only data
 * (SYNTHETIC_IDENTITY.md). */

import { pick } from './runtime'
import { num, str } from './map'
import { GetSettings } from '$wails/go/main/App'

export interface BusinessSettingsData {
  companyName: string
  baseCurrency: string
  defaultMarginPercent: number
  vatRatePercent: number
  /** 1 = January … 12 = December. */
  fiscalYearStartMonth: number
}

/* ---- mock: deterministic, no adversarial seasoning — a settings screen
 * has one row, not a dataset, so there's nothing to stress-test at scale. */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

let cache: BusinessSettingsData = {
  companyName: 'Al Manar Instrumentation & Trading W.L.L.',
  baseCurrency: 'BHD',
  defaultMarginPercent: 22,
  vatRatePercent: 10,
  fiscalYearStartMonth: 1,
}

async function mockFetch(): Promise<BusinessSettingsData> {
  await sleep(180)
  return { ...cache }
}

async function mockUpdate(data: BusinessSettingsData): Promise<void> {
  cache = { ...data }
  await sleep(150)
}

/* ---- real: fetch WIRED (best-effort key mapping, see file header);
 * UpdateSettings mutation INTEG-gapped. ---- */
function mapSettings(r: Record<string, unknown>): BusinessSettingsData {
  return {
    companyName: str(r.company_name),
    baseCurrency: str(r.base_currency) || 'BHD',
    defaultMarginPercent: num(r.default_margin_percent),
    vatRatePercent: num(r.vat_rate_percent),
    fiscalYearStartMonth: num(r.fiscal_year_start_month) || 1,
  }
}

async function realFetch(): Promise<BusinessSettingsData> {
  const r = await GetSettings()
  return mapSettings((r ?? {}) as Record<string, unknown>)
}

async function realUpdate(_data: BusinessSettingsData): Promise<void> {
  throw new Error(
    'INTEG gap: UpdateSettings takes an unverified Record<string, any> key schema — wires at K5 once the ' +
      'real Go handler\'s key names are confirmed, not guessed against a FINANCIAL hot-zone record.',
  )
}

/* ---- public switched API (viewmodel imports THESE) ---- */
export const fetchBusinessSettings = (): Promise<BusinessSettingsData> => pick(realFetch, mockFetch)()
export const updateBusinessSettings = (d: BusinessSettingsData): Promise<void> => pick(realUpdate, mockUpdate)(d)
