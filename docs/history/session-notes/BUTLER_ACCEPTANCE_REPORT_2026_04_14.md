# Butler Acceptance Report

- Date: 2026-04-14T11:12:27+05:30
- Source DB: `/Users/developer/projects/asymmflow/asymmflow.db`
- Working copy: `/var/folders/q3/gmsc2s_j6vx3frjg1zxptw3m0000gn/T/TestManualButlerBusinessAcceptance482357640/001/butler_acceptance.db`

## Results

- Overall result: 10/10 prompts passed through the grounded Butler pipeline against a temp copy of the live `asymmflow.db`.
- Important nuance: several prompts returned factual "no records found" answers for the current live data snapshot, which is a correct grounded outcome and not a Butler failure.
- Positive live-data hits in this pass included Jamie Wong notifications, Rhine Instruments supplier payment history, Rhine Instruments purchase history, and revenue projection.

### jamie_tasks

- Prompt: How many tasks are assigned to Jamie Wong right now?
- Status: PASS
- Response excerpt: Jamie Wong currently has 0 active task(s), 0 blocked task(s), and 0 overdue task(s). I also found 1 unread notification(s).

### jamie_notifications

- Prompt: What notifications does Jamie Wong have?
- Status: PASS
- Response excerpt: Jamie Wong currently has 0 active task(s), 0 blocked task(s), and 0 overdue task(s). I also found 1 unread notification(s).  Recent notifications: - Task reassigned | unread | 2026-04-08

### national_petroleum_notes

- Prompt: What notes do we have for National Petroleum Co.?
- Status: PASS
- Response excerpt: I checked notes for National Petroleum Co. group and found no recorded customer notes.

### national_petroleum_invoices_quarter

- Prompt: Show me National Petroleum Co. invoices this quarter
- Status: PASS
- Response excerpt: I checked Q2 2026 invoices for National Petroleum Co. group and found no recorded invoices in Q2 2026.  Account handlers on record: none found in current records.

### national_petroleum_line_items

- Prompt: What have we sold to National Petroleum Co.?
- Status: PASS
- Response excerpt: I checked sold line items for National Petroleum Co. group but found no invoice line-item records in current data.

### national_petroleum_offers_year

- Prompt: Show me National Petroleum Co. offers this year
- Status: PASS
- Response excerpt: I checked offers for National Petroleum Co. group in 2026 and found no recorded offers.  Account handlers on record: none found in current records.

### rhine_payment_history

- Prompt: Tell me about Rhine Instruments payment history
- Status: PASS
- Response excerpt: Here is the supplier payment history for Rhine Instruments.  Recorded supplier invoices: 76 totaling 450195.883 BHD Pending supplier invoices: 22 totaling 207249.692 BHD Recent supplier payments totaled 192735.780 BHD in th...

### rhine_purchase_history

- Prompt: What did we buy from Rhine Instruments?
- Status: PASS
- Response excerpt: Here is what we have bought from Rhine Instruments.  Recorded supplier invoices: 12 totaling 60633.978 BHD  Recent supplier invoices: - 6017272999 | 2024-12-11 | 10919.784 BHD | Paid - 163-22-DUP-3 | 2024-11-12 | 61.050 BHD...

### rhine_issue_history

- Prompt: Are there any active supplier issues for Rhine Instruments?
- Status: PASS
- Response excerpt: I checked supplier issues for Rhine Instruments and found no active issue records.

### revenue_projection

- Prompt: Give me the revenue projection
- Status: PASS
- Response excerpt: Latest revenue projection (grounded from current records):  - Last full year (2025) revenue: 760154.809 BHD - Current year (2026) YTD billed: 0.000 BHD - Active offer pipeline: 1569374.338 BHD  Projection scenarios: - Ru...
