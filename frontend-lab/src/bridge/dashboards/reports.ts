/* Reports bridge — mirrors GetReportData(category, "month"). The real Go
 * struct (`main.ReportData`, wailsjs/go/models.ts) is ONE union type shared
 * by all 5 categories, with only the active category's fields populated
 * server-side — the old screen branches its whole UI on `activeCategory` to
 * read the right subset. This bridge's `ReportsData` mirrors that struct
 * field-for-field (camelCased); the mock follows the same "only this
 * category's fields are non-empty" convention as the real backend. Real
 * fetch is WIRED — `GetReportData` is a real, working binding (recon K4:
 * "only CSV export implemented... PDF/Excel are coming-soon stubs"), unlike
 * `ExportReport`, which the Hub archetype has no screen-action concept for
 * (see Reports.parity.md). */
import { pick } from '../runtime'
import { num, str } from '../map'
import { GetReportData } from '$wails/go/main/InfraService'

export type ReportCategory = 'sales' | 'customers' | 'operations' | 'inventory' | 'financial'

export interface ReportsData {
  category: ReportCategory
  // sales
  winRate: number
  conversionRate: number
  avgDealSize: number
  pipeline: { stage: string; count: number; value: number }[]
  // customers
  avgPaymentDays: number
  collectionEfficiency: number
  gradeDistribution: { grade: string; count: number; percentage: number }[]
  typeDistribution: { type: string; label: string; count: number }[]
  // operations
  avgLeadTime: number
  onTimeDelivery: number
  pendingShipments: number
  ordersByStage: { stage: string; count: number }[]
  // inventory
  totalItems: number
  totalValue: number
  lowStockAlerts: number
  movements: { type: string; count: number; value: number }[]
  // financial
  receivablesOutstanding: number
  payablesOutstanding: number
  avgMonthlyRevenue: number
  collectionTarget: number
  collected: number
  overdue: { days: string; amount: number }[]
}

const EMPTY: Omit<ReportsData, 'category'> = {
  winRate: 0,
  conversionRate: 0,
  avgDealSize: 0,
  pipeline: [],
  avgPaymentDays: 0,
  collectionEfficiency: 0,
  gradeDistribution: [],
  typeDistribution: [],
  avgLeadTime: 0,
  onTimeDelivery: 0,
  pendingShipments: 0,
  ordersByStage: [],
  totalItems: 0,
  totalValue: 0,
  lowStockAlerts: 0,
  movements: [],
  receivablesOutstanding: 0,
  payablesOutstanding: 0,
  avgMonthlyRevenue: 0,
  collectionTarget: 0,
  collected: 0,
  overdue: [],
}

/* ---- mock: hand-authored per category (small fixed-shape dashboard
 * payload, same style as finance-overview.ts's per-year map — not a
 * bulk-generated row list). A couple of adversarial touches: a zero-value
 * pipeline stage (max-of-zero-safe bar fill) and an empty stage label. ---- */
const CATEGORY_DATA: Record<ReportCategory, ReportsData> = {
  sales: {
    ...EMPTY,
    category: 'sales',
    winRate: 0.62,
    conversionRate: 0.34,
    avgDealSize: 18450.5,
    pipeline: [
      { stage: 'Qualification', count: 24, value: 142_000 },
      { stage: 'Proposal', count: 14, value: 268_500 },
      { stage: 'Negotiation', count: 8, value: 196_400 },
      { stage: '', count: 0, value: 0 }, // adversarial: unlabeled, zero-value stage
      { stage: 'Closing', count: 3, value: 84_200 },
    ],
  },
  customers: {
    ...EMPTY,
    category: 'customers',
    avgPaymentDays: 38,
    collectionEfficiency: 0.91,
    gradeDistribution: [
      { grade: 'A', count: 42, percentage: 35 },
      { grade: 'B', count: 51, percentage: 42.5 },
      { grade: 'C', count: 19, percentage: 15.8 },
      { grade: 'D', count: 8, percentage: 6.7 },
    ],
    typeDistribution: [
      { type: 'oil_gas', label: 'Oil & Gas', count: 38 },
      { type: 'construction', label: 'Construction', count: 27 },
      { type: 'manufacturing', label: 'Manufacturing', count: 22 },
      { type: 'other', label: 'Other', count: 33 },
    ],
  },
  operations: {
    ...EMPTY,
    category: 'operations',
    avgLeadTime: 12,
    onTimeDelivery: 0.87,
    pendingShipments: 9,
    ordersByStage: [
      { stage: 'Processing', count: 11 },
      { stage: 'Picking', count: 6 },
      { stage: 'Shipped', count: 14 },
      { stage: 'Delivered', count: 58 },
    ],
  },
  inventory: {
    ...EMPTY,
    category: 'inventory',
    totalItems: 1842,
    totalValue: 612_480.25,
    lowStockAlerts: 6,
    movements: [
      { type: 'Inbound', count: 34, value: 128_900 },
      { type: 'Outbound', count: 41, value: 96_700 },
      { type: 'Adjustment', count: 3, value: 2_100 },
    ],
  },
  financial: {
    ...EMPTY,
    category: 'financial',
    receivablesOutstanding: 512_330,
    payablesOutstanding: 268_940,
    avgMonthlyRevenue: 348_500,
    collectionTarget: 400_000,
    collected: 316_200,
    overdue: [
      { days: '30–60', amount: 48_200 },
      { days: '60–90', amount: 21_500 },
      { days: '90+', amount: 15_800 },
    ],
  },
}

async function mockFetch(period?: string): Promise<ReportsData> {
  await new Promise((r) => setTimeout(r, 250))
  const category = (period as ReportCategory) || 'sales'
  return CATEGORY_DATA[category] ?? CATEGORY_DATA.sales
}

/* ---- real: fetch WIRED (GetReportData is a real, working binding) ---- */
function arr<T>(v: unknown, map: (x: Record<string, unknown>) => T): T[] {
  return Array.isArray(v) ? v.map((x) => map(x as Record<string, unknown>)) : []
}

function mapReportsData(category: ReportCategory, r: Record<string, unknown>): ReportsData {
  return {
    category,
    winRate: num(r.win_rate),
    conversionRate: num(r.conversion_rate),
    avgDealSize: num(r.avg_deal_size),
    pipeline: arr(r.pipeline, (p) => ({ stage: str(p.stage), count: num(p.count), value: num(p.value) })),
    avgPaymentDays: num(r.avg_payment_days),
    collectionEfficiency: num(r.collection_efficiency),
    gradeDistribution: arr(r.grade_distribution, (g) => ({ grade: str(g.grade), count: num(g.count), percentage: num(g.percentage) })),
    typeDistribution: arr(r.type_distribution, (t) => ({ type: str(t.type), label: str(t.label), count: num(t.count) })),
    avgLeadTime: num(r.avg_lead_time),
    onTimeDelivery: num(r.on_time_delivery),
    pendingShipments: num(r.pending_shipments),
    ordersByStage: arr(r.orders_by_stage, (s) => ({ stage: str(s.stage), count: num(s.count) })),
    totalItems: num(r.total_items),
    totalValue: num(r.total_value),
    lowStockAlerts: num(r.low_stock_alerts),
    movements: arr(r.movements, (m) => ({ type: str(m.type), count: num(m.count), value: num(m.value) })),
    receivablesOutstanding: num(r.receivables_outstanding),
    payablesOutstanding: num(r.payables_outstanding),
    avgMonthlyRevenue: num(r.avg_monthly_revenue),
    collectionTarget: num(r.collection_target),
    collected: num(r.collected),
    overdue: arr(r.overdue, (o) => ({ days: str(o.days), amount: num(o.amount) })),
  }
}

async function realFetch(period?: string): Promise<ReportsData> {
  const category = (period as ReportCategory) || 'sales'
  // GetReportData only accepts a fixed period preset (week/month/quarter/
  // year) — 'month' mirrors the old screen's mount call exactly (recon K4).
  const raw = await GetReportData(category, 'month')
  return mapReportsData(category, raw as unknown as Record<string, unknown>)
}

export const fetchReports = (period?: string): Promise<ReportsData> => pick(realFetch, mockFetch)(period)
