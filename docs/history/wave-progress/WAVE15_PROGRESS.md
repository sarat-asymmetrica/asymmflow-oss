# Wave 15 Progress Audit - i18n + Compliance

**Date**: 2026-05-07  
**Spec**: `CODEX_WAVE15_HANDOFF.md`  
**Commit range**: `3715ca7..9951642`

## Commit Table

| Ticket | Commit | Result |
|---|---|---|
| 1 | `3715ca7` | Embedded backend i18n package with 5 locale catalogs and tests |
| 3 | `b54ea7e` | Additive compliance engine interface, registry, DTOs, tests |
| 4 | `c3cf3dc` | Bahrain VAT engine: standard, exempt, zero-rated, invoice validation |
| 5 | `4ee94e1` | India GST engine: HSN rates, CGST/SGST, IGST, GSTIN validation |
| 6 | `7d4e34a` | India income tax calculator: old/new regimes, deductions, rebate, surcharge, cess |
| 2 | `2f099e7` | Frontend i18n store, Wails translations endpoint, proof UI wiring |
| 7 | `c23f0e2` | Async compliance hooks for finance events with validation trail |
| 8 | `775f9b8` | Compliance dashboard ViewModel and `InfraService.GetComplianceDashboard` |
| 9 | `9951642` | Full integration verification checkpoint |

## i18n Metrics

- Languages: 5 (`en`, `ar`, `hi`, `fr`, `es`)
- Message catalogs: 5 embedded JSON files under `pkg/i18n/messages/`
- Message keys: 67 English keys, with translated catalogs for Arabic, Hindi, French, and Spanish
- RTL support: Arabic sets `document.documentElement.dir = "rtl"`
- Frontend proof wiring: `EnterpriseSidebar`, `DashboardScreen`, `InvoicesScreen`, `SettingsScreen`
- Backend endpoint: `InfraService.GetTranslations(locale string)` plus `GetAvailableLocales()`

## Compliance Modules

- Core package: `pkg/compliance`
- Registry model: one engine per jurisdiction, additive and finance-logic neutral
- Bahrain VAT: 10% standard rate, zero-rated exports/international transport, exempt categories, BHD rounding to 3 decimals
- India GST: 0%, 5%, 12%, 18%, 28% slabs, 20 HSN/SAC seed mappings, intra-state CGST+SGST, inter-state IGST
- India income tax: FY 2025-26 old/new regime comparison, standard deductions, Section 87A rebate, surcharge, 4% health and education cess
- Test coverage added: 23 dedicated tests across i18n and compliance packages

## Source Sanity Checks

- Bahrain official portal confirms 10% VAT from 1 January 2022: https://bh.bh/new/en/business-vat_en.html
- CBIC GST overview confirms IGST on inter-state supplies and CGST/SGST concepts: https://cbic-gst.gov.in/about-gst.html
- CBIC rates portal provides official GST goods/services rate structure: https://cbic-gst.gov.in/gst-goods-services-rates.html
- Income Tax Department AY 2026-27 help confirms old/new slabs, 12L new-regime rebate condition, surcharge bands, and 4% cess: https://www.incometax.gov.in/iec/foportal/help/individual/return-applicable-1

## Integration Results

| Gate | Result |
|---|---|
| `go build ./...` | Passed |
| `go test ./... -count=1 -timeout 300s` | Passed |
| `cd frontend && npm run build` | Passed |
| `cd frontend && npm run check` | Passed with 0 errors, 13 existing warnings |
| `wails build` | Passed, built `build/bin/AsymmFlow.exe` |

## Verified Calculations

- Hindi translations load through embedded i18n and frontend Wails endpoint.
- Bahrain VAT: 1000 BHD standard goods produces 100 BHD VAT and 1100 BHD total.
- India GST: 1000 INR HSN `8481` intra-state produces CGST 90 + SGST 90.
- India GST: 1000 INR HSN `8481` inter-state produces IGST 180.
- India income tax: 10L comparison returns both old/new regimes and a recommendation.

## Issues And Deviations

- The i18n pass intentionally converted proof components only, per Ticket 2. Full string extraction across 186 components remains future work.
- Compliance hooks are additive and asynchronous. Existing finance services do not publish new rich compliance payloads yet, so the hook is ready for events without changing invoice behavior.
- `Registry` maps one engine per jurisdiction; India GST is the registered `IN` tax engine, while India income tax is a standalone calculator module.
- `wails build` still emits the known generated-bindings warning for an anonymous `{R1,R2,R3}` struct. It does not fail the build.

## Final Gate

Wave 15 is complete. The app remains on Wails v2 with the Wave 14 service architecture, now carrying embedded localization infrastructure and additive compliance modules for Bahrain and India.
