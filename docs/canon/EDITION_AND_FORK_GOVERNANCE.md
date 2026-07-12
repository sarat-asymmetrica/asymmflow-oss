# Edition And Fork Governance

Status: Draft v0.1
Created: 2026-05-22
Scope: AsymmFlow editions, canonical upstream, sovereign downstream forks

## Purpose

This document defines how AsymmFlow should be released, supported, and evolved
as sovereign operational software.

The goal is to avoid SaaS dependency while still making upgrades, support, and
upstream stewardship legible.

## Edition Definition

An edition is a meaningful, source-owned release of the system.

An edition includes:

- runnable software;
- source code;
- deployment instructions;
- schema contracts;
- migrations;
- invariant harnesses;
- overlay packs;
- capability catalog;
- implementation fit matrix;
- AI repair workflows;
- support boundary;
- migration path from prior editions where applicable.

An edition is not a marketing year label. It is a coherent operational machine.

## Edition Release Rule

Release a new edition only when there is meaningful evolutionary change.

Examples of meaningful change:

- new kernel constitution implemented;
- major schema generation/migration model;
- stable overlay pack architecture;
- AI repair harness workflow made canonical;
- new local-first sync architecture;
- major security/runtime improvement;
- substantial capability catalog maturity.

Examples that do not justify a new edition:

- cosmetic UI churn;
- minor feature accumulation;
- generic dashboard additions;
- small bug-fix bundles;
- roadmap pressure;
- subscription retention pressure.

## Maintenance Releases

Maintenance releases may include:

- security fixes;
- critical bugs;
- schema patch guidance;
- migration harness corrections;
- documentation corrections;
- compatibility fixes;
- dependency security updates.

Maintenance releases should not force major operational changes.

## Canonical Upstream Responsibilities

Asymmetrica maintains:

- primitive kernel;
- core operational engines;
- canonical overlays;
- schemas;
- migration guides;
- invariant harnesses;
- reference implementations;
- capability docs;
- security advisories;
- core bug fixes.

Upstream should optimize for clarity, durability, and safe evolution.

## Downstream Fork Responsibilities

Customers or implementers own:

- local deployment;
- operational data;
- local customizations;
- domain-specific automations;
- local integrations;
- fork-specific support;
- testing their modifications before deployment;
- deciding when to upgrade.

Asymmetrica may provide paid help, but does not implicitly own downstream forks.

## Support Boundaries

### Included With Canonical Stewardship

- security advisories;
- upstream bug fixes;
- schema and migration guidance;
- documentation updates;
- reference harness updates;
- critical upstream compatibility notes.

### Paid Separate Work

- fork debugging;
- custom integration work;
- custom workflow design;
- customer-specific migration execution;
- deployment operations;
- feature additions;
- deep architecture review;
- long-running fork maintenance.

## Compatibility Policy

Each release should classify changes:

- compatible patch;
- compatible extension;
- migration required;
- harness-assisted migration required;
- breaking kernel change;
- overlay-only change;
- fork-specific advisory.

Every migration-required change must provide:

- rationale;
- affected schemas;
- affected overlays;
- migration command or script;
- before/after fixtures;
- invariant harness;
- rollback recommendation;
- human review checklist.

## Fork Governance Metadata

Each fork should maintain a local governance file:

```text
FORK_GOVERNANCE.md
```

Recommended sections:

- upstream edition base;
- fork compatibility level;
- local overlays enabled;
- local patches;
- local integrations;
- local schema changes;
- local harnesses;
- upgrade policy;
- support contact;
- known divergence risks.

## AI And Forks

AI repair agents may help maintain forks, but must work through harnesses.

Required flow:

```text
issue detected
-> module identified
-> capability docs consulted
-> relevant harness selected
-> AI patch proposed
-> invariant checks run
-> human reviews
-> local deployment happens
-> fork governance note updated if compatibility changed
```

AI-generated fork changes should never silently modify production data, approve
financial state, post accounting entries, delete records, or rewrite migrations
without deterministic checks and human approval.

## Commercial Posture

This model allows several revenue paths without forcing dependency:

- edition purchase;
- canonical support subscription;
- paid migration support;
- paid custom overlay development;
- paid fork rescue/debugging;
- managed deployment for customers who want it;
- training and implementation services.

The ethical boundary is simple: the customer should be more capable after using
the system, not more trapped.

