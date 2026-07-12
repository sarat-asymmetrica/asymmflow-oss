package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =============================================================================
// MISSION I — OFFER LIFECYCLE PORTS (I-19 / I-20 / I-21)
// =============================================================================
// Covers CreateOfferRevision / RenewOffer lineage, the whitelisted
// UpdateOpportunityCommercialFields mutator, and GetUnifiedOfferThread /
// GetCostingsByOpportunity assembly — matching deployed PH semantics.
// =============================================================================

func setupOfferLifecycleTestApp(t *testing.T) *App {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&Offer{},
		&OfferItem{},
		&OfferNote{},
		&OfferFollowUp{},
		&Opportunity{},
		&OpportunityComment{},
		&Order{},
		&Invoice{},
		&RFQData{},
		&RFQComment{},
		&CostingSheetData{},
	))

	app := &App{
		db:                     db,
		cache:                  NewCache(),
		startupImporting:       false,
		startupImportStartTime: time.Now(),
		currentUserID:          "test-user",
		currentUser: &User{
			Base:     Base{ID: "test-user"},
			Username: "test-admin",
			RoleName: "admin",
			Role: Role{
				Name:        "admin",
				DisplayName: "Administrator",
				Permissions: `["*"]`,
			},
		},
	}
	t.Cleanup(app.cache.Stop)
	return app
}

func makeLifecycleOffer(t *testing.T, app *App, number, stage string, validity time.Time) *Offer {
	t.Helper()
	offer := &Offer{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferNumber:   number,
		CustomerID:    "CUST-1",
		CustomerName:  "Acme Instrumentation",
		Stage:         stage,
		QuotationDate: time.Now().AddDate(0, 0, -40),
		ValidityDate:  validity,
		TotalValueBHD: 1000.000,
	}
	require.NoError(t, app.db.Create(offer).Error)
	item := &OfferItem{
		Base:        Base{ID: uuid.New().String()},
		OfferID:     offer.ID,
		LineNumber:  1,
		Description: "Flow transmitter",
		Quantity:    2,
		UnitPrice:   500.000,
		TotalPrice:  1000.000,
	}
	require.NoError(t, app.db.Create(item).Error)
	return offer
}

// I-19: A revision clones line items, links back to the source and the root,
// stamps a fresh revision number, and marks the source as superseded/expired.
func TestCreateOfferRevision_LineageAndSupersede(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	source := makeLifecycleOffer(t, app, "OFF-100", "Quoted", time.Now().AddDate(0, 0, 30))

	rev, err := app.CreateOfferRevision(source.ID)
	require.NoError(t, err)
	require.NotNil(t, rev)

	assert.Equal(t, source.ID, rev.RevisionOfOfferID, "revision must point back to its source")
	assert.Equal(t, source.ID, rev.RevisionRootOfferID, "first revision root is the original offer")
	assert.Equal(t, 1, rev.RevisionNumber)
	assert.Equal(t, "Quoted", rev.Stage)
	assert.Empty(t, rev.SupersededByOfferID)
	assert.Nil(t, rev.SupersededAt)
	assert.NotEqual(t, source.ID, rev.ID)

	// Line items were cloned onto the new offer with new IDs.
	var revItems []OfferItem
	require.NoError(t, app.db.Where("offer_id = ?", rev.ID).Find(&revItems).Error)
	require.Len(t, revItems, 1)
	assert.NotEqual(t, "", revItems[0].ID)
	assert.Equal(t, "Flow transmitter", revItems[0].Description)

	// Source is now superseded and expired.
	var reloadedSource Offer
	require.NoError(t, app.db.First(&reloadedSource, "id = ?", source.ID).Error)
	assert.Equal(t, rev.ID, reloadedSource.SupersededByOfferID)
	require.NotNil(t, reloadedSource.SupersededAt)
	assert.Equal(t, "Expired", reloadedSource.Stage)
}

// I-19: revising a revision keeps the root pointing at the original offer.
func TestCreateOfferRevision_RootPreservedAcrossChain(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	original := makeLifecycleOffer(t, app, "OFF-200", "Quoted", time.Now().AddDate(0, 0, 30))

	rev1, err := app.CreateOfferRevision(original.ID)
	require.NoError(t, err)
	require.Equal(t, 1, rev1.RevisionNumber)

	rev2, err := app.CreateOfferRevision(rev1.ID)
	require.NoError(t, err)
	assert.Equal(t, rev1.ID, rev2.RevisionOfOfferID, "R2 links to R1")
	assert.Equal(t, original.ID, rev2.RevisionRootOfferID, "root stays the original across the chain")
	assert.Equal(t, 2, rev2.RevisionNumber)
	assert.Equal(t, "OFF-200-R2", rev2.OfferNumber)
}

// I-19: won offers may not be re-quoted through the revision path.
func TestCreateOfferRevision_RejectsWon(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	won := makeLifecycleOffer(t, app, "OFF-300", "Won", time.Now().AddDate(0, 0, 30))

	rev, err := app.CreateOfferRevision(won.ID)
	require.Error(t, err)
	assert.Nil(t, rev)
}

// I-20: an expired offer renews into a fresh Quoted offer linked to the source
// with a new 30-day validity window.
func TestRenewOffer_ExpiredLinksBack(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	expired := makeLifecycleOffer(t, app, "OFF-400", "Expired", time.Now().AddDate(0, 0, -5))

	renewed, err := app.RenewOffer(expired.ID)
	require.NoError(t, err)
	require.NotNil(t, renewed)
	assert.Equal(t, expired.ID, renewed.RevisionOfOfferID)
	assert.Equal(t, "Quoted", renewed.Stage)
	assert.True(t, renewed.ValidityDate.After(time.Now()), "renewal opens a forward validity window")
}

// I-20: a Quoted offer still inside its validity window is not renewable.
func TestRenewOffer_RejectsInValidityOffer(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	live := makeLifecycleOffer(t, app, "OFF-500", "Quoted", time.Now().AddDate(0, 0, 30))

	renewed, err := app.RenewOffer(live.ID)
	require.Error(t, err)
	assert.Nil(t, renewed)
}

// I-20: a Quoted offer past its validity date is treated as lapsed and renews.
func TestRenewOffer_LapsedQuotedRenews(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	lapsed := makeLifecycleOffer(t, app, "OFF-600", "Quoted", time.Now().AddDate(0, 0, -1))

	renewed, err := app.RenewOffer(lapsed.ID)
	require.NoError(t, err)
	require.NotNil(t, renewed)
	assert.Equal(t, lapsed.ID, renewed.RevisionOfOfferID)
}

// I-21: the commercial-fields mutator writes only whitelisted columns and
// ignores anything outside the allow-list.
func TestUpdateOpportunityCommercialFields_WhitelistEnforced(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	opp := &Opportunity{
		Base:         Base{ID: uuid.New().String()},
		FolderNumber: "F-1",
		Title:        "Original title",
		Stage:        "Qualified",
		Confidence:   0.10,
		Regime:       1,
	}
	require.NoError(t, app.db.Create(opp).Error)

	updated, err := app.UpdateOpportunityCommercialFields(opp.ID, map[string]any{
		"title":      "Corrected title", // whitelisted -> applied
		"stage":      "Proposal",        // whitelisted -> applied
		"confidence": 0.99,              // NOT whitelisted -> ignored
		"regime":     3,                 // NOT whitelisted -> ignored
		"id":         "hacked",          // NOT whitelisted -> ignored
	})
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, "Corrected title", updated.Title)
	assert.Equal(t, "Proposal", updated.Stage)
	assert.Equal(t, 0.10, updated.Confidence, "non-whitelisted confidence must be untouched")
	assert.Equal(t, 1, updated.Regime, "non-whitelisted regime must be untouched")
	assert.Equal(t, opp.ID, updated.ID, "primary key must be untouched")

	// A change-log comment records exactly the whitelisted fields that changed.
	var comments []OpportunityComment
	require.NoError(t, app.db.Where("opportunity_id = ?", opp.ID).Find(&comments).Error)
	require.Len(t, comments, 1)
	assert.Contains(t, comments[0].Comment, "stage")
	assert.Contains(t, comments[0].Comment, "title")
	assert.NotContains(t, comments[0].Comment, "confidence")
}

// I-21: supplying no whitelisted keys is an error, not a no-op write.
func TestUpdateOpportunityCommercialFields_RejectsEmptyAfterWhitelist(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	opp := &Opportunity{Base: Base{ID: uuid.New().String()}, Title: "Untouched"}
	require.NoError(t, app.db.Create(opp).Error)

	updated, err := app.UpdateOpportunityCommercialFields(opp.ID, map[string]any{
		"confidence": 0.5,
		"regime":     2,
	})
	require.Error(t, err)
	assert.Nil(t, updated)
}

// I-21: the unified thread merges offer notes and linked-opportunity comments
// and returns them oldest-first.
func TestGetUnifiedOfferThread_ChronologicalOrdering(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	offer := makeLifecycleOffer(t, app, "OFF-700", "Quoted", time.Now().AddDate(0, 0, 30))

	// Linked opportunity (by offer_id) with a comment in the middle of the timeline.
	opp := &Opportunity{
		Base:         Base{ID: uuid.New().String()},
		FolderNumber: "F-700",
		OfferID:      offer.ID,
		Title:        "Linked opp",
	}
	require.NoError(t, app.db.Create(opp).Error)

	base := time.Now().Add(-3 * time.Hour)
	require.NoError(t, app.db.Create(&OfferNote{
		Base:     Base{ID: uuid.New().String(), CreatedBy: "Alice", CreatedAt: base},
		OfferID:  offer.ID,
		NoteDate: base,
		Content:  "First: offer note",
	}).Error)
	require.NoError(t, app.db.Create(&OpportunityComment{
		OpportunityID: opp.ID,
		Comment:       "Second: opp comment",
		CreatedBy:     "Bob",
		CreatedAt:     base.Add(1 * time.Hour),
	}).Error)
	require.NoError(t, app.db.Create(&OfferNote{
		Base:     Base{ID: uuid.New().String(), CreatedBy: "Alice", CreatedAt: base.Add(2 * time.Hour)},
		OfferID:  offer.ID,
		NoteDate: base.Add(2 * time.Hour),
		Content:  "Third: offer note",
	}).Error)

	thread, err := app.GetUnifiedOfferThread(offer.ID)
	require.NoError(t, err)
	require.Len(t, thread, 3)
	assert.Equal(t, "First: offer note", thread[0].Comment)
	assert.Equal(t, "Second: opp comment", thread[1].Comment)
	assert.Equal(t, "Third: offer note", thread[2].Comment)
	assert.Equal(t, "opportunity", thread[1].SourceType)

	// Entries must be non-decreasing in time.
	for i := 1; i < len(thread); i++ {
		assert.False(t, thread[i].CreatedAt.Before(thread[i-1].CreatedAt))
	}
}

// I-21: duplicate identical entries collapse to one.
func TestGetUnifiedOfferThread_Dedupes(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	offer := makeLifecycleOffer(t, app, "OFF-800", "Quoted", time.Now().AddDate(0, 0, 30))

	ts := time.Now().Add(-time.Hour).Truncate(time.Second)
	for i := 0; i < 2; i++ {
		require.NoError(t, app.db.Create(&OfferNote{
			Base:     Base{ID: uuid.New().String(), CreatedBy: "Alice", CreatedAt: ts},
			OfferID:  offer.ID,
			NoteDate: ts,
			Content:  "Same content",
		}).Error)
	}

	thread, err := app.GetUnifiedOfferThread(offer.ID)
	require.NoError(t, err)
	assert.Len(t, thread, 1, "identical offer notes at the same instant collapse to one entry")
}

// GetCostingsByOpportunity returns pipeline costings (rfq_id==0) for the given
// opportunity, newest revision first, and ignores other opportunities' rows.
func TestGetCostingsByOpportunity_ScopedNewestFirst(t *testing.T) {
	app := setupOfferLifecycleTestApp(t)
	oppID := uuid.New().String()
	otherID := uuid.New().String()

	require.NoError(t, app.db.Create(&CostingSheetData{
		RFQID: 0, OpportunityRecordID: oppID, RevisionNumber: 1, Status: "draft",
	}).Error)
	require.NoError(t, app.db.Create(&CostingSheetData{
		RFQID: 0, OpportunityRecordID: oppID, RevisionNumber: 2, Status: "approved",
	}).Error)
	require.NoError(t, app.db.Create(&CostingSheetData{
		RFQID: 0, OpportunityRecordID: otherID, RevisionNumber: 1, Status: "draft",
	}).Error)
	// An RFQ-scoped costing (rfq_id != 0) must never appear here.
	require.NoError(t, app.db.Create(&CostingSheetData{
		RFQID: 7, OpportunityRecordID: oppID, RevisionNumber: 5, Status: "draft",
	}).Error)

	costings, err := app.GetCostingsByOpportunity(oppID)
	require.NoError(t, err)
	require.Len(t, costings, 2)
	assert.Equal(t, 2, costings[0].RevisionNumber, "newest revision first")
	assert.Equal(t, 1, costings[1].RevisionNumber)

	empty, err := app.GetCostingsByOpportunity("")
	require.NoError(t, err)
	assert.Empty(t, empty)
}
