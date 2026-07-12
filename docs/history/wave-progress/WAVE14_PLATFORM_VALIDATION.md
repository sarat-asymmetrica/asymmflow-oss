# Wave 14 Platform Validation

Date: 2026-05-07  
Project: AsymmFlow (`asymmflow`)  
Scope: Wails v3, Svelte 5 migration tooling, Pretext PDF library

## Wails v3

Status: available, with install caveat.

Official docs checked:
- `https://v3.wails.io/quick-start/installation/`
- `https://v3.wails.io/getting-started/installation/`

Validation command:

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@latest 2>&1
```

Result: failed from module proxy/source module because the published module contains `replace` directives.

Observed error:

```text
The go.mod file for the module providing named packages contains one or
more replace directives. It must not contain directives that would cause
it to be interpreted differently than if it were the main module.
```

Fallback validation:

```powershell
git clone --depth 1 https://github.com/wailsapp/wails.git %TEMP%\wails-v3-src-codex
cd %TEMP%\wails-v3-src-codex\v3\cmd\wails3
go install
wails3 version
wails3 doctor
```

Result:
- `wails3 version`: `v3.0.0-dev`
- `wails3 doctor`: system ready for Wails development
- Windows compatibility: doctor passed on Windows amd64 with WebView2 available
- Caveat: checkout reported Windows filename-length failures in Wails generator testdata, but the CLI still built and doctor completed

Recommendation:
- Proceed with Wails-v3-ready service architecture in Wave 14.
- Treat the actual v3 app runtime migration as experimental.
- Prefer attempting Wails v3 after service binding + Svelte 5 gates pass, so the fallback state remains useful even if v3 build wiring needs a later pass.

Follow-up result after Tickets 2-7:
- Wails v3 source examples and `pkg/application` API were inspected.
- The v3 app model uses `github.com/wailsapp/wails/v3/pkg/application`, `application.New(...)`, `application.NewService(...)`, and `app.Window.NewWithOptions(...)`.
- The existing AsymmFlow backend imports Wails v2 runtime APIs in many root service files for events, dialogs, file pickers, and message dialogs.
- Wails v3 does not expose the same `github.com/wailsapp/wails/v2/pkg/runtime` surface; these calls need a compatibility adapter or a dedicated runtime-dialog-event migration.

Ticket 8 decision:
- Defer the actual runtime switch to Wails v3.
- Reason: the CLI is available only through source install on this machine, the direct `go install ...@latest` path fails, source checkout hit Windows long-path errors in v3 testdata, and the v2 runtime API usage across the app makes a direct `main.go` flip too risky for a pure structural wave.
- The Wave 14 domain service layer is v3-ready preparation: the future migration can register `FinanceService`, `CRMService`, `ButlerService`, `DocumentsService`, `SyncServiceBinding`, and `InfraService` as Wails v3 services once runtime/dialog/event adapters are in place.

## Svelte 5 Migration Tool

Status: available.

Validation command:

```powershell
cd frontend
npx sv migrate svelte-5 --help 2>&1
```

Result:
- `sv@0.15.2` was installed by `npx`
- `sv migrate` help displayed successfully

Recommendation:
- Use `npx sv migrate svelte-5`.
- Run the migration after package upgrade.
- Expect manual follow-up because this frontend has 186 Svelte components and many Wails imports.

## Pretext PDF Library

Status: requested module not available.

Validation command:

```powershell
go list -m github.com/nicholasgasior/pretext@latest 2>&1
```

Result:

```text
remote: Repository not found.
fatal: repository 'https://github.com/nicholasgasior/pretext/' not found
```

Recommendation:
- Defer Pretext.
- Continue using existing PDF stack for this wave (`gofpdf`, `gopdf`, existing document services).
- Revisit document/PDF rendering as a dedicated wave if a maintained Go Pretext package or another strong candidate is identified.

## Wave 14 Execution Recommendation

Proceed with:
- Ticket 2: domain service delegation layer
- Ticket 3: Wails v2 binding of domain services and binding generation
- Ticket 5-7: Svelte 5 package/migration/fixes
- Ticket 4: frontend imports to domain service namespaces
- Ticket 8: experimental Wails v3 migration attempt, with documented fallback if the v3 app API/build path becomes unstable
