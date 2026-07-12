package evidence

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/finance/posting"
)

// buildBaseInput constructs a reusable CommandCenterInput with the given
// posting coverage totals. Cash, evidence sources, and follow-up tasks are
// held constant so the only variable across steps is the posting state.
func buildBaseInput(total, linked, missing int64) CommandCenterInput {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	return CommandCenterInput{
		Window: TimeWindow{
			Start: now,
			End:   now.AddDate(0, 0, 30),
			Label: "June 2024",
		},
		Cash: CashExposureInput{
			OpenAR:      5000,
			OverdueAR:   2000,
			DueInWindow: 1000,
		},
		EvidenceSources: []EvidenceSourceInput{
			{
				SourceType: "customer_invoice",
				Label:      "Customer invoices",
				Required:   5,
				Present:    5,
				Confidence: 0.95,
			},
			{
				SourceType: "purchase_order",
				Label:      "Purchase orders",
				Required:   4,
				Present:    3,
				Confidence: 0.75,
			},
		},
		BankAllocations: nil,
		PostingCoverage: posting.CoverageReport{
			Total:      total,
			Linked:     linked,
			Missing:    missing,
			IsComplete: missing == 0,
		},
		TrialBalanceGate: posting.TrialBalanceGate{
			FiscalYear:   2024,
			FiscalPeriod: 6,
			IsBalanced:   true,
			DebitTotal:   7000,
			CreditTotal:  7000,
			Difference:   0,
		},
		OpenFollowUpTasks: 2,
	}
}

// findPostingProposal returns the first ActionProposal whose
// RequiredDeterministicService is "finance_posting_service".
func findPostingProposal(proposals []ActionProposal) (ActionProposal, bool) {
	for _, p := range proposals {
		if p.RequiredDeterministicService == "finance_posting_service" {
			return p, true
		}
	}
	return ActionProposal{}, false
}

// TestClosedLoopProposalThroughDispatchToImprovedSnapshot proves the entire
// operator loop end-to-end:
//
//  1. Build CommandCenter with posting gaps → verify proposals generated.
//  2. Find the posting proposal.
//  3. Dispatch through PostingDispatcher with mock createDraft.
//  4. Rebuild CommandCenter with one fewer gap → verify snapshot diff.
func TestClosedLoopProposalThroughDispatchToImprovedSnapshot(t *testing.T) {
	ctx := context.Background()
	ts := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	// ── Step 1: Build initial CommandCenter with 3 missing journals ──────────
	input1 := buildBaseInput(10, 7, 3)
	center1 := BuildCommandCenter(input1)

	if len(center1.ActionProposals) == 0 {
		t.Fatal("step 1: expected ActionProposals to be non-empty when posting gaps exist")
	}
	if center1.Posting.MissingJournals != 3 {
		t.Fatalf("step 1: MissingJournals = %d, want 3", center1.Posting.MissingJournals)
	}

	snapshot1 := NewSnapshot(center1, 1, ts)

	// ── Step 2: Find the posting proposal ────────────────────────────────────
	proposal, ok := findPostingProposal(center1.ActionProposals)
	if !ok {
		t.Fatal("step 2: expected a proposal with RequiredDeterministicService=\"finance_posting_service\", none found")
	}
	if proposal.Action == "" {
		t.Fatal("step 2: posting proposal has empty Action")
	}

	// ── Step 3: Create approval and dispatch ──────────────────────────────────
	var capturedSourceType, capturedSourceID string
	mockCreateDraft := func(sourceType, sourceID string) (string, error) {
		capturedSourceType = sourceType
		capturedSourceID = sourceID
		return "journal_draft_test_001", nil
	}

	approval := DispatchApproval{
		Actor:      "operator@asymmflow.io",
		ActorType:  "operator",
		Reason:     "Reviewed missing journal gaps — approving draft creation.",
		ApprovedAt: ts,
		SourceType: "customer_invoice",
		SourceID:   "inv_001",
	}

	dispatcher := NewPostingDispatcher(mockCreateDraft)
	router := NewDispatchRouter(dispatcher)

	result, err := router.Dispatch(ctx, proposal, approval)
	if err != nil {
		t.Fatalf("step 3: Dispatch returned unexpected error: %v", err)
	}
	if !result.Executed {
		t.Fatal("step 3: expected result.Executed == true after operator dispatch")
	}
	if result.OutputRef != "journal_draft_test_001" {
		t.Fatalf("step 3: OutputRef = %q, want %q", result.OutputRef, "journal_draft_test_001")
	}

	// Verify the mock was actually called with the approval context.
	if capturedSourceType != "customer_invoice" {
		t.Fatalf("step 3: mock createDraft called with sourceType=%q, want %q", capturedSourceType, "customer_invoice")
	}
	if capturedSourceID != "inv_001" {
		t.Fatalf("step 3: mock createDraft called with sourceID=%q, want %q", capturedSourceID, "inv_001")
	}

	// ── Step 4: Rebuild CommandCenter with improved state (Missing 3→2) ───────
	input2 := buildBaseInput(10, 8, 2)
	center2 := BuildCommandCenter(input2)

	if center2.Posting.MissingJournals >= center1.Posting.MissingJournals {
		t.Fatalf("step 4: new MissingJournals (%d) must be < old (%d)",
			center2.Posting.MissingJournals, center1.Posting.MissingJournals)
	}

	snapshot2 := NewSnapshot(center2, 2, ts.Add(5*time.Minute))

	// ── Step 5: Diff and verify improvement ──────────────────────────────────
	diff := snapshot2.Diff(snapshot1)

	if !diff.HashChanged {
		t.Fatal("step 5: HashChanged must be true — state changed (fewer missing journals)")
	}
	if diff.ProposalCountDelta > 0 {
		t.Fatalf("step 5: ProposalCountDelta = %d, want <= 0 (fewer or equal proposals after fixing a gap)",
			diff.ProposalCountDelta)
	}

	// Confirm the diff version bookkeeping is correct.
	if diff.FromVersion != 1 {
		t.Fatalf("step 5: FromVersion = %d, want 1", diff.FromVersion)
	}
	if diff.ToVersion != 2 {
		t.Fatalf("step 5: ToVersion = %d, want 2", diff.ToVersion)
	}
}

// TestClosedLoopRejectsAgentDispatch proves the safety boundary: an agent
// actor must never be allowed to dispatch approved proposals.
func TestClosedLoopRejectsAgentDispatch(t *testing.T) {
	ctx := context.Background()

	input := buildBaseInput(10, 7, 3)
	center := BuildCommandCenter(input)

	proposal, ok := findPostingProposal(center.ActionProposals)
	if !ok {
		t.Fatal("expected a posting proposal to exist when MissingJournals > 0")
	}

	mockCreateDraft := func(sourceType, sourceID string) (string, error) {
		return fmt.Sprintf("journal_%s_%s", sourceType, sourceID), nil
	}

	agentApproval := DispatchApproval{
		Actor:      "auto-agent",
		ActorType:  "agent",
		SourceType: "customer_invoice",
		SourceID:   "inv_002",
		ApprovedAt: time.Now(),
	}

	router := NewDispatchRouter(NewPostingDispatcher(mockCreateDraft))

	_, err := router.Dispatch(ctx, proposal, agentApproval)
	if err == nil {
		t.Fatal("expected an error when ActorType is \"agent\", got nil")
	}
	if !strings.Contains(err.Error(), "agent") {
		t.Fatalf("error message should mention \"agent\", got: %q", err.Error())
	}
}

// TestClosedLoopSnapshotHashStableWhenNothingChanges proves that two
// CommandCenters built from identical inputs produce identical content hashes,
// and that the resulting diff shows no changes.
func TestClosedLoopSnapshotHashStableWhenNothingChanges(t *testing.T) {
	ts := time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)

	input := buildBaseInput(10, 7, 3)

	centerA := BuildCommandCenter(input)
	centerB := BuildCommandCenter(input)

	snapshotA := NewSnapshot(centerA, 1, ts)
	snapshotB := NewSnapshot(centerB, 2, ts.Add(10*time.Second))

	diff := snapshotB.Diff(snapshotA)

	if diff.HashChanged {
		t.Fatalf("HashChanged must be false for identical inputs: hashA=%s hashB=%s",
			snapshotA.ContentHash, snapshotB.ContentHash)
	}
	if diff.ProposalCountDelta != 0 {
		t.Fatalf("ProposalCountDelta = %d, want 0", diff.ProposalCountDelta)
	}
	if diff.FollowUpCountDelta != 0 {
		t.Fatalf("FollowUpCountDelta = %d, want 0", diff.FollowUpCountDelta)
	}
	if diff.EvidenceSourcesDelta != 0 {
		t.Fatalf("EvidenceSourcesDelta = %d, want 0", diff.EvidenceSourcesDelta)
	}
	if diff.AllocationCountDelta != 0 {
		t.Fatalf("AllocationCountDelta = %d, want 0", diff.AllocationCountDelta)
	}
}
