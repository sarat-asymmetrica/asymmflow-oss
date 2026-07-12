# Stress Testing And Load Capability Dossier

Last updated: 2026-04-25

## Purpose

This dossier explains why AsymmFlow is designed to operate under real trading load and defines the tests used to prove capability in difficult business environments.

AsymmFlow is an offline-first desktop ERP. Its primary load profile is not public-web concurrency; it is high-volume local transaction entry, document generation, OCR/import work, sync to cloud, and multi-device merge behavior.

## Current Data Baseline

As of the latest project baseline in repository context:

| Entity | Approximate count |
|---|---:|
| Active customers | 304 |
| Suppliers | 34 |
| Pipeline opportunities | 504 |
| Customer invoices | 468 |
| Customer orders | 175 |
| Offers | 67 |
| Supplier invoices | 484 |
| Supplier payments | 601 |
| Customer contacts | 535 |
| Delivery notes | 155 |

The live dataset already represents a realistic Bahrain industrial trading workload across sales, procurement, finance, and delivery.

## Load-Relevant Architecture

| Capability | Implementation basis | Load benefit |
|---|---|---|
| Offline-first SQLite | Local `ph_holdings.db` is primary | App continues without internet |
| WAL mode | SQLite write-ahead logging | Better read/write behavior |
| Busy timeout | SQLite waits before lock failure | Reduces transient lock errors |
| Single DB connection | PRAGMA-safe connection model | Predictable SQLite behavior |
| Transactions | Payments, invoices, delivery, supplier payments, PO updates | Prevents race-condition corruption |
| Atomic backup | `VACUUM INTO` | Consistent backup without copying a live DB file |
| Integrity check | Startup `PRAGMA integrity_check` | Detects corruption early |
| Merge-only sync | Pull does not delete local data | Protects local records during cloud sync |
| Dependency-ordered sync list | Master data before dependent transaction tables | Avoids orphan sync order |
| Log rotation | 50 MB rotation policy | Prevents unbounded log growth |

## Stress Test Categories

| Category | What it proves |
|---|---|
| Build and compile | Source code is internally consistent |
| Unit/regression tests | Core functions and bug fixes still work |
| Transaction race tests | Payment/invoice/delivery logic resists duplicate/overrun cases |
| Data-volume tests | Large lists, search, pagination, PDF/export functions remain usable |
| Import tests | Tally/OCR/Excel/bank statement import paths handle real files |
| Sync tests | Local/remote records sync without deleting local data |
| Backup/recovery tests | Database backup can be created and integrity checked |
| UI build tests | Svelte/TypeScript build remains valid |
| E2E smoke tests | Key user workflows remain clickable and visible |

## Recommended Command Evidence

Run from repository root unless noted.

| Command | Purpose | Pass criteria |
|---|---|---|
| `go test ./...` | Backend unit/regression suite | All non-manual tests pass |
| `npm run build` from `frontend` or repo script if configured | Frontend production build | Build succeeds without type/bundle errors |
| `./frontend/node_modules/.bin/playwright test` | UI smoke/E2E tests | Critical workflow tests pass |
| `sqlite3 ph_holdings.db "PRAGMA integrity_check;"` | Database integrity | Returns `ok` |
| `du -sh .[!.]* *` | Folder-size audit | Large generated folders identified |

Manual tests that require env flags should be run only when intentionally executing operational maintenance, not during normal regression runs.

## Proposed Load Scenarios

| Scenario | Load | Expected result |
|---|---:|---|
| Customer list load | 500 customers | Search/filter remains responsive |
| Opportunity list load | 1,000 opportunities | Year/stage/search filters remain usable |
| Invoice list load | 2,000 invoices | Pagination remains stable |
| Payment race | 20 concurrent payment attempts on one invoice | Only valid total up to outstanding amount is accepted |
| Supplier payment race | 20 concurrent supplier payment attempts | No overpayment beyond outstanding AP |
| Delivery race | 10 simultaneous DN creations for same order line | Remaining quantity cannot go negative |
| Serial allocation | 100 serials allocated across multiple DNs | No serial is shipped twice |
| PO threshold | POs below/above 5,000 BHD | Above-threshold PO enters approval path |
| Bank import | 1,000 bank lines | Statement imports and auto-match completes |
| Sync | 36+ sync tables with mixed creates/updates | Push/pull completes with no local deletes |
| Backup | Live DB backup during normal app use | Backup file created and integrity passes |

## Acceptance Targets

| Metric | Target |
|---|---|
| Backend tests | 100% pass for non-manual tests |
| Frontend build | Successful production build |
| SQLite integrity | `ok` |
| Payment overrun | 0 accepted overpayment cases |
| Supplier payment overrun | 0 accepted overpayment cases |
| Delivery overrun | 0 negative remaining quantity cases |
| Serial duplication | 0 duplicate shipped/delivered serials |
| Sync deletion safety | 0 local records deleted by pull |
| Backup retention | New backup created and old backup pruning limited to policy |

## Evidence Log

| Date | Evidence | Result | Notes |
|---|---|---|---|
| 2026-04-25 | Documentation/code review | Created | Stress test plan tied to current code controls |
| 2026-04-25 | SQLite integrity check | Pass | `PRAGMA integrity_check` returned `ok` for `ph_holdings.db` |
| 2026-04-25 | Frontend production build | Pass | `npm run build` passed from `frontend` |
| 2026-04-25 | Backend regression suite | Caveat | `go test ./...` ran after rebuilding `frontend/dist`; root package failed on deployment-environment tests only |
| 2026-04-25 | Deployment DB reconciliation test | Fail | `TestDeploymentDBCopyReconciliationAndPackaging` expected 5 rows but found 381 in the live AppData DB |
| 2026-04-25 | Deployment package test | Fail | `TestPrepareDeploymentPackage` requires `build/bin/AsymmFlow.app`, which is absent after generated build artifacts were cleaned |

## Current Test Caveat

The cleanup removed generated build binaries to reduce workspace size. That is appropriate for source cleanup, but it means deployment-package tests that expect a prebuilt Wails app bundle must be run after `wails build`. The current failures do not indicate a frontend build failure or active SQLite corruption; they indicate deployment fixture/environment assumptions.

## Client-Facing Summary

AsymmFlow is built for a difficult desktop ERP environment: intermittent internet, local-first data entry, financial transaction safety, document-heavy operations, and multi-device cloud backup. The system relies on local SQLite for continuity, transaction wrappers for money and delivery updates, controlled sync to Supabase, atomic backup, and explicit business invariants for margin/payment risk. The stress plan above should be executed before major deployment and after structural changes.
