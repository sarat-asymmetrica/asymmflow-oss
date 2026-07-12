package main

import (
	"ph_holdings_app/integration"
)

// ButlerService exposes domain-specific Wails bindings by delegating to App.
type ButlerService struct {
	app *App
}

func NewButlerService(app *App) *ButlerService {
	return &ButlerService{app: app}
}

// --- app_prediction_dashboard.go ---

func (s *ButlerService) BatchPredict(customers []Customer) (BatchResult, error) {
	return s.app.BatchPredict(customers)
}

func (s *ButlerService) ClearHistory() error {
	return s.app.ClearHistory()
}

func (s *ButlerService) ExportCustomerTemplate() Customer {
	return s.app.ExportCustomerTemplate()
}

func (s *ButlerService) GetConfig() (map[string]any, error) {
	return s.app.GetConfig()
}

func (s *ButlerService) GetCustomerHistory(customerID string) ([]PredictionRecord, error) {
	return s.app.GetCustomerHistory(customerID)
}

func (s *ButlerService) GetDashboardEvents(limit int) ([]DashboardEvent, error) {
	return s.app.GetDashboardEvents(limit)
}

func (s *ButlerService) GetDashboardStats() (DashboardStats, error) {
	return s.app.GetDashboardStats()
}

func (s *ButlerService) GetHistory(limit int) ([]PredictionRecord, error) {
	return s.app.GetHistory(limit)
}

func (s *ButlerService) GetMonthlyRevenueByCustomer() ([]CustomerRevenueData, error) {
	return s.app.GetMonthlyRevenueByCustomer()
}

func (s *ButlerService) GetStatistics() (Statistics, error) {
	return s.app.GetStatistics()
}

func (s *ButlerService) GetToolInstallInstructions() string {
	return s.app.GetToolInstallInstructions()
}

func (s *ButlerService) GetToolsStatus() *integration.ToolsReport {
	return s.app.GetToolsStatus()
}

func (s *ButlerService) Greet(name string) string {
	return s.app.Greet(name)
}

func (s *ButlerService) PredictPayment(customer Customer) (PaymentPrediction, error) {
	return s.app.PredictPayment(customer)
}

func (s *ButlerService) RefreshToolsStatus() *integration.ToolsReport {
	return s.app.RefreshToolsStatus()
}

func (s *ButlerService) ValidateCustomer(customer Customer) ValidationResult {
	return s.app.ValidateCustomer(customer)
}

// --- batch_operations.go ---

func (s *ButlerService) BatchCreateInvoiceItems(items []DBInvoiceItem) error {
	return s.app.BatchCreateInvoiceItems(items)
}

func (s *ButlerService) BatchUpdateCustomerGrade(customerIDs []string, grade string) error {
	return s.app.BatchUpdateCustomerGrade(customerIDs, grade)
}

func (s *ButlerService) BatchUpdateOrderStatus(orderIDs []string, status string) error {
	return s.app.BatchUpdateOrderStatus(orderIDs, status)
}

// --- butler_ai.go ---

func (s *ButlerService) AnalyzeDocumentWithButler(text string, docType string, metadata map[string]any) (ButlerOCRInsight, error) {
	return s.app.AnalyzeDocumentWithButler(text, docType, metadata)
}

func (s *ButlerService) ChatWithButler(message string) (ButlerResponse, error) {
	return s.app.ChatWithButler(message)
}

func (s *ButlerService) TestMistralConnection() (bool, error) {
	return s.app.TestMistralConnection()
}

// --- butler_reports.go ---

func (s *ButlerService) GenerateButlerReport(reportType string, query string) (string, error) {
	return s.app.GenerateButlerReport(reportType, query)
}

func (s *ButlerService) GetCashflowEvidenceAgentBrief(days int, maxChars int) (string, error) {
	return s.app.GetCashflowEvidenceAgentBrief(days, maxChars)
}

// --- chat_service.go ---

func (s *ButlerService) ChatWithButlerPersistent(conversationID, message string) (ChatResponse, error) {
	return s.app.ChatWithButlerPersistent(conversationID, message)
}

func (s *ButlerService) CreateConversation(title string) (*Conversation, error) {
	return s.app.CreateConversation(title)
}

func (s *ButlerService) DeleteConversation(conversationID string) error {
	return s.app.DeleteConversation(conversationID)
}

func (s *ButlerService) GenerateDailyBriefing(conversationID string) (ChatResponse, error) {
	return s.app.GenerateDailyBriefing(conversationID)
}

func (s *ButlerService) GetButlerDailyBriefing(conversationID string) (ChatResponse, error) {
	return s.app.GetButlerDailyBriefing(conversationID)
}

func (s *ButlerService) GetConversationMessages(conversationID string) ([]ChatMessage, error) {
	return s.app.GetConversationMessages(conversationID)
}

func (s *ButlerService) ListConversations() ([]Conversation, error) {
	return s.app.ListConversations()
}

func (s *ButlerService) PurgeAllConversations() error {
	return s.app.PurgeAllConversations()
}

// --- survival_intelligence.go ---

func (s *ButlerService) AcknowledgeAlert(alertID int) error {
	return s.app.AcknowledgeAlert(alertID)
}

func (s *ButlerService) DismissAlert(alertID int) error {
	return s.app.DismissAlert(alertID)
}

func (s *ButlerService) GetAlertSummary() (AlertSummary, error) {
	return s.app.GetAlertSummary()
}

func (s *ButlerService) GetSurvivalMetrics() (SurvivalMetrics, error) {
	return s.app.GetSurvivalMetrics()
}
