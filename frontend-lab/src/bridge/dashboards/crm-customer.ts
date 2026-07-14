/* CRM Customer Overview bridge — mirrors CRMCustomerDashboard
 * (`app_dashboard_datafix_surface.go:205`): revenue concentration, payment-
 * grade mix, and the top-customer ranking, year-scoped. Mock is deterministic
 * + synthetic (SYNTHETIC_IDENTITY.md canon); real wiring (GetCRMCustomerDashboard
 * / GetCRMCustomerDashboardByYear) lands at K5. */
import { pick } from '../runtime'

export interface CRMCustomerDashboardData {
  totalCustomers: number
  activeCustomers: number
  totalRevenue: number
  revenueYoy: number
  totalOutstanding: number
  overdueAmount: number
  overduePct: number
  topCustomers: { name: string; revenue: number; pct: number }[]
  gradeA: { count: number; revenue: number }
  gradeB: { count: number; revenue: number }
  gradeC: { count: number; revenue: number }
  gradeD: { count: number; revenue: number }
  top3RevenuePct: number
  top5RevenuePct: number
  top10RevenuePct: number
}

function mockData(): CRMCustomerDashboardData {
  return {
    totalCustomers: 118,
    activeCustomers: 94,
    totalRevenue: 3_842_600.5,
    revenueYoy: 8.6,
    totalOutstanding: 612_450.25,
    overdueAmount: 142_300.75,
    overduePct: 23.2,
    topCustomers: [
      { name: 'Gulf Fabrication W.L.L.', revenue: 612_000, pct: 15.9 },
      { name: 'Manama Process Systems', revenue: 488_000, pct: 12.7 },
      { name: 'Al Dana Engineering Co.', revenue: 365_500, pct: 9.5 },
      { name: 'Sitra Contracting', revenue: 298_750, pct: 7.8 },
      { name: 'Bahrain Water Authority — O&M', revenue: 241_900, pct: 6.3 },
    ],
    gradeA: { count: 22, revenue: 1_920_000 },
    gradeB: { count: 41, revenue: 1_150_000 },
    gradeC: { count: 38, revenue: 610_600 },
    gradeD: { count: 17, revenue: 162_000.5 },
    top3RevenuePct: 38.1,
    top5RevenuePct: 52.2,
    top10RevenuePct: 92.5,
  }
}

async function mockFetch(): Promise<CRMCustomerDashboardData> {
  await new Promise((r) => setTimeout(r, 250))
  return mockData()
}

async function realFetch(): Promise<CRMCustomerDashboardData> {
  throw new Error(
    'INTEG gap: crm-customer hub needs GetCRMCustomerDashboard() / GetCRMCustomerDashboardByYear(year) — wires at K5',
  )
}

export const fetchCrmCustomerDashboard = (): Promise<CRMCustomerDashboardData> => pick(realFetch, mockFetch)()
