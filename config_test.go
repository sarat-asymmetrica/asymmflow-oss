// ═══════════════════════════════════════════════════════════════════════════
// CONFIG TESTS - Verify configuration loading and validation
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigWithoutEnvFile(t *testing.T) {
	// Make sure .env doesn't exist for this test
	_ = os.Remove(".env")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Check defaults - Database.Path should contain "ph_holdings.db" (location varies by OS)
	if cfg.Database.Path == "" {
		t.Errorf("Expected non-empty database path, got empty string")
	}
	if !strings.Contains(cfg.Database.Path, "ph_holdings.db") {
		t.Errorf("Expected database path to contain 'ph_holdings.db', got %s", cfg.Database.Path)
	}

	if cfg.App.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got %s", cfg.App.LogLevel)
	}

	if cfg.App.WatcherDebounceMS != 300 {
		t.Errorf("Expected default debounce 300ms, got %d", cfg.App.WatcherDebounceMS)
	}

	if cfg.App.WatcherQueueSize != 1000 {
		t.Errorf("Expected default queue size 1000, got %d", cfg.App.WatcherQueueSize)
	}

	// Azure should be disabled by default
	if cfg.Azure.Enabled {
		t.Error("Azure should be disabled when credentials not configured")
	}

	t.Logf("✓ Config loaded with defaults successfully")
}

func TestLoadConfigWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("DATABASE_PATH", "./test_custom.db")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("DEBUG_MODE", "true")
	os.Setenv("WATCHER_DEBOUNCE_MS", "500")
	os.Setenv("ENABLE_FILE_WATCHER", "false")
	defer func() {
		os.Unsetenv("DATABASE_PATH")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("DEBUG_MODE")
		os.Unsetenv("WATCHER_DEBOUNCE_MS")
		os.Unsetenv("ENABLE_FILE_WATCHER")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Packaged database now takes precedence over ad-hoc env overrides when present.
	expectedDBPath := resolveConfiguredPath("./test_custom.db")
	if packaged := packagedDatabasePath(); packaged != "" {
		expectedDBPath = packaged
	}
	if cfg.Database.Path != expectedDBPath {
		t.Errorf("Expected database path '%s', got %s", expectedDBPath, cfg.Database.Path)
	}

	if cfg.App.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got %s", cfg.App.LogLevel)
	}

	if !cfg.App.DebugMode {
		t.Error("Expected debug mode to be true")
	}

	if cfg.App.WatcherDebounceMS != 500 {
		t.Errorf("Expected debounce 500ms, got %d", cfg.App.WatcherDebounceMS)
	}

	if cfg.App.EnableFileWatcher {
		t.Error("Expected file watcher to be disabled")
	}

	t.Logf("✓ Config loaded with environment variables successfully")
}

func TestLoadEnvFilesWithPrecedenceMergesMissingKeys(t *testing.T) {
	dir := t.TempDir()
	lowPriority := filepath.Join(dir, "appdata.env")
	highPriority := filepath.Join(dir, "bundle.env")

	if err := os.WriteFile(lowPriority, []byte("ASYMMFLOW_TEST_AIML_KEY=from-appdata\nASYMMFLOW_TEST_SYNC=false\nASYMMFLOW_TEST_OS_LOCK=from-appdata\n"), 0600); err != nil {
		t.Fatalf("failed to write low priority env: %v", err)
	}
	if err := os.WriteFile(highPriority, []byte("ASYMMFLOW_TEST_SYNC=true\nASYMMFLOW_TEST_LOG=debug\nASYMMFLOW_TEST_OS_LOCK=from-bundle\n"), 0600); err != nil {
		t.Fatalf("failed to write high priority env: %v", err)
	}

	keys := []string{
		"ASYMMFLOW_TEST_AIML_KEY",
		"ASYMMFLOW_TEST_SYNC",
		"ASYMMFLOW_TEST_LOG",
		"ASYMMFLOW_TEST_OS_LOCK",
	}
	original := make(map[string]string, len(keys))
	present := make(map[string]bool, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			original[key] = value
			present[key] = true
		}
		_ = os.Unsetenv(key)
	}
	defer func() {
		for _, key := range keys {
			if present[key] {
				_ = os.Setenv(key, original[key])
			} else {
				_ = os.Unsetenv(key)
			}
		}
	}()

	if err := os.Setenv("ASYMMFLOW_TEST_OS_LOCK", "from-process"); err != nil {
		t.Fatalf("failed to set process env: %v", err)
	}

	loaded := loadEnvFilesWithPrecedence([]string{highPriority, lowPriority})
	if len(loaded) != 2 || loaded[0] != highPriority || loaded[1] != lowPriority {
		t.Fatalf("expected loaded files in priority order, got %#v", loaded)
	}
	if got := os.Getenv("ASYMMFLOW_TEST_AIML_KEY"); got != "from-appdata" {
		t.Fatalf("expected missing AIML key to be supplied by lower-priority env, got %q", got)
	}
	if got := os.Getenv("ASYMMFLOW_TEST_SYNC"); got != "true" {
		t.Fatalf("expected high-priority env to override lower-priority value, got %q", got)
	}
	if got := os.Getenv("ASYMMFLOW_TEST_OS_LOCK"); got != "from-process" {
		t.Fatalf("expected real process env to be preserved, got %q", got)
	}
}

func TestGetDatabasePathResolvesRelativeEnvPath(t *testing.T) {
	dataDir := filepath.Join(".", "data")
	dbPath := filepath.Join(dataDir, "ph_holdings.db")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create test data dir: %v", err)
	}
	if err := os.WriteFile(dbPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	defer os.Remove(dbPath)
	defer os.Remove(dataDir)

	originalPHDB := os.Getenv("PH_DB_PATH")
	originalDB := os.Getenv("DATABASE_PATH")
	os.Unsetenv("PH_DB_PATH")
	os.Setenv("DATABASE_PATH", "./data/ph_holdings.db")
	defer func() {
		if originalPHDB == "" {
			os.Unsetenv("PH_DB_PATH")
		} else {
			os.Setenv("PH_DB_PATH", originalPHDB)
		}
		if originalDB == "" {
			os.Unsetenv("DATABASE_PATH")
		} else {
			os.Setenv("DATABASE_PATH", originalDB)
		}
	}()

	resolved := getDatabasePath()
	if packaged := packagedDatabasePath(); packaged != "" {
		if resolved != packaged {
			t.Fatalf("expected packaged database path %s to win, got %s", packaged, resolved)
		}
		return
	}
	if !strings.HasSuffix(filepath.ToSlash(resolved), "data/ph_holdings.db") {
		t.Fatalf("expected resolved path to end with data/ph_holdings.db, got %s", resolved)
	}
	if _, err := os.Stat(resolved); err != nil {
		t.Fatalf("expected resolved database path to exist: %v", err)
	}
}

func TestSeedAppDataDatabaseReplacesHollowExistingDB(t *testing.T) {
	dir := t.TempDir()
	appDataPath := filepath.Join(dir, "appdata", "ph_holdings.db")
	packagedPath := filepath.Join(dir, "package", "data", "ph_holdings.db")

	createDeploymentProfileDB(t, appDataPath, 0, 0, 0, 0, 0)
	db, err := sql.Open("sqlite3", appDataPath)
	if err != nil {
		t.Fatalf("failed to open hollow appdata db: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE license_keys (
			key TEXT PRIMARY KEY,
			role TEXT,
			display_name TEXT,
			device_hash TEXT,
			activated INTEGER,
			activated_at TEXT,
			notes TEXT,
			created_by TEXT
		);
		INSERT INTO license_keys (key, role, display_name, device_hash, activated, activated_at)
		VALUES ('PH-SLS-B4AA10', 'sales', 'Sales Test', 'device-123', 1, '2026-04-20 10:00:00');
	`); err != nil {
		db.Close()
		t.Fatalf("failed to seed hollow appdata license: %v", err)
	}
	db.Close()

	createDeploymentProfileDB(t, packagedPath, 453, 35, 196, 470, 586)
	db, err = sql.Open("sqlite3", packagedPath)
	if err != nil {
		t.Fatalf("failed to open packaged db: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE license_keys (
			key TEXT PRIMARY KEY,
			role TEXT,
			display_name TEXT,
			device_hash TEXT,
			activated INTEGER,
			activated_at TEXT,
			notes TEXT,
			created_by TEXT
		);
		INSERT INTO license_keys (key, role, display_name, device_hash, activated, activated_at)
		VALUES ('PH-SLS-B4AA10', 'sales', 'Sales Test', '', 0, NULL);
	`); err != nil {
		db.Close()
		t.Fatalf("failed to seed packaged license: %v", err)
	}
	db.Close()

	if !seedAppDataDatabaseFromPackaged(appDataPath, packagedPath) {
		t.Fatalf("expected hollow appdata db to be replaced from packaged seed")
	}

	profile := readDeploymentDatabaseProfile(appDataPath)
	if profile.Customers != 453 || profile.Orders != 196 || profile.Invoices != 470 {
		t.Fatalf("expected packaged business data after reseed, got %+v", profile)
	}

	db, err = sql.Open("sqlite3", appDataPath)
	if err != nil {
		t.Fatalf("failed to reopen reseeded db: %v", err)
	}
	defer db.Close()
	var activated int
	var deviceHash string
	if err := db.QueryRow(`SELECT activated, device_hash FROM license_keys WHERE key = 'PH-SLS-B4AA10'`).Scan(&activated, &deviceHash); err != nil {
		t.Fatalf("failed to load restored license activation: %v", err)
	}
	if activated != 1 || deviceHash != "device-123" {
		t.Fatalf("expected existing Sales Test activation to be preserved, got activated=%d device=%q", activated, deviceHash)
	}

	matches, err := filepath.Glob(appDataPath + ".reseed-backup-*")
	if err != nil || len(matches) == 0 {
		t.Fatalf("expected existing hollow db to be backed up, matches=%v err=%v", matches, err)
	}
}

func TestSeedAppDataDatabaseReplacesMateriallyStaleExistingDB(t *testing.T) {
	dir := t.TempDir()
	appDataPath := filepath.Join(dir, "appdata", "ph_holdings.db")
	packagedPath := filepath.Join(dir, "package", "data", "ph_holdings.db")

	createDeploymentProfileDB(t, appDataPath, 381, 35, 17, 0, 0)
	db, err := sql.Open("sqlite3", appDataPath)
	if err != nil {
		t.Fatalf("failed to open stale appdata db: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE license_keys (
			key TEXT PRIMARY KEY,
			role TEXT,
			display_name TEXT,
			device_hash TEXT,
			activated INTEGER,
			activated_at TEXT,
			notes TEXT,
			created_by TEXT
		);
		INSERT INTO license_keys (key, role, display_name, device_hash, activated, activated_at)
		VALUES ('PH-SLS-B4AA10', 'sales', 'Sales Test', 'sales-device', 1, '2026-04-20 11:00:00');
	`); err != nil {
		db.Close()
		t.Fatalf("failed to seed stale appdata license: %v", err)
	}
	db.Close()

	createDeploymentProfileDB(t, packagedPath, 453, 35, 196, 470, 586)
	db, err = sql.Open("sqlite3", packagedPath)
	if err != nil {
		t.Fatalf("failed to open packaged db: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE license_keys (
			key TEXT PRIMARY KEY,
			role TEXT,
			display_name TEXT,
			device_hash TEXT,
			activated INTEGER,
			activated_at TEXT,
			notes TEXT,
			created_by TEXT
		);
		INSERT INTO license_keys (key, role, display_name, device_hash, activated, activated_at)
		VALUES ('PH-SLS-B4AA10', 'sales', 'Sales Test', '', 0, NULL);
	`); err != nil {
		db.Close()
		t.Fatalf("failed to seed packaged license: %v", err)
	}
	db.Close()

	if !seedAppDataDatabaseFromPackaged(appDataPath, packagedPath) {
		t.Fatalf("expected materially stale appdata db to be replaced from packaged seed")
	}

	profile := readDeploymentDatabaseProfile(appDataPath)
	if profile.Opportunities != 586 || profile.Orders != 196 || profile.Invoices != 470 {
		t.Fatalf("expected current packaged business data after reseed, got %+v", profile)
	}

	db, err = sql.Open("sqlite3", appDataPath)
	if err != nil {
		t.Fatalf("failed to reopen reseeded db: %v", err)
	}
	defer db.Close()
	var activated int
	var deviceHash string
	if err := db.QueryRow(`SELECT activated, device_hash FROM license_keys WHERE key = 'PH-SLS-B4AA10'`).Scan(&activated, &deviceHash); err != nil {
		t.Fatalf("failed to load restored license activation: %v", err)
	}
	if activated != 1 || deviceHash != "sales-device" {
		t.Fatalf("expected Sales Test activation to survive stale DB reseed, got activated=%d device=%q", activated, deviceHash)
	}
}

func TestSeedAppDataDatabaseFlushesActivationWhenRequested(t *testing.T) {
	dir := t.TempDir()
	appDataPath := filepath.Join(dir, "appdata", "ph_holdings.db")
	packagedPath := filepath.Join(dir, "package", "data", "ph_holdings.db")

	createDeploymentProfileDB(t, appDataPath, 381, 35, 17, 0, 0)
	db, err := sql.Open("sqlite3", appDataPath)
	if err != nil {
		t.Fatalf("failed to open stale appdata db: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE license_keys (
			key TEXT PRIMARY KEY,
			role TEXT,
			display_name TEXT,
			device_hash TEXT,
			activated INTEGER,
			activated_at TEXT,
			notes TEXT,
			created_by TEXT
		);
		INSERT INTO license_keys (key, role, display_name, device_hash, activated, activated_at)
		VALUES ('PH-SLS-B4AA10', 'sales', 'Sales Test', 'sales-device', 1, '2026-04-20 11:00:00');
	`); err != nil {
		db.Close()
		t.Fatalf("failed to seed stale appdata license: %v", err)
	}
	db.Close()

	createDeploymentProfileDB(t, packagedPath, 453, 35, 196, 470, 586)
	db, err = sql.Open("sqlite3", packagedPath)
	if err != nil {
		t.Fatalf("failed to open packaged db: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE license_keys (
			key TEXT PRIMARY KEY,
			role TEXT,
			display_name TEXT,
			device_hash TEXT,
			activated INTEGER,
			activated_at TEXT,
			notes TEXT,
			created_by TEXT
		);
		INSERT INTO license_keys (key, role, display_name, device_hash, activated, activated_at)
		VALUES ('PH-SLS-B4AA10', 'sales', 'Sales Test', '', 0, NULL);
	`); err != nil {
		db.Close()
		t.Fatalf("failed to seed packaged license: %v", err)
	}
	db.Close()

	originalFlush, hadFlush := os.LookupEnv("ASYMMFLOW_FLUSH_LICENSE_ON_RESEED")
	if err := os.Setenv("ASYMMFLOW_FLUSH_LICENSE_ON_RESEED", "true"); err != nil {
		t.Fatalf("failed to set flush env: %v", err)
	}
	defer func() {
		if hadFlush {
			_ = os.Setenv("ASYMMFLOW_FLUSH_LICENSE_ON_RESEED", originalFlush)
		} else {
			_ = os.Unsetenv("ASYMMFLOW_FLUSH_LICENSE_ON_RESEED")
		}
	}()

	if !seedAppDataDatabaseFromPackaged(appDataPath, packagedPath) {
		t.Fatalf("expected stale appdata db to be replaced from packaged seed")
	}

	db, err = sql.Open("sqlite3", appDataPath)
	if err != nil {
		t.Fatalf("failed to reopen reseeded db: %v", err)
	}
	defer db.Close()
	var activated int
	var deviceHash string
	if err := db.QueryRow(`SELECT activated, device_hash FROM license_keys WHERE key = 'PH-SLS-B4AA10'`).Scan(&activated, &deviceHash); err != nil {
		t.Fatalf("failed to load flushed license activation: %v", err)
	}
	if activated != 0 || deviceHash != "" {
		t.Fatalf("expected Sales Test activation to be flushed, got activated=%d device=%q", activated, deviceHash)
	}
}

func createDeploymentProfileDB(t *testing.T, path string, customers, suppliers, orders, invoices, opportunities int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("failed to create db dir: %v", err)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE customers (id INTEGER PRIMARY KEY, deleted_at TEXT);`,
		`CREATE TABLE suppliers (id INTEGER PRIMARY KEY, deleted_at TEXT);`,
		`CREATE TABLE orders (id INTEGER PRIMARY KEY);`,
		`CREATE TABLE invoices (id INTEGER PRIMARY KEY);`,
		`CREATE TABLE opportunities (id INTEGER PRIMARY KEY);`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("failed to create test table: %v", err)
		}
	}
	insertRows := func(table string, count int) {
		t.Helper()
		for i := 0; i < count; i++ {
			if _, err := db.Exec("INSERT INTO " + table + " DEFAULT VALUES"); err != nil {
				t.Fatalf("failed to insert into %s: %v", table, err)
			}
		}
	}
	insertRows("customers", customers)
	insertRows("suppliers", suppliers)
	insertRows("orders", orders)
	insertRows("invoices", invoices)
	insertRows("opportunities", opportunities)
}

func TestConfigValidation(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Path: "./test.db",
		},
		App: AppConfig{
			LogLevel:          "info",
			WatcherDebounceMS: 300,
			WatcherQueueSize:  1000,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}

	// Test invalid log level
	cfg.App.LogLevel = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("Expected validation to fail with invalid log level")
	}
	cfg.App.LogLevel = "info" // Restore

	// Test invalid queue size
	cfg.App.WatcherQueueSize = 0
	if err := cfg.Validate(); err == nil {
		t.Error("Expected validation to fail with queue size 0")
	}
	cfg.App.WatcherQueueSize = 1000 // Restore

	t.Logf("✓ Config validation working correctly")
}

func TestToolAutoDetection(t *testing.T) {
	tools := &ToolsConfig{}
	tools.detectTools()

	// We can't guarantee what tools are installed, but the function should not panic
	t.Logf("✓ Tool auto-detection ran without errors")
	t.Logf("  Pandoc:    %s", maskNotFound(tools.PandocPath))
	t.Logf("  FFmpeg:    %s", maskNotFound(tools.FFmpegPath))
	t.Logf("  Tesseract: %s", maskNotFound(tools.TesseractPath))
}

func TestAzureConfigEnabled(t *testing.T) {
	// All fields set - should be enabled
	cfg := &Config{
		Azure: AzureConfig{
			TenantID:     "tenant-123",
			ClientID:     "client-456",
			ClientSecret: "secret-789",
		},
	}
	cfg.Azure.Enabled = cfg.Azure.TenantID != "" && cfg.Azure.ClientID != "" && cfg.Azure.ClientSecret != ""

	if !cfg.Azure.Enabled {
		t.Error("Azure should be enabled when all credentials are set")
	}

	// Missing client secret - should be disabled
	cfg2 := &Config{
		Azure: AzureConfig{
			TenantID:     "tenant-123",
			ClientID:     "client-456",
			ClientSecret: "",
		},
	}
	cfg2.Azure.Enabled = cfg2.Azure.TenantID != "" && cfg2.Azure.ClientID != "" && cfg2.Azure.ClientSecret != ""

	if cfg2.Azure.Enabled {
		t.Error("Azure should be disabled when credentials are incomplete")
	}

	t.Logf("✓ Azure enabled/disabled logic working correctly")
}

func TestMaskingFunctions(t *testing.T) {
	// Test secret masking
	masked := maskSecret("1234567890abcdef")
	if masked != "1234****cdef" {
		t.Errorf("Expected '1234****cdef', got '%s'", masked)
	}

	// Test short secret
	masked = maskSecret("abc")
	if masked != "****" {
		t.Errorf("Expected '****', got '%s'", masked)
	}

	// Test empty
	masked = maskSecret("")
	if masked != "(not set)" {
		t.Errorf("Expected '(not set)', got '%s'", masked)
	}

	// Test maskEmpty
	masked = maskEmpty("")
	if masked != "(not configured)" {
		t.Errorf("Expected '(not configured)', got '%s'", masked)
	}

	// Test maskNotFound
	masked = maskNotFound("")
	if masked != "(not found in PATH)" {
		t.Errorf("Expected '(not found in PATH)', got '%s'", masked)
	}

	t.Logf("✓ Masking functions working correctly")
}
