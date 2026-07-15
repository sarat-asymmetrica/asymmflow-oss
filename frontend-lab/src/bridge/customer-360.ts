/* Customer 360 bridge — self-contained: types + mock + real + switch.
 * Single-record view: unlike every other bridge module, there is no ledger/
 * entity list behind this screen, so the lab needs a small synthetic
 * directory (`fetchCustomer360Directory`) just to let the picker switch which
 * customer is shown — that directory isn't standing in for a real endpoint,
 * it's lab-only scaffolding (per the K4 brief, this screen scopes to exactly
 * two real bindings).
 *
 * Those two — `GetCustomer360` and `GetCustomer360Graph` (both on `App`) — are
 * now WIRED for real (INTEG). The mock originally invented a RICHER shape than
 * the backend provides (contact block, TRN, credit limit, a `regime` string);
 * per the owner's shape ruling the view was RESHAPED to the backend
 * (`main.Customer360Data` + `main.Customer360Graph`): drop the mock-invented
 * fields, surface what the bindings actually return. The mock now mirrors that
 * exact backend shape (synthetic canon only). */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import { GetCustomer360, GetCustomer360Graph } from '$wails/go/main/App'

export interface CustomerDirectoryEntry {
  id: string
  name: string
}

/** One row of `Customer360Data.recent_predictions` (butler.PredictionRecord).
 * `grade` stays `string` (not a closed union) so an unrecognized value flows
 * through to the UI's tone fallback rather than being narrowed away — same
 * posture as SerialTraceRow.status. */
export interface CustomerPrediction {
  id: string
  customerName: string
  grade: string
  predictedDays: number
  /** 0–1. */
  confidence: number
  /** 'YYYY-MM-DD' ('' for Go zero time). */
  createdAt: string
}

/** `Customer360Data.receivables_aging` (main.ReceivablesAgingSummary). */
export interface ReceivablesAging {
  current: number
  days30_60: number
  days60_90: number
  days90_120: number
  days120plus: number
  totalOutstanding: number
}

/** One row of `Customer360Data.payment_history` (main.PaymentHistoryEntry). */
export interface CustomerPayment {
  /** 'YYYY-MM-DD'. */
  paymentDate: string
  amountBhd: number
  invoiceNumber: string
  daysToPayment: number
  paymentMethod: string
}

/** One row of `Customer360Data.open_opportunities` (main.OpportunitySummary). */
export interface CustomerOpportunity {
  id: number
  project: string
  value: number
  status: string
  /** 'YYYY-MM-DD'. */
  createdAt: string
}

/** One row of `Customer360Data.recent_orders` (main.OrderSummary). */
export interface CustomerOrder {
  orderNumber: string
  /** 'YYYY-MM-DD'. */
  orderDate: string
  totalValueBhd: number
  status: string
}

/** Camel-case mirror of `main.Customer360Data` (wailsjs/go/models.ts:9356).
 * `code` has no backend source on this struct — it is honest-blanked ('') on
 * the real path (the mock still seeds one for picker variety). */
export interface Customer360Info {
  id: string
  code: string
  name: string
  customerType: string
  industry: string
  city: string
  country: string
  relationYears: number
  grade: string
  paymentTermsDays: number
  avgPaymentDays: number
  disputeCount: number
  isCreditBlocked: boolean
  requiresPrepayment: boolean
  /** Three-regime dynamics (0–1 each, loosely proportional). */
  r1: number
  r2: number
  r3: number
  lifetimeValue: number
  totalOrdersValue: number
  totalOrdersCount: number
  avgOrderValue: number
  /** 'YYYY-MM-DD' ('' if never ordered / Go zero time). */
  lastOrderDate: string
  hasAbbCompetition: boolean
  isEmergencyOnly: boolean
  receivablesAging: ReceivablesAging
  recentPredictions: CustomerPrediction[]
  paymentHistory: CustomerPayment[]
  openOpportunities: CustomerOpportunity[]
  recentOrders: CustomerOrder[]
}

/** Flat connections summary DERIVED from `main.Customer360Graph`. */
export interface CustomerConnections {
  totalConnections: number
  /** 0–1. Honest-blank (always 0): the backend graph carries no per-customer
   * centrality value (its `metrics` are graph-level density/avg-connections,
   * not a node score), so this field is never rendered — kept only for
   * shape stability. */
  centralityScore: number
  relatedProducts: string[]
  relatedSuppliers: string[]
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
const pad = (n: number, w: number): string => String(n).padStart(w, '0')

// Synthetic canon names (SYNTHETIC_IDENTITY.md) — mock/test data, exempt
// territory. Same monster/RTL/single-char/blank mix other bridges draw from.
const DIRECTORY: CustomerDirectoryEntry[] = [
  { id: 'c360-1', name: 'Gulf Fabrication W.L.L.' },
  { id: 'c360-2', name: 'Manama Process Systems' },
  { id: 'c360-3', name: 'Al Dana Engineering Co.' },
  {
    id: 'c360-4',
    name: 'Interntional Establishment for Industrial & Petrochemical Instrumentation Services and General Trading (formerly Gulf Technical Calibration & Measurement Systems Company) W.L.L.',
  },
  { id: 'c360-5', name: 'المؤسسة الدولية لخدمات الأجهزة الصناعية والبتروكيماوية والتجارة العامة ذ.م.م' },
  { id: 'c360-6', name: 'Sitra Contracting' },
  { id: 'c360-7', name: 'X' },
  { id: 'c360-8', name: 'Bahrain Water Authority — Directorate of Operations & Maintenance, Section 7' },
  { id: 'c360-9', name: '' },
]

const GRADES = ['A', 'B', 'C', 'D']
const TYPES = ['Direct', 'Distributor', 'End User', 'Government', '']
const INDUSTRIES = ['Oil & Gas', 'Construction', 'Manufacturing', 'Healthcare', 'Logistics', '']
const CITIES = ['Manama', 'Sitra', 'Riffa', 'Muharraq', '']
const COUNTRIES = ['Bahrain', 'Saudi Arabia', 'Qatar', '']
const PAYMENT_METHODS = ['Wire Transfer', 'Cheque', 'LC', 'Cash', '']
const ORDER_STATUSES = ['Delivered', 'InProduction', 'Shipped', 'UNKNOWN_STATE']
const OPP_STATUSES = ['Open', 'Quoted', 'Negotiation', 'UNKNOWN_STAGE']
const PROJECTS = [
  'Refinery Instrumentation Upgrade',
  'Water Treatment Calibration',
  '',
  'Extremely Long Project Title That Overflows Every Reasonable Column Width Allotted To A Single Line Of Detail Text In The UI',
]
const PRODUCT_CODES = ['PT-100-DIN', 'FV-2200-SS', 'RTD-PT1000', 'XMTR-4-20MA', 'GV-ANSI150', 'CAL-BENCH-A']
const SUPPLIER_NAMES = ['Endress+Hauser Bahrain', 'Emerson Gulf FZE', 'Yokogawa Middle East', 'Rotork Bahrain W.L.L.', '']

function indexFromId(id: string): number {
  const m = /(\d+)$/.exec(id)
  return m ? Number(m[1]) : 1
}

function generateInfo(id: string): Customer360Info {
  const i = indexFromId(id)
  const dir = DIRECTORY.find((d) => d.id === id)
  const rand = lcg(20260714 + i * 31)
  const name = dir?.name ?? `Customer ${i}`
  const grade = i % 9 === 0 ? 'UNKNOWN_GRADE' : GRADES[Math.floor(rand() * GRADES.length)]!

  // Adversarial money at deterministic positions: zero, negative, huge.
  const lifetimeValue = i % 7 === 0 ? 0 : i % 12 === 0 ? -18450.5 : Math.round(rand() * 2_500_000 * 1000) / 1000
  const totalOrdersCount = i % 6 === 0 ? 0 : Math.floor(rand() * 240)
  const totalOrdersValue = totalOrdersCount === 0 ? 0 : Math.round(rand() * 3_200_000 * 1000) / 1000
  const avgOrderValue = totalOrdersCount === 0 ? 0 : Math.round((totalOrdersValue / totalOrdersCount) * 1000) / 1000
  const avgPaymentDays = i % 8 === 0 ? 999 : Math.floor(rand() * 75)
  const disputeCount = i % 5 === 0 ? 0 : Math.floor(rand() * 6)

  // Three-regime dynamics — floats, occasionally all-zero (unclassified).
  const zeroDyn = i % 10 === 0
  const r1 = zeroDyn ? 0 : Math.round(rand() * 1000) / 1000
  const r2 = zeroDyn ? 0 : Math.round(rand() * 1000) / 1000
  const r3 = zeroDyn ? 0 : Math.round(rand() * 1000) / 1000

  const receivablesAging: ReceivablesAging = {
    current: i % 7 === 0 ? 0 : Math.round(rand() * 90_000 * 1000) / 1000,
    days30_60: Math.round(rand() * 60_000 * 1000) / 1000,
    days60_90: Math.round(rand() * 40_000 * 1000) / 1000,
    days90_120: i % 11 === 0 ? 999_999_999.999 : Math.round(rand() * 25_000 * 1000) / 1000,
    days120plus: Math.round(rand() * 15_000 * 1000) / 1000,
    totalOutstanding: 0, // filled below
  }
  receivablesAging.totalOutstanding =
    Math.round(
      (receivablesAging.current +
        receivablesAging.days30_60 +
        receivablesAging.days60_90 +
        receivablesAging.days90_120 +
        receivablesAging.days120plus) *
        1000,
    ) / 1000

  const recentPredictions: CustomerPrediction[] = []
  const predCount = i % 6 === 0 ? 0 : 3 + Math.floor(rand() * 5)
  for (let p = 1; p <= predCount; p++) {
    const day = 1 + Math.floor(rand() * 27)
    const gr = (i + p) % 11 === 0 ? 'UNKNOWN_GRADE' : GRADES[Math.floor(rand() * GRADES.length)]!
    recentPredictions.push({
      id: `pred-${id}-${p}`,
      customerName: name,
      grade: gr,
      predictedDays: i % 13 === 0 ? 999 : Math.floor(rand() * 90),
      confidence: Math.round((0.4 + rand() * 0.58) * 1000) / 1000,
      createdAt: `2026-${pad(1 + Math.floor(rand() * 12), 2)}-${pad(day, 2)}`,
    })
  }
  recentPredictions.sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1)) // newest first

  const paymentHistory: CustomerPayment[] = []
  const payCount = i % 4 === 0 ? 0 : 2 + Math.floor(rand() * 6)
  for (let p = 1; p <= payCount; p++) {
    const month = 1 + Math.floor(rand() * 12)
    const day = 1 + Math.floor(rand() * 27)
    paymentHistory.push({
      paymentDate: `2025-${pad(month, 2)}-${pad(day, 2)}`,
      amountBhd: p % 5 === 0 ? 0 : Math.round(rand() * 120_000 * 1000) / 1000,
      invoiceNumber: p % 7 === 0 ? '' : `INV-2025-${pad((i * 13 + p) % 9000 || 1, 4)}`,
      daysToPayment: p % 8 === 0 ? 999 : Math.floor(rand() * 120),
      paymentMethod: PAYMENT_METHODS[(i + p) % PAYMENT_METHODS.length]!,
    })
  }
  paymentHistory.sort((a, b) => (a.paymentDate < b.paymentDate ? 1 : -1))

  const openOpportunities: CustomerOpportunity[] = []
  const oppCount = i % 5 === 0 ? 0 : 1 + Math.floor(rand() * 4)
  for (let o = 1; o <= oppCount; o++) {
    const month = 1 + Math.floor(rand() * 12)
    const day = 1 + Math.floor(rand() * 27)
    openOpportunities.push({
      id: i * 100 + o,
      project: PROJECTS[(i + o) % PROJECTS.length]!,
      value: o % 6 === 0 ? 0 : Math.round(rand() * 900_000 * 1000) / 1000,
      status: OPP_STATUSES[(i + o) % OPP_STATUSES.length]!,
      createdAt: `2026-${pad(month, 2)}-${pad(day, 2)}`,
    })
  }

  const recentOrders: CustomerOrder[] = []
  const ordCount = totalOrdersCount === 0 ? 0 : 1 + Math.floor(rand() * 5)
  for (let o = 1; o <= ordCount; o++) {
    const month = 1 + Math.floor(rand() * 12)
    const day = 1 + Math.floor(rand() * 27)
    recentOrders.push({
      orderNumber: `ORD-2026-${pad((i * 17 + o) % 9000 || 1, 4)}`,
      orderDate: `2026-${pad(month, 2)}-${pad(day, 2)}`,
      totalValueBhd: o % 7 === 0 ? 0 : Math.round(rand() * 300_000 * 1000) / 1000,
      status: ORDER_STATUSES[(i + o) % ORDER_STATUSES.length]!,
    })
  }
  recentOrders.sort((a, b) => (a.orderDate < b.orderDate ? 1 : -1))

  const lastOrderDate = recentOrders[0]?.orderDate ?? (i % 6 === 0 ? '' : `2026-0${1 + (i % 9)}-15`)

  return {
    id,
    code: `C-${pad(i, 4)}`,
    name,
    customerType: TYPES[i % TYPES.length]!,
    industry: INDUSTRIES[i % INDUSTRIES.length]!,
    city: CITIES[i % CITIES.length]!,
    country: COUNTRIES[i % COUNTRIES.length]!,
    relationYears: i % 10 === 0 ? 0 : Math.floor(rand() * 15),
    grade,
    paymentTermsDays: i % 9 === 0 ? 0 : [30, 45, 60, 90][i % 4]!,
    avgPaymentDays,
    disputeCount,
    isCreditBlocked: i % 5 === 0,
    requiresPrepayment: i % 4 === 0,
    r1,
    r2,
    r3,
    lifetimeValue,
    totalOrdersValue,
    totalOrdersCount,
    avgOrderValue,
    lastOrderDate,
    hasAbbCompetition: i % 3 === 0,
    isEmergencyOnly: i % 7 === 0,
    receivablesAging,
    recentPredictions,
    paymentHistory,
    openOpportunities,
    recentOrders,
  }
}

function generateConnections(id: string): CustomerConnections {
  const i = indexFromId(id)
  const rand = lcg(20260714 + i * 53)
  const zero = i % 6 === 0 // a customer with no recorded connections — exercises the empty chip lists
  const products = zero ? [] : PRODUCT_CODES.filter((_, idx) => (idx + i) % 2 === 0)
  const suppliers = zero ? [] : SUPPLIER_NAMES.filter((_, idx) => (idx + i) % 2 === 1)
  return {
    totalConnections: zero ? 0 : products.length + suppliers.length + Math.floor(rand() * 5),
    centralityScore: 0, // honest-blank — see CustomerConnections.centralityScore
    relatedProducts: products,
    relatedSuppliers: suppliers,
  }
}

async function mockCustomer360(id: string): Promise<Customer360Info> {
  await sleep(220)
  return generateInfo(id)
}

async function mockCustomer360Connections(id: string): Promise<CustomerConnections> {
  await sleep(180)
  return generateConnections(id)
}

/* ---- real: WIRED to the two confirmed bindings, reshaped to backend ---- */

/** Narrows an unknown JSON array field to an iterable of records. */
function asRecords(v: unknown): Record<string, unknown>[] {
  return Array.isArray(v) ? (v as Record<string, unknown>[]) : []
}

function mapPrediction(r: Record<string, unknown>): CustomerPrediction {
  return {
    id: str(r.id),
    customerName: str(r.customer_name),
    grade: str(r.grade),
    predictedDays: num(r.predicted_days),
    confidence: num(r.confidence),
    createdAt: goDate(r.created_at),
  }
}

function mapAging(r: unknown): ReceivablesAging {
  const a = (r ?? {}) as Record<string, unknown>
  return {
    current: num(a.current),
    days30_60: num(a.days_30_60),
    days60_90: num(a.days_60_90),
    days90_120: num(a.days_90_120),
    days120plus: num(a.days_120_plus),
    totalOutstanding: num(a.total_outstanding),
  }
}

function mapPayment(r: Record<string, unknown>): CustomerPayment {
  return {
    paymentDate: goDate(r.payment_date),
    amountBhd: num(r.amount_bhd),
    invoiceNumber: str(r.invoice_number),
    daysToPayment: num(r.days_to_payment),
    paymentMethod: str(r.payment_method),
  }
}

function mapOpportunity(r: Record<string, unknown>): CustomerOpportunity {
  return {
    id: num(r.id),
    project: str(r.project),
    value: num(r.value),
    status: str(r.status),
    createdAt: goDate(r.created_at),
  }
}

function mapOrder(r: Record<string, unknown>): CustomerOrder {
  return {
    orderNumber: str(r.order_number),
    orderDate: goDate(r.order_date),
    totalValueBhd: num(r.total_value_bhd),
    status: str(r.status),
  }
}

async function realCustomer360(id: string): Promise<Customer360Info> {
  const d = (await GetCustomer360(id)) as unknown as Record<string, unknown>
  return {
    id: str(d.customer_id) || id,
    code: '', // honest-blank: main.Customer360Data carries no document code
    name: str(d.business_name),
    customerType: str(d.customer_type),
    industry: str(d.industry),
    city: str(d.city),
    country: str(d.country),
    relationYears: num(d.relation_years),
    grade: str(d.current_grade),
    paymentTermsDays: num(d.payment_terms_days),
    avgPaymentDays: num(d.avg_payment_days),
    disputeCount: num(d.dispute_count),
    isCreditBlocked: Boolean(d.is_credit_blocked),
    requiresPrepayment: Boolean(d.requires_prepayment),
    r1: num(d.r1),
    r2: num(d.r2),
    r3: num(d.r3),
    lifetimeValue: num(d.customer_lifetime_value),
    totalOrdersValue: num(d.total_orders_value),
    totalOrdersCount: num(d.total_orders_count),
    avgOrderValue: num(d.avg_order_value),
    lastOrderDate: goDate(d.last_order_date),
    hasAbbCompetition: Boolean(d.has_abb_competition),
    isEmergencyOnly: Boolean(d.is_emergency_only),
    receivablesAging: mapAging(d.receivables_aging),
    recentPredictions: asRecords(d.recent_predictions).map(mapPrediction),
    paymentHistory: asRecords(d.payment_history).map(mapPayment),
    openOpportunities: asRecords(d.open_opportunities).map(mapOpportunity),
    recentOrders: asRecords(d.recent_orders).map(mapOrder),
  }
}

/** Derives the flat connections summary from the node/edge graph:
 * the center node is the customer itself (id === customer_id, or type
 * 'customer') and is excluded; totalConnections counts the remaining
 * entities; related products/suppliers are their labels bucketed by type. */
async function realCustomer360Connections(id: string): Promise<CustomerConnections> {
  const g = (await GetCustomer360Graph(id)) as unknown as {
    customer_id?: unknown
    entities?: unknown
  }
  const centerId = str(g.customer_id) || id
  const nonCenter = asRecords(g.entities).filter(
    (e) => str(e.id) !== centerId && str(e.type) !== 'customer',
  )
  const labelsOfType = (t: string): string[] =>
    nonCenter.filter((e) => str(e.type) === t).map((e) => str(e.label))
  return {
    totalConnections: nonCenter.length,
    centralityScore: 0, // honest-blank — no per-customer centrality in the graph
    relatedProducts: labelsOfType('product'),
    relatedSuppliers: labelsOfType('supplier'),
  }
}

/* ---- public switched API (viewmodel imports THESE) ---- */

/** Lab-only picker directory — not a stand-in for a real endpoint (see file
 * header); always synthetic, never routed through `pick`. */
export async function fetchCustomer360Directory(): Promise<CustomerDirectoryEntry[]> {
  await sleep(120)
  return [...DIRECTORY]
}

export const fetchCustomer360 = (id: string): Promise<Customer360Info> =>
  pick(realCustomer360, mockCustomer360)(id)

export const fetchCustomer360Connections = (id: string): Promise<CustomerConnections> =>
  pick(realCustomer360Connections, mockCustomer360Connections)(id)
