// ═══════════════════════════════════════════════════════════════════════════
// PAYSLIP PDF GENERATION SERVICE
//
// MISSION (Wave 13 / P6): Render a single employee's payslip for a payroll
// period on the division's letterhead, following invoice_pdf_service.go's
// established pattern (letterhead, identity, amount-in-words, export dir).
//
// SCOPE DISCIPLINE: this reads existing payroll records only — the run,
// its items, and its components. It computes NOTHING new; every figure on
// the payslip is a value already stored by pkg/finance/payroll's run
// generation. If a payslip field ever needs a number that isn't already on
// PayrollRunItem/PayrollComponent, that is a stop-and-report, not a fix
// here.
//
// OUTPUT: exports/Reports/Payslip_<EmployeeCode>_<PeriodName>.pdf
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// GeneratePayslipPDF renders the payslip for one employee for one payroll
// period. It looks up the most recent payroll run item for that
// employee/period pair (a period can have more than one run — regenerate-
// after-correction is legal, see pkg/finance/payroll), then reads that
// item's stored components (or falls back to the item's own summed
// totals when no component rows exist, e.g. very old data).
func (a *App) GeneratePayslipPDF(employeeID, payrollPeriodID string) (string, error) {
	if err := a.payrollGuarded(a.requirePayrollView); err != nil {
		return "", err
	}

	employeeID = strings.TrimSpace(employeeID)
	payrollPeriodID = strings.TrimSpace(payrollPeriodID)
	if employeeID == "" {
		return "", fmt.Errorf("employee id is required")
	}
	if payrollPeriodID == "" {
		return "", fmt.Errorf("payroll period id is required")
	}

	log.Printf("📄 Generating payslip PDF: employeeID=%s payrollPeriodID=%s", employeeID, payrollPeriodID)

	var period PayrollPeriod
	if err := a.db.First(&period, "id = ?", payrollPeriodID).Error; err != nil {
		return "", fmt.Errorf("failed to fetch payroll period: %w", err)
	}

	var item PayrollRunItem
	err := a.db.
		Joins("JOIN payroll_runs ON payroll_runs.id = payroll_run_items.payroll_run_id").
		Where("payroll_run_items.employee_id = ? AND payroll_runs.payroll_period_id = ?", employeeID, payrollPeriodID).
		Order("payroll_runs.created_at DESC").
		First(&item).Error
	if err != nil {
		return "", fmt.Errorf("no payroll run item found for employee %s in period %s: %w", employeeID, period.Name, err)
	}

	var run PayrollRun
	if err := a.db.First(&run, "id = ?", item.PayrollRunID).Error; err != nil {
		return "", fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	var components []PayrollComponent
	if err := a.db.Where("payroll_run_item_id = ?", item.ID).Order("component_type ASC, name ASC").Find(&components).Error; err != nil {
		return "", fmt.Errorf("failed to fetch payroll components: %w", err)
	}
	if len(components) == 0 {
		// Fallback for run items generated before per-component rows existed
		// (or any other data gap) — the same totals the run already stored,
		// just not broken out line by line.
		if item.BaseSalary != 0 {
			components = append(components, PayrollComponent{ComponentType: "earning", Code: "BASE", Name: "Base Salary", Amount: item.BaseSalary})
		}
		if item.AllowancesTotal != 0 {
			components = append(components, PayrollComponent{ComponentType: "earning", Code: "ALLOW", Name: "Allowances", Amount: item.AllowancesTotal})
		}
		if item.DeductionsTotal != 0 {
			components = append(components, PayrollComponent{ComponentType: "deduction", Code: "DEDUCT", Name: "Deductions", Amount: item.DeductionsTotal})
		}
	}

	var employee Employee
	if err := a.db.First(&employee, "id = ?", employeeID).Error; err != nil {
		log.Printf("⚠️ Could not fetch employee details: %v", err)
	}

	employeeName := strings.TrimSpace(employee.FullName)
	if employeeName == "" {
		employeeName = item.EmployeeNameSnapshot
	}
	jobTitle := strings.TrimSpace(employee.JobTitle)
	if jobTitle == "" {
		jobTitle = item.JobTitleSnapshot
	}

	profile := companyDocumentProfile(run.Division)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 30)
	pdf.SetTopMargin(40)
	pdf.SetLeftMargin(15)
	pdf.SetRightMargin(15)
	pdf.SetHeaderFuncMode(func() {
		a.applyLetterheadForDivision(pdf, profile.Division)
	}, true)
	pdf.AddPage()

	// ------------------------------------------------------------------
	// TITLE
	// ------------------------------------------------------------------
	pdf.SetY(40)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(29, 29, 31)
	pdf.CellFormat(0, 8, "PAYSLIP", "", 0, "C", false, 0, "")
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.CellFormat(0, 5, sanitizeForPDF(period.Name), "", 0, "C", false, 0, "")
	pdf.Ln(10)

	// ------------------------------------------------------------------
	// EMPLOYEE IDENTITY BLOCK
	// ------------------------------------------------------------------
	pdf.SetX(15)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, sanitizeForPDF(employeeName))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(60, 60, 60)
	if jobTitle != "" {
		pdf.SetX(15)
		pdf.Cell(30, 5, "Job Title:")
		pdf.Cell(0, 5, sanitizeForPDF(jobTitle))
		pdf.Ln(5)
	}
	if employee.Department != "" {
		pdf.SetX(15)
		pdf.Cell(30, 5, "Department:")
		pdf.Cell(0, 5, sanitizeForPDF(employee.Department))
		pdf.Ln(5)
	}
	if employee.EmployeeCode != "" {
		pdf.SetX(15)
		pdf.Cell(30, 5, "Employee No.:")
		pdf.Cell(0, 5, sanitizeForPDF(employee.EmployeeCode))
		pdf.Ln(5)
	}
	pdf.SetX(15)
	pdf.Cell(30, 5, "Pay Period:")
	pdf.Cell(60, 5, fmt.Sprintf("%s to %s", period.PeriodStart.Format("02-Jan-2006"), period.PeriodEnd.Format("02-Jan-2006")))
	pdf.Cell(30, 5, "Run:")
	pdf.Cell(0, 5, sanitizeForPDF(run.RunNumber))
	pdf.Ln(8)

	// ------------------------------------------------------------------
	// EARNINGS / DEDUCTIONS TABLE
	// ------------------------------------------------------------------
	colDesc := 100.0
	colAmt := 40.0

	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(29, 29, 31)
	pdf.SetX(15)
	pdf.CellFormat(colDesc, 6, "Earnings", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colAmt, 6, "Amount ("+firstPopulatedString(run.Currency, "BHD")+")", "1", 0, "R", true, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(29, 29, 31)
	earningsTotal := 0.0
	for _, c := range components {
		if c.ComponentType != "earning" {
			continue
		}
		pdf.SetX(15)
		pdf.CellFormat(colDesc, 5, sanitizeForPDF(c.Name), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colAmt, 5, fmt.Sprintf("%.3f", c.Amount), "1", 0, "R", false, 0, "")
		pdf.Ln(5)
		earningsTotal += c.Amount
	}
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetX(15)
	pdf.CellFormat(colDesc, 6, "Total Earnings", "1", 0, "L", false, 0, "")
	pdf.CellFormat(colAmt, 6, fmt.Sprintf("%.3f", earningsTotal), "1", 0, "R", false, 0, "")
	pdf.Ln(9)

	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(29, 29, 31)
	pdf.SetX(15)
	pdf.CellFormat(colDesc, 6, "Deductions", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colAmt, 6, "Amount ("+firstPopulatedString(run.Currency, "BHD")+")", "1", 0, "R", true, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(29, 29, 31)
	deductionsTotal := 0.0
	for _, c := range components {
		if c.ComponentType != "deduction" {
			continue
		}
		pdf.SetX(15)
		pdf.CellFormat(colDesc, 5, sanitizeForPDF(c.Name), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colAmt, 5, fmt.Sprintf("%.3f", c.Amount), "1", 0, "R", false, 0, "")
		pdf.Ln(5)
		deductionsTotal += c.Amount
	}
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetX(15)
	pdf.CellFormat(colDesc, 6, "Total Deductions", "1", 0, "L", false, 0, "")
	pdf.CellFormat(colAmt, 6, fmt.Sprintf("%.3f", deductionsTotal), "1", 0, "R", false, 0, "")
	pdf.Ln(10)

	// ------------------------------------------------------------------
	// NET PAY
	// ------------------------------------------------------------------
	// P6: display the item's own stored NetPay (already computed by the
	// run-generation arithmetic) rather than re-deriving earnings minus
	// deductions here — a component fallback row can be an approximation
	// of the stored totals, but NetPay itself is never recomputed.
	netPay := item.NetPay
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetX(15)
	pdf.CellFormat(colDesc, 7, "NET PAY", "1", 0, "L", false, 0, "")
	pdf.CellFormat(colAmt, 7, fmt.Sprintf("%.3f %s", netPay, firstPopulatedString(run.Currency, "BHD")), "1", 0, "R", false, 0, "")
	pdf.Ln(10)

	// ------------------------------------------------------------------
	// AMOUNT IN WORDS
	// ------------------------------------------------------------------
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.SetX(15)
	pdf.Cell(45, 5, "Net Pay (in words):")
	pdf.Ln(5)
	pdf.SetX(15)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(60, 60, 60)
	pdf.MultiCell(140, 4, amountInWords(netPay), "", "", false)
	pdf.Ln(6)

	// ------------------------------------------------------------------
	// DIVISION IDENTITY FOOTER
	// ------------------------------------------------------------------
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(160, 160, 160)
	pdf.SetX(15)
	pdf.CellFormat(0, 5, fmt.Sprintf("%s  |  %s  |  This is a system-generated payslip.", profile.LegalName, period.Name), "", 0, "L", false, 0, "")

	// ------------------------------------------------------------------
	// SAVE
	// ------------------------------------------------------------------
	entityName := employeeName
	if employee.EmployeeCode != "" {
		entityName = employee.EmployeeCode
	}
	filename := fmt.Sprintf("Payslip_%s_%s.pdf", sanitizeFilename(entityName), sanitizeFilename(period.Name))
	saveDir := a.getExportDir("report", "", "Payslips", period.PeriodEnd.Year())
	filePath := filepath.Join(saveDir, filename)

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", fmt.Errorf("failed to save payslip PDF: %w", err)
	}

	log.Printf("✅ Payslip PDF generated: %s", filePath)
	return filePath, nil
}
