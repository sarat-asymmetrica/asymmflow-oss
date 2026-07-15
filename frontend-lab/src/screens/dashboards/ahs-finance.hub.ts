/* AHSDashboard as a HubDescriptor — a division-filtered variant of Finance
 * Overview (recon K4: "not genuinely bespoke... a division-filtered variant
 * of the standard Finance dashboard"). Deliberately thinner than
 * finance-overview.hub.ts: `DivisionFinancialSummary` carries no ratios, AR
 * aging, or YoY (see bridge/dashboards/ahs-finance.ts) — this descriptor
 * follows the FinanceOverviewHub.parity.md AHS-deferral note's preferred
 * resolution by giving AHS its OWN Hub instance (own entity, own Data type)
 * rather than conditional-visibility fields bolted onto finance-overview. */
import type { HubDescriptor } from '$kernel/hub'
import { formatMoney } from '$kernel/format'
import { fetchAhsFinance, type AhsFinanceData } from '../../bridge/dashboards/ahs-finance'

export const ahsFinanceDescriptor: HubDescriptor<AhsFinanceData> = {
  entity: 'ahs-finance',
  title: 'AHS Division Finance',
  subtitle: (d) => `${d.division} · FY${d.year} · ${d.isAudited ? 'audited' : 'unaudited'}`,
  fetch: fetchAhsFinance,

  period: {
    label: 'Fiscal Year',
    options: [
      { value: '2024', label: 'FY 2024' },
      { value: '2025', label: 'FY 2025' },
      { value: '2026', label: 'FY 2026' },
    ],
    default: '2026',
  },

  kpis: [
    {
      label: 'Revenue',
      content: 'money',
      value: (d) => d.revenue,
      delta: (d) => ({ text: d.isAudited ? 'Audited annual revenue' : `${d.invoiceCount} invoices`, tone: 'neutral' }),
    },
    {
      label: 'Net Result',
      content: 'money',
      value: (d) => d.netProfit,
      tone: (d) => (d.netProfit < 0 ? 'danger' : 'success'),
      delta: (d) => ({ text: d.netProfit < 0 ? 'Net loss' : 'Net profit', tone: d.netProfit < 0 ? 'danger' : 'success' }),
    },
    {
      label: 'Cash',
      content: 'money',
      value: (d) => d.cashEquivalents,
      delta: () => ({ text: 'Cash and bank balances', tone: 'neutral' }),
    },
    {
      label: 'Total Assets',
      content: 'money',
      value: (d) => d.totalAssets,
      delta: (d) => ({ text: `Equity: ${formatMoney(d.totalEquity)}`, tone: 'neutral' }),
    },
  ],

  // One stat-grid, mirroring the old screen's single P&L-style summary
  // table row-for-row. No distribution/comparison/callout widgets — there's
  // no aging/YoY/notices data in DivisionFinancialSummary to feed them
  // (deliberately thinner than finance-overview.hub.ts; see the bridge).
  widgets: [
    {
      type: 'stat-grid',
      title: 'Financial Summary',
      span: 2,
      sections: (d) => [
        {
          items: [
            { label: 'Total Revenue', content: 'money', value: d.revenue },
            { label: 'Cost of Sales', content: 'money', value: d.costOfSales },
            { label: 'Gross Profit', content: 'money', value: d.grossProfit },
            { label: 'Staff Costs', content: 'money', value: d.staffCosts },
            { label: 'Administrative Expenses', content: 'money', value: d.adminExpenses },
            { label: 'Net Result', content: 'money', value: d.netProfit, tone: d.netProfit < 0 ? 'danger' : 'success' },
            { label: 'Trade Receivables', content: 'money', value: d.tradeReceivables },
            { label: 'Cash & Bank', content: 'money', value: d.cashEquivalents },
            { label: 'Total Assets', content: 'money', value: d.totalAssets },
            { label: 'Total Liabilities', content: 'money', value: d.totalLiabilities },
            { label: 'Total Equity', content: 'money', value: d.totalEquity },
          ],
        },
      ],
    },
  ],
}
