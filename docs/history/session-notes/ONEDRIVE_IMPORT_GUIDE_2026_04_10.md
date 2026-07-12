# Sam 2025 OneDrive Import Guide

This guide is for staging and reviewing `Offers 2025` data before any import is committed.

## What You Need

- The AsymmFlow source snapshot included in the import package
- Go installed on your machine
- Your local `Offers 2025` OneDrive folder path

## Safe Workflow

Run these in order from the source snapshot root.

### 1. Generate a staging report only

This scans the 2025 folder, parses costing files, detects supplier invoices and quotes, matches customers, and writes audit CSV/JSON files into `reports/`.

```bash
ONEDRIVE_IMPORT_ROOT="/path/to/Offers 2025" \
ONEDRIVE_IMPORT_YEAR=2025 \
go test . -run 'TestManualExportOneDriveSeed$' -v
```

Expected output:

- A folder like `reports/onedrive_seed_2025_YYYYMMDD_HHMMSS`
- `opportunities.csv`
- `costings.csv`
- `line_items.csv`
- `supplier_documents.csv`
- `db_audit.csv`
- `summary.json`

### 2. Review the staged data first

Check:

- unresolved customer matches
- duplicate-looking matches
- folders with no commercial data
- obvious title/customer mismatches
- supplier invoice and supplier quote rows in `supplier_documents.csv`

Do not import yet if the staging report still looks noisy.

### 3. Commit the import only after review

```bash
ONEDRIVE_IMPORT_ROOT="/path/to/Offers 2025" \
ONEDRIVE_IMPORT_YEAR=2025 \
ONEDRIVE_IMPORT_COMMIT=1 \
go test . -run 'TestManualImportOneDrive$' -v
```

This writes imported opportunities with source `2025_onedrive`.

### 4. Audit what landed in the DB

Replace the report path below with the exact staging report directory produced in step 1.

```bash
ONEDRIVE_IMPORT_YEAR=2025 \
ONEDRIVE_AUDIT_SOURCE_DIR="reports/onedrive_seed_2025_YYYYMMDD_HHMMSS" \
go test . -run 'TestManualAuditOneDriveImport$' -v
```

Expected output:

- A folder like `reports/onedrive_post_import_audit_YYYYMMDD_HHMMSS`
- `anomalies.csv`
- `spot_check_samples.csv`
- `summary.json`

## Supplier Invoice And Quote Staging

The staging pass also writes `supplier_documents.csv`. This is review-only data for now.

Supported file types:

- PDF
- RTF
- DOCX

Legacy `.doc` files should be converted to `.docx`, `.rtf`, or PDF before staging. Supplier documents are normally detected from execution-style folders and filenames containing supplier, invoice, inv, quote, sales order, Rhine Instruments, or execution signals.

## Important Notes

- The import is year-aware.
- 2025 imports are tagged as `2025_onedrive`.
- 2026 imports remain tagged as `2026_onedrive`.
- This flow is safer than importing directly because it gives you a staging report first.
- Do not run old legacy date-fix utilities against imported 2025/2026 data.
- `supplier_documents.csv` is staging-only and should be reviewed before any DB import work is built on top of it.

## Recommended Review Criteria

- Customer matching confidence
- Folder-number normalization
- Stage correctness: `Qualified`, `Quoted`, `Won`
- Grand totals versus parsed line-item totals
- Missing `offer_items` or hollow commercial records
- Supplier invoice totals, invoice dates, quote totals, payment terms, and item counts

## After This

Once the 2025 OneDrive pass is staged and reviewed, we can reconcile that against Tally imports and supplier commercial documents to produce a cleaner commercial history across 2025 and 2026.
