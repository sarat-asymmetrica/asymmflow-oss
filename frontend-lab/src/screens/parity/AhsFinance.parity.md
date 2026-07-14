# Parity Ledger — AHSDashboard (old) vs AHS Division Finance HubDescriptor

Verdicts: DONE / EQUIV / ENGINE / SLOT / INTEG / DEFER.

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Division-scoped financial summary (year selector) | **EQUIV** | Own Hub instance (own entity/Data type), NOT conditional fields bolted onto Finance Overview. `DivisionFinancialSummary` carries no ratios/aging/YoY — a strict subset of the main finance struct — so a separate, deliberately-thin Hub is the honest shape (the resolution FinanceOverviewHub.parity.md preferred). Year selector = Hub `period`. |
| 2 | 4 KPI tiles (Revenue / Net Result / Cash / Total Assets) | **DONE** | Net Result toned by sign (success/danger). |
| 3 | P&L-style summary table (11 rows) | **DONE** | Rendered as a `stat-grid` "Financial Summary" (Revenue/COGS/Gross Profit/Staff/Admin/Net Result/Receivables/Cash/Assets/Liabilities/Equity), Net Result toned by sign. |
| 4 | Division key resolution | **INTEG** | `GetFinancialDashboardByDivision(year, divisionKey)` is real, but the division key MUST come from the registry (`GetDivisionRegistry` + `dashboardVariant==='ahs'`, Wave 12) — never hardcoded (L7). Real fetch throws an honest INTEG gap naming this; mock uses synthetic "Beacon Controls". |
| 5 | "No Data" onboarding empty-state | **DEFER** | Division-key-driven copy; ports cleanly at INTEG with the registry. |

## Reading
AHS is a thin division variant of the finance dashboard, and is honestly modeled as
such — a separate Hub instance matching the real subset binding, not a data-faking
conditional overlay. The only real dependency is the division-registry integration
(INTEG/K5), which is preserved as an honest throw rather than a hardcoded division literal.
