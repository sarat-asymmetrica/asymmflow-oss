# AsymmFlow v0.1 Release Roadmap

**Created**: 2026-05-08  
**Owner**: the maintainer + Codex  
**Purpose**: Living roadmap and execution log for moving AsymmFlow from refactor foundation to a shippable `0.1.0` pilot release.

---

## Product North Star

AsymmFlow v0.1 should not try to be all of ERPNext or Odoo. It should become a dependable **offline-first trading and distribution ERP nucleus**:

- Sales pipeline and quote-to-cash
- Purchasing and receive-to-pay
- Inventory and serial traceability
- Finance, reconciliation, and reporting
- AI-assisted document intake
- Compliance-ready localization and tax hooks

The strategic wedge is **offline-first emerging-market ERP with document intelligence and compliance modules**, not a generic broad-suite clone.

## Versioning Policy

AsymmFlow uses semantic versioning:

- `0.1.0-alpha.N`: internal autonomous sprint builds
- `0.1.0-beta.N`: feature-frozen pilot builds
- `0.1.0`: first documented pilot release
- `0.2.0`: next meaningful compatible feature bundle
- `1.0.0`: stable public product promise

Until `1.0.0`, minor versions may still include product-shaping changes, but releases must document migration and backup expectations.

## v0.1 Release Promise

`0.1.0` is ready when a new operator can install AsymmFlow, configure a company, import or seed data, and run the core trading/distribution loops without developer intervention.

## v0.1 Must-Prove Loops

| Loop | Required Flow |
|---|---|
| Quote-to-cash | RFQ/opportunity -> costing -> offer -> order -> delivery note -> invoice -> payment |
| Receive-to-pay | Purchase order -> GRN -> supplier invoice -> supplier payment |
| Bank/finance | Bank statement -> reconciliation -> finance reports |
| Master data | Customers/suppliers/products -> audit trail -> dedupe/cleanup |
| Compliance | Bahrain VAT baseline, India GST/IT modules available as additive engines |
| AI intake | Document/OCR intake routes into reviewable business objects |

## v0.1 Focus Areas

1. **Installability**: version manifest, Windows release bundle, predictable app/data paths, backup/restore validation, upgrade preflight.
2. **Accounting Confidence**: GL/journal posting spine, trial balance, AR/AP aging, period close and lock-date policy.
3. **Inventory Spine**: stock ledger, warehouse/bin balances, reserved/available/on-hand quantities, serial linkage, valuation snapshots.
4. **Onboarding**: setup wizard, import templates, demo tenant/sample data, new-company checklist.
5. **Operator UX**: fast search, consistent states, keyboard-friendly tables, export/print reliability, clear permissions.
6. **Release Discipline**: changelog, known issues, smoke checklist, one-command verification, support bundle export.

## Deferred Until After v0.1

- Manufacturing/BOM/MRP
- POS/eCommerce
- Marketplace/module store
- Full Odoo Studio-style customization
- Full Wails v3 runtime switch
- Full translation of every UI string
- Broad external API connector suite/Nango

---

## Wave Roadmap From 2026-05-08

| Wave | Name | Status | Objective | Exit Criteria |
|---|---|---|---|---|
| 16 | Release Engineering + Installer Spine | Complete | Make builds identifiable, packageable, and smoke-testable | Version manifest, build metadata endpoint, release checklist, package script, smoke test doc, backup/restore preflight |
| 17 | Accounting Posting Spine | Active | Convert finance events into auditable journal postings | Posting interfaces, invoice/payment/supplier posting previews, trial-balance gate |
| 18 | Inventory Ledger | Planned | Make stock movement truth explicit and queryable | Stock ledger, warehouse balances, reservations, valuation baseline |
| 19 | Setup + Import Wizard | Planned | Make new-company onboarding possible without developer intervention | Setup checklist, import templates, validation reports, demo tenant seed |
| 20 | v0.1 Hardening | Planned | Freeze and stabilize the pilot release | Full smoke test, known issues, changelog, beta tag candidate |
| 21 | Pilot Feedback Loop | Planned | Turn real operator feedback into patches and docs | Triage board, issue classes, `0.1.x` patch cadence |
| 22 | Expansion Modules | Planned | Resume market expansion after stable nucleus | Nango/external APIs, broader compliance, module packaging |

## Wave 16 Initial Ticket Plan

1. Release manifest and version metadata
2. Build metadata Wails endpoint and frontend display point
3. Release bundle script for Windows
4. Smoke test checklist and operator release checklist
5. Backup/restore preflight documentation
6. Full verification and Wave 16 progress audit

## Living Log

| Date | Entry |
|---|---|
| 2026-05-08 | Roadmap created after Wave 15. Direction pivoted from broad platform expansion to v0.1 productization: release engineering, accounting spine, inventory ledger, onboarding, hardening. |
| 2026-05-08 | Wave 16 started. Added build metadata, Settings display, Windows release packaging script, release checklist, verification script, and backup/restore preflight tooling. |
| 2026-05-08 | Wave 16 completed. Full `scripts/verify_release.ps1` gate passed, including `wails build`, producing `build/bin/AsymmFlow.exe`. |
| 2026-05-08 | Wave 17 started. Added additive accounting posting previews for customer/supplier invoices/payments and a posted-journal trial-balance gate. |
| 2026-05-08 | Wave 17 advanced. Added idempotent draft journal creation, source journal links, posting coverage report, and Accounting dashboard visibility. |
