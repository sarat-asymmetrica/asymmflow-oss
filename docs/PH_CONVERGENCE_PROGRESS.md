# PH Convergence — Wave 1 Progress

Honest self-audit of the campaign's first wave, written by the model
that did the work, against `FABLE_CAMPAIGN_PH_CONVERGENCE.md`. Branch
`feat/fable-ph-convergence-w1`, forked from `main` at the Wave 6 merge
(`24ac8f0`), 2026-07-05.

## What this wave was

Mission A (re-measure the divergence ledger) **and** Mission B (port
the money/integrity-critical divergence) plus the decided DECISION rows
and the opportunistic Band-3 ports. Missions C (data migration), D
(overlay/sovereign fork), and E (parallel-run reconciliation) are
untouched — they are the campaign's later waves.

## Measured timeline (20 commits, one sitting)

| Commit | Row | What |
| --- | --- | --- |
| 765b469 | Mission A | Verified ledger (`PH_CONVERGENCE_LEDGER.md`); Freeze-Law check PASS (zero PH commits since 2026-07-04) |
| 26759bd | Mission F | PC-D1..D4 recorded as the Commander decided them |
| a62c452 | 1-POVAT | PO VATAmount on the BHD subtotal, golden-first; threshold premise corrected (TotalBHD was already right) |
| 1cbe443 | 1-HOLLOW | Hollow-invoice send-block |
| 424a9d3 | 1-FM | Field-mask on 5 update methods; MatchStatus added beyond PH's list (its CHECK constraint proved it wipeable) |
| 36c25bb | E / PC-D1 | Child-count guard inside `pkg/crm/customer` deletes, approval workflow above |
| 7e85930 | A1 | Kernel-pure `pkg/kernel/jsondate` + UnmarshalJSON on the real pkg types (root types are aliases) |
| 30d8d35 | 1-HMAC | Salt-gated startup hash backfill + constant-time VerifyInvoiceHash |
| 523a9b7 | 1-RBAC | 19 unguarded bound mutators guarded; Seed/Migrate internal/exported admin split |
| d305685 | B4+B4a | Server-derived Draft totals + zero-rated recovery + line-item editor UI |
| 9c2617f | F | Credit-limit override via `pkg/approvals` + kernel actors + audit row + override modal |
| 6a263c3 | C / PC-D2 | Invoice PDF parity: division-bank filter, Attention fallback, 40mm margin, ref truncation |
| 3b638c5 | PC-D4 | Banking engine refuses edits on finalized statements (auto-revert deleted; golden deliberately rewritten) |
| 978eb91+405d505 | 1-SYNC / 3-CONN | varchar→TEXT widening, full-seed fetch, NULL-PK backfill (push-only), SKIP_REMOTE_MIGRATION, DSN timeouts + PingContext |
| 014cc5a | D1 | Digit-guard on the loose folder-number fallback |
| c6626da | B3 | Linked-Invoices table on order detail (binding existed, zero callers) |
| 1a3ce2a | D3 | Ordered-passes costing↔opportunity matcher + Disconnect + Start-Fresh confirm |
| 53a96fc | 3-PARSE | ZIP size guard, RTF ParseUint, OCR byte clamps |
| 15f14c7 | 3-PLAT | appDataDirPath consolidation + field-crypto salt exe→AppData cascade |

## Ledger outcomes (every actionable row resolved)

- **PORTED + tested (17 rows/sub-rows):** A1, B3-UI, B4+B4a, C×4, D1-guard,
  D3, E-guard, F, 1-FM, 1-RBAC, 1-HOLLOW, 1-HMAC, 1-POVAT, 1-SYNC×3+env,
  3-CONN, 3-PARSE×3, 3-PLAT (2 live sites + salt cascade).
- **DISSOLVED (verified, no port):** B1 (division inline + startup
  backfill; restart-timing caveat recorded), B2, B3-binding,
  D1/D2-matcher, 3-UI, 2 of the "7 platform bugs" (N/A), 2 more already
  Windows-aware on the ground.
- **DECISION → decided and BUILT:** PC-D1 (compose delete guards),
  PC-D2 (PDF full parity), PC-D3 (port Postgres hardenings), PC-D4
  (refuse-until-reopened).
- **DIFFERENT, verified correct, no port:** 1-FX — OSS's line-item JSON
  carries a populated `currency` end-to-end (`OfferItem.Currency` →
  `formatLineMoney`); PH's `source_currency` bug shape doesn't exist here.
- **Deferred by design:** Band-2 engine ports (product-supplier
  linkage, valuation fallback/weighted-average) — real capability gaps,
  engine-sized, next wave; Band-2 seed catalogues → overlay config
  (Mission D territory); identity-write consolidation into a
  `pkg/crm/customer` WritePolicy — recorded, not yet lifted.

## Honest notes

- **Corrected premises:** 1-POVAT's "corrupts the 5K threshold" did NOT
  hold in OSS (TotalBHD derivation was already arithmetically right;
  only the stored VATAmount was wrong). 3-PLAT's "StandardOverlayDirs
  probably solved most of these" did NOT hold either way — two sites
  (diagnostics, DB resolution) were already Windows-aware, three were
  genuinely blind.
- **Deliberate golden changes**, stated in their commits: PC-D4 rewrote
  the auto-revert golden to refuse→reopen→edit; B4's Draft rebuild also
  derives `outstanding_bhd` (backend contract; avoids a blocking
  validation PH ships with).
- **Deviations from PH, justified in place:** MatchStatus joined the
  1-FM mask (CHECK-constraint proof); NULL-PK backfill is push-only
  (the randomblob SQL is SQLite-specific); credit-override authority
  rides a dedicated `creditOverrideActor` so the delete-approval actor
  policy is untouched.
- One in-tx `logAudit` deadlocked SQLite during Row F; the audit row
  now writes after commit — which is also more truthful.
- `wailsjs` binding stubs for `CreateInvoiceWithCreditOverride` were
  hand-added mid-wave and regenerated verbatim by the wave-end build.

## Definition of done (this wave)

- Full `go test ./...` green; `go run ./cmd/hospitality` exit 0;
  `wails build -clean` succeeds; svelte-check 0 errors; bindings chore
  commit at wave end.
- Branch merged only by the Commander.

## Residue for the next wave

1. Band-2 engines: product-supplier resolution + valuation fallback /
   weighted-average costing (the two genuine capability gaps).
2. Identity-write consolidation (bidirectional Code↔CustomerID fill,
   update-path blank-refill, supplier Rating fallback) into
   `pkg/crm/customer`.
3. B1's restart-timing caveat: optional inline item copy at won-import.
4. Mission C (importer; mind the PRAGMA foreign_keys promote bug),
   Mission D (PH overlay.json + sovereign fork config — where the seed
   catalogues land), Mission E (parallel-run reconciliation, gated).

---

# PH Convergence — Wave 2 Progress

Branch `feat/fable-ph-convergence-w2`, forked from `main` at the Wave 1
merge (`4e7a3f7`), 2026-07-06. Freeze-Law re-check at wave start: PASS
(still zero `ph_holdings` commits since 2026-07-04).

## What this wave was

The Wave 1 residue, executed: the Band-2 engine ports (PC-D5 records
the seam ratification), identity-write consolidation, B1's inline item
copy, and Mission C's importer. Missions D (overlay/sovereign fork) and
E (parallel-run, Commander-gated) remain.

## Measured timeline

| Commit | Row | What |
| --- | --- | --- |
| f105231 | B1 | Inline invoice-item population at won-import (shared `repairInvoiceItemsFromOrder`; closes the restart-timing caveat) |
| e4ae969 | Band-2 (1) | `pkg/crm/customer/write.go`: bidirectional Code↔ID fill, G1 field-mask merges, Rating fallback, UUID identifier repair |
| fee2be8 | Band-2 (15-16) | `pkg/crm/supplierlink`: four-tier resolver + token search; seed/PO-inference/alert wiring; placeholder FK dead |
| 0affe99 | Band-2 (17-18) | `pkg/inventory`: reference-cost chain + weighted average; GRN receipts now create valued stock; discrepancy costing real |
| 4c83457 | Mission C | `cmd/phimport` + `pkg/data/phimport`: pinned-connection PRAGMA discipline, intersected columns, transform + honest report; latent tables join tradingModels() |

## Honest notes

- **Ground bugs found by porting, fixed in passing (each stated in its
  commit):** OSS UpdateCustomer's full `Save` was live-wiping 11
  server-owned metric columns on partial payloads; inventory, comment,
  post-sale and db_costing tables were defined but in NO migration set;
  stock-movement numbering used MySQL `YEAR()` (matches nothing on
  SQLite); `GetInventoryValuation(warehouseID *uint)` could never match
  the string warehouse IDs the model stores; `RaiseGRNDiscrepancy`
  shipped a literal `rejectedQty*100.0` placeholder into
  `SupplierIssue.CostBHD`.
- **Deliberate deviations from PH, justified in place:** seed products
  stay unlinked on resolution failure instead of being skipped (OSS
  supplier seeding is user-invoked; an empty link is honest, a dangling
  one is not); supplier Rating 0 falls back on update where PH HEAD
  copies unconditionally; the supplierlink token expansion also tries a
  multi-word token's leading words (brand-led product names resolve
  without per-name alias entries); PO-inference free-text fallback
  deliberately does NOT mine descriptions.
- **Deliberate schema-golden changes** (regenerated + reviewed in their
  commits): StockMovement provenance columns; Warehouse/InventoryItem/
  StockMovement/StockAdjustment and the latent comment/costing tables
  join `tradingModels()`.
- **Bound-surface change:** `GetInventoryValuation` now takes `*string`
  (PH parity); no live frontend caller existed, bindings regenerate at
  wave close.
- **Open decision surfaced, not taken:** `customer_receipts` transform
  (into `payments`) vs drop — the importer marks it PENDING DECISION
  with row counts; Commander call before any Mission E rehearsal.

## Definition of done (this wave)

- Full `go test ./...` green; `go run ./cmd/hospitality` exit 0;
  `wails build -clean` succeeds; svelte-check 0 errors; bindings chore
  at wave end. Branch merged only by the Commander.

## Residue for the next wave

1. Mission D: PH `overlay.json` + sovereign-fork config — supplier
   alias catalogue and seed enrichment land there as config
   (`supplierLinkAliases()` is the injection point).
2. `customer_receipts` decision, then a Mission E rehearsal on a COPY
   of PH data using `cmd/phimport` (Commander-gated).
3. Band-2 consumers not yet rewired: supplier-invoice 3-way match could
   use the reference-cost resolvers (`SupplierInvoiceItemUnitPriceBHD`
   is built and tested, unwired); OUT-side movements from
   delivery/dispatch still don't exist anywhere in OSS.
4. `FABLE_CAMPAIGN_SOVEREIGN_MESH.md` remains untracked in the repo
   root, unread by this campaign.

---

# Wave 3 Progress (2026-07-06)

Wave 2 merged to main (`ff2cc00`) with Commander authorization at wave
start. Wave 3 on `feat/fable-ph-convergence-w3`.

## Timeline (measured)

| Step | Outcome |
|---|---|
| PC-D7: customer_receipts → payments | Commander decided **transform**. Measured first: PH creates a payments row per receipt allocation and `payments` is already copied, so only the UNAPPLIED on-account remainder is carried — one invoice-less payment per live receipt (deterministic id/idempotency key, method normalised into the check constraint, receipt number + customer id in `reference`). Reversed/soft-deleted carry nothing; the report accounts for every receipt. Allocations skip as "represented". Commit `1abe4b8`. |
| Mission D survey | Full-repo measurement: the overlay seam already existed and was LIVE (divisions, VAT, FX, business rules, markups, jurisdiction, seed gating). Mission D re-scoped from "build overlay" to "close the residue" (PC-D8). |
| Mission D closure | Supplier alias vocabulary + license key prefix → overlay config (byte-identical defaults); banking division normaliser consolidated (deleting legacy real-company alias spellings from engine code); VAT default chain unified (setting → overlay → 10); VAT-CSV label overlay-driven (byte-identical at default); letterhead asset key from division profile. `docs/PH_SOVEREIGN_FORK.md` = the "company is a JSON file" guide + flag register. Commit `ce2ff1a`. |

## Honest notes

- **Behavior deviation (documented in PC-D8):** DB rows carrying the
  legacy "a h s trading …" division spellings now normalise to the
  default division unless the deployment declares them as overlay
  aliases. No such rows exist in the synthetic canon; the sovereign
  overlay declares them.
- **Appearance-adjacent change, bytes unchanged:** the VAT-return CSV
  label is now rendered from the overlay rate. At the built-in 10% the
  output is byte-identical; done only under that condition.
- **Not moved, flagged** (see PH_SOVEREIGN_FORK.md flag register):
  DeliveryTerms GORM default, demo bank/product/supplier seed rows
  (seed_sets gate is the config seam), legacy importer alias maps.

## Residue for the next wave

1. Mission E rehearsal on a COPY of PH data via `cmd/phimport`
   (Commander-gated) — PC-D7 unblocks it.
2. Band-2 consumers still unwired: supplier-invoice 3-way match vs the
   reference-cost resolvers; OUT-side stock movements from
   delivery/dispatch don't exist anywhere in OSS.
3. Flag-register items if/when their own golden-first passes are
   ordered (DeliveryTerms DDL default is the first candidate).
4. `FABLE_CAMPAIGN_SOVEREIGN_MESH.md` remains untracked/unread.

## Mission E rehearsal (2026-07-07) — RECONCILED

Commander-authorized rehearsal, executed entirely OUTSIDE this repo
(`C:\Projects\asymmflow\mission_e_rehearsal\`, PH snapshot of 2026-06-29
copied; no real data entered the repository). Full path walked as
documented in PH_SOVEREIGN_FORK.md: provision fresh DB (overlay +
`seed_sets: ["default-assets"]`) → phimport → boot on imported file.

- **Result: 25/25 reconciliation checks match to the fils.** Invoices
  480 (subtotal/VAT/grand/outstanding, per-status, per-year), payments
  93, supplier invoices 484 + payments 601, invoice/order items,
  offers, users; referential integrity of replaced tables intact; app
  boots with zero errors and recomputes 480/480 invoice hashes.
- Import: 107 tables / 17,595 rows; every skip/unmapped counted.
  `customer_receipts` empty in this snapshot → PC-D7 transform verified
  as a no-op (0 carried, correctly reported).
- Ground fixed in passing (PC-D9): fresh-provision gap
  (`chart_of_accounts`/`account_mappings` → criticalDeploymentModels),
  phimport `replaceTables` (seeded-baseline collisions; source ids win
  on a fresh destination) and `columnTransforms` (PO status 'Completed'
  → 'Closed', PH's own read-side normalisation).
- New findings for the shelf: banking-suite tables not provisioned on
  fresh files (PH has 2 statements / 12 lines / 2 accounts there);
  `extracted_documents` (359 rows) UNMAPPED — decide carry/skip;
  `intelligence_*` enrichment + `*_backup` tables UNMAPPED (probable
  skip-with-reason candidates); company_bank_accounts seed uses
  real-bank id slugs (`bank-ahli`, `bank-bbk`) — synthetic-canon nit.
- **Cutover remains Commander-gated.** The rehearsal proves the path on
  a June-29 snapshot; a cutover would repeat it on a fresh copy taken
  at switch time, then parallel-run.

---

# Wave 4 Progress (2026-07-07) — Mission G, "Prove the Parity, Certify the Cutover"

Branch `feat/fable-ph-convergence-w4`, forked from `main` @ `801f41a` (handoff
commit `84e7c70` on top). Freeze-Law re-check at wave start: **PASS** — zero
`ph_holdings` commits since 2026-07-04 (HEAD `ca24372`).

## What this wave was
Not a re-port of the ledger (Waves 1–3 shipped essentially all of it). Mission G
mapped the **unmeasured** surface — every screen, bound flow, report, and
startup job — classified all 315 elements, closed the money/integrity/security/
provisioning gaps, and emitted a cutover-readiness certificate.

## Deliverables
- `docs/PH_PARITY_MAP.md` — 315 elements classified across 9 domains (nine
  parallel read-only passes): **222 PRESENT · 16 DIVERGENT-INTENTIONAL · 36
  DIVERGENT-GAP · 34 ABSENT · 7 EXTRA**.
- `docs/PH_PARITY_CERTIFICATE.md` — the cutover artifact: verdict, coverage,
  the closed-this-wave table, the DIVERGENT-INTENTIONAL register, and the
  residual stop-and-ask / Mission-H list.
- PC-D10…D15 recorded in `docs/PH_CONVERGENCE_DECISIONS.md`.

## Measured timeline (closes, each tested unless noted)
| Row | What |
|---|---|
| G.1 | PH_PARITY_MAP full-surface classification (9-agent measurement) |
| Provisioning | Bank-recon+FX+VAT 15-table suite provisioned create-if-missing (FK-safe); latent live-startup regression caught + averted |
| Procurement | AP pay/approve gate tightened (Commander Q1) + latent `Verified` CHECK-constraint fresh-DB bug fixed; schema golden regenerated |
| Finance | Settlement policy ported: Draft non-payable + Overdue read-hydration |
| Procurement/Inv | 3-way match wired to reference-cost resolvers (0-priced PO line no longer escapes) |
| Reporting | Export RBAC + filename sanitize + payload cap + runway N/A guard |
| Reporting | Invoice-PDF document-context bank fetch (payable invoice for invoices:view); no PDF bytes changed |
| Reporting | PO-PDF DRAFT/PENDING watermark (Commander Q2) |
| Finance | `GenerateInvoiceNumber` RBAC guard restored |
| Documents | Inbox local-OCR fallback (offline-first restored) |
| Documents/Opp | D1/D2 list-level digit-guard + cleanLoose/splitLoose helpers |
| Opportunities | `DeleteOffer` + `DeleteOpportunity` ported with integrity guards |
| Housekeeping | 6 real-bank demo slugs → synthetic (invariant #2) |

## Honest notes
- **Zero financial-number divergences** were found on any core document — the
  strongest single parity signal; the audit paid for itself instead by finding a
  fresh-DB integrity bug (matched supplier invoices couldn't persist) and a
  would-be startup regression (unconditional banking-suite migrate on the live DB).
- **A self-reported claim was corrected mid-wave:** `DeleteOffer`/`DeleteOpportunity`
  were first called "dead frontend buttons"; a corrected grep shows no frontend
  caller (the match hit `DeleteOfferNote`/`DeleteOpportunityComment`). They were
  ABSENT backend features, now present. Recorded in the certificate.
- **Party-delete (handoff §3.2) reclassified PRESENT:** the pre-ratified
  orphaning divergence was stale — Wave 1 (PC-D1) already closed it (PC-D14).
- Customer-facing PDF signature-block appearance measured, not changed (Commander
  Q2: keep OSS + register).

## Honest thesis %
The convergence thesis — *"a vertical is configuration plus a thin domain
package"* — stands at **~96%** after this wave (was ~93% at Wave 4 handoff). The
substrate now demonstrably carries PH's full feature-and-flow surface with numbers
intact and the fresh-provision schema complete; the remaining ~4% is Mission H
(data convergence) plus a short, Commander-ruled set of appearance/no-caller items
— none of which is a parity risk.

## Residue for Mission H (data convergence) and beyond
1. **Data** into the 15 newly-provisioned banking/FX/VAT tables (provisioning done).
2. `extracted_documents` carry/skip (model-less raw table; confirm live data first).
3. OneDrive discovery-walk + disambiguation cluster (~18 fns) — live re-scan (PC-D15).
4. Customer-facing appearance: signature-block library (overlay-driven) + costing
   datasheet PDF bundle — Commander-gated.
5. No-caller feature methods (offer revision/renewal, opportunity/CRM getters,
   aging drill-down) — port when their UI is wired.
6. Smaller residues: D2 canonical-key route-dedup; `GetReportData` RBAC; runtime
   secret-settings persistence.

## Definition of done (this wave)
- `docs/PH_PARITY_MAP.md`, `docs/PH_PARITY_CERTIFICATE.md` complete.
- Every money/integrity/security/provisioning DIVERGENT-GAP closed golden-first
  with tests, or recorded with a reason; known tail (G.2) closed/verified.
- `go test ./...` green · `go vet ./...` clean · schema golden regenerated.
  (`wails build -clean` is Commander-run at packaging; bindings for the new
  `DeleteOffer`/`DeleteOpportunity`/`CreateInvoiceWith…` surfaces regenerate then.)
- Branch merged only by the Commander.

---

# Wave 5 Progress (2026-07-07) — Mission H, "Carry Everything, Reconcile Everything"

Branch `feat/fable-ph-convergence-w5`, forked from `main` @ `a0a9d18` (Wave 4
merge + W5 handoff). Freeze-Law re-check at wave start: **PASS** — zero
`ph_holdings` commits since 2026-07-04 (HEAD `ca24372`).

## What this wave was
Mission H: bring `cmd/phimport` from Mission-E coverage to the COMPLETE PH
table surface against the Mission-G-completed schema, build the reconciliation
gate as re-runnable code, dress-rehearse the full pipeline from zero, and
write the cutover runbook. Data carried as-is (faithful first); quality is
Mission I.

## Measured results (full-provision rehearsal, out-of-repo, fresh snapshot copy)

| Step | Measured |
|---|---|
| Fresh provision (sovereign overlay, `["default-assets"]`) | 134 tables from zero; banking/FX/VAT suite + `fiscal_periods` + `customer_name_mappings` present; **0** demo bank fixtures (PC-D18 fix) |
| Import | **124 tables / 17,611 rows carried; 0 UNMAPPED; 21 skips, all reasoned+counted**; 19 dropped legacy columns reported with non-empty counts (`column_drops`) |
| Banking carry | bank_accounts 2 · bank_statements 2 · bank_statement_lines 12 (rest of the 15-table suite honestly 0 in PH source) |
| Reconcile (pre-boot) | **62/62 checks to the fils** (Mission E core re-covered + banking/FX/VAT extension; invoices 480 live / 8,988,247.145 BHD grand) |
| First boot | zero errors; **480/480** live invoice hashes recomputed under the new salt |
| Reconcile (post-boot) | **62/62** — booting mutates nothing carried |

## What the wave found and fixed (ground, each tested)
- **Two more Mission-G-class provisioning gaps**: `fiscal_periods` and
  `customer_name_mappings` — models read by live services, in no boot
  migration set → registered in criticalDeploymentModels.
- **PC-D17 — credit-note VAT column drift**: PH `vat_bhd` vs OSS `vatbhd`;
  the intersect copy silently dropped credit-note VAT. Fixed via explicit
  `columnRenames`; a full drift scan of every carried table found no other
  live money drift (all other drifted-and-populated columns are dead in PH's
  own code → dropped faithfully, reported per-column).
- **PC-D18 — demo bank fixtures seeded on every boot** (caught by the NEW
  post-boot reconcile step): `seedCompanyBankAccountsInternal` ignored the
  `demo-bank` seed set from four call paths, injecting 5 synthetic-IBAN
  accounts on a sovereign install's first boot. Gate enforced at the seam,
  pinned by test.
- **Report honesty**: every skip path now counts source rows (a "no
  destination table" skip previously reported 0 rows for populated tables).
- **PC-D16 — `extracted_documents` (359)**: measured (flat scan metadata,
  no FKs, read by nothing in PH) → skip-with-reason; `*_backup`,
  `intelligence_*`, `sqlite_stat1`, `collaborative_pending_operations`
  adjudicated the same session.

## Deliverables
- `pkg/data/phreconcile` + `cmd/phreconcile` — the 62-check gate as code
  (carry counts, money to the fils, banking/FX/VAT, PC-D7-aware payments,
  key-list settings check tolerating only the app's own backup bookkeeping).
- `docs/PH_CUTOVER_RUNBOOK.md` — freeze → snapshot → provision → import →
  reconcile → boot → POST-BOOT reconcile → re-key → parallel-run → switch,
  with acceptance gates and rollback; every step is the rehearsed step.
- PC-D16/D17/D18 in `PH_CONVERGENCE_DECISIONS.md`.

## Honest residue (→ Mission I and cutover day)
- The `column_drops` ledger (19 columns, e.g. order_items brand/token,
  costing extraction aggregates, invoice TRN/VAT note breadcrumbs,
  supplier payment numbers, 5 tax_code-only customer tax IDs) is data PH's
  own app cannot see either; the archived snapshot preserves it and
  Mission I decides enrichment.
- PH source reality carried faithfully: 12 wrong-date (2026+) invoices, 32
  legacy offer shells (3 priced with no items) — flagged by the startup
  audit, deferred to Mission I with the darch/ph_data_master backfill.
- `costing_sheet_attachments` PENDING DECISION stands (0 rows in source;
  moot until PH attaches datasheets).
- OneDrive re-scan cluster (PC-D15) remains an at-cutover/Mission-I call.
- The rehearsal used the June-29 snapshot; cutover repeats the runbook on a
  freeze-day copy.

## Definition-of-done check
- Full-surface import: every source table carry/transform/skip-with-reason,
  0 unmapped ✅ · 15 banking/FX/VAT tables populated and reconciled ✅ (thin
  but real: 2/2/12) · full core check set re-passes, N=62>25 ✅ · rehearsal
  from zero green ✅ · runbook written ✅ · decisions/progress mirrored ✅ ·
  `go test ./...` green, `go vet` clean (pre-existing butler dead-code
  removed), svelte-check 0 ✅ · merge is Commander-gated.

---

# Mission I Wave 6 — "Harden the Substrate, Enrich the Carry" (2026-07-09)

Branch: `feat/fable-mission-i-w6`. Spec: `FABLE_CAMPAIGN_MISSION_I.md`.
Scope shipped: ALL of Band 0, Band 4 verify, Band 1 I-08/I-09/I-10, Band 3
I-32, AND the Wave-7 audit core (I-11..I-17) pulled forward.

## Stale-map findings (measured on the ground, per invariant 6)
Already closed before this wave started — parity map entries were stale:
- I-03 (GenerateInvoiceNumber RBAC), I-05/I-06 (supplier pay/approve gates,
  Commander-ratified in Mission G with `supplier_ap_gate_test.go`), I-07
  (3-way-match reference costs), I-09 (PO VAT BHD basis + zero-rated
  recovery, golden tests exist), I-10 (MON-007 hollow guard +
  `repairInvoiceItemsFromOrder`), I-33 (all 15 DIV-GAP tables + 2 more in
  `criticalDeploymentModels()` since Mission G/H).

## Shipped this wave
- **I-01/I-02/I-04** (`payment_service.go`, `customer_invoice_service.go`):
  every settlement mutation and 5 read paths now route through
  `customer_invoice_payment_policy`; MarkCustomerInvoicePaid now creates a
  Payment audit row (was zeroing balances recordlessly); MarkOverdue derives
  instead of asserting. Golden tests: `mission_i_band0_test.go`.
- **I-08 guards**: UpdateCustomerInvoice refuses client edits to Outstanding
  (open invoices) and hand-set settlement statuses; pkg payment-delete now
  derives status with policy math (credit-note-aware).
- **I-11**: 770 bound mutators audited; 17 unguarded fixed — headline:
  SeedLicenseKeys/SeedEmployeeKeys could mint auth credentials from the
  frontend ungated. Guarded-public/internal split for all startup callers.
- **I-12**: 52 update handlers audited; UpdateOrder no longer wipes
  QuantityShipped/QuantityInvoiced on item edits (LATENT IN DEPLOYED PH);
  costing approval mass-assign closed; unfiltered map handlers whitelisted
  (accounts, inventory, bank accounts, bank statements); GRN/supplier-invoice
  Save() wipe-by-omission closed; payment GL links + DN signatures masked.
- **I-13**: 27 unescaped DataTable render interpolations across 10 screens
  escaped (PH RES-001 class); `safeStr()` trap fixed.
- **I-16**: 5 swallowed write errors fixed (payroll item approve, expense
  approval audit row, bank-line link, graph saves, prediction create).
- **I-14/I-15/I-17**: verified present (opportunity child-guard absent in
  BOTH trees — parity, OSS stronger via delete-approval workflow).
- **I-32**: phimport re-run vs 2026-06-29 snapshot → 19-column drops ledger
  with exact non-empty counts captured (`mission_i_data/import_report.json`,
  out of repo) for the D-I-5 disposition review.

## Gates
Entry: Freeze Law zero commits ✅ · full suite green ✅ · svelte-check 0 ✅ ·
phimport re-run ✅. Exit: golden tests fail-without-fix ✅ · full
`go test ./...` green (81 pkgs) ✅ · `phreconcile` 62/62 ✅ · svelte-check 0 ✅.

## Remaining Mission I (next wave)
Band 2 ports: I-18 signature blocks (D-I-3 STOP-AND-ASK pending), I-19..I-25,
I-26..I-28 reporting fixes, I-29; Band 3 I-30/I-31 Commander dispositions;
Band 4 I-34 decision; Band LOW.

---

# Mission I — Wave 7 (2026-07-09, same session)

Band 2 ports, four parallel port agents, per-port review + commit.

## Shipped this wave
- **I-18** (`offer_signature_blocks.go`, `pkg/overlay`): PH signature-block
  renderer ported byte-for-byte (D-I-3 "replicate PH layout exactly").
  Signer identities are overlay configuration (`signature_blocks` +
  `signature_default`), alias-aware matching, synthetic-canon builtin
  defaults with a leakage-guard test. Wired at the REAL call sites:
  `exportCostingToPDF` (shared offer/costing pipeline — where PH draws it),
  credit-note PDF (replaces static "Authorized Signatory"), offer PreparedBy
  canonicalisation. End-to-end pdftotext assertions.
- **I-19..I-21** (`app_sales_pipeline.go`, `pkg/crm/domain.go`):
  CreateOfferRevision + RenewOffer ported with lineage columns
  (revision_of/revision_root/superseded_by/superseded_at — schema golden
  regenerated deliberately); UpdateOpportunityCommercialFields with
  27-column whitelist (PH gate parity); GetUnifiedOfferThread;
  GetCostingsByOpportunity (+ `opportunity_id` on costing_sheet_data).
  DeleteOffer guards verified byte-identical to PH. 11 golden tests.
- **I-23/I-26/I-27/I-28** (`finance_reporting_service.go`, `reports.go`,
  `purchase_order_pdf_service.go`): aging-bucket drill-through with
  settlement-policy-derived state (aggregate and drill-through share one
  helper, cannot drift); RBAC gates on GetReportData (per-type) + 3 bound
  PDF generators; diagonal DRAFT watermark on unapproved PO PDFs; invoice
  bank block verified already profile-sourced (locked with a test). 11 tests.
- **I-22** (`app_crm_surface.go`): GetCustomerRelatedProducts/Suppliers
  ported, OSS-adapted (no Brand×Token taxonomy; supplier via ProductMaster).
  3 golden tests. **I-24** (local OCR fallback) and **I-29** (OneDrive
  folder-number helpers) measured ALREADY PRESENT and wired — no port.
- Deflaked `TestInteractiveSession_ActivityPersistsThrottled` (timestamp
  granularity under full-suite load; backdated-row hardening).

## Gates
`go build ./...` clean · full `go test ./...` — one deliberate schema-golden
regen (offer lineage) + one pre-existing flake fixed; suite green on re-run ·
per-agent synthetic-canon leakage scan clean.

## Remaining Mission I
I-25 costing attachments (ties to D-I-4 `costing_sheet_attachments`);
Band 3 I-30 (12 wrong-date invoices, D-I-6) + I-31 (32 offer shells, D-I-7)
— review pack being generated to `mission_i_data/band3_review_pack.md`;
Band 4 I-34 absent-models (D-I-4); D-I-5 per-column dispositions; Band LOW.

## Wave 7 continuation — decision-gated closure (same day)
All Commander decisions landed and executed same-session:
- **PC-D20 (D-I-4)**: `costing_sheet_attachments` ported NOW → **I-25
  shipped** (attachment service + pdfcpu datasheet bundling, PH-parity
  RBAC, 12 golden tests, CGO_ENABLED=0 verified). The other 3 models
  deferred-with-spec: `docs/MISSION_I_DEFERRED_MODEL_SPECS.md` (headline:
  employee_archive_requests is a LIVE OSS behavioral gap — port first;
  extracted_documents should be closed, it has no Go model even in PH).
- **PC-D21 (D-I-6/D-I-7)**: ground measurement overturned the spec — the
  "12 wrong-date invoices" are live-FY2026 drafts with a placeholder date,
  NOT year typos. Anchored 10 to provenance, 2 flagged for SPOC; 35 shells
  triaged 20 delete / 12 archive / 3 repair-to-the-fils; SA-05-SULB
  flagged-unrepaired (no reconcilable in-system value — refused to
  fabricate). Follow-through code fix: startup audit now flags
  future-dated-vs-today instead of the stale `year >= 2026` cutoff.
- **PC-D22 (D-I-5)**: ALL 19 dropped import columns enriched back (all
  schema-gaps, not renames — verified). phreconcile gate grew 62 → 81
  checks. Fresh enriched import + Band-3 re-apply on the shadow DB:
  **79/81 with exactly the 2 deliberate Band-3 deltas, 0 unexpected;
  column_drops = 0; integrity ok.**

## Mission I standing state
Bands 0/1/2/4-decided complete; Band 3 executed on the shadow DB with a
full audit trail (`mission_i_data/band3_*`). Open residue for later waves:
2 SPOC invoice dates (PH26-035/037), SA-05-SULB manual repair,
employee_archive_requests + data_quality_reviews ports (specs written),
extracted_documents formal closure, Band LOW items, post-Mission-I
frontend pass (CustomerDetailView rollup binding, DataQualityScreen).
