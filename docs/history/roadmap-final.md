# AsymmFlow V5 Deployment Roadmap — FINAL

**Date**: 2026-03-17
**Seed File**: `canonical_seed.xlsx` (355 Parties, 500 Pipeline, 175 Orders, 468 Invoices)
**Goal**: Integrate canonical seed pipeline data, fix all P0-P3 audit findings, deploy production-ready
**Status**: Phases 1-5 COMPLETE. Phase 6 (verification/deploy) pending visual testing.

---

## Phase 1: Schema & Sync Fixes (Pre-Import Prerequisites)

**Why first**: Without these, imported data is lost (wrong columns) or never syncs (missing from sync list).

| # | Task | Files | Status |
|---|------|-------|--------|
| 1.1 | Expand `opportunities` table — add 13 columns (year, opp_number, folder_name, title, comment, eh_reference, loss_reason, order_date, delivery_date, payment_terms, owner_notes, source, customer_grade) | `database.go` | |
| 1.2 | Fix GORM AutoMigrate gap — ensure Opportunity Go struct matches new DDL | `database.go` | |
| 1.3 | Fix sync table name mismatch: `costing_sheets` → `costing_sheet_data`, `costing_items` → `costing_line_items` | `db_sync_service.go` | |
| 1.4 | Add `opportunities`, `rfq_data`, `costing_sheet_data`, `costing_line_items`, `offer_data`, `costing_history` to `dbSyncTables` | `db_sync_service.go` | |
| 1.5 | Fix `GetApplicationPaths()` nil dereference — add nil check in 3 PDF/CSV callers | `app.go`, `offer_pdf_service.go` | |

---

## Phase 2: Data Integrity & Customer Resolution

**Why second**: Clean foundation required before importing 500 pipeline entries.

| # | Task | Files | Status |
|---|------|-------|--------|
| 2.1 | Fix 3 orphaned customer_ids in existing opportunities (NGA, Horizon, Stonewell) | `ph_holdings.db` | |
| 2.2 | Build customer name resolution map (seed names → active customer_ids) — handle UUID/CUST-XXXX/short code formats | import script | |
| 2.3 | Normalize 6 data conflicts (seed vs DB on overlapping folder numbers) — seed is authoritative, upsert | import script | |
| 2.4 | Resolve duplicate folder number 148-25 (rename second to 148b-25) | import script | |
| 2.5 | Normalize owner display names to the canonical synthetic set | import script | |
| 2.6 | Extract client names from folder_name for 109 entries with no Client column | import script | |

---

## Phase 3: Pipeline Import (500 Opportunities)

**The main event**: Import canonical_seed.xlsx Pipeline sheet into `opportunities` table.

| # | Task | Files | Status |
|---|------|-------|--------|
| 3.1 | Write `ImportCanonicalSeed()` function — reads Pipeline sheet, resolves customers, maps statuses, upserts on folder_number | `app.go` | |
| 3.2 | Import 500 pipeline entries with full field mapping | runtime | |
| 3.3 | Import ~32 genuinely new customers from seed Parties (2025_excel source, not soft-deleted dupes) | runtime | |
| 3.4 | Enrich existing customers with seed grade/pay_grade data (NGA→B, Gulf Snipe→C) | runtime | |
| 3.5 | Verify import: count check, customer linkage, no orphans | runtime | |
| 3.6 | Push to Supabase — create opportunities table + sync | `db_sync_service.go`, Supabase | |

---

## Phase 4: Backend P0-P1 Fixes

| # | Sev | Task | Files |
|---|-----|------|-------|
| 4.1 | P1 | Fix `GenerateOfferPDF` VAT 0% bug — match ExportCostingToPDF smart fallback pattern | `offer_pdf_service.go` |
| 4.2 | P1 | Fix `generateOfferNumber()` double-BEGIN — remove inner `BEGIN EXCLUSIVE`, use GORM transaction only | `app.go` |
| 4.3 | P1 | Add RBAC to `BackfillOfferItemCostBreakdown()` and `RecalculateInvoiceItemCosts()` | `app.go` |
| 4.4 | P1 | Fix customer dropdown limit: 100 → 500 in both CostingSheetScreen and CustomersScreen | `CostingSheetScreen.svelte`, `CustomersScreen.svelte` |
| 4.5 | P1 | Resolve dual pipeline model — wire OpportunitiesScreen to read from `opportunities` table | `OpportunitiesScreen.svelte`, `app.go` |

---

## Phase 5: P2 Hardening

| # | Task | Files |
|---|------|-------|
| 5.1 | CSV header field escaping — wrap all header values with `escCSV()` | `app.go` |
| 5.2 | Server-side VatRate bounds check (0-100) | `app.go` |
| 5.3 | Server-side QuoteType whitelist validation | `app.go` |
| 5.4 | MobileNumber validation in `ValidateCustomerInput()` | `security_enhancements.go` |
| 5.5 | `GenerateOfferPDF` db nil check | `offer_pdf_service.go` |
| 5.6 | `SaveCostingAsOffer` transaction wrapping | `app.go` |
| 5.7 | Fix T&C VAT rate reactivity (update on vatRate change) | `CostingSheetScreen.svelte` |
| 5.8 | Remove Customer Code asterisk (not required) | `CustomersScreen.svelte` |
| 5.9 | Fix TRN validated as email | `security_enhancements.go` |
| 5.10 | Add RBAC to `GetBusinessVATRate()`, `RefreshToolsStatus()` | `app.go` |

---

## Phase 6: Verification & Deploy

| # | Task |
|---|------|
| 6.1 | `wails dev` — verify app launches clean |
| 6.2 | Test Phase 32 visual checklist (12 items from CLAUDE.md) |
| 6.3 | Test pipeline import — verify 500+ opportunities visible in UI |
| 6.4 | Test customer creation with new flow |
| 6.5 | Test PDF/CSV export with edge cases (0% VAT, commas in names) |
| 6.6 | Supabase sync verification — all new tables syncing |
| 6.7 | Build production binaries (Mac + Windows cross-compile) |
| 6.8 | Update CLAUDE.md with Phase 33 notes |

---

## Counts

| Phase | Items | Blocking? |
|-------|-------|-----------|
| Phase 1 | 5 | YES — import will fail without these |
| Phase 2 | 6 | YES — data integrity for import |
| Phase 3 | 6 | YES — the main deliverable |
| Phase 4 | 5 | YES — crash bugs + UX blockers |
| Phase 5 | 10 | NO — hardening, can ship without |
| Phase 6 | 8 | YES — verification before deploy |
| **Total** | **40** | |

---

## Data Summary

```
Orders:   175/175 exact match   — NO IMPORT NEEDED
Invoices: 468/468 exact match   — NO IMPORT NEEDED
Margins:  468/468 exact match   — NO IMPORT NEEDED
Parties:  266/355 matched       — ~32 genuinely new customers to create
Pipeline: 29/500 in DB          — 471 NEW opportunities to import (+ 29 upserts)
```

---

**Om Lokah Samastah Sukhino Bhavantu**
