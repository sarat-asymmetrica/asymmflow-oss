# Wave 5 Progress

Date: 2026-05-05

## Summary

Wave 5 implemented the domain model alias bridge for the low- and medium-coupling finance and CRM model families.

The key architectural change is that root-facing model names remain available to Wails and existing services, but many of those names now alias package-owned domain types in `pkg/finance`, `pkg/crm`, and the new shared `pkg/domain` base package.

## Verification

Every code tranche passed both gates:

- `go build ./...`
- `go test ./... -count=1 -timeout 300s`

Because the Windows C: drive was nearly full during the first build, Go temp/cache were pointed at D: for the rest of the wave:

```powershell
$env:GOTMPDIR='D:\go-tmp'
$env:GOCACHE='D:\go-cache'
```

## Commits

- `e1c7fa7` - share domain base model
- `d54aa9c` - alias Payment models to pkg/finance
- `afafad1` - alias Expense models to pkg/finance
- `4113cef` - alias FX models to pkg/finance
- `870769a` - alias Banking models to pkg/finance
- `7b85270` - move banking read queries into pkg/finance/banking
- `d8e08bc` - alias Fulfillment models to pkg/crm
- `0d23387` - alias Procurement models to pkg/crm

## Alias Counts

Current `database.go`:

- Lines: 1,617
- Type aliases: 25
- Remaining struct definitions: 65

Alias totals created this wave:

- 24 model aliases in `database.go`
- 1 shared `Base` alias in `database.go`
- 4 expense model aliases in `expense_service.go`
- 29 root-facing aliases total

## Aliased Finance Models

- `Payment`
- `SupplierPayment`
- `ExpenseCategory`
- `ExpenseVendor`
- `ExpenseEntry`
- `RecurringExpense`
- `BankExpenseEntry`
- `CurrencyExchangeRate`
- `FXRate`
- `FXRevaluation`
- `BankStatement`
- `BankStatementLine`
- `BankLinePaymentAllocation`
- `BankCashBalance`
- `StatementHash`
- `BookBankReconciliation`
- `OutstandingCheque`
- `DepositInTransit`
- `ChequeRegister`
- `BankStatementFile`
- `BankReconciliationAuditLog`

## Aliased CRM Models

- `DeliveryNote`
- `DeliveryNoteItem`
- `SerialNumber`
- `PurchaseOrder`
- `PurchaseOrderItem`
- `GoodsReceivedNote`
- `GRNItem`

The `GRNItem.BeforeSave` hook moved with the owned type into `pkg/crm/domain.go`.

## Shared Base

Created `pkg/domain.Base` and made root, finance, and CRM `Base` names aliases of it.

This was necessary because package-owned aliases embed their package `Base`; without a shared base, existing literals such as `Payment{Base: Base{ID: ...}}` would stop compiling after aliasing.

## Banking Logic Move

`pkg/finance/banking` now owns real SQL read logic over package-owned finance models for:

- `GetBankStatements`
- `GetBankStatementByID`
- `GetBankStatementLines`
- `GetUnmatchedLines`
- `GetAuditTrail`
- `GetAuditTrailByDateRange`

The corresponding root helper bodies were removed. Wails-facing `*App` methods still delegate through `a.bankingService()`.

## Deferred Banking Logic

The mutating banking flows still delegate to root helper functions:

- statement create/update/delete
- statement line edit/create/delete
- reconciliation finalization/reopen/summary/stats
- matching, unmatching, split allocation, categorization
- continuity report, hash save/check, force reimport, reverse action

Reason: these paths still depend on root-only policy/dependency seams such as delete approval, current user identity, payroll unlinking, audit logger, division resolution, and matcher helpers. The aliases now make the next move possible, but those dependencies should be passed as explicit ports rather than dragged into the package.

## Remaining High-Coupling Models

Wave 6 should handle the business spine carefully:

- `CustomerMaster`
- `SupplierMaster`
- `ProductMaster`
- `Offer`
- `OfferItem`
- `Opportunity`
- `Order`
- `OrderItem`
- `Invoice`
- `DBInvoiceItem`
- `CreditNote`
- `CreditNoteItem`
- `SupplierInvoice`
- `SupplierInvoiceItem`

These touch PDFs, costing, sync, Butler, customer history, accounting side effects, and frontend shape. They should move one family at a time.

## Wave 6 Recommendation

1. Introduce explicit banking ports for authorization, delete approval, actor identity, audit logging, division resolution, payroll unlinking, and bank-line parsing helpers.
2. Move the remaining banking mutating SQL bodies into `pkg/finance/banking` behind those ports.
3. Alias and extract `Offer`/`Opportunity`/`Order` as the next CRM pipeline spine.
4. Keep `Invoice` and `CustomerMaster` for the last slice of the spine because they have the broadest blast radius.
