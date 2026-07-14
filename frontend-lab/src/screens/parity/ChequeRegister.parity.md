# Parity Ledger — ChequeRegisterScreen (old) vs Cheque Register descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Three co-located sub-views (Outstanding / Registers / Stale), each a SEPARATE bank-account-scoped fetch | **ENGINE gap** | K1 builds the PRIMARY ledger only — Outstanding cheques. Registers (cheque-book usage) and the Stale-only tab are genuinely different fetches (`GetChequeRegisters`, `GetStaleCheques`), not a client-side filter of one dataset — needs either a screen-level layout composing multiple descriptors, or a `fetch` that accepts a view parameter. Same finding as K1-B synthesis #1 (5 of 7 screens in this cluster need multi-panel composition). |
| 2 | Bank-account selector driving all three sub-views | **ENGINE gap** | A "required selector that reloads data," not a `FilterSpec` over already-loaded rows (K1-B synthesis, ChequeRegister #6). Out of K1 scope per the brief. The real bridge works around this honestly: it fetches `GetActiveBankAccounts()` then merges every active account's `GetOutstandingCheques()` result into one feed (`cheque-register.ts:realFetchAll`), rather than faking a single "get all cheques" binding that doesn't exist. Mock shows cheques across two synthetic accounts at once for the same reason. |
| 3 | Outstanding cheques list (`GetOutstandingCheques`, `{cheques, total}`) | **DONE** | `fetch()` — unpaged, matches the real binding's per-account shape (merged across accounts as above). |
| 4 | Legal-transition-gated action set per cheque status (`chequeActionsFor()`, mirrors `cheque.go`) | **DONE** | Declared as `StatusSpec.transitions` (verified against `pkg/finance/cheque/cheque.go` directly — `MarkCleared`/`MarkStale`/`MarkBounced` gate on ISSUED/PRESENTED, `Cancel` gates on ISSUED only) and consumed via the shared `nextStates()` helper, same pattern as `purchase-orders.descriptor.ts`. |
| 5 | Mark Stale (row, ISSUED/PRESENTED only) | **DONE** | Plain confirm + mock mutation; real = INTEG-gap naming `MarkChequeStale`. The Wails binding exists and is fully wired on the old screen, but K1 doesn't wire real mutations (that's K5 quarantine-backend territory) — same trade-off `purchase-orders.ts`'s `setPurchaseOrderStatus` makes. |
| 6 | Cancel Cheque with operator-supplied reason (ISSUED only) | **DONE** | Row-aware reason `form` (ROW-AWARE FORMS, new since batch 1) — `FormSpec.submit(draft, row)` receives the clicked cheque, so the reason capture is real, not degraded to a plain confirm the way PurchaseOrders' Cancel had to be before this feature landed. Mock mutation; real = INTEG-gap naming `CancelCheque`. |
| 7 | Issue Cheque (screen create, gated on `nextChequeNumber !== 'N/A'`, needs account context + a live next-cheque-number preview) | **SLOT (financial hot-zone)** | Not built. Genuinely coupled to the deferred account-selector (#2) and a cross-call preview (`GetNextChequeNumber`) — building it as a bare header-only form would either fake the account context or drop the preview gate. Ledgered with #2. |
| 8 | Mark Cleared (row, bank-statement-line picker cross-referencing a THIRD data source) | **SLOT (financial hot-zone)** | Not built — deeply reconciliation-specific, needs its own ejection panel pulling `GetBankStatements`/`GetBankStatementLines` mid-action, not a form-archetype field list. Matches the census's own verdict for this capability. |
| 9 | New Cheque Book (`CreateChequeRegister`) | **SLOT** | Administrative, not money-movement, but lives on the Registers sub-view — ledgered with #1 rather than built standalone. |
| 10 | Row-action dropdown (⋮) instead of inline buttons | **DONE-shape** | `ActionSpec.visible` already models this — the dropdown vs. inline-button choice is a rendering detail the archetype/engine owns, not a descriptor concern. Two actions (Mark Stale, Cancel) render fine either way. |
| 11 | KPI strip (Outstanding Total, Outstanding Count, Stale Count, Next Cheque #) | **DONE**, partial | `SummarySpec`: Outstanding Cheques (count), Outstanding Total (BHD), Stale (count, amber/danger-toned) + a 6-segment status distribution bar. "Next Cheque #" is dropped — it's meaningless without a selected bank account (#2), so it's ledgered with the account-selector gap rather than faked. |
| 12 | Register "Used %" progress bar | **DEFER** | Belongs to the Registers sub-view (#1), not the Outstanding ledger this K1 build covers. |
| 13 | Age / staleness badge on outstanding cheques | **DONE**, improved | Old screen showed a bare "STALE" badge or nothing. K1 adds a computed `Age (days)` column (`Date.now()` minus `issuedDate`, same wall-clock pattern `supplier-invoices.descriptor.ts` uses for `daysUntilDue`) with a 150-day amber threshold ahead of the backend's 6-month stale window (`cheque.go:315`) — genuinely more informative than the old binary badge. |

## Reading

The cluster-wide "three sub-views, three fetches" gap (K1-B synthesis #1)
lands here concretely: Outstanding is the only sub-view this K1 build
covers, per the brief's multi-panel rule. The one design decision worth
flagging is the real bridge's account-merge (`realFetchAll` in
`cheque-register.ts`): rather than block on the deferred account-selector,
it fetches every active bank account and concatenates their outstanding
cheques — an honest, if slightly more expensive, way to keep the real path
functional without faking a binding that doesn't exist. Both built actions
(Mark Stale, Cancel) are gated by a `StatusSpec.transitions` table verified
directly against `pkg/finance/cheque/cheque.go`, not assumed from the
census — Cancel is the first ROW-AWARE reason form built in this batch,
capturing a real operator reason instead of the plain-confirm downgrade
PurchaseOrders' Cancel needed before this feature existed.
