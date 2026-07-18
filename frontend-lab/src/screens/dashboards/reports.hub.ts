/* ReportsScreen as a HubDescriptor — the old screen's 5 category tabs
 * (sales/customers/operations/inventory/financial) become the Hub's
 * `period` selector, reusing the same full-refetch mechanism
 * finance-overview.hub.ts uses for its fiscal-year selector (recon K4:
 * "cleanest fit in the whole batch... a showcase for the existing widget
 * set").
 *
 * One real wrinkle finance-overview's fiscal-year period doesn't have:
 * `HubKpiSpec.label`/`WidgetBase.title` are compile-time-fixed strings, but
 * the 5 categories are genuinely different REPORTS (no shared headline
 * metric, no shared bar-chart subject), not the same shape with different
 * values. See Reports.parity.md for how that's resolved — no KPI strip
 * (headline stats moved into a dynamically-labelled stat-grid widget
 * instead), and two generic-but-honest ranked-list widget slots grouped by
 * unit (money vs quantity) rather than one static widget per bar section. */
import type { HubDescriptor, RankedRow } from '$kernel/hub'
import { formatMoney } from '$kernel/format'
import { fetchReports, type ReportsData } from '../../bridge/dashboards/reports'

const pct = (n: number): string => `${(n * 100).toFixed(1)}%`
const days = (n: number): string => `${Math.round(n)}d`

/** Bar fill relative to the list's own max — mirrors the old screen's
 * `barPercent()` helper exactly (fill relative to the max item in the same
 * list, not a share-of-total). */
function ranked(items: { label: string; value: number; sublabel?: string }[]): RankedRow[] {
  const max = items.reduce((m, x) => Math.max(m, x.value), 0)
  return items.map((x, i) => ({
    rank: i + 1,
    label: x.label || '(unlabeled)',
    value: x.value,
    pct: max > 0 ? Math.min(100, (x.value / max) * 100) : 0,
    sublabel: x.sublabel,
  }))
}

const CATEGORY_LABEL: Record<ReportsData['category'], string> = {
  sales: 'Sales',
  customers: 'Customers',
  operations: 'Operations',
  inventory: 'Inventory',
  financial: 'Financial',
}

export const reportsDescriptor: HubDescriptor<ReportsData> = {
  entity: 'reports',
  title: 'Reports',
  subtitle: (d) => `${CATEGORY_LABEL[d.category]} · trailing month`,
  fetch: fetchReports,

  period: {
    label: 'Category',
    options: [
      { value: 'sales', label: 'Sales' },
      { value: 'customers', label: 'Customers' },
      { value: 'operations', label: 'Operations' },
      { value: 'inventory', label: 'Inventory' },
      { value: 'financial', label: 'Financial' },
    ],
    default: 'sales',
  },

  // No KPI strip: HubKpiSpec.label/content are fixed per tile, and the 5
  // categories share no common headline metric (rate vs money vs count all
  // occupy the #1 slot). Headline stats live in the "Headline Metrics"
  // stat-grid widget below instead, whose StatItem.label genuinely is a
  // function of the payload — see Reports.parity.md.
  kpis: [],

  widgets: [
    {
      type: 'stat-grid',
      title: 'Headline Metrics',
      span: 2,
      sections: (d) => {
        switch (d.category) {
          case 'sales':
            return [{ items: [
              { label: 'Win Rate', value: pct(d.winRate) },
              { label: 'Conversion', value: pct(d.conversionRate) },
              { label: 'Avg Deal Size', value: formatMoney(d.avgDealSize) },
            ] }]
          case 'customers':
            return [{ items: [
              { label: 'Avg Payment Days', value: days(d.avgPaymentDays) },
              { label: 'Collection Efficiency', value: pct(d.collectionEfficiency) },
            ] }]
          case 'operations':
            return [{ items: [
              { label: 'Avg Lead Time', value: days(d.avgLeadTime) },
              { label: 'On-Time Delivery', value: pct(d.onTimeDelivery) },
              { label: 'Pending Shipments', value: d.pendingShipments, content: 'quantity' },
            ] }]
          case 'inventory':
            return [{ items: [
              { label: 'Total Items', value: d.totalItems, content: 'quantity' },
              { label: 'Total Value', value: formatMoney(d.totalValue) },
              { label: 'Low Stock Alerts', value: d.lowStockAlerts, content: 'quantity', tone: d.lowStockAlerts > 0 ? 'danger' : 'success' },
            ] }]
          case 'financial':
            return [{ items: [
              { label: 'Receivables Outstanding', value: formatMoney(d.receivablesOutstanding) },
              { label: 'Payables Outstanding', value: formatMoney(d.payablesOutstanding) },
              { label: 'Avg Monthly Revenue', value: formatMoney(d.avgMonthlyRevenue) },
            ] }]
          default:
            return []
        }
      },
    },
    {
      // Every money-valued bar section (Pipeline, Stock Movements, Overdue
      // Receivables) lands here — RankedBarList's unit is fixed per widget,
      // so money- and count-shaped bars need separate slots (see
      // Reports.parity.md). Empty/zero for categories with no money bars.
      type: 'ranked',
      title: 'Value Breakdown',
      unit: 'money',
      rows: (d) => {
        if (d.category === 'sales') {
          return ranked(d.pipeline.map((p) => ({ label: p.stage, value: p.value, sublabel: `${p.count} deals` })))
        }
        if (d.category === 'inventory') {
          return ranked(d.movements.map((m) => ({ label: m.type, value: m.value, sublabel: `${m.count} movements` })))
        }
        if (d.category === 'financial') {
          return ranked(d.overdue.map((o) => ({ label: o.days, value: o.amount })))
        }
        return []
      },
    },
    {
      // Every count-valued bar section (Grade Distribution + Customer Type,
      // Orders by Stage) lands here.
      type: 'ranked',
      title: 'Count Breakdown',
      unit: 'quantity',
      rows: (d) => {
        if (d.category === 'customers') {
          return ranked([
            ...d.gradeDistribution.map((g) => ({ label: `Grade ${g.grade}`, value: g.count, sublabel: `${g.percentage.toFixed(0)}%` })),
            ...d.typeDistribution.map((t) => ({ label: t.label, value: t.count })),
          ])
        }
        if (d.category === 'operations') {
          return ranked(d.ordersByStage.map((s) => ({ label: s.stage, value: s.count })))
        }
        return []
      },
    },
    {
      // Financial's "Collections This Period" progress bar — a genuine
      // share-of-target, not a ranked list, so DistributionWidget fits
      // better than RankedBarList here.
      type: 'distribution',
      title: 'Collections Progress',
      orientation: 'horizontal',
      segments: (d) => {
        if (d.category !== 'financial') return []
        const remaining = Math.max(0, d.collectionTarget - d.collected)
        return [
          { key: 'collected', label: 'Collected', value: d.collected, tone: 'success' },
          { key: 'remaining', label: 'Remaining to Target', value: remaining, tone: 'neutral' },
        ]
      },
    },
  ],
}
