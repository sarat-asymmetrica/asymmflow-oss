/* Finance Overview bridge — mirrors GetFinancialDashboardForYear (the ~35-field
 * P&L/balance-sheet/ratios/AR-aging/YoY struct, recon K3a/K3b). Mock is
 * deterministic + synthetic, varies a couple of lines per fiscal year so the
 * period selector visibly changes the payload. Real wiring (year-scoped fetch
 * + the live-cash overlay from GetCashPosition) lands at K5. */
import { pick } from '../runtime'
import { num, str } from '../map'
import type { Tone } from '../../kernel/tones'
import { GetFinancialDashboardForYear } from '$wails/go/main/App'

export interface FinanceOverviewData {
  year: number
  as_of_date: string

  // P&L
  revenue: number
  revenue_yoy: number
  gross_profit: number
  gross_margin: number
  net_profit: number
  net_margin: number

  // Balance sheet
  cash_and_equiv: number
  trade_receivables: number
  total_assets: number
  current_assets: number
  non_current_assets: number
  total_liabilities: number
  current_liabilities: number
  total_equity: number

  // Ratios
  current_ratio: number
  quick_ratio: number
  cash_ratio: number
  debt_to_equity: number
  equity_ratio: number
  dso: number
  dio: number
  dpo: number
  cash_conv_cycle: number
  roe: number
  asset_turnover: number

  // AR aging
  ar_current: number
  ar_30_60: number
  ar_60_90: number
  ar_over_90: number

  // Prior year (for YoY comparison)
  py_revenue: number
  py_gross_profit: number
  py_net_profit: number

  notices: { label: string; text: string; tone: Tone }[]
}

const YEARS: Record<string, FinanceOverviewData> = {
  '2024': {
    year: 2024,
    as_of_date: '2024-12-31',
    revenue: 3_512_800,
    revenue_yoy: 6.8,
    gross_profit: 1_298_236,
    gross_margin: 36.9,
    net_profit: 298_588,
    net_margin: 8.5,
    cash_and_equiv: 512_640,
    trade_receivables: 421_900,
    total_assets: 2_684_500,
    current_assets: 1_412_300,
    non_current_assets: 1_272_200,
    total_liabilities: 1_186_900,
    current_liabilities: 812_400,
    total_equity: 1_497_600,
    current_ratio: 1.74,
    quick_ratio: 1.12,
    cash_ratio: 0.63,
    debt_to_equity: 0.79,
    equity_ratio: 0.56,
    dso: 39,
    dio: 52,
    dpo: 34,
    cash_conv_cycle: 57,
    roe: 19.9,
    asset_turnover: 1.31,
    ar_current: 231_000,
    ar_30_60: 108_500,
    ar_60_90: 54_200,
    ar_over_90: 28_200,
    py_revenue: 3_289_400,
    py_gross_profit: 1_190_120,
    py_net_profit: 261_450,
    notices: [
      { label: 'Cash', text: '1 bank statement pending reconciliation this month.', tone: 'warning' },
    ],
  },
  '2025': {
    year: 2025,
    as_of_date: '2025-12-31',
    revenue: 3_961_240,
    revenue_yoy: 12.8,
    gross_profit: 1_506_312,
    gross_margin: 38.0,
    net_profit: 368_395,
    net_margin: 9.3,
    cash_and_equiv: 638_910,
    trade_receivables: 468_720,
    total_assets: 2_961_800,
    current_assets: 1_573_600,
    non_current_assets: 1_388_200,
    total_liabilities: 1_248_300,
    current_liabilities: 856_700,
    total_equity: 1_713_500,
    current_ratio: 1.84,
    quick_ratio: 1.21,
    cash_ratio: 0.75,
    debt_to_equity: 0.73,
    equity_ratio: 0.58,
    dso: 35,
    dio: 48,
    dpo: 37,
    cash_conv_cycle: 46,
    roe: 21.5,
    asset_turnover: 1.38,
    ar_current: 258_400,
    ar_30_60: 117_900,
    ar_60_90: 58_100,
    ar_over_90: 34_320,
    py_revenue: 3_512_800,
    py_gross_profit: 1_298_236,
    py_net_profit: 298_588,
    notices: [
      { label: 'Cash', text: '2 bank statements not yet reconciled this month.', tone: 'warning' },
      { label: 'Credit', text: '1 customer over credit limit and unblocked.', tone: 'danger' },
    ],
  },
  '2026': {
    year: 2026,
    as_of_date: '2026-07-14',
    revenue: 4_182_450,
    revenue_yoy: 12.4,
    gross_profit: 1_608_922,
    gross_margin: 38.5,
    net_profit: 402_099,
    net_margin: 9.6,
    cash_and_equiv: 738_920,
    trade_receivables: 512_330,
    total_assets: 3_142_600,
    current_assets: 1_678_400,
    non_current_assets: 1_464_200,
    total_liabilities: 1_301_100,
    current_liabilities: 889_200,
    total_equity: 1_841_500,
    current_ratio: 1.89,
    quick_ratio: 1.24,
    cash_ratio: 0.83,
    debt_to_equity: 0.71,
    equity_ratio: 0.59,
    dso: 32,
    dio: 45,
    dpo: 39,
    cash_conv_cycle: 38,
    roe: 21.8,
    asset_turnover: 1.42,
    ar_current: 288_000,
    ar_30_60: 121_000,
    ar_60_90: 62_500,
    ar_over_90: 40_830,
    py_revenue: 3_961_240,
    py_gross_profit: 1_506_312,
    py_net_profit: 368_395,
    notices: [
      { label: 'Cash', text: '2 bank statements pending reconciliation this month.', tone: 'warning' },
    ],
  },
}

const DEFAULT_YEAR: FinanceOverviewData = YEARS['2026'] as FinanceOverviewData

function mockData(period?: string): FinanceOverviewData {
  return (period && YEARS[period]) || DEFAULT_YEAR
}

async function mockFetch(period?: string): Promise<FinanceOverviewData> {
  await new Promise((r) => setTimeout(r, 250))
  return mockData(period)
}

async function realFetch(period?: string): Promise<FinanceOverviewData> {
  const year = period ? Number(period) : new Date().getFullYear()
  const d = (await GetFinancialDashboardForYear(year)) as unknown as Record<string, unknown>
  // FinancialDashboard maps field-for-field onto FinanceOverviewData (the mock
  // was authored against this struct). `notices` has no backing binding — honest
  // blank (K3 hides empty widgets), never fabricated.
  return {
    year: num(d.year) || year,
    as_of_date: str(d.as_of_date),
    revenue: num(d.revenue),
    revenue_yoy: num(d.revenue_yoy),
    gross_profit: num(d.gross_profit),
    gross_margin: num(d.gross_margin),
    net_profit: num(d.net_profit),
    net_margin: num(d.net_margin),
    cash_and_equiv: num(d.cash_and_equiv),
    trade_receivables: num(d.trade_receivables),
    total_assets: num(d.total_assets),
    current_assets: num(d.current_assets),
    non_current_assets: num(d.non_current_assets),
    total_liabilities: num(d.total_liabilities),
    current_liabilities: num(d.current_liabilities),
    total_equity: num(d.total_equity),
    current_ratio: num(d.current_ratio),
    quick_ratio: num(d.quick_ratio),
    cash_ratio: num(d.cash_ratio),
    debt_to_equity: num(d.debt_to_equity),
    equity_ratio: num(d.equity_ratio),
    dso: num(d.dso),
    dio: num(d.dio),
    dpo: num(d.dpo),
    cash_conv_cycle: num(d.cash_conv_cycle),
    roe: num(d.roe),
    asset_turnover: num(d.asset_turnover),
    ar_current: num(d.ar_current),
    ar_30_60: num(d.ar_30_60),
    ar_60_90: num(d.ar_60_90),
    ar_over_90: num(d.ar_over_90),
    py_revenue: num(d.py_revenue),
    py_gross_profit: num(d.py_gross_profit),
    py_net_profit: num(d.py_net_profit),
    notices: [],
  }
}

export const fetchFinanceOverview = (period?: string): Promise<FinanceOverviewData> =>
  pick(realFetch, mockFetch)(period)
