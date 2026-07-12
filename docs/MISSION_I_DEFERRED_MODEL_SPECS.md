# Mission I — Deferred Model Port Specifications

**Scope:** Port blueprints for the three PH models the Commander deferred at decision
**D-I-4 / PC-D20** (2026-07-09): `employee_archive_requests`, `data_quality_reviews`,
and `extracted_documents`. All three are **ABSENT in the OSS repo today**. The fourth
D-I-4 model, `costing_sheet_attachments`, ports now (Wave 7, unblocks I-25) and is out
of scope here.

This document is the measured blueprint a future wave implements from. Every reference
below is `file:line` in the reference tree `C:\Projects\asymmflow\ph_holdings`
(READ-ONLY) or in this OSS repo. No real PH data appears here; any illustrative value
uses the synthetic canon (`SYNTHETIC_IDENTITY.md`).

> **Layer note.** All three PH models are root-surface (`package main`) GORM models
> embedding the main-package `Base` (id / created_at / updated_at / deleted_at /
> version / created_by). None carry sector vocabulary that would force a `pkg/`
> engine. Per `CLAUDE.md`'s layer law they port as **root-surface models registered
> through `tradingModels()`** (`trading_models.go`), which the composition seam
> (`pkg/runtime/composition`) migrates at boot. Company-specific facts stay
> configuration, not code.

---

## Summary

| Model / table | What it is | PH source | OSS today | Surface to port | Recommended |
|---|---|---|---|---|---|
| `employee_archive_requests` | Two-state approval record for archiving an employee, with cascade that closes access links + project memberships and notifies the requester | `employee_archive_service.go` (335 LOC) | **Behavioral gap**: OSS archives employees by direct field write — no request record, no admin gate on the transition, no cascade, no notification | 1 model + 4 new `Employee` columns; 2 public + 3 helper methods; 2 existing OSS callers to rewire; 1 review screen + 2 API wrappers | **1st — highest correctness value** |
| `data_quality_reviews` | Admin review-ledger over computed data-hygiene issues (blank/duplicate customers, orphan opportunities/offers); dismiss/resolve suppresses an issue from the queue | `user_feedback_hardening_service.go:53` (+ methods 693–924) | **Absent, additive**: no screen, no caller, no table | 1 model (+1 transient DTO); 3 public + ~4 helper methods; 1 new screen | **2nd — clean additive port** |
| `extracted_documents` | Flat OCR-scan metadata from the historical OneDrive extraction sweep (~359 rows in the PH snapshot) | *No Go model* — only a sync-exclusion string, `sync_coverage_service.go:207` | Absent, and **nothing reads it in PH either** | 0 models, 0 methods | **3rd — formally close as skip-with-reason (PC-D16); do not build unless a reader appears** |

**Recommended porting order and rationale**

1. **`employee_archive_requests` first.** It is the only one of the three with a
   *live correctness defect* in OSS. OSS `UpdateEmployeeProfile`
   (`collaboration_service.go:715`) and `SetEmployeeEmploymentState` (`:795`) archive
   an employee with a direct `is_active=false` / `employment_status='archived'` write:
   no approval record, no admin-only gate on the archive transition, no cascade to
   `employee_access_links` / `project_members`, and no requester notification. An
   archived person keeps active access links and active project memberships — a
   governance + data-integrity gap on a shipping surface. Porting closes it.
2. **`data_quality_reviews` second.** Larger surface (three public methods, an
   issue-detection scan, and a whole screen) but purely additive — OSS has no caller
   or screen today, so regression risk is near zero. It is an admin data-hygiene tool,
   valuable but not load-bearing for day-one transactions.
3. **`extracted_documents` last — and the recommendation is to *close*, not build.**
   PC-D16 already measured it: dead metadata, no reader in PH, superseded by the
   at-cutover OneDrive re-scan. Building an OSS model would be inventing a table with
   no consumer. The porting task is to record the skip-with-reason so a later wave
   does not re-litigate the question.

---

## 1. `employee_archive_requests`

### 1.1 Model schema

Defined at `employee_archive_service.go:13`, `TableName()` at `:34`.

| Column | Go field | Type / tags | Notes |
|---|---|---|---|
| (Base) | `Base` | `id` PK, `created_at`, `updated_at`, `deleted_at`, `version`, `created_by` | Same main-package `Base` OSS uses |
| `employee_id` | `EmployeeID` | `size:36; index` | Target employee |
| `employee_name` | `EmployeeName` | `size:255` | Denormalized label |
| `requested_by` | `RequestedBy` | `size:36; index` | Requesting admin's employee id |
| `requested_by_name` | `RequestedByName` | `size:255` | |
| `reason` | `Reason` | `type:text` | Required |
| `status` | `Status` | `size:30; index; default:'pending'` | `pending` → `approved` / `rejected` |
| `required_approvals` | `RequiredApprovals` | `int; default:1` | Two-approver scaffold; live flow collapses to 1 |
| `first_approved_by` / `_name` / `_at` | `FirstApprovedBy` … | `size:36` / `size:255` / `*time.Time` | First approval |
| `second_approved_by` / `_name` / `_at` | `SecondApprovedBy` … | same | Second approval |
| `rejected_by` / `_name` / `_at` | `RejectedBy` … | same | Rejection |
| `review_notes` | `ReviewNotes` | `type:text` | |

**Companion columns on `employees`** written by the cascade (`performEmployeeArchive`,
`:245`) — PH `Employee` (`collaboration_service.go:32-35`) has them; **OSS `Employee`
(`collaboration_service.go:15`) does not**:

| Column | Go field | Type |
|---|---|---|
| `archived_at` | `ArchivedAt` | `*time.Time` |
| `archived_by` | `ArchivedBy` | `size:36` |
| `archive_reason` | `ArchiveReason` | `type:text` |
| `archive_request_id` | `ArchiveRequestID` | `size:36` |

**Registration:** none in PH beyond GORM struct (PH AutoMigrates its full model set).
In OSS: add `&EmployeeArchiveRequest{}` to `tradingModels()` (`trading_models.go`, in
the People/Collaboration block near `&Employee{}` at line 96) **and** to
`criticalDeploymentModels()` (`deployment_audit.go:103`, alongside `&Employee{}`,
`&EmployeeAccessLink{}`), because a fresh-DB archive must not silent-no-op. Add the 4
`employees` columns to the OSS `Employee` struct.

### 1.2 Service surface

| Method | Location | RBAC / auth gate | Transaction | Lifecycle |
|---|---|---|---|---|
| `RequestEmployeeArchive(employeeID, reason)` | `employee_archive_service.go:36` | `requirePermission("hr:update")` **and** `currentSessionHasAdminRoleOnly()`; requires authenticated employee context; refuses self-archive | Yes — single `db.Transaction`: loads employee, rejects already-archived, upserts a `pending`/`approved` request, calls cascade | Creates request already `approved` w/ `RequiredApprovals=1` (solo-admin fast path); if a `pending` one exists, updates it |
| `ReviewEmployeeArchiveRequest(requestID, decision, notes)` | `:143` | same double gate | Yes — loads request, guards non-`pending`, applies approve/reject, calls cascade on approve | `pending` → `approved` (runs cascade) or `rejected` |
| `performEmployeeArchive(tx, request, reviewer, at)` | `:236` (helper) | — (runs inside caller tx) | Participates in caller tx | Sets employee `is_active=false`, `employment_status='archived'`, end/archive fields; **cascades**: `employee_access_links` → `access_status='archived'`, `is_primary=false`; `project_members` (active) → `is_active=false`, `left_at`. Refuses reviewer self-archive |
| `markEmployeeArchiveNotificationsRead(requestID)` | `:281` (helper) | — | single update | Marks matching `notifications` read |
| `notifyEmployeeArchiveRequester(request, decision)` | `:295` (helper) | — | single create | Writes a `notification` + receipt to the requester |

Both public methods also enqueue collaborative sync ops (`employee_archive_request`,
`employee`) and emit `employees:updated` / `notifications:updated` events.

### 1.3 Wiring

- **Auto-archive callers (the important ones):** PH `UpdateEmployee`
  (`collaboration_service.go:863`) and `SetEmployeeEmploymentState` (`:947`) detect an
  active→archived transition and route it through `RequestEmployeeArchive` instead of
  writing the fields directly. This is the enforcement point.
- **Frontend:** review UI in `NotificationsScreen.svelte` (imports
  `ReviewEmployeeArchiveRequest`, calls it at `:274`); API wrappers in
  `frontend/src/lib/api/collaboration.ts` (`RequestEmployeeArchive` at `:460`,
  `ReviewEmployeeArchiveRequest` at `:468`). No dedicated screen — archive is triggered
  from the People profile edit and reviewed from Notifications.
- **OSS fallback (measured):** OSS `UpdateEmployeeProfile`
  (`collaboration_service.go:715`) and `SetEmployeeEmploymentState` (`:795`) write
  `is_active` / `employment_status` **directly** in a plain `Updates` map — no request
  record, no admin-only gate on the archive transition (the PH `currentSessionHasAdminRoleOnly`
  check is absent), no cascade, no notification. Net degradation: archiving an employee
  in OSS leaves their `employee_access_links` and `project_members` **active**, produces
  no audit/approval trail, and OSS `NotificationsScreen.svelte` has no archive-review
  action. OSS has zero references to either the model or the methods.

### 1.4 Port plan

- **Layer:** root-surface. New file `employee_archive_service.go` (mirror PH),
  `package main`.
- **Model + columns:** add `EmployeeArchiveRequest`; add `ArchivedAt`, `ArchivedBy`,
  `ArchiveReason`, `ArchiveRequestID` to OSS `Employee`. Register in `tradingModels()`
  + `criticalDeploymentModels()`.
- **Rewire callers:** OSS `UpdateEmployeeProfile` and `SetEmployeeEmploymentState` gain
  the active→archived detection branch that routes through `RequestEmployeeArchive`
  (matching PH `:857-871` and `:942-955`). This is the behavioral fix, not just a data
  add.
- **Dependencies present in OSS:** `logAudit` (`app_auth_rbac.go:1054`),
  `GetCurrentEmployeeContext` (`collaboration_service.go:388`),
  `currentSessionHasAdminRoleOnly` (`delete_approval_service.go:25`), `Notification` /
  `NotificationReceipt` models, the collaborative-op enqueue + event helpers. No
  missing primitives.
- **OSS adaptations:** confirm the OSS `Notification` field set (`SourceType`,
  `SourceID`, `ActionRoute`, `ActionPayload`) matches PH's usage in
  `notifyEmployeeArchiveRequester`; adjust `ActionRoute` to the OSS People route.
  Frontend: add the review action to OSS `NotificationsScreen.svelte` and archive
  wrappers to the OSS collaboration API.
- **Schema-golden impact:** two tables change — new `employee_archive_requests` table
  and four new `employees` columns. Regenerate `testdata/trading_schema.golden`
  (`go test -run TestTradingModels_SchemaGolden -update-schema-golden .`) as a
  deliberate, reviewed diff.
- **Test plan (fails without the port):**
  - Schema-golden diff shows the new table + `employees` columns.
  - Deployment-audit test: `employee_archive_requests` present in the fresh-DB critical
    set (mirror the pattern in `deployment_banking_provision_test.go`).
  - Behavioral test (port PH `hybrid_feature_flow_test.go:716-870`): archiving via
    `UpdateEmployeeProfile`/`SetEmployeeEmploymentState` creates an approved request,
    flips `is_active`, sets `ArchiveRequestID`/`ArchiveReason`/`ArchivedAt`, and
    **cascades** access-link + project-member closure; non-admin is refused; self-archive
    is refused; reject leaves the employee active.

### 1.5 Effort shape

1 new model; 4 new columns on 1 existing model; 2 public + 3 helper methods (new
file); 2 existing OSS methods rewired; cascade touches 3 tables + writes notifications;
2 registration lists; 1 screen action + 2 API wrappers; 1 golden regeneration.

---

## 2. `data_quality_reviews`

### 2.1 Model schema

Defined at `user_feedback_hardening_service.go:53`, `TableName()` at `:70`.

| Column | Go field | Type / tags |
|---|---|---|
| (Base) | `Base` | id / created_at / updated_at / deleted_at / version / created_by |
| `issue_id` | `IssueID` | `size:180; uniqueIndex` — stable synthetic id of the computed issue |
| `issue_type` | `IssueType` | `size:80; index` |
| `severity` | `Severity` | `size:40; index` |
| `entity_type` | `EntityType` | `size:80; index` |
| `entity_id` | `EntityID` | `size:100; index` |
| `summary` | `Summary` | `size:500` |
| `detail` | `Detail` | `type:text` |
| `primary_action` | `PrimaryAction` | `size:255` |
| `status` | `Status` | `size:40; index` — `reviewed` / `resolved` / `dismissed` |
| `review_note` | `ReviewNote` | `type:text` |
| `reviewed_by_id` | `ReviewedByID` | `size:100; index` |
| `reviewed_by` | `ReviewedBy` | `size:255` |
| `reviewed_at` | `ReviewedAt` | `*time.Time` |

Transient DTO **`DataQualityIssue`** (`:38`, **not persisted**) is the computed-issue
shape the preview returns and the review consumes.

**Registration:** PH does **not** register this in AutoMigrate; instead it
self-provisions at call time via `ensureDataQualityReviewFoundation` (`:851`) — a raw
`CREATE TABLE IF NOT EXISTS` + `ensureSyncBaseColumns` + `addColumnIfNotExists` loop +
index creation. **OSS should not port that pattern** (`ensureSyncBaseColumns` does not
exist in OSS). Register `&DataQualityReview{}` in `tradingModels()` and drop the
self-migration; GORM AutoMigrate + the golden test replace it.

### 2.2 Service surface

| Method | Location | RBAC gate | Transaction | Behavior |
|---|---|---|---|---|
| `PreviewCustomerDataQuality(limit)` | `:693` | `requirePermission("customers:view")` | read-only scans | Computes issues live: blank/duplicate customer names (via `normalizeDataQualityName`, `:934`), opportunities missing title or customer link, offers missing customer. Overlays existing reviews; **suppresses `resolved`/`dismissed`** from the queue. Cap 500 (default 200) |
| `ReviewDataQualityIssue(issue, action, note)` | `:773` | `currentSessionHasAdminRoleOnly()` (admin-only; not a plain permission) | `Save` upsert keyed on `issue_id` | Actions `reviewed` / `resolved` / `dismissed`; stamps reviewer id/name/time; `logAudit`; emits `data-quality:updated` |
| `GetDataQualityReviewHistory(limit)` | `:831` | `requirePermission("customers:view")` | read | Recent reviews, newest first, cap 500 (default 100) |
| `dataQualityReviewsByIssueID()` | `:914` (helper) | — | read | Map issue_id → review for overlay |
| `normalizeDataQualityName` / `trimToLength` | `:934` / `:926` | — | pure | Dedup key + length clamp |

No cascades — a self-contained review ledger. No FK to the entities it references
(it stores `entity_type` + `entity_id` as loose pointers).

### 2.3 Wiring

- **Frontend:** `frontend/src/lib/screens/DataQualityScreen.svelte` — the only consumer.
  Loads `PreviewCustomerDataQuality(300)` + `GetDataQualityReviewHistory(50)` on mount
  (`:53-55`), calls `ReviewDataQualityIssue` per row (`:70`). Defines its own local
  `DataQualityIssue` / `DataQualityReview` TS types (`:12-39`).
- **OSS fallback (measured):** none — OSS has no `DataQualityScreen`, no methods, no
  table. The admin data-hygiene queue simply does not exist. No degradation of other
  features (nothing depends on it); it is a missing capability, not a broken one.

### 2.4 Port plan

- **Layer:** root-surface. New file (e.g. `data_quality_service.go`), `package main`.
  Port the three public methods + `dataQualityReviewsByIssueID` + the two pure helpers.
- **Registration:** add `&DataQualityReview{}` to `tradingModels()`; **do not** port
  `ensureDataQualityReviewFoundation` — replace raw self-migration with AutoMigrate.
- **Dependencies present in OSS:** `requirePermission`, `currentSessionHasAdminRoleOnly`,
  `logAudit`, `CustomerMaster` / `Opportunity` / `Offer` models, `firstNonEmpty`,
  `addColumnIfNotExists` (unused if AutoMigrate registers the model). The issue-scan
  reads `customers` (`deleted_at IS NULL`), `opportunities`, `offers` — all present in
  OSS `tradingModels()`.
- **OSS adaptations:** confirm OSS column/field names for the scanned entities match
  (OSS `CustomerMaster.BusinessName`, `Opportunity.CustomerID/CustomerName/Title/
  FolderName`, `Offer.CustomerID/CustomerName`). The dedup normalization keeps the same
  Bahrain-suffix stripping (`wll`, `bsc`, `company`, `co`) — domain context, not
  client data.
- **Schema-golden impact:** one new table `data_quality_reviews`. Regenerate the golden.
- **Frontend:** port `DataQualityScreen.svelte` and wire it into the OSS People/Admin
  navigation with `customers:view` visibility.
- **Test plan (fails without the port):**
  - Schema-golden diff shows `data_quality_reviews`.
  - Port PH `user_feedback_hardening_service_test.go:226-323`: preview flags a seeded
    blank/duplicate customer; non-admin `ReviewDataQualityIssue` is refused; `resolved`
    disappears from the next preview; history returns the review; `reviewed` (non-terminal)
    stays visible. Uses synthetic customers (e.g. two "Acme Instrumentation W.L.L." rows
    to trip the duplicate rule).

### 2.5 Effort shape

1 new model + 1 transient DTO; 3 public + ~3 helper methods (new file, self-migration
dropped); 0 cascades; 1 registration list; 1 new screen; 1 golden regeneration; 1
ported test file.

---

## 3. `extracted_documents`

### 3.1 What it is (measured)

**No Go model exists in PH.** The token `extracted_documents` appears only as a string
in the sync-coverage *exclusion* list (`sync_coverage_service.go:207`,
`isKnownLocalOnlyTable`) and in `ARCHITECTURE.md:399`. It is a raw table of flat OCR-scan
metadata produced by the historical OneDrive extraction sweep — **~359 rows in the PH
cutover snapshot, no blobs, no FK references, and no code path in PH reads it**
(measured and recorded in decision **PC-D16**, `docs/PH_CONVERGENCE_DECISIONS.md:311`).
The live OCR audit trail is a *different* table, `ocr_documents` (Go model `OCRDocument`,
present in both PH and OSS `tradingModels()`).

### 3.2 Service surface / wiring

None. No reader, no writer, no screen, no job — in PH or OSS. Its rows are superseded by
the at-cutover OneDrive re-scan (PC-D15).

### 3.3 Recommended disposition: close as skip-with-reason

The faithful-carry test (PC-D16, mirrored) is "does the source system's own app read
it?" — for `extracted_documents` the answer is no, so the migration owes the destination
nothing beyond counting and archiving the rows, which Mission H already does. **Do not
build an OSS model**: it would invent a table with no consumer and add a permanent line
to the schema-golden for dead data.

If a future need appears (e.g. OSS grows an OCR-extraction browsing surface), the
minimal shape is a flat root-surface model — scan-time metadata columns (source path,
document type, extracted-text pointer, confidence, engine, processed-at) mirroring the
`OCRDocument` fields already in OSS — registered in `tradingModels()`. But that is
**new-feature work triggered by a reader, not a parity port.** The Mission I task for
this model is to record the closure so a later wave does not re-open the question.

### 3.4 Effort shape

0 models, 0 methods, 0 screens to port. Disposition = documentation only.

---

## Registration checklist (for the implementing wave)

| Step | employee_archive_requests | data_quality_reviews | extracted_documents |
|---|---|---|---|
| New root-surface model | `EmployeeArchiveRequest` | `DataQualityReview` | — (skip) |
| New columns on existing model | 4 on `Employee` | — | — |
| Add to `tradingModels()` | ✓ | ✓ | — |
| Add to `criticalDeploymentModels()` | ✓ | optional | — |
| Rewire existing OSS methods | `UpdateEmployeeProfile`, `SetEmployeeEmploymentState` | — | — |
| New / changed screen | Notifications review action | new `DataQualityScreen` | — |
| Regenerate schema golden | ✓ | ✓ | — |
| Port test(s) | `hybrid_feature_flow_test.go` archive cases | `user_feedback_hardening_service_test.go` DQ cases | — |
