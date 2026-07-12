# Wave 10 Progress Audit

Date: 2026-05-06

## Scope Completed

Wave 10 built the additive adapter bridge from existing GORM/domain structs into the generated Cap'n Proto schema layer, then used the same bridge direction to pilot compact Butler prompt context and a dashboard V2 endpoint.

## Tickets

| Ticket | Status | Commit |
| --- | --- | --- |
| 1. Adapter package structure | Complete | `bb7af2d feat(codex): create adapter package structure for proto bridge` |
| 2. Finance adapter | Complete | `e686f7f feat(codex): create finance GORM-Proto adapter with roundtrip tests` |
| 3. CRM adapter | Complete | `2e970b2 feat(codex): create CRM GORM-Proto adapter with roundtrip tests` |
| 4. Butler adapter | Complete | `1064fe3 feat(codex): create Butler GORM-Proto adapter` |
| 5. Documents adapter | Complete | `96107e4 feat(codex): create Documents GORM-Proto adapter` |
| 6. Infra adapter | Complete | `56235d2 feat(codex): create Infra GORM-Proto adapter` |
| 7. Sync adapter | Complete | `ad9f5c7 feat(codex): create Sync GORM-Proto adapter` |
| 8. TOON at Butler boundary | Complete | `75f48ca feat(codex): encode Butler context with TOON` |
| 9. Dashboard V2 pilot | Complete | `d213a1d feat(codex): add dashboard stats proto pilot` |
| 10. Progress audit | Complete | this document |

## Adapter Coverage

- Shared helpers: time, deleted-at, integer, uint, and shared base conversions in `pkg/adapter`.
- Finance: invoices, invoice items, payments, bank accounts, purchase orders/items, bank statements/lines, credit notes/items, supplier invoices/items, supplier payments, chart accounts, journal entries/lines, VAT returns.
- CRM: customers, contacts, suppliers, supplier contacts, products, offers/items, opportunities, orders/items, delivery notes/items, serials, GRNs/items.
- Butler: all 13 generated Butler schema contracts, including response/action metadata, prediction records, conversations, and chat messages.
- Documents: company info, branding config, file watch events, bank statement files, classification results.
- Infra: settings, roles, users, devices, device users, sessions, alerts, audit logs, jobs, backup policies.
- Sync: file watch events, sync statuses, sync records, Tally invoice imports, Tally purchase imports.

## TOON Research Decision

Current TOON availability was checked before implementation:

- Official TOON repository: `https://github.com/toon-format/toon`
- Go package option: `https://pkg.go.dev/github.com/sstraus/toon_go/toon`
- Go package option: `https://pkg.go.dev/github.com/mateuszkardas/toon-go`

The official ecosystem is spec/TypeScript-led with community Go implementations. To avoid adding an unvetted serialization dependency into the offline-first Butler LLM boundary, Wave 10 implemented a small local encoder in `pkg/toon`. It handles the JSON-compatible context shape Butler already builds, including tabular encoding for uniform arrays.

Sample reduction from `pkg/toon` test:

```text
compact JSON=259 bytes (~65 tokens)
TOON=161 bytes (~41 tokens)
reduction=37.8%
```

The Butler context wrapper now emits `format: TOON` context through `pkg/butler/chat.MarshalContextForPromptCompact`, while preserving sanitization and max-length truncation.

## Dashboard V2 Pilot

`GetDashboardStatsV2()` is additive and leaves `GetDashboardStats()` unchanged. It calls the existing method, roundtrips the stats through generated `common.KeyValue_List`, and returns:

- the original `DashboardStats`
- `proto_schema: "common.KeyValue_List"`
- a frontend-safe key/value projection

Note: Wails JS/TS bindings were not regenerated in this wave. The Go endpoint exists and passes tests; frontend usage should run the normal Wails generation step in a later UI-facing pass.

## Validation

After each implementation ticket, the required gate was run:

```powershell
$env:GOTMPDIR='D:\go-tmp'
$env:GOCACHE='D:\go-cache'
New-Item -ItemType Directory -Force -Path $env:GOTMPDIR,$env:GOCACHE | Out-Null
go build -tags='' ./...
go test ./... -count=1 -timeout 300s
```

Final observed full-suite result after Ticket 9:

```text
go build -tags='' ./...: pass
go test ./... -count=1 -timeout 300s: pass
```

Targeted tests added:

- `pkg/adapter/finance`: invoice roundtrip
- `pkg/adapter/crm`: customer, order, offer roundtrips
- `pkg/adapter/butler`: chat message roundtrip
- `pkg/adapter/sync`: sync record and Tally invoice import roundtrips
- `pkg/toon`: tabular encoding reduction and token estimate
- `pkg/butler/chat`: TOON prompt context sanitization
- root package: dashboard stats V2 proto projection

## Guardrails Preserved

- No generated schema files were edited.
- No existing Wails method signatures were changed.
- No Svelte files were touched.
- No extracted `pkg/*/domain.go` or `pkg/*/service.go` files were edited.
- Adapter work is additive and remains outside existing production domain behavior.

## Follow-Up Candidates

- Regenerate Wails bindings for `GetDashboardStatsV2()` when the frontend wants to consume it.
- Add domain-specific dashboard schema in a future schema wave if `common.KeyValue_List` proves too generic.
- Expand full-fidelity roundtrip coverage beyond the highest-value finance/CRM/sync paths as the adapter layer becomes used by runtime flows.
- Consider swapping `pkg/toon` for an official or more mature Go TOON implementation later if the ecosystem stabilizes and dependency review is acceptable.
