package main

import (
	"context"
	"errors"
	"testing"
	"time"

	cashflowevidence "ph_holdings_app/pkg/cashflow/evidence"
	"ph_holdings_app/pkg/finance/posting"
)

// stubEvidenceReader implements SnapshotReader for testing the evidence service
// without a database.
type stubEvidenceReader struct {
	input cashflowevidence.CommandCenterInput
	err   error
}

func (s stubEvidenceReader) LoadCashflowEvidence(ctx context.Context, window cashflowevidence.TimeWindow) (cashflowevidence.CommandCenterInput, error) {
	if s.err != nil {
		return cashflowevidence.CommandCenterInput{}, s.err
	}
	result := s.input
	if result.Window.Label == "" {
		result.Window = window
	}
	return result, nil
}

// balancedInput returns a minimal CommandCenterInput that produces a clean,
// non-blocked CommandCenter with no action proposals.
func balancedInput() cashflowevidence.CommandCenterInput {
	return cashflowevidence.CommandCenterInput{
		PostingCoverage:  posting.BuildCoverageReport([]posting.CoverageRow{{SourceType: "customer_invoice", Total: 4, Linked: 4}}),
		TrialBalanceGate: posting.TrialBalanceGate{LineCount: 4, IsBalanced: true},
	}
}

// missingJournalsInput returns a CommandCenterInput that will produce a posting
// proposal because some journals are missing.
func missingJournalsInput() cashflowevidence.CommandCenterInput {
	return cashflowevidence.CommandCenterInput{
		PostingCoverage:  posting.BuildCoverageReport([]posting.CoverageRow{{SourceType: "customer_invoice", Total: 4, Linked: 2}}),
		TrialBalanceGate: posting.TrialBalanceGate{LineCount: 4, IsBalanced: true},
		Cash: cashflowevidence.CashExposureInput{
			OpenAR:    500,
			OverdueAR: 100,
		},
	}
}

func TestCashflowEvidenceServiceBuildCommandCenterFromStub(t *testing.T) {
	input := missingJournalsInput()
	svc := cashflowevidence.NewService(stubEvidenceReader{input: input})
	window := cashflowevidence.TimeWindow{Label: "Next 30 days"}

	center, err := svc.BuildCommandCenter(context.Background(), window)
	if err != nil {
		t.Fatalf("BuildCommandCenter: %v", err)
	}

	// Missing journals (2 of 4) must surface as a posting proposal.
	if center.Posting.MissingJournals == 0 {
		t.Fatal("expected MissingJournals > 0 from stub input")
	}
	foundPostingProposal := false
	for _, p := range center.ActionProposals {
		if p.Action == "finance.requestDraftJournal" {
			foundPostingProposal = true
			if p.RequiredDeterministicService != "finance_posting_service" {
				t.Fatalf("posting proposal service = %q, want finance_posting_service", p.RequiredDeterministicService)
			}
		}
	}
	if !foundPostingProposal {
		t.Fatalf("expected finance.requestDraftJournal proposal, got: %+v", center.ActionProposals)
	}
}

func TestCashflowEvidenceServicePropagatesReaderError(t *testing.T) {
	wantErr := errors.New("database connection lost")
	svc := cashflowevidence.NewService(stubEvidenceReader{err: wantErr})

	_, err := svc.BuildCommandCenter(context.Background(), cashflowevidence.TimeWindow{Label: "May"})
	if err == nil {
		t.Fatal("expected error from reader, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("error chain should contain original error: got %v", err)
	}
}

func TestCashflowEvidenceServiceWindowDefaults(t *testing.T) {
	// Reader returns input without a window — service should inject the requested one.
	input := balancedInput()
	// input.Window is zero-valued — no label, no timestamps.
	svc := cashflowevidence.NewService(stubEvidenceReader{input: input})

	requestedWindow := cashflowevidence.TimeWindow{
		Start: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		Label: "May 2026",
	}

	center, err := svc.BuildCommandCenter(context.Background(), requestedWindow)
	if err != nil {
		t.Fatalf("BuildCommandCenter: %v", err)
	}
	if center.Window.Label != "May 2026" {
		t.Fatalf("window label = %q, want %q", center.Window.Label, "May 2026")
	}
	if !center.Window.Start.Equal(requestedWindow.Start) {
		t.Fatalf("window start = %v, want %v", center.Window.Start, requestedWindow.Start)
	}
}

func TestCashflowEvidenceServicePostingDispatchIntegration(t *testing.T) {
	// Build a CommandCenter from stub input that has missing journals.
	input := missingJournalsInput()
	svc := cashflowevidence.NewService(stubEvidenceReader{input: input})

	center, err := svc.BuildCommandCenter(context.Background(), cashflowevidence.TimeWindow{Label: "May"})
	if err != nil {
		t.Fatalf("BuildCommandCenter: %v", err)
	}

	// Verify the posting proposal is present.
	var postingProposal cashflowevidence.ActionProposal
	for _, p := range center.ActionProposals {
		if p.Action == "finance.requestDraftJournal" {
			postingProposal = p
			break
		}
	}
	if postingProposal.Action == "" {
		t.Fatal("expected finance.requestDraftJournal proposal in command center")
	}

	// Wire up a PostingDispatcher with a mock createDraft function.
	var capturedSourceType, capturedSourceID string
	createDraft := func(sourceType, sourceID string) (string, error) {
		capturedSourceType = sourceType
		capturedSourceID = sourceID
		return "journal_draft_integration_001", nil
	}

	router := cashflowevidence.NewDispatchRouter(
		cashflowevidence.NewPostingDispatcher(createDraft),
	)

	approval := cashflowevidence.DispatchApproval{
		Actor:      "finance@example.com",
		ActorType:  "operator",
		Reason:     "Operator approved draft journal creation.",
		SourceType: "posting_coverage",
		SourceID:   "customer_invoice_missing_001",
		ApprovedAt: time.Now(),
	}

	result, err := router.Dispatch(context.Background(), postingProposal, approval)
	if err != nil {
		t.Fatalf("router.Dispatch: %v", err)
	}
	if !result.Executed {
		t.Fatal("result.Executed = false, want true")
	}
	if result.OutputRef != "journal_draft_integration_001" {
		t.Fatalf("OutputRef = %q, want %q", result.OutputRef, "journal_draft_integration_001")
	}
	if capturedSourceType != "posting_coverage" || capturedSourceID != "customer_invoice_missing_001" {
		t.Fatalf("createDraft called with (%q, %q), want (%q, %q)",
			capturedSourceType, capturedSourceID, "posting_coverage", "customer_invoice_missing_001")
	}
}
