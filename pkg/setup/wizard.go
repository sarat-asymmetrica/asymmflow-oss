// ═══════════════════════════════════════════════════════════════════════════
// ASYMMETRICA SETUP WIZARD - First Run Configuration
//
// MISSION: Guide user through initial setup with full flourish
//
// WORKFLOW:
//   1. Welcome to Asymmetrica (animated splash)
//   2. Folder Configuration (RFQ, Offers, Invoices, Rhine XML)
//   3. API Key Setup (AIMLAPI for AI pipelines)
//   4. GPU Detection (Intel Level Zero, NVIDIA CUDA)
//   5. Initial Scan (conflict detection, report generation)
//   6. Ready to Launch!
//
// Built with MATHEMATICAL RIGOR × PRODUCTION ROBUSTNESS × ZEN GARDENER ENERGY
// Day 200 - The Convergence
// ═══════════════════════════════════════════════════════════════════════════

package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ============================================================================
// SETUP CONFIGURATION
// ============================================================================

// SetupConfig holds all first-run configuration
type SetupConfig struct {
	// Meta
	Version       string    `json:"version"`
	SetupComplete bool      `json:"setup_complete"`
	SetupDate     time.Time `json:"setup_date"`

	// Folder Paths (OneDrive or local)
	Folders FolderConfig `json:"folders"`

	// API Keys
	APIKeys APIKeyConfig `json:"api_keys"`

	// GPU Configuration
	GPU GPUConfig `json:"gpu"`

	// Office Integration
	Office OfficeConfig `json:"office"`

	// Phi-3 Local LLM
	LocalLLM LocalLLMConfig `json:"local_llm"`
}

// FolderConfig holds watched folder paths
type FolderConfig struct {
	RFQPath       string   `json:"rfq_path"`       // RFQ emails/documents
	OffersPath    string   `json:"offers_path"`    // Generated offers
	InvoicesPath  string   `json:"invoices_path"`  // Invoice PDFs
	EHXMLPath     string   `json:"eh_xml_path"`    // Rhine Instruments pricing XML
	CustomersPath string   `json:"customers_path"` // Customer documents
	ReportsPath   string   `json:"reports_path"`   // Generated reports
	ExtraPaths    []string `json:"extra_paths"`    // Additional watch paths
}

// APIKeyConfig holds external API keys
type APIKeyConfig struct {
	AIMLAPI       string `json:"aimlapi_key"`    // AIMLAPI for 200+ models
	OpenAI        string `json:"openai_key"`     // Optional: Direct OpenAI
	Anthropic     string `json:"anthropic_key"`  // Optional: Direct Anthropic
	AzureEndpoint string `json:"azure_endpoint"` // Optional: Azure OpenAI
}

// GPUConfig holds GPU detection results
type GPUConfig struct {
	Detected      bool   `json:"detected"`
	Vendor        string `json:"vendor"` // Intel, NVIDIA, AMD
	DeviceName    string `json:"device_name"`
	VRAM          int64  `json:"vram_mb"`
	LevelZeroOK   bool   `json:"level_zero_ok"`  // Intel Level Zero available
	CUDAOK        bool   `json:"cuda_ok"`        // NVIDIA CUDA available
	UseGPU        bool   `json:"use_gpu"`        // User preference
	KernelsLoaded int    `json:"kernels_loaded"` // Number of SPIR-V kernels loaded
}

// OfficeConfig holds Microsoft Office integration settings
type OfficeConfig struct {
	OutlookEnabled    bool   `json:"outlook_enabled"`
	PowerPointEnabled bool   `json:"powerpoint_enabled"`
	ExcelEnabled      bool   `json:"excel_enabled"`
	WordEnabled       bool   `json:"word_enabled"`
	DefaultEmailFrom  string `json:"default_email_from"`
	SignaturePath     string `json:"signature_path"`
	TemplatePath      string `json:"template_path"`
}

// LocalLLMConfig holds Phi-3 configuration
type LocalLLMConfig struct {
	Enabled     bool    `json:"enabled"`
	ModelPath   string  `json:"model_path"`  // Path to Phi-3 weights
	UseGPU      bool    `json:"use_gpu"`     // Use GPU for inference
	MaxTokens   int     `json:"max_tokens"`  // Max generation tokens
	Temperature float64 `json:"temperature"` // Sampling temperature
}

// ============================================================================
// SETUP WIZARD
// ============================================================================

// SetupWizard manages the first-run configuration process
type SetupWizard struct {
	config     *SetupConfig
	configPath string
	dataDir    string
}

// NewSetupWizard creates a new setup wizard
func NewSetupWizard() (*SetupWizard, error) {
	// Determine data directory
	dataDir, err := getDataDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to determine data directory: %w", err)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	configPath := filepath.Join(dataDir, "asymmetrica_config.json")

	wizard := &SetupWizard{
		config:     &SetupConfig{Version: "1.0.0"},
		configPath: configPath,
		dataDir:    dataDir,
	}

	// Try to load existing config
	if err := wizard.LoadConfig(); err != nil {
		// No existing config, will need setup
		wizard.config.SetupComplete = false
	}

	return wizard, nil
}

// NeedsSetup returns true if first-run setup is required
func (w *SetupWizard) NeedsSetup() bool {
	return !w.config.SetupComplete
}

// GetConfig returns the current configuration
func (w *SetupWizard) GetConfig() *SetupConfig {
	return w.config
}

// GetDataDir returns the data directory path
func (w *SetupWizard) GetDataDir() string {
	return w.dataDir
}

// ============================================================================
// STEP 1: FOLDER CONFIGURATION
// ============================================================================

// SetFolders configures watched folder paths
func (w *SetupWizard) SetFolders(folders FolderConfig) error {
	// Validate paths exist or can be created
	paths := []string{
		folders.RFQPath,
		folders.OffersPath,
		folders.InvoicesPath,
		folders.EHXMLPath,
		folders.CustomersPath,
		folders.ReportsPath,
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		// Create directory if it doesn't exist
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	w.config.Folders = folders
	return w.SaveConfig()
}

// SuggestFolders returns suggested folder paths based on OneDrive detection
func (w *SetupWizard) SuggestFolders() FolderConfig {
	// Try to detect OneDrive path
	oneDrivePath := detectOneDrivePath()

	if oneDrivePath != "" {
		// OneDrive detected - suggest paths within it
		return FolderConfig{
			RFQPath:       filepath.Join(oneDrivePath, "Acme Instrumentation", "RFQs"),
			OffersPath:    filepath.Join(oneDrivePath, "Acme Instrumentation", "Offers"),
			InvoicesPath:  filepath.Join(oneDrivePath, "Acme Instrumentation", "Invoices"),
			EHXMLPath:     filepath.Join(oneDrivePath, "Acme Instrumentation", "Rhine Instruments Pricing"),
			CustomersPath: filepath.Join(oneDrivePath, "Acme Instrumentation", "Customers"),
			ReportsPath:   filepath.Join(oneDrivePath, "Acme Instrumentation", "Reports"),
		}
	}

	// Fallback to Documents folder
	homeDir, _ := os.UserHomeDir()
	docsPath := filepath.Join(homeDir, "Documents", "Acme Instrumentation")

	return FolderConfig{
		RFQPath:       filepath.Join(docsPath, "RFQs"),
		OffersPath:    filepath.Join(docsPath, "Offers"),
		InvoicesPath:  filepath.Join(docsPath, "Invoices"),
		EHXMLPath:     filepath.Join(docsPath, "Rhine Instruments Pricing"),
		CustomersPath: filepath.Join(docsPath, "Customers"),
		ReportsPath:   filepath.Join(docsPath, "Reports"),
	}
}

// ============================================================================
// STEP 2: API KEY CONFIGURATION
// ============================================================================

// SetAPIKeys configures external API keys
func (w *SetupWizard) SetAPIKeys(keys APIKeyConfig) error {
	w.config.APIKeys = keys

	// Also set as environment variables for subprocess access
	if keys.AIMLAPI != "" {
		os.Setenv("ASYMM_AIML_API_KEY", keys.AIMLAPI)
	}
	if keys.OpenAI != "" {
		os.Setenv("OPENAI_API_KEY", keys.OpenAI)
	}
	if keys.Anthropic != "" {
		os.Setenv("ANTHROPIC_API_KEY", keys.Anthropic)
	}

	return w.SaveConfig()
}

// ValidateAIMLAPIKey tests if the AIMLAPI key is valid
func (w *SetupWizard) ValidateAIMLAPIKey(key string) (bool, error) {
	// Validate key format first
	if key == "" {
		return false, fmt.Errorf("API key is empty")
	}
	if len(key) < 20 {
		return false, fmt.Errorf("API key appears too short")
	}

	// Test the API key with a simple request to AIMLAPI
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", "https://api.aimlapi.com/v1/models", nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to validate API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, fmt.Errorf("API key validation failed with status: %d", resp.StatusCode)
}

// ============================================================================
// STEP 3: GPU DETECTION
// ============================================================================

// DetectGPU scans for available GPU hardware
func (w *SetupWizard) DetectGPU() GPUConfig {
	gpu := GPUConfig{
		Detected: false,
		UseGPU:   true, // Default to using GPU if available
	}

	// Try Intel Level Zero detection
	if levelZeroInfo := detectIntelLevelZero(); levelZeroInfo != nil {
		gpu.Detected = true
		gpu.Vendor = "Intel"
		gpu.DeviceName = levelZeroInfo.DeviceName
		gpu.VRAM = levelZeroInfo.VRAM
		gpu.LevelZeroOK = true
	}

	// Try NVIDIA CUDA detection
	if cudaInfo := detectNVIDIACUDA(); cudaInfo != nil {
		gpu.Detected = true
		gpu.Vendor = "NVIDIA"
		gpu.DeviceName = cudaInfo.DeviceName
		gpu.VRAM = cudaInfo.VRAM
		gpu.CUDAOK = true
	}

	// Count available SPIR-V kernels
	gpu.KernelsLoaded = countSPIRVKernels()

	w.config.GPU = gpu
	return gpu
}

// SetGPUPreference sets whether to use GPU acceleration
func (w *SetupWizard) SetGPUPreference(useGPU bool) error {
	w.config.GPU.UseGPU = useGPU
	return w.SaveConfig()
}

// ============================================================================
// STEP 4: OFFICE INTEGRATION
// ============================================================================

// DetectOffice checks for Microsoft Office installation
func (w *SetupWizard) DetectOffice() OfficeConfig {
	office := OfficeConfig{}

	// Check for Outlook
	if isOutlookInstalled() {
		office.OutlookEnabled = true
	}

	// Check for PowerPoint
	if isPowerPointInstalled() {
		office.PowerPointEnabled = true
	}

	// Check for Excel
	if isExcelInstalled() {
		office.ExcelEnabled = true
	}

	// Check for Word
	if isWordInstalled() {
		office.WordEnabled = true
	}

	w.config.Office = office
	return office
}

// SetOfficeConfig updates Office integration settings
func (w *SetupWizard) SetOfficeConfig(office OfficeConfig) error {
	w.config.Office = office
	return w.SaveConfig()
}

// ============================================================================
// STEP 5: LOCAL LLM (PHI-3)
// ============================================================================

// DetectLocalLLM checks for Phi-3 model availability
func (w *SetupWizard) DetectLocalLLM() LocalLLMConfig {
	llm := LocalLLMConfig{
		Enabled:     false,
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	// Check for Phi-3 model in common locations
	modelPaths := []string{
		filepath.Join(w.dataDir, "models", "phi-3-mini"),
		filepath.Join(w.dataDir, "phi-3-mini"),
		"C:\\Models\\phi-3-mini",
	}

	for _, path := range modelPaths {
		if _, err := os.Stat(path); err == nil {
			llm.Enabled = true
			llm.ModelPath = path
			llm.UseGPU = w.config.GPU.Detected && w.config.GPU.UseGPU
			break
		}
	}

	w.config.LocalLLM = llm
	return llm
}

// SetLocalLLMConfig updates local LLM settings
func (w *SetupWizard) SetLocalLLMConfig(llm LocalLLMConfig) error {
	w.config.LocalLLM = llm
	return w.SaveConfig()
}

// ============================================================================
// STEP 6: INITIAL SCAN
// ============================================================================

// ScanResult holds results from initial folder scan
type ScanResult struct {
	TotalFiles   int              `json:"total_files"`
	FilesByType  map[string]int   `json:"files_by_type"`
	Conflicts    []ConflictReport `json:"conflicts"`
	Warnings     []string         `json:"warnings"`
	ScanDuration time.Duration    `json:"scan_duration"`
	ReportPath   string           `json:"report_path"`
}

// ConflictReport describes a detected conflict
type ConflictReport struct {
	Path        string `json:"path"`
	Type        string `json:"type"` // duplicate, naming, encoding
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
	Severity    string `json:"severity"` // low, medium, high
}

// RunInitialScan scans configured folders and generates conflict report
func (w *SetupWizard) RunInitialScan() (*ScanResult, error) {
	startTime := time.Now()

	result := &ScanResult{
		FilesByType: make(map[string]int),
		Conflicts:   make([]ConflictReport, 0),
		Warnings:    make([]string, 0),
	}

	// Collect all paths to scan
	paths := []string{
		w.config.Folders.RFQPath,
		w.config.Folders.OffersPath,
		w.config.Folders.InvoicesPath,
		w.config.Folders.EHXMLPath,
		w.config.Folders.CustomersPath,
	}

	// Scan each path
	for _, basePath := range paths {
		if basePath == "" {
			continue
		}

		err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot access %s: %v", path, err))
				return nil
			}

			if info.IsDir() {
				return nil
			}

			result.TotalFiles++

			// Count by extension
			ext := strings.ToLower(filepath.Ext(path))
			result.FilesByType[ext]++

			// Check for conflicts
			conflicts := w.detectFileConflicts(path, info)
			result.Conflicts = append(result.Conflicts, conflicts...)

			return nil
		})

		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Error scanning %s: %v", basePath, err))
		}
	}

	result.ScanDuration = time.Since(startTime)

	// Generate report
	reportPath, err := w.generateScanReport(result)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to generate report: %v", err))
	} else {
		result.ReportPath = reportPath
	}

	return result, nil
}

// detectFileConflicts checks a file for potential issues
func (w *SetupWizard) detectFileConflicts(path string, info os.FileInfo) []ConflictReport {
	conflicts := make([]ConflictReport, 0)

	// Check for naming conflicts (special characters, too long)
	if len(filepath.Base(path)) > 200 {
		conflicts = append(conflicts, ConflictReport{
			Path:        path,
			Type:        "naming",
			Description: "Filename exceeds 200 characters",
			Suggestion:  "Rename to shorter filename",
			Severity:    "medium",
		})
	}

	// Check for potential encoding issues (non-ASCII in filename)
	basename := filepath.Base(path)
	for _, r := range basename {
		if r > 127 {
			conflicts = append(conflicts, ConflictReport{
				Path:        path,
				Type:        "encoding",
				Description: "Filename contains non-ASCII characters",
				Suggestion:  "Consider renaming with ASCII characters only",
				Severity:    "low",
			})
			break
		}
	}

	// Check for very large files
	if info.Size() > 100*1024*1024 { // > 100MB
		conflicts = append(conflicts, ConflictReport{
			Path:        path,
			Type:        "size",
			Description: fmt.Sprintf("Large file: %d MB", info.Size()/(1024*1024)),
			Suggestion:  "Large files may slow down processing",
			Severity:    "low",
		})
	}

	return conflicts
}

// generateScanReport creates a markdown report of scan results
func (w *SetupWizard) generateScanReport(result *ScanResult) (string, error) {
	reportPath := filepath.Join(w.config.Folders.ReportsPath, fmt.Sprintf("initial_scan_%s.md", time.Now().Format("20060102_150405")))

	// Ensure reports directory exists
	if err := os.MkdirAll(filepath.Dir(reportPath), 0755); err != nil {
		return "", err
	}

	f, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Write report header
	fmt.Fprintf(f, "# Asymmetrica Initial Scan Report\n\n")
	fmt.Fprintf(f, "**Generated:** %s\n\n", time.Now().Format(time.RFC1123))
	fmt.Fprintf(f, "**Scan Duration:** %v\n\n", result.ScanDuration)

	// Summary
	fmt.Fprintf(f, "## Summary\n\n")
	fmt.Fprintf(f, "- **Total Files:** %d\n", result.TotalFiles)
	fmt.Fprintf(f, "- **Conflicts Found:** %d\n", len(result.Conflicts))
	fmt.Fprintf(f, "- **Warnings:** %d\n\n", len(result.Warnings))

	// Files by type
	fmt.Fprintf(f, "## Files by Type\n\n")
	fmt.Fprintf(f, "| Extension | Count |\n")
	fmt.Fprintf(f, "|-----------|-------|\n")
	for ext, count := range result.FilesByType {
		if ext == "" {
			ext = "(no extension)"
		}
		fmt.Fprintf(f, "| %s | %d |\n", ext, count)
	}
	fmt.Fprintf(f, "\n")

	// Conflicts
	if len(result.Conflicts) > 0 {
		fmt.Fprintf(f, "## Conflicts\n\n")
		for _, c := range result.Conflicts {
			fmt.Fprintf(f, "### %s (%s)\n\n", c.Type, c.Severity)
			fmt.Fprintf(f, "- **Path:** `%s`\n", c.Path)
			fmt.Fprintf(f, "- **Issue:** %s\n", c.Description)
			fmt.Fprintf(f, "- **Suggestion:** %s\n\n", c.Suggestion)
		}
	}

	// Warnings
	if len(result.Warnings) > 0 {
		fmt.Fprintf(f, "## Warnings\n\n")
		for _, w := range result.Warnings {
			fmt.Fprintf(f, "- %s\n", w)
		}
	}

	fmt.Fprintf(f, "\n---\n\n*Report generated by Asymmetrica Setup Wizard*\n")

	return reportPath, nil
}

// ============================================================================
// COMPLETE SETUP
// ============================================================================

// CompleteSetup marks setup as complete
func (w *SetupWizard) CompleteSetup() error {
	w.config.SetupComplete = true
	w.config.SetupDate = time.Now()
	return w.SaveConfig()
}

// ============================================================================
// PERSISTENCE
// ============================================================================

// SaveConfig persists configuration to disk
func (w *SetupWizard) SaveConfig() error {
	data, err := json.MarshalIndent(w.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(w.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// LoadConfig loads configuration from disk
func (w *SetupWizard) LoadConfig() error {
	data, err := os.ReadFile(w.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	if err := json.Unmarshal(data, w.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// getDataDirectory returns the application data directory
func getDataDirectory() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "Asymmetrica"), nil
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".asymmetrica"), nil
}

// detectOneDrivePath tries to find OneDrive folder
func detectOneDrivePath() string {
	// Common OneDrive paths on Windows
	homeDir, _ := os.UserHomeDir()

	paths := []string{
		filepath.Join(homeDir, "OneDrive"),
		filepath.Join(homeDir, "OneDrive - Personal"),
		filepath.Join(homeDir, "OneDrive - Business"),
	}

	// Also check environment variable
	if envPath := os.Getenv("OneDrive"); envPath != "" {
		paths = append([]string{envPath}, paths...)
	}

	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}

	return ""
}

// GPUInfo holds detected GPU information
type GPUInfo struct {
	DeviceName string
	VRAM       int64
}

// detectIntelLevelZero checks for Intel Level Zero support
func detectIntelLevelZero() *GPUInfo {
	// Try to run a Level Zero detection command
	// This would use the actual Level Zero API in production

	// Check if ze_loader.dll exists (Windows)
	if runtime.GOOS == "windows" {
		systemRoot := os.Getenv("SystemRoot")
		dllPath := filepath.Join(systemRoot, "System32", "ze_loader.dll")
		if _, err := os.Stat(dllPath); err == nil {
			return &GPUInfo{
				DeviceName: "Intel Integrated Graphics",
				VRAM:       2048, // Default assumption for iGPU
			}
		}
	}

	return nil
}

// detectNVIDIACUDA checks for NVIDIA CUDA support
func detectNVIDIACUDA() *GPUInfo {
	// Try nvidia-smi
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Parse output
	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) >= 2 {
		var vram int64
		fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &vram)
		return &GPUInfo{
			DeviceName: strings.TrimSpace(parts[0]),
			VRAM:       vram,
		}
	}

	return nil
}

// countSPIRVKernels counts available SPIR-V kernel files
func countSPIRVKernels() int {
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return 0
	}

	// Try multiple locations for asymm_mathematical_organism
	candidatePaths := []string{
		filepath.Join(wd, "..", "..", "asymm_mathematical_organism", "geometric_consciousness_imaging", "quaternion_os_level_zero_go", "kernels"),
		filepath.Join(filepath.Dir(filepath.Dir(wd)), "asymm_mathematical_organism", "geometric_consciousness_imaging", "quaternion_os_level_zero_go", "kernels"),
	}

	// Also check environment variable
	if envRoot := os.Getenv("ASYMM_MATH_ROOT"); envRoot != "" {
		candidatePaths = append(candidatePaths, filepath.Join(envRoot, "geometric_consciousness_imaging", "quaternion_os_level_zero_go", "kernels"))
	}

	// Try local paths too
	candidatePaths = append(candidatePaths, "kernels", "../kernels")

	count := 0
	for _, basePath := range candidatePaths {
		filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(path, ".spv") {
				count++
			}
			return nil
		})
	}

	return count
}

// Office detection helpers

// isOfficeAppInstalled checks for an Office app across Windows and macOS paths
func isOfficeAppInstalled(windowsPaths []string, macAppName string) bool {
	switch runtime.GOOS {
	case "windows":
		for _, p := range windowsPaths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	case "darwin":
		if macAppName != "" {
			if _, err := os.Stat("/Applications/" + macAppName + ".app"); err == nil {
				return true
			}
		}
	}
	return false
}

func isOutlookInstalled() bool {
	return isOfficeAppInstalled([]string{
		"C:\\Program Files\\Microsoft Office\\root\\Office16\\OUTLOOK.EXE",
		"C:\\Program Files (x86)\\Microsoft Office\\root\\Office16\\OUTLOOK.EXE",
	}, "Microsoft Outlook")
}

func isPowerPointInstalled() bool {
	return isOfficeAppInstalled([]string{
		"C:\\Program Files\\Microsoft Office\\root\\Office16\\POWERPNT.EXE",
		"C:\\Program Files (x86)\\Microsoft Office\\root\\Office16\\POWERPNT.EXE",
	}, "Microsoft PowerPoint")
}

func isExcelInstalled() bool {
	return isOfficeAppInstalled([]string{
		"C:\\Program Files\\Microsoft Office\\root\\Office16\\EXCEL.EXE",
		"C:\\Program Files (x86)\\Microsoft Office\\root\\Office16\\EXCEL.EXE",
	}, "Microsoft Excel")
}

func isWordInstalled() bool {
	return isOfficeAppInstalled([]string{
		"C:\\Program Files\\Microsoft Office\\root\\Office16\\WINWORD.EXE",
		"C:\\Program Files (x86)\\Microsoft Office\\root\\Office16\\WINWORD.EXE",
	}, "Microsoft Word")
}

// ============================================================================
// CONVERSATIONAL SETUP - The Arrival Ceremony
// ============================================================================

// SystemInfo holds detected system information for the conversational setup
type SystemInfo struct {
	OS     string `json:"os"`
	CPU    string `json:"cpu"`
	GPU    string `json:"gpu"`
	RAM    string `json:"ram"`
	HasGPU bool   `json:"hasGpu"`
}

// FolderStructureResult holds the result of folder creation
type FolderStructureResult struct {
	Success   bool   `json:"success"`
	HubPath   string `json:"hubPath"`
	InboxPath string `json:"inboxPath"`
	Error     string `json:"error,omitempty"`
}

// DetectOneDrivePath attempts to find the user's OneDrive folder
func (w *SetupWizard) DetectOneDrivePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Common OneDrive locations to check
	candidates := []string{
		filepath.Join(homeDir, "OneDrive"),
		filepath.Join(homeDir, "OneDrive - Personal"),
		filepath.Join(homeDir, "OneDrive - Business"),
	}

	// Also check for company-specific OneDrive folders
	entries, err := os.ReadDir(homeDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "OneDrive") {
				candidates = append(candidates, filepath.Join(homeDir, entry.Name()))
			}
		}
	}

	// Return the first one that exists
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path, nil
		}
	}

	// Fallback to generic OneDrive path even if it doesn't exist
	return filepath.Join(homeDir, "OneDrive"), nil
}

// DetectSystemInfo gathers system information for display
func (w *SetupWizard) DetectSystemInfo() SystemInfo {
	info := SystemInfo{
		OS:     "Windows",
		CPU:    "Unknown",
		GPU:    "Unknown",
		RAM:    "Unknown",
		HasGPU: false,
	}

	// Detect OS version
	if runtime.GOOS == "windows" {
		info.OS = "Windows 11" // Simplified for now
	} else {
		info.OS = runtime.GOOS
	}

	// Try to detect CPU
	if runtime.GOOS == "windows" {
		cpuCtx, cpuCancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(cpuCtx, "wmic", "cpu", "get", "name")
		output, err := cmd.Output()
		cpuCancel()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && line != "Name" {
					info.CPU = line
					break
				}
			}
		}
	}

	// Try to detect GPU
	gpuInfo := detectIntelLevelZero()
	if gpuInfo != nil {
		info.GPU = gpuInfo.DeviceName
		info.HasGPU = true
	} else {
		// Try NVIDIA
		gpuInfo = detectNVIDIACUDA()
		if gpuInfo != nil {
			info.GPU = gpuInfo.DeviceName
			info.HasGPU = true
		}
	}

	// Try to detect RAM
	if runtime.GOOS == "windows" {
		ramCtx, ramCancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ramCtx, "wmic", "computersystem", "get", "totalphysicalmemory")
		output, err := cmd.Output()
		ramCancel()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && line != "TotalPhysicalMemory" {
					var bytes int64
					fmt.Sscanf(line, "%d", &bytes)
					gb := bytes / (1024 * 1024 * 1024)
					info.RAM = fmt.Sprintf("%dGB", gb)
					break
				}
			}
		}
	}

	return info
}

// CreateFolderStructure creates the company hub folder structure
func (w *SetupWizard) CreateFolderStructure(onedrivePath, companyName string) FolderStructureResult {
	// Sanitize company name for folder
	safeName := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, companyName)

	hubName := safeName + " Hub"
	hubPath := filepath.Join(onedrivePath, hubName)

	// Define folder structure
	folders := []struct {
		name string
		desc string
	}{
		{"Inbox", "Drop documents here for processing"},
		{"Quotations", "Generated quotes and proposals"},
		{"Customers", "Customer-related documents"},
		{"Orders", "Order confirmations and tracking"},
		{"Archive", "Processed and archived items"},
	}

	// Create hub folder
	if err := os.MkdirAll(hubPath, 0755); err != nil {
		return FolderStructureResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create hub folder: %v", err),
		}
	}

	// Create subfolders
	var inboxPath string
	for _, folder := range folders {
		folderPath := filepath.Join(hubPath, folder.name)
		if err := os.MkdirAll(folderPath, 0755); err != nil {
			return FolderStructureResult{
				Success: false,
				Error:   fmt.Sprintf("Failed to create %s folder: %v", folder.name, err),
			}
		}
		if folder.name == "Inbox" {
			inboxPath = folderPath
		}
	}

	// Update config with new paths
	w.config.Folders = FolderConfig{
		RFQPath:       inboxPath,
		OffersPath:    filepath.Join(hubPath, "Quotations"),
		InvoicesPath:  filepath.Join(hubPath, "Archive"),
		CustomersPath: filepath.Join(hubPath, "Customers"),
		ReportsPath:   filepath.Join(hubPath, "Archive"),
	}

	// Save config
	w.SaveConfig()

	return FolderStructureResult{
		Success:   true,
		HubPath:   hubPath,
		InboxPath: inboxPath,
	}
}

// ExtractBundledModel extracts the PHI3 model from the installer bundle
// For now, this is a placeholder that simulates extraction
func (w *SetupWizard) ExtractBundledModel(progressCallback func(int)) error {
	// Simulate model extraction with progress
	for i := 0; i <= 100; i += 5 {
		if progressCallback != nil {
			progressCallback(i)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Update config
	w.config.LocalLLM.Enabled = true
	w.config.LocalLLM.ModelPath = filepath.Join(w.dataDir, "models", "phi3-mini.veda")
	w.SaveConfig()

	return nil
}

// WatchInboxForTestFile starts watching the inbox for a test file
func (w *SetupWizard) WatchInboxForTestFile(inboxPath string) error {
	// This will be handled by the Butler watcher
	// For now, just verify the path exists
	if _, err := os.Stat(inboxPath); os.IsNotExist(err) {
		return fmt.Errorf("inbox path does not exist: %s", inboxPath)
	}
	return nil
}

// CompleteConversationalSetup marks setup as complete
func (w *SetupWizard) CompleteConversationalSetup() error {
	w.config.SetupComplete = true
	w.config.SetupDate = time.Now()
	return w.SaveConfig()
}
