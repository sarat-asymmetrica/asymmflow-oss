/* SupplierPayments bridge module — self-contained: types + mock + real +
 * switch. K1 scope note: Record Payment (FX-aware) and Edit (locked linkage)
 * are financial-hot-zone SLOTs, ledgered rather than rebuilt loosely — see
 * screens/parity/SupplierPayments.parity.md. Delete is the one action K1
 * builds (mock full mutation, real = honest INTEG-gap).
 *
 * TWO-SOURCE MERGE (census: real SupplierPayment rows + synthetic
 * Expense-settlement rows from a different backend surface, tagged by
 * `source`). Per the brief, the merge is done here in the bridge: mock
 * generates + tags + sorts BOTH kinds; real maps only the real
 * GetAllSupplierPayments() rows and notes the Expense side as an INTEG gap
 * (a second, unrelated fetch this module doesn't attempt to compose). */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import { DeleteSupplierPayment, GetAllSupplierPayments } from '$wails/go/main/App'

export interface SupplierPaymentRow {
  id: string
  /** Pseudo-status discriminator: 'Supplier Invoice' | 'Expense'. */
  source: string
  paymentDate: string
  /** supplier_name (supplier-invoice rows) or vendor/category name (expense rows). */
  supplierName: string
  /** invoice_number (supplier-invoice rows) or entry_number (expense rows). */
  invoiceNumber: string
  amountBhd: number
  currency: string
  paymentMethod: string
  reference: string
  supplierInvoiceId: string
  amountForeign: number
  exchangeRate: number
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

const EXPENSE_VENDORS = [
  'Office Supplies Co.',
  'Bahrain Electricity & Water Authority',
  'Gulf Air Cargo & Travel',
  'Zain Bahrain Telecom',
  'Facilities & Maintenance Services W.L.L.',
  '',
]

const METHODS = ['Bank Transfer', 'Cheque', 'LC', 'Cash', 'Wire Transfer', 'PDC', 'Other']
const CURRENCIES = ['BHD', 'USD', 'EUR', 'GBP', 'AED', 'SAR']

let cache: SupplierPaymentRow[] | null = null

function generate(): SupplierPaymentRow[] {
  const rand = lcg(20260714)
  const rows: SupplierPaymentRow[] = []

  const nSupplier = 210
  for (let i = 1; i <= nSupplier; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 22)
    const year = 2024 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const paymentDate = `${year}-${pad(month, 2)}-${pad(day, 2)}`

    // A handful of rows carry an unrecognized source to exercise the badge's
    // neutral-fallback (mirrors the UNKNOWN_STATE monster used elsewhere).
    const source = i % 101 === 0 ? 'UNKNOWN_SOURCE' : 'Supplier Invoice'
    const currency = i % 41 === 0 ? 'USD' : i % 13 === 0 ? CURRENCIES[Math.floor(r * CURRENCIES.length)]! : 'BHD'
    const exchangeRate = currency === 'BHD' ? 1 : Math.round((0.2 + rand() * 4) * 10000) / 10000
    const amountForeign =
      i % 89 === 0 ? 765432109876.543 : i % 53 === 0 ? 0.001 : Math.round(rand() * 200_000) / 100
    const amountBhd = Math.round(amountForeign * exchangeRate * 1000) / 1000

    rows.push({
      id: `sp-${i}`,
      source,
      paymentDate,
      supplierName: i % 67 === 0 ? '' : SUPPLIERS[i % SUPPLIERS.length]!,
      invoiceNumber: i % 71 === 0 ? `SINV-${year}-${pad(i, 40)}` : `SINV-${year}-${pad(i, 4)}`,
      amountBhd,
      currency,
      paymentMethod: METHODS[i % METHODS.length]!,
      reference: i % 23 === 0 ? '' : `REF-${pad(i, 6)}`,
      supplierInvoiceId: `sinv-${((i * 7) % 240) + 1}`,
      amountForeign,
      exchangeRate,
    })
  }

  const nExpense = 85
  for (let j = 1; j <= nExpense; j++) {
    const monthIdx = Math.floor(rand() * 22)
    const year = 2024 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const paymentDate = `${year}-${pad(month, 2)}-${pad(day, 2)}`
    const amountBhd =
      j % 31 === 0 ? 543210987.654 : j % 19 === 0 ? 0.001 : Math.round(rand() * 15_000) / 100

    rows.push({
      id: `expense-${j}`,
      source: 'Expense',
      paymentDate,
      supplierName:
        j % 29 === 0
          ? 'Facilities, Maintenance, Utilities & General Corporate Services Vendor Consolidated Payee Record for the Extended Reporting Period (formerly Regional Office Overheads Account) — Manama Branch'
          : EXPENSE_VENDORS[j % EXPENSE_VENDORS.length]!,
      invoiceNumber: `EXP-${year}-${pad(j, 4)}`,
      amountBhd,
      currency: 'BHD', // expense settlements are domestic — no FX leg
      paymentMethod: METHODS[(j * 3) % METHODS.length]!,
      reference: j % 17 === 0 ? '' : `EXPREF-${pad(j, 5)}`,
      supplierInvoiceId: '',
      amountForeign: amountBhd,
      exchangeRate: 1,
    })
  }

  // Merge + sort newest-first, mirroring the old screen's combined list.
  rows.sort((a, b) => (a.paymentDate < b.paymentDate ? 1 : a.paymentDate > b.paymentDate ? -1 : 0))
  return rows
}

async function mockFetch(): Promise<SupplierPaymentRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockDelete(id: string): Promise<void> {
  cache ??= generate()
  cache = cache.filter((p) => p.id !== id)
  await sleep(120)
}

/* ---- real: fetch WIRED (supplier-invoice source only), mutation
 * INTEG-gapped (honest throw) ---- */
function mapSupplierPayment(r: Record<string, unknown>): SupplierPaymentRow {
  return {
    id: str(r.id),
    source: 'Supplier Invoice',
    paymentDate: goDate(r.payment_date),
    supplierName: str(r.supplier_name),
    invoiceNumber: str(r.invoice_number),
    amountBhd: num(r.amount_bhd),
    currency: str(r.currency) || 'BHD',
    paymentMethod: str(r.payment_method),
    reference: str(r.reference),
    supplierInvoiceId: str(r.supplier_invoice_id),
    amountForeign: num(r.amount_foreign),
    exchangeRate: num(r.exchange_rate) || 1,
  }
}

async function realFetch(): Promise<SupplierPaymentRow[]> {
  // Only the real SupplierPayment source is mapped here. The old screen's
  // second source (listExpenseEntries-derived settlement rows) is a
  // different backend surface entirely — composing it is an INTEG gap, not
  // a bridge-layer mapping problem (see the parity doc).
  const rows = await GetAllSupplierPayments()
  return (rows ?? []).map((x) => mapSupplierPayment(x as unknown as Record<string, unknown>))
}

async function realDelete(id: string): Promise<void> {
  // Backend DeleteSupplierPayment(id) deletes a supplier_payment by id behind
  // its own permission + delete-request guard. WHICH rows expose a delete
  // action — source==='Supplier Invoice' only, never expense settlements —
  // is a UI concern (the old screen's isExpenseSettlement guard); this seam
  // just passes the id through.
  await DeleteSupplierPayment(id)
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchSupplierPayments = (): Promise<SupplierPaymentRow[]> => pick(realFetch, mockFetch)()
export const deleteSupplierPayment = (id: string): Promise<void> => pick(realDelete, mockDelete)(id)
