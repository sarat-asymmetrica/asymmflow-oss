# Wave 7 Progress

Date: 2026-05-06
Scope source: `CODEX_WAVE7_HANDOFF.md`

## Summary

Wave 7 finished the next major tranche of the package-boundary refactor:

- Banking reconciliation lifecycle, balance continuity, matching, unmatching, and split allocation moved into `pkg/finance/banking`.
- Finance and CRM/infra model ownership moved further out of root `database.go` via aliases.
- SQLite moved from `mattn/go-sqlite3` / `gorm.io/driver/sqlite` to `ncruces/go-sqlite3` / `gormlite`, eliminating the primary CGO SQLite dependency.
- The ncruces migration exposed two test-runtime bottlenecks; both were fixed so the required 300s Go gate passes again.

## Commits

- `a90e74d refactor(codex): move reconciliation lifecycle into banking service`
- `349c528 refactor(codex): resolve bank account dependency and move continuity report`
- `29e947e refactor(codex): move banking matching and allocation into service`
- `088ec54 refactor(codex): alias Invoice and CreditNote models to pkg/finance`
- `356c45b refactor(codex): alias SupplierInvoice models to pkg/finance`
- `6e10b3a refactor(codex): replace mattn/go-sqlite3 with ncruces/go-sqlite3 (eliminate CGO)`
- `ebf4ca0 refactor(codex): alias remaining CRM and Infra models`

## Banking Extraction

Moved these reconciliation lifecycle/reporting flows into `pkg/finance/banking/service.go`:

- `ValidateStatementBalance`
- `FinalizeReconciliation`
- `ReopenReconciliation`
- `GetReconciliationStats`
- `GetReconciliationSummary`
- `GetBalanceContinuityReport`

Moved these matching/allocation flows into `pkg/finance/banking`:

- `AutoMatchBankLines`
- `ManualMatchLine`
- `UnmatchLine`
- `CreateSplitAllocation`

Added the bank-account model alias so package code can own balance-continuity lookups:

- `CompanyBankAccount = finance.CompanyBankAccount`

Root Wails methods remain as compatibility facades. Remaining banking work for Wave 8 is mainly cash-position helper extraction, because `GetCashPosition` delegates to package code that still calls root helper functions.

## Model Aliases

Added or completed aliases for:

- Finance: `Invoice`, `DBInvoiceItem`, `CreditNote`, `CreditNoteItem`, `SupplierInvoice`, `SupplierInvoiceItem`, `SupplierPayment`, `CompanyBankAccount`
- CRM/master data: `CustomerMaster`, `SupplierMaster`, `InventoryItem`, `StockMovement`, `StockAdjustment`, `Warehouse`
- Infra: `Setting`, `AuditLog`, `Job`

Moved table-name ownership for those CRM/infra models into package domain files.

Current root model shape:

- `database.go`: 944 lines
- Root aliases in `database.go`: 55
- Remaining root struct definitions in `database.go`: 35
- Root `func (a *App)` methods across Go files: 1,187

## SQLite Driver

Replaced the CGO SQLite stack with ncruces:

- GORM dialector: `github.com/ncruces/go-sqlite3/gormlite`
- `database/sql` driver: `github.com/ncruces/go-sqlite3/driver`
- Removed `gorm.io/driver/sqlite`, `gorm.io/datatypes`, and `github.com/mattn/go-sqlite3` references from Go files, `go.mod`, and `go.sum`.

Compatibility fixes applied:

- Windows database paths now use URI `file:` DSNs.
- Test databases use ncruces `memdb` where shared in-memory databases are required.
- `GraphNode`/`GraphEdge` JSON fields now use `json.RawMessage` via a local alias instead of `gorm.io/datatypes.JSON`.
- `EnsurePayrollFoundation` skips duplicate `AutoMigrate` passes when tables already exist.
- The opportunity-conflict stress test now uses `memdb` instead of temporary on-disk DBs to avoid Windows VFS lock overhead under concurrent load.

## Package LOC Snapshot

Selected package sizes after Wave 7:

- `pkg/finance`: 16 files, 3,008 LOC
- `pkg/crm`: 10 files, 874 LOC
- `pkg/infra`: 15 files, 381 LOC
- `pkg/domain`: 1 file, 22 LOC
- `pkg/graph`: 3 files, 1,284 LOC
- `pkg/ocr`: 60 files, 18,959 LOC

Root Go files outside `pkg`: 114,455 LOC.

## Verification

Each code tranche was verified before commit with the Wave gate:

```powershell
$env:GOTMPDIR='D:\go-tmp'
$env:GOCACHE='D:\go-cache'
go build -tags='' ./...
go test ./... -count=1 -timeout 300s
```

Final pass:

- `go build -tags='' ./...`: pass
- `go test ./... -count=1 -timeout 300s`: pass
- Root package: `ok ph_holdings_app 241.647s`
- All subpackages: `ok` or `[no test files]`

## Notes For Wave 8

Recommended next work:

- Extract cash-position snapshot helpers from root into `pkg/finance/banking`.
- Continue shrinking `database.go` by targeting the remaining low-risk root structs before deep auth/license/sync models.
- Consider moving payroll, expense, and activity-monitoring models into package domain files now that ncruces test hot spots are understood.
- Start reducing the 1,187 root `App` methods by grouping thin Wails facades into generated or package-owned API surfaces.
