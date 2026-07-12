// Package intent contains Butler's pure intent routing helpers.
package intent

import (
	"strings"
	"time"

	butler "ph_holdings_app/pkg/butler"
)

type ARProjectionScope struct {
	Start          time.Time
	End            time.Time
	Label          string
	OrderStart     time.Time
	IncludeOrders  bool
	IncludeOffers  bool
	InvoiceOnly    bool
	NeedsClarify   bool
	IntentDetected bool
}

func TryClarificationFastPath(intent butler.Intent, message string, hasFinanceAccess bool) (string, []butler.ButlerAction, bool) {
	q := NormalizeRouterText(message)
	if q == "" {
		return "", nil, false
	}

	if IsBroadCapabilitiesQuestion(q) && !IsCapabilitySelectionPrompt(q) {
		msg := "You are asking what Butler can do. I can answer that as a live ERP capability route instead of a long generic paragraph. Pick the view you want."
		return msg, []butler.ButlerAction{
			PromptAction("All capabilities", "Show me all Butler capabilities from live ERP data.", "capability_explainer", "all"),
			PromptAction("Sales and OCR", "Show me Butler capabilities for sales pipeline, OCR ingestion, opportunity creation, offer PDFs, and follow-ups.", "capability_explainer", "sales_ocr"),
			PromptAction("Finance and AR", "Show me Butler capabilities for finance, AR projections, orders-to-invoice exposure, and cash risk.", "capability_explainer", "finance_ar"),
			PromptAction("Tasks and briefings", "Show me task, follow-up, and daily briefing capabilities.", "capability_explainer", "tasks"),
			PromptAction("Reports", "Show me report and document generation capabilities.", "capability_explainer", "reports"),
		}, true
	}

	scope := ParseARProjectionScope(message)
	if scope.IntentDetected && scope.NeedsClarify {
		if !hasFinanceAccess {
			return "AR projections require finance:view permission. I can still help with non-financial order or follow-up context.", []butler.ButlerAction{}, true
		}
		msg := "You are asking for an AR projection. I can run it a few different ways, and the choice matters because issued invoices, confirmed uninvoiced orders, and weighted offers are different financial buckets."
		return msg, []butler.ButlerAction{
			PromptAction("Invoices only", "Show AR projection for next month using issued invoices only.", "ar_projection", "invoices_only"),
			PromptAction("Include confirmed orders", "Show AR projection for next month including issued invoices and confirmed uninvoiced orders.", "ar_projection", "invoices_confirmed_orders"),
			PromptAction("Add weighted offers", "Show AR projection for next month including issued invoices, confirmed orders, and weighted active offers.", "ar_projection", "weighted_pipeline"),
			PromptAction("Manager brief", "Create a manager AR brief for the next two months with confirmed orders, offer pipeline, and collection risk.", "ar_projection", "manager_brief"),
		}, true
	}

	if ShouldAskClarifyingQuestion(intent, q) {
		msg := "I need one more bit of direction so I do not guess. Pick the closest route and I will continue from there."
		return msg, BuildClarificationActions(hasFinanceAccess), true
	}

	return "", nil, false
}

func ShouldAskClarifyingQuestion(intent butler.Intent, q string) bool {
	q = strings.TrimSpace(q)
	if q == "" || IsCapabilitySelectionPrompt(q) || IsBroadCapabilitiesQuestion(q) {
		return false
	}

	words := strings.Fields(q)
	if len(words) == 0 {
		return false
	}

	exactVaguePrompts := map[string]bool{
		"help":                true,
		"please help":         true,
		"do it":               true,
		"do this":             true,
		"do that":             true,
		"do the needful":      true,
		"handle this":         true,
		"make it":             true,
		"make this":           true,
		"what about this":     true,
		"tell me about it":    true,
		"continue":            true,
		"proceed":             true,
		"run it":              true,
		"check it":            true,
		"save it":             true,
		"fix it":              true,
		"same as above":       true,
		"same thing":          true,
		"that one":            true,
		"this one":            true,
		"what should i do":    true,
		"what should we do":   true,
		"what is going on":    true,
		"what is happening":   true,
		"what's happening":    true,
		"can you do this":     true,
		"can you handle this": true,
	}
	if exactVaguePrompts[q] {
		return true
	}

	if HasReferenceToken(q) {
		return false
	}

	knownBusinessSignals := []string{
		"customer", "supplier", "offer", "quote", "quotation", "rfq", "order", "item", "model",
		"serial", "delivery", "grn", "purchase", "po", "invoice", "payment", "cash", "revenue",
		"profit", "margin", "task", "email", "follow up", "follow-up", "service", "calibration",
	}
	hasKnownSignal := false
	for _, signal := range knownBusinessSignals {
		if strings.Contains(q, signal) {
			hasKnownSignal = true
			break
		}
	}

	vagueSignals := []string{" this", " that", " it", " thing", " needful", " same", " above", " those", " these"}
	hasVagueSignal := false
	padded := " " + q + " "
	for _, signal := range vagueSignals {
		if strings.Contains(padded, signal+" ") || strings.Contains(padded, signal) {
			hasVagueSignal = true
			break
		}
	}

	if hasVagueSignal && len(words) <= 10 && !hasKnownSignal {
		return true
	}
	if intent.Confidence < 0.35 && len(words) <= 8 && !hasKnownSignal {
		return true
	}

	return false
}

func HasReferenceToken(q string) bool {
	for _, word := range strings.Fields(q) {
		cleaned := strings.Trim(word, ".,:;()[]{}")
		if len(cleaned) < 4 {
			continue
		}
		hasDigit := false
		for _, r := range cleaned {
			if r >= '0' && r <= '9' {
				hasDigit = true
				break
			}
		}
		if hasDigit {
			return true
		}
	}
	return false
}

func BuildClarificationActions(hasFinanceAccess bool) []butler.ButlerAction {
	actions := []butler.ButlerAction{
		PromptAction("Find offer/order", "Find an offer or order by number and show status, customer, line items, and next action without financial totals.", "safe_sales_lookup", "offer_order"),
		PromptAction("Draft customer email", "Draft a customer email for an offer, delivery, follow-up, or clarification request. Ask me for the customer and reference number if missing.", "safe_sales_lookup", "email"),
		PromptAction("Search item/model", "Search items or products by model, long code, serial number, or offer/order number.", "safe_sales_lookup", "item_search"),
		PromptAction("Create follow-up", "Create a follow-up task. Ask me for customer, reference, owner, and due date if missing.", "safe_sales_lookup", "follow_up"),
	}
	if hasFinanceAccess {
		actions = append(actions, PromptAction("Manager finance view", "Show the manager finance view with revenue, margin, AR/AP, cash, and pipeline-value context for the current question.", "manager_finance_lookup", "finance"))
	} else {
		actions = append(actions, PromptAction("Sales-safe pipeline", "Show sales pipeline by RFQ, offer, order count, status, owner, and next action only. Do not include revenue, profit, margin, cash, AR, AP, or payment amounts.", "safe_sales_lookup", "pipeline_status"))
	}
	return actions
}

func PromptAction(label, prompt, intentID, optionID string) butler.ButlerAction {
	return butler.ButlerAction{
		Type:   "clarify",
		Target: "butler_prompt",
		Label:  label,
		Data: map[string]any{
			"prompt":     prompt,
			"intent_id":  intentID,
			"option_id":  optionID,
			"command_ui": true,
		},
	}
}

func ParseARProjectionScope(message string) ARProjectionScope {
	q := NormalizeRouterText(message)
	now := time.Now()
	start, end, label := NextCalendarMonthWindow(now)
	if strings.Contains(q, "next two months") || strings.Contains(q, "next 2 months") || strings.Contains(q, "next sixty days") || strings.Contains(q, "next 60 days") {
		start = BeginningOfDay(now)
		end = start.AddDate(0, 0, 60)
		label = "next 60 days"
	} else if strings.Contains(q, "next thirty days") || strings.Contains(q, "next 30 days") {
		start = BeginningOfDay(now)
		end = start.AddDate(0, 0, 30)
		label = "next 30 days"
	}

	scope := ARProjectionScope{
		Start:      start,
		End:        end,
		Label:      label,
		OrderStart: time.Date(end.Year(), time.January, 1, 0, 0, 0, 0, time.Local),
	}

	hasAR := ContainsWord(q, "ar") ||
		strings.Contains(q, "accounts receivable") ||
		strings.Contains(q, "receivable") ||
		strings.Contains(q, "receivables") ||
		strings.Contains(q, "collection") ||
		strings.Contains(q, "collections")
	hasProjection := strings.Contains(q, "projection") ||
		strings.Contains(q, "predict") ||
		strings.Contains(q, "forecast") ||
		strings.Contains(q, "next month") ||
		strings.Contains(q, "next 30") ||
		strings.Contains(q, "next 60") ||
		strings.Contains(q, "next two months") ||
		strings.Contains(q, "next 2 months")
	hasOrderChallenge := (strings.Contains(q, "order") || strings.Contains(q, "orders")) &&
		(strings.Contains(q, "receivable") || ContainsWord(q, "ar") || strings.Contains(q, "pipeline") || strings.Contains(q, "invoice"))

	scope.IntentDetected = (hasAR && hasProjection) || hasOrderChallenge
	if !scope.IntentDetected {
		return scope
	}

	scope.InvoiceOnly = strings.Contains(q, "invoice only") ||
		strings.Contains(q, "invoices only") ||
		strings.Contains(q, "issued invoices only") ||
		strings.Contains(q, "booked ar only")
	scope.IncludeOrders = strings.Contains(q, "confirmed order") ||
		strings.Contains(q, "confirmed uninvoiced") ||
		strings.Contains(q, "uninvoiced order") ||
		strings.Contains(q, "include orders") ||
		strings.Contains(q, "orders in the pipeline") ||
		strings.Contains(q, "order pipeline") ||
		hasOrderChallenge
	scope.IncludeOffers = strings.Contains(q, "weighted") ||
		strings.Contains(q, "active offers") ||
		strings.Contains(q, "offer pipeline") ||
		strings.Contains(q, "opportunity pipeline") ||
		strings.Contains(q, "opportunities")

	if scope.IncludeOffers {
		scope.IncludeOrders = true
	}

	scope.NeedsClarify = !scope.InvoiceOnly && !scope.IncludeOrders && !scope.IncludeOffers
	return scope
}

func PipelineStageWeight(stage string) float64 {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "won":
		return 0.85
	case "proposal":
		return 0.50
	case "quoted":
		return 0.40
	case "qualified":
		return 0.25
	case "rfq":
		return 0.15
	default:
		return 0.10
	}
}

func IsBroadCapabilitiesQuestion(q string) bool {
	signals := []string{
		"what can you do",
		"what all can you do",
		"all that you can do",
		"tell me all that you can do",
		"what you can do",
		"tell us what you can do",
		"tell me what you can do",
		"what are your capabilities",
		"understand your capabilities",
		"your capabilities",
		"what can butler do",
		"how can you help",
		"what can u do",
	}
	for _, signal := range signals {
		if strings.Contains(q, signal) {
			return true
		}
	}
	return false
}

func IsCapabilitySelectionPrompt(q string) bool {
	selections := []string{
		"show me all butler capabilities",
		"show all butler capabilities",
		"all butler capabilities",
		"capabilities from live erp data",
		"capabilities for sales pipeline",
		"capabilities for finance",
		"ocr ingestion",
		"task follow up and daily briefing capabilities",
		"report and document generation capabilities",
	}
	for _, selection := range selections {
		if strings.Contains(q, selection) {
			return true
		}
	}
	return false
}

func NormalizeRouterText(message string) string {
	q := strings.ToLower(strings.TrimSpace(message))
	replacer := strings.NewReplacer(
		"â€™", "'",
		"â€œ", "\"",
		"â€", "\"",
		"\n", " ",
		"\t", " ",
	)
	q = replacer.Replace(q)
	for strings.Contains(q, "  ") {
		q = strings.ReplaceAll(q, "  ", " ")
	}
	return q
}

func ContainsWord(q, word string) bool {
	normalized := strings.ToLower(q)
	replacer := strings.NewReplacer(
		".", " ",
		",", " ",
		"?", " ",
		"!", " ",
		";", " ",
		":", " ",
		"/", " ",
		"\\", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"-", " ",
		"_", " ",
	)
	normalized = " " + replacer.Replace(normalized) + " "
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}
	return strings.Contains(normalized, " "+strings.ToLower(strings.TrimSpace(word))+" ")
}

func NextCalendarMonthWindow(now time.Time) (time.Time, time.Time, string) {
	location := now.Location()
	start := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, location)
	end := start.AddDate(0, 1, 0)
	return start, end, start.Format("January 2006")
}

func BeginningOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
