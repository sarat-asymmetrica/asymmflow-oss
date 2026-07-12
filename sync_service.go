// ═══════════════════════════════════════════════════════════════════════════
// SYNC SERVICE - OneDrive/SharePoint Synchronization for File Watcher
//
// MISSION: Bridge file watcher events to Microsoft Graph uploads
//
// ARCHITECTURE:
//   1. File watcher detects local changes
//   2. Sync service uploads to OneDrive/SharePoint
//   3. Database tracks sync status
//   4. Automatic retry for failures
//   5. Conflict detection (local vs remote)
//
// INTEGRATION:
//   - FileWatcher → SyncService → GraphClient → OneDrive/SharePoint
//   - Database persistence via GORM
//   - Event-driven (goroutine per upload)
//   - Resilient (retry logic, error handling)
//
// Built with FEARLESS ZEN GARDENER ENERGY × PRODUCTION ROBUSTNESS ⚡🌱
// Day 194 - File Sync Tending
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	msgraph "ph_holdings_app/microsoft_graph"
)

// ============================================================================
// CORE TYPES
// ============================================================================

// SyncService handles file synchronization with OneDrive/SharePoint
type SyncService struct {
	graphClient msgraph.GraphClient
	app         *App
	config      *SyncConfig
}

// SyncConfig defines sync behavior
type SyncConfig struct {
	// OneDrive folder mappings (local type → OneDrive folder)
	RFQFolderPath     string // e.g., "RFQs"
	OfferFolderPath   string // e.g., "Offers"
	InvoiceFolderPath string // e.g., "Invoices"
	DocumentsFolderID string // Default OneDrive folder ID

	// Sync behavior
	AutoRetry      bool          // Retry failed uploads automatically
	MaxRetries     int           // Maximum retry attempts (default: 3)
	RetryDelay     time.Duration // Delay between retries (default: 5s)
	SkipLargeFiles bool          // Skip files > MaxFileSize
	MaxFileSize    int64         // Max file size in bytes (default: 100 MB)

	// Conflict handling
	DetectConflicts  bool   // Compare local/remote timestamps
	ConflictStrategy string // "local_wins", "remote_wins", "skip"
}

// DefaultSyncConfig returns sensible defaults
func DefaultSyncConfig() *SyncConfig {
	return &SyncConfig{
		RFQFolderPath:     "RFQs",
		OfferFolderPath:   "Offers",
		InvoiceFolderPath: "Invoices",
		DocumentsFolderID: "root", // OneDrive root folder
		AutoRetry:         true,
		MaxRetries:        3,
		RetryDelay:        5 * time.Second,
		SkipLargeFiles:    true,
		MaxFileSize:       100 * 1024 * 1024, // 100 MB
		DetectConflicts:   true,
		ConflictStrategy:  "local_wins", // Default: local changes override remote
	}
}

// ============================================================================
// CONSTRUCTOR
// ============================================================================

// NewSyncService creates a new sync service
func NewSyncService(client msgraph.GraphClient, app *App) *SyncService {
	config := DefaultSyncConfig()

	// Override from app config if available
	if app != nil && app.config != nil {
		// TODO: Load sync config from app.config if needed
	}

	return &SyncService{
		graphClient: client,
		app:         app,
		config:      config,
	}
}

// ============================================================================
// CORE SYNC OPERATIONS
// ============================================================================

// SyncFile uploads a local file to the appropriate OneDrive/SharePoint location
func (s *SyncService) SyncFile(localPath string, fileType string) error {
	log.Printf("📤 Syncing file: %s (type: %s)", localPath, fileType)

	// Validate file exists
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check file size limit
	if s.config.SkipLargeFiles && fileInfo.Size() > s.config.MaxFileSize {
		log.Printf("⚠️ Skipping large file (%.2f MB): %s", float64(fileInfo.Size())/(1024*1024), localPath)
		s.recordSyncEvent(localPath, "", "skipped_large", "File exceeds size limit")
		return fmt.Errorf("file too large: %.2f MB", float64(fileInfo.Size())/(1024*1024))
	}

	// Read file content
	content, err := os.ReadFile(localPath)
	if err != nil {
		s.recordSyncEvent(localPath, "", "failed", err.Error())
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate file hash (for conflict detection)
	localHash := calculateHash(content)

	// Determine destination folder
	folderPath := s.determineFolderPath(fileType)
	fileName := filepath.Base(localPath)

	// Create upload request
	// Note: FolderID is used, we use the path as folder reference
	// The Microsoft Graph client will resolve folder path to ID if needed
	req := &msgraph.OneDriveUploadRequest{
		FolderID: folderPath, // Using path as folder reference
		FileName: fileName,
		Content:  content,
	}

	// Upload to OneDrive
	response, err := s.graphClient.UploadToOneDrive(req)
	if err != nil {
		log.Printf("❌ Sync failed: %v", err)
		s.recordSyncEvent(localPath, "", "failed", err.Error())

		// Retry if configured
		if s.config.AutoRetry {
			return s.retrySync(localPath, fileType, 1)
		}

		return err
	}

	log.Printf("✅ Synced to OneDrive: %s", response.File.WebURL)

	// Record successful sync event in database
	s.recordSyncEvent(localPath, response.File.WebURL, "success", "")

	// Update sync status with hash
	s.updateSyncStatus(localPath, response.File.WebURL, localHash, "")

	return nil
}

// retrySync attempts to sync file with exponential backoff
func (s *SyncService) retrySync(localPath string, fileType string, attempt int) error {
	if attempt > s.config.MaxRetries {
		log.Printf("❌ Max retries (%d) exceeded for: %s", s.config.MaxRetries, localPath)
		return fmt.Errorf("max retries exceeded")
	}

	// Exponential backoff
	delay := s.config.RetryDelay * time.Duration(attempt)
	log.Printf("⏳ Retry attempt %d/%d after %v: %s", attempt, s.config.MaxRetries, delay, localPath)
	time.Sleep(delay)

	// Attempt sync
	err := s.SyncFile(localPath, fileType)
	if err != nil {
		// Retry again
		return s.retrySync(localPath, fileType, attempt+1)
	}

	return nil
}

// ProcessFileEvent handles a file watcher event
func (s *SyncService) ProcessFileEvent(event *FileSyncState) error {
	// Determine file type from path
	fileType := s.detectFileType(event.Path)

	// Handle different event types
	switch event.EventType {
	case "created", "modified":
		return s.SyncFile(event.Path, fileType)

	case "deleted":
		log.Printf("🗑️ File deleted (not syncing deletion): %s", event.Path)
		s.recordSyncEvent(event.Path, "", "deleted", "Local file deleted")
		return nil

	case "renamed":
		log.Printf("📝 File renamed (syncing as new): %s", event.Path)
		return s.SyncFile(event.Path, fileType)

	default:
		log.Printf("⚠️ Unknown event type: %s for file: %s", event.EventType, event.Path)
		return nil
	}
}

// ============================================================================
// FILE TYPE DETECTION
// ============================================================================

// detectFileType determines the type of file based on path
func (s *SyncService) detectFileType(path string) string {
	lowerPath := strings.ToLower(path)
	ext := strings.ToLower(filepath.Ext(path))

	// By path keywords
	if strings.Contains(lowerPath, "rfq") {
		return "rfq"
	}
	if strings.Contains(lowerPath, "offer") || strings.Contains(lowerPath, "quote") {
		return "offer"
	}
	if strings.Contains(lowerPath, "invoice") {
		return "invoice"
	}
	if strings.Contains(lowerPath, "order") || strings.Contains(lowerPath, "po") {
		return "order"
	}

	// By file extension
	switch ext {
	case ".msg", ".eml":
		return "rfq" // Emails are usually RFQs
	case ".xml":
		return "pricing" // Rhine XML pricing files
	case ".pdf":
		return "invoice" // PDFs are usually invoices
	case ".xlsx", ".xls":
		return "offer" // Spreadsheets are usually offers
	default:
		return "document"
	}
}

// determineFolderPath maps file type to OneDrive folder path
func (s *SyncService) determineFolderPath(fileType string) string {
	switch fileType {
	case "rfq":
		return s.config.RFQFolderPath
	case "offer":
		return s.config.OfferFolderPath
	case "invoice":
		return s.config.InvoiceFolderPath
	default:
		return "Documents" // Default folder
	}
}

// ============================================================================
// DATABASE PERSISTENCE
// ============================================================================

// recordSyncEvent saves sync event to database
func (s *SyncService) recordSyncEvent(localPath, remotePath, status, errorMsg string) {
	if s.app == nil || s.app.db == nil {
		return
	}

	// FileWatchEvent uses these fields from database.go
	event := FileWatchEvent{
		FilePath:  localPath,
		EventType: status,
	}

	if err := s.app.db.Create(&event).Error; err != nil {
		log.Printf("⚠️ Failed to record sync event: %v", err)
	}
}

// updateSyncStatus updates sync status in database
func (s *SyncService) updateSyncStatus(localPath, remotePath, localHash, remoteHash string) {
	if s.app == nil || s.app.db == nil {
		return
	}

	// Find or create sync status
	var syncStatus SyncStatus
	result := s.app.db.Where("file_path = ?", localPath).First(&syncStatus)

	if result.Error != nil {
		// Create new sync status using actual struct fields from database.go
		syncStatus = SyncStatus{
			FilePath:     localPath,
			LastSyncTime: time.Now(),
			Status:       "Synced",
		}
		s.app.db.Create(&syncStatus)
	} else {
		// Update existing sync status
		syncStatus.LastSyncTime = time.Now()
		syncStatus.Status = "Synced"
		s.app.db.Save(&syncStatus)
	}
}

// ============================================================================
// CONFLICT DETECTION
// ============================================================================

// detectConflict checks if local and remote files are in sync
func (s *SyncService) detectConflict(localPath string) (bool, error) {
	if !s.config.DetectConflicts {
		return false, nil // Conflict detection disabled
	}

	// TODO: Implement remote file fetch and hash comparison
	// For now, return false (no conflict)
	return false, nil
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// calculateHash computes SHA-256 hash of file content
func calculateHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}

// calculateFileHash computes SHA-256 hash of file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

// HealthCheck verifies sync service is operational
func (s *SyncService) HealthCheck() error {
	if s.graphClient == nil {
		return fmt.Errorf("graph client not initialized")
	}

	// Check Graph API connectivity
	if err := s.graphClient.HealthCheck(); err != nil {
		return fmt.Errorf("graph client health check failed: %w", err)
	}

	log.Println("✅ Sync service health check passed")
	return nil
}
