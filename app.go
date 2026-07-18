package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"ph_holdings_app/integration"
	msgraph "ph_holdings_app/microsoft_graph"
	"ph_holdings_app/pkg/compliance"
	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/graph"
	"ph_holdings_app/pkg/infra/audit"
	"ph_holdings_app/pkg/infra/deploy"
	"ph_holdings_app/pkg/infra/events"
	"ph_holdings_app/pkg/overlay"
	"ph_holdings_app/pkg/runtime/composition"
)

// App struct
type App struct {
	ctx                       context.Context
	config                    *Config // Centralized configuration
	db                        *gorm.DB
	cache                     *Cache                      // In-memory cache for performance (P2 optimization)
	geometryBridge            *GeometryBridge             // Geometry pipeline integration
	fileWatcher               *FileWatcher                // OneDrive folder monitoring
	syncService               *SyncService                // OneDrive/SharePoint sync service
	graphClient               msgraph.GraphClient         // Microsoft Graph API client
	toolsValidator            *integration.ToolsValidator // External tools validation
	emailService              *EmailService               // Email service (wraps graphClient)
	authManager               *AuthManager                // OAuth2 PKCE authentication manager
	archaeologist             *ArchaeologistService       // Archaeology service for workspace scanning
	ocrService                *SimpleOCRService           // OCR service (Simplified Fly.io integration)
	settingsService           *SettingsService            // Settings service (encrypted persistence + GPU detection)
	jobQueue                  *JobQueue                   // Background job processing (reports, etc.)
	classifier                *DocumentClassifier         // Document type classifier (Invoice, RFQ, Quote, etc.)
	graphService              *graph.GraphService         // Entity graph service (Customer360 relationships)
	dbManager                 *DBManager                  // Dual-database manager (local SQLite + remote PostgreSQL)
	dbSyncService             *DBSyncService              // Supabase bidirectional sync service
	msgParser                 *MSGParserService           // MSG parser service for Outlook .msg files
	fieldCrypto               *FieldCrypto                // Field-level encryption (HKDF + AES-256-GCM)
	logFile                   *os.File                    // Log file handle for cleanup (FIX: resource leak)
	appStartTime              time.Time                   // App start time for uptime tracking
	bgSyncStop                chan struct{}               // Stop channel for StartBackgroundDBSync goroutine
	bgSyncWG                  sync.WaitGroup              // WaitGroup for StartBackgroundDBSync goroutine
	bgSyncStopOnce            sync.Once                   // Ensures bgSyncStop is closed exactly once
	collaborationSyncStop     chan struct{}               // Stop channel for collaboration polling loop
	collaborationSyncWG       sync.WaitGroup              // WaitGroup for collaboration polling loop
	collaborationSyncStopOnce sync.Once                   // Ensures collaboration stop is closed exactly once
	collaborationSyncMu       sync.Mutex                  // Prevent overlapping collaboration sync runs
	collaborationSyncInitMu   sync.Mutex                  // Guards collaboration loop initialization/reset
	shutdownOnce              sync.Once                   // Ensures app shutdown cleanup runs exactly once
	startupImporting          bool                        // Bypass RBAC during startup data import
	startupImportStartTime    time.Time                   // Timestamp when startup import began (for timeout enforcement)
	currentUser               *User                       // Current logged-in user
	currentUserID             string                      // Current user ID for quick access
	interactiveSessionID      string                      // DB-backed UserSession row for the interactive login (Wave 5 Mission B)
	interactiveLastTouch      time.Time                   // Last bound-call activity for the inactivity timeout
	interactiveLastPersist    time.Time                   // Last time activity was flushed to the session row (throttled)
	interactiveTimeout        time.Duration               // Idle window for interactive logins; 0 = default (Wave 6 Mission C.2)
	services                  AppServices                 // Domain service implementations for Wails delegation
	eventBus                  events.Bus                  // In-process domain event bus (publishers wired in)
	complianceHook            *compliance.ComplianceHook  // Compliance subscriber: validates finance events
	composition               *composition.Root           // Shared composition seam (overlay → DB → bus → compliance)
}

type tableNamer interface {
	TableName() string
}

// ThreeRegime is exposed for bindings (R1/R2/R3 JSON fields).
type ThreeRegime struct {
	R1 float64 `json:"r1"`
	R2 float64 `json:"r2"`
	R3 float64 `json:"r3"`
}

// =============================================================================
// CSRF PROTECTION - Defense-in-depth for state-changing operations
// =============================================================================
// While Wails bindings are already secure against external requests,
// CSRF tokens provide additional protection for critical financial operations.

var (
	csrfTokens     = make(map[string]time.Time) // token -> creation time
	csrfTokenMutex sync.Mutex
)

// GetCSRFToken generates a new CSRF token valid for 1 hour
// Used for defense-in-depth on critical operations (payments, invoices)
func (a *App) GetCSRFToken() string {
	token := generateSecureToken(32)

	csrfTokenMutex.Lock()
	defer csrfTokenMutex.Unlock()

	// Clean expired tokens (older than 1 hour)
	now := time.Now()
	for t, created := range csrfTokens {
		if now.Sub(created) > time.Hour {
			delete(csrfTokens, t)
		}
	}

	csrfTokens[token] = now
	log.Printf("🔐 CSRF token generated (active tokens: %d)", len(csrfTokens))
	return token
}

// ValidateCSRFToken checks if a token is valid and consumes it (one-time use)
func (a *App) ValidateCSRFToken(token string) bool {
	csrfTokenMutex.Lock()
	defer csrfTokenMutex.Unlock()

	if created, ok := csrfTokens[token]; ok {
		if time.Since(created) < time.Hour {
			delete(csrfTokens, token) // One-time use - consume on validation
			log.Printf("✅ CSRF token validated and consumed")
			return true
		}
		delete(csrfTokens, token) // Expired - remove
		log.Printf("⚠️ CSRF token expired")
	}
	log.Printf("❌ CSRF token invalid or not found")
	return false
}

// generateSecureToken creates a cryptographically secure random token
func generateSecureToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// SECURITY: crypto/rand failure is a critical system issue.
		// Do NOT fall back to predictable tokens - that silently degrades security.
		log.Fatalf("CRITICAL: crypto/rand.Read failed, cannot generate secure token: %v", err)
	}
	return hex.EncodeToString(bytes)
}

// sanitizeSearchQuery removes SQL special characters from user search input
// to prevent SQL injection via LIKE wildcards and escape sequences.
// This is a defense-in-depth measure on top of parameterized queries.
func sanitizeSearchQuery(query string) string {
	// Remove SQL wildcards and dangerous characters
	sanitized := strings.ReplaceAll(query, "%", "")
	sanitized = strings.ReplaceAll(sanitized, "_", "")
	sanitized = strings.ReplaceAll(sanitized, "'", "")
	sanitized = strings.ReplaceAll(sanitized, "\"", "")
	sanitized = strings.ReplaceAll(sanitized, "\\", "")
	sanitized = strings.ReplaceAll(sanitized, ";", "")
	sanitized = strings.ReplaceAll(sanitized, "--", "")
	sanitized = strings.ReplaceAll(sanitized, "/*", "")
	sanitized = strings.ReplaceAll(sanitized, "*/", "")
	sanitized = strings.TrimSpace(sanitized)
	return sanitized
}

// TimeNow exposes time.Time to Wails bindings.
func (a *App) TimeNow() time.Time {
	return time.Now()
}

// ApplicationPaths represents application directory paths
type ApplicationPaths struct {
	ProjectRoot   string `json:"project_root"`
	BatchOutput   string `json:"batch_output"`
	TestData      string `json:"test_data"`
	ReportOutput  string `json:"report_output"`
	AsymmMathRoot string `json:"asymm_math_root"`
}

// getAppPaths returns application paths without RBAC check (for internal backend use).
// Use GetApplicationPaths() for frontend-facing calls that need permission gating.
func (a *App) getAppPaths() *ApplicationPaths {
	projectRoot, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get working directory: %v, using fallback", err)
		if ex, exErr := os.Executable(); exErr == nil {
			projectRoot = filepath.Dir(ex)
		} else {
			projectRoot = "."
		}
	}

	// Production fix: if CWD is not writable (e.g., "/" from Finder launch),
	// use the data directory where the DB lives for all exports
	testDir := filepath.Join(projectRoot, "exports")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		// CWD not writable — use the deployment data plane
		// (%APPDATA%\Asymmetrica\<slug>\data). Never the legacy %APPDATA%\AsymmFlow.
		if dataDir := deploy.DataDir(); dataDir != "" {
			os.MkdirAll(dataDir, 0755)
			log.Printf("CWD not writable, using data dir: %s", dataDir)
			projectRoot = dataDir
		}
	}

	asymmMathRoot := os.Getenv("ASYMM_MATH_ROOT")
	if asymmMathRoot == "" {
		potentialPath := filepath.Join(filepath.Dir(projectRoot), "asymm_all_math")
		if _, err := os.Stat(potentialPath); err == nil {
			asymmMathRoot = potentialPath
		} else {
			asymmMathRoot = filepath.Join(projectRoot, "asymm_mathematical_organism")
		}
	}

	return &ApplicationPaths{
		ProjectRoot:   projectRoot,
		BatchOutput:   filepath.Join(projectRoot, "batch_output"),
		TestData:      filepath.Join(projectRoot, "test_data"),
		ReportOutput:  filepath.Join(projectRoot, "reports"),
		AsymmMathRoot: asymmMathRoot,
	}
}

// GetApplicationPaths returns the current application paths (RBAC-guarded for frontend calls)
func (a *App) GetApplicationPaths() *ApplicationPaths {
	if err := a.requirePermission("settings:view"); err != nil {
		return nil
	}
	return a.getAppPaths()
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// rotateLogFile checks if the log file exceeds maxSize bytes and rotates it.
// Keeps up to keepCount rotated files (app_debug.log.1 through .N).
func rotateLogFile(logPath string, maxSize int64, keepCount int) {
	info, err := os.Lstat(logPath) // Lstat: don't follow symlinks
	if err != nil || info.Mode()&os.ModeSymlink != 0 || info.Size() < maxSize {
		return // File doesn't exist or is within size limit
	}

	// Delete the oldest rotated log
	oldest := fmt.Sprintf("%s.%d", logPath, keepCount)
	os.Remove(oldest)

	// Shift existing rotated logs: .4 → .5, .3 → .4, etc.
	for i := keepCount - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", logPath, i)
		dst := fmt.Sprintf("%s.%d", logPath, i+1)
		os.Rename(src, dst)
	}

	// Rotate current log to .1
	os.Rename(logPath, logPath+".1")
	log.Printf("Log file rotated (was %d MB)", info.Size()/(1024*1024))
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.appStartTime = time.Now()

	appendStartupDiagnostic("ONSTARTUP: begin")
	appendStartupDiagnostic("ONSTARTUP: cwd=%s", func() string { d, _ := os.Getwd(); return d }())
	if ex, err := os.Executable(); err == nil {
		appendStartupDiagnostic("ONSTARTUP: exe=%s", ex)
	}

	// 1. Setup File Logging with rotation (FIX: Handle error, store file handle for cleanup)
	logPath := appDebugLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		appendStartupDiagnostic("ONSTARTUP: failed to create log directory for %s: %v", logPath, err)
	}
	rotateLogFile(logPath, 50*1024*1024, 5) // 50MB max, keep 5 rotated logs
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		appendStartupDiagnostic("ONSTARTUP: failed to open app log %s: %v", logPath, err)
		log.Printf(" Failed to open log file: %v, logging to stderr", err)
	} else {
		a.logFile = logFile // Store for cleanup in shutdown()
		log.SetOutput(logFile)
		appendStartupDiagnostic("ONSTARTUP: app log path=%s", logPath)
	}

	// 2. Initialize Structured Logger (dev mode for console, production mode for cloud)
	// Determine mode from environment or default to dev
	logMode := os.Getenv("LOG_MODE")
	if logMode == "" {
		logMode = "dev" // Default to dev mode (emoji + human-readable)
	}

	InitGlobalLogger(LoggerConfig{
		Mode:       logMode,
		Level:      LevelInfo, // Default to INFO level (can be configured later)
		OutputFile: logFile,   // Also write JSON to log file
	})

	defer func() {
		if r := recover(); r != nil {
			appendStartupDiagnostic("ONSTARTUP: panic: %v", r)
			log.Printf("🔥 CRITICAL PANIC: %v", r)
			if AppLogger != nil {
				AppLogger.Fatal("Application panic", fmt.Errorf("%v", r), map[string]any{
					"stack_trace": AppLogger.GetStackTrace(2),
				})
			}
		}
	}()

	// Use structured logging for startup banner
	AppLogger.Startup("STARTUP SEQUENCE INITIATED", map[string]any{
		"version":    "1.0.0",
		"go_version": goruntime.Version(),
		"os":         goruntime.GOOS,
		"arch":       goruntime.GOARCH,
	})

	// P1 FIX: Initialize security enhancements
	AppLogger.Info("Initializing security enhancements", nil)
	InitSecurityEnhancements(AppLogger)

	// Load configuration FIRST (from .env or environment)
	cfg, err := LoadConfig()
	if err != nil {
		AppLogger.Warn("Config error (using defaults)", map[string]any{
			"error": err.Error(),
		})
		// Create minimal default config. Use the same database path resolver
		// as the main config path so packaged apps do not silently fall back
		// to stale machine-level databases when validation/logging config fails.
		cfg = &Config{
			Database: DatabaseConfig{Path: getDatabasePath()},
			App: AppConfig{
				LogLevel:             "info",
				WatcherDebounceMS:    300,
				WatcherQueueSize:     1000,
				EnableFileWatcher:    true,
				EnableGeometryBridge: true,
				EnableAutoBackup:     false,
				BackupRetentionDays:  30,
			},
		}
		cfg.Tools.detectTools()
		cfg.LogConfig()
	}
	a.config = cfg

	// Load company/division overlay (config-driven profiles) through the
	// shared composition seam — the same search-dir cascade as
	// LoadConfig / getDatabasePath (executable dirs → CWD/data → CWD →
	// platform app-data dir).
	a.composition = composition.NewRoot()
	setActiveOverlay(a.composition.LoadOverlay(composition.StandardOverlayDirs()))

	// Wave 12.5 B3: wire the crm delivery-terms composer to the active overlay so
	// a new offer's empty delivery terms compose from ITS division (not the
	// hardcoded default-division column default). The default/empty case composes
	// "DAP Bahrain at your store or Acme Instrumentation" — byte-identical to the
	// legacy GORM column default.
	crm.ComposeOfferDeliveryTerms = func(division string) string {
		return "DAP Bahrain at your store or " + activeOverlay.NormalizeDivisionName(division)
	}

	// Initialize SQLite database (local, no network dependency)
	dbPath := cfg.Database.Path
	if dbPath == "" {
		dbPath = getDatabasePath()
	}

	// Mission DP1: getDatabasePath() (→ deploy.ResolveDatabasePath) already
	// returns an absolute, deterministic path — the six-priority CWD/exe-dir/
	// legacy-AppData archaeology that used to live here was retired. A relative
	// value can only arise from a relative PH_DB_PATH under an empty CWD; resolve
	// it against CWD as a last courtesy so `wails dev` from an odd shell survives.
	if !filepath.IsAbs(dbPath) {
		if abs, absErr := filepath.Abs(dbPath); absErr == nil {
			dbPath = abs
		}
	}

	// Ensure the data plane directory exists.
	dbDir := filepath.Dir(dbPath)
	if dbDir != "" && dbDir != "." {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			AppLogger.Warn("Failed to create database directory", map[string]any{
				"path":  dbDir,
				"error": err.Error(),
			})
		}
	}

	// Mission DP1 update contract: seed an absent data plane from the packaged
	// canon, refuse a downgrade, and back up + migrate (on a copy) before an
	// upgrade — all BEFORE the connection pool opens. A present, current
	// database is opened untouched (the anti-reseed invariant). This is the ONLY
	// path that may seed or migrate; nothing here ever touches the legacy dir.
	contractRes, contractErr := deploy.EnsureDatabase(deploy.ContractConfig{
		DBPath:       dbPath,
		SeedPath:     deploy.PackagedSeedPath(),
		BinarySchema: deploy.BinarySchemaVersion(),
		Migrate:      migrateDatabaseFileForContract,
		ForceReseed:  deploy.ForceReseedRequested(),
		Now:          time.Now(),
		Logf:         log.Printf,
	})
	if contractErr != nil {
		// Downgrade refusal or a failed migration: do NOT open the database. The
		// original data plane is intact; surface the reason and stop startup.
		AppLogger.Error("Database update contract refused to proceed", contractErr, map[string]any{
			"path": dbPath,
		})
		appendStartupDiagnostic("DB CONTRACT REFUSED: %v", contractErr)
		if ctx != nil {
			runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
				Type:    runtime.ErrorDialog,
				Title:   "Cannot Open Database",
				Message: contractErr.Error(),
			})
		} else {
			fmt.Fprintf(os.Stderr, "FATAL: %s\n", contractErr.Error())
		}
		return
	}
	appendStartupDiagnostic("DB CONTRACT: action=%s from=%d to=%d backup=%s",
		contractRes.Action, contractRes.FromSchema, contractRes.ToSchema, contractRes.BackupPath)

	AppLogger.Info("Connecting to SQLite database", map[string]any{
		"path": dbPath,
	})

	// Open SQLite through the composition seam, with PRAGMAs in the DSN so
	// every pooled connection gets them (WAL mode → MaxOpenConns > 1 is safe).
	// Wave 3 fix: the previous mattn-style params (?_journal_mode=WAL&…) were
	// silently IGNORED by the ncruces driver — the app actually ran
	// journal_mode=DELETE. composition.DefaultPragmas uses the ncruces
	// ?_pragma= form the driver honors (pinned by TestSQLiteDSN_PragmasAreHonored).
	dsn := composition.SQLiteDSN(filepath.Clean(dbPath), composition.DefaultPragmas...)
	a.db, err = a.composition.OpenSQLite(dsn, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // Reduce log noise in production
		// CRITICAL: SQLite cannot modify FK constraints on existing tables
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		// FIX 3: Sanitize error message - don't expose internal paths to user
		sanitizedMsg := "Failed to connect to database. Please check your installation and ensure the application has proper file permissions."

		// Log detailed error internally for debugging
		AppLogger.Error("Database connection failed", err, map[string]any{
			"path": dbPath,
		})

		if ctx != nil {
			runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
				Type:    runtime.ErrorDialog,
				Title:   "Database Connection Error",
				Message: sanitizedMsg, // User sees sanitized message only
			})
		} else {
			// Fallback: Write sanitized message to stderr
			fmt.Fprintf(os.Stderr, "FATAL DATABASE ERROR: %s\n", sanitizedMsg)
		}
		return
	}

	AppLogger.Info("SQLite database connected successfully", map[string]any{
		"path": dbPath,
	})

	// Configure connection pool for SQLite (limited settings)
	sqlDB, poolErr := a.db.DB()
	if poolErr == nil {
		sqlDB.SetMaxOpenConns(4) // WAL mode supports concurrent readers; PRAGMAs set via DSN
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetConnMaxLifetime(time.Hour)

		// P0 FIX: Enable foreign key constraints at runtime
		// This ensures referential integrity for all operations
		if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
			AppLogger.Warn("Failed to enable foreign keys", map[string]any{
				"error": err.Error(),
			})
		} else {
			AppLogger.Info("Foreign key constraints enabled", nil)
		}

		// Additional PRAGMAs not settable via DSN
		sqlDB.Exec("PRAGMA temp_store = MEMORY")
		sqlDB.Exec("PRAGMA mmap_size = 268435456")
		log.Println("SQLite PRAGMAs applied (WAL + busy_timeout via DSN, temp_store + mmap_size manual)")
	}

	// P2 FIX: Initialize in-memory cache for performance
	a.cache = NewCache()
	AppLogger.Info("Performance cache initialized", map[string]any{
		"ttl_short":  CacheTTLShort.String(),
		"ttl_medium": CacheTTLMedium.String(),
		"ttl_long":   CacheTTLLong.String(),
	})
	a.initServices()
	AppLogger.Info("Domain services initialized", nil)

	// Wire the engine-backed audit recorder into the security audit logger
	// now that the DB is up (Wave 3 B.2: one audit persistence path).
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.SetRecorder(audit.NewRecorder(a.db))
	}

	// Initialize field-level encryption EARLY - before any AutoMigrate or DB operations
	// that may trigger GORM hooks on encrypted models (e.g., CompanyBankAccount).
	// FieldCrypto only needs env vars / hardware ID, not the database.
	fc, fcErr := NewFieldCrypto()
	if fcErr != nil {
		AppLogger.Error("CRITICAL: FieldCrypto initialization failed — field-level encryption and document HMAC integrity are DISABLED. Encrypted fields will not be readable/writable.", fcErr, nil)
		log.Printf("CRITICAL: FieldCrypto initialization failed: %v — ALL field encryption disabled", fcErr)
	} else {
		a.fieldCrypto = fc
		globalFieldCrypto = fc
		AppLogger.Info("FieldCrypto initialized (early)", map[string]any{
			"key_derivation": "HKDF-SHA256",
			"cipher":         "AES-256-GCM",
			"key_version":    fc.CurrentVersion(),
		})
	}

	// Startup diagnostic: write progress to known location
	diagLog := func(msg string) {
		appendStartupDiagnostic("%s", msg)
		log.Println(msg)
	}
	diagLog(fmt.Sprintf("STARTUP: DB path = %s", dbPath))

	// Auto-migrate the schema - Full ERP schema
	// ENABLED: Runs on every startup to ensure client databases have latest schema
	// This is IDEMPOTENT - safe to run multiple times, only adds missing columns
	if a.db == nil {
		diagLog("STARTUP: Skipping migration - no DB")
		log.Println("⚠️ Skipping database migration - no database connection")
	} else {
		// Check if DB already has tables — skip AutoMigrate to avoid infinite loop
		// (GORM SQLite migrator has a bug recreating book_bank_reconciliations__temp endlessly)
		var tableCount int64
		a.db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tableCount)
		diagLog(fmt.Sprintf("STARTUP: DB has %d tables", tableCount))

		if tableCount > 50 {
			diagLog("STARTUP: Skipping AutoMigrate — DB schema already established")
			log.Printf("✅ Skipping AutoMigrate — %d tables already exist (schema stable)", tableCount)
		} else {
			diagLog("STARTUP: Beginning AutoMigrate...")
			// ROBUST MIGRATION through the composition seam: migrate each model
			// individually so existing tables (whose constraints SQLite cannot
			// modify) skip gracefully. The trading model-set lives in
			// tradingModels() (trading_models.go); its schema is pinned by
			// TestTradingModels_SchemaGolden.
			models := tradingModels()
			migratedCount, skippedCount := a.composition.MigrateModels(models, func(i, n int, name string, err error) {
				if err != nil {
					log.Printf("⚠️ Migration skipped for %s: %v", name, err)
					diagLog(fmt.Sprintf("MIGRATE[%d/%d] SKIP %s: %v", i, n, name, err))
				}
			})
			diagLog(fmt.Sprintf("STARTUP: AutoMigrate done (%d ok, %d skipped)", migratedCount, skippedCount))
			log.Printf("✅ Database migration complete: %d migrated, %d skipped (existing tables)", migratedCount, skippedCount)
		}

		// Ensure release-critical workflow foundations exist regardless of main migration path.
		foundationErr := a.ensureCriticalDeploymentFoundations()
		if foundationErr != nil {
			AppLogger.Error("Critical deployment foundation initialization failed", foundationErr, nil)
		}
		a.enforceCriticalDeploymentState("startup", foundationErr)

		// Seed license keys (10 per role) if needed. Seed bundles are selected
		// by the overlay (A.3): absent seed_sets → all run, as always.
		if activeOverlay.SeedEnabled("license-keys") {
			// Mission I (I-11): startup uses the unguarded internal; the bound
			// App method now carries a licenses:manage gate.
			if err := seedLicenseKeys(a); err != nil {
				AppLogger.Warn("License key seeding warning", map[string]any{"error": err.Error()})
			}
		}

		// Assign keys to named employees
		if activeOverlay.SeedEnabled("employee-keys") {
			if err := seedEmployeeKeys(a); err != nil {
				AppLogger.Warn("Employee key assignment warning", map[string]any{"error": err.Error()})
			}
		}

		if err := applyDeploymentLicenseActivationFlush(a); err != nil {
			AppLogger.Warn("Deployment license activation flush warning", map[string]any{"error": err.Error()})
		}

		if tableCount <= 50 {
			diagLog("STARTUP: Running custom migrations...")
			a.runCustomMigrations()
			diagLog("STARTUP: Custom migrations done")
		} else {
			diagLog("STARTUP: Skipping custom migrations — schema stable")
		}

		// Run database integrity check on startup
		integrityResult := a.runIntegrityCheck()
		if integrityResult != "ok" {
			AppLogger.Warn("Database integrity issue detected on startup", map[string]any{
				"result": integrityResult,
			})
		}
		if backupResult := a.runScheduledBackupIfDueInternal("startup"); backupResult["success"] == false {
			AppLogger.Warn("Scheduled database backup failed on startup", map[string]any{
				"error": backupResult["error"],
			})
		}

		// Wave 9.8 B4 follow-up (owner-ratified): employee compliance-document
		// expiry scan runs once at startup and then daily, so an expiring
		// visa/CPR/permit surfaces even if nobody opens the compliance tab.
		// The scan itself is idempotent (NotifiedAt-stamped), so overlapping
		// with the opportunistic ListEmployeeDocuments call is harmless. The
		// ticker goroutine exits on ctx.Done() — never leaked (the Wave 9.5
		// NewCache lesson).
		if scanResult, err := a.ScanExpiringEmployeeDocuments(); err != nil {
			AppLogger.Warn("Employee document expiry scan failed on startup", map[string]any{
				"error": err.Error(),
			})
		} else if scanResult.NotifiedCount > 0 {
			AppLogger.Info("Employee document expiry scan notified on startup", map[string]any{
				"notified": scanResult.NotifiedCount,
			})
		}
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if _, err := a.ScanExpiringEmployeeDocuments(); err != nil {
						AppLogger.Warn("Daily employee document expiry scan failed", map[string]any{
							"error": err.Error(),
						})
					}
				}
			}
		}()
	}

	AppLogger.Info("AsymmFlow app started successfully", map[string]any{
		"db_path":       cfg.Database.Path,
		"config_loaded": true,
	})

	// Initialize settings service (encrypted persistence)
	settingsSvc, err := NewSettingsService(a.db)
	if err != nil {
		AppLogger.Error("Settings service initialization failed", err, nil)
	} else {
		a.settingsService = settingsSvc
		AppLogger.Info("Settings Service initialized", map[string]any{
			"encryption": "AES-256",
		})

		// Perform GPU detection on startup
		gpuInfo, err := a.DetectGPU()
		if err != nil {
			AppLogger.GPU("GPU detection", false, map[string]any{
				"error":    err.Error(),
				"fallback": "CPU mode",
			})
		} else if gpuInfo.Detected {
			AppLogger.GPU("GPU detected", true, map[string]any{
				"device_name": gpuInfo.DeviceName,
				"vendor":      gpuInfo.Vendor,
				"vram_mb":     gpuInfo.VRAM,
			})
		} else {
			AppLogger.GPU("GPU detection", false, map[string]any{
				"fallback": "CPU mode",
			})
		}
	}

	// Wire FieldCrypto into SettingsService now that both are ready
	if a.fieldCrypto != nil && a.settingsService != nil {
		a.settingsService.SetFieldCrypto(a.fieldCrypto)
		AppLogger.Info("FieldCrypto wired into SettingsService", nil)
	}

	// Bank account encryption migration — skip for established DBs (already done)
	// This was taking 45s and blocking all API calls on single-connection mode
	{
		var bankTableCount int64
		a.db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&bankTableCount)
		if bankTableCount <= 50 {
			a.migrateBankAccountEncryptionInternal()
		}
	}

	// Initialize geometry bridge
	a.geometryBridge = NewGeometryBridge()
	AppLogger.Info("Geometry Bridge initialized", map[string]any{
		"pipelines": 4,
	})

	// Register Mistral API key provider (encrypted DB -> settings.json -> env fallback)
	SetMistralKeyProvider(func() string {
		// 1. Try encrypted settings DB first
		if a.settingsService != nil && a.fieldCrypto != nil {
			if encVal, err := a.settingsService.GetSetting("apiKeys.mistral_key"); err == nil && encVal != "" {
				// Value is stored pre-encrypted by SetAPIKeys; the IsEncrypted flag
				// triggers SettingsService.decrypt, but that uses the old key.
				// We stored the FieldCrypto-encrypted value directly, so decrypt here.
				if a.fieldCrypto.IsEncrypted(encVal) {
					if decrypted, err := a.fieldCrypto.Decrypt(encVal); err == nil && decrypted != "" {
						return decrypted
					}
				} else if encVal != "" {
					// Plain text value in DB (legacy)
					return encVal
				}
			}
		}

		// 2. Try settings.json file
		userSettings, err := a.loadUserSettings()
		if err == nil {
			if key := getSettingOrDefault(userSettings, "apiKeys.mistral_key", "").(string); key != "" {
				return key
			}
		}

		return ""
	})
	AppLogger.Info("Mistral API key provider registered (encrypted DB priority)", nil)

	// Register AIML API key provider (for Grok — primary Butler chat backend)
	// Uses same "apiKeys.aimlapi_key" setting key as the existing AI connectivity test
	SetAIMLKeyProvider(func() string {
		// 1. Try encrypted settings DB first
		if a.settingsService != nil && a.fieldCrypto != nil {
			if encVal, err := a.settingsService.GetSetting("apiKeys.aimlapi_key"); err == nil && encVal != "" {
				if a.fieldCrypto.IsEncrypted(encVal) {
					if decrypted, err := a.fieldCrypto.Decrypt(encVal); err == nil && decrypted != "" {
						return decrypted
					}
				} else if encVal != "" {
					return encVal
				}
			}
		}
		// 2. Try settings.json file
		userSettings, err := a.loadUserSettings()
		if err == nil {
			if key := getSettingOrDefault(userSettings, "apiKeys.aimlapi_key", "").(string); key != "" {
				return key
			}
		}
		return ""
	})
	AppLogger.Info("AIML API key provider registered (Grok primary backend)", nil)

	// Register AIML model preference provider (overrides default Grok model per environment/user setting)
	SetAIMLModelProvider(func() string {
		if a.settingsService != nil {
			if model, err := a.settingsService.GetSetting("apiKeys.aiml_model"); err == nil && strings.TrimSpace(model) != "" {
				return strings.TrimSpace(model)
			}
		}

		userSettings, err := a.loadUserSettings()
		if err == nil {
			if model := strings.TrimSpace(getSettingOrDefault(userSettings, "apiKeys.aiml_model", "").(string)); model != "" {
				return model
			}
		}

		return ""
	})
	AppLogger.Info("AIML model provider registered (Grok model preference)", nil)

	diagLog("STARTUP: Initializing DBManager...")
	a.InitDBManager()
	a.StartCollaborativeSyncLoop(8 * time.Second)
	diagLog("STARTUP: DBManager done")
	a.dbSyncService = newDBSyncService(a)

	diagLog("STARTUP: Initializing assets...")
	if activeOverlay.SeedEnabled("default-assets") {
		a.InitializeDefaultAssets()
	}
	diagLog("STARTUP: Assets done")

	// Seed default RBAC roles
	if activeOverlay.SeedEnabled("rbac-roles") {
		if err := a.SeedDefaultRoles(); err != nil {
			AppLogger.Error("Failed to seed default roles", fmt.Errorf("RBAC initialization failed: %w", err), map[string]any{
				"stage": "startup",
			})
		}
	}

	// Startup backfill: only repair missing RFQ document tracking.
	// Do NOT auto-rewrite transactional dates on startup.
	var countRFQsNeedingBackfill int64
	if a.db != nil {
		countWrongDates := a.countFutureDatedInvoices()
		a.db.Model(&RFQData{}).Where("document_hash IS NULL OR document_hash = ''").Count(&countRFQsNeedingBackfill)

		if countWrongDates > 0 {
			AppLogger.Warn("Found future-dated invoices during startup; automatic date mutation is disabled", map[string]any{
				"wrong_dates_found": countWrongDates,
			})
		}

		if countRFQsNeedingBackfill > 0 {
			AppLogger.Info("Found RFQs needing document tracking backfill", map[string]any{
				"rfqs_needing_doc_tracking": countRFQsNeedingBackfill,
			})
			rfqFixed, err := a.BackfillRFQDocumentTracking()
			if err != nil {
				AppLogger.Warn("RFQ document tracking backfill failed during startup", map[string]any{
					"error": err.Error(),
				})
			} else {
				AppLogger.Info("RFQ document tracking backfill completed", map[string]any{
					"rfqs_fixed": rfqFixed,
				})
			}
		}
	}

	diagLog("STARTUP: Initializing graph service...")
	a.graphService = graph.NewGraphService(a.db)
	diagLog("STARTUP: Graph done")
	AppLogger.Info("Entity Graph Service initialized", map[string]any{
		"feature": "Customer360 relationships",
	})

	diagLog("STARTUP: Validating tools...")
	a.toolsValidator = integration.NewToolsValidator()
	toolsReport := a.toolsValidator.ValidateAllTools()
	diagLog("STARTUP: Tools done. STARTUP COMPLETE.")
	AppLogger.Info("Tools Validator initialized", map[string]any{
		"summary":                toolsReport.Summary,
		"all_optional_available": toolsReport.AllOptional,
	})

	// Log any missing tools
	if !toolsReport.AllOptional {
		missingTools := a.toolsValidator.GetMissingTools()
		AppLogger.Warn("Missing optional tools", map[string]any{
			"missing_tools": missingTools,
			"help":          "Install instructions available via GetToolsStatus()",
		})
	}

	// Initialize Microsoft Graph client (if Azure configured)
	// Note: Currently uses mock client for development. For production Azure AD integration:
	//   1. Configure Azure AD app registration (Application ID, Client Secret, Tenant ID)
	//   2. Replace NewMockGraphClient with msgraph.NewGraphClient (real client)
	//   3. Add OAuth2 flow to obtain access tokens
	//   4. Enable scopes: Mail.Read, Calendars.Read, Contacts.Read
	if cfg.Azure.Enabled && cfg.Azure.TenantID != "" {
		graphConfig := &msgraph.GraphConfig{
			TenantID:     cfg.Azure.TenantID,
			ClientID:     cfg.Azure.ClientID,
			ClientSecret: cfg.Azure.ClientSecret,
		}
		a.graphClient = msgraph.NewMockGraphClient(graphConfig)
		AppLogger.Info("Microsoft Graph Client initialized", map[string]any{
			"mode":      "MOCK",
			"tenant_id": cfg.Azure.TenantID,
		})
	} else {
		AppLogger.Warn("Azure not configured - using mock Graph client", nil)
		a.graphClient = msgraph.NewMockGraphClient(nil)
	}

	// Initialize sync service
	a.syncService = NewSyncService(a.graphClient, a)
	AppLogger.Info("Sync Service initialized", nil)

	// Initialize email service (wraps graph client)
	a.emailService = NewEmailService(a.graphClient)
	if err := a.emailService.HealthCheck(); err != nil {
		AppLogger.Warn("Email service disabled", map[string]any{
			"error": err.Error(),
		})
	} else {
		AppLogger.Info("Email service initialized", nil)
	}

	// Initialize Simple OCR service (Fly.io + go-fitz)
	ocrSvc, err := NewSimpleOCRService()
	if err != nil {
		AppLogger.Error("OCR service initialization failed", err, map[string]any{
			"features_disabled": "OCR",
		})
	} else {
		a.ocrService = ocrSvc
		AppLogger.Info("Simple OCR Service initialized", map[string]any{
			"engine_1": "go-fitz (FREE, <100ms for vector PDFs)",
			"engine_2": "Fly.io Runtime (68 kernels, 18 languages)",
		})
	}

	// Initialize document classifier
	a.classifier = NewDocumentClassifier()
	AppLogger.Info("Document Classifier initialized", nil)

	// Initialize auth manager and start session cleanup worker
	a.authManager = NewAuthManager(a)
	if a.authManager != nil {
		// Start background session cleanup worker (runs every 6 hours)
		a.authManager.StartSessionCleanupWorker()
		log.Println("✓ Auth Session Manager initialized (cleanup worker started)")

		// Try to load cached token
		if a.authManager.loadTokenCache() {
			log.Println("✓ Authentication token loaded")
		}
	}
	log.Println("  → 2-stage classification: Keywords → Heuristics")

	// Initialize job queue for async operations (reports, etc.)
	if err := a.initializeJobQueueInternal(); err != nil {
		log.Printf("⚠ Job Queue initialization failed: %v", err)
		log.Println("  Async report generation will be disabled")
	}

	// Seed product database (VQC-Ready)
	if activeOverlay.SeedEnabled("demo-products") {
		log.Println("🔧 Attempting to seed product database...")
		if err := a.seedProductDatabaseInternal(); err != nil {
			log.Printf("⚠ Product seeding failed: %v", err)
		} else {
			log.Println("✅ Product database seeding complete")
		}
	}

	// Seed customer database (sample data for development/testing)
	if activeOverlay.SeedEnabled("demo-customers") {
		log.Println("🔧 Attempting to seed customer database...")
		if err := a.SeedCustomerDatabase(); err != nil {
			log.Printf("⚠ Customer seeding failed: %v", err)
		} else {
			log.Println("✅ Customer database seeding complete")
		}
	}

	log.Println("🔧 Attempting to backfill business customer IDs...")
	if results, err := a.backfillBusinessCustomerIDsInternal(); err != nil {
		log.Printf("⚠ Customer ID backfill failed: %v", err)
	} else {
		log.Printf("✅ Customer ID backfill complete: %v", results)
	}

	// Seed bank demo data (sample bank statements for demo purposes)
	if activeOverlay.SeedEnabled("demo-bank") {
		log.Println("🔧 Attempting to seed bank demo data...")
		if err := a.SeedBankDemoData(); err != nil {
			log.Printf("⚠ Bank demo seeding failed: %v", err)
		} else {
			log.Println("✅ Bank demo data seeding complete")
		}
	}

	// Backfill offer_items cost breakdown from costing_line_items
	log.Println("🔧 Attempting to backfill offer_items cost breakdown...")
	if results, err := a.BackfillOfferItemCostBreakdown(); err != nil {
		log.Printf("⚠ Offer items backfill failed: %v", err)
	} else {
		log.Printf("✅ Offer items backfill complete: %v", results)
	}

	// Repair won offers that still have no line items but already have verified
	// 2026 OneDrive opportunity product details.
	log.Println("🔧 Attempting to backfill won offers from opportunity product details...")
	if results, err := a.backfillWonOfferItemsFromOpportunityProductDetailsInternal(); err != nil {
		log.Printf("⚠ Won offer item repair failed: %v", err)
	} else {
		log.Printf("✅ Won offer item repair complete: %v", results)
	}

	log.Println("🔧 Attempting to backfill invoice items from linked orders...")
	if results, err := a.backfillInvoiceItemsFromOrdersInternal(); err != nil {
		log.Printf("⚠ Invoice item backfill failed: %v", err)
	} else {
		log.Printf("✅ Invoice item backfill complete: %v", results)
	}

	// MON-003: fill blank invoice integrity hashes (bulk-imported rows). Salt-gated
	// inside — never persists a SHA-256 fallback hash.
	if _, err := a.backfillInvoiceHashesInternal(); err != nil {
		log.Printf("⚠ Invoice hash backfill failed: %v", err)
	}

	// Auto-import Tally data if tables are empty
	// BE-6 FIX: Use defer + timeout to ensure RBAC always re-enabled even if import hangs/crashes
	if a.db != nil {
		var tallyInvoiceCount int64
		a.db.Model(&TallyInvoiceImport{}).Count(&tallyInvoiceCount)
		if tallyInvoiceCount == 0 {
			log.Println("📊 Tally tables empty - importing historical data from Excel files...")
			a.startupImporting = true
			a.startupImportStartTime = time.Now() // Record when RBAC bypass started
			// CRITICAL: defer guarantees this runs even if ImportAllTallyData panics
			defer func() {
				a.startupImporting = false
				log.Println("🔒 RBAC re-enabled after startup import")
			}()

			// Create context with 5-minute timeout to prevent infinite hang
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			// Run import with timeout protection
			type importResult struct {
				result *TallyImportResult
				err    error
			}
			resultChan := make(chan importResult, 1)
			go func() {
				r, e := a.ImportAllTallyData()
				resultChan <- importResult{r, e}
			}()

			select {
			case res := <-resultChan:
				if res.err != nil {
					log.Printf("⚠ Tally auto-import error: %v", res.err)
				} else {
					log.Printf("✅ Tally auto-import complete: %d records imported, %d duplicates, %d errors",
						res.result.Imported, res.result.Duplicates, res.result.Errors)
				}
			case <-ctx.Done():
				log.Printf("⚠ Tally auto-import timed out after 5 minutes - RBAC re-enabled for safety")
			}
		} else {
			log.Printf("✓ Tally data already loaded (%d invoice records)", tallyInvoiceCount)
		}
	}

	// Wire the in-process event bus + compliance subscriber now that startup
	// bulk imports are done, so runtime writes (file-watcher imports, manual
	// creates) publish domain events without flooding the hook during boot.
	a.initComplianceEventBus()

	// Initialize file watcher using config paths
	if cfg.App.EnableFileWatcher {
		watchConfig := &WatchConfig{
			// Use OneDrive paths from config
			RFQPath:       cfg.OneDrive.RFQPath,
			EHXMLPath:     cfg.OneDrive.EHPath,
			OfferPath:     cfg.OneDrive.OffersPath,
			InvoicePath:   cfg.OneDrive.InvoicesPath,
			Recursive:     true,
			DebounceDelay: time.Duration(cfg.App.WatcherDebounceMS) * time.Millisecond,
			MaxQueueSize:  cfg.App.WatcherQueueSize,
			IncludeExts:   supportedOCRWatcherExtensions(),
		}

		fw, err := NewFileWatcher(watchConfig)
		if err != nil {
			log.Printf("Warning: Failed to create file watcher: %v", err)
		} else {
			a.fileWatcher = fw
			a.registerFileWatcherHandlers()

			// Auto-start only if paths configured
			if watchConfig.hasValidPaths() {
				if err := a.fileWatcher.Start(); err != nil {
					log.Printf("Warning: Failed to start file watcher: %v", err)
				} else {
					log.Println("✓ File Watcher started - monitoring OneDrive folders")
				}
			} else {
				log.Println("✓ File Watcher initialized (awaiting path configuration)")
			}
		}
	} else {
		log.Println("✓ File Watcher disabled in config")
	}
}

// isFinanceReconciliationModel reports whether a model is part of the
// bank-reconciliation / FX / VAT-return suite added to the critical-deployment
// set in Mission G (Wave 4).
func isFinanceReconciliationModel(model any) bool {
	switch model.(type) {
	case *BankAccount, *BankStatement, *BankStatementLine, *BankStatementFile,
		*StatementHash, *BookBankReconciliation, *DepositInTransit, *ChequeRegister,
		*OutstandingCheque, *BankReconciliationAuditLog, *BankCashBalance,
		*BankExpenseEntry, *FXRate, *FXRevaluation, *VATReturn:
		return true
	default:
		return false
	}
}

func shouldSkipCriticalAutoMigrate(db *gorm.DB, model any) bool {
	if db == nil || model == nil {
		return false
	}

	// Finance reconciliation suite (Mission G): provisioning-parity tables —
	// CREATE them on a fresh DB, but never re-AutoMigrate an existing one. A
	// table rebuild on a mature deployment DB that already holds statement/FX
	// rows can trip a FOREIGN KEY check (bank_statements→bank_accounts, etc.),
	// and startup must not fail on pre-existing data. Existence is the guarantee.
	if isFinanceReconciliationModel(model) {
		return db.Migrator().HasTable(model)
	}

	switch model.(type) {
	case *RFQComment:
		namedModel, ok := model.(tableNamer)
		if !ok {
			return false
		}
		tableName := namedModel.TableName()
		if !db.Migrator().HasTable(tableName) {
			return false
		}

		requiredColumns := []string{"id", "rfq_id", "comment", "created_by", "created_at"}
		for _, column := range requiredColumns {
			if !db.Migrator().HasColumn(model, column) {
				return false
			}
		}
		log.Printf("ℹ️ Skipping AutoMigrate for %s: existing SQLite table already matches required columns", tableName)
		return true
	default:
		return false
	}
}

// beforeClose owns the desktop close path. Wails on macOS cancels the native
// window close and sends an internal quit message; bounding cleanup here keeps
// the app from lingering if a background sync or WebKit quit callback stalls.
func (a *App) beforeClose(ctx context.Context) bool {
	go func() {
		done := make(chan struct{})
		go func() {
			a.shutdown(ctx)
			close(done)
		}()

		select {
		case <-done:
			log.Println("AsymmFlow close cleanup finished; exiting process")
		case <-time.After(1500 * time.Millisecond):
			log.Println("WARNING: AsymmFlow close cleanup exceeded 1.5s; forcing process exit")
		}
		os.Exit(0)
	}()
	return true
}

func waitForShutdownStep(label string, timeout time.Duration, wait func()) bool {
	done := make(chan struct{})
	go func() {
		wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		log.Printf("WARNING: Shutdown step timed out after %v: %s", timeout, label)
		return false
	}
}

// shutdown cleans up resources on app exit
func (a *App) shutdown(ctx context.Context) {
	a.shutdownOnce.Do(func() {
		a.shutdownInternal(ctx)
	})
}

func (a *App) shutdownInternal(ctx context.Context) {
	log.Println("AsymmFlow app shutting down...")

	// Stop file watcher
	if a.fileWatcher != nil && a.fileWatcher.IsRunning() {
		if err := a.fileWatcher.Stop(); err != nil {
			log.Printf("Warning: Failed to stop file watcher: %v", err)
		} else {
			log.Println("File Watcher stopped")
		}
	}

	// Close OCR service (saves DNA database)
	if a.ocrService != nil {
		if err := a.ocrService.Close(); err != nil {
			log.Printf("Warning: Failed to close OCR service: %v", err)
		}
	}

	// Stop all background sync goroutines (wait for each to finish before closing connections)
	waitForShutdownStep("background database sync", 500*time.Millisecond, a.StopBackgroundDBSync)
	waitForShutdownStep("collaboration sync loop", 500*time.Millisecond, a.StopCollaborativeSyncLoop)
	if a.dbSyncService != nil {
		waitForShutdownStep("periodic sync service", 500*time.Millisecond, a.StopPeriodicSync)
	}
	if a.dbManager != nil {
		waitForShutdownStep("DBManager periodic sync", 500*time.Millisecond, a.dbManager.StopPeriodicSync)
		waitForShutdownStep("DBManager disconnect", 500*time.Millisecond, a.dbManager.Disconnect)
	}

	// Close database connection BEFORE log file (so close errors can be logged)
	if a.db != nil {
		if sqlDB, err := a.db.DB(); err == nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				log.Printf("WARNING: Database close error: %v", closeErr)
			}
		}
	}

	log.Println("AsymmFlow shut down complete")
	if a.logFile != nil {
		a.logFile.Sync()  // Flush buffers to disk before closing
		a.logFile.Close() // Close after final log message
	}
}

// runCustomMigrations runs custom schema migrations for SQLite compatibility
// SQLite doesn't support IF NOT EXISTS in ALTER TABLE, so we check first
func (a *App) runCustomMigrations() {
	if a.db == nil {
		return
	}

	log.Println("🔧 Running custom schema migrations...")

	// Migration 0: DROP UNIQUE INDEX on invoices.order_id (multiple invoices per order is valid!)
	// This fixes: "UNIQUE constraint failed: invoices.order_id"
	a.dropIndexIfExists("invoices", "idx_invoices_order_id")
	a.dropIndexIfExists("invoices", "idx_unique_invoice_order") // Legacy name variants
	// GORM creates indexes with format: uix_{table}_{column} for uniqueIndex
	a.db.Exec("DROP INDEX IF EXISTS uix_invoices_order_id")
	a.db.Exec("DROP INDEX IF EXISTS idx_invoices_order_id")

	// Migration 1: Add missing columns to invoices (field_visibility + VAT)
	a.addColumnIfNotExists("invoices", "field_visibility", "TEXT DEFAULT '{}'")
	a.addColumnIfNotExists("invoices", "vatbhd", "REAL DEFAULT 0")
	a.addColumnIfNotExists("invoices", "vat_percent", "REAL DEFAULT 0")
	a.addColumnIfNotExists("invoices", "journal_entry_id", "TEXT")

	// Migration 1b: Extended invoice header fields (Issue #19 - client Tally format)
	a.addColumnIfNotExists("invoices", "delivery_note_ref", "TEXT")
	a.addColumnIfNotExists("invoices", "mode_of_payment", "TEXT")
	a.addColumnIfNotExists("invoices", "suppliers_ref", "TEXT")
	a.addColumnIfNotExists("invoices", "other_references", "TEXT")
	a.addColumnIfNotExists("invoices", "buyers_order_number", "TEXT")
	a.addColumnIfNotExists("invoices", "buyers_order_date", "TEXT")
	a.addColumnIfNotExists("invoices", "despatch_document_no", "TEXT")
	a.addColumnIfNotExists("invoices", "delivery_note_date", "TEXT")
	a.addColumnIfNotExists("invoices", "despatched_through", "TEXT DEFAULT 'Direct'")
	a.addColumnIfNotExists("invoices", "destination", "TEXT DEFAULT 'Bahrain'")
	a.addColumnIfNotExists("invoices", "place_of_supply", "TEXT DEFAULT 'Kingdom of Bahrain'")
	a.addColumnIfNotExists("invoices", "terms_of_delivery", "TEXT DEFAULT 'Direct Bank Transfer'")

	// (removed) Former rfq_datas column migration was a permanent no-op: the live table is
	// rfq_data (singular), whose document_hash/visit_locations/product_details/source_doc_path
	// columns are already managed by AutoMigrate(&RFQData{}). See FABLE_WAVE9_SPEC_08. The
	// plural table never existed.
	a.addColumnIfNotExists("opportunities", "product_details", "TEXT")
	a.addColumnIfNotExists("opportunities", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())

	// Migration 3: Migrate offer numbers from OFR-YYYYMMDD-XXXX to XX-25 format
	a.migrateOfferNumbers()

	// Migration 3b: Ensure offers support division-aware storage for PH/AHS flows
	a.migrateOfferDivisionSupport()

	// Migration 4: Company bank accounts table
	if err := a.db.AutoMigrate(&CompanyBankAccount{}); err != nil {
		log.Printf("⚠️ CompanyBankAccount migration: %v", err)
	}
	a.seedCompanyBankAccountsInternal()
	a.ensureCrossModuleSchemaExtensions()

	// Migration 5: Currency exchange rates table
	if err := a.db.AutoMigrate(&CurrencyExchangeRate{}); err != nil {
		log.Printf("⚠️ CurrencyExchangeRate migration: %v", err)
	}
	if err := a.SeedDefaultExchangeRates(); err != nil {
		log.Printf("⚠️ Currency rate seeding: %v", err)
	}

	// Migration 6: Costing sheet revision tracking (Feature D)
	if !a.db.Migrator().HasColumn(&CostingSheetData{}, "revision_number") {
		log.Println("🔄 Migrating CostingSheetData for revision tracking...")
		a.db.Exec("ALTER TABLE costing_sheet_data ADD COLUMN revision_number INTEGER DEFAULT 1")
		a.db.Exec("ALTER TABLE costing_sheet_data ADD COLUMN parent_costing_id INTEGER")
		a.db.Exec("ALTER TABLE costing_sheet_data ADD COLUMN is_active BOOLEAN DEFAULT 1")

		// Set all existing costings as revision 1 and active
		a.db.Exec("UPDATE costing_sheet_data SET revision_number = 1, is_active = 1 WHERE revision_number IS NULL")
		log.Println("✅ Costing sheet revision tracking enabled")
	}

	// Migration 7: Fix invoice CHECK constraint for PartiallyPaid status
	// SQLite cannot ALTER TABLE to modify CHECK constraints, so we update sqlite_master directly.
	// This is safe because we only modify the constraint definition text, not data.
	a.migrateInvoiceCheckConstraint()

	// Migration 7b: Unify the fragmented Opportunity/RFQ stage vocabulary onto
	// the canonical 8-value enum (see stage_vocabulary.go). Idempotent; logs a
	// summary and any unmapped residual values instead of aborting boot.
	if err := a.migrateOpportunityStageVocabulary(); err != nil {
		log.Printf("⚠️ Stage vocabulary migration skipped: %v", err)
	}

	// Migration 8: Butler chat action metadata for approvals/workflow execution
	a.addColumnIfNotExists("chat_messages", "message_type", "TEXT DEFAULT 'chat'")
	a.addColumnIfNotExists("chat_messages", "action_type", "TEXT")
	a.addColumnIfNotExists("chat_messages", "action_target", "TEXT")
	a.addColumnIfNotExists("chat_messages", "action_label", "TEXT")
	a.addColumnIfNotExists("chat_messages", "action_data", "TEXT")
	a.addColumnIfNotExists("chat_messages", "action_metadata", "TEXT")
	a.addColumnIfNotExists("chat_messages", "action_status", "TEXT DEFAULT 'none'")

	// Migration 9: Journal entry source tracking for expense and payroll auto-posting
	a.addColumnIfNotExists("journal_entries", "source_type", "TEXT")
	a.addColumnIfNotExists("journal_entries", "source_id", "TEXT")
	a.addColumnIfNotExists("journal_entries", "is_auto_generated", "BOOLEAN DEFAULT 0")
	a.addColumnIfNotExists("journal_entries", "reversed_by_id", "TEXT")
	a.addColumnIfNotExists("journal_entries", "reverses_id", "TEXT")
	a.addColumnIfNotExists("journal_entries", "updated_by", "TEXT")
	a.addColumnIfNotExists("journal_lines", "updated_by", "TEXT")

	// Migration 10: Repair malformed 2026 OneDrive opportunity numbering/year metadata
	if err := repairImportedOpportunityMetadata(a.db); err != nil {
		log.Printf("⚠️ Opportunity metadata repair skipped: %v", err)
	}

	if err := backfillOpportunityProductDetailsFromOffers(a.db); err != nil {
		log.Printf("⚠️ Opportunity product-details backfill skipped: %v", err)
	}

	if err := repairImportedCommercialDocuments(a.db); err != nil {
		log.Printf("⚠️ Commercial document repair skipped: %v", err)
	}

	log.Println("✅ Custom schema migrations complete (invoices + bank accounts + currency rates + costing revisions)")
}

// dropIndexIfExists drops an index if it exists (SQLite compatible)
// SECURITY: Validates index name to prevent SQL injection in DDL
func (a *App) dropIndexIfExists(table, indexName string) {
	// SECURITY FIX: Validate index name to prevent SQL injection
	if !isValidSQLIdentifier(indexName) {
		log.Printf("⚠️ Invalid index name rejected: %s", indexName)
		return
	}

	// Try to drop - SQLite DROP INDEX IF EXISTS is safe
	if err := a.db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)).Error; err != nil {
		log.Printf("⚠️ Index drop note for %s: %v", indexName, err)
	}
}

// addColumnIfNotExists adds a column to a table if it doesn't already exist (SQLite compatible)
// SECURITY NOTE: Only call with hardcoded table/column names - values are validated but not parameterized
func (a *App) addColumnIfNotExists(table, column, columnType string) {
	// SECURITY: Validate identifiers to prevent SQL injection
	// Only allow alphanumeric and underscore characters
	validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !validIdentifier.MatchString(table) || !validIdentifier.MatchString(column) {
		log.Printf("⚠️ Invalid identifier rejected: table=%s, column=%s", table, column)
		return
	}

	if a.db == nil || !a.db.Migrator().HasTable(table) {
		return
	}

	// Check if column exists using PRAGMA
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name='%s'", table, column)
	if err := a.db.Raw(query).Scan(&count).Error; err != nil {
		log.Printf("⚠️ Failed to check column %s.%s: %v", table, column, err)
		return
	}

	if count == 0 {
		// Column doesn't exist, add it
		alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnType)
		if err := a.db.Exec(alterSQL).Error; err != nil {
			log.Printf("⚠️ Failed to add column %s.%s: %v", table, column, err)
		} else {
			log.Printf("✅ Added column %s.%s", table, column)
		}
	}
}

func (a *App) ensureCrossModuleSchemaExtensions() {
	if a == nil || a.db == nil {
		return
	}

	// Mature deployment databases skip runCustomMigrations() during startup,
	// so keep the offer division repair in the always-run foundation path too.
	a.migrateOfferDivisionSupport()
	a.addColumnIfNotExists("offers", "terms_and_conditions", "TEXT")
	a.addColumnIfNotExists("offers", "subject", "TEXT")
	a.addColumnIfNotExists("offers", "body", "TEXT")
	// I-25: datasheet scope binding on mature deployment DBs that skip full migration.
	a.addColumnIfNotExists("offers", "attachment_scope_id", "TEXT")
	if err := repairImportedCommercialDocuments(a.db); err != nil {
		log.Printf("⚠️ Commercial document repair skipped: %v", err)
	}

	// Keep division-aware finance/schema extensions consistent across startup,
	// deployment audits, and test harnesses so note fields and company routing
	// remain backed by real SQLite columns.
	for _, rfqTable := range []string{"rfq_data", "rfq_datas"} {
		if a.db.Migrator().HasTable(rfqTable) {
			a.addColumnIfNotExists(rfqTable, "rfq_ref", "TEXT")
		}
	}
	a.addColumnIfNotExists("opportunities", "product_details", "TEXT")
	a.addColumnIfNotExists("opportunities", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	if a.hasColumn("opportunities", "division") {
		if err := a.db.Exec("UPDATE opportunities SET division = " + divisionDefaultSQLLiteral() + " WHERE division IS NULL OR TRIM(division) = ''").Error; err != nil {
			log.Printf("⚠️ Opportunity division backfill skipped: %v", err)
		}
	}
	a.addColumnIfNotExists("purchase_orders", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("purchase_order_items", "supplier_part_number", "TEXT")
	// B3: dedicated GRN completion marker. Fresh DBs get this column via
	// AutoMigrate(&GoodsReceivedNote{}) in runCustomMigrations; mature
	// deployment DBs skip that path entirely, so it must also be added here.
	a.addColumnIfNotExists("goods_received_notes", "completed_at", "DATETIME")
	if a.hasColumn("goods_received_notes", "completed_at") {
		// Backfill: derive completed_at for GRNs that already posted a "GRN
		// Receipt" stock movement before this column existed. Idempotent —
		// only touches rows where completed_at IS NULL. Historical
		// all-rejected/no-item GRNs that were completed pre-B3 posted no
		// movement, so they stay NULL here (indistinguishable from
		// never-completed); that's acceptable — re-running CompleteGRN on
		// them is harmless since they post nothing to double-count.
		backfillSQL := `UPDATE goods_received_notes
			SET completed_at = (
				SELECT MIN(created_at) FROM stock_movements
				WHERE reference_type = 'goods_received_note' AND reference_id = goods_received_notes.id
			)
			WHERE completed_at IS NULL
			  AND EXISTS (
				SELECT 1 FROM stock_movements
				WHERE reference_type = 'goods_received_note' AND reference_id = goods_received_notes.id
			  )`
		if err := a.db.Exec(backfillSQL).Error; err != nil {
			log.Printf("⚠️ GRN completed_at backfill skipped: %v", err)
		}
	}
	a.addColumnIfNotExists("supplier_invoices", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("supplier_invoices", "supplier_name", "TEXT")
	a.addColumnIfNotExists("supplier_invoices", "po_number", "TEXT")
	a.addColumnIfNotExists("supplier_invoices", "ocr_document_id", "TEXT")
	a.addColumnIfNotExists("supplier_invoices", "ocr_confidence", "REAL DEFAULT 0")
	a.addColumnIfNotExists("payments", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("supplier_payments", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("credit_notes", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("bank_statements", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("bank_statements", "notes", "TEXT")
	if err := a.db.AutoMigrate(&BankLinePaymentAllocation{}); err != nil {
		log.Printf("⚠️ BankLinePaymentAllocation migration: %v", err)
	}
	a.addColumnIfNotExists("expense_entries", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("recurring_expenses", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("bank_expense_entries", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("employee_compensation_profiles", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("payroll_periods", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("payroll_runs", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	a.addColumnIfNotExists("payroll_payouts", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())

	a.backfillBankStatementMetadata()
	a.backfillDivisionAwareFinanceData()
}

func (a *App) hasColumn(table, column string) bool {
	if a.db == nil {
		return false
	}

	validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !validIdentifier.MatchString(table) || !validIdentifier.MatchString(column) {
		return false
	}

	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name='%s'", table, column)
	if err := a.db.Raw(query).Scan(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// normalizeDivisionSQL returns a SQL CASE expression that normalises columnExpr
// to a canonical division Key. It delegates to the active overlay so the
// division set + aliases are CONFIGURATION, not 15 hardcoded inline IN-lists.
// (Matching now mirrors the overlay's exact Key/alias normalisation rather than
// the old looser LIKE '%beacon%' substring test.)
func normalizeDivisionSQL(columnExpr string) string {
	return overlay.Active().DivisionNormalizationCase(columnExpr)
}

// divisionDefaultSQLLiteral returns the configured default division as a
// single-quoted, escaped SQL string literal (e.g. 'Acme Instrumentation') for
// use in migration DDL column defaults and blank-division backfills. The value
// comes from the active overlay (set at startup, before migrations run), so a
// different vertical's default division is a config edit, not a code edit.
func divisionDefaultSQLLiteral() string {
	d := normalizeDivisionName("") // active overlay's default division key
	return "'" + strings.ReplaceAll(d, "'", "''") + "'"
}

func (a *App) migrateOfferDivisionSupport() {
	if a.db == nil {
		return
	}

	a.addColumnIfNotExists("offers", "division", "TEXT DEFAULT "+divisionDefaultSQLLiteral())
	if !a.hasColumn("offers", "division") {
		return
	}

	if err := a.db.Exec("UPDATE offers SET division = " + divisionDefaultSQLLiteral() + " WHERE division IS NULL OR TRIM(division) = ''").Error; err != nil {
		log.Printf("⚠️ Failed to default offers.division: %v", err)
	}

	if a.hasColumn("orders", "offer_id") && a.hasColumn("orders", "division") {
		sql := fmt.Sprintf(`
			UPDATE offers
			SET division = (
				SELECT %s
				FROM orders
				WHERE orders.offer_id = offers.id
					AND orders.deleted_at IS NULL
					AND TRIM(COALESCE(orders.division, '')) <> ''
				ORDER BY orders.updated_at DESC, orders.created_at DESC
				LIMIT 1
			)
			WHERE EXISTS (
				SELECT 1 FROM orders
				WHERE orders.offer_id = offers.id
					AND orders.deleted_at IS NULL
					AND TRIM(COALESCE(orders.division, '')) <> ''
			)
		`, normalizeDivisionSQL("orders.division"))
		if err := a.db.Exec(sql).Error; err != nil {
			log.Printf("⚠️ Failed to backfill offers.division from orders: %v", err)
		}
	}

	if a.hasColumn("invoices", "offer_id") && a.hasColumn("invoices", "division") {
		sql := fmt.Sprintf(`
			UPDATE offers
			SET division = (
				SELECT %s
				FROM invoices
				WHERE invoices.offer_id = offers.id
					AND invoices.deleted_at IS NULL
					AND TRIM(COALESCE(invoices.division, '')) <> ''
				ORDER BY invoices.updated_at DESC, invoices.created_at DESC
				LIMIT 1
			)
			WHERE EXISTS (
				SELECT 1 FROM invoices
				WHERE invoices.offer_id = offers.id
					AND invoices.deleted_at IS NULL
					AND TRIM(COALESCE(invoices.division, '')) <> ''
			)
		`, normalizeDivisionSQL("invoices.division"))
		if err := a.db.Exec(sql).Error; err != nil {
			log.Printf("⚠️ Failed to backfill offers.division from invoices: %v", err)
		}
	}

	if err := a.db.Exec("UPDATE offers SET division = " + divisionDefaultSQLLiteral() + " WHERE division IS NULL OR TRIM(division) = ''").Error; err != nil {
		log.Printf("⚠️ Failed to finalize offers.division backfill: %v", err)
	}
}

func (a *App) backfillBankStatementMetadata() {
	if a.db == nil {
		return
	}

	var accounts []CompanyBankAccount
	if err := a.db.Where("is_active = ?", true).Find(&accounts).Error; err != nil {
		log.Printf("⚠️ Failed to load bank accounts for statement metadata backfill: %v", err)
		return
	}

	accountByID := make(map[string]CompanyBankAccount, len(accounts))
	for _, account := range accounts {
		accountByID[account.ID] = account
	}

	var statements []BankStatement
	if err := a.db.Find(&statements).Error; err != nil {
		log.Printf("⚠️ Failed to load bank statements for metadata backfill: %v", err)
		return
	}

	for _, stmt := range statements {
		updates := map[string]any{}

		if !stmt.PeriodEnd.IsZero() && (stmt.PeriodStart.IsZero() || stmt.PeriodStart.After(stmt.PeriodEnd)) {
			correctedStart := time.Date(stmt.PeriodEnd.Year(), stmt.PeriodEnd.Month(), 1, 0, 0, 0, 0, stmt.PeriodEnd.Location())
			updates["period_start"] = correctedStart
			stmt.PeriodStart = correctedStart
		}

		account, hasAccount := accountByID[stmt.BankAccountID]
		if hasAccount {
			if strings.TrimSpace(stmt.Division) == "" {
				updates["division"] = normalizeDivisionName(account.Division)
			}
			if strings.TrimSpace(stmt.StatementNumber) == "" || strings.HasPrefix(strings.TrimSpace(stmt.StatementNumber), "OCR-") {
				if rebuiltNumber := buildBankStatementNumber(account, stmt.PeriodStart, stmt.PeriodEnd); rebuiltNumber != "" {
					updates["statement_number"] = rebuiltNumber
				}
			}

			if strings.TrimSpace(stmt.Notes) == "" {
				notes := buildBankStatementSummaryNote(account, stmt.PeriodStart, stmt.PeriodEnd, stmt.OpeningBalance, stmt.ClosingBalance, stmt.TotalDebits, stmt.TotalCredits, stmt.Currency)
				if strings.TrimSpace(notes) != "" {
					updates["notes"] = notes
				}
			}
		}

		if len(updates) == 0 {
			continue
		}

		if err := a.db.Model(&BankStatement{}).Where("id = ?", stmt.ID).Updates(updates).Error; err != nil {
			log.Printf("⚠️ Failed to backfill bank statement metadata for %s: %v", stmt.ID, err)
		}
	}
}

func (a *App) backfillDivisionAwareFinanceData() {
	if a.db == nil {
		return
	}

	queries := []struct {
		name            string
		table           string
		requiredColumns map[string][]string
		sql             string
	}{
		{
			name:  "opportunities from offers",
			table: "opportunities",
			requiredColumns: map[string][]string{
				"opportunities": {"division", "offer_id"},
				"offers":        {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE opportunities
				SET division = COALESCE((
					SELECT %s
					FROM offers o
					WHERE o.id = opportunities.offer_id
				), %s)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("o.division"), divisionDefaultSQLLiteral()),
		},
		{
			name:  "company bank accounts",
			table: "company_bank_accounts",
			requiredColumns: map[string][]string{
				"company_bank_accounts": {"division"},
			},
			sql: "UPDATE company_bank_accounts SET division = " + divisionDefaultSQLLiteral() + " WHERE division IS NULL OR TRIM(division) = ''",
		},
		{
			name:  "bank statements from bank accounts",
			table: "bank_statements",
			requiredColumns: map[string][]string{
				"bank_statements":       {"division", "bank_account_id"},
				"company_bank_accounts": {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE bank_statements
				SET division = COALESCE((
					SELECT %s
					FROM company_bank_accounts cba
					WHERE cba.id = bank_statements.bank_account_id
				), %s)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("cba.division"), divisionDefaultSQLLiteral()),
		},
		{
			name:  "purchase orders from orders",
			table: "purchase_orders",
			requiredColumns: map[string][]string{
				"purchase_orders": {"division", "order_id"},
				"orders":          {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE purchase_orders
				SET division = COALESCE((
					SELECT %s
					FROM orders o
					WHERE o.id = purchase_orders.order_id
				), %s)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("o.division"), divisionDefaultSQLLiteral()),
		},
		{
			name:  "supplier invoices from orders/POs",
			table: "supplier_invoices",
			requiredColumns: map[string][]string{
				"supplier_invoices": {"division", "order_id", "purchase_order_id"},
				"orders":            {"id", "division"},
				"purchase_orders":   {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE supplier_invoices
				SET division = COALESCE(
					(
						SELECT %s
						FROM orders o
						WHERE o.id = supplier_invoices.order_id
					),
					(
						SELECT %s
						FROM purchase_orders po
						WHERE po.id = supplier_invoices.purchase_order_id
					),
					%s
				)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("o.division"), normalizeDivisionSQL("po.division"), divisionDefaultSQLLiteral()),
		},
		{
			name:  "credit notes from invoices",
			table: "credit_notes",
			requiredColumns: map[string][]string{
				"credit_notes": {"division", "invoice_id"},
				"invoices":     {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE credit_notes
				SET division = COALESCE((
					SELECT %s
					FROM invoices i
					WHERE i.id = credit_notes.invoice_id
				), %s)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("i.division"), divisionDefaultSQLLiteral()),
		},
		{
			name:  "customer payments from invoices",
			table: "payments",
			requiredColumns: map[string][]string{
				"payments": {"division", "invoice_id"},
				"invoices": {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE payments
				SET division = COALESCE((
					SELECT %s
					FROM invoices i
					WHERE i.id = payments.invoice_id
				), %s)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("i.division"), divisionDefaultSQLLiteral()),
		},
		{
			name:  "supplier payments from supplier invoices",
			table: "supplier_payments",
			requiredColumns: map[string][]string{
				"supplier_payments": {"division", "supplier_invoice_id"},
				"supplier_invoices": {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE supplier_payments
				SET division = COALESCE((
					SELECT %s
					FROM supplier_invoices si
					WHERE si.id = supplier_payments.supplier_invoice_id
				), %s)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("si.division"), divisionDefaultSQLLiteral()),
		},
		{
			name:  "expense entries from linked bank accounts and orders",
			table: "expense_entries",
			requiredColumns: map[string][]string{
				"expense_entries":       {"division", "bank_account_id", "order_id"},
				"company_bank_accounts": {"id", "division"},
				"orders":                {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE expense_entries
				SET division = COALESCE(
					(
						SELECT %s
						FROM company_bank_accounts cba
						WHERE cba.id = expense_entries.bank_account_id
					),
					(
						SELECT %s
						FROM orders o
						WHERE o.id = expense_entries.order_id
					),
					%s
				)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("cba.division"), normalizeDivisionSQL("o.division"), divisionDefaultSQLLiteral()),
		},
		{
			name:  "recurring expenses default division",
			table: "recurring_expenses",
			requiredColumns: map[string][]string{
				"recurring_expenses": {"division"},
			},
			sql: "UPDATE recurring_expenses SET division = " + divisionDefaultSQLLiteral() + " WHERE division IS NULL OR TRIM(division) = ''",
		},
		{
			name:  "bank expense entries from bank statements",
			table: "bank_expense_entries",
			requiredColumns: map[string][]string{
				"bank_expense_entries": {"division", "bank_statement_line_id"},
				"bank_statement_lines": {"id", "bank_statement_id"},
				"bank_statements":      {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE bank_expense_entries
				SET division = COALESCE((
					SELECT %s
					FROM bank_statement_lines bsl
					JOIN bank_statements bs ON bs.id = bsl.bank_statement_id
					WHERE bsl.id = bank_expense_entries.bank_statement_line_id
				), %s)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("bs.division"), divisionDefaultSQLLiteral()),
		},
		{
			name:  "employee compensation profiles default division",
			table: "employee_compensation_profiles",
			requiredColumns: map[string][]string{
				"employee_compensation_profiles": {"division"},
			},
			sql: "UPDATE employee_compensation_profiles SET division = " + divisionDefaultSQLLiteral() + " WHERE division IS NULL OR TRIM(division) = ''",
		},
		{
			name:  "payroll periods default division",
			table: "payroll_periods",
			requiredColumns: map[string][]string{
				"payroll_periods": {"division"},
			},
			sql: "UPDATE payroll_periods SET division = " + divisionDefaultSQLLiteral() + " WHERE division IS NULL OR TRIM(division) = ''",
		},
		{
			name:  "payroll runs from payroll periods",
			table: "payroll_runs",
			requiredColumns: map[string][]string{
				"payroll_runs":    {"division", "payroll_period_id"},
				"payroll_periods": {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE payroll_runs
				SET division = COALESCE((
					SELECT %s
					FROM payroll_periods pp
					WHERE pp.id = payroll_runs.payroll_period_id
				), %s)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("pp.division"), divisionDefaultSQLLiteral()),
		},
		{
			name:  "payroll payouts from payroll runs",
			table: "payroll_payouts",
			requiredColumns: map[string][]string{
				"payroll_payouts": {"division", "payroll_run_id"},
				"payroll_runs":    {"id", "division"},
			},
			sql: fmt.Sprintf(`
				UPDATE payroll_payouts
				SET division = COALESCE((
					SELECT %s
					FROM payroll_runs pr
					WHERE pr.id = payroll_payouts.payroll_run_id
				), %s)
				WHERE division IS NULL OR TRIM(division) = ''
			`, normalizeDivisionSQL("pr.division"), divisionDefaultSQLLiteral()),
		},
	}

	for _, query := range queries {
		if !a.db.Migrator().HasTable(query.table) {
			continue
		}
		skipQuery := false
		for tableName, columns := range query.requiredColumns {
			if !a.db.Migrator().HasTable(tableName) {
				skipQuery = true
				break
			}
			for _, column := range columns {
				if !a.db.Migrator().HasColumn(tableName, column) {
					skipQuery = true
					break
				}
			}
			if skipQuery {
				break
			}
		}
		if skipQuery {
			continue
		}
		if err := a.db.Exec(query.sql).Error; err != nil {
			log.Printf("⚠️ Failed division backfill for %s: %v", query.name, err)
		}
	}
}

// migrateInvoiceCheckConstraint updates the invoice CHECK constraint in existing production databases.
// SQLite doesn't support ALTER TABLE for CHECK constraints, so we modify sqlite_master directly
// using PRAGMA writable_schema and bump schema_version so the current connection reloads it.
func (a *App) migrateInvoiceCheckConstraint() {
	if a.db == nil {
		return
	}

	// Get the current CREATE TABLE SQL for invoices
	var createSQL string
	if err := a.db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='invoices'").Scan(&createSQL).Error; err != nil {
		log.Printf("⚠️ Cannot check invoice table schema: %v", err)
		return
	}

	if createSQL == "" {
		return // Table doesn't exist yet - will be created with correct constraint
	}

	desiredStatuses := []string{"Draft", "Sent", "Paid", "PartiallyPaid", "Overdue", "Cancelled", "Void", "Proforma"}
	hasAllStatuses := true
	for _, status := range desiredStatuses {
		if !strings.Contains(createSQL, "'"+status+"'") {
			hasAllStatuses = false
			break
		}
	}
	if hasAllStatuses {
		return
	}

	// Check if there's a CHECK constraint at all
	if !strings.Contains(createSQL, "CHECK") || !strings.Contains(createSQL, "'Paid'") {
		return // No relevant CHECK constraint to update
	}

	log.Printf("🔧 Migrating invoice CHECK constraint to include statuses: %s", strings.Join(desiredStatuses, ", "))

	statusConstraintPattern := regexp.MustCompile(`(?i)CHECK\s*\(\s*status\s+IN\s*\([^)]+\)\s*\)`)
	desiredConstraint := "CHECK (status IN ('Draft','Sent','Paid','PartiallyPaid','Overdue','Cancelled','Void','Proforma'))"
	newSQL := statusConstraintPattern.ReplaceAllString(createSQL, desiredConstraint)
	if newSQL == createSQL {
		log.Printf("⚠️ Could not locate CHECK constraint pattern in invoices table - skipping")
		return
	}

	// Apply the schema change
	if err := a.db.Exec("PRAGMA writable_schema = ON").Error; err != nil {
		log.Printf("⚠️ PRAGMA writable_schema failed: %v", err)
		return
	}

	if err := a.db.Exec("UPDATE sqlite_master SET sql = ? WHERE type='table' AND name='invoices'", newSQL).Error; err != nil {
		log.Printf("⚠️ Failed to update invoice CHECK constraint: %v", err)
		a.db.Exec("PRAGMA writable_schema = OFF")
		return
	}

	var schemaVersion int
	_ = a.db.Raw("PRAGMA schema_version").Scan(&schemaVersion).Error
	if schemaVersion >= 0 {
		_ = a.db.Exec(fmt.Sprintf("PRAGMA schema_version = %d", schemaVersion+1)).Error
	}
	a.db.Exec("PRAGMA writable_schema = OFF")

	// Verify integrity
	var integrityResult string
	a.db.Raw("PRAGMA integrity_check").Scan(&integrityResult)
	if integrityResult == "ok" {
		log.Printf("✅ Invoice CHECK constraint updated for invoice/proforma statuses (integrity: ok)")
	} else {
		log.Printf("⚠️ Invoice CHECK constraint updated but integrity check: %s", integrityResult)
	}
}

// migrateOfferNumbers converts offer numbers from OFR-YYYYMMDD-XXXX and QUO-YYYY-XX to XX-YY format
func (a *App) migrateOfferNumbers() {
	var offers []Offer
	if err := a.db.Find(&offers).Error; err != nil {
		log.Printf("⚠️ Failed to fetch offers for migration: %v", err)
		return
	}

	migrated := 0
	for _, offer := range offers {
		oldNum := offer.OfferNumber
		newNum := ""

		// Skip if already in XX-YY format
		if matched, _ := regexp.MatchString(`^\d+-\d{2}$`, oldNum); matched {
			continue
		}

		if strings.HasPrefix(oldNum, "QUO-") {
			// QUO-2025-50 → 50-25
			parts := strings.Split(oldNum, "-")
			if len(parts) == 3 && len(parts[1]) >= 4 {
				num := parts[2]       // "50"
				year := parts[1][2:4] // "25" from "2025"
				newNum = fmt.Sprintf("%s-%s", num, year)
			}
		} else if strings.HasPrefix(oldNum, "OFR-") {
			// OFR-20250131-0001 → need to assign new sequential number
			// For date-based format, we'll assign based on creation order
			year := time.Now().Year() % 100
			// Get next sequential number for this year
			var maxNum int
			yearSuffix := fmt.Sprintf("-%02d", year)
			a.db.Model(&Offer{}).
				Where("offer_number LIKE ?", "%"+yearSuffix).
				Select("COALESCE(MAX(CAST(SUBSTR(offer_number, 1, INSTR(offer_number, '-')-1) AS INTEGER)), 0)").
				Scan(&maxNum)
			newNum = fmt.Sprintf("%d-%02d", maxNum+1, year)
		}

		if newNum != "" && newNum != oldNum {
			if err := a.db.Model(&offer).Update("offer_number", newNum).Error; err != nil {
				log.Printf("⚠️ Failed to migrate offer %s → %s: %v", oldNum, newNum, err)
			} else {
				log.Printf("✅ Migrated offer number: %s → %s", oldNum, newNum)
				migrated++
			}
		}
	}

	if migrated > 0 {
		log.Printf("✅ Migrated %d offer numbers to XX-YY format", migrated)
	}
}

// PredictPayment performs payment prediction for a single customer

// migrateDatabaseFileForContract runs the trading-model migrations against a
// standalone database file. The Mission DP1 update contract invokes it on a
// COPY of the live data plane during a schema upgrade; the copy is atomically
// swapped in only if this returns nil, so a failed migration never touches the
// live database. It opens its own short-lived connection and closes it before
// returning so the caller can rename the file (Windows cannot rename an open
// file). Per-model skips (SQLite cannot alter constraints on existing tables)
// are tolerated exactly as the startup migrator tolerates them — only a failure
// to open the copy is fatal.
func migrateDatabaseFileForContract(path string) error {
	root := composition.NewRoot()
	dsn := composition.SQLiteDSN(filepath.Clean(path), composition.DefaultPragmas...)
	db, err := root.OpenSQLite(dsn, &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true, // same as startup()
	})
	if err != nil {
		return fmt.Errorf("open database copy for migration: %w", err)
	}
	defer func() {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			// Flush the WAL back into the main file BEFORE closing so the atomic
			// swap (which deletes the -wal/-shm siblings) can never drop committed
			// migration data. TRUNCATE is the strongest checkpoint; a clean close
			// would checkpoint anyway, but this makes the guarantee explicit on a
			// data-loss-critical path.
			_, _ = sqlDB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
			_ = sqlDB.Close()
		}
	}()

	migrated, skipped := root.MigrateModels(tradingModels(), func(index, total int, model string, mErr error) {
		if mErr != nil {
			log.Printf("⚠️ contract migration skipped %s (%d/%d): %v", model, index, total, mErr)
		}
	})
	log.Printf("🔧 contract migration on copy: %d migrated, %d skipped (%s)", migrated, skipped, path)
	return nil
}
