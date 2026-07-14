// ═══════════════════════════════════════════════════════════════════════════
// BUTLER AI CHAT - MISTRAL INTELLIGENCE PIPELINE
//
// MISSION: AI-powered business intelligence agent with contextual data injection
//
// ARCHITECTURE:
//   1. Intent Classification (keyword + entity extraction)
//   2. Context Builder (GORM queries based on intent domain)
//   3. Three-Regime Awareness (adjusts LLM guidance)
//   4. Mistral API Client (small for simple, large for complex)
//   5. Action Parser ([ACTIONS]...[/ACTIONS] JSON blocks)
//   6. Response Formatter (ButlerResponse for frontend)
//
// MODELS:
//   - mistral-small-latest: Simple queries (customer lookup, basic stats)
//   - mistral-large-latest: Complex multi-entity analysis, recommendations
//
// Built with MATHEMATICAL RIGOR × AI AUGMENTATION × BUSINESS INTELLIGENCE
// Day 195+ - Mistral Intelligence Pipeline
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	butlerdomain "ph_holdings_app/pkg/butler"
	butlerchat "ph_holdings_app/pkg/butler/chat"
)

// ============================================================================
// DATA STRUCTURES
// ============================================================================

type ButlerResponse = butlerdomain.ButlerResponse
type ButlerResponseMetadata = butlerdomain.ButlerResponseMetadata
type ButlerAction = butlerdomain.ButlerAction
type Intent = butlerdomain.Intent
type ButlerResolvedEntity = butlerdomain.ButlerResolvedEntity

// ============================================================================
// MISTRAL API CONSTANTS
// ============================================================================

const (
	mistralAPIURL      = "https://api.mistral.ai/v1/chat/completions"
	mistralModelSmall  = "mistral-small-latest"
	mistralModelLarge  = "mistral-large-latest"
	mistralMaxTokens   = 8192
	mistralTemperature = 0.4

	// AIML API (Grok) constants — primary LLM backend for chat queries
	// AIML API is OpenAI-compatible; Grok offers ~1M token context window
	// Model ID: verify exact name in your AIML API dashboard (https://api.aimlapi.com)
	aimlAPIURL      = "https://api.aimlapi.com/v1/chat/completions"
	aimlModelID     = "x-ai/grok-4-fast-reasoning" // xAI Grok 4 via AIML API (prefixed model IDs)
	aimlMaxTokens   = 16384
	aimlTemperature = 0.4

	// Intent classification constants
	MaxKeywordMatchScore        = 4.0
	ComplexQueryDomainThreshold = 3
	ComplexQueryLengthThreshold = 200
	MistralAPITimeout           = 120 * time.Second
	AimlAPITimeout              = 120 * time.Second
	butlerMaxContextChars       = 60000
	butlerMaxReportContextChars = 80000
)

var fallbackAIMLModels = []string{
	"x-ai/grok-4-1-fast-reasoning",
	"x-ai/grok-3-beta",
	"gpt-4o",
	"gpt-4o-mini",
}

// ============================================================================
// INTENT CLASSIFICATION
// ============================================================================

var (
	customerKeywords   = []string{"customer", "client", "buyer", "account", "national petroleum", "gulf smelting", "north grid", "delta petro", "national oil", "horizon petroleum", "coastal gas", "business with", "total business", "how much", "who is our", "top customer", "biggest customer", "best customer", "worst customer", "most profitable", "profitab", "dormant", "inactive", "haven't ordered", "not ordered", "lapsed", "churn", "retention", "new customer", "acquisition", "which customer"}
	supplierKeywords   = []string{"supplier", "vendor", "source", "rhine", "oxan", "helvetia", "bought from", "purchased from", "procurement", "supply chain", "top supplier", "lead time", "delivery time", "reliable", "on time", "supplier performance", "qc rate", "quality"}
	financialKeywords  = []string{"cash", "payment", "invoice", "receivable", "payable", "balance", "revenue", "profit", "margin", "ar", "aging", "dso", "outstanding", "collection", "money", "owe", "owes", "paid", "unpaid", "earned", "income", "expense", "cashflow", "cash flow", "how much did we", "total sales", "turnover", "gross", "net", "bhd", "fiscal", "financial year", "days to pay", "payment speed", "slow payer", "fast payer", "days sales outstanding", "margin by", "profit by"}
	operationsKeywords = []string{"quote", "rfq", "order", "pipeline", "delivery", "purchase", "costing", "offer", "grn", "shipment", "po", "purchase order", "win rate", "conversion", "open orders", "pending", "active orders", "quotation", "expir", "validity", "expire", "lapsing", "about to expire", "win loss", "lost reason", "competition", "competitor", "abb", "won", "lost deals", "service", "maintenance", "calibration", "install", "installation", "commission", "commissioning", "repair", "visit"}
	workKeywords       = []string{"task", "tasks", "assigned", "assignee", "employee", "employees", "staff", "notification", "notifications", "workload", "work queue", "my work", "team board", "commented", "comment", "blocked task", "blocked reason", "collaborative", "project task"}
	riskKeywords       = []string{"overdue", "late", "risk", "aging", "collection", "blocked", "dispute", "critical", "warning", "defaulter", "slow payer", "bad debt", "credit block", "action item", "what should", "what do i", "today", "urgent", "priorit", "focus", "stuck", "follow up", "pending approval", "awaiting", "service due", "warranty", "calibration due", "cashflow", "cash flow"}
	marketKeywords     = []string{"price", "market", "competitor", "trend", "industry", "steel", "instrumentation"}

	// Patterns that indicate complex queries needing mistral-large
	complexPatterns = []string{"compare", "analyze", "recommend", "strategy", "forecast", "predict", "trend", "correlat", "multiple", "relationship", "which", "most", "least", "highest", "lowest", "best", "worst", "over time", "by month", "by quarter", "by year", "profitab", "dormant", "churn", "performance"}

	// Action regex for parsing [ACTIONS]...[/ACTIONS] blocks
	actionBlockRegex = regexp.MustCompile(`(?s)\[ACTIONS\](.*?)\[/ACTIONS\]`)
)

// classifyIntent determines query domain and extracts entities
func classifyIntent(query string) Intent {
	queryLower := strings.ToLower(query)

	intent := Intent{
		RawQuery:   query,
		Domain:     "general",
		Confidence: 0.5,
		Keywords:   []string{},
	}

	customerScore := countKeywordMatches(queryLower, customerKeywords)
	supplierScore := countKeywordMatches(queryLower, supplierKeywords)
	workScore := countKeywordMatches(queryLower, workKeywords)
	financialScore := countKeywordMatches(queryLower, financialKeywords)
	operationsScore := countKeywordMatches(queryLower, operationsKeywords)
	riskScore := countKeywordMatches(queryLower, riskKeywords)
	marketScore := countKeywordMatches(queryLower, marketKeywords)

	// Count keyword matches per domain using a stable priority order so ties are deterministic.
	domainScores := []struct {
		domain string
		score  int
	}{
		{"customer", customerScore},
		{"supplier", supplierScore},
		{"work", workScore},
		{"financial", financialScore},
		{"operations", operationsScore},
		{"risk", riskScore},
		{"market", marketScore},
	}

	// Find highest scoring domain with deterministic tie-breaking by list order above.
	maxScore := 0
	for _, candidate := range domainScores {
		if candidate.score > maxScore {
			maxScore = candidate.score
			intent.Domain = candidate.domain
		}
	}

	// Set confidence based on match count
	intent.Confidence = float64(maxScore) / MaxKeywordMatchScore
	if intent.Confidence > 1.0 {
		intent.Confidence = 1.0
	}
	if maxScore == 0 {
		intent.Confidence = 0.3
	}

	// Extract entity names even for mixed-domain asks so downstream grounding can still resolve them.
	intent.EntityName = extractEntityName(query)
	if intent.EntityName != "" {
		switch {
		case supplierScore > customerScore && supplierScore > 0:
			intent.ReferenceKind = "supplier"
		case customerScore > 0:
			intent.ReferenceKind = "customer"
		case intent.Domain == "supplier":
			intent.ReferenceKind = "supplier"
		case intent.Domain == "customer":
			intent.ReferenceKind = "customer"
		}
	}

	intent.PersonName = extractPersonReference(query)
	if intent.PersonName != "" {
		if looksLikeAccountManagerReference(queryLower) {
			intent.ReferenceKind = "account_manager"
		} else if intent.Domain == "work" {
			intent.ReferenceKind = "employee"
		} else if intent.ReferenceKind == "" {
			intent.ReferenceKind = "contact_person"
		}
	}

	// Market queries need scraper
	if intent.Domain == "market" {
		intent.NeedsScraper = true
	}

	// Check if query is complex (needs larger model)
	for _, pattern := range complexPatterns {
		if strings.Contains(queryLower, pattern) {
			intent.IsComplex = true
			break
		}
	}

	// Multi-domain or long queries are complex
	domainCount := 0
	for _, candidate := range domainScores {
		if candidate.score > 0 {
			domainCount++
		}
	}
	if domainCount >= ComplexQueryDomainThreshold || len(query) > ComplexQueryLengthThreshold {
		intent.IsComplex = true
	}

	return intent
}

// countKeywordMatches counts how many keywords appear in the query
func countKeywordMatches(query string, keywords []string) int {
	count := 0
	for _, kw := range keywords {
		if strings.Contains(query, kw) {
			count++
		}
	}
	return count
}

// extractEntityName extracts a customer or supplier name from the query using
// three strategies in priority order: possessive detection, trigger-word scan,
// and all-caps acronym detection (NPC, NGA, DPC, GSC, etc.).
func extractEntityName(query string) string {
	words := strings.Fields(query)

	// --- Strategy 1: possessive ("NGA's", "Gulf Smelting's", "NPC's") ---
	possessiveRe := regexp.MustCompile(`(?i)\b([A-Z][A-Za-z&().\-]{1,40}(?:\s+[A-Z][A-Za-z&().\-]{1,40}){0,4})'s?\b`)
	if m := possessiveRe.FindStringSubmatch(query); len(m) > 1 {
		candidate := strings.TrimSpace(m[1])
		if !isEntityStopWord(strings.ToLower(candidate)) {
			return candidate
		}
	}

	// --- Strategy 2: trigger-word scan (expanded from original) ---
	triggers := map[string]bool{
		"about": true, "is": true, "for": true, "with": true, "from": true,
		"customer": true, "supplier": true, "client": true,
		"how's": true, "hows": true, "show": true, "get": true, "find": true,
		"tell": true, "regarding": true, "on": true, "contact": true,
		"invoice": true, "invoices": true, "order": true, "orders": true,
		"payment": true, "payments": true, "outstanding": true, "balance": true,
		"status": true, "update": true, "info": true, "details": true,
	}
	for i, word := range words {
		wLower := strings.ToLower(strings.TrimRight(word, "?,.'"))
		if triggers[wLower] && i+1 < len(words) {
			name := []string{}
			for j := i + 1; j < len(words) && j < i+6; j++ {
				w := strings.TrimRight(words[j], "?,.'")
				if isEntityStopWord(strings.ToLower(w)) {
					break
				}
				name = append(name, w)
			}
			if len(name) > 0 {
				return strings.TrimRight(strings.Join(name, " "), "?.,")
			}
		}
	}

	// --- Strategy 3: all-caps acronyms (NPC, NGA, DPC, NOA, CJV) ---
	// Skip generic ERP acronyms (AR, AP, PO, GRN, DN, VAT, BHD, FX, etc.)
	erpAcronyms := map[string]bool{
		"AR": true, "AP": true, "PO": true, "GRN": true, "DN": true, "VAT": true,
		"BHD": true, "FX": true, "AI": true, "ERP": true, "PDF": true, "OCR": true,
		"CEO": true, "CFO": true, "COO": true, "HR": true, "IT": true,
		"Q1": true, "Q2": true, "Q3": true, "Q4": true, "YTD": true, "MTD": true,
	}
	for _, word := range words {
		clean := strings.TrimRight(word, "?,.'")
		if len(clean) >= 2 && clean == strings.ToUpper(clean) && !erpAcronyms[clean] {
			return clean
		}
	}

	return ""
}

// isEntityStopWord returns true for words that are unlikely to be part of an entity name
func isEntityStopWord(w string) bool {
	stops := map[string]bool{
		"is": true, "the": true, "doing": true, "has": true, "with": true,
		"last": true, "this": true, "next": true, "year": true, "month": true,
		"quarter": true, "week": true, "total": true, "business": true,
		"in": true, "at": true, "to": true, "and": true, "a": true, "an": true,
		"we": true, "have": true, "had": true, "been": true, "was": true,
		"how": true, "what": true, "much": true, "many": true, "give": true,
		"me": true, "our": true, "their": true, "since": true, "during": true,
		"all": true, "can": true, "you": true, "any": true, "some": true,
		"its": true, "my": true, "your": true, "show": true,
		"get": true, "find": true, "tell": true, "list": true, "latest": true,
		"recent": true, "current": true, "pending": true, "overdue": true,
		"paid": true, "unpaid": true, "active": true, "closed": true,
		"payment": true, "payments": true, "history": true, "invoice": true, "invoices": true,
		"order": true, "orders": true, "task": true, "tasks": true, "notification": true,
		"notifications": true, "quotation": true, "quote": true, "calibration": true,
		"service": true, "visit": true, "package": true,
	}
	return stops[w]
}

func extractPersonReference(query string) string {
	triggerPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:contact|attention|attn|spoc|person)\s+(?:for|at|of)?\s*([A-Z][A-Za-z.\-]+(?:\s+[A-Z][A-Za-z.\-]+){0,3})`),
		regexp.MustCompile(`(?i)(?:account manager|salesperson|sales person|owner|handled by|handling)\s+(?:for|is|:)?\s*([A-Z][A-Za-z.\-]+(?:\s+[A-Z][A-Za-z.\-]+){0,3})`),
		regexp.MustCompile(`(?i)(?:tasks?\s+assigned\s+to|assigned\s+to|workload\s+for|notifications?\s+for|tasks?\s+for|employee|staff member)\s+([A-Z][A-Za-z.\-]+(?:\s+[A-Z][A-Za-z.\-]+){0,2})`),
		regexp.MustCompile(`(?i)(?:notifications?\s+does|tasks?\s+does)\s+([A-Z][A-Za-z.\-]+(?:\s+[A-Z][A-Za-z.\-]+){0,2})\s+have`),
	}
	for _, pattern := range triggerPatterns {
		if m := pattern.FindStringSubmatch(query); len(m) > 1 {
			return cleanExtractedPersonReference(m[1])
		}
	}

	fullNamePattern := regexp.MustCompile(`\b([A-Z][A-Za-z.\-]+(?:\s+[A-Z][A-Za-z.\-]+)+)\b`)
	if m := fullNamePattern.FindStringSubmatch(query); len(m) > 1 {
		return cleanExtractedPersonReference(m[1])
	}

	return ""
}

func cleanExtractedPersonReference(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}

	trailingNoise := []string{
		" today", " right now", " now", " currently", " this week", " this month", " this quarter",
	}
	lower := strings.ToLower(candidate)
	for _, suffix := range trailingNoise {
		if strings.HasSuffix(lower, suffix) {
			candidate = strings.TrimSpace(candidate[:len(candidate)-len(suffix)])
			lower = strings.ToLower(candidate)
		}
	}

	return strings.TrimSpace(candidate)
}

func looksLikeAccountManagerReference(queryLower string) bool {
	accountManagerHints := []string{
		"account manager", "salesperson", "sales person", "owner",
		"who is handling", "who's handling", "handled by", "managed by",
	}
	for _, hint := range accountManagerHints {
		if strings.Contains(queryLower, hint) {
			return true
		}
	}
	return false
}

// ============================================================================
// MAIN CHAT FUNCTION
// ============================================================================

// ChatWithButler handles AI chat requests with business context injection
func (a *App) ChatWithButler(message string) (ButlerResponse, error) {
	if err := a.requirePermission("intelligence:chat"); err != nil {
		return ButlerResponse{}, err
	}
	// 1. Classify intent
	intent := classifyIntent(message)
	log.Printf("🧠 Intent classified: domain=%s entity=%q confidence=%.2f complex=%v",
		intent.Domain, intent.EntityName, intent.Confidence, intent.IsComplex)
	hasFinanceAccess := a.requirePermission("finance:view") == nil

	// 2. RBAC: Check if user has permission to access financial data
	if requiresFinancePrivilege(intent, message) {
		userRole := a.GetCurrentUserRole()
		licenseRole := a.GetLicenseRole()
		if !hasFinanceAccess {
			log.Printf("🔒 Butler financial query blocked: user role=%s license role=%s lacks finance:view", userRole, licenseRole)
			return ButlerResponse{
				Message:    "Financial data access requires manager or admin privileges.\n\nYou can still ask me about:\n• RFQ, offer, and order status by number\n• Item, model, serial, and product lookups\n• Customer emails and follow-ups\n• Sales pipeline status without revenue, profit, margin, cash, AR/AP, or payment figures\n\nFor financial reports and metrics, please contact your manager.",
				Actions:    []ButlerAction{},
				Confidence: 0.95,
				Context: map[string]any{
					"blocked_domain": intent.Domain,
					"user_role":      userRole,
					"license_role":   licenseRole,
				},
				Metadata: buildButlerMetadata(intent, map[string]any{
					"blocked_domain": intent.Domain,
					"user_role":      userRole,
					"license_role":   licenseRole,
				}, false, "blocked", "", "", "", nil),
			}, nil
		}
	}

	if routeReply, routeActions, handled := a.tryButlerIntentClarificationFastPath(intent, message, hasFinanceAccess); handled {
		return ButlerResponse{
			Message:    routeReply,
			Actions:    routeActions,
			Confidence: 0.99,
			Context:    map[string]any{"fast_path": "butler_intent_clarification"},
			Metadata:   buildButlerMetadata(intent, map[string]any{"fast_path": "butler_intent_clarification"}, hasFinanceAccess, "grounded_sql", "", "", "", nil),
		}, nil
	}
	if groundedReply, groundedActions, handled := a.tryGroundedARProjectionFastPath(intent, message, hasFinanceAccess); handled {
		return ButlerResponse{
			Message:    groundedReply,
			Actions:    groundedActions,
			Confidence: 0.99,
			Context:    map[string]any{"fast_path": "grounded_ar_projection"},
			Metadata:   buildButlerMetadata(intent, map[string]any{"fast_path": "grounded_ar_projection"}, hasFinanceAccess, "grounded_sql", "", "", "", nil),
		}, nil
	}
	if groundedReply, handled := a.tryGroundedCapabilitiesFastPath(intent, message, hasFinanceAccess); handled {
		return ButlerResponse{
			Message:    groundedReply,
			Actions:    []ButlerAction{},
			Confidence: 0.99,
			Context:    map[string]any{"fast_path": "grounded_capabilities"},
			Metadata:   buildButlerMetadata(intent, map[string]any{"fast_path": "grounded_capabilities"}, hasFinanceAccess, "grounded_sql", "", "", "", nil),
		}, nil
	}
	if groundedReply, groundedActions, handled := a.tryGroundedManagerFinancialBriefFastPath(intent, message, hasFinanceAccess); handled {
		return ButlerResponse{
			Message:    groundedReply,
			Actions:    groundedActions,
			Confidence: 0.99,
			Context:    map[string]any{"fast_path": "grounded_manager_financial_brief"},
			Metadata:   buildButlerMetadata(intent, map[string]any{"fast_path": "grounded_manager_financial_brief"}, hasFinanceAccess, "grounded_sql", "", "", "", nil),
		}, nil
	}
	if groundedReply, handled := a.tryGroundedTaskCreationFastPath(intent, message); handled {
		return ButlerResponse{
			Message:    groundedReply,
			Actions:    []ButlerAction{},
			Confidence: 0.99,
			Context:    map[string]any{"fast_path": "grounded_task_create"},
			Metadata:   buildButlerMetadata(intent, map[string]any{"fast_path": "grounded_task_create"}, hasFinanceAccess, "grounded_sql", "", "", "", nil),
		}, nil
	}
	if groundedReply, handled := a.tryGroundedWorkFastPath(intent, message); handled {
		return ButlerResponse{
			Message:    groundedReply,
			Actions:    []ButlerAction{},
			Confidence: 0.98,
			Context:    map[string]any{"fast_path": "grounded_work"},
			Metadata:   buildButlerMetadata(intent, map[string]any{"fast_path": "grounded_work"}, hasFinanceAccess, "grounded_sql", "", "", "", nil),
		}, nil
	}
	if groundedReply, handled := a.tryGroundedSupplierFastPath(intent, message); handled {
		return ButlerResponse{
			Message:    groundedReply,
			Actions:    []ButlerAction{},
			Confidence: 0.98,
			Context:    map[string]any{"fast_path": "grounded_supplier"},
			Metadata:   buildButlerMetadata(intent, map[string]any{"fast_path": "grounded_supplier"}, hasFinanceAccess, "grounded_sql", "", "", "", nil),
		}, nil
	}
	if groundedReply, handled := a.tryGroundedCustomerFastPath(intent, message); handled {
		return ButlerResponse{
			Message:    groundedReply,
			Actions:    []ButlerAction{},
			Confidence: 0.98,
			Context:    map[string]any{"fast_path": "grounded_customer"},
			Metadata:   buildButlerMetadata(intent, map[string]any{"fast_path": "grounded_customer"}, hasFinanceAccess, "grounded_sql", "", "", "", nil),
		}, nil
	}

	// 3. Check if this is a report generation request
	if isReportRequest(message) {
		return a.handleReportRequest(message, intent)
	}

	// 4. Build full context (all domains — Grok's large context window can handle it)
	context := a.buildFullContext(intent)

	// 5. Calculate regime
	regime := a.calculateSystemRegime()
	context["system_regime"] = regime

	// 6. Build system prompt with full context and regime
	systemPrompt := buildMistralSystemPrompt(context, regime)

	// 7. Call AIML/Grok (primary) or fall back to Mistral
	var aiResponse string
	var aiErr error
	usedBackend := "Mistral"
	requestedModel := getAIMLModelID()
	usedModel := ""
	fallbackReason := ""
	if aimlKey := getAIMLAPIKey(); aimlKey != "" {
		aiResponse, usedModel, aiErr = callAIMLWithFallback(aimlKey, systemPrompt, message)
		usedBackend = "AIML/Grok"
		if aiErr != nil {
			fallbackReason = aiErr.Error()
			log.Printf("⚠️ AIML API error, falling back to Mistral: %v", aiErr)
			aiResponse, aiErr = callMistral(mistralModelLarge, systemPrompt, message)
			usedBackend = "Mistral (fallback)"
		}
	} else {
		aiResponse, aiErr = callMistral(mistralModelLarge, systemPrompt, message)
	}

	if aiErr != nil {
		log.Printf("❌ AI backend error: %v", aiErr)
		fallbackResponse := a.buildGroundedModelFallbackResponse(intent, message, context, hasFinanceAccess, aiErr)
		fallbackResponse.Metadata.UsedBackend = firstNonEmpty(fallbackResponse.Metadata.UsedBackend, "grounded_sql_fallback")
		fallbackResponse.Metadata.RequestedModel = requestedModel
		fallbackResponse.Metadata.UsedModel = usedModel
		fallbackResponse.Metadata.FallbackReason = firstNonEmpty(fallbackResponse.Metadata.FallbackReason, fallbackReason, aiErr.Error())
		return fallbackResponse, nil
	}

	// 8. Parse response (extract actions, clean message)
	butlerResponse := parseMistralResponse(aiResponse, context, intent)
	butlerResponse.Metadata = buildButlerMetadata(
		intent,
		context,
		hasFinanceAccess,
		usedBackend,
		requestedModel,
		usedModel,
		fallbackReason,
		nil,
	)

	log.Printf("🤖 Butler responded via %s: %s (confidence: %.2f, actions: %d)",
		usedBackend, butlerTruncate(butlerResponse.Message, 60), butlerResponse.Confidence, len(butlerResponse.Actions))

	return butlerResponse, nil
}

// isReportRequest detects if the user is asking for a report/PDF generation
func isReportRequest(message string) bool {
	lower := strings.ToLower(message)
	reportTriggers := []string{"generate report", "create report", "make report", "produce report",
		"generate a report", "create a report", "make a report", "pdf report",
		"generate pdf", "create pdf", "detailed report", "intelligence report",
		"write a report", "write report", "prepare report", "prepare a report",
		"prepare a document", "create a document", "financial document", "manager document",
		"manager brief", "management brief", "manager briefing", "management briefing",
		"report on", "report for", "report about"}
	for _, trigger := range reportTriggers {
		if strings.Contains(lower, trigger) {
			return true
		}
	}
	return false
}

// detectReportType determines which report type to generate from the message
func detectReportType(message string, intent Intent) string {
	lower := strings.ToLower(message)

	if strings.Contains(lower, "customer") || strings.Contains(lower, "client") {
		return "customer"
	}
	if strings.Contains(lower, "financial") || strings.Contains(lower, "cash") || strings.Contains(lower, "revenue") {
		return "financial"
	}
	if strings.Contains(lower, "risk") || strings.Contains(lower, "overdue") || strings.Contains(lower, "aging") {
		return "risk"
	}
	if strings.Contains(lower, "operation") || strings.Contains(lower, "pipeline") || strings.Contains(lower, "order") {
		return "operations"
	}
	if strings.Contains(lower, "supplier") || strings.Contains(lower, "vendor") || strings.Contains(lower, "procurement") {
		return "supplier"
	}

	// Fall back to intent domain
	switch intent.Domain {
	case "customer":
		return "customer"
	case "supplier":
		return "supplier"
	case "financial":
		return "financial"
	case "risk":
		return "risk"
	case "operations":
		return "operations"
	}

	return "executive"
}

// handleReportRequest generates a PDF report and returns the path in the response
func (a *App) handleReportRequest(message string, intent Intent) (ButlerResponse, error) {
	reportType := detectReportType(message, intent)
	log.Printf("📊 Report request detected: type=%s entity=%q", reportType, intent.EntityName)
	financeAccess := a.currentSessionHasPermission("finance:view")
	reportMetadata := buildButlerMetadata(intent, map[string]any{
		"request":     message,
		"report_type": reportType,
	}, financeAccess, "local_report", "", "", "", nil)

	filePath, err := a.GenerateButlerReport(reportType, intent.EntityName)
	if err != nil {
		// Check if this is a permission error
		errorMsg := err.Error()
		isPermissionError := strings.Contains(errorMsg, "permission denied") || strings.Contains(errorMsg, "finance:view")

		if isPermissionError {
			return ButlerResponse{
				Message:    "I cannot generate this report due to insufficient permissions.\n\nFinancial, risk, and executive reports require manager or admin privileges.\n\nYou can still request:\n• Customer reports\n• Supplier reports\n• Operations reports\n\nPlease contact your manager for financial report access.",
				Actions:    []ButlerAction{},
				Confidence: 0.95,
				Context:    map[string]any{"report_type": reportType, "blocked": "permission_denied"},
				Metadata:   reportMetadata,
			}, nil
		}

		return ButlerResponse{
			Message:    fmt.Sprintf("I attempted to generate the report but encountered an error: %v", err),
			Actions:    []ButlerAction{},
			Confidence: 0.3,
			Context:    map[string]any{"report_type": reportType, "error": err.Error()},
			Metadata:   buildButlerMetadata(intent, map[string]any{"report_type": reportType, "error": err.Error()}, financeAccess, "local_report", "", "", err.Error(), err),
		}, nil
	}

	responseMsg := fmt.Sprintf("Your %s report has been generated successfully.\n\nFile saved to:\n%s\n\nThe report contains live business data from your database combined with AI-powered analysis covering key metrics, risk factors, and strategic recommendations.",
		getReportTitle(reportType), filePath)

	return ButlerResponse{
		Message:    responseMsg,
		Actions:    []ButlerAction{{Type: "fetch", Target: "report", Data: filePath}},
		Confidence: 0.95,
		Context:    map[string]any{"report_type": reportType, "file_path": filePath},
		Metadata:   reportMetadata,
	}, nil
}

// ============================================================================
// MISTRAL API CLIENT
// ============================================================================

// callMistral makes HTTP request to Mistral API
func callMistral(model, systemPrompt, userMessage string) (string, error) {
	requestBody := map[string]any{
		"model": model,
		"messages": []map[string]any{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"max_tokens":  mistralMaxTokens,
		"temperature": mistralTemperature,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", mistralAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("request creation error: %w", err)
	}

	// Get API key from app config (fallback to env, then hardcoded for backward compatibility)
	apiKey := getMistralAPIKey()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: MistralAPITimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response error: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Mistral API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("JSON parse error: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("no response choices from Mistral")
	}

	return apiResponse.Choices[0].Message.Content, nil
}

// callMistralVision sends an image to Mistral's pixtral vision model for OCR
func callMistralVision(base64Image string, mimeType string, prompt string) (string, error) {
	apiKey := getMistralAPIKey()
	if apiKey == "" {
		return "", fmt.Errorf("Mistral API key not configured")
	}

	requestBody := map[string]any{
		"model": "pixtral-large-latest",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": prompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": "data:" + mimeType + ";base64," + base64Image,
						},
					},
				},
			},
		},
		"max_tokens":  8192,
		"temperature": 0.1,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", mistralAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("request creation error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: MistralAPITimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response error: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Mistral Vision API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("JSON parse error: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("no response from Mistral Vision")
	}

	return apiResponse.Choices[0].Message.Content, nil
}

// ============================================================================
// AIML API CLIENT (Grok — primary chat backend)
// ============================================================================

// settingsBasedAIMLKey is injected by the App during startup for settings-based key lookup
var settingsBasedAIMLKey func() string
var settingsBasedAIMLModel func() string

// SetAIMLKeyProvider allows the App to inject the settings-based AIML key provider
func SetAIMLKeyProvider(provider func() string) {
	settingsBasedAIMLKey = provider
}

// SetAIMLModelProvider allows the App to inject the settings-based AIML model provider.
func SetAIMLModelProvider(provider func() string) {
	settingsBasedAIMLModel = provider
}

// getAIMLAPIKey retrieves the AIML API key from settings or environment.
// Priority: settings DB → ASYMM/AIML env var.
func getAIMLAPIKey() string {
	// 1. Try settings-based provider (injected by App)
	if settingsBasedAIMLKey != nil {
		if key := strings.TrimSpace(settingsBasedAIMLKey()); key != "" {
			return key
		}
	}
	// 2. Try environment variables
	if key := strings.TrimSpace(os.Getenv("ASYMM_AIML_API_KEY")); key != "" {
		return key
	}
	if key := strings.TrimSpace(os.Getenv("AIML_API_KEY")); key != "" {
		return key
	}
	return ""
}

func getAIMLModelID() string {
	if settingsBasedAIMLModel != nil {
		if model := strings.TrimSpace(settingsBasedAIMLModel()); model != "" {
			return model
		}
	}
	if model := strings.TrimSpace(os.Getenv("ASYMM_AIML_MODEL")); model != "" {
		return model
	}
	if model := strings.TrimSpace(os.Getenv("AIML_MODEL")); model != "" {
		return model
	}
	return aimlModelID
}

func getAIMLModelCandidates() []string {
	requested := getAIMLModelID()
	models := []string{}
	if requested != "" {
		models = append(models, requested)
	}
	models = append(models, fallbackAIMLModels...)
	return dedupeStringSlice(models)
}

func dedupeStringSlice(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

// callAIML sends a chat request to the AIML API (OpenAI-compatible format).
// Uses explicit model selection to support runtime fallback orchestration.
func callAIML(apiKey, systemPrompt, userMessage, model string) (string, error) {
	requestBody := map[string]any{
		"model": model,
		"messages": []map[string]any{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"max_tokens":  aimlMaxTokens,
		"temperature": aimlTemperature,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", aimlAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("request creation error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: AimlAPITimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response error: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("AIML API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("JSON parse error: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("no response choices from AIML API")
	}

	return apiResponse.Choices[0].Message.Content, nil
}

func callAIMLWithFallback(apiKey, systemPrompt, userMessage string) (string, string, error) {
	var lastErr error
	for _, model := range getAIMLModelCandidates() {
		response, err := callAIML(apiKey, systemPrompt, userMessage, model)
		if err == nil {
			return response, model, nil
		}
		lastErr = err
		log.Printf("⚠️ AIML model attempt failed (%s): %v", model, err)
	}
	return "", "", lastErr
}

// callAIMLWithMessages sends the full messages array to AIML API (preserves conversation history).
func callAIMLWithMessages(apiKey string, messages []map[string]any) (string, string, error) {
	var lastErr error
	for _, model := range getAIMLModelCandidates() {
		requestBody := map[string]any{
			"model":       model,
			"messages":    messages,
			"max_tokens":  aimlMaxTokens,
			"temperature": aimlTemperature,
		}
		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			lastErr = fmt.Errorf("marshal error: %w", err)
			continue
		}
		req, err := http.NewRequest("POST", aimlAPIURL, bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("request creation error: %w", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{Timeout: AimlAPITimeout}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			log.Printf("⚠️ AIML model attempt failed (%s): %v", model, lastErr)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("AIML API error (status %d): %s", resp.StatusCode, string(body))
			log.Printf("⚠️ AIML model attempt failed (%s): %v", model, lastErr)
			continue
		}
		var apiResponse struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(body, &apiResponse); err != nil {
			lastErr = fmt.Errorf("JSON parse error: %w", err)
			log.Printf("⚠️ AIML model attempt failed (%s): %v", model, lastErr)
			continue
		}
		if len(apiResponse.Choices) == 0 {
			lastErr = fmt.Errorf("no response choices from AIML API")
			log.Printf("⚠️ AIML model attempt failed (%s): %v", model, lastErr)
			continue
		}
		return apiResponse.Choices[0].Message.Content, model, nil
	}
	return "", "", lastErr
}

// ============================================================================
// SYSTEM PROMPT BUILDER
// ============================================================================

// buildMistralSystemPrompt creates the PRD-style context-aware system prompt
func buildMistralSystemPrompt(context map[string]any, regime map[string]any) string {
	contextStr := marshalContextForPrompt(context, butlerMaxContextChars)

	// Build regime guidance
	regimeGuidance := buildRegimeGuidance(regime)

	companyName, companyIndustry, companyCountry := currentCompanyIdentity()
	prompt := fmt.Sprintf(`You are Butler, the business intelligence AI for %s — a %s-based %s company.
You have access to connected business data in this system and should answer only from that evidence.
Do not invent entities, people, products, projects, invoice numbers, payment events, competitor notes, or business assertions not present in context.

CURRENT SYSTEM DATE:
- Treat today as %s.
- Treat the current operating year as %d.
- Do not describe explicitly requested periods in %d as "future" unless the period is actually after today.

DATA-GROUNDING RULES:
- Answer only from facts present in context data and keys listed below.
- If a requested data point is not present, state explicitly: "I cannot confirm this from current data." Then immediately provide the closest grounded data, adjacent metric, or next-best action you can support from context.
- For period-scoped questions (quarter/month/year), use 'customer_period_summary' when provided.
- For company-wide quarter/month/year questions, use 'business_period_summary' when provided and treat it as authoritative.
- For year-scoped questions, use 'business_year_summary' when provided and treat it as authoritative.
- If the user asks whether the system has access to a year's data, answer with coverage across invoices, orders, offers, and opportunities rather than only invoiced revenue.
- If period summary is empty, do not infer from lifetime data and present period-specific result only.

RBAC AND SALES-SAFE RULES:
- Treat context as role-scoped. If financial_data, ar_summary, banking_data, cashflow_projection, dso_analysis, customer_profitability, and forecast_intelligence are absent, the current user is not cleared for finance.
- For finance-restricted users, do not answer revenue, profit, margin, cash, AR/AP, bank, payment, outstanding, salary, total sales, or pipeline-value questions, even if the wording references sales pipeline or offers.
- Sales-safe help is allowed for RFQ/offer/order status by number, item/model/serial lookup, customer emails, follow-ups, delivery status, and next actions without money, margin, cash, or payment figures.
- If a restricted user asks for financial details, briefly say manager/admin permission is required and offer the closest sales-safe route.

COMPANY PROFILE:
- Industry: Process Instrumentation & Control Systems (Bahrain)
- Currency: BHD (Bahraini Dinar, 3 decimal places = fils)
- Divisions: `+divisionsSummaryLine()+`
- VAT Rate: 10%%

ROLE-SCOPED DATA ACCESS — you may have live data on:
• CUSTOMERS: live customer accounts — profiles, grades (A/B/C/D), payment behavior, credit limits, outstanding AR, invoices, orders, contacts, CRM notes, follow-up tasks, active offers, AR risk tiers
• CUSTOMER ACTIVITY: Dormant customers (no orders 6+ months), new customers this year, lapsed customers (ordered last year but not this year), customer retention signals
• CUSTOMER PROFITABILITY: Revenue and gross margin by customer — top earners, lowest margin accounts (pricing opportunities)
• SUPPLIERS: live supplier records — profiles, PO history, invoice matching, payment history, GRN quality stats
• SUPPLIER PERFORMANCE: Average delivery lead times (PO date → GRN date), QC pass/fail rates by supplier — who is reliable, who isn't
• SALES PIPELINE: Offers (RFQ → Quotation → Won/Lost/Expired), costing sheets, win rates, opportunities by stage, competition markers when stored
• OFFER EXPIRY: Active quotes approaching ValidityDate — this week / this month / next 60 days, total value at risk
• COMPETITION: Win/loss trend, top lost-deal reasons, and supplier/product competitiveness signals from stored records
• ORDERS & INVOICES: live orders and customer invoices, including status, line items, and margins
• PROCUREMENT: live purchase orders (multi-currency EUR/USD/BHD), 3-way matching (PO/GRN/Invoice), GRN quality
• DELIVERY: live delivery notes — Prepared/Dispatched/Delivered, serial traceability
• SERVICE DUE INTELLIGENCE: Serial-tracked installations nearing calibration, warranty, or annual-service milestones
• FINANCE: AR/AP, bank balances (multiple accounts), FX rates, monthly revenue trends, payment predictions
• DSO ANALYSIS: Days Sales Outstanding per customer — who pays in 15 days vs 90 days, overall DSO trend
• BANKING: Bank statements, reconciliation status, outstanding/stale/bounced cheques
• PAYMENT CLOSURE READINESS: Opportunities with linked invoices fully paid and therefore eligible to review for closure
• PRODUCTS: live product catalog, pricing, margins, order frequency, serial tracking
• FORECASTING: Weighted pipeline by stage, opportunities closing in 60 days, payment day predictions, win rates by product type, salesperson performance
• RISK: Overdue AR/AP, credit-blocked customers, expiring offers, system alerts
• ACTION ITEMS: Prioritized list — overdue invoices to chase, offers expiring this week, stuck delivery notes, POs awaiting supplier acknowledgment, supplier invoices pending approval
• DOCUMENT PRECHECKS: Quotation readiness context — resolved customer, primary contact, recent offers, missing required fields before drafting
• STRUCTURED MEMORY: Recent entity notes, pending follow-ups, and tactical quick captures relevant to the active customer/workflow
• PROACTIVE BRIEFING: Compact daily operating picture — action items, risks, service follow-ups, and cashflow cues
• WORK MANAGEMENT: Employees, collaborative tasks, task comments, and task notifications
• DATABASE ACCESS: Cross-module record retrieval covering notes, offers and line items, orders and line items, invoices, supplier invoices, payments, bank statements, expenses, and related operational records
• INTELLIGENCE: Three-Regime dynamics (R1=Exploration/R2=Optimization/R3=Stabilization), payment behavior models

COMPLETE BUSINESS DATA:
%s

%s

YOUR CAPABILITIES:
1. Answer role-authorized questions about the business — customers, suppliers, finance, operations, pipeline, risk
2. Discuss FUTURE PROSPECTS — use only explicit pipeline data, payment predictions, and win probability from context
3. STRATEGIC ANALYSIS — recommend actions, identify patterns, surface insights from data trends
4. FORECASTING — for finance-cleared users estimate future revenue using pipeline × close probability and payment timing risk; for sales users limit pipeline answers to status/count/next action
5. WHAT-IF SCENARIOS — reason about hypothetical situations with explicit stated assumptions
7. RELATIONSHIP ANALYSIS — cross-reference customers, orders, offers, and payments to find patterns
8. DAILY PRIORITIZATION — tell the team what to focus on today: overdue collections, expiring offers, stuck deliveries, pending approvals
9. CUSTOMER HEALTH — identify dormant accounts, churn risk, and new customer opportunities worth nurturing
10. COMPETITIVE INTELLIGENCE — analyze win/loss patterns and pricing vs margin trade-offs when present in context
11. SUPPLIER BENCHMARKING — compare lead times and QC reliability by supplier from context
12. PAYMENT BEHAVIOR — finance-cleared only: DSO by customer, predict who will pay late, recommend collection priority
13. SERVICE FOLLOW-UPS — identify installed equipment that is likely due for calibration, warranty action, or annual service
14. DOCUMENT READINESS — before drafting quotations or service offers, verify resolved customer/contact context and missing fields
15. PROACTIVE BRIEFING — summarize what needs attention today across collections, operations, service, and approvals
16. CONTINUITY — use structured memory to recall recent notes, follow-ups, and tactical tasks without inventing new facts
17. CREATE CUSTOMERS — create new customer records from natural language (name, type, grade, city, contact)
18. CREATE SUPPLIERS — create new supplier records from natural language (name, type, country, contact, brands)

RESPONSE GUIDELINES:
- Always cite specific numbers from the context data when available
- For forward-looking questions, clearly distinguish between actual data and projections
- Be as detailed or as brief as the question requires — match depth to complexity
- Write in polished executive prose. Do not use emojis, markdown heading markers like ###, or asterisks for emphasis.
- Use short section titles and simple dash bullets only when they improve readability.
- If data is not in context, say so and offer what you do know
- Do not dead-end with a refusal when related grounded data exists; pivot to the nearest useful answer, comparison, shortlist, or clarifying question
- Suggest next actions only when supported by context
- For strategic recommendations, ground them in the actual data provided
- Never invent contact people, internal owners, invoice numbers, project names, or payment events that are absent from context
- Treat entity_resolution, customer_data, supplier_data, employee_context, contact_context, account_manager_context, service_due_installations, closeable_opportunities, quotation_precheck, work_data, and database_access as higher-trust sources when present
- Treat clarification_prompt and quotation_precheck.draft_payload as the preferred handoff rails when they are present
- Treat butler_memory and proactive_briefing as trusted context for continuity and daily-priority answers
- For "who handled"/ownership questions, only mention people present in context fields (e.g., user references, account_manager_context, team context tags); otherwise state no owner is currently recorded
- If a period is requested (quarter/year/month) and customer_period_summary.status is empty_period_window or no period rows exist, present that clearly as no activity in that period
- If business_year_summary is present, use it to distinguish invoice activity from broader commercial coverage such as orders, offers, and opportunities
- Do not claim a PDF, Excel export, or any document file was generated unless the context explicitly includes a real file path or generated artifact reference
- If entity_resolution indicates ambiguous=true, ask a short clarification question and show the alternatives instead of committing to one match
- For cashflow questions, prefer cashflow_projection monthly_projection when present and state that operating cost is a static assumption if included
- For quotation drafting, use quotation_precheck and draft_clues to summarize what is known, what was inferred from the user's request, and what fields are still missing
- For questions about external events (bank outages, AWS incidents, political disruption, etc.), do not speculate about real-world causation from outside knowledge. Instead, quantify the internal exposure you can see in the system and state clearly that the external event itself is not confirmed in current data
- For create/update/delete requests, ask concise clarifying questions when required fields are missing; otherwise provide a short preview and executable action block rather than only descriptive text

ACTION FORMAT CONTRACT (must be explicit and execute-ready):
%s

Available targets:
%s
`,
		companyName, companyCountry, companyIndustry,
		time.Now().Format("2006-01-02"), time.Now().Year(), time.Now().Year(), contextStr, regimeGuidance, buildButlerActionContractPrompt(), strings.Join(buildButlerActionTargetAliases(), ", "))

	return prompt
}

// buildRegimeGuidance generates regime-specific LLM guidance
func buildRegimeGuidance(regime map[string]any) string {
	dominant, _ := regime["dominant"].(string)
	r1, _ := regime["r1"].(float64)
	r2, _ := regime["r2"].(float64)
	r3, _ := regime["r3"].(float64)
	state, _ := regime["state"].(string)

	guidance := fmt.Sprintf("BUSINESS REGIME: %s (%s) [R1=%.0f%% R2=%.0f%% R3=%.0f%%]\n", dominant, state, r1*100, r2*100, r3*100)

	switch dominant {
	case "R1":
		guidance += `REGIME FOCUS: EXPLORATION
- Suggest new business opportunities and market expansion
- Highlight growth potential in customer/supplier relationships
- Recommend calculated risks for new ventures
- Be optimistic but data-grounded`
	case "R2":
		guidance += `REGIME FOCUS: OPTIMIZATION
- Focus on efficiency improvements and margin enhancement
- Suggest cost reduction and process improvements
- Recommend pricing adjustments based on data
- Balance growth with profitability`
	case "R3":
		guidance += `REGIME FOCUS: STABILIZATION
- Prioritize collections and cash flow management
- Flag overdue payments and high-risk accounts
- Recommend conservative credit decisions
- Focus on risk mitigation and working capital`
	}

	return guidance
}

// ============================================================================
// RESPONSE PARSING
// ============================================================================

func buildButlerActionContract() string {
	return buildButlerActionContractPrompt()
}

func buildButlerActionContractPrompt() string {
	return `Use [ACTIONS]...[/ACTIONS] to return machine-executable actions.

Rules:
- Emit JSON action objects only when the capability is supported by live context.
- Use lower-case values for type and target.
- Include a short label for every action.
- Never emit an action if required fields are missing or unclear.
- If data is missing, ask for clarification instead of guessing IDs, statuses, or entities.
- Never invent entities, ids, statuses, or workflow statuses.

General schema:
{"type":"...","target":"...","label":"...","data":{...}}

1) APPROVE / REJECT
{"type":"approve|reject","target":"<target>","label":"...","data":{"entity_id":"<id>","reason":"<optional>"}}
Requirements:
- required: type, target, data.entity_id
- supported targets: purchase_order, costing_sheet, supplier_invoice, stock_adjustment, offer, quotation, order, rfq, invoice
- Never invent ids.

2) UPDATE
{"type":"update","target":"<target>","label":"...","data":{"entity_id":"<id>","status":"<exact status/stage string>"}}
Requirements:
- required: type, target, data.entity_id, data.status or data.stage
- allowed status/stage by target:
  - opportunity: New, Qualified, Proposal, Quoted, Won, Lost, On Hold
  - purchase_order: Draft, Pending Approval, Approved, Sent, Acknowledged, Partially Received, Received, Closed, Cancelled
  - order: Draft, Confirmed, Processing, InProgress, Shipped, PartiallyDelivered, FullyDelivered, Delivered, Invoiced, Complete, Cancelled
  - rfq: RFQ Received, Offer Sent, Follow-up/Eval, PO/LOI Received, Order Placed, In Process, Delivered, Closed (Payment), Closed (Lost)
  - offer/quotation: draft, quoted, sent, accepted, rejected, won, lost
  - costing_sheet: draft, pending_approval, approved, rejected
  - follow_up: pending, in_progress, completed, cancelled, overdue
  - stock_adjustment: pending, approved, rejected
- opportunity updates may also include comment and owner_notes when the request is about notes or detail cleanup

3) CREATE
{"type":"create", "target":"<target>", "label":"...", "data":{...}}
Requirements by target:
- offer/quotation: customer_id or customer_name, and one of line_items/items OR amount (grand_total, total, total_amount, amount, amount_bhd)
- follow_up: customer_id or customer_name, title
- order: order_number/reference, customer_id or customer_name, and amount
- opportunity: customer_id or customer_name, and project/title/opportunity_name
- stock_adjustment: inventory_item_id or item_id, reason, and one of variance or (system_quantity + physical_quantity)
- customer: business_name (required), optional: customer_type, payment_grade, city, country, primary_email, primary_phone, industry
- supplier: supplier_name (required), optional: supplier_type, country, primary_contact, email, phone, brands_handled
- customer_contact: customer_id or customer_name, contact_name, optional job_title, email, phone, address, is_primary_contact
- supplier_contact: supplier_id or supplier_name, contact_name, optional job_title, email, phone, address, is_primary_contact
- optional fields: unit_cost, notes, due_date, priority, quote_type, vat_rate

4) OPEN / NAVIGATE
{"type":"open|navigate","target":"<screen_or_entity>","label":"...","data":{}}
Requirements:
- required: type and target
- target should map to app screens: dashboard, opportunities, relationships, finance, operations, intelligence, settings, customers, suppliers, orders, offers, rfq, costing_sheet, purchase_order, supplier_invoice, invoice, stock_adjustment
- include lookup ids when available (offer_id, order_id, customer_id)

5) ANALYZE / FETCH / DAILY_BRIEFING
{"type":"analyze|fetch|daily_briefing","target":"<topic>","label":"...","data":{}}
Requirements:
- required: type and target.

6) STATUS FIELDING
- Always send explicit status/stage in data.status or data.stage for update actions.
- Never emit placeholders.
- Do not claim readiness if required fields are missing.

Examples:
[ACTIONS]
[{"type":"approve","target":"costing_sheet","label":"Approve costing sheet","data":{"entity_id":123,"reason":"Margin approval complete"}}]
[{"type":"reject","target":"supplier_invoice","label":"Reject supplier invoice","data":{"entity_id":456,"reason":"Mismatch against GRN"}}]
[{"type":"create","target":"offer_draft","label":"Create offer draft","data":{"customer_id":789,"customer_name":"NPC","quote_type":"Quotation","vat_rate":10}}]
[{"type":"create","target":"follow_up","label":"Create follow-up","data":{"customer_name":"NPC","title":"Request additional requirements","due_date":"2026-01-12","priority":"high"}}]
[{"type":"create","target":"stock_adjustment","label":"Create stock adjustment","data":{"inventory_item_id":"ITM-0001","reason":"Physical audit adjustment","physical_quantity":40,"system_quantity":38,"unit_cost":12.5}}]
[{"type":"create","target":"order","label":"Create order","data":{"order_number":"ORD-2026-111","customer_name":"NPC","total_amount":12450}}]
[{"type":"create","target":"opportunity","label":"Create opportunity","data":{"customer_name":"NPC","title":"Upgrade custody transfer skids","amount":18500,"notes":"Created from Butler request"}}]
[{"type":"create","target":"customer","label":"Create customer","data":{"business_name":"Gulf Industrial Co","customer_type":"Corporate","city":"Manama","country":"Bahrain","industry":"Oil & Gas"}}]
[{"type":"create","target":"supplier","label":"Create supplier","data":{"supplier_name":"Rhine Instruments AG","supplier_type":"Manufacturer","country":"Switzerland","brands_handled":"Rhine Instruments"}}]
[{"type":"create","target":"customer_contact","label":"Create customer contact","data":{"customer_name":"National Petroleum Co.","contact_name":"Pat Morgan","job_title":"Instrument Engineer - Procurement","email":"pat.morgan@nationalpetroleum.example","phone":"+973-1700-0000","mobile":"+973-1700-0000","address":"PO Box 0000, Manama, Bahrain"}}]
[{"type":"update","target":"opportunity","label":"Update opportunity notes","data":{"entity_id":"OPP-123","comment":"Customer requested revised commercial split","owner_notes":"Follow up after internal pricing review"}}]
[{"type":"update","target":"rfq","label":"Move RFQ to In Process","data":{"entity_id":333,"stage":"In Process"}}]
[{"type":"update","target":"stock_adjustment","label":"Approve adjustment","data":{"entity_id":77,"status":"approved"}}]
[{"type":"navigate","target":"finance","label":"Open finance hub","data":{}}]
[{"type":"open","target":"orders","label":"Open recent orders","data":{"order_id":42}}]
[{"type":"fetch","target":"daily_briefing","label":"Show daily briefing","data":{}}]
[/ACTIONS]`
}

func buildButlerActionTargetAliases() []string {
	return []string{
		"customers",
		"suppliers",
		"customer",
		"supplier",
		"invoices",
		"offers",
		"order",
		"orders",
		"opportunity",
		"opportunities",
		"rfq",
		"rfqs",
		"offer",
		"offer_draft",
		"deliveries",
		"dashboard",
		"opportunities",
		"purchase_order",
		"purchase_orders",
		"po",
		"delivery_note",
		"delivery_notes",
		"bank_reconciliation",
		"cheque_register",
		"cash_position",
		"butler",
		"costing_sheet",
		"costingsheet",
		"costing",
		"supplier_invoice",
		"supplier_invoices",
		"follow_up",
		"follow_up_task",
		"tasks",
		"follow_ups",
		"customer_contact",
		"customer_contacts",
		"supplier_contact",
		"supplier_contacts",
		"contact",
		"contacts",
		"stock_adjustment",
		"stock_adjustments",
		"daily_briefing",
		"contact",
		"quotation",
		"payment",
		"payments",
	}
}

// parseMistralResponse converts Mistral output to ButlerResponse
func parseMistralResponse(aiText string, context map[string]any, intent Intent) ButlerResponse {
	// Extract actions from [ACTIONS]...[/ACTIONS] blocks
	actions := parseActionBlocks(aiText)

	// Clean the message (remove action blocks from display text)
	cleanMessage := actionBlockRegex.ReplaceAllString(aiText, "")
	cleanMessage = strings.TrimSpace(cleanMessage)
	displayMessage := normalizeAssistantMessageContent(cleanMessage, actions)

	// Calculate confidence
	confidence := 0.85
	if intent.Confidence < 0.5 {
		confidence = 0.6
	}
	if len(cleanMessage) < 20 {
		confidence = 0.5
	}
	if strings.Contains(strings.ToLower(cleanMessage), "don't have") ||
		strings.Contains(strings.ToLower(cleanMessage), "not available") {
		confidence = 0.4
	}
	if len(actions) > 0 {
		confidence += 0.05 // Slight boost for actionable responses
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return ButlerResponse{
		Message:    displayMessage,
		Actions:    actions,
		Confidence: confidence,
		Context:    context,
	}
}

func buildButlerMetadata(intent Intent, context map[string]any, hasFinanceAccess bool, usedBackend, requestedModel, usedModel, fallbackReason string, err error) ButlerResponseMetadata {
	metadata := ButlerResponseMetadata{
		UsedBackend:       usedBackend,
		RequestedModel:    requestedModel,
		UsedModel:         usedModel,
		FallbackReason:    strings.TrimSpace(fallbackReason),
		FinanceDataAccess: hasFinanceAccess,
		ContextMode:       intent.Domain,
		DataCoverage:      collectContextCoverage(context),
		EntityResolution:  getEntityResolutionFromContext(context),
		GeneratedAt:       time.Now().Format(time.RFC3339),
	}
	if err != nil {
		metadata.Error = err.Error()
	}
	return metadata
}

func collectContextCoverage(context map[string]any) []string {
	if len(context) == 0 {
		return []string{}
	}

	keys := []string{
		"business_summary",
		"business_period_summary",
		"business_year_summary",
		"entity_resolution",
		"customer_data",
		"customer_period_summary",
		"installed_base_summary",
		"ar_summary",
		"supplier_data",
		"financial_data",
		"operations_data",
		"risk_data",
		"banking_data",
		"credit_notes",
		"product_catalog",
		"delivery_data",
		"po_summary",
		"forecast_intelligence",
		"serial_inventory",
		"cashflow_projection",
		"dso_analysis",
		"offer_expiry",
		"customer_profitability",
		"supplier_performance",
		"customer_activity",
		"service_due_installations",
		"closeable_opportunities",
		"quotation_precheck",
		"competition_analysis",
		"competition_intelligence",
		"customer_360",
		"contact_context",
		"employee_context",
		"account_manager_context",
		"work_data",
		"database_access",
		"proactive_briefing",
		"action_items",
		"customer_notes",
	}

	coverage := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := context[key]; ok && !isContextEmpty(value) {
			coverage = append(coverage, key)
		}
	}
	return coverage
}

func isContextEmpty(value any) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case map[string]any:
		return len(v) == 0
	case map[any]any:
		return len(v) == 0
	case []any:
		return len(v) == 0
	case string:
		return strings.TrimSpace(v) == ""
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return false
		}
		switch string(data) {
		case "null", "{}", "[]":
			return true
		default:
			return false
		}
	}
}

func getEntityResolutionFromContext(context map[string]any) map[string]any {
	raw, ok := context["entity_resolution"]
	if !ok {
		return nil
	}
	resolution, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return resolution
}

// parseActionBlocks extracts structured actions from [ACTIONS]...[/ACTIONS] JSON
func parseActionBlocks(text string) []ButlerAction {
	actions := []ButlerAction{}

	matches := actionBlockRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		jsonStr := strings.TrimSpace(match[1])

		type actionAlias struct {
			Type        string `json:"type"`
			ActionType  string `json:"action_type"`
			Target      string `json:"target"`
			Topic       string `json:"topic"`
			ActionTag   string `json:"action_tag"`
			Screen      string `json:"screen"`
			Entity      string `json:"entity"`
			EntityType  string `json:"entity_type"`
			Resource    string `json:"resource"`
			Label       string `json:"label"`
			ActionLabel string `json:"action_label"`
			Name        string `json:"name"`
			Data        any    `json:"data"`
			Parameters  any    `json:"parameters"`
			Payload     any    `json:"payload"`
		}

		getStringFromData := func(raw any, keys ...string) string {
			if raw == nil {
				return ""
			}
			mapValue, ok := raw.(map[string]any)
			if !ok {
				return ""
			}
			for _, key := range keys {
				if text, ok := mapValue[key].(string); ok {
					text = strings.TrimSpace(text)
					if text != "" {
						return text
					}
				}
			}
			return ""
		}

		rawActions := []actionAlias{}
		if err := json.Unmarshal([]byte(jsonStr), &rawActions); err != nil {
			// Try single-action object fallback
			var single actionAlias
			if err2 := json.Unmarshal([]byte(jsonStr), &single); err2 == nil {
				rawActions = append(rawActions, single)
			} else {
				log.Printf("⚠️ Failed to parse action block: %v", err)
				continue
			}
		}

		for _, ra := range rawActions {
			typeValue := strings.ToLower(strings.TrimSpace(coalesceString(ra.Type, ra.ActionType, ra.ActionTag)))
			rawTarget := coalesceString(ra.Target, ra.Screen, ra.Topic, ra.Entity, ra.EntityType, ra.Resource)
			nType, nTarget := normalizedButlerActionTypeTarget(typeValue, rawTarget)
			if nType == "" {
				continue
			}
			target := nTarget
			label := strings.TrimSpace(coalesceString(ra.Label, ra.ActionLabel, ra.Name))
			data := coalesceInterface(ra.Data, ra.Parameters, ra.Payload)

			if target == "" && data != nil {
				if fallback := getStringFromData(data, "target", "entity", "entity_type", "screen", "topic", "resource"); fallback != "" {
					target = normalizeButlerActionTarget(fallback)
				}
			}

			if nType == "daily_briefing" && target == "" {
				target = "daily_briefing"
			}

			if nType == "create" && target == "" {
				if getStringFromData(data, "entity", "entity_type", "target", "table") == "quotation" {
					target = "offer"
				}
			}

			if label == "" {
				label = inferActionLabel(nType, target)
			}

			if target == "" || typeValue == "" {
				continue
			}

			actions = append(actions, ButlerAction{
				Type:   nType,
				Target: target,
				Label:  label,
				Data:   data,
			})
		}
	}

	return actions
}

func coalesceString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func coalesceInterface(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func inferActionLabel(actionType, target string) string {
	return butlerchat.InferActionLabel(actionType, target)
}

// ============================================================================
// UTILITY
// ============================================================================

// promptInjectionPatterns matches known Mistral/Llama prompt injection markers (case-insensitive)
var promptInjectionPatterns = regexp.MustCompile(`(?i)(\[/?INST\]|<</?SYS>>|<\||(\|>)|</?s>)`)

// contextCleanupPatterns matches action/system/instruction markers that could hijack responses (case-insensitive)
var contextCleanupPatterns = regexp.MustCompile(`(?i)(\[/?ACTIONS\]|SYSTEM:|INSTRUCTIONS:)`)

// sanitizeForPrompt removes potential prompt injection markers from user-supplied data
// before it is interpolated into Mistral API prompts.
// Does NOT truncate — callers are responsible for their own size limits.
func sanitizeForPrompt(input string) string {
	return butlerchat.SanitizeForPrompt(input)
}

func marshalContextForPrompt(context map[string]any, maxChars int) string {
	return butlerchat.MarshalContextForPromptCompact(context, maxChars)
}

// settingsBasedMistralKey is set by the App during startup to provide settings-based key lookup
var settingsBasedMistralKey func() string

// SetMistralKeyProvider allows the App to inject the settings-based key provider
func SetMistralKeyProvider(provider func() string) {
	settingsBasedMistralKey = provider
}

// getMistralAPIKey retrieves API key from settings, then environment
func getMistralAPIKey() string {
	// Helper to check if a key looks valid (not placeholder text)
	isValidKey := func(key string) bool {
		key = strings.TrimSpace(key)
		if key == "" {
			return false
		}
		// Reject placeholder values from settings UI
		lower := strings.ToLower(key)
		if strings.Contains(lower, "not set") || strings.Contains(lower, "not_set") ||
			strings.Contains(lower, "placeholder") || strings.Contains(lower, "your-") ||
			strings.Contains(lower, "your_") || strings.HasPrefix(key, "(") {
			return false
		}
		return true
	}

	// 1. Try settings-based provider (injected by App)
	if settingsBasedMistralKey != nil {
		if key := settingsBasedMistralKey(); isValidKey(key) {
			return key
		}
	}

	// 2. Try environment variable
	if key := os.Getenv("MISTRAL_API_KEY"); isValidKey(key) {
		return key
	}

	// 3. No key configured — Butler AI features stay disabled until a key is
	// supplied via MISTRAL_API_KEY or in-app settings. Never hardcode provider
	// credentials: anything returned here ships inside the distributed binary.
	return ""
}

// butlerTruncate truncates a string to maxLen characters
func butlerTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ============================================================================
// INTELLIGENT OCR ANALYSIS
// ============================================================================

// ButlerOCRInsight represents AI-extracted insights from document text
type ButlerOCRInsight struct {
	Summary           string           `json:"summary"`           // 1-2 sentence summary
	ExtractedItems    []map[string]any `json:"extracted_items"`   // Line items (flexible: RFQ items OR bank transactions)
	DetectedCustomer  string           `json:"detected_customer"` // Customer name if found
	DetectedProject   string           `json:"detected_project"`  // Project/reference if found
	RequiredDeadline  string           `json:"required_deadline"` // Deadline if mentioned
	Confidence        float64          `json:"confidence"`        // 0.0-1.0 confidence score
	SuggestedActions  []ButlerAction   `json:"suggested_actions"` // Recommended next steps
	DocumentType      string           `json:"document_type"`     // RFQ, PO, Invoice, etc.
	ExtractedMetadata map[string]any   `json:"metadata"`          // Key-value pairs extracted (bank name, balances, etc.)
}

// RFQLineItem represents a line item extracted from an RFQ/email
type RFQLineItem struct {
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	Unit        string `json:"unit"`
	PartNumber  string `json:"part_number"`
	Notes       string `json:"notes"`
}

// AnalyzeDocumentWithButler uses Mistral to extract structured insights from document text
// Note: metadata accepts map[string]interface{} for flexibility with OCR extracted_data
func (a *App) AnalyzeDocumentWithButler(text string, docType string, metadata map[string]any) (ButlerOCRInsight, error) {
	if err := a.requirePermission("intelligence:chat"); err != nil {
		return ButlerOCRInsight{}, err
	}
	if text == "" {
		return ButlerOCRInsight{}, fmt.Errorf("no text provided for analysis")
	}

	// SECURITY: Input validation - limit text size to prevent DoS
	const maxTextLength = 100000 // ~100KB, ~25K tokens max
	if len(text) > maxTextLength {
		text = text[:maxTextLength]
		log.Printf("⚠️ Butler: Text truncated to %d chars for analysis", maxTextLength)
	}

	// SECURITY: Validate docType to prevent injection
	validDocTypes := map[string]bool{
		"rfq": true, "RFQ": true, "po": true, "PO": true, "purchase_order": true, "PurchaseOrder": true,
		"invoice": true, "Invoice": true, "quote": true, "Quote": true,
		"email": true, "Email": true, "delivery_note": true, "DeliveryNote": true,
		"grn": true, "GRN": true, "unknown": true, "": true,
		"bank_statement": true, "BankStatement": true,
		"supplier_invoice": true, "SupplierInvoice": true,
		"quotation": true, "Quotation": true,
		"costing": true, "Costing": true,
		"excel_data": true, "ExcelData": true,
		"contract": true, "Contract": true,
		"Other": true,
	}
	if !validDocTypes[docType] {
		docType = "unknown"
		log.Printf("⚠️ Butler: Invalid document type sanitized to 'unknown'")
	}

	// Check API key availability
	apiKey := getMistralAPIKey()
	if apiKey == "" {
		return ButlerOCRInsight{
			Summary:    "Mistral API key not configured. Please add your API key in Settings > AI & Intelligence.",
			Confidence: 0.0,
		}, fmt.Errorf("MISTRAL_API_KEY not configured")
	}

	// Build context from metadata (handles any value type via %v)
	contextInfo := ""
	if metadata != nil {
		if from, ok := metadata["from"]; ok && from != nil && fmt.Sprintf("%v", from) != "" {
			contextInfo += fmt.Sprintf("From: %v\n", from)
		}
		if subject, ok := metadata["subject"]; ok && subject != nil && fmt.Sprintf("%v", subject) != "" {
			contextInfo += fmt.Sprintf("Subject: %v\n", subject)
		}
		if date, ok := metadata["date"]; ok && date != nil && fmt.Sprintf("%v", date) != "" {
			contextInfo += fmt.Sprintf("Date: %v\n", date)
		}
		// Also include customer_name and invoice_number if present (common OCR fields)
		if customer, ok := metadata["customer_name"]; ok && customer != nil && fmt.Sprintf("%v", customer) != "" {
			contextInfo += fmt.Sprintf("Customer: %v\n", customer)
		}
		if invNum, ok := metadata["invoice_number"]; ok && invNum != nil && fmt.Sprintf("%v", invNum) != "" {
			contextInfo += fmt.Sprintf("Invoice/Reference: %v\n", invNum)
		}
		if total, ok := metadata["total"]; ok && total != nil {
			contextInfo += fmt.Sprintf("Total Amount: %v\n", total)
		}
	}

	// Build analysis prompt
	aiCompanyName, aiCompanyIndustry, aiCompanyCountry := currentCompanyIdentity()
	systemPrompt := fmt.Sprintf(`You are a document analysis AI for %s, a %s company in %s.`, aiCompanyName, aiCompanyIndustry, aiCompanyCountry) + `

Your task is to extract structured information from business documents (emails, RFQs, purchase orders, invoices, bank statements, delivery notes, quotations).

COMPANY CONTEXT:
- Industry: Process Instrumentation
- Key Suppliers: Rhine Instruments, Oxan Analytics, Helvetia Metering, GIC
- Key Customers: NPC, Gulf Smelting, NGA, DPC, NOA (industrial/government)
- Products: Flow meters, level transmitters, pressure sensors, analyzers, temperature sensors

EXTRACTION RULES:
1. Identify the document type (RFQ, Purchase Order, Invoice, Quote, BankStatement, DeliveryNote, SupplierInvoice, Inquiry, Other)
2. Extract customer/sender company name
3. Extract project name or reference number if mentioned
4. Extract line items with quantities, descriptions, part/model numbers, product codes, unit prices, and currency when present
5. Identify any deadlines or urgency indicators
6. Note any special requirements or conditions
7. For bank statements: Extract bank name, account number, statement period, opening/closing balance, and list transactions as line items

OUTPUT FORMAT (JSON):
For RFQs, Invoices, POs, Quotes:
{
  "document_type": "RFQ|PO|Invoice|Quote|DeliveryNote|SupplierInvoice|Inquiry|Other",
  "summary": "1-2 sentence summary of the document",
  "customer": "Company name",
  "project": "Project name or reference",
  "deadline": "Date or urgency description",
  "line_items": [
    {"description": "...", "quantity": 1, "unit": "pcs", "part_number": "...", "product_code": "...", "model": "...", "unit_price": 0.000, "currency": "BHD", "raw_text": "..."}
  ],
  "metadata": {},
  "confidence": 0.85,
  "suggested_actions": ["Create RFQ", "Generate Costing", "Contact customer"]
}

For Bank Statements:
{
  "document_type": "BankStatement",
  "summary": "Bank statement summary with key figures",
  "customer": "",
  "project": "",
  "deadline": "",
  "line_items": [
    {"date": "2026-01-02", "description": "Payment received", "reference": "TXN123", "debit": 0, "credit": 500.000, "balance": 71058.251}
  ],
  "metadata": {
    "bank_name": "Demo Bank A",
    "account_number": "0000010000000001",
    "iban": "BH29DMOA10000000000001",
    "opening_balance": "70558.251",
    "closing_balance": "87980.027",
    "period_start": "01/01/2026",
    "period_end": "31/01/2026",
    "currency": "BHD",
    "total_debits": "123456.789",
    "total_credits": "140878.565"
  },
  "confidence": 0.95,
  "suggested_actions": ["Import to Bank Reconciliation"]
}

IMPORTANT:
- Return ONLY valid JSON, no markdown or explanation
- Use 0.0-1.0 for confidence based on how clear the information is
- If information is not found, use empty string or empty array
- Be conservative with quantities - only extract if clearly stated
- Do not put currency codes such as BHD, USD, or EUR in the "unit" field. Use "currency" for currency and "unit" only for physical units such as pcs, sets, m, kg, lot, days, or hours.
- For Rhine Instruments and instrumentation RFQs, preserve item/model codes exactly in part_number/product_code/model even when the description is long.
- For bank statements: DEBIT = money LEAVING the account (outgoing payments, transfers out, charges). CREDIT = money ENTERING the account (incoming payments, transfers in, deposits). Incoming transfers from customers (Fawri, INWD, deposit, TRF-IN, receipts) should be CREDITS. Outgoing payments (DGC, ONUS, Fawri-out, TRF-OUT, withdrawals, fees, charges) should be DEBITS. Use the running balance to validate: if the balance increases after a transaction it is a credit; if it decreases it is a debit. Preserve the bank statement exactly as presented: bank fees (EFTS Charges, SWIFT FEES, Corr Bnk Chg), VAT on fees, and similar service charges must remain as separate transaction rows if the statement shows them separately. Do NOT merge VAT/fee lines into a parent transaction. Use date format YYYY-MM-DD (e.g. 2026-01-15).`

	// Sanitize user-supplied data before injection into prompt
	userPrompt := fmt.Sprintf("Analyze this %s document and extract structured information:\n\n%s%s",
		docType, sanitizeForPrompt(contextInfo), sanitizeForPrompt(text))

	// Call Mistral API
	response, err := callMistral(mistralModelLarge, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("❌ Mistral analysis error: %v", err)
		return ButlerOCRInsight{
			Summary:    fmt.Sprintf("Analysis failed: %v", err),
			Confidence: 0.0,
		}, err
	}

	// Log raw response for debugging (truncated to 3000 chars)
	logLen := len(response)
	if logLen > 3000 {
		logLen = 3000
	}
	log.Printf("📋 Raw Mistral response (%d chars): %s", len(response), response[:logLen])

	// Parse JSON response
	insight := ButlerOCRInsight{
		ExtractedItems:    []map[string]any{},
		SuggestedActions:  []ButlerAction{},
		ExtractedMetadata: make(map[string]any),
	}

	// Try to parse as flexible JSON (captures ALL fields from Mistral, including bank transaction fields)
	var parsed struct {
		DocumentType     string           `json:"document_type"`
		Summary          string           `json:"summary"`
		Customer         string           `json:"customer"`
		Project          string           `json:"project"`
		Deadline         string           `json:"deadline"`
		LineItems        []map[string]any `json:"line_items"`
		Metadata         map[string]any   `json:"metadata"`
		Confidence       float64          `json:"confidence"`
		SuggestedActions []string         `json:"suggested_actions"`
	}

	// Extract JSON from response - robust approach that handles markdown wrapping,
	// trailing text, and partial truncation. Find first '{' and last '}' to isolate JSON.
	cleanResponse := strings.TrimSpace(response)
	jsonStart := strings.Index(cleanResponse, "{")
	jsonEnd := strings.LastIndex(cleanResponse, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		cleanResponse = cleanResponse[jsonStart : jsonEnd+1]
	}
	log.Printf("📋 Extracted JSON (%d chars), starts with: %.100s", len(cleanResponse), cleanResponse)

	if err := json.Unmarshal([]byte(cleanResponse), &parsed); err != nil {
		log.Printf("⚠️ Failed to parse AI response as JSON: %v (first 200 chars: %s)", err, cleanResponse[:min(len(cleanResponse), 200)])
		// Return raw response as summary
		insight.Summary = response
		insight.Confidence = 0.5
		return insight, nil
	}

	// Map parsed data to insight
	insight.DocumentType = parsed.DocumentType
	insight.Summary = parsed.Summary
	insight.DetectedCustomer = parsed.Customer
	insight.DetectedProject = parsed.Project
	insight.RequiredDeadline = parsed.Deadline
	insight.Confidence = parsed.Confidence
	insight.ExtractedMetadata = parsed.Metadata

	// Pass line items through directly - flexible map captures ALL fields
	// (RFQ: description, quantity, unit, part_number)
	// (Bank: date, description, reference, debit, credit, balance)
	insight.ExtractedItems = parsed.LineItems

	// Convert suggested actions
	for _, action := range parsed.SuggestedActions {
		insight.SuggestedActions = append(insight.SuggestedActions, ButlerAction{
			Type:   "navigate",
			Target: strings.ToLower(strings.ReplaceAll(action, " ", "_")),
			Data:   action,
		})
	}

	log.Printf("🧠 Document analyzed: type=%s customer=%q items=%d confidence=%.2f metadata=%v",
		insight.DocumentType, insight.DetectedCustomer, len(insight.ExtractedItems), insight.Confidence, parsed.Metadata)

	return insight, nil
}

// TestMistralConnection tests the Mistral API connection with a simple query
func (a *App) TestMistralConnection() (bool, error) {
	if err := a.requirePermission("intelligence:chat"); err != nil {
		return false, err
	}
	apiKey := getMistralAPIKey()
	if apiKey == "" {
		return false, fmt.Errorf("Mistral API key not configured. Add your key in Settings > AI & Intelligence")
	}

	// Make a simple test request
	response, err := callMistral(mistralModelSmall, "You are a test assistant.", "Say 'OK' if you can hear me.")
	if err != nil {
		return false, fmt.Errorf("Mistral API test failed: %v", err)
	}

	// Check if we got a reasonable response
	if len(response) > 0 {
		log.Printf("✅ Mistral API connection successful: %s", butlerTruncate(response, 50))
		return true, nil
	}

	return false, fmt.Errorf("empty response from Mistral API")
}
