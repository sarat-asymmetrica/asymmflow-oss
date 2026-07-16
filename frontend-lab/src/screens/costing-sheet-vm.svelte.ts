/* Costing Sheet viewmodel — L5's reactive half: opportunity picker state,
 * header/line-item/revision state, and the SACRED pricing waterfall. The
 * per-line math (calcLine) and sheet totals below are ported VERBATIM from
 * the old CostingSheetScreen.svelte's calculateLineItem (~lines 525-582) and
 * its summary $derived block (~lines 759-770) — every fallback, every
 * rounding op, and the profit/cost asymmetry are preserved byte-for-byte.
 * Do NOT "fix" anything here without a stop-and-ask; see the parity doc.
 *
 * Named `costing-sheet-vm` (not `costing-sheet.svelte.ts`) so its stem never
 * differs from `CostingSheet.svelte` by case only — that collides under
 * TypeScript's case-insensitive file resolution on Windows (see
 * book-bank-recon.svelte.ts's identical note). */

import {
  CURRENCY_RATES,
  FALLBACK_EXCHANGE_RATE,
  costingDivisionOptions,
  defaultCostingDivision,
  fetchCostingOpportunities,
  fetchOpportunityLineItems,
  fetchRevisionsForRFQ,
  fetchCostingCustomers,
  fetchPreparedByOptions,
  fetchCostingSettings,
  fetchRecentCostingSheets,
  createCostingSheet,
  updateCostingSheet,
  cloneCostingAsNewRevision,
  setActiveCostingRevision,
  saveCostingAsOffer,
  exportCostingToPDF,
  exportCostingToExcel,
  openExportedFile,
  type CostingOpportunityRow,
  type CostingLineRow,
  type CostingCustomerRow,
  type CostingRevisionRow,
  type CostingSheetSummaryRow,
  type CostingSettings,
  type CostingHeaderDraft,
  type CostingExportPayload,
  type CostingExportData,
  type CostingExportLineItem,
} from '../bridge/costing-sheet'

/* ---- pure numeric helpers (verbatim toFiniteNumber / nonNegativeNumber) ---- */

function toFiniteNumber(value: unknown, fallback = 0): number {
  if (value === null || value === undefined || value === '') return fallback
  const n = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(n) ? n : fallback
}
function nonNegativeNumber(value: unknown, fallback = 0): number {
  return Math.max(0, toFiniteNumber(value, fallback))
}

/* ---- the sacred per-line waterfall (verbatim order + fallbacks) ----
 * NOTE on freight%/margin%: the old code calls
 * `nonNegativeNumber(item.freightPercent)` and
 * `nonNegativeNumber(item.marginPercent)` with NO second argument at
 * calc-time, so an invalid/blank value falls back to 0 — NOT 9 / 20. The
 * "9" and "20" defaults only ever apply when a NEW blank line is created
 * (createBlankLine below), never as a calc-time rescue. Ported verbatim; do
 * not "fix" this asymmetry between customs/handling/finance (which DO
 * fall back to 5/4/1 at calc-time) and freight/margin (which fall back to 0). */
export interface CostingLineCalc {
  exchangeRate: number
  quantity: number
  fobBHD: number
  freightForeign: number
  freightBHD: number
  cf: number
  customsBHD: number
  landedCost: number
  handlingBHD: number
  financeBHD: number
  totalCost: number
  sellingPrice: number
  marginBHD: number
  suggestedPriceUnit: number
  effectivePrice: number
  totalSuggestedPrice: number
}

export function calcLine(line: CostingLineRow): CostingLineCalc {
  // 1. exchangeRate from currency table (fallback 0.45, clamp > 0).
  let exchangeRate = CURRENCY_RATES[line.currency] ?? FALLBACK_EXCHANGE_RATE
  if (!(exchangeRate > 0)) exchangeRate = FALLBACK_EXCHANGE_RATE

  // 2. quantity = max(1, qty).
  const quantity = Math.max(1, toFiniteNumber(line.quantity, 1))

  const fobForeign = nonNegativeNumber(line.fobForeign)
  const insurance = nonNegativeNumber(line.insurance)
  const customsPercent = nonNegativeNumber(line.customsPercent, 5)
  const handlingPercent = nonNegativeNumber(line.handlingPercent, 4)
  const financePercent = nonNegativeNumber(line.financePercent, 1)
  const otherCosts = nonNegativeNumber(line.otherCosts)
  const userPrice = nonNegativeNumber(line.userPrice)

  // 3. fobBHD = fobForeign * rate.
  const fobBHD = fobForeign * exchangeRate
  // 4-5. freight (verbatim: NO default — falls back to 0, not 9).
  const freightPercent = nonNegativeNumber(line.freightPercent)
  const freightForeign = fobForeign * (freightPercent / 100)
  const freightBHD = freightForeign * exchangeRate

  // 6. C&F = FOB + Freight.
  const cf = fobBHD + freightBHD
  // 7. Customs.
  const customsBHD = cf * (customsPercent / 100)
  // 8. Landed cost = C&F + Insurance + Customs.
  const landedCost = cf + insurance + customsBHD
  // 9-10. Handling / finance.
  const handlingBHD = landedCost * (handlingPercent / 100)
  const financeBHD = landedCost * (financePercent / 100)
  // 11. Total cost.
  const totalCost = landedCost + handlingBHD + financeBHD + otherCosts

  // 12. Markup (verbatim: NO default — falls back to 0, not 20).
  const marginPercent = nonNegativeNumber(line.marginPercent)
  const sellingPrice = totalCost * (1 + marginPercent / 100)
  // 13. Margin amount.
  const marginBHD = sellingPrice - totalCost
  // 14. Suggested price — Math.ceil, the ONLY rounding, rounds UP. Never
  // change to Math.round.
  const suggestedPriceUnit = Math.ceil(sellingPrice)
  // 15. Effective price: user override wins only when set AND > 0.
  const effectivePrice = line.userPriceSet && userPrice > 0 ? userPrice : suggestedPriceUnit
  // 16. Total suggested price.
  const totalSuggestedPrice = effectivePrice * quantity

  return {
    exchangeRate,
    quantity,
    fobBHD,
    freightForeign,
    freightBHD,
    cf,
    customsBHD,
    landedCost,
    handlingBHD,
    financeBHD,
    totalCost,
    sellingPrice,
    marginBHD,
    suggestedPriceUnit,
    effectivePrice,
    totalSuggestedPrice,
  }
}

/** A line counts toward export/save gating when it has an equipment name or
 * a positive FOB — verbatim getValidLineItems predicate. */
export function isValidLine(line: CostingLineRow): boolean {
  return Boolean(line.equipment?.trim()) || nonNegativeNumber(line.fobForeign) > 0
}

const DEFAULT_FREIGHT_PERCENT = 9
const DEFAULT_CUSTOMS_PERCENT = 5
const DEFAULT_HANDLING_PERCENT = 4
const DEFAULT_FINANCE_PERCENT = 1
const MAX_LINE_ITEMS = 100

export function createBlankLine(defaultMarginPercent = 20): CostingLineRow {
  return {
    equipment: '',
    model: '',
    longCode: '',
    detailedDescription: '',
    currency: 'BHD',
    quantity: 1,
    fobForeign: 0,
    freightPercent: DEFAULT_FREIGHT_PERCENT,
    customsPercent: DEFAULT_CUSTOMS_PERCENT,
    handlingPercent: DEFAULT_HANDLING_PERCENT,
    financePercent: DEFAULT_FINANCE_PERCENT,
    insurance: 0,
    otherCosts: 0,
    marginPercent: defaultMarginPercent,
    userPrice: 0,
    userPriceSet: false,
  }
}

/** Parses the old product_details-shaped seed JSON (description/part_number/
 * unit_price/quantity/currency) into fresh CostingLineRows. Tolerant of a
 * malformed/absent blob — returns [] rather than throwing (caller decides
 * the blank-line fallback), mirroring parseOpportunitySeedItems. */
export function parseSeedLineItems(productDetailsJSON: string, defaultMarginPercent = 20): CostingLineRow[] {
  if (!productDetailsJSON) return []
  try {
    const parsed: unknown = JSON.parse(productDetailsJSON)
    const items = Array.isArray(parsed) ? parsed : parsed && typeof parsed === 'object' ? [parsed] : []
    return items
      .map((raw) => {
        const item = raw as Record<string, unknown>
        const line = createBlankLine(defaultMarginPercent)
        line.equipment = String(item.description ?? item.equipment ?? item.name ?? '')
        line.model = String(item.part_number ?? item.model ?? item.product_code ?? '')
        line.currency = String(item.currency ?? 'BHD').toUpperCase() in CURRENCY_RATES ? String(item.currency ?? 'BHD').toUpperCase() : 'BHD'
        line.quantity = Number(item.quantity) || 1
        line.fobForeign = Number(item.unit_price) || 0
        return line
      })
      .filter((line) => line.equipment || line.model)
  } catch {
    return []
  }
}

/** Condensed near-duplicate customer resolution — a bounded port of the old
 * screen's namesRepresentSameParty engine (strip legal suffixes, collapse
 * whitespace, substring match on longer names). Not byte-identical to the
 * ~40-line original; documented as a deliberate simplification in the parity
 * doc, not a silent gap. */
function normalizeParty(name: string): string {
  return name
    .trim()
    .toLowerCase()
    .replace(/&/g, ' and ')
    .replace(/[^a-z0-9]+/g, ' ')
    .replace(/\b(wll|llc|bsc|ltd|limited|company|co)\b/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}
export function findMatchingCustomer(customers: CostingCustomerRow[], name: string): CostingCustomerRow | null {
  const target = normalizeParty(name)
  if (!target) return null
  return (
    customers.find((c) => {
      const n = normalizeParty(c.businessName)
      if (!n) return false
      if (n === target) return true
      return n.length >= 8 && target.length >= 8 && (n.includes(target) || target.includes(n))
    }) ?? null
  )
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10)
}

export function createDefaultHeader(division: string): CostingHeaderDraft {
  return {
    division,
    date: todayISO(),
    preparedBy: '',
    customerId: '',
    customerName: '',
    contactPerson: '',
    rfqReference: '',
    folderNumber: '',
    costingId: '',
    subject: '',
    quoteType: 'Quotation',
    estDelivery: '5-7 weeks',
    deliveryTerms: `DAP Bahrain at your store or ${division}`,
    paymentTerms: '30 days from Date of Delivery',
    orderType: 'General',
    countryOfOrigin: 'DE',
    cocCoo: 'No',
    testCertificate: 'Additional charges applicable as per OEM terms',
    installation: 'No',
    commissioning: 'No',
    testing: 'No',
    placeOfSupply: 'Kingdom of Bahrain',
    taxCategory: 'Standard',
    customerTRN: '',
  }
}

const DEFAULT_QUOTATION_BODY = `We thank you for the opportunity and are pleased to submit our techno-commercial offer for your review.

Please find our pricing and scope below. We trust the proposal meets your requirement and look forward to your valued order.`

function defaultTermsAndConditions(division: string, vatRatePercent: number): string {
  return `1. QUOTATION VALIDITY
This quotation is valid for thirty (30) days from the date of issue.

2. PRICES
All prices are in Bahraini Dinars (BHD) unless otherwise stated. Prices are exclusive of VAT (${vatRatePercent}%) which will be added to the invoice.

3. PAYMENT TERMS
As per the payment terms specified in this quotation. Late payments may incur interest charges.

4. DELIVERY
Delivery times are estimates and subject to manufacturer's confirmation. ${division} shall not be liable for delays beyond our control.

5. WARRANTY
All products carry the manufacturer's standard warranty. Extended warranty options are available upon request.

6. INSTALLATION & COMMISSIONING
Installation and commissioning services are available at additional cost unless included in the quotation.

7. FORCE MAJEURE
${division} shall not be liable for failure to perform due to causes beyond reasonable control.

8. GOVERNING LAW
This quotation is governed by the laws of the Kingdom of Bahrain.`
}

export const MAX_COSTING_LINE_ITEMS = MAX_LINE_ITEMS

/* ---- sheet-level totals (verbatim from the old $derived block, ~lines
 * 759-770). Extracted as standalone pure functions — same split as
 * book-bank-recon.svelte.ts's adjustedBankBalance/variance — so the class's
 * $derived fields and the unit tests call the SAME code, not a reimplemented
 * copy. PRESERVE the profit/cost asymmetry byte-for-byte: hiddenCharges is
 * added to cost only; discount/VAT are applied to revenue only. Do not "fix". */
export interface CostingSheetTotals {
  subtotal: number
  effectiveDiscount: number
  effectiveHiddenCharges: number
  effectiveVatRate: number
  netAmount: number
  vat: number
  grandTotal: number
  totalCost: number
  profit: number
  profitPercent: number
}

export function sheetSubtotal(lines: CostingLineRow[]): number {
  return lines.reduce((sum, l) => sum + toFiniteNumber(calcLine(l).totalSuggestedPrice), 0)
}

export function sheetTotals(
  lines: CostingLineRow[],
  discount: number,
  hiddenCharges: number,
  vatRate: number,
): CostingSheetTotals {
  const subtotal = sheetSubtotal(lines)
  const effectiveDiscount = nonNegativeNumber(discount)
  const effectiveHiddenCharges = nonNegativeNumber(hiddenCharges)
  const effectiveVatRate = Math.min(100, nonNegativeNumber(vatRate, 10))
  const netAmount = Math.max(0, subtotal - effectiveDiscount)
  const vat = netAmount * (effectiveVatRate / 100)
  const grandTotal = netAmount + vat
  const totalCost =
    lines.reduce((sum, l) => sum + toFiniteNumber(calcLine(l).totalCost) * Math.max(1, toFiniteNumber(l.quantity, 1)), 0) +
    effectiveHiddenCharges
  const profit = netAmount - totalCost
  const profitPercent = netAmount > 0 ? (profit / netAmount) * 100 : 0
  return { subtotal, effectiveDiscount, effectiveHiddenCharges, effectiveVatRate, netAmount, vat, grandTotal, totalCost, profit, profitPercent }
}

/** Map one input CostingLineRow → the FLAT CostingExportLineItem the real
 * SaveCostingAsOffer binding takes, folding in the sacred waterfall's computed
 * outputs (calcLine). Extracted as a pure function — same split as
 * sheetTotals/sheetSubtotal — so buildCostingExportData and the unit tests
 * exercise the SAME mapping. The suggestedPrice/totalPrice pair is what the
 * backend uses to build the offer's line items; the rest is detailed-costing
 * persistence. markupPercent is 0 so the offer item takes the line's
 * marginPercent (buildOfferItemsFromCostingData's markup-else-margin rule). */
export function costingExportLine(line: CostingLineRow, index: number): CostingExportLineItem {
  const c = calcLine(line)
  return {
    slNo: index + 1,
    supplier: '',
    equipment: line.equipment,
    model: line.model,
    serialNumber: '',
    longCode: line.longCode,
    specification: '',
    detailedDescription: line.detailedDescription,
    currency: line.currency,
    quantity: c.quantity,
    fob: nonNegativeNumber(line.fobForeign),
    freight: c.freightForeign,
    freightPercent: nonNegativeNumber(line.freightPercent),
    totalCost: c.totalCost,
    marginPercent: nonNegativeNumber(line.marginPercent),
    markupPercent: 0,
    suggestedPrice: c.effectivePrice,
    totalPrice: c.totalSuggestedPrice,
    exchangeRate: c.exchangeRate,
    fobBHD: c.fobBHD,
    freightBHD: c.freightBHD,
    insurance: nonNegativeNumber(line.insurance),
    customsPercent: nonNegativeNumber(line.customsPercent, 5),
    customsBHD: c.customsBHD,
    handlingPercent: nonNegativeNumber(line.handlingPercent, 4),
    handlingBHD: c.handlingBHD,
    financePercent: nonNegativeNumber(line.financePercent, 1),
    financeBHD: c.financeBHD,
    otherCosts: nonNegativeNumber(line.otherCosts),
    userPrice: nonNegativeNumber(line.userPrice),
    userPriceSet: line.userPriceSet,
  }
}

export class CostingSheetViewModel {
  // ---- picker mode ----
  loading = $state(true)
  error = $state<string | null>(null)
  opportunities = $state<CostingOpportunityRow[]>([])
  selectedOpportunityId = $state('')
  formOpen = $state(false)

  customers = $state<CostingCustomerRow[]>([])
  preparedByOptions = $state<string[]>([])
  settings = $state<CostingSettings>({ vatRatePercent: 10, defaultMarginPercent: 20 })
  settingsError = $state<string | null>(null)
  recentSheets = $state<CostingSheetSummaryRow[]>([])

  divisionOptions = costingDivisionOptions()

  // ---- form mode ----
  header = $state<CostingHeaderDraft>(createDefaultHeader(defaultCostingDivision()))
  lines = $state<CostingLineRow[]>([createBlankLine()])
  quotationBody = $state(DEFAULT_QUOTATION_BODY)
  termsAndConditions = $state(defaultTermsAndConditions(defaultCostingDivision(), 10))
  discount = $state(0)
  hiddenCharges = $state(0)
  vatRate = $state(10)
  showAdvanced = $state(false)

  // revisions (RFQ-scoped)
  revisions = $state<CostingRevisionRow[]>([])
  selectedRevisionId = $state<number | null>(null)
  loadingRevisions = $state(false)
  revisionError = $state<string | null>(null)
  /** Set when the loaded/current revision already produced a real Offer —
   * drives the Save-as-Offer confirm-before-overwrite hot-zone gate. */
  linkedOfferNumber = $state('')
  currentCostingId = $state<number | null>(null)

  savingCosting = $state(false)
  savingOffer = $state(false)
  saveError = $state<string | null>(null)
  confirmOverwriteOpen = $state(false)
  exporting = $state(false)
  exportError = $state<string | null>(null)
  lastSavedOfferNumber = $state('')

  selectedOpportunity = $derived.by(() => this.opportunities.find((o) => o.id === this.selectedOpportunityId) ?? null)

  previewOpportunities = $derived.by(() =>
    [...this.opportunities]
      .sort((a, b) => {
        const aHasValue = a.value > 0 ? 1 : 0
        const bHasValue = b.value > 0 ? 1 : 0
        if (aHasValue !== bHasValue) return bHasValue - aHasValue
        return b.createdAt.localeCompare(a.createdAt)
      })
      .slice(0, 6),
  )

  isRFQOpportunity = $derived.by(() => this.selectedOpportunity?.source === 'rfq')

  validLines = $derived.by(() => this.lines.filter(isValidLine))

  // ---- sacred sheet totals — call the extracted pure sheetTotals() above,
  // never a reimplementation, so the screen and the unit tests can never
  // silently disagree about what "profit" means.
  totals = $derived.by(() => sheetTotals(this.lines, this.discount, this.hiddenCharges, this.vatRate))
  subtotal = $derived.by(() => this.totals.subtotal)
  effectiveDiscount = $derived.by(() => this.totals.effectiveDiscount)
  effectiveHiddenCharges = $derived.by(() => this.totals.effectiveHiddenCharges)
  effectiveVatRate = $derived.by(() => this.totals.effectiveVatRate)
  netAmount = $derived.by(() => this.totals.netAmount)
  vat = $derived.by(() => this.totals.vat)
  grandTotal = $derived.by(() => this.totals.grandTotal)
  totalCost = $derived.by(() => this.totals.totalCost)
  profit = $derived.by(() => this.totals.profit)
  profitPercent = $derived.by(() => this.totals.profitPercent)

  calc(line: CostingLineRow): CostingLineCalc {
    return calcLine(line)
  }

  /** Customer <select> onchange — pre-fills name/contact from the matched
   * master record, same as the old screen's handleCustomerChange. */
  selectCustomerById(id: string): void {
    const c = this.customers.find((x) => x.id === id)
    this.header = c
      ? { ...this.header, customerId: c.id, customerName: c.businessName, contactPerson: c.contactPerson }
      : { ...this.header, customerId: id }
  }

  blankLine(): CostingLineRow {
    return createBlankLine(this.settings.defaultMarginPercent)
  }

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      const [opps, customers, preparedBy, recentSheets] = await Promise.all([
        fetchCostingOpportunities(),
        fetchCostingCustomers(),
        fetchPreparedByOptions(),
        fetchRecentCostingSheets(8),
      ])
      this.opportunities = opps
      this.customers = customers
      this.preparedByOptions = preparedBy
      this.recentSheets = recentSheets

      try {
        this.settings = await fetchCostingSettings()
        this.settingsError = null
      } catch (e) {
        // Adversarial case: settings reject -> VAT 10% / margin 20% fallback.
        this.settings = { vatRatePercent: 10, defaultMarginPercent: 20 }
        this.settingsError = e instanceof Error ? e.message : String(e)
      }
      this.vatRate = this.settings.vatRatePercent
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  private resetFormState(division: string): void {
    this.header = createDefaultHeader(division)
    this.lines = [this.blankLine()]
    this.quotationBody = DEFAULT_QUOTATION_BODY
    this.termsAndConditions = defaultTermsAndConditions(division, this.effectiveVatRate)
    this.discount = 0
    this.hiddenCharges = 0
    this.revisions = []
    this.selectedRevisionId = null
    this.revisionError = null
    this.linkedOfferNumber = ''
    this.currentCostingId = null
    this.saveError = null
    this.exportError = null
    this.lastSavedOfferNumber = ''
  }

  startBlank(): void {
    this.selectedOpportunityId = ''
    this.resetFormState(defaultCostingDivision())
    this.formOpen = true
  }

  closeForm(): void {
    this.formOpen = false
    this.selectedOpportunityId = ''
  }

  async selectOpportunity(id: string): Promise<void> {
    this.selectedOpportunityId = id
    const opp = this.opportunities.find((o) => o.id === id)
    if (!opp) return

    const division = opp.division || defaultCostingDivision()
    this.resetFormState(division)

    const matchedCustomer = findMatchingCustomer(this.customers, opp.customer)
    this.header = {
      ...this.header,
      customerName: matchedCustomer?.businessName || opp.customer,
      customerId: matchedCustomer?.id ?? '',
      contactPerson: matchedCustomer?.contactPerson ?? '',
      rfqReference: opp.ref,
      folderNumber: opp.folderRef,
      costingId: opp.ref || opp.folderRef,
      subject: opp.project,
      division,
    }

    // Seed line items: structured product_details JSON first, else (pipeline
    // only) a real GetOpportunityLineItems fetch, else a single blank line.
    const seeded = parseSeedLineItems(opp.productDetails, this.settings.defaultMarginPercent)
    if (seeded.length > 0) {
      this.lines = seeded
    } else {
      const fetched = await fetchOpportunityLineItems(opp)
      this.lines = fetched.length > 0 ? fetched : [this.blankLine()]
    }

    if (opp.source === 'rfq') {
      await this.loadRevisionsForRFQ(Number(opp.id))
    }

    this.formOpen = true
  }

  async loadRevisionsForRFQ(rfqId: number): Promise<void> {
    this.loadingRevisions = true
    this.revisionError = null
    try {
      this.revisions = await fetchRevisionsForRFQ(rfqId)
      const active = this.revisions.find((r) => r.isActive)
      if (active) {
        this.applyRevision(active)
      }
    } catch (e) {
      this.revisions = []
      this.revisionError = e instanceof Error ? e.message : String(e)
    } finally {
      this.loadingRevisions = false
    }
  }

  private applyRevision(rev: CostingRevisionRow): void {
    this.selectedRevisionId = rev.id
    this.currentCostingId = rev.id
    this.linkedOfferNumber = rev.offerNumber
    try {
      const parsed: unknown = JSON.parse(rev.items || '[]')
      this.lines = Array.isArray(parsed) && parsed.length > 0 ? (parsed as CostingLineRow[]) : [this.blankLine()]
      this.revisionError = null
    } catch (e) {
      // Malformed items JSON -> surface, don't crash; keep whatever lines
      // were already on the sheet rather than silently wiping them.
      this.revisionError = `Could not load revision ${rev.revisionNumber}: ${e instanceof Error ? e.message : String(e)}`
    }
  }

  selectRevision(rev: CostingRevisionRow): void {
    this.applyRevision(rev)
  }

  private resolvePreparedByOrBlock(): string {
    const preparedBy = this.header.preparedBy.trim()
    if (!preparedBy) {
      // Refuse-to-fake-identity: never fall back to a "System" ghost.
      this.saveError = 'Select who prepared this costing before saving.'
      return ''
    }
    return preparedBy
  }

  private buildExportPayload(): CostingExportPayload {
    return {
      header: { ...this.header },
      lines: this.validLines.map((l) => ({ ...l })),
      body: this.quotationBody,
      termsAndConditions: this.termsAndConditions,
      subtotal: this.subtotal,
      discount: this.effectiveDiscount,
      netAmount: this.netAmount,
      vat: this.vat,
      grandTotal: this.grandTotal,
      totalCost: this.totalCost,
      profit: this.profit,
      profitPercent: this.profitPercent,
      hiddenCharges: this.effectiveHiddenCharges,
      opportunityId: this.isRFQOpportunity && this.selectedOpportunity ? Number(this.selectedOpportunity.id) : 0,
      opportunityRecordId: this.selectedOpportunity?.source === 'pipeline' ? this.selectedOpportunity.id : '',
      projectName: this.selectedOpportunity?.project ?? '',
    }
  }

  /** Assemble the FLAT main.CostingExportData the real SaveCostingAsOffer
   * binding takes. Unlike buildExportPayload (the nested lab payload kept for
   * the costing-history JSON + PDF/Excel exports), this carries per-line
   * COMPUTED values from calcLine — the sacred waterfall's outputs — so the
   * server can build the offer's line items and persist the detailed cost
   * breakdown. Header + sheet totals come straight from state/`totals`; the
   * effective (defaulted, clamped) percents are the SAME expressions calcLine
   * uses internally, so each exported percent stays consistent with its
   * computed BHD amount. offerId is '' ⇒ the server CREATE path (see bridge). */
  private buildCostingExportData(): CostingExportData {
    const h = this.header
    const t = this.totals
    const lineItems: CostingExportLineItem[] = this.validLines.map((l, i) => costingExportLine(l, i))
    return {
      division: h.division,
      source: this.selectedOpportunity?.source ?? '',
      offerId: '',
      offerNumber: '',
      date: h.date,
      preparedBy: h.preparedBy,
      customerId: h.customerId,
      customerName: h.customerName,
      contactPerson: h.contactPerson,
      rfqReference: h.rfqReference,
      folderNumber: h.folderNumber,
      costingId: h.costingId,
      subject: h.subject,
      estDelivery: h.estDelivery,
      deliveryTerms: h.deliveryTerms,
      paymentTerms: h.paymentTerms,
      orderType: h.orderType,
      countryOfOrigin: h.countryOfOrigin,
      cocCoo: h.cocCoo,
      testCertificate: h.testCertificate,
      installation: h.installation,
      commissioning: h.commissioning,
      testing: h.testing,
      quoteType: h.quoteType,
      vatRate: t.effectiveVatRate,
      hiddenCharges: t.effectiveHiddenCharges,
      placeOfSupply: h.placeOfSupply,
      taxCategory: h.taxCategory,
      customerTRN: h.customerTRN,
      body: this.quotationBody,
      lineItems,
      subtotal: t.subtotal,
      discount: t.effectiveDiscount,
      netAmount: t.netAmount,
      vat: t.vat,
      grandTotal: t.grandTotal,
      totalCost: t.totalCost,
      profit: t.profit,
      profitPercent: t.profitPercent,
      opportunityId: this.isRFQOpportunity && this.selectedOpportunity ? Number(this.selectedOpportunity.id) : 0,
      opportunityRecordId: this.selectedOpportunity?.source === 'pipeline' ? this.selectedOpportunity.id : '',
      projectName: this.selectedOpportunity?.project ?? '',
      termsAndConditions: this.termsAndConditions,
    }
  }

  /** "Save Costing" — create-or-clone-as-revision, same logic the old
   * screen's "+ New Revision" / standalone Save Costing button both used. */
  async saveCosting(): Promise<void> {
    if (this.savingCosting || !this.isRFQOpportunity || !this.selectedOpportunity) return
    const preparedBy = this.resolvePreparedByOrBlock()
    if (!preparedBy) return

    this.savingCosting = true
    this.saveError = null
    try {
      const items = JSON.stringify(this.buildExportPayload())
      const rfqId = Number(this.selectedOpportunity.id)
      if (this.currentCostingId) {
        await cloneCostingAsNewRevision(this.currentCostingId, preparedBy)
      } else {
        const created = await createCostingSheet(rfqId, items, preparedBy)
        this.currentCostingId = created.id
      }
      await this.loadRevisionsForRFQ(rfqId)
    } catch (e) {
      this.saveError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingCosting = false
    }
  }

  async setActiveRevision(rev: CostingRevisionRow): Promise<void> {
    if (!this.selectedOpportunity) return
    try {
      await setActiveCostingRevision(rev.id)
      await this.loadRevisionsForRFQ(Number(this.selectedOpportunity.id))
    } catch (e) {
      this.revisionError = e instanceof Error ? e.message : String(e)
    }
  }

  /** HOT-ZONE: creates/overwrites an Offer. Gated behind an explicit confirm
   * when the current revision already produced one (linkedOfferNumber set) —
   * mirrors the old screen's sourceOfferId overwrite guard. */
  requestSaveAsOffer(): void {
    if (this.validLines.length === 0) {
      this.saveError = 'Add at least one line item with equipment or price before saving.'
      return
    }
    if (!this.resolvePreparedByOrBlock()) return
    this.saveError = null
    if (this.linkedOfferNumber) {
      this.confirmOverwriteOpen = true
      return
    }
    void this.confirmSaveAsOffer()
  }

  cancelSaveAsOffer(): void {
    this.confirmOverwriteOpen = false
  }

  async confirmSaveAsOffer(): Promise<void> {
    this.confirmOverwriteOpen = false
    if (this.savingOffer) return
    const preparedBy = this.resolvePreparedByOrBlock()
    if (!preparedBy) return

    this.savingOffer = true
    this.saveError = null
    try {
      const payload = this.buildExportPayload()
      const exportData = this.buildCostingExportData()

      // Non-blocking costing-history save (mirrors the old screen's
      // handleSaveAsOffer): refresh the underlying costing record so its
      // items match what's being offered. A failure here never stops the
      // offer save itself, but is surfaced rather than only console-logged.
      if (this.isRFQOpportunity && this.selectedOpportunity) {
        const rfqId = Number(this.selectedOpportunity.id)
        try {
          const items = JSON.stringify(payload)
          if (this.currentCostingId) {
            // Assemble the full CostingSheetData refresh (owner standing default,
            // R1 technique): the new items JSON + the VM's own authoritative
            // totals, which are the values summarisePersistedCosting derives.
            const t = this.totals
            await updateCostingSheet(this.currentCostingId, {
              items,
              subtotal: t.totalCost,
              finalPrice: t.grandTotal,
              totalMarkup: t.profit,
              marginPercent: t.profitPercent,
              customerName: this.header.customerName,
              rfqId,
            })
          } else {
            const created = await createCostingSheet(rfqId, items, preparedBy)
            this.currentCostingId = created.id
          }
        } catch {
          this.saveError = 'Offer will be saved, but the costing history could not be saved.'
        }
      }

      const offer = await saveCostingAsOffer(exportData)
      this.lastSavedOfferNumber = offer.offerNumber
      this.linkedOfferNumber = offer.offerNumber
      this.recentSheets = await fetchRecentCostingSheets(8).catch(() => this.recentSheets)
    } catch (e) {
      this.saveError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingOffer = false
    }
  }

  async exportPDF(): Promise<void> {
    if (this.validLines.length === 0) {
      this.exportError = 'Add at least one line item with equipment or price before exporting.'
      return
    }
    this.exporting = true
    this.exportError = null
    try {
      const path = await exportCostingToPDF(this.buildCostingExportData())
      if (path) await openExportedFile(path)
    } catch (e) {
      this.exportError = e instanceof Error ? e.message : String(e)
    } finally {
      this.exporting = false
    }
  }

  async exportExcel(): Promise<void> {
    if (this.validLines.length === 0) {
      this.exportError = 'Add at least one line item with equipment or price before exporting.'
      return
    }
    this.exporting = true
    this.exportError = null
    try {
      const path = await exportCostingToExcel(this.buildCostingExportData())
      if (path) await openExportedFile(path)
    } catch (e) {
      this.exportError = e instanceof Error ? e.message : String(e)
    } finally {
      this.exporting = false
    }
  }

  addLine(): void {
    if (this.lines.length >= MAX_LINE_ITEMS) return
    this.lines.push(this.blankLine())
  }

  removeLine(index: number): void {
    if (this.lines.length <= 1) return
    this.lines.splice(index, 1)
  }
}
