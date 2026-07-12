# Sovereign Infrastructure Vision

**Date**: June 7, 2026
**Status**: RESEARCH COMPLETE — Architecture decisions pending
**Context**: Acme Instrumentation deployment proved $0/month self-hosted PostgreSQL works. This document captures the broader vision for sovereign infrastructure across AsymmFlow as a general-purpose SMB platform.
**Catalyst**: Losing the Hetzner VPS forced us to discover that $0 infrastructure isn't just possible — it's the competitive advantage.

---

## The Thesis

> Software you can truly own means infrastructure you don't rent.

Cloud-first SaaS charges SMBs $93-368/month for database + storage + auth + monitoring + VPN. In emerging markets (India: 64M MSMEs, Africa: 244M small businesses), the price ceiling is $5-10/month. Cloud hosting costs alone exceed the entire software budget.

**The only way to be margin-positive at SMB prices is to eliminate infrastructure costs entirely.**

AsymmFlow's offline-first SQLite architecture accidentally solved this. Today we proved it works. This document turns an accident into a strategy.

---

## What We Proved (June 7, 2026)

```
Acme Instrumentation deployment — Bahrain receptionist PC:
  PostgreSQL 16 on Windows         → $0/month (vs $25/mo Hetzner VPS)
  DuckDNS for dynamic DNS          → $0/month (vs static IP costs)
  DMZ + Windows Firewall           → $0/month (vs VPN services)
  45-minute phone setup            → $0 DevOps (vs $50-150/hr consulting)
  264ms India-to-Bahrain latency   → Production-viable
  SQLite offline-first             → Zero downtime during migration

  Deployment record: ph_holdings/docs/DEPLOYMENT_RECORD_RECEPTIONIST_POSTGRES_2026_06_07.md
```

---

## The Five-Layer Sovereign Stack

Each layer has a cloud option (costs money) and one or more sovereign options ($0):

### Layer 1: Data (where structured data lives)

| Option | Type | Maturity | Best For |
|--------|------|----------|----------|
| **PostgreSQL on customer hardware** | Server DB | Battle-tested, 20+ years | Multi-device sync hub (PROVEN today) |
| **PocketBase (embedded in Go)** | Embedded backend | 54K stars, pre-v1.0 | Single-binary distribution, small deployments |
| **Turso/libSQL embedded replicas** | Distributed SQLite | Production-ready | Read-heavy workloads with edge distribution |
| **SQLite + Litestream** | SQLite + replication | Production-proven (Rails 8) | Backup/disaster recovery |

**Architecture decision (pending):** The TARGET_ARCHITECTURE specifies Turso embedded replicas. PocketBase is a viable alternative that bundles auth + files + realtime into one binary. Both are compatible with the current hexagonal architecture. Key trade-off:

```
Turso:      SQLite wire-compatible, distributed, but requires Turso cloud or self-hosted sqld
PocketBase: Single binary, embeds in Go (!!!), but SQLite single-writer, solo maintainer
PostgreSQL: Proven today, but requires separate install + config
```

**Recommendation:** Support ALL THREE as adapter implementations behind `pkg/sync` interfaces. The hexagonal architecture already enables this. Customer chooses based on their context:
- PocketBase → solo operator, one location, wants simplest possible setup
- PostgreSQL → multi-branch, existing DBA, proven reliability
- Turso → tech-forward customer who wants edge replication

### Layer 2: Sync (how devices share data)

| Pattern | Use For | Never Use For |
|---------|---------|---------------|
| **Server-authoritative** (current: SQLite→PostgreSQL) | Financial transactions, inventory, ledgers | — |
| **CRDTs** (cr-sqlite, Automerge, Yjs) | Collaborative notes, comments, task assignments | Ledger data, account balances, inventory counts |
| **Litestream** (WAL streaming) | Backup, disaster recovery | Multi-writer scenarios |

**Critical finding from CRDT research:** CRDTs guarantee eventual consistency. Two offline users could both "sell" the last item in stock. For ERP financial data, server-authoritative sync is CORRECT. CRDTs are mathematically wrong for ledgers.

**Recommendation:** Hybrid approach:
- `pkg/sync` — server-authoritative for all transactional data (invoices, payments, inventory)
- `pkg/collab` (future) — CRDTs for collaborative features (notes, comments, shared docs)

### Layer 3: Networking (how devices find each other)

| Option | Cost | Complexity | Best For |
|--------|------|------------|----------|
| **DuckDNS + port forward** | $0 | Low (PROVEN) | Simple single-office setups |
| **DuckDNS + DMZ** | $0 | Low (PROVEN today) | When router SPI blocks port forwards |
| **Cloudflare Tunnel** | $0 | Medium | Zero-config, no port forwarding, but no UDP, 100MB upload cap |
| **WireGuard on Raspberry Pi** | $50 one-time | Medium | Multi-branch, VPN mesh, full sovereignty |
| **Tailscale free tier** | $0 (3 users) | Lowest | Quick setup for tiny teams |

**Recommendation:** Ship a `pkg/infra/discovery` module that auto-detects and configures networking:
1. LAN detection (same network → use local IP)
2. DuckDNS updater (built into the app, no separate scheduled task)
3. Cloudflare Tunnel fallback (for restrictive networks)

### Layer 4: Storage (where files and blobs live)

| Option | Cost | What It Replaces |
|--------|------|-----------------|
| **Local filesystem + OneDrive/GDrive sync** | $0 | S3, Supabase Storage |
| **MinIO** | $0 (self-hosted) | S3-compatible API on local hardware |
| **PocketBase file storage** | $0 (embedded) | Bundled with the data layer |

**Recommendation:** PocketBase's built-in file storage is the simplest path. For customers needing S3 compatibility (e.g., existing integrations), MinIO on the same machine.

### Layer 5: Identity (who are you)

| Option | Cost | What It Replaces |
|--------|------|-----------------|
| **AsymmFlow license keys + device binding** | $0 | Auth0, Clerk ($23-100/mo) |
| **PocketBase auth** | $0 | Built-in email/password + OAuth2 (32 providers) + OTP + MFA |
| **Sovereign Auth (@asymm/auth)** | $0 | Our own package, already exists |

**Recommendation:** Current license key system works for Acme Instrumentation model. For the general-purpose platform, PocketBase auth gives us OAuth2 + MFA for free — embedded in the same binary.

---

## PocketBase Deep Dive: The One-Binary Vision

### What It Is
Single Go binary (~40MB) bundling: SQLite database, REST API, realtime subscriptions (SSE), auth (email + OAuth2 + OTP + MFA), file storage (local or S3), admin dashboard, and migrations system. Zero external dependencies.

### Why It Matters for AsymmFlow

PocketBase is designed to be **embedded as a Go library**:

```go
import "github.com/pocketbase/pocketbase"

func main() {
    // PocketBase starts inside your app
    app := pocketbase.New()

    // Add custom routes (your business logic)
    app.OnServe().BindFunc(func(se *core.ServeEvent) error {
        se.Router.GET("/api/cashflow", handleCashflow)
        return se.Next()
    })

    app.Start()
}
```

This means AsymmFlow could compile to ONE .exe that contains:
- Wails desktop app (frontend + backend)
- PocketBase (database + auth + files + realtime + admin)
- All business logic
- Everything a customer needs

**No PostgreSQL install. No pgAdmin. No psql. No config files. Double-click, done.**

### Production Capabilities (verified by research)

| Metric | Capability | Our Need |
|--------|-----------|----------|
| Concurrent connections | 10,000+ | 2-50 users |
| Write throughput | 50,000/min | ~100/min for typical ERP |
| Requests/month | 3.2M+ proven | ~500K for 50-user office |
| Memory | ~500MB under load | Receptionist PC has 4-16GB |
| Admin dashboard | Built-in Svelte UI | Free admin panel |

### Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Solo maintainer (Gani Georgiev) | MEDIUM | MIT license → forkable; 54K stars = community will maintain |
| Pre-v1.0, breaking changes | LOW | Pin versions, test on upgrade |
| SQLite single-writer lock | LOW for ERP | ERP write loads are modest; reads are unlimited |

### Integration Path with Current Architecture

```
Current (ph_holdings):
  app.go (God Object) → SQLite (mattn/go-sqlite3) → PostgreSQL (GORM sync)

Refactor target (asymmflow):
  cmd/asymmflow/main.go → pkg/*/services → SQLite (ncruces) → Turso replicas

PocketBase integration option:
  cmd/asymmflow/main.go → pkg/*/services → PocketBase (embedded) → built-in everything
                                         ↓
                          PocketBase provides: DB + Auth + Files + API + Admin
                          We provide: Business logic in pkg/kernel, pkg/finance, pkg/crm
```

The hexagonal architecture in asymmflow already separates domain logic from infrastructure. PocketBase slots in as an adapter behind the existing port interfaces. The `pkg/kernel/` primitives (money, approval, evidence, text) are infrastructure-agnostic — they work with any backend.

---

## Market Opportunity

### The Numbers

| Region | Addressable SMBs | Digital Maturity | Price Ceiling |
|--------|-----------------|------------------|---------------|
| India | 64M MSMEs, 4.5M with PCs | 12% digitally mature | $5-10/mo or $60-250 one-time |
| Africa | 244M small businesses | 40% use digital tools | Even lower than India |
| Middle East | $18.9B SaaS market (13.4% CAGR) | Growing fast | $10-25/mo for premium |
| Southeast Asia | 71M MSMEs | Varies widely | Similar to India |

### Why $0 Infrastructure Is a Business Requirement

At $5-10/month pricing, cloud hosting costs eat the ENTIRE margin:

```
Revenue per customer:     $5-10/month
Cloud database:          -$7-25/month (Supabase/RDS)
Cloud storage:           -$2-5/month
Cloud auth:              -$2-10/month
─────────────────────────────────────
Net margin:              NEGATIVE

With sovereign infrastructure:
Revenue per customer:     $5-10/month
Infrastructure cost:     -$0/month
─────────────────────────────────────
Net margin:              $5-10/month (100%!)
```

### Competitive Landscape

| Competitor | Offline | Self-hosted | Price | Our Advantage |
|-----------|---------|-------------|-------|---------------|
| **Tally Prime** | Full | Desktop-only | $215 one-time | We have sync, mobile, AI, multi-device |
| **Busy Accounting** | Full | Desktop-only | $60/year | Same as Tally |
| **Odoo Community** | No | Self-host avail | $0 + hosting | We have offline-first, zero-config |
| **ERPNext** | No | Self-host avail | $0 + hosting | We have offline-first, zero-config |
| **Zoho** | Limited | No | Free tier | We have full offline, data sovereignty |

### Distribution Strategy (from research)

1. **Chartered Accountants / CAs** — Tally built 28,000 partners this way. The CA recommends the software.
2. **WhatsApp** — 487M Indian users. Product demos via WhatsApp video. Support via WhatsApp.
3. **Word of mouth** — 63% of SMBs rely on it. The "my business didn't stop during a power cut" story sells itself.
4. **Module-by-module adoption** — Indian SMBs buy incrementally. Freemium billing → paid inventory → paid payroll.

### Data Sovereignty Tailwinds

All regulatory trends favor local-first:
- **India DPDP Act**: Full compliance by May 2027. Sensitive data must stay in India.
- **Saudi PDPL**: Enforceable since Sep 2024. Local storage requirements.
- **Nigeria NDPA**: Localization obligations since 2024.

Local-first = naturally compliant. No legal review needed. No GDPR DPA. The data is already on their machine.

---

## The Tally Comparison (Strategic Positioning)

Tally Prime dominates India (2M+ daily users, 70-90% accounting market share) with the SAME philosophy we independently arrived at: desktop-first, offline-first, one-time license.

**What Tally proves**: The model works. Desktop + offline + one-time license = 2M daily users.

**What Tally lacks** (our opportunity):

| Feature | Tally | AsymmFlow |
|---------|-------|-----------|
| Multi-device sync | No | Yes (SQLite + PostgreSQL/PocketBase) |
| Mobile access | No | Yes (Wails v3 + PWA roadmap) |
| AI-assisted workflows | No | Yes (Butler, Sarvam, OCR) |
| Open architecture | Closed-source | Inspectable, forkable |
| Cloud optional | Desktop-only | Sync when you want, offline when you don't |
| International | India-focused | Multi-currency, multi-language (17 langs) |
| Self-hosted infra | N/A (no server) | $0 on customer hardware |
| Modern UI | Windows Forms era | Svelte 5, responsive |

**We're not competing with Tally. We're building what Tally should have been if it was born in 2026.**

---

## Architecture Decision Records (Pending)

### ADR-001: PocketBase vs Turso vs PostgreSQL

**Status**: ✅ RESOLVED (2026-06-15) — ratified in
[`docs/architecture/adr/ADR-001-persistence-and-sync-stack.md`](../architecture/adr/ADR-001-persistence-and-sync-stack.md).

**Outcome (summary):** Local store stays **pure-Go ncruces SQLite** (offline-first,
single-binary, CGO-free). Sync is **optional and pluggable behind `pkg/sync`**: the
zero-config default is *SQLite with no sync*; **Supabase/Postgres** is the proven
option today; **Turso embedded replicas** (pure-Go HTTP client only — the CGO
`go-libsql` driver is rejected) is the target. **PocketBase is rejected** — not on
CGO grounds (it can build CGO-free via modernc), but because it adds no capability
ncruces doesn't already give us, would put a second SQLite engine in the binary,
re-platforms our sovereign auth/authority surface onto a framework, and is the wrong
risk posture (pre-v1.0, solo-maintainer) for a financial ledger's foundation. The
*shape* of the recommendation below (pluggable backends) is kept; the *PocketBase
default* is not. See the ADR for full rationale and citations.

<details><summary>Original pending options (superseded — kept for provenance)</summary>

**Options**:
- A) PocketBase as default, PostgreSQL as migration path for power users
- B) Turso as default (per TARGET_ARCHITECTURE), PocketBase as lightweight alternative
- C) PostgreSQL as default (proven today), PocketBase/Turso as future options
- D) All three behind adapter interfaces, customer chooses at install time

**Recommendation**: D — the hexagonal architecture supports this. Ship with PocketBase as the zero-config default (simplest setup), PostgreSQL as the proven option (for customers like Acme Instrumentation who already have it), and Turso when edge replication is needed.

</details>

### ADR-002: Built-in DuckDNS Client

**Status**: PROPOSED

Should AsymmFlow include a built-in DuckDNS/DDNS client so the app itself updates DNS, eliminating the need for a separate Scheduled Task?

**Recommendation**: Yes. A goroutine that pings DuckDNS every 5 minutes. Config in `.env`: `DDNS_PROVIDER=duckdns`, `DDNS_TOKEN=xxx`, `DDNS_DOMAIN=ph-trading`.

### ADR-003: Raspberry Pi Sovereign Server Product

**Status**: EXPLORATORY

A pre-configured Raspberry Pi 5 ($50) that customers buy as a one-time add-on:
- Pre-installed: PostgreSQL/PocketBase + WireGuard + MinIO + Uptime Kuma + Litestream
- 3W idle power (~$8/year electricity)
- UPS HAT for 12-hour battery backup ($20-40)
- Eliminates: DuckDNS, port forwarding, DMZ, Windows Firewall config

**Trade-off**: Higher one-time cost ($70-90 total) but zero configuration and true always-on.

### ADR-004: CRDT Collaborative Layer

**Status**: FUTURE

Add CRDTs (via cr-sqlite or Yjs) for collaborative features only:
- Shared notes on invoices/orders
- Real-time commenting
- Task assignment and status updates

NEVER for: financial data, inventory, ledgers (server-authoritative only).

---

## Evolution Path

### Phase: NOW (AsymmFlow Acme Instrumentation 2026 Edition)
```
SQLite (local) + PostgreSQL (receptionist PC) + DuckDNS
  → Proven, deployed, working
  → Address audit findings (report_storage.go Supabase hardcoding, connect_timeout)
  → Build fresh .exe, deploy to all machines
```

### Phase: NEXT (AsymmFlow General Purpose)
```
PocketBase embedded in Wails binary = ONE .exe ships everything
  → Zero-config install for new customers
  → Built-in auth replaces license key system
  → Built-in file storage replaces Supabase Storage
  → Built-in admin dashboard for IT admin at customer site
  → Litestream for automatic SQLite backup
  → Built-in DDNS client (no separate Scheduled Task)
```

### Phase: SCALE (AsymmFlow Platform)
```
cr-sqlite for peer-to-peer collaborative features
  → Notes, comments, task assignments sync without central server
Raspberry Pi "Sovereign Server" as $50 product add-on
  → Pre-configured, plug-and-play, no DuckDNS/DMZ/firewall needed
WireGuard mesh for multi-branch businesses
  → Secure tunnels between offices, no port forwarding
Cloudflare Tunnel as zero-config networking option
  → For customers behind restrictive ISPs/firewalls
```

---

## Appendix: Research Sources (June 7, 2026)

### PocketBase
- 54K GitHub stars, v0.39.x, solo maintainer (Gani Georgiev)
- 3.2M requests/month proven in production
- 50K write operations/minute on $4 VPS
- Go embedding is a first-class feature
- MIT license
- Sources: pocketbase.io, GitHub discussions, Better Stack guide, ByteSizeGo

### CRDTs & Local-First
- cr-sqlite: v0.16.3, SQLite extension for CRDTs, beta maturity
- Electric SQL: Postgres-to-SQLite sync, production-capable
- Litestream: WAL streaming, production-proven, Rails 8 integration
- Yjs: 920K weekly downloads, used by Notion and Jupyter
- Local-First Conf 2026: July 12-14, Berlin
- CRDT limitation: fundamentally wrong for ledger data (eventual consistency)
- Sources: vlcn.io, localfirstnews.com, Ink & Switch, Cinapse case study

### Sovereign Infrastructure
- MinIO: S3-compatible, single binary, AGPLv3
- Uptime Kuma: 65K stars, gold standard self-hosted monitoring
- WireGuard: kernel-level VPN, runs on Windows + Pi
- Cloudflare Tunnel: free, no port forwarding, but no UDP / 100MB cap
- Raspberry Pi 5: 3W idle, ~$8/year electricity, 7.6M units shipped FY2025
- Cloud savings: $255-980/month → $6-24/month self-hosted
- Sources: r/selfhosted, awesome-selfhosted, MassiveGRID, DemandSage

### Market
- India: 64M MSMEs, 4.5M with PCs, 12% digitally mature
- Africa: 244M small businesses, 40% digitized, $331B credit shortfall
- Tally Prime: 2M daily users, 28,000 dealer partners, $215 one-time
- DPDP Act (India): compliance by May 2027
- Distribution: CAs (primary), WhatsApp (487M users), word of mouth (63%)
- Sources: Ken Research, IDC/AMI, Grand View Research, SaaSBoomi

---

## The Mathematical Guarantee

```
S3 manifold:  States always valid        (||Q|| = 1.0)
CRDT:         Replicas always converge    (commutativity)
SQLite-first: App always works            (offline = valid state)
```

Three expressions of the same invariant: there is no invalid state.

We're not selling a database. We're not selling an ERP.
We're selling a mathematical guarantee that their business never stops.

---

*Om Lokah Samastah Sukhino Bhavantu*
*May all beings benefit from sovereign software.*
