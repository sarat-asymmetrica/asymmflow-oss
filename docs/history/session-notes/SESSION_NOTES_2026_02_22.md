# Session Notes — 2026-02-22

## What Was Done

### Phase 21: Delivery Fulfillment & Partial Invoicing
Implemented the end-to-end Order -> Deliver -> Invoice pipeline for partial deliveries.

**4 gaps closed:**
1. **OrderItem quantities update on delivery** — `QuantityShipped` updated in `CreateDNFromOrder`, `QuantityDelivered` updated in `ConfirmDeliveryNote`, both using atomic SQL increments
2. **DN-based partial invoicing** — invoices use delivered quantities from DN items, not full order quantities. `CreateInvoiceFromDN(deliveryNoteID)` wrapper added. VAT calculated from invoice subtotal for partial invoices.
3. **Invoiceable quantity validation** — when order has DNs, can only invoice delivered-but-not-yet-invoiced quantities. Error returned when nothing to invoice.
4. **ProgressOrderOnInvoice partial awareness** — sets "PartiallyInvoiced" when not all items fully invoiced, "Invoiced" when complete.

### Full Inventory/Stock/Warehouse Code Removal
Acme Instrumentation is buy-to-order (process instrumentation, made-to-order). No stock is kept. Removed all inventory-related code:

- **Deleted**: `inventory_service.go` (336 lines)
- **Removed from `app.go`**: 12 API functions (~480 lines) — GetInventoryItems, GetInventoryItem, CreateInventoryItem, UpdateInventoryItem, RecordStockMovement, calculateStockStatus, GetStockMovements, CreateStockAdjustment, ApproveStockAdjustment, GetLowStockItems, GetInventoryValuation, GetWarehouses, CreateWarehouse
- **Removed from `database.go`**: 4 struct types — InventoryItem, StockMovement, StockAdjustment, Warehouse
- **Removed from `reports.go` / `report_generators.go`**: inventory report functions, StockMovementSummary, getInventoryReport
- **Cleaned `tally_importer.go`**: Balance sheet inventory = 0 (was summing empty InventoryItem table)
- **Frontend cleaned**: mocks, types, ReportsScreen, ShowcaseScreen, DashboardScreen, FinancialDashboard

**Database tables left dormant**: `inventory_items`, `stock_movements`, `stock_adjustments`, `warehouses` — empty and harmless, no migration needed.

### Files Modified
| File | Change |
|------|--------|
| `database.go` | +QuantityDelivered field, -4 inventory structs, -WarehouseID FK |
| `delivery_note_service.go` | +QuantityShipped update, +QuantityDelivered update on confirm |
| `customer_invoice_service.go` | +DN-based partial invoicing, +invoiceable qty validation, +CreateInvoiceFromDN |
| `payment_service.go` | +PartiallyInvoiced awareness in ProgressOrderOnInvoice |
| `app.go` | -12 inventory/stock/warehouse functions |
| `inventory_service.go` | DELETED (336 lines) |
| `reports.go` | -StockMovementSummary, -inventory report fields, -getInventoryReport |
| `report_generators.go` | -addInventoryReportToPDF, -addInventoryReportToExcel |
| `tally_importer.go` | inventory asset = 0 |
| `frontend/.../wailsMock.ts` | -13 inventory mocks |
| `frontend/.../types.ts` | -InventoryReportData, -'inventory' from ReportCategory |
| `frontend/.../types/index.ts` | -'inventory' from ReportCategory |
| `frontend/.../ReportsScreen.svelte` | -inventory category |
| `frontend/.../ShowcaseScreen.svelte` | -inventory tab |
| `frontend/.../DashboardScreen.svelte` | -"Low Stock Alerts" action |
| `frontend/.../FinancialDashboard.svelte` | inventory = 0 |
| `REVISION_NOTES.md` | Updated with Phase 21 + removal |

### Build Verification
- `go build ./...` — PASS
- `npm run build` (frontend) — PASS
- `wails build` — PASS (production binary: `build/bin/AsymmFlow.app`)

---

## Pending: Deployment Issues List
The team deployment was done yesterday (2026-02-21). A list of issues was reported and needs to be tackled in the next session. These were not addressed in this session.
