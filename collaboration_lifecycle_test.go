package main

// Wave 8 P4 slice 7 (Bucket G): project lifecycle tail — whitelisted
// UpdateCollaborativeProject plus Archive/Shelve/Delete status wrappers,
// with terminal statuses escalating to projects:delete (admin wildcard only).

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateCollaborativeProject_WhitelistTrimsAndFilters(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Project{}))
	require.NoError(t, app.db.Create(&Project{Base: Base{ID: "p1", CreatedBy: "owner"}, Name: "Old", Status: "active"}).Error)

	got, err := app.UpdateCollaborativeProject("p1", map[string]any{
		"name":        "  New Name  ",
		"description": " retrofit scope ",
		"created_by":  "attacker", // not whitelisted — must be ignored
	})
	require.NoError(t, err)
	require.Equal(t, "New Name", got.Name)
	require.Equal(t, "retrofit scope", got.Description)

	var reloaded Project
	require.NoError(t, app.db.First(&reloaded, "id = ?", "p1").Error)
	require.Equal(t, "owner", reloaded.CreatedBy, "non-whitelisted column leaked through")

	// Updates containing only unsupported keys are rejected.
	_, err = app.UpdateCollaborativeProject("p1", map[string]any{"created_by": "x"})
	require.ErrorContains(t, err, "no supported project updates")

	// Blank name is rejected.
	_, err = app.UpdateCollaborativeProject("p1", map[string]any{"name": "   "})
	require.ErrorContains(t, err, "project name is required")

	// Unknown project id.
	_, err = app.UpdateCollaborativeProject("missing", map[string]any{"name": "X"})
	require.ErrorContains(t, err, "project not found")

	// Blank project id.
	_, err = app.UpdateCollaborativeProject("   ", map[string]any{"name": "X"})
	require.ErrorContains(t, err, "project id is required")
}

func TestUpdateCollaborativeProject_TerminalStatusRequiresAdmin(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Project{}))
	require.NoError(t, app.db.Create(&Project{Base: Base{ID: "p1"}, Name: "Live", Status: "active"}).Error)

	// projects:update alone cannot push a project into a terminal status.
	setDataQualityRole(app, "user-1", "manager", `["projects:view","projects:update"]`, "Manager")
	for _, terminal := range []string{"Archived", "shelved", "DELETED"} {
		_, err := app.UpdateCollaborativeProject("p1", map[string]any{"status": terminal})
		require.ErrorContains(t, err, "admin permission", "terminal status %q must escalate", terminal)
	}

	// Non-terminal status stays open to projects:update (and is normalised).
	got, err := app.UpdateCollaborativeProject("p1", map[string]any{"status": "  On Hold "})
	require.NoError(t, err)
	require.Equal(t, "on hold", got.Status)

	// Admin wildcard may archive.
	setDataQualityRole(app, "admin-1", "admin", `["*"]`, "Admin")
	got, err = app.UpdateCollaborativeProject("p1", map[string]any{"status": "ARCHIVED"})
	require.NoError(t, err)
	require.Equal(t, "archived", got.Status)
}

func TestProjectLifecycle_ArchiveShelveDelete(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Project{}))
	rows := []Project{
		{Base: Base{ID: "p1"}, Name: "Wrap", Status: "active"},
		{Base: Base{ID: "p2"}, Name: "Pause", Status: "active"},
		{Base: Base{ID: "p3"}, Name: "Dupe", Status: "active"},
	}
	require.NoError(t, app.db.Create(&rows).Error)

	archived, err := app.ArchiveCollaborativeProject("p1", "wrapped up")
	require.NoError(t, err)
	require.Equal(t, "archived", archived.Status)

	shelved, err := app.ShelveCollaborativeProject("p2", "paused for budget")
	require.NoError(t, err)
	require.Equal(t, "shelved", shelved.Status)

	require.NoError(t, app.DeleteCollaborativeProject("p3", "duplicate record"))
	var p3 Project
	require.NoError(t, app.db.First(&p3, "id = ?", "p3").Error, "delete is a status change — the row must remain")
	require.Equal(t, "deleted", p3.Status)

	// All three vanish from the active list but stay reachable with activeOnly=false.
	active, err := app.ListCollaborativeProjects(true)
	require.NoError(t, err)
	require.Empty(t, active)
	all, err := app.ListCollaborativeProjects(false)
	require.NoError(t, err)
	require.Len(t, all, 3)

	// Blank id rejected.
	require.Error(t, app.DeleteCollaborativeProject("   ", ""))
}
