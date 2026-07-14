/* The landing dashboard as a HubDescriptor. Old DashboardScreen: a hand-written
 * KPI wall + decision grid; here it's declared KPIs + widgets on the Hub
 * archetype. Every KPI/widget drill seeds a ledger's initialQuery (parity #4). */
import type { HubDescriptor } from '$kernel/hub'
import { fetchMainDashboard, type MainDashboardData } from '../../bridge/dashboards/main-dashboard'

export const mainDashboardDescriptor: HubDescriptor<MainDashboardData> = {
  entity: 'dashboard',
  title: 'Dashboard',
  subtitle: (d) => `Financial year ${d.year}`,
  fetch: fetchMainDashboard,

  kpis: [
    {
      label: `Revenue FY`,
      content: 'money',
      value: (d) => d.totalRevenue,
      delta: (d) => ({ text: `${d.monthGrowth >= 0 ? '+' : ''}${d.monthGrowth}% vs last month`, tone: d.monthGrowth >= 0 ? 'success' : 'danger' }),
    },
    {
      label: 'Cash Balance',
      content: 'money',
      value: (d) => d.cashBalance,
      delta: (d) => (d.cashNote ? { text: 'Check statements', tone: 'warning' } : { text: 'Current', tone: 'success' }),
    },
    {
      label: 'Accounts Receivable',
      content: 'money',
      value: (d) => d.outstandingAr,
      tone: (d) => (d.pendingInvoices > 0 ? 'danger' : 'success'),
      delta: (d) => (d.pendingInvoices > 0 ? { text: `${d.arDaysOverdue}d avg overdue`, tone: 'danger' } : null),
      nav: () => ({ key: 'invoices', query: { filters: { status: 'Overdue' } } }),
    },
    {
      label: 'Pipeline',
      content: 'money',
      value: (d) => d.pipelineValue,
      delta: () => ({ text: 'weighted by live offers', tone: 'neutral' }),
      nav: () => ({ key: 'offers' }),
    },
  ],

  widgets: [
    {
      type: 'distribution',
      title: 'Pipeline by Stage',
      orientation: 'horizontal',
      segments: (d) => d.pipeline.map((p) => ({ key: p.stage, label: p.stage, value: p.value, tone: p.tone, nav: p.nav })),
    },
    {
      type: 'distribution',
      title: 'Collections — AR Aging',
      orientation: 'vertical',
      segments: (d) => d.aging.map((a) => ({ key: a.key, label: a.label, value: a.value, tone: a.tone, nav: a.nav })),
    },
    {
      type: 'list',
      title: 'Operating Focus',
      rows: (d) => d.focus.map((f) => ({ label: f.label, detail: f.detail, tone: f.tone, nav: f.nav })),
    },
    {
      type: 'callout',
      title: 'Alerts',
      items: (d) => d.alerts,
    },
    {
      type: 'activity',
      title: 'Tasks',
      emptyMessage: 'No open tasks.',
      items: (d) => d.tasks.map((t) => ({ title: t.title, subtitle: t.subtitle, timestamp: t.timestamp })),
    },
  ],
}
