# Wave 9 Spec-04 Status Report — People & Projects Re-model (The Big Re-wire)

**Branch:** `feat/fable-wave9-4-people-projects` · **Commits:** 5 (4 feature + 1 bindings), off `main` — **NOT merged/pushed/tagged**, left for owner review.
**Gates (final commit `ecbcc04`):** `npx svelte-check` ✅ **0 errors / 14 warnings** (= pre-wave baseline; net **0 new errors, 0 new warnings**) · `npx vite build` ✅ (exit 0) · `go build ./...` ✅ · `go vet ./...` ✅ · `go test ./...` ✅ (exit 0, all 84 test packages) — see the flake note below.

> **`go test ./...` flake (documented, not ours):** the first full run showed one failure — `TestButlerSalesContextRedactsPipelineAmounts` (a Butler sales-context redaction test, entirely outside this wave's change surface). It **passes in isolation** (`go test ./ -run … -count=1` → PASS) and the **re-run of the full suite was clean (exit 0, 0 failures)**. This is the same cross-test-contamination flake Spec-03 documented ("one transient flake on the first run cleared on re-run"). No Butler/redaction code was touched this wave.

**Operating model:** Opus 4.8 orchestrator as senior designer/tech-lead — Phase-A recon work orders (4 parallel read-only agents), constitutional review of **every** diff before commit, central gating that never trusts a subagent's build claim, personal ownership of the shared-file seams (bindings regen, the ApprovalsQueue mount seam) and of one financial-boundary detail (payroll placement-only verification). Five Sonnet 5 `general-purpose` subagents wrote the code in file-disjoint batches; the monster-file rule was honored — **one** coder owned all of `WorkHub.svelte`, and the approvals-queue seam was pre-defined at a fixed path so two coders composed at gate time without racing.

**Commit map:**
- `0ada7fa` — B1, B2 (People cluster: identity home + payroll placement)
- `dcf3bcf` — B3, B4, B5a, B5d (WorkHub re-wire + backend member allocation & task counts)
- `7e4c79c` — B5b, B5c (approvals decoupled from read-state + persistent queue + 3 read-only list methods)
- `6ccb0e7` — C1, C2, C3 (bank-account CRUD to Settings, book-bank adjustments edit, receipts scale)
- `ecbcc04` — central wailsjs bindings regen (4 new/changed Go methods; CRLF-only no-op binding files reverted)

---

## Phase A — ground-truth verdicts (4 parallel recon agents; every anchor verified — the audit's PH anchors drifted hard)

| # | Question | Verdict |
|---|---|---|
| **A1** | Three identity systems + how employee↔login relate | **CONFIRMED, with the key structural find.** `Employee` (no `UserID`), `User`+`Role` (no `EmployeeID`), `LicenseKey` are three tables joined **only** by `EmployeeAccessLink` (EmployeeID+LicenseKey+UserID) — that join is the real substrate for the identity home. `GenerateLicenseKey` is bound but has **zero** frontend callers (issuance unreachable). Two live deactivation paths confirmed: `SetEmployeeEmploymentState` (inactive, no approval) vs `RequestEmployeeArchive` (approval-gated). **Correction:** the identity/archive work was **Wave 8 P3+P4**, not "Wave 6/P4" as the spec phrased it. |
| **A2** | Permission-key inventory + the `usermanagement` mismatch | **CONFIRMED — and worse than an orphan route.** OSS `App.svelte` registers the screen id `"usermanagement"` (`:121/:453`) but `screenPermissions` keyed it `"users"` → `hasScreenPermission("usermanagement")` returned `undefined` → **fell through the "no permission required" branch = a silent RBAC bypass**. Role→perm grants live in **two** hardcoded sources that must move together: DB-side `SeedDefaultRoles` (`app_auth_rbac.go`) + in-memory `rolePermissions` (`license_service.go`). Backend already supports HR-only payroll (`requirePayrollView` checks `payroll:*` first, falls back to `finance:*`). |
| **A3** | WorkHub model map | **CONFIRMED; anchors drifted.** `WorkHub.svelte` is 2256 lines (not ~2600); assignee dropdowns at `:1082`/`:1571` (+ `ContextTaskModal:162`), not the PH `:1101/1228/1792`. `ProjectMember.AllocationPercent` exists but is **hardcoded 100 server-side** (`AddCollaborativeProjectMember`) — wiring it needs a signature change. Task-count badge (`:1325`) reads `projectTasks` which only holds the *selected* project → every other row shows **0**. `activeOnly` hardcoded true (`:220`), no restore. **Customer/POC block does not exist at all** in the create form (deeper than "ungated" — must be built). **Modal-over-modal does NOT reproduce on OSS** (project detail is an inline panel; only one Modal) — reported as PH-only, no fix ported. |
| **A4** | Payroll placement + state machine | **CONFIRMED.** Mounts only inside FinanceHub (`:58`/`:169`), hub gated `finance:view`. `CompensationProfile.EmployeeID` unique FK. Approve→post→pay state machine (`GenerateRun→ApproveRun→PostRun→MarkPaid`) mapped precisely — keep-list, untouched. No "Set up payroll" deep-link exists. |
| **A5** | Approvals inventory + Notifications read-state bug | **CONFIRMED.** Only **delete-approval** and **employee-archive** are notification-routed (subject to the V.4 bug); the rest have their own gated screens. `isPendingDeleteApproval`/`isPendingEmployeeArchiveApproval` both AND `notification.status !== "read"`; the "Mark read" button flips status → Approve/Reject vanish permanently, and neither type has any other surface → the request is stranded. **No aggregator exists** — no "list all pending approvals" method; delete/archive had **no list method at all** (built this wave). Note: `RequestEmployeeArchive` currently auto-approves immediately, so pending archive rows are normally only peer-synced. |
| **A6** | Anchor drift for C-items | **CONFIRMED.** C1 bank-account CRUD lives in BankReconciliationScreen with a Spec-03 "recommended residue" comment; SettingsScreen's `currency` section is the shape to copy. C3 `UpdateBookBankReconciliationAdjustments` bound, zero callers, refuses finalized. C2 `ListCustomerReceipts(200,0)` flat cap; `CustomerMaster` has **no** division field (company-scope impossible without a migration). |

---

## Phase B / C — per-item status

| Item | Status | Notes |
|---|---|---|
| **B1a** Access section in employee detail | **shipped** | New **Access** tab in the employee detail shows each license link, the login `User`+role bound to it, and lets you link/reassign a license, bind an existing login user, or create+bind a new one (`CreateUser`, `users:*`-gated server-side). The separate top-level license composer is folded in — one home. |
| **B1b** Reachable Users/Access surface + key fix | **shipped** | `screenPermissions["users"]` → `usermanagement: "users:view"` — closes the silent bypass. `users:view` is admin+manager on the license side, admin-only on the DB side; **mutations (`users:create/update`) stay admin-only server-side**, so a manager who reaches the screen can view (which their existing server grants already permit) but not mutate — strictly **tighter** than the prior total bypass, no widening. Reachable via an admin-gated "Users & Access →" entry in PeopleHub. **Conflict-tools decision (recorded, not relocated):** opportunity-edit-conflict + activity monitoring stay in UserManagementScreen — relocating them exceeded safe file scope; now that the route is gated/reachable it's a naming nicety, not an orphan-route defect. |
| **B1c** Onboarding one continuous flow | **shipped** | Composer now collects email (inline-validated on blur), phone, start date, manager; on create it selects the new employee and scrolls the detail into view — no below-the-fold stranding. |
| **B1d** Job-shaped profile | **shipped** | Detail restructured into **Profile / Work / Access** tabs. Verified no sales metrics leak into the HR editor on OSS (the Go struct carries the fields but no screen renders them — already compliant). |
| **B1e** Archive is the only deactivation | **shipped** | Removed the `inactive` status option; the status vocabulary is now work-state only (active/on_leave/probation/contract) and the `employment_status` fallback no longer emits `inactive`. `SetEmployeeEmploymentState` stays bound for reactivation/work-state only; the sole deactivation path is the approval-gated `requestEmployeeArchive` (reason required, reversible, history retained — untouched). |
| **B2** Payroll lives with People | **shipped** | New People **Payroll** tab gated `payroll:view` (an HR-permissioned user runs payroll from People without `finance:view`); FinanceHub's tab left in place (costs nothing). "Set up payroll →" on the employee Work tab deep-links into that employee's comp profile (`presetEmployeeID` prop + one placement-only reactive block; **zero** changes to the generate/approve/post/pay state machine). |
| **B3.1** Members drive assignees | **shipped** | Task composer, task-detail reassign, and `ContextTaskModal` scope the assignee list to the in-context project's members; fall back to all-employees only when there is no project context (so cross-context task creation still works). Adding/removing a member visibly changes who can be assigned. |
| **B3.2** Per-person role + allocation | **shipped** | `AddCollaborativeProjectMember(projectID, employeeID, role, allocationPercent float64)` — the hardcoded `100` replaced; allocation persisted on create + update. UI: per-member editable role+allocation rows with individual Save (batch free-text role retired). |
| **B3.3** Per-project counts / badge fix | **shipped** | New `GetProjectTaskCounts() (map[string]int, error)` (single GROUP BY over `task_items`, `projects:view`-gated) — every project row now shows its **own** real count; the always-0 badge is fixed without loading task bodies. |
| **B3.4** One roster interaction | **shipped (no-op)** | Confirmed only the Team Board roster exists on OSS and it's already single-semantic (click=filter, drag=accelerator). No two-hidden-semantics defect to fix. |
| **B4.1** Start-project handoff | **shipped** | "Start Project" on Opportunity and Order preseeds customer/POC/lineage via a new `pendingProjectHandoff` store and navigates to Work (pattern #1). |
| **B4.2** Type-gated create form | **shipped** | Built the customer/POC composer block (absent on OSS) and gated it on `projectType === 'customer'`; POC email format-validated; lineage flows through `CreateCollaborativeProject` (struct already supported it — no backend change). |
| **B4.3** Post-create member step | **shipped** | After create the composer scrolls to and pulses the Project Members step — recoverable, not a dead-end. |
| **B5a** My Work upgrade | **shipped** | Overdue-first ordering + explicit overdue bucket + search/focus filters + completed-work toggle (`listMyTasks(true)` + client filter). Team Board not weakened. |
| **B5b** Approvals decoupled from read-state | **shipped** | `isPending*` now gate on the request's **real pending state** (membership in pending-id Sets sourced from the new list methods), the `status !== "read"` condition removed entirely. Reading a notification can no longer strand an approval. Existing server-resolved-reviewer + `confirm.askForReason` flow untouched. |
| **B5c** Persistent approvals queue | **shipped** | New `ApprovalsQueueScreen` (Article V.2 Task class) — every pending delete/employee-archive approval with owner + relative age, sorted by consequence then age, inline approve/reject (server-resolved reviewer), an "Open" deep-link that highlights the source notification, admin-gated, full loading/error/empty states. Mounted as a WorkHub **Approvals** tab. Notifications keep announcing; the queue is the durable home. |
| **B5d** Archived-projects view + restore | **shipped** | "Show Archived" toggle (`listProjects(false)`), Restore via `UpdateCollaborativeProject(id,{status:'active'})`; archive/shelve now route through canonical `confirm.askForReason` with a **required** reason (was optional free text). |
| **C1** Bank-account CRUD → Settings | **shipped (full relocation)** | Entire CRUD moved to a new SettingsScreen "Bank Accounts" section (`finance:create`-gated, section deep-link added). BankReconciliationScreen keeps only its read-only `GetActiveBankAccounts` picker + a "manage accounts" link (global `navigateToScreen` event → Settings §accounts). |
| **C2** Receipts workspace polish | **shipped** | 200-row flat cap replaced with real pagination (Load-more, `RECEIPT_PAGE_SIZE=50`, mirrors the payments block) — no silent truncation. Customer picker now typeahead (reused the in-file invoice-search pattern). **No company-scope** — `CustomerMaster` has no division field; reported honestly, stamped-division behavior unchanged. |
| **C3** Book-bank adjustments edit | **shipped** | DIT / outstanding-cheque inputs added to the Edit Adjustments modal, wired to `UpdateBookBankReconciliationAdjustments`; the existing `!is_reconciled` gate + server `IsReconciled` refusal keep finalized recs immutable. |

**Nothing skipped.** The only "did not reproduce" is the WorkHub modal-over-modal item (A3): it is structurally PH-only (OSS project detail is an inline panel, not a modal) — reported, not faked.

---

## Decisions taken (orchestrator)

1. **Permission-key fix by realignment, not by inventing a key.** Chose `usermanagement: "users:view"` — a permission the server already enforces on `ListUsers/GetUser/ListRoles` — over the dead `users:manage`. This makes the existing gate apply to the route; because the screen's writes stay `users:create/update` (admin-only), the net effect is strictly tighter than the prior bypass. (The coder's inline comment says "admin-only"; precisely, `users:view` is also held by `manager` on the license side — view-only, so the security outcome is unchanged. Noted for accuracy.)
2. **Approvals queue sourced from new read-only list methods, not from notifications.** To make the queue "live until done" and decouple approve/reject from read-state, I had the coder add three pure-read, permission-gated list methods (`deletion.Service.List`, `ListDeleteApprovalRequests`, `ListEmployeeArchiveRequests`) — no aggregator existed. Zero new mutations, zero permission widening; the auto-approve archive mechanic is untouched.
3. **Payroll is placement-only.** Verified the PayrollScreen diff is a preset prop + one reactive block (21 insertions), with no `GenerateRun/ApproveRun/PostRun/MarkPaid` logic touched — the state machine is byte-for-byte behaviorally intact, per the keep-list and the financial-semantics stop-and-report rule (zero authorizations this wave).
4. **Bindings regenerated centrally** via `wails generate module`; the 7 CRLF-only no-op binding files were reverted so the commit carries only the 4 real method changes (`App.d.ts/.js`, `SyncServiceBinding.d.ts/.js`).
5. **ApprovalsQueue seam pre-defined.** Fixed the screen path (`frontend/src/lib/screens/ApprovalsQueueScreen.svelte`, `embedded` prop) up front so the WorkHub coder mounted exactly what the approvals coder authored — they ran in parallel and composed cleanly at gate.

## Constitution deviations

- **Article VI (tokens) — two pre-existing-style hex additions, flagged not blocked (same class the owner ratified in Spec-01/02).** (a) `ApprovalsQueueScreen` shipped two bare consequence-badge hex values — **I corrected them** in review to the ratified `var(--text-danger,#…)` / `var(--color-warning,#…)` form. (b) `PeopleHub` adds one danger accent `#dc2626` **inside `color-mix()` blended with `var()` tokens**, consistent with the file's existing danger idiom (file is ~84% tokenized). (c) `SettingsScreen`'s C1 section uses the file's established raw-HTML + class palette (that screen has 102 pre-existing hex and imports no design-system components; a lone canonical component there would visually diverge) — the Coder matched convention. (b)+(c) are consistency-with-file, requested for ratification, no lone-hex regression introduced.
- **No financial-semantics changes.** Payroll math/transitions/refs untouched; no rounding/posting/tax changes; zero authorizations. No secrets, no real client data, layer model respected (UI → bindings → services; new business logic only in Go services).

## Keep-list attestation (§4.3 People + §4.4 Work)

- **Payroll status-driven buttons / state machine:** preserved — placement/nav wiring only, zero state-machine edits. ✅
- **Archive safety contract** (reason required, reversible, history retained) + **P4 approval ride:** preserved; `employee_archive_service.go` mutation logic untouched (only a read-only List added). ✅
- **Directory search + Active/Archive/All; auto `employee_code`; payroll inline help; company/division scoping:** preserved (People Payroll tab got a matching company toggle). ✅
- **Race guards** (`loadRequestSeq`/`projectContextRequestSeq`/`taskDetailRequestSeq`/`assignmentLoadToken`): preserved verbatim. ✅
- **ContextTaskModal context-passing** (customer/opportunity/order prefill): extended (members scoping) not removed. ✅
- **Notifications "Open task" deep-link:** preserved; the new "Open" handoff is additive. ✅
- **Task-delete two-press + block-reason; Team Board lanes/filters + drag accelerator; optimistic snapshot caching:** preserved. ✅
- **Bank-recon Finalize/Reopen gating + read-only picker; finalized book-bank immutability; single AR receipt path:** preserved (C1 kept the picker; C3 respects the finalized lock; C2 didn't alter the receipt model). ✅

## Known residue / follow-ups

- **`GenerateLicenseKey`/`GenerateBatchLicenseKeys` remain UI-unreachable** — the Access tab links/binds existing keys; key *issuance* is still not surfaced (out of this wave's scope; owner call whether to add an admin issuance UI).
- **Conflict-tools placement** — opportunity-edit-conflict resolution + activity monitoring still live in UserManagementScreen (now reachable/gated). A later slice could move conflict resolution to a sales-admin home.
- **Employee-archive queue is usually empty in practice** — `RequestEmployeeArchive` auto-approves on the happy path, so pending archive rows appear only via peer sync. The queue + list method are built and correct; they just rarely have archive rows to show on a single instance. (Delete-approvals are the queue's live content.)
- **Receipts customer picker not company-scoped** — `CustomerMaster` has no division field; unchanged pending a schema decision.
- **CRLF** line endings on many `.go` files remain pre-existing (repo-wide, owner-deferred normalization).

## Open questions for the owner

1. **B1b permission choice** — confirm `usermanagement` gated on **`users:view`** (view for admin+manager, mutations admin-only) is the intended access level, or should the whole screen be admin-only (`users:create`/a dedicated key)?
2. **Conflict-tools home** — leave opportunity-edit-conflict + activity monitoring in the (now-gated) UserManagementScreen, or schedule a move to sales-admin in Wave 9.5?
3. **License issuance UI** — surface `GenerateLicenseKey`/batch generation in the admin Access surface, or keep issuance out-of-app?
4. **Article VI hex** — ratify the two consistency-with-file hex cases (PeopleHub `color-mix` danger accent; SettingsScreen palette), consistent with the Spec-01/02 rulings?
5. **My Work / allocation semantics** — allocation is now captured per member (0–100); confirm you want it purely informational for now (no over-allocation warning / capacity enforcement this wave).

---

*Definition of done: Phase A recorded ✅ · B1–B5 + C1–C3 shipped (none skipped; one PH-only item reported) ✅ · gates green on the final commit ✅ · report written ✅. Branch left local for owner review — no merge, no push, no tag.*
