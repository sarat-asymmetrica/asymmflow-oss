package main

// Wave 8 P2-8: CreateEmployeeProfile must carry PH's admin-only overlay ON TOP
// of hr:create. A caller who holds hr:create but is NOT admin must be rejected.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateEmployeeProfile_RejectsNonAdmin(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Employee{}))

	// Non-admin who nonetheless holds hr:create — isolates the admin overlay
	// (requirePermission passes; the admin check is the sole rejecter).
	app.currentUser = &User{
		Base:     Base{ID: "hr-user"},
		Username: "hr-clerk",
		RoleName: "hr",
		Role: Role{
			Name:        "hr",
			DisplayName: "HR",
			Permissions: `["hr:create","hr:view"]`,
		},
	}
	app.currentUserID = "hr-user"

	_, err := app.CreateEmployeeProfile(Employee{FullName: "Blocked Hire"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "only admin can add employee profiles")

	// Confirm nothing was written.
	var count int64
	require.NoError(t, app.db.Model(&Employee{}).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestCreateEmployeeProfile_AllowsAdmin(t *testing.T) {
	app := setupTestApp(t) // default currentUser is admin with ["*"]
	require.NoError(t, app.db.AutoMigrate(&Employee{}))

	created, err := app.CreateEmployeeProfile(Employee{FullName: "  Ada Admin  "})
	require.NoError(t, err)
	require.Equal(t, "Ada Admin", created.FullName) // trimmed
	require.NotEmpty(t, created.EmployeeCode)       // auto-generated
	require.Equal(t, "active", created.EmploymentStatus)
	require.True(t, created.IsActive)
}
