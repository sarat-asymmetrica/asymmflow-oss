/* DeliveryNotes bridge module — self-contained: types + mock + real + switch.
 * Real fetch: GetDeliveryNotes() on `App`, verified against
 * wailsjs/go/main/App.d.ts (no params, no pagination — flat load). */
import { pick } from './runtime'
import { goDate, str } from './map'
import { ConfirmDeliveryNote, DeleteDeliveryNote, DispatchDeliveryNote, GetDeliveryNotes } from '$wails/go/main/App'

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
// The REAL backend flow is a 2-step machine: Prepared→Dispatched (DispatchDeliveryNote,
// driver/vehicle capture) → Delivered (ConfirmDeliveryNote, POD signature). There is no
// `InTransit` binding — it was an old-screen intermediate with no server transition, so
// it's terminal here (a mock-only state kept in the adversarial fixtures, actionless).
// `Signed`/`Cancelled` exist in the backend enum but have no UI path — not modeled.
const TRANSITIONS: Record<string, string[]> = {
  Prepared: ['Dispatched'],
  Dispatched: ['Delivered'],
  InTransit: [],
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

/** Dispatch (Prepared → Dispatched): capture driver + vehicle (R5). */
async function mockDispatch(id: string, driverName: string, vehicleNumber: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) {
    row.status = 'Dispatched'
    row.driverName = driverName
    row.vehicleNumber = vehicleNumber
  }
  await sleep(120)
}

/** Confirm Delivery (Dispatched → Delivered): capture POD signatory (R5). */
async function mockConfirm(id: string, _signedBy: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) row.status = 'Delivered'
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

async function realDispatch(id: string, driverName: string, vehicleNumber: string): Promise<void> {
  // DispatchDeliveryNote(id, driverName, vehicleNumber) — Prepared → Dispatched.
  // Server rejects any non-Prepared status and marks the DN's serials Shipped.
  await DispatchDeliveryNote(id, driverName, vehicleNumber)
}

async function realConfirm(id: string, signedBy: string): Promise<void> {
  // ConfirmDeliveryNote(id, signedBy) — Dispatched → Delivered (records POD
  // signatory + signed_at). Returns a non-fatal downstream warning string
  // (order-progression); the capture form only needs success/failure, so the
  // warning is dropped here (surfaced in logs server-side).
  await ConfirmDeliveryNote(id, signedBy)
}
async function realDelete(id: string): Promise<void> {
  // DeleteDeliveryNote(id) — App binding (verified App.d.ts:381). The Go service
  // runs guardDeleteOrRequest("delivery_notes:delete") server-side, which may
  // route through the delete-approval queue rather than a hard delete; the
  // descriptor already restricts this action to Prepared (pre-dispatch) rows.
  await DeleteDeliveryNote(id)
}

/* ---- public switched API (descriptors import THESE) ---- */
export const fetchDeliveryNotes = (): Promise<DeliveryNoteRow[]> => pick(realFetchAll, mockFetchAll)()
export const dispatchDeliveryNote = (id: string, driverName: string, vehicleNumber: string): Promise<void> =>
  pick(realDispatch, mockDispatch)(id, driverName, vehicleNumber)
export const confirmDeliveryNote = (id: string, signedBy: string): Promise<void> =>
  pick(realConfirm, mockConfirm)(id, signedBy)
export const deleteDeliveryNote = (id: string): Promise<void> => pick(realDelete, mockDelete)(id)

export const deliveryNoteTransitions = TRANSITIONS
