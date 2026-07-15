/* DeliveryNotes bridge module — self-contained: types + mock + real + switch.
 * Real fetch: GetDeliveryNotes() on `App`, verified against
 * wailsjs/go/main/App.d.ts (no params, no pagination — flat load). */
import { pick } from './runtime'
import { goDate, str } from './map'
import { GetDeliveryNotes } from '$wails/go/main/App'

export interface DeliveryNoteRow {
  id: string
  dnNumber: string
  /** Client-enriched in the old screen (joined against a separately-loaded
   * Orders list); mocked directly onto the row here. See parity #3. */
  orderReference: string
  customerName: string
  deliveryDate: string
  deliverySeq: number
  totalDeliveries: number
  transportMethod: string
  driverName: string
  vehicleNumber: string
  status: string
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

const STATUSES = ['Prepared', 'Dispatched', 'InTransit', 'Delivered']
// Old screen's live status transitions (Prepared→Dispatched→InTransit→Delivered).
// `Signed`/`Cancelled` exist in the backend enum but have no UI path — not modeled.
const TRANSITIONS: Record<string, string[]> = {
  Prepared: ['Dispatched'],
  Dispatched: ['InTransit'],
  InTransit: ['Delivered'],
  Delivered: [],
}
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
const TRANSPORT = ['Company Truck', 'Third-Party Courier', 'Customer Pickup', 'Freight Forwarder']
const DRIVERS = ['Ahmed Al-Sayed', 'Ravi Kumar', 'Mohammed Yusuf', '']

let cache: DeliveryNoteRow[] | null = null

function generate(): DeliveryNoteRow[] {
  const rand = lcg(19860412)
  const rows: DeliveryNoteRow[] = []
  for (let i = 1; i <= 260; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 24)
    const year = 2024 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)

    // Adversarial seasoning at deterministic positions:
    const status = i % 67 === 0 ? 'UNKNOWN_STATE' : STATUSES[Math.floor(r * STATUSES.length)]!
    const totalDeliveries = i % 31 === 0 ? 1 : 1 + Math.floor(rand() * 3)
    const deliverySeq = totalDeliveries === 1 ? 1 : 1 + Math.floor(rand() * totalDeliveries)
    const vehicleNumber = i % 43 === 0 ? '' : `BH-${pad(1000 + Math.floor(rand() * 8999), 4)}`

    rows.push({
      id: `dn-${i}`,
      dnNumber: `DN-${year}-${pad(i, 4)}`,
      orderReference: i % 53 === 0 ? 'N/A' : `ORD-${year}-${pad((i * 7) % 340 || 1, 4)}`,
      customerName: CUSTOMERS[i % CUSTOMERS.length]!,
      deliveryDate: `${year}-${pad(month, 2)}-${pad(day, 2)}`,
      deliverySeq,
      totalDeliveries,
      transportMethod: TRANSPORT[i % TRANSPORT.length]!,
      driverName: DRIVERS[i % DRIVERS.length]!,
      vehicleNumber,
      status,
    })
  }
  return rows
}

async function mockFetchAll(): Promise<DeliveryNoteRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

/** Simplified status advance (Prepared→Dispatched→InTransit→Delivered) — a
 * plain confirm, not the real screen's driver/vehicle capture (Dispatch) or
 * POD signature capture (Confirm Delivery). Those stay ledgered (parity #4/#5). */
async function mockAdvanceStatus(id: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) {
    const next = TRANSITIONS[row.status]?.[0]
    if (next) row.status = next
  }
  await sleep(120)
}

/** Delete — old screen allows this from any status; K1 restricts it to
 * `Prepared` (pre-dispatch) as an intentional safety improvement. */
async function mockDelete(id: string): Promise<void> {
  cache ??= generate()
  cache = cache.filter((r) => r.id !== id)
  await sleep(120)
}

/* ---- real: fetch WIRED, mutations are INTEG-gapped (honest throw) ---- */
function mapDeliveryNote(r: Record<string, unknown>): DeliveryNoteRow {
  return {
    id: str(r.id),
    dnNumber: str(r.dn_number),
    // order_reference/customer_name are NOT on the DeliveryNote struct — the
    // old screen client-joins against separately-loaded Orders/Customers
    // lists. Real integration needs that same join (or a backend change).
    // See parity #3.
    orderReference: str(r.order_reference) || 'N/A',
    customerName: str(r.customer_name) || 'Unknown',
    deliveryDate: goDate(r.delivery_date),
    deliverySeq: Number(r.delivery_sequence) || 1,
    totalDeliveries: Number(r.total_deliveries) || 1,
    transportMethod: str(r.transport_method),
    driverName: str(r.driver_name),
    vehicleNumber: str(r.vehicle_number),
    status: str(r.status),
  }
}

async function realFetchAll(): Promise<DeliveryNoteRow[]> {
  const rows = await GetDeliveryNotes()
  return (rows ?? []).map((r) => mapDeliveryNote(r as unknown as Record<string, unknown>))
}

async function realAdvanceStatus(_id: string): Promise<void> {
  throw new Error('INTEG gap: DispatchDeliveryNote / ConfirmDeliveryNote — wires at K5')
}
async function realDelete(_id: string): Promise<void> {
  throw new Error('INTEG gap: DeleteDeliveryNote — wires at K5')
}

/* ---- public switched API (descriptors import THESE) ---- */
export const fetchDeliveryNotes = (): Promise<DeliveryNoteRow[]> => pick(realFetchAll, mockFetchAll)()
export const advanceDeliveryNoteStatus = (id: string): Promise<void> =>
  pick(realAdvanceStatus, mockAdvanceStatus)(id)
export const deleteDeliveryNote = (id: string): Promise<void> => pick(realDelete, mockDelete)(id)

export const deliveryNoteTransitions = TRANSITIONS
