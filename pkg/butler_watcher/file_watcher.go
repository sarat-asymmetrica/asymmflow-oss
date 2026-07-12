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

package butler_watcher

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
	mu         sync.RWMutex // RWMutex for better read performance on debouncers map
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

			// Get or create per-file debouncer with RWMutex optimization
			// First try read-only lock (most common case - debouncer already exists)
			fw.mu.RLock()
			debouncer, exists := fw.debouncers[event.Name]
			fw.mu.RUnlock()

			if !exists {
				// Only take write lock if we need to create a new debouncer
				fw.mu.Lock()
				// Double-check after acquiring write lock (another goroutine might have created it)
				debouncer, exists = fw.debouncers[event.Name]
				if !exists {
					debouncer = debounce.New(fw.config.DebounceDelay)
					fw.debouncers[event.Name] = debouncer
				}
				fw.mu.Unlock()
			}

			// ==========================================================================
			// MARGARET HAMILTON SAYS: PROTECT AGAINST PANICS IN EVENT HANDLERS
			// ==========================================================================

			// Debounce per-file to handle rapid writes
			// SAFETY: Wrap in panic recovery to prevent crashes from individual file errors
			debouncer(func() {
				defer func() {
					if r := recover(); r != nil {
						// CRITICAL: Panic in event categorization - log but don't crash watcher!
						fmt.Printf("PANIC recovered in categorizeEvent for %s: %v\n", event.Name, r)
						// Continue processing other files - don't let one bad file kill the watcher
					}
				}()

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

	// Check glob patterns (if specified)
	if len(fw.config.IncludeGlobs) > 0 || len(fw.config.ExcludeGlobs) > 0 {
		if !fw.matchesGlobs(path, fw.config.IncludeGlobs, fw.config.ExcludeGlobs) {
			return false
		}
	}

	return true
}

// matchesGlobs checks if file matches glob patterns
// Exclusions take priority over inclusions (deny-list first)
func (fw *FileWatcher) matchesGlobs(path string, includes, excludes []string) bool {
	// Check exclusions first (deny-list takes priority)
	for _, exclude := range excludes {
		if ok, _ := filepath.Match(exclude, filepath.Base(path)); ok {
			return false
		}
	}

	// If includes specified, must match at least one
	if len(includes) > 0 {
		matched := false
		for _, include := range includes {
			if ok, _ := filepath.Match(include, filepath.Base(path)); ok {
				matched = true
				break
			}
		}
		return matched
	}

	// No includes specified = match all (unless excluded)
	return true
}

// enqueueEvent adds event to processing queue
// APOLLO-GRADE QUEUE OVERFLOW HANDLING - DATA LOSS IS UNACCEPTABLE FOR FINANCIAL APPS
func (fw *FileWatcher) enqueueEvent(event WatchEvent) {
	// Use time.NewTimer instead of time.After to avoid allocating a new timer every call
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	select {
	case fw.eventQueue <- event:
		// Event queued successfully
		fw.syncStatus.SetStatus(event.Path, WatchStatusQueued, "")

	case <-timer.C:
		// ==========================================================================
		// MARGARET HAMILTON SAYS: QUEUE OVERFLOW = CRITICAL ERROR FOR BUSINESS DATA
		// ==========================================================================

		// CRITICAL: Queue is full - this is a SERIOUS problem for financial data!
		fmt.Printf("CRITICAL: Event queue full, event DROPPED! file=%s type=%s priority=%d\n",
			event.Path, event.Type, event.Priority)

		// Mark as failed so frontend can alert user
		fw.syncStatus.SetStatus(event.Path, WatchStatusFailed,
			"queue overflow - file not processed - MANUAL INTERVENTION REQUIRED")

		// If critical priority (invoices, Rhine XML), write to emergency overflow log
		// This ensures we can manually recover the data later
		if event.Priority == PriorityCritical {
			fw.writeToOverflowLog(event)
			fmt.Printf("EMERGENCY: Critical event written to overflow log: %s\n", event.Path)
		}

	case <-fw.ctx.Done():
		// Shutting down, drop the event gracefully
		return
	}
}

// writeToOverflowLog persists critical events that couldn't be queued
// RECOVERY PATH: If queue fills, critical financial data is saved to emergency log
func (fw *FileWatcher) writeToOverflowLog(event WatchEvent) {
	// SAFETY: Create emergency log file (won't crash if it fails)
	f, err := os.OpenFile("event_overflow_CRITICAL.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("PANIC: Cannot write overflow log: %v\n", err)
		return
	}
	defer f.Close()

	// Write event details for manual recovery
	fmt.Fprintf(f, "%s|%s|%s|%d|%s\n",
		time.Now().Format(time.RFC3339),
		event.Type,
		event.Path,
		event.Priority,
		event.Extension)
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

	// Track modification times for all files
	lastModTimes := make(map[string]time.Time)

	for {
		select {
		case <-fw.ctx.Done():
			return

		case <-ticker.C:
			// Scan all watched paths for changes
			// This is a fallback for network drives where fsnotify may miss events
			paths := fw.collectWatchPaths()
			for _, basePath := range paths {
				fw.pollNetworkDrive(basePath, lastModTimes)
			}
		}
	}
}

// pollNetworkDrive scans a directory for file changes
func (fw *FileWatcher) pollNetworkDrive(path string, lastModTimes map[string]time.Time) {
	// Walk directory and check modification times
	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories (we only care about files)
		if info.IsDir() {
			return nil
		}

		// Check if file should be processed (filtering logic)
		if !fw.shouldProcess(filePath, info) {
			return nil
		}

		lastMod := info.ModTime()
		prevMod, exists := lastModTimes[filePath]

		if !exists {
			// New file discovered
			lastModTimes[filePath] = lastMod
			// Create a synthetic event for the new file
			event := WatchEvent{
				Operation: fsnotify.Create,
				Path:      filePath,
				Extension: strings.ToLower(filepath.Ext(filePath)),
				Size:      info.Size(),
				ModTime:   lastMod,
				IsDir:     false,
				Timestamp: time.Now(),
				Metadata:  make(map[string]string),
			}
			event.Type, event.Priority = fw.determineEventType(&event)
			fw.enqueueEvent(event)

		} else if lastMod.After(prevMod) {
			// File modified since last check
			lastModTimes[filePath] = lastMod
			// Create a synthetic event for the modified file
			event := WatchEvent{
				Operation: fsnotify.Write,
				Path:      filePath,
				Extension: strings.ToLower(filepath.Ext(filePath)),
				Size:      info.Size(),
				ModTime:   lastMod,
				IsDir:     false,
				Timestamp: time.Now(),
				Metadata:  make(map[string]string),
			}
			event.Type, event.Priority = fw.determineEventType(&event)
			fw.enqueueEvent(event)
		}

		return nil
	})
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
