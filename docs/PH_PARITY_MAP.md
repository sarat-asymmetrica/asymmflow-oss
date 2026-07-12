# PH Parity Map — full feature-and-flow surface (Mission G / Wave 4)

**Deliverable of PH Convergence Wave 4** (`FABLE_PH_CONVERGENCE_W4_HANDOFF.md`,
Mission G). A systematic, honest classification of deployed PH's entire surface
against the OSS substrate, produced 2026-07-07 by nine parallel read-only
measurement passes (the six-pass discipline that produced the Mission A ledger),
one per domain, over both repos.

- **Deployed PH (source of truth, read-only):** `C:\Projects\asymmflow\ph_holdings`,
  branch `ui-ux-hardening` @ `ca24372` (frozen; zero commits since 2026-07-04 —
  Freeze-Law PASS re-verified at wave start). Wails v2 + Svelte, ~278 root .go
  files, ~1381 App methods, 40 `*Screen.svelte`.
- **Substrate OSS:** `main` @ `801f41a` (this wave forked
  `feat/fable-ph-convergence-w4`). 234 root .go files, 1195 App methods, 39
  screens, plus a decomposed `pkg/` (crm, finance, kernel, approvals, inventory,
  sync, compliance, documents, overlay, …).

## Parity definition (handoff §2)
Parity = feature/flow **coverage**, not bug-for-bug behavioral identity. Each
surface element is exactly one of: **PRESENT** (equivalent flow, behavior
matches) · **DIVERGENT-INTENTIONAL** (differs by design; registered, not a gap) ·
**DIVERGENT-GAP** (genuine hole — missing validation, wrong number, absent step) ·
**ABSENT** (PH has a flow OSS lacks) · **EXTRA** (OSS has a flow PH lacks;
substrate value, no action).

## Summary (315 surface elements classified)

| Domain | PRESENT | DIV-INTENTIONAL | DIV-GAP | ABSENT | EXTRA |
|---|---|---|---|---|---|
| Finance / invoicing | 22 | 2 | 4 | 1 | 3 |
| Procurement | 27 | 2 | 4 | 0 | 1 |
| CRM (parties/contacts) | 26 | 3 | 0 | 3 | 0 |
| Opportunities / costing / offers | 52 | 0 | 1 | 11 | 0 |
| Inventory / stock | 13 | 0 | 1 | 0 | 0 |
| Documents / OCR / inbox | 15 | 0 | 2 | 6 | 0 |
| Reporting & PDF | 20 | 2 | 9 | 8 | 0 |
| Settings / RBAC / users / sync | 26 | 3 | 0 | 1 | 1 |
| Startup / schema-provision | 21 | 4 | 15 | 4 | 2 |
| **Total** | **222** | **16** | **36** | **34** | **7** |

**Reading the counts honestly:**
- **222 PRESENT** across the whole surface — the substrate carries the
  overwhelming majority of PH's flows, most byte-identical (Mission E already
  proved invoices reconcile to the fils).
- **The 15 DIV-GAP in Startup are a single structural finding**: the
  bank-reconciliation suite + FX suite + `vat_returns` (models already compiled
  and wired into live OSS services) are in **no boot migration set**, so a
  from-zero OSS DB never creates 15 tables. One-line-per-table registration fix
  (closed in G.3). Provisioning empty tables is in-scope parity (handoff §4 G.1);
  the *data* that flows into them is Mission-H deferred.
- **Zero financial-number divergences** were found on any core document — no case
  where OSS and PH compute a different invoice/PO/payment value. The gaps are
  missing controls, stale status labels, absent features, and appearance — not
  arithmetic. This is the strongest single parity signal in the map.

**Load-bearing correction to the handoff (the ground wins):** handoff §3.2
pre-ratified party-delete orphaning as a KEEP-OSS DIVERGENT-INTENTIONAL, on the
premise that "OSS orphans a party's transactional children." **That premise is
stale.** Wave 1 (PC-D1) moved the identical `CUSTOMER_HAS_LINKED_RECORDS` /
`SUPPLIER_HAS_LINKED_RECORDS` child-count guard into `pkg/crm/customer/delete.go:29,60`
— OSS now **blocks** the orphaning delete exactly as PH does. Party-delete is
classified **PRESENT (parity)**, not divergent. The pre-decided call is moot;
recorded here so a future Commander sees why.

---
## Finance / Invoicing

Repos: PH = `C:\Projects\asymmflow\ph_holdings` (read-only truth), OSS = `C:\Projects\asymmflow\asymmflow-oss`.
General shape: OSS is a near-line-for-line fork of PH's root finance services, PLUS a `FinanceService` wrapper (`service_finance.go`, ~1075 lines of thin delegators) and a few extracted engines (`pkg/documents/numbering`, `pkg/finance/{invoice,payment,fx}`, `pkg/approvals`, overlay identity). Most invoice math is byte-identical. The real divergences cluster in the **customer payment / settlement-status** path and the **customer-receipt** entity.

| Element (flow) | Classification | PH evidence | OSS evidence | Action |
|---|---|---|---|---|
| Create invoice from Order / DN / with-options | PRESENT | customer_invoice_service.go:447,566,575,602 | customer_invoice_service.go:460,562,571,594 + createInvoiceWithOptionsEx:633 (adds approvals/numbering/audit) | none |
| Create Proforma invoice (no outstanding) | PRESENT | customer_invoice_service.go:485 | customer_invoice_service.go:481 | none |
| Credit-limit override on create (pkg/approvals) | PRESENT (shipped W1-3) | ...WithCreditOverride:467 | :604,612 + approvals import | none |
| Draft server-derived totals + zero-rated recovery | PRESENT (shipped) | createInvoiceWithOptions body | createInvoiceWithOptionsEx:633 | none |
| VAT computation on invoice (10% BHD basis, roundInvoiceMoney) | PRESENT | roundInvoiceMoney:51; VAT in createInvoice | roundInvoiceMoney:59; identical | none |
| Update customer invoice (field-mask / Paid-guard) | PRESENT (shipped) | UpdateCustomerInvoice:1182 (Paid-outstanding guard) | :1231 (identical "cannot set Paid" guard :1277) | none |
| Delete customer invoice (reverse invoiced qty) | PRESENT | DeleteCustomerInvoice:1331 | :1375 + reverseInvoicedQuantities:1388 | none |
| Send customer invoice (hollow-invoice block) | PRESENT (shipped) | SendCustomerInvoice:1400 | :1417 | none |
| Invoice numbering / sequence (INV-{date}-{seq}, row-locked) | PRESENT (mechanism diverges, format byte-identical) | generateInvoiceNumberWithTx:1491 (inline InvoiceSequence lock) | GenerateInvoiceNumber:1486 delegates to numbering.New(...).Next | none (but see RBAC note) |
| RBAC on GenerateInvoiceNumber | DIVERGENT-GAP (minor) | requirePermission("invoices:create") at :1472 | OSS :1486 dropped the permission check | add invoices:create guard |
| Invoice HMAC hash + backfill + verify | PRESENT (shipped) | computeDocumentHMAC:28, backfillInvoiceHashesInternal:72, VerifyInvoiceHash:122 | :36,:80,:128 (identical; salt-guard preserved) | none |
| Backfill invoice items from orders (repair) | PRESENT | backfillInvoiceItemsFromOrdersInternal:254 (inline) | :253 refactored into repairInvoiceItemsFromOrder:307 + RBAC | none |
| Credit note create + over-credit guard (tx + row lock) | PRESENT | CreateCreditNote:35 | :37 (identical logic) | none |
| Credit note Issue / Apply workflow | PRESENT | IssueCreditNote:201, ApplyCreditNote:235 | :203,:237 (byte-identical apply) | none |
| Credit note numbering (CN-{date}-{seq}) | PRESENT (mechanism diverges) | GenerateCreditNoteNumber:306 inline seq | :308 delegates to numbering engine (byte-identical fmt) | none |
| Record customer payment (RecordPayment) | PRESENT (with guard divergence below) | payment_service.go:29 | payment_service.go:30 → recordPayment:34 | none |
| Non-payable status guard on RecordPayment | PRESENT (Mission G/I) | uses canRecordCustomerInvoicePayment (blocks Draft + all closed statuses) payment_service.go:77 | RecordPayment (Mission G) + updatePayment/RecordPartialPayment (Mission I W6 — inline maps replaced with the policy) | none — pinned by mission_i_band0_test.go |
| Post-payment status derivation | PRESENT (Mission I W6) | applyCustomerInvoicePaymentState → settlement policy | identical: RecordPayment/UpdatePayment/RecordPartialPayment/MarkPaid/MarkOverdue route through applyCustomerInvoicePaymentState; MarkPaid additionally creates the Payment audit row (PH parity) | none — golden tests in mission_i_band0_test.go |
| Settlement-status hydration on READ (Overdue auto-transition, outstanding rounding) | PRESENT (policy ported Mission G; all 7 read paths wired Mission I W6) | customer_invoice_payment_policy.go called on 7 read paths | identical: List, GetByID, Update-load, byCustomer, Overdue, Unpaid, byStatus all hydrate | none |
| RecordPartialPayment (idempotent, row-locked) | PRESENT | customer_invoice_service.go:1716 | :1701 (byte-identical incl. idempotency key) | none |
| Update / Delete payment (balance recalc + audit) | PRESENT | payment_service.go:249,549 | payment_service.go:284,511 (via pkg/finance/payment engine) | none |
| Customer Receipt entity + multi-invoice allocation | ABSENT | receipt_service.go: CreateCustomerReceipt:27, ApplyCustomerReceiptToInvoice:134, GetCustomerReceiptAllocations:212, applyCustomerReceiptToInvoiceTx:229, receipt numbering:318 | only referenced in pkg/data/phimport (import-time receipts→payments transform, PC-D7); no runtime CreateCustomerReceipt / allocation flow | decide: port receipt entity or confirm payments-only is the intended model (neither repo has a Receipt UI screen) |
| E-invoice UBL 2.1 XML | PRESENT (OSS improved) | einvoice_service.go:32 hardcoded phTrading TRN/name/addr | :28 supplier identity dispatched per-division via companyDocumentProfile(invoice.Division) | none — DIVERGENT-INTENTIONAL (overlay multi-division identity) |
| VAT return CSV export (excl Cancelled/Void/Proforma/Draft) | PRESENT (OSS improved) | ExportVATReturnData:240, label "(10%)" hardcoded, TRN hardcoded | :255, label uses overlay DefaultVATRate, TRN from companyDocumentProfile | none — DIVERGENT-INTENTIONAL |
| AR aging buckets / GetARAgingByCustomer | PRESENT | CalculateARAgingBuckets:1943, :2069 | :1951, :2066 | none |
| Overdue / Unpaid / by-status / by-date queries | PRESENT | :1621,1654,1685,1868 | :1630,1654,1676,1876 | none |
| Late-payment tracking & history | PRESENT | GetLatePaymentInvoices:2089, TrackLatePaymentHistory:2123 | :2086,:2120 | none |
| Currency handling on invoices (BHD 3-dp, BHDPrecisionMultiplier) | PRESENT | roundInvoiceMoney + customerInvoiceOutstandingBHD | roundInvoiceMoney:59 (identical); no outstanding-rounding helper (folded into hydrate gap) | none |
| Credit note PDF signature block | PRESENT (Mission I W7, I-18) | resolvePreparedBySignatureBlock + drawPHSignaturePDFLines (canonical signer list) credit_note_service.go:543 | ported byte-for-byte per D-I-3; signer identities via overlay `signature_blocks` (synthetic defaults) — offer_signature_blocks.go | none |
| FinanceService wrapper layer | EXTRA | — | service_finance.go (whole file) | none |
| pkg/finance/payment generic Service + pkg/finance/invoice delete engine | EXTRA | — | pkg/finance/payment/service.go, pkg/finance/invoice/delete.go | none |
| InvoiceListVM endpoint | EXTRA | — | invoice_list_vm_endpoint.go, service_finance.go:813 | none |

**Counts (post Mission I W7):** PRESENT 27 · DIVERGENT-INTENTIONAL 2 (einvoice identity, VAT-return label) · DIVERGENT-GAP 0 · ABSENT 1 · EXTRA 3

**Top gaps to close (money/integrity first):**
- ~~Draft invoices payable~~ · ~~no hydration on read~~ · ~~GenerateInvoiceNumber RBAC~~ — **all closed** (Mission G + Mission I W6, golden-tested).
- **Customer-receipt entity + allocation is ABSENT in OSS**: PH's `receipt_service.go` (receipt → many-invoice allocation, receipt numbering, reconciliation) exists only as an import-time transform in OSS (PC-D7). If receipts are a runtime need, this is a whole missing flow; if payments-only is intended, ratify it.

**Uncertainties / needs-Commander:**
- Whether the receipts→payments model (PC-D7) is the *ratified* substitute for PH's Receipt entity, or an import-only stopgap. Neither repo ships a receipt UI screen, so runtime impact may be nil — but the backend flow, tests (`customer_receipt_reconciliation_test.go`), and allocation semantics exist only in PH. (unverified — needs product call)
- ~~Post-payment status derivation dead if/else~~ — resolved Mission I W6: all mutators route through the settlement policy.
- ~~Credit-note PDF signature block differs~~ — closed Mission I W7 (I-18): PH block ported, identities overlay-resolved.
## Procurement

Deployed PH = `C:\Projects\asymmflow\ph_holdings` (`ui-ux-hardening`@ca24372).
OSS = `C:\Projects\asymmflow\asymmflow-oss` (`main`@801f41a). Line refs cited per repo.

OSS decomposes PO/GRN CRUD behind `a.procurementService()` (`app_services.go:247`,
`pkg/crm/procurement`), but the thin App wrappers and the business rules are
byte-for-byte equivalent to PH's root service files except where noted below.

| Element (flow) | Classification | PH evidence | OSS evidence | Action |
|---|---|---|---|---|
| **Purchase Orders** | | | | |
| PO create (auto-number, default dates, force Draft, items in tx) | PRESENT | purchase_order_service.go:473 CreatePurchaseOrder | purchase_order_service.go:187 createPurchaseOrder (via procurementService) | none |
| PO normalize + line-item validation (desc/qty/price guards, FX→BHD) | PRESENT | purchase_order_service.go:395 normalizePurchaseOrder | purchase_order_service.go:112 normalizePurchaseOrder | none |
| PO VAT basis = 10% on **SubtotalBHD** (MON-004; correct on non-BHD PO) | PRESENT | purchase_order_service.go:452-459 | purchase_order_service.go:169-173 | none (shipped Waves 1-3) |
| PO approval threshold = 5000 BHD → "Pending Approval" | PRESENT | purchase_order_service.go:14 const `purchaseOrderApprovalThresholdBHD=5000.0`; :522 | purchase_order_service.go:240 local `approvalThreshold:=5000.0`; :241 | none — same number; OSS threshold is a hardcoded local, not overlay-config (minor) |
| PO CreatedBy set for segregation-of-duties | PRESENT | purchase_order_service.go:513 | purchase_order_service.go:231 | none |
| PO approve (SoD: creator≠approver), status state-machine, financial-field lock on Approved/Sent/Received | PRESENT | purchase_order_service.go:901 ApprovePurchaseOrder; :683 UpdatePurchaseOrder | purchase_order_service.go:604 / :409 (via procurementService) | none |
| PO update crossing threshold → Pending Approval | PRESENT | UpdatePurchaseOrder + workflow_regression_test.go:791 | purchase_order_service.go:409 | none |
| PO send / amend / delete lifecycle | PRESENT | :958 SendPurchaseOrder; :1341 AmendPurchaseOrder; :1003 DeletePurchaseOrder | :656/:955/:701 | none |
| PO RBAC (`po:create`/`po:view`) on mutators | PRESENT | purchase_order_service.go:474 etc. | purchase_order_service.go:192 etc. | none |
| **Goods-Receipt Notes (GRN)** | | | | |
| GRN create / update (field-mask partial-update) / delete | PRESENT | grn_service.go:32/258/313 | grn_service.go:37/261/316 | none (field-mask shipped) |
| GRN QC status update + block-complete on QC Failed | PRESENT | grn_service.go:553 UpdateGRNQCStatus; :430 QC_FAILED guard | grn_service.go:516 / :392 | none |
| GRN discrepancy raise/resolve | PRESENT | grn_service.go:782/882 | grn_service.go:739/839 | none |
| GRN complete → PO qty-received update, over-receipt warn, double-count guard, PO status roll-up | PRESENT | grn_service.go:415 CompleteGRN; :504 updatePOStatus | grn_service.go:377 CompleteGRN; :467 updatePOStatus | none |
| Receive-against-PO (auto-create GRN) | PRESENT | grn_service.go via serial flow | grn_service.go:552 ReceiveAgainstPO (procurementService) | none |
| Serial capture at GRN receiving | PRESENT | serial_number_service.go:272 assignSerialsToGRN | serial_number_service.go:108 | none |
| **Procurement → Inventory receipt reconciliation** | | | | |
| GRN accepted qty → inventory item upsert + valued IN "GRN Receipt" movement, weighted-avg cost, in same tx | PRESENT | procurement_inventory_policy.go:56 reconcileInventoryReceipt → resolvePurchaseOrderItemReferenceCost (product_valuation_policy.go:29) + applyInventoryMovementValuation | procurement_inventory_policy.go:72 reconcileInventoryReceipt → `inventory.ResolvePurchaseOrderItemReferenceCost`/`ApplyMovementValuation` (pkg/inventory) | none (Band-2 supplierlink/valuation engine shipped W2) |
| Reference-cost resolvers exist & wired to receipt valuation | PRESENT | product_valuation_policy.go (whole file) | pkg/inventory/valuation.go | none |
| **Supplier Invoices** | | | | |
| Supplier-invoice create/update/delete (field-mask, default MatchStatus=Pending, server-derived totals) | PRESENT | supplier_invoice_service.go:31/261/490 | supplier_invoice_service.go:33/251/503 | none |
| Draft server-derived totals + zero-rated recovery, hollow-invoice send-block | PRESENT | supplier_invoice_service.go (Wave1) | supplier_invoice_service.go | none (shipped) |
| 3-way match structure (PO amount 2% tol, per-line unit-price 2% tol, GRN qty 2% tol, validation checks) | PRESENT | supplier_invoice_service.go:723 PerformThreeWayMatch | supplier_invoice_service.go:690 PerformThreeWayMatch | none |
| **3-way match PO-side price uses reference-cost resolvers** | **DIVERGENT-GAP** | PH resolves via `resolvePurchaseOrderItemReferenceCost(poItem)` (falls back to product standard cost when PO unit price = 0) + `resolveSupplierInvoiceItemUnitPriceBHD` — supplier_invoice_service.go:772-776 | OSS uses **raw** `poItem.UnitPriceBHD` and inline `invItem.UnitPrice`×FX — supplier_invoice_service.go:739-745; resolvers exist in pkg/inventory but are **not called here** | Wire `inventory.ResolvePurchaseOrderItemReferenceCost` into OSS PerformThreeWayMatch (the known-open Band-2 residue). Impact: a PO line with 0 unit price yields no price-variance flag in OSS |
| Matched 3-way sets invoice Status | DIVERGENT-INTENTIONAL | matched → `Status="Pending"` (supplier_invoice_service.go:854) | matched → `Status="Verified"` (supplier_invoice_service.go:823) | none — label difference; OSS "Verified" is the downstream-pay-eligible state |
| Price-variance-only outcome label | DIVERGENT-INTENTIONAL | MatchStatus="Pending" (:859) | MatchStatus="Review Required" (:828) | none — richer OSS label, same "not a hard discrepancy" meaning |
| **Supplier-invoice APPROVE gate** | **DIVERGENT-GAP (integrity)** | Refuses unless `MatchStatus == "Matched"` (supplier_invoice_service.go:558) AND SoD | Only refuses when `MatchStatus == "Discrepancy"` (supplier_invoice_service.go:552); allows Pending/Review-Required/never-matched + SoD | OSS lets an unmatched supplier invoice be approved. Decide intended strictness; PH is the tighter control |
| Approve segregation-of-duties (creator≠approver, require CreatedBy) | PRESENT | supplier_invoice_service.go:550-555 | supplier_invoice_service.go:544-549 | none |
| Update guard: Approved invoice must be Matched | DIVERGENT-GAP (integrity) | supplier_invoice_service.go:311-313 blocks Approved w/o Matched | OSS UpdateSupplierInvoice preserves MatchStatus but no Approved⇒Matched assertion (supplier_invoice_service.go:319) | Same root as approve-gate relaxation; verify intended |
| Dispute supplier invoice | PRESENT (label diff) | Status="Dispute" (supplier_invoice_service.go:590) | Status="Disputed" (supplier_invoice_service.go:584) | none — cosmetic status-string difference |
| Mark paid / unpaid / overdue queries, OCR-create | PRESENT | supplier_invoice_service.go:607/650/680/892 | supplier_invoice_service.go:601/637/657/861 | none |
| **Supplier Payments** | | | | |
| **RecordSupplierPayment eligibility gate** | **DIVERGENT-GAP (integrity — money)** | Requires `MatchStatus=="Matched"` AND `Status=="Approved"` (supplier_payment_service.go:37-42); else refuse | Only blocks `MatchStatus=="Discrepancy"`; allows `Status` in {Approved,Verified,Pending} (supplier_payment_service.go:43-48) | OSS pays invoices that were never matched/approved. PH refuses. Highest-value control divergence in this domain |
| Supplier-payment TOCTOU / race protection, future-date block, positive-amount | PRESENT | supplier_payment_service.go:76 tx-before-check; :45/:78 | supplier_payment_service.go:51/:78 | none |
| Supplier-payment update (amount>0, balance recalc, audit) / delete | PRESENT | supplier_payment_service.go:301/409 | supplier_payment_service.go:339/448 | none (supplier-payment delete bug fixed Wave 6) |
| Remaining-payment / payment-state policy engine | PRESENT | supplier_invoice_payment_policy.go (whole) | supplier_invoice_payment_policy.go | none |
| OCR routing → PO / supplier-invoice | PRESENT | app.go:16037 routeToPurchaseOrder; :16141 routeToSupplierInvoice | app_setup_documents_surface.go:3050 / :3127 | none |
| Supplier-payment matcher from bank statement lines | PRESENT | bank_transaction_matcher.go:393 matchToSupplierPayment | bank_transaction_matcher.go (equiv) | none |
| **Journal / accounting posting link** | | | | |
| Supplier-invoice & supplier-payment → double-entry posting preview | EXTRA | PH has no supplier posting-preview service (no `PreviewSupplierInvoicePosting`/`…PaymentPosting`) | accounting_posting_service.go:37 PreviewSupplierInvoicePosting; :49 PreviewSupplierPaymentPosting → pkg/finance/posting | note — OSS substrate value; verify PH GL path not double-counting |
| **OUT-side stock movement on delivery/dispatch** | PRESENT (both lack auto-decrement) | No inventory decrement in delivery_note_service.go; only manual adjustment OUT (app.go:22680/22728) | No inventory decrement in delivery_note_service.go; only manual adjustment OUT (app_accounting_inventory.go:1264/1312) | none — parity; neither auto-issues stock on DN dispatch. (Not an OSS-specific gap.) |

### Counts
- **PRESENT:** 27 · **DIVERGENT-INTENTIONAL:** 2 · **DIVERGENT-GAP:** 4 · **ABSENT:** 0 · **EXTRA:** 1

### Top gaps to close (money/integrity first)
- **RecordSupplierPayment gate is materially weaker in OSS** (supplier_payment_service.go:43-48 vs PH :37-42). PH refuses to pay until an invoice is BOTH `Matched` and `Approved`; OSS pays anything that isn't flagged `Discrepancy`, including never-matched / merely-`Pending` invoices. This is the highest-risk cash-control divergence — an AP clerk can disburse against an unvetted supplier invoice.
- **ApproveSupplierInvoice gate weaker in OSS** (supplier_invoice_service.go:552 vs PH :558). OSS approves unless `Discrepancy`; PH requires `Matched`. Same relaxation family as the payment gate; together they remove the "no pay/approve without a clean 3-way match" invariant.
- **3-way match does not use reference-cost resolvers in OSS** (supplier_invoice_service.go:739-745). The pre-ratified known-open Band-2 residue: OSS compares against raw `poItem.UnitPriceBHD` with no standard-cost fallback, so a PO line lacking a unit price silently passes the price check. Resolvers already exist in `pkg/inventory` (used by GRN receipt) — just unwired here.
- **Update-guard on Approved supplier invoices** (supplier_invoice_service.go:311-313 in PH) has no OSS counterpart — a post-approval edit can leave an Approved invoice without a Matched status. Low blast radius but closes the loop on the two gates above.

### Uncertainties / needs-Commander
- The approve/pay gate relaxations in OSS (Discrepancy-only vs PH's Matched+Approved) are **not** in the pre-ratified DIVERGENT-INTENTIONAL list. They read as genuine control gaps, but OSS's use of a distinct `Verified` status suggests a possibly-intended different lifecycle (Verified = pay-eligible). Recommend Commander confirm intended AP strictness before "fixing," since tightening changes who can disburse cash.
- `PreviewSupplierInvoicePosting`/`…PaymentPosting` (OSS EXTRA via pkg/finance/posting) — did not verify whether PH posts supplier invoices to a GL by another path; classified EXTRA on the preview surface only. (unverified — needs closer look at PH accounting screen backend.)
- OSS PO approval threshold is a hardcoded `5000.0` local, not overlay-config; brief lists thresholds-as-config as a goal. Same numeric behavior as PH today, so classified PRESENT, but flag if overlay-driven thresholds are expected.
## CRM (customers / suppliers / contacts / parties)

Deployed PH root: `C:\Projects\asymmflow\ph_holdings` (branch `ui-ux-hardening`).
OSS: `C:\Projects\asymmflow\asymmflow-oss` (`main`). CRM engine lives in `pkg/crm/*`
(esp. `pkg/crm/customer/{write.go,delete.go}`), called from thin App wrappers in
`app_order_customer_surface.go` and `app_crm_surface.go`. The Butler-create path is in
`app_costing_exports_surface.go`; folder creation in `app_setup_documents_surface.go`.

| Element (flow) | Classification | PH evidence | OSS evidence | Action |
|---|---|---|---|---|
| Customer create + code/ID auto-gen | PRESENT | app.go:11638 (inline `CUST-<PREFIX4><ms%100000>`, CustomerID←CustomerCode) | app_order_customer_surface.go:1355 → `crmcustomer.PrepareCustomerCreate` pkg/crm/customer/write.go:68 | none |
| Customer code-prefix algorithm (names w/ early spaces) | DIVERGENT-INTENTIONAL | app.go:11663 truncates `[:4]` THEN strips non-A-Z (shorter prefix) | write.go:22 `businessPrefix` collects first 4 A-Z runes (documented improvement) | none — codes still unique (ms suffix) |
| Customer update (field-mask / no blank-clobber) | PRESENT | app.go:12026 inline merge (G1: preserves server-owned metrics, unique keys fall back) | app_order_customer_surface.go:1689 → `crmcustomer.MergeCustomerUpdate` write.go:101 | none (Wave "identity-write WritePolicy" shipped) |
| Customer update re-repairs blank identifiers | DIVERGENT-INTENTIONAL | app.go:12026 does NOT re-run identifier assignment on update | write.go:135 `MergeCustomerUpdate` calls `AssignCustomerIdentifiers` (repairs legacy blanks in passing) | none — OSS strictly safer |
| Customer delete + child-record guard | PRESENT (pre-ratified divergence now CLOSED) | app.go:12170 blocks w/ `CUSTOMER_HAS_LINKED_RECORDS` (orders/invoices/offers/opps) | app_order_customer_surface.go:1765 → `crmcustomer.DeleteCustomer` delete.go:29 — SAME guard + SAME error code (PC-D1: guard moved into engine, protects every caller incl. approved requests) | none. NOTE: brief lists party-delete orphaning as DIVERGENT-INTENTIONAL; OSS engine now carries the identical block-guard, so behavior is parity, not orphaning |
| Business-customer-ID backfill (discard UUID-shaped) | PRESENT | app.go:11718 `BackfillBusinessCustomerIDs` (blesses any code) | app_order_customer_surface.go:1397 → `crmcustomer.NormalizeBusinessIdentifier`/`RepairCustomerBusinessID` write.go:179/191 (also discards UUID-shaped legacy codes — stricter) | none |
| Supplier create + code auto-gen + default rating(3) | PRESENT | app.go:11971 (inline `SUP-...`, `Rating==0→3`) | app_order_customer_surface.go:1657 → `crmcustomer.PrepareSupplierCreate` write.go:79 | none |
| Supplier update (field-mask, Rating 0 = keep) | PRESENT | app.go:12106 inline merge, Rating overwritten directly (note: PH takes Rating even if 0) | app_order_customer_surface.go:1729 → `crmcustomer.MergeSupplierUpdate` write.go:143 — `Rating!=0` guard (safer: 0 won't wipe) | none — OSS strictly safer |
| Supplier delete + child-record guard | PRESENT | app.go:12213 blocks w/ `SUPPLIER_HAS_LINKED_RECORDS` (POs/supplier invoices/payments) | app_order_customer_surface.go:1782 → `crmcustomer.DeleteSupplier` delete.go:60 — SAME guard/error | none |
| Customer contacts CRUD | PRESENT | app.go:11765–11834 (List/Add/Update/Delete) | app_order_customer_surface.go:1444–1510 | none |
| Customer contact update (load-then-overlay, no FK zero) | PRESENT | app.go:11794 INT-001 overlay (preserves CustomerID FK + audit) | app_order_customer_surface.go:1473 identical INT-001 overlay | none (field-mask contact protection shipped) |
| Supplier contacts CRUD | PRESENT | app.go:11841–11894 | app_order_customer_surface.go:1517–1580 | none |
| Supplier contact update (load-then-overlay) | PRESENT | app.go:11870 bare `Save` (NO overlay in PH) | app_order_customer_surface.go:1546 INT-001 overlay added (safer) | none — OSS strictly safer |
| Contact list ordering (primary first) | PRESENT | app.go:11773/11849 `ORDER BY is_primary_contact DESC, contact_name ASC` | app_order_customer_surface.go:1452/1524 identical | none |
| Primary-contact ↔ master sync + single-primary enforcement | ABSENT (but PH copy is DEAD) | supplier_contact_policy.go:28/76/98 `syncSupplierPrimaryContactRecord`/`refreshSupplierMasterPrimaryContact`/`ensureSinglePrimarySupplierContact` — DEFINED but ZERO CRUD call-sites (unwired in deployed PH) | no counterpart in OSS | none — dead code in PH; not a live behavioral gap. See uncertainties |
| Customer→products relationship rollup | PRESENT (backend; Mission I W7, I-22) | app.go:11383 `GetCustomerRelatedProducts` (+ `customerProductRollups` helper), capped 50, wired to CustomerDetailView.svelte | app_crm_surface.go `GetCustomerRelatedProducts` + `customerProductRollups` (OSS-adapted: no Brand×Token taxonomy; supplier via ProductMaster) | frontend binding in CustomerDetailView — post-Mission-I frontend pass |
| Customer→suppliers relationship rollup (via products) | PRESENT (backend; Mission I W7, I-22) | app.go:11433 `GetCustomerRelatedSuppliers` (distinct instrument families per supplier, capped 50) | app_crm_surface.go `GetCustomerRelatedSuppliers` (derived through product rollups) | frontend binding — post-Mission-I frontend pass |
| Customer 360 view / profile | PRESENT | app.go:3814 `GetCustomer360View`, app.go:4141 `GetCustomerFullProfile` | app_crm_surface.go:11/308 | none |
| Customer 360 graph (relationship network) | PRESENT | app.go:12639 `GetCustomer360Graph`, app.go:17867 `GetCustomerGraph(depth)` | app_order_customer_surface.go:2160, app_graph_contract_surface.go:286 | none |
| Customers by grade | PRESENT | app.go:3938 `GetCustomersByGrade` | app_crm_surface.go:114 | none |
| Customer create via Butler AI | PRESENT | app.go:19791 `CreateCustomerFromButler` | app_costing_exports_surface.go:430 (byte-identical) | none |
| Supplier create via Butler AI | PRESENT | app.go:19859 `CreateSupplierFromButler` | app_costing_exports_surface.go:498 (byte-identical) | none |
| List/Get customer + supplier (+cache, Select cols) | PRESENT | app.go:11088/11137/11902/11951 | app_order_customer_surface.go:1156/1205/1588/1637 | none |
| Paginated + optimized list variants | PRESENT | pagination.go:79/107, query_optimizations.go:300/306 | pagination.go:79/107, query_optimizations.go:300 | none |
| Customer/supplier search (list + filter, no dedicated Search*) | PRESENT | FilterOrders app.go (customer query); no `SearchCustomers` in PH either | app_order_customer_surface.go:1078 `FilterOrders`; no `SearchCustomers` in OSS either | none — same mechanism (list+filter) |
| Credit-limit fields (CreditLimitBHD, IsCreditBlocked, RequiresPrepayment, grades) | PRESENT | database.go:45+ CustomerMaster (default 50000, grade default 'C') | pkg/crm/domain.go:14+ (byte-identical field tags/defaults) | none (credit-limit override via pkg/approvals shipped) |
| Division/branding default on records | DIVERGENT-INTENTIONAL | database.go:284/361/487 `default:'PH Trading'` (also 'AHS Trading') hardcoded | pkg/crm/domain.go:219/288/380 `size:100` NO hardcoded default; division/branding are overlay config (SYNTHETIC_IDENTITY) | none — branding is config not code, by design |
| Seed customer/supplier DB | PRESENT | app.go:11508 `SeedCustomerDatabase` | app_order_customer_surface.go:1225 | none |
| Customer/supplier folder creation (OneDrive) | PRESENT | app.go:14884/14923 | app_setup_documents_surface.go:1940/1979 | none |
| Supplier notes / issues / resolve | PRESENT | app.go:4614/4645, ResolveSupplierIssue | app_crm_surface.go:713/744/769 | none |
| Follow-up tasks (CRM) | PRESENT (EXTRA-parity) | present in PH | app_order_customer_surface.go:2315–2451 | none |

### Counts
- **PRESENT:** 28 · **DIVERGENT-INTENTIONAL:** 3 · **DIVERGENT-GAP:** 0 · **ABSENT:** 1 (dead in PH) · **EXTRA:** 0

### Top gaps to close (money/integrity first)
- ~~`GetCustomerRelatedProducts` / `GetCustomerRelatedSuppliers` ABSENT~~ — **backend closed** Mission I W7 (I-22), golden-tested; CustomerDetailView.svelte binding lands in the post-Mission-I frontend pass.
- **(Non-gap, flagged for the ledger) Primary-contact sync / single-primary enforcement is DEAD in PH and absent in OSS** — `supplier_contact_policy.go` exists but has zero CRUD call-sites in deployed PH, so neither app keeps `SupplierMaster.PrimaryContact` in lockstep with the flagged `SupplierContact`, nor prevents multiple `is_primary_contact=true` rows. Not a regression against deployed PH; note only.

### Uncertainties / needs-Commander
- **No live financial-number divergence found in CRM.** Credit-limit fields, grades, and defaults are byte-identical between repos; the only value difference is the `division` default string ('PH Trading' vs blank), which is intentional synthetic/overlay behavior, not a bug.
- **Delete semantics:** the pre-ratified brief says OSS *orphans* party children on delete while PH blocks. The actual OSS code (`pkg/crm/customer/delete.go:29/60`) now carries the **identical** `*_HAS_LINKED_RECORDS` child-count block-guard (PC-D1), so OSS does **not** orphan — behavior is parity. Classified PRESENT, not DIVERGENT-INTENTIONAL. Flagging so the ratified-divergence ledger can be updated (that divergence appears closed).
- **Dead policy code in PH:** `customer_write_policy.go`, `supplier_write_policy.go`, `supplier_contact_policy.go` are all unwired in deployed PH (zero CRUD call-sites). PH's real behavior is the inline merge in `app.go`; OSS ported the equivalent into `pkg/crm/customer`. Verified via grep — but if PH wires these elsewhere at runtime I couldn't observe, the primary-contact-sync row would shift from "dead" to a genuine ABSENT gap *(unverified — needs closer look)*.
## Opportunities / Costing / Offers / RFQ

Deployed PH = `C:\Projects\asymmflow\ph_holdings` (ui-ux-hardening @ ca24372). OSS = `C:\Projects\asymmflow\asymmflow-oss` (main @ 801f41a). PH's sales surface is concentrated in `app.go`; OSS split it into `app_sales_pipeline.go` + `app_costing_exports_surface.go` + service files, with some logic pushed to `pkg/crm/pipeline` and `pkg/engines/costing_engine.go`. Surface parity is HIGH; the real gaps are a cluster of delete/revision/attachment flows that were never ported.

| Element (flow) | Classification | PH evidence | OSS evidence | Action |
|---|---|---|---|---|
| RFQ create (+WithReference, private createRFQ) | PRESENT | app.go:4755/4760/4764 | app_sales_pipeline.go:21/26/30 | none |
| RFQ list / get / update / update-notes / update-status / delete | PRESENT | app.go:4816/6053/6238/6332/6073/6199 | app_sales_pipeline.go:81/1180/1358/1448/1200/1318 | none |
| RFQ number generation (double-BEGIN fix) | PRESENT | app.go:6088 generateRFQNumber | app_sales_pipeline.go:1215 | none |
| RFQ stage update | PRESENT | app.go:6123 UpdateRFQStage | app_sales_pipeline.go:1250 | none |
| RFQ duplicate check (customer/project/hash) | PRESENT | app.go:6360 CheckDuplicateRFQ | app_sales_pipeline.go:1476 | none |
| RFQ cascade delete | PRESENT | app.go:6424 DeleteRFQWithCascade | app_sales_pipeline.go:1540 | none |
| RFQ comments (add/list) | PRESENT | app.go:6504/6617 | app_sales_pipeline.go:1625/1738 | none |
| RFQ → order full flow | PRESENT | pipeline_handlers.go:183 ProcessRFQToOrder | pipeline_handlers.go:182 | none |
| RFQ document tracking backfill | PRESENT | app.go:23928 | app_dashboard_datafix_surface.go:864 | none |
| Email → RFQ save | PRESENT | msg_parser.go:526 SaveParsedEmailAsRFQ | msg_parser.go:526 | none |
| Opportunity list / pipeline (dedup by canonical key) | PRESENT | app.go:4848 GetPipelineOpportunities | app_sales_pipeline.go:113; canonicalOpportunityKey app_sales_pipeline.go:311 | none |
| Opportunity canonical-key matching / collapse dedup | PRESENT | app.go (canonicalOpportunityKey usage) | app_sales_pipeline.go:161/311, butler_ai_context.go:136/151, app_prediction_dashboard.go:309 | none |
| Opportunity dup check | PRESENT | app.go:6393 CheckDuplicateOpportunity | app_sales_pipeline.go:1509 | none |
| Opportunity stage update (plain + optimistic-version) | PRESENT | app.go:6189; opportunity_conflict_service.go:203 | app_sales_pipeline.go:1308; opportunity_conflict_service.go:203 | none |
| Opportunity edit-conflict foundation / list / resolve | PRESENT | opportunity_conflict_service.go:64/75/105 | opportunity_conflict_service.go:64/75/105 (+ CRMService wrappers service_crm.go:762/774) | none |
| Opportunity details update (comment/ownerNotes ±version) | PRESENT | app.go:9790; opportunity_conflict_service.go:268 | app_sales_pipeline.go:4194; opportunity_conflict_service.go:268 | none |
| Opportunity comments (add/list/delete) | PRESENT | app.go:6546/6633/6585 | app_sales_pipeline.go:1667/1754/1706 | none |
| Customer opportunities / line items | PRESENT | app.go:3998/7419 | app_crm_surface.go:176; app_sales_pipeline.go:2459 | none |
| **Opportunity digit-guard (D1: folder number must contain a digit)** | PRESENT | onedrive_import_service.go:806 folderNumberHasDigit | onedrive_import_service.go:668 (D1 guard, cites PH 10f96a7) | none |
| **Loose folder-number token cleaning (D2)** | DIVERGENT-GAP (PARTIAL) | onedrive_import_service.go:810 cleanLooseOneDriveFolderNumberToken (+ splitLooseOneDriveFolderNumberToken:826) | 0 occurrences in OSS — helper never ported; OSS uses raw looseToken at onedrive_import_service.go:668 | Port `cleanLooseOneDriveFolderNumberToken` + `splitLooseOneDriveFolderNumberToken` defense-in-depth trim |
| Update opportunity commercial fields | ABSENT | user_feedback_hardening_service.go:87 UpdateOpportunityCommercialFields | 0 occurrences (whole file absent) | Port method |
| **Delete opportunity (admin-gated)** | ABSENT | user_feedback_hardening_service.go:183 DeleteOpportunity (requireAdminDelete + offers:delete) | 0 occurrences; delete_approval_service.go only handles `offer_note` | Port admin-gated opportunity delete |
| Unified offer thread (notes+followups+comments merged) | ABSENT | user_feedback_hardening_service.go:334 GetUnifiedOfferThread | 0 occurrences | Port if UI thread view is wired |
| Costing sheet create | PRESENT | app.go:6861 CreateCostingSheet | app_sales_pipeline.go:1973 | none |
| Costing by RFQ / active / set-active / clone-revision | PRESENT | app.go:6989/7039/7060/7103 | app_sales_pipeline.go:2068/2090/2111/2144 | none |
| Costing list / get / update / delete | PRESENT | app.go:7126/7148/7169/7207 | app_sales_pipeline.go:2167/2189/2210/2247 | none |
| Costing approve/reject (8% min-margin, Grade C/D, ABB 15% enforced) | PRESENT | app.go:7237/7299 | app_sales_pipeline.go:2277/2339; business_invariants.go:424/445/466/727/742 | none |
| Costing versioned update / new version (optimistic) | PRESENT | P1_SALES_PIPELINE_FIXES.go:139/224 | P1_SALES_PIPELINE_FIXES.go:139/223 | none |
| Costing calculate + risk assessment + min-margin | PRESENT | app.go:19387/19494 | app_costing_exports_surface.go:19/132; pkg/engines/costing_engine.go:206/377/387 | none |
| Costing engine (markup/margin math) | PRESENT | costing_engine.go (root) | pkg/engines/costing_engine.go | none |
| Costing↔opportunity import (ordered passes) | PRESENT | excel_costing_parser.go:1234 ImportCostingToOpportunity | excel_costing_parser.go:884 | none |
| Costing↔RFQ reference sync | PRESENT | app.go:8242 syncRFQReferenceFromCosting | app_sales_pipeline.go:3117 | none |
| Excel costing parse / scan / batch import | PRESENT | excel_costing_parser.go:1134/1162/1333 | excel_costing_parser.go:787/815/983 | none |
| Costing PDF / Excel export | PRESENT | app.go:19933/20586 | app_costing_exports_surface.go:572/1138 | none |
| Read costings by opportunity record-id (string) | ABSENT | app.go:7013 GetCostingsByOpportunity | 0 occurrences; OSS only exposes GetCostingsByRFQ (app_sales_pipeline.go:2068) | Port opportunity-keyed read (or confirm RFQ-keyed path suffices) |
| **Costing-sheet file attachment (attach/list/delete/open datasheets)** | ABSENT | costing_attachment_service.go:120/141/287/324/348 | 0 occurrences (CostingSheetAttachment* absent everywhere) | Port costing attachment service if datasheet attach is used |
| **Costing PDF datasheet bundle (append supplier datasheets into offer PDF)** | ABSENT | costing_pdf_bundle_service.go:18/172; app.go:230 copyLargeCostingPDFToCustomerOfferFolder | 0 occurrences | Port datasheet-append into PDF bundle |
| Offer create (from costing id) | PRESENT | app.go:7332 CreateOffer | app_sales_pipeline.go:2372 | none |
| Save costing as offer (tx-wrapped, terms/VAT/margin carried) | PRESENT | app.go:8032 SaveCostingAsOffer | app_sales_pipeline.go:2928 | none |
| Update offer from costing data | PRESENT | app.go:8268 updateOfferFromCostingData | app_sales_pipeline.go:3143 | none |
| Offer list / get / all / with-no-items | PRESENT | app.go:7378/7400/8537/8572 | app_sales_pipeline.go:2418/2440/3411/3446 | none |
| Offer full update / status update | PRESENT | app.go:9320/7470 | app_sales_pipeline.go:3706/2510 | none |
| Offer number generation + availability guard | PRESENT | app.go:8511 generateOfferNumber; 7811 ensureOfferNumberAvailable | app_sales_pipeline.go:3381/2814 | none |
| Offer number preference-for-costing helper | ABSENT | app.go:7831 availableOfferNumberForCosting | 0 occurrences (only ensureOfferNumberAvailable present) | Low priority; confirm SaveCostingAsOffer number-picking parity |
| Link offer ↔ opportunity (stage transition on link) | PRESENT | app.go:8439 linkOfferToOpportunity | app_sales_pipeline.go:3310 | none |
| Convert offer → order (+legacy path) | PRESENT | app.go:7493/7610 | app_sales_pipeline.go:2533/2650 | none |
| Mark offer won (creates order, PO capture) | PRESENT | app.go:9571 MarkOfferWon | app_sales_pipeline.go:3937 | none |
| Mark offer lost (won→lost guard) | PRESENT | app.go:9736 MarkOfferLost | app_sales_pipeline.go:4140 | none |
| Offer notes (add/get/delete) | PRESENT | app.go:9500/9533/9548 | app_sales_pipeline.go:3882/3911/3926 (→ pkg/crm/pipeline/delete.go:17) | none |
| Offer follow-ups (add/get/overdue/complete/cancel) | PRESENT | offer_followup_service.go:11/104/147/169/205 | offer_followup_service.go:10/49/92/114/143 | none |
| Offer PDF generation | PRESENT | offer_pdf_service.go:13 | offer_pdf_service.go:13 | none |
| Offer revision history (read) | PRESENT | p2_sales_ux_enhancements.go:32 GetOfferRevisionHistory | p2_sales_ux_enhancements.go:32 | none |
| **Create offer revision (new revision identity)** | ABSENT | app.go:8748 CreateOfferRevision; helper 8702 nextOfferRevisionIdentity | 0 occurrences | Port revision-create (history read exists but no way to create a revision) |
| **Renew offer (re-open/extend expired offer)** | ABSENT | app.go:8851 RenewOffer | 0 occurrences | Port offer renewal |
| **Delete offer (won-guard + linked order/invoice guards)** | ABSENT | app.go:8625 DeleteOffer (blocks won, blocks if linked orders/invoices, unlinks opportunity in tx) | 0 occurrences; only softDeleteOfferIfUnlinked (onedrive_import_service.go:1022, internal) + DeleteOfferNote | Port user-facing offer delete WITH its integrity guards |
| Auto-expire offers | PRESENT | P1_SALES_PIPELINE_FIXES.go:27 AutoExpireOffers | P1_SALES_PIPELINE_FIXES.go:27 | none |
| Low-margin offers report | PRESENT | P1_SALES_PIPELINE_FIXES.go:329 GetLowMarginOffers | P1_SALES_PIPELINE_FIXES.go:327 | none |
| Bulk offer stage update / offer search / paginated list | PRESENT | p2_sales_ux_enhancements.go:173/411; pagination.go:236 | same paths in OSS | none |
| Butler offer draft (grounded fast path + CreateOfferDraftFromButler) | PRESENT | butler_grounded_fastpath.go:1621; app.go:19683 | butler_grounded_fastpath.go:877; app_costing_exports_surface.go:322 | none |
| Butler customer-year / offer-overview summary responses | ABSENT | butler_grounded_fastpath.go:1236/1290 buildCustomerYearOffersResponse / buildCustomerOfferOverviewResponse | 0 occurrences | Butler-domain; port if those grounded responses are expected |
| Offer number repair util (from references) | ABSENT | app.go:2389 repairGeneratedOfferNumbersFromReferences | 0 occurrences | Migration/repair util — low priority |
| Won-offer item backfill from opportunity product details | PRESENT | app.go:8908 | app_sales_pipeline.go:3514 | none |
| Offer item cost-breakdown backfill | PRESENT | app.go:23987 | app_dashboard_datafix_surface.go:923 | none |
| Data reconciliation (extracted offers ↔ DB) | PRESENT | data_reconciliation.go:152/302/387 | data_reconciliation.go:152/298/383 | none |
| Offers batch OCR ingest / RFQ+quotation extraction | PRESENT | app.go:16996/17179/17226 | app_setup_documents_surface.go:3953/4131/4178 | none |
| Route to opportunity / RFQ (ingestion) | PRESENT | app.go:15751/15868 | app_setup_documents_surface.go:2802/2899 | none |

**Counts:** PRESENT 52 · DIVERGENT-INTENTIONAL 0 · DIVERGENT-GAP 1 · ABSENT 11 · EXTRA 0

**Top gaps to close (money/integrity first):**
- **DeleteOffer ABSENT (integrity)** — PH's `DeleteOffer` (app.go:8625) refuses to delete a *won* offer, refuses when linked orders/invoices exist, and unlinks the opportunity inside a transaction. OSS has no user-facing offer delete at all (only internal `softDeleteOfferIfUnlinked`), so a frontend delete button would be dead AND the guards protecting won/linked offers don't exist. Port with the guards intact.
- **DeleteOpportunity ABSENT (integrity)** — PH admin-gates opportunity deletion (`requireAdminDelete` + `offers:delete`, user_feedback_hardening_service.go:183). OSS lacks it entirely; no controlled path to remove a bad opportunity.
- **Offer revision + renewal ABSENT (lifecycle)** — OSS can *read* revision history (p2_sales_ux_enhancements.go:32) but cannot `CreateOfferRevision` (app.go:8748) or `RenewOffer` (app.go:8851). The revision lifecycle is half-wired: history with no way to create a revision or renew an expired quote.
- **cleanLooseOneDriveFolderNumberToken (D2) PARTIAL** — the D1 digit-guard is ported (onedrive_import_service.go:668) but the paired cleaning/splitting helpers (PH onedrive_import_service.go:810/826) are not, so loose OneDrive folder tokens aren't trimmed/validated defense-in-depth before becoming a folder number.
- **Costing attachment + datasheet PDF bundle ABSENT (customer-facing doc)** — PH attaches supplier datasheets to a costing and appends them into the offer PDF bundle (costing_attachment_service.go, costing_pdf_bundle_service.go). OSS has neither; offer PDFs cannot carry datasheets. Touches customer-facing document content.

**Uncertainties / needs-Commander:**
- The costing datasheet-bundle absence changes what a customer receives (offer PDF without appended datasheets). If PH clients rely on bundled datasheets, this is a customer-facing document-appearance divergence — stop-and-ask before deciding whether OSS should match.
- `GetCostingsByOpportunity` (opportunity record-id keyed) is absent but OSS's `GetCostingsByRFQ` covers the RFQ-keyed read; whether any opportunity carries costings NOT reachable via its RFQ id is unverified — needs a closer look at how OSS unifies opportunity vs RFQ ids.
- `availableOfferNumberForCosting` absent but `ensureOfferNumberAvailable` present — SaveCostingAsOffer number-collision behavior looked equivalent on skim but was not exhaustively traced (unverified — needs closer look).
## Inventory / Stock Movement / Products

| Element (flow) | Classification | PH evidence | OSS evidence | Action |
|---|---|---|---|---|
| Product catalogue search + lookup by code | PRESENT | `product_service.go:13` `SearchProducts`, `:53` `GetProductByCode` | `product_service.go:29`, `:69` | none |
| Product seed/create + supplier-link normalization on write | PRESENT | `product_service.go:135` `normalizeProductSupplierLink` (in `seedProductDatabaseInternal`) | `product_service.go:148` `supplierlink.NormalizeProductSupplierLink` | none |
| Product↔supplier resolution engine (commercial-token/alias) | PRESENT | `product_supplier_link_policy.go:8` `resolveSupplierForProduct`, `:52` `canonicalSupplierCodeRef`; callers in `inventory_service.go`, `purchase_order_service.go`, `app.go:11367` | **Wave 2 engine** `pkg/crm/supplierlink/supplierlink.go:125` `ResolveSupplierForProduct`, `:90` `FindSupplierByCommercialToken`, `:31` `CanonicalCode`; aliases from overlay via `product_service.go:16` `supplierLinkAliases()` | none — verified PRESENT |
| Warehouses list/create | PRESENT | `app.go:22815` `GetWarehouses`, `:22833` `CreateWarehouse` | `app_accounting_inventory.go:1403`, `:1421` | none |
| Inventory item CRUD (get/list/create/update) | PRESENT (minor signature divergence) | `app.go:22339` `GetInventoryItems(warehouseID *string,…)`, `:22393` Create, `:22450` Update | `app_accounting_inventory.go:965` `GetInventoryItems(warehouseID *uint,…)`, `:1014`, `:1048` | none — OSS binds `warehouseID`/`itemID` as `*uint` vs PH `*string`; item PK is string in both queries, so filter is cosmetic. (unverified — needs closer look if UI passes string warehouse ids) |
| Stock movement IN/OUT record + balance-before/after + stock-status + movement numbering | PRESENT | `app.go:22472` `RecordStockMovement` (INSUFFICIENT_STOCK guard, `MOV-YYYY-NNNNN`), `:22581` `calculateStockStatus` | `app_accounting_inventory.go:1065`, `:1165` — same guard, same numbering (date-range scan) | none |
| Weighted-average valuation w/ reference-cost fallback on movement | PRESENT | `product_valuation_policy.go:68` `applyInventoryMovementValuation` (called `app.go:22542`) | **Wave 2 engine** `pkg/inventory/valuation.go:86` `ApplyMovementValuation` (called `app_accounting_inventory.go:1125`). **Exact port** — identical IN newAvg formula, OUT zero-cost backfill, unrounded TotalValue | none — verified PRESENT |
| Reference-cost resolver chain (unit→last-purchase→product-standard→zero) | PRESENT | `product_valuation_policy.go:10/29/40/51` `resolveProduct/PO/SupplierInvoice/InventoryItem…Cost` | `pkg/inventory/valuation.go:25/41/50/63` `ResolveProduct/PurchaseOrderItem/InventoryItemUnitCost`, `SupplierInvoiceItemUnitPriceBHD` | none |
| Stock adjustment create + approve (OUT-side via negative variance) | PRESENT | `app.go:22635` `CreateStockAdjustment` (`Direction="OUT"` @ 22680/22728), `:22689` `ApproveStockAdjustment` | `app_accounting_inventory.go:1219`, `:1264`/`:1312` OUT, `:1273` Approve | none — OUT movements DO exist in OSS via adjustment path |
| GRN receipt → inventory item ensure + stock IN reconcile | PRESENT | `procurement_inventory_policy.go:12` `ensureInventoryItemForReceipt`, `:56` `reconcileInventoryReceipt` (called from `CompleteGRN` `grn_service.go:472`) | `procurement_inventory_policy.go:24`, `:72` (called `grn_service.go:435`); uses `inventory.ResolvePurchaseOrderItemReferenceCost` + `ApplyMovementValuation` (`:100`/`:112`) | none |
| GRN discrepancy costing (cost impact from PO/product contract) | PRESENT | `grn_service.go:782` `RaiseGRNDiscrepancy` → `resolvePurchaseOrderItemReferenceCost` @ `:817`; `costImpact = rejectedQty*unitCost` | `grn_service.go:739` → `inventory.ResolvePurchaseOrderItemReferenceCost` @ `:775` | none |
| Stock alerts: low-stock / slow-moving / summary | PRESENT | `inventory_service.go:32/132/208` | `inventory_service.go:33/135/211` (line-for-line equivalent) | none |
| Reorder suggestions | PRESENT | `inventory_service.go:264` `GetReorderSuggestions` | `inventory_service.go:267` | none |
| Inventory valuation rollup (warehouse-filtered, BHD-rounded) | PRESENT | `app.go:22777` `GetInventoryValuation` → `resolveInventoryItemTotalValue` @ 22799 | `app_accounting_inventory.go:1365` → `inventory.ResolveInventoryItemTotalValue` @ 1387 | none |
| Supplier-invoice 3-way match — unit-price variance consuming reference-cost resolvers | **DIVERGENT-GAP** | `supplier_invoice_service.go:723` `PerformThreeWayMatch`: line 772 `resolveSupplierInvoiceItemUnitPriceBHD(invoice,invItem)` + 773 `resolvePurchaseOrderItemReferenceCost(poItem)` (falls back to product standard cost; FX≤0 passes price through) | `supplier_invoice_service.go:690` `PerformThreeWayMatch`: lines **739–742 inline** `invItem.UnitPrice * invoice.ExchangeRate`, **745** raw `poItem.UnitPriceBHD` — resolvers NOT called | **Wire OSS 3-way match to `inventory.SupplierInvoiceItemUnitPriceBHD` + `ResolvePurchaseOrderItemReferenceCost`.** Two live consequences: (1) a PO line with 0 `UnitPriceBHD` is skipped (`>0` guard) instead of falling back to product standard cost → variance silently unchecked; (2) non-BHD invoice with `ExchangeRate==0` computes unit price = 0 → false 100% variance. Band-2 consumer unwired (matches known-open). |
| Delivery/dispatch → stock OUT decrement | ABSENT (both) — parity | Neither `delivery_note_service.go` nor `serial_number_service.go` calls `RecordStockMovement` / decrements `QuantityOnHand`; DN dispatch tracks serials only | `delivery_note_service.go` has no inventory/stock refs either | none — PH itself does NOT wire DN→stock-out, so OSS is at parity. NOT a gap vs PH. (The "OUT-side may not exist in OSS" note resolves to: adjustment-based OUT exists in both; DN-based OUT exists in neither.) |

- **Counts:** PRESENT 13 · DIVERGENT-INTENTIONAL 0 · DIVERGENT-GAP 1 · ABSENT 0 (1 parity-absent, both sides) · EXTRA 0

- **Top gaps to close (money/integrity first):**
  - **OSS 3-way match bypasses the reference-cost resolvers** (`supplier_invoice_service.go:739–748`). It uses raw `poItem.UnitPriceBHD` and an inline FX multiply, so (a) zero-priced PO lines escape price validation instead of falling back to product standard cost, and (b) a non-BHD supplier invoice with a missing/zero exchange rate collapses the unit price to 0 and fires a spurious 100% variance flag. This is the last unwired Band-2 consumer in inventory. Financial-control accuracy — surface as bug.

- **Uncertainties / needs-Commander:**
  - `GetInventoryItems`/`GetStockMovements`/`GetInventoryItem` bind `warehouseID`/`itemID` as `*uint` in OSS vs `*string` in PH. Item PK is queried as a string in both, so the mismatch only affects the warehouse/id *filter* path. Low risk, but if the Svelte layer passes string warehouse IDs the OSS filter would no-op. (unverified — needs a UI-binding check.)
  - Valuation engine is a byte-faithful port (float64, no intermediate rounding, rounding only at `GetInventoryValuation` reporting edge via `roundBHD`). No customer-facing document appearance touched. No stop-and-ask.
## Documents / OCR / Inbox / Import

| Element (flow) | Classification | PH evidence | OSS evidence | Action |
|---|---|---|---|---|
| SimpleOCRService core (dispatch by ext: image/Excel/MSG/EML/DOCX/RTF/PDF) | PRESENT | ocr_service_simple.go:112 ProcessDocument | ocr_service_simple.go:105 ProcessDocument | none |
| 3-PARSE robustness: RTF control-code stripper (hex/unicode/binary body) | PRESENT | ocr_service_simple.go:936 extractTextFromRTF | ocr_service_simple.go:929 extractTextFromRTF (byte-for-byte) | none |
| Image byte-handling (PNG/JPG/BMP/TIFF/WEBP → base64 mime) | PRESENT | ocr_service_simple.go:283 processImage | ocr_service_simple.go:276 processImage | none |
| ZIP/OOXML byte-handling (.docx via nguyenthenguyen/docx; .xlsx) | PRESENT | ocr_service_simple.go:848 processDOCX | ocr_service_simple.go:841 processDOCX | none |
| MSG parser (OLE compound doc, prop streams, RTF body capture) | PRESENT | ocr_service_simple.go:501 processMSG | ocr_service_simple.go:494 processMSG | none |
| UTF-16LE decode / EML RFC5322 / vector-PDF detect | PRESENT | ocr_service_simple.go:1033/776/1311 | ocr_service_simple.go:1026/769/1304 (vector-PDF delegated to pkg/documents/ocr) | none |
| Fly.io Runtime OCR client + Mistral vision fallback | PRESENT | ocr_service_simple.go:1090 callFlyOCR / :1625 ocrWithMistralVision | ocr_service_simple.go:1083 / :1536 | none |
| Inbox document processing endpoint + RBAC guard | PRESENT | runtime_handlers.go:38 ProcessInboxDocument (`documents:classify`) | runtime_handlers.go:37 same guard | none (Phase-33 no-auth issue closed in OSS) |
| **Inbox LOCAL OCR fallback when Runtime unreachable** | **DIVERGENT-GAP** | runtime_handlers.go:68 → :117 processInboxDocumentLocally runs `ocrService.ProcessDocument(absPath,"auto")` (real offline extraction) | runtime_handlers.go:67-77 returns stub: type="unknown", confidence 0.0, "Review manually" — NO local OCR path | Port processInboxDocumentLocally; offline-first invariant otherwise broken for inbox |
| RFQ-from-OCR (normalized line-item population) | PRESENT | app.go:15916 "RFQ has N normalized line items from OCR/Butler" | app_setup_documents_surface.go:2947 same | none |
| Document classifier (RBAC + symlink + home-dir guards) | PRESENT | document_classifier.go (1302 L) | document_classifier.go (1320 L) — identical func set | none |
| Data archaeology scan (entity relationship discovery) | PRESENT | archaeologist.go (908 L) | archaeologist.go (900 L) — identical func set | none |
| OneDrive import — core flow (scan→candidates→offer/order derive→import) | PRESENT | onedrive_import_service.go:importSingleDeal, findOneDriveOpportunityCandidates | onedrive_import_service.go:1483 importSingleDeal, :841 findOneDriveOpportunityCandidates | none |
| Division stamp on import (opportunity + invoice) | PRESENT | onedrive_import_service.go:2457 invoice.Division = normalizeDivisionName | onedrive_import_service.go:1723 (opportunity) + :1978 (invoice) | none |
| Folder-number digit-guard — PARSE level (D1) | PRESENT | onedrive_import_service.go:806 folderNumberHasDigit @ loose fallback | onedrive_import_service.go:668 same D1 guard (cites "PH 10f96a7") + :2035 folderNumberHasDigit | none |
| Folder-number digit-guard — LIST/enrich level | DIVERGENT-GAP | app.go:5026 (normalizeOpportunityForList) adds `folderNumberHasDigit(meta.FolderNumber)` before overwrite — comment: collapse "hid ~83 pipeline opportunities" | app_sales_pipeline.go:282 normalizeOpportunityForList overwrites FolderNumber WITHOUT the digit guard | Add `folderNumberHasDigit()` to OSS:282 (belt-and-suspenders; parse-guard covers most cases) |
| Loose folder-number token cleaner (D2) | ABSENT | onedrive_import_service.go:810 cleanLooseOneDriveFolderNumberToken + :826 splitLooseOneDriveFolderNumberToken | not present (only folderNumberHasDigit ported) | Port cleanLoose/splitLoose — known PARTIAL residue, pairs with D1 |
| Sequence-token normalization + parent folder-number inheritance | ABSENT | onedrive_import_service.go:843 normalizeOneDriveSequenceToken, :853 inheritOneDriveParentFolderNumber (wired at :700/:733/:1615/:1635) | not present | Port; affects seq keying + nested-folder number inheritance |
| Discovery-walk hardening (batch containers, nested child deals, direct-file deals, merge/dedupe) | ABSENT | mergeDiscoveredFiles, isOneDriveBatchContainerName, hasExtractableOneDriveChildDeal, collectImmediateDealFiles, isDirectFileOneDriveDealPath, hasNumberedOneDriveFolderName, hasOneDriveDealRelevantFiles, uniqueSortedSectionPaths (all wired in scan loop ~:1615-1758) | none of these exist; OSS scan uses simpler extractDealSectionPaths/collectDealFiles only | Port cluster if messy real-world OneDrive trees (batch folders, direct-file deals) matter for cutover |
| Section-name aliasing / canonicalization | ABSENT | onedrive_import_service.go:469 oneDriveFolderHasAlias, canonicalOneDriveSectionName (:578/:617) | none | Port for alias-named section folders |
| Offer-# / folder-# disambiguation on collision | ABSENT | disambiguateImportedOfferNumber (:2191), disambiguateOneDriveOpportunityFolderNumber (:2074/:2102), selectOneDriveOpportunityCandidatesForDeal (:2063) | none | Port; without it colliding imports may cross-link |
| Costing-workbook classification at path | ABSENT | classifyDiscoveredFileAtPath (:670/:1811), isPotentialOneDriveCostingWorkbook | none | Port for costing-file routing during scan |
| extracted_documents table (document ingestion store) | ABSENT | referenced only in sync_coverage_service.go:207 (sync table-name list; NO GORM model in PH either — legacy raw table) | zero references anywhere; OSS document store = OCRDocument + QuickCapture (AutoMigrated in app_setup_documents_surface.go:2387/2577) | Confirm whether `extracted_documents` is live in deployed PH DB; if so add model+AutoMigrate. Fresh-provision gap flagged in brief holds — table not created on fresh OSS DB |
| Tally importer / import_2026 seed | PRESENT | tally_importer.go (1045), import_2026_data.go (825) | tally_importer.go (1025), import_2026_data.go (822) — near-identical | none |

**Counts:** PRESENT 15 · DIVERGENT-INTENTIONAL 0 · DIVERGENT-GAP 2 · ABSENT 6 · EXTRA 0

**Top gaps to close (integrity first):**
- **Inbox local-OCR fallback ABSENT** (runtime_handlers.go:67 stub vs PH :117 real local extraction) — biggest gap: OSS inbox is fully dependent on the remote Fly.io Runtime; when it's down, documents get type="unknown"/0.0 confidence and are never extracted, despite a fully-capable local SimpleOCRService sitting right there. Directly violates the offline-first invariant.
- **Folder-number digit-guard incomplete (D1/D2 PARTIAL)** — parse-level D1 is ported (OSS:668) but (a) the list-level guard is missing (app_sales_pipeline.go:282 overwrites FolderNumber with no `folderNumberHasDigit` check) and (b) `cleanLooseOneDriveFolderNumberToken`/`splitLooseOneDriveFolderNumberToken` (D2) never ported. PH added these because a digit-less token collapsed ~83 pipeline opportunities onto one key and cross-linked costings. Low live risk (parse-guard covers the common path) but it is the exact residue the brief flagged.
- **OneDrive discovery-walk + disambiguation hardening ABSENT** (~18 PH functions: batch containers, nested/direct-file deals, section aliasing, offer/folder-# disambiguation, sequence normalization, parent-# inheritance, costing-workbook classification). Core import works; these are the messy-real-world-tree refinements. Matters most if a live OneDrive re-scan is part of the PH cutover, since without them some deals are missed/misrouted and colliding numbers can cross-link.

**Uncertainties / needs-Commander:**
- `extracted_documents` provisioning: PH has NO GORM model for it (only a sync-table-name string in sync_coverage_service.go:207) — so it's a legacy raw table that may or may not carry live data in the deployed Postgres. Whether OSS must recreate it depends on whether the deployed PH DB actually populates it. Needs a look at the live schema before deciding model+AutoMigrate. (unverified — needs closer look)
- The ~18-function OneDrive discovery cluster: classified ABSENT by presence, but I did not diff behavior deal-by-deal against a real folder tree — the practical impact (how many deals OSS's simpler scan would miss vs PH) is unquantified. (unverified — needs closer look)
## Reporting & Documents (PDF / CSV / exports)

Scope: every PDF/CSV/export generator in deployed PH vs OSS — invoice, PO, offer/quotation,
credit-note, delivery-note/GRN, VAT reconciliation, costing exports, analytics/report exports,
dashboard reports. PH = `C:\Projects\asymmflow\ph_holdings` (ui-ux-hardening @ ca24372).
OSS = `C:\Projects\asymmflow\asymmflow-oss` (main @ 801f41a).

| Element (flow) | Classification | PH evidence | OSS evidence | Action |
|---|---|---|---|---|
| Customer invoice PDF — letterhead / company identity | DIVERGENT-INTENTIONAL | invoice_pdf_service.go:4,98 ("PH Trading") | invoice_pdf_service.go:4,84 ("Acme Instrumentation") | none (overlay letterhead) |
| Customer invoice PDF — 40mm top margin (was 50mm) | PRESENT | invoice_pdf_service.go:148,163 SetTopMargin(40)/SetY(40) | invoice_pdf_service.go:134,149 identical | none (Wave 1 shipped, confirmed both) |
| Supplier invoice PDF — 50mm top margin retained | PRESENT | invoice_pdf_service.go:777 SetTopMargin(50) | invoice_pdf_service.go:766 SetTopMargin(50) | none (50 vs 40 confirm: supplier=50 both) |
| Invoice PDF — buyer Attention fallback (no customer addr) | PRESENT | invoice_pdf_service.go:342-353 | invoice_pdf_service.go:328-339 (byte-identical) | none (shipped) |
| Invoice PDF — ref truncation helper (truncatePDFTextToWidth) | PRESENT | invoice_pdf_service.go:998 | invoice_pdf_service.go:1062 (identical, moved) | none (shipped) |
| Invoice PDF — field-visibility default fallback | PRESENT | invoice_pdf_service.go:54-88 hardcoded fallback | invoice_pdf_service.go:54-80 `defaults` (same values) | none (equivalent) |
| **Invoice PDF — signature block appearance** | **DIVERGENT-GAP (cosmetic, customer-facing)** | invoice_pdf_service.go:640-647: NO signer line/label; `SetY(declY+20)` leaves ~20mm blank for a physical signature (SPOC #6: signer name/designation is internal jargon, must not print) | invoice_pdf_service.go:626-637: draws `"________________________"` underline + Helvetica-7 `"Authorized Signatory"` label at x=130 | **STOP-AND-ASK** — OSS prints a signature underline + "Authorized Signatory" that PH deliberately removed |
| **Invoice PDF — bank-details fetch RBAC/seed** | **DIVERGENT-GAP (functional/security)** | invoice_pdf_service.go:129 → `getActiveBankAccountsForDocuments()` bank_accounts_service.go:181 (no finance:view gate; auto-seeds if empty) | invoice_pdf_service.go:115 → `GetActiveBankAccounts()` bank_accounts_service.go:155 (requires `finance:view`; no seed fallback) | close — OSS invoice PDF fails / omits banks for an operator holding only invoices:view, and shows no banks when table empty |
| Invoice PDF — division-bank filter (PH banks not on AHS inv) | PRESENT | invoice_pdf_service.go:668 | invoice_pdf_service.go:658 (same logic) | none (shipped) |
| Credit-note PDF (GenerateCreditNotePDF) — core | PRESENT | credit_note_service.go:356 | credit_note_service.go:325 | none |
| Credit-note numbering (CN-YYYYMMDD-seq, tx-locked) | PRESENT | credit_note_service.go:314-352 inline seq | pkg/documents/numbering/numbering.go:78,154 (Engine.Next, same format) | none (mechanism decomposed) |
| **Credit-note PDF — signature block** | **DIVERGENT-GAP (cosmetic, customer-facing)** | credit_note_service.go: `resolvePreparedBySignatureBlock` + `drawPHSignaturePDFLines(pdf,130,y,60,3.5,6.6,…)` named prepared-by block | credit_note_service.go: `Cell(60,5,"For <LegalName>")` + `Cell(60,5,"Authorized Signatory")` | **STOP-AND-ASK** — named signer block vs generic "For LegalName / Authorized Signatory" |
| **Delivery-note PDF — signature block + footer geometry** | **DIVERGENT-GAP (cosmetic, customer-facing)** | delivery_note_service.go: `signatureBlockHeight=34.0`, `deliveryNoteFooterSafeY=286.0`, `drawPHSignaturePDFLines(pdf,35,signatureY+10,70,2.45,5.1,…)` named prepared-by | delivery_note_service.go: `signatureBlockHeight=22.0`, `deliveryNoteFooterSafeY=268.0`, `SetXY(35,signatureY+12); Cell(70,5,"For <LegalName>")` | **STOP-AND-ASK** — 12mm shorter block, footer-safe-Y 18mm higher, generic "For LegalName" vs named block |
| PO PDF & Offer PDF — signature block | DIVERGENT-GAP (cosmetic) | purchase_order_pdf_service.go:621 `resolvePreparedBySignatureBlock`; offer via offer_signature_blocks.go | offer_signature_blocks.go **ABSENT**; OSS PO/offer generic | STOP-AND-ASK (same signature-helper family) |
| Signature-helper library (offer_signature_blocks.go) | ABSENT | offer_signature_blocks.go: resolveDocumentSignerName / resolvePreparedBySignatureBlock / drawPHSignaturePDFLines | file absent; no equivalent in pkg/ | root cause of all 4 signature deltas above |
| **PO PDF — DRAFT/PENDING watermark banner** | **DIVERGENT-GAP (document integrity)** | purchase_order_pdf_service.go:68-69,114-120: `canonicalizePurchaseOrderStatus`; red banner "DRAFT / PENDING APPROVAL - NOT VALID FOR SUPPLIER ISSUE" | no banner; PO PDF renders identically regardless of approval state | close — OSS can emit a supplier PO PDF with no unapproved-draft marking |
| PO PDF — supplier part number line in item desc | DIVERGENT-GAP (minor) | purchase_order_pdf_service.go:319-321 appends "Supplier Part: …" | absent | low priority |
| PO PDF — bank details (hardcoded BBK/NBB vs Demo Bank) | DIVERGENT-INTENTIONAL | purchase_order_pdf_service.go:522-540 (BBK/NBB, real IBANs) | :510-528 (Demo Bank D, demo IBANs) | none (overlay demo data) |
| Offer/quotation PDF — bundled costing datasheet attachments | ABSENT | offer_pdf_service.go:74-81 attaches `listCostingSheetAttachmentsByScope` into exportData | offer PDF omits attachment-scope block entirely | close if attachment feature desired |
| Costing PDF export (ExportCostingToPDF) | PRESENT | costing_pdf_bundle_service.go + engines | app_costing_exports_surface.go:572-579 | none |
| Costing Excel export (ExportCostingToExcel) | PRESENT | costing_excel_export_test.go / engine | app_costing_exports_surface.go:1138 | none |
| **Costing PDF datasheet BUNDLING (merge attached PDFs)** | **ABSENT** | costing_pdf_bundle_service.go (whole file: appendCostingPDFDatasheets, pdfcpu/gofpdi merge, 8 fns) | no counterpart anywhere in repo | PH merges supplier datasheet PDFs into the costing/offer PDF; OSS cannot |
| **Costing sheet attachment service** | **ABSENT** | costing_attachment_service.go (AttachCostingSheetFile, ListCostingSheetAttachments, 15 fns) | no counterpart | feeds offer-PDF bundling above |
| VAT reconciliation report (GetVATReconciliation) | PRESENT | finance_reporting_service.go:694 | finance_reporting_service.go:538 | none |
| **Payment aging drill-down (GetInvoicesByAgingBucket)** | **ABSENT** | finance_reporting_service.go:196 + helper agingBucketForDueDate:58 | no counterpart anywhere in repo | close — dashboard aging bucket has no drill-down list in OSS |
| Payment aging summary (GetPaymentAgingReport) | PRESENT | finance_reporting_service.go:75 | finance_reporting_service.go:48 | none |
| Cashflow projection / margin-by-customer / margin-by-product | PRESENT | finance_reporting_service.go:316,560,616 | finance_reporting_service.go:169,404,460 | none |
| Dashboard report PDF (GenerateDashboardReport) | PRESENT | reports.go:257 | reports.go:254 | none |
| Prediction-history / Customer-360 report | PRESENT | reports.go:365,456 | reports.go:362,453 | none |
| Sales/customers/operations/inventory/financial report data | PRESENT | reports.go:802-1072 | reports.go:770-1040 | none |
| **Analytics report EXPORT — RBAC guard** | **DIVERGENT-GAP (security)** | reports.go:1159-1161 `requireReportAccess(reportType)` before export | ExportReport reports.go:1125 has NO access check | close — any user exports any report type |
| **Analytics report EXPORT — filename sanitization** | **DIVERGENT-GAP (security)** | reports.go:1185 `sanitizeFileName(reportType)` into filename | OSS interpolates raw `reportType` into filename (reports.go:1131) | close — path/name-injection via reportType |
| Analytics report EXPORT — payload size cap | DIVERGENT-GAP | reports.go:1166 `maxReportExportPayloadBytes` guard + isKnownExportReportType | none | close — unbounded export payload |
| Analytics report EXPORT — PDF format | ABSENT | reports.go:1493 exportReportToPDF + drawCommercialExtractPDF:1521 + drawSummaryReportPDF:1553 | ExportReport returns "PDF export coming soon" (reports.go:1139) | feature gap |
| Analytics report EXPORT — Excel format | ABSENT | reports.go maps excel→csv (functional) | returns "Excel export coming soon" | feature gap |
| Analytics report EXPORT — finance-pack CSV | ABSENT | reports.go:1303 exportFinancePackToCSV | no counterpart | richer finance CSV missing |
| Analytics report EXPORT — commercial-extract CSV | ABSENT | reports.go:1429 exportCommercialExtractToCSV | no counterpart | richer commercial CSV missing |
| Analytics report EXPORT — basic CSV | PRESENT | reports.go:1632 exportReportToCSV | reports.go:1147 exportReportToCSV | none |
| Report runway figure display | DIVERGENT-GAP (reporting correctness) | report_generators.go: guards `RunwayMonths>=0` → "N/A" when unknown/negative | report_generators.go: unconditional `%.1f months` (prints e.g. "-1.0 months") | close — OSS shows nonsense runway |
| Butler reports (butler_reports.go) | PRESENT | butler_reports.go (793 ln) | butler_reports.go:591 (same App fns) | none (internal decomposition) |
| Report storage / GRN service / delivery CRUD | PRESENT | report_storage.go, grn_service.go, delivery_note_service.go | identical / pkg/crm/fulfillment | none (mechanism differs) |
| Bank-statement EXPORT generator | N/A (neither) | import-only (bank_statement_parser.go) | import-only | not a document generator |
| CRM data export | PRESENT | manual_onedrive_seed_export_test.go / exports surface | manual_onedrive_seed_export_test.go present | none |

**Counts:** PRESENT 20 · DIVERGENT-INTENTIONAL 2 · DIVERGENT-GAP 9 · ABSENT 8 · EXTRA 0

**Top gaps to close (money/integrity first):**
- **Invoice-PDF bank fetch RBAC/seed** (DIVERGENT-GAP): OSS `GenerateInvoicePDF` calls `GetActiveBankAccounts()` which requires `finance:view` and never seeds — an operator with only `invoices:view` (who passed the doc gate) gets an error or an invoice with no bank/payment details. PH added `getActiveBankAccountsForDocuments()` precisely to avoid this. Customer would receive an unpayable invoice.
- **PO-PDF DRAFT/PENDING watermark absent** (integrity): OSS can print a supplier-ready PO PDF with no "NOT VALID FOR SUPPLIER ISSUE" banner on an unapproved/draft PO — a real procurement-control hole PH closed.
- **Report-export security** (2 gaps): OSS `ExportReport` has no `requireReportAccess` RBAC and no filename sanitization — any user can export any report type and steer the output filename via unsanitized `reportType`.
- (Integrity/completeness) **GetInvoicesByAgingBucket ABSENT**: no receivables aging drill-down in OSS (matches the open "dashboard aging" whistleblow); and **costing datasheet bundling ABSENT** (offer PDFs lose merged supplier datasheets).

**Uncertainties / needs-Commander (customer-facing document appearance — STOP-AND-ASK, do NOT byte-change):**
The signature block diverges on EVERY counterparty-facing PDF because `offer_signature_blocks.go` (resolveDocumentSignerName / resolvePreparedBySignatureBlock / drawPHSignaturePDFLines) was never ported to OSS. Measured deltas for the certificate:
- **Invoice:** PH omits any signer line, reserves ~20mm blank (`SetY(declY+20)`) for a physical signature (SPOC #6). OSS draws an underscore rule `"________________________"` + Helvetica-7 `"Authorized Signatory"` at x=130. → OSS shows a label PH deliberately deleted.
- **Credit note:** PH draws a named prepared-by block via `drawPHSignaturePDFLines(pdf,130,y,60,3.5,6.6,…)`. OSS draws `"For <LegalName>"` + `"Authorized Signatory"`.
- **Delivery note:** PH `signatureBlockHeight=34`, `footerSafeY=286`, named prepared-by block at (35,signatureY+10). OSS `signatureBlockHeight=22`, `footerSafeY=268`, `"For <LegalName>"` at (35,signatureY+12). → 12mm shorter block, footer-safe boundary 18mm higher on the page.
- **PO / Offer:** same helper family; OSS generic. (unverified exact OSS PO signature text — helper simply absent.)
These are appearance-only and must be MEASURED not modified; flagging for the certificate, not for a byte edit.
## Settings / RBAC / Users / Auth / Sync / Bank-Accounts

| Element (flow) | Classification | PH evidence | OSS evidence | Action |
|---|---|---|---|---|
| Users CRUD — Create/Update/Deactivate/List/Get | PRESENT | app.go (CreateUser/UpdateUser/DeactivateUser) | app_auth_rbac.go:341/418/456/298/321 | none |
| CreateUser input validation (XSS/length/format) | PRESENT | ValidateUserInput (Phase 33 verified) | app_auth_rbac.go:358-368 (GlobalValidator.ValidateUserInput + dept/title) | none |
| ResetUserPassword | PRESENT | app.go | app_auth_rbac.go:482 (requirePermission users:update, complexity, must_change) | none |
| Password security — bcrypt cost 12, complexity, crypto/rand gen | PRESENT | field/license services | app_auth_rbac.go:230-295 (bcryptCost=12, validatePasswordComplexity, generateSecurePassword) | none |
| Roles — SeedDefaultRoles / ListRoles / GetRole | PRESENT | app.go RBAC seed (5 roles) | app_auth_rbac.go:18/192/210 (admin/manager/sales/operations/staff) | none |
| requirePermission server-side middleware | PRESENT | app.go (guards ~70 fns) | app_auth_rbac.go:530 (user→license fallback, alias/category match) | none |
| Startup-import RBAC bypass scoped + 5-min timeout | PRESENT | app.go:11433 (Phase 17 P0) | app_auth_rbac.go:533-555 (importAllowedPerms allowlist + timeout) | none |
| Permission matching — aliases, category, wildcard `:*` | PRESENT | app.go permissionGranted | app_auth_rbac.go:622-695 (permissionAliases, permissionGranted) | none |
| checkUserPermission / HasPermission / CheckPermissionByRole | PRESENT | app.go | app_auth_rbac.go:729/763/964 | none |
| GetUserPermissions — role∪license merge | PRESENT | app.go | app_auth_rbac.go:799-851 | none |
| **RBAC guards on ~20 mutators (Wave 1)** | PRESENT (verified) | n/a | user CRUD, bank CRUD, watcher/sync (app_watcher.go:245/261/297/368/396/427), SeedRoles, SeedBank | none — Wave 1 shipped |
| Login (device) | PRESENT | device_service.go:691 LoginDevice | device_service.go:296 LoginDevice | none |
| Logout (recursion fix) | PRESENT | auth_handler.go:315 / auth_session.go:380 | auth_handler.go:303 / auth_session.go:375 | none |
| Session timeout | PRESENT | 8-hour auto-logout (CLAUDE.md guard) | session_inactivity.go (30-min inactivity via touchInteractiveSession @ app_auth_rbac.go:560) | none |
| OAuth token stored as SHA-256 hash | PRESENT | auth_session.go (Phase 17) | auth_session.go (hashToken) | none |
| Audit log — GetAuditLogs / logAudit | PRESENT | app.go:19043 GetAuditLogs (inline) | app_auth_rbac.go:1020 GetAuditLogs; logAudit routed via pkg/infra/audit.Recorder:1054-1088 | none (OSS = Wave 3 single-path recorder, behavior superset) |
| Notifications — MarkNotificationAsRead | PRESENT | collaboration_service.go:1573 | collaboration_service.go:1373 | none |
| Bank accounts CRUD — Create/Update/Delete/Get/GetActive/GetAll | PRESENT | bank_accounts_service.go:242/302/344 (RBAC finance:*) | bank_accounts_service.go:210/270/312/155/191 (same guards) | none |
| SeedCompanyBankAccounts — admin `*` hard-guard (RBAC-003) | PRESENT | bank_accounts_service.go:48-53 | bank_accounts_service.go:32-37 (internal variant for startup) | none |
| DeleteBankAccount — linked-statement block | PRESENT | bank_accounts_service.go (deactivate if linked) | bank_accounts_service.go:321-331 | none |
| **Bank-account field encryption (IBAN/acct#)** | DIVERGENT-INTENTIONAL | field_crypto encrypts seeded IBAN/acct | OSS removed HW-bound crypto; plaintext + MigrateBankAccountEncryption strips leftover (bank_accounts_service.go:344-437) | none — IBANs print on invoices, not secret; HW-bound crypto broke DB portability |
| Field crypto — HKDF-SHA256 + AES-256-GCM, PBKDF2 600k, **random salt** | PRESENT | field_crypto.go (identical) | field_crypto.go (byte-identical; loadOrCreateSalt random, no deterministic fallback:73-76) | none — salt concern resolved, both random |
| Field-crypto key export/import/rotate | PRESENT | field_crypto.go | field_crypto.go:230/268/283 | none |
| Settings — GetSettings / UpdateSettings | PRESENT | app.go:13030/13125 | app_setup_documents_surface.go:75/168 (requirePermission settings:update) | none |
| SettingsService — encrypted persistence (HKDF/AES, legacy-compat) | PRESENT | settings_service.go | settings_service.go:27-50 | none |
| DB-sync settings — Get/Update | PRESENT | db_sync_service.go:249/278 | db_sync_service.go:163/192 | none |
| **Cloud sync engine** | DIVERGENT-INTENTIONAL | db_sync_service.go bidir self-hosted Postgres (36 tables); 3 legacy Postgres hardenings | pkg/sync/turso/cdc.go forward path + db_sync_service.go retained | none — pre-ratified; PH Postgres hardenings moot by design |
| File-watcher + sync control APIs (Start/Stop/Trigger/Retry/Clear) | PRESENT | app.go watcher/sync APIs | app_watcher.go:245-441 (all RBAC settings:update) | none |
| HMAC document integrity — compute/backfill/verify (Wave 1) | PRESENT (verified) | field_crypto.go HMAC-SHA256 + customer_invoice_service.go | customer_invoice_service.go:36/80/119/128 (BackfillInvoiceHashes+internal, VerifyInvoiceHash); invoice_hash_verify_test.go | none — Wave 1 shipped |
| Company/division settings + branding | PRESENT | company_branding.go + hardcoded divisions | company_branding.go + pkg/overlay (divisions/VAT/FX/aliases config) | none |
| Reconciliation tooling — data + bank + book-bank | PRESENT | data_reconciliation.go, bank_reconciliation_service.go, book_bank_reconciliation_service.go | same files present + pkg/finance/banking | none |
| License-based auth — ValidateLicense/GetLicenseRole/HasLicensePermission | PRESENT | license_service.go | license_service.go + app_auth_rbac.go:604-608 | none |
| Delete-permission admin-only enforcement in middleware | EXTRA | (delete approval separate) | app_auth_rbac.go:564-571 (isDeletePermission → admin-role-only) | note — OSS hardening |
| Runtime integration secret-settings persistence | ABSENT | runtime_integration_settings.go:126/137 (persistSecretSetting, applyPersistedRuntimeSettings, getStoredSettingValue) | no counterpart found | low-pri: OSS has no encrypted persist path for runtime API keys (OCR/Mistral) beyond in-memory config/env |

### Counts
- **PRESENT:** 26 · **DIVERGENT-INTENTIONAL:** 3 · **DIVERGENT-GAP:** 0 · **ABSENT:** 1 · **EXTRA:** 1

### Top gaps to close (money/integrity first)
- **Runtime integration secret-settings persistence (ABSENT, low-pri):** PH's `runtime_integration_settings.go` (`persistSecretSetting`/`applyPersistedRuntimeSettings`) encrypts and persists runtime API keys (OCR/Mistral) into settings; OSS only has in-memory `config` + env. No financial/integrity impact — AI/OCR is optional and disabled-by-default in OSS — but a deployed operator cannot save an API key through the UI and have it survive restart via encrypted storage. Confirm whether this is intended (env-only config) before porting.

### Uncertainties / needs-Commander
- **None financial.** RBAC (~20 mutators), HMAC verify+backfill, and sync hardenings from Wave 1 are all verified PRESENT. Bank-account encryption removal and Turso/CDC sync are pre-ratified DIVERGENT-INTENTIONAL and correctly classified — no action.
- Bank-account plaintext divergence touches customer-facing invoice appearance only insofar as IBAN/account numbers are printed (already the design intent); no stop-and-ask needed.
## Startup / Scheduled Jobs / Fresh-Provision Schema Parity

Deployed PH `ui-ux-hardening@ca24372` vs OSS `main@801f41a`. Both apps share a
common `startup()` fork, so boot-time jobs are near-identical; the real divergence
is in the **fresh-provision model set** (part 2).

### Part 1 — Startup & scheduled jobs

| Element (flow) | Classification | PH evidence | OSS evidence | Action |
|---|---|---|---|---|
| Boot AutoMigrate (skipped when >50 tables) | PRESENT | app.go:700-886 (inline model list) | app.go:579-604 (via `tradingModels()`+composition seam) | none |
| Critical-deployment foundation migrate (always-run) | PRESENT | app.go:889 `ensureCriticalDeploymentFoundations` | app.go:607 `ensureCriticalDeploymentFoundations` → deployment_audit.go:126 | none |
| Custom migrations (column adds, seeds; skipped >50 tbl) | PRESENT | app.go:913-919 `runCustomMigrations` | app.go (runCustomMigrations; CompanyBankAccount@1265, CurrencyExchangeRate@1272, BankLinePaymentAllocation@1421) | none |
| RFQ document-tracking backfill | PRESENT | app.go:1088-1112 `BackfillRFQDocumentTracking` | app.go:808-834 | none |
| Business customer-ID backfill | PRESENT | app.go:1234 `BackfillBusinessCustomerIDs` | app.go:958 | none |
| Offer-items cost-breakdown backfill | PRESENT | app.go:1250 `BackfillOfferItemCostBreakdown` | app.go:976 | none |
| Won-offer-items-from-opportunity backfill | PRESENT | app.go:1259 `BackfillWonOfferItemsFromOpportunityProductDetails` | app.go:985 | none |
| Invoice-items-from-orders backfill (SHIPPED) | PRESENT | app.go:1266 `BackfillInvoiceItemsFromOrders` | app.go:992 | none |
| Invoice-hash (HMAC) backfill (SHIPPED) | PRESENT | app.go:1750 `backfillInvoiceHashesInternal` | app.go:1001 `backfillInvoiceHashesInternal` | none |
| Hollow-invoice send-block (SHIPPED, guard not backfill) | PRESENT | invoice send path | pkg/finance invoice guard | none |
| Opportunity product-details backfill (from offers) | PRESENT | app.go:1643 `backfillOpportunityProductDetailsFromOffers` | app.go:1319 | none |
| Opportunity division backfill | PRESENT | app.go:1776 | app.go:1406 | none |
| Bank-statement metadata backfill | PRESENT | app.go:1815/1911 `backfillBankStatementMetadata` | app.go:1432/1540 | none |
| Division-aware finance backfill | PRESENT | app.go:1816/1971 `backfillDivisionAwareFinanceData` | app.go:1433/1600 | none |
| License-key + employee-key seeding | PRESENT | app.go:897-908 | app.go:615-618 (overlay-gated `SeedLicenseKeys`) | none |
| Startup DB integrity check (`PRAGMA integrity_check`) | PRESENT | app.go:922 `runIntegrityCheck` | app.go (runIntegrityCheck) | none |
| Scheduled backup if due (`VACUUM INTO`, keep 7) | PRESENT | app.go:928 `runScheduledBackupIfDueInternal("startup")` | app.go:647 | none |
| Bank-account encryption migration (≤50 tbl only) | PRESENT | app.go:976-984 `migrateBankAccountEncryptionInternal` | app.go (same guard) | none |
| File watcher (OneDrive RFQ/EH/Offer/Invoice) init+start | PRESENT | app.go:1326-1352 `NewFileWatcher`/`Start` | app.go:1056-1090 (identical WatchConfig) | none |
| Background cloud-DB sync goroutine (10-min) | DIVERGENT-INTENTIONAL | `dbSyncService` / `StartBackgroundDBSync` (Postgres) | app.go:791 `newDBSyncService`; forward path is Turso/CDC `pkg/sync/turso/cdc.go` | none (pre-ratified sync divergence) |
| Tally auto-import goroutine (5-min timeout, RBAC-bypassed) | EXTRA | — | app.go:1020-1049 | none (substrate; not a PH gap) |
| Compliance in-process event bus wiring | EXTRA | — | app.go:1054 `initComplianceEventBus` | none |
| GPU detection on startup | PRESENT | app.go:951 `DetectGPU` | app.go (DetectGPU) | none |
| Graceful shutdown (WaitGroup, 1.5s bound, DB-before-log) | PRESENT | shutdown path | app.go:1125 `beforeClose` / 1161 `shutdown` | none |

**Part-1 result:** startup jobs are effectively 1:1. The only divergences are the
pre-ratified sync-path swap (Postgres→Turso/CDC) and two OSS-only extras (Tally
auto-import, compliance event bus). No startup-job gaps.

---

### Part 2 — Fresh-provision schema parity (HIGH VALUE)

**OSS fresh-provision table universe** = `tradingModels()` (trading_models.go:16-167,
pinned by `testdata/trading_schema.golden` — 89 tables) **+** `criticalDeploymentModels()`
(deployment_audit.go:83-123, always-run: expense/payroll/collab/chart_of_accounts/
account_mappings) **+** scattered runCustomMigrations `AutoMigrate` (CompanyBankAccount,
CurrencyExchangeRate) **+** OCRDocument/QuickCapture (app_setup_documents_surface.go).

**Key finding:** the entire **bank-reconciliation suite** and **FX suite** already
exist as compiled OSS models (`pkg/finance`, aliased in database.go:385-434) and are
actively read/written by live services (`bank_integrity_service.go`,
`book_bank_reconciliation_service.go`, `cheque_register_service.go`,
`cashflow_evidence_service.go`, wired in `app_services.go:28/96/191`) — but they are
in **no boot migration set**, so a from-zero OSS DB never creates the tables. This is
a pure *registration* gap (add the struct to a model set), not model authoring.

#### Tables PH has that a fresh OSS DB does NOT create

| Table (PH) | PH model | OSS model status | Classification | Where to add in OSS |
|---|---|---|---|---|
| `bank_statements` | database.go:2082 | EXISTS `finance.BankStatement` (db.go:389), unmigrated | DIVERGENT-GAP (integrity) | criticalDeploymentModels() deployment_audit.go:122 |
| `bank_statement_lines` | database.go:2122 | EXISTS `finance.BankStatementLine` (db.go:393), unmigrated | DIVERGENT-GAP (integrity) | deployment_audit.go:122 |
| `bank_accounts` | finance.BankAccount | EXISTS `finance.BankAccount` (db.go:385), unmigrated | DIVERGENT-GAP (integrity) | deployment_audit.go:122 |
| `bank_statement_files` | database.go:2323 | EXISTS `finance.BankStatementFile` (db.go:431), unmigrated | DIVERGENT-GAP | deployment_audit.go:122 |
| `statement_hashes` | database.go:2192 | EXISTS `finance.StatementHash` (db.go:410), unmigrated | DIVERGENT-GAP (dup-import guard) | deployment_audit.go:122 |
| `book_bank_reconciliations` | database.go:2218 | EXISTS `finance.BookBankReconciliation` (db.go:413), unmigrated | DIVERGENT-GAP | deployment_audit.go:122 |
| `deposits_in_transit` | database.go:2259 | EXISTS `finance.DepositInTransit` (db.go:419), unmigrated | DIVERGENT-GAP | deployment_audit.go:122 |
| `cheque_registers` | finance.ChequeRegister | EXISTS `finance.ChequeRegister` (db.go:422), unmigrated | DIVERGENT-GAP | deployment_audit.go:122 |
| `outstanding_cheques` | finance.OutstandingCheque | EXISTS `finance.OutstandingCheque` (db.go:416), unmigrated | DIVERGENT-GAP | deployment_audit.go:122 |
| `bank_reconciliation_audit_logs` | database.go:2343 | EXISTS `finance.BankReconciliationAuditLog` (db.go:434), unmigrated | DIVERGENT-GAP (audit trail) | deployment_audit.go:122 |
| `bank_cash_balances` | database.go:2147 | EXISTS `finance.BankCashBalance` (db.go:404), unmigrated | DIVERGENT-GAP | deployment_audit.go:122 |
| `bank_expense_entries` | database.go:2162 | EXISTS `finance.BankExpenseEntry` (db.go:407), unmigrated | DIVERGENT-GAP | deployment_audit.go:122 |
| `fx_rates` | database.go:2286 | EXISTS `finance.FXRate` (db.go:425), unmigrated | DIVERGENT-GAP (money) | deployment_audit.go:122 |
| `fx_revaluations` | database.go:2306 | EXISTS `finance.FXRevaluation` (db.go:428), unmigrated | DIVERGENT-GAP (money) | deployment_audit.go:122 |
| `vat_returns` | finance.VATReturn | EXISTS `finance.VATReturn` (db.go:158), unmigrated | DIVERGENT-GAP | deployment_audit.go:122 |
| `extracted_documents` | (no Go model; raw/sync table, sync_coverage_service.go:207) | ABSENT (no OSS model at all) | ABSENT | new model or skip-with-reason (data is Mission-H deferred) |
| `costing_sheet_attachments` | costing_attachment_service.go:28 | ABSENT | ABSENT | new model if attachments in scope |
| `data_quality_reviews` | user_feedback_hardening_service.go:53 | ABSENT | ABSENT | new model if DQ screen in scope |
| `employee_archive_requests` | employee_archive_service.go:13 | ABSENT | ABSENT | new model if archive flow in scope |
| `customer_receipts` / `customer_receipt_allocations` | database.go:1045/1065 | ABSENT (PC-D7 folds receipts→payments) | DIVERGENT-INTENTIONAL | none (converged into `payments`; W3 PC-D7) |
| `intelligence_opportunity_enrichment` / `_line_item_` / `_order_` | database.go:1863+ | ABSENT | DIVERGENT-INTENTIONAL (skip-with-reason) | none (PH intelligence overlay; not substrate) |
| `data_update_manifests` / `_operations` / `_receipts` | database.go:1814+ | ABSENT | DIVERGENT-INTENTIONAL (skip-with-reason) | none (PH's push-distribution; OSS uses Turso/CDC) |

**Notes on "gap to close" vs "data deferred":** every DIVERGENT-GAP row above is a
missing **TABLE** — closing it is one line (register the already-compiled struct in a
model set). The **data** that would populate these tables (imported statements, FX
history, VAT filings) is Mission-H deferred and out of scope here.

**Recommended fix site:** `criticalDeploymentModels()` (deployment_audit.go:83-123),
appended before the closing `}` at line 122/123, mirroring the ChartOfAccount/
AccountMapping precedent added there for the same "invisible-until-fresh-provision"
reason (comment at deployment_audit.go:117-120). That set runs unconditionally on
every boot, so it also repairs mature client DBs that predate the banking module —
`tradingModels()` alone (trading_models.go:165) only runs on ≤50-table fresh DBs.
Also add the 12 banking + 3 FX/VAT table names to `criticalDeploymentTableNames`
(deployment_audit.go:51-81) so the deployment audit's `MissingTables` check flags
regressions.

---

- **Counts:** PRESENT 21 · DIVERGENT-INTENTIONAL 4 · DIVERGENT-GAP 15 · ABSENT 4 · EXTRA 2
- **Top gaps to close (money/integrity first):**
  - **Bank-reconciliation suite (12 tables) unmigrated on fresh OSS DB** — the entire
    Finance-Hub Bank Reconciliation / Book-Bank / Cheque-Register / Cash-Position
    feature crashes or silent-no-ops on a from-zero install; models already exist and
    are wired, only registration is missing. One-line-per-table fix.
  - **FX suite (`fx_rates`, `fx_revaluations`) + `vat_returns` unmigrated** — multi-
    currency revaluation and VAT-return persistence have compiled models but no table;
    money-affecting.
  - **`extracted_documents` absent (no OSS model)** — OCR extraction-surface table PH
    syncs; needs either a new model or an explicit skip-with-reason (data is Mission-H).
- **Uncertainties / needs-Commander:**
  - `customer_receipts` folding into `payments` (PC-D7) is treated as
    DIVERGENT-INTENTIONAL — confirm no fresh-DB code path still expects a
    `customer_receipts` table (unverified — needs closer look at receipt_service.go).
  - Whether `costing_sheet_attachments`, `data_quality_reviews`,
    `employee_archive_requests` are in-scope for the substrate or PH-overlay-only
    (classified ABSENT pending scope call).
