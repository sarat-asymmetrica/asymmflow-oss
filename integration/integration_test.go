// ═══════════════════════════════════════════════════════════════════════════
// INTEGRATION TESTS - E2E Microsoft Ecosystem Integration
//
// MISSION: Validate all integration components work together
//   - Graph Client (SharePoint, Teams, OneDrive)
//   - COM Automation (Excel, Word, Outlook)
//   - Copilot Bridge (NL → Pipeline)
//   - Tool Executor (9 external tools)
//
// Built with COMPREHENSIVE × PRODUCTION × ZERO_FLAKINESS 🕉️⚡💎
// ═══════════════════════════════════════════════════════════════════════════

package integration

import (
	"context"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// GRAPH CLIENT TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestGraphClient_UploadAuditReport(t *testing.T) {
	client := NewMockGraphClient()

	upload := &AuditReportUpload{
		CustomerID:   1234,
		CustomerName: "Test Customer",
		ReportType:   "Payment Prediction",
		Content:      []byte("Mock audit report content"),
		FileName:     "audit_report.pdf",
	}

	ctx := context.Background()
	url, err := client.UploadAuditReport(ctx, upload)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if url == "" {
		t.Fatal("Expected non-empty URL")
	}

	if client.UploadCount != 1 {
		t.Errorf("Expected 1 upload, got %d", client.UploadCount)
	}
}

func TestGraphClient_SendAuditNotification(t *testing.T) {
	client := NewMockGraphClient()

	notification := &AuditNotification{
		CustomerID:    1234,
		CustomerName:  "Test Customer",
		ReportType:    "Risk Assessment",
		RiskGrade:     "B",
		Quality:       0.87,
		SharePointURL: "https://mock.sharepoint.com/file/123",
		Timestamp:     time.Now(),
	}

	ctx := context.Background()
	err := client.SendAuditNotification(ctx, notification)
	if err != nil {
		t.Fatalf("Notification failed: %v", err)
	}

	if client.NotificationCount != 1 {
		t.Errorf("Expected 1 notification, got %d", client.NotificationCount)
	}

	if len(client.SentMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(client.SentMessages))
	}
}

func TestGraphClient_FilePermissions(t *testing.T) {
	client := NewMockGraphClient()

	permissions := []FilePermission{
		{UserEmail: "user1@example.com", Role: "read"},
		{UserEmail: "user2@example.com", Role: "write"},
	}

	ctx := context.Background()
	err := client.SetFilePermissions(ctx, "file_123", permissions)
	if err != nil {
		t.Fatalf("Set permissions failed: %v", err)
	}
}

func TestGraphClient_CreateMeeting(t *testing.T) {
	client := NewMockGraphClient()

	meeting := &Meeting{
		Subject:   "Audit Review",
		Start:     time.Now().Add(24 * time.Hour),
		End:       time.Now().Add(25 * time.Hour),
		Attendees: []string{"user1@example.com", "user2@example.com"},
		Body:      "Review audit findings",
		Location:  "Conference Room A",
		IsOnline:  true,
	}

	ctx := context.Background()
	meetingID, err := client.CreateMeeting(ctx, meeting)
	if err != nil {
		t.Fatalf("Create meeting failed: %v", err)
	}

	if meetingID == "" {
		t.Fatal("Expected non-empty meeting ID")
	}

	if len(client.CreatedMeetings) != 1 {
		t.Errorf("Expected 1 meeting, got %d", len(client.CreatedMeetings))
	}
}

func TestGraphClient_Webhooks(t *testing.T) {
	client := NewMockGraphClient()

	ctx := context.Background()

	// Register webhook
	webhookID, err := client.RegisterWebhook(ctx, "SharePoint", "file.created", "https://example.com/webhook")
	if err != nil {
		t.Fatalf("Register webhook failed: %v", err)
	}

	if webhookID == "" {
		t.Fatal("Expected non-empty webhook ID")
	}

	if len(client.RegisteredWebhooks) != 1 {
		t.Errorf("Expected 1 webhook, got %d", len(client.RegisteredWebhooks))
	}

	// Unregister webhook
	err = client.UnregisterWebhook(ctx, webhookID)
	if err != nil {
		t.Fatalf("Unregister webhook failed: %v", err)
	}

	if len(client.RegisteredWebhooks) != 0 {
		t.Errorf("Expected 0 webhooks after unregister, got %d", len(client.RegisteredWebhooks))
	}
}

func TestGraphClient_HealthCheck(t *testing.T) {
	client := NewMockGraphClient()

	ctx := context.Background()
	err := client.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// COPILOT BRIDGE TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestCopilotBridge_ParseIntent_PaymentPrediction(t *testing.T) {
	bridge := NewProductionCopilotBridge()

	intent, err := bridge.ParseIntent("Predict when customer #1234 will pay invoice BHD 5,000")
	if err != nil {
		t.Fatalf("Parse intent failed: %v", err)
	}

	if intent.Action != "predict" {
		t.Errorf("Expected action 'predict', got '%s'", intent.Action)
	}

	if intent.Target != "payment" {
		t.Errorf("Expected target 'payment', got '%s'", intent.Target)
	}

	if intent.Parameters["customer_id"] != "1234" {
		t.Errorf("Expected customer_id '1234', got '%s'", intent.Parameters["customer_id"])
	}

	if intent.Parameters["amount"] != "5,000" {
		t.Errorf("Expected amount '5,000', got '%s'", intent.Parameters["amount"])
	}
}

func TestCopilotBridge_ParseIntent_RiskAnalysis(t *testing.T) {
	bridge := NewProductionCopilotBridge()

	intent, err := bridge.ParseIntent("Analyze risk for customer #9999")
	if err != nil {
		t.Fatalf("Parse intent failed: %v", err)
	}

	if intent.Action != "analyze" {
		t.Errorf("Expected action 'analyze', got '%s'", intent.Action)
	}

	if intent.Target != "risk" {
		t.Errorf("Expected target 'risk', got '%s'", intent.Target)
	}

	if intent.Parameters["customer_id"] != "9999" {
		t.Errorf("Expected customer_id '9999', got '%s'", intent.Parameters["customer_id"])
	}
}

func TestCopilotBridge_SelectGeometry_PaymentPrediction(t *testing.T) {
	bridge := NewProductionCopilotBridge()

	intent := &Intent{
		Action: "predict",
		Target: "payment",
	}

	geometry, err := bridge.SelectGeometry(intent)
	if err != nil {
		t.Fatalf("Select geometry failed: %v", err)
	}

	if geometry != "quaternion_s3" {
		t.Errorf("Expected quaternion_s3, got %s", geometry)
	}
}

func TestCopilotBridge_SelectGeometry_OfferAnalysis(t *testing.T) {
	bridge := NewProductionCopilotBridge()

	intent := &Intent{
		Action: "analyze",
		Target: "offer",
	}

	geometry, err := bridge.SelectGeometry(intent)
	if err != nil {
		t.Fatalf("Select geometry failed: %v", err)
	}

	if geometry != "banach" {
		t.Errorf("Expected banach, got %s", geometry)
	}
}

func TestCopilotBridge_ProcessIntent_E2E(t *testing.T) {
	bridge := NewProductionCopilotBridge()

	req := &CopilotRequest{
		Text:      "Predict payment for customer #1234",
		UserID:    "user@example.com",
		SessionID: "session_123",
	}

	ctx := context.Background()
	response, err := bridge.ProcessIntent(ctx, req)
	if err != nil {
		t.Fatalf("Process intent failed: %v", err)
	}

	if response.Intent == nil {
		t.Fatal("Expected non-nil intent")
	}

	if response.GeometryUsed == "" {
		t.Fatal("Expected non-empty geometry")
	}

	if response.ExecutionTime == 0 {
		t.Error("Expected non-zero execution time")
	}

	if len(response.Suggestions) == 0 {
		t.Error("Expected suggestions")
	}
}

func TestCopilotBridge_FormatForExcel(t *testing.T) {
	bridge := NewProductionCopilotBridge()

	results := map[string]any{
		"quality": 0.87,
	}

	data, err := bridge.FormatForExcel(results)
	if err != nil {
		t.Fatalf("Format for Excel failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Expected non-empty data")
	}

	// Check header row
	if len(data[0]) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(data[0]))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TOOL EXECUTOR TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestToolExecutor_CheckToolAvailability(t *testing.T) {
	executor := NewMockToolExecutor()

	tools, err := executor.CheckToolAvailability()
	if err != nil {
		t.Fatalf("Check availability failed: %v", err)
	}

	expectedTools := []string{"pandoc", "tesseract", "ghostscript", "imagemagick", "ffmpeg", "jq", "sqlite", "graphviz", "curl"}
	for _, name := range expectedTools {
		if _, exists := tools[name]; !exists {
			t.Errorf("Expected tool %s to be detected", name)
		}
	}
}

func TestToolExecutor_IsToolAvailable(t *testing.T) {
	executor := NewMockToolExecutor()

	if !executor.IsToolAvailable("pandoc") {
		t.Error("Expected pandoc to be available")
	}

	if executor.IsToolAvailable("nonexistent_tool") {
		t.Error("Expected nonexistent_tool to be unavailable")
	}
}

func TestToolExecutor_ConvertDocument(t *testing.T) {
	executor := NewMockToolExecutor()

	ctx := context.Background()
	err := executor.ConvertDocument(ctx, "input.md", "output.pdf", "pdf")
	if err != nil {
		t.Fatalf("Convert document failed: %v", err)
	}

	if executor.ConversionsRun != 1 {
		t.Errorf("Expected 1 conversion, got %d", executor.ConversionsRun)
	}
}

func TestToolExecutor_ExtractTextFromImage(t *testing.T) {
	executor := NewMockToolExecutor()

	ctx := context.Background()
	text, err := executor.ExtractTextFromImage(ctx, "invoice.png")
	if err != nil {
		t.Fatalf("OCR failed: %v", err)
	}

	if text == "" {
		t.Fatal("Expected non-empty text")
	}

	if executor.OCRsPerformed != 1 {
		t.Errorf("Expected 1 OCR, got %d", executor.OCRsPerformed)
	}
}

func TestToolExecutor_GenerateDiagram(t *testing.T) {
	executor := NewMockToolExecutor()

	dotContent := `
digraph G {
    A -> B;
    B -> C;
}
`

	ctx := context.Background()
	err := executor.GenerateDiagram(ctx, dotContent, "diagram.png", "png")
	if err != nil {
		t.Fatalf("Generate diagram failed: %v", err)
	}

	if executor.DiagramsCreated != 1 {
		t.Errorf("Expected 1 diagram, got %d", executor.DiagramsCreated)
	}
}

func TestToolExecutor_QueryJSON(t *testing.T) {
	executor := NewMockToolExecutor()

	ctx := context.Background()
	result, err := executor.QueryJSON(ctx, "data.json", ".results")
	if err != nil {
		t.Fatalf("JSON query failed: %v", err)
	}

	if result == "" {
		t.Fatal("Expected non-empty result")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// INTEGRATION SCENARIOS (E2E)
// ═══════════════════════════════════════════════════════════════════════════

func TestIntegration_CompleteAuditFlow(t *testing.T) {
	// Complete flow: Intent → Pipeline → SharePoint → Teams
	ctx := context.Background()

	// 1. Parse intent
	bridge := NewProductionCopilotBridge()
	intent, err := bridge.ParseIntent("Generate audit report for customer #1234")
	if err != nil {
		t.Fatalf("Parse intent failed: %v", err)
	}

	// 2. Select geometry
	geometry, err := bridge.SelectGeometry(intent)
	if err != nil {
		t.Fatalf("Select geometry failed: %v", err)
	}

	// 3. Execute pipeline (mock)
	results, err := bridge.ExecutePipeline(ctx, intent, geometry)
	if err != nil {
		t.Fatalf("Execute pipeline failed: %v", err)
	}

	// 4. Upload to SharePoint
	graphClient := NewMockGraphClient()
	upload := &AuditReportUpload{
		CustomerID:   1234,
		CustomerName: "Test Customer",
		ReportType:   "Audit Report",
		Content:      []byte("Mock report content"),
	}

	sharePointURL, err := graphClient.UploadAuditReport(ctx, upload)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// 5. Send Teams notification
	notification := &AuditNotification{
		CustomerID:    1234,
		CustomerName:  "Test Customer",
		ReportType:    "Audit Report",
		RiskGrade:     "B",
		Quality:       0.87,
		SharePointURL: sharePointURL,
		Timestamp:     time.Now(),
	}

	err = graphClient.SendAuditNotification(ctx, notification)
	if err != nil {
		t.Fatalf("Notification failed: %v", err)
	}

	// Verify complete flow
	if results == nil {
		t.Fatal("Expected non-nil results")
	}
	if sharePointURL == "" {
		t.Fatal("Expected non-empty SharePoint URL")
	}
	if graphClient.NotificationCount != 1 {
		t.Errorf("Expected 1 notification, got %d", graphClient.NotificationCount)
	}
}

func TestIntegration_ExcelAutomation(t *testing.T) {
	// Flow: NL → Pipeline → Excel → Teams
	ctx := context.Background()

	// 1. Parse intent
	bridge := NewProductionCopilotBridge()
	req := &CopilotRequest{
		Text: "Show me payment predictions for all customers",
	}

	response, err := bridge.ProcessIntent(ctx, req)
	if err != nil {
		t.Fatalf("Process intent failed: %v", err)
	}

	// 2. Format for Excel
	excelData, err := bridge.FormatForExcel(response.DetailedResults)
	if err != nil {
		t.Fatalf("Format for Excel failed: %v", err)
	}

	if len(excelData) == 0 {
		t.Fatal("Expected non-empty Excel data")
	}

	// 3. (Would insert into Excel via COM automation - tested separately)
	t.Logf("Excel data: %d rows", len(excelData))
}

func TestIntegration_ToolChain(t *testing.T) {
	// Tool chain: Image → OCR → JSON → SQL → Diagram
	ctx := context.Background()
	executor := NewMockToolExecutor()

	// 1. OCR invoice image
	text, err := executor.ExtractTextFromImage(ctx, "invoice.png")
	if err != nil {
		t.Fatalf("OCR failed: %v", err)
	}

	// 2. Query JSON (extract structured data)
	jsonResult, err := executor.QueryJSON(ctx, "data.json", ".invoices")
	if err != nil {
		t.Fatalf("JSON query failed: %v", err)
	}

	// 3. SQL query (aggregate)
	sqlResult, err := executor.ExecuteSQLQuery(ctx, "ph_holdings.db", "SELECT COUNT(*) FROM customers")
	if err != nil {
		t.Fatalf("SQL query failed: %v", err)
	}

	// 4. Generate diagram
	dotContent := `digraph { A -> B; }`
	err = executor.GenerateDiagram(ctx, dotContent, "flow.png", "png")
	if err != nil {
		t.Fatalf("Diagram generation failed: %v", err)
	}

	// Verify chain
	if text == "" || jsonResult == "" || sqlResult == "" {
		t.Fatal("Expected non-empty results from tool chain")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// BENCHMARKS
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkGraphClient_UploadAuditReport(b *testing.B) {
	client := NewMockGraphClient()
	ctx := context.Background()

	upload := &AuditReportUpload{
		CustomerID:   1234,
		CustomerName: "Benchmark Customer",
		ReportType:   "Payment Prediction",
		Content:      make([]byte, 1024*1024), // 1 MB
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.UploadAuditReport(ctx, upload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopilotBridge_ParseIntent(b *testing.B) {
	bridge := NewProductionCopilotBridge()
	text := "Predict when customer #1234 will pay invoice BHD 5,000"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bridge.ParseIntent(text)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkToolExecutor_ExtractTextFromImage(b *testing.B) {
	executor := NewMockToolExecutor()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExtractTextFromImage(ctx, "invoice.png")
		if err != nil {
			b.Fatal(err)
		}
	}
}
