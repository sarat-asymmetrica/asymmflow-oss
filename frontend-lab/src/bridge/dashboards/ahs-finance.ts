/* AHS Division Finance bridge — mirrors GetFinancialDashboardByDivision, a
 * STRICT SUBSET of Finance Overview's payload: no ratios, no balance-sheet
 * breakdown beyond assets/liabilities/equity totals, no AR aging, no YoY
 * (verified against `main.DivisionFinancialSummary` in
 * wailsjs/go/models.ts, which carries none of those fields — confirms
 * FinanceOverviewHub.parity.md's deferred-AHS note).
 *
 * The old screen resolved its division key at runtime from the division
 * registry (`getDivisions().find(d => getDashboardVariant(d.key) ===
 * 'ahs')`). This bridge stands in with the synthetic canon's "Beacon
 * Controls" division (see bridge/mock.ts's DIVISIONS) until that registry
 * is wired — see AhsFinance.parity.md. */
import { pick } from '../runtime'

const DIVISION_KEY = 'Beacon Controls'

export interface AhsFinanceData {
  division: string
  year: number
  invoiceCount: number
  revenue: number
  costOfSales: number
  grossProfit: number
  staffCosts: number
  adminExpenses: number
  netProfit: number
  cashEquivalents: number
  tradeReceivables: number
  totalAssets: number
  totalLiabilities: number
  totalEquity: number
  isAudited: boolean
  source: string
}

/* ---- mock: hand-authored per year, same style as finance-overview.ts ---- */
const YEARS: Record<string, AhsFinanceData> = {
  '2024': {
    division: DIVISION_KEY,
    year: 2024,
    invoiceCount: 96,
    revenue: 812_400,
    costOfSales: 498_600,
    grossProfit: 313_800,
    staffCosts: 128_200,
    adminExpenses: 41_500,
    netProfit: 144_100,
    cashEquivalents: 96_300,
    tradeReceivables: 118_900,
    totalAssets: 612_400,
    totalLiabilities: 248_100,
    totalEquity: 364_300,
    isAudited: true,
    source: 'Audited financial statements FY2024',
  },
  '2025': {
    division: DIVISION_KEY,
    year: 2025,
    invoiceCount: 108,
    revenue: 894_200,
    costOfSales: 541_900,
    grossProfit: 352_300,
    staffCosts: 136_800,
    adminExpenses: 44_900,
    netProfit: 170_600,
    cashEquivalents: 112_700,
    tradeReceivables: 131_400,
    totalAssets: 674_800,
    totalLiabilities: 261_500,
    totalEquity: 413_300,
    isAudited: true,
    source: 'Audited financial statements FY2025',
  },
  '2026': {
    division: DIVISION_KEY,
    year: 2026,
    invoiceCount: 61,
    // Adversarial touch: an in-year loss (Net Result tone danger by sign),
    // unlike the two prior audited years — exercises the threshold, not
    // just the happy path.
    revenue: 468_900,
    costOfSales: 412_400,
    grossProfit: 56_500,
    staffCosts: 72_100,
    adminExpenses: 23_800,
    netProfit: -39_400,
    cashEquivalents: 128_900,
    tradeReceivables: 96_200,
    totalAssets: 706_100,
    totalLiabilities: 267_900,
    totalEquity: 438_200,
    isAudited: false,
    source: 'Management accounts, in-year (unaudited)',
  },
}

const DEFAULT_YEAR: AhsFinanceData = YEARS['2026'] as AhsFinanceData

async function mockFetch(period?: string): Promise<AhsFinanceData> {
  await new Promise((r) => setTimeout(r, 250))
  return (period && YEARS[period]) || DEFAULT_YEAR
}

async function realFetch(period?: string): Promise<AhsFinanceData> {
  // GetFinancialDashboardByDivision(year, divisionKey) is a real, working
  // binding — but divisionKey must come from the division registry
  // (GetDivisionRegistry + dashboardVariant === 'ahs' lookup, Wave 12
  // division registry), never a hardcoded literal (L7). Not wired here —
  // see AhsFinance.parity.md.
  throw new Error(`INTEG gap: GetFinancialDashboardByDivision(${period ?? 'current'}, <registry-resolved division>) — wires at K5`)
}

export const fetchAhsFinance = (period?: string): Promise<AhsFinanceData> => pick(realFetch, mockFetch)(period)
