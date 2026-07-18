# FABLE Wave K3 — Hub Archetype + Dashboards — Report

**Branch:** `exp/frontend-kernel` (LOCAL-ONLY) · **Date:** 2026-07-14
**Orchestrator:** Opus 4.8 · **Coders:** 5× Sonnet 5 (2 recon + widget builder + 2 dashboard), gated.
**Commits:** `d7d1531` (engine + main Dashboard) + this (3 dashboards + responsive nav + report).

## 1. What K3 delivered

The **fourth archetype — `Hub`** (KPI tiles + a mixed widget grid from one typed
payload) and the **4 real dashboards** of the old frontend, rendered from descriptors:

| Dashboard | Entity | Widgets exercised |
|---|---|---|
| Dashboard (landing) | dashboard | KPIs · distribution (h+v) · list · callout · activity |
| Finance Overview | finance-overview | KPIs · period selector · distribution · **stat-grid** (4 ratio sections) · vertical AR-aging · **comparison bars** (YoY) · callout |
| CRM Customer Overview | crm-customer | KPIs · **ranked-bar list** · stat-grid (concentration) · **donut** (grade revenue) |
| CRM Supplier Overview | crm-supplier | KPIs · ranked-bar list · list |

Per-screen parity ledgers in `frontend-lab/src/screens/parity/` (Finance/CRM hubs).

## 2. The widget library (the anti-card canvas)

Eight reusable, presentational widgets + a bespoke ejection slot — **zero chart
libraries** (recon confirmed the old frontend hand-rolls every bar; kernel widgets
are SVG/CSS on the palette):

- **KPI tile** — value + trend delta + threshold tone + drill-down nav.
- **distribution** — horizontal stacked bar (pipeline, balance sheet) OR vertical
  bars (AR aging), tone-coloured, clickable segments.
- **ranked-bar list** — Top-N with rank + inline bar + value (top customers/suppliers).
- **stat-tile grid** — tone-thresholded tiles in sections (financial ratios,
  concentration risk).
- **list** / **activity-feed** / **callout** — rows / timeline / toned alerts.
- **comparison bars** — paired prior-vs-current with a %-change badge (YoY).
- **donut** — NEW: an SVG ring for categorical share (grade/type mix). The clearest
  anti-card win the census found — replaces 4 flat count tiles with one proportion chart.

**Colour is computed, not eyeballed.** The categorical series palette
(`--k-series-1..6`, blue/aqua/yellow/green/violet/red) was run through the dataviz
colourblind validator on the white card surface: worst adjacent CVD ΔE **24.2**
(target ≥12). The aqua/yellow sub-3:1 contrast WARN is satisfied by the **relief
rule** — every widget ships a labelled legend, so colour is never the sole channel.
Status meaning stays on the reserved `--k-tone-*`; a series slot never impersonates a
status.

## 3. Drill-downs — `initialQuery` seeding, proven live

KPIs and widget segments carry a `NavIntent { key, query }`. `App.navigate()` switches
to the target screen and seeds its `initialQuery` (parity #4). **Verified end-to-end:**
clicking the Dashboard's "Accounts Receivable" KPI lands on the Invoices ledger with
the **Overdue filter already applied (21 rows)**. `initialQuery` was added to
EntityMaster too, so entity screens seed identically.

## 4. Scope rulings (from recon-K3a + recon-K3b)

- **Real Hubs = 4 dashboards.** FinanceHub/SalesHub/CRMHub/OperationsHub are **tab-shell
  navigators, not dashboards** → they become a `TabShell` in the K5 app shell.
  IntelligenceHub (Butler AI chat), PeopleHub (directory + payroll, PII-sensitive),
  WorkHub (kanban) → **K4 bespoke**, not Hubs.
- **AHSDashboard** is a division-variant of Finance Overview whose binding is a strict
  subset (no ratios/aging/YoY) — **ledgered** as conditional-widget-visibility keyed off
  the division registry (K5), not a second dashboard.
- **Card-wall grids** inside the CRM dashboards (all-customers / all-suppliers) are
  EntityMaster territory, **not** Hub widgets — ledgered, not built.

## 5. Fix-don't-preserve

- **CRM Supplier ranked bar restored** — the old supplier dashboard was missing the
  inline bar its customer twin had (a real inconsistency); the kernel gives both the
  same `ranked-bar` widget.
- **Overdue-tone symmetry** — the supplier struct lacks an `overdue_pct` field; it's now
  derived once in the bridge and toned with the **same >20% threshold** as the customer
  dashboard (the old screens diverged).

## 6. Engine notes / deferred

- **Per-widget independent async** (a widget's own load/error state, so one slow binding
  never blanks the page) is ledgered ENGINE for K5 — K3 mock uses one `fetch()`.
- **Responsive harness nav** — the lab's fixed 200px sidebar was squeezing content to
  ~155px at 420px (money values can't truncate). Fixed: the nav collapses to an
  off-canvas overlay at ≤720px so product screens get the full viewport — which is what
  the real K5 shell must do anyway. The detector now measures true screen width.
- **CCC formula** (DSO+DIO−DPO) folded into the ratio stat-grid; the bespoke equation
  box is ledgered (low reuse).

## 7. Gate results (green)

- `npm run check` — **0 errors, 0 warnings, 249 files**
- `npm run test` — **26 passing**
- `npm run build` — clean
- **Layout-detector — CLEAN on all 21 product screens (18 ledger/entity + 4 dashboards…
  minus overlap) at 1440 + 420**, dashboards WITH every widget rendered. Only Showcase
  (dev kitchen-sink, intentional 3000px overflow demo) is exempt.

## 8. Verdict

K3 lands the Hub archetype and the four dashboards at flip-grade: a real data-viz widget
library (validated palette, no chart dependency), working drill-downs into the K1/K2
ledgers, and the anti-card mandate delivered most visibly here — mixed widget grids,
a donut, tone-thresholded ratios, YoY comparison bars. Gates green, detector clean.
Ready for review before K4 (bespoke screens on primitives — Butler, Costing, Bank
Recon, auth chrome, and the K3-deferred People/Work/Intelligence/Pricing screens).
