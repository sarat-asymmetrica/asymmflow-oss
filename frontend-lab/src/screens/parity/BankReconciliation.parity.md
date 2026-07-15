# BankReconciliation — parity notes

**Entity:** `bank-reconciliation` · **Group:** Finance · **Archetype:** bespoke (K4 L-monster)

Old: `BankReconciliationScreen.svelte` (2140 lines) — transaction-level statement
import + line matching. **NOT** `BookBankRecon` (month-end running-total comparison).
New: `bridge/bank-reconciliation.ts` + `bank-reconciliation-vm.svelte.ts` +
`BankReconciliation.svelte`. Headline: the old screen's entire hand-rolled allocation
panel (search/add/remove/amount-edit/remainder) collapses into the shared
`AllocationMatchPanel` primitive **with zero modifications**.

## Capability census

| Capability | Verdict | Notes |
|---|---|---|
| Bank-account picker (division-scoped) | **DONE/EQUIV** | Division is a data field; client-side scoping lands with the K5 divisions store (inherited kernel simplification, not unique to this build). |
| Cash-position KPI | **DONE** | `StatTileGrid`; `GetCashPosition()` untyped map defensively probed (current_balance_bhd/CurrentBalanceBHD/…). |
| Cash-position discrepancy notices | **DONE/EQUIV** | `CalloutWidget`; adds a per-statement discrepancy callout the old screen didn't separately surface. |
| Statements list / Statement-lines list | **DONE** | Two `DataTable`s; lines add first-class "Detected" (extracted customer) + "Confidence" columns (old screen only inline-colored these). |
| Two-phase import (Preview → Confirm/Discard) | **INTEG** | Shape preserved (nothing persists until Confirm); all 3 bindings INTEG-gapped, mock performs it interactively. |
| Dead CSV-import bindings | **DROP** | `ImportBankStatementCSV`/`ImportBankStatementWithDialog` not resurrected (old screen abandoned them). |
| Manual match (single) | **INTEG** | `AllocationMatchPanel` 1-allocation → `ManualMatchLine`. |
| Split allocation (multi) | **INTEG** | `AllocationMatchPanel` >1 allocation → `CreateSplitAllocation`. The old hand-rolled allocation UI is now the shared primitive — the rebuild's headline simplification. |
| Candidate typing (credit→Customer Invoice; debit→Supplier Invoice/Expense/Supplier Payment/Payroll Payout) | **DONE** | `allowedCandidateTypes()` in the VM (L2). |
| Single-select types (Supplier Payment, Payroll Payout) | **DONE/EQUIV** | `AllocationMatchPanel.singleSelectTypes` enforces one-at-a-time structurally (old screen used a toast-and-reject). |
| Unmatch / Auto-match | **INTEG** | `UnmatchLine` / `AutoMatchBankLines`. |
| Finalize (gated on 0 unmatched) | **INTEG (HOT-ZONE)** | Gate preserved verbatim: `disabled={totalUnmatched > 0}`. Needs an authenticated actor — see kernel gap. |
| Delete statement | **INTEG (HOT-ZONE)** | `ConfirmDialog` + `DeleteBankStatement`. |
| Edit statement / Add-Edit-Delete line | **INTEG** | Modal + FormGrid (kernel `k-input` controls). |
| **"Editing a matched line clears its match"** | **PRESERVED** | VM captures `editingLineWasMatched`; the Edit-line modal shows a dedicated warning callout when true. Non-obvious side effect preserved. |
| One-click OCR "Fix Debit/Credit" flip | **SIMPLIFIED** | Dropped the one-click button; manual retype in the Edit-line modal still corrects an OCR flip. Flagged, not a capability loss. |
| Audit trail (`GetAuditTrail`) | **EQUIV (NEW)** | Old screen never called this real binding; added an `ActivityFeed` drawer ("Audit Trail" toolbar button). Cheap real FETCH. |
| "Step 1 of 2" close-the-month framing + cross-nav to BookBankRecon | **DEFER** | Not ported — month-end sequencing UX out of scope. Flag if Book-vs-Bank cross-nav is wanted later. |
| Bank-account CRUD | **EQUIV** | Reads `GetActiveBankAccounts()` only; CRUD stays in Settings → Bank Accounts (matches old delegated behavior). |

## INTEG / hot-zone ledger (13 mutations, all named)
PreviewBankStatementImportWithDialog · ConfirmBankStatementImport · DiscardBankStatementImportPreview ·
AutoMatchBankLines · ManualMatchLine · CreateSplitAllocation · UnmatchLine · **FinalizeReconciliation** (HOT) ·
**DeleteBankStatement** (HOT) · UpdateBankStatement · CreateBankStatementLine · UpdateBankStatementLine ·
DeleteBankStatementLine — each throws `INTEG gap: <Binding> — wires at K5`; mock performs the action so flows demo.
FETCH wired real (10): GetActiveBankAccounts, GetBankStatements, GetBankStatementLines, GetCashPosition,
ListCustomerInvoices, GetSupplierInvoices, GetAllSupplierPayments, ListExpenseEntries, ListUnreconciledPayrollPayouts, GetAuditTrail.

## Orchestrator notes
- Form controls refactored to kernel `k-field`/`k-field-label`/`k-input`/`k-field-wide` classes (L1/L2), consistent with Payroll/Accounting/CostingSheet.
- Adversarial mock (all in auto-selected stmt-1): empty + 200-char descriptions, RTL extracted-customer, huge 999999.999 / tiny 0.001, OCR-flip both-nonzero line (UNKNOWN type), duplicate references, orphan-matched line (points at non-existent invoice), unmatched-credit line with zero candidates (panel empty-state), a 230-row candidate pool (proves the amount-proximity sort + 40-row cap), a discrepancy!=0 statement. Whole-screen empty-bankAccounts zero-state structurally reachable. Synthetic Gulf names only.

## Kernel gaps / stop-and-asks
1. **No session/currentUser store** in the lab — `FinalizeReconciliation`/`ManualMatchLine` need an acting user; a documented placeholder `actor = 'lab-user'` is used (all mutations INTEG-gapped anyway). **Ask: land a shared session primitive at K5 before wiring these for real.**
2. Audit-trail drawer added as an EQUIV improvement (real binding, previously unused) — confirm it should stay.
3. Book-vs-Bank cross-nav ("Step 1 of 2") not ported — confirm whether K5 should restore it.
