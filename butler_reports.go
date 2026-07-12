// ═══════════════════════════════════════════════════════════════════════════
// BUTLER INTELLIGENCE REPORTS - AI-POWERED PDF GENERATION
//
// MISSION: Generate detailed business intelligence reports via Mistral AI
//          combining live database context with AI analysis into professional PDFs
//
// REPORT TYPES:
//   - customer: Customer analysis (specific or portfolio)
//   - financial: Financial health and cash flow
//   - risk: Risk assessment and overdue analysis
//   - operations: Pipeline and fulfillment status
//   - supplier: Supplier performance and procurement
//   - executive: Executive summary (all domains)
//
// PIPELINE:
//   1. Classify report type from user request
//   2. Build rich context (GORM queries)
//   3. Call Mistral for structured analysis
//   4. Generate multi-page PDF (cover + data + analysis)
//   5. Return file path for download
//
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	butlerreports "ph_holdings_app/pkg/butler/reports"
)

// ============================================================================
// REPORT GENERATION ENTRY POINT
// ============================================================================

// GenerateButlerReport creates a detailed PDF intelligence report
// reportType: customer, financial, risk, operations, supplier, executive
// query: optional context (e.g., customer name, specific question)
func (a *App) GenerateButlerReport(reportType string, query string) (string, error) {
	log.Printf("📊 Generating intelligence report: type=%s query=%q", reportType, query)

	// P1 FIX: Rate limiting for report generation (3 per minute per user)
	if GlobalRateLimiter != nil {
		rateLimitKey := "report:" + a.getCurrentUserID()
		if !GlobalRateLimiter.Allow(rateLimitKey, RateLimitConfig.ReportGenerationPerMinute, 1*time.Minute/time.Duration(RateLimitConfig.ReportGenerationPerMinute)) {
			log.Printf("🚫 Rate limit exceeded for report generation by user %s", a.getCurrentUserID())
			return "", fmt.Errorf("too many report requests, please wait a moment before generating another report")
		}
	}

	// 0. RBAC: Check if report type requires finance:view permission
	requiresFinanceAccess := butlerreports.RequiresFinanceAccess(reportType)

	if requiresFinanceAccess {
		userRole := a.GetCurrentUserRole()
		hasFinanceAccess := a.currentSessionHasPermission("finance:view")

		if !hasFinanceAccess {
			log.Printf("🔒 Butler report generation blocked: type=%s user_role=%s lacks finance:view", reportType, userRole)
			return "", fmt.Errorf("permission denied: %s reports require manager or admin privileges (finance:view permission)", reportType)
		}
	}

	// 1. Build intent from report type
	intent := Intent{
		Domain:     reportType,
		EntityName: query,
		Confidence: 1.0,
		IsComplex:  true, // Reports always use large model
	}
	intent.Domain = butlerreports.DomainForReportType(reportType)

	// 2. Build rich context
	context := a.buildIntentContext(intent)

	// 3. Call Mistral for structured analysis
	contextJSON := marshalContextForPrompt(context, butlerMaxReportContextChars)
	reportPrompt := buildReportPrompt(reportType, query, contextJSON)

	aiAnalysis, err := callMistral(mistralModelLarge, reportPrompt, query)
	if err != nil {
		log.Printf("❌ Mistral report analysis failed: %v", err)
		// Generate report without AI analysis
		aiAnalysis = "AI analysis unavailable. Report contains raw data only."
	}

	// 4. Generate PDF
	filePath, err := a.generateIntelligenceReportPDF(reportType, query, context, aiAnalysis)
	if err != nil {
		return "", fmt.Errorf("PDF generation failed: %w", err)
	}

	log.Printf("✅ Intelligence report generated: %s", filePath)
	return filePath, nil
}

// ============================================================================
// REPORT PROMPT BUILDER
// ============================================================================

func buildReportPrompt(reportType, query, contextJSON string) string {
	return butlerreports.BuildPrompt(reportType, query, contextJSON)
}

// ============================================================================
// PDF GENERATION
// ============================================================================

// letterheadImagePath returns the default-division letterhead path (the active
// overlay's default division). Division-aware document flows should call
// letterheadImagePathForDivision directly.
func (a *App) letterheadImagePath() string {
	return a.letterheadImagePathForDivision(activeOverlay.DefaultDivision())
}

// addLetterheadBackground places the full letterhead template as a page background
func (a *App) addLetterheadBackground(pdf *gofpdf.Fpdf) {
	a.applyLetterheadForDivision(pdf, activeOverlay.DefaultDivision())
	// Set cursor below the header area (logo is ~35mm tall)
	pdf.SetY(42)
}

// addLetterheadFallback renders a simple text letterhead when image is unavailable
func addLetterheadFallback(pdf *gofpdf.Fpdf) {
	addLetterheadFallbackForDivision(pdf, activeOverlay.DefaultDivision())
}

func (a *App) generateIntelligenceReportPDF(reportType, query string, context map[string]any, aiAnalysis string) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTopMargin(50)           // Start content below letterhead header (~45mm tall)
	pdf.SetAutoPageBreak(true, 35) // Bottom margin: 35mm to avoid letterhead footer
	pdf.SetLeftMargin(18)
	pdf.SetRightMargin(18)

	// Set header function so letterhead appears on ALL pages including auto-break pages
	pdf.SetHeaderFuncMode(func() {
		a.applyLetterheadForDivision(pdf, activeOverlay.DefaultDivision())
	}, true) // true = allow content to overlap the header area

	// === COVER PAGE ===
	pdf.AddPage()
	butlerreports.AddCoverPage(pdf, reportType, query)

	// === CONTENT PAGES (data + analysis flow continuously, no forced page breaks) ===
	pdf.AddPage()

	// Data section
	hasData := a.hasDataForReport(reportType, context)
	if hasData {
		a.addDataPage(pdf, reportType, context)
	}

	// AI Analysis section (flows right after data, no new page)
	cleanAnalysis := strings.TrimSpace(aiAnalysis)
	if cleanAnalysis != "" && cleanAnalysis != "AI analysis unavailable. Report contains raw data only." {
		// Add a divider between data and analysis
		if hasData {
			pdf.Ln(6)
			pdf.SetDrawColor(200, 200, 200)
			pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
			pdf.Ln(6)
		}
		butlerreports.AddAnalysisPage(pdf, cleanAnalysis)
	}

	// === PAGE NUMBERS (positioned above letterhead footer) ===
	totalPages := pdf.PageCount()
	for i := 1; i <= totalPages; i++ {
		pdf.SetPage(i)
		pdf.SetY(260) // Just above the letterhead footer area
		pdf.SetFont("Helvetica", "I", 7)
		pdf.SetTextColor(160, 160, 160)
		pdf.CellFormat(0, 5, fmt.Sprintf("Intelligence Report  |  Page %d of %d  |  Confidential", i, totalPages), "", 0, "C", false, 0, "")
	}

	// Save
	exportDir := a.getExportDir("report", "", "", time.Now().Year())

	timestamp := time.Now().Format("20060102_150405")
	// Sanitize query: only allow alphanumeric, underscore, and hyphen (prevent path traversal)
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	safeQuery := reg.ReplaceAllString(query, "_")
	if len(safeQuery) > 30 {
		safeQuery = safeQuery[:30]
	}
	if safeQuery == "" {
		safeQuery = "general"
	}
	filename := fmt.Sprintf("intelligence_%s_%s_%s.pdf", reportType, safeQuery, timestamp)
	filePath := filepath.Join(exportDir, filename)

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", fmt.Errorf("failed to save PDF: %w", err)
	}

	return filePath, nil
}

// hasDataForReport checks if we actually have data to render for this report type
func (a *App) hasDataForReport(reportType string, context map[string]any) bool {
	return butlerreports.HasDataForReport(reportType, context)
}

// addCoverPage creates a professional cover page (letterhead already rendered above)
func addCoverPage(pdf *gofpdf.Fpdf, reportType, query string) {
	// Report title - large and prominent
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetTextColor(29, 29, 31)
	reportTitle := getReportTitle(reportType)
	pdf.Cell(0, 10, reportTitle)
	pdf.Ln(14)

	if query != "" {
		pdf.SetFont("Helvetica", "", 13)
		pdf.SetTextColor(60, 60, 65)
		pdf.Cell(0, 8, fmt.Sprintf("Subject: %s", sanitizeForPDF(query)))
		pdf.Ln(14)
	}

	// Metadata block
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(40, 6, "Date:")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(0, 6, time.Now().Format("2 January 2006"))
	pdf.Ln(7)

	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(40, 6, "Period:")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(0, 6, fmt.Sprintf("Year to Date %d", time.Now().Year()))
	pdf.Ln(7)

	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(40, 6, "Classification:")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(0, 6, "Internal - Confidential")
	pdf.Ln(7)

	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(40, 6, "Prepared by:")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(0, 6, activeOverlay.CompanyDisplayName+" Business Intelligence")
	pdf.Ln(14)

	// Disclaimer
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(130, 130, 130)
	pdf.MultiCell(0, 4, "This report is generated from live ERP data and AI-powered analysis. All figures are derived from actual transaction records. Intended for internal management use only.", "", "", false)
}

func getReportTitle(reportType string) string {
	return butlerreports.ReportTitle(reportType)
}

// addDataPage adds raw business data tables
func (a *App) addDataPage(pdf *gofpdf.Fpdf, reportType string, context map[string]any) {
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 7, "Key Business Data")
	pdf.Ln(8)

	// Business summary (always present)
	if summary, ok := context["business_summary"].(map[string]any); ok {
		addSectionHeader(pdf, "Company Overview")
		addMetricRow(pdf, "Total Customers", fmt.Sprintf("%v", summary["total_customers"]))
		addMetricRow(pdf, "Total Suppliers", fmt.Sprintf("%v", summary["total_suppliers"]))
		addMetricRow(pdf, "Total Invoices", fmt.Sprintf("%v", summary["total_invoices"]))
		addMetricRow(pdf, "Total Orders", fmt.Sprintf("%v", summary["total_orders"]))
		if rev, ok := summary["total_revenue_bhd"].(float64); ok {
			addMetricRow(pdf, "Total Revenue", fmt.Sprintf("%.3f BHD", rev))
		}
		if out, ok := summary["total_outstanding_bhd"].(float64); ok {
			addMetricRow(pdf, "Total Outstanding", fmt.Sprintf("%.3f BHD", out))
		}
		if grades, ok := summary["grade_distribution"].(map[string]int64); ok {
			addMetricRow(pdf, "Grade A Customers", fmt.Sprintf("%d", grades["A"]))
			addMetricRow(pdf, "Grade B Customers", fmt.Sprintf("%d", grades["B"]))
			addMetricRow(pdf, "Grade C Customers", fmt.Sprintf("%d", grades["C"]))
			addMetricRow(pdf, "Grade D Customers", fmt.Sprintf("%d", grades["D"]))
		}
		pdf.Ln(4)
	}

	// Domain-specific data
	switch reportType {
	case "customer":
		a.addCustomerDataToPDF(pdf, context)
	case "financial":
		a.addFinancialDataToPDF(pdf, context)
	case "risk":
		a.addRiskDataToPDF(pdf, context)
	case "operations":
		a.addOperationsDataToPDF(pdf, context)
	case "supplier":
		a.addSupplierDataToPDF(pdf, context)
	case "executive":
		// Add all sections
		a.addFinancialDataToPDF(pdf, context)
		a.addOperationsDataToPDF(pdf, context)
		a.addRiskDataToPDF(pdf, context)
	}
}

func (a *App) addCustomerDataToPDF(pdf *gofpdf.Fpdf, context map[string]any) {
	if custData, ok := context["customer_data"].(map[string]any); ok {
		if customer, ok := custData["customer"].(map[string]any); ok {
			addSectionHeader(pdf, fmt.Sprintf("Customer: %v", customer["name"]))
			addMetricRow(pdf, "Grade", fmt.Sprintf("%v", customer["grade"]))
			addMetricRow(pdf, "Payment Grade", fmt.Sprintf("%v", customer["payment_grade"]))
			addMetricRow(pdf, "Avg Payment Days", fmt.Sprintf("%.0f", customer["avg_payment_days"]))
			addMetricRow(pdf, "Outstanding", fmt.Sprintf("%.3f BHD", customer["outstanding_bhd"]))
			addMetricRow(pdf, "Overdue Days", fmt.Sprintf("%v", customer["overdue_days"]))
			addMetricRow(pdf, "Total Orders", fmt.Sprintf("%v", customer["total_orders"]))
			addMetricRow(pdf, "Total Value", fmt.Sprintf("%.3f BHD", customer["total_value"]))
			addMetricRow(pdf, "AR Risk Tier", fmt.Sprintf("%v", customer["ar_risk_tier"]))
			pdf.Ln(6)
		}

		// Top outstanding customers
		if topCusts, ok := custData["top_outstanding_customers"].([]map[string]any); ok && len(topCusts) > 0 {
			addSectionHeader(pdf, "Top Outstanding Customers")
			addTableHeader(pdf, []string{"Customer", "Outstanding (BHD)", "Grade", "Overdue"})
			for _, c := range topCusts {
				addTableRow(pdf, []string{
					fmt.Sprintf("%v", c["name"]),
					fmt.Sprintf("%.3f", c["outstanding"]),
					fmt.Sprintf("%v", c["grade"]),
					fmt.Sprintf("%v days", c["overdue"]),
				})
			}
			pdf.Ln(6)
		}
	}

	if ar, ok := context["ar_summary"].(map[string]any); ok {
		addSectionHeader(pdf, "Accounts Receivable Summary")
		addMetricRow(pdf, "Total Receivables", fmt.Sprintf("%.3f BHD", ar["total_receivables_bhd"]))
		addMetricRow(pdf, "Overdue Receivables", fmt.Sprintf("%.3f BHD", ar["overdue_receivables_bhd"]))
		if pct, ok := ar["overdue_percentage"].(float64); ok {
			addMetricRow(pdf, "Overdue %", fmt.Sprintf("%.1f%%", pct))
		}
		pdf.Ln(3)
	}
}

func (a *App) addFinancialDataToPDF(pdf *gofpdf.Fpdf, context map[string]any) {
	if finData, ok := context["financial_data"].(map[string]any); ok {
		addSectionHeader(pdf, "Financial Metrics")
		addMetricRow(pdf, "Total Invoiced", fmt.Sprintf("%.3f BHD", finData["total_invoiced_bhd"]))
		addMetricRow(pdf, "Total Paid", fmt.Sprintf("%.3f BHD", finData["total_paid_bhd"]))
		addMetricRow(pdf, "Total Outstanding", fmt.Sprintf("%.3f BHD", finData["total_outstanding_bhd"]))
		addMetricRow(pdf, "Avg Days to Payment", fmt.Sprintf("%.0f days", finData["avg_days_to_payment"]))

		if counts, ok := finData["invoice_counts"].(map[string]int64); ok {
			addMetricRow(pdf, "Invoices Sent", fmt.Sprintf("%d", counts["sent"]))
			addMetricRow(pdf, "Invoices Paid", fmt.Sprintf("%d", counts["paid"]))
			addMetricRow(pdf, "Invoices Overdue", fmt.Sprintf("%d", counts["overdue"]))
		}
		pdf.Ln(3)
	}
}

func (a *App) addRiskDataToPDF(pdf *gofpdf.Fpdf, context map[string]any) {
	if riskData, ok := context["risk_data"].(map[string]any); ok {
		addSectionHeader(pdf, "Risk Assessment")
		addMetricRow(pdf, "Total Overdue", fmt.Sprintf("%.3f BHD", riskData["total_overdue_bhd"]))
		addMetricRow(pdf, "Critical Risk Customers", fmt.Sprintf("%v", riskData["critical_risk_customers"]))
		addMetricRow(pdf, "High Risk Customers", fmt.Sprintf("%v", riskData["high_risk_customers"]))

		if blocked, ok := riskData["credit_blocked_customers"].([]string); ok && len(blocked) > 0 {
			addMetricRow(pdf, "Credit Blocked", strings.Join(blocked, ", "))
		}

		// Overdue invoices table
		if overdueList, ok := riskData["overdue_invoices"].([]map[string]any); ok && len(overdueList) > 0 {
			pdf.Ln(4)
			addSectionHeader(pdf, "Overdue Invoices Detail")
			addTableHeader(pdf, []string{"Invoice", "Customer", "Outstanding", "Days Overdue"})
			for _, inv := range overdueList {
				addTableRow(pdf, []string{
					fmt.Sprintf("%v", inv["invoice"]),
					fmt.Sprintf("%v", inv["customer"]),
					fmt.Sprintf("%.3f BHD", inv["outstanding"]),
					fmt.Sprintf("%v", inv["days_overdue"]),
				})
			}
		}
		pdf.Ln(3)
	}
}

func (a *App) addOperationsDataToPDF(pdf *gofpdf.Fpdf, context map[string]any) {
	if opsData, ok := context["operations_data"].(map[string]any); ok {
		addSectionHeader(pdf, "Operations Pipeline")
		if offers, ok := opsData["offers"].(map[string]int64); ok {
			addMetricRow(pdf, "Total Offers", fmt.Sprintf("%d", offers["total"]))
			addMetricRow(pdf, "Quoted", fmt.Sprintf("%d", offers["quoted"]))
			addMetricRow(pdf, "Won", fmt.Sprintf("%d", offers["won"]))
			addMetricRow(pdf, "Lost", fmt.Sprintf("%d", offers["lost"]))
		}
		if wr, ok := opsData["win_rate"].(float64); ok {
			addMetricRow(pdf, "Win Rate", fmt.Sprintf("%.1f%%", wr))
		}
		addMetricRow(pdf, "Active Orders", fmt.Sprintf("%v", opsData["active_orders"]))
		addMetricRow(pdf, "Active Order Value", fmt.Sprintf("%.3f BHD", opsData["active_order_value_bhd"]))
		addMetricRow(pdf, "Active POs", fmt.Sprintf("%v", opsData["active_purchase_orders"]))
		addMetricRow(pdf, "Active PO Value", fmt.Sprintf("%.3f BHD", opsData["active_po_value_bhd"]))
		addMetricRow(pdf, "Pending Deliveries", fmt.Sprintf("%v", opsData["pending_deliveries"]))
		pdf.Ln(3)
	}
}

func (a *App) addSupplierDataToPDF(pdf *gofpdf.Fpdf, context map[string]any) {
	if suppData, ok := context["supplier_data"].(map[string]any); ok {
		if supplier, ok := suppData["supplier"].(map[string]any); ok {
			addSectionHeader(pdf, fmt.Sprintf("Supplier: %v", supplier["name"]))
			addMetricRow(pdf, "Type", fmt.Sprintf("%v", supplier["type"]))
			addMetricRow(pdf, "Country", fmt.Sprintf("%v", supplier["country"]))
			addMetricRow(pdf, "Rating", fmt.Sprintf("%v / 5", supplier["rating"]))
			addMetricRow(pdf, "Lead Time", fmt.Sprintf("%v days", supplier["lead_time_days"]))
			addMetricRow(pdf, "Brands", fmt.Sprintf("%v", supplier["brands"]))
			pdf.Ln(6)
		}

		if topSupp, ok := suppData["top_suppliers"].([]map[string]any); ok && len(topSupp) > 0 {
			addSectionHeader(pdf, "Top Suppliers by PO Value")
			addTableHeader(pdf, []string{"Supplier", "Total PO (BHD)", "Rating", "Lead Days"})
			for _, s := range topSupp {
				addTableRow(pdf, []string{
					fmt.Sprintf("%v", s["name"]),
					fmt.Sprintf("%.3f", s["total_po"]),
					fmt.Sprintf("%v/5", s["rating"]),
					fmt.Sprintf("%v", s["lead_days"]),
				})
			}
		}
		pdf.Ln(3)
	}
}

// addAnalysisPage adds the AI-generated analysis with proper markdown rendering
func addAnalysisPage(pdf *gofpdf.Fpdf, aiAnalysis string) {
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 7, "Analysis")
	pdf.Ln(8)

	// Clean up the AI text (remove any [ACTIONS] blocks, sanitize encoding)
	cleanText := actionBlockRegex.ReplaceAllString(aiAnalysis, "")
	cleanText = strings.TrimSpace(cleanText)
	cleanText = sanitizeForPDF(cleanText) // Convert UTF-8 special chars to ASCII

	// Process line by line for proper markdown rendering
	lines := strings.Split(cleanText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines (just add small spacing)
		if line == "" {
			pdf.Ln(2)
			continue
		}

		// Handle --- dividers
		if line == "---" || line == "***" || line == "___" {
			pdf.Ln(1)
			pdf.SetDrawColor(200, 200, 200)
			pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
			pdf.Ln(2)
			continue
		}

		// Handle ## headers
		if strings.HasPrefix(line, "## ") {
			headerText := strings.TrimPrefix(line, "## ")
			headerText = stripMarkdownBold(headerText)
			pdf.Ln(3)
			pdf.SetFont("Helvetica", "B", 11)
			pdf.SetTextColor(29, 29, 31)
			pdf.MultiCell(0, 5, sanitizeForPDF(headerText), "", "", false)
			pdf.Ln(1)
			continue
		}

		// Handle ### subheaders
		if strings.HasPrefix(line, "### ") {
			headerText := strings.TrimPrefix(line, "### ")
			headerText = stripMarkdownBold(headerText)
			pdf.Ln(2)
			pdf.SetFont("Helvetica", "B", 10)
			pdf.SetTextColor(50, 50, 55)
			pdf.MultiCell(0, 5, sanitizeForPDF(headerText), "", "", false)
			pdf.Ln(1)
			continue
		}

		// Handle numbered section headers: "1. EXECUTIVE SUMMARY" or "1. **SECTION**"
		if isNumberedHeader(line) {
			headerText := cleanHeaderText(line)
			pdf.Ln(3)
			pdf.SetFont("Helvetica", "B", 11)
			pdf.SetTextColor(29, 29, 31)
			pdf.MultiCell(0, 5, sanitizeForPDF(headerText), "", "", false)
			pdf.Ln(1)
			continue
		}

		// Handle standalone bold lines as subheaders: "**Something**"
		if isStandaloneBold(line) {
			headerText := stripMarkdownBold(line)
			pdf.Ln(1)
			pdf.SetFont("Helvetica", "B", 9)
			pdf.SetTextColor(40, 40, 45)
			pdf.MultiCell(0, 4, sanitizeForPDF(headerText), "", "", false)
			pdf.Ln(1)
			continue
		}

		// Handle bullet points
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			bulletText := line[2:]
			bulletText = stripMarkdownBold(bulletText)
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetTextColor(40, 40, 40)
			pdf.SetX(22)
			pdf.MultiCell(0, 4, "- "+sanitizeForPDF(bulletText), "", "", false)
			continue
		}

		// Regular text - strip ** and render
		plainText := stripMarkdownBold(line)
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(40, 40, 40)
		pdf.MultiCell(0, 4, sanitizeForPDF(plainText), "", "", false)
	}
}

// stripMarkdownBold removes all ** markers from text, returning plain text
func stripMarkdownBold(s string) string {
	return butlerreports.StripMarkdownBold(s)
}

// sanitizeForPDF replaces UTF-8 special characters with ASCII equivalents
// gofpdf uses Windows-1252 encoding and cannot render multi-byte UTF-8 chars
func sanitizeForPDF(s string) string {
	return butlerreports.SanitizeForPDF(s)
}

// isNumberedHeader checks if a line is a numbered section header like "1. TITLE" or "1. **TITLE**"
func isNumberedHeader(s string) bool {
	return butlerreports.IsNumberedHeader(s)
}

// cleanHeaderText extracts clean header text from numbered headers
func cleanHeaderText(s string) string {
	return butlerreports.CleanHeaderText(s)
}

// isStandaloneBold checks if the entire line is wrapped in ** (a bold subheader)
func isStandaloneBold(s string) bool {
	return butlerreports.IsStandaloneBold(s)
}

// ============================================================================
// PDF HELPER FUNCTIONS
// ============================================================================

func addSectionHeader(pdf *gofpdf.Fpdf, title string) {
	butlerreports.AddSectionHeader(pdf, title)
}

func addMetricRow(pdf *gofpdf.Fpdf, label, value string) {
	butlerreports.AddMetricRow(pdf, label, value)
}

func addTableHeader(pdf *gofpdf.Fpdf, headers []string) {
	butlerreports.AddTableHeader(pdf, headers)
}

func addTableRow(pdf *gofpdf.Fpdf, values []string) {
	butlerreports.AddTableRow(pdf, values)
}
