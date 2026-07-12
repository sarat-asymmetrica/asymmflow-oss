# Sovereign Substrate Roadmap

Status: Draft v0.1
Created: 2026-05-22
Scope: Six-month-equivalent roadmap depth for AsymmFlow sovereign operational substrate

## Purpose

This roadmap is intentionally not mapped to calendar dates.

It describes the level of architectural, product, testing, migration, and
governance work that a conventional team might spread across roughly six months.
The point is to force rigorous choices, then measure actual AI-assisted
execution time against this depth map instead of accepting inherited assumptions
about refactoring difficulty.

Do not use this document to slow down execution. Use it to remove vagueness.

## Measurement Rule

For every implementation run derived from this roadmap, record:

- starting commit;
- target checkpoint;
- elapsed time;
- files changed;
- tests/harnesses added;
- verification run;
- failed iterations;
- human review needed;
- whether the work was faster, slower, or about equal to expectation;
- discovered blockers;
- next checkpoint.

This creates local evidence about AI-assisted development speed.

## North Star

Transform `asymmflow` from an industrial/trading ERP into the canonical
pre-release sovereign operational substrate.

The target shape:

```text
Primitive Kernel
-> Operational Engines
-> Domain Packs / Overlays
-> Sovereign Forks
```

The first proof loops:

```text
Business Memory Intake + Source Registry
Cashflow Evidence + Accounting Posting Spine
Distribution/industrial overlay as migrated proof domain
```

## Operating Bias

Pre-release is the time to change the bones.

Move boldly, but not carelessly:

- use rollback-by-commit;
- write invariants before broad moves;
- use fixture equivalence;
- keep operator language intact where useful;
- keep sector assumptions out of the kernel;
- do not let AI mutate authority state;
- measure actual work, not imagined work.

## Roadmap Map

### Track A: Doctrine And Governance

Goal:

Make the sovereign software philosophy operationally enforceable.

Checkpoints:

1. Create sovereign software constitution.
2. Define edition and fork governance.
3. Define AI repair-agent workflow.
4. Define implementation fit matrix.
5. Define capability catalog structure.
6. Update existing master roadmap to point at the sovereign substrate map.
7. Add ecosystem-level decision note.

Exit evidence:

- docs exist and are linked;
- no conflicting roadmap is treated as canon;
- future handoffs can cite exact governance docs.

Adversarial tests:

- Can a future agent explain what is upstream versus downstream?
- Can a customer-facing claim avoid SaaS dependency language?
- Can a feature request be classified as kernel, engine, overlay, or fork?
- Can support boundaries be stated without ambiguity?

### Track B: Kernel Constitution And Primitive Mapping

Goal:

Define and implement the primitive kernel without contaminating it with sector
language.

Checkpoints:

1. Complete kernel constitution.
2. Inventory current concrete terms and map them to primitives.
3. Create `pkg/kernel` skeleton with identity, evidence, events, approval,
   money, asset, workflow, policy, and timeline packages.
4. Add constructor and invariant tests for each primitive.
5. Add migration projection fixtures from current records into kernel concepts.
6. Add schema strategy for kernel records.
7. Add docs showing which concrete terms remain overlay vocabulary.

Exit evidence:

- kernel primitives compile and test;
- current terms have mapping tables;
- no sector-specific concept is required by kernel tests;
- UI language can remain concrete through overlay projections.

Adversarial tests:

- Can a restaurant order, salon appointment, household warranty, and purchase
  order all map without changing kernel?
- Can a supplier invoice be represented without making `SupplierInvoice` a
  primitive?
- Does evidence survive projection and round-trip?
- Can agents interact with kernel objects without authority mutation?

### Track C: Harness And Migration Infrastructure

Goal:

Make broad AI-assisted refactoring safe through authoritative harnesses.

Checkpoints:

1. Create proof-bench folder and template.
2. Create before/after equivalence harness pattern.
3. Add kernel projection round-trip harness.
4. Add source provenance preservation harness.
5. Add money precision and allocation harness.
6. Add approval boundary harness.
7. Add generated schema churn review checklist.
8. Add AI repair prompt packs tied to harnesses.
9. Add elapsed-time measurement log template.

Exit evidence:

- every large refactor has a harness category;
- future agents can select tests from failure type;
- migration work can be measured.

Adversarial tests:

- Can a migration fail loudly when source evidence is lost?
- Can a generated binding churn be detected and reviewed?
- Can an AI repair pass be rejected for weakening invariants?
- Can broad package moves preserve operator-visible behavior?

### Track D: Business Memory As Evidence Spine

Goal:

Promote Business Memory from intake module to reusable evidence spine.

Current state:

- canonical candidates exist;
- source registry is durable and runtime-visible;
- review records and export/replay exist;
- agent actions are bounded.

Checkpoints:

1. Define Business Memory's relationship to kernel `Evidence`.
2. Add kernel evidence projections for source assets and candidates.
3. Add event envelope for candidate reviewed, source upserted, context assembled,
   link requested.
4. Add module-aware sync impact notes and conflict policy draft.
5. Harden UI provenance and review ergonomics.
6. Add export bundle index for support/migration use.
7. Add adversarial tests for provenance loss, duplicate sources, privacy class
   preservation, and agent forbidden actions.
8. Add proof bench for messy folder/email/PDF to reviewed candidate.

Exit evidence:

- Business Memory can serve overlays without becoming sector-specific;
- provenance survives export/replay;
- review actions are deterministic and auditable;
- AI remains inspect/explain/draft/recommend only.

Adversarial tests:

- Duplicate source assets merge without losing audit refs.
- Candidate review cannot be recorded by agent actor.
- Export/replay preserves source identity.
- Privacy class survives context pack generation.

### Track E: Cashflow Evidence Closed Operator Loop

Goal:

Turn Cashflow Evidence from command-center proof into a closed operator loop.

Current state:

- read model exists;
- allocation-aware preflight exists;
- action proposals exist;
- posting coverage/trial balance can be surfaced;
- review state exists in earlier cashflow review work.

Checkpoints:

1. Wire real snapshot sources for invoices, payments, bank allocations,
   documents, and posting coverage.
2. Add drilldown from risk/coverage/allocation rows to source records.
3. Add proposal states: draft, needs input, approved for deterministic action,
   rejected, exported.
4. Add export support bundle with cash exposure, missing evidence, allocation
   state, proposal decisions, posting coverage, and trial-balance state.
5. Add deterministic request seam for draft journal or follow-up creation,
   behind approval.
6. Add event refresh/subscription boundary.
7. Add UI operator loop: inspect, review, approve request, export.
8. Add proof bench for overdue invoice with partial bank allocation and missing
   source document.

Exit evidence:

- operator ends with action/export/review state, not just dashboard insight;
- no hidden auto-posting;
- posting authority remains deterministic;
- evidence pack can be handed to accountant/operator.

Adversarial tests:

- Partial allocation cannot be reported as full settlement.
- Draft posting request cannot bypass trial-balance gate.
- Agent cannot approve or post.
- Missing evidence is preserved in export, not hidden by summary.

### Track F: Accounting Posting And Settlement Kernel

Goal:

Generalize Wave 17 posting work into a reusable settlement/posting engine.

Checkpoints:

1. Map posting concepts to kernel Money Movement and Settlement.
2. Add period-lock checks before draft creation/posting.
3. Add controlled posting action for generated drafts.
4. Add batch backfill workflow for historical documents missing journal links.
5. Add support-bundle export for posting coverage.
6. Add reversal and correction policy draft.
7. Add property tests for balanced entries and currency precision.
8. Add overlay mapping for invoices/payments as settlement projections.

Exit evidence:

- posting remains deterministic;
- generated drafts can become posted only through controlled approval;
- coverage gaps are visible;
- settlement concepts are not tied to one document type.

Adversarial tests:

- unbalanced entries are rejected;
- duplicate source drafts remain idempotent;
- period-locked posting fails;
- account mapping fallback is explicit and logged.

### Track G: Overlay Extraction

Goal:

Reclassify current PH/trading/instrumentation logic as an overlay and prove the
kernel supports it.

Checkpoints:

1. Define `distribution-pack` or `industrial-trading-pack` overlay manifest.
2. Map concrete workflows to kernel primitives.
3. Identify overlay-owned vocabulary and UI language.
4. Extract overlay fixtures from current workflow tests.
5. Add adapter seams from current records to kernel projections.
6. Move only after equivalence harnesses exist.
7. Add overlay proof bench for quote-to-cash and receive-to-pay.
8. Preserve current operator terminology where it aids clarity.

Exit evidence:

- current business behavior is preserved;
- sector language is outside the kernel;
- overlay has fixtures, harnesses, and support boundary;
- future overlays have a reference pattern.

Adversarial tests:

- quote revision identity survives projection;
- delivery/serial traceability survives projection;
- supplier invoice settlement remains auditable;
- old UI terms can be backed by kernel primitives.

### Track H: Inventory And Asset Evidence Ledger

Goal:

Start from pure stock/asset evidence primitives rather than old inventory screen
assumptions.

Checkpoints:

1. Define asset/evidence ledger handoff.
2. Implement pure asset identity and movement kernel.
3. Map serial numbers, GRNs, delivery notes, stock movements, warranties, and
   household assets to asset primitives.
4. Add valuation policy as overlay, not kernel.
5. Add tests for receipt, reservation, delivery, transfer, adjustment, and
   evidence link.
6. Add source asset links from Business Memory.
7. Add read-only readiness ViewModel.
8. Add overlay-specific inventory UI only after kernel proof bench.

Exit evidence:

- stock/asset truth is evidence-linked;
- valuation is policy-driven;
- serial/lot/custody evidence can generalize beyond industrial trading;
- no current inventory mutation is changed without harness.

Adversarial tests:

- reservation cannot exceed available policy unless explicitly allowed;
- delivery cannot lose source/custody evidence;
- valuation method is explicit;
- serial identity remains stable across migration.

### Track I: Agent-Safe Module Surface

Goal:

Prove all AI-facing surfaces are useful but non-authoritative.

Checkpoints:

1. Create shared agent-surface contract.
2. Add actor typing across Business Memory and Cashflow actions.
3. Add denial tests for approve, link, post, delete, reverse, file, and create
   authoritative record.
4. Add suggestion-to-approval audit trail.
5. Add TOON context requirements for source citation.
6. Add Butler/Codex distinction if needed.
7. Add repair-agent workflow into capability catalog.

Exit evidence:

- AI can assist evolution and operations without authority bypass;
- every accepted suggestion links evidence, actor, and deterministic command;
- denial tests exist for important mutation classes.

Adversarial tests:

- an agent actor cannot record review decision;
- an agent proposal cannot post a journal;
- an agent cannot weaken policy to make action pass;
- context packs cite missing evidence honestly.

### Track J: Capability Catalog And Public Surface

Goal:

Make the ecosystem legible to humans and future agents.

Checkpoints:

1. Create repo-local capability catalog entries.
2. Add capability maturity labels.
3. Link each capability to code paths, tests, docs, and harnesses.
4. Add implementation fit matrix entries.
5. Generate or maintain catalog index.
6. Add public-facing copy that uses sovereign software language.
7. Add reference implementation pages for proof loops.
8. Add AI context-kit export path for capability docs.

Exit evidence:

- future Codex instances can discover what exists and what maturity it has;
- humans can evaluate fit without enterprise cosplay;
- capabilities have proof artifacts, not just names.

Adversarial tests:

- no capability claims pilot-ready without linked evidence;
- limitations are visible;
- human approval boundary is explicit;
- implementation fit includes avoid-if and regret-risk language.

### Track K: Edition Packaging

Goal:

Package AsymmFlow 2026 Edition as source-owned software, not SaaS.

Checkpoints:

1. Define edition manifest.
2. Define included source/docs/schemas/harnesses.
3. Define install/deploy modes.
4. Define support boundary.
5. Add fork governance template.
6. Add backup/restore and migration preflight.
7. Add release verification script updates for sovereign edition artifacts.
8. Add support bundle index covering Business Memory, Cashflow, Posting,
   Overlay, Kernel, and Fork metadata.
9. Run release harness.

Exit evidence:

- edition is understandable as an owned machine;
- customer/fork responsibilities are explicit;
- upstream maintenance boundary is explicit;
- release does not depend on SaaS subscription logic.

Adversarial tests:

- can a customer understand what they own?
- can a fork record divergence?
- can a maintenance release avoid forced upgrade pressure?
- can restore/migration be rehearsed?

## Current Best Next Implementation Chain

The most useful next autonomous chain is:

1. Track A docs and roadmap wiring.
2. Track B kernel skeleton and primitive tests.
3. Track C proof-bench scaffolding.
4. Track D Business Memory evidence projection.
5. Track E Cashflow closed operator loop.
6. Track G distribution overlay manifest and mapping.

This chain moves from doctrine to kernel to harness to proof loops. It avoids
both vague philosophy and premature feature accumulation.

## Stop Conditions

Stop a run only when:

- the selected checkpoint ladder is complete and a promoted next checkpoint has
  been started or explicitly queued;
- verification fails in a way that needs human product/architecture judgment;
- a real external dependency is missing;
- continuing would touch red-zone files without approval;
- a migration risk cannot be represented by a harness yet.

Do not stop merely because:

- one doc exists;
- one test passes;
- one package compiles;
- the work seems broad;
- conventional assumptions suggest it should take longer.

## Final Definition Of Done

The six-month-equivalent roadmap is complete when:

- kernel constitution has implementation teeth;
- current industrial/trading behavior is an overlay, not kernel gravity;
- Business Memory and Cashflow Evidence are closed proof loops;
- AI repair workflow is harness-guided;
- capability catalog is legible;
- edition/fork governance is explicit;
- release packaging supports ownership;
- actual elapsed-time evidence exists for major checkpoint classes.

