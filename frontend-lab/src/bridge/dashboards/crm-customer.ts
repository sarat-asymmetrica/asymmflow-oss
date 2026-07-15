/* CRM Customer Overview bridge — mirrors CRMCustomerDashboard
 * (`app_dashboard_datafix_surface.go:205`): revenue concentration, payment-
 * grade mix, and the top-customer ranking, year-scoped. Mock is deterministic
 * + synthetic (SYNTHETIC_IDENTITY.md canon); real wiring (GetCRMCustomerDashboard
 * / GetCRMCustomerDashboardByYear) lands at K5. */
import { pick } from '../runtime'
import { num, str } from '../map'
import { GetCRMCustomerDashboard, GetCRMCustomerDashboardByYear } from '$wails/go/main/App'

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

async function mockFetch(_period?: string): Promise<CRMCustomerDashboardData> {
  await new Promise((r) => setTimeout(r, 250))
  return mockData()
}

async function realFetch(period?: string): Promise<CRMCustomerDashboardData> {
  const y = period ? Number(period) : NaN
  const raw = Number.isFinite(y) ? await GetCRMCustomerDashboardByYear(y) : await GetCRMCustomerDashboard()
  const d = raw as unknown as Record<string, unknown>
  const totalRevenue = num(d.total_revenue)
  const cards = (d.top_customers as unknown[] | null) ?? []
  return {
    totalCustomers: num(d.total_customers),
    activeCustomers: num(d.active_customers),
    totalRevenue,
    revenueYoy: num(d.revenue_yoy),
    totalOutstanding: num(d.total_outstanding),
    overdueAmount: num(d.overdue_amount),
    overduePct: num(d.overdue_pct),
    // CustomerMetricCard carries no revenue-share pct — derive it (VM-legit,
    // not fabricated): the card's revenue over the dashboard total.
    topCustomers: cards.map((raw) => {
      const c = raw as Record<string, unknown>
      const revenue = num(c.total_revenue)
      return {
        name: str(c.business_name),
        revenue,
        pct: totalRevenue > 0 ? Math.round((revenue / totalRevenue) * 1000) / 10 : 0,
      }
    }),
    gradeA: { count: num(d.grade_a_count), revenue: num(d.grade_a_revenue) },
    gradeB: { count: num(d.grade_b_count), revenue: num(d.grade_b_revenue) },
    gradeC: { count: num(d.grade_c_count), revenue: num(d.grade_c_revenue) },
    gradeD: { count: num(d.grade_d_count), revenue: num(d.grade_d_revenue) },
    top3RevenuePct: num(d.top3_revenue_pct),
    top5RevenuePct: num(d.top5_revenue_pct),
    top10RevenuePct: num(d.top10_revenue_pct),
  }
}

export const fetchCrmCustomerDashboard = (period?: string): Promise<CRMCustomerDashboardData> =>
  pick(realFetch, mockFetch)(period)
