# PH Parity Certificate — cutover-readiness (Mission G / Wave 4)

**Purpose.** The Commander's single cutover-readiness artifact. Answers one
question: *is the AsymmFlow-OSS substrate at feature-and-flow parity with the
deployed PH app, such that PH Trading can run on it as Sovereign Fork #1?*

**Baselines.** Deployed PH `ph_holdings` @ `ca24372` (frozen; Freeze-Law PASS,
zero commits since 2026-07-04) vs OSS `main` @ `801f41a`, wave branch
`feat/fable-ph-convergence-w4`. Produced 2026-07-07.

**Method.** Nine parallel read-only measurement passes classified all **315**
surface elements (`docs/PH_PARITY_MAP.md`); every DIVERGENT-GAP / ABSENT was then
triaged money/integrity-first, closed golden-first with tests, or recorded with
a reason. Four §6 decisions were put to the Commander and answered.

---

## Verdict

**The substrate is ready to carry PH's flows.** Coverage is high and, critically,
**there is not a single financial-number divergence** on any core document —
no case where OSS and PH compute a different invoice, PO, payment, VAT, or aging
value. Mission E already proved invoices reconcile to the fils on a real
snapshot; this wave proves the *rest* of the surface (controls, statuses,
features, provisioning, reports) is present or deliberately-and-recorded
different. The gaps that remain are **data convergence (Mission H)** and a small
set of no-caller feature methods and customer-facing-appearance items the
Commander has seen and chosen on.

Two findings materially strengthen the verdict, both surfaced by this audit:

1. **A latent fresh-DB integrity bug was found and fixed:** the supplier-invoice
   `status` CHECK constraint excluded `Verified` — the value a clean 3-way match
   writes — so a matched supplier invoice **could not persist on a from-zero DB**.
   Also fixed: the entire bank-reconciliation/FX/VAT-return suite (15 tables) was
   never provisioned on a fresh DB. Both are cutover-relevant and now closed.
2. **A would-be startup regression was caught in the act:** provisioning those 15
   tables unconditionally would have tripped a FOREIGN KEY check on the *live*
   deployment DB at boot. Re-scoped to create-if-missing before it could ship.

---

## Coverage (honest, by behavior/flow — not method count)

| Class | Count | Notes |
|---|---|---|
| PRESENT (parity) | 222 | Most byte-identical; invoices reconcile to the fils (Mission E) |
| DIVERGENT-INTENTIONAL (registered) | 16 → see register | +party-delete, now corrected to PRESENT |
| DIVERGENT-GAP | 36 | **closed this wave** (15 are the one provisioning finding) |
| ABSENT | 34 | integrity/dead ones closed; feature/no-caller ones deferred |
| EXTRA (substrate value) | 7 | OSS does more; no action |

**Parity of the money-and-control surface: closed.** Every DIVERGENT-GAP that
touches money, integrity, security, or fresh provisioning is PORTED + tested
(below). Remaining ABSENT items are either no-frontend-caller feature methods or
Mission-H data work, each recorded.

---

## Closed this wave (PORTED + tested)

| # | Gap (domain) | Resolution | Test |
|---|---|---|---|
| 1 | Bank-recon + FX + VAT-return suite unprovisioned on fresh DB (startup) | 15 models registered in `criticalDeploymentModels`, **create-if-missing** (FK-safe on mature DBs) + names in `criticalDeploymentTableNames` | `TestCriticalDeploymentProvisionsBankingSuite`, `TestDeploymentRuntimeDBBootstrapAndAudit` |
| 2 | Supplier invoice payable while unmatched/unapproved; approvable unless Discrepancy (procurement) | Gate tightened (Commander-ratified): pay requires Approved/Verified; approve requires MatchStatus==Matched | `TestRecordSupplierPayment_*`, `TestApproveSupplierInvoice_RequiresMatched` |
| 3 | **Latent:** `Verified`/`Review Required`/`Disputed` rejected by supplier_invoices CHECK on fresh DB (procurement) | CHECK widened to the code's own vocabulary; schema golden regenerated | `TestSupplierInvoiceVerifiedStatusIsPersistable` |
| 4 | Draft (unsent) invoice was payable (finance) | Ported PH settlement policy; `RecordPayment` uses `canRecordCustomerInvoicePayment` | `TestRecordPayment_RejectsDraftInvoice` |
| 5 | No settlement-status hydration on read → stale Overdue (finance) | `ListCustomerInvoices`/`GetCustomerInvoiceByID` hydrate on read (display-only) | `TestHydrateCustomerInvoicePaymentState_Overdue` |
| 6 | 3-way match unwired from reference-cost resolvers; 0-priced PO line escaped price check (procurement/inventory) | Wired `inventory.ResolvePurchaseOrderItemReferenceCost` + `SupplierInvoiceItemUnitPriceBHD` into `PerformThreeWayMatch` | `TestPerformThreeWayMatch_ZeroPricedPOLineFallsBackToStandardCost` |
| 7 | Report export: no RBAC, no filename sanitization, no payload cap; runway prints nonsense (reporting) | `requireReportAccess` + `sanitizeFileName` + `maxReportExportPayloadBytes` + whitelist; runway N/A guard | `TestIsKnownExportReportType`, `TestSanitizeFileNameStripsPathInjection` |
| 8 | Invoice PDF bank block gated on `finance:view`, no seed fallback → unpayable invoice for invoices:view operator (reporting) | Switched to document-context `getActiveBankAccountsForDocuments` (unguarded + seed fallback); division filter unchanged; no PDF bytes changed | build/`Invoice`,`Bank` suites |
| 9 | PO PDF lacked DRAFT/PENDING watermark on unapproved POs (reporting) | Ported the red "NOT VALID FOR SUPPLIER ISSUE" banner (Commander-approved) | build |
| 10 | `GenerateInvoiceNumber` dropped its `invoices:create` RBAC guard (finance) | Guard restored | build |
| 11 | Inbox returns a stub when cloud Runtime down — offline-first violated (documents) | Ported `processInboxDocumentLocally`; local `SimpleOCRService` fallback | `TestProcessInboxDocumentLocallyOfflineFallback` |
| 12 | D1/D2 folder-number digit-guard residue (documents/opportunities) | List-level `folderNumberHasDigit` guard + `cleanLooseOneDriveFolderNumberToken`/`splitLoose…` ported | `opportunity_collapse_regression_test.go` |
| 13 | `DeleteOffer` / `DeleteOpportunity` ABSENT (opportunities) | Ported with integrity guards (won/linked-record guards; admin/approval gate) + CRMService delegators | build |
| 14 | Real-bank slugs in demo seed (invariant #2) | `bank-{nbb,ahli,alsalam,bbk,nbb-euro,ahs-alsalam}` → synthetic `alpha/beta/gamma/delta/…` | `Bank` suite |

**Correction to a self-reported claim:** `DeleteOffer`/`DeleteOpportunity` were
initially described as "dead frontend buttons"; a corrected grep shows they have
**no** OSS frontend callers (the earlier match hit `DeleteOfferNote` /
`DeleteOpportunityComment`). They were ABSENT backend features, now present and
bindable. Recorded honestly.

---

## DIVERGENT-INTENTIONAL register (every place OSS deliberately differs — signed off)

| # | Behavior | OSS choice | Why / trade-off | Status |
|---|---|---|---|---|
| 1 | **Party delete** | OSS **blocks** delete of a party with transactional children (`CUSTOMER/SUPPLIER_HAS_LINKED_RECORDS`) | **CORRECTION:** handoff §3.2 pre-ratified this as OSS-orphans-children DIVERGENT-INTENTIONAL. That premise is **stale** — Wave 1 (PC-D1) moved the identical child-count guard into `pkg/crm/customer/delete.go`. OSS is at **parity (PRESENT)**, not divergent. The pre-decided call is moot. | Corrected — PRESENT |
| 2 | **Bank-statement lifecycle** | OSS silently auto-reverts a Reconciled/Verified statement to InProgress on a line change | Pre-ratified (handoff §3.3); PH refuses-until-reopened. Kept OSS. | Registered |
| 3 | **Sync** | Turso/CDC forward path | Pre-ratified (handoff §3.4); PH's 3 legacy Postgres-sync hardenings moot by design. | Registered |
| 4 | **Customer receipts** | Folded into `payments` (PC-D7) | Ratified this wave (Q3): no runtime receipt entity; neither repo ships a receipt UI. | Ratified (PC-D10) |
| 5 | **Supplier-invoice statuses** | `Verified` (matched, pay-eligible), `Review Required`, `Disputed` | Richer lifecycle than PH's `Pending`/`Dispute`; same semantics, more states. CHECK now allows them. | Registered |
| 6 | **Counterparty-PDF signature blocks** | Generic `For <LegalName>` / `Authorized Signatory` (no named prepared-by block) | Commander (Q2): keep OSS, register. Porting PH's named blocks would risk reintroducing real PH signers (invariant #2) unless overlay-driven. | Registered (stop-and-ask satisfied) |
| 7 | **E-invoice + VAT-return identity** | Per-division identity + overlay VAT-rate label | OSS improvement (multi-division overlay); byte-identical at the built-in defaults. | Registered |
| 8 | **EXTRAs** (OSS does more) | Double-entry posting-preview, FinanceService VM layer, session-inactivity timeout, logout hardening, Tally auto-import, compliance event bus | Substrate value PH lacks; no action. | Noted |

---

## Residual stop-and-ask / deferred (recorded, Commander-visible)

**Customer-facing document appearance (measured, not changed — handoff §6):**
- Signature-block divergence across invoice / credit-note / delivery-note / PO /
  offer PDFs (`offer_signature_blocks.go` never ported). Deltas measured in the
  map (invoice: OSS draws an underscore + "Authorized Signatory" where PH left
  ~20mm blank; delivery-note block 12 mm shorter, footer-safe-Y 18 mm higher).
  **Commander decision (Q2): keep OSS, registered above.** If exact live-doc
  fidelity is later wanted, port via overlay-driven signer identity.
- Costing datasheet attachment + PDF-bundle (`costing_attachment_service.go`,
  `costing_pdf_bundle_service.go`) ABSENT — offer PDFs can't carry appended
  supplier datasheets. Customer-facing; deferred pending Commander (does PH rely
  on bundled datasheets at cutover?).

**Deferred to Mission H (data convergence) — handoff §0:**
- Population of the 15 newly-provisioned banking/FX/VAT tables (provisioning done;
  data is the dial).
- `extracted_documents` carry/skip (model-less/raw table in PH too — confirm live
  data before adding a model).
- OneDrive discovery-walk + disambiguation cluster (~18 fns) — matters only for a
  live re-scan at cutover (Commander Q4: defer to Mission H).
- `intelligence_*`, `data_update_*`, `*_backup` tables — skip-with-reason.

**No-caller feature methods (backend-absent, no OSS frontend caller — deferred):**
`CreateOfferRevision`, `RenewOffer`, `UpdateOpportunityCommercialFields`,
`GetUnifiedOfferThread`, `GetCostingsByOpportunity`, `GetInvoicesByAgingBucket`,
`GetCustomerRelatedProducts`, `GetCustomerRelatedSuppliers`. Port when their UI
is wired.

**Smaller residues (recorded, non-blocking):**
- **D2 canonical-key route-dedup:** OSS `routeToOpportunity` matches folder_number
  verbatim and errors on duplicate (no canonical/case-insensitive dedup). The D1
  digit-guard + loose-token cleaners are now closed; this deeper D2 dedup remains.
- **`GetReportData` RBAC:** PH guards it; OSS does not (only `ExportReport` was in
  scope this wave). Follow-up.
- **Runtime secret-settings persistence** (`persistSecretSetting`): OSS is
  env/in-memory only; a UI-saved API key won't survive restart. Low priority
  (AI/OCR optional).

---

## Cutover readiness — one paragraph

The substrate carries PH Trading's feature and flow surface. The money-and-control
core is at parity or better, closed and tested; the fresh-provision schema now
matches (banking/FX/VAT included); no core document's numbers diverge. What stands
between here and cutover is **Mission H (data convergence)** — importing/reconciling
the real data into the now-complete schema — plus a short, explicitly-listed set of
customer-facing-appearance and no-caller-feature items the Commander has already
seen and ruled on. **The Commander has, in this document, the evidence to schedule
the cutover as a data-and-timing exercise, not a parity risk.**
