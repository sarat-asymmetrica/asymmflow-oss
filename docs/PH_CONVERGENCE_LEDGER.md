# PH Convergence — Divergence Ledger (verified)

Mission A deliverable of the PH Convergence campaign
(`FABLE_CAMPAIGN_PH_CONVERGENCE.md`). Every `[seed]`/VERIFY/DECISION row
from the campaign spec re-measured on the ground, 2026-07-05, by six
parallel read-only measurement passes over both repos. PH short SHAs are
in the `ph_holdings` repo; file:line citations are current as of OSS
`main` post-Wave-6 merge (`24ac8f0`).

**Freeze-Law check (spec §2): PASS.** `ph_holdings` has zero commits
since 2026-07-04; its last commit is `ca24372` (2026-06-29), already row
F. The spec's ledger snapshot is current; no exception-class commits to
fold in.

Verdict vocabulary: **DISSOLVED** (substrate already covers it, possibly
via a different mechanism) · **PORT** (absent; clean port needed) ·
**PARTIAL** (some paths covered — stated which) · **DIFFERENT** (OSS
chose another path) · **DECISION** (Commander's call; facts laid out).

---

## Band 0 — June-29 SPOC fixes (top priority)

| # | Row | Seed verdict | **Verified verdict** |
|---|---|---|---|
| A1 | Flexible date UnmarshalJSON | PORT | **PORT** — confirmed |
| B1 | Won-import line-items + division | VERIFY | **DISSOLVED** (different mechanism; timing caveat) |
| B2 | Hollow-invoice toolkit | DISSOLVED | **DISSOLVED** (pre-verified) |
| B3 | Linked-Invoices table | PORT (UI) | **binding DISSOLVED · UI PORT** |
| B4 | Draft editor + server-derived totals | PORT | **PORT** (backend + UI) |
| B4a | Zero-rated VAT on update | VERIFY→PORT | **folds into B4** (must ride the same recompute) |
| C | Invoice PDF (gated) | PORT (gated) | **4× PORT + 1 equivalent — stop-and-ask** |
| D1/D2 | Opportunity routing | VERIFY→PORT | **matcher DISSOLVED · digit-guard PORT** |
| D3 | Costing↔opportunity flows | PORT (UI) | **PARTIAL → PORT (UI)** |
| E | Party-delete semantics | DECISION | **DECISION** — OSS orphans children today |
| F | Credit-limit override | PORT (via kernel) | **PORT via `pkg/approvals`** — confirmed |

### A1 — flexible date parsing — PORT
PH `425c0b2` adds `parseFlexibleJSONDate` + custom `UnmarshalJSON` on
`Order` (order_date/required_date) and `ExpenseEntry`
(expense_date/due_date): accepts date-only `2006-01-02` AND RFC3339;
empty stays zero so GORM skips the column. OSS has no model
`UnmarshalJSON` anywhere; `parseFlexibleDate`
(`bank_statement_parser.go:1772`) is bank-statement-only. Rejecting
endpoints today: `UpdateOrder` (`app_order_customer_surface.go:351`),
`CreateExpenseEntry` (`expense_service.go:262`,
`service_finance.go:601`). `CreateOrder` is safe (takes a string and
parses manually). **Port home matters:** root types are aliases
(`type Order = crm.Order`, `database.go:86`), so the methods must land
on `crm.Order` (`pkg/crm/domain.go:348`) and `finance.ExpenseEntry`
(`pkg/finance/domain.go:360`). Empty-stays-zero is load-bearing.

### B1 — won-import line-items + division — DISSOLVED
Division stamp already inline: `onedrive_import_service.go:1969` sets
`invoice.Division = normalizeDivisionName(order.Division)` in the same
`importSingleDeal` position as PH `3c5127b`. Item population is covered
by a different mechanism: `BackfillInvoiceItemsFromOrders`
(`customer_invoice_service.go:161`), run at every startup
(`app.go:992-997`), test-covered. **Caveat:** a freshly won-imported
invoice stays item-less until the next restart (boot-time batch vs PH's
inline copy). Optional enhancement, not a gap: inline item copy after
`onedrive_import_service.go:2000`.

### B3 — Linked Invoices — binding DISSOLVED, UI PORT
`GetInvoicesByOrder` exists bound twice (`invoice_traceability.go:222`
with `invoices:view` guard; `service_finance.go:823`), generated into
wailsjs — OSS is ahead of PH's pre-fix state. Frontend callers: zero;
no order-detail screen renders linked invoices. Port = the `a1aae97`
Svelte block only (lazy-load on row click, table with
StatusBadge/formatters).

### B4 + B4a — server-derived totals on Draft edit — PORT (backend + UI)
OSS `UpdateCustomerInvoice` (`customer_invoice_service.go:1092-1116`)
deletes/recreates items on Draft but **never recomputes header totals**
— the comment at `:1073` calls GrandTotal "computed / not editable" but
nothing recomputes it. Stale-totals bug live in OSS. The UI is worse
than read-only: the Edit modal (`InvoicesScreen.svelte:1196-1207`) has
a writable Amount input bound to `grand_total_bhd` that the server then
silently ignores. **B4a folds in:** OSS has the effective-rate recovery
at CREATE (`customer_invoice_service.go:815-822`) but the UPDATE
recompute doesn't exist at all — a naive B4 port without PH `70b05d2`'s
VATBHD/Subtotal recovery would reintroduce the zero-rated→10% reset
bug. Port the recompute block WITH the rate recovery and carry PH's
`TestUpdateInvoicePreservesZeroRatedVAT`.

### C — invoice PDF — 4× PORT + 1 equivalent, STOP-AND-ASK GATED
Measured only; no edits (customer-facing financial document). Against
OSS `invoice_pdf_service.go`:
- Division-bank filter: **PORT** — `:637-643` loops ALL bank accounts
  unfiltered; a PH invoice can list AHS banks.
- Buyer-address Attention* fallback: **PORT** — `:321-337` prints
  customer master address unconditionally.
- "For LegalName" signature: **equivalent** — `:609-616` already prints
  `For <LegalName>` + "Authorized Signatory"; layout differs from PH's
  blank wet-signature space (appearance choice to confirm, not defect).
- 40mm top margin: **PORT** (cosmetic) — OSS at 50mm (`:131`, `:145`).
- Ref-number width truncation: **PORT** — `:235` plain Cell; helper
  absent.

### D1/D2 — opportunity routing — matcher DISSOLVED, digit-guard PORT
Canonical-key matcher `rankExistingOpportunityForImport` present
(`onedrive_import_service.go:811`, same scoring shape as PH) and
`routeToOpportunity` present (`app_setup_documents_surface.go:2799`) —
the seed carried them. `folderNumberHasDigit` +
`cleanLooseOneDriveFolderNumberToken` guard: **absent** (zero code
hits) — customer-word→folder-number collapse is unguarded in OSS.
Port the guard + PH's `opportunity_collapse_regression_test.go` cases.

### D3 — costing↔opportunity flows — PARTIAL → PORT (UI)
OSS `CostingSheetScreen.svelte` has the baseline opportunity→costing
pre-fill (`:44`, `:877-898`, `:905-926`). Missing all three PH
hardening flows: ordered-passes matcher
`findOpportunityFromPendingLaunch` (exact id → exact ref →
same-customer last), `disconnectFromOpportunity`, and the Start-Fresh
confirm modal (PH `505ae76` + `ca24372`). Frontend-only port.

### E — party-delete semantics — DECISION (live integrity gap in OSS)
The load-bearing difference is the **child-count check**, which OSS
lacks at every layer:
- OSS wrappers (`app_order_customer_surface.go:1796-1827`) do
  `guardDeleteOrRequest` + permission, then delegate to
  `pkg/crm/customer/delete.go:17-54`, which verifies existence only.
- `guardDeleteOrRequest` (`delete_approval_service.go:63-76`) answers
  only "who may delete"; `performApprovedDelete` (`:210-276`) re-enters
  the same path. **No layer counts children.**
- PH (`app.go:12170-12251`) composes BOTH: the same
  guardDeleteOrRequest primitive PLUS a child-count guard returning
  `CUSTOMER_HAS_LINKED_RECORDS` with exact counts (Order/Invoice/Offer/
  Opportunity; supplier mirror counts PO/SupplierInvoice/
  SupplierPayment).

**Today an admin or approved delete in OSS orphans a party's
transactional children; PH blocks it.** Recommended composition: add
the child-count check inside `pkg/crm/customer/delete.go` (protects
every caller), typed `*_HAS_LINKED_RECORDS` error with counts; keep the
approval workflow layered above. A blind port would NOT regress OSS —
the spec's fear inverted — but composition is still the right shape.
**Commander decides** (stop-and-ask registry).

### F — credit-limit override — PORT via kernel
OSS confirmed hard-block-only: `CreateInvoiceWithOptions` is 3-param
(`customer_invoice_service.go:488`), over-limit returns an error inside
the atomic tx (`:849-887`, `SELECT … FOR UPDATE`), no bypass;
`isManagementRole`/override path: zero hits. PH's design (`a4dfeaf`,
`223af59`, `ca24372`): management-only (finance excluded), required
reason, `CREDIT_LIMIT_OVERRIDE` audit row, chokepoint on the sink.
**Kernel-routed port sketch:** over-limit raises a
`Finding{Code:"credit_limit_exceeded", RequiresApproval:true}` →
`DecisionPending` (`pkg/approvals/approvals.go:47-114`); release
requires `approvals.Transition` to `DecisionApproved` by an actor with
`CanApprove()` (`:127-144`) plus a required reason via kernel
`approval.Record`. Actor authority via `currentApprovalActor`
(`delete_approval_service.go:101-117`); PH's finance-exclusion becomes
"which sessions get AuthorityApprove". Replaces the bespoke gate with
the substrate's actor-authority boundary — the AI-authority boundary
and audit trail come free.

---

## Band 1 — money/integrity invariants

| Row | **Verified verdict** |
|---|---|
| 1-FM field-mask updates | **PORT** (all 3 methods + 2 siblings) |
| 1-RBAC unguarded mutators | **PORT** (20 mutators + 3 seed/migrate guards) |
| 1-HOLLOW send-block | **PORT** |
| 1-HMAC backfill/verify | **PORT** |
| 1-POVAT VAT basis | **PORT** (field only — threshold NOT corrupted; premise corrected) |
| 1-FX currency labels | **DIFFERENT** — verify backend populates `currency` |
| 1-SYNC hardenings | **DECISION** (Postgres-sync vs Turso/CDC) |
| 1-PROMOTE FK bug | note carried to Mission C |

### 1-FM — field-mask partial-update protection — PORT
All three OSS methods still raw-`Save` the client payload (GORM Save
writes zero-values), wiping server-owned fields on partial payloads:
- `UpdateSupplierInvoice` (`supplier_invoice_service.go:251`, Save
  `:311`): wipes ApprovedBy/At, POMatchOK, GRNMatchOK, MatchStatus,
  OCRDocumentID/Confidence, JournalEntryID, audit fields, Version.
- `UpdateGRN` (`grn_service.go:260`, Save `:289`): wipes QCStatus/
  Notes/Date/By + audit.
- `UpdateCustomerContact` (`app_order_customer_surface.go:1511`, Save
  `:1518`): can orphan the contact by wiping CustomerID.
PH masks: `supplier_invoice_service.go:261/331-348`,
`grn_service.go:258/287-301`, `app.go:11794/11801-11815`. Fold in the
same-pattern siblings `UpdateSupplierContact` and
`UpdateSupplierInvoiceWithPayment`.

### 1-RBAC — unguarded frontend-bound mutators — PORT (loudest row)
Only `SeedDefaultRoles` is guarded in OSS. **20 bound state-mutators
have no permission check** (verified method bodies):
`StartFileWatcher:244` / `StopFileWatcher:257` /
`ConfigureWatchPaths:290` / `TriggerSync:358` / `RetryFailedSyncs:383`
/ `ClearSyncHistory:411` (app_watcher.go); `StartArchaeologyScan:236` /
`CancelScan:353` (archaeologist.go); `ApplyReconciliation:227` /
`AddManualCustomerMapping:787` / `ExportReconciliationReport:806`
(data_reconciliation.go); `AcknowledgeAlert:197` / `DismissAlert:206`
(survival_intelligence.go); `MarkInboxDocumentProcessed:224` /
`StoreCustomerGraph:415` (runtime_handlers.go);
`SaveParsedEmailAsRFQ:526` (msg_parser.go);
`CreateCostingSheetVersion:139` /
`UpdateCostingSheetWithVersionCheck:220` (P1_SALES_PIPELINE_FIXES.go);
`ProcessInvoice:43` (pipeline_handlers.go). Plus three unguarded
overwrite vectors needing PH's internal/exported admin split:
`SeedCompanyBankAccounts` (`bank_accounts_service.go:29` — writes
IBANs), `SeedProductDatabase` (`product_service.go:69`),
`MigrateBankAccountEncryption` (`bank_accounts_service.go:345`).

### 1-HOLLOW — send-block — PORT
OSS `SendCustomerInvoice` (`customer_invoice_service.go:1170`) checks
Draft status (`:1185`) but never counts items — a hollow invoice with a
non-zero total can be sent. PH guard: `customer_invoice_service.go:
1419-1424` (MON-007). Five-line port after the Draft check.

### 1-HMAC — backfill + verify — PORT
OSS computes `computeDocumentHMAC` at creation
(`customer_invoice_service.go:32`, invoked `:924`) but has **no startup
backfill and no VerifyInvoiceHash** — bulk-imported invoices carry
blank hashes and nothing can check tampering. (The many InvoiceHashB64
hits are the unrelated ZATCA XML hash in `pkg/compliance/saudi`.) Port
PH's salt-gated `backfillInvoiceHashesInternal`
(`customer_invoice_service.go:72`, startup wire `app.go:1750`) +
`VerifyInvoiceHash` (`:122`, `hmac.Equal` constant-time).

### 1-POVAT — PO VAT basis — PORT, premise corrected
OSS stores `po.VATAmount = roundTo3(po.SubtotalForeign * 0.10)`
(`purchase_order_service.go:141`) — wrong by 1/rate on any non-BHD PO
(≈2.4× for EUR), wrong on the PO PDF VAT line and any input-VAT read of
the field. **However the spec's premise "feeds the 5K threshold" does
NOT hold in OSS:** `TotalBHD = TotalForeign × ExchangeRate`
(`:142-143`) is arithmetically identical to PH's fixed
`SubtotalBHD + VATAmount`, and the approval threshold reads TotalBHD
(`:210`, `:540-551`) — correct in both repos. The create-from-order
path (`:854-859`) is all-BHD and fine. One-line financial fix at
`:141`, golden-first.

### 1-FX — non-BHD line-item labels — DIFFERENT (one verification owed)
PH's June-15 fix (`5a30a34`) switched its OpportunityDetail from a
nonexistent `item.currency` to `source_currency`. OSS reads
`item.currency` too (`OpportunityDetail.svelte:135`) **but declares and
maps the field** (`:20`, `:223`) and already uses `formatLineMoney` at
`:563-564`. Not the same bug — provided the backend actually populates
`currency` on line-item JSON. Residual check: confirm it's populated;
if blank, same defect class via a different field.

### 1-SYNC — Postgres sync hardenings — DECISION
OSS keeps the legacy Postgres sync trio (`db_manager.go`,
`db_sync_service.go`, `sync_service_impl.go`) with `MigrateRemote`
(`db_manager.go:165`) but none of `ffbe9c7`'s hardenings
(varchar→TEXT widening, NULL-PK UUID backfill, `SKIP_REMOTE_MIGRATION`)
— zero code hits. OSS also carries the newer CDC path
(`pkg/sync/turso/cdc.go`) as its forward direction. Port the three
hardenings ONLY if PH stays on Postgres sync; moot on Turso/CDC.
**Commander decides** (stop-and-ask registry).

### 1-PROMOTE — carried note for Mission C
PH `5545793`: `PRAGMA foreign_keys=OFF` inside a transaction is a
SQLite no-op — the data-promote had never worked. The Mission C
importer must set the pragma OUTSIDE any transaction.

---

## Band 2 — the write-policy seam — DECISION (recommendation attached)

Full behavior-by-behavior measurement (18 behaviors across PH's five
policy files) decomposes the seam into four distinct fates — **do not
treat it as one unit**:

1. **Identity-write core (behaviors 1-5): subsumed inline, 3 gaps.**
   OSS Create/UpdateCustomer + Create/UpdateSupplier
   (`app_order_customer_surface.go:1379-1399`, `:1722-1750`,
   `:1691-1711`, `:1770-1785`) carry the same ID formats and audit
   preservation, but miss: bidirectional CustomerCode↔CustomerID fill,
   update-path blank-field refill (GORM Save can blank an omitted
   code), and the supplier Rating fallback. Recommended: lift into
   `pkg/crm/customer` as a WritePolicy and close the gaps in passing.
2. **Bank-reconciliation lifecycle (6-8): semantic fork, flag don't
   auto-port.** PH refuses edits on Reconciled/Verified statements
   until reopened; OSS's banking engine **silently auto-reverts**
   finalized statements to InProgress on line change
   (`pkg/finance/banking/service.go:646-650`), and lacks the status
   normalizer + editable-status whitelist. A deliberate design fork to
   ratify, not a hole. **Commander decides.**
3. **Seed enrichment + canonical supplier catalogue (9-14): overlay
   config, not code.** PH's principal names violate the OSS
   synthetic-data invariant; express as overlay seed config (Mission D
   territory) with synthetic values in this repo.
4. **Product-supplier linkage (15-16) + valuation fallback /
   weighted-average costing (17-18): genuine capability gaps — port as
   engines.** OSS ships a `"sup_"+code` placeholder
   (`product_service.go:118`) vs PH's multi-token resolver, and has no
   reference-cost fallback chain or weighted-average movement valuation
   (only the reporting rollup at `app_accounting_inventory.go:1365`).

**Wave 2 resolution (2026-07-06, PC-D5):** all four fates executed.
(1) Identity-write core → `pkg/crm/customer/write.go`: bidirectional
Code↔ID fill, PH's strip-until-4 prefix rule, G1 field-mask merges on
both updates (the OSS `Save(&incoming)` was live-wiping 11 server-owned
customer metric columns), supplier Rating-0 fallback, UUID-shaped
identifier repair. (2) Bank-recon: shipped in Wave 1 (PC-D4).
(3) Seed catalogues: deferred to Mission D as overlay config; the
supplierlink engine takes its vocabulary by injection. **Wave 3
resolution (2026-07-06, PC-D8):** the vocabulary is now overlay config
(`overlay.supplier_aliases` → `supplierLinkAliases()`); a sovereign
deployment ships its real principal catalogue in overlay.json. Demo
seed rows stay code as fixtures gated by `seed_sets` — the config seam
is the gate, not the rows. License key prefix, banking division
normaliser, VAT default chain, and the letterhead asset key also
closed into the overlay (see PC-D8 / docs/PH_SOVEREIGN_FORK.md). (4) Engines
built: `pkg/crm/supplierlink` (four-tier resolver + two-pass token
search, wired into seed / PO inference / inventory alerts) and
`pkg/inventory` (reference-cost chain + fallback-aware weighted
average, wired into `RecordStockMovement`, the new GRN-receipt
reconciliation `procurement_inventory_policy.go`, discrepancy costing —
replacing a literal `rejectedQty*100.0` placeholder — and the valuation
rollup). Ground bugs the port exposed, fixed in passing: inventory +
comment + costing tables were in NO migration set (fresh DBs never
created them), movement numbering used MySQL `YEAR()` (matched nothing
on SQLite), and `GetInventoryValuation` filtered string warehouse IDs
with a `*uint`.

---

## Band 3 — robustness (port opportunistically, non-blocking)

| Row | **Verified verdict** |
|---|---|
| 3-PARSE (ZIP / RTF / OCR) | **3× PORT** (mechanical, same defects live in OSS) |
| 3-PLAT (7 platform bugs) | **5× PORT + 2 N/A** |
| 3-UI canonicalization | **DISSOLVED** (Onyx & Ether) |
| 3-CONN connection hardening | **PARTIAL → PORT** |

- **3-PARSE:** OSS still has the unguarded `int64(UncompressedSize64)`
  (`artifacts.go:168,192`), the byte-dropping
  `strconv.ParseInt(hex,16,8)` (`ocr_service_simple.go:965`), and
  unclamped float→uint8 in predator vision
  (`pkg/ocr/predator/predator_vision.go:152-155,330-333`). Mirror the
  PH diffs + security tests.
- **3-PLAT:** the spec's hope that `StandardOverlayDirs` already solved
  these does NOT hold — it covers only overlay/DB resolution. Five
  sites still hardcode `~/.local/share` or `$HOME`
  (`config.go:777-779`, `startup_diagnostics.go:19`,
  `app.go:201,412,443`, `app_costing_exports_surface.go:1770`), and
  the **field-crypto salt** (`field_crypto.go:338-348`) is exe-dir-only
  with no writability probe — silent field-encryption failure on a
  Program Files install. Highest-value single port. Consolidate via one
  `appDataDirPath()` helper. N/A: NSIS installer script, dev-master-key
  test strip (OSS invariant #1 makes it moot).
- **3-CONN:** OSS `db_manager.go` has pool lifetime settings
  (`:116-118`) but no `statement_timeout`/`connect_timeout` in the DSN
  (`:88-99`), no `SetConnMaxIdleTime`, and two unbounded `Ping()`
  (`:121`, `:136`) where PH uses 5s `PingContext`. Blast radius = the
  optional online-sync path only.

---

## Summary

| Verdict | Rows |
|---|---|
| DISSOLVED | B1, B2, B3-binding, D1/D2-matcher, 3-UI |
| PORT (clean, pre-authorized by spec Mission B order) | A1, B3-UI, B4+B4a, D1/D2-guard, D3, F, 1-FM, 1-RBAC, 1-HOLLOW, 1-HMAC, 1-POVAT, 3-PARSE ×3, 3-PLAT ×5, 3-CONN, Band-2 engines (linkage, valuation), Band-2 identity-write consolidation |
| STOP-AND-ASK gated | C (invoice PDF appearance, 4 sub-ports) |
| DECISION (Commander) | E (delete semantics — composition recommended), 1-SYNC (Postgres vs Turso/CDC), Band-2 bank-recon lifecycle fork |
| DIFFERENT, residual check | 1-FX (verify backend populates line-item `currency`) |
| Carried note | 1-PROMOTE → Mission C importer |

**Live integrity gaps found in OSS by this measurement** (the campaign
paying for itself before a single port): party deletes orphan
transactional children (E); 20 unguarded bound mutators incl. the
IBAN-writing seed (1-RBAC); hollow invoices sendable (1-HOLLOW); no
invoice-hash verify/backfill (1-HMAC); PO VATAmount stored on the
foreign subtotal (1-POVAT); Draft-invoice edits leave stale header
totals with a UI that pretends Amount is editable (B4); field-crypto
salt hard-fails on Program Files installs (3-PLAT).

Mission B priority order stands as the spec wrote it — F, B4/B4a,
1-FM, 1-POVAT — with 1-RBAC promoted alongside them on the strength of
the measurement.
