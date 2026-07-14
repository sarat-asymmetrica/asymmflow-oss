# BookBankRecon — parity notes

**Entity:** `book-bank-recon` · **Group:** Finance · **Archetype:** bespoke (K4)

## Old screen

`BookBankReconciliationScreen` — classic month-end book-vs-bank balance
statement (deposits in transit, outstanding cheques, book errors/NSF/interest,
variance, finalize). Split view: list of reconciliations (left) + detail
(right). **Not** the same screen as `BankReconciliationScreen`, which is
transaction-level statement-line import/match — confirmed distinct by recon
K4 census (see `recon-K4.md`, "BankRecon vs BookBankRecon — not duplicates").

Old bindings (all real on the old screen):
- `GetActiveBankAccounts` (`App`)
- `GetBookBankReconciliations` / `CreateBookBankReconciliation` /
  `UpdateBookBankReconciliationAdjustments` / `FinalizeBookBankReconciliation` /
  `GetReconciliationVariances` / `GetDepositsInTransit` (`InfraService`)
- `GetOutstandingCheques` (`FinanceService`)

## This build

- List (left): `DataTable` — period / account / status / variance (money,
  toned success when reconciled, danger otherwise).
- Detail (right): `Card` header (account, period, status badge) +
  the new **`BalanceComparisonPanel`** kernel primitive (bank statement
  column vs book/GL column, each with its line items and a bold total,
  plus a full-width variance banner) + a `Finalize` action gated behind
  `ConfirmDialog` (financial hot-zone — explicit confirm, preserved from the
  old screen's finalize-locks-the-record behavior).
- All arithmetic (adjusted bank balance, adjusted book balance, variance,
  reconciled predicate) lives once in `book-bank-recon.svelte.ts` (L5) and
  feeds both the list's variance column and the detail panel's banner, so
  the two surfaces can never disagree about what counts as reconciled.

## Not built (deferred, no old-screen precedent lost)

- **New/Edit reconciliation** (`CreateBookBankReconciliation`) and
  **editing adjustment lines** (`UpdateBookBankReconciliationAdjustments`)
  are not built as a `FormModal` in this pass — the brief left this
  optional ("a FormModal or ledger it") and the K4 scope here is the
  comparison-panel + finalize flow the census flagged as the reusable
  piece. Synthetic dataset ships with adjustment lines already populated
  per reconciliation so the panel has real content to render; wiring a
  create/edit form is a follow-up, not a gap in this screen's own logic.
- `GetReconciliationVariances` has no separate surface here — its
  information (the variance) is computed client-side from the same lines
  the comparison panel already renders, one definition instead of two.

## Bridge / mock

`bridge/book-bank-recon.ts` — self-contained mock + real + `pick()` switch,
seeded LCG (`20260714`), 26 synthetic reconciliations across 5 bank accounts
(BHD + USD). Adversarial monsters woven in at deterministic positions:
huge statement balance (`123456789012.345`), negative book balance
(overdrawn control account), a row with zero supporting lines (empty
deposits/cheques/adjustments — exercises `BalanceComparisonPanel`'s
empty-lines path), an unbroken 70-char account-name token, an empty account
name, an `UNKNOWN_STATE` status (unmapped `StatusSpec` tone → renders
neutral, doesn't crash), and rows engineered to land exactly on the
reconciled tolerance (proves the success-tone banner path, not just danger).

Real fetch/finalize are INTEG-gap throws:
- `fetchBookBankReconciliations` → names `GetBookBankReconciliations` +
  `GetDepositsInTransit` + `GetOutstandingCheques` (a 3-call aggregation per
  record, not a straight 1:1 swap — same shape as `cheque-register.ts`'s
  `realFetchAll`).
- `finalizeBookBankReconciliation` → names `FinalizeBookBankReconciliation`.

## New kernel primitive: BalanceComparisonPanel

`kernel/primitives/BalanceComparisonPanel.svelte` — generic N-column balance
comparison (title + lines + bold total per column) plus a variance banner
(tone success/"Reconciled" when `|variance| < 0.001`, else tone danger
showing the money variance). Reusable beyond this screen — the census
flagged it as the shared piece for any month-end two-sided reconciliation.
Owns its own layout CSS per kernel convention; tokens only, no raw hex;
numeric font on all money; every label/note truncates with a title tooltip
(min-width:0 throughout — safe at the 420px truncation-detector width).
