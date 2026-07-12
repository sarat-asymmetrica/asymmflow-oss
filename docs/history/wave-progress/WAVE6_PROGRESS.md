# Wave 6 Progress

Date: 2026-05-06
Scope source: `CODEX_WAVE6_HANDOFF.md`

## Summary

Wave 6 continued the package-boundary refactor in two lanes:

- Banking mutation ownership moved deeper into `pkg/finance/banking`, with root Wails methods preserved as thin facades.
- CRM model ownership moved deeper into `pkg/crm`, replacing more root structs with aliases while preserving table names.

## Commits

- `0a7733a refactor(codex): define banking dependency ports`
- `cf901b3 refactor(codex): move bank statement mutations into banking service`
- `3a69ced refactor(codex): move bank statement line mutations into banking service`
- `3afedad refactor(codex): move banking integrity helpers into service`
- `7cb12ab refactor(codex): alias CRM offer pipeline models`
- `6f01045 refactor(codex): alias CRM order product models`
- `113ef9a refactor(codex): alias CRM contact leaf models`

## Banking Extraction

Added package-facing dependency ports in `pkg/finance/banking/ports.go`:

- `AuthorizationPort`
- `AuditPort`
- `FinancialAuditPort`
- `DivisionPort`
- `DeleteApprovalPort`

Added root adapters in `app_banking_ports.go` and wired them through `app_services.go`.

Moved these statement flows into `pkg/finance/banking/service.go`:

- `CreateBankStatement`
- `UpdateBankStatement`
- `DeleteBankStatement`
- `CreateBankStatementLine`
- `UpdateBankStatementLine`
- `DeleteBankStatementLine`

Moved these integrity/audit helpers into the banking service:

- `ValidateStatementContinuity`
- `ComputeStatementHash`
- `CheckDuplicateStatement`
- `SaveStatementHash`
- `ForceReimportStatement`
- `LogReconciliationAction`
- `ReverseAction`

Kept the public `App` method names stable for Wails bindings.

Deferred banking items:

- Matching/allocation flows: `AutoMatchBankLines`, `ManualMatchLine`, `UnmatchLine`, `CreateSplitAllocation`
- Reconciliation lifecycle/reporting handlers: `ValidateStatementBalance`, `FinalizeReconciliation`, `ReopenReconciliation`, summaries/stats
- `GetBalanceContinuityReport`, because it still depends on root `CompanyBankAccount`

## CRM Aliases

Moved TableName ownership into `pkg/crm/domain.go` and replaced the root structs with aliases for:

- `Offer`
- `OfferItem`
- `Opportunity`
- `OfferFollowUp`
- `OfferNote`
- `FollowUpTask`
- `Order`
- `OrderItem`
- `ProductMaster`
- `GradeChange`
- `CustomerContact`
- `SupplierContact`
- `EntityNote`
- `SupplierIssue`

Current `database.go` shape after Wave 6:

- 1,312 lines
- 51 root struct definitions
- 39 root aliases

## Verification

Each tranche was verified before commit with:

```powershell
$env:GOTMPDIR='D:\go-tmp'
$env:GOCACHE='D:\go-cache'
go build ./...
go test ./... -count=1 -timeout 300s
```

The final Go test pass completed cleanly:

- `ph_holdings_app`: `ok`
- `ph_holdings_app/integration`: `ok`
- all package tests: `ok` or `[no test files]`

## Notes For Wave 7

Recommended next work:

- Continue banking extraction with reconciliation lifecycle methods before matching/allocation.
- Move `CompanyBankAccount` or introduce a bank-account read port so `GetBalanceContinuityReport` can leave root.
- Consider extracting bank matching/allocation as a dedicated tranche because it touches invoices, supplier invoices, expenses, payroll, and audit logs.
- Continue model aliasing with the remaining low-risk root models, while still avoiding the deferred invoice/customer/supplier core called out in the Wave 6 handoff.
