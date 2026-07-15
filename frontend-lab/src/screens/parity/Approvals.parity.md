# Parity Ledger — ApprovalsQueueScreen (old) vs Approvals Queue descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | List merges `listDeleteApprovals` + `listEmployeeArchiveApprovals` | **DONE (fetch INTEG-gapped)** | Real bridge names `ListDeleteApprovalRequests('')` + `ListEmployeeArchiveRequests('')` (confirmed real signatures, `delete_approval_service.go`) — empty string returns every status per `pkg/infra/deletion.Service.List`. Fetch is INTEG-gapped per the K4 brief (admin-privileged + employee-PII surface), mirroring Opportunities' two-source gate rather than the K1 ledgers' wired-real fetch. |
| 2 | Approve (row, confirm) | **DONE** | Plain confirm + mock mutation. Real throws naming `ReviewDeleteApprovalRequest`/`ReviewEmployeeArchiveRequest` with `decision="approve"` — both confirmed real, same status vocabulary (`pending`→`approved`) verified verbatim in `pkg/infra/deletion/deletion.go:200-222` and `employee_archive_service_test.go:180-182`. |
| 3 | Reject with reason (row) | **DONE** | Row-aware reason `form` (ROW-AWARE FORMS pattern) — the reviewer note maps to the real `notes` param (`ReviewDeleteApprovalRequest(id, "reject", notes)`); mock accepts but doesn't surface the note as a column (it isn't part of `ApprovalRow`, matching the old screen's own admin-review-note handling being separate from the request's original `reason`). |
| 4 | `embedded` prop — WorkHub hosts this screen as a tab | **DEFER** | Out of scope: the registry only mounts the standalone screen. WorkHub itself is a K5/K6 DEFER per the census (needs a new operational-hub composition primitive) — when it's built, it should re-host this same descriptor rather than duplicate the queue. |
| 5 | RBAC: admin-only via `license_role` check | **EQUIV** | Real `ListDeleteApprovalRequests` already gates server-side (returns `[]` for non-admin sessions, `delete_approval_service.go:106-108`) — the kernel doesn't need a client-side gate to preserve this; it's enforced regardless of what the frontend shows, and this screen's real fetch is INTEG-gapped anyway (#1). |
| 6 | Two-stage employee-archive approval (`RequiredApprovals`, first/second approver) | **DEFER** | `EmployeeArchiveRequest` has `required_approvals`/`first_approved_by`/`second_approved_by` fields for a two-signature workflow; this K4 build treats archive requests as single-decision (pending→approved/rejected) like delete requests, matching what the old screen's `reviewEmployeeArchiveRequest` call site actually does per-click (each click is one decision) — the two-stage bookkeeping itself lives server-side, not in this screen's UI. Not re-modeled here; flagged for whoever wires K5 in case the two-signature state needs its own column. |

## Reading

Both request kinds share one real status/decision vocabulary end to end
(`pending`/`approved`/`rejected`, `"approve"`/`"reject"`) — confirmed
directly against `pkg/infra/deletion/deletion.go` and the employee-archive
test suite, not assumed from naming — so the merge in `ApprovalRow` is
honest, not a shape-coercion. The one thing this K4 build declines to model
is the employee-archive path's two-signature requirement (#6): the real
backend tracks `RequiredApprovals`/first-approver/second-approver, but the
old screen's own click-to-review flow collapses that into "approve" or
"reject" per click just like this descriptor does, so nothing was lost by
staying at that level.
