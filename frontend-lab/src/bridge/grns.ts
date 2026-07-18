/* GRNs bridge module — self-contained: types + mock + real + switch.
 * K1 scope note: GRNScreen's real actions (Receive from PO, QC Review,
 * Complete) are all ledgered (SLOT/financial hot-zone — see
 * screens/parity/GRNs.parity.md), so this module is read-only: fetch +
 * mapping only, no mutations to switch. */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import { CompleteGRN, ListGRNs, UpdateGRNQCStatus } from '$wails/go/main/App'
import { actingUserId } from '../stores/session.svelte'

export interface GRNRow {
  id: string
  grnNumber: string
  purchaseOrderId: string
  poNumber: string
  supplierName: string
  receivedDate: string
  receivedBy: string
  qcStatus: string
  qcDate: string
  qcBy: string
  qcNotes: string
  itemsCount: number
  totalReceived: number
  totalAccepted: number
  totalRejected: number
  /** 0–1 fraction as returned by the backend — mapper/mock do NOT ×100. */
  acceptanceRate: number
  isCompleted: boolean
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts) ---- */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')

const SUPPLIERS = [
  'Bahrain Precision Instruments W.L.L.',
  'Gulf Valve & Actuator Trading Co.',
  'Al Manar Industrial Supplies',
  'International Establishment for Process Control Equipment, Calibration Services, Spare Parts Distribution and General Engineering Trading (formerly Gulf Technical Instrumentation Company) W.L.L.',
  'شركة الخليج للتوريدات الصناعية والمعايرة ذ.م.م',
  'Sitra Metal Works',
  'X',
  'Bahrain Ports Authority — Procurement & Logistics Directorate, Warehouse 4',
]

const QC_STATUSES = ['Pending', 'Passed', 'Failed', 'Partial']

let cache: GRNRow[] | null = null

function generate(): GRNRow[] {
  const rand = lcg(20260714)
  const rows: GRNRow[] = []
  const n = 220
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 20)
    const year = 2024 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const received = `${year}-${pad(month, 2)}-${pad(day, 2)}`

    const qcStatus = i % 97 === 0 ? 'UNKNOWN_STATE' : QC_STATUSES[i % QC_STATUSES.length]!
    const qcResolved = qcStatus === 'Passed' || qcStatus === 'Failed' || qcStatus === 'Partial'

    const totalReceived =
      i % 89 === 0 ? 987654321.987 : i % 53 === 0 ? 0.001 : Math.round(rand() * 5000) / 10
    // Adversarial acceptance-rate seasoning straddling the threshold-colour
    // boundaries (≥0.95 green, ≥0.80 amber, <0.80 red — see GRNs.descriptor.ts).
    let acceptanceRate: number
    if (i % 43 === 0) acceptanceRate = 0
    else if (i % 47 === 0) acceptanceRate = 1
    else if (i % 31 === 0) acceptanceRate = 0.9499
    else if (i % 37 === 0) acceptanceRate = 0.95
    else if (i % 29 === 0) acceptanceRate = 0.7999
    else if (i % 23 === 0) acceptanceRate = 0.8
    else acceptanceRate = qcStatus === 'Failed' ? Math.round(rand() * 4000) / 10000 : Math.round((6000 + rand() * 4000)) / 10000
    const totalAccepted = qcResolved ? Math.round(totalReceived * acceptanceRate * 1000) / 1000 : 0
    const totalRejected = qcResolved ? Math.round((totalReceived - totalAccepted) * 1000) / 1000 : 0

    rows.push({
      id: `grn-${i}`,
      grnNumber: i % 71 === 0 ? `GRN-${year}-${pad(i, 40)}` : `GRN-${year}-${pad(i, 4)}`,
      purchaseOrderId: `po-${((i * 7) % 260) + 1}`,
      poNumber: `PO-${year}-${pad(((i * 7) % 260) + 1, 4)}`,
      supplierName: i % 67 === 0 ? '' : SUPPLIERS[i % SUPPLIERS.length]!,
      receivedDate: received,
      receivedBy: i % 19 === 0 ? '' : `store.keeper.${(i % 12) + 1}@example.bh`,
      qcStatus,
      qcDate: qcResolved ? received : '',
      qcBy: qcResolved && i % 13 !== 0 ? `qc.inspector.${(i % 6) + 1}@example.bh` : '',
      qcNotes: i % 17 === 0 ? '' : i % 31 === 0 ? 'Rejected units: crushed packaging, moisture ingress on 3 crates, awaiting supplier credit note.' : 'OK',
      itemsCount: 1 + Math.floor(r * 40),
      totalReceived,
      totalAccepted,
      totalRejected,
      acceptanceRate,
      isCompleted: qcResolved && qcStatus !== 'Failed' && i % 3 !== 0,
    })
  }
  return rows
}

async function mockFetch(): Promise<GRNRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

const todayIso = (): string => new Date().toISOString().slice(0, 10)

/** QC Review (R5): stamp the QC verdict + notes. Passed/Failed/Pending. */
async function mockQCReview(id: string, status: string, notes: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) {
    row.qcStatus = status
    row.qcNotes = notes
    row.qcBy = actingUserId()
    row.qcDate = todayIso()
  }
  await sleep(140)
}

/** Complete (R5): close the GRN (server updates the linked PO + defaults a
 * still-Pending QC to Passed). Server refuses if QC has Failed. */
async function mockComplete(id: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) {
    row.isCompleted = true
    if (row.qcStatus === 'Pending') row.qcStatus = 'Passed'
  }
  await sleep(140)
}

/* ---- real: fetch WIRED (no mutations to gap — this module is read-only) ---- */
function mapGRN(r: Record<string, unknown>): GRNRow {
  return {
    id: str(r.id),
    grnNumber: str(r.grn_number),
    purchaseOrderId: str(r.purchase_order_id),
    poNumber: str(r.po_number),
    supplierName: str(r.supplier_name),
    receivedDate: goDate(r.received_date),
    receivedBy: str(r.received_by),
    qcStatus: str(r.qc_status) || 'Pending',
    qcDate: goDate(r.qc_date),
    qcBy: str(r.qc_by),
    qcNotes: str(r.qc_notes),
    itemsCount: num(r.items_count),
    totalReceived: num(r.total_received),
    totalAccepted: num(r.total_accepted),
    totalRejected: num(r.total_rejected),
    acceptanceRate: num(r.acceptance_rate),
    isCompleted: !!r.is_completed,
  }
}

async function realFetch(): Promise<GRNRow[]> {
  // Flat load, mirrors GRNScreen.svelte's mount call — the qcStatus param
  // exists server-side but the old screen filters client-side after the
  // first load; K1 keeps that behavior (filter chip below, not a re-fetch).
  const rows = await ListGRNs(1000, 0, '')
  return (rows ?? []).map((x) => mapGRN(x as unknown as Record<string, unknown>))
}

async function realQCReview(id: string, status: string, notes: string): Promise<void> {
  // UpdateGRNQCStatus(id, status, notes, qcBy). qcBy is re-derived server-side
  // from the session (client value ignored, kept for binding stability) — we
  // still pass the session actor, never a caller-supplied identity.
  await UpdateGRNQCStatus(id, status, notes, actingUserId())
}

async function realComplete(id: string): Promise<void> {
  // CompleteGRN(id) — closes the GRN, updates the linked PO status, and
  // defaults a Pending QC to Passed. Server refuses a Failed-QC GRN.
  await CompleteGRN(id)
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchGRNs = (): Promise<GRNRow[]> => pick(realFetch, mockFetch)()
export const updateGRNQCReview = (id: string, status: string, notes: string): Promise<void> =>
  pick(realQCReview, mockQCReview)(id, status, notes)
export const completeGRN = (id: string): Promise<void> => pick(realComplete, mockComplete)(id)
