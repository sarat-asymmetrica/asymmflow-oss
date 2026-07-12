package main

import (
	"encoding/json"
	"fmt"
	"strings"

	butlerdomain "ph_holdings_app/pkg/butler"
)

type appButlerContext struct {
	app *App
}

func (p appButlerContext) GetCustomerByID(id string) (any, error) {
	if p.app == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	return p.app.GetCustomer(strings.TrimSpace(id))
}

func (p appButlerContext) GetCustomerByName(name string) (any, error) {
	if p.app == nil || p.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var customer CustomerMaster
	query := strings.TrimSpace(name)
	if query == "" {
		return nil, fmt.Errorf("customer name is required")
	}
	if err := p.app.db.
		Where("LOWER(business_name) = LOWER(?) OR LOWER(customer_id) = LOWER(?)", query, query).
		First(&customer).Error; err == nil {
		return customer, nil
	}
	if err := p.app.db.
		Where("business_name LIKE ? OR customer_id LIKE ?", "%"+query+"%", "%"+query+"%").
		Order("business_name ASC").
		First(&customer).Error; err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}
	return customer, nil
}

func (p appButlerContext) GetSupplierByID(id string) (any, error) {
	if p.app == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	return p.app.GetSupplier(strings.TrimSpace(id))
}

func (p appButlerContext) FindEntity(query string) ([]any, error) {
	if p.app == nil || p.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return []any{}, nil
	}
	like := "%" + query + "%"
	results := []any{}

	var customers []CustomerMaster
	if err := p.app.db.Where("business_name LIKE ? OR customer_id LIKE ?", like, like).Limit(10).Find(&customers).Error; err == nil {
		for _, customer := range customers {
			results = append(results, customer)
		}
	}
	var suppliers []SupplierMaster
	if err := p.app.db.Where("business_name LIKE ? OR supplier_id LIKE ?", like, like).Limit(10).Find(&suppliers).Error; err == nil {
		for _, supplier := range suppliers {
			results = append(results, supplier)
		}
	}
	var offers []Offer
	if err := p.app.db.Where("offer_number LIKE ? OR customer_name LIKE ?", like, like).Limit(10).Find(&offers).Error; err == nil {
		for _, offer := range offers {
			results = append(results, offer)
		}
	}
	var orders []Order
	if err := p.app.db.Where("order_number LIKE ? OR customer_name LIKE ?", like, like).Limit(10).Find(&orders).Error; err == nil {
		for _, order := range orders {
			results = append(results, order)
		}
	}
	return results, nil
}

func (p appButlerContext) GetInvoiceSummary(customerID string) (any, error) {
	if p.app == nil || p.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var row struct {
		InvoiceCount   int64   `json:"invoice_count"`
		TotalBHD       float64 `json:"total_bhd"`
		OutstandingBHD float64 `json:"outstanding_bhd"`
		OverdueBHD     float64 `json:"overdue_bhd"`
	}
	err := p.app.db.Model(&Invoice{}).
		Where("customer_id = ?", strings.TrimSpace(customerID)).
		Select(`COUNT(*) AS invoice_count,
			COALESCE(SUM(grand_total_bhd), 0) AS total_bhd,
			COALESCE(SUM(outstanding_bhd), 0) AS outstanding_bhd,
			COALESCE(SUM(CASE WHEN status = 'Overdue' THEN outstanding_bhd ELSE 0 END), 0) AS overdue_bhd`).
		Scan(&row).Error
	return row, err
}

func (p appButlerContext) GetPaymentHistory(customerID string) (any, error) {
	if p.app == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	return p.app.GetPaymentHistory(strings.TrimSpace(customerID), 20), nil
}

func (p appButlerContext) GetOutstandingBalance(customerID string) (float64, error) {
	if p.app == nil || p.app.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	var total float64
	err := p.app.db.Model(&Invoice{}).
		Where("customer_id = ? AND outstanding_bhd > 0", strings.TrimSpace(customerID)).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&total).Error
	return total, err
}

func (p appButlerContext) GetCashPosition() (any, error) {
	if p.app == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	return computeCashPositionSnapshot(p.app)
}

func (p appButlerContext) CreateTask(description, assignee string) (any, error) {
	if p.app == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	title := strings.TrimSpace(description)
	if len(title) > 80 {
		title = strings.TrimSpace(title[:80])
	}
	if title == "" {
		title = "Butler follow-up"
	}

	var assigneeID *string
	if strings.TrimSpace(assignee) != "" {
		if resolved := p.app.resolveEmployeeReference(assignee); resolved != nil && strings.TrimSpace(resolved.EntityID) != "" {
			id := resolved.EntityID
			assigneeID = &id
		}
	}

	return p.app.CreateCollaborativeTask(TaskItem{
		Title:              title,
		Description:        strings.TrimSpace(description),
		TaskType:           "butler",
		Status:             "open",
		Priority:           "medium",
		AssigneeEmployeeID: assigneeID,
	})
}

func (p appButlerContext) GetPendingTasks() ([]any, error) {
	if p.app == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	tasks, err := p.app.ListCollaborativeTeamTasks(false)
	if err != nil {
		return nil, err
	}
	results := make([]any, 0, len(tasks))
	for _, task := range tasks {
		if strings.EqualFold(task.Status, "completed") || strings.EqualFold(task.Status, "archived") {
			continue
		}
		results = append(results, task)
	}
	return results, nil
}

func (p appButlerContext) CreateOfferDraft(data any) (any, error) {
	if p.app == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	var req ButlerOfferDraftRequest
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	return p.app.CreateOfferDraftFromButler(req)
}

func (p appButlerContext) GetPipelineSummary() (any, error) {
	if p.app == nil || p.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := p.RawQuery(`SELECT COALESCE(stage, 'Unstaged') AS stage, COUNT(*) AS count, COALESCE(SUM(revenue_bhd), 0) AS total_bhd
		FROM opportunities
		WHERE deleted_at IS NULL
		GROUP BY COALESCE(stage, 'Unstaged')
		ORDER BY total_bhd DESC`)
	if err != nil {
		return nil, err
	}
	return map[string]any{"stages": rows}, nil
}

func (p appButlerContext) CurrentUser() butlerdomain.UserInfo {
	if p.app == nil || p.app.currentUser == nil {
		return butlerdomain.UserInfo{}
	}
	user := p.app.currentUser
	return butlerdomain.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		FullName:    user.FullName,
		Role:        p.app.GetCurrentUserRole(),
	}
}

func (p appButlerContext) CurrentDivision() string {
	return activeOverlay.DefaultDivision()
}

func (p appButlerContext) CurrentEmployee() any {
	if p.app == nil {
		return nil
	}
	current, err := p.app.GetCurrentEmployeeContext()
	if err != nil {
		return nil
	}
	return current
}

func (p appButlerContext) RawQuery(sql string, args ...any) ([]map[string]any, error) {
	return appButlerDatabasePort(p).RawQuery(sql, args...)
}

func (a *App) butlerAppContext() butlerdomain.ButlerAppContext {
	if a.services.butlerContext != nil {
		return a.services.butlerContext
	}
	return appButlerContext{app: a}
}
