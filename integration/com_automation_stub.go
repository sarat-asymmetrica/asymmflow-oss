// ═══════════════════════════════════════════════════════════════════════════
// COM AUTOMATION STUB - Cross-platform stub for non-Windows systems
//
// MISSION: Provide no-op implementations for macOS/Linux builds
//
// Built with CROSS_PLATFORM × GRACEFUL_DEGRADATION 🕉️⚡💎
// ═══════════════════════════════════════════════════════════════════════════

//go:build !windows
// +build !windows

package integration

import (
	"fmt"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// STUB IMPLEMENTATION (Non-Windows)
// ═══════════════════════════════════════════════════════════════════════════

// ProductionCOMAutomation stub for non-Windows platforms
type ProductionCOMAutomation struct{}

// NewProductionCOMAutomation creates stub (always returns error on non-Windows)
func NewProductionCOMAutomation() (*ProductionCOMAutomation, error) {
	return nil, fmt.Errorf("COM automation is only available on Windows")
}

// OpenExcelWorkbook stub
func (c *ProductionCOMAutomation) OpenExcelWorkbook(path string) error {
	return fmt.Errorf("COM automation not available on this platform")
}

// InsertExcelData stub
func (c *ProductionCOMAutomation) InsertExcelData(data *ExcelWorkbookData) error {
	return fmt.Errorf("COM automation not available on this platform")
}

// RunExcelMacro stub
func (c *ProductionCOMAutomation) RunExcelMacro(workbookPath, macroName string) error {
	return fmt.Errorf("COM automation not available on this platform")
}

// CloseExcel stub
func (c *ProductionCOMAutomation) CloseExcel() error {
	return nil
}

// GenerateWordDocument stub
func (c *ProductionCOMAutomation) GenerateWordDocument(data *WordDocumentData) error {
	return fmt.Errorf("COM automation not available on this platform")
}

// SendOutlookEmail stub
func (c *ProductionCOMAutomation) SendOutlookEmail(to []string, subject, body string, attachments []string) error {
	return fmt.Errorf("COM automation not available on this platform")
}

// CreateOutlookCalendarEvent stub
func (c *ProductionCOMAutomation) CreateOutlookCalendarEvent(subject string, start, end time.Time, location string) error {
	return fmt.Errorf("COM automation not available on this platform")
}

// CheckOfficeInstallation stub
func (c *ProductionCOMAutomation) CheckOfficeInstallation() (map[string]bool, error) {
	return map[string]bool{
		"Excel":   false,
		"Word":    false,
		"Outlook": false,
	}, nil
}

// Cleanup stub
func (c *ProductionCOMAutomation) Cleanup() {}
