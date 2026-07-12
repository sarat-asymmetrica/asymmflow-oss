package intake

import (
	"testing"
	"time"
)

func TestSourceAssetStableIDUsesContentHashWhenAvailable(t *testing.T) {
	first := SourceAssetStableID(SourceAssetInput{
		Kind: SourceKindPDF,
		Path: "C:\\Inbox\\invoice.pdf",
		Hash: "sha256:abc",
	})
	second := SourceAssetStableID(SourceAssetInput{
		Kind: SourceKindPDF,
		Path: "D:\\Archive\\renamed.pdf",
		Hash: "sha256:abc",
	})
	if first != second {
		t.Fatalf("hash-backed source ids should match: %q != %q", first, second)
	}
}

func TestSourceAssetRegistryDetectsDuplicateAndMergesEvidence(t *testing.T) {
	registry := NewSourceAssetRegistry()
	firstSeen := time.Date(2026, 5, 14, 9, 0, 0, 0, time.UTC)
	lastSeen := firstSeen.Add(time.Hour)

	created, duplicate, err := registry.Upsert(SourceAssetInput{
		Kind:             SourceKindEmail,
		Path:             "C:\\Inbox\\rfq.eml",
		Label:            "RFQ email",
		PrivacyClass:     SourcePrivacyConfidential,
		ProcessingStatus: SourceStatusNormalized,
		CandidateIDs:     []string{"candidate-1"},
		AuditRefs:        []AuditRef{{Type: "inbox", SourceID: "rfq.eml", Summary: "first import"}},
		SeenAt:           firstSeen,
	})
	if err != nil {
		t.Fatalf("first upsert returned error: %v", err)
	}
	if duplicate {
		t.Fatal("first upsert should not be duplicate")
	}

	merged, duplicate, err := registry.Upsert(SourceAssetInput{
		Kind:             SourceKindEmail,
		Path:             "C:\\Inbox\\rfq.eml",
		Label:            "RFQ email duplicate",
		ImportBatchID:    "batch-2",
		ProcessingStatus: SourceStatusCandidateGenerated,
		CandidateIDs:     []string{"candidate-1", "candidate-2"},
		AuditRefs:        []AuditRef{{Type: "candidate", SourceID: "candidate-2", Summary: "second candidate"}},
		SeenAt:           lastSeen,
	})
	if err != nil {
		t.Fatalf("second upsert returned error: %v", err)
	}
	if !duplicate {
		t.Fatal("second upsert should detect duplicate source")
	}
	if merged.ID != created.ID {
		t.Fatalf("merged id = %q, want %q", merged.ID, created.ID)
	}
	if merged.ImportBatchID != "batch-2" {
		t.Fatalf("import batch = %q", merged.ImportBatchID)
	}
	if merged.ProcessingStatus != SourceStatusCandidateGenerated {
		t.Fatalf("status = %q", merged.ProcessingStatus)
	}
	if len(merged.CandidateIDs) != 2 {
		t.Fatalf("candidate ids = %+v", merged.CandidateIDs)
	}
	if len(merged.AuditRefs) != 2 {
		t.Fatalf("audit refs = %+v", merged.AuditRefs)
	}
	if !merged.FirstSeenAt.Equal(firstSeen) || !merged.LastSeenAt.Equal(lastSeen) {
		t.Fatalf("seen range = %s to %s", merged.FirstSeenAt, merged.LastSeenAt)
	}
}

func TestNewSourceAssetDefaultsPrivacyAndStatus(t *testing.T) {
	asset, err := NewSourceAsset(SourceAssetInput{
		Path:  "C:\\Inbox\\statement.xlsx",
		Label: "statement",
	}, Options{Now: func() time.Time {
		return time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	}})
	if err != nil {
		t.Fatalf("NewSourceAsset returned error: %v", err)
	}
	if asset.Kind != SourceKindExcel {
		t.Fatalf("kind = %q", asset.Kind)
	}
	if asset.PrivacyClass != SourcePrivacyInternal {
		t.Fatalf("privacy = %q", asset.PrivacyClass)
	}
	if asset.ProcessingStatus != SourceStatusDiscovered {
		t.Fatalf("status = %q", asset.ProcessingStatus)
	}
}
