package main

// Wave 8 P2-4/5/6/7: restore collaboration business rules the refactor dropped —
// project linkage/POC fields, member-add notifications, employee opportunity
// rollup, and the terminal-status project filter.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// P2-4
func TestCreateCollaborativeProject_PersistsLinkageFields(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Project{}))

	in := Project{
		Name:             "  Refinery Retrofit  ",
		CustomerName:     "  Acme Instrumentation ",
		EndUserName:      " Gulf Refining ",
		OpportunityKey:   " OPP-2026-014 ",
		CustomerPOCName:  " Jane Roe ",
		CustomerPOCEmail: " jane@acme.test ",
		CustomerPOCPhone: " +973-1234 ",
	}
	got, err := app.CreateCollaborativeProject(in)
	require.NoError(t, err)
	require.Equal(t, "Acme Instrumentation", got.CustomerName)
	require.Equal(t, "OPP-2026-014", got.OpportunityKey)
	require.Equal(t, "jane@acme.test", got.CustomerPOCEmail)

	var reloaded Project
	require.NoError(t, app.db.First(&reloaded, "id = ?", got.ID).Error)
	require.Equal(t, "Gulf Refining", reloaded.EndUserName)
	require.Equal(t, "+973-1234", reloaded.CustomerPOCPhone)
	require.Equal(t, "Jane Roe", reloaded.CustomerPOCName)
}

// P2-5
func TestAddCollaborativeProjectMember_EmitsNotification(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Project{}, &ProjectMember{}, &Employee{}, &Notification{}, &NotificationReceipt{}))

	require.NoError(t, app.db.Create(&Project{Base: Base{ID: "proj-1"}, Name: "Retrofit", Status: "active"}).Error)
	require.NoError(t, app.db.Create(&Employee{Base: Base{ID: "emp-1"}, FullName: "Jane Roe", EmployeeCode: "E001"}).Error)

	_, err := app.AddCollaborativeProjectMember("proj-1", "emp-1", "Engineer", 100)
	require.NoError(t, err)

	var notes []Notification
	require.NoError(t, app.db.Where("employee_id = ? AND source_type = ?", "emp-1", "project").Find(&notes).Error)
	require.Len(t, notes, 1)
	require.Equal(t, "project", notes[0].NotificationType)
	require.Equal(t, "proj-1", notes[0].SourceID)
	require.Equal(t, "unread", notes[0].Status)
	require.Contains(t, notes[0].Message, "Engineer")

	// Re-add (update path) must notify again.
	_, err = app.AddCollaborativeProjectMember("proj-1", "emp-1", "Lead", 100)
	require.NoError(t, err)
	var count int64
	require.NoError(t, app.db.Model(&Notification{}).Where("employee_id = ?", "emp-1").Count(&count).Error)
	require.Equal(t, int64(2), count)
}

// P2-6
func TestListEmployeeContributionSummaries_OpportunityRollup(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Employee{}, &ProjectMember{}, &TaskItem{}, &EmployeeAccessLink{}, &Opportunity{}))

	require.NoError(t, app.db.Create(&Employee{
		Base: Base{ID: "emp-1"}, EmployeeCode: "E001",
		FullName: "Jane Roe", IsActive: true, EmploymentStatus: "active",
	}).Error)

	inYear := time.Date(time.Now().Year(), time.March, 1, 0, 0, 0, 0, time.Local)
	mkOpp := func(id, stage string, rev float64) Opportunity {
		return Opportunity{Base: Base{ID: id}, FolderNumber: "FLD-" + id, Salesperson: "Jane Roe", Stage: stage, RevenueBHD: rev, OfferDate: inYear}
	}
	require.NoError(t, app.db.Create(&[]Opportunity{
		mkOpp("o1", "Won", 1200.500),
		mkOpp("o2", "Lost", 0),
		mkOpp("o3", "Proposal", 0),
	}).Error)

	// Out-of-year opportunity: pin BOTH offer_date and created_at to last year,
	// else GORM's auto-set created_at (now) would pull it into the YTD window.
	lastYear := inYear.AddDate(-1, 0, 0)
	old := mkOpp("o4", "Won", 999)
	old.OfferDate = lastYear
	old.CreatedAt = lastYear
	require.NoError(t, app.db.Create(&old).Error)

	sums, err := app.ListEmployeeContributionSummaries()
	require.NoError(t, err)
	require.Len(t, sums, 1)
	s := sums[0]
	require.Equal(t, 3, s.OpportunityYTD)
	require.Equal(t, 1, s.OpportunityWonYTD)
	require.Equal(t, 1, s.OpportunityLostYTD)
	require.Equal(t, 1200.5, s.RevenueYTD)
}

// P2-7
func TestListCollaborativeProjects_ActiveOnlyFilter(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Project{}))

	rows := []Project{
		{Base: Base{ID: "p1"}, Name: "Active", Status: "active"},
		{Base: Base{ID: "p2"}, Name: "Archived", Status: "archived"},
		{Base: Base{ID: "p3"}, Name: "Shelved", Status: "shelved"},
		{Base: Base{ID: "p4"}, Name: "Deleted", Status: "deleted"},
		{Base: Base{ID: "p5"}, Name: "MixedCase", Status: "Archived"},
		{Base: Base{ID: "p6"}, Name: "NullStatus", Status: ""},
	}
	require.NoError(t, app.db.Create(&rows).Error)

	all, err := app.ListCollaborativeProjects(false)
	require.NoError(t, err)
	require.Len(t, all, 6)

	active, err := app.ListCollaborativeProjects(true)
	require.NoError(t, err)
	got := map[string]bool{}
	for _, p := range active {
		got[p.ID] = true
	}
	// active + empty-status kept; archived (any case)/shelved/deleted excluded.
	require.True(t, got["p1"], "active row wrongly excluded")
	require.True(t, got["p6"], "empty-status row wrongly excluded")
	require.False(t, got["p2"] || got["p3"] || got["p4"] || got["p5"], "terminal-status rows leaked into active set")
	require.Len(t, active, 2)
}
