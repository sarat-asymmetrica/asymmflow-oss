# ADR-001: Persistence & Sync Stack — ncruces SQLite, pluggable sync, PocketBase rejected

- **Status:** Accepted
- **Date:** 2026-06-15
- **Deciders:** AsymmFlow ecosystem dev sprint (Phase 4 — infra decision gate)
- **Supersedes:** the *pending* "ADR-001: PocketBase vs Turso vs PostgreSQL"
  inline section of [`docs/roadmap/SOVEREIGN_INFRASTRUCTURE_VISION.md`](../../roadmap/SOVEREIGN_INFRASTRUCTURE_VISION.md)
- **Related invariants:** CLAUDE.md #3 (No CGO), #4 (AI-authority boundary), #5
  (financial semantics are sacred); `docs/architecture/TARGET_ARCHITECTURE.md`
  (names Turso embedded replicas as the target sync layer)

---

## Context and problem statement

The sovereign-infrastructure vision doc left the database/persistence decision
explicitly open ("Status: NEEDS DECISION"), recommending that AsymmFlow ship
**PocketBase as the zero-config default**, with PostgreSQL and Turso as
alternatives behind adapter interfaces. That recommendation was written as
research, never ratified, and predates the work the codebase has actually
shipped. This ADR closes the gate.

The question to settle: **what is the canonical local store, what is the sync
tier, and is PocketBase in or out?**

### Ground truth in the repo as of this decision

| Layer | What's actually there | Evidence |
|---|---|---|
| **Local store** | Pure-Go **ncruces** SQLite + its `gormlite` GORM dialector. WAL, `busy_timeout=5000`, `synchronous=NORMAL`, `foreign_keys=ON`, mmap. Single `ph_holdings.db`. | `go.mod:16-17`; DSN + `gorm.Open` at `app.go:483`; path resolution in `config.go` (`getDatabasePath`) |
| **CGO** | **None.** Zero `import "C"`, zero `#cgo`. SQLite runs as pure-Go WASM (`go-sqlite3-wasm/v2`). CGO elimination already shipped in Wave 7 (mattn/go-sqlite3 fully removed). | grep across all `*.go`; `go.mod` (no mattn-sqlite); CLAUDE.md invariant #3 |
| **Live sync today** | **Supabase / PostgreSQL** via pure-Go `pgx`, through `DBManager` + `DBSyncService`. Env-gated by `ENABLE_CLOUD_SYNC`; **off by default / fully offline** when no credentials. | `gorm.io/driver/postgres` in `go.mod`; `db_manager.go:103`; `loadDatabaseConfig` in `config.go` |
| **Turso** | The **pure-Go HTTP client** `tursodatabase/libsql-client-go` is in `go.mod`. Code exists in `pkg/sync/turso/` + `pkg/sync/engine/` but is **stubbed and unwired** — `Sync()` returns `nil`, no production importer. The CGO embedded driver `go-libsql` is **absent** from go.mod/go.sum. | `go.mod:22`; `pkg/sync/turso/client.go:79-81`; no non-test importer of `pkg/sync/engine` |
| **PocketBase** | **Vaporware.** Zero code, zero deps, zero go.mod entry. Mentioned only in one roadmap doc as a pending recommendation. | only `docs/roadmap/SOVEREIGN_INFRASTRUCTURE_VISION.md` |

### A ghost we killed before it bit us

The earlier framing assumed **PocketBase requires CGO** (via mattn/go-sqlite3)
and therefore violates invariant #3. **That is no longer true.** Current
PocketBase defaults to the pure-Go `modernc.org/sqlite` driver and only pulls
mattn/CGO when `CGO_ENABLED=1`. So PocketBase *can* build CGO-free.

We therefore do **not** reject PocketBase on a CGO technicality (it would not
survive scrutiny). We reject it on architectural grounds that do — below.

---

## Decision drivers

- **Invariant #3 — No CGO.** Non-negotiable, already enforced by construction.
- **Invariant #4 — AI-authority boundary / sovereignty.** We own identity, RBAC,
  licensing, and field encryption today. The substrate's whole thesis is
  *sovereign forks* — adopters own their authority surface.
- **Invariant #5 / #6 — financial semantics are sacred; keep it green.** The
  foundational persistence layer of a financial ledger is the highest-blast-radius
  component in the system. Changes here are stop-and-ask, not convenience-driven.
- **Offline-first, single-binary, zero-config** for low-connectivity / low-IT
  deployment contexts (the substrate is meant to run on a cheap laptop with no
  server and intermittent network).
- **Don't re-platform a shipped, tested foundation without a capability gain.**

---

## Considered options

### Local store
- **(L1) ncruces SQLite** — pure-Go, what's shipped. *Chosen.*
- **(L2) modernc SQLite** — pure-Go alternative; would mean ripping out a tested stack for no gain.
- **(L3) mattn/go-sqlite3** — CGO. Violates #3. Already removed in Wave 7.

### Sync tier
- **(S1) Supabase / PostgreSQL (pgx, pure-Go)** — proven, live today, env-gated, off by default.
- **(S2) Turso embedded replicas + CDC** — the `TARGET_ARCHITECTURE` target; pure-Go HTTP client in go.mod; embedded-replica feature would need the CGO `go-libsql` driver.
- **(S3) No sync** — pure local SQLite. The actual zero-config default.

### Backend framework
- **(B1) PocketBase** — bundles auth + admin UI + collections + realtime + its own SQLite (modernc) into one binary.
- **(B2) Keep our own service/engine/overlay architecture** — what's shipped.

---

## Decision outcome

1. **Local store: ratify pure-Go ncruces SQLite (L1).** It is the canonical,
   offline-first, single-binary, CGO-free floor. No change.

2. **Sync tier is optional and pluggable behind the `pkg/sync` port
   interfaces.** We **accept the *shape* of the vision doc's "Recommendation D"**
   (sync backends sit behind adapter interfaces; the deployer chooses) while
   **rejecting its specific default.** The sanctioned positions:
   - **Zero-config default = ncruces SQLite with no sync (S3).** This is a
     *simpler* zero-config default than PocketBase — nothing to install,
     configure, or run. The single-binary benefit the vision doc attributed to
     PocketBase is **already ours** without it.
   - **Proven sync option (today) = Supabase / PostgreSQL (S1)** via pure-Go pgx,
     env-gated, off by default. Remains supported.
   - **Target sync option = Turso embedded replicas + CDC (S2)**, via the
     **pure-Go `libsql-client-go` HTTP client already in go.mod**. The CGO
     `go-libsql` embedded-replica driver is **rejected**; embedded replicas wait
     for a pure-Go path or stay HTTP-client-based. *(The Turso code is stubbed
     today; wiring it is future work, tracked separately — this ADR ratifies the
     direction and the CGO constraint, not a delivery date.)*

3. **PocketBase (B1): rejected** as a persistence / backend / sync option.

### Why PocketBase is rejected (the grounds that hold)

- **It solves a problem we already solved.** PocketBase's headline benefit —
  single binary, zero-config, CGO-free SQLite — is *already* what ncruces gives
  us. There is no net-new capability that justifies adopting it.
- **It would mean a second SQLite engine in the binary.** PocketBase's pure-Go
  path is `modernc.org/sqlite`, **not** ncruces. Adopting PocketBase means either
  shipping two pure-Go SQLite implementations in one binary (incoherent, bloated)
  or ripping out the shipped, tested ncruces stack to re-platform onto
  modernc-via-PocketBase. Both disturb a green foundation for zero gain.
- **It's a framework, not a library.** PocketBase is an opinionated full backend
  (auth, admin UI, collections, hooks, realtime, its own router & migrations).
  Adopting it re-platforms identity / auth / the authority surface onto
  PocketBase's model — directly ceding the sovereignty we hold today (server-side
  RBAC, our license system, AES-256-GCM field encryption) and colliding with
  invariant #4 and the sovereign-fork thesis.
- **Wrong risk posture for a financial ledger.** The vision doc itself flags
  PocketBase as pre-v1.0, solo-maintainer, with a single-writer SQLite lock. The
  *foundational* persistence layer of a financial substrate is the wrong place to
  take a pre-v1.0, single-maintainer dependency.
- **CGO is *not* the reason.** Stated explicitly so no future reader resurrects a
  dead argument: PocketBase can build CGO-free via modernc. We reject it on the
  four grounds above, not on #3.

---

## Consequences

### Positive
- Stays CGO-free by construction; single-binary, offline-first floor preserved.
- Identity / auth / authority surface stays sovereign and under our control.
- No new pre-v1.0, single-maintainer dependency in the foundation.
- The sync tier is honestly modeled: a proven option (Supabase) ships today; a
  target option (Turso) has a ratified, CGO-safe path.
- The good half of the vision doc's recommendation (pluggable backends behind
  `pkg/sync`) is kept; only the PocketBase default is dropped.

### Negative / costs (stated honestly)
- We **forgo PocketBase's "free" OAuth2 + MFA + admin UI.** Auth/identity remains
  ours to build and own. (Consistent with the sovereignty thesis, but it is a
  real cost — we are choosing control over a convenience freebie.)
- **Turso embedded replicas remain future work.** The code is stubbed; until it's
  wired, the only *live* multi-device sync is Supabase/Postgres. We are ratifying
  a direction, not shipping the feature.
- If a pure-Go embedded-replica path for libsql never materializes, the "local
  speed + auto cloud sync" target stays HTTP-client-based (slightly different
  performance characteristics than true embedded replicas).

### Neutral / follow-ups (NOT decided here)
- **ADR-002 (DuckDNS), ADR-003 (Raspberry Pi product), ADR-004 (CRDT collab)**
  remain open in the vision doc; this ADR does not touch them.
- Wiring `pkg/sync/turso` into the app, and the choice between keeping Supabase
  vs. cutting over to Turso, are **separate future ADRs / work items**.
- Whether to delete or keep the stubbed `pkg/sync/turso` + `pkg/sync/engine`
  packages in the interim is a housekeeping call, not an architecture decision.

---

## Compliance check (invariants)

| Invariant | Status under this decision |
|---|---|
| #1 No secrets in source | Unaffected (sync credentials remain env-loaded) |
| #2 No real client data | Unaffected |
| #3 No CGO | **Upheld** — ncruces stays; CGO `go-libsql` explicitly rejected |
| #4 AI-authority boundary / sovereignty | **Strengthened** — authority surface stays ours, not PocketBase's |
| #5 Financial semantics sacred | **Upheld** — foundational store unchanged; no ledger-affecting change |
| #6 Keep it green | Unaffected — documentation-only decision, no code change |

---

## Citations

- Local store / PRAGMAs: `app.go:483-488`, `go.mod:16-17`
- DB path: `config.go` (`getDatabasePath`, default `data/ph_holdings.db`)
- Live Supabase/Postgres sync: `db_manager.go:103`, `config.go` (`loadDatabaseConfig`, `ENABLE_CLOUD_SYNC`)
- Turso (pure-Go HTTP client, stubbed): `go.mod:22`, `pkg/sync/turso/client.go:79-81`
- No-CGO invariant: `CLAUDE.md` (Invariants #3), `README.md`, `docs/architecture/TARGET_ARCHITECTURE.md`
- Turso as target sync layer: `docs/architecture/TARGET_ARCHITECTURE.md`
- Superseded pending decision: `docs/roadmap/SOVEREIGN_INFRASTRUCTURE_VISION.md` (§ ADR-001)
- PocketBase CGO-free posture (modernc default): PocketBase Go docs, "Extend with Go — Overview"
