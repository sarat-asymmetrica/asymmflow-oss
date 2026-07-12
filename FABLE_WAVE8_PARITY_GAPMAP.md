# Wave 8 — PH Parity Gap Map (recon v1)

**Goal:** full feature + flow parity between deployed **ph_holdings** and the sovereign
**asymmflow-oss** substrate, so PH stops living on redeployments.

**Method (ground truth, not artifacts):** diffed the **Wails bound-method surface** — the
exported methods on the structs listed in each `main.go` `Bind:` block — parsed directly
from Go source, *not* from `frontend/wailsjs/go/**/*.d.ts`.

> ⚠️ Why not the generated `.d.ts`? OSS's generated bindings are **stale** — they predate the
> Mission I merge, so a `.d.ts` diff reports ~13 phantom gaps (CreateOfferRevision, RenewOffer,
> AttachCostingSheetFile, GetInvoicesByAgingBucket, …) that already exist in Go. **Regenerate
> bindings (`wails generate module`) before shipping**, but never audit parity from them.

- ph_holdings binds **one** struct (`*App`) → **897** exported methods.
- OSS binds **`*App` + 6 services** (Finance, CRM, Butler, Documents, Sync, Infra) → **881** exported methods.
- **42** methods exist in ph_holdings and are absent from the OSS surface by name.
- **26** methods are OSS-only (sovereign additions — not parity debt).

## Scope caveats (what this recon does NOT yet cover)

1. **Divergence is invisible to a name diff.** A method present in both by the same name can
   differ in signature, RBAC gate, or behavior. Mission I already found several deployed-PH
   behaviors that were *wrong* (UpdateOrder fulfillment wipe). Phase 2 = behavioral parity pass.
2. **Frontend route/view parity is a separate axis.** A bound method can exist with no UI wiring
   it, and vice versa. Phase 3 = `frontend/src` route + view inventory diff.
3. This is the **backend RPC surface** only.

---

## Bucket A — Intentionally excluded (do NOT build) — 12

Deployment / OTA-migration / remote-schema plumbing. This is the *rented-infrastructure* model
the convergence exists to escape. Sovereign builds do bulk data ops via **CLI tooling**
(`phimport.exe` / `phreconcile.exe`, already in-tree), not in-app admin RPCs pushed to deployments.

| Method | Source (ph_holdings) | Disposition |
|---|---|---|
| MigrateRemoteSchema | sync_coverage_service.go | Exclude — remote-schema push |
| GetSyncCoverageReport | sync_coverage_service.go | Exclude (or reframe as local diagnostic) |
| PromoteSprint4Shadow | sprint4_rebuild.go | Exclude — CLI shadow flow covers it |
| RebuildMasterDataIntoShadow | sprint4_rebuild.go | Exclude — CLI shadow flow covers it |
| ApplyDataUpdateManifest | data_update_service.go | Exclude — OTA data-update machinery |
| ApplyPendingDataUpdatesNow | data_update_service.go | Exclude |
| ValidateDataUpdateManifest | data_update_service.go | Exclude |
| ListPendingDataUpdateManifests | data_update_service.go | Exclude |
| EnsureAdminDataUpdateFoundation | data_update_service.go | Exclude |
| ApplyCommercialDeploymentCleanup | deployment_cleanup_service.go | Exclude |
| PreviewCommercialDeploymentCleanup | deployment_cleanup_service.go | Exclude |
| GetBackendHealth | (health) | Verify — OSS has `pkg/infra/health`; may already be exposed differently |

## Bucket B — Real gap: Accounting / finance flows — 10  ⭐ highest value — ✅ CLOSED (P3 slices 2 + 4)

The largest coherent missing flow: **customer receipts + ledgers + operational statements.**
Confirmed absent (OSS's 3 "Receipt" methods are inventory/notification, unrelated).

> ✅ **All 10 ported.** Slice 2 (`006f5ef`) delivered the 4 receipt methods; slice 4
> (`2efc2e7`) delivered the 6 ledger/statement methods below. Bucket B is complete.

| Method | Flow |
|---|---|
| CreateCustomerReceipt | AR: receive customer payment |
| ApplyCustomerReceiptToInvoice | AR: allocate receipt → invoice(s) |
| ListCustomerReceipts | AR: receipts list |
| GetCustomerReceiptAllocations | AR: allocation detail |
| GetCustomerLedger | Statement of account (customer) |
| GetSupplierLedger | Statement of account (supplier) |
| GetExpenseLedger | Expense ledger |
| GetOperationalLedger | GL view |
| GetOperationalBalanceSheet | Financial statement |
| GetOperationalProfitLoss | Financial statement |

## Bucket C — Real gap: Dashboards / operational reports — 4 — ✅ CLOSED (`04907b6`, 2026-07-10)

| Method | Flow |
|---|---|
| GetDashboardARAgingReportYTD | ✅ ported — ports PH's buildARAgingReport refactor, which also upgrades GetARAgingReport with collectibility normalization (stale Draft/Cancelled outstanding no longer inflates buckets) |
| GetDashboardPipelineByStageYTD | ✅ ported — activity-year stage totals over normalized/deduped opportunities |
| GetInventoryMovementsWorkspace | ✅ ported — recent movements feed via inventory:view-gated GetStockMovements |
| GetInventoryPendingFulfillmentReport | ✅ ported — pending/available/shortage per order line with DN fallback |

## Bucket D — Real gap: Sales / procurement flows — 5 — ✅ CLOSED (`69811b7`, 2026-07-10)

| Method | Flow |
|---|---|
| CreatePOsFromOrder | ✅ ported — one draft PO per supplier (supplierlink resolution, deterministic ordering, hard error on unresolvable lines) |
| PreviewOrderDeleteCascade | ✅ ported — exact dependent-record counts + payment block; DeleteOrder refactored onto the same snapshot (widened legacy link matching + in-tx payment re-check, TOCTOU fix) |
| ImportTenderFolders | ✅ ported — idempotent T-&lt;n&gt; RFQ import |
| PreviewTenderFolders | ✅ ported |
| GetPreparedByOptions | ✅ ported — sovereign divergence: PH's hardcoded real staff names replaced by overlay signature blocks (synthetic canon; no-real-people invariant) |

## Bucket E — Real gap: HR / employee archive — 2

Already flagged as a **live OSS gap** in Mission I (PC-D20 deferred model
`employee_archive_requests`; archiving today is an ungated field write with no access-link cascade).

| Method | Flow |
|---|---|
| RequestEmployeeArchive | Raise employee-archive request |
| ReviewEmployeeArchiveRequest | Approve/reject archive request |

## Bucket F — Real gap: Data-quality review — 3

| Method | Flow |
|---|---|
| PreviewCustomerDataQuality | Surface customer data issues |
| ReviewDataQualityIssue | Disposition a data-quality issue |
| GetDataQualityReviewHistory | Review audit trail |

## Bucket G — Partial ports (finish the lifecycle) — 6 — ✅ CLOSED (P4 slice 7 + recompute tail 2026-07-10)

Capability partly present; the CRUD/analytics tail is missing.

| Method | Note |
|---|---|
| ArchiveCollaborativeProject | ✅ ported (`6235dbd`, P4 slice 7) |
| DeleteCollaborativeProject | ✅ ported (`6235dbd`) — status-change delete, history survives |
| ShelveCollaborativeProject | ✅ ported (`6235dbd`) |
| UpdateCollaborativeProject | ✅ ported (`6235dbd`) — whitelisted, terminal statuses admin-gated |
| RecomputeAllCustomerAggregates | ✅ ported (`1ad005d`) — idempotent computed-column recompute; internal-entity markers now overlay config (PH hardcodes 'PH'/'PH Trading'); customers:edit |
| RecomputeCustomerPrediction | ✅ ported (`1ad005d`) — payment-history quality gate (nil, not a guessed grade, when no signal); predictions:create |

---

## Suggested Wave 8 build order

1. **Bucket B (finance/AR)** — biggest coherent flow, highest business value; likely blocks real PH cutover.
2. **Bucket E (employee archive)** — already scoped in Mission I deferred specs; small, self-contained, closes a known security gap.
3. **Bucket G (collab + CRM recompute)** — cheap; finishing partially-present flows.
4. **Bucket C + D (dashboards, PO-from-order, tender, data-quality)** — feature-complete the operational surface.
5. **Regenerate Wails bindings**, then **Phase 2 (behavioral divergence pass)** and **Phase 3 (frontend route/view diff)**.

---

## Phase 2 — Divergence audit (shared methods that differ)

855 methods exist in BOTH apps. "Present in both" ≠ "same behavior."

### 2a. Signature divergence (mechanical, complete) — 5 of 855

| Method | Divergence | Verdict |
|---|---|---|
| `CreateInvoiceWithOptions` | ph has extra param `creditOverrideReason string`; OSS's is 3-arg | **OSS-FIXED (corrected)** — OSS did NOT lose the override; it split it into a dedicated `CreateInvoiceWithCreditOverride` method routed through the kernel approval seam with a synchronous durable `CREDIT_LIMIT_OVERRIDE` audit row. ⚠️ Frontend calling the old 4-arg must switch to `CreateInvoiceWithCreditOverride`. |
| `GetInventoryItem` | `itemID string` (ph) vs `itemID uint` (OSS) | ID-type divergence — verify frontend passes numbers; see Phase 2b inventory finding |
| `GetInventoryItems` | `warehouseID *string` vs `*uint` | same cluster |
| `GetStockMovements` | `itemID *string` vs `*uint` | same cluster |
| `UpdateInventoryItem` | `itemID string` vs `uint` | same cluster |

The refactor preserved the RPC contract almost perfectly — only 5/855 signatures drifted, and 4 are the single inventory ID-type decision.

### 2b. Behavioral divergence (same signature, different logic) — IN PROGRESS

Parallel deep-read audits running across: Finance, Sales pipeline, Inventory/fulfillment,
Security/RBAC, CRM/customers. Each classifies findings as **OSS-MISSING** (port to OSS),
**OSS-FIXED** (OSS corrected a deployed-PH bug — document, don't regress), or
**OSS-REGRESSION** (refactor made OSS weaker than deployed — urgent). Results appended on completion.

#### Inventory & Fulfillment (complete)

- 🔴 **OSS-REGRESSION (fix first): inventory `uint` IDs are a latent bug.** `InventoryItem`/`StockMovement`
  use **string UUID PKs in both apps** (`pkg/crm/domain.go:629,653`, `pkg/domain/base.go:13`), and OSS's own
  `RecordStockMovement` treats the ID as a string (`app_accounting_inventory.go:1129-1131`). But
  `GetInventoryItem(itemID uint)`, `GetInventoryItems(*uint)`, `GetStockMovements(*uint)`,
  `UpdateInventoryItem(uint)` (`app_accounting_inventory.go:984,1016,1067,1218`) render `WHERE id = <number>`
  against a UUID column → always no-match. Dormant only because no inventory UI calls them yet (`frontend/src`
  refs are mock-only). **Action: revert the 4 signatures to `string`/`*string`, regenerate Wails bindings.**
  This also *unmasks* the quantity-whitelist fix below.
- 🟢 **OSS-FIXED (keep; deployed PH still buggy — document):**
  - `UpdateGRN` — OSS restores omitted PO/warehouse/receiving fields before `Save()` (`grn_service.go:308-322`); ph blanks them (`ph_holdings/grn_service.go:303`), breaking 3-way-match.
  - `UpdateInventoryItem` — OSS whitelists updatable columns, protecting ledger-owned `quantity_on_hand`/`reserved` (`app_accounting_inventory.go:1078-1093`); ph passes the raw map (`app.go:22463`).
  - `UpdateDeliveryNote` — OSS masks `SignedBy/SignedAt/SignatureImage` on save (`delivery_note_service.go:538-540`); ph lets a pre-dispatch edit forge proof-of-delivery (`delivery_note_service.go:519`).
- Checked & NOT divergent: CreateDeliveryNote, GetDeliveryNote*, PerformThreeWayMatch, CreateGRN, GetGRN*, UpdateGRNQCStatus, RecordStockMovement, calculateStockStatus.

#### Sales Pipeline (complete)

- 🔴 **OSS-MISSING (port to OSS):**
  - `UpdateRFQ` **status allow-list too narrow** — OSS rejects live PH pipeline stages ("RFQ Received", "PO/LOI Received", "Closed (Payment)", …); accepts only {New,In Progress,Quoted,Proposal,Won,Lost,Closed,On Hold}. Breaks on real PH data. Expand to PH's vocabulary. (ph `app.go:6255-6261` vs oss `app_sales_pipeline.go:1381-1384`)
  - `MarkOfferWon` **dropped customer-PO-required guard** — OSS wins offers / creates orders with blank `CustomerPONumber`. Restore the trimmed-PO check. (ph `app.go:9582-9585` vs oss `app_sales_pipeline.go:4872-4874`)
  - `CreateOrder` **lost RES-004 input validation** (order# required/≤100, customer ≤255, status ≤50). Port guards. (ph `app.go:10013-10025` vs oss `app_order_customer_surface.go:232`)
  - `UpdateRFQ` **`Stage` not synced with `Status`** — OSS leaves `Stage` stale. Add `rfq.Stage = updates.Status`. (ph `app.go:6298-6299` vs oss `app_sales_pipeline.go:1421`)
- 🟢 **OSS-FIXED (keep; PH still buggy):** `UpdateOrder` fulfillment-counter wipe + CreatedBy/CreatedAt mass-assign — confirmed deployed PH still resets QuantityShipped/Invoiced on every edit. (ph `app.go:10185-10231` vs oss `app_order_customer_surface.go:391-464`)
- 🟡 **DIVERGENT-UNCLEAR (human decision):** → ✅ both RESOLVED (P5 decisions 2 & 3, 2026-07-10):
  - `MarkOfferWon` auto-Draft-PO → **removed** (`69811b7`); the enforced `CreatePOsFromOrder` flow is ported. (Was: hardcoded `EUR`, `ExchangeRate:0`, no supplier.)
  - `UpdateRFQ`/`DeleteRFQ` RBAC → **keep OSS** (stronger edit gate; delete via admin-approval queue = OSS-FIXED).
- Verified identical: CreateOffer, ConvertOfferToOrder, CreateOfferRevision, RenewOffer (+lineage), MarkOfferLost, UpdateOpportunityStage, UpdateOpportunityCommercialFields.

#### CRM & Customers (complete)

- 🔴 **OSS-MISSING (port to OSS):**
  - `CreateCollaborativeProject` **drops 6 customer-linkage fields** — OSS `Project` model has no CustomerName/EndUserName/OpportunityKey/CustomerPOC{Name,Email,Phone}; project loses all customer/end-user/POC linkage. Add fields to model + method (+ Update when lifecycle ported). (ph `collaboration_service.go:1609-1645` vs oss `:1417-1447`)
  - `AddCollaborativeProjectMember` **omits member notification** — port `createProjectMemberNotification` + wire both create/reactivate branches. (ph `collaboration_service.go:1842,1859` vs oss `:1467-1555`)
  - `ListEmployeeContributionSummaries` **omits opportunity/revenue YTD rollup** (OpportunityYTD/WonYTD/LostYTD/RevenueYTD) — OSS only counts projects/tasks. Port the 4 fields + aggregation loop. (ph `collaboration_service.go:1177-1205` vs oss ~`:1022-1093`)
  - `ListCollaborativeProjects` **filter too narrow (latent)** — OSS excludes only `archived`; PH excludes archived+shelved+deleted. A shelved/deleted project synced from a PH peer renders active in OSS. Widen alongside the shelve/delete port. (ph `:1654-1657` vs oss `:1456-1459`)
- 🟡 **DIVERGENT-UNCLEAR:** `GetCustomerRelatedProducts`/`GetCustomerRelatedSuppliers` — PH Brand×Token vs OSS ProductMaster SKU. → ✅ **RESOLVED (P5 decision 4): keep OSS SKU** — the sovereign data model is deliberate; numeric parity is not a goal. (ph `app.go:11189-11459` vs oss `app_crm_surface.go:160-391`)
- 🟢 **OSS-FIXED (keep):** Customer CRUD (Create/Update/Delete/contacts) faithfully ported into the `crmcustomer` engine; DeleteCustomer child-record safety guard preserved (`pkg/crm/customer/delete.go:29-57`); EnsureCollaborativeFoundation gained a `settings:update` gate (I-11).
- ⚠️ **Cross-domain (HR) — possible OSS-REGRESSION, cross-check with Security audit:** OSS dropped `currentSessionHasAdminRoleOnly()` on `CreateEmployeeProfile`/`UpdateEmployeeProfile`/`SetEmployeeEmploymentState` and the archive-request routing (`EmployeeArchiveRequest` model not migrated). PH enforces admin-only + approval. (Ties to Bucket E: RequestEmployeeArchive/ReviewEmployeeArchiveRequest.)

## Phase 3 — Frontend route/view parity (complete)

Both apps: Svelte + Vite, hand-rolled router in `App.svelte` (`screenLoaders` map), feature
screens reached as **tabs inside Hub screens** (SalesHub/OperationsHub/FinanceHub/CRMHub/PeopleHub/WorkHub).
**Top-level nav: 9/9 parity.** All gaps are Hub-level tabs, not top-level routes.

**10 hub-level feature gaps (deployed → sovereign):**

| Feature | Hub | Sovereign status | Backend gap link |
|---|---|---|---|
| Data Quality review | CRMHub | screen **absent** | Bucket F |
| Serial-number traceability | OperationsHub | screen **absent** | 🆕 not on backend list — investigate |
| Receiving (GRN) | OperationsHub | deprecated/removed | — (GRN backend present) |
| Accounting (balance sheet / P&L) | FinanceHub | screen exists but **orphaned/unwired** | Bucket B |
| Reports & Exports | FinanceHub | screen exists but **orphaned** | partial (report RBAC ported I-27) |
| Book-Bank Reconciliation | FinanceHub | tab **commented out** | 🆕 verify backend |
| Cheque Register | FinanceHub | tab **commented out** | 🆕 verify backend |
| FX Revaluation | FinanceHub | tab **commented out** | 🆕 verify backend |
| Audit Trail Viewer | FinanceHub | tab **commented out** | 🆕 verify backend |
| Employee Archive (+ directory filter, task list, license reassign) | PeopleHub | feature **stripped** | Bucket E |
| Project archive/shelve/delete/update actions | WorkHub | imports omitted | Bucket G |

- **Sovereign-only screens:** CustomersScreen, SuppliersScreen — both **stale dead code** (deployed deleted them; CRM uses Customer360). No genuine sovereign-only feature. Plus an i18n layer (infra).
- **⚠️ Correction to Bucket B (receipts/AR):** deployed AND sovereign both ship the receipts UI — FinanceHub "Receipts"/"Payments Received" tab (`PaymentsScreen`). So customer-payment capability likely exists in OSS under a different method family; the RPC-surface "gap" is narrower than first stated (probably just the receipt-allocation sub-model). **Finance audit to confirm.**
- Deployed `FinancialDashboard` is richer (more aging/pipeline/receivable widgets) than sovereign's shell — polish, not a hard gap.

**🟢 Resolved — 4 commented-out FinanceHub tabs + Serials are FRONTEND-ONLY gaps (backend ready in OSS):**
`Cheque` (18 methods), `BookBank`/`Reconcil` (recon suite), `FXRevaluation` (Calculate/Post/Reverse), `AuditTrail` (GetAuditTrail*), and `Serial` traceability (RegisterSerials, GetAvailableSerials, CreateDNWithSerials, …) all have full bound backends in OSS. These tabs just need **re-wiring/uncommenting in the Svelte Hubs** — no backend build. Cheapest parity wins in the whole audit.

---

#### Finance & Invoicing (complete)

- 🔴 **OSS-MISSING (port to OSS):**
  - **Aggregate money rounding dropped** — OSS stores raw-float VAT/GrandTotal/SupplierCost/GrossMargin; PH wraps them in `roundInvoiceMoney` (MON-005). Sub-fils noise persists AND feeds `computeDocumentHMAC`. Apply `roundInvoiceMoney` to aggregate writes. (ph `customer_invoice_service.go:913-953` vs oss `:954-993`)
  - `MarkSupplierInvoicePaid` **writes no reconciling ledger row** — PH creates a `SupplierPayment` for the remaining balance so `SUM(payments)==total`; OSS only flips `Status/PaymentStatus="Paid"`, understating the payment ledger. Create the reconciling entry. (ph `supplier_invoice_service.go:607-647` vs oss `:626-659`)
  - **Invoice number generated outside the create tx** — OSS calls `GenerateInvoiceNumber()` before `db.Transaction`, so a rollback burns the number → sequence gaps. Move inside the tx (PH uses `generateInvoiceNumberWithTx`). (ph `customer_invoice_service.go:1036-1042` vs oss `:770/1001`)
- 🟡 **DIVERGENT-UNCLEAR — coupled supplier-status set** → ✅ **RESOLVED (P5 decision 1: HYBRID, built 2026-07-10):**
  - `PerformThreeWayMatch` vocabulary: **keep OSS** (`Verified`/`Disputed`/`Review Required` — the internally-consistent redesign stays).
  - `RecordSupplierPayment` eligibility: **tightened to Approved-only** — the explicit `ApproveSupplierInvoice` step (SoD: match ≠ disbursement authority) is now mandatory; Verified is no longer directly payable.
  - Settlement: `applySupplierInvoicePaymentState` (landed in P1 as `supplier_invoice_payment_policy.go`) is now wired into `RecordSupplierPayment`, so `invoice.Status` settles from the ledger. Latent fix: the policy now preserves "Verified"/"Disputed" instead of collapsing them to "Pending".
- 🟢 **OSS-FIXED / at parity (keep):** `CreateInvoiceWithCreditOverride` (kernel-approval seam, durable audit — see 2a correction); customer settlement policy, `RecordPayment`, credit notes, journal post/reverse all byte-identical.

#### Security / RBAC (complete)

Both apps share the same `requirePermission` chokepoint + admin-only delete gate; OSS adds a 30-min
session-inactivity touch. The refactor into `*_surface.go` + `*Guarded` wrappers is where divergences hide.

- 🔴🔴 **OSS-REGRESSION (refactor opened auth holes — FIX FIRST, these are live security bugs not features):**
  | Method | Severity | PH gate → OSS | Fix |
  |---|---|---|---|
  | `ScanOfferFolders` | **HIGH** — frontend-callable filesystem importer (takes `basePath`), fully ungated | `documents:create` → none | add `requirePermission("documents:create")` |
  | `GetInvoiceWithItems` | **HIGH** — unauthenticated financial-data read | `invoices:view` → none | add `requirePermission("invoices:view")` |
  | `ReconcileOfferData` | MED — filesystem reconcile over caller path | `documents:view` → none | add `requirePermission("documents:view")` |
  | `Start/RecordBatch/Heartbeat/End UserActivitySession` | LOW — telemetry | `dashboard:view` → none | gate each |
  | `Start/StopCollaborativeSyncLoop` | LOW | `settings:update` → none | gate (or confirm unbound) |
  | `GeneratePONumber` / `GenerateGRNNumber` | TRIVIAL — sequence read | `po:create`/`grn:create` → none | gate for parity |
  Refs: OSS `excel_costing_parser.go:815`, `query_optimizations.go:324`, `data_reconciliation.go:152`, `user_activity_monitoring.go:427-544`, `collaboration_sync.go:246-291`.
- 🟡 **RESOLVED (CRM lead, downgraded to P2):** All three employee-profile methods ARE permission-gated in OSS (`hr:create`/`hr:update`) — not unauthenticated. Only difference: PH's `CreateEmployeeProfile` has an EXTRA `currentSessionHasAdminRoleOnly()` overlay on top of `hr:create` that OSS dropped. UpdateEmployeeProfile/SetEmployeeEmploymentState are at parity. → weakened defense-in-depth, not an open hole; restore the admin overlay as a P2 item, not P0.
- 🟢 **OSS-FIXED (~19 methods, deployed PH still ungated — document):** EnsureLicenseTableExists, InitializeJobQueue, CancelJob, CleanupOldJobs, SeedAdditionalBankAccounts, OpenExportedFile, Backfill{WonOfferItems,BusinessCustomerIDs}, SetAPIKeys, TestAIConnection, UpdatePOStatus, Generate*Report, Ensure*Foundation.
- ✅ **False alarms cleared:** `DeleteRFQ`/`DeleteRFQWithCascade` (`offers:delete`→`offers:edit` string diff) is NOT a regression — OSS routes through `guardDeleteOrRequest` → admin-only approval queue. Cheque/FX/Serial/License/Banking methods all gated via `*Guarded` wrappers. Also **corrects the Mission I merge-note framing**: deployed PH already gates SeedLicenseKeys/SeedEmployeeKeys etc. — the "frontend-callable license minting" hole was OSS-pre-Mission-I, never deployed PH.
- 🟡 **Permission-string diffs (both gated, OSS neutral-or-better):** GetSettings (`settings:update`→`settings:view`, OSS correct), BackfillInvoiceItemsFromOrders (→`invoices:update`), UpdateRFQ (→`offers:edit`, stronger).

---

## CONSOLIDATED — Wave 8 recommended action order

**Headline:** the refactor is sound — 855/855 shared methods, only 5 signature drifts, and OSS is *ahead* of deployed PH on ~22 integrity/security hardenings. But it opened **6 auth regressions** and shed a handful of PH business-rule guards + field richness. Net remaining work is small and surgical, not a rebuild.

> **✅ P0 + P1 CLOSED (2026-07-09).** Autonomous sprint complete; full root suite green after each. Commits on `main`:
> `88d01f6` P0 gates · `3926b2e` invoice rounding · `eb976be` numbering-in-tx · `d186a60` inventory IDs · `2af38b6` supplier ledger · `82d5204` regenerated bindings.
>
> **✅ P2 CLOSED (2026-07-09).** All 8 business-rule guards restored; full root suite green (159s) after each batch. Commits on `main`:
> `b23c7f4` RFQ status/stage parity + MarkOfferWon PO guard · `94310b7` CreateOrder RES-004 · `1c4e4fe` CreateEmployeeProfile admin overlay · `0d2bae7` 4 collaboration ports (+schema golden) · `7cb714c` regenerated bindings.
>
> **✅ P3 CLOSED (2026-07-10).** All 4 slices done; full root suite green after each (194s, 162s, 234s, 234s). **Buckets B, E, F all closed.**
> - **Slice 1 — employee_archive_requests model + flow (Bucket E) — DONE.** `fd9d950` archive model+flow+tests+golden · `4f6f5cd` bindings.
> - **Slice 2 — customer receipt-allocation sub-model (Bucket B) — DONE.** `006f5ef` receipt models+service+tests+golden (+`finance.Payment.ReceiptID`) · `92daedc` bindings.
> - **Slice 4 — operational financial statements (Bucket B tail) — DONE.** `2efc2e7` ledger/P&L/balance-sheet service+8 tests (read-only, no schema) · `d80d07f` bindings. Fixed a would-be OSS-REGRESSION: swapped PH's hardcoded `'PH Trading'` division default for the config-driven `normalizeDivisionSQL` CASE so blank-division rows scope to the overlay default instead of vanishing. **Closes Bucket B.**
> - **Slice 3 — customer data-quality review ledger (Bucket F) — DONE.** `9ba4944` preview/review/history service+model+5 tests+golden · `82721bb` bindings. Dropped PH's `ensureSyncBaseColumns`-based self-migration for `tradingModels()` + AutoMigrate registration. **Closes Bucket F.**
> - Remaining Wave 8 work: P4 frontend re-wiring, P5 human decisions.
>
> **✅ P4 CLOSED (2026-07-10).** Frontend re-wiring of orphaned/commented screens — all 7 slices done, including unblocking slice 7's backend (Bucket G project methods). Commits on `main`:
> - **Slice 1 — FinanceHub reconciliation tabs — DONE.** `e97b395` — uncommented Cheques/Book-Bank/FX/Audit-Trail into `FinanceHub.svelte` (screens + bindings already built). Frontend `vite build` green; no Go change → no binding regen. **P4 green-gate is `vite build`, not `go test`.**
> - **Slices 2+3 — route orphaned Reports + Accounting screens — DONE.** `1e44634` — both fully-built screens registered nowhere; wired into `screenLoaders` + sidebar navItems with permission gates mirroring their backends (`reports:view` / `finance:view`) + `nav.reports`/`nav.accounting` i18n keys across all 5 locales. Touches Go-embedded JSON (not the binding surface) → still no binding regen. Gates: 5 locale JSON parse + `go test ./pkg/i18n/...` + `vite build`, all green.
> - **Slice 4 — PeopleHub employee-archive — DONE.** `4b6d029` — surfaces the P3 slice-1 backend: `requestEmployeeArchive`/`reviewEmployeeArchiveRequest` wrappers in `collaboration.ts` (bindings live on `*App`, unlike the rest of the file's SyncServiceBinding imports) + 4 archive fields on `EmployeeProfile`; PeopleHub gains the admin-gated archive panel (reason required), archived-state panel, Active/Archive/All directory filter + counts; NotificationsScreen gains admin approve/reject on peer-synced `employee_archive_approval` notifications (PH parity — there is deliberately NO list-pending endpoint; the reviewer path rides the notification feed). Deactivate replaced by the archive flow, Reactivate now admin-only on inactive profiles. Hub-internal labels are plain strings (no i18n touch). `vite build` green.
> - **Slice 5 — CRMHub Data Quality tab — DONE.** `9e31640` — new `DataQualityScreen.svelte` (ported from frozen PH) surfacing the P3 slice-3 backend: KPI strip, issue-type filter + search, review cards (note + reviewed/resolved/dismissed), review history. Issue-type option values verified against `data_quality_service.go`. CRMHub gains the third tab; openIssue routes customer→Customer detail, supplier→Supplier detail, offer/opportunity→Sales hub via `navigateToScreen` (OSS has no CRMHub `navigate` dispatch listener, unlike PH). `vite build` green (CRMHub chunk 85.6KB→93.5KB proves it compiled in).
> - **Slice 6 — OperationsHub Serial traceability — DONE.** `8ebbed8` — `SerialTraceScreen.svelte` ported from PH (read-only PO→GRN→DN→Invoice→Customer lifecycle search on `SearchSerials`, warranty status computed live) + wired as the OperationsHub "Serials" tab. Two sovereign adjustments: local statusColor map (PH's shared util + theme tokens absent here) and results header inside the Card body (OSS Card has no header slot). `vite build` green (OperationsHub chunk 112.9KB→118.4KB).
> - **Slice 7 — WorkHub project lifecycle — DONE (was backend-blocked; unblocked + built).** Backend `6235dbd`: ported PH's whitelisted `UpdateCollaborativeProject` + `Archive/Shelve/DeleteCollaborativeProject` (terminal statuses escalate projects:update→projects:delete = admin wildcard only; delete is a status change, history survives; audit-logged with reason) + SyncServiceBinding delegates + 3 tests; full root suite green (219s). **Closes the 4 project methods of Bucket G** (the 2 CRM recompute methods remain open). Bindings `fc02ec8` (wails generate module). Frontend `5eab663`: 4 collaboration.ts wrappers + WorkHub project hero edit toggle (name/type/description) + "Project Administration" subpanel (Archive/Shelve/Delete with audit reason, terminal projects leave the active list). `vite build` green (WorkHub 42.1KB→45.2KB).

> **✅ WAVE 8 BACKEND SLATE CLOSED (2026-07-10).** After P4, the remaining buckets and P5 decisions all closed in one sprint. Commits on `main`:
> - **Bucket G tail** — `1ad005d` RecomputeAllCustomerAggregates + RecomputeCustomerPrediction (+bindings `7f2190b`, docs `c768069`). **Bucket G closed.**
> - **Bucket C** — `04907b6` dashboard AR-aging/pipeline YTD + inventory pending-fulfillment/movements (+bindings `1582383`, docs `21da87e`). Side upgrade: GetARAgingReport now collectibility-normalized. **Bucket C closed.**
> - **Bucket D + P5-2** — `69811b7` CreatePOsFromOrder / PreviewOrderDeleteCascade (+TOCTOU-safe DeleteOrder refactor) / tender preview+import / GetPreparedByOptions (synthetic-canon names) / MarkOfferWon auto-PO removed (+bindings `b7aedb8`). **Bucket D closed.**
> - **P5-1 hybrid** — supplier controls: Approved-only payment eligibility + settlement policy wired into RecordSupplierPayment + Verified/Disputed vocabulary preserved on hydrate.
> - **P5-3 / P5-4** — keep-OSS decisions recorded (no code change).
>
> **Every real parity gap on this map is now closed.** Remaining known non-goals: Bucket A (deliberately excluded), the 2 stale sovereign-only screens, and the operational-ledger frontend surface (P3 slice 4's GetOperational* methods still have no calling screen — optional future slice).

### P0 — Security regressions (close the holes the refactor opened) — hours ✅ DONE (`88d01f6`)
Add `requirePermission` to the 6 OSS-REGRESSION methods above. Start with `ScanOfferFolders` + `GetInvoiceWithItems`. **These block a safe PH cutover.** (HR employee-profile admin-overlay resolved → P2, not a hole.)
Shipped: gated ScanOfferFolders + BatchImportCostingSheets (a 2nd ungated importer found during verification), GetInvoiceWithItems, ReconcileOfferData, the 4 UserActivitySession methods, GeneratePONumber/GenerateGRNNumber, and Start/StopCollaborativeSyncLoop (gated at the SyncServiceBinding surface — the *App method is shared with startup). **8 gates total** (spec said 6; verification found 2 more).

### P1 — Money & ledger correctness — hours-to-days ✅ DONE (`3926b2e` `eb976be` `d186a60` `2af38b6` `82d5204`)
- ✅ Invoice aggregate rounding (`roundInvoiceMoney` on VAT/GrandTotal/margin — also fixes HMAC drift). `3926b2e`
- ✅ `MarkSupplierInvoicePaid` reconciling ledger row — ported PH's supplier-invoice payment-state policy + transient `OutstandingBHD`. `2af38b6`
- ✅ Invoice numbering inside the create tx (`generateInvoiceNumberWithTx` via `numbering.NextInTx`; rollback no longer leaves a sequence gap). `eb976be`
- ✅ Inventory `uint`→`string` ID revert (5 signatures + wrappers) + regenerated bindings — unblocks inventory UI and unmasks the quantity-whitelist protection. **Also fixed a latent nested-tx deadlock in `ApproveStockAdjustment` the ID revert unmasked.** `d186a60` + `82d5204`

### P2 — Business-rule guards PH has, OSS dropped — days ✅ DONE (`b23c7f4` `94310b7` `1c4e4fe` `0d2bae7` `7cb714c`)
- ✅ **UpdateRFQ status vocabulary + Stage sync** — restored the 10 pipeline-stage names to `validStatuses`, the 7 short status names to `UpdateRFQStage`'s `validStages`, and the dropped `rfq.Stage = updates.Status` sync line. `b23c7f4`
- ✅ **MarkOfferWon customer-PO guard** — restored trim + non-empty guard so an offer can't be won with a blank/whitespace PO. `b23c7f4`
- ✅ **CreateOrder RES-004 validation** — restored trim + bounds on orderNumber (1–100), customerName (1–255), status (≤50) via the structured `newError()` codes. `94310b7`
- ✅ **CreateEmployeeProfile admin overlay** — restored PH's `currentSessionHasAdminRoleOnly()` check on top of `hr:create` (a non-admin holding hr:create could previously mint profiles). `1c4e4fe`
- ✅ **CreateCollaborativeProject 6 linkage fields** — restored CustomerName/EndUserName/OpportunityKey/CustomerPOC{Name,Email,Phone} on the Project model + trim/persist on create; schema golden regenerated (projects +6 cols +opportunity_key index). `0d2bae7`
- ✅ **AddCollaborativeProjectMember notification** — restored the member-assignment notification on both create and update paths (`createProjectMemberNotification` + `currentProjectActorName` helpers). `0d2bae7`
- ✅ **ListEmployeeContributionSummaries rollup** — restored the per-employee YTD opportunity rollup (count/won/lost + won-revenue, name-matched, current-year filtered) + 4 summary fields. `0d2bae7`
- ✅ **ListCollaborativeProjects filter** — restored PH's `LOWER(COALESCE(status,'')) NOT IN ('archived','shelved','deleted')` so the activeOnly filter is case-insensitive, NULL-safe, and excludes all 3 terminal statuses. `0d2bae7`
- Bindings regenerated for the 10 new frontend-facing fields (`7cb714c`).

### P3 — Feature ports (Bucket B/E/F backends) — days-to-weeks · ✅ DONE (all 4 slices; Buckets B, E, F closed)
Sliced for incremental delivery; each slice ships model + service + tests + bindings behind a green suite.
- ✅ **Slice 1 — `employee_archive_requests` model + flow (Bucket E)** — ported PH's `employee_archive_service.go`: `EmployeeArchiveRequest` model, admin-only `RequestEmployeeArchive` (request→archive in one tx) + `ReviewEmployeeArchiveRequest` (approve/reject for peer-synced pending requests), cascading archive (employee `is_active`/`employment_status`/archive metadata + access-link demotion + project-membership close), requester notification. Added 4 archive columns to `Employee`; registered the new table in both the runtime collaborative-migration list and `tradingModels()` (schema golden regenerated: +`employee_archive_requests` table +4 `employees` cols). 7 tests. **Establishes the request→approval scaffold the remaining slices reuse.** Closes the HR archive gate (feeds P4 PeopleHub employee-archive). `fd9d950` + bindings `4f6f5cd`
- ✅ **Slice 2 — customer receipt-allocation sub-model (Bucket B)** — ported PH's `receipt_service.go`: `CustomerReceipt` header + `CustomerReceiptAllocation` link models, `CreateCustomerReceipt` (on-account OR invoice-applied in one tx), `ApplyCustomerReceiptToInvoice` (allocate on-account balance later, auto-fills min(unapplied, outstanding) when amount≤0), `ListCustomerReceipts`, `GetCustomerReceiptAllocations`. Each allocation creates a `Payment` (linked via new `finance.Payment.ReceiptID`) + advances invoice settlement state (`applyCustomerInvoicePaymentState`) + the receipt's applied/unapplied balance. Reuses existing helpers (`roundBHD`, `canRecordCustomerInvoicePayment`, `normalizeDivisionName`); added the 2 tables + `receipt_id` column to `tradingModels()` (schema golden regenerated). Customer + division cross-checks enforced. 7 tests. `006f5ef` + bindings `92daedc`
- ✅ **Slice 3 — customer data-quality review ledger (Bucket F)** — ported PH's `data_quality_reviews` surface: `PreviewCustomerDataQuality` (live scan over customers/opportunities/offers — blank/duplicate customer names via a Bahrain-suffix-aware dedup key, opportunities missing a title or customer link, offers missing a customer — overlaid with existing dispositions, suppressing resolved/dismissed), admin-only `ReviewDataQualityIssue` (upsert keyed on issue_id: reviewed/resolved/dismissed, stamps reviewer + logs audit + emits `data-quality:updated`), `GetDataQualityReviewHistory`. **Sovereign divergence:** PH self-provisions the table at call time via `ensureDataQualityReviewFoundation` (which leans on `ensureSyncBaseColumns`, absent on this substrate); dropped the self-migration and registered `&DataQualityReview{}` in `tradingModels()` so AutoMigrate + the pinned golden own the table (golden regenerated: +`data_quality_reviews` table + its indexes, no other table touched). 5 tests (all issue types flagged, admin-only gate, resolved-suppressed vs reviewed-stays-visible, history, unknown-action + `customers:view` gate). Wires the CRMHub Data Quality screen (P4). `9ba4944` + bindings `82721bb`. **Bucket F closed → P3 COMPLETE (4-of-4).**
- ✅ **Slice 4 — operational financial statements (Bucket B tail)** — ported PH's `operational_accounting_service.go`: `GetOperationalLedger` (unified debit/credit ledger across customer invoices, receipts, invoice payments, supplier invoices, supplier settlements, expenses — chronological, running balance), the three thin views (`GetCustomerLedger`/`GetSupplierLedger`/`GetExpenseLedger`), `GetOperationalProfitLoss` (revenue/COGS/gross-margin/expenses/net-income + per-category breakdown), `GetOperationalBalanceSheet` (cash/AR/customer-credits/AP/expense-liability net position). **Read-only** — aggregates already-migrated transaction tables, so NO new models, migration list, or schema golden change (simpler than slices 1–2). Gated `finance:view`. **Sovereign divergence fixed (would be an OSS-REGRESSION under a verbatim port):** PH hardcodes `'PH Trading'` as both the division-normaliser default and the COALESCE fallback; on this substrate the division set + default are overlay config (default "Acme Instrumentation"), so `applyDivisionFilter` + the customer-payment inline filter now use the config-driven `normalizeDivisionSQL` CASE (the same expression the dashboard/backfill queries use) — without it, blank-division rows normalise to the stale literal and silently vanish from every unfiltered report. 8 tests (incl. a blank-division regression guard). Wires the FinanceHub Accounting screen (P4). `2efc2e7` + bindings `d80d07f`

### P4 — Frontend re-wiring (backend already exists) — hours each, high ROI · 🔄 IN PROGRESS
Uncomment/wire FinanceHub tabs (Book-Bank Recon, Cheques, FX Revaluation, Audit Trail), Serial traceability (OperationsHub), Accounting + Reports screens (orphaned), WorkHub project lifecycle actions, PeopleHub employee-archive.

Landscape mapped 2026-07-10 (which screens exist, their binding surface, wiring cost). Routing = a screen must appear in `App.svelte` `screenLoaders` + `screens[]` + `EnterpriseSidebar.svelte` `navItems`; hub sub-tabs just import + switch on a local `activeTab`. Ranked smallest→largest wiring effort:
1. ✅ **Slice 1 — FinanceHub reconciliation tabs — DONE (`e97b395`).** Cheques / Book-Bank / FX / Audit Trail: four fully-built, fully-bound screens were commented out of `FinanceHub.svelte`; uncommented the 3 blocks (imports, tab defs, render). No Go change → no binding regen. `vite build` green (FinanceHub 211KB→255KB).
2. ✅ **Slice 2 — Reports screen (orphaned) — DONE (`1e44634`).** `ReportsScreen.svelte` (bound to `GetReportData`/`ExportReport`/`GetDashboardStats`) routed via `screenLoaders` + sidebar navItem (`reports:view`, matching its `requireReportAccess` gate) + `nav.reports` in all 5 locales.
3. ✅ **Slice 3 — Accounting screen (orphaned) — DONE (`1e44634`, same commit as slice 2).** `AccountingScreen.svelte` routed the same way (`finance:view`, matching its `GetChartOfAccounts`/`GetTrialBalanceGate` gate) + `nav.accounting` in all 5 locales. ⚠️ **Note:** it calls a *different* accounting set, NOT the six `GetOperational*` methods from P3 slice 4 — those remain called by zero frontend files. Surfacing the operational ledger is a separate medium build (a future slice, if wanted).
4. ✅ **Slice 4 — PeopleHub employee-archive — DONE (`4b6d029`).** Wrappers + admin archive panel + directory Active/Archive filter + notification-feed reviewer actions (see P4 progress block).
5. ✅ **Slice 5 — CRM Data Quality tab — DONE (`9e31640`).** New `DataQualityScreen.svelte` + CRMHub tab + cross-hub openIssue routing (see P4 progress block).
6. ✅ **Slice 6 — Serial traceability (OperationsHub) — DONE (`8ebbed8`).** PH SerialTraceScreen ported + Serials tab (see P4 progress block).
7. ✅ **Slice 7 — WorkHub project lifecycle — DONE (`6235dbd` backend + `fc02ec8` bindings + `5eab663` UI).** Backend block removed by porting the Bucket-G project methods; wrappers + edit toggle + admin actions live (see P4 progress block). **P4 slate complete — all 7 slices closed.** Remaining Wave 8: Bucket G's 2 CRM recompute methods, Bucket C/D backends, P5 human decisions.

### P5 — Human decisions ✅ ALL DECIDED + EXECUTED (Sarat, 2026-07-10)
1. **Supplier-status trio → HYBRID (built).** Keep OSS's status vocabulary (Verified = clean 3-way match), restore PH's segregation-of-duties: `RecordSupplierPayment` now requires the explicit `ApproveSupplierInvoice` step (Approved only — Verified is no longer directly payable), and every payment settles `invoice.Status` through the supplier payment-state policy (fully paid → "Paid", partial stays "Approved"/"Partial"). Also fixed a latent P1-port bug: `supplierInvoiceNonPaymentStatus` now preserves OSS's "Verified"/"Disputed" instead of collapsing them to "Pending" on hydrate.
2. **MarkOfferWon auto-Draft-PO → REMOVED (built, `69811b7`).** The EUR/0-rate/no-supplier draft is gone; PO creation is the deliberate `CreatePOsFromOrder` flow, matching deployed PH.
3. **UpdateRFQ/DeleteRFQ RBAC → KEEP OSS (no code change).** `offers:edit` on update (stronger than PH's dashboard:view) and the soft-delete admin-approval queue on delete (equivalent control, better audit trail). Recorded as OSS-FIXED.
4. **Related-products/suppliers taxonomy → KEEP OSS SKU (no code change).** The ProductMaster SKU catalog IS the sovereign data model (Mission H imported into it); numeric parity with PH's Brand×Token aggregation is intentionally NOT a goal. Recorded as intentional divergence.

### Explicitly DON'T build (Bucket A) — deployment/OTA/migration plumbing
The sovereign model uses CLI tooling (`phimport`/`phreconcile`); do not port the 12 remote-schema/data-update-manifest methods.

---

## Reproduce this diff

```bash
# ph_holdings surface
grep -rhoE "^func \([a-z]+ \*?App\) [A-Z][A-Za-z0-9_]*" ph_holdings/*.go | awk '{print $NF}' | sort -u > ph.txt
# OSS surface (App + 6 bound services)
grep -rhoE "^func \([a-z]+ \*?(App|FinanceService|CRMService|ButlerService|DocumentsService|SyncServiceBinding|InfraService)\) [A-Z][A-Za-z0-9_]*" asymmflow-oss/*.go | awk '{print $NF}' | sort -u > oss.txt
comm -23 ph.txt oss.txt   # parity gaps
```
