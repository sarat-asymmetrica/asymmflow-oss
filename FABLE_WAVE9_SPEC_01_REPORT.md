# Wave 9 Spec-01 Status Report
**Branch:** feat/fable-wave9-1-dead-ends · **Commits:** 7 (920b07e..6d06093), off `main` — NOT merged/pushed, left for owner review
**Gates (final commit):** vite build ✅ · svelte-check ✅ 0 errors / 14 warnings (= pre-wave baseline; net **0 new errors, 0 new warnings**) · go test — not run (no Go source touched; frontend-only wave)

**Operating model:** Opus 4.8 orchestrator as senior designer (spec + constitutional review + gating + small corrections); five Sonnet 5 subagents wrote the code in disjoint-file batches. Every diff was reviewed against the Design Constitution before commit and the gates were run centrally by the orchestrator (never trusting a subagent's build claim). One shared foundation piece — the canonical confirm primitive — was authored by the orchestrator directly.

---

## Phase A — ground-truth verdicts

| Check | Verdict | Notes |
|---|---|---|
| **A1** Logout / user menu in shell | **ALREADY FIXED** | `EnterpriseHeader.svelte` renders a visible **Sign out** button (Wave 6 Mission C.1) → `LogoutInteractiveSession()` + `app:logout` event (consumed in App.svelte). Avatar in header + sidebar. No Wave 9.1 work needed. |
| **A2** Employee-archive approval surface | **ALREADY FIXED** | Reachable on `NotificationsScreen.svelte` (`isPendingEmployeeArchiveApproval`, `reviewEmployeeArchive` → `reviewEmployeeArchiveRequest`). Its reject-reason used a native `prompt()` → folded into the B9 sweep. |
| **A3** Supplier-invoice Edit Status/PaymentStatus dropdowns vs P5-1 gate | **CONFIRMED — but DIVERGED from hypothesis; severity HIGH** | Not a hard-error as the audit guessed — a **silent off-ledger bypass**. The Edit modal exposes free-jump `<select>` for `status` (SupplierInvoicesScreen.svelte:1447) and `payment_status` (:1467). Backend `UpdateSupplierInvoice` field-masks `ApprovedBy/MatchStatus` (INT-001) so it won't forge the approval trail or crash — **but** it writes the `status` column freely, and `payment_status='Paid'` auto-flips `status='Paid'` + stamps `PaymentDate` with **no supplier-payment ledger entry, no journal, no SoD / 3-way-match gate** (those live only in `ApproveSupplierInvoice`/`MarkSupplierInvoicePaid`). Net: a user can mark an invoice Approved/Paid off-ledger from Edit, bypassing the gated chain. **Fix deferred to Wave 9.2 per spec — NOT done here.** (B9 did convert this screen's separate native approve-`confirm()`; it did NOT touch these dropdowns.) |
| **A4** Wave 8 backend wiring | **CONFIRMED unwired** — all exist in Go + wailsjs bindings, zero frontend callers | `GetInventoryPendingFulfillmentReport` → wired in **B2**. `GetDashboardPipelineByStageYTD` / `GetDashboardARAgingReportYTD` → **still uncalled**: OSS `DashboardScreen` has no pipeline-by-stage or AR-aging-bucket widget to feed them (B1.3 skipped, not invented). `CreatePOsFromOrder` (plural) → no frontend caller; OSS Orders uses singular `CreatePOFromOrder`, made item-aware in **B7**. `PreviewOrderDeleteCascade` → no caller; OSS Orders delete has no cascade preview (residue, out of B7 AC). `GetPreparedByOptions` → no caller (a Wave 9.3/9.5 costing item). |
| **A5** Audit anchors resolve on OSS | **PARTIAL — significant drift, corrected per-item** | Screens live at `frontend/src/lib/screens/` (audit cited `ph_holdings`). Backend goes through `wailsjs/go/{main.App,CRMService,FinanceService}` bindings. Key corrected divergences fed to coders: **FinancialDashboard on OSS has NO drill navigation** (the audit's "proven pendingInvoiceFilter pattern" does not exist here → B1 had to *build* the drills + add param consumption); **Offers create/edit modals are LIVE on OSS** (not dead code → B8 kept them); **PO `{#if false}` block absent** on OSS (→ B8 skip); native-dialog sweep found 18 real sites across 13 files (→ B9). |

---

## Phase B — items

| Item | Status | Commit | Notes |
|---|---|---|---|
| **B1** Dashboard drill-throughs | **shipped (1 sub-item skipped as N/A)** | 4d5f8f3 | KPI cards + task rows now clickable launchpads; AR→Overdue filter threaded FinanceHub→InvoicesScreen; pipeline-stage filter threaded SalesHub→OpportunitiesScreen (orchestrator added the one-line `params` forward SalesHub was missing). Fixed a latent nav bug (`bank-recon`→`bank_recon`). **B1.3 (pipeline/aging widgets) skipped** — those widgets don't exist on OSS DashboardScreen; the two YTD backends stay uncalled (reported, not invented). |
| **B2** Operations "Fulfillment" tab | **shipped** | 9f143cb | New `InventoryFulfillmentScreen` (canonical DataTable, loading/error/empty states) behind a new hub tab; `GetInventoryPendingFulfillmentReport(500)` wired; rows deep-link to the sales order (report carries no per-line PO field — order is the correct target). |
| **B3** 360 continuity | **shipped (audit premise corrected)** | 4d5f8f3 | Supplier-360 PO/invoice rows + customer-360 order/invoice/RFQ rows now keyboard-accessible drills; "New PO for this supplier" + "New RFQ for this customer" preseeded handoffs. Audit said to *mirror* the customer-360's working drills — but customer-360 had none either; made both consistent. Some drills land surface-only (targets outside this batch's file scope); real preselect where the landing screen was in scope (invoices, opportunities). |
| **B4** Cheque lifecycle row actions | **shipped** | 32c5408 | Canonical Dropdown menu exposing only the legal next transitions per status, gated to the real Go preconditions (Clear/Stale accept ISSUED\|PRESENTED, Cancel ISSUED-only); confirm primitive on every action (required reason on Cancel); in-place refresh. |
| **B5** Serial-trace deep-links | **shipped** | 9f143cb | PO/DN/Invoice refs are real navigating buttons (DataTable cell-snippet — `{@html}` can't carry handlers). `grn_number` left plain: no GRN tab exists on OSS to link to. |
| **B6** DN ergonomics | **shipped** | 32c5408 | Address/contact auto-fill from order/customer (editable); Dispatch opens an inline capture Modal for missing driver/vehicle instead of dead-ending; create-form status picker removed; create default corrected Draft→Prepared to match `DispatchDeliveryNote`'s precondition. |
| **B7** Order→PO handoff | **shipped** | 271af3f | `CreatePOFromOrder` now passes the order's item IDs (was `[]`); lands inside the new PO draft (`pendingOpenPO`) with an "approve to send" toast; zero-item orders show disabled+explained DN/PO CTAs. Exactly one PO-creation path remains. |
| **B8** Dead-code + Inbox | **shipped** | 271af3f | Orphaned `InboxScreen` **retired** (deleted; zero importers anywhere — cleaner than the audit implied). Verified the audit's other "dead code" doesn't apply on OSS: Offers create/edit modals are **live** (kept), PO `{#if false}` block **absent** (nothing to delete). Added the PurchaseOrdersScreen readers for the B7/B3 handoffs. |
| **B9** Native confirm/prompt sweep | **shipped** | 6d06093 | All 18 native `confirm()`/`prompt()` sites across 13 screens/components → the canonical primitive (variant by consequence; `askForReason` for the 3 text-capturing sites). Fixed a latent bug (Notifications reject submitted an empty reason on cancel; now early-returns). `grep` for native dialogs now returns only 3 documented exceptions (a loop var named `alert`, two dev-showcase demo files). |

**Definition of done:** Phase A table complete ✅ · all nine B-items shipped (B1 with one N/A sub-item) ✅ · gates green on the final commit ✅ · report written ✅.

---

## Decisions taken
- **Canonical confirm primitive built as foundation (commit 920b07e).** The only promise-based confirm on OSS was a *local* function inside OrdersScreen — not reusable. Rather than 13 screen-local dialogs, the orchestrator authored one store-backed primitive (`$lib/stores/confirm.ts`) rendered once by `<ConfirmHost/>` in App.svelte (mirrors the toast store pattern): `confirm.ask()` for yes/no and `confirm.askForReason()` for decisions that capture a reason. Satisfies Article III.6 + VI.2 and made B9 a clean sweep.
- **Inbox = RETIRE** (spec default). `InboxScreen` was unrouted, had `===` no-op filters, and duplicated the Capture/OCR triage flow. Grep found **zero importers** anywhere, so it was deleted outright (no App.svelte references even needed removing). Rationale: retiring beats reviving redundant, broken triage UI.
- **B1.3 not invented.** The two YTD dashboard backends have no corresponding widget on OSS DashboardScreen; building pipeline-donut / aging-bar UI is net-new feature work beyond "kill the dead ends," so it was left for a later wave and reported honestly rather than faked.
- **Surface-vs-preselect drills (B3).** Where a drill's landing screen was outside the batch's file scope (e.g. supplier-invoice detail), the row navigates to the correct *surface* rather than force a half-wired preselect; full preselect was implemented only where the landing screen was owned in the same batch.

## Constitution deviations requested
1. **Article VI (no raw hex in screen code) — `DashboardScreen.svelte`, new hover/focus CSS.** The entire DashboardScreen stylesheet is pre-token raw hex (`#35a66f`, `#79a9df`, …) with zero `var(--…)` usage. The new KPI-card hover/focus states reuse the file's *existing* hex value (`#79a9df`, already its focus color) rather than introduce a lone token that would visually diverge from the rest of the screen. Full tokenization of DashboardScreen is pre-existing debt beyond this wave's scope. **Request:** accept as-is for Wave 9.1, or schedule a DashboardScreen tokenization pass. *(Note: the one brand-new file, `InventoryFulfillmentScreen`, had its status-color hex converted by the orchestrator to the `var(--token,#fallback)` pattern — so new-file code is compliant; only the pre-existing hex-only screen is at issue.)*

*No other deviations.* No financial-semantics changes, no secrets, no real client data, layer model respected.

## Keep-list attestation
All keep-list behaviors on touched screens were preserved:
- **Sales (Offers/Orders):** status-driven conditional CTAs, RFQ→Offer→Order traceability, MarkOfferWon PO capture, won/lost edit-locking, cascade-preview delete — untouched by B7/B8/B9 (B9 swapped the dialog layer only, keeping every guard). ✅
- **Inventory:** Order→DN store handoff + no-items guard (B6/B7 strengthened it), GRN remaining-qty defaulting (B9 converted only the Complete-confirm, guard intact), serial-trace read-only search + empty/loading + warranty coloring (B5 preserved), `CreatePOsFromOrder` per-supplier backend untouched. ✅
- **CRM/shell:** customer-360 drill-throughs + full-master edit, guarded soft-delete surfacing backend block reasons (B9 kept the block-reason toasts), role-adaptive dashboard + Operating Focus deep-links with tab/company params (B1 preserved and *fixed* the broken Cash link). ✅
- **Finance AR/AP:** the gated Match→Approve→Settle chain (B9 converted only the native approve-confirm; the gate itself and the A3 dropdowns were left exactly as-is for Wave 9.2), apply-receipt handoff untouched. ✅
- **Finance recon:** bank-recon Finalize/Reopen gating + split-allocation match preserved (B9 converted 3 deletes, logic intact), cheque next-number preview untouched (B4 added actions only to Outstanding/Stale tabs; Registers tab untouched). ✅

## Known residue / follow-ups
- **A3 supplier-invoice off-ledger Edit bypass (HIGH)** — remove the state-advancing Status/PaymentStatus controls from the Edit modal; **Wave 9.2** (mandatory).
- **B1.3** — OSS DashboardScreen has no pipeline-by-stage / AR-aging-bucket widgets; `GetDashboardPipelineByStageYTD` / `GetDashboardARAgingReportYTD` remain uncalled. Add the widgets (then their drills) in a later wave.
- **PreviewOrderDeleteCascade** unwired — OSS Orders delete has no cascade preview (the sales keep-list's cascade-preview strength is absent on OSS). `GetPreparedByOptions` unwired (costing Prepared-By, Wave 9.3/9.5).
- **Cheque "Mark Cleared"** passes an empty `bankStatementLineID` — functionally clears (backend only checks status), but the clear isn't linked to a specific bank line; consider routing this through bank reconciliation instead. (Coder C flag.)
- **Legacy Draft-status DNs** created before B6 will still fail Dispatch's Prepared-only check — pre-existing data condition, not in B6 scope beyond the create-default fix.
- **DashboardScreen tokenization** — see the constitution deviation above.

## Open questions for the owner
1. **A3 severity acknowledgement** — the supplier-invoice Edit modal is a *live off-ledger settlement path* right now (mark Paid with no ledger/journal). Confirm this is acceptable to carry until Wave 9.2, or fast-track the control removal.
2. **DashboardScreen raw-hex deviation** — accept the reuse-existing-hex choice for this wave, or authorize a tokenization pass for that screen?
3. **PreviewOrderDeleteCascade** — the audit keep-list expects a cascade-preview delete on Orders, but OSS has none. Want it wired in a follow-up (backend already exists)?
