# WorkHub — parity notes

**Entity:** `work` · **Group:** Operations · **Archetype:** bespoke hub (K4-deferred, TabShell)

Old: `WorkHub.svelte` (1445 lines) + 5 child components + `collaboration.ts`. New: `bridge/work.ts` +
`work-vm.svelte.ts` + `WorkHub.svelte`. Transport confirmed Wails IPC only (SyncServiceBinding + App).
Shape: **TabShell** (My Work / Team Board / Projects / Approvals) + a shared header (active-employee
identity + StatTileGrid: My Open / Team Open / Blocked / Active Projects), tab badges (open counts,
danger tone when blocked).

## Capability census

| # | Old capability | Verdict | Notes |
|---|---|---|---|
| 1 | 4-tab switch + role-derived default | **EQUIV** | TabShell; default `my_work` (no role signal in the lab yet — Owner Q1). |
| 2 | 6 parallel list fetches | **DONE** | All real single-call FETCH (ListMy/TeamCollaborativeTasks, ListCollaborativeProjects, ListEmployeeProfiles, GetProjectTaskCounts, GetCurrentEmployeeContext). |
| 3 | 30s snapshot cache | **DEFER** | Loads fresh. |
| 4/5 | sessionStorage pending-task-open + pendingProjectHandoff cross-screen | **DEFER** | `vm.openTask(id)` exists; K5 router wires it. |
| 6 | Project-membership-scoped assignee lists | **SIMPLIFIED** | Full roster always (Owner Q2). |
| 7 | Task delete "press again" toggle | **FIXED/UNIFIED** | ConfirmDialog (danger), consistent with project delete (Design Constitution III.6). Irreversible stated. |
| 8 | Project archive/shelve/delete — mandatory reason | **DONE + INTEG** | ConfirmDialog `reasonLabel`+`requireReason`; Archive/Shelve/DeleteCollaborativeProject INTEG-gapped (HOT); delete states cascade note. |
| 9 | Project restore | **DONE + INTEG** | `UpdateCollaborativeProject(status:'active')`; no reason (matches old). |
| 10 | **Member batch-add allocation precheck-before-batch** (WARN>100%, cancel writes nothing) | **PRESERVED** | `requestAddMembers()` prechecks ALL via GetEmployeeAllocationSummary, stores pending batch; only `commitAddMembers()` writes. Cancel clears. Verified live vs the 340%-over mock member. |
| 11 | Per-member allocation edit precheck + WARN | **PRESERVED** | Same precheck→WARN→commit shape. |
| 12 | Per-member inline edit (all rows simultaneous) | **EQUIV** | Select-a-row-to-edit (one form below the DataTable) — avoids 50+ simultaneous forms on the perf-monster project. Same precheck path. |
| 13 | Customer/POC block (customer type only) | **DONE** | Conditional render + payload shaping preserved. |
| 14 | Post-create "land on member step" scroll+pulse | **DROP** | Cosmetic; roster is directly below the composer now. |
| 15 | Live sync event listeners (EventsOn) | **DEFER** | No realtime push in the lab; explicit reload after mutations. |
| 16 | Team Board aggregate view | **NET-NEW** | DistributionWidget (tasks-by-status) above the list; renders UNKNOWN_STATUS as a labeled segment. |
| — | Approvals tab | **DONE (embedded)** | `<DocumentLedger descriptor={approvalsDescriptor} embedded />` — the built ledger, unchanged. |

## INTEG / hot-zone ledger (14 mutations, all named)
CreateCollaborativeProject · UpdateCollaborativeProject · **ArchiveCollaborativeProject** (HOT reason) ·
**ShelveCollaborativeProject** (HOT reason) · **DeleteCollaborativeProject** (HOT reason) · AddCollaborativeProjectMember ·
CreateCollaborativeTask · UpdateCollaborativeTask · UpdateCollaborativeTaskStatus · ReassignCollaborativeTask ·
UpdateCollaborativeTaskDueDate · AddCollaborativeTaskComment · DeleteCollaborativeTask · RefreshCollaborativeWorkspace —
each throws `INTEG gap: <Binding> — wires at K5`; mock performs it (interactive lab).

## Orchestrator notes
- Form controls use kernel `k-field`/`k-field-row`/`k-input` classes (L1/L2 clean).
- Live-caught mock bug FIXED: `proj-mega` cycled 55 members through 20 employees → duplicate (project,employee)
  memberships crashed Svelte's keyed `{#each}` + violated the real upsert semantics. Fixed by a 60-employee roster + unique indices.
- Adversarial mock (60 employees, 12 projects, ~285 tasks): unassigned/no-project/no-due tasks, 430-day-overdue task,
  UNKNOWN_STATUS on a task + a project, empty project, 55-member/220-task perf project, exactly-100% + 340%-over members,
  blocked task with empty reason, RTL customer name + no-domain POC email. Synthetic identity only.

## Owner questions (ratified defaults applied)
1. Default tab fixed to `my_work` (no role signal until K5 auth). Acceptable?
2. Assignee lists simplified to full roster (not project-scoped). Restore at K5 or keep?
3. Blocked-reason: inline-refusal (VM refuses+surfaces error if empty) vs a `requireReason` ConfirmDialog for consistency?

## Kernel gap (minor)
- No multi-select control (the "Add Members" checklist is hand-rolled from `k-field`/`k-field-row`). Candidate if a 2nd screen wants it.
