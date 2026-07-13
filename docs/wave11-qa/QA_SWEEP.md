# The Mirror — standing QA sweep (Wave 11)

A repeatable, browser-driven screenshot sweep of every primary screen. Born in
Wave 11 to find the visual defects the compile/test gates can't see (flat renders,
broken tab layouts, identity leaks). "Run the mirror" is now a standing capability:
rerun it after any UI change to catch regressions no unit test will.

## What it does

Drives the **vite dev server** in headless Chromium with the **synthetic mock
Wails bridge** installed, navigates to each `NAV_ITEMS` screen (plus a few
deep-link surfaces) by URL hash, and writes a full-page screenshot per screen at
two widths. It asserts nothing about content — the screenshots ARE the
deliverable; a human (or the orchestrator) reviews them. Screens that render blank
without data are themselves findings.

## Run it

```bash
cd frontend
# one-time, if not already installed:
npx playwright install --with-deps chromium

# run the sweep (auto-starts the dev server via playwright.config webServer):
npx playwright test tests/e2e/wave11-sweep.spec.ts --project=chromium --workers=2
```

Output lands in **`docs/wave11-qa/<width>/<screen>.png`** (`1440/` and `1100/`).
Review every image. Diff against the committed baseline to spot regressions.

Optional deep-dive probe (computed styles / token resolution for a single screen)
lives in `tests/e2e/wave11-debug.spec.ts` — copy/adapt it when root-causing a new
defect. It is a debugging aid, not part of the standing sweep.

## How it renders truthfully (the harness)

The app has **two bound-service gaps** a bare browser can't fill; the mock bridge
(`tests/e2e/helpers/mockWailsBridge.ts`) closes them:

1. **Auth/device gate** — `ValidateLicense`/`CheckDeviceStatus` are mocked to
   return an approved admin (`permissions: ['*']`), so every screen is reachable.
2. **i18n at startup** — the app calls `InfraService.GetTranslations(locale)`
   during boot; unmocked it throws and the whole UI drops to its error boundary.
   The sweep passes the real `pkg/i18n/messages/en.json` into
   `installMockWailsBridge(page, { translations })` so labels render as designed.

All seven Go services (`App`, `InfraService`, `CRMService`, `FinanceService`,
`DocumentsService`, `ButlerService`, `SyncServiceBinding`) are wired to the mock's
generic proxy — method-name dispatch with safe no-op defaults — so a screen that
calls any of them still renders its layout. Data-less panels degrade to empty
states (or a warning toast), which is exactly what we want to see.

## Navigation map (source of truth: `frontend/src/lib/config/navItems.ts`)

Primary nav: `dashboard · opportunities · operations · finance · accounting ·
reports · work · people · notifications · relationships · intelligence · settings`.
Deep-link surfaces also swept: `usermanagement · deployment`.

Navigate by hash: `window.location.hash = '#<id>'`. `NAV_ITEMS` is the one list
that drives the sidebar, the Alt+N order, and the shell permission gate — keep the
sweep's screen list in step with it.

## Synthetic invariant (law)

This repo is public. Every screenshot, fixture, and example here uses **synthetic
identity only** (Acme Instrumentation, National Petroleum Co., Jordan Avery, ...).
Deployment identity — real names, colors, keys — NEVER appears in this repo, its
docs, or its committed screenshots. See `SYNTHETIC_IDENTITY.md`.

## Adding a screen / sub-view

Edit the `SCREENS` array in `tests/e2e/wave11-sweep.spec.ts`. For a screen whose
defect only shows in a sub-view (e.g. People → Employee Detail tabs), drive the
interaction in a dedicated spec modeled on `wave11-debug.spec.ts` and screenshot
after the state is reached.
