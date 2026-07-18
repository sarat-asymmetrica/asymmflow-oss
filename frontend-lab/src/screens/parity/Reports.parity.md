# Parity Ledger — ReportsScreen (old) vs Reports HubDescriptor

Verdicts: DONE / EQUIV / ENGINE / SLOT / INTEG / DEFER.

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | 5 category tabs (sales/customers/operations/inventory/financial) | **EQUIV** | Modeled as the Hub `period` selector (full-refetch mechanism, same as Finance Overview's fiscal-year selector). `GetReportData(category,'month')` is REAL and wired directly (not INTEG). |
| 2 | Per-category headline stats | **EQUIV** | The 5 categories share NO common headline metric (a %, a count, 3 money figures occupy the #1 slot). `HubKpiSpec.label` is a fixed string, so a static KPI strip can't express this. Resolved: `kpis: []` (Hub renders nothing) + a `stat-grid` "Headline Metrics" widget whose `StatItem.label` IS a function of the payload. Honest, not a regression. |
| 3 | Hand-rolled horizontal bar charts per section | **DONE** | Mapped to `RankedBarList` (money "Value Breakdown", count "Count Breakdown" — separate slots because `RankedBarList.unit` is fixed per widget) + one `distribution` for financial's Collections-progress (share-of-target). Empty slots auto-hidden (Hub `hasContent`). |
| 4 | CSV export (`ExportReport`, real) | **ENGINE** | `HubDescriptor` has no `actions` concept (unlike `LedgerDescriptor`). A Hub-level export/action affordance is a real engine gap — ledgered, not built. |
| 5 | PDF/Excel export "coming soon" stubs | **DEFER** | Never implemented in the old screen. |

## Reading
Reports is the cleanest Hub fit in the K4 batch. The one real tension — a
`period`-driven Hub assumes "same shape, different values", but report categories are
genuinely different shapes — is resolved honestly by moving the per-category headline
into a payload-computed stat-grid rather than a fixed KPI strip. Same class of gap
`FinanceOverviewHub.parity.md` flagged for AHS; confirmed independently. Only deferred
items: a Hub-level export action (ENGINE) and the PDF/Excel stubs (DEFER).
