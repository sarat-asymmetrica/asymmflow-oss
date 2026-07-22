package main

import (
	"fmt"
	"strings"
	"time"

	butlerdomain "ph_holdings_app/pkg/butler"
)

type appButlerDatabasePort struct {
	app *App
}

func (p appButlerDatabasePort) QueryInvoices(filter map[string]any) ([]map[string]any, error) {
	return p.QueryTable("invoices", filter, 0)
}

func (p appButlerDatabasePort) QueryCustomers(filter map[string]any) ([]map[string]any, error) {
	return p.QueryTable("customers", filter, 0)
}

func (p appButlerDatabasePort) QueryOrders(filter map[string]any) ([]map[string]any, error) {
	return p.QueryTable("orders", filter, 0)
}

func (p appButlerDatabasePort) QueryPayments(filter map[string]any) ([]map[string]any, error) {
	return p.QueryTable("payments", filter, 0)
}

func (p appButlerDatabasePort) QueryOffers(filter map[string]any) ([]map[string]any, error) {
	return p.QueryTable("offers", filter, 0)
}

func (p appButlerDatabasePort) QueryTable(table string, filter map[string]any, limit int) ([]map[string]any, error) {
	if p.app == nil || p.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if !isValidSQLIdentifier(table) {
		return nil, fmt.Errorf("invalid table name")
	}

	var rows []map[string]any
	query := p.app.db.Table(table)
	for key, value := range filter {
		if !isValidSQLIdentifier(key) {
			return nil, fmt.Errorf("invalid filter column %q", key)
		}
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (p appButlerDatabasePort) RawQuery(sql string, args ...any) ([]map[string]any, error) {
	if p.app == nil || p.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var rows []map[string]any
	if err := p.app.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (p appButlerDatabasePort) Count(table string, filter map[string]any) (int64, error) {
	if p.app == nil || p.app.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	if !isValidSQLIdentifier(table) {
		return 0, fmt.Errorf("invalid table name")
	}
	var count int64
	query := p.app.db.Table(table)
	for key, value := range filter {
		if !isValidSQLIdentifier(key) {
			return 0, fmt.Errorf("invalid filter column %q", key)
		}
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

type appButlerUserContextPort struct {
	app *App
}

func (p appButlerUserContextPort) CurrentUserID() string {
	if p.app == nil {
		return ""
	}
	return p.app.getCurrentUserID()
}

func (p appButlerUserContextPort) CurrentUserName() string {
	if p.app == nil || p.app.currentUser == nil {
		return ""
	}
	return firstNonEmpty(p.app.currentUser.DisplayName, p.app.currentUser.FullName, p.app.currentUser.Username)
}

func (p appButlerUserContextPort) CurrentUserRole() string {
	if p.app == nil {
		return ""
	}
	return p.app.GetCurrentUserRole()
}

func (p appButlerUserContextPort) CurrentLicenseRole() string {
	if p.app == nil {
		return ""
	}
	return p.app.GetLicenseRole()
}

func (p appButlerUserContextPort) CurrentDivision() string {
	return activeOverlay.DefaultDivision()
}

func (p appButlerUserContextPort) HasPermission(action string) bool {
	if p.app == nil {
		return false
	}
	return p.app.currentSessionHasPermission(action) || p.app.requirePermission(action) == nil
}

type appButlerLLMPort struct{}

func (appButlerLLMPort) ChatCompletion(systemPrompt, userMessage string, maxTokens int) (string, error) {
	return callMistral(mistralModelLarge, systemPrompt, userMessage)
}

func (appButlerLLMPort) ChatCompletionWithHistory(messages []butlerdomain.ChatMessage, maxTokens int) (string, error) {
	payload := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		payload = append(payload, map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	return callMistralWithMessages(mistralModelLarge, payload)
}

type appButlerAuditPort struct {
	app *App
}

func (p appButlerAuditPort) LogAction(entityType, entityID, action, detail, userID string) error {
	if p.app == nil {
		return fmt.Errorf("app not initialized")
	}
	resourceID := entityID
	p.app.logAudit(&userID, action, entityType, &resourceID, detail)
	return nil
}

type appButlerWorkflowPort struct {
	app *App
}

func (p appButlerWorkflowPort) GenerateReport(reportType, query string) (string, error) {
	if p.app == nil {
		return "", fmt.Errorf("app not initialized")
	}
	return p.app.GenerateButlerReport(reportType, query)
}

func (p appButlerWorkflowPort) CalculateUninvoicedOrderExposure(start, end time.Time) (float64, int, error) {
	if p.app == nil {
		return 0, 0, fmt.Errorf("app not initialized")
	}
	return p.app.calculateUninvoicedOrderExposure(start, end)
}

func (p appButlerWorkflowPort) CreateTask(title, description, assigneeID, customerID string) (map[string]any, error) {
	if p.app == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	var assignee *string
	if strings.TrimSpace(assigneeID) != "" {
		assignee = &assigneeID
	}
	var customer *string
	if strings.TrimSpace(customerID) != "" {
		customer = &customerID
	}
	task := TaskItem{
		Title:       title,
		Description: description,
		CustomerID:  customer,
		Status:      "pending",
		Priority:    "medium",
	}
	task.AssigneeEmployeeID = assignee
	created, err := p.app.CreateCollaborativeTask(task)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":          created.ID,
		"title":       created.Title,
		"description": created.Description,
		"assignee_id": created.AssigneeEmployeeID,
		"customer_id": created.CustomerID,
		"status":      created.Status,
		"priority":    created.Priority,
	}, nil
}

func (a *App) butlerDatabasePort() butlerdomain.DatabasePort {
	return appButlerDatabasePort{app: a}
}

func (a *App) butlerUserContextPort() butlerdomain.UserContextPort {
	return appButlerUserContextPort{app: a}
}

func (a *App) butlerLLMPort() butlerdomain.LLMPort {
	return appButlerLLMPort{}
}

func (a *App) butlerAuditPort() butlerdomain.AuditPort {
	return appButlerAuditPort{app: a}
}

func (a *App) butlerWorkflowPort() butlerdomain.WorkflowPort {
	return appButlerWorkflowPort{app: a}
}
