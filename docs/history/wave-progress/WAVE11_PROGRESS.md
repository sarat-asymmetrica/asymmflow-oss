# Wave 11 Progress Report

Date: 2026-05-07

## Summary

Wave 11 created the `internal/viewmodel` layer as the display-ready contract between AsymmFlow and future consumers: Svelte screens, per-screen agents, lab workflows, marketplace modules, and API clients.

The layer is additive. Existing Wails methods, Svelte files, generated schemas, domain structs, domain services, and Wave 10 adapters were left untouched.

## Commits

| Ticket | Status | Commit |
| --- | --- | --- |
| 1. Package structure | Complete | `efc38a5 feat(codex): create ViewModel package structure` |
| 2. Shared ViewModels | Complete | `1279d65 feat(codex): create shared ViewModel types` |
| 3. Finance ViewModels | Complete | `d91b6d2 feat(codex): create Finance ViewModels with builders` |
| 4. CRM ViewModels | Complete | `d830a65 feat(codex): create CRM ViewModels with builders` |
| 5. Butler ViewModels | Complete | `f5572fb feat(codex): create Butler ViewModels` |
| 6. Documents ViewModels | Complete | `e8b82f0 feat(codex): create Documents ViewModels` |
| 7. Dashboard + Settings ViewModels | Complete | `f8ed436 feat(codex): create Dashboard and Settings ViewModels` |
| 8. InvoiceListVM Wails endpoint | Complete | `5a2ef99 feat(codex): wire InvoiceListVM to Wails endpoint` |
| 9. Progress report | Complete | this document |

## Counts

- ViewModel types defined: 76
- Builder functions added: 7
- Tests added: 4
- Pilot Wails endpoint added: 1

Builder functions:

- `finance.BuildInvoiceListVM`
- `finance.BuildInvoiceDetailVM`
- `finance.BuildCashPositionVM`
- `crm.BuildCustomerListVM`
- `crm.BuildPipelineVM`
- `crm.BuildPipelineSnapshotVM`
- `crm.BuildOrderListVM`

Tests:

- `TestBuildInvoiceListVMFormatsDisplayValues`
- `TestBuildInvoiceDetailVMFormatsItemsAndActions`
- `TestBuildCustomerListVMFormatsRowsAndGrades`
- `TestBuildPipelineVMComputesValueAndWinRate`

## Screens Covered

Shared contracts:

- Generic lists
- Tables
- Dashboard shells
- Status badges
- Form fields
- Summary cards
- Actions and breadcrumbs

Finance:

- Invoice list
- Invoice detail
- Bank reconciliation
- Cash position
- Expense dashboard
- Payroll summary
- Financial dashboard

CRM:

- Customer list
- Customer detail
- Customer 360
- Pipeline
- Offer detail
- Order list
- Order detail
- Supplier dashboard

Butler:

- Chat
- Conversation list
- Daily briefing
- Prediction cards
- Insight cards

Documents:

- Document upload
- OCR result
- Inbox
- PDF preview

Dashboard and settings:

- Main dashboard
- Settings sections and fields

## Pilot Endpoint

Added:

```go
func (a *App) GetInvoiceListVM(page, pageSize int) (financevm.InvoiceListVM, error)
```

Behavior:

- Computes `offset` from `page` and `pageSize`.
- Calls existing `ListCustomerInvoices(pageSize, offset)`.
- Reuses existing invoice permissions and pagination bounds from that method.
- Converts the returned domain invoices into display-ready `InvoiceListVM`.

Wails bindings were not regenerated in this wave. The Go method exists and passes build/test, ready for a future UI-facing pass.

## Design Notes

All ViewModels follow the Wave 11 display-ready rules:

- Money is formatted as strings such as `BHD 1,234.50`.
- Dates are human-readable strings such as `1 May 2026`.
- IDs are strings.
- Statuses use `StatusBadgeVM`.
- Contextual actions are included where builders have enough context.

Dashboard composition note:

The handoff sketch placed `MainDashboardVM` in the root `internal/viewmodel` package with direct fields of `finance.CashPositionVM` and `crm.PipelineSnapshotVM`. In Go, child ViewModel packages already import the root package for shared contracts like `ActionButton`; importing child packages back into the root would create an import cycle. Wave 11 therefore uses `PanelRefVM` for composed dashboard child panels while preserving the root file placement and JSON shape intent.

## Validation

After every ticket, the required gate was run:

```powershell
$env:GOTMPDIR='D:\go-tmp'
$env:GOCACHE='D:\go-cache'
New-Item -ItemType Directory -Force -Path $env:GOTMPDIR,$env:GOCACHE | Out-Null
go build -tags='' ./...
go test ./... -count=1 -timeout 300s
```

Environment note:

- A stale Go linker temp directory in `C:\Users\YourName\AppData\Local\Temp\go-link-*` caused one early `pkg/ocr/fitz` link failure.
- That stale temp directory was removed after verifying it was inside the user temp root.
- Root package tests time out if `%TEMP%/%TMP%` are forced onto the external drive because SQLite/PDF-heavy tests slow down. The stable setup is `GOTMPDIR/GOCACHE` on `D:` with normal local `%TEMP%/%TMP%`.

Final observed gate after Ticket 8:

```text
go build -tags='' ./...: pass
go test ./... -count=1 -timeout 300s: pass
```

## Follow-Up Candidates

- Regenerate Wails bindings for `GetInvoiceListVM` when the frontend is ready to consume it.
- Add screen-specific builders for CustomerDetail, OfferDetail, OrderDetail, BankReconciliation, OCRResult, and Butler Chat once the data-loading paths are selected.
- Consider a dedicated `internal/viewmodel/dashboard` package if dashboard composition needs strong typed child package fields instead of `PanelRefVM`.
- Add Proto-to-ViewModel builders where the runtime starts consuming Wave 10 Cap'n Proto messages directly.
