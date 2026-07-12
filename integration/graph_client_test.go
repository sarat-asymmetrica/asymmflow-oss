// ═══════════════════════════════════════════════════════════════════════════
// GRAPH CLIENT TESTS - Microsoft Graph Integration Tests
//
// Tests both Mock and Production implementations
// Built with TESTABILITY × PRODUCTION × REUSE 🕉️⚡💎
// ═══════════════════════════════════════════════════════════════════════════

package integration

import (
	"context"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// MOCK CLIENT TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestMockGraphClient_UploadAuditReport(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	upload := &AuditReportUpload{
		CustomerID:   12345,
		CustomerName: "Test Company Ltd",
		ReportType:   "Payment Prediction",
		Content:      []byte("Test audit report content"),
		FileName:     "test_report.pdf",
		Metadata: map[string]any{
			"risk_grade": "B",
			"quality":    0.85,
		},
	}

	url, err := client.UploadAuditReport(ctx, upload)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if url == "" {
		t.Fatal("Expected non-empty URL")
	}

	if client.UploadCount != 1 {
		t.Errorf("Expected UploadCount=1, got %d", client.UploadCount)
	}

	if len(client.UploadedFiles) != 1 {
		t.Errorf("Expected 1 uploaded file, got %d", len(client.UploadedFiles))
	}
}

func TestMockGraphClient_DownloadFile(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	// Upload first
	upload := &AuditReportUpload{
		CustomerID:   12345,
		CustomerName: "Test Company",
		ReportType:   "Risk Assessment",
		Content:      []byte("Test content"),
	}

	url, err := client.UploadAuditReport(ctx, upload)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Extract file ID from URL (using url for file ID extraction)
	_ = url            // URL returned from upload
	fileID := "file_0" // Mock always uses this ID for first upload

	// Download
	content, err := client.DownloadFile(ctx, fileID)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("Expected non-empty content")
	}

	if client.DownloadCount != 1 {
		t.Errorf("Expected DownloadCount=1, got %d", client.DownloadCount)
	}
}

func TestMockGraphClient_SendAuditNotification(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	notification := &AuditNotification{
		CustomerID:    12345,
		CustomerName:  "Test Company",
		ReportType:    "Payment Prediction",
		RiskGrade:     "A",
		Quality:       0.95,
		SharePointURL: "https://test.sharepoint.com/file/123",
		Timestamp:     time.Now(),
	}

	err := client.SendAuditNotification(ctx, notification)
	if err != nil {
		t.Fatalf("Notification failed: %v", err)
	}

	if client.NotificationCount != 1 {
		t.Errorf("Expected NotificationCount=1, got %d", client.NotificationCount)
	}

	if len(client.SentMessages) != 1 {
		t.Errorf("Expected 1 sent message, got %d", len(client.SentMessages))
	}
}

func TestMockGraphClient_SendTeamsMessage(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	err := client.SendTeamsMessage(ctx, "channel_123", "Test message")
	if err != nil {
		t.Fatalf("SendTeamsMessage failed: %v", err)
	}

	if len(client.SentMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(client.SentMessages))
	}
}

func TestMockGraphClient_SendOutlookEmail(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	err := client.SendOutlookEmail(ctx, "user@example.com", "Test Subject", "Test Body")
	if err != nil {
		t.Fatalf("SendOutlookEmail failed: %v", err)
	}

	if len(client.SentEmails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(client.SentEmails))
	}
}

func TestMockGraphClient_CreateMeeting(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	meeting := &Meeting{
		Subject:   "Test Meeting",
		Start:     time.Now().Add(24 * time.Hour),
		End:       time.Now().Add(25 * time.Hour),
		Attendees: []string{"user1@example.com", "user2@example.com"},
		Body:      "Test meeting body",
		Location:  "Conference Room A",
		IsOnline:  true,
	}

	meetingID, err := client.CreateMeeting(ctx, meeting)
	if err != nil {
		t.Fatalf("CreateMeeting failed: %v", err)
	}

	if meetingID == "" {
		t.Fatal("Expected non-empty meeting ID")
	}

	if len(client.CreatedMeetings) != 1 {
		t.Errorf("Expected 1 meeting, got %d", len(client.CreatedMeetings))
	}
}

func TestMockGraphClient_UpdateMeeting(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	newSubject := "Updated Subject"
	updates := &MeetingUpdate{
		Subject: &newSubject,
	}

	err := client.UpdateMeeting(ctx, "meeting_123", updates)
	if err != nil {
		t.Fatalf("UpdateMeeting failed: %v", err)
	}
}

func TestMockGraphClient_RegisterWebhook(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	webhookID, err := client.RegisterWebhook(ctx, "SharePoint", "file.created", "https://example.com/webhook")
	if err != nil {
		t.Fatalf("RegisterWebhook failed: %v", err)
	}

	if webhookID == "" {
		t.Fatal("Expected non-empty webhook ID")
	}

	if len(client.RegisteredWebhooks) != 1 {
		t.Errorf("Expected 1 webhook, got %d", len(client.RegisteredWebhooks))
	}
}

func TestMockGraphClient_UnregisterWebhook(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	// Register first
	webhookID, _ := client.RegisterWebhook(ctx, "SharePoint", "file.created", "https://example.com/webhook")

	// Unregister
	err := client.UnregisterWebhook(ctx, webhookID)
	if err != nil {
		t.Fatalf("UnregisterWebhook failed: %v", err)
	}

	if len(client.RegisteredWebhooks) != 0 {
		t.Errorf("Expected 0 webhooks, got %d", len(client.RegisteredWebhooks))
	}
}

func TestMockGraphClient_ProcessWebhookEvent(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	rawEvent := []byte(`{"eventType": "file.created"}`)

	event, err := client.ProcessWebhookEvent(ctx, rawEvent)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}

	if event.EventType == "" {
		t.Fatal("Expected non-empty event type")
	}
}

func TestMockGraphClient_HealthCheck(t *testing.T) {
	client := NewMockGraphClient()

	err := client.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}

func TestMockGraphClient_SetFilePermissions(t *testing.T) {
	client := NewMockGraphClient()
	ctx := context.Background()

	permissions := []FilePermission{
		{UserEmail: "user1@example.com", Role: "read"},
		{UserEmail: "user2@example.com", Role: "write"},
	}

	err := client.SetFilePermissions(ctx, "file_123", permissions)
	if err != nil {
		t.Fatalf("SetFilePermissions failed: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// INTEGRATION TESTS (Require Real Credentials)
// ═══════════════════════════════════════════════════════════════════════════

func TestProductionGraphClient_Integration(t *testing.T) {
	// Skip unless integration tests are enabled
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would require real Azure AD credentials
	// Set environment variables:
	//   GRAPH_CLIENT_ID
	//   GRAPH_TENANT_ID
	//   GRAPH_CLIENT_SECRET
	//   GRAPH_SITE_ID
	//   etc.

	t.Skip("Integration test requires real Azure AD credentials")

	// Example integration test:
	// config := &GraphClientConfig{
	// 	ClientID:     os.Getenv("GRAPH_CLIENT_ID"),
	// 	TenantID:     os.Getenv("GRAPH_TENANT_ID"),
	// 	ClientSecret: os.Getenv("GRAPH_CLIENT_SECRET"),
	// 	SiteID:       os.Getenv("GRAPH_SITE_ID"),
	// 	// ... other config
	// }
	//
	// client, err := NewProductionGraphClient(config)
	// if err != nil {
	// 	t.Fatalf("Failed to create client: %v", err)
	// }
	//
	// ctx := context.Background()
	// err = client.HealthCheck(ctx)
	// if err != nil {
	// 	t.Fatalf("Health check failed: %v", err)
	// }
}

// ═══════════════════════════════════════════════════════════════════════════
// BENCHMARK TESTS
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkMockGraphClient_UploadAuditReport(b *testing.B) {
	client := NewMockGraphClient()
	ctx := context.Background()

	upload := &AuditReportUpload{
		CustomerID:   12345,
		CustomerName: "Benchmark Company",
		ReportType:   "Payment Prediction",
		Content:      make([]byte, 1024*1024), // 1 MB
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.UploadAuditReport(ctx, upload)
		if err != nil {
			b.Fatalf("Upload failed: %v", err)
		}
	}
}

func BenchmarkMockGraphClient_SendNotification(b *testing.B) {
	client := NewMockGraphClient()
	ctx := context.Background()

	notification := &AuditNotification{
		CustomerID:    12345,
		CustomerName:  "Benchmark Company",
		ReportType:    "Payment Prediction",
		RiskGrade:     "A",
		Quality:       0.95,
		SharePointURL: "https://test.sharepoint.com/file/123",
		Timestamp:     time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := client.SendAuditNotification(ctx, notification)
		if err != nil {
			b.Fatalf("Notification failed: %v", err)
		}
	}
}
