# Parity Ledger — CRMSupplierDashboard (old) vs CRM Supplier Overview Hub (K3a widen)

Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL entities at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | 4 KPI tiles (Suppliers, YTD Purchases, Payables, Overdue) | **DONE (mock) / INTEG gap (real)** | `crmSupplierHubDescriptor.kpis`. `YTD Purchases` has no delta, matching the old screen (`CRMSupplierDashboard` struct carries no YoY field, unlike the customer twin — recon K3a's noted asymmetry, preserved honestly rather than fabricated). Real fetch throws naming `GetCRMSupplierDashboard()`/`GetCRMSupplierDashboardByYear(year)`, wired at K5. |
| 2 | Overdue % of payables, computed inline in the old template | **DONE, fixed** | Old screen computed the % ad hoc per-render with no reusable function; this port derives it once via `overduePayablesPct()` in the bridge and applies the **same >20% warning threshold** the customer dashboard's `overdue_pct` field drives — recon K3a explicitly flagged the two dashboards' inconsistent overdue-tone logic as a bug to fix during the port, not preserve. |
| 3 | Top Suppliers by Purchases — rank + name + rating + amount, **no bar fill** | **DONE, fixed** | **The bug this widen exists to fix.** Old screen rendered this as a plain `list/table-widget`, missing the inline bar its customer counterpart has (recon K3a: "the Top Suppliers list lost the bar visualization... an inconsistency worth fixing in the rebuild"). Now a `ranked` widget, `unit: 'money'`, same shape as CRM Customer's Top-10, with `sublabel` carrying the active-PO count and rows drilling to `suppliers`. Star rating (`SupplierMetricCard.rating`) is not carried into the row — no `RankedRow` field for it today; flagged as a possible future `sublabel` extension, not a gap in this pass since active-PO count is arguably the more decision-relevant sublabel for this KPI's job (purchasing volume, not quality rating). |
| 4 | Active POs — top-5 suppliers × active PO count | **DONE** | `list` widget, same 5 suppliers as the ranked panel, `value` = `"{n} POs"`. |
| 5 | All Suppliers card-wall grid (brand/country/rating badges, searchable) | **DEFER → EntityMaster** | Not built here — same verdict as the customer twin's card wall. Belongs to the Suppliers `EntityMaster` descriptor (`src/screens/suppliers.descriptor.ts`), not the Hub widget catalog. |
| 6 | Inline "Create Supplier" modal | **DEFER** | Out of this widen's scope — FormModal-archetype territory. |
| 7 | Year / All-Years period toggle | **DEFER** | Same as the customer twin — no period selector wired in this pass; flagged alongside it for the real-binding work. |

## Reading

This dashboard is the smaller of the CRM pair — 2 analytics panels instead
of 3, no grade/concentration equivalent (the supplier struct has no grade
field to visualize). The headline fix is #3: recon K3a caught the old
supplier screen missing the ranked-bar visualization its customer twin has,
despite both being described as parallel "Top-N" panels — this port closes
that gap by giving both dashboards the identical `ranked` widget type. #2
closes the second inconsistency recon K3a flagged (mismatched overdue-tone
threshold logic) by centralizing the derived percentage in one bridge
function rather than reintroducing a second inline computation. Both
dashboards now share the same widget vocabulary and the same tone
thresholds — the "near-twins currently out of sync" problem the census
predicted is resolved, not carried forward.
