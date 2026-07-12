package cashflow

import (
	"testing"

	"ph_holdings_app/pkg/cashflow/evidence"
)

func TestBuildCommandCenterVMExposesTrustActions(t *testing.T) {
	vm := BuildCommandCenterVM(evidence.CommandCenter{
		Window: evidence.TimeWindow{Label: "Next 30 days"},
		Cash: evidence.CashExposure{
			OpenAR:    1000,
			OverdueAR: 250,
			Priority:  evidence.PriorityHigh,
			Status:    evidence.StatusAttention,
		},
		EvidenceSources: []evidence.EvidenceSourceStatus{{
			SourceType: "invoice_pdf",
			Label:      "Invoice PDFs",
			Required:   5,
			Present:    4,
			Missing:    1,
			Confidence: 0.8,
			Status:     evidence.StatusAttention,
		}},
		BankAllocations: []evidence.AllocationEvidence{{
			AllocationID:        "alloc-1",
			BankStatementLineID: "bank-line-1",
			SourceType:          "customer_invoice",
			SourceID:            "inv-100",
			Amount:              125.5,
			AllocationType:      "customer_invoice",
			AllocationStatus:    "matched",
			Confidence:          0.9,
			Status:              evidence.StatusReady,
		}},
		AllocationSummary: evidence.AllocationSummary{
			TotalAllocations: 1,
			Matched:          1,
			TotalAmount:      125.5,
		},
		Posting: evidence.PostingReadiness{
			Status:            evidence.StatusReady,
			Message:           "Ready",
			TotalSources:      5,
			TrialBalanceReady: true,
		},
		UnmatchedBankLines:   2,
		UnmatchedBankAmount:  33.5,
		ExportableAuditItems: 7,
		OverallStatus:        evidence.StatusAttention,
		NextAction:           "Request or link missing evidence before exporting the pack.",
	})

	if vm.WindowLabel != "Next 30 days" {
		t.Fatalf("window label = %q", vm.WindowLabel)
	}
	if vm.OverallStatus.Label != "Attention" {
		t.Fatalf("status = %+v, want Attention", vm.OverallStatus)
	}
	if len(vm.EvidenceSources) != 1 || vm.EvidenceSources[0].CompletenessLabel != "4/5 present" {
		t.Fatalf("evidence sources = %+v", vm.EvidenceSources)
	}
	if len(vm.BankAllocations) != 1 || vm.BankAllocations[0].AllocationID != "alloc-1" || vm.BankAllocations[0].AmountLabel != "BHD 125.500" {
		t.Fatalf("bank allocations = %+v", vm.BankAllocations)
	}
	if !vm.Actions[1].Enabled {
		t.Fatal("draft follow-up action should be enabled for high-priority cash")
	}
	if !vm.Actions[2].Enabled {
		t.Fatal("export action should be enabled when audit items exist")
	}
}
