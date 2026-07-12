# Capability Catalog Plan

Status: Draft v0.1
Created: 2026-05-22
Scope: Public/internal capability universe for Asymmetrica sovereign software

## Purpose

The ecosystem needs a legible capability universe, not random repo sprawl.

The catalog should show:

- mature reusable engines;
- workflows;
- overlays;
- schemas;
- proof benches;
- implementation fit;
- maturity status;
- human approval boundaries;
- recommended adoption paths.

## Proposed Structure

```text
/capabilities
/reference-implementations
/proof-benches
/schemas
/examples
/docs
/playbooks
/overlays
/editions
```

This can begin as repo-local docs and manifests. It can later become a public
site, generated catalog, or package index.

## Capability Record Contract

Each capability should define:

- capability ID;
- name;
- maturity label;
- problem solved;
- best-fit users;
- inputs;
- outputs;
- kernel primitives used;
- operational engines used;
- overlays using it;
- human approval boundary;
- agent-safe API;
- forbidden agent operations;
- schemas;
- storage adapters;
- UI/ViewModel surfaces;
- harnesses;
- proof artifacts;
- known limitations;
- implementation fit;
- support posture;
- migration notes.

## Initial Capabilities

### Business Memory Intake

Problem:

Messy documents, emails, scans, screenshots, folders, and inbox records lose
operational meaning unless they become reviewable evidence.

Inputs:

- files;
- inbox records;
- OCR maps;
- document classification;
- source assets;
- operator corrections.

Outputs:

- candidates;
- context packs;
- review records;
- source provenance;
- JSON/TOON export bundles.

Human approval boundary:

AI may inspect/explain/draft/recommend. Operators approve links and deterministic
record actions.

Maturity:

pilot-ready foundation after current source registry durability, with UI
ergonomics and event/sync hardening still needed.

### Cashflow Evidence

Problem:

Owners reconstruct receivables, payment evidence, bank allocations, and posting
readiness manually.

Inputs:

- invoices;
- payments;
- bank lines;
- allocations;
- documents;
- posting coverage;
- trial-balance state;
- Business Memory source evidence.

Outputs:

- command center;
- risk rows;
- missing evidence;
- allocation summaries;
- action proposals;
- evidence packs.

Human approval boundary:

AI may explain and draft. Deterministic finance services own posting, matching,
approval, and settlement.

Maturity:

experimental to pilot-ready. Allocation-aware preflight exists; operator loop
closure and real snapshot wiring are next.

### Accounting Posting Spine

Problem:

Business documents need balanced accounting intent and coverage before any
automation touches books.

Inputs:

- customer invoices;
- customer payments;
- supplier invoices;
- supplier payments;
- account mappings;
- chart of accounts.

Outputs:

- posting previews;
- draft journals;
- trial-balance gates;
- coverage reports.

Human approval boundary:

Generated drafts require controlled posting action and review before ledger
authority changes.

Maturity:

pilot-ready foundation.

### Source Asset Registry

Problem:

Source provenance disappears as files become records.

Inputs:

- source kind;
- path;
- label;
- content hash;
- import batch;
- candidate IDs;
- audit refs.

Outputs:

- stable source assets;
- repository records;
- export provenance;
- UI provenance summaries.

Human approval boundary:

Registry writes preserve source evidence; authoritative business actions remain
outside the registry.

Maturity:

pilot-ready foundation.

### Approval Engine

Problem:

Irreversible actions need explicit human authority and audit.

Inputs:

- proposed action;
- actor;
- evidence;
- policy;
- reason;
- correlation ID.

Outputs:

- approval decision;
- rejection;
- audit event;
- deterministic service command.

Maturity:

foundation in current patterns; needs canonical kernel/service extraction.

### Overlay Pack System

Problem:

Sector logic should be attachable without rewriting the kernel.

Inputs:

- kernel primitives;
- engine capabilities;
- domain vocabulary;
- policies;
- fixtures.

Outputs:

- overlay workflow;
- UI language;
- migrations;
- domain reports;
- fork customization seam.

Maturity:

research/prototype until first overlay extraction is harnessed.

## Proof Bench Contract

A proof bench is not only a unit test. It is evidence that a capability preserves
operational truth.

Each proof bench should include:

- fixture data;
- expected canonical state;
- expected export;
- invariant checks;
- adversarial invalid cases;
- migration round-trip if applicable;
- AI repair prompt for common failure;
- elapsed-time tracking for future comparison.

## Catalog Generation Path

Initial repo-local path:

```text
docs/capabilities/*.md
docs/modules/*.manifest.json
docs/proof-benches/*.md
```

Later generated path:

```text
capability manifest
-> catalog index
-> public docs page
-> AI context kit
-> repair-agent prompt pack
```

## Catalog Anti-Drift Rule

A capability may not call itself pilot-ready or canonical unless the catalog
entry links to:

- code paths;
- tests;
- harnesses;
- docs;
- known limitations;
- human approval boundary;
- support posture.

Planning text alone is not maturity evidence.

