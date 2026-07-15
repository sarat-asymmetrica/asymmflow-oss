# FABLE Wave K6 — Consolidated Parity Sign-Off & Flip Checklist

**Branch:** `exp/frontend-kernel` · **Worktree:** `c:\Projects\asymmflow\asymmflow-lab`
**Status:** FLIP-PREP (docs only). Nothing destructive has run. The old `frontend/`
is **untouched**, no `wails build` has been triggered, and nothing has been pushed.
**Purpose:** one place for the owner to sign off screen-by-screen that the kernel
app (`frontend-lab/`) reaches parity with the legacy `frontend/`, and to see exactly
what real-binding wiring (INTEG) remains before graduation.

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
| Orders · `orders` | OrdersScreen | Ledger | ✅ | real | mock (INTEG) | `QuickMarkOrderDelivered`; delivery-% batch join (ENGINE) | ☐ |
| RFQs · `rfqs` | RFQScreen | Ledger | ✅ | real | mock (INTEG) | `UpdateRFQStage`, `UpdateRFQStatus` | ☐ |
| Offers · `offers` | OffersScreen | Ledger | ✅ | real | read-only | `GenerateOfferPDF`/`OpenExportedFile` (PDF); Won/Lost + create ledgered | ☐ |
| Opportunities · `opportunities` | OpportunitiesScreen | Ledger | ✅ | **real** | mock (INTEG) | ✅ `GetRFQs`+`GetPipelineOpportunities` merge + `ListCustomers` (I2); still I3: `CreateRFQWithReference`, `DeleteRFQ`, `DeleteRFQWithCascade` 🔥 | ☐ |
| Pricing · `pricing` | PricingScreen | Bespoke | ✅ | real* | `SimulateMargin` wired | `fetchPricingCustomers` (no real customer/win-rate endpoint yet) | ☐ |
| Customer 360 · `customer-360` | Customer360 | Bespoke | ✅ | **real** | read-only | ✅ `GetCustomer360`+`GetCustomer360Graph` — view RESHAPED to backend (owner ruling): dropped mock-invented contact/TRN/credit/regime; connections derived from the graph | ☑ |
| Costing Sheet · `costing-sheet` | CostingSheet (3026 L) | Bespoke | ✅ | real | mock (INTEG) | `CreateCostingSheet`, `UpdateCostingSheet`, `SaveCostingAsOffer` 🔥, `Clone…`, `SetActiveCostingRevision`, PDF/Excel export | ☐ |
| Sales Hub · `sales-hub` | SalesHub | Bespoke (TabShell) | ✅ | composition | composition | none new — composes Opportunities/Costing/Offers/Orders (see those rows). `SalesAdminTools` tab DEFER | ☐ |
| Relationships Hub · `crm-hub` | CRMHub | Bespoke (TabShell) | ✅ | composition | composition | none new — composes crm-customer/crm-supplier/data-quality | ☐ |

### Finance

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| Finance Overview · `finance-overview` | FinancialDashboard | Hub | ✅ | **real** | read-only | ✅ `GetFinancialDashboardForYear` (I2); `GetCashPosition` already wired (bank-recon), no separate overlay consumer; CCC formula box (SLOT) | ☑ |
| Invoices · `invoices` | InvoicesScreen (2930 L) | Ledger | ✅ | real | mock (INTEG) — create-Draft form shipped | `SendCustomerInvoice`, `GenerateInvoicePDF`, `GetAvailableDeliveryNotesForOrder`; edit/proforma/credit-override slots 🔥 | ☐ |
| Payments · `payments` | PaymentsScreen | Ledger | ✅ | real | **wired** | ✅ `ReverseCustomerReceipt` 🔥 (I3; server-gated + audit, receipt_reversal_test.go); `GetAllPayments` History panel deferred (ENGINE) | ☑ |
| Credit Notes · `credit-notes` | CreditNotes (in Invoices) | Ledger | ✅ | real | **wired** (apply) | ✅ `ApplyCreditNote` 🔥 (I3; reduces AR + auto-Paid + guards, integ_ar_hotzone_test.go); still gapped: `GenerateCreditNotePDF`, Issue form (SLOT) | ☐ |
| Supplier Invoices · `supplier-invoices` | SupplierInvoicesScreen | Ledger | ✅ | real | read-only | `PerformThreeWayMatch`, Approve (SoD), Mark Paid, New — all ledgered 🔥 | ☐ |
| Supplier Payments · `supplier-payments` | SupplierPaymentsScreen | Ledger | ✅ | real | mock (INTEG) | `DeleteSupplierPayment` 🔥; Expenses-settlement merge wrapper; Record/Edit ledgered | ☐ |
| Cheque Register · `cheque-register` | ChequeRegisterScreen | Ledger | ✅ | real | mock (INTEG) | `MarkChequeStale`, `CancelCheque`; `GetChequeRegisters`/`GetStaleCheques` sub-views (ENGINE) | ☐ |
| Expenses · `expenses` | ExpensesScreen | Ledger | ✅ | real | mock (INTEG) | `SubmitExpenseEntry`, `Approve`, `Reject`, `Post`, `DeleteExpenseEntry` 🔥; `getExpenseDashboardSummary` | ☐ |
| AHS Division Finance · `ahs-finance` | AHSDashboard | Hub | ✅ | **real** | read-only | ✅ `GetFinancialDashboardByDivision` with division resolved from registry (`dashboardVariant==='ahs'`, I1.2/I2) | ☑ |
| FX Revaluation · `fx-revaluation` | FXRevaluationScreen | Ledger | ✅ | real | mock (INTEG) | `PostFXRevaluation` 🔥, `ReverseRevaluation` 🔥; Exposure/Rates tabs (ENGINE) | ☐ |
| Book vs Bank Recon · `book-bank-recon` | BookBankReconciliationScreen | Bespoke | ✅ | real (3-call agg) | mock (INTEG) | `FinalizeBookBankReconciliation` 🔥; Create/Update recon forms | ☐ |
| Accounting · `accounting` | AccountingScreen (2098 L) | Bespoke | ✅ | real (8 fetches) | mock (INTEG) | `CreateAccount`, `UpdateAccount`, `CreateJournalEntry` 🔥, `ReviewCashflowEvidenceProposal`, 5× CSV/VAT export | ☐ |
| Bank Reconciliation · `bank-reconciliation` | BankReconciliationScreen (2140 L) | Bespoke | ✅ | real (10 fetches) | mock (INTEG) | `FinalizeReconciliation` 🔥, `DeleteBankStatement` 🔥, `AutoMatchBankLines`, `ManualMatchLine`, import preview/confirm, +7 line ops | ☐ |
| Finance Hub · `finance-hub` | FinanceHub (13 tabs) | Bespoke (TabShell) | ✅ | composition | composition | none new — composes overview + 11 finance screens (see those rows). Division selector DEFER | ☐ |

### Operations

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| CRM Supplier Overview · `crm-supplier` | CRMSupplierDashboard | Hub | ✅ | **real** | read-only | ✅ `GetCRMSupplierDashboard`/`…ByYear` (I2); top-supplier share pct derived | ☑ |
| Purchase Orders · `purchase-orders` | PurchaseOrdersScreen | Ledger | ✅ | real | mock (INTEG) | `UpdatePOStatus`; Approve (SoD), Receive Items 🔥, multi-currency create ledgered | ☐ |
| Delivery Notes · `delivery-notes` | DeliveryNotesScreen | Ledger | ✅ | real | mock (INTEG) | `DispatchDeliveryNote`, `ConfirmDeliveryNote`, `DeleteDeliveryNote`; order/customer enrichment join | ☐ |
| Goods Received · `grns` | GRNScreen | Ledger | ✅ | real | read-only | Receive / QC Review / Complete — all ledgered SLOT 🔥 | ☐ |
| Suppliers · `suppliers` | SuppliersScreen | Entity | ✅ | **real** | mock (INTEG) delete | ✅ `GetSupplierFullProfile` via `profile.enrich` (I2); still I3: `DeleteSupplier`; create + contacts/issues/notes ledgered | ☐ |
| Inventory Fulfillment · `inventory-fulfillment` | InventoryFulfillmentScreen | Ledger | ✅ | real | read-only | none on data; row-click "Open Order" nav (INTEG, app-shell router) | ☐ |
| Serial Trace · `serial-trace` | SerialTraceScreen | Bespoke | ✅ | **real** | read-only | ✅ `SearchSerials`, `GetRecentlyDeliveredSerials` (I2) | ☑ |
| Work · `work` | WorkHub (1445 L) | Bespoke (TabShell) | ✅ | real (6 fetches) | mock (INTEG) | 14 task/project mutations incl. `Delete/Archive/ShelveCollaborativeProject` 🔥, `CreateCollaborativeTask`, `UpdateCollaborativeTaskStatus`, … | ☐ |
| Operations Hub · `operations-hub` | OperationsHub | Bespoke (TabShell) | ✅ | composition | composition | none new — composes PO/DN/Fulfillment/Serial-Trace. Per-tab badge counts DEFER | ☐ |

### People

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| Payroll · `payroll` | PayrollScreen (1167 L) | Bespoke | ✅ | real (6 fetches) | mock (INTEG) | `GeneratePayrollRun`, `ApprovePayrollRun`, `PostPayrollRun` 🔥, `MarkPayrollRunPaid`, `UpsertEmployeeCompensationProfile`, `CreatePayrollPeriod`, employee picker | ☐ |
| People · `people` | PeopleHub (1879 L, PII) | Bespoke (TabShell) | ✅ | real (10 fetches) | mock (INTEG) | 13 PII/credential mutations incl. `Create/UpdateEmployeeProfile`, `RequestEmployeeArchive`, `GenerateLicenseKey`, `Create/DeleteEmployeeDocument` 🔥 | ☐ |

### System

| Screen · `key` | Old screen | Type | Migr. | Read data | Mutations | INTEG-pending (real bindings) | ☐ |
|---|---|---|---|---|---|---|---|
| Users · `users` | UserManagementScreen | Entity | ✅ | real | read-only (RBAC) | Create/Update/role-assign deliberately **not built** (RBAC hot-zone) — wire at INTEG via server-gated call | ☐ |
| Approvals Queue · `approvals` | ApprovalsQueueScreen | Ledger | ✅ | **real** | mock (INTEG) | ✅ `ListDeleteApprovalRequests`+`ListEmployeeArchiveRequests` fetch (I2); still I3: `ReviewDeleteApprovalRequest`, `ReviewEmployeeArchiveRequest` 🔥 | ☐ |
| Audit Trail · `audit-trail` | AuditTrailViewer | Ledger | ✅ | **real** | mock (INTEG) | ✅ `GetActiveBankAccounts`→`GetBankStatements`→`GetAuditTrail` chain (I2); still I3: `ReverseAction` 🔥 | ☐ |
| Data Quality · `data-quality` | DataQualityScreen | Ledger | ✅ | real (preview real) | mock (INTEG) | `ReviewDataQualityIssue`; review-history panel (ENGINE) | ☐ |
| Notifications · `notifications` | NotificationsScreen | Bespoke | ✅ | **real** | mock (INTEG) | ✅ `ListNotificationFeed` fetch + `MarkNotificationAsRead` (I2); reviewStatus/requester honest-blank (not on record); still I3: approve/reject reviews; live-push DEFER | ☐ |
| Bank Accounts · `bank-accounts` | SettingsScreen (split) | Ledger | ✅ | real | mock (INTEG) | Create/Update/Delete bank account (division-scoped + encrypted IBAN/SWIFT) 🔥 | ☐ |
| Currency Rates · `currency-rates` | SettingsScreen (split) | Ledger | ✅ | real | **wired** | ~~`SetExchangeRate`~~ ✅ wired via kernel `map.goTime` date→time.Time bridge (I1.3); Go round-trip + persistence test green | ☑ |
| Business Settings · `business-settings` | SettingsScreen (split) | Bespoke | ✅ | real | mock (INTEG) | `UpdateSettings` (unverified key vocabulary — confirm against Go handler first) | ☐ |
| Butler · `butler` | ButlerScreen (2960 L) | Bespoke | ✅ | real (chat) | mock (INTEG) | `executeButlerAction` seam over 23 write actions; `ChatWithButlerPersistent`, `DeleteConversation`, `PurgeAllConversations` | ☐ |
| Deployment · `deployment` | DeploymentHub (1093 L) | Bespoke (TabShell) | ✅ | real (7 fetches) | mock (INTEG) | `UpdatePilotDeploymentChecklistItem`, `TriggerCollaborativeSyncNow`, `RetryCollaborativePendingOperations` 🔥, export bundle/signoff, +2 | ☐ |
| OneDrive Import · `onedrive-import` | (unrouted Go service) | Bespoke (Wizard) | ✅ | mock (INTEG) | mock (INTEG) | `DetectOneDrivePath`, `ValidateOneDrivePath`, `ScanOneDrivePaths`, `ImportOneDriveDeals` 🔥 (Import creates offers) | ☐ |

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

---

## Consolidated INTEG roster (feeds Task #4 — **owner-gated**)

Every real binding still throwing `INTEG gap: …` today, grouped by risk. This is the
wiring backlog for the sovereign-mesh / owner's **local Postgres** runtime — **not** the
legacy DuckDNS-Postgres and **not** the live PH SQLite at `%APPDATA%\Roaming\AsymmFlow`.
Per the handoff, INTEG does not start until the owner confirms the Postgres/runtime env.

- **🔥 Financial / irreversible hot-zones (wire last, with tests):** invoice send/PDF +
  edit/proforma/credit-override; `ReverseCustomerReceipt`; `ApplyCreditNote`; supplier-invoice
  3-way-match/approve/pay; `DeleteSupplierPayment`; `PostFXRevaluation`/`ReverseRevaluation`;
  `FinalizeBookBankReconciliation`; `FinalizeReconciliation`/`DeleteBankStatement`;
  `CreateJournalEntry`; PO Receive Items; GRN Receive/Complete; payroll generate/approve/post;
  `SaveCostingAsOffer`; `DeleteRFQWithCascade`; `ImportOneDriveDeals`; delete-approval reviews.
- **Reads still on mock (straight `pick()` swap, low risk):** dashboards (main = `GetDashboardStats`
  + pipeline + AR-aging YTD; CRM customer/supplier; AHS-by-division; finance-overview `GetFinancialDashboardForYear`),
  Opportunities 2-source fetch, `GetCustomer360`,
  Serial Trace searches, Audit Trail chain, Approvals/Notifications fetches.
- **Secondary-fetch depth (blank-till-wired today):** `GetCustomerFullProfile`,
  `GetSupplierFullProfile`, `GetCashPosition` (live-cash overlay).
- **Cross-cutting prerequisites (build once, unblocks many):** app-shell **session/currentUser**
  store (BankRecon uses placeholder `actor='lab-user'`); **divisions registry** store
  (`GetDivisionRegistry`) for AHS + payment/invoice division scoping; a real **date→`time.Time`**
  form bridge (`SetExchangeRate`); a secrets-storage decision for AI provider keys (Settings DEFER).

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
