/* Finance Overview — the old FinancialDashboard as a HubDescriptor. The
 * richest dashboard in the census (recon K3a/K3b): P&L, balance sheet,
 * ratios, AR aging, YoY — exercising the distribution, stat-grid, and
 * comparison widgets together, plus the fiscal-year period selector.
 * AHSDashboard (division-variant strict subset — no ratios/aging/YoY) is
 * NOT built here; see FinanceOverviewHub.parity.md. */
import type { HubDescriptor } from '$kernel/hub'
import type { Tone } from '$kernel/tones'
import { fetchFinanceOverview, type FinanceOverviewData } from '../../bridge/dashboards/finance-overview'

const pct = (n: number): string => `${n.toFixed(1)}%`
const days = (n: number): string => `${Math.round(n)}d`
const ratio = (n: number): string => `${n.toFixed(2)}x`

/** Threshold helper — value crosses `good`/`warn` bands; `lowerIsBetter`
 * flips the direction (e.g. debt-to-equity, DSO/DIO/CCC — smaller is healthier). */
function threshold(value: number, good: number, warn: number, lowerIsBetter = false): Tone {
  if (lowerIsBetter) {
    if (value <= good) return 'success'
    if (value <= warn) return 'warning'
    return 'danger'
  }
  if (value >= good) return 'success'
  if (value >= warn) return 'warning'
  return 'danger'
}

export const financeOverviewDescriptor: HubDescriptor<FinanceOverviewData> = {
  entity: 'finance-overview',
  title: 'Finance Overview',
  subtitle: (d) => `Fiscal year ${d.year} · as of ${d.as_of_date}`,
  fetch: fetchFinanceOverview,

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
      delta: (d) => ({ text: `${d.revenue_yoy >= 0 ? '+' : ''}${d.revenue_yoy}% YoY`, tone: d.revenue_yoy >= 0 ? 'success' : 'danger' }),
    },
    {
      label: 'Cash Balance',
      content: 'money',
      value: (d) => d.cash_and_equiv,
    },
    {
      label: 'Accounts Receivable',
      content: 'money',
      value: (d) => d.trade_receivables,
      nav: () => ({ key: 'invoices' }),
    },
    {
      label: 'Net Profit',
      content: 'money',
      value: (d) => d.net_profit,
      delta: (d) => ({ text: `Margin: ${d.net_margin}%`, tone: 'neutral' }),
    },
  ],

  widgets: [
    {
      type: 'distribution',
      title: 'Balance Sheet',
      orientation: 'horizontal',
      segments: (d) => [
        { key: 'current-assets', label: 'Current Assets', value: d.current_assets, tone: 'info' },
        { key: 'non-current-assets', label: 'Non-Current Assets', value: d.non_current_assets, tone: 'success' },
        { key: 'liabilities', label: 'Liabilities', value: d.total_liabilities, tone: 'warning' },
        { key: 'equity', label: 'Equity', value: d.total_equity, tone: 'neutral' },
      ],
    },
    {
      type: 'stat-grid',
      title: 'Key Financial Ratios',
      sections: (d) => [
        {
          title: 'Liquidity',
          items: [
            { label: 'Current Ratio', value: ratio(d.current_ratio), tone: threshold(d.current_ratio, 1.5, 1) },
            { label: 'Quick Ratio', value: ratio(d.quick_ratio), tone: threshold(d.quick_ratio, 1, 0.7) },
            { label: 'Cash Ratio', value: ratio(d.cash_ratio), tone: threshold(d.cash_ratio, 0.5, 0.2) },
          ],
        },
        {
          title: 'Solvency',
          items: [
            { label: 'Debt to Equity', value: ratio(d.debt_to_equity), tone: threshold(d.debt_to_equity, 1, 2, true) },
            { label: 'Equity Ratio', value: pct(d.equity_ratio * 100), tone: threshold(d.equity_ratio * 100, 50, 30) },
          ],
        },
        {
          title: 'Efficiency',
          items: [
            { label: 'DSO', value: days(d.dso), tone: threshold(d.dso, 30, 45, true) },
            { label: 'DIO', value: days(d.dio), tone: threshold(d.dio, 45, 60, true) },
            { label: 'DPO', value: days(d.dpo) },
            { label: 'Cash Conv. Cycle', value: days(d.cash_conv_cycle), tone: threshold(d.cash_conv_cycle, 45, 70, true) },
          ],
        },
        {
          title: 'Profitability',
          items: [
            { label: 'Gross Margin', value: pct(d.gross_margin), tone: threshold(d.gross_margin, 40, 25) },
            { label: 'Net Margin', value: pct(d.net_margin), tone: threshold(d.net_margin, 10, 5) },
            { label: 'ROE', value: pct(d.roe), tone: threshold(d.roe, 15, 8) },
          ],
        },
      ],
    },
    {
      type: 'distribution',
      title: 'Receivables Aging',
      orientation: 'vertical',
      segments: (d) => [
        { key: 'current', label: 'Current', value: d.ar_current, tone: 'success' },
        { key: '30-60', label: '30–60d', value: d.ar_30_60, tone: 'info' },
        { key: '60-90', label: '60–90d', value: d.ar_60_90, tone: 'warning' },
        { key: 'over-90', label: '90d+', value: d.ar_over_90, tone: 'danger' },
      ],
    },
    {
      type: 'comparison',
      title: 'Year over Year',
      span: 2,
      baseLabel: 'Prior Year',
      currentLabel: 'Current',
      rows: (d) => [
        { label: 'Revenue', base: d.py_revenue, current: d.revenue, currency: 'BHD' },
        { label: 'Gross Profit', base: d.py_gross_profit, current: d.gross_profit, currency: 'BHD' },
        { label: 'Net Profit', base: d.py_net_profit, current: d.net_profit, currency: 'BHD' },
      ],
    },
    {
      type: 'callout',
      title: 'Statement Check',
      items: (d) => d.notices,
    },
  ],
}
