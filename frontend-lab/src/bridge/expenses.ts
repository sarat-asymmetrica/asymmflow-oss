/* Expenses bridge module — self-contained: types + mock + real + switch.
 * K1 scope: the PRIMARY panel = Entries (rendered as a TABLE — the old
 * screen's card/list layout reads as legacy, the data is tabular). Recurring
 * schedules, the Approvals queue, the bank-candidate Workspace, and
 * category/vendor master-data CRUD are all separate panels on the old
 * screen — ledgered as a multi-panel ENGINE gap, see
 * screens/parity/Expenses.parity.md. */
import { pick } from './runtime'
import { goDate, num, str } from './map'
import { ListExpenseEntries } from '$wails/go/main/FinanceService'

export interface ExpenseEntryRow {
  id: string
  entryNumber: string
  division: string
  expenseDate: string
  dueDate: string
  description: string
  categoryId: string
  categoryName: string
  vendorId: string
  vendorName: string
  sourceType: string
  costCenter: string
  currency: string
  amount: number
  vatAmount: number
  totalAmount: number
  /** lowercase vocabulary: draft/submitted/approved/rejected/posted — NOT
   * TitleCase like every other screen in this cluster (census-flagged). */
  status: string
  /** lowercase: unpaid/paid. */
  paymentStatus: string
  paymentMethod: string
  paymentReference: string
  bankAccountId: string
  notes: string
  approvedAt: string
  postedAt: string
  paidAt: string
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
const todayIso = (): string => new Date().toISOString().slice(0, 10)

const DIVISIONS = ['Acme Instrumentation', 'Beacon Controls']

const CATEGORIES = [
  'Office Supplies',
  'Travel & Transport',
  'Utilities',
  'Professional Fees',
  'Bank Charges',
  'IT & Software',
  'Repairs & Maintenance',
  'Marketing & Business Development',
]

const VENDORS = [
  'Gulf Stationery Trading',
  'Bahrain Telecommunications Co.',
  'Al Waha Business Services',
  'Manama Fleet Maintenance W.L.L.',
  'International Establishment for Facilities Management, Cleaning Contracts and General Maintenance Services (formerly Gulf Property Care Company) W.L.L.',
  'مؤسسة الخليج لخدمات الصيانة العامة',
  'X',
  '',
]

const PAYMENT_METHODS = ['Bank Transfer', 'Cheque', 'Cash']

let cache: ExpenseEntryRow[] | null = null

function generate(): ExpenseEntryRow[] {
  const rand = lcg(20260714)
  const rows: ExpenseEntryRow[] = []
  const n = 280
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 20)
    const year = 2024 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const expenseDate = `${year}-${pad(month, 2)}-${pad(day, 2)}`

    // Status vocabulary is lowercase — do NOT TitleCase it (census gotcha).
    const status =
      i % 97 === 0
        ? 'unknown_state'
        : i % 11 === 0
          ? 'posted'
          : i % 7 === 0
            ? 'approved'
            : i % 5 === 0
              ? 'rejected'
              : i % 2 === 0
                ? 'submitted'
                : 'draft'
    const paid = status === 'posted' && i % 3 !== 0
    const paymentStatus = paid ? 'paid' : 'unpaid'

    const currency = i % 41 === 0 ? 'USD' : 'BHD'
    const amount = i % 89 === 0 ? 456789012.345 : i % 53 === 0 ? 0.001 : Math.round(rand() * 200_000) / 100
    const vatAmount = Math.round(amount * 0.1 * 1000) / 1000
    const totalAmount = Math.round((amount + vatAmount) * 1000) / 1000

    const category = CATEGORIES[i % CATEGORIES.length]!
    const vendor = VENDORS[i % VENDORS.length]!

    const description =
      i % 83 === 0
        ? 'Reimbursement for consolidated Q3 field-service travel, accommodation, per-diem, courier and incidental expenses across three concurrent site visits (Sitra, Hidd, Askar) — see attached scanned receipts bundle for the full cost-line breakdown.'
        : i % 59 === 0
          ? 'إيصال صيانة دورية للمركبة'
          : `${category} — ${vendor || 'misc vendor'}`

    rows.push({
      id: `exp-${i}`,
      entryNumber: i % 71 === 0 ? `EXP-${year}-${pad(i, 40)}` : `EXP-${year}-${pad(i, 4)}`,
      division: DIVISIONS[i % DIVISIONS.length]!,
      expenseDate,
      dueDate: status === 'draft' ? '' : expenseDate,
      description,
      categoryId: `cat-${(i % CATEGORIES.length) + 1}`,
      categoryName: category,
      vendorId: vendor ? `ven-${(i % VENDORS.length) + 1}` : '',
      vendorName: vendor,
      sourceType: i % 13 === 0 ? 'bank-derived' : 'manual',
      costCenter: i % 19 === 0 ? '' : `CC-${(i % 6) + 1}`,
      currency,
      amount,
      vatAmount,
      totalAmount,
      status,
      paymentStatus,
      paymentMethod: paid ? PAYMENT_METHODS[i % PAYMENT_METHODS.length]! : '',
      paymentReference: paid ? `PMT-${pad(i, 5)}` : '',
      bankAccountId: paid ? 'bank-1' : '',
      notes: i % 29 === 0 ? '' : 'Auto-generated synthetic entry.',
      approvedAt: status === 'approved' || status === 'posted' ? expenseDate : '',
      postedAt: status === 'posted' ? expenseDate : '',
      paidAt: paid ? expenseDate : '',
    })
  }
  return rows
}

async function mockFetchAll(): Promise<ExpenseEntryRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockSubmit(id: string): Promise<void> {
  cache ??= generate()
  const e = cache.find((x) => x.id === id)
  if (e) e.status = 'submitted'
  await sleep(150)
}

async function mockApprove(id: string): Promise<void> {
  cache ??= generate()
  const e = cache.find((x) => x.id === id)
  if (e) {
    e.status = 'approved'
    e.approvedAt = todayIso()
  }
  await sleep(150)
}

async function mockReject(id: string, _reason: string): Promise<void> {
  cache ??= generate()
  const e = cache.find((x) => x.id === id)
  if (e) e.status = 'rejected'
  await sleep(150)
}

async function mockPost(id: string): Promise<void> {
  cache ??= generate()
  const e = cache.find((x) => x.id === id)
  if (e) {
    e.status = 'posted'
    e.postedAt = todayIso()
  }
  await sleep(150)
}

async function mockDelete(id: string): Promise<void> {
  cache ??= generate()
  cache = cache.filter((x) => x.id !== id)
  await sleep(120)
}

/* ---- real: fetch WIRED, mutations are INTEG-gapped (honest throw) ---- */
function mapExpenseEntry(r: Record<string, unknown>): ExpenseEntryRow {
  return {
    id: str(r.id),
    entryNumber: str(r.entry_number),
    division: str(r.division),
    expenseDate: goDate(r.expense_date),
    dueDate: goDate(r.due_date),
    description: str(r.description),
    categoryId: str(r.category_id),
    categoryName: str(r.category_name),
    vendorId: str(r.vendor_id),
    vendorName: str(r.vendor_name),
    sourceType: str(r.source_type) || 'manual',
    costCenter: str(r.cost_center),
    currency: str(r.currency) || 'BHD',
    amount: num(r.amount),
    vatAmount: num(r.vat_amount),
    totalAmount: num(r.total_amount),
    status: str(r.status) || 'draft',
    paymentStatus: str(r.payment_status) || 'unpaid',
    paymentMethod: str(r.payment_method),
    paymentReference: str(r.payment_reference),
    bankAccountId: str(r.bank_account_id),
    notes: str(r.notes),
    approvedAt: goDate(r.approved_at),
    postedAt: goDate(r.posted_at),
    paidAt: goDate(r.paid_at),
  }
}

async function realFetchAll(): Promise<ExpenseEntryRow[]> {
  // status='' (all statuses), includePaid=true — mirrors the old screen's
  // listExpenseEntries('', true) mount call.
  const rows = await ListExpenseEntries('', true)
  return (rows ?? []).map((x) => mapExpenseEntry(x as unknown as Record<string, unknown>))
}

async function realSubmit(_id: string): Promise<void> {
  throw new Error('INTEG gap: SubmitExpenseEntry — wires at K5')
}

async function realApprove(_id: string): Promise<void> {
  throw new Error('INTEG gap: ApproveExpenseEntry — wires at K5')
}

async function realReject(_id: string, _reason: string): Promise<void> {
  throw new Error('INTEG gap: RejectExpenseEntry — wires at K5')
}

async function realPost(_id: string): Promise<void> {
  throw new Error('INTEG gap: PostExpenseEntry — wires at K5')
}

async function realDelete(_id: string): Promise<void> {
  throw new Error('INTEG gap: DeleteExpenseEntry — wires at K5')
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchExpenseEntries = (): Promise<ExpenseEntryRow[]> => pick(realFetchAll, mockFetchAll)()
export const submitExpenseEntry = (id: string): Promise<void> => pick(realSubmit, mockSubmit)(id)
export const approveExpenseEntry = (id: string): Promise<void> => pick(realApprove, mockApprove)(id)
export const rejectExpenseEntry = (id: string, reason: string): Promise<void> =>
  pick(realReject, mockReject)(id, reason)
export const postExpenseEntry = (id: string): Promise<void> => pick(realPost, mockPost)(id)
export const deleteExpenseEntry = (id: string): Promise<void> => pick(realDelete, mockDelete)(id)
