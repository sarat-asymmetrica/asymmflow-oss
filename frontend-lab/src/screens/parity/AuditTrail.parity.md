# Parity Ledger — AuditTrailViewer (old) vs Audit Trail descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Browse per bank account → per statement → `GetAuditTrail(statementId)` | **DONE (fetch INTEG-gapped)** | Real `GetAuditTrail` takes a `bank_statement_id`, not a flat "all actions" query — the real fetch chain is `GetActiveBankAccounts()` → `GetBankStatements(accountId)` per account → `GetAuditTrail(statementId)` per statement, all confirmed real (`FinanceService`/`App`). K1's `cheque-register.ts` merges across bank accounts the same shape for its real fetch; this screen adds a third level (per-statement), and per the K4 brief the whole real side is INTEG-gapped rather than half-wired. Mock flattens straight to one feed. |
| 2 | Row-click = read-only detail view; Reverse = separate explicit action (Article V.4/B3(c)) | **DONE, preserved exactly** | **This is the hot-zone finding, not a footnote.** The `DocumentLedger` archetype already keeps row-selection (detail panel) and row actions (the actions column) on two separate paths — nothing in this descriptor wires Reverse to selection, so the old screen's deliberate anti-pattern-guard survives by construction. Anyone adding a `slots.detail` override later must not attach a reverse call to it. |
| 3 | Reverse a non-reversed action, mandatory reason | **DONE** | Row-aware reason `form` (ROW-AWARE FORMS pattern), gated `visible: !r.reversed`. Real throws naming `ReverseAction(logId, user, reason)` — confirmed real signature (`bank_integrity_service.go:96`, identity resolved server-side per its own comment, client `user` value is informational only). |
| 4 | `amount` column | **INTEG (backend gap, unverified)** | `finance.BankReconciliationAuditLog` (the real model) has no `amount` field — only `action`, `action_detail` (free text), `confidence_score`, `reason`, etc. The old screen must be deriving a displayed amount from `action_detail` or a joined record; that mapping wasn't traced in this recon (out of scope — fetch is INTEG-gapped regardless, #1). Mock generates a plausible amount directly since the real path never executes in the lab. Flag for whoever wires K5: confirm the real amount source before mapping it. |
| 5 | `GetAuditTrailByDateRange` imported but unused (old screen) | **DEFER** | Real binding exists but the old screen never calls it — not built here either, matching its own dead-import status. |
| 6 | KPI strip (4 stats) | **DONE**, close | `SummarySpec`: Actions (count), Reversed (count, danger-toned) + a by-action-type distribution bar. The old screen's exact 4th stat wasn't traced; this is the same "count + reversed + distribution" shape every other K1/K4 ledger uses. |
| 7 | Action-type badge coloring (IMPORT/MATCH/UNMATCH/SPLIT/CATEGORIZE/RECONCILE/VERIFY) | **DONE** | `ColumnSpec.tone` on the `action` column (7-way tone map, unknown → neutral) — separate from the ledger's canonical `StatusSpec`, which is the reversed-state (#2's gate needs that, not the action type). |

## Reading

The load-bearing finding here is #2, not a checklist item to skim past: this
screen exists specifically because someone already fixed a click-to-reverse
footgun once (Article V.4/B3(c) is cited in the old screen's own comments per
the census), and the archetype's row-select/row-action separation means this
descriptor can't regress it even by accident — there's no code path from
"user clicks a row" to "ReverseAction fires." The `amount` column (#4) is the
one honest unknown: the real audit-log model has no amount field, so
whatever the old screen shows there is derived from somewhere this recon
didn't trace — flagged rather than guessed at, and moot for K4 since fetch
is INTEG-gapped end to end regardless.
