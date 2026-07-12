// Package infra contains shared infrastructure domain types.
package infra

import (
	"time"

	shareddomain "ph_holdings_app/pkg/domain"
)

type Base = shareddomain.Base

type Setting struct {
	Base
	Key         string `gorm:"uniqueIndex;size:255" json:"key"`
	Value       string `gorm:"type:varchar(5000)" json:"value"`
	Category    string `gorm:"index;size:100" json:"category"`
	Description string `gorm:"type:varchar(1000)" json:"description"`
	IsEncrypted bool   `gorm:"default:false" json:"is_encrypted"`
}

func (Setting) TableName() string { return "settings" }

type UserSession struct {
	Base
	UserID             string     `gorm:"index;size:36;constraint:OnDelete:CASCADE;" json:"user_id"`
	Token              string     `gorm:"uniqueIndex;size:255" json:"token"`
	RefreshToken       string     `gorm:"uniqueIndex;size:255" json:"refresh_token"`
	AccessTokenExpiry  time.Time  `gorm:"index" json:"access_token_expiry"`
	RefreshTokenExpiry time.Time  `gorm:"index" json:"refresh_token_expiry"`
	LastActivityAt     time.Time  `gorm:"index" json:"last_activity_at"`
	IsActive           bool       `gorm:"index;default:true" json:"is_active"`
	InvalidatedAt      *time.Time `json:"invalidated_at,omitempty"`
	InvalidatedReason  string     `gorm:"size:255" json:"invalidated_reason,omitempty"`
}

type Role struct {
	Base
	Name        string `gorm:"uniqueIndex;size:50" json:"name"`
	DisplayName string `gorm:"size:255" json:"display_name"`
	Description string `gorm:"type:varchar(1000)" json:"description"`
	Permissions string `gorm:"type:text" json:"permissions"` // FIX MEDIUM-2: Changed from varchar(5000) to text (no limit)
	IsActive    bool   `gorm:"index;default:true" json:"is_active"`
	IsSystem    bool   `gorm:"default:false" json:"is_system"`
}

type User struct {
	Base
	Username     string `gorm:"uniqueIndex;size:100" json:"username"`
	Email        string `gorm:"uniqueIndex;size:255" json:"email"`
	PasswordHash string `gorm:"size:255" json:"-"`
	RoleID       string `gorm:"index;size:36" json:"role_id"` // RBAC DEPRECATED - no FK constraint

	FullName    string `json:"full_name"`
	DisplayName string `json:"display_name"`
	Department  string `json:"department"`
	JobTitle    string `json:"job_title"`

	IsActive           bool       `gorm:"index;default:true" json:"is_active"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	PasswordChangedAt  *time.Time `json:"password_changed_at"`
	MustChangePassword bool       `gorm:"default:false" json:"must_change_password"`

	// Virtual / Relationships - RBAC DEPRECATED
	RoleName string `gorm:"-" json:"role_name,omitempty"`
	Role     Role   `gorm:"-" json:"role"` // Not a DB relationship anymore
}

type Device struct {
	Base
	MachineID     string     `gorm:"uniqueIndex;size:255" json:"machine_id"`                          // SHA-256 hash of hardware identifiers
	DeviceName    string     `gorm:"size:255" json:"device_name"`                                     // Hostname
	OSInfo        string     `gorm:"size:255" json:"os_info"`                                         // "Windows 11 Pro"
	FirstSeenAt   time.Time  `gorm:"autoCreateTime;index:idx_status_first_seen" json:"first_seen_at"` // FIX CRITICAL-3: Added composite index
	LastSeenAt    *time.Time `json:"last_seen_at"`
	Status        string     `gorm:"size:20;default:pending;index:idx_status_first_seen;check:status IN ('first_setup','pending','approved','blocked','revoked')" json:"status"` // FIX: Added 'first_setup' for device registration flow
	ApprovedBy    string     `gorm:"size:36" json:"approved_by"`                                                                                                                 // RBAC DEPRECATED - no FK constraint
	ApprovedAt    *time.Time `json:"approved_at"`
	IsAdminDevice bool       `gorm:"default:false" json:"is_admin_device"` // First device = admin device
	Notes         string     `gorm:"type:varchar(2000)" json:"notes"`

	// Virtual fields
	ApproverName string `gorm:"-" json:"approver_name,omitempty"` // FIX LOW-1: Added omitempty
}

type DeviceUser struct {
	Base
	DeviceID  string `gorm:"uniqueIndex:idx_device_user;index;size:36" json:"device_id"` // RBAC DEPRECATED - no FK constraint
	UserID    string `gorm:"uniqueIndex:idx_device_user;index;size:36" json:"user_id"`   // RBAC DEPRECATED - no FK constraint
	IsPrimary bool   `gorm:"default:false" json:"is_primary"`

	// Virtual fields - not DB relationships
	User   User   `gorm:"-" json:"user"`
	Device Device `gorm:"-" json:"device"`
}

type Alert struct {
	Base
	AlertType      string `gorm:"index;size:100" json:"alert_type"`
	Severity       string `gorm:"index;size:50;check:severity IN ('low','medium','high','critical')" json:"severity"`
	Title          string `gorm:"size:255" json:"title"`
	Message        string `gorm:"type:varchar(2000)" json:"message"`
	IsActive       bool   `gorm:"index;default:true" json:"is_active"`
	IsAcknowledged bool   `gorm:"index;default:false" json:"is_acknowledged"`
}

type AuditLog struct {
	Base
	UserID   string `gorm:"index;size:36;constraint:OnDelete:SET NULL;" json:"user_id"`
	Action   string `gorm:"index;size:50" json:"action"`
	Resource string `gorm:"index;size:100" json:"resource"`
	// ResourceID and Description were accepted by logAudit for years and
	// silently dropped ("simplified schema"). Wave 3 B.2 restores them: an
	// audit row that can't say WHICH resource or WHY is not an audit trail.
	ResourceID  string `gorm:"index;size:100" json:"resource_id"`
	Description string `gorm:"size:1000" json:"description"`
}

func (AuditLog) TableName() string { return "audit_logs" }

type Job struct {
	Base
	Type        string     `gorm:"index;size:50" json:"type"`
	Status      string     `gorm:"index;size:50;check:status IN ('pending','running','completed','failed','cancelled')" json:"status"`
	Input       string     `gorm:"type:varchar(5000)" json:"input"`
	Output      string     `gorm:"type:varchar(5000)" json:"output"`
	Error       string     `gorm:"type:varchar(2000)" json:"error,omitempty"`
	Progress    int        `gorm:"check:progress >= 0 AND progress <= 100" json:"progress"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Retry logic
	Attempts    int `gorm:"default:0;check:attempts >= 0" json:"attempts"`
	MaxAttempts int `gorm:"default:3;check:max_attempts >= 0" json:"max_attempts"`
}

func (Job) TableName() string { return "jobs" }

type BackupPolicy struct {
	AutoBackupEnabled bool   `json:"auto_backup_enabled"`
	FrequencyDays     int    `json:"frequency_days"`
	LastBackupAt      string `json:"last_backup_at"`
	LastBackupPath    string `json:"last_backup_path"`
	NextBackupDueAt   string `json:"next_backup_due_at"`
	DueNow            bool   `json:"due_now"`
}
