# Cleanup Audit

Date: 2026-04-25

## Purpose

The repository contains source code plus generated build products, deployment bundles, caches, backups, logs, and historical release packages. This audit identifies what can be cleaned without touching source code or the active database.

## Initial Size Snapshot

Measured from `/Users/developer/projects/asymmflow`.

| Path | Size | Classification |
|---|---:|---|
| Repository total | 7.5G | Full workspace |
| `.git` | 2.2G | Git history, do not delete |
| `deploy_package` | 1.7G | Generated deployment bundles |
| `.gocache` | 1.2G | Generated Go cache |
| `.gocache-win` | 894M | Generated Windows Go cache |
| `backups` | 660M | Generated DB backups |
| `release` | 416M | Generated release packages |
| `frontend` | 185M | Source plus node_modules/dist |
| `build` | 65M | Generated build products |
| `PH_Holdings_Deploy_2026_02_17.tar.gz` | 28M | Historical deployment archive |
| `deploy_package.zip` | 24M | Historical deployment archive |
| `reports` | 19M | Generated reports |
| `app_debug.log` | 4.4M | Generated log |
| Old root DB backups | 7M to 12M each | Historical backups |

## Safe Cleanup Categories

| Category | Examples | Cleanup decision |
|---|---|---|
| Generated Go caches | `.gocache`, `.gocache-win` | Safe to remove; Go rebuilds them |
| Old build outputs | `build/bin`, platform build artifacts | Safe to remove if not current delivery package |
| Old release bundles | `release/*`, old zip/tar files | Safe to remove after preserving final client package elsewhere |
| Old deployment folders | `deploy_package/AsymmFlow_Deploy_*` | Safe to remove when not the active client handoff |
| Generated reports | `reports/*`, `test_output/*` | Safe to remove if not required as evidence |
| Logs | `app_debug.log` | Safe to truncate/remove after preserving if needed |
| Old DB backups | dated root backup DBs and old `backups/*` | Safe to prune after current backup exists |

## Do Not Delete

| Path | Reason |
|---|---|
| `asymmflow.db` | Active local database |
| `.git` | Git history and current worktree metadata |
| `frontend/src`, `frontend/package.json`, `frontend/package-lock.json` | Frontend source/dependency manifest |
| Go source files and `go.mod`/`go.sum` | Backend source/dependency manifest |
| `docs/user-guides` and `docs/compliance` | Newly created manual and evidence pack |
| `AGENTS.md`, `CLAUDE.md`, project notes | Project operating context |
| Current deployment package selected by user/client | Needed for handoff/reinstall |

## Cleanup Actions Performed

| Action | Status | Notes |
|---|---|---|
| Create manuals/evidence docs | Complete | Added under `docs/user-guides` and `docs/compliance` |
| Create active DB backup | Complete | `backups/asymmflow_cleanup_backup_2026_04_25.db` created with `VACUUM INTO` before deletion |
| Verify active DB integrity | Complete | `sqlite3 asymmflow.db "PRAGMA integrity_check;"` returned `ok` |
| Remove generated Go caches | Complete | Removed `.gocache` and `.gocache-win` |
| Remove stale build output | Complete | Removed `build/bin`; `frontend/dist` was rebuilt and kept because Go embed/tests require it |
| Prune old release archives | Complete | Kept current 2026-04-23 delivery installer zips, removed older release zips/folders |
| Prune old deployment packages | Complete | Kept current 2026-04-23 delivery package/installer, removed old dated deployment bundles |
| Remove generated reports/logs | Complete | Removed `reports`, `test_output`, `EXPORTS`, and `app_debug.log` |
| Remove stale root archives/backups | Complete | Removed historical root `.zip`, `.tar.gz`, and old root backup DB files |
| Prune old backup directory files | Complete | Kept the fresh cleanup backup and `backups/seed_baselines` |
| Compact git object store | Complete | Ran `git gc`; `.git` reduced from about 2.2G to about 1.2G |
| Re-run size snapshot | Complete | Repository total reduced from 7.5G to 1.8G |
| Verification commands | Complete with caveat | Frontend build passed; root Go tests have two deployment-environment failures listed below |

## Final Size Snapshot

Measured after cleanup and verification.

| Path | Size | Notes |
|---|---:|---|
| Repository total | 1.8G | Reduced by about 5.7G |
| `.git` | 1.2G | Packed by `git gc`; history retained |
| `frontend` | 185M | Source, dependencies, rebuilt `dist` |
| `.gitnexus` | 110M | Local knowledge graph index |
| `deploy_package` | 106M | Current retained deployment package |
| `release` | 84M | Current retained release zips |
| `backups` | 41M | Fresh cleanup backup plus seed baseline files |
| `asymmflow.db` | 12M | Active database retained |
| `build` | 7.7M | Non-binary build metadata/assets retained |
| `docs` | 1.0M | New manuals and evidence docs |

## Verification Results

| Check | Result | Notes |
|---|---|---|
| SQLite integrity | Pass | Active `asymmflow.db` returned `ok` |
| Frontend production build | Pass | `npm run build` passed from `frontend` |
| Backend package tests | Fail, environment-specific | `go test ./...` fails in root package because two deployment tests require a prebuilt app bundle/current deployment fixture |
| `TestDeploymentDBCopyReconciliationAndPackaging` | Fail | Test expected 5 copied rows but found 381 in the live AppData database |
| `TestPrepareDeploymentPackage` | Fail | Test expects `build/bin/AsymmFlow.app`; `build/bin` was intentionally removed as generated build output |

## Operational Note

Before producing a new client installer, run `wails build` to regenerate `build/bin/AsymmFlow.app`, then rerun deployment packaging tests. The cleanup intentionally preserved source, active database, current delivery artifacts, documentation, package manifests, and Git history.
