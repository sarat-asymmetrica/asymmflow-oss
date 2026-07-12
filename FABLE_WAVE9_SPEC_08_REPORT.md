# FABLE Wave 9.8 — Spec-08 "Residue Zero" — Status Report

**Branch:** `feat/fable-wave9-8-residue-zero` (off `main` @ `b4e88fa`). **Not merged, not pushed, not tagged** — left for owner review.
**Operating model:** Opus 4.8 orchestrator (senior designer / tech-lead + correctness gate); Sonnet 5 coders in file-disjoint batches; Phase-A recon first; constitutional review of every diff; central bindings + schema-golden regen.
**Severity honesty is law:** an accurate red beats a false green. One pre-existing environmental red is reported truthfully below (§Gates).

---

## Phase A — recon verdicts

| # | Question | Verdict (feeds) |
|---|---|---|
| **A1** | GRN receive panel + serial/discrepancy wiring | Receive panel assembled in `PurchaseOrdersScreen.svelte openReceivePanel()` (:808-832); `ReceiveLine` (:194-205) carries no serial flag. `RequiresSerialTracking` exists **on the product only** (`ProductMaster`, `pkg/crm/domain.go:193`), NOT on `PurchaseOrderItem`. Threading precedent: `OrderFulfillmentItem` query-time overlay (`delivery_note_service.go:1280-1321`). Backend enforcement of record: `ReceiveAgainstPOWithSerials` `len(serials)==qty` (`grn_service.go:771-778`) — but only fires when serials are *present* (the gap). **Key finding:** `RaiseGRNDiscrepancy`/`ResolveGRNDiscrepancy` are **stubs** — the `GRNDiscrepancy` row is never persisted (`grn_service.go:1002 _ = discrepancy`); only the `SupplierIssue` side-effect actually lands. → shaped B1(b) to reuse the persisted `SupplierIssue`, not a new table. |
| **A2** | GL close model | `ExportGeneralLedgerCSV` (`statement_export_service.go:89`) shows movement-within-year only, running balance seeded at 0. Opening balance is a **pure computed roll-forward** from `journal_lines` where `fiscal_year < N`, same Asset/Expense sign rule as `PostJournalEntry` (`app_accounting_inventory.go:250-260`). **No** stored period-close/opening-balance table exists (searched). **No** on-screen GL card shares the derivation → fix is export-only. Computed roll-forward recommended over a stored close model. |
| **A3** | Allocation reality | `ProjectMember.AllocationPercent` (`collaboration_service.go:83`, table `project_members`), captured via `AddCollaborativeProjectMember` (:1708), clamped ≤100. **No** cross-project aggregate read exists today (confirms "decorative"). Cross-project total = SUM over `is_active=true` + `projects.status='active'`, excluding the edited project. WARN precedent to mirror: the duplicate-check pattern (`CheckDuplicateRFQ` + `QuickCaptureModal` confirm→same-save, no `force` param). |
| **A4** | Employee compliance surface | Model is `Employee` (`collaboration_service.go:16-38`, table `employees`). **Child table** recommended over columns. FieldCrypto pattern to mirror = **`SettingsService`** (the current-correct one), NOT the deprecated hardware-bound bank-account path. RBAC: reuse `hr:create`/`hr:update` + admin-only + `hr:view`→`tasks:view`. Notification home = `NotificationsScreen` (Article V) via `createTaskNotification` shape. Confirmed PH has **nothing** for CPR/passport/visa/permit today. Golden regen via `-update-schema-golden`; `&Employee{}` registered in 5 migration lists. |
| **A5** | Stock-history repair | `StockMovement` (`pkg/crm/domain.go:651-676`, table `stock_movements`). Pre-9.7 doubles have empty `reference_id`; 9.7 fix stamps `ReferenceType/ReferenceID` in `ApproveStockAdjustment`. Heuristic groups by `inventory_item_id, quantity, direction` (+ empty `reference_id`, + timestamp window). Safe posting reuse = `RecordStockMovement` (never hand-insert). Compensating movement = opposite direction, `ReferenceType="StockMovementRepair"`. Scaffold to mirror = ph_holdings B2-toolkit (diagnose/repair/verify/checkpoint). |
| **A6** | `rfq_datas` no-op | Confirmed dead: live table is `rfq_data` (singular); `rfq_datas` (plural) never exists; `addColumnIfNotExists` silently no-ops on a missing table (`app.go:1391-1393`); the singular table already has all 4 columns via `RFQData` GORM struct + golden. → **delete** the dead block. |

---

## Phase B — per-item status

### B1 — GRN finishing ✅ SHIPPED
`pkg/crm/domain.go`, `pkg/finance/domain.go`, `purchase_order_service.go`, `grn_service.go`, `PurchaseOrdersScreen.svelte`, `grn_inventory_reconciliation_test.go`.
- **(a) Serial threading (no schema change):** `RequiresSerialTracking bool `gorm:"-"`` added to both `PurchaseOrderItem` structs (query-time overlay — confirmed absent from the golden). `enrichPOItemsWithSerialTracking` (`purchase_order_service.go:395-432`) stamps it via **one batched** `ProductMaster` lookup, wired into `getPurchaseOrders`/`getPurchaseOrderByID`. Frontend: serialized lines require `serials.length===receiveQty` and force the whole receive through `ReceiveAndCompletePOWithSerials`; non-serialized lines show no serial inputs.
- **Serialized-line-without-serials now rejected — two layers:** frontend routing + backend `SERIAL_REQUIRED` guards on both `ReceiveAgainstPOWithSerials` (empty-serials case, the recon gap) and `ReceiveAndCompletePO` (defense-in-depth). The `len(serials)==qty` check stays the enforcement of record. No posting/locking/rounding change.
- **(b) Discrepancy wiring (reuse persisted `SupplierIssue`, no new table/screen):** on a short-ship (`rejectedQty>0`) a reason is now **required**; after receive, `RaiseGRNDiscrepancy(grnId, itemId, reason, 'quantity_short', rejectedQty)` records a `SupplierIssue` that surfaces on the existing **`SupplierDetailView` → Issues tab** (resolvable via `ResolveSupplierIssue`). `grnId` sourced from `GRNResponse.id` — no posting change. Deprecated `GRNScreen` stays retired.
- **Test:** `TestReceive_SerializedProductWithoutSerials_Rejected` (both paths reject; no GRN stranded) — passes.
- **AC met:** serialized line cannot be received without serials ✅; short-ship recorded as discrepancy from the same panel ✅; no posting/locking change ✅; GRNScreen retired ✅.

### B2 — GL opening-balance carry-forward ✅ SHIPPED
`statement_export_service.go` (only). Computed roll-forward: one added query (`fiscal_year < year AND is_posted`), reduced per account with the **identical** Asset/Expense sign rule, seeds each account's running balance and emits an "Opening Balance" row. Accounts with a nonzero opening but no in-year activity still appear. Header note rewritten to state balances **are** carried. **Year-1 opens at 0** (empty prior set → map zero-value — verified by inspection). `accountByID` is built from *all* non-deleted accounts, so prior-year-only accounts resolve to their true `AccountType` (sign-correct). No posting/journal/table touched. **AC met.**

### B3 — Allocation capacity WARN ✅ SHIPPED
`collaboration_service.go` (new read-only `GetEmployeeAllocationSummary` + `AllocationSummary`/`AllocationProjectLine` structs), `collaboration.ts`, `WorkHub.svelte`, `collaboration_allocation_summary_test.go`.
Server computes `OtherProjectsTotal` (SUM over active memberships in active projects, excluding the edited project) — client never sums for the decision. `handleSaveMember` + `handleAddProjectMembers` call the summary first; if `otherTotal + newAllocation > 100`, `confirm.ask({variant:'warning'})` shows the person, the true total, and the per-project breakdown; confirm falls through to the **unmodified** `addProjectMember` — no `force` param, no schema change, no hard block. Gated read-only (`projects:view`). Test covers over-allocation, exclude-project math, inactive/archived exclusion, blank-id error — passes. **AC met.**

### B4 — Visa/CPR/permit tracking ✅ SHIPPED (the one new feature)
`employee_compliance_service.go` (new), `employee_compliance_service_test.go` (new), `PeopleHub.svelte`; central: registered `&EmployeeDocument{}` in 5 migration lists + golden + bindings.
New child table `employee_documents` (CPR/passport/visa/permit + number + expiry + permit subtype + notes). **PII: the document number is FieldCrypto-encrypted at rest** (`DocNumberEncrypted`, `json:"-"`) mirroring the current-correct `SettingsService` pattern — `encryptDocumentNumber` **refuses to store** if FieldCrypto is unavailable (never plaintext-falls-back); decrypt is `IsEncrypted`-guarded and never leaks a raw value; list view masks to last-4. **RBAC:** writes require `hr:create`/`hr:update` + admin-only session (mirrors `CreateEmployeeProfile`); reads `hr:view`→`tasks:view` — no new permission strings, no role widening. **Expiring-soon (≤60 days):** `ScanExpiringEmployeeDocuments` publishes a `document_expiry` notification through the existing Article-V `NotificationsScreen` (mirrors `createTaskNotification`), idempotent via `NotifiedAt` (re-armed on expiry change); run opportunistically from `ListEmployeeDocuments` and exposed for optional startup wiring. Frontend: new "Compliance" sub-tab in PeopleHub (list with masked number + days-until-expiry cue + add/edit/delete). **Zero payroll/money contact.** 6 unit tests pass (encrypts-at-rest, round-trip, non-admin rejected, scan notifies + idempotent, expiry re-arm, soft-delete). Golden diff = only `employee_documents` (+ 4 indexes), `doc_number_encrypted` column, no plaintext number column. **AC met.**

### B5 — Historical stock-movement repair plan ✅ DELIVERED (report + draft, zero writes)
`b2_stock_adjustment_diagnostic_test.go`, `b2_stock_adjustment_repair_test.go`, `b2_stock_adjustment_verify_test.go`, `b2_stock_adjustment_checkpoint_test.go` (all new; package `main`; inert by default).
- **Toolkit (mirrors ph_holdings B2-toolkit):** diagnose (read-only) → repair (snapshot-first + dry-run) → verify → checkpoint, each gated behind its own env flag (`B2_STOCK_DIAGNOSE` / `B2_STOCK_COMMIT` [+ `B2_STOCK_DRYRUN`] / `B2_STOCK_VERIFY` / `B2_STOCK_CHECKPOINT`), DB path from `B2_STOCK_DB_PATH`. With **no env set every test `t.Skip`s** — proven: `go test -run TestB2Stock .` touches no DB and passes. Repair posts **compensating** movements only (Article III — never deletes), via the safe `RecordStockMovement` entry point, idempotent (skips any extra already carrying `ReferenceType="StockMovementRepair"`).
- **Count (read-only, owner-provided copy, aggregate only):** run against a scratchpad copy of `ph_holdings/ph_holdings.db` (16 MB, 2026-06-29; copied so the live file was never opened for write) → **0 doubled groups, 0 extra movements**. Independent read-only cross-check: that DB's `stock_movements` table is **empty (0 rows)** — so the 0-count reflects *this snapshot only*. **Owner action:** re-run `B2_STOCK_DIAGNOSE=1 B2_STOCK_DB_PATH=<current-prod.db>` against the *current* production DB before concluding; if it reports doubles, dry-run then commit with the delivered script.
- **AC met:** count reported ✅; repair script exists with dry-run + verify ✅; **zero writes performed by this wave** ✅.

### B6 — Micro-residue ✅ SHIPPED
`app.go`, `DESIGN_CONSTITUTION.md`.
- **(a)** Deleted the dead `rfq_datas` migration block (`app.go:1278-1282`), replaced with a one-line explanatory comment; the adjacent `opportunities` migration line was preserved; the guarded `rfq_data`/`rfq_datas` loop (~:1434) left intact (harmless, `HasTable`-guarded).
- **(b)** Recorded ratification §0.1 in `DESIGN_CONSTITUTION.md` Article VII as a new "Ratifications log" item (grep-able: `Offer.Stage`, `ratified`, `Spec-08`) — Offer.Stage keeps its own DB-CHECK vocabulary, dormant Cap'n Proto enum untouched, future waves must not re-open.
- **AC met:** no dead migration calls ✅; ruling findable ✅.

---

## Central integration (orchestrator-owned)

- Registered `&EmployeeDocument{}` after `&Employee{}` in all 5 migration lists: `trading_models.go tradingModels()`, `collaboration_service.go`, `collaboration_sync.go`, `db_manager.go`, `deployment_audit.go`.
- **Schema-golden** regenerated deliberately (`-update-schema-golden`) — diff reviewed: **only** the new `employee_documents` table + 4 indexes; no `requires_serial_tracking` column (B1's `gorm:"-"` overlay confirmed schema-neutral); no other table touched.
- **Wails bindings** regenerated centrally (`wails generate module`): all 6 new methods (`GetEmployeeAllocationSummary`, `Create/Update/Delete/ListEmployeeDocument`, `ScanExpiringEmployeeDocuments`), `requires_serial_tracking` on both `PurchaseOrderItem` variants, and the new DTO classes.
- **Integration fix (caught at gate):** B3 had imported `GetEmployeeAllocationSummary` from `SyncServiceBinding`, but that method exists only on `main/App` (no delegating wrapper) — moved the import to `main/App`. (Root cause: my file-ownership map assigned `collaboration.ts` to both B3 and B4; the concurrent Edits did NOT clobber each other, but the wrong import surface slipped through — fixed centrally.)
- **Frontend fix (caught at gate):** B4 used `toast.error` (nonexistent on the toast store) — corrected to `toast.danger` to match the rest of the file. svelte-check returned to 0 errors.

---

## Gates

- `go build ./...` — **clean**
- `go vet ./...` — **clean**
- `svelte-check` — **0 errors / 14 warnings** (matches baseline exactly)
- `vite build` — **clean** (pre-existing >500 kB chunk advisory only); `frontend/dist/index.html` restored post-build per convention
- schema-golden — regenerated deliberately, diff = only `employee_documents`
- `go test -count=1 -timeout 1800s ./...` — **green except the one pre-existing red below.** The final run failed with *exactly* one test — `TestGetHardwareID_ByteIdenticalToWmic` — identical to the clean-baseline run captured before any change on this branch (same single failure, ~247s). **This wave introduced zero new failures.** All B1/B3/B4 tests pass; B5's four scripts `t.Skip` (inert by default), as designed.

**Pre-existing environmental red (NOT introduced by this wave, out of scope):** `TestGetHardwareID_ByteIdenticalToWmic` fails on this machine because `wmic baseboard get serialnumber` returns the BIOS placeholder `"Default string"` while `getHardwareID()` correctly returns the stable persisted hostname (`"Sarat-AI"`). This is crypto-key-derivation (`settings_service.go`), touched by nothing in B1-B6, and was **already failing on `main` before any change on this branch** (verified against a clean baseline run). Per CLAUDE.md invariant #5 (crypto = stop-and-ask) and this spec's hard boundary (only §3 items move), it was deliberately left untouched. It is tracked as a known Spec-07 residue item ("hardware-ID sidecar security").

---

## Deviations & judgment calls

1. **B1(b) reuses `SupplierIssue`, not a new `grn_discrepancies` table.** A1 found `RaiseGRNDiscrepancy`/`ResolveGRNDiscrepancy` are stubs that never persist a `GRNDiscrepancy` row — only the `SupplierIssue` side-effect lands. The spec's wording ("records through the existing backend", "no new screen", "a list in an existing operations surface") is honored by surfacing the discrepancy as a `SupplierIssue` on the existing `SupplierDetailView` Issues tab. No new persistence table was built. Residue: `ResolveGRNDiscrepancy` remains a no-op stub (untouched per the "do not build a new screen" ruling); resolution flows through `ResolveSupplierIssue`.
2. **B5 count is 0 because the available DB snapshot's `stock_movements` table is empty.** Reported honestly rather than presented as a clean production audit. The script is delivered for the owner to run against the current production DB.
3. **B4 collaborative-sync payload excludes the encrypted document number** (`json:"-"` on `DocNumberEncrypted`) — a benign PII-minimizing gap; cross-device sync won't propagate the number. Acceptable for offline-first scope; flagged as minor residue.

---

## Keep-list attestation

No audit §4 keep-listed screen or Wave 9.x shipped behavior was altered. RBAC is server-side and **never widened** (B3/B4 reuse existing permission strings; no role changes). PII additions (B4) use FieldCrypto, not plaintext columns. No posting/rounding/tax/payment/payroll math was touched (B2 = display/export derivation only; B5 = zero writes). Deprecated `GRNScreen` stays retired. No sensory/brand work; no Spec-09 ecosystem items (CRLF/decompositions/DPAPI/staticcheck) touched — the CRLF line-ending advisories are the repo's existing autocrlf behavior, deliberately left for Spec-09.

---

## Residue after this wave

- `ResolveGRNDiscrepancy` GRN-side resolution stub (resolution works via `SupplierIssue`).
- B5 repair script is **delivered, not run** — owner to execute against current production DB.
- B4 encrypted doc-number not propagated over collaborative sync (minor).
- Pre-existing `TestGetHardwareID_ByteIdenticalToWmic` environmental failure (Spec-07 residue: hardware-ID sidecar).
- Everything else on the PH-domain deferred ledger is closed. Remaining open work = Spec-09 (ecosystem) + Wave 10 (owner-reserved Sensory & Brand).

---

## Owner questions

1. **B5 production run:** do you want me to run the delivered diagnostic against the *current* production PH DB (read-only) to get a real doubles count, or will you run it yourself? (This wave performed zero writes and only saw an empty-movements snapshot.)
2. **B4 startup scan:** `ScanExpiringEmployeeDocuments` currently runs opportunistically when the compliance surface is opened. Want it wired to app startup / a daily tick as well? (Left out to avoid editing `app.go` startup during a parallel batch; trivial to add.)
3. **`TestGetHardwareID_ByteIdenticalToWmic`:** confirm it stays a Spec-07/09 hardware-ID item and is out of scope here (I did not touch crypto-key derivation).
