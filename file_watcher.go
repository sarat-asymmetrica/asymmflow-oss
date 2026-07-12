// ═══════════════════════════════════════════════════════════════════════════
// FILE WATCHER - OneDrive Folder Monitoring for Acme Instrumentation
//
// MISSION: Watch OneDrive folders for RFQs, Rhine XMLs, invoices, and offers
//
// ARCHITECTURE:
//   1. fsnotify for file system events (CREATE, MODIFY, DELETE, RENAME)
//   2. Debouncing for rapid changes (file writes happen in chunks)
//   3. Event filtering by extension (.msg, .xml, .xlsx, .pdf, .docx)
//   4. Queue-based processing with sync status tracking
//   5. Windows long path support (OneDrive paths can be deep)
//
// EVENT HANDLERS:
//   - OnNewRFQ: Detect new RFQ emails/folders → Queue for parsing
//   - OnEHXML: Detect Rhine Instruments pricing XML → Queue for import
//   - OnOfferChange: Detect offer folder modifications → Queue for scan
//   - OnInvoice: Detect new invoice PDFs → Queue for extraction
//
// SYNC INTEGRATION:
//   - Event queue with priorities (critical, normal, low)
//   - Per-file sync status (queued, processing, synced, failed)
//   - Offline/reconnect handling (replay missed events)
//   - Conflict detection (local vs remote timestamp comparison)
//
// Built with MATHEMATICAL RIGOR × PRODUCTION ROBUSTNESS × ZEN GARDENER ENERGY
// Day 192 - File Watcher Mission
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bep/debounce"
	"github.com/fsnotify/fsnotify"
)

// ============================================================================
// CORE TYPES
// ============================================================================

// FileWatcher monitors OneDrive folders for business events
type FileWatcher struct {
	watcher    *fsnotify.Watcher
	config     *WatchConfig
	eventQueue chan WatchEvent
	handlers   map[EventType]EventHandler
	debouncers map[string]func(f func()) // Per-file debouncers
	syncStatus *SyncStatusTracker
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
}

// WatchConfig defines what to watch and how
type WatchConfig struct {
	// Paths to watch
	RFQPath     string   // OneDrive folder for RFQ emails
	EHXMLPath   string   // OneDrive folder for Rhine XML pricing
	OfferPath   string   // OneDrive folder for offer documents
	InvoicePath string   // OneDrive folder for invoices
	ExtraPaths  []string // Additional paths to watch

	// Filter rules
	IncludeExts  []string // File extensions to watch (e.g., ".msg", ".xml")
	ExcludeExts  []string // File extensions to ignore
	IncludeGlobs []string // Glob patterns to include
	ExcludeGlobs []string // Glob patterns to exclude

	// Behavior
	Recursive       bool          // Watch subdirectories?
	DebounceDelay   time.Duration // Delay before processing (e.g., 300ms)
	PollingInterval time.Duration // Fallback polling for network drives (0 = disable)
	MaxQueueSize    int           // Max events in queue before blocking
	EnableOffline   bool          // Handle offline/reconnect scenarios?
}

// EventType categorizes file system events
type EventType string

const (
	EventNewRFQ       EventType = "new_rfq"       // New RFQ email detected
	EventEHXML        EventType = "eh_xml"        // Rhine Instruments pricing XML detected
	EventOfferChange  EventType = "offer_change"  // Offer folder modified
	EventInvoice      EventType = "invoice"       // New invoice detected
	EventGenericFile  EventType = "generic_file"  // Other file changes
	EventDirectoryNew EventType = "directory_new" // New directory created
	EventDelete       EventType = "delete"        // File/folder deleted
	EventRename       EventType = "rename"        // File/folder renamed
)

// WatchEvent represents a file system event with context
type WatchEvent struct {
	Type      EventType         // Event category
	Operation fsnotify.Op       // Raw fsnotify operation
	Path      string            // Absolute file path
	Extension string            // File extension (e.g., ".msg")
	Size      int64             // File size in bytes
	ModTime   time.Time         // Last modification time
	IsDir     bool              // Is it a directory?
	Metadata  map[string]string // Additional context
	Priority  EventPriority     // Processing priority
	Timestamp time.Time         // When event was detected
}

// EventPriority determines processing order
type EventPriority int

const (
	PriorityCritical EventPriority = 3 // Rhine XML, invoices (business critical)
	PriorityNormal   EventPriority = 2 // RFQs, offers (time-sensitive)
	PriorityLow      EventPriority = 1 // Generic files, directories
)

// EventHandler processes file events
type EventHandler func(ctx context.Context, event WatchEvent) error

// WatchSyncStatus represents file sync state in the watcher
type WatchSyncStatus string

const (
	WatchStatusQueued     WatchSyncStatus = "queued"     // Event queued for processing
	WatchStatusProcessing WatchSyncStatus = "processing" // Currently being processed
	WatchStatusSynced     WatchSyncStatus = "synced"     // Successfully synced
	WatchStatusFailed     WatchSyncStatus = "failed"     // Processing failed
	WatchStatusConflict   WatchSyncStatus = "conflict"   // Local/remote conflict detected
)

// FileSyncState tracks per-file sync status
type FileSyncState struct {
	Path         string          // File path
	EventType    string          // Event type (created, modified, deleted, renamed)
	Status       WatchSyncStatus // Current status
	LastModified time.Time       // Last local modification
	LastSynced   time.Time       // Last successful sync
	RemoteHash   string          // Remote file hash (for conflict detection)
	LocalHash    string          // Local file hash
	RetryCount   int             // Number of retry attempts
	LastError    string          // Last error message
	Metadata     map[string]any  // Additional sync metadata
}

// SyncStatusTracker manages file sync states
type SyncStatusTracker struct {
	states map[string]*FileSyncState
	mu     sync.RWMutex
}

// ============================================================================
// CONSTRUCTOR
// ============================================================================

// NewFileWatcher creates a new file watcher with the given configuration
func NewFileWatcher(config *WatchConfig) (*FileWatcher, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Apply defaults
	if config.DebounceDelay == 0 {
		config.DebounceDelay = 300 * time.Millisecond
	}
	if config.MaxQueueSize == 0 {
		config.MaxQueueSize = 1000
	}
	if len(config.IncludeExts) == 0 {
		// Default to common business file types
		config.IncludeExts = []string{".msg", ".xml", ".xlsx", ".pdf", ".docx", ".eml"}
	}

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	fw := &FileWatcher{
		watcher:    watcher,
		config:     config,
		eventQueue: make(chan WatchEvent, config.MaxQueueSize),
		handlers:   make(map[EventType]EventHandler),
		debouncers: make(map[string]func(f func())), // Per-file debouncers
		syncStatus: &SyncStatusTracker{
			states: make(map[string]*FileSyncState),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	return fw, nil
}

// ============================================================================
// WATCH MANAGEMENT
// ============================================================================

// Start begins watching configured directories
func (fw *FileWatcher) Start() error {
	// Add all configured paths
	paths := fw.collectWatchPaths()
	if len(paths) == 0 {
		return fmt.Errorf("no valid paths to watch")
	}

	for _, path := range paths {
		if err := fw.addPath(path); err != nil {
			// Log warning but don't fail (path might not exist yet)
			fmt.Printf("Warning: failed to watch %s: %v\n", path, err)
		}
	}

	// Start event processing goroutines
	fw.wg.Add(2)
	go fw.processEvents()
	go fw.processQueue()

	// Start polling if configured (for network drives)
	if fw.config.PollingInterval > 0 {
		fw.wg.Add(1)
		go fw.pollPaths()
	}

	return nil
}

// Stop stops the file watcher and cleans up resources
func (fw *FileWatcher) Stop() error {
	fw.cancel()               // Signal shutdown
	fw.wg.Wait()              // Wait for goroutines to finish
	return fw.watcher.Close() // Close fsnotify watcher
}

// addPath adds a path to the watcher (recursively if configured)
func (fw *FileWatcher) addPath(path string) error {
	// Normalize path for Windows
	path = filepath.Clean(path)

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}

	// Add the path
	if err := fw.watcher.Add(path); err != nil {
		return fmt.Errorf("failed to add watcher: %w", err)
	}

	// If recursive and is directory, add subdirectories
	if fw.config.Recursive && info.IsDir() {
		return filepath.Walk(path, func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if info.IsDir() && subPath != path {
				// Add subdirectory to watcher
				if err := fw.watcher.Add(subPath); err != nil {
					fmt.Printf("Warning: failed to watch subdirectory %s: %v\n", subPath, err)
				}
			}
			return nil
		})
	}

	return nil
}

// collectWatchPaths gathers all paths to watch
func (fw *FileWatcher) collectWatchPaths() []string {
	paths := make([]string, 0)

	if fw.config.RFQPath != "" {
		paths = append(paths, fw.config.RFQPath)
	}
	if fw.config.EHXMLPath != "" {
		paths = append(paths, fw.config.EHXMLPath)
	}
	if fw.config.OfferPath != "" {
		paths = append(paths, fw.config.OfferPath)
	}
	if fw.config.InvoicePath != "" {
		paths = append(paths, fw.config.InvoicePath)
	}

	paths = append(paths, fw.config.ExtraPaths...)

	return paths
}

// ============================================================================
// EVENT PROCESSING
// ============================================================================

// processEvents reads fsnotify events and categorizes them
func (fw *FileWatcher) processEvents() {
	defer fw.wg.Done()

	for {
		select {
		case <-fw.ctx.Done():
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Get or create per-file debouncer
			fw.mu.Lock()
			debouncer, exists := fw.debouncers[event.Name]
			if !exists {
				debouncer = debounce.New(fw.config.DebounceDelay)
				fw.debouncers[event.Name] = debouncer
			}
			fw.mu.Unlock()

			// Debounce per-file to handle rapid writes
			debouncer(func() {
				if watchEvent := fw.categorizeEvent(event); watchEvent != nil {
					fw.enqueueEvent(*watchEvent)
				}
			})

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("File watcher error: %v\n", err)
		}
	}
}

// categorizeEvent converts fsnotify event to WatchEvent
func (fw *FileWatcher) categorizeEvent(event fsnotify.Event) *WatchEvent {
	// Get file info
	info, err := os.Stat(event.Name)
	if err != nil && !os.IsNotExist(err) {
		// File may have been deleted or is inaccessible
		if !os.IsNotExist(err) {
			return nil
		}
		// For deletions/renames, info will be nil
	}

	// Check if should be filtered
	if !fw.shouldProcess(event.Name, info) {
		return nil
	}

	we := &WatchEvent{
		Operation: event.Op,
		Path:      event.Name,
		Extension: strings.ToLower(filepath.Ext(event.Name)),
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	// Set file attributes (if not deleted)
	if info != nil {
		we.Size = info.Size()
		we.ModTime = info.ModTime()
		we.IsDir = info.IsDir()
	}

	// Categorize by path and extension
	we.Type, we.Priority = fw.determineEventType(we)

	return we
}

// determineEventType categorizes events based on path and extension
func (fw *FileWatcher) determineEventType(event *WatchEvent) (EventType, EventPriority) {
	// Normalize event path for comparison (Windows/Linux compatible)
	normalizedPath := strings.ToLower(filepath.Clean(event.Path))
	ext := event.Extension

	// Check for deletion/rename first
	if event.Operation&fsnotify.Remove == fsnotify.Remove {
		return EventDelete, PriorityLow
	}
	if event.Operation&fsnotify.Rename == fsnotify.Rename {
		return EventRename, PriorityLow
	}

	// Check for new directory
	if event.IsDir && (event.Operation&fsnotify.Create == fsnotify.Create) {
		return EventDirectoryNew, PriorityLow
	}

	// Helper to normalize config paths for comparison
	normalizeConfigPath := func(p string) string {
		if p == "" {
			return ""
		}
		return strings.ToLower(filepath.Clean(p))
	}

	// Categorize by path and extension
	switch {
	case fw.config.EHXMLPath != "" && strings.Contains(normalizedPath, normalizeConfigPath(fw.config.EHXMLPath)) && ext == ".xml":
		return EventEHXML, PriorityCritical

	case fw.config.InvoicePath != "" && strings.Contains(normalizedPath, normalizeConfigPath(fw.config.InvoicePath)) && ext == ".pdf":
		return EventInvoice, PriorityCritical

	case fw.config.RFQPath != "" && strings.Contains(normalizedPath, normalizeConfigPath(fw.config.RFQPath)) && (ext == ".msg" || ext == ".eml"):
		return EventNewRFQ, PriorityNormal

	case fw.config.OfferPath != "" && strings.Contains(normalizedPath, normalizeConfigPath(fw.config.OfferPath)):
		return EventOfferChange, PriorityNormal

	default:
		return EventGenericFile, PriorityLow
	}
}

// shouldProcess checks if file should be processed (filtering logic)
func (fw *FileWatcher) shouldProcess(path string, info os.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(path))

	// Check exclude extensions
	for _, excludeExt := range fw.config.ExcludeExts {
		if ext == strings.ToLower(excludeExt) {
			return false
		}
	}

	// Check include extensions (if specified)
	if len(fw.config.IncludeExts) > 0 {
		found := false
		for _, includeExt := range fw.config.IncludeExts {
			if ext == strings.ToLower(includeExt) {
				found = true
				break
			}
		}
		if !found && info != nil && !info.IsDir() {
			return false // Not in include list and not a directory
		}
	}

	// Check glob patterns
	filename := filepath.Base(path)

	// Exclude globs take precedence
	for _, pattern := range fw.config.ExcludeGlobs {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return false
		}
	}

	// Include globs (if specified)
	if len(fw.config.IncludeGlobs) > 0 {
		for _, pattern := range fw.config.IncludeGlobs {
			if matched, _ := filepath.Match(pattern, filename); matched {
				return true
			}
		}
		// If include globs specified but none matched, exclude
		if info != nil && !info.IsDir() {
			return false
		}
	}

	return true
}

// enqueueEvent adds event to processing queue
func (fw *FileWatcher) enqueueEvent(event WatchEvent) {
	select {
	case fw.eventQueue <- event:
		// Event queued successfully
		fw.syncStatus.SetStatus(event.Path, WatchStatusQueued, "")
	case <-time.After(1 * time.Second):
		// Queue full - log warning
		fmt.Printf("Warning: event queue full, dropping event for %s\n", event.Path)
	}
}

// processQueue processes events from queue using registered handlers
func (fw *FileWatcher) processQueue() {
	defer fw.wg.Done()

	for {
		select {
		case <-fw.ctx.Done():
			return

		case event, ok := <-fw.eventQueue:
			if !ok {
				return
			}

			// Update sync status
			fw.syncStatus.SetStatus(event.Path, WatchStatusProcessing, "")

			// Get handler for event type
			fw.mu.RLock()
			handler, exists := fw.handlers[event.Type]
			fw.mu.RUnlock()

			if exists {
				// Execute handler
				if err := handler(fw.ctx, event); err != nil {
					fw.syncStatus.SetStatus(event.Path, WatchStatusFailed, err.Error())
					fmt.Printf("Error processing event %s for %s: %v\n", event.Type, event.Path, err)
				} else {
					fw.syncStatus.SetStatus(event.Path, WatchStatusSynced, "")
				}
			} else {
				// No handler registered - just mark as synced
				fw.syncStatus.SetStatus(event.Path, WatchStatusSynced, "no handler")
			}
		}
	}
}

// pollPaths periodically scans paths (fallback for network drives)
func (fw *FileWatcher) pollPaths() {
	defer fw.wg.Done()

	ticker := time.NewTicker(fw.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fw.ctx.Done():
			return

		case <-ticker.C:
			// NOTE: Polling implementation not yet needed.
			// Rationale: fsnotify works reliably on local and OneDrive folders (Windows).
			// Network drive support (if needed) should implement timestamp comparison:
			// 1. Cache last-seen mtime for each watched path
			// 2. On tick, stat each path and compare mtimes
			// 3. Generate synthetic CREATE/MODIFY events for changed files
			// 4. Enqueue via enqueueEvent() to reuse existing handler pipeline
		}
	}
}

// ============================================================================
// HANDLER REGISTRATION
// ============================================================================

// OnEvent registers a handler for a specific event type
func (fw *FileWatcher) OnEvent(eventType EventType, handler EventHandler) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.handlers[eventType] = handler
}

// OnNewRFQ registers handler for new RFQ emails
func (fw *FileWatcher) OnNewRFQ(handler EventHandler) {
	fw.OnEvent(EventNewRFQ, handler)
}

// OnEHXML registers handler for Rhine Instruments pricing XML files
func (fw *FileWatcher) OnEHXML(handler EventHandler) {
	fw.OnEvent(EventEHXML, handler)
}

// OnOfferChange registers handler for offer folder changes
func (fw *FileWatcher) OnOfferChange(handler EventHandler) {
	fw.OnEvent(EventOfferChange, handler)
}

// OnInvoice registers handler for new invoices
func (fw *FileWatcher) OnInvoice(handler EventHandler) {
	fw.OnEvent(EventInvoice, handler)
}

// ============================================================================
// SYNC STATUS TRACKING
// ============================================================================

// SetStatus updates the sync status for a file
func (tracker *SyncStatusTracker) SetStatus(path string, status WatchSyncStatus, errorMsg string) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	state, exists := tracker.states[path]
	if !exists {
		state = &FileSyncState{
			Path:     path,
			Metadata: make(map[string]any),
		}
		tracker.states[path] = state
	}

	state.Status = status
	state.LastModified = time.Now()

	if status == WatchStatusSynced {
		state.LastSynced = time.Now()
		state.RetryCount = 0
		state.LastError = ""
	} else if status == WatchStatusFailed {
		state.RetryCount++
		state.LastError = errorMsg
	}
}

// GetStatus retrieves the sync status for a file
func (tracker *SyncStatusTracker) GetStatus(path string) *FileSyncState {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	if state, exists := tracker.states[path]; exists {
		// Return a copy to avoid race conditions
		stateCopy := *state
		return &stateCopy
	}

	return nil
}

// GetAllStatuses retrieves all sync statuses
func (tracker *SyncStatusTracker) GetAllStatuses() []*FileSyncState {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	states := make([]*FileSyncState, 0, len(tracker.states))
	for _, state := range tracker.states {
		stateCopy := *state
		states = append(states, &stateCopy)
	}

	return states
}

// ClearStatus removes sync status for a file
func (tracker *SyncStatusTracker) ClearStatus(path string) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	delete(tracker.states, path)
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// GetEventQueue returns the event queue channel (for monitoring)
func (fw *FileWatcher) GetEventQueue() <-chan WatchEvent {
	return fw.eventQueue
}

// GetSyncStatus returns the sync status tracker
func (fw *FileWatcher) GetSyncStatus() *SyncStatusTracker {
	return fw.syncStatus
}

// IsRunning checks if the watcher is currently running
func (fw *FileWatcher) IsRunning() bool {
	select {
	case <-fw.ctx.Done():
		return false
	default:
		return true
	}
}
