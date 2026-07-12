# Session pickup note — 2026-06-15 (Phase 4 + loose ends)

**Status:** all work below is **complete and GREEN** (`go test ./...` = 53 packages,
0 failures) but **UNCOMMITTED**. Branch `main`, last commit `70adcb6` (Phase 3).
The push/commit is the Commander's to do next session.

> Detailed running status lives in the auto-memory `ecosystem_dev_sprint.md`.
> This note is the human-facing "where we left off."

---

## What got done today

### 1. Phase 4 — infra decision gate (ADR-001) ✅ (docs only)
Ratified the persistence/sync stack and killed the PocketBase ghost.
- **NEW** `docs/architecture/adr/ADR-001-persistence-and-sync-stack.md` — the first
  real ADR. Local store = **pure-Go ncruces SQLite**; sync = optional + pluggable
  behind `pkg/sync` (default = SQLite-no-sync; **Supabase/Postgres** proven today;
  **Turso embedded replicas** target via the pure-Go HTTP client — CGO `go-libsql`
  rejected). **PocketBase rejected** — *not* on CGO grounds (it builds CGO-free via
  modernc; that was a training-data ghost) but on architecture: zero net-new
  capability over ncruces, 2nd SQLite engine, re-platforms sovereign auth, wrong
  risk posture for a ledger.
- **NEW** `docs/architecture/adr/README.md` (ADR convention, so 002/003/004 have a home).
- **EDIT** `docs/roadmap/SOVEREIGN_INFRASTRUCTURE_VISION.md` — pending ADR-001 → RESOLVED
  (original kept in a `<details>` for provenance).
- **EDIT** `docs/README.md` — points at `architecture/adr/`.

### 2. Deferred loose end — contract grade→terms ✅
- **EDIT** `contract_service.go` — `GetPaymentTermsForGrade` was a divergent **dead**
  switch (caller commented out). Routed it through `overlay.Active().PaymentTerms()`.
  Zero live-behaviour change; it just can't drift from canon anymore.

### 3. Deferred loose end — division SQL enumeration ✅
- **EDIT** `pkg/overlay/overlay.go` — added `DivisionNormalizationCase(col)` +
  `sqlQuote()`: generates the division-normalisation CASE/IN-list from
  `Divisions`+`Aliases` (SQL-escaped). Config-driven.
- **EDIT** `app.go` — rerouted `normalizeDivisionSQL` to the generator (old
  `LIKE '%beacon%'` → exact alias IN-list, aligned with overlay canon) and converted
  **all 11 inline-CASE backfill queries** in `backfillDivisionAwareFinanceData`
  (15 hardcoded Beacon IN-lists → 0) to use it + `divisionDefaultSQLLiteral()`.
- **NEW** `pkg/overlay/division_sql_test.go` (pins exact generator output + escaping).
- **NEW** `division_backfill_test.go` (runs the backfill on an in-memory DB; exercises
  both single- and multi-source COALESCE shapes; proves Beacon/Acme routing intact).

### 4. Deferred loose end — branding display strings ✅
- **19 company-level display strings → `activeOverlay.CompanyDisplayName`** (field already
  existed, no struct/JSON change):
  `email_service.go` (3, via a threaded 7-arg Sprintf), `report_generators.go` (2),
  `reports.go` (6), `butler_reports.go` (BI label), `app_order_customer_surface.go`
  (6 email subjects), `app_setup_documents_surface.go` (`companyName` default — byte-identical).

### 1bis. Deferred loose end — EUR→BHD rate (Commander chose CONFIG-DRIVE) ✅
Resolved a **real latent inconsistency**: import-time `eh_parser` used `EUR_TO_BHD = 0.41`
(persisted into `OfferItem.UnitPrice`) while the live path used the canonical `0.45`
(and actively blacklisted/overwrote 0.41). Now there is **one source of truth**.
- **EDIT** `pkg/overlay/overlay.go` — `CompanyOverlay.ExchangeRatesToBase` map +
  `ExchangeRateToBase(currency)` (base/empty/unknown → 1.0, case-insensitive).
- **EDIT** `data/overlay.json` — canonical rates (EUR 0.45/USD 0.376/GBP 0.52/CHF 0.425/SAR 0.100/AED 0.102).
- **NEW** `pkg/overlay/exchange_rates_test.go`.
- Routed **every** consumer to the overlay: `app_sales_pipeline.go` (central
  `defaultExchangeRateToBHD`, removed `const defaultEURToBHDRate`),
  `pkg/engines/eh_parser.go` (removed `const EUR_TO_BHD = 0.41`, `NewEHParser` reads
  overlay), `import_2026_data.go` (the persisted 0.41→0.45 collapse),
  `engine_bridge.go` (removed re-export), `app_dashboard_datafix_surface.go` (×3),
  `app_setup_documents_surface.go` (seed maps + reconciliation).
- **EDIT** `pkg/engines/eh_parser_test.go` — now asserts 100 EUR = 45 via the configured rate.
- Kept the defensive 0.410/0.44 stale-rate detection (cleans up legacy stored data).

---

## Uncommitted changes (for review + push)

```
 M app.go                                 (normalizeDivisionSQL + 11 backfill queries + FX const removed)
 M app_dashboard_datafix_surface.go       (FX → overlay)
 M app_order_customer_surface.go          (email subjects → CompanyDisplayName)
 M app_sales_pipeline.go                  (defaultExchangeRateToBHD → overlay, const removed)
 M app_setup_documents_surface.go         (companyName + FX seed → overlay)
 M butler_reports.go                      (BI label → CompanyDisplayName)
 M contract_service.go                    (grade→terms → overlay)
 M data/overlay.json                      (exchange_rates_to_base)
 M docs/README.md                         (adr/ pointer)
 M docs/roadmap/SOVEREIGN_INFRASTRUCTURE_VISION.md  (ADR-001 RESOLVED)
 M email_service.go                       (ERP header/footer → CompanyDisplayName)
 M engine_bridge.go                       (removed EUR_TO_BHD re-export)
 M import_2026_data.go                    (persisted FX → overlay)
 M pkg/engines/eh_parser.go               (removed EUR_TO_BHD const, NewEHParser → overlay)
 M pkg/engines/eh_parser_test.go          (config-driven rate test)
 M pkg/overlay/overlay.go                 (DivisionNormalizationCase + ExchangeRatesToBase)
 M report_generators.go                   (report titles → CompanyDisplayName)
 M reports.go                             (PDF/CSV headers → CompanyDisplayName)
?? division_backfill_test.go
?? docs/architecture/adr/                 (ADR-001 + README)
?? pkg/overlay/division_sql_test.go
?? pkg/overlay/exchange_rates_test.go
```

Suggested commit grouping (Commander's call): (a) Phase 4 ADR docs; (b) loose-ends
config extraction — contract terms + division SQL + branding strings; (c) EUR FX
config-drive. All three are independently green.

---

## Pending / deferred follow-ups (documented, not forgotten)

- **Banking `pkg/finance/banking/service.go` `normalizeDivisionName` switch** — has a
  divergent legacy alias set (incl. "a h s trading") and **no test**; changing it
  silently re-routes stored division values. Needs a test first, then a deliberate call.
- **Contract T&C per-division clause text** (`contract_service.go` ~740–798) — still
  hardcodes "Acme Instrumentation" in clause bodies; same bug-class as the Phase-2b
  einvoice fix (Beacon contracts say the wrong entity). Needs the contract's division
  threaded into clause generation — document semantics, so a deliberate change.
- **AI COMPANY-PROFILE blocks** (`butler_ai.go:1031`, `chat_service.go:2010`) +
  pricing prompt (`runtime_handlers.go`) + OAuth callback page (`auth_handler.go`) —
  lower-visibility hardcoded brand strings.
- **Per-division value switches** (`app_costing_exports_surface.go:1183`,
  `app_setup_documents_surface.go:1874/1932`) — need a per-division title-case display
  field on `DivisionProfile`.
- **Frontend e2e mock** `frontend/tests/e2e/helpers/mockWailsBridge.ts:405` still says
  `exchange_rate: 0.41` — TS, not in the Go gate; the frontend-suite owner should update
  the mock + any 41-expecting assertions together.
- **Wire `currency_exchange_rates` DB table into the conversion hot path** — admin rate
  edits still don't affect the math (pre-existing). Overlay is now the math's single
  source; the DB table stays seeded-from-overlay + display. A clean future enhancement.

---

## Next step

**Phase 5 — the demo app with core-loop parity** is the only remaining sprint phase
(RFQ→Costing→Offer→Order→Invoice→Payment, Finance dashboard, CRM 360, Butler, OCR,
RBAC, on the overlay paradigm). It is the largest chunk. Sprint phases 0–4 + loose-ends
are all done and green.

Also still on the Commander's own queue (pre-existing, not sprint blockers): the
`go vet -tags manual` failure on `copySanitizedDeploymentDB` redeclared (from the
initial release), and the Mistral key rotation + LICENSE finalisation.
