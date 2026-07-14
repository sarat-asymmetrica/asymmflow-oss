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

export async function markInvoicePaid(id: string): Promise<void> {
  invoices ??= generate()
  const inv = invoices.find((i) => i.id === id)
  if (inv) inv.status = 'Paid'
  await new Promise((r) => setTimeout(r, 120))
}
