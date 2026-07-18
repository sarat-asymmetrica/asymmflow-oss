/* Serial Trace bridge module — self-contained: types + mock + real +
 * switch. Read-only: search a serial number and trace its lifecycle
 * PO -> GRN -> Delivery Note -> Invoice -> Customer, with warranty-date
 * coloring. No mutations exist on the old screen or here. Per the K4
 * brief, BOTH real calls are INTEG-gapped (not just the usual
 * mutations-only split) — the bespoke-screen track keeps read wiring at
 * K5 alongside the ledger-archetype screens rather than special-casing
 * one bespoke view early. */
import { pick } from './runtime'
import { goDate, str } from './map'
import { SearchSerials, GetRecentlyDeliveredSerials } from '$wails/go/main/App'

export interface SerialTraceRow {
  id: string
  serialNo: string
  productCode: string
  status: string
  poNumber: string
  grnNumber: string
  dnNumber: string
  invoiceNumber: string
  customerName: string
  warrantyStartDate: string
  warrantyEndDate: string
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

const STATUSES = ['Available', 'Reserved', 'Shipped', 'Delivered', 'Returned']
const PRODUCT_CODES = ['PT-100-DIN', 'FV-2200-SS', 'RTD-PT1000', 'XMTR-4-20MA', 'GV-ANSI150', 'CAL-BENCH-A']
const CUSTOMER_NAMES = [
  'Gulf Fabrication W.L.L.',
  'Manama Process Systems',
  'Al Dana Engineering Co.',
  'Interntional Establishment for Industrial & Petrochemical Instrumentation Services and General Trading (formerly Gulf Technical Calibration & Measurement Systems Company) W.L.L.',
  'المؤسسة الدولية لخدمات الأجهزة الصناعية والبتروكيماوية والتجارة العامة ذ.م.م',
  'Sitra Contracting',
  '',
]

let cache: SerialTraceRow[] | null = null

function generate(): SerialTraceRow[] {
  const rand = lcg(20260714)
  const rows: SerialTraceRow[] = []
  const n = 260
  for (let i = 1; i <= n; i++) {
    const status = STATUSES[Math.floor(rand() * STATUSES.length)]!
    const delivered = status === 'Delivered' || status === 'Returned'
    const monthIdx = Math.floor(rand() * 24)
    const year = 2024 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const receivedDate = `${year}-${pad(month, 2)}-${pad(day, 2)}`

    // Warranty window: months from received date. Deterministic mix of
    // already-expired and still-valid so the tone split is exercised.
    const warrantyMonths = i % 47 === 0 ? 0 : [12, 24, 36][i % 3]!
    const warrantyStart = delivered ? receivedDate : ''
    let warrantyEnd = ''
    if (delivered && warrantyMonths > 0) {
      const endMs = new Date(`${receivedDate}T00:00:00`).getTime() + warrantyMonths * 30 * 86_400_000
      warrantyEnd = new Date(endMs).toISOString().slice(0, 10)
    }

    const serialNo =
      i % 71 === 0
        ? `SN-${pad(i, 40)}` // unbroken 40-digit monster token
        : `SN-${year}-${pad(i, 5)}`

    const customerName = delivered ? CUSTOMER_NAMES[i % CUSTOMER_NAMES.length]! : ''

    rows.push({
      id: `sn-${i}`,
      serialNo,
      productCode: i % 83 === 0 ? '' : PRODUCT_CODES[i % PRODUCT_CODES.length]!,
      status: i % 97 === 0 ? 'UNKNOWN_STATE' : status,
      poNumber: `PO-${year}-${pad(i, 4)}`,
      grnNumber: i % 19 === 0 ? '' : `GRN-${year}-${pad(i, 4)}`,
      dnNumber: !delivered && rand() < 0.4 ? '' : `DN-${year}-${pad(i, 4)}`,
      invoiceNumber: !delivered ? '' : `INV-${year}-${pad(i, 4)}`,
      customerName,
      warrantyStartDate: warrantyStart,
      warrantyEndDate: warrantyEnd,
    })
  }
  return rows
}

async function mockSearch(query: string, limit: number): Promise<SerialTraceRow[]> {
  cache ??= generate()
  await sleep(220)
  const needle = query.trim().toLowerCase()
  if (!needle) return []
  const matches = cache.filter((r) =>
    [r.serialNo, r.productCode, r.poNumber, r.grnNumber, r.dnNumber, r.invoiceNumber, r.customerName]
      .join(' ')
      .toLowerCase()
      .includes(needle),
  )
  return matches.slice(0, limit)
}

async function mockRecentlyDelivered(n: number): Promise<SerialTraceRow[]> {
  cache ??= generate()
  await sleep(180)
  const delivered = cache.filter((r) => r.status === 'Delivered' || r.status === 'Returned')
  // Deterministic dataset, so "recent" = highest index generated (later i
  // biases toward later synthetic dates via the same monthIdx spread).
  return delivered.slice(-n).reverse()
}

/* ---- real: BOTH SearchSerials and GetRecentlyDeliveredSerials are real,
 * working, single-call `wailsjs/go/main/App` bindings (crm.SerialNumber:
 * serial_no/product_code/status/po_number/grn_number/dn_number/
 * invoice_number/customer_name/warranty_start_date/warranty_end_date) — no
 * account-merge or aggregation complexity like cheque-register.ts's
 * realFetchAll needed. They stay INTEG-gapped anyway per the K4 brief for
 * this screen, so the bindings aren't imported here at all; wiring them at
 * K5 is a straight `pick(realSearch, mockSearch)` swap plus the field
 * mapping documented above, same shape as data-quality.ts's realFetch. ---- */
function mapSerial(raw: unknown): SerialTraceRow {
  const r = raw as Record<string, unknown>
  return {
    id: str(r.id),
    serialNo: str(r.serial_no),
    productCode: str(r.product_code),
    status: str(r.status),
    poNumber: str(r.po_number),
    grnNumber: str(r.grn_number),
    dnNumber: str(r.dn_number),
    invoiceNumber: str(r.invoice_number),
    customerName: str(r.customer_name),
    warrantyStartDate: goDate(r.warranty_start_date),
    warrantyEndDate: goDate(r.warranty_end_date),
  }
}

async function realSearch(query: string, limit: number): Promise<SerialTraceRow[]> {
  const rows = await SearchSerials(query, limit)
  return (rows ?? []).map(mapSerial)
}

async function realRecentlyDelivered(n: number): Promise<SerialTraceRow[]> {
  const rows = await GetRecentlyDeliveredSerials(n)
  return (rows ?? []).map(mapSerial)
}

/* ---- public switched API (viewmodel imports THESE) ---- */
export const searchSerials = (query: string, limit: number): Promise<SerialTraceRow[]> =>
  pick(realSearch, mockSearch)(query, limit)
export const recentlyDeliveredSerials = (n: number): Promise<SerialTraceRow[]> =>
  pick(realRecentlyDelivered, mockRecentlyDelivered)(n)
