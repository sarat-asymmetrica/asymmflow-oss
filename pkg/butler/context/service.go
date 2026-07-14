// Package context builds the Butler's grounded business context: the
// intent-aware and full-context data bundles handed to the model, entity
// reference resolution, period/year summaries, and the finance redaction
// rules. Read-only by construction (invariant 4: Butler paths inspect,
// explain and draft — they never persist).
//
// Moved from the trading root in Wave 6 (Mission A.1). The host stays
// behind HostPort: work/task context, employee resolution and quick
// captures live with the collaboration hub, and the cashflow projection
// with the finance reporting surface — ports, never relocation (W4-D9).
// RBAC stays with the host too: callers compute hasFinanceAccess at the
// chokepoint and pass it down.
package context

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	butlerdomain "ph_holdings_app/pkg/butler"
	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/finance"
	financepayroll "ph_holdings_app/pkg/finance/payroll"
	"ph_holdings_app/pkg/infra"
	"ph_holdings_app/pkg/kernel/text"
	"ph_holdings_app/pkg/overlay"
)

// HostPort is what the context builder needs from the host application:
// reads that live with the collaboration hub (work data, employees, quick
// captures) and the finance reporting surface (cashflow projection).
type HostPort interface {
	WorkContext(intent Intent) map[string]any
	ResolveEmployeeReference(reference string) *ButlerResolvedEntity
	EmployeeContext(resolution *ButlerResolvedEntity) map[string]any
	RecentOpenQuickCaptures() []map[string]any
	CashflowProjectionContext() map[string]any
	OpenDedupedOpportunities() []Opportunity
}

// Service builds grounded context bundles from the primary database.
type Service struct {
	db   *gorm.DB
	host HostPort
}

// New constructs a context Service. db may be nil (builders return empty
// context), matching the historical root behavior during early startup.
func New(db *gorm.DB, host HostPort) *Service {
	return &Service{db: db, host: host}
}

// Butler domain vocabulary.
type (
	Intent               = butlerdomain.Intent
	ButlerResolvedEntity = butlerdomain.ButlerResolvedEntity
	ButlerAction         = butlerdomain.ButlerAction
	PredictionRecord     = butlerdomain.PredictionRecord
)

// Model vocabulary: these aliases keep the moved bodies byte-close to the
// root originals; the models themselves moved to pkg/* in earlier waves.
type (
	CustomerMaster         = crm.CustomerMaster
	CustomerContact        = crm.CustomerContact
	SupplierMaster         = crm.SupplierMaster
	SupplierContact        = crm.SupplierContact
	SupplierIssue          = crm.SupplierIssue
	ProductMaster          = crm.ProductMaster
	Offer                  = crm.Offer
	OfferItem              = crm.OfferItem
	OfferFollowUp          = crm.OfferFollowUp
	OfferNote              = crm.OfferNote
	Opportunity            = crm.Opportunity
	Order                  = crm.Order
	OrderItem              = crm.OrderItem
	DeliveryNote           = crm.DeliveryNote
	PurchaseOrder          = crm.PurchaseOrder
	GoodsReceivedNote      = crm.GoodsReceivedNote
	GRNItem                = crm.GRNItem
	SerialNumber           = crm.SerialNumber
	EntityNote             = crm.EntityNote
	FollowUpTask           = crm.FollowUpTask
	Invoice                = finance.Invoice
	DBInvoiceItem          = finance.DBInvoiceItem
	CreditNote             = finance.CreditNote
	Payment                = finance.Payment
	SupplierInvoice        = finance.SupplierInvoice
	SupplierInvoiceItem    = finance.SupplierInvoiceItem
	SupplierPayment        = finance.SupplierPayment
	ExpenseEntry           = finance.ExpenseEntry
	BankAccount            = finance.BankAccount
	BankStatement          = finance.BankStatement
	BankStatementLine      = finance.BankStatementLine
	BookBankReconciliation = finance.BookBankReconciliation
	OutstandingCheque      = finance.OutstandingCheque
	CompanyBankAccount     = finance.CompanyBankAccount
	CurrencyExchangeRate   = finance.CurrencyExchangeRate
	PayrollRun             = financepayroll.Run
	User                   = infra.User
	Alert                  = infra.Alert
)

// buildIntentContext gathers relevant data based on classified intent
func (svc *Service) BuildIntentContext(intent Intent, hasFinanceAccess bool) map[string]any {
	context := make(map[string]any)

	if svc.db == nil {
		return context
	}

	// Check if user has finance access (for redacting financial data from non-finance contexts)

	// Always include dashboard summary (redacted for non-finance users)
	summary := svc.BusinessSummary()
	if !hasFinanceAccess {
		delete(summary, "total_revenue_bhd")
		delete(summary, "total_outstanding_bhd")
	}
	context["business_summary"] = summary
	context["system_time"] = svc.getSystemTimeContext()
	if periodSummary := svc.getBusinessPeriodSummary(intent.RawQuery); len(periodSummary) > 0 {
		context["business_period_summary"] = periodSummary
	}
	if yearSummary := svc.BusinessYearSummary(intent.RawQuery); len(yearSummary) > 0 {
		context["business_year_summary"] = yearSummary
	}
	svc.injectGroundedReferenceContext(context, intent, hasFinanceAccess)
	context["butler_memory"] = svc.getButlerMemoryContext(intent)
	context["proactive_briefing"] = svc.getProactiveBriefingContext(hasFinanceAccess)
	context["work_data"] = svc.host.WorkContext(intent)

	// Domain-specific context injection
	switch intent.Domain {
	case "customer":
		customerCtx, ok := context["customer_data"].(map[string]any)
		if !ok || len(customerCtx) == 0 {
			customerCtx = svc.getCustomerContext(intent.EntityName)
		}
		if !hasFinanceAccess {
			// Redact financial data from customer context for non-finance users
			svc.redactFinancialFromContext(customerCtx)
		}
		context["customer_data"] = customerCtx
		if quarterSummary := svc.getCustomerPeriodSummary(intent.EntityName, intent.RawQuery); len(quarterSummary) > 0 {
			context["customer_period_summary"] = quarterSummary
		} else if _, ok := parseCustomerPeriodLabel(intent.RawQuery); ok {
			context["customer_period_summary"] = svc.getMissingCustomerPeriodSummary(intent.EntityName, intent.RawQuery)
		}
		context["installed_base_summary"] = svc.getInstalledBaseSummary(intent.EntityName)
		if hasFinanceAccess {
			context["ar_summary"] = svc.getARSummary()
		}

	case "supplier":
		context["supplier_data"] = svc.getSupplierContext(intent.EntityName)
		context["po_summary"] = svc.getPurchaseOrderSummary()

	case "financial":
		context["financial_data"] = svc.getFinancialContext()
		context["ar_summary"] = svc.getARSummary()
		context["cashflow_projection"] = svc.host.CashflowProjectionContext()

	case "operations":
		context["operations_data"] = svc.getOperationsContext()
		context["po_summary"] = svc.getPurchaseOrderSummary()
		context["risk_data"] = svc.getRiskContext() // Include risk for operations awareness
		context["service_due_installations"] = svc.getServiceDueInstallationsContext()
		context["quotation_precheck"] = svc.getQuotationPrecheckContext(intent)

	case "risk":
		context["risk_data"] = svc.getRiskContext()
		context["ar_summary"] = svc.getARSummary()
		context["operations_data"] = svc.getOperationsContext() // Include ops for delivery risk context
		context["service_due_installations"] = svc.getServiceDueInstallationsContext()
		context["closeable_opportunities"] = svc.getCloseableOpportunitiesContext()
		if hasFinanceAccess {
			context["cashflow_projection"] = svc.host.CashflowProjectionContext()
		}

	default:
		// General query - provide comprehensive overview
		context["customer_data"] = svc.getCustomerContext(intent.EntityName)
		if quarterSummary := svc.getCustomerPeriodSummary(intent.EntityName, intent.RawQuery); len(quarterSummary) > 0 {
			context["customer_period_summary"] = quarterSummary
		} else if _, ok := parseCustomerPeriodLabel(intent.RawQuery); ok {
			context["customer_period_summary"] = svc.getMissingCustomerPeriodSummary(intent.EntityName, intent.RawQuery)
		}
		context["installed_base_summary"] = svc.getInstalledBaseSummary(intent.EntityName)
		context["ar_summary"] = svc.getARSummary()
		context["operations_data"] = svc.getOperationsContext()
		context["risk_data"] = svc.getRiskContext()
		context["service_due_installations"] = svc.getServiceDueInstallationsContext()
		context["closeable_opportunities"] = svc.getCloseableOpportunitiesContext()
		context["quotation_precheck"] = svc.getQuotationPrecheckContext(intent)
		if hasFinanceAccess {
			context["financial_data"] = svc.getFinancialContext()
			context["cashflow_projection"] = svc.host.CashflowProjectionContext()
		}
	}

	if !hasFinanceAccess {
		svc.redactRestrictedBusinessContext(context)
	}

	return context
}

// buildFullContext gathers ALL domain data for Grok's large context window.
// Unlike buildIntentContext (domain-specific), this loads everything at once —
// customer, supplier, financial, operations, risk — and lets Grok figure out
// what's relevant to the user's query. RBAC filtering still applies.
func (svc *Service) BuildFullContext(intent Intent, hasFinanceAccess bool) map[string]any {
	context := make(map[string]any)

	if svc.db == nil {
		return context
	}

	// Business summary (always included)
	summary := svc.BusinessSummary()
	if !hasFinanceAccess {
		delete(summary, "total_revenue_bhd")
		delete(summary, "total_outstanding_bhd")
	}
	context["business_summary"] = summary
	context["system_time"] = svc.getSystemTimeContext()
	if periodSummary := svc.getBusinessPeriodSummary(intent.RawQuery); len(periodSummary) > 0 {
		context["business_period_summary"] = periodSummary
	}
	if yearSummary := svc.BusinessYearSummary(intent.RawQuery); len(yearSummary) > 0 {
		context["business_year_summary"] = yearSummary
	}
	svc.injectGroundedReferenceContext(context, intent, hasFinanceAccess)
	context["butler_memory"] = svc.getButlerMemoryContext(intent)
	context["proactive_briefing"] = svc.getProactiveBriefingContext(hasFinanceAccess)

	// Customer context — includes entity-specific data if name was extracted from query
	customerCtx, ok := context["customer_data"].(map[string]any)
	if !ok || len(customerCtx) == 0 {
		customerCtx = svc.getCustomerContext(intent.EntityName)
	}
	if !hasFinanceAccess {
		svc.redactFinancialFromContext(customerCtx)
	}
	context["customer_data"] = customerCtx

	// Supplier context — entity-specific if supplier name extracted, always top suppliers
	context["supplier_data"] = svc.getSupplierContext(intent.EntityName)
	if quarterSummary := svc.getCustomerPeriodSummary(intent.EntityName, intent.RawQuery); len(quarterSummary) > 0 {
		context["customer_period_summary"] = quarterSummary
	} else if _, ok := parseCustomerPeriodLabel(intent.RawQuery); ok {
		context["customer_period_summary"] = svc.getMissingCustomerPeriodSummary(intent.EntityName, intent.RawQuery)
	}

	// Financial context (only if user has finance:view permission)
	if hasFinanceAccess {
		context["financial_data"] = svc.getFinancialContext()
		context["ar_summary"] = svc.getARSummary()
	}
	context["action_items"] = svc.getActionItemsContext()
	context["competition_intelligence"] = svc.getCompetitionContext()

	// Operations context (always included — non-sensitive pipeline data)
	context["operations_data"] = svc.getOperationsContext()
	context["po_summary"] = svc.getPurchaseOrderSummary()

	// Risk context (financial amounts redacted for non-finance users)
	riskCtx := svc.getRiskContext()
	if !hasFinanceAccess {
		delete(riskCtx, "total_overdue_bhd")
		delete(riskCtx, "total_overdue_ap_bhd")
		delete(riskCtx, "overdue_invoices")
		delete(riskCtx, "overdue_supplier_invoices")
	}
	context["risk_data"] = riskCtx

	// Banking context (cash position, unreconciled lines, outstanding cheques)
	if hasFinanceAccess {
		context["banking_data"] = svc.getBankingContext()
	}

	// Credit notes (outstanding credits against customers)
	if hasFinanceAccess {
		context["credit_notes"] = svc.getCreditNoteContext()
	}

	// Product catalog (always — needed for quoting and procurement discussions)
	context["product_catalog"] = svc.getProductContext()

	// Delivery pipeline (active DNs and recent completions)
	context["delivery_data"] = svc.getDeliveryContext()

	// Forward-looking intelligence (pipeline projections, payment predictions, win rates, trends)
	context["forecast_intelligence"] = svc.getForecastContext()

	// Serial number overview (inventory status and traceability)
	context["serial_inventory"] = svc.getSerialContext()

	// DSO — Days Sales Outstanding per customer (who pays slow/fast)
	if hasFinanceAccess {
		context["dso_analysis"] = svc.getDSOContext()
		context["cashflow_projection"] = svc.host.CashflowProjectionContext()
	}

	// Offer expiry — active quotes approaching their ValidityDate
	context["offer_expiry"] = svc.getOfferExpiryContext()

	// Customer profitability — revenue and margin rankings (finance-gated)
	if hasFinanceAccess {
		context["customer_profitability"] = svc.getCustomerProfitabilityContext()
	}

	// Supplier delivery performance — lead times and QC pass rates
	context["supplier_performance"] = svc.getSupplierPerformanceContext()

	// Customer activity — dormant, new, and lapsed customers
	context["customer_activity"] = svc.getCustomerActivityContext()
	context["service_due_installations"] = svc.getServiceDueInstallationsContext()
	context["closeable_opportunities"] = svc.getCloseableOpportunitiesContext()
	context["quotation_precheck"] = svc.getQuotationPrecheckContext(intent)
	context["work_data"] = svc.host.WorkContext(intent)
	context["database_access"] = svc.getDatabaseAccessContext(intent, hasFinanceAccess)
	context["database_catalog"] = svc.DatabaseInventoryContext(hasFinanceAccess)

	// Action items — prioritized list of things needing attention today
	context["action_items"] = svc.getActionItemsContext()

	// Competition — win/loss analysis, ABB flag, lost reasons, win rate by product
	context["competition_analysis"] = svc.getCompetitionContext()

	if !hasFinanceAccess {
		svc.redactRestrictedBusinessContext(context)
	}

	return context
}

func BuildContextSearchTerms(intent Intent) []string {
	stopWords := map[string]bool{
		"a": true, "about": true, "all": true, "an": true, "and": true, "any": true,
		"are": true, "assigned": true, "at": true, "be": true, "by": true, "can": true,
		"current": true, "currently": true, "customer": true, "data": true, "did": true,
		"do": true, "does": true, "for": true, "from": true, "get": true, "give": true,
		"has": true, "have": true, "how": true, "i": true, "in": true, "into": true,
		"is": true, "it": true, "its": true, "line": true, "me": true, "more": true,
		"my": true, "need": true, "notes": true, "of": true, "on": true, "or": true,
		"our": true, "please": true, "show": true, "statements": true, "tasks": true,
		"tell": true, "that": true, "the": true, "their": true, "them": true, "there": true,
		"these": true, "this": true, "to": true, "us": true, "view": true, "what": true,
		"when": true, "where": true, "which": true, "who": true, "with": true, "you": true,
	}

	raw := strings.NewReplacer("?", " ", ",", " ", ".", " ", ":", " ", ";", " ", "(", " ", ")", " ", "/", " ", "\\", " ", "\n", " ").Replace(strings.ToLower(intent.RawQuery))
	candidates := make([]string, 0, 16)
	candidates = append(candidates, strings.ToLower(strings.TrimSpace(intent.EntityName)))
	candidates = append(candidates, strings.ToLower(strings.TrimSpace(intent.PersonName)))
	for _, token := range strings.Fields(raw) {
		token = strings.TrimSpace(token)
		if len(token) < 3 || stopWords[token] {
			continue
		}
		candidates = append(candidates, token)
	}

	seen := make(map[string]bool)
	terms := make([]string, 0, 10)
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		terms = append(terms, candidate)
		if len(terms) >= 10 {
			break
		}
	}
	return terms
}

func SelectContextEntries(entries []map[string]any, terms []string, limit int) []map[string]any {
	if len(entries) == 0 {
		return []map[string]any{}
	}

	selected := entries
	if len(terms) > 0 {
		matched := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			raw, _ := json.Marshal(entry)
			haystack := strings.ToLower(string(raw))
			for _, term := range terms {
				if term != "" && strings.Contains(haystack, term) {
					matched = append(matched, entry)
					break
				}
			}
		}
		if len(matched) > 0 {
			selected = matched
		}
	}

	if limit > 0 && len(selected) > limit {
		selected = selected[:limit]
	}
	return selected
}

func (svc *Service) getDatabaseAccessContext(intent Intent, hasFinanceAccess bool) map[string]any {
	result := make(map[string]any)
	if svc.db == nil {
		return result
	}

	terms := BuildContextSearchTerms(intent)
	result["search_terms"] = terms
	result["notes_ledger"] = svc.getCrossModuleNotesContext(terms, hasFinanceAccess)
	result["commercial_records"] = svc.getCommercialRecordContext(terms)
	if hasFinanceAccess {
		result["financial_records"] = svc.getFinancialRecordContext(terms)
	}
	return result
}

func (svc *Service) DatabaseInventoryContext(hasFinanceAccess bool) map[string]any {
	result := map[string]any{
		"query_policy":        "All answers must use controlled ERP queries and RBAC. Finance, banking, payroll, payments, and invoice amounts require finance:view.",
		"finance_data_access": hasFinanceAccess,
	}
	if svc.db == nil {
		return result
	}

	tables, err := svc.db.Migrator().GetTables()
	if err != nil {
		result["error"] = err.Error()
		return result
	}
	sort.Strings(tables)

	financeTerms := []string{"invoice", "payment", "bank", "cash", "cheque", "expense", "payroll", "salary", "journal", "account", "vat", "credit_note", "supplier_invoice"}
	tableEntries := make([]map[string]any, 0, len(tables))
	moduleCounts := make(map[string]int)
	for _, table := range tables {
		if strings.HasPrefix(table, "sqlite_") {
			continue
		}
		module := classifyDatabaseTableModule(table)
		moduleCounts[module]++
		row := map[string]any{
			"table":  table,
			"module": module,
		}
		if hasFinanceAccess || !databaseTableLooksFinancial(table, financeTerms) {
			var count int64
			if err := svc.db.Table(table).Count(&count).Error; err == nil {
				row["rows"] = count
			} else {
				row["count_error"] = err.Error()
			}
			row["access"] = "queryable"
		} else {
			row["access"] = "finance_view_required"
		}
		tableEntries = append(tableEntries, row)
	}

	result["total_tables"] = len(tableEntries)
	result["modules"] = moduleCounts
	result["tables"] = tableEntries
	result["controlled_query_surfaces"] = []string{
		"customers", "suppliers", "contacts", "orders", "order_items", "offers", "offer_items", "opportunities", "rfqs",
		"costing_sheets", "purchase_orders", "goods_received_notes", "delivery_notes", "serial_numbers", "products",
		"invoices", "payments", "supplier_invoices", "supplier_payments", "bank_statements", "expenses", "payroll",
		"employees", "tasks", "comments", "notifications", "notes", "audit_logs",
	}
	return result
}

func databaseTableLooksFinancial(table string, financeTerms []string) bool {
	lower := strings.ToLower(table)
	for _, term := range financeTerms {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

func classifyDatabaseTableModule(table string) string {
	lower := strings.ToLower(table)
	switch {
	case strings.Contains(lower, "customer") || strings.Contains(lower, "supplier") || strings.Contains(lower, "contact") || strings.Contains(lower, "entity_note"):
		return "relationships"
	case strings.Contains(lower, "offer") || strings.Contains(lower, "opportunit") || strings.Contains(lower, "rfq") || strings.Contains(lower, "costing"):
		return "commercial"
	case strings.Contains(lower, "order") || strings.Contains(lower, "delivery") || strings.Contains(lower, "grn") || strings.Contains(lower, "serial") || strings.Contains(lower, "purchase"):
		return "operations"
	case strings.Contains(lower, "invoice") || strings.Contains(lower, "payment") || strings.Contains(lower, "bank") || strings.Contains(lower, "expense") || strings.Contains(lower, "payroll") || strings.Contains(lower, "cheque") || strings.Contains(lower, "journal") || strings.Contains(lower, "vat"):
		return "finance"
	case strings.Contains(lower, "employee") || strings.Contains(lower, "task") || strings.Contains(lower, "project") || strings.Contains(lower, "notification"):
		return "work"
	case strings.Contains(lower, "conversation") || strings.Contains(lower, "chat") || strings.Contains(lower, "prediction") || strings.Contains(lower, "memory"):
		return "intelligence"
	case strings.Contains(lower, "license") || strings.Contains(lower, "role") || strings.Contains(lower, "user") || strings.Contains(lower, "device") || strings.Contains(lower, "sync") || strings.Contains(lower, "setting") || strings.Contains(lower, "audit"):
		return "system"
	default:
		return "other"
	}
}

func (svc *Service) getCrossModuleNotesContext(terms []string, hasFinanceAccess bool) map[string]any {
	result := make(map[string]any)
	if svc.db == nil {
		return result
	}

	var entityNotes []EntityNote
	_ = svc.db.Order("created_at DESC").Limit(30).Find(&entityNotes).Error
	entityEntries := make([]map[string]any, 0, len(entityNotes))
	for _, note := range entityNotes {
		entityEntries = append(entityEntries, map[string]any{
			"module":      "entity_notes",
			"entity_type": note.EntityType,
			"entity_id":   note.EntityID,
			"note_type":   note.NoteType,
			"content":     note.Content,
			"created_at":  note.CreatedAt.Format("2006-01-02"),
		})
	}
	result["entity_notes"] = SelectContextEntries(entityEntries, terms, 10)

	var offerNotes []OfferNote
	_ = svc.db.Order("note_date DESC").Limit(20).Find(&offerNotes).Error
	offerNoteEntries := make([]map[string]any, 0, len(offerNotes))
	for _, note := range offerNotes {
		var offer Offer
		_ = svc.db.Select("offer_number, customer_name, stage").Where("id = ?", note.OfferID).First(&offer).Error
		offerNoteEntries = append(offerNoteEntries, map[string]any{
			"module":       "offer_notes",
			"offer_id":     note.OfferID,
			"offer_number": offer.OfferNumber,
			"customer":     offer.CustomerName,
			"stage":        offer.Stage,
			"content":      note.Content,
			"note_date":    note.NoteDate.Format("2006-01-02"),
		})
	}
	result["offer_notes"] = SelectContextEntries(offerNoteEntries, terms, 8)

	var statements []BankStatement
	_ = svc.db.Where("TRIM(COALESCE(notes,'')) != ''").Order("statement_date DESC").Limit(20).Find(&statements).Error
	statementEntries := make([]map[string]any, 0, len(statements))
	for _, statement := range statements {
		statementEntries = append(statementEntries, map[string]any{
			"module":           "bank_statements",
			"statement_id":     statement.ID,
			"statement_number": statement.StatementNumber,
			"statement_date":   statement.StatementDate.Format("2006-01-02"),
			"division":         statement.Division,
			"status":           statement.Status,
			"notes":            statement.Notes,
		})
	}
	result["bank_statement_notes"] = SelectContextEntries(statementEntries, terms, 8)

	if hasFinanceAccess {
		var recons []BookBankReconciliation
		_ = svc.db.Where("TRIM(COALESCE(notes,'')) != ''").Order("reconciliation_date DESC").Limit(15).Find(&recons).Error
		reconEntries := make([]map[string]any, 0, len(recons))
		for _, recon := range recons {
			reconEntries = append(reconEntries, map[string]any{
				"module":              "book_bank_reconciliations",
				"reconciliation_id":   recon.ID,
				"reconciliation_date": recon.ReconciliationDate.Format("2006-01-02"),
				"difference":          recon.Difference,
				"notes":               recon.Notes,
			})
		}
		result["book_bank_notes"] = SelectContextEntries(reconEntries, terms, 6)

		var expenses []ExpenseEntry
		_ = svc.db.Where("TRIM(COALESCE(notes,'')) != ''").Order("expense_date DESC").Limit(20).Find(&expenses).Error
		expenseEntries := make([]map[string]any, 0, len(expenses))
		for _, expense := range expenses {
			expenseEntries = append(expenseEntries, map[string]any{
				"module":       "expense_entries",
				"entry_number": expense.EntryNumber,
				"division":     expense.Division,
				"expense_date": expense.ExpenseDate.Format("2006-01-02"),
				"description":  expense.Description,
				"total_amount": expense.TotalAmount,
				"notes":        expense.Notes,
			})
		}
		result["expense_notes"] = SelectContextEntries(expenseEntries, terms, 8)

		var runs []PayrollRun
		_ = svc.db.Where("TRIM(COALESCE(notes,'')) != ''").Order("created_at DESC").Limit(10).Find(&runs).Error
		payrollEntries := make([]map[string]any, 0, len(runs))
		for _, run := range runs {
			payrollEntries = append(payrollEntries, map[string]any{
				"module":       "payroll_runs",
				"run_number":   run.RunNumber,
				"division":     run.Division,
				"status":       run.Status,
				"net_total":    run.NetTotal,
				"notes":        run.Notes,
				"generated_at": FormatOptionalDate(run.GeneratedAt),
			})
		}
		result["payroll_notes"] = SelectContextEntries(payrollEntries, terms, 5)
	}

	return result
}

func (svc *Service) getCommercialRecordContext(terms []string) map[string]any {
	result := make(map[string]any)
	if svc.db == nil {
		return result
	}

	var offers []Offer
	_ = svc.db.Order("quotation_date DESC").Limit(15).Find(&offers).Error
	offerEntries := make([]map[string]any, 0, len(offers))
	for _, offer := range offers {
		entry := map[string]any{
			"offer_id":      offer.ID,
			"offer_number":  offer.OfferNumber,
			"customer":      offer.CustomerName,
			"stage":         offer.Stage,
			"date":          offer.QuotationDate.Format("2006-01-02"),
			"validity_date": offer.ValidityDate.Format("2006-01-02"),
			"value_bhd":     offer.TotalValueBHD,
			"division":      offer.Division,
			"quote_type":    offer.QuoteType,
		}
		var items []OfferItem
		_ = svc.db.Where("offer_id = ?", offer.ID).Order("line_number ASC").Find(&items).Error
		lineItems := make([]map[string]any, 0, len(items))
		for _, item := range items {
			lineItems = append(lineItems, map[string]any{
				"line_number":     item.LineNumber,
				"description":     item.Description,
				"equipment":       item.Equipment,
				"model":           item.Model,
				"quantity":        item.Quantity,
				"unit_price_bhd":  item.UnitPrice,
				"total_price_bhd": item.TotalPrice,
				"margin_pct":      item.MarginPercent,
			})
		}
		entry["line_items"] = lineItems
		offerEntries = append(offerEntries, entry)
	}
	result["offers"] = SelectContextEntries(offerEntries, terms, 8)

	var orders []Order
	_ = svc.db.Order("order_date DESC").Limit(15).Find(&orders).Error
	orderEntries := make([]map[string]any, 0, len(orders))
	for _, order := range orders {
		entry := map[string]any{
			"order_id":      order.ID,
			"order_number":  order.OrderNumber,
			"customer":      order.CustomerName,
			"status":        order.Status,
			"date":          order.OrderDate.Format("2006-01-02"),
			"value_bhd":     order.GrandTotalBHD,
			"division":      order.Division,
			"offer_number":  order.OfferNumber,
			"customer_po":   order.CustomerPONumber,
			"payment_terms": order.PaymentTerms,
		}
		var items []OrderItem
		_ = svc.db.Where("order_id = ?", order.ID).Order("line_number ASC").Find(&items).Error
		lineItems := make([]map[string]any, 0, len(items))
		for _, item := range items {
			lineItems = append(lineItems, map[string]any{
				"line_number":     item.LineNumber,
				"description":     item.Description,
				"equipment":       item.Equipment,
				"model":           item.Model,
				"quantity":        item.Quantity,
				"unit_price_bhd":  item.UnitPrice,
				"total_price_bhd": item.TotalPrice,
				"margin_pct":      item.MarginPercent,
			})
		}
		entry["line_items"] = lineItems
		orderEntries = append(orderEntries, entry)
	}
	result["orders"] = SelectContextEntries(orderEntries, terms, 8)

	var invoices []Invoice
	_ = svc.db.Order("invoice_date DESC").Limit(15).Find(&invoices).Error
	invoiceEntries := make([]map[string]any, 0, len(invoices))
	for _, invoice := range invoices {
		entry := map[string]any{
			"invoice_id":      invoice.ID,
			"invoice_number":  invoice.InvoiceNumber,
			"customer":        invoice.CustomerName,
			"status":          invoice.Status,
			"invoice_date":    invoice.InvoiceDate.Format("2006-01-02"),
			"due_date":        invoice.DueDate.Format("2006-01-02"),
			"grand_total_bhd": invoice.GrandTotalBHD,
			"outstanding_bhd": invoice.OutstandingBHD,
			"division":        invoice.Division,
			"offer_number":    invoice.OfferNumber,
		}
		var items []DBInvoiceItem
		_ = svc.db.Where("invoice_id = ?", invoice.ID).Order("line_number ASC").Find(&items).Error
		lineItems := make([]map[string]any, 0, len(items))
		for _, item := range items {
			lineItems = append(lineItems, map[string]any{
				"line_number":     item.LineNumber,
				"description":     item.Description,
				"equipment":       item.Equipment,
				"model":           item.Model,
				"quantity":        item.Quantity,
				"unit_price_bhd":  item.Rate,
				"total_price_bhd": item.TotalBHD,
				"margin_pct":      item.MarginPercent,
			})
		}
		entry["line_items"] = lineItems
		invoiceEntries = append(invoiceEntries, entry)
	}
	result["customer_invoices"] = SelectContextEntries(invoiceEntries, terms, 8)

	var supplierInvoices []SupplierInvoice
	_ = svc.db.Order("invoice_date DESC").Limit(15).Find(&supplierInvoices).Error
	supplierInvoiceEntries := make([]map[string]any, 0, len(supplierInvoices))
	for _, invoice := range supplierInvoices {
		entry := map[string]any{
			"supplier_invoice_id": invoice.ID,
			"invoice_number":      invoice.InvoiceNumber,
			"supplier":            invoice.SupplierName,
			"status":              invoice.Status,
			"invoice_date":        invoice.InvoiceDate.Format("2006-01-02"),
			"due_date":            invoice.DueDate.Format("2006-01-02"),
			"total_bhd":           invoice.TotalBHD,
			"payment_status":      invoice.PaymentStatus,
			"division":            invoice.Division,
			"po_number":           invoice.PONumber,
		}
		var items []SupplierInvoiceItem
		_ = svc.db.Where("supplier_invoice_id = ?", invoice.ID).Order("line_number ASC").Find(&items).Error
		lineItems := make([]map[string]any, 0, len(items))
		for _, item := range items {
			lineItems = append(lineItems, map[string]any{
				"line_number": item.LineNumber,
				"description": item.Description,
				"quantity":    item.Quantity,
				"unit_price":  item.UnitPrice,
				"total_price": item.TotalPrice,
				"currency":    item.Currency,
			})
		}
		entry["line_items"] = lineItems
		supplierInvoiceEntries = append(supplierInvoiceEntries, entry)
	}
	result["supplier_invoices"] = SelectContextEntries(supplierInvoiceEntries, terms, 8)

	var opportunities []Opportunity
	_ = svc.db.Order("offer_date DESC").Limit(15).Find(&opportunities).Error
	opportunityEntries := make([]map[string]any, 0, len(opportunities))
	for _, opportunity := range opportunities {
		opportunityEntries = append(opportunityEntries, map[string]any{
			"opportunity_id": opportunity.ID,
			"folder_number":  opportunity.FolderNumber,
			"title":          FirstNonEmpty(opportunity.Title, opportunity.FolderName),
			"customer":       opportunity.CustomerName,
			"stage":          opportunity.Stage,
			"salesperson":    opportunity.Salesperson,
			"revenue_bhd":    opportunity.RevenueBHD,
			"comment":        opportunity.Comment,
			"owner_notes":    opportunity.OwnerNotes,
			"division":       opportunity.Division,
		})
	}
	result["opportunities"] = SelectContextEntries(opportunityEntries, terms, 8)

	var followUps []OfferFollowUp
	_ = svc.db.Order("follow_up_date DESC").Limit(20).Find(&followUps).Error
	followUpEntries := make([]map[string]any, 0, len(followUps))
	for _, followUp := range followUps {
		var offer Offer
		_ = svc.db.Select("offer_number, customer_name").Where("id = ?", followUp.OfferID).First(&offer).Error
		followUpEntries = append(followUpEntries, map[string]any{
			"offer_id":       followUp.OfferID,
			"offer_number":   offer.OfferNumber,
			"customer":       offer.CustomerName,
			"follow_up_date": followUp.FollowUpDate.Format("2006-01-02"),
			"status":         followUp.Status,
			"notes":          followUp.Notes,
		})
	}
	result["offer_follow_ups"] = SelectContextEntries(followUpEntries, terms, 8)

	return result
}

func (svc *Service) getFinancialRecordContext(terms []string) map[string]any {
	result := make(map[string]any)
	if svc.db == nil {
		return result
	}

	var payments []Payment
	_ = svc.db.Order("payment_date DESC").Limit(20).Find(&payments).Error
	paymentEntries := make([]map[string]any, 0, len(payments))
	for _, payment := range payments {
		var invoice Invoice
		_ = svc.db.Select("customer_name").Where("id = ?", payment.InvoiceID).First(&invoice).Error
		paymentEntries = append(paymentEntries, map[string]any{
			"payment_id":      payment.ID,
			"invoice_number":  payment.InvoiceNumber,
			"customer":        invoice.CustomerName,
			"amount_bhd":      payment.AmountBHD,
			"payment_date":    payment.PaymentDate.Format("2006-01-02"),
			"payment_method":  payment.PaymentMethod,
			"reference":       payment.Reference,
			"days_to_payment": payment.DaysToPayment,
			"division":        payment.Division,
		})
	}
	result["customer_payments"] = SelectContextEntries(paymentEntries, terms, 10)

	var supplierPayments []SupplierPayment
	_ = svc.db.Order("payment_date DESC").Limit(20).Find(&supplierPayments).Error
	supplierPaymentEntries := make([]map[string]any, 0, len(supplierPayments))
	for _, payment := range supplierPayments {
		supplierPaymentEntries = append(supplierPaymentEntries, map[string]any{
			"supplier_payment_id": payment.ID,
			"supplier":            payment.SupplierName,
			"invoice_number":      payment.InvoiceNumber,
			"amount_bhd":          payment.AmountBHD,
			"payment_date":        payment.PaymentDate.Format("2006-01-02"),
			"payment_method":      payment.PaymentMethod,
			"reference":           payment.Reference,
			"currency":            payment.Currency,
			"division":            payment.Division,
			"notes":               payment.Notes,
		})
	}
	result["supplier_payments"] = SelectContextEntries(supplierPaymentEntries, terms, 10)

	var statements []BankStatement
	_ = svc.db.Order("statement_date DESC").Limit(10).Find(&statements).Error
	statementEntries := make([]map[string]any, 0, len(statements))
	for _, statement := range statements {
		entry := map[string]any{
			"statement_id":     statement.ID,
			"statement_number": statement.StatementNumber,
			"statement_date":   statement.StatementDate.Format("2006-01-02"),
			"status":           statement.Status,
			"division":         statement.Division,
			"opening_balance":  statement.OpeningBalance,
			"closing_balance":  statement.ClosingBalance,
			"total_debits":     statement.TotalDebits,
			"total_credits":    statement.TotalCredits,
			"notes":            statement.Notes,
		}
		var lines []BankStatementLine
		_ = svc.db.Where("bank_statement_id = ?", statement.ID).Order("transaction_date DESC, line_number DESC").Limit(5).Find(&lines).Error
		lineEntries := make([]map[string]any, 0, len(lines))
		for _, line := range lines {
			lineEntries = append(lineEntries, map[string]any{
				"line_number":        line.LineNumber,
				"transaction_date":   line.TransactionDate.Format("2006-01-02"),
				"description":        line.Description,
				"reference":          line.Reference,
				"debit":              line.Debit,
				"credit":             line.Credit,
				"is_matched":         line.IsMatched,
				"transaction_type":   line.TransactionType,
				"extracted_customer": line.ExtractedCustomer,
				"extracted_supplier": line.ExtractedSupplier,
				"notes":              line.Notes,
			})
		}
		entry["lines"] = lineEntries
		statementEntries = append(statementEntries, entry)
	}
	result["bank_statements"] = SelectContextEntries(statementEntries, terms, 6)

	var expenses []ExpenseEntry
	_ = svc.db.Order("expense_date DESC").Limit(15).Find(&expenses).Error
	expenseEntries := make([]map[string]any, 0, len(expenses))
	for _, expense := range expenses {
		expenseEntries = append(expenseEntries, map[string]any{
			"expense_id":     expense.ID,
			"entry_number":   expense.EntryNumber,
			"division":       expense.Division,
			"expense_date":   expense.ExpenseDate.Format("2006-01-02"),
			"description":    expense.Description,
			"status":         expense.Status,
			"payment_status": expense.PaymentStatus,
			"total_amount":   expense.TotalAmount,
			"payment_method": expense.PaymentMethod,
			"payment_ref":    expense.PaymentReference,
			"notes":          expense.Notes,
		})
	}
	result["expenses"] = SelectContextEntries(expenseEntries, terms, 8)

	return result
}

func FirstNonEmptyStringPointer(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func (svc *Service) getServiceDueInstallationsContext() map[string]any {
	result := make(map[string]any)
	if svc.db == nil {
		return result
	}

	type serviceCandidate struct {
		SerialID           string
		SerialNo           string
		ProductID          string
		ProductCode        string
		ProductName        string
		CustomerID         string
		CustomerName       string
		DeliveryAddress    string
		InvoiceID          string
		InvoiceNumber      string
		InvoiceDate        *time.Time
		LineValueBHD       float64
		OEMName            string
		WarrantyEndDate    *time.Time
		CalibrationDueDate *time.Time
		WarrantyStartDate  *time.Time
	}

	var rows []serviceCandidate
	svc.db.Table("serial_numbers").
		Select(`serial_numbers.id as serial_id, serial_numbers.serial_no, serial_numbers.product_id, serial_numbers.product_code,
			products.product_name, serial_numbers.customer_id, serial_numbers.customer_name,
			delivery_notes.delivery_address, serial_numbers.invoice_id, serial_numbers.invoice_number,
			invoices.invoice_date, COALESCE(invoice_items.total_bhd, invoices.grand_total_bhd, 0) as line_value_bhd,
			suppliers.supplier_name as oem_name, serial_numbers.warranty_end_date, serial_numbers.calibration_due_date,
			serial_numbers.warranty_start_date`).
		Joins("LEFT JOIN products ON products.id = serial_numbers.product_id").
		Joins("LEFT JOIN suppliers ON suppliers.id = products.supplier_id").
		Joins("LEFT JOIN invoices ON invoices.id = serial_numbers.invoice_id").
		Joins("LEFT JOIN invoice_items ON invoice_items.invoice_id = serial_numbers.invoice_id AND invoice_items.product_id = serial_numbers.product_id").
		Joins("LEFT JOIN delivery_notes ON delivery_notes.id = invoices.delivery_note_id").
		Where("serial_numbers.status = ?", "Delivered").
		Order("COALESCE(invoice_items.total_bhd, invoices.grand_total_bhd, 0) DESC, invoices.invoice_date DESC").
		Limit(50).
		Scan(&rows)

	now := time.Now()
	dueSoon := now.AddDate(0, 0, 45)
	top := make([]map[string]any, 0, 5)
	seen := make(map[string]bool)

	for _, row := range rows {
		serviceDate, dueReason, isDue := deriveServiceDueStatus(now, dueSoon, row.InvoiceDate, row.WarrantyStartDate, row.WarrantyEndDate, row.CalibrationDueDate)
		if !isDue {
			continue
		}

		groupKey := FirstNonEmpty(row.InvoiceID+"|"+row.ProductID, row.InvoiceNumber+"|"+row.ProductCode, row.SerialID)
		if groupKey == "" || seen[groupKey] {
			continue
		}
		seen[groupKey] = true

		top = append(top, map[string]any{
			"customer_name":       row.CustomerName,
			"customer_id":         row.CustomerID,
			"site":                row.DeliveryAddress,
			"equipment":           FirstNonEmpty(row.ProductName, row.ProductCode),
			"product_code":        row.ProductCode,
			"serial_no":           row.SerialNo,
			"oem":                 row.OEMName,
			"invoice_number":      row.InvoiceNumber,
			"invoice_date":        FormatOptionalDate(row.InvoiceDate),
			"estimated_value_bhd": row.LineValueBHD,
			"service_due_date":    serviceDate,
			"due_reason":          dueReason,
		})
		if len(top) >= 5 {
			break
		}
	}

	result["top_due_installations"] = top
	result["field_follow_up_candidates"] = buildFieldFollowUpCandidates(top)
	result["basis"] = "Delivered serial-tracked installations due by calibration date, warranty milestone, or age-from-delivery heuristic"
	result["window_days"] = 45
	return result
}

func deriveServiceDueStatus(now, dueSoon time.Time, invoiceDate, warrantyStartDate, warrantyEndDate, calibrationDueDate *time.Time) (string, string, bool) {
	if calibrationDueDate != nil && !calibrationDueDate.After(dueSoon) {
		return calibrationDueDate.Format("2006-01-02"), "calibration due", true
	}
	if warrantyEndDate != nil && !warrantyEndDate.After(dueSoon) {
		return warrantyEndDate.Format("2006-01-02"), "warranty milestone due", true
	}
	baseDate := invoiceDate
	if warrantyStartDate != nil {
		baseDate = warrantyStartDate
	}
	if baseDate != nil {
		heuristicDue := baseDate.AddDate(1, 0, 0)
		if !heuristicDue.After(dueSoon) {
			return heuristicDue.Format("2006-01-02"), "annual service heuristic", true
		}
	}
	return "", "", false
}

func (svc *Service) getCloseableOpportunitiesContext() map[string]any {
	result := make(map[string]any)
	if svc.db == nil {
		return result
	}

	type closeableCandidate struct {
		OpportunityID    string
		FolderNumber     string
		Title            string
		CustomerID       string
		CustomerName     string
		Stage            string
		Salesperson      string
		OfferID          string
		OfferNumber      string
		RevenueBHD       float64
		InvoiceNumber    string
		OutstandingBHD   float64
		PaymentDate      *time.Time
		PaymentAmountBHD float64
	}

	var rows []closeableCandidate
	svc.db.Table("opportunities").
		Select(`opportunities.id as opportunity_id, opportunities.folder_number, opportunities.title,
			opportunities.customer_id, opportunities.customer_name, opportunities.stage, opportunities.salesperson,
			opportunities.offer_id, invoices.offer_number, opportunities.revenue_bhd,
			invoices.invoice_number, invoices.outstanding_bhd, payments.payment_date, payments.amount_bhd as payment_amount_bhd`).
		Joins("LEFT JOIN invoices ON invoices.offer_id = opportunities.offer_id").
		Joins("LEFT JOIN payments ON payments.invoice_id = invoices.id").
		Where("opportunities.stage NOT IN ?", []string{"Won", "Lost"}).
		Where("opportunities.offer_id <> ''").
		Where("invoices.id IS NOT NULL").
		Where("invoices.outstanding_bhd <= ?", 0.001).
		Order("payments.payment_date DESC, invoices.invoice_date DESC").
		Limit(25).
		Scan(&rows)

	candidates := make([]map[string]any, 0, len(rows))
	seen := make(map[string]bool)
	for _, row := range rows {
		if seen[row.OpportunityID] {
			continue
		}
		seen[row.OpportunityID] = true
		candidates = append(candidates, map[string]any{
			"opportunity_id":        row.OpportunityID,
			"folder_number":         row.FolderNumber,
			"title":                 FirstNonEmpty(row.Title, row.OfferNumber, row.InvoiceNumber),
			"customer_id":           row.CustomerID,
			"customer_name":         row.CustomerName,
			"current_stage":         row.Stage,
			"salesperson":           row.Salesperson,
			"offer_id":              row.OfferID,
			"offer_number":          row.OfferNumber,
			"invoiced_reference":    row.InvoiceNumber,
			"payment_received_date": FormatOptionalDate(row.PaymentDate),
			"payment_amount_bhd":    row.PaymentAmountBHD,
			"expected_revenue_bhd":  row.RevenueBHD,
			"close_readiness":       "Fully paid invoice linked to opportunity offer",
			"evidence_trail": []map[string]any{
				{"step": "offer_linked", "value": row.OfferID != "", "detail": row.OfferNumber},
				{"step": "invoice_linked", "value": row.InvoiceNumber != "", "detail": row.InvoiceNumber},
				{"step": "invoice_paid", "value": true, "detail": fmt.Sprintf("Outstanding %.3f BHD", row.OutstandingBHD)},
				{"step": "payment_recorded", "value": row.PaymentDate != nil, "detail": FormatOptionalDate(row.PaymentDate)},
			},
		})
	}

	result["candidates"] = candidates
	result["rule"] = "Candidate when linked invoice exists against the opportunity offer and outstanding amount is effectively zero"
	return result
}

func (svc *Service) getInstalledBaseSummary(customerName string) map[string]any {
	result := make(map[string]any)
	if svc.db == nil || strings.TrimSpace(customerName) == "" {
		return result
	}

	escaped := text.EscapeLike(customerName)
	type installedBaseRow struct {
		CustomerName       string
		ProductCode        string
		ProductName        string
		OEMName            string
		SerialNo           string
		InvoiceNumber      string
		InvoiceDate        *time.Time
		DeliveryAddress    string
		WarrantyEndDate    *time.Time
		CalibrationDueDate *time.Time
	}

	var rows []installedBaseRow
	svc.db.Table("serial_numbers").
		Select(`serial_numbers.customer_name, serial_numbers.product_code, products.product_name, suppliers.supplier_name as oem_name,
			serial_numbers.serial_no, serial_numbers.invoice_number, invoices.invoice_date, delivery_notes.delivery_address,
			serial_numbers.warranty_end_date, serial_numbers.calibration_due_date`).
		Joins("LEFT JOIN products ON products.id = serial_numbers.product_id").
		Joins("LEFT JOIN suppliers ON suppliers.id = products.supplier_id").
		Joins("LEFT JOIN invoices ON invoices.id = serial_numbers.invoice_id").
		Joins("LEFT JOIN delivery_notes ON delivery_notes.id = invoices.delivery_note_id").
		Where("serial_numbers.customer_name LIKE ? ESCAPE '\\'", "%"+escaped+"%").
		Order("invoices.invoice_date DESC").
		Limit(25).
		Scan(&rows)

	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]any{
			"customer_name":        row.CustomerName,
			"equipment":            FirstNonEmpty(row.ProductName, row.ProductCode),
			"product_code":         row.ProductCode,
			"oem":                  row.OEMName,
			"serial_no":            row.SerialNo,
			"invoice_number":       row.InvoiceNumber,
			"invoice_date":         FormatOptionalDate(row.InvoiceDate),
			"site":                 row.DeliveryAddress,
			"warranty_end_date":    FormatOptionalDate(row.WarrantyEndDate),
			"calibration_due_date": FormatOptionalDate(row.CalibrationDueDate),
		})
	}

	result["customer_name"] = customerName
	result["serial_tracked_items"] = items
	result["installed_base_count"] = len(items)
	return result
}

func (svc *Service) getButlerMemoryContext(intent Intent) map[string]any {
	ctx := make(map[string]any)
	if svc.db == nil {
		return ctx
	}

	resolution := svc.ResolveBestEntityReference(intent)
	if resolution != nil {
		switch resolution.EntityType {
		case "customer":
			ctx["entity_notes"] = svc.getEntityNotesForMemory("customer", resolution.EntityID)
			ctx["follow_ups"] = svc.getFollowUpsForMemory(resolution.EntityID)
		case "customer_contact":
			ctx["entity_notes"] = svc.getEntityNotesForMemory("customer", resolution.RelatedCustomerID)
			ctx["follow_ups"] = svc.getFollowUpsForMemory(resolution.RelatedCustomerID)
		}
	}

	ctx["quick_memory"] = svc.host.RecentOpenQuickCaptures()
	return ctx
}

func (svc *Service) getEntityNotesForMemory(entityType, entityID string) []map[string]any {
	if strings.TrimSpace(entityID) == "" {
		return []map[string]any{}
	}

	var notes []EntityNote
	svc.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Limit(5).
		Find(&notes)
	if entityType == "customer" && len(notes) == 0 {
		var customer CustomerMaster
		if err := svc.db.Where("customer_id = ?", entityID).First(&customer).Error; err == nil && strings.TrimSpace(customer.ID) != "" {
			svc.db.Where("entity_type = ? AND entity_id = ?", entityType, customer.ID).
				Order("created_at DESC").
				Limit(5).
				Find(&notes)
		}
	}

	memories := make([]map[string]any, 0, len(notes))
	for _, note := range notes {
		memories = append(memories, map[string]any{
			"type":       note.NoteType,
			"content":    note.Content,
			"created_at": note.CreatedAt.Format("2006-01-02"),
		})
	}
	return memories
}

func (svc *Service) getFollowUpsForMemory(customerID string) []map[string]any {
	if strings.TrimSpace(customerID) == "" {
		return []map[string]any{}
	}

	var tasks []FollowUpTask
	svc.db.Where("customer_id = ? AND status IN ?", customerID, []string{"pending", "in_progress", "overdue"}).
		Order("due_date ASC").
		Limit(5).
		Find(&tasks)

	memories := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		memories = append(memories, map[string]any{
			"title":       task.Title,
			"status":      task.Status,
			"priority":    task.Priority,
			"due_date":    task.DueDate.Format("2006-01-02"),
			"description": task.Description,
		})
	}
	return memories
}

type contactMatch struct {
	CustomerContact
	BusinessName string
	BusinessID   string
}

func (svc *Service) getProactiveBriefingContext(hasFinanceAccess bool) map[string]any {
	ctx := make(map[string]any)
	actionItems := svc.getActionItemsContext()
	riskData := svc.getRiskContext()
	serviceDue := svc.getServiceDueInstallationsContext()

	ctx["action_items"] = actionItems
	ctx["risk_summary"] = riskData
	ctx["service_due"] = serviceDue

	if hasFinanceAccess {
		ctx["cashflow_projection"] = svc.host.CashflowProjectionContext()
		ctx["ar_summary"] = svc.getARSummary()
	}

	return ctx
}

func (svc *Service) getQuotationPrecheckContext(intent Intent) map[string]any {
	result := make(map[string]any)
	resolution := svc.ResolveBestEntityReference(intent)
	if resolution == nil {
		result["status"] = "needs_customer_resolution"
		result["missing_fields"] = []string{"customer"}
		result["draft_clues"] = extractQuotationDraftClues(intent.RawQuery)
		return result
	}

	customerName := resolution.DisplayName
	if resolution.EntityType == "customer_contact" && resolution.RelatedCustomer != "" {
		customerName = resolution.RelatedCustomer
	}

	customerCtx := svc.getCustomerContext(customerName)
	customer, _ := customerCtx["customer"].(map[string]any)
	contacts, _ := customerCtx["contacts"].([]map[string]any)
	offers, _ := customerCtx["active_offers"].([]map[string]any)

	var primaryContact map[string]any
	for _, contact := range contacts {
		if isPrimary, _ := contact["is_primary"].(bool); isPrimary {
			primaryContact = contact
			break
		}
	}
	if primaryContact == nil && len(contacts) > 0 {
		primaryContact = contacts[0]
	}

	missing := make([]string, 0)
	if customerName == "" {
		missing = append(missing, "customer")
	}
	if len(primaryContact) == 0 {
		missing = append(missing, "contact_person")
	}
	paymentTerms, _ := customer["payment_terms_days"].(int)
	if paymentTerms == 0 {
		missing = append(missing, "payment_terms")
	}

	result["status"] = "ready"
	if len(missing) > 0 {
		result["status"] = "needs_inputs"
	}
	if resolution.Ambiguous {
		result["status"] = "needs_clarification"
	}
	result["resolved_customer"] = customer
	result["primary_contact"] = primaryContact
	result["recent_active_offers"] = offers
	result["missing_fields"] = missing
	result["draft_clues"] = extractQuotationDraftClues(intent.RawQuery)
	result["ambiguity_summary"] = buildAmbiguitySummary(resolution)
	result["draft_payload"] = buildQuotationDraftPayload(customer, primaryContact, extractQuotationDraftClues(intent.RawQuery), missing, resolution)
	result["default_guidance"] = map[string]any{
		"vat_percent": 10,
		"quote_types": []string{"Quotation", "Budgetary Quote", "Estimate", "Technical", "Commercial"},
	}
	return result
}

func (svc *Service) injectGroundedReferenceContext(context map[string]any, intent Intent, hasFinanceAccess bool) {
	resolution := svc.ResolveBestEntityReference(intent)
	if resolution == nil {
		return
	}

	context["entity_resolution"] = map[string]any{
		"reference_kind":      intent.ReferenceKind,
		"entity_type":         resolution.EntityType,
		"entity_id":           resolution.EntityID,
		"display_name":        resolution.DisplayName,
		"confidence":          resolution.Confidence,
		"match_reason":        resolution.MatchReason,
		"related_customer":    resolution.RelatedCustomer,
		"related_customer_id": resolution.RelatedCustomerID,
		"ambiguous":           resolution.Ambiguous,
		"alternatives":        resolution.Alternatives,
	}
	context["clarification_prompt"] = buildAmbiguitySummary(resolution)

	switch resolution.EntityType {
	case "customer":
		customerCtx := svc.getCustomerContext(resolution.DisplayName)
		if !hasFinanceAccess {
			svc.redactFinancialFromContext(customerCtx)
		}
		context["customer_data"] = customerCtx
		context["customer_360"] = customerCtx
		context["installed_base_summary"] = svc.getInstalledBaseSummary(resolution.DisplayName)
	case "customer_contact":
		contactCtx := svc.getCustomerContactContext(resolution)
		if !hasFinanceAccess {
			delete(contactCtx, "linked_customer_financials")
		}
		context["contact_context"] = contactCtx
		if resolution.RelatedCustomer != "" {
			customerCtx := svc.getCustomerContext(resolution.RelatedCustomer)
			if !hasFinanceAccess {
				svc.redactFinancialFromContext(customerCtx)
			}
			context["customer_data"] = customerCtx
			context["customer_360"] = customerCtx
			context["installed_base_summary"] = svc.getInstalledBaseSummary(resolution.RelatedCustomer)
		}
	case "user":
		context["account_manager_context"] = svc.getAccountManagerContext(resolution)
	case "supplier":
		context["supplier_data"] = svc.getSupplierContext(resolution.DisplayName)
	case "employee":
		context["employee_context"] = svc.host.EmployeeContext(resolution)
	}

	context["action_items"] = svc.getActionItemsContext()
	context["competition_intelligence"] = svc.getCompetitionContext()
}

func (svc *Service) ResolveBestEntityReference(intent Intent) *ButlerResolvedEntity {
	if svc.db == nil {
		return nil
	}

	reference := strings.TrimSpace(intent.EntityName)
	if reference == "" {
		reference = strings.TrimSpace(intent.PersonName)
	}
	if reference == "" {
		return nil
	}

	if intent.ReferenceKind == "account_manager" {
		if user := svc.resolveUserReference(reference); user != nil {
			return user
		}
	}

	if intent.ReferenceKind == "employee" || intent.Domain == "work" {
		if employee := svc.host.ResolveEmployeeReference(reference); employee != nil {
			return employee
		}
	}

	if intent.ReferenceKind == "supplier" || intent.Domain == "supplier" {
		if supplier := svc.ResolveSupplierReference(reference); supplier != nil {
			return supplier
		}
	}

	if customer := svc.ResolveCustomerReference(reference); customer != nil {
		return customer
	}

	if supplier := svc.ResolveSupplierReference(reference); supplier != nil {
		return supplier
	}

	if contact := svc.resolveCustomerContactReference(reference); contact != nil {
		return contact
	}

	if employee := svc.host.ResolveEmployeeReference(reference); employee != nil {
		return employee
	}

	if user := svc.resolveUserReference(reference); user != nil {
		return user
	}

	return nil
}

func (svc *Service) ResolveCustomerReference(reference string) *ButlerResolvedEntity {
	if reference == "" {
		return nil
	}

	var customer CustomerMaster
	exactErr := svc.db.Where(
		"business_name = ? OR short_code = ? OR customer_code = ? OR customer_id = ?",
		reference, reference, reference, reference,
	).First(&customer).Error
	if exactErr == nil {
		return &ButlerResolvedEntity{
			EntityType:  "customer",
			EntityID:    customer.CustomerID,
			DisplayName: customer.BusinessName,
			Confidence:  0.98,
			MatchReason: "exact customer match",
		}
	}

	escaped := text.EscapeLike(reference)
	pattern := "%" + escaped + "%"
	var matches []CustomerMaster
	if err := svc.db.Where(
		"business_name LIKE ? ESCAPE '\\' OR short_code LIKE ? ESCAPE '\\' OR customer_code LIKE ? ESCAPE '\\'",
		pattern, pattern, pattern,
	).Order("business_name").Limit(3).Find(&matches).Error; err == nil && len(matches) > 0 {
		customer = matches[0]
		resolution := &ButlerResolvedEntity{
			EntityType:  "customer",
			EntityID:    customer.CustomerID,
			DisplayName: customer.BusinessName,
			Confidence:  0.82,
			MatchReason: "fuzzy customer match",
		}
		if len(matches) > 1 {
			resolution.Ambiguous = true
			resolution.Confidence = 0.55
			resolution.MatchReason = "multiple customer matches"
			resolution.Alternatives = customerAlternatives(matches)
		}
		return resolution
	}

	return nil
}

func (svc *Service) resolveCustomerContactReference(reference string) *ButlerResolvedEntity {
	if reference == "" {
		return nil
	}

	var match contactMatch
	exactErr := svc.db.Table("customer_contacts").
		Select("customer_contacts.*, customers.business_name, customers.customer_id as business_id").
		Joins("JOIN customers ON customers.id = customer_contacts.customer_id").
		Where("customer_contacts.contact_name = ?", reference).
		Order("customer_contacts.is_primary_contact DESC, customer_contacts.contact_name").
		Scan(&match).Error
	if exactErr == nil && match.ContactName != "" {
		return &ButlerResolvedEntity{
			EntityType:        "customer_contact",
			EntityID:          match.ID,
			DisplayName:       match.ContactName,
			Confidence:        0.96,
			MatchReason:       "exact contact match",
			RelatedCustomerID: match.BusinessID,
			RelatedCustomer:   match.BusinessName,
		}
	}

	escaped := text.EscapeLike(reference)
	pattern := "%" + escaped + "%"
	var matches []contactMatch
	if err := svc.db.Table("customer_contacts").
		Select("customer_contacts.*, customers.business_name, customers.customer_id as business_id").
		Joins("JOIN customers ON customers.id = customer_contacts.customer_id").
		Where("customer_contacts.contact_name LIKE ? ESCAPE '\\'", pattern).
		Order("customer_contacts.is_primary_contact DESC, customer_contacts.contact_name").
		Limit(3).
		Scan(&matches).Error; err == nil && len(matches) > 0 && matches[0].ContactName != "" {
		match = matches[0]
		resolution := &ButlerResolvedEntity{
			EntityType:        "customer_contact",
			EntityID:          match.ID,
			DisplayName:       match.ContactName,
			Confidence:        0.79,
			MatchReason:       "fuzzy contact match",
			RelatedCustomerID: match.BusinessID,
			RelatedCustomer:   match.BusinessName,
		}
		if len(matches) > 1 {
			resolution.Ambiguous = true
			resolution.Confidence = 0.5
			resolution.MatchReason = "multiple contact matches"
			resolution.Alternatives = contactAlternatives(matches)
		}
		return resolution
	}

	return nil
}

func (svc *Service) resolveUserReference(reference string) *ButlerResolvedEntity {
	if reference == "" {
		return nil
	}

	var user User
	exactErr := svc.db.Where(
		"full_name = ? OR display_name = ? OR username = ?",
		reference, reference, reference,
	).First(&user).Error
	if exactErr == nil {
		return &ButlerResolvedEntity{
			EntityType:  "user",
			EntityID:    user.ID,
			DisplayName: FirstNonEmpty(user.DisplayName, user.FullName, user.Username),
			Confidence:  0.95,
			MatchReason: "exact internal user match",
		}
	}

	escaped := text.EscapeLike(reference)
	pattern := "%" + escaped + "%"
	var matches []User
	if err := svc.db.Where(
		"full_name LIKE ? ESCAPE '\\' OR display_name LIKE ? ESCAPE '\\' OR username LIKE ? ESCAPE '\\'",
		pattern, pattern, pattern,
	).Order("full_name").Limit(3).Find(&matches).Error; err == nil && len(matches) > 0 {
		user = matches[0]
		resolution := &ButlerResolvedEntity{
			EntityType:  "user",
			EntityID:    user.ID,
			DisplayName: FirstNonEmpty(user.DisplayName, user.FullName, user.Username),
			Confidence:  0.78,
			MatchReason: "fuzzy internal user match",
		}
		if len(matches) > 1 {
			resolution.Ambiguous = true
			resolution.Confidence = 0.52
			resolution.MatchReason = "multiple internal user matches"
			resolution.Alternatives = userAlternatives(matches)
		}
		return resolution
	}

	return nil
}

func (svc *Service) ResolveSupplierReference(reference string) *ButlerResolvedEntity {
	if reference == "" {
		return nil
	}

	var supplier SupplierMaster
	exactErr := svc.db.Where(
		"supplier_name = ? OR supplier_code = ?",
		reference, reference,
	).First(&supplier).Error
	if exactErr == nil {
		return &ButlerResolvedEntity{
			EntityType:  "supplier",
			EntityID:    supplier.ID,
			DisplayName: supplier.SupplierName,
			Confidence:  0.97,
			MatchReason: "exact supplier match",
		}
	}

	escaped := text.EscapeLike(reference)
	pattern := "%" + escaped + "%"
	var matches []SupplierMaster
	if err := svc.db.Where(
		"supplier_name LIKE ? ESCAPE '\\' OR supplier_code LIKE ? ESCAPE '\\'",
		pattern, pattern,
	).Order("supplier_name").Limit(3).Find(&matches).Error; err == nil && len(matches) > 0 {
		supplier = matches[0]
		resolution := &ButlerResolvedEntity{
			EntityType:  "supplier",
			EntityID:    supplier.ID,
			DisplayName: supplier.SupplierName,
			Confidence:  0.81,
			MatchReason: "fuzzy supplier match",
		}
		if len(matches) > 1 {
			resolution.Ambiguous = true
			resolution.Confidence = 0.56
			resolution.MatchReason = "multiple supplier matches"
			resolution.Alternatives = supplierAlternatives(matches)
		}
		return resolution
	}

	return nil
}

func (svc *Service) getCustomerContactContext(resolution *ButlerResolvedEntity) map[string]any {
	result := make(map[string]any)
	if resolution == nil || resolution.EntityType != "customer_contact" {
		return result
	}

	var contact CustomerContact
	if err := svc.db.First(&contact, "id = ?", resolution.EntityID).Error; err != nil {
		return result
	}

	result["contact"] = map[string]any{
		"id":                 contact.ID,
		"name":               contact.ContactName,
		"job_title":          contact.JobTitle,
		"email":              contact.Email,
		"phone":              contact.Phone,
		"is_primary_contact": contact.IsPrimaryContact,
	}
	result["linked_customer"] = map[string]any{
		"customer_id":   resolution.RelatedCustomerID,
		"business_name": resolution.RelatedCustomer,
	}

	if resolution.RelatedCustomer != "" {
		customerCtx := svc.getCustomerContext(resolution.RelatedCustomer)
		if customerSummary, ok := customerCtx["customer"]; ok {
			result["linked_customer_financials"] = customerSummary
		}
		if offers, ok := customerCtx["active_offers"]; ok {
			result["active_offers"] = offers
		}
		if tasks, ok := customerCtx["pending_tasks"]; ok {
			result["pending_tasks"] = tasks
		}
	}

	return result
}

func (svc *Service) getAccountManagerContext(resolution *ButlerResolvedEntity) map[string]any {
	result := make(map[string]any)
	if resolution == nil || resolution.EntityType != "user" {
		return result
	}

	var user User
	if err := svc.db.First(&user, "id = ?", resolution.EntityID).Error; err != nil {
		return result
	}

	identifiers := UniqueNonEmptyStrings(user.FullName, user.DisplayName, user.Username)
	result["user"] = map[string]any{
		"id":           user.ID,
		"full_name":    user.FullName,
		"display_name": user.DisplayName,
		"username":     user.Username,
		"department":   user.Department,
		"job_title":    user.JobTitle,
		"is_active":    user.IsActive,
	}

	if len(identifiers) == 0 {
		return result
	}

	var opportunities []Opportunity
	if err := svc.db.Where("salesperson IN ?", identifiers).
		Order("offer_date DESC").
		Limit(10).
		Find(&opportunities).Error; err != nil {
		return result
	}

	openCount := 0
	totalPipeline := 0.0
	customers := map[string]bool{}
	oppSummary := make([]map[string]any, 0, len(opportunities))
	for _, opp := range opportunities {
		if opp.Stage != "Won" && opp.Stage != "Lost" {
			openCount++
			totalPipeline += opp.RevenueBHD
		}
		if opp.CustomerName != "" {
			customers[opp.CustomerName] = true
		}
		oppSummary = append(oppSummary, map[string]any{
			"folder_number": opp.FolderNumber,
			"title":         FirstNonEmpty(opp.Title, opp.FolderName),
			"customer_name": opp.CustomerName,
			"stage":         opp.Stage,
			"revenue_bhd":   opp.RevenueBHD,
			"offer_date":    opp.OfferDate.Format("2006-01-02"),
			"expected_date": FormatOptionalDate(opp.ExpectedDate),
		})
	}

	result["portfolio_summary"] = map[string]any{
		"open_opportunities": openCount,
		"tracked_customers":  len(customers),
		"open_pipeline_bhd":  totalPipeline,
	}
	result["recent_opportunities"] = oppSummary

	return result
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func UniqueNonEmptyStrings(values ...string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

func FormatOptionalDate(ts *time.Time) string {
	if ts == nil {
		return ""
	}
	return ts.Format("2006-01-02")
}

func customerAlternatives(matches []CustomerMaster) []map[string]any {
	alternatives := make([]map[string]any, 0, len(matches))
	for _, match := range matches {
		alternatives = append(alternatives, map[string]any{
			"entity_type": "customer",
			"id":          match.CustomerID,
			"name":        match.BusinessName,
			"short_code":  match.ShortCode,
		})
	}
	return alternatives
}

func contactAlternatives(matches []contactMatch) []map[string]any {
	alternatives := make([]map[string]any, 0, len(matches))
	for _, match := range matches {
		alternatives = append(alternatives, map[string]any{
			"entity_type":   "customer_contact",
			"id":            match.ID,
			"name":          match.ContactName,
			"customer_name": match.BusinessName,
			"job_title":     match.JobTitle,
		})
	}
	return alternatives
}

func userAlternatives(matches []User) []map[string]any {
	alternatives := make([]map[string]any, 0, len(matches))
	for _, match := range matches {
		alternatives = append(alternatives, map[string]any{
			"entity_type":  "user",
			"id":           match.ID,
			"display_name": FirstNonEmpty(match.DisplayName, match.FullName, match.Username),
			"department":   match.Department,
			"job_title":    match.JobTitle,
		})
	}
	return alternatives
}

func supplierAlternatives(matches []SupplierMaster) []map[string]any {
	alternatives := make([]map[string]any, 0, len(matches))
	for _, match := range matches {
		alternatives = append(alternatives, map[string]any{
			"entity_type":   "supplier",
			"id":            match.ID,
			"display_name":  match.SupplierName,
			"supplier_code": match.SupplierCode,
			"country":       match.Country,
		})
	}
	return alternatives
}

func extractQuotationDraftClues(query string) map[string]any {
	clues := make(map[string]any)
	if strings.TrimSpace(query) == "" {
		return clues
	}

	queryLower := strings.ToLower(query)
	if strings.Contains(queryLower, "service visit") {
		clues["service_type"] = "service_visit"
	}
	if strings.Contains(queryLower, "calibration") {
		clues["scope"] = "calibration"
	}

	dayRe := regexp.MustCompile(`(?i)(\d+)\s*day`)
	if m := dayRe.FindStringSubmatch(query); len(m) > 1 {
		clues["visit_days"] = atoiSafe(m[1])
	}

	priceRe := regexp.MustCompile(`(?i)(?:daily price.*?around|price.*?around|around)\s*(\d+(?:\.\d+)?)\s*(?:bd|bhd)`)
	if m := priceRe.FindStringSubmatch(query); len(m) > 1 {
		clues["daily_rate_bhd"] = atofSafe(m[1])
	}

	optionalRe := regexp.MustCompile(`(?i)(?:optional.*?line item.*?)(\d+(?:\.\d+)?)\s*(?:bd|bhd)`)
	if m := optionalRe.FindStringSubmatch(query); len(m) > 1 {
		clues["optional_line_amount_bhd"] = atofSafe(m[1])
		clues["optional_line_label"] = "mobilization_of_repair_expert"
	}

	if strings.Contains(queryLower, "expert") && clues["optional_line_label"] == nil {
		clues["optional_line_label"] = "mobilization_of_repair_expert"
	}

	return clues
}

func atoiSafe(value string) int {
	var out int
	fmt.Sscanf(value, "%d", &out)
	return out
}

func atofSafe(value string) float64 {
	var out float64
	fmt.Sscanf(value, "%f", &out)
	return out
}

func Round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func (svc *Service) getSystemTimeContext() map[string]any {
	now := time.Now()
	return map[string]any{
		"current_date":    now.Format("2006-01-02"),
		"current_time":    now.Format(time.RFC3339),
		"current_year":    now.Year(),
		"current_quarter": fmt.Sprintf("Q%d", (int(now.Month())-1)/3+1),
		"timezone":        now.Location().String(),
	}
}

func buildFieldFollowUpCandidates(items []map[string]any) []map[string]any {
	candidates := make([]map[string]any, 0, len(items))
	for idx, item := range items {
		candidate := map[string]any{
			"rank":                idx + 1,
			"customer_name":       item["customer_name"],
			"site":                item["site"],
			"equipment":           item["equipment"],
			"oem":                 item["oem"],
			"estimated_value_bhd": item["estimated_value_bhd"],
			"service_due_date":    item["service_due_date"],
			"follow_up_reason":    item["due_reason"],
			"recommended_action":  "Schedule field check and propose service visit",
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func buildAmbiguitySummary(resolution *ButlerResolvedEntity) map[string]any {
	if resolution == nil || !resolution.Ambiguous || len(resolution.Alternatives) == 0 {
		return map[string]any{
			"needs_clarification": false,
		}
	}

	return map[string]any{
		"needs_clarification": true,
		"message":             fmt.Sprintf("I found multiple %s matches for %q. Please tell me which one you mean.", resolution.EntityType, resolution.DisplayName),
		"alternatives":        resolution.Alternatives,
	}
}

func buildQuotationDraftPayload(customer map[string]any, primaryContact map[string]any, clues map[string]any, missing []string, resolution *ButlerResolvedEntity) map[string]any {
	payload := map[string]any{
		"ready_to_draft": false,
		"document_type":  "Quotation",
	}

	if len(clues) > 0 {
		if scope, ok := clues["scope"]; ok {
			payload["scope"] = scope
		}
		if serviceType, ok := clues["service_type"]; ok {
			payload["service_type"] = serviceType
		}
		if visitDays, ok := clues["visit_days"]; ok {
			payload["visit_days"] = visitDays
		}
		if dailyRate, ok := clues["daily_rate_bhd"]; ok {
			payload["daily_rate_bhd"] = dailyRate
		}
		if optionalAmount, ok := clues["optional_line_amount_bhd"]; ok {
			payload["optional_line_amount_bhd"] = optionalAmount
		}
		if optionalLabel, ok := clues["optional_line_label"]; ok {
			payload["optional_line_label"] = optionalLabel
		}
	}

	if resolution != nil && !resolution.Ambiguous {
		payload["resolved_entity_type"] = resolution.EntityType
		payload["resolved_entity_name"] = resolution.DisplayName
	}

	if customer != nil {
		if name, ok := customer["name"]; ok && name != "" {
			payload["customer_name"] = name
		}
		if grade, ok := customer["grade"]; ok {
			payload["customer_grade"] = grade
		}
		if paymentTermsDays, ok := customer["payment_terms_days"]; ok {
			payload["payment_terms_days"] = paymentTermsDays
		}
	}

	if len(primaryContact) > 0 {
		payload["attention_person"] = primaryContact["name"]
		payload["attention_phone"] = primaryContact["phone"]
		payload["attention_email"] = primaryContact["email"]
	}

	lineItems := make([]map[string]any, 0, 2)
	if visitDays, ok := clues["visit_days"].(int); ok && visitDays > 0 {
		if dailyRate, ok := clues["daily_rate_bhd"].(float64); ok && dailyRate > 0 {
			lineItems = append(lineItems, map[string]any{
				"description":     "Service visit",
				"quantity":        visitDays,
				"unit_price_bhd":  Round3(dailyRate),
				"total_price_bhd": Round3(float64(visitDays) * dailyRate),
			})
		}
	}
	if optionalAmount, ok := clues["optional_line_amount_bhd"].(float64); ok && optionalAmount > 0 {
		label := "Optional line item"
		if optionalLabel, ok := clues["optional_line_label"].(string); ok && optionalLabel != "" {
			label = optionalLabel
		}
		lineItems = append(lineItems, map[string]any{
			"description":     label,
			"quantity":        1,
			"unit_price_bhd":  Round3(optionalAmount),
			"total_price_bhd": Round3(optionalAmount),
			"optional":        true,
		})
	}
	payload["line_items"] = lineItems
	payload["missing_fields"] = missing
	payload["ready_to_draft"] = len(missing) == 0 && resolution != nil && !resolution.Ambiguous && len(lineItems) > 0

	return payload
}

// getBusinessSummary provides high-level business metrics
func (svc *Service) BusinessSummary() map[string]any {
	summary := make(map[string]any)
	now := time.Now()

	var customerCount, supplierCount, invoiceCount, orderCount int64
	svc.db.Model(&CustomerMaster{}).Count(&customerCount)
	svc.db.Model(&SupplierMaster{}).Count(&supplierCount)
	svc.db.Model(&Invoice{}).Count(&invoiceCount)
	svc.db.Model(&Order{}).Count(&orderCount)

	summary["total_customers"] = customerCount
	summary["total_suppliers"] = supplierCount
	summary["total_invoices"] = invoiceCount
	summary["total_orders"] = orderCount
	summary["as_of_date"] = now.Format("2006-01-02")
	summary["current_year"] = now.Year()

	// Customer grade distribution
	var gradeA, gradeB, gradeC, gradeD int64
	svc.db.Model(&CustomerMaster{}).Where("customer_grade = ?", "A").Count(&gradeA)
	svc.db.Model(&CustomerMaster{}).Where("customer_grade = ?", "B").Count(&gradeB)
	svc.db.Model(&CustomerMaster{}).Where("customer_grade = ?", "C").Count(&gradeC)
	svc.db.Model(&CustomerMaster{}).Where("customer_grade = ?", "D").Count(&gradeD)
	summary["grade_distribution"] = map[string]int64{"A": gradeA, "B": gradeB, "C": gradeC, "D": gradeD}

	// Revenue from invoices
	var totalRevenue float64
	svc.db.Model(&Invoice{}).Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&totalRevenue)
	summary["total_revenue_bhd"] = totalRevenue

	// Outstanding receivables
	var totalOutstanding float64
	svc.db.Model(&Invoice{}).Where("status != ?", "Paid").Select("COALESCE(SUM(outstanding_bhd), 0)").Scan(&totalOutstanding)
	summary["total_outstanding_bhd"] = totalOutstanding

	// Yearly revenue breakdown
	var allInvoices []Invoice
	svc.db.Select("grand_total_bhd, invoice_date, status").Find(&allInvoices)
	yearlyRevenue := make(map[string]float64)
	yearlyInvCount := make(map[string]int)
	yearlyPaidCount := make(map[string]int)
	for _, inv := range allInvoices {
		year := inv.InvoiceDate.Format("2006")
		yearlyRevenue[year] += inv.GrandTotalBHD
		yearlyInvCount[year]++
		if inv.Status == "Paid" {
			yearlyPaidCount[year]++
		}
	}
	yearBreakdown := make([]map[string]any, 0)
	for year, rev := range yearlyRevenue {
		yearBreakdown = append(yearBreakdown, map[string]any{
			"year":           year,
			"revenue_bhd":    fmt.Sprintf("%.3f", rev),
			"invoice_count":  yearlyInvCount[year],
			"paid_count":     yearlyPaidCount[year],
			"collection_pct": fmt.Sprintf("%.1f", float64(yearlyPaidCount[year])/float64(yearlyInvCount[year])*100),
		})
	}
	summary["yearly_revenue"] = yearBreakdown

	// Total payments received
	var totalPaymentsReceived float64
	svc.db.Model(&Payment{}).Select("COALESCE(SUM(amount_bhd), 0)").Scan(&totalPaymentsReceived)
	summary["total_payments_received_bhd"] = totalPaymentsReceived

	// Top 10 customers by total business
	var topByBusiness []CustomerMaster
	svc.db.Where("total_orders_value > 0").Order("total_orders_value DESC").Limit(10).Find(&topByBusiness)
	topBizList := make([]map[string]any, 0)
	for _, c := range topByBusiness {
		topBizList = append(topBizList, map[string]any{
			"name":         c.BusinessName,
			"total_value":  c.TotalOrdersValue,
			"total_orders": c.TotalOrdersCount,
			"outstanding":  c.OutstandingBHD,
			"grade":        c.CustomerGrade,
		})
	}
	summary["top_customers_by_business"] = topBizList

	var latestInvoice Invoice
	if err := svc.db.Order("invoice_date DESC").Limit(1).First(&latestInvoice).Error; err == nil {
		summary["latest_invoice_date"] = latestInvoice.InvoiceDate.Format("2006-01-02")
		summary["latest_invoice_year"] = latestInvoice.InvoiceDate.Year()
	}

	var latestOrder Order
	if err := svc.db.Order("order_date DESC").Limit(1).First(&latestOrder).Error; err == nil {
		summary["latest_order_date"] = latestOrder.OrderDate.Format("2006-01-02")
		summary["latest_order_year"] = latestOrder.OrderDate.Year()
	}

	return summary
}

// getCustomerContext fetches customer-specific data
func (svc *Service) getCustomerContext(entityName string) map[string]any {
	result := make(map[string]any)

	if entityName != "" {
		// Find specific customer by name (fuzzy match with wildcard escaping)
		var customer CustomerMaster
		escapedName := text.EscapeLike(entityName)
		query := "%" + escapedName + "%"
		if err := svc.db.Where("business_name LIKE ? ESCAPE '\\'", query).First(&customer).Error; err == nil {
			result["customer"] = map[string]any{
				"id":               customer.CustomerID,
				"name":             customer.BusinessName,
				"grade":            customer.CustomerGrade,
				"payment_grade":    customer.PaymentGrade,
				"avg_payment_days": customer.AvgPaymentDays,
				"outstanding_bhd":  customer.OutstandingBHD,
				"overdue_days":     customer.OverdueDays,
				"total_orders":     customer.TotalOrdersCount,
				"total_value":      customer.TotalOrdersValue,
				"industry":         customer.Industry,
				"credit_blocked":   customer.IsCreditBlocked,
				"ar_risk_tier":     customer.ARRiskTier,
			}

			// Get ALL invoices for this customer (for yearly aggregation)
			var invoices []Invoice
			svc.db.Where("customer_id = ?", customer.CustomerID).
				Order("invoice_date DESC").Find(&invoices)

			// Build yearly aggregates
			yearlyTotals := make(map[string]float64)
			yearlyCount := make(map[string]int)
			yearlyOutstanding := make(map[string]float64)
			for _, inv := range invoices {
				year := inv.InvoiceDate.Format("2006")
				yearlyTotals[year] += inv.GrandTotalBHD
				yearlyCount[year]++
				yearlyOutstanding[year] += inv.OutstandingBHD
			}

			yearSummary := make([]map[string]any, 0)
			for year, total := range yearlyTotals {
				yearSummary = append(yearSummary, map[string]any{
					"year":          year,
					"total_bhd":     fmt.Sprintf("%.3f", total),
					"invoice_count": yearlyCount[year],
					"outstanding":   fmt.Sprintf("%.3f", yearlyOutstanding[year]),
				})
			}
			result["yearly_business_summary"] = yearSummary

			// Recent invoices (last 10)
			invSummary := make([]map[string]any, 0)
			limit := 10
			if len(invoices) < limit {
				limit = len(invoices)
			}
			for _, inv := range invoices[:limit] {
				invSummary = append(invSummary, map[string]any{
					"number":      inv.InvoiceNumber,
					"date":        inv.InvoiceDate.Format("2006-01-02"),
					"total":       inv.GrandTotalBHD,
					"outstanding": inv.OutstandingBHD,
					"status":      inv.Status,
				})
			}
			result["recent_invoices"] = invSummary

			// Get recent payments
			var payments []Payment
			svc.db.Joins("JOIN invoices ON invoices.id = payments.invoice_id").
				Where("invoices.customer_id = ?", customer.CustomerID).
				Order("payments.payment_date DESC").Limit(10).Find(&payments)
			paySummary := make([]map[string]any, 0)
			for _, pay := range payments {
				paySummary = append(paySummary, map[string]any{
					"amount":      pay.AmountBHD,
					"date":        pay.PaymentDate.Format("2006-01-02"),
					"method":      pay.PaymentMethod,
					"days_to_pay": pay.DaysToPayment,
				})
			}
			result["recent_payments"] = paySummary

			// Customer contacts (decision makers)
			var contacts []CustomerContact
			svc.db.Where("customer_id = ?", customer.ID).Find(&contacts)
			contactList := make([]map[string]any, 0)
			for _, c := range contacts {
				contactList = append(contactList, map[string]any{
					"name":       c.ContactName,
					"job_title":  c.JobTitle,
					"email":      c.Email,
					"phone":      c.Phone,
					"is_primary": c.IsPrimaryContact,
				})
			}
			result["contacts"] = contactList

			// Recent orders with line item details
			var orders []Order
			svc.db.Where("customer_id = ?", customer.CustomerID).
				Order("order_date DESC").Limit(5).Find(&orders)
			orderSummary := make([]map[string]any, 0)
			for _, o := range orders {
				orderEntry := map[string]any{
					"number":   o.OrderNumber,
					"date":     o.OrderDate.Format("2006-01-02"),
					"total":    o.GrandTotalBHD,
					"status":   o.Status,
					"division": o.Division,
				}
				// Get line items for product mix insight
				var items []OrderItem
				svc.db.Where("order_id = ?", o.ID).Find(&items)
				itemList := make([]map[string]any, 0)
				for _, item := range items {
					itemList = append(itemList, map[string]any{
						"equipment":      item.Equipment,
						"model":          item.Model,
						"qty":            item.Quantity,
						"unit_price_bhd": item.UnitPrice,
						"margin_pct":     item.MarginPercent,
					})
				}
				orderEntry["items"] = itemList
				orderSummary = append(orderSummary, orderEntry)
			}
			result["recent_orders_detail"] = orderSummary

			// Entity notes (CRM notes)
			var notes []EntityNote
			svc.db.Where("entity_type = ? AND entity_id = ?", "customer", customer.ID).
				Order("created_at DESC").Limit(10).Find(&notes)
			noteList := make([]map[string]any, 0)
			for _, n := range notes {
				noteList = append(noteList, map[string]any{
					"type":    n.NoteType,
					"content": n.Content,
					"date":    n.CreatedAt.Format("2006-01-02"),
				})
			}
			result["notes"] = noteList

			// Pending follow-up tasks
			var tasks []FollowUpTask
			svc.db.Where("customer_id = ? AND status IN ?", customer.ID, []string{"pending", "in_progress", "overdue"}).
				Order("due_date ASC").Limit(5).Find(&tasks)
			taskList := make([]map[string]any, 0)
			for _, t := range tasks {
				taskList = append(taskList, map[string]any{
					"title":    t.Title,
					"due_date": t.DueDate.Format("2006-01-02"),
					"status":   t.Status,
					"priority": t.Priority,
				})
			}
			result["pending_tasks"] = taskList

			// Active offers for this customer (pipeline)
			var offers []Offer
			svc.db.Where("customer_id = ? AND stage NOT IN ?", customer.CustomerID, []string{"Lost", "Expired"}).
				Order("quotation_date DESC").Limit(5).Find(&offers)
			offerList := make([]map[string]any, 0)
			for _, o := range offers {
				offerList = append(offerList, map[string]any{
					"number":   o.OfferNumber,
					"stage":    o.Stage,
					"value":    o.TotalValueBHD,
					"margin":   o.EstimatedMargin,
					"date":     o.QuotationDate.Format("2006-01-02"),
					"validity": o.ValidityDate.Format("2006-01-02"),
				})
			}
			result["active_offers"] = offerList
		}
	}

	// Top customers by outstanding
	var topCustomers []CustomerMaster
	svc.db.Where("outstanding_bhd > 0").Order("outstanding_bhd DESC").Limit(5).Find(&topCustomers)
	topList := make([]map[string]any, 0)
	for _, c := range topCustomers {
		topList = append(topList, map[string]any{
			"name":        c.BusinessName,
			"outstanding": c.OutstandingBHD,
			"grade":       c.CustomerGrade,
			"overdue":     c.OverdueDays,
		})
	}
	result["top_outstanding_customers"] = topList

	return result
}

func (svc *Service) getCustomerPeriodSummary(entityName, query string) map[string]any {
	if strings.TrimSpace(entityName) == "" || strings.TrimSpace(query) == "" {
		return nil
	}

	start, end, label, ok := ParseQuarterWindowFromQuery(query)
	if !ok {
		return nil
	}

	resolution := svc.ResolveCustomerReference(entityName)
	if resolution == nil {
		return map[string]any{
			"status":         "customer_not_resolved",
			"period":         label,
			"message":        fmt.Sprintf("Could not resolve a unique customer for %q in context.", entityName),
			"invoices":       []map[string]any{},
			"invoices_count": 0,
			"orders":         []map[string]any{},
			"orders_count":   0,
		}
	}

	var invoices []Invoice
	if err := svc.db.Where("customer_id = ? AND invoice_date >= ? AND invoice_date <= ? AND status NOT IN ?", resolution.EntityID, start, end, []string{"Cancelled", "Void", "Proforma", "Draft"}).
		Order("invoice_date DESC").Find(&invoices).Error; err != nil {
		return map[string]any{
			"status":         "error",
			"period":         label,
			"error":          err.Error(),
			"invoices":       []map[string]any{},
			"invoices_count": 0,
			"orders":         []map[string]any{},
			"orders_count":   0,
		}
	}

	invoiceSummary := make([]map[string]any, 0, len(invoices))
	invoiceTotals := 0.0
	for _, inv := range invoices {
		invoiceTotals += inv.GrandTotalBHD
		invoiceSummary = append(invoiceSummary, map[string]any{
			"number":      inv.InvoiceNumber,
			"date":        inv.InvoiceDate.Format("2006-01-02"),
			"total":       inv.GrandTotalBHD,
			"status":      inv.Status,
			"outstanding": inv.OutstandingBHD,
		})
	}

	var orders []Order
	if err := svc.db.Where("customer_id = ? AND order_date >= ? AND order_date <= ?", resolution.EntityID, start, end).
		Order("order_date DESC").Find(&orders).Error; err != nil {
		return map[string]any{
			"status":             "error",
			"period":             label,
			"error":              err.Error(),
			"invoices":           invoiceSummary,
			"invoices_count":     len(invoices),
			"invoices_total_bhd": invoiceTotals,
			"orders":             []map[string]any{},
			"orders_count":       0,
		}
	}

	orderSummary := make([]map[string]any, 0, len(orders))
	orderTotals := 0.0
	for _, order := range orders {
		orderTotals += order.GrandTotalBHD
		orderSummary = append(orderSummary, map[string]any{
			"number":   order.OrderNumber,
			"date":     order.OrderDate.Format("2006-01-02"),
			"total":    order.GrandTotalBHD,
			"status":   order.Status,
			"division": order.Division,
		})
	}

	status := "available"
	if len(invoices) == 0 && len(orders) == 0 {
		status = "empty_period_window"
	}

	return map[string]any{
		"status":             status,
		"period":             label,
		"invoices_count":     len(invoices),
		"invoices_total_bhd": Round3(invoiceTotals),
		"invoices":           invoiceSummary,
		"orders_count":       len(orders),
		"orders_total_bhd":   Round3(orderTotals),
		"orders":             orderSummary,
	}
}

func (svc *Service) getMissingCustomerPeriodSummary(entityName, query string) map[string]any {
	periodLabel, ok := parseCustomerPeriodLabel(query)
	if !ok {
		return map[string]any{
			"status": "not_requested",
		}
	}

	resolution := svc.ResolveCustomerReference(entityName)
	if resolution == nil {
		return map[string]any{
			"status":  "customer_not_resolved",
			"period":  periodLabel,
			"message": fmt.Sprintf("Could not resolve a unique customer for %q in context.", entityName),
		}
	}

	return map[string]any{
		"status":             "empty_period_window",
		"period":             periodLabel,
		"invoices_count":     0,
		"invoices_total_bhd": 0.0,
		"invoices":           []map[string]any{},
		"orders_count":       0,
		"orders_total_bhd":   0.0,
		"orders":             []map[string]any{},
		"message":            "No invoices or orders found in this period in the connected data.",
	}
}

func (svc *Service) getBusinessPeriodSummary(query string) map[string]any {
	start, end, label, ok := ParseQuarterWindowFromQuery(query)
	if !ok || svc.db == nil {
		return nil
	}

	var invoices []Invoice
	if err := svc.db.Where("invoice_date >= ? AND invoice_date <= ? AND status NOT IN ?", start, end, []string{"Cancelled", "Void", "Proforma", "Draft"}).
		Order("invoice_date DESC").Find(&invoices).Error; err != nil {
		return map[string]any{
			"status": "error",
			"period": label,
			"error":  err.Error(),
		}
	}

	var orders []Order
	if err := svc.db.Where("order_date >= ? AND order_date <= ?", start, end).
		Order("order_date DESC").Find(&orders).Error; err != nil {
		return map[string]any{
			"status": "error",
			"period": label,
			"error":  err.Error(),
		}
	}

	var wonOpportunities []Opportunity
	if err := svc.db.Where("stage = ? AND ((closed_date IS NOT NULL AND closed_date >= ? AND closed_date <= ?) OR (order_date IS NOT NULL AND order_date >= ? AND order_date <= ?))",
		"Won", start, end, start, end).
		Order("COALESCE(order_date, closed_date) DESC").
		Find(&wonOpportunities).Error; err != nil {
		return map[string]any{
			"status": "error",
			"period": label,
			"error":  err.Error(),
		}
	}

	type stageSummary struct {
		Stage      string
		Count      int64
		TotalValue float64
	}
	var openStageRows []stageSummary
	svc.db.Model(&Opportunity{}).
		Where("stage NOT IN ?", []string{"Won", "Lost", "Expired"}).
		Where("(expected_date BETWEEN ? AND ?) OR (offer_date BETWEEN ? AND ?) OR (created_at BETWEEN ? AND ?)",
			start, end, start, end, start, end).
		Select("stage, COUNT(*) as count, COALESCE(SUM(revenue_bhd), 0) as total_value").
		Group("stage").
		Scan(&openStageRows)

	invoiceTotal := 0.0
	invoiceSummary := make([]map[string]any, 0, len(invoices))
	for _, inv := range invoices {
		invoiceTotal += inv.GrandTotalBHD
		invoiceSummary = append(invoiceSummary, map[string]any{
			"number":      inv.InvoiceNumber,
			"customer":    inv.CustomerName,
			"date":        inv.InvoiceDate.Format("2006-01-02"),
			"total_bhd":   Round3(inv.GrandTotalBHD),
			"status":      inv.Status,
			"outstanding": Round3(inv.OutstandingBHD),
		})
	}

	orderTotal := 0.0
	orderSummary := make([]map[string]any, 0, len(orders))
	for _, order := range orders {
		orderTotal += order.GrandTotalBHD
		orderSummary = append(orderSummary, map[string]any{
			"number":      order.OrderNumber,
			"customer":    order.CustomerName,
			"date":        order.OrderDate.Format("2006-01-02"),
			"total_bhd":   Round3(order.GrandTotalBHD),
			"status":      order.Status,
			"offer_id":    order.OfferID,
			"opportunity": order.RFQID,
		})
	}

	wonTotal := 0.0
	wonSummary := make([]map[string]any, 0, len(wonOpportunities))
	for _, opp := range wonOpportunities {
		wonTotal += opp.RevenueBHD
		wonSummary = append(wonSummary, map[string]any{
			"folder_number": opp.FolderNumber,
			"customer":      opp.CustomerName,
			"title":         FirstNonEmpty(opp.Title, opp.FolderName),
			"closed_date":   FormatOptionalDate(opp.ClosedDate),
			"order_date":    FormatOptionalDate(opp.OrderDate),
			"revenue_bhd":   Round3(opp.RevenueBHD),
			"offer_id":      opp.OfferID,
			"source":        opp.Source,
		})
	}

	openStageSummary := make([]map[string]any, 0, len(openStageRows))
	for _, row := range openStageRows {
		openStageSummary = append(openStageSummary, map[string]any{
			"stage":           row.Stage,
			"count":           row.Count,
			"total_value_bhd": Round3(row.TotalValue),
		})
	}

	status := "available"
	if len(invoices) == 0 && len(orders) == 0 && len(wonOpportunities) == 0 {
		status = "empty_period_window"
	}

	return map[string]any{
		"status":                      status,
		"period":                      label,
		"window_start":                start.Format("2006-01-02"),
		"window_end":                  end.Format("2006-01-02"),
		"invoices_count":              len(invoices),
		"invoices_total_bhd":          Round3(invoiceTotal),
		"invoices":                    invoiceSummary,
		"orders_count":                len(orders),
		"orders_total_bhd":            Round3(orderTotal),
		"orders":                      orderSummary,
		"won_opportunities_count":     len(wonOpportunities),
		"won_opportunities_total_bhd": Round3(wonTotal),
		"won_opportunities":           wonSummary,
		"open_pipeline_in_period":     openStageSummary,
		"message":                     "This period summary is authoritative for company-wide quarter questions.",
	}
}

func (svc *Service) BusinessYearSummary(query string) map[string]any {
	start, end, year, label, ok := ParseYearWindowFromQuery(query)
	if !ok || svc.db == nil {
		return nil
	}

	result := map[string]any{
		"status":       "available",
		"year":         year,
		"period":       label,
		"window_start": start.Format("2006-01-02"),
		"window_end":   end.Format("2006-01-02"),
	}

	var invoices []Invoice
	if err := svc.db.Where("invoice_date >= ? AND invoice_date <= ? AND status NOT IN ?",
		start, end, []string{"Cancelled", "Void", "Proforma", "Draft"}).
		Order("invoice_date ASC").
		Find(&invoices).Error; err != nil {
		return map[string]any{
			"status": "error",
			"year":   year,
			"period": label,
			"error":  err.Error(),
		}
	}

	invoiceTotal := 0.0
	for _, inv := range invoices {
		invoiceTotal += inv.GrandTotalBHD
	}

	var orders []Order
	if err := svc.db.Where("order_date >= ? AND order_date <= ?", start, end).
		Order("order_date ASC").
		Find(&orders).Error; err != nil {
		return map[string]any{
			"status": "error",
			"year":   year,
			"period": label,
			"error":  err.Error(),
		}
	}

	orderTotal := 0.0
	for _, order := range orders {
		orderTotal += order.GrandTotalBHD
	}

	var offers []Offer
	if err := svc.db.Where("quotation_date >= ? AND quotation_date <= ?", start, end).
		Order("quotation_date ASC").
		Find(&offers).Error; err != nil {
		return map[string]any{
			"status": "error",
			"year":   year,
			"period": label,
			"error":  err.Error(),
		}
	}

	offerTotal := 0.0
	for _, offer := range offers {
		offerTotal += offer.TotalValueBHD
	}

	var opportunities []Opportunity
	if err := svc.db.Where("year = ?", year).
		Order("offer_date ASC, created_at ASC").
		Find(&opportunities).Error; err != nil {
		return map[string]any{
			"status": "error",
			"year":   year,
			"period": label,
			"error":  err.Error(),
		}
	}

	opportunityTotal := 0.0
	openOpportunityCount := 0
	openOpportunityTotal := 0.0
	wonOpportunityCount := 0
	wonOpportunityTotal := 0.0
	stageRollup := make(map[string]map[string]any)
	for _, opp := range opportunities {
		opportunityTotal += opp.RevenueBHD
		if opp.Stage != "Won" && opp.Stage != "Lost" && opp.Stage != "Expired" {
			openOpportunityCount++
			openOpportunityTotal += opp.RevenueBHD
		}
		if opp.Stage == "Won" {
			wonOpportunityCount++
			wonOpportunityTotal += opp.RevenueBHD
		}
		row, exists := stageRollup[opp.Stage]
		if !exists {
			row = map[string]any{
				"stage":           opp.Stage,
				"count":           0,
				"total_value_bhd": 0.0,
			}
			stageRollup[opp.Stage] = row
		}
		row["count"] = row["count"].(int) + 1
		row["total_value_bhd"] = Round3(row["total_value_bhd"].(float64) + opp.RevenueBHD)
	}

	stageBreakdown := make([]map[string]any, 0, len(stageRollup))
	for _, row := range stageRollup {
		stageBreakdown = append(stageBreakdown, row)
	}
	sort.Slice(stageBreakdown, func(i, j int) bool {
		return fmt.Sprint(stageBreakdown[i]["stage"]) < fmt.Sprint(stageBreakdown[j]["stage"])
	})

	availableDataTypes := make([]string, 0, 4)
	if len(invoices) > 0 {
		availableDataTypes = append(availableDataTypes, "invoices")
	}
	if len(orders) > 0 {
		availableDataTypes = append(availableDataTypes, "orders")
	}
	if len(offers) > 0 {
		availableDataTypes = append(availableDataTypes, "offers")
	}
	if len(opportunities) > 0 {
		availableDataTypes = append(availableDataTypes, "opportunities")
	}

	if len(availableDataTypes) == 0 {
		result["status"] = "empty_year_window"
		result["message"] = "No invoices, orders, offers, or opportunities were found for this year in the connected data."
	}

	result["available_data_types"] = availableDataTypes
	result["invoices"] = map[string]any{
		"count":       len(invoices),
		"total_bhd":   Round3(invoiceTotal),
		"first_date":  firstInvoiceDate(invoices),
		"latest_date": lastInvoiceDate(invoices),
	}
	result["orders"] = map[string]any{
		"count":       len(orders),
		"total_bhd":   Round3(orderTotal),
		"first_date":  firstOrderDate(orders),
		"latest_date": lastOrderDate(orders),
	}
	result["offers"] = map[string]any{
		"count":       len(offers),
		"total_bhd":   Round3(offerTotal),
		"first_date":  firstOfferDate(offers),
		"latest_date": lastOfferDate(offers),
	}
	result["opportunities"] = map[string]any{
		"count":                      len(opportunities),
		"total_value_bhd":            Round3(opportunityTotal),
		"open_count":                 openOpportunityCount,
		"open_pipeline_bhd":          Round3(openOpportunityTotal),
		"won_count":                  wonOpportunityCount,
		"won_value_bhd":              Round3(wonOpportunityTotal),
		"stage_breakdown":            stageBreakdown,
		"latest_activity_date":       latestOpportunityActivityDate(opportunities),
		"latest_recorded_offer_date": latestOpportunityOfferDate(opportunities),
	}
	result["message"] = "This year summary is authoritative for questions about whether data exists for a given year and how complete that coverage is."

	return result
}

func parseCustomerPeriodLabel(query string) (string, bool) {
	_, _, periodLabel, ok := ParseQuarterWindowFromQuery(query)
	return periodLabel, ok
}

func ParseYearWindowFromQuery(query string) (time.Time, time.Time, int, string, bool) {
	now := time.Now()
	q := strings.ToLower(strings.TrimSpace(query))

	year := 0
	switch {
	case strings.Contains(q, "this year") || strings.Contains(q, "current year"):
		year = now.Year()
	case strings.Contains(q, "last year") || strings.Contains(q, "previous year"):
		year = now.Year() - 1
	default:
		re := regexp.MustCompile(`\b(20\d{2})\b`)
		if match := re.FindStringSubmatch(q); len(match) > 1 {
			parsed, err := strconv.Atoi(match[1])
			if err == nil {
				year = parsed
			}
		}
	}

	if year == 0 {
		return time.Time{}, time.Time{}, 0, "", false
	}

	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(year, time.December, 31, 23, 59, 59, 0, time.Local)
	return start, end, year, fmt.Sprintf("%d", year), true
}

func firstInvoiceDate(invoices []Invoice) string {
	if len(invoices) == 0 {
		return ""
	}
	return invoices[0].InvoiceDate.Format("2006-01-02")
}

func lastInvoiceDate(invoices []Invoice) string {
	if len(invoices) == 0 {
		return ""
	}
	return invoices[len(invoices)-1].InvoiceDate.Format("2006-01-02")
}

func firstOrderDate(orders []Order) string {
	if len(orders) == 0 {
		return ""
	}
	return orders[0].OrderDate.Format("2006-01-02")
}

func lastOrderDate(orders []Order) string {
	if len(orders) == 0 {
		return ""
	}
	return orders[len(orders)-1].OrderDate.Format("2006-01-02")
}

func firstOfferDate(offers []Offer) string {
	if len(offers) == 0 {
		return ""
	}
	return offers[0].QuotationDate.Format("2006-01-02")
}

func lastOfferDate(offers []Offer) string {
	if len(offers) == 0 {
		return ""
	}
	return offers[len(offers)-1].QuotationDate.Format("2006-01-02")
}

func latestOpportunityOfferDate(opportunities []Opportunity) string {
	latest := time.Time{}
	for _, opp := range opportunities {
		if opp.OfferDate.After(latest) {
			latest = opp.OfferDate
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.Format("2006-01-02")
}

func latestOpportunityActivityDate(opportunities []Opportunity) string {
	latest := time.Time{}
	for _, opp := range opportunities {
		for _, candidate := range []time.Time{opp.OfferDate, opp.CreatedAt, opp.UpdatedAt} {
			if candidate.After(latest) {
				latest = candidate
			}
		}
		if opp.ExpectedDate != nil && opp.ExpectedDate.After(latest) {
			latest = *opp.ExpectedDate
		}
		if opp.ClosedDate != nil && opp.ClosedDate.After(latest) {
			latest = *opp.ClosedDate
		}
		if opp.OrderDate != nil && opp.OrderDate.After(latest) {
			latest = *opp.OrderDate
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.Format("2006-01-02")
}

func ParseQuarterWindowFromQuery(query string) (time.Time, time.Time, string, bool) {
	now := time.Now()
	q := strings.ToLower(strings.TrimSpace(query))

	if strings.Contains(q, "last 2 quarter") || strings.Contains(q, "last two quarter") {
		month := int(now.Month())
		currentQuarter := (month-1)/3 + 1
		endQuarter := currentQuarter - 1
		endYear := now.Year()
		if endQuarter == 0 {
			endQuarter = 4
			endYear--
		}

		startQuarter := endQuarter - 1
		startYear := endYear
		if startQuarter == 0 {
			startQuarter = 4
			startYear--
		}

		startMonth := time.Month((startQuarter-1)*3 + 1)
		start := time.Date(startYear, startMonth, 1, 0, 0, 0, 0, time.Local)

		endStartMonth := time.Month((endQuarter-1)*3 + 1)
		endStart := time.Date(endYear, endStartMonth, 1, 0, 0, 0, 0, time.Local)
		end := endStart.AddDate(0, 3, 0).Add(-time.Second)

		label := fmt.Sprintf("last 2 quarters (Q%d %d to Q%d %d)", startQuarter, startYear, endQuarter, endYear)
		return start, end, label, true
	}

	if strings.Contains(q, "last quarter") || strings.Contains(q, "previous quarter") {
		month := int(now.Month())
		currentQuarter := (month-1)/3 + 1
		previousQuarter := currentQuarter - 1
		year := now.Year()
		if previousQuarter == 0 {
			previousQuarter = 4
			year--
		}
		startMonth := time.Month((previousQuarter-1)*3 + 1)
		start := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.Local)
		end := start.AddDate(0, 3, 0).Add(-time.Second)
		label := fmt.Sprintf("Q%d %d", previousQuarter, year)
		return start, end, label, true
	}

	if strings.Contains(q, "this quarter") || strings.Contains(q, "current quarter") {
		month := int(now.Month())
		currentQuarter := (month-1)/3 + 1
		year := now.Year()
		startMonth := time.Month((currentQuarter-1)*3 + 1)
		start := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.Local)
		end := start.AddDate(0, 3, 0).Add(-time.Second)
		label := fmt.Sprintf("Q%d %d", currentQuarter, year)
		return start, end, label, true
	}

	re := regexp.MustCompile(`(?i)\bq([1-4])(?:\s+)?(?:\(?(\d{4})\)?)?`)
	m := re.FindStringSubmatch(q)
	if len(m) == 3 {
		var quarter int
		fmt.Sscanf(m[1], "%d", &quarter)
		year := now.Year()
		if m[2] != "" {
			fmt.Sscanf(m[2], "%d", &year)
		}
		startMonth := time.Month((quarter-1)*3 + 1)
		start := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.Local)
		end := start.AddDate(0, 3, 0).Add(-time.Second)
		label := fmt.Sprintf("Q%d %d", quarter, year)
		return start, end, label, true
	}

	return time.Time{}, time.Time{}, "", false
}

// redactFinancialFromContext removes financial data from context maps for non-finance users
func (svc *Service) redactFinancialFromContext(ctx map[string]any) {
	// Redact financial fields from customer data
	if customer, ok := ctx["customer"].(map[string]any); ok {
		delete(customer, "outstanding_bhd")
		delete(customer, "payment_grade")
		delete(customer, "avg_payment_days")
		delete(customer, "overdue_days")
		delete(customer, "total_value")
		delete(customer, "ar_risk_tier")
		delete(customer, "credit_blocked")
	}
	// Remove invoice amounts and payment data entirely
	delete(ctx, "recent_invoices")
	delete(ctx, "recent_payments")
	delete(ctx, "top_outstanding_customers")
}

func (svc *Service) redactRestrictedBusinessContext(context map[string]any) {
	if context == nil {
		return
	}

	for _, key := range []string{
		"financial_data",
		"ar_summary",
		"cashflow_projection",
		"banking_data",
		"credit_notes",
		"dso_analysis",
		"customer_profitability",
		"forecast_intelligence",
		"business_period_summary",
		"business_year_summary",
		"customer_period_summary",
	} {
		delete(context, key)
	}

	if customerCtx, ok := context["customer_data"].(map[string]any); ok {
		svc.redactFinancialFromContext(customerCtx)
	}
	if supplierCtx, ok := context["supplier_data"].(map[string]any); ok {
		redactSupplierFinancialContext(supplierCtx)
	}
	if operationsCtx, ok := context["operations_data"].(map[string]any); ok {
		redactOperationsFinancialContext(operationsCtx)
	}
	if riskCtx, ok := context["risk_data"].(map[string]any); ok {
		redactRiskFinancialContext(riskCtx)
	}
	redactFinancialKeysRecursive(context)
}

func redactSupplierFinancialContext(ctx map[string]any) {
	for _, key := range []string{
		"yearly_purchase_summary",
		"recent_purchase_orders",
		"total_po_value_recent",
		"recent_supplier_invoices",
		"total_supplier_invoice_value",
		"recent_payments",
	} {
		delete(ctx, key)
	}
	redactListMaps(ctx["issues"], func(row map[string]any) {
		delete(row, "cost_bhd")
	})
	redactListMaps(ctx["top_suppliers"], func(row map[string]any) {
		delete(row, "total_po")
	})
}

func redactOperationsFinancialContext(ctx map[string]any) {
	for _, key := range []string{
		"active_order_value_bhd",
		"active_po_value_bhd",
		"active_offer_pipeline_bhd",
	} {
		delete(ctx, key)
	}
	redactListMaps(ctx["recent_orders"], func(row map[string]any) {
		delete(row, "total")
	})
	redactListMaps(ctx["recent_offers_detail"], func(row map[string]any) {
		delete(row, "value")
		delete(row, "margin")
		redactListMaps(row["items"], func(item map[string]any) {
			delete(item, "unit_price_bhd")
			delete(item, "margin_pct")
		})
	})
	redactListMaps(ctx["active_opportunities"], func(row map[string]any) {
		delete(row, "revenue")
	})
}

func redactRiskFinancialContext(ctx map[string]any) {
	for _, key := range []string{
		"total_overdue_bhd",
		"total_overdue_ap_bhd",
		"overdue_invoices",
		"overdue_supplier_invoices",
		"high_risk_customers",
		"critical_risk_customers",
	} {
		delete(ctx, key)
	}
}

func redactListMaps(value any, redact func(map[string]any)) {
	switch entries := value.(type) {
	case []map[string]any:
		for _, entry := range entries {
			redact(entry)
		}
	case []any:
		for _, entry := range entries {
			if row, ok := entry.(map[string]any); ok {
				redact(row)
			}
		}
	}
}

func redactFinancialKeysRecursive(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isRestrictedFinanceContextKey(key) {
				delete(typed, key)
				continue
			}
			redactFinancialKeysRecursive(child)
		}
	case []map[string]any:
		for _, child := range typed {
			redactFinancialKeysRecursive(child)
		}
	case []any:
		for _, child := range typed {
			redactFinancialKeysRecursive(child)
		}
	}
}

func isRestrictedFinanceContextKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if lower == "" {
		return false
	}
	fragments := []string{
		"amount",
		"balance",
		"bhd",
		"cash",
		"cost",
		"credit_limit",
		"expense",
		"grand_total",
		"margin",
		"outstanding",
		"payable",
		"payment",
		"price",
		"profit",
		"receivable",
		"revenue",
		"salary",
		"subtotal",
		"supplier_invoice_value",
		"total_po",
		"total_value",
		"unit_price",
		"value_bhd",
		"weighted_pipeline",
	}
	for _, fragment := range fragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

// getSupplierContext fetches supplier-specific data
func (svc *Service) getSupplierContext(entityName string) map[string]any {
	result := make(map[string]any)

	if entityName != "" {
		var supplier SupplierMaster
		escapedName := text.EscapeLike(entityName)
		query := "%" + escapedName + "%"
		if err := svc.db.Where("supplier_name LIKE ? ESCAPE '\\'", query).First(&supplier).Error; err == nil {
			result["supplier"] = map[string]any{
				"id":             supplier.ID,
				"name":           supplier.SupplierName,
				"type":           supplier.SupplierType,
				"country":        supplier.Country,
				"rating":         supplier.Rating,
				"lead_time_days": supplier.LeadTimeDays,
				"brands":         supplier.BrandsHandled,
				"contact":        supplier.PrimaryContact,
				"email":          supplier.Email,
			}

			// Get ALL POs for this supplier (for yearly aggregation)
			var pos []PurchaseOrder
			svc.db.Where("supplier_id = ?", supplier.ID).
				Order("po_date DESC").Find(&pos)

			// Build yearly aggregates
			yearlyPO := make(map[string]float64)
			yearlyPOCount := make(map[string]int)
			for _, po := range pos {
				year := po.PODate.Format("2006")
				yearlyPO[year] += po.TotalBHD
				yearlyPOCount[year]++
			}
			yearlyPOSummary := make([]map[string]any, 0)
			for year, total := range yearlyPO {
				yearlyPOSummary = append(yearlyPOSummary, map[string]any{
					"year":      year,
					"total_bhd": fmt.Sprintf("%.3f", total),
					"po_count":  yearlyPOCount[year],
				})
			}
			result["yearly_purchase_summary"] = yearlyPOSummary

			// Recent POs (last 10)
			poSummary := make([]map[string]any, 0)
			var totalPOValue float64
			limit := 10
			if len(pos) < limit {
				limit = len(pos)
			}
			for _, po := range pos[:limit] {
				totalPOValue += po.TotalBHD
				poSummary = append(poSummary, map[string]any{
					"number": po.PONumber,
					"date":   po.PODate.Format("2006-01-02"),
					"total":  po.TotalBHD,
					"status": po.Status,
				})
			}
			result["recent_purchase_orders"] = poSummary
			result["total_po_value_recent"] = totalPOValue

			// Supplier invoices
			var supInvoices []SupplierInvoice
			svc.db.Where("supplier_id = ?", supplier.ID).
				Order("invoice_date DESC").Limit(10).Find(&supInvoices)
			supInvSummary := make([]map[string]any, 0)
			var totalSupInvValue float64
			for _, si := range supInvoices {
				totalSupInvValue += si.TotalBHD
				supInvSummary = append(supInvSummary, map[string]any{
					"number": si.InvoiceNumber,
					"date":   si.InvoiceDate.Format("2006-01-02"),
					"total":  si.TotalBHD,
					"status": si.Status,
				})
			}
			result["recent_supplier_invoices"] = supInvSummary
			result["total_supplier_invoice_value"] = totalSupInvValue

			// Supplier contacts
			var contacts []SupplierContact
			svc.db.Where("supplier_id = ?", supplier.ID).Find(&contacts)
			contactList := make([]map[string]any, 0)
			for _, c := range contacts {
				contactList = append(contactList, map[string]any{
					"name":       c.ContactName,
					"job_title":  c.JobTitle,
					"email":      c.Email,
					"phone":      c.Phone,
					"is_primary": c.IsPrimaryContact,
				})
			}
			result["contacts"] = contactList

			// Entity notes (CRM notes for supplier)
			var notes []EntityNote
			svc.db.Where("entity_type = ? AND entity_id = ?", "supplier", supplier.ID).
				Order("created_at DESC").Limit(10).Find(&notes)
			noteList := make([]map[string]any, 0)
			for _, n := range notes {
				noteList = append(noteList, map[string]any{
					"type":    n.NoteType,
					"content": n.Content,
					"date":    n.CreatedAt.Format("2006-01-02"),
				})
			}
			result["notes"] = noteList

			// Supplier issues (quality/delivery problems)
			var issues []SupplierIssue
			svc.db.Where("supplier_id = ?", supplier.ID).
				Order("created_at DESC").Limit(5).Find(&issues)
			issueList := make([]map[string]any, 0)
			for _, iss := range issues {
				issueList = append(issueList, map[string]any{
					"order_ref":   iss.OrderRef,
					"description": iss.Description,
					"status":      iss.Status,
					"cost_bhd":    iss.CostBHD,
				})
			}
			result["issues"] = issueList

			// Supplier payments summary
			var supPayments []SupplierPayment
			svc.db.Where("supplier_id = ?", supplier.ID).
				Order("payment_date DESC").Limit(10).Find(&supPayments)
			supPayList := make([]map[string]any, 0)
			for _, sp := range supPayments {
				supPayList = append(supPayList, map[string]any{
					"amount":   sp.AmountBHD,
					"date":     sp.PaymentDate.Format("2006-01-02"),
					"method":   sp.PaymentMethod,
					"currency": sp.Currency,
				})
			}
			result["recent_payments"] = supPayList

			// GRN quality stats for this supplier
			var grnCount int64
			var grnPassedCount int64
			svc.db.Model(&GoodsReceivedNote{}).Joins("JOIN purchase_orders ON purchase_orders.id = goods_received_notes.purchase_order_id").
				Where("purchase_orders.supplier_id = ?", supplier.ID).Count(&grnCount)
			svc.db.Model(&GoodsReceivedNote{}).Joins("JOIN purchase_orders ON purchase_orders.id = goods_received_notes.purchase_order_id").
				Where("purchase_orders.supplier_id = ? AND goods_received_notes.qc_status = ?", supplier.ID, "Passed").Count(&grnPassedCount)
			if grnCount > 0 {
				result["grn_quality"] = map[string]any{
					"total_grns": grnCount,
					"passed":     grnPassedCount,
					"pass_rate":  fmt.Sprintf("%.1f%%", float64(grnPassedCount)/float64(grnCount)*100),
				}
			}
		}
	}

	// Top suppliers by recent PO value (compute from PO table)
	var allSuppliers []SupplierMaster
	svc.db.Limit(20).Find(&allSuppliers)
	topList := make([]map[string]any, 0)
	for _, s := range allSuppliers {
		var poTotal float64
		svc.db.Model(&PurchaseOrder{}).Where("supplier_id = ?", s.ID).
			Select("COALESCE(SUM(total_bhd), 0)").Scan(&poTotal)
		if poTotal > 0 {
			topList = append(topList, map[string]any{
				"name":      s.SupplierName,
				"total_po":  poTotal,
				"rating":    s.Rating,
				"lead_days": s.LeadTimeDays,
			})
		}
	}
	result["top_suppliers"] = topList

	return result
}

// getFinancialContext fetches financial metrics
func (svc *Service) getFinancialContext() map[string]any {
	result := make(map[string]any)
	now := time.Now()
	result["as_of_date"] = now.Format("2006-01-02")
	result["current_year"] = now.Year()

	// Invoice statistics
	var totalInvoiced, totalPaid, totalOutstanding float64
	svc.db.Model(&Invoice{}).Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&totalInvoiced)
	svc.db.Model(&Invoice{}).Where("status = ?", "Paid").Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&totalPaid)
	svc.db.Model(&Invoice{}).Where("status != ?", "Paid").Select("COALESCE(SUM(outstanding_bhd), 0)").Scan(&totalOutstanding)

	result["total_invoiced_bhd"] = totalInvoiced
	result["total_paid_bhd"] = totalPaid
	result["total_outstanding_bhd"] = totalOutstanding

	// Invoice status counts
	var sentCount, paidCount, overdueCount int64
	svc.db.Model(&Invoice{}).Where("status = ?", "Sent").Count(&sentCount)
	svc.db.Model(&Invoice{}).Where("status = ?", "Paid").Count(&paidCount)
	svc.db.Model(&Invoice{}).Where("status = ?", "Overdue").Count(&overdueCount)
	result["invoice_counts"] = map[string]int64{"sent": sentCount, "paid": paidCount, "overdue": overdueCount}

	// Average days to payment
	var avgDays float64
	svc.db.Model(&Payment{}).Select("COALESCE(AVG(days_to_payment), 0)").Scan(&avgDays)
	result["avg_days_to_payment"] = avgDays

	// Recent payments
	var recentPayments []Payment
	svc.db.Order("payment_date DESC").Limit(10).Find(&recentPayments)
	payList := make([]map[string]any, 0)
	for _, p := range recentPayments {
		payList = append(payList, map[string]any{
			"invoice": p.InvoiceNumber,
			"amount":  p.AmountBHD,
			"date":    p.PaymentDate.Format("2006-01-02"),
			"method":  p.PaymentMethod,
		})
	}
	result["recent_payments"] = payList

	// Yearly financial breakdown
	var allInvs []Invoice
	svc.db.Select("grand_total_bhd, outstanding_bhd, invoice_date, status, gross_margin_percent").Find(&allInvs)
	yearlyData := make(map[string]map[string]float64)
	for _, inv := range allInvs {
		year := inv.InvoiceDate.Format("2006")
		if yearlyData[year] == nil {
			yearlyData[year] = map[string]float64{}
		}
		yearlyData[year]["invoiced"] += inv.GrandTotalBHD
		yearlyData[year]["outstanding"] += inv.OutstandingBHD
		yearlyData[year]["count"]++
		if inv.Status == "Paid" {
			yearlyData[year]["paid_count"]++
			yearlyData[year]["paid_value"] += inv.GrandTotalBHD
		}
		if inv.GrossMarginPercent > 0 {
			yearlyData[year]["margin_sum"] += inv.GrossMarginPercent
			yearlyData[year]["margin_count"]++
		}
	}
	yearlyFinance := make([]map[string]any, 0)
	for year, data := range yearlyData {
		entry := map[string]any{
			"year":            year,
			"total_invoiced":  fmt.Sprintf("%.3f", data["invoiced"]),
			"total_paid":      fmt.Sprintf("%.3f", data["paid_value"]),
			"outstanding":     fmt.Sprintf("%.3f", data["outstanding"]),
			"invoice_count":   int(data["count"]),
			"collection_rate": fmt.Sprintf("%.1f%%", data["paid_count"]/data["count"]*100),
		}
		if data["margin_count"] > 0 {
			entry["avg_margin"] = fmt.Sprintf("%.1f%%", data["margin_sum"]/data["margin_count"])
		}
		yearlyFinance = append(yearlyFinance, entry)
	}
	result["yearly_financial_summary"] = yearlyFinance

	// Supplier payments summary
	var totalSupplierPayments float64
	svc.db.Model(&SupplierPayment{}).Select("COALESCE(SUM(amount_bhd), 0)").Scan(&totalSupplierPayments)
	result["total_supplier_payments_bhd"] = totalSupplierPayments

	// Bank accounts (cash position)
	var bankAccounts []BankAccount
	svc.db.Where("is_active = ?", true).Find(&bankAccounts)
	bankList := make([]map[string]any, 0)
	var totalBankBalance float64
	for _, ba := range bankAccounts {
		totalBankBalance += ba.CurrentBalance
		bankList = append(bankList, map[string]any{
			"bank":     ba.BankName,
			"account":  ba.AccountName,
			"currency": ba.Currency,
			"balance":  ba.CurrentBalance,
		})
	}
	result["bank_accounts"] = bankList
	result["total_bank_balance_bhd"] = totalBankBalance

	// Current FX rates
	var fxRates []CurrencyExchangeRate
	svc.db.Where("effective_to IS NULL").Find(&fxRates)
	rateList := make([]map[string]any, 0)
	for _, fx := range fxRates {
		rateList = append(rateList, map[string]any{
			"currency": fx.CurrencyCode,
			"rate":     fx.Rate,
			"since":    fx.EffectiveFrom.Format("2006-01-02"),
		})
	}
	result["current_fx_rates"] = rateList

	// Accounts payable summary (unpaid supplier invoices)
	var totalAP float64
	var apCount int64
	svc.db.Model(&SupplierInvoice{}).Where("status NOT IN ?", []string{"Paid", "Rejected"}).
		Select("COALESCE(SUM(total_bhd), 0)").Scan(&totalAP)
	svc.db.Model(&SupplierInvoice{}).Where("status NOT IN ?", []string{"Paid", "Rejected"}).Count(&apCount)
	result["total_payables_bhd"] = totalAP
	result["unpaid_supplier_invoices"] = apCount

	ov := overlay.Active()
	divisionCase := ov.DivisionNormalizationCase("division")
	// Wave 12.5 B4: emit one revenue entry PER configured division (keyed by the
	// division's registry Key) instead of a frozen two-slot primary/secondary
	// breakdown, so a 3rd+ division's revenue is no longer silently dropped. This
	// feeds Butler's LLM context only — no document identity is stamped here.
	divisionRevenue := make(map[string]any, len(ov.Divisions))
	for _, d := range ov.Divisions {
		var rev float64
		svc.db.Model(&Invoice{}).Where(divisionCase+" = ?", d.Key).
			Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&rev)
		divisionRevenue[d.Key] = rev
	}
	result["division_revenue"] = divisionRevenue

	var latestInvoice Invoice
	if err := svc.db.Order("invoice_date DESC").Limit(1).First(&latestInvoice).Error; err == nil {
		result["latest_invoice_date"] = latestInvoice.InvoiceDate.Format("2006-01-02")
		result["latest_invoice_year"] = latestInvoice.InvoiceDate.Year()
	}

	var latestPayment Payment
	if err := svc.db.Order("payment_date DESC").Limit(1).First(&latestPayment).Error; err == nil {
		result["latest_payment_date"] = latestPayment.PaymentDate.Format("2006-01-02")
	}

	return result
}

// getOperationsContext fetches operations pipeline data
func (svc *Service) getOperationsContext() map[string]any {
	result := make(map[string]any)

	// Offer pipeline
	var totalOffers, quotedOffers, wonOffers, lostOffers int64
	svc.db.Model(&Offer{}).Count(&totalOffers)
	svc.db.Model(&Offer{}).Where("stage = ?", "Quoted").Count(&quotedOffers)
	svc.db.Model(&Offer{}).Where("stage = ?", "Won").Count(&wonOffers)
	svc.db.Model(&Offer{}).Where("stage = ?", "Lost").Count(&lostOffers)
	result["offers"] = map[string]int64{
		"total": totalOffers, "quoted": quotedOffers, "won": wonOffers, "lost": lostOffers,
	}

	// Calculate win rate
	if wonOffers+lostOffers > 0 {
		result["win_rate"] = float64(wonOffers) / float64(wonOffers+lostOffers) * 100
	}

	// Order pipeline
	var activeOrders int64
	var orderValue float64
	svc.db.Model(&Order{}).Where("status NOT IN ?", []string{"Completed", "Cancelled"}).Count(&activeOrders)
	svc.db.Model(&Order{}).Where("status NOT IN ?", []string{"Completed", "Cancelled"}).
		Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&orderValue)
	result["active_orders"] = activeOrders
	result["active_order_value_bhd"] = orderValue

	// PO pipeline
	var activePOs int64
	var poValue float64
	svc.db.Model(&PurchaseOrder{}).Where("status NOT IN ?", []string{"Completed", "Cancelled"}).Count(&activePOs)
	svc.db.Model(&PurchaseOrder{}).Where("status NOT IN ?", []string{"Completed", "Cancelled"}).
		Select("COALESCE(SUM(total_bhd), 0)").Scan(&poValue)
	result["active_purchase_orders"] = activePOs
	result["active_po_value_bhd"] = poValue

	// Delivery notes pending
	var pendingDN int64
	svc.db.Model(&DeliveryNote{}).Where("status != ?", "Delivered").Count(&pendingDN)
	result["pending_deliveries"] = pendingDN

	// RFQ pipeline (RFQs are Offers at stage "RFQ")
	var totalRFQs int64
	svc.db.Model(&Offer{}).Where("stage = ?", "RFQ").Count(&totalRFQs)
	result["open_rfqs"] = totalRFQs

	// GRN stats
	var totalGRNs int64
	svc.db.Model(&GoodsReceivedNote{}).Count(&totalGRNs)
	result["total_grns"] = totalGRNs

	// Recent orders (last 10 for context)
	var recentOrders []Order
	svc.db.Order("created_at DESC").Limit(10).Find(&recentOrders)
	orderList := make([]map[string]any, 0)
	for _, o := range recentOrders {
		orderList = append(orderList, map[string]any{
			"number":   o.OrderNumber,
			"customer": o.CustomerName,
			"total":    o.GrandTotalBHD,
			"status":   o.Status,
			"date":     o.OrderDate.Format("2006-01-02"),
		})
	}
	result["recent_orders"] = orderList

	// Offer pipeline value
	var totalOfferValue float64
	svc.db.Model(&Offer{}).Where("stage NOT IN ?", []string{"Lost", "Cancelled"}).
		Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&totalOfferValue)
	result["active_offer_pipeline_bhd"] = totalOfferValue

	// Recent offers with item-level detail (product mix in pipeline)
	var recentOffers []Offer
	svc.db.Where("stage NOT IN ?", []string{"Lost", "Expired"}).
		Order("quotation_date DESC").Limit(5).Find(&recentOffers)
	offerDetailList := make([]map[string]any, 0)
	for _, o := range recentOffers {
		offerEntry := map[string]any{
			"number":   o.OfferNumber,
			"customer": o.CustomerName,
			"stage":    o.Stage,
			"value":    o.TotalValueBHD,
			"margin":   o.EstimatedMargin,
			"date":     o.QuotationDate.Format("2006-01-02"),
		}
		var items []OfferItem
		svc.db.Where("offer_id = ?", o.ID).Find(&items)
		itemList := make([]map[string]any, 0)
		for _, item := range items {
			itemList = append(itemList, map[string]any{
				"equipment":      item.Equipment,
				"model":          item.Model,
				"qty":            item.Quantity,
				"unit_price_bhd": item.UnitPrice,
				"margin_pct":     item.MarginPercent,
				"currency":       item.Currency,
			})
		}
		offerEntry["items"] = itemList
		offerDetailList = append(offerDetailList, offerEntry)
	}
	result["recent_offers_detail"] = offerDetailList

	// Delivery fulfillment status
	var totalDN, deliveredDN, partialDN int64
	svc.db.Model(&DeliveryNote{}).Count(&totalDN)
	svc.db.Model(&DeliveryNote{}).Where("status = ?", "Delivered").Count(&deliveredDN)
	svc.db.Model(&DeliveryNote{}).Where("is_partial_delivery = ?", true).Count(&partialDN)
	result["delivery_stats"] = map[string]any{
		"total":     totalDN,
		"delivered": deliveredDN,
		"partial":   partialDN,
	}

	// GRN quality overview
	var totalGRNItems int64
	var rejectedItems float64
	svc.db.Model(&GRNItem{}).Count(&totalGRNItems)
	svc.db.Model(&GRNItem{}).Select("COALESCE(SUM(quantity_rejected), 0)").Scan(&rejectedItems)
	result["grn_quality"] = map[string]any{
		"total_items":    totalGRNItems,
		"total_rejected": rejectedItems,
	}

	// Pending offer follow-ups (upcoming pipeline actions)
	var pendingFollowUps []OfferFollowUp
	svc.db.Where("status = ?", "pending").Order("follow_up_date ASC").Limit(5).Find(&pendingFollowUps)
	followUpList := make([]map[string]any, 0)
	for _, fu := range pendingFollowUps {
		// Get offer number for context
		var offer Offer
		offerNum := ""
		if err := svc.db.Select("offer_number, customer_name").Where("id = ?", fu.OfferID).First(&offer).Error; err == nil {
			offerNum = offer.OfferNumber
		}
		followUpList = append(followUpList, map[string]any{
			"offer":    offerNum,
			"customer": offer.CustomerName,
			"date":     fu.FollowUpDate.Format("2006-01-02"),
			"notes":    fu.Notes,
		})
	}
	result["pending_follow_ups"] = followUpList

	// Product catalog summary
	var totalProducts, activeProducts int64
	svc.db.Model(&ProductMaster{}).Count(&totalProducts)
	svc.db.Model(&ProductMaster{}).Where("is_active = ?", true).Count(&activeProducts)
	result["product_catalog"] = map[string]any{
		"total":  totalProducts,
		"active": activeProducts,
	}

	// Opportunity pipeline (Three-Regime dynamics)
	var opportunities []Opportunity
	svc.db.Where("stage NOT IN ?", []string{"Won", "Lost", "Expired"}).
		Order("confidence DESC").Limit(10).Find(&opportunities)
	oppList := make([]map[string]any, 0)
	for _, opp := range opportunities {
		oppList = append(oppList, map[string]any{
			"folder":     opp.FolderNumber,
			"customer":   opp.CustomerName,
			"stage":      opp.Stage,
			"revenue":    opp.RevenueBHD,
			"confidence": opp.Confidence,
			"regime":     opp.Regime,
		})
	}
	result["active_opportunities"] = oppList

	return result
}

// getRiskContext fetches risk-related data
func (svc *Service) getRiskContext() map[string]any {
	result := make(map[string]any)

	// Overdue invoices
	var overdueInvoices []Invoice
	svc.db.Where("status = ? OR (status = ? AND due_date < ?)", "Overdue", "Sent", time.Now()).
		Order("outstanding_bhd DESC").Limit(10).Find(&overdueInvoices)
	overdueList := make([]map[string]any, 0)
	var totalOverdue float64
	for _, inv := range overdueInvoices {
		daysOverdue := int(time.Since(inv.DueDate).Hours() / 24)
		if daysOverdue < 0 {
			daysOverdue = 0
		}
		totalOverdue += inv.OutstandingBHD
		overdueList = append(overdueList, map[string]any{
			"invoice":      inv.InvoiceNumber,
			"customer":     inv.CustomerName,
			"outstanding":  inv.OutstandingBHD,
			"days_overdue": daysOverdue,
		})
	}
	result["overdue_invoices"] = overdueList
	result["total_overdue_bhd"] = totalOverdue

	// Credit-blocked customers
	var blockedCustomers []CustomerMaster
	svc.db.Where("is_credit_blocked = ?", true).Find(&blockedCustomers)
	blockedList := make([]string, 0)
	for _, c := range blockedCustomers {
		blockedList = append(blockedList, c.BusinessName)
	}
	result["credit_blocked_customers"] = blockedList

	// High-risk AR tiers
	var criticalCount, highCount int64
	svc.db.Model(&CustomerMaster{}).Where("ar_risk_tier = ?", "Critical").Count(&criticalCount)
	svc.db.Model(&CustomerMaster{}).Where("ar_risk_tier = ?", "High").Count(&highCount)
	result["critical_risk_customers"] = criticalCount
	result["high_risk_customers"] = highCount

	// Active alerts
	var alerts []Alert
	svc.db.Where("is_active = ? AND is_acknowledged = ?", true, false).
		Order("CASE severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END").
		Limit(10).Find(&alerts)
	alertList := make([]map[string]any, 0)
	for _, al := range alerts {
		alertList = append(alertList, map[string]any{
			"type":     al.AlertType,
			"severity": al.Severity,
			"title":    al.Title,
			"message":  al.Message,
		})
	}
	result["active_alerts"] = alertList

	// Overdue supplier invoices (AP risk)
	var overdueSupInv []SupplierInvoice
	svc.db.Where("status NOT IN ? AND due_date < ?", []string{"Paid", "Rejected"}, time.Now()).
		Order("total_bhd DESC").Limit(10).Find(&overdueSupInv)
	overdueAPList := make([]map[string]any, 0)
	var totalOverdueAP float64
	for _, si := range overdueSupInv {
		daysOverdue := int(time.Since(si.DueDate).Hours() / 24)
		totalOverdueAP += si.TotalBHD
		overdueAPList = append(overdueAPList, map[string]any{
			"supplier":     si.SupplierName,
			"invoice":      si.InvoiceNumber,
			"amount":       si.TotalBHD,
			"days_overdue": daysOverdue,
		})
	}
	result["overdue_supplier_invoices"] = overdueAPList
	result["total_overdue_ap_bhd"] = totalOverdueAP

	// Expiring offers (validity ending soon - opportunity risk)
	var expiringOffers []Offer
	svc.db.Where("stage = ? AND validity_date BETWEEN ? AND ?", "Quoted",
		time.Now(), time.Now().AddDate(0, 0, 14)).
		Order("validity_date ASC").Limit(5).Find(&expiringOffers)
	expiringList := make([]map[string]any, 0)
	for _, o := range expiringOffers {
		expiringList = append(expiringList, map[string]any{
			"number":   o.OfferNumber,
			"customer": o.CustomerName,
			"value":    o.TotalValueBHD,
			"expires":  o.ValidityDate.Format("2006-01-02"),
		})
	}
	result["expiring_offers"] = expiringList

	// Overdue follow-up tasks
	var overdueTasks []FollowUpTask
	svc.db.Where("status IN ? AND due_date < ?", []string{"pending", "overdue"}, time.Now()).
		Order("due_date ASC").Limit(10).Find(&overdueTasks)
	taskList := make([]map[string]any, 0)
	for _, t := range overdueTasks {
		taskList = append(taskList, map[string]any{
			"title":    t.Title,
			"due_date": t.DueDate.Format("2006-01-02"),
			"priority": t.Priority,
			"type":     t.Type,
		})
	}
	result["overdue_tasks"] = taskList

	return result
}

// getARSummary provides accounts receivable summary
func (svc *Service) getARSummary() map[string]any {
	result := make(map[string]any)

	var totalAR, overdueAR float64
	svc.db.Model(&Invoice{}).Where("status != ?", "Paid").
		Select("COALESCE(SUM(outstanding_bhd), 0)").Scan(&totalAR)
	svc.db.Model(&Invoice{}).Where("status = ?", "Overdue").
		Select("COALESCE(SUM(outstanding_bhd), 0)").Scan(&overdueAR)

	result["total_receivables_bhd"] = totalAR
	result["overdue_receivables_bhd"] = overdueAR
	if totalAR > 0 {
		result["overdue_percentage"] = (overdueAR / totalAR) * 100
	}

	return result
}

// getBankingContext fetches bank accounts, unreconciled lines, and cheque position
func (svc *Service) getBankingContext() map[string]any {
	result := make(map[string]any)

	// Active bank accounts with balances (from latest bank statement per account)
	var accounts []CompanyBankAccount
	svc.db.Where("is_active = ?", true).Find(&accounts)
	accountList := make([]map[string]any, 0)
	var totalCashBHD float64
	for _, acc := range accounts {
		// Get closing balance from most recent statement
		var latestStmt BankStatement
		balance := 0.0
		if err := svc.db.Where("bank_account_id = ?", acc.ID).
			Order("period_end DESC").First(&latestStmt).Error; err == nil {
			balance = latestStmt.ClosingBalance
		}
		totalCashBHD += balance
		accountList = append(accountList, map[string]any{
			"name":         acc.AccountName,
			"bank":         acc.BankName,
			"currency":     acc.Currency,
			"balance":      balance,
			"booking_rate": acc.BookingRate,
		})
	}
	result["bank_accounts"] = accountList
	result["total_cash_bhd"] = totalCashBHD

	// Unreconciled statement lines (pending reconciliation work)
	var unmatchedCount int64
	var unmatchedCredits, unmatchedDebits float64
	svc.db.Model(&BankStatementLine{}).Where("is_matched = ?", false).Count(&unmatchedCount)
	svc.db.Model(&BankStatementLine{}).Where("is_matched = ? AND credit > 0", false).
		Select("COALESCE(SUM(credit), 0)").Scan(&unmatchedCredits)
	svc.db.Model(&BankStatementLine{}).Where("is_matched = ? AND debit > 0", false).
		Select("COALESCE(SUM(debit), 0)").Scan(&unmatchedDebits)
	result["unreconciled_lines"] = unmatchedCount
	result["unreconciled_credits_bhd"] = unmatchedCredits
	result["unreconciled_debits_bhd"] = unmatchedDebits

	// Outstanding cheques (ISSUED or PRESENTED — money gone but not cleared)
	var outstandingCheques []OutstandingCheque
	svc.db.Where("status IN ?", []string{"ISSUED", "PRESENTED"}).
		Order("issued_date DESC").Limit(20).Find(&outstandingCheques)
	chequeList := make([]map[string]any, 0)
	var totalOutstandingCheques float64
	for _, ch := range outstandingCheques {
		totalOutstandingCheques += ch.Amount
		chequeList = append(chequeList, map[string]any{
			"payee":   ch.PayeeName,
			"amount":  ch.Amount,
			"status":  ch.Status,
			"number":  ch.ChequeNumber,
			"date":    ch.IssuedDate.Format("2006-01-02"),
			"purpose": ch.Purpose,
		})
	}
	result["outstanding_cheques"] = chequeList
	result["total_outstanding_cheques_bhd"] = totalOutstandingCheques

	// Problem cheques (STALE or BOUNCED — need action)
	var problemCheques []OutstandingCheque
	svc.db.Where("status IN ?", []string{"STALE", "BOUNCED"}).Find(&problemCheques)
	problemList := make([]map[string]any, 0)
	for _, ch := range problemCheques {
		problemList = append(problemList, map[string]any{
			"payee":  ch.PayeeName,
			"amount": ch.Amount,
			"status": ch.Status,
			"number": ch.ChequeNumber,
		})
	}
	result["problem_cheques"] = problemList

	return result
}

// getCreditNoteContext fetches credit note pipeline (Draft → Issued → Applied)
func (svc *Service) getCreditNoteContext() map[string]any {
	result := make(map[string]any)

	// Counts by status
	var draftCount, issuedCount, appliedCount int64
	svc.db.Model(&CreditNote{}).Where("status = ?", "Draft").Count(&draftCount)
	svc.db.Model(&CreditNote{}).Where("status = ?", "Issued").Count(&issuedCount)
	svc.db.Model(&CreditNote{}).Where("status = ?", "Applied").Count(&appliedCount)
	result["credit_note_counts"] = map[string]int64{
		"draft": draftCount, "issued": issuedCount, "applied": appliedCount,
	}

	// Outstanding credit (Issued but not yet applied — reduces future AR)
	var outstandingCredit float64
	svc.db.Model(&CreditNote{}).Where("status = ?", "Issued").
		Select("COALESCE(SUM(grand_total_bhd), 0)").Scan(&outstandingCredit)
	result["outstanding_credit_bhd"] = outstandingCredit

	// Recent credit notes (last 10)
	var recentCNs []CreditNote
	svc.db.Order("credit_note_date DESC").Limit(10).Find(&recentCNs)
	cnList := make([]map[string]any, 0)
	for _, cn := range recentCNs {
		cnList = append(cnList, map[string]any{
			"number":   cn.CreditNoteNumber,
			"customer": cn.CustomerName,
			"amount":   cn.GrandTotalBHD,
			"status":   cn.Status,
			"reason":   cn.Reason,
			"date":     cn.CreditNoteDate.Format("2006-01-02"),
		})
	}
	result["recent_credit_notes"] = cnList

	return result
}

// getProductContext fetches the full product catalog with pricing and sales velocity
func (svc *Service) getProductContext() map[string]any {
	result := make(map[string]any)
	if svc.db == nil {
		return result
	}

	// Full product catalog
	var products []ProductMaster
	svc.db.Where("is_active = ?", true).Find(&products)
	productList := make([]map[string]any, 0)
	categoryMap := make(map[string]int)
	for _, p := range products {
		categoryMap[p.ProductCategory]++
		productList = append(productList, map[string]any{
			"code":           p.ProductCode,
			"name":           p.ProductName,
			"category":       p.ProductCategory,
			"standard_cost":  p.StandardCostBHD,
			"standard_price": p.StandardPriceBHD,
			"stock_qty":      p.StockQuantity,
			"supplier_code":  p.SupplierCode,
			"part_number":    p.PartNumber,
			"description":    p.Description,
		})
	}
	result["products"] = productList
	result["product_categories"] = categoryMap
	result["total_active_products"] = len(products)

	// Top products by order frequency (which products get ordered most)
	type productFreq struct {
		Equipment string
		Model     string
		Count     int64
		TotalQty  float64
	}
	var topProducts []productFreq
	svc.db.Model(&OrderItem{}).
		Select("equipment, model, COUNT(*) as count, SUM(quantity) as total_qty").
		Group("equipment, model").
		Order("count DESC").
		Limit(10).
		Scan(&topProducts)
	topList := make([]map[string]any, 0)
	for _, tp := range topProducts {
		topList = append(topList, map[string]any{
			"equipment":   tp.Equipment,
			"model":       tp.Model,
			"order_count": tp.Count,
			"total_qty":   tp.TotalQty,
		})
	}
	result["top_ordered_products"] = topList

	return result
}

// getDeliveryContext fetches active and recent delivery notes
func (svc *Service) getDeliveryContext() map[string]any {
	result := make(map[string]any)
	if svc.db == nil {
		return result
	}

	// Active deliveries (Prepared + Dispatched — in progress)
	var activeDNs []DeliveryNote
	svc.db.Where("status IN ?", []string{"Prepared", "Dispatched"}).
		Order("delivery_date ASC").Find(&activeDNs)
	activeList := make([]map[string]any, 0)
	for _, dn := range activeDNs {
		activeList = append(activeList, map[string]any{
			"dn_number":   dn.DNNumber,
			"customer_id": dn.CustomerID,
			"status":      dn.Status,
			"date":        dn.DeliveryDate.Format("2006-01-02"),
			"transport":   dn.TransportMethod,
			"driver":      dn.DriverName,
			"contact":     dn.ContactPerson,
		})
	}
	result["active_deliveries"] = activeList
	result["active_delivery_count"] = len(activeDNs)

	// Recent completed deliveries (last 20)
	var recentDNs []DeliveryNote
	svc.db.Where("status = ?", "Delivered").
		Order("delivery_date DESC").Limit(20).Find(&recentDNs)
	recentList := make([]map[string]any, 0)
	for _, dn := range recentDNs {
		recentList = append(recentList, map[string]any{
			"dn_number":   dn.DNNumber,
			"customer_id": dn.CustomerID,
			"date":        dn.DeliveryDate.Format("2006-01-02"),
			"partial":     dn.IsPartialDelivery,
		})
	}
	result["recent_deliveries"] = recentList

	// Delivery stats by transport method
	type transportStat struct {
		TransportMethod string
		Count           int64
	}
	var transportStats []transportStat
	svc.db.Model(&DeliveryNote{}).
		Select("transport_method, COUNT(*) as count").
		Group("transport_method").
		Scan(&transportStats)
	statsMap := make(map[string]int64)
	for _, ts := range transportStats {
		if ts.TransportMethod != "" {
			statsMap[ts.TransportMethod] = ts.Count
		}
	}
	result["transport_breakdown"] = statsMap

	return result
}

// getForecastContext builds forward-looking intelligence: pipeline projections,
// payment predictions, win probabilities, and revenue trends by month
func (svc *Service) getForecastContext() map[string]any {
	result := make(map[string]any)
	if svc.db == nil {
		return result
	}
	now := time.Now()
	result["as_of_date"] = now.Format("2006-01-02")
	result["current_year"] = now.Year()

	// Open-opportunity dedup rides on the sales-pipeline/OneDrive
	// normalization helpers, which stay with the host (W5-D5: don't drag
	// the neighbors) — the host returns the already-deduped open set.
	openOpportunities := svc.host.OpenDedupedOpportunities()

	// --- Pipeline by stage with weighted value ---
	type stageData struct {
		Count      int64
		TotalValue float64
	}
	stageBreakdown := make(map[string]*stageData)
	for _, opp := range openOpportunities {
		stage := strings.TrimSpace(opp.Stage)
		if stage == "" {
			stage = "New"
		}
		if stageBreakdown[stage] == nil {
			stageBreakdown[stage] = &stageData{}
		}
		stageBreakdown[stage].Count++
		stageBreakdown[stage].TotalValue += opp.RevenueBHD
	}
	pipelineStages := make([]map[string]any, 0)
	var weightedPipeline float64
	stageWeights := map[string]float64{
		"Lead": 0.10, "Qualified": 0.25, "Proposal": 0.50, "Negotiation": 0.75,
	}
	for stage, s := range stageBreakdown {
		w := stageWeights[stage]
		if w == 0 {
			w = 0.50
		}
		weighted := s.TotalValue * w
		weightedPipeline += weighted
		pipelineStages = append(pipelineStages, map[string]any{
			"stage":             stage,
			"count":             s.Count,
			"total_value_bhd":   fmt.Sprintf("%.3f", s.TotalValue),
			"weighted_bhd":      fmt.Sprintf("%.3f", weighted),
			"close_probability": fmt.Sprintf("%.0f%%", w*100),
		})
	}
	result["pipeline_by_stage"] = pipelineStages
	result["weighted_pipeline_bhd"] = fmt.Sprintf("%.3f", weightedPipeline)

	// --- Opportunities closing soon (next 60 days) ---
	closingSoon := make([]Opportunity, 0)
	for _, opp := range openOpportunities {
		if opp.ExpectedDate == nil {
			continue
		}
		if opp.ExpectedDate.Before(now) || opp.ExpectedDate.After(now.AddDate(0, 0, 60)) {
			continue
		}
		closingSoon = append(closingSoon, opp)
	}
	sort.Slice(closingSoon, func(i, j int) bool {
		return closingSoon[i].ExpectedDate.Before(*closingSoon[j].ExpectedDate)
	})
	if len(closingSoon) > 10 {
		closingSoon = closingSoon[:10]
	}
	closingList := make([]map[string]any, 0)
	for _, opp := range closingSoon {
		expectedStr := ""
		if opp.ExpectedDate != nil {
			expectedStr = opp.ExpectedDate.Format("2006-01-02")
		}
		closingList = append(closingList, map[string]any{
			"folder":      opp.FolderNumber,
			"customer":    opp.CustomerName,
			"stage":       opp.Stage,
			"revenue":     opp.RevenueBHD,
			"profit":      opp.ProfitBHD,
			"confidence":  opp.Confidence,
			"expected":    expectedStr,
			"salesperson": opp.Salesperson,
			"regime":      opp.Regime,
		})
	}
	result["closing_soon"] = closingList

	// --- Payment predictions by customer grade ---
	var predictions []PredictionRecord
	svc.db.Order("confidence DESC").Limit(20).Find(&predictions)
	predList := make([]map[string]any, 0)
	for _, p := range predictions {
		predList = append(predList, map[string]any{
			"customer":       p.CustomerName,
			"grade":          p.Grade,
			"predicted_days": p.PredictedDays,
			"confidence":     fmt.Sprintf("%.0f%%", p.Confidence*100),
		})
	}
	result["payment_predictions"] = predList

	// --- Monthly revenue trend (last 12 months from invoices) ---
	var allInvoices []Invoice
	twelveMonthsAgo := time.Now().AddDate(0, -12, 0)
	svc.db.Where("invoice_date > ? AND status IN ?", twelveMonthsAgo,
		[]string{"Sent", "Paid", "PartiallyPaid", "Overdue"}).
		Select("invoice_date, grand_total_bhd, status").Find(&allInvoices)
	monthlyRevenue := make(map[string]float64)
	for _, inv := range allInvoices {
		key := inv.InvoiceDate.Format("2006-01")
		monthlyRevenue[key] += inv.GrandTotalBHD
	}
	result["monthly_revenue_last_12m"] = monthlyRevenue

	// --- Win rate by product type (using Opportunity) ---
	type productWin struct {
		ProductType string
		Total       int64
		Won         int64
	}
	var productWins []productWin
	svc.db.Model(&Opportunity{}).
		Select("product_type, COUNT(*) as total, SUM(CASE WHEN stage = 'Won' THEN 1 ELSE 0 END) as won").
		Where("product_type != ''").
		Group("product_type").
		Scan(&productWins)
	winByProduct := make([]map[string]any, 0)
	for _, pw := range productWins {
		rate := 0.0
		if pw.Total > 0 {
			rate = float64(pw.Won) / float64(pw.Total) * 100
		}
		winByProduct = append(winByProduct, map[string]any{
			"product_type": pw.ProductType,
			"total":        pw.Total,
			"won":          pw.Won,
			"win_rate":     fmt.Sprintf("%.1f%%", rate),
		})
	}
	result["win_rate_by_product"] = winByProduct

	// --- Salesperson pipeline performance ---
	type salespersonPerf struct {
		Salesperson string
		OpenCount   int64
		TotalValue  float64
	}
	var salesPerf []salespersonPerf
	perfMap := make(map[string]*salespersonPerf)
	for _, opp := range openOpportunities {
		if strings.TrimSpace(opp.Salesperson) == "" {
			continue
		}
		if perfMap[opp.Salesperson] == nil {
			perfMap[opp.Salesperson] = &salespersonPerf{Salesperson: opp.Salesperson}
		}
		perfMap[opp.Salesperson].OpenCount++
		perfMap[opp.Salesperson].TotalValue += opp.RevenueBHD
	}
	for _, perf := range perfMap {
		salesPerf = append(salesPerf, *perf)
	}
	sort.Slice(salesPerf, func(i, j int) bool {
		return salesPerf[i].TotalValue > salesPerf[j].TotalValue
	})
	salesList := make([]map[string]any, 0)
	for _, sp := range salesPerf {
		salesList = append(salesList, map[string]any{
			"salesperson":  sp.Salesperson,
			"open_opps":    sp.OpenCount,
			"pipeline_bhd": fmt.Sprintf("%.3f", sp.TotalValue),
		})
	}
	result["salesperson_pipeline"] = salesList

	return result
}

// getSerialContext fetches serial number inventory and traceability overview
func (svc *Service) getSerialContext() map[string]any {
	result := make(map[string]any)
	if svc.db == nil {
		return result
	}

	// Status distribution
	type statusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []statusCount
	svc.db.Model(&SerialNumber{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts)
	statusMap := make(map[string]int64)
	for _, sc := range statusCounts {
		statusMap[sc.Status] = sc.Count
	}
	result["serial_status_distribution"] = statusMap

	// Shipped but not yet confirmed delivered (in-transit risk)
	var shippedNotDelivered int64
	svc.db.Model(&SerialNumber{}).Where("status = ?", "Shipped").Count(&shippedNotDelivered)
	result["serials_in_transit"] = shippedNotDelivered

	// Recently received (last 30 days)
	var recentlyReceived int64
	svc.db.Model(&SerialNumber{}).
		Where("created_at > ?", time.Now().AddDate(0, 0, -30)).
		Count(&recentlyReceived)
	result["serials_received_last_30d"] = recentlyReceived

	// Serials by product (top 10)
	type productSerial struct {
		ProductCode string
		Count       int64
	}
	var productSerials []productSerial
	svc.db.Model(&SerialNumber{}).
		Select("product_code, COUNT(*) as count").
		Group("product_code").
		Order("count DESC").
		Limit(10).
		Scan(&productSerials)
	serialsByProduct := make([]map[string]any, 0)
	for _, ps := range productSerials {
		serialsByProduct = append(serialsByProduct, map[string]any{
			"product_code": ps.ProductCode,
			"count":        ps.Count,
		})
	}
	result["serials_by_product"] = serialsByProduct

	return result
}

// getDSOContext computes Days Sales Outstanding — overall and per-customer.
// Uses payments.days_to_payment which is pre-calculated at payment recording time.
func (svc *Service) getDSOContext() map[string]any {
	ctx := make(map[string]any)
	if svc.db == nil {
		return ctx
	}

	// Overall DSO
	var overallDSO float64
	svc.db.Model(&Payment{}).
		Select("COALESCE(AVG(days_to_payment), 0)").
		Where("days_to_payment > 0").
		Scan(&overallDSO)
	ctx["overall_dso_days"] = math.Round(overallDSO*10) / 10

	// Per-customer DSO — slowest 10 and fastest 10
	type CustomerDSO struct {
		CustomerName string  `gorm:"column:customer_name"`
		AvgDays      float64 `gorm:"column:avg_days"`
		PaymentCount int     `gorm:"column:payment_count"`
	}
	var customerDSOs []CustomerDSO
	svc.db.Raw(`
		SELECT i.customer_name,
		       ROUND(AVG(p.days_to_payment), 1) as avg_days,
		       COUNT(p.id) as payment_count
		FROM payments p
		JOIN invoices i ON p.invoice_id = i.id
		WHERE p.days_to_payment > 0 AND i.deleted_at IS NULL
		GROUP BY i.customer_id, i.customer_name
		HAVING payment_count >= 2
		ORDER BY avg_days DESC
		LIMIT 20
	`).Scan(&customerDSOs)

	slowest, fastest := []map[string]any{}, []map[string]any{}
	for _, c := range customerDSOs {
		if len(slowest) < 10 {
			slowest = append(slowest, map[string]any{
				"customer": c.CustomerName, "avg_days": c.AvgDays, "payments": c.PaymentCount,
			})
		}
	}
	for i := len(customerDSOs) - 1; i >= 0 && len(fastest) < 10; i-- {
		c := customerDSOs[i]
		fastest = append(fastest, map[string]any{
			"customer": c.CustomerName, "avg_days": c.AvgDays, "payments": c.PaymentCount,
		})
	}
	ctx["slowest_payers"] = slowest
	ctx["fastest_payers"] = fastest

	return ctx
}

// getOfferExpiryContext fetches active offers approaching their ValidityDate.
// Groups by urgency: this week / this month / next 60 days.
func (svc *Service) getOfferExpiryContext() map[string]any {
	ctx := make(map[string]any)
	if svc.db == nil {
		return ctx
	}

	now := time.Now()
	in7 := now.AddDate(0, 0, 7)
	in30 := now.AddDate(0, 0, 30)
	in60 := now.AddDate(0, 0, 60)

	type ExpiringOffer struct {
		OfferNumber   string    `gorm:"column:offer_number"`
		CustomerName  string    `gorm:"column:customer_name"`
		TotalValueBHD float64   `gorm:"column:total_value_bhd"`
		ValidityDate  time.Time `gorm:"column:validity_date"`
		Stage         string    `gorm:"column:stage"`
	}
	var expiring []ExpiringOffer
	svc.db.Raw(`
		SELECT offer_number, customer_name, total_value_bhd, validity_date, stage
		FROM offers
		WHERE validity_date BETWEEN ? AND ?
		  AND stage IN ('RFQ', 'Quoted')
		  AND deleted_at IS NULL
		ORDER BY validity_date ASC
		LIMIT 50
	`, now, in60).Scan(&expiring)

	week, month, later := []map[string]any{}, []map[string]any{}, []map[string]any{}
	var totalAtRisk float64
	for _, o := range expiring {
		entry := map[string]any{
			"offer": o.OfferNumber, "customer": o.CustomerName,
			"value_bhd": o.TotalValueBHD, "expires": o.ValidityDate.Format("2006-01-02"), "stage": o.Stage,
		}
		totalAtRisk += o.TotalValueBHD
		if o.ValidityDate.Before(in7) {
			week = append(week, entry)
		} else if o.ValidityDate.Before(in30) {
			month = append(month, entry)
		} else {
			later = append(later, entry)
		}
	}
	ctx["expiring_this_week"] = week
	ctx["expiring_this_month"] = month
	ctx["expiring_30_to_60_days"] = later
	ctx["total_value_at_risk_bhd"] = math.Round(totalAtRisk*1000) / 1000
	ctx["count_expiring_60_days"] = len(expiring)

	return ctx
}

// getCustomerProfitabilityContext ranks customers by revenue and margin.
func (svc *Service) getCustomerProfitabilityContext() map[string]any {
	ctx := make(map[string]any)
	if svc.db == nil {
		return ctx
	}

	type CustomerProfit struct {
		CustomerName    string  `gorm:"column:customer_name"`
		TotalRevenueBHD float64 `gorm:"column:total_revenue"`
		TotalMarginBHD  float64 `gorm:"column:total_margin"`
		AvgMarginPct    float64 `gorm:"column:avg_margin_pct"`
		InvoiceCount    int     `gorm:"column:invoice_count"`
	}

	var topRevenue []CustomerProfit
	svc.db.Raw(`
		SELECT customer_name,
		       SUM(grand_total_bhd) as total_revenue,
		       SUM(gross_margin_bhd) as total_margin,
		       ROUND(AVG(gross_margin_percent), 2) as avg_margin_pct,
		       COUNT(*) as invoice_count
		FROM invoices
		WHERE status NOT IN ('Cancelled', 'Void', 'Proforma', 'Draft')
		  AND deleted_at IS NULL
		GROUP BY customer_id, customer_name
		ORDER BY total_revenue DESC
		LIMIT 15
	`).Scan(&topRevenue)

	revenueResult := []map[string]any{}
	for _, p := range topRevenue {
		revenueResult = append(revenueResult, map[string]any{
			"customer": p.CustomerName, "revenue_bhd": p.TotalRevenueBHD,
			"margin_bhd": p.TotalMarginBHD, "avg_margin_pct": p.AvgMarginPct,
			"invoice_count": p.InvoiceCount,
		})
	}
	ctx["top_customers_by_revenue"] = revenueResult

	// Lowest margin customers (pricing opportunities)
	var lowMargin []CustomerProfit
	svc.db.Raw(`
		SELECT customer_name,
		       SUM(grand_total_bhd) as total_revenue,
		       SUM(gross_margin_bhd) as total_margin,
		       ROUND(AVG(gross_margin_percent), 2) as avg_margin_pct,
		       COUNT(*) as invoice_count
		FROM invoices
		WHERE status NOT IN ('Cancelled', 'Void', 'Proforma', 'Draft')
		  AND deleted_at IS NULL AND gross_margin_percent > 0
		GROUP BY customer_id, customer_name
		HAVING invoice_count >= 2
		ORDER BY avg_margin_pct ASC
		LIMIT 10
	`).Scan(&lowMargin)

	lowResult := []map[string]any{}
	for _, p := range lowMargin {
		lowResult = append(lowResult, map[string]any{
			"customer": p.CustomerName, "revenue_bhd": p.TotalRevenueBHD,
			"avg_margin_pct": p.AvgMarginPct, "invoice_count": p.InvoiceCount,
		})
	}
	ctx["lowest_margin_customers"] = lowResult

	return ctx
}

// getSupplierPerformanceContext calculates delivery lead times and QC pass rates per supplier.
func (svc *Service) getSupplierPerformanceContext() map[string]any {
	ctx := make(map[string]any)
	if svc.db == nil {
		return ctx
	}

	type SupplierPerf struct {
		SupplierName string  `gorm:"column:supplier_name"`
		AvgLeadDays  float64 `gorm:"column:avg_lead_days"`
		GRNCount     int     `gorm:"column:grn_count"`
		QCPassRate   float64 `gorm:"column:qc_pass_rate"`
	}
	var perfs []SupplierPerf
	svc.db.Raw(`
		SELECT po.supplier_name,
		       ROUND(AVG(CAST((julianday(grn.received_date) - julianday(po.po_date)) AS REAL)), 1) as avg_lead_days,
		       COUNT(grn.id) as grn_count,
		       ROUND(SUM(CASE WHEN grn.qc_status = 'Passed' THEN 1 ELSE 0 END) * 100.0 / COUNT(grn.id), 1) as qc_pass_rate
		FROM goods_received_notes grn
		JOIN purchase_orders po ON grn.purchase_order_id = po.id
		WHERE po.deleted_at IS NULL AND grn.deleted_at IS NULL
		  AND grn.received_date IS NOT NULL AND po.po_date IS NOT NULL
		GROUP BY po.supplier_id, po.supplier_name
		HAVING grn_count >= 1
		ORDER BY avg_lead_days ASC
	`).Scan(&perfs)

	result := []map[string]any{}
	for _, p := range perfs {
		result = append(result, map[string]any{
			"supplier": p.SupplierName, "avg_lead_days": p.AvgLeadDays,
			"deliveries": p.GRNCount, "qc_pass_rate_pct": p.QCPassRate,
		})
	}
	ctx["supplier_delivery_performance"] = result

	return ctx
}

// getCustomerActivityContext identifies dormant customers, new customers this year,
// and customers who ordered last year but not this year.
func (svc *Service) getCustomerActivityContext() map[string]any {
	ctx := make(map[string]any)
	if svc.db == nil {
		return ctx
	}

	now := time.Now()
	sixMonthsAgo := now.AddDate(0, -6, 0)
	oneYearAgo := now.AddDate(-1, 0, 0)
	thisYearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	type CustomerActivity struct {
		CustomerName    string    `gorm:"column:customer_name"`
		LastOrderDate   time.Time `gorm:"column:last_order_date"`
		TotalOrders     int       `gorm:"column:total_orders"`
		TotalRevenueBHD float64   `gorm:"column:total_revenue"`
	}

	// Dormant: had invoices historically but nothing in last 6 months
	var dormant []CustomerActivity
	svc.db.Raw(`
		SELECT customer_name,
		       MAX(invoice_date) as last_order_date,
		       COUNT(*) as total_orders,
		       SUM(grand_total_bhd) as total_revenue
		FROM invoices
		WHERE status NOT IN ('Cancelled', 'Void', 'Proforma', 'Draft') AND deleted_at IS NULL
		GROUP BY customer_id, customer_name
		HAVING last_order_date < ?
		ORDER BY total_revenue DESC
		LIMIT 15
	`, sixMonthsAgo).Scan(&dormant)

	dormantResult := []map[string]any{}
	for _, c := range dormant {
		dormantResult = append(dormantResult, map[string]any{
			"customer": c.CustomerName, "last_order": c.LastOrderDate.Format("2006-01-02"),
			"days_silent":            int(now.Sub(c.LastOrderDate).Hours() / 24),
			"historical_revenue_bhd": c.TotalRevenueBHD,
		})
	}
	ctx["dormant_customers"] = dormantResult

	// New customers who placed their first-ever order this year
	type NewCust struct {
		CustomerName string    `gorm:"column:customer_name"`
		FirstOrder   time.Time `gorm:"column:first_order"`
		OrderCount   int       `gorm:"column:order_count"`
		RevenueBHD   float64   `gorm:"column:revenue_bhd"`
	}
	var newCustomers []NewCust
	svc.db.Raw(`
		SELECT customer_name,
		       MIN(invoice_date) as first_order,
		       COUNT(*) as order_count,
		       SUM(grand_total_bhd) as revenue_bhd
		FROM invoices
		WHERE status NOT IN ('Cancelled', 'Void', 'Proforma', 'Draft') AND deleted_at IS NULL
		GROUP BY customer_id, customer_name
		HAVING first_order >= ?
		ORDER BY revenue_bhd DESC
		LIMIT 10
	`, thisYearStart).Scan(&newCustomers)

	newResult := []map[string]any{}
	for _, c := range newCustomers {
		newResult = append(newResult, map[string]any{
			"customer": c.CustomerName, "first_order": c.FirstOrder.Format("2006-01-02"),
			"orders": c.OrderCount, "revenue_bhd": c.RevenueBHD,
		})
	}
	ctx["new_customers_this_year"] = newResult

	// Lapsed: ordered last year but not placed anything yet this year
	var lapsed []CustomerActivity
	svc.db.Raw(`
		SELECT customer_name,
		       MAX(invoice_date) as last_order_date,
		       COUNT(*) as total_orders,
		       SUM(grand_total_bhd) as total_revenue
		FROM invoices
		WHERE invoice_date BETWEEN ? AND ?
		  AND status NOT IN ('Cancelled', 'Void', 'Proforma', 'Draft') AND deleted_at IS NULL
		  AND customer_id NOT IN (
		      SELECT DISTINCT customer_id FROM invoices
		      WHERE invoice_date >= ?
		        AND status NOT IN ('Cancelled', 'Void', 'Proforma', 'Draft')
		        AND deleted_at IS NULL
		  )
		GROUP BY customer_id, customer_name
		ORDER BY total_revenue DESC
		LIMIT 10
	`, oneYearAgo, thisYearStart, thisYearStart).Scan(&lapsed)

	lapsedResult := []map[string]any{}
	for _, c := range lapsed {
		lapsedResult = append(lapsedResult, map[string]any{
			"customer":              c.CustomerName,
			"last_order":            c.LastOrderDate.Format("2006-01-02"),
			"prev_year_revenue_bhd": c.TotalRevenueBHD,
		})
	}
	ctx["lapsed_this_year"] = lapsedResult

	return ctx
}

// getActionItemsContext builds a prioritized list of items requiring attention right now.
func (svc *Service) getActionItemsContext() map[string]any {
	ctx := make(map[string]any)
	if svc.db == nil {
		return ctx
	}

	now := time.Now()
	in7Days := now.AddDate(0, 0, 7)
	sevenDaysAgo := now.AddDate(0, 0, -7)
	fourteenDaysAgo := now.AddDate(0, 0, -14)

	// 1. Overdue invoices to chase (top 10 by amount)
	type OverdueInv struct {
		CustomerName   string    `gorm:"column:customer_name"`
		InvoiceNumber  string    `gorm:"column:invoice_number"`
		OutstandingBHD float64   `gorm:"column:outstanding_bhd"`
		DueDate        time.Time `gorm:"column:due_date"`
	}
	var overdueInvoices []OverdueInv
	svc.db.Raw(`
		SELECT customer_name, invoice_number, outstanding_bhd, due_date
		FROM invoices
		WHERE status IN ('Overdue', 'Sent', 'PartiallyPaid')
		  AND due_date < ? AND outstanding_bhd > 0 AND deleted_at IS NULL
		ORDER BY outstanding_bhd DESC LIMIT 10
	`, now).Scan(&overdueInvoices)

	overdueResult := []map[string]any{}
	for _, inv := range overdueInvoices {
		overdueResult = append(overdueResult, map[string]any{
			"customer": inv.CustomerName, "invoice": inv.InvoiceNumber,
			"outstanding_bhd": inv.OutstandingBHD,
			"days_overdue":    int(now.Sub(inv.DueDate).Hours() / 24),
		})
	}
	ctx["overdue_invoices_to_chase"] = overdueResult

	// 2. Offers expiring this week
	type ExpiryItem struct {
		OfferNumber   string    `gorm:"column:offer_number"`
		CustomerName  string    `gorm:"column:customer_name"`
		TotalValueBHD float64   `gorm:"column:total_value_bhd"`
		ValidityDate  time.Time `gorm:"column:validity_date"`
	}
	var expiringOffers []ExpiryItem
	svc.db.Raw(`
		SELECT offer_number, customer_name, total_value_bhd, validity_date
		FROM offers
		WHERE validity_date BETWEEN ? AND ? AND stage IN ('RFQ', 'Quoted') AND deleted_at IS NULL
		ORDER BY validity_date ASC LIMIT 10
	`, now, in7Days).Scan(&expiringOffers)

	expiryResult := []map[string]any{}
	for _, o := range expiringOffers {
		expiryResult = append(expiryResult, map[string]any{
			"offer": o.OfferNumber, "customer": o.CustomerName,
			"value_bhd": o.TotalValueBHD, "expires": o.ValidityDate.Format("2006-01-02"),
		})
	}
	ctx["offers_expiring_this_week"] = expiryResult

	// 3. Delivery notes stuck as Prepared for >7 days
	type StuckDN struct {
		DNNumber  string    `gorm:"column:dn_number"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}
	var stuckDNs []StuckDN
	svc.db.Raw(`
		SELECT dn_number, created_at FROM delivery_notes
		WHERE status = 'Prepared' AND created_at < ? AND deleted_at IS NULL
		ORDER BY created_at ASC LIMIT 10
	`, sevenDaysAgo).Scan(&stuckDNs)

	stuckResult := []map[string]any{}
	for _, dn := range stuckDNs {
		stuckResult = append(stuckResult, map[string]any{
			"dn": dn.DNNumber, "days_pending": int(now.Sub(dn.CreatedAt).Hours() / 24),
		})
	}
	ctx["stuck_delivery_notes"] = stuckResult

	// 4. POs sent but not acknowledged for >14 days
	type SentPO struct {
		PONumber     string    `gorm:"column:po_number"`
		SupplierName string    `gorm:"column:supplier_name"`
		TotalBHD     float64   `gorm:"column:total_bhd"`
		UpdatedAt    time.Time `gorm:"column:updated_at"`
	}
	var sentPOs []SentPO
	svc.db.Raw(`
		SELECT po_number, supplier_name, total_bhd, updated_at
		FROM purchase_orders
		WHERE status = 'Sent' AND updated_at < ? AND deleted_at IS NULL
		ORDER BY total_bhd DESC LIMIT 10
	`, fourteenDaysAgo).Scan(&sentPOs)

	sentPOResult := []map[string]any{}
	for _, po := range sentPOs {
		sentPOResult = append(sentPOResult, map[string]any{
			"po": po.PONumber, "supplier": po.SupplierName,
			"value_bhd": po.TotalBHD, "days_waiting": int(now.Sub(po.UpdatedAt).Hours() / 24),
		})
	}
	ctx["pos_awaiting_acknowledgment"] = sentPOResult

	// 5. Supplier invoices pending approval for >7 days
	type PendingSupInv struct {
		InvoiceNumber string    `gorm:"column:invoice_number"`
		SupplierName  string    `gorm:"column:supplier_name"`
		TotalBHD      float64   `gorm:"column:total_bhd"`
		InvoiceDate   time.Time `gorm:"column:invoice_date"`
	}
	var pendingSI []PendingSupInv
	svc.db.Raw(`
		SELECT invoice_number, supplier_name, total_bhd, invoice_date
		FROM supplier_invoices
		WHERE status = 'Pending' AND invoice_date < ? AND deleted_at IS NULL
		ORDER BY total_bhd DESC LIMIT 10
	`, sevenDaysAgo).Scan(&pendingSI)

	pendingSIResult := []map[string]any{}
	for _, si := range pendingSI {
		pendingSIResult = append(pendingSIResult, map[string]any{
			"invoice": si.InvoiceNumber, "supplier": si.SupplierName,
			"value_bhd": si.TotalBHD, "days_pending": int(now.Sub(si.InvoiceDate).Hours() / 24),
		})
	}
	ctx["supplier_invoices_pending_approval"] = pendingSIResult

	return ctx
}

// getCompetitionContext analyzes win/loss patterns, ABB competition, and lost deal reasons.
func (svc *Service) getCompetitionContext() map[string]any {
	ctx := make(map[string]any)
	if svc.db == nil {
		return ctx
	}

	var totalOffers, abbOffers, abbWins, totalWins int64
	svc.db.Model(&Offer{}).Where("deleted_at IS NULL").Count(&totalOffers)
	svc.db.Model(&Offer{}).Where("has_abb_competition = ? AND deleted_at IS NULL", true).Count(&abbOffers)
	svc.db.Model(&Offer{}).Where("has_abb_competition = ? AND stage = ? AND deleted_at IS NULL", true, "Won").Count(&abbWins)
	svc.db.Model(&Offer{}).Where("stage = ? AND deleted_at IS NULL", "Won").Count(&totalWins)

	var abbWinRate, overallWinRate float64
	if abbOffers > 0 {
		abbWinRate = math.Round(float64(abbWins)/float64(abbOffers)*1000) / 10
	}
	if totalOffers > 0 {
		overallWinRate = math.Round(float64(totalWins)/float64(totalOffers)*1000) / 10
	}
	ctx["total_offers"] = totalOffers
	ctx["abb_competitive_offers"] = abbOffers
	ctx["win_rate_vs_abb_pct"] = abbWinRate
	ctx["overall_win_rate_pct"] = overallWinRate

	// Top lost reasons
	type LostReason struct {
		Reason string `gorm:"column:lost_reason"`
		Count  int    `gorm:"column:cnt"`
	}
	var lostReasons []LostReason
	svc.db.Raw(`
		SELECT lost_reason, COUNT(*) as cnt
		FROM offers
		WHERE stage = 'Lost' AND lost_reason != '' AND lost_reason IS NOT NULL AND deleted_at IS NULL
		GROUP BY lost_reason ORDER BY cnt DESC LIMIT 10
	`).Scan(&lostReasons)

	reasonResult := []map[string]any{}
	for _, r := range lostReasons {
		reasonResult = append(reasonResult, map[string]any{"reason": r.Reason, "count": r.Count})
	}
	ctx["top_lost_reasons"] = reasonResult

	// Win rate by product code (requires joining offer_items → offers)
	type ProductWinRate struct {
		ProductCode string `gorm:"column:product_code"`
		TotalOffers int    `gorm:"column:total_offers"`
		WonOffers   int    `gorm:"column:won_offers"`
	}
	var productWinRates []ProductWinRate
	svc.db.Raw(`
		SELECT oi.product_code,
		       COUNT(DISTINCT o.id) as total_offers,
		       SUM(CASE WHEN o.stage = 'Won' THEN 1 ELSE 0 END) as won_offers
		FROM offer_items oi
		JOIN offers o ON oi.offer_id = o.id
		WHERE o.deleted_at IS NULL AND oi.deleted_at IS NULL AND oi.product_code != ''
		GROUP BY oi.product_code HAVING total_offers >= 3
		ORDER BY won_offers * 1.0 / total_offers DESC LIMIT 10
	`).Scan(&productWinRates)

	productResult := []map[string]any{}
	for _, p := range productWinRates {
		winRate := 0.0
		if p.TotalOffers > 0 {
			winRate = math.Round(float64(p.WonOffers)/float64(p.TotalOffers)*1000) / 10
		}
		productResult = append(productResult, map[string]any{
			"product": p.ProductCode, "win_rate_pct": winRate,
			"total_offers": p.TotalOffers, "won": p.WonOffers,
		})
	}
	ctx["win_rate_by_product"] = productResult

	return ctx
}

// getPurchaseOrderSummary provides PO overview
func (svc *Service) getPurchaseOrderSummary() map[string]any {
	result := make(map[string]any)

	var totalPOs int64
	var totalValue float64
	svc.db.Model(&PurchaseOrder{}).Count(&totalPOs)
	svc.db.Model(&PurchaseOrder{}).Select("COALESCE(SUM(total_bhd), 0)").Scan(&totalValue)

	result["total_pos"] = totalPOs
	result["total_value_bhd"] = totalValue

	return result
}

// ============================================================================
// THREE-REGIME CALCULATION
// ============================================================================

// calculateSystemRegime determines current system's three-regime state
func (svc *Service) CalculateSystemRegime() map[string]any {
	if svc.db == nil {
		return map[string]any{
			"dominant": "R3", "r1": 0.10, "r2": 0.20, "r3": 0.70, "state": "Stabilization",
		}
	}

	// R1 indicators: new activity, exploration
	var recentOrders int64
	svc.db.Model(&Order{}).Where("created_at > ?", time.Now().AddDate(0, 0, -30)).Count(&recentOrders)
	var recentCustomers int64
	svc.db.Model(&CustomerMaster{}).Where("created_at > ?", time.Now().AddDate(0, 0, -30)).Count(&recentCustomers)
	r1Score := float64(recentOrders)*0.6 + float64(recentCustomers)*0.4

	// R2 indicators: optimization opportunity
	var avgMargin float64
	svc.db.Model(&Invoice{}).Where("gross_margin_percent > 0").
		Select("COALESCE(AVG(gross_margin_percent), 18)").Scan(&avgMargin)
	r2Score := avgMargin

	// R3 indicators: stability/collection pressure
	var overdueCount int64
	svc.db.Model(&Invoice{}).Where("status = ?", "Overdue").Count(&overdueCount)
	var totalInvoices int64
	svc.db.Model(&Invoice{}).Where("status != ?", "Paid").Count(&totalInvoices)
	r3Score := 50.0
	if totalInvoices > 0 {
		overdueRatio := float64(overdueCount) / float64(totalInvoices)
		r3Score = 50.0 + overdueRatio*50.0 // Higher overdue = more stabilization needed
	}

	// Normalize
	total := r1Score + r2Score + r3Score
	var r1, r2, r3 float64
	if total > 0 {
		r1 = r1Score / total
		r2 = r2Score / total
		r3 = r3Score / total
	} else {
		r1, r2, r3 = 0.10, 0.20, 0.70
	}

	// Determine dominant
	dominant := "R3"
	state := "Stabilization"
	if r1 > r2 && r1 > r3 {
		dominant = "R1"
		state = "Exploration"
	} else if r2 > r1 && r2 > r3 {
		dominant = "R2"
		state = "Optimization"
	}

	return map[string]any{
		"dominant": dominant,
		"r1":       r1,
		"r2":       r2,
		"r3":       r3,
		"state":    state,
	}
}

// ============================================================================
