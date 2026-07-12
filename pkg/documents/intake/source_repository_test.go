package intake

import (
	"context"
	"testing"
	"time"
)

func TestMemorySourceAssetRepositoryPersistsStableIDAndListsByCandidate(t *testing.T) {
	ctx := context.Background()
	repo := NewMemorySourceAssetRepository()
	seenAt := time.Date(2026, 5, 15, 9, 0, 0, 0, time.UTC)
	asset, err := NewSourceAsset(SourceAssetInput{
		Kind:             SourceKindPDF,
		Path:             "C:\\Inbox\\supplier-invoice.pdf",
		Label:            "Supplier invoice",
		Hash:             "sha256:stable",
		ProcessingStatus: SourceStatusCandidateGenerated,
		CandidateIDs:     []string{"candidate-1"},
		SeenAt:           seenAt,
	})
	if err != nil {
		t.Fatalf("NewSourceAsset returned error: %v", err)
	}

	saved, duplicate, err := repo.Upsert(ctx, asset)
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if duplicate {
		t.Fatal("first source upsert should not be a duplicate")
	}
	if saved.ID != SourceAssetStableID(SourceAssetInput{Kind: SourceKindPDF, Hash: "sha256:stable"}) {
		t.Fatalf("saved id = %q", saved.ID)
	}

	got, ok, err := repo.Get(ctx, saved.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !ok || got.ID != saved.ID {
		t.Fatalf("Get should find saved source, ok=%v source=%+v", ok, got)
	}

	byCandidate, err := repo.ListByCandidate(ctx, "candidate-1")
	if err != nil {
		t.Fatalf("ListByCandidate returned error: %v", err)
	}
	if len(byCandidate) != 1 || byCandidate[0].ID != saved.ID {
		t.Fatalf("candidate list = %+v", byCandidate)
	}
}

func TestMemorySourceAssetRepositoryMergesDuplicateEvidence(t *testing.T) {
	ctx := context.Background()
	repo := NewMemorySourceAssetRepository()
	firstSeen := time.Date(2026, 5, 15, 9, 0, 0, 0, time.UTC)
	lastSeen := firstSeen.Add(2 * time.Hour)

	first, err := NewSourceAsset(SourceAssetInput{
		Kind:             SourceKindEmail,
		Path:             "C:\\Inbox\\rfq.eml",
		Label:            "RFQ email",
		ImportBatchID:    "batch-1",
		PrivacyClass:     SourcePrivacyInternal,
		ProcessingStatus: SourceStatusNormalized,
		CandidateIDs:     []string{"candidate-1"},
		AuditRefs:        []AuditRef{{Type: "inbox", SourceID: "rfq.eml", Summary: "first import"}},
		SeenAt:           firstSeen,
	})
	if err != nil {
		t.Fatalf("first NewSourceAsset returned error: %v", err)
	}
	second, err := NewSourceAsset(SourceAssetInput{
		Kind:             SourceKindEmail,
		Path:             "C:\\Inbox\\rfq.eml",
		Label:            "RFQ email reviewed",
		ImportBatchID:    "batch-2",
		PrivacyClass:     SourcePrivacyConfidential,
		ProcessingStatus: SourceStatusReviewed,
		CandidateIDs:     []string{"candidate-1", "candidate-2"},
		AuditRefs:        []AuditRef{{Type: "review", SourceID: "candidate-2", Summary: "operator reviewed"}},
		SeenAt:           lastSeen,
	})
	if err != nil {
		t.Fatalf("second NewSourceAsset returned error: %v", err)
	}

	if _, _, err := repo.Upsert(ctx, first); err != nil {
		t.Fatalf("first Upsert returned error: %v", err)
	}
	merged, duplicate, err := repo.Upsert(ctx, second)
	if err != nil {
		t.Fatalf("second Upsert returned error: %v", err)
	}
	if !duplicate {
		t.Fatal("second upsert should detect duplicate source asset")
	}
	if merged.ID != first.ID {
		t.Fatalf("merged id = %q, want %q", merged.ID, first.ID)
	}
	if merged.ImportBatchID != "batch-2" {
		t.Fatalf("import batch = %q", merged.ImportBatchID)
	}
	if merged.PrivacyClass != SourcePrivacyConfidential {
		t.Fatalf("privacy = %q", merged.PrivacyClass)
	}
	if merged.ProcessingStatus != SourceStatusReviewed {
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

func TestMemorySourceAssetRepositoryListFilters(t *testing.T) {
	ctx := context.Background()
	repo := NewMemorySourceAssetRepository()
	pdf := mustSourceAsset(t, SourceAssetInput{
		Kind:             SourceKindPDF,
		Path:             "C:\\Inbox\\invoice.pdf",
		Label:            "invoice",
		PrivacyClass:     SourcePrivacyRestricted,
		ProcessingStatus: SourceStatusReviewed,
		CandidateIDs:     []string{"candidate-1"},
		SeenAt:           time.Date(2026, 5, 15, 9, 0, 0, 0, time.UTC),
	})
	email := mustSourceAsset(t, SourceAssetInput{
		Kind:             SourceKindEmail,
		Path:             "C:\\Inbox\\rfq.eml",
		Label:            "rfq",
		PrivacyClass:     SourcePrivacyInternal,
		ProcessingStatus: SourceStatusDiscovered,
		CandidateIDs:     []string{"candidate-2"},
		SeenAt:           time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
	})
	if _, _, err := repo.Upsert(ctx, email); err != nil {
		t.Fatalf("email Upsert returned error: %v", err)
	}
	if _, _, err := repo.Upsert(ctx, pdf); err != nil {
		t.Fatalf("pdf Upsert returned error: %v", err)
	}

	filtered, err := repo.List(ctx, SourceAssetListFilter{
		Kind:             SourceKindPDF,
		ProcessingStatus: SourceStatusReviewed,
		PrivacyClass:     SourcePrivacyRestricted,
		CandidateID:      "candidate-1",
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != pdf.ID {
		t.Fatalf("filtered sources = %+v", filtered)
	}
}

func TestValidateSourceAssetRejectsInvalidInputs(t *testing.T) {
	ctx := context.Background()
	repo := NewMemorySourceAssetRepository()
	valid := mustSourceAsset(t, SourceAssetInput{
		Kind:   SourceKindFolder,
		Path:   "C:\\Inbox",
		Label:  "Inbox folder",
		SeenAt: time.Date(2026, 5, 15, 9, 0, 0, 0, time.UTC),
	})

	tests := map[string]SourceAsset{
		"missing id":      withSourceAsset(valid, func(asset *SourceAsset) { asset.ID = "" }),
		"missing kind":    withSourceAsset(valid, func(asset *SourceAsset) { asset.Kind = "" }),
		"missing label":   withSourceAsset(valid, func(asset *SourceAsset) { asset.Label = "" }),
		"missing privacy": withSourceAsset(valid, func(asset *SourceAsset) { asset.PrivacyClass = "" }),
		"missing status":  withSourceAsset(valid, func(asset *SourceAsset) { asset.ProcessingStatus = "" }),
		"missing first":   withSourceAsset(valid, func(asset *SourceAsset) { asset.FirstSeenAt = time.Time{} }),
		"missing last":    withSourceAsset(valid, func(asset *SourceAsset) { asset.LastSeenAt = time.Time{} }),
		"last before first": withSourceAsset(valid, func(asset *SourceAsset) {
			asset.LastSeenAt = asset.FirstSeenAt.Add(-time.Minute)
		}),
	}

	for name, asset := range tests {
		t.Run(name, func(t *testing.T) {
			if _, _, err := repo.Upsert(ctx, asset); err == nil {
				t.Fatal("expected invalid source asset to be rejected")
			}
		})
	}
}

func mustSourceAsset(t *testing.T, input SourceAssetInput) SourceAsset {
	t.Helper()
	asset, err := NewSourceAsset(input)
	if err != nil {
		t.Fatalf("NewSourceAsset returned error: %v", err)
	}
	return asset
}

func withSourceAsset(asset SourceAsset, mutate func(*SourceAsset)) SourceAsset {
	mutate(&asset)
	return asset
}
