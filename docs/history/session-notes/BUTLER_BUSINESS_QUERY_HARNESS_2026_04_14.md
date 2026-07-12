# Butler Business Query Harness

## Purpose

This harness is the reusable validation layer for Butler business usability.
It is designed to catch regressions in:

- intent routing
- entity extraction and entity resolution
- grounded fast-path answers
- cross-module context coverage
- prompt readiness for real business questions

The automated regression lives in:

- `/Users/developer/projects/asymmflow/butler_prompt_harness_test.go`
- `/Users/developer/projects/asymmflow/butler_business_usability_test.go`

## Automated Coverage

The current automated harness verifies:

- work-domain questions for collaborative tasks and notifications
- customer questions for invoices, offers, sold line items, and notes
- supplier questions for payment history and supplier context
- finance questions for revenue projection and cash outlook
- quotation-readiness questions using `quotation_precheck`
- year/period access checks using `business_year_summary`
- cross-module coverage for `database_access`, `work_data`, and `employee_context`

## Manual Prompt Bank

Use these prompts as the ongoing business usability question bank.
When a real client prompt appears in production, add it here and convert it into an automated regression where possible.

- How many tasks are assigned to Jamie Wong right now?
- What notifications does Jamie Wong have?
- Which tasks are blocked this week?
- Who has the heaviest workload today?

- What notes do we have for National Petroleum Co.?
- Show me National Petroleum Co. invoices this quarter.
- Show me National Petroleum Co. invoices.
- What have we sold to National Petroleum Co.?

- Show me National Petroleum Co. offers this year.
- Show me National Petroleum Co. offers.
- Which opportunities for National Petroleum Co. are still open?
- Can we draft a quotation for National Petroleum Co. calibration service?

- Tell me about Rhine Instruments payment history.
- What did we buy from Rhine Instruments?
- Which supplier has the worst lead time?
- Are there any active supplier issues for Rhine Instruments?

- What is our cash outlook this year?
- Give me the revenue projection.
- Which overdue customers need attention today?
- Who are our slowest paying customers?

- What bank statement notes do we have for National Petroleum Co.?
- Which receipts are still unreconciled?
- What customer payments were recorded this month?
- What supplier payments went out this month?

- What service follow-ups are due?
- What can we close this month?
- Which offers are expiring soon?
- What is our pipeline win rate?

- Do you have access to 2026 data?
- What data do we have for 2025?
- Which divisions are active in finance right now?
- What changed most recently in the business records?

## Extension Rules

- Prefer deterministic regressions first: intent, entity resolution, fast paths, context coverage.
- Only rely on live LLM evaluation when deterministic coverage cannot represent the case.
- For every production miss, add:
- the original user wording
- the expected grounded source of truth
- the nearest deterministic assertion we can automate
- If a prompt mentions a person, customer, supplier, bank statement, or specific year, make sure the harness seeds that entity explicitly.

## Suggested Run Commands

- `go test -run 'TestButlerBusinessHarness_.*|TestClassifyIntent_WorkQueryUsesEmployeeReference|TestResolveBestEntityReference_SupplierAndEmployee|TestTryGroundedWorkFastPath_TaskSummaryByEmployee|TestBuildFullContext_IncludesCrossModuleDatabaseAndWorkAccess' -v .`
- `go test ./...`
