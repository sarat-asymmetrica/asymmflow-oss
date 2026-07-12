package intake

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"ph_holdings_app/pkg/kernel/text"
)

type ReviewDecision string

const (
	ReviewDecisionAcceptProposal  ReviewDecision = "accept_proposal"
	ReviewDecisionNeedsInput      ReviewDecision = "needs_input"
	ReviewDecisionCorrectField    ReviewDecision = "correct_field"
	ReviewDecisionRejectCandidate ReviewDecision = "reject_candidate"
	ReviewDecisionArchive         ReviewDecision = "archive"
)

type ReviewRecord struct {
	ID                           string         `json:"id"`
	CandidateID                  string         `json:"candidate_id"`
	SourceID                     string         `json:"source_id"`
	Decision                     ReviewDecision `json:"decision"`
	ReviewStatus                 ReviewStatus   `json:"review_status"`
	ProposedDeterministicService string         `json:"proposed_deterministic_service,omitempty"`
	Actor                        string         `json:"actor"`
	Reason                       string         `json:"reason,omitempty"`
	CorrelationID                string         `json:"correlation_id"`
	CreatedAt                    time.Time      `json:"created_at"`
}

type ReviewRecordInput struct {
	Candidate                    Candidate
	Decision                     ReviewDecision
	Actor                        string
	Reason                       string
	ProposedDeterministicService string
	CorrelationID                string
	Now                          time.Time
}

type ReviewQueue struct {
	mu      sync.RWMutex
	records map[string]ReviewRecord
}

func NewReviewQueue() *ReviewQueue {
	return &ReviewQueue{records: map[string]ReviewRecord{}}
}

func NewReviewRecord(input ReviewRecordInput) (ReviewRecord, error) {
	candidate := normalizeCandidate(input.Candidate, Options{})
	decision := input.Decision
	if decision == "" {
		return ReviewRecord{}, fmt.Errorf("review decision is required")
	}
	status, err := ReviewStatusForDecision(decision)
	if err != nil {
		return ReviewRecord{}, err
	}
	if strings.TrimSpace(candidate.ID) == "" {
		return ReviewRecord{}, fmt.Errorf("candidate id is required")
	}
	actor := strings.TrimSpace(input.Actor)
	if actor == "" {
		actor = "operator"
	}
	createdAt := input.Now
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	service := text.FirstNonEmpty(input.ProposedDeterministicService, firstDeterministicService(candidate.SuggestedLinks))
	correlationID := text.FirstNonEmpty(input.CorrelationID, buildID("intake-review", candidate.ID, string(decision), service))
	return ReviewRecord{
		ID:                           buildID("review", candidate.ID, string(decision), service),
		CandidateID:                  candidate.ID,
		SourceID:                     candidate.Source.ID,
		Decision:                     decision,
		ReviewStatus:                 status,
		ProposedDeterministicService: service,
		Actor:                        actor,
		Reason:                       strings.TrimSpace(input.Reason),
		CorrelationID:                correlationID,
		CreatedAt:                    createdAt,
	}, nil
}

func ReviewStatusForDecision(decision ReviewDecision) (ReviewStatus, error) {
	switch decision {
	case ReviewDecisionAcceptProposal:
		return ReviewStatusLinked, nil
	case ReviewDecisionNeedsInput:
		return ReviewStatusNeedsReview, nil
	case ReviewDecisionCorrectField:
		return ReviewStatusCorrected, nil
	case ReviewDecisionRejectCandidate:
		return ReviewStatusRejected, nil
	case ReviewDecisionArchive:
		return ReviewStatusArchived, nil
	default:
		return "", fmt.Errorf("unsupported review decision %q", decision)
	}
}

func (q *ReviewQueue) Save(record ReviewRecord) ReviewRecord {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.records == nil {
		q.records = map[string]ReviewRecord{}
	}
	key := reviewRecordKey(record)
	if existing, ok := q.records[key]; ok {
		return existing
	}
	q.records[key] = record
	return record
}

func (q *ReviewQueue) Get(id string) (ReviewRecord, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	for _, record := range q.records {
		if record.ID == id {
			return record, true
		}
	}
	return ReviewRecord{}, false
}

func (q *ReviewQueue) ListByCandidate(candidateID string) []ReviewRecord {
	q.mu.RLock()
	defer q.mu.RUnlock()
	records := []ReviewRecord{}
	for _, record := range q.records {
		if candidateID == "" || record.CandidateID == candidateID {
			records = append(records, record)
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})
	return records
}

func reviewRecordKey(record ReviewRecord) string {
	return buildID(record.CandidateID, string(record.Decision), record.ProposedDeterministicService, record.CorrelationID)
}

func firstDeterministicService(links []SuggestedLink) string {
	for _, link := range links {
		if strings.TrimSpace(link.RequiredDeterministicService) != "" {
			return strings.TrimSpace(link.RequiredDeterministicService)
		}
	}
	return ""
}
