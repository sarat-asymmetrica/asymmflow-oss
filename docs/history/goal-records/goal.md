"Refactor from Hell — Wave 1 Pipeline: Schema Extraction + Directory Scaffolding + Initial
  Domain Split

  This is a ROLLING PIPELINE. Complete each tranche in order. After each tranche, commit and
  immediately proceed to the next. Keep going until all tranches are done or you hit a STOP condition.

  STOP CONDITIONS (pause and report):
  - go build ./... fails and you can't fix it in 3 attempts
  - go test ./... regresses (new failures that weren't there before)
  - You need to make an architectural decision not covered by docs/TARGET_ARCHITECTURE.md
  - You've completed all tranches

  Read docs/TARGET_ARCHITECTURE.md and docs/GENERATIVE_REFACTOR_PLAN.md for full context.

  ═══════════════════════════════════════════
  TRANCHE 1: Create Target Directory Structure
  ═══════════════════════════════════════════

  Create the full pkg/ directory tree as specified in docs/TARGET_ARCHITECTURE.md:

  pkg/finance/{domain.go,ports.go,invoice/,payment/,banking/,expense/,payroll/,fx/,credit/,einvoice/,r
  eporting/}
  pkg/crm/{domain.go,ports.go,customer/,pipeline/,procurement/,fulfillment/,product/}
  pkg/documents/{domain.go,ports.go,pdf/,ocr/,excel/,classifier/,extraction/}
  pkg/butler/{domain.go,ports.go,chat/,intent/,fastpath/,reports/,prediction/}
  pkg/sync/{domain.go,ports.go,engine/,turso/,onedrive/,collaboration/,tally/,etl/}
  pkg/infra/{domain.go,ports.go,db/,auth/,config/,cache/,jobs/,security/,logging/,otel/,events/}

  Each domain.go starts with just 'package <name>' and a comment.
  Each ports.go starts with just 'package <name>' and a comment.
  Subdirectory packages get a minimal .go file with just the package declaration.

  Verify: go build ./...

  Commit: 'refactor(codex): scaffold target directory structure'

  ═══════════════════════════════════════════
  TRANCHE 2: Extract Type Definitions from database.go
  ═══════════════════════════════════════════

  database.go contains 90 struct definitions. Extract them into the appropriate pkg/*/domain.go files:

  - Finance types (Invoice, InvoiceItem, Payment, CreditNote, CreditNoteItem, BankStatement,
  BankTransaction, Expense, PayrollEntry, InvoiceSequence) → pkg/finance/domain.go
  - CRM types (CustomerMaster, Supplier, Offer, OfferItem, Opportunity, Order, OrderItem,
  PurchaseOrder, POItem, DeliveryNote, DNItem, GRN, GRNItem, SerialNumber, Product, Contract) →
  pkg/crm/domain.go
  - Document types (any document/classification related structs) → pkg/documents/domain.go
  - Butler types (chat, conversation, prediction related) → pkg/butler/domain.go
  - Sync types (sync state, collaboration, import related) → pkg/sync/domain.go
  - Infra types (config, auth, session, job, cache related) → pkg/infra/domain.go

  IMPORTANT: Do NOT remove the structs from database.go yet! Just COPY them to the new locations. We
  keep both alive until all references migrate. Add type aliases or re-exports in database.go if
  needed to avoid breaking imports.

  Each domain.go file should have proper package declaration and necessary imports (time, etc).

  Verify: go build ./...

  Commit: 'refactor(codex): extract domain type definitions to pkg packages'

  ═══════════════════════════════════════════
  TRANCHE 3: Create the Event Bus Foundation
  ═══════════════════════════════════════════

  Create pkg/infra/events/bus.go with:
  - Event interface { Name() string }
  - Handler type: func(ctx context.Context, event Event) error
  - Bus interface { Publish(ctx, Event) error; Subscribe(string, Handler) }
  - InMemoryBus implementation (simple sync dispatch for now)

  Create pkg/infra/events/events.go with domain event types:
  - InvoiceCreated, PaymentRecorded, OfferWon, OfferLost
  - DocumentClassified, BankStatementImported
  - DeliveryNoteCreated, SerialNumberRegistered

  Each event struct embeds a BaseEvent{Timestamp, CorrelationID} and implements Name().

  Verify: go build ./... AND write a basic test in pkg/infra/events/bus_test.go

  Commit: 'refactor(codex): implement in-process event bus'

  ═══════════════════════════════════════════
  TRANCHE 4: Create Port Interfaces for Finance Domain
  ═══════════════════════════════════════════

  In pkg/finance/ports.go, define interfaces based on the actual methods in the codebase:

  Read customer_invoice_service.go, payment_service.go, bank_reconciliation_service.go,
  expense_service.go, and the finance-related methods on *App in app.go.

  Create interfaces:
  - InvoiceRepository (CRUD for invoices + items)
  - PaymentRepository (CRUD for payments)
  - BankingRepository (statements, transactions, reconciliation records)
  - InvoiceService (business operations: create, void, credit, etc.)
  - PaymentService (record, allocate, predict)
  - BankingService (import, match, reconcile)
  - ExpenseService (create, approve, reimburse)

  Each interface should have 5-10 methods max (CME Axiom 4: Minimality).
  Method signatures should match the existing code's inputs/outputs as closely as possible.

  Verify: go build ./...

  Commit: 'refactor(codex): define finance domain port interfaces'

  ═══════════════════════════════════════════
  TRANCHE 5: Create Port Interfaces for CRM Domain
  ═══════════════════════════════════════════

  Same approach as Tranche 4 but for pkg/crm/ports.go.

  Read the relevant service files and app.go methods. Create interfaces:
  - CustomerRepository
  - OfferRepository
  - OrderRepository
  - PipelineService (opportunities, follow-ups)
  - ProcurementService (POs, GRNs)
  - FulfillmentService (delivery notes, serial numbers)

  Verify: go build ./...

  Commit: 'refactor(codex): define CRM domain port interfaces'

  ═══════════════════════════════════════════
  TRANCHE 6: Create Port Interfaces for Remaining Domains
  ═══════════════════════════════════════════

  pkg/documents/ports.go:
  - PDFGenerator, OCRService, DocumentClassifier, ExcelParser

  pkg/butler/ports.go:
  - ChatService, IntentRouter, ReportGenerator, Predictor

  pkg/sync/ports.go:
  - SyncEngine, CloudStorage, CollaborationService

  pkg/infra/ports.go:
  - Database, Cache, JobQueue, AuthManager, CryptoService

  Verify: go build ./...

  Commit: 'refactor(codex): define document, butler, sync, and infra port interfaces'

  ═══════════════════════════════════════════
  TRANCHE 7 (STRETCH): Create Taskfile.yml
  ═══════════════════════════════════════════

  Create a Taskfile.yml at the repo root with tasks:
  - build: go build ./...
  - test: go test ./... -count=1 -timeout 300s
  - test:unit: go test ./... -count=1 -timeout 300s -short
  - lint: go vet ./...
  - clean: remove build artifacts
  - frontend:build: cd frontend && npm run build
  - frontend:dev: cd frontend && npm run dev
  - audit: reference to running wave0 scripts

  Verify: task build (if 'task' is on PATH, otherwise just verify the YAML is valid)

  Commit: 'refactor(codex): add Taskfile.yml build system'

  ═══════════════════════════════════════════

  AFTER ALL TRANCHES: Run final 'go build ./...' and 'go test ./... -count=1'. Report what was
  completed and any issues encountered."