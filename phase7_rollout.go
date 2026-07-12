package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	phase7FollowUpBackfillSetting = "phase7_followup_backfill_completed_at"
	phase7FollowUpBackfillCount   = "phase7_followup_backfill_count"
	pilotDeploymentChecklistKey   = "pilot_deployment_checklist"
)

type Phase7RolloutStatus struct {
	FollowUpBackfillCompletedAt string `json:"followup_backfill_completed_at"`
	FollowUpBackfillCount       int    `json:"followup_backfill_count"`
	LegacyFollowUpTasks         int    `json:"legacy_followup_tasks"`
	MigratedLegacyTasks         int    `json:"migrated_legacy_tasks"`
	PendingCollaborativeOps     int    `json:"pending_collaborative_ops"`
	FailedCollaborativeOps      int    `json:"failed_collaborative_ops"`
	DeadLetterCollaborativeOps  int    `json:"dead_letter_collaborative_ops"`
	PayrollPayoutsAwaitingRecon int    `json:"payroll_payouts_awaiting_recon"`
}

type Phase7ActionResult struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Processed int    `json:"processed"`
}

type PilotReadinessSummary struct {
	GeneratedAt            string `json:"generated_at"`
	TotalEmployees         int    `json:"total_employees"`
	ReadyEmployees         int    `json:"ready_employees"`
	EmployeesWithIssues    int    `json:"employees_with_issues"`
	EmployeesMissingAccess int    `json:"employees_missing_access"`
	ActivatedLicenses      int    `json:"activated_licenses"`
	UnlinkedLicenses       int    `json:"unlinked_licenses"`
	PendingDevices         int    `json:"pending_devices"`
	BlockedDevices         int    `json:"blocked_devices"`
	ApprovedDevices        int    `json:"approved_devices"`
}

type PilotReadinessRow struct {
	EmployeeID      string   `json:"employee_id"`
	EmployeeCode    string   `json:"employee_code"`
	EmployeeName    string   `json:"employee_name"`
	Department      string   `json:"department"`
	JobTitle        string   `json:"job_title"`
	EmploymentState string   `json:"employment_state"`
	AccessStatus    string   `json:"access_status"`
	LicenseKey      string   `json:"license_key"`
	LicenseRole     string   `json:"license_role"`
	LicenseActive   bool     `json:"license_active"`
	LicenseAssigned string   `json:"license_assigned_to"`
	DeviceID        string   `json:"device_id"`
	DeviceName      string   `json:"device_name"`
	DeviceStatus    string   `json:"device_status"`
	LastSeenAt      string   `json:"last_seen_at"`
	UserID          string   `json:"user_id"`
	UserName        string   `json:"user_name"`
	ReadyForPilot   bool     `json:"ready_for_pilot"`
	Issues          []string `json:"issues"`
}

type PilotSupportBundleResult struct {
	Path string `json:"path"`
	Rows int    `json:"rows"`
}

type PilotExportResult struct {
	Path string `json:"path"`
}

type PilotChecklistItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	Notes       string `json:"notes"`
	CompletedAt string `json:"completed_at"`
}

// Mission I (I-11): bound data-repair is gated; startup uses the internal.
func (a *App) EnsurePhase7Rollout() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	return a.ensurePhase7RolloutInternal()
}

func (a *App) ensurePhase7RolloutInternal() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if err := a.normalizeCollaborativePendingOperations(); err != nil {
		return err
	}
	if err := a.backfillLegacyFollowUpTasks(); err != nil {
		return err
	}
	return nil
}

func (a *App) GetPhase7RolloutStatus() Phase7RolloutStatus {
	if !a.canAccessPhase7Rollout() {
		return Phase7RolloutStatus{}
	}
	if a.db == nil {
		return Phase7RolloutStatus{}
	}

	status := Phase7RolloutStatus{}
	var setting Setting
	if err := a.db.Where("key = ?", phase7FollowUpBackfillSetting).First(&setting).Error; err == nil {
		status.FollowUpBackfillCompletedAt = strings.TrimSpace(setting.Value)
	}
	if err := a.db.Where("key = ?", phase7FollowUpBackfillCount).First(&setting).Error; err == nil {
		fmt.Sscanf(strings.TrimSpace(setting.Value), "%d", &status.FollowUpBackfillCount)
	}

	var count int64
	_ = a.db.Model(&FollowUpTask{}).Count(&count).Error
	status.LegacyFollowUpTasks = int(count)
	_ = a.db.Model(&TaskItem{}).Where("legacy_follow_up_id IS NOT NULL").Count(&count).Error
	status.MigratedLegacyTasks = int(count)
	_ = a.db.Model(&CollaborativePendingOperation{}).Where("status = ?", "pending").Count(&count).Error
	status.PendingCollaborativeOps = int(count)
	_ = a.db.Model(&CollaborativePendingOperation{}).Where("status = ?", "failed").Count(&count).Error
	status.FailedCollaborativeOps = int(count)
	_ = a.db.Model(&CollaborativePendingOperation{}).Where("status = ?", "dead_letter").Count(&count).Error
	status.DeadLetterCollaborativeOps = int(count)
	_ = a.db.Model(&PayrollPayout{}).Where("status = ? AND bank_statement_line_id IS NULL", "paid").Count(&count).Error
	status.PayrollPayoutsAwaitingRecon = int(count)

	return status
}

func (a *App) ListCollaborativePendingOperations(status string, limit int) ([]CollaborativePendingOperation, error) {
	if !a.canAccessPhase7Rollout() {
		return nil, fmt.Errorf("access denied")
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	status = strings.TrimSpace(strings.ToLower(status))
	if limit <= 0 {
		limit = 25
	}
	if limit > 200 {
		limit = 200
	}

	query := a.db.Model(&CollaborativePendingOperation{})
	switch status {
	case "", "active":
		query = query.Where("status IN ?", []string{"pending", "failed", "dead_letter"})
	case "pending", "failed", "dead_letter", "synced":
		query = query.Where("status = ?", status)
	default:
		return nil, fmt.Errorf("invalid status filter")
	}

	var ops []CollaborativePendingOperation
	if err := query.Order("updated_at DESC, created_at DESC").Limit(limit).Find(&ops).Error; err != nil {
		return nil, err
	}
	return ops, nil
}

func (a *App) RetryCollaborativePendingOperation(operationID string) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return fmt.Errorf("operation id is required")
	}

	var op CollaborativePendingOperation
	if err := a.db.First(&op, "id = ?", operationID).Error; err != nil {
		return err
	}
	if op.Status == "synced" {
		return fmt.Errorf("operation is already synced")
	}

	now := time.Now()
	if err := a.db.Model(&op).Updates(map[string]any{
		"status":          "pending",
		"attempts":        0,
		"error_message":   "",
		"next_attempt_at": &now,
		"updated_at":      now,
	}).Error; err != nil {
		return err
	}

	a.queueCollaborativeSync("admin_retry_single")
	return nil
}

func (a *App) RetryCollaborativePendingOperations(status string, limit int) (Phase7ActionResult, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return Phase7ActionResult{}, err
	}
	if a.db == nil {
		return Phase7ActionResult{}, fmt.Errorf("database not initialized")
	}

	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case "", "failed", "dead_letter", "active":
	default:
		return Phase7ActionResult{}, fmt.Errorf("invalid retry status")
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	query := a.db.Model(&CollaborativePendingOperation{})
	if status == "" || status == "active" {
		query = query.Where("status IN ?", []string{"failed", "dead_letter"})
	} else {
		query = query.Where("status = ?", status)
	}

	var ids []string
	if err := query.Order("updated_at DESC, created_at DESC").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return Phase7ActionResult{}, err
	}
	if len(ids) == 0 {
		return Phase7ActionResult{
			Status:    "noop",
			Message:   "No collaborative operations matched the retry filter.",
			Processed: 0,
		}, nil
	}

	now := time.Now()
	if err := a.db.Model(&CollaborativePendingOperation{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":          "pending",
			"attempts":        0,
			"error_message":   "",
			"next_attempt_at": &now,
			"updated_at":      now,
		}).Error; err != nil {
		return Phase7ActionResult{}, err
	}

	a.queueCollaborativeSync("admin_retry_bulk")
	return Phase7ActionResult{
		Status:    "ok",
		Message:   fmt.Sprintf("Re-queued %d collaborative operations for sync.", len(ids)),
		Processed: len(ids),
	}, nil
}

func (a *App) TriggerCollaborativeSyncNow() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	return a.RefreshCollaborativeWorkspace()
}

func (a *App) RerunPhase7FollowUpBackfill() (Phase7ActionResult, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return Phase7ActionResult{}, err
	}
	if a.db == nil {
		return Phase7ActionResult{}, fmt.Errorf("database not initialized")
	}
	if !a.db.Migrator().HasTable("followup_tasks") {
		return Phase7ActionResult{
			Status:    "noop",
			Message:   "Legacy follow-up table does not exist on this database.",
			Processed: 0,
		}, nil
	}

	var before int64
	_ = a.db.Model(&TaskItem{}).Where("legacy_follow_up_id IS NOT NULL").Count(&before).Error
	_ = a.db.Unscoped().Where("key IN ?", []string{phase7FollowUpBackfillSetting, phase7FollowUpBackfillCount}).Delete(&Setting{}).Error

	if err := a.backfillLegacyFollowUpTasks(); err != nil {
		return Phase7ActionResult{}, err
	}

	var after int64
	_ = a.db.Model(&TaskItem{}).Where("legacy_follow_up_id IS NOT NULL").Count(&after).Error
	processed := int(after - before)
	if processed < 0 {
		processed = 0
	}

	return Phase7ActionResult{
		Status:    "ok",
		Message:   fmt.Sprintf("Phase 7 legacy follow-up backfill re-ran successfully. %d task(s) added.", processed),
		Processed: processed,
	}, nil
}

func (a *App) canAccessPhase7Rollout() bool {
	return a.requirePermission("settings:update") == nil
}

func (a *App) GetPilotReadinessSummary() PilotReadinessSummary {
	if !a.canAccessPhase7Rollout() || a.db == nil {
		return PilotReadinessSummary{}
	}

	rows := a.buildPilotReadinessRows()
	summary := PilotReadinessSummary{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		TotalEmployees: len(rows),
	}

	for _, row := range rows {
		if row.ReadyForPilot {
			summary.ReadyEmployees++
		}
		if len(row.Issues) > 0 {
			summary.EmployeesWithIssues++
		}
		if containsIssueTag(row.Issues, "missing_access_link") {
			summary.EmployeesMissingAccess++
		}
	}

	var count int64
	_ = a.db.Model(&LicenseKey{}).Where("activated = ?", true).Count(&count).Error
	summary.ActivatedLicenses = int(count)
	_ = a.db.Model(&LicenseKey{}).Where("activated = ? AND key NOT IN (?)", true, a.linkedLicenseSubquery()).Count(&count).Error
	summary.UnlinkedLicenses = int(count)
	_ = a.db.Model(&Device{}).Where("status = ?", "pending").Count(&count).Error
	summary.PendingDevices = int(count)
	_ = a.db.Model(&Device{}).Where("status = ?", "blocked").Count(&count).Error
	summary.BlockedDevices = int(count)
	_ = a.db.Model(&Device{}).Where("status = ?", "approved").Count(&count).Error
	summary.ApprovedDevices = int(count)

	return summary
}

func (a *App) ListPilotReadinessRows(onlyIssues bool) ([]PilotReadinessRow, error) {
	if !a.canAccessPhase7Rollout() {
		return nil, fmt.Errorf("access denied")
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows := a.buildPilotReadinessRows()
	if !onlyIssues {
		return rows, nil
	}

	filtered := make([]PilotReadinessRow, 0, len(rows))
	for _, row := range rows {
		if len(row.Issues) > 0 {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
}

func (a *App) ExportPilotSupportBundle() (PilotSupportBundleResult, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return PilotSupportBundleResult{}, err
	}
	if a.db == nil {
		return PilotSupportBundleResult{}, fmt.Errorf("database not initialized")
	}

	payload := map[string]any{
		"generated_at":      time.Now().UTC().Format(time.RFC3339),
		"pilot_summary":     a.GetPilotReadinessSummary(),
		"pilot_rows":        a.buildPilotReadinessRows(),
		"rollout_status":    a.GetPhase7RolloutStatus(),
		"queue_snapshot":    a.exportableCollaborativeOps(),
		"all_devices":       a.exportableDevices(),
		"license_inventory": a.exportableLicenses(),
	}

	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return PilotSupportBundleResult{}, err
	}

	paths := a.getAppPaths()
	reportDir := filepath.Join(paths.ReportOutput, "pilot_support")
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return PilotSupportBundleResult{}, err
	}
	filename := fmt.Sprintf("pilot_support_bundle_%s.json", time.Now().Format("20060102_150405"))
	fullPath := filepath.Join(reportDir, filename)
	if err := os.WriteFile(fullPath, bytes, 0644); err != nil {
		return PilotSupportBundleResult{}, err
	}

	rows := payload["pilot_rows"].([]PilotReadinessRow)
	return PilotSupportBundleResult{Path: fullPath, Rows: len(rows)}, nil
}

func (a *App) ExportPilotSignoffReport() (PilotExportResult, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return PilotExportResult{}, err
	}
	if a.db == nil {
		return PilotExportResult{}, fmt.Errorf("database not initialized")
	}

	summary := a.GetPilotReadinessSummary()
	rollout := a.GetPhase7RolloutStatus()
	checklist := a.loadPilotDeploymentChecklist()
	rows := a.buildPilotReadinessRows()
	issueRows := make([]PilotReadinessRow, 0)
	for _, row := range rows {
		if len(row.Issues) > 0 {
			issueRows = append(issueRows, row)
		}
	}

	var builder strings.Builder
	builder.WriteString("# Pilot Sign-Off Report\n\n")
	builder.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC1123)))
	builder.WriteString("## Readiness Summary\n")
	builder.WriteString(fmt.Sprintf("- Total employees: %d\n", summary.TotalEmployees))
	builder.WriteString(fmt.Sprintf("- Ready employees: %d\n", summary.ReadyEmployees))
	builder.WriteString(fmt.Sprintf("- Employees with issues: %d\n", summary.EmployeesWithIssues))
	builder.WriteString(fmt.Sprintf("- Missing access links: %d\n", summary.EmployeesMissingAccess))
	builder.WriteString(fmt.Sprintf("- Activated licenses: %d\n", summary.ActivatedLicenses))
	builder.WriteString(fmt.Sprintf("- Unlinked licenses: %d\n", summary.UnlinkedLicenses))
	builder.WriteString(fmt.Sprintf("- Pending devices: %d\n", summary.PendingDevices))
	builder.WriteString(fmt.Sprintf("- Blocked devices: %d\n", summary.BlockedDevices))
	builder.WriteString(fmt.Sprintf("- Pending collaborative ops: %d\n", rollout.PendingCollaborativeOps))
	builder.WriteString(fmt.Sprintf("- Failed collaborative ops: %d\n", rollout.FailedCollaborativeOps))
	builder.WriteString(fmt.Sprintf("- Dead letter ops: %d\n", rollout.DeadLetterCollaborativeOps))
	builder.WriteString("\n## Checklist\n")
	for _, item := range checklist {
		status := "Open"
		if item.Completed {
			status = "Completed"
		}
		builder.WriteString(fmt.Sprintf("- [%s] %s\n", status, item.Title))
		if item.Description != "" {
			builder.WriteString(fmt.Sprintf("  - %s\n", item.Description))
		}
		if item.CompletedAt != "" {
			builder.WriteString(fmt.Sprintf("  - Completed at: %s\n", item.CompletedAt))
		}
		if strings.TrimSpace(item.Notes) != "" {
			builder.WriteString(fmt.Sprintf("  - Notes: %s\n", strings.TrimSpace(item.Notes)))
		}
	}
	builder.WriteString("\n## Outstanding Employee Issues\n")
	if len(issueRows) == 0 {
		builder.WriteString("- No active employee rollout issues.\n")
	} else {
		for _, row := range issueRows {
			builder.WriteString(fmt.Sprintf("- %s (%s): %s\n", row.EmployeeName, row.EmployeeCode, strings.Join(row.Issues, ", ")))
		}
	}

	paths := a.getAppPaths()
	reportDir := filepath.Join(paths.ReportOutput, "pilot_support")
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return PilotExportResult{}, err
	}
	filename := fmt.Sprintf("pilot_signoff_%s.md", time.Now().Format("20060102_150405"))
	fullPath := filepath.Join(reportDir, filename)
	if err := os.WriteFile(fullPath, []byte(builder.String()), 0644); err != nil {
		return PilotExportResult{}, err
	}

	return PilotExportResult{Path: fullPath}, nil
}

func (a *App) GetPilotDeploymentChecklist() ([]PilotChecklistItem, error) {
	if !a.canAccessPhase7Rollout() {
		return nil, fmt.Errorf("access denied")
	}
	return a.loadPilotDeploymentChecklist(), nil
}

func (a *App) UpdatePilotDeploymentChecklistItem(itemID string, completed bool, notes string) ([]PilotChecklistItem, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return nil, fmt.Errorf("checklist item id is required")
	}

	items := a.loadPilotDeploymentChecklist()
	updated := false
	now := time.Now().UTC().Format(time.RFC3339)
	for idx := range items {
		if items[idx].ID != itemID {
			continue
		}
		items[idx].Completed = completed
		items[idx].Notes = strings.TrimSpace(notes)
		if completed {
			if items[idx].CompletedAt == "" {
				items[idx].CompletedAt = now
			}
		} else {
			items[idx].CompletedAt = ""
		}
		updated = true
		break
	}

	if !updated {
		return nil, fmt.Errorf("unknown checklist item")
	}

	payload, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	a.saveSetting(pilotDeploymentChecklistKey, string(payload))
	return items, nil
}

func (a *App) buildPilotReadinessRows() []PilotReadinessRow {
	if a.db == nil {
		return nil
	}

	var employees []Employee
	_ = a.db.Where("is_active = ?", true).Order("full_name ASC").Find(&employees).Error

	var accessLinks []EmployeeAccessLink
	_ = a.db.Order("is_primary DESC, created_at ASC").Find(&accessLinks).Error

	var licenses []LicenseKey
	_ = a.db.Find(&licenses).Error

	var devices []Device
	_ = a.db.Find(&devices).Error

	var users []User
	_ = a.db.Select("id", "full_name", "display_name", "is_active").Find(&users).Error

	linkByEmployee := make(map[string]EmployeeAccessLink)
	for _, link := range accessLinks {
		if _, exists := linkByEmployee[link.EmployeeID]; !exists || link.IsPrimary {
			linkByEmployee[link.EmployeeID] = link
		}
	}

	licenseByKey := make(map[string]LicenseKey, len(licenses))
	for _, license := range licenses {
		licenseByKey[license.Key] = license
	}

	deviceByID := make(map[string]Device, len(devices))
	deviceByMachine := make(map[string]Device, len(devices))
	for _, device := range devices {
		deviceByID[device.ID] = device
		deviceByMachine[device.MachineID] = device
	}

	userByID := make(map[string]User, len(users))
	for _, user := range users {
		userByID[user.ID] = user
	}

	rows := make([]PilotReadinessRow, 0, len(employees))
	for _, employee := range employees {
		row := PilotReadinessRow{
			EmployeeID:      employee.ID,
			EmployeeCode:    employee.EmployeeCode,
			EmployeeName:    employee.FullName,
			Department:      employee.Department,
			JobTitle:        employee.JobTitle,
			EmploymentState: employee.EmploymentStatus,
			AccessStatus:    "missing",
		}

		link, hasLink := linkByEmployee[employee.ID]
		if !hasLink {
			row.Issues = append(row.Issues, "missing_access_link")
			rows = append(rows, row)
			continue
		}

		row.AccessStatus = fallbackDisplay(strings.TrimSpace(link.AccessStatus), "active")
		row.LicenseKey = link.LicenseKey
		row.DeviceID = link.DeviceID
		row.UserID = link.UserID

		if row.AccessStatus != "active" {
			row.Issues = append(row.Issues, "access_not_active")
		}
		if strings.TrimSpace(link.LicenseKey) == "" {
			row.Issues = append(row.Issues, "missing_license_key")
		}

		if license, ok := licenseByKey[link.LicenseKey]; ok {
			row.LicenseRole = license.Role
			row.LicenseActive = license.Activated
			row.LicenseAssigned = license.DisplayName
			if !license.Activated {
				row.Issues = append(row.Issues, "license_not_activated")
			}
			if strings.TrimSpace(link.DeviceID) == "" && strings.TrimSpace(license.DeviceHash) != "" {
				if device, ok := deviceByMachine[license.DeviceHash]; ok {
					row.DeviceID = device.ID
				}
			}
			if row.DeviceID != "" && strings.TrimSpace(license.DeviceHash) != "" {
				if device, ok := deviceByID[row.DeviceID]; ok && device.MachineID != "" && device.MachineID != license.DeviceHash {
					row.Issues = append(row.Issues, "license_device_mismatch")
				}
			}
		} else {
			row.Issues = append(row.Issues, "missing_license_record")
		}

		if row.DeviceID != "" {
			if device, ok := deviceByID[row.DeviceID]; ok {
				row.DeviceName = device.DeviceName
				row.DeviceStatus = device.Status
				if device.LastSeenAt != nil {
					row.LastSeenAt = device.LastSeenAt.UTC().Format(time.RFC3339)
				}
				switch device.Status {
				case "approved":
				case "pending":
					row.Issues = append(row.Issues, "device_pending")
				case "blocked":
					row.Issues = append(row.Issues, "device_blocked")
				case "revoked":
					row.Issues = append(row.Issues, "device_revoked")
				case "first_setup":
					row.Issues = append(row.Issues, "device_first_setup")
				default:
					row.Issues = append(row.Issues, "device_status_unknown")
				}
			} else {
				row.Issues = append(row.Issues, "missing_device_record")
			}
		} else {
			row.Issues = append(row.Issues, "device_not_linked")
		}

		if row.UserID != "" {
			if user, ok := userByID[row.UserID]; ok {
				row.UserName = firstNonEmptyPhase7(user.FullName, user.DisplayName)
				if !user.IsActive {
					row.Issues = append(row.Issues, "user_inactive")
				}
			} else {
				row.Issues = append(row.Issues, "missing_user_record")
			}
		} else {
			row.Issues = append(row.Issues, "user_not_linked")
		}

		row.ReadyForPilot = len(row.Issues) == 0
		rows = append(rows, row)
	}

	return rows
}

func (a *App) loadPilotDeploymentChecklist() []PilotChecklistItem {
	defaults := defaultPilotDeploymentChecklist()
	if a.db == nil {
		return defaults
	}

	var setting Setting
	if err := a.db.Where("key = ?", pilotDeploymentChecklistKey).First(&setting).Error; err != nil {
		return defaults
	}

	raw := strings.TrimSpace(setting.Value)
	if raw == "" {
		return defaults
	}

	var saved []PilotChecklistItem
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		log.Printf("WARN: invalid pilot checklist setting, using defaults: %v", err)
		return defaults
	}

	overrides := make(map[string]PilotChecklistItem, len(saved))
	for _, item := range saved {
		overrides[item.ID] = item
	}

	merged := make([]PilotChecklistItem, 0, len(defaults))
	for _, item := range defaults {
		if override, ok := overrides[item.ID]; ok {
			item.Completed = override.Completed
			item.Notes = strings.TrimSpace(override.Notes)
			item.CompletedAt = strings.TrimSpace(override.CompletedAt)
		}
		merged = append(merged, item)
	}
	return merged
}

func defaultPilotDeploymentChecklist() []PilotChecklistItem {
	return []PilotChecklistItem{
		{
			ID:          "access_mapping",
			Title:       "Confirm employee-license-device mapping",
			Description: "Link pilot employees to the correct license keys, confirm approved devices, and resolve any access mismatches before rollout.",
		},
		{
			ID:          "connectivity_check",
			Title:       "Verify collaborative sync is live",
			Description: "Ensure pilot devices can reach the collaborative lane and complete at least one successful fast task/notification sync while online.",
		},
		{
			ID:          "task_assignment",
			Title:       "Run cross-device task assignment test",
			Description: "Have Alex Rivera assign, reassign, and complete a task across pilot devices to confirm notifications, queue replay, and activity tracking.",
		},
		{
			ID:          "finance_permissions",
			Title:       "Validate finance and payroll permissions",
			Description: "Confirm only the intended roles can access expenses, approvals, payroll runs, payout tracking, and related admin surfaces.",
		},
		{
			ID:          "support_bundle",
			Title:       "Export and review a pilot support bundle",
			Description: "Capture a rollout support bundle after the first pilot exercise so queue health, license inventory, and readiness issues are documented.",
		},
		{
			ID:          "user_training",
			Title:       "Complete admin and manager onboarding",
			Description: "Walk Alex Rivera and pilot managers through task assignment, access relinking, queue recovery, and the deployment dashboard before full rollout.",
		},
		{
			ID:          "pilot_signoff",
			Title:       "Record pilot sign-off",
			Description: "Capture final notes, blockers, and approval to expand from pilot devices to the broader employee rollout.",
		},
	}
}

func (a *App) linkedLicenseSubquery() []string {
	var keys []string
	if a.db == nil {
		return keys
	}
	_ = a.db.Model(&EmployeeAccessLink{}).Where("license_key <> ''").Pluck("license_key", &keys).Error
	if len(keys) == 0 {
		return []string{""}
	}
	return keys
}

func (a *App) exportableCollaborativeOps() []CollaborativePendingOperation {
	if a.db == nil {
		return nil
	}
	var ops []CollaborativePendingOperation
	_ = a.db.Where("status IN ?", []string{"pending", "failed", "dead_letter"}).
		Order("updated_at DESC, created_at DESC").
		Limit(250).
		Find(&ops).Error
	return ops
}

func (a *App) exportableDevices() []Device {
	if a.db == nil {
		return nil
	}
	var devices []Device
	_ = a.db.Order("status ASC, first_seen_at DESC").Find(&devices).Error
	return devices
}

func (a *App) exportableLicenses() []LicenseKey {
	if a.db == nil {
		return nil
	}
	var licenses []LicenseKey
	_ = a.db.Order("issued_at DESC").Find(&licenses).Error
	return licenses
}

func containsIssueTag(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (a *App) normalizeCollaborativePendingOperations() error {
	if a.db == nil || !a.db.Migrator().HasTable("collaborative_pending_operations") {
		return nil
	}

	now := time.Now()
	if err := a.db.Model(&CollaborativePendingOperation{}).
		Where("status IN ? AND next_attempt_at IS NULL", []string{"pending", "failed"}).
		Updates(map[string]any{"next_attempt_at": &now, "updated_at": now}).Error; err != nil {
		return fmt.Errorf("failed to normalize collaborative queue: %w", err)
	}

	if err := a.db.Model(&CollaborativePendingOperation{}).
		Where("status = ? AND attempts >= ?", "failed", 25).
		Updates(map[string]any{
			"status":        "dead_letter",
			"updated_at":    now,
			"error_message": gorm.Expr("COALESCE(error_message, '') || ?", " [dead-lettered after repeated retry failures]"),
		}).Error; err != nil {
		return fmt.Errorf("failed to dead-letter exhausted collaborative operations: %w", err)
	}

	return nil
}

func (a *App) backfillLegacyFollowUpTasks() error {
	if a.db == nil || !a.db.Migrator().HasTable("followup_tasks") {
		return nil
	}
	if a.hasSettingValue(phase7FollowUpBackfillSetting) {
		return nil
	}

	fallbackEmployeeID := a.getPhase7FallbackEmployeeID()
	if fallbackEmployeeID == "" {
		log.Printf("⚠️ Phase 7 rollout: no employee available for legacy follow-up backfill")
		return nil
	}

	var followUps []FollowUpTask
	if err := a.db.Order("created_at ASC").Find(&followUps).Error; err != nil {
		return fmt.Errorf("failed to load legacy follow-up tasks: %w", err)
	}
	if len(followUps) == 0 {
		a.saveSetting(phase7FollowUpBackfillSetting, time.Now().UTC().Format(time.RFC3339))
		a.saveSetting(phase7FollowUpBackfillCount, "0")
		return nil
	}

	tx := a.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	backfilled := 0
	for _, legacy := range followUps {
		var existing TaskItem
		if err := tx.Where("legacy_follow_up_id = ?", legacy.ID).First(&existing).Error; err == nil {
			continue
		} else if err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return fmt.Errorf("failed to inspect legacy follow-up migration state: %w", err)
		}

		creatorEmployeeID := a.resolveLegacyTaskEmployeeID(legacy.CreatedBy, fallbackEmployeeID)
		description := strings.TrimSpace(legacy.Description)
		if note := strings.TrimSpace(legacy.Notes); note != "" {
			if description != "" {
				description += "\n\n"
			}
			description += "Legacy notes: " + note
		}
		if contact := strings.TrimSpace(legacy.Contact); contact != "" {
			if description != "" {
				description += "\n"
			}
			description += "Legacy contact: " + contact
		}
		if legacy.Amount > 0 {
			if description != "" {
				description += "\n"
			}
			description += fmt.Sprintf("Legacy amount reference: %.3f BHD", legacy.Amount)
		}

		status := mapLegacyFollowUpStatus(legacy.Status, legacy.CompletedAt, legacy.DueDate)
		legacyID := legacy.ID
		task := TaskItem{
			Base: Base{
				ID:        uuid.New().String(),
				CreatedAt: legacy.CreatedAt,
				UpdatedAt: legacy.UpdatedAt,
				CreatedBy: legacy.CreatedBy,
			},
			Title:              firstNonEmptyPhase7(strings.TrimSpace(legacy.Title), "Legacy Follow-Up"),
			Description:        description,
			TaskType:           "customer_followup",
			LegacyFollowUpID:   &legacyID,
			Status:             status,
			Priority:           normalizeLegacyPriority(legacy.Priority),
			DueDate:            timePtrIfNonZero(legacy.DueDate),
			CustomerID:         stringPtrIfNonEmpty(strings.TrimSpace(legacy.CustomerID)),
			CreatorEmployeeID:  creatorEmployeeID,
			AssigneeEmployeeID: stringPtrIfNonEmpty(creatorEmployeeID),
			CompletedAt:        legacy.CompletedAt,
		}
		if status == "in_progress" {
			startedAt := legacy.CreatedAt
			task.StartedAt = &startedAt
		}

		if err := tx.Create(&task).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to backfill legacy follow-up task %s: %w", legacy.ID, err)
		}

		detail := fmt.Sprintf("Migrated legacy follow-up %q into collaborative tasks", task.Title)
		activity := TaskActivity{
			Base:         Base{ID: uuid.New().String(), CreatedAt: legacy.CreatedAt, UpdatedAt: legacy.CreatedAt, CreatedBy: "system"},
			TaskID:       task.ID,
			EmployeeID:   creatorEmployeeID,
			ActivityType: "migrated_legacy_followup",
			Detail:       detail,
		}
		if err := tx.Create(&activity).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create migration activity for follow-up %s: %w", legacy.ID, err)
		}

		backfilled++
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit legacy follow-up backfill: %w", err)
	}

	a.saveSetting(phase7FollowUpBackfillSetting, time.Now().UTC().Format(time.RFC3339))
	a.saveSetting(phase7FollowUpBackfillCount, fmt.Sprintf("%d", backfilled))
	log.Printf("✅ Phase 7 rollout: backfilled %d legacy follow-up tasks", backfilled)
	return nil
}

func (a *App) hasSettingValue(key string) bool {
	if a.db == nil {
		return false
	}
	var setting Setting
	if err := a.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return false
	}
	return strings.TrimSpace(setting.Value) != ""
}

func (a *App) getPhase7FallbackEmployeeID() string {
	if a.db == nil {
		return ""
	}

	var employee Employee
	if err := a.db.Where("is_active = ?", true).Order("created_at ASC").First(&employee).Error; err == nil {
		return employee.ID
	}
	if err := a.db.Order("created_at ASC").First(&employee).Error; err == nil {
		return employee.ID
	}
	return ""
}

func (a *App) resolveLegacyTaskEmployeeID(actorID, fallbackEmployeeID string) string {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return fallbackEmployeeID
	}

	var employee Employee
	if err := a.db.First(&employee, "id = ?", actorID).Error; err == nil {
		return employee.ID
	}

	var link EmployeeAccessLink
	if err := a.db.Where("user_id = ?", actorID).First(&link).Error; err == nil && strings.TrimSpace(link.EmployeeID) != "" {
		return link.EmployeeID
	}

	var user User
	if err := a.db.Select("id", "full_name", "display_name").First(&user, "id = ?", actorID).Error; err == nil {
		for _, candidate := range []string{strings.TrimSpace(user.FullName), strings.TrimSpace(user.DisplayName)} {
			if candidate == "" {
				continue
			}
			if err := a.db.Where("LOWER(full_name) = LOWER(?)", candidate).First(&employee).Error; err == nil {
				return employee.ID
			}
		}
	}

	return fallbackEmployeeID
}

func mapLegacyFollowUpStatus(status string, completedAt *time.Time, dueDate time.Time) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed":
		return "completed"
	case "in_progress":
		return "in_progress"
	case "cancelled":
		return "archived"
	case "overdue":
		return "blocked"
	}
	if completedAt != nil {
		return "completed"
	}
	if !dueDate.IsZero() && dueDate.Before(time.Now()) {
		return "blocked"
	}
	return "open"
}

func normalizeLegacyPriority(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "low", "medium", "high", "urgent":
		return strings.ToLower(strings.TrimSpace(priority))
	default:
		return "medium"
	}
}

func stringPtrIfNonEmpty(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func timePtrIfNonZero(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	v := value
	return &v
}

func firstNonEmptyPhase7(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
