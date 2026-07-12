# CODEX_GOAL_MODULE_CONTRACT_HANDOFF

Status: Draft v0.1
Created: 2026-05-14
Scope: `the AsymmFlow repository`

## Goal Command

Complete the AsymmFlow Module Contract Foundation goal without stopping until the repo has a concrete module contract foundation, mapped against existing AsymmFlow domains, with verification and documentation updated.

## Current Context

Required context was read in order:

1. `C:\Projects\ASYMMETRICA_ECOSYSTEM_LOG.md`
2. `C:\Projects\AGENTIC_SWARM_PROTOCOL.md`
3. `C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md`
4. `docs/CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md`
5. Repo status plus `docs/V0_1_RELEASE_ROADMAP_2026_05_08.md` and `docs/WAVE17_PROGRESS.md`

The current target architecture is:

```text
Core Kernel -> Domain Module -> ViewModel -> UI Component -> Agent Surface
```

Older roadmaps are evidence, not canon. The 2026-05-14 master roadmap is the current planning spine.

## Dirty Worktree Boundary

Pre-edit `git status --short --branch` showed existing unrelated dirty work:

```text
## master
 M AGENTS.md
 M CLAUDE.md
?? .codex/
?? CODEX_WAVE*_HANDOFF.md
?? docs/ARCHAEOLOGICAL_REPORT.md
?? docs/AUTONOMOUS_SPRINT_METHODOLOGY.md
?? docs/CME_SCORING_GATES.md
?? docs/CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md
?? docs/COLLABORATION_PROTOCOL.md
?? docs/GENERATIVE_REFACTOR_PLAN.md
?? docs/OPERATING_PRINCIPLES.md
?? docs/TARGET_ARCHITECTURE.md
?? docs/WAVE0_AUDIT.md
?? docs/audit_results/
?? goal.md
```

This goal only owns:

- `docs/MODULE_CONTRACT_FOUNDATION.md`
- `docs/templates/module_manifest.example.json`
- `docs/CODEX_GOAL_MODULE_CONTRACT_HANDOFF.md`
- the Goal 1 note in `docs/CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md`

Do not revert or stage unrelated files.

## Product Bar

The module contract applies the "$1000/mo justification" filters to every future module:

- ROI proof: each module must name the labor, leakage, subscription cost, compliance risk, or uncertainty it removes.
- Workflow closure: each module must end with an action, artifact, posting, export, approval, claim pack, or decision.
- Engine leverage: each module must name the deterministic engine, invariant, memory/OCR/classification flow, optimization routine, or local-first runtime advantage.
- Operator trust: each module must expose inspectability, correction, approval, export, audit, and repeatability.

## Architecture Constraints

- Go backend/domain services remain authoritative for approvals, persistence, posting, filing, deletion, and invariant enforcement.
- Agents inspect, explain, draft, recommend, and assemble evidence only.
- MVVM is the default frontend architecture.
- Cap'n Proto is preferred for durable schemas, engine contracts, generated bindings, sync envelopes, and local-first records.
- TOON is preferred for agent/context transfer, compact source summaries, and evidence handoffs.
- JSON is preferred for manifests, browser payloads, simple configs, third-party APIs, and interop.
- UI-facing modules must follow `C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md`.

## Orchestration

Read-only subagents were dispatched for bounded inspection:

- Domain seam explorer: map Finance, Documents, Butler, Compliance, Inventory, and Cashflow Evidence against current code/docs.
- Documentation/verification explorer: identify doc/template locations and verification commands.

The orchestrator retained ownership of final docs, integration, verification, and commit boundary.

## Artifacts

### Module Contract

Primary contract doc:

```text
docs/MODULE_CONTRACT_FOUNDATION.md
```

It defines required module shape across:

- schemas and data contracts
- pure kernels
- domain services
- storage adapters
- ViewModels
- UI surfaces
- events
- permissions
- audit trails
- tests
- agent-safe APIs
- launch readiness checklist

### Manifest Template

Template path:

```text
docs/templates/module_manifest.example.json
```

JSON is used because this is a human-editable planning/config artifact. Durable runtime module records should later move to Cap'n Proto if generated/runtime consumption becomes necessary. Agent briefings and evidence packs should use TOON.

### Roadmap Update

`docs/CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md` is updated only to sharpen Goal 1 from "candidate spec" to concrete artifacts.

## Domain Mapping Summary

| Domain | Current Fit | Key Seams | Next Refactor |
| --- | --- | --- | --- |
| Finance | Strong | `pkg/finance`, `pkg/finance/posting`, `schemas/finance.capnp`, `accounting_posting_service.go`, `internal/viewmodel/finance`, `AccountingScreen.svelte` | Move root services behind adapters; finish posting action, period-lock checks, support-bundle export. |
| Documents | Medium | `pkg/documents`, `schemas/documents.capnp`, `document_classifier.go`, `ocr_service_simple.go`, `InboxScreen.svelte` | Add canonical evidence record, review queue VM, source-link audit, TOON evidence pack. |
| Butler | Medium-strong | `pkg/butler`, `schemas/butler.capnp`, `butler_grounded_fastpath.go`, `butler_intent_router.go`, `internal/viewmodel/butler` | Formalize action tiers; keep agent writes draft-only; emit cited context packs. |
| Compliance | Strong kernel fit | `pkg/compliance`, Bahrain/India engines, `pkg/compliance/hooks.go`, `internal/viewmodel/compliance_vm.go` | Add ports/manifest, filing/export surfaces, typed event payloads, jurisdiction-pack readiness. |
| Inventory | Medium | `pkg/crm` inventory/serial structs, `pkg/crm/ports.go`, `app_accounting_inventory.go`, `inventory_service.go`, `serial_number_service.go`, `schemas/crm.capnp` | Extract stock ledger kernel; split inventory module from CRM; add valuation/reservation evidence. |
| Cashflow Evidence | Emerging composition | Finance posting coverage, trial-balance gate, AR/cashflow reports, bank reconciliation, invoice traceability, document/OCR, Butler context | Build first-class read model, event subscriptions, evidence schema, command center VM, export. |

## Verification Gates

Because this goal is documentation and template-only, no Go, schema, or frontend source files were intentionally changed. Required verification is therefore:

- read-only/status checks before editing
- doc/template existence checks
- JSON validity for the manifest
- markdown/content search checks for mandatory sections
- `git diff --check` for whitespace/conflict markers
- final git status boundary review

If code or schema files are touched in a later goal, use:

- `go build ./...`
- `go test ./... -count=1 -timeout 300s`
- `cd frontend && npm run build`
- `cd frontend && npm run check`
- `powershell -NoProfile -File schemas/generate.ps1 -CheckOnly` for Cap'n Proto schema edits
- `.\scripts\verify_release.ps1 -SkipWailsBuild` for the full no-Wails sanity gate

## Command Log

Commands actually run before/during this goal:

| Command | Result |
| --- | --- |
| `rg -n "AsymmFlow|module contract|Module Contract|WAVE17|V0_1_RELEASE|Core Kernel|Domain Module|ViewModel|Engine Generalization" C:\Users\YourName\.codex\memories\MEMORY.md` | Passed; found prior AsymmFlow roadmap and deterministic nucleus notes. |
| `Get-Content -Raw C:\Projects\ASYMMETRICA_ECOSYSTEM_LOG.md` | Passed. |
| `Get-Content -Raw C:\Projects\AGENTIC_SWARM_PROTOCOL.md` | Passed. |
| `Get-Content -Raw C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md` | Passed. |
| `Get-Content -Raw docs\CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md` | Passed. |
| `git status --short --branch` | Passed; dirty boundary recorded above. |
| `Test-Path docs\V0_1_RELEASE_ROADMAP_2026_05_08.md` | Passed; returned `True`. |
| `Test-Path docs\WAVE17_PROGRESS.md` | Passed; returned `True`. |
| `rg --files docs` | Passed. |
| `Get-Content -Raw docs\V0_1_RELEASE_ROADMAP_2026_05_08.md` | Passed. |
| `Get-Content -Raw docs\WAVE17_PROGRESS.md` | Passed. |
| `Get-Content -Raw AGENTS.md` | Passed; file contains older encoded text but GitNexus guidance and repo facts were readable. |
| `Get-ChildItem -Name` | Passed. |
| `Get-ChildItem -Name pkg` | Passed. |
| `Get-ChildItem -Name frontend\src\lib\screens` | Passed. |
| `rg -n "PreviewCustomerInvoicePosting|CreateDraftJournalFromPosting|CoverageReport|TrialBalanceGate|ProcessInboxDocument|DocumentClassifier|Butler|VAT|GST|Inventory|Serial|Stock|CashFlow|requirePermission|Audit" -S .` | Passed; produced very large output, useful seams sampled. |
| `rg --files schemas` | Passed. |
| `rg --files internal\viewmodel` | Passed. |
| `rg --files docs | rg "template|manifest|contract|module|architecture|progress|roadmap|handoff"` | Passed; no existing docs/templates manifest location found. |
| `Get-ChildItem -Recurse -Directory docs | Select-Object -ExpandProperty FullName` | Passed; confirmed no existing `docs/templates` directory. |
| `Get-Content -Raw pkg\finance\domain.go` | Passed. |
| `Get-Content -Raw pkg\finance\ports.go` | Passed. |
| `Get-Content -Raw pkg\finance\posting\posting.go` | Passed. |
| `Get-Content -Raw internal\viewmodel\finance\finance_vm.go` | Passed. |
| `Get-Content -Raw pkg\documents\domain.go` | Passed. |
| `Get-Content -Raw pkg\documents\ports.go` | Passed. |
| `Get-Content -Raw pkg\butler\domain.go` | Passed. |
| `Get-Content -Raw pkg\butler\ports.go` | Passed. |
| `Get-ChildItem -Recurse -File pkg\compliance | Select-Object -ExpandProperty FullName` | Passed. |
| `Get-Content -Raw pkg\compliance\ports.go` | Failed; file does not exist. This is recorded as a refactor gap. |
| `Get-Content -Raw internal\viewmodel\compliance_vm.go` | Passed. |
| `Get-Content -Raw pkg\crm\ports.go` | Passed. |
| `Get-Content -Raw pkg\compliance\compliance.go` | Passed. |
| `Get-Content -Raw pkg\compliance\hooks.go` | Passed. |
| `Get-Content -Raw pkg\compliance\bahrain\vat.go` | Passed. |
| `Get-Content -Raw pkg\compliance\india\gst.go` | Passed. |
| `Get-Content -Raw pkg\infra\events\events.go` | Passed. |
| `Get-Content -Raw pkg\infra\events\bus.go` | Passed. |
| `Get-Content -Raw schemas\finance.capnp` | Passed. |
| `Get-Content -Raw schemas\butler.capnp` | Passed. |
| `Get-Content -Raw Taskfile.yml` | Passed. |
| `Get-Content -Raw frontend\package.json` | Passed. |
| `Get-Content -Raw scripts\verify_release.ps1` | Passed. |
| `Get-Content -Raw schemas\generate.ps1` | Passed. |
| `Get-Content -Raw docs\MODULE_CONTRACT_FOUNDATION.md` | Passed; reviewed generated contract doc. |
| `Get-Content -Raw docs\CODEX_GOAL_MODULE_CONTRACT_HANDOFF.md` | Passed; reviewed generated handoff. |
| `Get-Content -Raw docs\templates\module_manifest.example.json \| ConvertFrom-Json \| Out-Null; Write-Output "JSON OK"` | Passed; manifest is valid JSON. |
| `rg -n "Finance|Documents|Butler|Compliance|Inventory|Cashflow Evidence|schemas and data contracts|pure kernels|domain services|storage adapters|ViewModels|UI surfaces|events|permissions|audit trails|tests|agent-safe APIs|launch readiness" docs\MODULE_CONTRACT_FOUNDATION.md docs\CODEX_GOAL_MODULE_CONTRACT_HANDOFF.md` | Passed; mandatory sections and domain mappings found. |
| conflict-marker search across the module contract docs and roadmap | Passed with no matches after removing this command from the log to avoid self-matching. |
| `git diff --check -- docs\CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md` | Passed. |
| `git diff -- docs\CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md` | No tracked diff shown because the roadmap file is currently untracked in this worktree. |
| `git status --short --branch` | Passed; showed expected new module-contract docs plus pre-existing unrelated dirty work. |
| `rg --files pkg\documents pkg\butler pkg\finance\banking pkg\crm\procurement pkg\crm\fulfillment` | Passed; confirmed package seams reported by read-only explorer. |
| `Get-Content -Raw docs\templates\module_manifest.example.json` | Passed; reviewed manifest content. |
| `Get-Content -Raw docs\templates\module_manifest.example.json \| ConvertFrom-Json \| Out-Null; Write-Output "JSON OK"` | Passed again after updates; manifest remains valid JSON. |
| mandatory-section/domain `rg` check across module contract and handoff | Passed after updates; required sections and domain mappings present. |
| conflict-marker search across the module contract docs and roadmap | Passed after updates with no matches; `rg` exit code 1 means no matches. |
| `git add docs\MODULE_CONTRACT_FOUNDATION.md docs\templates\module_manifest.example.json docs\CODEX_GOAL_MODULE_CONTRACT_HANDOFF.md docs\CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md` | First attempt failed with `.git/index.lock` permission denied under sandbox; escalated retry passed. Git warned LF will be replaced by CRLF on next touch. |
| `git diff --cached --name-status` | Passed; staged only the four goal-owned docs. |
| `git diff --cached --check` | Passed; no whitespace errors. |
| `git diff --cached --stat` | Passed; 4 files, 1295 insertions. |
| `git status --short --branch` | Passed; staged set is goal-owned docs only, unrelated dirty work remains unstaged. |
| `npx.cmd gitnexus status` | Passed; reported GitNexus index stale, indexed commit `ac97535`, current commit `b0e4a70`. |
| `npx.cmd gitnexus --help` | Passed; no `detect_changes` command exists in the local CLI help, so staged Git diff checks are the available pre-commit scope check. |

Final verification commands should be appended after they run.

## Residual Risks

- This goal defines the contract and maps current seams; it does not refactor modules into that shape yet.
- Inventory remains partly embedded in CRM/root app files rather than an independent module.
- Documents need a canonical evidence model before Business Memory Intake can be cleanly implemented.
- Butler action safety exists conceptually but still needs explicit permission-tier enforcement tests.
- Compliance has clean engines but no `pkg/compliance/ports.go` yet.
- Cashflow Evidence is still a proposed composition module, not a shipped command center.

## Recommended Next Goal

```text
/goal Complete the AsymmFlow Engine Generalization Inventory without stopping until every major existing engine is classified into pure kernel, domain service, storage adapter, ViewModel, UI surface, and agent surface; Finance, Documents, Butler, Compliance, Inventory, Sync, and Cashflow Evidence reuse opportunities are ranked; the top five reusable kernels and top five product loops are documented; verification/status checks are recorded; and a rollback-safe commit captures the inventory docs.
```

## Exit Criteria

- `docs/CODEX_GOAL_MODULE_CONTRACT_HANDOFF.md` exists and records context, artifacts, commands, verification, risks, and next goal.
- `docs/MODULE_CONTRACT_FOUNDATION.md` exists and defines the module contract.
- `docs/templates/module_manifest.example.json` exists and is valid JSON.
- Finance, Documents, Butler, Compliance, Inventory, and Cashflow Evidence are mapped.
- Existing matching seams and future refactor seams are identified.
- Verification commands are run and recorded.
- A rollback-safe commit is created if the docs-only write boundary is clean.
