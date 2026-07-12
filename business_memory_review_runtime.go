package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	documentsvm "ph_holdings_app/internal/viewmodel/documents"
	adapterdocuments "ph_holdings_app/pkg/adapter/documents"
	"ph_holdings_app/pkg/documents/intake"
)

type BusinessMemoryReviewDecisionRequest struct {
	CandidateID                  string `json:"candidate_id"`
	Decision                     string `json:"decision"`
	Actor                        string `json:"actor"`
	ActorType                    string `json:"actor_type,omitempty"`
	Reason                       string `json:"reason,omitempty"`
	ProposedDeterministicService string `json:"proposed_deterministic_service,omitempty"`
	CorrelationID                string `json:"correlation_id,omitempty"`
}

type BusinessMemoryReviewResult struct {
	Record          intake.ReviewRecord        `json:"record"`
	Queue           documentsvm.IntakeReviewVM `json:"queue"`
	ContextPack     intake.ContextPack         `json:"context_pack"`
	ContextPackTOON string                     `json:"context_pack_toon"`
}

type BusinessMemoryContextPackResult struct {
	CandidateID     string             `json:"candidate_id"`
	ContextPack     intake.ContextPack `json:"context_pack"`
	ContextPackTOON string             `json:"context_pack_toon"`
}

type BusinessMemoryReviewExportResult struct {
	CandidateID string                    `json:"candidate_id"`
	Bundle      intake.ReviewExportBundle `json:"bundle"`
	JSON        string                    `json:"json"`
	TOON        string                    `json:"toon"`
}

func (a *App) GetBusinessMemoryReviewQueue(selectedID string) (documentsvm.IntakeReviewVM, error) {
	if err := a.requirePermission("documents:view"); err != nil {
		return documentsvm.IntakeReviewVM{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	service, err := a.businessMemoryReviewService(ctx)
	if err != nil {
		return documentsvm.IntakeReviewVM{}, err
	}
	sourceRepo, err := a.businessMemorySourceAssetRepository(ctx)
	if err != nil {
		return documentsvm.IntakeReviewVM{}, err
	}
	states, err := a.businessMemoryReviewQueueStates(ctx, service, sourceRepo)
	if err != nil {
		return documentsvm.IntakeReviewVM{}, err
	}
	return documentsvm.BuildIntakeReviewVMFromQueueStates(states, selectedID), nil
}

func (a *App) RecordBusinessMemoryReviewDecision(req BusinessMemoryReviewDecisionRequest) (*BusinessMemoryReviewResult, error) {
	if err := a.requirePermission("documents:classify"); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	service, err := a.businessMemoryReviewService(ctx)
	if err != nil {
		return nil, err
	}
	sourceRepo, err := a.businessMemorySourceAssetRepository(ctx)
	if err != nil {
		return nil, err
	}
	candidate, err := a.businessMemoryCandidateByID(req.CandidateID)
	if err != nil {
		return nil, err
	}
	decision, err := parseBusinessMemoryReviewDecision(req.Decision)
	if err != nil {
		return nil, err
	}
	serviceTarget := strings.TrimSpace(req.ProposedDeterministicService)
	if serviceTarget == "" {
		serviceTarget = firstBusinessMemoryDeterministicTarget(candidate)
	}
	correlationID := strings.TrimSpace(req.CorrelationID)
	if correlationID == "" {
		correlationID = businessMemoryCorrelationID(candidate.ID, decision, serviceTarget)
	}

	record, err := service.RecordDecision(ctx, intake.ReviewCommand{
		Candidate:                    candidate,
		Decision:                     decision,
		Actor:                        strings.TrimSpace(req.Actor),
		ActorType:                    strings.TrimSpace(req.ActorType),
		Reason:                       strings.TrimSpace(req.Reason),
		ProposedDeterministicService: serviceTarget,
		CorrelationID:                correlationID,
	})
	if err != nil {
		return nil, err
	}

	states, err := a.businessMemoryReviewQueueStates(ctx, service, sourceRepo)
	if err != nil {
		return nil, err
	}
	pack := service.BuildContextPack(candidate)
	return &BusinessMemoryReviewResult{
		Record:          record,
		Queue:           documentsvm.BuildIntakeReviewVMFromQueueStates(states, candidate.ID),
		ContextPack:     pack,
		ContextPackTOON: intake.FormatContextPackTOON(pack),
	}, nil
}

func (a *App) GenerateBusinessMemoryContextPack(candidateID string) (*BusinessMemoryContextPackResult, error) {
	if err := a.requirePermission("documents:view"); err != nil {
		return nil, err
	}
	candidate, err := a.businessMemoryCandidateByID(candidateID)
	if err != nil {
		return nil, err
	}
	pack := intake.BuildContextPack(candidate)
	return &BusinessMemoryContextPackResult{
		CandidateID:     candidate.ID,
		ContextPack:     pack,
		ContextPackTOON: intake.FormatContextPackTOON(pack),
	}, nil
}

func (a *App) ExportBusinessMemoryReviewBundle(candidateID string) (*BusinessMemoryReviewExportResult, error) {
	if err := a.requirePermission("documents:view"); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	service, err := a.businessMemoryReviewService(ctx)
	if err != nil {
		return nil, err
	}
	sourceRepo, err := a.businessMemorySourceAssetRepository(ctx)
	if err != nil {
		return nil, err
	}
	candidate, err := a.businessMemoryCandidateByID(candidateID)
	if err != nil {
		return nil, err
	}
	sourceAssets, err := a.businessMemorySourceAssetsForCandidate(ctx, sourceRepo, candidate)
	if err != nil {
		return nil, err
	}
	records, err := service.ListReviewRecords(ctx, candidate.ID)
	if err != nil {
		return nil, err
	}
	bundle, err := intake.NewReviewExportBundleWithSources(candidate, records, sourceAssets, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	payload, err := intake.ExportReviewBundleJSON(bundle)
	if err != nil {
		return nil, err
	}
	return &BusinessMemoryReviewExportResult{
		CandidateID: candidate.ID,
		Bundle:      bundle,
		JSON:        string(payload),
		TOON:        intake.ExportReviewBundleTOON(bundle),
	}, nil
}

func (a *App) businessMemoryReviewService(ctx context.Context) (*intake.ReviewService, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	repo := adapterdocuments.NewGORMBusinessMemoryReviewRepository(a.db)
	if err := repo.Migrate(ctx); err != nil {
		return nil, err
	}
	return intake.NewReviewService(repo)
}

func (a *App) businessMemorySourceAssetRepository(ctx context.Context) (intake.SourceAssetRepository, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	repo := adapterdocuments.NewGORMBusinessMemorySourceAssetRepository(a.db)
	if err := repo.Migrate(ctx); err != nil {
		return nil, err
	}
	return repo, nil
}

func (a *App) businessMemoryReviewQueueStates(ctx context.Context, service *intake.ReviewService, sourceRepo intake.SourceAssetRepository) ([]intake.ReviewQueueState, error) {
	docs, err := a.GetInboxDocuments("")
	if err != nil {
		return nil, err
	}
	states := make([]intake.ReviewQueueState, 0, len(docs))
	for _, doc := range docs {
		candidate := inboxDocumentToBusinessMemoryCandidate(doc)
		if _, _, err := sourceRepo.Upsert(ctx, candidateToBusinessMemorySourceAsset(candidate)); err != nil {
			return nil, err
		}
		state, err := service.BuildQueueStateWithSources(ctx, candidate, sourceRepo)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func (a *App) businessMemorySourceAssetsForCandidate(ctx context.Context, sourceRepo intake.SourceAssetRepository, candidate intake.Candidate) ([]intake.SourceAsset, error) {
	if _, _, err := sourceRepo.Upsert(ctx, candidateToBusinessMemorySourceAsset(candidate)); err != nil {
		return nil, err
	}
	return sourceRepo.ListByCandidate(ctx, candidate.ID)
}

func (a *App) businessMemoryCandidateByID(candidateID string) (intake.Candidate, error) {
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return intake.Candidate{}, fmt.Errorf("candidate id is required")
	}
	docs, err := a.GetInboxDocuments("")
	if err != nil {
		return intake.Candidate{}, err
	}
	for _, doc := range docs {
		candidate := inboxDocumentToBusinessMemoryCandidate(doc)
		if candidate.ID == candidateID || candidate.Source.ID == candidateID {
			return candidate, nil
		}
	}
	return intake.Candidate{}, fmt.Errorf("business memory candidate %q not found", candidateID)
}

func inboxDocumentToBusinessMemoryCandidate(doc InboxDocument) intake.Candidate {
	return intake.FromInboxDocument(intake.InboxDocumentInput{
		ID:               doc.ID,
		DocumentID:       doc.DocumentID,
		FileName:         doc.FileName,
		FilePath:         doc.FilePath,
		DocumentType:     doc.DocumentType,
		Status:           doc.Status,
		Confidence:       doc.Confidence,
		ExtractedData:    doc.ExtractedData,
		SuggestedActions: doc.SuggestedActions,
		ProcessedAt:      doc.ProcessedAt,
		CreatedAt:        doc.CreatedAt,
	})
}

func candidateToBusinessMemorySourceAsset(candidate intake.Candidate) intake.SourceAsset {
	seenAt := time.Now().UTC()
	if candidate.Source.ProcessedAt != nil && !candidate.Source.ProcessedAt.IsZero() {
		seenAt = candidate.Source.ProcessedAt.UTC()
	}
	status := intake.SourceStatusCandidateGenerated
	switch candidate.ReviewStatus {
	case intake.ReviewStatusLinked, intake.ReviewStatusCorrected:
		status = intake.SourceStatusReviewed
	case intake.ReviewStatusRejected, intake.ReviewStatusArchived:
		status = intake.SourceStatusArchived
	}
	asset, err := intake.NewSourceAsset(intake.SourceAssetInput{
		Kind:             candidate.SourceKind,
		Path:             candidate.Source.Path,
		Label:            firstBusinessMemoryNonEmpty(candidate.Source.Label, candidate.Source.ID, candidate.ID),
		PrivacyClass:     intake.SourcePrivacyInternal,
		ProcessingStatus: status,
		CandidateIDs:     []string{candidate.ID},
		AuditRefs:        candidate.AuditRefs,
		SeenAt:           seenAt,
	})
	if err != nil {
		return intake.SourceAsset{
			ID:               firstBusinessMemoryNonEmpty(candidate.Source.ID, candidate.ID),
			Kind:             candidate.SourceKind,
			Label:            firstBusinessMemoryNonEmpty(candidate.Source.Label, candidate.Source.ID, candidate.ID, "Inbox source"),
			PrivacyClass:     intake.SourcePrivacyInternal,
			ProcessingStatus: status,
			CandidateIDs:     []string{candidate.ID},
			AuditRefs:        candidate.AuditRefs,
			FirstSeenAt:      seenAt,
			LastSeenAt:       seenAt,
		}
	}
	if strings.TrimSpace(candidate.Source.ID) != "" {
		asset.ID = candidate.Source.ID
	}
	return asset
}

func parseBusinessMemoryReviewDecision(value string) (intake.ReviewDecision, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch normalized {
	case string(intake.ReviewDecisionAcceptProposal), "accept", "approved", "approve":
		return intake.ReviewDecisionAcceptProposal, nil
	case string(intake.ReviewDecisionNeedsInput), "needs_review", "need_input":
		return intake.ReviewDecisionNeedsInput, nil
	case string(intake.ReviewDecisionCorrectField), "correct", "corrected":
		return intake.ReviewDecisionCorrectField, nil
	case string(intake.ReviewDecisionRejectCandidate), "reject", "rejected":
		return intake.ReviewDecisionRejectCandidate, nil
	case string(intake.ReviewDecisionArchive), "archived":
		return intake.ReviewDecisionArchive, nil
	default:
		return "", fmt.Errorf("unsupported business memory review decision %q", value)
	}
}

func firstBusinessMemoryDeterministicTarget(candidate intake.Candidate) string {
	for _, link := range candidate.SuggestedLinks {
		if target := strings.TrimSpace(link.RequiredDeterministicService); target != "" {
			return target
		}
	}
	return "documents.intake.review"
}

func businessMemoryCorrelationID(candidateID string, decision intake.ReviewDecision, serviceTarget string) string {
	return strings.Join([]string{
		"business-memory-review",
		strings.TrimSpace(candidateID),
		string(decision),
		strings.TrimSpace(serviceTarget),
	}, ":")
}

func firstBusinessMemoryNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
