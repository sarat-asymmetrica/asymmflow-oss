/* CRM Supplier Overview — old CRMSupplierDashboard.svelte's 2-panel analytics
 * row as a HubDescriptor. Recon K3a flags the old screen as the "dead" twin
 * of CRMCustomerDashboard: same Top-N shape but missing the bar fill. This
 * port fixes that — Top Suppliers gets the same `ranked` widget its customer
 * counterpart has. The all-suppliers card-wall grid is EntityMaster
 * territory, not a hub widget — see CrmSupplierHub.parity.md. */
import type { HubDescriptor } from '$kernel/hub'
import { fetchCrmSupplierDashboard, overduePayablesPct, type CRMSupplierDashboardData } from '../../bridge/dashboards/crm-supplier'

export const crmSupplierHubDescriptor: HubDescriptor<CRMSupplierDashboardData> = {
  entity: 'crm-supplier',
  title: 'CRM Supplier Overview',
  fetch: fetchCrmSupplierDashboard,

  kpis: [
    {
      label: 'Suppliers',
      content: 'quantity',
      value: (d) => d.totalSuppliers,
      delta: (d) => ({ text: `${d.activeSuppliers} active`, tone: 'neutral' }),
    },
    {
      label: 'YTD Purchases',
      content: 'money',
      value: (d) => d.totalPurchases,
    },
    {
      label: 'Payables',
      content: 'money',
      value: (d) => d.outstandingPayables,
    },
    {
      label: 'Overdue',
      content: 'money',
      value: (d) => d.overduePayables,
      tone: (d) => (overduePayablesPct(d) > 20 ? 'warning' : 'neutral'),
      delta: (d) => ({ text: `${overduePayablesPct(d)}%`, tone: overduePayablesPct(d) > 20 ? 'warning' : 'neutral' }),
    },
  ],

  widgets: [
    {
      type: 'ranked',
      title: 'Top Suppliers by Purchases',
      unit: 'money',
      rows: (d) =>
        d.topSuppliers.map((s, i) => ({
          rank: i + 1,
          label: s.name,
          value: s.purchases,
          pct: s.pct,
          sublabel: `${s.activePos} POs`,
          nav: { key: 'suppliers' },
        })),
    },
    {
      type: 'list',
      title: 'Active POs',
      rows: (d) => d.topSuppliers.map((s) => ({ label: s.name, value: `${s.activePos} POs` })),
    },
  ],
}
