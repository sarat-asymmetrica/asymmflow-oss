# Fable Wave 6 — Progress Audit

Honest self-audit of Wave 6 ("Empty the God's Pockets"), written by the
model that did the work, against the written handoff
`FABLE_WAVE6_HANDOFF.md`. Timestamps from the git log of
`feat/fable-wave6-empty-pockets` (all 2026-07-04, local). The wave
spanned a power outage between the two A.1 commits; work resumed from
the session summary with no rework.

## Measured timeline

| Time | Commit | What |
| --- | --- | --- |
| 13:35 | de4678f | **Mission B** payroll refuse-to-generate on deductions > gross; whole-run refusal names each employee with amounts; golden changed deliberately (W6-D1) |
| 13:59 | 285681f | **A.1a** butler context builders → `pkg/butler/context` (~4,540 LOC of logic; six-hole HostPort; RBAC bool at the chokepoint — W6-D2) |
| 16:13 | 175b21f | **A.1b** grounded fastpath response builders → same package (~720 LOC; try* orchestrators stay root — W6-D3) |
| 16:27 | 120f226 | **A.2a** payment + supplier-payment deletes → `pkg/finance/payment`; **live bug found and fixed** (Commander-authorized): the supplier-payment delete rolled back into a column that has never existed (W6-D4) |
| 16:33 | 6439ad5 | **A.2b** customer + supplier invoice deletes → `pkg/finance/invoice`; quantity-invoiced reversal stays root behind a closure (finance does not import crm) |
| 16:38 | 6772d05 | **A.2c** PO + GRN deletes → `pkg/crm/procurement`; delivery note delete → `pkg/crm/fulfillment` |
| 16:42 | 264356f | **A.2d** expense category/vendor/entry deletes → `pkg/finance/expense`; handler-shell fields deleted, not rewired |
| 16:46 | 80b20f0 | **A.2e** customer/supplier + contact deletes → `pkg/crm/customer`; A.2 ledger closed (W6-D5) |
| 16:51 | 92fbaa7 | **C.1** visible Sign out in the header → `LogoutInteractiveSession`, `app:logout` → login screen |
| 16:58 | 9d244bf | **C.2 (stretch)** configurable inactivity timeout (5–480 min clamp, live-apply — W6-D6) |
| 17:01 | d176ba5 | **A.3** pipeline measured; one db-only leaf (`DeleteOfferNote`) peeled to `pkg/crm/pipeline`; no new ports minted (W6-D7) |

Plus the wave-end chore commit (regenerated Wails bindings) after this
audit.

## Mission status

### Mission A.1 — butler read paths: DONE (the headline)

`butler_ai_context.go` went from **5,070 → 459 lines** (thin delegates +
the six host-hub method bodies); `butler_grounded_fastpath.go` from
**1,682 → 964** (the nine try* orchestrators and their inference
helpers). `pkg/butler/context` now holds ~5,580 lines: the intent/full
context builders, entity resolvers, business summaries, finance
redaction rules, the pure query parsers, and the thirteen grounded
response builders — with new pkg tests for resolution
(exact/fuzzy/ambiguous), redaction-on-false, the parsers, and two
response builders. RBAC stays at the host chokepoint as a bool
parameter; the collaboration hub and cashflow projection ride behind
HostPort; no AI-vendor coupling entered pkg (the package builds context;
it never calls a model).

Honest notes:
- The boundary cut mid-file TWICE (W5-D3 held): the employee
  task-overview builder stays root (hub TaskItem), and
  `tryGroundedOfferDraftFastPath` stays root (orchestrator) despite
  sitting inside the moved line range.
- `pkg/butler/fastpath` (the map-port twin from an earlier partial
  migration) was deliberately NOT the destination — moving typed-GORM
  reads into map ports would have been a rewrite in refactor's clothes
  (W6-D3 / Mirror-D3). Unifying the twins is future work, honestly
  scoped.
- This package's `FirstNonEmpty` returns UNTRIMMED values; kernel
  `text.FirstNonEmpty` trims. Deliberately not merged.

### Mission A.2 — delete extraction: DONE, ledger closed (W6-D5)

Final tally across the Executor's dispatch: **15 moved, 2 already in
pkg, 6 stay root deliberately**.

- Moved (with pkg goldens, guard + RBAC always at root): payments (2),
  invoices (2), PO/GRN/DN (3), expenses (3), customers/suppliers/
  contacts (4), offer notes (1, via A.3).
- Already in pkg: bank statement + line (the Wave 5 banking peel).
- Stay: Order (ONE transaction spanning crm and finance — a saga, not a
  package boundary), RFQ/RFQWithCascade/CostingSheet (root-owned
  models), QuickCapture + CollaborativeTask (hubs, W4-D9).
- New kernel-pure `pkg/kernel/apperr` carries the host's
  `[CODE] Message: details` error contract into pkg — the string IS the
  contract (nothing type-asserts `*AppError`).

**The find of the wave (W6-D4):** the supplier-payment delete "rolled
back" the paid amount into `supplier_invoices.amount_paid_bhd` — a
column that no model or migration in the repo defines. The delete could
never succeed on a schema built from these models (pinned empirically
before fixing). The Commander authorized the in-wave fix: re-derive
`payment_status` from the payments that remain, exactly as
`UpdateSupplierPayment` already does. W4-D2 holds: the straggler is a
live bug — that is half the value of the peel.

### Mission A.3 — pipeline ports: DONE as measurement (W6-D7)

4,213 LOC, 149× `a.db`, 43× RBAC calls, root-owned models, and the
numbering stop-and-ask. One genuinely self-contained leaf existed and
was peeled (`DeleteOfferNote`). No new ports minted: the A.2 calling
convention (plain pkg functions over `*gorm.DB`; root wrappers hold
guard, RBAC, and host events) IS the port vocabulary the future
pipeline peel reuses. The Offer NN-YY numbering site was not touched.

### Mission B — payroll refusal: DONE (Commander-decided, W6-D1)

Whole-run refusal before the transaction opens; the error names every
offending employee with exact deduction and gross amounts. The Wave 5
clamp golden became the refusal golden — a deliberate, stated change.
Balanced-journal goldens pass byte-identical. Scope ended exactly at
the refusal (capping and receivable-booking were considered by the
Commander and not chosen).

### Mission C — session lifecycle: DONE including the stretch

- **C.1**: Sign out button in the EnterpriseHeader → the existing
  `LogoutInteractiveSession` binding (audit reason `user_logout`), then
  `app:logout` → App.svelte clears auth state → login screen. The
  frontend transition happens even if the backend call fails.
- **C.2 (stretch)**: `security.session_timeout_minutes` in settings
  (Settings → General). Read once at login, applied live on save,
  clamped to 5–480 minutes (W6-D6: a policy knob clamps — refusing a
  mistyped timeout means no timeout at all). Default stays the Wave 5
  30-minute policy.

## Honest accounting (LOC of logic moved, not method count)

| Move | pkg logic |
| --- | --- |
| `pkg/butler/context/service.go` | 4,840 lines |
| `pkg/butler/context/grounded_responses.go` | 739 lines |
| Nine delete extractions across 6 packages | 657 lines |
| `pkg/kernel/apperr` | 28 lines |

Root shrink: `butler_ai_context.go` −4,611; `butler_grounded_fastpath.go`
−718; ten root service files each lost their delete bodies (net branch
diff: +7,997 / −6,149 across 45 files, which includes ~600 lines of new
pkg tests and docs).

What did NOT move, and why it's in the decisions doc: the Order cascade
(saga), the RFQ/costing lifecycle (root models), the hubs (always
ports), the OneDrive/ETL machinery and PDF canvas (out of scope by
handoff).

## Definition of done

- Full `go test ./...` green (run after every extraction and at wave
  end); `go run ./cmd/hospitality` exit 0.
- `wails build -clean` + `svelte-check` 0 errors + bindings chore
  commit at wave end.
- Branch `feat/fable-wave6-empty-pockets` awaiting Commander review —
  nothing merged by the model.
