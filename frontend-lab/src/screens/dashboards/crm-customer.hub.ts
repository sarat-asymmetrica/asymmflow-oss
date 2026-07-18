/* CRM Customer Overview — old CRMCustomerDashboard.svelte's 3-panel analytics
 * row as a HubDescriptor. The Grade Distribution panel is the anti-card win
 * this port exists for: 4 flat count/revenue tiles become one donut (recon
 * K3a flags this as the clearest "should be a chart" spot in the whole
 * census). The all-customers card-wall grid is EntityMaster territory, not a
 * hub widget — see CrmCustomerHub.parity.md. */
import type { HubDescriptor } from '$kernel/hub'
import { fetchCrmCustomerDashboard, type CRMCustomerDashboardData } from '../../bridge/dashboards/crm-customer'

/** Concentration/overdue tone ramp shared by both KPI and stat-grid tiles. */
function concentrationTone(pct: number): 'neutral' | 'warning' | 'danger' {
  if (pct > 90) return 'danger'
  if (pct > 50) return 'warning'
  return 'neutral'
}

export const crmCustomerHubDescriptor: HubDescriptor<CRMCustomerDashboardData> = {
  entity: 'crm-customer',
  title: 'CRM Customer Overview',
  fetch: fetchCrmCustomerDashboard,

  kpis: [
    {
      label: 'Customers',
      content: 'quantity',
      value: (d) => d.totalCustomers,
      delta: (d) => ({ text: `${d.activeCustomers} active`, tone: 'neutral' }),
    },
    {
      label: 'YTD Business',
      content: 'money',
      value: (d) => d.totalRevenue,
      delta: (d) => ({
        text: `${d.revenueYoy >= 0 ? '+' : ''}${d.revenueYoy}% YoY`,
        tone: d.revenueYoy >= 0 ? 'success' : 'danger',
      }),
    },
    {
      label: 'Open Exposure',
      content: 'money',
      value: (d) => d.totalOutstanding,
      delta: () => ({ text: 'AR + uninvoiced orders', tone: 'neutral' }),
    },
    {
      label: 'Overdue',
      content: 'money',
      value: (d) => d.overdueAmount,
      tone: (d) => (d.overduePct > 20 ? 'warning' : 'neutral'),
      delta: (d) => ({ text: `${d.overduePct}%`, tone: d.overduePct > 20 ? 'warning' : 'neutral' }),
    },
  ],

  widgets: [
    {
      type: 'ranked',
      title: 'Top Customers by Business',
      unit: 'money',
      rows: (d) =>
        d.topCustomers.map((c, i) => ({
          rank: i + 1,
          label: c.name,
          value: c.revenue,
          pct: c.pct,
          nav: { key: 'customers' },
        })),
    },
    {
      type: 'stat-grid',
      title: 'Concentration Risk',
      sections: (d) => [
        {
          items: [
            { label: 'Top 3 %', value: d.top3RevenuePct, content: 'quantity', tone: concentrationTone(d.top3RevenuePct) },
            { label: 'Top 5 %', value: d.top5RevenuePct, content: 'quantity', tone: concentrationTone(d.top5RevenuePct) },
            { label: 'Top 10 %', value: d.top10RevenuePct, content: 'quantity', tone: concentrationTone(d.top10RevenuePct) },
          ],
        },
      ],
    },
    {
      type: 'donut',
      title: 'Revenue by Grade',
      centerLabel: 'Grades',
      segments: (d) => [
        { key: 'a', label: 'Grade A', value: d.gradeA.revenue, tone: 'success' },
        { key: 'b', label: 'Grade B', value: d.gradeB.revenue, tone: 'info' },
        { key: 'c', label: 'Grade C', value: d.gradeC.revenue, tone: 'warning' },
        { key: 'd', label: 'Grade D', value: d.gradeD.revenue, tone: 'danger' },
      ],
    },
  ],
}
