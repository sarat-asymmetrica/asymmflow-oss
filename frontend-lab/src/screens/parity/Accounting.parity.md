# Accounting — parity notes

**Entity:** `accounting` · **Group:** Finance · **Archetype:** bespoke (K4 L-monster)

Old screen: `AccountingScreen` (2098 lines, 4-tab console: Financial Overview /
Chart of Accounts / Journal Entries / Reports). New: `bridge/accounting.ts` +
`accounting-vm.svelte.ts` + `Accounting.svelte` — bespoke-on-primitives driven by
the `ViewSwitcher` primitive (horizontal segmented tab bar) instead of the old
hand-rolled left-nav. Highest financial hot-zone (GL, journal, VAT, trial balance).

## Capability census

| Old capability | Verdict | Notes |
|---|---|---|
| 4-tab nav | **EQUIV** | `ViewSwitcher` (horizontal) replaces the hand-rolled left-nav; same 4 sections. |
| Dashboard stats (assets/liabilities/equity/cash) | **EQUIV** | `StatTileGrid` over a live reduction of `GetChartOfAccounts` rows (old screen used a static object). |
| Asset/Liability/Equity split | **EQUIV** | `DonutWidget` — new (old screen had no composition chart). |
| Posting coverage | **DONE** | `DistributionWidget` by source_type from `GetPostingCoverageReport` (old screen never surfaced it). |
| Trial balance | **DONE** | `CalloutWidget` from `GetTrialBalanceGate`; `is_balanced` comes from the gate, not client-computed. |
| Cashflow Evidence command center | **DONE** | `StatTileGrid` KPIs + `ListWidget` evidence sources, mirroring the old derived metrics. |
| Action-proposal review (Approve/Needs-Input/Reject) | **EQUIV + INTEG** | Hand-composed Card+Badge+Button row per proposal (no kernel actionable-list widget). Framed as a **human decision log**, never an execution: `SyncCashflowEvidenceProposalReviews` + `ReviewCashflowEvidenceProposal` INTEG-gapped. |
| Chart of Accounts | **DONE** | One flat `DataTable` (code/name/type badge/balance toned/status) + `FilterChips` by type (incl. UNKNOWN_TYPE fallback + an "All" chip) — replaces 5 grouped card sections, data-equivalent + denser. |
| Add/Edit Account | **INTEG** | Modal + FormGrid (kernel `k-input` controls); `CreateAccount`/`UpdateAccount` INTEG-gapped (mock fully interactive). |
| Journal Entries | **DONE** | `DataTable`; credit cell tones danger when \|debit−credit\|>0.001 (display aid). |
| New Voucher (debit/credit lines) | **EQUIV** | Shared `LineItemsEditor` widget (account select + debit + credit, footer totals, Balanced/Unbalanced badge). Balance badge is **display-only** (`isDebitCreditBalanced`, unit-tested) — never authorizes the submit. |
| Post journal entry | **INTEG** | `CreateJournalEntry` INTEG-gapped; server-side balance/posting validation not reproduced client-side. |
| Reports: P&L / Balance Sheet | **DONE (FETCH real)** | `GenerateProfitAndLoss`/`GenerateBalanceSheet` wired real → typography Stack of labeled rows + section totals; loss years tone red, zero-revenue margins guarded. |
| CSV/VAT/Evidence-pack exports | **INTEG** | `ExportBalanceSheetCSV`/`ExportGeneralLedgerCSV`/`ExportJournalCSV`/`ExportVATReturnData`/`ExportCashflowEvidencePack` — all INTEG-gapped. |
| Bahrain VAT Summary card + fixed VAT_RATE=0.1 | **DROP** | Orchestrator-ratified. The old card computed VAT by substring-matching "sales"/"purchase" in free-text journal descriptions at a hardcoded 10% — fragile, mis-categorizes most entries, unrelated to the CoA VAT accounts or the real `ExportVATReturnData` binding. NOT reproduced. VAT accounts remain visible/filterable on the CoA; the VAT Return card calls the real binding. |

## Hot-zone / INTEG ledger (10 mutations, all named)
CreateAccount · UpdateAccount · CreateJournalEntry · SyncCashflowEvidenceProposalReviews ·
ReviewCashflowEvidenceProposal · SyncCashflowEvidenceProposalReviews (G3) · ExportCashflowEvidencePack ·
ExportBalanceSheetCSV · ExportGeneralLedgerCSV · ExportJournalCSV · ExportVATReturnData — all WIRED
(G3/G4): the 5 exports return the on-disk path (artifact-proven Go tests) and the sync reconciles the
review worklist (never posts). Mocks remain as the lab feature.
FETCH wired real (8/8): GetChartOfAccounts('All'), GetJournalEntries(year,0,null,100),
GetPostingCoverageReport(), GetTrialBalanceGate(year,0), GetCashflowEvidenceCommandCenter(30),
ListCashflowEvidenceProposalReviews(30,false), GenerateProfitAndLoss(year), GenerateBalanceSheet(year).
`GetJournalEntries` args resolved (fiscalYear, fiscalPeriod, isPosted *bool, limit) — no stop-and-ask.

## Orchestrator notes
- **Form controls** refactored to the kernel-owned `k-field`/`k-field-label`/`k-input` classes
  (single-source in `styles/kernel.css`) — the agent's `.acc-*` skin was the same L1/L2 duplication
  Payroll surfaced; killed at the root. One `.acc-error` raw hex → `var(--k-tone-danger-fg)`.
- Adversarial mock: 25 accounts (5 types + UNKNOWN, 200-char + empty names, negative/huge balances,
  VAT accounts both directions, inactive excluded from the voucher picker), 64 journal entries
  (unbalanced, reversal, 0-line, 40-line split, RTL + 200-char descriptions), coverage with missing>0,
  a 0.001-fils trial-balance boundary, cashflow statuses cycling ready/review/blocked/critical, loss + zero-revenue P&L years.

## Kernel gap (resolved)
No shared form-control primitive → **resolved** by the kernel `k-input`/`k-field` classes (same fix as Payroll).
No kernel "actionable list" widget for inline multi-button rows (proposal review) → hand-composed on primitives; candidate for a future widget.
