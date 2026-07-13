# A5 — Perceived-Latency Hot Spots (feeds B1 skeletons)

## Top 5 (ranked)
1. **`frontend/src/lib/screens/DashboardScreen.svelte`** — `onMount`→`loadData()` 3-way `Promise.all` (stats + pipeline + AR aging) behind ONE screen-wide `{#if loading}` → centered `WabiSpinner size="lg"`. Blank → everything pops at once. Per-panel flags (`pipelineLoading`, `agingLoading`) exist but wasted behind outer gate. **Skeleton:** 4 stat cards + 4-panel grid (2 list panels, 2 bar-chart panels).
2. **`frontend/src/lib/screens/WorkHub.svelte`** — widest fan-out: 6-way `Promise.all`; cold start double-loads (`load()` then `load({refreshRemote:true})`). No spinner — panels render empty until arrays populate (worse than a spinner). **Skeleton:** task-row list + project-card grid + team-board grid.
3. **`frontend/src/lib/screens/CustomerDetailView.svelte`** (+ near-identical `SupplierDetailView.svelte`) — single fat `GetCustomerFullProfile()`; centered `WabiSpinner size="lg"` then full header+tabs+table pop. **Skeleton:** header block + static tab strip + stat tiles + ~6-row table.
4. **`frontend/src/lib/screens/FinancialDashboard.svelte`** — true waterfall: `await loadAvailableYears()` blocks before `Promise.all([loadDashboard(), loadCashPosition()])`. Centered spinner + "Loading financial data...". **Skeleton:** 3-4 stat cards + 2 bar-chart panels + cash-notice list.
5. **`frontend/src/lib/screens/CRMCustomerDashboard.svelte`** (+ identical `CRMSupplierDashboard.svelte`) — single `loadDashboard()` gates whole view behind centered spinner. **Skeleton:** 4-KPI stat row + table ~8 rows × 5 cols + filter bar reserved.

Honorable mentions (same centered-spinner/table-pop): InvoicesScreen, OrdersScreen, PaymentsScreen, AccountingScreen, BankReconciliationScreen, ChequeRegisterScreen, SuppliersScreen.

## WabiSpinner
`frontend/src/lib/components/ui/WabiSpinner.svelte`, re-exported via `components/ui/index.ts`. Canvas-based hand-drawn wobbly circle (60-segment sine perturbation, breathing radius, pulsing dot). `tempo` (meditative/calm/alert), `size` (24/40/64). Only loading primitive today — used in 56 files / ~90+ sites, almost always as a single centered element gating an entire screen/panel = the blank-then-pop pattern.

## B1 skeleton targets / recommendation
Build TWO reusable primitives rather than bespoke per-screen skeletons (most offenders share table/list/stat-card shapes):
- **`TableSkeleton`** (rows × cols, header bar) — for CRM dashboards, list screens.
- **`CardGridSkeleton`** (stat-card row + panel grid) — for Dashboard, FinancialDashboard.
Apply to the 5 above with content-shaped placeholders; zero layout shift on load (reserve final dimensions). Do not change data-loading logic (flows frozen) — only swap the loading VISUAL from centered spinner to content-shaped skeleton. Skeletons must be static under reduced-motion (coordinate with B2).
