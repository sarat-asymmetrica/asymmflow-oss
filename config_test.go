// ═══════════════════════════════════════════════════════════════════════════
// CONFIG TESTS - Verify configuration loading and validation
//
// Mission DP1: database-path resolution and the seed/migrate/stamp update
// contract moved to pkg/infra/deploy (see pkg/infra/deploy/deploy_paths_test.go
// and update_contract_test.go). The count-heuristic reseed tests that used to
// live here were retired with the behavior they covered — the anti-reseed proof
// is now the byte-compare test in update_contract_test.go. The DATABASE_PATH
// escape hatch was retired in favor of PH_DB_PATH.
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
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
	// PH_DB_PATH is the sole dev escape hatch (DATABASE_PATH retired, Mission DP1).
	// An absolute PH_DB_PATH is honored verbatim as the database path.
	customDB := filepath.Join(t.TempDir(), "test_custom.db")
	t.Setenv("PH_DB_PATH", customDB)
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DEBUG_MODE", "true")
	t.Setenv("WATCHER_DEBOUNCE_MS", "500")
	t.Setenv("ENABLE_FILE_WATCHER", "false")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if cfg.Database.Path != filepath.Clean(customDB) {
		t.Errorf("Expected database path '%s', got %s", filepath.Clean(customDB), cfg.Database.Path)
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

// TestGetDatabasePathRelativeEnvResolvesAgainstCWD proves the surviving CWD
// dev-DB flow: a relative PH_DB_PATH is resolved against the working directory
// (the mechanism that keeps `wails dev` finding an in-repo database after the
// six-priority archaeology was retired).
func TestGetDatabasePathRelativeEnvResolvesAgainstCWD(t *testing.T) {
	t.Setenv("PH_DB_PATH", filepath.Join("data", "ph_holdings.db"))

	resolved := getDatabasePath()
	if !filepath.IsAbs(resolved) {
		t.Fatalf("expected relative PH_DB_PATH to resolve to an absolute path, got %s", resolved)
	}
	if !strings.HasSuffix(filepath.ToSlash(resolved), "data/ph_holdings.db") {
		t.Fatalf("expected resolved path to end with data/ph_holdings.db, got %s", resolved)
	}
	cwd, _ := os.Getwd()
	if !strings.HasPrefix(resolved, filepath.Clean(cwd)) {
		t.Fatalf("expected relative PH_DB_PATH to resolve under CWD %s, got %s", cwd, resolved)
	}
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
