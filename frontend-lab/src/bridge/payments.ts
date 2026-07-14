/* Payments bridge module — self-contained: types + mock + real + switch.
 * PaymentsScreen is a TWO-panel screen (Receipts + Payment History, recon
 * K1-B synthesis #1 — multi-panel ENGINE gap, no descriptor answer yet). Per
 * the build brief's multi-panel rule this module models the PRIMARY ledger
 * only: customer receipts (`ListCustomerReceipts`, App, paged). Payment
 * History (`GetAllPayments`) is a second co-located ledger, ledgered in
 * Payments.parity.md — not built here. */
import { pick } from './runtime'
import { goDate, num, str } from './map'
import { ListCustomerReceipts } from '$wails/go/main/App'

export interface ReceiptRow {
  id: string
  receiptNumber: string
  customerName: string
  division: string
  receiptDate: string
  amountBhd: number
  appliedAmountBhd: number
  unappliedAmountBhd: number
  paymentMethod: string
  reference: string
  status: string
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

const DIVISIONS = ['Acme Instrumentation', 'Beacon Controls']
const STATUSES = ['OnAccount', 'PartiallyApplied', 'Applied', 'Reversed']
const METHODS = ['Cash', 'Cheque', 'Bank Transfer', 'Credit Card', 'LC', 'PDC', 'Other']
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

let cache: ReceiptRow[] | null = null

function generate(): ReceiptRow[] {
  const rand = lcg(20260714)
  const rows: ReceiptRow[] = []
  const n = 280
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 18)
    const year = 2025 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)

    // Adversarial seasoning at deterministic positions: unknown status, a
    // giant receipt, a 0.001-BHD receipt, an empty reference. CustomerReceipt
    // has no currency field (always BHD) — no USD monster here, unlike the
    // multi-currency screens in this cluster.
    const status = i % 97 === 0 ? 'UNKNOWN_STATE' : STATUSES[i % STATUSES.length]!
    const amountBhd = i % 89 === 0 ? 987654321.123 : i % 53 === 0 ? 0.001 : Math.round(rand() * 400_000) / 100

    let appliedAmountBhd: number
    let unappliedAmountBhd: number
    if (status === 'OnAccount') {
      appliedAmountBhd = 0
      unappliedAmountBhd = amountBhd
    } else if (status === 'PartiallyApplied') {
      appliedAmountBhd = Math.round(amountBhd * (0.2 + rand() * 0.6) * 1000) / 1000
      unappliedAmountBhd = Math.round((amountBhd - appliedAmountBhd) * 1000) / 1000
    } else if (status === 'Applied') {
      appliedAmountBhd = amountBhd
      unappliedAmountBhd = 0
    } else {
      // Reversed and UNKNOWN_STATE — no live balance either way.
      appliedAmountBhd = 0
      unappliedAmountBhd = 0
    }

    rows.push({
      id: `rcpt-${i}`,
      receiptNumber: `RCPT-${year}-${pad(i, 4)}`,
      customerName: CUSTOMERS[i % CUSTOMERS.length]!,
      division: DIVISIONS[i % DIVISIONS.length]!,
      receiptDate: `${year}-${pad(month, 2)}-${pad(day, 2)}`,
      amountBhd,
      appliedAmountBhd,
      unappliedAmountBhd,
      paymentMethod: METHODS[Math.floor(r * METHODS.length)]!,
      reference: i % 37 === 0 ? '' : `REF-${pad(i, 5)}`,
      status,
    })
  }
  return rows
}

async function mockFetchPage(limit: number, offset: number): Promise<ReceiptRow[]> {
  cache ??= generate()
  await sleep(offset === 0 ? 250 : 120)
  return cache.slice(offset, offset + limit)
}
async function mockFetchAll(): Promise<ReceiptRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

/** Reverse Receipt — gated to zero-application receipts only (the old
 * screen's guard: `applied_amount_bhd<=0.001 && status!=='Reversed'`,
 * preserved as the action's `visible`). Reversal of an already-applied
 * receipt is out of scope here, same as the old screen. */
async function mockReverseReceipt(id: string, _reason: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) {
    row.status = 'Reversed'
    row.appliedAmountBhd = 0
    row.unappliedAmountBhd = 0
  }
  await sleep(150)
}

/* ---- real: fetch WIRED, mutations are INTEG-gapped (honest throw) ---- */
function mapReceipt(r: Record<string, unknown>): ReceiptRow {
  return {
    id: str(r.id),
    receiptNumber: str(r.receipt_number),
    customerName: str(r.customer_name),
    division: str(r.division),
    receiptDate: goDate(r.receipt_date),
    amountBhd: num(r.amount_bhd),
    appliedAmountBhd: num(r.applied_amount_bhd),
    unappliedAmountBhd: num(r.unapplied_amount_bhd),
    paymentMethod: str(r.payment_method),
    reference: str(r.reference),
    status: str(r.status) || 'OnAccount',
  }
}

async function realFetchPage(limit: number, offset: number): Promise<ReceiptRow[]> {
  const rows = await ListCustomerReceipts(limit, offset)
  return (rows ?? []).map((r) => mapReceipt(r as unknown as Record<string, unknown>))
}
async function realFetchAll(): Promise<ReceiptRow[]> {
  return realFetchPage(200, 0)
}

async function realReverseReceipt(_id: string, _reason: string): Promise<void> {
  throw new Error('INTEG gap: ReverseCustomerReceipt — wires at K5')
}

/* ---- public switched API (descriptors import THESE) ---- */
export const fetchReceiptsPage = (l: number, o: number): Promise<ReceiptRow[]> =>
  pick(realFetchPage, mockFetchPage)(l, o)
export const fetchReceipts = (): Promise<ReceiptRow[]> => pick(realFetchAll, mockFetchAll)()
export const reverseReceipt = (id: string, reason: string): Promise<void> =>
  pick(realReverseReceipt, mockReverseReceipt)(id, reason)
