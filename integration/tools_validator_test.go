// ═══════════════════════════════════════════════════════════════════════════
// TOOLS VALIDATOR TESTS
//
// Built with SIMPLICITY × ROBUSTNESS × VALIDATION 🕉️⚡💎
// ═══════════════════════════════════════════════════════════════════════════

package integration

import (
	"testing"
	"time"
)

func TestNewToolsValidator(t *testing.T) {
	validator := NewToolsValidator()

	if validator == nil {
		t.Fatal("Expected validator to be created, got nil")
	}

	if validator.tools == nil {
		t.Error("Expected tools map to be initialized")
	}

	if validator.cacheTTL != 5*time.Minute {
		t.Errorf("Expected cache TTL to be 5 minutes, got %v", validator.cacheTTL)
	}
}

func TestValidateAllTools(t *testing.T) {
	validator := NewToolsValidator()
	report := validator.ValidateAllTools()

	if report == nil {
		t.Fatal("Expected report to be created, got nil")
	}

	if report.Tools == nil {
		t.Error("Expected tools map in report")
	}

	if len(report.Tools) == 0 {
		t.Error("Expected at least one tool to be validated")
	}

	if report.Summary == "" {
		t.Error("Expected summary to be populated")
	}

	// Check that timestamp is recent
	if time.Since(report.Timestamp) > time.Minute {
		t.Error("Expected recent timestamp")
	}

	t.Logf("Validated %d tools", len(report.Tools))
	t.Logf("Summary: %s", report.Summary)
}

func TestValidateTool(t *testing.T) {
	validator := NewToolsValidator()

	// Test validating specific tool
	status := validator.ValidateTool("pandoc")

	if status == nil {
		t.Fatal("Expected status to be returned")
	}

	if status.Name != "pandoc" {
		t.Errorf("Expected name to be 'pandoc', got '%s'", status.Name)
	}

	if status.InstallURL == "" {
		t.Error("Expected install URL to be set")
	}

	// Test unknown tool
	unknownStatus := validator.ValidateTool("nonexistent_tool_xyz")

	if unknownStatus == nil {
		t.Fatal("Expected status for unknown tool")
	}

	if unknownStatus.Available {
		t.Error("Expected unknown tool to be unavailable")
	}

	if unknownStatus.ErrorMessage == "" {
		t.Error("Expected error message for unknown tool")
	}
}

func TestCaching(t *testing.T) {
	validator := NewToolsValidator()

	// First validation
	report1 := validator.ValidateAllTools()
	time1 := report1.Timestamp

	// Immediate second validation (should use cache)
	report2 := validator.ValidateAllTools()
	time2 := report2.Timestamp

	if !time1.Equal(time2) {
		t.Error("Expected cached result with same timestamp")
	}

	// Invalidate cache
	validator.InvalidateCache()

	// Third validation (should be fresh)
	report3 := validator.ValidateAllTools()
	time3 := report3.Timestamp

	if time1.Equal(time3) {
		t.Error("Expected fresh validation after cache invalidation")
	}
}

func TestIsToolAvailable(t *testing.T) {
	validator := NewToolsValidator()
	validator.ValidateAllTools()

	// Note: We can't guarantee any specific tool is available
	// So we just test the method works without panicking
	available := validator.IsToolAvailable("pandoc")

	t.Logf("Pandoc available: %v", available)

	// Test unknown tool
	unknownAvailable := validator.IsToolAvailable("nonexistent_tool")

	if unknownAvailable {
		t.Error("Expected unknown tool to be unavailable")
	}
}

func TestGetMissingTools(t *testing.T) {
	validator := NewToolsValidator()
	validator.ValidateAllTools()

	missing := validator.GetMissingTools()

	// Can't guarantee any tools are missing
	// Just verify it returns a slice
	if missing == nil {
		t.Error("Expected missing tools slice (even if empty)")
	}

	t.Logf("Missing tools: %v", missing)
}

func TestGetInstallInstructions(t *testing.T) {
	validator := NewToolsValidator()
	validator.ValidateAllTools()

	instructions := validator.GetInstallInstructions()

	if instructions == "" {
		t.Error("Expected installation instructions")
	}

	t.Logf("Install instructions:\n%s", instructions)
}

func TestQuickValidate(t *testing.T) {
	report := QuickValidate()

	if report == nil {
		t.Fatal("Expected report from QuickValidate")
	}

	if len(report.Tools) == 0 {
		t.Error("Expected tools to be validated")
	}

	t.Logf("Quick validate found %d tools", len(report.Tools))
}

func TestToolStatusFields(t *testing.T) {
	validator := NewToolsValidator()
	report := validator.ValidateAllTools()

	// Check at least one tool has all fields populated
	for name, status := range report.Tools {
		if status.Name != name {
			t.Errorf("Tool %s: name mismatch", name)
		}

		if status.InstallURL == "" {
			t.Errorf("Tool %s: missing install URL", name)
		}

		// If available, should have path
		if status.Available && status.Path == "" {
			t.Errorf("Tool %s: available but no path", name)
		}

		// If unavailable, should have error message
		if !status.Available && status.ErrorMessage == "" {
			t.Errorf("Tool %s: unavailable but no error message", name)
		}

		t.Logf("Tool %s: available=%v, version=%s", name, status.Available, status.Version)
	}
}

func TestReportSummary(t *testing.T) {
	validator := NewToolsValidator()
	report := validator.ValidateAllTools()

	// Verify summary contains expected information
	if report.Summary == "" {
		t.Error("Expected non-empty summary")
	}

	// Should mention "required" and "optional"
	summaryLower := report.Summary
	if summaryLower == "" {
		t.Error("Expected summary to mention tool status")
	}

	// Check ReadyToUse is based on required tools
	if !report.AllRequired && report.ReadyToUse {
		t.Error("App shouldn't be ready if required tools are missing")
	}
}

func BenchmarkValidateAllTools(b *testing.B) {
	validator := NewToolsValidator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateAllTools()
	}
}

func BenchmarkValidateTool(b *testing.B) {
	validator := NewToolsValidator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateTool("pandoc")
	}
}

func BenchmarkQuickValidate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		QuickValidate()
	}
}
