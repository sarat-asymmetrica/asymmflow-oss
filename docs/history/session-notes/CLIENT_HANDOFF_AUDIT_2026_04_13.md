# AsymmFlow Client Handoff Audit

Date: 2026-04-13  
Scope: client handoff readiness, deployment-path integrity, critical workflow confidence  
Method: GitNexus-guided inventory plus source review, test/build verification, and deployment packaging checks

## Executive Summary

The repo is in a strong handoff position for a deployment candidate built on 2026-04-13.

- GitNexus was used as a navigation layer to inventory the codebase and prioritize high-risk pathways.
- Final conclusions were confirmed in source and validation, not from graph output alone.
- The deployment-path issue from the prior release has an explicit guard in the packaged `.env` and passed the deployment reconciliation tests on this machine.

## Evidence Snapshot

### Graph indicated

- Repo index available in GitNexus as `ph_holdings`
- Indexed stats observed: 955 files, 22,124 symbols, 44,917 edges, 273 processes
- Critical deployment-relevant symbols and files surfaced quickly:
  - `config.go`
  - `app.go`
  - `deployment_audit.go`
  - `deployment_audit_test.go`
  - `manual_deployment_package_test.go`

### Confirmed in source

- DB path precedence is implemented in [config.go](/Users/developer/House_of_Projects/ph_holdings/config.go:212):
  - env override first
  - existing runtime app-data DB next
  - packaged DB next
  - local/CWD fallback only after those
- Startup contains packaged-launch path resolution safeguards in [app.go](/Users/developer/House_of_Projects/ph_holdings/app.go:338).
- Deployment audit bootstraps critical collaboration, expense, payroll, and rollout tables in [deployment_audit.go](/Users/developer/House_of_Projects/ph_holdings/deployment_audit.go:98).
- Packaging sanitizes license activation state and leaves the packaged DB as a first-run seed; live data is written to the persistent app-data DB in [manual_deployment_package_test.go](/Users/developer/House_of_Projects/ph_holdings/manual_deployment_package_test.go:339).

### Confirmed by validation

- `go test ./...` passed on 2026-04-13
- `npm run build` passed on 2026-04-13
- `wails build` passed on 2026-04-13
- Deployment gates passed on 2026-04-13:
  - `go test -run 'TestDeploymentDataAuditFlagsBlockingAndWarningIssues|TestDeploymentRuntimeDBBootstrapAndAudit|TestDeploymentDBCopyReconciliationAndPackaging' -v .`
  - `go test -run 'TestLoadConfigWithoutEnvFile|TestLoadConfigWithEnvVars' -v .`

## Critical Flow Assessment

### 1. Deployment and database location

Status: strong confidence

- Packaged deployments explicitly pin the DB path to `data/ph_holdings.db`.
- Runtime installs prefer the existing app-data database when it already exists, preserving activated-device state and local data across upgrades.
- Deployment reconciliation tests verified that runtime, repo, and packaged sanitized copies align on customer/order/invoice/offer counts and that packaged license keys are deactivated.

### 2. Startup and critical foundation materialization

Status: strong confidence

- Startup config fallback and DB initialization are guarded in `app.go`.
- Deployment foundation bootstrap covers collaboration, expenses, payroll, and phase rollout support in `deployment_audit.go`.
- Runtime deployment bootstrap audit passed against the machine’s runtime database.

### 3. Finance and transaction integrity

Status: moderate-to-strong confidence

- Service-level tests exist for payment recording and related transaction behavior in [payment_service_test.go](/Users/developer/House_of_Projects/ph_holdings/payment_service_test.go:1).
- Deployment audit blocks hollow invoices, hollow orders, zero-total active orders, and operational offers without items.
- Finance-related frontend regression coverage exists in [frontend/tests/e2e/finance-opportunity-regressions.spec.ts](/Users/developer/House_of_Projects/ph_holdings/frontend/tests/e2e/finance-opportunity-regressions.spec.ts:1).

### 4. Sales and commercial workflows

Status: moderate confidence

- Source and test assets indicate coverage for opportunities, costing, offers, orders, and RFQ-to-order flows.
- Existing Playwright coverage exercises:
  - opportunity notes persistence
  - supplier invoice edit/payment-entry flow
  - costing prefill from opportunity
  - offer header persistence

### 5. Operations and fulfillment

Status: moderate confidence

- Repo structure and AGENTS project summary confirm PO, GRN, delivery note, and serial traceability pathways.
- There is coverage around operations regressions and delivery-related screens, but this remains an area where manual signoff should stay strict during client handoff.

## Residual Risks

- GitNexus caller/process coverage is incomplete for some Wails-bound methods, so graph silence must not be treated as proof of no coupling.
- Frontend production build passes but emits existing Svelte accessibility and unused-selector warnings.
- Windows cross-build was attempted after the Mac build refresh; if the `.exe` cannot be refreshed in the current environment, package provenance for Windows should be recorded explicitly.
- Handoff confidence is highest for deployment-path integrity and lower for broad “all pathways are logically perfect” claims without additional manual workflow signoff.

## Recommendation

Recommendation: deploy with confidence after packaged-app manual signoff, with the deployment DB-path checklist treated as mandatory.

The highest-confidence claim supported by current evidence is:

- deployment path handling is intentionally fixed and validated
- the repo builds and tests cleanly
- the package generation flow is ready to be used for client handoff

The claim not yet supported is:

- every end-user workflow is exhaustively signoff-tested in packaged-app form on both platforms
