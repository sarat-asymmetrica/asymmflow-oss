/* Offers bridge module — self-contained: types + mock + real + switch.
 * Old screen: OffersScreen.svelte, `GetAllOffers()` — no params, no paging.
 * See recon-K1-A.md OffersScreen section: Create/Edit eject to CostingSheet,
 * Won/Lost are financial-hot-zone document-creating/terminal actions, PDF is
 * a bindings-only row action — every mutating capability is LEDGER here. */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import { GetAllOffers } from '$wails/go/main/App'
import type { Tone } from '$kernel/tones'

export interface OfferRow {
  id: string
  number: string
  revisionNumber: number
  customer: string
  quotationDate: string
  validityDate: string
  value: number
  estimatedMargin: number
  /** Effective stage: raw `stage` from the struct, UNLESS the offer is still
   * live (RFQ/Quoted) and `validityDate` has passed — then 'Expired', mirroring
   * the old screen's `isOfferExpired()` (which treats the client-computed
   * signal and the backend's `AutoExpireOffers` job as one truth). */
  stage: string
  division: string
  /** Independent of the stage badge — the old screen color-codes the "Valid
   * Until" cell itself when the offer is within its expiry window. */
  isExpiringSoon: boolean
}

const EXPIRING_SOON_DAYS = 14

function daysUntil(dateStr: string): number | null {
  if (!dateStr) return null
  const target = new Date(dateStr).getTime()
  if (Number.isNaN(target)) return null
  const now = new Date()
  const today = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())
  return Math.round((target - today) / 86_400_000)
}

function effectiveStage(rawStage: string, validityDate: string): string {
  if (rawStage === 'Won' || rawStage === 'Lost' || rawStage === 'Expired') return rawStage
  const days = daysUntil(validityDate)
  return days != null && days < 0 ? 'Expired' : rawStage
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts pattern) ---- */

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number) {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number) => String(n).padStart(w, '0')

const DIVISIONS = ['Acme Instrumentation', 'Beacon Controls']
const CUSTOMERS = [
  'Gulf Fabrication W.L.L.',
  'Manama Process Systems',
  'Al Dana Engineering Co.',
  'Interntional Establishment for Industrial & Petrochemical Instrumentation Services and General Trading (formerly Gulf Technical Calibration & Measurement Systems Company) W.L.L.',
  'المؤسسة الدولية لخدمات الأجهزة الصناعية والبتروكيماوية والتجارة العامة ذ.م.م',
  'Sitra Contracting',
  'X',
  'Bahrain Water Authority — Directorate of Operations & Maintenance, Section 7',
]
const RAW_STAGES = ['RFQ', 'Quoted', 'Won', 'Lost']

let cache: OfferRow[] | null = null

function generate(): OfferRow[] {
  const rand = lcg(20260714 + 2)
  const rows: OfferRow[] = []
  for (let i = 1; i <= 180; i++) {
    const r = rand()
    const rawStage = i % 101 === 0 ? 'UNKNOWN_STAGE' : RAW_STAGES[Math.floor(r * RAW_STAGES.length)]!
    const quoteOffsetDays = -Math.floor(rand() * 240)
    const quotationDate = shiftDate(quoteOffsetDays)
    // Spread validity around "now" so the fixture always has expired,
    // expiring-soon, and comfortably-valid rows without re-seeding daily.
    const validityOffsetDays = -120 + Math.floor(rand() * 260)
    const validityDate = i % 19 === 0 ? '' : shiftDate(validityOffsetDays)
    const value =
      i % 89 === 0 ? 76543210987.654 : i % 53 === 0 ? 0.001 : Math.round(rand() * 2_000_000) / 100

    const stage = effectiveStage(rawStage, validityDate)
    rows.push({
      id: `off-${i}`,
      number: `OFR-${pad(i, 4)}`,
      revisionNumber: i % 23 === 0 ? 2 : 0,
      customer: CUSTOMERS[i % CUSTOMERS.length]!,
      quotationDate,
      validityDate,
      value,
      estimatedMargin: Math.round((rand() * 0.35 - 0.05) * 1000) / 1000,
      stage,
      division: DIVISIONS[i % DIVISIONS.length]!,
      isExpiringSoon:
        stage !== 'Expired' &&
        stage !== 'Won' &&
        stage !== 'Lost' &&
        (() => {
          const d = daysUntil(validityDate)
          return d != null && d >= 0 && d <= EXPIRING_SOON_DAYS
        })(),
    })
  }
  return rows
}

function shiftDate(offsetDays: number): string {
  const d = new Date()
  d.setUTCDate(d.getUTCDate() + offsetDays)
  return d.toISOString().slice(0, 10)
}

async function mockFetch(): Promise<OfferRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

/* ---- real: fetch WIRED, mutations are INTEG-gapped (honest throw) ---- */

function mapOffer(r: Record<string, unknown>): OfferRow {
  const validityDate = goDate(r.validity_date)
  const rawStage = str(r.stage)
  const stage = effectiveStage(rawStage, validityDate)
  return {
    id: str(r.id),
    number: str(r.offer_number),
    revisionNumber: num(r.revision_number),
    customer: str(r.customer_name),
    quotationDate: goDate(r.quotation_date),
    validityDate,
    value: num(r.total_value_bhd),
    estimatedMargin: num(r.estimated_margin),
    stage,
    division: str(r.division),
    isExpiringSoon:
      stage !== 'Expired' &&
      stage !== 'Won' &&
      stage !== 'Lost' &&
      (() => {
        const d = daysUntil(validityDate)
        return d != null && d >= 0 && d <= EXPIRING_SOON_DAYS
      })(),
  }
}

async function realFetch(): Promise<OfferRow[]> {
  const rows = await GetAllOffers()
  return (rows ?? []).map((x) => mapOffer(x as unknown as Record<string, unknown>))
}

/* ---- public switched API (descriptors import THESE) ---- */

export const fetchOffers = (): Promise<OfferRow[]> => pick(realFetch, mockFetch)()

/** Tone for the "Valid Until" cell — independent 2nd signal alongside the
 * status badge (recon: the richest 2-signal cell in the K1-A cluster). */
export function validityTone(row: OfferRow): Tone {
  if (row.stage === 'Expired') return 'danger'
  if (row.isExpiringSoon) return 'warning'
  return 'neutral'
}
