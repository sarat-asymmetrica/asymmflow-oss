/* Currency Rates bridge module — self-contained: types + mock + real +
 * switch (pricing.ts/suppliers.ts pattern). K4 SettingsScreen split: this is
 * the "FX Rates" tab standalone (see screens/parity/Settings.parity.md).
 *
 * Real bindings confirmed on App (wailsjs/go/main/App.d.ts):
 * GetActiveCurrencyRates / SetExchangeRate(currency, rate, asOfDate, source).
 * Model finance.CurrencyExchangeRate (wailsjs/go/models.ts:4587) has NO
 * "pair" concept — one rate per currency_code, implicitly against BHD (the
 * booking currency), which is why the row/form shape below is "currency +
 * rate + as-of date + source", not a two-currency pair. List-fetch is wired
 * for real; SetExchangeRate stays INTEG-gapped (see realSetRate below).
 * Synthetic-only data (SYNTHETIC_IDENTITY.md). */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import { GetActiveCurrencyRates } from '$wails/go/main/App'

export interface CurrencyRateRow {
  id: string
  currency: string
  rate: number
  asOfDate: string
  source: string
  notes: string
}

export interface CurrencyRateDraft {
  currency: string
  rate: number | null
  asOfDate: string
  source: string
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts pattern) ---- */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}

// Plausible BHD-quoted rates per currency (fixed-peg USD included for realism).
const CURRENCY_BASE: Record<string, number> = {
  USD: 0.376,
  EUR: 0.41,
  GBP: 0.48,
  SAR: 0.1002,
  AED: 0.1023,
  KWD: 1.226,
}
const SOURCES = ['CBB Reference', 'Manual Entry', 'Bank Quote', '']

let cache: CurrencyRateRow[] | null = null

function generate(): CurrencyRateRow[] {
  const rand = lcg(20260714 ^ 0xfa01)
  const rows: CurrencyRateRow[] = []
  let i = 0
  for (const [currency, base] of Object.entries(CURRENCY_BASE)) {
    i++
    const jitter = (rand() - 0.5) * 0.01
    rows.push({
      id: `rate-${i}`,
      currency,
      rate: Math.round((base + jitter) * 10000) / 10000,
      asOfDate: `2026-07-${String(1 + (i % 13)).padStart(2, '0')}`,
      source: SOURCES[i % SOURCES.length]!,
      notes: i % 3 === 0 ? 'Verified against CBB daily bulletin' : '',
    })
  }
  return rows
}

async function mockFetchAll(): Promise<CurrencyRateRow[]> {
  cache ??= generate()
  await sleep(200)
  return [...cache]
}

async function mockSetRate(draft: CurrencyRateDraft): Promise<void> {
  cache ??= generate()
  const existing = cache.find((r) => r.currency === draft.currency)
  if (existing) {
    existing.rate = draft.rate ?? existing.rate
    existing.asOfDate = draft.asOfDate
    existing.source = draft.source
  } else {
    cache.unshift({
      id: `rate-new-${cache.length + 1}`,
      currency: draft.currency,
      rate: draft.rate ?? 0,
      asOfDate: draft.asOfDate,
      source: draft.source,
      notes: '',
    })
  }
  await sleep(150)
}

/* ---- real: list-fetch WIRED, SetExchangeRate mutation INTEG-gapped ---- */
function mapRate(r: Record<string, unknown>): CurrencyRateRow {
  return {
    id: str(r.id),
    currency: str(r.currency_code),
    rate: num(r.rate),
    asOfDate: goDate(r.effective_from),
    source: str(r.set_by),
    notes: str(r.notes),
  }
}

async function realFetchAll(): Promise<CurrencyRateRow[]> {
  const rows = await GetActiveCurrencyRates()
  return (rows ?? []).map((r) => mapRate(r as unknown as Record<string, unknown>))
}

async function realSetRate(_draft: CurrencyRateDraft): Promise<void> {
  throw new Error(
    'INTEG gap: SetExchangeRate(currency, rate, asOfDate, source) takes a Go time.Time for asOfDate — ' +
      'wires at K5 once the form layer has a real date-to-time.Time bridge, not a naive string pass-through.',
  )
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchCurrencyRates = (): Promise<CurrencyRateRow[]> => pick(realFetchAll, mockFetchAll)()
export const setCurrencyRate = (d: CurrencyRateDraft): Promise<void> => pick(realSetRate, mockSetRate)(d)
