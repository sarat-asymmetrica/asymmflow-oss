# AsymmFlow Regression Harness Plan

Date: 2026-04-13  
Method: GitNexus-guided hotspot discovery plus source/test review

## Goal

Use GitNexus to find high-value seams, then protect them with the cheapest harness that guards a real business invariant.

## Existing Strengths

- Broad Go test coverage already exists across deployment, finance, OCR, graph, auth, config, and domain services.
- Playwright coverage already exists for finance/opportunity, order flow, pricing flow, RFQ-to-order, OCR payment flow, and sales/operations regressions.
- Deployment-specific guards already exist for DB path selection and packaged DB reconciliation.

## Recommended Next Harnesses

### 1. Deployment DB path contract harness

Priority: P0

- Invariant: packaged installs must treat `data/asymmflow.db` as a first-run seed and must use the persistent app-data DB for live writes.
- Harness type: Go integration test plus packaged-app smoke run
- Focus files:
  - [config.go](/Users/developer/projects/asymmflow/config.go:212)
  - [app.go](/Users/developer/projects/asymmflow/app.go:338)
  - [deployment_audit_test.go](/Users/developer/projects/asymmflow/deployment_audit_test.go:193)

### 2. Wails contract harness for app path and settings wiring

Priority: P1

- Invariant: frontend settings/deployment screens receive the same path semantics the backend uses internally.
- Harness type: Wails contract test or mock bridge coverage
- Focus seams:
  - `GetApplicationPaths`
  - deployment workspace data
  - settings screen path display

### 3. Invoice → payment → status transition harness

Priority: P1

- Invariant: recording payments updates outstanding amounts and invoice status exactly once and rejects overpayment/duplicate edge cases.
- Harness type: DB integration harness
- Existing base: [payment_service_test.go](/Users/developer/projects/asymmflow/payment_service_test.go:1)
- Extension target:
  - reconciliation interactions
  - partial-to-full payment transitions
  - rollback behavior on failed writes

### 4. Offer → order → invoice commercial chain harness

Priority: P1

- Invariant: downstream documents retain valid child records and header linkage after upstream edits and conversions.
- Harness type: integration or E2E smoke harness
- Existing base:
  - [frontend/tests/e2e/finance-opportunity-regressions.spec.ts](/Users/developer/projects/asymmflow/frontend/tests/e2e/finance-opportunity-regressions.spec.ts:1)
  - `frontend/tests/e2e/rfq-to-order.spec.ts`
  - `frontend/tests/e2e/order-flow.spec.ts`

### 5. Runtime foundation bootstrap harness

Priority: P1

- Invariant: startup on a client runtime DB always materializes critical collaboration, expense, payroll, and rollout tables without breaking existing data.
- Harness type: deployment/runtime DB integration harness
- Existing base:
  - [deployment_audit.go](/Users/developer/projects/asymmflow/deployment_audit.go:98)
  - [deployment_audit_test.go](/Users/developer/projects/asymmflow/deployment_audit_test.go:183)

### 6. Sync boundary harness

Priority: P2

- Invariant: SQLite remains the source of truth and sync code neither drops newly required tables nor misroutes packaged installs toward stale machine data.
- Harness type: integration harness around sync config and table inventory
- Focus seams:
  - `db_sync_service.go`
  - `db_manager.go`
  - `sync_service.go`

### 7. Packaged-app UI smoke harness

Priority: P2

- Invariant: the packaged app can launch and open the handoff-critical screens without dead data or blocking startup issues.
- Harness type: packaged-app manual or automated smoke harness
- Screen set:
  - Opportunities
  - Costing
  - Offers
  - Orders
  - Operations
  - Customer Invoices
  - Payments Received
  - Payments Made
  - Expenses
  - Payroll
  - Work
  - Deployment

## Harness Design Rules

- Prefer state-transition and data-integrity assertions over snapshots.
- Use GitNexus to choose seams, not to define the assertions.
- Derive assertions from source, schema, and domain rules.
- Tag each harness with the business invariant it protects.

## Suggested Rollout Order

1. Deployment DB path contract harness
2. Runtime foundation bootstrap harness
3. Invoice → payment → status transition harness
4. Offer → order → invoice chain harness
5. Wails contract harness
6. Sync boundary harness
7. Packaged-app UI smoke harness
