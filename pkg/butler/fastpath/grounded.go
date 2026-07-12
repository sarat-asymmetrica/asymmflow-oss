// Package fastpath contains grounded Butler responses that use local ERP data
// before falling back to an external model.
package fastpath

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	butler "ph_holdings_app/pkg/butler"
	butlerintent "ph_holdings_app/pkg/butler/intent"
)

type Service struct {
	DB       butler.DatabasePort
	Workflow butler.WorkflowPort
	AppCtx   butler.ButlerAppContext
	Now      func() time.Time
}

type projectionStageRow struct {
	Stage    string
	Count    int64
	TotalBHD float64
}

func (s Service) TryCapabilities(message string, hasFinanceAccess bool) (string, bool) {
	if s.DB == nil {
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

	customerCount := s.countRaw("SELECT COUNT(*) AS count FROM customers WHERE deleted_at IS NULL")
	supplierCount := s.countRaw("SELECT COUNT(*) AS count FROM suppliers WHERE deleted_at IS NULL")
	orderCount := s.countRaw("SELECT COUNT(*) AS count FROM orders WHERE deleted_at IS NULL AND status NOT IN ('Cancelled', 'Canceled', 'Void')")
	invoiceCount := s.countRaw("SELECT COUNT(*) AS count FROM invoices WHERE deleted_at IS NULL AND status NOT IN ('Cancelled', 'Void', 'Proforma')")
	offerCount := s.countRaw("SELECT COUNT(*) AS count FROM offers WHERE deleted_at IS NULL AND stage NOT IN ('Lost', 'Expired')")
	opportunityCount := s.countRaw("SELECT COUNT(*) AS count FROM opportunities WHERE deleted_at IS NULL")
	taskCount := s.countRaw("SELECT COUNT(*) AS count FROM task_items WHERE deleted_at IS NULL AND status NOT IN ('completed', 'archived')")
	productCount := s.countRaw("SELECT COUNT(*) AS count FROM products WHERE deleted_at IS NULL AND is_active = ?", true)

	currentYear := s.now().Year()
	yearStart := time.Date(currentYear, time.January, 1, 0, 0, 0, 0, time.Local)
	yearEnd := yearStart.AddDate(1, 0, 0)
	currentYearOrders := s.scalarFloat(`SELECT COALESCE(SUM(CASE WHEN grand_total_bhd > 0 THEN grand_total_bhd ELSE total_value_bhd END), 0) AS total
		FROM orders
		WHERE deleted_at IS NULL AND order_date >= ? AND order_date < ? AND status NOT IN ('Cancelled', 'Canceled', 'Void')`, yearStart, yearEnd)
	currentYearOffers := s.scalarFloat(`SELECT COALESCE(SUM(total_value_bhd), 0) AS total
		FROM offers
		WHERE deleted_at IS NULL AND quotation_date >= ? AND quotation_date < ? AND stage IN ('RFQ', 'Quoted', 'Won')`, yearStart, yearEnd)

	financeLine := "Financial data is permission-gated. I can discuss operational context unless your role has finance:view."
	if hasFinanceAccess {
		financeLine = fmt.Sprintf("For FY%d I can see confirmed orders of %s BHD and offer coverage of %s BHD.", currentYear, formatBHD(currentYearOrders), formatBHD(currentYearOffers))
	}

	tableCount := s.countRaw("SELECT COUNT(*) AS count FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'")
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

func (s Service) TryARProjection(message string, hasFinanceAccess bool) (string, []butler.ButlerAction, bool) {
	if s.DB == nil {
		return "", nil, false
	}

	scope := butlerintent.ParseARProjectionScope(message)
	if !scope.IntentDetected || scope.NeedsClarify {
		return "", nil, false
	}
	if !hasFinanceAccess {
		return "AR projections require finance:view permission. I can still help with non-financial order or follow-up context.", []butler.ButlerAction{}, true
	}

	now := s.now()
	currentOpenAR := s.scalarFloat(`SELECT COALESCE(SUM(outstanding_bhd), 0) AS total
		FROM invoices
		WHERE deleted_at IS NULL AND status IN ('Sent', 'PartiallyPaid', 'Overdue') AND outstanding_bhd > 0`)
	overdueAR := s.scalarFloat(`SELECT COALESCE(SUM(outstanding_bhd), 0) AS total
		FROM invoices
		WHERE deleted_at IS NULL AND status IN ('Sent', 'PartiallyPaid', 'Overdue') AND outstanding_bhd > 0 AND due_date < ?`, now)
	invoiceDueInWindow := s.scalarFloat(`SELECT COALESCE(SUM(outstanding_bhd), 0) AS total
		FROM invoices
		WHERE deleted_at IS NULL AND status IN ('Sent', 'PartiallyPaid', 'Overdue') AND outstanding_bhd > 0 AND due_date >= ? AND due_date < ?`, scope.Start, scope.End)
	invoiceDueByWindowEnd := s.scalarFloat(`SELECT COALESCE(SUM(outstanding_bhd), 0) AS total
		FROM invoices
		WHERE deleted_at IS NULL AND status IN ('Sent', 'PartiallyPaid', 'Overdue') AND outstanding_bhd > 0 AND due_date < ?`, scope.End)

	uninvoicedOrderExposure := 0.0
	pendingOrderCount := 0
	orderErrText := ""
	if scope.IncludeOrders {
		var err error
		if s.Workflow != nil {
			uninvoicedOrderExposure, pendingOrderCount, err = s.Workflow.CalculateUninvoicedOrderExposure(scope.OrderStart, scope.End)
		} else {
			err = fmt.Errorf("workflow port not configured")
		}
		if err != nil {
			orderErrText = fmt.Sprintf(" Order exposure could not be calculated: %v", err)
		}
	}

	weightedPipeline := 0.0
	pipelineLines := []string{}
	if scope.IncludeOffers {
		weightedPipeline, pipelineLines = s.calculateWeightedPipeline(scope.OrderStart, scope.End)
	}

	totalAttention := invoiceDueByWindowEnd
	if scope.IncludeOrders {
		totalAttention += uninvoicedOrderExposure
	}
	if scope.IncludeOffers {
		totalAttention += weightedPipeline
	}

	orderLine := "- Confirmed uninvoiced orders were not included in this view."
	if scope.IncludeOrders {
		orderLine = fmt.Sprintf("- Confirmed uninvoiced orders from %s through %s: %d orders, %s BHD expected new AR once invoiced.%s",
			scope.OrderStart.Format("2 Jan 2006"),
			scope.End.Add(-time.Nanosecond).Format("2 Jan 2006"),
			pendingOrderCount,
			formatBHD(uninvoicedOrderExposure),
			orderErrText,
		)
	}

	pipelineText := "- Weighted active offer/opportunity pipeline was not included in this view."
	if scope.IncludeOffers {
		if len(pipelineLines) == 0 {
			pipelineText = "- No weighted active offer/opportunity pipeline was found for this horizon."
		} else {
			pipelineText = strings.Join(pipelineLines, "\n")
		}
	}

	mode := "invoice-only"
	if scope.IncludeOrders && scope.IncludeOffers {
		mode = "issued invoices + confirmed orders + weighted active offers"
	} else if scope.IncludeOrders {
		mode = "issued invoices + confirmed orders"
	}

	response := fmt.Sprintf(`AR projection - %s

Basis
- Mode: %s.
- Window: %s to %s.
- Order exposure uses the fresh-start operating year from %s so confirmed 2026 orders are not ignored.

Issued invoice AR
- Current open invoice AR: %s BHD.
- Already overdue AR: %s BHD.
- Invoice AR due inside this window: %s BHD.
- Booked invoice AR due by window end, including overdue balances: %s BHD.

Confirmed order receivable path
%s

Weighted pipeline path
%s

Projection
- Receivable attention for this basis: %s BHD.
- This is an AR creation and collection focus number, not guaranteed cash collection.
- Confirmed orders should create receivables only after invoicing; cash timing then depends on payment terms and customer behavior.

Why this route exists
- Butler is using local SQL before the external model here.
- The earlier zero-collection style answer was mixing cash collection prediction with AR creation and was not enough for management decisions. Orders are now explicitly counted when you choose the confirmed-order basis.`, scope.Label, mode, scope.Start.Format("2 Jan 2006"), scope.End.Add(-time.Nanosecond).Format("2 Jan 2006"), scope.OrderStart.Format("2 Jan 2006"), formatBHD(currentOpenAR), formatBHD(overdueAR), formatBHD(invoiceDueInWindow), formatBHD(invoiceDueByWindowEnd), orderLine, pipelineText, formatBHD(totalAttention))

	actions := []butler.ButlerAction{
		butlerintent.PromptAction("Invoice-only view", "Show AR projection for next month using issued invoices only.", "ar_projection", "invoices_only"),
		butlerintent.PromptAction("Include orders", "Show AR projection for next month including issued invoices and confirmed uninvoiced orders.", "ar_projection", "invoices_confirmed_orders"),
		butlerintent.PromptAction("Add weighted offers", "Show AR projection for next month including issued invoices, confirmed orders, and weighted active offers.", "ar_projection", "weighted_pipeline"),
		butlerintent.PromptAction("Manager brief", "Create a manager AR brief for the next two months with confirmed orders, offer pipeline, and collection risk.", "ar_projection", "manager_brief"),
	}

	return response, actions, true
}

func (s Service) TryTaskCreation(intent butler.Intent, message string) (string, bool) {
	if s.AppCtx == nil {
		return "", false
	}

	assigneeRef, taskDetails, matched := ParseTaskCreationRequest(intent, message)
	if !matched {
		return "", false
	}
	if strings.TrimSpace(assigneeRef) == "" || strings.TrimSpace(taskDetails) == "" {
		return "I can create that task, but I need both the assignee and the task details.", true
	}

	createdTask, err := s.AppCtx.CreateTask(taskDetails, assigneeRef)
	if err != nil {
		return fmt.Sprintf("I understood that as a task request for %s, but I could not create it: %v", assigneeRef, err), true
	}

	title := fieldString(createdTask, "Title", "title")
	if title == "" {
		title = taskDetails
	}
	return fmt.Sprintf("Created the task %q for %s. The assignment notification was queued as well.", title, assigneeRef), true
}

func (s Service) TryOfferDraft(message, customerHint, productHint string) (string, []butler.ButlerAction, bool) {
	if s.AppCtx == nil {
		return "", nil, false
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
		productQuery := q
		if strings.TrimSpace(customerHint) != "" {
			custLower := strings.ToLower(strings.TrimSpace(customerHint))
			for _, prep := range []string{"for ", "to "} {
				productQuery = strings.Replace(productQuery, prep+custLower, "", 1)
			}
		}
		productHint = InferProductHintFromText(productQuery)
	}
	if strings.TrimSpace(customerHint) == "" || strings.TrimSpace(productHint) == "" {
		return "", nil, false
	}

	qty, okQty := ExtractNumericToken(q, `(?i)(?:quantity|qty)\s*(?:is|=|:|would\s+be|to\s+be)?\s*([0-9]+(?:\.[0-9]+)?)`)
	price, okPrice := ExtractNumericToken(q, `(?i)(?:price(?:\s+per\s+unit)?|unit price)\s*(?:is|=|at|:|would\s+be|to\s+be)?\s*([0-9]+(?:\.[0-9]+)?)`)
	if !okPrice {
		price, okPrice = ExtractNumericToken(q, `(?i)([0-9]+(?:\.[0-9]+)?)\s*bhd`)
	}
	if !okQty || !okPrice || qty <= 0 || price <= 0 {
		return "", nil, false
	}

	customer, err := s.AppCtx.GetCustomerByName(customerHint)
	if err != nil {
		return "", nil, false
	}
	customerID := firstNonEmpty(fieldString(customer, "CustomerID", "customer_id"), fieldString(customer, "ID", "id"))
	displayName := firstNonEmpty(fieldString(customer, "BusinessName", "business_name"), customerHint)
	if customerID == "" {
		return "", nil, false
	}

	total := qty * price
	action := butler.ButlerAction{
		Type:   "create_offer_draft",
		Target: "quotation",
		Label:  "Create offer draft",
		Data: map[string]any{
			"customer_id":   customerID,
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
	return msg, []butler.ButlerAction{action}, true
}

func ParseTaskCreationRequest(intent butler.Intent, message string) (string, string, bool) {
	original := strings.TrimSpace(message)
	if original == "" {
		return "", "", false
	}

	lower := strings.ToLower(original)
	createSignals := []string{
		"create a task",
		"create task",
		"add a task",
		"add task",
		"make a task",
		"make task",
		"assign a task",
		"assign task",
	}
	matchedSignal := false
	for _, signal := range createSignals {
		if strings.Contains(lower, signal) {
			matchedSignal = true
			break
		}
	}
	if !matchedSignal {
		return "", "", false
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:can you|could you|please|buddy)?\s*(?:create|add|make)\s+(?:a\s+)?task\s+for\s+([A-Za-z][A-Za-z.\-\s]{0,40}?)\s+(?:to\s+)?(.+)`),
		regexp.MustCompile(`(?i)(?:can you|could you|please|buddy)?\s*assign\s+(?:a\s+)?task\s+to\s+([A-Za-z][A-Za-z.\-\s]{0,40}?)\s+(?:to\s+)?(.+)`),
	}
	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(original); len(matches) == 3 {
			return strings.TrimSpace(matches[1]), SanitizeTaskCreationDetails(matches[2]), true
		}
	}

	employeeRef := strings.TrimSpace(intent.PersonName)
	if employeeRef == "" {
		return "", "", true
	}

	employeePattern := regexp.MustCompile(`(?i)(?:for|to)\s+` + regexp.QuoteMeta(employeeRef) + `\s+(.*)$`)
	if matches := employeePattern.FindStringSubmatch(original); len(matches) == 2 {
		return employeeRef, SanitizeTaskCreationDetails(matches[1]), true
	}

	return employeeRef, "", true
}

func SanitizeTaskCreationDetails(details string) string {
	cleaned := strings.TrimSpace(details)
	if cleaned == "" {
		return ""
	}

	trailingMarkers := []string{
		"make sure",
		"ensure",
		"notify",
		"notification",
		"notifications",
	}
	lower := strings.ToLower(cleaned)
	for _, marker := range trailingMarkers {
		if idx := strings.Index(lower, marker); idx >= 0 {
			cleaned = strings.TrimSpace(cleaned[:idx])
			lower = strings.ToLower(cleaned)
		}
	}

	// Cut at the first genuine sentence boundary — terminal punctuation followed by
	// whitespace and a capital letter (or end of string). This avoids truncating on
	// abbreviations embedded in entity names such as "Co.", "Ltd." or "W.L.L.".
	if loc := regexp.MustCompile(`[.!?]\s+[A-Z]`).FindStringIndex(cleaned); loc != nil {
		cleaned = strings.TrimSpace(cleaned[:loc[0]])
	}

	cleaned = strings.TrimRight(cleaned, " .!?")
	cleaned = strings.TrimSpace(strings.Trim(cleaned, " ,;:-"))
	if cleaned == "" {
		return ""
	}

	runes := []rune(cleaned)
	if len(runes) > 0 && runes[0] >= 'a' && runes[0] <= 'z' {
		runes[0] = runes[0] - 32
	}
	return string(runes)
}

func InferProductHintFromText(q string) string {
	re := regexp.MustCompile(`(?i)\bfor\s+(.+?)(?:\.\s*|,\s*|;\s*|\s+quantity|\s+qty|\s+price|\s+bhd|$)`)
	m := re.FindStringSubmatch(strings.TrimSpace(q))
	if len(m) != 2 {
		return ""
	}
	return strings.TrimSpace(strings.Trim(m[1], "."))
}

func ExtractNumericToken(text, pattern string) (float64, bool) {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0, false
	}
	var out float64
	_, err := fmt.Sscanf(m[1], "%f", &out)
	if err != nil {
		return 0, false
	}
	return out, true
}

func (s Service) calculateWeightedPipeline(start, end time.Time) (float64, []string) {
	rows := []projectionStageRow{}
	rows = append(rows, s.stageRows(`SELECT COALESCE(stage, 'Unstaged') AS stage, COUNT(*) AS count, COALESCE(SUM(total_value_bhd), 0) AS total_bhd
		FROM offers
		WHERE deleted_at IS NULL
			AND quotation_date >= ?
			AND quotation_date < ?
			AND stage IN ('RFQ', 'Qualified', 'Proposal', 'Quoted', 'Won')
		GROUP BY COALESCE(stage, 'Unstaged')
		ORDER BY total_bhd DESC`, start, end)...)
	rows = append(rows, s.stageRows(`SELECT COALESCE(stage, 'Unstaged') AS stage, COUNT(*) AS count, COALESCE(SUM(revenue_bhd), 0) AS total_bhd
		FROM opportunities
		WHERE deleted_at IS NULL
			AND (year = ? OR (offer_date >= ? AND offer_date < ?))
			AND stage IN ('RFQ', 'Qualified', 'Proposal', 'Quoted', 'Won')
		GROUP BY COALESCE(stage, 'Unstaged')
		ORDER BY total_bhd DESC`, end.Year(), start, end)...)

	if len(rows) == 0 {
		return 0, []string{}
	}

	weighted := 0.0
	lines := []string{"- Weighted active pipeline, separated from confirmed orders:"}
	for _, row := range rows {
		if row.Count == 0 || row.TotalBHD <= 0 {
			continue
		}
		weight := butlerintent.PipelineStageWeight(row.Stage)
		weightedValue := row.TotalBHD * weight
		weighted += weightedValue
		lines = append(lines, fmt.Sprintf("  - %s: %d records, %s BHD at %.0f%% = %s BHD",
			firstNonEmpty(strings.TrimSpace(row.Stage), "Unstaged"),
			row.Count,
			formatBHD(row.TotalBHD),
			weight*100,
			formatBHD(weightedValue),
		))
	}

	if weighted <= 0 {
		return 0, []string{}
	}
	lines = append(lines, fmt.Sprintf("- Weighted pipeline contribution: %s BHD.", formatBHD(weighted)))
	return round3(weighted), lines
}

func (s Service) stageRows(sql string, args ...any) []projectionStageRow {
	rows, err := s.DB.RawQuery(sql, args...)
	if err != nil {
		return nil
	}
	result := make([]projectionStageRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, projectionStageRow{
			Stage:    mapString(row, "stage"),
			Count:    int64(mapFloat(row, "count")),
			TotalBHD: mapFloat(row, "total_bhd"),
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].TotalBHD > result[j].TotalBHD
	})
	return result
}

func (s Service) countRaw(sql string, args ...any) int64 {
	rows, err := s.DB.RawQuery(sql, args...)
	if err != nil || len(rows) == 0 {
		return 0
	}
	for _, value := range rows[0] {
		return int64(toFloat(value))
	}
	return 0
}

func (s Service) scalarFloat(sql string, args ...any) float64 {
	rows, err := s.DB.RawQuery(sql, args...)
	if err != nil || len(rows) == 0 {
		return 0
	}
	for _, value := range rows[0] {
		return toFloat(value)
	}
	return 0
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func mapString(row map[string]any, key string) string {
	value, ok := lookup(row, key)
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}

func fieldString(value any, fieldNames ...string) string {
	if value == nil {
		return ""
	}
	if row, ok := value.(map[string]any); ok {
		for _, name := range fieldNames {
			if value, ok := lookup(row, name); ok {
				return strings.TrimSpace(fmt.Sprint(value))
			}
		}
		return ""
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}
	for _, name := range fieldNames {
		field := rv.FieldByName(name)
		if field.IsValid() && field.CanInterface() {
			return strings.TrimSpace(fmt.Sprint(field.Interface()))
		}
	}
	return ""
}

func mapFloat(row map[string]any, key string) float64 {
	value, ok := lookup(row, key)
	if !ok {
		return 0
	}
	return toFloat(value)
}

func lookup(row map[string]any, key string) (any, bool) {
	if value, ok := row[key]; ok {
		return value, true
	}
	lower := strings.ToLower(key)
	for rowKey, value := range row {
		if strings.ToLower(rowKey) == lower {
			return value, true
		}
	}
	return nil, false
}

func toFloat(value any) float64 {
	switch v := value.(type) {
	case nil:
		return 0
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case []byte:
		parsed, _ := strconv.ParseFloat(string(v), 64)
		return parsed
	case string:
		parsed, _ := strconv.ParseFloat(v, 64)
		return parsed
	default:
		parsed, _ := strconv.ParseFloat(fmt.Sprint(v), 64)
		return parsed
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func formatBHD(value float64) string {
	return fmt.Sprintf("%.3f", value)
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}
