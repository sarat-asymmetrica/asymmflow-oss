# Archaeological Audit Report

Generated: 2026-05-05

Source spec: `docs/WAVE0_AUDIT.md`
Raw outputs: `docs/audit_results/`

## Executive Summary

Wave 0 completed all Phase 1-7 scans and saved raw outputs under `docs/audit_results/`. The frontend build passed, and `go build ./...` passed after rerunning outside the sandbox because the sandbox could not trim the Go cache. `go test ./...` ran to completion and failed with real project failures in the root package and `pkg/ocr/orchestrator`.

Size profile from `supplemental_loc_inventory.txt`:

| Metric | Count |
| --- | ---: |
| Go non-test LOC | 126,505 |
| Frontend `src` LOC | 86,932 |
| Go files scanned | 336 |
| Go non-test files scanned | 218 |
| Root Go files | 202 |
| Root test files | 73 |

Health score: 52/100.

Rationale: the app compiles and has visible domain depth, but test health is red, core logic is still concentrated in very large `*App` methods, coverage gaps are broad, and native OCR/desktop dependencies make the dependency surface hard to regenerate safely.

Top 5 risks:

1. Test suite is red: `go test ./...` fails in root and OCR orchestrator packages (`phase6A_test_results.txt`).
2. God-object and long-function concentration: 202 functions exceed 100 lines; `app.go` startup is 918 lines, and multiple PDF/export functions exceed 500 lines (`phase5B_long_functions.txt`).
3. Unique business logic is scattered across app, finance, OCR, Butler, reconciliation, and import files: 166 algorithm hits and 2,158 business-rule/BHD hits (`phase7_unique_algorithms.txt`, `phase7_unique_business_rules.txt`).
4. Dead/demo/manual operational files are mixed into the build tree: 10 build-ignored files, 13 manual tests, 8 demo files, and 19 orphan exported functions (`phase1*.txt`).
5. CGO/native dependency surface is broad: 84 matches across `go-sqlite3`, `go-fitz`, and `go-ole` (`phase5C_cgo_deps.txt`).

## Dead Code Map

### Delete Candidates

Build-ignored files from `phase1A_build_ignored_files.txt`:

- `benchmark_runner.go`
- `classifier_test_simple.go`
- `example_simple_ocr.go`
- `logger_demo.go`
- `scanner_demo.go`
- `test_classifier_standalone.go`
- `verify_password_security.go`
- `examples/arabic_rtl_demo.go`
- `examples/file_watcher_demo.go`
- `examples/generate_sample_invoice.go`

Demo files from `phase1C_demo_files.txt`:

- `business_invariants_demo.go`
- `demo_scenarios.go`
- `examples/arabic_rtl_demo.go`
- `examples/file_watcher_demo.go`
- `logger_demo.go`
- `pkg/ocr/octonion/demo.go`
- `scanner_demo.go`
- `unified_demo.go`

Recommendation: delete only after a separate wave confirms no client deployment or documentation flow depends on these files. Do not delete in Wave 0.

### Quarantine Candidates

Manual tests from `phase1B_manual_test_scripts.txt`: 13 files. These mutate databases, require local assets/env, or encode one-off operational workflows. They should move behind explicit build tags or a `manual/` quarantine path before normal `go test ./...` can become a reliable gate.

Notable examples:

- `manual_deployment_package_test.go`
- `manual_onedrive_import_test.go`
- `manual_onedrive_post_import_audit_test.go`
- `manual_supabase_schema_test.go`
- `manual_won_offer_repair_test.go`

### Keep Or Extract

Orphan exported functions from `phase1D_orphan_functions.txt`: 19. Some are likely real archaeological tooling or package factory functions rather than delete candidates. Review before removal:

- `GenerateArtifactC_ArchaeologyReport`
- `SaveArtifactA_WorkspaceIndex`
- `SaveArtifactB_EvidenceExtract`
- `SaveArtifactC_ArchaeologyReport`
- `ApplyLowRiskMasterDataCleanup`
- OCR `Example*` functions in `pkg/ocr/ksum`

Stale-file scan from `ph_holdings` history found no root Go files untouched before `2026-02-05` (`phase1E_stale_files_3months.txt` is empty).

## Domain Discovery

Temporal coupling from `phase2B_coupling_pairs.txt` did not cross the spec's "natural domain" threshold of count > 10. The highest pairs are still useful because they show where change pressure concentrates:

| Count | Pair |
| ---: | --- |
| 8 | `app.go` <-> `butler_ai.go` |
| 8 | `app.go` <-> `license_service.go` |
| 7 | `app.go` <-> `customer_invoice_service.go` |
| 7 | `app.go` <-> `supplier_invoice_service.go` |
| 7 | `butler_ai.go` <-> `supplier_invoice_service.go` |
| 6 | `app.go` <-> `database.go` |
| 6 | `app.go` <-> `db_sync_service.go` |
| 6 | `app.go` <-> `offer_pdf_service.go` |
| 6 | `app.go` <-> `payment_service.go` |
| 6 | `app.go` <-> `purchase_order_service.go` |

Interpretation:

- `app.go` is still the integration gravity well. It co-changes with AI, licensing, finance, sync, payment, and PDF services.
- Finance document workflows are a natural boundary, but currently mixed across invoice, supplier invoice, payment, offer PDF, and app methods.
- Banking reconciliation is a bounded domain with distinct matching/reconciliation files and high unique-logic value.
- OCR/document classification is tangled: it includes CGO, test fixture failures, classifier logic, Butler analysis, and several demo/example remnants.
- Delivery/serial lifecycle is a cleaner candidate than Butler or OCR: it is mostly in `delivery_note_service.go`, `serial_number_service.go`, `grn_service.go`, and invoice linkage.

## Hot Path Traces

### Offer -> Order -> Invoice -> Payment

From `phase3A_offer_order_invoice_payment.txt`:

- `app.go:6428` `CreateOffer`
- `app.go:17320` `CreateOfferDraftFromButler`
- `app.go:6589` `ConvertOfferToOrder`
- `customer_invoice_service.go:350` `CreateInvoiceFromOrder`
- `customer_invoice_service.go:452` `CreateInvoiceFromOrderWithDN`
- `customer_invoice_service.go:461` `CreateInvoiceFromDN`
- `customer_invoice_service.go:484` `CreateInvoiceWithOptions`
- `payment_service.go:28` `RecordPayment`
- `supplier_payment_service.go:19` `RecordSupplierPayment`

This path is business-critical and crosses `app.go`, customer invoicing, supplier invoicing, and payment services.

### Bank Statement Import -> Match -> Reconcile

From `phase3B_bank_import_match_reconcile.txt`:

- `bank_statement_ai_assist.go:66` `parseImportedPDFBankStatement`
- `bank_transaction_matcher.go:64` `AutoMatchBankLines`
- `bank_transaction_matcher.go:452` `ManualMatchLine`
- `bank_reconciliation_service.go:842` `FinalizeReconciliation`
- `book_bank_reconciliation_service.go:138` `CreateBookBankReconciliation`
- `book_bank_reconciliation_service.go:513` `AutoMatchDepositsToStatement`
- `book_bank_reconciliation_service.go:554` `AutoMatchChequesToStatement`
- `supplier_invoice_service.go:669` `PerformThreeWayMatch`

This is a good first domain extraction candidate because matching tolerances, cheque/deposit behavior, and reconciliation audit trails are unique and bounded.

### OCR -> Classify -> Extract

From `phase3C_ocr_classify_extract.txt`:

- `app.go:13009` `ProcessDocumentWithOCR`
- `app.go:14895` `ExtractRFQDocument`
- `app.go:14922` `ExtractInvoiceDocument`
- `app.go:14942` `ExtractQuotationDocument`
- `app.go:15012` `ProcessWithGoFitz`
- `butler_ai.go:6800` `AnalyzeDocumentWithButler`
- `document_classifier.go:612` `AIClassifyDocumentType`
- `document_classifier.go:765` `ClassifyDocument`
- `supplier_invoice_service.go:840` `CreateSupplierInvoiceFromOCR`

This path is tangled and dependency-heavy. Extract the classifiers and OCR engine interfaces before regenerating surrounding CRUD.

### Butler AI Query

From `phase3D_butler_ai_query.txt`:

- `butler_ai.go:412` `ChatWithButler`
- `chat_service.go:1553` `ChatWithButlerPersistent`
- `chat_service.go:1332` `GenerateDailyBriefing`
- `butler_intent_router.go:395` `calculateWeightedButlerPipeline`
- `butler_grounded_fastpath.go` grounded fast paths
- `butler_reports.go:44` `GenerateButlerReport`

Butler is not safe to regenerate wholesale. It contains prompt-routing, grounded fallback behavior, entity resolution, permissions context, and PH-specific memory/context logic.

### Delivery Note + Serial Numbers

From `phase3E_delivery_serial.txt`:

- `delivery_note_service.go:20` `CreateDeliveryNote`
- `delivery_note_service.go:837` `CreateDNFromOrder`
- `delivery_note_service.go:1493` `CreateDNWithSerials`
- `serial_number_service.go:25` `RegisterSerials`
- `serial_number_service.go:265` `assignSerialsToGRN`
- `serial_number_service.go:311` `allocateSerialsToDN`
- `serial_number_service.go:342` `markSerialsShipped`
- `serial_number_service.go:358` `markSerialsDelivered`
- `serial_number_service.go:392` `linkSerialsToInvoice`
- `grn_service.go:638` `ReceiveAgainstPOWithSerials`

This is a strong bounded domain for a later extraction wave: serial lifecycle logic is unique but more contained than finance or Butler.

## Dependency Graph

Internal import counts from `phase4A_internal_imports.txt`:

| Count | Internal package |
| ---: | --- |
| 8 | `pkg/ocr/fitz` |
| 4 | `pkg/ocr/octonion` |
| 3 | `microsoft_graph` |
| 2 | `pkg/runtime` |
| 2 | `pkg/ocr/ksum` |
| 2 | `pkg/ui_alchemy` |
| 2 | `pkg/survival_garden` |
| 2 | `pkg/ocr/predator` |
| 2 | `pkg/ocr` |

Most referenced structs from `phase4B_struct_usage.txt`:

| Count | Struct |
| ---: | --- |
| 3,567 | `Invoice` |
| 3,381 | `Order` |
| 2,439 | `Offer` |
| 1,717 | `Payment` |
| 637 | `Expense` |
| 629 | `Opportunity` |
| 283 | `DeliveryNote` |
| 278 | `BankStatement` |
| 234 | `PurchaseOrder` |
| 180 | `CustomerMaster` |
| 95 | `CreditNote` |
| 55 | `SerialNumber` |

Service coupling from `phase4C_service_coupling.txt` is weak as a fan-in metric because the Wave 0 script greps for service file basenames, not service types or method call sites. Treat the function counts as useful, but do not infer architectural independence from the zero fan-in results.

High-function-count services include:

- `delivery_note_service.go`: 29 functions
- `license_service.go`: 22 functions
- `purchase_order_service.go`: 20 functions
- `supplier_invoice_service.go`: 18 functions
- `cheque_register_service.go`: 17 functions
- `grn_service.go`: 17 functions
- `assets_service.go`: 16 functions

## Technical Debt

TODO/FIXME/HACK/XXX scan from `phase5A_tech_debt_todos.txt`: 15 hits.

Representative items:

- `geometry_bridge.go`: geometry classification placeholder
- `msg_parser.go`: missing document hash and extracted item JSON conversion
- `offer_followup_service.go`: completed-by user context deferred
- `sync_service.go`: remote file fetch/hash comparison not implemented
- `pkg/ocr/aimlapi.go`: concurrent processing/rate limiting deferred

Long functions from `phase5B_long_functions.txt`: 202 functions over 100 lines.

Largest functions:

| Lines | Function |
| ---: | --- |
| 918 | `app.go:271 startup` |
| 612 | `invoice_pdf_service.go:89 GenerateInvoicePDF` |
| 607 | `purchase_order_pdf_service.go:48 GeneratePurchaseOrderPDF` |
| 582 | `excel_template_generator.go:15 GenerateDataImportTemplate` |
| 557 | `app.go:17577 exportCostingToPDF` |
| 536 | `onedrive_import_service.go:1474 importSingleDeal` |
| 477 | `customer_invoice_service.go:484 CreateInvoiceWithOptions` |
| 331 | `chat_service.go:1553 ChatWithButlerPersistent` |
| 314 | `app.go:2421 GetDashboardStats` |
| 307 | `app.go:14131 routeToBankStatement` |

CGO/native dependency scan from `phase5C_cgo_deps.txt`: 84 matches.

Key dependencies:

- `github.com/mattn/go-sqlite3` in `config.go` and command utilities.
- `github.com/gen2brain/go-fitz` in `ocr_service_simple.go`, `pkg/ocr/fitz`, and OCR orchestrator code.
- `github.com/go-ole/go-ole` in `integration/com_automation.go` and `pkg/integration/com_automation.go`.

## Test Health

Phase 6A command: `go test ./... -count=1 -timeout 300s`

Result: failed. Raw output: `phase6A_test_results.txt`.

Root package failures:

- `TestGetExportDirGroupsDocumentsByEntityWithoutYearDepth`: expected temp export path, actual user Documents export path.
- Deployment tests failed on Windows temp cleanup because SQLite DB files remained open.
- `TestPrepareDeploymentPackage`: missing `build/bin/AsymmFlow.app` bundle.
- `TestManualAuditOneDriveImport`: missing latest OneDrive seed report.
- `TestOfferMetadataExtraction`: expected offer folder/files not present.
- `TestOpportunityConflict_StressSyncsBulkConcurrentActivityAndConflicts`: temp DB file remained open.
- `TestSchemaAudit_NoteAndDivisionPersistenceColumnsAreBackedBySQLite`: temp DB file remained open.
- `ExampleOfferScanner`: expected 88 offers / 23.9% execution / 3 VERTEX offers; actual 16 / 6.2% / 0.

OCR orchestrator failures:

- `TestTieredProcessingFlow`: missing `old_faded_scan.pdf`.
- `TestTieredProcess_Cascade`: missing `new.pdf`.

Passing packages included `integration`, `pkg/api`, `pkg/data`, `pkg/engines`, `pkg/graph`, `pkg/ocr`, `pkg/ocr/fitz`, `pkg/ocr/ksum`, `pkg/ocr/octonion`, `pkg/ocr/predator`, `pkg/ocr/sparse`, and `pkg/vqc`.

Test classification from Phase 6B:

| Class | Count |
| --- | ---: |
| Unit-ish tests (`setupTestApp` or `:memory:`) | 28 |
| Integration/env-gated tests | 7 |
| Manual scripts | 13 |

Coverage gaps from `phase6C_coverage_gaps.txt`: 91 root Go files have no same-name test file.

High-risk gaps include:

- `butler_ai.go`
- `database.go`
- `db_manager.go`
- `db_sync_service.go`
- `delivery_note_service.go`
- `invoice_pdf_service.go`
- `license_service.go`
- `offer_pdf_service.go`
- `payment_intelligence.go`
- `purchase_order_service.go`
- `serial_number_service.go`
- `supplier_invoice_service.go`

## Unique Logic Inventory

The generator must not erase or blindly recreate these areas.

### Business Rules

`phase7_unique_business_rules.txt` produced 2,158 hits for business-rule/BHD/Bahrain/division signals. Major categories:

- BHD currency and VAT calculations.
- Acme Instrumentation/AHS/PH Machinery division behavior.
- Bahrain-specific banking and reconciliation behavior.
- Offer/order/invoice/credit/payment status guards.
- Serial number traceability and government-client compliance.
- Deployment and sync rules around offline-first SQLite/Supabase.
- PDF/export branding and letterhead behavior.

### Algorithms

`phase7_unique_algorithms.txt` produced 166 algorithm hits. Important clusters:

- Payment prediction: `PredictPayment`, `BatchPredict`, `PaymentPredictor.Predict`.
- Costing/margin logic: `CalculateCosting`, `calculateMargin`, ROI and hidden-cost style calculations.
- Bank matching/reconciliation: `AutoMatchBankLines`, `matchToCustomerInvoice`, `matchToSupplierPayment`, `bankMatchTolerance`, `isExpenseReconcilable`.
- Document/OCR classification: `classifyDocumentType`, `DocumentClassifier.Classify`, OCR file classification, AI document classification.
- Data reconciliation: `matchOffersToDB`, `matchInvoicesToDB`, `fuzzyMatchCustomers`, `findBestCustomerMatch`.
- Butler finance intelligence: `calculateWeightedButlerPipeline`, `calculateSystemRegime`, grounded financial brief paths.
- FX/finance: `CalculateFXRevaluation`, VAT reconciliation, AR aging, financial year calculations.
- Traceability: invoice margin recalculation and serial linkage.

### Cannot Be Generated Safely

Do not regenerate these without first extracting tests and behavior fixtures:

- Bank statement matching tolerances and reconciliation audit behavior.
- Butler grounded fast paths, entity resolution, and permission-aware context.
- Offer/costing/invoice PDF layout rules, especially VAT and division-specific branding.
- OCR classification and engine selection strategy.
- Serial lifecycle transitions from GRN to DN to invoice.
- Payment prediction and customer-risk algorithms.
- Sync conflict behavior and offline-first database bootstrap.

## Recommended Wave Priority

### Wave 1: Test Gate Stabilization

Before source refactors, quarantine manual tests, add build tags for environment-dependent tests, fix temp DB close leaks, and add local OCR fixtures or skip gates for missing PDFs. A red `go test ./...` makes every later refactor harder to trust.

### Wave 2: Bank Reconciliation Domain

Extract first as the main domain wave. It is bounded enough to isolate, has high unique value, and contains concrete algorithms generators cannot infer: statement parsing, auto-match tolerances, cheque/deposit reconciliation, and audit logs.

### Wave 3: Delivery + Serial Lifecycle

Extract the GRN -> DN -> delivery -> invoice serial lifecycle after reconciliation. It is important but more contained, and it will benefit from explicit state-machine tests.

### Wave 4: Finance Document Generation

Refactor invoice, offer, purchase order, supplier invoice, credit note, e-invoice, and costing exports only after fixtures exist. These functions are long, layout-sensitive, and client-visible.

### Wave 5: OCR + Butler

Treat OCR and Butler as tangled domains. First preserve classifiers, prompt routes, engine selection, and grounded fast paths; then regenerate only the CRUD/storage shells around them.

## Completion Checklist

| Requirement | Evidence |
| --- | --- |
| Read Wave 0 audit spec | `docs/WAVE0_AUDIT.md` |
| Save raw outputs to `docs/audit_results/` | 36 audit/support artifacts present, including all phase outputs |
| Prerequisite frontend build | `prereq_frontend_build.txt`, build passed |
| Prerequisite Go build | `prereq_go_build_status.txt`, elevated build exit code 0 |
| Phase 1A build-ignored files | `phase1A_build_ignored_files.txt` |
| Phase 1B manual tests | `phase1B_manual_test_scripts.txt` |
| Phase 1C demo files | `phase1C_demo_files.txt` |
| Phase 1D orphan functions | `phase1D_all_exported_funcs.txt`, `phase1D_orphan_functions.txt` |
| Phase 1E stale files | `phase1E_file_last_touched.txt`, `phase1E_stale_files_3months.txt` |
| Phase 2A co-change raw history | `phase2A_go_commits.txt`, `phase2A_co_change_raw.txt` |
| Phase 2B coupling pairs | `extract_coupling.py`, `phase2B_coupling_pairs.txt` |
| Phase 3 hot path traces | `phase3A_*.txt` through `phase3E_*.txt` |
| Phase 4 dependency analysis | `phase4A_internal_imports.txt`, `phase4B_struct_usage.txt`, `phase4C_service_coupling.txt` |
| Phase 5 technical debt | `phase5A_tech_debt_todos.txt`, `phase5B_long_functions.txt`, `phase5C_cgo_deps.txt` |
| Phase 6A test run | `phase6A_test_results.txt` |
| Phase 6B test classification | `phase6B_tests_unit.txt`, `phase6B_tests_integration.txt`, `phase6B_tests_manual.txt` |
| Phase 6C coverage gaps | `phase6C_coverage_gaps.txt` |
| Phase 7 unique logic inventory | `phase7_unique_business_rules.txt`, `phase7_unique_algorithms.txt` |
| Synthesize report using template | `docs/ARCHAEOLOGICAL_REPORT.md` |
| Do not modify source code in this wave | Only docs/audit result artifacts were added; no application source files intentionally edited |
