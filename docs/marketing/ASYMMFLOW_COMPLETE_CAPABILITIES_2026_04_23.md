# AsymmFlow Complete Capability Profile

Source-grounded marketing document  
Prepared: 23 April 2026

## Executive Positioning

AsymmFlow is an offline-first ERP and operating intelligence platform for trading, engineering supply, procurement, delivery, finance, and management teams. It brings the commercial pipeline, costing, quotations, customer orders, procurement, supplier invoices, delivery notes, invoicing, banking, payroll, expenses, tasks, people operations, OCR capture, and AI-assisted business intelligence into one desktop application.

The product is designed around a local operating database, so daily business work can continue even when the internet is unavailable. Optional cloud sync, OCR, and AI services extend the system when connectivity and credentials are configured, but the core ERP workflows remain anchored in the local application.

AsymmFlow is not just a record-keeping tool. It is built to preserve the operating memory of a company: customers, suppliers, quotations, commercial terms, delivery commitments, payment behavior, employee work, document evidence, and the context behind decisions.

## Scope And Claim Discipline

This document is based on the AsymmFlow codebase and local database inspected on 23 April 2026. It is intentionally written as a professional marketing and capability profile, while avoiding claims that are not supported by the application.

Capability labels used in this document:

| Label | Meaning |
| --- | --- |
| Primary workflow | Visible in the current hub or screen structure and intended for regular user operation. |
| Service-backed capability | Implemented in backend services and/or secondary screens, even if not always foregrounded in the main hub tabs. |
| Optional integration | Available when external credentials, cloud services, or deployment configuration are present. |

Accuracy guardrails:

- Core ERP operations are local and offline-first.
- Cloud sync requires configured Supabase credentials and network access.
- AI assistance requires configured model credentials and network access.
- OCR can use local extraction for supported digital documents, while scanned and image-heavy workflows may use configured vision or remote OCR services.
- AI and OCR outputs support review, capture, and workflow acceleration. They should not be marketed as autonomous decision-making.
- Some service-backed modules may be in controlled rollout or secondary navigation even though the underlying capability exists.

## Product At A Glance

| Area | Capability |
| --- | --- |
| Application type | Desktop ERP application built with Wails, Go, Svelte, and TypeScript. |
| Operating model | Offline-first local database with optional background cloud synchronization. |
| Local database | SQLite operating database with a broad ERP schema across sales, finance, procurement, HR, tasks, documents, AI, and system administration. |
| Cloud sync | Optional Supabase bidirectional sync with status indicator, manual sync controls, first-run sync, health checks, and pending operation handling. |
| User interface | 58 Svelte screen modules across commercial, finance, operations, CRM, intelligence, work, people, settings, and deployment areas. |
| Backend depth | 40 service files plus core application, database, sync, OCR, licensing, reporting, and security services. |
| Permission model | 642 permission guard call sites and 102 unique permission strings in the inspected codebase. |
| Intelligence layer | Butler AI conversations, grounded business context, permission-aware finance redaction, action suggestions, daily briefings, and AI-assisted reports. |
| Document capture | OCR and structured extraction for PDFs, images, Excel files, Word documents, emails, MSG files, bank statements, RFQs, invoices, POs, quotations, and delivery documents. |
| Reporting outputs | Letterhead PDFs, CSV exports, VAT return data, UBL e-invoice XML, support bundles, graph JSON exports, and operational reports. |
| Security foundation | Role-based access, license activation, device binding, audit trails, delete approvals, path validation, document sanitization, and encrypted/signed sensitive fields. |

## Operating Hubs

AsymmFlow is organized around practical business hubs rather than isolated modules.

| Hub | What It Covers |
| --- | --- |
| Sales Hub | RFQs, opportunities, costing sheets, quotations, offers, customer orders, commercial terms, and conversion workflows. |
| Operations Hub | Supplier purchase orders, supplier invoices, delivery notes, receiving, GRN services, QC, serial traceability, and delivery execution. |
| Finance Hub | Financial dashboard, invoices, payments received, payments made, expenses, approvals, payroll, bank reconciliation, and related finance services. |
| CRM Hub | Customer and supplier dashboards, master data, relationship history, contacts, 360-degree views, notes, risk indicators, and profile enrichment. |
| Intelligence Hub | Butler AI, business Q&A, document analysis, report generation, data archaeology, and entity discovery. |
| Work Hub | Tasks, projects, team board, comments, status updates, assignments, blocking, due dates, and collaborative work tracking. |
| People Hub | Employee directory, roles, access links, managers, active status, contribution summaries, and employee-project relationships. |
| Settings And Admin | Company settings, directories, AI keys, business rules, currency rates, imports, financial reports, Supabase sync, deployment, licensing, devices, and user management. |

## Executive Dashboard And Daily Workspace

The main dashboard is role-aware. Finance users see financial operating signals, while non-finance users see commercial and task-oriented signals.

Capabilities include:

- Personalized greeting and current user context.
- Permission-aware KPI selection.
- Finance-visible KPIs such as revenue, cash balance, accounts receivable, and pipeline value.
- Non-finance KPIs such as active RFQs, pipeline value, active orders, and win rate.
- Operating focus cards for cash balance, accounts receivable, follow-ups, pipeline conversion, and operational handoffs.
- Follow-up task summaries from commercial follow-up data and collaborative task data.
- Context task modal access from the dashboard.
- Real-time event refresh hooks for updated metrics.
- Role-aware visibility so sensitive finance values are not shown to users without finance permissions.

### Multi-Division Dashboard Support

The codebase includes division-filtered financial dashboard support, including an Beacon Controls WLL dashboard.

Capabilities include:

- Division-specific financial dashboard loading.
- Financial year selector.
- Revenue, net result, cash, assets, equity, cost of sales, gross profit, staff cost, and administrative expense display.
- Audited source indicators when source data is marked audited.
- No-data state explaining how division-tagged costings, offers, orders, and invoices feed the dashboard.

### Pricing Strategy Simulation

The pricing screen supports customer-specific pricing exploration and margin simulation.

Capabilities include:

- Customer selection for pricing analysis.
- Target margin slider.
- Backend margin simulation using customer name and target margin.
- Projected win rate, current win rate, confidence, recommendation, and warning display when simulation data is returned.
- Strategy labels such as premium, value-balanced, and price-sensitive customer regimes where configured.

## Sales And Commercial Pipeline

### Opportunity And RFQ Management

AsymmFlow supports the complete front end of the commercial cycle, from inquiry and RFQ capture through qualification, quotation, win/loss tracking, and handoff into order fulfillment.

Capabilities include:

- Opportunity and RFQ listing with search, filters, status views, year filters, and customer linkage.
- Pipeline stages including Qualified, Proposal, Quoted, Won, Lost, and imported pipeline states.
- RFQ creation with customer, project, RFQ reference, received date, due date, priority, estimated value, and notes.
- Merged commercial view combining RFQs with imported pipeline opportunities while deduplicating by folder or RFQ reference.
- Pipeline statistics such as quoted count, won count, total value, and win rate.
- Customer and project context on opportunities.
- Source fields such as folder number, EH reference, owner notes, payment terms, delivery terms, source, and pipeline comments.
- Task creation from commercial context, allowing opportunities to become assigned work items.
- Imported canonical pipeline support, including large opportunity sets and customer name resolution.

### Costing Sheets

The costing sheet workflow is one of the most detailed commercial modules in the application. It is built for engineered product quotations where margin, currency, freight, landed cost, VAT, commercial terms, and customer-facing document quality matter.

Capabilities include:

- Costing creation and editing linked to RFQs, pipeline opportunities, customers, suppliers, and products.
- Multi-line item costing with serial number column, equipment description, model number, supplier code, long code, specification, quantity, and detailed description.
- Multi-currency inputs including BHD, EUR, USD, GBP, CHF, and related exchange-rate handling.
- Cost components including FOB, freight, insurance, customs, handling, finance, other cost, and unit pricing.
- Suggested selling price, margin calculation, profit calculation, discount, VAT, and grand total.
- Editable VAT rate with support for explicit zero-rated quotations where applicable.
- Hidden charges calculation support for internal profitability impact without exposing those charges on customer-facing outputs.
- Business rule support for default margin and VAT settings.
- Commercial document types such as Quotation, Budgetary Quote, Estimate, Technical Quote, and Commercial Offer.
- Delivery time, delivery terms, payment terms, order type, country of origin, certificates, installation, commissioning, testing, and tax category fields.
- Prepared-by staff selection and division selection.
- Terms and conditions support.
- Revision handling, active revision selection, and cloning to new revisions.
- PDF export with customer-facing layout and annexure handling.
- Excel or CSV export with escaped fields for spreadsheet use.
- Save-as-offer workflow that carries commercial terms into the offer module.
- Unsaved-change warning on costing navigation.

### Offers And Quotations

The offers workflow turns costing and commercial intent into tracked customer-facing quotations.

Capabilities include:

- Offer listing, search, filtering, and status tracking.
- Offer creation from costing sheets and manual commercial data.
- Quote type, VAT rate, payment terms, delivery terms, delivery weeks, origin, contact person, customer reference, attention, issuer, and terms support.
- Offer line items with quantity, cost, pricing, discount, and margin information.
- PDF generation using the configured quotation type and VAT rate.
- Status progression including RFQ, Quoted, Won, and Lost.
- Conversion of won offers into customer orders.
- Lost reason capture and competition context.
- Validity date and expiry tracking.
- Offer notes and follow-up management.
- Pending follow-up tracking and completion.
- Legacy offer detection for offers without item details.

### Customer Orders

Customer orders bridge confirmed commercial work into procurement, delivery, and invoicing.

Capabilities include:

- Customer order listing with search, year, customer, and status filters.
- Order creation and editing with linked customer and item details.
- Status tracking across Confirmed, In Progress, Partially Delivered, Fully Delivered, Invoiced, and Cancelled states.
- Order item tracking with ordered, shipped, and invoiced quantities.
- Costing context retained on order items where available.
- Delivery and fulfillment visibility.
- Quick delivery status actions.
- Invoice and proforma invoice creation from orders.
- Purchase order creation from customer orders.
- Order issue checks for missing or zero-value item data.
- Task creation from order context.

## Procurement, Operations, And Supply Chain

### Purchase Orders

The purchase order workflow supports supplier ordering, approval, receiving, amendment, and document generation.

Capabilities include:

- Manual purchase order creation and purchase order creation from customer orders.
- Supplier, customer order, project, and item linkage.
- Multi-currency procurement with exchange-rate support.
- Status tracking through Draft, Sent, Acknowledged, Partially Received, Fully Received, Closed, and related operational states.
- Approval logic including financial threshold handling.
- Purchase order number generation.
- Amendment support and amendment history.
- PDF generation for supplier-facing purchase orders.
- Supplier payment and due-date context.
- Receiving handoff into GRN workflows.

### Supplier Invoices And 3-Way Matching

Supplier invoice handling connects procurement, receiving, finance, and payment workflows.

Capabilities include:

- Supplier invoice creation, editing, listing, and search.
- Linking to purchase orders and GRNs.
- Match status tracking for matched, partial, mismatch, and pending cases.
- 3-way comparison between PO, GRN, and supplier invoice.
- Approval states such as Pending, Verified, Approved, and Paid.
- Payment status tracking including Pending, Partial, Paid, and Overdue.
- Multi-currency supplier invoice support.
- Overdue warnings and payment modal support.
- Optional creation of supplier payment records from invoice context.
- Segregation controls for supplier invoice approval flows.

### Goods Received Notes And QC

The codebase includes GRN workflows for receiving goods against purchase orders or manually, with quality-control support.

Capabilities include:

- GRN creation from purchase orders or manual receiving.
- GRN listing and filtering.
- Ordered, previously received, currently received, rejected, and remaining quantity tracking.
- QC status tracking including Pending, Passed, Failed, and Partial.
- Serial number capture at receiving where applicable.
- Acceptance-rate visibility.
- GRN completion workflows.
- Discrepancy creation and resolution services.

### Delivery Notes And Dispatch

Delivery notes manage the movement of goods from prepared shipment through dispatch and delivery confirmation.

Capabilities include:

- Delivery note creation and editing.
- Delivery note number generation.
- Customer, order, delivery address, contact, driver, vehicle, and transport method capture.
- Status tracking across Draft, Prepared, Dispatched, In Transit, Delivered, and Signed states.
- Delivery item line management.
- Partial delivery support.
- Dispatch and delivery confirmation workflows.
- Delivery sequence and operational route context.
- PDF generation for delivery documentation.
- Pending delivery summaries and delivery-area reporting.
- Integration points for serial allocation.

### Serial Traceability

The serial number service supports equipment traceability across receiving, delivery, invoicing, and customer ownership.

Capabilities include:

- Serial registration and lookup.
- Serial search by product, customer, status, or related document.
- Assignment to GRN items.
- Allocation to delivery notes.
- Marking serials as shipped or delivered.
- Linking serials to invoices and invoice items.
- Warranty and calibration certificate metadata updates.
- Available serial selection for delivery workflows.

### Inventory And Stock Controls

The backend includes inventory services for stock master data, movements, valuation, adjustments, and replenishment support.

Capabilities include:

- Inventory item creation and listing.
- Warehouse and stock movement foundations.
- Inventory valuation reporting.
- Stock movement recording.
- Stock adjustment creation and approval.
- Low-stock item detection.
- Reorder suggestions.
- Slow-moving stock alert services.
- Inventory alert summaries.

## Finance, Accounting, And Compliance

### Financial Dashboard

The financial dashboard gives management a consolidated view of profitability, liquidity, working capital, and balance sheet position.

Capabilities include:

- Year selector for financial views.
- Revenue, cash balance, accounts receivable, and net profit KPIs.
- P&L summary with revenue, cost of goods, gross profit, expenses, and net profit.
- Balance sheet view across assets, liabilities, and equity.
- Liquidity ratios including current ratio, quick ratio, and cash ratio.
- Solvency metrics including debt-to-equity and equity ratio.
- Efficiency metrics including DSO, DIO, DPO, cash conversion cycle, asset turnover, and receivables turnover.
- Profitability metrics including ROA, ROE, gross margin, and net margin.
- Accounts receivable aging buckets.
- Audited-data indicators where verified financial statements are present, and live ERP indicators for current periods.

### Customer Invoicing

Customer invoicing is integrated with orders, payments, credit notes, document generation, and financial reporting.

Capabilities include:

- Customer invoice listing, creation, editing, sending, and deletion controls.
- Invoice generation from customer orders.
- Proforma invoice support.
- Status tracking across Draft, Sent, Paid, Overdue, Partially Paid, and excluded statuses for reporting logic.
- Invoice item lines with equipment, description, quantity, price, VAT, and totals.
- Customer and order context on invoices.
- HMAC-style document integrity support in backend services.
- Field visibility controls for generated invoice PDFs.
- Serial number inclusion on invoices where serial data is available.
- PDF generation for invoices.
- Outstanding, overdue, revenue, and payment status visibility.

### Credit Notes

Credit note workflows support controlled corrections and customer account adjustments.

Capabilities include:

- Credit note creation against customer invoices.
- Credit note item lines and reasons.
- Draft, issued, and applied lifecycle.
- Prevention of over-crediting against invoice outstanding balances.
- Credit note PDF generation.
- Application of credit notes to invoice balances.

### Customer Payments

Payment recording is linked to invoices, cash reporting, and bank reconciliation.

Capabilities include:

- Customer payment creation, editing, and deletion.
- Payment allocation to invoices.
- Outstanding-balance prefill and reconciliation support.
- Payment methods including bank transfer, cheque, cash, credit card, LC, PDC, wire transfer, online, and other methods.
- KPI views for collected amount, monthly collections, average days, and payment count.
- Filters for all periods, current month, and current quarter.
- Bank reconciliation navigation from payment context.

### Supplier Payments

Supplier payment workflows support payables execution and reconciliation.

Capabilities include:

- Supplier payment creation, editing, deletion, and listing.
- Linking payments to supplier invoices.
- Supplier payment summaries.
- Race-condition protection in backend payment operations.
- Integration with banking and reconciliation candidates.

### VAT, E-Invoicing, And Tax Outputs

The finance layer includes compliance-oriented exports and electronic invoice generation.

Capabilities include:

- UBL 2.1 e-invoice XML generation.
- VAT return CSV export.
- VAT reconciliation reporting.
- Output VAT and input VAT treatment with exclusion of invalid supplier invoice states.
- Exclusion of Cancelled, Void, Proforma, and Draft invoices from relevant VAT return calculations.
- Support for VAT rate settings and explicit zero-rated quotation/invoice flows where implemented.

### Bank Reconciliation

Bank reconciliation turns bank statement lines into matched finance evidence.

Capabilities include:

- Bank statement import from PDF, CSV, and dialog-driven workflows.
- Company bank account management.
- Bank statement line listing, editing, deleting, and manual addition.
- Auto-matching of statement lines.
- Manual matching and unmatching.
- Candidate matching from customer invoices, supplier invoices, supplier payments, expenses, payroll payouts, and related finance records.
- Split allocation support.
- Reconciliation finalization and verification workflows.
- Audit trail visibility.
- Matched, unmatched, debit, and credit summary KPIs.
- Cash position context from bank data.

### Book-Bank Reconciliation

The codebase includes book-bank reconciliation services for comparing ERP book balances with bank statement positions.

Capabilities include:

- Book-bank reconciliation creation, updating, finalization, and listing.
- Latest reconciliation lookup.
- Reconciliation reporting and variance summaries.
- Deposits in transit creation, clearing, and return handling.
- Auto-match services for deposits and cheques.
- Variance analysis between book and bank balances.

### Cheque Register

Cheque management supports practical cash operations and outstanding instrument tracking.

Capabilities include:

- Cheque register creation and listing.
- Cheque number sequencing and exhaustion checks.
- Cheque issue workflow.
- Status updates for presented, cleared, stale, bounced, cancelled, and reissued cheques.
- Outstanding cheque reporting.
- Stale cheque detection.
- Register-level reporting.

### FX Revaluation

Foreign-currency finance services help track exposure and unrealized gains or losses.

Capabilities include:

- FX rate creation, retrieval, history, listing, and deletion.
- Latest exchange-rate lookup.
- Foreign-currency exposure reporting.
- Revaluation calculation.
- Listing of unposted revaluations.
- Posting and reversal of revaluation entries.
- Gain and loss summaries.
- Revalue-all workflows for foreign accounts.

### Accounting

AsymmFlow includes accounting foundations for ledger structure and journal control.

Capabilities include:

- Chart of accounts management.
- Journal entry creation.
- Journal line handling.
- Journal validation and balance checks.
- Posting and reversal workflows.
- Accounting screen for finance users.
- Integration points from expenses, payroll, and finance services.

### Expenses

Expense workflows support controlled operational spend capture, approval, posting, and payment.

Capabilities include:

- Expense category and vendor management.
- Expense entry creation with amount, VAT, date, due date, cost center, and notes.
- Draft, submitted, approved, rejected, posted, and paid flows.
- Expense submission, approval, rejection, posting, and payment workflows.
- Recurring expenses with frequency, auto-submit settings, and next-run dates.
- Bank expense candidate creation from bank statement context.
- Expense dashboard metrics for drafts, submitted items, approved unpaid items, recurring items, month-to-date spend, and upcoming expenses.
- Expense attachments, allocation models, and approval audit structures.
- Expense journal posting integration.

### Payroll

Payroll services connect employee compensation, payroll periods, approvals, payment execution, and finance posting.

Capabilities include:

- Employee compensation profiles with base salary, housing allowance, transport allowance, other allowance, standard deductions, tax deductions, employer cost, and effective dates.
- Payroll period creation and tracking.
- Payroll run generation.
- Payroll run approval.
- Payroll posting.
- Payroll payment marking with bank account and reference details.
- Payroll run items, components, and payout records.
- Unreconciled payroll payout support for bank reconciliation.
- Payroll dashboard metrics such as active profiles, open periods, draft runs, approved unpaid runs, month-to-date net pay, and upcoming liability.
- Payroll journal and expense synchronization support.

### Financial Reports

Finance reporting services provide management, compliance, and operational views beyond transaction screens.

Capabilities include:

- Payment aging report.
- Cash flow projection.
- Margin analysis by customer and product.
- VAT reconciliation.
- Period close and closed-period checks.
- P&L generation.
- Balance sheet generation.
- Butler-assisted report generation with letterhead output.

### Business Intelligence Reports Screen

The reports screen provides category-based analytics and export controls.

Capabilities include:

- Report categories for sales, customers, operations, inventory, and financial reporting.
- Date range controls.
- Dashboard metric enrichment where available.
- Sales metrics such as win rate, conversion rate, average deal size, and pipeline value charts.
- Financial metrics such as runway and cash balance where data is available.
- Export modal with selectable output format.
- Report export through backend export services.

## CRM And Relationship Management

### Customer Master Data

Customer data is a first-class part of the application, not just a lookup table.

Capabilities include:

- Customer creation, editing, deletion controls, listing, and search.
- Business name, trading name, customer code, customer type, address, city, country, industry, phone, email, mobile number, website, and tax fields.
- TRN, VAT, payment terms, credit limit, credit status, and risk indicators.
- Customer grade and relationship history support.
- Active, deleted, and duplicate cleanup handling.
- Customer contacts with add, edit, delete, and list workflows.
- Customer notes.
- Customer order history, recent orders, opportunities, and invoice context.
- Customer graph and customer 360-degree views.
- Customer profitability, activity, payment behavior, and risk context for intelligence workflows.

### Supplier Master Data

Supplier management is connected to procurement, invoices, payments, lead time, and supplier quality.

Capabilities include:

- Supplier creation, editing, deletion controls, listing, and search.
- Supplier contacts with add, edit, delete, and list workflows.
- Supplier notes and issue tracking.
- Supplier 360-degree profile.
- Supplier invoice and payment history.
- Supplier lead-time metrics.
- Late supplier reporting.
- Supplier performance context for Butler and reporting.

### Customer And Supplier 360-Degree Views

Relationship screens provide operational context without forcing users to assemble it manually from separate lists.

Capabilities include:

- Customer 360-degree detail view.
- Supplier 360-degree detail view.
- Customer graph and relationship data.
- Supplier invoice, purchase order, payment, and issue context.
- Customer orders, invoices, contacts, notes, and opportunities.
- Navigation from dashboards into detailed records.

### Data Quality And Master Data Cleanup

The application includes data quality workflows and services that support long-lived customer and supplier master data.

Capabilities include:

- Duplicate customer detection and cleanup support.
- Customer name resolution for seed and import workflows.
- Customer grade enrichment.
- Soft-delete and merge-safe approaches in database operations.
- Master data cleanup review artifacts.
- Protection against orphaned records during merge operations.

## Intelligence Layer

### Butler AI

Butler is the embedded intelligence layer inside AsymmFlow. It is designed to answer operational business questions using application context rather than acting as a generic chatbot.

Capabilities include:

- Persistent AI conversations with saved messages.
- Conversation list, load, delete, and purge actions.
- Markdown table rendering for structured answers.
- Grounded context builders across commercial, financial, operational, banking, supplier, customer, product, delivery, serial, risk, credit, DSO, offer expiry, forecasting, and action-item domains.
- Permission-aware redaction of financial context where users lack access.
- Natural-language questions over customers, suppliers, invoices, payments, offers, tasks, revenue, risk, delivery, and operations.
- Fast-path grounded answers for common business questions.
- Daily briefing generation.
- Action suggestions that can navigate to or create relevant workflow targets.
- Draft offer and workflow-assist capabilities where the user remains in control.
- Butler report generation with structured report data and letterhead PDF output.
- Prompt sanitization and prompt-injection hardening in relevant services.

### Butler-Aware Action Layer

The intelligence layer can connect conversations to practical work.

Capabilities include:

- Action aliases for creating or opening offers, follow-ups, orders, purchase orders, invoices, RFQs, costing sheets, supplier invoices, stock adjustments, customers, suppliers, contacts, and briefings.
- Navigation from AI-suggested actions into the relevant app screens.
- Context task creation from commercial and operational screens.
- Document analysis handoff into Quick Capture and entity creation.

### Data Archaeology And Entity Discovery

AsymmFlow includes analysis tools for discovering relationships and evidence across documents, source artifacts, and business data.

Capabilities include:

- Data archaeology scan workflows.
- Archive or source path analysis.
- Evidence extraction and artifact generation.
- Entity graph build and rebuild services.
- Entity graph search.
- Graph statistics.
- Node relationship inspection.
- Graph JSON export.
- Customer graph views.

## OCR, Document Capture, And Data Ingestion

### Drag-And-Drop Capture

AsymmFlow includes a global document capture flow that turns files into reviewed business records.

Capabilities include:

- Global file drop after license validation.
- Supported drop extensions include PDF, DOCX, XLSX, PNG, JPG, JPEG, MSG, and EML in the current app shell.
- OCR processing from the dropped file path.
- Automatic document type detection when set to auto.
- Quick Capture modal for reviewing extracted fields.
- Raw text inspection.
- Editable extracted fields before saving.
- Customer, supplier, and bank account matching.
- Line-item review.
- Duplicate RFQ and opportunity checks.
- Save-to-entity routing into appropriate business modules.

### Smart Inbox

The application includes an inbox screen for document processing and review.

Capabilities include:

- Inbox document listing.
- Inbox statistics for ready, needs-review, processed, and total documents.
- Document type breakdown.
- Filters for all, new, and review states.
- Re-processing of inbox documents.
- Classification result display.
- Archive or mark-processed workflow.
- Selected document detail view.

### OCR And Extraction Engines

The OCR service supports multiple document formats and extraction strategies.

Capabilities include:

- PDF text extraction, including local extraction paths for digital PDFs.
- Image document processing for PNG, JPG, JPEG, BMP, TIFF, TIF, and WEBP where configured.
- Excel extraction for XLSX and XLS files.
- Word document extraction for DOCX.
- RTF text extraction.
- MSG email parsing.
- EML email parsing.
- Batch processing support.
- Processor statistics.
- Fallback paths across local extraction, configured vision models, and remote OCR runtime where available.
- Confidence, engine, processing time, cache, and cost metadata in OCR result structures.

### AI Document Classification

Document classification combines model-assisted classification with deterministic fallback rules.

Capabilities include:

- Classification for RFQ, invoice, supplier invoice, purchase order, quotation, delivery note, bank statement, contract, report, and other documents.
- Route recommendation by document type.
- Key field extraction.
- Line-item extraction.
- Confidence scoring.
- Filename and folder-based document classification for local archive scans.
- File metadata extraction including size and modified time.

### Structured Business Capture

After OCR, AsymmFlow can turn extracted evidence into actual ERP records.

Capabilities include:

- Save RFQs and inquiries into opportunities.
- Save customer invoices into finance records where appropriate.
- Save supplier invoices into supplier invoice workflows.
- Save purchase orders into procurement workflows.
- Save delivery notes into operations workflows.
- Save bank statements into bank reconciliation workflows.
- Store contracts and reports as document evidence.
- Save OCR audit records.
- Extract bank statement metadata such as account, IBAN, currency, statement period, opening balance, closing balance, debit count, credit count, and statement lines.

### Butler Document Analysis

The OCR layer can hand documents to Butler for structured business interpretation.

Capabilities include:

- AI-assisted document analysis for RFQ, purchase order, invoice, quotation, bank statement, delivery note, supplier invoice, inquiry, and other document types.
- Structured extraction of customer, project, deadline, item lines, metadata, confidence, and recommended actions.
- Bank statement extraction of bank name, account, IBAN, period, currency, opening balance, closing balance, debit totals, credit totals, and line transactions.
- Review-first workflow before saving extracted data.

### Email And Archive Import

AsymmFlow includes tooling for bringing business communication and historical files into the operating system.

Capabilities include:

- MSG parsing.
- EML parsing.
- Batch email parsing.
- Parsed email save-as-RFQ workflow.
- OneDrive or local folder path validation and scan services.
- Deal discovery from folder structures.
- Customer matching during archive import.
- File discovery for offers, costings, purchase orders, acknowledgements, RFQs, delivery notes, shipping documents, and technical documents.
- Controlled import of opportunities, costings, and line items from discovered deal folders.

## Work Management, Tasks, And Collaboration

### Work Hub

The Work Hub turns business operations into accountable work.

Capabilities include:

- My Work view.
- Team board.
- Project view.
- Task creation with title, description, priority, due date, project, and assignee.
- Task status tracking across open, in progress, blocked, and completed states.
- Task assignment and reassignment.
- Due date changes.
- Task detail editing.
- Block and unblock workflows.
- Task comments.
- Task activity history.
- Project creation.
- Project membership.
- Snapshot caching for responsive work views.
- Remote refresh support where sync is configured.

### Notifications

Notifications connect tasking, approvals, and cross-device collaboration.

Capabilities include:

- Persistent notification feed.
- Unread-only filtering.
- Date grouping.
- Mark-as-read behavior.
- Task notification opening into Work Hub.
- Delete approval notification handling.
- Notification receipts.
- Sidebar unread badge.
- Polling refresh in the app shell.

### Collaborative Sync And Pending Operations

Collaboration services include foundations for pending operation handling and retry.

Capabilities include:

- Collaborative pending operation records.
- Retry controls from settings or deployment support flows.
- Remote schema and cursor handling for collaborative sync.
- Support bundle export for troubleshooting.
- Sync status and readiness checks.

## People, HR, And Access Operations

### People Hub

The People Hub connects employee identity, access, work, and contribution tracking.

Capabilities include:

- Employee directory.
- Employee profile creation and updates.
- Active and inactive employee status.
- Department, job title, manager, start date, end date, contact, and emergency contact fields.
- Employee notes.
- Manager relationships.
- Project assignments.
- Contribution summaries.
- Employee access links.
- License key association with employees.

### User And Device Administration

Administration workflows cover both people and devices.

Capabilities include:

- User management screen.
- Role assignment.
- License key listing, generation, updating, and revocation.
- Device identity generation from machine characteristics.
- Device approval, blocking, and unblocking.
- First-admin setup flow.
- Pending device flows.
- Password hashing for admin setup and user credentials.
- Audit trail viewer.

## Security, Licensing, And Permissions

### Role-Based Access Control

The codebase applies permission checks extensively across backend endpoints and frontend navigation.

Capabilities include:

- Permission-filtered navigation.
- Backend permission guards using named permissions.
- Role definitions for Admin, Manager, Sales, Operations, and Staff.
- Broad permission namespaces across finance, invoices, payments, customers, suppliers, purchase orders, delivery, GRN, documents, intelligence, tasks, projects, notifications, HR, payroll, expenses, settings, users, sync, and reporting.
- Wildcard administrative permission support.
- Permission-aware AI context redaction.

### Licensing

Licensing controls product activation and device access.

Capabilities include:

- License key format tied to role.
- Cryptographically generated license material.
- Device-bound activation.
- Device transfer handling for developer master access where enabled.
- Activation rate limiting.
- License validation.
- License listing.
- License updating.
- License revocation with protected master-key behavior.
- Employee-license mapping support.

### Security Hardening

The application includes multiple hardening measures around sensitive operations.

Capabilities include:

- Path validation for document processing.
- File type validation and size limits in OCR services.
- Sanitized logging for sensitive values.
- Prompt sanitization for AI services.
- Database nil checks in critical services.
- Backup file permission controls.
- Soft-delete and audit-friendly data handling.
- Delete approval workflow for guarded deletions.
- Segregation controls in supplier invoice approval.
- Business rule validation around margin, VAT, mobile numbers, quote types, and financial inputs.

## Settings, Deployment, And Administration

### Settings Flow

Settings are designed as an operational control center rather than a simple preferences page.

Capabilities include:

- General company settings.
- Base currency and company identity settings.
- Outlook and Excel integration toggles.
- Directory mappings for RFQs, offers, invoices, EH XML, customers, and reports.
- AI and intelligence settings, including API keys and model selection.
- AI connection tests.
- Business rule settings including default margin and VAT.
- Currency rate management.
- Data import tools.
- Financial report generation tools.
- Supabase sync configuration and health checks.
- Manual backup and backup status visibility.
- Support bundle export.
- Collaborative pending operation retry.
- Deployment workspace access.

### Deployment Hub

Deployment workflows support rollout readiness, support, and operational signoff.

Capabilities include:

- Pilot rollout audit.
- Employee, license, and device readiness checks.
- Pilot checklist.
- Support controls.
- Support bundle export.
- Pilot signoff report export.
- License display name and employee access reassignment.
- Sync trigger support.
- Data audit visibility.
- Phase rollout status tracking.

### Setup And Activation

Initial setup and activation are built into the application lifecycle.

Capabilities include:

- License activation screen.
- Setup admin screen.
- Setup wizard.
- First-run admin creation.
- Device pending state handling.
- App-level license validation before entering the main workspace.

## Reporting, Documents, And Exports

AsymmFlow includes document generation and export capabilities across commercial, finance, operations, and support workflows.

Capabilities include:

- Offer and quotation PDFs.
- Costing PDFs.
- Costing spreadsheet exports.
- Customer invoice PDFs.
- Proforma invoice support.
- Credit note PDFs.
- Purchase order PDFs.
- Delivery note PDFs.
- Butler report PDFs.
- VAT return CSV export.
- UBL e-invoice XML generation.
- Bank statement import and reconciliation outputs.
- Financial reports including P&L and balance sheet.
- Entity graph JSON export.
- Support bundle export.
- Pilot signoff report export.
- Master data cleanup and audit artifacts.

## Offline-First Architecture And Sync

### Local Operating Database

The application is built around a local SQLite database as the operating source for day-to-day work.

Capabilities include:

- Local-first database operation.
- WAL mode for concurrent read/write behavior.
- Busy timeout handling for database locks.
- Normal synchronous mode for balanced local performance.
- Local cache and mmap tuning.
- Schema migrations.
- Integrity check on startup.
- Backup using SQLite VACUUM INTO.
- Backup pruning with last-backup tracking.
- Manual backup trigger.
- Backup status reporting.
- Database path discovery.

### Optional Cloud Synchronization

Cloud sync is an optional operating layer for multi-device use.

Capabilities include:

- Supabase connection management.
- Sync health checks.
- Manual sync now operation.
- First-run sync support.
- Background sync service.
- Table-level push and pull.
- Sync progress reporting.
- Sidebar sync status indicator.
- Offline, connecting, active, and paused status visibility.
- Sync error handling and circuit-breaker style controls.
- Pending collaborative operation retry.

### Practical Offline Message

The safest external positioning is:

> AsymmFlow keeps the operating ERP database local so users can keep working without internet. When cloud credentials and connectivity are available, the system can synchronize data across devices and use AI or OCR services to accelerate document capture and analysis.

This statement is stronger and more accurate than saying every feature is fully offline, because cloud sync and external AI services depend on network access.

## Current Data Footprint Snapshot

The inspected local database contains a substantial operating dataset. These figures are a snapshot from 23 April 2026 and should not be treated as contractual production counts.

| Data Area | Snapshot Count |
| --- | ---: |
| Active customer records | 381 |
| Suppliers | 35 |
| Products | 20 |
| Active opportunities | 518 |
| RFQs | 38 |
| Offers | 163 |
| Customer orders | 196 |
| Customer invoices | 470 |
| Customer payments | 93 |
| Supplier invoices | 484 |
| Supplier payments | 601 |
| Purchase orders | 52 |
| Delivery notes | 155 |
| Employee records | 15 |
| Task items | 56 |

Pipeline snapshot:

| Stage | Opportunities | Pipeline Value |
| --- | ---: | ---: |
| Qualified | 169 | 1,474,763.12 BHD |
| Won | 169 | 539,631.88 BHD |
| Lost | 117 | 683,063.39 BHD |
| Quoted | 57 | 598,377.05 BHD |
| Proposal | 6 | 144,198.80 BHD |

## Feature Inventory Matrix

| Domain | Detailed Capabilities |
| --- | --- |
| Executive dashboard | Permission-aware KPIs, cash and AR signals, pipeline value, active RFQs, active orders, win rate, operating focus, follow-ups, task signals. |
| Multi-division reporting | Division-filtered financial dashboard, Beacon Controls dashboard, financial year selector, audited source indicators, no-data guidance for division tagging. |
| Sales pipeline | RFQs, opportunities, pipeline stages, win/loss status, year filters, customer matching, imported pipeline fields, RFQ creation, task creation from commercial context. |
| Pricing | Customer-specific pricing simulation, target margin slider, estimated win rate, confidence, recommendation, and pricing regime labels where configured. |
| Costing | Multi-line costing, currency conversion, landed cost, margin, VAT, discounts, hidden charges, quote type, terms, certificates, country of origin, revisions, PDF export, spreadsheet export, save as offer. |
| Offers | Quotation management, PDF output, VAT rate, payment terms, delivery terms, follow-ups, notes, validity, won/lost conversion, customer order creation. |
| Orders | Order CRUD, fulfillment status, shipped and invoiced quantities, invoice/proforma creation, PO creation, quick delivery actions, task context. |
| Procurement | Purchase orders, supplier linkage, customer order linkage, approval thresholds, status progression, amendments, PDF generation, receiving handoff. |
| Receiving | GRNs, manual receiving, PO receiving, quantity variance, QC status, serial capture, discrepancy services. |
| Delivery | Delivery notes, dispatch, partial delivery, signed delivery state, driver/vehicle/contact/address fields, PDF, serial allocation support. |
| Serial traceability | Register, search, assign to GRN, allocate to delivery, mark shipped/delivered, link to invoice, warranty, calibration certificate metadata. |
| Inventory | Inventory items, valuation, stock movements, stock adjustments, low-stock detection, reorder suggestions, slow-moving alerts. |
| Customer finance | Invoices, proformas, payments, credit notes, outstanding balances, overdue tracking, invoice PDFs, payment KPIs. |
| Supplier finance | Supplier invoices, matching, approvals, supplier payments, overdue handling, reconciliation candidates. |
| Banking | Bank account management, bank statement import, auto-match, manual match, split allocation, finalization, cash position, audit trail. |
| Treasury controls | Cheque registers, outstanding cheques, stale/bounced/cancelled/reissued states, book-bank reconciliation, deposits in transit, FX rates, FX revaluation. |
| Accounting | Chart of accounts, journal entries, validation, posting, reversal, finance posting integration. |
| Compliance | VAT return export, VAT reconciliation, UBL e-invoice XML, tax-aware document fields. |
| Payroll | Compensation profiles, payroll periods, payroll runs, approval, posting, payment marking, payout reconciliation. |
| Expenses | Categories, vendors, expense entries, approval, posting, payment, recurring expenses, bank candidates, dashboard metrics. |
| CRM | Customer and supplier master data, contacts, notes, 360-degree views, graphs, history, risk indicators, data quality cleanup. |
| Intelligence | Butler conversations, grounded answers, permission-aware redaction, daily briefing, actions, report generation, entity graph, data archaeology. |
| Reports | Sales, customer, operations, inventory, and financial report categories, dashboard metric enrichment, date ranges, export modal, backend report export. |
| OCR | Drag-and-drop capture, file extraction, classification, line items, Quick Capture review, entity routing, bank statement parsing, email parsing, smart inbox processing. |
| Work management | Tasks, comments, status, blockers, projects, memberships, team board, assignments, notifications, activity history. |
| HR and access | Employees, managers, active status, access links, license associations, contribution summaries, device approval. |
| Security | RBAC, license activation, device binding, audit trail, delete approvals, path validation, sanitized logs, sensitive-field protections. |
| Settings | Company, directories, AI keys, business rules, VAT, margins, currency rates, data import, financial reports, sync, backups, support tools. |
| Deployment | Pilot audit, readiness checks, support bundles, signoff reports, license/device/employee readiness, sync support. |

## Professional Marketing Language

The following language is safe to reuse in brochures, proposals, and client presentations.

### Short Description

AsymmFlow is an offline-first ERP and business intelligence workspace for trading and engineering supply companies. It connects sales, costing, procurement, delivery, invoicing, banking, payroll, expenses, task management, customer relationships, OCR capture, and AI-assisted operational intelligence in one desktop application.

### Operational Description

AsymmFlow is designed for companies that need more than accounting software and more discipline than spreadsheets. It tracks the full operating chain from RFQ to quotation, order, supplier procurement, receiving, delivery, invoicing, payment, reconciliation, and management reporting. The system keeps the local business database available on the desktop, with optional cloud synchronization and AI services when configured.

### Intelligence Description

The Butler intelligence layer helps users ask operational questions, analyze documents, prepare reports, and move from insight to action. It uses AsymmFlow business context such as customers, suppliers, invoices, payments, offers, tasks, delivery commitments, and risk indicators, while respecting user permissions and keeping review steps in the workflow.

### OCR Description

AsymmFlow can capture business documents from PDFs, images, spreadsheets, Word files, and email formats, extract key fields, classify document types, and route reviewed data into RFQs, invoices, supplier invoices, purchase orders, delivery notes, bank statements, and document evidence records.

### Offline-First Description

AsymmFlow is built around a local operating database so users can continue core ERP work without depending on an always-on internet connection. Optional cloud sync, AI, and OCR services can be enabled where connectivity and credentials are available.

### Security Description

AsymmFlow combines role-based access, device-bound licensing, permission-filtered navigation, guarded backend endpoints, audit trails, delete approvals, and deployment readiness tools to keep operational control inside the system.

## Claim-Safe Sales Notes

Use these points when positioning the product:

- Say "offline-first" rather than "every feature works offline."
- Say "AI-assisted" rather than "fully autonomous."
- Say "structured OCR capture with review" rather than "perfect OCR."
- Say "optional cloud synchronization" rather than "cloud-dependent."
- Say "role-based access across backend and navigation" rather than "security by UI only."
- Say "service-backed modules exist for cheque register, book-bank reconciliation, FX revaluation, inventory, GRN, serial traceability, expenses, and payroll" if the sales conversation needs to distinguish currently foregrounded tabs from deeper product capability.
- Say "the current dataset demonstrates real operating depth" rather than promising a fixed number of records in every deployment.

## Capability Summary

AsymmFlow has the breadth of an ERP, the practical workflow depth of a trading operations system, and the intelligence layer of a modern document-aware business workspace. Its strongest product story is not a single feature. It is the fact that commercial, finance, procurement, delivery, people, documents, tasks, and AI context are connected in the same application.

The application can be positioned professionally as:

> An offline-first ERP and operating intelligence platform that helps trading and engineering supply teams manage the full business cycle from inquiry to cash, procurement to delivery, and document capture to management insight.
