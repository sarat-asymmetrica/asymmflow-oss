/* Suppliers bridge module — self-contained: types + mock + real + switch.
 * Real binding: `ListSuppliers(limit, offset)` on CRMService (verified against
 * frontend/wailsjs/go/main/CRMService.d.ts:353), called load-all per the
 * census (server clamps 100 default / 1000 max). The real list `SELECT` is
 * deliberately narrow (recon-K2 #2) — tax_id/brands/address/bank fields only
 * exist after `GetSupplierFullProfile`, which K2 does not wire (INTEG gap,
 * see mapSupplier below). SupplierMaster has NO `status` field — status is
 * derived from `is_active` (2-state), the old 3-state Active/Inactive/Pending
 * vocabulary is NOT carried forward (Pending has no backing data). */
import { pick } from './runtime'
import { goDate, num, str } from './map'
import { ListSuppliers } from '$wails/go/main/CRMService'
import { GetSupplierFullProfile } from '$wails/go/main/App'

/** The profile-depth fields ListSuppliers' narrow SELECT omits — filled by a
 * SECOND fetch (GetSupplierFullProfile) when a supplier row is selected. */
export type SupplierProfilePatch = Pick<
  SupplierRow,
  'taxId' | 'address' | 'bankName' | 'accountNumber' | 'iban' | 'swiftCode' | 'rating' | 'totalPurchases' | 'totalPOs' | 'avgPOValue' | 'openIssues'
>

export interface SupplierRow {
  id: string
  code: string
  name: string
  country: string
  leadTimeDays: number
  supplierType: string
  primaryContact: string
  email: string
  phone: string
  isActive: boolean
  status: string // derived: 'Active' | 'Inactive'
  /* ---- profile-only fields (list fetch prunes these; see mapSupplier) ---- */
  taxId: string
  address: string
  bankName: string
  accountNumber: string
  iban: string
  swiftCode: string
  paymentTerms: string
  rating: number
  /* ---- profile KPIs (from GetSupplierFullProfile — K2 does not wire it) ---- */
  totalPurchases: number
  totalPOs: number
  avgPOValue: number
  openIssues: number
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
const CONTACTS = ['Ahmed Khalil', 'Fatima Al-Sayed', 'Yousif Mansoor', 'Layla Haidar', 'Karim Nasser', '']
const COUNTRIES = ['Bahrain', 'Saudi Arabia', 'UAE', 'Kuwait', 'Qatar', 'Oman', 'Germany', 'USA', '']
const SUPPLIER_TYPES = ['Manufacturer', 'Distributor', 'Agent', 'Service Provider']
const TERMS = ['Net 30', 'Net 60', 'Advance', 'LC 90 days', 'Net 15']
const BANKS = ['Ahli United Bank', 'National Bank of Bahrain', 'BBK', 'Standard Chartered', '']

let cache: SupplierRow[] | null = null

function generate(): SupplierRow[] {
  const rand = lcg(20260714 ^ 0x5077)
  const rows: SupplierRow[] = []
  const n = 210
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const name = SUPPLIERS[i % SUPPLIERS.length]!
    const isActive = i % 6 !== 0 // ~83% active, matches old screen's mostly-active data
    const hasBank = i % 5 !== 0 // some suppliers have no bank details on file at all
    const rating = i % 47 === 0 ? 0 : Math.round(rand() * 5 * 10) / 10

    rows.push({
      id: `sup-${i}`,
      code: i % 71 === 0 ? `S-${pad(i, 40)}` : `S-${pad(i, 4)}`, // unbroken monster token
      name,
      country: COUNTRIES[i % COUNTRIES.length]!,
      leadTimeDays: i % 89 === 0 ? 999 : Math.floor(rand() * 90),
      supplierType: SUPPLIER_TYPES[i % SUPPLIER_TYPES.length]!,
      primaryContact: i % 67 === 0 ? '' : CONTACTS[i % CONTACTS.length]!,
      email: i % 67 === 0 ? '' : `procurement${i}@example.bh`,
      phone: i % 67 === 0 ? '' : `+973 ${pad(Math.floor(r * 99999999), 8)}`,
      isActive,
      status: isActive ? 'Active' : 'Inactive',
      taxId: i % 13 === 0 ? '' : `TRN-${pad(100000 + i, 9)}`,
      address: i % 23 === 0 ? '' : `Building ${i}, Road ${100 + (i % 50)}, Manama`,
      bankName: hasBank ? BANKS[i % (BANKS.length - 1)]! : '',
      accountNumber: hasBank ? pad(Math.floor(rand() * 1e10), 12) : '',
      iban: hasBank ? `BH${pad(i, 2)}AUBB${pad(Math.floor(rand() * 1e12), 14)}` : '',
      swiftCode: hasBank ? 'AUBBBHBM' : '',
      paymentTerms: TERMS[i % TERMS.length]!,
      rating,
      totalPurchases:
        i % 89 === 0 ? 987654321098.765 : i % 53 === 0 ? 0.001 : Math.round(rand() * 900_000) / 100,
      totalPOs: i % 29 === 0 ? 0 : Math.floor(rand() * 60),
      avgPOValue: Math.round(rand() * 40_000) / 100,
      openIssues: i % 31 === 0 ? 7 : i % 3 === 0 ? 1 : 0,
    })
  }
  return rows
}

async function mockFetchAll(): Promise<SupplierRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockDeleteSupplier(id: string): Promise<void> {
  cache ??= generate()
  cache = cache.filter((s) => s.id !== id)
  await sleep(150)
}

/* ---- real: fetch WIRED, mutations are INTEG-gapped (honest throw) ----
 * List-fetch column pruning (recon-K2): ListSuppliers' SELECT excludes
 * tax_id/address/bank fields and the full profile KPIs — those require a
 * second fetch (GetSupplierFullProfile) that K2 does not wire. They are
 * zeroed/blanked here rather than faked; see Suppliers.parity.md #2. */
function mapSupplier(r: Record<string, unknown>): SupplierRow {
  const isActive = Boolean(r.is_active)
  return {
    id: str(r.id),
    code: str(r.supplier_code),
    name: str(r.supplier_name),
    country: str(r.country),
    leadTimeDays: num(r.lead_time_days),
    supplierType: str(r.supplier_type),
    primaryContact: str(r.primary_contact),
    email: str(r.email),
    phone: str(r.phone),
    isActive,
    status: isActive ? 'Active' : 'Inactive',
    // profile-only, blank on the list fetch (INTEG gap — GetSupplierFullProfile not wired):
    taxId: '',
    address: '',
    bankName: '',
    accountNumber: '',
    iban: '',
    swiftCode: '',
    paymentTerms: str(r.payment_terms),
    rating: num(r.rating),
    totalPurchases: 0,
    totalPOs: 0,
    avgPOValue: 0,
    openIssues: 0,
  }
}

async function realFetchAll(): Promise<SupplierRow[]> {
  const rows = await ListSuppliers(500, 0)
  return (rows ?? []).map((x) => mapSupplier(x as unknown as Record<string, unknown>))
}

async function realDeleteSupplier(_id: string): Promise<void> {
  throw new Error(
    'INTEG gap: DeleteSupplier — wires at K5. (Server refuses if the supplier has POs, ' +
      'invoices, or contacts on file; the descriptor must surface that 4xx honestly, not assume success.)',
  )
}

/* ---- secondary profile fetch (INTEG): GetSupplierFullProfile ---- */
async function realSupplierProfile(id: string): Promise<SupplierProfilePatch> {
  const p = (await GetSupplierFullProfile(id)) as unknown as Record<string, unknown>
  return {
    taxId: str(p.tax_id),
    address: str(p.address),
    bankName: str(p.bank_name),
    accountNumber: str(p.account_number),
    iban: str(p.iban),
    swiftCode: str(p.swift_code),
    rating: num(p.rating),
    totalPurchases: num(p.total_purchases),
    totalPOs: num(p.total_pos),
    avgPOValue: num(p.avg_po_value),
    openIssues: num(p.open_issues),
  }
}

/** Under mock the list row already carries full profile data; the enrich merge
 * re-supplies the same fields from cache so the path is exercised identically. */
async function mockSupplierProfile(id: string): Promise<SupplierProfilePatch> {
  cache ??= generate()
  await sleep(150)
  const row = cache.find((s) => s.id === id)
  if (!row) return {} as SupplierProfilePatch
  const { taxId, address, bankName, accountNumber, iban, swiftCode, rating, totalPurchases, totalPOs, avgPOValue, openIssues } = row
  return { taxId, address, bankName, accountNumber, iban, swiftCode, rating, totalPurchases, totalPOs, avgPOValue, openIssues }
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchSuppliers = (): Promise<SupplierRow[]> => pick(realFetchAll, mockFetchAll)()
export const deleteSupplier = (id: string): Promise<void> =>
  pick(realDeleteSupplier, mockDeleteSupplier)(id)
export const fetchSupplierProfile = (id: string): Promise<SupplierProfilePatch> =>
  pick(realSupplierProfile, mockSupplierProfile)(id)
