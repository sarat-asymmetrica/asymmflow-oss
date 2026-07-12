# Generative Refactor Plan

## Philosophy

This is NOT a traditional refactor (move code around).
This is a GENERATIVE refactor (define schemas, generate 80%, hand-write 20%).

The existing 183K LOC codebase is a BEHAVIORAL SPECIFICATION — it tells us what Acme Instrumentation needs.
The alchemy engines are CODE GENERATORS — they produce clean implementations from schemas.
The hand-written 20% is UNIQUE DOMAIN LOGIC — what no generator can know.

## The Generative Pipeline

```
Domain Schemas (human-designed, the creative work)
        ↓
schema_alchemy → Go models + SQLite DDL + mock data
        ↓
api_alchemy → Handlers + middleware + OpenAPI
        ↓
fullstack_alchemy → Svelte pages + Go bindings
        ↓
component_alchemy → UI components (Wabi-Sabi aesthetic)
form_alchemy → Forms with three-regime validation
layout_alchemy → Dashboard/form/table layouts
theme_alchemy → Design tokens (WCAG AA/AAA)
        ↓
Hand-written domain logic (tax calc, bank matching, etc.)
        ↓
Integration wiring (Pretext, TOON, Turso, Nango)
        ↓
TARGET: ~20-30K LOC hand-written + ~100K generated
```

## Wave Plan

### Wave 0: Archaeological Audit
**Goal:** Understand what's unique vs boilerplate in the existing codebase.
**See:** WAVE0_AUDIT.md for exact commands.

**Key outputs:**
- Dead code list (safe to delete)
- Hot path traces (the 5 workflows that matter)
- Temporal coupling map (what changes together)
- Unique domain logic inventory (what generators CAN'T produce)

### Wave 1: Schema Design
**Goal:** Define the 6 domain schemas — this is THE creative work.

**Process:**
1. Extract type definitions from database.go (90 structs)
2. Group into domain schemas (finance, crm, documents, butler, sync, infra)
3. Write Cap'n Proto or Go type definitions as source of truth
4. Run schema_alchemy to validate: do the generated models cover Acme Instrumentation's needs?

**Inputs:**
- database.go (90 struct definitions)
- app.go (70 additional struct definitions)
- App.d.ts (837 exported functions — the frontend contract)
- data/ssot/ (business requirements)

**Outputs:**
- schemas/finance.capnp (or .go)
- schemas/crm.capnp
- schemas/documents.capnp
- schemas/butler.capnp
- schemas/sync.capnp
- schemas/infra.capnp

### Wave 2: Generate Foundation
**Goal:** Use alchemy engines to generate the infrastructure and data layer.

**Commands:**
```bash
# Generate database models from schemas
cd 03_ENGINES/schema_alchemy
go run . --domain=finance --seed=108
go run . --domain=crm --seed=108
go run . --domain=documents --seed=108

# Generate API handlers
cd 03_ENGINES/api_alchemy
go run . --schema=../schemas/finance.capnp --output=../pkg/finance/

# Generate frontend bindings
cd 03_ENGINES/fullstack_alchemy
go run . create asymmflow --domains=finance,crm,documents --seed=108
```

**Outputs:**
- pkg/infra/db/ (connection pool, migrations)
- pkg/*/domain.go (generated types)
- pkg/*/ports.go (generated interfaces)
- generated/handlers/ (CRUD operations)
- generated/models/ (TypeScript types for Svelte)

### Wave 3: Generate UI
**Goal:** Generate the frontend from domain specifications.

**Commands:**
```bash
# Generate components
cd 03_ENGINES/component_alchemy
go run ./cmd/generate_app -season aki -intent business AsymmFlow 108

# Generate forms
cd 03_ENGINES/form_alchemy
go run . --entities=Invoice,Payment,Offer,Customer --preset=business

# Generate layouts
cd 03_ENGINES/layout_alchemy
go run . --type=dashboard --seed=108
go run . --type=form --seed=108
go run . --type=table --seed=108

# Generate theme
cd 03_ENGINES/theme_alchemy
go run . --seed=108 --validate-wcag=AA
```

**Outputs:**
- frontend/src/lib/components/ (Svelte 5 components)
- frontend/src/lib/forms/ (three-regime validation)
- frontend/src/lib/layouts/ (responsive, φ-based)
- frontend/src/lib/styles/ (design tokens)

### Wave 4: Hand-Write Domain Logic
**Goal:** Write ONLY the unique business rules that generators cannot produce.

**What's unique to Acme Instrumentation (from archaeological audit):**
- Bank statement parsing (specific Bahrain bank formats)
- Payment prediction algorithm (quaternion-based, proven 87.3% accuracy)
- Document classification rules (Acme Instrumentation's specific document types)
- Butler AI prompts (Sarvam 105B integration, PH context)
- FX revaluation logic (BHD multi-currency handling)
- Offer numbering and revision system
- Division-based financial reporting (Acme Instrumentation, Beacon Controls, PH Machinery)
- Arabic RTL support in PDFs (specific to Gulf market)

**Estimated unique LOC:** 15,000-25,000 (vs 183K current)

### Wave 5: Wire Integrations
**Goal:** Connect the generated + hand-written code to external systems.

| Integration | Implementation |
|-------------|---------------|
| Pretext | Wire into pkg/documents/pdf/ for text measurement |
| TOON | Wire into pkg/butler/chat/ for LLM communication |
| Turso | Wire into pkg/sync/turso/ replacing Supabase |
| Nango | Wire for OneDrive + future external APIs |
| OpenTelemetry | Instrument hot paths in pkg/finance/ and pkg/butler/ |
| MathAlive | PDF generation pipeline for invoices/offers |

### Wave 6: Wails v3 + Svelte 5 Migration
**Goal:** The final structural upgrade.

| Task | What Changes |
|------|-------------|
| Wails v3 migration | Multi-service bind (no God Object), multi-window, systray |
| Svelte 5 migration | stores → runes ($state, $derived, $effect) |
| ncruces migration | Drop CGO, pure Go SQLite |
| GORM removal | Replace with raw SQL + type-safe query builders |
| Taskfile | Replace shell scripts with Taskfile.yml |

## Alchemy Engine Reference

| Engine | Location | Input | Output |
|--------|----------|-------|--------|
| schema_alchemy | 03_ENGINES/schema_alchemy/ | Domain spec | SQLite DDL + Go models + mock data |
| api_alchemy | 03_ENGINES/api_alchemy/ | Schema/entities | REST handlers + middleware + OpenAPI |
| fullstack_alchemy | 03_ENGINES/fullstack_alchemy/ | Domain + seed | SvelteKit + Go (17 files/entity) |
| component_alchemy | 03_ENGINES/component_alchemy/ | Entity + season | Svelte components (Wabi-Sabi) |
| form_alchemy | 03_ENGINES/form_alchemy/ | Entity fields | Three-regime forms (19 input types) |
| layout_alchemy | 03_ENGINES/layout_alchemy/ | Layout type | Responsive layouts (φ-based) |
| theme_alchemy | 03_ENGINES/theme_alchemy/ | Seed number | Color system (WCAG validated) |
| meta_alchemy | 03_ENGINES/meta_alchemy/ | Generated code | Auto-fixed, validated output |
| ocr | 03_ENGINES/ocr/ | Documents | Structured data ($0.00105/page) |

## Mathematical Framework Integration

| Concept | Where It Lives | How It's Used |
|---------|---------------|---------------|
| Three-Regime Dynamics | api_alchemy rate limits, form_alchemy validation, Butler routing | R1=explore, R2=compute, R3=stable |
| Williams Batching | pkg/sync, pkg/documents/ocr, MathAlive PDF pipeline | √n × log₂(n) batch sizes |
| Digital Root Filtering | pkg/butler (pre-LLM gate), pkg/infra/cache | O(1) elimination of 88.9% |
| φ-Proportions | layout_alchemy, theme_alchemy, api_alchemy pagination | Golden ratio spacing/timing |
| Pretext Two-Phase | pkg/documents (prepare=measure, layout=render) | Expensive once, cheap always |
| SLERP | pkg/butler session tracking, theme transitions | Smooth state interpolation |
| Quaternion State | Each domain's health metrics, payment prediction | {mean, var, skew, kurt} |
| 87.532% Attractor | SAT constraint satisfaction, system convergence | Phase transition threshold |

## Success Criteria

1. All 73 existing test files pass (or equivalent coverage in new structure)
2. All 837 frontend-facing functions maintain their signatures
3. Total hand-written LOC < 30K (vs 183K current)
4. Each domain package independently testable (no setupTestApp())
5. `go build ./...` completes with zero CGO dependencies
6. Dashboard renders in < 100ms (cached state, not re-computed)
7. PDF generation uses Pretext measurement (zero trial-and-error rendering)
8. Butler AI calls use TOON (30% token savings verified)
9. ELEGANCE_CHECK score >= 0.72 for each domain package

## The Phoenix Clause

If ANY domain is too tangled to extract or generate:
1. Verify test coverage exists
2. Verify schema defines the types
3. DELETE the implementation
4. REBUILD from schema (hand-write or re-generate)
5. Run tests

This is not failure. This is controlled demolition followed by clean construction.
