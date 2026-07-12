# Business Memory Intake Status

Status: Durable contract ladder complete; source registry durability is runtime-visible and documented
Created: 2026-05-14
Scope: `pkg/documents/intake`, `internal/viewmodel/documents`, `InboxScreen.svelte`, Butler-safe context packs

## Product Loop

```text
unstructured message/document/folder
-> classified intake candidate
-> extracted fields and source evidence
-> review/correction state
-> suggested deterministic link/action
-> Butler-readable context pack
-> audit trail and next module queue
```

## Checkpoint A Contract

The canonical intake candidate lives in `pkg/documents/intake` as a pure Go package. It is a JSON-facing contract for reviewable business memory, not an authority service.

Required fields currently represented:

- `id`
- `source`
- `source_kind`
- `business_object_type`
- `classification`
- `extracted_fields`
- `suggested_links`
- `review_status`
- `audit_refs`
- `confidence`
- `warnings`

Source kinds:

- `message`
- `email`
- `pdf`
- `scan`
- `screenshot`
- `excel`
- `folder`
- `inbox_record`
- `other`

Review statuses:

- `new`
- `needs_review`
- `corrected`
- `linked`
- `rejected`
- `archived`

Extracted-field statuses:

- `extracted`
- `missing`
- `inferred`
- `needs_confirmation`
- `corrected`

## Normalization Sources

Checkpoint A normalizes these existing shapes without importing root `main` package runtime types:

- Runtime `InboxProcessResult` equivalent through `InboxProcessResultInput`.
- Stored `InboxDocument` equivalent through `InboxDocumentInput`.
- OCR/extraction map output through `OCRExtractionInput`.

The normalizers preserve source path/kind, document classification, confidence, field status, suggested deterministic service target, warnings, and audit references.

## Authority Boundary

Business Memory Intake candidates are review proposals. They may be inspected, corrected, linked by deterministic services, and packaged for Butler context. The pure kernel does not approve, post, delete, create authoritative records, or mutate inbox/accounting/CRM state.

## Data Contract Boundary

Current durable target:

- Cap'n Proto now owns the Business Memory durable/cross-module shape in `schemas/documents.capnp`.
- Go structs in `pkg/documents/intake` remain the pure kernel and deterministic service input/output authority.
- JSON remains appropriate for module manifests, Wails/browser payloads, fixtures, operator/support exports, and third-party interop.
- TOON remains appropriate for Butler/Codex/agent context packs where compact source-cited context is more valuable than binary durability.

Durable schema additions:

- `BusinessMemorySourceKind`
- `BusinessMemoryReviewStatus`
- `BusinessMemoryFieldStatus`
- `BusinessMemoryReviewDecision`
- `BusinessMemorySourceRef`
- `BusinessMemoryClassification`
- `BusinessMemoryExtractedField`
- `BusinessMemorySuggestedLink`
- `BusinessMemoryAuditRef`
- `BusinessMemoryCandidate`
- `BusinessMemoryContextPack`
- `BusinessMemoryReviewRecord`
- batch wrappers for candidates, context packs, and review records

The schema remains additive to the existing document/OCR/classifier contracts. Review decisions still require deterministic service/storage wiring before authoritative mutations are exposed through Wails or UI commands.

## Durable Adapter Bridge

Current generated surfaces:

- Go generated schemas: `schemas/go/documents/documents.capnp.go`.
- TypeScript generated schemas: `frontend/src/lib/types/schemas/index.ts`.
- Adapter bridge: `pkg/adapter/documents/business_memory.go`.
- Adapter tests: `pkg/adapter/documents/business_memory_test.go`.

Round-trip coverage now exists for:

- `intake.Candidate` <-> `BusinessMemoryCandidate`.
- `intake.ContextPack` <-> `BusinessMemoryContextPack`.
- `intake.ReviewRecord` <-> `BusinessMemoryReviewRecord`.

This makes the Cap'n Proto contract current for durable/cross-module readiness while preserving the pure Go kernel as the deterministic authority layer.

## Durable Review Storage

Current storage boundary:

- Repository contract: `pkg/documents/intake/review_repository.go`.
- In-memory deterministic adapter: `MemoryReviewRecordRepository`.
- GORM durable adapter: `pkg/adapter/documents/business_memory_storage.go`.
- Durable table model: `business_memory_review_records`.
- Idempotency key: candidate ID + decision + proposed deterministic service + correlation ID.
- Explicit migration method: `GORMBusinessMemoryReviewRepository.Migrate(ctx)`.

Migration note:

The main app's startup path intentionally skips broad `AutoMigrate` once an established SQLite database has more than 50 tables. Therefore the durable adapter exposes an explicit migration method and is covered with focused GORM tests rather than assuming mature client databases will pick up the new table through startup automigration.

Storage tests cover save, list, get, idempotent duplicate writes, and invalid decision rejection.

## Deterministic Review Service

Current service boundary:

- Service: `pkg/documents/intake/review_service.go`.
- Tests: `pkg/documents/intake/review_service_test.go`.
- Authority: deterministic service records review decisions through a `ReviewRecordRepository`.
- Required mutation inputs: candidate ID, source ID, supported decision, operator actor, and correlation ID.
- Idempotency: delegated to the repository using the review record idempotency key.
- Read/query support: get review record, list records by candidate, and build queue state with last review plus context pack.
- Agent boundary: actor type `agent` is rejected from `RecordDecision`; agents remain limited to inspect/explain/draft/recommend/context assembly paths.

The service may assemble context packs and queue state, but it does not create accounting, CRM, inventory, procurement, or other authoritative business records.

## Runtime, Wails, And UI Wiring

Current Wails/runtime boundary:

- App methods: `GetBusinessMemoryReviewQueue`, `RecordBusinessMemoryReviewDecision`, and `GenerateBusinessMemoryContextPack`.
- Service wrappers: `DocumentsService` exposes the same methods for the domain-scoped frontend surface.
- Permissions: queue/context pack reads use `documents:view`; operator review decisions use `documents:classify` so existing seeded roles retain access.
- Persistence: the runtime service constructs the GORM review repository, runs the explicit repository migration, and records review decisions idempotently by candidate, decision, deterministic service target, and correlation ID.
- ViewModel: `BuildIntakeReviewVMFromQueueStates` surfaces persisted last review state, actor/reason, decision status, command availability, and deterministic service target.
- UI: the Inbox Business Memory panel loads persisted queue state, records operator review choices through Wails, and displays the last saved review instead of relying only on transient proposal labels.
- Bindings: `wails generate module` refreshed `frontend/wailsjs` for the new methods and payloads.
- Source registry: runtime review/export paths construct the durable source asset repository, run its explicit migration, upsert inbox-derived source assets, and expose compact source provenance through the selected candidate ViewModel and Inbox Business Memory panel.

## Export And Replay

Current evidence bundle boundary:

- Pure bundle: `pkg/documents/intake.ReviewExportBundle`.
- JSON export: `ExportReviewBundleJSON` for operator/support bundles and browser interop.
- JSON replay: `ReplayReviewBundleJSON` validates the schema version, normalizes the candidate, rebuilds the context pack, and sorts review records.
- TOON export: `ExportReviewBundleTOON` for Butler/Codex/agent context, including allowed/forbidden agent-action boundaries and persisted review records.
- Wails export: `ExportBusinessMemoryReviewBundle` returns JSON and TOON strings plus the typed bundle for the selected inbox candidate.
- Source registry provenance: `ReviewExportBundle.SourceAssets` and `ReviewQueueState.SourceAssets` can carry registry-derived source metadata alongside candidate-local source refs.
- Cap'n Proto evidence: Business Memory candidate/context/review record adapter tests remain the durable contract readiness check.

## Source Asset Registry

Promoted continuation status:

- Pure registry: `pkg/documents/intake/source_registry.go`.
- Repository boundary: `pkg/documents/intake/source_repository.go`.
- Durable adapter: `pkg/adapter/documents/business_memory_source_storage.go`.
- Source identity: stable IDs use source kind plus content hash when available, then path, then label.
- Tracked fields: source ID, kind, path, label, hash, import batch, privacy class, processing status, candidate IDs, audit refs, first seen time, and last seen time.
- Duplicate handling: `SourceAssetRegistry.Upsert` detects duplicate stable IDs and merges candidate IDs, audit refs, import batch, status, and seen-time range.
- In-memory repository: `MemorySourceAssetRepository` provides idempotent upsert, get, list, and list-by-candidate semantics for deterministic tests and future durable adapter parity.
- GORM repository: `GORMBusinessMemorySourceAssetRepository` exposes explicit `Migrate(ctx)` and stores candidate IDs / audit refs as JSON text without assuming startup automigration on mature client databases.
- Runtime surfacing: `GetBusinessMemoryReviewQueue`, `RecordBusinessMemoryReviewDecision`, and `ExportBusinessMemoryReviewBundle` now preserve source asset provenance through queue state, exports, Wails bindings, and the Inbox review surface.
- Tests cover hash-backed stable IDs, duplicate merge behavior, candidate/audit merges, seen-time range preservation, default privacy/status/source-kind inference, repository filters, durable SQLite persistence, and invalid input rejection.

## PH Source-Track Acceptance Constraints

The `ph_holdings` source-track lessons are carried as acceptance constraints, not as a blind code backport:

- Revision provenance: commercial documents need durable revision identity before operator-visible truth claims.
- Allocation-aware evidence: bank/cashflow matching must support customer invoices, supplier invoices, expenses, partial allocations, and mixed matches.
- Approval queues: destructive or authoritative actions need explicit operator approval states.
- Sync/conflict policy: durable source and review records need module-aware sync envelopes before cross-device/client merge promises.
- UI/backend action inventories: launch readiness requires visible action ownership, command IDs, and backend support status.

## Agent-Safe Contract

Allowed agent/Butler actions:

- inspect
- explain
- draft
- recommend
- assemble context

Forbidden agent/Butler actions:

- approve
- link
- post
- delete
- create authoritative business records

## Checkpoint Ledger

### Checkpoint A

- Status: implemented.
- ROI proof: reduces retyping and evidence reconstruction by turning inbox/OCR outputs into one reviewable candidate shape.
- Workflow closure: produces source-backed candidates with field status, warnings, suggested deterministic service targets, and audit refs.
- Engine leverage: reuses existing Runtime inbox, stored inbox, OCR/extraction maps, and classifier confidence instead of adding a disconnected parser.
- Operator trust: source provenance, confidence, field-level status, warnings, review status, and audit refs are visible in the contract.
- Verification:
  - `Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null` passed.
  - `go test ./pkg/documents/... -count=1` passed.
  - `git diff --check` passed with LF/CRLF normalization warnings only.

### Checkpoint B

- Status: implemented.
- Target: ViewModel and bounded Inbox review surface.
- Verification:
  - `go test ./internal/viewmodel/documents -count=1` passed.
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
  - `npm.cmd --prefix frontend run build` passed with baseline Svelte warnings.
  - `git diff --check` passed with LF/CRLF normalization warnings only.

### Checkpoint C

- Status: implemented.
- Target: Butler-readable context pack that remains inspect/explain/draft/recommend only.
- Verification:
  - `go test ./pkg/documents/... -count=1` passed.
  - `Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null` passed.
  - `git diff --check` passed with LF/CRLF normalization warnings only.
  - `go build ./...` passed.
  - `git diff --check` passed.

### Checkpoint D

- Status: implemented as durable-review preparation.
- Target: durable review queue preparation, or inventory asset ledger preflight if persistence is unsuitable.
- Current durability: `pkg/documents/intake` now defines the review record shape, decision-to-status mapping, idempotency key behavior, and an in-memory review queue adapter with tests.
- Persistence boundary: no database migration was added in this checkpoint. The durable storage migration should create an intake review table keyed by candidate id, decision, deterministic service target, and correlation id, then expose it through a deterministic document review service before Wails/UI writes are enabled.
- Verification:
  - `go test ./pkg/documents/... -count=1` passed.
  - `go build ./...` passed.
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
  - `git diff --check` passed.

### Checkpoint E

- Status: implemented.
- Target: durable review service exposed to Wails/ViewModel/UI.
- Verification:
  - `go test ./internal/viewmodel/documents -count=1` passed.
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.
  - `go test . -run "TestParseBusinessMemoryReviewDecision|TestInboxDocumentToBusinessMemoryCandidate" -count=1` passed.
  - `go build ./...` passed.
  - `wails generate module` passed; generator emitted the existing `Not found: struct { R1 float64 ... }` warning.
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
  - `git diff --check` passed with LF/CRLF normalization warnings only.

### Checkpoint F

- Status: implemented.
- Target: Inbox review commands backed by durable operator decisions.
- Command coverage: accept proposal, needs input, correct field, reject candidate, and archive review are visible as operator actions without changing the deterministic service authority boundary.
- Product component preservation: the review surface still uses `KpiStatusStrip`, `EvidenceSourceList`, and `ActionProposalCard`; this checkpoint only extends commands and labels.
- Verification:
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
  - `npm.cmd --prefix frontend run build` passed with the same baseline Svelte warnings.
  - `go test ./internal/viewmodel/documents -count=1` passed.

### Checkpoint G

- Status: implemented.
- Target: final audit, export, replay, and durable-contract ledger.
- Export/replay coverage: JSON bundle export/replay, TOON evidence export, Wails document-service export method, Cap'n Proto adapter test evidence.
- Verification:
  - `Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null` passed.
  - `powershell -ExecutionPolicy Bypass -NoProfile -File schemas\generate.ps1 -CheckOnly` passed.
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.
  - `go test ./internal/viewmodel/documents -count=1` passed.
  - `go build ./...` passed.
  - `wails generate module` passed with the existing `Not found: struct { R1 float64 ... }` warning.
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
  - `npm.cmd --prefix frontend run build` passed with baseline Svelte warnings.
  - `git diff --check` passed with LF/CRLF normalization warnings only.
  - Optional broad gate `go test ./... -count=1 -timeout 300s` passed.

### Promotion A.1

- Status: implemented.
- Target: Business Memory Source Asset Registry first tested checkpoint.
- Verification:
  - `go test ./pkg/documents/... -count=1` passed.

### Source Registry Durability - Checkpoint 0

- Status: recorded.
- Start commit: `00ec6cf`.
- Starting status: clean `master`.
- `ph_holdings` source-track evidence: used only through `docs/CODEX_PH_HOLDINGS_SOURCE_TRACK_RECON_2026_05_15.md`; no source-track repo writes or blind backporting.
- Active checkpoint: Checkpoint 1, Source Registry Repository Boundary.
- GitNexus status: informational; index reported stale at `6d8a69d` while current commit was `00ec6cf`, caused by the docs-only audit commit after source-registry code landed.

### Source Registry Durability - Checkpoint 1

- Status: implemented.
- Target: source registry repository boundary.
- Completed changes:
  - Added `SourceAssetRepository` and `SourceAssetListFilter`.
  - Added `MemorySourceAssetRepository` with idempotent upsert keyed by stable source ID, get, list, and list-by-candidate methods.
  - Preserved duplicate merge behavior from `SourceAssetRegistry.Upsert`.
  - Added validation for required source ID, kind, label, privacy class, processing status, and seen-time range.
  - Added focused repository tests for stable ID persistence, duplicate evidence merge, candidate/audit reference merge, first/last seen preservation, filters, and invalid input rejection.
- Verification:
  - `go test ./pkg/documents/... -count=1` passed.

### Source Registry Durability - Checkpoint 2

- Status: implemented.
- Target: durable GORM adapter for source assets.
- Completed changes:
  - Added `BusinessMemorySourceAssetModel` and `GORMBusinessMemorySourceAssetRepository`.
  - Added explicit `Migrate(ctx)` for `business_memory_source_assets`.
  - Stored source ID, kind, path, label, hash, import batch, privacy class, processing status, candidate IDs, audit refs, first seen, and last seen.
  - Serialized candidate IDs and audit refs as JSON text for predictable local storage.
  - Preserved duplicate upsert semantics by reusing the intake merge helper.
  - Added SQLite-backed adapter tests for persistence, duplicate merge, candidate listing, filters, and invalid input rejection.
- Migration note: this checkpoint does not add startup automigration; the adapter follows the Business Memory review storage pattern with explicit migration ownership.
- Verification:
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.
  - `go build ./...` passed.

### Source Registry Durability - Checkpoint 3

- Status: implemented.
- Target: review queue and export integration for source registry provenance.
- Completed changes:
  - Extended `ReviewQueueState` with optional `SourceAssets`.
  - Added `ReviewService.BuildQueueStateWithSources` to enrich candidate queue state from a `SourceAssetRepository`.
  - Extended `ReviewExportBundle` with optional `SourceAssets`.
  - Added `NewReviewExportBundleWithSources` for JSON/TOON bundles that include source registry provenance.
  - Updated TOON export to cite source asset ID, kind, label/path/hash, privacy class, processing status, candidate IDs, and audit reference count.
  - Added tests proving JSON replay preserves source assets and TOON export cites registry provenance.
- Verification:
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.

### Source Registry Durability - Checkpoint 4

- Status: implemented.
- Target: runtime, ViewModel, Wails binding, and Inbox surface provenance.
- Commit: `eca8771`.
- Completed changes:
  - Wired the durable source asset repository through Business Memory queue, decision, and export runtime paths.
  - Added inbox-derived source asset upserts before queue state and export bundle construction.
  - Exposed `sourceRegistry` summaries through `internal/viewmodel/documents`.
  - Added a compact source registry provenance block to `BusinessMemoryReviewPanel.svelte`.
  - Refreshed Wails models for `documents.SourceRegistryItemVM`, `intake.SourceAsset`, and `ReviewExportBundle.source_assets`.
- Verification:
  - `go test ./internal/viewmodel/documents -count=1` passed.
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.
  - `go test . -run "TestParseBusinessMemoryReviewDecision|TestInboxDocumentToBusinessMemoryCandidate|TestCandidateToBusinessMemorySourceAsset" -count=1` passed.
  - `go build ./...` passed.
  - `wails generate module` passed with the known generated-struct warning for anonymous `r1/r2/r3` prediction fields.
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.

### Source Registry Durability - Checkpoint 5

- Status: implemented.
- Target: manifest, status, and action inventory truth after source registry durability/runtime surfacing.
- Commit: `docs(asymmflow): record business memory source registry durability`.
- Completed changes:
  - Updated the Business Memory manifest `status`, `next_goal.primary_gap`, source registry state, runtime/viewmodel/UI state, tests, and launch readiness entries.
  - Added PH source-track acceptance constraints for revision provenance, allocation-aware evidence, approval queues, sync/conflict policy, and UI/backend action inventories.
  - Recorded the current remaining gap as the Cashflow Evidence allocation-aware preflight promotion, not more Business Memory source registry foundation.
- Verification:
  - `Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null` passed.
  - `git diff --check` passed with LF/CRLF normalization warnings only.

## Remaining Gaps

- Cashflow Evidence operator loop: allocation-aware read-model preflight is now available; future work should wire real snapshot sources, operator review state, and UI drill-downs.
- Business Memory sync/conflict policy: future durable source/review records need module-aware sync envelopes before distributed client merge claims.
- Operator action inventory: future surfaces should continue to distinguish inspect/draft/recommend actions from approval/link/post/delete authority.

## Next Goal

Next handoff:

```text
docs/CODEX_GOAL_BUSINESS_MEMORY_SOURCE_REGISTRY_DURABILITY_HANDOFF.md
```

Current completion state:

- The original intake handoff checkpoints A through D are implemented and committed.
- The durable contract handoff's 21-checkpoint ladder is implemented and committed through `9baf305`.
- Promoted continuation A.1, the pure Business Memory Source Asset Registry checkpoint, is implemented and committed in `6d8a69d`.
- The source registry durability chain is implemented through repository boundary, durable adapter, review/export provenance, runtime/ViewModel/UI surfacing, and this manifest/status checkpoint.

Next architecture target:

- Promote the Cashflow Evidence allocation-aware preflight so bank matching can model partial and mixed allocations before the operator loop expands.
- Preserve Cap'n Proto as the durable/cross-module contract, Go structs as pure/service authority, JSON for manifests/Wails/fixtures/operator exports, and TOON for Butler/Codex/agent context packs.
