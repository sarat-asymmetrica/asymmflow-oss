# FABLE WAVE 9 — Critical UI/UX Flow Audit of the Deployed Application

**Date:** 2026-07-10 · **Auditors:** 7 parallel domain agents over `ph_holdings/frontend` (the deployed reference) · **Commissioned by:** Sarat
**Purpose:** Wave 8 closed backend feature parity. Before any further UI passes on `asymmflow-oss`, grade the *deployed* app's flows for UI/UX quality — so subsequent waves shore up the weak flows instead of faithfully porting their warts, and invest polish where flows are already excellent.

**Scope discipline:** This audit is the *flow-quality* layer. It deliberately excludes ground covered by prior audits: CSS token/color/spacing polish (`UI_UX_AUDIT_2026_06_16.md`), component-vocabulary sprawl (`UI_UX_POLISH_AUDIT_2026_06_20.md`), feature *reachability* re-wiring (`FEATURE_FLOW_AUDIT_FINDINGS.md`, Sprints 1–3.5), and backend correctness (Wave 8). All file:line citations are into `ph_holdings/frontend/src` as read on 2026-07-10; OSS screens are ~1:1 ports, so findings map directly — known OSS deltas are annotated inline as **[OSS]**.

**Rubric (each flow graded on all seven):** 1 task efficiency · 2 flow continuity (handoffs, no dead ends) · 3 information hierarchy (job-shaped, not schema-shaped) · 4 form ergonomics · 5 feedback & safety · 6 consistency · 7 complexity smells.

---

## 1. Verdict matrix

39 flows graded: **9 🟢 EXCELLENT · 16 🟡 ADEQUATE · 14 🔴 SUBOPTIMAL.** The owner's hypothesis is confirmed with precision: the sales pipeline is the app's gold standard (4 of its 5 flows 🟢), and the later-added domains score worst exactly where they *diverged from* the pipeline's patterns — People and Work are almost entirely 🔴.

| Domain | Flow | Verdict |
|---|---|---|
| **Sales pipeline** | RFQ/opportunity intake & pipeline mgmt | 🟡 |
| | Costing sheet creation & revision | 🟢 |
| | Offer create/revise/send/won/lost | 🟢 |
| | Offer→Order conversion | 🟢 |
| | Order lifecycle & DN/PO/Invoice handoffs | 🟢 |
| **Inventory & Ops** | Purchase order creation & lifecycle | 🟡 |
| | Goods receipt (PO→GRN→stock-in) | 🟢 |
| | Stock visibility ("what do we have / owe") | 🔴 |
| | Delivery note from order (closing artifact) | 🟡 |
| | Serial number trace | 🟢 |
| **HR / People** | Employee onboarding | 🔴 |
| | Profile & day-to-day HR admin | 🔴 |
| | Archive / deactivation | 🟡 |
| | Payroll run | 🔴 |
| | App users / roles vs employees | 🔴 |
| **Projects / Work** | Project creation & setup | 🔴 |
| | Allocating people/work to a project | 🔴 |
| | Day-to-day tracking (my plate / project X) | 🔴 |
| | Project administration & lifecycle | 🟡 |
| | Work/approvals routing to people | 🔴 |
| **Finance AR/AP** | Customer invoice creation & lifecycle | 🟡 |
| | Customer receipt & allocation | 🟡 |
| | Supplier invoice → match → approve → pay | 🟡 |
| | Supplier payment recording | 🔴 |
| | Expense entry | 🔴 |
| | FinanceHub tab IA | 🟡 |
| **Finance recon/acct** | Bank reconciliation (match) | 🟡 |
| | Book-vs-bank reconciliation (prove) | 🔴 |
| | Cheque register lifecycle | 🔴 |
| | FX revaluation run | 🟡 |
| | Accounting / financial statements | 🟡 |
| | FinancialDashboard daily monitoring | 🟢 |
| | Reports consumption + audit trail | 🔴 |
| **CRM + shell** | Global navigation IA | 🟡 |
| | Dashboard as morning-start surface | 🟡 |
| | Customer lookup → 360 → act | 🟢 |
| | Supplier 360 | 🟡 |
| | Data-quality review | 🟢 |
| | Notifications / Inbox | 🟡 |

---

## 2. The excellence rubric — what makes the sales pipeline good (transferable patterns)

Extracted from the baseline audit; these are the patterns every 🔴 fix should adopt, with PH evidence:

1. **Context-preserving cross-screen handoffs** (sessionStorage/store + dual nav events): finish A and land inside B with the job set up. `pendingCostingOpportunity` (OpportunitiesScreen:654-671 → CostingSheetScreen:831-857), `pendingCostingOffer`, `pendingDNCreate`, `pendingOrderOpen`, `pendingReceiptApply`, `pendingInvoiceFilter`.
2. **Status drives the next action** — render only the legal next step(s): Offers action column by `stage` (351-450); Order detail CTAs gated on `fulfillment_pct`/`invoicing_pct` (1690-1721); supplier invoice Match→Approve→Settle chain (SupplierInvoicesScreen:311-343); payroll approve→post→pay (PayrollScreen:607-609); bank-recon Finalize gate.
3. **First-class revision model, not overwrite** — costing revisions with `is_active` pointer (2011-2062); offer Re-quote (from Lost) + Renew (from Expired).
4. **Recoverable dead-ends** — credit-limit block opens a reason-captured override modal (OrdersScreen:810-858); expired offers get a tab + Renew.
5. **Conversion moments are single-purpose and guarded** — `MarkOfferWon` requires customer PO, then auto-creates the order (OffersScreen:984-1007).
6. **Crash-safe long work** — costing localStorage draft engine + `beforeunload` + Resume/Discard (589-733, 1856-1871).
7. **Integrity surfaced, not hidden** — no-items/zero-value/legacy-shell banners; delete-cascade preview; suggested-vs-override pricing with visible `user-overridden` state.
8. **Smart defaults + auto-fill** — costing pre-fills customer/contact/reference/division/line items from the opportunity; per-division delivery terms; Prepared-By from current user.
9. **One confirm primitive** — promise-based `askActionConfirmation` (OrdersScreen:318-344) + double-submit guards for every irreversible action.

---

## 3. Cross-cutting defect themes (the same disease in many organs)

These recur across domains; fixing them as *themes* (one pattern, swept app-wide) beats per-screen patching.

**T1 — Ghost actor identity [H].** Financial and compliance actions are attributed to hardcoded strings, making audit trails meaningless: GRN `received_by/qc_by = 'System User'` (GRNScreen:120,134,428); supplier approval `'System Admin'` (SupplierInvoicesScreen:171,809); bank recon finalize/match/unmatch `'admin'` (BankReconciliationScreen:528,802,821); book-bank finalize `'admin'` (:334); FX post `'admin'` (:375); audit-trail reverse `'admin'` (AuditTrailViewer:290); costing Prepared-By falls back to `'System'`. **[OSS]** Wave 8 P5-1 put real SoD in the supplier-approval *backend*; the frontend must now thread the authenticated user everywhere.

**T2 — Dead-end data displays [H].** Numbers that lead nowhere: dashboard KPI cards + pipeline donut + aging bars inert (DashboardScreen:946-953, 1038-1053, 1094-1102); dashboard task rows unclickable (:1116-1123); supplier-360 PO/invoice rows unclickable while the customer 360's ARE (SupplierDetailView:613-654); journal-audit rows dead (AccountingScreen:259-276); serial-trace references plain text (SerialTraceScreen:76-106); Alerts panel static (:988-995).

**T3 — Built-but-unreachable surfaces [H].** Working features users cannot reach: pending-fulfillment report fetched then discarded (OperationsHub:56,65 — the back-to-back trader's core question!); UserManagementScreen orphaned + permission key mismatch `users` vs `usermanagement` (App.svelte:604-612,795-811); InboxScreen orphaned AND its New/Review filters are `===` no-ops (InboxScreen:143-152); cheque Clear/Cancel/Stale handlers exist but wired to nothing (ChequeRegisterScreen:341-361,435-455); manual-GRN path never rendered; dead Offer create+edit modals (OffersScreen:1469-1581,1584-1738); dead item-aware `handleCreatePO` (OrdersScreen:860-892); ~490-line `{#if false}` PO modal block (PurchaseOrdersScreen:1022-1509); receipt "apply unapplied balance from the receipt workflow" points at a workflow that doesn't exist (PaymentsScreen:548-551).

**T4 — Free-jump status controls (the anti-pattern of excellence #2).** PO detail = flat 8-button status grid, every state one click away (PurchaseOrderWorkspace:152-161); supplier-invoice Edit modal exposes Status/PaymentStatus dropdowns that bypass the gated Match→Approve→Settle chain (SupplierInvoicesScreen:1561-1568); DN create exposes a status select the code sometimes ignores (DeliveryNotesScreen:1260-1264 vs 588); employee status `inactive` coexists with Archive as a second deactivation path (PeopleHub:701-710 vs 748-760).

**T5 — Native `confirm()`/`prompt()` one-offs.** Recurring crude dialogs where the canonical Modal exists (and native dialogs are hazardous in a WebView shell): supplier approve (SupplierInvoicesScreen:805), notification rejections (NotificationsScreen:251,271), supplier issue resolve (SupplierDetailView:297), contact delete (CustomerDetailView:195), bank-recon deletes (BankReconciliationScreen:959,983,1168).

**T6 — Hardcoded FX rates.** Stale currency maps baked into UI: PurchaseOrdersScreen:836-844; SupplierInvoicesScreen:690-699; supplier-payment create has NO rate field at all (SupplierPaymentsScreen:510-522 renders it in edit only).

**T7 — Duplicate paths to one job.** Three supplier-settlement paths (RecordSupplierPayment / MarkSupplierInvoicePaid / UpdateSupplierInvoiceWithPayment) across two screens; two live PO-creation paths (+1 dead) where only one carries line items; two divergent order line-item editors; three "+ Add Receipt" buttons on one screen; the expense Categories/Vendors managers rendered twice on one panel (ExpensesScreen:570-705); two reconciliation screens for one monthly task.

**T8 — Missing as-of dates.** Book-bank recon stamps `new Date()` (BookBankReconciliationScreen:240); FX revaluation likewise (FXRevaluationScreen:345,360). Month-end work cannot be dated month-end.

**T9 — Destructive/posting guard calibration is inconsistent.** FX *posts a journal entry* on bare row-click (FXRevaluationScreen:474) and "Revalue All" runs unconfirmed (:450), project Delete is one-click (WorkHub:1601) — while task delete is two-press and sales uses `askActionConfirmation` everywhere. One promise-based confirm primitive should back every irreversible action (excellence #9).

**T10 — Identity fragmentation (People).** HR employees, license keys, and login users/roles are three disconnected systems with no in-app explanation (PeopleHub:491-524 vs UserManagementScreen:768-856).

**T11 — Wrong-audience information hierarchy.** Sales metrics headline the HR profile editor (PeopleHub:619-632); Reports frames a Bahrain trader in SaaS vocabulary — RUNWAY/BURN/MRR (ReportsScreen:216-252); a 12-checkbox PDF field-visibility panel sits inside invoice *creation* (InvoicesScreen:1409-1479).

**T12 — Two P&Ls, no explanation.** AccountingScreen shows the live operational P&L while FinancialDashboard shows audited/Tally numbers for the same year; they don't reconcile and nothing says which is authoritative (AccountingScreen:119-122 vs FinancialDashboard:115).

---

## 4. Domain reports (full findings)

The complete per-domain reports — verdicts, justifications, defect lists with file:line and concrete fixes, keep-lists, and each domain's "biggest single UX win" — follow. **Keep-lists are binding:** any Wave 9 slice touching these screens must preserve those behaviors.

### 4.1 Sales pipeline (baseline) — audit-sales

**RFQ/opportunity intake & pipeline management — 🟡** Feature-rich (pipeline+RFQ merge deduped by folder, year filter, sort chips, KPIs, tender import) but the two-source data model leaks into the UI and the modals are overloaded. Weakest link of the flow.
- [M] OpportunitiesScreen.svelte:101-102,880 — deprecated stages (New/Qualified/Proposal/On Hold) offered as filters while cards collapse them all to "Pipeline"; filtering "New" returns cards labelled "Pipeline". Fix: drop deprecated stages from filters or show true stage.
- [M] OpportunitiesScreen.svelte:673-706 — create form collects Priority/received_date/due_date that `CreateRFQWithReference` silently discards. Fix: wire through or remove.
- [M] OpportunitiesScreen.svelte:517-593,927-1074 — Edit modal is ~25 fields in one flat grid (incl. free-text cost/profit disconnected from costing). Fix: section into Commercial/Classification/Financials/Notes.
- [M] OpportunitiesScreen.svelte:453-460 — change-log load errors masked as a permissions message. Fix: distinguish empty/error/unauthorized.
- [L] OpportunitiesScreen.svelte:1228-1239 — "Delete" and cascade "Delete All" side-by-side in identical danger styling. Fix: two-step confirm for cascade.
- [L] OpportunitiesScreen.svelte:1042 — line-item currency read-only even for manual items. Fix: editable when no OCR source.

**Costing sheet creation & revision — 🟢** The reference for "organized around the job": Excel-parity calculator, suggested-vs-override pricing, revisions, draft recovery, per-division defaults, progressive disclosure.
- [M] CostingSheetScreen.svelte:1751-1767 — costing persist inside Save-as-Offer fails silently (console.warn), offer saves, history doesn't, user sees success. Fix: warning toast.
- [M] CostingSheetScreen.svelte:159,439-445 — line items hard-capped at 10; longer seeded RFQs silently truncated. Fix: raise/remove cap.
- [M] CostingSheetScreen.svelte:1732-1839 — only persist path is "Save as Offer"; no standalone "Save costing". Fix: add explicit save.
- [L] CostingSheetScreen.svelte:1772-1774 — Save-as-Offer silently updates an existing offer. Fix: confirm on update path.
- [L] CostingSheetScreen.svelte:2104 — Prepared-By falls back to 'System' (T1). Fix: blank when unknown.

**Offer create/revise/send/won/lost — 🟢** Best-organized screen: stage tabs with counts, expiry enrichment, notes thread, follow-ups, conditional actions, structured lost reasons, requote/renew, permission-gated delete.
- [M] OffersScreen.svelte:655-666,1469-1581 — inline Create Offer modal is verified-unreachable dead code (New Offer correctly routes to Costing). Fix: delete.
- [M] OffersScreen.svelte:1082-1088,1584-1738 — inline Edit Offer modal likewise dead. Fix: delete.
- [M] OffersScreen.svelte:316-329,468-474 — two disagreeing expiry notions (client-computed `is_expired` vs `stage === 'Expired'` tab); an offer can render expired inside the Quoted tab. Fix: reconcile.
- [L] :2048 note placeholder says "opportunity" on an offer; :1786 "Successfully Closed" listed first under "Reason for Loss"; :1903-2099 xl View modal carries an editable header + costing table + notes in one surface.

**Offer→Order conversion — 🟢** Single-purpose, guarded, auto-creates the order.
- [L] OffersScreen.svelte:996-1001 — after Won, toast only; no "View Order" affordance. Fix: add one.

**Order lifecycle & handoffs — 🟢** Order detail is a command center: RFQ→Offer→Order traceability chain, delivery progress, linked invoices, state-gated CTAs, cascade-preview delete.
- [M] OrdersScreen.svelte:860-892,926-964,1695 — modal wires `CreatePOsFromOrder(order.id, [])` (all items) while the item-aware `handleCreatePO` is dead. Fix: keep+wire the item-aware one, delete the other.
- [M] OrdersScreen.svelte:762-768,1691-1698 — DN/Supplier-Order buttons look enabled on zero-item orders and fail only on click. Fix: disable + tooltip.
- [M] OrdersScreen.svelte:585-714,1439-1528 — manual order editor is a thinner divergent clone of Costing's line editor. Fix: converge on a shared component.
- [L] :415,423-432 `ListOrders(10000,0)` + N+1 delivery-status calls (~175 round-trips per open); :1327-1334 row "+ Invoice" unconfirmed while modal path confirms; :1699-1703 "Mark as Delivered" skips GRN with a weak warning.

**Keep list (sales):** the pending-store handoff pattern; status-driven conditional CTAs; suggested-vs-override pricing; the revision model; credit-limit override-with-reason; MarkOfferWon PO capture; integrity banners + cascade preview; costing draft recovery; won/lost edit-locking; order traceability chain.

### 4.2 Inventory & Operations — audit-inventory

**PO creation & lifecycle — 🟡** Inline workspace (no modal-in-modal) is good; flat 8-button status grid (T4), hardcoded FX (T6), and a misleading order-link that carries no items drag it down.
- [M] PurchaseOrderWorkspace.svelte:152-161 — 8-button free-jump status grid; user learns legal transitions by hitting backend errors. Fix: render only next legal transition(s).
- [M] PurchaseOrderWorkspace.svelte:178-185 — "Link to Order" pulls no line items; the item-carrying path is `CreatePOsFromOrder` on Orders. Two paths, one artifact, only one carries data (T7). Fix: populate items on link, or remove the dropdown and point to the Orders action.
- [M] PurchaseOrdersScreen.svelte:836-844 — hardcoded currency map (T6).
- [M] OrdersScreen.svelte:956 — after `CreatePOsFromOrder`, app navigates to the PO *list*, not the new draft. Fix: open the created PO + "approve to send" toast.
- [L] PurchaseOrdersScreen.svelte:1022-1509 — ~490-line `{#if false}` dead modal block (T3). Fix: delete.
- [L] PurchaseOrdersScreen.svelte:876-891 — `querySelectorAll` DOM-poking to toggle button state. Fix: data-driven disabled.

**Goods receipt (PO→GRN→stock-in) — 🟢** Best form in the domain: PO items loaded, remaining-qty defaults, partial-receipt columns, capped inputs, inline serial capture; the PO→GRN handoff pre-selects context (excellence #1).
- [M] GRNScreen.svelte:227-234 — "Complete" renders regardless of qc_status/completion; re-completable. Fix: hide once complete, gate on QC.
- [M] GRNScreen.svelte:120,134,428 — 'System User' attribution (T1). Fix: logged-in user.
- [L] GRNScreen.svelte:104,116-122 — manual-GRN path exists but is never rendered (T3). Fix: remove or wire.

**Stock visibility — 🔴** Effectively unanswerable in the shipped app. InventoryScreen is commented out of the hub ("no warehouse") — defensible for on-hand, but it also buried pending-fulfillment: the back-to-back trader's core question, already computed.
- [H] OperationsHub.svelte:10-11,29-30,183-184 — InventoryScreen unrouted anywhere; on-hand/movements/valuation/pending all unreachable.
- [H] OperationsHub.svelte:56,65 — `GetInventoryPendingFulfillmentReport(500)` fetched, result discarded (T3). Fix: "Fulfillment" tab.
- [M] InventoryScreen.svelte:264-303,321-456 — if re-surfaced: 13 raw inputs + hand-rolled tables; needs a canonical-components pass first.

**Delivery note from order (the closing artifact) — 🟡** Core handoff is 🟢 (store-based, pre-filled, guarded); dragged down by form ergonomics and a muddled status lifecycle at the close-the-loop moment.
- [H] DeliveryNotesScreen.svelte:492-495,1209-1216 — required Delivery Address never auto-fills from customer/order; retyped on every DN. Fix: default, editable.
- [M] :658-661 vs 1234-1248 — Dispatch requires driver+vehicle that create leaves optional → create→Dispatch→rejected→edit→Dispatch loop. Fix: prompt inline on Dispatch.
- [M] :1260-1264,588,616-621 — create-form status select that one path force-overrides to 'Prepared' (T4). Fix: drop the picker; lifecycle via detail actions.
- [M] :673-682,44-49 — Confirm Delivery hardcodes signature 'Auto-confirmed'; no POD capture; 'Signed' status unreachable. Fix: capture recipient name on Confirm.
- [M] :571-583 — serial path = create then patch-update; patch failure only console.warns (divergent create paths). Fix: single create call.
- [M] :1425-1443 — after Confirm Delivery, silence: no handoff to order fulfillment status or invoice creation. Fix: "Order fully delivered — create invoice?" when remaining hits zero.

**Serial trace — 🟢** Clean read-only search, real empty/loading states, warranty coloring.
- [M] SerialTraceScreen.svelte:76-106 — PO/GRN/DN/Invoice references not clickable (T2). Fix: deep-link each.
- [L] :168-170 — blank until query; a "recently delivered" default would help.

**Keep list (inventory):** Order→DN store handoff + no-items guard; PO→GRN pre-selecting row action; GRN remaining-qty defaulting/capping; `CreatePOsFromOrder` per-supplier split; PO inline workspace; serial trace as-is; `PageLayout embedded` single-create-button pattern.

**Biggest single win:** surface pending-fulfillment as an Operations "Fulfillment" tab — the data call and the table both already exist. **[OSS]** `GetInventoryPendingFulfillmentReport` + `GetInventoryMovementsWorkspace` were ported in Wave 8 Bucket C; this win is wiring-only on OSS.

### 4.3 HR / People — audit-people

**Employee onboarding — 🔴** Fragmented across three disconnected panels; create collects only name/department/title; no guidance to the fields that must follow.
- [H] PeopleHub.svelte:203-213 — after create, the half-finished profile sits below the fold; no scroll/focus. Fix: auto-scroll + "add contact & start details" toast.
- [H] PeopleHub.svelte:476-489 — create omits email/phone/start date/manager, forcing a second trip every hire. Fix: promote the essentials into the composer.
- [M] PeopleHub.svelte:491-524 — license access is a separate top-level composer (T10). Fix: fold into the employee detail.
- [M] PeopleHub.svelte:677-679 — email never format-validated. Fix: inline validation.

**Profile & day-to-day HR admin — 🔴** Schema-shaped monster: sales metrics headline the HR editor (T11); no visa/CPR/permit tracking for a Bahrain company; two overlapping deactivation concepts (T4).
- [H] PeopleHub.svelte:619-632 — Opportunities/Won-Lost/Revenue YTD above the edit form. Fix: move to contribution overview.
- [M] :701-710 vs 748-760 — status `inactive` vs Archive, two deactivation paths. Fix: Archive is the only deactivation; status = active/on_leave/probation/contract.
- [M] :566-826 — profile/task history/projects/access/archive in one unbounded scroll. Fix: tabs (Profile / Work / Access).
- [L] :78-91 — no visa/passport/CPR/permit fields (feature gap — owner decision).

**Archive / deactivation — 🟡** Safest flow here: reason required, reversible, history retained. But `requestEmployeeArchive` naming implies an approval mechanic the UI doesn't surface.
- [M] PeopleHub.svelte:284-313,749-759 — archives instantly despite request-flow naming; no reviewer queue visible. **[OSS]** P4 slice 4 added the approval ride on NotificationsScreen — on OSS the reviewer path EXISTS; verify discoverability instead.
- [L] :755-758 — one-click red Archive (reason-gated only); add confirm to match app patterns.

**Payroll run — 🔴** Mechanics sound (state-driven approve→post→pay, required refs); placement broken: Finance → Admin group → sub-tab, invisible to HR-only roles, disconnected from the employee record.
- [H] FinanceHub.svelte:262-269 — payroll buried + gated `finance:view`. Fix: surface in People, gate on an HR/payroll permission.
- [M] PayrollScreen.svelte:423-428 — comp profiles picked from a bare dropdown; no "Set up payroll" deep-link from the employee. Fix: add it.
- [L] PayrollScreen.svelte:371-373 — one long scroll; tab the three sections.

**App users/roles vs employees — 🔴** UserManagementScreen is orphaned + permission-mismatched (T3); three identity systems unexplained (T10).
- [H] App.svelte:604-612,795-811 — dead `usermanagement` route; permission keyed `users` which nothing emits. Fix: reachable entry + aligned key.
- [H] PeopleHub.svelte:491-524 vs UserManagementScreen.svelte:768-856 — employees / license keys / login users disconnected. Fix: employee record = single home (embed login+role, license).
- [M] UserManagementScreen.svelte:714-763,654-712 — Opportunity edit-conflict resolution & activity monitoring live here; surprising placement. Fix: move conflict resolution to sales admin.
- [L] UserManagementScreen.svelte:601-624 — roles are display-only while implying management. Fix: label read-only or add editing.

**Keep list (people):** payroll status-driven buttons; archive safety contract; directory search + Active/Archive/All; auto employee_code; payroll inline help; company/division scoping; `loadRequestSeq`/`assignmentLoadToken` race guards.

**Biggest single win:** the employee record becomes the single home for a person — embed login account/role + license assignment, deep-link payroll setup. Kills the orphan route, the three-identity confusion, and the fragmented onboarding in one move.

### 4.4 Projects / Work allocation — audit-work

**Project creation & setup — 🔴** Flat 9-field composer always visible; customer/POC fields shown for internal projects; `opportunity_id`/`order_id` supported by the model but never populated; no inbound handoff from Opportunity/Order; create dead-ends.
- [H] WorkHub.svelte:594-605 — no Opportunity→Project or Order→Project handoff anywhere (the benchmark's defining strength). Fix: "Start project" action on Opportunity/Order, preseeded.
- [H] WorkHub.svelte:1453-1495 — customer/POC block unconditional though `projectType` defaults internal. Fix: gate on type === customer.
- [M] WorkHub.svelte:586-645 — after create, no modal/prompt for members/tasks. Fix: open the project modal on the member step.
- [L] WorkHub.svelte:1489 — POC email unvalidated.

**Allocating people/work — 🔴** The domain's root defect: "project members" and "task assignees" are two disconnected mechanisms; membership is decorative.
- [H] WorkHub.svelte:1101/1228/1792 — assignee dropdowns list ALL employees, never members; adding a member changes nothing. Fix: membership drives the assignee list (or drop members).
- [M] :1356-1372 vs 1521-1533 — the same "Assignment Roster" has two hidden per-tab semantics (drag vs click-prefill). Fix: one interaction.
- [M] :660/1629 — one batch free-text "Role" for all selected members; `allocation_percent` in the model has no UI. Fix: per-person role/allocation.
- [L] :991-1038 — drag-and-drop assignment as a one-off; keep as accelerator only.

**Day-to-day tracking — 🔴** The worker gets a weaker surface than the manager; the dashboard's task list is inert; project rollups are wrong.
- [H] DashboardScreen.svelte:1116-1123 — dashboard task rows unclickable (T2). Fix: dispatch `openCollaborativeTask` like Notifications does.
- [H] WorkHub.svelte:1504 — every project row's task-count badge reads the *selected* project's tasks; unselected rows show wrong/zero. Fix: per-project counts.
- [M] :1242-1279 — "My Work" has no filters/sort/overdue bucket, strictly weaker than Team Board. Fix: overdue-first + status/focus filter.
- [M] :242/1257 — completed work hidden everywhere, no history toggle.

**Project administration — 🟡** Consolidated lifecycle actions + audit reason are right; guard calibration inverted (T9); archived projects vanish.
- [H] WorkHub.svelte:1601/959-989 — one-click project Delete while task delete is two-press. Fix: match the stronger guard.
- [M] :1599-1600 — Shelve vs Archive side-by-side, unexplained. Fix: helper text or merge.
- [M] :503-505/242 — `activeOnly=true` always; no archived view or restore path. Fix: Archived filter + restore.
- [L] :961 — audit reason defaults to a generic string, never required.

**Work/approvals routing — 🔴** Task notifications route well; approvals are fragile and the Inbox is broken.
- [H] NotificationsScreen.svelte:187-199,388,403 — Approve/Reject render only while unread; *reading* a pending approval strands it. Fix: gate on the request's pending state; add an approvals queue.
- [M] InboxScreen.svelte:143,150 — filter buttons use `===` (no-op) instead of assignment; filters do nothing.
- [M] NotificationsScreen.svelte:251,271 — `window.prompt()` for rejection reasons (T5).
- [L] :236-240 — 80ms `setTimeout` navigation race; consume a pending-task key on mount instead.
- [L] InboxScreen.svelte:63-78 — classified documents never become tasks (no ContextTaskModal wire).

Cross-cutting: [M] WorkHub.svelte:1683 vs 1723 — modal-over-modal (task atop project); WorkHub is a ~2600-line multi-responsibility monster with nine `savingX` flags.

**Keep list (work):** ContextTaskModal's context-passing (customer/opportunity/order prefill — benchmark quality); Notifications "Open task" deep-link with pre-seeded payload; task-delete two-press + block-reason guard; Team Board lanes/filters + drag accelerator; optimistic snapshot caching / requestSeq guards.

**Biggest single win:** unify members and assignees into one concept — membership drives the assignee list and per-project workload rollups. Collapses the domain's root confusion and fixes "state of project X" in one change.

### 4.5 Finance AR/AP — audit-finance-arap

**Customer invoice creation & lifecycle — 🟡** Coherent lifecycle + a genuinely good credit-override; spoiled by over-disclosure at creation and an order-only constraint.
- [M] InvoicesScreen.svelte:1409-1479 — 12-checkbox PDF field-visibility panel inside creation (T11). Fix: move behind "Customize fields" at the PDF step.
- [M] :816-822 — invoice creation refuses to open without an unfulfilled order; no scratch/proforma path, no explanation. Fix: blank-invoice path or explanatory empty state.
- [L] :1315-1351 — credit notes in a hand-rolled table vs canonical DataTable.
- [L] :634-641,657-665 — overdue invoices have no chase/reminder affordance.

**Customer receipt & allocation — 🟡** The apply-to-invoice path is the domain's best form (auto-fill to outstanding, Apply Full Open Balance, live full/partial/over helper + overpay prevention). Undermined by the on-account dead-end.
- [H] PaymentsScreen.svelte:548-551,424 — unapplied on-account balances have NO later-apply UI; the error message points at a nonexistent workflow (T3). Fix: "Apply unapplied balance" row action opening the apply modal pre-scoped. **[OSS]** PC-D7 ported unapplied-remainder transforms; this UI gap is the visible half.
- [M] :293-296,432 — "Avg Days to Collection" corrupted by receipts hardcoded `days_to_payment: 0`. Fix: exclude receipts.
- [M] :591-594 — mistaken on-account receipt cannot be voided/corrected. Fix: allow reversal.
- [L] :427,194-201 — misleading Invoice#/0d cells on receipt rows; [L] :768,827,869 — three buttons, one modal, three labels.

**Supplier invoice → match → approve → pay — 🟡** The gated Match→Approve→Settle chain IS the benchmark pattern; spoiled by ghost approver (T1), stale FX (T6), a bypass path (T4), and native confirm (T5).
- [H] SupplierInvoicesScreen.svelte:171,809 — approver hardcoded 'System Admin'. **[OSS]** backend SoD (creator≠approver) shipped in Wave 8 P5-1 — the frontend must pass the real user or approvals will misattribute/fail.
- [M] :690-699 — hardcoded FX map. Fix: FX service.
- [M] :1329-1346,663-670 — independently editable header Subtotal/VAT can disagree with lines. Fix: derived totals.
- [M] :1561-1568,959-978 — Edit modal exposes Status/PaymentStatus dropdowns bypassing the gated chain. Fix: remove state-advancing controls from Edit. **[OSS]** with P5-1's Approved-only gate these bypasses now produce hard errors — removal is mandatory, not cosmetic.
- [M] :805 — native `confirm()` for approval.
- [L] :1500-1506 — 3-way-match progress modal is an instant-closing spinner; result only in a toast. Fix: show the per-leg match result in place.

**Supplier payment recording — 🔴** Three settlement paths across two screens (T7), no create-time FX field, no overpay guard.
- [H] SupplierPaymentsScreen.svelte:260-315 vs SupplierInvoicesScreen.svelte:831-854,975-981 — RecordSupplierPayment / MarkSupplierInvoicePaid / UpdateSupplierInvoiceWithPayment: three code paths, one intent. Fix: one gated Settle modal.
- [M] :510-522,286-291 — exchange-rate field renders in edit only; non-BHD creates are blind.
- [M] :260-269 — nothing prevents overpaying an invoice. Fix: cap against outstanding like the AR flow.
- [L] :167-184,438-439 — paid expenses mixed into the same grid as supplier payments.

**Expense entry — 🔴** Setup UI rendered twice pushes the actual entry form below the fold; approvals live in a different tab group.
- [H] ExpensesScreen.svelte:570-645,647-705 — Categories+Vendors managers rendered twice before the entry form at 707. Fix: one manager behind Setup; lead with Quick Entry.
- [M] FinanceHub.svelte:74-79,265-266 + ExpensesScreen.svelte:922-960 — submit→approve loop crosses tab groups. Fix: approvals queue inside the Expenses workspace.
- [L] ExpensesScreen.svelte:707-737 — flat 9-control grid; group money vs classification.

**FinanceHub tab IA — 🟡** Nine tabs, broadly job-shaped, with sensible Bank&FX/Admin grouping and the PH/AHS selector. Headline: AP intake isn't here at all.
- [H] FinanceHub.svelte:5-21,52-64 — SupplierInvoicesScreen absent from Finance; bill intake/match lives in another hub while settlement lives here — the bookkeeper's AP loop is fractured. Fix: bring supplier invoices into Finance beside Supplier Settlements.
- [M] :52-79 — ~14 real destinations across two depths; Approvals hidden under Admin, away from Expenses.
- [L] :54-56 — asymmetric vocabulary ("Receipts" vs "Supplier Settlements"). Fix: consistent AR/AP naming.

**Keep list (AR/AP):** the Apply-Receipt handoff (`pendingReceiptApply` → pre-selected invoice + pre-filled amount); apply-receipt ergonomics; the gated Match→Approve→Settle chain; confirm-twice + posted/paid locks; PH/AHS `matchesCompany` scoping.

**Biggest single win:** unify supplier AP — supplier invoices into FinanceHub next to settlements, three payment paths collapsed into the one gated Settle action. **[OSS]** P5-1 already made the backend single-path; the frontend collapse completes it.

### 4.6 Finance reconciliation / accounting / reports — audit-finance-recon

**Bank reconciliation — 🟡** Strongest-built screen in the domain (split view, auto-match, split-allocation manual match, Finalize gate) but a 1650-line seven-modal monster hosting unrelated account CRUD, with ghost actors.
- [H] BankReconciliationScreen.svelte:528,802,821 — finalize/match/unmatch actor = 'admin' (T1).
- [M] :841 — statement import commits with no parsed-row preview. Fix: preview/confirm step.
- [M] :576 — six large fetches per line-click in the match modal. Fix: cache per session.
- [M] :878,1267 — bank-account CRUD buried here; move to Settings/Admin.
- [L] :959,983,1168 — native confirm() deletes (T5); [L] :1236 — status string in a KPI slot.

**Book-vs-bank reconciliation — 🔴** The domain's biggest problem: two sibling tabs are two halves of ONE monthly task, unlabelled, unlinked — and this half cannot actually be completed.
- [H] BookBankReconciliationScreen.svelte:513-529 vs 399-404 — Deposits-in-Transit and Outstanding Cheques are displayed but have NO input fields; `GetDepositsInTransit`/`GetOutstandingCheques` imported, never called. A real rec can't balance.
- [H] :240 — reconciliation date always `new Date()`; no as-of month-end (T8).
- [M] FinanceHub.svelte:68-69 — name the two-step sequence ("1. Match transactions / 2. Prove balance") and cross-link.
- [M] :334 — 'admin' actor (T1). [L] :503-565 — duplicated New/Edit adjustment grid.

**Cheque register — 🔴** Issuance is well done; the lifecycle is unreachable.
- [H] ChequeRegisterScreen.svelte:27,341-361,435-455 — MarkChequeCleared never called; stale/cancel handlers wired to nothing; no actions column. Issued→Cleared dead-ends (T3). Fix: row-action menu.
- [M] :311-339 — free-text payee, no link to supplier/settlement; cheques disconnected from AP.
- [L] :448-454 — Stale tab reuses Outstanding columns; relationship unclear.

**FX revaluation — 🟡** Good exposure/gain-loss KPIs; posting is dangerously unguarded.
- [H] FXRevaluationScreen.svelte:474 — bare row-click POSTS a journal entry (T9). Fix: explicit confirmed button.
- [M] :450,357 — unconfirmed "Revalue All"; :482-500 — update-rate modal shows no current rate ("was X → now Y"); :345,360 — no as-of date (T8); :375 — 'admin' actor (T1). [L] :421 — cryptic dual-figure KPI.

**Accounting / statements — 🟡** The best "how's the business doing" surface (P&L/BS/ledgers/journal + date presets + company toggle) — the owner CAN read a P&L. Undercut by:
- [H] AccountingScreen.svelte:119-122 vs FinancialDashboard.svelte:115 — two non-reconciling P&Ls in one hub (operational vs audited/Tally), no authority label (T12).
- [M] :217-220 — status filter is free-text requiring exact strings. Fix: dropdown.
- [M] — no export anywhere for BS/GL/journal (ReportsScreen dropped them); owner can't send statements to their accountant.
- [M] :259-276 — journal rows don't expand to double-entry lines (T2).

**FinancialDashboard — 🟢** The bright spot, mirroring the benchmark: Cash→bank-recon, AR→filtered invoices, aging buckets→matching invoice filters via `pendingInvoiceFilter`. Refinements only:
- [M] :355,363,371 — ratio health colors hardcoded green regardless of value.
- [M] :126-176 — backend failure silently renders a hardcoded FY2024 dataset under the selected year. Fix: explicit degraded state.
- [M] :248-273 — Revenue/NetProfit KPIs not drillable while Cash/AR are.
- [L] :595-605 — hardcoded per-year YoY prose.

**Reports + audit trail — 🔴**
- [M] ReportsScreen.svelte:254 — the whole catalog (incl. Sales/Ops packs) renders only under the "financial" tab.
- [M] :216-252 — RUNWAY/BURN/MRR SaaS framing for a Bahrain trader (T11).
- [M] :277-290 — non-financial categories dump auto-labelled numeric KPIs.
- [M] AuditTrailViewer.svelte:313,263 — "Audit Trail" is bank-rec-only; "who changed this invoice" unanswerable; no recent-activity view.
- [M] :379 — row-click opens the REVERSE modal (view reads as undo) (T9). [L] :290 — 'admin' reverser (T1).

**Keep list (recon/acct):** FinancialDashboard drill-throughs (the template for this domain); bank-recon Finalize/Reopen gating + split-allocation match modal + Fix-Debit/Credit path; Accounting date presets + statement layout; ReportsScreen only advertising real exports; cheque next-number preview.

**Biggest single win:** merge the two recon screens into one guided "close the month" flow — expose the two-step sequence, add DIT/outstanding-cheque inputs + as-of date, pull outstanding cheques from the register, link each step to the next.

### 4.7 CRM + global shell — audit-crm-shell

**Global navigation IA — 🟡** 10 destinations, lifecycle-shaped sidebar — a reasonable map. But "Relationships" hides the CRM, two screens are unreachable, and a second nav ordering contradicts the sidebar.
- [H] EnterpriseSidebar.svelte:246-254 / EnterpriseHeader.svelte:47-52 — no logout/lock/switch-user anywhere in the shell. **[OSS]** Wave 6 shipped logout + inactivity timeout — verify the OSS shell exposes them via the avatar/user menu, then treat as addressed.
- [M] App.svelte:621-631 vs EnterpriseSidebar.svelte:115-134 — Alt+N shortcut order ≠ sidebar order; Settings missing from the array so its header falls back to "Workspace". Fix: one shared nav list.
- [M] App.svelte:612,810 — orphaned `usermanagement` route + `users` permission-key mismatch (also in People report).
- [L] "Relationships" → "CRM"/"Customers & Suppliers"; [L] Deployment reachable only via a Settings sub-tab.

**Dashboard as morning-start — 🟡** Role-adaptive KPI/focus/alerts split is genuinely good; Operating Focus buttons deep-link correctly. But the biggest widgets dead-end (T2).
- [H] DashboardScreen.svelte:946-953 — the four headline KPI cards are inert. Fix: Revenue→finance, AR→invoices, Pipeline→opportunities, Cash→bank-recon.
- [M] :1038-1053 — pipeline donut/rows hover-only; no stage-filtered drill into Opportunities. **[OSS]** `GetDashboardPipelineByStageYTD` shipped in Wave 8 — wire both at once.
- [M] :1094-1102 — aging bars display-only. **[OSS]** `GetDashboardARAgingReportYTD` shipped in Wave 8; FinancialDashboard already proves the drill pattern.
- [L] :988-995 — static Alerts panel echoes Operating Focus without actions.

**Customer lookup → 360 → act — 🟢** Continuous, context-preserving, rich: drill-through rows with preselect, full-master edit load, guarded delete, lazy Insights.
- [M] CustomerDetailView.svelte:381-387 — you can view everything but *start* nothing: no "New RFQ / New Order / New Invoice for this customer". Fix: preseeded New-RFQ action.
- [L] :195 — native confirm() on contact delete (T5); [L] — 1266-line multi-responsibility screen (maintenance note).

**Supplier 360 — 🟡** Structurally parallel to the customer 360, minus its continuity.
- [H] SupplierDetailView.svelte:613-654 — PO/invoice rows NOT clickable (the customer twin's are) (T2). Fix: mirror the drill-through.
- [M] :297 — native prompt() for issue resolution (T5). [L] — no "New PO for this supplier" action.

**Data-quality review — 🟢** Queue + history, filters, every row actionable (Open/Reviewed/Resolve/Dismiss + note), smart in-hub routing, full loading/empty/error states.
- [L] two routing mechanisms (in-hub vs bubbled navigate); [L] dense per-row action column at scale.

**Notifications / Inbox — 🟡** Notifications is solid and actionable (day grouping, admin-gated approve/reject, task deep-links). Inbox is not a live surface.
- [H] InboxScreen — not in `screenLoaders` (App.svelte:598-614), no sidebar entry: orphaned dead code overlapping the Capture/OCR flow (T3). Fix: wire deliberately as document triage, or remove.
- [M] NotificationsScreen.svelte:251,271 — native prompt() rejections (T5).
- [L] InboxScreen.svelte:143-152 — broken `===` filters if ever surfaced.

**Keep list (CRM/shell):** role-adaptive dashboard; Operating Focus deep-links with tab/company params; customer-360 drill-throughs + full-master edit; guarded soft-delete surfacing backend block reasons; data-quality act-on-it column + routing; global drag-drop Capture affordance; CRMHub master-detail pattern.

**Biggest single win:** make the dashboard's numbers launchpads — KPI cards + pipeline stages + aging buckets drill into their already-existing filtered destinations. The routing plumbing exists; on OSS the two YTD backends shipped in Wave 8.

---

## 5. Proposed Wave 9 plan of action

Sequenced by leverage; each slice is frontend-dominant on `asymmflow-oss` unless noted. Cadence per slice: `feat(...)` → gate (`npx vite build` + `svelte-check`; `go test .` only when Go changes) → `docs(wave9)`. Every slice must respect the domain keep-lists in §4.

### Wave 9.0 — OSS ground-truth check (half-slice, read-only)
Findings above are from deployed PH; before fixing, verify the handful of known-divergent spots on OSS: logout/user-menu (Wave 6), employee-archive approval surface (P4), supplier settlement paths vs the P5-1 Approved-only gate (the Edit-modal bypass likely now hard-errors — a live UX bug on OSS worse than on PH), and which Wave 8 backends (fulfillment report, YTD dashboards, CreatePOsFromOrder, PreviewOrderDeleteCascade, GetPreparedByOptions) are still unwired.

### Wave 9.1 — Kill the dead ends (quick wins, ~1 slice)
Highest value-per-line in the audit; all frontend glue, several powered by Wave 8 backends:
1. Dashboard drill-throughs: KPI cards, pipeline stages, aging buckets, task rows (T2) — wire to `GetDashboardPipelineByStageYTD` / `GetDashboardARAgingReportYTD`.
2. Operations "Fulfillment" tab from the already-fetched pending-fulfillment report (the inventory domain's biggest win).
3. Supplier-360 row drill-throughs + "New PO" action; customer-360 "New RFQ" action.
4. Cheque lifecycle row actions (wire the existing Clear/Cancel/Stale handlers).
5. Serial-trace reference deep-links.
6. DN delivery-address auto-fill + driver/vehicle prompt on Dispatch + drop the create-status picker.
7. `CreatePOsFromOrder` → open the created PO (not the list); wire the item-aware handler; disable zero-item CTAs.
8. Dead-code deletion: Offers create/edit modals, PO `{#if false}` block, dead `handleCreatePO`; decide Inbox (wire as document triage or retire) and fix/remove its `===` filters.
9. Native `confirm()`/`prompt()` → canonical Modal sweep (T5).

### Wave 9.2 — One job, one path (money flows)
1. Supplier AP unification: SupplierInvoices into FinanceHub beside Settlements; collapse the three settlement paths into the gated Settle action; remove the Edit-modal status bypass (mandatory post-P5-1); real approver identity; FX from service; derived header totals; in-place 3-way-match result.
2. Receipts: "Apply unapplied balance" row action + receipt void/reversal + KPI fix + labels.
3. Expenses: single manager behind Setup, Quick-Entry-first, approvals adjacent.
4. Invoice creation: PDF-visibility panel out of the create modal; explained empty state (or proforma path — owner call).
5. Supplier payments: create-time FX field + overpay cap.
6. FinanceHub IA: AR/AP naming symmetry; consider promoting Approvals.

### Wave 9.3 — Close the month (recon + trust)
1. Unify bank + book-bank recon into one guided two-step flow; DIT/outstanding-cheque inputs; as-of date pickers (also FX reval); pull outstanding cheques from the register; statement-import preview.
2. Actor-identity sweep (T1): thread the authenticated user through GRN, supplier approval, bank/book recon, FX, audit reverse, costing Prepared-By.
3. Guard calibration (T9): FX post/Revalue-All behind explicit confirm; audit-trail row-click = details, reverse = distinct action; project delete two-press.
4. The two-P&L problem: label operational vs audited sources, one-line explanation (owner input on authority).
5. Statement/GL export for the accountant.
6. Reports: per-category catalogs; replace RUNWAY/BURN/MRR with trader vocabulary.

### Wave 9.4 — People & projects re-model (the big re-wire)
1. Employee record = single identity home: embed login user/role + license assignment; reachable Users/Access surface (fixes the orphan route + permission key); onboarding as one continuous flow (essentials in the composer, scroll-to-detail, tabbed detail panel); Archive as the only deactivation.
2. Payroll surfaced in People (permission story included) + "Set up payroll" deep-link from the employee.
3. Members = assignees: membership drives the assignee list + per-project workload rollups; fix the project task-count badge; per-person role/allocation.
4. Project creation: type-gated form; "Start project" handoff from Opportunity/Order (populate `opportunity_id`/`order_id`); post-create member step.
5. My Work upgrades (overdue-first, filters, completed toggle); approvals decoupled from read-state + approvals queue; archived-projects view + restore.

### Wave 9.5 — Polish the excellent (invest in strengths)
Sales residuals (expiry reconciliation, edit-modal sectioning, create-form field wiring, costing 10-line cap + standalone Save Costing + persist-failure toast, View Order after Won); order/costing line-editor convergence; PO next-legal-transition actions; GRN attribution/Complete gating; DN POD capture + delivered→invoice prompt; nav IA (label, Alt-order, shared nav list); opportunity intake cleanups; FinancialDashboard refinements (ratio thresholds, degraded state, drillable Revenue/NetProfit).

**Deliberately NOT in Wave 9:** visa/permit tracking (new feature — owner decision), audit-trail generalization beyond labeling (backend scope), report-pack redesign beyond categorization/vocabulary, WorkHub/CustomerDetailView structural decomposition (do opportunistically within slices).

---

*Full raw domain reports preserved at session scratchpad `audit-{sales,inventory,people,work,finance-arap,finance-recon,crm-shell}.md`. This document is the ground truth for Wave 9 planning; update verdicts in place as slices land.*
