// ═══════════════════════════════════════════════════════════════════════════
// TYPES - Microsoft Graph Data Structures
//
// MISSION: Type definitions for SharePoint, Teams, OneDrive integration
//
// Built with CLARITY × PRODUCTION × ZERO CRUFT 🕉️⚡💎
// ═══════════════════════════════════════════════════════════════════════════

package microsoft_graph

import (
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// CONFIGURATION
// ═══════════════════════════════════════════════════════════════════════════

// GraphConfig holds Microsoft Graph API credentials and endpoints
type GraphConfig struct {
	// Authentication (Azure AD)
	TenantID     string // Azure AD tenant ID
	ClientID     string // App registration client ID
	ClientSecret string // App registration secret

	// SharePoint
	SharePointSiteID    string // Site ID (default upload location)
	SharePointLibraryID string // Document library ID

	// Teams
	TeamsChannelID string // Teams channel for notifications
	TeamsTeamID    string // Team ID

	// OneDrive
	OneDriveFolderID string // OneDrive folder for user uploads

	// Options
	NotifyOnComplete bool // Send Teams notification on completion
	ExportVedicDoc   bool // Export VedicDoc format to SharePoint
	ExportJSON       bool // Export JSON to SharePoint
	RetryAttempts    int  // Retry failed uploads (default: 3)
	TimeoutSeconds   int  // API timeout (default: 30)
}

// DefaultGraphConfig returns sensible defaults
func DefaultGraphConfig() *GraphConfig {
	return &GraphConfig{
		NotifyOnComplete: true,
		ExportVedicDoc:   true,
		ExportJSON:       true,
		RetryAttempts:    3,
		TimeoutSeconds:   30,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// SHAREPOINT STRUCTURES
// ═══════════════════════════════════════════════════════════════════════════

// SharePointFile represents a file in SharePoint document library
type SharePointFile struct {
	ID          string         // File ID
	Name        string         // File name
	Path        string         // Full path in library
	Size        int64          // File size in bytes
	Created     time.Time      // Creation timestamp
	Modified    time.Time      // Last modified timestamp
	WebURL      string         // Browser URL
	DownloadURL string         // Direct download URL
	Metadata    map[string]any // Custom metadata
}

// SharePointUploadRequest represents a file upload request
type SharePointUploadRequest struct {
	SiteID     string         // Target site ID
	LibraryID  string         // Target library ID
	FolderPath string         // Folder path (e.g., "/Processed/OCR")
	FileName   string         // File name
	Content    []byte         // File content
	Metadata   map[string]any // Custom metadata (e.g., three-regime signature)
}

// SharePointUploadResponse represents upload result
type SharePointUploadResponse struct {
	Success  bool
	File     *SharePointFile
	Error    string
	Duration time.Duration
}

// ═══════════════════════════════════════════════════════════════════════════
// TEAMS STRUCTURES
// ═══════════════════════════════════════════════════════════════════════════

// TeamsMessage represents a Teams channel message
type TeamsMessage struct {
	ChannelID   string            // Target channel ID
	TeamID      string            // Team ID
	Subject     string            // Message subject
	Body        string            // Message body (HTML or plain text)
	Attachments []TeamsAttachment // File attachments
	Metadata    map[string]any    // Additional metadata
}

// TeamsAttachment represents a file attachment in Teams
type TeamsAttachment struct {
	Name        string // File name
	ContentType string // MIME type
	ContentURL  string // URL to content (SharePoint link)
}

// TeamsNotification represents a pipeline completion notification
type TeamsNotification struct {
	Status        string    // "Success" or "Failed"
	FileName      string    // Processed file name
	Geometry      string    // Geometry used (S³, Banach, etc.)
	Signature     string    // Three-regime signature [R1%, R2%, R3%]
	Duration      string    // Processing time
	Quality       float64   // Output quality score
	SharePointURL string    // Link to SharePoint file
	Timestamp     time.Time // Completion timestamp
}

// TeamsMessageResponse represents Teams API response
type TeamsMessageResponse struct {
	Success   bool
	MessageID string
	Error     string
	Duration  time.Duration
}

// ═══════════════════════════════════════════════════════════════════════════
// ONEDRIVE STRUCTURES
// ═══════════════════════════════════════════════════════════════════════════

// OneDriveFile represents a file in OneDrive
type OneDriveFile struct {
	ID          string    // File ID
	Name        string    // File name
	Path        string    // Full path
	Size        int64     // File size in bytes
	Created     time.Time // Creation timestamp
	Modified    time.Time // Last modified timestamp
	WebURL      string    // Browser URL
	DownloadURL string    // Direct download URL
}

// OneDriveUploadRequest represents OneDrive upload request
type OneDriveUploadRequest struct {
	FolderID string // Target folder ID
	FileName string // File name
	Content  []byte // File content
}

// OneDriveUploadResponse represents OneDrive upload result
type OneDriveUploadResponse struct {
	Success  bool
	File     *OneDriveFile
	Error    string
	Duration time.Duration
}

// ═══════════════════════════════════════════════════════════════════════════
// CALENDAR STRUCTURES
// ═══════════════════════════════════════════════════════════════════════════

// CalendarEvent represents a meeting/event
type CalendarEvent struct {
	Subject   string    // Meeting subject
	Body      string    // Meeting body (HTML)
	Start     time.Time // Start time
	End       time.Time // End time
	Location  string    // Location name
	Attendees []string  // Email addresses of attendees
	IsOnline  bool      // Is Teams meeting?
}

// ═══════════════════════════════════════════════════════════════════════════
// BATCH OPERATIONS
// ═══════════════════════════════════════════════════════════════════════════

// BatchUploadRequest represents batch file upload
type BatchUploadRequest struct {
	SharePointUploads []SharePointUploadRequest // SharePoint files
	OneDriveUploads   []OneDriveUploadRequest   // OneDrive files
	Notification      *TeamsNotification        // Optional notification
}

// BatchUploadResponse represents batch upload result
type BatchUploadResponse struct {
	Success           bool
	SharePointResults []SharePointUploadResponse
	OneDriveResults   []OneDriveUploadResponse
	NotificationSent  bool
	TotalDuration     time.Duration
	Errors            []string
}
