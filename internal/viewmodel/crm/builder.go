package crm

import (
	"fmt"
	"sort"
	"strings"
	"time"

	vm "ph_holdings_app/internal/viewmodel"
	financevm "ph_holdings_app/internal/viewmodel/finance"
	"ph_holdings_app/internal/viewmodel/shared"
	domain "ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/kernel/text"
)

var pipelineOrder = []string{"Qualified", "Proposal", "Quoted", "Negotiation", "Won", "Lost"}

// BuildCustomerListVM constructs a customer list ViewModel.
func BuildCustomerListVM(customers []domain.CustomerMaster, page, pageSize int) CustomerListVM {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = len(customers)
	}
	grades := map[string]int{}
	rows := make([]shared.TableRow, 0, len(customers))
	for _, customer := range customers {
		grade := text.FirstNonEmpty(customer.CustomerGrade, customer.PaymentGrade, "C")
		grades[grade]++
		status := CustomerStatusBadge(customer.Status, customer.IsCreditBlocked)
		rows = append(rows, shared.TableRow{
			ID: customer.ID,
			Fields: map[string]any{
				"businessName": customer.BusinessName,
				"customerCode": text.FirstNonEmpty(customer.CustomerCode, customer.CustomerID),
				"grade":        GradeBadge(grade),
				"phone":        text.FirstNonEmpty(customer.MobileNumber, customer.PrimaryPhone),
				"email":        customer.PrimaryEmail,
				"outstanding":  financevm.FormatMoney(customer.OutstandingBHD, "BHD"),
				"status":       status,
			},
			Status: status.Label,
			Actions: []vm.ActionButton{
				{Label: "Open 360", Action: "customer.open360", Icon: "contact", Variant: "primary", Enabled: true},
				{Label: "Create Follow-up", Action: "customer.createFollowUp", Icon: "calendar-plus", Variant: "secondary", Enabled: true},
			},
		})
	}

	return CustomerListVM{
		Table: shared.TableVM{
			Columns: []shared.TableColumn{
				{Key: "businessName", Label: "Customer", Type: "text", Sortable: true},
				{Key: "customerCode", Label: "Code", Type: "text", Sortable: true, Width: "110px"},
				{Key: "grade", Label: "Grade", Type: "status", Sortable: true, Width: "90px"},
				{Key: "phone", Label: "Phone", Type: "text", Sortable: false, Width: "130px"},
				{Key: "email", Label: "Email", Type: "text", Sortable: false},
				{Key: "outstanding", Label: "Outstanding", Type: "currency", Sortable: true, Align: "right", Currency: "BHD"},
				{Key: "status", Label: "Status", Type: "status", Sortable: true, Width: "110px"},
			},
			Rows:       rows,
			TotalRows:  len(customers),
			Page:       page,
			PageSize:   pageSize,
			SortColumn: "businessName",
			Filters: []shared.TableFilter{
				{Column: "businessName", Type: "text"},
				{Column: "grade", Type: "select", Options: gradeOptions()},
				{Column: "status", Type: "select", Options: statusOptions()},
			},
		},
		TotalCustomers:    len(customers),
		GradeDistribution: gradeDistribution(grades),
		Actions: []vm.ActionButton{
			{Label: "New Customer", Action: "customer.create", Icon: "plus", Variant: "primary", Enabled: true},
			{Label: "Export", Action: "customer.export", Icon: "download", Variant: "secondary", Enabled: len(customers) > 0},
		},
	}
}

// BuildPipelineVM constructs the opportunity pipeline ViewModel.
func BuildPipelineVM(opportunities []domain.Opportunity) PipelineVM {
	stageCounts := map[string]int{}
	stageValues := map[string]float64{}
	stageItems := map[string][]OpportunityCardVM{}
	closed := 0
	won := 0
	openCount := 0
	totalOpenValue := 0.0
	for _, opp := range opportunities {
		stage := text.FirstNonEmpty(opp.Stage, "Qualified")
		stageCounts[stage]++
		stageValues[stage] += opp.RevenueBHD
		stageItems[stage] = append(stageItems[stage], OpportunityCardVM{
			ID:           opp.ID,
			FolderNumber: opp.FolderNumber,
			Title:        text.FirstNonEmpty(opp.Title, opp.FolderName),
			CustomerName: opp.CustomerName,
			ValueDisplay: financevm.FormatMoney(opp.RevenueBHD, "BHD"),
			ExpectedDate: financevm.FormatDate(timePtrValue(opp.ExpectedDate)),
			Status:       PipelineStatusBadge(stage),
		})
		if stage == "Won" || stage == "Lost" {
			closed++
			if stage == "Won" {
				won++
			}
		} else {
			openCount++
			totalOpenValue += opp.RevenueBHD
		}
	}

	stages := make([]PipelineStageVM, 0, len(stageCounts))
	seen := map[string]bool{}
	for _, stage := range pipelineOrder {
		if count := stageCounts[stage]; count > 0 {
			stages = append(stages, buildPipelineStage(stage, count, stageValues[stage], stageItems[stage]))
			seen[stage] = true
		}
	}
	for stage, count := range stageCounts {
		if !seen[stage] {
			stages = append(stages, buildPipelineStage(stage, count, stageValues[stage], stageItems[stage]))
		}
	}

	winRate := "0.0%"
	if closed > 0 {
		winRate = fmt.Sprintf("%.1f%%", (float64(won)/float64(closed))*100)
	}
	return PipelineVM{
		Stages:               stages,
		TotalPipelineValue:   financevm.FormatMoney(totalOpenValue, "BHD"),
		WinRate:              winRate,
		OpenOpportunityCount: openCount,
		Actions: []vm.ActionButton{
			{Label: "New Opportunity", Action: "opportunity.create", Icon: "plus", Variant: "primary", Enabled: true},
			{Label: "Import Pipeline", Action: "opportunity.import", Icon: "upload", Variant: "secondary", Enabled: true},
		},
	}
}

// BuildPipelineSnapshotVM builds a compact dashboard pipeline snapshot.
func BuildPipelineSnapshotVM(opportunities []domain.Opportunity) PipelineSnapshotVM {
	pipeline := BuildPipelineVM(opportunities)
	topStage := ""
	topCount := -1
	for _, stage := range pipeline.Stages {
		if stage.Stage != "Won" && stage.Stage != "Lost" && stage.Count > topCount {
			topStage = stage.Stage
			topCount = stage.Count
		}
	}
	return PipelineSnapshotVM{
		OpenCount:     pipeline.OpenOpportunityCount,
		WeightedValue: pipeline.TotalPipelineValue,
		TopStage:      topStage,
		WinRate:       pipeline.WinRate,
	}
}

// BuildOrderListVM constructs an order list ViewModel.
func BuildOrderListVM(orders []domain.Order, page, pageSize int) OrderListVM {
	rows := make([]shared.TableRow, 0, len(orders))
	for _, order := range orders {
		status := OrderStatusBadge(order.Status)
		total := order.GrandTotalBHD
		if total == 0 {
			total = order.TotalValueBHD
		}
		rows = append(rows, shared.TableRow{
			ID: order.ID,
			Fields: map[string]any{
				"orderNumber":  order.OrderNumber,
				"customerName": order.CustomerName,
				"orderDate":    financevm.FormatDate(order.OrderDate),
				"requiredDate": financevm.FormatDate(order.RequiredDate),
				"status":       status,
				"total":        financevm.FormatMoney(total, "BHD"),
			},
			Status: status.Label,
			Actions: []vm.ActionButton{
				{Label: "Open", Action: "order.open", Icon: "folder-open", Variant: "primary", Enabled: true},
				{Label: "Create Invoice", Action: "order.createInvoice", Icon: "file-plus", Variant: "secondary", Enabled: order.Status != "Cancelled"},
			},
		})
	}
	return OrderListVM{
		Table: shared.TableVM{
			Columns: []shared.TableColumn{
				{Key: "orderNumber", Label: "Order", Type: "text", Sortable: true, Width: "130px"},
				{Key: "customerName", Label: "Customer", Type: "text", Sortable: true},
				{Key: "orderDate", Label: "Order Date", Type: "date", Sortable: true, Width: "130px"},
				{Key: "requiredDate", Label: "Required", Type: "date", Sortable: true, Width: "130px"},
				{Key: "status", Label: "Status", Type: "status", Sortable: true, Width: "130px"},
				{Key: "total", Label: "Total", Type: "currency", Sortable: true, Align: "right", Currency: "BHD"},
			},
			Rows:       rows,
			TotalRows:  len(orders),
			Page:       page,
			PageSize:   pageSize,
			SortColumn: "orderDate",
			SortDesc:   true,
		},
		Actions: []vm.ActionButton{
			{Label: "New Order", Action: "order.create", Icon: "plus", Variant: "primary", Enabled: true},
		},
	}
}

// CustomerStatusBadge maps customer state to a display badge.
func CustomerStatusBadge(status string, creditBlocked bool) shared.StatusBadgeVM {
	if creditBlocked {
		return shared.StatusBadgeVM{Label: "Credit Blocked", Color: "red", Icon: "shield-alert"}
	}
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active", "":
		return shared.StatusBadgeVM{Label: "Active", Color: "green", Icon: "check-circle"}
	case "inactive":
		return shared.StatusBadgeVM{Label: "Inactive", Color: "gray", Icon: "pause-circle"}
	default:
		return shared.StatusBadgeVM{Label: status, Color: "gray"}
	}
}

// GradeBadge maps customer grades to display badges.
func GradeBadge(grade string) shared.StatusBadgeVM {
	grade = strings.ToUpper(text.FirstNonEmpty(grade, "C"))
	switch grade {
	case "A":
		return shared.StatusBadgeVM{Label: "A", Color: "green", Icon: "star"}
	case "B":
		return shared.StatusBadgeVM{Label: "B", Color: "blue", Icon: "star"}
	case "D":
		return shared.StatusBadgeVM{Label: "D", Color: "red", Icon: "alert-triangle"}
	default:
		return shared.StatusBadgeVM{Label: grade, Color: "amber", Icon: "circle"}
	}
}

// PipelineStatusBadge maps opportunity stages to display badges.
func PipelineStatusBadge(stage string) shared.StatusBadgeVM {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "won":
		return shared.StatusBadgeVM{Label: "Won", Color: "green", Icon: "trophy"}
	case "lost":
		return shared.StatusBadgeVM{Label: "Lost", Color: "red", Icon: "x-circle"}
	case "quoted", "proposal":
		return shared.StatusBadgeVM{Label: stage, Color: "blue", Icon: "file-text"}
	default:
		return shared.StatusBadgeVM{Label: text.FirstNonEmpty(stage, "Qualified"), Color: "amber", Icon: "activity"}
	}
}

// OrderStatusBadge maps order state to a display badge.
func OrderStatusBadge(status string) shared.StatusBadgeVM {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "delivered", "completed", "closed":
		return shared.StatusBadgeVM{Label: status, Color: "green", Icon: "check-circle"}
	case "cancelled", "canceled":
		return shared.StatusBadgeVM{Label: "Cancelled", Color: "red", Icon: "ban"}
	case "invoiced", "processing":
		return shared.StatusBadgeVM{Label: text.FirstNonEmpty(status, "Processing"), Color: "blue", Icon: "package"}
	default:
		return shared.StatusBadgeVM{Label: text.FirstNonEmpty(status, "Open"), Color: "amber", Icon: "clock"}
	}
}

func buildPipelineStage(stage string, count int, value float64, items []OpportunityCardVM) PipelineStageVM {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ValueDisplay > items[j].ValueDisplay
	})
	return PipelineStageVM{
		Stage:        stage,
		Count:        count,
		ValueDisplay: financevm.FormatMoney(value, "BHD"),
		Color:        PipelineStatusBadge(stage).Color,
		Items:        items,
	}
}

func gradeDistribution(grades map[string]int) []GradeBucketVM {
	order := []string{"A", "B", "C", "D"}
	out := make([]GradeBucketVM, 0, len(order))
	for _, grade := range order {
		if count := grades[grade]; count > 0 {
			out = append(out, GradeBucketVM{Grade: grade, Count: count, Color: GradeBadge(grade).Color})
		}
	}
	return out
}

func gradeOptions() []vm.Option {
	return []vm.Option{{Value: "A", Label: "A"}, {Value: "B", Label: "B"}, {Value: "C", Label: "C"}, {Value: "D", Label: "D"}}
}

func statusOptions() []vm.Option {
	return []vm.Option{{Value: "Active", Label: "Active"}, {Value: "Inactive", Label: "Inactive"}, {Value: "CreditBlocked", Label: "Credit Blocked"}}
}

func timePtrValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
