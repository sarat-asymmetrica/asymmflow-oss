# AsymmFlow v0.1 Release Checklist

**Purpose**: Repeatable release checklist for `0.1.x` pilot builds.

## Build Identity

- Version matches `pkg/infra/release/manifest.json`.
- Settings screen shows version, channel, and commit prefix.
- `scripts/build_release_windows.ps1` creates a versioned artifact folder and `.zip`.
- Dirty builds are allowed for internal alpha only; beta/release builds must be clean.

## Required Gates

```powershell
.\scripts\verify_release.ps1
```

For faster inner-loop checks before a final package build:

```powershell
.\scripts\verify_release.ps1 -SkipWailsBuild
```

## v0.1 Smoke Test

- Launch app from `build/bin/AsymmFlow.exe`.
- Verify app opens without startup errors.
- Open Settings and confirm displayed version/build metadata.
- Run manual backup from Settings and confirm backup info updates.
- Run backup/restore preflight:
  ```powershell
  .\scripts\preflight_backup_restore.ps1 -ActiveDb "$env:APPDATA\AsymmFlow\ph_holdings.db"
  ```
- Open Dashboard, Customers, Invoices, Purchase Orders, Delivery Notes, and Payments.
- Generate one existing invoice PDF.
- Export one existing report.
- Confirm app closes cleanly.

## Release Artifact Contents

- `AsymmFlow.exe`
- `manifest.json`
- `build-info.json`
- `V0_1_RELEASE_ROADMAP_2026_05_08.md`
- `BACKUP_RESTORE_PREFLIGHT_V0_1.md`

## Known Deferred Items

- Installer UX is not yet a signed MSI/NSIS installer.
- Backup/restore preflight is scripted; destructive restore remains manual by design.
- Full v0.1 business-loop smoke automation is scheduled for Wave 20.
