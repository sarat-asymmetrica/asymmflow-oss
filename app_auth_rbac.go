package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"ph_holdings_app/pkg/infra/audit"
	"ph_holdings_app/pkg/kernel/actor"
)

func (a *App) SeedDefaultRoles() error {
	// Allow startup/system initialization before any authenticated session exists.
	if a.currentUser != nil || a.currentUserID != "" || a.GetLicenseRole() != "" {
		if err := a.requirePermission("*"); err != nil {
			return err
		}
	}
	roles := []struct {
		ID          string
		Name        string
		DisplayName string
		Description string
		Permissions string
	}{
		{
			ID:          "role_admin",
			Name:        "admin",
			DisplayName: "Administrator",
			Description: "Full system access with user management capabilities",
			Permissions: `["*"]`, // Wildcard = all permissions
		},
		{
			ID:          "role_manager",
			Name:        "manager",
			DisplayName: "Manager",
			Description: "All operations INCLUDING finance (no device/user management)",
			Permissions: `[
				"dashboard:view",
				"customers:view", "customers:edit",
				"suppliers:view", "suppliers:create", "suppliers:edit", "suppliers:update",
				"invoices:view", "invoices:create", "invoices:update", "invoices:approve",
				"orders:view", "orders:create", "orders:edit", "orders:update",
				"payments:view", "payments:create", "payments:update", "payments:record",
				"reports:view", "reports:generate",
				"settings:view",
				"data:import",
				"documents:view", "documents:create", "documents:classify",
				"intelligence:chat",
				"finance:view", "finance:create",
				"expenses:view", "expenses:create", "expenses:update",
				"payroll:view", "payroll:create", "payroll:update", "payroll:approve",
				"projects:view", "projects:create", "projects:update",
				"tasks:view", "tasks:create", "tasks:update",
				"notifications:view", "notifications:update",
				"hr:view", "hr:create", "hr:update",
				"offers:view", "offers:create", "offers:edit",
				"po:view", "po:create", "po:update", "po:approve",
				"delivery_notes:view", "delivery_notes:create", "delivery_notes:update", "delivery_notes:dispatch", "delivery_notes:confirm"
			]`,
		},
		{
			ID:          "role_sales",
			Name:        "sales",
			DisplayName: "Sales",
			Description: "Sales + CRM access, NO finance access",
			Permissions: `[
				"dashboard:view",
				"customers:view", "customers:edit",
				"suppliers:view", "suppliers:create",
				"orders:view", "orders:create", "orders:update",
				"offers:view", "offers:create", "offers:edit",
				"rfq:view", "rfq:create",
				"invoices:view", "invoices:create", "invoices:update",
				"payments:view",
				"projects:view", "projects:create", "projects:update",
				"tasks:view", "tasks:create", "tasks:update",
				"notifications:view", "notifications:update",
				"po:view", "po:create",
				"delivery_notes:view", "delivery_notes:create", "delivery_notes:update",
				"documents:view", "documents:create", "documents:classify",
				"intelligence:chat",
				"reports:view"
			]`,
		},
		{
			ID:          "role_operations",
			Name:        "operations",
			DisplayName: "Operations",
			Description: "Operations only, NO finance access",
			Permissions: `[
				"dashboard:view",
				"suppliers:view", "suppliers:create", "suppliers:edit", "suppliers:update",
				"po:view", "po:create", "po:update", "po:send",
				"projects:view", "projects:create", "projects:update",
				"tasks:view", "tasks:create", "tasks:update",
				"notifications:view", "notifications:update",
				"grn:view", "grn:create",
				"delivery_notes:view", "delivery_notes:create", "delivery_notes:update", "delivery_notes:dispatch", "delivery_notes:confirm",
				"orders:view", "orders:update",
				"invoices:view", "invoices:create", "invoices:update",
				"documents:view", "documents:create", "documents:classify",
				"intelligence:chat",
				"reports:view",
				"settings:view"
			]`,
		},
		{
			ID:          "role_staff",
			Name:        "staff",
			DisplayName: "Staff",
			Description: "Read-only access, NO finance access",
			Permissions: `[
				"dashboard:view",
				"customers:view",
				"suppliers:view",
				"invoices:view",
				"orders:view",
				"payments:view",
				"documents:view", "documents:create", "documents:classify",
				"intelligence:chat",
				"reports:view",
				"settings:view",
				"offers:view",
				"projects:view",
				"tasks:view", "tasks:create", "tasks:update",
				"notifications:view", "notifications:update"
			]`,
		},
	}

	for _, roleData := range roles {
		// Check if role exists by name
		var existingRole Role
		result := a.db.Where("name = ?", roleData.Name).First(&existingRole)

		if result.Error != nil {
			// Role doesn't exist, create it
			newRole := Role{
				Base:        Base{ID: roleData.ID},
				Name:        roleData.Name,
				DisplayName: roleData.DisplayName,
				Description: roleData.Description,
				Permissions: roleData.Permissions,
				IsActive:    true,
				IsSystem:    true,
			}

			if err := a.db.Create(&newRole).Error; err != nil {
				log.Printf("Error creating role %s: %v", roleData.Name, err)
				continue
			}

			log.Printf("Created system role: %s (%s)", roleData.DisplayName, roleData.Name)
		} else {
			// Role exists, update permissions in case they changed
			existingRole.DisplayName = roleData.DisplayName
			existingRole.Description = roleData.Description
			existingRole.Permissions = roleData.Permissions
			existingRole.IsSystem = true
			existingRole.IsActive = true

			if err := a.db.Save(&existingRole).Error; err != nil {
				log.Printf("Error updating role %s: %v", roleData.Name, err)
				continue
			}

			log.Printf("Updated system role: %s (%s)", roleData.DisplayName, roleData.Name)
		}
	}

	if AppLogger != nil {
		AppLogger.Info("RBAC roles seeded", map[string]any{
			"roles_count": len(roles),
			"roles":       []string{"admin", "manager", "sales", "operations", "staff"},
		})
	}

	return nil
}

// ... in CreateUser ...
// a.logAudit(&user.ID, "CREATE", "users", &user.ID, "") // Commented out temporarily or fixed signature match

// ListRoles retrieves all roles
func (a *App) ListRoles() ([]Role, error) {
	// P0 FIX: Admin-only function
	if err := a.requirePermission("users:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var roles []Role
	if err := a.db.Where("is_active = ?", true).Order("id").Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	return roles, nil
}

// GetRole retrieves a single role by ID
func (a *App) GetRole(roleID uint) (*Role, error) {
	if err := a.requirePermission("users:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var role Role
	if err := a.db.First(&role, roleID).Error; err != nil {
		return nil, fmt.Errorf("role not found: %w", err)
	}

	return &role, nil
}

// =============================================================================
// PASSWORD SECURITY HELPERS
// =============================================================================

const (
	bcryptCost            = 12 // Industry standard cost for bcrypt
	minPasswordLength     = 8
	defaultPasswordLength = 16
)

// hashPassword securely hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(bytes), nil
}

// verifyPassword compares a plain password with a bcrypt hash
func verifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// validatePasswordComplexity ensures password meets security requirements
func validatePasswordComplexity(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}

	// Check for at least one number
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	// Check for at least one letter
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)

	if !hasNumber || !hasLetter {
		return fmt.Errorf("password must contain both letters and numbers")
	}

	return nil
}

// generateSecurePassword creates a cryptographically secure random password
func generateSecurePassword(length int) (string, error) {
	if length < minPasswordLength {
		length = defaultPasswordLength
	}

	// Generate random bytes
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure password: %w", err)
	}

	// Convert to base64 and trim to desired length
	password := base64.URLEncoding.EncodeToString(bytes)[:length]

	// Ensure it meets complexity requirements
	if err := validatePasswordComplexity(password); err != nil {
		// Rare edge case - regenerate
		return generateSecurePassword(length)
	}

	return password, nil
}

// ListUsers retrieves all users with their roles
func (a *App) ListUsers() ([]User, error) {
	if err := a.requirePermission("users:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var users []User
	if err := a.db.Where("deleted_at IS NULL").Order("full_name").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	for i := range users {
		if err := a.hydrateUserRole(&users[i]); err != nil {
			return nil, fmt.Errorf("failed to load role for user %s: %w", users[i].ID, err)
		}
	}

	return users, nil
}

// GetUser retrieves a user by ID
func (a *App) GetUser(userID string) (*User, error) {
	if err := a.requirePermission("users:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var user User
	if err := a.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, newError("USER_NOT_FOUND", "User not found", err.Error())
	}
	if err := a.hydrateUserRole(&user); err != nil {
		return nil, newError("USER_ROLE_NOT_FOUND", "User role not found", err.Error())
	}

	return &user, nil
}

// CreateUser creates a new user
func (a *App) CreateUser(username, email, password, fullName, department, jobTitle string, roleID string) (*User, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// SERVER-SIDE PERMISSION CHECK: Require users:create or admin (wildcard)
	if err := a.requirePermission("users:create"); err != nil {
		log.Printf("🔒 CreateUser blocked: %v", err)
		return nil, err
	}

	// Validate required fields
	if username == "" || email == "" {
		return nil, fmt.Errorf("username and email are required")
	}

	// Validate all user inputs for length, XSS, and format
	if GlobalValidator != nil {
		if err := GlobalValidator.ValidateUserInput(username, password, fullName, email); err != nil {
			return nil, fmt.Errorf("input validation failed: %w", err)
		}
		if err := GlobalValidator.ValidateString("Department", department, 100, false); err != nil {
			return nil, fmt.Errorf("input validation failed: %w", err)
		}
		if err := GlobalValidator.ValidateString("Job Title", jobTitle, 100, false); err != nil {
			return nil, fmt.Errorf("input validation failed: %w", err)
		}
	}

	// Validate password complexity
	if password != "" {
		if err := validatePasswordComplexity(password); err != nil {
			return nil, fmt.Errorf("password validation failed: %w", err)
		}
	}

	// Check if role exists
	var role Role
	if err := a.db.Where("id = ?", roleID).First(&role).Error; err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	// Hash password using bcrypt
	passwordHash := ""
	if password != "" {
		hash, err := hashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		passwordHash = hash
	}

	now := time.Now()
	user := User{
		Username:          username,
		Email:             email,
		PasswordHash:      passwordHash,
		FullName:          fullName,
		DisplayName:       strings.Split(fullName, " ")[0], // First name
		Department:        department,
		JobTitle:          jobTitle,
		RoleID:            role.ID,
		IsActive:          true,
		PasswordChangedAt: &now,
	}

	if err := a.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Log audit
	// a.logAudit(&user.ID, "CREATE", "users", &user.ID, fmt.Sprintf("Created user %s", username))

	return &user, nil
}

// UpdateUser updates an existing user
func (a *App) UpdateUser(userID string, fullName, email, department, jobTitle string, roleID string, isActive bool) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// SERVER-SIDE PERMISSION CHECK: Require users:update or admin (wildcard)
	if err := a.requirePermission("users:update"); err != nil {
		log.Printf("🔒 UpdateUser blocked: %v", err)
		return err
	}

	var user User
	if err := a.db.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update fields
	updates := map[string]any{
		"full_name":    fullName,
		"email":        email,
		"department":   department,
		"job_title":    jobTitle,
		"role_id":      roleID,
		"is_active":    isActive,
		"display_name": strings.Split(fullName, " ")[0],
	}

	if err := a.db.Model(&user).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Log audit event
	// a.logAudit("UPDATE", "users", &userID, fmt.Sprintf("Updated user: %s", fullName), nil)

	return nil
}

// DeactivateUser soft-deletes a user (can be reactivated)
func (a *App) DeactivateUser(userID string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// SERVER-SIDE PERMISSION CHECK: Require users:delete or admin (wildcard)
	if err := a.requirePermission("users:delete"); err != nil {
		log.Printf("🔒 DeactivateUser blocked: %v", err)
		return err
	}

	now := time.Now()
	if err := a.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]any{
		"is_active":  false,
		"deleted_at": &now,
	}).Error; err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	// Log audit event
	// a.logAudit("DELETE", "users", &userID, "Deactivated user", nil)

	return nil
}

// ResetUserPassword resets a user's password
func (a *App) ResetUserPassword(userID string, newPassword string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// SERVER-SIDE PERMISSION CHECK: Require users:update or admin (wildcard)
	if err := a.requirePermission("users:update"); err != nil {
		log.Printf("🔒 ResetUserPassword blocked: %v", err)
		return err
	}

	if newPassword == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Validate password complexity
	if err := validatePasswordComplexity(newPassword); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	// Hash password using bcrypt
	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()

	if err := a.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]any{
		"password_hash":        passwordHash,
		"password_changed_at":  &now,
		"must_change_password": true,
		"failed_logins":        0,
		"locked_until":         nil,
	}).Error; err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	// Log audit event
	// a.logAudit("UPDATE", "users", &userID, "Password reset", nil)

	return nil
}

// requirePermission validates that the current user has the required permission.
// This is the SERVER-SIDE permission middleware that prevents unauthorized operations.
// Supports both User-based auth AND License-based auth.
// Returns error if permission denied, nil if allowed.
func (a *App) requirePermission(permission string) error {
	// SECURITY: RBAC ENABLED - Permission checks enforced
	// Bypass only during startup data import (with 5-minute timeout)
	if a.startupImporting {
		// FIX 1: Enforce timeout to prevent indefinite RBAC bypass
		if time.Since(a.startupImportStartTime) > 5*time.Minute {
			a.startupImporting = false
			log.Println("⚠️ WARNING: Startup import timeout exceeded (5 min), re-enabling RBAC for security")
		} else {
			// FIX 2: Only bypass RBAC for import/data-seeding operations, not all permissions
			importAllowedPerms := map[string]bool{
				"settings:update": true,
				"settings:view":   true,
				"finance:create":  true,
				"finance:read":    true,
				"orders:create":   true,
				"orders:read":     true,
				"po:create":       true,
				"po:read":         true,
			}
			if importAllowedPerms[permission] {
				return nil // Allow only import-related operations during startup
			}
			// All other permissions (users:manage, licenses, admin, etc.) still enforced
		}
	}

	// Wave 5 Mission B: interactive sessions expire after 30 minutes of
	// inactivity. Every bound call lands here, so this is both the
	// enforcement point and the activity signal (session_inactivity.go).
	if err := a.touchInteractiveSession(); err != nil {
		return err
	}

	if isDeletePermission(permission) && !a.currentSessionHasAdminRoleOnly() {
		role := strings.TrimSpace(a.GetCurrentUserRole())
		if role == "" {
			role = "unknown"
		}
		log.Printf("🚫 DELETE BLOCKED: role '%s' attempted '%s' without admin approval", role, permission)
		return fmt.Errorf("delete requires admin approval: permission '%s' cannot be used by role %s", permission, role)
	}

	// METHOD 1: Check User-based authentication first
	if a.currentUser != nil {
		if err := a.checkUserPermission(permission); err == nil {
			return nil
		} else if a.HasLicensePermission(permission) {
			log.Printf("✅ RBAC FALLBACK: license granted '%s' for current user role '%s'", permission, a.currentUser.Role.Name)
			return nil
		} else {
			return err
		}
	}

	// Try to load user from currentUserID
	if a.currentUserID != "" && a.db != nil {
		var user User
		if err := a.db.First(&user, "id = ?", a.currentUserID).Error; err == nil {
			_ = a.hydrateUserRole(&user)
			a.currentUser = &user
			if err := a.checkUserPermission(permission); err == nil {
				return nil
			} else if a.HasLicensePermission(permission) {
				log.Printf("✅ RBAC FALLBACK: license granted '%s' for current user id '%s'", permission, a.currentUserID)
				return nil
			} else {
				return err
			}
		}
	}

	// METHOD 2: Fall back to License-based authentication
	// This is the primary auth method for Acme Instrumentation (license key activation)
	licenseRole := a.GetLicenseRole()
	if licenseRole == "admin" || licenseRole == "developer" {
		return nil // Admin/dev has full access
	}
	if a.HasLicensePermission(permission) {
		return nil // License grants permission
	}

	// Get license role for error message
	if licenseRole != "" {
		log.Printf("🚫 RBAC DENIED: License role '%s' attempted '%s'", licenseRole, permission)
		return fmt.Errorf("access denied: permission '%s' required (your role: %s)", permission, licenseRole)
	}

	// No valid auth method found
	return fmt.Errorf("access denied: not authenticated - please activate your license")
}

func permissionAliases(permission string) []string {
	aliases := map[string]struct{}{
		permission: {},
	}

	aliasGroups := [][]string{
		{"customers:edit", "customers:update"},
		{"suppliers:edit", "suppliers:update"},
		{"offers:edit", "offers:update"},
		{"invoices:edit", "invoices:update"},
		{"po:view", "purchase_orders:view"},
		{"po:create", "purchase_orders:create"},
		{"po:update", "purchase_orders:update"},
		{"po:approve", "purchase_orders:approve"},
		{"po:send", "purchase_orders:send"},
		{"delivery:view", "delivery_notes:view"},
		{"delivery:create", "delivery_notes:create"},
		{"delivery:update", "delivery_notes:update"},
		{"delivery:dispatch", "delivery_notes:dispatch"},
		{"delivery:confirm", "delivery_notes:confirm"},
	}

	for _, group := range aliasGroups {
		found := false
		for _, candidate := range group {
			if candidate == permission {
				found = true
				break
			}
		}
		if found {
			for _, candidate := range group {
				aliases[candidate] = struct{}{}
			}
		}
	}

	results := make([]string, 0, len(aliases))
	for candidate := range aliases {
		results = append(results, candidate)
	}
	return results
}

func permissionGranted(granted, required string) bool {
	if granted == "*" || required == "*" {
		return true
	}

	grantedAliases := permissionAliases(granted)
	requiredAliases := permissionAliases(required)

	for _, grantedCandidate := range grantedAliases {
		for _, requiredCandidate := range requiredAliases {
			if grantedCandidate == requiredCandidate {
				return true
			}
			if strings.HasSuffix(grantedCandidate, ":*") {
				prefix := strings.TrimSuffix(grantedCandidate, ":*")
				if strings.HasPrefix(requiredCandidate, prefix+":") {
					return true
				}
			}
			if strings.Contains(requiredCandidate, ":") {
				category := strings.Split(requiredCandidate, ":")[0]
				if grantedCandidate == category {
					return true
				}
			}
		}
	}

	return false
}

func (a *App) hydrateUserRole(user *User) error {
	if user == nil || a.db == nil {
		return nil
	}
	if user.Role.Permissions != "" || user.Role.Name != "" {
		if user.RoleName == "" {
			if user.Role.DisplayName != "" {
				user.RoleName = user.Role.DisplayName
			} else {
				user.RoleName = user.Role.Name
			}
		}
		return nil
	}
	if strings.TrimSpace(user.RoleID) == "" {
		return nil
	}

	var role Role
	if err := a.db.Where("id = ?", user.RoleID).First(&role).Error; err != nil {
		return err
	}
	user.Role = role
	if role.DisplayName != "" {
		user.RoleName = role.DisplayName
	} else {
		user.RoleName = role.Name
	}
	return nil
}

// checkUserPermission checks permission against User's role (internal helper)
func (a *App) checkUserPermission(permission string) error {
	if a.currentUser == nil {
		return fmt.Errorf("access denied: no user context")
	}
	if err := a.hydrateUserRole(a.currentUser); err != nil {
		log.Printf("⚠️ RBAC: Failed to hydrate role for user %s: %v", a.currentUser.ID, err)
	}

	// Admin/wildcard role has all permissions
	if a.currentUser.Role.Permissions == `["*"]` || strings.Contains(a.currentUser.Role.Permissions, `"*"`) {
		return nil
	}

	// Parse role permissions
	var permissions []string
	if err := json.Unmarshal([]byte(a.currentUser.Role.Permissions), &permissions); err != nil {
		log.Printf("⚠️ RBAC: Failed to parse permissions for user %s: %v", a.currentUser.ID, err)
		return fmt.Errorf("access denied: invalid role configuration")
	}

	// Check if user has the required permission
	// FIX 2: Enhanced permission matching with category-level support
	for _, p := range permissions {
		if permissionGranted(p, permission) {
			return nil
		}
	}

	// Permission denied - log for audit
	log.Printf("🚫 RBAC DENIED: User %s (role: %s) attempted %s", a.currentUser.ID, a.currentUser.Role.Name, permission)
	return fmt.Errorf("access denied: permission '%s' required (your role: %s)", permission, a.currentUser.Role.Name)
}

// HasPermission checks if a user has a specific permission
func (a *App) HasPermission(userID string, permission string) bool {
	if a.db == nil {
		return false
	}

	var user User
	if err := a.db.First(&user, "id = ?", userID).Error; err != nil {
		return false
	}
	if err := a.hydrateUserRole(&user); err != nil {
		log.Printf("Warning: Failed to hydrate role for user %s: %v", userID, err)
		return false
	}

	// Management role has wildcard access
	if user.Role.Permissions == `["*"]` {
		return true
	}

	// Check if permission in role's permission list
	var permissions []string
	if err := json.Unmarshal([]byte(user.Role.Permissions), &permissions); err != nil {
		log.Printf("Warning: Failed to parse permissions for role %s: %v", user.Role.Name, err)
		return false
	}

	for _, p := range permissions {
		if permissionGranted(p, permission) {
			return true
		}
	}

	return false
}

// GetUserPermissions returns all permissions for a user
func (a *App) GetUserPermissions(userID string) ([]string, error) {
	if result, err := a.ValidateLicense(); err == nil && result.Valid {
		if strings.HasPrefix(userID, "license:") {
			return result.Permissions, nil
		}
	}

	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var user User
	if err := a.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if err := a.hydrateUserRole(&user); err != nil {
		return nil, fmt.Errorf("failed to load user role: %w", err)
	}

	// Management role has all permissions
	if user.Role.Permissions == `["*"]` {
		return []string{"*"}, nil
	}

	var permissions []string
	if err := json.Unmarshal([]byte(user.Role.Permissions), &permissions); err != nil {
		return nil, fmt.Errorf("failed to parse permissions: %w", err)
	}

	if (a.currentUser != nil && a.currentUser.ID == userID) || a.currentUserID == userID {
		if result, err := a.ValidateLicense(); err == nil && result.Valid {
			seen := make(map[string]struct{}, len(permissions)+len(result.Permissions))
			merged := make([]string, 0, len(permissions)+len(result.Permissions))
			for _, perm := range permissions {
				if _, ok := seen[perm]; ok {
					continue
				}
				seen[perm] = struct{}{}
				merged = append(merged, perm)
			}
			for _, perm := range result.Permissions {
				if _, ok := seen[perm]; ok {
					continue
				}
				seen[perm] = struct{}{}
				merged = append(merged, perm)
			}
			return merged, nil
		}
	}

	return permissions, nil
}

// isManagementRole reports whether the given role may make an Admin/Manager
// decision (PH SPOC #9). Single source of truth for the role list so the
// credit-override gates stay in sync. Finance is deliberately excluded.
func isManagementRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin", "administrator", "developer", "manager", "management":
		return true
	default:
		return false
	}
}

// creditOverrideActor maps the authenticated session onto a kernel actor for
// the credit-limit override. Management roles carry approve authority; every
// other session may only propose. Distinct from currentApprovalActor (delete
// approvals, admin-only) so the two policies can diverge without coupling.
func (a *App) creditOverrideActor() (actor.Actor, error) {
	authority := actor.AuthorityPropose
	if isManagementRole(a.GetCurrentUserRole()) {
		authority = actor.AuthorityApprove
	}
	id := strings.TrimSpace(a.getCurrentUserID())
	if id == "" {
		id = "unauthenticated-session"
		authority = actor.AuthorityObserve
	}
	return actor.New(actor.Input{
		ID:          id,
		DisplayName: a.getCurrentUserDisplayName(),
		Type:        actor.TypeOperator,
		Authority:   authority,
	})
}

// GetCurrentUserRole returns the current user's role name
// Uses device-based authentication system (see device_service.go)
func (a *App) GetCurrentUserRole() string {
	// Return from current user context (set during device login)
	if a.currentUser != nil && a.currentUser.Role.Name != "" {
		return a.currentUser.Role.Name
	}

	// Fallback: query user by ID if currentUser not loaded
	if a.currentUserID != "" && a.db != nil {
		var user User
		if err := a.db.First(&user, "id = ?", a.currentUserID).Error; err == nil {
			_ = a.hydrateUserRole(&user)
			return user.Role.Name
		}
	}

	if role := strings.TrimSpace(a.GetLicenseRole()); role != "" {
		return role
	}

	// Final fallback: staff role (least privilege principle)
	return "staff"
}

func (a *App) currentSessionHasPermission(permission string) bool {
	return a.requirePermission(permission) == nil
}

func (a *App) currentSessionCanViewFinanceDashboard() bool {
	if a.currentUser != nil {
		roleName := strings.ToLower(strings.TrimSpace(a.currentUser.Role.Name))
		if roleName == "" {
			roleName = strings.ToLower(strings.TrimSpace(a.currentUser.RoleName))
		}
		if roleName == "" {
			roleName = strings.ToLower(strings.TrimSpace(a.currentUser.Role.DisplayName))
		}
		switch roleName {
		case "admin", "administrator", "developer", "manager", "finance":
			return true
		case "sales", "operations", "staff":
			return false
		}

		rolePerms := strings.TrimSpace(a.currentUser.Role.Permissions)
		if rolePerms == `["*"]` || strings.Contains(rolePerms, `"*"`) {
			return true
		}
		var permissions []string
		if err := json.Unmarshal([]byte(rolePerms), &permissions); err == nil {
			for _, p := range permissions {
				if permissionGranted(p, "finance:view") {
					return true
				}
			}
		}
	}

	roleName := strings.ToLower(strings.TrimSpace(a.GetCurrentUserRole()))
	switch roleName {
	case "admin", "administrator", "developer", "manager", "finance":
		return true
	case "sales", "operations", "staff":
		return false
	}

	for _, p := range rolePermissions[roleName] {
		if permissionGranted(p, "finance:view") {
			return true
		}
	}
	return false
}

// CheckPermissionByRole checks if a given role has the specified permission
// This is a helper for permission checking without needing a user ID
func (a *App) CheckPermissionByRole(roleName, permission string) bool {
	if a.db == nil {
		return false
	}

	var role Role
	if err := a.db.Where("name = ?", roleName).First(&role).Error; err != nil {
		return false
	}

	// Admin/wildcard role has all permissions
	if role.Permissions == `["*"]` || strings.Contains(role.Permissions, `"*"`) {
		return true
	}

	// Check if permission in role's permission list
	var permissions []string
	if err := json.Unmarshal([]byte(role.Permissions), &permissions); err != nil {
		log.Printf("Warning: Failed to parse permissions for role %s: %v", role.Name, err)
		return false
	}

	for _, p := range permissions {
		if permissionGranted(p, permission) {
			return true
		}
	}

	return false
}

// GetRolePermissionsList returns permissions as string slice for a role name
func (a *App) GetRolePermissionsList(roleName string) ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var role Role
	if err := a.db.Where("name = ?", roleName).First(&role).Error; err != nil {
		return nil, fmt.Errorf("role not found: %w", err)
	}

	// Wildcard role
	if role.Permissions == `["*"]` {
		return []string{"*"}, nil
	}

	var permissions []string
	if err := json.Unmarshal([]byte(role.Permissions), &permissions); err != nil {
		return nil, fmt.Errorf("failed to parse permissions: %w", err)
	}

	return permissions, nil
}

// GetAuditLogs retrieves audit logs with optional filters
func (a *App) GetAuditLogs(limit int, resource string, action string) ([]AuditLog, error) {
	// P0 FIX: Admin-only function - audit logs are sensitive
	if err := a.requirePermission("users:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 100
	}

	query := a.db.Order("created_at DESC").Limit(limit)

	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}

	var logs []AuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve audit logs: %w", err)
	}

	return logs, nil
}

// logAudit records an action in the audit log through the pkg/infra/audit
// engine — the ONE audit-recording path (Wave 3 B.2). resourceID and
// description are persisted now; the old inline version accepted and
// silently dropped them.
func (a *App) logAudit(userID *string, action, resource string, resourceID *string, description string) {
	rec := a.auditRecorder()
	if rec == nil {
		return
	}

	// Handle nil pointers (system action / no specific resource row)
	var uid, rid string
	if userID != nil {
		uid = *userID
	}
	if resourceID != nil {
		rid = *resourceID
	}

	// Async, matching the historical behavior: the write must survive request
	// cancellation and never block the caller.
	rec.RecordAsync(audit.Entry{
		UserID:      uid,
		Action:      action,
		Resource:    resource,
		ResourceID:  rid,
		Description: description,
	}, func(err error) {
		log.Printf("⚠️ audit write failed (%s %s): %v", action, resource, err)
	})
}

// auditRecorder lazily builds the engine recorder over the app database.
func (a *App) auditRecorder() *audit.Recorder {
	if a == nil || a.db == nil {
		return nil
	}
	return audit.NewRecorder(a.db)
}

// GetCurrentUserStub returns the current authenticated user
// Uses device-based authentication system (see device_service.go)
func (a *App) GetCurrentUserStub() (*User, error) {
	// Priority 1: Return current user from device login context
	if a.currentUser != nil {
		_ = a.hydrateUserRole(a.currentUser)
		return a.currentUser, nil
	}

	// Priority 2: Load user by currentUserID if available
	if a.currentUserID != "" && a.db != nil {
		var user User
		if err := a.db.First(&user, "id = ?", a.currentUserID).Error; err == nil {
			_ = a.hydrateUserRole(&user)
			a.currentUser = &user // Cache for future calls
			return &user, nil
		}
	}

	// Priority 3: Represent an activated license as the current session identity.
	if result, err := a.ValidateLicense(); err == nil && result.Valid {
		rolePermissionsJSON, _ := json.Marshal(result.Permissions)
		displayName := strings.TrimSpace(result.DisplayName)
		if displayName == "" {
			displayName = strings.Title(result.Role)
		}
		username := strings.ToLower(strings.ReplaceAll(displayName, " ", "."))

		return &User{
			Base:        Base{ID: fmt.Sprintf("license:%s", result.Role)},
			Username:    username,
			FullName:    displayName,
			DisplayName: displayName,
			RoleName:    result.Role,
			Role: Role{
				Name:        result.Role,
				DisplayName: strings.Title(result.Role),
				Permissions: string(rolePermissionsJSON),
				IsActive:    true,
				IsSystem:    true,
			},
			IsActive: true,
		}, nil
	}

	// Priority 4: Development fallback - return first management user, or create one
	var user User
	result := a.db.Joins("LEFT JOIN roles ON users.role_id = roles.id").
		Where("roles.name = ?", "management").
		First(&user)

	if result.RowsAffected == 0 {
		// No management user exists - seed roles and create default admin
		if err := a.SeedDefaultRoles(); err != nil {
			return nil, err
		}

		// Create default admin
		var mgmtRole Role
		if err := a.db.Where("name = ?", "management").First(&mgmtRole).Error; err != nil {
			return nil, fmt.Errorf("management role not found")
		}

		// Generate secure random password
		securePassword, err := generateSecurePassword(16)
		if err != nil {
			return nil, fmt.Errorf("failed to generate admin password: %w", err)
		}

		adminUser, err := a.CreateUser(
			"admin",
			"admin@phholdings.com",
			securePassword,
			"System Administrator",
			"Management",
			"Administrator",
			mgmtRole.ID,
		)
		if err != nil {
			return nil, err
		}

		// Log the generated password to console for first-time setup
		// In production, this should be sent to admin via secure channel
		log.Printf("⚠️  FIRST-TIME SETUP: Admin user created")
		log.Printf("📧 Email: admin@phholdings.com")
		log.Printf("🔑 Temporary Password: %s", securePassword)
		log.Printf("⚠️  Please change this password immediately after first login!")

		user = *adminUser
		user.Role = mgmtRole
	}

	user.RoleName = user.Role.DisplayName
	return &user, nil
}

func (a *App) currentSessionIsManagementOrAbove() bool {
	role := strings.ToLower(strings.TrimSpace(a.GetCurrentUserRole()))
	if role == "admin" || role == "administrator" || role == "manager" || role == "management" || role == "developer" {
		return true
	}
	return a.currentSessionHasPermission("finance:view")
}

func (a *App) currentSessionIsAdministrator() bool {
	return a.currentSessionHasAdminRoleOnly()
}

func (a *App) getCurrentUserDisplayName() string {
	if employeeCtx, err := a.GetCurrentEmployeeContext(); err == nil && strings.TrimSpace(employeeCtx.EmployeeName) != "" {
		return strings.TrimSpace(employeeCtx.EmployeeName)
	}

	if a.authManager != nil {
		a.authManager.mu.RLock()
		profile := a.authManager.Profile
		a.authManager.mu.RUnlock()
		if profile != nil && strings.TrimSpace(profile.DisplayName) != "" {
			return strings.TrimSpace(profile.DisplayName)
		}
	}

	if a.db != nil {
		deviceHash := a.getDeviceHash()
		var license LicenseKey
		if err := a.db.Where("device_hash = ? AND activated = 1", deviceHash).First(&license).Error; err == nil {
			if strings.TrimSpace(license.DisplayName) != "" {
				return strings.TrimSpace(license.DisplayName)
			}
		}
	}

	if currentUser, err := a.GetCurrentUserStub(); err == nil && currentUser != nil {
		for _, candidate := range []string{currentUser.DisplayName, currentUser.FullName, currentUser.Username, currentUser.Email} {
			if strings.TrimSpace(candidate) != "" {
				return strings.TrimSpace(candidate)
			}
		}
	}

	return a.getCurrentUserID()
}

// getCurrentUserID returns the ID of the currently authenticated user.
// It checks multiple sources in priority order:
// 1. AuthManager Profile (Microsoft Graph SSO)
// 2. License-based identity (employee name from active license)
// 3. Current User from GetCurrentUserStub (development/fallback)
// 4. Returns "system" as last resort for background jobs
func (a *App) getCurrentUserID() string {
	if employeeCtx, err := a.GetCurrentEmployeeContext(); err == nil && strings.TrimSpace(employeeCtx.EmployeeID) != "" {
		return employeeCtx.EmployeeID
	}

	// Priority 1: Check AuthManager for Microsoft Graph authenticated user
	if a.authManager != nil {
		a.authManager.mu.RLock()
		profile := a.authManager.Profile
		a.authManager.mu.RUnlock()

		if profile != nil && profile.Mail != "" {
			// Try to find user by email from Microsoft Graph profile
			var user User
			if err := a.db.Where("email = ?", profile.Mail).First(&user).Error; err == nil {
				return user.ID
			}
			// User exists in Microsoft Graph but not in local DB yet
			return "sso:" + profile.ID
		}
	}

	// Priority 2: License-based identity - use employee name from active license
	if a.db != nil {
		deviceHash := a.getDeviceHash()
		var license LicenseKey
		if err := a.db.Where("device_hash = ? AND activated = 1", deviceHash).First(&license).Error; err == nil {
			if license.DisplayName != "" {
				return license.DisplayName
			}
			return "license:" + license.Role
		}
	}

	// Priority 3: Check current user stub (development mode or existing session)
	if currentUser, err := a.GetCurrentUserStub(); err == nil && currentUser != nil {
		return currentUser.ID
	}

	// Priority 4: Fallback to "system" for background jobs or unauthenticated operations
	return "system"
}

// =============================================================================
// COSTING SHEET API (CostingSheetScreen.svelte)
// =============================================================================

// CostingLineItem represents a single line item for costing calculation
type CostingLineItem struct {
	Description   string  `json:"description"`
	ProductCode   string  `json:"product_code"`
	ProductType   string  `json:"product_type"`
	Quantity      int     `json:"quantity"`
	UnitCostBHD   float64 `json:"unit_cost_bhd"`
	MarginPercent float64 `json:"margin_percent"` // 0.15 = 15%
}

// CostingRequest is the input for costing calculation
type CostingRequest struct {
	CustomerID        uint              `json:"customer_id"`
	OpportunityID     *uint             `json:"opportunity_id,omitempty"`
	Items             []CostingLineItem `json:"items"`
	ApplyDiscount     bool              `json:"apply_discount"`
	RequestedDiscount float64           `json:"requested_discount,omitempty"` // 0.05 = 5%
	Notes             string            `json:"notes,omitempty"`
}

// CostingLineResult represents the calculated result for a line item
type CostingLineResult struct {
	CostingLineItem
	TotalCostBHD    float64 `json:"total_cost_bhd"`
	UnitSellBHD     float64 `json:"unit_sell_bhd"`
	TotalSellBHD    float64 `json:"total_sell_bhd"`
	UnitProfitBHD   float64 `json:"unit_profit_bhd"`
	TotalProfitBHD  float64 `json:"total_profit_bhd"`
	ActualMarginPct float64 `json:"actual_margin_pct"`
}

// CostingResult represents the complete costing calculation result
type CostingResult struct {
	CustomerID        uint                `json:"customer_id"`
	CustomerName      string              `json:"customer_name"`
	CustomerGrade     string              `json:"customer_grade"`
	OpportunityID     *uint               `json:"opportunity_id,omitempty"`
	Items             []CostingLineResult `json:"items"`
	TotalCostBHD      float64             `json:"total_cost_bhd"`
	TotalSellBHD      float64             `json:"total_sell_bhd"`
	TotalDiscountBHD  float64             `json:"total_discount_bhd"`
	TotalFinalBHD     float64             `json:"total_final_bhd"`
	TotalProfitBHD    float64             `json:"total_profit_bhd"`
	StandardMarginPct float64             `json:"standard_margin_pct"`
	ActualMarginPct   float64             `json:"actual_margin_pct"`
	PaymentTerms      string              `json:"payment_terms"`
	AdvanceRequired   float64             `json:"advance_required"`
	ApprovalStatus    string              `json:"approval_status"` // "AUTO_APPROVED", "NEEDS_APPROVAL", "DECLINED"
	RiskWarnings      []string            `json:"risk_warnings"`
	RecommendedAction string              `json:"recommended_action"`
	NeedsApproval     bool                `json:"needs_approval"`
	ValidUntil        string              `json:"valid_until"`
	CalculatedAt      time.Time           `json:"calculated_at"`
}

// CalculateCosting calculates margins, profits, and approval status for a costing request
