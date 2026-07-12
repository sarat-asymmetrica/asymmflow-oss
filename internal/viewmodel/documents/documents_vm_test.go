package documents

import (
	"testing"
	"time"

	vm "ph_holdings_app/internal/viewmodel"
	"ph_holdings_app/pkg/documents/intake"
)

func TestBuildIntakeReviewVMMapsQueueAndSelectedCandidate(t *testing.T) {
	review := BuildIntakeReviewVM([]IntakeReviewInput{
		{
			ID:                 "cand-001",
			SourceLabel:        "RFQ email from GSC",
			SourceKind:         "email",
			BusinessObjectType: "rfq",
			Classification:     "RFQ",
			ReviewStatus:       "needs_review",
			Confidence:         0.76,
			ExtractedFields: []IntakeExtractedFieldInput{
				{Name: "customer", Label: "Customer", Value: "GSC", Status: "extracted", Confidence: 0.91, Required: true, SourceRef: "email:from"},
				{Name: "due_date", Label: "Due Date", Status: "missing", Required: true, SourceRef: "email:body"},
			},
			SuggestedActions: []IntakeActionProposalInput{
				{Action: "link_rfq", SourceType: "email", Label: "Create opportunity draft", Reason: "RFQ language and customer were detected.", Priority: "high", RequiredDeterministicService: "crm.opportunity_review"},
			},
			AuditRefs: []string{"audit:rfq-email"},
		},
		{
			ID:                 "cand-002",
			SourceLabel:        "Supplier invoice PDF",
			SourceKind:         "pdf",
			BusinessObjectType: "supplier_invoice",
			Classification:     "SupplierInvoice",
			ReviewStatus:       "linked",
			Confidence:         0.93,
		},
	}, "cand-001")

	if len(review.QueueMetrics) != 4 {
		t.Fatalf("queue metrics = %d, want 4", len(review.QueueMetrics))
	}
	if review.QueueMetrics[1].Label != "Review" || review.QueueMetrics[1].Value != "1" {
		t.Fatalf("review metric = %+v", review.QueueMetrics[1])
	}
	if review.Selected == nil {
		t.Fatal("selected candidate is nil")
	}
	if review.Selected.ID != "cand-001" {
		t.Fatalf("selected id = %q, want cand-001", review.Selected.ID)
	}
	if review.Selected.ReviewStatus.Label != "Needs Review" {
		t.Fatalf("status = %+v", review.Selected.ReviewStatus)
	}
	if len(review.Selected.ExtractedFields) != 2 {
		t.Fatalf("fields = %d, want 2", len(review.Selected.ExtractedFields))
	}
	if review.Selected.ExtractedFields[1].Status.Label != "Missing" {
		t.Fatalf("missing field status = %+v", review.Selected.ExtractedFields[1].Status)
	}
	if len(review.Selected.Sources) != 1 {
		t.Fatalf("sources = %d, want fallback source", len(review.Selected.Sources))
	}
	if review.Selected.Sources[0].Present != 1 || review.Selected.Sources[0].Missing != 1 {
		t.Fatalf("source completeness = %+v", review.Selected.Sources[0])
	}
	if got := review.Selected.ActionProposals[0].RequiredDeterministicService; got != "crm.opportunity_review" {
		t.Fatalf("proposal service = %q", got)
	}
}

func TestBuildIntakeReviewVMFromQueueStatesShowsPersistedReview(t *testing.T) {
	reviewedAt := time.Date(2026, 5, 14, 14, 0, 0, 0, time.UTC)
	state := intake.ReviewQueueState{
		Candidate: intake.Candidate{
			ID:                 "cand-reviewed",
			Source:             intake.SourceRef{ID: "src-reviewed", Label: "Invoice PDF", Kind: intake.SourceKindPDF},
			SourceKind:         intake.SourceKindPDF,
			BusinessObjectType: "customer_invoice",
			Classification:     intake.Classification{Type: "invoice", Confidence: 0.9},
			ReviewStatus:       intake.ReviewStatusNeedsReview,
			Confidence:         0.9,
			SuggestedLinks: []intake.SuggestedLink{{
				ID:                           "link-reviewed",
				Label:                        "Review invoice",
				Reason:                       "Invoice candidate",
				BusinessObjectType:           "customer_invoice",
				RequiredDeterministicService: "finance.invoice.review",
			}},
		},
		LastReview: &intake.ReviewRecord{
			ID:                           "review-1",
			CandidateID:                  "cand-reviewed",
			Decision:                     intake.ReviewDecisionAcceptProposal,
			ReviewStatus:                 intake.ReviewStatusLinked,
			Actor:                        "operator",
			Reason:                       "verified",
			CorrelationID:                "corr-reviewed",
			ProposedDeterministicService: "finance.invoice.review",
			CreatedAt:                    reviewedAt,
		},
		SourceAssets: []intake.SourceAsset{{
			ID:               "src-reviewed",
			Kind:             intake.SourceKindPDF,
			Path:             "C:\\Inbox\\invoice.pdf",
			Label:            "Invoice PDF",
			PrivacyClass:     intake.SourcePrivacyConfidential,
			ProcessingStatus: intake.SourceStatusReviewed,
			CandidateIDs:     []string{"cand-reviewed"},
			AuditRefs:        []intake.AuditRef{{Type: "review", SourceID: "review-1", Summary: "reviewed"}},
			FirstSeenAt:      reviewedAt.Add(-time.Hour),
			LastSeenAt:       reviewedAt,
		}},
	}

	review := BuildIntakeReviewVMFromQueueStates([]intake.ReviewQueueState{state}, "cand-reviewed")
	if review.Selected == nil {
		t.Fatal("selected candidate is nil")
	}
	if review.Selected.ReviewStatus.Label != "Linked" {
		t.Fatalf("status = %+v", review.Selected.ReviewStatus)
	}
	if review.Selected.LastReview == nil {
		t.Fatal("last review is nil")
	}
	if review.Selected.LastReview.Actor != "operator" || review.Selected.LastReview.CorrelationID != "corr-reviewed" {
		t.Fatalf("last review = %+v", review.Selected.LastReview)
	}
	if review.Selected.ServiceTarget != "finance.invoice.review" {
		t.Fatalf("service target = %q", review.Selected.ServiceTarget)
	}
	if review.Candidates[0].LastReviewAction != "accept_proposal" {
		t.Fatalf("summary last action = %q", review.Candidates[0].LastReviewAction)
	}
	if len(review.Selected.SourceRegistry) != 1 {
		t.Fatalf("source registry = %+v", review.Selected.SourceRegistry)
	}
	if review.Selected.SourceRegistry[0].SourceID != "src-reviewed" || !review.Selected.SourceRegistry[0].CurrentCandidate {
		t.Fatalf("source registry provenance = %+v", review.Selected.SourceRegistry[0])
	}
	if review.Selected.SourceRegistry[0].CandidateCount != 1 || review.Selected.SourceRegistry[0].AuditRefCount != 1 {
		t.Fatalf("source registry counts = %+v", review.Selected.SourceRegistry[0])
	}
	assertAction(t, review.Selected.ReviewCommands, "businessMemory.review.correctField")
	assertAction(t, review.Selected.ReviewCommands, "businessMemory.review.archive")
}

func TestBuildIntakeReviewVMKeepsActionsAsReviewProposals(t *testing.T) {
	review := BuildIntakeReviewVM([]IntakeReviewInput{{
		ID:                 "cand-003",
		SourceLabel:        "Bank statement scan",
		SourceKind:         "scan",
		BusinessObjectType: "bank_statement",
		ReviewStatus:       "new",
		Confidence:         0.64,
		Warnings:           []string{"low OCR confidence"},
	}}, "")

	if review.Selected == nil {
		t.Fatal("selected candidate is nil")
	}
	if review.QueueMetrics[0].Status != "blocked" {
		t.Fatalf("queue status = %q, want blocked", review.QueueMetrics[0].Status)
	}
	if len(review.Selected.ActionProposals) != 1 {
		t.Fatalf("proposal count = %d, want fallback proposal", len(review.Selected.ActionProposals))
	}
	proposal := review.Selected.ActionProposals[0]
	if proposal.Action != "review_proposal" {
		t.Fatalf("proposal action = %q, want review_proposal", proposal.Action)
	}
	if proposal.RequiredDeterministicService != "finance.review_link" {
		t.Fatalf("proposal service = %q", proposal.RequiredDeterministicService)
	}
	for _, command := range review.Selected.ReviewCommands {
		if command.Action == "approve" || command.Action == "link" || command.Action == "post" || command.Action == "delete" {
			t.Fatalf("review command exposes autonomous mutation: %+v", command)
		}
	}
}

func assertAction(t *testing.T, actions []vm.ActionButton, want string) {
	t.Helper()
	for _, action := range actions {
		if action.Action == want && action.Enabled {
			return
		}
	}
	t.Fatalf("enabled action %q not found in %+v", want, actions)
}

func TestBuildIntakeReviewVMFromCandidatesAdaptsCanonicalIntake(t *testing.T) {
	review := BuildIntakeReviewVMFromCandidates([]intake.Candidate{{
		ID:                 "cand-004",
		Source:             intake.SourceRef{ID: "src-004", Label: "Supplier invoice"},
		SourceKind:         intake.SourceKindPDF,
		BusinessObjectType: "supplier_invoice",
		Classification:     intake.Classification{Type: "SupplierInvoice", Confidence: 0.88},
		ReviewStatus:       intake.ReviewStatusNew,
		Confidence:         0.88,
		ExtractedFields: []intake.ExtractedField{{
			Name:       "invoice_number",
			Label:      "Invoice Number",
			Value:      "INV-2044",
			Status:     intake.FieldStatusExtracted,
			Confidence: 0.92,
			Source:     "inbox_document",
		}},
		SuggestedLinks: []intake.SuggestedLink{{
			ID:                           "link-supplier-invoice",
			Label:                        "Review supplier invoice",
			Reason:                       "Supplier invoice candidate was normalized.",
			BusinessObjectType:           "supplier_invoice",
			RequiredDeterministicService: "finance.supplier_invoice.review",
		}},
		AuditRefs: []intake.AuditRef{{Type: "inbox_document", SourceID: "src-004", Summary: "Normalized from stored InboxDocument."}},
	}}, "")

	if review.Selected == nil {
		t.Fatal("selected candidate is nil")
	}
	if review.Selected.SourceKind != "pdf" {
		t.Fatalf("source kind = %q", review.Selected.SourceKind)
	}
	if review.Selected.ExtractedFields[0].Value != "INV-2044" {
		t.Fatalf("field = %+v", review.Selected.ExtractedFields[0])
	}
	if review.Selected.ActionProposals[0].RequiredDeterministicService != "finance.supplier_invoice.review" {
		t.Fatalf("proposal = %+v", review.Selected.ActionProposals[0])
	}
	if len(review.Selected.AuditRefs) != 1 {
		t.Fatalf("audit refs = %+v", review.Selected.AuditRefs)
	}
}
