package main

// Wave 8 P2-1: UpdateRFQ must accept the full PH status vocabulary AND keep
// Stage synced to Status (the refactor pruned the vocabulary and dropped the
// `rfq.Stage = updates.Status` sync line). UpdateRFQStage must likewise accept
// the short status-style stage names PH allowed.
//
// Wave 9 Spec-07 (stage vocabulary unification): the Status->Stage sync now
// canonicalizes the derived Stage onto the 8-value canonical enum
// (stage_vocabulary.go) instead of persisting the raw legacy Status string
// verbatim — writing "Offer Sent" straight into Stage was exactly the kind
// of vocabulary fragmentation that inflated PipelineValueBHD / deflated
// WinRate on the dashboard. Status itself is untouched.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateRFQSyncsStageToStatus(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&RFQData{}))

	require.NoError(t, app.db.Create(&RFQData{
		ID:     1,
		Client: "Acme Instrumentation",
		Status: "New",
		Stage:  "RFQ Received",
	}).Error)

	// (A) Status vocabulary: a pipeline-stage name must be accepted, not rejected.
	updated, err := app.UpdateRFQ(1, RFQUpdateRequest{Status: "Offer Sent"})
	require.NoError(t, err)
	require.Equal(t, "Offer Sent", updated.Status)
	// (B) Status->Stage sync: Stage must track Status, canonicalized onto the
	// 8-value enum ("Offer Sent" -> "Quoted" per the ratified migration map).
	require.Equal(t, "Quoted", updated.Stage)

	// Persisted, not just in-memory.
	var reloaded RFQData
	require.NoError(t, app.db.First(&reloaded, "id = ?", uint(1)).Error)
	require.Equal(t, "Offer Sent", reloaded.Status)
	require.Equal(t, "Quoted", reloaded.Stage)

	// (C) Invalid status still rejected.
	_, err = app.UpdateRFQ(1, RFQUpdateRequest{Status: "Bogus"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid status")
}

func TestUpdateRFQStageAcceptsStatusVocabulary(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&RFQData{}))

	require.NoError(t, app.db.Create(&RFQData{
		ID:    2,
		Stage: "Follow-up/Eval",
	}).Error)

	// Short status-style stage names must be valid (parity with PH).
	require.NoError(t, app.UpdateRFQStage(2, "Won"))

	var reloaded RFQData
	require.NoError(t, app.db.First(&reloaded, "id = ?", uint(2)).Error)
	require.Equal(t, "Won", reloaded.Stage)

	// Terminal-state guard still fires for a legacy stored stage (canonicalized
	// before the invariant check).
	require.NoError(t, app.db.Model(&RFQData{}).Where("id = ?", uint(2)).Update("stage", "Closed (Lost)").Error)
	err := app.UpdateRFQStage(2, "Offer Sent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "revenue inflation guard")

	// AND for a row already migrated to the canonical terminal value "Lost" —
	// the regression the state machine must catch once stored stages are
	// canonical, not just legacy strings.
	require.NoError(t, app.db.Model(&RFQData{}).Where("id = ?", uint(2)).Update("stage", "Lost").Error)
	err = app.UpdateRFQStage(2, "Quoted")
	require.Error(t, err)
	require.Contains(t, err.Error(), "revenue inflation guard")

	// Canonical "Won" may only move to Lost or stay Won.
	require.NoError(t, app.db.Model(&RFQData{}).Where("id = ?", uint(2)).Update("stage", "Won").Error)
	err = app.UpdateRFQStage(2, "Quoted")
	require.Error(t, err)
	require.Contains(t, err.Error(), "data integrity guard")
	require.NoError(t, app.UpdateRFQStage(2, "Lost")) // Won -> Lost (payment failed) is allowed
}
