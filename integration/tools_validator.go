// ═══════════════════════════════════════════════════════════════════════════
// TOOLS VALIDATOR - Runtime Validation with Graceful Degradation
//
// MISSION: Validate external tool availability at runtime
//   - Check each tool once at startup
//   - Cache results for performance
//   - Provide clear feedback about missing tools
//   - Distinguish required vs optional tools
//   - Enable graceful degradation
//
// PHILOSOPHY:
//   - App should START even with missing tools
//   - User gets clear feedback about what's available
//   - Required tools block related features only
//   - Optional tools degrade gracefully
//
// Built with SIMPLICITY × ROBUSTNESS × USER_EXPERIENCE 🕉️⚡💎
// ═══════════════════════════════════════════════════════════════════════════

package integration

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// TYPES
// ═══════════════════════════════════════════════════════════════════════════

// ToolStatus represents runtime status of a single tool
type ToolStatus struct {
	Name         string `json:"name"`
	Available    bool   `json:"available"`
	Path         string `json:"path"`
	Version      string `json:"version"`
	Required     bool   `json:"required"`
	InstallURL   string `json:"install_url"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ToolsReport represents complete tools validation report
type ToolsReport struct {
	AllRequired bool                   `json:"all_required"`
	AllOptional bool                   `json:"all_optional"`
	Tools       map[string]*ToolStatus `json:"tools"`
	Timestamp   time.Time              `json:"timestamp"`
	Summary     string                 `json:"summary"`
	ReadyToUse  bool                   `json:"ready_to_use"`
}

// ToolConfig defines tool validation configuration
type ToolConfig struct {
	Name        string
	Command     string
	VersionFlag string
	Required    bool
	InstallURL  string
	Description string
}

// ═══════════════════════════════════════════════════════════════════════════
// VALIDATOR
// ═══════════════════════════════════════════════════════════════════════════

// ToolsValidator manages tool validation with caching
type ToolsValidator struct {
	tools     map[string]*ToolStatus
	lastCheck time.Time
	cacheTTL  time.Duration
	mu        sync.RWMutex
}

// NewToolsValidator creates validator with default cache TTL
func NewToolsValidator() *ToolsValidator {
	return &ToolsValidator{
		tools:    make(map[string]*ToolStatus),
		cacheTTL: 5 * time.Minute, // Cache for 5 minutes
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TOOL CONFIGURATIONS
// ═══════════════════════════════════════════════════════════════════════════

var toolConfigs = []ToolConfig{
	{
		Name:        "pandoc",
		Command:     "pandoc",
		VersionFlag: "--version",
		Required:    false, // Optional - enhances document conversion
		InstallURL:  "https://pandoc.org/installing.html",
		Description: "Universal document converter",
	},
	{
		Name:        "ffmpeg",
		Command:     "ffmpeg",
		VersionFlag: "-version",
		Required:    false, // Optional - media processing
		InstallURL:  "https://ffmpeg.org/download.html",
		Description: "Media file processing",
	},
	{
		Name:        "tesseract",
		Command:     "tesseract",
		VersionFlag: "--version",
		Required:    false, // Optional - OCR functionality
		InstallURL:  "https://github.com/tesseract-ocr/tesseract",
		Description: "OCR text extraction",
	},
	{
		Name:        "imagemagick",
		Command:     "magick",
		VersionFlag: "--version",
		Required:    false, // Optional - image processing
		InstallURL:  "https://imagemagick.org/script/download.php",
		Description: "Image manipulation",
	},
	{
		Name:        "jq",
		Command:     "jq",
		VersionFlag: "--version",
		Required:    false, // Optional - JSON processing
		InstallURL:  "https://stedolan.github.io/jq/download/",
		Description: "JSON query and manipulation",
	},
	{
		Name:        "ghostscript",
		Command:     "gs",
		VersionFlag: "--version",
		Required:    false, // Optional - PDF manipulation
		InstallURL:  "https://ghostscript.com/download/",
		Description: "PDF processing and manipulation",
	},
	{
		Name:        "libreoffice",
		Command:     "soffice",
		VersionFlag: "--version",
		Required:    false, // Optional - Office document conversion
		InstallURL:  "https://www.libreoffice.org/download/",
		Description: "Office document processing",
	},
}

// ═══════════════════════════════════════════════════════════════════════════
// VALIDATION LOGIC
// ═══════════════════════════════════════════════════════════════════════════

// ValidateAllTools checks all configured tools
func (v *ToolsValidator) ValidateAllTools() *ToolsReport {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Check if cache is still valid
	if time.Since(v.lastCheck) < v.cacheTTL && len(v.tools) > 0 {
		return v.buildReport()
	}

	// Perform fresh validation
	v.tools = make(map[string]*ToolStatus)

	for _, config := range toolConfigs {
		status := v.validateTool(config)
		v.tools[config.Name] = status
	}

	v.lastCheck = time.Now()

	return v.buildReport()
}

// ValidateTool checks a specific tool
func (v *ToolsValidator) ValidateTool(name string) *ToolStatus {
	v.mu.RLock()

	// Check cache first
	if status, exists := v.tools[name]; exists && time.Since(v.lastCheck) < v.cacheTTL {
		v.mu.RUnlock()
		return status
	}
	v.mu.RUnlock()

	// Find tool config
	var config *ToolConfig
	for _, tc := range toolConfigs {
		if tc.Name == name {
			config = &tc
			break
		}
	}

	if config == nil {
		return &ToolStatus{
			Name:         name,
			Available:    false,
			ErrorMessage: "Unknown tool",
		}
	}

	// Validate and cache
	v.mu.Lock()
	status := v.validateTool(*config)
	v.tools[name] = status
	v.mu.Unlock()

	return status
}

// validateTool performs actual validation for a single tool
func (v *ToolsValidator) validateTool(config ToolConfig) *ToolStatus {
	status := &ToolStatus{
		Name:       config.Name,
		Required:   config.Required,
		InstallURL: config.InstallURL,
		Available:  false,
	}

	// Try to find the tool
	path, err := exec.LookPath(config.Command)
	if err != nil {
		status.ErrorMessage = fmt.Sprintf("Tool not found in PATH: %v", err)
		return status
	}

	status.Path = path

	// Try to get version
	cmd := exec.Command(config.Command, config.VersionFlag)
	output, err := cmd.CombinedOutput()

	if err != nil {
		status.ErrorMessage = fmt.Sprintf("Tool exists but failed version check: %v", err)
		// Tool might still work even if version flag fails
		status.Available = true // Give benefit of doubt
		return status
	}

	// Extract first line as version
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		status.Version = strings.TrimSpace(lines[0])
		// Limit version string length
		if len(status.Version) > 100 {
			status.Version = status.Version[:100] + "..."
		}
	}

	status.Available = true
	return status
}

// buildReport constructs ToolsReport from current state
func (v *ToolsValidator) buildReport() *ToolsReport {
	report := &ToolsReport{
		Tools:       v.tools,
		Timestamp:   v.lastCheck,
		AllRequired: true,
		AllOptional: true,
		ReadyToUse:  true,
	}

	requiredCount := 0
	requiredAvailable := 0
	optionalCount := 0
	optionalAvailable := 0

	for _, status := range v.tools {
		if status.Required {
			requiredCount++
			if status.Available {
				requiredAvailable++
			} else {
				report.AllRequired = false
				report.ReadyToUse = false
			}
		} else {
			optionalCount++
			if status.Available {
				optionalAvailable++
			} else {
				report.AllOptional = false
			}
		}
	}

	// Build summary message
	report.Summary = fmt.Sprintf(
		"Tools Status: %d/%d required available, %d/%d optional available",
		requiredAvailable, requiredCount,
		optionalAvailable, optionalCount,
	)

	return report
}

// IsToolAvailable checks if specific tool is available (cached)
func (v *ToolsValidator) IsToolAvailable(name string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if status, exists := v.tools[name]; exists {
		return status.Available
	}

	return false
}

// InvalidateCache forces fresh validation on next check
func (v *ToolsValidator) InvalidateCache() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.lastCheck = time.Time{} // Zero time forces revalidation
}

// GetMissingTools returns list of unavailable tools
func (v *ToolsValidator) GetMissingTools() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var missing []string
	for name, status := range v.tools {
		if !status.Available {
			missing = append(missing, name)
		}
	}

	return missing
}

// GetInstallInstructions returns formatted installation guide
func (v *ToolsValidator) GetInstallInstructions() string {
	missing := v.GetMissingTools()

	if len(missing) == 0 {
		return "All tools are available! ✓"
	}

	var sb strings.Builder
	sb.WriteString("Missing Tools Installation Guide:\n\n")

	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, name := range missing {
		if status, exists := v.tools[name]; exists {
			reqStr := "Optional"
			if status.Required {
				reqStr = "Required"
			}
			sb.WriteString(fmt.Sprintf("• %s (%s)\n", name, reqStr))
			sb.WriteString(fmt.Sprintf("  Install from: %s\n\n", status.InstallURL))
		}
	}

	return sb.String()
}

// ═══════════════════════════════════════════════════════════════════════════
// CONVENIENCE FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

// QuickValidate performs one-time validation without caching
func QuickValidate() *ToolsReport {
	validator := NewToolsValidator()
	validator.cacheTTL = 0 // Disable cache for one-time check
	return validator.ValidateAllTools()
}

// ValidateAndLog validates all tools and logs results
func ValidateAndLog() *ToolsReport {
	report := QuickValidate()

	fmt.Println("\n" + strings.Repeat("═", 70))
	fmt.Println("EXTERNAL TOOLS VALIDATION REPORT")
	fmt.Println(strings.Repeat("═", 70))
	fmt.Println(report.Summary)
	fmt.Println()

	for name, status := range report.Tools {
		statusIcon := "✓"
		if !status.Available {
			statusIcon = "✗"
		}

		reqStr := "optional"
		if status.Required {
			reqStr = "REQUIRED"
		}

		fmt.Printf("%s %s (%s)\n", statusIcon, name, reqStr)
		if status.Available {
			fmt.Printf("  Version: %s\n", status.Version)
			fmt.Printf("  Path: %s\n", status.Path)
		} else {
			fmt.Printf("  Status: Not available\n")
			fmt.Printf("  Install: %s\n", status.InstallURL)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("═", 70))

	if report.ReadyToUse {
		fmt.Println("✓ Application is READY TO USE")
	} else {
		fmt.Println("⚠ Some required tools are missing")
	}

	fmt.Println(strings.Repeat("═", 70) + "\n")

	return report
}
