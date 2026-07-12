package main

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestStageMigration_AggregateBeforeAfter proves the historical migration
// moves the dashboard's financial aggregates: mis-tagged terminal legacy
// stages ("Order Placed", "Closed (Payment)", "Closed (Lost)") were being
// counted as ACTIVE pipeline (inflating PipelineValueBHD) and excluded from
// totalClosed (deflating/distorting WinRate) before the migration runs.
func TestStageMigration_AggregateBeforeAfter(t *testing.T) {
	app := makeDashboardTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Opportunity{}))

	seed := []struct {
		stage      string
		revenueBHD float64
	}{
		// Terminal-legacy: today wrongly counted as active pipeline.
		{"Order Placed", 10000},
		{"Closed (Payment)", 20000},
		{"Closed (Lost)", 5000},
		// Active-legacy: stays active pipeline before AND after (just renamed).
		{"RFQ Received", 3000},
		{"Offer Sent", 4000},
		// Already-canonical terminal stages: unaffected by the migration.
		{"Won", 7000},
		{"Lost", 2000},
		// Already-canonical active stage: unaffected by the migration.
		{"New", 1000},
	}
	for _, s := range seed {
		opp := Opportunity{
			Base:         Base{ID: uuid.New().String()},
			FolderNumber: uuid.New().String(),
			Stage:        s.stage,
			RevenueBHD:   s.revenueBHD,
		}
		require.NoError(t, app.db.Create(&opp).Error)
	}

	before, err := app.GetDashboardStats()
	require.NoError(t, err)

	require.NoError(t, app.migrateOpportunityStageVocabulary())

	after, err := app.GetDashboardStats()
	require.NoError(t, err)

	t.Logf("BEFORE WinRate=%.2f%% PipelineValueBHD=%.3f  AFTER WinRate=%.2f%% PipelineValueBHD=%.3f",
		before.WinRate, before.PipelineValueBHD, after.WinRate, after.PipelineValueBHD)

	require.NotEqual(t, before.WinRate, after.WinRate,
		"migration should change WinRate: mis-tagged terminal rows were excluded from totalClosed before migration")
	require.Less(t, after.PipelineValueBHD, before.PipelineValueBHD,
		"migration should shrink PipelineValueBHD: mis-tagged terminal rows were counted as active pipeline before migration")

	// Pin the exact numbers so a future regression is caught precisely, not
	// just "some change happened".
	require.InDelta(t, 50.0, before.WinRate, 0.001)
	require.InDelta(t, 60.0, after.WinRate, 0.001)
	require.InDelta(t, 43000.0, before.PipelineValueBHD, 0.001)
	require.InDelta(t, 8000.0, after.PipelineValueBHD, 0.001)
}

// TestStageMigration_Idempotent proves running the historical migration
// twice is a no-op the second time: canonical values never match the legacy
// WHERE clauses, so nothing changes on re-run.
func TestStageMigration_Idempotent(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Opportunity{}, &RFQData{}))

	require.NoError(t, app.db.Create(&Opportunity{Base: Base{ID: uuid.New().String()}, FolderNumber: uuid.New().String(), Stage: "Order Placed", RevenueBHD: 100}).Error)
	require.NoError(t, app.db.Create(&Opportunity{Base: Base{ID: uuid.New().String()}, FolderNumber: uuid.New().String(), Stage: "Won", RevenueBHD: 50}).Error)
	require.NoError(t, app.db.Create(&RFQData{RFQNumber: "STAGE-IDEMP-1", Client: "Idempotent Test 1", Stage: "RFQ Received"}).Error)
	require.NoError(t, app.db.Create(&RFQData{RFQNumber: "STAGE-IDEMP-2", Client: "Idempotent Test 2", Stage: "Closed (Lost)"}).Error)

	require.NoError(t, app.migrateOpportunityStageVocabulary())

	var oppStagesAfterFirst []string
	require.NoError(t, app.db.Model(&Opportunity{}).Order("id").Pluck("stage", &oppStagesAfterFirst).Error)
	var rfqStagesAfterFirst []string
	require.NoError(t, app.db.Model(&RFQData{}).Order("id").Pluck("stage", &rfqStagesAfterFirst).Error)

	// Sanity: the first run actually mapped legacy -> canonical.
	require.Contains(t, oppStagesAfterFirst, "Won")
	require.NotContains(t, oppStagesAfterFirst, "Order Placed")
	require.Contains(t, rfqStagesAfterFirst, "New")
	require.Contains(t, rfqStagesAfterFirst, "Lost")
	require.NotContains(t, rfqStagesAfterFirst, "RFQ Received")
	require.NotContains(t, rfqStagesAfterFirst, "Closed (Lost)")

	require.NoError(t, app.migrateOpportunityStageVocabulary())

	var oppStagesAfterSecond []string
	require.NoError(t, app.db.Model(&Opportunity{}).Order("id").Pluck("stage", &oppStagesAfterSecond).Error)
	var rfqStagesAfterSecond []string
	require.NoError(t, app.db.Model(&RFQData{}).Order("id").Pluck("stage", &rfqStagesAfterSecond).Error)

	require.Equal(t, oppStagesAfterFirst, oppStagesAfterSecond,
		"second migration run must not change any Opportunity.Stage values (0 rows affected)")
	require.Equal(t, rfqStagesAfterFirst, rfqStagesAfterSecond,
		"second migration run must not change any RFQData.Stage values (0 rows affected)")
}

// TestStageWrite_RejectsNonCanonical proves the unified allowlist actually
// rejects non-canonical stage values on the live write paths.
func TestStageWrite_RejectsNonCanonical(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&RFQData{}))

	require.NoError(t, app.db.Create(&RFQData{ID: 500, Client: "Stage Reject Test", Stage: "New"}).Error)

	err := app.UpdateRFQStage(500, "Bogus Stage")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid stage")

	require.NoError(t, app.UpdateRFQStage(500, "Won"))
	var reloaded RFQData
	require.NoError(t, app.db.First(&reloaded, "id = ?", uint(500)).Error)
	require.Equal(t, "Won", reloaded.Stage)

	require.Error(t, validateOpportunityStageValue("Bogus"))
	// "Expired" was missing from the old 7-value list; the unified allowlist adds it.
	require.NoError(t, validateOpportunityStageValue("Expired"))
	require.NoError(t, validateOpportunityStageValue("New"))
}

// TestCanonicalize_MapsRatifiedRows is a direct unit test of
// canonicalizeOpportunityStage against the owner-ratified migration map.
func TestCanonicalize_MapsRatifiedRows(t *testing.T) {
	cases := []struct {
		raw      string
		expected string
		mapped   bool
	}{
		{"Order Placed", "Won", true},
		{"Follow-up/Eval", "Quoted", true},
		{"RFQ Received", "New", true},
		{"", "New", true},
		{"Won", "Won", false},
		{"Closed (Payment)", "Won", true},
		{"Closed (Lost)", "Lost", true},
		{"Closed", "Closed", false}, // unrecognized: left unchanged, never guessed
	}
	for _, c := range cases {
		got, mapped := canonicalizeOpportunityStage(c.raw)
		require.Equal(t, c.expected, got, "canonicalizeOpportunityStage(%q) value", c.raw)
		require.Equal(t, c.mapped, mapped, "canonicalizeOpportunityStage(%q) mapped flag", c.raw)
	}
}
