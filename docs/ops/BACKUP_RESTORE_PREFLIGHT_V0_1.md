# AsymmFlow v0.1 Backup/Restore Preflight

**Date**: 2026-05-08  
**Wave**: 16 - Release Engineering + Installer Spine  
**Purpose**: Verify that a pilot release has a usable active database, a valid backup, and a restore-tested backup copy before packaging or deployment.

## Scope

This preflight is intentionally conservative. It validates database health and copies a backup into a sandbox location. It does not replace the active production database.

## When To Run

- Before cutting any `0.1.x` release package.
- Before a client import, migration, cleanup, or seed refresh.
- After a support recovery, before cloud sync is restarted.
- Before deleting old backups or deployment bundles.

## Default Command

For the repository/dev database when a `backups` directory exists next to `ph_holdings.db`:

```powershell
.\scripts\preflight_backup_restore.ps1
```

For a packaged Windows install using the default app data location:

```powershell
.\scripts\preflight_backup_restore.ps1 -ActiveDb "$env:APPDATA\AsymmFlow\ph_holdings.db"
```

For an explicit backup file:

```powershell
.\scripts\preflight_backup_restore.ps1 `
  -ActiveDb "$env:APPDATA\AsymmFlow\ph_holdings.db" `
  -BackupPath "D:\AsymmFlowBackups\ph_holdings_20260508_120000.db"
```

## What The Script Checks

1. The active database file exists.
2. The newest matching backup exists in the backup directory, unless `-BackupPath` is supplied.
3. `PRAGMA integrity_check` returns `ok` for the active database.
4. `PRAGMA foreign_key_check` returns zero rows for the active database.
5. The same checks pass for the backup database.
6. The backup is copied to `release_artifacts\restore-preflight\restore-test.db`.
7. The sandbox restore copy passes the same SQLite checks.

## Manual Operator Flow

1. Launch AsymmFlow.
2. Open Settings and run a manual backup.
3. Close AsymmFlow cleanly.
4. Run the preflight script.
5. Record the active DB path, backup path, and script result in the release notes.
6. Only after the backup copy passes, continue with release packaging or migration.

## Restore Rules

- Never overwrite the active database until the damaged/original file has been copied aside.
- Never restore from a backup that fails either integrity or foreign-key checks.
- Restore with cloud sync paused if local correctness is uncertain.
- Restart sync only after Dashboard, Customers, Invoices, Orders, Payments, and Settings load correctly.
- Keep at least one known-good backup outside the repo and outside the client machine.

## Current Limitations

- The script verifies data-store health, not every business workflow.
- It does not test Supabase round-trip sync.
- It does not perform a destructive replacement of the active DB by design.
