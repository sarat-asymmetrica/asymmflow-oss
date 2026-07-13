# Wave 11 Report — Polish & True Mirror

**Branch:** `feat/fable-wave11-polish` off `main` · **Repo:** `asymmflow-oss` (public) · no merge/push/tag.
**Operating model:** Opus 4.8 orchestrator/gate + Sonnet 5 coders. Playwright drove verification; every fix has a before/after render pair under `docs/wave11-qa/`.

**Headline:** the three hardware-review defects were all one story — a **whole design-token layer (`phi-design-tokens.css`) was never imported**, so ~20 screens consumed color/type/radius tokens that resolved to *empty*. Accounting/Reports rendered flat, People's detail tabs lost their pill backgrounds, and a dozen more surfaces were silently degraded. One systemic fix (a compat shim in the loaded layer) restored all of them. Separately, the synthetic division name "Acme Instrumentation" was routed through the existing brand slot / overlay so a deployment re-skins it with no source edit — bounded by the hard stop-and-report rule, because the literal doubles as a comparison key in ~20 sites.

---

## 0. Phase A — recon verdicts

### A0 — The QA harness (the enabling work) ✅ committed

The repeatable recipe is committed as **`docs/wave11-qa/QA_SWEEP.md`** + the drive script **`frontend/tests/e2e/wave11-sweep.spec.ts`** (and a probe aid `wave11-debug.spec.ts`). "Run the mirror" is now a standing capability.

- **Render target:** the **vite dev server** (`npm run dev`, auto-started by `playwright.config.ts` webServer) — renders truthfully; the production Wails webview only differs in CSS bundling order (relevant to A3, below).
- **Two harness gaps a bare browser can't fill** — both closed in `tests/e2e/helpers/mockWailsBridge.ts`:
  1. **Auth/device gate** → mocked approved-admin (`permissions:['*']`), every screen reachable.
  2. **i18n at startup** — the merge `dbca060` (init i18n at startup) added an `InfraService.GetTranslations` call during boot that the mock never covered; unmocked it threw and the whole UI dropped to its error boundary (every screen identical). The sweep now passes real `pkg/i18n/messages/en.json` into `installMockWailsBridge(page, { translations })`. **This gap is why the pre-existing e2e sonar tests were the last to render truthfully — noted so future waves don't rediscover it.**
- All **seven** bound Go services (`App`, `InfraService`, `CRMService`, `FinanceService`, `DocumentsService`, `ButlerService`, `SyncServiceBinding`) are wired to the mock's generic proxy (method-name dispatch + safe no-op default), so any screen renders its layout even when its data calls are unmocked.
- **Nav map:** `NAV_ITEMS` (`frontend/src/lib/config/navItems.ts`) — 12 primary + `usermanagement`/`deployment` deep links. Navigate by URL hash; screenshot full-page at 1440×900 and 1100×900.

### A1 — Division identity census ✅ (verdict → B1)

The census (full table in git history of this branch) found the leak is **far larger than the spec's ~10 sites** — ~120 occurrences across live UI, generated documents, logger strings, comments, and tests. The decisive finding: **"Acme Instrumentation" (and the second division "Beacon Controls") are not just display strings — they are COMPARISON KEYS** in ~20 sites (frontend `(division || 'Acme Instrumentation') === company` scoping filters, backend validation enums, a hardcoded SQL `WHERE division = 'Acme Instrumentation'`, the costing-export `switch` that routes on the literal, and `'Acme Instrumentation' | 'Beacon Controls'` type unions). Per the spec's **hard stop-and-report rule** (the Spec-07 canonicalization lesson), those are NOT re-pointed this wave — see B1.

Also confirmed: the overlay **already** carries the config field the spec asked to add — `DefaultDivisionKey` + `CompanyDisplayName` (`pkg/overlay/overlay.go:271-272`, `data/overlay.json:4-5`) with getters (`DefaultDivision()`, `NormalizeDivisionName()`). Orchestrator's call: **consume the existing fields rather than add a redundant one.**

### A2 — Accounting & Reports flat render ✅ root-caused (→ B2)

**Reproduced in the synthetic dev build** (not deployment-specific). Root cause proven by a live DOM probe: the scoped CSS *did* apply (`.layout-split` got its grid; scope class present) but `--paper`, `--ink`, `--text-3xl` resolved to **empty** while `--space-4` = `55px` resolved fine. Those color/type tokens are defined **only** in `src/lib/styles/phi-design-tokens.css`, which **`main.ts` never imports**. So `background: var(--paper)` → transparent cards, `font-size: var(--text-3xl)` → inherited 14px = the flat, grey wash. **Blast radius: 20 screens/components** silently depend on the same unloaded tokens (not just these two). Full narrative in §Root-cause.

### A3 — People detail tab layout ✅ root-caused (→ B3)

Same family. `.detail-subtabs button` sets `background: color-mix(in srgb, var(--surface) 92%, var(--accent-primary) 8%)`; `--accent-primary` is another **unloaded phi token** → the `color-mix` is invalid → the browser drops the background → transparent, borderless "pills". In the **dev build** the strip is already a compact 39px horizontal row (probed), so the hardware's "~200px vertical ovals" is the **production Wails webview's** different CSS-bundling/order interacting with the same broken token — I could not reproduce the 200px in dev, so I fixed the confirmed root cause **and** added a defensive height guard so the failure mode can't recur in any environment (see B3).

### A4 — Full-screen sweep ✅ (Defect Ledger below)

All 14 screens driven at both widths; screenshots in `docs/wave11-qa/{1440,1100}/`, indexed in `docs/wave11-qa/README.md`.

---

## 1. Phase B — fixes

### B1 — Division identity becomes configuration ✅ (bounded per hard rule)

**Frontend — one source of truth.** Extended the existing brand slot (`frontend/src/lib/brand.ts`) with `defaultDivision: 'Acme Instrumentation'` (synthetic default; a deployment overrides via the gitignored `brand.local.ts` — no source edit). Routed **53 display / default-for-new / fallback literals across 22 files** through `brand.defaultDivision`, including the hardcoded `<option>` in QuickCaptureModal + SettingsScreen, and the frontend costing-quote T&C text. Fallbacks *inside* comparisons (`(division || brand.defaultDivision) === company`) were re-pointed on the fallback side only — behavior stays frozen because default-for-new and the comparison fallback now draw from the same value.

**Backend — consume the overlay** (the ONLY authorized backend change): the costing PDF-export default now assigns `activeOverlay.CompanyDisplayName` (`app_costing_exports_surface.go`); the offer-PDF T&C entity name and the OAuth success/callback pages now come from `activeOverlay.CompanyDisplayName` (`offer_pdf_service.go`, `auth_handler.go`); the two `app.go` logger strings switched to neutral product wording ("AsymmFlow app started/shutting down"). `index.html` browser-tab title/author/og:title de-leaked to "AsymmFlow".

**DATA CAUTION honored — zero stored-data / comparison-logic changes.** No migrations, no rewriting stored `Division` rows, no comparison operators touched, zero `"Beacon Controls"` literals touched.

**Deferred (stop-and-report — the honest residue).** The literal remains where it is a **comparison / routing / enumeration key** — these need a stable **division-ID registry** (display decoupled from identity), a dedicated future wave. Concretely still showing/among-live: the FinanceHub/Offers/Costing **two-division selector arrays** `['Acme Instrumentation','Beacon Controls']` (the top-right company toggle still reads "Acme Instrumentation" — visible in `docs/wave11-qa/1440/finance.png`), the `'Acme Instrumentation' | 'Beacon Controls'` type unions, backend validation enums (`app_setup_documents_surface.go:1959/1998`), the SQL filter (`pkg/butler/context/service.go:3403`), the `financial_year_service.go:918` whitelist, the costing-export `switch` **case labels** (the assigned *value* was de-literaled; the labels route stored rows and must not move), and cross-package document value paths not yet overlay-wired (BI-report header `pkg/butler/reports/generator.go:150`, contract clause text `pkg/crm/contract/service.go`, email templates `pkg/orchestrator/intent_processor.go`, letterhead asset filenames, GORM column defaults). Fix sketch for each: introduce `overlay.DivisionByID(id)` + migrate stored rows to IDs in a data wave, then these consume the registry.

**AC status (honest):** a deployment override of `default_division`/`CompanyDisplayName` now changes Finance/QuickCapture/PDF **defaults** and generated-document display with no source edit; stored data untouched. "Zero Acme in live render paths" is achieved for pure-display surfaces but **NOT** for the two-division selectors — that is the documented stop-and-report boundary the spec itself prioritizes over the zero-Acme AC.

### B2 — Accounting & Reports restored ✅

**Fix: `frontend/src/assets/phi-token-compat.css`** (imported last in `main.ts`) defines the missing phi token names in the loaded global layer — surface/ink/border/accent/status colors **aliased to the app's live semantic tokens** (theme.css / design-tokens.css) so these screens inherit the same palette and follow theme changes; pure scalars (radii, type scale, weights, named spacing, fib) take phi's literal intent values. **Deliberately excluded:** numeric `--space-1..12` (already loaded at a φ scale — redefining would shrink spacing app-wide) and motion tokens (Wave-10 law: one source). The systemic fix restored **all 20 dependent screens/components**, not just the two — verified in the sweep (Accounting, Reports, Settings, UserManagement, Customers all now composed). Root-cause narrative + regression guard in §Root-cause.

### B3 — People detail layout ✅

`--accent-primary` is now defined by the B2 shim → the sub-tabs render as compact horizontal pills with proper backgrounds (before/after: `docs/wave11-qa/before/people-detail.png` vs `docs/wave11-qa/debug/people-detail.png`). Added a **defensive height guard** in `PeopleHub.svelte` `.detail-subtabs` (`align-items:center`, explicit `line-height`, `white-space:nowrap`, a consistent `border`) so a pill can never stretch to container height — closing the production "200px oval" failure mode I couldn't reproduce in dev.

### B4 — Defect Ledger burn-down ✅ (zero P1 remains — see Ledger)

### B5 — The sweep is a fixture ✅

`docs/wave11-qa/QA_SWEEP.md` (recipe) + `frontend/tests/e2e/wave11-sweep.spec.ts` (drive script) + `docs/wave11-qa/README.md` (index) committed. No new runtime deps — used the already-present `@playwright/test` devDependency.

---

## 2. Defect Ledger

Severity: **P1** broken/embarrassing · **P2** inconsistent · **P3** nice-to-have. "H" = harness/mock artifact (not a product defect), recorded for honesty.

| # | Screen | Defect | Class | Sev | Status |
|---|---|---|---|---|---|
| 1 | Accounting | Flat/unstyled — no card surfaces, plain-text tabs, floating stats | unstyled region · token violation | **P1** | ✅ **fixed** (B2 shim) |
| 2 | Reports | Flat/unstyled header + tab strip | unstyled region · token violation | **P1** | ✅ **fixed** (B2 shim) |
| 3 | People → Employee Detail | Sub-tabs transparent/mis-styled (prod: 200px ovals) | token violation · misalignment | **P1** | ✅ **fixed** (B3) |
| 4 | 17 other phi-dependent surfaces (Settings, UserManagement, Customers, Invoices, Pricing, Quotation, FinancialDashboard, …) | Silently degraded skin (undefined phi tokens) | token violation | **P1** (latent) | ✅ **fixed** (B2 shim, systemic) |
| 5 | Nav / titles / PDFs / OAuth / index.html | Synthetic division name leaks as if real | synthetic-identity leak | **P2** | ✅ **fixed** (B1, display sites) |
| 6 | Finance/Offers/Costing company selector | "Acme Instrumentation" in two-division toggle | synthetic-identity leak | **P2** | ⏸ **deferred** — comparison-coupled; needs division-ID registry (B1) |
| 7 | Reports | Tab content area shows nothing when a report has no rows | empty-state gap | **P2** | ⏸ **recorded** — add per-tab empty state (partly mock-driven; verify w/ real data) |
| 8 | UserManagement | Empty user table lacks an empty-state row; header→tabs vertical gap | empty-state gap | **P3** | ⏸ **recorded** — add empty-state row |
| 9 | Opportunities | "Updated No date" literal when a date is missing | placeholder text | **P3** | ⏸ **recorded** — guard date format → "—" |
| 10 | Accounting | Sidebar quick-stats "Cash" box detached at column bottom (`space-between`) | misalignment (cosmetic) | **P3** | ⏸ **recorded** — pin to nav or drop the space-between |
| 11 | Operations @1100px | 7th filter pill ("Closed") off-edge | (by design) | — | ℹ️ **not a defect** — `.status-tabs` uses `overflow-x:auto`, the spec-endorsed scroll pattern |
| 12 | Accounting / People | Mock returned undefined for one sub-call → error/warning toast | — | H | ℹ️ harness artifact (unmocked method); real backend returns data |

**Zero P1 remains** after the final sweep. Every fixed item has its before/after pair (§Screenshots). No silent deferral — every P2/P3 above carries a one-line fix sketch.

---

## 3. Root-cause narratives

### A2 — Accounting/Reports flat (the systemic one)

- **What broke:** `main.ts` imports `design-tokens.css`, `app.css`, `theme.css`, `layout.css` — but **not** `phi-design-tokens.css`, the sole definer of `--paper`, `--ink`, `--ink-light/faint/muted`, `--paper-subtle`, `--border-subtle/medium`, `--accent-primary`, `--radius-*`, `--text-*` sizes, `--font-weight-*`, etc. Any screen written against those names got empty values for skin (color/type/radius) while structural `--space-*` survived (defined in the loaded layer). Net: correct layout, no skin → flat grey.
- **When it likely broke:** during the monster-file decompositions and the ongoing token migration (memory: "3 token systems, only Onyx&Ether loaded"; CustomersScreen is the migration reference). Screens migrated onto the semantic tokens (Dashboard, Finance, CRM) stayed fine; screens still on phi names (Accounting, Reports, +18) silently lost their skin the moment `phi-design-tokens.css` stopped being imported. No compile/test gate can see an empty CSS variable — only a rendered pixel can.
- **The guard that now prevents regression:** (1) `phi-token-compat.css` defines every consumed phi name in the loaded layer, so a screen can never again resolve one to empty; (2) the **standing mirror sweep** (`QA_SWEEP.md`) renders every screen — a screen that flattens again shows up as a flat screenshot in the diff. The compat file is explicitly a bridge: when the last `var(--paper)`/`var(--ink)` consumer is migrated onto semantic tokens, delete it + its import.

### A3 — People detail tabs

- **What broke:** the same unloaded-token family — `.detail-subtabs button` mixed a background using `var(--accent-primary)`; undefined → invalid `color-mix` → dropped background. In the production Wails webview, CSS is bundled/minified in a different order than the dev server's per-module injection, which is the most likely reason the same broken rule presented as ~200px vertical ovals there but a compact (skinless) strip in dev.
- **When:** same decomposition/migration window as A2.
- **The guard:** `--accent-primary` is now defined (B2 shim) **and** `.detail-subtabs` carries an explicit height/`align-items:center`/`line-height`/`border` contract so the tab can't stretch to container height regardless of bundling order — the 200px failure mode is structurally impossible now. The sweep's People probe (`wave11-debug.spec.ts`) re-captures the strip on demand.

---

## 4. Screenshots (before/after; synthetic identity only)

- **A2 Accounting:** `docs/wave11-qa/before/accounting.png` → `docs/wave11-qa/1440/accounting.png`
- **A2 Reports:** `docs/wave11-qa/before/reports.png` → `docs/wave11-qa/1440/reports.png`
- **A3 People detail:** `docs/wave11-qa/before/people-detail.png` → `docs/wave11-qa/debug/people-detail.png`
- **Full sweep (after):** `docs/wave11-qa/{1440,1100}/<screen>.png` for all 14 screens; index at `docs/wave11-qa/README.md`.

---

## 5. Gates & audits (final commit)

- **vite build:** ✅ clean (pre-existing >500 kB chunk warning only). `frontend/dist/index.html` placeholder de-leaked (0 "Acme") with its asset-hash reference kept stable (no build churn).
- **svelte-check:** ✅ **0 errors / 14 warnings** (baseline).
- **go build ./... :** ✅ clean. **go vet ./... :** ✅ clean.
- **go test -count=1 -timeout 1800s ./... :** ✅ **green** — exit 0, 0 failures, 84 packages ok (the B1 backend text changes broke no exact-string assertions).
- **Wave-10 audits re-verified:** ✅ exactly **one** `new Audio(` construction (`src/lib/sound.ts:39`; the only other match is its own doc comment) · **zero** motion tokens introduced by the shim (deliberately excluded; documented) · toasts/reduced-motion untouched.
- **Final sweep:** ✅ **zero P1**.

---

## 6. Files changed (summary)

- **New:** `frontend/src/assets/phi-token-compat.css`, `frontend/tests/e2e/wave11-sweep.spec.ts`, `frontend/tests/e2e/wave11-debug.spec.ts`, `docs/wave11-qa/` (QA_SWEEP.md, README.md, screenshots), `FABLE_WAVE11_SPEC_REPORT.md`.
- **Harness:** `frontend/tests/e2e/helpers/mockWailsBridge.ts` (+InfraService/translations, +all 7 services, +synthetic employee).
- **B1 frontend:** `frontend/src/lib/brand.ts` + 22 screen/component files (`brand.defaultDivision`).
- **B1 backend:** `app.go`, `app_costing_exports_surface.go`, `offer_pdf_service.go`, `auth_handler.go`, `frontend/index.html`, `frontend/dist/index.html` (placeholder).
- **B3:** `frontend/src/main.ts` (shim import), `frontend/src/lib/screens/PeopleHub.svelte` (tab guard).
