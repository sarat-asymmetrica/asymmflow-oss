// Package cashflow contains display-ready contracts for the Cashflow Evidence module.
package cashflow

import (
	"fmt"
	"math"

	vm "ph_holdings_app/internal/viewmodel"
	"ph_holdings_app/internal/viewmodel/shared"
	"ph_holdings_app/pkg/cashflow/evidence"
	"ph_holdings_app/pkg/kernel/text"
)

type CommandCenterVM struct {
	WindowLabel     string                 `json:"windowLabel"`
	OverallStatus   shared.StatusBadgeVM   `json:"overallStatus"`
	SummaryCards    []vm.SummaryCard       `json:"summaryCards"`
	EvidenceSources []EvidenceSourceVM     `json:"evidenceSources"`
	BankAllocations []AllocationEvidenceVM `json:"bankAllocations,omitempty"`
	Posting         PostingReadinessVM     `json:"posting"`
	NextAction      string                 `json:"nextAction"`
	Actions         []vm.ActionButton      `json:"actions"`
}

type EvidenceSourceVM struct {
	SourceType        string               `json:"sourceType"`
	Label             string               `json:"label"`
	CompletenessLabel string               `json:"completenessLabel"`
	ConfidenceLabel   string               `json:"confidenceLabel"`
	Status            shared.StatusBadgeVM `json:"status"`
}

type AllocationEvidenceVM struct {
	AllocationID        string               `json:"allocationId"`
	BankStatementLineID string               `json:"bankStatementLineId"`
	SourceLabel         string               `json:"sourceLabel"`
	AmountLabel         string               `json:"amountLabel"`
	AllocationType      string               `json:"allocationType"`
	AllocationStatus    string               `json:"allocationStatus"`
	ConfidenceLabel     string               `json:"confidenceLabel"`
	Status              shared.StatusBadgeVM `json:"status"`
}

type PostingReadinessVM struct {
	Status            shared.StatusBadgeVM `json:"status"`
	Message           string               `json:"message"`
	TotalSources      int64                `json:"totalSources"`
	MissingJournals   int64                `json:"missingJournals"`
	DraftEntries      int64                `json:"draftEntries"`
	TrialBalanceReady bool                 `json:"trialBalanceReady"`
}

func BuildCommandCenterVM(center evidence.CommandCenter) CommandCenterVM {
	sources := make([]EvidenceSourceVM, 0, len(center.EvidenceSources))
	for _, source := range center.EvidenceSources {
		sources = append(sources, EvidenceSourceVM{
			SourceType:        source.SourceType,
			Label:             source.Label,
			CompletenessLabel: fmt.Sprintf("%d/%d present", source.Present, source.Required),
			ConfidenceLabel:   formatPercent(source.Confidence),
			Status:            badgeForStatus(source.Status),
		})
	}
	allocations := make([]AllocationEvidenceVM, 0, len(center.BankAllocations))
	for _, allocation := range center.BankAllocations {
		allocations = append(allocations, AllocationEvidenceVM{
			AllocationID:        allocation.AllocationID,
			BankStatementLineID: allocation.BankStatementLineID,
			SourceLabel:         text.FirstNonEmpty(allocation.SourceType+" "+allocation.SourceID, allocation.SourceID, allocation.SourceType, "Unlinked source"),
			AmountLabel:         formatMoney(allocation.Amount),
			AllocationType:      text.FirstNonEmpty(allocation.AllocationType, "unknown"),
			AllocationStatus:    text.FirstNonEmpty(allocation.AllocationStatus, "proposed"),
			ConfidenceLabel:     formatPercent(allocation.Confidence),
			Status:              badgeForStatus(allocation.Status),
		})
	}

	return CommandCenterVM{
		WindowLabel:   text.FirstNonEmpty(center.Window.Label, "Current cashflow window"),
		OverallStatus: badgeForStatus(center.OverallStatus),
		SummaryCards: []vm.SummaryCard{
			{Label: "Open AR", Value: formatMoney(center.Cash.OpenAR), Subtext: "customer receivables", Color: "forest"},
			{Label: "Overdue AR", Value: formatMoney(center.Cash.OverdueAR), Subtext: string(center.Cash.Priority), Color: colorForStatus(center.Cash.Status)},
			{Label: "Unmatched Bank", Value: formatMoney(center.UnmatchedBankAmount), Subtext: fmt.Sprintf("%d lines", center.UnmatchedBankLines), Color: "amber"},
			{Label: "Audit Items", Value: fmt.Sprintf("%d", center.ExportableAuditItems), Subtext: "export-ready", Color: "gray"},
		},
		EvidenceSources: sources,
		BankAllocations: allocations,
		Posting: PostingReadinessVM{
			Status:            badgeForStatus(center.Posting.Status),
			Message:           center.Posting.Message,
			TotalSources:      center.Posting.TotalSources,
			MissingJournals:   center.Posting.MissingJournals,
			DraftEntries:      center.Posting.DraftEntries,
			TrialBalanceReady: center.Posting.TrialBalanceReady,
		},
		NextAction: center.NextAction,
		Actions: []vm.ActionButton{
			{Label: "Inspect Sources", Action: "cashflowEvidence.inspectSources", Icon: "search", Variant: "secondary", Enabled: true},
			{Label: "Draft Follow-Up", Action: "cashflowEvidence.draftFollowUp", Icon: "send", Variant: "primary", Enabled: center.Cash.Priority != evidence.PriorityNone},
			{Label: "Export Evidence", Action: "cashflowEvidence.exportPack", Icon: "download", Variant: "secondary", Enabled: center.ExportableAuditItems > 0},
		},
	}
}

func badgeForStatus(status evidence.Status) shared.StatusBadgeVM {
	switch status {
	case evidence.StatusReady:
		return shared.StatusBadgeVM{Label: "Ready", Color: "green", Icon: "check-circle"}
	case evidence.StatusAttention:
		return shared.StatusBadgeVM{Label: "Attention", Color: "amber", Icon: "alert-circle"}
	case evidence.StatusBlocked:
		return shared.StatusBadgeVM{Label: "Blocked", Color: "red", Icon: "alert-triangle"}
	default:
		return shared.StatusBadgeVM{Label: "Empty", Color: "gray", Icon: "circle"}
	}
}

func colorForStatus(status evidence.Status) string {
	switch status {
	case evidence.StatusBlocked:
		return "red"
	case evidence.StatusAttention:
		return "amber"
	case evidence.StatusReady:
		return "forest"
	default:
		return "gray"
	}
}

func formatMoney(amount float64) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = math.Abs(amount)
	}
	return fmt.Sprintf("%sBHD %.3f", sign, amount)
}

func formatPercent(value float64) string {
	if value <= 0 {
		return "not scored"
	}
	return fmt.Sprintf("%.0f%%", math.Round(value*100))
}
