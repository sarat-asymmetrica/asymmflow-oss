// ═══════════════════════════════════════════════════════════════════════════
// CONFIGURATION SYSTEM - Production-Ready Environment Management
//
// MISSION: Centralized config with .env support, validation, and auto-detection
//
// FEATURES:
//   1. Environment variable loading via .env file
//   2. Sensible defaults for all paths and settings
//   3. Path validation (directories must exist)
//   4. Auto-detection of external tools (pandoc, ffmpeg, tesseract)
//   5. Masked logging of sensitive values (Azure credentials)
//
// ARCHITECTURE:
//   - Config struct with nested sections (OneDrive, Database, Azure, Tools, App)
//   - LoadConfig() reads .env and validates
//   - getEnv() helper with default fallback
//   - validatePaths() ensures OneDrive folders exist
//   - detectTool() finds executables in PATH
//
// DEPLOYMENT PATHS (Mission DP1): database-path resolution and the seed/
// migrate/stamp update contract moved to pkg/infra/deploy. The six-priority
// path archaeology and the count-heuristic reseed engine that used to live here
// were retired — resolution is now configuration, not inference.
//
// Built with MATHEMATICAL RIGOR × PRODUCTION ROBUSTNESS × CONFIGURATION CLARITY
// Day 192 - Configuration System Wave 2
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/ncruces/go-sqlite3/driver" // register the database/sql "sqlite3" driver process-wide (pure-Go; CGO banned)

	"ph_holdings_app/pkg/infra/deploy"
	"ph_holdings_app/pkg/runtime/composition"
)

// ============================================================================
// CONFIG STRUCTURE
// ============================================================================

// Config holds all application configuration
type Config struct {
	OneDrive OneDriveConfig
	Database DatabaseConfig
	Azure    AzureConfig
	AI       AIConfig
	Supabase SupabaseConfig
	Tools    ToolsConfig
	App      AppConfig
}

// SupabaseConfig defines Supabase/PostgreSQL connection settings
type SupabaseConfig struct {
	URL           string // Supabase project URL
	AnonKey       string // Public anon key
	ServiceKey    string // Service role key (server-side only)
	DBHost        string // PostgreSQL host (db.<project>.supabase.co)
	DBPort        string // PostgreSQL port (default 5432)
	DBName        string // Database name (default postgres)
	DBUser        string // Database user (default postgres)
	DBPassword    string // Database password
	DBSSLMode     string // SSL mode (default require)
	StorageBucket string // Storage bucket name for reports
	Enabled       bool   // True if URL and key are configured
}

// OneDriveConfig defines OneDrive folder paths
type OneDriveConfig struct {
	RFQPath      string // C:/Users/.../OneDrive/Acme Instrumentation/RFQs
	EHPath       string // C:/Users/.../OneDrive/Acme Instrumentation/Supplier Data
	OffersPath   string // C:/Users/.../OneDrive/Acme Instrumentation/Offers
	InvoicesPath string // C:/Users/.../OneDrive/Acme Instrumentation/Invoices
}

// DatabaseConfig defines database settings
type DatabaseConfig struct {
	Path string // ./ph_holdings.db
}

// AzureConfig defines Microsoft Graph / Azure AD credentials
type AzureConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	Enabled      bool // True if all three fields are non-empty
}

// AIConfig defines AI provider settings
type AIConfig struct {
	Provider string // "mistral", "openai", "anthropic" — AIMLAPI/Grok removed in Wave 13
	APIKey   string
	Model    string // Model name/ID
}

// ToolsConfig defines external tool paths
type ToolsConfig struct {
	PandocPath    string // Auto-detected if empty
	FFmpegPath    string // Auto-detected if empty
	TesseractPath string // Auto-detected if empty
}

// AppConfig defines application behavior settings
type AppConfig struct {
	LogLevel                 string // debug, info, warn, error
	DebugMode                bool
	EnableDeveloperMasterKey bool   // Allow the env-supplied developer master key (ASYMMFLOW_MASTER_KEY) in trusted/dev environments only
	WatcherDebounceMS        int    // Debounce delay in milliseconds
	WatcherQueueSize         int    // Max event queue size
	EnableFileWatcher        bool   // Auto-start file watcher?
	EnableGeometryBridge     bool   // Enable three-regime routing?
	EnableAutoBackup         bool   // Hourly database backups?
	BackupRetentionDays      int    // How long to keep backups
	AllowedOrigins           string // Comma-separated CORS origins (default: localhost only)
	RateLimitPerMinute       int    // Rate limit per IP per minute (default: 60)
}

// ============================================================================
// CONFIG LOADING
// ============================================================================

// executableSearchDirs delegates to the shared composition seam so every
// vertical resolves portable-deployment config from the same locations.
func executableSearchDirs() []string {
	return composition.ExecutableSearchDirs()
}

func loadEnvFilesWithPrecedence(envLocations []string) []string {
	originalEnv := make(map[string]struct{})
	for _, pair := range os.Environ() {
		key, _, ok := strings.Cut(pair, "=")
		if ok && key != "" {
			originalEnv[key] = struct{}{}
		}
	}

	loaded := make([]string, 0, len(envLocations))
	for i := len(envLocations) - 1; i >= 0; i-- {
		loc := envLocations[i]
		values, err := godotenv.Read(loc)
		if err != nil {
			continue
		}
		for key, value := range values {
			if _, protected := originalEnv[key]; protected {
				continue
			}
			_ = os.Setenv(key, value)
		}
		loaded = append(loaded, loc)
	}

	for left, right := 0, len(loaded)-1; left < right; left, right = left+1, right-1 {
		loaded[left], loaded[right] = loaded[right], loaded[left]
	}
	return loaded
}

// getDatabasePath returns the resolved live-database path. It delegates to the
// deploy package's total three-step resolver: (1) PH_DB_PATH env (dev escape,
// logged loudly); (2) portable.flag next to the exe → exe-dir data\;
// (3) DataDir(). Nothing else — DATABASE_PATH, CWD scanning, exe-dir search,
// and packaged-path pinning were all retired in Mission DP1. Path resolution
// no longer seeds, replaces, or migrates anything; that is the update
// contract's job (deploy.EnsureDatabase, invoked at boot in app.go).
func getDatabasePath() string {
	path, source := deploy.ResolveDatabasePathVerbose()
	if source == "PH_DB_PATH" {
		log.Printf("📂 Database path from PH_DB_PATH (dev escape hatch): %s", path)
	} else {
		log.Printf("📂 Database path (%s): %s", source, path)
	}
	return path
}

// LoadConfig loads configuration from .env file (if exists) and environment
func LoadConfig() (*Config, error) {
	// Try to load .env file from multiple locations
	// This is needed because the app can run from build/bin/ or project root
	envLocations := make([]string, 0, 6)

	// Prefer executable-adjacent locations over platform data so a deployment package's
	// own .env wins over stale config from an older install. This also covers macOS
	// app bundles, where the executable lives inside Contents/MacOS.
	for _, baseDir := range executableSearchDirs() {
		envLocations = append(envLocations, filepath.Join(baseDir, ".env"))
	}

	// Current directory remains useful in development.
	envLocations = append(envLocations, ".env")

	// Data plane (production fallback) — the per-user data directory keyed off
	// the deployment slug (%APPDATA%\Asymmetrica\<slug>\data). The legacy
	// %APPDATA%\AsymmFlow directory is never consulted (Mission DP1 invariant).
	if dataDir := deploy.DataDir(); dataDir != "" {
		envLocations = append(envLocations, filepath.Join(dataDir, ".env"))
	}

	loadedEnvFiles := loadEnvFilesWithPrecedence(envLocations)
	if len(loadedEnvFiles) == 0 {
		log.Println("Note: No .env file found, using environment variables only")
	} else {
		for _, loc := range loadedEnvFiles {
			log.Printf("✓ Loaded configuration from %s", loc)
		}
	}

	// Build config with defaults
	cfg := &Config{
		OneDrive: OneDriveConfig{
			RFQPath:      getEnv("ONEDRIVE_RFQ_PATH", ""),
			EHPath:       getEnv("ONEDRIVE_EH_PATH", ""),
			OffersPath:   getEnv("ONEDRIVE_OFFERS_PATH", ""),
			InvoicesPath: getEnv("ONEDRIVE_INVOICES_PATH", ""),
		},
		Database: DatabaseConfig{
			Path: getDatabasePath(),
		},
		Azure: AzureConfig{
			TenantID:     getEnv("AZURE_TENANT_ID", ""),
			ClientID:     getEnv("AZURE_CLIENT_ID", ""),
			ClientSecret: getEnv("AZURE_CLIENT_SECRET", ""),
		},
		Tools: ToolsConfig{
			PandocPath:    getEnv("PANDOC_PATH", ""),
			FFmpegPath:    getEnv("FFMPEG_PATH", ""),
			TesseractPath: getEnv("TESSERACT_PATH", ""),
		},
		App: AppConfig{
			LogLevel:                 getEnv("LOG_LEVEL", "info"),
			DebugMode:                getEnvBool("DEBUG_MODE", false),
			EnableDeveloperMasterKey: getEnvBool("ENABLE_DEVELOPER_MASTER_KEY", false),
			WatcherDebounceMS:        getEnvInt("WATCHER_DEBOUNCE_MS", 300),
			WatcherQueueSize:         getEnvInt("WATCHER_QUEUE_SIZE", 1000),
			EnableFileWatcher:        getEnvBool("ENABLE_FILE_WATCHER", true),
			EnableGeometryBridge:     getEnvBool("ENABLE_GEOMETRY_BRIDGE", true),
			EnableAutoBackup:         getEnvBool("ENABLE_AUTO_BACKUP", false),
			BackupRetentionDays:      getEnvInt("BACKUP_RETENTION_DAYS", 30),
			AllowedOrigins:           getEnv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:34115,http://localhost:8080,http://127.0.0.1:5173,http://127.0.0.1:34115,http://127.0.0.1:8080"),
			RateLimitPerMinute:       getEnvInt("RATE_LIMIT_PER_MINUTE", 60),
		},
	}

	// Database configuration - check DATABASE_URL first (Render), then SUPABASE_* vars
	cfg.Supabase = loadDatabaseConfig()

	// Determine if Azure integration is enabled
	cfg.Azure.Enabled = cfg.Azure.TenantID != "" &&
		cfg.Azure.ClientID != "" &&
		cfg.Azure.ClientSecret != ""

	// Auto-detect tool paths if not specified
	cfg.Tools.detectTools()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Log loaded configuration (mask secrets)
	cfg.LogConfig()

	return cfg, nil
}

// ============================================================================
// VALIDATION
// ============================================================================

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Database path must be writable
	dbDir := filepath.Dir(c.Database.Path)
	if dbDir != "" && dbDir != "." {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("cannot create database directory %s: %w", dbDir, err)
		}
	}

	// OneDrive paths are optional, but if specified, must exist
	// We'll validate them in ValidateOneDrivePaths() which can be called separately

	// Log level validation
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.App.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.App.LogLevel)
	}

	// Watcher settings validation
	if c.App.WatcherDebounceMS < 0 {
		return fmt.Errorf("watcher debounce delay cannot be negative: %d", c.App.WatcherDebounceMS)
	}
	if c.App.WatcherQueueSize < 1 {
		return fmt.Errorf("watcher queue size must be at least 1: %d", c.App.WatcherQueueSize)
	}

	return nil
}

// ValidateOneDrivePaths checks if OneDrive paths exist (optional validation)
func (c *Config) ValidateOneDrivePaths() []error {
	var errs []error

	checkPath := func(name, path string) {
		if path == "" {
			return // Empty paths are allowed
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("%s path does not exist: %s", name, path))
		}
	}

	checkPath("RFQ", c.OneDrive.RFQPath)
	checkPath("Rhine Instruments", c.OneDrive.EHPath)
	checkPath("Offers", c.OneDrive.OffersPath)
	checkPath("Invoices", c.OneDrive.InvoicesPath)

	return errs
}

// ============================================================================
// TOOL AUTO-DETECTION
// ============================================================================

// detectTools attempts to find external tools in PATH
func (tc *ToolsConfig) detectTools() {
	// Pandoc
	if tc.PandocPath == "" {
		tc.PandocPath = detectTool("pandoc")
	}

	// FFmpeg
	if tc.FFmpegPath == "" {
		tc.FFmpegPath = detectTool("ffmpeg")
	}

	// Tesseract
	if tc.TesseractPath == "" {
		tc.TesseractPath = detectTool("tesseract")
	}
}

// detectTool searches for an executable in PATH
func detectTool(name string) string {
	// On Windows, try with .exe extension
	if runtime.GOOS == "windows" {
		if path, err := exec.LookPath(name + ".exe"); err == nil {
			return path
		}
	}

	// Try without extension
	if path, err := exec.LookPath(name); err == nil {
		return path
	}

	// Not found
	return ""
}

// ============================================================================
// LOGGING
// ============================================================================

// LogConfig prints the loaded configuration (masking secrets)
func (c *Config) LogConfig() {
	log.Println("═══════════════════════════════════════════════════════════")
	log.Println("CONFIGURATION LOADED")
	log.Println("═══════════════════════════════════════════════════════════")

	// OneDrive Paths
	log.Printf("OneDrive RFQ Path:      %s", maskEmpty(c.OneDrive.RFQPath))
	log.Printf("OneDrive Rhine Instruments Path:      %s", maskEmpty(c.OneDrive.EHPath))
	log.Printf("OneDrive Offers Path:   %s", maskEmpty(c.OneDrive.OffersPath))
	log.Printf("OneDrive Invoices Path: %s", maskEmpty(c.OneDrive.InvoicesPath))

	// Database
	log.Printf("Database Path:          %s", c.Database.Path)

	// Azure
	if c.Azure.Enabled {
		log.Printf("Azure Integration:      ENABLED")
		log.Printf("  Tenant ID:            %s", maskSecret(c.Azure.TenantID))
		log.Printf("  Client ID:            %s", maskSecret(c.Azure.ClientID))
		log.Printf("  Client Secret:        %s", maskSecret(c.Azure.ClientSecret))
	} else {
		log.Printf("Azure Integration:      DISABLED (credentials not configured)")
	}

	// Tools
	log.Printf("Pandoc Path:            %s", maskNotFound(c.Tools.PandocPath))
	log.Printf("FFmpeg Path:            %s", maskNotFound(c.Tools.FFmpegPath))
	log.Printf("Tesseract Path:         %s", maskNotFound(c.Tools.TesseractPath))

	// App Settings
	log.Printf("Log Level:              %s", c.App.LogLevel)
	log.Printf("Debug Mode:             %v", c.App.DebugMode)
	log.Printf("Developer Master Key:   %s", enabledStr(c.App.EnableDeveloperMasterKey))
	log.Printf("File Watcher:           %s", enabledStr(c.App.EnableFileWatcher))
	log.Printf("Geometry Bridge:        %s", enabledStr(c.App.EnableGeometryBridge))
	log.Printf("Auto Backup:            %s", enabledStr(c.App.EnableAutoBackup))
	if c.App.EnableAutoBackup {
		log.Printf("  Retention Days:       %d", c.App.BackupRetentionDays)
	}
	log.Printf("Allowed CORS Origins:   %s", c.App.AllowedOrigins)
	log.Printf("Rate Limit/Minute:      %d", c.App.RateLimitPerMinute)

	log.Println("═══════════════════════════════════════════════════════════")
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// getEnv retrieves an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool retrieves a boolean environment variable
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Warning: Invalid boolean for %s: %s (using default: %v)", key, value, defaultValue)
			return defaultValue
		}
		return parsed
	}
	return defaultValue
}

// getEnvInt retrieves an integer environment variable
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("Warning: Invalid integer for %s: %s (using default: %d)", key, value, defaultValue)
			return defaultValue
		}
		return parsed
	}
	return defaultValue
}

// maskSecret shows only first/last 4 chars of a secret
func maskSecret(secret string) string {
	if secret == "" {
		return "(not set)"
	}
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}

// maskEmpty shows "(not configured)" for empty strings
func maskEmpty(value string) string {
	if value == "" {
		return "(not configured)"
	}
	return value
}

// maskNotFound shows "(not found)" for empty tool paths
func maskNotFound(value string) string {
	if value == "" {
		return "(not found in PATH)"
	}
	return value
}

// enabledStr returns "ENABLED" or "DISABLED"
func enabledStr(enabled bool) string {
	if enabled {
		return "ENABLED"
	}
	return "DISABLED"
}

// ============================================================================
// VALIDATION HELPERS
// ============================================================================

// EnsureOneDrivePaths validates and creates OneDrive paths if needed
func (c *Config) EnsureOneDrivePaths() error {
	paths := []struct {
		name string
		path *string
	}{
		{"RFQ", &c.OneDrive.RFQPath},
		{"Rhine Instruments", &c.OneDrive.EHPath},
		{"Offers", &c.OneDrive.OffersPath},
		{"Invoices", &c.OneDrive.InvoicesPath},
	}

	for _, p := range paths {
		if *p.path == "" {
			continue // Skip empty paths
		}

		// Expand ~ to home directory
		if strings.HasPrefix(*p.path, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot expand home directory for %s: %w", p.name, err)
			}
			*p.path = filepath.Join(home, (*p.path)[1:])
		}

		// Check if path exists
		if _, err := os.Stat(*p.path); os.IsNotExist(err) {
			// Try to create the directory
			if err := os.MkdirAll(*p.path, 0755); err != nil {
				return fmt.Errorf("cannot create %s path %s: %w", p.name, *p.path, err)
			}
			log.Printf("Created %s directory: %s", p.name, *p.path)
		}
	}

	return nil
}

// ============================================================================
// DATABASE URL PARSING (Render.com / Heroku / Railway support)
// ============================================================================

// loadDatabaseConfig loads Supabase config for cloud sync (SQLite is primary)
func loadDatabaseConfig() SupabaseConfig {
	// Check if cloud sync is enabled
	enableSync := getEnvBool("ENABLE_CLOUD_SYNC", false)

	// Priority 1: DATABASE_URL (if provided by hosting platform)
	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL != "" {
		cfg, err := parseDatabaseURL(databaseURL)
		if err != nil {
			log.Printf("Warning: Failed to parse DATABASE_URL: %v, trying SUPABASE_* vars", err)
		} else {
			cfg.Enabled = enableSync
			log.Println("Supabase config loaded from DATABASE_URL")
			return cfg
		}
	}

	// Priority 2: Individual SUPABASE_* environment variables
	cfg := SupabaseConfig{
		URL:           getEnv("SUPABASE_URL", ""),
		AnonKey:       getEnv("SUPABASE_ANON_KEY", ""),
		ServiceKey:    getEnv("SUPABASE_SERVICE_KEY", ""),
		DBHost:        getEnv("SUPABASE_DB_HOST", ""),
		DBPort:        getEnv("SUPABASE_DB_PORT", "5432"),
		DBName:        getEnv("SUPABASE_DB_NAME", "postgres"),
		DBUser:        getEnv("SUPABASE_DB_USER", "postgres"),
		DBPassword:    getEnv("SUPABASE_DB_PASSWORD", ""),
		DBSSLMode:     getEnv("SUPABASE_DB_SSLMODE", "require"),
		StorageBucket: getEnv("SUPABASE_STORAGE_BUCKET", "reports"),
	}

	// Only enable if credentials are provided AND sync is enabled
	cfg.Enabled = enableSync && cfg.DBHost != "" && cfg.DBPassword != ""

	if cfg.Enabled {
		log.Printf("Supabase sync enabled (host: %s)", cfg.DBHost)
	} else if enableSync && cfg.DBHost == "" {
		log.Println("Supabase sync disabled (no credentials configured)")
	} else {
		log.Println("Supabase sync disabled (ENABLE_CLOUD_SYNC=false)")
	}

	return cfg
}

// parseDatabaseURL parses a PostgreSQL connection URL into SupabaseConfig
// Format: postgres://user:password@host:port/dbname?sslmode=require
func parseDatabaseURL(rawURL string) (SupabaseConfig, error) {
	cfg := SupabaseConfig{
		DBPort:        "5432",
		DBSSLMode:     "require",
		StorageBucket: getEnv("SUPABASE_STORAGE_BUCKET", "reports"),
	}

	// Handle both postgres:// and postgresql:// schemes
	rawURL = strings.Replace(rawURL, "postgresql://", "postgres://", 1)

	// Parse the URL
	u, err := parsePostgresURL(rawURL)
	if err != nil {
		return cfg, err
	}

	cfg.DBUser = u.user
	cfg.DBPassword = u.password
	cfg.DBHost = u.host
	if u.port != "" {
		cfg.DBPort = u.port
	}
	cfg.DBName = u.dbname
	if u.sslmode != "" {
		cfg.DBSSLMode = u.sslmode
	}

	// Mark as enabled (for DATABASE_URL, we don't need URL/AnonKey)
	cfg.URL = "render://" + u.host // Placeholder to indicate remote DB
	cfg.AnonKey = "database-url"   // Placeholder
	cfg.Enabled = true

	return cfg, nil
}

// postgresURLParts holds parsed URL components
type postgresURLParts struct {
	user     string
	password string
	host     string
	port     string
	dbname   string
	sslmode  string
}

// parsePostgresURL manually parses postgres:// URLs to avoid net/url issues with passwords
func parsePostgresURL(rawURL string) (postgresURLParts, error) {
	var p postgresURLParts

	// Remove scheme
	if !strings.HasPrefix(rawURL, "postgres://") {
		return p, fmt.Errorf("invalid postgres URL: must start with postgres://")
	}
	rawURL = strings.TrimPrefix(rawURL, "postgres://")

	// Split by @ to separate credentials from host
	atIdx := strings.LastIndex(rawURL, "@")
	if atIdx == -1 {
		return p, fmt.Errorf("invalid postgres URL: missing @ separator")
	}

	credentials := rawURL[:atIdx]
	hostPart := rawURL[atIdx+1:]

	// Parse credentials (user:password)
	colonIdx := strings.Index(credentials, ":")
	if colonIdx == -1 {
		p.user = credentials
	} else {
		p.user = credentials[:colonIdx]
		p.password = credentials[colonIdx+1:]
	}

	// Split host part by / to get dbname
	slashIdx := strings.Index(hostPart, "/")
	if slashIdx == -1 {
		return p, fmt.Errorf("invalid postgres URL: missing database name")
	}

	hostPort := hostPart[:slashIdx]
	dbQuery := hostPart[slashIdx+1:]

	// Parse host:port
	if strings.Contains(hostPort, ":") {
		parts := strings.SplitN(hostPort, ":", 2)
		p.host = parts[0]
		p.port = parts[1]
	} else {
		p.host = hostPort
	}

	// Parse dbname and query params
	if strings.Contains(dbQuery, "?") {
		parts := strings.SplitN(dbQuery, "?", 2)
		p.dbname = parts[0]
		// Parse query params for sslmode
		for _, param := range strings.Split(parts[1], "&") {
			kv := strings.SplitN(param, "=", 2)
			if len(kv) == 2 && kv[0] == "sslmode" {
				p.sslmode = kv[1]
			}
		}
	} else {
		p.dbname = dbQuery
	}

	return p, nil
}
