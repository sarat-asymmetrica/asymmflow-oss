# FABLE Wave K6 — Consolidated Parity Sign-Off & Flip Checklist

**Branch:** `exp/frontend-kernel` · **Worktree:** `c:\Projects\asymmflow\asymmflow-lab`
**Status:** FLIP-PREP (docs only). Nothing destructive has run. The old `frontend/`
is **untouched**, no `wails build` has been triggered, and nothing has been pushed.
**Purpose:** one place for the owner to sign off screen-by-screen that the kernel
app (`frontend-lab/`) reaches parity with the legacy `frontend/`, and to see exactly
what real-binding wiring (INTEG) remains before graduation.

> ### ★ GAP-CLOSE COMPLETE (2026-07-16) — every real binding is WIRED. `INTEG gap:` count = **0**.
> The INTEG → Residue → Gap-Close campaigns drove ~160 honest `INTEG gap:` throws to **zero**. Every
> `mock (INTEG)` mutation cell below is now backed by a real, Go-proven (or artifact-proven) binding
> — or the affordance was retired by owner ruling. The gap count is pinned mechanically by
> `frontend-lab/tests/gap-count-zero.test.ts`. The only things left before the K6 flip are the owner's
> human smoke pass and the parked R6 bundle-split decision. Two decisions are surfaced for owner
> ratification: the settlement-binding deviation (`CreateCustomerReceipt` vs the ruling's literal
> `ApplyCustomerReceiptToInvoice`) and the butler costing-sheet "update status" redirect (its Go
> handler omits `Status`, so wiring it would be a pretend-persist). The per-screen `INTEG-pending`
> cells below are retained as the historical wiring ledger; treat this banner as the live truth.

This document consolidates the ~42 per-screen parity ledgers under
`frontend-lab/src/screens/parity/*.md` (+ `frontend-lab/PARITY_INVOICES.md`) plus the
composition notes in `FABLE_KERNEL_CAMPAIGN_LOG.md` into a single sign-off table. Each
row's detailed capability census lives in the linked parity doc — this is the roll-up.

---

## How to read this

**Migrated** — the screen is built in the kernel app, registered in `registry.ts`, and
passes the full gate suite (svelte-check 0/0, vitest, `wails build` frontend compile,
and the `tests/gate.mjs` layout detector @1440 + @420). Every product screen below is
`✅` here — that is the K-campaign's baseline invariant, not a per-screen claim.

**Read data** — where the *list/detail data* comes from today:
- `real` — the real Wails read binding is wired now; you see live data in a Wails build.
- `real*` — primary list is real; a *secondary* fetch (e.g. a full-profile second call)
  is not wired yet, so those extra fields are blanked/zeroed honestly (not faked).
- `mock (INTEG)` — reads are deliberately on the adversarial mock, by design, pending the
  owner-gated INTEG pass. A straight `pick()` swap wires them; no reshaping needed.

**Mutations** — create/approve/post/delete/etc.:
- `wired` — real mutation binding is called today.
- `mock (INTEG)` — the mutation runs against the mock; the real function throws an honest
  `Error('INTEG gap: <Binding> — wires at INTEG')` so nothing silently pretends to persist.
- `read-only` — the screen intentionally has no mutations (report/ledger view).

**INTEG-pending bindings** — the real Wails functions still to be wired for this screen.
This is the master to-do for Task #4 (INTEG completion), rolled up per screen. Financial
hot-zones are marked 🔥.

**Verdicts** used in the underlying ledgers: `DONE` (at parity now) · `EQUIV` (different
mechanism, same job, kernel way better) · `ENGINE` (needs a kernel feature that benefits
many screens) · `SLOT` (needs an L4 ejection component) · `INTEG` (needs real bindings) ·
`DEFER` (deliberately out of scope, tracked) · `RETIRE` (dropped on purpose).

> **The golden rule of this migration, re-stated for sign-off:** nothing renders a value
> the real Go backend cannot actually provide. Every gap is *ledgered*, never faked. So a
> `mock (INTEG)` cell is a wiring task, **not** a missing feature.

---

## Sign-off table

### Home

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| Dashboard · `dashboard` | DashboardScreen | Hub | ✅ | **real** | read-only | ✅ `GetDashboardStats`+`GetDashboardPipelineByStageYTD`+`GetDashboardARAgingReportYTD` composed (I2); focus/alerts/tasks honest-blank (no backing binding) | ☑ |

### Sales

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| CRM Customer Overview · `crm-customer` | CRMCustomerDashboard | Hub | ✅ | **real** | read-only | ✅ `GetCRMCustomerDashboard`/`…ByYear` (I2); top-customer share pct derived | ☑ |
| Customers · `customers` | CustomerDetailView | Entity | ✅ | **real** | wired (hold/reactivate) | ✅ `GetCustomerFullProfile` profile depth via new `profile.enrich` engine (I2) | ☑ |
| Orders · `orders` | OrdersScreen | Ledger | ✅ | real | **wired** | ✅ `QuickMarkOrderDelivered` (I3); delivery-% batch join (ENGINE); Create Invoice/PO/cascade separately ledgered | ☑ |
| RFQs · `rfqs` | RFQScreen | Ledger | ✅ | real | **wired** | ✅ `UpdateRFQStatus` (writes the col the row reads) + `DeleteRFQ` (I3); `dueDate` phantom noted | ☑ |
| Offers · `offers` | OffersScreen | Ledger | ✅ | real | read-only | `GenerateOfferPDF`/`OpenExportedFile` (PDF); Won/Lost + create ledgered | ☐ |
| Opportunities · `opportunities` | OpportunitiesScreen | Ledger | ✅ | **real** | **wired** | ✅ read merge (I2) + `CreateRFQWithReference`+`DeleteRFQ`/`DeleteOpportunity`(by source)+`DeleteRFQWithCascade` 🔥 (I3; cascade Go test deferred) | ☑ |
| Pricing · `pricing` | PricingScreen | Bespoke | ✅ | real* | `SimulateMargin` wired | ✅ `fetchPricingCustomers` WIRED (G1.4) — new read-only `GetCustomerWinRates` Go aggregation over real offer Won/Lost history (the old screen HARDCODED this list); regime derived from real win-rate | ☐ |
| Customer 360 · `customer-360` | Customer360 | Bespoke | ✅ | **real** | read-only | ✅ `GetCustomer360`+`GetCustomer360Graph` — view RESHAPED to backend (owner ruling): dropped mock-invented contact/TRN/credit/regime; connections derived from the graph | ☑ |
| Costing Sheet · `costing-sheet` | CostingSheet (3026 L) | Bespoke | ✅ | real | **wired** | ✅ `CreateCostingSheet`+`Clone…`+`SetActiveCostingRevision` (I3) + `SaveCostingAsOffer` 🔥 (R1.1; VM assembles flat CostingExportData w/ calcLine-computed lines, create path, integ_costing_hotzone_test.go); ✅ `UpdateCostingSheet` WIRED (G2, full struct assembled from the VM's authoritative totals) + PDF/Excel export WIRED + artifact-proven (G4, flat CostingExportData); in-place offer overwrite (needs offer UUID) remains DEFER | ☐ |
| Sales Hub · `sales-hub` | SalesHub | Bespoke (TabShell) | ✅ | composition | composition | none new — composes Opportunities/Costing/Offers/Orders (see those rows). `SalesAdminTools` tab DEFER | ☐ |
| Relationships Hub · `crm-hub` | CRMHub | Bespoke (TabShell) | ✅ | composition | composition | none new — composes crm-customer/crm-supplier/data-quality | ☐ |

### Finance

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| Finance Overview · `finance-overview` | FinancialDashboard | Hub | ✅ | **real** | read-only | ✅ `GetFinancialDashboardForYear` (I2); `GetCashPosition` already wired (bank-recon), no separate overlay consumer; CCC formula box (SLOT) | ☑ |
| Invoices · `invoices` | InvoicesScreen (2930 L) | Ledger | ✅ | real | **wired** (send/delete) | ✅ `SendCustomerInvoice` (R5; Draft→Sent, confirm, send_invoice_guard_test.go); ✅ settlement WIRED (G1.2) as a receipt-capture modal (`CreateCustomerReceipt` invoice-bound — real Payment+allocation, no status flip); standalone create RETIRED (G1.3, raised from Orders); `GenerateInvoicePDF` + proforma/edit/credit-override slots remain SLOT/DEFER (not INTEG throws) 🔥 | ☐ |
| Payments · `payments` | PaymentsScreen | Ledger | ✅ | real | **wired** | ✅ `ReverseCustomerReceipt` 🔥 (I3; server-gated + audit, receipt_reversal_test.go); `GetAllPayments` History panel deferred (ENGINE) | ☑ |
| Credit Notes · `credit-notes` | CreditNotes (in Invoices) | Ledger | ✅ | real | **wired** (apply) | ✅ `ApplyCreditNote` 🔥 (I3; reduces AR + auto-Paid + guards, integ_ar_hotzone_test.go); still gapped: `GenerateCreditNotePDF`, Issue form (SLOT) | ☐ |
| Supplier Invoices · `supplier-invoices` | SupplierInvoicesScreen | Ledger | ✅ | real | **wired** | ✅ `PerformThreeWayMatch`+`ApproveSupplierInvoice`(SoD)+`MarkSupplierInvoicePaid` 🔥 (I3; supplier_ap_gate_test.go) now CONSUMED as descriptor actions (R1.2: 3-way-match/approve/mark-paid w/ confirm+capture form); `CreateSupplierInvoice` gapped (struct-arg) | ☑ |
| Supplier Payments · `supplier-payments` | SupplierPaymentsScreen | Ledger | ✅ | real | **wired** (delete) | ✅ `DeleteSupplierPayment` 🔥 (I3; removes + re-derives invoice payment_status, integ_ap_hotzone_test.go); Record/Edit form-SLOTs deferred | ☐ |
| Cheque Register · `cheque-register` | ChequeRegisterScreen | Ledger | ✅ | real | **wired** | ✅ `MarkChequeStale` + `CancelCheque` (I3); Registers/Stale sub-views (ENGINE) | ☑ |
| Expenses · `expenses` | ExpensesScreen | Ledger | ✅ | real | **wired** | ✅ Submit/Approve/Reject/`DeleteExpenseEntry` 🔥 (I3) + `PostExpenseEntry` 🔥 (R1.3; owner-ratified, posts a real GL journal entry, confirm names the GL effect, integ_expense_hotzone_test.go) | ☑ |
| AHS Division Finance · `ahs-finance` | AHSDashboard | Hub | ✅ | **real** | read-only | ✅ `GetFinancialDashboardByDivision` with division resolved from registry (`dashboardVariant==='ahs'`, I1.2/I2) | ☑ |
| FX Revaluation · `fx-revaluation` | FXRevaluationScreen | Ledger | ✅ | real | **wired** | ✅ `PostFXRevaluation`+`ReverseRevaluation` 🔥 (I3; actor from session; fx_revaluation_golden_test.go); Exposure/Rates tabs (ENGINE) | ☑ |
| Book vs Bank Recon · `book-bank-recon` | BookBankReconciliationScreen | Bespoke | ✅ | **real** | **wired** (finalize) | ✅ `FinalizeBookBankReconciliation` 🔥 (I3 + R2 Go test) + recon-history/deposits/cheques reads wired (R3; record-own-totals, not live pending); Create/Update forms deferred | ☑ |
| Accounting · `accounting` | AccountingScreen (2098 L) | Bespoke | ✅ | real (8 fetches) | **wired** | ✅ `CreateAccount`+`CreateJournalEntry` 🔥 (I3, integ_accounting_hotzone_test.go)+`ReviewCashflowEvidenceProposal`+`UpdateAccount` (R3; verified whitelist, drops posting-owned balance, integ_residue_r3_test.go); ✅ 5× CSV/VAT/evidence export WIRED + artifact-proven (G4) + `SyncCashflowEvidenceProposalReviews` WIRED (G3, review sync never posts) | ☐ |
| Bank Reconciliation · `bank-reconciliation` | BankReconciliationScreen (2140 L) | Bespoke | ✅ | real (10 fetches) | **wired** | ✅ `FinalizeReconciliation`+`DeleteBankStatement` 🔥 (I3; integ_recon_hotzone_test.go)+`AutoMatch`+`ManualMatch`+`SplitAlloc`+`Unmatch`+two-phase import+line delete + `UpdateBankStatement`+`Create/UpdateBankStatementLine` (R3; verified whitelists, existing bank_reconciliation_service_test.go) | ☑ |
| Finance Hub · `finance-hub` | FinanceHub (13 tabs) | Bespoke (TabShell) | ✅ | composition | composition | none new — composes overview + 11 finance screens (see those rows). Division selector DEFER | ☐ |

### Operations

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| CRM Supplier Overview · `crm-supplier` | CRMSupplierDashboard | Hub | ✅ | **real** | read-only | ✅ `GetCRMSupplierDashboard`/`…ByYear` (I2); top-supplier share pct derived | ☑ |
| Purchase Orders · `purchase-orders` | PurchaseOrdersScreen | Ledger | ✅ | real | **wired** | ✅ `UpdatePOStatus` (I3) + `ReceiveAgainstPO` 🔥 (R5; per-line receive/reject capture modal via the new `ActionSpec.modal` L4 seam + LineItemsEditor, GetPurchaseOrderByID load, client guard mirrors server over-receive, existing grn_receive_and_complete_test.go); Approve/multi-currency-create = ledgered SLOT | ☑ |
| Delivery Notes · `delivery-notes` | DeliveryNotesScreen | Ledger | ✅ | real | **wired** | ✅ `DeleteDeliveryNote` (I3) + `DispatchDeliveryNote`(driver/vehicle form)+`ConfirmDeliveryNote`(POD signatory form) 🔥 (R5; real 2-step flow Prepared→Dispatched→Delivered, mock InTransit fiction retired) | ☑ |
| Goods Received · `grns` | GRNScreen | Ledger | ✅ | real | **wired** | ✅ `UpdateGRNQCStatus`(QC verdict form)+`CompleteGRN`(confirm) 🔥 (R5; qcBy from session); Receive-from-PO now lives on the PO ledger (`ReceiveAgainstPO` capture modal, R5 — same binding creates the GRN) | ☑ |
| Suppliers · `suppliers` | SuppliersScreen | Entity | ✅ | **real** | **wired** (delete) | ✅ `GetSupplierFullProfile` via `profile.enrich` (I2) + `DeleteSupplier` (R3; server refuses if linked); create + contacts/issues/notes ledgered | ☐ |
| Inventory Fulfillment · `inventory-fulfillment` | InventoryFulfillmentScreen | Ledger | ✅ | real | read-only | none on data; row-click "Open Order" nav (INTEG, app-shell router) | ☐ |
| Serial Trace · `serial-trace` | SerialTraceScreen | Bespoke | ✅ | **real** | read-only | ✅ `SearchSerials`, `GetRecentlyDeliveredSerials` (I2) | ☑ |
| Work · `work` | WorkHub (1445 L) | Bespoke (TabShell) | ✅ | real (6 fetches) | **wired** | ✅ all 14 task/project mutations (R3; hot-zone Delete/Archive/Shelve thread mandatory audit reason; due-date RFC3339 bridge; type-gate + existing collaboration_service tests) | ☑ |
| Operations Hub · `operations-hub` | OperationsHub | Bespoke (TabShell) | ✅ | composition | composition | none new — composes PO/DN/Fulfillment/Serial-Trace. Per-tab badge counts DEFER | ☐ |

### People

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| Payroll · `payroll` | PayrollScreen (1167 L) | Bespoke | ✅ | real (6 fetches) | **wired** | ✅ `GeneratePayrollRun`+`Approve`+`Post` 🔥+`MarkPaid`+`CreatePayrollPeriod` (I3; payroll_golden_test.go); ✅ `UpsertEmployeeCompensationProfile` WIRED (G2, financial+PII, Go-proven) + employee picker WIRED (`ListEmployeeProfiles`, cross-domain read) | ☐ |
| People · `people` | PeopleHub (1879 L, PII) | Bespoke (TabShell) | ✅ | real (10 fetches) | **wired** | ✅ all 13 PII/credential mutations (R3; no secret leakage, `actingUserId()` for GenerateLicenseKey, field-encrypted doc numbers sent plaintext; type-gate + existing employee/archive tests) 🔥 | ☑ |

### System

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| Users · `users` | UserManagementScreen | Entity | ✅ | real | read-only (RBAC) | Create/Update/role-assign deliberately **not built** (RBAC hot-zone) — wire at INTEG via server-gated call | ☐ |
| Approvals Queue · `approvals` | ApprovalsQueueScreen | Ledger | ✅ | **real** | **wired** | ✅ fetch (I2) + `ReviewDeleteApprovalRequest`/`ReviewEmployeeArchiveRequest` 🔥 both kinds (I3; server-derived reviewer; existing app_test/employee_archive_service_test cover) | ☑ |
| Audit Trail · `audit-trail` | AuditTrailViewer | Ledger | ✅ | **real** | **wired** | ✅ read chain (I2) + `ReverseAction` 🔥 (I3; actor from session, never trusted from row) | ☑ |
| Data Quality · `data-quality` | DataQualityScreen | Ledger | ✅ | real (preview real) | **wired** | ✅ `ReviewDataQualityIssue` (I3; admin-gated server-side); review-history panel (ENGINE) | ☑ |
| Notifications · `notifications` | NotificationsScreen | Bespoke | ✅ | **real** | **wired** (fetch/read) | ✅ `ListNotificationFeed`+`MarkNotificationAsRead` (I2); ✅ reviews WIRED (G3) — `sourceId` enrichment → `ReviewDeleteApprovalRequest`/`ReviewEmployeeArchiveRequest` by kind (server-derived reviewer); live-push DEFER | ☐ |
| Bank Accounts · `bank-accounts` | SettingsScreen (split) | Ledger | ✅ | real | **wired** | ✅ `DeleteBankAccount`+`CreateBankAccount`+`UpdateBankAccount` 🔥 (R1.4; PLAINTEXT contract — IBAN/SWIFT stored plaintext by design, encryption was removed & migration strips it; integ_bank_account_hotzone_test.go asserts roundtrip) | ☑ |
| Currency Rates · `currency-rates` | SettingsScreen (split) | Ledger | ✅ | real | **wired** | ~~`SetExchangeRate`~~ ✅ wired via kernel `map.goTime` date→time.Time bridge (I1.3); Go round-trip + persistence test green | ☑ |
| Business Settings · `business-settings` | SettingsScreen (split) | Bespoke | ✅ | **real** (keys fixed R3) | **wired** (AI key) | ✅ AI-provider key (R4): `SetAPIKeys` encrypt-at-rest write + new `GetAIProviderKeyStatus` DB-backed masked read (round-trips; GetSettings reads a different store) 🔥, integ_residue_r4_test.go; ✅ `UpdateSettings` WIRED (G3) via fetch-merge-write — round-trips the full GetSettings object, overlays only the 5 owned fields, preserves apiKeys/folders/language/theme; fetch-side `mapSettings` fixed to real keys (R3) | ☐ |
| Butler · `butler` | ButlerScreen (2960 L) | Bespoke | ✅ | real (chat) | **wired** (partial) | ✅ `ChatWithButlerPersistent`+`DeleteConversation`+`PurgeAllConversations` (R3); ✅ write-action SPLIT (G1.1): 19 draft/update bindings WIRED (human arms+confirms = the actor), 4 approve-class RETIRED to the Approvals Queue with a 15-case vitest boundary tripwire | ☐ |
| Deployment · `deployment` | DeploymentHub (1093 L) | Bespoke (TabShell) | ✅ | real (7 fetches) | **wired** (partial) | ✅ `UpdatePilotDeploymentChecklistItem`+`TriggerCollaborativeSyncNow`+`RetryCollaborativePendingOperations/Operation` 🔥+`ReassignEmployeeLicenseAccess`+`UpdateLicenseDisplayName` (R3); ✅ export bundle/signoff WIRED + artifact-proven (G4) | ☐ |
| OneDrive Import · `onedrive-import` | (unrouted Go service) | Bespoke (Wizard) | ✅ | **real** | **wired** | ✅ `DetectOneDrivePath`+`ValidateOneDrivePath`+`ScanOneDrivePaths`+`ImportOneDriveDeals` 🔥 (I3; server skips deals w/o confirmed customer → slip degrades to skip/error, never a wrong offer) | ☑ |

### Reports

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| Reports · `reports` | ReportsScreen | Hub | ✅ | real | read-only | none (fetch wired real); CSV export (ENGINE), PDF/Excel stubs (DEFER) | ☐ |

**Lab-only (not a product screen, excluded from the flip):** `showcase` — the kernel
component gallery. Stays in the lab app; will not ship to end users.

---

## Deliberately retired / not carried forward (owner-ratified)

These are **intentional** drops the owner already ratified in-wave — listed here so the
flip sign-off is explicit that they are gone by design, not lost in migration.

| Dropped thing | Where it lived | Why | Ruling |
|---|---|---|---|
| IntelligenceHub screen | old `IntelligenceHub` | Pure duplicate of Butler's surface | RETIRE → Butler (nav `intelligence` → `butler`) |
| Settings "Deployment" tab | old `SettingsScreen` | Duplicate of DeploymentHub stats + a nav shortcut | RETIRE → route to DeploymentHub |
| Weekly per-employee activity monitor | old `DeploymentHub` | Surveillance-adjacent; no business need | RETIRE (no flag/binding/UI) |
| Bahrain VAT Summary card (hardcoded `VAT_RATE=0.1`) | old `AccountingScreen` | Fragile substring matching + hardcoded rate | DROP (orchestrator-ratified) |
| Supplier/User "Pending" status tab | old Suppliers/Users | No backing field server-side (`\|\| 'Active'` fallback) | FIX → honest 2-state from `is_active` |
| `EcosystemDashboard` | non-Wails dev/research tool | Edge-tab scraping local runtime, never an end-user screen | NOT MIGRATED (dev tool, out of scope) |
| localStorage draft autosave / sessionStorage cross-screen handoffs | Costing/Work/PO/etc. | Cross-screen handoff pattern replaced by the nav store | DEFER/DROP per screen |
| Standalone invoice-create (+New Invoice form) | Invoices screen | Invoices are raised from an order (`CreateInvoiceWithOptions`), never conjured standalone | RETIRE (owner ruling G1.3) → Orders |
| Butler approve-class actions (`ApprovePurchaseOrder`, `ApproveStockAdjustment`, `ApproveSupplierInvoice`, `ApproveCostingSheet`) | Butler action vocabulary | AI-authority boundary — the agent never puts an approval one click away | RETIRE (owner ruling G1.1) → butler replies pointing at the Approvals Queue; mechanically tripwired |

---

## Consolidated INTEG roster — ✅ CLOSED (`INTEG gap:` 0)

This was the wiring backlog. It is now **fully closed** across the INTEG, Residue, and Gap-Close
campaigns — every binding listed here is wired-and-verified (or the affordance was retired by owner
ruling). The bridge throws ZERO `INTEG gap:` today, pinned by `tests/gap-count-zero.test.ts`.

- **🔥 Financial / irreversible hot-zones** — ALL wired + Go-proven: invoice send + settlement
  (receipt capture, G1.2); `ReverseCustomerReceipt`; `ApplyCreditNote`; supplier-invoice
  3-way-match/approve/pay; `DeleteSupplierPayment`; `PostFXRevaluation`/`ReverseRevaluation`;
  `FinalizeBookBankReconciliation`; `FinalizeReconciliation`/`DeleteBankStatement`; `CreateJournalEntry`;
  PO Receive Items; GRN Receive/Complete; payroll generate/approve/post + `UpsertEmployeeCompensationProfile`
  (G2); `SaveCostingAsOffer` + `UpdateCostingSheet` (G2); `DeleteRFQWithCascade` (RFQ-only, descriptor-gated,
  G3); `ImportOneDriveDeals`; delete/archive-approval reviews (Approvals Queue + Notifications cards, G3);
  the 4 butler approve-class bindings RETIRED (G1.1).
- **Reads** — all real: dashboards (main/CRM/AHS/finance-overview), Opportunities 2-source, `GetCustomer360`,
  Serial Trace, Audit Trail, Approvals/Notifications, pricing win-rate (`GetCustomerWinRates`, G1.4).
- **Secondary-fetch depth** — wired: `GetCustomerFullProfile` / `GetSupplierFullProfile` (profile.enrich),
  `GetCashPosition`. (List mappers honestly blank profile-depth fields until the profile opens — not a gap.)
- **Cross-cutting prerequisites** — all built: session/currentUser store (`actingUserId`), divisions registry,
  date→`time.Time` form bridge (`map.goTime`), AI-provider key encrypted-at-rest (`SetAPIKeys` + masked read).
- **Exports (G4)** — all 10 wired + artifact-proven (5 CSV/VAT/evidence, 3 costing, 2 pilot bundles).
- **Settings** — `UpdateSettings` wired via fetch-merge-write (G3).

---

## Kernel gaps still open (not blockers; tracked engine work)

- `ProfileKpiSpec.tone` — can't color a profile KPI by a computed condition (Customers credit-block).
- `profile.tabs` / `profile.slots` — nested tabbed detail + nested CRUD collections
  (Suppliers/Customers contacts/issues/notes; 5-tab detail views).
- `ColumnSpec.rowAction` — a button-style per-row action (DeploymentHub retry). OneDrive's
  stateful checkbox/select used the existing `ColumnSpec.cell` L4 ejection instead.
- Multi-panel / secondary-status-badge composition — several finance ledgers want a co-located
  second panel (Payment History, review-history, Exposure/Rates tabs) or a dual-status column.
- Hub-level `actions` — Reports CSV export has nowhere to hang on a `HubDescriptor`.

None of these block the flip: each is a *deferred capability with a ledgered reason*, and the
screen ships at honest parity without it.

---

## Owner smoke checklist (run in a Wails build before you say "flip")

Do this in a real `wails dev` / `wails build` of the **kernel** app (`frontend-lab`), not the
lab dev server, so you're exercising the real shell + nav + (where wired) real bindings. The
automated gate already proves layout @1440/@420 and type/test green on every screen — this
checklist is the *human* pass over feel + real-data reads.

**Shell & navigation**
- [ ] App boots to Dashboard; sidebar shows all groups (Home/Sales/Finance/Operations/People/System/Reports).
- [ ] Every sidebar entry opens its screen with no console errors.
- [ ] RBAC-gated entries hide/show correctly for a non-admin session.
- [ ] Deep-link a hub tab (e.g. `finance-hub` → Payroll tab) — lands on the right tab.
- [ ] A dashboard KPI/widget drill-down navigates to the target ledger **with its filter seeded**.

**Read-data screens that are wired real today** (should show live Postgres data, not mock):
- [ ] Invoices, Payments, Credit Notes, Supplier Invoices/Payments, Cheque Register, Expenses,
      FX Revaluation list rows look right vs the legacy screen.
- [ ] Orders / RFQs / Offers / POs / Delivery Notes / GRNs / Inventory list rows match legacy.
- [ ] Customers / Suppliers master lists match; open a profile — wired fields populate, the
      profile-depth fields (TRN/aging/KPIs) are blank (expected: `GetXFullProfile` not wired yet).
- [ ] Accounting / Bank Reconciliation / Book-vs-Bank / Payroll / People / Work / Deployment /
      Reports / Data Quality / Pricing / Costing Sheet load their real fetches without error.

**Mock-backed screens (expected: adversarial synthetic data, honest INTEG throws on write)**
- [ ] Dashboards (main + CRM + AHS), Opportunities, Customer 360, Serial Trace, Audit Trail,
      Approvals, Notifications, OneDrive Import render with synthetic data and no layout breakage.
- [ ] Trigger a mutation on any mock screen (e.g. OneDrive Import → Import) — it should complete
      against the mock **or** surface a clear `INTEG gap: <Binding>` message; it must never
      silently claim to have persisted to the real DB.

**Look & feel**
- [ ] Fonts/spacing/tone match the design constitution (no muddy fonts, no flat-token screens).
- [ ] Resize the window narrow — no horizontal page scroll; tables scroll inside their own region.
- [ ] Spot-check a couple of screens in dark mode if the shell supports it.

**Data safety**
- [ ] Confirm no real client names / TRNs / bank details / people appear anywhere (synthetic only).

---

## The flip itself — **NOT executed. Owner go required.**

When (and only when) the owner says go, K6 execution is:

1. **Confirm the parity table above is fully ☐→☑.** Any un-ticked row is a blocker or an
   accepted-known-gap the owner explicitly signs off.
2. **Repoint the Wails build** from `frontend/` to `frontend-lab/` (build config + `go:embed`).
3. **`wails build -clean`** smoke — the kernel app compiles into the single binary and launches.
4. **Delete `frontend/`** (the legacy app) — reversible via git until pushed.
5. **Full gate suite** green post-delete: `go build ./...`, `svelte-check` 0/0, vitest,
   `tests/gate.mjs` layout detector @1440/@420.
6. **Owner decides graduation** (merge / push). This document does **none** of steps 2–6.

Until then: `frontend/` is untouched, no binary is built, nothing is pushed.

---

*Generated at K6 flip-prep. Sources: `frontend-lab/src/screens/parity/*.md`,
`frontend-lab/PARITY_INVOICES.md`, `FABLE_KERNEL_CAMPAIGN_LOG.md`, `frontend-lab/src/screens/registry.ts`.*
