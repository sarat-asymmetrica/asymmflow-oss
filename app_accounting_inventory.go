package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"ph_holdings_app/pkg/inventory"
	"ph_holdings_app/pkg/kernel/money"
)

func (a *App) GetChartOfAccounts(accountType string) ([]ChartOfAccount, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var accounts []ChartOfAccount
	query := a.db.Where("deleted_at IS NULL").Order("account_code ASC")

	if accountType != "" && accountType != "All" {
		query = query.Where("account_type = ?", accountType)
	}

	if err := query.Find(&accounts).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve chart of accounts", err.Error())
	}

	log.Printf("✅ Retrieved %d accounts (type: %s)", len(accounts), accountType)
	return accounts, nil
}

// CreateAccount creates a new account in the chart of accounts
func (a *App) CreateAccount(account ChartOfAccount) (*ChartOfAccount, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate required fields
	if account.AccountCode == "" || account.AccountName == "" || account.AccountType == "" {
		return nil, newError("INVALID_INPUT", "Account code, name, and type are required", "")
	}

	// Check for duplicate account code
	var existing ChartOfAccount
	if err := a.db.Where("account_code = ? AND deleted_at IS NULL", account.AccountCode).First(&existing).Error; err == nil {
		return nil, newError("DUPLICATE_ACCOUNT", "Account code already exists", account.AccountCode)
	}

	// Set defaults
	account.IsActive = true
	account.Balance = 0
	account.CreatedBy = a.getCurrentUserID()

	if err := a.db.Create(&account).Error; err != nil {
		return nil, newError("DB_CREATE_FAILED", "Failed to create account", err.Error())
	}

	log.Printf("✅ Created account: %s - %s", account.AccountCode, account.AccountName)
	return &account, nil
}

// UpdateAccount updates an existing account
func (a *App) UpdateAccount(accountID string, updates map[string]any) error {
	if err := a.requirePermission("finance:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	if strings.TrimSpace(accountID) == "" {
		return newError("INVALID_INPUT", "Account ID is required", "")
	}

	// Mission I (I-12): the client map was applied unfiltered — balance is
	// posting-owned (journal lines only) and audit fields are server-owned.
	allowedColumns := map[string]bool{
		"account_code": true, "account_name": true, "account_type": true,
		"is_active": true, "is_vat_account": true, "vat_direction": true,
		"parent_account_id": true, "account_group": true,
	}
	filtered := make(map[string]any, len(updates))
	for key, value := range updates {
		if allowedColumns[key] {
			filtered[key] = value
		} else {
			log.Printf("⚠️ UpdateAccount: dropped non-editable column %q", key)
		}
	}
	if len(filtered) == 0 {
		return newError("INVALID_INPUT", "No editable fields in update payload", "")
	}

	if err := a.db.Model(&ChartOfAccount{}).Where("id = ?", accountID).Updates(filtered).Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to update account", err.Error())
	}

	log.Printf("✅ Updated account %s", accountID)
	return nil
}

// GetJournalEntries retrieves journal entries with optional filters
func (a *App) GetJournalEntries(fiscalYear int, fiscalPeriod int, isPosted *bool, limit int) ([]JournalEntry, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	if limit <= 0 {
		limit = 100
	}

	var entries []JournalEntry
	query := a.db.Preload("Lines").Where("deleted_at IS NULL").Order("entry_date DESC, entry_number DESC").Limit(limit)

	if fiscalYear > 0 {
		query = query.Where("fiscal_year = ?", fiscalYear)
	}
	if fiscalPeriod > 0 {
		query = query.Where("fiscal_period = ?", fiscalPeriod)
	}
	if isPosted != nil {
		query = query.Where("is_posted = ?", *isPosted)
	}

	if err := query.Find(&entries).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve journal entries", err.Error())
	}

	log.Printf("✅ Retrieved %d journal entries", len(entries))
	return entries, nil
}

// CreateJournalEntry creates a new journal entry with lines
func (a *App) CreateJournalEntry(entry JournalEntry) (*JournalEntry, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate
	if len(entry.Lines) == 0 {
		return nil, newError("INVALID_INPUT", "Journal entry must have at least one line", "")
	}

	// Calculate totals
	var debitTotal, creditTotal float64
	for _, line := range entry.Lines {
		debitTotal += line.Debit
		creditTotal += line.Credit
	}

	// Validate balanced entry
	if debitTotal != creditTotal {
		return nil, newError("UNBALANCED_ENTRY",
			fmt.Sprintf("Debits (%.2f) must equal credits (%.2f)", debitTotal, creditTotal), "")
	}

	// Generate entry number if not provided
	if entry.EntryNumber == "" {
		year := time.Now().Year()
		var count int64
		a.db.Model(&JournalEntry{}).Where("fiscal_year = ?", year).Count(&count)
		entry.EntryNumber = fmt.Sprintf("JE-%d-%04d", year, count+1)
	}

	// Set totals and defaults
	entry.DebitTotal = debitTotal
	entry.CreditTotal = creditTotal
	entry.IsPosted = false
	entry.CreatedBy = a.getCurrentUserID()

	// Set fiscal period
	if entry.FiscalYear == 0 {
		entry.FiscalYear = entry.EntryDate.Year()
	}
	if entry.FiscalPeriod == 0 {
		entry.FiscalPeriod = int(entry.EntryDate.Month())
	}

	// Create in transaction
	tx := a.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&entry).Error; err != nil {
		tx.Rollback()
		return nil, newError("DB_CREATE_FAILED", "Failed to create journal entry", err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		return nil, newError("DB_COMMIT_FAILED", "Failed to commit journal entry", err.Error())
	}

	log.Printf("✅ Created journal entry: %s (%.2f BHD)", entry.EntryNumber, debitTotal)
	return &entry, nil
}

// PostJournalEntry posts a journal entry (makes it permanent)
func (a *App) PostJournalEntry(entryID uint) error {
	if err := a.requirePermission("finance:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var entry JournalEntry
	if err := a.db.Preload("Lines.Account").First(&entry, entryID).Error; err != nil {
		return newError("ENTRY_NOT_FOUND", "Journal entry not found", err.Error())
	}

	if entry.IsPosted {
		return newError("ALREADY_POSTED", "Journal entry is already posted", "")
	}

	// Start transaction
	tx := a.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update account balances
	for _, line := range entry.Lines {
		var account ChartOfAccount
		if err := tx.First(&account, line.AccountID).Error; err != nil {
			tx.Rollback()
			return newError("ACCOUNT_NOT_FOUND", fmt.Sprintf("Account #%s not found", line.AccountID), err.Error())
		}

		// Update balance based on account type
		// Debit increases: Assets, Expenses
		// Credit increases: Liabilities, Equity, Revenue
		var balanceChange float64
		if account.AccountType == "Asset" || account.AccountType == "Expense" {
			balanceChange = line.Debit - line.Credit
		} else {
			balanceChange = line.Credit - line.Debit
		}

		account.Balance += balanceChange
		if err := tx.Save(&account).Error; err != nil {
			tx.Rollback()
			return newError("DB_UPDATE_FAILED", "Failed to update account balance", err.Error())
		}
	}

	// Mark entry as posted
	now := time.Now()
	entry.IsPosted = true
	entry.PostedAt = &now
	entry.PostedBy = a.getCurrentUserID()

	if err := tx.Save(&entry).Error; err != nil {
		tx.Rollback()
		return newError("DB_UPDATE_FAILED", "Failed to post journal entry", err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		return newError("DB_COMMIT_FAILED", "Failed to commit posting", err.Error())
	}

	log.Printf("✅ Posted journal entry: %s", entry.EntryNumber)
	return nil
}

// ReverseJournalEntry creates a reversal entry for corrections (P1 FIX)
// Creates a new journal entry with reversed debits/credits and links to original
func (a *App) ReverseJournalEntry(entryID string, reason string) (*JournalEntry, error) {
	if err := a.requirePermission("finance:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Load original entry with lines
	var originalEntry JournalEntry
	if err := a.db.Preload("Lines").Where("id = ?", entryID).First(&originalEntry).Error; err != nil {
		return nil, newError("ENTRY_NOT_FOUND", "Original journal entry not found", err.Error())
	}

	// Verify entry is posted (can only reverse posted entries)
	if !originalEntry.IsPosted {
		return nil, newError("INVALID_STATE", "Can only reverse posted journal entries", "")
	}

	// Check if already reversed
	if originalEntry.ReversedByID != "" {
		return nil, newError("ALREADY_REVERSED",
			fmt.Sprintf("Entry already reversed by %s", originalEntry.ReversedByID), "")
	}

	// Create reversal entry
	now := time.Now()
	reversalEntry := JournalEntry{
		Base: Base{
			ID:        uuid.New().String(),
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: a.getCurrentUserID(),
		},
		EntryDate:       now,
		Description:     fmt.Sprintf("REVERSAL: %s - %s", originalEntry.Description, reason),
		FiscalYear:      now.Year(),
		FiscalPeriod:    int(now.Month()),
		IsAutoGenerated: true,
		ReversesID:      originalEntry.ID,
		SourceType:      "reversal",
		SourceID:        originalEntry.ID,
		DebitTotal:      originalEntry.CreditTotal, // Swap totals
		CreditTotal:     originalEntry.DebitTotal,
		IsPosted:        false, // Created as draft, needs posting
	}

	// Generate entry number
	var count int64
	a.db.Model(&JournalEntry{}).Where("fiscal_year = ?", now.Year()).Count(&count)
	reversalEntry.EntryNumber = fmt.Sprintf("JE-%d-%04d-REV", now.Year(), count+1)

	// Create reversed lines (swap debits and credits)
	for _, originalLine := range originalEntry.Lines {
		reversalLine := JournalLine{
			Base: Base{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			AccountID:   originalLine.AccountID,
			AccountName: originalLine.AccountName,
			Debit:       originalLine.Credit, // Swap
			Credit:      originalLine.Debit,  // Swap
			Description: fmt.Sprintf("Reversal: %s", originalLine.Description),
		}
		reversalEntry.Lines = append(reversalEntry.Lines, reversalLine)
	}

	// Create reversal entry in database
	tx := a.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&reversalEntry).Error; err != nil {
		tx.Rollback()
		return nil, newError("DB_CREATE_FAILED", "Failed to create reversal entry", err.Error())
	}

	// Update original entry to mark as reversed
	if err := tx.Model(&originalEntry).Update("reversed_by_id", reversalEntry.ID).Error; err != nil {
		tx.Rollback()
		return nil, newError("DB_UPDATE_FAILED", "Failed to mark original entry as reversed", err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		return nil, newError("DB_COMMIT_FAILED", "Failed to commit reversal", err.Error())
	}

	log.Printf("✅ Created reversal entry %s for original entry %s (Reason: %s)",
		reversalEntry.EntryNumber, originalEntry.EntryNumber, reason)

	return &reversalEntry, nil
}

// ValidateJournalEntryBalance validates that debits equal credits (P1 FIX)
// Can be called before posting to ensure balanced entry
func (a *App) ValidateJournalEntryBalance(entryID string) (bool, float64, float64, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return false, 0, 0, err
	}
	if a.db == nil {
		return false, 0, 0, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var entry JournalEntry
	if err := a.db.Preload("Lines").Where("id = ?", entryID).First(&entry).Error; err != nil {
		return false, 0, 0, newError("ENTRY_NOT_FOUND", "Journal entry not found", err.Error())
	}

	var debitTotal, creditTotal float64
	for _, line := range entry.Lines {
		debitTotal += line.Debit
		creditTotal += line.Credit
	}

	// Round to 3 decimals (BHD precision)
	debitTotal = math.Round(debitTotal*1000) / 1000
	creditTotal = math.Round(creditTotal*1000) / 1000

	isBalanced := (debitTotal == creditTotal)

	if !isBalanced {
		log.Printf("⚠️ UNBALANCED ENTRY: %s - Debits: %.3f BHD, Credits: %.3f BHD (Diff: %.3f BHD)",
			entry.EntryNumber, debitTotal, creditTotal, debitTotal-creditTotal)
	}

	return isBalanced, debitTotal, creditTotal, nil
}

// ARAgingReport represents accounts receivable aging
type ARAgingReport struct {
	Current     float64         `json:"current"`
	Days30      float64         `json:"days_30"`
	Days60      float64         `json:"days_60"`
	Days90      float64         `json:"days_90"`
	Days120Plus float64         `json:"days_120_plus"`
	Total       float64         `json:"total"`
	Details     []ARAgingDetail `json:"details,omitempty"`
}

type ARAgingDetail struct {
	CustomerID   string  `json:"customer_id"`
	CustomerName string  `json:"customer_name"`
	InvoiceID    string  `json:"invoice_id"`
	InvoiceNum   string  `json:"invoice_number"`
	InvoiceDate  string  `json:"invoice_date"`
	DueDate      string  `json:"due_date"`
	Amount       float64 `json:"amount"`
	DaysOverdue  int     `json:"days_overdue"`
	AgingBucket  string  `json:"aging_bucket"`
}

// GetARAgingReport generates accounts receivable aging report
func (a *App) GetARAgingReport() (*ARAgingReport, error) {
	// P0 FIX: Finance-only function
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	return a.buildARAgingReport(nil, nil)
}

// GetDashboardARAgingReportYTD returns all outstanding AR aging totals for the
// dashboard. No year filter is applied — aging reflects what is currently owed
// regardless of invoice date, so multi-year receivables are always visible.
// Details are stripped: the dashboard widget only renders bucket totals.
func (a *App) GetDashboardARAgingReportYTD() (*ARAgingReport, error) {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return nil, err
	}
	report, err := a.buildARAgingReport(nil, nil)
	if report != nil {
		report.Details = nil
	}
	return report, err
}

// buildARAgingReport is the shared aging engine (ported from deployed PH's
// refactor): open-balance invoices are normalized through the customer-invoice
// payment-state policy so Draft/Cancelled/Void/Proforma rows with stale
// outstanding never inflate the buckets.
func (a *App) buildARAgingReport(start *time.Time, end *time.Time) (*ARAgingReport, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	report := &ARAgingReport{
		Details: make([]ARAgingDetail, 0),
	}

	// Query all open-balance invoices and normalize collectible/open state in-memory.
	var invoices []Invoice
	query := a.db.Where("outstanding_bhd > 0")
	if start != nil && end != nil {
		query = query.Where("invoice_date >= ? AND invoice_date < ?", *start, *end)
	}
	err := query.Order("due_date ASC").Find(&invoices).Error

	if err != nil {
		log.Printf("⚠️ AR aging query failed: %v", err)
		return report, nil // Return empty report on error
	}

	// Calculate aging buckets
	now := time.Now()
	for _, inv := range invoices {
		state := customerInvoicePaymentStateFromInvoice(inv, now)
		if !state.IsCollectible {
			continue
		}

		daysOverdue := int(now.Sub(inv.DueDate).Hours() / 24)
		if daysOverdue < 0 {
			daysOverdue = 0 // Not yet due
		}

		amount := state.OutstandingBHD
		detail := ARAgingDetail{
			CustomerID:   inv.CustomerID,
			CustomerName: inv.CustomerName,
			InvoiceID:    inv.ID,
			InvoiceNum:   inv.InvoiceNumber,
			InvoiceDate:  inv.InvoiceDate.Format("2006-01-02"),
			DueDate:      inv.DueDate.Format("2006-01-02"),
			Amount:       amount,
			DaysOverdue:  daysOverdue,
		}

		// Categorize into aging buckets
		switch {
		case daysOverdue == 0:
			detail.AgingBucket = "Current"
			report.Current += amount
		case daysOverdue <= 30:
			detail.AgingBucket = "30+ Days"
			report.Days30 += amount
		case daysOverdue <= 60:
			detail.AgingBucket = "60+ Days"
			report.Days60 += amount
		case daysOverdue <= 90:
			detail.AgingBucket = "90+ Days"
			report.Days90 += amount
		default:
			detail.AgingBucket = "120+ Days"
			report.Days120Plus += amount
		}

		report.Details = append(report.Details, detail)
	}

	report.Total = report.Current + report.Days30 + report.Days60 + report.Days90 + report.Days120Plus

	log.Printf("✅ Generated AR aging report: Total BHD %.2f across %d invoices", report.Total, len(invoices))
	return report, nil
}

// SalesPipelineData represents RFQ pipeline distribution by status
type SalesPipelineData struct {
	Stage string  `json:"stage"`
	Count int     `json:"count"`
	Value float64 `json:"value"`
	Color string  `json:"color"`
}

// GetSalesPipeline returns RFQ distribution by status for pipeline visualization
func (a *App) GetSalesPipeline() ([]SalesPipelineData, error) {
	if err := a.requirePermission("sales:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Query RFQs grouped by status
	var results []struct {
		Status string
		Count  int64
		Value  float64
	}

	err := a.db.Model(&RFQData{}).
		Select("status, COUNT(*) as count, SUM(value) as value").
		Group("status").
		Order("count DESC").
		Scan(&results).Error

	if err != nil {
		log.Printf("⚠️ Sales pipeline query failed: %v", err)
		return []SalesPipelineData{}, nil
	}

	// Map to pipeline data with colors
	stageColors := map[string]string{
		"pending":     "#F59E0B", // Amber
		"qualified":   "#3B82F6", // Blue
		"proposal":    "#8B5CF6", // Purple
		"negotiation": "#EC4899", // Pink
		"won":         "#10B981", // Green
		"lost":        "#EF4444", // Red
	}

	pipeline := make([]SalesPipelineData, 0)
	for _, r := range results {
		color := stageColors[r.Status]
		if color == "" {
			color = "#6B7280" // Gray for unknown
		}

		pipeline = append(pipeline, SalesPipelineData{
			Stage: r.Status,
			Count: int(r.Count),
			Value: r.Value,
			Color: color,
		})
	}

	log.Printf("✅ Generated sales pipeline: %d stages", len(pipeline))
	return pipeline, nil
}

// GetDashboardPipelineByStageYTD returns current-activity-year pipeline stage
// totals for all dashboard roles (ported from deployed PH). Opportunities are
// normalized/deduplicated the same way the dashboard stats path does.
func (a *App) GetDashboardPipelineByStageYTD() ([]SalesPipelineData, error) {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	year, yearStart, yearEnd := a.dashboardYTDWindow()
	var raw []Opportunity
	if err := a.db.Find(&raw).Error; err != nil {
		log.Printf("⚠️ Dashboard pipeline query failed: %v", err)
		return []SalesPipelineData{}, nil
	}

	grouped := make(map[string]*SalesPipelineData)
	for _, opp := range raw {
		normalized := normalizeOpportunityForList(opp)
		if shouldSuppressSyntheticOCR(normalized) || !opportunityInDashboardYTD(normalized, year, yearStart, yearEnd) {
			continue
		}

		stage := strings.TrimSpace(normalized.Stage)
		if stage == "" {
			stage = "Unstaged"
		}
		key := strings.ToLower(stage)
		row, exists := grouped[key]
		if !exists {
			row = &SalesPipelineData{
				Stage: stage,
				Color: dashboardStageColor(stage),
			}
			grouped[key] = row
		}
		row.Count++
		row.Value += normalized.RevenueBHD
	}

	pipeline := make([]SalesPipelineData, 0, len(grouped))
	for _, row := range grouped {
		pipeline = append(pipeline, *row)
	}
	sort.Slice(pipeline, func(i, j int) bool {
		if pipeline[i].Count == pipeline[j].Count {
			return pipeline[i].Value > pipeline[j].Value
		}
		return pipeline[i].Count > pipeline[j].Count
	})

	log.Printf("Retrieved dashboard pipeline stage data for %d stages (YTD %d)", len(pipeline), year)
	return pipeline, nil
}

func dashboardStageColor(stage string) string {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "qualified":
		return "#0EA5E9"
	case "proposal", "quoted":
		return "#6366F1"
	case "negotiation":
		return "#F59E0B"
	case "won":
		return "#10B981"
	case "lost":
		return "#EF4444"
	default:
		return "#6B7280"
	}
}

// InventoryPendingFulfillmentRow is one order line in the pending-fulfillment
// report: ordered vs delivered vs invoiced quantities, with stock availability
// and shortage computed against the inventory catalog.
type InventoryPendingFulfillmentRow struct {
	OrderID           string  `json:"order_id"`
	OrderNumber       string  `json:"order_number"`
	CustomerName      string  `json:"customer_name"`
	ProductCode       string  `json:"product_code"`
	Description       string  `json:"description"`
	OrderedQuantity   float64 `json:"ordered_quantity"`
	DeliveredQuantity float64 `json:"delivered_quantity"`
	InvoicedQuantity  float64 `json:"invoiced_quantity"`
	PendingQuantity   float64 `json:"pending_quantity"`
	AvailableQuantity float64 `json:"available_quantity"`
	ShortageQuantity  float64 `json:"shortage_quantity"`
	Status            string  `json:"status"`
}

// GetInventoryPendingFulfillmentReport lists order lines still awaiting
// delivery, with availability/shortage against current stock (ported from
// deployed PH's user-feedback hardening service).
func (a *App) GetInventoryPendingFulfillmentReport(limit int) ([]InventoryPendingFulfillmentRow, error) {
	if err := a.requirePermission("inventory:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 || limit > 1000 {
		limit = 500
	}

	var orders []Order
	if err := a.db.Preload("Items").Order("order_date DESC, created_at DESC").Limit(limit).Find(&orders).Error; err != nil {
		return nil, fmt.Errorf("failed to load order fulfillment: %w", err)
	}

	rows := make([]InventoryPendingFulfillmentRow, 0)
	for _, order := range orders {
		for _, item := range order.Items {
			delivered := item.QuantityShipped
			if delivered == 0 {
				var dnDelivered float64
				_ = a.db.Model(&DeliveryNoteItem{}).
					Where("order_item_id = ?", item.ID).
					Select("COALESCE(SUM(quantity_delivered), 0)").
					Scan(&dnDelivered).Error
				delivered = dnDelivered
			}
			pending := item.Quantity - delivered
			if pending < 0 {
				pending = 0
			}
			productCode := firstNonEmptyString(item.ProductCode, item.Model)
			var available float64
			if strings.TrimSpace(productCode) != "" {
				_ = a.db.Model(&InventoryItem{}).
					Where("deleted_at IS NULL AND is_active = ? AND (product_code = ? OR product_id = ?)", true, productCode, item.ProductID).
					Select("COALESCE(SUM(quantity_available), 0)").
					Scan(&available).Error
			}
			shortage := pending - available
			if shortage < 0 {
				shortage = 0
			}
			rows = append(rows, InventoryPendingFulfillmentRow{
				OrderID:           order.ID,
				OrderNumber:       order.OrderNumber,
				CustomerName:      order.CustomerName,
				ProductCode:       productCode,
				Description:       firstNonEmptyString(item.Description, item.Equipment, item.Specification),
				OrderedQuantity:   item.Quantity,
				DeliveredQuantity: delivered,
				InvoicedQuantity:  item.QuantityInvoiced,
				PendingQuantity:   pending,
				AvailableQuantity: available,
				ShortageQuantity:  shortage,
				Status:            order.Status,
			})
		}
	}
	return rows, nil
}

// GetInventoryMovementsWorkspace returns the recent stock-movement feed for the
// inventory movements view (delegates to the inventory:view-gated
// GetStockMovements with an unfiltered window).
func (a *App) GetInventoryMovementsWorkspace(limit int) ([]StockMovement, error) {
	if limit <= 0 || limit > 1000 {
		limit = 250
	}
	return a.GetStockMovements(nil, "All", time.Time{}, time.Time{}, limit)
}

// GetAPAgingReport generates accounts payable aging report
// Based on unpaid supplier invoices, grouped by due date buckets
func (a *App) GetAPAgingReport() (*ARAgingReport, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	report := &ARAgingReport{
		Details: make([]ARAgingDetail, 0),
	}

	// Query unpaid/outstanding supplier invoices
	// Note: SupplierInvoice doesn't have OutstandingBHD field, so we use TotalBHD for unpaid invoices
	var invoices []SupplierInvoice
	err := a.db.Where("payment_status != ? AND deleted_at IS NULL", "Paid").
		Order("due_date ASC").
		Find(&invoices).Error

	if err != nil {
		log.Printf("⚠️ AP aging query failed: %v", err)
		return report, nil // Return empty report on error
	}

	// Calculate aging buckets (same logic as AR aging, but for payables)
	now := time.Now()
	for _, inv := range invoices {
		daysOverdue := int(now.Sub(inv.DueDate).Hours() / 24)
		if daysOverdue < 0 {
			daysOverdue = 0 // Not yet due
		}

		// Use TotalBHD for unpaid invoices
		amount := inv.TotalBHD
		detail := ARAgingDetail{
			CustomerID:   inv.SupplierID,   // Using same struct, but for suppliers
			CustomerName: inv.SupplierName, // Supplier name in customer field
			InvoiceID:    inv.ID,
			InvoiceNum:   inv.InvoiceNumber,
			InvoiceDate:  inv.InvoiceDate.Format("2006-01-02"),
			DueDate:      inv.DueDate.Format("2006-01-02"),
			Amount:       amount,
			DaysOverdue:  daysOverdue,
		}

		// Categorize into aging buckets (same as AR)
		switch {
		case daysOverdue == 0:
			detail.AgingBucket = "Current"
			report.Current += amount
		case daysOverdue <= 30:
			detail.AgingBucket = "30+ Days"
			report.Days30 += amount
		case daysOverdue <= 60:
			detail.AgingBucket = "60+ Days"
			report.Days60 += amount
		case daysOverdue <= 90:
			detail.AgingBucket = "90+ Days"
			report.Days90 += amount
		default:
			detail.AgingBucket = "120+ Days"
			report.Days120Plus += amount
		}

		report.Details = append(report.Details, detail)
	}

	// Calculate total
	report.Total = report.Current + report.Days30 + report.Days60 + report.Days90 + report.Days120Plus

	log.Printf("✅ Generated AP aging report: %d unpaid supplier invoices, Total: %.3f BHD",
		len(invoices), report.Total)
	return report, nil
}

// GetVATReturns retrieves VAT returns
func (a *App) GetVATReturns(fiscalYear int, status string) ([]VATReturn, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var returns []VATReturn
	query := a.db.Where("deleted_at IS NULL").Order("period_start DESC")

	if fiscalYear > 0 {
		query = query.Where("fiscal_year = ?", fiscalYear)
	}
	if status != "" && status != "All" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&returns).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve VAT returns", err.Error())
	}

	log.Printf("✅ Retrieved %d VAT returns", len(returns))
	return returns, nil
}

// GenerateVATReturn generates a VAT return for a period
func (a *App) GenerateVATReturn(periodStart, periodEnd time.Time) (*VATReturn, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	year := periodStart.Year()
	quarter := (int(periodStart.Month())-1)/3 + 1

	// Placeholder for actual calculation logic
	totalOutputVAT := 0.0
	totalInputVAT := 0.0

	// Create return object
	vatReturnObj := VATReturn{
		ReturnNumber: fmt.Sprintf("VAT-%d-Q%d", year, quarter),
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		FiscalYear:   year,
		Quarter:      quarter,
		NetVAT:       totalOutputVAT - totalInputVAT,
		Status:       "Draft",
	}

	if err := a.db.Create(&vatReturnObj).Error; err != nil {
		return nil, newError("DB_CREATE_FAILED", "Failed to create VAT return", err.Error())
	}

	return &vatReturnObj, nil
}

// FileVATReturn marks a VAT return as filed
func (a *App) FileVATReturn(returnID uint) error {
	if err := a.requirePermission("finance:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	now := time.Now()
	updates := map[string]any{
		"status":   "Filed",
		"filed_at": now,
		"filed_by": a.getCurrentUserID(),
	}

	if err := a.db.Model(&VATReturn{}).Where("id = ?", returnID).Updates(updates).Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to file VAT return", err.Error())
	}

	log.Printf("✅ Filed VAT return #%d", returnID)
	return nil
}

// ProfitLossReport represents a P&L statement
type ProfitLossReport struct {
	PeriodStart   string           `json:"period_start"`
	PeriodEnd     string           `json:"period_end"`
	Revenue       []AccountBalance `json:"revenue"`
	TotalRevenue  float64          `json:"total_revenue"`
	COGS          []AccountBalance `json:"cogs"`
	TotalCOGS     float64          `json:"total_cogs"`
	GrossProfit   float64          `json:"gross_profit"`
	GrossMargin   float64          `json:"gross_margin"`
	Expenses      []AccountBalance `json:"expenses"`
	TotalExpenses float64          `json:"total_expenses"`
	NetIncome     float64          `json:"net_income"`
	NetMargin     float64          `json:"net_margin"`
}

type AccountBalance struct {
	AccountCode string  `json:"account_code"`
	AccountName string  `json:"account_name"`
	Balance     float64 `json:"balance"`
}

// GetProfitLoss generates a Profit & Loss statement
func (a *App) GetProfitLoss(periodStart, periodEnd time.Time) (*ProfitLossReport, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	report := &ProfitLossReport{
		PeriodStart: periodStart.Format("2006-01-02"),
		PeriodEnd:   periodEnd.Format("2006-01-02"),
		Revenue:     make([]AccountBalance, 0),
		COGS:        make([]AccountBalance, 0),
		Expenses:    make([]AccountBalance, 0),
	}

	// Get revenue accounts
	var revenueAccounts []ChartOfAccount
	if err := a.db.Where("account_type = ? AND is_active = ? AND deleted_at IS NULL", "Revenue", true).
		Find(&revenueAccounts).Error; err == nil {
		for _, acc := range revenueAccounts {
			report.Revenue = append(report.Revenue, AccountBalance{
				AccountCode: acc.AccountCode,
				AccountName: acc.AccountName,
				Balance:     acc.Balance,
			})
			report.TotalRevenue += acc.Balance
		}
	}

	// Get expense accounts (filter COGS separately if needed)
	var expenseAccounts []ChartOfAccount
	if err := a.db.Where("account_type = ? AND is_active = ? AND deleted_at IS NULL", "Expense", true).
		Find(&expenseAccounts).Error; err == nil {
		for _, acc := range expenseAccounts {
			// Separate COGS from other expenses
			if strings.Contains(strings.ToLower(acc.AccountName), "cost of goods") ||
				strings.Contains(strings.ToLower(acc.AccountName), "cogs") {
				report.COGS = append(report.COGS, AccountBalance{
					AccountCode: acc.AccountCode,
					AccountName: acc.AccountName,
					Balance:     acc.Balance,
				})
				report.TotalCOGS += acc.Balance
			} else {
				report.Expenses = append(report.Expenses, AccountBalance{
					AccountCode: acc.AccountCode,
					AccountName: acc.AccountName,
					Balance:     acc.Balance,
				})
				report.TotalExpenses += acc.Balance
			}
		}
	}

	// Calculate metrics
	report.GrossProfit = report.TotalRevenue - report.TotalCOGS
	if report.TotalRevenue > 0 {
		report.GrossMargin = report.GrossProfit / report.TotalRevenue
	}

	report.NetIncome = report.GrossProfit - report.TotalExpenses
	if report.TotalRevenue > 0 {
		report.NetMargin = report.NetIncome / report.TotalRevenue
	}

	log.Printf("✅ Generated P&L: Revenue %.2f, Net Income %.2f", report.TotalRevenue, report.NetIncome)
	return report, nil
}

// BalanceSheetReport represents a balance sheet
type BalanceSheetReport struct {
	AsOfDate         string           `json:"as_of_date"`
	Assets           []AccountBalance `json:"assets"`
	TotalAssets      float64          `json:"total_assets"`
	Liabilities      []AccountBalance `json:"liabilities"`
	TotalLiabilities float64          `json:"total_liabilities"`
	Equity           []AccountBalance `json:"equity"`
	TotalEquity      float64          `json:"total_equity"`
}

// GetBalanceSheet generates a balance sheet
func (a *App) GetBalanceSheet(asOfDate time.Time) (*BalanceSheetReport, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	report := &BalanceSheetReport{
		AsOfDate:    asOfDate.Format("2006-01-02"),
		Assets:      make([]AccountBalance, 0),
		Liabilities: make([]AccountBalance, 0),
		Equity:      make([]AccountBalance, 0),
	}

	// Get assets
	var assets []ChartOfAccount
	if err := a.db.Where("account_type = ? AND is_active = ? AND deleted_at IS NULL", "Asset", true).
		Find(&assets).Error; err == nil {
		for _, acc := range assets {
			report.Assets = append(report.Assets, AccountBalance{
				AccountCode: acc.AccountCode,
				AccountName: acc.AccountName,
				Balance:     acc.Balance,
			})
			report.TotalAssets += acc.Balance
		}
	}

	// Get liabilities
	var liabilities []ChartOfAccount
	if err := a.db.Where("account_type = ? AND is_active = ? AND deleted_at IS NULL", "Liability", true).
		Find(&liabilities).Error; err == nil {
		for _, acc := range liabilities {
			report.Liabilities = append(report.Liabilities, AccountBalance{
				AccountCode: acc.AccountCode,
				AccountName: acc.AccountName,
				Balance:     acc.Balance,
			})
			report.TotalLiabilities += acc.Balance
		}
	}

	// Get equity
	var equity []ChartOfAccount
	if err := a.db.Where("account_type = ? AND is_active = ? AND deleted_at IS NULL", "Equity", true).
		Find(&equity).Error; err == nil {
		for _, acc := range equity {
			report.Equity = append(report.Equity, AccountBalance{
				AccountCode: acc.AccountCode,
				AccountName: acc.AccountName,
				Balance:     acc.Balance,
			})
			report.TotalEquity += acc.Balance
		}
	}

	log.Printf("✅ Generated Balance Sheet: Assets %.2f, Liabilities %.2f, Equity %.2f",
		report.TotalAssets, report.TotalLiabilities, report.TotalEquity)
	return report, nil
}

// SeedDefaultChartOfAccounts creates a basic chart of accounts for Bahrain trading company
func (a *App) SeedDefaultChartOfAccounts() error {
	// SECURITY: Admin-only permission for seed functions
	if err := a.requirePermission("*"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Check if accounts already exist
	var count int64
	a.db.Model(&ChartOfAccount{}).Count(&count)
	if count > 0 {
		log.Printf("ℹ️  Chart of accounts already seeded (%d accounts)", count)
		return nil
	}

	defaultAccounts := []ChartOfAccount{
		// Assets
		{AccountCode: "1000", AccountName: "Cash", AccountType: "Asset", IsActive: true},
		{AccountCode: "1100", AccountName: "Petty Cash", AccountType: "Asset", IsActive: true},
		{AccountCode: "1200", AccountName: "Accounts Receivable", AccountType: "Asset", IsActive: true},
		{AccountCode: "1300", AccountName: "Inventory", AccountType: "Asset", IsActive: true},
		{AccountCode: "1500", AccountName: "Fixed Assets", AccountType: "Asset", IsActive: true},
		{AccountCode: "1510", AccountName: "Accumulated Depreciation", AccountType: "Asset", IsActive: true},

		// Liabilities
		{AccountCode: "2000", AccountName: "Accounts Payable", AccountType: "Liability", IsActive: true},
		{AccountCode: "2100", AccountName: "VAT Payable", AccountType: "Liability", IsActive: true},
		{AccountCode: "2200", AccountName: "Accrued Expenses", AccountType: "Liability", IsActive: true},
		{AccountCode: "2500", AccountName: "Long-term Debt", AccountType: "Liability", IsActive: true},

		// Equity
		{AccountCode: "3000", AccountName: "Capital", AccountType: "Equity", IsActive: true},
		{AccountCode: "3100", AccountName: "Retained Earnings", AccountType: "Equity", IsActive: true},
		{AccountCode: "3200", AccountName: "Current Year Earnings", AccountType: "Equity", IsActive: true},

		// Revenue
		{AccountCode: "4000", AccountName: "Sales Revenue", AccountType: "Revenue", IsActive: true},
		{AccountCode: "4100", AccountName: "Service Revenue", AccountType: "Revenue", IsActive: true},
		{AccountCode: "4900", AccountName: "Other Income", AccountType: "Revenue", IsActive: true},

		// Expenses
		{AccountCode: "5000", AccountName: "Cost of Goods Sold", AccountType: "Expense", IsActive: true},
		{AccountCode: "6000", AccountName: "Salaries & Wages", AccountType: "Expense", IsActive: true},
		{AccountCode: "6100", AccountName: "Rent Expense", AccountType: "Expense", IsActive: true},
		{AccountCode: "6200", AccountName: "Utilities", AccountType: "Expense", IsActive: true},
		{AccountCode: "6300", AccountName: "Office Supplies", AccountType: "Expense", IsActive: true},
		{AccountCode: "6400", AccountName: "Marketing & Advertising", AccountType: "Expense", IsActive: true},
		{AccountCode: "6500", AccountName: "Travel & Entertainment", AccountType: "Expense", IsActive: true},
		{AccountCode: "6600", AccountName: "Professional Fees", AccountType: "Expense", IsActive: true},
		{AccountCode: "6700", AccountName: "Depreciation Expense", AccountType: "Expense", IsActive: true},
		{AccountCode: "6800", AccountName: "Bank Charges", AccountType: "Expense", IsActive: true},
		{AccountCode: "6900", AccountName: "Miscellaneous Expense", AccountType: "Expense", IsActive: true},
	}

	for i := range defaultAccounts {
		defaultAccounts[i].CreatedBy = a.getCurrentUserID()
		defaultAccounts[i].Balance = 0
	}

	if err := a.db.Create(&defaultAccounts).Error; err != nil {
		return newError("DB_CREATE_FAILED", "Failed to seed chart of accounts", err.Error())
	}

	log.Printf("✅ Seeded %d default accounts for Bahrain trading company", len(defaultAccounts))
	return nil
}

// ============================================================================
// INVENTORY MANAGEMENT API (Phase 3B)
// ============================================================================

// GetInventoryItems retrieves inventory items with optional filters
func (a *App) GetInventoryItems(warehouseID *string, stockStatus string, lowStockOnly bool) ([]InventoryItem, error) {
	if err := a.requirePermission("inventory:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var items []InventoryItem
	query := a.db.Where("deleted_at IS NULL AND is_active = ?", true).Order("product_code ASC")

	if warehouseID != nil && *warehouseID != "" {
		query = query.Where("warehouse_id = ?", *warehouseID)
	}

	if stockStatus != "" && stockStatus != "All" {
		query = query.Where("stock_status = ?", stockStatus)
	}

	if lowStockOnly {
		query = query.Where("quantity_on_hand <= reorder_point")
	}

	if err := query.Find(&items).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve inventory items", err.Error())
	}

	log.Printf("✅ Retrieved %d inventory items", len(items))
	return items, nil
}

// GetInventoryItem retrieves a single inventory item by ID
func (a *App) GetInventoryItem(itemID string) (*InventoryItem, error) {
	if err := a.requirePermission("inventory:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Look up by the string UUID primary key. (No Preload("Movements") — the
	// InventoryItem model has no such association; movements are fetched via
	// GetStockMovements by inventory_item_id.)
	var item InventoryItem
	if err := a.db.First(&item, "id = ?", itemID).Error; err != nil {
		return nil, newError("ITEM_NOT_FOUND", "Inventory item not found", err.Error())
	}

	return &item, nil
}

// CreateInventoryItem creates a new inventory item
func (a *App) CreateInventoryItem(item InventoryItem) (*InventoryItem, error) {
	if err := a.requirePermission("inventory:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate
	if item.ProductID != "" {
		var product ProductMaster
		if err := a.db.First(&product, "id = ?", item.ProductID).Error; err == nil {
			// result["product_name"] = product.ProductName
			// result["product_code"] = product.ProductCode
		}
	}

	// Set defaults
	item.QuantityOnHand = 0
	item.QuantityReserved = 0
	item.QuantityAvailable = 0
	item.StockStatus = "OutOfStock"
	item.IsActive = true
	item.CreatedBy = a.getCurrentUserID()

	if err := a.db.Create(&item).Error; err != nil {
		return nil, newError("DB_CREATE_FAILED", "Failed to create inventory item", err.Error())
	}

	log.Printf("✅ Created inventory item for product #%s", item.ProductID)
	return &item, nil
}

// UpdateInventoryItem updates an inventory item
func (a *App) UpdateInventoryItem(itemID string, updates map[string]any) error {
	if err := a.requirePermission("inventory:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Mission I (I-12): quantities are ledger-owned (RecordStockMovement) —
	// an unfiltered map let a client rewrite quantity_on_hand and bypass the
	// stock-movement audit trail entirely.
	allowedColumns := map[string]bool{
		"reorder_point": true, "minimum_stock": true, "maximum_stock": true,
		"unit_cost": true, "stock_status": true, "is_active": true,
		"warehouse_id": true,
	}
	filtered := make(map[string]any, len(updates))
	for key, value := range updates {
		if allowedColumns[key] {
			filtered[key] = value
		} else {
			log.Printf("⚠️ UpdateInventoryItem: dropped non-editable column %q", key)
		}
	}
	if len(filtered) == 0 {
		return newError("INVALID_INPUT", "No editable fields in update payload", "")
	}

	if err := a.db.Model(&InventoryItem{}).Where("id = ?", itemID).Updates(filtered).Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to update inventory item", err.Error())
	}

	log.Printf("✅ Updated inventory item %s", itemID)
	return nil
}

// RecordStockMovement records a stock movement and updates inventory
func (a *App) RecordStockMovement(movement StockMovement) (*StockMovement, error) {
	if err := a.requirePermission("inventory:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate
	if movement.InventoryItemID == "" || movement.Quantity <= 0 {
		return nil, newError("INVALID_INPUT", "Inventory item ID and positive quantity required", "")
	}

	if movement.Direction != "IN" && movement.Direction != "OUT" {
		return nil, newError("INVALID_INPUT", "Direction must be IN or OUT", "")
	}

	// Start transaction
	tx := a.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get current inventory item (explicit condition — the PK is a string).
	var item InventoryItem
	if err := tx.First(&item, "id = ?", movement.InventoryItemID).Error; err != nil {
		tx.Rollback()
		return nil, newError("ITEM_NOT_FOUND", "Inventory item not found", err.Error())
	}

	// Record balance before
	movement.BalanceBefore = item.QuantityOnHand

	// Update quantity based on direction
	if movement.Direction == "IN" {
		item.QuantityOnHand += movement.Quantity
	} else {
		if item.QuantityOnHand < movement.Quantity {
			tx.Rollback()
			return nil, newError("INSUFFICIENT_STOCK",
				fmt.Sprintf("Insufficient stock: have %.2f, need %.2f", item.QuantityOnHand, movement.Quantity), "")
		}
		item.QuantityOnHand -= movement.Quantity
	}

	// Update available quantity
	item.QuantityAvailable = item.QuantityOnHand - item.QuantityReserved

	// Record balance after
	movement.BalanceAfter = item.QuantityOnHand

	// Update stock status
	item.StockStatus = a.calculateStockStatus(item.QuantityOnHand, item.ReorderPoint, item.MinimumStock, item.MaximumStock)

	// Band-2 rows 17-18: fallback-aware weighted-average valuation. The old
	// inline version averaged against the raw stored unit cost, so a
	// zero-cost item was valued at zero silently; the engine resolves through
	// unit cost → last purchase cost → product standard cost first.
	inventory.ApplyMovementValuation(&item, &movement, inventory.ResolveInventoryItemUnitCost(tx, item))

	// Generate movement number if not provided. Date-range scan, not
	// YEAR(...) — that is MySQL syntax and silently matched nothing on SQLite.
	if movement.MovementNumber == "" {
		year := time.Now().Year()
		yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
		yearEnd := yearStart.AddDate(1, 0, 0)
		var count int64
		tx.Model(&StockMovement{}).Where("movement_date >= ? AND movement_date < ?", yearStart, yearEnd).Count(&count)
		movement.MovementNumber = fmt.Sprintf("MOV-%d-%05d", year, count+1)
	}

	// Set defaults
	movement.CreatedBy = a.getCurrentUserID()
	now := time.Now()
	item.LastMovementAt = &now

	// Save movement
	if err := tx.Create(&movement).Error; err != nil {
		tx.Rollback()
		return nil, newError("DB_CREATE_FAILED", "Failed to create stock movement", err.Error())
	}

	// Update inventory item
	if err := tx.Save(&item).Error; err != nil {
		tx.Rollback()
		return nil, newError("DB_UPDATE_FAILED", "Failed to update inventory item", err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		return nil, newError("DB_COMMIT_FAILED", "Failed to commit stock movement", err.Error())
	}

	log.Printf("✅ Recorded stock movement: %s (%s %.2f units)",
		movement.MovementNumber, movement.Direction, movement.Quantity)
	return &movement, nil
}

// calculateStockStatus determines stock status based on quantities
func (a *App) calculateStockStatus(onHand, reorderPoint, minStock, maxStock float64) string {
	if onHand <= 0 {
		return "OutOfStock"
	}
	if onHand <= reorderPoint {
		return "LowStock"
	}
	if maxStock > 0 && onHand >= maxStock {
		return "Overstock"
	}
	return "InStock"
}

// GetStockMovements retrieves stock movements with filters
func (a *App) GetStockMovements(itemID *string, movementType string, startDate, endDate time.Time, limit int) ([]StockMovement, error) {
	if err := a.requirePermission("inventory:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	if limit <= 0 {
		limit = 100
	}

	var movements []StockMovement
	query := a.db.Where("deleted_at IS NULL").Order("movement_date DESC, created_at DESC").Limit(limit)

	if itemID != nil && *itemID != "" {
		query = query.Where("inventory_item_id = ?", *itemID)
	}

	if movementType != "" && movementType != "All" {
		query = query.Where("movement_type = ?", movementType)
	}

	if !startDate.IsZero() {
		query = query.Where("movement_date >= ?", startDate)
	}

	if !endDate.IsZero() {
		query = query.Where("movement_date <= ?", endDate)
	}

	if err := query.Find(&movements).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve stock movements", err.Error())
	}

	log.Printf("✅ Retrieved %d stock movements", len(movements))
	return movements, nil
}

// CreateStockAdjustment creates a stock adjustment (requires approval)
func (a *App) CreateStockAdjustment(adjustment StockAdjustment) error {
	if err := a.requirePermission("inventory:create"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	if adjustment.InventoryItemID == "" || adjustment.Reason == "" {
		return newError("INVALID_INPUT", "Item ID and reason required", "")
	}

	if strings.TrimSpace(adjustment.Status) == "" {
		adjustment.Status = "Pending"
	}
	if adjustment.AdjustmentDate.IsZero() {
		adjustment.AdjustmentDate = time.Now()
	}

	// Calculate variance if not provided
	if adjustment.Variance == 0 && (adjustment.SystemQuantity != 0 || adjustment.PhysicalQuantity != 0) {
		adjustment.Variance = adjustment.PhysicalQuantity - adjustment.SystemQuantity
	}

	// Generate number
	if adjustment.AdjustmentNumber == "" {
		adjustment.AdjustmentNumber = fmt.Sprintf("ADJ-%d", time.Now().Unix())
	}

	// Save
	if err := a.db.Create(&adjustment).Error; err != nil {
		return newError("DB_CREATE_FAILED", "Failed to create adjustment", err.Error())
	}

	// Article III: posting happens at the authorization moment, not at
	// creation. The adjustment persists as Status="Pending" — no
	// StockMovement is posted here. ApproveStockAdjustment is the sole
	// owner of the movement post (previously both functions posted,
	// double-applying the variance on a create→approve sequence).
	return nil
}

// ApproveStockAdjustment approves an adjustment and creates stock movement
func (a *App) ApproveStockAdjustment(adjustmentID string) error {
	if err := a.requirePermission("inventory:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// NOTE (Wave 8 P1): do NOT open an outer transaction here. RecordStockMovement
	// runs its own a.db.Begin()/Commit(); wrapping it in a second transaction
	// deadlocks the embedded SQLite with "database is locked". This latent nesting
	// was previously unreachable — the old uint itemID never matched the string PK,
	// so ApproveStockAdjustment always returned NOT_FOUND before hitting it. Sequenced
	// on a.db, mirroring the sibling CreateStockAdjustment (create → RecordStockMovement).
	var adjustment StockAdjustment
	if err := a.db.First(&adjustment, "id = ?", adjustmentID).Error; err != nil {
		return newError("ADJUSTMENT_NOT_FOUND", "Stock adjustment not found", err.Error())
	}

	if adjustment.Status != "Pending" {
		return newError("INVALID_STATUS", "Adjustment is not pending", "")
	}

	// Create stock movement for the adjustment
	movement := StockMovement{
		InventoryItemID: adjustment.InventoryItemID,
		MovementType:    "Adjustment",
		MovementDate:    adjustment.AdjustmentDate,
		Quantity:        math.Abs(adjustment.Variance),
		Direction:       "IN",
		UnitCost:        adjustment.UnitCost,
		ReferenceType:   "StockAdjustment",
		ReferenceID:     adjustment.ID,
	}

	if adjustment.Variance < 0 {
		movement.Direction = "OUT"
	}

	// Record the movement (this atomically updates inventory in its own tx)
	if _, err := a.RecordStockMovement(movement); err != nil {
		return err
	}

	// Mark adjustment as approved
	now := time.Now()
	adjustment.Status = "Approved"
	adjustment.ApprovedAt = &now
	adjustment.ApprovedBy = a.getCurrentUserID()

	if err := a.db.Save(&adjustment).Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to approve adjustment", err.Error())
	}

	log.Printf("✅ Approved stock adjustment: %s", adjustment.AdjustmentNumber)
	return nil
}

// GetLowStockItems retrieves items below reorder point
func (a *App) GetLowStockItems() ([]InventoryItem, error) {
	if err := a.requirePermission("inventory:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var items []InventoryItem
	if err := a.db.Where("deleted_at IS NULL AND is_active = ? AND quantity_on_hand <= reorder_point", true).
		Order("quantity_on_hand ASC").
		Find(&items).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve low stock items", err.Error())
	}

	log.Printf("⚠️  Found %d low stock items", len(items))
	return items, nil
}

// GetInventoryValuation calculates total inventory value. Band-2 rows 17-18:
// per-item fallback resolution (stored total → quantity × reference cost), so
// zero-cost rows no longer report as worthless; the warehouse filter takes the
// string warehouse ID the model actually stores (the old *uint matched
// nothing). Only the reported total is rounded (BHD, 3 decimals).
func (a *App) GetInventoryValuation(warehouseID *string) (map[string]any, error) {
	if err := a.requirePermission("inventory:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	query := a.db.Where("deleted_at IS NULL AND is_active = ?", true)

	if warehouseID != nil && strings.TrimSpace(*warehouseID) != "" {
		query = query.Where("warehouse_id = ?", strings.TrimSpace(*warehouseID))
	}

	var items []InventoryItem
	if err := query.Find(&items).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to calculate inventory valuation", err.Error())
	}

	totalValue := 0.0
	totalQty := 0.0
	for _, item := range items {
		totalValue += inventory.ResolveInventoryItemTotalValue(a.db, item)
		totalQty += item.QuantityOnHand
	}

	valuation := map[string]any{
		"total_value":    money.RoundFloat64(totalValue, 3),
		"total_items":    int64(len(items)),
		"total_quantity": totalQty,
		"warehouse_id":   warehouseID,
	}

	log.Printf("✅ Inventory valuation: %.2f BHD (%d items)", totalValue, len(items))
	return valuation, nil
}

// GetWarehouses retrieves all warehouses
func (a *App) GetWarehouses() ([]Warehouse, error) {
	if err := a.requirePermission("inventory:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var warehouses []Warehouse
	if err := a.db.Where("deleted_at IS NULL").Order("code ASC").Find(&warehouses).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve warehouses", err.Error())
	}

	log.Printf("✅ Retrieved %d warehouses", len(warehouses))
	return warehouses, nil
}

// CreateWarehouse creates a new warehouse
func (a *App) CreateWarehouse(warehouse Warehouse) (*Warehouse, error) {
	if err := a.requirePermission("inventory:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate
	if warehouse.Code == "" || warehouse.Name == "" {
		return nil, newError("INVALID_INPUT", "Warehouse code and name are required", "")
	}

	// Check for duplicate code
	var existing Warehouse
	if err := a.db.Where("code = ? AND deleted_at IS NULL", warehouse.Code).First(&existing).Error; err == nil {
		return nil, newError("DUPLICATE_CODE", "Warehouse code already exists", warehouse.Code)
	}

	// Set defaults
	warehouse.IsActive = true
	warehouse.CreatedBy = a.getCurrentUserID()

	if err := a.db.Create(&warehouse).Error; err != nil {
		return nil, newError("DB_CREATE_FAILED", "Failed to create warehouse", err.Error())
	}

	log.Printf("✅ Created warehouse: %s - %s", warehouse.Code, warehouse.Name)
	return &warehouse, nil
}

// ============================================================================
// POST-SALE NOTES & WARRANTY TRACKING
// ============================================================================

// GetPostSaleNotes retrieves post-sale notes for a specific order
func (a *App) GetPostSaleNotes(orderID string) ([]PostSaleNote, error) {
	if err := a.requirePermission("notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var notes []PostSaleNote
	if err := a.db.Where("order_id = ? AND deleted_at IS NULL", orderID).
		Order("created_at DESC").
		Find(&notes).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve post-sale notes", err.Error())
	}

	log.Printf("✅ Retrieved %d post-sale notes for order %s", len(notes), orderID)
	return notes, nil
}

// CreatePostSaleNote creates a new post-sale note (repair, warranty, etc.)
func (a *App) CreatePostSaleNote(note PostSaleNote) (*PostSaleNote, error) {
	if err := a.requirePermission("notes:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate required fields
	if note.OrderID == "" {
		return nil, newError("INVALID_INPUT", "Order ID is required", "")
	}
	if note.NoteType == "" {
		return nil, newError("INVALID_INPUT", "Note type is required", "")
	}
	if note.Description == "" {
		return nil, newError("INVALID_INPUT", "Description is required", "")
	}

	// Verify order exists
	var order Order
	if err := a.db.Where("id = ?", note.OrderID).First(&order).Error; err != nil {
		return nil, newError("ORDER_NOT_FOUND", "Order does not exist", note.OrderID)
	}

	// Set order number and defaults
	note.OrderNumber = order.OrderNumber
	note.CreatedBy = a.getCurrentUserID()

	if err := a.db.Create(&note).Error; err != nil {
		return nil, newError("DB_CREATE_FAILED", "Failed to create post-sale note", err.Error())
	}

	log.Printf("✅ Created post-sale note: %s for order %s (Cost: %.2f BHD)",
		note.NoteType, note.OrderNumber, note.CostBHD)
	return &note, nil
}

// UpdatePostSaleNote updates an existing post-sale note
func (a *App) UpdatePostSaleNote(note PostSaleNote) (*PostSaleNote, error) {
	if err := a.requirePermission("notes:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Check if note exists
	var existing PostSaleNote
	if err := a.db.Where("id = ? AND deleted_at IS NULL", note.ID).First(&existing).Error; err != nil {
		return nil, newError("NOTE_NOT_FOUND", "Post-sale note not found", "")
	}

	// Update allowed fields
	updates := map[string]any{
		"note_type":   note.NoteType,
		"description": note.Description,
		"cost_bhd":    note.CostBHD,
		"note_date":   note.NoteDate,
		"resolved_at": note.ResolvedAt,
		"resolution":  note.Resolution,
	}

	if err := a.db.Model(&existing).Updates(updates).Error; err != nil {
		return nil, newError("DB_UPDATE_FAILED", "Failed to update post-sale note", err.Error())
	}

	// Reload updated note
	if err := a.db.Where("id = ?", note.ID).First(&existing).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to reload note", err.Error())
	}

	log.Printf("✅ Updated post-sale note %s", note.ID)
	return &existing, nil
}

// DeletePostSaleNote soft-deletes a post-sale note
func (a *App) DeletePostSaleNote(noteID uint) error {
	if err := a.requirePermission("notes:delete"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var note PostSaleNote
	if err := a.db.Where("id = ? AND deleted_at IS NULL", noteID).First(&note).Error; err != nil {
		return newError("NOTE_NOT_FOUND", "Post-sale note not found", "")
	}

	// Soft delete
	now := time.Now()
	if err := a.db.Model(&note).Update("deleted_at", now).Error; err != nil {
		return newError("DB_DELETE_FAILED", "Failed to delete note", err.Error())
	}

	log.Printf("✅ Deleted post-sale note %d", noteID)
	return nil
}

// ============================================================================
// MCKINSEY FINANCIAL DASHBOARD API
// ============================================================================

// FinancialDashboard represents McKinsey-standard financial metrics
// Based on Acme Instrumentation WLL Audited Financials FS2024 (Demo Auditors)
type FinancialDashboard struct {
	// Period Info
	Period    string `json:"period"`
	PriorYear string `json:"prior_year"`
	AsOfDate  string `json:"as_of_date"`
	Source    string `json:"source"`

	// P&L Summary
	Revenue      float64 `json:"revenue"`
	RevenueYoY   float64 `json:"revenue_yoy"` // YoY change %
	COGS         float64 `json:"cogs"`
	GrossProfit  float64 `json:"gross_profit"`
	GrossMargin  float64 `json:"gross_margin"` // %
	OpEx         float64 `json:"opex"`
	EBITDA       float64 `json:"ebitda"`
	EBITDAMargin float64 `json:"ebitda_margin"` // %
	NetProfit    float64 `json:"net_profit"`
	NetMargin    float64 `json:"net_margin"` // %

	// Balance Sheet
	TotalAssets      float64 `json:"total_assets"`
	CurrentAssets    float64 `json:"current_assets"`
	NonCurrentAssets float64 `json:"non_current_assets"`
	TotalLiabilities float64 `json:"total_liabilities"`
	CurrentLiab      float64 `json:"current_liabilities"`
	TotalEquity      float64 `json:"total_equity"`

	// Cash Position
	CashAndEquiv   float64 `json:"cash_and_equiv"`
	FixedDeposits  float64 `json:"fixed_deposits"`
	TotalLiquidity float64 `json:"total_liquidity"`

	// Working Capital
	TradeReceivables float64 `json:"trade_receivables"`
	Inventory        float64 `json:"inventory"`
	TradePayables    float64 `json:"trade_payables"`
	WorkingCapital   float64 `json:"working_capital"`

	// Key Ratios - Liquidity
	CurrentRatio float64 `json:"current_ratio"`
	QuickRatio   float64 `json:"quick_ratio"`
	CashRatio    float64 `json:"cash_ratio"`

	// Key Ratios - Solvency
	DebtToEquity float64 `json:"debt_to_equity"`
	EquityRatio  float64 `json:"equity_ratio"` // %

	// Key Ratios - Efficiency
	DSO             float64 `json:"dso"`             // Days Sales Outstanding
	DIO             float64 `json:"dio"`             // Days Inventory Outstanding
	DPO             float64 `json:"dpo"`             // Days Payables Outstanding
	CashConvCycle   float64 `json:"cash_conv_cycle"` // DSO + DIO - DPO
	AssetTurnover   float64 `json:"asset_turnover"`
	ReceivablesTurn float64 `json:"receivables_turn"`

	// Key Ratios - Profitability
	ROA float64 `json:"roa"` // Return on Assets %
	ROE float64 `json:"roe"` // Return on Equity %

	// AR Aging Summary
	ARCurrent    float64 `json:"ar_current"` // 0-30 days
	AR30_60      float64 `json:"ar_30_60"`
	AR60_90      float64 `json:"ar_60_90"`
	AROver90     float64 `json:"ar_over_90"`
	AROverdue    float64 `json:"ar_overdue"`     // Total overdue
	AROverduePct float64 `json:"ar_overdue_pct"` // % of AR overdue

	// YoY Comparisons (Prior Year values)
	PY_Revenue     float64 `json:"py_revenue"`
	PY_GrossProfit float64 `json:"py_gross_profit"`
	PY_NetProfit   float64 `json:"py_net_profit"`
	PY_TotalAssets float64 `json:"py_total_assets"`
}

// GetFinancialDashboard returns McKinsey-standard financial metrics
// Real data from Acme Instrumentation WLL Audited Financials FS2024
