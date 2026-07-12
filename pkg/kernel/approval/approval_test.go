package approval

import (
	"strings"
	"testing"
	"time"
)

// shared fixture time used across all tests.
var fixedNow = time.Date(2026, 5, 27, 10, 0, 0, 123456789, time.UTC)

// validInput returns a fully populated RecordInput that passes all validation.
func validInput() RecordInput {
	return RecordInput{
		SubjectKey:    "proposal-abc-123",
		SubjectType:   "cashflow_proposal",
		Decision:      DecisionPending,
		Actor:         "alice",
		ActorType:     ActorOperator,
		Reason:        "looks good",
		CorrelationID: "corr-xyz-999",
	}
}

// --- NewRecord tests ----------------------------------------------------------

func TestNewRecordValidInput(t *testing.T) {
	rec, err := NewRecord(validInput(), fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(rec.ID, "apr_") {
		t.Errorf("ID should start with 'apr_', got %q", rec.ID)
	}
	if rec.SubjectKey != "proposal-abc-123" {
		t.Errorf("SubjectKey not preserved: got %q", rec.SubjectKey)
	}
	if rec.SubjectType != "cashflow_proposal" {
		t.Errorf("SubjectType not preserved: got %q", rec.SubjectType)
	}
	if rec.Decision != DecisionPending {
		t.Errorf("Decision not preserved: got %q", rec.Decision)
	}
	if rec.Actor != "alice" {
		t.Errorf("Actor not preserved: got %q", rec.Actor)
	}
	if rec.ActorType != ActorOperator {
		t.Errorf("ActorType not preserved: got %q", rec.ActorType)
	}
	if rec.Reason != "looks good" {
		t.Errorf("Reason not preserved: got %q", rec.Reason)
	}
	if rec.CorrelationID != "corr-xyz-999" {
		t.Errorf("CorrelationID not preserved: got %q", rec.CorrelationID)
	}
	if !rec.Timestamp.Equal(fixedNow) {
		t.Errorf("Timestamp not preserved: got %v", rec.Timestamp)
	}
}

func TestNewRecordRejectsEmptySubjectKey(t *testing.T) {
	cases := []string{"", "   ", "\t"}
	for _, key := range cases {
		t.Run("key="+key, func(t *testing.T) {
			inp := validInput()
			inp.SubjectKey = key
			_, err := NewRecord(inp, fixedNow)
			if err == nil {
				t.Error("expected error for empty SubjectKey, got nil")
			}
		})
	}
}

func TestNewRecordRejectsEmptyCorrelationID(t *testing.T) {
	cases := []string{"", "   ", "\t"}
	for _, corr := range cases {
		t.Run("corr="+corr, func(t *testing.T) {
			inp := validInput()
			inp.CorrelationID = corr
			_, err := NewRecord(inp, fixedNow)
			if err == nil {
				t.Error("expected error for empty CorrelationID, got nil")
			}
		})
	}
}

func TestNewRecordRejectsUnknownDecision(t *testing.T) {
	inp := validInput()
	inp.Decision = Decision("bogus")
	_, err := NewRecord(inp, fixedNow)
	if err == nil {
		t.Error("expected error for unknown Decision, got nil")
	}
}

func TestNewRecordRejectsAgentApproval(t *testing.T) {
	inp := validInput()
	inp.ActorType = ActorAgent
	inp.Decision = DecisionApproved
	_, err := NewRecord(inp, fixedNow)
	if err == nil {
		t.Fatal("expected error when agent tries to approve, got nil")
	}
	if !strings.Contains(err.Error(), "agent") {
		t.Errorf("error message should mention 'agent', got: %q", err.Error())
	}
}

func TestNewRecordAllowsAgentDraft(t *testing.T) {
	// Agents CAN suggest (needs_input) — only approval is blocked.
	inp := validInput()
	inp.ActorType = ActorAgent
	inp.Decision = DecisionNeedsInput
	rec, err := NewRecord(inp, fixedNow)
	if err != nil {
		t.Fatalf("agent+needs_input should be allowed, got error: %v", err)
	}
	if rec.ActorType != ActorAgent {
		t.Errorf("ActorType not preserved: got %q", rec.ActorType)
	}
	if rec.Decision != DecisionNeedsInput {
		t.Errorf("Decision not preserved: got %q", rec.Decision)
	}
}

// --- ValidTransition tests ---------------------------------------------------

func TestValidTransitionAllValid(t *testing.T) {
	valid := [][2]Decision{
		// pending_review exits (4)
		{DecisionPending, DecisionApproved},
		{DecisionPending, DecisionRejected},
		{DecisionPending, DecisionNeedsInput},
		{DecisionPending, DecisionSuperseded},
		// needs_input exits (4)
		{DecisionNeedsInput, DecisionPending},
		{DecisionNeedsInput, DecisionApproved},
		{DecisionNeedsInput, DecisionRejected},
		{DecisionNeedsInput, DecisionSuperseded},
		// approved exits (1)
		{DecisionApproved, DecisionSuperseded},
		// rejected exits (2)
		{DecisionRejected, DecisionPending},
		{DecisionRejected, DecisionSuperseded},
	}
	for _, pair := range valid {
		from, to := pair[0], pair[1]
		t.Run(string(from)+"->"+string(to), func(t *testing.T) {
			if !ValidTransition(from, to) {
				t.Errorf("expected valid transition %q -> %q, got false", from, to)
			}
		})
	}
}

func TestValidTransitionAllInvalid(t *testing.T) {
	invalid := [][2]Decision{
		// superseded is terminal
		{DecisionSuperseded, DecisionPending},
		{DecisionSuperseded, DecisionApproved},
		{DecisionSuperseded, DecisionRejected},
		{DecisionSuperseded, DecisionNeedsInput},
		// approved cannot revert
		{DecisionApproved, DecisionPending},
		{DecisionApproved, DecisionRejected},
		{DecisionApproved, DecisionNeedsInput},
		// rejected cannot reach approved or needs_input directly
		{DecisionRejected, DecisionApproved},
		{DecisionRejected, DecisionNeedsInput},
		// self-transitions
		{DecisionPending, DecisionPending},
		{DecisionApproved, DecisionApproved},
		{DecisionRejected, DecisionRejected},
		{DecisionNeedsInput, DecisionNeedsInput},
		{DecisionSuperseded, DecisionSuperseded},
		// unknown states
		{Decision(""), DecisionApproved},
		{Decision("unknown"), DecisionApproved},
	}
	for _, pair := range invalid {
		from, to := pair[0], pair[1]
		t.Run(string(from)+"->"+string(to), func(t *testing.T) {
			if ValidTransition(from, to) {
				t.Errorf("expected invalid transition %q -> %q, got true", from, to)
			}
		})
	}
}

// --- IsTerminal tests --------------------------------------------------------

func TestIsTerminal(t *testing.T) {
	cases := []struct {
		d        Decision
		terminal bool
	}{
		{DecisionSuperseded, true},
		{DecisionPending, false},
		{DecisionApproved, false},
		{DecisionRejected, false},
		{DecisionNeedsInput, false},
		{Decision("bogus"), false},
	}
	for _, tc := range cases {
		t.Run(string(tc.d), func(t *testing.T) {
			got := IsTerminal(tc.d)
			if got != tc.terminal {
				t.Errorf("IsTerminal(%q) = %v, want %v", tc.d, got, tc.terminal)
			}
		})
	}
}

// --- IsKnown tests -----------------------------------------------------------

func TestIsKnown(t *testing.T) {
	known := []Decision{
		DecisionPending,
		DecisionApproved,
		DecisionRejected,
		DecisionNeedsInput,
		DecisionSuperseded,
	}
	for _, d := range known {
		t.Run("known/"+string(d), func(t *testing.T) {
			if !IsKnown(d) {
				t.Errorf("IsKnown(%q) should be true", d)
			}
		})
	}

	unknown := []Decision{
		Decision("bogus"),
		Decision(""),
		Decision("APPROVED"),
		Decision("Pending"),
	}
	for _, d := range unknown {
		t.Run("unknown/"+string(d), func(t *testing.T) {
			if IsKnown(d) {
				t.Errorf("IsKnown(%q) should be false", d)
			}
		})
	}
}
