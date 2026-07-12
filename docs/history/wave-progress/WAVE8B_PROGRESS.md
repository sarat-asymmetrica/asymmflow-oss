# Wave 8B Progress

Date: 2026-05-06

## Scope Completed

Wave 8B focused on finishing the Butler package boundary, starting the documents package boundary, and collapsing the remaining model definitions in `database.go`.

## Commits

- `6693aa6` - `refactor(codex): create ButlerAppContext port and adapter`
- `682705b` - `refactor(codex): extract grounded butler fastpaths`
- `24cb4e6` - `refactor(codex): split butler AI context builders`
- `7ae115f` - `refactor(codex): move M79 predictor to butler prediction`
- `4277ad3` - `refactor(codex): add documents PDF generator facade`
- `5c696f4` - `refactor(codex): extract document classifier engine`
- `5c17d75` - `refactor(codex): isolate OCR fitz engine`
- `9ae0fa5` - `refactor(codex): extract document Excel and email parsers`
- `4c01119` - `refactor(codex): finish database model alias pass`

## Ticket Notes

1. ButlerAppContext was added in `pkg/butler/ports.go`, with root adapter wiring in `app_butler_context.go` and `app_services.go`.
2. Grounded task and offer fastpath helpers moved into `pkg/butler/fastpath`; root wrappers preserve existing Wails-facing behavior and test wording.
3. `butler_ai.go` was reduced from 6,986 lines to 1,931 lines. App-dependent context builders were split into `butler_ai_context.go` with a dependency note because they still use GORM models, RBAC checks, and root helpers.
4. Root M79 payment prediction moved into `pkg/butler/prediction`; root files now provide compatibility aliases/wrappers.
5. `pkg/documents/pdf` now exposes the reusable PDF generator facade. Root `PDFGenerator` remains in place because contract rendering accesses package-private generator internals.
6. Deterministic and filesystem document classifier engines were copied into `pkg/documents/classifier`. AI/Wails endpoints remain rooted for permission and LLM access.
7. `pkg/documents/ocr` now defines `OCREngine` and `FitzEngine`; `ocr_service_simple.go` no longer imports `go-fitz` directly.
8. Excel costing and MSG email parser engines were extracted into `pkg/documents/excel` and `pkg/documents/email`. DB-backed import/save methods remain rooted.
9. `database.go` now has 5 remaining root structs, down from 33. Remaining structs are local sync/import/file-watch records without clean package twins yet.

## Validation

Every implementation ticket was gated with:

```powershell
$env:GOTMPDIR='D:\go-tmp'
$env:GOCACHE='D:\go-cache'
go build -tags='' ./...
go test ./... -count=1 -timeout 300s
```

Final observed gates passed cleanly after each committed ticket.

## Remaining Follow-Up

- Move the rest of Butler prompt/LLM helpers into `pkg/butler/chat` once action normalization and history-aware chat helpers are unified.
- Fully migrate invoice, offer, purchase-order, and contract PDF generation after root `PDFGenerator` internals are no longer accessed directly.
- Move PDF data extraction and annexure backfill after their OCR and GORM dependencies are expressed as ports.
- Create package twins for the remaining 5 root structs in `database.go` if sync/import/file-watch modules get dedicated packages.
