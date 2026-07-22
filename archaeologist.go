// Package main - Archaeologist Service
// σ: ASYMM-ARCHAEOLOGIST | ρ: archaeologist.go | γ: v1 | κ: Transform messy workspace to truthful map
// Author: Agent Ramanujan (maintainer + AI pair)
// Date: December 17, 2025
//
// Purpose: Convert messy workspace (folder/ZIP) into three artifacts:
//  1. Workspace Index (filesystem map - no AI)
//  2. Evidence Extracts (ACE OCR batch processing)
//  3. Archaeology Report (human-readable analysis)
//
// Philosophy: NO RECONCILIATION. NO PRICING. NO AUTOMATION. JUST SEEING.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ph_holdings_app/pkg/ocr"
)

// ═══════════════════════════════════════════════════════════════════════════
// ARCHAEOLOGIST SERVICE CORE
// ═══════════════════════════════════════════════════════════════════════════

// ArchaeologistService manages document archaeology operations
type ArchaeologistService struct {
	// ACE OCR engine for document processing
	ocrEngine *ocr.ACEEngine

	// Active scans tracking
	scans   map[string]*ArchaeologyScan
	scansMu sync.RWMutex

	// Configuration
	mistralAPIKey string
}

// ArchaeologyScan represents an active archaeology scan
type ArchaeologyScan struct {
	ID        string
	Request   *ScanRequest
	StartTime time.Time
	EndTime   *time.Time
	Status    string // "running", "completed", "failed", "cancelled"

	// Progress tracking
	Progress *ScanProgress
	Result   *ScanResult
	Error    error

	// Cancellation support
	ctx    context.Context
	cancel context.CancelFunc

	// Synchronization
	mu sync.RWMutex
}

// ═══════════════════════════════════════════════════════════════════════════
// REQUEST/RESPONSE TYPES
// ═══════════════════════════════════════════════════════════════════════════

// ScanRequest defines parameters for an archaeology scan
type ScanRequest struct {
	SourcePath string `json:"source_path"` // Folder or ZIP path
	IsZIP      bool   `json:"is_zip"`      // True if source is ZIP
	OutputDir  string `json:"output_dir"`  // Where to save artifacts
}

// ScanProgress represents current scan progress (for streaming to frontend)
type ScanProgress struct {
	Phase          string    `json:"phase"`           // "indexing", "ocr", "report_generation", "completed"
	CurrentFile    string    `json:"current_file"`    // Currently processing file
	FilesProcessed int       `json:"files_processed"` // Number of files completed
	TotalFiles     int       `json:"total_files"`     // Total files to process
	Percentage     float64   `json:"percentage"`      // Overall completion (0-100)
	Messages       []string  `json:"messages"`        // Human-readable log messages
	LastUpdate     time.Time `json:"last_update"`     // Timestamp of last update
}

// ScanResult contains final scan results and artifact paths
type ScanResult struct {
	WorkspaceIndexPath    string                 `json:"workspace_index_path"`    // Path to workspace index JSON
	EvidenceExtractsPath  string                 `json:"evidence_extracts_path"`  // Path to evidence extracts JSON
	ArchaeologyReportPath string                 `json:"archaeology_report_path"` // Path to human-readable report
	Summary               ArchaeologyScanSummary `json:"summary"`                 // High-level summary stats
}

// ArchaeologyScanSummary provides high-level statistics
type ArchaeologyScanSummary struct {
	TotalFiles          int            `json:"total_files"`           // Total files discovered
	ProcessedFiles      int            `json:"processed_files"`       // Files successfully processed
	SkippedFiles        int            `json:"skipped_files"`         // Files skipped (unsupported)
	FailedFiles         int            `json:"failed_files"`          // Files failed to process
	FileTypeBreakdown   map[string]int `json:"file_type_breakdown"`   // Count by extension
	AverageConfidence   float64        `json:"average_confidence"`    // Average OCR confidence
	TotalProcessingTime time.Duration  `json:"total_processing_time"` // Total time taken
}

// ═══════════════════════════════════════════════════════════════════════════
// WORKSPACE INDEX TYPES (Filesystem-only, no AI)
// ═══════════════════════════════════════════════════════════════════════════

// WorkspaceIndex represents the complete filesystem structure
type WorkspaceIndex struct {
	SourcePath  string           `json:"source_path"`   // Original source path
	IsZIP       bool             `json:"is_zip"`        // Was source a ZIP?
	IndexedAt   time.Time        `json:"indexed_at"`    // When was this indexed
	TotalFiles  int              `json:"total_files"`   // Total files discovered
	TotalSizeMB float64          `json:"total_size_mb"` // Total size in megabytes
	Files       []FileIndexEntry `json:"files"`         // All files found
	Directories []string         `json:"directories"`   // All directories found
}

// FileIndexEntry represents a single file in the workspace
type FileIndexEntry struct {
	Path         string    `json:"path"`          // Relative path from workspace root
	FullPath     string    `json:"full_path"`     // Absolute path
	Extension    string    `json:"extension"`     // File extension
	SizeBytes    int64     `json:"size_bytes"`    // File size
	ModifiedTime time.Time `json:"modified_time"` // Last modified timestamp
	IsSupported  bool      `json:"is_supported"`  // Can we process this file?
}

// ═══════════════════════════════════════════════════════════════════════════
// EVIDENCE EXTRACT TYPES (ACE OCR results)
// ═══════════════════════════════════════════════════════════════════════════

// EvidenceExtracts contains all OCR-extracted content
type EvidenceExtracts struct {
	SourcePath        string            `json:"source_path"`        // Original source
	ProcessedAt       time.Time         `json:"processed_at"`       // When processed
	TotalProcessed    int               `json:"total_processed"`    // Files processed
	AverageConfidence float64           `json:"average_confidence"` // Mean confidence
	Extracts          []EvidenceExtract `json:"extracts"`           // Individual extracts
}

// EvidenceExtract represents OCR result for one file
type EvidenceExtract struct {
	FilePath       string            `json:"file_path"`       // Relative path
	Extension      string            `json:"extension"`       // File extension
	ExtractedText  string            `json:"extracted_text"`  // Raw OCR text
	Confidence     float64           `json:"confidence"`      // OCR confidence
	PageCount      int               `json:"page_count"`      // Number of pages
	Fields         map[string]string `json:"fields"`          // Structured fields
	ProcessingTime time.Duration     `json:"processing_time"` // Time to process
	Tier           string            `json:"tier"`            // Processing tier used
	GPUUsed        bool              `json:"gpu_used"`        // Was GPU used?
	Error          string            `json:"error,omitempty"` // Error if failed
}

// ═══════════════════════════════════════════════════════════════════════════
// ARCHAEOLOGY REPORT TYPES (Human-readable)
// ═══════════════════════════════════════════════════════════════════════════

// ArchaeologyReportData is the final human-readable output
type ArchaeologyReportData struct {
	Title           string                    `json:"title"`
	GeneratedAt     time.Time                 `json:"generated_at"`
	SourcePath      string                    `json:"source_path"`
	Summary         ArchaeologyReportSummary  `json:"summary"`
	FileCategories  []ArchaeologyFileCategory `json:"file_categories"`
	Recommendations []string                  `json:"recommendations"`
}

// ArchaeologyReportSummary provides executive summary
type ArchaeologyReportSummary struct {
	TotalFiles        int            `json:"total_files"`
	ProcessedFiles    int            `json:"processed_files"`
	SkippedFiles      int            `json:"skipped_files"`
	FailedFiles       int            `json:"failed_files"`
	TotalSizeMB       float64        `json:"total_size_mb"`
	FileTypeBreakdown map[string]int `json:"file_type_breakdown"`
	ProcessingTime    time.Duration  `json:"processing_time"`
}

// ArchaeologyFileCategory groups files by type with insights
type ArchaeologyFileCategory struct {
	Category    string   `json:"category"`    // e.g., "Invoices", "Spreadsheets"
	Count       int      `json:"count"`       // Number of files
	Files       []string `json:"files"`       // File paths
	Observation string   `json:"observation"` // Human insight
}

// ═══════════════════════════════════════════════════════════════════════════
// SERVICE INITIALIZATION
// ═══════════════════════════════════════════════════════════════════════════

// NewArchaeologistService creates a new archaeology service
func NewArchaeologistService(mistralAPIKey string) (*ArchaeologistService, error) {
	// Create ACE OCR engine with GPU acceleration
	ocrConfig := &ocr.EngineConfig{
		EnableGPU:             true,
		GPUBackend:            ocr.GPULevelZero,
		MaxWorkers:            8,
		DefaultLanguage:       ocr.LangEnglish,
		EnablePreprocessing:   true,
		EnableVedicValidation: true,
		FallbackToMistral:     true,
		MistralAPIKey:         mistralAPIKey,
	}

	engine, err := ocr.NewACEEngine(ocrConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCR engine: %w", err)
	}

	return &ArchaeologistService{
		ocrEngine:     engine,
		scans:         make(map[string]*ArchaeologyScan),
		mistralAPIKey: mistralAPIKey,
	}, nil
}

// Close releases resources
func (s *ArchaeologistService) Close() error {
	if s.ocrEngine != nil {
		return s.ocrEngine.Close()
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// WAILS BINDINGS (Add to App struct)
// ═══════════════════════════════════════════════════════════════════════════

// StartArchaeologyScan initiates a new archaeology scan
// Returns: scan ID for tracking progress
func (a *App) StartArchaeologyScan(sourcePath string, isZIP bool, outputDir string) (string, error) {
	if err := a.requirePermission("documents:classify"); err != nil {
		return "", err
	}
	// Validate input
	if sourcePath == "" {
		return "", fmt.Errorf("source path is required")
	}
	if outputDir == "" {
		return "", fmt.Errorf("output directory is required")
	}

	// Check source exists
	if _, err := os.Stat(sourcePath); err != nil {
		return "", fmt.Errorf("source path does not exist: %w", err)
	}

	// Ensure archaeologist service exists. Key comes from the app's standard Mistral
	// key resolver (getMistralAPIKey) — never a bespoke env lookup.
	if a.archaeologist == nil {
		var err error
		a.archaeologist, err = NewArchaeologistService(getMistralAPIKey())
		if err != nil {
			return "", fmt.Errorf("failed to initialize archaeologist: %w", err)
		}
	}

	// Create scan request
	req := &ScanRequest{
		SourcePath: sourcePath,
		IsZIP:      isZIP,
		OutputDir:  outputDir,
	}

	// Generate scan ID
	scanID := fmt.Sprintf("scan_%d", time.Now().UnixNano())

	// Create cancellable context
	ctx, cancel := context.WithCancel(a.ctx)

	// Initialize scan
	scan := &ArchaeologyScan{
		ID:        scanID,
		Request:   req,
		StartTime: time.Now(),
		Status:    "running",
		Progress: &ScanProgress{
			Phase:      "indexing",
			Percentage: 0,
			Messages:   []string{"Scan started"},
			LastUpdate: time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Register scan
	a.archaeologist.scansMu.Lock()
	a.archaeologist.scans[scanID] = scan
	a.archaeologist.scansMu.Unlock()

	// Launch scan in background
	go a.runArchaeologyScan(scan)

	log.Printf("🏺 Archaeology scan started: %s (source: %s)", scanID, sourcePath)
	return scanID, nil
}

// GetScanProgress retrieves current progress for a scan
func (a *App) GetScanProgress(scanID string) (ScanProgress, error) {
	if a.archaeologist == nil {
		return ScanProgress{}, fmt.Errorf("archaeologist service not initialized")
	}

	a.archaeologist.scansMu.RLock()
	scan, exists := a.archaeologist.scans[scanID]
	a.archaeologist.scansMu.RUnlock()

	if !exists {
		return ScanProgress{}, fmt.Errorf("scan not found: %s", scanID)
	}

	scan.mu.RLock()
	defer scan.mu.RUnlock()

	if scan.Progress == nil {
		return ScanProgress{}, fmt.Errorf("scan progress not available")
	}

	return *scan.Progress, nil
}

// GetScanResult retrieves final results for a completed scan
func (a *App) GetScanResult(scanID string) (ScanResult, error) {
	if a.archaeologist == nil {
		return ScanResult{}, fmt.Errorf("archaeologist service not initialized")
	}

	a.archaeologist.scansMu.RLock()
	scan, exists := a.archaeologist.scans[scanID]
	a.archaeologist.scansMu.RUnlock()

	if !exists {
		return ScanResult{}, fmt.Errorf("scan not found: %s", scanID)
	}

	scan.mu.RLock()
	defer scan.mu.RUnlock()

	if scan.Status != "completed" {
		return ScanResult{}, fmt.Errorf("scan not completed (status: %s)", scan.Status)
	}

	if scan.Result == nil {
		return ScanResult{}, fmt.Errorf("scan result not available")
	}

	return *scan.Result, nil
}

// CancelScan cancels a running scan
func (a *App) CancelScan(scanID string) error {
	if err := a.requirePermission("documents:classify"); err != nil {
		return err
	}
	if a.archaeologist == nil {
		return fmt.Errorf("archaeologist service not initialized")
	}

	a.archaeologist.scansMu.RLock()
	scan, exists := a.archaeologist.scans[scanID]
	a.archaeologist.scansMu.RUnlock()

	if !exists {
		return fmt.Errorf("scan not found: %s", scanID)
	}

	scan.mu.Lock()
	defer scan.mu.Unlock()

	if scan.Status != "running" {
		return fmt.Errorf("scan is not running (status: %s)", scan.Status)
	}

	scan.cancel()
	scan.Status = "cancelled"
	endTime := time.Now()
	scan.EndTime = &endTime

	log.Printf("🚫 Archaeology scan cancelled: %s", scanID)
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// SCAN EXECUTION (Background goroutine)
// ═══════════════════════════════════════════════════════════════════════════

// runArchaeologyScan executes the archaeology scan workflow
func (a *App) runArchaeologyScan(scan *ArchaeologyScan) {
	defer func() {
		if r := recover(); r != nil {
			scan.mu.Lock()
			scan.Status = "failed"
			scan.Error = fmt.Errorf("scan panicked: %v", r)
			endTime := time.Now()
			scan.EndTime = &endTime
			scan.mu.Unlock()
			log.Printf("❌ Scan panic: %s - %v", scan.ID, r)

			// Emit error event
			emitter := NewScanEventEmitter(a.ctx, scan.ID)
			emitter.EmitError(scan.Error, "", "panic")
		}
	}()

	// Create event emitter for streaming to frontend
	emitter := NewScanEventEmitter(a.ctx, scan.ID)
	emitter.EmitStarted(scan.Request.SourcePath)
	emitter.EmitMessage("Looking at your workspace...", "info")

	// Phase 1: Index Workspace (30% - Emergence)
	emitter.EmitPhase("indexing", "Discovering files...")
	scan.updateProgress("indexing", "Discovering files...", 0)
	workspaceIndex, err := a.indexWorkspace(scan)
	if err != nil {
		emitter.EmitError(err, "", "indexing")
		scan.fail(err)
		return
	}
	emitter.EmitMessage(fmt.Sprintf("Found %d files across %d folders", workspaceIndex.TotalFiles, len(workspaceIndex.Directories)), "info")
	scan.updateProgress("indexing", fmt.Sprintf("Indexed %d files", workspaceIndex.TotalFiles), 30)

	// Phase 2: Process Documents (50% - Stabilization - OCR batch)
	emitter.EmitPhase("ocr", "Starting to read documents...")
	emitter.EmitMessage("Processing documents with ACE OCR...", "info")
	scan.updateProgress("ocr", "Processing documents with ACE OCR...", 30)
	evidenceExtracts, err := a.processDocuments(scan, workspaceIndex, emitter)
	if err != nil {
		emitter.EmitError(err, "", "ocr")
		scan.fail(err)
		return
	}
	emitter.EmitMessage(fmt.Sprintf("Processed %d documents (avg confidence: %.2f%%)", evidenceExtracts.TotalProcessed, evidenceExtracts.AverageConfidence*100), "success")
	scan.updateProgress("ocr", fmt.Sprintf("Processed %d documents", evidenceExtracts.TotalProcessed), 80)

	// Phase 3: Generate Report (20% - Optimization)
	emitter.EmitPhase("report_generation", "Compiling archaeology report...")
	emitter.EmitMessage("Almost there... generating report", "info")
	scan.updateProgress("report_generation", "Generating archaeology report...", 80)
	report, err := a.generateReport(scan, workspaceIndex, evidenceExtracts)
	if err != nil {
		emitter.EmitError(err, "", "report_generation")
		scan.fail(err)
		return
	}

	// Save artifacts
	emitter.EmitMessage("Saving artifacts...", "info")
	scan.updateProgress("report_generation", "Saving artifacts...", 90)
	result, err := a.saveArtifacts(scan, workspaceIndex, evidenceExtracts, report)
	if err != nil {
		emitter.EmitError(err, "", "saving")
		scan.fail(err)
		return
	}

	// Complete
	scan.complete(result)
	emitter.EmitMessage("All done! Everything looks good", "success")
	emitter.EmitComplete(ScanCompleteEvent{
		Success:        true,
		TotalFiles:     result.Summary.TotalFiles,
		ProcessedFiles: result.Summary.ProcessedFiles,
		SkippedFiles:   result.Summary.SkippedFiles,
		ErrorFiles:     result.Summary.FailedFiles,
		ReportPath:     result.ArchaeologyReportPath,
	})
	log.Printf("✅ Archaeology scan completed: %s (processed %d files)", scan.ID, result.Summary.ProcessedFiles)
}

// ═══════════════════════════════════════════════════════════════════════════
// PHASE 1: WORKSPACE INDEXING (Pure filesystem, no AI)
// ═══════════════════════════════════════════════════════════════════════════

// indexWorkspace creates a complete filesystem map
func (a *App) indexWorkspace(scan *ArchaeologyScan) (*WorkspaceIndex, error) {
	req := scan.Request
	index := &WorkspaceIndex{
		SourcePath:  req.SourcePath,
		IsZIP:       req.IsZIP,
		IndexedAt:   time.Now(),
		Files:       []FileIndexEntry{},
		Directories: []string{},
	}

	// Supported extensions for OCR processing
	supportedExts := map[string]bool{
		".pdf":  true,
		".xlsx": true,
		".docx": true,
		".msg":  true,
		".xml":  true,
		".jpg":  true,
		".png":  true,
		".eml":  true,
	}

	var totalSize int64

	// Walk filesystem
	err := filepath.Walk(req.SourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check for cancellation
		select {
		case <-scan.ctx.Done():
			return fmt.Errorf("scan cancelled")
		default:
		}

		if info.IsDir() {
			index.Directories = append(index.Directories, path)
			return nil
		}

		// File entry
		ext := strings.ToLower(filepath.Ext(path))
		relPath, _ := filepath.Rel(req.SourcePath, path)

		entry := FileIndexEntry{
			Path:         relPath,
			FullPath:     path,
			Extension:    ext,
			SizeBytes:    info.Size(),
			ModifiedTime: info.ModTime(),
			IsSupported:  supportedExts[ext],
		}

		index.Files = append(index.Files, entry)
		totalSize += info.Size()

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to index workspace: %w", err)
	}

	index.TotalFiles = len(index.Files)
	index.TotalSizeMB = float64(totalSize) / (1024 * 1024)

	log.Printf("🗂️  Workspace indexed: %d files (%.2f MB)", index.TotalFiles, index.TotalSizeMB)
	return index, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// PHASE 2: DOCUMENT PROCESSING (ACE OCR batch)
// ═══════════════════════════════════════════════════════════════════════════

// processDocuments runs ACE OCR on all supported files
func (a *App) processDocuments(scan *ArchaeologyScan, index *WorkspaceIndex, emitter *ScanEventEmitter) (*EvidenceExtracts, error) {
	extracts := &EvidenceExtracts{
		SourcePath:  scan.Request.SourcePath,
		ProcessedAt: time.Now(),
		Extracts:    []EvidenceExtract{},
	}

	// Filter to supported files
	var supportedFiles []FileIndexEntry
	for _, file := range index.Files {
		if file.IsSupported {
			supportedFiles = append(supportedFiles, file)
		}
	}

	if len(supportedFiles) == 0 {
		log.Printf("⚠️  No supported files to process")
		emitter.EmitMessage("No supported files to process", "warn")
		return extracts, nil
	}

	// Build batch request
	batchReqs := make([]*ocr.ProcessRequest, len(supportedFiles))
	for i, file := range supportedFiles {
		batchReqs[i] = &ocr.ProcessRequest{
			Source:       file.FullPath,
			SourceType:   ocr.SourceFile,
			DocumentType: detectArchaeologyDocumentType(file.Extension),
			Language:     ocr.LangEnglish,
			EnableGPU:    true,
			Tier:         ocr.TierLocal,
		}
	}

	// Progress channel - emit to both internal state and frontend
	progressChan := make(chan ocr.BatchProgress, 10)
	go func() {
		for progress := range progressChan {
			pct := 30 + (progress.Percentage * 0.5) // Map 0-100% to 30-80%
			scan.updateProgress("ocr", fmt.Sprintf("Processing %s", progress.CurrentFile), pct)

			// Emit progress and file status to frontend
			emitter.EmitProgress("ocr", progress.CurrentFile, progress.Completed, progress.Total)
			emitter.EmitFileStatus(progress.CurrentFile, filepath.Ext(progress.CurrentFile), "processing", 0)
		}
	}()

	// Execute batch
	batchReq := &ocr.BatchRequest{
		Requests:       batchReqs,
		MaxConcurrency: 4,
		ProgressChan:   progressChan,
		Context:        scan.ctx,
	}

	batchResp, err := a.archaeologist.ocrEngine.ProcessBatch(scan.ctx, batchReq)
	close(progressChan)

	if err != nil {
		return nil, fmt.Errorf("batch processing failed: %w", err)
	}

	// Convert results to extracts
	var totalConfidence float64
	for i, resp := range batchResp.Results {
		extract := EvidenceExtract{
			FilePath:       supportedFiles[i].Path,
			Extension:      supportedFiles[i].Extension,
			ExtractedText:  resp.Text,
			Confidence:     resp.Confidence,
			PageCount:      resp.PageCount,
			Fields:         resp.Fields,
			ProcessingTime: resp.ProcessingTime,
			Tier:           tierToString(resp.Tier),
			GPUUsed:        resp.GPUUsed,
		}

		if len(resp.Errors) > 0 {
			extract.Error = resp.Errors[0].Message
			emitter.EmitFileStatus(supportedFiles[i].Path, supportedFiles[i].Extension, "error", 0)
		} else {
			emitter.EmitFileStatus(supportedFiles[i].Path, supportedFiles[i].Extension, "done", resp.Confidence)

			// Quality feedback
			if resp.Confidence < 0.7 {
				emitter.EmitMessage(fmt.Sprintf("Low quality scan detected: %s", supportedFiles[i].Path), "warn")
			}
		}

		extracts.Extracts = append(extracts.Extracts, extract)
		totalConfidence += resp.Confidence
	}

	extracts.TotalProcessed = len(extracts.Extracts)
	if extracts.TotalProcessed > 0 {
		extracts.AverageConfidence = totalConfidence / float64(extracts.TotalProcessed)
	}

	log.Printf("🔍 Documents processed: %d/%d succeeded (avg confidence: %.2f%%)",
		batchResp.SuccessCount, len(supportedFiles), extracts.AverageConfidence*100)

	return extracts, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// PHASE 3: REPORT GENERATION (Human-readable insights)
// ═══════════════════════════════════════════════════════════════════════════

// generateReport creates a human-readable archaeology report
func (a *App) generateReport(scan *ArchaeologyScan, index *WorkspaceIndex, extracts *EvidenceExtracts) (*ArchaeologyReportData, error) {
	report := &ArchaeologyReportData{
		Title:       "Workspace Archaeology Report",
		GeneratedAt: time.Now(),
		SourcePath:  scan.Request.SourcePath,
		Summary: ArchaeologyReportSummary{
			TotalFiles:        index.TotalFiles,
			ProcessedFiles:    extracts.TotalProcessed,
			SkippedFiles:      index.TotalFiles - extracts.TotalProcessed,
			TotalSizeMB:       index.TotalSizeMB,
			FileTypeBreakdown: make(map[string]int),
			ProcessingTime:    time.Since(scan.StartTime),
		},
	}

	// File type breakdown
	for _, file := range index.Files {
		ext := file.Extension
		if ext == "" {
			ext = "no_extension"
		}
		report.Summary.FileTypeBreakdown[ext]++
	}

	// Categorize files
	categories := map[string]*ArchaeologyFileCategory{
		"invoices":     {Category: "Invoices", Observation: "Financial documents requiring reconciliation"},
		"spreadsheets": {Category: "Spreadsheets", Observation: "Data analysis and costing sheets"},
		"documents":    {Category: "Documents", Observation: "General correspondence and contracts"},
		"images":       {Category: "Images", Observation: "Visual documentation and scans"},
		"emails":       {Category: "Emails", Observation: "Communication records"},
		"other":        {Category: "Other", Observation: "Miscellaneous files"},
	}

	for _, file := range index.Files {
		var cat *ArchaeologyFileCategory
		switch strings.ToLower(file.Extension) {
		case ".pdf":
			cat = categories["invoices"]
		case ".xlsx", ".csv":
			cat = categories["spreadsheets"]
		case ".docx", ".doc", ".txt":
			cat = categories["documents"]
		case ".jpg", ".png", ".tiff":
			cat = categories["images"]
		case ".msg", ".eml":
			cat = categories["emails"]
		default:
			cat = categories["other"]
		}

		cat.Count++
		cat.Files = append(cat.Files, file.Path)
	}

	// Add non-empty categories to report
	for _, cat := range categories {
		if cat.Count > 0 {
			report.FileCategories = append(report.FileCategories, *cat)
		}
	}

	// Recommendations
	report.Recommendations = []string{
		fmt.Sprintf("Total of %d files discovered across %d directories", index.TotalFiles, len(index.Directories)),
		fmt.Sprintf("Successfully processed %d documents with average confidence %.2f%%", extracts.TotalProcessed, extracts.AverageConfidence*100),
		"Review Evidence Extracts for detailed OCR content",
		"Use Workspace Index for complete filesystem structure",
	}

	if report.Summary.SkippedFiles > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("%d files skipped (unsupported format)", report.Summary.SkippedFiles))
	}

	return report, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ARTIFACT PERSISTENCE
// ═══════════════════════════════════════════════════════════════════════════

// saveArtifacts writes all three artifacts to disk
func (a *App) saveArtifacts(scan *ArchaeologyScan, index *WorkspaceIndex, extracts *EvidenceExtracts, report *ArchaeologyReportData) (*ScanResult, error) {
	outputDir := scan.Request.OutputDir

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")

	// 1. Workspace Index
	indexPath := filepath.Join(outputDir, fmt.Sprintf("workspace_index_%s.json", timestamp))
	if err := saveJSON(indexPath, index); err != nil {
		return nil, fmt.Errorf("failed to save workspace index: %w", err)
	}

	// 2. Evidence Extracts
	extractsPath := filepath.Join(outputDir, fmt.Sprintf("evidence_extracts_%s.json", timestamp))
	if err := saveJSON(extractsPath, extracts); err != nil {
		return nil, fmt.Errorf("failed to save evidence extracts: %w", err)
	}

	// 3. Archaeology Report
	reportPath := filepath.Join(outputDir, fmt.Sprintf("archaeology_report_%s.json", timestamp))
	if err := saveJSON(reportPath, report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	// Build result summary
	result := &ScanResult{
		WorkspaceIndexPath:    indexPath,
		EvidenceExtractsPath:  extractsPath,
		ArchaeologyReportPath: reportPath,
		Summary: ArchaeologyScanSummary{
			TotalFiles:          index.TotalFiles,
			ProcessedFiles:      extracts.TotalProcessed,
			SkippedFiles:        index.TotalFiles - extracts.TotalProcessed,
			FileTypeBreakdown:   report.Summary.FileTypeBreakdown,
			AverageConfidence:   extracts.AverageConfidence,
			TotalProcessingTime: time.Since(scan.StartTime),
		},
	}

	log.Printf("💾 Artifacts saved to: %s", outputDir)
	return result, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// HELPER FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

// updateProgress updates scan progress (thread-safe)
func (s *ArchaeologyScan) updateProgress(phase string, message string, percentage float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Progress == nil {
		s.Progress = &ScanProgress{}
	}

	s.Progress.Phase = phase
	s.Progress.Percentage = percentage
	s.Progress.Messages = append(s.Progress.Messages, message)
	s.Progress.LastUpdate = time.Now()

	// Keep last 50 messages
	if len(s.Progress.Messages) > 50 {
		s.Progress.Messages = s.Progress.Messages[len(s.Progress.Messages)-50:]
	}
}

// complete marks scan as completed
func (s *ArchaeologyScan) complete(result *ScanResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = "completed"
	s.Result = result
	endTime := time.Now()
	s.EndTime = &endTime

	if s.Progress != nil {
		s.Progress.Phase = "completed"
		s.Progress.Percentage = 100
		s.Progress.Messages = append(s.Progress.Messages, "Scan completed successfully")
		s.Progress.LastUpdate = time.Now()
	}
}

// fail marks scan as failed
func (s *ArchaeologyScan) fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = "failed"
	s.Error = err
	endTime := time.Now()
	s.EndTime = &endTime

	if s.Progress != nil {
		s.Progress.Messages = append(s.Progress.Messages, fmt.Sprintf("Scan failed: %v", err))
		s.Progress.LastUpdate = time.Now()
	}

	log.Printf("❌ Scan failed: %s - %v", s.ID, err)
}

// detectArchaeologyDocumentType maps file extension to document type
func detectArchaeologyDocumentType(ext string) ocr.DocumentType {
	switch strings.ToLower(ext) {
	case ".pdf":
		return ocr.DocTypeInvoice // Default PDFs to invoice type
	case ".xlsx":
		return ocr.DocTypeBOQ
	case ".docx":
		return ocr.DocTypeContract
	case ".msg", ".eml":
		return ocr.DocTypeLetterhead
	case ".jpg", ".png":
		return ocr.DocTypeGeneric
	default:
		return ocr.DocTypeUnknown
	}
}

// tierToString converts ProcessingTier to string
func tierToString(tier ocr.ProcessingTier) string {
	switch tier {
	case ocr.TierLocal:
		return "local"
	case ocr.TierCloudOCR:
		return "cloud_ocr"
	case ocr.TierConsensus:
		return "consensus"
	default:
		return "unknown"
	}
}

// saveJSON writes a struct to JSON file
func saveJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
