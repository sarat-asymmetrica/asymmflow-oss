package main

// Wave 8 P3 (Bucket E): employee archive request→archive flow.
// Covers the admin overlay, input validation, the self-archive guard, the
// cascading archive (employee + access links + project memberships), and the
// review approve/reject paths with requester notification.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// archiveTestModels migrates every table the archive flow touches.
func archiveTestModels(t *testing.T, app *App) {
	t.Helper()
	require.NoError(t, app.db.AutoMigrate(
		&Employee{},
		&EmployeeAccessLink{},
		&EmployeeArchiveRequest{},
		&Project{},
		&ProjectMember{},
		&Notification{},
		&NotificationReceipt{},
		&CollaborativePendingOperation{},
	))
}

// seedActingAdmin wires currentUserID → active access link → employee so
// GetCurrentEmployeeContext resolves the acting admin.
func seedActingAdmin(t *testing.T, app *App) {
	t.Helper()
	require.NoError(t, app.db.Create(&Employee{
		Base: Base{ID: "admin-emp"}, EmployeeCode: "ADM", FullName: "Ada Admin",
		IsActive: true, EmploymentStatus: "active",
	}).Error)
	require.NoError(t, app.db.Create(&EmployeeAccessLink{
		Base: Base{ID: "link-admin"}, EmployeeID: "admin-emp", UserID: "test-user",
		AccessStatus: "active", IsPrimary: true,
	}).Error)
}

func TestRequestEmployeeArchive_RejectsNonAdmin(t *testing.T) {
	app := setupTestApp(t)
	archiveTestModels(t, app)

	// Holds hr:update but is NOT admin — isolates the admin overlay.
	app.currentUser = &User{
		Base:     Base{ID: "hr-user"},
		Username: "hr-clerk",
		RoleName: "hr",
		Role: Role{
			Name:        "hr",
			DisplayName: "HR",
			Permissions: `["hr:update","hr:view"]`,
		},
	}
	app.currentUserID = "hr-user"

	_, err := app.RequestEmployeeArchive("victim-emp", "left the company")
	require.Error(t, err)
	require.Contains(t, err.Error(), "only admin can archive employees")

	var count int64
	require.NoError(t, app.db.Model(&EmployeeArchiveRequest{}).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestRequestEmployeeArchive_ValidatesInput(t *testing.T) {
	app := setupTestApp(t) // admin by default
	archiveTestModels(t, app)

	_, err := app.RequestEmployeeArchive("   ", "reason")
	require.ErrorContains(t, err, "employee id is required")

	_, err = app.RequestEmployeeArchive("victim-emp", "   ")
	require.ErrorContains(t, err, "archive reason is required")
}

func TestRequestEmployeeArchive_RejectsSelfArchive(t *testing.T) {
	app := setupTestApp(t)
	archiveTestModels(t, app)
	seedActingAdmin(t, app)

	// Target == acting admin's own employee id.
	_, err := app.RequestEmployeeArchive("admin-emp", "cannot archive self")
	require.ErrorContains(t, err, "cannot archive their own")
}

// Wave 9.7 B7(d): RequestEmployeeArchive is now a two-step, review-gated action.
// A request creates a PENDING queue item and archives nothing; approving it from
// the approvals queue performs the archive + full cascade.
func TestRequestEmployeeArchive_CreatesPendingThenReviewArchivesAndCascades(t *testing.T) {
	app := setupTestApp(t)
	archiveTestModels(t, app)
	seedActingAdmin(t, app)

	// Victim with an active access link and an active project membership.
	require.NoError(t, app.db.Create(&Employee{
		Base: Base{ID: "victim-emp"}, EmployeeCode: "VIC", FullName: "Vic Tim",
		IsActive: true, EmploymentStatus: "active",
	}).Error)
	require.NoError(t, app.db.Create(&EmployeeAccessLink{
		Base: Base{ID: "link-victim"}, EmployeeID: "victim-emp", UserID: "victim-user",
		AccessStatus: "active", IsPrimary: true,
	}).Error)
	require.NoError(t, app.db.Create(&Project{Base: Base{ID: "proj-1"}, Name: "Retrofit", Status: "active"}).Error)
	require.NoError(t, app.db.Create(&ProjectMember{
		Base: Base{ID: "pm-1"}, ProjectID: "proj-1", EmployeeID: "victim-emp", IsActive: true,
	}).Error)

	// Step 1: request creates a PENDING review item and archives nothing.
	req, err := app.RequestEmployeeArchive("victim-emp", "  performance  ")
	require.NoError(t, err)
	require.Equal(t, "pending", req.Status)
	require.Equal(t, "performance", req.Reason) // trimmed
	require.NotEmpty(t, req.ID)

	var stillActive Employee
	require.NoError(t, app.db.First(&stillActive, "id = ?", "victim-emp").Error)
	require.True(t, stillActive.IsActive, "employee must stay active until an admin approves the archive")
	require.Equal(t, "active", stillActive.EmploymentStatus)

	// It surfaces in the approvals queue.
	pending, err := app.ListEmployeeArchiveRequests("pending")
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, req.ID, pending[0].ID)

	// Step 2: approving from the queue performs the archive + full cascade.
	reviewed, err := app.ReviewEmployeeArchiveRequest(req.ID, "approve", "confirmed")
	require.NoError(t, err)
	require.Equal(t, "approved", reviewed.Status)
	require.Equal(t, "admin-emp", reviewed.SecondApprovedBy)

	// Employee archived with full metadata.
	var victim Employee
	require.NoError(t, app.db.First(&victim, "id = ?", "victim-emp").Error)
	require.False(t, victim.IsActive)
	require.Equal(t, "archived", victim.EmploymentStatus)
	require.Equal(t, "admin-emp", victim.ArchivedBy)
	require.Equal(t, "performance", victim.ArchiveReason)
	require.Equal(t, req.ID, victim.ArchiveRequestID)
	require.NotNil(t, victim.ArchivedAt)
	require.NotNil(t, victim.EndDate)

	// Access link archived + demoted.
	var link EmployeeAccessLink
	require.NoError(t, app.db.First(&link, "id = ?", "link-victim").Error)
	require.Equal(t, "archived", link.AccessStatus)
	require.False(t, link.IsPrimary)

	// Project membership closed.
	var member ProjectMember
	require.NoError(t, app.db.First(&member, "id = ?", "pm-1").Error)
	require.False(t, member.IsActive)
	require.NotNil(t, member.LeftAt)

	// Idempotency: a second request on an already-archived employee is rejected.
	_, err = app.RequestEmployeeArchive("victim-emp", "again")
	require.ErrorContains(t, err, "already archived")
}

func TestReviewEmployeeArchiveRequest_ApproveArchives(t *testing.T) {
	app := setupTestApp(t)
	archiveTestModels(t, app)
	seedActingAdmin(t, app)

	require.NoError(t, app.db.Create(&Employee{
		Base: Base{ID: "victim-emp"}, EmployeeCode: "VIC", FullName: "Vic Tim",
		IsActive: true, EmploymentStatus: "active",
	}).Error)
	// A pending request (as if synced from a peer node).
	require.NoError(t, app.db.Create(&EmployeeArchiveRequest{
		Base: Base{ID: "req-1"}, EmployeeID: "victim-emp", EmployeeName: "Vic Tim",
		RequestedBy: "requester-emp", RequestedByName: "Reg Requester",
		Reason: "restructuring", Status: "pending", RequiredApprovals: 1,
	}).Error)

	req, err := app.ReviewEmployeeArchiveRequest("req-1", "approve", "confirmed")
	require.NoError(t, err)
	require.Equal(t, "approved", req.Status)
	require.Equal(t, "admin-emp", req.SecondApprovedBy)
	require.Equal(t, "confirmed", req.ReviewNotes)

	var victim Employee
	require.NoError(t, app.db.First(&victim, "id = ?", "victim-emp").Error)
	require.False(t, victim.IsActive)
	require.Equal(t, "archived", victim.EmploymentStatus)

	// Requester notified of the outcome.
	var notes []Notification
	require.NoError(t, app.db.Where("employee_id = ? AND source_type = ?", "requester-emp", "employee_archive_approval").Find(&notes).Error)
	require.Len(t, notes, 1)
	require.Contains(t, notes[0].Message, "approved")
}

func TestReviewEmployeeArchiveRequest_RejectDoesNotArchive(t *testing.T) {
	app := setupTestApp(t)
	archiveTestModels(t, app)
	seedActingAdmin(t, app)

	require.NoError(t, app.db.Create(&Employee{
		Base: Base{ID: "victim-emp"}, EmployeeCode: "VIC", FullName: "Vic Tim",
		IsActive: true, EmploymentStatus: "active",
	}).Error)
	require.NoError(t, app.db.Create(&EmployeeArchiveRequest{
		Base: Base{ID: "req-1"}, EmployeeID: "victim-emp", EmployeeName: "Vic Tim",
		RequestedBy: "requester-emp", Status: "pending", RequiredApprovals: 1,
	}).Error)

	req, err := app.ReviewEmployeeArchiveRequest("req-1", "reject", "insufficient cause")
	require.NoError(t, err)
	require.Equal(t, "rejected", req.Status)
	require.Equal(t, "admin-emp", req.RejectedBy)
	require.NotNil(t, req.RejectedAt)

	// Employee stays active.
	var victim Employee
	require.NoError(t, app.db.First(&victim, "id = ?", "victim-emp").Error)
	require.True(t, victim.IsActive)
	require.Equal(t, "active", victim.EmploymentStatus)

	// A second review on a now-terminal request is refused.
	_, err = app.ReviewEmployeeArchiveRequest("req-1", "approve", "")
	require.ErrorContains(t, err, "already rejected")
}

func TestReviewEmployeeArchiveRequest_ValidatesDecision(t *testing.T) {
	app := setupTestApp(t)
	archiveTestModels(t, app)
	seedActingAdmin(t, app)

	_, err := app.ReviewEmployeeArchiveRequest("req-1", "maybe", "")
	require.ErrorContains(t, err, "decision must be approve or reject")

	_, err = app.ReviewEmployeeArchiveRequest("   ", "approve", "")
	require.ErrorContains(t, err, "archive request id is required")
}
