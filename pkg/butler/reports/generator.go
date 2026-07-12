// Package reports contains Butler intelligence report generation helpers.
package reports

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

var actionBlockRegex = regexp.MustCompile(`(?s)\[ACTIONS\](.*?)\[/ACTIONS\]`)

func DomainForReportType(reportType string) string {
	switch reportType {
	case "executive":
		return "general"
	case "risk":
		return "risk"
	case "customer":
		return "customer"
	case "supplier":
		return "supplier"
	case "financial":
		return "financial"
	case "operations":
		return "operations"
	default:
		return reportType
	}
}

func RequiresFinanceAccess(reportType string) bool {
	switch reportType {
	case "financial", "risk", "executive":
		return true
	default:
		return false
	}
}

func BuildPrompt(reportType, query, contextJSON string) string {
	return fmt.Sprintf(`You are a senior management consultant preparing a business intelligence brief for the executive team at Acme Instrumentation WLL, a process instrumentation and industrial automation company based in Bahrain.

REPORT FOCUS: %s
SPECIFIC SUBJECT: %s

LIVE BUSINESS DATA:
%s

DELIVERABLE:
Produce a concise, insight-driven management brief structured as follows:

1. EXECUTIVE SUMMARY
Two to three sentences capturing the headline finding and its business implication.

2. KEY FIGURES
The 6-8 most decision-relevant metrics, presented as a clean bullet list with exact values.

3. ANALYSIS
Three to five paragraphs of substantive analysis. Organize by business theme (e.g., Revenue Performance, Working Capital, Portfolio Concentration, Collection Efficiency). Lead each paragraph with the insight, then support with data.

4. RISK FACTORS
Identify 3-5 specific business risks grounded in the data. For each, state the risk, the evidence, and the potential impact.

5. RECOMMENDATIONS
Provide 3-5 specific, actionable management actions. Each should state what to do, why, and the expected benefit.

6. OUTLOOK
One paragraph on the near-term business trajectory based on current trends.

WRITING STANDARDS:
- Write in clear, direct business English. No jargon, no technical terms, no mathematical language.
- Use exact figures from the data. Currency is BHD (Bahraini Dinar, 3 decimal places).
- Lead with insights, not data dumps. Every number should support a point.
- Tone: authoritative, concise, suitable for a board presentation.
- Do NOT use asterisks, markdown formatting, or any special characters for emphasis.
- Use "- " for bullet points.
- Use numbered sections as shown above.
- If data is insufficient for a conclusion, state what additional data would be needed.
- Do NOT use words like "regime", "stabilization phase", "exploration", "optimization" or any systems/mathematical terminology. Use plain business language only.

Generate the report now.`, reportType, query, contextJSON)
}

func HasDataForReport(reportType string, context map[string]any) bool {
	switch reportType {
	case "customer":
		_, ok := context["customer_data"]
		_, ok2 := context["ar_summary"]
		return ok || ok2
	case "financial":
		_, ok := context["financial_data"]
		return ok
	case "risk":
		_, ok := context["risk_data"]
		return ok
	case "operations":
		_, ok := context["operations_data"]
		return ok
	case "supplier":
		_, ok := context["supplier_data"]
		return ok
	case "executive":
		return true
	default:
		_, ok := context["business_summary"]
		return ok
	}
}

func AddCoverPage(pdf *gofpdf.Fpdf, reportType, query string) {
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetTextColor(29, 29, 31)
	reportTitle := ReportTitle(reportType)
	pdf.Cell(0, 10, reportTitle)
	pdf.Ln(14)

	if query != "" {
		pdf.SetFont("Helvetica", "", 13)
		pdf.SetTextColor(60, 60, 65)
		pdf.Cell(0, 8, fmt.Sprintf("Subject: %s", SanitizeForPDF(query)))
		pdf.Ln(14)
	}

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
	pdf.Cell(0, 6, "Acme Instrumentation Business Intelligence")
	pdf.Ln(14)

	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(130, 130, 130)
	pdf.MultiCell(0, 4, "This report is generated from live ERP data and AI-powered analysis. All figures are derived from actual transaction records. Intended for internal management use only.", "", "", false)
}

func ReportTitle(reportType string) string {
	switch reportType {
	case "customer":
		return "Customer Intelligence Report"
	case "financial":
		return "Financial Health Report"
	case "risk":
		return "Risk Assessment Report"
	case "operations":
		return "Operations Pipeline Report"
	case "supplier":
		return "Supplier Performance Report"
	case "executive":
		return "Executive Summary Report"
	default:
		return "Intelligence Report"
	}
}

func AddAnalysisPage(pdf *gofpdf.Fpdf, aiAnalysis string) {
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 7, "Analysis")
	pdf.Ln(8)

	cleanText := actionBlockRegex.ReplaceAllString(aiAnalysis, "")
	cleanText = strings.TrimSpace(cleanText)
	cleanText = SanitizeForPDF(cleanText)

	lines := strings.Split(cleanText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			pdf.Ln(2)
			continue
		}
		if line == "---" || line == "***" || line == "___" {
			pdf.Ln(1)
			pdf.SetDrawColor(200, 200, 200)
			pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
			pdf.Ln(2)
			continue
		}
		if strings.HasPrefix(line, "## ") {
			headerText := strings.TrimPrefix(line, "## ")
			headerText = StripMarkdownBold(headerText)
			pdf.Ln(3)
			pdf.SetFont("Helvetica", "B", 11)
			pdf.SetTextColor(29, 29, 31)
			pdf.MultiCell(0, 5, SanitizeForPDF(headerText), "", "", false)
			pdf.Ln(1)
			continue
		}
		if strings.HasPrefix(line, "### ") {
			headerText := strings.TrimPrefix(line, "### ")
			headerText = StripMarkdownBold(headerText)
			pdf.Ln(2)
			pdf.SetFont("Helvetica", "B", 10)
			pdf.SetTextColor(50, 50, 55)
			pdf.MultiCell(0, 5, SanitizeForPDF(headerText), "", "", false)
			pdf.Ln(1)
			continue
		}
		if IsNumberedHeader(line) {
			headerText := CleanHeaderText(line)
			pdf.Ln(3)
			pdf.SetFont("Helvetica", "B", 11)
			pdf.SetTextColor(29, 29, 31)
			pdf.MultiCell(0, 5, SanitizeForPDF(headerText), "", "", false)
			pdf.Ln(1)
			continue
		}
		if IsStandaloneBold(line) {
			headerText := StripMarkdownBold(line)
			pdf.Ln(1)
			pdf.SetFont("Helvetica", "B", 9)
			pdf.SetTextColor(40, 40, 45)
			pdf.MultiCell(0, 4, SanitizeForPDF(headerText), "", "", false)
			pdf.Ln(1)
			continue
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			bulletText := line[2:]
			bulletText = StripMarkdownBold(bulletText)
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetTextColor(40, 40, 40)
			pdf.SetX(22)
			pdf.MultiCell(0, 4, "- "+SanitizeForPDF(bulletText), "", "", false)
			continue
		}

		plainText := StripMarkdownBold(line)
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(40, 40, 40)
		pdf.MultiCell(0, 4, SanitizeForPDF(plainText), "", "", false)
	}
}

func StripMarkdownBold(s string) string {
	return strings.ReplaceAll(s, "**", "")
}

func SanitizeForPDF(s string) string {
	replacer := strings.NewReplacer(
		"\u2019", "'",
		"\u2018", "'",
		"\u201C", "\"",
		"\u201D", "\"",
		"\u2013", "-",
		"\u2014", "-",
		"\u2026", "...",
		"\u2022", "-",
		"\u2023", "-",
		"\u2043", "-",
		"\u00A0", " ",
		"\u200B", "",
		"\u00B7", "-",
		"\u2212", "-",
		"\u2032", "'",
		"\u2033", "\"",
		"\u00AB", "\"",
		"\u00BB", "\"",
		"\u2010", "-",
		"\u2011", "-",
		"\u2015", "-",
		"\u00D7", "x",
		"\u00F7", "/",
	)
	return replacer.Replace(s)
}

func IsNumberedHeader(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 3 {
		return false
	}
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(s) {
		return false
	}
	if s[i] != '.' && s[i] != ')' {
		return false
	}
	rest := strings.TrimSpace(s[i+1:])
	rest = StripMarkdownBold(rest)
	if len(rest) < 3 || len(rest) > 80 {
		return false
	}
	upper := strings.ToUpper(rest)
	if rest == upper {
		return true
	}
	if strings.Contains(s, "**") {
		return true
	}
	return false
}

func CleanHeaderText(s string) string {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i < len(s) && (s[i] == '.' || s[i] == ')') {
		i++
	}
	result := strings.TrimSpace(s[i:])
	result = StripMarkdownBold(result)
	return result
}

func IsStandaloneBold(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 5 {
		return false
	}
	if strings.HasPrefix(s, "**") && strings.HasSuffix(s, "**") && strings.Count(s, "**") == 2 {
		return true
	}
	return false
}

func AddSectionHeader(pdf *gofpdf.Fpdf, title string) {
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, SanitizeForPDF(title))
	pdf.Ln(7)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(60, 60, 60)
}

func AddMetricRow(pdf *gofpdf.Fpdf, label, value string) {
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(110, 110, 115)
	pdf.Cell(65, 5, SanitizeForPDF(label))
	pdf.SetTextColor(29, 29, 31)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.Cell(0, 5, SanitizeForPDF(value))
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 9)
}

func AddTableHeader(pdf *gofpdf.Fpdf, headers []string) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(45, 45, 48)

	colWidth := 170.0 / float64(len(headers))
	for _, h := range headers {
		pdf.CellFormat(colWidth, 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(7)
	pdf.SetTextColor(29, 29, 31)
	pdf.SetFont("Helvetica", "", 9)
}

func AddTableRow(pdf *gofpdf.Fpdf, values []string) {
	colWidth := 170.0 / float64(len(values))
	for i, v := range values {
		align := "L"
		if i > 0 {
			align = "C"
		}
		pdf.CellFormat(colWidth, 6, SanitizeForPDF(v), "1", 0, align, false, 0, "")
	}
	pdf.Ln(6)
}
