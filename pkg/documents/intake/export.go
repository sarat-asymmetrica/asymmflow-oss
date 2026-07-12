package intake

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ph_holdings_app/pkg/kernel/text"
)

const ReviewExportSchemaVersion = "business-memory-review-bundle/v1"

type ReviewExportBundle struct {
	SchemaVersion string         `json:"schema_version"`
	ExportedAt    time.Time      `json:"exported_at"`
	Candidate     Candidate      `json:"candidate"`
	ContextPack   ContextPack    `json:"context_pack"`
	SourceAssets  []SourceAsset  `json:"source_assets,omitempty"`
	ReviewRecords []ReviewRecord `json:"review_records"`
}

func NewReviewExportBundle(candidate Candidate, records []ReviewRecord, exportedAt time.Time) (ReviewExportBundle, error) {
	return NewReviewExportBundleWithSources(candidate, records, nil, exportedAt)
}

func NewReviewExportBundleWithSources(candidate Candidate, records []ReviewRecord, sourceAssets []SourceAsset, exportedAt time.Time) (ReviewExportBundle, error) {
	candidate = normalizeCandidate(candidate, Options{})
	if strings.TrimSpace(candidate.ID) == "" {
		return ReviewExportBundle{}, fmt.Errorf("candidate id is required")
	}
	if exportedAt.IsZero() {
		exportedAt = time.Now().UTC()
	}
	records = sortReviewRecords(records)
	sourceAssets = sortSourceAssets(sourceAssets)
	for _, asset := range sourceAssets {
		if err := ValidateSourceAsset(asset); err != nil {
			return ReviewExportBundle{}, err
		}
	}
	return ReviewExportBundle{
		SchemaVersion: ReviewExportSchemaVersion,
		ExportedAt:    exportedAt.UTC(),
		Candidate:     candidate,
		ContextPack:   BuildContextPack(candidate),
		SourceAssets:  append([]SourceAsset(nil), sourceAssets...),
		ReviewRecords: append([]ReviewRecord(nil), records...),
	}, nil
}

func ExportReviewBundleJSON(bundle ReviewExportBundle) ([]byte, error) {
	if strings.TrimSpace(bundle.SchemaVersion) == "" {
		bundle.SchemaVersion = ReviewExportSchemaVersion
	}
	if strings.TrimSpace(bundle.Candidate.ID) == "" {
		return nil, fmt.Errorf("candidate id is required")
	}
	return json.MarshalIndent(bundle, "", "  ")
}

func ReplayReviewBundleJSON(payload []byte) (ReviewExportBundle, error) {
	var bundle ReviewExportBundle
	if err := json.Unmarshal(payload, &bundle); err != nil {
		return ReviewExportBundle{}, err
	}
	if bundle.SchemaVersion != ReviewExportSchemaVersion {
		return ReviewExportBundle{}, fmt.Errorf("unsupported review export schema %q", bundle.SchemaVersion)
	}
	if strings.TrimSpace(bundle.Candidate.ID) == "" {
		return ReviewExportBundle{}, fmt.Errorf("candidate id is required")
	}
	bundle.Candidate = normalizeCandidate(bundle.Candidate, Options{})
	bundle.ContextPack = BuildContextPack(bundle.Candidate)
	bundle.SourceAssets = sortSourceAssets(bundle.SourceAssets)
	bundle.ReviewRecords = sortReviewRecords(bundle.ReviewRecords)
	return bundle, nil
}

func ExportReviewBundleTOON(bundle ReviewExportBundle) string {
	var b strings.Builder
	writeLine(&b, "business_memory_review_bundle:")
	writeLine(&b, "  schema_version: %s", text.FirstNonEmpty(bundle.SchemaVersion, ReviewExportSchemaVersion))
	if !bundle.ExportedAt.IsZero() {
		writeLine(&b, "  exported_at: %s", bundle.ExportedAt.UTC().Format(time.RFC3339))
	}
	writeLine(&b, "  candidate_id: %s", bundle.Candidate.ID)
	writeLine(&b, "  source_summary: %s", sourceSummary(bundle.Candidate.Source))
	writeLine(&b, "  review_status: %s", bundle.Candidate.ReviewStatus)
	writeLine(&b, "  source_assets:")
	if len(bundle.SourceAssets) == 0 {
		writeLine(&b, "    - none")
	} else {
		for _, asset := range sortSourceAssets(bundle.SourceAssets) {
			writeLine(&b, "    - id: %s", asset.ID)
			writeLine(&b, "      kind: %s", asset.Kind)
			writeLine(&b, "      label: %s", asset.Label)
			if asset.Path != "" {
				writeLine(&b, "      path: %s", asset.Path)
			}
			if asset.Hash != "" {
				writeLine(&b, "      hash: %s", asset.Hash)
			}
			writeLine(&b, "      privacy_class: %s", asset.PrivacyClass)
			writeLine(&b, "      processing_status: %s", asset.ProcessingStatus)
			writeLine(&b, "      candidate_ids: %s", strings.Join(asset.CandidateIDs, ","))
			writeLine(&b, "      audit_ref_count: %d", len(asset.AuditRefs))
		}
	}
	writeLine(&b, "  context_pack:")
	writeIndentedBlock(&b, FormatContextPackTOON(bundle.ContextPack), "    ")
	writeLine(&b, "  review_records:")
	if len(bundle.ReviewRecords) == 0 {
		writeLine(&b, "    - none")
		return strings.TrimRight(b.String(), "\n")
	}
	for _, record := range sortReviewRecords(bundle.ReviewRecords) {
		writeLine(&b, "    - id: %s", record.ID)
		writeLine(&b, "      decision: %s", record.Decision)
		writeLine(&b, "      review_status: %s", record.ReviewStatus)
		writeLine(&b, "      actor: %s", record.Actor)
		writeLine(&b, "      deterministic_service: %s", text.FirstNonEmpty(record.ProposedDeterministicService, "none"))
		writeLine(&b, "      correlation_id: %s", record.CorrelationID)
		if record.Reason != "" {
			writeLine(&b, "      reason: %s", record.Reason)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func writeIndentedBlock(b *strings.Builder, text string, indent string) {
	for _, line := range strings.Split(text, "\n") {
		writeLine(b, "%s%s", indent, line)
	}
}
