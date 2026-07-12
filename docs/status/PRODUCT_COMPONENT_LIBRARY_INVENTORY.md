# Product Component Library Inventory

Status: First implementation slice complete
Created: 2026-05-14
Scope: `frontend/src/lib/components/ui`, `frontend/src/lib/components/layout`, `AccountingScreen.svelte`, `internal/viewmodel/shared`

## Current Reusable Components

The repo already has a mixed but useful reusable surface:

| Family | Current components | Notes |
| --- | --- | --- |
| Enterprise primitives | `Button`, `Card`, `KPICard`, `Badge`, `StatusBadge`, `Tabs`, `Table`, `DataTable` | Good base for forms, dashboards, and tabular work. Some components still use older Svelte event syntax and the indigo enterprise palette. |
| Form primitives | `Input`, `Select`, `Textarea`, `CurrencyInput`, `DatePicker`, `Toggle`, `FormGroup` | Adequate for module forms and correction flows. |
| Wabi primitives | `WabiButton`, `WabiCard`, `WabiBadge`, `WabiStatCard`, `WabiEmptyState`, `WabiSkeleton`, `WabiSpinner`, `WabiModal`, `WabiTooltip` | Matches much of the warm operational surface language already used by AsymmFlow screens. |
| Layout primitives | `PageLayout`, `ModuleLayout`, `Sidebar`, `SplitView`, `Modal` | Useful shell pieces, but product workflow widgets should remain in `ui` or a future `product` namespace. |
| Demo/showcase | `ShowcaseScreen.svelte`, `DesignSystemShowcase.svelte`, `ComponentShowcase.svelte`, `DataTable.demo.svelte` | `ShowcaseScreen.svelte` is the practical fixture route currently wired from `App.svelte`. |

## ViewModel Primitives

`internal/viewmodel/shared/shared_vm.go` currently provides:

- `TableVM`, `TableColumn`, `TableRow`, `TableFilter`
- `DashboardVM`
- `StatusBadgeVM`

Cashflow Evidence already exposes module-specific ViewModel/read-model contracts through `pkg/cashflow/evidence` and `internal/viewmodel/cashflow`. This slice did not need new Go ViewModel primitives because the extracted components consume display-ready browser payloads and emit UI intent only.

## Duplicated Screen-Local Patterns

The Cashflow Evidence command-center in `AccountingScreen.svelte` had reusable product patterns that were not accounting-specific:

| Pattern | Screen-local source | Reusable component |
| --- | --- | --- |
| Compact KPI/status strip | Attention, posting, bank-match, and evidence-pack cells | `KpiStatusStrip.svelte` |
| Evidence source readiness grid | Source label, present/required counts, missing count, confidence, priority | `EvidenceSourceList.svelte` |
| Advisory action proposal card | Source type, proposal label, reason, deterministic service, review controls | `ActionProposalCard.svelte` |

These are product workflow primitives, not accounting authority. They render ViewModel/read-model state and call supplied handlers for operator intent.

## First Extraction Targets

Completed targets:

- `frontend/src/lib/components/ui/KpiStatusStrip.svelte`
- `frontend/src/lib/components/ui/EvidenceSourceList.svelte`
- `frontend/src/lib/components/ui/ActionProposalCard.svelte`
- Exports from `frontend/src/lib/components/ui/index.ts`
- `AccountingScreen.svelte` refactor to consume the extracted components
- `ShowcaseScreen.svelte` fixture section for product operator components

Usage example:

```svelte
<KpiStatusStrip items={cashflowEvidenceMetrics} />
<EvidenceSourceList sources={cashflowEvidence?.evidence_sources || []} />
<ActionProposalCard
  proposal={proposal}
  reviewLabel={proposalReviewStatus(proposal) || proposal.required_deterministic_service}
  hasReview={Boolean(review)}
  reviewing={reviewingCashflowProposal === review?.id}
  onApprove={() => reviewCashflowProposal(proposal, "approved")}
  onNeedsInput={() => reviewCashflowProposal(proposal, "needs_input")}
  onReject={() => reviewCashflowProposal(proposal, "rejected")}
/>
```

## Module Fit Guidance

Use `KpiStatusStrip` when a module needs compact, stable, scan-friendly operator metrics with status or priority: receivables, intake queues, compliance readiness, inventory risk, sync health, and support-bundle checks.

Use `EvidenceSourceList` when a module needs provenance/readiness rows: source type, present versus required evidence, missing counts, confidence, status, and priority.

Use `ActionProposalCard` when a module presents advisory work that still requires deterministic service authority. The component is appropriate for inspect/draft/recommend surfaces; it should not execute domain mutations directly.

## Deferred Components

Later waves should extract:

- `ReviewQueuePanel` for grouped pending/approved/rejected/needs-input/superseded states.
- `AuditTrailList` and `SourceLinkList` once another screen repeats the invoice traceability/evidence-pack pattern.
- `ReadinessGate` for trial-balance, compliance, sync, and import readiness surfaces.
- `ProductEmptyState`, `ProductLoadingState`, and `ProductErrorState` only after aligning the existing Wabi states with the AsymmFlow product palette.
- `PrecisionRecordCell` or similar if long record names, URLs, or multilingual source text start driving layout risk.

## Verification Contract

Minimum gates for component-library changes:

- `npm.cmd --prefix frontend run check`
- `npm.cmd --prefix frontend run build`
- `git diff --check`

Go checks are required only when shared ViewModel or backend contracts change.
