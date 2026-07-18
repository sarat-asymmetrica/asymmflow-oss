# Parity Ledger — Credit Notes (old, embedded in InvoicesScreen) vs Credit Notes descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Standalone ledger squatting inside InvoicesScreen | **EQUIV** | Confirmed exactly as PARITY_INVOICES.md #14 predicted: `credit-notes.descriptor.ts`, its own screen, linked from Invoices by action at INTEG. |
| 2 | List credit notes (`ListCreditNotes(100, 0)` called flat, no follow-up) | **DONE**, **fixed** | The old screen never paged past its first 100 rows despite the binding supporting `limit, offset` (recon K1-B #392/#398, same class of bug Payments' receipts had pre-Wave-9). `fetchPage`/`pageSize` (50) is wired from day one here — not preserved as a silent cap. |
| 3 | Division scoping (silent `matchesCompany(note.division)`) | **DONE** | Division filter chip, derived from data; consumes the divisions registry store at INTEG (L7). |
| 4 | Status: Draft / Issued / Applied | **DONE** | Derived status filter chip + `StatusSpec.tones` (Draft=neutral, Issued=info, Applied=success) + summary distribution bar. Unknown statuses surface honestly (a synthetic `UNKNOWN_STATE` row is woven into the mock). |
| 5 | Issue Credit Note (invoice picker + reason + line items, qty>0/rate>0 validation) | **SLOT** | Not built. Same line-item repeater territory as Invoices' proforma/credit-override flows (PARITY_INVOICES #7-#9) — the form archetype doesn't have a reusable line-item sub-pattern yet. |
| 6 | PDF (`GenerateCreditNotePDF`) | **DEFER** | Not built for K1, matching the same scope call PurchaseOrders.parity.md made for its PDF action (#10) — a pure scope call, not a gap; no guard rail is at risk. |
| 7 | Apply — reduces AR, fires with **NO confirmation** in the old screen | **DONE**, **fixed** | Recon K1-B flagged this explicitly (#391/#397) as a gap to fix, not preserve — directly comparable to Reverse Receipt (Payments), which DOES require confirm. Built as a confirm-gated row action (`ActionSpec.confirm`, naming the invoice being credited); mock mutation flips status to `Applied`, real mutation throws an honest INTEG-gap naming `ApplyCreditNote` (K1 mutations are gated at K5 regardless of the old binding being fully wired). |

## Reading

This cluster's third confirmed "fix, don't preserve" gap (after Payments'
Reverse-only-when-unapplied guard and the receipts pagination it already
fixed in Wave 9) lands here too: Apply reduces a customer's outstanding
balance and now requires an explicit confirm, matching the standard every
other AR-reducing action in this cluster uses. The two deep features — the
line-item Issue form and PDF export — stay ledgered exactly where
PARITY_INVOICES.md and PurchaseOrders.parity.md already drew that line for
comparable create/PDF actions elsewhere; nothing here needed a new
exception.
