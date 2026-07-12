package main

import (
	"time"

	butlercontext "ph_holdings_app/pkg/butler/context"
)

// BUTLER CONTEXT — host side.
// ============================================================================
//
// The context builders moved to pkg/butler/context in Wave 6 (Mission A.1).
// What remains here is the hub-facing half: RBAC computation at the entry
// points, the collaboration-hub reads (work data, employees, task threads,
// quick captures), the finance-reporting cashflow projection, and the
// sales-pipeline opportunity dedup — all served to the package through
// butlercontext.HostPort (ports, never relocation: W4-D9).

func (a *App) butlerContextService() *butlercontext.Service {
	return butlercontext.New(a.db, appButlerContextHost{app: a})
}

// appButlerContextHost adapts root hub reads to butlercontext.HostPort.
type appButlerContextHost struct {
	app *App
}

func (h appButlerContextHost) WorkContext(intent Intent) map[string]any {
	return h.app.getWorkContext(intent)
}

func (h appButlerContextHost) ResolveEmployeeReference(reference string) *ButlerResolvedEntity {
	return h.app.resolveEmployeeReference(reference)
}

func (h appButlerContextHost) EmployeeContext(resolution *ButlerResolvedEntity) map[string]any {
	return h.app.getEmployeeContext(resolution)
}

func (h appButlerContextHost) RecentOpenQuickCaptures() []map[string]any {
	return h.app.getRecentOpenQuickCaptures()
}

func (h appButlerContextHost) CashflowProjectionContext() map[string]any {
	return h.app.getCashflowProjectionContext()
}

func (h appButlerContextHost) OpenDedupedOpportunities() []Opportunity {
	return h.app.openDedupedOpportunities()
}

// --- Thin delegates: RBAC is computed HERE (the host owns permissions) and
// passed down; the builders themselves are read-only by construction.

func (a *App) buildIntentContext(intent Intent) map[string]any {
	return a.butlerContextService().BuildIntentContext(intent, a.requirePermission("finance:view") == nil)
}

func (a *App) buildFullContext(intent Intent) map[string]any {
	return a.butlerContextService().BuildFullContext(intent, a.requirePermission("finance:view") == nil)
}

func (a *App) getDatabaseInventoryContext(hasFinanceAccess bool) map[string]any {
	return a.butlerContextService().DatabaseInventoryContext(hasFinanceAccess)
}

func (a *App) resolveBestEntityReference(intent Intent) *ButlerResolvedEntity {
	return a.butlerContextService().ResolveBestEntityReference(intent)
}

func (a *App) resolveCustomerReference(reference string) *ButlerResolvedEntity {
	return a.butlerContextService().ResolveCustomerReference(reference)
}

func (a *App) resolveSupplierReference(reference string) *ButlerResolvedEntity {
	return a.butlerContextService().ResolveSupplierReference(reference)
}

func (a *App) getBusinessSummary() map[string]any {
	return a.butlerContextService().BusinessSummary()
}

func (a *App) getBusinessYearSummary(query string) map[string]any {
	return a.butlerContextService().BusinessYearSummary(query)
}

func (a *App) calculateSystemRegime() map[string]any {
	return a.butlerContextService().CalculateSystemRegime()
}

// --- Pure helpers whose canonical bodies moved with the context builders;
// these names keep the existing root call sites working.

func firstNonEmpty(values ...string) string {
	return butlercontext.FirstNonEmpty(values...)
}

func uniqueNonEmptyStrings(values ...string) []string {
	return butlercontext.UniqueNonEmptyStrings(values...)
}

func round3(value float64) float64 {
	return butlercontext.Round3(value)
}

func formatOptionalDate(ts *time.Time) string {
	return butlercontext.FormatOptionalDate(ts)
}

func firstNonEmptyStringPointer(value *string) string {
	return butlercontext.FirstNonEmptyStringPointer(value)
}

func buildContextSearchTerms(intent Intent) []string {
	return butlercontext.BuildContextSearchTerms(intent)
}

func selectContextEntries(entries []map[string]any, terms []string, limit int) []map[string]any {
	return butlercontext.SelectContextEntries(entries, terms, limit)
}

// openDedupedOpportunities returns the open pipeline with the OneDrive/OCR
// normalization and cross-source dedup applied — the same block that fed
// the forecast context before the Wave 6 peel; the normalization helpers
// live with the sales-pipeline cluster.
func (a *App) openDedupedOpportunities() []Opportunity {
	var allOpportunities []Opportunity
	a.db.Find(&allOpportunities)

	closedKeys := make(map[string]bool)
	for _, opp := range allOpportunities {
		normalized := normalizeOpportunityForList(opp)
		if normalized.Stage != "Won" && normalized.Stage != "Lost" {
			continue
		}
		key := canonicalOpportunityKey(normalized)
		if key != "" {
			closedKeys[key] = true
		}
	}

	openByKey := make(map[string]Opportunity)
	for _, opp := range allOpportunities {
		normalized := normalizeOpportunityForList(opp)
		if shouldSuppressSyntheticOCR(normalized) {
			continue
		}
		if normalized.Stage == "Won" || normalized.Stage == "Lost" || normalized.Stage == "Expired" {
			continue
		}
		key := canonicalOpportunityKey(normalized)
		if key != "" && closedKeys[key] {
			continue
		}
		if key == "" {
			key = normalized.ID
		}
		existing, exists := openByKey[key]
		if !exists || shouldPreferOpportunity(normalized, existing) {
			openByKey[key] = normalized
		}
	}

	openOpportunities := make([]Opportunity, 0, len(openByKey))
	for _, opp := range openByKey {
		openOpportunities = append(openOpportunities, opp)
	}
	return openOpportunities
}

// --- Collaboration-hub and finance-reporting reads (the HostPort bodies).

func (a *App) getWorkContext(intent Intent) map[string]any {
	result := make(map[string]any)
	if a.db == nil {
		return result
	}

	terms := buildContextSearchTerms(intent)

	var employees []Employee
	_ = a.db.Where("is_active = ?", true).Order("full_name ASC").Limit(25).Find(&employees).Error
	employeeEntries := make([]map[string]any, 0, len(employees))
	employeeNames := make(map[string]string, len(employees))
	for _, employee := range employees {
		employeeNames[employee.ID] = employee.FullName
		employeeEntries = append(employeeEntries, map[string]any{
			"employee_id": employee.ID,
			"name":        employee.FullName,
			"department":  employee.Department,
			"job_title":   employee.JobTitle,
			"status":      employee.EmploymentStatus,
			"notes":       employee.Notes,
		})
	}
	result["employees"] = selectContextEntries(employeeEntries, terms, 8)

	var tasks []TaskItem
	_ = a.db.Order("updated_at DESC").Limit(30).Find(&tasks).Error
	taskEntries := make([]map[string]any, 0, len(tasks))
	taskIDs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
		taskEntries = append(taskEntries, map[string]any{
			"task_id":        task.ID,
			"title":          task.Title,
			"description":    task.Description,
			"status":         task.Status,
			"priority":       task.Priority,
			"task_type":      task.TaskType,
			"blocked_reason": task.BlockedReason,
			"due_date":       formatOptionalDate(task.DueDate),
			"creator_name":   employeeNames[task.CreatorEmployeeID],
			"assignee_name":  employeeNames[firstNonEmptyStringPointer(task.AssigneeEmployeeID)],
		})
	}
	selectedTasks := selectContextEntries(taskEntries, terms, 10)
	result["tasks"] = selectedTasks

	taskIDTitle := make(map[string]string, len(tasks))
	for _, task := range tasks {
		taskIDTitle[task.ID] = task.Title
	}

	var comments []TaskComment
	if len(taskIDs) > 0 {
		_ = a.db.Where("task_id IN ?", taskIDs).Order("created_at DESC").Limit(25).Find(&comments).Error
	}
	commentEntries := make([]map[string]any, 0, len(comments))
	for _, comment := range comments {
		commentEntries = append(commentEntries, map[string]any{
			"task_id":    comment.TaskID,
			"task_title": taskIDTitle[comment.TaskID],
			"employee":   employeeNames[comment.EmployeeID],
			"body":       comment.Body,
			"created_at": comment.CreatedAt.Format("2006-01-02"),
		})
	}
	result["task_comments"] = selectContextEntries(commentEntries, terms, 8)

	var notifications []Notification
	_ = a.db.Where("source_type = ?", "task").Order("created_at DESC").Limit(20).Find(&notifications).Error
	notificationEntries := make([]map[string]any, 0, len(notifications))
	for _, notification := range notifications {
		notificationEntries = append(notificationEntries, map[string]any{
			"notification_id": notification.ID,
			"title":           notification.Title,
			"message":         notification.Message,
			"status":          notification.Status,
			"employee_id":     notification.EmployeeID,
			"source_id":       notification.SourceID,
			"created_at":      notification.CreatedAt.Format("2006-01-02"),
		})
	}
	result["task_notifications"] = selectContextEntries(notificationEntries, terms, 8)
	result["search_terms"] = terms

	return result
}
func (a *App) getRecentOpenQuickCaptures() []map[string]any {
	var captures []QuickCapture
	a.db.Where("status IN ?", []string{"Open", "In Progress"}).
		Order("created_at DESC").
		Limit(5).
		Find(&captures)

	items := make([]map[string]any, 0, len(captures))
	for _, capture := range captures {
		items = append(items, map[string]any{
			"title":      capture.Title,
			"priority":   capture.Priority,
			"status":     capture.Status,
			"source":     firstNonEmpty(capture.Tags, "quick_capture"),
			"created_at": capture.CreatedAt.Format("2006-01-02"),
		})
	}
	return items
}
func (a *App) resolveEmployeeReference(reference string) *ButlerResolvedEntity {
	if reference == "" {
		return nil
	}

	var employee Employee
	exactErr := a.db.Where(
		"full_name = ? OR preferred_name = ? OR employee_code = ? OR email = ?",
		reference, reference, reference, reference,
	).First(&employee).Error
	if exactErr == nil {
		return &ButlerResolvedEntity{
			EntityType:  "employee",
			EntityID:    employee.ID,
			DisplayName: firstNonEmpty(employee.PreferredName, employee.FullName, employee.EmployeeCode),
			Confidence:  0.97,
			MatchReason: "exact employee match",
		}
	}

	escaped := escapeLikeWildcards(reference)
	pattern := "%" + escaped + "%"
	var matches []Employee
	if err := a.db.Where(
		"full_name LIKE ? ESCAPE '\\' OR preferred_name LIKE ? ESCAPE '\\' OR employee_code LIKE ? ESCAPE '\\' OR email LIKE ? ESCAPE '\\'",
		pattern, pattern, pattern, pattern,
	).Order("preferred_name, full_name").Limit(3).Find(&matches).Error; err == nil && len(matches) > 0 {
		employee = matches[0]
		resolution := &ButlerResolvedEntity{
			EntityType:  "employee",
			EntityID:    employee.ID,
			DisplayName: firstNonEmpty(employee.PreferredName, employee.FullName, employee.EmployeeCode),
			Confidence:  0.84,
			MatchReason: "fuzzy employee match",
		}
		if len(matches) > 1 {
			resolution.Ambiguous = true
			resolution.Confidence = 0.58
			resolution.MatchReason = "multiple employee matches"
			resolution.Alternatives = employeeAlternatives(matches)
		}
		return resolution
	}

	return nil
}
func (a *App) getEmployeeContext(resolution *ButlerResolvedEntity) map[string]any {
	result := make(map[string]any)
	if resolution == nil || resolution.EntityType != "employee" {
		return result
	}

	var employee Employee
	if err := a.db.First(&employee, "id = ?", resolution.EntityID).Error; err != nil {
		return result
	}

	result["employee"] = map[string]any{
		"id":                employee.ID,
		"employee_code":     employee.EmployeeCode,
		"full_name":         employee.FullName,
		"preferred_name":    employee.PreferredName,
		"department":        employee.Department,
		"job_title":         employee.JobTitle,
		"employment_status": employee.EmploymentStatus,
		"is_active":         employee.IsActive,
		"notes":             employee.Notes,
	}

	var tasks []TaskItem
	_ = a.db.Where("assignee_employee_id = ?", employee.ID).
		Order("updated_at DESC").
		Limit(10).
		Find(&tasks).Error
	taskSummary := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		taskSummary = append(taskSummary, map[string]any{
			"title":       task.Title,
			"status":      task.Status,
			"priority":    task.Priority,
			"task_type":   task.TaskType,
			"due_date":    formatOptionalDate(task.DueDate),
			"description": task.Description,
		})
	}
	result["assigned_tasks"] = taskSummary

	var notifications []Notification
	_ = a.db.Where("employee_id = ?", employee.ID).
		Order("created_at DESC").
		Limit(10).
		Find(&notifications).Error
	notificationSummary := make([]map[string]any, 0, len(notifications))
	for _, notification := range notifications {
		notificationSummary = append(notificationSummary, map[string]any{
			"title":       notification.Title,
			"message":     notification.Message,
			"status":      notification.Status,
			"source_type": notification.SourceType,
			"created_at":  notification.CreatedAt.Format("2006-01-02"),
		})
	}
	result["notifications"] = notificationSummary

	return result
}
func employeeAlternatives(matches []Employee) []map[string]any {
	alternatives := make([]map[string]any, 0, len(matches))
	for _, match := range matches {
		alternatives = append(alternatives, map[string]any{
			"entity_type":  "employee",
			"id":           match.ID,
			"display_name": firstNonEmpty(match.PreferredName, match.FullName, match.EmployeeCode),
			"department":   match.Department,
			"job_title":    match.JobTitle,
		})
	}
	return alternatives
}
func (a *App) getCashflowProjectionContext() map[string]any {
	result := make(map[string]any)
	if a.db == nil {
		return result
	}

	projection, err := a.GetCashFlowProjection(365)
	if err != nil {
		result["status"] = "unavailable"
		result["error"] = err.Error()
		return result
	}

	type monthlyBucket struct {
		Inflows  float64
		Outflows float64
	}

	buckets := make(map[string]*monthlyBucket)
	order := make([]string, 0, 12)
	for _, day := range projection.DailyProjections {
		key := day.Date.Format("2006-01")
		if buckets[key] == nil {
			buckets[key] = &monthlyBucket{}
			order = append(order, key)
		}
		buckets[key].Inflows += day.ExpectedInflows
		buckets[key].Outflows += day.ExpectedOutflows
	}

	staticOpex := 16500.0
	monthly := make([]map[string]any, 0, len(order))
	for _, key := range order {
		bucket := buckets[key]
		net := bucket.Inflows - bucket.Outflows - staticOpex
		monthly = append(monthly, map[string]any{
			"month":                       key,
			"payments_from_customers_bhd": round3(bucket.Inflows),
			"payments_to_suppliers_bhd":   round3(bucket.Outflows),
			"operations_cost_bhd":         round3(staticOpex),
			"net_cashflow_bhd":            round3(net),
		})
	}

	result["status"] = "available"
	result["start_date"] = projection.StartDate.Format("2006-01-02")
	result["end_date"] = projection.EndDate.Format("2006-01-02")
	result["opening_cash_bhd"] = round3(projection.OpeningCash)
	result["projected_cash_bhd"] = round3(projection.ProjectedCash)
	result["monthly_projection"] = monthly
	result["static_operations_cost_bhd"] = staticOpex
	result["basis"] = "Customer inflows from projected invoice collections, supplier outflows from due supplier invoices, plus static operating cost"
	return result
}

func parseYearWindowFromQuery(query string) (time.Time, time.Time, int, string, bool) {
	return butlercontext.ParseYearWindowFromQuery(query)
}

func parseQuarterWindowFromQuery(query string) (time.Time, time.Time, string, bool) {
	return butlercontext.ParseQuarterWindowFromQuery(query)
}
