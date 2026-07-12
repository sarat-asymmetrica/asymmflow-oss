package main

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	crmcustomer "ph_holdings_app/pkg/crm/customer"
	"ph_holdings_app/pkg/survival_garden"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

func (a *App) SimulateSurvivalGarden(cashRunway float64, monthlyBurn float64, expenses []map[string]any, months int) ([]map[string]any, error) {
	// SECURITY: Require dashboard view permission
	if err := a.requirePermission("dashboard:view"); err != nil {
		return nil, err
	}

	// INPUT VALIDATION (Margaret Hamilton says: VALIDATE EVERYTHING)
	if cashRunway < 0 {
		return nil, fmt.Errorf("cashRunway cannot be negative: %.2f", cashRunway)
	}
	if monthlyBurn <= 0 {
		return nil, fmt.Errorf("monthlyBurn must be positive: %.2f", monthlyBurn)
	}
	if months < 0 || months > 240 { // 20 years max
		return nil, fmt.Errorf("months out of range [0, 240]: %d", months)
	}

	// SAFE TYPE CONVERSION
	goExpenses := make([]survival_garden.Expense, len(expenses))
	for i, exp := range expenses {
		name, ok := exp["name"].(string)
		if !ok {
			return nil, fmt.Errorf("expense %d: invalid name type", i)
		}

		amount, ok := exp["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("expense %d (%s): invalid amount type", i, name)
		}
		if amount < 0 {
			return nil, fmt.Errorf("expense %d (%s): negative amount %.2f", i, name, amount)
		}

		weight, ok := exp["weight"].(float64)
		if !ok {
			return nil, fmt.Errorf("expense %d (%s): invalid weight type", i, name)
		}
		if weight < 0 || weight > 1 {
			return nil, fmt.Errorf("expense %d (%s): weight out of range [0,1]: %.2f", i, name, weight)
		}

		goExpenses[i] = survival_garden.Expense{
			Name:   name,
			Amount: amount,
			Weight: weight,
		}
	}

	params := survival_garden.SimulationParams{
		CashRunway:       cashRunway,
		MonthlyBurn:      monthlyBurn,
		Expenses:         goExpenses,
		MonthsToSimulate: months,
		GPUAccelerate:    true, // Request GPU acceleration
	}

	states, err := survival_garden.SimulateSurvivalGarden(params)
	if err != nil {
		log.Printf("❌ Garden simulation error: %v", err)
		return nil, fmt.Errorf("garden simulation error: %w", err)
	}

	// Convert []GardenState to []map[string]interface{} for frontend
	result := make([]map[string]any, len(states))
	for i, state := range states {
		result[i] = map[string]any{
			"waterLevel":    state.WaterLevel,
			"stoneHeights":  state.StoneHeights,
			"particleCount": state.ParticleCount,
			"regime":        state.Regime,
			"temperature":   state.Temperature,
			"turbulence":    state.Turbulence,
		}
	}

	log.Printf("✅ Survival Garden: Simulated %d months (GPU=%t)", len(states), params.GPUAccelerate)
	return result, nil
}

// ============================================================================
// EMAIL INTEGRATION - Microsoft Graph Email via Reports
// ============================================================================

// SendReportByEmail generates a report and sends it via email
func (a *App) SendReportByEmail(reportType string, to string, params map[string]any) error {
	if err := a.requirePermission("reports:export"); err != nil {
		return err
	}
	if a.emailService == nil {
		return fmt.Errorf("email service not initialized")
	}

	var pdfPath string
	var err error
	var subject string
	var reportName string

	switch reportType {
	case "dashboard":
		pdfPath, err = a.GenerateDashboardReport()
		subject = activeOverlay.CompanyDisplayName + " - Dashboard Report"
		reportName = "Dashboard Analytics"
	case "customer360":
		customerID, ok := params["customerID"].(string)
		if !ok {
			return fmt.Errorf("customerID required for customer360 report")
		}
		pdfPath, err = a.GenerateCustomer360Report(customerID)
		subject = fmt.Sprintf("%s - Customer Report: %s", activeOverlay.CompanyDisplayName, customerID)
		reportName = fmt.Sprintf("Customer 360 View - %s", customerID)
	case "history":
		limitFloat, ok := params["limit"].(float64)
		if !ok {
			limitFloat = 100 // Default limit
		}
		limit := int(limitFloat)
		pdfPath, err = a.GeneratePredictionHistoryReport(limit)
		subject = activeOverlay.CompanyDisplayName + " - Prediction History Report"
		reportName = "Payment Prediction History"
	default:
		return fmt.Errorf("unknown report type: %s", reportType)
	}

	if err != nil {
		return fmt.Errorf("failed to generate report: %v", err)
	}

	// Format professional email body
	bodyHTML := a.emailService.FormatReportBody(reportType, reportName)

	// Send email with attachment
	err = a.emailService.SendReportEmail(to, subject, bodyHTML, pdfPath)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("✅ Report emailed successfully to %s", to)
	return nil
}

// CreateReportDraft generates a report and creates an email draft
func (a *App) CreateReportDraft(reportType string, to string, params map[string]any) (string, error) {
	if err := a.requirePermission("reports:export"); err != nil {
		return "", err
	}
	if a.emailService == nil {
		return "", fmt.Errorf("email service not initialized")
	}

	var pdfPath string
	var err error
	var subject string
	var reportName string

	switch reportType {
	case "dashboard":
		pdfPath, err = a.GenerateDashboardReport()
		subject = activeOverlay.CompanyDisplayName + " - Dashboard Report"
		reportName = "Dashboard Analytics"
	case "customer360":
		customerID, ok := params["customerID"].(string)
		if !ok {
			return "", fmt.Errorf("customerID required for customer360 report")
		}
		pdfPath, err = a.GenerateCustomer360Report(customerID)
		subject = fmt.Sprintf("%s - Customer Report: %s", activeOverlay.CompanyDisplayName, customerID)
		reportName = fmt.Sprintf("Customer 360 View - %s", customerID)
	case "history":
		limitFloat, ok := params["limit"].(float64)
		if !ok {
			limitFloat = 100 // Default limit
		}
		limit := int(limitFloat)
		pdfPath, err = a.GeneratePredictionHistoryReport(limit)
		subject = activeOverlay.CompanyDisplayName + " - Prediction History Report"
		reportName = "Payment Prediction History"
	default:
		return "", fmt.Errorf("unknown report type: %s", reportType)
	}

	if err != nil {
		return "", fmt.Errorf("failed to generate report: %v", err)
	}

	// Format professional email body
	bodyHTML := a.emailService.FormatReportBody(reportType, reportName)

	// Create draft
	draftID, err := a.emailService.CreateReportDraft(to, subject, bodyHTML, pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to create draft: %v", err)
	}

	log.Printf("✅ Draft created successfully: %s", draftID)
	return draftID, nil
}

// ============================================================================
// ORDERS MANAGEMENT (Stubs for OrdersScreen.svelte)
// ============================================================================

// CreateOrder creates a new order manually
func (a *App) CreateOrder(orderNumber string, customerName string, amount float64, orderDateStr string, status string) (*Order, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// SERVER-SIDE PERMISSION CHECK: Require orders:create or admin (wildcard)
	if err := a.requirePermission("orders:create"); err != nil {
		log.Printf("🔒 CreateOrder blocked: %v", err)
		return nil, newError("PERMISSION_DENIED", err.Error(), "")
	}

	// RES-004: validate string inputs (amount is validated below). Orders
	// previously accepted empty/oversized order numbers and customer names.
	orderNumber = strings.TrimSpace(orderNumber)
	customerName = strings.TrimSpace(customerName)
	if orderNumber == "" || len(orderNumber) > 100 {
		return nil, newError("INVALID_ORDER_NUMBER", "Order number is required and must be at most 100 characters", "")
	}
	if customerName == "" || len(customerName) > 255 {
		return nil, newError("INVALID_CUSTOMER_NAME", "Customer name is required and must be at most 255 characters", "")
	}
	if len(status) > 50 {
		return nil, newError("INVALID_ORDER_STATUS", "Order status must be at most 50 characters", "")
	}

	var orderDate time.Time
	var err error
	if orderDateStr != "" {
		orderDate, err = time.Parse("2006-01-02", orderDateStr)
		if err != nil {
			return nil, newError("INVALID_DATE", "Invalid date format (expected YYYY-MM-DD)", err.Error())
		}
	} else {
		orderDate = time.Now()
	}

	// Validate amount
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return nil, newError("INVALID_AMOUNT", "Order amount must be a positive number", "")
	}
	if amount > 1000000000 {
		return nil, newError("INVALID_AMOUNT", "Order amount exceeds maximum allowed value", "")
	}
	amount = math.Round(amount*1000) / 1000 // Round to 3 decimal places (BHD precision)

	// Lookup customer ID
	var customer CustomerMaster
	var customerID string
	if err := a.db.Where("business_name = ?", customerName).First(&customer).Error; err == nil {
		customerID = customer.ID
	}

	order := &Order{
		OrderNumber:   orderNumber,
		CustomerName:  customerName,
		CustomerID:    customerID,
		GrandTotalBHD: amount,
		TotalValueBHD: amount,
		OrderDate:     orderDate,
		Status:        status,
		Division:      activeOverlay.DefaultDivision(),
		// Defaults
		PaymentTerms:  "Net 30",
		DeliveryTerms: "DDP",
	}

	if err := a.db.Create(order).Error; err != nil {
		log.Printf("❌ Failed to create order: %v", err)
		return nil, err
	}

	log.Printf("✅ Created Order #%s for %s (%.2f BHD)", orderNumber, customerName, amount)
	return order, nil
}

// CreateOrderWithItems creates a new order header AND its line items in a
// SINGLE transaction. Wave 9.7 tight-ship fix: the manual-create flow used to
// call CreateOrder (header only) followed by a separate UpdateOrder (attach
// items) from the frontend — if the second call failed, a header-only
// "ghost" order persisted (visible via GetOrdersWithNoItems). This method
// collapses both steps into one atomic insert so a partial order can never
// be observed. Validation mirrors CreateOrder (order number / customer name /
// status bounds, amount bounds); item normalization mirrors UpdateOrder's
// item-replacement branch (rounding, synthetic-summary filtering, computed
// totals) — neither is changed here, just fused into one transaction.
func (a *App) CreateOrderWithItems(order Order, items []OrderItem) (*Order, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// SERVER-SIDE PERMISSION CHECK: same gate as CreateOrder.
	if err := a.requirePermission("orders:create"); err != nil {
		log.Printf("🔒 CreateOrderWithItems blocked: %v", err)
		return nil, newError("PERMISSION_DENIED", err.Error(), "")
	}

	orderNumber := strings.TrimSpace(order.OrderNumber)
	customerName := strings.TrimSpace(order.CustomerName)
	if orderNumber == "" || len(orderNumber) > 100 {
		return nil, newError("INVALID_ORDER_NUMBER", "Order number is required and must be at most 100 characters", "")
	}
	if customerName == "" || len(customerName) > 255 {
		return nil, newError("INVALID_CUSTOMER_NAME", "Customer name is required and must be at most 255 characters", "")
	}
	if len(order.Status) > 50 {
		return nil, newError("INVALID_ORDER_STATUS", "Order status must be at most 50 characters", "")
	}
	if len(items) == 0 {
		return nil, newError("INVALID_ITEMS", "At least one order line item is required", "")
	}

	// Uniqueness check up front (also enforced by the DB unique index; this
	// gives a friendlier error before we open the transaction).
	var dupeCount int64
	if err := a.db.Model(&Order{}).Where("order_number = ?", orderNumber).Count(&dupeCount).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to validate order number uniqueness", err.Error())
	}
	if dupeCount > 0 {
		return nil, fmt.Errorf("order number %s already exists", orderNumber)
	}

	if order.OrderDate.IsZero() {
		order.OrderDate = time.Now()
	}

	roundMoney := func(value float64) float64 {
		return math.Round(value*1000) / 1000
	}

	// Lookup customer ID if the caller didn't already resolve one.
	customerID := strings.TrimSpace(order.CustomerID)
	if customerID == "" {
		var customer CustomerMaster
		if err := a.db.Where("business_name = ?", customerName).First(&customer).Error; err == nil {
			customerID = customer.ID
		}
	}

	// Item normalization mirrors UpdateOrder's item-replacement branch: strip
	// synthetic commercial-summary rows, round money fields, backfill
	// unit-price/total-price from one another, and renumber lines. Item IDs
	// are left as supplied by the caller (a brand-new order has no prior
	// items to preserve fulfillment counters against) — gorm's Base.BeforeCreate
	// only assigns a UUID when ID is empty, so this is a no-op for the normal
	// (no-ID) create path.
	normalizedItems := make([]OrderItem, 0, len(items))
	computedTotal := 0.0
	for _, item := range items {
		if isSyntheticCommercialSummary(item.Description, item.ProductCode, item.Model, item.Equipment) {
			continue
		}

		item.OrderID = "" // set once the order ID is known, inside the transaction
		item.LineNumber = len(normalizedItems) + 1

		item.Quantity = roundMoney(item.Quantity)
		item.UnitPrice = roundMoney(item.UnitPrice)
		item.TotalPrice = roundMoney(item.TotalPrice)

		if item.UnitPrice <= 0 && item.Quantity > 0 && item.TotalPrice > 0 {
			item.UnitPrice = roundMoney(item.TotalPrice / item.Quantity)
		}
		if item.TotalPrice <= 0 && item.Quantity > 0 && item.UnitPrice > 0 {
			item.TotalPrice = roundMoney(item.Quantity * item.UnitPrice)
		}

		computedTotal += item.TotalPrice
		normalizedItems = append(normalizedItems, item)
	}

	if len(normalizedItems) == 0 {
		return nil, newError("INVALID_ITEMS", "At least one non-summary order line item is required", "")
	}

	totalValue := roundMoney(computedTotal)
	if totalValue <= 0 || math.IsNaN(totalValue) || math.IsInf(totalValue, 0) {
		return nil, newError("INVALID_AMOUNT", "Order amount must be a positive number", "")
	}
	if totalValue > 1000000000 {
		return nil, newError("INVALID_AMOUNT", "Order amount exceeds maximum allowed value", "")
	}

	division := strings.TrimSpace(order.Division)
	if division == "" {
		division = activeOverlay.DefaultDivision()
	}
	paymentTerms := order.PaymentTerms
	if paymentTerms == "" {
		paymentTerms = "Net 30"
	}
	deliveryTerms := order.DeliveryTerms
	if deliveryTerms == "" {
		deliveryTerms = "DDP"
	}

	newOrder := &Order{
		OrderNumber:       orderNumber,
		CustomerPONumber:  order.CustomerPONumber,
		CustomerID:        customerID,
		CustomerName:      customerName,
		OrderDate:         order.OrderDate,
		RequiredDate:      order.RequiredDate,
		TotalValueBHD:     totalValue,
		GrandTotalBHD:     totalValue,
		Status:            order.Status,
		PaymentTerms:      paymentTerms,
		DeliveryTerms:     deliveryTerms,
		OfferID:           order.OfferID,
		OfferNumber:       order.OfferNumber,
		RFQID:             order.RFQID,
		CustomerReference: order.CustomerReference,
		AttentionPerson:   order.AttentionPerson,
		AttentionCompany:  order.AttentionCompany,
		AttentionPhone:    order.AttentionPhone,
		AttentionAddress:  order.AttentionAddress,
		DeliveryWeeks:     order.DeliveryWeeks,
		CountryOfOrigin:   order.CountryOfOrigin,
		IssuedBy:          order.IssuedBy,
		ContactPhone:      order.ContactPhone,
		DiscountPercent:   order.DiscountPercent,
		Division:          division,
	}

	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(newOrder).Error; err != nil {
			return newError("DB_CREATE_FAILED", "Failed to create order", err.Error())
		}

		for i := range normalizedItems {
			normalizedItems[i].OrderID = newOrder.ID
		}
		if err := tx.Create(&normalizedItems).Error; err != nil {
			return newError("DB_CREATE_FAILED", "Failed to save order items", err.Error())
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Reload with items
	if err := a.db.Preload("Items").First(newOrder, "id = ?", newOrder.ID).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to reload order", err.Error())
	}

	log.Printf("✅ Created Order #%s with %d item(s) for %s (%.3f BHD)", orderNumber, len(normalizedItems), customerName, totalValue)
	return newOrder, nil
}

// ListOrders retrieves all orders with optional pagination
func (a *App) ListOrders(limit int, offset int) ([]Order, error) {
	if err := a.requirePermission("orders:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Default limit: 100, max limit: 1000
	if limit <= 0 {
		limit = 100
	} else if limit > 1000 {
		limit = 1000
	}

	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	var orders []Order
	// P2 FIX: Use Select() to fetch only necessary columns for list view (no items preload)
	query := a.db.Model(&Order{}).
		Select("id, order_number, customer_id, customer_name, order_date, required_date, total_value_bhd, grand_total_bhd, status, customer_po_number, division, created_at, updated_at").
		Where(`
			NOT (
				COALESCE(total_value_bhd, 0) <= 0
				AND NOT EXISTS (
					SELECT 1
					FROM order_items oi
					WHERE oi.order_id = orders.id
					  AND oi.deleted_at IS NULL
				)
			)
		`).
		Order("order_date DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&orders).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to list orders", err.Error())
	}
	orders = dedupeOrdersForList(orders)

	log.Printf("Retrieved %d orders (limit=%d, offset=%d)", len(orders), limit, offset)
	return orders, nil
}

// GetOrder retrieves a single order by ID
func (a *App) GetOrder(orderID string) (Order, error) {
	if err := a.requirePermission("orders:view"); err != nil {
		return Order{}, err
	}
	if a.db == nil {
		return Order{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var order Order
	// P2 FIX: Preload items to avoid N+1 query
	if err := a.db.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
		return Order{}, newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	return order, nil
}

// UpdateOrder updates an existing order (full update)
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) UpdateOrder(id string, order Order) (*Order, error) {
	if err := a.requirePermission("orders:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Check if order exists
	var existing Order
	if err := a.db.First(&existing, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newError("NOT_FOUND", "Order not found", fmt.Sprintf("ID: %s", id))
		}
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve order", err.Error())
	}

	roundMoney := func(value float64) float64 {
		return math.Round(value*1000) / 1000
	}

	// Preserve ID (prevent ID override)
	order.ID = existing.ID

	// Uniqueness check: if order number changed, verify no other order has the same number
	if order.OrderNumber != "" && order.OrderNumber != existing.OrderNumber {
		var count int64
		a.db.Model(&Order{}).Where("order_number = ? AND id != ?", order.OrderNumber, order.ID).Count(&count)
		if count > 0 {
			return nil, fmt.Errorf("order number %s already exists", order.OrderNumber)
		}
	}

	if order.CustomerID == "" && strings.TrimSpace(order.CustomerName) != "" {
		var customer CustomerMaster
		if err := a.db.Where("business_name = ?", strings.TrimSpace(order.CustomerName)).First(&customer).Error; err == nil {
			order.CustomerID = customer.ID
		}
	}

	err := a.db.Transaction(func(tx *gorm.DB) error {
		if order.Items != nil {
			// Mission I (I-12): item replacement previously wiped fulfillment
			// tracking (QuantityShipped/QuantityInvoiced) on every order edit —
			// the delete+recreate rebuilt rows from the client payload, which
			// never carries those server-owned counters. Snapshot them first and
			// carry them onto the recreated rows (matched by prior item ID, then
			// by line number for payloads that omit IDs). Latent in PH too;
			// surfaced to the Commander in the Mission I wave report.
			var priorItems []OrderItem
			if err := tx.Where("order_id = ?", existing.ID).Find(&priorItems).Error; err != nil {
				return newError("DB_QUERY_FAILED", "Failed to load existing order items", err.Error())
			}
			fulfillmentByID := make(map[string]OrderItem, len(priorItems))
			fulfillmentByLine := make(map[int]OrderItem, len(priorItems))
			for _, prior := range priorItems {
				fulfillmentByID[prior.ID] = prior
				fulfillmentByLine[prior.LineNumber] = prior
			}

			normalizedItems := make([]OrderItem, 0, len(order.Items))
			computedTotal := 0.0

			for _, item := range order.Items {
				if isSyntheticCommercialSummary(item.Description, item.ProductCode, item.Model, item.Equipment) {
					continue
				}
				if prior, ok := fulfillmentByID[item.ID]; ok {
					item.QuantityShipped = prior.QuantityShipped
					item.QuantityInvoiced = prior.QuantityInvoiced
				} else if prior, ok := fulfillmentByLine[item.LineNumber]; ok && item.ID == "" {
					item.QuantityShipped = prior.QuantityShipped
					item.QuantityInvoiced = prior.QuantityInvoiced
				}
				item.ID = ""
				item.OrderID = existing.ID
				item.LineNumber = len(normalizedItems) + 1

				item.Quantity = roundMoney(item.Quantity)
				item.UnitPrice = roundMoney(item.UnitPrice)
				item.TotalPrice = roundMoney(item.TotalPrice)

				if item.UnitPrice <= 0 && item.Quantity > 0 && item.TotalPrice > 0 {
					item.UnitPrice = roundMoney(item.TotalPrice / item.Quantity)
				}
				if item.TotalPrice <= 0 && item.Quantity > 0 && item.UnitPrice > 0 {
					item.TotalPrice = roundMoney(item.Quantity * item.UnitPrice)
				}

				computedTotal += item.TotalPrice
				normalizedItems = append(normalizedItems, item)
			}

			if err := tx.Where("order_id = ?", existing.ID).Delete(&OrderItem{}).Error; err != nil {
				return newError("DB_UPDATE_FAILED", "Failed to replace order items", err.Error())
			}

			if len(normalizedItems) > 0 {
				if err := tx.Create(&normalizedItems).Error; err != nil {
					return newError("DB_CREATE_FAILED", "Failed to save order items", err.Error())
				}
				order.TotalValueBHD = roundMoney(computedTotal)
				order.GrandTotalBHD = roundMoney(computedTotal)
			} else if order.TotalValueBHD > 0 {
				order.TotalValueBHD = roundMoney(order.TotalValueBHD)
				order.GrandTotalBHD = roundMoney(order.TotalValueBHD)
			}
		}

		// Mission I (I-12): CreatedBy/CreatedAt are server-owned audit fields —
		// never mass-assignable from a client payload.
		if err := tx.Model(&existing).Omit("Items", "CreatedBy", "CreatedAt").Updates(order).Error; err != nil {
			return newError("DB_UPDATE_FAILED", "Failed to update order", err.Error())
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Reload with items
	if err := a.db.Preload("Items").First(&existing, "id = ?", id).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to reload order", err.Error())
	}

	log.Printf("✅ Updated Order: %s", id)
	return &existing, nil
}

// ============================================================================
// Order delete cascade (ported from deployed PH): a single snapshot collector
// feeds both the safe-delete preview (PreviewOrderDeleteCascade) and the
// delete itself, which re-collects INSIDE the transaction so the payment
// check cannot be raced (TOCTOU). Linked records are matched on the same
// widened conditions PH uses — legacy rows link invoices by order number,
// customer reference, customer PO, or offer, not just order_id.
// ============================================================================

type orderDeleteCascadeSnapshot struct {
	Order             Order
	InvoiceIDs        []string
	PurchaseOrderIDs  []string
	DeliveryNoteIDs   []string
	OrderItemCount    int64
	InvoiceItemCount  int64
	PurchaseItemCount int64
	DeliveryItemCount int64
	PaymentCount      int64
}

func appendOrderLinkClause(clauses *[]string, args *[]any, column string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	*clauses = append(*clauses, fmt.Sprintf("%s = ?", column))
	*args = append(*args, value)
}

func orderLinkedInvoiceCondition(order Order) (string, []any) {
	clauses := []string{}
	args := []any{}
	appendOrderLinkClause(&clauses, &args, "order_id", order.ID)
	appendOrderLinkClause(&clauses, &args, "order_id", order.OrderNumber)
	appendOrderLinkClause(&clauses, &args, "customer_reference", order.OrderNumber)
	appendOrderLinkClause(&clauses, &args, "customer_reference", order.CustomerReference)
	appendOrderLinkClause(&clauses, &args, "customer_po_number", order.CustomerPONumber)
	appendOrderLinkClause(&clauses, &args, "offer_id", order.OfferID)
	appendOrderLinkClause(&clauses, &args, "offer_number", order.OfferNumber)
	if len(clauses) == 0 {
		return "order_id = ?", []any{order.ID}
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func orderLinkedChildCondition(order Order) (string, []any) {
	clauses := []string{}
	args := []any{}
	appendOrderLinkClause(&clauses, &args, "order_id", order.ID)
	appendOrderLinkClause(&clauses, &args, "order_id", order.OrderNumber)
	if len(clauses) == 0 {
		return "order_id = ?", []any{order.ID}
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func (a *App) collectOrderDeleteCascade(db *gorm.DB, orderID string) (*orderDeleteCascadeSnapshot, error) {
	var order Order
	if err := db.First(&order, "id = ?", strings.TrimSpace(orderID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newError("ORDER_NOT_FOUND", "Order does not exist", orderID)
		}
		return nil, fmt.Errorf("failed to load order: %w", err)
	}

	invoiceWhere, invoiceArgs := orderLinkedInvoiceCondition(order)
	childWhere, childArgs := orderLinkedChildCondition(order)

	snapshot := &orderDeleteCascadeSnapshot{Order: order}
	if err := db.Model(&Invoice{}).Where(invoiceWhere, invoiceArgs...).Pluck("id", &snapshot.InvoiceIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to find linked invoices: %w", err)
	}
	if err := db.Model(&PurchaseOrder{}).Where(childWhere, childArgs...).Pluck("id", &snapshot.PurchaseOrderIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to find linked purchase orders: %w", err)
	}
	if err := db.Model(&DeliveryNote{}).Where(childWhere, childArgs...).Pluck("id", &snapshot.DeliveryNoteIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to find linked delivery notes: %w", err)
	}
	if err := db.Model(&OrderItem{}).Where("order_id = ?", order.ID).Count(&snapshot.OrderItemCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count order items: %w", err)
	}
	if len(snapshot.InvoiceIDs) > 0 {
		if err := db.Model(&DBInvoiceItem{}).Where("invoice_id IN ?", snapshot.InvoiceIDs).Count(&snapshot.InvoiceItemCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count invoice items: %w", err)
		}
		if err := db.Model(&Payment{}).Where("invoice_id IN ?", snapshot.InvoiceIDs).Count(&snapshot.PaymentCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count linked payments: %w", err)
		}
	}
	if len(snapshot.PurchaseOrderIDs) > 0 {
		if err := db.Model(&PurchaseOrderItem{}).Where("purchase_order_id IN ?", snapshot.PurchaseOrderIDs).Count(&snapshot.PurchaseItemCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count purchase order items: %w", err)
		}
	}
	if len(snapshot.DeliveryNoteIDs) > 0 {
		if err := db.Model(&DeliveryNoteItem{}).Where("delivery_note_id IN ?", snapshot.DeliveryNoteIDs).Count(&snapshot.DeliveryItemCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count delivery note items: %w", err)
		}
	}
	return snapshot, nil
}

func orderDeletePreviewMap(snapshot *orderDeleteCascadeSnapshot) map[string]any {
	summary := []string{fmt.Sprintf("%d order item(s)", snapshot.OrderItemCount)}
	if len(snapshot.InvoiceIDs) > 0 {
		summary = append(summary, fmt.Sprintf("%d customer invoice(s)", len(snapshot.InvoiceIDs)))
	}
	if snapshot.InvoiceItemCount > 0 {
		summary = append(summary, fmt.Sprintf("%d invoice line item(s)", snapshot.InvoiceItemCount))
	}
	if len(snapshot.PurchaseOrderIDs) > 0 {
		summary = append(summary, fmt.Sprintf("%d purchase order(s)", len(snapshot.PurchaseOrderIDs)))
	}
	if snapshot.PurchaseItemCount > 0 {
		summary = append(summary, fmt.Sprintf("%d purchase order line item(s)", snapshot.PurchaseItemCount))
	}
	if len(snapshot.DeliveryNoteIDs) > 0 {
		summary = append(summary, fmt.Sprintf("%d delivery note(s)", len(snapshot.DeliveryNoteIDs)))
	}
	if snapshot.DeliveryItemCount > 0 {
		summary = append(summary, fmt.Sprintf("%d delivery note line item(s)", snapshot.DeliveryItemCount))
	}

	blocked := snapshot.PaymentCount > 0
	blockReason := ""
	if blocked {
		blockReason = fmt.Sprintf("cannot delete order with %d linked payment(s); cancel or reverse payments first", snapshot.PaymentCount)
	}

	return map[string]any{
		"order_id":                  snapshot.Order.ID,
		"order_number":              snapshot.Order.OrderNumber,
		"invoice_count":             len(snapshot.InvoiceIDs),
		"invoice_item_count":        snapshot.InvoiceItemCount,
		"purchase_order_count":      len(snapshot.PurchaseOrderIDs),
		"purchase_order_item_count": snapshot.PurchaseItemCount,
		"delivery_note_count":       len(snapshot.DeliveryNoteIDs),
		"delivery_note_item_count":  snapshot.DeliveryItemCount,
		"order_item_count":          snapshot.OrderItemCount,
		"payment_count":             snapshot.PaymentCount,
		"blocked":                   blocked,
		"block_reason":              blockReason,
		"summary":                   summary,
	}
}

// PreviewOrderDeleteCascade tells the UI exactly which dependent records will be removed.
func (a *App) PreviewOrderDeleteCascade(orderID string) (map[string]any, error) {
	if err := a.requirePermission("orders:delete"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	snapshot, err := a.collectOrderDeleteCascade(a.db, orderID)
	if err != nil {
		return nil, err
	}
	return orderDeletePreviewMap(snapshot), nil
}

// DeleteOrder deletes an order and its dependent records after payment checks.
func (a *App) DeleteOrder(orderID string) error {
	if ok, err := a.guardDeleteOrRequest("orders:delete", "order", orderID, "Order"); !ok {
		return err
	}
	if err := a.requirePermission("orders:delete"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	snapshot, err := a.collectOrderDeleteCascade(a.db, orderID)
	if err != nil {
		return err
	}
	if snapshot.PaymentCount > 0 {
		return newError("PAYMENT_EXISTS",
			fmt.Sprintf("cannot delete order with %d payment(s) - use Cancel instead (audit requirement)", snapshot.PaymentCount),
			"")
	}

	err = a.db.Transaction(func(tx *gorm.DB) error {
		fresh, err := a.collectOrderDeleteCascade(tx, orderID)
		if err != nil {
			return err
		}
		if fresh.PaymentCount > 0 {
			return newError("PAYMENT_EXISTS",
				fmt.Sprintf("cannot delete order with %d payment(s) - use Cancel instead (audit requirement)", fresh.PaymentCount),
				"")
		}

		if len(fresh.InvoiceIDs) > 0 {
			if err := tx.Where("invoice_id IN ?", fresh.InvoiceIDs).Delete(&DBInvoiceItem{}).Error; err != nil {
				return fmt.Errorf("failed to delete invoice items: %w", err)
			}
			if err := tx.Where("id IN ?", fresh.InvoiceIDs).Delete(&Invoice{}).Error; err != nil {
				return fmt.Errorf("failed to delete invoices: %w", err)
			}
		}

		if len(fresh.PurchaseOrderIDs) > 0 {
			if err := tx.Where("purchase_order_id IN ?", fresh.PurchaseOrderIDs).Delete(&PurchaseOrderItem{}).Error; err != nil {
				return fmt.Errorf("failed to delete purchase order items: %w", err)
			}
			if err := tx.Where("id IN ?", fresh.PurchaseOrderIDs).Delete(&PurchaseOrder{}).Error; err != nil {
				return fmt.Errorf("failed to delete purchase orders: %w", err)
			}
		}

		if len(fresh.DeliveryNoteIDs) > 0 {
			if err := tx.Where("delivery_note_id IN ?", fresh.DeliveryNoteIDs).Delete(&DeliveryNoteItem{}).Error; err != nil {
				return fmt.Errorf("failed to delete delivery note items: %w", err)
			}
			if err := tx.Where("id IN ?", fresh.DeliveryNoteIDs).Delete(&DeliveryNote{}).Error; err != nil {
				return fmt.Errorf("failed to delete delivery notes: %w", err)
			}
		}

		if err := tx.Where("order_id = ?", fresh.Order.ID).Delete(&OrderItem{}).Error; err != nil {
			return fmt.Errorf("failed to delete order items: %w", err)
		}

		if err := tx.Delete(&Order{}, "id = ?", fresh.Order.ID).Error; err != nil {
			return fmt.Errorf("failed to delete order: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("cascade delete order failed: %w", err)
	}

	log.Printf("🗑️ Cascade deleted order %s and dependent records: %v", orderID, orderDeletePreviewMap(snapshot)["summary"])
	return nil
}

// UpdateOrderStage updates the stage of an order
// SECURITY: Validates state machine transitions to prevent invalid progressions
func (a *App) UpdateOrderStage(orderID string, stage string) error {
	if err := a.requirePermission("orders:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Valid order state transitions
	var validOrderTransitions = map[string][]string{
		"Draft":              {"Confirmed", "Cancelled"},
		"Confirmed":          {"Processing", "InProgress", "Cancelled"},
		"Processing":         {"PartiallyDelivered", "FullyDelivered", "Shipped", "Cancelled"},
		"InProgress":         {"PartiallyDelivered", "FullyDelivered", "Shipped", "Cancelled"},
		"Shipped":            {"PartiallyDelivered", "FullyDelivered", "Delivered"},
		"PartiallyDelivered": {"FullyDelivered", "Delivered"},
		"FullyDelivered":     {"Invoiced", "Complete"},
		"Delivered":          {"Invoiced", "Complete"},
		"Invoiced":           {"Complete"},
		"Complete":           {},
		"Cancelled":          {},
	}

	// Fetch current order to check current status
	var currentOrder Order
	if err := a.db.Where("id = ?", orderID).First(&currentOrder).Error; err != nil {
		return newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// Validate transition
	validNextStates, exists := validOrderTransitions[currentOrder.Status]
	if !exists {
		return newError("INVALID_STATE", fmt.Sprintf("Unknown current status: %s", currentOrder.Status), "")
	}

	isValidTransition := false
	for _, validState := range validNextStates {
		if validState == stage {
			isValidTransition = true
			break
		}
	}

	if !isValidTransition && currentOrder.Status != stage {
		return newError("INVALID_TRANSITION",
			fmt.Sprintf("Invalid order status transition from '%s' to '%s'", currentOrder.Status, stage),
			"")
	}

	// NOTE: Database field is "status", frontend uses "stage" terminology
	result := a.db.Model(&Order{}).Where("id = ?", orderID).Update("status", stage)
	if result.Error != nil {
		return newError("DB_UPDATE_FAILED", "Failed to update order stage", result.Error.Error())
	}

	log.Printf("✅ Updated Order #%s stage: %s → %s", orderID, currentOrder.Status, stage)
	return nil
}

// QuickMarkOrderDelivered marks an order as fully delivered in one click
// This is a simplified alternative to the full GRN workflow for cases where
// users just want to mark the order as delivered without detailed QC tracking
func (a *App) QuickMarkOrderDelivered(orderID string) (string, error) {
	if err := a.requirePermission("orders:update"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// 1. Get order details
	var order Order
	if err := a.db.Where("id = ?", orderID).First(&order).Error; err != nil {
		return "", newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// 2. Check if order is already delivered
	if order.Status == "FullyDelivered" {
		return "", newError("ALREADY_DELIVERED", "Order is already marked as fully delivered", "")
	}

	// 3. Check if order is cancelled
	if order.Status == "Cancelled" {
		return "", newError("ORDER_CANCELLED", "Cannot mark cancelled order as delivered", "")
	}

	// 4. Update order status to FullyDelivered
	result := a.db.Model(&Order{}).Where("id = ?", orderID).Update("status", "FullyDelivered")
	if result.Error != nil {
		return "", newError("DB_UPDATE_FAILED", "Failed to update order status", result.Error.Error())
	}

	message := fmt.Sprintf("Order %s marked as fully delivered", order.OrderNumber)
	log.Printf("✅ %s", message)

	// Emit event to notify frontend to refresh dashboard
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "data:refresh", map[string]any{
			"source":       "order-delivered",
			"order_id":     orderID,
			"order_number": order.OrderNumber,
		})
	}

	return message, nil
}

// CreateShipment creates shipments for multiple orders
func (a *App) CreateShipment(orderIds []string, trackingNumber string, courier string, estimatedDelivery string, notes string) error {
	if err := a.requirePermission("orders:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	if len(orderIds) == 0 {
		return newError("INVALID_INPUT", "At least one order ID is required", "")
	}

	// Parse estimated delivery date if provided. Wave 9.7 tight-ship fix: this
	// used to dereference shipmentDate unconditionally below, which panicked
	// whenever estimatedDelivery was empty or failed to parse (the frontend's
	// only caller, DeliveryTrackingScreen, sent an ISO string that never
	// matched the "2006-01-02" layout — that screen has since been retired as
	// orphaned, but this endpoint stays bound and must not panic regardless of
	// caller). Absent/unparseable input now leaves shipmentDate as the zero
	// time.Time instead of dereferencing a nil pointer.
	var shipmentDate time.Time
	if estimatedDelivery != "" {
		parsedDate, err := time.Parse("2006-01-02", estimatedDelivery)
		if err != nil {
			log.Printf("⚠ Invalid date format for estimatedDelivery: %s", estimatedDelivery)
		} else {
			shipmentDate = parsedDate
		}
	}

	// Create a shipment for each order
	for _, orderID := range orderIds {
		shipment := &Shipment{
			OrderID:        orderID,
			TrackingNumber: trackingNumber,
			CourierName:    courier,
			ShipmentDate:   shipmentDate,
			// Wave 9.7 tight-ship fix: "Packed" is not in the Shipment.Status
			// CHECK constraint (chk_shipments_status allows only Pending, In
			// Transit, Delivered, Failed, Cancelled) — every prior call to
			// CreateShipment was failing the DB constraint. "Pending" is the
			// correct initial state for a newly created shipment.
			Status: "Pending",
		}

		if err := a.db.Create(shipment).Error; err != nil {
			return newError("DB_CREATE_FAILED", fmt.Sprintf("Failed to create shipment for order #%s", orderID), err.Error())
		}

		// Update order status to indicate it's been shipped
		a.db.Model(&Order{}).Where("id = ?", orderID).Updates(map[string]any{
			"is_shipped": true,
			"status":     "Shipped",
		})

		log.Printf("✅ Created Shipment #%s for Order #%s (Tracking: %s, Courier: %s)",
			shipment.ID, orderID, trackingNumber, courier)
	}

	return nil
}

// ListShipments returns all shipments with order and customer info
func (a *App) ListShipments() ([]map[string]any, error) {
	if err := a.requirePermission("orders:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var shipments []Shipment
	if err := a.db.Order("created_at DESC").Find(&shipments).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to fetch shipments", err.Error())
	}

	result := make([]map[string]any, 0, len(shipments))
	for _, s := range shipments {
		orderNumber := ""
		customerName := ""

		if s.OrderID != "" {
			var order Order
			if err := a.db.First(&order, "id = ?", s.OrderID).Error; err == nil {
				orderNumber = order.OrderNumber
				customerName = order.CustomerName
			}
		}

		result = append(result, map[string]any{
			"id":                 s.ID,
			"order_id":           s.OrderID,
			"order_number":       orderNumber,
			"tracking_number":    s.TrackingNumber,
			"courier":            s.CourierName,
			"status":             s.Status,
			"estimated_delivery": s.ShipmentDate,
			"delivered_at":       s.DeliveredDate,
			"customer_name":      customerName,
			"created_at":         s.CreatedAt,
			"updated_at":         s.UpdatedAt,
		})
	}

	return result, nil
}

// UpdateShipment updates shipment status
func (a *App) UpdateShipment(shipmentID string, status, notes string) error {
	if err := a.requirePermission("orders:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	updates := map[string]any{"updated_at": time.Now()}
	if status != "" {
		updates["status"] = status
	}

	if err := a.db.Model(&Shipment{}).Where("id = ?", shipmentID).Updates(updates).Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to update shipment", err.Error())
	}

	log.Printf("✅ Updated Shipment #%s to status: %s", shipmentID, status)
	return nil
}

// ConfirmDelivery marks shipment as delivered
func (a *App) ConfirmDelivery(shipmentID string) error {
	if err := a.requirePermission("orders:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var shipment Shipment
	if err := a.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		return newError("SHIPMENT_NOT_FOUND", "Shipment not found", err.Error())
	}

	now := time.Now()
	shipment.Status = "Delivered"
	shipment.DeliveredDate = &now

	if err := a.db.Save(&shipment).Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to confirm delivery", err.Error())
	}

	// Update related order
	if shipment.OrderID != "" {
		var order Order
		if err := a.db.First(&order, "id = ?", shipment.OrderID).Error; err == nil {
			order.Status = "Delivered"
			a.db.Save(&order)
		}
	}

	log.Printf("✅ Confirmed delivery for Shipment #%s", shipmentID)
	return nil
}

// ============================================================================
// PARTIAL FULFILLMENT TRACKING (Phase 2)
// ============================================================================

// FulfillmentStatus represents the fulfillment status of an order
type FulfillmentStatus struct {
	OrderID          string            `json:"order_id"`
	TotalItems       int               `json:"total_items"`
	TotalQuantity    float64           `json:"total_quantity"`
	ShippedQuantity  float64           `json:"shipped_quantity"`
	InvoicedQuantity float64           `json:"invoiced_quantity"`
	FulfillmentPct   float64           `json:"fulfillment_pct"` // 0.0 - 1.0
	InvoicingPct     float64           `json:"invoicing_pct"`   // 0.0 - 1.0
	Status           string            `json:"status"`          // "Not Started", "Partial", "Fully Shipped", "Fully Invoiced"
	Items            []ItemFulfillment `json:"items"`
}

// ItemFulfillment represents fulfillment status for a single line item
type ItemFulfillment struct {
	ItemID           string  `json:"item_id"`
	LineNumber       int     `json:"line_number"`
	ProductCode      string  `json:"product_code"`
	Description      string  `json:"description"`
	Quantity         float64 `json:"quantity"`
	QuantityShipped  float64 `json:"quantity_shipped"`
	QuantityInvoiced float64 `json:"quantity_invoiced"`
	RemainingToShip  float64 `json:"remaining_to_ship"`
	ShippedPct       float64 `json:"shipped_pct"`
}

// GetOrderFulfillmentStatus calculates the fulfillment status for an order
func (a *App) GetOrderFulfillmentStatus(orderID string) (*FulfillmentStatus, error) {
	if err := a.requirePermission("orders:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get order with items
	var order Order
	if err := a.db.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
		return nil, newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	status := &FulfillmentStatus{
		OrderID:    orderID,
		TotalItems: len(order.Items),
		Items:      make([]ItemFulfillment, 0, len(order.Items)),
	}

	for _, item := range order.Items {
		status.TotalQuantity += item.Quantity
		status.ShippedQuantity += item.QuantityShipped
		status.InvoicedQuantity += item.QuantityInvoiced

		itemStatus := ItemFulfillment{
			ItemID:           item.ID,
			LineNumber:       item.LineNumber,
			ProductCode:      item.ProductCode,
			Description:      item.Description,
			Quantity:         item.Quantity,
			QuantityShipped:  item.QuantityShipped,
			QuantityInvoiced: item.QuantityInvoiced,
			RemainingToShip:  item.Quantity - item.QuantityShipped,
		}

		if item.Quantity > 0 {
			itemStatus.ShippedPct = item.QuantityShipped / item.Quantity
		}

		status.Items = append(status.Items, itemStatus)
	}

	// Calculate percentages
	if status.TotalQuantity > 0 {
		status.FulfillmentPct = status.ShippedQuantity / status.TotalQuantity
		status.InvoicingPct = status.InvoicedQuantity / status.TotalQuantity
	}

	// Determine status string
	if status.FulfillmentPct == 0 {
		status.Status = "Not Started"
	} else if status.FulfillmentPct < 1.0 {
		status.Status = "Partially Shipped"
	} else if status.InvoicingPct < 1.0 {
		status.Status = "Fully Shipped"
	} else {
		status.Status = "Fully Invoiced"
	}

	log.Printf("✅ Order #%s fulfillment: %.0f%% shipped, %.0f%% invoiced",
		orderID, status.FulfillmentPct*100, status.InvoicingPct*100)
	return status, nil
}

// UpdateOrderItemShipped updates the shipped quantity for an order item
func (a *App) UpdateOrderItemShipped(itemID uint, quantityShipped float64) error {
	if err := a.requirePermission("orders:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get the item
	var item OrderItem
	if err := a.db.First(&item, itemID).Error; err != nil {
		return newError("ITEM_NOT_FOUND", "Order item not found", err.Error())
	}

	// Validate: can't ship more than ordered
	if quantityShipped > item.Quantity {
		return newError("INVALID_QUANTITY",
			fmt.Sprintf("Cannot ship %.2f, only %.2f ordered", quantityShipped, item.Quantity), "")
	}

	// Update shipped quantity
	item.QuantityShipped = quantityShipped
	if err := a.db.Save(&item).Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to update item shipment", err.Error())
	}

	// Update order status based on fulfillment
	a.updateOrderFulfillmentStatus(item.OrderID)

	log.Printf("✅ Updated Item #%d shipped quantity to %.2f", itemID, quantityShipped)
	return nil
}

// UpdateOrderItemInvoiced updates the invoiced quantity for an order item
func (a *App) UpdateOrderItemInvoiced(itemID uint, quantityInvoiced float64) error {
	if err := a.requirePermission("orders:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get the item
	var item OrderItem
	if err := a.db.First(&item, itemID).Error; err != nil {
		return newError("ITEM_NOT_FOUND", "Order item not found", err.Error())
	}

	// Validate: can't invoice more than shipped
	if quantityInvoiced > item.QuantityShipped {
		return newError("INVALID_QUANTITY",
			fmt.Sprintf("Cannot invoice %.2f, only %.2f shipped", quantityInvoiced, item.QuantityShipped), "")
	}

	// Update invoiced quantity
	item.QuantityInvoiced = quantityInvoiced
	if err := a.db.Save(&item).Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to update item invoicing", err.Error())
	}

	// Update order status based on fulfillment
	a.updateOrderFulfillmentStatus(item.OrderID)

	log.Printf("✅ Updated Item #%d invoiced quantity to %.2f", itemID, quantityInvoiced)
	return nil
}

// CreateInvoiceFromOrder is now implemented in customer_invoice_service.go

// updateOrderFulfillmentStatus updates quantities on order items
func (a *App) updateOrderFulfillmentStatus(orderID string) {
	status, err := a.GetOrderFulfillmentStatus(orderID)
	if err != nil {
		log.Printf("⚠ Could not update order fulfillment status: %v", err)
		return
	}

	// Map fulfillment status to order status
	newStatus := ""
	switch status.Status {
	case "Not Started":
		newStatus = "PO_Received"
	case "Partially Shipped":
		newStatus = "In_Production" // Or a new "Partial_Shipment" status
	case "Fully Shipped":
		newStatus = "Shipped"
	case "Fully Invoiced":
		newStatus = "Delivered"
	}

	if newStatus != "" {
		a.db.Model(&Order{}).Where("id = ?", orderID).Update("status", newStatus)
		log.Printf("✅ Order #%s status updated to %s (fulfillment: %s)", orderID, newStatus, status.Status)
	}
}

// RecordPartialShipment records a partial shipment for multiple items
func (a *App) RecordPartialShipment(orderID string, shipments map[string]float64, trackingNumber, courier, notes string) error {
	if err := a.requirePermission("orders:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Start a transaction
	tx := a.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update each item's shipped quantity
	for itemID, qtyShipped := range shipments {
		var item OrderItem
		if err := tx.First(&item, "id = ?", itemID).Error; err != nil {
			tx.Rollback()
			return newError("ITEM_NOT_FOUND", fmt.Sprintf("Order item #%s not found", itemID), err.Error())
		}

		// Add to existing shipped quantity
		newShipped := item.QuantityShipped + qtyShipped
		if newShipped > item.Quantity {
			tx.Rollback()
			return newError("INVALID_QUANTITY",
				fmt.Sprintf("Cannot ship %.2f for item #%s, only %.2f remaining",
					qtyShipped, itemID, item.Quantity-item.QuantityShipped), "")
		}

		item.QuantityShipped = newShipped
		if err := tx.Save(&item).Error; err != nil {
			tx.Rollback()
			return newError("DB_UPDATE_FAILED", "Failed to update item shipment", err.Error())
		}
	}

	// Create shipment record
	now := time.Now()
	shipment := &Shipment{
		OrderID:        orderID,
		TrackingNumber: trackingNumber,
		CourierName:    courier,
		ShipmentDate:   now,
		Status:         "In Transit",
	}

	if err := tx.Create(shipment).Error; err != nil {
		tx.Rollback()
		return newError("DB_CREATE_FAILED", "Failed to create shipment record", err.Error())
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return newError("DB_COMMIT_FAILED", "Failed to commit partial shipment", err.Error())
	}

	// Update order status
	a.updateOrderFulfillmentStatus(orderID)

	log.Printf("✅ Recorded partial shipment for Order #%s: %d items, tracking: %s",
		orderID, len(shipments), trackingNumber)
	return nil
}

// FilterOrders retrieves orders with advanced filtering (WAVE 2 AGENT 4)
// Supports filtering by customer name/ID, date range, and status
func (a *App) FilterOrders(customerQuery string, dateFrom string, dateTo string, status string, limit int, offset int) ([]Order, error) {
	if err := a.requirePermission("orders:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Default limit: 100, max limit: 1000
	if limit <= 0 {
		limit = 100
	} else if limit > 1000 {
		limit = 1000
	}

	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	var orders []Order
	query := a.db.Model(&Order{})

	// Filter by customer (name OR ID, case-insensitive partial match)
	if customerQuery != "" {
		// SECURITY FIX: Escape LIKE wildcards to prevent LIKE injection
		escapedQuery := escapeLikeWildcards(customerQuery)
		query = query.Where("customer_name LIKE ? ESCAPE '\\' OR customer_id IN (SELECT id FROM customers WHERE customer_id LIKE ? ESCAPE '\\' OR business_name LIKE ? ESCAPE '\\')",
			"%"+escapedQuery+"%", "%"+escapedQuery+"%", "%"+escapedQuery+"%")
	}

	// Filter by date range
	if dateFrom != "" {
		fromDate, err := time.Parse("2006-01-02", dateFrom)
		if err == nil {
			query = query.Where("order_date >= ?", fromDate)
		} else {
			log.Printf("⚠ Invalid dateFrom format: %s (expected YYYY-MM-DD)", dateFrom)
		}
	}

	if dateTo != "" {
		toDate, err := time.Parse("2006-01-02", dateTo)
		if err == nil {
			// Add 1 day to include the entire end date
			query = query.Where("order_date < ?", toDate.AddDate(0, 0, 1))
		} else {
			log.Printf("⚠ Invalid dateTo format: %s (expected YYYY-MM-DD)", dateTo)
		}
	}

	// Filter by status
	if status != "" && status != "All" {
		query = query.Where("status = ?", status)
	}

	// Apply ordering, limit, offset
	query = query.Order("order_date DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	// Execute query with indexed fields (customer_id, order_date, status all indexed)
	if err := query.Find(&orders).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to filter orders", err.Error())
	}

	log.Printf("Filtered orders: %d results (customer=%s, dateFrom=%s, dateTo=%s, status=%s)",
		len(orders), customerQuery, dateFrom, dateTo, status)

	return orders, nil
}

// ============================================================================
// CUSTOMERS MANAGEMENT (Stubs for CustomersScreen.svelte)
// ============================================================================

// ListCustomers retrieves all customers with optional pagination
func (a *App) ListCustomers(limit int, offset int) ([]CustomerMaster, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Default limit: 100, max limit: 1000
	if limit <= 0 {
		limit = 100
	} else if limit > 1000 {
		limit = 1000
	}

	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	// P2 FIX: Check cache first
	cacheKey := fmt.Sprintf("%s:limit=%d:offset=%d", CacheKeyCustomerList, limit, offset)
	if a.cache != nil {
		if cached, ok := a.cache.Get(cacheKey); ok {
			return cached.([]CustomerMaster), nil
		}
	}

	var customers []CustomerMaster
	// P2 FIX: Use Select() to fetch only necessary columns for list view
	query := a.db.
		Select("id, customer_id, customer_code, business_name, city, country, payment_grade, customer_grade, total_orders_value, total_orders_count, outstanding_bhd, is_credit_blocked, created_at, updated_at").
		Order("business_name ASC")
	query = query.Limit(limit).Offset(offset)
	if err := query.Find(&customers).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to list customers", err.Error())
	}

	// P2 FIX: Cache result
	if a.cache != nil {
		a.cache.Set(cacheKey, customers, CacheTTLMedium)
	}

	log.Printf("Retrieved %d customers (limit=%d, offset=%d)", len(customers), limit, offset)
	return customers, nil
}

// GetCustomer retrieves a single customer by ID
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) GetCustomer(id string) (CustomerMaster, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return CustomerMaster{}, err
	}
	if a.db == nil {
		return CustomerMaster{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var customer CustomerMaster
	if err := a.db.First(&customer, "id = ? OR customer_id = ?", id, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return CustomerMaster{}, newError("NOT_FOUND", "Customer not found", fmt.Sprintf("ID: %s", id))
		}
		return CustomerMaster{}, newError("DB_QUERY_FAILED", "Failed to retrieve customer", err.Error())
	}

	return customer, nil
}

// SeedCustomerDatabase populates the database with sample customers if empty
func (a *App) SeedCustomerDatabase() error {
	// SECURITY: Admin-only permission (wildcard "*" or settings:update for managers)
	// Changed from settings:update to "*" for stricter admin-only access
	if err := a.requirePermission("*"); err != nil {
		return err
	}

	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Check if customers with ACTUAL data already exist (not empty records)
	var validCount int64
	if err := a.db.Model(&CustomerMaster{}).Where("business_name != '' AND business_name IS NOT NULL").Count(&validCount).Error; err != nil {
		return fmt.Errorf("failed to count customers: %w", err)
	}

	if validCount > 0 {
		log.Printf("📊 Customer database already has %d valid records, skipping seed", validCount)
		return nil
	}

	// Delete any empty/corrupted customer records first
	if err := a.db.Where("business_name = '' OR business_name IS NULL").Delete(&CustomerMaster{}).Error; err != nil {
		log.Printf("⚠ Failed to clean empty customer records: %v", err)
	}

	// Sample customers for Acme Instrumentation (Bahrain industrial equipment)
	sampleCustomers := []CustomerMaster{
		{
			CustomerID:         "NPC-001",
			CustomerType:       "EC", // End Customer
			BusinessName:       "National Petroleum Co.",
			ShortCode:          "NPC",
			City:               "Manama",
			Country:            "Bahrain",
			Industry:           "Oil & Gas",
			RelationYears:      15,
			PaymentGrade:       "A",
			PaymentTermsDays:   30,
			AvgPaymentDays:     25.5,
			IsCreditBlocked:    false,
			RequiresPrepayment: false,
			HasABBCompetition:  false,
			IsEmergencyOnly:    false,
		},
		{
			CustomerID:         "GSC-001",
			CustomerType:       "EC", // End Customer
			BusinessName:       "Gulf Smelting Co.",
			ShortCode:          "GSC",
			City:               "Manama",
			Country:            "Bahrain",
			Industry:           "Metals & Mining",
			RelationYears:      12,
			PaymentGrade:       "A",
			PaymentTermsDays:   45,
			AvgPaymentDays:     40.0,
			IsCreditBlocked:    false,
			RequiresPrepayment: false,
			HasABBCompetition:  false,
			IsEmergencyOnly:    false,
		},
		{
			CustomerID:         "DPC-001",
			CustomerType:       "EC", // End Customer
			BusinessName:       "Delta Petrochemicals",
			ShortCode:          "DPC",
			City:               "Sitra",
			Country:            "Bahrain",
			Industry:           "Petrochemicals",
			RelationYears:      10,
			PaymentGrade:       "B",
			PaymentTermsDays:   60,
			AvgPaymentDays:     55.0,
			IsCreditBlocked:    false,
			RequiresPrepayment: false,
			HasABBCompetition:  false,
			IsEmergencyOnly:    false,
		},
		{
			CustomerID:         "SUMMIT-001",
			CustomerType:       "EP", // Engineering/EPC
			BusinessName:       "Summit Energy",
			ShortCode:          "SUMMIT",
			City:               "Manama",
			Country:            "Bahrain",
			Industry:           "Oil & Gas",
			RelationYears:      8,
			PaymentGrade:       "A",
			PaymentTermsDays:   30,
			AvgPaymentDays:     28.0,
			IsCreditBlocked:    false,
			RequiresPrepayment: false,
			HasABBCompetition:  false,
			IsEmergencyOnly:    false,
		},
		{
			CustomerID:         "HELIX-BH-001",
			CustomerType:       "SI", // System Integrator
			BusinessName:       "Helix Automation (Bahrain)",
			ShortCode:          "HELIX",
			City:               "Manama",
			Country:            "Bahrain",
			Industry:           "Industrial Automation",
			RelationYears:      5,
			PaymentGrade:       "A",
			PaymentTermsDays:   45,
			AvgPaymentDays:     42.0,
			IsCreditBlocked:    false,
			RequiresPrepayment: false,
			HasABBCompetition:  false,
			IsEmergencyOnly:    false,
		},
	}

	// Insert sample customers
	for _, customer := range sampleCustomers {
		if err := a.db.Create(&customer).Error; err != nil {
			log.Printf("⚠ Failed to seed customer %s: %v", customer.BusinessName, err)
		} else {
			log.Printf("  ✓ Seeded: %s (%s)", customer.BusinessName, customer.CustomerID)
		}
	}

	log.Printf("🌱 Seeded %d sample customers", len(sampleCustomers))
	return nil
}

// CreateCustomer creates a new customer
func (a *App) CreateCustomer(customer CustomerMaster) (*CustomerMaster, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// SERVER-SIDE PERMISSION CHECK: Require customers:create or admin (wildcard)
	if err := a.requirePermission("customers:create"); err != nil {
		log.Printf("🔒 CreateCustomer blocked: %v", err)
		return nil, newError("PERMISSION_DENIED", err.Error(), "")
	}

	// P1 FIX: Input validation
	if GlobalValidator != nil {
		if err := GlobalValidator.ValidateCustomerInput(&customer); err != nil {
			return nil, newError("VALIDATION_FAILED", "Invalid customer data", err.Error())
		}
	}

	// Identity policy (Band-2): required name, bidirectional Code↔ID fill,
	// default code generation — shared with every other identity-write path.
	if err := crmcustomer.PrepareCustomerCreate(&customer, time.Now()); err != nil {
		return nil, err
	}

	if err := a.db.Create(&customer).Error; err != nil {
		return nil, newError("DB_CREATE_FAILED", "Failed to create customer", err.Error())
	}

	log.Printf("✅ Created Customer: %s", customer.BusinessName)
	return &customer, nil
}

func looksLikeUUID(raw string) bool {
	return regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(strings.TrimSpace(raw))
}

func businessCustomerIDFromRecord(customer CustomerMaster) string {
	// Band-2: delegates to the engine repair rule, which additionally discards
	// UUID-shaped legacy codes instead of blessing them as business IDs.
	return crmcustomer.RepairCustomerBusinessID(customer)
}

// BackfillBusinessCustomerIDs is the bound entry point. Mission I (I-11):
// gated — startup uses backfillBusinessCustomerIDsInternal.
func (a *App) BackfillBusinessCustomerIDs() (map[string]any, error) {
	if err := a.requirePermission("customers:update"); err != nil {
		return nil, err
	}
	return a.backfillBusinessCustomerIDsInternal()
}

func (a *App) backfillBusinessCustomerIDsInternal() (map[string]any, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var customers []CustomerMaster
	if err := a.db.
		Where("deleted_at IS NULL AND (customer_id = id OR customer_id = '' OR customer_id IS NULL)").
		Find(&customers).Error; err != nil {
		return nil, fmt.Errorf("failed to load customers needing business ids: %w", err)
	}

	updated := 0
	examples := make([]string, 0, len(customers))
	for _, customer := range customers {
		nextID := businessCustomerIDFromRecord(customer)
		nextCode := crmcustomer.NormalizeBusinessIdentifier(customer.CustomerCode, customer.ID)
		if nextCode == "" {
			nextCode = nextID
		}

		if err := a.db.Model(&CustomerMaster{}).
			Where("id = ?", customer.ID).
			Updates(map[string]any{
				"customer_id":   nextID,
				"customer_code": nextCode,
			}).Error; err != nil {
			return nil, fmt.Errorf("failed to backfill customer id for %s: %w", customer.BusinessName, err)
		}
		updated++
		if len(examples) < 10 {
			examples = append(examples, fmt.Sprintf("%s→%s", customer.BusinessName, nextID))
		}
	}

	return map[string]any{
		"checked":  len(customers),
		"updated":  updated,
		"examples": examples,
	}, nil
}

// ============================================================================
// CUSTOMER CONTACTS MANAGEMENT
// ============================================================================

// ListCustomerContacts returns all contacts for a customer
func (a *App) ListCustomerContacts(customerID string) ([]CustomerContact, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	var contacts []CustomerContact
	if err := a.db.Where("customer_id = ?", customerID).Order("is_primary_contact DESC, contact_name ASC").Find(&contacts).Error; err != nil {
		return nil, fmt.Errorf("failed to list contacts: %w", err)
	}
	return contacts, nil
}

// AddCustomerContact adds a new contact to a customer
func (a *App) AddCustomerContact(contact CustomerContact) (*CustomerContact, error) {
	if err := a.requirePermission("customers:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	if err := a.db.Create(&contact).Error; err != nil {
		return nil, fmt.Errorf("failed to add contact: %w", err)
	}
	return &contact, nil
}

// UpdateCustomerContact updates an existing customer contact
func (a *App) UpdateCustomerContact(contact CustomerContact) (*CustomerContact, error) {
	if err := a.requirePermission("customers:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	// INT-001: load-then-overlay. The previous bare Save wrote the whole inbound
	// struct, so a partial payload could zero the CustomerID FK (orphaning the
	// contact) or the audit metadata. Preserve server-owned fields; every other
	// CustomerContact field is user-editable.
	var existing CustomerContact
	if err := a.db.First(&existing, "id = ?", contact.ID).Error; err != nil {
		return nil, newError("CONTACT_NOT_FOUND", "Customer contact not found", err.Error())
	}
	if strings.TrimSpace(contact.CustomerID) == "" {
		contact.CustomerID = existing.CustomerID
	}
	contact.CreatedAt = existing.CreatedAt
	contact.CreatedBy = existing.CreatedBy
	contact.DeletedAt = existing.DeletedAt
	contact.Version = existing.Version
	if err := a.db.Save(&contact).Error; err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}
	return &contact, nil
}

// DeleteCustomerContact removes a contact
func (a *App) DeleteCustomerContact(contactID string) error {
	if ok, err := a.guardDeleteOrRequest("customers:delete", "customer_contact", contactID, "Customer contact"); !ok {
		return err
	}
	if err := a.requirePermission("customers:delete"); err != nil {
		return err
	}
	return crmcustomer.DeleteCustomerContact(a.db, contactID)
}

// ============================================================================
// SUPPLIER CONTACTS MANAGEMENT
// ============================================================================

// ListSupplierContacts returns all contacts for a supplier
func (a *App) ListSupplierContacts(supplierID string) ([]SupplierContact, error) {
	if err := a.requirePermission("suppliers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	var contacts []SupplierContact
	if err := a.db.Where("supplier_id = ?", supplierID).Order("is_primary_contact DESC, contact_name ASC").Find(&contacts).Error; err != nil {
		return nil, fmt.Errorf("failed to list contacts: %w", err)
	}
	return contacts, nil
}

// AddSupplierContact adds a new contact to a supplier
func (a *App) AddSupplierContact(contact SupplierContact) (*SupplierContact, error) {
	if err := a.requirePermission("suppliers:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	if err := a.db.Create(&contact).Error; err != nil {
		return nil, fmt.Errorf("failed to add contact: %w", err)
	}
	return &contact, nil
}

// UpdateSupplierContact updates an existing supplier contact
func (a *App) UpdateSupplierContact(contact SupplierContact) (*SupplierContact, error) {
	if err := a.requirePermission("suppliers:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	// INT-001: load-then-overlay, mirroring UpdateCustomerContact — a partial
	// payload must not zero the SupplierID FK or audit metadata.
	var existing SupplierContact
	if err := a.db.First(&existing, "id = ?", contact.ID).Error; err != nil {
		return nil, newError("CONTACT_NOT_FOUND", "Supplier contact not found", err.Error())
	}
	if strings.TrimSpace(contact.SupplierID) == "" {
		contact.SupplierID = existing.SupplierID
	}
	contact.CreatedAt = existing.CreatedAt
	contact.CreatedBy = existing.CreatedBy
	contact.DeletedAt = existing.DeletedAt
	contact.Version = existing.Version
	if err := a.db.Save(&contact).Error; err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}
	return &contact, nil
}

// DeleteSupplierContact removes a contact
func (a *App) DeleteSupplierContact(contactID string) error {
	if ok, err := a.guardDeleteOrRequest("suppliers:delete", "supplier_contact", contactID, "Supplier contact"); !ok {
		return err
	}
	if err := a.requirePermission("suppliers:delete"); err != nil {
		return err
	}
	return crmcustomer.DeleteSupplierContact(a.db, contactID)
}

// ============================================================================
// SUPPLIERS MANAGEMENT (SupplierListScreen.svelte)
// ============================================================================

// ListSuppliers retrieves all suppliers with OEM rebate tracking and optional pagination
func (a *App) ListSuppliers(limit int, offset int) ([]SupplierMaster, error) {
	if err := a.requirePermission("suppliers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Default limit: 100, max limit: 1000
	if limit <= 0 {
		limit = 100
	} else if limit > 1000 {
		limit = 1000
	}

	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	// P2 FIX: Check cache first
	cacheKey := fmt.Sprintf("%s:limit=%d:offset=%d", CacheKeySupplierList, limit, offset)
	if a.cache != nil {
		if cached, ok := a.cache.Get(cacheKey); ok {
			return cached.([]SupplierMaster), nil
		}
	}

	var suppliers []SupplierMaster
	// P2 FIX: Use Select() to fetch only necessary columns for list view
	query := a.db.
		Select("id, supplier_code, supplier_name, country, lead_time_days, supplier_type, rating, payment_terms, primary_contact, email, phone, created_at, updated_at").
		Order("supplier_name ASC")
	query = query.Limit(limit).Offset(offset)
	if err := query.Find(&suppliers).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to list suppliers", err.Error())
	}

	// P2 FIX: Cache result
	if a.cache != nil {
		a.cache.Set(cacheKey, suppliers, CacheTTLMedium)
	}

	log.Printf("📦 Retrieved %d suppliers (limit=%d, offset=%d)", len(suppliers), limit, offset)
	return suppliers, nil
}

// GetSupplier retrieves a single supplier by ID
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) GetSupplier(id string) (SupplierMaster, error) {
	if err := a.requirePermission("suppliers:view"); err != nil {
		return SupplierMaster{}, err
	}
	if a.db == nil {
		return SupplierMaster{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var supplier SupplierMaster
	if err := a.db.First(&supplier, "id = ? OR supplier_code = ?", id, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return SupplierMaster{}, newError("NOT_FOUND", "Supplier not found", fmt.Sprintf("ID: %s", id))
		}
		return SupplierMaster{}, newError("DB_QUERY_FAILED", "Failed to retrieve supplier", err.Error())
	}

	return supplier, nil
}

// CreateSupplier creates a new supplier in the database
func (a *App) CreateSupplier(supplier SupplierMaster) (*SupplierMaster, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// SERVER-SIDE PERMISSION CHECK: Require suppliers:create or admin (wildcard)
	if err := a.requirePermission("suppliers:create"); err != nil {
		log.Printf("🔒 CreateSupplier blocked: %v", err)
		return nil, newError("PERMISSION_DENIED", err.Error(), "")
	}

	// P1 FIX: Input validation
	if GlobalValidator != nil {
		if err := GlobalValidator.ValidateSupplierInput(&supplier); err != nil {
			return nil, newError("VALIDATION_FAILED", "Invalid supplier data", err.Error())
		}
	}

	// Identity policy (Band-2): required name, code generation, default rating.
	if err := crmcustomer.PrepareSupplierCreate(&supplier, time.Now()); err != nil {
		return nil, err
	}

	if err := a.db.Create(&supplier).Error; err != nil {
		return nil, newError("DB_CREATE_FAILED", "Failed to create supplier", err.Error())
	}

	log.Printf("✅ Created Supplier: %s", supplier.SupplierName)
	return &supplier, nil
}

// UpdateCustomer updates an existing customer in the database
func (a *App) UpdateCustomer(customer CustomerMaster) (*CustomerMaster, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// SERVER-SIDE PERMISSION CHECK: Require customers:edit or admin (wildcard)
	if err := a.requirePermission("customers:edit"); err != nil {
		log.Printf("🔒 UpdateCustomer blocked: %v", err)
		return nil, newError("PERMISSION_DENIED", err.Error(), "")
	}

	// Band-2 (PH G1): load the existing row and MERGE only the user-editable
	// fields onto it. The previous a.db.Save(&customer) wrote EVERY column, so
	// any field the caller omitted (zero-valued) silently clobbered the stored
	// value — wiping server-owned metrics (order totals, AR risk, outstanding,
	// computed grade) on partial payloads. The merge keeps server-owned columns
	// intact; blanking an editable field remains a legitimate edit.
	var existing CustomerMaster
	if err := a.db.First(&existing, "id = ?", customer.ID).Error; err != nil {
		return nil, newError("CUSTOMER_NOT_FOUND", "Customer not found", err.Error())
	}

	crmcustomer.MergeCustomerUpdate(&existing, customer, time.Now())

	// Validate the merged record (mirrors CreateCustomer).
	if GlobalValidator != nil {
		if err := GlobalValidator.ValidateCustomerInput(&existing); err != nil {
			return nil, newError("VALIDATION_FAILED", "Invalid customer data", err.Error())
		}
	}

	if err := a.db.Save(&existing).Error; err != nil {
		return nil, newError("DB_UPDATE_FAILED", "Failed to update customer", err.Error())
	}

	log.Printf("✅ Customer updated (merge): %s (%s) v%d", existing.BusinessName, existing.ID, existing.Version)
	return &existing, nil
}

// UpdateSupplier updates an existing supplier in the database
func (a *App) UpdateSupplier(supplier SupplierMaster) (*SupplierMaster, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// SERVER-SIDE PERMISSION CHECK: Require suppliers:edit or admin (wildcard)
	if err := a.requirePermission("suppliers:edit"); err != nil {
		log.Printf("🔒 UpdateSupplier blocked: %v", err)
		return nil, newError("PERMISSION_DENIED", err.Error(), "")
	}

	// Band-2 (PH G1): merge only user-editable fields onto the existing row
	// (same non-destructive pattern as UpdateCustomer); Rating 0 means "not
	// provided" and falls back instead of wiping a real rating.
	var existing SupplierMaster
	if err := a.db.First(&existing, "id = ?", supplier.ID).Error; err != nil {
		return nil, newError("SUPPLIER_NOT_FOUND", "Supplier not found", err.Error())
	}

	crmcustomer.MergeSupplierUpdate(&existing, supplier, time.Now())

	if GlobalValidator != nil {
		if err := GlobalValidator.ValidateSupplierInput(&existing); err != nil {
			return nil, newError("VALIDATION_FAILED", "Invalid supplier data", err.Error())
		}
	}

	if err := a.db.Save(&existing).Error; err != nil {
		return nil, newError("DB_UPDATE_FAILED", "Failed to update supplier", err.Error())
	}

	log.Printf("✅ Supplier updated (merge): %s (%s) v%d", existing.SupplierName, existing.ID, existing.Version)
	return &existing, nil
}

// DeleteCustomer soft deletes a customer from the database
func (a *App) DeleteCustomer(id string) error {
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	if ok, err := a.guardDeleteOrRequest("customers:delete", "customer", id, "Customer"); !ok {
		return err
	}

	// SERVER-SIDE PERMISSION CHECK: Require customers:delete or admin (wildcard)
	if err := a.requirePermission("customers:delete"); err != nil {
		log.Printf("🔒 DeleteCustomer blocked: %v", err)
		return newError("PERMISSION_DENIED", err.Error(), "")
	}
	return crmcustomer.DeleteCustomer(a.db, id)
}

// DeleteSupplier soft deletes a supplier from the database
func (a *App) DeleteSupplier(id string) error {
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	if ok, err := a.guardDeleteOrRequest("suppliers:delete", "supplier", id, "Supplier"); !ok {
		return err
	}

	// SERVER-SIDE PERMISSION CHECK: Require suppliers:delete or admin (wildcard)
	if err := a.requirePermission("suppliers:delete"); err != nil {
		log.Printf("🔒 DeleteSupplier blocked: %v", err)
		return newError("PERMISSION_DENIED", err.Error(), "")
	}
	return crmcustomer.DeleteSupplier(a.db, id)
}

// UpdateSupplierGoals updates supplier rebate targets
func (a *App) UpdateSupplierGoals(supplierID uint, targetAmount float64, currentAmount float64) error {
	if err := a.requirePermission("suppliers:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Update supplier record with target/current amounts
	// NOTE: SupplierMaster struct would need new fields: TargetRebateAmount, CurrentRebateAmount
	// For now, we'll add them dynamically via map update
	updates := map[string]any{
		"target_rebate_amount":  targetAmount,
		"current_rebate_amount": currentAmount,
	}

	result := a.db.Model(&SupplierMaster{}).Where("id = ?", supplierID).Updates(updates)
	if result.Error != nil {
		return newError("DB_UPDATE_FAILED", "Failed to update supplier goals", result.Error.Error())
	}

	log.Printf("🎯 Updated Supplier #%d goals: Target=%.2f, Current=%.2f", supplierID, targetAmount, currentAmount)
	return nil
}

// ============================================================================
// CRM SUPPLIER DASHBOARD - MCKINSEY-STYLE COMMAND CENTER
// ============================================================================

// CRMSupplierDashboard represents McKinsey-style supplier command center metrics
type CRMSupplierDashboard struct {
	TotalSuppliers      int                  `json:"total_suppliers"`
	ActiveSuppliers     int                  `json:"active_suppliers"`
	TotalPurchases      float64              `json:"total_purchases"` // YTD
	OutstandingPayables float64              `json:"outstanding_payables"`
	OverduePayables     float64              `json:"overdue_payables"`
	TopSuppliers        []SupplierMetricCard `json:"top_suppliers"`
	Suppliers           []SupplierMetricCard `json:"suppliers"` // All suppliers for cards
}

type SupplierMetricCard struct {
	ID             string  `json:"id"`
	SupplierName   string  `json:"supplier_name"`
	SupplierType   string  `json:"supplier_type"`
	Rating         int     `json:"rating"`
	TotalPurchases float64 `json:"total_purchases"`
	ActivePOs      int     `json:"active_pos"`
	OutstandingBHD float64 `json:"outstanding_bhd"`
	OverdueBHD     float64 `json:"overdue_bhd"`
	BrandsHandled  string  `json:"brands_handled"`
	Country        string  `json:"country"`
}

// GetCRMSupplierDashboard returns McKinsey-style supplier command center metrics
func (a *App) GetCRMSupplierDashboard() CRMSupplierDashboard {
	if err := a.requirePermission("suppliers:view"); err != nil {
		return CRMSupplierDashboard{}
	}
	dashboard := CRMSupplierDashboard{}

	if a.db == nil {
		log.Printf("⚠️ GetCRMSupplierDashboard: Database not initialized")
		return dashboard
	}

	// 1. Get all suppliers
	var suppliers []SupplierMaster
	a.db.Find(&suppliers)
	dashboard.TotalSuppliers = len(suppliers)

	// 2. Detect the latest year with data (handles future system dates)
	var latestYear int
	row := a.db.Raw("SELECT MAX(CAST(strftime('%Y', invoice_date) AS INTEGER)) FROM supplier_invoices WHERE deleted_at IS NULL").Row()
	row.Scan(&latestYear)
	if latestYear == 0 {
		latestYear = time.Now().Year()
	}

	ytdStart := time.Date(latestYear, 1, 1, 0, 0, 0, 0, time.UTC)
	activeStart := time.Now().AddDate(-1, 0, 0)

	// 3. Get supplier invoices for YTD purchases (this is where actual purchase data lives)
	var allSupplierInvoices []SupplierInvoice
	a.db.Where("invoice_date >= ? AND deleted_at IS NULL", ytdStart).Find(&allSupplierInvoices)

	// Also get purchase orders for active PO count
	var pos []PurchaseOrder
	a.db.Where("po_date >= ? AND deleted_at IS NULL", ytdStart).Find(&pos)

	// Build supplier metrics maps - use supplier_name as key since that's populated
	supplierPurchasesByName := make(map[string]float64)
	supplierPurchasesByID := make(map[string]float64)
	supplierActivePOs := make(map[string]int)
	var totalPurchases float64

	// Calculate purchases from supplier invoices (actual data)
	for _, inv := range allSupplierInvoices {
		if inv.SupplierName != "" {
			supplierPurchasesByName[inv.SupplierName] += inv.TotalBHD
		}
		if inv.SupplierID != "" {
			supplierPurchasesByID[inv.SupplierID] += inv.TotalBHD
		}
		totalPurchases += inv.TotalBHD
	}

	// Count active POs per supplier
	for _, po := range pos {
		if po.Status != "Closed" && po.Status != "Received" {
			supplierActivePOs[po.SupplierID]++
		}
	}
	dashboard.TotalPurchases = totalPurchases

	// 4. Get supplier invoices for outstanding calculations (use payment_status field)
	var supplierInvoices []SupplierInvoice
	a.db.Where("payment_status NOT IN ('Paid', 'Cancelled')").Find(&supplierInvoices)

	supplierOutstanding := make(map[string]float64)
	supplierOverdue := make(map[string]float64)
	var totalOutstanding, totalOverdue float64

	for _, inv := range supplierInvoices {
		supplierOutstanding[inv.SupplierID] += inv.TotalBHD
		totalOutstanding += inv.TotalBHD

		if time.Since(inv.DueDate).Hours()/24 > 30 {
			supplierOverdue[inv.SupplierID] += inv.TotalBHD
			totalOverdue += inv.TotalBHD
		}
	}
	dashboard.OutstandingPayables = totalOutstanding
	dashboard.OverduePayables = totalOverdue

	// 5. Calculate active suppliers (had POs in last 12 months)
	var recentPOs []PurchaseOrder
	a.db.Where("po_date >= ?", activeStart).Select("DISTINCT supplier_id").Find(&recentPOs)
	activeSupplierSet := make(map[string]bool)
	for _, po := range recentPOs {
		activeSupplierSet[po.SupplierID] = true
	}
	dashboard.ActiveSuppliers = len(activeSupplierSet)

	// 6. Build supplier cards - use both ID and Name for lookups
	cards := make([]SupplierMetricCard, 0, len(suppliers))
	for _, s := range suppliers {
		// Try to get purchases by ID first, then by name
		purchases := supplierPurchasesByID[s.ID]
		if purchases == 0 {
			purchases = supplierPurchasesByName[s.SupplierName]
		}
		cards = append(cards, SupplierMetricCard{
			ID:             s.ID,
			SupplierName:   s.SupplierName,
			SupplierType:   s.SupplierType,
			Rating:         s.Rating,
			TotalPurchases: purchases,
			ActivePOs:      supplierActivePOs[s.ID],
			OutstandingBHD: supplierOutstanding[s.ID],
			OverdueBHD:     supplierOverdue[s.ID],
			BrandsHandled:  s.BrandsHandled,
			Country:        s.Country,
		})
	}

	// 7. Sort by total purchases descending
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].TotalPurchases > cards[j].TotalPurchases
	})
	dashboard.Suppliers = cards

	// 8. Top suppliers
	if len(cards) > 10 {
		dashboard.TopSuppliers = cards[:10]
	} else {
		dashboard.TopSuppliers = cards
	}

	log.Printf("📦 CRM Supplier Dashboard: %d suppliers, %.0f BHD purchases, %.0f BHD payables",
		dashboard.TotalSuppliers, dashboard.TotalPurchases, dashboard.OutstandingPayables)

	return dashboard
}

// GetCRMSupplierDashboardByYear returns the supplier CRM dashboard filtered to a specific year
// When year is 0, falls back to GetCRMSupplierDashboard (auto-detect latest year)
func (a *App) GetCRMSupplierDashboardByYear(year int) CRMSupplierDashboard {
	if year == 0 {
		return a.GetCRMSupplierDashboard()
	}
	if err := a.requirePermission("suppliers:view"); err != nil {
		return CRMSupplierDashboard{}
	}
	// Validate year bounds
	if year < 2020 || year > time.Now().Year()+1 {
		return CRMSupplierDashboard{}
	}
	dashboard := CRMSupplierDashboard{}

	if a.db == nil {
		log.Printf("⚠️ GetCRMSupplierDashboardByYear: Database not initialized")
		return dashboard
	}

	// 1. Get all suppliers
	var suppliers []SupplierMaster
	a.db.Find(&suppliers)
	dashboard.TotalSuppliers = len(suppliers)

	log.Printf("📦 CRM Supplier Dashboard (year=%d)", year)

	ytdStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	ytdEnd := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	activeStart := time.Date(year-1, 1, 1, 0, 0, 0, 0, time.UTC)

	// 3. Get supplier invoices for specified year
	var allSupplierInvoices []SupplierInvoice
	a.db.Where("invoice_date >= ? AND invoice_date < ? AND deleted_at IS NULL", ytdStart, ytdEnd).Find(&allSupplierInvoices)

	// Also get purchase orders for active PO count
	var pos []PurchaseOrder
	a.db.Where("po_date >= ? AND po_date < ? AND deleted_at IS NULL", ytdStart, ytdEnd).Find(&pos)

	// Build supplier metrics maps
	supplierPurchasesByName := make(map[string]float64)
	supplierPurchasesByID := make(map[string]float64)
	supplierActivePOs := make(map[string]int)
	var totalPurchases float64

	for _, inv := range allSupplierInvoices {
		if inv.SupplierName != "" {
			supplierPurchasesByName[inv.SupplierName] += inv.TotalBHD
		}
		if inv.SupplierID != "" {
			supplierPurchasesByID[inv.SupplierID] += inv.TotalBHD
		}
		totalPurchases += inv.TotalBHD
	}

	for _, po := range pos {
		if po.Status != "Closed" && po.Status != "Received" {
			supplierActivePOs[po.SupplierID]++
		}
	}
	dashboard.TotalPurchases = totalPurchases

	// 4. Get supplier invoices for outstanding within the year
	var supplierInvoices []SupplierInvoice
	a.db.Where("invoice_date >= ? AND invoice_date < ? AND payment_status NOT IN ('Paid', 'Cancelled')", ytdStart, ytdEnd).Find(&supplierInvoices)

	supplierOutstanding := make(map[string]float64)
	supplierOverdue := make(map[string]float64)
	var totalOutstanding, totalOverdue float64

	for _, inv := range supplierInvoices {
		supplierOutstanding[inv.SupplierID] += inv.TotalBHD
		totalOutstanding += inv.TotalBHD

		if time.Since(inv.DueDate).Hours()/24 > 30 {
			supplierOverdue[inv.SupplierID] += inv.TotalBHD
			totalOverdue += inv.TotalBHD
		}
	}
	dashboard.OutstandingPayables = totalOutstanding
	dashboard.OverduePayables = totalOverdue

	// 5. Calculate active suppliers (had POs in the year range)
	var recentPOs []PurchaseOrder
	a.db.Where("po_date >= ? AND po_date < ?", activeStart, ytdEnd).Select("DISTINCT supplier_id").Find(&recentPOs)
	activeSupplierSet := make(map[string]bool)
	for _, po := range recentPOs {
		activeSupplierSet[po.SupplierID] = true
	}
	dashboard.ActiveSuppliers = len(activeSupplierSet)

	// 6. Build supplier cards
	cards := make([]SupplierMetricCard, 0, len(suppliers))
	for _, s := range suppliers {
		purchases := supplierPurchasesByID[s.ID]
		if purchases == 0 {
			purchases = supplierPurchasesByName[s.SupplierName]
		}
		cards = append(cards, SupplierMetricCard{
			ID:             s.ID,
			SupplierName:   s.SupplierName,
			SupplierType:   s.SupplierType,
			Rating:         s.Rating,
			TotalPurchases: purchases,
			ActivePOs:      supplierActivePOs[s.ID],
			OutstandingBHD: supplierOutstanding[s.ID],
			OverdueBHD:     supplierOverdue[s.ID],
			BrandsHandled:  s.BrandsHandled,
			Country:        s.Country,
		})
	}

	// 7. Sort by total purchases descending
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].TotalPurchases > cards[j].TotalPurchases
	})
	dashboard.Suppliers = cards

	// 8. Top suppliers
	if len(cards) > 10 {
		dashboard.TopSuppliers = cards[:10]
	} else {
		dashboard.TopSuppliers = cards
	}

	log.Printf("📦 CRM Supplier Dashboard (year=%d): %d suppliers, %.0f BHD purchases, %.0f BHD payables",
		year, dashboard.TotalSuppliers, dashboard.TotalPurchases, dashboard.OutstandingPayables)

	return dashboard
}

// ============================================================================
// CUSTOMER 360 GRAPH - RELATIONSHIP VISUALIZATION
// ============================================================================

// GraphEntity represents a node in the Customer 360 graph
type GraphEntity struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"` // "customer", "order", "invoice", "contact", "product"
	Label    string         `json:"label"`
	Data     map[string]any `json:"data"`
	Position *GraphPosition `json:"position,omitempty"`
}

// GraphRelation represents an edge in the Customer 360 graph
type GraphRelation struct {
	ID     string  `json:"id"`
	Source string  `json:"source"` // Entity ID
	Target string  `json:"target"` // Entity ID
	Type   string  `json:"type"`   // "ordered", "paid", "contacted", "purchased"
	Label  string  `json:"label"`
	Weight float64 `json:"weight,omitempty"` // Strength of relationship
}

// GraphPosition represents x/y coordinates for graph layout
type GraphPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GraphMetrics provides graph statistics
type GraphMetrics struct {
	TotalNodes         int     `json:"total_nodes"`
	TotalEdges         int     `json:"total_edges"`
	AverageConnections float64 `json:"average_connections"`
	GraphDensity       float64 `json:"graph_density"`
}

// Customer360Graph represents the complete relationship graph for a customer
type Customer360Graph struct {
	CustomerID string          `json:"customer_id"`
	Entities   []GraphEntity   `json:"entities"`
	Relations  []GraphRelation `json:"relations"`
	Metrics    GraphMetrics    `json:"metrics"`
}

// GetCustomer360Graph builds a relationship graph for a customer
func (a *App) GetCustomer360Graph(customerID string) (Customer360Graph, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return Customer360Graph{}, err
	}
	if a.db == nil {
		return Customer360Graph{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var customer CustomerMaster
	if err := a.db.Where("customer_id = ?", customerID).First(&customer).Error; err != nil {
		return Customer360Graph{}, newError("CUSTOMER_NOT_FOUND", "Customer not found", err.Error())
	}

	entities := []GraphEntity{}
	relations := []GraphRelation{}

	// 1. Add customer node (center)
	customerNode := GraphEntity{
		ID:    fmt.Sprintf("customer-%s", customer.ID),
		Type:  "customer",
		Label: customer.BusinessName,
		Data: map[string]any{
			"grade":          customer.PaymentGrade,
			"total_value":    customer.TotalOrdersValue,
			"relation_years": customer.RelationYears,
		},
		Position: &GraphPosition{X: 0, Y: 0}, // Center
	}
	entities = append(entities, customerNode)

	// 2. Add recent orders
	var orders []Order
	a.db.Where("customer_id = ?", customer.ID).Order("order_date DESC").Limit(10).Find(&orders)
	for i, order := range orders {
		orderNode := GraphEntity{
			ID:    fmt.Sprintf("order-%s", order.ID),
			Type:  "order",
			Label: order.OrderNumber,
			Data: map[string]any{
				"value":  order.TotalValueBHD,
				"date":   order.OrderDate,
				"status": order.Status,
			},
			Position: &GraphPosition{X: float64(i * 100), Y: 200}, // Arc below customer
		}
		entities = append(entities, orderNode)

		// Relation: customer -> order
		relations = append(relations, GraphRelation{
			ID:     fmt.Sprintf("rel-customer-order-%s", order.ID),
			Source: customerNode.ID,
			Target: orderNode.ID,
			Type:   "ordered",
			Label:  "ordered",
			Weight: order.TotalValueBHD,
		})
	}

	// 3. Add recent invoices
	var invoices []Invoice
	a.db.Where("customer_id = ?", customer.ID).Order("invoice_date DESC").Limit(10).Find(&invoices)
	for i, invoice := range invoices {
		invoiceNode := GraphEntity{
			ID:    fmt.Sprintf("invoice-%s", invoice.ID),
			Type:  "invoice",
			Label: invoice.InvoiceNumber,
			Data: map[string]any{
				"value":       invoice.GrandTotalBHD,
				"date":        invoice.InvoiceDate,
				"status":      invoice.Status,
				"outstanding": invoice.OutstandingBHD,
			},
			Position: &GraphPosition{X: float64(i * 100), Y: -200}, // Arc above customer
		}
		entities = append(entities, invoiceNode)

		// Relation: customer -> invoice
		relations = append(relations, GraphRelation{
			ID:     fmt.Sprintf("rel-customer-invoice-%s", invoice.ID),
			Source: customerNode.ID,
			Target: invoiceNode.ID,
			Type:   "invoiced",
			Label:  "invoiced",
			Weight: invoice.GrandTotalBHD,
		})
	}

	// 4. Add contacts
	var contacts []CustomerContact
	a.db.Where("customer_id = ?", customer.ID).Find(&contacts)
	for i, contact := range contacts {
		contactNode := GraphEntity{
			ID:    fmt.Sprintf("contact-%s", contact.ID),
			Type:  "contact",
			Label: contact.ContactName,
			Data: map[string]any{
				"title":      contact.JobTitle,
				"email":      contact.Email,
				"is_primary": contact.IsPrimaryContact,
			},
			Position: &GraphPosition{X: float64(i * 150), Y: 400}, // Bottom arc
		}
		entities = append(entities, contactNode)

		// Relation: customer -> contact
		relations = append(relations, GraphRelation{
			ID:     fmt.Sprintf("rel-customer-contact-%s", contact.ID),
			Source: customerNode.ID,
			Target: contactNode.ID,
			Type:   "has_contact",
			Label:  contact.JobTitle,
		})
	}

	// Calculate metrics
	totalNodes := len(entities)
	totalEdges := len(relations)
	avgConnections := 0.0
	if totalNodes > 0 {
		avgConnections = float64(totalEdges) / float64(totalNodes)
	}
	graphDensity := 0.0
	maxPossibleEdges := totalNodes * (totalNodes - 1)
	if maxPossibleEdges > 0 {
		graphDensity = float64(totalEdges) / float64(maxPossibleEdges)
	}

	metrics := GraphMetrics{
		TotalNodes:         totalNodes,
		TotalEdges:         totalEdges,
		AverageConnections: avgConnections,
		GraphDensity:       graphDensity,
	}

	graph := Customer360Graph{
		CustomerID: customerID,
		Entities:   entities,
		Relations:  relations,
		Metrics:    metrics,
	}

	log.Printf("🌐 Built Customer 360 Graph for %s: %d nodes, %d edges", customerID, totalNodes, totalEdges)
	return graph, nil
}

// GetCustomer360 is an alias for GetCustomer360View (frontend uses this name)
func (a *App) GetCustomer360(customerID string) (Customer360Data, error) {
	return a.GetCustomer360View(customerID)
}

// ============================================================================
// FOLLOW-UP TASKS MANAGEMENT
// ============================================================================

// ListFollowUps retrieves all follow-up tasks with optional limit
func (a *App) ListFollowUps(limit int) ([]FollowUpTask, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Default limit: 100, max limit: 1000
	if limit <= 0 {
		limit = 100
	} else if limit > 1000 {
		limit = 1000
	}

	var tasks []FollowUpTask
	query := a.db.Where("status != ?", "completed").Order("due_date ASC")
	query = query.Limit(limit)
	if err := query.Find(&tasks).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to list follow-ups", err.Error())
	}

	log.Printf("📋 Retrieved %d follow-up tasks", len(tasks))
	return tasks, nil
}

// CreateFollowUp creates a new follow-up task
func (a *App) CreateFollowUp(task FollowUpTask) (FollowUpTask, error) {
	if err := a.requirePermission("tasks:create"); err != nil {
		return FollowUpTask{}, err
	}
	if a.db == nil {
		return FollowUpTask{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate required fields
	if task.Title == "" {
		return FollowUpTask{}, newError("INVALID_INPUT", "Title is required", "")
	}
	if task.CustomerID == "" {
		return FollowUpTask{}, newError("INVALID_INPUT", "Customer ID is required", "")
	}
	if task.DueDate.IsZero() {
		return FollowUpTask{}, newError("INVALID_INPUT", "Due date is required", "")
	}

	// Set defaults
	if task.Status == "" {
		task.Status = "pending"
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	if task.Type == "" {
		task.Type = "general"
	}

	result := a.db.Create(&task)
	if result.Error != nil {
		log.Printf("❌ Failed to create follow-up: %v", result.Error)
		return FollowUpTask{}, newError("DB_CREATE_FAILED", "Failed to create follow-up", result.Error.Error())
	}

	log.Printf("✅ Created Follow-up #%s: %s (due: %s)", task.ID, task.Title, task.DueDate.Format("2006-01-02"))
	return task, nil
}

// UpdateFollowUp updates an existing follow-up task
func (a *App) UpdateFollowUp(id uint, task FollowUpTask) error {
	if err := a.requirePermission("tasks:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate task exists
	var existing FollowUpTask
	if err := a.db.First(&existing, id).Error; err != nil {
		return newError("FOLLOWUP_NOT_FOUND", "Follow-up task not found", err.Error())
	}

	// Update fields
	updates := map[string]any{
		"title":       task.Title,
		"description": task.Description,
		"due_date":    task.DueDate,
		"status":      task.Status,
		"priority":    task.Priority,
		"type":        task.Type,
		"amount":      task.Amount,
		"contact":     task.Contact,
		"notes":       task.Notes,
	}

	result := a.db.Model(&FollowUpTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return newError("DB_UPDATE_FAILED", "Failed to update follow-up", result.Error.Error())
	}

	log.Printf("✅ Updated Follow-up #%d: %s", id, task.Title)
	return nil
}

// CompleteFollowUp marks a follow-up task as complete
func (a *App) CompleteFollowUp(id string) error {
	if err := a.requirePermission("tasks:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate task exists
	var task FollowUpTask
	if err := a.db.First(&task, "id = ?", id).Error; err != nil {
		return newError("FOLLOWUP_NOT_FOUND", "Follow-up task not found", err.Error())
	}

	// Update status and completion timestamp
	now := time.Now()
	updates := map[string]any{
		"status":       "completed",
		"completed_at": &now,
	}

	result := a.db.Model(&FollowUpTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return newError("DB_UPDATE_FAILED", "Failed to complete follow-up", result.Error.Error())
	}

	log.Printf("✅ Completed Follow-up #%s: %s", id, task.Title)
	return nil
}

// GetOverdueFollowUps retrieves all overdue follow-up tasks
func (a *App) GetOverdueFollowUps() ([]FollowUpTask, error) {
	if err := a.requirePermission("tasks:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	now := time.Now()
	var tasks []FollowUpTask
	query := a.db.Where("status = ? AND due_date < ?", "pending", now).Order("due_date ASC")
	if err := query.Find(&tasks).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to get overdue follow-ups", err.Error())
	}

	log.Printf("⚠️ Found %d overdue follow-up tasks", len(tasks))
	return tasks, nil
}

// ============================================================================
// SETTINGS MANAGEMENT (SettingsScreen.svelte)
// ============================================================================

// GetSettings returns current app settings (safe for frontend - secrets masked)
// getSettingsFilePath returns the path to the settings JSON file
