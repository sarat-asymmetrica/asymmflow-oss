# A1 — Interaction Inventory (feeds B1)

## Canonical button reality
Three "Button" components exist; only ONE is live in the shipping app:

| Component | Path | Live? |
|---|---|---|
| **Button.svelte** (de-facto canonical) | `frontend/src/lib/components/ui/Button.svelte` | **YES** — `<Button` in 290 call sites / 37 files |
| WabiButton.svelte | `frontend/src/lib/components/ui/WabiButton.svelte` | narrow (3 files: ErrorBoundary, EcosystemDashboard, routes/ecosystem/+page) |
| Button.svelte (Onyx & Ether) | `packages/ui/src/form/Button.svelte` | **NO** — `@asymmflow/ui` not a frontend dep. Reference impl only. |

## Current states
- **Button.svelte (live):** `:hover`/`:active` per variant are flat background swaps. **No transform/scale on press — zero tactile depression.** `.btn-ghost` and `.btn-secondary` have NO `:active` at all. Focus: `.btn:focus-visible { outline: 2px solid var(--brand-indigo); outline-offset: 2px; }` (component-local, hardcoded var).
- **WabiButton.svelte:** hover only, no `:active` anywhere. Focus uses `var(--color-ink,#1c1c1c)`.
- **packages/ui Button (orphaned reference):** `.af-btn:active:not(:disabled):not(.af-btn--loading){ transform: scale(0.985); }` + per-variant `:active` bg/shadow + `color-mix()` hover. Inherits global `:focus-visible{outline:2px solid var(--af-focus-ring)}` (`--af-focus-ring:#2A7532`). **This is the press pattern to port.**

## Ad-hoc bypasses
- 101 files contain raw `<button>`; 486 raw tags vs 290 `<Button>` uses.
- By dir: lib/screens 48, lib/components/ui 18, lib/components misc 15, workhub 5, layout 4, customer 2, consciousness 2, asyl 2.
- **WorkHub** (`lib/components/workhub/*`): only `WorkHubTaskDetailModal` uses `<Button>`. Other 5 panels hand-roll ~19 raw `<button>` with bare `.primary`/`.danger`/unclassed — no `:hover`/`:active`/`:focus-visible` locally. **Highest-value, lowest-risk convergence target.**
- **CustomerDetail** (`lib/components/customer/*`): mostly compliant. 2 bypasses: `CustomerContactsStrip.svelte:39` (`.add-contact-btn-inline`), `CustomerDetailHeader.svelte:29` (`.back-btn`). Also `CustomerDetailHeader.svelte:41` fakes danger via inline `style="color:#e74c3c;border-color:#e74c3c"` on a `variant="secondary"` — should be `variant="danger"`.

## Focus-ring token status
No single token. Four overlapping `:focus-visible` rules:
1. `frontend/src/styles/global.css:218` — `outline:2px solid var(--brand-indigo)` (global `*`) — **note: global.css may be unimported (see A2); verify)**
2. `frontend/src/lib/styles/accessibility.css:49` — `var(--color-ink,#1c1c1c)` (also likely orphaned)
3. Component-local in Button.svelte (`--brand-indigo`) + WabiButton (`--color-ink`)
4. `packages/tokens/css/base.css:91` — `--af-focus-ring:#2A7532` (only semantic token, unused system)

`--brand-indigo` is aliased to `var(--carbon)` (near-black).

## B1 recommendations
1. Define a single `--focus-ring` token in `frontend/src/assets/design-tokens.css` (the LIVE token file per A2), value = current `--brand-indigo`, so no visual regression. Point Button.svelte + WabiButton at it.
2. Own the press state in `frontend/src/lib/components/ui/Button.svelte` (NOT packages/ui). Port `transform: scale(0.985)` (or ~0.97 per Constitution IV.2) on `:active` for all variants, add missing `:active` to ghost/secondary. ≤100ms via `--transition-fast`.
3. Converge ad-hoc buttons, priority: (a) WorkHub 5 panels; (b) CustomerContactsStrip:39 + CustomerDetailHeader:29; (c) CustomerDetailHeader:41 → `variant="danger"`.
4. Do NOT adopt packages/ui this wave — reference only.
