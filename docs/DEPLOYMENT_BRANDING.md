# Deployment Branding — the rebrand recipe

AsymmFlow ships a **synthetic** identity ("AsymmFlow"). Branding is *configuration,
not code* (repo law): a deployment re-skins its identity by overriding config/token
values and swapping build assets — **zero source edits**. This document lists every
brand slot, the one place each is overridden, and the two-step rebrand recipe.

> **Synthetic invariant (CLAUDE.md #2):** the committed tree must only ever contain
> the synthetic default. Real deployment identity (names, logos, colors, letterheads)
> lives in **gitignored** local files (web layer) and in **deployment-side** assets
> (`overlay.json`, icons) that are authored on the deployment's own branch/host — never
> committed here.

---

## The brand slots

| # | Slot | What it controls | Where it's overridden | Layer |
|---|------|------------------|-----------------------|-------|
| 1 | **App wordmark + mark** | Sidebar header + login/lock surface | `frontend/src/lib/brand.local.ts` (gitignored) | Web runtime config |
| 2 | **Accent color** | The accent used by the wordmark badge (and any token consumer) | `--brand-indigo` token, or `accentVar` in `brand.local.ts` | Web token |
| 3 | **PDF / print header identity** | Legal name, letterhead, address, VAT/TRN on generated documents | `overlay.json` next to the binary | Backend config |
| 4 | **Desktop app name** | Window title / installer name | `wails.json` name (binary/installer) + `main.brandWindowTitle` ldflags (window title) | Build config |
| 5 | **Desktop app icon** | Taskbar / window / installer icon | `build/appicon.png` + `build/windows/icon.ico` | Build asset (baked) |

---

## 1–2. Web-layer identity (wordmark, mark, accent) — runtime config, ONE file

The single source of app identity is `frontend/src/lib/brand.ts`, which exports a
`brand` object with `{ wordmark, mark, accentVar }`. The shipped default is synthetic:

```ts
{ wordmark: 'AsymmFlow', mark: 'AF', accentVar: 'var(--brand-indigo)' }
```

Both consumers — `EnterpriseSidebar.svelte` (sidebar header) and `LoginScreen.svelte`
(login/lock) — read from this one object. To re-skin, a deployment creates a
**gitignored** `frontend/src/lib/brand.local.ts` that default-exports a partial override:

```ts
// frontend/src/lib/brand.local.ts  — GITIGNORED, deployment-local, NEVER committed
import type { BrandIdentity } from './brand';
const override: Partial<BrandIdentity> = {
  wordmark: 'Your Company',
  mark: 'YC',
  accentVar: '#5CB550',   // a literal color, or 'var(--your-accent-token)'
};
export default override;
```

`brand.ts` resolves this via `import.meta.glob('./brand.local.ts', { eager: true })`:
when the file is **absent** (the shipped state) the glob is empty and the default
stands; when present, its fields shallow-merge over the default. No source edit, no
build flag. The pattern `frontend/src/lib/brand.local.ts` is in `.gitignore`.

**Accent, deeper:** the wordmark badge tints from `accentVar`. If a deployment wants the
accent applied through the design-token layer instead (so other token consumers pick it
up), override the `--brand-indigo` value in a deployment token file. **Keep it lightly
personal** (owner ratification 2026-07-13): the accent + wordmark + icon carry identity;
do **not** restyle components, charts, or semantic status colors. In particular the
deal-stage / status colors are semantic tokens and must stay put — see the collision note
below.

### Accent-green vs. semantic-success collision (checked)
The flagship deployment uses a green accent. The app's semantic status palette
(including "success"/"done") is a **separate** set of tokens, and the `DealTimeline`
component renders on **monochrome contrast tokens** (not hues) — so a green accent cannot
be confused with a "done = green" node there (there is no green in the timeline). Any
deployment that also remaps a *status* token to green must re-check screens where accent
and status co-occur; the shipped synthetic default has no such collision.

## 3. PDF / print header identity — `overlay.json` (already config-driven)

Generated documents (offers, invoices, DNs, POs, statements) resolve their letterhead,
legal name, address, and VAT/TRN from the **company overlay**, not from source. The
backend (`company_branding.go` → `pkg/overlay`) loads an `overlay.json` placed next to
the binary (or in a config search dir) via `overlay.LoadOverlay`; with no file it falls
back to `overlay.BuiltinDefaults()` (synthetic). See `data/overlay.json` for an annotated
example. A deployment supplies its own `overlay.json` (authored deployment-side) — no code
change in this repo. Letterhead image assets are likewise deployment-supplied and
referenced from the overlay.

## 4–5. Desktop app name + icon — BUILD assets (baked at build time)

Wails bakes the app name and icon into the binary at build time — these are **build
assets, not runtime config**:
- **Name:** `wails.json` → `name` (default `AsymmFlow`).
- **Icon:** replace `build/appicon.png` (source PNG Wails derives platform icons from)
  and, on Windows, `build/windows/icon.ico`. Then rebuild (`wails build -clean`).

A deployment keeps its real name/icon on its own branch or applies them as a pre-build
asset swap in its pipeline; the committed repo keeps the synthetic defaults.

The runtime **window title** (OS title bar, alt-tab) is a separate slot: the
`main.brandWindowTitle` package var in `main.go` (synthetic default `AsymmFlow`),
overridden at build time with no source edit —
`wails build -ldflags "-X 'main.brandWindowTitle=Your Product'"`. `wails.json` `name`
only sets the binary/installer name, not the runtime title.

---

## The two-step rebrand recipe

1. **Config/token override (runtime):**
   - Add `frontend/src/lib/brand.local.ts` (wordmark, mark, accent) — gitignored.
   - (If shipping printed documents) place your `overlay.json` next to the binary.
2. **Build-asset swap (once, at build):**
   - Replace `build/appicon.png` (+ `build/windows/icon.ico`), set `wails.json` `name`,
     and run `wails build -clean`.

That's it — no source files change. The flagship deployment's actual override files are
authored on the deployment (ph_holdings convergence) side and never enter this repo.
