// Package documents contains display-ready ViewModels for document workflows.
package documents

import (
	"fmt"
	"math"
	"strings"

	vm "ph_holdings_app/internal/viewmodel"
	"ph_holdings_app/internal/viewmodel/shared"
	"ph_holdings_app/pkg/documents/intake"
	"ph_holdings_app/pkg/kernel/text"
)

// DocumentUploadVM is the display contract for document intake.
type DocumentUploadVM struct {
	AcceptedTypes        []string               `json:"acceptedTypes"`
	MaxFileSizeDisplay   string                 `json:"maxFileSizeDisplay"`
	DropzoneLabel        string                 `json:"dropzoneLabel"`
	UploadProgress       ProgressVM             `json:"uploadProgress"`
	OCRProgress          ProgressVM             `json:"ocrProgress"`
	ClassificationResult ClassificationResultVM `json:"classificationResult"`
	Actions              []vm.ActionButton      `json:"actions"`
}

// ProgressVM displays a long-running document workflow.
type ProgressVM struct {
	Label       string               `json:"label"`
	Percent     int                  `json:"percent"`
	Status      shared.StatusBadgeVM `json:"status"`
	Description string               `json:"description,omitempty"`
}

// ClassificationResultVM displays a document classifier result.
type ClassificationResultVM struct {
	DocumentType      string               `json:"documentType"`
	ConfidenceDisplay string               `json:"confidenceDisplay"`
	Status            shared.StatusBadgeVM `json:"status"`
	MatchedEntity     string               `json:"matchedEntity,omitempty"`
	Keywords          []string             `json:"keywords,omitempty"`
}

// OCRResultVM is the display contract for extracted OCR output.
type OCRResultVM struct {
	DocumentID        string               `json:"documentId"`
	FileName          string               `json:"fileName"`
	ExtractedText     string               `json:"extractedText"`
	ConfidenceDisplay string               `json:"confidenceDisplay"`
	ConfidenceBadge   shared.StatusBadgeVM `json:"confidenceBadge"`
	Fields            []ExtractedFieldVM   `json:"fields"`
	Actions           []vm.ActionButton    `json:"actions"`
}

// ExtractedFieldVM displays one OCR extracted field.
type ExtractedFieldVM struct {
	Name              string               `json:"name"`
	Label             string               `json:"label"`
	Value             string               `json:"value"`
	ConfidenceDisplay string               `json:"confidenceDisplay,omitempty"`
	Status            shared.StatusBadgeVM `json:"status"`
}

// InboxVM is the display contract for the document inbox.
type InboxVM struct {
	Table         shared.TableVM    `json:"table"`
	StatusFilters []vm.Option       `json:"statusFilters"`
	SummaryCards  []vm.SummaryCard  `json:"summaryCards"`
	Actions       []vm.ActionButton `json:"actions"`
	IntakeReview  IntakeReviewVM    `json:"intakeReview"`
}

// InboxDocumentVM is a display-ready inbox document row.
type InboxDocumentVM struct {
	ID            string               `json:"id"`
	FileName      string               `json:"fileName"`
	DocumentType  string               `json:"documentType"`
	ReceivedAt    string               `json:"receivedAt"`
	Status        shared.StatusBadgeVM `json:"status"`
	Confidence    string               `json:"confidence"`
	MatchedEntity string               `json:"matchedEntity,omitempty"`
	Actions       []vm.ActionButton    `json:"actions"`
}

// IntakeReviewInput is the ViewModel boundary for canonical business-memory
// candidates. Worker A can adapt the pure intake package into this shape without
// giving the ViewModel authority over normalization or persistence.
type IntakeReviewInput struct {
	ID                 string
	SourceLabel        string
	SourceKind         string
	BusinessObjectType string
	Classification     string
	ReviewStatus       string
	Confidence         float64
	ExtractedFields    []IntakeExtractedFieldInput
	Sources            []IntakeEvidenceSourceInput
	SuggestedActions   []IntakeActionProposalInput
	Warnings           []string
	AuditRefs          []string
	LastReview         *IntakeReviewRecordInput
	SourceRegistry     []SourceRegistrySummaryInput
}

// IntakeExtractedFieldInput carries one extracted candidate field.
type IntakeExtractedFieldInput struct {
	Name       string
	Label      string
	Value      string
	Status     string
	Confidence float64
	SourceRef  string
	Required   bool
}

// IntakeEvidenceSourceInput carries source/provenance completeness.
type IntakeEvidenceSourceInput struct {
	SourceType  string
	Label       string
	Required    int
	Present     int
	Missing     int
	Confidence  float64
	Status      string
	Priority    string
	LastUpdated string
}

// IntakeActionProposalInput carries one deterministic action suggestion.
type IntakeActionProposalInput struct {
	Action                       string
	SourceType                   string
	Label                        string
	Reason                       string
	Priority                     string
	RequiredDeterministicService string
}

type IntakeReviewRecordInput struct {
	Decision                     string
	ReviewStatus                 string
	Actor                        string
	Reason                       string
	CorrelationID                string
	CreatedAt                    string
	ProposedDeterministicService string
}

type SourceRegistrySummaryInput struct {
	SourceID          string
	Kind              string
	Label             string
	Path              string
	PrivacyClass      string
	ProcessingStatus  string
	CandidateCount    int
	CurrentCandidate  bool
	AuditRefCount     int
	LastSeenAtDisplay string
}

// IntakeReviewVM is the bounded Inbox review workbench state.
type IntakeReviewVM struct {
	QueueMetrics []KpiStatusItemVM          `json:"queueMetrics"`
	Selected     *IntakeCandidateReviewVM   `json:"selected,omitempty"`
	Candidates   []IntakeCandidateSummaryVM `json:"candidates"`
	Actions      []vm.ActionButton          `json:"actions"`
}

// KpiStatusItemVM mirrors the reusable frontend KpiStatusStrip shape.
type KpiStatusItemVM struct {
	Label    string `json:"label"`
	Value    string `json:"value"`
	Meta     string `json:"meta,omitempty"`
	Status   string `json:"status,omitempty"`
	Priority string `json:"priority,omitempty"`
}

// IntakeCandidateSummaryVM is a compact queue row.
type IntakeCandidateSummaryVM struct {
	ID                 string               `json:"id"`
	SourceLabel        string               `json:"sourceLabel"`
	SourceKind         string               `json:"sourceKind"`
	BusinessObjectType string               `json:"businessObjectType"`
	Classification     string               `json:"classification"`
	ReviewStatus       shared.StatusBadgeVM `json:"reviewStatus"`
	ConfidenceDisplay  string               `json:"confidenceDisplay"`
	WarningCount       int                  `json:"warningCount"`
	LastReviewAction   string               `json:"lastReviewAction,omitempty"`
	ServiceTarget      string               `json:"serviceTarget,omitempty"`
}

// IntakeCandidateReviewVM is the selected candidate review surface.
type IntakeCandidateReviewVM struct {
	IntakeCandidateSummaryVM
	ExtractedFields []IntakeFieldRowVM     `json:"extractedFields"`
	Sources         []EvidenceSourceItemVM `json:"sources"`
	SourceRegistry  []SourceRegistryItemVM `json:"sourceRegistry,omitempty"`
	ActionProposals []ActionProposalItemVM `json:"actionProposals"`
	AuditRefs       []string               `json:"auditRefs,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
	LastReview      *IntakeReviewRecordVM  `json:"lastReview,omitempty"`
	ReviewCommands  []vm.ActionButton      `json:"reviewCommands"`
}

type IntakeReviewRecordVM struct {
	Decision                     string `json:"decision"`
	ReviewStatus                 string `json:"reviewStatus"`
	Actor                        string `json:"actor"`
	Reason                       string `json:"reason,omitempty"`
	CorrelationID                string `json:"correlationId"`
	CreatedAt                    string `json:"createdAt"`
	ProposedDeterministicService string `json:"proposedDeterministicService,omitempty"`
}

type SourceRegistryItemVM struct {
	SourceID          string `json:"sourceId"`
	Kind              string `json:"kind"`
	Label             string `json:"label"`
	Path              string `json:"path,omitempty"`
	PrivacyClass      string `json:"privacyClass"`
	ProcessingStatus  string `json:"processingStatus"`
	CandidateCount    int    `json:"candidateCount"`
	CurrentCandidate  bool   `json:"currentCandidate"`
	AuditRefCount     int    `json:"auditRefCount"`
	LastSeenAtDisplay string `json:"lastSeenAtDisplay,omitempty"`
}

// IntakeFieldRowVM is a display-ready extracted field row.
type IntakeFieldRowVM struct {
	Name              string               `json:"name"`
	Label             string               `json:"label"`
	Value             string               `json:"value"`
	ConfidenceDisplay string               `json:"confidenceDisplay,omitempty"`
	Status            shared.StatusBadgeVM `json:"status"`
	SourceRef         string               `json:"sourceRef,omitempty"`
	Required          bool                 `json:"required"`
}

// EvidenceSourceItemVM mirrors the reusable frontend EvidenceSourceList shape.
type EvidenceSourceItemVM struct {
	SourceType  string  `json:"source_type,omitempty"`
	Label       string  `json:"label"`
	Required    int     `json:"required,omitempty"`
	Present     int     `json:"present,omitempty"`
	Missing     int     `json:"missing,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
	Status      string  `json:"status,omitempty"`
	Priority    string  `json:"priority,omitempty"`
	LastUpdated string  `json:"last_updated,omitempty"`
}

// ActionProposalItemVM mirrors the reusable frontend ActionProposalCard shape.
type ActionProposalItemVM struct {
	Action                       string `json:"action,omitempty"`
	SourceType                   string `json:"source_type,omitempty"`
	Label                        string `json:"label"`
	Reason                       string `json:"reason"`
	Priority                     string `json:"priority,omitempty"`
	RequiredDeterministicService string `json:"required_deterministic_service,omitempty"`
}

// BuildIntakeReviewVM maps canonical intake candidates to display state for the
// Inbox review surface. It only prepares state and proposal labels.
func BuildIntakeReviewVM(candidates []IntakeReviewInput, selectedID string) IntakeReviewVM {
	summaries := make([]IntakeCandidateSummaryVM, 0, len(candidates))
	var selected *IntakeCandidateReviewVM

	for _, candidate := range candidates {
		review := buildIntakeCandidateReviewVM(candidate)
		summaries = append(summaries, review.IntakeCandidateSummaryVM)
		if selectedID != "" && candidate.ID == selectedID {
			selected = &review
		}
	}

	if selected == nil && len(candidates) > 0 {
		preferred := 0
		for i, candidate := range candidates {
			status := normalizeStatus(candidate.ReviewStatus)
			if status == "needs_review" || status == "new" {
				preferred = i
				break
			}
		}
		review := buildIntakeCandidateReviewVM(candidates[preferred])
		selected = &review
	}

	return IntakeReviewVM{
		QueueMetrics: buildQueueMetrics(candidates),
		Selected:     selected,
		Candidates:   summaries,
		Actions: []vm.ActionButton{
			{Label: "Inspect Sources", Action: "businessMemory.inspectSources", Icon: "search", Variant: "secondary", Enabled: selected != nil},
			{Label: "Record Review Choice", Action: "businessMemory.recordReviewChoice", Icon: "check-circle", Variant: "primary", Enabled: selected != nil},
			{Label: "Draft Context Pack", Action: "businessMemory.draftContextPack", Icon: "file-text", Variant: "secondary", Enabled: selected != nil},
		},
	}
}

// BuildIntakeReviewVMFromCandidates adapts the pure intake package into the
// document ViewModel display boundary.
func BuildIntakeReviewVMFromCandidates(candidates []intake.Candidate, selectedID string) IntakeReviewVM {
	inputs := make([]IntakeReviewInput, 0, len(candidates))
	for _, candidate := range candidates {
		inputs = append(inputs, intakeCandidateToReviewInput(candidate))
	}
	return BuildIntakeReviewVM(inputs, selectedID)
}

// BuildIntakeReviewVMFromQueueStates adapts deterministic review service state
// into the document ViewModel display boundary.
func BuildIntakeReviewVMFromQueueStates(states []intake.ReviewQueueState, selectedID string) IntakeReviewVM {
	inputs := make([]IntakeReviewInput, 0, len(states))
	for _, state := range states {
		input := intakeCandidateToReviewInput(state.Candidate)
		if state.LastReview != nil {
			input.LastReview = intakeReviewRecordToInput(*state.LastReview)
		}
		input.SourceRegistry = intakeSourceAssetsToInputs(state.Candidate.ID, state.SourceAssets)
		inputs = append(inputs, input)
	}
	return BuildIntakeReviewVM(inputs, selectedID)
}

func intakeCandidateToReviewInput(candidate intake.Candidate) IntakeReviewInput {
	fields := make([]IntakeExtractedFieldInput, 0, len(candidate.ExtractedFields))
	for _, field := range candidate.ExtractedFields {
		fields = append(fields, IntakeExtractedFieldInput{
			Name:       field.Name,
			Label:      field.Label,
			Value:      field.Value,
			Status:     string(field.Status),
			Confidence: field.Confidence,
			SourceRef:  field.Source,
			Required:   true,
		})
	}

	links := make([]IntakeActionProposalInput, 0, len(candidate.SuggestedLinks))
	for _, link := range candidate.SuggestedLinks {
		links = append(links, IntakeActionProposalInput{
			Action:                       text.FirstNonEmpty(link.ID, "review_proposal"),
			SourceType:                   string(candidate.SourceKind),
			Label:                        link.Label,
			Reason:                       link.Reason,
			Priority:                     priorityForCandidate(IntakeReviewInput{ReviewStatus: string(candidate.ReviewStatus), Confidence: candidate.Confidence, Warnings: candidate.Warnings}),
			RequiredDeterministicService: link.RequiredDeterministicService,
		})
	}

	auditRefs := make([]string, 0, len(candidate.AuditRefs))
	for _, ref := range candidate.AuditRefs {
		auditRefs = append(auditRefs, text.FirstNonEmpty(ref.Summary, ref.Type+":"+ref.SourceID))
	}

	source := IntakeEvidenceSourceInput{
		SourceType: string(candidate.SourceKind),
		Label:      text.FirstNonEmpty(candidate.Source.Label, candidate.Source.ID, candidate.ID),
		Confidence: candidate.Confidence,
		Status:     completenessStatus(missingFields(fields)),
		Priority:   priorityForCandidate(IntakeReviewInput{ReviewStatus: string(candidate.ReviewStatus), Confidence: candidate.Confidence, Warnings: candidate.Warnings}),
	}
	if candidate.Source.ProcessedAt != nil {
		source.LastUpdated = candidate.Source.ProcessedAt.Format("2006-01-02")
	}
	source.Required, source.Present, source.Missing = sourceCountsFromFields(fields)

	return IntakeReviewInput{
		ID:                 candidate.ID,
		SourceLabel:        text.FirstNonEmpty(candidate.Source.Label, candidate.Source.ID),
		SourceKind:         string(candidate.SourceKind),
		BusinessObjectType: candidate.BusinessObjectType,
		Classification:     candidate.Classification.Type,
		ReviewStatus:       string(candidate.ReviewStatus),
		Confidence:         candidate.Confidence,
		ExtractedFields:    fields,
		Sources:            []IntakeEvidenceSourceInput{source},
		SuggestedActions:   links,
		Warnings:           append([]string(nil), candidate.Warnings...),
		AuditRefs:          auditRefs,
	}
}

func intakeSourceAssetsToInputs(candidateID string, assets []intake.SourceAsset) []SourceRegistrySummaryInput {
	out := make([]SourceRegistrySummaryInput, 0, len(assets))
	for _, asset := range assets {
		lastSeen := ""
		if !asset.LastSeenAt.IsZero() {
			lastSeen = asset.LastSeenAt.Format("2006-01-02")
		}
		out = append(out, SourceRegistrySummaryInput{
			SourceID:          asset.ID,
			Kind:              string(asset.Kind),
			Label:             text.FirstNonEmpty(asset.Label, asset.Path, asset.ID),
			Path:              asset.Path,
			PrivacyClass:      string(asset.PrivacyClass),
			ProcessingStatus:  string(asset.ProcessingStatus),
			CandidateCount:    len(asset.CandidateIDs),
			CurrentCandidate:  stringSliceContains(asset.CandidateIDs, candidateID),
			AuditRefCount:     len(asset.AuditRefs),
			LastSeenAtDisplay: lastSeen,
		})
	}
	return out
}

func intakeReviewRecordToInput(record intake.ReviewRecord) *IntakeReviewRecordInput {
	createdAt := ""
	if !record.CreatedAt.IsZero() {
		createdAt = record.CreatedAt.Format("2006-01-02 15:04")
	}
	return &IntakeReviewRecordInput{
		Decision:                     string(record.Decision),
		ReviewStatus:                 string(record.ReviewStatus),
		Actor:                        record.Actor,
		Reason:                       record.Reason,
		CorrelationID:                record.CorrelationID,
		CreatedAt:                    createdAt,
		ProposedDeterministicService: record.ProposedDeterministicService,
	}
}

func buildQueueMetrics(candidates []IntakeReviewInput) []KpiStatusItemVM {
	var needsReview, corrected, linked, warnings int
	confidenceSum := 0.0
	scored := 0

	for _, candidate := range candidates {
		switch normalizeStatus(candidate.ReviewStatus) {
		case "needs_review", "new":
			needsReview++
		case "corrected":
			corrected++
		case "linked":
			linked++
		}
		if candidate.Confidence > 0 {
			confidenceSum += clamp01(candidate.Confidence)
			scored++
		}
		warnings += len(candidate.Warnings)
	}

	averageConfidence := "not scored"
	if scored > 0 {
		averageConfidence = formatPercent(confidenceSum / float64(scored))
	}

	return []KpiStatusItemVM{
		{Label: "Candidates", Value: fmt.Sprintf("%d", len(candidates)), Meta: "intake queue", Status: queueStatus(needsReview, warnings)},
		{Label: "Review", Value: fmt.Sprintf("%d", needsReview), Meta: "awaiting operator", Status: reviewMetricStatus(needsReview)},
		{Label: "Corrected", Value: fmt.Sprintf("%d", corrected), Meta: "field edits", Status: "ready"},
		{Label: "Linked", Value: fmt.Sprintf("%d", linked), Meta: averageConfidence, Status: linkedMetricStatus(linked, len(candidates))},
	}
}

func buildIntakeCandidateReviewVM(candidate IntakeReviewInput) IntakeCandidateReviewVM {
	summary := IntakeCandidateSummaryVM{
		ID:                 text.FirstNonEmpty(candidate.ID, candidate.SourceLabel, "untracked-candidate"),
		SourceLabel:        text.FirstNonEmpty(candidate.SourceLabel, candidate.ID, "Unlabeled source"),
		SourceKind:         text.FirstNonEmpty(candidate.SourceKind, "other"),
		BusinessObjectType: text.FirstNonEmpty(candidate.BusinessObjectType, "unknown"),
		Classification:     text.FirstNonEmpty(candidate.Classification, "unclassified"),
		ReviewStatus:       badgeForReviewStatus(candidate.ReviewStatus),
		ConfidenceDisplay:  formatPercent(candidate.Confidence),
		WarningCount:       len(candidate.Warnings),
		ServiceTarget:      serviceForBusinessObject(candidate.BusinessObjectType),
	}
	if candidate.LastReview != nil {
		summary.LastReviewAction = candidate.LastReview.Decision
		if candidate.LastReview.ReviewStatus != "" {
			summary.ReviewStatus = badgeForReviewStatus(candidate.LastReview.ReviewStatus)
		}
		if candidate.LastReview.ProposedDeterministicService != "" {
			summary.ServiceTarget = candidate.LastReview.ProposedDeterministicService
		}
	}

	fields := make([]IntakeFieldRowVM, 0, len(candidate.ExtractedFields))
	for _, field := range candidate.ExtractedFields {
		fields = append(fields, IntakeFieldRowVM{
			Name:              text.FirstNonEmpty(field.Name, field.Label),
			Label:             text.FirstNonEmpty(field.Label, field.Name, "Field"),
			Value:             text.FirstNonEmpty(field.Value, "Missing"),
			ConfidenceDisplay: formatPercent(field.Confidence),
			Status:            badgeForFieldStatus(field.Status, field.Value),
			SourceRef:         field.SourceRef,
			Required:          field.Required,
		})
	}

	sources := make([]EvidenceSourceItemVM, 0, len(candidate.Sources))
	for _, source := range candidate.Sources {
		required, present, missing := normalizeSourceCounts(source.Required, source.Present, source.Missing)
		sources = append(sources, EvidenceSourceItemVM{
			SourceType:  text.FirstNonEmpty(source.SourceType, candidate.SourceKind, "source"),
			Label:       text.FirstNonEmpty(source.Label, candidate.SourceLabel, "Source"),
			Required:    required,
			Present:     present,
			Missing:     missing,
			Confidence:  clamp01(firstPositive(source.Confidence, candidate.Confidence)),
			Status:      text.FirstNonEmpty(source.Status, completenessStatus(missing)),
			Priority:    text.FirstNonEmpty(source.Priority, priorityForStatus(source.Status, missing)),
			LastUpdated: source.LastUpdated,
		})
	}
	if len(sources) == 0 {
		required, present, missing := sourceCountsFromFields(candidate.ExtractedFields)
		sources = append(sources, EvidenceSourceItemVM{
			SourceType: text.FirstNonEmpty(candidate.SourceKind, "inbox_record"),
			Label:      text.FirstNonEmpty(candidate.SourceLabel, "Inbox source"),
			Required:   required,
			Present:    present,
			Missing:    missing,
			Confidence: clamp01(candidate.Confidence),
			Status:     completenessStatus(missing),
			Priority:   priorityForStatus("", missing),
		})
	}

	sourceRegistry := make([]SourceRegistryItemVM, 0, len(candidate.SourceRegistry))
	for _, source := range candidate.SourceRegistry {
		sourceRegistry = append(sourceRegistry, SourceRegistryItemVM{
			SourceID:          text.FirstNonEmpty(source.SourceID, "untracked-source"),
			Kind:              text.FirstNonEmpty(source.Kind, candidate.SourceKind, "source"),
			Label:             text.FirstNonEmpty(source.Label, source.Path, candidate.SourceLabel, "Source"),
			Path:              source.Path,
			PrivacyClass:      text.FirstNonEmpty(source.PrivacyClass, "internal"),
			ProcessingStatus:  text.FirstNonEmpty(source.ProcessingStatus, "discovered"),
			CandidateCount:    source.CandidateCount,
			CurrentCandidate:  source.CurrentCandidate,
			AuditRefCount:     source.AuditRefCount,
			LastSeenAtDisplay: source.LastSeenAtDisplay,
		})
	}

	proposals := make([]ActionProposalItemVM, 0, len(candidate.SuggestedActions))
	for _, action := range candidate.SuggestedActions {
		proposals = append(proposals, ActionProposalItemVM{
			Action:                       text.FirstNonEmpty(action.Action, "review_proposal"),
			SourceType:                   text.FirstNonEmpty(action.SourceType, candidate.SourceKind, "business memory"),
			Label:                        text.FirstNonEmpty(action.Label, "Review proposed link"),
			Reason:                       text.FirstNonEmpty(action.Reason, "Operator review is required before any business record changes."),
			Priority:                     text.FirstNonEmpty(action.Priority, priorityForCandidate(candidate)),
			RequiredDeterministicService: text.FirstNonEmpty(action.RequiredDeterministicService, serviceForBusinessObject(candidate.BusinessObjectType)),
		})
	}
	if len(proposals) == 0 {
		proposals = append(proposals, ActionProposalItemVM{
			Action:                       "review_proposal",
			SourceType:                   text.FirstNonEmpty(candidate.SourceKind, "business memory"),
			Label:                        "Review candidate before linking",
			Reason:                       "No deterministic link should be created until extracted fields and provenance are confirmed.",
			Priority:                     priorityForCandidate(candidate),
			RequiredDeterministicService: serviceForBusinessObject(candidate.BusinessObjectType),
		})
	}

	lastReview := intakeReviewRecordVM(candidate.LastReview)

	return IntakeCandidateReviewVM{
		IntakeCandidateSummaryVM: summary,
		ExtractedFields:          fields,
		Sources:                  sources,
		SourceRegistry:           sourceRegistry,
		ActionProposals:          proposals,
		AuditRefs:                append([]string(nil), candidate.AuditRefs...),
		Warnings:                 append([]string(nil), candidate.Warnings...),
		LastReview:               lastReview,
		ReviewCommands: []vm.ActionButton{
			{Label: "Accept Proposal", Action: "businessMemory.review.acceptProposal", Icon: "check", Variant: "primary", Enabled: true},
			{Label: "Needs Input", Action: "businessMemory.review.needsInput", Icon: "alert-circle", Variant: "secondary", Enabled: true},
			{Label: "Correct Field", Action: "businessMemory.review.correctField", Icon: "edit", Variant: "secondary", Enabled: true},
			{Label: "Reject Candidate", Action: "businessMemory.review.rejectCandidate", Icon: "x", Variant: "secondary", Enabled: true},
			{Label: "Archive Review", Action: "businessMemory.review.archive", Icon: "archive", Variant: "secondary", Enabled: true},
		},
	}
}

func stringSliceContains(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func intakeReviewRecordVM(record *IntakeReviewRecordInput) *IntakeReviewRecordVM {
	if record == nil {
		return nil
	}
	return &IntakeReviewRecordVM{
		Decision:                     text.FirstNonEmpty(record.Decision, "review_recorded"),
		ReviewStatus:                 record.ReviewStatus,
		Actor:                        record.Actor,
		Reason:                       record.Reason,
		CorrelationID:                record.CorrelationID,
		CreatedAt:                    record.CreatedAt,
		ProposedDeterministicService: record.ProposedDeterministicService,
	}
}

func badgeForReviewStatus(status string) shared.StatusBadgeVM {
	switch normalizeStatus(status) {
	case "linked":
		return shared.StatusBadgeVM{Label: "Linked", Color: "green", Icon: "link"}
	case "corrected":
		return shared.StatusBadgeVM{Label: "Corrected", Color: "green", Icon: "edit"}
	case "rejected":
		return shared.StatusBadgeVM{Label: "Rejected", Color: "red", Icon: "x-circle"}
	case "archived":
		return shared.StatusBadgeVM{Label: "Archived", Color: "gray", Icon: "archive"}
	case "needs_review":
		return shared.StatusBadgeVM{Label: "Needs Review", Color: "amber", Icon: "alert-circle"}
	default:
		return shared.StatusBadgeVM{Label: "New", Color: "gray", Icon: "circle"}
	}
}

func badgeForFieldStatus(status string, value string) shared.StatusBadgeVM {
	switch normalizeStatus(status) {
	case "corrected":
		return shared.StatusBadgeVM{Label: "Corrected", Color: "green", Icon: "edit"}
	case "needs_confirmation":
		return shared.StatusBadgeVM{Label: "Confirm", Color: "amber", Icon: "alert-circle"}
	case "inferred":
		return shared.StatusBadgeVM{Label: "Inferred", Color: "amber", Icon: "sparkles"}
	case "missing":
		return shared.StatusBadgeVM{Label: "Missing", Color: "red", Icon: "alert-triangle"}
	case "extracted":
		return shared.StatusBadgeVM{Label: "Extracted", Color: "green", Icon: "check-circle"}
	default:
		if strings.TrimSpace(value) == "" {
			return shared.StatusBadgeVM{Label: "Missing", Color: "red", Icon: "alert-triangle"}
		}
		return shared.StatusBadgeVM{Label: "Extracted", Color: "green", Icon: "check-circle"}
	}
}

func sourceCountsFromFields(fields []IntakeExtractedFieldInput) (int, int, int) {
	required := 0
	present := 0
	for _, field := range fields {
		if field.Required {
			required++
		}
		if strings.TrimSpace(field.Value) != "" && normalizeStatus(field.Status) != "missing" {
			present++
		}
	}
	if required == 0 {
		required = len(fields)
	}
	missing := required - present
	if missing < 0 {
		missing = 0
	}
	return required, present, missing
}

func missingFields(fields []IntakeExtractedFieldInput) int {
	_, _, missing := sourceCountsFromFields(fields)
	return missing
}

func normalizeSourceCounts(required, present, missing int) (int, int, int) {
	if required < 0 {
		required = 0
	}
	if present < 0 {
		present = 0
	}
	if missing < 0 {
		missing = 0
	}
	if required == 0 {
		required = present + missing
	}
	if missing == 0 && required > present {
		missing = required - present
	}
	return required, present, missing
}

func queueStatus(needsReview, warnings int) string {
	if warnings > 0 {
		return "blocked"
	}
	if needsReview > 0 {
		return "review"
	}
	return "ready"
}

func reviewMetricStatus(needsReview int) string {
	if needsReview > 0 {
		return "review"
	}
	return "ready"
}

func linkedMetricStatus(linked, total int) string {
	if total == 0 || linked < total {
		return "review"
	}
	return "ready"
}

func completenessStatus(missing int) string {
	if missing > 0 {
		return "review"
	}
	return "ready"
}

func priorityForStatus(status string, missing int) string {
	normalized := normalizeStatus(status)
	if normalized == "blocked" || normalized == "critical" {
		return "high"
	}
	if missing > 0 {
		return "medium"
	}
	return "low"
}

func priorityForCandidate(candidate IntakeReviewInput) string {
	if len(candidate.Warnings) > 0 || normalizeStatus(candidate.ReviewStatus) == "needs_review" {
		return "high"
	}
	if candidate.Confidence > 0 && candidate.Confidence < 0.8 {
		return "medium"
	}
	return "low"
}

func serviceForBusinessObject(objectType string) string {
	normalized := normalizeStatus(objectType)
	switch normalized {
	case "invoice", "supplier_invoice", "bank_statement":
		return "finance.review_link"
	case "rfq", "quotation", "opportunity":
		return "crm.review_link"
	case "purchase_order", "delivery_note", "order":
		return "operations.review_link"
	default:
		return "documents.review_link"
	}
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	status = strings.ReplaceAll(status, "-", "_")
	status = strings.ReplaceAll(status, " ", "_")
	return status
}

func formatPercent(value float64) string {
	if value <= 0 {
		return "not scored"
	}
	return fmt.Sprintf("%.0f%%", math.Round(clamp01(value)*100))
}

func firstPositive(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

// PDFPreviewVM is the display contract for PDF preview workflows.
type PDFPreviewVM struct {
	DocumentID       string            `json:"documentId"`
	Title            string            `json:"title"`
	PageCount        int               `json:"pageCount"`
	CurrentPage      int               `json:"currentPage"`
	PreviewURL       string            `json:"previewUrl"`
	ThumbnailURLs    []string          `json:"thumbnailUrls,omitempty"`
	DownloadFileName string            `json:"downloadFileName"`
	Actions          []vm.ActionButton `json:"actions"`
}
