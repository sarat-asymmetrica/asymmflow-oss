/* Credit Notes bridge module — self-contained: types + mock + real + switch.
 * Old frontend: a sub-ledger squatting inside InvoicesScreen (PARITY_INVOICES
 * #14, recon K1-B). This module gives it its own ~80-line standalone screen,
 * as predicted. `ListCreditNotes(limit, offset)` is paged from day one —
 * the old screen called it as a flat 100-row load with no follow-up
 * (recon K1-B #392/#398), a silent-cap bug fixed here, not preserved. */
import { pick } from './runtime'
import { goDate, num, str } from './map'
import { ApplyCreditNote, ListCreditNotes } from '$wails/go/main/App'

export interface CreditNoteRow {
  id: string
  creditNoteNumber: string
  creditNoteDate: string
  invoiceNumber: string
  customerName: string
  reason: string
  grandTotalBhd: number
  status: string
  division: string
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
const STATUSES = ['Draft', 'Issued', 'Applied']
const REASONS = [
  'Pricing correction',
  'Damaged goods returned',
  'Duplicate invoice',
  'Quantity short-shipped',
  'Agreed commercial discount',
]
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

let cache: CreditNoteRow[] | null = null

function generate(): CreditNoteRow[] {
  const rand = lcg(20260714)
  const rows: CreditNoteRow[] = []
  const n = 140
  for (let i = 1; i <= n; i++) {
    const monthIdx = Math.floor(rand() * 18)
    const year = 2025 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)

    // Adversarial seasoning at deterministic positions: unknown status, a
    // giant credit, a 0.001-BHD credit, an empty invoice reference.
    const status = i % 61 === 0 ? 'UNKNOWN_STATE' : STATUSES[i % STATUSES.length]!
    const grandTotalBhd = i % 71 === 0 ? 456789012.345 : i % 43 === 0 ? 0.001 : Math.round(rand() * 60_000) / 100

    rows.push({
      id: `cn-${i}`,
      creditNoteNumber: `CN-${year}-${pad(i, 4)}`,
      creditNoteDate: `${year}-${pad(month, 2)}-${pad(day, 2)}`,
      invoiceNumber: i % 31 === 0 ? '' : `INV-${year}-${pad(((i * 3) % 500) + 1, 4)}`,
      customerName: CUSTOMERS[i % CUSTOMERS.length]!,
      reason: REASONS[i % REASONS.length]!,
      grandTotalBhd,
      status,
      division: DIVISIONS[i % DIVISIONS.length]!,
    })
  }
  return rows
}

async function mockFetchPage(limit: number, offset: number): Promise<CreditNoteRow[]> {
  cache ??= generate()
  await sleep(offset === 0 ? 250 : 120)
  return cache.slice(offset, offset + limit)
}
async function mockFetchAll(): Promise<CreditNoteRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

/** Apply — reduces AR. The old screen fired this with NO confirm; the
 * descriptor adds `ActionSpec.confirm` as an intentional improvement
 * (recon K1-B #391/#397 flags this as a gap to fix, not preserve). */
async function mockApplyCreditNote(id: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row && row.status !== 'Applied') row.status = 'Applied'
  await sleep(150)
}

/* ---- real: fetch WIRED, mutations are INTEG-gapped (honest throw) ---- */
function mapCreditNote(r: Record<string, unknown>): CreditNoteRow {
  return {
    id: str(r.id),
    creditNoteNumber: str(r.credit_note_number),
    creditNoteDate: goDate(r.credit_note_date),
    invoiceNumber: str(r.invoice_number),
    customerName: str(r.customer_name),
    reason: str(r.reason),
    grandTotalBhd: num(r.grand_total_bhd),
    status: str(r.status) || 'Draft',
    division: str(r.division),
  }
}

async function realFetchPage(limit: number, offset: number): Promise<CreditNoteRow[]> {
  const rows = await ListCreditNotes(limit, offset)
  return (rows ?? []).map((r) => mapCreditNote(r as unknown as Record<string, unknown>))
}
async function realFetchAll(): Promise<CreditNoteRow[]> {
  return realFetchPage(200, 0)
}

async function realApplyCreditNote(id: string): Promise<void> {
  // ApplyCreditNote(creditNoteID) — reduces AR by applying the note to its
  // linked invoice and marks it Applied (see credit_note_service.go). The
  // descriptor gates it behind a confirm (a fix over the old no-confirm apply).
  await ApplyCreditNote(id)
}

/* ---- public switched API (descriptors import THESE) ---- */
export const fetchCreditNotesPage = (l: number, o: number): Promise<CreditNoteRow[]> =>
  pick(realFetchPage, mockFetchPage)(l, o)
export const fetchCreditNotes = (): Promise<CreditNoteRow[]> => pick(realFetchAll, mockFetchAll)()
export const applyCreditNote = (id: string): Promise<void> => pick(realApplyCreditNote, mockApplyCreditNote)(id)
