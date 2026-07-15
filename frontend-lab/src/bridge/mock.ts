/* Lab mock bridge — stands in for the Wails bindings until integration.
 * Adversarial by doctrine: monsters are woven INTO the dataset, not a
 * separate fixture nobody loads. Deterministic (seeded LCG) so Playwright
 * baselines are stable run-to-run. Division names use the synthetic canon
 * (SYNTHETIC_IDENTITY.md); this file is mock/test data, exempt territory. */

export interface InvoiceRow {
  id: string
  number: string
  customer: string
  division: string
  status: string
  issueDate: string
  dueDate: string
  amount: number
  currency: string
}

const DIVISIONS = ['Acme Instrumentation', 'Beacon Controls']
const STATUSES = ['Draft', 'Sent', 'Paid', 'Overdue', 'Cancelled']
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

/** Deterministic LCG — stable adversarial data, stable screenshots. */
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}

function pad(n: number, w: number): string {
  return String(n).padStart(w, '0')
}

let invoices: InvoiceRow[] | null = null

function generate(): InvoiceRow[] {
  const rand = lcg(20260714)
  const rows: InvoiceRow[] = []
  for (let i = 1; i <= 500; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 18) // spread across 18 months
    const year = 2025 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const issue = `${year}-${pad(month, 2)}-${pad(day, 2)}`
    const due = `${year}-${pad(month, 2)}-${pad(Math.min(day + 30, 28), 2)}`

    // Adversarial seasoning at deterministic positions:
    const status =
      i % 97 === 0 ? 'UNKNOWN_STATE' : STATUSES[Math.floor(r * STATUSES.length)]!
    const amount =
      i % 89 === 0 ? 123456789012.345 : i % 53 === 0 ? 0.001 : Math.round(rand() * 8_000_000) / 100
    const currency = i % 41 === 0 ? 'USD' : 'BHD'
    const customer = CUSTOMERS[i % CUSTOMERS.length]!

    rows.push({
      id: `inv-${i}`,
      number:
        i % 71 === 0
          ? `INV-2026-Q3-${pad(i, 40)}` // unbroken 40-digit monster token
          : `INV-${year}-${pad(i, 4)}`,
      customer,
      division: DIVISIONS[i % DIVISIONS.length]!,
      status,
      issueDate: issue,
      dueDate: due,
      amount,
      currency,
    })
  }
  return rows
}

export async function fetchInvoices(): Promise<InvoiceRow[]> {
  invoices ??= generate()
  await new Promise((r) => setTimeout(r, 250)) // visible skeleton, honest async
  return [...invoices]
}

/** Paged variant — mirrors ListCustomerInvoices(limit, offset). */
export async function fetchInvoicesPage(limit: number, offset: number): Promise<InvoiceRow[]> {
  invoices ??= generate()
  await new Promise((r) => setTimeout(r, offset === 0 ? 250 : 120))
  return invoices.slice(offset, offset + limit)
}

export interface NewInvoiceDraft {
  customer: string
  division: string
  issueDate: string
  dueDate: string
  amount: number | null
  currency: string
  notes: string
}

let createdCount = 0

export async function createInvoice(draft: NewInvoiceDraft): Promise<void> {
  invoices ??= generate()
  createdCount++
  invoices.unshift({
    id: `inv-new-${createdCount}`,
    number: `INV-2026-N${pad(createdCount, 3)}`,
    customer: draft.customer,
    division: draft.division,
    status: 'Draft',
    issueDate: draft.issueDate,
    dueDate: draft.dueDate,
    amount: draft.amount ?? 0,
    currency: draft.currency,
  })
  await new Promise((r) => setTimeout(r, 150))
}

export async function deleteInvoice(id: string): Promise<void> {
  invoices ??= generate()
  invoices = invoices.filter((i) => i.id !== id)
  await new Promise((r) => setTimeout(r, 120))
}

/** Select options — at INTEG these come from bindings / the divisions store. */
export async function customerOptions(): Promise<{ value: string; label: string }[]> {
  await new Promise((r) => setTimeout(r, 100))
  return CUSTOMERS.filter((c) => c).map((c) => ({ value: c, label: c }))
}

export async function markInvoicePaid(id: string): Promise<void> {
  invoices ??= generate()
  const inv = invoices.find((i) => i.id === id)
  if (inv) inv.status = 'Paid'
  await new Promise((r) => setTimeout(r, 120))
}

/* ---- Customers (EntityMaster pilot) ---- */

export interface CustomerRow {
  id: string
  code: string
  name: string
  city: string
  status: string
  phone: string
  email: string
  paymentTerms: string
  creditLimit: number
  balance: number
  openOrders: number
  lastOrderDate: string
  /* ---- CustomerFullProfile fields (K2 widen — see Customers.parity.md) ----
   * Profile-only: only populated after GetCustomerFullProfile, which the
   * list fetch (ListCustomers) does not return. Real mapping blanks/zeroes
   * these (INTEG gap); mock generates full adversarial values. */
  trn: string
  industry: string
  relationYears: number
  paymentTermsDays: number
  isCreditBlocked: boolean
  arCurrent: number
  ar30: number
  ar60: number
  ar90: number
  rfqsFloated: number
  rfqsWon: number
  winRate: number
}

const CITIES = ['Manama', 'Sitra', 'Riffa', 'Muharraq', 'Hamad Town', '']
const TERMS = ['Net 30', 'Net 60', 'Advance', 'LC 90 days']
const CUSTOMER_STATUSES = ['Active', 'Dormant', 'On Hold', 'Blacklisted']
const INDUSTRIES = [
  'Oil & Gas',
  'Construction',
  'Manufacturing',
  'Healthcare',
  'Retail',
  'Logistics',
  'Government',
  '',
]

let customers: CustomerRow[] | null = null

function generateCustomers(): CustomerRow[] {
  const rand = lcg(19770707)
  const rows: CustomerRow[] = []
  for (let i = 1; i <= 120; i++) {
    const name = CUSTOMERS[i % CUSTOMERS.length]!
    const monthIdx = Math.floor(rand() * 18)
    const year = 2025 + Math.floor(monthIdx / 12)
    const balance = i % 53 === 0 ? -12345.678 : Math.round(rand() * 3_000_000) / 100
    // AR aging buckets split the balance across current/30/60/90 — a monster
    // row (i%37===0) parks the entire balance in the 90+ bucket to exercise
    // the danger-tone threshold on a fully-overdue receivable.
    const allOverdue = i % 37 === 0
    const arCurrent = allOverdue ? 0 : Math.round(balance * (0.3 + rand() * 0.3) * 100) / 100
    const ar30 = allOverdue ? 0 : Math.round(balance * rand() * 0.2 * 100) / 100
    const ar60 = allOverdue ? 0 : Math.round(balance * rand() * 0.15 * 100) / 100
    const ar90 = allOverdue ? balance : Math.round((balance - arCurrent - ar30 - ar60) * 100) / 100
    const rfqsFloated = i % 59 === 0 ? 0 : Math.floor(rand() * 40)
    const rfqsWon = rfqsFloated === 0 ? 0 : Math.floor(rand() * rfqsFloated)

    rows.push({
      id: `cus-${i}`,
      code: `C-${pad(i, 4)}`,
      name,
      city: CITIES[i % CITIES.length]!,
      status: i % 83 === 0 ? 'MIGRATED_LEGACY' : CUSTOMER_STATUSES[Math.floor(rand() * CUSTOMER_STATUSES.length)]!,
      phone: i % 13 === 0 ? '' : `+973 ${pad(Math.floor(rand() * 99999999), 8)}`,
      email:
        i % 17 === 0
          ? 'accounts.receivable.department.regional.office@extremely-long-corporate-domain-name.example.com.bh'
          : `finance${i}@example.bh`,
      paymentTerms: TERMS[i % TERMS.length]!,
      creditLimit: i % 89 === 0 ? 999999999.999 : Math.round(rand() * 5_000_000) / 100,
      balance,
      openOrders: i % 29 === 0 ? 0 : Math.floor(rand() * 14),
      lastOrderDate: i % 29 === 0 ? '' : `${year}-${pad((monthIdx % 12) + 1, 2)}-${pad(1 + Math.floor(rand() * 27), 2)}`,
      trn: i % 11 === 0 ? '' : `TRN-${pad(200000 + i, 9)}`,
      industry: INDUSTRIES[i % INDUSTRIES.length]!,
      relationYears: i % 97 === 0 ? 0 : Math.floor(rand() * 20),
      paymentTermsDays: i % 89 === 0 ? 999 : [30, 60, 0, 90][i % 4]!,
      isCreditBlocked: i % 19 === 0,
      arCurrent,
      ar30,
      ar60,
      ar90,
      rfqsFloated,
      rfqsWon,
      winRate: rfqsFloated === 0 ? 0 : Math.round((rfqsWon / rfqsFloated) * 1000) / 10,
    })
  }
  return rows
}

export async function fetchCustomers(): Promise<CustomerRow[]> {
  customers ??= generateCustomers()
  await new Promise((r) => setTimeout(r, 250))
  return [...customers]
}

export async function setCustomerStatus(id: string, status: string): Promise<void> {
  customers ??= generateCustomers()
  const c = customers.find((x) => x.id === id)
  if (c) c.status = status
  await new Promise((r) => setTimeout(r, 120))
}
