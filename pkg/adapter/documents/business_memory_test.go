package documents

import (
	"testing"
	"time"

	"ph_holdings_app/pkg/documents/intake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusinessMemoryCandidateRoundtrip(t *testing.T) {
	processedAt := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)
	original := intake.Candidate{
		ID: "candidate-1",
		Source: intake.SourceRef{
			ID:          "source-1",
			Label:       "invoice.pdf",
			Path:        "C:/Inbox/invoice.pdf",
			Kind:        intake.SourceKindPDF,
			ProcessedAt: &processedAt,
		},
		SourceKind:         intake.SourceKindPDF,
		BusinessObjectType: "customer_invoice",
		Classification: intake.Classification{
			Type:       "invoice",
			Method:     "classifier",
			RouteTo:    "finance.invoice.review",
			Reason:     "Matched invoice keywords",
			Keywords:   []string{"invoice", "amount"},
			Confidence: 0.91,
		},
		ExtractedFields: []intake.ExtractedField{
			{Name: "invoice_number", Label: "Invoice Number", Value: "INV-1", Status: intake.FieldStatusExtracted, Confidence: 0.96, Source: "ocr"},
			{Name: "trn", Label: "TRN", Status: intake.FieldStatusMissing, Source: "ocr"},
		},
		SuggestedLinks: []intake.SuggestedLink{
			{ID: "link-1", Label: "Review invoice", Reason: "Invoice candidate", BusinessObjectType: "customer_invoice", RequiredDeterministicService: "finance.invoice.review"},
		},
		ReviewStatus: intake.ReviewStatusNeedsReview,
		AuditRefs: []intake.AuditRef{
			{Type: "inbox_document", SourceID: "source-1", Summary: "Stored inbox record", Timestamp: "2026-05-14T10:30:00Z"},
		},
		Confidence: 0.91,
		Warnings:   []string{"missing trn"},
	}

	protoCandidate, err := BusinessMemoryCandidateToProto(original)
	require.NoError(t, err)
	back, err := BusinessMemoryCandidateFromProto(*protoCandidate)
	require.NoError(t, err)

	assert.Equal(t, original.ID, back.ID)
	assert.Equal(t, original.Source.ID, back.Source.ID)
	assert.Equal(t, original.Source.Label, back.Source.Label)
	require.NotNil(t, back.Source.ProcessedAt)
	assert.Equal(t, processedAt, *back.Source.ProcessedAt)
	assert.Equal(t, original.SourceKind, back.SourceKind)
	assert.Equal(t, original.BusinessObjectType, back.BusinessObjectType)
	assert.Equal(t, original.Classification, back.Classification)
	assert.Equal(t, original.ExtractedFields, back.ExtractedFields)
	assert.Equal(t, original.SuggestedLinks, back.SuggestedLinks)
	assert.Equal(t, original.ReviewStatus, back.ReviewStatus)
	assert.Equal(t, original.AuditRefs, back.AuditRefs)
	assert.Equal(t, original.Confidence, back.Confidence)
	assert.Equal(t, original.Warnings, back.Warnings)
}

func TestBusinessMemoryContextPackRoundtrip(t *testing.T) {
	candidate := intake.Candidate{
		ID:                 "candidate-2",
		Source:             intake.SourceRef{ID: "email-1", Label: "RFQ email", Kind: intake.SourceKindEmail},
		SourceKind:         intake.SourceKindEmail,
		BusinessObjectType: "opportunity",
		Classification:     intake.Classification{Type: "rfq", Confidence: 0.82},
		ExtractedFields: []intake.ExtractedField{
			{Name: "customer", Label: "Customer", Value: "NPC", Status: intake.FieldStatusExtracted, Confidence: 0.9, Source: "email"},
			{Name: "deadline", Label: "Deadline", Status: intake.FieldStatusMissing, Source: "email"},
		},
		SuggestedLinks: []intake.SuggestedLink{
			{ID: "link-2", Label: "Create opportunity", Reason: "RFQ received", BusinessObjectType: "opportunity", RequiredDeterministicService: "crm.opportunity.review"},
		},
		ReviewStatus: intake.ReviewStatusNeedsReview,
		AuditRefs:    []intake.AuditRef{{Type: "email", SourceID: "email-1", Summary: "Imported email"}},
		Confidence:   0.82,
	}
	original := intake.BuildContextPack(candidate)

	protoPack, err := BusinessMemoryContextPackToProto(original)
	require.NoError(t, err)
	back, err := BusinessMemoryContextPackFromProto(*protoPack)
	require.NoError(t, err)

	assert.Equal(t, original.CandidateID, back.CandidateID)
	assert.Equal(t, original.SourceSummary, back.SourceSummary)
	assert.Equal(t, original.SourceKind, back.SourceKind)
	assert.Equal(t, original.BusinessObjectType, back.BusinessObjectType)
	assert.Equal(t, original.Classification.Type, back.Classification.Type)
	assert.Equal(t, original.Classification.Confidence, back.Classification.Confidence)
	assert.Equal(t, original.ExtractedFields, back.ExtractedFields)
	assert.Equal(t, original.MissingFields, back.MissingFields)
	assert.Equal(t, original.SuggestedDeterministicServiceTargets, back.SuggestedDeterministicServiceTargets)
	assert.Equal(t, original.ReviewStatus, back.ReviewStatus)
	assert.Empty(t, back.Warnings)
	assert.Equal(t, original.AuditRefs, back.AuditRefs)
	assert.Equal(t, original.AllowedAgentActions, back.AllowedAgentActions)
	assert.Equal(t, original.ForbiddenAgentActions, back.ForbiddenAgentActions)
}

func TestBusinessMemoryReviewRecordRoundtrip(t *testing.T) {
	createdAt := time.Date(2026, 5, 14, 11, 0, 0, 0, time.UTC)
	original := intake.ReviewRecord{
		ID:                           "review-1",
		CandidateID:                  "candidate-1",
		SourceID:                     "source-1",
		Decision:                     intake.ReviewDecisionAcceptProposal,
		ReviewStatus:                 intake.ReviewStatusLinked,
		ProposedDeterministicService: "finance.invoice.review",
		Actor:                        "operator",
		Reason:                       "Fields verified",
		CorrelationID:                "corr-1",
		CreatedAt:                    createdAt,
	}

	protoRecord, err := BusinessMemoryReviewRecordToProto(original)
	require.NoError(t, err)
	back, err := BusinessMemoryReviewRecordFromProto(*protoRecord)
	require.NoError(t, err)

	assert.Equal(t, original, back)
}
