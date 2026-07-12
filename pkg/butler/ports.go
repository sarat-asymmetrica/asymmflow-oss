// Package butler defines the Butler intelligence domain ports.
package butler

import (
	"context"
	"time"
)

type DatabasePort interface {
	QueryInvoices(filter map[string]any) ([]map[string]any, error)
	QueryCustomers(filter map[string]any) ([]map[string]any, error)
	QueryOrders(filter map[string]any) ([]map[string]any, error)
	QueryPayments(filter map[string]any) ([]map[string]any, error)
	QueryOffers(filter map[string]any) ([]map[string]any, error)
	QueryTable(table string, filter map[string]any, limit int) ([]map[string]any, error)
	RawQuery(sql string, args ...any) ([]map[string]any, error)
	Count(table string, filter map[string]any) (int64, error)
}

type UserContextPort interface {
	CurrentUserID() string
	CurrentUserName() string
	CurrentUserRole() string
	CurrentLicenseRole() string
	CurrentDivision() string
	HasPermission(action string) bool
}

type LLMPort interface {
	ChatCompletion(systemPrompt, userMessage string, maxTokens int) (string, error)
	ChatCompletionWithHistory(messages []ChatMessage, maxTokens int) (string, error)
}

type AuditPort interface {
	LogAction(entityType, entityID, action, detail, userID string) error
}

type WorkflowPort interface {
	GenerateReport(reportType, query string) (string, error)
	CalculateUninvoicedOrderExposure(start, end time.Time) (float64, int, error)
	CreateTask(title, description, assigneeID, customerID string) (map[string]any, error)
}

type UserInfo struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	FullName    string `json:"full_name"`
	Role        string `json:"role"`
}

type ButlerAppContext interface {
	GetCustomerByID(id string) (any, error)
	GetCustomerByName(name string) (any, error)
	GetSupplierByID(id string) (any, error)
	FindEntity(query string) ([]any, error)

	GetInvoiceSummary(customerID string) (any, error)
	GetPaymentHistory(customerID string) (any, error)
	GetOutstandingBalance(customerID string) (float64, error)
	GetCashPosition() (any, error)

	CreateTask(description, assignee string) (any, error)
	GetPendingTasks() ([]any, error)

	CreateOfferDraft(data any) (any, error)
	GetPipelineSummary() (any, error)

	CurrentUser() UserInfo
	CurrentDivision() string
	CurrentEmployee() any

	RawQuery(sql string, args ...any) ([]map[string]any, error)
}

type ChatService interface {
	StartConversation(ctx context.Context, title string) (Conversation, error)
	GetConversation(ctx context.Context, conversationID string) (Conversation, error)
	ListMessages(ctx context.Context, conversationID string) ([]ChatMessage, error)
	SendMessage(ctx context.Context, conversationID, content string) (ChatMessage, error)
	SummarizeConversation(ctx context.Context, conversationID string) (string, error)
}

type IntentRouter interface {
	Route(ctx context.Context, message string) (map[string]any, error)
	ResolveEntities(ctx context.Context, message string) (map[string]any, error)
	ExecuteAction(ctx context.Context, intent string, payload map[string]any) (map[string]any, error)
}

type ReportGenerator interface {
	CreateReportDraft(ctx context.Context, reportType, to string, params map[string]any) (string, error)
	SendReportByEmail(ctx context.Context, reportType, to string, params map[string]any) error
	GeneratePDFReport(ctx context.Context, reportType string, params map[string]any) (string, error)
}

type Predictor interface {
	PredictPayment(ctx context.Context, customerID string) (PredictionRecord, error)
	PredictWinProbability(ctx context.Context, offerID string) (WinProbabilityPrediction, error)
	RecommendDiscount(ctx context.Context, offerID string) (DiscountRecommendationRecord, error)
	GetCustomerHistory(ctx context.Context, customerID string) ([]PredictionRecord, error)
}
