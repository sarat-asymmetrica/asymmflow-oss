// ═══════════════════════════════════════════════════════════════════════════
// COM AUTOMATION - Excel/Word/Outlook Automation via Windows COM
//
// MISSION: Direct Office automation for Acme Instrumentation desktop integration
//   - Excel: Open workbooks, insert audit data, trigger macros
//   - Word: Generate reports from templates
//   - Outlook: Send notifications, create calendar events
//
// ARCHITECTURE:
//   - Windows-only (COM requires Win32 OLE)
//   - Build tags ensure cross-platform compilation
//   - Graceful degradation if Office not installed
//   - Interface-based for testability
//
// Built with WINDOWS_NATIVE × PRODUCTION × TESTABILITY 🕉️⚡💎
// ═══════════════════════════════════════════════════════════════════════════

//go:build windows
// +build windows

package integration

import (
	"fmt"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// ═══════════════════════════════════════════════════════════════════════════
// TYPES
// ═══════════════════════════════════════════════════════════════════════════

// ExcelWorkbookData and WordDocumentData are defined in com_types.go (cross-platform)

// OutlookNotificationData represents Outlook email/calendar
type OutlookNotificationData struct {
	Type        string // "email" or "calendar"
	To          []string
	Subject     string
	Body        string
	Attachments []string

	// Calendar-specific
	Start    time.Time
	End      time.Time
	Location string
}

// ═══════════════════════════════════════════════════════════════════════════
// INTERFACE
// ═══════════════════════════════════════════════════════════════════════════

// COMAutomation interface for Office automation
type COMAutomation interface {
	// Excel operations
	OpenExcelWorkbook(path string) error
	InsertExcelData(data *ExcelWorkbookData) error
	RunExcelMacro(workbookPath, macroName string) error
	CloseExcel() error

	// Word operations
	GenerateWordDocument(data *WordDocumentData) error

	// Outlook operations
	SendOutlookEmail(to []string, subject, body string, attachments []string) error
	CreateOutlookCalendarEvent(subject string, start, end time.Time, location string) error

	// Health
	CheckOfficeInstallation() (map[string]bool, error)
}

// ═══════════════════════════════════════════════════════════════════════════
// PRODUCTION IMPLEMENTATION (Windows COM)
// ═══════════════════════════════════════════════════════════════════════════

// ProductionCOMAutomation implements COMAutomation using Windows COM
type ProductionCOMAutomation struct {
	excelApp    *ole.IDispatch
	wordApp     *ole.IDispatch
	outlookApp  *ole.IDispatch
	initialized bool
}

// NewProductionCOMAutomation creates COM automation client
func NewProductionCOMAutomation() (*ProductionCOMAutomation, error) {
	// Initialize OLE
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return nil, fmt.Errorf("COM initialization failed: %w", err)
	}

	return &ProductionCOMAutomation{
		initialized: true,
	}, nil
}

// OpenExcelWorkbook opens Excel workbook
func (c *ProductionCOMAutomation) OpenExcelWorkbook(path string) error {
	if c.excelApp == nil {
		// Create Excel application
		unknown, err := oleutil.CreateObject("Excel.Application")
		if err != nil {
			return fmt.Errorf("failed to create Excel application: %w", err)
		}
		c.excelApp, err = unknown.QueryInterface(ole.IID_IDispatch)
		if err != nil {
			return fmt.Errorf("failed to query Excel interface: %w", err)
		}

		// Make Excel visible (for debugging)
		oleutil.PutProperty(c.excelApp, "Visible", true)
	}

	// Open workbook
	workbooks := oleutil.MustGetProperty(c.excelApp, "Workbooks").ToIDispatch()
	defer workbooks.Release()

	_, err := oleutil.CallMethod(workbooks, "Open", path)
	if err != nil {
		return fmt.Errorf("failed to open workbook %s: %w", path, err)
	}

	return nil
}

// InsertExcelData inserts data into Excel workbook
func (c *ProductionCOMAutomation) InsertExcelData(data *ExcelWorkbookData) error {
	if c.excelApp == nil {
		if err := c.OpenExcelWorkbook(data.WorkbookPath); err != nil {
			return err
		}
	}

	// Get active workbook
	workbook := oleutil.MustGetProperty(c.excelApp, "ActiveWorkbook").ToIDispatch()
	defer workbook.Release()

	// Get or create sheet
	sheets := oleutil.MustGetProperty(workbook, "Worksheets").ToIDispatch()
	defer sheets.Release()

	var sheet *ole.IDispatch
	if data.SheetName != "" {
		sheet = oleutil.MustGetProperty(sheets, "Item", data.SheetName).ToIDispatch()
	} else {
		sheet = oleutil.MustGetProperty(sheets, "Item", 1).ToIDispatch()
	}
	defer sheet.Release()

	// Parse start cell (e.g., "A2" -> row=2, col=1)
	startRow, startCol := parseExcelCell(data.StartCell)

	// Insert data row by row
	for i, row := range data.Data {
		for j, value := range row {
			cell := oleutil.MustGetProperty(sheet, "Cells", startRow+i, startCol+j).ToIDispatch()
			oleutil.PutProperty(cell, "Value", value)
			cell.Release()
		}
	}

	// Run macro if specified
	if data.MacroName != "" {
		if err := c.RunExcelMacro(data.WorkbookPath, data.MacroName); err != nil {
			return fmt.Errorf("failed to run macro: %w", err)
		}
	}

	// Save workbook
	oleutil.CallMethod(workbook, "Save")

	return nil
}

// RunExcelMacro runs Excel VBA macro
func (c *ProductionCOMAutomation) RunExcelMacro(workbookPath, macroName string) error {
	if c.excelApp == nil {
		return fmt.Errorf("Excel not initialized")
	}

	_, err := oleutil.CallMethod(c.excelApp, "Run", macroName)
	if err != nil {
		return fmt.Errorf("failed to run macro %s: %w", macroName, err)
	}

	return nil
}

// CloseExcel closes Excel application
func (c *ProductionCOMAutomation) CloseExcel() error {
	if c.excelApp != nil {
		oleutil.CallMethod(c.excelApp, "Quit")
		c.excelApp.Release()
		c.excelApp = nil
	}
	return nil
}

// GenerateWordDocument generates Word document from template
func (c *ProductionCOMAutomation) GenerateWordDocument(data *WordDocumentData) error {
	// Create Word application
	unknown, err := oleutil.CreateObject("Word.Application")
	if err != nil {
		return fmt.Errorf("failed to create Word application: %w", err)
	}
	defer unknown.Release()

	wordApp, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("failed to query Word interface: %w", err)
	}
	defer wordApp.Release()

	// Open template
	documents := oleutil.MustGetProperty(wordApp, "Documents").ToIDispatch()
	defer documents.Release()

	doc, err := oleutil.CallMethod(documents, "Open", data.TemplatePath)
	if err != nil {
		return fmt.Errorf("failed to open template: %w", err)
	}
	docDispatch := doc.ToIDispatch()
	defer docDispatch.Release()

	// Replace placeholders
	for placeholder, value := range data.Replacements {
		// Use Find & Replace
		selection := oleutil.MustGetProperty(wordApp, "Selection").ToIDispatch()
		find := oleutil.MustGetProperty(selection, "Find").ToIDispatch()

		oleutil.PutProperty(find, "Text", placeholder)
		oleutil.CallMethod(find, "Execute")

		oleutil.PutProperty(selection, "Text", value)

		find.Release()
		selection.Release()
	}

	// Save as new document
	oleutil.CallMethod(docDispatch, "SaveAs", data.OutputPath)
	oleutil.CallMethod(docDispatch, "Close")

	return nil
}

// SendOutlookEmail sends email via Outlook
func (c *ProductionCOMAutomation) SendOutlookEmail(to []string, subject, body string, attachments []string) error {
	if c.outlookApp == nil {
		// Create Outlook application
		unknown, err := oleutil.CreateObject("Outlook.Application")
		if err != nil {
			return fmt.Errorf("failed to create Outlook application: %w", err)
		}
		c.outlookApp, err = unknown.QueryInterface(ole.IID_IDispatch)
		if err != nil {
			return fmt.Errorf("failed to query Outlook interface: %w", err)
		}
	}

	// Create mail item
	mail, err := oleutil.CallMethod(c.outlookApp, "CreateItem", 0) // 0 = olMailItem
	if err != nil {
		return fmt.Errorf("failed to create mail item: %w", err)
	}
	mailDispatch := mail.ToIDispatch()
	defer mailDispatch.Release()

	// Set recipients
	for _, recipient := range to {
		oleutil.PutProperty(mailDispatch, "To", recipient)
	}

	// Set subject and body
	oleutil.PutProperty(mailDispatch, "Subject", subject)
	oleutil.PutProperty(mailDispatch, "Body", body)

	// Add attachments
	if len(attachments) > 0 {
		attachmentsObj := oleutil.MustGetProperty(mailDispatch, "Attachments").ToIDispatch()
		defer attachmentsObj.Release()

		for _, attachment := range attachments {
			oleutil.CallMethod(attachmentsObj, "Add", attachment)
		}
	}

	// Send
	oleutil.CallMethod(mailDispatch, "Send")

	return nil
}

// CreateOutlookCalendarEvent creates calendar event
func (c *ProductionCOMAutomation) CreateOutlookCalendarEvent(subject string, start, end time.Time, location string) error {
	if c.outlookApp == nil {
		// Create Outlook application
		unknown, err := oleutil.CreateObject("Outlook.Application")
		if err != nil {
			return fmt.Errorf("failed to create Outlook application: %w", err)
		}
		c.outlookApp, err = unknown.QueryInterface(ole.IID_IDispatch)
		if err != nil {
			return fmt.Errorf("failed to query Outlook interface: %w", err)
		}
	}

	// Create appointment item
	appt, err := oleutil.CallMethod(c.outlookApp, "CreateItem", 1) // 1 = olAppointmentItem
	if err != nil {
		return fmt.Errorf("failed to create appointment: %w", err)
	}
	apptDispatch := appt.ToIDispatch()
	defer apptDispatch.Release()

	// Set properties
	oleutil.PutProperty(apptDispatch, "Subject", subject)
	oleutil.PutProperty(apptDispatch, "Start", start)
	oleutil.PutProperty(apptDispatch, "End", end)
	oleutil.PutProperty(apptDispatch, "Location", location)

	// Save
	oleutil.CallMethod(apptDispatch, "Save")

	return nil
}

// CheckOfficeInstallation checks which Office apps are installed
func (c *ProductionCOMAutomation) CheckOfficeInstallation() (map[string]bool, error) {
	result := map[string]bool{
		"Excel":   false,
		"Word":    false,
		"Outlook": false,
	}

	// Try to create each application
	apps := []string{"Excel.Application", "Word.Application", "Outlook.Application"}
	names := []string{"Excel", "Word", "Outlook"}

	for i, app := range apps {
		unknown, err := oleutil.CreateObject(app)
		if err == nil {
			result[names[i]] = true
			unknown.Release()
		}
	}

	return result, nil
}

// Cleanup releases COM resources
func (c *ProductionCOMAutomation) Cleanup() {
	c.CloseExcel()

	if c.wordApp != nil {
		c.wordApp.Release()
	}
	if c.outlookApp != nil {
		c.outlookApp.Release()
	}

	if c.initialized {
		ole.CoUninitialize()
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════════════

// parseExcelCell parses Excel cell reference (e.g., "A2" -> row=2, col=1)
func parseExcelCell(cell string) (row int, col int) {
	if len(cell) < 2 {
		return 1, 1
	}

	// Simple parser (handles A-Z columns, numeric rows)
	col = int(cell[0] - 'A' + 1)
	row = 0
	for i := 1; i < len(cell); i++ {
		if cell[i] >= '0' && cell[i] <= '9' {
			row = row*10 + int(cell[i]-'0')
		}
	}

	if row == 0 {
		row = 1
	}

	return row, col
}
