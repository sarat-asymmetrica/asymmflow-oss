// ═══════════════════════════════════════════════════════════════════════════
// CLIENT - Microsoft Graph API Client Interface
//
// MISSION: Mock-able Graph client for SharePoint, Teams, OneDrive
//
// DESIGN:
//   - Interface-based (easy mocking for tests)
//   - Retry logic built-in
//   - Production logging
//   - Error handling
//
// Built with INTERFACE × PRODUCTION × TESTABILITY 🕉️⚡💎
// ═══════════════════════════════════════════════════════════════════════════

package microsoft_graph

import (
	"fmt"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// INTERFACE (Mock-able!)
// ═══════════════════════════════════════════════════════════════════════════

// GraphClient is the main interface for Microsoft Graph operations
// Implementations:
//   - ProductionGraphClient: Real Azure AD auth + API calls
//   - MockGraphClient: In-memory mock for testing (below)
type GraphClient interface {
	// SharePoint operations
	UploadToSharePoint(req *SharePointUploadRequest) (*SharePointUploadResponse, error)
	GetSharePointFile(siteID, fileID string) (*SharePointFile, error)

	// Teams operations
	SendTeamsMessage(msg *TeamsMessage) (*TeamsMessageResponse, error)
	SendNotification(notification *TeamsNotification) (*TeamsMessageResponse, error)

	// OneDrive operations
	UploadToOneDrive(req *OneDriveUploadRequest) (*OneDriveUploadResponse, error)
	GetOneDriveFile(fileID string) (*OneDriveFile, error)

	// Email operations
	SendEmail(to, subject, htmlBody string) error
	SendEmailWithAttachment(to, subject, htmlBody, fileName string, fileContent []byte) error
	CreateEmailDraft(to, subject, htmlBody string) (string, error)

	// Calendar operations
	CreateCalendarEvent(event *CalendarEvent) (string, error)

	// Batch operations
	BatchUpload(req *BatchUploadRequest) (*BatchUploadResponse, error)

	// Health check
	HealthCheck() error
}

// ═══════════════════════════════════════════════════════════════════════════
// MOCK IMPLEMENTATION (For Testing!)
// ═══════════════════════════════════════════════════════════════════════════

// MockGraphClient implements GraphClient for testing
// No real API calls - everything stored in-memory
type MockGraphClient struct {
	Config *GraphConfig

	// In-memory storage
	SharePointFiles map[string]*SharePointFile // fileID -> file
	OneDriveFiles   map[string]*OneDriveFile   // fileID -> file
	TeamMessages    []*TeamsMessage            // All sent messages
	EmailsSent      []map[string]any           // All sent emails
	EmailDrafts     []map[string]any           // All email drafts
	CalendarEvents  []*CalendarEvent           // All calendar events

	// Counters for assertions
	SharePointUploadCount int
	OneDriveUploadCount   int
	TeamsMessageCount     int
	NotificationCount     int
	EmailCount            int
	DraftCount            int
	EventCount            int

	// Simulate failures (for testing error handling)
	SimulateSharePointError bool
	SimulateTeamsError      bool
	SimulateOneDriveError   bool
	SimulateEmailError      bool
	SimulateCalendarError   bool
}

// NewMockGraphClient creates a new mock client
func NewMockGraphClient(config *GraphConfig) *MockGraphClient {
	if config == nil {
		config = DefaultGraphConfig()
	}

	return &MockGraphClient{
		Config:          config,
		SharePointFiles: make(map[string]*SharePointFile),
		OneDriveFiles:   make(map[string]*OneDriveFile),
		TeamMessages:    make([]*TeamsMessage, 0),
		EmailsSent:      make([]map[string]any, 0),
		EmailDrafts:     make([]map[string]any, 0),
		CalendarEvents:  make([]*CalendarEvent, 0),
	}
}

// UploadToSharePoint mock implementation
func (m *MockGraphClient) UploadToSharePoint(req *SharePointUploadRequest) (*SharePointUploadResponse, error) {
	start := time.Now()

	if m.SimulateSharePointError {
		return &SharePointUploadResponse{
			Success:  false,
			Error:    "Simulated SharePoint upload error",
			Duration: time.Since(start),
		}, fmt.Errorf("simulated SharePoint error")
	}

	// Generate mock file ID
	fileID := fmt.Sprintf("sp_file_%d", m.SharePointUploadCount)

	file := &SharePointFile{
		ID:          fileID,
		Name:        req.FileName,
		Path:        fmt.Sprintf("%s/%s", req.FolderPath, req.FileName),
		Size:        int64(len(req.Content)),
		Created:     time.Now(),
		Modified:    time.Now(),
		WebURL:      fmt.Sprintf("https://mock-sharepoint.com/sites/%s/file/%s", req.SiteID, fileID),
		DownloadURL: fmt.Sprintf("https://mock-sharepoint.com/download/%s", fileID),
		Metadata:    req.Metadata,
	}

	m.SharePointFiles[fileID] = file
	m.SharePointUploadCount++

	return &SharePointUploadResponse{
		Success:  true,
		File:     file,
		Duration: time.Since(start),
	}, nil
}

// GetSharePointFile mock implementation
func (m *MockGraphClient) GetSharePointFile(siteID, fileID string) (*SharePointFile, error) {
	file, exists := m.SharePointFiles[fileID]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}
	return file, nil
}

// SendTeamsMessage mock implementation
func (m *MockGraphClient) SendTeamsMessage(msg *TeamsMessage) (*TeamsMessageResponse, error) {
	start := time.Now()

	if m.SimulateTeamsError {
		return &TeamsMessageResponse{
			Success:  false,
			Error:    "Simulated Teams message error",
			Duration: time.Since(start),
		}, fmt.Errorf("simulated Teams error")
	}

	m.TeamMessages = append(m.TeamMessages, msg)
	m.TeamsMessageCount++

	messageID := fmt.Sprintf("teams_msg_%d", m.TeamsMessageCount)

	return &TeamsMessageResponse{
		Success:   true,
		MessageID: messageID,
		Duration:  time.Since(start),
	}, nil
}

// SendNotification mock implementation
func (m *MockGraphClient) SendNotification(notification *TeamsNotification) (*TeamsMessageResponse, error) {
	// Convert notification to Teams message
	body := formatNotificationBody(notification)

	msg := &TeamsMessage{
		ChannelID: m.Config.TeamsChannelID,
		TeamID:    m.Config.TeamsTeamID,
		Subject:   fmt.Sprintf("Pipeline Complete: %s", notification.FileName),
		Body:      body,
	}

	m.NotificationCount++

	return m.SendTeamsMessage(msg)
}

// UploadToOneDrive mock implementation
func (m *MockGraphClient) UploadToOneDrive(req *OneDriveUploadRequest) (*OneDriveUploadResponse, error) {
	start := time.Now()

	if m.SimulateOneDriveError {
		return &OneDriveUploadResponse{
			Success:  false,
			Error:    "Simulated OneDrive upload error",
			Duration: time.Since(start),
		}, fmt.Errorf("simulated OneDrive error")
	}

	fileID := fmt.Sprintf("od_file_%d", m.OneDriveUploadCount)

	file := &OneDriveFile{
		ID:          fileID,
		Name:        req.FileName,
		Path:        fmt.Sprintf("/drive/items/%s/%s", req.FolderID, req.FileName),
		Size:        int64(len(req.Content)),
		Created:     time.Now(),
		Modified:    time.Now(),
		WebURL:      fmt.Sprintf("https://mock-onedrive.com/file/%s", fileID),
		DownloadURL: fmt.Sprintf("https://mock-onedrive.com/download/%s", fileID),
	}

	m.OneDriveFiles[fileID] = file
	m.OneDriveUploadCount++

	return &OneDriveUploadResponse{
		Success:  true,
		File:     file,
		Duration: time.Since(start),
	}, nil
}

// GetOneDriveFile mock implementation
func (m *MockGraphClient) GetOneDriveFile(fileID string) (*OneDriveFile, error) {
	file, exists := m.OneDriveFiles[fileID]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}
	return file, nil
}

// BatchUpload mock implementation
func (m *MockGraphClient) BatchUpload(req *BatchUploadRequest) (*BatchUploadResponse, error) {
	start := time.Now()

	response := &BatchUploadResponse{
		Success:           true,
		SharePointResults: make([]SharePointUploadResponse, 0),
		OneDriveResults:   make([]OneDriveUploadResponse, 0),
		Errors:            make([]string, 0),
	}

	// Upload SharePoint files
	for _, spReq := range req.SharePointUploads {
		spRes, err := m.UploadToSharePoint(&spReq)
		if err != nil {
			response.Success = false
			response.Errors = append(response.Errors, err.Error())
		}
		response.SharePointResults = append(response.SharePointResults, *spRes)
	}

	// Upload OneDrive files
	for _, odReq := range req.OneDriveUploads {
		odRes, err := m.UploadToOneDrive(&odReq)
		if err != nil {
			response.Success = false
			response.Errors = append(response.Errors, err.Error())
		}
		response.OneDriveResults = append(response.OneDriveResults, *odRes)
	}

	// Send notification if requested
	if req.Notification != nil && m.Config.NotifyOnComplete {
		_, err := m.SendNotification(req.Notification)
		if err != nil {
			response.Errors = append(response.Errors, err.Error())
		} else {
			response.NotificationSent = true
		}
	}

	response.TotalDuration = time.Since(start)

	return response, nil
}

// HealthCheck mock implementation
func (m *MockGraphClient) HealthCheck() error {
	// Mock always healthy unless errors simulated
	if m.SimulateSharePointError || m.SimulateTeamsError || m.SimulateOneDriveError || m.SimulateEmailError || m.SimulateCalendarError {
		return fmt.Errorf("mock client has simulated errors enabled")
	}
	return nil
}

// SendEmail mock implementation
func (m *MockGraphClient) SendEmail(to, subject, htmlBody string) error {
	if m.SimulateEmailError {
		return fmt.Errorf("simulated email error")
	}

	email := map[string]any{
		"to":      to,
		"subject": subject,
		"body":    htmlBody,
		"type":    "sent",
	}

	m.EmailsSent = append(m.EmailsSent, email)
	m.EmailCount++

	return nil
}

// SendEmailWithAttachment mock implementation
func (m *MockGraphClient) SendEmailWithAttachment(to, subject, htmlBody, fileName string, fileContent []byte) error {
	if m.SimulateEmailError {
		return fmt.Errorf("simulated email error")
	}

	email := map[string]any{
		"to":             to,
		"subject":        subject,
		"body":           htmlBody,
		"attachmentName": fileName,
		"attachmentSize": len(fileContent),
		"type":           "sent_with_attachment",
	}

	m.EmailsSent = append(m.EmailsSent, email)
	m.EmailCount++

	return nil
}

// CreateEmailDraft mock implementation
func (m *MockGraphClient) CreateEmailDraft(to, subject, htmlBody string) (string, error) {
	if m.SimulateEmailError {
		return "", fmt.Errorf("simulated email error")
	}

	draftID := fmt.Sprintf("draft_%d", m.DraftCount)

	draft := map[string]any{
		"id":      draftID,
		"to":      to,
		"subject": subject,
		"body":    htmlBody,
		"type":    "draft",
	}

	m.EmailDrafts = append(m.EmailDrafts, draft)
	m.DraftCount++

	return draftID, nil
}

// CreateCalendarEvent mock implementation
func (m *MockGraphClient) CreateCalendarEvent(event *CalendarEvent) (string, error) {
	if m.SimulateCalendarError {
		return "", fmt.Errorf("simulated calendar error")
	}

	eventID := fmt.Sprintf("event_%d", m.EventCount)

	// Store a copy of the event
	eventCopy := &CalendarEvent{
		Subject:   event.Subject,
		Body:      event.Body,
		Start:     event.Start,
		End:       event.End,
		Location:  event.Location,
		Attendees: make([]string, len(event.Attendees)),
		IsOnline:  event.IsOnline,
	}
	copy(eventCopy.Attendees, event.Attendees)

	m.CalendarEvents = append(m.CalendarEvents, eventCopy)
	m.EventCount++

	return eventID, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// HELPER FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

// formatNotificationBody creates HTML body for Teams notification
func formatNotificationBody(n *TeamsNotification) string {
	statusEmoji := "✅"
	if n.Status != "Success" {
		statusEmoji = "❌"
	}

	return fmt.Sprintf(`
<h2>%s Pipeline Processing Complete</h2>
<ul>
<li><strong>File:</strong> %s</li>
<li><strong>Status:</strong> %s</li>
<li><strong>Geometry:</strong> %s</li>
<li><strong>Signature:</strong> %s</li>
<li><strong>Duration:</strong> %s</li>
<li><strong>Quality:</strong> %.1f%%</li>
<li><strong>SharePoint:</strong> <a href="%s">View Document</a></li>
<li><strong>Timestamp:</strong> %s</li>
</ul>
`,
		statusEmoji,
		n.FileName,
		n.Status,
		n.Geometry,
		n.Signature,
		n.Duration,
		n.Quality*100,
		n.SharePointURL,
		n.Timestamp.Format("2006-01-02 15:04:05"),
	)
}
