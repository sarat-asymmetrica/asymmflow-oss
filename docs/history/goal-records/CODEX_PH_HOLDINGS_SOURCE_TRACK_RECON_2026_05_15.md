# Acme Instrumentation Source Track Reconciliation - 2026-05-15

Source repo: `C:\Projects\asymmflow\ph_holdings`
Target repo: `the AsymmFlow repository`

## Pull Status

`ph_holdings` was fetched and pulled from `origin/main`.

Result:

- Branch: `main`
- Current HEAD: `88dac3214257d51e58f644f6418c9917faaf2f03`
- Pull result: already up to date
- Local dirty files before and after pull: `AGENTS.md`, `CLAUDE.md`
- Dirty-file scope: GitNexus instruction block drift only; no product code changes

The latest source-track commit is:

```text
88dac32 Fix offer reference numbering and revisions
```

The preceding large client-feedback/training commit is:

```text
82ce9ee Stabilize final training deployment
```

## Interpretation

The rough/client-feedback track is valuable, but it should not be merged wholesale into `asymmflow`. The refactor repo already contains the key source-track artifacts and has since reorganized many of them into module/service/viewmodel boundaries.

Therefore the right action is:

```text
ph_holdings latest -> acceptance evidence and roadmap constraints
not
ph_holdings latest -> blind file backport
```

## Latest Source-Track Learnings

### Offer revision truth matters

Evidence from `ph_holdings`:

- `88dac32 Fix offer reference numbering and revisions`
- `CostingSheetData.revision_number`
- `GetCostingsByRFQ`
- `SetActiveCostingRevision`
- `CloneCostingAsNewRevision`
- `offerNumberWithRevision`
- `workflow_regression_test.go`

Target implication:

Offer/costing revision chains are not optional polish. Client feedback exposed that quote revision identity and PDF/reference numbering are part of commercial truth. Future Cashflow Evidence, Business Memory, and Sales Pipeline work must preserve revision provenance, active revision selection, and customer-facing reference stability.

### Training/deployment telemetry became a real operator need

Evidence from `ph_holdings`:

- `82ce9ee Stabilize final training deployment`
- `user_activity_monitoring.go`
- `frontend/src/lib/telemetry/activityMonitor.ts`
- `docs/user-guides/*`
- `docs/testing/UI_RUNTIME_MANUAL_ACTION_INDEX_2026_04_26.*`

Target implication:

Operator readiness is not just build/install. Training support, activity visibility, guide coverage, and manual action indexes are part of launch readiness. The refactor roadmap should include a pilot-readiness surface that shows module readiness and training/support signals.

### Destructive actions need approval queues

Evidence from `ph_holdings`:

- `delete_approval_service.go`
- `DeleteApprovalRequest`
- notification review UI paths
- banking delete approval port

Target implication:

The agent-safe mutation boundary should reuse the same idea: irreversible actions become reviewable requests, not immediate side effects. Business Memory link requests, Cashflow posting requests, and future Inventory corrections should all pass through explicit approval/review flows.

### Bank reconciliation had to become allocation-aware

Evidence from `ph_holdings`:

- `0ea406b Finalize deployment hardening and bank reconciliation allocations`
- `68b2a85 Add expense targets to bank reconciliation matching`
- `bank_line_payment_allocations`
- `allocation_type`
- `BankLinePaymentAllocation`

Target implication:

Cashflow Evidence must model many-to-one and one-to-many payment evidence, not simple invoice/payment pairs. It should include allocation state for customer invoices, supplier invoices, expenses, and partial/mixed matches.

### Sync and deployment drift became product risks

Evidence from `ph_holdings`:

- `sync_record_normalization.go`
- `sync_record_normalization_test.go`
- user activity sync restrictions
- Supabase migration additions for allocations and costing revisions
- deployment signoff/checklist docs

Target implication:

Local-first module state needs explicit sync envelopes and conflict policy. Business Memory and Cashflow should not rely on implicit table sync assumptions when they gain durable storage.

### User conflict resolution is a real collaboration seam

Evidence from `ph_holdings`:

- `opportunity_conflict_service.go`
- `opportunity_conflict_service_test.go`
- User Management conflict review controls

Target implication:

Agent and multi-user workflows need conflict review from the beginning. Business Memory review records, Cashflow proposal reviews, and Inventory corrections should carry actor, status, conflict, and resolution metadata.

### UI/backend action inventories are unusually valuable

Evidence from `ph_holdings`:

- `scripts/button_backend_audit.mjs`
- `docs/testing/UI_BACKEND_ACTION_AUDIT_2026_04_17.*`
- `docs/testing/UI_BUTTON_INVENTORY_2026_04_17.*`
- `button_backend_safe_smoke_test.go`

Target implication:

Before launch, AsymmFlow should keep an operator-action inventory that maps visible buttons to backend commands, permissions, and test coverage. This is directly useful for multi-Codex work because it exposes unowned or unsafe actions.

## Confirmed In Refactor Repo

The following source-track artifacts or equivalents are present in `asymmflow`:

- Offer revision methods and UI wiring.
- `user_activity_monitoring.go` and frontend activity telemetry.
- `delete_approval_service.go`.
- `opportunity_conflict_service.go`.
- `sync_record_normalization.go`.
- `bank_line_payment_allocations` and allocation-type model fields.
- `docs/testing/UI_BACKEND_ACTION_AUDIT_2026_04_17.*`.
- `docs/user-guides/FIELD_AND_WORKFLOW_REFERENCE.md`.
- `ocr_rbac_routing_test.go`.
- `button_backend_safe_smoke_test.go`.
- `frontend/src/lib/stores/textScale.ts`.

This confirms the refactor branch already absorbed the rough-track lessons at the file/capability level. The next step is architectural reconciliation, not raw backporting.

## Roadmap Updates To Carry Forward

1. Add a Sales Revision Integrity checkpoint before any launch claim.
2. Add allocation-aware cashflow evidence to the Cashflow Evidence sprint.
3. Treat delete approvals as the model for agent-safe mutation requests.
4. Add module-aware sync/conflict policy before durable Business Memory or Cashflow state is considered launch-ready.
5. Convert user guides and UI/backend action inventories into launch-readiness gates.
6. Keep `ph_holdings` as a rough-source benchmark for client reality checks, while `asymmflow` remains the architecture-forward product line.
