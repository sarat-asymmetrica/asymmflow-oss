# Parity Ledger — CRMCustomerDashboard (old) vs CRM Customer Overview Hub (K3a widen)

Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL entities at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | 4 KPI tiles (Customers, YTD Business, Open Exposure, Overdue) | **DONE (mock) / INTEG gap (real)** | `crmCustomerHubDescriptor.kpis` — same 4 tiles, same thresholds (Overdue tones warning at `overdue_pct > 20`). Real fetch throws naming `GetCRMCustomerDashboard()`/`GetCRMCustomerDashboardByYear(year)`, wired at K5. |
| 2 | Top 10 by Business — ranked bar list | **DONE** | `ranked` widget, `unit: 'money'`, rows drill to `customers` (cross-screen nav — CRMHub's old master-detail swap becomes a screen nav here, an intentional mechanism change per recon K3a's "Hub should support both" note; nav-target-only, no in-place detail slot in this widen). |
| 3 | Concentration Risk stat tiles (Top3/5/10 %) | **DONE** | `stat-grid` widget, one section, 3 items, tone-thresholded (`concentrationTone`: neutral ≤50, warning >50, danger >90) — matches the old screen's stated thresholds (warning 50/70%, danger 90%; a single >50 catch-all covers both warning breakpoints since 70>50). |
| 4 | Grade Distribution — 4 flat stat tiles (A/B/C/D count+revenue) | **DONE, upgraded** | **The anti-card win.** Old screen: 4 static number tiles. New: one `donut` widget, "Revenue by Grade", A=success/B=info/C=warning/D=danger — recon K3a's own synthesis flagged this as the clearest "should be a chart, not tiles" spot in the whole 5-screen census. Grade *counts* are not separately re-rendered (the old tiles showed count+revenue per grade; the donut shows revenue share only) — a deliberate simplification, not a data-shape gap, since count-per-grade has no drill-down value the ranked list + revenue share doesn't already cover. |
| 5 | All Customers card-wall grid (searchable, filterable, per-card metrics) | **DEFER → EntityMaster** | Not built here. This is filterable/searchable/paginated row data with drill-through — the Customers `EntityMaster` descriptor's territory (`src/screens/customers.descriptor.ts`), not a Hub widget. Matches recon K3a's explicit verdict ("ledger-in-disguise problem"). |
| 6 | Inline "Create Customer" modal | **DEFER** | Out of this widen's scope — FormModal-archetype territory, not attempted here. |
| 7 | Year / All-Years period toggle (`GetCRMCustomerDashboard()` vs `...ByYear(year)`) | **DEFER** | `HubDescriptor.period` exists and could drive this, but the brief scoped this build to the mock payload only — no period selector wired. Flagged for a later pass alongside the real-binding INTEG work, since the two different bridge calls need to be picked by `period`, not just a query param. |

## Reading

The 3-panel analytics row (ranked list, concentration tiles, grade mix) ports
cleanly onto the Hub archetype's existing widget catalog with no new engine
work — `ranked`, `stat-grid`, and `donut` all pre-exist from the main
dashboard's widen. The one real design decision made here is #4: the grade
tiles become a donut rather than being ported 1:1, per the orchestrator's
explicit anti-card mandate for this dashboard. The card-wall grid (#5) is
correctly left out — it belongs to `EntityMaster`, and folding it in here
would recreate the "second ledger squatting inside the dashboard" smell
recon K3a called out. Both KPI drill (`nav: { key: 'customers' }`) and the
old master-detail swap converge on the same destination screen; the old
in-place swap mechanism (`dispatch('select', {id})`) is not reproduced here
since the Hub archetype only exposes cross-screen `navigate`.
