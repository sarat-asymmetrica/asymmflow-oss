# Wave 17 Progress Audit - Accounting Posting Spine

**Date**: 2026-05-08  
**Roadmap**: `docs/V0_1_RELEASE_ROADMAP_2026_05_08.md`  
**Status**: Active

## Commit Table

| Ticket | Commit | Result |
|---|---|---|
| 1 | `a9ff523` | Added additive posting preview package, customer/supplier invoice/payment previews, Wails endpoints, bindings, and tests |
| 2 | `6814f7c` | Added posted-journal trial-balance gate, Wails endpoint, bindings, and tests |
| 3 | `3e8a1c6` | Added account-mapping resolution and idempotent draft journal creation from posting previews |
| 4 | `e43020c` | Added posting coverage report for eligible documents missing journal links |
| 5 | `ef180b0` | Surfaced posting coverage and trial-balance status in Accounting dashboard |

## What Landed

- New package: `pkg/finance/posting`
- Source preview coverage:
  - Customer invoice: debit AR, credit revenue, credit output VAT
  - Customer payment: debit bank/cash, credit AR
  - Supplier invoice: debit purchases, debit input VAT, credit AP
  - Supplier payment: debit AP, credit bank/cash
- New FinanceService/App endpoints:
  - `PreviewCustomerInvoicePosting`
  - `PreviewCustomerPaymentPosting`
  - `PreviewSupplierInvoicePosting`
  - `PreviewSupplierPaymentPosting`
  - `CreateDraftJournalFromPosting`
  - `GetPostingCoverageReport`
  - `GetTrialBalanceGate`
- Generated frontend bindings now expose `posting.Entry`, `posting.Line`, `posting.AccountRef`, `posting.CoverageReport`, `posting.CoverageRow`, `posting.TrialBalanceGate`, and `posting.TrialBalanceRow`.
- Accounting dashboard now shows posting coverage, missing journal links by source type, draft-entry count, and trial-balance status.

## Verification Results

| Gate | Result |
|---|---|
| `go test ./pkg/finance/posting -count=1` | Passed |
| `go test . -run "TestPreview.*Posting|TestGetTrialBalanceGate|TestCreateDraftJournalFromPosting|TestGetPostingCoverageReport" -count=1` | Passed |
| `go build ./...` | Passed |
| `cd frontend && npm run check` | Passed with 0 errors, 13 existing warnings |
| `wails generate module` | Passed with known anonymous `{R1,R2,R3}` warning |

## Current Boundary

This wave slice is intentionally additive. It produces balanced posting intent, draft journal entries, source `JournalEntryID` links, and ledger health gates. It does not yet auto-post the generated drafts or mutate account balances for invoices/payments.

## Next Tickets

1. Add controlled posting action for generated drafts, reusing `PostJournalEntry` semantics after review.
2. Add batch backfill workflow for historical documents missing journal links.
3. Add drill-down lists from coverage rows to the affected source documents.
4. Add period-lock checks before draft creation/posting.
5. Add support-bundle export for posting coverage and trial-balance evidence.
