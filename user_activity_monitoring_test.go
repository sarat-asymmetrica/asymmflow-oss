package main

import (
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newActivityMonitoringTestApp(t *testing.T) *App {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	app := &App{
		db:     db,
		config: &Config{App: AppConfig{EnableDeveloperMasterKey: true}},
	}
	require.NoError(t, db.AutoMigrate(
		&LicenseKey{},
		&Employee{},
		&EmployeeAccessLink{},
		&UserActivitySession{},
		&UserActivityEvent{},
		&UserActivityWeeklySummary{},
	))
	return app
}

func activateActivityMonitoringLicense(t *testing.T, app *App, key, role, displayName string) {
	t.Helper()
	now := time.Now()
	require.NoError(t, app.db.Model(&LicenseKey{}).Where("1 = 1").Updates(map[string]any{
		"activated":    false,
		"device_hash":  "",
		"activated_at": nil,
	}).Error)
	require.NoError(t, app.db.Create(&LicenseKey{
		Key:         key,
		Role:        role,
		DisplayName: displayName,
		DeviceHash:  app.getDeviceHash(),
		Activated:   true,
		ActivatedAt: &now,
		Notes:       "test key",
		CreatedBy:   "test",
	}).Error)
}

func TestUserActivityMonitoringAccessIsAllowlistedOnly(t *testing.T) {
	app := newActivityMonitoringTestApp(t)

	activateActivityMonitoringLicense(t, app, "PH-ADM-E2E4C2", "admin", "Casey")
	require.False(t, app.CanViewUserActivityMonitoring())
	_, err := app.GetWeeklyUserActivityReport("")
	require.Error(t, err)

	activateActivityMonitoringLicense(t, app, adminMonitoringLicenseKey, "admin", "Jordan")
	require.True(t, app.CanViewUserActivityMonitoring())
	_, err = app.GetWeeklyUserActivityReport("")
	require.NoError(t, err)

	activateActivityMonitoringLicense(t, app, masterKey, "admin", "Developer")
	require.True(t, app.CanViewUserActivityMonitoring(), "supervisor/master key must be allowed even when the display name is Developer")
}

func TestUserActivityMonitoringRecordsAndReportsWeeklyUsage(t *testing.T) {
	app := newActivityMonitoringTestApp(t)

	activateActivityMonitoringLicense(t, app, "PH-SLS-9A70F9", "sales", "Jamie")
	session, err := app.StartUserActivitySession("test")
	require.NoError(t, err)
	require.NotEmpty(t, session.SessionID)

	err = app.RecordUserActivityBatch([]UserActivityEventInput{
		{
			SessionID:         session.SessionID,
			EventType:         "search",
			Category:          "search",
			Screen:            "opportunities",
			ActionLabel:       "Search opportunities",
			SearchText:        "pump package",
			ActiveSeconds:     120,
			MeaningfulSeconds: 90,
		},
		{
			SessionID:   session.SessionID,
			EventType:   "click",
			Category:    "create",
			Screen:      "opportunities",
			ActionLabel: "Create costing sheet",
		},
		{
			SessionID:   session.SessionID,
			EventType:   "search",
			Category:    "search",
			Screen:      "settings",
			ActionLabel: "Search",
			SearchText:  "api_key=should-not-store",
		},
	})
	require.NoError(t, err)
	require.NoError(t, app.RecordUserActivityHeartbeat(UserActivityHeartbeatInput{
		SessionID:         session.SessionID,
		Screen:            "opportunities",
		ActiveSeconds:     60,
		MeaningfulSeconds: 60,
		EventCount:        2,
		SearchCount:       1,
		CreateCount:       1,
	}))

	var redacted UserActivityEvent
	require.NoError(t, app.db.Where("search_hash <> '' AND search_redacted = ?", true).First(&redacted).Error)
	require.Equal(t, "[redacted]", redacted.SearchText)
	require.NotEmpty(t, redacted.SearchHash)

	activateActivityMonitoringLicense(t, app, adminMonitoringLicenseKey, "admin", "Jordan")
	report, err := app.GetWeeklyUserActivityReport("")
	require.NoError(t, err)
	require.Len(t, report.Users, 1)
	require.Equal(t, "Jamie", report.Users[0].EmployeeName)
	require.Greater(t, report.Users[0].MeaningfulHours, 0.0)
	require.GreaterOrEqual(t, report.Users[0].SearchCount, 2)
	require.GreaterOrEqual(t, report.Users[0].CreateCount, 1)
	require.Equal(t, []string{"Jordan", "Sam"}, report.MonitoringPrincipals)
}

func TestUserActivityMonitoringPullGateIsNotAdminWide(t *testing.T) {
	app := newActivityMonitoringTestApp(t)
	manager := NewDBManager(app.db, SupabaseConfig{})
	manager.canPullActivityMonitoring = app.currentSessionCanAccessActivityMonitoring

	activateActivityMonitoringLicense(t, app, "PH-ADM-4A0185", "admin", "Alex")
	require.False(t, manager.activityMonitoringPullAllowed())

	activateActivityMonitoringLicense(t, app, adminMonitoringLicenseKey, "admin", "Jordan")
	require.True(t, manager.activityMonitoringPullAllowed())
	require.True(t, isUserActivitySyncTable(userActivityTableEvents))
}

func TestUserActivityMonitoringFoundationAddsMissingColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE user_activity_sessions (id TEXT PRIMARY KEY)").Error)
	require.NoError(t, db.Exec("CREATE TABLE user_activity_events (id TEXT PRIMARY KEY)").Error)
	require.NoError(t, db.Exec("CREATE TABLE user_activity_weekly_summaries (id TEXT PRIMARY KEY)").Error)

	app := &App{db: db}
	require.NoError(t, app.ensureUserActivityMonitoringFoundationInternal())
	require.True(t, db.Migrator().HasColumn(&UserActivitySession{}, "meaningful_seconds"))
	require.True(t, db.Migrator().HasColumn(&UserActivityEvent{}, "search_hash"))
	require.True(t, db.Migrator().HasColumn(&UserActivityWeeklySummary{}, "efficiency_score"))
}

func TestUserActivityMonitoringFullDownloadSkipsRestrictedTables(t *testing.T) {
	local, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	remote, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, local.AutoMigrate(&UserActivitySession{}))
	require.NoError(t, remote.AutoMigrate(&UserActivitySession{}))
	require.NoError(t, remote.Create(&UserActivitySession{
		Base:              Base{ID: "remote-activity-session"},
		SessionID:         "remote-session",
		EmployeeName:      "Jamie",
		LicenseRole:       "sales",
		StartedAt:         time.Now().Add(-time.Hour),
		LastSeenAt:        time.Now(),
		ActiveSeconds:     3600,
		MeaningfulSeconds: 2700,
		EventCount:        12,
	}).Error)

	manager := &DBManager{
		local:                     local,
		remote:                    remote,
		canPullActivityMonitoring: func() bool { return false },
	}
	downloaded, err := manager.DownloadFullDatabase()
	require.NoError(t, err)
	require.Equal(t, 0, downloaded)

	var localCount int64
	require.NoError(t, local.Model(&UserActivitySession{}).Count(&localCount).Error)
	require.EqualValues(t, 0, localCount)

	manager.canPullActivityMonitoring = func() bool { return true }
	downloaded, err = manager.DownloadFullDatabase()
	require.NoError(t, err)
	require.Equal(t, 1, downloaded)
	require.NoError(t, local.Model(&UserActivitySession{}).Count(&localCount).Error)
	require.EqualValues(t, 1, localCount)
}

func TestUserActivityMonitoringRoleOutputsAreRahulOnlyReadable(t *testing.T) {
	app := newActivityMonitoringTestApp(t)

	roles := []struct {
		key         string
		role        string
		displayName string
		screen      string
		action      string
	}{
		{"PH-SLS-ROLE01", "sales", "Jamie", "opportunities", "Create costing sheet"},
		{"PH-OPS-ROLE01", "operations", "Riley", "delivery-notes", "Create delivery note"},
		{"PH-MGR-ROLE01", "manager", "Casey", "invoices", "Edit invoice"},
		{"PH-STF-ROLE01", "staff", "Staff User", "dashboard", "Search dashboard"},
	}

	for i, role := range roles {
		activateActivityMonitoringLicense(t, app, role.key, role.role, role.displayName)
		session, err := app.StartUserActivitySession("role-output-test")
		require.NoError(t, err)
		require.NoError(t, app.RecordUserActivityBatch([]UserActivityEventInput{{
			SessionID:         session.SessionID,
			EventType:         "click",
			Category:          "update",
			Screen:            role.screen,
			ActionLabel:       role.action,
			ActiveSeconds:     90 + i,
			MeaningfulSeconds: 60 + i,
		}}))
		require.NoError(t, app.RecordUserActivityHeartbeat(UserActivityHeartbeatInput{
			SessionID:         session.SessionID,
			Screen:            role.screen,
			ActiveSeconds:     300,
			MeaningfulSeconds: 240,
			EventCount:        3,
			UpdateCount:       1,
		}))
	}

	activateActivityMonitoringLicense(t, app, "PH-ADM-NOTRAH", "admin", "Alex")
	require.False(t, app.CanViewUserActivityMonitoring())
	_, err := app.GetWeeklyUserActivityReport("")
	require.Error(t, err)

	activateActivityMonitoringLicense(t, app, adminMonitoringLicenseKey, "admin", "Jordan")
	report, err := app.GetWeeklyUserActivityReport("")
	require.NoError(t, err)
	require.Len(t, report.Users, len(roles))
	require.Len(t, report.ChartRows, len(roles))
	require.Greater(t, report.TotalActiveHours, 0.0)
	require.Greater(t, report.TotalMeaningfulHours, 0.0)
	require.Greater(t, report.AverageEfficiency, 0.0)

	seen := map[string]bool{}
	for _, user := range report.Users {
		seen[user.LicenseRole] = true
		require.NotEmpty(t, user.TopScreens)
		require.Greater(t, user.UpdateCount, 0)
	}
	require.True(t, seen["sales"])
	require.True(t, seen["operations"])
	require.True(t, seen["manager"])
	require.True(t, seen["staff"])
}
