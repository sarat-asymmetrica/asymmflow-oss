package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func permissionsJSON(permissions []string) string {
	payload, _ := json.Marshal(permissions)
	return string(payload)
}

func setConflictTestUser(app *App, id, roleName, displayName string, permissions []string) {
	app.currentUserID = id
	app.currentUser = &User{
		Base:        Base{ID: id},
		Username:    id,
		FullName:    displayName,
		DisplayName: displayName,
		RoleName:    roleName,
		Role: Role{
			Name:        roleName,
			DisplayName: roleName,
			Permissions: permissionsJSON(permissions),
			IsActive:    true,
		},
		IsActive: true,
	}
}

func seedConflictOpportunity(t *testing.T, app *App, folder string) Opportunity {
	t.Helper()
	opp := Opportunity{
		Base:         Base{ID: uuid.New().String(), Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		FolderNumber: folder,
		CustomerName: "Concurrent Customer",
		Title:        "Concurrent Flowmeter Upgrade",
		Stage:        "Quoted",
		RevenueBHD:   12000,
	}
	require.NoError(t, app.db.Create(&opp).Error)
	return opp
}

func TestOpportunityConflict_StaleStageEditIsFlaggedAndAdminApplies(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureOpportunityConflictFoundation())
	opp := seedConflictOpportunity(t, app, "OPP-CONFLICT-1")

	setConflictTestUser(app, "sales-a", "sales", "Jamie", []string{"offers:view", "offers:edit"})
	updated, err := app.UpdateOpportunityStageWithVersion(opp.ID, "Won", opp.Version)
	require.NoError(t, err)
	require.Equal(t, "Won", updated.Stage)
	require.Equal(t, 2, updated.Version)

	setConflictTestUser(app, "sales-b", "sales", "Riley", []string{"offers:view", "offers:edit"})
	_, err = app.UpdateOpportunityStageWithVersion(opp.ID, "Lost", opp.Version)
	require.Error(t, err)
	require.Contains(t, err.Error(), "conflict")

	var pending []OpportunityEditConflict
	require.NoError(t, app.db.Where("opportunity_id = ? AND status = ?", opp.ID, opportunityConflictStatusPending).Find(&pending).Error)
	require.Len(t, pending, 1)
	require.Equal(t, "stage_update", pending[0].Operation)
	require.Equal(t, 1, pending[0].ExpectedVersion)
	require.Equal(t, 2, pending[0].CurrentVersion)
	require.Contains(t, pending[0].ProposedChangesJSON, "Lost")

	var stillWon Opportunity
	require.NoError(t, app.db.First(&stillWon, "id = ?", opp.ID).Error)
	require.Equal(t, "Won", stillWon.Stage)

	setConflictTestUser(app, "admin", "admin", "Jordan", []string{"*"})
	resolved, err := app.ResolveOpportunityEditConflict(pending[0].ID, "apply", "Accepted later lost update")
	require.NoError(t, err)
	require.Equal(t, opportunityConflictStatusApplied, resolved.Conflict.Status)
	require.Equal(t, "Lost", resolved.Opportunity.Stage)
	require.Equal(t, 3, resolved.Opportunity.Version)
}

func TestOpportunityConflict_StaleDetailsEditPreservesOwnerNotesForSales(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureOpportunityConflictFoundation())
	opp := seedConflictOpportunity(t, app, "OPP-CONFLICT-2")
	opp.OwnerNotes = "management only"
	require.NoError(t, app.db.Save(&opp).Error)

	setConflictTestUser(app, "manager", "manager", "Casey", []string{"offers:view", "offers:edit", "finance:view"})
	updated, err := app.UpdateOpportunityDetailsWithVersion(opp.ID, opp.Version, "manager note", "manager owner note")
	require.NoError(t, err)
	require.Equal(t, 2, updated.Version)

	setConflictTestUser(app, "sales", "sales", "Jamie", []string{"offers:view", "offers:edit"})
	_, err = app.UpdateOpportunityDetailsWithVersion(opp.ID, opp.Version, "sales stale note", "sales should not overwrite owner note")
	require.Error(t, err)
	require.Contains(t, err.Error(), "conflict")

	var conflict OpportunityEditConflict
	require.NoError(t, app.db.Where("opportunity_id = ? AND status = ?", opp.ID, opportunityConflictStatusPending).First(&conflict).Error)
	require.Contains(t, conflict.ProposedChangesJSON, "sales stale note")
	require.NotContains(t, conflict.ProposedChangesJSON, "sales should not overwrite")
	require.Contains(t, conflict.ProposedChangesJSON, "manager owner note")
}

func TestOpportunityConflict_StressSyncsBulkConcurrentActivityAndConflicts(t *testing.T) {
	local := openConflictStressDB(t)
	remote := openConflictStressDB(t)
	require.NoError(t, local.AutoMigrate(&Opportunity{}, &OpportunityEditConflict{}, &UserActivitySession{}, &UserActivityEvent{}, &UserActivityWeeklySummary{}, &SyncRecord{}))
	require.NoError(t, remote.AutoMigrate(&Opportunity{}, &OpportunityEditConflict{}, &UserActivitySession{}, &UserActivityEvent{}, &UserActivityWeeklySummary{}, &SyncRecord{}))

	app := &App{db: local, cache: NewCache(), currentUserID: "stress-user"}
	t.Cleanup(app.cache.Stop)
	setConflictTestUser(app, "stress-user", "sales", "Stress Sales", []string{"offers:view", "offers:edit"})

	const opportunityCount = 1000
	const concurrentOpportunityEdits = 120
	const activitySessionCount = 2400
	for i := 0; i < opportunityCount; i++ {
		opp := Opportunity{
			Base:         Base{ID: uuid.New().String(), Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			FolderNumber: fmt.Sprintf("STRESS-%03d", i),
			CustomerName: fmt.Sprintf("Customer %03d", i),
			Title:        "Stress opportunity",
			Stage:        "Quoted",
			RevenueBHD:   float64(1000 + i),
		}
		require.NoError(t, local.Create(&opp).Error)
	}

	var seeded []Opportunity
	require.NoError(t, local.Limit(concurrentOpportunityEdits).Find(&seeded).Error)
	var wg sync.WaitGroup
	for _, opp := range seeded {
		opp := opp
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = app.UpdateOpportunityStageWithVersion(opp.ID, "Won", opp.Version)
			_, _ = app.UpdateOpportunityStageWithVersion(opp.ID, "Lost", opp.Version)
		}()
	}
	wg.Wait()

	for i := 0; i < activitySessionCount; i++ {
		session := UserActivitySession{
			Base:              Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			SessionID:         fmt.Sprintf("stress-session-%03d", i),
			EmployeeName:      fmt.Sprintf("User %02d", i%12),
			LicenseRole:       []string{"sales", "operations", "manager", "staff"}[i%4],
			StartedAt:         time.Now().Add(-time.Duration(i) * time.Second),
			LastSeenAt:        time.Now(),
			ActiveSeconds:     120,
			MeaningfulSeconds: 90,
			EventCount:        5,
		}
		require.NoError(t, local.Create(&session).Error)
	}

	manager := &DBManager{local: local, remote: remote, canPullActivityMonitoring: func() bool { return true }}
	pushedOpportunities, err := manager.syncTable("opportunities", time.Time{}, "push")
	require.NoError(t, err)
	pushedConflicts, err := manager.syncTable("opportunity_edit_conflicts", time.Time{}, "push")
	require.NoError(t, err)
	pushedActivity, err := manager.syncTable("user_activity_sessions", time.Time{}, "push")
	require.NoError(t, err)
	require.Equal(t, opportunityCount, pushedOpportunities)
	require.GreaterOrEqual(t, pushedConflicts, 1)
	require.Equal(t, activitySessionCount, pushedActivity)

	var remoteOpportunityCount, remoteConflictCount, remoteActivityCount int64
	require.NoError(t, remote.Model(&Opportunity{}).Count(&remoteOpportunityCount).Error)
	require.NoError(t, remote.Model(&OpportunityEditConflict{}).Count(&remoteConflictCount).Error)
	require.NoError(t, remote.Model(&UserActivitySession{}).Count(&remoteActivityCount).Error)
	require.EqualValues(t, opportunityCount, remoteOpportunityCount)
	require.GreaterOrEqual(t, remoteConflictCount, int64(1))
	require.EqualValues(t, activitySessionCount, remoteActivityCount)
}

func openConflictStressDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := memdb.TestDB(t, url.Values{
		"_pragma": {"busy_timeout(5000)"},
	})
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(2)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	return db
}
