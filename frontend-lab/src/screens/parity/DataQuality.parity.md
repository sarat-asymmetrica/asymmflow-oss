# Parity Ledger — DataQualityScreen (old) vs Data Quality LedgerDescriptor

Verdicts: DONE / EQUIV / ENGINE / SLOT / INTEG / DEFER.

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Issue queue (customer/opportunity/offer data-quality issues) | **DONE** | DocumentLedger; `PreviewCustomerDataQuality` wired REAL. Columns: entity type/name, issue kind, severity, review status, detail. |
| 2 | 4 KPI stat cards | **DONE** | Rebuilt as the ledger `summary` strip (issue counts + by-severity distribution). |
| 3 | Severity + review-status (two dimensions) | **DONE/ENGINE** | `severity` is the primary badge; `reviewStatus` a second toned column (same single-badge-per-row constraint + `secondaryStatus` ENGINE gap as Expenses/SupplierInvoices). |
| 4 | Review action (single shared textarea) | **EQUIV** | Replaced by 3 row-aware reason-form actions (Mark Reviewed / Resolve / Dismiss); each hides only when the row is already in that exact status — no invented state machine, matches the old screen's unconditional button availability. `ReviewDataQualityIssue` INTEG-gapped (mutation, wires at K5). |
| 5 | Review-history table (`GetDataQualityReviewHistory`) | **ENGINE** | A second table on the screen — un-buildable on the single-table ledger archetype (multi-panel gap, same as Expenses/Payments). Ledgered, not built. |
| 6 | Search + filters | **DONE** | One `searchText`; derived issue-kind + severity filters (with counts). |

## Reading
Data Quality fits the ledger archetype cleanly for its primary queue. The deep features
it shares with the finance cluster — dual-status rows and a co-located history panel —
are the same two ENGINE gaps already ledgered wave-wide (secondaryStatus, multi-panel
composition), not new. Preview is real; the mutating review actions INTEG-gap per the
standing convention.
