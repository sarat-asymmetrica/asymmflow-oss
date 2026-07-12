package intake

import (
	"fmt"
	"sort"
	"strings"

	"ph_holdings_app/pkg/kernel/text"
)

const (
	AgentActionInspect         = "inspect"
	AgentActionExplain         = "explain"
	AgentActionDraft           = "draft"
	AgentActionRecommend       = "recommend"
	AgentActionAssembleContext = "assemble context"

	ForbiddenAgentActionApprove = "approve"
	ForbiddenAgentActionLink    = "link"
	ForbiddenAgentActionPost    = "post"
	ForbiddenAgentActionDelete  = "delete"
	ForbiddenAgentActionCreate  = "create authoritative business records"
)

var DefaultAllowedAgentActions = []string{
	AgentActionInspect,
	AgentActionExplain,
	AgentActionDraft,
	AgentActionRecommend,
	AgentActionAssembleContext,
}

var DefaultForbiddenAgentActions = []string{
	ForbiddenAgentActionApprove,
	ForbiddenAgentActionLink,
	ForbiddenAgentActionPost,
	ForbiddenAgentActionDelete,
	ForbiddenAgentActionCreate,
}

type ContextPack struct {
	CandidateID                          string           `json:"candidate_id"`
	SourceSummary                        string           `json:"source_summary"`
	SourceKind                           SourceKind       `json:"source_kind"`
	BusinessObjectType                   string           `json:"business_object_type"`
	Classification                       Classification   `json:"classification"`
	ExtractedFields                      []ExtractedField `json:"extracted_fields"`
	MissingFields                        []string         `json:"missing_fields"`
	SuggestedDeterministicServiceTargets []string         `json:"suggested_deterministic_service_targets"`
	ReviewStatus                         ReviewStatus     `json:"review_status"`
	Warnings                             []string         `json:"warnings,omitempty"`
	AuditRefs                            []AuditRef       `json:"audit_refs"`
	AllowedAgentActions                  []string         `json:"allowed_agent_actions"`
	ForbiddenAgentActions                []string         `json:"forbidden_agent_actions"`
}

func BuildContextPack(candidate Candidate) ContextPack {
	candidate = normalizeCandidate(candidate, Options{})
	return ContextPack{
		CandidateID:                          candidate.ID,
		SourceSummary:                        sourceSummary(candidate.Source),
		SourceKind:                           candidate.SourceKind,
		BusinessObjectType:                   candidate.BusinessObjectType,
		Classification:                       candidate.Classification,
		ExtractedFields:                      append([]ExtractedField(nil), candidate.ExtractedFields...),
		MissingFields:                        missingFieldNames(candidate.ExtractedFields),
		SuggestedDeterministicServiceTargets: deterministicServiceTargets(candidate.SuggestedLinks),
		ReviewStatus:                         candidate.ReviewStatus,
		Warnings:                             append([]string(nil), candidate.Warnings...),
		AuditRefs:                            append([]AuditRef(nil), candidate.AuditRefs...),
		AllowedAgentActions:                  append([]string(nil), DefaultAllowedAgentActions...),
		ForbiddenAgentActions:                append([]string(nil), DefaultForbiddenAgentActions...),
	}
}

func FormatContextPackTOON(pack ContextPack) string {
	var b strings.Builder
	writeLine(&b, "business_memory_intake_context:")
	writeLine(&b, "  candidate_id: %s", pack.CandidateID)
	writeLine(&b, "  source_summary: %s", pack.SourceSummary)
	writeLine(&b, "  source_kind: %s", pack.SourceKind)
	writeLine(&b, "  business_object_type: %s", pack.BusinessObjectType)
	writeLine(&b, "  classification: %s", text.FirstNonEmpty(pack.Classification.Type, "unknown"))
	writeLine(&b, "  confidence: %.2f", pack.Classification.Confidence)
	writeLine(&b, "  review_status: %s", pack.ReviewStatus)
	writeStringList(&b, "  missing_fields", pack.MissingFields)
	writeStringList(&b, "  suggested_deterministic_service_targets", pack.SuggestedDeterministicServiceTargets)
	writeFields(&b, pack.ExtractedFields)
	writeAuditRefs(&b, pack.AuditRefs)
	writeStringList(&b, "  warnings", pack.Warnings)
	writeStringList(&b, "  allowed_agent_actions", pack.AllowedAgentActions)
	writeStringList(&b, "  forbidden_agent_actions", pack.ForbiddenAgentActions)
	return strings.TrimRight(b.String(), "\n")
}

func BuildContextPackTOON(candidate Candidate) string {
	return FormatContextPackTOON(BuildContextPack(candidate))
}

func sourceSummary(source SourceRef) string {
	parts := []string{}
	if source.Label != "" {
		parts = append(parts, source.Label)
	}
	if source.Path != "" && source.Path != source.Label {
		parts = append(parts, source.Path)
	}
	if source.ID != "" && source.ID != source.Label && source.ID != source.Path {
		parts = append(parts, source.ID)
	}
	if len(parts) == 0 {
		return "unlabeled intake source"
	}
	return strings.Join(parts, " | ")
}

func missingFieldNames(fields []ExtractedField) []string {
	var missing []string
	for _, field := range fields {
		if field.Status == FieldStatusMissing || strings.TrimSpace(field.Value) == "" {
			missing = append(missing, text.FirstNonEmpty(field.Label, field.Name))
		}
	}
	return uniqueStrings(missing)
}

func deterministicServiceTargets(links []SuggestedLink) []string {
	targets := make([]string, 0, len(links))
	for _, link := range links {
		target := strings.TrimSpace(link.RequiredDeterministicService)
		if target != "" {
			targets = append(targets, target)
		}
	}
	return uniqueStrings(targets)
}

func writeLine(b *strings.Builder, format string, args ...any) {
	if len(args) == 0 {
		b.WriteString(format)
	} else {
		b.WriteString(fmt.Sprintf(format, args...))
	}
	b.WriteByte('\n')
}

func writeStringList(b *strings.Builder, label string, values []string) {
	values = uniqueStrings(values)
	writeLine(b, "%s:", label)
	if len(values) == 0 {
		writeLine(b, "    - none")
		return
	}
	for _, value := range values {
		writeLine(b, "    - %s", value)
	}
}

func writeFields(b *strings.Builder, fields []ExtractedField) {
	writeLine(b, "  extracted_fields:")
	if len(fields) == 0 {
		writeLine(b, "    - none")
		return
	}
	fields = append([]ExtractedField(nil), fields...)
	sort.SliceStable(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	for _, field := range fields {
		writeLine(b, "    - name: %s", text.FirstNonEmpty(field.Name, "field"))
		writeLine(b, "      value: %s", text.FirstNonEmpty(field.Value, "missing"))
		writeLine(b, "      status: %s", field.Status)
		if field.Source != "" {
			writeLine(b, "      source: %s", field.Source)
		}
	}
}

func writeAuditRefs(b *strings.Builder, refs []AuditRef) {
	writeLine(b, "  audit_refs:")
	if len(refs) == 0 {
		writeLine(b, "    - none")
		return
	}
	refs = append([]AuditRef(nil), refs...)
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Type == refs[j].Type {
			return refs[i].SourceID < refs[j].SourceID
		}
		return refs[i].Type < refs[j].Type
	})
	for _, ref := range refs {
		writeLine(b, "    - type: %s", text.FirstNonEmpty(ref.Type, "source"))
		writeLine(b, "      source_id: %s", text.FirstNonEmpty(ref.SourceID, "unknown"))
		if ref.Summary != "" {
			writeLine(b, "      summary: %s", ref.Summary)
		}
	}
}
