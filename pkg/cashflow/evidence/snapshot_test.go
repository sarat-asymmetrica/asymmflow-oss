package evidence

import (
	"testing"
	"time"
)

func minimalCenter() CommandCenter {
	return CommandCenter{
		OverallStatus: StatusReady,
		Window: TimeWindow{
			Label: "test",
			Start: time.Now(),
			End:   time.Now().Add(30 * 24 * time.Hour),
		},
		AllocationSummary: AllocationSummary{},
		Posting:           PostingReadiness{Status: StatusReady},
	}
}

func TestNewSnapshotHashIsDeterministic(t *testing.T) {
	center := minimalCenter()
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)

	s1 := NewSnapshot(center, 1, t1)
	s2 := NewSnapshot(center, 1, t2)

	if s1.ContentHash != s2.ContentHash {
		t.Errorf("expected same hash for identical center regardless of GeneratedAt, got %q vs %q", s1.ContentHash, s2.ContentHash)
	}
}

func TestNewSnapshotHashChangesWithContent(t *testing.T) {
	base := minimalCenter()
	modified := minimalCenter()
	modified.OverallStatus = StatusAttention

	now := time.Now()
	s1 := NewSnapshot(base, 1, now)
	s2 := NewSnapshot(modified, 1, now)

	if s1.ContentHash == s2.ContentHash {
		t.Error("expected different hashes for centers with different OverallStatus")
	}

	modified2 := minimalCenter()
	modified2.OpenFollowUpTasks = 5
	s3 := NewSnapshot(modified2, 1, now)
	if s1.ContentHash == s3.ContentHash {
		t.Error("expected different hashes for centers with different OpenFollowUpTasks")
	}
}

func TestSnapshotDiffIdenticalSnapshots(t *testing.T) {
	center := minimalCenter()
	now := time.Now()
	s1 := NewSnapshot(center, 1, now)
	s2 := NewSnapshot(center, 2, now)

	d := s2.Diff(s1)

	if d.HashChanged {
		t.Error("expected HashChanged=false for identical centers")
	}
	if d.StatusChanged {
		t.Error("expected StatusChanged=false for identical centers")
	}
	if d.ProposalCountDelta != 0 {
		t.Errorf("expected ProposalCountDelta=0, got %d", d.ProposalCountDelta)
	}
	if d.FollowUpCountDelta != 0 {
		t.Errorf("expected FollowUpCountDelta=0, got %d", d.FollowUpCountDelta)
	}
	if d.EvidenceSourcesDelta != 0 {
		t.Errorf("expected EvidenceSourcesDelta=0, got %d", d.EvidenceSourcesDelta)
	}
	if d.AllocationCountDelta != 0 {
		t.Errorf("expected AllocationCountDelta=0, got %d", d.AllocationCountDelta)
	}
	if d.PostingReadinessChanged {
		t.Error("expected PostingReadinessChanged=false for identical centers")
	}
	if d.CashExposureChanged {
		t.Error("expected CashExposureChanged=false for identical centers")
	}
}

func TestSnapshotDiffDetectsStatusChange(t *testing.T) {
	old := minimalCenter()
	old.OverallStatus = StatusReady

	newer := minimalCenter()
	newer.OverallStatus = StatusAttention

	now := time.Now()
	s1 := NewSnapshot(old, 1, now)
	s2 := NewSnapshot(newer, 2, now)

	d := s2.Diff(s1)

	if !d.StatusChanged {
		t.Error("expected StatusChanged=true")
	}
	if d.OldStatus != StatusReady {
		t.Errorf("expected OldStatus=%q, got %q", StatusReady, d.OldStatus)
	}
	if d.NewStatus != StatusAttention {
		t.Errorf("expected NewStatus=%q, got %q", StatusAttention, d.NewStatus)
	}
	if !d.HashChanged {
		t.Error("expected HashChanged=true when status differs")
	}
}

func TestSnapshotDiffDetectsProposalCountChange(t *testing.T) {
	old := minimalCenter()
	newer := minimalCenter()
	newer.ActionProposals = []ActionProposal{
		{Action: "test.action", Label: "Test", Priority: PriorityLow},
	}

	now := time.Now()
	s1 := NewSnapshot(old, 1, now)
	s2 := NewSnapshot(newer, 2, now)

	d := s2.Diff(s1)

	if d.ProposalCountDelta != 1 {
		t.Errorf("expected ProposalCountDelta=1, got %d", d.ProposalCountDelta)
	}
}

func TestSnapshotDiffDetectsPostingReadinessChange(t *testing.T) {
	old := minimalCenter()
	old.Posting = PostingReadiness{Status: StatusReady}

	newer := minimalCenter()
	newer.Posting = PostingReadiness{Status: StatusAttention}

	now := time.Now()
	s1 := NewSnapshot(old, 1, now)
	s2 := NewSnapshot(newer, 2, now)

	d := s2.Diff(s1)

	if !d.PostingReadinessChanged {
		t.Error("expected PostingReadinessChanged=true")
	}
}

func TestSnapshotDiffDetectsCashExposureChange(t *testing.T) {
	old := minimalCenter()
	old.Cash = CashExposure{OpenAR: 1000.0, OverdueAR: 0.0}

	newer := minimalCenter()
	newer.Cash = CashExposure{OpenAR: 2500.0, OverdueAR: 0.0}

	now := time.Now()
	s1 := NewSnapshot(old, 1, now)
	s2 := NewSnapshot(newer, 2, now)

	d := s2.Diff(s1)

	if !d.CashExposureChanged {
		t.Error("expected CashExposureChanged=true when OpenAR differs")
	}
}

func TestSnapshotVersionsArePreserved(t *testing.T) {
	center := minimalCenter()
	now := time.Now()
	s1 := NewSnapshot(center, 7, now)
	s2 := NewSnapshot(center, 42, now)

	d := s2.Diff(s1)

	if d.FromVersion != 7 {
		t.Errorf("expected FromVersion=7, got %d", d.FromVersion)
	}
	if d.ToVersion != 42 {
		t.Errorf("expected ToVersion=42, got %d", d.ToVersion)
	}
}
