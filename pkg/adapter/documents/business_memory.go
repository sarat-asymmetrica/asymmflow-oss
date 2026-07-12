package documents

import (
	"ph_holdings_app/pkg/adapter"
	"ph_holdings_app/pkg/documents/intake"
	protodocuments "ph_holdings_app/schemas/go/documents"

	capnp "capnproto.org/go/capnp/v3"
)

func BusinessMemoryCandidateToProto(m intake.Candidate) (*protodocuments.BusinessMemoryCandidate, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protodocuments.NewBusinessMemoryCandidate(seg)
	if err != nil {
		return nil, err
	}
	if err := populateBusinessMemoryCandidate(seg, p, m); err != nil {
		return nil, err
	}
	return &p, nil
}

func BusinessMemoryCandidateFromProto(p protodocuments.BusinessMemoryCandidate) (intake.Candidate, error) {
	m := intake.Candidate{}
	m.ID, _ = p.Id()
	source, err := p.Source()
	if err == nil && source.IsValid() {
		m.Source = businessMemorySourceRefFromProto(source)
	}
	m.SourceKind = sourceKindFromProto(p.SourceKind())
	m.BusinessObjectType, _ = p.BusinessObjectType()
	classification, err := p.Classification()
	if err == nil && classification.IsValid() {
		m.Classification = businessMemoryClassificationFromProto(classification)
	}
	if fields, err := p.ExtractedFields(); err == nil {
		m.ExtractedFields = businessMemoryExtractedFieldsFromProto(fields)
	}
	if links, err := p.SuggestedLinks(); err == nil {
		m.SuggestedLinks = businessMemorySuggestedLinksFromProto(links)
	}
	m.ReviewStatus = reviewStatusFromProto(p.ReviewStatus())
	if refs, err := p.AuditRefs(); err == nil {
		m.AuditRefs = businessMemoryAuditRefsFromProto(refs)
	}
	m.Confidence = p.Confidence()
	if warnings, err := p.Warnings(); err == nil {
		m.Warnings = textListToStrings(warnings)
	}
	return m, nil
}

func BusinessMemoryContextPackToProto(m intake.ContextPack) (*protodocuments.BusinessMemoryContextPack, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protodocuments.NewBusinessMemoryContextPack(seg)
	if err != nil {
		return nil, err
	}
	if err := populateBusinessMemoryContextPack(seg, p, m); err != nil {
		return nil, err
	}
	return &p, nil
}

func BusinessMemoryContextPackFromProto(p protodocuments.BusinessMemoryContextPack) (intake.ContextPack, error) {
	m := intake.ContextPack{}
	m.CandidateID, _ = p.CandidateId()
	m.SourceSummary, _ = p.SourceSummary()
	m.SourceKind = sourceKindFromProto(p.SourceKind())
	m.BusinessObjectType, _ = p.BusinessObjectType()
	classification, err := p.Classification()
	if err == nil && classification.IsValid() {
		m.Classification = businessMemoryClassificationFromProto(classification)
	}
	if fields, err := p.ExtractedFields(); err == nil {
		m.ExtractedFields = businessMemoryExtractedFieldsFromProto(fields)
	}
	if missing, err := p.MissingFields(); err == nil {
		m.MissingFields = textListToStrings(missing)
	}
	if targets, err := p.SuggestedDeterministicServiceTargets(); err == nil {
		m.SuggestedDeterministicServiceTargets = textListToStrings(targets)
	}
	m.ReviewStatus = reviewStatusFromProto(p.ReviewStatus())
	if warnings, err := p.Warnings(); err == nil {
		m.Warnings = textListToStrings(warnings)
	}
	if refs, err := p.AuditRefs(); err == nil {
		m.AuditRefs = businessMemoryAuditRefsFromProto(refs)
	}
	if actions, err := p.AllowedAgentActions(); err == nil {
		m.AllowedAgentActions = textListToStrings(actions)
	}
	if actions, err := p.ForbiddenAgentActions(); err == nil {
		m.ForbiddenAgentActions = textListToStrings(actions)
	}
	return m, nil
}

func BusinessMemoryReviewRecordToProto(m intake.ReviewRecord) (*protodocuments.BusinessMemoryReviewRecord, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protodocuments.NewBusinessMemoryReviewRecord(seg)
	if err != nil {
		return nil, err
	}
	for _, err := range []error{
		p.SetId(m.ID),
		p.SetCandidateId(m.CandidateID),
		p.SetSourceId(m.SourceID),
		p.SetProposedDeterministicService(m.ProposedDeterministicService),
		p.SetActor(m.Actor),
		p.SetReason(m.Reason),
		p.SetCorrelationId(m.CorrelationID),
		p.SetCreatedAt(adapter.TimeToText(m.CreatedAt)),
	} {
		if err != nil {
			return nil, err
		}
	}
	p.SetDecision(reviewDecisionToProto(m.Decision))
	p.SetReviewStatus(reviewStatusToProto(m.ReviewStatus))
	return &p, nil
}

func BusinessMemoryReviewRecordFromProto(p protodocuments.BusinessMemoryReviewRecord) (intake.ReviewRecord, error) {
	m := intake.ReviewRecord{}
	m.ID, _ = p.Id()
	m.CandidateID, _ = p.CandidateId()
	m.SourceID, _ = p.SourceId()
	m.Decision = reviewDecisionFromProto(p.Decision())
	m.ReviewStatus = reviewStatusFromProto(p.ReviewStatus())
	m.ProposedDeterministicService, _ = p.ProposedDeterministicService()
	m.Actor, _ = p.Actor()
	m.Reason, _ = p.Reason()
	m.CorrelationID, _ = p.CorrelationId()
	if createdAt, err := p.CreatedAt(); err == nil {
		m.CreatedAt = adapter.TextToTime(createdAt)
	}
	return m, nil
}

func populateBusinessMemoryCandidate(seg *capnp.Segment, p protodocuments.BusinessMemoryCandidate, m intake.Candidate) error {
	for _, err := range []error{
		p.SetId(m.ID),
		p.SetBusinessObjectType(m.BusinessObjectType),
	} {
		if err != nil {
			return err
		}
	}
	p.SetSourceKind(sourceKindToProto(m.SourceKind))
	p.SetReviewStatus(reviewStatusToProto(m.ReviewStatus))
	p.SetConfidence(m.Confidence)

	source, err := p.NewSource()
	if err != nil {
		return err
	}
	if err := populateBusinessMemorySourceRef(source, m.Source); err != nil {
		return err
	}
	classification, err := p.NewClassification()
	if err != nil {
		return err
	}
	if err := populateBusinessMemoryClassification(seg, classification, m.Classification); err != nil {
		return err
	}
	if err := setBusinessMemoryExtractedFields(seg, p.NewExtractedFields, m.ExtractedFields); err != nil {
		return err
	}
	if err := setBusinessMemorySuggestedLinks(seg, p.NewSuggestedLinks, m.SuggestedLinks); err != nil {
		return err
	}
	if err := setBusinessMemoryAuditRefs(seg, p.NewAuditRefs, m.AuditRefs); err != nil {
		return err
	}
	return setTextList(seg, p.SetWarnings, m.Warnings)
}

func populateBusinessMemoryContextPack(seg *capnp.Segment, p protodocuments.BusinessMemoryContextPack, m intake.ContextPack) error {
	for _, err := range []error{
		p.SetCandidateId(m.CandidateID),
		p.SetSourceSummary(m.SourceSummary),
		p.SetBusinessObjectType(m.BusinessObjectType),
	} {
		if err != nil {
			return err
		}
	}
	p.SetSourceKind(sourceKindToProto(m.SourceKind))
	p.SetReviewStatus(reviewStatusToProto(m.ReviewStatus))

	classification, err := p.NewClassification()
	if err != nil {
		return err
	}
	if err := populateBusinessMemoryClassification(seg, classification, m.Classification); err != nil {
		return err
	}
	if err := setBusinessMemoryExtractedFields(seg, p.NewExtractedFields, m.ExtractedFields); err != nil {
		return err
	}
	for _, err := range []error{
		setTextList(seg, p.SetMissingFields, m.MissingFields),
		setTextList(seg, p.SetSuggestedDeterministicServiceTargets, m.SuggestedDeterministicServiceTargets),
		setTextList(seg, p.SetWarnings, m.Warnings),
		setBusinessMemoryAuditRefs(seg, p.NewAuditRefs, m.AuditRefs),
		setTextList(seg, p.SetAllowedAgentActions, m.AllowedAgentActions),
		setTextList(seg, p.SetForbiddenAgentActions, m.ForbiddenAgentActions),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

func populateBusinessMemorySourceRef(p protodocuments.BusinessMemorySourceRef, m intake.SourceRef) error {
	for _, err := range []error{
		p.SetId(m.ID),
		p.SetLabel(m.Label),
		p.SetPath(m.Path),
		p.SetProcessedAt(adapter.TimePtrToText(m.ProcessedAt)),
	} {
		if err != nil {
			return err
		}
	}
	p.SetKind(sourceKindToProto(m.Kind))
	return nil
}

func businessMemorySourceRefFromProto(p protodocuments.BusinessMemorySourceRef) intake.SourceRef {
	m := intake.SourceRef{Kind: sourceKindFromProto(p.Kind())}
	m.ID, _ = p.Id()
	m.Label, _ = p.Label()
	m.Path, _ = p.Path()
	if processedAt, err := p.ProcessedAt(); err == nil {
		m.ProcessedAt = adapter.TextToTimePtr(processedAt)
	}
	return m
}

func populateBusinessMemoryClassification(seg *capnp.Segment, p protodocuments.BusinessMemoryClassification, m intake.Classification) error {
	for _, err := range []error{
		p.SetType(m.Type),
		p.SetMethod(m.Method),
		p.SetRouteTo(m.RouteTo),
		p.SetReason(m.Reason),
		setTextList(seg, p.SetKeywords, m.Keywords),
	} {
		if err != nil {
			return err
		}
	}
	p.SetConfidence(m.Confidence)
	return nil
}

func businessMemoryClassificationFromProto(p protodocuments.BusinessMemoryClassification) intake.Classification {
	m := intake.Classification{Confidence: p.Confidence()}
	m.Type, _ = p.Type()
	m.Method, _ = p.Method()
	m.RouteTo, _ = p.RouteTo()
	m.Reason, _ = p.Reason()
	if keywords, err := p.Keywords(); err == nil {
		m.Keywords = textListToStrings(keywords)
	}
	return m
}

func setBusinessMemoryExtractedFields(seg *capnp.Segment, newList func(int32) (protodocuments.BusinessMemoryExtractedField_List, error), fields []intake.ExtractedField) error {
	list, err := newList(int32(len(fields)))
	if err != nil {
		return err
	}
	for i, field := range fields {
		p, err := protodocuments.NewBusinessMemoryExtractedField(seg)
		if err != nil {
			return err
		}
		for _, err := range []error{p.SetName(field.Name), p.SetLabel(field.Label), p.SetValue(field.Value), p.SetSource(field.Source)} {
			if err != nil {
				return err
			}
		}
		p.SetStatus(fieldStatusToProto(field.Status))
		p.SetConfidence(field.Confidence)
		if err := list.Set(i, p); err != nil {
			return err
		}
	}
	return nil
}

func businessMemoryExtractedFieldsFromProto(list protodocuments.BusinessMemoryExtractedField_List) []intake.ExtractedField {
	fields := make([]intake.ExtractedField, list.Len())
	for i := 0; i < list.Len(); i++ {
		p := list.At(i)
		fields[i].Name, _ = p.Name()
		fields[i].Label, _ = p.Label()
		fields[i].Value, _ = p.Value()
		fields[i].Status = fieldStatusFromProto(p.Status())
		fields[i].Confidence = p.Confidence()
		fields[i].Source, _ = p.Source()
	}
	return fields
}

func setBusinessMemorySuggestedLinks(seg *capnp.Segment, newList func(int32) (protodocuments.BusinessMemorySuggestedLink_List, error), links []intake.SuggestedLink) error {
	list, err := newList(int32(len(links)))
	if err != nil {
		return err
	}
	for i, link := range links {
		p, err := protodocuments.NewBusinessMemorySuggestedLink(seg)
		if err != nil {
			return err
		}
		for _, err := range []error{p.SetId(link.ID), p.SetLabel(link.Label), p.SetReason(link.Reason), p.SetBusinessObjectType(link.BusinessObjectType), p.SetRequiredDeterministicService(link.RequiredDeterministicService)} {
			if err != nil {
				return err
			}
		}
		if err := list.Set(i, p); err != nil {
			return err
		}
	}
	return nil
}

func businessMemorySuggestedLinksFromProto(list protodocuments.BusinessMemorySuggestedLink_List) []intake.SuggestedLink {
	links := make([]intake.SuggestedLink, list.Len())
	for i := 0; i < list.Len(); i++ {
		p := list.At(i)
		links[i].ID, _ = p.Id()
		links[i].Label, _ = p.Label()
		links[i].Reason, _ = p.Reason()
		links[i].BusinessObjectType, _ = p.BusinessObjectType()
		links[i].RequiredDeterministicService, _ = p.RequiredDeterministicService()
	}
	return links
}

func setBusinessMemoryAuditRefs(seg *capnp.Segment, newList func(int32) (protodocuments.BusinessMemoryAuditRef_List, error), refs []intake.AuditRef) error {
	list, err := newList(int32(len(refs)))
	if err != nil {
		return err
	}
	for i, ref := range refs {
		p, err := protodocuments.NewBusinessMemoryAuditRef(seg)
		if err != nil {
			return err
		}
		for _, err := range []error{p.SetType(ref.Type), p.SetSourceId(ref.SourceID), p.SetSummary(ref.Summary), p.SetTimestamp(ref.Timestamp)} {
			if err != nil {
				return err
			}
		}
		if err := list.Set(i, p); err != nil {
			return err
		}
	}
	return nil
}

func businessMemoryAuditRefsFromProto(list protodocuments.BusinessMemoryAuditRef_List) []intake.AuditRef {
	refs := make([]intake.AuditRef, list.Len())
	for i := 0; i < list.Len(); i++ {
		p := list.At(i)
		refs[i].Type, _ = p.Type()
		refs[i].SourceID, _ = p.SourceId()
		refs[i].Summary, _ = p.Summary()
		refs[i].Timestamp, _ = p.Timestamp()
	}
	return refs
}

func setTextList(seg *capnp.Segment, setter func(capnp.TextList) error, values []string) error {
	list, err := textList(seg, values)
	if err != nil {
		return err
	}
	return setter(list)
}

func textListToStrings(list capnp.TextList) []string {
	values := make([]string, 0, list.Len())
	for i := 0; i < list.Len(); i++ {
		value, err := list.At(i)
		if err == nil {
			values = append(values, value)
		}
	}
	return values
}

func sourceKindToProto(kind intake.SourceKind) protodocuments.BusinessMemorySourceKind {
	switch kind {
	case intake.SourceKindMessage:
		return protodocuments.BusinessMemorySourceKind_message
	case intake.SourceKindEmail:
		return protodocuments.BusinessMemorySourceKind_email
	case intake.SourceKindPDF:
		return protodocuments.BusinessMemorySourceKind_pdf
	case intake.SourceKindScan:
		return protodocuments.BusinessMemorySourceKind_scan
	case intake.SourceKindScreenshot:
		return protodocuments.BusinessMemorySourceKind_screenshot
	case intake.SourceKindExcel:
		return protodocuments.BusinessMemorySourceKind_excel
	case intake.SourceKindFolder:
		return protodocuments.BusinessMemorySourceKind_folder
	case intake.SourceKindInboxRecord:
		return protodocuments.BusinessMemorySourceKind_inboxRecord
	default:
		return protodocuments.BusinessMemorySourceKind_other
	}
}

func sourceKindFromProto(kind protodocuments.BusinessMemorySourceKind) intake.SourceKind {
	switch kind {
	case protodocuments.BusinessMemorySourceKind_message:
		return intake.SourceKindMessage
	case protodocuments.BusinessMemorySourceKind_email:
		return intake.SourceKindEmail
	case protodocuments.BusinessMemorySourceKind_pdf:
		return intake.SourceKindPDF
	case protodocuments.BusinessMemorySourceKind_scan:
		return intake.SourceKindScan
	case protodocuments.BusinessMemorySourceKind_screenshot:
		return intake.SourceKindScreenshot
	case protodocuments.BusinessMemorySourceKind_excel:
		return intake.SourceKindExcel
	case protodocuments.BusinessMemorySourceKind_folder:
		return intake.SourceKindFolder
	case protodocuments.BusinessMemorySourceKind_inboxRecord:
		return intake.SourceKindInboxRecord
	default:
		return intake.SourceKindOther
	}
}

func reviewStatusToProto(status intake.ReviewStatus) protodocuments.BusinessMemoryReviewStatus {
	switch status {
	case intake.ReviewStatusNeedsReview:
		return protodocuments.BusinessMemoryReviewStatus_needsReview
	case intake.ReviewStatusCorrected:
		return protodocuments.BusinessMemoryReviewStatus_corrected
	case intake.ReviewStatusLinked:
		return protodocuments.BusinessMemoryReviewStatus_linked
	case intake.ReviewStatusRejected:
		return protodocuments.BusinessMemoryReviewStatus_rejected
	case intake.ReviewStatusArchived:
		return protodocuments.BusinessMemoryReviewStatus_archived
	default:
		return protodocuments.BusinessMemoryReviewStatus_new
	}
}

func reviewStatusFromProto(status protodocuments.BusinessMemoryReviewStatus) intake.ReviewStatus {
	switch status {
	case protodocuments.BusinessMemoryReviewStatus_needsReview:
		return intake.ReviewStatusNeedsReview
	case protodocuments.BusinessMemoryReviewStatus_corrected:
		return intake.ReviewStatusCorrected
	case protodocuments.BusinessMemoryReviewStatus_linked:
		return intake.ReviewStatusLinked
	case protodocuments.BusinessMemoryReviewStatus_rejected:
		return intake.ReviewStatusRejected
	case protodocuments.BusinessMemoryReviewStatus_archived:
		return intake.ReviewStatusArchived
	default:
		return intake.ReviewStatusNew
	}
}

func fieldStatusToProto(status intake.FieldStatus) protodocuments.BusinessMemoryFieldStatus {
	switch status {
	case intake.FieldStatusMissing:
		return protodocuments.BusinessMemoryFieldStatus_missing
	case intake.FieldStatusInferred:
		return protodocuments.BusinessMemoryFieldStatus_inferred
	case intake.FieldStatusNeedsConfirmation:
		return protodocuments.BusinessMemoryFieldStatus_needsConfirmation
	case intake.FieldStatusCorrected:
		return protodocuments.BusinessMemoryFieldStatus_corrected
	default:
		return protodocuments.BusinessMemoryFieldStatus_extracted
	}
}

func fieldStatusFromProto(status protodocuments.BusinessMemoryFieldStatus) intake.FieldStatus {
	switch status {
	case protodocuments.BusinessMemoryFieldStatus_missing:
		return intake.FieldStatusMissing
	case protodocuments.BusinessMemoryFieldStatus_inferred:
		return intake.FieldStatusInferred
	case protodocuments.BusinessMemoryFieldStatus_needsConfirmation:
		return intake.FieldStatusNeedsConfirmation
	case protodocuments.BusinessMemoryFieldStatus_corrected:
		return intake.FieldStatusCorrected
	default:
		return intake.FieldStatusExtracted
	}
}

func reviewDecisionToProto(decision intake.ReviewDecision) protodocuments.BusinessMemoryReviewDecision {
	switch decision {
	case intake.ReviewDecisionNeedsInput:
		return protodocuments.BusinessMemoryReviewDecision_needsInput
	case intake.ReviewDecisionCorrectField:
		return protodocuments.BusinessMemoryReviewDecision_correctField
	case intake.ReviewDecisionRejectCandidate:
		return protodocuments.BusinessMemoryReviewDecision_rejectCandidate
	case intake.ReviewDecisionArchive:
		return protodocuments.BusinessMemoryReviewDecision_archive
	default:
		return protodocuments.BusinessMemoryReviewDecision_acceptProposal
	}
}

func reviewDecisionFromProto(decision protodocuments.BusinessMemoryReviewDecision) intake.ReviewDecision {
	switch decision {
	case protodocuments.BusinessMemoryReviewDecision_needsInput:
		return intake.ReviewDecisionNeedsInput
	case protodocuments.BusinessMemoryReviewDecision_correctField:
		return intake.ReviewDecisionCorrectField
	case protodocuments.BusinessMemoryReviewDecision_rejectCandidate:
		return intake.ReviewDecisionRejectCandidate
	case protodocuments.BusinessMemoryReviewDecision_archive:
		return intake.ReviewDecisionArchive
	default:
		return intake.ReviewDecisionAcceptProposal
	}
}
