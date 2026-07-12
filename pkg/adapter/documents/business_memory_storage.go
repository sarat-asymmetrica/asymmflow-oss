package documents

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ph_holdings_app/pkg/documents/intake"

	"gorm.io/gorm"
)

type BusinessMemoryReviewRecordModel struct {
	ID                           string    `gorm:"primaryKey;size:128" json:"id"`
	CandidateID                  string    `gorm:"size:128;index;uniqueIndex:uix_business_memory_review_idempotency,priority:1" json:"candidate_id"`
	SourceID                     string    `gorm:"size:128;index" json:"source_id"`
	Decision                     string    `gorm:"size:48;uniqueIndex:uix_business_memory_review_idempotency,priority:2" json:"decision"`
	ReviewStatus                 string    `gorm:"size:48;index" json:"review_status"`
	ProposedDeterministicService string    `gorm:"size:160;uniqueIndex:uix_business_memory_review_idempotency,priority:3" json:"proposed_deterministic_service"`
	Actor                        string    `gorm:"size:128;index" json:"actor"`
	Reason                       string    `gorm:"size:1000" json:"reason"`
	CorrelationID                string    `gorm:"size:128;index;uniqueIndex:uix_business_memory_review_idempotency,priority:4" json:"correlation_id"`
	CreatedAt                    time.Time `gorm:"index" json:"created_at"`
}

func (BusinessMemoryReviewRecordModel) TableName() string {
	return "business_memory_review_records"
}

type GORMBusinessMemoryReviewRepository struct {
	db *gorm.DB
}

func NewGORMBusinessMemoryReviewRepository(db *gorm.DB) *GORMBusinessMemoryReviewRepository {
	return &GORMBusinessMemoryReviewRepository{db: db}
}

func (r *GORMBusinessMemoryReviewRepository) Migrate(ctx context.Context) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("business memory review repository database is required")
	}
	return r.db.WithContext(ctx).AutoMigrate(&BusinessMemoryReviewRecordModel{})
}

func (r *GORMBusinessMemoryReviewRepository) Save(ctx context.Context, record intake.ReviewRecord) (intake.ReviewRecord, error) {
	if r == nil || r.db == nil {
		return intake.ReviewRecord{}, fmt.Errorf("business memory review repository database is required")
	}
	if err := intake.ValidateReviewRecord(record); err != nil {
		return intake.ReviewRecord{}, err
	}

	var existing BusinessMemoryReviewRecordModel
	query := r.db.WithContext(ctx).
		Where("candidate_id = ? AND decision = ? AND proposed_deterministic_service = ? AND correlation_id = ?",
			record.CandidateID,
			string(record.Decision),
			record.ProposedDeterministicService,
			record.CorrelationID,
		).
		First(&existing)
	if query.Error == nil {
		return businessMemoryReviewRecordFromModel(existing), nil
	}
	if !errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return intake.ReviewRecord{}, query.Error
	}

	model := businessMemoryReviewRecordToModel(record)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			if err := r.db.WithContext(ctx).
				Where("candidate_id = ? AND decision = ? AND proposed_deterministic_service = ? AND correlation_id = ?",
					record.CandidateID,
					string(record.Decision),
					record.ProposedDeterministicService,
					record.CorrelationID,
				).
				First(&existing).Error; err == nil {
				return businessMemoryReviewRecordFromModel(existing), nil
			}
		}
		return intake.ReviewRecord{}, err
	}
	return businessMemoryReviewRecordFromModel(model), nil
}

func (r *GORMBusinessMemoryReviewRepository) Get(ctx context.Context, id string) (intake.ReviewRecord, bool, error) {
	if r == nil || r.db == nil {
		return intake.ReviewRecord{}, false, fmt.Errorf("business memory review repository database is required")
	}
	var model BusinessMemoryReviewRecordModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return intake.ReviewRecord{}, false, nil
	}
	if err != nil {
		return intake.ReviewRecord{}, false, err
	}
	return businessMemoryReviewRecordFromModel(model), true, nil
}

func (r *GORMBusinessMemoryReviewRepository) ListByCandidate(ctx context.Context, candidateID string) ([]intake.ReviewRecord, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("business memory review repository database is required")
	}
	query := r.db.WithContext(ctx).Order("created_at ASC")
	if candidateID != "" {
		query = query.Where("candidate_id = ?", candidateID)
	}
	var models []BusinessMemoryReviewRecordModel
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	records := make([]intake.ReviewRecord, 0, len(models))
	for _, model := range models {
		records = append(records, businessMemoryReviewRecordFromModel(model))
	}
	return records, nil
}

func businessMemoryReviewRecordToModel(record intake.ReviewRecord) BusinessMemoryReviewRecordModel {
	return BusinessMemoryReviewRecordModel{
		ID:                           record.ID,
		CandidateID:                  record.CandidateID,
		SourceID:                     record.SourceID,
		Decision:                     string(record.Decision),
		ReviewStatus:                 string(record.ReviewStatus),
		ProposedDeterministicService: record.ProposedDeterministicService,
		Actor:                        record.Actor,
		Reason:                       record.Reason,
		CorrelationID:                record.CorrelationID,
		CreatedAt:                    record.CreatedAt,
	}
}

func businessMemoryReviewRecordFromModel(model BusinessMemoryReviewRecordModel) intake.ReviewRecord {
	return intake.ReviewRecord{
		ID:                           model.ID,
		CandidateID:                  model.CandidateID,
		SourceID:                     model.SourceID,
		Decision:                     intake.ReviewDecision(model.Decision),
		ReviewStatus:                 intake.ReviewStatus(model.ReviewStatus),
		ProposedDeterministicService: model.ProposedDeterministicService,
		Actor:                        model.Actor,
		Reason:                       model.Reason,
		CorrelationID:                model.CorrelationID,
		CreatedAt:                    model.CreatedAt,
	}
}
