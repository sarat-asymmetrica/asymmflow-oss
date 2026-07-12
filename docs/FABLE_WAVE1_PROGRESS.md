# FABLE Wave 1 — Progress (Mission A: Overlay Extraction)

**Goal of Mission A:** turn the trading company's hardcoded facts into
**configuration**, so the AsymmFlow core becomes a reusable substrate and the
question *"could a second vertical be composed today?"* gets a credible answer.

**Status: Phase 2 (config extraction) COMPLETE and green.** The domain-package
relocation (workflow code → `overlays/trading/`) is a deliberately-deferred later
wave (see Decisions D9).

---

## What shipped (Phase 2a → 2e)

| Sub-phase | Outcome | Commit |
|---|---|---|
| 2a Foundation | `pkg/overlay` package: `CompanyOverlay`/`DivisionProfile`, `BuiltinDefaults()`, `LoadOverlay()` cascade; `company_branding.go` delegates to an `activeOverlay` singleton; `data/overlay.json` example | `69ded6f` |
| 2b Identity consumers | e-invoice supplier identity dispatched per `invoice.Division` (**fixed a real bug**: Beacon used to emit Acme's TRN); contract PDF + AI-prompt identity routed through the overlay | `54006c2` |
| 2c Business rules + markup | margins, discounts, grade terms, ABB floor, approval/large-order thresholds, product markup → `overlay.BusinessRules` + `ProductMarkupRules`; engines read `overlay.Active()`; **the ABB 15% floor unified across 4 call sites** | `1b2920d` |
| 2d Division defaults | removed 19 GORM `default` tags + ~30 Go/SQL hardcoded `"Acme Instrumentation"` defaults → overlay default at write time; migration DDL config-driven | `adffda7` |
| 2e Manifest + docs | `trading_distribution.manifest.json`, this progress doc, and `FABLE_WAVE1_DECISIONS.md` | (this commit) |

### The overlay surface now (one file changes a deployment's identity + policy)
`data/overlay.json` → company display name / industry / country / currency / VAT;
per-division legal name, VAT/TRN, address, bank details, letterhead, aliases;
business rules (min/ABB/emergency/approval margins, large-order + monthly-cost
thresholds, named competitors, per-grade terms/advance/discount/day-ceiling);
product markup rules + default margin. **No recompile.**

### Repo shape (before → after)

- **Root `package main` files:** sprint start ~226 → after Phase 1 engine de-dup
  ~212 → **now 213** (141 non-test + 72 test). Phase 2 was *config extraction*, not
  file movement, so the root file count is intentionally ~flat; the value delivered
  is **~50 hardcoded company facts removed from code**, not files deleted. (~105K
  non-test LOC in root; the `pkg/` tree is the well-factored part.)
- **Engines:** single source of truth in `pkg/engines` (Phase 1); `pkg/overlay`
  added as the pure-Go config seam (Phase 2).
- **Green gate:** `go build ./...` + `go test ./...` = **49 packages ok, 0
  failures** at every checkpoint.

---

## Could a second vertical be composed today?

**Short answer: a second vertical can be CONFIGURED today; it cannot yet be
COMPOSED as a separate code module today.** Honestly:

### Yes — these are already config (drop in a new `overlay.json`)

- A different **company identity** (name, industry, country, currency, VAT rate).
- A different set of **divisions/sister-companies** (legal names, VAT/TRN,
  addresses, bank details, letterheads, name aliases) — and the **default
  division** that blank records inherit.
- Different **policy numbers**: minimum/ABB/emergency/approval margins, large-order
  and monthly-cost thresholds, named competitors, per-grade payment terms / advance
  / max-discount / day-ceilings, product-type markup and default margin.
- The e-invoice / costing / branding paths already **dispatch on the overlay**, so
  the right TRN, terms, and letterhead come out per division.

A second *trading-shaped* company (different name, divisions, margins, VAT) is a
**config exercise** right now.

### Not yet — these block a genuinely *different* vertical

1. **The domain workflow is still `package main`.** RFQ→costing→offer→order→GRN→
   invoice→payment, serials, delivery notes live in ~141 root files, not an
   `overlays/trading/` package. A *non-trading* vertical (e.g. services, retail)
   would need its own domain package; the generic capabilities it would reuse
   (document numbering, approval routing, PDF assembly) are not yet promoted to
   `pkg/` engines. **This is the next wave (Decisions D9).**
2. **Division-SET enumeration is still partly in code.** Variant-spelling → canonical
   normalization and "is this one of our divisions?" checks are hardcoded in SQL
   `IN`-lists, a drifted `pkg/finance/banking` alias switch, `validDivisions` maps,
   and an Acme-vs-Beacon revenue breakdown. A vertical with three+ divisions, or
   different spellings, needs these generated from `overlay.Divisions` (D5).
3. **Branding/identity strings** (report titles, contract T&C text, the `companyName`
   setting default, seed/template/asset filenames) still say "Acme Instrumentation"
   in places that are display/identity rather than division-default (D8).
4. **One financial inconsistency is parked for a human:** two EUR→BHD rates
   (0.41 vs 0.45) must be reconciled by decision, not by code (D6).
5. **The event bus has subscribers but no production publishers** — compliance hooks
   listen to events nobody emits yet (Mission B / Phase 3).

### Honest "thesis proven" estimate: ~60%

The *configuration* layer of the thesis is proven and load-bearing (identity,
divisions, policy, defaults all move via `overlay.json`, byte-identically, with a
green suite). The *composition* layer (a second vertical as its own overlay
**package** reusing promoted engines) is scaffolded by the manifest but not built.

---

## Recommended next moves (for the Commander)

- **Decide:** is the `overlays/trading/` domain-package move a **Phase 2f** now, or a
  later wave? It is large and higher-risk; the config extraction already proves the
  thesis (D9).
- **Decide:** the EUR→BHD rate (0.41 vs 0.45) — pick one or make it a per-currency
  overlay/setting value (D6). This is a financial call.
- **Then Phase 3 (Mission B):** kernel vocab primitives (`Actor, Party, Request,
  Asset, Workflow, Policy, Timeline`) + wire **event-bus publishers** so the
  compliance subscribers fire.
- **Cheap follow-ups** that further the thesis without risk: generate the division-
  SET enumerations from `overlay.Divisions` (D5); route branding strings through
  `CompanyDisplayName`/`LegalName` (D8).

---

*See `FABLE_WAVE1_DECISIONS.md` for the full rationale behind every seam, and
`docs/modules/trading_distribution.manifest.json` for the module contract.*
