# FABLE Wave 1 — Decisions Log (Mission A: Overlay Extraction)

**Scope:** Phase 2 of the ecosystem sprint — making the trading company's facts
**configuration, not code**, to prove "could a second vertical be composed today?"
Every judgment call below is recorded so a future instance (or a builder) can see
*why* the seams are where they are.

**Hard invariant honored throughout:** *Financial semantics are sacred.* Every
relocated number is byte-identical to its former hardcoded value; Phase 2 changed
**where** the numbers live, never the numbers. Verified by regression tests and a
green full suite (`go test ./...` — 49 packages, 0 failures) at every checkpoint.

---

## D0. Config package = `pkg/overlay` (foundation, Phase 2a)

- The reusable configuration mechanism is a **pure-Go `pkg/overlay` package**
  (no deps beyond stdlib), so every layer (kernel, engines, package main) can read
  it without import cycles.
- `CompanyOverlay` + `DivisionProfile` structs, `BuiltinDefaults()`, and a
  `LoadOverlay()` cascade (exe-dir > `data/` > app-data > builtin). `LoadOverlay`
  **never returns nil** — offline-first.
- **Built-in defaults = the current synthetic Acme/Beacon values.** They are
  already fictional (OSS hygiene sprint), so they double as the reference overlay's
  example config and keep all tests green. `data/overlay.json` ships as the
  annotated example.

## D1. Low-blast-radius wiring (Phase 2a/2b)

- Kept the existing free functions `normalizeDivisionName(div)` /
  `companyDocumentProfile(div)` **signatures unchanged**; they delegate to a
  package-level `activeOverlay` singleton set once at `startup()` (right after
  `LoadConfig`). Callers were not churned.
- `app.go startup()` calls `setActiveOverlay(LoadOverlay(...))` **before**
  `AutoMigrate` / `runCustomMigrations`, so the configured overlay is in effect
  during migration (this ordering is load-bearing for D6).

## D2. `pkg/engines` reads the overlay via `Active()` / `SetActive()` (Phase 2c)

- **Decision:** `pkg/engines` cannot import package main, so business rules are
  read through a process-wide singleton in `pkg/overlay`:
  `overlay.Active()` / `overlay.SetActive()`. This is the handoff's recommended
  route (a): **one source of truth**, set at startup alongside the package-main
  delegation (`setActiveOverlay` now calls `overlay.SetActive`).
- Rejected route (b) (inject rules into `NewCostingEngine(...)`) because it would
  thread config through many constructors and risk two sources of truth.

## D3. Business rules + product markup → overlay config (Phase 2c)

- Added `BusinessRules`, `[]ProductMarkupRule`, and `DefaultProductMargin` to
  `CompanyOverlay` (see `pkg/overlay/business_rules.go`). Populated
  `BuiltinDefaults()` with the **exact** prior constants, each annotated with its
  former `file:line`.
- **Grade policy consolidation:** the handoff's suggested struct had both
  `GradeAMaxDiscount`/`GradeBMaxDiscount` *and* `GradePaymentTerms`. To avoid the
  *same number living in two config fields* (which the handoff explicitly warns
  against), all per-grade numbers live in **one** place:
  `GradePaymentTerms[grade] = {Terms, AdvancePct, MaxDiscount, MaxDays}`. Accessors
  `CustomerDiscount(grade)` / `PaymentTerms(grade)` / `ProductMargin(type)` back
  the engines.
- **The ABB 15% floor was duplicated in FOUR places** (`business_invariants.go`,
  `pkg/engines/costing_engine.go`, `app_costing_exports_surface.go`, and the
  `pkg/engines/geometry_bridge.go` tender pipeline). All four now read the single
  `ABBCompetitionMinMargin`. The tender pipeline compares an achieved margin in
  **percent**, so it uses `ABBCompetitionMinMargin*100`; `0.15*100 == 15.0`
  exactly (guarded by `TestABBThresholdPercentExact`, after verifying the IEEE-754
  boundary) — byte-identical, no risk of a boundary flip.
- **Named competitors** are config (`["ABB"]`). Risk-warning messages now derive
  the competitor name from `CompetitorName()`; output is byte-identical with the
  default ("ABB"), but the string is no longer hardcoded.
- The `assessCostingRisk` Grade-C secondary 15% threshold reuses
  `ABBCompetitionMinMargin` (it was the same constant); documented here so the
  reuse is intentional, not accidental.

## D4. Division write-time defaults → overlay-driven (Phase 2d)

- Removed every hardcoded `"Acme Instrumentation"` **default-division** literal so
  a different default division is a config edit. Three categories:
  1. **Go runtime fallbacks** → `activeOverlay.DefaultDivision()` (company_branding
     resolve* helpers, the three `CurrentDivision` ports, costing-export defaults,
     butler-report default letterhead, and the **`pkg/finance/banking`** resolve\*Tx
     helpers + that package's own `normalizeDivisionName` default case).
  2. **GORM struct tags** — removed `default:'Acme Instrumentation'` from all 19
     `Division` fields. A read-only trace (subagent) showed **15 of 19** create
     paths already normalize division before write (so they were already
     config-driven via `normalizeDivisionName("")`). The **4** that relied on the
     tag — `Invoice`, `BankStatement`, `Opportunity`, `Order` — got explicit fills
     at their importer/seed/raw-map create sites. `Invoice` **inherits its order's
     division** (`normalizeDivisionName(order.Division)`), which is more correct
     than a blind default.
  3. **Migration DDL** — `app.go` column `DEFAULT`s and blank-division `UPDATE`
     backfills now use `divisionDefaultSQLLiteral()` (active overlay default,
     single-quote-escaped). Byte-identical for the reference build.
- **Why call-site fills instead of GORM `BeforeCreate` hooks:** the four structs
  embed `Base`, whose `BeforeCreate` sets the ID. A naive added hook would *shadow*
  `Base.BeforeCreate` and break ID generation; and one path (`Opportunity` OneDrive
  import) is a raw column-map `Table().Create(map)` that bypasses model hooks
  entirely. Surgical call-site fills are safer and cover the raw path.

## D5. Deferred — division-SET / alias enumeration (documented, per handoff)

The handoff sanctioned leaving the "DB helper" SQL and division-set enumerations.
These map *variant spellings → canonical* or *enumerate the division set*; they are
a distinct concern from default-fills and a larger refactor (generate from
`overlay.Divisions`). Left in place, byte-identical, as a follow-up:

- `normalizeDivisionSQL` CASE + the inline Beacon `IN ('beacon controls', ...)` CASE
  expressions in `app.go` migrations (alias spellings + their unknown-default).
- `pkg/finance/banking` `normalizeDivisionName` hardcoded alias switch — note it is
  a **second, drifted** copy of the overlay's normalization (it even carries an
  `"a h s trading"` legacy Beacon alias the overlay lacks). Consolidating the two
  `normalizeDivisionName` implementations + alias sets into the overlay is a
  follow-up.
- `financial_year_service.go` `validDivisions` map, `app_setup_documents_surface.go`
  `companies`/`!=` checks, the Acme-vs-Beacon revenue breakdown in
  `butler_ai_context.go`, and the Excel division→display-name mapping.

## D6. EUR→BHD exchange-rate inconsistency — NOT silently reconciled (needs Commander)

- There are **two different EUR→BHD rates**: `const EUR_TO_BHD = 0.41`
  (`pkg/engines/eh_parser.go`, the Rhine basket path, pinned by a test) and
  `const defaultEURToBHDRate = 0.45` (`app_sales_pipeline.go`, the live costing/
  currency-table path; the setup code even self-describes "Default rate updated to
  0.45" and treats 0.41/0.44 as stale system rates to overwrite).
- The handoff called this "an inconsistency to reconcile into one config value."
  **Reconciling would change a financial value** for one of the two paths
  (invariant #5: stop-and-ask). **Decision: left byte-identical, flagged for the
  Commander.** It is a market-rate/decision question, not a config relocation; the
  broader exchange-rate tables were also a handoff "lower-priority follow-up."

## D7. `contract_service.GetPaymentTermsForGrade` left untouched (Phase 2c)

- A *second* grade→terms mapping exists in `contract_service.go` with **different
  term strings** ("30 Days Net" vs costing's "Net 45 days") though the same advance
  fractions. Folding it into the shared `GradePolicy.Terms` would change contract
  **document output** (financial/document semantics). Left untouched; consolidation
  needs the Commander to choose which term-string vocabulary is canonical.

## D8. Branding/identity strings — out of scope for Phase 2 division work

- Hardcoded **company-name** strings in report titles, contract T&C text, the
  `companyName` setting default, OAuth/log messages, and seed/template/asset
  filenames (letterhead, costing master file, setup folder) are **identity/branding**
  (2b-adjacent), not division write-time defaults. They are noted as a branding-
  consolidation follow-up (route through `CompanyDisplayName`/`LegalName`). The one
  trivial case repointed opportunistically: `Greet()`.

## D9. Domain-package move DEFERRED (Phase 2 = config extraction only)

- The full `overlays/trading/` move — relocating RFQ→costing→quote / GRN / serial /
  delivery-note **workflow code** out of `package main` into an overlay package, and
  promoting generic capabilities (document numbering, approval routing, PDF
  assembly) to `pkg/` engines — is **deferred** (large, higher-risk). Phase 2
  deliberately scopes to the **config** extraction, which is the core of the "config
  not code" thesis. Whether the domain-package move is a Phase 2f or a later wave is
  a decision for the Commander.

---

### Verification discipline (applied every checkpoint)

- `go build -gcflags=-e ./...` (full error list, beats the 10-error cap).
- Targeted regression tests, then the **full** `go test ./...` re-run independently
  (never trusting a self-report) — green at each commit.
- A `data/overlay.json` round-trip check confirmed the shipped example loads to
  values byte-identical to `BuiltinDefaults()`.
- LSP `<new-diagnostics>` were treated as advisory (mostly pre-existing
  `interface{}`→`any` style lints in untouched code); `go build`/`go test` are the
  gate.

**Commits:** `69ded6f` (2a) · `54006c2` (2b) · `1b2920d` (2c) · `adffda7` (2d) ·
this doc + manifest + progress (2e).
