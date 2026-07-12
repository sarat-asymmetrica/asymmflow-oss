# The AsymmFlow Design Constitution

**Version 1.0 · June 2026**
**Lineage:** Onyx & Ether v2.0 (Acme Instrumentation deployment) × the Show & Sell Standard §5 ("forbid the mean") × the Asymmetrica mathematical substrate.

> Serious software can be gorgeous. The notion that it can't is a failure of imagination,
> not a law of enterprise. This document exists so that every component, screen, and
> theme built for AsymmFlow — by humans or by agents — lands above the statistical
> mean of all web design, on purpose, every time.

---

## 0. The thesis: calm at the center, magic at the edges

An ERP earns trust at the **data surfaces** — tables, forms, ledgers, reconciliations.
Those surfaces are quiet, dense, monochrome-leaning, and ruthlessly legible.
It earns love at the **edges** — transitions, arrival moments, feedback, ambient
depth. That is where motion, three.js, and generative art live.

The two never trade places:

| Zone | Examples | Personality |
|---|---|---|
| **Center (calm)** | DataTable, forms, ledgers, detail views | Editorial restraint. Contrast over color. Patterns over alerts. Zero decoration. |
| **Edges (magic)** | Login ceremony, page transitions, empty states, dashboards' ambient layer, success moments | Earned motion, depth, generative identity. Confidence, not a circus. |

A component that mixes the zones (an animated gradient table header, a confetti
ledger) violates the constitution.

---

## 1. The five layers

```
@asymmflow/glyphs      icon language + generative marks          (identity)
@asymmflow/scenes      three.js ambient & data-art               (the magic edges)
@asymmflow/patterns    composed UX: CommandBar, QuickCapture,    (workflows)
                       DataShell, permission-gated nav
@asymmflow/ui          primitives: Button…DataTable              (the calm center)
@asymmflow/motion      quaternion/SLERP transition engine        (how state moves)
@asymmflow/tokens      the single token source + theme contract  (what everything is)
```

Each layer may import only from layers **below** it. Nothing imports upward,
nothing imports sideways into app code.

---

## 2. The hard rules (enforced, not aspirational)

1. **No component imports `wailsjs/`, app stores, or domain services.**
   Typed props in, events out. This is what makes the library portable across
   every module of the modular monolith — and every future product.
2. **One token source.** Every color, radius, duration, easing, z-index, and
   spacing value resolves to a `--af-*` custom property from `@asymmflow/tokens`.
   A raw hex value or millisecond literal inside a component is a defect.
3. **Themes are data, not code.** A theme is a complete assignment of the token
   contract (see `@asymmflow/tokens` `Theme` type). This is the seam where the
   style_alchemy engine generates themes from seeds — mechanically.
4. **Showcase-driven development.** A component does not exist until it renders
   in the showcase with all of its states (default / hover / focus / active /
   disabled / loading / error / empty / RTL where relevant).
5. **`prefers-reduced-motion` is a first-class theme.** Every animation has a
   reduced equivalent (usually opacity-only or none). Non-negotiable.
6. **Accessibility floor:** visible focus states everywhere, AA contrast,
   ≥44px touch targets (the `--af-tap-min` token — applied under
   `@media (pointer: coarse)` so refined desktop density is preserved while touch
   devices get full targets), semantic landmarks, focus containment in every
   layered surface (modal, drawer, command bar). The floor, not the ceiling.
7. **Interruptible motion.** Any transition can be redirected mid-flight without
   snapping. (This is what the SLERP engine is *for* — see §5.)

---

## 3. The blocklist — "an underspecified agent wrote this" tells

Inherited from the Show & Sell Standard, binding here:

```
✗ the indigo→violet→fuchsia gradient (the #1 AI-design fingerprint)
✗ everything one border-radius, everything one soft drop-shadow
✗ Inter-everywhere with zero pairing / zero hierarchy contrast
✗ gray-on-gray low-contrast body text
✗ emoji-as-icons in product UI
✗ glassmorphism with no reason · animate-on-everything
✗ colored status pills for every state (color is a scarce resource — spend it rarely)
✗ pure #000 on pure #FFF (derive neutrals from the accent; warmth is intentional)
✗ decorative parallax · spinning logos · lottie-confetti
```

And the bar, also binding:

```
Linear / Stripe / Vercel / Apple restraint · editorial typography
ONE confident accent + a properly-derived neutral ramp
real grid + intentional asymmetry + rhythmic, generous whitespace
strong hierarchy: size · weight · measure doing the work, not boxes and borders
motion must be earned — it communicates state, never decorates
specific, opinionated microcopy — never lorem-energy
```

---

## 4. The grammar (what stays fixed across all themes)

Themes change *values*; the grammar — the relationships — is constitutional.

### 4a. Type
- Two families max: a **display/numeric** face (tabular numerals, used for titles,
  KPIs, and every financial figure) and a **body/UI** face. Default pairing:
  Space Grotesk × Rubik (proven in the field at Acme Instrumentation).
- Modular scale ratio ≈ 1.2 (minor third): 11 → 13 → 14 → 16 → 19 → 24 → 29 → 48.
- ALL financial figures use `font-variant-numeric: tabular-nums lining-nums`. Always.
- Labels: 11px, 600, uppercase, 0.08em tracking. The quiet backbone of density.

### 4b. Space — the φ ladder
Additive Fibonacci-style scale, 4px-grid compatible, ratio → φ:

```
--af-space-1: 4px    --af-space-5: 32px
--af-space-2: 8px    --af-space-6: 52px
--af-space-3: 12px   --af-space-7: 84px
--af-space-4: 20px   --af-space-8: 136px
```

Each step is the sum of the previous two. Vertical rhythm uses the ladder; ad-hoc
margins are defects.

### 4c. Color
- ONE accent. A neutral ramp **derived from** the accent (tinted, never pure gray).
- Semantic tokens only in markup (`--af-surface`, never `#FFFFFF`).
- Status (success/warn/danger) exists but is spent **rarely** — the default
  status language is monochrome iconography and weight, per Onyx & Ether.
- AA contrast floor on all text tokens, verified in the showcase.

### 4d. Elevation
- Resting cards: 1px border, **no shadow**.
- Hover: "the Lift" — a single soft ambient shadow + nothing else.
- Only layered surfaces (modal, dropdown, command bar) get true elevation.
- Never colored shadows.

### 4e. Motion — the three-regime policy
Motion tokens are organized by **regime**, matching the substrate's universal
30/20/50 dynamics:

| Regime | Used for | Duration | Easing character |
|---|---|---|---|
| **R1 · Explore** | entrances, reveals, arrivals | 300–500ms | decelerate (fast in, soft landing) |
| **R2 · Optimize** | micro-interactions: hover, press, toggle, focus | 90–200ms | tight symmetric |
| **R3 · Stabilize** | exits, settles, confirmations, collapses | 200–300ms | accelerate-out / gentle settle |

- Animate **opacity + transform only** (GPU-composited). Never width/height/top/left.
  - *Exception — one-shot geometry reveals:* SVG entrance draws (line-draw and
    arc-draw via `stroke-dashoffset`) are permitted **only** when they (a) run once
    on mount, (b) are `prefers-reduced-motion` gated, and (c) are driven by motion
    tokens. Sweeping an arc has no transform equivalent, and a single mount-time
    paint costs nothing on the main thread. Continuous or interactive animation
    stays opacity + transform — no exceptions.
- Stagger siblings 40–60ms. Never linear easing except continuous loops.
- Nothing exceeds 700ms without an explicit ceremony justification (login, first-run).
- Complex multi-property state changes route through `@asymmflow/motion`'s SLERP
  core (§5) so they remain coherent and interruptible.

---

## 5. The mathematical substrate (where it genuinely earns its place)

We use the math where it produces an experiential difference, never as branding:

- **SLERP state transitions** (`@asymmflow/motion`): a UI state (position, scale,
  opacity, rotation) is encoded as a point; transitions follow the geodesic at
  constant angular velocity. Result: every property arrives *together*, and a
  mid-flight redirect re-SLERPs from the current point — no tween-killing jank.
  This is the tactile signature of the system: AsymmFlow never snaps.
- **φ in the spacing ladder and stagger rhythms** (§4b): consonant proportions
  users feel but can't name.
- **Three-regime motion taxonomy** (§4e): a *vocabulary* that keeps motion
  consistent across dozens of components and contributors.
- **Generative identity** (`@asymmflow/scenes`, `@asymmflow/glyphs`): seeds →
  deterministic quaternion walks on S³ → unique-but-coherent marks, login
  ceremonies, and ambient fields. Same engine family as style_alchemy.

What we do NOT do: claim the math makes buttons better. A button is better
because its states are legible and its hit target is 44px.

---

## 6. Conformance checklist (per component, enforced in review)

```
[ ] Renders in showcase with all states, light + density variants
[ ] Zero raw hex / px-duration / z-index literals (tokens only)
[ ] No imports from wailsjs/, app stores, or screens
[ ] Keyboard path complete; focus visible; ARIA roles correct
[ ] prefers-reduced-motion variant present
[ ] RTL-safe (logical properties; verified for ar locale)
[ ] Types exported; props documented in the showcase page
```

---

**Lineage acknowledgment:** Onyx & Ether was shipped under deployment pressure by
Jordan and refined in the field at Acme Instrumentation, Bahrain. This constitution is its
formalization, not its replacement — the field-proven decisions (density, tabular
numerals, monochrome status language, the Lift) are kept as constitutional law.

**Om Lokah Samastah Sukhino Bhavantu** — may every builder who clones this repo
ship something beautiful with it.
