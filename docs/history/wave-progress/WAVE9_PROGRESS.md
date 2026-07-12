# Wave 9 Progress - Cap'n Proto Schema Construction

Date: 2026-05-06
Branch: master

## Scope Completed

Wave 9 established the Cap'n Proto contract layer without replacing the existing GORM domain structs. The generated Go code lives under `schemas/go/`, and frontend contract interfaces live under `frontend/src/lib/types/schemas/`.

## Tickets

| Ticket | Status | Commit | Notes |
| --- | --- | --- | --- |
| 1. Verify toolchain | Complete | `7ee2df9` | Verified `capnp` and `capnpc-go`; created `schemas/`. The external HDD is used for `D:\go-tmp` and `D:\go-cache`. |
| 2. `common.capnp` | Complete | `987f20c` | Added shared `Base`, money/address/contact helpers, paging, validation issues, and 8 enums. |
| 3. `finance.capnp` | Complete | `7a3a74a` | Added 45 finance/banking data contracts, including invoices, payments, POs, banking, FX, reconciliation, and Tally-facing summaries. |
| 4. `crm.capnp` | Complete | `b6b0699` | Added 35 CRM/pipeline/fulfillment/inventory contracts. PurchaseOrder was intentionally not duplicated because finance owns it. |
| 5. `butler.capnp` | Complete | `997dd81` | Added the requested 13 Butler data contracts for responses, actions, intent, entities, predictions, conversations, and messages. |
| 6. `documents.capnp` | Complete | `cc82d62` | Added company/branding, bank statement file, OCR/classification, Butler OCR insight, and OCR pipeline request/result contracts. |
| 7. `infra.capnp` | Complete | `65001dd` | Added User, Role, Device, DeviceUser, UserSession, Setting, AuditLog, Job, Alert, and BackupPolicy. |
| 8. `sync.capnp` | Complete | `d3c20a4` | Added FileWatchEvent, file sync state, SyncStatus, SyncRecord, TallyInvoiceImport, and TallyPurchaseImport. Go package is `syncschema` to avoid stdlib `sync` confusion. |
| 9. Generator script | Complete | `8d60768` | Added `schemas/generate.ps1` with Go generation, include-path discovery, safe output placement, and fallback TypeScript generation. |
| 10. Go generation | Complete | `9577a80` | Generated Go packages under `schemas/go/<package>/`. Added `capnproto.org/go/capnp/v3 v3.1.0-alpha.2` and `github.com/colega/zeropool`. |
| 11. TypeScript generation | Complete | `24fba65` | Generated namespace-based TypeScript interfaces in `frontend/src/lib/types/schemas/index.ts`. |
| 12. Progress report | Complete | pending commit | This document. |

## Generated Outputs

- Go:
  - `schemas/go/common/common.capnp.go`
  - `schemas/go/finance/finance.capnp.go`
  - `schemas/go/crm/crm.capnp.go`
  - `schemas/go/butler/butler.capnp.go`
  - `schemas/go/documents/documents.capnp.go`
  - `schemas/go/infra/infra.capnp.go`
  - `schemas/go/syncschema/sync.capnp.go`
- TypeScript:
  - `frontend/src/lib/types/schemas/index.ts`

## Validation

Cap'n Proto compilation:

```powershell
powershell -ExecutionPolicy Bypass -File schemas/generate.ps1 -CheckOnly
```

Go validation after every ticket:

```powershell
$env:GOTMPDIR='D:\go-tmp'
$env:GOCACHE='D:\go-cache'
New-Item -ItemType Directory -Force -Path $env:GOTMPDIR,$env:GOCACHE | Out-Null
go build -tags='' ./...
go test ./... -count=1 -timeout 300s
```

Frontend validation for Ticket 11:

```powershell
cd frontend
npm run build
```

Final observed status:

- `capnp compile`: pass for all schemas
- `go build -tags='' ./...`: pass
- `go test ./... -count=1 -timeout 300s`: pass, including generated schema packages
- `frontend/npm run build`: pass

## Notes

- The Go plugin writes generated files into a flat temp folder, so `schemas/generate.ps1` parses the generated package line and moves each file into `schemas/go/<package>/` to match the `$Go.import(...)` annotations.
- TypeScript generation uses the fallback parser because no `capnpc-ts`, `capnpc-typescript`, or `capnpc-js` plugin was available on PATH.
- Existing domain files, `database.go`, and package ports were not modified for the schema tickets. Generated schemas coexist with the current GORM model layer.
