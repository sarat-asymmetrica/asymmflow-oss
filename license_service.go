// ============================================================================
// LICENSE SERVICE - Key-Based Installer System
//
// MISSION: Generate and validate license keys that bind roles to PCs
//
// KEY FORMAT: PH-{ROLE}-{6-char-hash}
//   - PH-ADM-A1B2C3 = Admin
//   - PH-MGR-X4Y5Z6 = Manager
//   - PH-SLS-M7N8O9 = Sales
//   - PH-OPS-P0Q1R2 = Operations
//   - PH-STF-S1T2U3 = Staff
//
// ARCHITECTURE:
//   1. Key binds to PC (device hash)
//   2. Role determines feature access
//   3. Butler AI queries restricted by role
//   4. License persists in SQLite
// ============================================================================

package main

import (
	crypto_rand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// BE-7 FIX: License rate limiting to prevent brute force attacks
// Limits to 10 attempts per minute per IP/device
var (
	licenseAttempts     = make(map[string][]time.Time) // deviceHash -> timestamps
	licenseAttemptsMu   sync.Mutex
	maxAttemptsPerMin   = 10
	rateLimitWindowSecs = 60
)

// LicenseKey represents a license key in the database
type LicenseKey struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Key         string     `gorm:"uniqueIndex;size:20" json:"key"`   // PH-ADM-A1B2C3
	Role        string     `gorm:"size:20" json:"role"`              // admin, manager, sales, operations
	DisplayName string     `gorm:"size:100" json:"display_name"`     // Employee name (e.g. "Rishu", "Casey")
	DeviceHash  string     `gorm:"size:64;index" json:"device_hash"` // Bound to this PC
	Activated   bool       `gorm:"default:false" json:"activated"`
	ActivatedAt *time.Time `json:"activated_at"`
	IssuedAt    time.Time  `gorm:"autoCreateTime" json:"issued_at"`
	ExpiresAt   *time.Time `json:"expires_at"`                 // Optional expiry
	Notes       string     `gorm:"size:255" json:"notes"`      // Admin notes
	CreatedBy   string     `gorm:"size:100" json:"created_by"` // Who generated this key
}

// LicenseActivationResult is returned after license activation
type LicenseActivationResult struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	Role        string   `json:"role"`
	DisplayName string   `json:"display_name"`
	Permissions []string `json:"permissions"`
	DeviceHash  string   `json:"device_hash"`
}

// LicenseValidationResult is returned when checking current license
type LicenseValidationResult struct {
	Valid       bool     `json:"valid"`
	Role        string   `json:"role"`
	Key         string   `json:"key"`
	DisplayName string   `json:"display_name"`
	Permissions []string `json:"permissions"`
	ExpiresAt   *string  `json:"expires_at"`
}

// licenseKeyPrefix is the leading token of every license key
// ({PREFIX}-{ROLE}-{6-char}). It is overlay configuration (Mission D) with a
// built-in "PH" default, so existing activations keep validating unchanged.
func licenseKeyPrefix() string {
	return activeOverlay.LicenseKeyPrefixOrDefault()
}

// Role prefix mapping
var rolePrefixes = map[string]string{
	"admin":      "ADM",
	"manager":    "MGR",
	"sales":      "SLS",
	"operations": "OPS",
	"staff":      "STF",
	"developer":  "DEV", // Developer/test keys with full admin access
}

// Reverse mapping: prefix -> role
var prefixToRole = map[string]string{
	"ADM": "admin",
	"MGR": "manager",
	"SLS": "sales",
	"OPS": "operations",
	"STF": "staff",
	"DEV": "admin", // DEV keys get admin permissions
}

// Master key — optional developer override, loaded from the environment and
// DISABLED by default (gated by developerMasterKeyEnabled / ENABLE_DEVELOPER_MASTER_KEY).
// Never hardcode a usable value here: a public default key is a backdoor that
// ships inside every binary. Empty by default means "no master key".
var masterKey = os.Getenv("ASYMMFLOW_MASTER_KEY")

// Role permissions mapping
var rolePermissions = map[string][]string{
	"admin":     {"*"}, // Full access
	"developer": {"*"}, // Full access for testing (DEV keys)
	"manager": {
		"dashboard:view",
		"offers:view", "offers:create", "offers:edit",
		"orders:view", "orders:create", "orders:edit", "orders:update",
		"finance:view", "finance:create",
		"expenses:view", "expenses:create", "expenses:update",
		"payroll:view", "payroll:create", "payroll:update", "payroll:approve",
		"invoices:view", "invoices:create", "invoices:update", "invoices:approve", // Invoice management
		"payments:view", "payments:create", "payments:update", "payments:record", // Payment management (NO payments:delete - admin only)
		"customers:view", "customers:edit",
		"suppliers:view", "suppliers:create", "suppliers:edit", "suppliers:update",
		"po:view", "po:create", "po:update", "po:approve", // Full PO management
		"delivery_notes:view", "delivery_notes:create", "delivery_notes:update", "delivery_notes:dispatch", "delivery_notes:confirm",
		"projects:view", "projects:create", "projects:update",
		"tasks:view", "tasks:create", "tasks:update",
		"notifications:view", "notifications:update",
		"hr:view", "hr:create", "hr:update",
		"grn:view", "grn:create",
		"reports:view", "reports:generate",
		"settings:view",
		"data:import",
		"documents:create", "documents:view", "documents:classify", // OCR features
		"intelligence:chat", "intelligence:reports",
		"users:view",
	},
	"sales": {
		"dashboard:view",
		"offers:view", "offers:create", "offers:edit",
		"orders:view", "orders:create", "orders:update",
		"invoices:view", "invoices:create", "invoices:update", // Can create and update draft customer invoices from won orders
		"payments:view", // Can VIEW payments but NOT create (manager/admin only)
		"projects:view", "projects:create", "projects:update",
		"tasks:view", "tasks:create", "tasks:update",
		"notifications:view", "notifications:update",
		"customers:view", "customers:edit",
		"suppliers:view", "suppliers:create",
		"po:view", "po:create", // Sales can raise supplier POs tied to their commercial flow
		"delivery_notes:view", "delivery_notes:create", "delivery_notes:update", // Sales can prepare DNs from won orders
		"rfq:view", "rfq:create",
		"reports:view",
		"documents:create", "documents:view", "documents:classify", // OCR features for scanning customer docs
		"intelligence:chat",
	},
	"operations": {
		"dashboard:view",
		"orders:view", "orders:update",
		"invoices:view", "invoices:create", "invoices:update", // Can create/update draft customer invoices after delivery/order processing
		"suppliers:view", "suppliers:create", "suppliers:edit", "suppliers:update",
		"projects:view", "projects:create", "projects:update",
		"tasks:view", "tasks:create", "tasks:update",
		"notifications:view", "notifications:update",
		"grn:view", "grn:create",
		"po:view", "po:create", "po:update", "po:send", // Full PO lifecycle
		"delivery_notes:view", "delivery_notes:create", "delivery_notes:update", "delivery_notes:dispatch", "delivery_notes:confirm",
		"reports:view",
		"settings:view",
		"documents:create", "documents:view", "documents:classify", // OCR features for scanning supplier docs
		"intelligence:chat",
	},
	"staff": {
		"dashboard:view",
		"customers:view",
		"suppliers:view",
		"invoices:view",
		"orders:view",
		"payments:view",
		"offers:view",
		"projects:view",
		"tasks:view", "tasks:create", "tasks:update",
		"notifications:view", "notifications:update",
		"reports:view",
		"settings:view",
		"documents:create", "documents:view", "documents:classify",
		"intelligence:chat",
	},
}

func (a *App) developerMasterKeyEnabled() bool {
	if a != nil && a.config != nil {
		return a.config.App.EnableDeveloperMasterKey
	}
	return getEnvBool("ENABLE_DEVELOPER_MASTER_KEY", false)
}

// GenerateLicenseKey creates a new license key for a role (admin only)
func (a *App) GenerateLicenseKey(role, notes, createdBy string) (string, error) {
	return a.licenseService().GenerateLicenseKey(role, notes, createdBy)
}

func generateLicenseKey(a *App, role, notes, createdBy string) (string, error) {
	if err := a.requirePermission("licenses:manage"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	role = strings.ToLower(role)
	prefix, ok := rolePrefixes[role]
	if !ok {
		return "", fmt.Errorf("invalid role: %s (valid: admin, manager, sales, operations, staff)", role)
	}

	// Generate cryptographically random key instead of deterministic hash
	randomBytes := make([]byte, 3) // 3 bytes = 6 hex chars
	if _, err := crypto_rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random key: %v", err)
	}
	hashStr := strings.ToUpper(hex.EncodeToString(randomBytes))

	key := fmt.Sprintf("%s-%s-%s", licenseKeyPrefix(), prefix, hashStr)

	// Store in database
	license := LicenseKey{
		Key:       key,
		Role:      role,
		Activated: false,
		Notes:     notes,
		CreatedBy: createdBy,
	}

	if err := a.db.Create(&license).Error; err != nil {
		return "", fmt.Errorf("failed to store license key: %w", err)
	}

	log.Printf("Generated license key: %s-%s-****** (role: %s)", licenseKeyPrefix(), prefix, role)
	return key, nil
}

// GenerateBatchLicenseKeys creates multiple license keys at once (admin only)
func (a *App) GenerateBatchLicenseKeys(role string, count int, notes, createdBy string) ([]string, error) {
	return a.licenseService().GenerateBatchLicenseKeys(role, count, notes, createdBy)
}

func generateBatchLicenseKeys(a *App, role string, count int, notes, createdBy string) ([]string, error) {
	if err := a.requirePermission("licenses:manage"); err != nil {
		return nil, err
	}
	if count < 1 || count > 100 {
		return nil, fmt.Errorf("count must be between 1 and 100")
	}

	keys := make([]string, 0, count)
	for i := 0; i < count; i++ {
		key, err := a.GenerateLicenseKey(role, notes, createdBy)
		if err != nil {
			return keys, fmt.Errorf("failed at key %d: %w", i+1, err)
		}
		keys = append(keys, key)
		// Small delay to ensure unique timestamps
		time.Sleep(time.Millisecond)
	}

	return keys, nil
}

// ActivateLicense activates a license key and binds it to this PC
func (a *App) ActivateLicense(key string) (LicenseActivationResult, error) {
	return a.licenseService().ActivateLicense(key)
}

func activateLicense(a *App, key string) (LicenseActivationResult, error) {
	key = strings.ToUpper(strings.TrimSpace(key))

	if a.db == nil {
		return LicenseActivationResult{Success: false, Message: "Database not ready. Please restart the application."}, nil
	}

	// BE-7 FIX: Rate limiting - prevent brute force attacks on license keys
	deviceHash := a.getDeviceHash()
	licenseAttemptsMu.Lock()
	now := time.Now()
	windowStart := now.Add(-time.Duration(rateLimitWindowSecs) * time.Second)

	// Clean old attempts outside window
	attempts := licenseAttempts[deviceHash]
	validAttempts := make([]time.Time, 0, len(attempts))
	for _, t := range attempts {
		if t.After(windowStart) {
			validAttempts = append(validAttempts, t)
		}
	}

	// Check if rate limited
	if len(validAttempts) >= maxAttemptsPerMin {
		licenseAttemptsMu.Unlock()
		log.Printf("🚫 License activation rate limited for device: %s (%d attempts in last minute)", deviceHash[:16]+"...", len(validAttempts))
		return LicenseActivationResult{
			Success: false,
			Message: "Too many activation attempts. Please wait 1 minute before trying again.",
		}, nil
	}

	// Record this attempt
	licenseAttempts[deviceHash] = append(validAttempts, now)
	licenseAttemptsMu.Unlock()

	if masterKey != "" && key == masterKey {
		if !a.developerMasterKeyEnabled() {
			log.Printf("🚫 Master key activation blocked on this build/device")
			return LicenseActivationResult{
				Success: false,
				Message: "Invalid license key. Please contact your administrator for a valid key.",
			}, nil
		}

		// Master key — reusable, never device-bound, works across rebuilds
		// Return success immediately; persist to DB in background (startup AutoMigrate may hold the connection)
		go func() {
			err := a.db.Transaction(func(tx *gorm.DB) error {
				var existing LicenseKey
				if err := tx.Where("key = ?", masterKey).First(&existing).Error; err != nil {
					now := time.Now()
					return tx.Create(&LicenseKey{
						Key:         masterKey,
						Role:        "admin",
						DisplayName: "Developer",
						DeviceHash:  deviceHash,
						Activated:   true,
						ActivatedAt: &now,
						Notes:       "Reusable master key",
						CreatedBy:   "system",
					}).Error
				}
				if existing.DeviceHash != "" && existing.DeviceHash != deviceHash {
					log.Printf("CRITICAL: Master key device transfer from %s... to %s...", existing.DeviceHash[:16], deviceHash[:16])
				}
				return tx.Model(&existing).Updates(map[string]any{
					"device_hash": deviceHash,
					"activated":   true,
				}).Error
			})
			if err != nil {
				log.Printf("Warning: Master key DB persist failed: %v (will retry on next startup)", err)
			} else {
				log.Printf("Master key persisted to DB for device: %s...", deviceHash[:16])
			}
		}()

		log.Printf("Master key activated on device: %s...", deviceHash[:16])
		return LicenseActivationResult{
			Success:     true,
			Message:     "Master key activated (admin access)",
			Role:        "admin",
			DisplayName: "Developer",
			Permissions: rolePermissions["admin"],
			DeviceHash:  deviceHash,
		}, nil
	}

	// Validate key format: {PREFIX}-XXX-YYYYYY (overlay-configured prefix,
	// built-in "PH" → the historic 13-char PH-ADM-A1B2C3 shape, unchanged).
	wantPrefix := licenseKeyPrefix() + "-"
	if !strings.HasPrefix(key, wantPrefix) || len(key) != len(wantPrefix)+10 {
		return LicenseActivationResult{
			Success: false,
			Message: fmt.Sprintf("Invalid key format. Keys look like: %sADM-A1B2C3", wantPrefix),
		}, nil
	}

	// Check if this device already has an active license
	var existingLicense LicenseKey
	if err := a.db.Where("device_hash = ? AND activated = 1", deviceHash).First(&existingLicense).Error; err == nil {
		return LicenseActivationResult{
			Success:     true,
			Message:     "This device is already licensed",
			Role:        existingLicense.Role,
			DisplayName: existingLicense.DisplayName,
			Permissions: rolePermissions[existingLicense.Role],
			DeviceHash:  deviceHash,
		}, nil
	}

	// Use transaction to prevent race condition on concurrent activation
	tx := a.db.Begin()
	if tx.Error != nil {
		return LicenseActivationResult{Success: false, Message: "Database error"}, tx.Error
	}

	// Find the license key - SECURITY: Keys must be pre-generated by admin
	// Auto-creation REMOVED to prevent bypass attacks (P0 security fix)
	var license LicenseKey
	if err := tx.Where("key = ?", key).First(&license).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			log.Printf("🔒 License key not found: %s (auto-creation disabled for security)", key)
			return LicenseActivationResult{
				Success: false,
				Message: "Invalid license key. Please contact your administrator for a valid key.",
			}, nil
		}
		return LicenseActivationResult{Success: false, Message: "Database error"}, err
	}

	// Check if already activated on another device
	if license.Activated && license.DeviceHash != deviceHash {
		tx.Rollback()
		return LicenseActivationResult{
			Success: false,
			Message: "This license key is already activated on another device.",
		}, nil
	}

	// Check expiry
	if license.ExpiresAt != nil && license.ExpiresAt.Before(time.Now()) {
		tx.Rollback()
		return LicenseActivationResult{
			Success: false,
			Message: "This license key has expired.",
		}, nil
	}

	// Activate the license within transaction
	now = time.Now()
	license.Activated = true
	license.ActivatedAt = &now
	license.DeviceHash = deviceHash

	if err := tx.Save(&license).Error; err != nil {
		tx.Rollback()
		return LicenseActivationResult{Success: false, Message: "Failed to activate license"}, err
	}

	if err := tx.Commit().Error; err != nil {
		return LicenseActivationResult{Success: false, Message: "Failed to activate license"}, err
	}

	log.Printf("License activated: %s (role: %s) on device: %s", key, license.Role, deviceHash[:16]+"...")

	// NOTE: First-run sync is handled by the frontend (LicenseActivationScreen.svelte)
	// with proper UI feedback. Do NOT call it here to avoid race conditions
	// and duplicate sync attempts that could cause data loss.

	return LicenseActivationResult{
		Success:     true,
		Message:     fmt.Sprintf("License activated successfully! Role: %s", strings.Title(license.Role)),
		Role:        license.Role,
		DisplayName: license.DisplayName,
		Permissions: rolePermissions[license.Role],
		DeviceHash:  deviceHash,
	}, nil
}

// ValidateLicense checks if this device has a valid license
func (a *App) ValidateLicense() (LicenseValidationResult, error) {
	return a.licenseService().ValidateLicense()
}

func validateLicense(a *App) (LicenseValidationResult, error) {
	log.Println("🔑 ValidateLicense: Starting...")

	// Check database is initialized
	if a.db == nil {
		log.Println("🔑 ValidateLicense: ERROR - Database not initialized!")
		return LicenseValidationResult{Valid: false}, fmt.Errorf("database not initialized")
	}

	log.Println("🔑 ValidateLicense: Getting device hash...")
	deviceHash := a.getDeviceHash()
	log.Printf("🔑 ValidateLicense: Device hash obtained (%s...)", deviceHash[:16])

	log.Println("🔑 ValidateLicense: Querying database for license...")
	var license LicenseKey

	// First: check for master key (device-hash-independent — works across all launch methods)
	if a.developerMasterKeyEnabled() {
		if err := a.db.Where("key = ? AND activated = 1", masterKey).First(&license).Error; err == nil {
			// Master key found and activated — always valid regardless of device hash
			// Silently update device hash if changed (different launch method)
			if license.DeviceHash != deviceHash {
				a.db.Model(&license).Update("device_hash", deviceHash)
			}
			log.Printf("🔑 ValidateLicense: Master key valid (role: %s)", license.Role)
		} else if err := a.db.Where("device_hash = ? AND activated = 1", deviceHash).First(&license).Error; err != nil {
			// No master key, no device-bound license
			if err == gorm.ErrRecordNotFound {
				log.Printf("🔑 ValidateLicense: No active license found for device hash: %s", deviceHash[:16])
				return LicenseValidationResult{Valid: false}, nil
			}
			log.Printf("🔑 ValidateLicense: Database error: %v", err)
			return LicenseValidationResult{Valid: false}, err
		}
	} else if err := a.db.Where("device_hash = ? AND activated = 1", deviceHash).First(&license).Error; err != nil {
		// No master key, no device-bound license
		if err == gorm.ErrRecordNotFound {
			log.Printf("🔑 ValidateLicense: No active license found for device hash: %s", deviceHash[:16])
			return LicenseValidationResult{Valid: false}, nil
		}
		log.Printf("🔑 ValidateLicense: Database error: %v", err)
		return LicenseValidationResult{Valid: false}, err
	}
	log.Printf("🔑 ValidateLicense: Found license: %s (role: %s)", license.Key, license.Role)

	// Check expiry
	if license.ExpiresAt != nil && license.ExpiresAt.Before(time.Now()) {
		return LicenseValidationResult{
			Valid: false,
			Role:  license.Role,
			Key:   license.Key,
		}, nil
	}

	var expiresAt *string
	if license.ExpiresAt != nil {
		exp := license.ExpiresAt.Format("2006-01-02")
		expiresAt = &exp
	}

	return LicenseValidationResult{
		Valid:       true,
		Role:        license.Role,
		Key:         license.Key,
		DisplayName: license.DisplayName,
		Permissions: rolePermissions[license.Role],
		ExpiresAt:   expiresAt,
	}, nil
}

// GetLicenseRole returns the current device's license role (for Butler AI restrictions)
func (a *App) GetLicenseRole() string {
	return a.licenseService().GetLicenseRole()
}

func getLicenseRole(a *App) string {
	result, err := a.ValidateLicense()
	if err != nil || !result.Valid {
		return "" // No valid license
	}
	return result.Role
}

// HasLicensePermission checks if the current license has a specific permission
func (a *App) HasLicensePermission(permission string) bool {
	return a.licenseService().HasLicensePermission(permission)
}

func hasLicensePermission(a *App, permission string) bool {
	result, err := a.ValidateLicense()
	if err != nil || !result.Valid {
		return false
	}

	for _, perm := range result.Permissions {
		if permissionGranted(perm, permission) {
			return true
		}
	}
	return false
}

// ListLicenseKeys returns all license keys (admin only)
func (a *App) ListLicenseKeys() ([]LicenseKey, error) {
	return a.licenseService().ListLicenseKeys()
}

func listLicenseKeys(a *App) ([]LicenseKey, error) {
	if err := a.requirePermission("licenses:manage"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var licenses []LicenseKey
	if err := a.db.Order("issued_at DESC").Find(&licenses).Error; err != nil {
		return nil, err
	}
	return licenses, nil
}

func (a *App) UpdateLicenseDisplayName(key, displayName string) (LicenseKey, error) {
	return a.licenseService().UpdateLicenseDisplayName(key, displayName)
}

func updateLicenseDisplayName(a *App, key, displayName string) (LicenseKey, error) {
	if err := a.requirePermission("licenses:manage"); err != nil {
		return LicenseKey{}, err
	}
	if a.db == nil {
		return LicenseKey{}, fmt.Errorf("database not initialized")
	}

	key = strings.ToUpper(strings.TrimSpace(key))
	displayName = strings.TrimSpace(displayName)
	if key == "" {
		return LicenseKey{}, fmt.Errorf("license key is required")
	}
	if displayName == "" {
		return LicenseKey{}, fmt.Errorf("display name is required")
	}

	var license LicenseKey
	if err := a.db.Where("key = ?", key).First(&license).Error; err != nil {
		return LicenseKey{}, fmt.Errorf("license key not found: %w", err)
	}

	license.DisplayName = displayName
	license.Notes = fmt.Sprintf("Assigned to %s", displayName)
	if err := a.db.Save(&license).Error; err != nil {
		return LicenseKey{}, fmt.Errorf("failed to update license display name: %w", err)
	}

	return license, nil
}

// RevokeLicense deactivates a license key (admin only)
func (a *App) RevokeLicense(key string) error {
	return a.licenseService().RevokeLicense(key)
}

func revokeLicense(a *App, key string) error {
	if err := a.requirePermission("licenses:manage"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	// Master key cannot be revoked (it's hardcoded and would just re-activate)
	if masterKey != "" && strings.ToUpper(strings.TrimSpace(key)) == masterKey {
		return fmt.Errorf("master key cannot be revoked")
	}
	result := a.db.Model(&LicenseKey{}).Where("key = ?", key).Updates(map[string]any{
		"activated":    false,
		"device_hash":  "",
		"activated_at": nil,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("license key not found: %s", key)
	}
	log.Printf("License revoked: %s", key)
	return nil
}

// getDeviceHash generates a unique hash for this PC (reuses GetMachineID from device_service.go)
func (a *App) getDeviceHash() string {
	log.Println("🔐 getDeviceHash: Calling GetMachineID...")
	hash := GetMachineID()
	log.Printf("🔐 getDeviceHash: Complete, hash=%s...", hash[:16])
	return hash
}

// EnsureLicenseTableExists creates the license_keys table if it doesn't exist.
// Mission I (I-11): bound to the frontend — DDL is gated; startup paths use
// the internal ensureLicenseTableExists.
func (a *App) EnsureLicenseTableExists() error {
	if err := a.requirePermission("licenses:manage"); err != nil {
		return err
	}
	return a.licenseService().EnsureLicenseTableExists()
}

func ensureLicenseTableExists(a *App) error {
	return a.db.AutoMigrate(&LicenseKey{})
}

// SeedLicenseKeys generates 10 keys per role if fewer than 10 unactivated keys exist per role.
// Called on startup (via the internal seedLicenseKeys) to ensure the org always
// has enough keys available. Mission I (I-11): the bound method mints
// authentication credentials, so it is gated — a frontend caller could
// previously create license keys with no RBAC check at all.
func (a *App) SeedLicenseKeys() error {
	if err := a.requirePermission("licenses:manage"); err != nil {
		return err
	}
	return a.licenseService().SeedLicenseKeys()
}

func seedLicenseKeys(a *App) error {
	log.Println("🔑 SeedLicenseKeys: Starting...")

	roles := []string{"admin", "manager", "sales", "operations", "staff", "developer"}
	targetPerRole := 10
	var totalSeeded int

	for _, role := range roles {
		prefix, ok := rolePrefixes[role]
		if !ok {
			log.Printf("🔑 SeedLicenseKeys: No prefix for role '%s', skipping", role)
			continue
		}

		// Count existing unactivated keys for this role
		var existingCount int64
		if err := a.db.Model(&LicenseKey{}).Where("role = ? AND activated = 0", role).Count(&existingCount).Error; err != nil {
			log.Printf("⚠️ SeedLicenseKeys: Failed to count keys for role '%s': %v", role, err)
			continue
		}

		needed := targetPerRole - int(existingCount)
		log.Printf("🔑 SeedLicenseKeys: role=%s prefix=%s existing=%d needed=%d", role, prefix, existingCount, needed)
		if needed <= 0 {
			continue
		}

		for i := 0; i < needed; i++ {
			// Generate cryptographically random key instead of deterministic hash
			randomBytes := make([]byte, 3) // 3 bytes = 6 hex chars
			if _, err := crypto_rand.Read(randomBytes); err != nil {
				log.Printf("Failed to generate random key: %v", err)
				continue
			}
			hashStr := strings.ToUpper(hex.EncodeToString(randomBytes))
			key := fmt.Sprintf("%s-%s-%s", licenseKeyPrefix(), prefix, hashStr)

			// Check for collision and retry with new random bytes
			var collision int64
			a.db.Model(&LicenseKey{}).Where("key = ?", key).Count(&collision)
			if collision > 0 {
				if _, err := crypto_rand.Read(randomBytes); err != nil {
					log.Printf("Failed to generate random key on retry: %v", err)
					continue
				}
				hashStr = strings.ToUpper(hex.EncodeToString(randomBytes))
				key = fmt.Sprintf("%s-%s-%s", licenseKeyPrefix(), prefix, hashStr)
			}

			license := LicenseKey{
				Key:       key,
				Role:      role,
				Activated: false,
				Notes:     "Pre-generated for Acme Instrumentation org",
				CreatedBy: "system",
			}

			if err := a.db.Create(&license).Error; err != nil {
				log.Printf("⚠️ Failed to seed key %s: %v", key, err)
				continue
			}
			log.Printf("🔑 Created key for role: %s", role)
			totalSeeded++
		}
	}

	if totalSeeded > 0 {
		log.Printf("🔑 Seeded %d new license keys", totalSeeded)
	} else {
		log.Println("🔑 SeedLicenseKeys: No new keys needed (all roles have 10+ unactivated keys)")
	}

	// SECURITY FIX: Only log key counts per role, never actual key values
	var availableKeys []LicenseKey
	a.db.Where("activated = 0").Order("role, key").Find(&availableKeys)
	roleCounts := make(map[string]int)
	for _, k := range availableKeys {
		roleCounts[k.Role]++
	}
	if len(availableKeys) > 0 {
		log.Printf("🔑 Available license keys: %d total", len(availableKeys))
		for role, count := range roleCounts {
			log.Printf("   %s: %d keys available", role, count)
		}
	} else {
		log.Println("🔑 No unactivated license keys available")
	}

	return nil
}

type employeeLicenseSpec struct {
	Role        string
	DisplayName string
	Key         string
	Notes       string
}

func phTradingEmployeeLicenseSpecs() []employeeLicenseSpec {
	return []employeeLicenseSpec{
		{Role: "admin", DisplayName: "Admin One", Key: "PH-ADM-4A0185", Notes: "Example admin key"},
		{Role: "admin", DisplayName: "Admin Two", Key: "PH-ADM-E91CC2", Notes: "Example admin key"},
		{Role: "admin", DisplayName: "Admin Three", Key: "PH-ADM-8F2A41", Notes: "Example admin key"},
		{Role: "admin", DisplayName: "Admin Four", Key: "PH-ADM-E2E4C2", Notes: "Example admin key"},
		{Role: "admin", DisplayName: "Admin Five", Key: "PH-ADM-CD4680", Notes: "Example admin key"},
		{Role: "manager", DisplayName: "Manager One", Key: "PH-MGR-CE8180", Notes: "Example manager key"},
		{Role: "sales", DisplayName: "Sales One", Key: "PH-SLS-9A70F9", Notes: "Example sales key"},
		{Role: "sales", DisplayName: "Sales Two", Key: "PH-SLS-62C4D4", Notes: "Example sales key"},
		{Role: "sales", DisplayName: "Sales Three", Key: "PH-SLS-7749F3", Notes: "Example sales key"},
		{Role: "sales", DisplayName: "Sales Support", Key: "PH-SLS-3C37C1", Notes: "Example sales support key"},
		{Role: "operations", DisplayName: "Operations One", Key: "PH-OPS-206C69", Notes: "Example operations key"},
	}
}

func phTradingRoleTestLicenseSpecs() []employeeLicenseSpec {
	return []employeeLicenseSpec{
		{Role: "admin", DisplayName: "Admin Test", Key: "PH-ADM-6C9BF2", Notes: "Acme Instrumentation test key for admin role"},
		{Role: "manager", DisplayName: "Manager Test", Key: "PH-MGR-49F07C", Notes: "Acme Instrumentation test key for manager role"},
		{Role: "sales", DisplayName: "Sales Test", Key: "PH-SLS-B4AA10", Notes: "Acme Instrumentation test key for sales role"},
		{Role: "operations", DisplayName: "Operations Test", Key: "PH-OPS-44490F", Notes: "Acme Instrumentation test key for operations role"},
		{Role: "staff", DisplayName: "Staff Test", Key: "PH-STF-B47E0B", Notes: "Acme Instrumentation test key for staff role"},
	}
}

func phTradingNamedLicenseSpecs() []employeeLicenseSpec {
	specs := phTradingEmployeeLicenseSpecs()
	specs = append(specs, phTradingRoleTestLicenseSpecs()...)
	return specs
}

func deploymentLicenseActivationFlushStamp() string {
	if stamp := strings.TrimSpace(os.Getenv("ASYMMFLOW_LICENSE_FLUSH_STAMP")); stamp != "" {
		return stamp
	}
	return ""
}

// ApplyDeploymentLicenseActivationFlush resets existing device bindings once per
// explicit deployment stamp. Database reseeding preserves activations by default;
// normal update installers must not make users enter their keys again.
func (a *App) ApplyDeploymentLicenseActivationFlush() error {
	// Mission I (I-11): bound method resets device bindings — gated; startup
	// uses the internal applyDeploymentLicenseActivationFlush.
	if err := a.requirePermission("licenses:manage"); err != nil {
		return err
	}
	return a.licenseService().ApplyDeploymentLicenseActivationFlush()
}

func applyDeploymentLicenseActivationFlush(a *App) error {
	if a == nil || a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	stamp := deploymentLicenseActivationFlushStamp()
	if stamp == "" {
		return nil
	}

	const settingKey = "deployment_license_activation_flush_stamp"
	var existing Setting
	if err := a.db.Unscoped().Where("key = ?", settingKey).First(&existing).Error; err == nil {
		if strings.TrimSpace(existing.Value) == stamp {
			return nil
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return a.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&LicenseKey{}).
			Where("activated = ? OR COALESCE(device_hash, '') <> '' OR activated_at IS NOT NULL", true).
			Updates(map[string]any{
				"activated":    false,
				"device_hash":  "",
				"activated_at": nil,
			})
		if result.Error != nil {
			return result.Error
		}

		updates := map[string]any{
			"value":        stamp,
			"category":     "deployment",
			"description":  "Last deployment stamp that reset local license activations",
			"is_encrypted": false,
			"deleted_at":   nil,
		}
		if existing.ID != "" {
			if err := tx.Unscoped().Model(&existing).Updates(updates).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Create(&Setting{
				Key:         settingKey,
				Value:       stamp,
				Category:    "deployment",
				Description: "Last deployment stamp that reset local license activations",
				IsEncrypted: false,
			}).Error; err != nil {
				return err
			}
		}

		log.Printf("🔑 Deployment license activation flush applied for stamp %s (%d bindings reset)", stamp, result.RowsAffected)
		return nil
	})
}

// SeedEmployeeKeys creates specific license keys linked to named employees.
// These are the primary keys given to Acme Instrumentation staff.
// Called on startup after SeedLicenseKeys.
func (a *App) SeedEmployeeKeys() error {
	// Mission I (I-11): mints named credentials — gated; startup uses the
	// internal seedEmployeeKeys.
	if err := a.requirePermission("licenses:manage"); err != nil {
		return err
	}
	return a.licenseService().SeedEmployeeKeys()
}

func seedEmployeeKeys(a *App) error {
	log.Println("🔑 SeedEmployeeKeys: Assigning keys to employees...")

	var assigned int
	for _, emp := range phTradingNamedLicenseSpecs() {
		if strings.TrimSpace(emp.Key) != "" {
			upserted, err := a.ensureNamedEmployeeKey(emp)
			if err != nil {
				log.Printf("⚠️ Failed to ensure key for %s: %v", emp.DisplayName, err)
				continue
			}
			if upserted {
				assigned++
			}
			continue
		}

		// Check if this employee already has a key assigned
		var existing LicenseKey
		if err := a.db.Where("display_name = ?", emp.DisplayName).First(&existing).Error; err == nil {
			if strings.TrimSpace(existing.Notes) == "" && strings.TrimSpace(emp.Notes) != "" {
				_ = a.db.Model(&existing).Update("notes", emp.Notes).Error
			}
			// Already assigned
			continue
		}

		// Find an unactivated, unnamed key for this role
		var availableKey LicenseKey
		if err := a.db.Where("role = ? AND activated = 0 AND (display_name = '' OR display_name IS NULL)", emp.Role).
			First(&availableKey).Error; err != nil {
			log.Printf("⚠️ No available %s key for %s - generating new one", emp.Role, emp.DisplayName)
			// Generate a cryptographically random key for this employee
			prefix := rolePrefixes[emp.Role]
			randomBytes := make([]byte, 3)
			if _, err := crypto_rand.Read(randomBytes); err != nil {
				log.Printf("⚠️ Failed to generate random key for %s: %v", emp.DisplayName, err)
				continue
			}
			hashStr := strings.ToUpper(hex.EncodeToString(randomBytes))
			key := fmt.Sprintf("PH-%s-%s", prefix, hashStr)

			newKey := LicenseKey{
				Key:         key,
				Role:        emp.Role,
				DisplayName: emp.DisplayName,
				Activated:   false,
				Notes:       firstNonEmptyString(emp.Notes, fmt.Sprintf("Assigned to %s", emp.DisplayName)),
				CreatedBy:   "system",
			}
			if err := a.db.Create(&newKey).Error; err != nil {
				log.Printf("⚠️ Failed to create key for %s: %v", emp.DisplayName, err)
				continue
			}
			log.Printf("🔑 Created & assigned key to %s (%s)", emp.DisplayName, emp.Role)
			assigned++
			continue
		}

		// Assign the existing key to this employee
		availableKey.DisplayName = emp.DisplayName
		availableKey.Notes = firstNonEmptyString(emp.Notes, fmt.Sprintf("Assigned to %s", emp.DisplayName))
		if err := a.db.Save(&availableKey).Error; err != nil {
			log.Printf("⚠️ Failed to assign key to %s: %v", emp.DisplayName, err)
			continue
		}
		log.Printf("🔑 Assigned key to %s (%s)", emp.DisplayName, emp.Role)
		assigned++
	}

	if assigned > 0 {
		log.Printf("🔑 Assigned %d employee keys", assigned)
	}

	// SECURITY FIX: Only log employee assignment counts, never actual keys
	var employeeKeys []LicenseKey
	a.db.Where("display_name != '' AND display_name IS NOT NULL").Order("role, display_name").Find(&employeeKeys)
	if len(employeeKeys) > 0 {
		activatedCount := 0
		for _, k := range employeeKeys {
			if k.Activated {
				activatedCount++
			}
		}
		log.Printf("🔑 Employee keys: %d assigned (%d activated, %d available)",
			len(employeeKeys), activatedCount, len(employeeKeys)-activatedCount)
	}

	return nil
}

func (a *App) ensureNamedEmployeeKey(emp employeeLicenseSpec) (bool, error) {
	if a == nil || a.db == nil {
		return false, fmt.Errorf("database not initialized")
	}

	key := strings.ToUpper(strings.TrimSpace(emp.Key))
	if key == "" {
		return false, nil
	}

	updates := map[string]any{
		"role":         emp.Role,
		"display_name": emp.DisplayName,
		"notes":        firstNonEmptyString(emp.Notes, fmt.Sprintf("Assigned to %s", emp.DisplayName)),
		"created_by":   "system",
	}

	var byKey LicenseKey
	if err := a.db.Where("key = ?", key).First(&byKey).Error; err == nil {
		return false, a.db.Model(&byKey).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	var byName LicenseKey
	if err := a.db.Where("role = ? AND display_name = ?", emp.Role, emp.DisplayName).
		Order("activated DESC, id ASC").
		First(&byName).Error; err == nil {
		if byName.Activated {
			record := LicenseKey{
				Key:         key,
				Role:        emp.Role,
				DisplayName: emp.DisplayName,
				Activated:   false,
				Notes:       firstNonEmptyString(emp.Notes, fmt.Sprintf("Assigned to %s", emp.DisplayName)),
				CreatedBy:   "system",
			}
			return true, a.db.Create(&record).Error
		}
		updates["key"] = key
		updates["activated"] = false
		updates["activated_at"] = nil
		updates["device_hash"] = ""
		return true, a.db.Model(&byName).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	record := LicenseKey{
		Key:         key,
		Role:        emp.Role,
		DisplayName: emp.DisplayName,
		Activated:   false,
		Notes:       firstNonEmptyString(emp.Notes, fmt.Sprintf("Assigned to %s", emp.DisplayName)),
		CreatedBy:   "system",
	}
	return true, a.db.Create(&record).Error
}

// CheckFirstInstall returns true if no licenses exist (first installation)
func (a *App) CheckFirstInstall() bool {
	return a.licenseService().CheckFirstInstall()
}

func checkFirstInstall(a *App) bool {
	var count int64
	a.db.Model(&LicenseKey{}).Where("activated = 1").Count(&count)
	return count == 0
}

// NeedsLicenseActivation checks if this device needs to activate a license
func (a *App) NeedsLicenseActivation() (bool, error) {
	return a.licenseService().NeedsLicenseActivation()
}

func needsLicenseActivation(a *App) (bool, error) {
	result, err := a.ValidateLicense()
	if err != nil {
		return true, err
	}
	return !result.Valid, nil
}
