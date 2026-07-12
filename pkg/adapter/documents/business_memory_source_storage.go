package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ph_holdings_app/pkg/documents/intake"

	"gorm.io/gorm"
)

type BusinessMemorySourceAssetModel struct {
	ID               string    `gorm:"primaryKey;size:160" json:"id"`
	Kind             string    `gorm:"size:48;index" json:"kind"`
	Path             string    `gorm:"size:1000;index" json:"path"`
	Label            string    `gorm:"size:512" json:"label"`
	Hash             string    `gorm:"size:160;index" json:"hash"`
	ImportBatchID    string    `gorm:"size:128;index" json:"import_batch_id"`
	PrivacyClass     string    `gorm:"size:48;index" json:"privacy_class"`
	ProcessingStatus string    `gorm:"size:48;index" json:"processing_status"`
	CandidateIDsJSON string    `gorm:"type:text" json:"candidate_ids_json"`
	AuditRefsJSON    string    `gorm:"type:text" json:"audit_refs_json"`
	FirstSeenAt      time.Time `gorm:"index" json:"first_seen_at"`
	LastSeenAt       time.Time `gorm:"index" json:"last_seen_at"`
}

func (BusinessMemorySourceAssetModel) TableName() string {
	return "business_memory_source_assets"
}

type GORMBusinessMemorySourceAssetRepository struct {
	db *gorm.DB
}

func NewGORMBusinessMemorySourceAssetRepository(db *gorm.DB) *GORMBusinessMemorySourceAssetRepository {
	return &GORMBusinessMemorySourceAssetRepository{db: db}
}

func (r *GORMBusinessMemorySourceAssetRepository) Migrate(ctx context.Context) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("business memory source asset repository database is required")
	}
	return r.db.WithContext(ctx).AutoMigrate(&BusinessMemorySourceAssetModel{})
}

func (r *GORMBusinessMemorySourceAssetRepository) Upsert(ctx context.Context, asset intake.SourceAsset) (intake.SourceAsset, bool, error) {
	if r == nil || r.db == nil {
		return intake.SourceAsset{}, false, fmt.Errorf("business memory source asset repository database is required")
	}
	if err := intake.ValidateSourceAsset(asset); err != nil {
		return intake.SourceAsset{}, false, err
	}

	var existing BusinessMemorySourceAssetModel
	query := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(asset.ID)).First(&existing)
	duplicate := query.Error == nil
	if query.Error != nil && !errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return intake.SourceAsset{}, false, query.Error
	}
	if duplicate {
		current, err := businessMemorySourceAssetFromModel(existing)
		if err != nil {
			return intake.SourceAsset{}, false, err
		}
		asset = intake.MergeSourceAssets(current, asset)
	}

	model, err := businessMemorySourceAssetToModel(asset)
	if err != nil {
		return intake.SourceAsset{}, false, err
	}
	if duplicate {
		if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
			return intake.SourceAsset{}, false, err
		}
		saved, err := businessMemorySourceAssetFromModel(modelWithStableTimes(model))
		return saved, true, err
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return intake.SourceAsset{}, false, err
	}
	saved, err := businessMemorySourceAssetFromModel(modelWithStableTimes(model))
	return saved, false, err
}

func (r *GORMBusinessMemorySourceAssetRepository) Get(ctx context.Context, id string) (intake.SourceAsset, bool, error) {
	if r == nil || r.db == nil {
		return intake.SourceAsset{}, false, fmt.Errorf("business memory source asset repository database is required")
	}
	var model BusinessMemorySourceAssetModel
	err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return intake.SourceAsset{}, false, nil
	}
	if err != nil {
		return intake.SourceAsset{}, false, err
	}
	asset, err := businessMemorySourceAssetFromModel(model)
	if err != nil {
		return intake.SourceAsset{}, false, err
	}
	return asset, true, nil
}

func (r *GORMBusinessMemorySourceAssetRepository) List(ctx context.Context, filter intake.SourceAssetListFilter) ([]intake.SourceAsset, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("business memory source asset repository database is required")
	}
	query := r.db.WithContext(ctx).Order("first_seen_at ASC, id ASC")
	if filter.Kind != "" {
		query = query.Where("kind = ?", string(filter.Kind))
	}
	if filter.ProcessingStatus != "" {
		query = query.Where("processing_status = ?", string(filter.ProcessingStatus))
	}
	if filter.PrivacyClass != "" {
		query = query.Where("privacy_class = ?", string(filter.PrivacyClass))
	}

	var models []BusinessMemorySourceAssetModel
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	assets := make([]intake.SourceAsset, 0, len(models))
	for _, model := range models {
		asset, err := businessMemorySourceAssetFromModel(model)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(filter.CandidateID) != "" && !sourceAssetIncludesCandidate(asset, filter.CandidateID) {
			continue
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func (r *GORMBusinessMemorySourceAssetRepository) ListByCandidate(ctx context.Context, candidateID string) ([]intake.SourceAsset, error) {
	return r.List(ctx, intake.SourceAssetListFilter{CandidateID: candidateID})
}

func businessMemorySourceAssetToModel(asset intake.SourceAsset) (BusinessMemorySourceAssetModel, error) {
	if err := intake.ValidateSourceAsset(asset); err != nil {
		return BusinessMemorySourceAssetModel{}, err
	}
	candidateIDs, err := marshalSourceAssetJSON(asset.CandidateIDs)
	if err != nil {
		return BusinessMemorySourceAssetModel{}, err
	}
	auditRefs, err := marshalSourceAssetJSON(asset.AuditRefs)
	if err != nil {
		return BusinessMemorySourceAssetModel{}, err
	}
	return BusinessMemorySourceAssetModel{
		ID:               asset.ID,
		Kind:             string(asset.Kind),
		Path:             asset.Path,
		Label:            asset.Label,
		Hash:             asset.Hash,
		ImportBatchID:    asset.ImportBatchID,
		PrivacyClass:     string(asset.PrivacyClass),
		ProcessingStatus: string(asset.ProcessingStatus),
		CandidateIDsJSON: candidateIDs,
		AuditRefsJSON:    auditRefs,
		FirstSeenAt:      asset.FirstSeenAt,
		LastSeenAt:       asset.LastSeenAt,
	}, nil
}

func businessMemorySourceAssetFromModel(model BusinessMemorySourceAssetModel) (intake.SourceAsset, error) {
	var candidateIDs []string
	if err := unmarshalSourceAssetJSON(model.CandidateIDsJSON, &candidateIDs); err != nil {
		return intake.SourceAsset{}, err
	}
	var auditRefs []intake.AuditRef
	if err := unmarshalSourceAssetJSON(model.AuditRefsJSON, &auditRefs); err != nil {
		return intake.SourceAsset{}, err
	}
	asset := intake.SourceAsset{
		ID:               model.ID,
		Kind:             intake.SourceKind(model.Kind),
		Path:             model.Path,
		Label:            model.Label,
		Hash:             model.Hash,
		ImportBatchID:    model.ImportBatchID,
		PrivacyClass:     intake.SourcePrivacyClass(model.PrivacyClass),
		ProcessingStatus: intake.SourceProcessingStatus(model.ProcessingStatus),
		CandidateIDs:     candidateIDs,
		AuditRefs:        auditRefs,
		FirstSeenAt:      model.FirstSeenAt,
		LastSeenAt:       model.LastSeenAt,
	}
	if err := intake.ValidateSourceAsset(asset); err != nil {
		return intake.SourceAsset{}, err
	}
	return asset, nil
}

func marshalSourceAssetJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalSourceAssetJSON(payload string, out any) error {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		payload = "[]"
	}
	return json.Unmarshal([]byte(payload), out)
}

func sourceAssetIncludesCandidate(asset intake.SourceAsset, candidateID string) bool {
	candidateID = strings.TrimSpace(candidateID)
	for _, id := range asset.CandidateIDs {
		if strings.TrimSpace(id) == candidateID {
			return true
		}
	}
	return false
}

func modelWithStableTimes(model BusinessMemorySourceAssetModel) BusinessMemorySourceAssetModel {
	model.FirstSeenAt = model.FirstSeenAt.UTC()
	model.LastSeenAt = model.LastSeenAt.UTC()
	return model
}
