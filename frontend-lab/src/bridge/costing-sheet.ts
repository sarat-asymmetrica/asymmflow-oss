/* Costing Sheet bridge module — self-contained: types + mock + real + switch.
 * Old screen: CostingSheetScreen.svelte (3026 lines) — a two-mode workspace:
 * pick an opportunity (merged RFQs + pipeline Opportunities, same merge
 * opportunities.ts documents) or start blank, then build a cost/quote sheet
 * whose pricing waterfall is the SACRED math (ported verbatim into
 * screens/costing-sheet-vm.svelte.ts's calcLine — this file carries no
 * arithmetic).
 *
 * Per the K5 build brief, unlike opportunities.ts (which INTEG-gaps its whole
 * merge), THIS screen's opportunity merge + line-item/revision/settings reads
 * are real FETCH bindings — only the write side (create/update/clone/activate/
 * save-as-offer/export/open-file) is INTEG-gapped. Division vocabulary mirrors
 * bridge/mock.ts's divisionOptions()/DIVISIONS precedent (L7): a static mock
 * list now, wired to the real divisions store at K5 — never a bare literal
 * scattered through the screen. Synthetic-only data (SYNTHETIC_IDENTITY.md). */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import { getDivisionOptions, getDefaultDivisionKey } from '../stores/divisions.svelte'
import { GetRFQs, GetPipelineOpportunities, GetOpportunityLineItems, ListCustomers, GetPreparedByOptions, SaveCostingAsOffer } from '$wails/go/main/App'
import { GetCostingSheets, GetCostingsByRFQ, CreateCostingSheet, CloneCostingAsNewRevision, SetActiveCostingRevision } from '$wails/go/main/CRMService'
import { GetSettings } from '$wails/go/main/DocumentsService'
import type { main } from '$wails/go/models'

/* ---- FX table — hardcoded per the sacred spec, no live binding. ---- */
export const CURRENCY_RATES: Record<string, number> = {
  BHD: 1.0,
  EUR: 0.45,
  USD: 0.376,
  GBP: 0.52,
  CHF: 0.43,
}
export const CURRENCY_OPTIONS: string[] = Object.keys(CURRENCY_RATES)
export const FALLBACK_EXCHANGE_RATE = 0.45

/* ---- Division vocabulary (L7): sourced from the divisions store — the real
 * GetDivisionRegistry under Wails, the BUILTIN synthetic fallback under mock
 * (I1). ONE source; never a bare literal in the screen/VM. The costing VM reads
 * these at construction (post-boot, so the registry is loaded). */
export function costingDivisionOptions(): { value: string; label: string }[] {
  return getDivisionOptions()
}
export function defaultCostingDivision(): string {
  return getDefaultDivisionKey()
}

// Synthetic division literals for MOCK opportunity seeding only (like every
// other bridge mock generator) — NOT the form vocabulary, which is the store's.
const MOCK_DIVISIONS = ['Acme Instrumentation', 'Beacon Controls']

/* ---- Types ---- */

export interface CostingOpportunityRow {
  id: string
  source: 'rfq' | 'pipeline'
  /** User-facing reference shown in the picker (RFQ ref / eh_ref / folder no). */
  ref: string
  /** Dedup key against RFQ folder numbers — a pipeline row whose folder_number
   * matches a live RFQ's rfq_number is the SAME funnel item, not a duplicate. */
  folderRef: string
  customer: string
  project: string
  /** 0 = "value pending" (not yet priced), rendered specially by the picker. */
  value: number
  status: string
  /** '' for RFQ-sourced rows — RFQData carries no division field. */
  division: string
  year: number
  createdAt: string
  /** Raw JSON blob of seed line items ('' if none) — parsed client-side by
   * the VM (parseSeedLineItems), same shape old screen's product_details held. */
  productDetails: string
}

/** The editable INPUT fields of one costing line. Computed fields (fobBHD,
 * landedCost, suggestedPriceUnit, …) are deliberately NOT stored here — the
 * VM's calcLine(row) derives them fresh on every read (LineItemsEditor's
 * contract: readonly cells call vm.calc(row), never a stored field), so a
 * currency/percent edit can never leave a stale computed value behind. */
export interface CostingLineRow {
  equipment: string
  model: string
  longCode: string
  detailedDescription: string
  currency: string
  quantity: number
  fobForeign: number
  freightPercent: number
  customsPercent: number
  handlingPercent: number
  financePercent: number
  insurance: number
  otherCosts: number
  marginPercent: number
  userPrice: number
  userPriceSet: boolean
}

export interface CostingCustomerRow {
  id: string
  businessName: string
  /** CustomerMaster has no contact_person/primary_contact field in the
   * verified Go struct (models.ts:616) — best-effort read off the raw record,
   * same tolerant-field doctrine as rfqs.ts's productCountFrom. Unverified. */
  contactPerson: string
}

export interface CostingRevisionRow {
  id: number
  revisionNumber: number
  isActive: boolean
  status: string
  createdAt: string
  createdBy: string
  finalPrice: number
  /** Non-empty when this revision already produced a real Offer — drives the
   * Save-as-Offer confirm-before-overwrite hot-zone gate. */
  offerNumber: string
  /** Raw JSON blob (CostingLineRow[]) — may be malformed; callers must
   * try/catch (see VM.selectRevision). */
  items: string
}

export interface CostingSheetSummaryRow {
  ref: string
  customerName: string
  totalSellBHD: number
}

export interface CostingSettings {
  vatRatePercent: number
  defaultMarginPercent: number
}

export interface CostingHeaderDraft {
  division: string
  date: string
  preparedBy: string
  customerId: string
  customerName: string
  contactPerson: string
  rfqReference: string
  folderNumber: string
  costingId: string
  subject: string
  quoteType: string
  estDelivery: string
  deliveryTerms: string
  paymentTerms: string
  orderType: string
  countryOfOrigin: string
  cocCoo: string
  testCertificate: string
  installation: string
  commissioning: string
  testing: string
  placeOfSupply: string
  taxCategory: string
  customerTRN: string
}

/** Payload shape for the save/export mutations — a lab-scoped subset of the
 * real Go main.CostingExportData (models.ts:8816); the mutations are
 * INTEG-gapped so this never crosses the wire for real, but keeps the mock
 * round-trip honest about what a save/export call carries. */
export interface CostingExportPayload {
  header: CostingHeaderDraft
  lines: CostingLineRow[]
  body: string
  termsAndConditions: string
  subtotal: number
  discount: number
  netAmount: number
  vat: number
  grandTotal: number
  totalCost: number
  profit: number
  profitPercent: number
  hiddenCharges: number
  opportunityId: number
  opportunityRecordId: string
  projectName: string
}

/** One line of the FLAT export payload that crosses the wire to the real
 * `SaveCostingAsOffer(main.CostingExportData)` binding. Mirrors
 * models.ts `main.CostingExportLineItem` field-for-field (camelCase json
 * keys). Unlike CostingLineRow (input-only), this carries the COMPUTED
 * per-line values (fobBHD, suggestedPrice, totalPrice, …) the VM derives via
 * calcLine — the backend stores them for detailed-costing persistence and
 * uses suggestedPrice/totalPrice to build the offer's line items. */
export interface CostingExportLineItem {
  slNo: number
  supplier: string
  equipment: string
  model: string
  serialNumber: string
  longCode: string
  specification: string
  detailedDescription: string
  currency: string
  quantity: number
  fob: number
  freight: number
  freightPercent: number
  totalCost: number
  marginPercent: number
  markupPercent: number
  suggestedPrice: number
  totalPrice: number
  exchangeRate: number
  fobBHD: number
  freightBHD: number
  insurance: number
  customsPercent: number
  customsBHD: number
  handlingPercent: number
  handlingBHD: number
  financePercent: number
  financeBHD: number
  otherCosts: number
  userPrice: number
  userPriceSet: boolean
}

/** The FLAT CostingExportData the real save-as-offer binding takes — mirrors
 * models.ts `main.CostingExportData` exactly (only the two datasheet-attachment
 * fields, attachmentScopeId/attachments, are omitted: this lab has no
 * attachment surface and the Go side treats them as optional — empty scope =
 * no datasheets merged). Assembled by the VM's buildCostingExportData(); the
 * bridge maps it 1:1 to the binding arg. `offerId` empty ⇒ CREATE a new offer
 * (offers:create); a non-empty UUID would route the server's UPDATE path. */
export interface CostingExportData {
  division: string
  source: string
  offerId: string
  offerNumber: string
  date: string
  preparedBy: string
  customerId: string
  customerName: string
  contactPerson: string
  rfqReference: string
  folderNumber: string
  costingId: string
  subject: string
  estDelivery: string
  deliveryTerms: string
  paymentTerms: string
  orderType: string
  countryOfOrigin: string
  cocCoo: string
  testCertificate: string
  installation: string
  commissioning: string
  testing: string
  quoteType: string
  vatRate: number
  hiddenCharges: number
  placeOfSupply: string
  taxCategory: string
  customerTRN: string
  body: string
  lineItems: CostingExportLineItem[]
  subtotal: number
  discount: number
  netAmount: number
  vat: number
  grandTotal: number
  totalCost: number
  profit: number
  profitPercent: number
  opportunityId: number
  opportunityRecordId: string
  projectName: string
  termsAndConditions: string
}

/* ---- shared mock helpers (see bridge/mock.ts pattern) ---- */

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')

function parseYear(value: unknown): number {
  const n = Number(value)
  if (Number.isFinite(n) && n >= 2000 && n <= 2100) return n
  if (value) {
    const d = new Date(value as string)
    if (!Number.isNaN(d.getTime())) return d.getFullYear()
  }
  return new Date().getFullYear()
}

const CUSTOMERS = [
  'Gulf Fabrication W.L.L.',
  'Gulf Fabrication WLL', // near-duplicate of the row above — exercises fuzzy customer matching
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
const EQUIPMENT_POOL = [
  'Coriolis Flow Meter',
  'Pressure Transmitter',
  'Level Radar',
  'Control Valve',
  'Temperature Transmitter',
  'Analyzer Shelter Unit',
  'Positioner',
  'RTD Sensor Assembly',
]

function shiftDate(offsetDays: number): string {
  const d = new Date()
  d.setUTCDate(d.getUTCDate() + offsetDays)
  return d.toISOString().slice(0, 10)
}

/* ---- Opportunities: pipeline+rfq mix, adversarially seasoned (see
 * BUILD_CONTEXT §Adversarial mock). Deterministic (seeded LCG). ---- */

let oppCache: CostingOpportunityRow[] | null = null
/** RFQ-sourced row (source:'rfq') whose id is reserved for a specific
 * revisions adversarial scenario — named for readability at the call sites
 * below (fetchRevisionsForRFQ branches on these). */
const RFQ_ZERO_REVISIONS_ID = 2
const RFQ_ACTIVE_NOT_FIRST_ID = 6
const RFQ_MALFORMED_ITEMS_ID = 10
const RFQ_LINKED_OFFER_ID = 14
/** Pipeline-sourced row whose line items stress the 100-row cap and every
 * per-line adversarial case in one seed (see mockLineItemsFor). */
const STRESS_OPPORTUNITY_ID = 'opp-pipeline-13'

function generateOpportunities(): CostingOpportunityRow[] {
  const rand = lcg(20260715)
  const rows: CostingOpportunityRow[] = []
  const n = 60
  for (let i = 1; i <= n; i++) {
    const source: CostingOpportunityRow['source'] = i % 2 === 0 ? 'rfq' : 'pipeline'
    const monthIdx = Math.floor(rand() * 18)
    const year0 = 2025 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const createdAt = `${year0}-${pad(month, 2)}-${pad(day, 2)}`

    let customer = CUSTOMERS[i % CUSTOMERS.length]!
    let project = PROJECTS[i % PROJECTS.length]!
    let value = i % 13 === 0 ? 0 : Math.round(rand() * 900_000) / 100
    let yearRaw: number | string = year0
    const status = i % 47 === 0 ? 'UNKNOWN_STAGE' : STATUSES[Math.floor(rand() * STATUSES.length)]!

    // Adversarial seasoning at deterministic positions (BUILD_CONTEXT):
    if (i === 5) customer = '' // empty customer name
    if (i === 11) customer = 'مؤسسة الخليج الدولية للتصنيع والتجارة ذ.م.م' // RTL name
    if (i === 17) project = 'Complete replacement and recommissioning of the entire flow metering, pressure transmission, level radar, and control valve instrumentation loop across Train 1 through Train 4 of the process unit including spares, calibration, and full documentation handover '.slice(0, 200)
    if (i === 29) yearRaw = 9999 // out-of-range year — parseYear must clamp

    // RFQ-sourced ids are plain numeric strings (matching realFetchOpportunities'
    // mapRFQToOpportunity, which uses the raw RFQData.id) — the VM does
    // Number(opp.id) to call GetCostingsByRFQ/CreateCostingSheet, so a mock id
    // must parse the same way the real one does, not a prefixed "rfq-N" token.
    const id = source === 'rfq' ? String(i) : `opp-pipeline-${i}`
    const ref = source === 'rfq' ? `RFQ-${pad(i, 4)}` : `OPP-${year0}-${pad(i, 4)}`
    const folderRef = source === 'rfq' ? ref : `RFQ-${pad(i, 4)}` // deliberately never collides in this fixture

    // A couple of rows carry small structured seed line items — same shape
    // old screen's product_details JSON blob held (description/part_number/
    // unit_price/quantity/currency).
    let productDetails = ''
    if (i % 9 === 0 && i !== 29) {
      productDetails = JSON.stringify([
        { description: EQUIPMENT_POOL[i % EQUIPMENT_POOL.length], part_number: `PN-${pad(i, 3)}`, unit_price: Math.round(rand() * 2000 * 100) / 100, quantity: 1 + Math.floor(rand() * 4), currency: 'EUR' },
        { description: EQUIPMENT_POOL[(i + 1) % EQUIPMENT_POOL.length], part_number: `PN-${pad(i + 1, 3)}`, unit_price: Math.round(rand() * 1200 * 100) / 100, quantity: 1 + Math.floor(rand() * 3), currency: 'USD' },
      ])
    }

    rows.push({
      id,
      source,
      ref,
      folderRef,
      customer,
      project,
      value,
      status,
      division: source === 'pipeline' ? MOCK_DIVISIONS[i % MOCK_DIVISIONS.length]! : '',
      year: parseYear(yearRaw),
      createdAt,
      productDetails,
    })
  }
  return rows
}

async function mockFetchOpportunities(): Promise<CostingOpportunityRow[]> {
  oppCache ??= generateOpportunities()
  await sleep(250)
  return [...oppCache]
}

/* ---- Line items: the per-opportunity seed set. One designated pipeline
 * opportunity (STRESS_OPPORTUNITY_ID) returns exactly 100 rows — the
 * LineItemsEditor maxRows cap — with every per-line adversarial case folded
 * into the first 8 rows. ---- */

function mockLineItemsFor(row: CostingOpportunityRow): CostingLineRow[] | null {
  if (row.id !== STRESS_OPPORTUNITY_ID) return null
  const rand = lcg(20260715 + 42)
  const rows: CostingLineRow[] = []
  for (let i = 0; i < 100; i++) {
    const base: CostingLineRow = {
      equipment: `${EQUIPMENT_POOL[i % EQUIPMENT_POOL.length]} #${i + 1}`,
      model: `MOD-${pad(i + 1, 4)}`,
      longCode: `LC-${pad(i + 1, 6)}-CFG`,
      detailedDescription: '',
      currency: CURRENCY_OPTIONS[i % CURRENCY_OPTIONS.length]!,
      quantity: 1 + Math.floor(rand() * 5),
      fobForeign: Math.round(rand() * 5_000 * 100) / 100,
      freightPercent: 9,
      customsPercent: 5,
      handlingPercent: 4,
      financePercent: 1,
      insurance: Math.round(rand() * 50 * 100) / 100,
      otherCosts: 0,
      marginPercent: 20,
      userPrice: 0,
      userPriceSet: false,
    }
    switch (i) {
      case 0:
        base.quantity = 0 // clamp to 1
        break
      case 1:
        base.fobForeign = -500 // clamp to 0
        break
      case 2:
        base.currency = 'XYZ' // unknown currency -> 0.45 fallback
        break
      case 3:
        base.userPriceSet = true
        base.userPrice = 0 // falls through to suggested
        break
      case 4:
        base.equipment = ''
        base.fobForeign = 0 // excluded from "valid" lines
        break
      case 5:
        base.fobForeign = 999_999_999 // layout stress
        break
      case 6:
        base.marginPercent = 0
        break
      case 7:
        base.marginPercent = 500
        break
      default:
        break
    }
    rows.push(base)
  }
  return rows
}

async function mockFetchOpportunityLineItems(row: CostingOpportunityRow): Promise<CostingLineRow[]> {
  await sleep(180)
  const stress = mockLineItemsFor(row)
  if (stress) return stress
  return [] // callers fall back to a single blank line when this is empty
}

/* ---- Revisions: RFQ-scoped costing-sheet history. ---- */

function mockLineItemsAsJSON(seedIdx: number): string {
  const rand = lcg(20260715 + 100 + seedIdx)
  const lines: CostingLineRow[] = Array.from({ length: 2 + (seedIdx % 3) }, (_, k) => ({
    equipment: `${EQUIPMENT_POOL[(seedIdx + k) % EQUIPMENT_POOL.length]}`,
    model: `MOD-R${seedIdx}${k}`,
    longCode: '',
    detailedDescription: '',
    currency: 'BHD',
    quantity: 1 + Math.floor(rand() * 3),
    fobForeign: Math.round(rand() * 3000 * 100) / 100,
    freightPercent: 9,
    customsPercent: 5,
    handlingPercent: 4,
    financePercent: 1,
    insurance: 0,
    otherCosts: 0,
    marginPercent: 20,
    userPrice: 0,
    userPriceSet: false,
  }))
  return JSON.stringify(lines)
}

function generateRevisions(rfqId: number): CostingRevisionRow[] {
  if (rfqId === RFQ_ZERO_REVISIONS_ID) return []

  if (rfqId === RFQ_ACTIVE_NOT_FIRST_ID) {
    // is_active flag on the 2nd entry, not the first — the VM must find by
    // the flag, never assume list order.
    return [
      { id: 6001, revisionNumber: 1, isActive: false, status: 'Draft', createdAt: shiftDate(-40), createdBy: 'A. Yusuf', finalPrice: 4200, offerNumber: '', items: mockLineItemsAsJSON(1) },
      { id: 6002, revisionNumber: 2, isActive: true, status: 'Approved', createdAt: shiftDate(-20), createdBy: 'A. Yusuf', finalPrice: 4550, offerNumber: '', items: mockLineItemsAsJSON(2) },
      { id: 6003, revisionNumber: 3, isActive: false, status: 'Draft', createdAt: shiftDate(-5), createdBy: 'M. Haidar', finalPrice: 4700, offerNumber: '', items: mockLineItemsAsJSON(3) },
    ]
  }

  if (rfqId === RFQ_MALFORMED_ITEMS_ID) {
    return [
      { id: 10001, revisionNumber: 1, isActive: true, status: 'Draft', createdAt: shiftDate(-10), createdBy: 'S. Kanoo', finalPrice: 1800, offerNumber: '', items: '{not valid json' },
      { id: 10002, revisionNumber: 2, isActive: false, status: 'Draft', createdAt: shiftDate(-2), createdBy: 'S. Kanoo', finalPrice: 1900, offerNumber: '', items: mockLineItemsAsJSON(4) },
    ]
  }

  if (rfqId === RFQ_LINKED_OFFER_ID) {
    // Already produced a real Offer — Save-as-Offer must confirm overwrite.
    return [
      { id: 14001, revisionNumber: 1, isActive: true, status: 'Approved', createdAt: shiftDate(-15), createdBy: 'F. Al Khalifa', finalPrice: 9800, offerNumber: 'OFR-0142', items: mockLineItemsAsJSON(5) },
    ]
  }

  // Ordinary RFQs: ~2/3 have a plain single active revision, ~1/3 have none.
  if (rfqId % 3 === 0) return []
  return [
    { id: 20000 + rfqId, revisionNumber: 1, isActive: true, status: 'Draft', createdAt: shiftDate(-30), createdBy: 'A. Yusuf', finalPrice: Math.round((rfqId * 137.5) % 9000) + 500, offerNumber: '', items: mockLineItemsAsJSON(rfqId) },
  ]
}

async function mockFetchRevisions(rfqId: number): Promise<CostingRevisionRow[]> {
  await sleep(200)
  return generateRevisions(rfqId)
}

/* ---- Customers / Prepared-By / Settings / Recent sheets ---- */

let custCache: CostingCustomerRow[] | null = null
function generateCustomers(): CostingCustomerRow[] {
  return CUSTOMERS.map((name, i) => ({
    id: `cust-${i + 1}`,
    businessName: name,
    contactPerson: name ? `Contact ${i + 1}` : '',
  }))
}
async function mockFetchCustomers(): Promise<CostingCustomerRow[]> {
  custCache ??= generateCustomers()
  await sleep(200)
  return [...custCache]
}

async function mockFetchPreparedByOptions(): Promise<string[]> {
  await sleep(120)
  return ['A. Yusuf', 'M. Haidar', 'S. Kanoo', 'F. Al Khalifa']
}

/** Settings reject deterministically in mock mode — the VM catches this and
 * falls back to VAT 10% / margin 20% (BUILD_CONTEXT adversarial case). */
async function mockFetchSettings(): Promise<CostingSettings> {
  await sleep(100)
  throw new Error('Settings unavailable (mock)')
}

let sheetCache: CostingSheetSummaryRow[] | null = null
function generateRecentSheets(): CostingSheetSummaryRow[] {
  const rand = lcg(20260715 + 7)
  return Array.from({ length: 8 }, (_, i) => ({
    ref: `CS-${pad(i + 1, 4)}`,
    customerName: CUSTOMERS[i % CUSTOMERS.length]!,
    totalSellBHD: Math.round(rand() * 15_000 * 1000) / 1000,
  }))
}
async function mockFetchRecentSheets(limit: number): Promise<CostingSheetSummaryRow[]> {
  sheetCache ??= generateRecentSheets()
  await sleep(150)
  return sheetCache.slice(0, limit)
}

/* ---- Mutations: mock actually works (lab demo); real is INTEG-gapped. ---- */

let mockRevisionSeq = 90000
async function mockCreateCostingSheet(_rfqId: number, _items: string, _preparedBy: string): Promise<{ id: number; revisionNumber: number }> {
  await sleep(200)
  mockRevisionSeq += 1
  return { id: mockRevisionSeq, revisionNumber: 1 }
}
async function mockUpdateCostingSheet(id: number, _items: string, _preparedBy: string): Promise<void> {
  await sleep(200)
  void id
}
async function mockCloneCostingAsNewRevision(_sourceId: number, _preparedBy: string): Promise<{ id: number; revisionNumber: number }> {
  await sleep(200)
  mockRevisionSeq += 1
  return { id: mockRevisionSeq, revisionNumber: 2 }
}
async function mockSetActiveCostingRevision(_id: number): Promise<void> {
  await sleep(150)
}
let mockOfferSeq = 200
async function mockSaveCostingAsOffer(_data: CostingExportData): Promise<{ offerNumber: string }> {
  await sleep(250)
  mockOfferSeq += 1
  return { offerNumber: `OFR-${pad(mockOfferSeq, 4)}` }
}
async function mockExportCostingToPDF(_payload: CostingExportPayload): Promise<string> {
  await sleep(300)
  return 'C:\\Exports\\costing-mock.pdf'
}
async function mockExportCostingToExcel(_payload: CostingExportPayload): Promise<string> {
  await sleep(300)
  return 'C:\\Exports\\costing-mock.xlsx'
}
async function mockOpenExportedFile(_path: string): Promise<void> {
  await sleep(100)
}

/* ---- real: fetches WIRED (opportunities merge, line items, revisions,
 * customers, prepared-by, settings, recent sheets); every mutation +
 * OpenExportedFile (side-effecting) INTEG-gapped, naming the exact binding. ---- */

function mapRFQToOpportunity(r: Record<string, unknown>): CostingOpportunityRow {
  const rfqNumber = str(r.rfq_number)
  return {
    id: str(num(r.id)),
    source: 'rfq',
    ref: str(r.rfq_ref) || rfqNumber || `RFQ-${str(num(r.id))}`,
    folderRef: rfqNumber,
    customer: str(r.client),
    project: str(r.project),
    value: num(r.value),
    status: str(r.status) || str(r.stage),
    division: '',
    year: parseYear(r.created_at),
    createdAt: goDate(r.created_at),
    productDetails: str(r.product_details),
  }
}

function mapPipelineToOpportunity(o: Record<string, unknown>): CostingOpportunityRow {
  return {
    id: str(o.id),
    source: 'pipeline',
    ref: str(o.eh_ref) || str(o.folder_number),
    folderRef: str(o.folder_number),
    customer: str(o.customer_name),
    project: str(o.title) || str(o.folder_name),
    value: num(o.revenue_bhd),
    status: str(o.stage),
    division: str(o.division),
    year: parseYear(o.year),
    createdAt: goDate(o.updated_at) || goDate(o.offer_date),
    productDetails: str(o.product_details),
  }
}

async function realFetchOpportunities(): Promise<CostingOpportunityRow[]> {
  const [rfqs, pipeline] = await Promise.all([GetRFQs(200, 0), GetPipelineOpportunities(500, 0)])
  const rfqRows = (rfqs ?? [])
    .filter((r) => str((r as unknown as Record<string, unknown>).rfq_number).trim())
    .map((r) => mapRFQToOpportunity(r as unknown as Record<string, unknown>))
  const rfqFolders = new Set(rfqRows.map((r) => r.folderRef).filter(Boolean))
  const pipelineRows = (pipeline ?? [])
    .map((o) => mapPipelineToOpportunity(o as unknown as Record<string, unknown>))
    .filter((p) => !rfqFolders.has(p.folderRef))
  return [...pipelineRows, ...rfqRows].sort((a, b) => b.createdAt.localeCompare(a.createdAt))
}

function mapOfferItemToLine(o: Record<string, unknown>): CostingLineRow {
  const fobForeign = num(o.fob)
  const freight = num(o.freight)
  return {
    equipment: str(o.equipment) || str(o.description),
    model: str(o.model) || str(o.product_code),
    longCode: str(o.long_code),
    detailedDescription: str(o.detailed_description),
    currency: str(o.currency) || 'BHD',
    quantity: num(o.quantity) || 1,
    fobForeign,
    freightPercent: fobForeign > 0 ? (freight / fobForeign) * 100 : 9,
    customsPercent: num(o.customs_percent) || 5,
    handlingPercent: num(o.handling_percent) || 4,
    financePercent: num(o.finance_percent) || 1,
    insurance: num(o.insurance),
    otherCosts: num(o.other_costs),
    marginPercent: num(o.margin_percent),
    userPrice: num(o.user_price),
    userPriceSet: Boolean(o.user_price_set),
  }
}

async function realFetchOpportunityLineItems(row: CostingOpportunityRow): Promise<CostingLineRow[]> {
  if (row.source !== 'pipeline') return []
  const items = await GetOpportunityLineItems(row.id)
  return (items ?? []).map((o) => mapOfferItemToLine(o as unknown as Record<string, unknown>))
}

function mapRevision(r: Record<string, unknown>): CostingRevisionRow {
  return {
    id: num(r.id),
    revisionNumber: num(r.revision_number),
    isActive: Boolean(r.is_active),
    status: str(r.status),
    createdAt: goDate(r.created_at),
    createdBy: str(r.created_by),
    finalPrice: num(r.final_price),
    offerNumber: str(r.offer_number),
    items: str(r.items),
  }
}

async function realFetchRevisions(rfqId: number): Promise<CostingRevisionRow[]> {
  const rows = await GetCostingsByRFQ(rfqId)
  return (rows ?? []).map((r) => mapRevision(r as unknown as Record<string, unknown>))
}

function mapCustomer(c: Record<string, unknown>): CostingCustomerRow {
  return {
    id: str(c.id),
    businessName: str(c.business_name),
    // contact_person is not a verified CustomerMaster field (see doc above) —
    // read defensively off the raw record, blank if absent.
    contactPerson: str(c.contact_person) || str(c.primary_contact),
  }
}

async function realFetchCustomers(): Promise<CostingCustomerRow[]> {
  const rows = await ListCustomers(500, 0)
  return (rows ?? []).map((c) => mapCustomer(c as unknown as Record<string, unknown>))
}

async function realFetchPreparedByOptions(): Promise<string[]> {
  const rows = await GetPreparedByOptions()
  return rows ?? []
}

/** GetSettings returns an unverified Record<string, any> (same caveat as
 * business-settings.ts) — old screen read settingsResult.business.vat_rate /
 * .default_margin; mapped defensively here with the same fallback (10/20)
 * the mock's rejection path exercises. */
async function realFetchSettings(): Promise<CostingSettings> {
  const r = (await GetSettings()) as Record<string, unknown> | null
  const business = (r?.business ?? {}) as Record<string, unknown>
  const vat = business.vat_rate
  const margin = business.default_margin
  return {
    vatRatePercent: typeof vat === 'number' ? vat : 10,
    defaultMarginPercent: typeof margin === 'number' ? margin : 20,
  }
}

function mapSheetSummary(s: Record<string, unknown>): CostingSheetSummaryRow {
  return {
    ref: str(s.rfq_name) || str(s.offer_number) || `CS-${str(num(s.id))}`,
    customerName: str(s.customer_name),
    totalSellBHD: num(s.total_value_bhd) || num(s.final_price),
  }
}

async function realFetchRecentSheets(limit: number): Promise<CostingSheetSummaryRow[]> {
  const rows = await GetCostingSheets(limit)
  return (rows ?? []).map((s) => mapSheetSummary(s as unknown as Record<string, unknown>))
}

async function realCreateCostingSheet(rfqId: number, items: string, preparedBy: string): Promise<{ id: number; revisionNumber: number }> {
  // CRMService.CreateCostingSheet(rfqId number, items string, preparedBy string) -> CostingSheetData
  const result = (await CreateCostingSheet(rfqId, items, preparedBy)) as unknown as Record<string, unknown>
  return { id: num(result.id), revisionNumber: num(result.revision_number) }
}
async function realUpdateCostingSheet(_id: number, _items: string, _preparedBy: string): Promise<void> {
  // GAP: binding is UpdateCostingSheet(id number, data main.CostingSheetData) -> CostingSheetData.
  // arg2 is a FULL CostingSheetData struct (rfq_id, final_price, subtotal, margin_percent, …),
  // but this call only carries (items, preparedBy) — cannot assemble the struct without guessing
  // required pricing/linkage fields. A wrong wire is worse than an honest gap.
  throw new Error('INTEG gap: UpdateCostingSheet — binding takes (id, CostingSheetData struct); this call only carries (items, preparedBy), which cannot assemble a full CostingSheetData')
}
async function realCloneCostingAsNewRevision(sourceId: number, preparedBy: string): Promise<{ id: number; revisionNumber: number }> {
  // CRMService.CloneCostingAsNewRevision(sourceId number, preparedBy string) -> CostingSheetData
  const result = (await CloneCostingAsNewRevision(sourceId, preparedBy)) as unknown as Record<string, unknown>
  return { id: num(result.id), revisionNumber: num(result.revision_number) }
}
async function realSetActiveCostingRevision(id: number): Promise<void> {
  // CRMService.SetActiveCostingRevision(id number) -> void
  await SetActiveCostingRevision(id)
}
async function realSaveCostingAsOffer(data: CostingExportData): Promise<{ offerNumber: string }> {
  // HOT-ZONE: creates an Offer. The VM (buildCostingExportData) assembles the
  // FLAT CostingExportData with per-line COMPUTED values from calcLine, so the
  // shapes now map 1:1 to main.CostingExportData — no guessing. `offerId` is
  // empty ⇒ the server's CREATE path (offers:create); its duplicate/uniqueness
  // guards (one active offer per RFQ, unique offer number) surface honestly.
  // The two optional attachment fields are absent (no lab attachment surface);
  // the structural cast satisfies the generated model type (map.ts precedent).
  const offer = await SaveCostingAsOffer(data as unknown as main.CostingExportData)
  const rec = offer as unknown as Record<string, unknown>
  return { offerNumber: str(rec.offer_number) }
}
async function realExportCostingToPDF(_payload: CostingExportPayload): Promise<string> {
  throw new Error('INTEG gap: ExportCostingToPDF — wires at K5')
}
async function realExportCostingToExcel(_payload: CostingExportPayload): Promise<string> {
  throw new Error('INTEG gap: ExportCostingToExcel — wires at K5')
}
async function realOpenExportedFile(_path: string): Promise<void> {
  throw new Error('INTEG gap: OpenExportedFile — wires at K5 (side-effecting: opens a file on disk)')
}

/* ---- public switched API (VM imports THESE) ---- */

export const fetchCostingOpportunities = (): Promise<CostingOpportunityRow[]> =>
  pick(realFetchOpportunities, mockFetchOpportunities)()
export const fetchOpportunityLineItems = (row: CostingOpportunityRow): Promise<CostingLineRow[]> =>
  pick(realFetchOpportunityLineItems, mockFetchOpportunityLineItems)(row)
export const fetchRevisionsForRFQ = (rfqId: number): Promise<CostingRevisionRow[]> =>
  pick(realFetchRevisions, mockFetchRevisions)(rfqId)
export const fetchCostingCustomers = (): Promise<CostingCustomerRow[]> =>
  pick(realFetchCustomers, mockFetchCustomers)()
export const fetchPreparedByOptions = (): Promise<string[]> =>
  pick(realFetchPreparedByOptions, mockFetchPreparedByOptions)()
export const fetchCostingSettings = (): Promise<CostingSettings> =>
  pick(realFetchSettings, mockFetchSettings)()
export const fetchRecentCostingSheets = (limit: number): Promise<CostingSheetSummaryRow[]> =>
  pick(realFetchRecentSheets, mockFetchRecentSheets)(limit)

export const createCostingSheet = (rfqId: number, items: string, preparedBy: string): Promise<{ id: number; revisionNumber: number }> =>
  pick(realCreateCostingSheet, mockCreateCostingSheet)(rfqId, items, preparedBy)
export const updateCostingSheet = (id: number, items: string, preparedBy: string): Promise<void> =>
  pick(realUpdateCostingSheet, mockUpdateCostingSheet)(id, items, preparedBy)
export const cloneCostingAsNewRevision = (sourceId: number, preparedBy: string): Promise<{ id: number; revisionNumber: number }> =>
  pick(realCloneCostingAsNewRevision, mockCloneCostingAsNewRevision)(sourceId, preparedBy)
export const setActiveCostingRevision = (id: number): Promise<void> =>
  pick(realSetActiveCostingRevision, mockSetActiveCostingRevision)(id)
export const saveCostingAsOffer = (data: CostingExportData): Promise<{ offerNumber: string }> =>
  pick(realSaveCostingAsOffer, mockSaveCostingAsOffer)(data)
export const exportCostingToPDF = (payload: CostingExportPayload): Promise<string> =>
  pick(realExportCostingToPDF, mockExportCostingToPDF)(payload)
export const exportCostingToExcel = (payload: CostingExportPayload): Promise<string> =>
  pick(realExportCostingToExcel, mockExportCostingToExcel)(payload)
export const openExportedFile = (path: string): Promise<void> =>
  pick(realOpenExportedFile, mockOpenExportedFile)(path)
