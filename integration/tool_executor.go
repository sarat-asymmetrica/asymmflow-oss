// ═══════════════════════════════════════════════════════════════════════════
// TOOL EXECUTOR - External Tool Integration Layer
//
// MISSION: Integrate 9 essential external tools for document processing
//   1. Pandoc - Universal document conversion
//   2. Tesseract - OCR (text extraction from images)
//   3. Ghostscript - PDF manipulation
//   4. ImageMagick - Image processing
//   5. FFmpeg - Media processing
//   6. jq - JSON manipulation
//   7. SQLite - Local database queries
//   8. Graphviz - Diagram generation
//   9. curl/PowerShell - HTTP requests
//
// ARCHITECTURE:
//   - Graceful degradation (works without tools installed)
//   - Tool availability detection
//   - Standardized error handling
//   - Windows + cross-platform support
//
// Built with SIMPLICITY × ROBUSTNESS × ZERO_DEPENDENCIES 🕉️⚡💎
// ═══════════════════════════════════════════════════════════════════════════

package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// TYPES
// ═══════════════════════════════════════════════════════════════════════════

// Tool represents an external tool
type Tool struct {
	Name       string
	Command    string
	Version    string
	Available  bool
	InstallURL string
}

// ToolExecutor interface for external tool execution
type ToolExecutor interface {
	// Tool detection
	CheckToolAvailability() (map[string]*Tool, error)
	IsToolAvailable(name string) bool

	// Document conversion (Pandoc)
	ConvertDocument(ctx context.Context, input, output, format string) error

	// OCR (Tesseract)
	ExtractTextFromImage(ctx context.Context, imagePath string) (string, error)

	// PDF manipulation (Ghostscript)
	CompressPDF(ctx context.Context, input, output string, quality int) error
	MergePDFs(ctx context.Context, inputs []string, output string) error

	// Image processing (ImageMagick)
	ResizeImage(ctx context.Context, input, output string, width, height int) error
	ConvertImageFormat(ctx context.Context, input, output, format string) error

	// Media processing (FFmpeg)
	ExtractAudioFromVideo(ctx context.Context, input, output string) error

	// JSON manipulation (jq)
	QueryJSON(ctx context.Context, input, query string) (string, error)

	// Database (SQLite)
	ExecuteSQLQuery(ctx context.Context, dbPath, query string) (string, error)

	// Diagrams (Graphviz)
	GenerateDiagram(ctx context.Context, dotContent, output, format string) error

	// HTTP (curl/PowerShell)
	MakeHTTPRequest(ctx context.Context, url, method string, headers map[string]string) (string, error)
}

// ═══════════════════════════════════════════════════════════════════════════
// PRODUCTION IMPLEMENTATION
// ═══════════════════════════════════════════════════════════════════════════

// ProductionToolExecutor implements ToolExecutor
type ProductionToolExecutor struct {
	tools map[string]*Tool
}

// NewProductionToolExecutor creates tool executor
func NewProductionToolExecutor() *ProductionToolExecutor {
	executor := &ProductionToolExecutor{
		tools: make(map[string]*Tool),
	}

	// Detect available tools
	executor.CheckToolAvailability()

	return executor
}

// CheckToolAvailability detects which tools are installed
func (e *ProductionToolExecutor) CheckToolAvailability() (map[string]*Tool, error) {
	toolConfigs := []struct {
		name       string
		command    string
		installURL string
	}{
		{"pandoc", "pandoc", "https://pandoc.org/installing.html"},
		{"tesseract", "tesseract", "https://github.com/tesseract-ocr/tesseract"},
		{"ghostscript", "gs", "https://ghostscript.com/download/"},
		{"imagemagick", "magick", "https://imagemagick.org/script/download.php"},
		{"ffmpeg", "ffmpeg", "https://ffmpeg.org/download.html"},
		{"jq", "jq", "https://stedolan.github.io/jq/download/"},
		{"sqlite", "sqlite3", "https://www.sqlite.org/download.html"},
		{"graphviz", "dot", "https://graphviz.org/download/"},
		{"curl", "curl", "https://curl.se/download.html"},
	}

	for _, tc := range toolConfigs {
		tool := &Tool{
			Name:       tc.name,
			Command:    tc.command,
			InstallURL: tc.installURL,
		}

		// Try to execute with --version
		cmd := exec.Command(tc.command, "--version")
		suppressCommandWindow(cmd)
		output, err := cmd.CombinedOutput()
		if err == nil {
			tool.Available = true
			tool.Version = strings.Split(string(output), "\n")[0]
		}

		e.tools[tc.name] = tool
	}

	return e.tools, nil
}

// IsToolAvailable checks if specific tool is available
func (e *ProductionToolExecutor) IsToolAvailable(name string) bool {
	tool, exists := e.tools[name]
	return exists && tool.Available
}

// ═══════════════════════════════════════════════════════════════════════════
// TOOL IMPLEMENTATIONS
// ═══════════════════════════════════════════════════════════════════════════

// ConvertDocument converts document using Pandoc
func (e *ProductionToolExecutor) ConvertDocument(ctx context.Context, input, output, format string) error {
	if !e.IsToolAvailable("pandoc") {
		return fmt.Errorf("pandoc not available. Install from: %s", e.tools["pandoc"].InstallURL)
	}

	cmd := exec.CommandContext(ctx, "pandoc", input, "-o", output, "-t", format)
	suppressCommandWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc conversion failed: %w", err)
	}

	return nil
}

// ExtractTextFromImage extracts text using Tesseract OCR
func (e *ProductionToolExecutor) ExtractTextFromImage(ctx context.Context, imagePath string) (string, error) {
	if !e.IsToolAvailable("tesseract") {
		return "", fmt.Errorf("tesseract not available. Install from: %s", e.tools["tesseract"].InstallURL)
	}

	// Tesseract writes to file, so create temp output
	outputBase := filepath.Join(os.TempDir(), fmt.Sprintf("ocr_%d", time.Now().Unix()))
	outputFile := outputBase + ".txt"

	cmd := exec.CommandContext(ctx, "tesseract", imagePath, outputBase)
	suppressCommandWindow(cmd)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract OCR failed: %w", err)
	}

	// Read extracted text
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read OCR output: %w", err)
	}

	// Cleanup
	os.Remove(outputFile)

	return string(content), nil
}

// CompressPDF compresses PDF using Ghostscript
func (e *ProductionToolExecutor) CompressPDF(ctx context.Context, input, output string, quality int) error {
	if !e.IsToolAvailable("ghostscript") {
		return fmt.Errorf("ghostscript not available. Install from: %s", e.tools["ghostscript"].InstallURL)
	}

	// Quality settings: 0 = screen (low), 1 = ebook (medium), 2 = printer (high)
	qualitySettings := []string{"/screen", "/ebook", "/printer"}
	setting := qualitySettings[1] // Default to ebook
	if quality >= 0 && quality < len(qualitySettings) {
		setting = qualitySettings[quality]
	}

	cmd := exec.CommandContext(ctx, "gs",
		"-sDEVICE=pdfwrite",
		"-dCompatibilityLevel=1.4",
		"-dPDFSETTINGS="+setting,
		"-dNOPAUSE",
		"-dQUIET",
		"-dBATCH",
		"-sOutputFile="+output,
		input,
	)

	suppressCommandWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("PDF compression failed: %w", err)
	}

	return nil
}

// MergePDFs merges multiple PDFs using Ghostscript
func (e *ProductionToolExecutor) MergePDFs(ctx context.Context, inputs []string, output string) error {
	if !e.IsToolAvailable("ghostscript") {
		return fmt.Errorf("ghostscript not available. Install from: %s", e.tools["ghostscript"].InstallURL)
	}

	args := []string{
		"-sDEVICE=pdfwrite",
		"-dNOPAUSE",
		"-dQUIET",
		"-dBATCH",
		"-sOutputFile=" + output,
	}
	args = append(args, inputs...)

	cmd := exec.CommandContext(ctx, "gs", args...)
	suppressCommandWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("PDF merge failed: %w", err)
	}

	return nil
}

// ResizeImage resizes image using ImageMagick
func (e *ProductionToolExecutor) ResizeImage(ctx context.Context, input, output string, width, height int) error {
	if !e.IsToolAvailable("imagemagick") {
		return fmt.Errorf("imagemagick not available. Install from: %s", e.tools["imagemagick"].InstallURL)
	}

	size := fmt.Sprintf("%dx%d", width, height)
	cmd := exec.CommandContext(ctx, "magick", input, "-resize", size, output)

	suppressCommandWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("image resize failed: %w", err)
	}

	return nil
}

// ConvertImageFormat converts image format using ImageMagick
func (e *ProductionToolExecutor) ConvertImageFormat(ctx context.Context, input, output, format string) error {
	if !e.IsToolAvailable("imagemagick") {
		return fmt.Errorf("imagemagick not available. Install from: %s", e.tools["imagemagick"].InstallURL)
	}

	cmd := exec.CommandContext(ctx, "magick", input, output)
	suppressCommandWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("image conversion failed: %w", err)
	}

	return nil
}

// ExtractAudioFromVideo extracts audio using FFmpeg
func (e *ProductionToolExecutor) ExtractAudioFromVideo(ctx context.Context, input, output string) error {
	if !e.IsToolAvailable("ffmpeg") {
		return fmt.Errorf("ffmpeg not available. Install from: %s", e.tools["ffmpeg"].InstallURL)
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", input,
		"-vn", // No video
		"-acodec", "copy",
		output,
	)

	suppressCommandWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("audio extraction failed: %w", err)
	}

	return nil
}

// QueryJSON queries JSON using jq
func (e *ProductionToolExecutor) QueryJSON(ctx context.Context, input, query string) (string, error) {
	if !e.IsToolAvailable("jq") {
		return "", fmt.Errorf("jq not available. Install from: %s", e.tools["jq"].InstallURL)
	}

	cmd := exec.CommandContext(ctx, "jq", query, input)
	suppressCommandWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("jq query failed: %w", err)
	}

	return string(output), nil
}

// isReadOnlySQLQuery validates that a query is a safe, read-only SELECT statement.
// SECURITY: Prevents command injection via sqlite3 dot-commands (.shell, .system)
// and rejects any write/DDL operations.
func isReadOnlySQLQuery(query string) bool {
	q := strings.TrimSpace(strings.ToUpper(query))
	// Reject dot-commands (sqlite3 shell commands like .shell, .system)
	if strings.HasPrefix(q, ".") {
		return false
	}
	// Only allow SELECT queries
	if !strings.HasPrefix(q, "SELECT") {
		return false
	}
	// Reject dangerous patterns
	dangerous := []string{"DROP", "DELETE", "INSERT", "UPDATE", "ALTER", "CREATE", "ATTACH", "DETACH"}
	for _, d := range dangerous {
		if strings.Contains(q, d) {
			return false
		}
	}
	return true
}

// ExecuteSQLQuery executes SQLite query
func (e *ProductionToolExecutor) ExecuteSQLQuery(ctx context.Context, dbPath, query string) (string, error) {
	if !e.IsToolAvailable("sqlite") {
		return "", fmt.Errorf("sqlite3 not available. Install from: %s", e.tools["sqlite"].InstallURL)
	}

	// SECURITY: Validate query is read-only and contains no dot-commands
	if !isReadOnlySQLQuery(query) {
		return "", fmt.Errorf("query rejected: only read-only SELECT statements are allowed")
	}

	cmd := exec.CommandContext(ctx, "sqlite3", dbPath, query)
	suppressCommandWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("SQL query failed: %w", err)
	}

	return string(output), nil
}

// GenerateDiagram generates diagram using Graphviz
func (e *ProductionToolExecutor) GenerateDiagram(ctx context.Context, dotContent, output, format string) error {
	if !e.IsToolAvailable("graphviz") {
		return fmt.Errorf("graphviz not available. Install from: %s", e.tools["graphviz"].InstallURL)
	}

	// Write DOT content to temp file
	dotFile := filepath.Join(os.TempDir(), fmt.Sprintf("diagram_%d.dot", time.Now().Unix()))
	if err := os.WriteFile(dotFile, []byte(dotContent), 0644); err != nil {
		return fmt.Errorf("failed to write DOT file: %w", err)
	}
	defer os.Remove(dotFile)

	// Generate diagram
	cmd := exec.CommandContext(ctx, "dot",
		"-T"+format,
		"-o", output,
		dotFile,
	)

	suppressCommandWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("diagram generation failed: %w", err)
	}

	return nil
}

// MakeHTTPRequest makes HTTP request using curl
func (e *ProductionToolExecutor) MakeHTTPRequest(ctx context.Context, url, method string, headers map[string]string) (string, error) {
	if !e.IsToolAvailable("curl") {
		return "", fmt.Errorf("curl not available. Install from: %s", e.tools["curl"].InstallURL)
	}

	args := []string{"-X", method}

	// Add headers
	for key, value := range headers {
		args = append(args, "-H", fmt.Sprintf("%s: %s", key, value))
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, "curl", args...)
	suppressCommandWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}

	return string(output), nil
}

// ═══════════════════════════════════════════════════════════════════════════
// MOCK IMPLEMENTATION (For Testing)
// ═══════════════════════════════════════════════════════════════════════════

// MockToolExecutor implements ToolExecutor for testing
type MockToolExecutor struct {
	tools           map[string]*Tool
	ConversionsRun  int
	OCRsPerformed   int
	DiagramsCreated int
}

// NewMockToolExecutor creates mock executor
func NewMockToolExecutor() *MockToolExecutor {
	tools := make(map[string]*Tool)

	// Mock all tools as available
	toolNames := []string{"pandoc", "tesseract", "ghostscript", "imagemagick", "ffmpeg", "jq", "sqlite", "graphviz", "curl"}
	for _, name := range toolNames {
		tools[name] = &Tool{
			Name:      name,
			Available: true,
			Version:   "mock-1.0.0",
		}
	}

	return &MockToolExecutor{
		tools: tools,
	}
}

// CheckToolAvailability mock
func (m *MockToolExecutor) CheckToolAvailability() (map[string]*Tool, error) {
	return m.tools, nil
}

// IsToolAvailable mock
func (m *MockToolExecutor) IsToolAvailable(name string) bool {
	tool, exists := m.tools[name]
	return exists && tool.Available
}

// ConvertDocument mock
func (m *MockToolExecutor) ConvertDocument(ctx context.Context, input, output, format string) error {
	m.ConversionsRun++
	return nil
}

// ExtractTextFromImage mock
func (m *MockToolExecutor) ExtractTextFromImage(ctx context.Context, imagePath string) (string, error) {
	m.OCRsPerformed++
	return "Mock extracted text from " + imagePath, nil
}

// CompressPDF mock
func (m *MockToolExecutor) CompressPDF(ctx context.Context, input, output string, quality int) error {
	return nil
}

// MergePDFs mock
func (m *MockToolExecutor) MergePDFs(ctx context.Context, inputs []string, output string) error {
	return nil
}

// ResizeImage mock
func (m *MockToolExecutor) ResizeImage(ctx context.Context, input, output string, width, height int) error {
	return nil
}

// ConvertImageFormat mock
func (m *MockToolExecutor) ConvertImageFormat(ctx context.Context, input, output, format string) error {
	return nil
}

// ExtractAudioFromVideo mock
func (m *MockToolExecutor) ExtractAudioFromVideo(ctx context.Context, input, output string) error {
	return nil
}

// QueryJSON mock
func (m *MockToolExecutor) QueryJSON(ctx context.Context, input, query string) (string, error) {
	return `{"result": "mock"}`, nil
}

// ExecuteSQLQuery mock
func (m *MockToolExecutor) ExecuteSQLQuery(ctx context.Context, dbPath, query string) (string, error) {
	return "mock|result\n1|test", nil
}

// GenerateDiagram mock
func (m *MockToolExecutor) GenerateDiagram(ctx context.Context, dotContent, output, format string) error {
	m.DiagramsCreated++
	return nil
}

// MakeHTTPRequest mock
func (m *MockToolExecutor) MakeHTTPRequest(ctx context.Context, url, method string, headers map[string]string) (string, error) {
	return `{"status": "ok"}`, nil
}
