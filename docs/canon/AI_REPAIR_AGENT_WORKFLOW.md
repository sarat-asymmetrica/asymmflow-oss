# AI Repair Agent Workflow

Status: Draft v0.1
Created: 2026-05-22
Scope: Harness-guided AI coding and repair for sovereign forks and canonical upstream

## Purpose

This document defines how AI should help evolve AsymmFlow without becoming an
uncontrolled authority.

AI is valuable because it reduces maintenance, migration, integration, and
debugging cost. It is not allowed to bypass deterministic services, invariant
harnesses, or human approval.

## Authority Boundary

Allowed AI actions:

- inspect;
- explain;
- classify;
- summarize;
- draft;
- recommend;
- scaffold;
- propose patches;
- propose migrations;
- generate fixtures;
- select relevant harnesses;
- produce operator explanations.

Forbidden AI actions:

- approve authoritative actions;
- post accounting entries;
- mutate production business state;
- delete records;
- file tax returns;
- reverse transactions;
- deploy without human approval;
- weaken invariants to make tests pass;
- silently modify migrations or schemas;
- conceal uncertainty.

## Standard Repair Loop

```text
issue detected
-> classify failure domain
-> identify module/capability
-> load capability docs and harness guide
-> reproduce with fixture or failing test
-> generate patch plan
-> apply smallest coherent patch
-> run focused tests
-> run adjacent invariant checks
-> run build/typecheck
-> update docs or fork governance if behavior changed
-> human reviews diff and evidence
-> deploy locally if approved
```

## Repair Request Shape

Every AI repair request should include:

- repo and branch;
- upstream edition or commit;
- fork compatibility level;
- observed symptom;
- expected behavior;
- impacted module;
- known recent local changes;
- production data sensitivity;
- allowed write zones;
- forbidden write zones;
- verification gates;
- rollback point;
- required final ledger.

## Harness Selection Matrix

| Failure type | First harness | Adjacent harness |
|---|---|---|
| Kernel primitive failure | kernel unit/property tests | schema round-trip, migration fixtures |
| Business Memory intake failure | `go test ./pkg/documents/...` | adapter docs tests, ViewModel tests |
| Source provenance loss | source registry tests | export/replay tests, UI ViewModel tests |
| Cashflow calculation failure | `go test ./pkg/cashflow/evidence` | posting tests, finance service focused tests |
| Posting/accounting failure | `go test ./pkg/finance/posting` | trial-balance gate, draft journal tests |
| UI command mismatch | ViewModel tests | frontend check, action inventory review |
| Schema change failure | `schemas/generate.ps1 -CheckOnly` | adapter round-trip tests |
| Sync/conflict issue | sync package tests | event envelope tests, migration fixtures |
| Overlay migration issue | overlay fixture equivalence | kernel projection round-trip |
| Fork integration issue | local integration test | capability fit matrix and fork governance review |

## AI Patch Constraints

AI must:

- read current source before editing;
- preserve unrelated dirty work;
- state pure core versus side-effect boundary;
- keep edits inside assigned authority zones;
- add or update tests before claiming behavior;
- separate baseline warnings from new regressions;
- report exact commands run;
- report tests not run;
- preserve human approval for authoritative actions.

AI must not:

- use broad rewrites without harnesses;
- normalize unrelated files;
- hide generated churn;
- treat planning docs as proof of shipped behavior;
- claim release readiness from focused tests alone;
- convert domain overlays into kernel primitives merely for naming elegance.

## Adversarial Verification Levels

### Level 0: Static Read

Use for docs, planning, and low-risk copy changes.

Required:

- source/doc read;
- diff review;
- link/path verification.

### Level 1: Focused Unit Harness

Use for kernel or pure-engine changes.

Required:

- focused package tests;
- invalid input tests;
- deterministic fixture checks.

### Level 2: Boundary Harness

Use when storage, schemas, ViewModels, or exports change.

Required:

- focused package tests;
- adapter/storage tests;
- schema generation or check-only gate;
- ViewModel tests;
- manifest parse if applicable.

### Level 3: Workflow Harness

Use for user-visible product loops.

Required:

- backend focused tests;
- build;
- frontend type/check gate;
- UI smoke or screenshot where feasible;
- export/replay check;
- action/permission boundary check.

### Level 4: Migration Harness

Use for schema or overlay migrations.

Required:

- before/after fixtures;
- migration dry run;
- rollback rehearsal where feasible;
- invariant checks;
- generated artifact review;
- human migration approval checklist.

### Level 5: Release Harness

Use for edition packaging.

Required:

- release script;
- build artifact;
- support bundle;
- security/dependency review;
- baseline warnings ledger;
- install/run smoke;
- restore preflight;
- docs completeness check.

## Prompt Template

```text
You are repairing [module] in [repo].

Goal:
[specific outcome]

Authority:
Green: [paths]
Amber: [paths requiring rationale]
Red: [forbidden paths]

Inputs:
- symptom:
- expected behavior:
- relevant capability doc:
- relevant harness:
- starting commit:
- dirty status:

Rules:
- preserve unrelated changes;
- do not weaken invariants;
- do not mutate authoritative business state;
- do not claim tests not run;
- update docs if behavior changes.

Verification:
- [commands]

Final ledger:
- changed paths;
- tests run;
- tests not run;
- known risks;
- rollback point;
- next recommended checkpoint.
```

## Metrics

Track actual AI repair performance:

- elapsed time;
- files touched;
- tests added;
- tests run;
- first-pass pass/fail;
- number of repair iterations;
- generated churn volume;
- human review comments;
- post-merge defects;
- whether estimate was high, low, or accurate.

The purpose is not to prove AI is always fast. The purpose is to replace
training-data assumptions about refactoring difficulty with repo-specific
evidence.

