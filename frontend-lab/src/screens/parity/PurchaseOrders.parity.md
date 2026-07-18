# Parity Ledger — PurchaseOrdersScreen (old) vs Purchase Orders descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | List all POs (`GetPurchaseOrders()`, unpaged) | **DONE** | Flat `fetch()`, no `fetchPage` — matches the real binding's shape (unpaged, despite `ListPurchaseOrdersPaginated` existing unused on the backend). |
| 2 | Status tabs (static, omits Cancelled/Pending Approval/Approved) | **DONE**/EQUIV, **fixed** | `options: 'derive'` — every status actually present in the data gets a chip, closing the census-flagged gap in the old screen's tab list rather than reproducing it. |
| 3 | Multi-currency Net Amount column (7 currencies) + BHD-total column | **DONE** | `ColumnSpec.currency = (r) => r.currency` on Net Amount; Total incl. VAT stays BHD-only, matching the census exactly. |
| 4 | Status-transition legality table (`PO_STATUS_TRANSITIONS`) gating available row actions | **DONE** | Declared as `StatusSpec.transitions` (data) AND consumed via the kernel's `nextStates()` pure helper (`ledger-core.ts`) inside `legalTargets()` — the same table drives both the audit-visible schema and the actual gating, so it can't drift into two versions of the truth the way `PO_STATUS_TRANSITIONS`/`getAvailableTransitions` did in the old screen. The >5000 BHD threshold value and the `RECEIVABLE_STATUSES` receiving-lockout are still duplicated from the backend here (same drift risk the old screen carried — a future `GetPurchaseOrderPolicy()`-style binding should own these, not either frontend). |
| 5 | Simple status flips (Draft→Pending Approval, Pending Approval→Draft, →Sent, →Acknowledged) | **DONE** | Row actions, gated by `legalTargets()`, plain-string confirm, mock mutation; real = INTEG-gap naming `UpdatePOStatus`. |
| 6 | Approve (Pending Approval → Approved, SoD-gated, `ApprovePurchaseOrder(id, userId)`) | **SLOT (financial hot-zone)** | Not built. Deliberately excluded from the generic status-flip actions above — it's a distinct binding with a segregation-of-duties requirement (approver ≠ creator, server-enforced), not a plain `UpdatePOStatus` call. Reimplementing it as a bare confirm would silently drop the SoD guard. |
| 7 | Cancel PO with operator-supplied reason (`confirm.askForReason` in the old screen) | **ENGINE gap** | Built, but *degraded*: it's a plain-string confirm (`Cancel PO for PO-2026-0001?`), not a reason-capturing form. Root cause verified in the engine, not assumed: `ActionHost.svelte` sets `formAction = { action, row }` when a row action declares `.form`, but only forwards `formAction.action.form` (the spec) to `<FormModal>` — the row is never passed through, and `FormSpec.initial()`/`.submit(draft)` are both zero-context (no row parameter in their signatures, per `kernel/form.ts`). So a row-scoped "confirm-as-form" genuinely isn't buildable today without an engine change threading the selected row into the form draft. This is the same gap K1-A's recon flagged generically (synthesis #3, Orders #3) — this screen is where it first blocks something concretely. Flagging to the orchestrator rather than working around it with a module-level mutable-row hack. |
| 8 | Receive Items (highest-risk: posts inventory, creates a GRN, can raise supplier discrepancies) | **SLOT (financial hot-zone)** | Not built — per the brief, ledgered as-is. Needs its own line-by-line ejection panel (serials, rejects, discrepancies), not a form-archetype field list. |
| 9 | New PO / Edit PO (multi-currency, live subtotal/VAT/BHD math, 7-currency support) | **SLOT** | Not built. Same territory as Invoices #7–#10 — needs the form archetype's derived/computed-read-only-field support, which doesn't exist yet. |
| 10 | PDF generation (`GeneratePurchaseOrderPDF`) | **DEFER** | Not built for K1 (brief's scope discipline covers the mutating actions explicitly; PDF export was left out to keep this screen's action surface to status-flips only — no guard rail is at risk either way, this is a pure scope call, not a gap). |
| 11 | Sessionstorage handoff (preseed supplier on create from elsewhere) | **DEFER** | Cross-screen navigation glue; revisits once the app-shell nav model exists. |
| 12 | Summary stats strip (Total POs, Total Value, Pending Receipts, Fully Received) | **DONE** | `SummarySpec`: count, Total Value (BHD), Pending Receipts (amber when >0, driven by the same `RECEIVABLE_STATUSES` list the receiving-lockout guard uses), Fully Received, + a 9-segment status distribution bar. |

## Reading

This is the furthest-along screen in the K1-B cluster on the OLD side (every
binding real, no INTEG gaps in the census) — which makes it a clean test of
where the KERNEL's gaps are, not the backend's. Two genuine engine findings
came out of building it: the status-transition table now has exactly one
source of truth via `nextStates()` (a real improvement, not just parity), and
the row-scoped "confirm as form" pattern the brief suggested for Cancel does
not actually exist in `ActionHost`/`FormModal` yet — verified by reading
both files, not assumed. Cancel ships today as a plain confirm (reason
capture lost) rather than either faking row-context with a module-level
variable or blocking the whole action on an engine change; that trade-off is
called out above (#7) for the orchestrator to weigh against building the
"richer confirm" ENGINE feature K1-A independently flagged (synthesis #3).

Everything genuinely financial and irreversible — Approve, Receive Items,
multi-currency create/edit — stays ledgered, unbuilt, with its guard rails
intact in the old screen and undisturbed here.
