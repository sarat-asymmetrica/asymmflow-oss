# Wave 9 Spec 04 — People & Projects Re-model (The Big Re-wire)

**Mission:** Wave 9.4 from `FABLE_WAVE9_UIUX_AUDIT.md` §5 (employee identity home, payroll placement, members=assignees, project handoffs, approvals queue) + three residue items ruled at the Spec-03 gate.
**Repo:** `asymmflow-oss`. **Branch:** `feat/fable-wave9-4-people-projects` off `main`. Do not merge or push; leave for owner review.
**Authority documents, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` → `FABLE_WAVE9_UIUX_AUDIT.md` → this spec.
**Prior art:** read all three prior reports (`FABLE_WAVE9_SPEC_0{1,2,3}_REPORT.md`). Spec-03 gate rulings now in force: two-P&L labels stay as built (actual-source labeling ratified); Proforma→Sent conversion ratified; receipt model = the single AR money-in path ratified.

## 0. Read before anything

1. `CLAUDE.md` — **security posture is load-bearing this wave:** you will touch roles/permissions/identity. Server-side RBAC (`requirePermission`) must be preserved; auth-adjacent changes get tests. Financial semantics: stop-and-report (no authorizations this wave).
2. `DESIGN_CONSTITUTION.md` — Articles I.5 (job-shaped hierarchy), II (patterns #1/#2/#8), III, **V (alarm philosophy — the approvals queue IS Article V.2's Task class made real)**, VI.
3. `FABLE_WAVE9_UIUX_AUDIT.md` §4.3 (People) + §4.4 (Work) — findings, **binding keep-lists**, and each domain's "biggest single win" (they are B1 and B3); §3 T2/T10/T11; §5 Wave 9.4.
4. All three prior reports.

**Data-sensitivity invariant:** `../ph_holdings` readable for reference; real client names/figures never enter this repo.

## 1. Operating model

Identical to Spec-02/03: **Opus 4.8 orchestrator as senior designer/tech-lead** (Phase-A recon work orders → coder batches → constitutional review of every diff → central gating → personal ownership of shared-file seams and bindings regen); **Sonnet 5 subagents** (`model: "sonnet"`, `subagent_type: "general-purpose"`) write the code.

**Lessons inherited (do not relearn):**
- Audit anchors cite deployed PH and drift hard — verify every anchor before coding.
- `git checkout -- frontend/dist/index.html` after any build mutates the placeholder.
- Gate baseline: `npx vite build` clean · `npx svelte-check` **0 errors / 14 warnings** · `go test ./...` green (run `-count=1` at least once). Net-new = failure.
- Identity resolves server-side where attribution/SoD matters; UI displays the operator and blocks when unknown.
- **File-overlap discipline (critical this wave):** `WorkHub.svelte` is a ~2600-line multi-responsibility monster and `PeopleHub.svelte` is its sibling. Batches touching the same monster must be SEQUENCED, never parallel. Opportunistic decomposition (extracting components while working) is allowed and encouraged within slices — wholesale rewrite is not.

## 2. Phase A — recon (read-only, do first; verdicts in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | Map the three identity systems on OSS: HR employees, login users/roles, license keys — models, bindings, create/update paths, and what Wave 6/P4 already changed. How does an employee relate (or not) to a login user today? | B1 |
| A2 | Permission-key inventory: what keys does the sidebar/nav emit, what does App.svelte check, does the orphaned `usermanagement` route + `users` key mismatch exist on OSS? How are role→permission grants stored, and what does adding an HR/payroll permission require (frontend + `requirePermission` side)? | B1, B2 |
| A3 | WorkHub model map: project members vs task assignees (tables/fields), does `allocation_percent` exist in the model, the per-project task-count badge bug, archived projects (`activeOnly` handling), modal-over-modal sites. | B3, B4, B5 |
| A4 | Payroll today: where mounted, what permission gates it, how comp profiles link to employees, the approve→post→pay state machine (keep-list — must survive untouched). | B2 |
| A5 | Approvals inventory: which approval request types exist (employee archive, expenses, delete-approvals, anything else); the NotificationsScreen read-state bug on OSS (Approve/Reject rendered only while unread?); what a persistent approvals queue can hang off. | B5 |
| A6 | Verify anchors for every B/C item (incl. what Spec-01 already fixed — e.g. dashboard task rows); note drift. | all |

## 3. Phase B — the re-model

**B1 — Employee record = the single home for a person (T10 — the People domain's biggest win).**
(a) Embed login-account/role and license assignment into the employee detail (an Access tab/section): see at a glance whether this person can log in, as what role, with which license; grant/change from here. The separate top-level license composer folds in.
(b) A reachable Users/Access admin surface: fix the orphaned `usermanagement` route + permission-key mismatch if A2 confirms it on OSS; whatever admin-only management remains (roles overview, conflict tools) gets one legitimate, permission-gated home. If the audit's note holds (opportunity edit-conflict tools living here), decide their right home and record it.
(c) Onboarding as one continuous flow (patterns #1/#8): essentials in the create composer (name, department, title + email, phone, start date, manager); on create, focus moves to the new profile ("add contact & start details" handoff — no below-the-fold stranding); email format-validated inline.
(d) The profile is job-shaped (T11): tabs/sections Profile / Work / Access; sales metrics move to a contribution overview, never above the HR editor.
(e) **Archive is the only deactivation** (Article III.1): the status field's vocabulary becomes work-state only (e.g. active / on_leave / probation / contract); `inactive`-via-status is removed as a second deactivation path. Preserve the archive safety contract (reason required, reversible, history retained) and the P4 approval ride.
**AC:** one place answers "who is this person and what can they do in the app"; onboarding never strands a half-finished profile; RBAC checks remain server-side (tests where touched); exactly one deactivation path.

**B2 — Payroll lives with People.** Surface payroll in the People hub gated on an HR/payroll permission (A2 tells you the mechanics; Finance visibility may remain if it costs nothing, but People is the home). "Set up payroll" deep-link from the employee record into that employee's comp profile (pattern #1). The approve→post→pay state machine and required refs are keep-list — placement changes only.
**AC:** an HR-permissioned user finds and runs payroll from People without `finance:view`; an employee's payroll setup is one click from their record; the payroll state machine is byte-for-byte behaviorally intact.

**B3 — Members = assignees (the Work domain's biggest win).** One concept: project membership drives every assignee dropdown (task create/edit, board, batch ops) — no more all-employees lists on project-scoped work. Per-person role and allocation on membership (wire `allocation_percent` if A3 confirms it; else per-person role only, report). Per-project workload rollups derived from membership+tasks; fix the task-count badge reading the selected project's tasks for every row. One roster interaction (kill the two hidden per-tab semantics; keep drag as accelerator only).
**AC:** adding/removing a member visibly changes who can be assigned; every project row shows ITS OWN counts; a member has a role/allocation, not one batch free-text.

**B4 — Projects start from the work that spawned them.** "Start project" action on Opportunity and Order (pattern #1 handoff) preseeding `opportunity_id`/`order_id`, customer/POC, and name; the create form gates its customer/POC block on `projectType === customer` (T-gated form); after create, the project opens on the member step (recoverable, not a dead-end). POC email validated.
**AC:** a won opportunity or live order becomes a project in one action with lineage populated; internal projects never show customer fields; create always lands somewhere useful.

**B5 — The worker's surface + the approvals queue (Article V made real).**
(a) My Work: overdue-first ordering + an overdue bucket, status/focus filters, completed-work toggle — at least as capable as the Team Board.
(b) **Approvals decoupled from read-state:** Approve/Reject render based on the request's pending state, never on unread status (the canonical Article V.4 violation — reading may never strand an actionable item). 
(c) **A persistent approvals queue** (Article V.2 Task class): one surface listing every pending approval (A5's inventory) with owner + age, living until *done*, deep-linking into each request. Notifications keep announcing; the queue is the durable home.
(d) Archived-projects view + restore path (`activeOnly` toggle); audit reason required on archive/shelve, with Shelve-vs-Archive explained or merged.
**AC:** reading a notification never makes an approval unactionable; a manager can answer "what awaits my approval?" in one place; a worker can answer "what's on my plate, worst first?" in one glance; archived projects are findable and restorable.

## 4. Phase C — residue ruled at the Spec-03 gate

**C1 — Bank-account CRUD to Settings/Admin (ruled: full relocation).** Move the account CRUD demoted inside BankReconciliationScreen to its legitimate Settings/Admin home; recon keeps a read-only account picker + a "manage accounts" link (pattern #4).
**AC:** account management has one admin home; the recon screen only reconciles.

**C2 — Receipts workspace polish.** Replace the 200-row cap with pagination or windowed search; customer picker gets typeahead search. Company-scope the picker ONLY if the model supports it (Spec-03 found `CustomerMaster` has no division — if still true, report that honestly and leave the stamped-division behavior as-is).
**AC:** the receipts list scales past 200 without silent truncation; finding a customer among hundreds is typing, not scrolling.

**C3 — Book-bank adjustments edit path.** `UpdateBookBankReconciliationAdjustments` is bound but reachable only via create. Expose an edit affordance on the prove step for un-finalized reconciliations (finalized stays locked — keep-list), or record a reasoned decision to keep create-only.
**AC:** a bookkeeper can correct DIT/cheque figures before finalizing without deleting the rec; finalized recs remain immutable.

### Suggested coder batching (adjust from Phase A; respect the monster-file rule)
- Coder 1: B1 + B2 (PeopleHub + UserManagementScreen + App.svelte routes + payroll placement — the whole People cluster, ONE coder because PeopleHub overlaps)
- Coder 2: B3 + B4 (WorkHub data model re-wire — sequenced within one coder)
- Coder 3: B5 (My Work + NotificationsScreen + approvals queue; WorkHub portions land AFTER Coder 2 — orchestrator sequences the seam)
- Coder 4: C1 + C3 (recon/settings cluster)
- Coder 5: C2 (receipts)
- Orchestrator: shared-file seams (App.svelte routing, sidebar, bindings regen) + constitutional review

## 5. Hard boundaries

- **No Wave 9.5 polish**, no sensory/brand work (owner-reserved wave), no new HR feature domains (visa/permit tracking remains unratified — owner decision).
- **RBAC/auth changes** keep enforcement server-side and add tests; never widen a permission to make UI wiring easier (that's a security regression, not a UX fix).
- **Payroll money math untouched** — placement and navigation only. Financial semantics: stop-and-report (zero authorizations this wave).
- **Keep-lists binding** (§4.3 + §4.4): payroll status-driven buttons; archive safety contract; directory search + Active/Archive/All; auto employee_code; company/division scoping; race guards (`loadRequestSeq`/`assignmentLoadToken`); ContextTaskModal context-passing; Notifications "Open task" deep-link; task-delete two-press + block-reason; Team Board lanes/filters + drag accelerator; optimistic snapshot caching.
- **Explicitly deferred (owner-acknowledged, do NOT do):** GL opening-balance carry-forward (needs a prior-period close model); repo-wide CRLF normalization; WorkHub wholesale decomposition.
- **No merge, no push, no tag.**

## 6. Definition of done + status report

Done = Phase A recorded; B1–B5 + C1–C3 shipped or explicitly skipped with reason; gates green on final commit; report written.

Write `FABLE_WAVE9_SPEC_04_REPORT.md`, commit it, and paste it verbatim as your final message (the established template: Phase A verdicts, per-item status+commits, decisions, constitution deviations, keep-list attestation, residue, owner questions). Severity honesty is law: an accurate red beats a false green.
