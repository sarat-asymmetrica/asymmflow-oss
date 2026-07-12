package main

import "ph_holdings_app/pkg/graph"

// tradingModels is the trading vertical's registered model-set: every GORM
// model the trading deployment migrates at boot. Moved verbatim from the
// inline AutoMigrate list in startup() (Wave 3 A.2) — order preserved, since
// AutoMigrate runs model-by-model and earlier failures are skipped
// individually. The schema this set produces is pinned by
// TestTradingModels_SchemaGolden; if you add/remove/reshape a model,
// regenerate the golden deliberately (see that test) so schema drift is
// always an explicit, reviewed diff.
//
// Hospitality registers its own set in overlays/hospitality; the composition
// seam (pkg/runtime/composition) migrates whichever set the vertical hands it.
func tradingModels() []any {
	return []any{
		// Predictions (existing)
		&PredictionRecord{},
		&CustomerSnapshot{},
		&ActualOutcome{},

		// Master Data
		&CustomerMaster{},
		&CustomerContact{},
		&SupplierMaster{},
		&SupplierContact{},
		&ProductMaster{},
		&EntityNote{},
		&SupplierIssue{},

		// Transactions
		&Offer{},
		&OfferItem{},
		&Order{},
		&OrderItem{},
		&Invoice{},
		&DBInvoiceItem{},
		&InvoiceSequence{}, // Race-condition-safe invoice numbering
		&Payment{},
		&CustomerReceipt{},           // P3 slice 2: customer receipt header
		&CustomerReceiptAllocation{}, // P3 slice 2: receipt→invoice-payment link
		&Shipment{},

		// Analytics
		&CostingHistory{},
		&GradeChange{},

		// Sync
		&SyncStatus{},
		&FileWatchEvent{},

		// RFQs → Costings → Offers
		&RFQData{},
		&CostingSheetData{},
		&CostingLineItemData{},
		&OfferData{},

		// Technical datasheet attachments bundled into costing/offer PDFs (I-25)
		&CostingSheetAttachment{},

		// Runtime Integration - Inbox
		&InboxDocument{},

		// Follow-up Tasks
		&FollowUpTask{},

		// Critical Alerts System (Wave 1 Agent 6)
		&Alert{},

		// SSOT Prediction Accuracy Tracking (Wave 1 Agent 5)
		&WinProbabilityPrediction{},
		&DiscountRecommendationRecord{},
		&PaymentPredictionAccuracy{},

		// Settings Persistence (Wave 1 Agent 3)
		&Setting{},

		// OCR Document Results (Wave 1 Agent 1 - OCR Pipeline)
		&OCRDocument{},
		&QuickCapture{},

		// Contract Generation System (Wave 2 Agent 5)
		&ContractTemplate{},
		&ContractClause{},
		&Contract{},

		// Entity Graph System (Customer360)
		&graph.GraphNode{},
		&graph.GraphEdge{},

		// RBAC System (Phase 1 - User Management)
		&Role{},
		&User{},
		&AuditLog{},
		&UserSession{},

		// Device Registration & Approval System
		&Device{},
		&DeviceUser{},
		&Employee{},
		&EmployeeDocument{}, // Wave 9.8 B4: employee document-expiry (visa/CPR/permit) tracking
		&EmployeeAccessLink{},
		&UserActivitySession{},
		&UserActivityEvent{},
		&UserActivityWeeklySummary{},
		&Project{},
		&ProjectMember{},
		&Notification{},
		&NotificationReceipt{},
		&DeleteApprovalRequest{},
		&EmployeeArchiveRequest{},
		&TaskItem{},
		&TaskComment{},
		&TaskActivity{},
		&CollaborativePendingOperation{},

		// Operations Pipeline (Procurement → Delivery)
		&PurchaseOrder{},
		&PurchaseOrderItem{},
		&GoodsReceivedNote{},
		&GRNItem{},
		&SupplierInvoice{},
		&SupplierInvoiceItem{},
		&DeliveryNote{},
		&DeliveryNoteItem{},

		// Inventory (Band-2: GRN receipts create stock, so these tables must
		// exist on a fresh DB — they were defined but never migrated before)
		&Warehouse{},
		&InventoryItem{},
		&StockMovement{},
		&StockAdjustment{},

		// Latent tables (Mission C): live code reads/writes these, but they
		// were in no migration set, so a fresh DB never created them —
		// rfq_comments is even deleted from by the sales pipeline.
		&RFQComment{},
		&OpportunityComment{},
		&PostSaleNote{},
		&DBCostingSheet{},
		&DBCostingItem{},
		&DBCostingAdditionalCost{},

		// Chat Persistence (Butler AI Conversations)
		&Conversation{},
		&ChatMessage{},

		// Supplier Payments
		&SupplierPayment{},

		// Offer Follow-Ups
		&OfferFollowUp{},
		&OfferNote{},

		// Sync Infrastructure
		&SyncRecord{},

		// Tally Data Imports
		&TallyInvoiceImport{},
		&TallyPurchaseImport{},

		// Phase 23: E-Invoicing & Serial Tracking
		&CreditNote{},
		&CreditNoteItem{},
		&SerialNumber{},

		// Phase 33: Pipeline Opportunities
		&Opportunity{},
		&OpportunityEditConflict{},
		&ARAgingBucket{},
		&BankLinePaymentAllocation{},

		// Wave 8 P3 slice 3: customer data-quality review ledger (Bucket F).
		// PH self-provisions this at call time; the sovereign substrate migrates
		// it here so the golden covers it (self-migration dropped in the port).
		&DataQualityReview{},
	}
}
