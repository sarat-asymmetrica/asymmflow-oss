package main

// Wave 9.8 B3: allocation capacity WARN (not block). GetEmployeeAllocationSummary
// is the read-only precheck the frontend calls before saving a project member's
// allocation — it must compute the OTHER-active-projects total server-side so
// the client never has to (and never gets to) do that math itself.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetEmployeeAllocationSummary_OverAllocationAcrossProjects(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Project{}, &ProjectMember{}, &Employee{}))

	require.NoError(t, app.db.Create(&Employee{Base: Base{ID: "emp-1"}, FullName: "Jane Roe", EmployeeCode: "E001"}).Error)

	require.NoError(t, app.db.Create(&Project{Base: Base{ID: "proj-a"}, Name: "Refinery Retrofit", Status: "active"}).Error)
	require.NoError(t, app.db.Create(&Project{Base: Base{ID: "proj-b"}, Name: "Pipeline Survey", Status: "active"}).Error)
	require.NoError(t, app.db.Create(&Project{Base: Base{ID: "proj-c"}, Name: "Archived Retrofit", Status: "archived"}).Error)
	require.NoError(t, app.db.Create(&Project{Base: Base{ID: "proj-d"}, Name: "Former Gig", Status: "active"}).Error)

	require.NoError(t, app.db.Create(&ProjectMember{
		Base: Base{ID: "pm-a"}, ProjectID: "proj-a", EmployeeID: "emp-1",
		Role: "Engineer", AllocationPercent: 70, IsActive: true,
	}).Error)
	require.NoError(t, app.db.Create(&ProjectMember{
		Base: Base{ID: "pm-b"}, ProjectID: "proj-b", EmployeeID: "emp-1",
		Role: "Engineer", AllocationPercent: 50, IsActive: true,
	}).Error)
	// Inactive membership must be excluded from the total (different project
	// than proj-b to avoid the (project_id, employee_id) unique index).
	// Created active then flipped off via an explicit column Update — GORM's
	// `default:true` tag on IsActive otherwise silently re-applies the
	// default on a Create with the false zero-value.
	require.NoError(t, app.db.Create(&ProjectMember{
		Base: Base{ID: "pm-inactive"}, ProjectID: "proj-d", EmployeeID: "emp-1",
		Role: "Former", AllocationPercent: 999, IsActive: true,
	}).Error)
	require.NoError(t, app.db.Model(&ProjectMember{}).Where("id = ?", "pm-inactive").Update("is_active", false).Error)
	// Membership on an archived (non-active) project must be excluded.
	require.NoError(t, app.db.Create(&ProjectMember{
		Base: Base{ID: "pm-c"}, ProjectID: "proj-c", EmployeeID: "emp-1",
		Role: "Engineer", AllocationPercent: 100, IsActive: true,
	}).Error)

	// Excluding proj-a (the membership being edited): only proj-b's 50%
	// active/active membership should count.
	summary, err := app.GetEmployeeAllocationSummary("emp-1", "proj-a")
	require.NoError(t, err)
	require.Equal(t, "emp-1", summary.EmployeeID)
	require.Equal(t, 50.0, summary.OtherProjectsTotal)
	require.Len(t, summary.Projects, 1)
	require.Equal(t, "proj-b", summary.Projects[0].ProjectID)
	require.Equal(t, "Pipeline Survey", summary.Projects[0].ProjectName)
	require.Equal(t, 50.0, summary.Projects[0].AllocationPercent)

	// (OtherProjectsTotal + newAllocation) > 100 is the WARN condition the
	// frontend evaluates: 50 + 60 = 110 > 100.
	require.Greater(t, summary.OtherProjectsTotal+60, 100.0)

	// Without an exclusion, proj-a's 70% also counts: 70 + 50 = 120.
	fullSummary, err := app.GetEmployeeAllocationSummary("emp-1", "")
	require.NoError(t, err)
	require.Equal(t, 120.0, fullSummary.OtherProjectsTotal)
	require.Len(t, fullSummary.Projects, 2)

	// Blank employee id rejected.
	_, err = app.GetEmployeeAllocationSummary("   ", "proj-a")
	require.ErrorContains(t, err, "employee id is required")
}
