/* Orders bridge module — self-contained: types + mock + real + switch.
 * Mirrors the invoices pilot's paged shape (ListOrders(limit, offset) on
 * CRMService — verified against wailsjs/go/main/CRMService.d.ts). */
import { pick } from './runtime'
import { goDate, num, str } from './map'
import { ListOrders } from '$wails/go/main/CRMService'

export interface OrderRow {
  id: string
  orderNumber: string
  customerName: string
  customerPoNumber: string
  totalValueBhd: number
  orderDate: string
  status: string
  division: string
  /** Mocked in K1 — real value needs a second call (GetOrderDeliveryStatusBatch)
   * that a single fetchPage() row doesn't carry. See ORDERS parity #5. */
  deliveryPercent: number
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts pattern) ---- */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number) {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number) => String(n).padStart(w, '0')

const DIVISIONS = ['Acme Instrumentation', 'Beacon Controls']
const STATUSES = ['Confirmed', 'InProgress', 'PartiallyDelivered', 'FullyDelivered', 'Invoiced', 'Cancelled']
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

let cache: OrderRow[] | null = null

function generate(): OrderRow[] {
  const rand = lcg(20260714)
  const rows: OrderRow[] = []
  for (let i = 1; i <= 340; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 24)
    const year = 2024 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)

    // Adversarial seasoning at deterministic positions:
    const status = i % 61 === 0 ? 'UNKNOWN_STATE' : STATUSES[Math.floor(r * STATUSES.length)]!
    const totalValueBhd =
      i % 91 === 0 ? 987654321.123 : i % 47 === 0 ? 0.001 : Math.round(rand() * 500_000) / 100
    const customerPoNumber = i % 37 === 0 ? '' : `PO-${pad(i, 5)}`
    const deliveryPercent =
      status === 'FullyDelivered' || status === 'Invoiced'
        ? 100
        : status === 'Cancelled'
          ? 0
          : Math.floor(rand() * 100)

    rows.push({
      id: `ord-${i}`,
      orderNumber: `ORD-${year}-${pad(i, 4)}`,
      customerName: CUSTOMERS[i % CUSTOMERS.length]!,
      customerPoNumber,
      totalValueBhd,
      orderDate: `${year}-${pad(month, 2)}-${pad(day, 2)}`,
      status,
      division: DIVISIONS[i % DIVISIONS.length]!,
      deliveryPercent,
    })
  }
  return rows
}

async function mockFetchPage(limit: number, offset: number): Promise<OrderRow[]> {
  cache ??= generate()
  await sleep(offset === 0 ? 250 : 120)
  return cache.slice(offset, offset + limit)
}
async function mockFetchAll(): Promise<OrderRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

/** Fulfillment-only status flip (QuickMarkOrderDelivered) — not financial.
 * Create Invoice/Proforma/PO and cascade-delete stay ledgered (K1-A #3/#4). */
async function mockMarkDelivered(id: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) {
    row.status = 'FullyDelivered'
    row.deliveryPercent = 100
  }
  await sleep(120)
}

/* ---- real: fetch WIRED, mutations are INTEG-gapped (honest throw) ---- */
function mapOrder(r: Record<string, unknown>): OrderRow {
  return {
    id: str(r.id),
    orderNumber: str(r.order_number),
    customerName: str(r.customer_name),
    customerPoNumber: str(r.customer_po_number),
    totalValueBhd: num(r.total_value_bhd ?? r.grand_total_bhd),
    orderDate: goDate(r.order_date),
    status: str(r.status),
    division: str(r.division),
    // GetOrderDeliveryStatusBatch is a separate per-page round-trip; not
    // available from a single ListOrders row. See ORDERS parity #5.
    deliveryPercent: 0,
  }
}

async function realFetchPage(limit: number, offset: number): Promise<OrderRow[]> {
  const rows = await ListOrders(limit, offset)
  return (rows ?? []).map((r) => mapOrder(r as unknown as Record<string, unknown>))
}
async function realFetchAll(): Promise<OrderRow[]> {
  return realFetchPage(200, 0)
}

async function realMarkDelivered(_id: string): Promise<void> {
  throw new Error('INTEG gap: QuickMarkOrderDelivered — wires at K5')
}

/* ---- public switched API (descriptors import THESE) ---- */
export const fetchOrdersPage = (l: number, o: number): Promise<OrderRow[]> =>
  pick(realFetchPage, mockFetchPage)(l, o)
export const fetchOrders = (): Promise<OrderRow[]> => pick(realFetchAll, mockFetchAll)()
export const markOrderDelivered = (id: string): Promise<void> =>
  pick(realMarkDelivered, mockMarkDelivered)(id)
