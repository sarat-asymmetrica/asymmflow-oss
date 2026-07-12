package intake

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"ph_holdings_app/pkg/kernel/text"
)

type SourcePrivacyClass string

const (
	SourcePrivacyInternal     SourcePrivacyClass = "internal"
	SourcePrivacyConfidential SourcePrivacyClass = "confidential"
	SourcePrivacyRestricted   SourcePrivacyClass = "restricted"
)

type SourceProcessingStatus string

const (
	SourceStatusDiscovered         SourceProcessingStatus = "discovered"
	SourceStatusNormalized         SourceProcessingStatus = "normalized"
	SourceStatusCandidateGenerated SourceProcessingStatus = "candidate_generated"
	SourceStatusReviewed           SourceProcessingStatus = "reviewed"
	SourceStatusArchived           SourceProcessingStatus = "archived"
	SourceStatusError              SourceProcessingStatus = "error"
)

type SourceAsset struct {
	ID               string                 `json:"id"`
	Kind             SourceKind             `json:"kind"`
	Path             string                 `json:"path,omitempty"`
	Label            string                 `json:"label"`
	Hash             string                 `json:"hash,omitempty"`
	ImportBatchID    string                 `json:"import_batch_id,omitempty"`
	PrivacyClass     SourcePrivacyClass     `json:"privacy_class"`
	ProcessingStatus SourceProcessingStatus `json:"processing_status"`
	CandidateIDs     []string               `json:"candidate_ids,omitempty"`
	AuditRefs        []AuditRef             `json:"audit_refs,omitempty"`
	FirstSeenAt      time.Time              `json:"first_seen_at"`
	LastSeenAt       time.Time              `json:"last_seen_at"`
}

type SourceAssetInput struct {
	Kind             SourceKind
	Path             string
	Label            string
	Hash             string
	ImportBatchID    string
	PrivacyClass     SourcePrivacyClass
	ProcessingStatus SourceProcessingStatus
	CandidateIDs     []string
	AuditRefs        []AuditRef
	SeenAt           time.Time
}

type SourceAssetRegistry struct {
	mu     sync.RWMutex
	assets map[string]SourceAsset
}

func NewSourceAsset(input SourceAssetInput, opts ...Options) (SourceAsset, error) {
	asset := normalizeSourceAsset(input, mergeOptions(opts...))
	if strings.TrimSpace(asset.ID) == "" {
		return SourceAsset{}, fmt.Errorf("source asset id is required")
	}
	if strings.TrimSpace(asset.Label) == "" {
		return SourceAsset{}, fmt.Errorf("source asset label is required")
	}
	return asset, nil
}

func NewSourceAssetRegistry() *SourceAssetRegistry {
	return &SourceAssetRegistry{assets: map[string]SourceAsset{}}
}

func (r *SourceAssetRegistry) Upsert(input SourceAssetInput, opts ...Options) (SourceAsset, bool, error) {
	asset, err := NewSourceAsset(input, opts...)
	if err != nil {
		return SourceAsset{}, false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.assets == nil {
		r.assets = map[string]SourceAsset{}
	}
	existing, duplicate := r.assets[asset.ID]
	if duplicate {
		asset = mergeSourceAssets(existing, asset)
	}
	r.assets[asset.ID] = asset
	return asset, duplicate, nil
}

func (r *SourceAssetRegistry) Get(id string) (SourceAsset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	asset, ok := r.assets[strings.TrimSpace(id)]
	return asset, ok
}

func SourceAssetStableID(input SourceAssetInput) string {
	kind := input.Kind
	if kind == "" {
		kind = SourceKindFromPath(text.FirstNonEmpty(input.Path, input.Label))
	}
	if kind == SourceKindOther && strings.TrimSpace(input.Path) == "" {
		kind = SourceKindOther
	}
	fingerprint := text.FirstNonEmpty(input.Hash, input.Path, input.Label)
	return buildID("source-asset", string(kind), fingerprint)
}

func normalizeSourceAsset(input SourceAssetInput, opts Options) SourceAsset {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	seenAt := input.SeenAt
	if seenAt.IsZero() {
		seenAt = opts.Now().UTC()
	}
	kind := input.Kind
	if kind == "" {
		kind = SourceKindFromPath(text.FirstNonEmpty(input.Path, input.Label))
	}
	if kind == "" {
		kind = SourceKindOther
	}
	privacy := input.PrivacyClass
	if privacy == "" {
		privacy = SourcePrivacyInternal
	}
	status := input.ProcessingStatus
	if status == "" {
		status = SourceStatusDiscovered
	}
	asset := SourceAsset{
		ID:               SourceAssetStableID(input),
		Kind:             kind,
		Path:             strings.TrimSpace(input.Path),
		Label:            text.FirstNonEmpty(input.Label, input.Path, input.Hash, "unlabeled source"),
		Hash:             strings.TrimSpace(input.Hash),
		ImportBatchID:    strings.TrimSpace(input.ImportBatchID),
		PrivacyClass:     privacy,
		ProcessingStatus: status,
		CandidateIDs:     uniqueStrings(input.CandidateIDs),
		AuditRefs:        sortedAuditRefs(input.AuditRefs),
		FirstSeenAt:      seenAt,
		LastSeenAt:       seenAt,
	}
	return asset
}

func mergeSourceAssets(existing SourceAsset, incoming SourceAsset) SourceAsset {
	merged := existing
	merged.Label = text.FirstNonEmpty(incoming.Label, existing.Label)
	merged.Path = text.FirstNonEmpty(incoming.Path, existing.Path)
	merged.Hash = text.FirstNonEmpty(incoming.Hash, existing.Hash)
	merged.ImportBatchID = text.FirstNonEmpty(incoming.ImportBatchID, existing.ImportBatchID)
	if incoming.Kind != "" {
		merged.Kind = incoming.Kind
	}
	if incoming.PrivacyClass != "" {
		merged.PrivacyClass = incoming.PrivacyClass
	}
	if incoming.ProcessingStatus != "" {
		merged.ProcessingStatus = incoming.ProcessingStatus
	}
	merged.CandidateIDs = uniqueStrings(append(merged.CandidateIDs, incoming.CandidateIDs...))
	merged.AuditRefs = sortedAuditRefs(append(merged.AuditRefs, incoming.AuditRefs...))
	if merged.FirstSeenAt.IsZero() || (!incoming.FirstSeenAt.IsZero() && incoming.FirstSeenAt.Before(merged.FirstSeenAt)) {
		merged.FirstSeenAt = incoming.FirstSeenAt
	}
	if incoming.LastSeenAt.After(merged.LastSeenAt) {
		merged.LastSeenAt = incoming.LastSeenAt
	}
	return merged
}

func MergeSourceAssets(existing SourceAsset, incoming SourceAsset) SourceAsset {
	return mergeSourceAssets(existing, incoming)
}
