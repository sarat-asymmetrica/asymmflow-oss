package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================================================================
// EVENT NAMES - Frontend subscribes to these
// ============================================================================

const (
	EventScanStarted  = "archaeologist:scan:started"
	EventScanProgress = "archaeologist:scan:progress"
	EventScanPhase    = "archaeologist:scan:phase"
	EventScanMessage  = "archaeologist:scan:message"
	EventScanFile     = "archaeologist:scan:file"
	EventScanComplete = "archaeologist:scan:complete"
	EventScanError    = "archaeologist:scan:error"
)

// ============================================================================
// EVENT PAYLOAD STRUCTS (JSON serializable for Wails)
// ============================================================================

// ScanStartedEvent - Initial event when scan begins
type ScanStartedEvent struct {
	ScanID     string    `json:"scan_id"`
	SourcePath string    `json:"source_path"`
	StartTime  time.Time `json:"start_time"`
}

// ScanProgressEvent - Periodic updates during processing
type ScanProgressEvent struct {
	ScanID         string  `json:"scan_id"`
	Phase          string  `json:"phase"`
	CurrentFile    string  `json:"current_file"`
	FilesProcessed int     `json:"files_processed"`
	TotalFiles     int     `json:"total_files"`
	Percentage     float64 `json:"percentage"`
	ElapsedMs      int64   `json:"elapsed_ms"`
}

// ScanPhaseEvent - Major phase transitions
type ScanPhaseEvent struct {
	ScanID  string `json:"scan_id"`
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

// ScanMessageEvent - Human-readable conversational updates
type ScanMessageEvent struct {
	ScanID    string    `json:"scan_id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // "info", "warn", "error", "success"
}

// ScanFileEvent - Per-file status updates
type ScanFileEvent struct {
	ScanID     string  `json:"scan_id"`
	FilePath   string  `json:"file_path"`
	FileType   string  `json:"file_type"`
	Status     string  `json:"status"` // "processing", "done", "skipped", "error"
	Confidence float64 `json:"confidence,omitempty"`
}

// ScanCompleteEvent - Final event when scan finishes
type ScanCompleteEvent struct {
	ScanID         string    `json:"scan_id"`
	Success        bool      `json:"success"`
	TotalFiles     int       `json:"total_files"`
	ProcessedFiles int       `json:"processed_files"`
	SkippedFiles   int       `json:"skipped_files"`
	ErrorFiles     int       `json:"error_files"`
	ElapsedMs      int64     `json:"elapsed_ms"`
	ReportPath     string    `json:"report_path"`
	EndTime        time.Time `json:"end_time"`
}

// ScanErrorEvent - Error notifications
type ScanErrorEvent struct {
	ScanID    string    `json:"scan_id"`
	Error     string    `json:"error"`
	FilePath  string    `json:"file_path,omitempty"`
	Phase     string    `json:"phase"`
	Timestamp time.Time `json:"timestamp"`
}

// ============================================================================
// SCAN EVENT EMITTER - Core streaming handler
// ============================================================================

// ScanEventEmitter handles progress streaming to frontend via Wails events
type ScanEventEmitter struct {
	ctx       context.Context
	scanID    string
	mu        sync.Mutex
	messages  []string
	startTime time.Time
}

// NewScanEventEmitter creates a new event emitter for a scan session
func NewScanEventEmitter(ctx context.Context, scanID string) *ScanEventEmitter {
	return &ScanEventEmitter{
		ctx:       ctx,
		scanID:    scanID,
		messages:  make([]string, 0, 100),
		startTime: time.Now(),
	}
}

// EmitStarted signals scan initialization
func (e *ScanEventEmitter) EmitStarted(sourcePath string) {
	event := ScanStartedEvent{
		ScanID:     e.scanID,
		SourcePath: sourcePath,
		StartTime:  e.startTime,
	}

	runtime.EventsEmit(e.ctx, EventScanStarted, event)
	log.Printf("[ScanEventEmitter] 🚀 Scan started: %s (ID: %s)", sourcePath, e.scanID)
}

// EmitProgress sends periodic progress updates
func (e *ScanEventEmitter) EmitProgress(phase string, currentFile string, processed, total int) {
	elapsed := time.Since(e.startTime)
	percentage := 0.0
	if total > 0 {
		percentage = (float64(processed) / float64(total)) * 100.0
	}

	event := ScanProgressEvent{
		ScanID:         e.scanID,
		Phase:          phase,
		CurrentFile:    currentFile,
		FilesProcessed: processed,
		TotalFiles:     total,
		Percentage:     percentage,
		ElapsedMs:      elapsed.Milliseconds(),
	}

	runtime.EventsEmit(e.ctx, EventScanProgress, event)
	log.Printf("[ScanEventEmitter] ⚡ Progress: %d/%d (%.1f%%) - %s",
		processed, total, percentage, currentFile)
}

// EmitPhase announces major phase transitions
func (e *ScanEventEmitter) EmitPhase(phase string, message string) {
	event := ScanPhaseEvent{
		ScanID:  e.scanID,
		Phase:   phase,
		Message: message,
	}

	runtime.EventsEmit(e.ctx, EventScanPhase, event)
	log.Printf("[ScanEventEmitter] 📍 Phase: %s - %s", phase, message)
}

// EmitMessage sends human-readable conversational updates
func (e *ScanEventEmitter) EmitMessage(message string, level string) {
	e.mu.Lock()
	e.messages = append(e.messages, message)
	e.mu.Unlock()

	event := ScanMessageEvent{
		ScanID:    e.scanID,
		Message:   message,
		Timestamp: time.Now(),
		Level:     level,
	}

	runtime.EventsEmit(e.ctx, EventScanMessage, event)

	// Log with appropriate emoji
	emoji := "ℹ️"
	switch level {
	case "success":
		emoji = "✅"
	case "warn":
		emoji = "⚠️"
	case "error":
		emoji = "❌"
	}
	log.Printf("[ScanEventEmitter] %s %s", emoji, message)
}

// EmitFileStatus updates status for a specific file
func (e *ScanEventEmitter) EmitFileStatus(filePath, fileType, status string, confidence float64) {
	event := ScanFileEvent{
		ScanID:     e.scanID,
		FilePath:   filePath,
		FileType:   fileType,
		Status:     status,
		Confidence: confidence,
	}

	runtime.EventsEmit(e.ctx, EventScanFile, event)
	log.Printf("[ScanEventEmitter] 📄 File: %s [%s] - %s (confidence: %.2f)",
		filePath, fileType, status, confidence)
}

// EmitComplete signals successful completion
func (e *ScanEventEmitter) EmitComplete(result ScanCompleteEvent) {
	// Populate timing
	result.ScanID = e.scanID
	result.EndTime = time.Now()
	result.ElapsedMs = time.Since(e.startTime).Milliseconds()

	runtime.EventsEmit(e.ctx, EventScanComplete, result)
	log.Printf("[ScanEventEmitter] 🎉 Scan complete: %d files processed in %dms",
		result.ProcessedFiles, result.ElapsedMs)
}

// EmitError reports errors
func (e *ScanEventEmitter) EmitError(err error, filePath, phase string) {
	event := ScanErrorEvent{
		ScanID:    e.scanID,
		Error:     err.Error(),
		FilePath:  filePath,
		Phase:     phase,
		Timestamp: time.Now(),
	}

	runtime.EventsEmit(e.ctx, EventScanError, event)
	log.Printf("[ScanEventEmitter] ❌ Error in phase %s: %v (file: %s)",
		phase, err, filePath)
}

// ============================================================================
// CONVERSATIONAL MESSAGE TEMPLATES
// ============================================================================

// These are examples of human-readable messages to emit during scanning.
// Use EmitMessage() with these or similar messages for a delightful UX.

var ConversationalMessages = struct {
	// Discovery phase
	StartingDiscovery  string
	FoundFiles         func(count, folders int) string
	ScanningDeeper     string
	DetectingFileTypes string

	// Processing phase
	StartingProcessing string
	ReadingDocument    func(filename string) string
	DetectedArabic     string
	DetectedEnglish    string
	DetectedBilingual  string
	LowQualityScan     string
	HighQualityScan    string
	ExtractingFields   func(count int) string

	// Completion phase
	AlmostDone       string
	GeneratingReport string
	AllDone          func(issueCount int) string
	NoIssues         string
}{
	StartingDiscovery: "Looking at your workspace...",
	FoundFiles: func(count, folders int) string {
		return fmt.Sprintf("Found %d files across %d folders", count, folders)
	},
	ScanningDeeper:     "Scanning deeper into subfolders...",
	DetectingFileTypes: "Detecting document types...",

	StartingProcessing: "Starting to read documents...",
	ReadingDocument: func(filename string) string {
		return fmt.Sprintf("Reading: %s", filename)
	},
	DetectedArabic:    "This one has Arabic text - switching to bilingual mode",
	DetectedEnglish:   "English document detected",
	DetectedBilingual: "Bilingual document (Arabic + English)",
	LowQualityScan:    "Found a low-quality scan - I'll note that for you",
	HighQualityScan:   "Crystal clear scan quality - nice!",
	ExtractingFields: func(count int) string {
		return fmt.Sprintf("Extracted %d fields from this document", count)
	},

	AlmostDone:       "Almost there... generating report",
	GeneratingReport: "Compiling archaeology report...",
	AllDone: func(issueCount int) string {
		if issueCount > 0 {
			return fmt.Sprintf("All done! Found %d things that might need your attention", issueCount)
		}
		return "All done! Everything looks good"
	},
	NoIssues: "No issues detected - all systems nominal",
}

// ============================================================================
// EXAMPLE USAGE PATTERN
// ============================================================================

/*
func ExampleArchaeologistScan(ctx context.Context, sourcePath string) error {
	// 1. Create emitter
	scanID := fmt.Sprintf("scan_%d", time.Now().Unix())
	emitter := NewScanEventEmitter(ctx, scanID)

	// 2. Start scan
	emitter.EmitStarted(sourcePath)
	emitter.EmitMessage(ConversationalMessages.StartingDiscovery, "info")

	// 3. Discovery phase
	files, folders := discoverFiles(sourcePath)
	emitter.EmitMessage(ConversationalMessages.FoundFiles(len(files), folders), "info")
	emitter.EmitPhase("discovery", "File discovery complete")

	// 4. Processing phase
	emitter.EmitPhase("processing", "Starting document processing")
	emitter.EmitMessage(ConversationalMessages.StartingProcessing, "info")

	for i, file := range files {
		// File-level progress
		emitter.EmitProgress("processing", file.Name, i+1, len(files))
		emitter.EmitMessage(ConversationalMessages.ReadingDocument(file.Name), "info")
		emitter.EmitFileStatus(file.Path, file.Type, "processing", 0)

		// Process file (OCR, extraction, etc)
		result, err := processFile(file)
		if err != nil {
			emitter.EmitError(err, file.Path, "processing")
			emitter.EmitFileStatus(file.Path, file.Type, "error", 0)
			continue
		}

		// Success
		emitter.EmitFileStatus(file.Path, file.Type, "done", result.Confidence)

		// Quality feedback
		if result.Confidence < 0.7 {
			emitter.EmitMessage(ConversationalMessages.LowQualityScan, "warn")
		}
	}

	// 5. Report generation
	emitter.EmitPhase("reporting", "Generating archaeology report")
	emitter.EmitMessage(ConversationalMessages.GeneratingReport, "info")

	reportPath, issueCount := generateReport(files)

	// 6. Complete
	emitter.EmitMessage(ConversationalMessages.AllDone(issueCount), "success")
	emitter.EmitComplete(ScanCompleteEvent{
		Success:        true,
		TotalFiles:     len(files),
		ProcessedFiles: len(files),
		ReportPath:     reportPath,
	})

	return nil
}
*/
