/* Business Settings bridge module — self-contained: types + mock + real +
 * switch (pricing.ts/suppliers.ts pattern). K4 SettingsScreen split: this is
 * the "general + business rules" consolidation (see
 * screens/parity/Settings.parity.md) — app prefs + default margin/VAT/
 * currency/company-name/fiscal-year-start, the one bespoke form piece of
 * the old 10-tab screen.
 *
 * Real bindings confirmed on App (wailsjs/go/main/App.d.ts):
 * GetSettings/UpdateSettings — both are generic `Record<string, any>` (no
 * typed Go model exists for "settings"). R3: the key vocabulary is now
 * CONFIRMED against `app_setup_documents_surface.go`'s GetSettings/
 * UpdateSettings/saveUserSettings — and it does NOT match what this file
 * originally assumed:
 *   - top level: `companyName` (camelCase, not `company_name`), `currency`
 *     (not `base_currency`), plus `language`/`theme`/`folders`/`apiKeys`/
 *     `gpu`/`office` — none of which this screen's flat shape carries.
 *   - margin/VAT are nested one level down: `business.default_margin` /
 *     `business.vat_rate` (not top-level `default_margin_percent` /
 *     `vat_rate_percent`).
 *   - there is no `fiscal_year_start_month` key anywhere in GetSettings —
 *     that field has nothing real to bind to; it's mock-only until a real
 *     key exists.
 * mapSettings below is fixed to the confirmed keys. UpdateSettings stays
 * INTEG-gapped, but now for a stronger, confirmed reason: `saveUserSettings`
 * (app_setup_documents_surface.go) does a FULL FILE OVERWRITE of
 * settings.json with exactly the map it's given — no merge with what's on
 * disk. Sending this screen's narrow 5-field BusinessSettingsData as-is
 * would silently WIPE `folders`/`apiKeys` (including the Mistral/AIML keys)
 * and every other top-level key GetSettings returns. Wiring this mutation
 * safely means round-tripping the ENTIRE settings object (fetch-merge-write),
 * which is out of scope for this bridge's narrow shape — deferred to K5.
 * Synthetic-only data (SYNTHETIC_IDENTITY.md). */

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

/* ---- real: fetch WIRED against the CONFIRMED key schema (see file header);
 * UpdateSettings mutation INTEG-gapped (confirmed full-file-overwrite risk). ---- */
function mapSettings(r: Record<string, unknown>): BusinessSettingsData {
  const business = (r.business as Record<string, unknown> | undefined) ?? {}
  return {
    companyName: str(r.companyName),
    baseCurrency: str(r.currency) || 'BHD',
    defaultMarginPercent: num(business.default_margin),
    vatRatePercent: num(business.vat_rate),
    // No fiscal-year-start key exists in Go's GetSettings — nothing to map.
    fiscalYearStartMonth: num(r.fiscal_year_start_month) || 1,
  }
}

async function realFetch(): Promise<BusinessSettingsData> {
  const r = await GetSettings()
  return mapSettings((r ?? {}) as Record<string, unknown>)
}

async function realUpdate(_data: BusinessSettingsData): Promise<void> {
  throw new Error(
    'INTEG gap: UpdateSettings persists via saveUserSettings, a FULL settings.json overwrite (no merge) — ' +
      "sending this screen's narrow 5-field shape would wipe folders/apiKeys and every other top-level " +
      'GetSettings key. Confirmed against app_setup_documents_surface.go; wires at K5 with a fetch-merge-write.',
  )
}

/* ---- public switched API (viewmodel imports THESE) ---- */
export const fetchBusinessSettings = (): Promise<BusinessSettingsData> => pick(realFetch, mockFetch)()
export const updateBusinessSettings = (d: BusinessSettingsData): Promise<void> => pick(realUpdate, mockUpdate)(d)
