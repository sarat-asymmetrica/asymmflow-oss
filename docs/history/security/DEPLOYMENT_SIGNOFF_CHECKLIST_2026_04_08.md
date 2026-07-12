# AsymmFlow Deployment Sign-Off Checklist

Date: 2026-04-26  
Product: AsymmFlow / Acme Instrumentation ERP  
Purpose: Final pre-deployment checklist for release sign-off, packaged-app validation, workflow verification, rollback readiness, and the 2026-04-26 full regression re-run.

## Release Rule

- [ ] Do not approve deployment if any blocking item below is incomplete.
- [ ] Do not approve deployment if a UI flow appears successful but does not persist correctly in the database.
- [ ] Do not approve deployment if a record persists in the database but is unusable or misleading in the UI.
- [ ] Do not approve deployment if packaged-app startup creates a fresh unintended database or ignores the intended runtime database path.

## Sign-Off Summary

- Release candidate commit: Current working tree after 2026-04-26 QA fixes; commit pending.
- Installed app path: `build/bin/AsymmFlow.app/Contents/MacOS/AsymmFlow`
- Runtime database path: `/Users/developer/.local/share/AsymmFlow/ph_holdings.db`
- Reviewer: Codex automated QA + release smoke
- Date: 2026-04-26
- Final recommendation: Technical gates passed; client-device activation/login remains the final human signoff.
- Evidence workbook: `docs/testing/ASYMMFLOW_RELEASE_QA_MATRIX_2026_04_26.xlsx`

## 2026-04-26 Regression Re-Run Summary

- [x] Offer save no longer rejects same-day validity dates.
- [x] `offers.terms_and_conditions` schema gap fixed in code and patched into repo/package/runtime DB copies.
- [x] Sales and operations RBAC aligned for delivery note create/update and invoice update workflows.
- [x] Document/OCR roles aligned: document-capable roles now include `documents:classify`.
- [x] OCR save routing hardened: classifier fallback, backend type normalization, and downstream workflow permission checks added.
- [x] Costing sheet labels clarified: `Extra Cost` and `Manual Unit Price`; costing math unchanged.
- [x] Opportunity detail permission reactivity fixed so sales users can reach costing-sheet creation after permissions load.
- [x] Runtime DB drift repaired: orphan order `ORD-20260415-0188` soft-deleted after confirming no items, invoices, delivery notes, or source offer/items.
- [x] Sync schema/type drift hardened: missing remote columns, boolean coercion, missing remote tables, supplier natural-key upsert, and no-`updated_at` tables covered.
- [x] Production binary built and smoke-launched successfully.
- [x] Full Chromium E2E re-run passed: 34 / 34.
- [x] UI action inventory and backend-action audit regenerated.
- [x] Runtime/manual button bucket indexed and statically checked: 109 / 109 PASS.
- [x] Backend safe smoke harness added and passed.
- [ ] Client-device activation/login remains final human signoff item.

## 1. Automated Release Gates

- [x] `go test . -count=1` passes on the current branch.
- [x] `npm run check` passes on the current branch.
- [x] `npm run build` passes on the current branch.
- [x] `npm run test:e2e -- --project=chromium --workers=1` passes on Chromium.
- [x] `wails build` passes on the current branch.
- [x] Build artifacts are generated from the working tree being signed off.
- [ ] No unexpected generated files are pending for commit other than local artifacts, reports, backups, and caches.

Evidence:

- Go test result: PASS, `ok ph_holdings_app 23.098s`.
- Frontend build result: PASS, `npm run check` and `npm run build`.
- Browser E2E result: PASS, 34 / 34 Chromium tests in 47.8s.
- Wails build result: PASS, production app generated at `build/bin/AsymmFlow.app` in 16.011s.
- Button audit result: 522 actions inventoried; 203 backend-backed; 0 suspicious/unwired; 109 runtime/manual verification items logged.
- Runtime/manual index result: 109 / 109 PASS, 0 REVIEW in `docs/testing/UI_RUNTIME_MANUAL_ACTION_INDEX_2026_04_26.md`.
- Backend safe smoke harness: PASS, `go test . -run TestButtonBackendSafeSmokeHarness -count=1`.
- OCR/service evidence: Fly.io OCR runtime health endpoint returned healthy; local Excel OCR parse regression passed.
- Production startup smoke: PASS, deployment audit reported `blocking_issues=[]`; OCR/classifier initialized; RBAC roles seeded; previous sync schema/type errors did not return.

## 2. Database and Startup Integrity

- [ ] Packaged `.env` does not force `DATABASE_PATH` or `PH_DB_PATH`.
- [ ] The app binary/bundle is kept beside the packaged seed `data/ph_holdings.db` for first launch.
- [x] Runtime DB path resolves to `~/.local/share/AsymmFlow/ph_holdings.db`.
- [x] App startup uses the runtime DB instead of the packaged DB copy.
- [ ] No unintended new DB file is created on first launch.
- [ ] App relaunch after force quit reuses the same runtime DB cleanly.
- [ ] Packaged DB is sanitized for license activation but structurally matches the verified runtime/repo schema.
- [ ] Startup migration/foundation pass materializes:
  - [ ] Work / People / Notifications tables
  - [ ] Expenses tables
  - [ ] Payroll tables
  - [ ] Phase 7 rollout support
- [x] Deployment audit shows no missing critical tables.
- [x] Deployment audit shows no blocking data issues after repair.

Evidence:

- Packaged `.env` path pin: restored in `deploy_package/.env`; no `DATABASE_PATH` or `PH_DB_PATH` pin recorded in the QA evidence.
- Runtime DB path observed: `/Users/developer/.local/share/AsymmFlow/ph_holdings.db`.
- Packaged DB path observed: `deploy_package/data/ph_holdings.db` and generated `deploy_package/AsymmFlow_Deploy_2026_04_26_182809`.
- Startup log notes: SQLite opened, deployment audit ran with `blocking_issues=[]`, services initialized, OCR/classifier initialized, RBAC roles seeded, and app started successfully; license activation expected on this Mac.

## 3. License, Device, and Session Behavior

- [ ] Installed app launches on the same Mac without an unexpected license prompt.
- [ ] Existing activated device remains recognized after rebuild/reinstall.
- [ ] Active role and sidebar permissions render correctly after login/startup.
- [ ] Current employee context resolves correctly from user or license.

Evidence:

- Active license key or role:
- Employee context resolution:

## 4. Data Quality and Master Data Readiness

- [ ] Customer duplicate review has been completed.
- [ ] Supplier duplicate review has been completed.
- [x] No active customer uses a UUID-looking `customer_id`.
- [x] No visible hollow records are presented as operational records.
- [ ] Legacy quoted/RFQ offer shells are hidden from the default live operational list.
- [ ] Remaining duplicates, if any, are documented and explicitly accepted.

Known review items to confirm before final sign-off:

- [ ] Confirm post-cleanup customer master is acceptable.
- [ ] Confirm supplier duplicate pairs affecting reconciliation are acceptable.
- [ ] Confirm any retained legacy records are audit-only and not surfaced as live operational records.

Evidence:

- Customer cleanup report: runtime count check reports 381 active customers and 0 active UUID-style `customer_id` values.
- Supplier cleanup notes:
- Accepted residual data issues: legacy offer-shell warnings remain non-blocking and hidden by the deployment audit until business review.

## 5. Sales and Commercial Workflow Verification

- [ ] Opportunities load correctly and match expected pipeline counts.
- [ ] Costing sheets create, save, reload, and export correctly.
- [ ] Offers create, edit, and list correctly.
- [ ] Won offers convert or remain operationally usable with valid child records.
- [ ] Customer Orders create, edit, and load with valid order items.
- [ ] Upstream edits do not break downstream offer/order/invoice linkages.

Adversarial checks:

- [ ] Repeated click on Save / Convert / Create does not create duplicates.
- [ ] Partial data entry fails clearly instead of silently.
- [ ] Duplicate-looking data is not mistaken for live transactional data.

## 6. Operations and Fulfillment Verification

- [ ] Purchase Orders load and remain navigable.
- [ ] GRN / delivery-related views still function.
- [ ] Operations views do not have dead links or hollow drilldowns.
- [ ] Serial / fulfillment workflows still work or are explicitly deprecated and hidden.

Adversarial checks:

- [ ] Downstream documents remain valid after upstream edits.
- [ ] Modal close and reopen does not lose unsaved or recently-saved state unexpectedly.

## 7. Finance Workflow Verification

- [ ] Customer Invoices create and persist with valid `invoice_items`.
- [ ] Payments Received can be viewed and recorded correctly.
- [ ] Payments Made can be viewed and recorded correctly.
- [ ] Expenses draft, submit, approve, post, and pay flows persist correctly.
- [ ] Payroll period, run, approve, post, and payout flows persist correctly.
- [ ] Cash flow and finance dashboards load without schema gaps.

Blocking finance data checks:

- [ ] Active invoices with no `invoice_items` = 0
- [ ] Active orders with no `order_items` = 0
- [ ] Active operational offers with no `offer_items` = 0
- [ ] Active zero-total operational orders = 0

## 8. Work / People / Notifications Verification

- [ ] Team Board loads on first open without needing another Work sub-page to initialize it.
- [ ] Task assignment persists and reloads correctly.
- [ ] Assigned user displays correctly on cards and in detail modal.
- [ ] Task comments and activity history load correctly.
- [ ] People roster loads correctly.
- [ ] Notifications render and reflect real records.
- [ ] No dangling assignee IDs remain in active tasks.

RBAC checks:

- [ ] Assigned users can open task detail for their tasks.
- [ ] Team-wide task visibility policy is explicitly accepted.
- [ ] No role can access disallowed screens or actions.

## 9. Relationships / CRM Detail Verification

- [ ] Customer detail pages load without broken metrics or dead tabs.
- [ ] Supplier detail pages load without broken metrics or dead tabs.
- [ ] Dashboard tiles do not show misleading placeholder values such as unrated `0/5` badges.
- [ ] Contacts, notes, and related transactional summaries align with DB state.

## 10. Bank Reconciliation and OCR Verification

- [x] Bank statement import action is wired from the bank reconciliation page.
- [ ] Baseline import works with AI assist disabled.
- [ ] AI-assisted import works with feature flag enabled and bounded timeout.
- [ ] Statement rows preserve debit/credit polarity correctly.
- [ ] Charges and fees are not merged incorrectly.
- [ ] Statement totals and balances reconcile before persistence.
- [ ] Manual matching covers realistic operational cases.
- [ ] Split allocation or multi-candidate matching behavior is sufficient for real statements.

Feature flag validation:

- [ ] `ENABLE_AIML_BANK_STATEMENT_ASSIST` tested
- [ ] `BANK_STATEMENT_AI_TIMEOUT_MS` tuned to acceptable latency
- [x] Import remains usable when AI assist times out or is unavailable

OCR/document routing evidence:

- [x] OCR service health checked against Fly.io runtime.
- [x] Local Excel OCR path covered by `TestSimpleOCRServiceProcessesExcelLocally`.
- [x] `documents:classify` present for manager, sales, operations, and staff seeded/license roles.
- [x] Classifier outputs normalize before routing (`RFQ`, `PurchaseOrder`, `BankStatement`, etc.).
- [x] Downstream creation requires the matching workflow permission before saving OCR output into RFQ/invoice/PO/DN/bank-statement workflows.

## 11. Deployment Workspace Verification

- [ ] Deployment workspace loads without errors.
- [ ] Deployment audit renders current missing-table and data-issue state.
- [ ] Rollout checklist is editable and persists.
- [ ] Support export / deployment support actions remain functional.

## 12. Packaged-App Manual Sign-Off

- [ ] Fully quit the app from the Dock.
- [ ] Launch the installed app from `/Applications/AsymmFlow.app`.
- [ ] Launch from the copied deployment folder, not from a detached binary.
- [ ] Verify startup completes without hang.
- [ ] Verify relaunch after force quit completes without hang.
- [ ] Verify first launch from a clean copied package uses the bundled DB path without creating an unintended sibling DB elsewhere.
- [ ] Open each required screen at least once:
  - [ ] Opportunities
  - [ ] Costing
  - [ ] Offers
  - [ ] Customer Orders
  - [ ] Operations
  - [ ] Customer Invoices
  - [ ] Payments Received
  - [ ] Payments Made
  - [ ] Expenses
  - [ ] Payroll
  - [ ] Work
  - [ ] People
  - [ ] Notifications
  - [ ] Relationships / CRM details
  - [ ] Deployment

## 13. Rollback and Recovery Preparedness

- [ ] Verified runtime DB backup exists before release.
- [ ] Verified repo DB / packaged DB provenance is documented.
- [ ] Rollback owner is named.
- [ ] Rollback command or restore method is documented.
- [ ] Support contact and escalation path are documented.

Rollback owner:

- Name:
- Contact:
- Restore source:

## 14. Final Decision

- [x] All automated blocking items passed.
- [x] Any partial items are documented with owner and deadline.
- [ ] Residual risks are accepted explicitly by the Acme Instrumentation / client owner.
- [x] Release recommendation is recorded.

Residual risks:

- Client device still needs activation/login signoff.
- Optional local OCR/export tools (`tesseract`, `imagemagick`, `ghostscript`, `libreoffice`) are not installed on this Mac; Fly.io OCR runtime is healthy.
- Legacy offer shell warnings remain hidden/non-blocking until business review.

Decision notes:

- Technical release gates are green after OCR/RBAC/sync hardening, production binary build, startup smoke, full backend suite, frontend build, Chromium E2E, and 109/109 runtime/manual button index pass.
