# Data Resilience And Security Dossier

Last updated: 2026-04-25

## Purpose

This document summarizes the application's resilience and security posture for operational and client assurance.

## Resilience Controls

| Control | Description | Business value |
|---|---|---|
| Offline-first database | SQLite is the primary data store | Work continues without internet |
| Cloud sync | Supabase sync for multi-device/cloud backup | Shared state and recovery option |
| Merge-only pull | Pull does not delete local records | Reduces sync-loss risk |
| Backup on startup/manual | Atomic SQLite backup using `VACUUM INTO` | Recoverable restore point |
| Backup retention | Keeps recent backup set | Controls disk growth |
| Integrity check | SQLite integrity check on startup | Detects DB corruption |
| Graceful shutdown | Sync goroutines stop before DB/log close | Reduces shutdown corruption |
| Log rotation | Rotates large debug log files | Prevents unbounded disk usage |
| WAL and busy timeout | SQLite tuning for desktop use | Better local reliability |

## Security Controls

| Control | Description |
|---|---|
| License keys | Device-bound role activation by key prefix |
| Role permissions | Screen and API permissions by role |
| Admin wildcard | Full access restricted to admin/developer roles |
| Backend `requirePermission` | Sensitive APIs check permissions server-side |
| Password hashing | Bcrypt helper for password auth paths |
| OAuth PKCE | OAuth integration uses PKCE callback flow where configured |
| Token cache permissions | Token cache is written with owner-only permissions where used |
| Field crypto / HMAC | Document hashes support integrity checks |
| Log sanitization | Sensitive strings are sanitized in logs |
| Secure backup permissions | Backup files are chmod-restricted where supported |

## Financial Integrity Controls

| Area | Control |
|---|---|
| Customer invoice creation | Credit blocked and credit limit checks occur inside transaction |
| Customer payments | Outstanding balance checked inside transaction |
| Supplier payments | Outstanding AP checked inside transaction |
| Supplier invoice approval | Creation/update/approval are separate workflow steps |
| Purchase orders | High-value threshold requires approval path |
| Delivery notes | Quantities and serial updates are transactional |
| Credit notes | Credit note lifecycle is Draft, Issued, Applied |
| VAT export | Excludes non-posted/cancelled categories according to finance rules |

## Data Protection Recommendations

| Recommendation | Reason |
|---|---|
| Keep `.env` outside shared deployment bundles | Prevent credential leakage |
| Do not email raw database backups | Contains business/financial/customer data |
| Store backups in encrypted disk/user profile | Protects confidential ERP data |
| Restrict admin license keys | Admin can access all data and settings |
| Rotate cloud/API keys after handoff | Reduces support-period exposure |
| Review cleanup manifest before deletion | Avoid accidental loss of active package/database |
| Keep one current deployment package outside repo | Allows client reinstall without keeping many old packages in source tree |

## Operational Recovery Procedure

1. Stop the application.
2. Copy the newest known-good backup to a safe temporary location.
3. Verify backup with `sqlite3 backup.db "PRAGMA integrity_check;"`.
4. Replace the damaged local DB only after preserving a copy of it.
5. Start the application.
6. Validate Dashboard, Customers, Invoices, Orders, Supplier Invoices, Payments, and Sync status.
7. Run manual sync only after local data is confirmed correct.

## Risk Register

| Risk | Mitigation |
|---|---|
| Local disk failure | External/off-machine backup and Supabase sync |
| Internet outage | Offline-first operation |
| Sync credentials invalid | Local work continues; admin fixes credentials later |
| Duplicate payment entry | Transaction validation and idempotency controls |
| Over-delivery | Transactional quantity validation |
| Admin key misuse | Limit admin keys and audit assignment |
| Folder bloat | Scheduled cleanup of generated packages/caches/backups |

