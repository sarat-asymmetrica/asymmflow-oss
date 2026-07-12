package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"
)

const (
	userActivityTableEvents          = "user_activity_events"
	userActivityTableSessions        = "user_activity_sessions"
	userActivityTableWeeklySummaries = "user_activity_weekly_summaries"
	adminMonitoringLicenseKey        = "PH-ADM-DEMO01"
)

var defaultMonitoringAllowedNames = map[string]struct{}{
	"jordan": {},
	"sam":    {},
}

var defaultMonitoringAllowedKeys = map[string]struct{}{
	adminMonitoringLicenseKey: {},
	masterKey:                 {},
}

type UserActivityEvent struct {
	Base
	EventTime         time.Time `gorm:"index" json:"event_time"`
	EventType         string    `gorm:"index;size:40" json:"event_type"`
	Category          string    `gorm:"index;size:40" json:"category"`
	Screen            string    `gorm:"index;size:100" json:"screen"`
	Route             string    `gorm:"size:255" json:"route"`
	ActionLabel       string    `gorm:"size:255" json:"action_label"`
	ActionKey         string    `gorm:"size:120" json:"action_key"`
	ResourceType      string    `gorm:"index;size:80" json:"resource_type"`
	ResourceID        string    `gorm:"index;size:120" json:"resource_id"`
	SearchText        string    `gorm:"size:255" json:"search_text"`
	SearchHash        string    `gorm:"size:64" json:"search_hash"`
	SearchRedacted    bool      `gorm:"default:false" json:"search_redacted"`
	MetadataJSON      string    `gorm:"type:text" json:"metadata_json"`
	UserID            string    `gorm:"index;size:120" json:"user_id"`
	EmployeeID        string    `gorm:"index;size:36" json:"employee_id"`
	EmployeeName      string    `gorm:"index;size:120" json:"employee_name"`
	LicenseKeyHash    string    `gorm:"index;size:64" json:"license_key_hash"`
	LicenseRole       string    `gorm:"index;size:40" json:"license_role"`
	DeviceHash        string    `gorm:"index;size:64" json:"device_hash"`
	SessionID         string    `gorm:"index;size:64" json:"session_id"`
	ActiveSeconds     int       `json:"active_seconds"`
	MeaningfulSeconds int       `json:"meaningful_seconds"`
	IdleSeconds       int       `json:"idle_seconds"`
}

func (UserActivityEvent) TableName() string { return userActivityTableEvents }

type UserActivitySession struct {
	Base
	SessionID         string     `gorm:"uniqueIndex;size:64" json:"session_id"`
	StartedAt         time.Time  `gorm:"index" json:"started_at"`
	EndedAt           *time.Time `gorm:"index" json:"ended_at,omitempty"`
	LastSeenAt        time.Time  `gorm:"index" json:"last_seen_at"`
	UserID            string     `gorm:"index;size:120" json:"user_id"`
	EmployeeID        string     `gorm:"index;size:36" json:"employee_id"`
	EmployeeName      string     `gorm:"index;size:120" json:"employee_name"`
	LicenseKeyHash    string     `gorm:"index;size:64" json:"license_key_hash"`
	LicenseRole       string     `gorm:"index;size:40" json:"license_role"`
	DeviceHash        string     `gorm:"index;size:64" json:"device_hash"`
	Source            string     `gorm:"size:40" json:"source"`
	PrimaryScreen     string     `gorm:"size:100" json:"primary_screen"`
	IsOpen            bool       `gorm:"index;default:true" json:"is_open"`
	ActiveSeconds     int        `json:"active_seconds"`
	MeaningfulSeconds int        `json:"meaningful_seconds"`
	IdleSeconds       int        `json:"idle_seconds"`
	EventCount        int        `json:"event_count"`
	SearchCount       int        `json:"search_count"`
	CreateCount       int        `json:"create_count"`
	UpdateCount       int        `json:"update_count"`
	ExportCount       int        `json:"export_count"`
	NavigationCount   int        `json:"navigation_count"`
}

func (UserActivitySession) TableName() string { return userActivityTableSessions }

type UserActivityWeeklySummary struct {
	Base
	WeekStart         time.Time `gorm:"index" json:"week_start"`
	WeekEnd           time.Time `gorm:"index" json:"week_end"`
	GeneratedAt       time.Time `gorm:"index" json:"generated_at"`
	UserKey           string    `gorm:"index;size:120" json:"user_key"`
	UserID            string    `gorm:"index;size:120" json:"user_id"`
	EmployeeID        string    `gorm:"index;size:36" json:"employee_id"`
	EmployeeName      string    `gorm:"index;size:120" json:"employee_name"`
	LicenseKeyHash    string    `gorm:"index;size:64" json:"license_key_hash"`
	LicenseRole       string    `gorm:"index;size:40" json:"license_role"`
	DeviceHash        string    `gorm:"index;size:64" json:"device_hash"`
	ActiveSeconds     int       `json:"active_seconds"`
	MeaningfulSeconds int       `json:"meaningful_seconds"`
	IdleSeconds       int       `json:"idle_seconds"`
	EventCount        int       `json:"event_count"`
	SearchCount       int       `json:"search_count"`
	CreateCount       int       `json:"create_count"`
	UpdateCount       int       `json:"update_count"`
	ExportCount       int       `json:"export_count"`
	NavigationCount   int       `json:"navigation_count"`
	EfficiencyScore   float64   `json:"efficiency_score"`
	TopScreensJSON    string    `gorm:"type:text" json:"top_screens_json"`
	TopActionsJSON    string    `gorm:"type:text" json:"top_actions_json"`
	TopSearchesJSON   string    `gorm:"type:text" json:"top_searches_json"`
}

func (UserActivityWeeklySummary) TableName() string { return userActivityTableWeeklySummaries }

type UserActivityEventInput struct {
	SessionID         string         `json:"session_id"`
	EventTime         string         `json:"event_time"`
	EventType         string         `json:"event_type"`
	Category          string         `json:"category"`
	Screen            string         `json:"screen"`
	Route             string         `json:"route"`
	ActionLabel       string         `json:"action_label"`
	ActionKey         string         `json:"action_key"`
	ResourceType      string         `json:"resource_type"`
	ResourceID        string         `json:"resource_id"`
	SearchText        string         `json:"search_text"`
	Metadata          map[string]any `json:"metadata"`
	ActiveSeconds     int            `json:"active_seconds"`
	MeaningfulSeconds int            `json:"meaningful_seconds"`
	IdleSeconds       int            `json:"idle_seconds"`
}

type UserActivityHeartbeatInput struct {
	SessionID         string `json:"session_id"`
	Screen            string `json:"screen"`
	ActiveSeconds     int    `json:"active_seconds"`
	MeaningfulSeconds int    `json:"meaningful_seconds"`
	IdleSeconds       int    `json:"idle_seconds"`
	EventCount        int    `json:"event_count"`
	SearchCount       int    `json:"search_count"`
	CreateCount       int    `json:"create_count"`
	UpdateCount       int    `json:"update_count"`
	ExportCount       int    `json:"export_count"`
	NavigationCount   int    `json:"navigation_count"`
}

type UserActivityMetric struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type UserActivityUserReport struct {
	UserKey         string               `json:"user_key"`
	UserID          string               `json:"user_id"`
	EmployeeID      string               `json:"employee_id"`
	EmployeeName    string               `json:"employee_name"`
	LicenseRole     string               `json:"license_role"`
	ActiveHours     float64              `json:"active_hours"`
	MeaningfulHours float64              `json:"meaningful_hours"`
	IdleHours       float64              `json:"idle_hours"`
	EfficiencyScore float64              `json:"efficiency_score"`
	EventCount      int                  `json:"event_count"`
	SearchCount     int                  `json:"search_count"`
	CreateCount     int                  `json:"create_count"`
	UpdateCount     int                  `json:"update_count"`
	ExportCount     int                  `json:"export_count"`
	NavigationCount int                  `json:"navigation_count"`
	TopScreens      []UserActivityMetric `json:"top_screens"`
	TopActions      []UserActivityMetric `json:"top_actions"`
	TopSearches     []UserActivityMetric `json:"top_searches"`
	LastActivityAt  string               `json:"last_activity_at"`
}

type UserActivityChartRow struct {
	Label           string  `json:"label"`
	ActiveHours     float64 `json:"active_hours"`
	MeaningfulHours float64 `json:"meaningful_hours"`
	EfficiencyScore float64 `json:"efficiency_score"`
}

type UserActivityWeeklyReport struct {
	WeekStart             string                   `json:"week_start"`
	WeekEnd               string                   `json:"week_end"`
	GeneratedAt           string                   `json:"generated_at"`
	TotalActiveHours      float64                  `json:"total_active_hours"`
	TotalMeaningfulHours  float64                  `json:"total_meaningful_hours"`
	AverageEfficiency     float64                  `json:"average_efficiency"`
	UserCount             int                      `json:"user_count"`
	Users                 []UserActivityUserReport `json:"users"`
	ChartRows             []UserActivityChartRow   `json:"chart_rows"`
	MonitoringPrincipals  []string                 `json:"monitoring_principals"`
	ConfidentialityNotice string                   `json:"confidentiality_notice"`
}

type userActivityIdentity struct {
	UserID         string
	EmployeeID     string
	EmployeeName   string
	LicenseKey     string
	LicenseKeyHash string
	LicenseRole    string
	DeviceHash     string
}

func isUserActivitySyncTable(table string) bool {
	switch strings.ToLower(strings.TrimSpace(table)) {
	case userActivityTableEvents, userActivityTableSessions, userActivityTableWeeklySummaries:
		return true
	default:
		return false
	}
}

// Mission I (I-11): bound DDL is gated; startup and internal lazy callers use
// the internal.
func (a *App) EnsureUserActivityMonitoringFoundation() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	return a.ensureUserActivityMonitoringFoundationInternal()
}

func (a *App) ensureUserActivityMonitoringFoundationInternal() error {
	if a == nil || a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return a.db.AutoMigrate(&UserActivitySession{}, &UserActivityEvent{}, &UserActivityWeeklySummary{})
}

func (a *App) CanViewUserActivityMonitoring() bool {
	return a.currentSessionCanAccessActivityMonitoring()
}

func (a *App) requireUserActivityMonitoringAccess() error {
	if a.currentSessionCanAccessActivityMonitoring() {
		return nil
	}
	return fmt.Errorf("access denied: confidential activity monitoring is restricted to developer role")
}

func (a *App) currentSessionCanAccessActivityMonitoring() bool {
	if a == nil || a.db == nil {
		return false
	}

	allowedKeys := monitoringAllowedKeySet()
	allowedNames := monitoringAllowedNameSet()

	if license, err := a.getActiveLicenseRecord(); err == nil && license != nil {
		if _, ok := allowedKeys[strings.ToUpper(strings.TrimSpace(license.Key))]; ok {
			return true
		}
		if monitoringNameAllowed(license.DisplayName, allowedNames) {
			return true
		}
	}

	if employeeCtx, err := a.GetCurrentEmployeeContext(); err == nil {
		if _, ok := allowedKeys[strings.ToUpper(strings.TrimSpace(employeeCtx.LicenseKey))]; ok {
			return true
		}
		if monitoringNameAllowed(employeeCtx.EmployeeName, allowedNames) {
			return true
		}
	}

	if a.currentUser != nil {
		for _, candidate := range []string{a.currentUser.DisplayName, a.currentUser.FullName, a.currentUser.Username, a.currentUser.Email} {
			if monitoringNameAllowed(candidate, allowedNames) {
				return true
			}
		}
	}

	if a.authManager != nil {
		a.authManager.mu.RLock()
		profile := a.authManager.Profile
		a.authManager.mu.RUnlock()
		if profile != nil {
			for _, candidate := range []string{profile.DisplayName, profile.UserPrincipalName, profile.Mail} {
				if monitoringNameAllowed(candidate, allowedNames) {
					return true
				}
			}
		}
	}

	return false
}

func monitoringAllowedKeySet() map[string]struct{} {
	allowed := make(map[string]struct{}, len(defaultMonitoringAllowedKeys))
	for key := range defaultMonitoringAllowedKeys {
		allowed[strings.ToUpper(strings.TrimSpace(key))] = struct{}{}
	}
	for _, key := range strings.Split(os.Getenv("ASYMMFLOW_MONITORING_ALLOWED_KEYS"), ",") {
		key = strings.ToUpper(strings.TrimSpace(key))
		if key != "" {
			allowed[key] = struct{}{}
		}
	}
	return allowed
}

func monitoringAllowedNameSet() map[string]struct{} {
	allowed := make(map[string]struct{}, len(defaultMonitoringAllowedNames))
	for name := range defaultMonitoringAllowedNames {
		allowed[name] = struct{}{}
	}
	for _, name := range strings.Split(os.Getenv("ASYMMFLOW_MONITORING_ALLOWED_NAMES"), ",") {
		normalized := normalizeMonitoringPrincipalName(name)
		if normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}
	return allowed
}

func monitoringNameAllowed(name string, allowed map[string]struct{}) bool {
	normalized := normalizeMonitoringPrincipalName(name)
	if normalized == "" {
		return false
	}
	if _, ok := allowed[normalized]; ok {
		return true
	}
	for allowedName := range allowed {
		if allowedName != "" && strings.Contains(normalized, allowedName) {
			return true
		}
	}
	return false
}

func normalizeMonitoringPrincipalName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (a *App) currentUserActivityIdentity() userActivityIdentity {
	identity := userActivityIdentity{
		UserID:       "system",
		EmployeeName: "System",
	}
	if a == nil {
		return identity
	}
	if a.db != nil {
		deviceHash := a.getDeviceHash()
		identity.DeviceHash = deviceHash
		if license, err := a.getActiveLicenseRecord(); err == nil && license != nil {
			identity.LicenseKey = strings.ToUpper(strings.TrimSpace(license.Key))
			identity.LicenseKeyHash = hashActivityValue(identity.LicenseKey)
			identity.LicenseRole = strings.ToLower(strings.TrimSpace(license.Role))
			if strings.TrimSpace(license.DisplayName) != "" {
				identity.EmployeeName = strings.TrimSpace(license.DisplayName)
			}
			identity.UserID = firstNonEmptyString(identity.EmployeeName, "license:"+identity.LicenseRole)
		}
	}
	if employeeCtx, err := a.GetCurrentEmployeeContext(); err == nil {
		identity.EmployeeID = employeeCtx.EmployeeID
		if strings.TrimSpace(employeeCtx.EmployeeName) != "" {
			identity.EmployeeName = strings.TrimSpace(employeeCtx.EmployeeName)
		}
		if strings.TrimSpace(employeeCtx.UserID) != "" {
			identity.UserID = strings.TrimSpace(employeeCtx.UserID)
		}
		if strings.TrimSpace(employeeCtx.LicenseKey) != "" {
			identity.LicenseKey = strings.ToUpper(strings.TrimSpace(employeeCtx.LicenseKey))
			identity.LicenseKeyHash = hashActivityValue(identity.LicenseKey)
		}
		if strings.TrimSpace(employeeCtx.LicenseRole) != "" {
			identity.LicenseRole = strings.ToLower(strings.TrimSpace(employeeCtx.LicenseRole))
		}
		if strings.TrimSpace(employeeCtx.DeviceID) != "" && identity.DeviceHash == "" {
			identity.DeviceHash = strings.TrimSpace(employeeCtx.DeviceID)
		}
	}
	if a.currentUser != nil {
		if strings.TrimSpace(a.currentUser.ID) != "" {
			identity.UserID = strings.TrimSpace(a.currentUser.ID)
		}
		if strings.TrimSpace(a.currentUser.Role.Name) != "" && identity.LicenseRole == "" {
			identity.LicenseRole = strings.ToLower(strings.TrimSpace(a.currentUser.Role.Name))
		}
		if identity.EmployeeName == "" || identity.EmployeeName == "System" {
			identity.EmployeeName = firstNonEmptyString(a.currentUser.DisplayName, a.currentUser.FullName, a.currentUser.Username, "User")
		}
	}
	if identity.UserID == "" {
		identity.UserID = a.getCurrentUserID()
	}
	if identity.EmployeeName == "" || identity.EmployeeName == "System" {
		identity.EmployeeName = a.getCurrentUserDisplayName()
	}
	if identity.LicenseKeyHash == "" && identity.LicenseKey != "" {
		identity.LicenseKeyHash = hashActivityValue(identity.LicenseKey)
	}
	if identity.LicenseRole == "" {
		identity.LicenseRole = strings.ToLower(strings.TrimSpace(a.GetCurrentUserRole()))
	}
	return identity
}

func hashActivityValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (a *App) StartUserActivitySession(source string) (UserActivitySession, error) {
	if a == nil || a.db == nil {
		return UserActivitySession{}, fmt.Errorf("database not initialized")
	}
	// Wave 8 P0: PH gates activity telemetry writes with dashboard:view.
	if err := a.requirePermission("dashboard:view"); err != nil {
		return UserActivitySession{}, err
	}
	if err := a.ensureUserActivityMonitoringFoundationInternal(); err != nil {
		return UserActivitySession{}, err
	}

	identity := a.currentUserActivityIdentity()
	now := time.Now()
	session := UserActivitySession{
		Base:           Base{CreatedBy: identity.UserID},
		SessionID:      fmt.Sprintf("uas_%d_%s", now.UnixNano(), shortHash(identity.DeviceHash+identity.UserID)),
		StartedAt:      now,
		LastSeenAt:     now,
		UserID:         identity.UserID,
		EmployeeID:     identity.EmployeeID,
		EmployeeName:   identity.EmployeeName,
		LicenseKeyHash: identity.LicenseKeyHash,
		LicenseRole:    identity.LicenseRole,
		DeviceHash:     identity.DeviceHash,
		Source:         limitActivityField(source, 40),
		IsOpen:         true,
	}
	if session.Source == "" {
		session.Source = "desktop"
	}
	if err := a.db.Create(&session).Error; err != nil {
		return UserActivitySession{}, fmt.Errorf("failed to start activity session: %w", err)
	}
	return session, nil
}

func (a *App) RecordUserActivityBatch(events []UserActivityEventInput) error {
	if a == nil || a.db == nil {
		return nil
	}
	// Wave 8 P0: PH gates activity telemetry writes with dashboard:view.
	if err := a.requirePermission("dashboard:view"); err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}
	if err := a.ensureUserActivityMonitoringFoundationInternal(); err != nil {
		return err
	}

	identity := a.currentUserActivityIdentity()
	records := make([]UserActivityEvent, 0, len(events))
	sessionDeltas := map[string]UserActivityHeartbeatInput{}
	for _, input := range events {
		record := a.userActivityEventFromInput(input, identity)
		records = append(records, record)
		if record.SessionID != "" {
			delta := sessionDeltas[record.SessionID]
			delta.SessionID = record.SessionID
			delta.Screen = firstNonEmptyString(delta.Screen, record.Screen)
			delta.ActiveSeconds += record.ActiveSeconds
			delta.MeaningfulSeconds += record.MeaningfulSeconds
			delta.IdleSeconds += record.IdleSeconds
			delta.EventCount++
			switch record.Category {
			case "search":
				delta.SearchCount++
			case "create":
				delta.CreateCount++
			case "update", "save", "edit":
				delta.UpdateCount++
			case "export":
				delta.ExportCount++
			case "navigation":
				delta.NavigationCount++
			}
			sessionDeltas[record.SessionID] = delta
		}
	}

	if err := a.db.CreateInBatches(records, 100).Error; err != nil {
		return fmt.Errorf("failed to record user activity: %w", err)
	}
	for _, delta := range sessionDeltas {
		if err := a.applyUserActivitySessionDelta(delta, false); err != nil {
			log.Printf("activity monitoring session delta warning: %v", err)
		}
	}
	return nil
}

func (a *App) RecordUserActivityHeartbeat(input UserActivityHeartbeatInput) error {
	if a == nil || a.db == nil {
		return nil
	}
	// Wave 8 P0: PH gates activity telemetry writes with dashboard:view.
	if err := a.requirePermission("dashboard:view"); err != nil {
		return err
	}
	if err := a.ensureUserActivityMonitoringFoundationInternal(); err != nil {
		return err
	}
	identity := a.currentUserActivityIdentity()
	event := UserActivityEventInput{
		SessionID:         input.SessionID,
		EventType:         "heartbeat",
		Category:          "heartbeat",
		Screen:            input.Screen,
		ActiveSeconds:     input.ActiveSeconds,
		MeaningfulSeconds: input.MeaningfulSeconds,
		IdleSeconds:       input.IdleSeconds,
		Metadata: map[string]any{
			"event_count":      input.EventCount,
			"search_count":     input.SearchCount,
			"create_count":     input.CreateCount,
			"update_count":     input.UpdateCount,
			"export_count":     input.ExportCount,
			"navigation_count": input.NavigationCount,
		},
	}
	record := a.userActivityEventFromInput(event, identity)
	if err := a.db.Create(&record).Error; err != nil {
		return fmt.Errorf("failed to record activity heartbeat: %w", err)
	}
	return a.applyUserActivitySessionDelta(input, false)
}

func (a *App) EndUserActivitySession(sessionID string) error {
	if a == nil || a.db == nil {
		return nil
	}
	// Wave 8 P0: PH gates activity telemetry writes with dashboard:view.
	if err := a.requirePermission("dashboard:view"); err != nil {
		return err
	}
	sessionID = limitActivityField(sessionID, 64)
	if sessionID == "" {
		return nil
	}
	now := time.Now()
	return a.db.Model(&UserActivitySession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]any{
			"last_seen_at": now,
			"ended_at":     &now,
			"is_open":      false,
		}).Error
}

func (a *App) userActivityEventFromInput(input UserActivityEventInput, identity userActivityIdentity) UserActivityEvent {
	eventTime := time.Now()
	if strings.TrimSpace(input.EventTime) != "" {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(input.EventTime)); err == nil {
			eventTime = parsed
		}
	}
	searchText, searchHash, redacted := sanitizeActivitySearchText(input.SearchText)
	metadataJSON := sanitizeActivityMetadata(input.Metadata)
	category := classifyActivityCategory(input.Category, input.EventType, input.ActionLabel)
	return UserActivityEvent{
		Base:              Base{CreatedBy: identity.UserID},
		EventTime:         eventTime,
		EventType:         limitActivityField(firstNonEmptyString(input.EventType, category), 40),
		Category:          limitActivityField(category, 40),
		Screen:            limitActivityField(input.Screen, 100),
		Route:             limitActivityField(input.Route, 255),
		ActionLabel:       limitActivityField(input.ActionLabel, 255),
		ActionKey:         limitActivityField(input.ActionKey, 120),
		ResourceType:      limitActivityField(input.ResourceType, 80),
		ResourceID:        limitActivityField(input.ResourceID, 120),
		SearchText:        searchText,
		SearchHash:        searchHash,
		SearchRedacted:    redacted,
		MetadataJSON:      metadataJSON,
		UserID:            identity.UserID,
		EmployeeID:        identity.EmployeeID,
		EmployeeName:      identity.EmployeeName,
		LicenseKeyHash:    identity.LicenseKeyHash,
		LicenseRole:       identity.LicenseRole,
		DeviceHash:        identity.DeviceHash,
		SessionID:         limitActivityField(input.SessionID, 64),
		ActiveSeconds:     clampActivitySeconds(input.ActiveSeconds),
		MeaningfulSeconds: clampActivitySeconds(input.MeaningfulSeconds),
		IdleSeconds:       clampActivitySeconds(input.IdleSeconds),
	}
}

func (a *App) applyUserActivitySessionDelta(input UserActivityHeartbeatInput, closeSession bool) error {
	sessionID := limitActivityField(input.SessionID, 64)
	if sessionID == "" {
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"last_seen_at":       now,
		"active_seconds":     gorm.Expr("active_seconds + ?", clampActivitySeconds(input.ActiveSeconds)),
		"meaningful_seconds": gorm.Expr("meaningful_seconds + ?", clampActivitySeconds(input.MeaningfulSeconds)),
		"idle_seconds":       gorm.Expr("idle_seconds + ?", clampActivitySeconds(input.IdleSeconds)),
		"event_count":        gorm.Expr("event_count + ?", maxActivityInt(input.EventCount, 0)),
		"search_count":       gorm.Expr("search_count + ?", maxActivityInt(input.SearchCount, 0)),
		"create_count":       gorm.Expr("create_count + ?", maxActivityInt(input.CreateCount, 0)),
		"update_count":       gorm.Expr("update_count + ?", maxActivityInt(input.UpdateCount, 0)),
		"export_count":       gorm.Expr("export_count + ?", maxActivityInt(input.ExportCount, 0)),
		"navigation_count":   gorm.Expr("navigation_count + ?", maxActivityInt(input.NavigationCount, 0)),
	}
	if strings.TrimSpace(input.Screen) != "" {
		updates["primary_screen"] = limitActivityField(input.Screen, 100)
	}
	if closeSession {
		updates["ended_at"] = &now
		updates["is_open"] = false
	}
	return a.db.Model(&UserActivitySession{}).Where("session_id = ?", sessionID).Updates(updates).Error
}

func sanitizeActivitySearchText(value string) (string, string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", false
	}
	hash := hashActivityValue(strings.ToLower(value))
	lower := strings.ToLower(value)
	sensitiveMarkers := []string{"password", "secret", "token", "api_key", "apikey", "license", "key=", "bearer", "credential"}
	for _, marker := range sensitiveMarkers {
		if strings.Contains(lower, marker) {
			return "[redacted]", hash, true
		}
	}
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) > 120 {
		runes := []rune(value)
		value = string(runes[:120])
	}
	return value, hash, false
}

func sanitizeActivityMetadata(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	safe := make(map[string]any, len(metadata))
	for key, value := range metadata {
		key = limitActivityField(key, 80)
		if key == "" {
			continue
		}
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "password") ||
			strings.Contains(lowerKey, "secret") ||
			strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "key") ||
			strings.Contains(lowerKey, "credential") {
			safe[key] = "[redacted]"
			continue
		}
		switch typed := value.(type) {
		case string:
			safe[key] = limitActivityField(typed, 255)
		case float64, float32, int, int64, bool, nil:
			safe[key] = typed
		default:
			safe[key] = limitActivityField(fmt.Sprintf("%v", typed), 255)
		}
	}
	payload, err := json.Marshal(safe)
	if err != nil {
		return ""
	}
	return string(payload)
}

func classifyActivityCategory(category, eventType, label string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	if category != "" {
		return category
	}
	text := strings.ToLower(strings.TrimSpace(eventType + " " + label))
	switch {
	case strings.Contains(text, "search") || strings.Contains(text, "filter"):
		return "search"
	case strings.Contains(text, "create") || strings.Contains(text, "add") || strings.Contains(text, "new"):
		return "create"
	case strings.Contains(text, "save") || strings.Contains(text, "update") || strings.Contains(text, "edit") || strings.Contains(text, "approve"):
		return "update"
	case strings.Contains(text, "export") || strings.Contains(text, "download") || strings.Contains(text, "print") || strings.Contains(text, "pdf") || strings.Contains(text, "excel"):
		return "export"
	case strings.Contains(text, "delete") || strings.Contains(text, "remove"):
		return "delete"
	case strings.Contains(text, "navigation") || strings.Contains(text, "navigate"):
		return "navigation"
	case strings.Contains(text, "heartbeat"):
		return "heartbeat"
	default:
		return "action"
	}
}

func limitActivityField(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if value == "" || maxLen <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	return string(runes[:maxLen])
}

func clampActivitySeconds(seconds int) int {
	if seconds < 0 {
		return 0
	}
	if seconds > 3600 {
		return 3600
	}
	return seconds
}

func shortHash(value string) string {
	hash := hashActivityValue(value)
	if len(hash) > 10 {
		return hash[:10]
	}
	if hash == "" {
		return "session"
	}
	return hash
}

func maxActivityInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type activityAggregate struct {
	UserKey           string
	UserID            string
	EmployeeID        string
	EmployeeName      string
	LicenseKeyHash    string
	LicenseRole       string
	DeviceHash        string
	ActiveSeconds     int
	MeaningfulSeconds int
	IdleSeconds       int
	EventCount        int
	SearchCount       int
	CreateCount       int
	UpdateCount       int
	ExportCount       int
	NavigationCount   int
	LastActivityAt    time.Time
	ScreenCounts      map[string]int
	ActionCounts      map[string]int
	SearchCounts      map[string]int
}

func (a *App) GetWeeklyUserActivityReport(weekStart string) (UserActivityWeeklyReport, error) {
	if err := a.requireUserActivityMonitoringAccess(); err != nil {
		return UserActivityWeeklyReport{}, err
	}
	if a == nil || a.db == nil {
		return UserActivityWeeklyReport{}, fmt.Errorf("database not initialized")
	}
	if err := a.ensureUserActivityMonitoringFoundationInternal(); err != nil {
		return UserActivityWeeklyReport{}, err
	}

	start := parseActivityWeekStart(weekStart, time.Now())
	end := start.AddDate(0, 0, 7)
	var events []UserActivityEvent
	if err := a.db.Where("event_time >= ? AND event_time < ?", start, end).
		Order("event_time ASC").
		Find(&events).Error; err != nil {
		return UserActivityWeeklyReport{}, fmt.Errorf("failed to load activity events: %w", err)
	}

	aggregates := map[string]*activityAggregate{}
	for _, event := range events {
		userKey := activityUserKey(event)
		agg := aggregates[userKey]
		if agg == nil {
			agg = &activityAggregate{
				UserKey:        userKey,
				UserID:         event.UserID,
				EmployeeID:     event.EmployeeID,
				EmployeeName:   firstNonEmptyString(event.EmployeeName, event.UserID, "Unknown User"),
				LicenseKeyHash: event.LicenseKeyHash,
				LicenseRole:    event.LicenseRole,
				DeviceHash:     event.DeviceHash,
				ScreenCounts:   map[string]int{},
				ActionCounts:   map[string]int{},
				SearchCounts:   map[string]int{},
			}
			aggregates[userKey] = agg
		}
		agg.ActiveSeconds += event.ActiveSeconds
		agg.MeaningfulSeconds += event.MeaningfulSeconds
		agg.IdleSeconds += event.IdleSeconds
		agg.EventCount++
		switch event.Category {
		case "search":
			agg.SearchCount++
		case "create":
			agg.CreateCount++
		case "update", "save", "edit":
			agg.UpdateCount++
		case "export":
			agg.ExportCount++
		case "navigation":
			agg.NavigationCount++
		}
		if event.Screen != "" {
			agg.ScreenCounts[event.Screen]++
		}
		action := firstNonEmptyString(event.ActionLabel, event.ActionKey, event.EventType)
		if action != "" && event.Category != "heartbeat" {
			agg.ActionCounts[action]++
		}
		if event.SearchText != "" && event.SearchText != "[redacted]" {
			agg.SearchCounts[event.SearchText]++
		}
		if event.EventTime.After(agg.LastActivityAt) {
			agg.LastActivityAt = event.EventTime
		}
	}

	users := make([]UserActivityUserReport, 0, len(aggregates))
	summaries := make([]UserActivityWeeklySummary, 0, len(aggregates))
	totalActive := 0
	totalMeaningful := 0
	totalEfficiency := 0.0
	for _, agg := range aggregates {
		efficiency := activityEfficiencyScore(agg.MeaningfulSeconds, agg.ActiveSeconds)
		totalActive += agg.ActiveSeconds
		totalMeaningful += agg.MeaningfulSeconds
		totalEfficiency += efficiency
		topScreens := topActivityMetrics(agg.ScreenCounts, 5)
		topActions := topActivityMetrics(agg.ActionCounts, 5)
		topSearches := topActivityMetrics(agg.SearchCounts, 5)
		users = append(users, UserActivityUserReport{
			UserKey:         agg.UserKey,
			UserID:          agg.UserID,
			EmployeeID:      agg.EmployeeID,
			EmployeeName:    agg.EmployeeName,
			LicenseRole:     agg.LicenseRole,
			ActiveHours:     secondsToHours(agg.ActiveSeconds),
			MeaningfulHours: secondsToHours(agg.MeaningfulSeconds),
			IdleHours:       secondsToHours(agg.IdleSeconds),
			EfficiencyScore: efficiency,
			EventCount:      agg.EventCount,
			SearchCount:     agg.SearchCount,
			CreateCount:     agg.CreateCount,
			UpdateCount:     agg.UpdateCount,
			ExportCount:     agg.ExportCount,
			NavigationCount: agg.NavigationCount,
			TopScreens:      topScreens,
			TopActions:      topActions,
			TopSearches:     topSearches,
			LastActivityAt:  formatOptionalActivityTime(agg.LastActivityAt),
		})
		summaries = append(summaries, UserActivityWeeklySummary{
			Base:              Base{CreatedBy: "activity_monitor"},
			WeekStart:         start,
			WeekEnd:           end,
			GeneratedAt:       time.Now(),
			UserKey:           agg.UserKey,
			UserID:            agg.UserID,
			EmployeeID:        agg.EmployeeID,
			EmployeeName:      agg.EmployeeName,
			LicenseKeyHash:    agg.LicenseKeyHash,
			LicenseRole:       agg.LicenseRole,
			DeviceHash:        agg.DeviceHash,
			ActiveSeconds:     agg.ActiveSeconds,
			MeaningfulSeconds: agg.MeaningfulSeconds,
			IdleSeconds:       agg.IdleSeconds,
			EventCount:        agg.EventCount,
			SearchCount:       agg.SearchCount,
			CreateCount:       agg.CreateCount,
			UpdateCount:       agg.UpdateCount,
			ExportCount:       agg.ExportCount,
			NavigationCount:   agg.NavigationCount,
			EfficiencyScore:   efficiency,
			TopScreensJSON:    mustActivityMetricsJSON(topScreens),
			TopActionsJSON:    mustActivityMetricsJSON(topActions),
			TopSearchesJSON:   mustActivityMetricsJSON(topSearches),
		})
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].MeaningfulHours == users[j].MeaningfulHours {
			return users[i].EmployeeName < users[j].EmployeeName
		}
		return users[i].MeaningfulHours > users[j].MeaningfulHours
	})

	chartRows := make([]UserActivityChartRow, 0, len(users))
	for _, user := range users {
		chartRows = append(chartRows, UserActivityChartRow{
			Label:           user.EmployeeName,
			ActiveHours:     user.ActiveHours,
			MeaningfulHours: user.MeaningfulHours,
			EfficiencyScore: user.EfficiencyScore,
		})
	}

	if len(summaries) > 0 {
		if err := a.replaceWeeklyActivitySummaries(start, summaries); err != nil {
			log.Printf("activity monitoring weekly summary warning: %v", err)
		}
	}

	avgEfficiency := 0.0
	if len(users) > 0 {
		avgEfficiency = roundActivityFloat(totalEfficiency / float64(len(users)))
	}
	return UserActivityWeeklyReport{
		WeekStart:             start.Format("2006-01-02"),
		WeekEnd:               end.Add(-24 * time.Hour).Format("2006-01-02"),
		GeneratedAt:           time.Now().Format(time.RFC3339),
		TotalActiveHours:      secondsToHours(totalActive),
		TotalMeaningfulHours:  secondsToHours(totalMeaningful),
		AverageEfficiency:     avgEfficiency,
		UserCount:             len(users),
		Users:                 users,
		ChartRows:             chartRows,
		MonitoringPrincipals:  []string{"Jordan", "Sam"},
		ConfidentialityNotice: "Confidential internal efficiency report. Access is restricted to developer role.",
	}, nil
}

func (a *App) replaceWeeklyActivitySummaries(weekStart time.Time, summaries []UserActivityWeeklySummary) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("week_start = ?", weekStart).Delete(&UserActivityWeeklySummary{}).Error; err != nil {
			return err
		}
		return tx.CreateInBatches(summaries, 100).Error
	})
}

func parseActivityWeekStart(value string, fallback time.Time) time.Time {
	if strings.TrimSpace(value) != "" {
		if parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value)); err == nil {
			return startOfActivityWeek(parsed)
		}
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value)); err == nil {
			return startOfActivityWeek(parsed)
		}
	}
	return startOfActivityWeek(fallback)
}

func startOfActivityWeek(t time.Time) time.Time {
	local := t.Local()
	weekday := int(local.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	return start.AddDate(0, 0, -(weekday - 1))
}

func activityUserKey(event UserActivityEvent) string {
	return firstNonEmptyString(event.EmployeeID, event.LicenseKeyHash, event.UserID, event.DeviceHash, "unknown")
}

func activityEfficiencyScore(meaningfulSeconds, activeSeconds int) float64 {
	if activeSeconds <= 0 {
		return 0
	}
	score := (float64(meaningfulSeconds) / float64(activeSeconds)) * 100
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return roundActivityFloat(score)
}

func secondsToHours(seconds int) float64 {
	return roundActivityFloat(float64(seconds) / 3600)
}

func roundActivityFloat(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func topActivityMetrics(counts map[string]int, limit int) []UserActivityMetric {
	metrics := make([]UserActivityMetric, 0, len(counts))
	for key, count := range counts {
		if strings.TrimSpace(key) == "" || count <= 0 {
			continue
		}
		metrics = append(metrics, UserActivityMetric{
			Key:   limitActivityField(key, 120),
			Label: limitActivityField(key, 120),
			Count: count,
		})
	}
	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].Count == metrics[j].Count {
			return metrics[i].Label < metrics[j].Label
		}
		return metrics[i].Count > metrics[j].Count
	})
	if limit > 0 && len(metrics) > limit {
		return metrics[:limit]
	}
	return metrics
}

func mustActivityMetricsJSON(metrics []UserActivityMetric) string {
	payload, err := json.Marshal(metrics)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func formatOptionalActivityTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
