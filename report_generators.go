package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

// =============================================================================
// REPORT GENERATION HANDLERS (PDF + EXCEL)
// =============================================================================
//
// PURPOSE:
// - Generate professional PDF reports with charts
// - Generate Excel reports with data tables and formulas
// - Support all report categories: sales, customers, operations, financial
//
// LIBRARIES:
// - gofpdf: Pure Go PDF generation (no Pandoc dependency!)
// - excelize: Excel XLSX generation with full formatting
//
// BUSINESS CONTEXT:
// - Reports are survival metrics (runway, collections, margins)
// - Must be audit-trail compliant (timestamp, metadata)
// - Export to accountants/investors (professional formatting)
// =============================================================================

// RegisterReportHandlers registers all report generation job handlers
func (a *App) RegisterReportHandlers() {
	a.jobQueue.RegisterHandler("report_generate", a.handleReportGenerate)
}

// handleReportGenerate processes report generation jobs
func (a *App) handleReportGenerate(ctx context.Context, job *Job) error {
	var input ReportGenerateInput
	if err := json.Unmarshal([]byte(job.Input), &input); err != nil {
		return fmt.Errorf("invalid job input: %w", err)
	}

	// Update progress
	job.Progress = 10
	a.jobQueue.updateJob(job)

	// Get report data
	dateRange := calculateDateRange(input.DateRange.Start, input.DateRange.End)
	reportData, err := a.GetReportData(input.Category, dateRange)
	if err != nil {
		return fmt.Errorf("failed to fetch report data: %w", err)
	}

	job.Progress = 30
	a.jobQueue.updateJob(job)

	// Generate file based on format
	var filePath string
	// Convert DateRange to expected format
	dateRangeStruct := struct{ Start, End string }{
		Start: input.DateRange.Start,
		End:   input.DateRange.End,
	}
	switch input.Format {
	case "pdf":
		filePath, err = a.generatePDFReport(input.Category, reportData, dateRangeStruct)
	case "excel":
		filePath, err = a.generateExcelReport(input.Category, reportData, dateRangeStruct)
	case "csv":
		filePath, err = a.generateCSVReport(input.Category, reportData, dateRangeStruct)
	default:
		return fmt.Errorf("unsupported format: %s", input.Format)
	}

	if err != nil {
		return fmt.Errorf("report generation failed: %w", err)
	}

	job.Progress = 90
	a.jobQueue.updateJob(job)

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat generated file: %w", err)
	}

	// Prepare output
	output := ReportGenerateOutput{
		FilePath:  filePath,
		FileSize:  fileInfo.Size(),
		Generated: time.Now(),
	}

	outputJSON, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	job.Output = string(outputJSON)
	return nil
}

// =============================================================================
// PDF REPORT GENERATION
// =============================================================================

func (a *App) generatePDFReport(category string, data ReportData, dateRange struct{ Start, End string }) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(0, 10, fmt.Sprintf("%s - %s Report", activeOverlay.CompanyDisplayName, titleCase(category)))
	pdf.Ln(12)

	// Metadata
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 6, fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Date Range: %s to %s", dateRange.Start, dateRange.End))
	pdf.Ln(10)

	pdf.SetTextColor(0, 0, 0)

	// Category-specific content
	switch category {
	case "sales":
		a.addSalesReportToPDF(pdf, data)
	case "customers":
		a.addCustomersReportToPDF(pdf, data)
	case "operations":
		a.addOperationsReportToPDF(pdf, data)
	case "inventory":
		a.addInventoryReportToPDF(pdf, data)
	case "financial":
		a.addFinancialReportToPDF(pdf, data)
	default:
		return "", fmt.Errorf("unknown report category: %s", category)
	}

	// Footer
	pdf.SetY(-15)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 10, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "C", false, 0, "")

	// Save to file
	exportDir := a.getExportDir("report", "", "", time.Now().Year())

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_report_%s.pdf", category, timestamp)
	filePath := filepath.Join(exportDir, filename)

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", fmt.Errorf("failed to save PDF: %w", err)
	}

	return filePath, nil
}

func (a *App) addSalesReportToPDF(pdf *gofpdf.Fpdf, data ReportData) {
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Key Metrics")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 11)
	pdf.Cell(70, 7, "Win Rate:")
	pdf.Cell(0, 7, fmt.Sprintf("%.1f%%", data.WinRate*100))
	pdf.Ln(7)

	pdf.Cell(70, 7, "Conversion Rate:")
	pdf.Cell(0, 7, fmt.Sprintf("%.1f%%", data.ConversionRate*100))
	pdf.Ln(7)

	pdf.Cell(70, 7, "Average Deal Size:")
	pdf.Cell(0, 7, fmt.Sprintf("%.2f BHD", data.AvgDealSize))
	pdf.Ln(12)

	// Pipeline table
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Pipeline by Stage")
	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(60, 7, "Stage", "1", 0, "L", false, 0, "")
	pdf.CellFormat(40, 7, "Count", "1", 0, "C", false, 0, "")
	pdf.CellFormat(50, 7, "Value (BHD)", "1", 0, "R", false, 0, "")
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 10)
	for _, stage := range data.Pipeline {
		pdf.CellFormat(60, 7, stage.Stage, "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 7, fmt.Sprintf("%d", stage.Count), "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 7, fmt.Sprintf("%.2f", stage.Value), "1", 0, "R", false, 0, "")
		pdf.Ln(7)
	}
}

func (a *App) addCustomersReportToPDF(pdf *gofpdf.Fpdf, data ReportData) {
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Customer Metrics")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 11)
	pdf.Cell(80, 7, "Average Payment Days:")
	pdf.Cell(0, 7, fmt.Sprintf("%.0f days", data.AvgPaymentDays))
	pdf.Ln(7)

	pdf.Cell(80, 7, "Collection Efficiency:")
	pdf.Cell(0, 7, fmt.Sprintf("%.1f%%", data.CollectionEff*100))
	pdf.Ln(12)

	// Grade distribution
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Grade Distribution")
	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(40, 7, "Grade", "1", 0, "C", false, 0, "")
	pdf.CellFormat(50, 7, "Customers", "1", 0, "C", false, 0, "")
	pdf.CellFormat(50, 7, "Percentage", "1", 0, "C", false, 0, "")
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 10)
	for _, grade := range data.GradeDistribution {
		pdf.CellFormat(40, 7, grade.Grade, "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 7, fmt.Sprintf("%d", grade.Count), "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 7, fmt.Sprintf("%.1f%%", grade.Percentage), "1", 0, "C", false, 0, "")
		pdf.Ln(7)
	}
}

func (a *App) addOperationsReportToPDF(pdf *gofpdf.Fpdf, data ReportData) {
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Operations Metrics")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 11)
	pdf.Cell(80, 7, "Average Lead Time:")
	pdf.Cell(0, 7, fmt.Sprintf("%d days", data.AvgLeadTime))
	pdf.Ln(7)

	pdf.Cell(80, 7, "On-Time Delivery:")
	pdf.Cell(0, 7, fmt.Sprintf("%.1f%%", data.OnTimeDelivery*100))
	pdf.Ln(7)

	pdf.Cell(80, 7, "Pending Shipments:")
	pdf.Cell(0, 7, fmt.Sprintf("%d", data.PendingShipments))
	pdf.Ln(12)
}

func (a *App) addInventoryReportToPDF(pdf *gofpdf.Fpdf, data ReportData) {
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Inventory Metrics")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 11)
	pdf.Cell(70, 7, "Total Items:")
	pdf.Cell(0, 7, fmt.Sprintf("%d", data.TotalItems))
	pdf.Ln(7)

	pdf.Cell(70, 7, "Total Value:")
	pdf.Cell(0, 7, fmt.Sprintf("%.2f BHD", data.TotalValue))
	pdf.Ln(7)

	pdf.Cell(70, 7, "Low Stock Alerts:")
	pdf.Cell(0, 7, fmt.Sprintf("%d", data.LowStockAlerts))
	pdf.Ln(12)
}

func (a *App) addFinancialReportToPDF(pdf *gofpdf.Fpdf, data ReportData) {
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Financial Health")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 11)
	pdf.Cell(70, 7, "Receivables Outstanding:")
	pdf.Cell(0, 7, fmt.Sprintf("%.2f BHD", data.ReceivablesOutstanding))
	pdf.Ln(7)

	pdf.Cell(70, 7, "Payables Outstanding:")
	pdf.Cell(0, 7, fmt.Sprintf("%.2f BHD", data.PayablesOutstanding))
	pdf.Ln(7)

	pdf.Cell(70, 7, "Avg Monthly Revenue:")
	pdf.Cell(0, 7, fmt.Sprintf("%.2f BHD", data.AvgMonthlyRevenue))
	pdf.Ln(12)

	// Collections
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Collections")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 11)
	pdf.Cell(70, 7, "Collected:")
	pdf.Cell(0, 7, fmt.Sprintf("%.2f BHD", data.Collected))
	pdf.Ln(7)

	pdf.Cell(70, 7, "Collection Target:")
	pdf.Cell(0, 7, fmt.Sprintf("%.2f BHD", data.CollectionTarget))
	pdf.Ln(12)

	// Overdue receivables
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Overdue Receivables")
	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(70, 7, "Aging", "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 7, "Amount (BHD)", "1", 0, "R", false, 0, "")
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 10)
	totalOverdue := 0.0
	for _, bucket := range data.Overdue {
		pdf.CellFormat(70, 7, bucket.Days, "1", 0, "L", false, 0, "")
		pdf.CellFormat(60, 7, fmt.Sprintf("%.2f", bucket.Amount), "1", 0, "R", false, 0, "")
		pdf.Ln(7)
		totalOverdue += bucket.Amount
	}

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(70, 7, "TOTAL", "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 7, fmt.Sprintf("%.2f", totalOverdue), "1", 0, "R", false, 0, "")
}

// =============================================================================
// EXCEL REPORT GENERATION
// =============================================================================

func (a *App) generateExcelReport(category string, data ReportData, dateRange struct{ Start, End string }) (string, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Report"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// Title
	f.SetCellValue(sheetName, "A1", fmt.Sprintf("%s - %s Report", activeOverlay.CompanyDisplayName, titleCase(category)))
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16},
	})
	f.SetCellStyle(sheetName, "A1", "A1", titleStyle)

	// Metadata
	f.SetCellValue(sheetName, "A2", fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))
	f.SetCellValue(sheetName, "A3", fmt.Sprintf("Date Range: %s to %s", dateRange.Start, dateRange.End))

	// Category-specific content
	switch category {
	case "sales":
		a.addSalesReportToExcel(f, sheetName, data)
	case "customers":
		a.addCustomersReportToExcel(f, sheetName, data)
	case "operations":
		a.addOperationsReportToExcel(f, sheetName, data)
	case "inventory":
		a.addInventoryReportToExcel(f, sheetName, data)
	case "financial":
		a.addFinancialReportToExcel(f, sheetName, data)
	default:
		return "", fmt.Errorf("unknown report category: %s", category)
	}

	// Save to file
	exportDir := a.getExportDir("report", "", "", time.Now().Year())

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_report_%s.xlsx", category, timestamp)
	filePath := filepath.Join(exportDir, filename)

	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("failed to save Excel: %w", err)
	}

	return filePath, nil
}

func (a *App) addSalesReportToExcel(f *excelize.File, sheet string, data ReportData) {
	// Headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})

	// Metrics
	f.SetCellValue(sheet, "A5", "Key Metrics")
	f.SetCellStyle(sheet, "A5", "A5", headerStyle)
	f.SetCellValue(sheet, "A6", "Win Rate")
	f.SetCellValue(sheet, "B6", data.WinRate)
	f.SetCellValue(sheet, "A7", "Conversion Rate")
	f.SetCellValue(sheet, "B7", data.ConversionRate)
	f.SetCellValue(sheet, "A8", "Avg Deal Size (BHD)")
	f.SetCellValue(sheet, "B8", data.AvgDealSize)

	// Format percentages
	f.SetCellStyle(sheet, "B6", "B7", getPercentStyle(f))
	f.SetCellStyle(sheet, "B8", "B8", getCurrencyStyle(f))

	// Pipeline table
	row := 10
	f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Pipeline by Stage")
	f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "A"+fmt.Sprint(row), headerStyle)
	row++

	f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Stage")
	f.SetCellValue(sheet, "B"+fmt.Sprint(row), "Count")
	f.SetCellValue(sheet, "C"+fmt.Sprint(row), "Value (BHD)")
	f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "C"+fmt.Sprint(row), headerStyle)
	row++

	for _, stage := range data.Pipeline {
		f.SetCellValue(sheet, "A"+fmt.Sprint(row), stage.Stage)
		f.SetCellValue(sheet, "B"+fmt.Sprint(row), stage.Count)
		f.SetCellValue(sheet, "C"+fmt.Sprint(row), stage.Value)
		f.SetCellStyle(sheet, "C"+fmt.Sprint(row), "C"+fmt.Sprint(row), getCurrencyStyle(f))
		row++
	}
}

func (a *App) addCustomersReportToExcel(f *excelize.File, sheet string, data ReportData) {
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})

	f.SetCellValue(sheet, "A5", "Customer Metrics")
	f.SetCellStyle(sheet, "A5", "A5", headerStyle)
	f.SetCellValue(sheet, "A6", "Avg Payment Days")
	f.SetCellValue(sheet, "B6", data.AvgPaymentDays)
	f.SetCellValue(sheet, "A7", "Collection Efficiency")
	f.SetCellValue(sheet, "B7", data.CollectionEff)
	f.SetCellStyle(sheet, "B7", "B7", getPercentStyle(f))

	row := 9
	f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Grade Distribution")
	f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "A"+fmt.Sprint(row), headerStyle)
	row++

	f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Grade")
	f.SetCellValue(sheet, "B"+fmt.Sprint(row), "Customers")
	f.SetCellValue(sheet, "C"+fmt.Sprint(row), "Percentage")
	f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "C"+fmt.Sprint(row), headerStyle)
	row++

	for _, grade := range data.GradeDistribution {
		f.SetCellValue(sheet, "A"+fmt.Sprint(row), grade.Grade)
		f.SetCellValue(sheet, "B"+fmt.Sprint(row), grade.Count)
		f.SetCellValue(sheet, "C"+fmt.Sprint(row), grade.Percentage/100)
		f.SetCellStyle(sheet, "C"+fmt.Sprint(row), "C"+fmt.Sprint(row), getPercentStyle(f))
		row++
	}
}

func (a *App) addOperationsReportToExcel(f *excelize.File, sheet string, data ReportData) {
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})

	f.SetCellValue(sheet, "A5", "Operations Metrics")
	f.SetCellStyle(sheet, "A5", "A5", headerStyle)
	f.SetCellValue(sheet, "A6", "Avg Lead Time (days)")
	f.SetCellValue(sheet, "B6", data.AvgLeadTime)
	f.SetCellValue(sheet, "A7", "On-Time Delivery")
	f.SetCellValue(sheet, "B7", data.OnTimeDelivery)
	f.SetCellStyle(sheet, "B7", "B7", getPercentStyle(f))
	f.SetCellValue(sheet, "A8", "Pending Shipments")
	f.SetCellValue(sheet, "B8", data.PendingShipments)
}

func (a *App) addInventoryReportToExcel(f *excelize.File, sheet string, data ReportData) {
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})

	f.SetCellValue(sheet, "A5", "Inventory Metrics")
	f.SetCellStyle(sheet, "A5", "A5", headerStyle)
	f.SetCellValue(sheet, "A6", "Total Items")
	f.SetCellValue(sheet, "B6", data.TotalItems)
	f.SetCellValue(sheet, "A7", "Total Value (BHD)")
	f.SetCellValue(sheet, "B7", data.TotalValue)
	f.SetCellStyle(sheet, "B7", "B7", getCurrencyStyle(f))
	f.SetCellValue(sheet, "A8", "Low Stock Alerts")
	f.SetCellValue(sheet, "B8", data.LowStockAlerts)
}

func (a *App) addFinancialReportToExcel(f *excelize.File, sheet string, data ReportData) {
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})

	f.SetCellValue(sheet, "A5", "Financial Health")
	f.SetCellStyle(sheet, "A5", "A5", headerStyle)
	f.SetCellValue(sheet, "A6", "Receivables Outstanding (BHD)")
	f.SetCellValue(sheet, "B6", data.ReceivablesOutstanding)
	f.SetCellValue(sheet, "A7", "Payables Outstanding (BHD)")
	f.SetCellValue(sheet, "B7", data.PayablesOutstanding)
	f.SetCellValue(sheet, "A8", "Avg Monthly Revenue (BHD)")
	f.SetCellValue(sheet, "B8", data.AvgMonthlyRevenue)
	f.SetCellStyle(sheet, "B6", "B8", getCurrencyStyle(f))

	// Collections
	row := 10
	f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Collections")
	f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "A"+fmt.Sprint(row), headerStyle)
	row++
	f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Collected (BHD)")
	f.SetCellValue(sheet, "B"+fmt.Sprint(row), data.Collected)
	f.SetCellStyle(sheet, "B"+fmt.Sprint(row), "B"+fmt.Sprint(row), getCurrencyStyle(f))
	row++
	f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Collection Target (BHD)")
	f.SetCellValue(sheet, "B"+fmt.Sprint(row), data.CollectionTarget)
	f.SetCellStyle(sheet, "B"+fmt.Sprint(row), "B"+fmt.Sprint(row), getCurrencyStyle(f))
	row += 2

	// Overdue receivables
	f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Overdue Receivables")
	f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "A"+fmt.Sprint(row), headerStyle)
	row++
	f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Aging")
	f.SetCellValue(sheet, "B"+fmt.Sprint(row), "Amount (BHD)")
	f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "B"+fmt.Sprint(row), headerStyle)
	row++

	for _, bucket := range data.Overdue {
		f.SetCellValue(sheet, "A"+fmt.Sprint(row), bucket.Days)
		f.SetCellValue(sheet, "B"+fmt.Sprint(row), bucket.Amount)
		f.SetCellStyle(sheet, "B"+fmt.Sprint(row), "B"+fmt.Sprint(row), getCurrencyStyle(f))
		row++
	}

	// Total formula
	f.SetCellValue(sheet, "A"+fmt.Sprint(row), "TOTAL")
	startRow := row - len(data.Overdue)
	endRow := row - 1
	f.SetCellFormula(sheet, "B"+fmt.Sprint(row), fmt.Sprintf("=SUM(B%d:B%d)", startRow, endRow))
	f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "B"+fmt.Sprint(row), headerStyle)
}

// =============================================================================
// CSV REPORT GENERATION (Reuse existing function)
// =============================================================================

func (a *App) generateCSVReport(category string, data ReportData, dateRange struct{ Start, End string }) (string, error) {
	exportDir := a.getExportDir("report", "", "", time.Now().Year())

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_report_%s.csv", category, timestamp)
	filePath := filepath.Join(exportDir, filename)

	dataJSON, _ := json.Marshal(data)
	return a.exportReportToCSV(filePath, category, string(dataJSON))
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func calculateDateRange(start, end string) string {
	startTime, _ := time.Parse("2006-01-02", start)
	endTime, _ := time.Parse("2006-01-02", end)
	daysDiff := int(endTime.Sub(startTime).Hours() / 24)

	if daysDiff <= 7 {
		return "week"
	} else if daysDiff <= 31 {
		return "month"
	} else if daysDiff <= 92 {
		return "quarter"
	}
	return "year"
}

func getPercentStyle(f *excelize.File) int {
	style, _ := f.NewStyle(&excelize.Style{
		NumFmt: 10, // 0.00%
	})
	return style
}

func getCurrencyStyle(f *excelize.File) int {
	style, _ := f.NewStyle(&excelize.Style{
		NumFmt:       164, // Custom format
		CustomNumFmt: stringPtr("#,##0.00"),
	})
	return style
}

func stringPtr(s string) *string {
	return &s
}
