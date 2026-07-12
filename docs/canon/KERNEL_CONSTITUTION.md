# AsymmFlow Kernel Constitution

Status: Draft v0.1
Created: 2026-05-22
Scope: Primitive kernel extraction for sovereign operational software

## Purpose

This document defines what belongs in the AsymmFlow primitive kernel and what
must remain outside it.

The kernel contains universal operational truths. It must not contain sector
assumptions, industrial instrumentation assumptions, trading-company language,
restaurant workflows, home workflows, or compliance-specific policy.

The kernel should remain stable enough that domain packs and sovereign forks can
evolve without repeatedly rewriting foundational concepts.

## Layer Model

```text
Layer 1: Primitive Kernel
Layer 2: Operational Engines
Layer 3: Domain Packs / Overlays
Layer 4: Sovereign Forks
```

The kernel is not the application. It is the durable language from which
applications can be composed.

## Kernel Admission Rule

A concept may enter the kernel only if it is:

- cross-domain;
- needed by multiple overlays;
- stable across release editions;
- expressible without sector vocabulary;
- testable through invariants;
- useful for continuity, migration, or audit.

If a concept only makes sense in one industry or compliance regime, it belongs
in an overlay.

## Kernel Rejection Rule

Do not put these directly in the kernel:

- PurchaseOrder;
- Quotation;
- SupplierInvoice;
- DeliveryNote;
- GRN;
- RestaurantOrder;
- SalonAppointment;
- GSTReturn;
- VATInvoice;
- IndustrialInstrument;
- RFQ as a sector-specific sales artifact.

Those terms may remain valid in overlays and UI language. They must not become
primitive constitutional concepts.

## Primitive Vocabulary

The first kernel vocabulary should be small, tested, and composable.

### Actor

A human, service, agent, device, organization, or process that can observe,
propose, approve, execute, or audit an action.

Kernel obligations:

- identity;
- role or actor type;
- permission claims;
- authority level;
- audit presence.

### Party

An entity that participates in commitments, transfers, settlements, evidence,
or communication.

Kernel obligations:

- stable identity;
- names and aliases;
- relationship roles;
- contact channels;
- risk/compliance flags as attachable attributes, not fixed sector fields.

### Evidence

A source-backed fact, document, observation, message, artifact, image, scan,
file, statement, or operator note.

Kernel obligations:

- source identity;
- provenance;
- content hash where available;
- privacy class;
- confidence/status;
- linked objects;
- audit references.

### Request

A solicited or internally created expression of need.

Examples in overlays:

- RFQ;
- purchase request;
- service request;
- household maintenance request;
- kitchen stock request.

### Offer

A proposed exchange, service, delivery, or response to a request.

Examples in overlays:

- quotation;
- vendor offer;
- service estimate;
- subscription plan;
- household contractor quote.

### Commitment

A mutually accepted obligation or operational promise.

Examples in overlays:

- customer order;
- purchase order;
- service booking;
- delivery commitment;
- maintenance schedule.

### Transfer

A movement of money, goods, assets, documents, responsibility, or operational
state between parties, locations, accounts, or custody contexts.

Examples in overlays:

- stock movement;
- bank allocation;
- payment;
- delivery;
- handoff;
- asset assignment.

### Settlement

A reconciliation of obligations against transfers, evidence, accounts, or
policy.

Examples in overlays:

- invoice payment settlement;
- supplier payment match;
- reimbursement closure;
- tax filing reconciliation;
- subscription charge review.

### Approval

A deterministic authority transition from proposal to allowed action.

Kernel obligations:

- actor;
- subject;
- decision;
- reason;
- timestamp;
- correlation ID;
- before/after or action intent.

### Policy

A rule set that constrains actions, classifications, calculations, or routing.

Kernel obligations:

- version;
- jurisdiction/domain scope when applicable;
- effective period;
- evidence requirements;
- violation status;
- override/approval rules.

### Event

A durable fact that something happened.

Kernel obligations:

- event type;
- subject;
- actor;
- occurred time;
- correlation ID;
- payload schema;
- source evidence or command reference.

### Timeline

An ordered view of events, evidence, decisions, transfers, and commitments for
an operational object.

### Workflow

A typed progression of states, decisions, approvals, actions, and evidence
requirements.

### Document

A structured or unstructured artifact with source identity and operational
meaning.

Document is a kernel primitive only as an evidence-bearing artifact. Specific
document types belong in overlays.

### Asset

A thing with identity, custody, lifecycle, value, maintenance, or evidence.

Asset may represent physical inventory, equipment, warranty items, household
assets, software licenses, or operational resources.

### Money Movement

A transfer of value across accounts, parties, obligations, or settlements.

Money primitives must support multiple currencies, evidence, allocations,
posting intent, and settlement references without embedding one jurisdiction's
tax model.

## Mapping From Current Concrete Terms

| Current term | Kernel primitive | Likely overlay |
|---|---|---|
| RFQ | Request | distribution-pack |
| Quotation / Offer | Offer | distribution-pack |
| Customer Order | Commitment | distribution-pack |
| Purchase Order | Commitment | procurement overlay |
| Supplier Invoice | Evidence + Settlement request | finance/procurement overlay |
| Customer Invoice | Settlement request | finance overlay |
| Payment | Money Movement + Settlement | finance overlay |
| Delivery Note | Transfer + Evidence | fulfillment overlay |
| GRN | Transfer + Evidence | inventory/procurement overlay |
| Serial Number | Asset identity | asset/inventory overlay |
| Bank Statement Line | Evidence + Money Movement source | finance overlay |
| Business Memory Candidate | Evidence review object | documents engine / kernel evidence bridge |
| Cashflow Action Proposal | Proposed Workflow action | cashflow engine |

## Kernel Package Direction

Candidate future packages:

```text
pkg/kernel/identity
pkg/kernel/evidence
pkg/kernel/events
pkg/kernel/workflow
pkg/kernel/approval
pkg/kernel/money
pkg/kernel/asset
pkg/kernel/policy
pkg/kernel/timeline
```

This package map is a target, not a command to immediately move all existing
code. Migration must proceed through harness-backed equivalence checks.

## Kernel Test Strategy

Every kernel primitive needs:

- constructor validation tests;
- equality/stable identity tests;
- serialization round-trip tests;
- invariant tests;
- migration fixtures from current overlay terms;
- fuzz or property tests for identifiers and monetary values where useful;
- adversarial invalid input tests;
- agent-boundary tests when the primitive can be acted on.

## Kernel Invariants

Initial invariant families:

- identity is stable and collision-resistant within its scope;
- evidence never loses source identity during transformation;
- approvals cannot be recorded without actor, subject, decision, and reason;
- money movement values preserve currency and precision;
- settlements cannot silently exceed available obligations without explicit
  policy;
- events require correlation IDs when crossing module boundaries;
- workflows distinguish proposed, approved, executed, rejected, and reversed
  states;
- agents cannot directly record authority transitions.

## Migration Principle

Do not begin with a giant renaming exercise.

Begin by creating kernel vocabulary and equivalence harnesses, then migrate
concrete modules behind adapters:

```text
current concrete record
-> kernel projection
-> overlay projection
-> round-trip / invariant check
-> package extraction
-> UI language preserved where useful
```

The UI may continue to say "Purchase Order" where operators expect it. The
kernel should understand it as a domain projection of `Commitment`.

