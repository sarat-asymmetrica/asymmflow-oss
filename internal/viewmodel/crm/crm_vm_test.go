package crm

import (
	"testing"
	"time"

	domain "ph_holdings_app/pkg/crm"
)

func TestBuildCustomerListVMFormatsRowsAndGrades(t *testing.T) {
	customers := []domain.CustomerMaster{
		{Base: domain.Base{ID: "cust-1"}, BusinessName: "Gulf Smelting", CustomerCode: "ALB", CustomerGrade: "A", Status: "Active", OutstandingBHD: 1200.5},
		{Base: domain.Base{ID: "cust-2"}, BusinessName: "NGA", CustomerCode: "NGA", CustomerGrade: "C", Status: "Active", IsCreditBlocked: true, OutstandingBHD: 50},
	}

	got := BuildCustomerListVM(customers, 1, 20)
	if got.TotalCustomers != 2 || len(got.GradeDistribution) != 2 {
		t.Fatalf("unexpected customer summary: %#v", got)
	}
	if got.Table.Rows[0].Fields["outstanding"] != "BHD 1,200.50" {
		t.Fatalf("unexpected money formatting: %#v", got.Table.Rows[0].Fields["outstanding"])
	}
	if got.Table.Rows[1].Status != "Credit Blocked" {
		t.Fatalf("expected credit blocked status, got %q", got.Table.Rows[1].Status)
	}
}

func TestBuildPipelineVMComputesValueAndWinRate(t *testing.T) {
	expected := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	opps := []domain.Opportunity{
		{Base: domain.Base{ID: "opp-1"}, FolderNumber: "26-001", Title: "Flow meters", CustomerName: "Gulf Smelting", Stage: "Qualified", RevenueBHD: 1000, ExpectedDate: &expected},
		{Base: domain.Base{ID: "opp-2"}, FolderNumber: "26-002", Title: "Analyzer", CustomerName: "NGA", Stage: "Won", RevenueBHD: 500},
		{Base: domain.Base{ID: "opp-3"}, FolderNumber: "26-003", Title: "Valve", CustomerName: "NPC", Stage: "Lost", RevenueBHD: 250},
	}

	got := BuildPipelineVM(opps)
	if got.TotalPipelineValue != "BHD 1,000.00" || got.WinRate != "50.0%" {
		t.Fatalf("unexpected pipeline summary: %#v", got)
	}
	if got.OpenOpportunityCount != 1 {
		t.Fatalf("expected one open opportunity, got %d", got.OpenOpportunityCount)
	}
}
