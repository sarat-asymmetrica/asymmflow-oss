# Domain Model Alias Plan

Date: 2026-05-05

## Purpose

The target architecture wants generated domain models under `pkg/finance`, `pkg/crm`, `pkg/documents`, `pkg/butler`, `pkg/sync`, and `pkg/infra`, with a thin application shell exposing Wails bindings.

Today, the root package still owns the canonical GORM structs in `database.go`. Several package-level domain files already mirror those types, but most services cannot fully move logic into packages yet because Go packages cannot import `package main`, and importing `main` models from domain packages would invert the target dependency graph.

The next bridge is a controlled type-alias migration: keep the Wails method signatures stable while making selected root types aliases of package-owned domain types.

## Current Constraint

- `database.go` defines the real structs used by GORM migrations, Wails bindings, and existing root services.
- `pkg/finance/domain.go` and `pkg/crm/domain.go` duplicate many of those structs.
- Existing extracted services use generics to avoid importing `package main`; this is a useful bridge, but it keeps model ownership ambiguous.
- A domain package cannot safely own logic until it also owns the model types used by that logic.

## Alias Direction

Use aliases in the root package:

```go
type Payment = finance.Payment
type SupplierPayment = finance.SupplierPayment
```

This preserves frontend-facing Go type names while moving ownership into the package. The root package remains the Wails adapter layer, not the model source of truth.

## Readiness Gate

Before aliasing any model family:

- The package model must match the root model field-for-field for persisted fields.
- GORM tags and JSON tags must be identical unless a migration deliberately changes them.
- `TableName()` methods, hooks, embedded `Base`, soft-delete behavior, and indexes must be checked.
- The model must not depend on another root-only type through a field, slice, or hook.
- `go build ./...` must pass after the alias.
- `go test ./... -count=1 -timeout 300s` must pass before committing.
- Wails generated bindings must be regenerated/tested in a frontend-aware wave if exported shape changes.

## Batch Order

### Batch 1: Leaf Finance Models

Good first candidates:

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

Why: these already have service facades and comparatively narrow relationships. They provide quick signal on whether the alias approach works with GORM, tests, and Wails.

### Batch 2: Banking Models

Candidates:

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

Why: Wave 3 and Wave 4 now route cash, reconciliation, matcher, statement CRUD, and audit/integrity entry points through `pkg/finance/banking`. After aliasing, the next wave can move SQL bodies into that package rather than just forwarding through handlers.

### Batch 3: Fulfillment and Procurement

Candidates from `pkg/crm/domain.go`:

- `DeliveryNote`
- `DeliveryNoteItem`
- `SerialNumber`
- `PurchaseOrder`
- `PurchaseOrderItem`
- `GoodsReceivedNote`
- `GRNItem`
- `Warehouse`
- `InventoryItem`
- `StockMovement`
- `StockAdjustment`

Why: Wave 2 already added fulfillment/procurement service facades. These models are coupled to orders/products, so alias them in a smaller slice than banking.

### Batch 4: CRM and Pipeline

Candidates:

- `CustomerContact`
- `SupplierContact`
- `EntityNote`
- `SupplierIssue`
- `ProductMaster`
- `Opportunity`
- `OfferFollowUp`
- `OfferNote`
- `GradeChange`
- `FollowUpTask`

Why: these unlock `app_sales_pipeline.go`, `app_crm_surface.go`, and the customer/order adapters, but they touch more frontend state and sync flows.

### Batch 5: High-Coupling Financial Documents

Defer until the lower batches are proven:

- `Invoice`
- `DBInvoiceItem`
- `CreditNote`
- `CreditNoteItem`
- `Offer`
- `OfferItem`
- `Order`
- `OrderItem`
- `CustomerMaster`
- `SupplierMaster`
- `SupplierInvoice`
- `SupplierInvoiceItem`

Why: these are the business spine. They have PDF generation, costing, sync, customer history, accounting side effects, and Wails/TypeScript surface area. Alias them only after the package-domain pattern is boring.

### Batch 6: Infra/System Models

Defer until an infra domain owns the exact structs:

- `Role`
- `User`
- `Device`
- `DeviceUser`
- `LicenseKey`
- `Setting`
- `SyncRecord`
- `AuditLog`
- `Job`
- `BackupPolicy`

Why: auth, license, sync, migration, and startup code still live close to `App`. These should move with an infra package boundary, not as isolated aliases.

## Do Not Alias Yet

- Any model with a root-only hook, custom migration dependency, or hidden table-name behavior.
- Any model whose package duplicate differs from `database.go`.
- Any model used by generated Wails TypeScript where field tags or optionality differ.
- Any encrypted or credential-bearing model until field crypto and redaction paths are checked.

## Mechanical Protocol

For each batch:

1. Compare root and package structs.
2. Fix package struct drift first.
3. Replace root struct definition with a type alias.
4. Keep root-only helper methods beside the alias only when Go allows it; otherwise move methods with the type.
5. Run `gofmt`.
6. Run `go build ./...`.
7. Run `go test ./... -count=1 -timeout 300s`.
8. Commit one model family at a time.

## Wave 5 Recommendation

Start with Batch 1 and Batch 2. The ideal Wave 5 outcome is:

- `Payment` and supplier payment models owned by `pkg/finance`.
- Expense models owned by `pkg/finance`.
- Banking models owned by `pkg/finance`.
- Banking SQL bodies moved from root helper functions into `pkg/finance/banking` once aliases compile cleanly.

That is the smallest step that converts the service facades from adapter shells into real domain packages.
