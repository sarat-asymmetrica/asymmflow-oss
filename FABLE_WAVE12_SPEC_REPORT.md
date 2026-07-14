# Wave 12 Report — The Division Registry (one vocabulary, end to end)

**Branch:** `feat/fable-wave12-division-registry` (off `main` @ `31ca7c2`). No merge, no push, no tag.
**Operating model:** Opus 4.8 orchestrator (gate) + Sonnet 5 coders. Every coder diff was independently re-verified by the orchestrator before commit.
**Thesis:** the `pkg/overlay` division registry is now the ONLY source of division vocabulary in the ERP — frontend included. A deployment's real divisions flow from `overlay.json` into every selector, validation, comparison, document label, and dashboard dispatch with **zero source edits**. The ~20-site hardcoded-pair landmine (`"Acme Instrumentation" | "Beacon Controls"`) that silently mis-scopes financial data is now **structurally impossible** — a comment-stripping audit test fails the build if a division name is ever re-hardcoded in executable code.

This was a **refactor wave**: for the synthetic default overlay, observable behavior is **byte-identical** — same selectors, same validations, same PDFs, same scoping, same goldens.

---

## 0. Headline result

| Acceptance criterion | Result |
|---|---|
| (1) Literal audit passes — zero synthetic division literals in live code paths | ✅ `TestNoSyntheticDivisionLiteralsInLiveCode` green; verified it TRIPS on an injected code literal and IGNORES prose |
| (2) A modified `overlay.json` (3 differently-named divisions) drives selectors/validation/scoping/PDFs with NO source edit | ✅ proven — `TestWave12OverrideDemo` against `overlay.demo3.json` (evidence §4) |
| (3) DEFAULT overlay behavior byte-identical (goldens/PDFs/QA sweep) | ✅ schema-golden, FX, payroll, costing-PDF, AHS-branding goldens all green; full suite green |
| (4) Stored data untouched | ✅ zero data migrations, zero stored-value rewrites — normalization lives only at the comparison/display boundary |

**Gate baseline:** `go build ./...` clean · `go vet ./...` clean · `svelte-check` 0 errors / 14 warnings (baseline held) · `vite build` clean · `go test ./...` **84 packages ok, 0 failures / 0 panics** (run ALONE, `-count=1 -timeout 1800s`).

---

## 1. Phase A — the census (the centerpiece)

Three parallel read-only recon agents produced the full census (`recon_go.md`, `recon_frontend.md`, `recon_storeddata.md`). Raw scale: **137 Go hits / 62 files** + **47 frontend hits / 20 files**. After classification, the substantive (executable) sites and their fates:

### A1 — literal census by ROLE and fate

| Role | Sites (representative) | Fate |
|---|---|---|
| **switch → legal/doc name** (PDF) | `app_costing_exports_surface.go:1249-1255` | → `DivisionDocumentDisplayName(NormalizeDivisionName(...))` (new profile field; byte-identical) |
| **document output** (PDF/console) | `purchase_order_pdf_service.go:536/554/572/597`, `pkg/engines/costing_engine.go:427`, `pkg/orchestrator/intent_processor.go:590` | → `ToUpper(CompanyDisplayName)` / `Profile(default).LegalName` / `ToUpper(DefaultDivision())` (byte-identical, verified) |
| **validation-enum** | `app_setup_documents_surface.go:1959/1998`, `financial_year_service.go:918` | → `IsKnownDivision(...)`; error lists registry keys dynamically |
| **comparison / scoping** | `financial_year_service.go:932/942/969`, `pkg/butler/context/service.go:3403/3405` | → normalize param + `DivisionNormalizationCase` (both sides, byte-identical rows) |
| **folder / path builder** | `app_setup_documents_surface.go:1901`, `pkg/setup/wizard.go:205-216`, costing/ssot filenames | → iterate / derive from registry |
| **default-for-new** (SQL DEFAULT) | `expense_service.go:95`, `payroll_service.go:62` | → DDL built from `DefaultDivision()` |
| **frontend selector / option** | FinanceHub, CostingSheet, QuickCapture, Settings, Offers, PeopleHub, Payroll, BankRecon, Expenses… | → `{#each getDivisionKeys()}` / `$derived(...)` from the store |
| **frontend type-union** | `CompanyName`, `PayrollCompany`, 7× screen-local `company?:` | → `string` + `isKnownDivision`/`normalizeDivision` runtime guards (A4 verdict) |
| **frontend default-for-new** | App.svelte nav (385/387), QuickCapture, ~40 `brand.defaultDivision` reads | → `getDefaultDivisionKey()` (`brand.defaultDivision` kept as the builtin bootstrap seed) |
| **composed string** (Spec-07) | CostingSheet delivery-terms + "is-old-default?" check | → composed FROM registry; comparison normalizes BOTH sides |
| **bespoke dashboard dispatch** | `FinanceHub.svelte:148`, `financial_year_service.go:932` | → `dashboard_variant == "ahs"` registry flag [OWNER-RULED] |
| **butler / logger / prompt** | `pkg/butler/reports/generator.go:44/150`, `intent_processor.go:540/564`, `runtime_handlers.go:585`, `butler_ai.go:1031`, `chat_service.go:2010`, classifiers | → composed from overlay / neutral wording |
| **seed / fixture** | bank accounts, license notes, letterhead assets, ssot import | → overlay-derived (byte-identical for synthetic) |
| **doc / comment** (~110) | pervasive | left as synthetic-canon references (allowed per `SYNTHETIC_IDENTITY.md`); audit strips comments |

### A2 — frontend exposure design (verdict)
One read-only binding `InfraService.GetDivisionRegistry() → { divisions:[{key, legalName, aliases, dashboardVariant}], defaultKey, companyDisplayName }` (a one-line wrap of the live `overlay.Active()` singleton), loaded once at startup into a new `divisions.svelte.ts` store that **mirrors the `initI18n` rune pattern** (module `$state`, getter functions, `initDivisions()` in App.svelte's single `onMount`). Aliases **are** exposed (B4's TS-side normalize needs them). Fallback = an audit-exempt `BUILTIN_DIVISION_REGISTRY` constant (the frontend mirror of `overlay.BuiltinDefaults()`), so the selector is **never empty** pre-login / in DESIGN_MODE / on a failed call. `brand.defaultDivision` folded in as the bootstrap seed; its ~40 consumers migrated to `getDefaultDivisionKey()`.

### A3 — stored-data reality (verdict)
~20 tables carry a `division` column. Every live write path already normalizes before storing; **no live path stores an alias/cased spelling**. Therefore canonicalizing the read-side comparisons is a **no-op on row sets for the synthetic seed** — the refactor is behavior-preserving. Two raw-comparison sites (`butler/context/service.go`, `financial_year_service.go`) were the both-sides targets for B4; both now canonicalize.

### A4 — typing strategy (verdict)
Replace the division unions with **plain `string` + runtime registry-membership/alias guards at the boundaries**, NOT a branded `DivisionKey`. Rationale: every current "boundary" already casts through `as`, so the union's compiler guarantee was already illusory; the real safety net (registry membership + alias tolerance) is runtime-only regardless of the static type. 10 union sites migrated.

---

## 2. Phase B — what shipped

- **B1 — binding + store** (`f7a47b0`): `GetDivisionRegistry()` + `divisions.svelte.ts` + startup wire + design-mode mock + regenerated bindings. No consumer migration.
- **B3 + B4(backend)** (`ded99bc`): two new optional `DivisionProfile` fields (`DocumentDisplayName`, `DashboardVariant`) + `DivisionDocumentDisplayName`/`IsKnownDivision` helpers; PDF label, validation enums, folder/path builders, SQL DEFAULT DDL, financial-year audited-override + normalization, butler revenue-KPI canonicalization, and all prompt/log/display strings converged; seed drift fixed.
- **B2 + B4(frontend) + B5** (`06579f4`): every frontend selector/default/comparison reads the store; type unions → `string` + guards; owner-ruled FinanceHub + SupplierPayments sites; CostingSheet Spec-07 composed string; backend document-output convergence (PO PDF / costing / quotation); seed hygiene; and the **B5 audit tripwire**.

### Owner rulings (financial hot-zone — spec §3)
1. **FinanceHub bespoke dashboard** (`Beacon Controls` → AHSDashboard): ruled **registry flag** → `dashboard_variant: "ahs"`; `getDashboardVariant(selectedCompany) === 'ahs'`. Also gates the `financial_year_service` audited-demo override.
2. **SupplierPayments expense-merge** (`== 'Acme Instrumentation'`): ruled **default-division** → `company === getDefaultDivisionKey()`.
Both byte-identical for the synthetic default; both now generalize correctly to a real multi-division deployment.

---

## 3. Byte-identical proof — the sensitive sites

| Site | Old literal | New expression | Identity |
|---|---|---|---|
| Costing PDF label | `case "Beacon Controls": "Beacon Controls WLL"` / `"": CompanyDisplayName` | `DivisionDocumentDisplayName(NormalizeDivisionName(data.Division))` | `""`/`"Acme…"`→"Acme Instrumentation WLL"; `"Beacon…"`→"Beacon Controls WLL" ✔ (new `document_display_name` field, distinct from the dotted `legal_name`) |
| PO PDF header | `"ACME INSTRUMENTATION WLL"` | `strings.ToUpper(CompanyDisplayName)` | ToUpper("Acme Instrumentation WLL") = "ACME INSTRUMENTATION WLL" ✔ |
| Costing console | `"ACME INSTRUMENTATION W.L.L - QUOTATION"` | `Profile(default).LegalName` | = "ACME INSTRUMENTATION W.L.L" ✔ |
| Quotation template | `"ACME INSTRUMENTATION - QUOTATION"` | `ToUpper(DefaultDivision())` | = "ACME INSTRUMENTATION" ✔ |
| financial_year SQL | `WHERE division = ?` (raw param) | normalize param first | identity on exact keys → same rows ✔ |
| butler revenue KPI | `Where("division = ?", "<literal>")` | `DivisionNormalizationCase("division") = ?` (canonical key) | same rows for synthetic seed ✔ |
| AHSDashboard badge | `"Beacon Controls WLL"` | `` `${ahsDivisionKey} WLL` `` | key + literal suffix preserves exact display ✔ |

---

## 4. AC #2 — the override demo (three divisions, zero source edits)

`overlay.demo3.json` (throwaway-synthetic: **Nimbus Metrology** / **Vanta Automation** / **Orbit Analytics**; kept in the session scratchpad, never committed) staged as `overlay.json` and loaded via the app's own `overlay.LoadOverlay`. `TestWave12OverrideDemo` (temporary, run then deleted) exercises every registry surface — verbatim output:

```
[overlay] Loaded company overlay from: …\TestWave12OverrideDemo…\overlay.json
SELECTOR (GetDivisionRegistry): defaultKey="Nimbus Metrology" companyDisplayName="Nimbus Group WLL"
  division: key="Nimbus Metrology" legalName="NIMBUS METROLOGY W.L.L" dashboardVariant="" aliases=[]
  division: key="Vanta Automation" legalName="VANTA AUTOMATION W.L.L" dashboardVariant="" aliases=[vanta automation wll vanta automation w.l.l]
  division: key="Orbit Analytics" legalName="ORBIT ANALYTICS W.L.L" dashboardVariant="ahs" aliases=[orbit analytics wll]
VALIDATION: keys+aliases accepted; old default 'Acme Instrumentation' correctly rejected
NORMALIZATION: 'vanta automation wll' -> "Vanta Automation" ; unknown -> "Nimbus Metrology"
SCOPING (DivisionNormalizationCase):
        CASE
          WHEN LOWER(TRIM(COALESCE(division, ''))) IN ('vanta automation', 'vanta automation wll', 'vanta automation w.l.l') THEN 'Vanta Automation'
          WHEN LOWER(TRIM(COALESCE(division, ''))) IN ('orbit analytics', 'orbit analytics wll') THEN 'Orbit Analytics'
          ELSE 'Nimbus Metrology'
        END
DOCUMENT: PDF labels = 'Nimbus Metrology WLL' / 'Vanta Automation WLL' / 'Orbit Analytics WLL' (no source edit)
DASHBOARD: bespoke 'ahs' dashboard now follows 'Orbit Analytics' (the flagged division) — no hardcoded name
--- PASS: TestWave12OverrideDemo (0.02s)
```

The bespoke "ahs" dashboard now follows **Orbit Analytics** (the flagged division) — proving the dispatch is driven by the registry flag, not a hardcoded name. Selectors (GetDivisionRegistry → 3), validation (IsKnownDivision accepts the 3 + aliases, rejects the old "Acme Instrumentation"), scoping (DivisionNormalizationCase spans all 3 + aliases), and PDF labels (…WLL for each) all reconfigure from config alone.

### AC #3 visual confirmation — the standing QA sweep
The Wave-11 browser sweep (`tests/e2e/wave11-sweep.spec.ts`, headless Chromium + synthetic mock bridge) was re-run: **28/28 screens pass, no blank/broken renders**. `docs/wave11-qa/1440/finance.png` confirms the FinanceHub company selector renders **"Acme Instrumentation" | "Beacon Controls"** — the two synthetic divisions, now sourced from `getDivisionKeys()` — identical to the pre-refactor baseline. (Regenerated screenshots were reverted; they carry only binary-render nondeterminism, not layout changes.) The default-overlay division-bearing UI is visually unchanged.

---

## 5. What the convergence data-migration campaign can now assume (the inherited contract)

1. **A deployment declares its divisions + aliases in `overlay.json`.** The keys are the canonical vocabulary end-to-end (frontend store, Go registry, SQL scoping).
2. **Every division comparison canonicalizes.** Read-side comparisons pass through `NormalizeDivisionName` (Go) / `normalizeDivision` (TS) / `DivisionNormalizationCase` (SQL). Migrated rows carrying an alias or cased spelling will bucket correctly.
3. **Unknown stored strings normalize to the default division** (never dropped, never mis-scoped to a random division).
4. **Zero stored-value rewrites happened.** Normalization is a boundary concern only; migrated data may be written as canonical keys, but nothing in this wave depends on that.
5. **The audit keeps the vocabulary single.** Any future code that re-hardcodes a division name fails `TestNoSyntheticDivisionLiteralsInLiveCode`.
6. **Selectors/validation/PDFs are config-reconfigurable** — a migration that introduces real division names needs no frontend/backend source change to make them appear, validate, scope, and print correctly.

---

## 6. Severity honesty — residuals & whistleblows (NOT fixed; scope-flagged)

- **`pkg/crm/domain.go:232` — GORM struct-tag column default** `default:'DAP Bahrain at your store or Acme Instrumentation'`. Struct tags are compile-time constants and cannot hold a runtime registry value; changing it would alter the schema golden (stop-and-ask). Audit-exempt as a builtin default. The app-level delivery-terms value (now registry-composed, both frontend and the costing path) overrides this DB default in practice; the residual only surfaces if a row is created with no delivery terms at all. **Recommend** a future `BeforeCreate` hook if a deployment needs the column default itself localized.
- **`einvoice_service.go:49-50` — e-invoice TRN/name/address hardcoding** (pre-existing, noted in-code). This is a per-division **VATNumber** scoping concern, not a division-*name* literal, so it is out of Wave 12's vocabulary scope and does not trip the audit. But it IS a real division-scoping gap: `DivisionProfile.VATNumber` already exists in the registry — a Beacon invoice should emit Beacon's TRN. **Recommend** a dedicated compliance-scoping wave; flagged here so convergence doesn't inherit it silently.
- **butler revenue-KPI shape** (`pkg/butler/context/service.go`): the KPI still emits a fixed two-slot (primary/secondary) breakdown. Canonicalization is correct and byte-identical, but the *shape* does not yet fan out to N>2 divisions. Out of scope (a display-shape change, not a vocabulary hardcode); noted for a future dashboard wave.

---

## 7. Definition of done

A1–A4 verdicts delivered · B1–B5 shipped and gated · three-part AC proven · gates green (suite run ALONE) · stored data untouched · Wave-10 audits + Wave-11 QA sweep unaffected (no motion/sound/announce changes; division selectors render identically for the synthetic default). Commits: `f7a47b0` · `ded99bc` · `06579f4`. No merge, no push, no tag.
