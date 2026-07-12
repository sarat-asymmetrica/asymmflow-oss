# Wave 8 Progress Audit

Date: 2026-05-06
Agent: Codex

## Summary

Wave 8 established real Butler package boundaries and started the Documents package boundary without touching frontend code or `pkg/finance/banking/`.

The work intentionally used small, validated extraction slices for the tangled Butler files. The largest orchestration paths still need a dedicated follow-up pass, but the package structure is now active instead of placeholder-only.

## Commits

- `266c5d8` `refactor(codex): define butler dependency ports`
- `214b1a3` `refactor(codex): extract intent router to butler package`
- `a3d85fb` `refactor(codex): extract grounded fastpath service`
- `5b75497` `refactor(codex): extract butler report helpers`
- `99effb5` `refactor(codex): extract butler persistence helpers`
- `094c570` `refactor(codex): extract butler chat core helpers`
- `f1b5e6c` `refactor(codex): extract payment intelligence engine`
- `cc5ba95` `refactor(codex): define document dependency ports`

## Validation

Each committed ticket was validated with:

- `go build -tags='' ./...`
- `go test ./... -count=1 -timeout 300s`

Final observed test gate for the last ticket passed across root, integration, API, data, engines, graph, OCR, infra events, and VQC packages.

## Metrics

- `butler_ai.go`: 6,986 LOC
- `pkg/butler/` total Go LOC: 1,603
- `pkg/documents/` total Go LOC: 101
- `database.go` remaining struct definitions: 33
- `database.go` type aliases: 57
- Root `func (a *App)` methods: 1,193
- go-fitz isolated behind documents OCR interface: No

## Completed Scope

### Butler

- Added Butler dependency ports in `pkg/butler/ports.go`.
- Added root Butler port adapters in `app_butler_ports.go`.
- Moved Butler response/action/intent DTOs into `pkg/butler`.
- Extracted pure intent routing into `pkg/butler/intent`.
- Added grounded fastpath service in `pkg/butler/fastpath` for capability and AR projection paths using ports.
- Extracted report prompt/data/rendering helpers into `pkg/butler/reports`.
- Extracted replay-safe persistence helpers into `pkg/butler/persistence`.
- Aliased `Conversation` and `ChatMessage` to `pkg/butler` models.
- Extracted prompt sanitization/context marshalling/action label helpers into `pkg/butler/chat`.
- Extracted DB-backed payment intelligence engine into `pkg/butler/prediction`.

### Documents

- Enriched `pkg/documents/ports.go` with `StoragePort`, `ConfigPort`, `FinanceDataPort`, `CompanyInfo`, and `BrandingConfig`.

## Deferred Items

- Full `butler_ai.go` orchestration extraction remains. The prompt-heavy and app-state-heavy functions need broader ports before moving safely.
- The M79 payment predictor in `predictor.go`, `batch.go`, and `customer.go` remains root-side. It has wide root test coverage and an older `pkg/engines` variant with a slightly different customer shape.
- Most of `butler_grounded_fastpath.go` remains root-side. Customer/supplier/task grounded paths still depend on App entity resolution and task workflow methods.
- PDF service extraction was not started in this wave. The existing PDF functions are long and should be moved as an isolated documents pass.
- Document classifier, OCR interface isolation, Excel/email/extraction service moves remain for the next wave.
- go-fitz is still imported outside `pkg/documents/ocr`; no OCR isolation was attempted in this wave.

## Recommended Wave 9

- Finish Butler first with a broader `ButlerAppContext`/workflow port for entity resolution, task creation, current employee context, and context building.
- Move the remaining grounded fastpaths in two groups: customer/supplier lookup, then task/offer-draft actions.
- Move the M79 predictor only after reconciling root `Customer` with `pkg/engines.Customer`.
- Then run a Documents wave: PDF services, classifier, OCR interface, Excel/email/extraction.
- Keep the same build/test/commit cadence.
