package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	butlerdomain "ph_holdings_app/pkg/butler"
	crm "ph_holdings_app/pkg/crm"
	shareddomain "ph_holdings_app/pkg/domain"
	finance "ph_holdings_app/pkg/finance"
	infra "ph_holdings_app/pkg/infra"
	infradb "ph_holdings_app/pkg/infra/db"
)

// =============================================================================
// INFRASTRUCTURE: UUID & SYNC BASE
// =============================================================================

// Base provides common fields for all database models including UUID primary keys,
// timestamps, soft deletes, optimistic locking (Version), and audit trail (CreatedBy).
type Base = shareddomain.Base

// =============================================================================
// MODULE: CRM & MASTER DATA
// =============================================================================

// CustomerMaster represents a customer in the CRM system with payment grading,
// AR tracking, credit limits, and relationship history.
type CustomerMaster = crm.CustomerMaster

// CustomerContact stores multiple contact persons for a customer account.
type CustomerContact = crm.CustomerContact

// SupplierContact stores multiple contact persons for a supplier account.
type SupplierContact = crm.SupplierContact

// SupplierMaster represents a supplier in the procurement system with ratings,
// lead times, payment terms, and bank details for payments.
type SupplierMaster = crm.SupplierMaster

// EntityNote stores notes for customers and suppliers
type EntityNote = crm.EntityNote

// SupplierIssue tracks instrumentation problems with suppliers
type SupplierIssue = crm.SupplierIssue

// ProductMaster stores catalog, supplier, pricing, and technical product data.
type ProductMaster = crm.ProductMaster

// =============================================================================
// MODULE: SALES & RFQ
// =============================================================================

// Offer captures quotation header data, terms, pricing, and linked line items.
type Offer = crm.Offer

// OfferItem captures a quotation line with full costing detail.
type OfferItem = crm.OfferItem

// Opportunity tracks the sales pipeline and canonical seed opportunity data.
type Opportunity = crm.Opportunity

// FollowUpTask stores CRM follow-up actions and reminders.
type FollowUpTask = crm.FollowUpTask

// OfferFollowUp schedules and tracks follow-up actions on offers.
type OfferFollowUp = crm.OfferFollowUp

// OfferNote stores freeform notes/comments on offers.
type OfferNote = crm.OfferNote

// ARAgingBucket represents AR aging analysis with risk bucketing
type ARAgingBucket = finance.ARAgingBucket

// =============================================================================
// MODULE: OPERATIONS & LOGISTICS
// =============================================================================

// Order represents a confirmed customer order with fulfillment tracking.
type Order = crm.Order

// OrderItem represents a line item in an order with fulfillment tracking.
type OrderItem = crm.OrderItem

// GradeChange records customer grade transitions.
type GradeChange = crm.GradeChange

// Shipment tracks courier deliveries for orders with tracking numbers and status.
type Shipment = crm.Shipment

// PostSaleNote tracks warranty claims, repairs, and expenses after order delivery
type PostSaleNote = crm.PostSaleNote

// DeliveryNote tracks customer deliveries with partial delivery support
type DeliveryNote = crm.DeliveryNote

type DeliveryNoteItem = crm.DeliveryNoteItem

// =============================================================================
// MODULE: COSTING ENGINE
// =============================================================================

// DBCostingSheet captures complete cost analysis for customer quotations with
// line items, Bahrain logistics costs, and margin calculations. Converts to Offers.
type DBCostingSheet = crm.DBCostingSheet

// DBCostingItem represents a line item in a costing sheet with unit costs,
// margins, and pricing calculations.
type DBCostingItem = crm.DBCostingItem

// DBCostingAdditionalCost represents ad-hoc costs in a costing sheet such as
// special handling, permits, or one-time charges.
type DBCostingAdditionalCost = crm.DBCostingAdditionalCost

// =============================================================================
// MODULE: FINANCE & ACCOUNTING
// =============================================================================

type Invoice = finance.Invoice
type DBInvoiceItem = finance.DBInvoiceItem
type InvoiceSequence = finance.InvoiceSequence

// =============================================================================
// MODULE: CREDIT NOTES (Phase 23 - E-Invoicing)
// =============================================================================

type CreditNote = finance.CreditNote
type CreditNoteItem = finance.CreditNoteItem

// =============================================================================
// MODULE: SERIAL NUMBER TRACKING (Phase 23)
// =============================================================================

// SerialNumber tracks individual serialized items through the full lifecycle:
// PO receipt (GRN) → inventory → dispatch (DN) → delivery → customer invoicing.
type SerialNumber = crm.SerialNumber

// ChartOfAccount represents a GL account in the accounting system with hierarchy,
// VAT tracking, and balance management for Tally Killer integration.
type ChartOfAccount = finance.ChartOfAccount

// JournalEntry represents a GL journal entry with auto-posting from invoices,
// payments, and reversals. Supports Tally Killer accounting automation.
type JournalEntry = finance.JournalEntry

// JournalLine represents a debit or credit line in a journal entry with
// account reference and description.
type JournalLine = finance.JournalLine

// VATReturn represents a VAT return filing for a fiscal period with calculated
// net VAT payable/receivable.
type VATReturn = finance.VATReturn

// Payment represents a customer payment against an invoice with days-to-payment
// tracking for payment prediction analytics and GL integration.
type Payment = finance.Payment

// =============================================================================
// MODULE: INVENTORY & WAREHOUSING
// =============================================================================

// InventoryItem represents stock levels for a product at a warehouse with
// reorder points, stock status, and valuation tracking.
type InventoryItem = crm.InventoryItem

// StockMovement represents an inventory transaction with direction (in/out),
// quantity, and balance tracking for audit trails.
type StockMovement = crm.StockMovement

// StockAdjustment represents a variance correction between system and physical
// counts with approval workflow and value impact tracking.
type StockAdjustment = crm.StockAdjustment

// Warehouse represents a physical storage location for inventory management.
type Warehouse = crm.Warehouse

// =============================================================================
// MODULE: INTELLIGENCE & ANALYTICS
// =============================================================================

// PredictionRecord stores payment prediction results for customers with
// confidence scores and three-regime dynamics (R1, R2, R3).
type PredictionRecord = butlerdomain.PredictionRecord

// Setting represents application configuration with categories and optional
// encryption for sensitive values.
type Setting = infra.Setting

// CurrencyExchangeRate stores exchange rates to BHD (base currency)
type CurrencyExchangeRate = finance.CurrencyExchangeRate

// UserSession represents an active user session with access/refresh tokens
// and expiry tracking for authentication management.
type UserSession = infra.UserSession

type WinProbabilityPrediction = butlerdomain.WinProbabilityPrediction

type DiscountRecommendationRecord = butlerdomain.DiscountRecommendationRecord

type CustomerSnapshot = butlerdomain.CustomerSnapshot

type ActualOutcome = butlerdomain.ActualOutcome

type CostingHistory = crm.CostingHistory

type CostingLineItemData = crm.CostingLineItemData

type PaymentPredictionAccuracy = butlerdomain.PaymentPredictionAccuracy

// =============================================================================
// MODULE: ADMINISTRATION & LOGGING
// =============================================================================

// Role represents a user role with permission sets for RBAC authorization.
type Role = infra.Role

// User represents an application user with authentication credentials,
// role assignment, and last login tracking.
type User = infra.User

// Device represents a registered installation of the application
type Device = infra.Device

// DeviceUser links users to devices (a device can have multiple users)
type DeviceUser = infra.DeviceUser

// Alert represents a system alert with severity levels and acknowledgment tracking.
type Alert = infra.Alert

// AuditLog records user actions on resources for compliance and security tracking.
type AuditLog = infra.AuditLog

// FileWatchEvent records file system change events for document monitoring.
type FileWatchEvent struct {
	Base
	FilePath  string `gorm:"index;size:1000" json:"file_path"`
	EventType string `json:"event_type"`
}

func (FileWatchEvent) TableName() string { return "file_watch_events" }

// SyncStatus tracks synchronization state for files with last sync timestamps.
type SyncStatus struct {
	Base
	FilePath string `gorm:"uniqueIndex;size:1000" json:"file_path"`
	Status   string `gorm:"index;size:50" json:"status"`

	LastSyncTime time.Time `json:"last_sync_time"`
}

func (SyncStatus) TableName() string { return "sync_status" }

// Job represents an async background job with progress tracking, retry logic,
// and status management for long-running operations.
type Job = infra.Job

// =============================================================================
// MODULE: OPERATIONS PIPELINE - PROCUREMENT
// =============================================================================

// PurchaseOrder represents a purchase order to a supplier with multi-currency
// support, approval workflow, and partial receiving tracking.
type PurchaseOrder = crm.PurchaseOrder

// PurchaseOrderItem represents a line item in a purchase order with quantities
// and pricing in both foreign currency and BHD.
type PurchaseOrderItem = crm.PurchaseOrderItem

// GoodsReceivedNote tracks receipt of goods from supplier with quality control
// workflow, rejection tracking, and warehouse assignment.
type GoodsReceivedNote = crm.GoodsReceivedNote

// GRNItem represents a line item in a GRN with quantities received, accepted,
// and rejected, with auto-calculated acceptance via BeforeSave hook.
type GRNItem = crm.GRNItem

// =============================================================================
// MODULE: OPERATIONS PIPELINE - SUPPLIER INVOICES
// =============================================================================

// SupplierInvoice represents a supplier invoice with OCR support, 3-way matching
// against PO and GRN, approval workflow, and GL integration.
type SupplierInvoice = finance.SupplierInvoice
type SupplierInvoiceItem = finance.SupplierInvoiceItem

// =============================================================================
// MODULE: CHAT PERSISTENCE (Butler AI Conversations)
// =============================================================================

// Conversation represents a Butler AI chat session with title, summary,
// and message history for persistent intelligence interactions.
type Conversation = butlerdomain.Conversation

// ChatMessage represents a single message in a conversation with role
// (user/assistant/system), content, and token usage tracking.
type ChatMessage = butlerdomain.ChatMessage

// =============================================================================
// MODULE: SUPPLIER PAYMENTS
// =============================================================================

// SupplierPayment tracks payments made to suppliers for their invoices with
// multi-currency support, payment method, and GL integration.
type SupplierPayment = finance.SupplierPayment

// =============================================================================
// MODULE: SYNC INFRASTRUCTURE
// =============================================================================

// SyncRecord tracks synchronization state between local and remote databases
// with conflict resolution (local_wins/remote_wins) and version tracking.
type SyncRecord struct {
	Base
	SyncTable     string    `gorm:"index;size:100" json:"sync_table"`
	RecordID      string    `gorm:"index;size:36" json:"record_id"`
	SyncedAt      time.Time `gorm:"index;autoUpdateTime" json:"synced_at"`
	Direction     string    `gorm:"size:10;check:direction IN ('push','pull')" json:"direction"` // "push" or "pull"
	RemoteVersion int       `gorm:"check:remote_version >= 0" json:"remote_version"`
	LocalVersion  int       `gorm:"check:local_version >= 0" json:"local_version"`
	ConflictState string    `gorm:"size:20;check:conflict_state IN ('none','local_wins','remote_wins')" json:"conflict_state"` // "none", "local_wins", "remote_wins"
}

func (SyncRecord) TableName() string { return "sync_records" }

// =============================================================================
// MODULE: TALLY DATA IMPORTS
// =============================================================================

// TallyInvoiceImport stores imported Tally invoice records with customer matching,
// status tracking, and raw data preservation for reconciliation.
type TallyInvoiceImport struct {
	Base
	ImportBatch       string    `gorm:"index;size:36" json:"import_batch"`
	Year              int       `gorm:"index;check:year >= 2000 AND year <= 2100" json:"year"`
	InvoiceNumber     string    `gorm:"index;size:100" json:"invoice_number"`
	CustomerName      string    `gorm:"index;size:255" json:"customer_name"`
	MatchedCustomerID string    `gorm:"index;size:36" json:"matched_customer_id"`
	InvoiceDate       time.Time `gorm:"autoCreateTime:false" json:"invoice_date"`
	Amount            float64   `gorm:"check:amount >= 0" json:"amount"`
	Currency          string    `gorm:"size:3" json:"currency"`
	Status            string    `gorm:"index;size:50;check:status IN ('imported','matched','duplicate','error','pending')" json:"status"` // imported, matched, duplicate, error
	RawData           string    `gorm:"type:varchar(5000)" json:"raw_data"`
}

func (TallyInvoiceImport) TableName() string { return "tally_invoice_imports" }

// TallyPurchaseImport stores imported Tally purchase records with supplier matching,
// status tracking, and raw data preservation for reconciliation.
type TallyPurchaseImport struct {
	Base
	ImportBatch       string    `gorm:"index;size:36" json:"import_batch"`
	Year              int       `gorm:"index;check:year >= 2000 AND year <= 2100" json:"year"`
	InvoiceNumber     string    `gorm:"index;size:100" json:"invoice_number"`
	SupplierName      string    `gorm:"index;size:255" json:"supplier_name"`
	MatchedSupplierID string    `gorm:"index;size:36" json:"matched_supplier_id"`
	InvoiceDate       time.Time `gorm:"autoCreateTime:false" json:"invoice_date"`
	Amount            float64   `gorm:"check:amount >= 0" json:"amount"`
	Currency          string    `gorm:"size:3" json:"currency"`
	Status            string    `gorm:"index;size:50;check:status IN ('imported','matched','duplicate','error','pending')" json:"status"` // imported, matched, duplicate, error
	RawData           string    `gorm:"type:varchar(5000)" json:"raw_data"`
}

func (TallyPurchaseImport) TableName() string { return "tally_purchase_imports" }

// =============================================================================
// MODULE: ACCOUNTING ENGINE (Tally Killer)
// =============================================================================

// AccountMapping maps transaction types to GL account codes for automatic
// journal entry generation (AR, AP, Revenue, COGS, VAT, etc.).
type AccountMapping = finance.AccountMapping

// FiscalPeriod represents a fiscal period (month) within a fiscal year with
// open/closed/locked status and closing audit trail.
type FiscalPeriod = finance.FiscalPeriod

// BankAccount represents a bank account for reconciliation with GL integration,
// balance tracking, and multi-currency support.
type BankAccount = finance.BankAccount

// BankStatement represents an imported bank statement with opening/closing
// balances, reconciliation status, and line items for matching.
type BankStatement = finance.BankStatement

// BankStatementLine represents a single line in a bank statement with
// automatic/manual matching to payments and journal entries.
type BankStatementLine = finance.BankStatementLine

// =============================================================================
// MODULE: BANK RECONCILIATION SSOT (Feature B - February 2026)
// =============================================================================

// BankLinePaymentAllocation supports split payments where one bank line
// pays multiple invoices or one invoice is paid across multiple bank lines.
type BankLinePaymentAllocation = finance.BankLinePaymentAllocation

// BankCashBalance is the SSOT for cash position per bank account per date.
type BankCashBalance = finance.BankCashBalance

// BankExpenseEntry auto-creates expense entries for bank fees, VAT, etc.
type BankExpenseEntry = finance.BankExpenseEntry

// StatementHash prevents duplicate statement imports via SHA-256 hash.
type StatementHash = finance.StatementHash

// StatementBalanceValidation bundles a balance-check result with its
// discrepancy amount (Wails 3-return marshaling workaround — see the type
// doc comment in pkg/finance/domain.go).
type StatementBalanceValidation = finance.StatementBalanceValidation

// DuplicateStatementCheck bundles a duplicate-detection result with the
// matching hash record, if any (Wails 3-return marshaling workaround).
type DuplicateStatementCheck = finance.DuplicateStatementCheck

// BookBankReconciliation for traditional book vs bank reconciliation.
type BookBankReconciliation = finance.BookBankReconciliation

// OutstandingCheque tracks cheques from issuance through clearance.
type OutstandingCheque = finance.OutstandingCheque

// DepositInTransit tracks deposits made but not yet appearing on bank statement.
type DepositInTransit = finance.DepositInTransit

// ChequeRegister tracks cheque books and sequential numbering.
type ChequeRegister = finance.ChequeRegister

// FXRate stores foreign exchange rates for multi-currency accounts.
type FXRate = finance.FXRate

// FXRevaluation tracks unrealized FX gain/loss for foreign currency accounts.
type FXRevaluation = finance.FXRevaluation

// BankStatementFile archives original PDF/CSV files for audit trail.
type BankStatementFile = finance.BankStatementFile

// BankReconciliationAuditLog provides complete audit trail for all reconciliation actions.
type BankReconciliationAuditLog = finance.BankReconciliationAuditLog

// =============================================================================
// DATABASE BACKUP & INTEGRITY
// =============================================================================

const maxBackups = 7

const (
	backupAutoEnabledSetting = "backup_auto_enabled"
	backupFrequencyDaysKey   = "backup_frequency_days"
	backupLastAtSetting      = "backup_last_at"
	backupLastPathSetting    = "backup_last_path"
)

type BackupPolicy = infra.BackupPolicy

// BackupDatabase creates an atomic backup of the SQLite database using VACUUM INTO.
// Backups are stored in a "backups/" directory next to the database file.
// Keeps the last 7 backups and deletes older ones.
func (a *App) BackupDatabase() (string, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return "", err
	}
	return a.backupDatabaseInternal()
}

func (a *App) backupDatabaseInternal() (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	sqlDB, err := a.db.DB()
	if err != nil {
		return "", fmt.Errorf("cannot access database connection: %w", err)
	}

	// Delegates to the promoted pkg/infra/db engine (Wave 2 Mission A):
	// VACUUM INTO + 0600 permissions + rotation, stem derived from the
	// database's own filename (identical output for the reference build).
	backupPath, err := (&infradb.Backuper{DB: sqlDB, Keep: maxBackups}).Backup(time.Now())
	if err != nil {
		return "", err
	}

	log.Printf("Database backup created: %s", backupPath)
	a.recordBackupCompletion(backupPath, time.Now())
	return backupPath, nil
}

func (a *App) recordBackupCompletion(path string, at time.Time) {
	if a == nil || a.db == nil {
		return
	}
	a.saveSetting(backupLastAtSetting, at.UTC().Format(time.RFC3339))
	a.saveSetting(backupLastPathSetting, path)
}

func (a *App) getSettingString(key, fallback string) string {
	if a == nil || a.db == nil {
		return fallback
	}
	var setting Setting
	if err := a.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return fallback
	}
	if strings.TrimSpace(setting.Value) == "" {
		return fallback
	}
	return strings.TrimSpace(setting.Value)
}

func (a *App) getBackupPolicyInternal() BackupPolicy {
	enabledRaw := strings.ToLower(a.getSettingString(backupAutoEnabledSetting, "true"))
	enabled := enabledRaw != "false" && enabledRaw != "0" && enabledRaw != "no"

	frequencyDays, err := strconv.Atoi(a.getSettingString(backupFrequencyDaysKey, "7"))
	if err != nil || frequencyDays < 1 {
		frequencyDays = 7
	}
	if frequencyDays > 30 {
		frequencyDays = 30
	}

	lastAt := a.getSettingString(backupLastAtSetting, "")
	lastPath := a.getSettingString(backupLastPathSetting, "")
	policy := BackupPolicy{
		AutoBackupEnabled: enabled,
		FrequencyDays:     frequencyDays,
		LastBackupAt:      lastAt,
		LastBackupPath:    lastPath,
		DueNow:            enabled,
	}

	if parsed, err := time.Parse(time.RFC3339, lastAt); err == nil {
		next := parsed.AddDate(0, 0, frequencyDays)
		policy.NextBackupDueAt = next.UTC().Format(time.RFC3339)
		policy.DueNow = enabled && !time.Now().Before(next)
	}

	return policy
}

func (a *App) GetBackupPolicy() (BackupPolicy, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return BackupPolicy{}, err
	}
	return a.getBackupPolicyInternal(), nil
}

func (a *App) SaveBackupPolicy(autoEnabled bool, frequencyDays int) (BackupPolicy, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return BackupPolicy{}, err
	}
	if frequencyDays < 1 {
		frequencyDays = 1
	}
	if frequencyDays > 30 {
		frequencyDays = 30
	}
	a.saveSetting(backupAutoEnabledSetting, fmt.Sprintf("%t", autoEnabled))
	a.saveSetting(backupFrequencyDaysKey, fmt.Sprintf("%d", frequencyDays))
	return a.getBackupPolicyInternal(), nil
}

func (a *App) RunScheduledBackupIfDue(reason string) map[string]any {
	if err := a.requirePermission("settings:update"); err != nil {
		return map[string]any{"success": false, "error": err.Error()}
	}
	return a.runScheduledBackupIfDueInternal(reason)
}

func (a *App) runScheduledBackupIfDueInternal(reason string) map[string]any {
	policy := a.getBackupPolicyInternal()
	if !policy.AutoBackupEnabled {
		return map[string]any{"success": true, "skipped": true, "reason": "auto backup disabled", "policy": policy}
	}
	if !policy.DueNow {
		return map[string]any{"success": true, "skipped": true, "reason": "not due", "policy": policy}
	}
	path, err := a.backupDatabaseInternal()
	if err != nil {
		return map[string]any{"success": false, "error": err.Error(), "policy": policy}
	}
	log.Printf("Scheduled database backup completed (%s): %s", strings.TrimSpace(reason), path)
	return map[string]any{
		"success":     true,
		"skipped":     false,
		"backup_path": path,
		"timestamp":   time.Now().Format(time.RFC3339),
		"policy":      a.getBackupPolicyInternal(),
	}
}

// (pruneOldBackups was promoted into pkg/infra/db.Backuper as part of the
// Wave 2 engine extraction; rotation now happens inside Backup.)

// runIntegrityCheck is the internal implementation (used at startup, no RBAC).
func (a *App) runIntegrityCheck() string {
	if a.db == nil {
		return "error: database not initialized"
	}

	sqlDB, err := a.db.DB()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	var result string
	if err := sqlDB.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		log.Printf("CRITICAL: Database integrity check failed: %v", err)
		return fmt.Sprintf("error: %v", err)
	}

	if result != "ok" {
		log.Printf("WARNING: Database integrity issues detected: %s", result)
	} else {
		log.Println("Database integrity check: OK")
	}

	return result
}

// RunIntegrityCheck is the exported version with RBAC guard (expensive PRAGMA).
func (a *App) RunIntegrityCheck() string {
	if err := a.requirePermission("settings:view"); err != nil {
		return "error: permission denied"
	}
	return a.runIntegrityCheck()
}

// TriggerBackup is an exported function callable from the UI to create a manual backup.
func (a *App) TriggerBackup() map[string]any {
	if err := a.requirePermission("settings:update"); err != nil {
		return map[string]any{
			"success": false,
			"error":   "Permission denied: admin access required",
		}
	}

	path, err := a.BackupDatabase()
	if err != nil {
		return map[string]any{
			"success": false,
			"error":   err.Error(),
		}
	}

	return map[string]any{
		"success":     true,
		"backup_path": path,
		"timestamp":   time.Now().Format(time.RFC3339),
	}
}

// GetBackupInfo returns information about existing backups.
func (a *App) GetBackupInfo() map[string]any {
	if err := a.requirePermission("settings:view"); err != nil {
		return map[string]any{
			"count":       0,
			"last_backup": "",
			"error":       "permission denied",
		}
	}
	// Compute backup dir the same way as BackupDatabase — next to the DB file
	backupDir := filepath.Join(".", "backups") // default
	if a.db != nil {
		if sqlDB, err := a.db.DB(); err == nil {
			var seq int
			var dbName, dbPath string
			if err := sqlDB.QueryRow("PRAGMA database_list").Scan(&seq, &dbName, &dbPath); err == nil && dbPath != "" {
				backupDir = filepath.Join(filepath.Dir(dbPath), "backups")
			}
		}
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return map[string]any{
			"count":       0,
			"last_backup": "",
		}
	}

	var backups []string
	var totalSize int64
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".db" {
			backups = append(backups, e.Name())
			if info, err := e.Info(); err == nil {
				totalSize += info.Size()
			}
		}
	}

	sort.Strings(backups)

	lastBackup := ""
	if len(backups) > 0 {
		lastBackup = backups[len(backups)-1]
	}

	return map[string]any{
		"count":         len(backups),
		"last_backup":   lastBackup,
		"total_size_mb": float64(totalSize) / (1024 * 1024),
		"backup_dir":    backupDir,
	}
}
