package intake

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type ReviewRecordRepository interface {
	Save(ctx context.Context, record ReviewRecord) (ReviewRecord, error)
	Get(ctx context.Context, id string) (ReviewRecord, bool, error)
	ListByCandidate(ctx context.Context, candidateID string) ([]ReviewRecord, error)
}

type MemoryReviewRecordRepository struct {
	mu      sync.RWMutex
	records map[string]ReviewRecord
}

func NewMemoryReviewRecordRepository() *MemoryReviewRecordRepository {
	return &MemoryReviewRecordRepository{records: map[string]ReviewRecord{}}
}

func (r *MemoryReviewRecordRepository) Save(_ context.Context, record ReviewRecord) (ReviewRecord, error) {
	if err := ValidateReviewRecord(record); err != nil {
		return ReviewRecord{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.records == nil {
		r.records = map[string]ReviewRecord{}
	}
	key := ReviewRecordIdempotencyKey(record)
	if existing, ok := r.records[key]; ok {
		return existing, nil
	}
	r.records[key] = record
	return record, nil
}

func (r *MemoryReviewRecordRepository) Get(_ context.Context, id string) (ReviewRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, record := range r.records {
		if record.ID == id {
			return record, true, nil
		}
	}
	return ReviewRecord{}, false, nil
}

func (r *MemoryReviewRecordRepository) ListByCandidate(_ context.Context, candidateID string) ([]ReviewRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	records := make([]ReviewRecord, 0, len(r.records))
	for _, record := range r.records {
		if candidateID == "" || record.CandidateID == candidateID {
			records = append(records, record)
		}
	}
	return sortReviewRecords(records), nil
}

func ValidateReviewRecord(record ReviewRecord) error {
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("review record id is required")
	}
	if strings.TrimSpace(record.CandidateID) == "" {
		return fmt.Errorf("candidate id is required")
	}
	if strings.TrimSpace(record.Actor) == "" {
		return fmt.Errorf("review actor is required")
	}
	if strings.TrimSpace(record.CorrelationID) == "" {
		return fmt.Errorf("correlation id is required")
	}
	if _, err := ReviewStatusForDecision(record.Decision); err != nil {
		return err
	}
	if record.ReviewStatus == "" {
		return fmt.Errorf("review status is required")
	}
	return nil
}

func ReviewRecordIdempotencyKey(record ReviewRecord) string {
	return reviewRecordKey(record)
}

func sortReviewRecords(records []ReviewRecord) []ReviewRecord {
	out := append([]ReviewRecord(nil), records...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}
