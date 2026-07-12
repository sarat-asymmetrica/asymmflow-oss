# Wave 16 Progress Audit - v0.1 Release Engineering

**Date**: 2026-05-08  
**Roadmap**: `docs/V0_1_RELEASE_ROADMAP_2026_05_08.md`  
**Commit range**: `0122f7c^..79a2e16`

## Commit Table

| Ticket | Commit | Result |
|---|---|---|
| Roadmap | `0122f7c` | Added dated v0.1 release roadmap and living execution log |
| 1 | `ea067e2` | Added release manifest package, build metadata model, tests, and `InfraService.GetBuildInfo` |
| 2 | `e124f28` | Added Settings build display, Windows release bundle script, and v0.1 release checklist |
| 3 | `ecc0337` | Added one-command release verification script |
| 4 | `79a2e16` | Added SQLite integrity command, backup/restore preflight script, and preflight runbook |

## Release Identity

- Product: `AsymmFlow`
- Version: `0.1.0-alpha.1`
- Channel: `alpha`
- Manifest: `pkg/infra/release/manifest.json`
- Runtime endpoint: `InfraService.GetBuildInfo()`
- Frontend display: Settings system info panel shows version, channel, and commit prefix when available.

## Release Tooling

- `scripts/verify_release.ps1` runs Go build, Go tests, frontend build, frontend check, and optional `wails build`.
- `scripts/build_release_windows.ps1` stamps version/commit/build-time ldflags and builds a Windows amd64 artifact bundle.
- Release artifacts now include the roadmap, release checklist, backup/restore preflight runbook, and preflight helper script.
- `cmd/sqlite_integrity` verifies SQLite database health using the app's pure-Go SQLite driver.
- `scripts/preflight_backup_restore.ps1` checks active DB integrity, backup DB integrity, and a sandbox restore copy.

## Backup/Restore Evidence

| Gate | Result |
|---|---|
| `go run ./cmd/sqlite_integrity -db ph_holdings.db` | Passed |
| `.\scripts\preflight_backup_restore.ps1 -ActiveDb "$env:APPDATA\AsymmFlow\ph_holdings.db"` | Passed |
| Active AppData DB integrity | `ok`, 0 foreign-key check rows |
| Latest AppData backup integrity | `ok`, 0 foreign-key check rows |
| Sandbox restore copy integrity | `ok`, 0 foreign-key check rows |

## Verification Results

| Gate | Result |
|---|---|
| `go test ./cmd/sqlite_integrity ./pkg/infra/release -count=1` | Passed |
| `.\scripts\verify_release.ps1 -SkipWailsBuild` | Passed |
| `go build ./...` | Passed via verification script |
| `go test ./... -count=1 -timeout 300s` | Passed via verification script |
| `cd frontend && npm run build` | Passed via verification script |
| `cd frontend && npm run check` | Passed with 0 errors, 13 existing warnings |
| `.\scripts\verify_release.ps1` | Passed, including `wails build` |

## Known Warnings

`npm run check` still reports 13 existing warnings across 10 Svelte files. They are unchanged from the previous waves and are mostly Svelte 5 state/a11y migration warnings. No errors are reported.

## Issues And Deviations

- The Windows bundle script is ready but has not been used to create a signed installer. v0.1 alpha packaging remains a zip-style release bundle.
- The backup/restore preflight intentionally avoids destructive restore into the active database.
- Supabase/cloud sync round-trip is outside Wave 16 and remains a later pilot-readiness gate.

## Final Gate

Wave 16 is complete. The final full `.\scripts\verify_release.ps1` run passed on 2026-05-08, including `wails build`, and produced `build/bin/AsymmFlow.exe`.
