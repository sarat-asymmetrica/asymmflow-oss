package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ═══════════════════════════════════════════════════════════════════════════
// DATABASE SYNC SERVICE - Hybrid Sync with Progress UI
//
// Features:
//   - Real-time progress updates via Wails events
//   - Merge-only sync (NEVER deletes local data)
//   - Admin-triggered manual sync
//   - Configurable background sync frequency
//   - First-run sync with full progress bar
//
// Events emitted:
//   - sync:progress - Real-time progress updates
//   - sync:complete - Sync completed successfully
//   - sync:error    - Sync failed with error
//
// Built with safety-first approach - local data is NEVER deleted during sync
// ═══════════════════════════════════════════════════════════════════════════

// DBSyncProgress represents the current sync progress
type DBSyncProgress struct {
	Phase           string  `json:"phase"`            // "checking", "uploading", "downloading", "complete", "error"
	CurrentTable    string  `json:"current_table"`    // Current table being synced
	TablesCompleted int     `json:"tables_completed"` // Number of tables completed
	TablesTotal     int     `json:"tables_total"`     // Total number of tables
	RecordsSynced   int     `json:"records_synced"`   // Records synced so far
	RecordsTotal    int     `json:"records_total"`    // Total records to sync (estimated)
	Percentage      float64 `json:"percentage"`       // Overall percentage (0-100)
	Message         string  `json:"message"`          // Human-readable status
	Error           string  `json:"error,omitempty"`  // Error message if any
}

// DBSyncSettings holds user-configurable sync preferences
type DBSyncSettings struct {
	AutoSyncEnabled  bool   `json:"auto_sync_enabled"`
	SyncFrequencyMin int    `json:"sync_frequency_min"` // Minutes between syncs
	LastSyncAt       string `json:"last_sync_at,omitempty"`
	LastSyncStatus   string `json:"last_sync_status,omitempty"` // "success", "failed", "in_progress"
	RecordsSynced    int    `json:"records_synced,omitempty"`
}

// DBSyncResult represents the result of a sync operation
type DBSyncResult struct {
	Success         bool   `json:"success"`
	RecordsPushed   int    `json:"records_pushed"`
	RecordsPulled   int    `json:"records_pulled"`
	TablesProcessed int    `json:"tables_processed"`
	Duration        string `json:"duration"`
	Error           string `json:"error,omitempty"`
}

// Tables to sync (in dependency order)
var dbSyncTables = []string{
	"roles",
	"customers",
	"suppliers",
	"products",
	"customer_contacts",
	"supplier_contacts",
	"offers",
	"offer_items",
	"orders",
	"order_items",
	"invoices",
	"invoice_items",
	"credit_notes",
	"credit_note_items",
	"payments",
	"supplier_invoices",
	"supplier_invoice_items",
	"supplier_payments",
	"expense_categories",
	"expense_vendors",
	"expense_entries",
	"expense_allocations",
	"recurring_expenses",
	"expense_attachments",
	"expense_approvals",
	"purchase_orders",
	"purchase_order_items",
	"goods_received_notes",
	"grn_items",
	"serial_numbers",
	"delivery_notes",
	"delivery_note_items",
	"entity_notes",
	"invoice_sequences",
	"license_keys",
	"user_activity_sessions",
	"user_activity_events",
	"user_activity_weekly_summaries",
	"employees",
	"employee_access_links",
	"employee_compensation_profiles",
	"payroll_periods",
	"payroll_runs",
	"payroll_run_items",
	"payroll_components",
	"payroll_payouts",
	"projects",
	"project_members",
	"task_items",
	"task_comments",
	"task_activity",
	"notifications",
	"notification_receipts",
	"delete_approval_requests",
	"bank_accounts",
	"bank_statements",
	"bank_statement_lines",
	"bank_line_payment_allocations",
	"bank_expense_entries",
	"costing_sheet_data",
	"costing_line_items",
	"costing_history",
	"opportunities",
	"opportunity_edit_conflicts",
	"rfq_data",
	"offer_data",
}

func activeDBSyncTables() []string {
	excluded := map[string]bool{}
	for _, raw := range strings.Split(os.Getenv("ASYMMFLOW_SYNC_EXCLUDE_TABLES"), ",") {
		table := strings.TrimSpace(raw)
		if table != "" {
			excluded[table] = true
		}
	}
	if len(excluded) == 0 {
		return append([]string(nil), dbSyncTables...)
	}

	tables := make([]string, 0, len(dbSyncTables))
	for _, table := range dbSyncTables {
		if excluded[table] {
			continue
		}
		tables = append(tables, table)
	}
	return tables
}

// emitDBSyncProgress sends a sync progress event to the frontend
func (a *App) emitDBSyncProgress(progress DBSyncProgress) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "sync:progress", progress)
	}
}

// GetDBSyncSettings returns the current sync settings
func (a *App) GetDBSyncSettings() DBSyncSettings {
	if err := a.requirePermission("settings:update"); err != nil {
		return DBSyncSettings{}
	}
	settings := DBSyncSettings{
		AutoSyncEnabled:  true,
		SyncFrequencyMin: 30,
		LastSyncStatus:   "unknown",
	}

	// Try to load from database
	var setting Setting
	if err := a.db.Where("key = ?", "db_sync_auto_enabled").First(&setting).Error; err == nil {
		settings.AutoSyncEnabled = setting.Value == "true"
	}
	if err := a.db.Where("key = ?", "db_sync_frequency_min").First(&setting).Error; err == nil {
		fmt.Sscanf(setting.Value, "%d", &settings.SyncFrequencyMin)
	}
	if err := a.db.Where("key = ?", "db_sync_last_at").First(&setting).Error; err == nil {
		settings.LastSyncAt = setting.Value
	}
	if err := a.db.Where("key = ?", "db_sync_last_status").First(&setting).Error; err == nil {
		settings.LastSyncStatus = setting.Value
	}

	return settings
}

// UpdateDBSyncSettings updates the sync settings
func (a *App) UpdateDBSyncSettings(autoEnabled bool, frequencyMin int) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	// Validate frequency
	if frequencyMin < 5 {
		frequencyMin = 5 // Minimum 5 minutes
	}
	if frequencyMin > 1440 {
		frequencyMin = 1440 // Maximum 24 hours
	}

	// Save to database using upsert pattern
	a.saveSetting("db_sync_auto_enabled", fmt.Sprintf("%t", autoEnabled))
	a.saveSetting("db_sync_frequency_min", fmt.Sprintf("%d", frequencyMin))

	log.Printf("DB Sync settings updated: autoEnabled=%t, frequencyMin=%d", autoEnabled, frequencyMin)
	return nil
}

// saveSetting helper to save a setting
func (a *App) saveSetting(key, value string) {
	// Upsert: update if exists, insert if not (avoids accumulating soft-deleted rows)
	var existing Setting
	if err := a.db.Unscoped().Where("key = ?", key).First(&existing).Error; err == nil {
		a.db.Unscoped().Model(&existing).Updates(map[string]any{
			"value":      value,
			"deleted_at": nil,
		})
	} else {
		a.db.Create(&Setting{Key: key, Value: value})
	}
}

// SyncNowWithProgress performs a manual sync with real-time progress updates
func (a *App) SyncNowWithProgress() DBSyncResult {
	if err := a.requirePermission("settings:update"); err != nil {
		return DBSyncResult{Success: false, Error: err.Error()}
	}
	startTime := time.Now()
	result := DBSyncResult{Success: false}

	// Check if dbManager is available
	if a.dbManager == nil {
		a.emitDBSyncProgress(DBSyncProgress{
			Phase:   "error",
			Message: "Database manager not initialized",
			Error:   "Database manager not initialized",
		})
		result.Error = "Database manager not initialized"
		return result
	}

	// Check if online
	if !a.dbManager.IsOnline() {
		a.emitDBSyncProgress(DBSyncProgress{
			Phase:      "checking",
			Message:    "Connecting to cloud...",
			Percentage: 5,
		})

		if err := a.dbManager.ConnectRemote(); err != nil {
			a.emitDBSyncProgress(DBSyncProgress{
				Phase:   "error",
				Message: "Cannot connect to cloud",
				Error:   err.Error(),
			})
			result.Error = "Cannot connect to cloud: " + err.Error()
			return result
		}
	}

	syncTables := activeDBSyncTables()
	totalTables := len(syncTables)
	if totalTables == 0 {
		result.Success = true
		result.Duration = time.Since(startTime).Round(time.Second).String()
		a.emitDBSyncProgress(DBSyncProgress{
			Phase:      "complete",
			Message:    "No sync tables are enabled",
			Percentage: 100,
		})
		return result
	}
	totalRecordsSynced := 0

	// PHASE 1: Push local changes to cloud
	a.emitDBSyncProgress(DBSyncProgress{
		Phase:       "uploading",
		Message:     "Uploading local changes...",
		TablesTotal: totalTables,
		Percentage:  5,
	})

	pushedCount := 0
	syncErrors := []string{}

	for i, table := range syncTables {
		a.emitDBSyncProgress(DBSyncProgress{
			Phase:           "uploading",
			CurrentTable:    table,
			TablesCompleted: i,
			TablesTotal:     totalTables,
			RecordsSynced:   pushedCount,
			Percentage:      float64(5 + (i*40)/totalTables),
			Message:         fmt.Sprintf("Uploading %s...", table),
		})

		tableSince := a.dbManager.getLastSyncTimeForTable(table, "push")
		count, err := a.syncTablePush(table, tableSince)
		if err != nil {
			log.Printf("Push error for %s: %v", table, err)
			syncErrors = append(syncErrors, fmt.Sprintf("push %s: %v", table, err))
			continue
		}
		pushedCount += count
		totalRecordsSynced += count
	}

	result.RecordsPushed = pushedCount

	// PHASE 2: Pull cloud changes to local (MERGE ONLY - no deletes!)
	a.emitDBSyncProgress(DBSyncProgress{
		Phase:         "downloading",
		Message:       "Downloading cloud changes...",
		TablesTotal:   totalTables,
		RecordsSynced: totalRecordsSynced,
		Percentage:    50,
	})

	pulledCount := 0

	for i, table := range syncTables {
		a.emitDBSyncProgress(DBSyncProgress{
			Phase:           "downloading",
			CurrentTable:    table,
			TablesCompleted: i,
			TablesTotal:     totalTables,
			RecordsSynced:   totalRecordsSynced + pulledCount,
			Percentage:      float64(50 + (i*45)/totalTables),
			Message:         fmt.Sprintf("Downloading %s...", table),
		})

		tableSince := a.dbManager.getLastSyncTimeForTable(table, "pull")
		count, err := a.syncTablePull(table, tableSince)
		if err != nil {
			log.Printf("Pull error for %s: %v", table, err)
			syncErrors = append(syncErrors, fmt.Sprintf("pull %s: %v", table, err))
			continue
		}
		pulledCount += count
		totalRecordsSynced += count
	}

	result.RecordsPulled = pulledCount

	if len(syncErrors) > 0 {
		duration := time.Since(startTime)
		result.Success = false
		result.TablesProcessed = totalTables
		result.Duration = duration.Round(time.Second).String()
		result.Error = fmt.Sprintf("%d sync table errors: %s", len(syncErrors), strings.Join(syncErrors, "; "))
		a.saveSetting("db_sync_last_status", "failed")
		a.emitDBSyncProgress(DBSyncProgress{
			Phase:           "error",
			TablesCompleted: totalTables,
			TablesTotal:     totalTables,
			RecordsSynced:   totalRecordsSynced,
			Percentage:      100,
			Message:         "Sync finished with table errors",
			Error:           result.Error,
		})
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "sync:complete", result)
		}
		log.Printf("DB Sync completed with errors: pushed=%d, pulled=%d, duration=%s, errors=%s", pushedCount, pulledCount, result.Duration, result.Error)
		return result
	}

	// Update last sync time
	a.saveSetting("db_sync_last_at", time.Now().Format(time.RFC3339))
	a.saveSetting("db_sync_last_status", "success")

	// Complete!
	duration := time.Since(startTime)
	result.Success = true
	result.TablesProcessed = totalTables
	result.Duration = duration.Round(time.Second).String()

	a.emitDBSyncProgress(DBSyncProgress{
		Phase:           "complete",
		TablesCompleted: totalTables,
		TablesTotal:     totalTables,
		RecordsSynced:   totalRecordsSynced,
		RecordsTotal:    totalRecordsSynced,
		Percentage:      100,
		Message:         fmt.Sprintf("Sync complete! %d records synced", totalRecordsSynced),
	})

	// Emit completion event
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "sync:complete", result)
	}

	log.Printf("DB Sync completed: pushed=%d, pulled=%d, duration=%s", pushedCount, pulledCount, duration)
	return result
}

// validateRecordForSync checks if a record is valid before pushing to cloud
// Returns true if record is valid, false if it should be skipped
func (a *App) validateRecordForSync(table string, record map[string]any) (bool, string) {
	// Skip records without ID
	if record["id"] == nil || record["id"] == "" {
		return false, "missing id"
	}

	// Table-specific validation
	switch table {
	case "customers":
		if record["business_name"] == nil || record["business_name"] == "" {
			return false, "customer missing business_name"
		}
	case "suppliers":
		if record["supplier_name"] == nil || record["supplier_name"] == "" {
			return false, "supplier missing supplier_name"
		}
	case "invoices":
		if record["invoice_number"] == nil || record["invoice_number"] == "" {
			return false, "invoice missing invoice_number"
		}
		// Validate amounts are not negative
		if amount, ok := record["grand_total_bhd"].(float64); ok && amount < 0 {
			return false, "invoice has negative grand_total"
		}
	case "orders":
		if record["order_number"] == nil || record["order_number"] == "" {
			return false, "order missing order_number"
		}
	case "payments":
		if amount, ok := record["amount"].(float64); ok && amount < 0 {
			return false, "payment has negative amount"
		}
	case "offers":
		if record["offer_number"] == nil || record["offer_number"] == "" {
			return false, "offer missing offer_number"
		}
	case "products":
		if record["name"] == nil || record["name"] == "" {
			return false, "product missing name"
		}
	}

	// Check for obviously corrupted timestamps
	if createdAt, ok := record["created_at"].(time.Time); ok {
		// Reject dates before 2020 or after 2030 (obviously wrong)
		if createdAt.Year() < 2020 || createdAt.Year() > 2030 {
			return false, "invalid created_at timestamp"
		}
	}

	return true, ""
}

// syncTablePush pushes local changes to cloud (safe upsert with validation)
func (a *App) syncTablePush(table string, since time.Time) (int, error) {
	if a.dbManager.Remote() == nil {
		return 0, nil
	}
	localColumnTypes, err := syncDBColumnTypes(a.db, table)
	if err != nil {
		return 0, fmt.Errorf("inspect local %s columns: %w", table, err)
	}
	if !syncColumnExists(localColumnTypes, "updated_at") {
		log.Printf("Skipping push sync for %s: local table has no updated_at column for incremental sync", table)
		return 0, nil
	}

	// Count records to sync
	var count int64
	if err := a.db.Table(table).Where("updated_at > ?", since).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count local %s records: %w", table, err)
	}
	if count == 0 {
		return 0, nil
	}

	// Get records
	rows, err := a.db.Table(table).Where("updated_at > ?", since).Rows()
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	remoteColumnTypes, err := syncDBColumnTypes(a.dbManager.Remote(), table)
	if err != nil {
		return 0, fmt.Errorf("inspect remote %s columns: %w", table, err)
	}

	synced := 0
	skipped := 0
	for rows.Next() {
		var record map[string]any
		if err := a.db.ScanRows(rows, &record); err != nil {
			continue
		}

		// VALIDATION: Check record integrity before pushing
		valid, reason := a.validateRecordForSync(table, record)
		if !valid {
			log.Printf("Sync validation failed for %s: %s (id=%v)", table, reason, record["id"])
			skipped++
			continue
		}

		remoteRecord := normalizeRecordForRemoteSyncSchema(record, remoteColumnTypes)

		// SAFE UPSERT: Update if exists, create if not
		id := remoteRecord["id"]
		if err := syncUpsertRecordByIDOrNaturalKey(a.dbManager.Remote(), table, remoteRecord, remoteColumnTypes); err != nil {
			return synced, fmt.Errorf("upsert remote %s/%v: %w", table, id, err)
		}
		synced++
	}

	if skipped > 0 {
		log.Printf("Sync %s: %d synced, %d skipped (validation failed)", table, synced, skipped)
	}

	if synced > 0 {
		a.db.Create(&SyncRecord{
			SyncTable:     table,
			RecordID:      fmt.Sprintf("batch_%d", synced),
			SyncedAt:      time.Now(),
			Direction:     "push",
			ConflictState: "none",
		})
	}

	return synced, nil
}

// syncTablePull pulls cloud changes to local (MERGE ONLY - never deletes!)
func (a *App) syncTablePull(table string, since time.Time) (int, error) {
	if a.dbManager.Remote() == nil {
		return 0, nil
	}
	if isUserActivitySyncTable(table) && !a.currentSessionCanAccessActivityMonitoring() {
		log.Printf("Skipping pull sync for %s: confidential activity monitoring is restricted to developer role", table)
		return 0, nil
	}
	remoteColumnTypes, err := syncDBColumnTypes(a.dbManager.Remote(), table)
	if err != nil {
		return 0, fmt.Errorf("inspect remote %s columns: %w", table, err)
	}
	if !syncColumnExists(remoteColumnTypes, "updated_at") {
		log.Printf("Skipping pull sync for %s: remote table has no updated_at column for incremental sync", table)
		return 0, nil
	}

	// Count records to sync
	var count int64
	if err := a.dbManager.Remote().Table(table).Where("updated_at > ?", since).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count remote %s records: %w", table, err)
	}
	if count == 0 {
		return 0, nil
	}
	localColumnTypes, err := syncDBColumnTypes(a.db, table)
	if err != nil {
		return 0, fmt.Errorf("inspect local %s columns: %w", table, err)
	}

	// Get records
	rows, err := a.dbManager.Remote().Table(table).Where("updated_at > ?", since).Rows()
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	synced := 0
	for rows.Next() {
		var record map[string]any
		if err := a.dbManager.Remote().ScanRows(rows, &record); err != nil {
			continue
		}

		// SAFE MERGE: Update if exists, create if not
		// NEVER DELETE local data!
		id := record["id"]
		if id == nil {
			continue
		}
		record = normalizeRecordForRemoteSyncSchema(record, localColumnTypes)

		result := a.db.Table(table).Where("id = ?", id).Updates(record)
		if result.Error != nil {
			return synced, fmt.Errorf("update local %s/%v: %w", table, id, result.Error)
		}
		if result.RowsAffected == 0 {
			// Record doesn't exist locally, create it
			if err := a.db.Table(table).Create(record).Error; err != nil {
				return synced, fmt.Errorf("create local %s/%v: %w", table, id, err)
			}
		}
		synced++
	}

	if synced > 0 {
		a.db.Create(&SyncRecord{
			SyncTable:     table,
			RecordID:      fmt.Sprintf("batch_%d", synced),
			SyncedAt:      time.Now(),
			Direction:     "pull",
			ConflictState: "none",
		})
	}

	return synced, nil
}

// FirstRunSyncWithProgress performs first-run sync with progress updates
func (a *App) FirstRunSyncWithProgress() DBSyncResult {
	if err := a.requirePermission("settings:update"); err != nil {
		return DBSyncResult{Success: false, Error: err.Error()}
	}
	startTime := time.Now()
	result := DBSyncResult{Success: false}

	if a.dbManager == nil {
		a.InitDBManager()
	}

	if a.dbManager == nil {
		result.Error = "Database manager not initialized"
		a.emitDBSyncProgress(DBSyncProgress{
			Phase: "error",
			Error: result.Error,
		})
		return result
	}

	// Check if sync is needed
	a.emitDBSyncProgress(DBSyncProgress{
		Phase:      "checking",
		Message:    "Checking if sync is needed...",
		Percentage: 5,
	})

	// Check local data
	var localCustomers, localOrders, localInvoices int64
	a.db.Table("customers").Count(&localCustomers)
	a.db.Table("orders").Count(&localOrders)
	a.db.Table("invoices").Count(&localInvoices)

	if localCustomers > 0 || localOrders > 0 || localInvoices > 0 {
		// Local has data, skip first-run sync
		a.emitDBSyncProgress(DBSyncProgress{
			Phase:      "complete",
			Message:    "Local database has data, no sync needed",
			Percentage: 100,
		})
		result.Success = true
		result.Duration = time.Since(startTime).Round(time.Second).String()
		return result
	}

	// Connect to cloud
	a.emitDBSyncProgress(DBSyncProgress{
		Phase:      "checking",
		Message:    "Connecting to cloud...",
		Percentage: 10,
	})

	if !a.dbManager.IsOnline() {
		if err := a.dbManager.ConnectRemote(); err != nil {
			result.Error = "Cannot connect to cloud"
			a.emitDBSyncProgress(DBSyncProgress{
				Phase: "error",
				Error: result.Error,
			})
			return result
		}
	}

	// Check cloud data
	var cloudCustomers int64
	if a.dbManager.Remote() != nil {
		a.dbManager.Remote().Table("customers").Count(&cloudCustomers)
	}

	if cloudCustomers == 0 {
		a.emitDBSyncProgress(DBSyncProgress{
			Phase:      "complete",
			Message:    "Cloud database is empty, nothing to download",
			Percentage: 100,
		})
		result.Success = true
		result.Duration = time.Since(startTime).Round(time.Second).String()
		return result
	}

	// Download all tables with progress
	syncTables := activeDBSyncTables()
	totalTables := len(syncTables)
	if totalTables == 0 {
		result.Success = true
		result.Duration = time.Since(startTime).Round(time.Second).String()
		a.emitDBSyncProgress(DBSyncProgress{
			Phase:      "complete",
			Message:    "No sync tables are enabled",
			Percentage: 100,
		})
		return result
	}
	totalRecords := 0

	a.emitDBSyncProgress(DBSyncProgress{
		Phase:       "downloading",
		Message:     fmt.Sprintf("Downloading data from cloud (%d customers found)...", cloudCustomers),
		TablesTotal: totalTables,
		Percentage:  15,
	})

	for i, table := range syncTables {
		// Get count from cloud
		var tableCount int64
		if a.dbManager.Remote() != nil {
			a.dbManager.Remote().Table(table).Count(&tableCount)
		}

		a.emitDBSyncProgress(DBSyncProgress{
			Phase:           "downloading",
			CurrentTable:    table,
			TablesCompleted: i,
			TablesTotal:     totalTables,
			RecordsSynced:   totalRecords,
			Percentage:      float64(15 + (i*80)/totalTables),
			Message:         fmt.Sprintf("Downloading %s (%d records)...", table, tableCount),
		})

		if tableCount == 0 {
			continue
		}

		// Download using SAFE MERGE (not destructive!)
		count, err := a.downloadTableSafe(table)
		if err != nil {
			log.Printf("Download error for %s: %v", table, err)
			continue
		}

		totalRecords += count

		a.emitDBSyncProgress(DBSyncProgress{
			Phase:           "downloading",
			CurrentTable:    table,
			TablesCompleted: i + 1,
			TablesTotal:     totalTables,
			RecordsSynced:   totalRecords,
			Percentage:      float64(15 + ((i+1)*80)/totalTables),
			Message:         fmt.Sprintf("Downloaded %s (%d records)", table, count),
		})
	}

	// Complete!
	duration := time.Since(startTime)
	result.Success = true
	result.RecordsPulled = totalRecords
	result.TablesProcessed = totalTables
	result.Duration = duration.Round(time.Second).String()

	a.emitDBSyncProgress(DBSyncProgress{
		Phase:           "complete",
		TablesCompleted: totalTables,
		TablesTotal:     totalTables,
		RecordsSynced:   totalRecords,
		RecordsTotal:    totalRecords,
		Percentage:      100,
		Message:         fmt.Sprintf("Download complete! %d records", totalRecords),
	})

	// Update sync status
	a.saveSetting("db_sync_last_at", time.Now().Format(time.RFC3339))
	a.saveSetting("db_sync_last_status", "success")

	// Emit completion event
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "sync:complete", result)
	}

	log.Printf("First-run sync completed: %d records in %s", totalRecords, duration)
	return result
}

// downloadTableSafe downloads a table using MERGE (insert or update, never delete)
func (a *App) downloadTableSafe(table string) (int, error) {
	if a.dbManager.Remote() == nil {
		return 0, fmt.Errorf("remote database not connected")
	}

	// Download in batches
	batchSize := 100
	offset := 0
	downloaded := 0

	for {
		var records []map[string]any
		result := a.dbManager.Remote().Table(table).Offset(offset).Limit(batchSize).Find(&records)
		if result.Error != nil {
			return downloaded, result.Error
		}

		if len(records) == 0 {
			break
		}

		// SAFE MERGE: Insert or update each record (NEVER DELETE!)
		for _, record := range records {
			id := record["id"]
			if id == nil {
				continue
			}

			// Try update first
			updateResult := a.db.Table(table).Where("id = ?", id).Updates(record)
			if updateResult.RowsAffected == 0 {
				// Doesn't exist, create it
				if err := a.db.Table(table).Create(record).Error; err != nil {
					// Log but continue - might be constraint issues
					log.Printf("Insert error for %s: %v", table, err)
					continue
				}
			}
			downloaded++
		}

		offset += len(records)
		if len(records) < batchSize {
			break
		}
	}

	return downloaded, nil
}

// StartBackgroundDBSync starts the background database sync with configurable frequency.
// Only callable once — subsequent calls are no-ops to prevent goroutine leaks.
func (a *App) StartBackgroundDBSync() {
	// Guard against double-start (bgSyncStop non-nil means already started)
	if a.bgSyncStop != nil {
		log.Println("Background DB sync already started, ignoring duplicate call")
		return
	}
	// Read settings without RBAC (this is a background startup call)
	var autoEnabled bool
	var freqMin int
	var setting Setting
	if err := a.db.Where("key = ?", "db_sync_auto_enabled").First(&setting).Error; err == nil {
		autoEnabled = setting.Value == "true"
	} else {
		autoEnabled = true // default
	}
	if err := a.db.Where("key = ?", "db_sync_frequency_min").First(&setting).Error; err == nil {
		fmt.Sscanf(setting.Value, "%d", &freqMin)
	}
	if freqMin < 5 {
		freqMin = 30 // default
	}

	if !autoEnabled {
		log.Println("Background DB sync disabled by user settings")
		return
	}

	frequency := time.Duration(freqMin) * time.Minute

	a.bgSyncStop = make(chan struct{})
	a.bgSyncWG.Add(1)
	go func() {
		defer a.bgSyncWG.Done()

		// Initial delay to let app fully start
		select {
		case <-time.After(60 * time.Second):
		case <-a.bgSyncStop:
			return
		}

		ticker := time.NewTicker(frequency)
		defer ticker.Stop()

		log.Printf("Background DB sync started (interval: %v)", frequency)

		for {
			select {
			case <-ticker.C:
				// Silent sync in background (no progress events to avoid UI noise)
				if a.dbManager != nil && a.dbManager.IsOnline() {
					lastSync := a.dbManager.getLastSyncTime()

					pushed, pushErr := a.dbManager.SyncToRemote(lastSync)
					pulled, pullErr := a.dbManager.SyncFromRemote(lastSync)
					if pushErr != nil || pullErr != nil {
						log.Printf("Background DB sync completed with errors: pushed=%d pulled=%d push=%v pull=%v", pushed, pulled, pushErr, pullErr)
						a.saveSetting("db_sync_last_status", "failed")
						continue
					}

					if pushed > 0 || pulled > 0 {
						log.Printf("Background DB sync: pushed=%d, pulled=%d", pushed, pulled)
						a.saveSetting("db_sync_last_at", time.Now().Format(time.RFC3339))
						a.saveSetting("db_sync_last_status", "success")
					}
				}
			case <-a.bgSyncStop:
				log.Println("Background DB sync stopped (shutdown)")
				return
			}
		}
	}()
}

// StopBackgroundDBSync stops the background DB sync goroutine and waits for it to finish.
func (a *App) StopBackgroundDBSync() {
	if a.bgSyncStop != nil {
		a.bgSyncStopOnce.Do(func() { close(a.bgSyncStop) })
		a.bgSyncWG.Wait()
	}
}
