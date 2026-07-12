package evidence

import (
	"strings"

	"ph_holdings_app/pkg/kernel/text"
)

type ActionProposal struct {
	Action                       string   `json:"action"`
	Label                        string   `json:"label"`
	Reason                       string   `json:"reason"`
	Priority                     Priority `json:"priority"`
	SourceType                   string   `json:"source_type"`
	MutatesState                 bool     `json:"mutates_state"`
	RequiredDeterministicService string   `json:"required_deterministic_service"`
}

func ProposalReviewKey(proposal ActionProposal) string {
	parts := []string{
		strings.TrimSpace(proposal.Action),
		strings.TrimSpace(proposal.SourceType),
		strings.TrimSpace(proposal.RequiredDeterministicService),
		strings.TrimSpace(proposal.Label),
	}
	for i, part := range parts {
		parts[i] = strings.ToLower(part)
	}
	return strings.Join(parts, "|")
}

func BuildActionProposals(center CommandCenter) []ActionProposal {
	proposals := make([]ActionProposal, 0, 5)
	for _, source := range center.EvidenceSources {
		if source.Missing <= 0 {
			continue
		}
		proposals = append(proposals, ActionProposal{
			Action:                       "cashflowEvidence.inspectSources",
			Label:                        "Inspect missing evidence",
			Reason:                       text.FirstNonEmpty(source.Label, source.SourceType, "Evidence") + " has missing source records.",
			Priority:                     source.Priority,
			SourceType:                   source.SourceType,
			MutatesState:                 false,
			RequiredDeterministicService: "documents_or_finance_source_linking",
		})
	}
	if center.Posting.MissingJournals > 0 || center.Posting.DraftEntries > 0 {
		proposals = append(proposals, ActionProposal{
			Action:                       "finance.requestDraftJournal",
			Label:                        "Review draft posting",
			Reason:                       center.Posting.Message,
			Priority:                     center.Posting.Priority,
			SourceType:                   "posting_coverage",
			MutatesState:                 false,
			RequiredDeterministicService: "finance_posting_service",
		})
	}
	if center.AllocationSummary.Unresolved > 0 || center.AllocationSummary.Conflicts > 0 {
		proposals = append(proposals, ActionProposal{
			Action:                       "cashflowEvidence.inspectAllocations",
			Label:                        "Inspect bank allocations",
			Reason:                       "Bank evidence has partial, mixed, or unresolved allocation records.",
			Priority:                     PriorityHigh,
			SourceType:                   "bank_allocation",
			MutatesState:                 false,
			RequiredDeterministicService: "bank_reconciliation_service",
		})
	}
	if center.UnmatchedBankLines > 0 {
		proposals = append(proposals, ActionProposal{
			Action:                       "banking.resolveMatch",
			Label:                        "Resolve bank matches",
			Reason:                       "Bank evidence has unmatched statement lines.",
			Priority:                     PriorityHigh,
			SourceType:                   "bank_reconciliation",
			MutatesState:                 false,
			RequiredDeterministicService: "bank_reconciliation_service",
		})
	}
	if center.Cash.Priority == PriorityHigh || center.Cash.Priority == PriorityUrgent {
		proposals = append(proposals, ActionProposal{
			Action:                       "cashflowEvidence.draftFollowUp",
			Label:                        "Draft receivables follow-up",
			Reason:                       "Receivables exposure needs operator attention.",
			Priority:                     center.Cash.Priority,
			SourceType:                   "receivables",
			MutatesState:                 false,
			RequiredDeterministicService: "task_or_collections_service",
		})
	}
	if center.ExportableAuditItems > 0 {
		proposals = append(proposals, ActionProposal{
			Action:                       "cashflowEvidence.exportPack",
			Label:                        "Export evidence pack",
			Reason:                       "Verified audit items are available for operator or accountant review.",
			Priority:                     PriorityLow,
			SourceType:                   "support_bundle",
			MutatesState:                 false,
			RequiredDeterministicService: "cashflow_evidence_export_service",
		})
	}
	return proposals
}
