# FABLE CAMPAIGN — MISSION I: "Harden the Substrate, Enrich the Carry"

Written 2026-07-09 by the Opus 4.6 instance running the Commander's
(Sarat's) strategy session, for the Fable 5 instance that will execute
this mission. This is a MISSION spec within the PH Convergence campaign
(`FABLE_CAMPAIGN_PH_CONVERGENCE.md`). Waves 1–5 proved the substrate
carries PH's data faithfully (62/62 reconciliation, 0 UNMAPPED, 124
tables / 17,611 rows). **Mission I closes the remaining parity gaps,
ports the hardening PH earned in the field, and repairs the carried
data's known quality defects.**

The Commander is available. Ask when a decision is his; do not ask when
this document already answers it.

---

## 0. What Mission I IS and IS NOT

**IS:** The post-convergence hardening + enrichment pass. Waves 1–5
proved the carry is faithful and the financial numbers are byte-identical.
Mission I makes the substrate *operationally complete* — closing the
DIV-GAP and ABSENT items from `PH_PARITY_MAP.md` that affect real
workflows, porting field-hardening from `ph_holdings` that the substrate
never received, and repairing the data quality issues the import
faithfully carried.

**IS NOT:** A second convergence campaign. The data migration is done.
The reconciliation gates pass. This is polish, hardening, and enrichment
— not re-architecture.

**Explicit exclusions (Commander-decided):**
- OneDrive discovery-walk cluster (~18 fns) — deferred, not in scope
- Letterhead assets — separate concern, assets live in `ph_holdings`
- Sovereign mesh / P2P campaign — separate spec (`FABLE_CAMPAIGN_SOVEREIGN_MESH.md`)
- `D:\ph_data_master` backfill (355-op patch) — separate workstream

---

## 1. Invariants (inherited from the convergence campaign + new)

1. **PH is live.** `ph_holdings` is read-only reference. Zero commits
   since the 2026-07-04 Freeze Law. Verify with `git log` before starting.
2. **Real data never enters this repo.** All data work in out-of-repo
   working directories. The column_drops ledger and data quality items
   reference PH data by shape, never by content.
3. **Financial semantics are sacred.** Every money-path change gets a
   golden test. Server-derived totals, never trust client.
4. **Port through the kernel, not around it.** Use OSS's decomposed
   `pkg/` architecture. No monolith regression.
5. **NEW — Hardening ports carry their tests.** Every PH hardening fix
   ported must include a test that fails without the fix. No "trust me,
   it's the same code" — prove it.
6. **NEW — The parity map may be stale.** Recon (2026-07-09) found that
   some gaps listed in `PH_PARITY_MAP.md` are already partially closed
   (e.g., `customer_invoice_payment_policy.go` exists in OSS, just
   underwired). **Measure on the ground before porting.** Update the
   parity map as gaps close.

---

## 2. Bands (priority order)

### Band 0 — Money & Integrity Fixes (DO FIRST — silent-wrong-money risk)

These are small, high-impact fixes where OSS can currently produce
incorrect financial behavior. Most are 1–5 line changes.

| # | Gap | What | Size | File(s) |
|---|---|---|---|---|
| I-01 | Draft-payable hole | `updatePayment` uses stale inline status map missing "Draft" — payments can be edited onto unsent Draft invoices. PH blocks this via `customerInvoiceClosedWorkflowStatuses`. | **1 line** | `payment_service.go:583-587` |
| I-02 | Settlement-status hydration underwired | `customer_invoice_payment_policy.go` EXISTS in OSS (byte-for-byte equivalent to PH) but is only called on 2 read paths vs PH's 7. Missing: List, byCustomer, Overdue, Unpaid, byStatus endpoints. Past-due invoices never display "Overdue." | **~5 one-line insertions** | `customer_invoice_service.go` |
| I-03 | GenerateInvoiceNumber RBAC | OSS dropped `requirePermission("invoices:create")` guard on number generation. | **1 line** | `customer_invoice_service.go:1486` |
| I-04 | Post-payment status dead if/else | Inline branches all set "PartiallyPaid"; Overdue handling lost. Wire to settlement policy. | **small refactor** | `payment_service.go:156-181` |
| I-05 | Supplier payment eligibility gate | OSS `RecordSupplierPayment` pays unmatched/unapproved invoices. PH requires Matched + Approved. **Commander decision needed** on strictness level. | **decision + ~10 lines** | `supplier_invoice_payment_policy.go` or equivalent |
| I-06 | ApproveSupplierInvoice guard | Approved-must-be-Matched update guard missing. | **decision + small fix** | `supplier_invoice_service.go` |
| I-07 | 3-way match reference-cost resolvers | `PerformThreeWayMatch` doesn't use `inventory.*` reference-cost resolvers — match compares without reference prices. | **wire existing code** | procurement area |

### Band 1 — Field-Hardening Ports (PH lessons the substrate never learned)

These are patterns and fixes from PH's 3 hardening campaigns (cleanup →
deployment-readiness → Abhie SPOC feedback) that must be replicated in
OSS. Source docs: `SECURITY_FIXES_2026_02_16.md`,
`docs/ABHIE_MAY_OBSERVATIONS_FIX_PROOF_2026_05_22.md`,
`docs/testing/OCR_RBAC_SYNC_AUDIT_2026_04_26.md`.

| # | Theme | What to port | Scope |
|---|---|---|---|
| I-08 | Server-derived totals | Invoice Amount edit must recompute Subtotal/VAT/Grand/Outstanding server-side from line items. Payment-driven Outstanding guard runs BEFORE recompute. Never trust client-submitted totals. | **audit all invoice update paths** |
| I-09 | VAT correctness suite | Zero-rated VAT recovery on edit; PO VAT on BHD basis; 10% default when VATPercent absent. | **audit + port** |
| I-10 | Hollow-invoice guards | Block SENDING invoices with no line items (MON-007). Backfill hollow items. | **port + golden test** |
| I-11 | RBAC sweep | PH found 24 frontend-bound mutators with NO permission check. Audit every `App.*` method in OSS exposed to frontend — each needs a `requirePermission` gate. Use PH's internal/exported split pattern (privileged ops: guarded `Public` func + unguarded `*Internal` for startup/tests). | **systematic audit** |
| I-12 | Destructive-save prevention | Load-then-overlay field-mask: full-struct `Save()` calls wipe approval trails, journal links, 3-way-match flags, QC/OCR linkage on partial payloads. Every update handler must not clobber unset fields. | **audit all update paths** |
| I-13 | Stored XSS | `escapeHtml()` on every interpolated DB value in DataTable render funcs + `{@html}` sites. PH found 5 vulnerable render paths (RES-001). | **frontend audit + fix** |
| I-14 | Delete guards | Supplier/customer/opportunity delete only when child-count is zero. Verify OSS has the same guards (Wave 1 ported party-delete guards; verify opportunity). | **verify + port gaps** |
| I-15 | Transaction atomicity | DN operations wrapped in transactions. Verify all multi-table writes in OSS use transactions. | **audit** |
| I-16 | Swallowed DB errors | PH fixed silent error swallowing (RES-002/004/007). Verify OSS logs DB errors. | **audit** |
| I-17 | Credit-limit override chokepoint | `isManagementRole` helper + mandatory audit-reason persistence on credit-limit overrides. Verify OSS has this from Wave 1–3 ports. | **verify** |

### Band 2 — Feature Ports (operational completeness)

Missing features that affect PH's day-to-day workflows. Each is a
discrete port from `ph_holdings` into OSS's `pkg/` architecture.

| # | Domain | Feature | Shape |
|---|---|---|---|
| I-18 | Offers | Signature block library — `offer_signature_blocks.go` (~340 lines). Port Go logic (resolvers, DB-override, PDF renderer). **PH-specific data (6 staff blocks, +973 phones, @platinumholdings emails) goes into overlay.json**, not hardcoded. Pattern: `companyDocumentProfile` already does this for letterhead/TRN/banks. **STOP-AND-ASK on appearance/layout** — it's customer-facing, measure don't guess. | **port + overlay extraction** |
| I-19 | Offers | `DeleteOffer` (won/linked guards), `DeleteOpportunity` (admin-gated) | **port** |
| I-20 | Offers | `CreateOfferRevision`, `RenewOffer` | **port** |
| I-21 | Offers | `UpdateOpportunityCommercialFields`, `GetUnifiedOfferThread`, `GetCostingsByOpportunity` | **port** |
| I-22 | CRM | `GetCustomerRelatedProducts`, `GetCustomerRelatedSuppliers` (rollup queries) | **port** |
| I-23 | Finance | `GetInvoicesByAgingBucket` drill-down | **port** |
| I-24 | Documents | Inbox local-OCR fallback `processInboxDocumentLocally` — offline-first invariant. When cloud Runtime is unreachable, OCR locally. | **port (integrity)** |
| I-25 | Costing | Costing attachment service + PDF datasheet bundling. Ties to signature block appearance decision. | **port (decision-gated)** |
| I-26 | Reporting | Export RBAC (`requireReportAccess`), filename sanitization, payload cap, runway negative-months guard | **port security fixes** |
| I-27 | Reporting | PO-PDF DRAFT/PENDING watermark | **port** |
| I-28 | Reporting | Invoice-PDF bank fetch — operator gets unpayable invoice if bank data not seeded. Wire to overlay bank config. | **port + overlay** |
| I-29 | Procurement | `folderNumberHasDigit` guard + `cleanLooseOneDriveFolderNumberToken` (D1/D2 residue from Mission G) | **port (small)** |

### Band 3 — Data Quality Repairs (source reality, not import bugs)

These were faithfully carried by `phimport` and flagged. Repair in the
destination, not the import pipeline.

| # | Issue | Detail | Approach |
|---|---|---|---|
| I-30 | 12 wrong-date invoices | Invoices with 2026+ dates from data-entry errors. Flagged by startup audit. | **Manual review + date correction in destination DB.** Each needs Commander sign-off on the correct date (source documents). |
| I-31 | 32 offer shells | Legacy quoted/RFQ offer shells; 3 are priced with no line items. Hidden from default live list. | **Business review:** Commander decides which to archive, which to repair, which to delete. Non-blocking. |
| I-32 | 19 dropped columns | Legacy columns faithfully dropped (brand/token, costing aggregates, invoice TRN/VAT breadcrumbs, supplier payment numbers, customer tax IDs, contact salutations). All dead in PH's own code. `column_drops` computed at runtime by `phimport.Run`. | **Decision per column:** enrich back into new schema where valuable (e.g., supplier payment refs for audit trail), or formally close as "not owed." Re-run `phimport` against latest snapshot to get exact list with non-empty counts. |

### Band LOW — Nice-to-Have (do if velocity allows)

Items identified in recon but not load-bearing for PH operations:

| # | Item | Notes |
|---|---|---|
| I-L1 | Runtime secret-settings persistence | Settings re-entry UX after import |
| I-L2 | PO-PDF supplier-part line | Minor PDF layout detail |
| I-L3 | `availableOfferNumberForCosting` | Offer-number utility |
| I-L4 | Offer-number repair util | Data repair tooling |
| I-L5 | Butler year/overview responses | AI assistant polish |
| I-L6 | Analytics export PDF/Excel/finance-pack/commercial-extract formats | New impl, not a port — scope TBD |

**SKIP (dead in PH's own code):** `primary-contact sync` (`supplier_contact_policy.go`).

### Band 4 — Schema Provisioning Cleanup

| # | Issue | Detail |
|---|---|---|
| I-33 | 15 DIV-GAP tables not in boot migration | 12 bank-recon + `fx_rates` + `fx_revaluations` + `vat_returns` — models already compiled and wired into live services, just not registered in `criticalDeploymentModels()`. One line per table in `deployment_audit.go:122`. | 
| I-34 | 4 ABSENT new models | `extracted_documents`, `costing_sheet_attachments`, `data_quality_reviews`, `employee_archive_requests` — decision-gated. Commander decides which are needed for PH day-one. |

---

## 3. Wave structure (suggested)

Mission I is one mission but likely 2–3 waves of execution:

**Wave 6 (first wave of Mission I):**
- All of Band 0 (money/integrity — small, high-impact, do first)
- Band 4 (schema provisioning — mechanical, de-risks everything else)
- Band 1 items I-08 through I-10 (server-derived totals, VAT, hollow guards — the financial hardening core)
- Band 3 item I-32 (re-run phimport to get exact column_drops ledger — informs later decisions)

**Wave 7:**
- Band 1 items I-11 through I-17 (RBAC sweep, destructive-save, XSS, audits)
- Band 2 items I-18 through I-21 (signature blocks + offer lifecycle — the biggest feature ports)
- Band 2 items I-26 through I-28 (reporting security + PDF fixes)

**Wave 8 (or fold into Wave 7 if velocity allows):**
- Band 2 items I-22 through I-25, I-29 (CRM rollups, aging drill-down, local OCR, costing attachments)
- Band 3 items I-30, I-31 (data quality — needs Commander review per item)
- Band 4 item I-34 (new models — decision-gated)

**After Mission I — separate pass:**
- Frontend component + screen inspection (compare `ph_holdings/frontend/` screens against OSS)
- Commit-history hardening audit (diff PH's `ui-ux-hardening` branch for frontend-specific fixes)
- These are explicitly deferred to AFTER Mission I starts, per Commander's instruction

---

## 4. Measurement

### Entry gate (before Wave 6 starts)
- [ ] `git -C ph_holdings log --since=2026-07-04` confirms zero non-exception commits (Freeze Law holds)
- [ ] `go test ./...` passes on `feat/fable-ph-convergence-w5` (60 packages)
- [ ] `svelte-check` clean
- [ ] Re-run `phimport` against latest PH snapshot → capture `column_drops` with exact per-column non-empty counts

### Exit gate (Mission I complete)
- [ ] Every Band 0 item has a golden test that fails without the fix
- [ ] RBAC audit: every frontend-exposed `App.*` mutator has a `requirePermission` call
- [ ] `PH_PARITY_MAP.md` updated: every closed gap re-classified as PRESENT
- [ ] `phreconcile` still passes 62/62 (no regression)
- [ ] `go test ./...` green; `go vet` clean; `svelte-check` 0 errors
- [ ] Decisions doc updated with new PC-D## entries for any Commander calls
- [ ] 12 wrong-date invoices + 32 offer shells have Commander dispositions

---

## 5. Decision register (items needing Commander input)

| # | Question | When needed |
|---|---|---|
| D-I-1 | Supplier payment gate strictness: require Matched + Approved (PH-strict) or allow partial states? | Before I-05 |
| D-I-2 | ApproveSupplierInvoice: must invoice be Matched before Approved? | Before I-06 |
| D-I-3 | Signature block appearance/layout for OSS offer PDFs. Customer-facing — measure don't guess. | Before I-18 |
| D-I-4 | Which of the 4 absent models are needed for PH day-one? (`extracted_documents`, `costing_sheet_attachments`, `data_quality_reviews`, `employee_archive_requests`) | Before I-34 |
| D-I-5 | Per-column disposition on the 19 dropped columns: enrich back or formally close? | During I-32 |
| D-I-6 | Per-invoice correct dates for the 12 wrong-date invoices (needs source documents). | During I-30 |
| D-I-7 | Per-offer disposition on 32 offer shells: archive, repair, or delete? | During I-31 |

---

## 6. Source references

| Document | Purpose |
|---|---|
| `FABLE_CAMPAIGN_PH_CONVERGENCE.md` | Parent campaign spec |
| `docs/PH_PARITY_MAP.md` | 315-element surface classification (may be stale — re-measure) |
| `docs/PH_CONVERGENCE_PROGRESS.md` | Wave 1–5 execution log |
| `docs/PH_CONVERGENCE_DECISIONS.md` | PC-D1 through PC-D18 |
| `docs/PH_CUTOVER_RUNBOOK.md` | Dress-rehearsed cutover procedure |
| `docs/PH_SOVEREIGN_FORK.md` | Overlay architecture guide |
| PH `SECURITY_FIXES_2026_02_16.md` | Security hardening source |
| PH `docs/ABHIE_MAY_OBSERVATIONS_FIX_PROOF_2026_05_22.md` | SPOC feedback fixes |
| PH `docs/testing/OCR_RBAC_SYNC_AUDIT_2026_04_26.md` | RBAC/sync audit |
| `pkg/data/phimport/phimport.go` | Import pipeline (column_drops logic at :450-462) |
