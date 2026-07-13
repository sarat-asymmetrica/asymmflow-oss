# A2 — Motion Census + Token-Layer Verdict (feeds B2)

## B2 CANONICAL TOKEN FILE (the verdict)
**`frontend/src/assets/design-tokens.css`** — the ONLY token CSS in the real import chain.

Import chain from `frontend/src/main.ts`:
```
main.ts:1  import './assets/design-tokens.css'   ← FIRST (canonical)
main.ts:2  import './app.css'                     (fonts only, @font-face)
main.ts:3  import './assets/theme.css'            (semantic color remap)
main.ts:4  import './assets/layout.css'
```
It ALREADY owns (lines 114-120):
```
--transition-fast: 120ms cubic-bezier(0.25,0.1,0.25,1);
--transition-base: 200ms cubic-bezier(0.25,0.1,0.25,1);
--transition-slow: 400ms cubic-bezier(0.25,0.1,0.25,1);
--transition-spring: 500ms cubic-bezier(0.34,1.56,0.64,1);   /* spring — B2 should retire/avoid */
--easing-smooth: cubic-bezier(0.25,0.1,0.25,1);
--easing-spring: cubic-bezier(0.34,1.56,0.64,1);              /* spring */
```
**New `--motion-*` duration/easing tokens go HERE.** Recommend defining `--motion-fast` (~120ms), `--motion-base` (~200ms), `--ease-standard`, `--ease-decelerate` and pointing existing `--transition-*` at them (or aliasing) so there is one source. Do NOT introduce a spring/bounce easing into the vocabulary (trading desk, not a game).

### Other token files — DO NOT edit (read-only verdict)
| File | Status |
|---|---|
| `frontend/src/app.css` | live but @font-face only |
| `frontend/src/assets/theme.css` | live, color remap, no motion |
| `frontend/src/styles/design-tokens.css` | DUPLICATE NAME, v1.0, **never imported** — dead |
| `frontend/src/lib/styles/phi-design-tokens.css` | **never imported** — dead |
| `frontend/src/lib/styles/wabi-sabi.css` | imported only by dead `routes/+layout.svelte` (SvelteKit remnant, not wired to Vite/Wails) — dead |
| `packages/tokens/css/*` | not consumed by frontend/src at all; aspirational future system (already has `--af-motion-*`) — leave |

## prefers-reduced-motion — EFFECTIVELY ABSENT
- Only LIVE instance: component-scoped block in `DataTable.svelte:674` (used in 19 screens).
- Three global resets exist but are ALL orphaned/unimported: `accessibility.css:61`, `wabi-sabi.css:378` (dead route only), `global.css:241`.
- **B2 must add a LIVE global `@media (prefers-reduced-motion: reduce)` reset** — put it in the canonical chain (design-tokens.css or an imported global). Verify it actually loads (grep the import from main.ts). This is an AC: reduced-motion must render the app fully static.

## Modals/drawers/toasts today (inconsistent — B2 unifies)
- **Global toast:** `ToastContainer.svelte` `animate:flip {duration:144}`; per-toast `WabiSabiToast.svelte` `in:fade/out:fade {duration:180}`.
- **Global confirm:** `ConfirmHost.svelte` → `lib/components/layout/Modal.svelte` — **ZERO animation** (just `{#if open}`).
- **Global OCR modal:** `QuickCaptureModal.svelte` — inline `svelte/transition` `fade`+`scale {duration:200, start:0.95}`.
- **Third modal:** `lib/components/Modal.svelte` — raw CSS `@keyframes fadeIn 150ms` + `slideUp 200ms`.
- **Drawers:** none exist in live app.
- **Verdict:** 3 modal impls, 3 durations (0/150/200ms), no shared token. B2 should give modal/toast a consistent fade+small-translate enter/exit driven by the motion tokens. (Do not rewrite modal STRUCTURE — flows frozen — only converge the transition timing/easing onto tokens.)

## Animations >250ms (flag; bring under ~250ms unless deliberate)
- `design-tokens.css:658 .animate-flourish` translateY/opacity 500ms spring
- `design-tokens.css:654 .animate-slide-up` 400ms
- `IntelligenceHub.svelte:33-34` opacity+transform 0.5s spring
- progress-bar width transitions 0.5–0.6s: `RegimeBadge:191`, `MathematicalRigorBadge:260`, `FlowIndicator:185`, `ConfidenceMeter:141`, `SurvivalPanel:376` (these animate `width` — functional, may keep but consider)
- `OrigamiNav.svelte:409 @keyframes pulse` 2s infinite; GSAP `elastic.out` at :50,:136
- shimmer/skeleton loops ~1.5-2s (Skeleton, WabiSkeleton, DataTable:602) — infinite loaders, acceptable but must stop under reduced-motion
- `HoloCard.svelte:52` tilt 0.6s; `CursorFollower.svelte:126` spring 0.25s

Spring/bounce easing `cubic-bezier(0.34,1.56,0.64,1)` appears in design-tokens.css, IntelligenceHub, CursorFollower — B2 vocabulary should NOT include it. Many of these live in "consciousness"/showcase components that may be low-traffic; prioritize the ones on real business screens.
