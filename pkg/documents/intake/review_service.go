package intake

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	ReviewActorOperator = "operator"
	ReviewActorAgent    = "agent"
)

type ReviewService struct {
	repo ReviewRecordRepository
	now  func() time.Time
}

type ReviewCommand struct {
	Candidate                    Candidate
	Decision                     ReviewDecision
	Actor                        string
	ActorType                    string
	Reason                       string
	ProposedDeterministicService string
	CorrelationID                string
}

type ReviewQueueState struct {
	Candidate    Candidate
	Records      []ReviewRecord
	LastReview   *ReviewRecord
	ContextPack  ContextPack
	SourceAssets []SourceAsset
}

func NewReviewService(repo ReviewRecordRepository, opts ...Options) (*ReviewService, error) {
	if repo == nil {
		return nil, fmt.Errorf("review record repository is required")
	}
	now := time.Now
	if len(opts) > 0 && opts[0].Now != nil {
		now = opts[0].Now
	}
	return &ReviewService{repo: repo, now: now}, nil
}

func (s *ReviewService) RecordDecision(ctx context.Context, command ReviewCommand) (ReviewRecord, error) {
	if s == nil || s.repo == nil {
		return ReviewRecord{}, fmt.Errorf("review service repository is required")
	}
	if err := validateReviewCommand(command); err != nil {
		return ReviewRecord{}, err
	}
	record, err := NewReviewRecord(ReviewRecordInput{
		Candidate:                    command.Candidate,
		Decision:                     command.Decision,
		Actor:                        strings.TrimSpace(command.Actor),
		Reason:                       command.Reason,
		ProposedDeterministicService: command.ProposedDeterministicService,
		CorrelationID:                command.CorrelationID,
		Now:                          s.now().UTC(),
	})
	if err != nil {
		return ReviewRecord{}, err
	}
	return s.repo.Save(ctx, record)
}

func (s *ReviewService) GetReviewRecord(ctx context.Context, id string) (ReviewRecord, bool, error) {
	if s == nil || s.repo == nil {
		return ReviewRecord{}, false, fmt.Errorf("review service repository is required")
	}
	return s.repo.Get(ctx, id)
}

func (s *ReviewService) ListReviewRecords(ctx context.Context, candidateID string) ([]ReviewRecord, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("review service repository is required")
	}
	return s.repo.ListByCandidate(ctx, candidateID)
}

func (s *ReviewService) BuildQueueState(ctx context.Context, candidate Candidate) (ReviewQueueState, error) {
	if s == nil || s.repo == nil {
		return ReviewQueueState{}, fmt.Errorf("review service repository is required")
	}
	candidate = normalizeCandidate(candidate, Options{Now: s.now})
	records, err := s.repo.ListByCandidate(ctx, candidate.ID)
	if err != nil {
		return ReviewQueueState{}, err
	}
	state := ReviewQueueState{
		Candidate:   candidate,
		Records:     records,
		ContextPack: BuildContextPack(candidate),
	}
	if len(records) > 0 {
		last := records[len(records)-1]
		state.LastReview = &last
	}
	return state, nil
}

func (s *ReviewService) BuildContextPack(candidate Candidate) ContextPack {
	return BuildContextPack(candidate)
}

func (s *ReviewService) BuildQueueStateWithSources(ctx context.Context, candidate Candidate, sourceRepo SourceAssetRepository) (ReviewQueueState, error) {
	state, err := s.BuildQueueState(ctx, candidate)
	if err != nil {
		return ReviewQueueState{}, err
	}
	if sourceRepo == nil {
		return state, nil
	}
	assets, err := sourceRepo.ListByCandidate(ctx, state.Candidate.ID)
	if err != nil {
		return ReviewQueueState{}, err
	}
	if len(assets) == 0 && strings.TrimSpace(state.Candidate.Source.ID) != "" {
		if asset, ok, err := sourceRepo.Get(ctx, state.Candidate.Source.ID); err != nil {
			return ReviewQueueState{}, err
		} else if ok {
			assets = append(assets, asset)
		}
	}
	state.SourceAssets = sortSourceAssets(assets)
	return state, nil
}

func validateReviewCommand(command ReviewCommand) error {
	if strings.TrimSpace(command.Actor) == "" {
		return fmt.Errorf("review actor is required")
	}
	if strings.TrimSpace(command.CorrelationID) == "" {
		return fmt.Errorf("correlation id is required")
	}
	if strings.TrimSpace(command.Candidate.ID) == "" {
		return fmt.Errorf("candidate id is required")
	}
	if strings.TrimSpace(command.Candidate.Source.ID) == "" {
		return fmt.Errorf("candidate source id is required")
	}
	if _, err := ReviewStatusForDecision(command.Decision); err != nil {
		return err
	}
	if normalizeActorType(command.ActorType) == ReviewActorAgent {
		return fmt.Errorf("agent actors may inspect, explain, draft, recommend, and assemble context only; review decisions require deterministic operator authority")
	}
	return nil
}

func normalizeActorType(actorType string) string {
	actorType = strings.TrimSpace(strings.ToLower(actorType))
	if actorType == "" {
		return ReviewActorOperator
	}
	return actorType
}
