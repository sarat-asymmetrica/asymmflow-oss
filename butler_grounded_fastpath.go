package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	butlerfastpath "ph_holdings_app/pkg/butler/fastpath"
)

func (a *App) tryGroundedCapabilitiesFastPath(intent Intent, message string, hasFinanceAccess bool) (string, bool) {
	if a != nil {
		if msg, handled := a.butlerFastpathService().TryCapabilities(message, hasFinanceAccess); handled {
			return msg, true
		}
	}

	if a == nil || a.db == nil {
		return "", false
	}

	q := strings.ToLower(strings.TrimSpace(message))
	if q == "" {
		return "", false
	}

	capabilitySignals := []string{
		"what can you do", "what all can you do", "all that you can do", "tell me all that you can do",
		"what you can do", "tell us what you can do", "tell me what you can do",
		"what are your capabilities", "what can butler do", "how can you help", "what can u do",
		"show me all butler capabilities", "show all butler capabilities", "all butler capabilities",
		"capabilities from live erp data", "capabilities for sales pipeline", "capabilities for finance",
		"ocr ingestion", "task follow-up and daily briefing capabilities", "task follow up and daily briefing capabilities",
		"report and document generation capabilities",
	}
	matched := false
	for _, signal := range capabilitySignals {
		if strings.Contains(q, signal) {
			matched = true
			break
		}
	}
	if !matched {
		return "", false
	}

	var customerCount, supplierCount, orderCount, invoiceCount, offerCount, opportunityCount, taskCount, productCount int64
	_ = a.db.Model(&CustomerMaster{}).Count(&customerCount).Error
	_ = a.db.Model(&SupplierMaster{}).Count(&supplierCount).Error
	_ = a.db.Model(&Order{}).Where("status NOT IN ?", []string{"Cancelled", "Canceled", "Void"}).Count(&orderCount).Error
	_ = a.db.Model(&Invoice{}).Where("status NOT IN ?", []string{"Cancelled", "Void", "Proforma"}).Count(&invoiceCount).Error
	_ = a.db.Model(&Offer{}).Where("stage NOT IN ?", []string{"Lost", "Expired"}).Count(&offerCount).Error
	_ = a.db.Model(&Opportunity{}).Count(&opportunityCount).Error
	_ = a.db.Model(&TaskItem{}).Where("status NOT IN ?", []string{"completed", "archived"}).Count(&taskCount).Error
	_ = a.db.Model(&ProductMaster{}).Where("is_active = ?", true).Count(&productCount).Error

	currentYear := time.Now().Year()
	yearStart := time.Date(currentYear, time.January, 1, 0, 0, 0, 0, time.Local)
	yearEnd := yearStart.AddDate(1, 0, 0)
	var currentYearOrders, currentYearOffers float64
	_ = a.db.Model(&Order{}).
		Where("order_date >= ? AND order_date < ? AND status NOT IN ?", yearStart, yearEnd, []string{"Cancelled", "Canceled", "Void"}).
		Select("COALESCE(SUM(CASE WHEN grand_total_bhd > 0 THEN grand_total_bhd ELSE total_value_bhd END), 0)").
		Scan(&currentYearOrders).Error
	_ = a.db.Model(&Offer{}).
		Where("quotation_date >= ? AND quotation_date < ? AND stage IN ?", yearStart, yearEnd, []string{"RFQ", "Quoted", "Won"}).
		Select("COALESCE(SUM(total_value_bhd), 0)").
		Scan(&currentYearOffers).Error

	financeLine := "Financial data is permission-gated. I can discuss operational context unless your role has finance:view."
	if hasFinanceAccess {
		financeLine = fmt.Sprintf("For FY%d I can see confirmed orders of %s BHD and offer coverage of %s BHD.", currentYear, formatBHD(currentYearOrders), formatBHD(currentYearOffers))
	}

	databaseInventory := a.getDatabaseInventoryContext(hasFinanceAccess)
	tableCount, _ := databaseInventory["total_tables"].(int)
	if tableCount == 0 {
		if tables, ok := databaseInventory["tables"].([]map[string]any); ok {
			tableCount = len(tables)
		}
	}
	modulesLine := "relationships, commercial pipeline, operations, work management, products, intelligence, and system records"
	if hasFinanceAccess {
		modulesLine = "relationships, commercial pipeline, operations, finance, banking, payroll, work management, products, intelligence, and system records"
	}

	response := fmt.Sprintf(`I can help as a grounded ERP intelligence layer, not a generic chatbot.

Live data I can use right now
- Customers: %d
- Suppliers: %d
- Orders: %d
- Customer invoices: %d
- Active or won offers: %d
- Pipeline opportunities: %d
- Active work tasks: %d
- Active products: %d
- Database tables indexed for guarded querying: %d

What I can do
- Answer customer, supplier, order, offer, invoice, delivery, and task questions from local ERP data.
- Build manager briefings for pipeline, AR, cash position, and operating risk.
- Create follow-up tasks, offer drafts, customers, suppliers, and contacts when the required fields are present.
- Prepare PDF intelligence reports from local data.
- Keep continuity across this conversation and use recent context when you refer to the same customer, offer, or task.
- Query across %s using controlled ERP access paths instead of guessing from memory.

Finance access
- %s

How I will behave
- I will separate actual records from projections.
- I will say when a number is unavailable instead of inventing it.
- If the external AI service is slow, I will still answer from local ERP data where possible.`, customerCount, supplierCount, orderCount, invoiceCount, offerCount, opportunityCount, taskCount, productCount, tableCount, modulesLine, financeLine)

	return response, true
}

func isManagerFinancialBriefRequest(message string) bool {
	q := strings.ToLower(strings.TrimSpace(message))
	if q == "" {
		return false
	}

	hasDocumentSignal := strings.Contains(q, "document") ||
		strings.Contains(q, "report") ||
		strings.Contains(q, "brief") ||
		strings.Contains(q, "briefing") ||
		strings.Contains(q, "manager")
	hasFinanceSignal := strings.Contains(q, "financial") ||
		strings.Contains(q, "finance") ||
		strings.Contains(q, "ar") ||
		strings.Contains(q, "receivable") ||
		strings.Contains(q, "cash") ||
		strings.Contains(q, "standing")
	hasCommercialSignal := strings.Contains(q, "pipeline") ||
		strings.Contains(q, "offer") ||
		strings.Contains(q, "order") ||
		strings.Contains(q, "forecast") ||
		strings.Contains(q, "prediction")

	return hasDocumentSignal && hasFinanceSignal && hasCommercialSignal
}

func requestedBriefYear(message string) int {
	_, _, year, _, ok := parseYearWindowFromQuery(message)
	if ok {
		return year
	}
	return time.Now().Year()
}

func (a *App) tryGroundedManagerFinancialBriefFastPath(intent Intent, message string, hasFinanceAccess bool) (string, []ButlerAction, bool) {
	if a == nil || a.db == nil || !isManagerFinancialBriefRequest(message) {
		return "", nil, false
	}
	if !hasFinanceAccess {
		return "This manager financial brief needs finance:view permission because it includes AR, cash, and supplier exposure. I can still prepare an operations-only pipeline brief if you want that version.", []ButlerAction{}, true
	}

	year := requestedBriefYear(message)
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(1, 0, 0)
	now := time.Now()
	horizon := now.AddDate(0, 0, 60)

	type rollupRow struct {
		Label    string  `gorm:"column:label"`
		Count    int64   `gorm:"column:count"`
		TotalBHD float64 `gorm:"column:total_bhd"`
	}

	var orderCount int64
	var orderTotal float64
	_ = a.db.Model(&Order{}).
		Where("order_date >= ? AND order_date < ? AND status NOT IN ?", start, end, []string{"Cancelled", "Canceled", "Void"}).
		Count(&orderCount).Error
	_ = a.db.Model(&Order{}).
		Where("order_date >= ? AND order_date < ? AND status NOT IN ?", start, end, []string{"Cancelled", "Canceled", "Void"}).
		Select("COALESCE(SUM(CASE WHEN grand_total_bhd > 0 THEN grand_total_bhd ELSE total_value_bhd END), 0)").
		Scan(&orderTotal).Error

	var activeOrderCount int64
	var activeOrderTotal float64
	_ = a.db.Model(&Order{}).
		Where("order_date >= ? AND order_date < ? AND status NOT IN ?", start, end, []string{"Delivered", "Completed", "Closed", "Cancelled", "Canceled", "Void"}).
		Count(&activeOrderCount).Error
	_ = a.db.Model(&Order{}).
		Where("order_date >= ? AND order_date < ? AND status NOT IN ?", start, end, []string{"Delivered", "Completed", "Closed", "Cancelled", "Canceled", "Void"}).
		Select("COALESCE(SUM(CASE WHEN grand_total_bhd > 0 THEN grand_total_bhd ELSE total_value_bhd END), 0)").
		Scan(&activeOrderTotal).Error

	var offerRows []rollupRow
	_ = a.db.Raw(`
		SELECT COALESCE(stage, 'Unstaged') AS label, COUNT(*) AS count, COALESCE(SUM(total_value_bhd), 0) AS total_bhd
		FROM offers
		WHERE deleted_at IS NULL
			AND quotation_date >= ?
			AND quotation_date < ?
			AND stage IN ('RFQ', 'Quoted', 'Won')
		GROUP BY COALESCE(stage, 'Unstaged')
		ORDER BY total_bhd DESC
	`, start, end).Scan(&offerRows).Error

	var offerCount int64
	var offerTotal float64
	for _, row := range offerRows {
		offerCount += row.Count
		offerTotal += row.TotalBHD
	}

	var opportunityRows []rollupRow
	_ = a.db.Raw(`
		SELECT COALESCE(stage, 'Unstaged') AS label, COUNT(*) AS count, COALESCE(SUM(revenue_bhd), 0) AS total_bhd
		FROM opportunities
		WHERE deleted_at IS NULL
			AND year = ?
			AND stage NOT IN ('Lost', 'Expired')
		GROUP BY COALESCE(stage, 'Unstaged')
		ORDER BY total_bhd DESC
	`, year).Scan(&opportunityRows).Error

	var openInvoiceAR, dueNext60, overdueAR, supplierAP float64
	openInvoiceStatuses := []string{"Sent", "PartiallyPaid", "Overdue"}
	_ = a.db.Model(&Invoice{}).
		Where("status IN ? AND outstanding_bhd > 0", openInvoiceStatuses).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&openInvoiceAR).Error
	_ = a.db.Model(&Invoice{}).
		Where("status IN ? AND outstanding_bhd > 0 AND due_date <= ?", openInvoiceStatuses, horizon).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&dueNext60).Error
	_ = a.db.Model(&Invoice{}).
		Where("status IN ? AND outstanding_bhd > 0 AND due_date < ?", openInvoiceStatuses, now).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&overdueAR).Error
	_ = a.db.Model(&SupplierInvoice{}).
		Where("status NOT IN ? AND COALESCE(payment_status, '') != ?", []string{"Paid", "Rejected", "Dispute"}, "Paid").
		Select("COALESCE(SUM(total_bhd), 0)").
		Scan(&supplierAP).Error

	uninvoicedOrderExposure, pendingOrderCount, _ := a.calculateUninvoicedOrderExposure(start, end)
	predicted60DayAR := dueNext60 + uninvoicedOrderExposure

	cashBalance := 0.0
	cashNote := "No latest bank statement balance was available."
	if cash, err := computeCashPositionSnapshot(a); err == nil && cash != nil {
		cashBalance = cash.CashBalanceBHD
		cashNote = fmt.Sprintf("Latest statement cash balance is %s BHD.", formatBHD(cashBalance))
		if len(cash.Notices) > 0 {
			cashNote += " Statement notices: " + strings.Join(cash.Notices, " ")
		}
	}

	stageLines := make([]string, 0, len(offerRows))
	for _, row := range offerRows {
		stageLines = append(stageLines, fmt.Sprintf("- %s offers: %d records, %s BHD", firstNonEmpty(row.Label, "Unstaged"), row.Count, formatBHD(row.TotalBHD)))
	}
	if len(stageLines) == 0 {
		stageLines = append(stageLines, "- No RFQ, Quoted, or Won offers found for this year.")
	}

	opportunityLines := make([]string, 0, len(opportunityRows))
	for _, row := range opportunityRows {
		opportunityLines = append(opportunityLines, fmt.Sprintf("- %s opportunities: %d records, %s BHD", firstNonEmpty(row.Label, "Unstaged"), row.Count, formatBHD(row.TotalBHD)))
	}
	if len(opportunityLines) == 0 {
		opportunityLines = append(opportunityLines, "- No active opportunity rows found for this year.")
	}

	liquidityGap := cashBalance + dueNext60 - supplierAP
	reportText := fmt.Sprintf(`Manager financial brief - FY%d order, offer, and AR outlook

As of %s

Commercial position
- Confirmed orders: %d orders, %s BHD.
- Active order backlog: %d orders, %s BHD.
- Offer coverage: %d RFQ/Quoted/Won offers, %s BHD.
- Uninvoiced confirmed-order exposure: %d orders, %s BHD.

Offer pipeline by stage
%s

Opportunity pipeline by stage
%s

AR outlook for the next 60 days
- Current open invoice AR: %s BHD.
- Already overdue AR: %s BHD.
- Invoice AR due by %s: %s BHD.
- Likely invoiceable confirmed-order exposure: %s BHD.
- Conservative 60-day AR focus: %s BHD.

Financial standing
- %s
- Open supplier payable exposure: %s BHD.
- Cash plus 60-day invoice AR less open supplier exposure: %s BHD.

Manager interpretation
- FY%d has real commercial coverage even where posted invoice revenue is still light.
- Collections should separate existing invoice AR from uninvoiced order exposure; they are not the same risk.
- The next two months should focus on turning active orders into clean invoices, then chasing existing overdue invoices.
- This document was generated from local ERP data without waiting for the external model.`, year, now.Format("2 Jan 2006 15:04"), orderCount, formatBHD(orderTotal), activeOrderCount, formatBHD(activeOrderTotal), offerCount, formatBHD(offerTotal), pendingOrderCount, formatBHD(uninvoicedOrderExposure), strings.Join(stageLines, "\n"), strings.Join(opportunityLines, "\n"), formatBHD(openInvoiceAR), formatBHD(overdueAR), horizon.Format("2 Jan 2006"), formatBHD(dueNext60), formatBHD(uninvoicedOrderExposure), formatBHD(predicted60DayAR), cashNote, formatBHD(supplierAP), formatBHD(liquidityGap), year)

	actions := []ButlerAction{}
	if strings.Contains(strings.ToLower(message), "pdf") ||
		strings.Contains(strings.ToLower(message), "document") ||
		strings.Contains(strings.ToLower(message), "report") {
		filePath, err := a.generateIntelligenceReportPDF("financial", fmt.Sprintf("FY%d Pipeline AR Outlook", year), map[string]any{
			"manager_financial_brief": map[string]any{
				"year":                       year,
				"confirmed_order_value_bhd":  round3(orderTotal),
				"offer_pipeline_bhd":         round3(offerTotal),
				"open_invoice_ar_bhd":        round3(openInvoiceAR),
				"uninvoiced_order_ar_bhd":    round3(uninvoicedOrderExposure),
				"predicted_60_day_ar_bhd":    round3(predicted60DayAR),
				"cash_balance_bhd":           round3(cashBalance),
				"open_supplier_payables_bhd": round3(supplierAP),
			},
		}, reportText)
		if err == nil {
			reportText += fmt.Sprintf("\n\nGenerated document\n- %s", filePath)
			actions = append(actions, ButlerAction{Type: "fetch", Target: "report", Label: "Open manager financial brief", Data: filePath})
		} else {
			reportText += fmt.Sprintf("\n\nDocument note\n- I prepared the brief in chat, but PDF generation failed locally: %v", err)
		}
	}

	return reportText, actions, true
}

func (a *App) buildGroundedModelFallbackResponse(intent Intent, message string, context map[string]any, hasFinanceAccess bool, backendErr error) ButlerResponse {
	if msg, actions, handled := a.tryGroundedManagerFinancialBriefFastPath(intent, message, hasFinanceAccess); handled {
		return ButlerResponse{
			Message:    msg,
			Actions:    actions,
			Confidence: 0.88,
			Context:    map[string]any{"fast_path": "grounded_manager_financial_brief", "model_error": fmt.Sprint(backendErr)},
			Metadata:   buildButlerMetadata(intent, context, hasFinanceAccess, "grounded_sql_fallback", "", "", fmt.Sprint(backendErr), backendErr),
		}
	}
	if msg, handled := a.tryGroundedCapabilitiesFastPath(intent, message, hasFinanceAccess); handled {
		return ButlerResponse{
			Message:    msg,
			Actions:    []ButlerAction{},
			Confidence: 0.9,
			Context:    map[string]any{"fast_path": "grounded_capabilities", "model_error": fmt.Sprint(backendErr)},
			Metadata:   buildButlerMetadata(intent, context, hasFinanceAccess, "grounded_sql_fallback", "", "", fmt.Sprint(backendErr), backendErr),
		}
	}

	summary := a.getBusinessSummary()
	customerCount := fmt.Sprint(summary["total_customers"])
	supplierCount := fmt.Sprint(summary["total_suppliers"])
	orderCount := fmt.Sprint(summary["total_orders"])
	invoiceCount := fmt.Sprint(summary["total_invoices"])
	databaseInventory := a.getDatabaseInventoryContext(hasFinanceAccess)
	tableCount := fmt.Sprint(databaseInventory["total_tables"])
	moduleCoverage := "relationships, commercial pipeline, operations, work management, products, intelligence, and system records"
	if hasFinanceAccess {
		moduleCoverage = "relationships, commercial pipeline, operations, finance, banking, payroll, work management, products, intelligence, and system records"
	}

	lines := []string{
		"The external intelligence model did not respond quickly, so I am answering from local ERP data.",
		"",
		"Current local coverage",
		fmt.Sprintf("- Customers: %s", customerCount),
		fmt.Sprintf("- Suppliers: %s", supplierCount),
		fmt.Sprintf("- Orders: %s", orderCount),
		fmt.Sprintf("- Customer invoices: %s", invoiceCount),
		fmt.Sprintf("- Queryable database tables: %s", tableCount),
		fmt.Sprintf("- Covered modules: %s", moduleCoverage),
	}

	if hasFinanceAccess {
		if yearSummary := a.getBusinessYearSummary(message); len(yearSummary) > 0 {
			lines = append(lines,
				"",
				"Year coverage",
				fmt.Sprintf("- Period: %v", yearSummary["period"]),
				fmt.Sprintf("- Invoices: %v", yearSummary["invoices"]),
				fmt.Sprintf("- Orders: %v", yearSummary["orders"]),
				fmt.Sprintf("- Offers: %v", yearSummary["offers"]),
				fmt.Sprintf("- Opportunities: %v", yearSummary["opportunities"]),
			)
		}
	}

	lines = append(lines,
		"",
		"What I can answer locally",
		"- Customers, suppliers, contacts, notes, offers, opportunities, RFQs, orders, purchase orders, deliveries, products, serials, tasks, and notifications.",
		"- Finance, AR, AP, payments, bank statements, expenses, payroll, VAT, and cash position when your role has finance:view.",
		"- Manager reports, pipeline summaries, relationship history, task follow-ups, and cross-module record lookup from guarded ERP queries.",
	)

	return ButlerResponse{
		Message:    strings.Join(lines, "\n"),
		Actions:    []ButlerAction{},
		Confidence: 0.72,
		Context:    context,
		Metadata:   buildButlerMetadata(intent, context, hasFinanceAccess, "grounded_sql_fallback", "", "", fmt.Sprint(backendErr), backendErr),
	}
}

func (a *App) tryGroundedCustomerFastPath(intent Intent, message string) (string, bool) {
	return a.tryGroundedCustomerFastPathWithHint(intent, message, "")
}

func (a *App) tryGroundedTaskCreationFastPath(intent Intent, message string) (string, bool) {
	if a != nil {
		if msg, handled := a.butlerFastpathService().TryTaskCreation(intent, message); handled {
			return msg, true
		}
	}

	if a == nil || a.db == nil {
		return "", false
	}

	assigneeRef, taskDetails, matched := parseGroundedTaskCreationRequest(intent, message)
	if !matched {
		return "", false
	}
	if strings.TrimSpace(assigneeRef) == "" || strings.TrimSpace(taskDetails) == "" {
		return "I can create that task, but I need both the assignee and the task details.", true
	}

	assignee := a.resolveEmployeeReference(assigneeRef)
	if assignee == nil || strings.TrimSpace(assignee.EntityID) == "" {
		return fmt.Sprintf("I understood that as a task creation request, but I could not resolve %s to an active employee.", assigneeRef), true
	}

	title := strings.TrimSpace(taskDetails)
	if title == "" {
		return "I understood the assignee, but the task title/details were blank after parsing the request.", true
	}

	taskType := "general"
	if strings.Contains(strings.ToLower(taskDetails), "follow up") || strings.Contains(strings.ToLower(taskDetails), "follow-up") {
		taskType = "follow_up"
	}

	var customerID *string
	if customerRef := inferCustomerReferenceFromQuery(Intent{}, strings.ToLower(taskDetails)); strings.TrimSpace(customerRef) != "" {
		if customerIDs, _ := a.butlerContextService().ResolveCustomerScope(customerRef); len(customerIDs) > 0 {
			customerID = &customerIDs[0]
		}
	}

	assigneeID := assignee.EntityID
	createdTask, err := a.CreateCollaborativeTask(TaskItem{
		Title:              title,
		Description:        taskDetails,
		TaskType:           taskType,
		Priority:           "medium",
		AssigneeEmployeeID: &assigneeID,
		CustomerID:         customerID,
	})
	if err != nil {
		return fmt.Sprintf("I understood that as a task request for %s, but I could not create it: %v", assignee.DisplayName, err), true
	}

	var notificationCount int64
	_ = a.db.Model(&Notification{}).
		Where("employee_id = ? AND source_type = ? AND source_id = ? AND notification_type = ?", assigneeID, "task", createdTask.ID, "task").
		Count(&notificationCount).Error

	response := fmt.Sprintf("Created the task %q for %s.", createdTask.Title, assignee.DisplayName)
	if notificationCount > 0 {
		response += " The assignment notification was queued as well."
	} else {
		response += " The task is created, but I could not confirm the assignment notification row yet."
	}
	if createdTask.CustomerID != nil && strings.TrimSpace(*createdTask.CustomerID) != "" {
		response += fmt.Sprintf(" I also linked it to customer %s.", *createdTask.CustomerID)
	}

	return response, true
}

func (a *App) tryGroundedWorkFastPath(intent Intent, message string) (string, bool) {
	if a == nil || a.db == nil {
		return "", false
	}

	q := strings.ToLower(strings.TrimSpace(message))
	if q == "" {
		return "", false
	}

	workSignals := []string{
		"task", "tasks", "workload", "notification", "notifications", "assigned",
		"assignee", "my work", "team board", "blocked", "open task", "open tasks",
	}
	matchedSignal := false
	for _, signal := range workSignals {
		if strings.Contains(q, signal) {
			matchedSignal = true
			break
		}
	}
	if !matchedSignal {
		return "", false
	}

	if isSelfWorkReference(q) {
		current, err := a.GetCurrentEmployeeContext()
		if err == nil && strings.TrimSpace(current.EmployeeID) != "" {
			displayName := firstNonEmpty(strings.TrimSpace(current.EmployeeName), "you")
			return a.buildEmployeeTaskOverviewResponse(current.EmployeeID, displayName, q), true
		}
	}

	employeeRef := inferEmployeeReferenceFromQuery(intent, message)
	if employeeRef == "" {
		return "", false
	}

	resolution := a.resolveEmployeeReference(employeeRef)
	if resolution == nil || strings.TrimSpace(resolution.EntityID) == "" {
		return "", false
	}

	return a.buildEmployeeTaskOverviewResponse(resolution.EntityID, resolution.DisplayName, q), true
}

func isSelfWorkReference(q string) bool {
	selfSignals := []string{
		" my task", " my tasks", " for me", "assigned to me", "task for me", "tasks for me",
		"new task for me", "new tasks for me", "me today", "my work", "my workload",
	}
	padded := " " + strings.ToLower(strings.TrimSpace(q)) + " "
	for _, signal := range selfSignals {
		if strings.Contains(padded, signal) {
			return true
		}
	}
	return strings.Contains(padded, " tasks today ") || strings.Contains(padded, " task today ")
}

func (a *App) tryGroundedSupplierFastPath(intent Intent, message string) (string, bool) {
	if a == nil || a.db == nil {
		return "", false
	}

	q := strings.ToLower(strings.TrimSpace(message))
	if q == "" {
		return "", false
	}

	supplierSignals := []string{
		"supplier", "suppliers", "buy from", "bought from", "purchase", "purchases",
		"payment history", "supplier payment", "supplier payments", "issue", "issues",
	}
	matchedSignal := false
	for _, signal := range supplierSignals {
		if strings.Contains(q, signal) {
			matchedSignal = true
			break
		}
	}
	if !matchedSignal {
		return "", false
	}

	supplierRef := inferSupplierReferenceFromQuery(intent, message)
	if supplierRef == "" {
		return "", false
	}

	resolution := a.resolveSupplierReference(supplierRef)
	if resolution == nil || strings.TrimSpace(resolution.EntityID) == "" {
		return "", false
	}

	switch {
	case strings.Contains(q, "payment history") || strings.Contains(q, "supplier payment"):
		return a.butlerContextService().BuildSupplierPaymentHistoryResponse(resolution.EntityID, resolution.DisplayName), true
	case strings.Contains(q, "issue"):
		return a.butlerContextService().BuildSupplierIssueOverviewResponse(resolution.EntityID, resolution.DisplayName), true
	case strings.Contains(q, "buy from") || strings.Contains(q, "bought from") || strings.Contains(q, "purchase"):
		return a.butlerContextService().BuildSupplierPurchaseOverviewResponse(resolution.EntityID, resolution.DisplayName), true
	default:
		return "", false
	}
}

func (a *App) tryGroundedCustomerFastPathWithHint(intent Intent, message, customerHint string) (string, bool) {
	if a == nil || a.db == nil {
		return "", false
	}

	q := strings.ToLower(strings.TrimSpace(message))
	if q == "" {
		return "", false
	}

	if (strings.Contains(q, "all three are same") || strings.Contains(q, "all are the same") || strings.Contains(q, "same customer")) && strings.TrimSpace(customerHint) != "" {
		_, displayName := a.butlerContextService().ResolveCustomerScope(customerHint)
		if strings.TrimSpace(displayName) == "" {
			displayName = strings.TrimSpace(customerHint)
		}
		return fmt.Sprintf("Understood. I will treat those records as a single customer group (%s) for this chat and proceed with grouped data/actions.", displayName), true
	}

	if strings.Contains(q, "revenue projection") || strings.Contains(q, "revenue projections") || strings.Contains(q, "projected revenue") {
		return a.butlerContextService().BuildRevenueProjectionResponse(), true
	}

	customerRef := inferCustomerReferenceFromQuery(intent, q)
	if customerRef == "" && strings.TrimSpace(customerHint) != "" {
		customerRef = strings.TrimSpace(customerHint)
	}
	if customerRef == "" {
		return "", false
	}

	customerIDs, displayName := a.butlerContextService().ResolveCustomerScope(customerRef)
	if len(customerIDs) == 0 && strings.TrimSpace(customerHint) != "" && !strings.EqualFold(strings.TrimSpace(customerHint), strings.TrimSpace(customerRef)) {
		customerIDs, displayName = a.butlerContextService().ResolveCustomerScope(customerHint)
	}
	if len(customerIDs) == 0 {
		return "", false
	}

	if strings.Contains(q, "invoice") && (strings.Contains(q, "this quarter") || strings.Contains(q, "current quarter")) {
		return a.butlerContextService().BuildCustomerQuarterInvoiceAndHandlerResponse(customerIDs, displayName, q), true
	}

	if strings.Contains(q, "note") || strings.Contains(q, "notes") {
		return a.butlerContextService().BuildCustomerNotesResponse(customerIDs, displayName), true
	}

	if strings.Contains(q, "line item") || strings.Contains(q, "line items") || strings.Contains(q, "what we have sold") || strings.Contains(q, "what have we sold") || strings.Contains(q, "sold to them") || strings.Contains(q, "sold to") || strings.Contains(q, "equipment sold") || strings.Contains(q, "what sold") || strings.Contains(q, "products sold") {
		return a.butlerContextService().BuildCustomerLineItemsResponse(customerIDs, displayName), true
	}

	if strings.Contains(q, "invoice") {
		return a.butlerContextService().BuildCustomerInvoiceOverviewResponse(customerIDs, displayName), true
	}

	if strings.Contains(q, "offer") && strings.Contains(q, "this year") {
		return a.butlerContextService().BuildCustomerYearOffersResponse(customerIDs, displayName, time.Now().Year()), true
	}

	if strings.Contains(q, "offer") {
		return a.butlerContextService().BuildCustomerOfferOverviewResponse(customerIDs, displayName), true
	}

	return "", false
}

func inferEmployeeReferenceFromQuery(intent Intent, message string) string {
	if strings.TrimSpace(intent.PersonName) != "" {
		return strings.TrimSpace(intent.PersonName)
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:tasks?\s+assigned\s+to|assigned\s+to|workload\s+for|notifications?\s+for|tasks?\s+for)\s+([A-Z][A-Za-z.\-]+(?:\s+[A-Z][A-Za-z.\-]+){0,2})`),
		regexp.MustCompile(`(?i)(?:notifications?\s+does|tasks?\s+does)\s+([A-Z][A-Za-z.\-]+(?:\s+[A-Z][A-Za-z.\-]+){0,2})\s+have`),
		regexp.MustCompile(`(?i)\b([A-Z][A-Za-z.\-]+(?:\s+[A-Z][A-Za-z.\-]+){0,2})'s\s+(?:tasks?|notifications?)\b`),
	}
	for _, pattern := range patterns {
		if m := pattern.FindStringSubmatch(message); len(m) > 1 {
			return strings.TrimSpace(m[1])
		}
	}

	return ""
}

func parseGroundedTaskCreationRequest(intent Intent, message string) (string, string, bool) {
	return butlerfastpath.ParseTaskCreationRequest(intent, message)
}

func sanitizeTaskCreationDetails(details string) string {
	return butlerfastpath.SanitizeTaskCreationDetails(details)
}

func inferCustomerReferenceFromQuery(intent Intent, q string) string {
	if strings.TrimSpace(intent.EntityName) != "" {
		return strings.TrimSpace(intent.EntityName)
	}

	// Preposition-led reference: "... for/from/to/with <name> ...".
	re := regexp.MustCompile(`\b(?:for|from|to|with)\s+([a-z0-9&\.\-\s]{2,60})`)
	if m := re.FindStringSubmatch(q); len(m) == 2 {
		if candidate := trimCustomerReferenceCandidate(m[1]); candidate != "" {
			return candidate
		}
	}

	// Fallback: "show me <name> invoices/offers/orders/notes" phrasing where no
	// preposition precedes the name. Strip a leading filler verb, then cut at the
	// first document/time keyword. The downstream resolver does a fuzzy LIKE match,
	// so a partial-but-distinctive reference is enough.
	stripped := q
	for _, prefix := range []string{"show me ", "show ", "give me ", "list ", "get me ", "get ", "find ", "pull up ", "what are ", "what is "} {
		if strings.HasPrefix(stripped, prefix) {
			stripped = strings.TrimPrefix(stripped, prefix)
			break
		}
	}
	if stripped != q {
		if candidate := trimCustomerReferenceCandidate(stripped); candidate != "" {
			return candidate
		}
	}

	return ""
}

// trimCustomerReferenceCandidate cleans an extracted customer reference by cutting
// it at the first document-type or time-window keyword and trimming punctuation.
func trimCustomerReferenceCandidate(s string) string {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	cutKeywords := []string{
		" invoices", " invoice", " offers", " offer", " orders", " order",
		" notes", " note", " line item", " line-item", " payment", " quotation",
		" this quarter", " current quarter", " this year", " sold", " bought",
	}
	cutAt := len(s)
	for _, kw := range cutKeywords {
		if idx := strings.Index(lower, kw); idx >= 0 && idx < cutAt {
			cutAt = idx
		}
	}
	s = strings.TrimSpace(s[:cutAt])
	s = strings.Trim(s, ".,?'\"")
	return strings.TrimSpace(s)
}

func inferSupplierReferenceFromQuery(intent Intent, message string) string {
	if candidate := strings.TrimSpace(intent.EntityName); candidate != "" {
		lowerCandidate := strings.ToLower(candidate)
		if !strings.Contains(lowerCandidate, "issue") &&
			!strings.Contains(lowerCandidate, "payment") &&
			!strings.Contains(lowerCandidate, "history") &&
			!strings.Contains(lowerCandidate, "purchase") {
			return candidate
		}
	}

	q := strings.TrimSpace(message)
	if q == "" {
		return ""
	}

	re := regexp.MustCompile(`(?i)\b(?:about|from|for)\s+([^?]+)`)
	m := re.FindStringSubmatch(q)
	if len(m) != 2 {
		return ""
	}

	candidate := strings.TrimSpace(m[1])
	trimSuffixes := []string{
		"payment history",
		"payments",
		"payment",
		"issues",
		"issue",
		"purchase history",
		"purchases",
	}
	for _, suffix := range trimSuffixes {
		candidate = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(candidate), suffix))
	}
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return ""
	}
	return candidate
}

func (a *App) buildEmployeeTaskOverviewResponse(employeeID, employeeName, q string) string {
	type taskRow struct {
		Title       string
		Status      string
		Priority    string
		TaskType    string
		Description string
		DueDate     *time.Time
	}

	var tasks []taskRow
	a.db.Table("task_items").
		Select("title, status, priority, task_type, description, due_date").
		Where("assignee_employee_id = ?", employeeID).
		Order("updated_at DESC, created_at DESC").
		Limit(20).
		Scan(&tasks)

	openCount := 0
	completedCount := 0
	blockedCount := 0
	overdueCount := 0
	now := time.Now()
	lines := make([]string, 0, minInt(5, len(tasks)))
	for _, task := range tasks {
		status := strings.ToLower(strings.TrimSpace(task.Status))
		switch status {
		case "done", "completed", "closed":
			completedCount++
		default:
			openCount++
		}
		if status == "blocked" {
			blockedCount++
		}
		if task.DueDate != nil && task.DueDate.Before(now) && status != "done" && status != "completed" && status != "closed" {
			overdueCount++
		}
		if len(lines) < 5 {
			line := fmt.Sprintf("- %s | %s | %s", strings.TrimSpace(task.Title), strings.TrimSpace(task.Status), strings.TrimSpace(task.Priority))
			if task.DueDate != nil {
				line += " | due " + task.DueDate.Format("2006-01-02")
			}
			lines = append(lines, line)
		}
	}

	type notificationRow struct {
		Title      string
		Status     string
		SourceType string
		CreatedAt  time.Time
	}
	var notifications []notificationRow
	a.db.Table("notifications").
		Select("title, status, source_type, created_at").
		Where("employee_id = ?", employeeID).
		Order("created_at DESC").
		Limit(20).
		Scan(&notifications)

	unreadNotifications := 0
	notificationLines := make([]string, 0, 3)
	for _, notification := range notifications {
		if strings.EqualFold(strings.TrimSpace(notification.Status), "unread") {
			unreadNotifications++
		}
		if len(notificationLines) < 3 {
			notificationLines = append(notificationLines, fmt.Sprintf("- %s | %s | %s", strings.TrimSpace(notification.Title), strings.TrimSpace(notification.Status), notification.CreatedAt.Format("2006-01-02")))
		}
	}

	if len(tasks) == 0 && len(notifications) == 0 {
		return fmt.Sprintf("I checked the collaborative work records for %s and I cannot confirm any assigned tasks or notifications from current data.", employeeName)
	}

	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s currently has %d active task(s)", employeeName, openCount))
	if completedCount > 0 {
		builder.WriteString(fmt.Sprintf(", %d completed task(s)", completedCount))
	}
	builder.WriteString(fmt.Sprintf(", %d blocked task(s), and %d overdue task(s).", blockedCount, overdueCount))
	builder.WriteString(fmt.Sprintf(" I also found %d unread notification(s).", unreadNotifications))

	if len(lines) > 0 {
		builder.WriteString("\n\nRecent assigned tasks:\n")
		builder.WriteString(strings.Join(lines, "\n"))
	}

	if strings.Contains(q, "notification") && len(notificationLines) > 0 {
		builder.WriteString("\n\nRecent notifications:\n")
		builder.WriteString(strings.Join(notificationLines, "\n"))
	}

	return builder.String()
}

func (a *App) tryGroundedOfferDraftFastPath(message, customerHint, productHint string) (string, []ButlerAction, bool) {
	if a != nil {
		if msg, actions, handled := a.butlerFastpathService().TryOfferDraft(message, customerHint, productHint); handled {
			return msg, actions, true
		}
	}

	q := strings.ToLower(strings.TrimSpace(message))
	if q == "" {
		return "", nil, false
	}
	if !strings.Contains(q, "quantity") && !strings.Contains(q, "qty") {
		return "", nil, false
	}
	if !strings.Contains(q, "price") && !strings.Contains(q, "bhd") {
		return "", nil, false
	}
	if strings.TrimSpace(productHint) == "" {
		// Strip "for [customer]" phrase from query before product extraction to avoid
		// capturing "NPC for Probe FMP51" instead of just "Probe FMP51"
		productQuery := q
		if strings.TrimSpace(customerHint) != "" {
			custLower := strings.ToLower(strings.TrimSpace(customerHint))
			// Remove "for customer" or "to customer" as a unit
			for _, prep := range []string{"for ", "to "} {
				productQuery = strings.Replace(productQuery, prep+custLower, "", 1)
			}
		}
		productHint = inferProductHintFromText(productQuery)
	}
	if strings.TrimSpace(customerHint) == "" || strings.TrimSpace(productHint) == "" {
		return "", nil, false
	}

	qty, okQty := extractNumericToken(q, `(?i)(?:quantity|qty)\s*(?:is|=|:|would\s+be|to\s+be)?\s*([0-9]+(?:\.[0-9]+)?)`)
	price, okPrice := extractNumericToken(q, `(?i)(?:price(?:\s+per\s+unit)?|unit price)\s*(?:is|=|at|:|would\s+be|to\s+be)?\s*([0-9]+(?:\.[0-9]+)?)`)
	if !okPrice {
		price, okPrice = extractNumericToken(q, `(?i)([0-9]+(?:\.[0-9]+)?)\s*bhd`)
	}
	if !okQty || !okPrice || qty <= 0 || price <= 0 {
		return "", nil, false
	}

	customerIDs, displayName := a.butlerContextService().ResolveCustomerScope(customerHint)
	if len(customerIDs) == 0 {
		return "", nil, false
	}

	total := qty * price
	action := ButlerAction{
		Type:   "create_offer_draft",
		Target: "quotation",
		Label:  "Create offer draft",
		Data: map[string]any{
			"customer_id":   customerIDs[0],
			"customer_name": displayName,
			"amount":        total,
			"line_items": []map[string]any{
				{
					"equipment":      productHint,
					"description":    productHint,
					"quantity":       qty,
					"unit_price_bhd": price,
					"total_price":    total,
				},
			},
		},
	}

	msg := fmt.Sprintf("Prepared a grounded offer draft for %s:\n- Item: %s\n- Quantity: %.2f\n- Unit Price: %.3f BHD\n- Line Total: %.3f BHD\n\nRun the action to create the draft in the offers module.",
		displayName, productHint, qty, price, total)
	return msg, []ButlerAction{action}, true
}

func inferProductHintFromText(q string) string {
	return butlerfastpath.InferProductHintFromText(q)
}

func extractNumericToken(text, pattern string) (float64, bool) {
	return butlerfastpath.ExtractNumericToken(text, pattern)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
