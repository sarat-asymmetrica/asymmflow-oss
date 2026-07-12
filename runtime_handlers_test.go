package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProcessInboxDocumentLocallyOfflineFallback verifies the offline-first inbox
// fallback: when the cloud Runtime is unreachable, ProcessInboxDocument delegates
// to processInboxDocumentLocally, which must classify supplied text locally and
// return a real document type + non-zero confidence — NOT the "unknown"/0.0 stub.
func TestProcessInboxDocumentLocallyOfflineFallback(t *testing.T) {
	app := setupTestApp(t)

	// No real file on disk → the fallback uses the supplied raw text path
	// (exercising local classification without needing the OCR HTTP engine).
	rawText := "Request for Quotation\nPlease quote analyzer spares and calibration."
	result, err := app.processInboxDocumentLocally("inbox/offline_rfq.pdf", rawText)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Regression guard against the old stub behaviour.
	require.NotEqual(t, "unknown", result.DetectedType, "offline fallback must classify, not return the unknown stub")
	require.Equal(t, "rfq", result.DetectedType)
	require.Greater(t, result.ClassificationConfidence, 0.0, "offline fallback must carry real confidence, not 0.0")
	require.Equal(t, rawText, result.ExtractedText)
	require.Equal(t, "local_ocr_fallback", result.Entities["source"])
	require.NotEmpty(t, result.SuggestedActions)
}
