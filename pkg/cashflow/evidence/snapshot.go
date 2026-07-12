package evidence

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Snapshot is a versioned, content-addressed capture of a CommandCenter.
type Snapshot struct {
	Version     int           `json:"version"`
	GeneratedAt time.Time     `json:"generated_at"`
	ContentHash string        `json:"content_hash"`
	Window      TimeWindow    `json:"window"`
	Center      CommandCenter `json:"center"`
}

// NewSnapshot creates a versioned snapshot with a deterministic content hash.
// The hash is computed from the JSON-serialized CommandCenter, so two identical
// centers always produce the same hash regardless of when they're snapshotted.
func NewSnapshot(center CommandCenter, version int, generatedAt time.Time) Snapshot {
	b, _ := json.Marshal(center)
	sum := sha256.Sum256(b)
	hash := fmt.Sprintf("%x", sum[:])[:16]
	return Snapshot{
		Version:     version,
		GeneratedAt: generatedAt,
		ContentHash: hash,
		Window:      center.Window,
		Center:      center,
	}
}

// SnapshotDiff describes what changed between two snapshots.
type SnapshotDiff struct {
	FromVersion   int    `json:"from_version"`
	ToVersion     int    `json:"to_version"`
	HashChanged   bool   `json:"hash_changed"`
	StatusChanged bool   `json:"status_changed"`
	OldStatus     Status `json:"old_status,omitempty"`
	NewStatus     Status `json:"new_status,omitempty"`

	ProposalCountDelta   int `json:"proposal_count_delta"`
	FollowUpCountDelta   int `json:"follow_up_count_delta"`
	EvidenceSourcesDelta int `json:"evidence_sources_delta"`
	AllocationCountDelta int `json:"allocation_count_delta"`

	PostingReadinessChanged bool `json:"posting_readiness_changed"`
	CashExposureChanged     bool `json:"cash_exposure_changed"`
}

// Diff computes what changed between this snapshot and a previous one.
func (s Snapshot) Diff(previous Snapshot) SnapshotDiff {
	d := SnapshotDiff{
		FromVersion: previous.Version,
		ToVersion:   s.Version,
		HashChanged: s.ContentHash != previous.ContentHash,
	}

	if s.Center.OverallStatus != previous.Center.OverallStatus {
		d.StatusChanged = true
		d.OldStatus = previous.Center.OverallStatus
		d.NewStatus = s.Center.OverallStatus
	}

	d.ProposalCountDelta = len(s.Center.ActionProposals) - len(previous.Center.ActionProposals)
	d.FollowUpCountDelta = s.Center.OpenFollowUpTasks - previous.Center.OpenFollowUpTasks
	d.EvidenceSourcesDelta = len(s.Center.EvidenceSources) - len(previous.Center.EvidenceSources)
	d.AllocationCountDelta = s.Center.AllocationSummary.TotalAllocations - previous.Center.AllocationSummary.TotalAllocations

	d.PostingReadinessChanged = s.Center.Posting.Status != previous.Center.Posting.Status

	d.CashExposureChanged = s.Center.Cash.OpenAR != previous.Center.Cash.OpenAR ||
		s.Center.Cash.OverdueAR != previous.Center.Cash.OverdueAR

	return d
}
