# Codex Autonomous Execution Spec — Wave 9: Cap'n Proto Schemas

**Date**: 2026-05-06
**From**: Claude (Opus 4.6, Senior Architect) + the maintainer
**To**: Codex (GPT-5.5, Senior Architect)
**Run Target**: Autonomous until complete
**Previous Runs**: Waves 0-8B complete. 57 commits. Extraction ERA complete. 6 domain packages with real logic. butler_ai.go 1,719 LOC, database.go 5 structs / 85 aliases, go-fitz isolated. Tests GREEN.
**Build Verification**: `go build ./...` and `go test ./... -count=1 -timeout 300s` after every ticket.
**Disk space**: Use `$env:GOTMPDIR='D:\go-tmp'` and `$env:GOCACHE='D:\go-cache'`.

---

## 0. Context — The Construction Era Begins

Waves 2-8B were DEMOLITION — breaking the God Object into domain packages.
Wave 9 is the first CONSTRUCTION wave — creating the canonical type system.

Cap'n Proto schemas become the **single source of truth** for all domain types:
- Every struct gets a permanent schema with numbered fields
- Go types are generated ALONGSIDE existing GORM structs (not replacing them)
- TypeScript interfaces are generated for frontend type safety
- Schema evolution rules prevent breaking changes forever

**Installed tools**:
- `capnp` v1.3.0 — `C:\ProgramData\chocolatey\bin\capnp.exe`
- `capnpc-go` — `C:\Users\YourName\go\bin\capnpc-go.exe`

---

## 1. Tickets

### Dependency Graph

```
Ticket 1 (Toolchain verification) → ALL other tickets
Ticket 2 (common.capnp) → Tickets 3-8 (all schemas import common)
Tickets 3-8 (domain schemas) → independent of each other
Ticket 9 (Generation pipeline) → after all schemas
Ticket 10 (Generate + verify Go) → after pipeline
Ticket 11 (Generate TypeScript) → after pipeline
Ticket 12 (Progress audit) → last
```

---

### Ticket 1: Verify Toolchain

**Deliverables**:
1. Create `schemas/` directory
2. Create a minimal test schema `schemas/test.capnp` with one struct
3. Run `capnp compile -ogo schemas/test.capnp` — verify it exits 0
4. Run `capnpc-go` on the output — verify Go code generates
5. Delete the test schema after verification

If `capnp` or `capnpc-go` are not on PATH, check:
- `C:\ProgramData\chocolatey\bin\capnp.exe`
- `C:\Users\YourName\go\bin\capnpc-go.exe`

If the tools are missing or broken, document the error and STOP — don't write schemas without a working compiler.

**Checklist**:
- [ ] `capnp compile` runs successfully
- [ ] Go code generation works
- [ ] `schemas/` directory exists

**Commit**: `build(codex): verify capnp toolchain and create schemas directory`

---

### Ticket 2: Write common.capnp

**Deliverables**: Create `schemas/common.capnp` with shared base types.

```capnp
@0x...; # Generate a unique ID with: capnp id

using Go = import "/go.capnp";
$Go.package("common");
$Go.import("ph_holdings_app/schemas/common");

struct Base {
  id        @0 :UInt64;
  createdAt @1 :Text;  # ISO 8601
  updatedAt @2 :Text;
  deletedAt @3 :Text;  # Empty string = not deleted
  version   @4 :UInt32;
  createdBy @5 :Text;
}

enum Division {
  phTrading      @0;
  phTradingAlumi @1;
  both           @2;
}

# ... add more shared enums: Status, Priority, Currency, etc.
```

**Source**: `pkg/domain/base.go` for the Base struct. Scan all domain files for commonly used enums (Status strings that appear in 3+ domains should become shared enums).

Common enums to extract:
- `Division` — phTrading, phTradingAlumi, both
- `Currency` — bhd, usd, eur, gbp, inr, etc.
- `DocumentStatus` — draft, submitted, approved, rejected, posted, cancelled
- `PaymentMethod` — bankTransfer, cheque, cash, card
- `MatchStatus` — unmatched, matched, partialMatch, disputed

**Checklist**:
- [ ] `capnp compile schemas/common.capnp` exits 0
- [ ] Base struct has permanent field numbers
- [ ] At least 5 shared enums defined

**Commit**: `schema(codex): write common.capnp with shared base types and enums`

---

### Ticket 3: Write finance.capnp

**Deliverables**: Create `schemas/finance.capnp` with ALL finance domain types.

**Source files to read**:
- `pkg/finance/domain.go` — main finance types
- `pkg/finance/banking/domain.go` or `pkg/finance/banking/service.go` — banking-specific types

**Types to include** (45 total, grouped):

**Core Invoicing**:
- Invoice, DBInvoiceItem, InvoiceSequence
- CreditNote, CreditNoteItem
- Payment

**Supplier**:
- SupplierInvoice, SupplierInvoiceItem, SupplierPayment

**Procurement**:
- PurchaseOrder, PurchaseOrderItem

**Expense**:
- ExpenseCategory, ExpenseVendor, ExpenseEntry, RecurringExpense
- ExpenseDashboardSummary, BankExpenseEntry

**Accounting**:
- ChartOfAccount, JournalEntry, JournalLine
- VATReturn, FiscalPeriod, AccountMapping

**Banking**:
- CompanyBankAccount, BankAccount
- BankStatement, BankStatementLine, BankStatementFile
- BankLinePaymentAllocation, BankCashBalance
- StatementHash, BalanceGap, BalanceContinuityReportData
- BankReconciliationAuditLog
- BookBankReconciliation, OutstandingCheque, DepositInTransit
- ChequeRegister

**FX**:
- CurrencyExchangeRate, FXRate, FXRevaluation
- ARAgingBucket

**Rules**:
- Import common.capnp for Base
- Every struct MUST have permanent field numbers starting at @0
- Use `Text` for all string fields
- Use `Float64` for all monetary amounts (BHD values)
- Use `Text` for dates (ISO 8601 format)
- Use `UInt64` for foreign key IDs
- Use `List(ItemType)` for embedded item arrays
- Group related types with comments

**Checklist**:
- [ ] `capnp compile schemas/finance.capnp` exits 0
- [ ] All 45 finance types have schemas
- [ ] Field numbers are permanent and sequential per struct

**Commit**: `schema(codex): write finance.capnp with 45 domain types`

---

### Ticket 4: Write crm.capnp

**Deliverables**: Create `schemas/crm.capnp` with ALL CRM domain types.

**Source**: `pkg/crm/domain.go`

**Types to include** (35 total):

**Customer/Supplier Master**:
- CustomerMaster, CustomerContact, SupplierMaster, SupplierContact
- EntityNote, SupplierIssue, GradeChange

**Products & Inventory**:
- ProductMaster, InventoryItem, StockMovement, StockAdjustment, Warehouse
- SerialNumber, CostingHistory

**Sales Pipeline**:
- Opportunity, Offer, OfferItem, OfferFollowUp, OfferNote
- DBCostingSheet, DBCostingItem, DBCostingAdditionalCost, CostingLineItemData

**Orders & Fulfillment**:
- Order, OrderItem, Shipment, PostSaleNote
- DeliveryNote, DeliveryNoteItem
- GoodsReceivedNote, GRNItem
- FollowUpTask

**Same rules as finance.capnp** for field types and numbering.

**Checklist**:
- [ ] `capnp compile schemas/crm.capnp` exits 0
- [ ] All 35 CRM types have schemas
- [ ] No duplicate definitions with finance.capnp (PurchaseOrder is in finance only)

**Commit**: `schema(codex): write crm.capnp with 35 domain types`

---

### Ticket 5: Write butler.capnp

**Deliverables**: Create `schemas/butler.capnp` with Butler domain types.

**Source**: `pkg/butler/domain.go`, `pkg/butler/ports.go`

**Types to include** (13):
- ButlerResponse, ButlerResponseMetadata, ButlerAction
- Intent
- ButlerResolvedEntity
- PredictionRecord, WinProbabilityPrediction, DiscountRecommendationRecord
- CustomerSnapshot, ActualOutcome, PaymentPredictionAccuracy
- Conversation, ChatMessage

**Note**: DO NOT schema-ify the port interfaces (DatabasePort, LLMPort, etc.) — those are implementation contracts, not data types.

**Checklist**:
- [ ] `capnp compile schemas/butler.capnp` exits 0
- [ ] All 13 Butler data types have schemas
- [ ] Conversation and ChatMessage are the anchor types for chat persistence

**Commit**: `schema(codex): write butler.capnp with 13 domain types`

---

### Ticket 6: Write documents.capnp

**Deliverables**: Create `schemas/documents.capnp` with Documents domain types.

**Source**: `pkg/documents/domain.go`, `pkg/documents/ports.go`

**Types to include**:
- CompanyInfo, BrandingConfig (from ports.go — these are data types, not interfaces)
- BankStatementFile (document metadata)
- OCRResult (general OCR output shape)
- ClassificationResult (document classification)
- Add any additional document-related types found in `pkg/documents/*/`

Also check `pkg/ocr/orchestrator/` for ProcessRequest, ProcessResponse, BatchRequest, BatchResponse — include these as they're the OCR API contract.

**Checklist**:
- [ ] `capnp compile schemas/documents.capnp` exits 0
- [ ] OCR request/response types captured

**Commit**: `schema(codex): write documents.capnp with document and OCR types`

---

### Ticket 7: Write infra.capnp

**Deliverables**: Create `schemas/infra.capnp` with infrastructure types.

**Source**: `pkg/infra/domain.go` or scan `pkg/infra/**/*.go` for type definitions.

**Types to include** (10):
- User, Role, Device, DeviceUser
- UserSession
- Setting, AuditLog, Job
- Alert, BackupPolicy

**Checklist**:
- [ ] `capnp compile schemas/infra.capnp` exits 0
- [ ] All auth/admin types have schemas

**Commit**: `schema(codex): write infra.capnp with 10 infrastructure types`

---

### Ticket 8: Write sync.capnp

**Deliverables**: Create `schemas/sync.capnp` with sync/import types.

**Source**: `database.go` (the 5 remaining root structs)

**Types to include** (4-5):
- FileWatchEvent
- SyncStatus, SyncRecord
- TallyInvoiceImport, TallyPurchaseImport

These are the last 5 root structs in database.go. Giving them a schema home here also prepares for future `pkg/sync/` package creation.

**Checklist**:
- [ ] `capnp compile schemas/sync.capnp` exits 0

**Commit**: `schema(codex): write sync.capnp with sync and import types`

---

### Ticket 9: Create Generation Pipeline

**Deliverables**: Create `schemas/generate.ps1` (PowerShell script for Windows):

```powershell
# schemas/generate.ps1 — Generate Go + TypeScript from Cap'n Proto schemas
$ErrorActionPreference = "Stop"

$schemas = @(
    "common", "finance", "crm", "butler",
    "documents", "infra", "sync"
)

# Create output directories
New-Item -ItemType Directory -Force -Path "schemas/go" | Out-Null
New-Item -ItemType Directory -Force -Path "schemas/ts" | Out-Null

foreach ($schema in $schemas) {
    Write-Host "Compiling $schema.capnp..."
    capnp compile -ogo:"schemas/go" "schemas/$schema.capnp"
    if ($LASTEXITCODE -ne 0) { throw "capnp compile failed for $schema" }
}

Write-Host "All schemas compiled successfully."
Write-Host "Go types in: schemas/go/"
```

Also create a TypeScript type generator. Options (in priority order):
1. If `capnpc-ts` is available, use it
2. Otherwise, write a simple PowerShell script that reads .capnp files and generates TypeScript interfaces (struct → interface, enum → union type)
3. If neither works, document the gap for manual TypeScript generation later

**Checklist**:
- [ ] `schemas/generate.ps1` runs without error
- [ ] All 7 schemas compile
- [ ] Output directories created

**Commit**: `build(codex): create schema generation pipeline`

---

### Ticket 10: Generate Go Types and Verify Build

**Deliverables**:
1. Run `schemas/generate.ps1`
2. Generated Go files land in `schemas/go/`
3. Create `schemas/go/go_package.go` if needed to declare the Go package
4. Verify `go build ./...` still passes (generated code doesn't conflict with existing GORM structs)

**IMPORTANT**: The generated Go types live in `schemas/go/` — they are SEPARATE from the GORM structs in `pkg/*/domain.go`. Both coexist. The generated types are for transport/API, the GORM types are for persistence. Future waves will bridge them.

If `go build` fails due to naming conflicts:
- Generated types in `schemas/go/` use package name `schemas` — no conflict expected
- If there IS a conflict, namespace the generated package differently

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Generated Go files exist in `schemas/go/`

**Commit**: `build(codex): generate Go types from capnp schemas`

---

### Ticket 11: Generate TypeScript Types

**Deliverables**:
1. Generate TypeScript interfaces from schemas
2. Output to `frontend/src/lib/types/schemas/`
3. Each schema file → one `.ts` file (e.g., `finance.ts`, `crm.ts`)

If a TypeScript Cap'n Proto generator is not available, write a PowerShell script that:
- Reads each `.capnp` file
- Extracts struct definitions
- Generates TypeScript interfaces with matching field names
- Generates TypeScript enums from Cap'n Proto enums

Example output:
```typescript
// Generated from schemas/finance.capnp — DO NOT EDIT
export interface Invoice {
  id: number;
  invoiceNumber: string;
  customerID: number;
  grandTotalBHD: number;
  status: string;
  dueDate: string;
  items: DBInvoiceItem[];
}
```

This is a stretch ticket — if the TypeScript generator is too complex to build in this wave, document the gap and move on.

**Checklist**:
- [ ] TypeScript files generated (or gap documented)
- [ ] `go build ./...` still passes
- [ ] `go test ./...` still passes

**Commit**: `build(codex): generate TypeScript types from capnp schemas`

---

### Ticket 12: Wave 9 Progress Audit

**Deliverables**:
1. Count of schema files created
2. Count of types defined across all schemas
3. capnp compile status (all pass?)
4. Go generation status (clean build?)
5. TypeScript generation status
6. List any types that were skipped and why
7. Write `docs/WAVE9_PROGRESS.md`

**Commit**: `docs(codex): write wave 9 progress report`

---

## 2. Schema Writing Rules

### Field Numbering
- Every field gets a permanent number: `@0`, `@1`, `@2`, ...
- Numbers are NEVER reused or changed once assigned
- If a field is removed in the future, its number is retired

### Type Mapping (Go → Cap'n Proto)

| Go Type | Cap'n Proto Type |
|---------|-----------------|
| `string` | `Text` |
| `int`, `int64` | `Int64` |
| `uint`, `uint64` | `UInt64` |
| `int32` | `Int32` |
| `float64` | `Float64` |
| `bool` | `Bool` |
| `time.Time` | `Text` (ISO 8601) |
| `[]byte` | `Data` |
| `[]ItemType` | `List(ItemType)` |
| `*ItemType` (pointer) | Regular field (Cap'n Proto fields are optional by default) |
| `map[string]interface{}` | Skip or use `List(KeyValue)` |
| `json.RawMessage` | `Data` |
| `gorm.DeletedAt` | `Text` (empty = not deleted) |

### Struct Naming
- Use PascalCase matching Go struct names exactly
- Group related structs with comment headers

### What to SKIP
- GORM tags (`gorm:"..."`) — not relevant to schema
- JSON tags — schema defines its own serialization
- Methods — only data fields
- Interfaces — schemas are for data types only
- `map[string]interface{}` fields — too dynamic for schemas, skip with comment
- Computed/transient fields (if any)

### Import Pattern
Every schema imports common:
```capnp
using Common = import "common.capnp";
```

And references Base as:
```capnp
struct Invoice {
  base @0 :Common.Base;
  invoiceNumber @1 :Text;
  # ...
}
```

---

## 3. Quality Gates

After EVERY ticket:
1. `capnp compile schemas/*.capnp` exits 0 (for schema tickets)
2. `go build ./...` exits 0 (for generation tickets)
3. `go test ./... -count=1 -timeout 300s` exits 0

### Special Rules
- If `capnp` is not on PATH, try the full path: `C:\ProgramData\chocolatey\bin\capnp.exe`
- If `capnpc-go` is not on PATH, try: `C:\Users\YourName\go\bin\capnpc-go.exe`
- If capnp toolchain fails completely in Ticket 1, STOP and write a detailed error report instead of proceeding with broken tools
- Generated code goes in `schemas/go/` — NEVER modify `pkg/*/domain.go`

---

## 4. Autonomy Contract

- Start with Ticket 1 (toolchain). If it fails, STOP with error report.
- Tickets 2-8 (schemas) can be done in any order after Ticket 2 (common must be first).
- Do NOT stop between tickets.
- STOP conditions: capnp toolchain broken; build fails after 3 fix attempts; disk full.
- **Priority**: Schemas (Tickets 2-8) are highest value. Generation (Tickets 9-11) are stretch. If running long, complete all schemas first.

---

## 5. What NOT To Touch

- `pkg/*/domain.go` — NEVER modify existing GORM structs
- `pkg/*/ports.go` — NEVER modify existing interfaces
- `database.go` — do not change
- Frontend Svelte files — no UI changes
- Any `.go` files outside `schemas/` directory

---

## 6. Expected Outcome

- 7 `.capnp` schema files in `schemas/`
- ~138 types defined with permanent field numbers
- All schemas compile with `capnp compile`
- Go types generated in `schemas/go/` (alongside, not replacing GORM)
- TypeScript interfaces generated in `frontend/src/lib/types/schemas/` (stretch)
- Generation pipeline script for future re-generation
- Build and tests remain GREEN

---

## Sign-Off

This is the FIRST construction wave. The schemas are the FOUNDATION that everything else builds on — Cap'n Proto schemas → generated types → MVVM ViewModels → Wails v3 services → Svelte 5 runes → Feature Team code generation.

Get the schemas right, with permanent field numbers, and every future wave gets safer. The compiler becomes the GUARDIAN of backwards compatibility.

🏗️ Construction begins. Schema first. Types generated. Build green. GO.
