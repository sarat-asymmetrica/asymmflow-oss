# Administrator User Guide

Last updated: 2026-04-25

## Role Purpose

Administrators operate the full AsymmFlow environment. This includes license activation, user/device control, system settings, data imports, backup policy, sync health, AI keys, finance setup, and emergency support actions.

## First Start And License Activation

1. Open AsymmFlow.
2. If activation is required, enter the license key exactly as issued.
3. Use an admin key for system setup: `PH-ADM-*`.
4. Use the developer master key (value redacted) only for developer support when master-key access is intentionally enabled.
5. After activation, confirm the left sidebar shows the expected role and user display name.

What to enter:

| Field | Enter | Notes |
|---|---|---|
| License key | Full `PH-ADM-XXXXXX` key | Case-insensitive in practice, but enter exactly as issued |
| Device/user display name | The assigned employee name | This appears in user context and audit records where available |

## Daily Administrator Checks

1. Check the sync indicator in the sidebar.
2. Review Settings, Backup, and Sync health.
3. Review Notifications for pending approvals or delete requests.
4. Confirm Finance dashboard data for Acme Instrumentation and Beacon Controls.
5. Confirm no failed background sync or rollout operations remain unresolved.

## Settings

Open `Settings` from the sidebar.

Main sections and fields:

| Section | Field | What to enter |
|---|---|---|
| General | Company Name | Legal or display name used in app settings |
| General | Base Currency | Usually `BHD` |
| General | Interface Theme | User interface preference |
| Office | Enable Outlook Integration | Turn on only if Microsoft integration is configured |
| Office | Enable Excel Automations | Turn on where Excel export/import is used |
| Folders | Folder paths | Local folders for imports, exports, OCR, or watched documents |
| AI | Mistral API Key | Butler chat + OCR/document analysis key (single provider as of Wave 13) |
| GPU | Use GPU for Local Inference | Enable only on machines with supported GPU |
| Business | Default Margin (%) | Default commercial margin target |
| Business | VAT Rate (%) | Bahrain VAT rate used by finance/costing defaults |
| Currency | New Rate | BHD conversion rate, e.g. `0.376` for USD to BHD |
| Currency | Notes | Source of rate, e.g. central bank or finance instruction |
| Imports | Import Year | Year for Tally invoice/purchase imports |
| Reports | Report Year | Year for P&L or balance sheet generation |
| Supabase | URL, anon key, service key, DB host, port, DB name, password | Cloud sync credentials |
| Backup | Auto backup enabled | Whether startup/manual backup policy is active |
| Backup | Frequency | Days between scheduled backups |

Use `Save Settings` after changes. Use test buttons for AI and Supabase before assuming configuration is live.

## License And People Access

Use People Hub to maintain employee records and connect employee profiles to license keys where enabled.

Employee fields:

| Field | What to enter |
|---|---|
| Full name | Employee legal or working name |
| Preferred name | Short name used by team |
| Email | Work email |
| Phone | Mobile or office contact |
| Department | Sales, Operations, Finance, Admin, etc. |
| Job title | Actual job title |
| Manager | Reporting manager |
| Employment status | Active or inactive status |
| Start date | Employment start date |
| Emergency contact | Optional HR emergency contact |
| Notes | Responsibilities, coverage, or support context |

Access linking:

1. Select employee.
2. Select an available license key.
3. Click `Link Access`.
4. Confirm the employee appears with the intended role.

## Data Imports

Admin import controls sit under Settings.

| Action | Use When | Expected Result |
|---|---|---|
| Import All Tally Data | Initial or full finance import | Imports invoices, purchases, AR, supplier payments where configured |
| Import Tally Invoices | Customer invoice history import | Creates or matches customer invoice records |
| Import Tally Purchases | Supplier purchase history import | Creates or matches supplier invoice/purchase records |
| Import AR Defaulters | AR risk data update | Updates customer payment/risk profile |
| Import Supplier Payments | Supplier payment import | Creates supplier payment history records |

Before import, confirm the import file/source is correct and the import year is set. After import, review finance dashboards and reconciliation reports.

## Backup And Recovery

AsymmFlow uses local SQLite as the primary database. Backup behavior includes atomic SQLite `VACUUM INTO` backup, backup retention, and startup integrity checks.

Admin procedure:

1. Go to Settings.
2. Open Backup section.
3. Review last backup path/time.
4. Click manual backup before major imports, cleanup, or deployment.
5. Keep the newest production database and at least one known-good backup.

Do not delete `ph_holdings.db` or the newest deployment database when cleaning the folder.

## Cloud Sync

The sync system is merge-only and designed not to delete local data during pull. Admin controls include Supabase credential test, manual sync, sync status, and background interval settings where enabled.

Use manual sync when:

| Situation | Action |
|---|---|
| A second device needs latest records | Click sidebar sync if enabled |
| Import completed locally | Run sync after validating import |
| Sync status is yellow/gray | Test Supabase settings, then retry |
| Network is unstable | Continue local work; sync later |

## Admin Finance Controls

Admins can access all finance screens:

| Screen | Admin responsibilities |
|---|---|
| Financial Dashboard | Review P&L, balance sheet, ratios, company selector |
| Customer Invoices | Create/send invoices, generate PDFs, credit notes |
| Payments Received | Record/edit customer payments, verify outstanding balances |
| Payments Made | Record/edit supplier payments |
| Expenses | Create categories/vendors, approve and pay expenses |
| Payroll | Maintain compensation profiles, periods, runs, approvals, payouts |
| Bank Recon | Import statements, match lines, finalize reconciliation |

Financial records should be corrected through the matching workflow wherever possible. Avoid editing outstanding values manually except for controlled correction work.

## Deployment And Cleanup

Admin cleanup rules:

1. Keep source code, current database, docs, and current release evidence.
2. Remove generated caches, old deployment folders, old release archives, old build products, logs, and old DB backups only after a current backup exists.
3. Record cleanup in `docs/cleanup`.
4. Run at least `go test ./...` and `npm run build` when dependencies are present after cleanup-sensitive work.

