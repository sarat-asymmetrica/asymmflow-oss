package intake

import (
	"strings"
	"testing"
)

func TestBuildContextPackKeepsAgentsDraftOnly(t *testing.T) {
	candidate := Candidate{
		ID: "cand-ctx-1",
		Source: SourceRef{
			ID:    "email-77",
			Label: "RFQ email",
			Path:  "C:\\Inbox\\RFQ-77.eml",
			Kind:  SourceKindEmail,
		},
		SourceKind:         SourceKindEmail,
		BusinessObjectType: "opportunity",
		Classification: Classification{
			Type:       "RFQ",
			Confidence: 0.78,
		},
		ExtractedFields: []ExtractedField{
			{Name: "customer_name", Label: "Customer Name", Value: "GSC", Status: FieldStatusExtracted, Source: "email.body"},
			{Name: "bid_deadline", Label: "Bid Deadline", Status: FieldStatusMissing, Source: "email.body"},
		},
		SuggestedLinks: []SuggestedLink{
			{Label: "Create opportunity draft", RequiredDeterministicService: "crm.opportunity.review"},
		},
		ReviewStatus: ReviewStatusNeedsReview,
		AuditRefs: []AuditRef{
			{Type: "stored_inbox_document", SourceID: "email-77", Summary: "review source"},
		},
		Confidence: 0.78,
		Warnings:   []string{"missing bid deadline"},
	}

	pack := BuildContextPack(candidate)

	if pack.CandidateID != "cand-ctx-1" {
		t.Fatalf("candidate id = %q", pack.CandidateID)
	}
	if len(pack.MissingFields) != 1 || pack.MissingFields[0] != "Bid Deadline" {
		t.Fatalf("missing fields = %#v", pack.MissingFields)
	}
	if len(pack.SuggestedDeterministicServiceTargets) != 1 || pack.SuggestedDeterministicServiceTargets[0] != "crm.opportunity.review" {
		t.Fatalf("service targets = %#v", pack.SuggestedDeterministicServiceTargets)
	}
	assertContains(t, pack.AllowedAgentActions, AgentActionInspect)
	assertContains(t, pack.AllowedAgentActions, AgentActionAssembleContext)
	assertContains(t, pack.ForbiddenAgentActions, ForbiddenAgentActionApprove)
	assertContains(t, pack.ForbiddenAgentActions, ForbiddenAgentActionCreate)
}

func TestBuildContextPackTOONIncludesBoundaryAndEvidence(t *testing.T) {
	candidate := FromInboxDocument(InboxDocumentInput{
		DocumentID:   "bank-may",
		FileName:     "bank_statement_may.xlsx",
		FilePath:     "C:\\Inbox\\bank_statement_may.xlsx",
		DocumentType: "BankStatement",
		Status:       "NeedsReview",
		Confidence:   0.72,
		ExtractedData: map[string]string{
			"account_number":  "12345678",
			"closing_balance": "",
		},
		SuggestedActions: []string{"Import bank statement for reconciliation"},
	})

	text := BuildContextPackTOON(candidate)

	required := []string{
		"business_memory_intake_context:",
		"candidate_id:",
		"source_kind: excel",
		"business_object_type: bank_statement",
		"missing_fields:",
		"Closing Balance",
		"finance.banking.review",
		"allowed_agent_actions:",
		"inspect",
		"forbidden_agent_actions:",
		"create authoritative business records",
	}
	for _, snippet := range required {
		if !strings.Contains(text, snippet) {
			t.Fatalf("context pack missing %q:\n%s", snippet, text)
		}
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("%q not found in %#v", want, values)
}
