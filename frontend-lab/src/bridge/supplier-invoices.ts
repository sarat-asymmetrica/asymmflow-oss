/* SupplierInvoices bridge module — self-contained: types + mock + real +
 * switch. K1 scope note: every mutating capability on this screen (New/Edit,
 * 3-Way Match, Approve, Mark Paid) is a financial hot-zone or a form-archetype
 * SLOT — see screens/parity/SupplierInvoices.parity.md. Fetch + three action
 * adapters are wired to real bindings (Approve = SoD via the session actor;
 * Mark Paid; 3-Way Match). New/Create stays an honest gap: the CreateSupplier-
 * Invoice binding takes a full finance.SupplierInvoice struct with no clean
 * frontend draft, so it is intentionally not adapted here. */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import {
  ApproveSupplierInvoice,
  GetSupplierInvoices,
  MarkSupplierInvoicePaid,
  PerformThreeWayMatch,
} from '$wails/go/main/App'
import { actingUserId } from '../stores/session.svelte'

export interface SupplierInvoiceRow {
  id: string
  supplierId: string
  supplierName: string
  invoiceNumber: string
  purchaseOrderId: string
  grnId: string
  currency: string
  subtotalForeign: number
  vatForeign: number
  totalForeign: number
  totalBhd: number
  /** 3-way-match dimension: Pending | Matched | Discrepancy | Review Required | Dispute. */
  matchStatus: string
  poMatchOk: boolean
  grnMatchOk: boolean
  /** Lifecycle dimension: Pending | Approved | Rejected | Paid | Verified | Disputed | Dispute. */
  status: string
  /** Settlement dimension: Unpaid | Scheduled | Paid ('Overdue' is client-derived, not stored). */
  paymentStatus: string
  dueDate: string
  invoiceDate: string
  discrepancyReason: string
}

/** PerformThreeWayMatch result — mirrors main.ThreeWayMatchResult exactly. */
export interface ThreeWayMatchResult {
  matched: boolean
  reason: string
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

const MATCH_STATUSES = ['Pending', 'Matched', 'Discrepancy', 'Review Required', 'Dispute']
const STATUSES = ['Pending', 'Approved', 'Rejected', 'Paid', 'Verified', 'Disputed', 'Dispute']
const PAYMENT_STATUSES = ['Unpaid', 'Scheduled', 'Paid']
const CURRENCIES = ['BHD', 'USD', 'EUR', 'GBP', 'AED', 'SAR']

let cache: SupplierInvoiceRow[] | null = null

function generate(): SupplierInvoiceRow[] {
  const rand = lcg(20260714)
  const rows: SupplierInvoiceRow[] = []
  const n = 240
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 22)
    const invYear = 2024 + Math.floor(monthIdx / 12)
    const invMonth = (monthIdx % 12) + 1
    const invDay = 1 + Math.floor(rand() * 27)
    const invoiceDate = `${invYear}-${pad(invMonth, 2)}-${pad(invDay, 2)}`
    // Due date spread wide so the due-date tone (overdue/soon/future) and the
    // client-derived Overdue payment-status all get real coverage.
    const dueOffsetDays = -60 + Math.floor(rand() * 150) // -60 .. +89 days from invoice date
    const dueDateObj = new Date(`${invoiceDate}T00:00:00`)
    dueDateObj.setDate(dueDateObj.getDate() + dueOffsetDays)
    const dueDate = i % 61 === 0 ? '' : dueDateObj.toISOString().slice(0, 10)

    const matchStatus = i % 97 === 0 ? 'UNKNOWN_STATE' : MATCH_STATUSES[i % MATCH_STATUSES.length]!
    const status = STATUSES[Math.floor(r * STATUSES.length)]!
    const paymentStatus = matchStatus === 'Matched' && status === 'Paid' ? 'Paid' : PAYMENT_STATUSES[i % PAYMENT_STATUSES.length]!

    const currency = i % 41 === 0 ? 'USD' : i % 13 === 0 ? CURRENCIES[Math.floor(r * CURRENCIES.length)]! : 'BHD'
    const exchangeRate = currency === 'BHD' ? 1 : Math.round((0.2 + rand() * 4) * 10000) / 10000
    const subtotalForeign =
      i % 89 === 0 ? 876543210987.654 : i % 53 === 0 ? 0.001 : Math.round(rand() * 250_000) / 100
    const vatForeign = Math.round(subtotalForeign * 0.1 * 1000) / 1000
    const totalForeign = Math.round((subtotalForeign + vatForeign) * 100) / 100
    const totalBhd = Math.round((subtotalForeign + vatForeign) * exchangeRate * 1000) / 1000

    rows.push({
      id: `sinv-${i}`,
      supplierId: `sup-${(i % SUPPLIERS.length) + 1}`,
      supplierName: i % 67 === 0 ? '' : SUPPLIERS[i % SUPPLIERS.length]!,
      invoiceNumber: i % 71 === 0 ? `SINV-${invYear}-${pad(i, 40)}` : `SINV-${invYear}-${pad(i, 4)}`,
      purchaseOrderId: i % 31 === 0 ? '' : `po-${((i * 7) % 260) + 1}`,
      grnId: i % 19 === 0 ? '' : `grn-${((i * 11) % 220) + 1}`,
      currency,
      subtotalForeign,
      vatForeign,
      totalForeign,
      totalBhd,
      matchStatus,
      poMatchOk: matchStatus === 'Matched',
      grnMatchOk: matchStatus === 'Matched' && i % 5 !== 0,
      status,
      paymentStatus,
      dueDate,
      invoiceDate,
      discrepancyReason:
        matchStatus === 'Discrepancy'
          ? i % 3 === 0
            ? 'Quantity billed exceeds GRN accepted quantity on line 2.'
            : 'Unit price on invoice does not match PO rate — pending supplier credit note.'
          : '',
    })
  }
  return rows
}

async function mockFetch(): Promise<SupplierInvoiceRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockApprove(id: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) row.status = 'Approved'
  await sleep(140)
}

async function mockMarkPaid(id: string, _paymentRef: string, _paymentMethod: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) {
    row.status = 'Paid'
    row.paymentStatus = 'Paid'
  }
  await sleep(140)
}

async function mockThreeWayMatch(id: string): Promise<ThreeWayMatchResult> {
  cache ??= generate()
  await sleep(160)
  const row = cache.find((r) => r.id === id)
  if (!row) return { matched: false, reason: `Supplier invoice ${id} not found.` }
  const matched = row.matchStatus === 'Matched'
  return {
    matched,
    reason: matched
      ? 'PO, GRN and invoice amounts agree within tolerance.'
      : row.discrepancyReason || `Match status is ${row.matchStatus}.`,
  }
}

/* ---- real: fetch WIRED (no mutations to gap — this module is read-only) ---- */
function mapSupplierInvoice(r: Record<string, unknown>): SupplierInvoiceRow {
  return {
    id: str(r.id),
    supplierId: str(r.supplier_id),
    supplierName: str(r.supplier_name),
    invoiceNumber: str(r.invoice_number),
    purchaseOrderId: str(r.purchase_order_id),
    grnId: str(r.grn_id),
    currency: str(r.currency) || 'BHD',
    subtotalForeign: num(r.subtotal_foreign),
    vatForeign: num(r.vat_foreign),
    totalForeign: num(r.total_foreign),
    totalBhd: num(r.total_bhd),
    matchStatus: str(r.match_status) || 'Pending',
    poMatchOk: !!r.po_match_ok,
    grnMatchOk: !!r.grn_match_ok,
    status: str(r.status) || 'Pending',
    paymentStatus: str(r.payment_status) || 'Unpaid',
    dueDate: goDate(r.due_date),
    invoiceDate: goDate(r.invoice_date),
    discrepancyReason: str(r.discrepancy_reason),
  }
}

async function realFetch(): Promise<SupplierInvoiceRow[]> {
  const rows = await GetSupplierInvoices()
  return (rows ?? []).map((x) => mapSupplierInvoice(x as unknown as Record<string, unknown>))
}

async function realApprove(id: string): Promise<void> {
  // Segregation-of-duties: the approver is the acting session user, not a
  // free-text field. The backend re-derives from its own identity when handed
  // a shared/admin id and rejects an empty approver.
  await ApproveSupplierInvoice(id, actingUserId())
}

async function realMarkPaid(id: string, paymentRef: string, paymentMethod: string): Promise<void> {
  // MarkSupplierInvoicePaid(id, paymentRef, paymentMethod). Backend enforces the
  // invoice is Approved (or already Paid, for idempotency) before settling.
  await MarkSupplierInvoicePaid(id, paymentRef, paymentMethod)
}

async function realThreeWayMatch(id: string): Promise<ThreeWayMatchResult> {
  const res = await PerformThreeWayMatch(id)
  const r = res as unknown as Record<string, unknown>
  return { matched: !!r.matched, reason: str(r.reason) }
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchSupplierInvoices = (): Promise<SupplierInvoiceRow[]> => pick(realFetch, mockFetch)()
export const approveSupplierInvoice = (id: string): Promise<void> => pick(realApprove, mockApprove)(id)
export const markSupplierInvoicePaid = (
  id: string,
  paymentRef: string,
  paymentMethod: string,
): Promise<void> => pick(realMarkPaid, mockMarkPaid)(id, paymentRef, paymentMethod)
export const performThreeWayMatch = (id: string): Promise<ThreeWayMatchResult> =>
  pick(realThreeWayMatch, mockThreeWayMatch)(id)
