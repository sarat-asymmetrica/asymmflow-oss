/* Customer 360 bridge — self-contained: types + mock + real + switch.
 * Single-record view: unlike every other bridge module, there is no ledger/
 * entity list behind this screen, so the lab needs a small synthetic
 * directory (`fetchCustomer360Directory`) just to let the picker switch which
 * customer is shown — that directory isn't standing in for a real endpoint,
 * it's lab-only scaffolding (per the K4 brief, this screen scopes to exactly
 * two real bindings).
 *
 * Those two — `GetCustomer360` (`App`) and `GetCustomer360Graph`
 * (`CRMService`) — are confirmed real per recon-K4.md, but INTEG-gapped per
 * the K4 brief (customer financial history + predictive grading, read-only
 * moderate-sensitivity data): the mock supplies full synthetic data end to
 * end, the real side throws naming the exact call it stands in for. */

import { pick } from './runtime'

export interface CustomerDirectoryEntry {
  id: string
  name: string
}

export interface GradePrediction {
  id: string
  date: string
  /** Mirrors a real grade vocabulary loosely — kept as `string` (not a closed
   * union) so an unrecognized value flows through to the UI's tone fallback
   * rather than being narrowed away, same posture as SerialTraceRow.status. */
  grade: string
  /** 0–1. */
  confidence: number
  predictedDays: number
}

export interface Customer360Info {
  id: string
  code: string
  name: string
  regime: string
  lifetimeValue: number
  avgPaymentDays: number
  disputeCount: number
  contact: {
    contactPerson: string
    phone: string
    email: string
    address: string
  }
  commercial: {
    paymentTerms: string
    creditLimit: number
    trn: string
    industry: string
    relationYears: number
  }
  predictions: GradePrediction[]
}

export interface CustomerConnections {
  totalConnections: number
  /** 0–1. */
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

const REGIMES = ['Prompt', 'Standard', 'Slow', 'AtRisk']
const GRADES = ['A', 'B', 'C', 'D']
const INDUSTRIES = ['Oil & Gas', 'Construction', 'Manufacturing', 'Healthcare', 'Logistics', '']
const TERMS = ['Net 30', 'Net 60', 'Advance', 'LC 90 days']
const CONTACT_NAMES = ['Aisha Al-Rumaihi', 'Mohammed Bucheeri', 'Fatima Al-Zayani', 'Yusuf Kanoo', '']
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
  const regime = i % 9 === 0 ? 'UNKNOWN_REGIME' : REGIMES[Math.floor(rand() * REGIMES.length)]!
  const lifetimeValue = i % 7 === 0 ? 0 : Math.round(rand() * 2_500_000 * 100) / 100
  const avgPaymentDays = i % 8 === 0 ? 999 : Math.floor(rand() * 75)
  const disputeCount = i % 5 === 0 ? 0 : Math.floor(rand() * 6)

  const predictions: GradePrediction[] = []
  const predCount = i % 6 === 0 ? 0 : 3 + Math.floor(rand() * 5)
  for (let p = 1; p <= predCount; p++) {
    const monthIdx = Math.floor(rand() * 12)
    const day = 1 + Math.floor(rand() * 27)
    const grade = (i + p) % 11 === 0 ? 'UNKNOWN_GRADE' : GRADES[Math.floor(rand() * GRADES.length)]!
    predictions.push({
      id: `pred-${id}-${p}`,
      date: `2026-${pad(1 + (monthIdx % 12), 2)}-${pad(day, 2)}`,
      grade,
      confidence: Math.round((0.4 + rand() * 0.58) * 1000) / 1000,
      predictedDays: i % 13 === 0 ? 999 : Math.floor(rand() * 90),
    })
  }
  predictions.sort((a, b) => (a.date < b.date ? 1 : -1)) // newest first

  return {
    id,
    code: `C-${pad(i, 4)}`,
    name,
    regime,
    lifetimeValue,
    avgPaymentDays,
    disputeCount,
    contact: {
      contactPerson: i % 4 === 0 ? '' : CONTACT_NAMES[i % CONTACT_NAMES.length]!,
      phone: i % 6 === 0 ? '' : `+973 ${pad(Math.floor(rand() * 99999999), 8)}`,
      email:
        i % 9 === 0
          ? 'accounts.receivable.department.regional.office@extremely-long-corporate-domain-name.example.com.bh'
          : i % 6 === 0
            ? ''
            : `finance${i}@example.bh`,
      address:
        i % 5 === 0 ? '' : `Building ${100 + i}, Road ${1000 + i * 3}, Block ${300 + i}, Manama, Kingdom of Bahrain`,
    },
    commercial: {
      paymentTerms: TERMS[i % TERMS.length]!,
      creditLimit: i % 11 === 0 ? 999999999.999 : Math.round(rand() * 1_500_000 * 100) / 100,
      trn: i % 7 === 0 ? '' : `TRN-${pad(200000 + i, 9)}`,
      industry: INDUSTRIES[i % INDUSTRIES.length]!,
      relationYears: i % 10 === 0 ? 0 : Math.floor(rand() * 15),
    },
    predictions,
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
    centralityScore: zero ? 0 : Math.round(rand() * 1000) / 1000,
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

/* ---- real: INTEG-gapped, naming the exact bindings for K5 ---- */

async function realCustomer360(_id: string): Promise<Customer360Info> {
  void _id
  // INTEG SHAPE-DIVERGENCE (not a straight swap): the real GetCustomer360 →
  // main.Customer360Data is NARROWER than this view's Customer360Info — it
  // carries no contactPerson/phone/email/address, no TRN, no creditLimit, and
  // no `regime` string (only current_grade + R1/R2/R3), while adding fields
  // this view doesn't show (receivables aging, payment history, open opps,
  // recent orders). Wiring it verbatim would blank half the panels. Deferred
  // for an owner shape decision (reshape Customer360Info to the backend, or
  // compose a supplementary customer-detail fetch for contact/TRN/credit) —
  // see the I2 wave report. Stays honest-synthetic until then (read-only, no
  // persistence risk).
  throw new Error('INTEG gap: GetCustomer360 — real main.Customer360Data shape diverges from this view; needs owner shape decision (see I2 report)')
}

async function realCustomer360Connections(_id: string): Promise<CustomerConnections> {
  void _id
  // INTEG SHAPE-DIVERGENCE: GetCustomer360Graph → main.Customer360Graph is a
  // node/edge graph (GraphEntity[] + GraphRelation[]), not this view's flat
  // {totalConnections, centralityScore, relatedProducts[], relatedSuppliers[]}
  // summary — connections would have to be DERIVED from the graph (count nodes,
  // bucket related entities by type). Deferred with GetCustomer360 above.
  throw new Error('INTEG gap: GetCustomer360Graph — real main.Customer360Graph is a node/edge graph; connections need derivation (see I2 report)')
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
