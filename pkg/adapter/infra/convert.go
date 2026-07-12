// Package infra converts infrastructure models to and from generated Proto messages.
package infra

import (
	"strings"

	"ph_holdings_app/pkg/adapter"
	gorminfra "ph_holdings_app/pkg/infra"
	commonproto "ph_holdings_app/schemas/go/common"
	protoinfra "ph_holdings_app/schemas/go/infra"

	capnp "capnproto.org/go/capnp/v3"
)

func newMessage() (*capnp.Message, *capnp.Segment, error) {
	return capnp.NewMessage(capnp.SingleSegment(nil))
}

func setBase(seg *capnp.Segment, setter func(commonproto.Base) error, base gorminfra.Base) error {
	pb, err := adapter.BaseToProto(seg, base)
	if err != nil {
		return err
	}
	return setter(pb)
}

func deviceStatus(status string) protoinfra.DeviceStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "first_setup":
		return protoinfra.DeviceStatus_firstSetup
	case "approved":
		return protoinfra.DeviceStatus_approved
	case "blocked":
		return protoinfra.DeviceStatus_blocked
	case "revoked":
		return protoinfra.DeviceStatus_revoked
	default:
		return protoinfra.DeviceStatus_pending
	}
}

func jobStatus(status string) protoinfra.JobStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return protoinfra.JobStatus_running
	case "completed":
		return protoinfra.JobStatus_completed
	case "failed":
		return protoinfra.JobStatus_failed
	case "cancelled", "canceled":
		return protoinfra.JobStatus_cancelled
	default:
		return protoinfra.JobStatus_pending
	}
}

func riskLevel(severity string) commonproto.RiskLevel {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "low":
		return commonproto.RiskLevel_low
	case "medium":
		return commonproto.RiskLevel_medium
	case "high":
		return commonproto.RiskLevel_high
	case "critical":
		return commonproto.RiskLevel_critical
	default:
		return commonproto.RiskLevel_unknown
	}
}

func SettingToProto(m gorminfra.Setting) (*protoinfra.Setting, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protoinfra.NewSetting(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetKey(m.Key), p.SetValue(m.Value), p.SetCategory(m.Category), p.SetDescription(m.Description)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetIsEncrypted(m.IsEncrypted)
	return &p, nil
}

func SettingFromProto(p protoinfra.Setting) (gorminfra.Setting, error) {
	m := gorminfra.Setting{}
	m.Key, _ = p.Key()
	m.Value, _ = p.Value()
	m.Category, _ = p.Category()
	m.Description, _ = p.Description()
	m.IsEncrypted = p.IsEncrypted()
	return m, nil
}

func RoleToProto(m gorminfra.Role) (*protoinfra.Role, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protoinfra.NewRole(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetName(m.Name), p.SetDisplayName(m.DisplayName), p.SetDescription(m.Description), p.SetPermissions(m.Permissions)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetIsActive(m.IsActive)
	p.SetIsSystem(m.IsSystem)
	return &p, nil
}

func RoleFromProto(p protoinfra.Role) (gorminfra.Role, error) {
	m := gorminfra.Role{}
	m.Name, _ = p.Name()
	m.DisplayName, _ = p.DisplayName()
	m.Permissions, _ = p.Permissions()
	m.IsActive = p.IsActive()
	m.IsSystem = p.IsSystem()
	return m, nil
}

func UserToProto(m gorminfra.User) (*protoinfra.User, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protoinfra.NewUser(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetUsername(m.Username), p.SetEmail(m.Email), p.SetPasswordHash(m.PasswordHash), p.SetRoleId(m.RoleID), p.SetFullName(m.FullName), p.SetDisplayName(m.DisplayName), p.SetDepartment(m.Department), p.SetJobTitle(m.JobTitle), p.SetLastLoginAt(adapter.TimePtrToText(m.LastLoginAt)), p.SetPasswordChangedAt(adapter.TimePtrToText(m.PasswordChangedAt)), p.SetRoleName(m.RoleName)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetIsActive(m.IsActive)
	p.SetMustChangePassword(m.MustChangePassword)
	return &p, nil
}

func UserFromProto(p protoinfra.User) (gorminfra.User, error) {
	m := gorminfra.User{}
	m.Username, _ = p.Username()
	m.Email, _ = p.Email()
	m.RoleID, _ = p.RoleId()
	m.FullName, _ = p.FullName()
	m.DisplayName, _ = p.DisplayName()
	m.IsActive = p.IsActive()
	return m, nil
}

func DeviceToProto(m gorminfra.Device) (*protoinfra.Device, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protoinfra.NewDevice(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetMachineId(m.MachineID), p.SetDeviceName(m.DeviceName), p.SetOsInfo(m.OSInfo), p.SetFirstSeenAt(adapter.TimeToText(m.FirstSeenAt)), p.SetLastSeenAt(adapter.TimePtrToText(m.LastSeenAt)), p.SetApprovedBy(m.ApprovedBy), p.SetApprovedAt(adapter.TimePtrToText(m.ApprovedAt)), p.SetNotes(m.Notes), p.SetApproverName(m.ApproverName)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetStatus(deviceStatus(m.Status))
	p.SetIsAdminDevice(m.IsAdminDevice)
	return &p, nil
}

func DeviceFromProto(p protoinfra.Device) (gorminfra.Device, error) {
	m := gorminfra.Device{}
	m.MachineID, _ = p.MachineId()
	m.DeviceName, _ = p.DeviceName()
	m.OSInfo, _ = p.OsInfo()
	m.IsAdminDevice = p.IsAdminDevice()
	return m, nil
}

func DeviceUserToProto(m gorminfra.DeviceUser) (*protoinfra.DeviceUser, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protoinfra.NewDeviceUser(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetDeviceId(m.DeviceID), p.SetUserId(m.UserID)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetIsPrimary(m.IsPrimary)
	return &p, nil
}

func DeviceUserFromProto(p protoinfra.DeviceUser) (gorminfra.DeviceUser, error) {
	m := gorminfra.DeviceUser{}
	m.DeviceID, _ = p.DeviceId()
	m.UserID, _ = p.UserId()
	m.IsPrimary = p.IsPrimary()
	return m, nil
}

func UserSessionToProto(m gorminfra.UserSession) (*protoinfra.UserSession, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protoinfra.NewUserSession(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetUserId(m.UserID), p.SetToken(m.Token), p.SetRefreshToken(m.RefreshToken), p.SetAccessTokenExpiry(adapter.TimeToText(m.AccessTokenExpiry)), p.SetRefreshTokenExpiry(adapter.TimeToText(m.RefreshTokenExpiry)), p.SetLastActivityAt(adapter.TimeToText(m.LastActivityAt)), p.SetInvalidatedAt(adapter.TimePtrToText(m.InvalidatedAt)), p.SetInvalidatedReason(m.InvalidatedReason)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetIsActive(m.IsActive)
	return &p, nil
}

func UserSessionFromProto(p protoinfra.UserSession) (gorminfra.UserSession, error) {
	m := gorminfra.UserSession{}
	m.UserID, _ = p.UserId()
	m.Token, _ = p.Token()
	m.RefreshToken, _ = p.RefreshToken()
	m.IsActive = p.IsActive()
	return m, nil
}

func AlertToProto(m gorminfra.Alert) (*protoinfra.Alert, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protoinfra.NewAlert(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetAlertType(m.AlertType), p.SetTitle(m.Title), p.SetMessage_(m.Message)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetSeverity(riskLevel(m.Severity))
	p.SetIsActive(m.IsActive)
	p.SetIsAcknowledged(m.IsAcknowledged)
	return &p, nil
}

func AlertFromProto(p protoinfra.Alert) (gorminfra.Alert, error) {
	m := gorminfra.Alert{}
	m.AlertType, _ = p.AlertType()
	m.Title, _ = p.Title()
	m.Message, _ = p.Message_()
	m.IsActive = p.IsActive()
	m.IsAcknowledged = p.IsAcknowledged()
	return m, nil
}

func AuditLogToProto(m gorminfra.AuditLog) (*protoinfra.AuditLog, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protoinfra.NewAuditLog(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetUserId(m.UserID), p.SetAction(m.Action), p.SetResource(m.Resource)} {
		if err != nil {
			return nil, err
		}
	}
	return &p, nil
}

func AuditLogFromProto(p protoinfra.AuditLog) (gorminfra.AuditLog, error) {
	m := gorminfra.AuditLog{}
	m.UserID, _ = p.UserId()
	m.Action, _ = p.Action()
	m.Resource, _ = p.Resource()
	return m, nil
}

func JobToProto(m gorminfra.Job) (*protoinfra.Job, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protoinfra.NewJob(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetType(m.Type), p.SetInput(m.Input), p.SetOutput(m.Output), p.SetError(m.Error), p.SetStartedAt(adapter.TimePtrToText(m.StartedAt)), p.SetCompletedAt(adapter.TimePtrToText(m.CompletedAt))} {
		if err != nil {
			return nil, err
		}
	}
	p.SetStatus(jobStatus(m.Status))
	p.SetProgress(int64(m.Progress))
	p.SetAttempts(int64(m.Attempts))
	p.SetMaxAttempts(int64(m.MaxAttempts))
	return &p, nil
}

func JobFromProto(p protoinfra.Job) (gorminfra.Job, error) {
	m := gorminfra.Job{}
	m.Type, _ = p.Type()
	m.Input, _ = p.Input()
	m.Output, _ = p.Output()
	m.Error, _ = p.Error()
	m.Progress = int(p.Progress())
	m.Attempts = int(p.Attempts())
	m.MaxAttempts = int(p.MaxAttempts())
	return m, nil
}

func BackupPolicyToProto(m gorminfra.BackupPolicy) (*protoinfra.BackupPolicy, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protoinfra.NewBackupPolicy(seg)
	if err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetLastBackupAt(m.LastBackupAt), p.SetLastBackupPath(m.LastBackupPath), p.SetNextBackupDueAt(m.NextBackupDueAt)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetAutoBackupEnabled(m.AutoBackupEnabled)
	p.SetFrequencyDays(int64(m.FrequencyDays))
	p.SetDueNow(m.DueNow)
	return &p, nil
}

func BackupPolicyFromProto(p protoinfra.BackupPolicy) (gorminfra.BackupPolicy, error) {
	m := gorminfra.BackupPolicy{AutoBackupEnabled: p.AutoBackupEnabled(), FrequencyDays: int(p.FrequencyDays()), DueNow: p.DueNow()}
	m.LastBackupAt, _ = p.LastBackupAt()
	m.LastBackupPath, _ = p.LastBackupPath()
	m.NextBackupDueAt, _ = p.NextBackupDueAt()
	return m, nil
}
