package integration

import (
	"fmt"
	"os"
	"path/filepath"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// OutlookAutomation for Outlook email operations via COM
type OutlookAutomation struct {
	app *ole.IDispatch
}

// NewOutlookAutomation creates COM connection to Outlook
func NewOutlookAutomation() (*OutlookAutomation, error) {
	ole.CoInitialize(0)

	unknown, err := oleutil.CreateObject("Outlook.Application")
	if err != nil {
		return nil, fmt.Errorf("Outlook not installed or COM failed: %w", err)
	}

	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, err
	}

	return &OutlookAutomation{app: app}, nil
}

// CreateDraft creates email draft in Outlook
func (o *OutlookAutomation) CreateDraft(to, subject, body string, attachmentPaths []string) error {
	// Get namespace
	ns, err := oleutil.GetProperty(o.app, "GetNamespace")
	if err != nil {
		return err
	}
	defer ns.Clear()

	// Get default mail folder
	folder, err := oleutil.CallMethod(ns.ToIDispatch(), "GetDefaultFolder", 6) // olFolderDrafts = 6
	if err != nil {
		return err
	}
	defer folder.Clear()

	// Create mail item
	items, err := oleutil.GetProperty(folder.ToIDispatch(), "Items")
	if err != nil {
		return err
	}
	defer items.Clear()

	mail, err := oleutil.CallMethod(items.ToIDispatch(), "Add")
	if err != nil {
		return err
	}

	mailItem := mail.ToIDispatch()
	defer mailItem.Release()

	// Set properties
	oleutil.PutProperty(mailItem, "To", to)
	oleutil.PutProperty(mailItem, "Subject", subject)
	oleutil.PutProperty(mailItem, "Body", body)

	// Add attachments
	for _, path := range attachmentPaths {
		if _, err := os.Stat(path); err == nil {
			attachments, _ := oleutil.GetProperty(mailItem, "Attachments")
			attachments.ToIDispatch().Release()
			oleutil.CallMethod(attachments.ToIDispatch(), "Add", path)
		}
	}

	// Save (don't send)
	oleutil.CallMethod(mailItem, "Save")

	return nil
}

// Close releases COM resources
func (o *OutlookAutomation) Close() {
	if o.app != nil {
		o.app.Release()
	}
}

// ExcelAutomation for Excel operations via COM
type ExcelAutomation struct {
	app *ole.IDispatch
}

// NewExcelAutomation creates COM connection to Excel
func NewExcelAutomation() (*ExcelAutomation, error) {
	ole.CoInitialize(0)

	unknown, err := oleutil.CreateObject("Excel.Application")
	if err != nil {
		return nil, fmt.Errorf("Excel not installed: %w", err)
	}

	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, err
	}

	// Make Excel visible
	oleutil.PutProperty(app, "Visible", true)

	return &ExcelAutomation{app: app}, nil
}

// OpenWorkbook opens Excel file
func (e *ExcelAutomation) OpenWorkbook(filePath string) error {
	absPath, _ := filepath.Abs(filePath)

	workbooks, err := oleutil.GetProperty(e.app, "Workbooks")
	if err != nil {
		return err
	}
	defer workbooks.Clear()

	_, err = oleutil.CallMethod(workbooks.ToIDispatch(), "Open", absPath)
	return err
}

// CloseWorkbook closes active workbook
func (e *ExcelAutomation) CloseWorkbook() error {
	activeBook, err := oleutil.GetProperty(e.app, "ActiveWorkbook")
	if err != nil {
		return err
	}
	defer activeBook.Clear()

	_, err = oleutil.CallMethod(activeBook.ToIDispatch(), "Close", false) // Don't save
	return err
}

// Close releases COM resources
func (e *ExcelAutomation) Close() {
	if e.app != nil {
		oleutil.CallMethod(e.app, "Quit")
		e.app.Release()
	}
}

// WordAutomation for Word operations via COM
type WordAutomation struct {
	app *ole.IDispatch
}

// NewWordAutomation creates COM connection to Word
func NewWordAutomation() (*WordAutomation, error) {
	ole.CoInitialize(0)

	unknown, err := oleutil.CreateObject("Word.Application")
	if err != nil {
		return nil, fmt.Errorf("Word not installed: %w", err)
	}

	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, err
	}

	// Make Word visible
	oleutil.PutProperty(app, "Visible", true)

	return &WordAutomation{app: app}, nil
}

// OpenDocument opens Word file
func (w *WordAutomation) OpenDocument(filePath string) error {
	absPath, _ := filepath.Abs(filePath)

	documents, err := oleutil.GetProperty(w.app, "Documents")
	if err != nil {
		return err
	}
	defer documents.Clear()

	_, err = oleutil.CallMethod(documents.ToIDispatch(), "Open", absPath)
	return err
}

// SaveAsWordML converts doc to DOCX and returns new path
func (w *WordAutomation) ExportToPDF(inputPath, outputPath string) error {
	absInput, _ := filepath.Abs(inputPath)
	absOutput, _ := filepath.Abs(outputPath)

	// Open document
	documents, _ := oleutil.GetProperty(w.app, "Documents")
	doc, _ := oleutil.CallMethod(documents.ToIDispatch(), "Open", absInput)
	docItem := doc.ToIDispatch()
	defer docItem.Release()

	// Export to PDF (wdFormatPDF = 17)
	_, err := oleutil.CallMethod(docItem, "ExportAsFixedFormat", absOutput, 17)

	// Close without saving
	oleutil.CallMethod(docItem, "Close", false)

	return err
}

// Close releases COM resources
func (w *WordAutomation) Close() {
	if w.app != nil {
		oleutil.CallMethod(w.app, "Quit")
		w.app.Release()
	}
}
