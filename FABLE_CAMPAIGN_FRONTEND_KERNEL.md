# Campaign — The Frontend Kernel: Full Migration (the "full monty")

**Mission:** replace the old frontend (60 screens, ~59k LOC, `frontend/`) with the
kernel architecture proven in `frontend-lab/` — archetype engines rendering typed
descriptors on layout primitives — reaching **flip-grade parity**, then flipping
`wails.json` and deleting the old tree. The thesis is already proven; this campaign
is execution at scale.

**Repo/branch:** worktree `C:\Projects\asymmflow\asymmflow-lab`, branch
`exp/frontend-kernel` (LOCAL-ONLY — never pushed until the flip graduates to main
by owner decision). Old `frontend/` stays untouched as the parity reference until
the flip wave. `wails.json` already points at `frontend-lab` (flip rehearsal).

**Operating model:** Opus 4.8 orchestrator + Sonnet 5 coders; orchestrator gates
every wave. Fable reviews wave reports independently before anything merges anywhere.

**Authority chain:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` →
`frontend-lab/KERNEL.md` (the constitution: 5 pillars, laws L1–L7, layout doctrine)
→ `frontend-lab/PARITY_INVOICES.md` (the parity-ledger method) → this spec.
Read KERNEL.md FIRST. Every law in it is binding for every wave below.

## 0. State of the world (verified, commit `d1b260c`)

- **Archetypes live:** `DocumentLedger`, `EntityMaster`, `FormModal` + `ActionHost`
  (one action path: `form` → FormModal, `confirm` → ConfirmDialog, else `run()`).
- **Primitives/controls:** PageShell, Stack, Row, Grid, Card, Scroll, Toolbar,
  FormGrid, DataTable, Modal; Button, SearchInput, FilterChips, Badge, EmptyState,
  ConfirmDialog. Overflow/anti-collapse doctrine baked in.
- **Engine features:** paged fetch (`fetchPage`/`loadMore`), initialQuery seeding,
  column visibility, derived filters, viewmodels in `.svelte.ts` with pure cores
  (`ledger-core.ts`, `form-core.ts`) — 19 vitest tests.
- **Bridge:** `src/bridge/index.ts` switches real-Wails vs deterministic mock at
  runtime; `src/bridge/real.ts` is the ONE Go-model adapter seam. INTEG spike
  PASSED: real RBAC, real activation, real GORM data (incl. RTL Arabic) rendered
  with ZERO descriptor changes.
- **Pilots:** Invoices (ledger) and Customers (entity) verified against mock AND
  real backend. `tests/layout-detector.mjs` = the 3-rule layout-truth check.

## 1. Inherited lessons (do not relearn — these cost real time)

1. **Kernel CSS lessons:** min-width:0 prevents overflow but permits collapse-to-
   zero (anti-collapse: flexible text gets a flex-basis, siblings wrap) · an element
   cannot `@container`-query its own size (wrapper owns container-type) · fixed-layout
   tables need `min-width = Σ column minimums` or the grow column crushes to zero ·
   declared truncation (ellipsis+nowrap) is policy, not overflow.
2. **Svelte 5:** `bind:this` erases component generics — ActionHost is the one
   `any`-typed variance seam; don't fight it elsewhere. VM = `$derived(new VM(descriptor))`
   + one `$effect` load — never onMount+effect double-fetch.
3. **⚠️ INTEG QUARANTINE (safety-critical, non-negotiable):** `wails dev` auto-
   discovers `%APPDATA%\Roaming\AsymmFlow\ph_holdings.db` — a REAL PH-adjacent
   environment whose settings.json has an ENABLED remote Postgres sync. Every dev
   run MUST use the quarantine env, set with BASH syntax:
   `export PH_DB_PATH=<scratch>\integ.db` + `export APPDATA=<scratch>\appdata`.
   Never let a run touch Roaming\AsymmFlow. Verify the scratch DB file appears
   before interacting. (`cmd/devkey <db>` prints a seeded admin key; activate via
   `window.go.main.App.ActivateLicense(key)`.)
4. **Tooling gotchas:** the "PowerShell" tool in this environment may execute via
   bash — `$env:` assignments silently no-op; use bash `export`. Go test suite runs
   ALONE (flakes under concurrent svelte-check). `go clean -cache` if C: fills.
   Playwright-MCP screenshots need lowercase `c:/Projects/...` absolute paths.
5. **Backend validation is discoverable, not guessable:** e.g. customer_type ∈
   {Corporate, Government, Individual, SME, EC}. When a real binding rejects input,
   read the Go service for the rule and encode it in the descriptor/form.
6. All repo-wide laws hold: synthetic identity (no PH strings in any committed
   file), one `new Audio(`, motion tokens single-sourced, reduced-motion static,
   division vocabulary only via registry/store, zero announce toasts.

## 2. Wave plan

Each wave: recon census → build → verify (gates + Playwright + detector at 1440/
900/420 + adversarial data) → `FABLE_WAVE_K<N>_REPORT.md` (committed) → STOP for
Fable review. Severity honesty is law. No wave touches `frontend/` (old tree).

**K1 — Ledger blitz.** Census every old screen of the document-ledger family
(Orders, PurchaseOrders, Quotations, RFQs, Offers, DeliveryNotes, GRNs,
ChequeRegister, Expenses, SupplierInvoices, SupplierPayments, Payments, plus the
credit-notes sub-ledger from PARITY_INVOICES #14). For each: parity ledger (the
PARITY_INVOICES.md method, one per screen, honest verdicts) → descriptor + real-
bridge adapter + forms/confirms for its actions → INTEG-gap entries for flows
needing new engine features. Engines may gain features ONLY when ≥2 screens need
them (e.g. row grouping, totals footer, date-range filter — likely). AC: every
K1 screen at mock+real parity minus explicitly-ledgered gaps; per-screen parity
docs are the report's centerpiece.

**K2 — Entity blitz.** Suppliers, Users/UserManagement, Products/Inventory,
warehouse masters; fold the old detail views (SupplierDetailView, CustomerDetailView,
Customer360's tabular parts) into EntityMaster profiles with ejection slots for
genuinely custom panels (e.g. Customer360 graph = SLOT, keep-or-defer verdict).

**K3 — Hub archetype + dashboards.** Design ONE `Hub` archetype: KPI tiles +
widget grid from a descriptor (`HubDescriptor`: kpis, widgets with component refs =
ejection-first since charts are bespoke). Rebuild DashboardScreen, FinanceHub,
SalesHub, CRMHub, OperationsHub, PeopleHub, WorkHub, IntelligenceHub. Chart.js is
allowed as a dependency ONLY if the census proves need; prefer simple SVG/CSS for
sparklines. Butler/dashboard drills use `initialQuery` seeding into ledgers.

**K4 — Bespoke screens on primitives.** CostingSheet, BankReconciliation +
BookBankReconciliation (matching UIs), ButlerScreen, SettingsScreen, Login/
Activation/SetupWizard/PendingApproval (auth chrome), Reports/Accounting,
remaining specials. These are ejected-by-design: hand-written views on kernel
primitives + viewmodels (L1/L5 still bind — no raw layout CSS, logic in
`.svelte.ts` with pure cores where feasible).

**K5 — App shell + INTEG completion + harness.** The real shell: sidebar nav
(brand slots from `brand.ts`), routing, auth flow against real bindings
(activation → device login → RBAC-gated nav), i18n (port the `initI18n` pattern),
divisions store fed by `GetDivisionRegistry`, `$wails` alias repointed to
`frontend-lab/wailsjs` (its own generation). Close every INTEG gap in
`bridge/real.ts` (settlement via receipts, create-from-order, UpdateCustomer
round-trip). Harness: Playwright sweep running `layout-detector.mjs` on EVERY
screen at 3 widths with adversarial fixtures; L1 tripwire (no raw layout CSS/hex/
px in screens — audit test like the division tripwire); L2 duplication tripwire
(one formatDate, one search impl). Optional if cheap: pretext arithmetic checks
for declared column widths (KERNEL.md pillar 5) — else ledger it.

**K6 — The flip.** Per-screen parity sign-off table (old vs new, every screen,
verdict + evidence). Owner smoke checklist. Then: delete `frontend/` (one commit),
`wails build` packaged smoke, full repo gates (go test ALONE, svelte-check, vite,
all audits). The flip commit message carries the parity table. NO PUSH — the
owner decides when the branch graduates.

## 3. Hard boundaries

- LOCAL-ONLY branch. No push, no tag, no public remotes. Synthetic invariant
  everywhere (mock data uses synthetic canon; `frontend-lab/` is tripwire-exempt).
- Old `frontend/` is read-only reference until K6. The PH fork is out of scope.
- Financial semantics are the hot zone: a descriptor must never change WHAT rows
  a screen shows or WHICH actions are legal vs the old screen without a ledgered,
  owner-visible verdict. When in doubt: stop-and-report.
- Zero data migrations, zero stored-value writes outside real-binding calls in
  quarantined INTEG runs.
- Gates green at every wave end: `npm run check` 0/0 · `npm run test` all green ·
  `npm run build` clean · detector zero-violation · (K5+) full repo suite ALONE.

## 4. Definition of done

All 60 old screens either rebuilt (descriptor or bespoke-on-primitives) or
explicitly retired with owner sign-off in a wave report; K6 flip executed; gates
green; parity table complete; the campaign report names every deferred item with
its ledger entry. The prize: one kernel, ~40 descriptors, a handful of bespoke
screens — and "add a screen" becomes a config change forever after.
