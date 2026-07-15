/* InventoryFulfillment bridge module — self-contained: types + mock + real.
 * Read-only report (recon-K2): no create/edit/delete, no status transitions
 * — this module exposes fetch ONLY. Real binding: `GetInventoryPendingFulfillmentReport(limit)`
 * on App (verified against frontend/wailsjs/go/main/App.d.ts:885), called
 * unpaged like the old screen (`GetInventoryPendingFulfillmentReport(500)`). */
import { pick } from './runtime'
import { num, str } from './map'
import { GetInventoryPendingFulfillmentReport } from '$wails/go/main/App'

export interface InventoryFulfillmentRow {
  orderId: string
  orderNumber: string
  customerName: string
  productCode: string
  description: string
  orderedQuantity: number
  deliveredQuantity: number
  invoicedQuantity: number
  pendingQuantity: number
  availableQuantity: number
  shortageQuantity: number
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
const PRODUCTS = ['PRC-2200', 'FLW-4410', 'TMP-0091', 'VLV-7735', 'ACT-1180', 'GAU-3302']
const DESCRIPTIONS = [
  'Pressure transmitter, 4-20mA, 0-100 bar range, 316SS wetted parts',
  'Flow meter, electromagnetic, DN80, IP68 rated',
  'Temperature transmitter, RTD input, dual compartment housing',
  'Control valve, globe pattern, pneumatic actuator, fail-closed',
  'Rotary actuator, spring-return, ATEX certified',
  'Pressure gauge, glycerin-filled, 100mm dial, bottom connection',
]
/** Known order-status vocabulary → tone bucket (matches the descriptor's
 * StatusSpec.tones keys exactly). Values outside this list are a deliberate
 * adversary — the engine renders them neutral rather than crashing. */
const STATUSES = ['Delivered', 'Invoiced', 'Closed', 'Complete', 'Pending', 'Processing', 'Open', 'Cancelled', 'Lost']

let cache: InventoryFulfillmentRow[] | null = null

function generate(): InventoryFulfillmentRow[] {
  const rand = lcg(20260714 ^ 0x1f57)
  const rows: InventoryFulfillmentRow[] = []
  const n = 250
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const year = 2025 + Math.floor(rand() * 2)
    const ordered =
      i % 89 === 0 ? 987654321.987 : i % 53 === 0 ? 0.001 : 1 + Math.floor(rand() * 500)
    const delivered = i % 31 === 0 ? 0 : Math.round(ordered * (0.2 + rand() * 0.8) * 100) / 100
    const invoiced = Math.round(delivered * (0.5 + rand() * 0.5) * 100) / 100
    const pending = Math.max(0, Math.round((ordered - delivered) * 100) / 100)
    const available = i % 19 === 0 ? 0 : Math.floor(rand() * 300)
    const shortage = Math.max(0, Math.round((pending - available) * 100) / 100)
    const status = i % 97 === 0 ? 'ON_HOLD_CUSTOMS_INSPECTION' : STATUSES[i % STATUSES.length]!

    rows.push({
      orderId: `ord-${i}`,
      orderNumber: i % 71 === 0 ? `ORD-${year}-${pad(i, 40)}` : `ORD-${year}-${pad(i, 4)}`,
      customerName: i % 67 === 0 ? '' : CUSTOMERS[i % CUSTOMERS.length]!,
      productCode: PRODUCTS[i % PRODUCTS.length]!,
      description: i % 41 === 0 ? '' : DESCRIPTIONS[i % DESCRIPTIONS.length]!,
      orderedQuantity: ordered,
      deliveredQuantity: delivered,
      invoicedQuantity: invoiced,
      pendingQuantity: pending,
      availableQuantity: available,
      shortageQuantity: shortage,
      status,
    })
  }
  return rows
}

async function mockFetchAll(): Promise<InventoryFulfillmentRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

/* ---- real: fetch WIRED (no mutations — this module is read-only) ---- */
function mapRow(r: Record<string, unknown>): InventoryFulfillmentRow {
  return {
    orderId: str(r.order_id),
    orderNumber: str(r.order_number),
    customerName: str(r.customer_name),
    productCode: str(r.product_code),
    description: str(r.description),
    orderedQuantity: num(r.ordered_quantity),
    deliveredQuantity: num(r.delivered_quantity),
    invoicedQuantity: num(r.invoiced_quantity),
    pendingQuantity: num(r.pending_quantity),
    availableQuantity: num(r.available_quantity),
    shortageQuantity: num(r.shortage_quantity),
    status: str(r.status),
  }
}

async function realFetchAll(): Promise<InventoryFulfillmentRow[]> {
  const rows = await GetInventoryPendingFulfillmentReport(500)
  return (rows ?? []).map((x) => mapRow(x as unknown as Record<string, unknown>))
}

/* ---- public switched API (descriptor imports THIS) ---- */
export const fetchInventoryFulfillment = (): Promise<InventoryFulfillmentRow[]> =>
  pick(realFetchAll, mockFetchAll)()
