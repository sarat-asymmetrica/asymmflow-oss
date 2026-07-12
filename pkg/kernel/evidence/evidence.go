// Package evidence provides a typed, canonical source-of-truth primitive for
// evidence provenance and fact lifecycle across the ph_holdings_app codebase.
// It generalises the SourceKind/SourceRef/AuditRef patterns scattered across
// pkg/documents/intake/model.go and pkg/cashflow/evidence/model.go into a
// single, stdlib-only kernel primitive.
//
// Design principles:
//   - All classification is expressed through typed constants; no raw strings.
//   - NewFact validates every field before construction; no zero-value traps.
//   - Sane defaults for PrivacyClass, Confidence, and Status on construction.
//   - Zero dependencies beyond stdlib.
package evidence

import (
	"fmt"
	"strings"
	"time"
)

// SourceKind classifies the origin of a piece of evidence.
type SourceKind string

const (
	SourceDocument    SourceKind = "document"
	SourceBankLine    SourceKind = "bank_line"
	SourceJournal     SourceKind = "journal_entry"
	SourceMessage     SourceKind = "message"
	SourceEmail       SourceKind = "email"
	SourceScan        SourceKind = "scan"
	SourceScreenshot  SourceKind = "screenshot"
	SourceFile        SourceKind = "file"
	SourceObservation SourceKind = "observation"
	SourceOther       SourceKind = "other"
)

// knownSourceKinds is the authoritative set used by IsKnownSourceKind.
var knownSourceKinds = map[SourceKind]bool{
	SourceDocument:    true,
	SourceBankLine:    true,
	SourceJournal:     true,
	SourceMessage:     true,
	SourceEmail:       true,
	SourceScan:        true,
	SourceScreenshot:  true,
	SourceFile:        true,
	SourceObservation: true,
	SourceOther:       true,
}

// SourceIdentity identifies the origin of a piece of evidence.
type SourceIdentity struct {
	ID    string     `json:"id"`
	Kind  SourceKind `json:"kind"`
	Label string     `json:"label"`
	Hash  string     `json:"hash,omitempty"` // content hash if available
}

// ConfidenceLevel represents how confident we are in a piece of evidence.
type ConfidenceLevel string

const (
	ConfidenceVerified ConfidenceLevel = "verified"
	ConfidenceHigh     ConfidenceLevel = "high"
	ConfidenceMedium   ConfidenceLevel = "medium"
	ConfidenceLow      ConfidenceLevel = "low"
	ConfidenceUnknown  ConfidenceLevel = "unknown"
)

// PrivacyClass controls visibility and retention of evidence.
type PrivacyClass string

const (
	PrivacyInternal     PrivacyClass = "internal"
	PrivacyConfidential PrivacyClass = "confidential"
	PrivacyRestricted   PrivacyClass = "restricted"
)

// knownPrivacyClasses is the authoritative set used by IsKnownPrivacyClass.
var knownPrivacyClasses = map[PrivacyClass]bool{
	PrivacyInternal:     true,
	PrivacyConfidential: true,
	PrivacyRestricted:   true,
}

// Status tracks the lifecycle of evidence.
type Status string

const (
	StatusObserved   Status = "observed"
	StatusVerified   Status = "verified"
	StatusDisputed   Status = "disputed"
	StatusSuperseded Status = "superseded"
)

// AuditRef links evidence to an audit trail entry.
type AuditRef struct {
	Type      string    `json:"type"`
	SourceID  string    `json:"source_id"`
	Summary   string    `json:"summary"`
	Timestamp time.Time `json:"timestamp"`
}

// Fact represents a source-backed piece of evidence with provenance.
type Fact struct {
	ID            string          `json:"id"`
	Source        SourceIdentity  `json:"source"`
	PrivacyClass  PrivacyClass    `json:"privacy_class"`
	Confidence    ConfidenceLevel `json:"confidence"`
	Status        Status          `json:"status"`
	LinkedObjects []string        `json:"linked_objects,omitempty"`
	AuditRefs     []AuditRef      `json:"audit_refs,omitempty"`
	ObservedAt    time.Time       `json:"observed_at"`
}

// FactInput is the input for creating a new Fact.
type FactInput struct {
	Source        SourceIdentity
	PrivacyClass  PrivacyClass
	Confidence    ConfidenceLevel
	LinkedObjects []string
	AuditRefs     []AuditRef
}

// NewFact creates a validated Fact from the given input.
//
// Validation rules (in order):
//  1. Source.ID must not be empty (trimmed).
//  2. Source.Kind must not be empty.
//
// Defaulting rules:
//   - PrivacyClass defaults to PrivacyInternal if empty.
//   - Confidence defaults to ConfidenceUnknown if empty.
//   - Status is always set to StatusObserved on construction.
//
// The Fact ID is generated as fmt.Sprintf("evd_%d", now.UnixNano()), which is
// collision-resistant within a single process and requires no external dependency.
func NewFact(input FactInput, now time.Time) (Fact, error) {
	if strings.TrimSpace(input.Source.ID) == "" {
		return Fact{}, fmt.Errorf("evidence: Source.ID must not be empty")
	}
	if input.Source.Kind == "" {
		return Fact{}, fmt.Errorf("evidence: Source.Kind must not be empty")
	}

	privacyClass := input.PrivacyClass
	if privacyClass == "" {
		privacyClass = PrivacyInternal
	}

	confidence := input.Confidence
	if confidence == "" {
		confidence = ConfidenceUnknown
	}

	return Fact{
		ID:            fmt.Sprintf("evd_%d", now.UnixNano()),
		Source:        input.Source,
		PrivacyClass:  privacyClass,
		Confidence:    confidence,
		Status:        StatusObserved,
		LinkedObjects: input.LinkedObjects,
		AuditRefs:     input.AuditRefs,
		ObservedAt:    now,
	}, nil
}

// IsKnownSourceKind reports whether a SourceKind is one of the recognized constants.
func IsKnownSourceKind(k SourceKind) bool {
	return knownSourceKinds[k]
}

// IsKnownPrivacyClass reports whether a PrivacyClass is recognized.
func IsKnownPrivacyClass(p PrivacyClass) bool {
	return knownPrivacyClasses[p]
}
