package main

// Wave 9.8 B4: employee document-expiry tracking.
// Proves: (1) create encrypts the document number at rest, (2) list/read
// decrypts it back to the original plaintext, (3) RBAC denies a non-admin
// HR session exactly like CreateEmployeeProfile / RequestEmployeeArchive,
// and (4) the expiry scan notifies only documents within the lookahead
// window and is idempotent (NotifiedAt dedupes a second scan).

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func complianceTestModels(t *testing.T, app *App) {
	t.Helper()
	require.NoError(t, app.db.AutoMigrate(
		&Employee{},
		&EmployeeDocument{},
		&Notification{},
		&NotificationReceipt{},
		&CollaborativePendingOperation{},
	))
}

// setupComplianceTestApp wires a real FieldCrypto (the same pattern used by
// comprehensive_e2e_test.go) since setupTestApp does not set one up, and
// CreateEmployeeDocument refuses to store PII without it.
func setupComplianceTestApp(t *testing.T) *App {
	t.Helper()
	app := setupTestApp(t)
	complianceTestModels(t, app)
	fc, err := NewFieldCrypto()
	require.NoError(t, err)
	app.fieldCrypto = fc
	return app
}

func seedComplianceEmployee(t *testing.T, app *App) string {
	t.Helper()
	emp := Employee{Base: Base{ID: "emp-1"}, EmployeeCode: "E1", FullName: "Vic Employee", IsActive: true, EmploymentStatus: "active"}
	require.NoError(t, app.db.Create(&emp).Error)
	return emp.ID
}

func TestCreateEmployeeDocument_EncryptsAtRest(t *testing.T) {
	app := setupComplianceTestApp(t) // default currentUser is admin with ["*"]
	employeeID := seedComplianceEmployee(t, app)

	expires := time.Now().Add(30 * 24 * time.Hour)
	dto, err := app.CreateEmployeeDocument(EmployeeDocument{
		EmployeeID: employeeID,
		DocType:    "visa",
		ExpiresOn:  &expires,
	}, "V-1234567890")
	require.NoError(t, err)
	require.Equal(t, "V-1234567890", dto.DocNumber, "DTO carries the decrypted value")
	require.Equal(t, "••••••••7890", dto.DocNumberMasked)

	// The raw DB column must never contain the plaintext.
	var stored EmployeeDocument
	require.NoError(t, app.db.First(&stored, "id = ?", dto.ID).Error)
	require.NotEqual(t, "V-1234567890", stored.DocNumberEncrypted)
	require.NotContains(t, stored.DocNumberEncrypted, "1234567890")
	require.True(t, app.fieldCrypto.IsEncrypted(stored.DocNumberEncrypted))
}

func TestCreateEmployeeDocument_RoundTripsThroughList(t *testing.T) {
	app := setupComplianceTestApp(t)
	employeeID := seedComplianceEmployee(t, app)

	expires := time.Now().Add(10 * 24 * time.Hour)
	_, err := app.CreateEmployeeDocument(EmployeeDocument{
		EmployeeID: employeeID,
		DocType:    "cpr",
	}, "1234567890123")
	_ = expires
	require.NoError(t, err)

	docs, err := app.ListEmployeeDocuments(employeeID)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	require.Equal(t, "1234567890123", docs[0].DocNumber, "read path must decrypt back to the original plaintext")
}

func TestCreateEmployeeDocument_RejectsNonAdmin(t *testing.T) {
	app := setupComplianceTestApp(t)
	employeeID := seedComplianceEmployee(t, app)

	// Holds hr:create but is NOT admin — same overlay as CreateEmployeeProfile.
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

	_, err := app.CreateEmployeeDocument(EmployeeDocument{
		EmployeeID: employeeID,
		DocType:    "passport",
	}, "P-9999999")
	require.Error(t, err)
	require.Contains(t, err.Error(), "only admin can manage employee compliance documents")

	var count int64
	require.NoError(t, app.db.Model(&EmployeeDocument{}).Count(&count).Error)
	require.Equal(t, int64(0), count, "nothing should have been written")
}

func TestScanExpiringEmployeeDocuments_NotifiesWithinWindowAndIsIdempotent(t *testing.T) {
	app := setupComplianceTestApp(t)
	employeeID := seedComplianceEmployee(t, app)

	withinWindow := time.Now().Add(10 * 24 * time.Hour)
	outsideWindow := time.Now().Add(200 * 24 * time.Hour)

	dueSoon, err := app.CreateEmployeeDocument(EmployeeDocument{
		EmployeeID: employeeID,
		DocType:    "visa",
		ExpiresOn:  &withinWindow,
	}, "VISA-DUE-SOON")
	require.NoError(t, err)

	notDue, err := app.CreateEmployeeDocument(EmployeeDocument{
		EmployeeID: employeeID,
		DocType:    "passport",
		ExpiresOn:  &outsideWindow,
	}, "PASSPORT-NOT-DUE")
	require.NoError(t, err)

	// First scan: only the within-window document is notified.
	result, err := app.ScanExpiringEmployeeDocuments()
	require.NoError(t, err)
	require.Equal(t, 1, result.NotifiedCount)

	var notes []Notification
	require.NoError(t, app.db.Where("source_type = ? AND notification_type = ?", "employee_document", "document_expiry").Find(&notes).Error)
	require.Len(t, notes, 1)
	require.Equal(t, dueSoon.ID, notes[0].SourceID)
	require.Contains(t, notes[0].Message, "VISA") // doc type label, never the raw number

	var notDueRow EmployeeDocument
	require.NoError(t, app.db.First(&notDueRow, "id = ?", notDue.ID).Error)
	require.Nil(t, notDueRow.NotifiedAt, "document outside the lookahead window must not be marked notified")

	var dueSoonRow EmployeeDocument
	require.NoError(t, app.db.First(&dueSoonRow, "id = ?", dueSoon.ID).Error)
	require.NotNil(t, dueSoonRow.NotifiedAt)

	// Second scan is a no-op: NotifiedAt dedupes it, no new notification.
	result2, err := app.ScanExpiringEmployeeDocuments()
	require.NoError(t, err)
	require.Equal(t, 0, result2.NotifiedCount)

	var notesAfter []Notification
	require.NoError(t, app.db.Where("source_type = ? AND notification_type = ?", "employee_document", "document_expiry").Find(&notesAfter).Error)
	require.Len(t, notesAfter, 1, "a second scan must not duplicate the notification")
}

func TestUpdateEmployeeDocument_ExpiryChangeRearmsNotification(t *testing.T) {
	app := setupComplianceTestApp(t)
	employeeID := seedComplianceEmployee(t, app)

	soon := time.Now().Add(5 * 24 * time.Hour)
	created, err := app.CreateEmployeeDocument(EmployeeDocument{
		EmployeeID:    employeeID,
		DocType:       "permit",
		PermitSubtype: "work permit",
		ExpiresOn:     &soon,
	}, "PERMIT-123")
	require.NoError(t, err)

	_, err = app.ScanExpiringEmployeeDocuments()
	require.NoError(t, err)

	var afterFirstScan EmployeeDocument
	require.NoError(t, app.db.First(&afterFirstScan, "id = ?", created.ID).Error)
	require.NotNil(t, afterFirstScan.NotifiedAt)

	renewed := time.Now().Add(400 * 24 * time.Hour)
	_, err = app.UpdateEmployeeDocument(created.ID, EmployeeDocument{ExpiresOn: &renewed}, "")
	require.NoError(t, err)

	var afterRenewal EmployeeDocument
	require.NoError(t, app.db.First(&afterRenewal, "id = ?", created.ID).Error)
	require.Nil(t, afterRenewal.NotifiedAt, "renewing the expiry date must re-arm the notification")
}

func TestDeleteEmployeeDocument_SoftDeletes(t *testing.T) {
	app := setupComplianceTestApp(t)
	employeeID := seedComplianceEmployee(t, app)

	created, err := app.CreateEmployeeDocument(EmployeeDocument{
		EmployeeID: employeeID,
		DocType:    "cpr",
	}, "CPR-000111")
	require.NoError(t, err)

	require.NoError(t, app.DeleteEmployeeDocument(created.ID))

	docs, err := app.ListEmployeeDocuments(employeeID)
	require.NoError(t, err)
	require.Len(t, docs, 0)

	var count int64
	require.NoError(t, app.db.Unscoped().Model(&EmployeeDocument{}).Where("id = ?", created.ID).Count(&count).Error)
	require.Equal(t, int64(1), count, "soft delete must leave the row in place, just marked deleted")
}
