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
// Built with MATHEMATICAL RIGOR × PRODUCTION ROBUSTNESS × CONFIGURATION CLARITY
// Day 192 - Configuration System Wave 2
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/ncruces/go-sqlite3/driver"
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
	Provider string // "aimlapi", "openai", "anthropic"
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

func resolveConfiguredPath(configuredPath string) string {
	if strings.TrimSpace(configuredPath) == "" {
		return configuredPath
	}
	if filepath.IsAbs(configuredPath) {
		return configuredPath
	}

	candidates := make([]string, 0, len(executableSearchDirs())+1)
	for _, baseDir := range executableSearchDirs() {
		candidates = append(candidates, filepath.Join(baseDir, configuredPath))
	}
	candidates = append(candidates, configuredPath)

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Clean(candidate)
		}
	}

	if dirs := executableSearchDirs(); len(dirs) > 0 {
		return filepath.Clean(filepath.Join(dirs[0], configuredPath))
	}
	return filepath.Clean(configuredPath)
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

func packagedDatabasePath() string {
	for _, baseDir := range executableSearchDirs() {
		candidates := []string{
			filepath.Join(baseDir, "data", "ph_holdings.db"),
			filepath.Join(baseDir, "ph_holdings.db"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return filepath.Clean(candidate)
			}
		}
	}
	return ""
}

// appDataDirPath resolves the per-user application data directory —
// %APPDATA%\AsymmFlow on Windows, ~/.local/share/AsymmFlow elsewhere. Every
// path fallback should route through here; hardcoding the POSIX layout left
// several sites platform-blind (3-PLAT).
func appDataDirPath() string {
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "AsymmFlow")
		}
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "AsymmFlow")
	}
	return ""
}

func appDataDatabasePath() string {
	if appDataDir := appDataDirPath(); appDataDir != "" {
		return filepath.Join(appDataDir, "ph_holdings.db")
	}
	return ""
}

type deploymentDatabaseProfile struct {
	Path             string
	Valid            bool
	Customers        int64
	Suppliers        int64
	Orders           int64
	Invoices         int64
	Opportunities    int64
	NamedLicenseKeys int64
}

type preservedLicenseActivation struct {
	Key         string
	Role        string
	DisplayName string
	DeviceHash  string
	ActivatedAt string
}

func sqliteReadonlyDSN(path string) string {
	return fmt.Sprintf("file:%s?mode=ro&_busy_timeout=5000", filepath.ToSlash(filepath.Clean(path)))
}

func readDeploymentDatabaseProfile(path string) deploymentDatabaseProfile {
	profile := deploymentDatabaseProfile{Path: path}
	if strings.TrimSpace(path) == "" || !isSQLiteDatabaseFile(path) {
		return profile
	}
	db, err := sql.Open("sqlite3", sqliteReadonlyDSN(path))
	if err != nil {
		log.Printf("⚠️ Could not inspect database profile %s: %v", path, err)
		return profile
	}
	defer db.Close()

	profile.Valid = true
	profile.Customers = sqliteTableCount(db, "customers", "deleted_at IS NULL")
	profile.Suppliers = sqliteTableCount(db, "suppliers", "deleted_at IS NULL")
	profile.Orders = sqliteTableCount(db, "orders", "")
	profile.Invoices = sqliteTableCount(db, "invoices", "")
	profile.Opportunities = sqliteTableCount(db, "opportunities", "")
	profile.NamedLicenseKeys = sqliteTableCount(db, "license_keys", "COALESCE(display_name, '') <> ''")
	return profile
}

func sqliteTableCount(db *sql.DB, tableName, whereClause string) int64 {
	if db == nil || strings.TrimSpace(tableName) == "" {
		return 0
	}
	var exists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, tableName).Scan(&exists); err != nil || exists == 0 {
		return 0
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if strings.TrimSpace(whereClause) != "" {
		query += " WHERE " + whereClause
	}
	var count int64
	if err := db.QueryRow(query).Scan(&count); err != nil {
		log.Printf("⚠️ Could not count %s in deployment database profile: %v", tableName, err)
		return 0
	}
	return count
}

func isDeploymentSeedRich(profile deploymentDatabaseProfile) bool {
	if !profile.Valid {
		return false
	}
	return profile.Customers >= 50 &&
		(profile.Orders >= 10 || profile.Invoices >= 10 || profile.Opportunities >= 50)
}

func shouldReplaceAppDataDatabase(existing, packaged deploymentDatabaseProfile) bool {
	if !isDeploymentSeedRich(packaged) {
		return false
	}
	if !existing.Valid {
		return true
	}
	// Empty or activation-only databases from failed first launches must not
	// shadow the bundled company database.
	if existing.Customers == 0 && existing.Orders == 0 && existing.Invoices == 0 && existing.Opportunities == 0 {
		return true
	}
	if existing.Customers < 50 && packaged.Customers >= existing.Customers*3 {
		return true
	}
	if existing.Orders == 0 && existing.Invoices == 0 && existing.Opportunities < 25 {
		return true
	}
	// A stale client app-data database can contain enough rows to look "valid"
	// while still missing the current deployment pipeline. The packaged
	// deployment DB is authoritative when it is materially richer.
	if packaged.Opportunities >= 100 && existing.Opportunities < packaged.Opportunities/2 {
		return true
	}
	if packaged.Orders >= 50 && existing.Orders < packaged.Orders/2 && existing.Opportunities < packaged.Opportunities*3/4 {
		return true
	}
	if deploymentProfileScore(existing)*4 < deploymentProfileScore(packaged)*3 {
		return true
	}
	return false
}

func deploymentProfileScore(profile deploymentDatabaseProfile) int64 {
	if !profile.Valid {
		return 0
	}
	return profile.Customers +
		profile.Suppliers +
		profile.Orders*3 +
		profile.Invoices*2 +
		profile.Opportunities
}

func deploymentDatabaseReseedStamp() string {
	return strings.TrimSpace(os.Getenv("ASYMMFLOW_DB_RESEED_STAMP"))
}

func appDataDatabaseHasReseedStamp(path, stamp string) bool {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(stamp) == "" || !isSQLiteDatabaseFile(path) {
		return false
	}
	db, err := sql.Open("sqlite3", sqliteReadonlyDSN(path))
	if err != nil {
		return false
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'settings'`).Scan(&count); err != nil || count == 0 {
		return false
	}

	var value string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key = ? AND deleted_at IS NULL`, "deployment_database_reseed_stamp").Scan(&value); err != nil {
		return false
	}
	return strings.TrimSpace(value) == stamp
}

func shouldForceDeploymentDatabaseReseed(appDataPath string) bool {
	stamp := deploymentDatabaseReseedStamp()
	return stamp != "" && !appDataDatabaseHasReseedStamp(appDataPath, stamp)
}

func markDeploymentDatabaseReseedStamp(path string) {
	stamp := deploymentDatabaseReseedStamp()
	if strings.TrimSpace(path) == "" || stamp == "" || !isSQLiteDatabaseFile(path) {
		return
	}
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_busy_timeout=5000", filepath.ToSlash(filepath.Clean(path))))
	if err != nil {
		log.Printf("⚠️ Could not open AppData database to mark deployment reseed stamp: %v", err)
		return
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'settings'`).Scan(&count); err != nil || count == 0 {
		return
	}

	id := "deployment-database-reseed-stamp"
	_, err = db.Exec(`
		INSERT INTO settings (
			id, key, value, category, description, is_encrypted,
			created_at, updated_at, version, created_by, deleted_at
		)
		VALUES (
			?, 'deployment_database_reseed_stamp', ?, 'deployment',
			'Last deployment stamp that replaced the local AppData database from the packaged seed',
			0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1, 'startup-reseed', NULL
		)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			category = excluded.category,
			description = excluded.description,
			is_encrypted = 0,
			updated_at = CURRENT_TIMESTAMP,
			deleted_at = NULL
	`, id, stamp)
	if err != nil {
		log.Printf("⚠️ Could not mark deployment reseed stamp: %v", err)
	}
}

func loadPreservedLicenseActivations(path string) []preservedLicenseActivation {
	if strings.TrimSpace(path) == "" || !isSQLiteDatabaseFile(path) {
		return nil
	}
	db, err := sql.Open("sqlite3", sqliteReadonlyDSN(path))
	if err != nil {
		return nil
	}
	defer db.Close()

	if sqliteTableCount(db, "license_keys", "") == 0 {
		return nil
	}

	rows, err := db.Query(`
		SELECT key, role, COALESCE(display_name, ''), COALESCE(device_hash, ''), COALESCE(CAST(activated_at AS TEXT), '')
		FROM license_keys
		WHERE activated = 1
		  AND COALESCE(device_hash, '') <> ''
		  AND role <> 'developer'
		  AND key NOT LIKE ?
	`, licenseKeyPrefix()+"-DEV-%")
	if err != nil {
		log.Printf("⚠️ Could not read existing license activations before database reseed: %v", err)
		return nil
	}
	defer rows.Close()

	var activations []preservedLicenseActivation
	for rows.Next() {
		var activation preservedLicenseActivation
		if err := rows.Scan(&activation.Key, &activation.Role, &activation.DisplayName, &activation.DeviceHash, &activation.ActivatedAt); err == nil {
			activations = append(activations, activation)
		}
	}
	return activations
}

func restorePreservedLicenseActivations(path string, activations []preservedLicenseActivation) {
	if len(activations) == 0 || strings.TrimSpace(path) == "" || !isSQLiteDatabaseFile(path) {
		return
	}
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_busy_timeout=5000", filepath.ToSlash(filepath.Clean(path))))
	if err != nil {
		log.Printf("⚠️ Could not reopen reseeded database for license activation restore: %v", err)
		return
	}
	defer db.Close()

	for _, activation := range activations {
		if strings.TrimSpace(activation.Key) == "" || strings.TrimSpace(activation.DeviceHash) == "" {
			continue
		}
		result, err := db.Exec(`
			UPDATE license_keys
			SET activated = 1,
			    activated_at = COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP),
			    device_hash = ?,
			    display_name = COALESCE(NULLIF(display_name, ''), ?)
			WHERE key = ?
		`, activation.ActivatedAt, activation.DeviceHash, activation.DisplayName, activation.Key)
		if err != nil {
			log.Printf("⚠️ Could not restore activation for %s: %v", activation.DisplayName, err)
			continue
		}
		if rows, _ := result.RowsAffected(); rows == 0 && activation.Role != "" {
			_, err = db.Exec(`
				INSERT INTO license_keys (key, role, display_name, device_hash, activated, activated_at, notes, created_by)
				VALUES (?, ?, ?, ?, 1, COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP), 'Preserved during deployment database reseed', 'startup-reseed')
			`, activation.Key, activation.Role, activation.DisplayName, activation.DeviceHash, activation.ActivatedAt)
			if err != nil {
				log.Printf("⚠️ Could not insert preserved activation for %s: %v", activation.DisplayName, err)
			}
		}
	}
	log.Printf("🔑 Restored %d existing license activation(s) after database reseed", len(activations))
}

func backupExistingAppDataDatabase(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	if _, err := os.Stat(path); err != nil {
		return
	}
	backupPath := fmt.Sprintf("%s.reseed-backup-%s", path, time.Now().Format("20060102_150405"))
	if err := copyFileContents(path, backupPath, 0600); err != nil {
		log.Printf("⚠️ Could not back up existing AppData database before reseed: %v", err)
		return
	}
	log.Printf("🧷 Backed up existing AppData database before reseed: %s", backupPath)
}

func copyFileContents(src, dst string, perm os.FileMode) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	destination, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		return err
	}
	if err := destination.Sync(); err != nil {
		destination.Close()
		return err
	}
	return destination.Close()
}

func seedAppDataDatabaseFromPackaged(appDataPath, packagedPath string) bool {
	if strings.TrimSpace(appDataPath) == "" || strings.TrimSpace(packagedPath) == "" {
		return false
	}
	appendStartupDiagnostic("DB RESEED: evaluating appData=%s packaged=%s stamp=%s", appDataPath, packagedPath, deploymentDatabaseReseedStamp())
	var preservedActivations []preservedLicenseActivation
	preserveActivations := preserveLicenseActivationsOnReseed()
	if _, err := os.Stat(appDataPath); err == nil {
		if !isSQLiteDatabaseFile(appDataPath) {
			log.Printf("⚠️ Existing AppData database failed SQLite header validation; reseeding from packaged database: %s", appDataPath)
		} else {
			existingProfile := readDeploymentDatabaseProfile(appDataPath)
			packagedProfile := readDeploymentDatabaseProfile(packagedPath)
			forceReseed := shouldForceDeploymentDatabaseReseed(appDataPath)
			if !forceReseed && !shouldReplaceAppDataDatabase(existingProfile, packagedProfile) {
				log.Printf("📂 Database path (AppData existing): %s", appDataPath)
				return true
			}
			if forceReseed {
				log.Printf("⚠️ Deployment database reseed requested for stamp %s; replacing AppData database from packaged seed", deploymentDatabaseReseedStamp())
			} else {
				log.Printf(
					"⚠️ Existing AppData database appears incomplete (customers=%d orders=%d invoices=%d opportunities=%d); reseeding from packaged database (customers=%d orders=%d invoices=%d opportunities=%d)",
					existingProfile.Customers,
					existingProfile.Orders,
					existingProfile.Invoices,
					existingProfile.Opportunities,
					packagedProfile.Customers,
					packagedProfile.Orders,
					packagedProfile.Invoices,
					packagedProfile.Opportunities,
				)
			}
			if preserveActivations {
				preservedActivations = loadPreservedLicenseActivations(appDataPath)
			} else {
				log.Printf("🔑 License activation flush requested during database reseed")
			}
			backupExistingAppDataDatabase(appDataPath)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("⚠️ Could not stat AppData database %s: %v", appDataPath, err)
		return false
	}

	if !isSQLiteDatabaseFile(packagedPath) {
		log.Printf("⚠️ Packaged database failed SQLite header validation: %s", packagedPath)
		return false
	}

	if err := copySQLiteSeedAtomically(packagedPath, appDataPath); err != nil {
		appendStartupDiagnostic("DB RESEED: copy failed: %v", err)
		log.Printf("⚠️ Could not seed AppData database from packaged seed %s: %v", packagedPath, err)
		return false
	}
	markDeploymentDatabaseReseedStamp(appDataPath)
	if preserveActivations {
		restorePreservedLicenseActivations(appDataPath, preservedActivations)
	}
	appendStartupDiagnostic("DB RESEED: success appData=%s", appDataPath)
	log.Printf("📂 Database path (seeded AppData from packaged database): %s", appDataPath)
	return true
}

func preserveLicenseActivationsOnReseed() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("ASYMMFLOW_FLUSH_LICENSE_ON_RESEED")))
	return raw != "1" && raw != "true" && raw != "yes" && raw != "on"
}

func isSQLiteDatabaseFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	header := make([]byte, 16)
	if _, err := io.ReadFull(file, header); err != nil {
		return false
	}
	return string(header) == "SQLite format 3\x00"
}

func copySQLiteSeedAtomically(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return fmt.Errorf("create app data database directory: %w", err)
	}

	tmpPath := dst + ".tmp"
	_ = os.Remove(tmpPath)

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := destination.Sync(); err != nil {
		destination.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := destination.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if !isSQLiteDatabaseFile(tmpPath) {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("copied seed failed SQLite header validation")
	}
	_ = os.Remove(dst + "-wal")
	_ = os.Remove(dst + "-shm")
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("remove existing AppData database before reseed: %w", err)
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Remove(dst + "-wal")
	_ = os.Remove(dst + "-shm")
	if err := os.Chmod(dst, 0600); err != nil {
		log.Printf("⚠️ Could not restrict AppData database permissions for %s: %v", dst, err)
	}
	return nil
}

func isLegacyPackagedDatabasePin(configuredPath, resolvedPath string) bool {
	cleanConfigured := filepath.ToSlash(filepath.Clean(strings.TrimSpace(configuredPath)))
	if cleanConfigured != "data/ph_holdings.db" {
		return false
	}
	packaged := packagedDatabasePath()
	return packaged != "" && filepath.Clean(resolvedPath) == filepath.Clean(packaged)
}

// getDatabasePath returns the database path with proper precedence
func getDatabasePath() string {
	// Priority 1: Environment variable (PH_DB_PATH or DATABASE_PATH)
	if envPath := strings.TrimSpace(os.Getenv("PH_DB_PATH")); envPath != "" {
		resolved := resolveConfiguredPath(envPath)
		log.Printf("📂 Database path from PH_DB_PATH: %s -> %s", envPath, resolved)
		return resolved
	}
	if envPath := strings.TrimSpace(os.Getenv("DATABASE_PATH")); envPath != "" {
		resolved := resolveConfiguredPath(envPath)
		if isLegacyPackagedDatabasePin(envPath, resolved) {
			if appDataPath := appDataDatabasePath(); seedAppDataDatabaseFromPackaged(appDataPath, resolved) {
				log.Printf("📂 DATABASE_PATH points at packaged seed; using persistent AppData DB instead: %s", appDataPath)
				return appDataPath
			}
		}
		log.Printf("📂 Database path from DATABASE_PATH: %s -> %s", envPath, resolved)
		return resolved
	}

	packagedPath := packagedDatabasePath()

	// Priority 2: Existing machine-level application database, but only after
	// comparing it against the packaged seed. A stale app-data DB from an older
	// install must not shadow a richer deployment database.
	appDataPath := appDataDatabasePath()
	if appDataPath != "" {
		if _, err := os.Stat(appDataPath); err == nil {
			if packagedPath != "" {
				if seedAppDataDatabaseFromPackaged(appDataPath, packagedPath) {
					return appDataPath
				}
				log.Printf("⚠️ Existing AppData DB could not be validated against packaged seed; using packaged DB: %s", packagedPath)
				return packagedPath
			}
			log.Printf("📂 Database path (AppData existing): %s", appDataPath)
			return appDataPath
		}
	}

	// Priority 3: Seed the machine-level application database from the packaged
	// database on first run. The packaged DB is a seed, not the live writable DB,
	// so rebuilds and app bundle replacements do not erase local bank/OCR data.
	if packagedPath != "" {
		if appDataPath != "" && seedAppDataDatabaseFromPackaged(appDataPath, packagedPath) {
			return appDataPath
		}
		log.Printf("📂 Database path (packaged app): %s", packagedPath)
		return packagedPath
	}

	// Priority 4: Executable directory (for portable deployment)
	// Prefer the packaged database over CWD so desktop shortcuts and Finder/Explorer
	// launches don't silently fall back to some other working directory.
	for _, baseDir := range executableSearchDirs() {
		candidates := []string{
			filepath.Join(baseDir, "data", "ph_holdings.db"),
			filepath.Join(baseDir, "ph_holdings.db"),
		}
		for _, execPath := range candidates {
			if _, err := os.Stat(execPath); err == nil {
				log.Printf("📂 Database path (exec search): %s", execPath)
				return execPath
			}
		}
	}

	// Priority 5: Current working directory - preferred in development once
	// no packaged database is present next to the executable.
	localPath := "./ph_holdings.db"
	if _, err := os.Stat(localPath); err == nil {
		absPath, _ := filepath.Abs(localPath)
		log.Printf("📂 Database path (local exists): %s", absPath)
		return localPath
	}

	// Priority 6: Application data directory (Windows: %APPDATA%, Unix: ~/.local/share)
	if appDataPath != "" {
		os.MkdirAll(filepath.Dir(appDataPath), 0755)
		log.Printf("📂 Database path (AppData): %s", appDataPath)
		return appDataPath
	}

	// Priority 7: Current working directory (fallback - will create new)
	log.Printf("📂 Database path (fallback): %s", localPath)
	return localPath
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

	// Platform data directory (production fallback) — %APPDATA%\AsymmFlow on
	// Windows, ~/.local/share/AsymmFlow elsewhere (3-PLAT).
	if dataDir := appDataDirPath(); dataDir != "" {
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
