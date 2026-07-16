/* Purchase Orders bridge module — self-contained: types + mock + real + switch. */
import { pick } from './runtime'
import { goDate, num, str } from './map'
import { GetPurchaseOrders, GetPurchaseOrderByID, ReceiveAgainstPO, UpdatePOStatus } from '$wails/go/main/App'
import type { crm } from '$wails/go/models'

export interface PurchaseOrderRow {
  id: string
  poNumber: string
  orderId: string
  supplierId: string
  supplierName: string
  poDate: string
  expectedDelivery: string
  currency: string
  exchangeRate: number
  subtotalForeign: number
  totalBhd: number
  vatAmount: number
  totalForeign: number
  paymentTerms: string
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

// 7-currency support per the old screen; JPY seasoned in as a monster (0-decimal
// currency, exchange rate far from 1) to exercise the money column's per-row
// `currency` override.
const CURRENCIES = ['BHD', 'USD', 'EUR', 'GBP', 'AED', 'SAR', 'JPY']
const TERMS = ['Net 30', 'Net 60', 'Advance', 'LC 90 days', 'Net 15']

const STATUSES = [
  'Draft',
  'Pending Approval',
  'Approved',
  'Sent',
  'Acknowledged',
  'Partially Received',
  'Received',
  'Closed',
  'Cancelled',
]

let cache: PurchaseOrderRow[] | null = null

function generate(): PurchaseOrderRow[] {
  const rand = lcg(20260714)
  const rows: PurchaseOrderRow[] = []
  const n = 260
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 24)
    const year = 2024 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const poDate = `${year}-${pad(month, 2)}-${pad(day, 2)}`
    const expectedDelivery = `${year}-${pad(month, 2)}-${pad(Math.min(day + 14, 28), 2)}`

    const status = i % 97 === 0 ? 'UNKNOWN_STATE' : STATUSES[i % STATUSES.length]!
    const currency = i % 41 === 0 ? 'JPY' : CURRENCIES[Math.floor(r * CURRENCIES.length)]!
    const exchangeRate = currency === 'BHD' ? 1 : Math.round((0.2 + rand() * 4) * 10000) / 10000
    const subtotalForeign =
      i % 89 === 0 ? 987654321098.765 : i % 53 === 0 ? 0.001 : Math.round(rand() * 400_000) / 100
    const vatRate = 0.1
    const subtotalBhd = subtotalForeign * exchangeRate
    const vatAmount = Math.round(subtotalBhd * vatRate * 1000) / 1000
    const totalForeign = Math.round((subtotalForeign + subtotalForeign * vatRate) * 100) / 100
    const totalBhd = Math.round((subtotalBhd + vatAmount) * 1000) / 1000

    rows.push({
      id: `po-${i}`,
      poNumber: i % 71 === 0 ? `PO-${year}-${pad(i, 40)}` : `PO-${year}-${pad(i, 4)}`,
      orderId: `ord-${((i * 7) % 180) + 1}`,
      supplierId: `sup-${(i % SUPPLIERS.length) + 1}`,
      supplierName: i % 67 === 0 ? '' : SUPPLIERS[i % SUPPLIERS.length]!,
      poDate,
      expectedDelivery,
      currency,
      exchangeRate,
      subtotalForeign,
      totalBhd,
      vatAmount,
      totalForeign,
      paymentTerms: TERMS[i % TERMS.length]!,
      status,
    })
  }
  return rows
}

async function mockFetchAll(): Promise<PurchaseOrderRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockSetStatus(id: string, status: string): Promise<void> {
  cache ??= generate()
  const po = cache.find((p) => p.id === id)
  if (po) po.status = status
  await sleep(150)
}

/* ---- real: fetch WIRED, mutations are INTEG-gapped (honest throw) ---- */
function mapPurchaseOrder(r: Record<string, unknown>): PurchaseOrderRow {
  return {
    id: str(r.id),
    poNumber: str(r.po_number),
    orderId: str(r.order_id),
    supplierId: str(r.supplier_id),
    supplierName: str(r.supplier_name),
    poDate: goDate(r.po_date),
    expectedDelivery: goDate(r.expected_delivery),
    currency: str(r.currency) || 'BHD',
    exchangeRate: num(r.exchange_rate) || 1,
    subtotalForeign: num(r.subtotal_foreign),
    totalBhd: num(r.total_bhd),
    vatAmount: num(r.vat_amount),
    totalForeign: num(r.total_foreign),
    paymentTerms: str(r.payment_terms),
    status: str(r.status) || 'Draft',
  }
}

async function realFetchAll(): Promise<PurchaseOrderRow[]> {
  const rows = await GetPurchaseOrders()
  return (rows ?? []).map((x) => mapPurchaseOrder(x as unknown as Record<string, unknown>))
}

async function realSetStatus(id: string, status: string): Promise<void> {
  // UpdatePOStatus(id, status) — App binding (verified App.d.ts:1821, string,
  // string). The Go service enforces the legal-transition map and the
  // "po:update" permission server-side (purchase_order_service.go:668). The
  // SoD-gated Approve transition (Pending Approval→Approved) is deliberately NOT
  // routed here — the descriptor never offers it as a plain flip; it needs
  // ApprovePurchaseOrder(id, userId), a separate binding. Receiving posts
  // inventory through a ledgered Receive-Items panel, also not this setter.
  // See PurchaseOrders.parity.md.
  await UpdatePOStatus(id, status)
}

/* ---- Receive Items against PO (R5): a second bridge surface — the ledger's
 * flat row doesn't carry line items (GetPurchaseOrders never preloads them),
 * so receiving needs its own by-ID fetch + its own mutation. ---- */

export interface POReceiveLine {
  poItemId: string
  productCode: string
  description: string
  quantityOrdered: number
  quantityAlreadyReceived: number
}

/** Plain shape the VM assembles per accepted line; cast to crm.GRNItem at the
 * wire boundary — the generated class carries codegen-only fields (id,
 * timestamps, grn_id, quantity_accepted, …) that ReceiveAgainstPO computes
 * server-side and this caller never sets (map.ts precedent). */
export interface GRNItemInput {
  po_item_id: string
  quantity_received: number
  quantity_rejected: number
  rejection_reason?: string
}

function mockReceiveLines(poId: string): POReceiveLine[] {
  return [
    {
      poItemId: `${poId}-item-1`,
      productCode: 'VLV-2200-SS',
      description: 'Ball valve, 2" 316SS, flanged ANSI 300#',
      quantityOrdered: 50,
      quantityAlreadyReceived: 0,
    },
    {
      // Already fully received on a prior GRN — must still render (audit
      // context) but the modal's guard refuses any further quantity.
      poItemId: `${poId}-item-2`,
      productCode: 'GSK-STD-004',
      description: 'Spiral-wound gasket set, graphite-filled, per attached BOM',
      quantityOrdered: 20,
      quantityAlreadyReceived: 20,
    },
    {
      // 200-char description monster.
      poItemId: `${poId}-item-3`,
      productCode: 'ACT-9000-XL',
      description:
        'Pneumatic quarter-turn rack-and-pinion actuator, double-acting, spring-return fail-safe configuration, NAMUR-mounted solenoid interface, anodized aluminium body, fitted with position indicator and limit switch box, OEM cert pack attached'.slice(
          0,
          200,
        ),
      quantityOrdered: 5,
      quantityAlreadyReceived: 2,
    },
    {
      // Zero-quantity line — data artifact from a since-corrected PO revision.
      poItemId: `${poId}-item-4`,
      productCode: 'MISC-0000',
      description: 'Zero-quantity line (superseded revision, kept for audit trail)',
      quantityOrdered: 0,
      quantityAlreadyReceived: 0,
    },
  ]
}

async function mockFetchReceiveLines(poId: string): Promise<POReceiveLine[]> {
  await sleep(200)
  return mockReceiveLines(poId)
}

async function mockReceiveAgainstPO(_poId: string, _items: GRNItemInput[]): Promise<void> {
  await sleep(200)
}

function mapReceiveLine(r: Record<string, unknown>): POReceiveLine {
  return {
    poItemId: str(r.id),
    productCode: str(r.product_code),
    description: str(r.description),
    quantityOrdered: num(r.quantity),
    quantityAlreadyReceived: num(r.quantity_received),
  }
}

async function realFetchReceiveLines(poId: string): Promise<POReceiveLine[]> {
  // GetPurchaseOrderByID Preloads .items (PurchaseOrderItem[]) — the flat
  // ledger fetch (GetPurchaseOrders) never carries them.
  const po = await GetPurchaseOrderByID(poId)
  const items = (po?.items ?? []) as unknown as Record<string, unknown>[]
  return items.map(mapReceiveLine)
}

async function realReceiveAgainstPO(poId: string, items: GRNItemInput[]): Promise<void> {
  // ReceiveAgainstPO(poId, []crm.GRNItem) — the server computes
  // quantity_accepted and enforces the per-line over-receive guard
  // (already_received + received <= ordered); ReceivedBy resolves from the
  // session server-side, so no actor arg is threaded here.
  await ReceiveAgainstPO(poId, items as unknown as crm.GRNItem[])
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchPurchaseOrders = (): Promise<PurchaseOrderRow[]> => pick(realFetchAll, mockFetchAll)()
export const setPurchaseOrderStatus = (id: string, status: string): Promise<void> =>
  pick(realSetStatus, mockSetStatus)(id, status)
export const fetchPOReceiveLines = (poId: string): Promise<POReceiveLine[]> =>
  pick(realFetchReceiveLines, mockFetchReceiveLines)(poId)
export const receiveAgainstPO = (poId: string, items: GRNItemInput[]): Promise<void> =>
  pick(realReceiveAgainstPO, mockReceiveAgainstPO)(poId, items)
