/* RFQs bridge module — self-contained: types + mock + real + switch.
 * Old screen: RFQScreen.svelte, `GetRFQs(100, 0)` called once, no Load-More —
 * treated here as an unpaged fetch (see recon-K1-A.md, RFQScreen section). */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import { DeleteRFQ, GetRFQs, UpdateRFQStatus } from '$wails/go/main/App'

export interface RFQRow {
  id: string
  number: string
  client: string
  project: string
  /** Query-time computed column on the old screen; origin unverified from the
   * struct alone (census gotcha) — best-effort parse of `product_details` here. */
  productCount: number
  value: number
  notes: string
  status: string
  createdAt: string
  /** PHANTOM: collected by the old create form and rendered as a column, but
   * `CreateRFQ` has no due-date parameter and `RFQData` has no column for it —
   * never persisted. Always '' from the real bridge. See RFQs.parity.md #5. */
  dueDate: string
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

const CLIENTS = [
  'Gulf Fabrication W.L.L.',
  'Manama Process Systems',
  'Al Dana Engineering Co.',
  'Interntional Establishment for Industrial & Petrochemical Instrumentation Services and General Trading (formerly Gulf Technical Calibration & Measurement Systems Company) W.L.L.',
  'المؤسسة الدولية لخدمات الأجهزة الصناعية والبتروكيماوية والتجارة العامة ذ.م.م',
  'Sitra Contracting',
  'X',
  'Bahrain Water Authority — Directorate of Operations & Maintenance, Section 7',
]
const PROJECTS = [
  'Flow metering skid replacement',
  'DCS migration — Phase 2',
  'Tank farm level instrumentation',
  '',
  'Turbine control retrofit',
  'Analyzer shelter upgrade',
]
const STATUSES = ['Pending', 'Qualified', 'Proposal', 'Negotiation', 'Won', 'Lost']

let cache: RFQRow[] | null = null

function generate(): RFQRow[] {
  const rand = lcg(20260714 + 1)
  const rows: RFQRow[] = []
  for (let i = 1; i <= 120; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 18)
    const year = 2025 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const created = `${year}-${pad(month, 2)}-${pad(day, 2)}`
    const due = `${year}-${pad(month, 2)}-${pad(Math.min(day + 21, 28), 2)}`

    // Adversarial seasoning at deterministic positions:
    const status = i % 97 === 0 ? 'UNKNOWN_STAGE' : STATUSES[Math.floor(r * STATUSES.length)]!
    const value =
      i % 89 === 0 ? 98765432109.876 : i % 53 === 0 ? 0.001 : Math.round(rand() * 1_500_000) / 100

    rows.push({
      id: `rfq-${i}`,
      number: `RFQ-${pad(i, 4)}`,
      client: CLIENTS[i % CLIENTS.length]!,
      project: PROJECTS[i % PROJECTS.length]!,
      productCount: i % 41 === 0 ? 0 : 1 + Math.floor(rand() * 12),
      value,
      notes: i % 31 === 0 ? '' : `Follow-up ${pad(i, 3)}`,
      status,
      createdAt: created,
      dueDate: i % 17 === 0 ? '' : due,
    })
  }
  return rows
}

async function mockFetch(): Promise<RFQRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockUpdateStage(id: string, stage: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((x) => x.id === id)
  if (row) row.status = stage
  await sleep(120)
}

async function mockDelete(id: string): Promise<void> {
  cache ??= generate()
  cache = cache.filter((x) => x.id !== id)
  await sleep(120)
}

/* ---- real: fetch WIRED, mutations are INTEG-gapped (honest throw) ---- */

/** Best-effort product count from the `product_details` JSON blob — the old
 * screen renders this column but it isn't a real struct field; unverified
 * without reading GetRFQs' query (out of recon scope, flagged in parity doc). */
function productCountFrom(raw: unknown): number {
  const s = str(raw)
  if (!s) return 0
  try {
    const parsed = JSON.parse(s)
    return Array.isArray(parsed) ? parsed.length : 0
  } catch {
    return 0
  }
}

function mapRFQ(r: Record<string, unknown>): RFQRow {
  const id = num(r.id)
  return {
    id: str(id),
    number: str(r.rfq_number) || `RFQ-${pad(id, 4)}`,
    client: str(r.client),
    project: str(r.project),
    productCount: productCountFrom(r.product_details),
    value: num(r.value),
    notes: str(r.notes),
    status: str(r.status),
    createdAt: goDate(r.created_at),
    dueDate: '', // phantom field — never persisted, see RFQRow.dueDate doc
  }
}

async function realFetch(): Promise<RFQRow[]> {
  const rows = await GetRFQs(200, 0)
  return (rows ?? []).map((x) => mapRFQ(x as unknown as Record<string, unknown>))
}

async function realUpdateStage(id: string, stage: string): Promise<void> {
  // The row's editable pipeline column is `status` (mapped from r.status), so
  // this routes to UpdateRFQStatus(id uint, status) — a free-form write on the
  // SAME column the row reads. UpdateRFQStage writes a separate `stage` column
  // under canonical-vocabulary state-machine validation that would REJECT this
  // screen's status values (e.g. Pending/Negotiation), so it is the wrong
  // binding here despite the adapter's historical name.
  await UpdateRFQStatus(num(id), stage)
}

async function realDelete(id: string): Promise<void> {
  // DeleteRFQ(id uint) — hard delete; the server owns referential-integrity guards.
  await DeleteRFQ(num(id))
}

/* ---- public switched API (descriptors import THESE) ---- */

export const fetchRFQs = (): Promise<RFQRow[]> => pick(realFetch, mockFetch)()
export const updateRFQStage = (id: string, stage: string): Promise<void> =>
  pick(realUpdateStage, mockUpdateStage)(id, stage)
export const deleteRFQ = (id: string): Promise<void> => pick(realDelete, mockDelete)(id)
