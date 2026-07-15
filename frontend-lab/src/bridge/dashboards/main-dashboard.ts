/* Main dashboard bridge — the landing-page payload (mirrors GetDashboardStats
 * + GetDashboardPipelineByStageYTD + GetDashboardARAgingReportYTD + tasks,
 * composed into one object). Mock is deterministic + synthetic; real wiring
 * (several bindings, per-widget independent load) lands at K5. */
import { pick } from '../runtime'
import { num, str } from '../map'
import type { Tone } from '../../kernel/tones'
import type { NavIntent } from '../../kernel/hub'
import {
  GetDashboardStats,
  GetDashboardPipelineByStageYTD,
  GetDashboardARAgingReportYTD,
} from '$wails/go/main/App'

export interface MainDashboardData {
  year: number
  totalRevenue: number
  monthGrowth: number
  cashBalance: number
  cashNote: boolean
  outstandingAr: number
  arDaysOverdue: number
  pendingInvoices: number
  pipelineValue: number
  pipeline: { stage: string; count: number; value: number; tone: Tone; nav?: NavIntent }[]
  aging: { key: string; label: string; value: number; tone: Tone; nav?: NavIntent }[]
  focus: { label: string; detail: string; tone: Tone; nav?: NavIntent }[]
  alerts: { label: string; text: string; tone: Tone }[]
  tasks: { title: string; subtitle: string; timestamp: string }[]
}

const overdue: NavIntent = { key: 'invoices', query: { filters: { status: 'Overdue' } } }

function mockData(): MainDashboardData {
  return {
    year: 2026,
    totalRevenue: 4_182_450.75,
    monthGrowth: 12.4,
    cashBalance: 738_920.12,
    cashNote: true,
    outstandingAr: 512_330.44,
    arDaysOverdue: 18,
    pendingInvoices: 23,
    pipelineValue: 1_845_000.0,
    pipeline: [
      { stage: 'Quoted', count: 41, value: 820_000, tone: 'info', nav: { key: 'offers' } },
      { stage: 'Won', count: 18, value: 540_000, tone: 'success', nav: { key: 'offers' } },
      { stage: 'RFQ', count: 33, value: 310_000, tone: 'neutral', nav: { key: 'rfqs' } },
      { stage: 'Lost', count: 12, value: 175_000, tone: 'danger', nav: { key: 'offers' } },
    ],
    aging: [
      { key: 'current', label: 'Current', value: 288_000, tone: 'success', nav: overdue },
      { key: '30', label: '1–30d', value: 121_000, tone: 'info', nav: overdue },
      { key: '60', label: '31–60d', value: 62_500, tone: 'warning', nav: overdue },
      { key: '90', label: '61–90d', value: 28_900, tone: 'warning', nav: overdue },
      { key: '120', label: '90d+', value: 11_930, tone: 'danger', nav: overdue },
    ],
    focus: [
      { label: 'Collect overdue receivables', detail: '23 invoices past due · BHD 103,330', tone: 'danger', nav: overdue },
      { label: 'Follow up expiring offers', detail: '7 offers expire within 14 days', tone: 'warning', nav: { key: 'offers' } },
      { label: 'Approve pending purchase orders', detail: '4 POs awaiting approval', tone: 'info', nav: { key: 'purchase-orders' } },
    ],
    alerts: [
      { label: 'Cash', text: 'Two bank statements not yet reconciled this month.', tone: 'warning' },
      { label: 'Credit', text: '1 customer over credit limit and unblocked.', tone: 'danger' },
      { label: 'Stock', text: '6 order lines short against available inventory.', tone: 'warning' },
    ],
    tasks: [
      { title: 'Prepare Q2 board pack', subtitle: 'Finance · due Thursday', timestamp: '2d' },
      { title: 'Confirm delivery — Gulf Fabrication', subtitle: 'Operations', timestamp: '4h' },
      { title: 'Review supplier price list', subtitle: 'Procurement', timestamp: '1d' },
    ],
  }
}

async function mockFetch(): Promise<MainDashboardData> {
  await new Promise((r) => setTimeout(r, 250))
  return mockData()
}

/* Pipeline stage → tone + drill-down, mirroring the mock's intent (L2: one
 * mapping, not scattered). Unknown stages fall back to neutral with no nav. */
const PIPELINE_TONE: Record<string, Tone> = {
  RFQ: 'neutral',
  Quoted: 'info',
  Quotation: 'info',
  Won: 'success',
  Lost: 'danger',
}
const PIPELINE_NAV: Record<string, NavIntent> = {
  RFQ: { key: 'rfqs' },
  Quoted: { key: 'offers' },
  Quotation: { key: 'offers' },
  Won: { key: 'offers' },
  Lost: { key: 'offers' },
}

async function realFetch(): Promise<MainDashboardData> {
  // 3-binding composition (per-widget independent load stays a deferred ENGINE;
  // K3 note). Promise.all fails-together — acceptable for the landing payload.
  const [stats, pipelineRaw, aging] = await Promise.all([
    GetDashboardStats(),
    GetDashboardPipelineByStageYTD(),
    GetDashboardARAgingReportYTD(),
  ])
  const s = stats as unknown as Record<string, unknown>
  const a = aging as unknown as Record<string, unknown>

  const pipeline: MainDashboardData['pipeline'] = (pipelineRaw ?? []).map((row) => {
    const p = row as unknown as Record<string, unknown>
    const stage = str(p.stage)
    const nav = PIPELINE_NAV[stage]
    // Omit `nav` entirely when unknown (exactOptionalPropertyTypes), never set undefined.
    return { stage, count: num(p.count), value: num(p.value), tone: PIPELINE_TONE[stage] ?? 'neutral', ...(nav ? { nav } : {}) }
  })

  const agingBuckets: MainDashboardData['aging'] = [
    { key: 'current', label: 'Current', value: num(a.current), tone: 'success', nav: overdue },
    { key: '30', label: '1–30d', value: num(a.days_30), tone: 'info', nav: overdue },
    { key: '60', label: '31–60d', value: num(a.days_60), tone: 'warning', nav: overdue },
    { key: '90', label: '61–90d', value: num(a.days_90), tone: 'warning', nav: overdue },
    { key: '120', label: '90d+', value: num(a.days_120_plus), tone: 'danger', nav: overdue },
  ]

  return {
    year: num(s.activity_year) || new Date().getFullYear(),
    totalRevenue: num(s.total_revenue),
    monthGrowth: num(s.month_growth),
    cashBalance: num(s.cash_balance_bhd),
    cashNote: !!str(s.cash_position_note),
    outstandingAr: num(s.outstanding_ar),
    arDaysOverdue: num(s.ar_days_overdue),
    pendingInvoices: num(s.pending_invoices),
    pipelineValue: num(s.pipeline_value_bhd),
    pipeline,
    aging: agingBuckets,
    // No backing binding in the dashboard roster (GetDashboardStats + pipeline +
    // AR-aging only) — honest blank, not faked. The Hub hides empty widgets (K3).
    focus: [],
    alerts: [],
    tasks: [],
  }
}

export const fetchMainDashboard = (): Promise<MainDashboardData> => pick(realFetch, mockFetch)()
