package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/finance/posting"
)

func TestAssessPostingReadinessBlocksUnbalancedTrialBalance(t *testing.T) {
	readiness := AssessPostingReadiness(
		posting.BuildCoverageReport([]posting.CoverageRow{{SourceType: "customer_invoice", Total: 2, Linked: 2}}),
		posting.TrialBalanceGate{LineCount: 2, Difference: 10, IsBalanced: false},
	)

	if readiness.Status != StatusBlocked {
		t.Fatalf("status = %s, want %s", readiness.Status, StatusBlocked)
	}
	if readiness.Priority != PriorityUrgent {
		t.Fatalf("priority = %s, want %s", readiness.Priority, PriorityUrgent)
	}
}

func TestAssessPostingReadinessDowngradesWhenSomeAccountsBalanced(t *testing.T) {
	readiness := AssessPostingReadiness(
		posting.BuildCoverageReport([]posting.CoverageRow{{SourceType: "customer_invoice", Total: 2, Linked: 2}}),
		posting.TrialBalanceGate{
			LineCount:          4,
			Difference:         50,
			IsBalanced:         false,
			BalancedAccounts:   []string{"1000-AR"},
			ImbalancedAccounts: []string{"2000-AP"},
		},
	)

	if readiness.Status != StatusAttention {
		t.Fatalf("status = %s, want %s", readiness.Status, StatusAttention)
	}
	if readiness.Priority != PriorityHigh {
		t.Fatalf("priority = %s, want %s", readiness.Priority, PriorityHigh)
	}
	if readiness.BalancedAccountCount != 1 {
		t.Fatalf("BalancedAccountCount = %d, want 1", readiness.BalancedAccountCount)
	}
	if readiness.ImbalancedAccountCount != 1 {
		t.Fatalf("ImbalancedAccountCount = %d, want 1", readiness.ImbalancedAccountCount)
	}
}

func TestBuildCommandCenterPrioritizesMissingEvidenceBeforeBankCleanup(t *testing.T) {
	center := BuildCommandCenter(CommandCenterInput{
		Cash: CashExposureInput{
			OpenAR:      1000,
			OverdueAR:   200,
			DueInWindow: 300,
		},
		EvidenceSources: []EvidenceSourceInput{
			{SourceType: "invoice_pdf", Label: "Invoice PDFs", Required: 4, Present: 3, Confidence: 0.95},
		},
		PostingCoverage: posting.BuildCoverageReport([]posting.CoverageRow{{SourceType: "customer_invoice", Total: 4, Linked: 4}}),
		TrialBalanceGate: posting.TrialBalanceGate{
			LineCount:  8,
			IsBalanced: true,
		},
		UnmatchedBankLines:   3,
		UnmatchedBankAmount:  77.1254,
		ExportableAuditItems: 9,
	})

	if center.OverallStatus != StatusAttention {
		t.Fatalf("overall status = %s, want %s", center.OverallStatus, StatusAttention)
	}
	if center.EvidenceSources[0].Missing != 1 {
		t.Fatalf("missing evidence = %d, want 1", center.EvidenceSources[0].Missing)
	}
	if center.NextAction != "Request or link missing evidence before exporting the pack." {
		t.Fatalf("next action = %q", center.NextAction)
	}
	if len(center.ActionProposals) == 0 {
		t.Fatal("expected action proposals")
	}
	for _, proposal := range center.ActionProposals {
		if proposal.MutatesState {
			t.Fatalf("proposal must not mutate state: %+v", proposal)
		}
		if proposal.RequiredDeterministicService == "" {
			t.Fatalf("proposal missing deterministic service: %+v", proposal)
		}
	}
	if center.UnmatchedBankAmount != 77.125 {
		t.Fatalf("unmatched amount = %.3f, want 77.125", center.UnmatchedBankAmount)
	}
}

func TestBuildCommandCenterCarriesAllocationAwareReadModel(t *testing.T) {
	center := BuildCommandCenter(CommandCenterInput{
		BankAllocations: []AllocationEvidenceInput{
			{
				AllocationID:        "alloc-1",
				BankStatementLineID: "bank-line-1",
				SourceType:          "customer_invoice",
				SourceID:            "inv-100",
				Amount:              125.5556,
				AllocationType:      "customer_invoice",
				Confidence:          0.97,
				AllocationStatus:    "matched",
			},
			{
				BankStatementLineID: "bank-line-2",
				SourceType:          "supplier_invoice",
				SourceID:            "bill-42",
				Amount:              50,
				AllocationType:      "partial",
				Confidence:          0.72,
				AllocationStatus:    "partial",
			},
			{
				AllocationID:     "alloc-conflict",
				SourceType:       "expense",
				Amount:           10,
				AllocationType:   "mixed",
				AllocationStatus: "conflict",
			},
		},
		PostingCoverage:  posting.BuildCoverageReport(nil),
		TrialBalanceGate: posting.TrialBalanceGate{IsBalanced: true},
	})

	if len(center.BankAllocations) != 3 {
		t.Fatalf("allocations = %+v", center.BankAllocations)
	}
	if center.BankAllocations[0].Amount != 125.556 || center.BankAllocations[0].Status != StatusReady {
		t.Fatalf("matched allocation = %+v", center.BankAllocations[0])
	}
	if center.BankAllocations[1].AllocationID == "" || center.BankAllocations[1].Status != StatusAttention {
		t.Fatalf("partial allocation = %+v", center.BankAllocations[1])
	}
	if center.OverallStatus != StatusBlocked {
		t.Fatalf("overall status = %s, want blocked", center.OverallStatus)
	}
	if center.AllocationSummary.TotalAllocations != 3 || center.AllocationSummary.Matched != 1 || center.AllocationSummary.Partial != 1 || center.AllocationSummary.Mixed != 1 || center.AllocationSummary.Conflicts != 1 {
		t.Fatalf("allocation summary = %+v", center.AllocationSummary)
	}
	if !strings.Contains(center.NextAction, "allocations") {
		t.Fatalf("next action = %q", center.NextAction)
	}
	foundProposal := false
	for _, proposal := range center.ActionProposals {
		if proposal.Action == "cashflowEvidence.inspectAllocations" {
			foundProposal = true
			if proposal.MutatesState {
				t.Fatalf("allocation proposal must be read-only: %+v", proposal)
			}
		}
	}
	if !foundProposal {
		t.Fatalf("missing allocation proposal: %+v", center.ActionProposals)
	}
}

func TestProposalReviewKeyIsStableForSameDeterministicAction(t *testing.T) {
	first := ActionProposal{
		Action:                       " cashflowEvidence.draftFollowUp ",
		Label:                        "Draft Receivables Follow-Up",
		SourceType:                   "Receivables",
		RequiredDeterministicService: "task_or_collections_service",
	}
	second := ActionProposal{
		Action:                       "cashflowEvidence.draftFollowUp",
		Label:                        "draft receivables follow-up",
		SourceType:                   "receivables",
		RequiredDeterministicService: "task_or_collections_service",
	}

	if ProposalReviewKey(first) != ProposalReviewKey(second) {
		t.Fatalf("proposal review keys should match: %q vs %q", ProposalReviewKey(first), ProposalReviewKey(second))
	}
}

func TestAssessCashExposureMarksOverdueMajorityUrgent(t *testing.T) {
	cash := AssessCashExposure(CashExposureInput{OpenAR: 1000, OverdueAR: 600})

	if cash.Priority != PriorityUrgent {
		t.Fatalf("priority = %s, want %s", cash.Priority, PriorityUrgent)
	}
	if cash.Status != StatusBlocked {
		t.Fatalf("status = %s, want %s", cash.Status, StatusBlocked)
	}
}

type stubReader struct {
	input CommandCenterInput
	err   error
}

func (r stubReader) LoadCashflowEvidence(ctx context.Context, window TimeWindow) (CommandCenterInput, error) {
	return r.input, r.err
}

func TestServiceBuildCommandCenterRequiresReader(t *testing.T) {
	_, err := NewService(nil).BuildCommandCenter(context.Background(), TimeWindow{Label: "May"})
	if err == nil {
		t.Fatal("expected error for missing reader")
	}
}

func TestServiceBuildCommandCenterPropagatesReaderError(t *testing.T) {
	wantErr := errors.New("boom")
	_, err := NewService(stubReader{err: wantErr}).BuildCommandCenter(context.Background(), TimeWindow{Label: "May"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestServiceBuildCommandCenterUsesRequestedWindowWhenReaderOmitsIt(t *testing.T) {
	center, err := NewService(stubReader{
		input: CommandCenterInput{
			PostingCoverage:  posting.BuildCoverageReport(nil),
			TrialBalanceGate: posting.TrialBalanceGate{IsBalanced: true},
		},
	}).BuildCommandCenter(context.Background(), TimeWindow{Label: "Next 30 days"})
	if err != nil {
		t.Fatalf("BuildCommandCenter: %v", err)
	}
	if center.Window.Label != "Next 30 days" {
		t.Fatalf("window label = %q, want requested window", center.Window.Label)
	}
}

func TestAgentBriefIsDraftOnlyAndTOONEncoded(t *testing.T) {
	center := BuildCommandCenter(CommandCenterInput{
		Cash: CashExposureInput{OpenAR: 1000, OverdueAR: 250},
		EvidenceSources: []EvidenceSourceInput{
			{SourceType: "invoice_pdf", Label: "Invoice PDFs", Required: 2, Present: 1, Confidence: 0.8},
		},
		PostingCoverage:  posting.BuildCoverageReport([]posting.CoverageRow{{SourceType: "customer_invoice", Total: 2, Linked: 1}}),
		TrialBalanceGate: posting.TrialBalanceGate{LineCount: 2, IsBalanced: true},
	})

	brief := BuildAgentBrief(center)
	if brief.MutatesState {
		t.Fatal("agent brief must not mutate state")
	}
	if len(brief.ForbiddenOperations) == 0 {
		t.Fatal("expected forbidden operations")
	}

	encoded, err := MarshalAgentBriefTOON(center, 0)
	if err != nil {
		t.Fatalf("MarshalAgentBriefTOON: %v", err)
	}
	if !strings.Contains(encoded, "format: TOON") || !strings.Contains(encoded, "mutates_state: false") {
		t.Fatalf("unexpected encoded brief:\n%s", encoded)
	}
}

func TestEvidencePackIsDeterministicReadModelExport(t *testing.T) {
	generatedAt := time.Date(2026, 5, 14, 8, 0, 0, 0, time.UTC)
	center := BuildCommandCenter(CommandCenterInput{
		Cash: CashExposureInput{OpenAR: 1000, DueInWindow: 250},
		EvidenceSources: []EvidenceSourceInput{
			{SourceType: "bank_reconciliation", Label: "Bank reconciliation", Required: 3, Present: 2, Confidence: 0.8},
		},
		BankAllocations: []AllocationEvidenceInput{{
			AllocationID:        "alloc-pack",
			BankStatementLineID: "bank-pack",
			SourceType:          "customer_invoice",
			SourceID:            "inv-pack",
			Amount:              88,
			AllocationType:      "customer_invoice",
			Confidence:          0.91,
			AllocationStatus:    "matched",
		}},
		PostingCoverage:      posting.BuildCoverageReport([]posting.CoverageRow{{SourceType: "customer_invoice", Total: 3, Linked: 2}}),
		TrialBalanceGate:     posting.TrialBalanceGate{LineCount: 2, IsBalanced: true},
		ExportableAuditItems: 4,
	})

	pack := BuildEvidencePack(center, generatedAt)
	if pack.SchemaVersion != EvidencePackSchemaVersion {
		t.Fatalf("schema version = %q", pack.SchemaVersion)
	}
	if pack.MutatesState {
		t.Fatal("evidence pack export must be read-only")
	}
	if len(pack.SourceSummary) != 1 || pack.SourceSummary[0].Missing != 1 {
		t.Fatalf("source summary = %+v", pack.SourceSummary)
	}
	if len(pack.ActionProposals) == 0 {
		t.Fatal("expected action proposals in evidence pack")
	}
	if len(pack.BankAllocations) != 1 || pack.AllocationSummary.Matched != 1 {
		t.Fatalf("allocation export = %+v", pack)
	}
	if len(pack.ForbiddenOperations) == 0 || len(pack.DeterministicHints) == 0 {
		t.Fatalf("missing safety metadata: %+v", pack)
	}

	encoded, err := MarshalEvidencePackJSON(center, generatedAt)
	if err != nil {
		t.Fatalf("MarshalEvidencePackJSON: %v", err)
	}
	var decoded EvidencePack
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json evidence pack should round trip: %v\n%s", err, string(encoded))
	}
	if decoded.SchemaVersion != EvidencePackSchemaVersion {
		t.Fatalf("decoded schema = %q", decoded.SchemaVersion)
	}

	toonPack, err := MarshalEvidencePackTOON(center, generatedAt, 0)
	if err != nil {
		t.Fatalf("MarshalEvidencePackTOON: %v", err)
	}
	if !strings.Contains(toonPack, "format: TOON") || !strings.Contains(toonPack, EvidencePackSchemaVersion) {
		t.Fatalf("unexpected TOON pack:\n%s", toonPack)
	}
}
