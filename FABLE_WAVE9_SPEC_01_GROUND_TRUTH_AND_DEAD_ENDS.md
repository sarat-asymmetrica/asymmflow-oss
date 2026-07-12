# Wave 9 Spec 01 — Ground Truth & Kill the Dead Ends

**Mission:** Wave 9.0 (OSS ground-truth check) + Wave 9.1 (kill the dead ends) from `FABLE_WAVE9_UIUX_AUDIT.md` §5.
**Repo:** `asymmflow-oss` (this repo). **Branch:** `feat/fable-wave9-1-dead-ends` off `main`. Do not merge or push; leave the branch for owner review.
**Authority documents, in order:** `CLAUDE.md` (repo invariants) → `DESIGN_CONSTITUTION.md` (design law) → `FABLE_WAVE9_UIUX_AUDIT.md` (evidence, keep-lists, plan) → this spec.

## 0. Read before anything

1. `CLAUDE.md`
2. `DESIGN_CONSTITUTION.md` — every diff is reviewed against it
3. `FABLE_WAVE9_UIUX_AUDIT.md` — §2 (patterns), §3 (themes), §4 keep-lists for inventory/CRM-shell/work/finance domains, §5 Waves 9.0–9.1
4. This spec, fully, before spawning any coder

**Data-sensitivity invariant:** you may read `../ph_holdings` for reference implementations, but real client names/figures never enter this repo — synthetic canon only (`SYNTHETIC_IDENTITY.md`).

## 1. Operating model — who you are

You are an **Opus 4.8 orchestrator acting as a senior designer/tech lead**. You write specs-of-work and review code; **Sonnet 5 subagents write the code** (Agent tool, `model: "sonnet"`, `subagent_type: "general-purpose"`).

- **Delegate in batches** (§4 suggests a batching). Each coder prompt must contain: the exact work items with acceptance criteria, the relevant audit findings (file:line), the applicable constitution articles quoted or referenced, the domain keep-list, and the gate commands.
- **Review every diff yourself** before it is committed — constitutional review (Article VII.1): reject or fix work that "works" but violates the law (wrong component, raw hex, native confirm, free-jump status, ghost actor, duplicated path). You personally correct small substandard details rather than round-tripping; re-delegate large misses.
- **Keep it green at every commit:** `cd frontend && npx vite build && npx svelte-check` (treat NEW errors as failures; pre-existing warnings are not yours to fix). If Go is touched: `go test ./...`. Note: `go build ./...` needs `frontend/dist` — build frontend first.
- **Commit per work item** (small, coherent): `feat(wave9.1): <item> — <what>` or `chore(wave9.1): delete <dead code>`. Final commit: `docs(wave9): spec-01 status report`.
- **Never treat a subagent's claim as verification.** Run the gates yourself after each batch lands.

## 2. Phase A — Wave 9.0 ground-truth check (read-only, do first)

The audit graded **deployed PH**; OSS is a ~1:1 port with known deltas. Before fixing anything, verify each item below on THIS repo and record a verdict table (finding → `CONFIRMED HERE` / `ALREADY FIXED` / `DIVERGED: <how>`). Adjust the Phase B work-list accordingly — do not "fix" what Wave 6/8 already fixed.

| # | Check | Expectation |
|---|---|---|
| A1 | Logout / user menu in the shell | Wave 6 shipped logout + inactivity timeout — verify exposed in shell UI |
| A2 | Employee-archive approval surface | P4 added the approval ride on NotificationsScreen — verify reachable/discoverable |
| A3 | Supplier-invoice Edit-modal Status/PaymentStatus dropdowns vs the P5-1 Approved-only backend gate | Likely a LIVE hard-error bypass here — if confirmed, report severity; the fix itself belongs to Wave 9.2, do not do it now |
| A4 | Wave 8 backend wiring status: `GetInventoryPendingFulfillmentReport`, `GetDashboardPipelineByStageYTD`, `GetDashboardARAgingReportYTD`, `CreatePOsFromOrder`, `PreviewOrderDeleteCascade`, `GetPreparedByOptions` | Backends exist (Wave 8); check which have frontend callers |
| A5 | Spot-check that the audit's file:line anchors for Phase B items resolve in this repo's `frontend/src` | Adjust anchors where drifted |

Commit the table as part of the final status report (no code changes in Phase A).

## 3. Phase B — Wave 9.1 work items

Nine items, audit §5 Wave 9.1, expanded with acceptance criteria (AC). PH file:line anchors are hypotheses until Phase A confirms them here.

**B1 — Dashboard drill-throughs (T2).** KPI cards (Revenue→finance, AR→invoices filtered, Pipeline→opportunities, Cash→bank-recon), pipeline donut/stage rows → stage-filtered Opportunities, aging bars → matching invoice filters, task rows → `openCollaborativeTask`. Wire pipeline/aging to `GetDashboardPipelineByStageYTD` / `GetDashboardARAgingReportYTD`. *Copy the proven pattern: FinancialDashboard's `pendingInvoiceFilter` drills (constitution pattern #1).*
**AC:** every headline number on DashboardScreen navigates somewhere filtered-correct; no dead KPI remains; navigation preserves company scoping.

**B2 — Operations "Fulfillment" tab.** The pending-fulfillment report is fetched then discarded (OperationsHub:56,65). Surface it as a tab — the back-to-back trader's core question.
**AC:** tab renders the report via canonical DataTable; rows deep-link to their order/PO; empty/loading/error states present.

**B3 — 360 continuity.** Supplier-360: make PO/invoice rows clickable (mirror customer-360's drills) + "New PO for this supplier" preseeded action. Customer-360: "New RFQ for this customer" preseeded action (pattern #1 handoff).
**AC:** supplier rows drill like the customer twin's; both "New X" actions land in the target screen with the party pre-filled.

**B4 — Cheque lifecycle row actions.** Clear/Cancel/Stale handlers exist but are wired to nothing (ChequeRegisterScreen:341-361,435-455).
**AC:** row-action menu exposes only legal transitions per current status (pattern #2, no free-jump); actions use the confirm primitive; register reflects new state without reload.

**B5 — Serial-trace deep-links.** PO/GRN/DN/Invoice references are plain text (SerialTraceScreen:76-106).
**AC:** each reference navigates to its document, preserving the established cross-screen nav pattern.

**B6 — DN ergonomics at the closing moment.** (a) Delivery address auto-fills from customer/order, editable (pattern #8); (b) Dispatch prompts inline for missing driver/vehicle instead of the create→reject→edit loop (pattern #4); (c) remove the create-form status picker — lifecycle only via detail actions (Article III.3).
**AC:** creating a DN from an order requires zero re-typing of known data; Dispatch never dead-ends; no status select at create.

**B7 — Order→PO handoff correctness.** `CreatePOsFromOrder` opens the created PO draft (not the list) + toast; wire the item-aware handler, delete the item-less dead one; disable DN/Supplier-Order CTAs on zero-item orders with tooltip (pattern #7) instead of failing on click.
**AC:** the happy path lands inside the new PO; zero-item orders show disabled+explained CTAs; exactly one PO-creation path from orders remains.

**B8 — Dead-code deletion + Inbox decision.** Delete: Offers inline create/edit modals (OffersScreen:1469-1581,1584-1738), the ~490-line `{#if false}` PO modal block (PurchaseOrdersScreen:1022-1509), dead `handleCreatePO` remnants after B7. **Inbox: default decision = RETIRE** the orphaned InboxScreen (it is unrouted, its filters are `===` no-ops, and document triage is served by the Capture/OCR flow). If you find evidence wiring it is genuinely low-cost AND non-redundant, you may instead wire it as document triage — either way, record the decision + rationale in the report.
**AC:** grep confirms no references to deleted symbols; build green; decision documented.

**B9 — Native `confirm()`/`prompt()` → canonical Modal sweep (T5, app-wide).** Known sites: supplier approve, NotificationsScreen rejections (needs a reason-input modal), supplier issue resolve, customer contact delete, bank-recon deletes — plus any others a sweep finds.
**AC:** `grep -rn "window.confirm\|window.prompt\|confirm(\|prompt(" frontend/src` (filtered to real native calls) returns zero; every replacement uses the one confirm primitive / Modal with reason capture where a reason was collected before.

### Suggested coder batching (adjust from Phase A findings)
- Coder 1: B1 + B2 (dashboard/ops wiring, Wave 8 backends)
- Coder 2: B3 + B5 (360 + trace continuity)
- Coder 3: B4 + B6 (cheque + DN lifecycles)
- Coder 4: B7 + B8 (orders/PO handoff + dead-code)
- Coder 5: B9 (modal sweep — run LAST so it sweeps the others' output too)

## 4. Hard boundaries

- **Do not** start Wave 9.2+ work (supplier AP unification, recon merge, People re-model) even where adjacent — one exception: A3's severity report.
- **Do not** touch financial semantics (rounding, posting order, tax) — stop-and-report instead (CLAUDE.md invariant 5).
- **Do not** violate a keep-list behavior (audit §4) — the keep-lists for inventory, CRM/shell, work, and finance domains all touch this wave's screens.
- **Do not** merge, push, or tag. Branch stays local for review.
- **Owner-decision items** (proforma invoices, visa tracking, etc.) remain out of scope.

## 5. Definition of done + status report

Done = Phase A table complete; all nine B-items shipped (or explicitly skipped with reason); gates green on the final commit; report written.

Write the report to `FABLE_WAVE9_SPEC_01_REPORT.md`, commit it, and paste it verbatim into the conversation as your final message — the owner carries it to the final gate. Template:

```markdown
# Wave 9 Spec-01 Status Report
**Branch:** feat/fable-wave9-1-dead-ends · **Commits:** <n> (<first>..<last>)
**Gates:** vite build ✅/❌ · svelte-check ✅/❌ (new errors: n) · go test (if run) ✅/❌

## Phase A — ground-truth verdicts
| Check | Verdict | Notes |
(A1–A5)

## Phase B — items
| Item | Status (shipped/partial/skipped) | Commits | Notes |
(B1–B9)

## Decisions taken
(e.g. Inbox retire/wire + rationale)

## Constitution deviations requested
(article, where, why — or "none")

## Keep-list attestation
(confirm each touched domain's keep-list behaviors were preserved, or list breaks)

## Known residue / follow-ups
## Open questions for the owner
```

Severity honesty is law: failed gates, skipped items, and hacks are reported as such — an accurate red report beats a false green one.
