package evidence

import (
	"fmt"
	"strings"

	"ph_holdings_app/pkg/toon"
)

var forbiddenAgentOperations = []string{"approve", "post", "persist", "delete", "reverse", "file_tax_return"}

type AgentBrief struct {
	Mode                      string             `json:"mode"`
	MutatesState              bool               `json:"mutates_state"`
	OverallStatus             Status             `json:"overall_status"`
	NextDeterministicAction   string             `json:"next_deterministic_action"`
	ForbiddenOperations       []string           `json:"forbidden_operations"`
	Cash                      CashExposure       `json:"cash"`
	Posting                   PostingReadiness   `json:"posting"`
	EvidenceSources           []AgentSourceBrief `json:"evidence_sources"`
	AllocationSummary         AllocationSummary  `json:"allocation_summary"`
	UnmatchedBankLines        int                `json:"unmatched_bank_lines"`
	ExportableAuditItems      int                `json:"exportable_audit_items"`
	ActionProposals           []ActionProposal   `json:"action_proposals"`
	DeterministicCommandHints []string           `json:"deterministic_command_hints"`
}

type AgentSourceBrief struct {
	SourceType string `json:"source_type"`
	Label      string `json:"label"`
	Missing    int    `json:"missing"`
	Status     Status `json:"status"`
}

func BuildAgentBrief(center CommandCenter) AgentBrief {
	sources := make([]AgentSourceBrief, 0, len(center.EvidenceSources))
	for _, source := range center.EvidenceSources {
		sources = append(sources, AgentSourceBrief{
			SourceType: source.SourceType,
			Label:      source.Label,
			Missing:    source.Missing,
			Status:     source.Status,
		})
	}

	return AgentBrief{
		Mode:                    "inspect_explain_draft_recommend",
		MutatesState:            false,
		OverallStatus:           center.OverallStatus,
		NextDeterministicAction: center.NextAction,
		ForbiddenOperations:     append([]string(nil), forbiddenAgentOperations...),
		Cash:                    center.Cash,
		Posting:                 center.Posting,
		EvidenceSources:         sources,
		AllocationSummary:       center.AllocationSummary,
		UnmatchedBankLines:      center.UnmatchedBankLines,
		ExportableAuditItems:    center.ExportableAuditItems,
		ActionProposals:         append([]ActionProposal(nil), center.ActionProposals...),
		DeterministicCommandHints: []string{
			"cashflowEvidence.inspectSources",
			"cashflowEvidence.draftFollowUp",
			"cashflowEvidence.exportPack",
			"finance.requestDraftJournal",
		},
	}
}

func MarshalAgentBriefTOON(center CommandCenter, maxChars int) (string, error) {
	encoded, err := toon.Marshal(BuildAgentBrief(center))
	if err != nil {
		return "", err
	}
	out := "format: TOON\n" + encoded
	if maxChars > 0 && len(out) > maxChars {
		return strings.TrimSpace(out[:maxChars]) + fmt.Sprintf("\n... [cashflow evidence brief truncated at %d chars]", maxChars), nil
	}
	return strings.TrimSpace(out), nil
}
