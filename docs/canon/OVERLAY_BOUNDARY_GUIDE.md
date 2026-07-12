# Overlay Boundary Guide

Status: Draft v0.1
Created: 2026-05-22
Scope: Domain packs, compliance packs, and sovereign forks

## Purpose

This guide defines how AsymmFlow domain logic should sit above the primitive
kernel without contaminating it.

The current repo began as a serious industrial/trading/instrumentation system.
That work remains valuable. It should become the first real overlay proving the
kernel can support demanding operational workflows.

## Boundary Model

```text
Primitive Kernel: universal operational truths
Operational Engines: reusable logic over kernel objects
Domain Packs / Overlays: sector language, workflows, policy, UI, integrations
Sovereign Forks: customer-specific modifications and local reality
```

## What Belongs In An Overlay

An overlay may contain:

- sector vocabulary;
- operator-facing nouns;
- domain workflows;
- default screen composition;
- reports and exports;
- compliance mapping;
- integrations;
- fixtures;
- seed data;
- policy defaults;
- local conventions;
- domain-specific invariants;
- migration adapters from legacy terms.

## What Must Not Belong In An Overlay

An overlay should not redefine:

- identity semantics;
- core evidence provenance;
- base approval semantics;
- money precision;
- event envelope;
- permission model;
- timeline model;
- AI authority boundary;
- migration and harness protocol.

Those belong in the kernel or operational engine layer.

## Initial Overlay Targets

### Tier 1

High-fit early overlays:

- restaurants;
- cafes;
- delivery kitchens;
- salons;
- wellness/spa;
- small AI/software startups;
- general trading/distribution SMBs;
- AsymmFlow Home.

Why these first:

- high operational density;
- owner-led workflows;
- fragmented tooling;
- low enterprise inertia;
- real document/message chaos;
- strong local-first benefit;
- clear approval and evidence loops.

### Tier 2

Later overlays:

- procurement teams;
- field service;
- creator/media teams;
- light manufacturing;
- small clinics, if compliance burden is understood.

### Tier 3

Avoid early gravity unless a specific client justifies it:

- heavy industrial instrumentation;
- deep enterprise procurement;
- large enterprise integrations;
- certification-first enterprise ecosystems.

The current PH/trading/instrumentation use case should become a proof overlay,
not the center of gravity for all architecture.

## Overlay Contract

Each overlay must define:

- overlay ID;
- kernel primitives used;
- operational engines used;
- domain vocabulary;
- workflows;
- policies;
- permissions;
- UI surfaces;
- evidence sources;
- exports;
- invariant suite;
- fixtures;
- migration adapters;
- AI-safe commands;
- support boundary;
- maturity status.

## Example: Distribution Pack

Operator terms:

- RFQ;
- quotation;
- customer order;
- purchase order;
- GRN;
- delivery note;
- serial traceability;
- customer invoice;
- supplier invoice.

Kernel mapping:

- RFQ -> Request;
- quotation -> Offer;
- order / PO -> Commitment;
- GRN / delivery note -> Transfer + Evidence;
- serial number -> Asset identity;
- invoice -> Settlement request;
- payment -> Money Movement + Settlement;
- costing approval -> Policy + Approval;
- customer/supplier -> Party roles.

Operational engines:

- OCR/classification;
- Business Memory;
- Cashflow Evidence;
- posting;
- matching/reconciliation;
- timeline;
- approval;
- sync.

## Example: Restaurant Pack

Operator terms:

- menu item;
- order;
- kitchen ticket;
- stock prep;
- delivery app payout;
- supplier purchase;
- wastage;
- shift close.

Kernel mapping:

- order -> Commitment;
- kitchen ticket -> Workflow task;
- stock prep -> Transfer / Asset movement;
- payout -> Money Movement + Settlement;
- wastage -> Asset adjustment + Evidence;
- shift close -> Settlement + Approval;
- supplier purchase -> Request / Commitment / Transfer.

## Example: Home Pack

Operator terms:

- household bill;
- subscription;
- warranty;
- repair;
- shopping list;
- family task;
- maintenance timeline;
- document folder.

Kernel mapping:

- bill -> Settlement request;
- subscription -> Recurring Commitment;
- warranty -> Evidence + Policy;
- repair -> Request + Commitment + Transfer;
- shopping list -> Request;
- task -> Workflow item;
- maintenance timeline -> Timeline;
- document folder -> Source Asset / Evidence set.

## Compliance Overlays

Compliance must be attachable policy, not foundation.

Examples:

- India GST Pack;
- Bahrain VAT Pack;
- Saudi e-invoice Pack;
- local payroll or income-tax pack.

Compliance overlays should provide:

- policy version;
- jurisdiction;
- effective dates;
- affected primitives;
- calculation engine;
- validation engine;
- filing/export workflow;
- evidence requirements;
- operator approval boundary;
- legal update process.

Compliance overlays must not rewrite kernel money, evidence, identity, or event
semantics.

## Fork Layer

Sovereign forks may add:

- local fields;
- custom workflows;
- local integrations;
- custom reports;
- customer-specific automation;
- domain-specific UI copy;
- private deployment scripts;
- internal role policies.

Forks should not directly modify canonical kernel behavior unless the fork is
intentionally leaving upstream compatibility.

## Fork Compatibility Levels

| Level | Meaning | Upstream support expectation |
|---|---|---|
| Clean overlay config | Uses canonical extension points | High |
| Extension package | Adds modules without kernel changes | Medium-high |
| Patched overlay | Modifies overlay code | Medium |
| Patched engine | Modifies reusable engine | Low-medium |
| Patched kernel | Changes primitive semantics | Low and explicit |
| Divergent fork | No longer migration-compatible | Custom consulting only |

## Overlay Migration Rule

Every extraction from current concrete code must produce:

- mapping table from old terms to kernel primitives;
- before/after fixture;
- invariant suite;
- adapter seam;
- migration notes;
- rollback plan;
- known semantic loss, if any.

No overlay migration is complete because code moved. It is complete when the
semantic harness proves that operational truth survived the move.

