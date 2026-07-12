# Hybrid Features Handoff — 2026-04-05

## Executive Summary

This session series introduced a major expansion to AsymmFlow across collaboration, HR, finance, payroll, deployment tooling, and supporting test coverage.

The work was intentionally split into two architectural lanes:

- `Collaborative lane`:
  - online-first behavior with local cache
  - tasks
  - notifications
  - projects
  - employee directory / People Ops
  - deployment readiness tooling
- `Finance lane`:
  - local-first SQLite-backed accounting behavior
  - expenses
  - recurring expenses
  - payroll
  - accounting and cashflow integration

The user identity model was also expanded without breaking the existing license-key architecture:

- `license key -> device -> employee access link -> employee`
- `user` remains optional and separate for RBAC/auth compatibility

The result is a hybrid operating model:

- collaborative workflows can move quickly across employee systems
- finance remains grounded in local accounting truth
- deployment/admin tooling now exists to manage rollout and recovery

This document is the source-of-truth handoff for the changes delivered through Phases 1 through 8 and the current testing/deployment state.

---

## Outcome Delivered

### What is now in the product

- New `Work` hub for collaborative tasks, team management, project-linked work, and activity history
- New `People` hub for employee directory, reporting relationships, access linking, and contribution history
- New `Notifications` center with unread counts and persistent read state
- Expanded `Finance Hub` with:
  - `Expenses`
  - `Recurring`
  - `Approvals`
  - `Compensation`
  - `Payroll Runs`
  - `Payout Tracking`
- New `Deployment` hub for pilot readiness, checklisting, support export, and rollout recovery
- Contextual task creation from customers, opportunities, and orders
- New expense ledger and recurring expense flow
- New payroll generation, approval, posting, and payout tracking flow
- Migration/backfill/recovery support for legacy follow-up tasks and collaborative pending operations
- Expanded backend regression coverage across the new hybrid feature set

### What is active in the local runtime environment

- The live desktop database has been migrated for the new collaboration, HR, expense, payroll, and rollout features
- Jordan Lee exists as an employee in the runtime DB
- Jordan Lee admin access is already active on this machine
- The actual runtime admin key is intentionally not duplicated in this repo document for safety, but it has already been provided separately during the session

---

## Architecture Decisions

### 1. Two-lane architecture

#### Collaborative lane

Purpose:

- near-immediate cross-device collaboration when online
- local cache for resilience
- employee-specific work surfaces

Key domains:

- `employees`
- `employee_access_links`
- `projects`
- `project_members`
- `task_items`
- `task_comments`
- `task_activity`
- `notifications`
- `notification_receipts`
- `collaborative_pending_operations`

Core backend files:

- [collaboration_service.go](/Users/developer/projects/asymmflow/collaboration_service.go)
- [collaboration_sync.go](/Users/developer/projects/asymmflow/collaboration_sync.go)
- [phase7_rollout.go](/Users/developer/projects/asymmflow/phase7_rollout.go)

#### Finance lane

Purpose:

- preserve accounting correctness and offline operability
- integrate payroll and expenses into cashflow and posting flows

Key domains:

- `expense_categories`
- `expense_vendors`
- `expense_entries`
- `expense_allocations`
- `recurring_expenses`
- `expense_attachments`
- `expense_approvals`
- `employee_compensation_profiles`
- `payroll_periods`
- `payroll_runs`
- `payroll_run_items`
- `payroll_components`
- `payroll_payouts`

Core backend files:

- [expense_service.go](/Users/developer/projects/asymmflow/expense_service.go)
- [payroll_service.go](/Users/developer/projects/asymmflow/payroll_service.go)
- [finance_reporting_service.go](/Users/developer/projects/asymmflow/finance_reporting_service.go)

### 2. Identity model

The app was not restructured to merge employees and users into a single record.

Instead:

- `license_keys` remain the device/seat activation mechanism
- `employees` are now the People Ops identity layer
- `users` remain optional for auth/RBAC compatibility
- `employee_access_links` connect the runtime identity chain together

This preserved the existing licensing model while enabling:

- employee-specific work queues
- task assignment by employee
- HR reporting by employee
- deployment auditing of employee/device/license readiness

### 3. Dedicated collaboration sync path

The system does not full-sync the entire ERP database whenever a task changes.

Instead:

- committed collaboration actions queue into `collaborative_pending_operations`
- those actions use a priority collaboration sync path
- local state remains durable
- cross-device visibility improves without abusing the heavier ERP sync loop

---

## Phase-by-Phase Delivery Summary

## Phase 1 — Identity and Collaboration Foundation

Implemented:

- employee model
- employee/license linking
- project model foundation
- notification model foundation
- unread count plumbing
- shell routes and nav scaffolding for `Work`, `People`, and `Notifications`
- collaborative client abstraction on the frontend

Important files:

- [collaboration_service.go](/Users/developer/projects/asymmflow/collaboration_service.go)
- [app.go](/Users/developer/projects/asymmflow/app.go)
- [license_service.go](/Users/developer/projects/asymmflow/license_service.go)
- [frontend/src/lib/api/collaboration.ts](/Users/developer/projects/asymmflow/frontend/src/lib/api/collaboration.ts)
- [frontend/src/App.svelte](/Users/developer/projects/asymmflow/frontend/src/App.svelte)
- [frontend/src/lib/components/ui/EnterpriseSidebar.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/components/ui/EnterpriseSidebar.svelte)

## Phase 2 — Task MVP and Immediate Cross-Device Delivery

Implemented:

- task creation and assignment
- task comments and task activity
- notification generation on collaborative actions
- dedicated collaboration sync transport
- collaboration polling/pull loop
- local cache updates on task actions

Important files:

- [collaboration_service.go](/Users/developer/projects/asymmflow/collaboration_service.go)
- [collaboration_sync.go](/Users/developer/projects/asymmflow/collaboration_sync.go)
- [db_manager.go](/Users/developer/projects/asymmflow/db_manager.go)
- [db_sync_service.go](/Users/developer/projects/asymmflow/db_sync_service.go)
- [frontend/src/lib/screens/WorkHub.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/WorkHub.svelte)
- [frontend/src/lib/screens/NotificationsScreen.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/NotificationsScreen.svelte)

## Phase 3 — Projects and Team Work Management

Implemented:

- project members
- project-linked work surfaces
- team board
- task detail with activity trail
- reassignment
- due-date updates
- project stats and project activity
- contextual task creation from CRM and operations surfaces

Important files:

- [collaboration_service.go](/Users/developer/projects/asymmflow/collaboration_service.go)
- [frontend/src/lib/screens/WorkHub.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/WorkHub.svelte)
- [frontend/src/lib/components/ContextTaskModal.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/components/ContextTaskModal.svelte)
- [frontend/src/lib/screens/CustomerDetailView.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/CustomerDetailView.svelte)
- [frontend/src/lib/screens/OpportunitiesScreen.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/OpportunitiesScreen.svelte)
- [frontend/src/lib/screens/OrdersScreen.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/OrdersScreen.svelte)

## Phase 4 — People Ops HR Core

Implemented:

- employee directory
- employee profile editing
- org/reporting relationships
- contribution summaries driven by real task/project history
- access-link management
- employee state and manager/admin flows

Important files:

- [collaboration_service.go](/Users/developer/projects/asymmflow/collaboration_service.go)
- [frontend/src/lib/screens/PeopleHub.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/PeopleHub.svelte)

## Phase 5 — Accounting-Grade Expenses

Implemented:

- expense categories and vendors
- expense lifecycle:
  - draft
  - submitted
  - approved
  - rejected
  - posted
  - paid
- recurring expenses
- approval tracking
- bank-derived expense candidate import
- cashflow integration for unpaid approved expenses and recurring commitments
- finance UI tabs for expenses and recurring flows

Important files:

- [expense_service.go](/Users/developer/projects/asymmflow/expense_service.go)
- [finance_reporting_service.go](/Users/developer/projects/asymmflow/finance_reporting_service.go)
- [frontend/src/lib/api/expenses.ts](/Users/developer/projects/asymmflow/frontend/src/lib/api/expenses.ts)
- [frontend/src/lib/screens/ExpensesScreen.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/ExpensesScreen.svelte)
- [frontend/src/lib/screens/FinanceHub.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/FinanceHub.svelte)

## Phase 6 — Payroll

Implemented:

- employee compensation profiles
- payroll periods
- payroll run generation
- payroll approvals
- payroll posting
- payout tracking
- cashflow integration for payroll liabilities

Important files:

- [payroll_service.go](/Users/developer/projects/asymmflow/payroll_service.go)
- [finance_reporting_service.go](/Users/developer/projects/asymmflow/finance_reporting_service.go)
- [frontend/src/lib/api/payroll.ts](/Users/developer/projects/asymmflow/frontend/src/lib/api/payroll.ts)
- [frontend/src/lib/screens/PayrollScreen.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/PayrollScreen.svelte)

## Phase 7 — Hardening, Migration, and Rollout Safety

Implemented:

- legacy follow-up backfill into collaborative tasks
- collaborative pending-operation normalization and deduping
- rollout status tracking
- support bundle export
- payout reconciliation support paths
- admin recovery actions for failed or dead-lettered collaborative operations

Important files:

- [phase7_rollout.go](/Users/developer/projects/asymmflow/phase7_rollout.go)
- [phase7_rollout_test.go](/Users/developer/projects/asymmflow/phase7_rollout_test.go)
- [bank_transaction_matcher.go](/Users/developer/projects/asymmflow/bank_transaction_matcher.go)
- [frontend/src/lib/screens/SettingsScreen.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/SettingsScreen.svelte)
- [frontend/src/lib/screens/BankReconciliationScreen.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/BankReconciliationScreen.svelte)

## Phase 8 — Deployment, Rollout, and Adoption Tooling

Implemented:

- pilot readiness summary
- employee/license/device/user readiness rows
- rollout issue filtering
- pilot checklist persistence
- support bundle export
- sign-off report export
- dedicated `Deployment` workspace

Important files:

- [phase7_rollout.go](/Users/developer/projects/asymmflow/phase7_rollout.go)
- [frontend/src/lib/screens/DeploymentHub.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/DeploymentHub.svelte)
- [frontend/src/App.svelte](/Users/developer/projects/asymmflow/frontend/src/App.svelte)

---

## Frontend Surfaces Added or Expanded

### New or heavily expanded screens

- [frontend/src/lib/screens/WorkHub.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/WorkHub.svelte)
- [frontend/src/lib/screens/PeopleHub.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/PeopleHub.svelte)
- [frontend/src/lib/screens/NotificationsScreen.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/NotificationsScreen.svelte)
- [frontend/src/lib/screens/ExpensesScreen.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/ExpensesScreen.svelte)
- [frontend/src/lib/screens/PayrollScreen.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/PayrollScreen.svelte)
- [frontend/src/lib/screens/DeploymentHub.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/screens/DeploymentHub.svelte)

### Key shell/navigation files updated

- [frontend/src/App.svelte](/Users/developer/projects/asymmflow/frontend/src/App.svelte)
- [frontend/src/lib/components/ui/EnterpriseSidebar.svelte](/Users/developer/projects/asymmflow/frontend/src/lib/components/ui/EnterpriseSidebar.svelte)
- [frontend/wailsjs/go/main/App.js](/Users/developer/projects/asymmflow/frontend/wailsjs/go/main/App.js)
- [frontend/wailsjs/go/main/App.d.ts](/Users/developer/projects/asymmflow/frontend/wailsjs/go/main/App.d.ts)

---

## Test Coverage Added and Updated

## New comprehensive hybrid regression coverage

File:

- [hybrid_feature_flow_test.go](/Users/developer/projects/asymmflow/hybrid_feature_flow_test.go)

Key tests added:

- [hybrid_feature_flow_test.go#L112](/Users/developer/projects/asymmflow/hybrid_feature_flow_test.go#L112)
  - current employee resolution from activated license
- [hybrid_feature_flow_test.go#L135](/Users/developer/projects/asymmflow/hybrid_feature_flow_test.go#L135)
  - task creation, notification lifecycle, read flow
- [hybrid_feature_flow_test.go#L228](/Users/developer/projects/asymmflow/hybrid_feature_flow_test.go#L228)
  - employee/license reassignment behavior
- [hybrid_feature_flow_test.go#L265](/Users/developer/projects/asymmflow/hybrid_feature_flow_test.go#L265)
  - project/contribution summary behavior
- [hybrid_feature_flow_test.go#L315](/Users/developer/projects/asymmflow/hybrid_feature_flow_test.go#L315)
  - expense lifecycle and cashflow projection
- [hybrid_feature_flow_test.go#L403](/Users/developer/projects/asymmflow/hybrid_feature_flow_test.go#L403)
  - payroll lifecycle and cashflow projection
- [hybrid_feature_flow_test.go#L468](/Users/developer/projects/asymmflow/hybrid_feature_flow_test.go#L468)
  - pilot readiness, checklist, support exports, sign-off export

## RBAC coverage update

File:

- [rbac_license_alignment_test.go](/Users/developer/projects/asymmflow/rbac_license_alignment_test.go)

Key test added:

- [rbac_license_alignment_test.go#L218](/Users/developer/projects/asymmflow/rbac_license_alignment_test.go#L218)
  - hybrid feature permission alignment by role

## Existing rollout coverage retained

File:

- [phase7_rollout_test.go](/Users/developer/projects/asymmflow/phase7_rollout_test.go)

Purpose:

- verifies legacy follow-up backfill
- verifies collaborative pending-operation dedupe behavior

---

## Verification Status

The following verification passes were completed at the end of the session:

### Backend

Command:

```bash
env GOCACHE=/Users/developer/projects/asymmflow/.gocache go test ./...
```

Status:

- passed

Note:

- local `GOCACHE` was used because the sandbox environment does not allow writes to the default user cache path

### Frontend

Command:

```bash
npm run build
```

Status:

- passed

Notes:

- build succeeded
- there are still existing Svelte accessibility/style warnings elsewhere in the repo
- these warnings are currently non-blocking

### Desktop packaging

Command:

```bash
wails build
```

Status:

- passed

Output:

- built [build/bin/AsymmFlow.app](/Users/developer/projects/asymmflow/build/bin/AsymmFlow.app)

---

## Runtime Testing State for Jordan Lee

Confirmed in the live desktop runtime DB:

- Jordan Lee exists as an employee
- Jordan Lee admin access is active on this machine
- the runtime DB contains the new hybrid feature tables

This means Jordan Lee can test:

- Work
- People
- Notifications
- Deployment
- Expenses
- Payroll

without additional feature enabling work in the runtime environment.

---

## Known Remaining Gaps and Non-Blocking Issues

These are the most important unresolved items after feature delivery.

### Collaboration and hardening

- richer conflict-handling UX for simultaneous task edits
- clearer offline queue visibility for admins and end users
- more explicit retry/recovery affordances around collaborative sync failures

### Finance and reconciliation

- deeper UI for expense allocations
- richer expense attachment management UI
- more polished payroll and expense matching inside bank reconciliation

### HR and reporting

- more sophisticated performance/trend reporting
- department-level reporting views
- more management-facing summary surfaces

### UI and accessibility

- repo-wide Svelte accessibility warnings remain in multiple older screens/components
- several surfaces still need UI polish now that the functional architecture is in place
- chunk-size warning in the frontend build suggests eventual code-splitting work would be useful

### General production polish

- rollout messaging and support exports can be refined further
- some screens would benefit from stronger empty states, validation copy, and spacing polish

---

## Recommended Next Session Focus

Next session should not start another large feature stream.

The most logical next pass is:

1. hardening
- collaborative conflict handling
- queue visibility and retry behavior
- rollout/admin recovery polish

2. UI issue resolution
- accessibility warnings in touched screens first
- fit-and-finish on Work, People, Expenses, Payroll, Deployment
- consistency passes across filters, empty states, badges, modals, and tables

3. finance polish
- bank reconciliation UX improvements for payroll/expense matching
- expense attachment/allocation UI improvements

4. production validation
- pilot-style manual walkthrough on the live app
- multi-device behavior checks
- regression sweeps on the new modules after UI changes

---

## Suggested Manual Test Order

For the next session, use this test flow:

1. `People`
- verify Jordan Lee employee profile
- verify access linking and org info

2. `Work`
- create task
- assign task
- add comment
- update due date
- change status
- reassign task

3. `Notifications`
- verify unread badge and read flow

4. `Customer / Opportunity / Orders`
- create contextual tasks from each surface

5. `Finance Hub`
- expense draft to paid flow
- recurring expense creation and generation
- compensation setup
- payroll period/run approval/posting/payout flow

6. `Deployment`
- readiness review
- checklist update
- support/sign-off export

---

## Final Status

The hybrid collaboration, HR, expenses, payroll, rollout tooling, and expanded test coverage are now in place and build successfully.

The product is ready for the next session to focus on:

- hardening
- UI issue resolution
- reconciliation polish
- production-style manual validation
