# CLAUDE.md — AsymmFlow

Guidance for AI agents (and humans) working in this repository.

**Project**: AsymmFlow — an offline-first, single-binary ERP substrate (Wails + Go + Svelte).
**Demo data is fictional** — see [`SYNTHETIC_IDENTITY.md`](SYNTHETIC_IDENTITY.md). Never
reintroduce real company names, tax IDs, bank details, people, or financial figures.

## Quick start

```bash
go mod download
cd frontend && npm install && cd ..
cp .env.example .env        # all values optional; app runs offline with none
wails dev                   # hot reload
wails build -clean          # production build
```

> `go build ./...` needs `frontend-lab/dist` (it is `go:embed`-ed by `main.go`). Run a
> frontend build first, or use `wails build`.

## Architecture

```
Frontend: Svelte 5 + TypeScript (Onyx & Ether design system in packages/)
Backend:  Go + Wails v2.11 (pure-Go SQLite via ncruces — CGO is banned, keep it banned)
Database: SQLite (primary, offline-first) + optional Supabase sync
AI:       Mistral (Butler) — optional, disabled until a key is supplied
```

### Layer model (the law)
`pure kernel → domain service → storage adapter → ViewModel adapter → agent adapter`

- **Kernel** (`pkg/kernel/{money,approval,evidence,text}`): dependency-free,
  sector-agnostic primitives. Domain vocabulary (PurchaseOrder, Quotation, GRN,
  VATInvoice) may **never** live in the kernel.
- **Engines** (`pkg/…`): reusable capabilities (finance/posting, documents, compliance).
- **Overlay**: company/sector-specific behavior. Company-specific facts (TRN, thresholds,
  branding) are **configuration**, not code.

## Invariants (non-negotiable)

1. **No secrets in source.** No API keys, passwords, tokens, or usable default
   "master" keys. Everything sensitive loads from env / in-app settings.
2. **No real client data.** Use the synthetic canon for any sample/test/demo data.
3. **No CGO.** ncruces SQLite stays.
4. **AI-authority boundary.** Agents may inspect/explain/draft/recommend; only
   deterministic services may approve/post/persist/delete.
5. **Financial semantics are sacred.** Rounding, posting order, and tax behavior
   changes are stop-and-ask, not judgment calls.
6. **Keep it green.** `go build ./...` clean and tests passing at every checkpoint.

## Security posture

bcrypt password hashing; AES-256-GCM field encryption (HKDF + PBKDF2, random salt,
per-message nonce); server-side RBAC (`requirePermission`) on bound endpoints;
parameterized SQL + `isValidSQLIdentifier` for dynamic identifiers; CSRF/XSS guards;
OAuth tokens stored as SHA-256 hashes. When touching auth, crypto, payments, or the
audit trail, preserve these and add tests.

## Conventions

- Currency BHD (3 decimals), VAT configurable (10% default) — domain context, not secret.
- Commit in small coherent steps; never commit secrets or real data.
- Before editing a symbol, understand its callers; keep changes scoped and green.

**Build → Test → Ship. Measure, don't estimate.**
