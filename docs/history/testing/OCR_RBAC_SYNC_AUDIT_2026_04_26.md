# OCR, RBAC, Sync Audit - 2026-04-26

Purpose: capture the backend checks added after the client reported delivery-note, OCR/document, costing-save, and sync reliability concerns.

## Findings Fixed

- `documents:classify` was missing from document-capable role/license grants. Manager, sales, operations, and staff now receive it consistently in seeded roles and generated license keys.
- OCR classification could fall back to a nil/`auto` type when public classification failed or when role permissions diverged. The OCR path now has an internal classifier fallback after the public permission gate.
- Classifier outputs such as `RFQ`, `PurchaseOrder`, and `BankStatement` were not normalized before routing. Backend save now normalizes them before persistence.
- OCR save-to-entity now checks downstream workflow permission before creating RFQs, invoices, POs, delivery notes, or bank-statement records.
- Production sync smoke exposed schema/type drift risks. Remote sync now drops missing remote columns, coerces boolean fields only when the remote column is boolean, skips missing remote tables, uses supplier natural-key upsert, and skips timestamp sync for tables without `updated_at`.
- Runtime DB audit found one partial active order with no line items or downstream records: `ORD-20260415-0188`. A runtime DB backup was created at `/tmp/ph_holdings_runtime_before_orphan_order_repair_20260426.db`, then the orphan order was soft-deleted.

## Verification

| Check | Result | Evidence |
| --- | --- | --- |
| Full backend suite | PASS | `go test . -count=1` -> `ok ph_holdings_app 23.098s` |
| OCR/RBAC/sync focused tests | PASS | OCR routing, seeded/license RBAC matrix, local Excel OCR, and sync-normalization tests passed |
| OCR runtime health | PASS | `https://asymmetrica-runtime.fly.dev/health` returned healthy runtime status |
| Frontend check/build | PASS | `npm run check`; `npm run build` |
| Chromium E2E | PASS | `npm run test:e2e -- --project=chromium --workers=1` -> 34/34 |
| Production binary build | PASS | `wails build` -> built `build/bin/AsymmFlow.app/Contents/MacOS/AsymmFlow` in 16.011s |
| Production startup smoke | PASS | Startup audit reported `blocking_issues=[]`; OCR/classifier initialized; RBAC roles seeded; prior sync schema/type errors did not return |
| Runtime DB invariants | PASS | active orders 187, active hollow orders 0, UUID-style customer IDs 0, active customers 381, active invoices 470 |
| Button inventory | PASS | 522 actions, 203 backend-backed, 0 suspicious/unwired |
| Runtime/manual button index | PASS | 109/109 PASS, 0 REVIEW |

## Residual Notes

- This Mac does not have an active production license, so the packaged-app smoke correctly stops at the expected activation state for protected operations.
- Optional local tools (`tesseract`, `imagemagick`, `ghostscript`, `libreoffice`) are not installed on this Mac. Fly.io OCR is healthy, and local Excel OCR is covered.
- Legacy offer-shell warnings remain non-blocking and hidden until business review.
