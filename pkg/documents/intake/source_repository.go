package intake

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type SourceAssetRepository interface {
	Upsert(ctx context.Context, asset SourceAsset) (SourceAsset, bool, error)
	Get(ctx context.Context, id string) (SourceAsset, bool, error)
	List(ctx context.Context, filter SourceAssetListFilter) ([]SourceAsset, error)
	ListByCandidate(ctx context.Context, candidateID string) ([]SourceAsset, error)
}

type SourceAssetListFilter struct {
	Kind             SourceKind
	ProcessingStatus SourceProcessingStatus
	PrivacyClass     SourcePrivacyClass
	CandidateID      string
}

type MemorySourceAssetRepository struct {
	mu     sync.RWMutex
	assets map[string]SourceAsset
}

func NewMemorySourceAssetRepository() *MemorySourceAssetRepository {
	return &MemorySourceAssetRepository{assets: map[string]SourceAsset{}}
}

func (r *MemorySourceAssetRepository) Upsert(_ context.Context, asset SourceAsset) (SourceAsset, bool, error) {
	if err := ValidateSourceAsset(asset); err != nil {
		return SourceAsset{}, false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.assets == nil {
		r.assets = map[string]SourceAsset{}
	}
	id := strings.TrimSpace(asset.ID)
	existing, duplicate := r.assets[id]
	if duplicate {
		asset = mergeSourceAssets(existing, asset)
	}
	r.assets[id] = asset
	return asset, duplicate, nil
}

func (r *MemorySourceAssetRepository) Get(_ context.Context, id string) (SourceAsset, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	asset, ok := r.assets[strings.TrimSpace(id)]
	return asset, ok, nil
}

func (r *MemorySourceAssetRepository) List(_ context.Context, filter SourceAssetListFilter) ([]SourceAsset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	assets := make([]SourceAsset, 0, len(r.assets))
	for _, asset := range r.assets {
		if !sourceAssetMatchesFilter(asset, filter) {
			continue
		}
		assets = append(assets, asset)
	}
	return sortSourceAssets(assets), nil
}

func (r *MemorySourceAssetRepository) ListByCandidate(ctx context.Context, candidateID string) ([]SourceAsset, error) {
	return r.List(ctx, SourceAssetListFilter{CandidateID: candidateID})
}

func ValidateSourceAsset(asset SourceAsset) error {
	if strings.TrimSpace(asset.ID) == "" {
		return fmt.Errorf("source asset id is required")
	}
	if asset.Kind == "" {
		return fmt.Errorf("source asset kind is required")
	}
	if strings.TrimSpace(asset.Label) == "" {
		return fmt.Errorf("source asset label is required")
	}
	if asset.PrivacyClass == "" {
		return fmt.Errorf("source asset privacy class is required")
	}
	if asset.ProcessingStatus == "" {
		return fmt.Errorf("source asset processing status is required")
	}
	if asset.FirstSeenAt.IsZero() {
		return fmt.Errorf("source asset first seen timestamp is required")
	}
	if asset.LastSeenAt.IsZero() {
		return fmt.Errorf("source asset last seen timestamp is required")
	}
	if asset.LastSeenAt.Before(asset.FirstSeenAt) {
		return fmt.Errorf("source asset last seen timestamp cannot be before first seen timestamp")
	}
	return nil
}

func sourceAssetMatchesFilter(asset SourceAsset, filter SourceAssetListFilter) bool {
	if filter.Kind != "" && asset.Kind != filter.Kind {
		return false
	}
	if filter.ProcessingStatus != "" && asset.ProcessingStatus != filter.ProcessingStatus {
		return false
	}
	if filter.PrivacyClass != "" && asset.PrivacyClass != filter.PrivacyClass {
		return false
	}
	if strings.TrimSpace(filter.CandidateID) != "" && !containsString(asset.CandidateIDs, filter.CandidateID) {
		return false
	}
	return true
}

func sortSourceAssets(assets []SourceAsset) []SourceAsset {
	out := append([]SourceAsset(nil), assets...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].FirstSeenAt.Equal(out[j].FirstSeenAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].FirstSeenAt.Before(out[j].FirstSeenAt)
	})
	return out
}

func containsString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
