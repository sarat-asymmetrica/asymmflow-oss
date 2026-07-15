# The Frontend Kernel — Constitution

**What this is:** a ground-up rebuild of the AsymmFlow frontend as a *kernel* —
a small set of hardened engines that render screens from declared data — replacing
60 hand-written screens (~59k LOC, `formatDate` defined 20 times) with roughly
5 archetype engines + ~40 typed descriptors + a handful of genuinely bespoke screens.

**Why:** the old frontend grew by copy-instantiation. Agents (and rushed humans)
duplicate unless the architecture forbids it. This architecture forbids it.

It is the frontend mirror of the backend's overlay philosophy:

| Backend (shipped)                          | Frontend kernel (this)                     |
|--------------------------------------------|--------------------------------------------|
| `pkg/overlay` division registry            | Descriptor registry                         |
| "branding is configuration, not code"      | "screens are configuration, not code"       |
| `NormalizeDivisionName` — one path          | Archetype engines — one path                |
| `division_literal_audit_test.go` tripwire  | Layout-CSS + duplication tripwires          |
| Byte-identical refactor law                 | Visual/functional parity law vs old screens |

## The five pillars

1. **Backend is the schema authority.** Entities, fields, types, relations live in
   Go structs + overlay. The frontend never re-declares what the backend knows.
   (Base descriptors will eventually be emitted via a `GetUISchema()` binding;
   until then, hand-authored base descriptors mirror the bindings layer.)
2. **Typed TS descriptors, compiled in.** A screen is a `*.descriptor.ts` — entity,
   columns, filters, per-status actions. Full IDE checking, diffable, reviewable.
   No runtime JSON interpreter until a real need appears.
3. **Archetype engines render descriptors.** `DocumentLedger`, `EntityMaster`,
   `DetailView`, `Hub`, `Wizard`. Each written ONCE, hardened, viewmodel in
   `.svelte.ts` (unit-testable without a browser).
4. **Layout primitives own all layout.** `PageShell`, `Stack`, `Row`, `Grid`,
   `Card`, `Toolbar`, `FormGrid`, `DataTable`. Overflow-proofing (min-width:0,
   designated scroll regions) is solved inside primitives, once. Screens and
   archetypes compose primitives; they do not write layout CSS.
5. **Layout truth is computed, not observed.** Descriptors declare widths and
   content classes, so overflow is verifiable as arithmetic (pretext) at dev time,
   plus Playwright screenshots as the outer moat.

## Laws

- **L1 — No raw layout CSS in screens.** No `display:`, `margin:`, `float:`,
  raw `px` spacing, or hex colors in screen/descriptor-level code. Only kernel
  primitives may contain layout CSS. (Tripwire test enforces this.)
- **L2 — One definition per utility.** Formatting (dates, money, TRNs), search,
  sorting, status badges: defined once in `$kernel`, imported everywhere.
- **L3 — Tokens only through the semantic layer.** The lab consumes
  `../frontend/src/assets/design-tokens.css` (one-source law, unchanged).
  Kernel-owned additions are namespaced `--k-*` and never redefine existing tokens.
- **L4 — Graceful ejection.** Every archetype accepts slot overrides at any
  granularity (cell → column → panel → whole screen bespoke). A screen must
  never fight the engine; it steps outside it, on the primitives, explicitly.
- **L5 — Viewmodels are `.svelte.ts`, views are thin.** All logic (filtering,
  derivation, bridge calls) lives in rune-based viewmodel modules, unit-tested
  in vitest. `.svelte` files bind and render; they do not compute.
- **L6 — Parity is the acceptance test.** Until the flip, every rebuilt screen
  is judged against the old screen: same capabilities, adversarial-data clean,
  side-by-side screenshots. The old frontend stays untouched as the reference.
- **L7 — All existing repo laws apply.** Synthetic identity, motion tokens
  single-sourced, one `new Audio(`, reduced-motion static, division vocabulary
  from the registry (`divisions.svelte.ts` pattern), zero announce toasts.

## Layout doctrine (the anti-jank physics)

- Exactly ONE scroll region per screen by default: `PageShell`'s content area.
  Anything else that scrolls declares it (`<Scroll>` region) — never accidental.
- Every flex/grid child gets `min-width: 0` from its primitive parent. The
  classic blow-out bug is structurally impossible.
- Wide content (tables) lives in primitives that own `overflow-x` internally.
- Container queries over viewport queries: components adapt to their container,
  so composition never breaks responsiveness.
- Adversarial fixtures are the default test data: 200-char names, 15-digit
  amounts, empty lists, 500 rows, RTL text. Happy-path-only data is a bug.
- **Anti-collapse:** `min-width: 0` prevents overflow but permits collapse-to-
  zero (a title wrapping one-letter-per-line). Flexible text containers declare
  a readable flex-basis and siblings wrap rather than squeeze (found live in
  PageShell on day one — the detector must assert BOTH no-overflow AND
  no-degenerate-text-column, i.e. no multi-word text element narrower than ~4ch).

## Status

Sandbox on branch `exp/frontend-kernel` (worktree `asymmflow-lab`), local-only,
never pushed. Old frontend untouched; Wails still builds it. The flip criterion:
pilot screens at functional + visual parity, gates green, tripwires in place.
