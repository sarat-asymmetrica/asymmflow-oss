# Wave 12 Spec — The Division Registry (one vocabulary, end to end)

**Mission:** make `pkg/overlay`'s division registry the ONLY source of division vocabulary in the application — frontend included — so that a deployment's real divisions flow from `overlay.json` into every selector, validation, comparison, and document with zero source edits. This wave is the **gate to the convergence data migration**: migrated production rows will carry real division names, and today ~20 comparison sites hardcode the synthetic pair `"Acme Instrumentation" | "Beacon Controls"` — a silent mis-scoping landmine for financial data. After this wave, that class of bug is structurally impossible.

**Sequencing:** after Wave 11 (merged `e4af964`). BEFORE any convergence data mission that writes real rows. This is a **refactor wave**: for the synthetic default overlay, observable behavior must be **identical** — same selectors, same validations, same PDFs, same scoping.
**Repo:** `asymmflow-oss` (PUBLIC). **Branch:** `feat/fable-wave12-division-registry` off `main`. No merge, no push, no tag.
**Operating model:** Opus 4.8 orchestrator + Sonnet 5 coders; orchestrator gates.
**Authority docs:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` (Article III: one job, one path — this wave is Article III applied to division identity) → Wave 9 keep-lists + all shipped Wave 9–11 behavior → this spec.

**Lessons inherited (do not relearn):** the Spec-07 canonicalization law — *a vocabulary change invalidates every comparison against the old vocabulary; canonicalize BOTH sides of every invariant, and distrust tests written with legacy literal strings* · the schema-golden tripwire (`tradingModels()` feeds both the golden and the startup migration) · gate baseline: vite clean, svelte-check 0 errors/14 warnings, go build/vet clean, `go test -count=1 -timeout 1800s ./...` green — run the suite ALONE, it flakes under concurrent load · Wave-10 audits permanent (one `new Audio(`, motion tokens one source, zero announce toasts, reduced-motion static) · the Wave-11 QA sweep (`docs/wave11-qa/QA_SWEEP.md`) exists — run it before/after; division selectors are visible UI.

## 0. What already exists (build on it, do not duplicate it)

`pkg/overlay/overlay.go` already has the registry core, live in production paths:
- `Divisions []DivisionProfile` — each with canonical `Key`, `LegalName`, `Aliases`, letterhead facts.
- `NormalizeDivisionName(raw)` — case-insensitive Key/alias → canonical Key, default fallback.
- `Profile(key)`, `DefaultDivisionKey`, `CompanyDisplayName`.
- `DivisionNormalizationCase(col)` — config-driven SQL CASE for backfills.
- Wave 11 added `brand.defaultDivision` (frontend, display/default only) + overlay `CompanyDisplayName` consumption.

The gap is **reach**: the frontend has NO access to the registry (hardcoded unions/arrays/options), and a set of backend sites bypass it (literal validation enums, switches, folder builders). Wave 12 is a plumbing wave, not an invention wave.

## 1. Phase A — recon (read-only; census in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | **The full literal census.** Every `"Acme Instrumentation"` / `"Beacon Controls"` in live code (frontend + Go), classified by ROLE: display · default-for-new · dropdown/selector option · type union (`CompanyName`, `PayrollCompany`, screen-local `company?:` props) · validation enum (e.g. `app_setup_documents_surface.go:1959/1998`) · comparison/scoping (SQL filters, if/switch) · folder/path builder (`:1901`) · seed/fixture (synthetic canon — stays) · doc/comment (neutral rewording only). The Wave-11 report's ~20 deferred comparison sites are the seed of this census; complete it. | B2/B3 |
| A2 | **Frontend exposure design.** No division binding exists today. Design ONE: e.g. `GetDivisionRegistry()` returning `{ divisions: [{key, legalName}], defaultKey, companyDisplayName }`, called once at startup (the `initI18n` pattern from the delivery wave) into a single `divisions` store; `brand.defaultDivision` becomes a fallback consumed only until the binding resolves (or is folded in — orchestrator's call, justified). How do the two ad-hoc nav fallbacks in `App.svelte` and QuickCapture's defaults consume it? What happens pre-login / if the call fails (fallback = builtin synthetic default — never an empty selector)? | B1 |
| A3 | **Stored-data reality.** Distinct `Division` values actually present in the synthetic seed DB per table (orders, invoices, bank_accounts, expenses, payroll…). Where does raw stored text meet a comparison today WITHOUT passing `NormalizeDivisionName`? Those are the both-sides sites for B4. | B4 |
| A4 | **The typing strategy.** The TS unions (`CompanyName`, `PayrollCompany`, screen-local `company?: 'Acme…'|'Beacon…'`) cannot survive a runtime vocabulary. Verdict on replacement: `string` + registry membership validation at the boundaries, or a branded `DivisionKey` type alias. Inventory every union site and every place that would now need a runtime guard. | B2 |

## 2. Phase B — convergence onto the registry

**B1 — The binding + the store.** Implement A2's design: one read-only binding exposing the registry (keys, legal names, default, company display name) + one frontend `divisions` store loaded at startup. Bindings regenerated (`wails generate module`). No caching cleverness — the overlay is static per process.

**B2 — Frontend convergence.** Every A1 frontend site consumes the store:
- Selector arrays / dropdown `<option>`s (CostingSheet's `divisionOptions`, QuickCapture, FinanceHub company toggle, Payroll, BankRecon, Expenses…) render from the store — labels AND values.
- Type unions replaced per A4; runtime guards at input boundaries where the compiler used to (pretend to) guarantee membership.
- Defaults flow `divisions.defaultKey` (via the existing `brand.defaultDivision` seam per A2's verdict).
- Delivery-terms style composed strings (CostingSheet's "DAP Bahrain at your store or …") compose from the registry value, and their "is this the old default?" comparisons normalize BOTH sides (Spec-07 law — compare against composed-from-registry, not a frozen literal).

**B3 — Backend convergence.** Every A1 backend site goes through the overlay:
- Validation enums → membership check against `Divisions` keys (accept aliases via `NormalizeDivisionName`; reject only true unknowns; error text lists the registry's keys dynamically).
- Folder/path builders (`companies = []string{...}`) iterate the registry.
- The PDF `divisionValue` switch in `app_costing_exports_surface.go:1246-1252` (`Key → "… WLL"`) → `Profile(key).LegalName` or an explicit new profile field if LegalName's casing differs from what documents historically printed — **byte-identical output for the synthetic pair is the acceptance test**; if LegalName doesn't match the historic strings exactly, add a dedicated field rather than changing document output.
- Butler prompt text / logger strings → composed from overlay values or neutral product wording.
- Seeds and test fixtures KEEP synthetic literals (they are data/canon, not vocabulary) — but any seed that today writes division strings should write registry keys (same strings for synthetic; the point is the reference).

**B4 — Canonicalize both sides.** Every comparison that touches a stored division value passes through `NormalizeDivisionName` (Go) or a mirrored TS normalize fed from the store's keys+aliases (expose aliases in the binding if B2 needs them). SQL filters use `DivisionNormalizationCase` or normalized parameters. **Zero stored-value rewrites: normalization lives at the comparison/display boundary, never in an UPDATE.**

**B5 — The audit becomes a tripwire.** Add a test (Go or a script wired like the repo's other audits) that greps live source (excluding tests/fixtures/seeds/docs/`BuiltinDefaults`) for the synthetic division literals and FAILS if any return. This is the regression guard that keeps the registry the one path (Article III) after this wave.

**AC (whole wave):** (1) literal audit passes — zero synthetic division literals in live code paths; (2) a modified `overlay.json` with different divisions (e.g. three divisions with new names — use throwaway synthetic names, gitignored if saved) shows correct selectors, validation, scoping, and PDFs with NO source edit — prove it and screenshot it for the report; (3) with the DEFAULT overlay, behavior is byte-identical: same golden/PDF outputs, same test results, QA sweep before/after shows no visual diff on division-bearing screens; (4) stored data untouched.

## 3. Hard boundaries

- **Financial scoping is the hot zone.** Division scopes money. Any site where canonicalizing a comparison could CHANGE which rows a query returns for the synthetic seed: stop-and-report with the evidence, don't guess. Zero authorizations this wave.
- **Zero data migrations, zero stored-value rewrites.** Convergence owns data.
- **Byte-identical synthetic behavior** — this is a refactor. Golden/PDF/document outputs for the default overlay must not change (the schema-golden tripwire is live; run the goldens).
- Synthetic invariant (public repo): any multi-division proof overlay uses throwaway synthetic names only.
- Wave-10 audits + Wave-11 QA sweep green at final commit. Keep-lists binding. No merge, no push, no tag.

## 4. Definition of done + report

Done = A1–A4 verdicts (the census is the report's centerpiece — every site listed with its role and its fate); B1–B5 shipped; the three-part AC proven (audit green · override demo screenshotted · byte-identical default); gates green (suite run ALONE).

Write `FABLE_WAVE12_SPEC_REPORT.md`, commit, paste verbatim — established template + the census table + the override-demo evidence + an explicit **"what convergence can now assume"** section (the contract the data-migration campaign inherits: e.g. "all division comparisons canonicalize; a deployment declares divisions + aliases in overlay.json; unknown stored strings normalize to the default"). Severity honesty is law.
