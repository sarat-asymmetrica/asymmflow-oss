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

func TestGORMBusinessMemoryReviewRepositoryPersistsAndIsIdempotent(t *testing.T) {
	db := openBusinessMemoryReviewTestDB(t)
	ctx := context.Background()
	repo := NewGORMBusinessMemoryReviewRepository(db)
	require.NoError(t, repo.Migrate(ctx))

	record := intake.ReviewRecord{
		ID:                           "review-1",
		CandidateID:                  "candidate-1",
		SourceID:                     "source-1",
		Decision:                     intake.ReviewDecisionAcceptProposal,
		ReviewStatus:                 intake.ReviewStatusLinked,
		ProposedDeterministicService: "finance.invoice.review",
		Actor:                        "operator",
		Reason:                       "verified",
		CorrelationID:                "corr-1",
		CreatedAt:                    time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC),
	}

	saved, err := repo.Save(ctx, record)
	require.NoError(t, err)
	duplicate := record
	duplicate.ID = "review-duplicate"
	duplicate.Reason = "duplicate write"
	savedAgain, err := repo.Save(ctx, duplicate)
	require.NoError(t, err)
	assert.Equal(t, saved.ID, savedAgain.ID)
	assert.Equal(t, saved.Reason, savedAgain.Reason)

	got, ok, err := repo.Get(ctx, "review-1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, saved, got)

	list, err := repo.ListByCandidate(ctx, "candidate-1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "review-1", list[0].ID)
}

func TestGORMBusinessMemoryReviewRepositoryRejectsInvalidDecision(t *testing.T) {
	db := openBusinessMemoryReviewTestDB(t)
	ctx := context.Background()
	repo := NewGORMBusinessMemoryReviewRepository(db)
	require.NoError(t, repo.Migrate(ctx))

	_, err := repo.Save(ctx, intake.ReviewRecord{
		ID:            "review-1",
		CandidateID:   "candidate-1",
		Decision:      intake.ReviewDecision("post_without_operator"),
		ReviewStatus:  intake.ReviewStatusLinked,
		Actor:         "operator",
		CorrelationID: "corr-1",
	})
	require.Error(t, err)
}

func openBusinessMemoryReviewTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:business-memory-review-test?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db
}
