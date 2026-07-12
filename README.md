# AsymmFlow

An offline-first, single-binary **ERP substrate** for building vertical business
apps (trading, distribution, instrumentation, and beyond) with Go + Svelte + Wails.
Your data lives in a local SQLite file you own; an optional cloud sync is yours to
configure. No rent-seeking cloud, no vendor lock-in — own your fork.

> **Demo data is fictional.** This repository ships with a synthetic reference
> dataset (see [`SYNTHETIC_IDENTITY.md`](SYNTHETIC_IDENTITY.md)). Company names,
> tax IDs, bank details, people, and financial figures are invented for
> demonstration and do not represent any real organization.

## Tech stack

- **Backend**: Go 1.25+ / Wails v2.11 (pure-Go SQLite via ncruces — no CGO)
- **Frontend**: Svelte 5 + TypeScript + Vite (Onyx & Ether design system in `packages/`)
- **Database**: SQLite (primary, offline-first) + optional Supabase sync
- **AI** (optional): Mistral API for the "Butler" assistant — disabled until you supply a key
- **Platforms**: Windows (primary), macOS (Apple Silicon)

## Quick start

### Prerequisites
- Go 1.25+
- Node.js 20+
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Development
```bash
# 1. Dependencies
go mod download
cd frontend && npm install && cd ..

# 2. Config (all values optional — app runs fully offline with none)
cp .env.example .env
# Edit .env to enable cloud sync (Supabase) or the Butler AI assistant.

# 3. Run with hot reload
wails dev

# 4. Production build
wails build -clean
```

> `go build ./...` requires `frontend/dist` to exist (it is `go:embed`-ed by
> `main.go`). Run a frontend build first (`cd frontend && npm run build`) or use
> `wails build`, which builds the frontend for you.

## Configuration

All configuration is via environment variables (see `.env.example`); there are
**no hardcoded credentials**. Notable flags:

| Variable | Purpose | Default |
|---|---|---|
| `MISTRAL_API_KEY` | Enables the Butler AI assistant | unset (AI disabled) |
| `ENABLE_CLOUD_SYNC` | Turns on Supabase background sync | `false` |
| `SUPABASE_DB_*` | Your own Supabase project pooler connection | unset |
| `ENABLE_DEVELOPER_MASTER_KEY` + `ASYMMFLOW_MASTER_KEY` | Optional dev override key | disabled / empty |

## Features

- **Sales pipeline**: RFQ → Costing → Offer → Order → Invoice → Payment
- **CRM**: customer/supplier dashboards, 360° profiles, notes, issue tracking
- **Finance**: dashboard, AR aging, payments, bank reconciliation, VAT/e-invoicing (UBL 2.1)
- **Operations**: purchase orders, delivery notes, GRN, serial traceability
- **Butler AI** (optional): natural-language queries, PDF reports on letterhead
- **RBAC**: license-based activation with roles (Admin / Manager / Sales / Operations / Staff)
- **Offline-first**: SQLite primary with optional background cloud sync

## Architecture — the overlay model

AsymmFlow is a **substrate**, not a finished vertical. Behavior is layered so you can
build your own vertical on top without forking the core:

```
pure kernel → domain service → storage adapter → ViewModel adapter → agent adapter
```

- **Kernel** (`pkg/kernel/{money,approval,evidence,text}`) — dependency-free,
  sector-agnostic primitives. No domain vocabulary (no `PurchaseOrder`, `VATInvoice`)
  ever lives here.
- **Engines** (`pkg/…`) — reusable capabilities (finance/posting, documents, compliance).
- **Overlay** — your company/sector-specific behavior. Company-specific facts (TRN,
  thresholds, branding) are **configuration**, not code — so an overlay customizes the
  substrate without editing it.

The shipped demo is itself an overlay over synthetic data (see
[`SYNTHETIC_IDENTITY.md`](SYNTHETIC_IDENTITY.md)); swap the config and data for your own.

## Project structure

```
asymmflow/
  main.go                 # Entry point (go:embed frontend/dist)
  app.go                  # App lifecycle + Wails-bound API surface
  *_service.go            # Domain services (invoice, payment, bank, …)
  pkg/                    # Engines + kernel primitives (money, approval, evidence, text)
  packages/               # Onyx & Ether Svelte design system
  frontend/               # Svelte 5 + TS UI
  cmd/                    # CLI utilities (import, sync, migrate)
  docs/                   # Architecture & methodology docs
  SYNTHETIC_IDENTITY.md   # The fictional demo-data canon
```

## Testing

```bash
go test ./...                 # Go tests
cd frontend && npm run check  # TypeScript check
```

## Security

This codebase has been through multiple red-team passes. Highlights: bcrypt password
hashing, AES-256-GCM field encryption (HKDF + PBKDF2), server-side RBAC on bound
endpoints, parameterized SQL, and CSRF/XSS protections. See `docs/` for audit
records. Found something? Please open a security advisory rather than a public issue.

## License

AsymmFlow is licensed under the **[GNU AGPL-3.0](LICENSE)** — and **every release
automatically becomes [MIT](LICENSE-ROADMAP.md) two years after it ships.**

There is always a free MIT version of AsymmFlow trailing two years behind the
frontier. The newest work stays AGPL (share-alike, including over a network) for two
years, then joins the commons under MIT — automatically, on a fixed clock, forever.
We only ever loosen; we never tighten a grant already made. See
**[`LICENSE-ROADMAP.md`](LICENSE-ROADMAP.md)** for the full commitment and mechanism.

Contributions are welcome under a lightweight CLA that keeps the two-year MIT promise
real for everyone — see **[`CLA.md`](CLA.md)**. (TL;DR: you keep your copyright;
`git commit -s` to sign off.)
