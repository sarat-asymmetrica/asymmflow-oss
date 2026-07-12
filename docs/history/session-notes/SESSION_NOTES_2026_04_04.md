# Session Notes — 2026-04-04 — Commercial Workflow Stabilization, Data Repair, Butler Action Completion

## Session Summary

This session closed a large commercial workflow stabilization pass across sales, operations, finance, OCR ingestion, deployment packaging, licensing, and Butler AI.

The original client-reported regressions around:
- wrong `IMP-*` order numbering,
- broken supplier edit and supplier invoice flows,
- order/offer line-item double counting,
- opportunity-to-costing handoff,
- offer persistence,
- costing-sheet commercial logic,
- PO / invoice / bank reconciliation regressions,
- and incomplete Butler execution behavior

were addressed in code, data, and packaging.

Most business-critical issues are now resolved. The main remaining caveat is Mac packaging: `go build ./...` is clean, but fresh Mac `wails build` still hits the pre-existing linker failure, while Windows `wails build` passes.

---

## Major Work Completed

### 1. Commercial Workflow Repairs

- Fixed malformed imported customer orders using `IMP-*` numbers.
- Removed synthetic `Total for Order` rows from both `offer_items` and `order_items`.
- Hardened imported commercial document repair so totals are based on real line items only.
- Fixed supplier edit payload handling so supplier updates send clean `SupplierMaster` data rather than malformed composite objects.
- Fixed order update behavior and blocked synthetic summary rows from being treated as real order items.
- Fixed offer edit persistence for customer/opportunity-linked metadata and commercial header fields.
- Added / stabilized opportunity-to-costing navigation with prefilled customer/opportunity details.
- Fixed sales hub tab routing so `RFQs`, `Offers`, and `Customer Orders` remain clickable.

### 2. Costing Sheet and Commercial Output Changes

- Added per-line exchange rate support in costing.
- Changed freight to percentage-driven logic with default freight now `9%`, still editable by user.
- Switched pricing logic from margin framing to markup framing where required.
- Renamed pricing fields to support `Quote Price` plus system-generated suggested price behavior.
- Added costing-sheet `Subject` and editable `PDF Body` fields with default body text.
- Ensured these new commercial fields persist.
- Added broader persistence coverage for quote type, subject, body, and commercial payloads.

### 3. PDF / Document Output Fixes

- Centered key quotation / PO document titles instead of right-aligned rendering.
- Improved costing sheet PDF structure so company, attention, subject, and editable body render in a more correct commercial flow.
- Improved PO / offer / costing PDF formatting for functional parity with the client examples.
- Techno-commercial parity is functionally improved, though not claimed as pixel-perfect.

### 4. Opportunity / OCR Pipeline Repairs

- OCR document save now creates opportunities directly where appropriate rather than hiding them in RFQ-only storage.
- OCR-created opportunities default to `New`.
- Duplicate opportunity detection added with warning behavior instead of silent duplicate creation.
- Opportunities list now uses LIFO-friendly recency sorting.
- Opportunity search was expanded and corrected across customer, project, folder/reference, stage, owner, notes, and terms.
- Fixed stage normalization so missing stages do not incorrectly default to `Qualified`.

### 5. Supplier PO / Supplier Invoice / Operations Fixes

- Fixed supplier PO creation regressions, including sales role permission alignment for `po:create`.
- Fixed PO date validation issues caused by local/date parsing mismatches.
- Fixed malformed draft POs and cleaned invalid historical draft records from active/package DBs.
- Fixed PO detail modals to show supplier, line items, and correct totals more reliably.
- Repaired legacy corrupted completed POs where line items did not match stored header totals.
- Added supplier selection in supplier invoice edit.
- Fixed supplier invoice update date parsing failures.
- Hydrated full supplier invoice detail before edit so modal fields and line items reflect the actual invoice.
- Added optional supplier payment ledger creation during supplier invoice update.

### 6. Finance / Bank Reconciliation Repairs

- Bank statement OCR/import logic now preserves VAT, bank charges, and fees as separate rows instead of folding them into transaction totals.
- Removed the dangerous debit/credit auto-flip behavior; mismatches are now left for review instead of silently rewritten.
- Bank statement page import now uses the correct OCR path and no longer fails the way it did in the screenshots.
- Same-period bank statement imports now warn and require replace confirmation instead of silently overwriting.
- Expanded bank reconciliation candidate loading so more invoices are available during matching.
- Connected Bank Recon entry points from:
  - customer invoices,
  - supplier invoices,
  - payments received,
  - payments made.
- Added comma formatting and finance display cleanups where needed.

### 7. Dashboard / UI Cleanups

- Reduced dashboard KPI card footprint so desktop users do not need to scroll as much.
- Removed `Open Follow-ups` from the dashboard.
- Improved number formatting with commas on key finance surfaces.
- Reduced Butler / intelligence panel height so it fits better within the app viewport.

### 8. Traceability Removal

- Removed traceability tab and standalone screen from Operations.
- Deleted dead frontend traceability screen path.

### 9. Licensing / Packaging / Deployment Hardening

- Flushed packaged activation/device state.
- Regenerated packaged employee license keys.
- Hardened config/database-path resolution so packaged apps prefer bundled DB and bundled `.env` instead of stale runtime or appdata fallbacks.
- Added explicit packaged DB path pinning in deployment `.env`.
- Rebuilt Windows executable successfully with the path fixes.
- Rebuilt Mac app bundle where possible during the session, but final fresh Mac `wails build` remains blocked by the known linker issue.
- Refreshed deployment package line several times, final notable package:
  - `deploy_package/AsymmFlow_Deploy_2026_04_02_151020`

### 10. Butler AI Completion

This was the final originally unfinished phase from the agreed plan.

- Improved Butler grounding so it gives the closest useful, source-grounded answer instead of hard refusal where possible.
- Expanded Butler’s business-year coverage and commercial summary grounding.
- Completed Butler write-action execution flow with preview/confirm behavior before writes.
- Butler now supports or improves support for:
  - create customer,
  - create supplier,
  - create customer contact,
  - create supplier contact,
  - create opportunity,
  - create follow-up,
  - create order,
  - create offer draft,
  - update opportunity stage/details,
  - approval/update flows already supported by existing action plumbing.
- Clarification-first behavior was reinforced when required data is missing.
- Action contract, target aliases, validation metadata, and runtime action-state rendering were expanded.

---

## Database Changes Made During This Session

### Data Repair

- Removed all remaining `IMP-*` order numbers from live data.
- Removed all synthetic `Total for Order` rows from:
  - `offer_items`
  - `order_items`
- Repaired historical imported commercial totals so header totals reflect real items only.
- Cleaned malformed draft supplier POs with unusable supplier/item data.
- Reconciled `45` corrupted historical completed supplier POs whose line items disagreed with stored headers.
- Repaired legacy supplier invoice header totals where line items existed but header totals were zero/blank.
- Resynced stale runtime DB so 2026 customer orders data matched the working DB again.

### Customer Master Seeding

- Seeded canonical `CID` and customer short code data from the supplied reference list.
- Updated many matched existing customers to canonical CID/short-code values.
- Inserted missing reference rows where needed.
- Important note: legacy duplicate customer variants still remain in some cases, so CID normalization is improved but not the same as full customer dedupe/merge.

### Role / Permission Data

- Updated role data so `sales` includes `po:create`.
- Synced permission changes into both workspace and runtime DB contexts during testing.

### Deployment DB Sanitization

- Packaged DBs were sanitized to:
  - `0` activated keys,
  - `0` developer keys,
  - `0` device rows,
  - `0` device-user rows.

---

## Backups Created

Notable backups created during this session:

- `backups/ph_holdings_before_deploy_rekey_2026_04_02_120928.db`
- `backups/ph_holdings_before_po_cleanup_2026_04_02_130000.db`
- `backups/ph_holdings_before_po_reconciliation_2026_04_02_142500.db`
- `backups/ph_holdings_before_customer_reference_seed_2026_04_02_141500.db`
- `backups/ph_holdings_before_customer_cid_reseed_2026_04_02_150911.db`
- `backups/ph_holdings_before_runtime_resync_2026_04_02_154100.db`

---

## Files Modified (High-Signal Set)

This session touched many files. The most important ones were:

- `app.go`
- `database.go`
- `config.go`
- `license_service.go`
- `chat_service.go`
- `butler_ai.go`
- `bank_statement_parser.go`
- `purchase_order_service.go`
- `offer_pdf_service.go`
- `purchase_order_pdf_service.go`
- `onedrive_import_service.go`
- `workflow_regression_test.go`
- `manual_deployment_package_test.go`
- `manual_customer_reference_seed_test.go`
- `customer_reference_seed.go`

Frontend high-signal files:

- `frontend/src/lib/screens/DashboardScreen.svelte`
- `frontend/src/lib/screens/FinanceHub.svelte`
- `frontend/src/lib/screens/BankReconciliationScreen.svelte`
- `frontend/src/lib/screens/SupplierInvoicesScreen.svelte`
- `frontend/src/lib/screens/SupplierPaymentsScreen.svelte`
- `frontend/src/lib/screens/PaymentsScreen.svelte`
- `frontend/src/lib/screens/InvoicesScreen.svelte`
- `frontend/src/lib/screens/OpportunitiesScreen.svelte`
- `frontend/src/lib/screens/CostingSheetScreen.svelte`
- `frontend/src/lib/screens/OperationsHub.svelte`
- `frontend/src/lib/screens/ButlerScreen.svelte`
- `frontend/src/lib/components/QuickCaptureModal.svelte`
- `frontend/src/lib/components/OpportunityDetail.svelte`
- `frontend/wailsjs/go/main/App.js`
- `frontend/wailsjs/go/main/App.d.ts`

Deleted:

- `frontend/src/lib/screens/SerialTraceScreen.svelte`

---

## Verification Performed

### Passed

- `go build ./...`
- `npm run build`
- targeted Go workflow regressions
- targeted commercial / finance / persistence regressions
- Windows `wails build`

### Browser / UI Verification

- Focused browser smoke was rerun late in the session.
- Result: `5/6` passed against a live local app.
- The one failure was a brittle test selector against the expanded supplier-invoice modal, not evidence that the original supplier-invoice bug had returned.

### Data Checks

Verified live DB:

- `IMP-*` orders: `0`
- synthetic summary rows in `offer_items`: `0`
- synthetic summary rows in `order_items`: `0`

---

## Remaining Caveats

1. Mac packaging is still not fully closed.
Fresh Mac `wails build` still hits the existing linker issue:
- `ld: unknown file type in .../000000.o`

2. PDF output was functionally improved and code-checked, but not re-verified against every sample PDF in a fresh manual export pass during the final Butler work.

3. Customer CID seeding improved the customer master significantly, but legacy duplicates still exist and a future dedupe/merge pass would still be valuable.

4. Butler is now materially more capable and confirmation-gated, but the long-term target of “everything a manual user can do” should still be treated as an expanding capability set rather than a mathematically complete endpoint.

---

## Recommended Next Session Start

1. Fix the Mac `wails build` linker issue and produce a fresh Mac app/package.
2. Run one manual UAT sweep on:
   - supplier invoice edit,
   - supplier PO create,
   - bank statement import + replace flow,
   - opportunity OCR create,
   - Butler create-contact and create-opportunity actions.
3. If deployment is imminent, refresh package artifacts from the latest DB after the Mac build is restored.
4. Optional cleanup pass:
   - customer dedupe/merge for remaining legacy CID variants,
   - stronger finance MoM / forecast intelligence if requested,
   - broader Butler action coverage.

---

## Current Status

The original regression/stabilization plan is complete in code.

The largest unfinished item from the original roadmap, Butler execution behavior, is now closed in code and verified by build.

The main remaining blocker to a perfectly clean deployment handoff is fresh Mac packaging, not the business logic or workflow fixes themselves.
