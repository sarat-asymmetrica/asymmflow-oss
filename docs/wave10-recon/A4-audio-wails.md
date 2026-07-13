# A4 Recon — B4 Application Sound: Playback Mechanism & Opt-Out Setting

Wave 10 (Sensory & Brand). Read-only recon. Repo: `asymmflow-oss` (Wails v2.11 / WebView2 / Svelte 5 + Vite).

## 1. Asset embedding — recommended approach

**Recommendation: Vite JS asset import (not a `public/` folder, not base64).**

Evidence for how binary assets are already handled in this repo:

- There is **no `frontend/public/` directory** (`ls frontend` shows no `public`). Static
  assets are *not* served from a passthrough public folder.
- `frontend/src/app.css` already references binary files (fonts) with relative `url()`
  paths that Vite resolves at build time:
  ```css
  @font-face {
    src: url('./assets/fonts/google/Rubik-Regular.ttf') format('truetype');
  }
  ```
  This is Vite's default asset pipeline (no special plugin — `vite.config.ts` only loads
  `@sveltejs/vite-plugin-svelte` and a `manualChunks` split, nothing asset-related). Vite
  hashes these files and copies them into `frontend/dist/assets/` at build time; the CSS
  `url()` is rewritten to point at the hashed output path.
- `main.go` embeds the **built** output wholesale and serves it through Wails' own asset
  server — not a raw filesystem/network path:
  ```go
  //go:embed all:frontend/dist
  var assets embed.FS
  ...
  AssetServer: &assetserver.Options{ Assets: assets, ... }
  ```
  So anything Vite puts in `frontend/dist/assets/` is embedded into the Go binary and
  served by Wails' in-process asset server at app runtime — there is no external network
  call involved either way (embedded-serve vs. data URI are equally "local"), so pick
  for **bundle simplicity and cacheability**, not paranoia about the network.

Because the font `url()` pattern already proves Vite's binary-asset pipeline works
end-to-end through the Wails embed, the same mechanism works for a JS/TS import of a
`.wav`/`.ogg` file — Vite treats any imported non-JS/CSS file as an asset by default and
returns the resolved runtime URL as the import's default export:

```ts
// exact import syntax for this repo (Vite 5 default asset handling, no plugin needed)
import paidChimeUrl from '../../assets/sounds/paid-chime.wav';
```

`paidChimeUrl` is a string like `/assets/paid-chime.<hash>.wav` after build (and a
dev-server URL under `wails dev`). Place the file at
`frontend/src/assets/sounds/paid-chime.wav` (new `sounds/` subfolder alongside the
existing `fonts/` folder in `frontend/src/assets/`) to match the existing asset
layout convention.

Base64 data-URI is a viable fallback (keeps everything in the JS bundle, zero extra
files) but is not necessary here — the font precedent shows the asset-import path is
already proven in this codebase, so prefer consistency with existing conventions over a
data URI.

## 2. Autoplay / user-gesture confirmation

WebView2 is Chromium-based and follows the standard Chromium Media Engagement /
autoplay-with-sound policy: audio playback (`HTMLMediaElement.play()` or `new
Audio(url).play()`) initiated **synchronously inside a user-gesture event handler**
(e.g. a `click` handler) is allowed unconditionally — no flags, no `muted` trick, no
prior media-engagement score needed. This only fails if the `.play()` call happens
*outside* the gesture (e.g. after an `await` that yields past the gesture window, or on
a `setTimeout`/promise-chain callback not directly attached to the click). Practical
rule for B4: call `new Audio(paidChimeUrl).play()` as the **first statement** inside the
existing payment-submit click handler, before any `await`.

Concrete target handler in this repo: `handleModalSubmit` in
`frontend/src/lib/screens/PaymentsScreen.svelte` (bound via `on:click={handleModalSubmit}`
at line ~1470) — this is the "Record Payment" modal submit button, i.e. the actual
click gesture that should carry the sound trigger for the paid-in-full case. (Do not
wire it — Wave 10 build task, out of scope for this recon.)

Existing audio usage — confirmed **zero**:
```
grep -rniE "new Audio|<audio|\.play\(|\.wav|\.ogg|\.mp3" frontend/src  →  no matches
```
No `<audio>` elements, no `Audio()` constructors, no sound file references anywhere in
the frontend today. B4 would be the first and only application sound.

## 3. Opt-out setting location (default ON)

Settings are **not** a typed Go struct — they're a generic `map[string]any` persisted as
JSON, layered under top-level keys. Confirmed sources:

- Backend surface: `app_setup_documents_surface.go`
  - `func (a *App) GetSettings() (map[string]any, error)` (line 76) builds the map
    returned to the frontend, reading each field via
    `getSettingOrDefault(userSettings, "section.key", default)` — a dotted-path lookup
    over the persisted JSON blob (see `getSettingOrDefault`, line 143).
  - `func (a *App) UpdateSettings(settings map[string]any) error` (line 169) persists
    the whole map back via `a.saveUserSettings(settings)` (writes indented JSON to a
    `settings.json`-style file on disk, see the `MarshalIndent` block just above
    `GetSettings`, lines 62-73). Only a few fields get special in-memory side effects
    (folder paths, API keys, session timeout); everything else just round-trips through
    the JSON file untouched.
  - Bound to the frontend via `service_documents.go` `DocumentsService.GetSettings` /
    `DocumentsService.UpdateSettings` (thin pass-throughs to `a.app.GetSettings` /
    `a.app.UpdateSettings`), which is what the generated `wailsjs/go/main/DocumentsService`
    bindings expose.
  - There is a **separate**, stricter `SettingsService` (`settings_service.go`) with an
    encrypted per-key `Setting` DB table (HKDF/AES-GCM) — that one is used for
    sensitive values only (e.g. `apiKeys.aiml_model`, see the
    `a.settingsService.SetSetting("apiKeys.aiml_model", ...)` call in `UpdateSettings`).
    A sound-preference boolean is not sensitive, so it belongs in the plain JSON
    settings map, not the encrypted table.

- **Exact place to add the field**: in `GetSettings()` (app_setup_documents_surface.go,
  inside the `settings := map[string]any{...}` literal, ~line 96-136), add a new section
  (or extend `business`) — recommend a small new top-level section for clarity:
  ```go
  "sounds": map[string]any{
      "sound_on_paid_enabled": getSettingOrDefault(userSettings, "sounds.sound_on_paid_enabled", true),
  },
  ```
  Default `true` (opt-out model) falls straight out of `getSettingOrDefault`'s
  `defaultValue` argument — no extra plumbing needed for "default ON."
  `UpdateSettings` needs no special-case code for this field: since the whole map is
  persisted verbatim via `saveUserSettings(settings)`, a plain boolean under `sounds.*`
  round-trips automatically. Only add a branch there if the toggle needs an immediate
  side effect (it doesn't — it's just read at play-time).

- Frontend: `frontend/src/lib/screens/SettingsScreen.svelte` already has the pattern for
  a boolean settings section — mirror `office: { outlook_enabled, excel_enabled }` /
  `security: { session_timeout_minutes }` in the local `settings = $state({...})` object
  (around line 71-75): add `sounds: { sound_on_paid_enabled: true }`, wire a checkbox to
  it, and it flows through the existing `GetSettings()`/`UpdateSettings()` calls already
  imported at the top of the file (`UpdateSettings` from
  `wailsjs/go/main/App`, `GetSettings` from `wailsjs/go/main/DocumentsService`).
  There is no separate "user preferences" store/file distinct from this settings map —
  this generic JSON map *is* the user-preferences store for the whole app.

At the point of use (payment screen), the sound-trigger code should read the cached
settings value (however the app currently caches `GetSettings()` results — likely a
Svelte store or a fetch-on-mount local var in the consuming screen; not present today,
would need a small settings store/context read, which is in-scope for the Wave 10 build
task, not this recon) and simply skip `.play()` if `sound_on_paid_enabled === false`.

## 4. Minimal proof snippet (NOT wired into the app)

```svelte
<script lang="ts">
  // Exact Vite asset-import syntax verified against the fonts precedent in app.css
  // and vite.config.ts (no plugin needed — Vite's default asset handling).
  import paidChimeUrl from '../assets/sounds/paid-chime.wav';

  function handleClick() {
    // Must be the first thing in the click handler — no await before this line,
    // or WebView2/Chromium's autoplay-with-sound gesture attribution is lost.
    new Audio(paidChimeUrl).play().catch((err) => {
      // Playback failures (missing file, decode error) should never block the
      // real payment-posting flow — swallow and log only.
      console.warn('sound playback failed', err);
    });
  }
</script>

<button type="button" onclick={handleClick}>Simulate Paid</button>
```

`vite.config.ts` (`frontend/vite.config.ts`) confirms nothing needs to change there —
it has no `assetsInclude` restriction and no custom asset plugin, so Vite's built-in
asset-import behavior (which already covers `.ttf`/`.otf` via CSS `url()` in this repo)
applies to `.wav`/`.ogg`/`.mp3` out of the box.

## Summary of open questions for the Wave 10 build task (not answered by recon)

- Whether to add a dedicated `sounds` settings section (recommended above) or fold it
  into `business`.
- Where the frontend caches/reads `GetSettings()` today for screens outside
  SettingsScreen (a payment screen needs read access to `sounds.sound_on_paid_enabled`
  without a full settings-page round trip) — no existing shared settings store was found
  in this recon pass; likely needs one small addition (e.g. a Svelte store hydrated once
  at app boot from `GetSettings()`), same pattern as `frontend/src/lib/stores/textScale.ts`.
