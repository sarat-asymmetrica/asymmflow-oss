package evidence

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ph_holdings_app/pkg/toon"
)

const EvidencePackSchemaVersion = "cashflow_evidence_pack.v0.1"

type EvidencePack struct {
	SchemaVersion       string                   `json:"schema_version"`
	ModuleID            string                   `json:"module_id"`
	GeneratedAt         time.Time                `json:"generated_at"`
	Mode                string                   `json:"mode"`
	MutatesState        bool                     `json:"mutates_state"`
	CommandCenter       CommandCenter            `json:"command_center"`
	AgentBrief          AgentBrief               `json:"agent_brief"`
	SourceSummary       []EvidencePackSource     `json:"source_summary"`
	AllocationSummary   AllocationSummary        `json:"allocation_summary"`
	BankAllocations     []EvidencePackAllocation `json:"bank_allocations"`
	ActionProposals     []ActionProposal         `json:"action_proposals"`
	DeterministicHints  []string                 `json:"deterministic_hints"`
	ForbiddenOperations []string                 `json:"forbidden_operations"`
	ExportMetadata      map[string]any           `json:"export_metadata"`
}

type EvidencePackSource struct {
	SourceType string   `json:"source_type"`
	Label      string   `json:"label"`
	Required   int      `json:"required"`
	Present    int      `json:"present"`
	Missing    int      `json:"missing"`
	Status     Status   `json:"status"`
	Priority   Priority `json:"priority"`
	Confidence float64  `json:"confidence"`
}

type EvidencePackAllocation struct {
	AllocationID        string   `json:"allocation_id"`
	BankStatementLineID string   `json:"bank_statement_line_id"`
	SourceType          string   `json:"source_type"`
	SourceID            string   `json:"source_id"`
	Amount              float64  `json:"amount"`
	AllocationType      string   `json:"allocation_type"`
	AllocationStatus    string   `json:"allocation_status"`
	Status              Status   `json:"status"`
	Priority            Priority `json:"priority"`
	Confidence          float64  `json:"confidence"`
}

func BuildEvidencePack(center CommandCenter, generatedAt time.Time) EvidencePack {
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	brief := BuildAgentBrief(center)
	sources := make([]EvidencePackSource, 0, len(center.EvidenceSources))
	for _, source := range center.EvidenceSources {
		sources = append(sources, EvidencePackSource{
			SourceType: source.SourceType,
			Label:      source.Label,
			Required:   source.Required,
			Present:    source.Present,
			Missing:    source.Missing,
			Status:     source.Status,
			Priority:   source.Priority,
			Confidence: source.Confidence,
		})
	}
	allocations := make([]EvidencePackAllocation, 0, len(center.BankAllocations))
	for _, allocation := range center.BankAllocations {
		allocations = append(allocations, EvidencePackAllocation{
			AllocationID:        allocation.AllocationID,
			BankStatementLineID: allocation.BankStatementLineID,
			SourceType:          allocation.SourceType,
			SourceID:            allocation.SourceID,
			Amount:              allocation.Amount,
			AllocationType:      allocation.AllocationType,
			AllocationStatus:    allocation.AllocationStatus,
			Status:              allocation.Status,
			Priority:            allocation.Priority,
			Confidence:          allocation.Confidence,
		})
	}

	return EvidencePack{
		SchemaVersion:       EvidencePackSchemaVersion,
		ModuleID:            "cashflow_evidence",
		GeneratedAt:         generatedAt,
		Mode:                brief.Mode,
		MutatesState:        false,
		CommandCenter:       center,
		AgentBrief:          brief,
		SourceSummary:       sources,
		AllocationSummary:   center.AllocationSummary,
		BankAllocations:     allocations,
		ActionProposals:     append([]ActionProposal(nil), center.ActionProposals...),
		DeterministicHints:  append([]string(nil), brief.DeterministicCommandHints...),
		ForbiddenOperations: append([]string(nil), brief.ForbiddenOperations...),
		ExportMetadata: map[string]any{
			"format":          "json",
			"epistemic_state": "read_model_snapshot",
			"authority":       "deterministic_finance_read_service",
		},
	}
}

func MarshalEvidencePackJSON(center CommandCenter, generatedAt time.Time) ([]byte, error) {
	return json.MarshalIndent(BuildEvidencePack(center, generatedAt), "", "  ")
}

func MarshalEvidencePackTOON(center CommandCenter, generatedAt time.Time, maxChars int) (string, error) {
	encoded, err := toon.Marshal(BuildEvidencePack(center, generatedAt))
	if err != nil {
		return "", err
	}
	out := "format: TOON\n" + encoded
	if maxChars > 0 && len(out) > maxChars {
		return strings.TrimSpace(out[:maxChars]) + fmt.Sprintf("\n... [cashflow evidence pack truncated at %d chars]", maxChars), nil
	}
	return strings.TrimSpace(out), nil
}
