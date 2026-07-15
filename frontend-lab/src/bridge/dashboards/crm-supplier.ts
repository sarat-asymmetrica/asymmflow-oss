/* CRM Supplier Overview bridge — mirrors CRMSupplierDashboard
 * (`app_order_customer_surface.go:2209`): purchase concentration + active-PO
 * load per supplier, year-scoped. Unlike its customer twin, the Go struct
 * carries no `overdue_pct` field (recon K3a) — this bridge derives it from
 * overdue/outstanding so both dashboards use the same threshold logic. Mock
 * is deterministic + synthetic (SYNTHETIC_IDENTITY.md canon); real wiring
 * (GetCRMSupplierDashboard / GetCRMSupplierDashboardByYear) lands at K5. */
import { pick } from '../runtime'

export interface CRMSupplierDashboardData {
  totalSuppliers: number
  activeSuppliers: number
  totalPurchases: number
  outstandingPayables: number
  overduePayables: number
  topSuppliers: { name: string; purchases: number; pct: number; activePos: number }[]
}

/** Derived, not carried by the Go struct — same >20% threshold the customer
 * dashboard's `overdue_pct` field drives, computed here instead of stored. */
export function overduePayablesPct(d: CRMSupplierDashboardData): number {
  if (d.outstandingPayables <= 0) return 0
  return Math.round((d.overduePayables / d.outstandingPayables) * 1000) / 10
}

function mockData(): CRMSupplierDashboardData {
  return {
    totalSuppliers: 76,
    activeSuppliers: 58,
    totalPurchases: 2_215_400.75,
    outstandingPayables: 398_600.5,
    overduePayables: 94_200.25,
    topSuppliers: [
      { name: 'Al Khalifa Trading Est.', purchases: 412_000, pct: 18.6, activePos: 6 },
      { name: 'Riffa Industrial Supplies', purchases: 356_750, pct: 16.1, activePos: 4 },
      { name: 'Muharraq Marine Works', purchases: 289_300, pct: 13.1, activePos: 3 },
      { name: 'Hidd Steel & Piping Co.', purchases: 201_450, pct: 9.1, activePos: 2 },
      { name: 'Budaiya Calibration Services', purchases: 158_900, pct: 7.2, activePos: 5 },
    ],
  }
}

async function mockFetch(): Promise<CRMSupplierDashboardData> {
  await new Promise((r) => setTimeout(r, 250))
  return mockData()
}

async function realFetch(): Promise<CRMSupplierDashboardData> {
  throw new Error(
    'INTEG gap: crm-supplier hub needs GetCRMSupplierDashboard() / GetCRMSupplierDashboardByYear(year) — wires at K5',
  )
}

export const fetchCrmSupplierDashboard = (): Promise<CRMSupplierDashboardData> => pick(realFetch, mockFetch)()
