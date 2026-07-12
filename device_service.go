package main

// Wave 5 A.1: device identity + lifecycle logic lives in pkg/infra/device
// (fingerprinting, registration, list/block/unblock). The auth-entangled
// flows below (SetupAdminAccount, ApproveDevice, LoginDevice) stay with
// the host deliberately: they mint users, mutate the session, and lean on
// the auth/RBAC hub — which per W4-D9 gets ports, never relocation.

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	infradevice "ph_holdings_app/pkg/infra/device"
)

// DeviceRegistrationResult represents the result of device registration.
// The alias keeps the JSON contract and Wails binding shape at the root.
type DeviceRegistrationResult = infradevice.RegistrationResult

// GetMachineID generates a unique hardware fingerprint for this device
// Uses MAC address + hostname, hashed for privacy
func GetMachineID() string {
	return infradevice.MachineID()
}

// GetDeviceInfo returns information about the current device
func GetDeviceInfo() (name string, osInfo string) {
	return infradevice.Info()
}

func buildMachineIDHash(identifiers []string) string {
	return infradevice.HashIdentifiers(identifiers)
}

// RegisterDevice registers the current device and returns its status
// This is called on every app startup
func (a *App) RegisterDevice() (*DeviceRegistrationResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return a.deviceService().Register()
}

// SetupAdminAccount creates the first admin user and approves the admin device
// This is only called during first-time setup
func (a *App) SetupAdminAccount(username, password, fullName, email string) (*User, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Verify this is actually first setup
	var userCount int64
	a.db.Model(&User{}).Count(&userCount)
	if userCount > 0 {
		return nil, fmt.Errorf("admin account already exists")
	}

	// P1 FIX: Enhanced input validation
	if GlobalValidator != nil {
		if err := GlobalValidator.ValidateUserInput(username, password, fullName, email); err != nil {
			return nil, err
		}
	} else {
		// Fallback validation if security enhancements not initialized
		if username == "" || password == "" || fullName == "" {
			return nil, fmt.Errorf("username, password, and full name are required")
		}
		if len(password) < 8 {
			return nil, fmt.Errorf("password must be at least 8 characters")
		}
	}

	// Get admin role
	var adminRole Role
	if err := a.db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		return nil, fmt.Errorf("admin role not found - please restart the application")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create admin user
	now := time.Now()
	user := &User{
		Base:         Base{ID: uuid.New().String()},
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		RoleID:       adminRole.ID,
		FullName:     fullName,
		DisplayName:  fullName,
		IsActive:     true,
		LastLoginAt:  &now,
	}

	if err := a.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	// Mark current device as approved admin device
	machineID := GetMachineID()
	if err := a.db.Model(&Device{}).
		Where("machine_id = ?", machineID).
		Updates(map[string]any{
			"status":          "approved",
			"is_admin_device": true,
			"approved_at":     now,
			"approved_by":     user.ID,
		}).Error; err != nil {
		log.Printf("Warning: failed to update device status: %v", err)
	}

	// Get device ID
	var device Device
	a.db.Where("machine_id = ?", machineID).First(&device)

	// Link user to device
	deviceUser := &DeviceUser{
		Base:      Base{ID: uuid.New().String()},
		DeviceID:  device.ID,
		UserID:    user.ID,
		IsPrimary: true,
	}
	a.db.Create(deviceUser)

	// Set current user context
	a.currentUser = user
	a.currentUserID = user.ID
	a.beginInteractiveSession(user.ID)

	log.Printf("✅ Admin account created: %s (%s)", fullName, username)
	return user, nil
}

// ListPendingDevices returns devices awaiting approval (admin only)
func (a *App) ListPendingDevices() ([]Device, error) {
	if err := a.requirePermission("users:manage"); err != nil {
		return nil, err
	}
	return a.deviceService().ListPending()
}

// ListAllDevices returns all devices (admin only)
func (a *App) ListAllDevices() ([]Device, error) {
	if err := a.requirePermission("users:manage"); err != nil {
		return nil, err
	}
	return a.deviceService().ListAll()
}

// ApproveDevice approves a pending device and creates a user for it
func (a *App) ApproveDevice(deviceID, roleID, username, password, fullName, email string) error {
	if err := a.requirePermission("users:manage"); err != nil {
		return err
	}

	// P1 FIX: Enhanced input validation
	if GlobalValidator != nil {
		if err := GlobalValidator.ValidateUserInput(username, password, fullName, email); err != nil {
			return err
		}
	} else {
		// Fallback validation
		if deviceID == "" || roleID == "" || username == "" || password == "" || fullName == "" {
			return fmt.Errorf("all fields are required")
		}
		if len(password) < 8 {
			return fmt.Errorf("password must be at least 8 characters")
		}
	}

	// Verify device exists and is pending
	var device Device
	if err := a.db.First(&device, "id = ?", deviceID).Error; err != nil {
		return fmt.Errorf("device not found")
	}
	if device.Status != "pending" {
		return fmt.Errorf("device is not in pending status")
	}

	// Verify role exists
	var role Role
	if err := a.db.First(&role, "id = ?", roleID).Error; err != nil {
		return fmt.Errorf("role not found")
	}

	// Check username uniqueness
	var existingUser User
	if err := a.db.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return fmt.Errorf("username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &User{
		Base:         Base{ID: uuid.New().String()},
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		RoleID:       roleID,
		FullName:     fullName,
		DisplayName:  fullName,
		IsActive:     true,
	}

	if err := a.db.Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Update device status
	now := time.Now()
	if err := a.db.Model(&device).Updates(map[string]any{
		"status":      "approved",
		"approved_by": a.currentUserID,
		"approved_at": now,
	}).Error; err != nil {
		return fmt.Errorf("failed to update device status: %w", err)
	}

	// Link user to device
	deviceUser := &DeviceUser{
		Base:      Base{ID: uuid.New().String()},
		DeviceID:  deviceID,
		UserID:    user.ID,
		IsPrimary: true,
	}
	a.db.Create(deviceUser)

	// P1 FIX: Audit device approval
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogDeviceAction(a.currentUserID, deviceID, "device_approved", true, map[string]any{
			"device_name": device.DeviceName,
			"user_id":     user.ID,
			"username":    username,
			"role":        role.Name,
		})
	}

	log.Printf("✅ Device approved: %s, User: %s (%s), Role: %s",
		device.DeviceName, fullName, username, role.DisplayName)

	return nil
}

// BlockDevice blocks a device from accessing the system
func (a *App) BlockDevice(deviceID string) error {
	if err := a.requirePermission("users:manage"); err != nil {
		return err
	}

	device, err := a.deviceService().Block(deviceID)
	if err != nil {
		return err
	}

	// P1 FIX: Audit device blocking
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogDeviceAction(a.currentUserID, deviceID, "device_blocked", true, map[string]any{
			"device_name": device.DeviceName,
			"machine_id":  device.MachineID,
		})
	}

	return nil
}

// UnblockDevice unblocks a previously blocked device
func (a *App) UnblockDevice(deviceID string) error {
	if err := a.requirePermission("users:manage"); err != nil {
		return err
	}
	return a.deviceService().Unblock(deviceID)
}

// GetDeviceUsers returns users associated with a device
func (a *App) GetDeviceUsers(deviceID string) ([]DeviceUser, error) {
	if err := a.requirePermission("users:manage"); err != nil {
		return nil, err
	}
	return a.deviceService().Users(deviceID)
}

// LoginDevice authenticates a user on an approved device
func (a *App) LoginDevice(username, password string) (*DeviceRegistrationResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	machineID := GetMachineID()

	// P1 FIX: Rate limiting for login attempts (5 per minute per device)
	if GlobalRateLimiter != nil {
		rateLimitKey := "login:" + machineID
		if !GlobalRateLimiter.Allow(rateLimitKey, RateLimitConfig.LoginAttemptsPerMinute, 1*time.Minute/time.Duration(RateLimitConfig.LoginAttemptsPerMinute)) {
			log.Printf("🚫 Rate limit exceeded for login attempts from device %s", machineID[:16])
			return nil, fmt.Errorf("too many login attempts, please wait a moment and try again")
		}
	}

	// Check device is approved
	var device Device
	if err := a.db.Where("machine_id = ?", machineID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("device not registered")
	}

	if device.Status != "approved" && device.Status != "first_setup" {
		return nil, fmt.Errorf("device not approved")
	}

	// Find user
	var user User
	if err := a.db.Where("username = ? AND is_active = ?", username, true).First(&user).Error; err != nil {
		// P1 FIX: Audit failed login attempt
		if GlobalAuditLogger != nil {
			GlobalAuditLogger.LogPermissionChange("system", username, "login_attempt", "", "", false, "user not found or inactive")
		}
		return nil, fmt.Errorf("invalid username or password")
	}
	if err := a.hydrateUserRole(&user); err != nil {
		return nil, fmt.Errorf("failed to load role for user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// P1 FIX: Audit failed login attempt
		if GlobalAuditLogger != nil {
			GlobalAuditLogger.LogPermissionChange("system", user.ID, "login_attempt", "", "", false, "incorrect password")
		}
		return nil, fmt.Errorf("invalid username or password")
	}

	// Update last login
	now := time.Now()
	a.db.Model(&user).Update("last_login_at", now)
	a.db.Model(&device).Update("last_seen_at", now)

	// Set current user context
	a.currentUser = &user
	a.currentUserID = user.ID

	// NOTE (Wave 4 B.2): the GlobalSessionManager.CreateSession call that
	// lived here fed an in-memory map nothing ever read — deleted with the
	// dead SessionManager. DB-backed sessions are AuthManager's job
	// (auth_session.go); Wave 5 Mission B records this login there and
	// requirePermission enforces the 30-minute inactivity timeout.
	a.beginInteractiveSession(user.ID)

	// P1 FIX: Audit successful login
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogPermissionChange("system", user.ID, "login_success", "", user.Role.Name, true, "successful authentication")
	}

	log.Printf("✅ User logged in: %s (%s) on device %s", user.FullName, username, device.DeviceName)

	return &DeviceRegistrationResult{
		DeviceID:    device.ID,
		MachineID:   machineID,
		Status:      "approved",
		UserID:      user.ID,
		UserName:    user.FullName,
		RoleName:    user.Role.DisplayName,
		Permissions: infradevice.ParsePermissions(user.Role.Permissions),
	}, nil
}

// CheckDeviceStatus returns the current device status (for polling from pending screen)
func (a *App) CheckDeviceStatus() (*DeviceRegistrationResult, error) {
	return a.RegisterDevice()
}

// GetCurrentDeviceInfo returns information about the current device
func (a *App) GetCurrentDeviceInfo() (*Device, error) {
	return a.deviceService().Current()
}
