package evidence

import (
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)

func validInput() FactInput {
	return FactInput{
		Source: SourceIdentity{
			ID:    "src-001",
			Kind:  SourceDocument,
			Label: "Invoice #42",
			Hash:  "abc123",
		},
		PrivacyClass:  PrivacyConfidential,
		Confidence:    ConfidenceHigh,
		LinkedObjects: []string{"inv-001", "pay-007"},
		AuditRefs: []AuditRef{
			{Type: "intake", SourceID: "src-001", Summary: "Observed on upload", Timestamp: testNow},
		},
	}
}

func TestNewFactValidInput(t *testing.T) {
	input := validInput()
	fact, err := NewFact(input, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(fact.ID, "evd_") {
		t.Errorf("ID %q does not start with evd_", fact.ID)
	}
	if fact.Source.ID != input.Source.ID {
		t.Errorf("Source.ID: got %q, want %q", fact.Source.ID, input.Source.ID)
	}
	if fact.Source.Kind != input.Source.Kind {
		t.Errorf("Source.Kind: got %q, want %q", fact.Source.Kind, input.Source.Kind)
	}
	if fact.Source.Label != input.Source.Label {
		t.Errorf("Source.Label: got %q, want %q", fact.Source.Label, input.Source.Label)
	}
	if fact.Source.Hash != input.Source.Hash {
		t.Errorf("Source.Hash: got %q, want %q", fact.Source.Hash, input.Source.Hash)
	}
	if fact.PrivacyClass != input.PrivacyClass {
		t.Errorf("PrivacyClass: got %q, want %q", fact.PrivacyClass, input.PrivacyClass)
	}
	if fact.Confidence != input.Confidence {
		t.Errorf("Confidence: got %q, want %q", fact.Confidence, input.Confidence)
	}
	if fact.Status != StatusObserved {
		t.Errorf("Status: got %q, want %q", fact.Status, StatusObserved)
	}
	if !fact.ObservedAt.Equal(testNow) {
		t.Errorf("ObservedAt: got %v, want %v", fact.ObservedAt, testNow)
	}
}

func TestNewFactRejectsEmptySourceID(t *testing.T) {
	cases := []struct {
		name string
		id   string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tab only", "\t"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validInput()
			input.Source.ID = tc.id
			_, err := NewFact(input, testNow)
			if err == nil {
				t.Fatal("expected error for empty Source.ID, got nil")
			}
		})
	}
}

func TestNewFactRejectsEmptySourceKind(t *testing.T) {
	input := validInput()
	input.Source.Kind = ""
	_, err := NewFact(input, testNow)
	if err == nil {
		t.Fatal("expected error for empty Source.Kind, got nil")
	}
}

func TestNewFactDefaultsPrivacyClass(t *testing.T) {
	input := validInput()
	input.PrivacyClass = ""
	fact, err := NewFact(input, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fact.PrivacyClass != PrivacyInternal {
		t.Errorf("PrivacyClass: got %q, want %q", fact.PrivacyClass, PrivacyInternal)
	}
}

func TestNewFactDefaultsConfidence(t *testing.T) {
	input := validInput()
	input.Confidence = ""
	fact, err := NewFact(input, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fact.Confidence != ConfidenceUnknown {
		t.Errorf("Confidence: got %q, want %q", fact.Confidence, ConfidenceUnknown)
	}
}

func TestNewFactDefaultsStatusToObserved(t *testing.T) {
	// Status is always set to StatusObserved on construction regardless of input
	fact, err := NewFact(validInput(), testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fact.Status != StatusObserved {
		t.Errorf("Status: got %q, want %q", fact.Status, StatusObserved)
	}
}

func TestNewFactPreservesLinkedObjects(t *testing.T) {
	input := validInput()
	input.LinkedObjects = []string{"obj-a", "obj-b", "obj-c"}
	fact, err := NewFact(input, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fact.LinkedObjects) != len(input.LinkedObjects) {
		t.Fatalf("LinkedObjects length: got %d, want %d", len(fact.LinkedObjects), len(input.LinkedObjects))
	}
	for i, want := range input.LinkedObjects {
		if fact.LinkedObjects[i] != want {
			t.Errorf("LinkedObjects[%d]: got %q, want %q", i, fact.LinkedObjects[i], want)
		}
	}
}

func TestNewFactPreservesAuditRefs(t *testing.T) {
	input := validInput()
	input.AuditRefs = []AuditRef{
		{Type: "upload", SourceID: "src-001", Summary: "File received", Timestamp: testNow},
		{Type: "review", SourceID: "src-001", Summary: "Operator reviewed", Timestamp: testNow.Add(time.Hour)},
	}
	fact, err := NewFact(input, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fact.AuditRefs) != len(input.AuditRefs) {
		t.Fatalf("AuditRefs length: got %d, want %d", len(fact.AuditRefs), len(input.AuditRefs))
	}
	for i, want := range input.AuditRefs {
		got := fact.AuditRefs[i]
		if got.Type != want.Type || got.SourceID != want.SourceID || got.Summary != want.Summary {
			t.Errorf("AuditRefs[%d]: got %+v, want %+v", i, got, want)
		}
	}
}

func TestIsKnownSourceKind(t *testing.T) {
	known := []SourceKind{
		SourceDocument,
		SourceBankLine,
		SourceJournal,
		SourceMessage,
		SourceEmail,
		SourceScan,
		SourceScreenshot,
		SourceFile,
		SourceObservation,
		SourceOther,
	}
	for _, k := range known {
		if !IsKnownSourceKind(k) {
			t.Errorf("IsKnownSourceKind(%q) = false, want true", k)
		}
	}
	if IsKnownSourceKind("bogus") {
		t.Error(`IsKnownSourceKind("bogus") = true, want false`)
	}
}

func TestIsKnownPrivacyClass(t *testing.T) {
	known := []PrivacyClass{
		PrivacyInternal,
		PrivacyConfidential,
		PrivacyRestricted,
	}
	for _, p := range known {
		if !IsKnownPrivacyClass(p) {
			t.Errorf("IsKnownPrivacyClass(%q) = false, want true", p)
		}
	}
	if IsKnownPrivacyClass("bogus") {
		t.Error(`IsKnownPrivacyClass("bogus") = true, want false`)
	}
}
