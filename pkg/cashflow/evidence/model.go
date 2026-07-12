// Package evidence builds deterministic cashflow evidence command-center
// snapshots from existing finance, document, banking, and posting facts.
package evidence

import (
	"fmt"
	"strings"
	"time"

	"ph_holdings_app/pkg/finance/posting"
	"ph_holdings_app/pkg/kernel/money"
	"ph_holdings_app/pkg/kernel/text"
)

type Status string

const (
	StatusReady     Status = "ready"
	StatusAttention Status = "attention"
	StatusBlocked   Status = "blocked"
	StatusEmpty     Status = "empty"
)

type Priority string

const (
	PriorityNone   Priority = "none"
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Label string    `json:"label"`
}

type CommandCenterInput struct {
	Window               TimeWindow                `json:"window"`
	Cash                 CashExposureInput         `json:"cash"`
	EvidenceSources      []EvidenceSourceInput     `json:"evidence_sources"`
	BankAllocations      []AllocationEvidenceInput `json:"bank_allocations"`
	PostingCoverage      posting.CoverageReport    `json:"posting_coverage"`
	TrialBalanceGate     posting.TrialBalanceGate  `json:"trial_balance_gate"`
	UnmatchedBankLines   int                       `json:"unmatched_bank_lines"`
	UnmatchedBankAmount  float64                   `json:"unmatched_bank_amount"`
	OpenFollowUpTasks    int                       `json:"open_follow_up_tasks"`
	ExportableAuditItems int                       `json:"exportable_audit_items"`
}

type CashExposureInput struct {
	OpenAR                    float64 `json:"open_ar"`
	OverdueAR                 float64 `json:"overdue_ar"`
	DueInWindow               float64 `json:"due_in_window"`
	ConfirmedUninvoicedOrders float64 `json:"confirmed_uninvoiced_orders"`
	WeightedPipeline          float64 `json:"weighted_pipeline"`
}

type CashExposure struct {
	OpenAR                    float64  `json:"open_ar"`
	OverdueAR                 float64  `json:"overdue_ar"`
	DueInWindow               float64  `json:"due_in_window"`
	ConfirmedUninvoicedOrders float64  `json:"confirmed_uninvoiced_orders"`
	WeightedPipeline          float64  `json:"weighted_pipeline"`
	TotalAttention            float64  `json:"total_attention"`
	OverdueRatio              float64  `json:"overdue_ratio"`
	Priority                  Priority `json:"priority"`
	Status                    Status   `json:"status"`
}

type EvidenceSourceInput struct {
	SourceType string  `json:"source_type"`
	Label      string  `json:"label"`
	Required   int     `json:"required"`
	Present    int     `json:"present"`
	Confidence float64 `json:"confidence"`
}

type AllocationEvidenceInput struct {
	AllocationID        string  `json:"allocation_id"`
	BankStatementLineID string  `json:"bank_statement_line_id"`
	SourceType          string  `json:"source_type"`
	SourceID            string  `json:"source_id"`
	Amount              float64 `json:"amount"`
	AllocationType      string  `json:"allocation_type"`
	Confidence          float64 `json:"confidence"`
	AllocationStatus    string  `json:"allocation_status"`
}

type EvidenceSourceStatus struct {
	SourceType string   `json:"source_type"`
	Label      string   `json:"label"`
	Required   int      `json:"required"`
	Present    int      `json:"present"`
	Missing    int      `json:"missing"`
	Confidence float64  `json:"confidence"`
	Status     Status   `json:"status"`
	Priority   Priority `json:"priority"`
}

type AllocationEvidence struct {
	AllocationID        string   `json:"allocation_id"`
	BankStatementLineID string   `json:"bank_statement_line_id"`
	SourceType          string   `json:"source_type"`
	SourceID            string   `json:"source_id"`
	Amount              float64  `json:"amount"`
	AllocationType      string   `json:"allocation_type"`
	Confidence          float64  `json:"confidence"`
	AllocationStatus    string   `json:"allocation_status"`
	Status              Status   `json:"status"`
	Priority            Priority `json:"priority"`
}

type AllocationSummary struct {
	TotalAllocations int     `json:"total_allocations"`
	Matched          int     `json:"matched"`
	Partial          int     `json:"partial"`
	Mixed            int     `json:"mixed"`
	Conflicts        int     `json:"conflicts"`
	Unresolved       int     `json:"unresolved"`
	TotalAmount      float64 `json:"total_amount"`
}

type PostingReadiness struct {
	Status                 Status   `json:"status"`
	Priority               Priority `json:"priority"`
	TotalSources           int64    `json:"total_sources"`
	MissingJournals        int64    `json:"missing_journals"`
	DraftEntries           int64    `json:"draft_entries"`
	TrialBalanceReady      bool     `json:"trial_balance_ready"`
	BalancedAccountCount   int      `json:"balanced_account_count"`
	ImbalancedAccountCount int      `json:"imbalanced_account_count"`
	Message                string   `json:"message"`
}

type CommandCenter struct {
	Window               TimeWindow             `json:"window"`
	Cash                 CashExposure           `json:"cash"`
	EvidenceSources      []EvidenceSourceStatus `json:"evidence_sources"`
	BankAllocations      []AllocationEvidence   `json:"bank_allocations"`
	AllocationSummary    AllocationSummary      `json:"allocation_summary"`
	Posting              PostingReadiness       `json:"posting"`
	UnmatchedBankLines   int                    `json:"unmatched_bank_lines"`
	UnmatchedBankAmount  float64                `json:"unmatched_bank_amount"`
	OpenFollowUpTasks    int                    `json:"open_follow_up_tasks"`
	ExportableAuditItems int                    `json:"exportable_audit_items"`
	OverallStatus        Status                 `json:"overall_status"`
	NextAction           string                 `json:"next_action"`
	ActionProposals      []ActionProposal       `json:"action_proposals"`
}

func BuildCommandCenter(input CommandCenterInput) CommandCenter {
	out := CommandCenter{
		Window:               input.Window,
		Cash:                 AssessCashExposure(input.Cash),
		EvidenceSources:      AssessEvidenceSources(input.EvidenceSources),
		BankAllocations:      AssessAllocationEvidence(input.BankAllocations),
		Posting:              AssessPostingReadiness(input.PostingCoverage, input.TrialBalanceGate),
		UnmatchedBankLines:   input.UnmatchedBankLines,
		UnmatchedBankAmount:  round(input.UnmatchedBankAmount),
		OpenFollowUpTasks:    input.OpenFollowUpTasks,
		ExportableAuditItems: input.ExportableAuditItems,
	}
	out.AllocationSummary = SummarizeAllocations(out.BankAllocations)
	out.OverallStatus = overallStatus(out)
	out.NextAction = nextAction(out)
	out.ActionProposals = BuildActionProposals(out)
	return out
}

func AssessCashExposure(input CashExposureInput) CashExposure {
	out := CashExposure{
		OpenAR:                    round(input.OpenAR),
		OverdueAR:                 round(input.OverdueAR),
		DueInWindow:               round(input.DueInWindow),
		ConfirmedUninvoicedOrders: round(input.ConfirmedUninvoicedOrders),
		WeightedPipeline:          round(input.WeightedPipeline),
	}
	out.TotalAttention = round(out.OverdueAR + out.DueInWindow + out.ConfirmedUninvoicedOrders + out.WeightedPipeline)
	if out.OpenAR > 0 {
		out.OverdueRatio = round(out.OverdueAR / out.OpenAR)
	}
	out.Priority = priorityForCash(out)
	out.Status = statusForPriority(out.Priority)
	if out.OpenAR == 0 && out.TotalAttention == 0 {
		out.Status = StatusEmpty
	}
	return out
}

func AssessEvidenceSources(inputs []EvidenceSourceInput) []EvidenceSourceStatus {
	statuses := make([]EvidenceSourceStatus, 0, len(inputs))
	for _, input := range inputs {
		required := maxInt(input.Required, 0)
		present := maxInt(input.Present, 0)
		missing := required - present
		if missing < 0 {
			missing = 0
		}
		confidence := clamp(input.Confidence, 0, 1)
		priority := priorityForEvidence(required, missing, confidence)
		statuses = append(statuses, EvidenceSourceStatus{
			SourceType: strings.TrimSpace(input.SourceType),
			Label:      text.FirstNonEmpty(input.Label, input.SourceType, "Evidence"),
			Required:   required,
			Present:    present,
			Missing:    missing,
			Confidence: confidence,
			Status:     statusForPriority(priority),
			Priority:   priority,
		})
	}
	return statuses
}

func AssessAllocationEvidence(inputs []AllocationEvidenceInput) []AllocationEvidence {
	allocations := make([]AllocationEvidence, 0, len(inputs))
	for _, input := range inputs {
		confidence := clamp(input.Confidence, 0, 1)
		allocationStatus := normalizeAllocationStatus(input.AllocationStatus)
		allocationType := normalizeAllocationType(input.AllocationType)
		status, priority := statusForAllocation(input, confidence, allocationStatus)
		allocations = append(allocations, AllocationEvidence{
			AllocationID:        text.FirstNonEmpty(input.AllocationID, allocationIdentity(input), "untracked_allocation"),
			BankStatementLineID: strings.TrimSpace(input.BankStatementLineID),
			SourceType:          strings.TrimSpace(input.SourceType),
			SourceID:            strings.TrimSpace(input.SourceID),
			Amount:              round(input.Amount),
			AllocationType:      allocationType,
			Confidence:          confidence,
			AllocationStatus:    allocationStatus,
			Status:              status,
			Priority:            priority,
		})
	}
	return allocations
}

func SummarizeAllocations(allocations []AllocationEvidence) AllocationSummary {
	summary := AllocationSummary{TotalAllocations: len(allocations)}
	for _, allocation := range allocations {
		summary.TotalAmount = round(summary.TotalAmount + allocation.Amount)
		if allocation.AllocationType == "partial" || allocation.AllocationStatus == "partial" {
			summary.Partial++
		}
		if allocation.AllocationType == "mixed" {
			summary.Mixed++
		}
		switch allocation.AllocationStatus {
		case "matched", "confirmed":
			summary.Matched++
		case "conflict", "rejected":
			summary.Conflicts++
		case "proposed", "unmatched", "needs_review", "partial":
			summary.Unresolved++
		}
	}
	return summary
}

func AssessPostingReadiness(coverage posting.CoverageReport, gate posting.TrialBalanceGate) PostingReadiness {
	out := PostingReadiness{
		TotalSources:           coverage.Total,
		MissingJournals:        coverage.Missing,
		DraftEntries:           coverage.DraftEntries,
		TrialBalanceReady:      gate.IsBalanced,
		BalancedAccountCount:   len(gate.BalancedAccounts),
		ImbalancedAccountCount: len(gate.ImbalancedAccounts),
	}
	switch {
	case !gate.IsBalanced && gate.LineCount > 0 && len(gate.BalancedAccounts) == 0:
		out.Status = StatusBlocked
		out.Priority = PriorityUrgent
		out.Message = "Trial balance is not balanced; no accounts are individually balanced for posting."
	case !gate.IsBalanced && gate.LineCount > 0 && len(gate.BalancedAccounts) > 0:
		out.Status = StatusAttention
		out.Priority = PriorityHigh
		out.Message = fmt.Sprintf("Trial balance is not balanced; %d account(s) are individually balanced and may allow scoped posting.", len(gate.BalancedAccounts))
	case coverage.Total == 0:
		out.Status = StatusEmpty
		out.Priority = PriorityNone
		out.Message = "No eligible posting sources are in the current evidence window."
	case coverage.Missing > 0:
		out.Status = StatusAttention
		out.Priority = PriorityHigh
		out.Message = "Some eligible finance sources are missing journal links."
	case coverage.DraftEntries > 0:
		out.Status = StatusAttention
		out.Priority = PriorityMedium
		out.Message = "Draft journal entries exist and need operator review."
	default:
		out.Status = StatusReady
		out.Priority = PriorityNone
		out.Message = "All eligible sources have journal links and the trial balance is balanced."
	}
	return out
}

func priorityForCash(cash CashExposure) Priority {
	switch {
	case cash.OverdueAR > 0 && cash.OverdueRatio >= 0.5:
		return PriorityUrgent
	case cash.OverdueAR > 0 || cash.DueInWindow > 0:
		return PriorityHigh
	case cash.ConfirmedUninvoicedOrders > 0:
		return PriorityMedium
	case cash.WeightedPipeline > 0:
		return PriorityLow
	default:
		return PriorityNone
	}
}

func priorityForEvidence(required, missing int, confidence float64) Priority {
	switch {
	case required == 0:
		return PriorityNone
	case missing >= required:
		return PriorityUrgent
	case missing > 0:
		return PriorityHigh
	case confidence > 0 && confidence < 0.6:
		return PriorityMedium
	default:
		return PriorityNone
	}
}

func overallStatus(center CommandCenter) Status {
	if center.Posting.Status == StatusBlocked || center.Cash.Priority == PriorityUrgent {
		return StatusBlocked
	}
	for _, allocation := range center.BankAllocations {
		if allocation.Status == StatusBlocked {
			return StatusBlocked
		}
	}
	if center.Posting.Status == StatusAttention || center.Cash.Status == StatusAttention ||
		center.UnmatchedBankLines > 0 || center.OpenFollowUpTasks > 0 || center.AllocationSummary.Unresolved > 0 {
		return StatusAttention
	}
	for _, source := range center.EvidenceSources {
		if source.Status == StatusBlocked {
			return StatusBlocked
		}
		if source.Status == StatusAttention {
			return StatusAttention
		}
	}
	if center.Cash.Status == StatusEmpty && center.Posting.Status == StatusEmpty && len(center.EvidenceSources) == 0 {
		return StatusEmpty
	}
	return StatusReady
}

func nextAction(center CommandCenter) string {
	if center.Posting.Status == StatusBlocked {
		return "Fix trial-balance blockers before approving posting actions."
	}
	for _, source := range center.EvidenceSources {
		if source.Missing > 0 {
			return "Request or link missing evidence before exporting the pack."
		}
	}
	if center.Posting.MissingJournals > 0 {
		return "Review missing journal links and request deterministic draft postings."
	}
	if center.AllocationSummary.Unresolved > 0 || center.AllocationSummary.Conflicts > 0 {
		return "Review partial or mixed bank allocations before resolving cash evidence."
	}
	if center.UnmatchedBankLines > 0 {
		return "Resolve unmatched bank lines to improve cash evidence."
	}
	if center.Cash.Priority == PriorityHigh || center.Cash.Priority == PriorityUrgent {
		return "Draft collection follow-ups for overdue and due receivables."
	}
	if center.ExportableAuditItems > 0 {
		return "Export the evidence pack for operator or accountant review."
	}
	return "Monitor cashflow evidence readiness."
}

func statusForAllocation(input AllocationEvidenceInput, confidence float64, allocationStatus string) (Status, Priority) {
	switch allocationStatus {
	case "conflict", "rejected":
		return StatusBlocked, PriorityUrgent
	case "proposed", "unmatched", "needs_review":
		return StatusAttention, PriorityHigh
	case "partial":
		return StatusAttention, PriorityMedium
	}
	if strings.TrimSpace(input.BankStatementLineID) == "" || strings.TrimSpace(input.SourceID) == "" {
		return StatusAttention, PriorityHigh
	}
	if confidence > 0 && confidence < 0.6 {
		return StatusAttention, PriorityMedium
	}
	return StatusReady, PriorityNone
}

func normalizeAllocationType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "customer_invoice", "supplier_invoice", "expense", "partial", "mixed":
		return value
	case "":
		return "unknown"
	default:
		return value
	}
}

func normalizeAllocationStatus(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "matched", "confirmed", "partial", "proposed", "unmatched", "needs_review", "conflict", "rejected":
		return value
	case "":
		return "proposed"
	default:
		return value
	}
}

func allocationIdentity(input AllocationEvidenceInput) string {
	parts := []string{
		strings.TrimSpace(input.BankStatementLineID),
		strings.TrimSpace(input.SourceType),
		strings.TrimSpace(input.SourceID),
	}
	return strings.Trim(strings.Join(parts, ":"), ":")
}

func statusForPriority(priority Priority) Status {
	switch priority {
	case PriorityUrgent:
		return StatusBlocked
	case PriorityHigh, PriorityMedium, PriorityLow:
		return StatusAttention
	default:
		return StatusReady
	}
}

func round(v float64) float64 {
	return money.RoundFloat64(v, 3)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return round(v)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
