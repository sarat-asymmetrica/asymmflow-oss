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
 * mapSettings below is fixed to the confirmed keys. UpdateSettings is now WIRED
 * (G3) with the required FETCH-MERGE-WRITE: `saveUserSettings`
 * (app_setup_documents_surface.go) does a FULL FILE OVERWRITE of settings.json
 * with exactly the map it's given — no merge with what's on disk. So realUpdate
 * round-trips the ENTIRE GetSettings object and overlays ONLY this screen's 5
 * fields, leaving `folders`/`apiKeys` (incl. Mistral/AIML keys) and every other
 * top-level key intact. Synthetic-only data (SYNTHETIC_IDENTITY.md). */

import { pick } from './runtime'
import { num, str } from './map'
import { GetAIProviderKeyStatus, GetSettings, SetAPIKeys, UpdateSettings } from '$wails/go/main/App'

export interface BusinessSettingsData {
  companyName: string
  baseCurrency: string
  defaultMarginPercent: number
  vatRatePercent: number
  /** 1 = January … 12 = December. */
  fiscalYearStartMonth: number
}

/** R4 — the AI provider (Butler/Mistral) key. Read back MASKED only: the
 * server never returns the plaintext (maskSecret → '(not set)' | 'abcd****wxyz'),
 * so the UI can show "set / not set" + the last-4 without ever holding the
 * secret. Encryption-at-rest is the SERVER's job (SetAPIKeys → SetSetting with
 * encrypt=true, HKDF+AES-256-GCM); the client sends plaintext and never logs it. */
export interface AIProviderKeyState {
  /** Server-masked representation: '(not set)' or 'abcd****wxyz'. Never the raw key. */
  maskedKey: string
  isSet: boolean
}

/** The settings key the Butler's Mistral backend reads (see SetAPIKeys). */
const AI_PROVIDER_KEY_FIELD = 'mistral_key'

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

// Mock AI-key store: hold only the MASK, never a plaintext secret (mirrors the
// server contract — the real read is masked, the write is fire-and-forget).
let mockAIKeyMask = '(not set)'
async function mockFetchAIKey(): Promise<AIProviderKeyState> {
  await sleep(120)
  return { maskedKey: mockAIKeyMask, isSet: mockAIKeyMask !== '(not set)' }
}
async function mockSaveAIKey(key: string): Promise<void> {
  const k = key.trim()
  // Same mask shape maskSecret produces server-side (first4****last4, or **** if short).
  mockAIKeyMask = k.length <= 8 ? '****' : `${k.slice(0, 4)}****${k.slice(-4)}`
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

async function realUpdate(data: BusinessSettingsData): Promise<void> {
  // FETCH-MERGE-WRITE (G3). saveUserSettings does a FULL settings.json overwrite,
  // so sending this screen's narrow 5-field shape alone would wipe folders/apiKeys
  // and every other top-level key. We round-trip the ENTIRE GetSettings object and
  // overlay ONLY the fields this screen owns; everything else survives untouched.
  // (GetSettings returns apiKeys MASKED; the server ignores masked keys on write,
  // so echoing them back is a safe no-op — never a plaintext wipe.)
  const full = ((await GetSettings()) ?? {}) as Record<string, unknown>
  full.companyName = data.companyName
  full.currency = data.baseCurrency
  const business = (full.business as Record<string, unknown> | undefined) ?? {}
  business.default_margin = data.defaultMarginPercent
  business.vat_rate = data.vatRatePercent
  full.business = business
  // fiscalYearStartMonth is deliberately NOT merged: GetSettings has no such key
  // (ledgered mock-only), so writing it would never read back — it stays a no-op
  // in real mode rather than a phantom key. The other 4 fields persist for real.
  await UpdateSettings(full as unknown as Parameters<typeof UpdateSettings>[0])
}

async function realFetchAIKey(): Promise<AIProviderKeyState> {
  // GetAIProviderKeyStatus reads the SAME encrypted settings-DB store SetAPIKeys
  // writes to (unlike GetSettings, which reads settings.json), so the save→read
  // round-trip is honest. The server decrypts only to mask — we only ever see
  // '(not set)' | '****' | 'abcd****wxyz', never the plaintext.
  const r = await GetAIProviderKeyStatus()
  const masked = str(r?.maskedKey) || '(not set)'
  return { maskedKey: masked, isSet: Boolean(r?.isSet) }
}

async function realSaveAIKey(key: string): Promise<void> {
  // SetAPIKeys persists per-key via SetSetting(key, value, 'apiKeys', encrypt=true)
  // — HKDF + AES-256-GCM at rest. It is NOT the full-overwrite saveUserSettings
  // path, so this cannot wipe other settings (unlike UpdateSettings). The server
  // ignores '****'/empty, so re-submitting a masked value is a safe no-op.
  // Plaintext crosses the wire (server encrypts); NEVER logged/echoed here.
  await SetAPIKeys({ [AI_PROVIDER_KEY_FIELD]: key })
}

/* ---- public switched API (viewmodel imports THESE) ---- */
export const fetchBusinessSettings = (): Promise<BusinessSettingsData> => pick(realFetch, mockFetch)()
export const updateBusinessSettings = (d: BusinessSettingsData): Promise<void> => pick(realUpdate, mockUpdate)(d)
export const fetchAIProviderKey = (): Promise<AIProviderKeyState> => pick(realFetchAIKey, mockFetchAIKey)()
export const saveAIProviderKey = (key: string): Promise<void> => pick(realSaveAIKey, mockSaveAIKey)(key)
