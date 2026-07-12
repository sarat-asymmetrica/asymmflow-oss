# Architecture Decision Records (ADRs)

Short, durable records of significant architecture decisions: the context, the
options weighed, the call, and the consequences. An ADR is **law once Accepted** —
it explains *why* the architecture is the way it is, so a future reader (human or
agent) doesn't reopen a settled question or resurrect a dead argument.

## Convention

- **One file per decision**, named `ADR-NNN-short-slug.md` (zero-padded number).
- **Status lifecycle:** `Proposed` → `Accepted` → (later) `Superseded by ADR-XXX`
  or `Deprecated`. Never delete an ADR; supersede it so the history stays legible.
- **Format:** Status / Date / Deciders / Context / Decision drivers / Considered
  options / Decision outcome / Consequences (positive, negative, neutral) /
  Compliance check / Citations. (Loosely [MADR](https://adr.github.io/madr/).)
- **Cite ground truth.** Reference real `file:line` and the invariants in
  `CLAUDE.md`, so the decision is checkable, not vibes.
- Numbers are **append-only** and shared with the historical inline ADRs in
  `docs/roadmap/SOVEREIGN_INFRASTRUCTURE_VISION.md` (ADR-001..004). When one of
  those pending ADRs is ratified, give it a real file here and mark the inline
  section `RESOLVED — see ADR-NNN`.

## Index

| ADR | Title | Status |
|---|---|---|
| [ADR-001](ADR-001-persistence-and-sync-stack.md) | Persistence & sync stack — ncruces SQLite, pluggable sync, PocketBase rejected | **Accepted** (2026-06-15) |
| ADR-002 | Built-in DuckDNS / DDNS client | Proposed *(inline in vision doc)* |
| ADR-003 | Raspberry Pi sovereign-server product | Exploratory *(inline in vision doc)* |
| ADR-004 | CRDT collaborative layer (collab only, never ledgers) | Future *(inline in vision doc)* |
