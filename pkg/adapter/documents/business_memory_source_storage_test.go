package documents

import (
	"context"
	"testing"
	"time"

	"ph_holdings_app/pkg/documents/intake"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGORMBusinessMemorySourceAssetRepositoryPersistsAndMergesDuplicates(t *testing.T) {
	db := openBusinessMemorySourceAssetTestDB(t)
	ctx := context.Background()
	repo := NewGORMBusinessMemorySourceAssetRepository(db)
	require.NoError(t, repo.Migrate(ctx))
	firstSeen := time.Date(2026, 5, 15, 9, 0, 0, 0, time.UTC)
	lastSeen := firstSeen.Add(90 * time.Minute)

	first := mustBusinessMemorySourceAsset(t, intake.SourceAssetInput{
		Kind:             intake.SourceKindPDF,
		Path:             "C:\\Inbox\\invoice.pdf",
		Label:            "Supplier invoice",
		Hash:             "sha256:invoice",
		ImportBatchID:    "batch-1",
		PrivacyClass:     intake.SourcePrivacyInternal,
		ProcessingStatus: intake.SourceStatusNormalized,
		CandidateIDs:     []string{"candidate-1"},
		AuditRefs:        []intake.AuditRef{{Type: "inbox", SourceID: "invoice.pdf", Summary: "first import"}},
		SeenAt:           firstSeen,
	})
	second := mustBusinessMemorySourceAsset(t, intake.SourceAssetInput{
		Kind:             intake.SourceKindPDF,
		Path:             "D:\\Archive\\invoice-renamed.pdf",
		Label:            "Supplier invoice reviewed",
		Hash:             "sha256:invoice",
		ImportBatchID:    "batch-2",
		PrivacyClass:     intake.SourcePrivacyConfidential,
		ProcessingStatus: intake.SourceStatusReviewed,
		CandidateIDs:     []string{"candidate-1", "candidate-2"},
		AuditRefs:        []intake.AuditRef{{Type: "review", SourceID: "candidate-2", Summary: "operator reviewed"}},
		SeenAt:           lastSeen,
	})

	saved, duplicate, err := repo.Upsert(ctx, first)
	require.NoError(t, err)
	require.False(t, duplicate)
	merged, duplicate, err := repo.Upsert(ctx, second)
	require.NoError(t, err)
	require.True(t, duplicate)
	assert.Equal(t, saved.ID, merged.ID)
	assert.Equal(t, "batch-2", merged.ImportBatchID)
	assert.Equal(t, intake.SourcePrivacyConfidential, merged.PrivacyClass)
	assert.Equal(t, intake.SourceStatusReviewed, merged.ProcessingStatus)
	assert.ElementsMatch(t, []string{"candidate-1", "candidate-2"}, merged.CandidateIDs)
	assert.Len(t, merged.AuditRefs, 2)
	assert.True(t, merged.FirstSeenAt.Equal(firstSeen))
	assert.True(t, merged.LastSeenAt.Equal(lastSeen))

	got, ok, err := repo.Get(ctx, saved.ID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, merged.ID, got.ID)
	assert.ElementsMatch(t, merged.CandidateIDs, got.CandidateIDs)

	byCandidate, err := repo.ListByCandidate(ctx, "candidate-2")
	require.NoError(t, err)
	require.Len(t, byCandidate, 1)
	assert.Equal(t, merged.ID, byCandidate[0].ID)
}

func TestGORMBusinessMemorySourceAssetRepositoryListFilters(t *testing.T) {
	db := openBusinessMemorySourceAssetTestDB(t)
	ctx := context.Background()
	repo := NewGORMBusinessMemorySourceAssetRepository(db)
	require.NoError(t, repo.Migrate(ctx))
	pdf := mustBusinessMemorySourceAsset(t, intake.SourceAssetInput{
		Kind:             intake.SourceKindPDF,
		Path:             "C:\\Inbox\\invoice.pdf",
		Label:            "invoice",
		PrivacyClass:     intake.SourcePrivacyRestricted,
		ProcessingStatus: intake.SourceStatusReviewed,
		CandidateIDs:     []string{"candidate-1"},
		SeenAt:           time.Date(2026, 5, 15, 9, 0, 0, 0, time.UTC),
	})
	email := mustBusinessMemorySourceAsset(t, intake.SourceAssetInput{
		Kind:             intake.SourceKindEmail,
		Path:             "C:\\Inbox\\rfq.eml",
		Label:            "rfq",
		PrivacyClass:     intake.SourcePrivacyInternal,
		ProcessingStatus: intake.SourceStatusDiscovered,
		CandidateIDs:     []string{"candidate-2"},
		SeenAt:           time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
	})
	_, _, err := repo.Upsert(ctx, email)
	require.NoError(t, err)
	_, _, err = repo.Upsert(ctx, pdf)
	require.NoError(t, err)

	filtered, err := repo.List(ctx, intake.SourceAssetListFilter{
		Kind:             intake.SourceKindPDF,
		ProcessingStatus: intake.SourceStatusReviewed,
		PrivacyClass:     intake.SourcePrivacyRestricted,
		CandidateID:      "candidate-1",
	})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, pdf.ID, filtered[0].ID)
}

func TestGORMBusinessMemorySourceAssetRepositoryRejectsInvalidInputs(t *testing.T) {
	db := openBusinessMemorySourceAssetTestDB(t)
	ctx := context.Background()
	repo := NewGORMBusinessMemorySourceAssetRepository(db)
	require.NoError(t, repo.Migrate(ctx))

	_, _, err := repo.Upsert(ctx, intake.SourceAsset{
		ID:               "source-1",
		Kind:             intake.SourceKindFolder,
		Label:            "Inbox folder",
		PrivacyClass:     intake.SourcePrivacyInternal,
		ProcessingStatus: intake.SourceStatusDiscovered,
		FirstSeenAt:      time.Date(2026, 5, 15, 9, 0, 0, 0, time.UTC),
		LastSeenAt:       time.Date(2026, 5, 15, 8, 0, 0, 0, time.UTC),
	})
	require.Error(t, err)
}

func openBusinessMemorySourceAssetTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:business-memory-source-asset-test?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db
}

func mustBusinessMemorySourceAsset(t *testing.T, input intake.SourceAssetInput) intake.SourceAsset {
	t.Helper()
	asset, err := intake.NewSourceAsset(input)
	require.NoError(t, err)
	return asset
}
