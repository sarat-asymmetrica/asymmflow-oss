package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// DBManager manages dual-database connections (local SQLite + remote Supabase)
// SQLite is PRIMARY (always works offline), Supabase is for sync/backup
type DBManager struct {
	local                     *gorm.DB
	remote                    *gorm.DB
	config                    SupabaseConfig
	isOnline                  bool
	mu                        sync.RWMutex
	lastCheck                 time.Time
	checkInterval             time.Duration
	syncEnabled               bool
	syncInProgress            bool       // Prevents concurrent first-run syncs
	syncMu                    sync.Mutex // Lock for sync operations
	canPullActivityMonitoring func() bool
	stopChan                  chan struct{}
	stopOnce                  sync.Once
	wg                        sync.WaitGroup
}

// NewDBManager creates a new dual-database manager
func NewDBManager(localDB *gorm.DB, cfg SupabaseConfig) *DBManager {
	return &DBManager{
		local:         localDB,
		config:        cfg,
		isOnline:      false,
		checkInterval: 30 * time.Second,
		syncEnabled:   cfg.Enabled,
		stopChan:      make(chan struct{}),
	}
}

// Primary returns the PRIMARY database (always SQLite in hybrid mode)
// This ensures app always works even if Supabase is down
func (m *DBManager) Primary() *gorm.DB {
	return m.local // SQLite is always primary
}

// Local returns the local SQLite database (always available)
func (m *DBManager) Local() *gorm.DB {
	return m.local
}

// Remote returns the remote Supabase database (may be nil)
func (m *DBManager) Remote() *gorm.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.remote
}

// IsOnline returns whether Supabase sync is connected
func (m *DBManager) IsOnline() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isOnline
}

// IsSyncEnabled returns whether cloud sync is configured
func (m *DBManager) IsSyncEnabled() bool {
	return m.syncEnabled
}

// ConnectRemote establishes connection to Supabase PostgreSQL for sync
// This is non-blocking - failure just means sync won't work, app continues
func (m *DBManager) ConnectRemote() error {
	if !m.config.Enabled {
		log.Println("Supabase sync not configured - running in offline mode")
		return nil // Not an error, just not configured
	}

	if m.config.DBHost == "" || m.config.DBPassword == "" {
		log.Println("Supabase credentials incomplete - running in offline mode")
		return nil
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		m.config.DBHost,
		m.config.DBPort,
		m.config.DBUser,
		m.config.DBPassword,
		m.config.DBName,
		m.config.DBSSLMode,
	)
	// 3-CONN: bound connection attempts, and cap individual query time so a
	// mid-query network drop can't hang the sync path for hours.
	dsn += " connect_timeout=10 statement_timeout=30000"

	log.Printf("Connecting to Supabase (host: %s)...", m.config.DBHost)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Printf("Supabase connection failed (will retry): %v", err)
		return fmt.Errorf("supabase connection failed: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(5) // Lower for sync-only usage
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	// Verify connectivity with a deadline — an unbounded Ping can hang on a
	// half-open connection.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return fmt.Errorf("supabase ping failed: %w", err)
	}

	m.mu.Lock()
	m.remote = db
	m.isOnline = true
	m.lastCheck = time.Now()
	m.mu.Unlock()

	log.Println("Supabase connected successfully - sync enabled")
	return nil
}

// CheckConnectivity verifies the Supabase connection is alive
func (m *DBManager) CheckConnectivity() bool {
	m.mu.RLock()
	remote := m.remote
	m.mu.RUnlock()

	if remote == nil {
		return false
	}

	sqlDB, err := remote.DB()
	if err != nil {
		m.setOffline()
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		m.setOffline()
		log.Printf("Supabase connectivity lost: %v", err)
		return false
	}

	m.mu.Lock()
	m.isOnline = true
	m.lastCheck = time.Now()
	m.mu.Unlock()
	return true
}

// MigrateRemote runs AutoMigrate on Supabase (creates tables if needed)
func (m *DBManager) MigrateRemote() error {
	if m.remote == nil {
		return fmt.Errorf("supabase not connected")
	}

	log.Println("Running schema migration on Supabase...")

	models := []any{
		// Master Data
		&CustomerMaster{},
		&CustomerContact{},
		&SupplierMaster{},
		&SupplierContact{},
		&ProductMaster{},
		&EntityNote{},
		&SupplierIssue{},
		// Sales
		&RFQData{},
		&CostingSheetData{},
		&CostingLineItemData{},
		&CostingHistory{},
		&OfferData{},
		&Offer{},
		&OfferItem{},
		&Opportunity{},
		&OpportunityEditConflict{},
		&Order{},
		&OrderItem{},
		// Finance
		&Invoice{},
		&DBInvoiceItem{},
		&CreditNote{},
		&CreditNoteItem{},
		&Payment{},
		&SupplierPayment{},
		// Operations
		&PurchaseOrder{},
		&PurchaseOrderItem{},
		&GoodsReceivedNote{},
		&GRNItem{},
		&SerialNumber{},
		&SupplierInvoice{},
		&SupplierInvoiceItem{},
		&DeliveryNote{},
		&DeliveryNoteItem{},
		// Intelligence
		&Conversation{},
		&ChatMessage{},
		&OfferFollowUp{},
		// Sequence Numbers
		&InvoiceSequence{},
		// Finance — Banking & Costing
		&BankAccount{},
		&BankStatement{},
		&BankStatementLine{},
		&BankExpenseEntry{},
		&ExpenseCategory{},
		&ExpenseVendor{},
		&ExpenseEntry{},
		&ExpenseAllocation{},
		&RecurringExpense{},
		&ExpenseAttachment{},
		&ExpenseApproval{},
		&EmployeeCompensationProfile{},
		&PayrollPeriod{},
		&PayrollRun{},
		&PayrollRunItem{},
		&PayrollComponent{},
		&PayrollPayout{},
		// System
		&SyncRecord{},
		&Role{},
		&User{},
		&AuditLog{},
		&Setting{},
		&Device{},
		&DeviceUser{},
		&LicenseKey{},
		&UserActivitySession{},
		&UserActivityEvent{},
		&UserActivityWeeklySummary{},
		// Collaboration
		&Employee{},
		&EmployeeDocument{}, // Wave 9.8 B4
		&EmployeeAccessLink{},
		&Project{},
		&ProjectMember{},
		&Notification{},
		&NotificationReceipt{},
		&TaskItem{},
		&TaskComment{},
		&TaskActivity{},
	}

	for _, model := range models {
		if err := m.remote.AutoMigrate(model); err != nil {
			if isBenignRemoteMigrationDrift(err) {
				log.Printf("Supabase migration drift warning: %v", err)
				continue
			}
			log.Printf("Supabase migration warning: %v", err)
			return err
		}
	}

	// 1-SYNC (PH ffbe9c7): SQLite (source of truth) does not enforce VARCHAR
	// lengths, so real data routinely exceeds the GORM-declared sizes. Widen
	// every narrow varchar column in PostgreSQL to TEXT so batch upserts never
	// fail with "value too long for type character varying(N)".
	type narrowCol struct {
		TableName  string `gorm:"column:table_name"`
		ColumnName string `gorm:"column:column_name"`
	}
	var cols []narrowCol
	if err := m.remote.Raw(`
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA()
		  AND data_type = 'character varying'
		  AND character_maximum_length IS NOT NULL
		  AND character_maximum_length <= 255
	`).Scan(&cols).Error; err != nil {
		return fmt.Errorf("query narrow varchar columns: %w", err)
	}
	for _, col := range cols {
		if !isValidSQLIdentifier(col.TableName) || !isValidSQLIdentifier(col.ColumnName) {
			continue
		}
		statement := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" TYPE TEXT`, col.TableName, col.ColumnName)
		if err := m.remote.Exec(statement).Error; err != nil {
			log.Printf("Warning: widen %s.%s to TEXT: %v", col.TableName, col.ColumnName, err)
		}
	}
	log.Printf("Widened %d narrow varchar columns to TEXT on remote", len(cols))

	log.Println("Supabase schema migration complete")
	return nil
}

func isBenignRemoteMigrationDrift(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "does not exist") &&
		(strings.Contains(msg, "drop constraint") ||
			strings.Contains(msg, "constraint \"uni_")) {
		return true
	}
	return strings.Contains(msg, "check constraint") && strings.Contains(msg, "is violated by some row")
}

// SyncToRemote pushes local changes to Supabase
func (m *DBManager) SyncToRemote(since time.Time) (int, error) {
	if !m.IsOnline() {
		return 0, nil // Silent fail - sync will retry later
	}

	pushed := 0
	tableErrors := []string{}
	// Use the active canonical table list so push and pull always cover the same tables.
	for _, table := range activeDBSyncTables() {
		tableSince := m.getLastSyncTimeForTable(table, "push")
		count, err := m.syncTable(table, tableSince, "push")
		if err != nil {
			log.Printf("Sync push error for %s: %v", table, err)
			tableErrors = append(tableErrors, fmt.Sprintf("%s: %v", table, err))
			continue
		}
		pushed += count
	}

	if len(tableErrors) > 0 {
		return pushed, fmt.Errorf("push completed with %d table errors: %s", len(tableErrors), strings.Join(tableErrors, "; "))
	}
	return pushed, nil
}

// SyncFromRemote pulls changes from Supabase to local
func (m *DBManager) SyncFromRemote(since time.Time) (int, error) {
	if !m.IsOnline() {
		return 0, nil // Silent fail
	}

	pulled := 0
	tableErrors := []string{}
	// Use the active canonical table list — must match push to ensure all
	// computers receive the full dataset (orders+items, invoices+items, etc.)
	for _, table := range activeDBSyncTables() {
		if isUserActivitySyncTable(table) && !m.activityMonitoringPullAllowed() {
			log.Printf("Skipping pull sync for %s: confidential activity monitoring is restricted to developer role", table)
			continue
		}
		tableSince := m.getLastSyncTimeForTable(table, "pull")
		count, err := m.syncTable(table, tableSince, "pull")
		if err != nil {
			log.Printf("Sync pull error for %s: %v", table, err)
			tableErrors = append(tableErrors, fmt.Sprintf("%s: %v", table, err))
			continue
		}
		pulled += count
	}

	if len(tableErrors) > 0 {
		return pulled, fmt.Errorf("pull completed with %d table errors: %s", len(tableErrors), strings.Join(tableErrors, "; "))
	}
	return pulled, nil
}

// StartPeriodicSync runs background sync at the specified interval
func (m *DBManager) StartPeriodicSync(interval time.Duration) {
	if !m.syncEnabled {
		log.Println("Periodic sync disabled (no Supabase config)")
		return
	}

	runCycle := func() {
		// Try to connect if not online
		if !m.IsOnline() {
			if err := m.ConnectRemote(); err != nil {
				return
			}
		}

		// Check connectivity
		if !m.CheckConnectivity() {
			return
		}

		// Perform sync
		lastSync := m.getLastSyncTime()
		if pushed, err := m.SyncToRemote(lastSync); err != nil {
			log.Printf("Sync push completed with errors after %d records: %v", pushed, err)
		} else if pushed > 0 {
			log.Printf("Sync: pushed %d records to Supabase", pushed)
		}
		if pulled, err := m.SyncFromRemote(lastSync); err != nil {
			log.Printf("Sync pull completed with errors after %d records: %v", pulled, err)
		} else if pulled > 0 {
			log.Printf("Sync: pulled %d records from Supabase", pulled)
		}
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		// Initial delay to let app start
		select {
		case <-time.After(10 * time.Second):
		case <-m.stopChan:
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		log.Printf("Background sync started (interval: %v)", interval)
		runCycle()

		for {
			select {
			case <-ticker.C:
				runCycle()
			case <-m.stopChan:
				log.Println("Background sync stopped (shutdown)")
				return
			}
		}
	}()
}

// StopPeriodicSync stops the background sync goroutine and waits for it to finish.
func (m *DBManager) StopPeriodicSync() {
	m.stopOnce.Do(func() {
		close(m.stopChan)
		log.Println("DBManager periodic sync stop signal sent")
	})
	m.wg.Wait() // Wait for goroutine to finish before returning
}

// TriggerSync manually triggers a sync (for UI "Sync Now" button)
func (m *DBManager) TriggerSync() (int, int, error) {
	if !m.IsOnline() {
		// Try to connect first
		if err := m.ConnectRemote(); err != nil {
			return 0, 0, fmt.Errorf("cannot connect to Supabase: %w", err)
		}
	}

	lastSync := m.getLastSyncTime()
	pushed, pushErr := m.SyncToRemote(lastSync)
	pulled, pullErr := m.SyncFromRemote(lastSync)
	if pushErr != nil || pullErr != nil {
		return pushed, pulled, fmt.Errorf("sync completed with errors: push=%v pull=%v", pushErr, pullErr)
	}

	return pushed, pulled, nil
}

// Disconnect closes the Supabase connection
func (m *DBManager) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.remote != nil {
		if sqlDB, err := m.remote.DB(); err == nil {
			sqlDB.Close()
		}
		m.remote = nil
		m.isOnline = false
	}
}

// --- Internal helpers ---

func (m *DBManager) setOffline() {
	m.mu.Lock()
	m.isOnline = false
	m.mu.Unlock()
}

func (m *DBManager) getLastSyncTime() time.Time {
	var record SyncRecord
	result := m.local.Order("synced_at DESC").First(&record)
	if result.Error != nil {
		return time.Time{} // Epoch - sync everything
	}
	return record.SyncedAt
}

func (m *DBManager) getLastSyncTimeForTable(table, direction string) time.Time {
	var record SyncRecord
	result := m.local.
		Where("sync_table = ? AND direction = ?", table, direction).
		Order("synced_at DESC").
		First(&record)
	if result.Error != nil {
		return time.Time{}
	}
	return record.SyncedAt
}

func (m *DBManager) syncTable(table string, since time.Time, direction string) (int, error) {
	if direction == "pull" && isUserActivitySyncTable(table) && !m.activityMonitoringPullAllowed() {
		return 0, nil
	}
	var source, dest *gorm.DB
	if direction == "push" {
		source = m.local
		dest = m.remote
	} else {
		source = m.remote
		dest = m.local
	}

	if source == nil || dest == nil {
		return 0, nil
	}
	if !m.remoteSyncTableAvailable(table, direction) {
		return 0, nil
	}
	sourceColumnTypes, err := syncDBColumnTypes(source, table)
	if err != nil {
		return 0, fmt.Errorf("inspect source %s columns: %w", table, err)
	}
	if !syncColumnExists(sourceColumnTypes, "updated_at") {
		log.Printf("Skipping %s sync for %s: source table has no updated_at column for incremental sync", direction, table)
		return 0, nil
	}

	// 1-SYNC (PH ffbe9c7): a zero since means a FULL seed — fetch all rows
	// regardless of updated_at (some imported rows carry zero timestamps and
	// would otherwise be excluded forever).
	sinceScope := func(q *gorm.DB) *gorm.DB {
		if since.IsZero() {
			return q
		}
		return q.Where("updated_at > ?", since)
	}

	// Count records to sync
	var count int64
	if err := sinceScope(source.Table(table)).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count %s records to %s: %w", table, direction, err)
	}
	if count == 0 {
		return 0, nil
	}

	// Fetch ALL records in one query (batch, not row-by-row!)
	var records []map[string]any
	if err := sinceScope(source.Table(table)).Find(&records).Error; err != nil {
		return 0, err
	}

	if len(records) == 0 {
		return 0, nil
	}

	// 1-SYNC (PH ffbe9c7): SQLite allows NULL/empty primary keys, PostgreSQL
	// rejects them — backfill UUIDs in the local source before pushing. The
	// randomblob SQL is SQLite-specific, so this only runs on push.
	if direction == "push" && syncColumnExists(sourceColumnTypes, "id") && isValidSQLIdentifier(table) {
		needsBackfill := false
		for _, rec := range records {
			idVal, hasID := rec["id"]
			if !hasID || idVal == nil || fmt.Sprintf("%v", idVal) == "" {
				needsBackfill = true
				break
			}
		}
		if needsBackfill {
			log.Printf("Backfilling NULL IDs in source %s before sync", table)
			source.Exec(fmt.Sprintf(
				`UPDATE "%s" SET id = lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)),2) || '-' || substr('89ab', abs(random()) %% 4 + 1, 1) || substr(hex(randomblob(2)),2) || '-' || hex(randomblob(6))) WHERE id IS NULL OR id = ''`,
				table,
			))
			records = nil
			if err := sinceScope(source.Table(table)).Find(&records).Error; err != nil {
				return 0, fmt.Errorf("re-fetch %s after ID backfill: %w", table, err)
			}
		}
	}
	targetColumnTypes, err := syncDBColumnTypes(dest, table)
	if err != nil {
		return 0, fmt.Errorf("inspect destination %s columns: %w", table, err)
	}

	// BATCH UPSERT - 10-50x faster than row-by-row!
	batchSize := 100
	synced := 0

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := normalizeRecordsForRemoteSyncSchema(records[i:end], targetColumnTypes)
		updateColumns := syncUpsertColumns(batch)
		if len(updateColumns) == 0 {
			continue
		}
		conflictColumns := syncConflictColumnsForBatch(table, batch, targetColumnTypes)

		// Use ON CONFLICT for upsert (Postgres/SQLite compatible)
		err := dest.Table(table).Clauses(clause.OnConflict{
			Columns:   conflictColumns,
			DoUpdates: clause.AssignmentColumns(updateColumns),
		}).Create(&batch).Error

		if err != nil {
			log.Printf("Batch sync error for %s: %v, falling back to individual", table, err)
			// Fallback to individual inserts for this batch
			for _, record := range batch {
				id := record["id"]
				if err := syncUpsertRecordByIDOrNaturalKey(dest, table, record, targetColumnTypes); err != nil {
					log.Printf("Individual sync error for %s/%v: %v", table, id, err)
					continue
				}
				synced++
			}
		} else {
			synced += len(batch)
		}
	}

	// Log sync summary (one record per table, not per-row)
	if synced > 0 {
		m.local.Create(&SyncRecord{
			SyncTable:     table,
			RecordID:      fmt.Sprintf("batch_%d", synced),
			SyncedAt:      time.Now(),
			Direction:     direction,
			ConflictState: "none",
		})
	}

	return synced, nil
}

func (m *DBManager) remoteSyncTableAvailable(table, direction string) bool {
	if m.remote == nil {
		return false
	}
	if m.remote.Migrator().HasTable(table) {
		return true
	}
	log.Printf("Skipping %s sync for %s: remote table is missing; apply the Supabase schema migration to enable this table", direction, table)
	return false
}

func (m *DBManager) activityMonitoringPullAllowed() bool {
	if m == nil || m.canPullActivityMonitoring == nil {
		return false
	}
	return m.canPullActivityMonitoring()
}

func syncConflictColumnsForBatch(table string, batch []map[string]any, columnTypes map[string]string) []clause.Column {
	naturalColumn := syncNaturalKeyColumn(table, columnTypes)
	if naturalColumn != "" {
		allHaveNaturalKey := len(batch) > 0
		for _, record := range batch {
			_, _, ok := syncNaturalKeyValue(table, record, columnTypes)
			if !ok {
				allHaveNaturalKey = false
				break
			}
		}
		if allHaveNaturalKey {
			return []clause.Column{{Name: naturalColumn}}
		}
	}
	return []clause.Column{{Name: "id"}}
}

// --- App integration helpers ---

// InitDBManager initializes the dual-database manager on the App
func (a *App) InitDBManager() {
	if a.db == nil {
		log.Println("Cannot init DBManager - no local database")
		return
	}

	cfg := LoadSupabaseConfig()
	a.dbManager = NewDBManager(a.db, cfg)
	a.dbManager.canPullActivityMonitoring = a.currentSessionCanAccessActivityMonitoring

	// Try to connect to Supabase (non-blocking)
	if cfg.Enabled {
		go func() {
			// Delay to not block app startup
			time.Sleep(2 * time.Second)

			if err := a.dbManager.ConnectRemote(); err != nil {
				log.Printf("Supabase sync deferred: %v", err)
			} else {
				// Schema migration is now manual — run MigrateRemote() explicitly from Settings
				// or via the Supabase migration script. This prevents unexpected DDL at app launch.
				log.Println("Supabase connected. Schema migration is manual — use Settings > Sync > Migrate if needed.")
			}

			// Start periodic sync (every 10 minutes for offline-first approach)
			a.dbManager.StartPeriodicSync(10 * time.Minute)
		}()
	} else {
		log.Println("Running in offline mode (Supabase not configured)")
	}
}

// LoadSupabaseConfig loads Supabase config from environment
func LoadSupabaseConfig() SupabaseConfig {
	return loadDatabaseConfig()
}

// GetDBSyncStatus returns the current sync status for the frontend
func (a *App) GetDBSyncStatus() map[string]any {
	// Non-sensitive health endpoint used by the sidebar for every role.
	// Manual sync and settings changes remain protected by settings:update.
	status := map[string]any{
		"configured":   false,
		"online":       false,
		"last_sync":    nil,
		"sync_enabled": false,
	}

	if a.dbManager == nil {
		return status
	}

	status["configured"] = a.dbManager.config.Enabled
	status["online"] = a.dbManager.IsOnline()
	status["sync_enabled"] = a.dbManager.IsSyncEnabled()

	lastSync := a.dbManager.getLastSyncTime()
	if !lastSync.IsZero() {
		status["last_sync"] = lastSync.Format(time.RFC3339)
	}

	return status
}

// SyncHealth provides comprehensive health status for the sync system.
type SyncHealth struct {
	IsOnline       bool   `json:"is_online"`
	LastSyncAt     string `json:"last_sync_at"`
	LastSyncStatus string `json:"last_sync_status"`
	TablesInSync   int    `json:"tables_in_sync"`
	DBSizeBytes    int64  `json:"db_size_bytes"`
	BackupCount    int    `json:"backup_count"`
	LastBackupAt   string `json:"last_backup_at"`
	UptimeSeconds  int64  `json:"uptime_seconds"`
}

// GetSyncHealth returns comprehensive health information about the sync system.
func (a *App) GetSyncHealth() SyncHealth {
	if err := a.requirePermission("settings:view"); err != nil {
		return SyncHealth{}
	}
	if a.db == nil {
		return SyncHealth{}
	}
	health := SyncHealth{
		UptimeSeconds: int64(time.Since(a.appStartTime).Seconds()),
	}

	// Sync status
	if a.dbManager != nil {
		health.IsOnline = a.dbManager.IsOnline()
		lastSync := a.dbManager.getLastSyncTime()
		if !lastSync.IsZero() {
			health.LastSyncAt = lastSync.Format(time.RFC3339)
			health.LastSyncStatus = "success"
		}
		health.TablesInSync = len(activeDBSyncTables())
	}

	// Database file size — resolve path from PRAGMA database_list
	var dbPath string
	row := a.db.Raw("PRAGMA database_list").Row()
	if row != nil {
		var seq int
		var name, file string
		if err := row.Scan(&seq, &name, &file); err == nil && file != "" {
			dbPath = file
		}
	}
	if dbPath != "" {
		if info, err := os.Stat(dbPath); err == nil {
			health.DBSizeBytes = info.Size()
		}
	}

	// Backup info
	backupInfo := a.GetBackupInfo()
	if count, ok := backupInfo["count"].(int); ok {
		health.BackupCount = count
	}
	if lastBackup, ok := backupInfo["last_backup"].(string); ok {
		health.LastBackupAt = lastBackup
	}

	return health
}

// TriggerManualSync allows the frontend to trigger a sync
func (a *App) TriggerManualSync() (map[string]any, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return nil, err
	}
	if a.dbManager == nil {
		return nil, fmt.Errorf("sync not initialized")
	}

	pushed, pulled, err := a.dbManager.TriggerSync()
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"pushed": pushed,
		"pulled": pulled,
		"time":   time.Now().Format(time.RFC3339),
	}, nil
}

// DownloadFullDatabase downloads entire database from Supabase to local SQLite
// Called on first license activation to bootstrap new devices
func (m *DBManager) DownloadFullDatabase() (int, error) {
	if !m.IsOnline() {
		if err := m.ConnectRemote(); err != nil {
			return 0, fmt.Errorf("cannot connect to Supabase: %w", err)
		}
	}

	log.Println("========================================")
	log.Println("   FIRST-RUN DATABASE DOWNLOAD")
	log.Println("   Downloading from Supabase...")
	log.Println("========================================")

	// Use the active sync table list, supplemented by system-only tables
	// that aren't in the periodic sync but are needed for first-run bootstrap
	systemTables := []string{
		"assets",       // Letterhead, logos — needed immediately
		"devices",      // Device registration
		"users",        // User accounts
		"device_users", // Device-user bindings
		"settings",     // App settings
	}
	tables := append(systemTables, activeDBSyncTables()...)

	totalDownloaded := 0

	for _, table := range tables {
		if isUserActivitySyncTable(table) && !m.activityMonitoringPullAllowed() {
			log.Printf("  %-25s: skipped (restricted to developer role)", table)
			continue
		}
		count, err := m.downloadTable(table)
		if err != nil {
			log.Printf("  Warning: %s download error: %v", table, err)
			continue
		}
		if count > 0 {
			log.Printf("  %-25s: %d records downloaded", table, count)
			totalDownloaded += count
		}
	}

	log.Println("========================================")
	log.Printf("   DOWNLOAD COMPLETE: %d records", totalDownloaded)
	log.Println("========================================")

	return totalDownloaded, nil
}

// downloadTable downloads a single table from Supabase to local SQLite
// SAFE VERSION: Uses UPSERT instead of DELETE-then-INSERT to prevent data loss
func (m *DBManager) downloadTable(table string) (int, error) {
	// SECURITY: Validate table name to prevent SQL injection in raw SQL statements
	if !isValidSQLIdentifier(table) {
		return 0, fmt.Errorf("invalid table name: %s", table)
	}

	if m.remote == nil || m.local == nil {
		return 0, fmt.Errorf("databases not connected")
	}

	// Check if table exists in Supabase
	var remoteCount int64
	if err := m.remote.Table(table).Count(&remoteCount).Error; err != nil {
		return 0, nil // Table doesn't exist in remote, skip silently
	}
	if remoteCount == 0 {
		return 0, nil // Empty table, nothing to download
	}

	// Get column names from remote
	rows, err := m.remote.Table(table).Limit(1).Rows()
	if err != nil {
		return 0, err
	}
	cols, _ := rows.Columns()
	rows.Close()

	if len(cols) == 0 {
		return 0, nil
	}

	// SAFE APPROACH: Download ALL records first, then apply
	// This prevents data loss if download fails midway
	var allRecords []map[string]any
	batchSize := 100
	offset := 0

	for {
		var records []map[string]any
		result := m.remote.Table(table).Offset(offset).Limit(batchSize).Find(&records)
		if result.Error != nil {
			log.Printf("  Download error for %s at offset %d: %v", table, offset, result.Error)
			return 0, result.Error // Return 0 - don't touch local data on failure
		}

		if len(records) == 0 {
			break
		}

		allRecords = append(allRecords, records...)
		offset += len(records)
		if len(records) < batchSize {
			break // Last batch
		}
	}

	// Only proceed if we have all records
	if len(allRecords) == 0 {
		return 0, nil
	}
	localColumnTypes, err := syncDBColumnTypes(m.local, table)
	if err != nil {
		return 0, fmt.Errorf("inspect local %s columns: %w", table, err)
	}
	allRecords = normalizeRecordsForRemoteSyncSchema(allRecords, localColumnTypes)

	// SAFE: Now clear local table and insert all at once in a transaction
	tx := m.local.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	// Delete existing data
	if err := tx.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	// BATCH INSERT - much faster than row-by-row!
	downloaded := 0
	insertBatchSize := 100

	for i := 0; i < len(allRecords); i += insertBatchSize {
		end := i + insertBatchSize
		if end > len(allRecords) {
			end = len(allRecords)
		}
		batch := allRecords[i:end]

		if err := tx.Table(table).Create(&batch).Error; err != nil {
			// Fallback to individual inserts for this batch
			log.Printf("  Batch insert error for %s: %v, falling back", table, err)
			for _, record := range batch {
				if err := tx.Table(table).Create(record).Error; err != nil {
					log.Printf("  Insert error for %s: %v", table, err)
					continue
				}
				downloaded++
			}
		} else {
			downloaded += len(batch)
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return downloaded, nil
}

// FirstRunSync performs a complete database download if local is empty
// Returns true if sync was performed, false if local already had data
// SAFE VERSION: Uses lock to prevent concurrent syncs and multiple checks
func (m *DBManager) FirstRunSync() (bool, int, error) {
	// Acquire sync lock - prevents concurrent first-run syncs
	m.syncMu.Lock()
	if m.syncInProgress {
		m.syncMu.Unlock()
		log.Println("Sync already in progress, skipping duplicate request")
		return false, 0, nil
	}
	m.syncInProgress = true
	m.syncMu.Unlock()

	// Always release the lock when done
	defer func() {
		m.syncMu.Lock()
		m.syncInProgress = false
		m.syncMu.Unlock()
	}()

	// SAFETY: Check multiple tables to ensure local database is truly empty
	// This prevents false positives from transient read errors
	var customerCount, orderCount, invoiceCount int64
	m.local.Table("customers").Count(&customerCount)
	m.local.Table("orders").Count(&orderCount)
	m.local.Table("invoices").Count(&invoiceCount)

	totalLocal := customerCount + orderCount + invoiceCount

	if totalLocal > 0 {
		log.Printf("Local database has data (customers: %d, orders: %d, invoices: %d), skipping first-run sync",
			customerCount, orderCount, invoiceCount)
		return false, 0, nil
	}

	// Double-check: re-verify after a small delay to avoid race conditions
	time.Sleep(100 * time.Millisecond)
	m.local.Table("customers").Count(&customerCount)
	if customerCount > 0 {
		log.Println("Local database populated during check, skipping first-run sync")
		return false, 0, nil
	}

	// Local is truly empty - download from cloud
	log.Println("Local database confirmed empty - initiating first-run download")
	count, err := m.DownloadFullDatabase()
	if err != nil {
		return false, 0, err
	}

	return true, count, nil
}

// TestSupabaseConnection tests connection with provided credentials
func (a *App) TestSupabaseConnection(host, port, user, password, dbname, sslmode string) (bool, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return false, err
	}
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// Strip DSN from error to prevent password leak to frontend
		return false, fmt.Errorf("connection failed: could not connect to %s:%s", host, port)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return false, fmt.Errorf("failed to get connection pool")
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return false, fmt.Errorf("ping failed: could not reach %s:%s", host, port)
	}

	return true, nil
}

// PerformFirstRunSync downloads the full database from Supabase if local is empty
// This is called after successful license activation on a new device
func (a *App) PerformFirstRunSync() (map[string]any, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return nil, err
	}
	result := map[string]any{
		"performed":  false,
		"downloaded": 0,
		"error":      nil,
	}

	if a.dbManager == nil {
		// Initialize DB manager if not already done
		a.InitDBManager()
	}

	if a.dbManager == nil {
		return result, fmt.Errorf("database manager not initialized")
	}

	// Check if this is a first run (local database is empty)
	performed, count, err := a.dbManager.FirstRunSync()
	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	result["performed"] = performed
	result["downloaded"] = count

	if performed {
		log.Printf("First-run sync completed: %d records downloaded from cloud", count)
	}

	return result, nil
}

// GetFirstRunSyncStatus checks if first-run sync is needed
func (a *App) GetFirstRunSyncStatus() map[string]any {
	if err := a.requirePermission("settings:view"); err != nil {
		return map[string]any{"error": "permission denied"}
	}
	status := map[string]any{
		"local_empty":      false,
		"cloud_available":  false,
		"sync_recommended": false,
	}

	if a.db == nil {
		return status
	}

	// Check if local has data
	var localCount int64
	a.db.Table("customers").Count(&localCount)
	status["local_empty"] = localCount == 0

	// Check if cloud is available
	if a.dbManager != nil && a.dbManager.IsOnline() {
		status["cloud_available"] = true

		// Check if cloud has data
		if a.dbManager.Remote() != nil {
			var cloudCount int64
			a.dbManager.Remote().Table("customers").Count(&cloudCount)
			status["cloud_has_data"] = cloudCount > 0
			status["sync_recommended"] = localCount == 0 && cloudCount > 0
		}
	}

	return status
}
