# Wave 9 Spec-07 — Tight Ship 2 — Status Report

**Branch:** `feat/fable-wave9-7-tight-ship-2` off `main` (`5eb7e14`). **Not merged, not pushed, not tagged** — left for owner review.
**Operating model:** Opus 4.8 orchestrator as senior designer/tech-lead; Sonnet 5 coders in file-disjoint batches; Phase-A recon first; constitutional review of every diff; central gating + bindings regen.
**Gate baseline (start):** vite clean · svelte-check 0 errors / 14 warnings · `go build`/`go vet` clean · full `go test` green.

---

## 0. Headline

Every item in the Spec-06 report-only backlog was closed. The five scoped financial authorizations (B1–B5) were implemented exactly as ratified — nothing else financial moved. Two latent correctness defects that the coders' first-pass diffs would have shipped were caught in constitutional review and fixed centrally (B4 legacy-VAT reconstruction; B1 terminal-state guard regression). B7(d) was escalated to the owner mid-wave and implemented per the ratified choice (two-step archive via the approvals queue).

**B1 dashboard aggregate movement (the point of the wave), from a seeded dev DB in `TestStageMigration_AggregateBeforeAfter`:**

```
BEFORE  WinRate = 50.00%   PipelineValueBHD = 43000.000
AFTER   WinRate = 60.00%   PipelineValueBHD =  8000.000
```

Three mis-tagged terminal-legacy opportunities ("Order Placed" 10k, "Closed (Payment)" 20k, "Closed (Lost)" 5k = **35,000 BHD**) were being counted as *active pipeline*. Canonicalization drops PipelineValueBHD 43,000 → 8,000 (only genuinely-active rows remain) and corrects WinRate 50% → 60% (the two newly-recognized Won rows enter the closed denominator). The mis-tagged rows were lying; now they don't.

---

## 1. Phase A — recon verdicts

| # | Question | Verdict |
|---|---|---|
| **A1** | Stage vocabularies / allowlists / importers / prediction literals / MarkOfferWon-Lost | **CONFIRMED.** 4 overlapping vocabularies (`RFQData.Stage` legacy-9, `Opportunity.Stage` canonical-7, `Offer.Stage` DB-CHECK-5, dormant Cap'n Proto enum). Validation = 5 mutually-inconsistent allowlists (`UpdateRFQStage`, `UpdateRFQ`, `validateOpportunityStageValue` — missing `Expired`, Offer DB-CHECK, frontend display) + `UpdateOpportunity` field-whitelist with NO value check + 2 unvalidated importers (OneDrive, SSOT). `app_prediction_dashboard.go` branches on literals `"Won"`/`"Lost"`/`"Expired"` for WinRate + PipelineValueBHD. `MarkOfferWon`/`MarkOfferLost` update `RFQData.Status` but not `.Stage`. No committed dev DB (`ph_holdings.db` created at runtime; tables `opportunities`, `rfq_data`). Migration infra = `runCustomMigrations()` idempotent pattern. |
| **A2** | Stock-adjustment double-post | **CONFIRMED.** `CreateStockAdjustment` (`app_accounting_inventory.go:1460`) AND `ApproveStockAdjustment` (`:1514`) both call `RecordStockMovement` → a create→approve sequence posts TWO movements and applies the variance twice. Adjustment already has `Status` (`Pending`/`Approved`). Butler reaches it only via the frontend bridge (no Go Butler caller). **Movements carry no `reference_id` back to the adjustment**, so historical doubles are only heuristically countable (see B2). |
| **A3** | Expense SoD vs supplier precedent | **CONFIRMED.** `ApproveSupplierInvoice` (`supplier_invoice_service.go:380`) has the `CreatedBy==""`/`CreatedBy==approver` guard with `segregation of duties:` phrasing; identity server-resolved via `getCurrentUserID()`. `transitionExpenseEntry` (`expense_service.go:848`) has the actor (`getExpenseActorID`) and `entry.CreatedBy` in scope but NO SoD check; the function is shared across submit/approve/reject, so the guard must key on `nextStatus=="approved"`. Frontend already surfaces backend errors via `toast.danger`. No negative SoD test existed for either flow. |
| **A4** | `\|\| 10` VAT fallback sites | **CONFIRMED.** Frontend: `InvoicesScreen.svelte:414,436`; `CostingSheetScreen.svelte:686,1385`. Go analogs (`if x==0 { x=10 }`): `credit_note_service.go:97`, `einvoice_service.go:76`, `invoice_pdf_service.go:423`, `app_costing_exports_surface.go:357` and `:916`. Reference idiom that preserves genuine 0 AND reconstructs legacy rate: `customer_invoice_service.go:1592`. Default-when-absent = 10 (overlay). Test template: `TestUpdateCustomerInvoice_PreservesZeroRatedVAT`. |
| **A5** | Goods-receipt restoration surface | **CONFIRMED, with a critical nuance.** Stock posts ONLY in `CompleteGRN` (`grn_service.go:448`) via `reconcileInventoryReceipt` under a `SELECT … FOR UPDATE` + `CompletedAt`/`grnHasPostedMovement` idempotency guard; serials mint in `ReceiveAgainstPOWithSerials` via `assignSerialsToGRN`. `ReceiveAgainstPO` creates a PENDING GRN only. So the receive action must chain **create → complete** or it reproduces the cosmetic PO-status flip. Backend fully intact/bound; only the UI trigger (retired `GRNScreen`) was severed. PO status-driven button pattern lives at `PurchaseOrdersScreen.svelte:1453`. |
| **A6** | Payroll comp-profile clobber | **CONFIRMED. Fix = refuse cross-division clobber (option b).** `CompensationProfile` is `uniqueIndex` on `employee_id` ALONE (`pkg/finance/payroll/models.go:25`); `UpsertProfile` (`service.go:95`) matches on `employee_id` only and overwrites `division` in place → switching companies + save silently MOVES the single profile across divisions. Model + payroll consumption (`GenerateRun` reads one profile per employee) intend one-per-employee; scoping (option a) has no supporting intent and forces a composite-index migration on golden-test tables. Refusal guard leaves all money math untouched. |
| **A7** | Small-fry anchors | **CONFIRMED (all 8).** (a) `OrdersScreen.svelte` `saveOrder` create = `CreateOrder`+`UpdateOrder` non-atomic. (b) `DeliveryTrackingScreen.svelte` FULLY ORPHANED (not in nav, zero importers) + swapped `CreateShipment` args + nil-deref panic (`app_order_customer_surface.go:875`). (c) GRN-discrepancy stub inputs (`GRNScreen.svelte:802-822`) captured then dropped; backend `RaiseGRNDiscrepancy` exists but unwired; screen is retired. (d) `RequestEmployeeArchive` auto-approves (born `Status="approved"`); `ReviewEmployeeArchiveRequest` two-step path is dead. (e) WorkHub project-admin buttons render for all roles (server rejects correctly). (f) `CreateDNWithSerials` `cleanupDN` deletes only the DN header, orphaning `DeliveryNoteItem` rows. (g) `getHardwareID()` never persists — timeout variance could flip key material between boots. (h) empty catches at `QuotationScreen.svelte:99`, `UserManagementScreen.svelte:107`. |

---

## 2. Phase B — per-item status

### B1 — Stage-vocabulary consolidation, THE FULL FIX — ✅ SHIPPED
- **Canonical enum (single source of truth)**, `stage_vocabulary.go`: `New, Qualified, Proposal, Quoted, Won, Lost, Expired, On Hold`. Terminal = `Won/Lost/Expired`.
- **All Opportunity/RFQ write paths unified** to `canonicalizeOpportunityStage` + `isCanonicalOpportunityStage`: `UpdateRFQStage`, `UpdateRFQ` (status→stage sync), `validateOpportunityStageValue` (now includes `Expired`), `UpdateOpportunityCommercialFields` (added the missing value check). Both importers gated: OneDrive coerces+logs; SSOT carries a package-local copy (pkg/data can't import main) that coerces+logs. Interactive paths **reject** non-canonical; importers **coerce to "New" + log** (never persist a non-canonical value, never abort an import).
- **Idempotent historical migration** `migrateOpportunityStageVocabulary()` registered in `runCustomMigrations()`: rewrites `opportunities.stage` + `rfq_data.stage` per the ratified map, logs per-mapping row counts and any unmapped residuals, safe to run every boot (0 rows on the 2nd run — tested).
- **`MarkOfferWon`/`MarkOfferLost` now update `RFQData.Stage`** ("Won"/"Lost") alongside Status — kills the stale-Stage pipeline deflation.
- **`displayStage()` retained** untouched as the frontend safety net.
- **AC met:** non-canonical writes rejected everywhere incl. imports; migration idempotent; before→after aggregates printed (§0); win-rate no longer deflated by stale Stage.
- **Central fix during review (see §3):** the `UpdateRFQStage` state-machine terminal guard still compared against dead legacy literals — fixed to canonical values + added regression tests.

### B2 — Stock-adjustment double-post — ✅ SHIPPED
- `CreateStockAdjustment` no longer posts a movement (persists `Pending` only); `ApproveStockAdjustment` is the sole poster (Article III — posting at the authorization moment) and now stamps `ReferenceType="StockAdjustment"` / `ReferenceID` for provenance going forward.
- Regression tests: create alone = **0** movements; create→approve = **exactly 1**, variance applied once.
- **Historical count (report-only, untouched):** there is **no committed dev/production DB in the repo to count against**, and — critically — historical adjustment movements carry **no `reference_id`** linking back to their adjustment (only fixed forward, from this wave), so exact counting is impossible; only the heuristic query below can approximate it. On a fresh DB the count is 0. The owner should run this against the live DB:
  ```sql
  SELECT inventory_item_id, quantity, direction, COUNT(*) AS n
  FROM stock_movements
  WHERE movement_type = 'Adjustment' AND deleted_at IS NULL
  GROUP BY inventory_item_id, quantity, direction
  HAVING COUNT(*) > 1;
  ```
  Repair of any doubles found remains a separate owner decision (data surgery on stock history) — untouched this wave.

### B3 — Expense SoD — ✅ SHIPPED
- SoD guard in `transitionExpenseEntry`, keyed strictly on `nextStatus=="approved"` (submit/reject unaffected), mirroring `ApproveSupplierInvoice` exactly (same `segregation of duties:` phrasing, same server-resolved identity). Empty-creator and self-approval both refused.
- New `expense_sod_test.go`: self-approval rejected; distinct approver succeeds; self-submit/reject still allowed.
- **Collateral (expected, fixed centrally):** `TestExpenseLifecycleAndCashFlowProjection_…` self-approved as one admin — the guard now (correctly) rejects it. Updated to stamp a distinct creator so the admin is an arms-length approver. This is the guard biting, not a false positive.

### B4 — `\|\| 10` VAT fallback — ✅ SHIPPED (hardened in review)
- Frontend (0 preserved): `InvoicesScreen.svelte` gains a `vatOrDefault()` helper (null/undefined/''/NaN → 10, explicit 0 kept); `CostingSheetScreen.svelte` reuses the existing `toFiniteNumber` idiom at both draft-restore sites.
- Go derived documents: `credit_note_service.go`, `einvoice_service.go`, `invoice_pdf_service.go` no longer coerce a stored 0 up to 10.
- **Central hardening (see §3):** the coder's first pass removed the coercion outright, which would render/credit **legacy invoices** (`VATBHD>0` but `VATPercent=0`, column default) at **0%** — under-reporting VAT. Added a shared `effectiveInvoiceVATPercent()` helper (mirrors `customer_invoice_service.go:1592`) used at all three sites: genuine zero-rated (`VATBHD==0`) → 0; legacy-absent (`VATBHD>0`) → rate reconstructed from `VATBHD/SubtotalBHD`. New tests cover zero-rated preserved, non-zero unaffected, ZATCA XML 0%, and legacy reconstruction → 10%.
- **One documented coder judgment kept:** `app_costing_exports_surface.go:357` (`CreateOfferDraftFromButler`) keeps the `==0 → 10` default because `req.VatRate` is a non-pointer field on an AI-driven creation path where absent-vs-explicit-0 is structurally indistinguishable; silently emitting a 0% offer when Butler omits the field is the worse failure. `:916` (display-only, already guarded by `&& VAT>0`) left as-is. Hardcoded-10 creation paths left out of scope (creating a 0% invoice is a separate feature).

### B5 — Lean goods receipt (PO-flow action) — ✅ SHIPPED
- New atomic Go wrappers `ReceiveAndCompletePO` / `ReceiveAndCompletePOWithSerials` (`grn_service.go`): chain the existing create → `CompleteGRN`, rolling back the pending GRN (via `DeleteGRN`, which also releases claimed serials) if completion fails so nothing is stranded. **No new posting/serial logic** — routes through the intact `reconcileInventoryReceipt` / `assignSerialsToGRN` and the 9.5 FOR-UPDATE + 9.6 CompletedAt guard.
- `PurchaseOrdersScreen.svelte`: a status-driven **"Receive Items"** action (Sent/Acknowledged/Partially Received) opens a focused `WabiModal` panel scoped to the PO's lines — receive-now qty (keep-list: defaults to and caps at remaining; fully-received lines excluded), optional rejected qty, optional serial entry. Submit picks the serials variant when any serials are entered. Deprecated `GRNScreen` stays retired.
- Tests: partial receive posts stock + `Partially Received`, remainder → `Received`; idempotent (no double-post); serials minted `Available`.
- **Serial-flag nuance:** PO line payloads don't carry `RequiresSerialTracking`, and the spec forbade new plumbing — serial entry is optional per line (graceful degradation), with the backend enforcing `len(serials)==qty`. Wiring the per-line serial flag through PO detail is noted as residue.

### B6 — Payroll comp-profile clobber guard — ✅ SHIPPED
- Refusal guard in `UpsertProfile` (between lookup and update): if an existing row has a non-empty division and the incoming division differs → clear error, no overwrite. Same-division edits and first-time/legacy-empty sets still pass. **Zero** changes to the state machine / money math — payroll golden tests byte-identical.
- Frontend already surfaces the error via `toast.danger` (unchanged). New `clobber_guard_test.go`: cross-division refused (row untouched); same-division edit succeeds; legacy-empty first-set succeeds.

### B7 — Small fry — ✅ SHIPPED (all a–h)
- **(a)** New atomic `CreateOrderWithItems` (App + CRMService) inserts header + items in one transaction; `OrdersScreen` rewired off the CreateOrder+UpdateOrder pair. Rollback-on-item-failure tested. **Bonus latent bug fixed:** `CreateShipment` initial status was `"Packed"`, which violates the `chk_shipments_status` CHECK — every prior call failed at the DB; corrected to `"Pending"`.
- **(b)** `DeliveryTrackingScreen.svelte` **retired (deleted)** — verified zero importers, fully orphaned (Spec-01 InboxScreen precedent). Backend `CreateShipment` nil-deref fixed (guarded absent/unparseable date → zero time, no panic) so the still-bound method is safe.
- **(c)** Removed the dead reject-qty/rejection-reason stub inputs from the retired `GRNScreen.svelte` (a control that does nothing is a lie). Backend `RaiseGRNDiscrepancy`/`ResolveGRNDiscrepancy` remain available for future wiring into the new receive panel (noted as residue).
- **(d)** **Owner-escalated and ratified → made the review real.** See §4.
- **(e)** WorkHub project-admin buttons gated on `can('projects:update')` / `can('projects:delete')` (server enforcement already correct); no-permission users see an honest hint instead of dead buttons.
- **(f)** `CreateDNWithSerials` `cleanupDN` now also deletes the orphaned `DeliveryNoteItem` rows (both deletes run + log independently). Closes the 9.6 C4 observation.
- **(g)** Hardware-ID persistence: resolved ID written once to a **plaintext sidecar** (`<dbDir>/.hardware_id`, 0o600) and preferred on all later boots, so timeout variance can never flip key material. Plaintext is required (it derives the key — an encrypted store would deadlock). Key-derivation formula untouched — only the input is stabilized. Tested via an unexported path-override seam. **Security note in §4.**
- **(h)** Both empty catches now `console.error` + a toast (`QuotationScreen` warning; `UserManagementScreen` danger).

---

## 3. Defects caught in constitutional review and fixed centrally

Two of the coders' first-pass diffs were correct-looking but would have shipped real regressions. Both were caught reviewing the actual diffs against recon and fixed before gating:

1. **B1 — terminal-state guard regression (financial).** The coder canonicalized the input and the validator but left `UpdateRFQStage`'s state-machine enforcement comparing `currentRFQ.Stage` against dead legacy literals (`"Closed (Lost)"`, `"Closed (Payment)"`, …). After canonicalization + migration, stored stages are `"Lost"`/`"Won"`, so the terminal "revenue inflation guard" would **never fire** on migrated rows. Their added test used the *legacy* stored string, so it passed green and hid the gap. Fixed: canonicalize `currentRFQ.Stage` before the invariant checks (robust to unmigrated rows too), rewrote guards to canonical `Lost`/`Won`; added regression tests asserting the guard fires for canonical `"Lost"` and `"Won"` stored values.

2. **B4 — legacy-VAT under-reporting.** The coder removed the `==0→10` coercion outright; on a mature DB, legacy invoices carry `VATBHD>0` with `VATPercent=0` (column default) and would then render/credit/e-file at **0%**. Fixed with the shared `effectiveInvoiceVATPercent()` helper (reconstruct rate from `VATBHD/Subtotal` when the stored percent is absent; treat as genuine 0 only when `VATBHD==0`) + a legacy-reconstruction test.

---

## 4. Decisions, deviations & owner notes

- **Owner question raised & ratified mid-wave — B7(d) employee-archive.** Escalated the genuine policy fork; owner chose **"Make review real (2-step via ApprovalsQueue)."** Implemented: `RequestEmployeeArchive` now creates a genuinely `pending` request (no self-approve, no immediate cascade) that surfaces in the WorkHub approvals queue; archiving + the access-link/project-membership cascade happen only on `ReviewEmployeeArchiveRequest` approval. Per the ratified option, there is **no hard SoD block** on the reviewer (a single-admin shop approves from the queue — a visible, deferred act rather than a silent one), so single-admin deployments are never stranded. `PeopleHub` success toast updated to "submitted for approval". Existing archive test refactored to the two-step flow (request→pending→review→archive+cascade).
- **Deviation flagged (Article VII) — B1 Offer.Stage scoping.** B1 unifies the **Opportunity + RFQ** vocabularies (the financial surface the dashboard reads). **`Offer.Stage` intentionally keeps its own DB-CHECK vocabulary** (`RFQ/Quoted/Won/Lost/Expired`) — it's a separate entity, already DB-enforced, and uses `"RFQ"` not `"New"`; unifying it would require a CHECK-constraint migration and touch MarkOfferWon internals with no financial benefit. The dormant Cap'n Proto enum is left untouched. Flagged for ratification.
- **Security note — B7(g) hardware-ID sidecar.** Persisting the ID plaintext next to the DB means a stolen DB *directory* now also carries the key-derivation input; previously decryption also required the original hardware to re-resolve the ID. This is within the spec's explicit authorization ("persist … settings/local store") and the sidecar must be plaintext (it derives the key). Recommend a future hardening to an OS keystore (Windows DPAPI / macOS Keychain) that binds the value to the machine/user rather than the DB directory.
- **B4 Butler offer-draft default kept at 10%** (`app_costing_exports_surface.go:357`) — documented judgment on an AI-creation path with an indistinguishable absent-vs-0 field (see B4).
- **Pre-existing inconsistency noted (not fixed — out of scope):** `app.go:1279` runs `addColumnIfNotExists("rfq_datas", …)` (plural) while the live GORM table for `RFQData` is `rfq_data` (singular, per `db_sync_service.go`/`phimport`/`phreconcile` and the runtime SQL). B1's migration correctly targets the live table via `a.db.Model(&RFQData{})`; the `rfq_datas` column-add is a latent no-op unrelated to this wave.

---

## 5. Keep-list attestation (audit §4)

All touched screens preserve their §4 keep-list behaviors:
- **Sales:** MarkOfferWon PO capture, revision model, status-driven CTAs — untouched; only added the missing `RFQData.Stage` sync. `displayStage()` safety net retained.
- **Inventory:** GRN remaining-qty defaulting/capping and PO→receive pre-selecting row action **replicated** in the new PO receive panel; serial trace, `PageLayout embedded` single-create pattern intact; the existing FOR-UPDATE + CompletedAt guard is reused, not bypassed.
- **People:** payroll status-driven buttons, archive safety contract (now stronger — review-gated), company/division scoping, race guards — intact; payroll math byte-identical.
- **Work:** ContextTaskModal, task-delete two-press, Team Board — untouched; the ApprovalsQueue now genuinely carries employee-archive tasks.
- **AR/AP & recon:** gated Match→Approve→Settle, posted/paid locks, `matchesCompany` — untouched; expense SoD strengthened.
- **CRM/shell:** role-adaptive gating extended to WorkHub admin buttons; guarded soft-delete surfacing intact.

No native `confirm/alert/prompt`, no screen-local dialog forks, no raw-hex introductions, no RBAC widening (WorkHub gating is strictly tighter; server enforcement unchanged). Identity/SoD resolves server-side throughout.

---

## 6. Residue / follow-ups

- **B2 historical stock-movement repair** — count is deployment-specific + heuristic-only (no historical `reference_id`); repair deferred to the owner.
- **B5 per-line serial flag** — `RequiresSerialTracking` isn't in the PO line payload; the receive panel degrades to optional serial entry. Threading the flag through PO detail (and optionally wiring `RaiseGRNDiscrepancy` from the panel — B7c) is a clean, cheap follow-up.
- **B1** — `Offer.Stage`/Cap'n Proto unification (if ever wanted); the `app.go:1279 rfq_datas` no-op column-add cleanup.
- **Explicitly deferred per spec (untouched):** GL opening-balance carry-forward; visa/permit; allocation enforcement; CRLF normalization; WorkHub/CustomerDetailView decomposition; activity-monitoring relocation.
- **Pre-existing, non-wave:** `TestFileWatcher_HandlerError` timing flake (de-flaked; a failure is now real); staticcheck `interface{}→any` / `QF1012` hints across the tree (informational, not gate).

---

## 7. Owner questions

1. **B1 Offer.Stage scoping** — ratify leaving `Offer.Stage` on its own DB-CHECK vocabulary (recommended), or schedule full unification?
2. **B7(g) hardware-ID** — accept the plaintext sidecar for now, or prioritize an OS-keystore (DPAPI/Keychain) hardening next?
3. **B2 historical doubles** — do you want the heuristic run against the live PH DB and a repair plan drafted, or leave stock history as-is?

---

## 8. Gate

- `go build ./...` clean · `go vet ./...` clean.
- `svelte-check`: **0 errors / 14 warnings** (baseline; 652 files after `DeliveryTrackingScreen` retirement).
- `npx vite build`: clean (`dist/index.html` restored post-build per standing lesson).
- Wails bindings regenerated centrally (`wails generate module`) for the 3 new bound methods.
- Full `go test ./... -count=1 -timeout 1800s`: **GREEN** — exit 0, every package `ok`, 0 failures / 0 panics (the main `ph_holdings_app` package runs >600s under load, hence the 1800s timeout). Independently confirmed by a second full run.

*Severity honesty is law: an accurate red beats a false green. Two real regressions were caught in review and are reported above rather than buried.*
