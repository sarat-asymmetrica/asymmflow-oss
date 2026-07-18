/* Pricing bridge — self-contained: types + mock + real + switch.
 * Only `SimulateMargin` is real on the old PricingScreen.svelte (InfraService,
 * confirmed signature `SimulateMargin(customer string, proposedMargin float64)
 * (*main.MarginSimulation, error)` — mapped 1:1 below). The customer list +
 * win-rate/regime data the old screen renders in its sidebar (`overallStats`)
 * is a hardcoded literal array in that component, not a fetch of any kind —
 * this bridge does NOT port that array as if it were real. `fetchPricing
 * Customers` stays mock-only until a real customer/win-rate endpoint exists
 * (ledgered in Pricing.parity.md, not faked). */

import { pick } from './runtime'
import { SimulateMargin } from '$wails/go/main/InfraService'
import { GetCustomerWinRates } from '$wails/go/main/App'

export type Regime = 'Premium' | 'PriceSensitive' | 'ValueBalanced'

export interface PricingCustomerRow {
  id: string
  name: string
  regime: Regime
  currentWinRate: number
  revenue: number
}

export interface MarginSimulationResult {
  customer: string
  proposedMargin: number
  currentWinRate: number
  projectedWinRate: number
  confidence: number
  recommendedAction: string
  warning: string
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

// Synthetic canon names (SYNTHETIC_IDENTITY.md) — mock/test data, exempt
// territory. Reuses the adversarial monster/RTL/blank entries other bridges
// draw from, plus a regime tag for the sweet-spot math below.
const CUSTOMERS: { name: string; regime: Regime }[] = [
  { name: 'Gulf Fabrication W.L.L.', regime: 'Premium' },
  { name: 'Manama Process Systems', regime: 'PriceSensitive' },
  { name: 'Al Dana Engineering Co.', regime: 'ValueBalanced' },
  {
    name: 'Interntional Establishment for Industrial & Petrochemical Instrumentation Services and General Trading (formerly Gulf Technical Calibration & Measurement Systems Company) W.L.L.',
    regime: 'Premium',
  },
  { name: 'المؤسسة الدولية لخدمات الأجهزة الصناعية والبتروكيماوية والتجارة العامة ذ.م.م', regime: 'ValueBalanced' },
  { name: 'Sitra Contracting', regime: 'PriceSensitive' },
  { name: 'X', regime: 'ValueBalanced' },
  { name: 'Bahrain Water Authority — Directorate of Operations & Maintenance, Section 7', regime: 'Premium' },
]

let cache: PricingCustomerRow[] | null = null

function generate(): PricingCustomerRow[] {
  const rand = lcg(20260714 + 9)
  return CUSTOMERS.map((c, i) => ({
    id: `pcust-${i + 1}`,
    name: c.name,
    regime: c.regime,
    currentWinRate: Math.round((0.15 + rand() * 0.55) * 1000) / 1000,
    revenue: Math.round(rand() * 900_000 * 100) / 100,
  }))
}

async function mockFetchCustomers(): Promise<PricingCustomerRow[]> {
  cache ??= generate()
  await sleep(220)
  return [...cache]
}

/** Deterministic projection from (customer, margin) — a synthetic stand-in
 * for the real regression model, not a claim about actual price elasticity.
 * Distance from the customer's regime-implied "sweet spot" margin erodes (or
 * lifts) win rate; PriceSensitive customers react fastest, Premium slowest. */
async function mockSimulate(customerName: string, proposedMargin: number): Promise<MarginSimulationResult> {
  cache ??= generate()
  const row = cache.find((c) => c.name === customerName)
  const current = row?.currentWinRate ?? 0.3
  const regime = row?.regime ?? 'ValueBalanced'
  const sweetSpot = regime === 'Premium' ? 0.3 : regime === 'PriceSensitive' ? 0.12 : 0.2
  const sensitivity = regime === 'PriceSensitive' ? 2.4 : regime === 'Premium' ? 0.8 : 1.4
  const delta = (proposedMargin - sweetSpot) * sensitivity
  const projected = Math.max(0.02, Math.min(0.97, current - delta))
  const confidence = Math.max(0.35, 0.9 - Math.abs(proposedMargin - sweetSpot))
  const action =
    projected < current - 0.05
      ? 'Reduce proposed margin — win rate erodes materially at this level'
      : projected > current + 0.03
        ? 'Room to hold or raise — win rate improves at this margin'
        : 'Margin is close to the regime sweet spot — hold'
  await sleep(180)
  return {
    customer: customerName,
    proposedMargin,
    currentWinRate: current,
    projectedWinRate: projected,
    confidence,
    recommendedAction: action,
    warning:
      proposedMargin > 0.45
        ? 'Margin exceeds the typical band for this segment — verify against contract minimums'
        : '',
  }
}

/* ---- real: win-rate list now comes from GetCustomerWinRates (owner ruling
 * G1.4) — a read-only aggregation over the real offer won/lost history. The old
 * screen HARDCODED this list; that literal was the bug. SimulateMargin is
 * genuinely real — wired straight through. ---- */

/** Pricing REGIME is a display-only strategy tag (badge + guidance). The backend
 * has no regime concept — the legacy screen hardcoded it. We derive it from the
 * customer's REAL win-rate instead of inventing a table: readily-won customers
 * are the least price-sensitive (Premium), rarely-won the most (PriceSensitive).
 * Presentation policy lives here, not in the pure Go aggregation. */
function deriveRegime(winRate: number): Regime {
  if (winRate >= 0.6) return 'Premium'
  if (winRate <= 0.3) return 'PriceSensitive'
  return 'ValueBalanced'
}

async function realFetchCustomers(): Promise<PricingCustomerRow[]> {
  const rows = await GetCustomerWinRates()
  return (rows ?? []).map((r) => {
    const rec = r as unknown as Record<string, unknown>
    const id = String(rec.customer_id ?? '') || `name:${String(rec.customer_name ?? '')}`
    const winRate = Number(rec.win_rate ?? 0)
    return {
      id,
      name: String(rec.customer_name ?? ''),
      regime: deriveRegime(winRate),
      currentWinRate: Number.isFinite(winRate) ? winRate : 0,
      revenue: Number(rec.won_value_bhd ?? 0),
    }
  })
}

async function realSimulate(customerName: string, proposedMargin: number): Promise<MarginSimulationResult> {
  const res = await SimulateMargin(customerName, proposedMargin)
  return {
    customer: res.customer,
    proposedMargin: res.proposed_margin,
    currentWinRate: res.current_win_rate,
    projectedWinRate: res.estimated_win_rate,
    confidence: res.confidence,
    recommendedAction: res.recommended_action,
    warning: res.warning ?? '',
  }
}

/* ---- public switched API (screen/viewmodel imports THESE) ---- */

export const fetchPricingCustomers = (): Promise<PricingCustomerRow[]> => pick(realFetchCustomers, mockFetchCustomers)()
export const simulateMargin = (customer: string, margin: number): Promise<MarginSimulationResult> =>
  pick(realSimulate, mockSimulate)(customer, margin)
