package main

// Wave 8 P3 slice 3: customer data-quality review ledger (Bucket F).
// Ported from ph_holdings/user_feedback_hardening_service_test.go:226-335.
// Covers: the live issue scan flags all four issue types; review is admin-only;
// a resolved issue is suppressed from the queue while a reviewed (non-terminal)
// issue stays visible carrying its review context; history returns dispositions;
// and the customers:view gate on the read endpoints.
//
// Note vs PH: PH also has TestEnsureDataQualityReviewFoundationIsIdempotent,
// which is intentionally NOT ported — the self-migration it exercised was
// dropped (the model is registered in tradingModels() and the golden test
// TestTradingModels_SchemaGolden now covers the table's shape).

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func dataQualityTestModels(t *testing.T, app *App) {
	t.Helper()
	require.NoError(t, app.db.AutoMigrate(
		&CustomerMaster{},
		&Opportunity{},
		&Offer{},
		&DataQualityReview{},
	))
}

// setDataQualityRole swaps the in-memory session identity so tests can move
// between admin and non-admin roles. Mirrors the PH helper; no license row is
// seeded, so currentSessionHasAdminRoleOnly resolves purely from this role.
func setDataQualityRole(app *App, id, roleName, permissions, displayName string) {
	app.currentUserID = id
	app.currentUser = &User{
		Base:        Base{ID: id},
		Username:    id,
		DisplayName: displayName,
		RoleName:    roleName,
		Role: Role{
			Name:        roleName,
			DisplayName: roleName,
			Permissions: permissions,
		},
	}
}

// issueTypeSet collapses a preview result to the set of issue types it carries.
func issueTypeSet(issues []DataQualityIssue) map[string]bool {
	set := make(map[string]bool, len(issues))
	for _, issue := range issues {
		set[issue.IssueType] = true
	}
	return set
}

func TestPreviewCustomerDataQuality_FlagsAllIssueTypes(t *testing.T) {
	app := setupTestApp(t)
	dataQualityTestModels(t, app)

	// Blank-name customer → blank_customer_name.
	require.NoError(t, app.db.Create(&CustomerMaster{CustomerID: "C-0", CustomerCode: "C-0", BusinessName: ""}).Error)
	// Two name variations of the same entity → duplicate_customer.
	require.NoError(t, app.db.Create(&CustomerMaster{CustomerID: "C-1", CustomerCode: "C-1", BusinessName: "Acme Instrumentation W.L.L"}).Error)
	require.NoError(t, app.db.Create(&CustomerMaster{CustomerID: "C-2", CustomerCode: "C-2", BusinessName: "ACME INSTRUMENTATION WLL"}).Error)
	// Opportunity with no title/folder-name and no customer link → both
	// blank_opportunity_name and missing_customer_link.
	require.NoError(t, app.db.Create(&Opportunity{FolderNumber: "T-30", Stage: "RFQ Received"}).Error)
	// Offer with no customer → offer_missing_customer.
	require.NoError(t, app.db.Create(&Offer{OfferNumber: "29-26", CustomerID: "", CustomerName: "", Stage: "Quoted"}).Error)

	issues, err := app.PreviewCustomerDataQuality(50)
	require.NoError(t, err)

	types := issueTypeSet(issues)
	require.True(t, types["blank_customer_name"], "expected blank_customer_name")
	require.True(t, types["duplicate_customer"], "expected duplicate_customer")
	require.True(t, types["blank_opportunity_name"], "expected blank_opportunity_name")
	require.True(t, types["missing_customer_link"], "expected missing_customer_link")
	require.True(t, types["offer_missing_customer"], "expected offer_missing_customer")
}

func TestReviewDataQualityIssue_AdminOnlyAndSuppressesResolved(t *testing.T) {
	app := setupTestApp(t)
	dataQualityTestModels(t, app)

	require.NoError(t, app.db.Create(&CustomerMaster{CustomerID: "C-1", CustomerCode: "C-1", BusinessName: "Acme Instrumentation W.L.L"}).Error)
	require.NoError(t, app.db.Create(&CustomerMaster{CustomerID: "C-2", CustomerCode: "C-2", BusinessName: "ACME INSTRUMENTATION WLL"}).Error)

	// A manager holds customers:view (so preview works) but is not admin.
	setDataQualityRole(app, "manager-dq", "manager", `["customers:view","customers:edit"]`, "Manager")
	issues, err := app.PreviewCustomerDataQuality(20)
	require.NoError(t, err)
	require.NotEmpty(t, issues)

	target := issues[0]
	require.Equal(t, "duplicate_customer", target.IssueType)

	// Non-admin cannot disposition an issue.
	_, err = app.ReviewDataQualityIssue(target, "resolved", "manager should not be able to clear this")
	require.Error(t, err)
	require.Contains(t, err.Error(), "admin")

	// Admin resolves it.
	setDataQualityRole(app, "admin-dq", "admin", `["*"]`, "Admin Reviewer")
	review, err := app.ReviewDataQualityIssue(target, "resolved", "confirmed duplicate was already merged")
	require.NoError(t, err)
	require.Equal(t, "resolved", review.Status)
	require.Equal(t, target.ID, review.IssueID)
	require.Equal(t, "admin-dq", review.ReviewedByID)
	require.Equal(t, "Admin Reviewer", review.ReviewedBy)
	require.NotNil(t, review.ReviewedAt)

	// Resolved issue is suppressed from the next preview.
	refreshed, err := app.PreviewCustomerDataQuality(20)
	require.NoError(t, err)
	for _, issue := range refreshed {
		require.NotEqual(t, target.ID, issue.ID, "resolved issue should not reappear")
	}

	// History carries the disposition.
	history, err := app.GetDataQualityReviewHistory(10)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, target.ID, history[0].IssueID)
	require.Equal(t, "resolved", history[0].Status)
}

func TestReviewedDataQualityIssue_StaysVisibleWithContext(t *testing.T) {
	app := setupTestApp(t)
	dataQualityTestModels(t, app)

	require.NoError(t, app.db.Create(&Opportunity{Base: Base{ID: "opp-dq-review"}, FolderNumber: "T-31"}).Error)
	setDataQualityRole(app, "admin-dq-review", "admin", `["*"]`, "Admin Reviewer")

	issues, err := app.PreviewCustomerDataQuality(20)
	require.NoError(t, err)
	var target DataQualityIssue
	for _, issue := range issues {
		if issue.IssueType == "blank_opportunity_name" {
			target = issue
			break
		}
	}
	require.NotEmpty(t, target.ID, "expected a blank_opportunity_name issue")

	// "reviewed" is non-terminal — the issue stays in the queue, annotated.
	_, err = app.ReviewDataQualityIssue(target, "reviewed", "assigned to sales ops")
	require.NoError(t, err)

	refreshed, err := app.PreviewCustomerDataQuality(20)
	require.NoError(t, err)
	var reviewed DataQualityIssue
	found := false
	for _, issue := range refreshed {
		if issue.ID == target.ID {
			reviewed = issue
			found = true
			break
		}
	}
	require.True(t, found, "reviewed (non-terminal) issue should stay visible")
	require.Equal(t, "reviewed", reviewed.ReviewStatus)
	require.Equal(t, "assigned to sales ops", reviewed.ReviewNote)
	require.Equal(t, "Admin Reviewer", reviewed.ReviewedBy)
}

func TestDataQualityReview_RejectsUnknownAction(t *testing.T) {
	app := setupTestApp(t)
	dataQualityTestModels(t, app)

	setDataQualityRole(app, "admin-dq", "admin", `["*"]`, "Admin")
	_, err := app.ReviewDataQualityIssue(DataQualityIssue{ID: "some-issue"}, "banish", "")
	require.ErrorContains(t, err, "unsupported")

	// Empty issue id is rejected before any action check.
	_, err = app.ReviewDataQualityIssue(DataQualityIssue{ID: "   "}, "resolved", "")
	require.ErrorContains(t, err, "issue id is required")
}

func TestDataQualityReads_PermissionGate(t *testing.T) {
	app := setupTestApp(t)
	dataQualityTestModels(t, app)

	// A role without customers:view cannot read the queue or history.
	app.currentUser.Role.Permissions = `["dashboard:view"]`

	_, err := app.PreviewCustomerDataQuality(20)
	require.Error(t, err)

	_, err = app.GetDataQualityReviewHistory(20)
	require.Error(t, err)
}
