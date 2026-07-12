package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

type Employee struct {
	Base
	EmployeeCode      string     `gorm:"uniqueIndex;size:50" json:"employee_code"`
	FullName          string     `gorm:"index;size:255" json:"full_name"`
	PreferredName     string     `gorm:"size:255" json:"preferred_name"`
	Email             string     `gorm:"index;size:255" json:"email"`
	Phone             string     `gorm:"size:50" json:"phone"`
	Department        string     `gorm:"index;size:100" json:"department"`
	JobTitle          string     `gorm:"size:100" json:"job_title"`
	EmploymentStatus  string     `gorm:"index;size:30;default:'active'" json:"employment_status"`
	ManagerEmployeeID *string    `gorm:"index;size:36" json:"manager_employee_id"`
	StartDate         *time.Time `json:"start_date"`
	EndDate           *time.Time `json:"end_date"`
	EmergencyContact  string     `gorm:"type:varchar(500)" json:"emergency_contact"`
	Notes             string     `gorm:"type:text" json:"notes"`
	IsActive          bool       `gorm:"index;default:true" json:"is_active"`
	ArchivedAt        *time.Time `json:"archived_at"`
	ArchivedBy        string     `gorm:"size:36" json:"archived_by"`
	ArchiveReason     string     `gorm:"type:text" json:"archive_reason"`
	ArchiveRequestID  string     `gorm:"size:36" json:"archive_request_id"`

	ManagerName string `gorm:"-" json:"manager_name,omitempty"`
}

func (Employee) TableName() string { return "employees" }

type EmployeeAccessLink struct {
	Base
	EmployeeID   string `gorm:"uniqueIndex:idx_employee_license,priority:1;index;size:36" json:"employee_id"`
	LicenseKey   string `gorm:"uniqueIndex:idx_employee_license,priority:2;index;size:20" json:"license_key"`
	UserID       string `gorm:"index;size:36" json:"user_id"`
	DeviceID     string `gorm:"index;size:36" json:"device_id"`
	AccessStatus string `gorm:"index;size:30;default:'active'" json:"access_status"`
	IsPrimary    bool   `gorm:"default:true" json:"is_primary"`

	EmployeeName string `gorm:"-" json:"employee_name,omitempty"`
	DeviceName   string `gorm:"-" json:"device_name,omitempty"`
}

func (EmployeeAccessLink) TableName() string { return "employee_access_links" }

type Project struct {
	Base
	Name             string     `gorm:"index;size:255" json:"name"`
	ProjectType      string     `gorm:"index;size:30;default:'internal'" json:"project_type"`
	Description      string     `gorm:"type:text" json:"description"`
	Status           string     `gorm:"index;size:30;default:'active'" json:"status"`
	CustomerID       *string    `gorm:"index;size:36" json:"customer_id"`
	OpportunityID    *string    `gorm:"index;size:36" json:"opportunity_id"`
	OrderID          *string    `gorm:"index;size:36" json:"order_id"`
	CustomerName     string     `gorm:"size:255" json:"customer_name"`
	EndUserName      string     `gorm:"size:255" json:"end_user_name"`
	OpportunityKey   string     `gorm:"index;size:80" json:"opportunity_key"`
	CustomerPOCName  string     `gorm:"size:255" json:"customer_poc_name"`
	CustomerPOCEmail string     `gorm:"size:255" json:"customer_poc_email"`
	CustomerPOCPhone string     `gorm:"size:50" json:"customer_poc_phone"`
	StartsOn         *time.Time `json:"starts_on"`
	EndsOn           *time.Time `json:"ends_on"`
}

func (Project) TableName() string { return "projects" }

type ProjectMember struct {
	Base
	ProjectID         string     `gorm:"uniqueIndex:idx_project_member,priority:1;index;size:36" json:"project_id"`
	EmployeeID        string     `gorm:"uniqueIndex:idx_project_member,priority:2;index;size:36" json:"employee_id"`
	Role              string     `gorm:"size:100" json:"role"`
	AllocationPercent float64    `gorm:"default:100" json:"allocation_percent"`
	IsActive          bool       `gorm:"index;default:true" json:"is_active"`
	JoinedAt          *time.Time `json:"joined_at"`
	LeftAt            *time.Time `json:"left_at"`
	EmployeeName      string     `gorm:"-" json:"employee_name,omitempty"`
	ProjectName       string     `gorm:"-" json:"project_name,omitempty"`
}

func (ProjectMember) TableName() string { return "project_members" }

type Notification struct {
	Base
	EmployeeID       string     `gorm:"index;size:36" json:"employee_id"`
	NotificationType string     `gorm:"index;size:50" json:"notification_type"`
	Title            string     `gorm:"size:255" json:"title"`
	Message          string     `gorm:"type:text" json:"message"`
	Status           string     `gorm:"index;size:20;default:'unread'" json:"status"`
	SourceType       string     `gorm:"index;size:50" json:"source_type"`
	SourceID         string     `gorm:"index;size:36" json:"source_id"`
	ActionRoute      string     `gorm:"size:255" json:"action_route"`
	ActionPayload    string     `gorm:"type:text" json:"action_payload"`
	ReadAt           *time.Time `json:"read_at"`
	DeliveredAt      *time.Time `json:"delivered_at"`
}

func (Notification) TableName() string { return "notifications" }

type NotificationReceipt struct {
	Base
	NotificationID string    `gorm:"index;size:36" json:"notification_id"`
	EmployeeID     string    `gorm:"index;size:36" json:"employee_id"`
	DeviceID       string    `gorm:"index;size:36" json:"device_id"`
	ReceiptType    string    `gorm:"index;size:20" json:"receipt_type"`
	ReceivedAt     time.Time `gorm:"index" json:"received_at"`
}

func (NotificationReceipt) TableName() string { return "notification_receipts" }

type TaskItem struct {
	Base
	Title              string     `gorm:"index;size:255" json:"title"`
	Description        string     `gorm:"type:text" json:"description"`
	TaskType           string     `gorm:"index;size:30;default:'general'" json:"task_type"`
	LegacyFollowUpID   *string    `gorm:"index;size:36" json:"legacy_follow_up_id"`
	Status             string     `gorm:"index;size:30;default:'open'" json:"status"`
	BlockedReason      string     `gorm:"type:text" json:"blocked_reason"`
	Priority           string     `gorm:"index;size:20;default:'medium'" json:"priority"`
	DueDate            *time.Time `gorm:"index" json:"due_date"`
	CustomerID         *string    `gorm:"index;size:36" json:"customer_id"`
	OpportunityID      *string    `gorm:"index;size:36" json:"opportunity_id"`
	OrderID            *string    `gorm:"index;size:36" json:"order_id"`
	ProjectID          *string    `gorm:"index;size:36" json:"project_id"`
	CreatorEmployeeID  string     `gorm:"index;size:36" json:"creator_employee_id"`
	AssigneeEmployeeID *string    `gorm:"index;size:36" json:"assignee_employee_id"`
	WatchersJSON       string     `gorm:"type:text" json:"watchers_json"`
	StartedAt          *time.Time `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	LastCommentAt      *time.Time `json:"last_comment_at"`

	CreatorName  string `gorm:"-" json:"creator_name,omitempty"`
	AssigneeName string `gorm:"-" json:"assignee_name,omitempty"`
}

func (TaskItem) TableName() string { return "task_items" }

type TaskComment struct {
	Base
	TaskID       string `gorm:"index;size:36" json:"task_id"`
	EmployeeID   string `gorm:"index;size:36" json:"employee_id"`
	Body         string `gorm:"type:text" json:"body"`
	EmployeeName string `gorm:"-" json:"employee_name,omitempty"`
}

func (TaskComment) TableName() string { return "task_comments" }

type TaskActivity struct {
	Base
	TaskID       string `gorm:"index;size:36" json:"task_id"`
	EmployeeID   string `gorm:"index;size:36" json:"employee_id"`
	ActivityType string `gorm:"index;size:50" json:"activity_type"`
	Detail       string `gorm:"type:text" json:"detail"`
	MetadataJSON string `gorm:"type:text" json:"metadata_json"`
	EmployeeName string `gorm:"-" json:"employee_name,omitempty"`
}

func (TaskActivity) TableName() string { return "task_activity" }

type CollaborativePendingOperation struct {
	Base
	EntityType    string     `gorm:"index;size:50" json:"entity_type"`
	EntityID      string     `gorm:"index;size:36" json:"entity_id"`
	Operation     string     `gorm:"index;size:50" json:"operation"`
	Payload       string     `gorm:"type:text" json:"payload"`
	Status        string     `gorm:"index;size:20;default:'pending'" json:"status"`
	Attempts      int        `gorm:"default:0" json:"attempts"`
	LastAttemptAt *time.Time `json:"last_attempt_at"`
	NextAttemptAt *time.Time `json:"next_attempt_at"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`
}

func (CollaborativePendingOperation) TableName() string { return "collaborative_pending_operations" }

type CurrentEmployeeContext struct {
	EmployeeID   string   `json:"employee_id"`
	EmployeeName string   `json:"employee_name"`
	LicenseKey   string   `json:"license_key"`
	LicenseRole  string   `json:"license_role"`
	DeviceID     string   `json:"device_id"`
	UserID       string   `json:"user_id"`
	ResolvedBy   string   `json:"resolved_by"`
	Permissions  []string `json:"permissions"`
}

type EmployeeContributionSummary struct {
	EmployeeID         string  `json:"employee_id"`
	EmployeeCode       string  `json:"employee_code"`
	EmployeeName       string  `json:"employee_name"`
	Department         string  `json:"department"`
	JobTitle           string  `json:"job_title"`
	ManagerEmployeeID  string  `json:"manager_employee_id"`
	ManagerName        string  `json:"manager_name"`
	EmploymentStatus   string  `json:"employment_status"`
	IsActive           bool    `json:"is_active"`
	ActiveProjectCount int     `json:"active_project_count"`
	ActiveTaskCount    int     `json:"active_task_count"`
	CompletedTaskCount int     `json:"completed_task_count"`
	BlockedTaskCount   int     `json:"blocked_task_count"`
	OverdueTaskCount   int     `json:"overdue_task_count"`
	CompletionRate     float64 `json:"completion_rate"`
	OpportunityYTD     int     `json:"opportunity_ytd"`
	OpportunityWonYTD  int     `json:"opportunity_won_ytd"`
	OpportunityLostYTD int     `json:"opportunity_lost_ytd"`
	RevenueYTD         float64 `json:"revenue_ytd"`
	PrimaryLicenseKey  string  `json:"primary_license_key,omitempty"`
	PrimaryDeviceName  string  `json:"primary_device_name,omitempty"`
}

// Mission I (I-11): bound DDL is gated; startup uses the internal.
func (a *App) EnsureCollaborativeFoundation() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	return a.ensureCollaborativeFoundationInternal()
}

func (a *App) ensureCollaborativeFoundationInternal() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	models := []any{
		&Employee{},
		&EmployeeDocument{}, // Wave 9.8 B4
		&EmployeeAccessLink{},
		&Project{},
		&ProjectMember{},
		&Notification{},
		&NotificationReceipt{},
		&DeleteApprovalRequest{},
		&EmployeeArchiveRequest{},
		&TaskItem{},
		&TaskComment{},
		&TaskActivity{},
		&CollaborativePendingOperation{},
	}

	for _, model := range models {
		if err := a.db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	if err := a.seedEmployeesFromLicenseKeys(); err != nil {
		return err
	}

	return nil
}

func (a *App) seedEmployeesFromLicenseKeys() error {
	if a.db == nil {
		return nil
	}

	var licenses []LicenseKey
	if err := a.db.Where("display_name IS NOT NULL AND TRIM(display_name) != ''").Find(&licenses).Error; err != nil {
		return fmt.Errorf("failed to load license keys for employee seed: %w", err)
	}

	for _, license := range licenses {
		name := strings.TrimSpace(license.DisplayName)
		if name == "" {
			continue
		}

		var employee Employee
		if err := a.db.Where("LOWER(full_name) = LOWER(?)", name).First(&employee).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("failed to lookup employee seed %q: %w", name, err)
			}

			code, codeErr := a.generateEmployeeCode()
			if codeErr != nil {
				return codeErr
			}

			employee = Employee{
				Base:             Base{CreatedBy: "system"},
				EmployeeCode:     code,
				FullName:         name,
				PreferredName:    firstToken(name),
				Department:       inferDepartmentFromRole(license.Role),
				JobTitle:         strings.Title(license.Role),
				EmploymentStatus: "active",
				IsActive:         true,
			}

			if err := a.db.Create(&employee).Error; err != nil {
				return fmt.Errorf("failed to seed employee %q: %w", name, err)
			}
		}

		link, err := a.buildAccessLinkForLicense(employee.ID, license)
		if err != nil {
			return err
		}

		var existing EmployeeAccessLink
		if err := a.db.Where("employee_id = ? AND license_key = ?", employee.ID, license.Key).First(&existing).Error; err == nil {
			updates := map[string]any{
				"device_id":     link.DeviceID,
				"access_status": link.AccessStatus,
				"is_primary":    true,
			}
			if link.UserID != "" {
				updates["user_id"] = link.UserID
			}
			if err := a.db.Model(&existing).Updates(updates).Error; err != nil {
				return fmt.Errorf("failed to update employee access link for %q: %w", name, err)
			}
			continue
		}

		if err := a.db.Create(&link).Error; err != nil {
			return fmt.Errorf("failed to seed employee access link for %q: %w", name, err)
		}
	}

	return nil
}

func (a *App) generateEmployeeCode() (string, error) {
	var count int64
	if err := a.db.Model(&Employee{}).Count(&count).Error; err != nil {
		return "", fmt.Errorf("failed to generate employee code: %w", err)
	}
	return fmt.Sprintf("EMP-%04d", count+1), nil
}

func inferDepartmentFromRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin", "manager":
		return "Management"
	case "sales":
		return "Sales"
	case "operations":
		return "Operations"
	default:
		return "General"
	}
}

func firstToken(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func (a *App) buildAccessLinkForLicense(employeeID string, license LicenseKey) (EmployeeAccessLink, error) {
	link := EmployeeAccessLink{
		Base:         Base{CreatedBy: "system"},
		EmployeeID:   employeeID,
		LicenseKey:   license.Key,
		AccessStatus: "active",
		IsPrimary:    true,
	}

	if strings.TrimSpace(license.DeviceHash) != "" {
		var device Device
		if err := a.db.Where("machine_id = ?", license.DeviceHash).First(&device).Error; err == nil {
			link.DeviceID = device.ID

			var deviceUser DeviceUser
			if err := a.db.Where("device_id = ? AND is_primary = ?", device.ID, true).First(&deviceUser).Error; err == nil {
				link.UserID = deviceUser.UserID
			}
		}
	}

	return link, nil
}

func (a *App) getActiveLicenseRecord() (*LicenseKey, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	deviceHash := a.getDeviceHash()
	var license LicenseKey
	if a.developerMasterKeyEnabled() {
		if err := a.db.Where("key = ? AND activated = 1", masterKey).First(&license).Error; err == nil {
			return &license, nil
		}
	}
	if err := a.db.Where("device_hash = ? AND activated = 1", deviceHash).First(&license).Error; err != nil {
		return nil, err
	}
	return &license, nil
}

func (a *App) GetCurrentEmployeeContext() (CurrentEmployeeContext, error) {
	ctx := CurrentEmployeeContext{}
	if a.db == nil {
		return ctx, fmt.Errorf("database not initialized")
	}

	var employee Employee
	if a.currentUserID != "" {
		var link EmployeeAccessLink
		if err := a.db.Where("user_id = ? AND access_status = ?", a.currentUserID, "active").Order("is_primary DESC").First(&link).Error; err == nil {
			if err := a.db.First(&employee, "id = ?", link.EmployeeID).Error; err == nil {
				perms, _ := a.GetUserPermissions(a.currentUserID)
				return CurrentEmployeeContext{
					EmployeeID:   employee.ID,
					EmployeeName: employee.FullName,
					LicenseKey:   link.LicenseKey,
					DeviceID:     link.DeviceID,
					UserID:       link.UserID,
					ResolvedBy:   "user",
					Permissions:  perms,
				}, nil
			}
		}
	}

	license, err := a.getActiveLicenseRecord()
	if err == nil {
		var link EmployeeAccessLink
		if err := a.db.Where("license_key = ? AND access_status = ?", license.Key, "active").Order("is_primary DESC").First(&link).Error; err == nil {
			if err := a.db.First(&employee, "id = ?", link.EmployeeID).Error; err == nil {
				return CurrentEmployeeContext{
					EmployeeID:   employee.ID,
					EmployeeName: employee.FullName,
					LicenseKey:   license.Key,
					LicenseRole:  license.Role,
					DeviceID:     link.DeviceID,
					UserID:       link.UserID,
					ResolvedBy:   "license",
					Permissions:  rolePermissions[license.Role],
				}, nil
			}
		}

		name := strings.TrimSpace(license.DisplayName)
		if name != "" {
			if employee, err := a.findEmployeeByDisplayName(name); err == nil {
				return CurrentEmployeeContext{
					EmployeeID:   employee.ID,
					EmployeeName: employee.FullName,
					LicenseKey:   license.Key,
					LicenseRole:  license.Role,
					ResolvedBy:   "license_display_name",
					Permissions:  rolePermissions[license.Role],
				}, nil
			}
		}

		if fallback, fallbackErr := a.ensureEmployeeContextForLicense(license); fallbackErr == nil {
			return fallback, nil
		} else {
			log.Printf("⚠️ Failed to auto-resolve employee context for active license: %v", fallbackErr)
		}
	}

	return ctx, fmt.Errorf("employee context not resolved")
}

func (a *App) findEmployeeByDisplayName(name string) (Employee, error) {
	var employee Employee
	name = strings.TrimSpace(name)
	if a.db == nil {
		return employee, fmt.Errorf("database not initialized")
	}
	if name == "" {
		return employee, gorm.ErrRecordNotFound
	}

	if err := a.db.
		Where("deleted_at IS NULL AND is_active = ? AND (LOWER(full_name) = LOWER(?) OR LOWER(preferred_name) = LOWER(?) OR LOWER(employee_code) = LOWER(?))", true, name, name, name).
		Order("full_name ASC").
		First(&employee).Error; err == nil {
		return employee, nil
	} else if err != gorm.ErrRecordNotFound {
		return employee, err
	}

	normalizedTarget := normalizeEmployeeLookupName(name)
	if normalizedTarget == "" {
		return employee, gorm.ErrRecordNotFound
	}

	var employees []Employee
	if err := a.db.
		Where("deleted_at IS NULL AND is_active = ?", true).
		Order("full_name ASC").
		Find(&employees).Error; err != nil {
		return employee, err
	}
	for _, candidate := range employees {
		if normalizeEmployeeLookupName(candidate.FullName) == normalizedTarget ||
			normalizeEmployeeLookupName(candidate.PreferredName) == normalizedTarget ||
			normalizeEmployeeLookupName(candidate.EmployeeCode) == normalizedTarget {
			return candidate, nil
		}
	}

	return employee, gorm.ErrRecordNotFound
}

func normalizeEmployeeLookupName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func (a *App) ensureEmployeeContextForLicense(license *LicenseKey) (CurrentEmployeeContext, error) {
	ctx := CurrentEmployeeContext{}
	if a.db == nil {
		return ctx, fmt.Errorf("database not initialized")
	}
	if license == nil || strings.TrimSpace(license.Key) == "" {
		return ctx, fmt.Errorf("active license is unavailable")
	}

	name := strings.TrimSpace(license.DisplayName)
	if name == "" && a.currentUser != nil {
		name = firstNonEmptyCollaboration(a.currentUser.DisplayName, a.currentUser.FullName, a.currentUser.Username, a.currentUser.Email)
	}
	if name == "" && a.authManager != nil {
		a.authManager.mu.RLock()
		profile := a.authManager.Profile
		a.authManager.mu.RUnlock()
		if profile != nil {
			name = firstNonEmptyCollaboration(profile.DisplayName, profile.UserPrincipalName, profile.Mail)
		}
	}
	if name == "" {
		switch strings.ToLower(strings.TrimSpace(license.Role)) {
		case "admin", "management", "manager":
			name = "Administrator"
		case "sales":
			name = "Sales User"
		case "operations":
			name = "Operations User"
		default:
			name = "AsymmFlow User"
		}
	}

	employee, err := a.findEmployeeByDisplayName(name)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return ctx, fmt.Errorf("failed to lookup fallback employee: %w", err)
		}

		code, codeErr := a.generateEmployeeCode()
		if codeErr != nil {
			return ctx, codeErr
		}

		employee = Employee{
			Base:             Base{CreatedBy: "system"},
			EmployeeCode:     code,
			FullName:         name,
			PreferredName:    firstToken(name),
			Department:       inferDepartmentFromRole(license.Role),
			JobTitle:         strings.Title(strings.TrimSpace(license.Role)),
			EmploymentStatus: "active",
			IsActive:         true,
		}
		if employee.JobTitle == "" {
			employee.JobTitle = "User"
		}

		if err := a.db.Create(&employee).Error; err != nil {
			return ctx, fmt.Errorf("failed to create fallback employee: %w", err)
		}
	}

	link, err := a.buildAccessLinkForLicense(employee.ID, *license)
	if err != nil {
		return ctx, err
	}
	if link.UserID == "" {
		link.UserID = a.currentUserID
	}

	var existing EmployeeAccessLink
	if err := a.db.Where("employee_id = ? AND license_key = ?", employee.ID, license.Key).First(&existing).Error; err == nil {
		updates := map[string]any{
			"device_id":     link.DeviceID,
			"access_status": "active",
			"is_primary":    true,
		}
		if link.UserID != "" {
			updates["user_id"] = link.UserID
		}
		if err := a.db.Model(&existing).Updates(updates).Error; err != nil {
			return ctx, fmt.Errorf("failed to update fallback employee access link: %w", err)
		}
		link = existing
		link.DeviceID = firstNonEmptyCollaboration(updates["device_id"].(string), existing.DeviceID)
		if userID, ok := updates["user_id"].(string); ok && userID != "" {
			link.UserID = userID
		}
	} else if err == gorm.ErrRecordNotFound {
		if err := a.db.Create(&link).Error; err != nil {
			return ctx, fmt.Errorf("failed to create fallback employee access link: %w", err)
		}
	} else {
		return ctx, fmt.Errorf("failed to lookup fallback employee access link: %w", err)
	}

	log.Printf("✅ Auto-resolved employee context %s for active license role %s", employee.FullName, license.Role)
	return CurrentEmployeeContext{
		EmployeeID:   employee.ID,
		EmployeeName: employee.FullName,
		LicenseKey:   license.Key,
		LicenseRole:  license.Role,
		DeviceID:     link.DeviceID,
		UserID:       link.UserID,
		ResolvedBy:   "license_auto",
		Permissions:  rolePermissions[license.Role],
	}, nil
}

func firstNonEmptyCollaboration(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (a *App) CreateEmployeeProfile(employee Employee) (Employee, error) {
	if err := a.requirePermission("hr:create"); err != nil {
		return Employee{}, err
	}
	if !a.currentSessionHasAdminRoleOnly() {
		return Employee{}, fmt.Errorf("only admin can add employee profiles")
	}
	if a.db == nil {
		return Employee{}, fmt.Errorf("database not initialized")
	}

	employee.FullName = strings.TrimSpace(employee.FullName)
	if employee.FullName == "" {
		return Employee{}, fmt.Errorf("full name is required")
	}
	if employee.EmployeeCode == "" {
		code, err := a.generateEmployeeCode()
		if err != nil {
			return Employee{}, err
		}
		employee.EmployeeCode = code
	}
	if employee.PreferredName == "" {
		employee.PreferredName = firstToken(employee.FullName)
	}
	if employee.EmploymentStatus == "" {
		employee.EmploymentStatus = "active"
	}
	employee.IsActive = true
	employee.CreatedBy = a.getCurrentUserID()

	if err := a.db.Create(&employee).Error; err != nil {
		return Employee{}, fmt.Errorf("failed to create employee profile: %w", err)
	}
	if payload, err := json.Marshal(employee); err == nil {
		a.enqueueCollaborativeOperation("employee", employee.ID, "create", string(payload))
	}
	a.emitCollaborationEvent("employees:updated", map[string]any{
		"employee_id": employee.ID,
		"action":      "create",
	})
	a.queueCollaborativeSync("employee_create")
	return employee, nil
}

func (a *App) ListEmployeeProfiles(activeOnly bool) ([]Employee, error) {
	if err := a.requirePermission("hr:view"); err != nil {
		if taskErr := a.requirePermission("tasks:view"); taskErr != nil {
			return nil, err
		}
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := a.db.Order("full_name ASC")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	var employees []Employee
	if err := query.Find(&employees).Error; err != nil {
		return nil, fmt.Errorf("failed to list employee profiles: %w", err)
	}

	managerIDs := make([]string, 0)
	for _, employee := range employees {
		if employee.ManagerEmployeeID != nil && *employee.ManagerEmployeeID != "" {
			managerIDs = append(managerIDs, *employee.ManagerEmployeeID)
		}
	}

	managerNames := map[string]string{}
	if len(managerIDs) > 0 {
		var managers []Employee
		_ = a.db.Where("id IN ?", managerIDs).Find(&managers).Error
		for _, manager := range managers {
			managerNames[manager.ID] = manager.FullName
		}
	}

	for i := range employees {
		if employees[i].ManagerEmployeeID != nil {
			employees[i].ManagerName = managerNames[*employees[i].ManagerEmployeeID]
		}
	}

	return employees, nil
}

// GetPreparedByOptions returns the document "prepared by" picker choices:
// configured signature-block identities, employee names, and license display
// names, deduplicated and sorted. Sovereign divergence from deployed PH: PH
// additionally hardcodes real staff first names in source; here the seed list
// is ONLY the overlay's signature blocks (synthetic canon by default, real
// identities via the sovereign overlay.json) — per the no-real-people
// invariant, names are deployment configuration, never source.
func (a *App) GetPreparedByOptions() ([]string, error) {
	if err := a.requirePermission("offers:create"); err != nil {
		if editErr := a.requirePermission("offers:edit"); editErr != nil {
			if costingErr := a.requirePermission("costing:read"); costingErr != nil {
				return nil, err
			}
		}
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	names := map[string]struct{}{}
	addName := func(name string) {
		name = strings.TrimSpace(name)
		if name != "" {
			names[name] = struct{}{}
		}
	}

	for _, name := range defaultOfferSignatureNames() {
		addName(name)
	}

	var employees []Employee
	if err := a.db.Order("full_name ASC").Find(&employees).Error; err != nil {
		return nil, fmt.Errorf("failed to list prepared-by employee options: %w", err)
	}
	for _, employee := range employees {
		addName(employee.FullName)
		addName(employee.PreferredName)
	}

	var licenses []LicenseKey
	if err := a.db.Where("display_name IS NOT NULL AND TRIM(display_name) != ''").Order("display_name ASC").Find(&licenses).Error; err != nil {
		return nil, fmt.Errorf("failed to list prepared-by license options: %w", err)
	}
	for _, license := range licenses {
		addName(license.DisplayName)
	}

	options := make([]string, 0, len(names))
	for name := range names {
		options = append(options, name)
	}
	sort.Strings(options)
	return options, nil
}

func (a *App) UpdateEmployeeProfile(employee Employee) (Employee, error) {
	if err := a.requirePermission("hr:update"); err != nil {
		return Employee{}, err
	}
	if a.db == nil {
		return Employee{}, fmt.Errorf("database not initialized")
	}

	employee.ID = strings.TrimSpace(employee.ID)
	if employee.ID == "" {
		return Employee{}, fmt.Errorf("employee id is required")
	}
	employee.FullName = strings.TrimSpace(employee.FullName)
	if employee.FullName == "" {
		return Employee{}, fmt.Errorf("full name is required")
	}

	var existing Employee
	if err := a.db.First(&existing, "id = ?", employee.ID).Error; err != nil {
		return Employee{}, fmt.Errorf("employee not found: %w", err)
	}

	if employee.PreferredName == "" {
		employee.PreferredName = firstToken(employee.FullName)
	}
	if employee.EmploymentStatus == "" {
		employee.EmploymentStatus = existing.EmploymentStatus
	}
	if employee.EmployeeCode == "" {
		employee.EmployeeCode = existing.EmployeeCode
	}

	updates := map[string]any{
		"employee_code":       employee.EmployeeCode,
		"full_name":           employee.FullName,
		"preferred_name":      strings.TrimSpace(employee.PreferredName),
		"email":               strings.TrimSpace(employee.Email),
		"phone":               strings.TrimSpace(employee.Phone),
		"department":          strings.TrimSpace(employee.Department),
		"job_title":           strings.TrimSpace(employee.JobTitle),
		"employment_status":   strings.TrimSpace(employee.EmploymentStatus),
		"manager_employee_id": employee.ManagerEmployeeID,
		"start_date":          employee.StartDate,
		"end_date":            employee.EndDate,
		"emergency_contact":   strings.TrimSpace(employee.EmergencyContact),
		"notes":               strings.TrimSpace(employee.Notes),
		"is_active":           employee.IsActive,
		"updated_at":          time.Now(),
	}

	if err := a.db.Model(&existing).Updates(updates).Error; err != nil {
		return Employee{}, fmt.Errorf("failed to update employee profile: %w", err)
	}

	existing.EmployeeCode = employee.EmployeeCode
	existing.FullName = employee.FullName
	existing.PreferredName = strings.TrimSpace(employee.PreferredName)
	existing.Email = strings.TrimSpace(employee.Email)
	existing.Phone = strings.TrimSpace(employee.Phone)
	existing.Department = strings.TrimSpace(employee.Department)
	existing.JobTitle = strings.TrimSpace(employee.JobTitle)
	existing.EmploymentStatus = strings.TrimSpace(employee.EmploymentStatus)
	existing.ManagerEmployeeID = employee.ManagerEmployeeID
	existing.StartDate = employee.StartDate
	existing.EndDate = employee.EndDate
	existing.EmergencyContact = strings.TrimSpace(employee.EmergencyContact)
	existing.Notes = strings.TrimSpace(employee.Notes)
	existing.IsActive = employee.IsActive

	if payload, err := json.Marshal(existing); err == nil {
		a.enqueueCollaborativeOperation("employee", existing.ID, "update", string(payload))
	}
	a.emitCollaborationEvent("employees:updated", map[string]any{
		"employee_id": existing.ID,
		"action":      "update",
	})
	a.queueCollaborativeSync("employee_update")
	return existing, nil
}

func (a *App) SetEmployeeEmploymentState(employeeID string, isActive bool, employmentStatus string) (Employee, error) {
	if err := a.requirePermission("hr:update"); err != nil {
		return Employee{}, err
	}
	if a.db == nil {
		return Employee{}, fmt.Errorf("database not initialized")
	}

	employeeID = strings.TrimSpace(employeeID)
	if employeeID == "" {
		return Employee{}, fmt.Errorf("employee id is required")
	}
	employmentStatus = strings.TrimSpace(employmentStatus)
	if employmentStatus == "" {
		if isActive {
			employmentStatus = "active"
		} else {
			employmentStatus = "inactive"
		}
	}

	var employee Employee
	if err := a.db.First(&employee, "id = ?", employeeID).Error; err != nil {
		return Employee{}, fmt.Errorf("employee not found: %w", err)
	}

	var endDate *time.Time
	if !isActive {
		now := time.Now()
		endDate = &now
	} else {
		endDate = nil
	}

	if err := a.db.Model(&employee).Updates(map[string]any{
		"is_active":         isActive,
		"employment_status": employmentStatus,
		"end_date":          endDate,
		"updated_at":        time.Now(),
	}).Error; err != nil {
		return Employee{}, fmt.Errorf("failed to update employment state: %w", err)
	}

	employee.IsActive = isActive
	employee.EmploymentStatus = employmentStatus
	employee.EndDate = endDate

	if payload, err := json.Marshal(employee); err == nil {
		a.enqueueCollaborativeOperation("employee", employee.ID, "state_update", string(payload))
	}
	a.emitCollaborationEvent("employees:updated", map[string]any{
		"employee_id": employee.ID,
		"action":      "state_update",
		"is_active":   isActive,
	})
	a.queueCollaborativeSync("employee_state_update")
	return employee, nil
}

func (a *App) ReassignEmployeeManager(employeeID, managerEmployeeID string) (Employee, error) {
	if err := a.requirePermission("hr:update"); err != nil {
		return Employee{}, err
	}
	if a.db == nil {
		return Employee{}, fmt.Errorf("database not initialized")
	}

	employeeID = strings.TrimSpace(employeeID)
	managerEmployeeID = strings.TrimSpace(managerEmployeeID)
	if employeeID == "" {
		return Employee{}, fmt.Errorf("employee id is required")
	}
	if employeeID == managerEmployeeID && managerEmployeeID != "" {
		return Employee{}, fmt.Errorf("employee cannot manage themselves")
	}

	var employee Employee
	if err := a.db.First(&employee, "id = ?", employeeID).Error; err != nil {
		return Employee{}, fmt.Errorf("employee not found: %w", err)
	}

	var managerID *string
	if managerEmployeeID != "" {
		var manager Employee
		if err := a.db.First(&manager, "id = ?", managerEmployeeID).Error; err != nil {
			return Employee{}, fmt.Errorf("manager not found: %w", err)
		}
		managerID = &manager.ID
		employee.ManagerName = manager.FullName
	}

	if err := a.db.Model(&employee).Updates(map[string]any{
		"manager_employee_id": managerID,
		"updated_at":          time.Now(),
	}).Error; err != nil {
		return Employee{}, fmt.Errorf("failed to reassign employee manager: %w", err)
	}

	employee.ManagerEmployeeID = managerID
	if managerID == nil {
		employee.ManagerName = ""
	}

	if payload, err := json.Marshal(employee); err == nil {
		a.enqueueCollaborativeOperation("employee", employee.ID, "manager_update", string(payload))
	}
	a.emitCollaborationEvent("employees:updated", map[string]any{
		"employee_id": employee.ID,
		"action":      "manager_update",
	})
	a.queueCollaborativeSync("employee_manager_update")
	return employee, nil
}

func (a *App) ListEmployeeProjectAssignments(employeeID string) ([]ProjectMember, error) {
	if err := a.requirePermission("hr:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	employeeID = strings.TrimSpace(employeeID)
	if employeeID == "" {
		return []ProjectMember{}, nil
	}

	var assignments []ProjectMember
	if err := a.db.Where("employee_id = ?", employeeID).Order("is_active DESC, updated_at DESC").Find(&assignments).Error; err != nil {
		return nil, fmt.Errorf("failed to list employee project assignments: %w", err)
	}

	if len(assignments) == 0 {
		return []ProjectMember{}, nil
	}

	projectIDs := make([]string, 0, len(assignments))
	projectNames := map[string]string{}
	for _, assignment := range assignments {
		if assignment.ProjectID != "" {
			projectIDs = append(projectIDs, assignment.ProjectID)
		}
	}

	var projects []Project
	if err := a.db.Select("id", "name").Where("id IN ?", projectIDs).Find(&projects).Error; err == nil {
		for _, project := range projects {
			projectNames[project.ID] = project.Name
		}
	}

	for i := range assignments {
		assignments[i].ProjectName = projectNames[assignments[i].ProjectID]
	}

	return assignments, nil
}

func (a *App) ListEmployeeContributionSummaries() ([]EmployeeContributionSummary, error) {
	if err := a.requirePermission("hr:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var employees []Employee
	if err := a.db.Order("is_active DESC, full_name ASC").Find(&employees).Error; err != nil {
		return nil, fmt.Errorf("failed to load employees: %w", err)
	}
	if len(employees) == 0 {
		return []EmployeeContributionSummary{}, nil
	}

	summaries := make([]EmployeeContributionSummary, 0, len(employees))
	employeeIDs := make([]string, 0, len(employees))
	managerIDs := make([]string, 0)
	seenManagers := map[string]struct{}{}
	for _, employee := range employees {
		employeeIDs = append(employeeIDs, employee.ID)
		if employee.ManagerEmployeeID != nil && *employee.ManagerEmployeeID != "" {
			if _, ok := seenManagers[*employee.ManagerEmployeeID]; !ok {
				seenManagers[*employee.ManagerEmployeeID] = struct{}{}
				managerIDs = append(managerIDs, *employee.ManagerEmployeeID)
			}
		}
	}

	managerNames := a.lookupEmployeeNames(managerIDs)
	projectCounts := map[string]int{}
	taskCounts := map[string]int{}
	completedCounts := map[string]int{}
	blockedCounts := map[string]int{}
	overdueCounts := map[string]int{}
	licenseByEmployee := map[string]string{}
	deviceByEmployee := map[string]string{}
	opportunityCounts := map[string]int{}
	wonOpportunityCounts := map[string]int{}
	lostOpportunityCounts := map[string]int{}
	revenueYTD := map[string]float64{}

	var projectMembers []ProjectMember
	if err := a.db.Where("employee_id IN ? AND is_active = ?", employeeIDs, true).Find(&projectMembers).Error; err == nil {
		for _, member := range projectMembers {
			projectCounts[member.EmployeeID]++
		}
	}

	var tasks []TaskItem
	if err := a.db.Where("assignee_employee_id IN ?", employeeIDs).Find(&tasks).Error; err == nil {
		now := time.Now()
		for _, task := range tasks {
			if task.AssigneeEmployeeID == nil || *task.AssigneeEmployeeID == "" {
				continue
			}
			assigneeID := *task.AssigneeEmployeeID
			switch task.Status {
			case "completed", "archived":
				completedCounts[assigneeID]++
			default:
				taskCounts[assigneeID]++
				if task.Status == "blocked" {
					blockedCounts[assigneeID]++
				}
				if task.DueDate != nil && task.DueDate.Before(now) {
					overdueCounts[assigneeID]++
				}
			}
		}
	}

	employeeLookup := map[string]string{}
	for _, employee := range employees {
		for _, candidate := range []string{employee.ID, employee.EmployeeCode, employee.FullName, employee.PreferredName, employee.Email} {
			key := normalizeEmployeeLookupName(candidate)
			if key != "" {
				employeeLookup[key] = employee.ID
			}
		}
	}
	ytdStart := time.Date(time.Now().Year(), time.January, 1, 0, 0, 0, 0, time.Local)
	ytdEnd := ytdStart.AddDate(1, 0, 0)
	var opportunities []Opportunity
	if err := a.db.Where(
		"(offer_date >= ? AND offer_date < ?) OR (created_at >= ? AND created_at < ?)",
		ytdStart, ytdEnd, ytdStart, ytdEnd,
	).Find(&opportunities).Error; err == nil {
		for _, opportunity := range opportunities {
			matchedEmployeeID := ""
			for _, candidate := range []string{opportunity.Salesperson, opportunity.CreatedBy} {
				if employeeID := employeeLookup[normalizeEmployeeLookupName(candidate)]; employeeID != "" {
					matchedEmployeeID = employeeID
					break
				}
			}
			if matchedEmployeeID == "" {
				continue
			}
			opportunityCounts[matchedEmployeeID]++
			stage := strings.ToLower(strings.TrimSpace(opportunity.Stage))
			switch {
			case closedWonStage(stage):
				wonOpportunityCounts[matchedEmployeeID]++
				revenueYTD[matchedEmployeeID] += opportunity.RevenueBHD
			case stage == "lost":
				lostOpportunityCounts[matchedEmployeeID]++
			}
		}
	}

	var links []EmployeeAccessLink
	if err := a.db.Where("employee_id IN ?", employeeIDs).Order("is_primary DESC, updated_at DESC").Find(&links).Error; err == nil {
		deviceIDs := make([]string, 0)
		seenDeviceIDs := map[string]struct{}{}
		for _, link := range links {
			if _, ok := licenseByEmployee[link.EmployeeID]; !ok {
				licenseByEmployee[link.EmployeeID] = link.LicenseKey
			}
			if link.DeviceID != "" {
				if _, ok := seenDeviceIDs[link.DeviceID]; !ok {
					seenDeviceIDs[link.DeviceID] = struct{}{}
					deviceIDs = append(deviceIDs, link.DeviceID)
				}
			}
		}
		deviceNames := map[string]string{}
		if len(deviceIDs) > 0 {
			var devices []Device
			if err := a.db.Select("id", "device_name").Where("id IN ?", deviceIDs).Find(&devices).Error; err == nil {
				for _, device := range devices {
					deviceNames[device.ID] = device.DeviceName
				}
			}
		}
		for _, link := range links {
			if _, ok := deviceByEmployee[link.EmployeeID]; ok {
				continue
			}
			if link.DeviceID != "" {
				deviceByEmployee[link.EmployeeID] = deviceNames[link.DeviceID]
			}
		}
	}

	for _, employee := range employees {
		activeTaskCount := taskCounts[employee.ID]
		completedTaskCount := completedCounts[employee.ID]
		totalTracked := activeTaskCount + completedTaskCount
		completionRate := 0.0
		if totalTracked > 0 {
			completionRate = (float64(completedTaskCount) / float64(totalTracked)) * 100
		}

		managerName := ""
		managerID := ""
		if employee.ManagerEmployeeID != nil {
			managerID = *employee.ManagerEmployeeID
			managerName = managerNames[managerID]
		}

		summaries = append(summaries, EmployeeContributionSummary{
			EmployeeID:         employee.ID,
			EmployeeCode:       employee.EmployeeCode,
			EmployeeName:       employee.FullName,
			Department:         employee.Department,
			JobTitle:           employee.JobTitle,
			ManagerEmployeeID:  managerID,
			ManagerName:        managerName,
			EmploymentStatus:   employee.EmploymentStatus,
			IsActive:           employee.IsActive,
			ActiveProjectCount: projectCounts[employee.ID],
			ActiveTaskCount:    activeTaskCount,
			CompletedTaskCount: completedTaskCount,
			BlockedTaskCount:   blockedCounts[employee.ID],
			OverdueTaskCount:   overdueCounts[employee.ID],
			CompletionRate:     completionRate,
			OpportunityYTD:     opportunityCounts[employee.ID],
			OpportunityWonYTD:  wonOpportunityCounts[employee.ID],
			OpportunityLostYTD: lostOpportunityCounts[employee.ID],
			RevenueYTD:         roundTo3(revenueYTD[employee.ID]),
			PrimaryLicenseKey:  licenseByEmployee[employee.ID],
			PrimaryDeviceName:  deviceByEmployee[employee.ID],
		})
	}

	return summaries, nil
}

func (a *App) CreateEmployeeAccessLink(link EmployeeAccessLink) (EmployeeAccessLink, error) {
	if err := a.requirePermission("hr:update"); err != nil {
		return EmployeeAccessLink{}, err
	}
	if a.db == nil {
		return EmployeeAccessLink{}, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(link.EmployeeID) == "" {
		return EmployeeAccessLink{}, fmt.Errorf("employee id is required")
	}
	if strings.TrimSpace(link.LicenseKey) == "" {
		return EmployeeAccessLink{}, fmt.Errorf("license key is required")
	}

	var employee Employee
	if err := a.db.First(&employee, "id = ?", link.EmployeeID).Error; err != nil {
		return EmployeeAccessLink{}, fmt.Errorf("employee not found: %w", err)
	}

	var license LicenseKey
	if err := a.db.Where("key = ?", strings.ToUpper(strings.TrimSpace(link.LicenseKey))).First(&license).Error; err != nil {
		return EmployeeAccessLink{}, fmt.Errorf("license key not found: %w", err)
	}

	builtLink, err := a.buildAccessLinkForLicense(link.EmployeeID, license)
	if err != nil {
		return EmployeeAccessLink{}, err
	}
	if link.UserID != "" {
		builtLink.UserID = link.UserID
	}
	if link.DeviceID != "" {
		builtLink.DeviceID = link.DeviceID
	}
	if link.AccessStatus != "" {
		builtLink.AccessStatus = link.AccessStatus
	}
	builtLink.IsPrimary = true
	builtLink.CreatedBy = a.getCurrentUserID()

	if builtLink.IsPrimary {
		_ = a.db.Model(&EmployeeAccessLink{}).Where("employee_id = ?", link.EmployeeID).Update("is_primary", false).Error
	}

	var existing EmployeeAccessLink
	if err := a.db.Where("employee_id = ? AND license_key = ?", builtLink.EmployeeID, builtLink.LicenseKey).First(&existing).Error; err == nil {
		updates := map[string]any{
			"user_id":       builtLink.UserID,
			"device_id":     builtLink.DeviceID,
			"access_status": builtLink.AccessStatus,
			"is_primary":    builtLink.IsPrimary,
			"updated_at":    time.Now(),
		}
		if err := a.db.Model(&existing).Updates(updates).Error; err != nil {
			return EmployeeAccessLink{}, fmt.Errorf("failed to update employee access link: %w", err)
		}
		existing.UserID = builtLink.UserID
		existing.DeviceID = builtLink.DeviceID
		existing.AccessStatus = builtLink.AccessStatus
		existing.IsPrimary = builtLink.IsPrimary
		if payload, err := json.Marshal(existing); err == nil {
			a.enqueueCollaborativeOperation("employee_access_link", existing.ID, "update", string(payload))
		}
		a.emitCollaborationEvent("employees:updated", map[string]any{
			"employee_id": existing.EmployeeID,
			"action":      "access_link_update",
		})
		a.queueCollaborativeSync("employee_access_link_update")
		return existing, nil
	}

	if err := a.db.Create(&builtLink).Error; err != nil {
		return EmployeeAccessLink{}, fmt.Errorf("failed to create employee access link: %w", err)
	}
	if payload, err := json.Marshal(builtLink); err == nil {
		a.enqueueCollaborativeOperation("employee_access_link", builtLink.ID, "create", string(payload))
	}
	a.emitCollaborationEvent("employees:updated", map[string]any{
		"employee_id": builtLink.EmployeeID,
		"action":      "access_link_create",
	})
	a.queueCollaborativeSync("employee_access_link_create")
	return builtLink, nil
}

func (a *App) ReassignEmployeeLicenseAccess(employeeID, licenseKey string, syncDisplayName bool) (EmployeeAccessLink, error) {
	if err := a.requirePermission("hr:update"); err != nil {
		return EmployeeAccessLink{}, err
	}
	if a.db == nil {
		return EmployeeAccessLink{}, fmt.Errorf("database not initialized")
	}

	employeeID = strings.TrimSpace(employeeID)
	licenseKey = strings.ToUpper(strings.TrimSpace(licenseKey))
	if employeeID == "" {
		return EmployeeAccessLink{}, fmt.Errorf("employee id is required")
	}
	if licenseKey == "" {
		return EmployeeAccessLink{}, fmt.Errorf("license key is required")
	}

	var employee Employee
	if err := a.db.First(&employee, "id = ?", employeeID).Error; err != nil {
		return EmployeeAccessLink{}, fmt.Errorf("employee not found: %w", err)
	}

	var license LicenseKey
	if err := a.db.Where("key = ?", licenseKey).First(&license).Error; err != nil {
		return EmployeeAccessLink{}, fmt.Errorf("license key not found: %w", err)
	}

	builtLink, err := a.buildAccessLinkForLicense(employee.ID, license)
	if err != nil {
		return EmployeeAccessLink{}, err
	}
	builtLink.AccessStatus = "active"
	builtLink.IsPrimary = true
	builtLink.CreatedBy = a.getCurrentUserID()

	now := time.Now()
	var updatedLink EmployeeAccessLink
	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&EmployeeAccessLink{}).
			Where("license_key = ? AND employee_id <> ? AND access_status = ?", license.Key, employee.ID, "active").
			Updates(map[string]any{
				"access_status": "inactive",
				"is_primary":    false,
				"updated_at":    now,
			}).Error; err != nil {
			return fmt.Errorf("failed to release existing license assignment: %w", err)
		}

		if err := tx.Model(&EmployeeAccessLink{}).
			Where("employee_id = ?", employee.ID).
			Updates(map[string]any{
				"is_primary": false,
				"updated_at": now,
			}).Error; err != nil {
			return fmt.Errorf("failed to update employee primary access links: %w", err)
		}

		var existing EmployeeAccessLink
		if err := tx.Where("employee_id = ? AND license_key = ?", employee.ID, license.Key).First(&existing).Error; err == nil {
			updates := map[string]any{
				"user_id":       builtLink.UserID,
				"device_id":     builtLink.DeviceID,
				"access_status": builtLink.AccessStatus,
				"is_primary":    true,
				"updated_at":    now,
			}
			if err := tx.Model(&existing).Updates(updates).Error; err != nil {
				return fmt.Errorf("failed to update employee license access: %w", err)
			}
			existing.UserID = builtLink.UserID
			existing.DeviceID = builtLink.DeviceID
			existing.AccessStatus = builtLink.AccessStatus
			existing.IsPrimary = true
			updatedLink = existing
		} else if err == gorm.ErrRecordNotFound {
			if err := tx.Create(&builtLink).Error; err != nil {
				return fmt.Errorf("failed to create employee license access: %w", err)
			}
			updatedLink = builtLink
		} else {
			return fmt.Errorf("failed to lookup employee license access: %w", err)
		}

		if syncDisplayName {
			displayName := strings.TrimSpace(employee.PreferredName)
			if displayName == "" {
				displayName = firstToken(employee.FullName)
			}
			if displayName == "" {
				displayName = employee.FullName
			}
			if err := tx.Model(&license).Updates(map[string]any{
				"display_name": displayName,
				"notes":        fmt.Sprintf("Assigned to %s", employee.FullName),
			}).Error; err != nil {
				return fmt.Errorf("failed to sync license display name: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return EmployeeAccessLink{}, err
	}

	if payload, err := json.Marshal(updatedLink); err == nil {
		a.enqueueCollaborativeOperation("employee_access_link", updatedLink.ID, "update", string(payload))
	}
	a.emitCollaborationEvent("employees:updated", map[string]any{
		"employee_id": employee.ID,
		"action":      "license_reassigned",
		"license_key": license.Key,
	})
	a.queueCollaborativeSync("employee_license_reassign")
	return updatedLink, nil
}

func (a *App) ListEmployeeAccessLinks() ([]EmployeeAccessLink, error) {
	if err := a.requirePermission("hr:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var links []EmployeeAccessLink
	if err := a.db.Order("created_at DESC").Find(&links).Error; err != nil {
		return nil, fmt.Errorf("failed to list employee access links: %w", err)
	}

	employees := map[string]string{}
	devices := map[string]string{}
	for i := range links {
		if links[i].EmployeeID != "" {
			if _, ok := employees[links[i].EmployeeID]; !ok {
				var employee Employee
				if err := a.db.First(&employee, "id = ?", links[i].EmployeeID).Error; err == nil {
					employees[links[i].EmployeeID] = employee.FullName
				}
			}
			links[i].EmployeeName = employees[links[i].EmployeeID]
		}
		if links[i].DeviceID != "" {
			if _, ok := devices[links[i].DeviceID]; !ok {
				var device Device
				if err := a.db.First(&device, "id = ?", links[i].DeviceID).Error; err == nil {
					devices[links[i].DeviceID] = device.DeviceName
				}
			}
			links[i].DeviceName = devices[links[i].DeviceID]
		}
	}

	return links, nil
}

func (a *App) GetUnreadNotificationsCount() (int, error) {
	if a.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return 0, nil
	}

	var count int64
	if err := a.db.Model(&Notification{}).
		Where("employee_id = ? AND status = ?", current.EmployeeID, "unread").
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}
	return int(count), nil
}

func (a *App) ListNotificationFeed(limit int, unreadOnly bool) ([]Notification, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 50
	}

	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return []Notification{}, nil
	}

	query := a.db.Where("employee_id = ?", current.EmployeeID).Order("created_at DESC").Limit(limit)
	if unreadOnly {
		query = query.Where("status = ?", "unread")
	}

	var notifications []Notification
	if err := query.Find(&notifications).Error; err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	return notifications, nil
}

func (a *App) MarkNotificationAsRead(notificationID string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return err
	}

	var notification Notification
	if err := a.db.First(&notification, "id = ? AND employee_id = ?", notificationID, current.EmployeeID).Error; err != nil {
		return fmt.Errorf("notification not found: %w", err)
	}

	now := time.Now()
	if err := a.db.Model(&notification).Updates(map[string]any{
		"status":  "read",
		"read_at": &now,
	}).Error; err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	a.recordNotificationReceipt(notification.ID, current.EmployeeID, current.DeviceID, "read")
	if payload, err := json.Marshal(notification); err == nil {
		a.enqueueCollaborativeOperation("notification", notification.ID, "read", string(payload))
	}
	a.emitCollaborationEvent("notifications:updated", map[string]any{
		"notification_id": notification.ID,
		"employee_id":     current.EmployeeID,
		"status":          "read",
	})
	a.queueCollaborativeSync("notification_read")
	return nil
}

func (a *App) CreateCollaborativeProject(project Project) (Project, error) {
	if err := a.requirePermission("projects:create"); err != nil {
		return Project{}, err
	}
	if a.db == nil {
		return Project{}, fmt.Errorf("database not initialized")
	}
	project.Name = strings.TrimSpace(project.Name)
	if project.Name == "" {
		return Project{}, fmt.Errorf("project name is required")
	}
	if project.ProjectType == "" {
		project.ProjectType = "internal"
	}
	if project.Status == "" {
		project.Status = "active"
	}
	project.CustomerName = strings.TrimSpace(project.CustomerName)
	project.EndUserName = strings.TrimSpace(project.EndUserName)
	project.OpportunityKey = strings.TrimSpace(project.OpportunityKey)
	project.CustomerPOCName = strings.TrimSpace(project.CustomerPOCName)
	project.CustomerPOCEmail = strings.TrimSpace(project.CustomerPOCEmail)
	project.CustomerPOCPhone = strings.TrimSpace(project.CustomerPOCPhone)
	project.CreatedBy = a.getCurrentUserID()
	if err := a.db.Create(&project).Error; err != nil {
		return Project{}, fmt.Errorf("failed to create project: %w", err)
	}
	if payload, err := json.Marshal(project); err == nil {
		a.enqueueCollaborativeOperation("project", project.ID, "create", string(payload))
	}
	a.emitCollaborationEvent("projects:updated", map[string]any{
		"project_id": project.ID,
		"action":     "create",
	})
	a.queueCollaborativeSync("project_create")
	return project, nil
}

func (a *App) ListCollaborativeProjects(activeOnly bool) ([]Project, error) {
	if err := a.requirePermission("projects:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	query := a.db.Order("updated_at DESC")
	if activeOnly {
		// PH parity: exclude all three terminal statuses, case-insensitively and
		// NULL-safely (status != 'archived' silently drops NULL-status rows).
		query = query.Where("LOWER(COALESCE(status, '')) NOT IN ?", []string{"archived", "shelved", "deleted"})
	}
	var projects []Project
	if err := query.Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	return projects, nil
}

// Wave 8 P4 slice 7 (Bucket G): project lifecycle tail ported from the frozen
// PH reference — whitelisted update plus archive/shelve/delete status wrappers.
// Terminal statuses (archived/shelved/deleted) escalate to projects:delete,
// which only the admin wildcard grants.
func (a *App) UpdateCollaborativeProject(projectID string, updates map[string]any) (Project, error) {
	if err := a.requirePermission("projects:update"); err != nil {
		return Project{}, err
	}
	if a.db == nil {
		return Project{}, fmt.Errorf("database not initialized")
	}

	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return Project{}, fmt.Errorf("project id is required")
	}

	allowed := map[string]bool{
		"name": true, "project_type": true, "description": true, "status": true,
		"customer_id": true, "opportunity_id": true, "order_id": true,
		"customer_name": true, "end_user_name": true, "opportunity_key": true,
		"customer_poc_name": true, "customer_poc_email": true, "customer_poc_phone": true,
		"starts_on": true, "ends_on": true,
	}
	clean := make(map[string]any)
	for key, value := range updates {
		if !allowed[key] {
			continue
		}
		if text, ok := value.(string); ok {
			clean[key] = strings.TrimSpace(text)
			continue
		}
		clean[key] = value
	}
	if len(clean) == 0 {
		return Project{}, fmt.Errorf("no supported project updates supplied")
	}
	if name, ok := clean["name"].(string); ok && name == "" {
		return Project{}, fmt.Errorf("project name is required")
	}
	if status, ok := clean["status"].(string); ok {
		status = strings.ToLower(strings.TrimSpace(status))
		clean["status"] = status
		if status == "archived" || status == "shelved" || status == "deleted" {
			if err := a.requirePermission("projects:delete"); err != nil {
				return Project{}, fmt.Errorf("project archival/delete requires admin permission: %w", err)
			}
		}
	}
	clean["updated_at"] = time.Now()

	var project Project
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&project, "id = ?", projectID).Error; err != nil {
			return fmt.Errorf("project not found: %w", err)
		}
		if err := tx.Model(&Project{}).Where("id = ?", projectID).Updates(clean).Error; err != nil {
			return fmt.Errorf("failed to update project: %w", err)
		}
		return tx.First(&project, "id = ?", projectID).Error
	}); err != nil {
		return Project{}, err
	}

	if payload, err := json.Marshal(project); err == nil {
		a.enqueueCollaborativeOperation("project", project.ID, "update", string(payload))
	}
	a.emitCollaborationEvent("projects:updated", map[string]any{"project_id": project.ID, "action": "update"})
	a.queueCollaborativeSync("project_update")
	return project, nil
}

func (a *App) ArchiveCollaborativeProject(projectID, reason string) (Project, error) {
	return a.updateCollaborativeProjectStatus(projectID, "archived", reason)
}

func (a *App) ShelveCollaborativeProject(projectID, reason string) (Project, error) {
	return a.updateCollaborativeProjectStatus(projectID, "shelved", reason)
}

func (a *App) DeleteCollaborativeProject(projectID, reason string) error {
	if err := a.requirePermission("projects:delete"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return fmt.Errorf("project id is required")
	}
	archived, err := a.updateCollaborativeProjectStatus(projectID, "deleted", reason)
	if err != nil {
		return err
	}
	a.logAudit(nil, "delete", "project", &archived.ID, strings.TrimSpace(reason))
	return nil
}

func (a *App) updateCollaborativeProjectStatus(projectID, status, reason string) (Project, error) {
	project, err := a.UpdateCollaborativeProject(projectID, map[string]any{
		"status": status,
	})
	if err != nil {
		return Project{}, err
	}
	a.logAudit(nil, status, "project", &project.ID, strings.TrimSpace(reason))
	return project, nil
}

func (a *App) AddCollaborativeProjectMember(projectID, employeeID, role string, allocationPercent float64) (ProjectMember, error) {
	if err := a.requirePermission("projects:update"); err != nil {
		return ProjectMember{}, err
	}
	if a.db == nil {
		return ProjectMember{}, fmt.Errorf("database not initialized")
	}

	projectID = strings.TrimSpace(projectID)
	employeeID = strings.TrimSpace(employeeID)
	role = strings.TrimSpace(role)
	if projectID == "" {
		return ProjectMember{}, fmt.Errorf("project id is required")
	}
	if employeeID == "" {
		return ProjectMember{}, fmt.Errorf("employee id is required")
	}
	if role == "" {
		role = "Member"
	}
	allocationPercent = clampAllocationPercent(allocationPercent)

	var project Project
	if err := a.db.First(&project, "id = ?", projectID).Error; err != nil {
		return ProjectMember{}, fmt.Errorf("project not found: %w", err)
	}

	var employee Employee
	if err := a.db.First(&employee, "id = ?", employeeID).Error; err != nil {
		return ProjectMember{}, fmt.Errorf("employee not found: %w", err)
	}

	now := time.Now()
	member := ProjectMember{
		Base:              Base{CreatedBy: a.getCurrentUserID()},
		ProjectID:         projectID,
		EmployeeID:        employeeID,
		Role:              role,
		AllocationPercent: allocationPercent,
		IsActive:          true,
		JoinedAt:          &now,
	}

	var existing ProjectMember
	if err := a.db.Where("project_id = ? AND employee_id = ?", projectID, employeeID).First(&existing).Error; err == nil {
		updates := map[string]any{
			"role":               role,
			"is_active":          true,
			"allocation_percent": allocationPercent,
			"left_at":            nil,
			"updated_at":         time.Now(),
		}
		if existing.JoinedAt == nil {
			updates["joined_at"] = &now
		}
		if err := a.db.Model(&existing).Updates(updates).Error; err != nil {
			return ProjectMember{}, fmt.Errorf("failed to update project member: %w", err)
		}
		existing.Role = role
		existing.IsActive = true
		existing.AllocationPercent = allocationPercent
		existing.LeftAt = nil
		existing.EmployeeName = employee.FullName
		if payload, err := json.Marshal(existing); err == nil {
			a.enqueueCollaborativeOperation("project_member", existing.ID, "update", string(payload))
		}
		a.emitCollaborationEvent("projects:updated", map[string]any{
			"project_id":  projectID,
			"employee_id": employeeID,
			"action":      "member_update",
		})
		a.createProjectMemberNotification(employee.ID, project, currentProjectActorName(a), role, "updated")
		a.queueCollaborativeSync("project_member_update")
		return existing, nil
	}

	if err := a.db.Create(&member).Error; err != nil {
		return ProjectMember{}, fmt.Errorf("failed to add project member: %w", err)
	}
	member.EmployeeName = employee.FullName
	if payload, err := json.Marshal(member); err == nil {
		a.enqueueCollaborativeOperation("project_member", member.ID, "create", string(payload))
	}
	a.emitCollaborationEvent("projects:updated", map[string]any{
		"project_id":  projectID,
		"employee_id": employeeID,
		"action":      "member_create",
	})
	a.createProjectMemberNotification(employee.ID, project, currentProjectActorName(a), role, "added")
	a.queueCollaborativeSync("project_member_create")
	return member, nil
}

// clampAllocationPercent normalizes a requested allocation into the valid
// 0-100 range. A non-positive (zero/omitted/negative) value defaults to a
// full-time 100 — matching the prior hardcoded behavior for callers that
// don't care about partial allocation.
func clampAllocationPercent(value float64) float64 {
	if value <= 0 {
		return 100
	}
	if value > 100 {
		return 100
	}
	return value
}

// AllocationProjectLine names one of an employee's other active project
// memberships, for display in the Wave 9.8 B3 over-allocation warning.
type AllocationProjectLine struct {
	ProjectID         string  `json:"project_id"`
	ProjectName       string  `json:"project_name"`
	AllocationPercent float64 `json:"allocation_percent"`
}

// AllocationSummary is the read-only response for GetEmployeeAllocationSummary.
// OtherProjectsTotal is computed server-side (never trust a client-side sum)
// and excludes the project currently being edited.
type AllocationSummary struct {
	EmployeeID         string                  `json:"employee_id"`
	OtherProjectsTotal float64                 `json:"other_projects_total"`
	Projects           []AllocationProjectLine `json:"projects"`
}

// GetEmployeeAllocationSummary reports how much of an employee's capacity is
// already committed to OTHER active projects (excludeProjectID is left out of
// the total so a save-in-progress edit doesn't double-count itself). This is
// read-only and side-effect free: it exists so the UI can WARN — never
// block — when a save would push an employee over 100% allocation. Wave 9.8
// B3: allocation capacity is advisory, not a hard cap.
func (a *App) GetEmployeeAllocationSummary(employeeID string, excludeProjectID string) (AllocationSummary, error) {
	if err := a.requirePermission("projects:view"); err != nil {
		return AllocationSummary{}, err
	}
	if a.db == nil {
		return AllocationSummary{}, fmt.Errorf("database not initialized")
	}

	employeeID = strings.TrimSpace(employeeID)
	excludeProjectID = strings.TrimSpace(excludeProjectID)
	if employeeID == "" {
		return AllocationSummary{}, fmt.Errorf("employee id is required")
	}

	summary := AllocationSummary{EmployeeID: employeeID, Projects: []AllocationProjectLine{}}

	query := a.db.Model(&ProjectMember{}).
		Joins("JOIN projects ON projects.id = project_members.project_id").
		Where("project_members.employee_id = ? AND project_members.is_active = ? AND projects.status = ?", employeeID, true, "active")
	if excludeProjectID != "" {
		query = query.Where("project_members.project_id != ?", excludeProjectID)
	}

	if err := query.Session(&gorm.Session{}).
		Select("COALESCE(SUM(project_members.allocation_percent), 0)").
		Scan(&summary.OtherProjectsTotal).Error; err != nil {
		return AllocationSummary{}, fmt.Errorf("failed to compute allocation total: %w", err)
	}

	type projectLineRow struct {
		ProjectID         string
		ProjectName       string
		AllocationPercent float64
	}
	var rows []projectLineRow
	if err := query.Session(&gorm.Session{}).
		Select("project_members.project_id AS project_id, projects.name AS project_name, project_members.allocation_percent AS allocation_percent").
		Scan(&rows).Error; err != nil {
		return AllocationSummary{}, fmt.Errorf("failed to list allocation projects: %w", err)
	}
	for _, r := range rows {
		summary.Projects = append(summary.Projects, AllocationProjectLine(r))
	}

	return summary, nil
}

// GetProjectTaskCounts returns a total task count (all statuses) per project,
// keyed by project id, for lightweight list-row badges (Wave 9.4 B3.3) —
// avoids loading every task body just to count them.
func (a *App) GetProjectTaskCounts() (map[string]int, error) {
	if err := a.requirePermission("projects:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	type projectTaskCountRow struct {
		ProjectID string
		Count     int
	}
	var rows []projectTaskCountRow
	if err := a.db.Model(&TaskItem{}).
		Select("project_id, COUNT(*) as count").
		Where("project_id IS NOT NULL AND project_id != ''").
		Group("project_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to count project tasks: %w", err)
	}

	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.ProjectID] = row.Count
	}
	return counts, nil
}

func (a *App) ListCollaborativeProjectMembers(projectID string) ([]ProjectMember, error) {
	if err := a.requirePermission("projects:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(projectID) == "" {
		return []ProjectMember{}, nil
	}

	var members []ProjectMember
	if err := a.db.Where("project_id = ? AND is_active = ?", projectID, true).Order("created_at ASC").Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to list project members: %w", err)
	}
	employeeNames := a.lookupEmployeeNames(memberEmployeeIDs(members))
	for i := range members {
		members[i].EmployeeName = employeeNames[members[i].EmployeeID]
	}
	return members, nil
}

func (a *App) CreateCollaborativeTask(task TaskItem) (TaskItem, error) {
	if err := a.requirePermission("tasks:create"); err != nil {
		return TaskItem{}, err
	}
	if a.db == nil {
		return TaskItem{}, fmt.Errorf("database not initialized")
	}

	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return TaskItem{}, err
	}

	task.Title = strings.TrimSpace(task.Title)
	if task.Title == "" {
		return TaskItem{}, fmt.Errorf("task title is required")
	}
	if task.TaskType == "" {
		task.TaskType = "general"
	}
	if task.Status == "" {
		task.Status = "open"
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	task.CreatorEmployeeID = current.EmployeeID
	task.CreatedBy = current.EmployeeID

	if err := a.db.Create(&task).Error; err != nil {
		return TaskItem{}, fmt.Errorf("failed to create task: %w", err)
	}

	_ = a.createTaskActivity(task.ID, current.EmployeeID, "created", fmt.Sprintf("Created task %q", task.Title), nil)

	if task.AssigneeEmployeeID != nil && *task.AssigneeEmployeeID != "" {
		a.createTaskNotification(*task.AssigneeEmployeeID, task, current.EmployeeName, "assigned")
	}

	if payload, err := json.Marshal(task); err == nil {
		a.enqueueCollaborativeOperation("task", task.ID, "create", string(payload))
	}

	a.emitCollaborationEvent("tasks:updated", map[string]any{
		"task_id":     task.ID,
		"employee_id": task.AssigneeEmployeeID,
		"action":      "create",
	})
	a.queueCollaborativeSync("task_create")

	return a.decorateTask(task), nil
}

func normalizeCollaborativeTaskPriority(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "low", "medium", "high", "urgent":
		return strings.ToLower(strings.TrimSpace(priority))
	default:
		return "medium"
	}
}

func (a *App) UpdateCollaborativeTask(task TaskItem) (TaskItem, error) {
	if err := a.requirePermission("tasks:update"); err != nil {
		return TaskItem{}, err
	}
	if a.db == nil {
		return TaskItem{}, fmt.Errorf("database not initialized")
	}

	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return TaskItem{}, err
	}

	task.ID = strings.TrimSpace(task.ID)
	if task.ID == "" {
		return TaskItem{}, fmt.Errorf("task id is required")
	}

	var existing TaskItem
	if err := a.db.First(&existing, "id = ?", task.ID).Error; err != nil {
		return TaskItem{}, fmt.Errorf("task not found: %w", err)
	}

	title := strings.TrimSpace(task.Title)
	if title == "" {
		return TaskItem{}, fmt.Errorf("task title is required")
	}
	description := strings.TrimSpace(task.Description)
	priority := normalizeCollaborativeTaskPriority(task.Priority)
	taskType := strings.TrimSpace(task.TaskType)
	if taskType == "" {
		taskType = existing.TaskType
	}
	projectID := task.ProjectID
	if projectID != nil {
		trimmed := strings.TrimSpace(*projectID)
		if trimmed == "" {
			projectID = nil
		} else {
			projectID = &trimmed
		}
	}

	updates := map[string]any{
		"title":       title,
		"description": description,
		"priority":    priority,
		"task_type":   taskType,
		"project_id":  projectID,
	}

	if err := a.db.Model(&existing).Updates(updates).Error; err != nil {
		return TaskItem{}, fmt.Errorf("failed to update task: %w", err)
	}

	meta := map[string]any{
		"title":       title,
		"description": description,
		"priority":    priority,
		"task_type":   taskType,
	}
	_ = a.createTaskActivity(existing.ID, current.EmployeeID, "updated", fmt.Sprintf("Updated task %q", title), meta)

	if payload, err := json.Marshal(meta); err == nil {
		a.enqueueCollaborativeOperation("task", existing.ID, "update", string(payload))
	}

	a.emitCollaborationEvent("tasks:updated", map[string]any{
		"task_id": existing.ID,
		"action":  "update",
	})
	a.queueCollaborativeSync("task_update")

	updated, err := a.GetCollaborativeTask(existing.ID)
	if err != nil {
		return TaskItem{}, err
	}
	return *updated, nil
}

func (a *App) DeleteCollaborativeTask(taskID string) error {
	if ok, err := a.guardDeleteOrRequest("tasks:delete", "collaborative_task", taskID, "Task"); !ok {
		return err
	}
	if err := a.requirePermission("tasks:delete"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return err
	}

	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}

	var task TaskItem
	if err := a.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if err := a.db.Delete(&task).Error; err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	meta := map[string]any{"title": task.Title}
	_ = a.createTaskActivity(task.ID, current.EmployeeID, "deleted", fmt.Sprintf("Deleted task %q", task.Title), meta)

	if payload, err := json.Marshal(meta); err == nil {
		a.enqueueCollaborativeOperation("task", task.ID, "delete", string(payload))
	}

	a.emitCollaborationEvent("tasks:updated", map[string]any{
		"task_id": task.ID,
		"action":  "delete",
	})
	a.queueCollaborativeSync("task_delete")
	return nil
}

func (a *App) ListMyCollaborativeTasks(includeCompleted bool) ([]TaskItem, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return nil, err
	}
	a.repairDanglingTaskAssignees()
	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return []TaskItem{}, nil
	}
	return a.ListCollaborativeTasksForEmployee(current.EmployeeID, includeCompleted)
}

func (a *App) ListCollaborativeTasksForEmployee(employeeID string, includeCompleted bool) ([]TaskItem, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	query := a.db.Where("assignee_employee_id = ?", employeeID).Order("due_date ASC, updated_at DESC")
	if !includeCompleted {
		query = query.Where("status NOT IN ?", []string{"completed", "archived"})
	}
	var tasks []TaskItem
	if err := query.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	return a.decorateTasks(tasks), nil
}

func (a *App) ListCollaborativeTeamTasks(includeCompleted bool) ([]TaskItem, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	a.repairDanglingTaskAssignees()
	query := a.db.Order("updated_at DESC")
	if !includeCompleted {
		query = query.Where("status NOT IN ?", []string{"completed", "archived"})
	}
	var tasks []TaskItem
	if err := query.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to list team tasks: %w", err)
	}
	return a.decorateTasks(tasks), nil
}

func (a *App) ListCollaborativeProjectTasks(projectID string, includeCompleted bool) ([]TaskItem, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	a.repairDanglingTaskAssignees()
	if strings.TrimSpace(projectID) == "" {
		return []TaskItem{}, nil
	}

	query := a.db.Where("project_id = ?", projectID).Order("updated_at DESC")
	if !includeCompleted {
		query = query.Where("status NOT IN ?", []string{"completed", "archived"})
	}
	var tasks []TaskItem
	if err := query.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to list project tasks: %w", err)
	}
	return a.decorateTasks(tasks), nil
}

func (a *App) ListCollaborativeProjectActivity(projectID string) ([]TaskActivity, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(projectID) == "" {
		return []TaskActivity{}, nil
	}

	var taskIDs []string
	if err := a.db.Model(&TaskItem{}).Where("project_id = ?", projectID).Pluck("id", &taskIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to load project task ids: %w", err)
	}
	if len(taskIDs) == 0 {
		return []TaskActivity{}, nil
	}

	var activity []TaskActivity
	if err := a.db.Where("task_id IN ?", taskIDs).Order("created_at DESC").Find(&activity).Error; err != nil {
		return nil, fmt.Errorf("failed to list project activity: %w", err)
	}
	employeeNames := a.lookupEmployeeNames(activityEmployeeIDs(activity))
	for i := range activity {
		activity[i].EmployeeName = employeeNames[activity[i].EmployeeID]
	}
	return activity, nil
}

func (a *App) GetCollaborativeTask(taskID string) (*TaskItem, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	a.repairDanglingTaskAssignees()
	var task TaskItem
	if err := a.db.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}
	decorated := a.decorateTask(task)
	return &decorated, nil
}

func (a *App) ListCollaborativeTaskComments(taskID string) ([]TaskComment, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(taskID) == "" {
		return []TaskComment{}, nil
	}

	var comments []TaskComment
	if err := a.db.Where("task_id = ?", taskID).Order("created_at ASC").Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("failed to list task comments: %w", err)
	}
	employeeNames := a.lookupEmployeeNames(commentEmployeeIDs(comments))
	for i := range comments {
		comments[i].EmployeeName = employeeNames[comments[i].EmployeeID]
	}
	return comments, nil
}

func (a *App) ListCollaborativeTaskActivity(taskID string) ([]TaskActivity, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(taskID) == "" {
		return []TaskActivity{}, nil
	}

	var activity []TaskActivity
	if err := a.db.Where("task_id = ?", taskID).Order("created_at DESC").Find(&activity).Error; err != nil {
		return nil, fmt.Errorf("failed to list task activity: %w", err)
	}
	employeeNames := a.lookupEmployeeNames(activityEmployeeIDs(activity))
	for i := range activity {
		activity[i].EmployeeName = employeeNames[activity[i].EmployeeID]
	}
	return activity, nil
}

func (a *App) UpdateCollaborativeTaskStatus(taskID, status, note string) error {
	if err := a.requirePermission("tasks:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return err
	}

	status = strings.ToLower(strings.TrimSpace(status))
	note = strings.TrimSpace(note)
	if status == "" {
		return fmt.Errorf("status is required")
	}

	var task TaskItem
	if err := a.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	updates := map[string]any{
		"status": status,
	}
	now := time.Now()
	switch status {
	case "in_progress":
		updates["started_at"] = &now
		updates["blocked_reason"] = ""
	case "completed":
		updates["completed_at"] = &now
		updates["blocked_reason"] = ""
	case "open", "cancelled":
		updates["blocked_reason"] = ""
	case "blocked":
		if note == "" {
			return fmt.Errorf("blocked reason is required")
		}
		updates["blocked_reason"] = note
	default:
		return fmt.Errorf("unsupported task status: %s", status)
	}

	if err := a.db.Model(&task).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	meta := map[string]any{"note": note, "status": status}
	_ = a.createTaskActivity(task.ID, current.EmployeeID, "status_changed", fmt.Sprintf("Changed status to %s", status), meta)

	if task.CreatorEmployeeID != "" && task.CreatorEmployeeID != current.EmployeeID {
		a.createTaskNotification(task.CreatorEmployeeID, task, current.EmployeeName, "status_changed")
	}

	if payload, err := json.Marshal(meta); err == nil {
		a.enqueueCollaborativeOperation("task", task.ID, "status_update", string(payload))
	}

	a.emitCollaborationEvent("tasks:updated", map[string]any{
		"task_id": task.ID,
		"status":  status,
		"action":  "status_update",
	})
	a.queueCollaborativeSync("task_status_update")

	return nil
}

func (a *App) ReassignCollaborativeTask(taskID, assigneeEmployeeID string) error {
	if err := a.requirePermission("tasks:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return err
	}

	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}

	var task TaskItem
	if err := a.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	var newAssigneeID *string
	var newAssigneeName string
	assigneeEmployeeID = strings.TrimSpace(assigneeEmployeeID)
	if assigneeEmployeeID != "" {
		var employee Employee
		if err := a.db.First(&employee, "id = ?", assigneeEmployeeID).Error; err != nil {
			return fmt.Errorf("assignee not found: %w", err)
		}
		newAssigneeID = &employee.ID
		newAssigneeName = employee.FullName
	}

	var previousAssigneeID string
	var previousAssigneeName string
	if task.AssigneeEmployeeID != nil {
		previousAssigneeID = *task.AssigneeEmployeeID
		previousAssigneeName = a.lookupEmployeeNames([]string{previousAssigneeID})[previousAssigneeID]
	}

	if err := a.db.Model(&task).Update("assignee_employee_id", newAssigneeID).Error; err != nil {
		return fmt.Errorf("failed to reassign task: %w", err)
	}

	task.AssigneeEmployeeID = newAssigneeID
	meta := map[string]any{
		"previous_assignee_id":   previousAssigneeID,
		"previous_assignee_name": previousAssigneeName,
		"new_assignee_id":        assigneeEmployeeID,
		"new_assignee_name":      newAssigneeName,
	}
	detail := "Cleared assignee"
	if newAssigneeName != "" {
		detail = fmt.Sprintf("Reassigned task to %s", newAssigneeName)
	}
	_ = a.createTaskActivity(task.ID, current.EmployeeID, "reassigned", detail, meta)

	if assigneeEmployeeID != "" && assigneeEmployeeID != current.EmployeeID {
		a.createTaskNotification(assigneeEmployeeID, task, current.EmployeeName, "assigned")
	}
	if previousAssigneeID != "" && previousAssigneeID != current.EmployeeID && previousAssigneeID != assigneeEmployeeID {
		a.createTaskNotification(previousAssigneeID, task, current.EmployeeName, "reassigned")
	}

	if payload, err := json.Marshal(meta); err == nil {
		a.enqueueCollaborativeOperation("task", task.ID, "reassign", string(payload))
	}
	a.emitCollaborationEvent("tasks:updated", map[string]any{
		"task_id":     task.ID,
		"action":      "reassign",
		"assignee_id": assigneeEmployeeID,
		"previous_id": previousAssigneeID,
		"employee_id": current.EmployeeID,
	})
	a.queueCollaborativeSync("task_reassign")
	return nil
}

func (a *App) UpdateCollaborativeTaskDueDate(taskID, dueDateISO string) error {
	if err := a.requirePermission("tasks:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return err
	}

	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}

	var task TaskItem
	if err := a.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	var dueDate *time.Time
	if strings.TrimSpace(dueDateISO) != "" {
		parsed, err := time.Parse(time.RFC3339, dueDateISO)
		if err != nil {
			return fmt.Errorf("invalid due date: %w", err)
		}
		dueDate = &parsed
	}

	if err := a.db.Model(&task).Update("due_date", dueDate).Error; err != nil {
		return fmt.Errorf("failed to update due date: %w", err)
	}

	task.DueDate = dueDate
	meta := map[string]any{"due_date": dueDateISO}
	detail := "Cleared due date"
	if dueDate != nil {
		detail = fmt.Sprintf("Updated due date to %s", dueDate.Format("2006-01-02"))
	}
	_ = a.createTaskActivity(task.ID, current.EmployeeID, "due_date_updated", detail, meta)

	if task.CreatorEmployeeID != "" && task.CreatorEmployeeID != current.EmployeeID {
		a.createTaskNotification(task.CreatorEmployeeID, task, current.EmployeeName, "due_date_updated")
	}
	if task.AssigneeEmployeeID != nil && *task.AssigneeEmployeeID != "" && *task.AssigneeEmployeeID != current.EmployeeID {
		a.createTaskNotification(*task.AssigneeEmployeeID, task, current.EmployeeName, "due_date_updated")
	}

	if payload, err := json.Marshal(meta); err == nil {
		a.enqueueCollaborativeOperation("task", task.ID, "due_date_update", string(payload))
	}
	a.emitCollaborationEvent("tasks:updated", map[string]any{
		"task_id": task.ID,
		"action":  "due_date_update",
	})
	a.queueCollaborativeSync("task_due_date_update")
	return nil
}

func (a *App) AddCollaborativeTaskComment(taskID, body string) (TaskComment, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return TaskComment{}, err
	}
	if a.db == nil {
		return TaskComment{}, fmt.Errorf("database not initialized")
	}

	current, err := a.GetCurrentEmployeeContext()
	if err != nil {
		return TaskComment{}, err
	}

	body = strings.TrimSpace(body)
	if body == "" {
		return TaskComment{}, fmt.Errorf("comment body is required")
	}

	var task TaskItem
	if err := a.db.First(&task, "id = ?", taskID).Error; err != nil {
		return TaskComment{}, fmt.Errorf("task not found: %w", err)
	}

	comment := TaskComment{
		Base:       Base{CreatedBy: current.EmployeeID},
		TaskID:     task.ID,
		EmployeeID: current.EmployeeID,
		Body:       body,
	}
	if err := a.db.Create(&comment).Error; err != nil {
		return TaskComment{}, fmt.Errorf("failed to create task comment: %w", err)
	}

	now := time.Now()
	_ = a.db.Model(&task).Update("last_comment_at", &now).Error
	_ = a.createTaskActivity(task.ID, current.EmployeeID, "commented", "Added a comment", map[string]any{"comment_id": comment.ID})

	targets := map[string]struct{}{}
	if task.CreatorEmployeeID != "" && task.CreatorEmployeeID != current.EmployeeID {
		targets[task.CreatorEmployeeID] = struct{}{}
	}
	if task.AssigneeEmployeeID != nil && *task.AssigneeEmployeeID != "" && *task.AssigneeEmployeeID != current.EmployeeID {
		targets[*task.AssigneeEmployeeID] = struct{}{}
	}
	for employeeID := range targets {
		a.createTaskNotification(employeeID, task, current.EmployeeName, "commented")
	}

	if payload, err := json.Marshal(comment); err == nil {
		a.enqueueCollaborativeOperation("task_comment", comment.ID, "create", string(payload))
	}

	a.emitCollaborationEvent("tasks:updated", map[string]any{
		"task_id": task.ID,
		"action":  "comment",
	})
	a.queueCollaborativeSync("task_comment")

	comment.EmployeeName = current.EmployeeName
	return comment, nil
}

func (a *App) createTaskActivity(taskID, employeeID, activityType, detail string, metadata map[string]any) error {
	activity := TaskActivity{
		Base:         Base{CreatedBy: employeeID},
		TaskID:       taskID,
		EmployeeID:   employeeID,
		ActivityType: activityType,
		Detail:       detail,
	}
	if len(metadata) > 0 {
		payload, _ := json.Marshal(metadata)
		activity.MetadataJSON = string(payload)
	}
	if err := a.db.Create(&activity).Error; err != nil {
		return err
	}
	if payload, err := json.Marshal(activity); err == nil {
		a.enqueueCollaborativeOperation("task_activity", activity.ID, "create", string(payload))
	}
	return nil
}

func (a *App) createTaskNotification(employeeID string, task TaskItem, actorName, action string) {
	if strings.TrimSpace(employeeID) == "" || a.db == nil {
		return
	}

	actionText := strings.ReplaceAll(strings.TrimSpace(action), "_", " ")
	title := "Task update"
	projectName := ""
	switch action {
	case "assigned":
		title = "New task assigned"
	case "reassigned":
		title = "Task reassigned"
	case "status_changed":
		title = "Task status changed"
	case "commented":
		title = "New task comment"
	case "due_date_updated":
		title = "Task due date updated"
	}

	if task.ProjectID != nil && *task.ProjectID != "" {
		var project Project
		if err := a.db.Select("name").First(&project, "id = ?", *task.ProjectID).Error; err == nil {
			projectName = project.Name
		}
	}

	message := fmt.Sprintf("%s %s task %q", fallbackDisplay(actorName, "Someone"), actionText, task.Title)
	payload, _ := json.Marshal(map[string]any{
		"task_id":      task.ID,
		"task_title":   task.Title,
		"project_id":   task.ProjectID,
		"project_name": projectName,
		"actor_name":   fallbackDisplay(actorName, "System"),
		"action":       action,
	})

	notification := Notification{
		Base:             Base{CreatedBy: a.getCurrentUserID()},
		EmployeeID:       employeeID,
		NotificationType: "task",
		Title:            title,
		Message:          message,
		Status:           "unread",
		SourceType:       "task",
		SourceID:         task.ID,
		ActionRoute:      "#work",
		ActionPayload:    string(payload),
	}
	if err := a.db.Create(&notification).Error; err != nil {
		log.Printf("⚠️ Failed to create task notification: %v", err)
		return
	}

	a.recordNotificationReceipt(notification.ID, employeeID, "", "created")
	if queuePayload, err := json.Marshal(notification); err == nil {
		a.enqueueCollaborativeOperation("notification", notification.ID, "create", string(queuePayload))
	}
	a.emitCollaborationEvent("notifications:new", map[string]any{
		"notification_id": notification.ID,
		"employee_id":     employeeID,
		"source_id":       task.ID,
		"source_type":     "task",
	})
}

// currentProjectActorName resolves a human-friendly name for the acting user,
// preferring the resolved employee context, then the license username.
func currentProjectActorName(a *App) string {
	if a == nil {
		return "Someone"
	}
	if current, err := a.GetCurrentEmployeeContext(); err == nil && strings.TrimSpace(current.EmployeeName) != "" {
		return current.EmployeeName
	}
	if a.currentUser != nil && strings.TrimSpace(a.currentUser.Username) != "" {
		return a.currentUser.Username
	}
	return "Someone"
}

// createProjectMemberNotification notifies the newly added/updated member that
// they were assigned to a project (PH parity — mirrors createTaskNotification).
func (a *App) createProjectMemberNotification(employeeID string, project Project, actorName, role, action string) {
	if strings.TrimSpace(employeeID) == "" || a.db == nil {
		return
	}

	role = strings.TrimSpace(role)
	if role == "" {
		role = "Member"
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = "added"
	}
	title := "Project assignment"
	message := fmt.Sprintf("%s %s you as %s on project %q", fallbackDisplay(actorName, "Someone"), action, role, project.Name)
	payload, _ := json.Marshal(map[string]any{
		"project_id":   project.ID,
		"project_name": project.Name,
		"role":         role,
		"actor_name":   fallbackDisplay(actorName, "System"),
		"action":       action,
	})

	notification := Notification{
		Base:             Base{CreatedBy: a.getCurrentUserID()},
		EmployeeID:       employeeID,
		NotificationType: "project",
		Title:            title,
		Message:          message,
		Status:           "unread",
		SourceType:       "project",
		SourceID:         project.ID,
		ActionRoute:      "#work",
		ActionPayload:    string(payload),
	}
	if err := a.db.Create(&notification).Error; err != nil {
		log.Printf("⚠️ Failed to create project notification: %v", err)
		return
	}

	a.recordNotificationReceipt(notification.ID, employeeID, "", "created")
	if queuePayload, err := json.Marshal(notification); err == nil {
		a.enqueueCollaborativeOperation("notification", notification.ID, "create", string(queuePayload))
	}
	a.emitCollaborationEvent("notifications:new", map[string]any{
		"notification_id": notification.ID,
		"employee_id":     employeeID,
		"source_id":       project.ID,
		"source_type":     "project",
	})
}

func (a *App) recordNotificationReceipt(notificationID, employeeID, deviceID, receiptType string) {
	if a.db == nil || notificationID == "" || employeeID == "" {
		return
	}
	receipt := NotificationReceipt{
		Base:           Base{CreatedBy: "system"},
		NotificationID: notificationID,
		EmployeeID:     employeeID,
		DeviceID:       deviceID,
		ReceiptType:    receiptType,
		ReceivedAt:     time.Now(),
	}
	if err := a.db.Create(&receipt).Error; err != nil {
		return
	}
	if payload, err := json.Marshal(receipt); err == nil {
		a.enqueueCollaborativeOperation("notification_receipt", receipt.ID, "create", string(payload))
	}
}

func (a *App) enqueueCollaborativeOperation(entityType, entityID, operation, payload string) {
	if a.db == nil {
		return
	}
	if entityID = strings.TrimSpace(entityID); entityID == "" {
		return
	}

	now := time.Now()
	if operation != "create" {
		var existing CollaborativePendingOperation
		if err := a.db.Where("entity_type = ? AND entity_id = ? AND operation = ? AND status IN ?",
			entityType, entityID, operation, []string{"pending", "failed"}).
			Order("created_at DESC").
			First(&existing).Error; err == nil {
			if err := a.db.Model(&existing).Updates(map[string]any{
				"payload":         payload,
				"status":          "pending",
				"error_message":   "",
				"next_attempt_at": &now,
				"updated_at":      now,
			}).Error; err == nil {
				return
			}
		}
	}

	item := CollaborativePendingOperation{
		Base:          Base{CreatedBy: a.getCurrentUserID()},
		EntityType:    entityType,
		EntityID:      entityID,
		Operation:     operation,
		Payload:       payload,
		Status:        "pending",
		NextAttemptAt: timePtr(now),
	}
	if err := a.db.Create(&item).Error; err != nil {
		log.Printf("⚠️ Failed to enqueue collaborative operation %s/%s: %v", entityType, operation, err)
	}
}

func (a *App) emitCollaborationEvent(name string, payload any) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, name, payload)
	}
}

func (a *App) decorateTask(task TaskItem) TaskItem {
	if a.db == nil {
		return task
	}

	if task.CreatorEmployeeID != "" {
		var employee Employee
		if err := a.db.Select("id", "full_name").First(&employee, "id = ?", task.CreatorEmployeeID).Error; err == nil {
			task.CreatorName = employee.FullName
		}
	}
	if task.AssigneeEmployeeID != nil && *task.AssigneeEmployeeID != "" {
		var employee Employee
		if err := a.db.Select("id", "full_name").First(&employee, "id = ?", *task.AssigneeEmployeeID).Error; err == nil {
			task.AssigneeName = employee.FullName
		} else {
			task.AssigneeName = "Unknown assignee"
		}
	}

	return task
}

func (a *App) decorateTasks(tasks []TaskItem) []TaskItem {
	if a.db == nil || len(tasks) == 0 {
		return tasks
	}

	employeeIDs := make([]string, 0, len(tasks)*2)
	seen := map[string]struct{}{}
	for _, task := range tasks {
		if task.CreatorEmployeeID != "" {
			if _, ok := seen[task.CreatorEmployeeID]; !ok {
				seen[task.CreatorEmployeeID] = struct{}{}
				employeeIDs = append(employeeIDs, task.CreatorEmployeeID)
			}
		}
		if task.AssigneeEmployeeID != nil && *task.AssigneeEmployeeID != "" {
			if _, ok := seen[*task.AssigneeEmployeeID]; !ok {
				seen[*task.AssigneeEmployeeID] = struct{}{}
				employeeIDs = append(employeeIDs, *task.AssigneeEmployeeID)
			}
		}
	}

	names := a.lookupEmployeeNames(employeeIDs)
	for i := range tasks {
		if tasks[i].CreatorEmployeeID != "" {
			tasks[i].CreatorName = names[tasks[i].CreatorEmployeeID]
		}
		if tasks[i].AssigneeEmployeeID != nil && *tasks[i].AssigneeEmployeeID != "" {
			tasks[i].AssigneeName = names[*tasks[i].AssigneeEmployeeID]
			if tasks[i].AssigneeName == "" {
				tasks[i].AssigneeName = "Unknown assignee"
			}
		}
	}

	return tasks
}

func (a *App) repairDanglingTaskAssignees() {
	if a.db == nil {
		return
	}

	type danglingTaskAssignment struct {
		TaskID     string         `gorm:"column:task_id"`
		StaleID    string         `gorm:"column:stale_assignee_id"`
		HintedName sql.NullString `gorm:"column:hinted_assignee_name"`
	}

	var rows []danglingTaskAssignment
	if err := a.db.Raw(`
		SELECT
			t.id AS task_id,
			t.assignee_employee_id AS stale_assignee_id,
			(
				SELECT json_extract(ta.metadata_json, '$.new_assignee_name')
				FROM task_activity ta
				WHERE ta.task_id = t.id
				  AND ta.activity_type = 'reassigned'
				  AND ta.metadata_json IS NOT NULL
				ORDER BY ta.created_at DESC
				LIMIT 1
			) AS hinted_assignee_name
		FROM task_items t
		LEFT JOIN employees e
		  ON e.id = t.assignee_employee_id
		 AND e.deleted_at IS NULL
		WHERE t.deleted_at IS NULL
		  AND t.assignee_employee_id IS NOT NULL
		  AND TRIM(t.assignee_employee_id) != ''
		  AND e.id IS NULL
	`).Scan(&rows).Error; err != nil {
		return
	}

	for _, row := range rows {
		hintedName := strings.TrimSpace(row.HintedName.String)
		if hintedName == "" {
			continue
		}

		var employees []Employee
		if err := a.db.
			Where("deleted_at IS NULL AND is_active = 1 AND (LOWER(full_name) = LOWER(?) OR LOWER(preferred_name) = LOWER(?))", hintedName, hintedName).
			Find(&employees).Error; err != nil {
			continue
		}
		if len(employees) != 1 {
			continue
		}

		if err := a.db.Model(&TaskItem{}).
			Where("id = ? AND assignee_employee_id = ?", row.TaskID, row.StaleID).
			Update("assignee_employee_id", employees[0].ID).Error; err != nil {
			continue
		}

		log.Printf("✅ Repaired dangling task assignee: task=%s stale=%s -> employee=%s (%s)", row.TaskID, row.StaleID, employees[0].ID, employees[0].FullName)
	}
}

func (a *App) lookupEmployeeNames(ids []string) map[string]string {
	names := map[string]string{}
	if a.db == nil || len(ids) == 0 {
		return names
	}

	var employees []Employee
	if err := a.db.Select("id", "full_name").Where("id IN ?", ids).Find(&employees).Error; err != nil {
		return names
	}
	for _, employee := range employees {
		names[employee.ID] = employee.FullName
	}
	return names
}

func memberEmployeeIDs(members []ProjectMember) []string {
	ids := make([]string, 0, len(members))
	seen := map[string]struct{}{}
	for _, member := range members {
		if member.EmployeeID == "" {
			continue
		}
		if _, ok := seen[member.EmployeeID]; ok {
			continue
		}
		seen[member.EmployeeID] = struct{}{}
		ids = append(ids, member.EmployeeID)
	}
	return ids
}

func commentEmployeeIDs(comments []TaskComment) []string {
	ids := make([]string, 0, len(comments))
	seen := map[string]struct{}{}
	for _, comment := range comments {
		if comment.EmployeeID == "" {
			continue
		}
		if _, ok := seen[comment.EmployeeID]; ok {
			continue
		}
		seen[comment.EmployeeID] = struct{}{}
		ids = append(ids, comment.EmployeeID)
	}
	return ids
}

func activityEmployeeIDs(activity []TaskActivity) []string {
	ids := make([]string, 0, len(activity))
	seen := map[string]struct{}{}
	for _, item := range activity {
		if item.EmployeeID == "" {
			continue
		}
		if _, ok := seen[item.EmployeeID]; ok {
			continue
		}
		seen[item.EmployeeID] = struct{}{}
		ids = append(ids, item.EmployeeID)
	}
	return ids
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func fallbackDisplay(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}
