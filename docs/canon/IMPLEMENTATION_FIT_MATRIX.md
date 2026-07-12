# Implementation Fit Matrix

Status: Draft v0.1
Created: 2026-05-22
Scope: Choosing implementations under constraints

## Purpose

There is no universally best implementation. There is only best fit under
constraints.

This matrix translates engineering trade-offs into human language so operators,
maintainers, and AI repair agents can make choices without pretending all paths
are equivalent.

## Maturity Labels

| Label | Meaning |
|---|---|
| research | Conceptual or experimental; not operationally reliable |
| prototype | Demonstrates behavior but not durable workflow |
| experimental | Useful but still changing; limited support expectations |
| pilot-ready | Ready for controlled real-world use with support |
| production-used | Used in live operations with known support posture |
| canonical | Preferred upstream path for new work |
| deprecated | Kept for migration/compatibility only |

## Fit Template

Every major implementation option should state:

- best for;
- avoid if;
- trade-offs;
- failure modes;
- maintenance burden;
- switching cost;
- regret risk;
- maturity;
- support posture;
- harness requirements.

## Deployment Fit

| Option | Best for | Avoid if | Main trade-off | Maturity target |
|---|---|---|---|---|
| Local desktop Wails app | owner-led local operations, offline continuity, privacy | many concurrent browser users are primary | strong ownership, weaker browser-native collaboration | canonical for first edition |
| Local server on LAN | small teams, shared office workflows | nontechnical operator cannot manage local server | shared access, more deployment burden | pilot-ready |
| Managed cloud deployment | customers who prefer convenience | sovereignty and offline operation are primary | easier operations, more dependency | optional |
| Hybrid local-first sync | branches, field teams, intermittent internet | conflict policy is not understood | powerful continuity, high sync rigor needed | experimental to pilot-ready |
| Fully cloud SaaS | central hosted workflows | ownership-first promise is central | operational convenience, weaker sovereignty | not default |

## Storage Fit

| Option | Best for | Avoid if | Main trade-off | Maturity target |
|---|---|---|---|---|
| SQLite | local-first app, single-site ownership, simple backup | high-write multi-tenant cloud workload | durable and understandable, limited distributed concurrency | canonical |
| SQLite + sync envelopes | local-first multi-device | conflict semantics are vague | sovereignty plus replication, higher invariants needed | pilot-ready |
| Postgres/Supabase | managed cloud, teams, reporting | local sovereignty is primary | mature cloud DB, operational dependency | optional overlay |
| File-backed exports | support bundles, audit packs, migration | live transactional state | portable evidence, not active database | canonical support path |

## Schema Fit

| Option | Best for | Avoid if | Main trade-off | Maturity target |
|---|---|---|---|---|
| Cap'n Proto | durable contracts, local-first records, generated boundaries | throwaway browser config | stable and fast, generator discipline required | canonical |
| JSON | manifests, browser payloads, third-party interop | durable engine semantics need versioning | easy and inspectable, schema drift risk | canonical for manifests |
| TOON | AI/context packs, compact human-readable summaries | authoritative durable records | high signal for agents, not primary persistence | canonical for agent context |
| CSV | operator exports, accounting handoff | nested evidence semantics | universal, lossy for complex objects | support format |
| PDF | human-facing reports, evidence packs | machine round-trip needed | trustworthy artifact, not structured source | support format |

## AI Fit

| Option | Best for | Avoid if | Main trade-off | Maturity target |
|---|---|---|---|---|
| AI inspect/explain | operator understanding | source context is weak | high value, low authority risk | canonical |
| AI draft/recommend | follow-ups, proposed fixes, migration patches | deterministic validation is missing | speeds work, needs review gates | canonical |
| AI repair agent | fork maintenance, bug fixes | no harness exists | reduces cost, can drift without tests | pilot-ready |
| AI autonomous mutation | none for important state | financial, inventory, legal, destructive actions | high risk | forbidden |

## Overlay Fit

| Overlay | Best for | Avoid if | First proof loop | Maturity target |
|---|---|---|---|---|
| distribution-pack | trading, stock, quotes, invoices, bank reconciliation | pure service businesses with no inventory | quote-to-cash + cashflow evidence | pilot-ready |
| restaurant-pack | cafes, kitchens, delivery reconciliation | enterprise franchise complexity first | daily close + supplier/stock evidence | prototype to pilot-ready |
| salon-pack | appointments, packages, inventory, staff commissions | heavy manufacturing or B2B quotes | booking-to-settlement + membership evidence | prototype |
| startup-pack | small AI/software firms, invoices, tasks, docs | regulated finance as first target | client commitment + invoice + support evidence | prototype |
| home-pack | households as small organizations | enterprise accounting expectations | document/subscription/maintenance memory | prototype |
| compliance-pack | tax/e-invoice/regional policy | law freshness cannot be maintained | policy validation + evidence export | experimental to pilot-ready |

## Regret Risks

High regret choices:

- embedding sector nouns in the kernel;
- allowing AI to mutate authority state;
- treating sync as "just replication" without conflict policy;
- releasing broad overlays without proof loops;
- over-using managed cloud in the default story;
- under-documenting fork support boundaries;
- adding compliance claims before legal update process exists;
- equating generated bindings with durable schema design;
- turning Business Memory into direct record creation without review.

Low regret choices:

- kernel projection harnesses;
- source provenance invariants;
- export/replay bundles;
- deterministic approval gates;
- manifest-driven capability docs;
- migration dry runs;
- AI repair prompts tied to focused harnesses;
- module-level maturity labels.

