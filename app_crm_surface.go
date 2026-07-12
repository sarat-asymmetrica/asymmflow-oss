package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

func (a *App) GetCustomer360View(customerID string) (Customer360Data, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return Customer360Data{}, err
	}

	linkIndex := a.buildCustomerLinkIndex()
	customer, ok := linkIndex.resolve(customerID, customerID)
	if !ok {
		log.Printf("Error fetching customer %s: not found in customer linkage index", customerID)
		return Customer360Data{}, fmt.Errorf("customer not found: %s", customerID)
	}

	// Fetch recent predictions
	var predictions []PredictionRecord
	predictionIDs := uniqueNonEmptyStrings(customerID, customer.ID, customer.CustomerID, customer.CustomerCode)
	a.db.Where("customer_id IN ?", predictionIDs).
		Order("created_at desc").
		Limit(10).
		Find(&predictions)

	// Extract regime data from most recent prediction
	var r1, r2, r3 float64
	if len(predictions) > 0 {
		// Use most recent prediction's regime data
		r1 = predictions[0].R1
		r2 = predictions[0].R2
		r3 = predictions[0].R3
	}

	profile, profileErr := a.GetCustomerFullProfile(customer.ID)
	if profileErr != nil {
		log.Printf("⚠️ Customer360 profile linkage failed for %s: %v", customer.ID, profileErr)
	}

	// WAVE 2 AGENT 4: Fetch deep customer joins
	receivablesAging := profile.ARAgingBuckets
	paymentHistory := profile.PaymentHistory
	openOpportunities := a.GetCustomerOpportunities(customer.ID)
	if len(openOpportunities) == 0 {
		openOpportunities = profile.RecentRFQs
	}
	recentOrders := profile.RecentOrders
	customerLTV := profile.TotalRevenue
	totalOrdersValue := roundTo3(profile.AvgOrderValue * float64(profile.TotalOrders))

	// Assemble 360 view
	data := Customer360Data{
		CustomerID:         customer.CustomerID,
		BusinessName:       customer.BusinessName,
		CustomerType:       customer.CustomerType,
		Industry:           customer.Industry,
		City:               customer.City,
		Country:            customer.Country,
		RelationYears:      customer.RelationYears,
		CurrentGrade:       customer.PaymentGrade,
		PaymentTermsDays:   customer.PaymentTermsDays,
		AvgPaymentDays:     customer.AvgPaymentDays,
		DisputeCount:       customer.DisputeCount,
		IsCreditBlocked:    customer.IsCreditBlocked,
		RequiresPrepayment: customer.RequiresPrepayment,
		R1:                 r1,
		R2:                 r2,
		R3:                 r3,
		TotalOrdersValue:   totalOrdersValue,
		TotalOrdersCount:   profile.TotalOrders,
		AvgOrderValue:      profile.AvgOrderValue,
		LastOrderDate:      profile.LastOrderDate,
		HasABBCompetition:  customer.HasABBCompetition,
		IsEmergencyOnly:    customer.IsEmergencyOnly,
		RecentPredictions:  predictions,
		// WAVE 2 AGENT 4: Deep joins
		ReceivablesAging:      receivablesAging,
		PaymentHistory:        paymentHistory,
		OpenOpportunities:     openOpportunities,
		RecentOrders:          recentOrders,
		CustomerLifetimeValue: customerLTV,
	}

	log.Printf("Retrieved 360 view for %s (%s): Grade %s, R1=%.1f%% R2=%.1f%% R3=%.1f%%, %d predictions, LTV: %.2f BHD, Outstanding: %.2f BHD",
		customerID, customer.BusinessName, customer.PaymentGrade, r1*100, r2*100, r3*100, len(predictions), customerLTV, receivablesAging.TotalOutstanding)

	return data, nil
}

// GetAllCustomers retrieves list of all customers for selection/filtering
func (a *App) GetAllCustomers() ([]CustomerMaster, error) {
	// P0 FIX: Add permission check - customers:view required
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	var customers []CustomerMaster
	result := a.db.Order("business_name ASC").Find(&customers)

	if result.Error != nil {
		log.Printf("❌ Error retrieving customers: %v", result.Error)
		return nil, fmt.Errorf("failed to retrieve customers: %w", result.Error)
	}

	log.Printf("✅ Retrieved %d customers", len(customers))
	return customers, nil
}

// GetCustomersByGrade retrieves customers filtered by payment grade
func (a *App) GetCustomersByGrade(grade string) ([]CustomerMaster, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	var customers []CustomerMaster
	result := a.db.Where("payment_grade = ?", grade).
		Order("business_name ASC").
		Find(&customers)

	if result.Error != nil {
		log.Printf("❌ Error retrieving customers for grade %s: %v", grade, result.Error)
		return nil, fmt.Errorf("failed to retrieve customers for grade %s: %w", grade, result.Error)
	}

	log.Printf("✅ Retrieved %d customers with grade %s", len(customers), grade)
	return customers, nil
}

// ============================================================================
// I-22: CUSTOMER RELATED PRODUCTS / SUPPLIERS (aggregated from order history)
// ============================================================================

// CustomerRelatedProduct is a distinct product a customer has ordered, rolled up
// by total quantity / order count / last-ordered.
type CustomerRelatedProduct struct {
	ProductCode     string     `json:"product_code"`
	ProductName     string     `json:"product_name"`
	ProductCategory string     `json:"product_category"`
	SupplierName    string     `json:"supplier_name"`
	TotalQuantity   float64    `json:"total_quantity"`
	OrderCount      int        `json:"order_count"`
	LastOrdered     *time.Time `json:"last_ordered"`
}

// CustomerRelatedSupplier is a distinct supplier behind the products a customer
// has ordered, derived through those products.
type CustomerRelatedSupplier struct {
	SupplierID   string     `json:"supplier_id"`
	SupplierName string     `json:"supplier_name"`
	SupplierCode string     `json:"supplier_code"`
	SupplierType string     `json:"supplier_type"`
	ProductCount int        `json:"product_count"`
	LastOrdered  *time.Time `json:"last_ordered"`
}

// customerProductRollup is the internal per-product aggregation that both
// related-entity endpoints share so the order-history join is expressed once.
type customerProductRollup struct {
	ProductID       string
	ProductCode     string
	ProductName     string
	ProductCategory string
	Supplier        *SupplierMaster
	TotalQuantity   float64
	OrderCount      int
	LastOrdered     time.Time
}

// customerProductRollups walks a customer's order history and returns one rollup
// per distinct ordered product (keyed by product_id, falling back to product_code
// for legacy free-text lines). Internal helper — callers enforce RBAC.
//
// OSS adaptation: unlike deployed PH (which has no SKU catalog and rolls up on a
// Brand×Token instrument taxonomy tagged onto order lines), the OSS schema carries
// a real ProductMaster with a direct SupplierID/SupplierCode, so supplier is
// resolved straight off the hydrated product.
func (a *App) customerProductRollups(customerID string) ([]customerProductRollup, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	linkIndex := a.buildCustomerLinkIndex()
	customer, ok := linkIndex.resolve(customerID, customerID)
	if !ok {
		return nil, fmt.Errorf("customer not found: %s", customerID)
	}

	orders := a.linkedOrdersForCustomer(linkIndex, customer)
	if len(orders) == 0 {
		return nil, nil
	}
	orderDate := make(map[string]time.Time, len(orders))
	orderIDs := make([]string, 0, len(orders))
	for _, o := range orders {
		orderIDs = append(orderIDs, o.ID)
		orderDate[o.ID] = o.OrderDate
	}

	var items []OrderItem
	if err := a.db.Where("order_id IN ?", orderIDs).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}

	type agg struct {
		productID     string
		productCode   string
		totalQuantity float64
		orders        map[string]bool
		lastOrdered   time.Time
	}

	aggs := make(map[string]*agg)
	for _, it := range items {
		// Group by a real product id, falling back to the product code for legacy
		// free-text lines. A line with neither carries no information to roll up.
		var key string
		switch {
		case strings.TrimSpace(it.ProductID) != "":
			key = it.ProductID
		case strings.TrimSpace(it.ProductCode) != "":
			key = "code:" + strings.ToUpper(strings.TrimSpace(it.ProductCode))
		default:
			continue
		}
		g, ok := aggs[key]
		if !ok {
			g = &agg{productID: it.ProductID, productCode: it.ProductCode, orders: map[string]bool{}}
			aggs[key] = g
		}
		g.totalQuantity += it.Quantity
		g.orders[it.OrderID] = true
		if d := orderDate[it.OrderID]; d.After(g.lastOrdered) {
			g.lastOrdered = d
		}
		if strings.TrimSpace(g.productCode) == "" {
			g.productCode = it.ProductCode
		}
	}

	supplierCache := make(map[string]*SupplierMaster) // supplier id/code -> supplier

	rollups := make([]customerProductRollup, 0, len(aggs))
	for _, g := range aggs {
		r := customerProductRollup{
			ProductID:     g.productID,
			ProductCode:   g.productCode,
			TotalQuantity: roundTo3(g.totalQuantity),
			OrderCount:    len(g.orders),
			LastOrdered:   g.lastOrdered,
		}

		var product ProductMaster
		found := false
		if strings.TrimSpace(g.productID) != "" {
			if err := a.db.First(&product, "id = ?", g.productID).Error; err == nil {
				found = true
			}
		}
		if !found && strings.TrimSpace(g.productCode) != "" {
			if err := a.db.Where("product_code = ?", g.productCode).First(&product).Error; err == nil {
				found = true
			}
		}
		if found {
			r.ProductName = product.ProductName
			r.ProductCategory = product.ProductCategory
			r.Supplier = a.resolveSupplierForRelatedProduct(product, supplierCache)
		}
		if strings.TrimSpace(r.ProductName) == "" {
			r.ProductName = g.productCode // legacy line: show the code as the name
		}
		rollups = append(rollups, r)
	}
	return rollups, nil
}

// resolveSupplierForRelatedProduct resolves the supplier behind a product using
// the OSS ProductMaster's direct SupplierID (then SupplierCode) link, memoizing
// lookups across a single rollup pass. Returns nil when no supplier resolves.
func (a *App) resolveSupplierForRelatedProduct(product ProductMaster, cache map[string]*SupplierMaster) *SupplierMaster {
	if id := strings.TrimSpace(product.SupplierID); id != "" {
		if s, ok := cache["id:"+id]; ok {
			return s
		}
		var s SupplierMaster
		if err := a.db.First(&s, "id = ?", id).Error; err == nil {
			cache["id:"+id] = &s
			return &s
		}
		cache["id:"+id] = nil
	}
	if code := strings.TrimSpace(product.SupplierCode); code != "" {
		if s, ok := cache["code:"+code]; ok {
			return s
		}
		var s SupplierMaster
		if err := a.db.Where("supplier_code = ?", code).First(&s).Error; err == nil {
			cache["code:"+code] = &s
			return &s
		}
		cache["code:"+code] = nil
	}
	return nil
}

// GetCustomerRelatedProducts returns the distinct products a customer has ordered,
// aggregated by quantity / order count / last-ordered. (I-22, RBAC customers:view)
func (a *App) GetCustomerRelatedProducts(customerID string) ([]CustomerRelatedProduct, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	rollups, err := a.customerProductRollups(customerID)
	if err != nil {
		return nil, err
	}

	results := make([]CustomerRelatedProduct, 0, len(rollups))
	for _, r := range rollups {
		rp := CustomerRelatedProduct{
			ProductCode:     r.ProductCode,
			ProductName:     r.ProductName,
			ProductCategory: r.ProductCategory,
			TotalQuantity:   r.TotalQuantity,
			OrderCount:      r.OrderCount,
		}
		if r.Supplier != nil {
			rp.SupplierName = r.Supplier.SupplierName
		}
		if !r.LastOrdered.IsZero() {
			t := r.LastOrdered
			rp.LastOrdered = &t
		}
		results = append(results, rp)
	}

	// Sort by last_ordered desc, then order_count desc.
	sort.Slice(results, func(i, j int) bool {
		li, lj := results[i].LastOrdered, results[j].LastOrdered
		if (li == nil) != (lj == nil) {
			return li != nil
		}
		if li != nil && lj != nil && !li.Equal(*lj) {
			return li.After(*lj)
		}
		return results[i].OrderCount > results[j].OrderCount
	})

	const maxProducts = 50
	if len(results) > maxProducts {
		log.Printf("📦 GetCustomerRelatedProducts(%s): %d products, capped to %d", customerID, len(results), maxProducts)
		results = results[:maxProducts]
	}
	return results, nil
}

// GetCustomerRelatedSuppliers returns the distinct suppliers behind the products a
// customer has ordered, derived through products. (I-22, RBAC customers:view)
func (a *App) GetCustomerRelatedSuppliers(customerID string) ([]CustomerRelatedSupplier, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	rollups, err := a.customerProductRollups(customerID)
	if err != nil {
		return nil, err
	}

	type supAgg struct {
		supplier    SupplierMaster
		products    map[string]bool
		lastOrdered time.Time
	}
	bySupplier := make(map[string]*supAgg)
	for _, r := range rollups {
		if r.Supplier == nil {
			continue // product with no resolvable supplier — skip from supplier view
		}
		s, ok := bySupplier[r.Supplier.ID]
		if !ok {
			s = &supAgg{supplier: *r.Supplier, products: map[string]bool{}}
			bySupplier[r.Supplier.ID] = s
		}
		// Count distinct products per supplier by product id/code identity.
		pkey := strings.TrimSpace(r.ProductID)
		if pkey == "" {
			pkey = "code:" + strings.ToUpper(strings.TrimSpace(r.ProductCode))
		}
		s.products[pkey] = true
		if r.LastOrdered.After(s.lastOrdered) {
			s.lastOrdered = r.LastOrdered
		}
	}

	results := make([]CustomerRelatedSupplier, 0, len(bySupplier))
	for _, s := range bySupplier {
		rs := CustomerRelatedSupplier{
			SupplierID:   s.supplier.ID,
			SupplierName: s.supplier.SupplierName,
			SupplierCode: s.supplier.SupplierCode,
			SupplierType: s.supplier.SupplierType,
			ProductCount: len(s.products),
		}
		if !s.lastOrdered.IsZero() {
			t := s.lastOrdered
			rs.LastOrdered = &t
		}
		results = append(results, rs)
	}

	sort.Slice(results, func(i, j int) bool {
		li, lj := results[i].LastOrdered, results[j].LastOrdered
		if (li == nil) != (lj == nil) {
			return li != nil
		}
		if li != nil && lj != nil && !li.Equal(*lj) {
			return li.After(*lj)
		}
		return results[i].ProductCount > results[j].ProductCount
	})

	const maxSuppliers = 50
	if len(results) > maxSuppliers {
		log.Printf("🏭 GetCustomerRelatedSuppliers(%s): %d suppliers, capped to %d", customerID, len(results), maxSuppliers)
		results = results[:maxSuppliers]
	}
	return results, nil
}

// ============================================================================
// WAVE 2 AGENT 4: DEEP CUSTOMER JOINS - HELPER FUNCTIONS
// ============================================================================

// GetReceivablesAging calculates outstanding invoice amounts by aging bucket for a customer
func (a *App) GetReceivablesAging(customerID string) ReceivablesAgingSummary {
	if err := a.requirePermission("customers:view"); err != nil {
		return ReceivablesAgingSummary{}
	}
	linkIndex := a.buildCustomerLinkIndex()
	customer, ok := linkIndex.resolve(customerID, customerID)
	if !ok {
		log.Printf("Receivables aging: customer %s could not be resolved", customerID)
		return ReceivablesAgingSummary{}
	}
	invoices := a.linkedInvoicesForCustomer(linkIndex, customer)
	orders := a.linkedOrdersForCustomer(linkIndex, customer)
	orderExposure := linkedOrderExposure(orders, invoices)
	aging := linkedReceivablesAging(invoices, orderExposure)

	log.Printf("Receivables aging for customer %s: Total %.2f BHD (Current: %.2f, 30-60: %.2f, 60-90: %.2f, 90-120: %.2f, 120+: %.2f)",
		customerID, aging.TotalOutstanding, aging.Current, aging.Days30_60, aging.Days60_90, aging.Days90_120, aging.Days120Plus)

	return aging
}

// GetPaymentHistory retrieves last N payments for a customer
func (a *App) GetPaymentHistory(customerID string, limit int) []PaymentHistoryEntry {
	if err := a.requirePermission("payments:view"); err != nil {
		return []PaymentHistoryEntry{}
	}
	linkIndex := a.buildCustomerLinkIndex()
	customer, ok := linkIndex.resolve(customerID, customerID)
	if !ok {
		log.Printf("Payment history: customer %s could not be resolved", customerID)
		return []PaymentHistoryEntry{}
	}
	history := a.linkedPaymentHistoryForCustomer(linkIndex, customer, limit)

	log.Printf("Retrieved %d payment history entries for customer %s", len(history), customerID)
	return history
}

// GetCustomerOpportunities retrieves open RFQs/opportunities for a customer
func (a *App) GetCustomerOpportunities(customerID string) []OpportunitySummary {
	if err := a.requirePermission("offers:view"); err != nil {
		return []OpportunitySummary{}
	}
	linkIndex := a.buildCustomerLinkIndex()
	customer, ok := linkIndex.resolve(customerID, customerID)
	if !ok {
		log.Printf("Customer %s not found for opportunities lookup", customerID)
		return []OpportunitySummary{}
	}

	rfqs := a.linkedRFQsForCustomer(linkIndex, customer)
	offers := a.linkedOffersForCustomer(linkIndex, customer)
	pipeline := a.linkedOpportunitiesForCustomer(linkIndex, customer)
	pipelineEvents := buildCustomerPipelineEvents(customer.ID, rfqs, offers, pipeline, true)

	opportunities := make([]OpportunitySummary, 0, len(pipelineEvents))
	for _, event := range pipelineEvents {
		if closedLostStage(event.Status) || closedWonStage(event.Status) {
			continue
		}
		opportunities = append(opportunities, OpportunitySummary{
			Project:   firstNonEmpty(event.Project, event.Ref),
			Value:     event.Value,
			Status:    event.Status,
			CreatedAt: event.Date,
		})
	}
	sortOpportunitySummariesByCreatedAt(opportunities)
	if len(opportunities) > 10 {
		opportunities = opportunities[:10]
	}

	log.Printf("Retrieved %d open opportunities for customer %s (%s)", len(opportunities), customerID, customer.BusinessName)
	return opportunities
}

// GetCustomerRecentOrders retrieves last N orders for a customer
func (a *App) GetCustomerRecentOrders(customerID string, limit int) []OrderSummary {
	if err := a.requirePermission("orders:view"); err != nil {
		return []OrderSummary{}
	}
	linkIndex := a.buildCustomerLinkIndex()
	customer, ok := linkIndex.resolve(customerID, customerID)
	if !ok {
		log.Printf("Recent orders: customer %s could not be resolved", customerID)
		return []OrderSummary{}
	}
	orders := a.linkedOrdersForCustomer(linkIndex, customer)
	summary := make([]OrderSummary, 0, len(orders))
	for _, o := range orders {
		if !commercialOrderStatus(o.Status) {
			continue
		}
		summary = append(summary, OrderSummary{
			OrderNumber:   o.OrderNumber,
			OrderDate:     o.OrderDate,
			TotalValueBHD: customerCommercialOrderValue(o),
			Status:        o.Status,
		})
		if limit > 0 && len(summary) >= limit {
			break
		}
	}

	log.Printf("Retrieved %d recent orders for customer %s", len(summary), customerID)
	return summary
}

// ============================================================================
// CUSTOMER & SUPPLIER FULL PROFILE APIS
// ============================================================================

// CustomerFullProfile represents complete customer profile with all relationships
type CustomerFullProfile struct {
	// Basic Info
	ID           string `json:"id"`
	CustomerID   string `json:"customer_id"`
	BusinessName string `json:"business_name"`
	CustomerType string `json:"customer_type"`
	ShortCode    string `json:"short_code"`
	TRN          string `json:"trn"`
	Industry     string `json:"industry"`

	// Address
	AddressLine1 string `json:"address_line1"`
	City         string `json:"city"`
	Country      string `json:"country"`

	// Contacts
	Contacts []CustomerContact `json:"contacts"`

	// Financial Profile
	PaymentGrade     string  `json:"payment_grade"`
	PaymentTermsDays int     `json:"payment_terms_days"`
	CreditLimit      float64 `json:"credit_limit"`
	IsCreditBlocked  bool    `json:"is_credit_blocked"`

	// Metrics
	TotalRevenue  float64    `json:"total_revenue"`
	TotalOrders   int        `json:"total_orders"`
	AvgOrderValue float64    `json:"avg_order_value"`
	LastOrderDate *time.Time `json:"last_order_date"`
	RelationYears int        `json:"relation_years"`

	// AR Status
	OutstandingBHD float64                 `json:"outstanding_bhd"`
	OverdueBHD     float64                 `json:"overdue_bhd"`
	ARAgingBuckets ReceivablesAgingSummary `json:"ar_aging_buckets"`

	// Relationship History
	RFQsFloated    int                   `json:"rfqs_floated"`
	RFQsWon        int                   `json:"rfqs_won"`
	WinRate        float64               `json:"win_rate"`
	RecentRFQs     []OpportunitySummary  `json:"recent_rfqs"`
	RecentOrders   []OrderSummary        `json:"recent_orders"`
	RecentInvoices []InvoiceSummary      `json:"recent_invoices"`
	PaymentHistory []PaymentHistoryEntry `json:"payment_history"`

	// Notes
	Notes []EntityNote `json:"notes"`
}

type InvoiceSummary struct {
	ID            string    `json:"id"`
	InvoiceNumber string    `json:"invoice_number"`
	InvoiceDate   time.Time `json:"invoice_date"`
	GrandTotalBHD float64   `json:"grand_total_bhd"`
	Status        string    `json:"status"`
}

// GetCustomerFullProfile retrieves complete customer profile with all relationships
func (a *App) GetCustomerFullProfile(customerID string) (CustomerFullProfile, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return CustomerFullProfile{}, err
	}
	profile := CustomerFullProfile{}

	if a.db == nil {
		return profile, fmt.Errorf("database not initialized")
	}

	// 1. Get customer master through the canonical linkage layer. Transactional rows
	// use a mix of UUIDs, business customer codes, and names depending on import path.
	linkIndex := a.buildCustomerLinkIndex()
	customer, ok := linkIndex.resolve(customerID, customerID)
	if !ok {
		return profile, fmt.Errorf("customer not found: %s", customerID)
	}

	profile.ID = customer.ID
	profile.CustomerID = customer.CustomerID
	if strings.TrimSpace(profile.CustomerID) == "" || profile.CustomerID == customer.ID || looksLikeUUID(profile.CustomerID) {
		profile.CustomerID = businessCustomerIDFromRecord(customer)
	}
	profile.BusinessName = customer.BusinessName
	profile.CustomerType = customer.CustomerType
	profile.ShortCode = customer.ShortCode
	profile.TRN = customer.TRN
	profile.Industry = customer.Industry
	profile.AddressLine1 = customer.AddressLine1
	profile.City = customer.City
	profile.Country = customer.Country
	profile.PaymentGrade = customer.PaymentGrade
	profile.PaymentTermsDays = customer.PaymentTermsDays
	profile.CreditLimit = customer.CreditLimitBHD
	profile.IsCreditBlocked = customer.IsCreditBlocked
	profile.LastOrderDate = customer.LastOrderDate
	profile.RelationYears = customer.RelationYears

	invoices := a.linkedInvoicesForCustomer(linkIndex, customer)
	orders := a.linkedOrdersForCustomer(linkIndex, customer)
	offers := a.linkedOffersForCustomer(linkIndex, customer)
	rfqs := a.linkedRFQsForCustomer(linkIndex, customer)
	opportunities := a.linkedOpportunitiesForCustomer(linkIndex, customer)

	var postedInvoiceValue float64
	for _, inv := range invoices {
		if invoicePostedStatus(inv.Status) {
			postedInvoiceValue += inv.GrandTotalBHD
		}
	}

	var totalOrderValue float64
	ordersByOfferID := make(map[string]bool)
	for _, order := range orders {
		if !commercialOrderStatus(order.Status) {
			continue
		}
		orderValue := customerCommercialOrderValue(order)
		totalOrderValue += orderValue
		profile.TotalOrders++
		if profile.LastOrderDate == nil || order.OrderDate.After(*profile.LastOrderDate) {
			last := order.OrderDate
			profile.LastOrderDate = &last
		}
		if strings.TrimSpace(order.OfferID) != "" {
			ordersByOfferID[order.OfferID] = true
		}
	}
	if profile.TotalOrders > 0 {
		profile.AvgOrderValue = roundTo3(totalOrderValue / float64(profile.TotalOrders))
	}

	pipelineEvents := buildCustomerPipelineEvents(customer.ID, rfqs, offers, opportunities, true)
	orderExposure := linkedOrderExposure(orders, invoices)
	var activePipelineValue float64
	for _, event := range pipelineEvents {
		if event.Value <= 0 || closedLostStage(event.Status) {
			continue
		}
		if strings.TrimSpace(event.OfferID) != "" && ordersByOfferID[event.OfferID] {
			continue
		}
		activePipelineValue += event.Value
	}
	profile.TotalRevenue = roundTo3(postedInvoiceValue + orderExposure + activePipelineValue)

	// 2. Get contacts
	var allContacts []CustomerContact
	a.db.Find(&allContacts)
	contacts := make([]CustomerContact, 0, len(allContacts))
	for _, contact := range allContacts {
		if linkIndex.matches(customer, contact.CustomerID, "") {
			contacts = append(contacts, contact)
		}
	}
	profile.Contacts = contacts

	// 3. Get AR aging
	profile.ARAgingBuckets = linkedReceivablesAging(invoices, orderExposure)
	profile.OutstandingBHD = profile.ARAgingBuckets.TotalOutstanding

	// Calculate overdue
	profile.OverdueBHD = profile.ARAgingBuckets.Days30_60 + profile.ARAgingBuckets.Days60_90 +
		profile.ARAgingBuckets.Days90_120 + profile.ARAgingBuckets.Days120Plus

	// 4. Get RFQ stats
	wonCount := 0
	recentRFQs := make([]OpportunitySummary, 0)
	for _, event := range pipelineEvents {
		profile.RFQsFloated++
		if closedWonStage(event.Status) {
			wonCount++
		}
		recentRFQs = append(recentRFQs, OpportunitySummary{
			Project:   firstNonEmpty(event.Project, event.Ref),
			Value:     event.Value,
			Status:    event.Status,
			CreatedAt: event.Date,
		})
	}
	profile.RFQsWon = wonCount
	if profile.RFQsFloated > 0 {
		profile.WinRate = float64(wonCount) / float64(profile.RFQsFloated) * 100
	}
	sortOpportunitySummariesByCreatedAt(recentRFQs)
	if len(recentRFQs) > 5 {
		recentRFQs = recentRFQs[:5]
	}
	profile.RecentRFQs = recentRFQs

	// 5. Get recent orders
	recentOrders := make([]OrderSummary, 0, len(orders))
	for _, order := range orders {
		if !commercialOrderStatus(order.Status) {
			continue
		}
		recentOrders = append(recentOrders, OrderSummary{
			OrderNumber:   order.OrderNumber,
			OrderDate:     order.OrderDate,
			TotalValueBHD: customerCommercialOrderValue(order),
			Status:        order.Status,
		})
	}
	sortOrderSummariesByDate(recentOrders)
	if len(recentOrders) > 5 {
		recentOrders = recentOrders[:5]
	}
	profile.RecentOrders = recentOrders

	// 6. Get recent invoices
	recentInvoices := make([]InvoiceSummary, 0, len(invoices))
	for _, inv := range invoices {
		recentInvoices = append(recentInvoices, InvoiceSummary{
			ID:            inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			InvoiceDate:   inv.InvoiceDate,
			GrandTotalBHD: inv.GrandTotalBHD,
			Status:        inv.Status,
		})
	}
	sortInvoiceSummariesByDate(recentInvoices)
	if len(recentInvoices) > 5 {
		recentInvoices = recentInvoices[:5]
	}
	profile.RecentInvoices = recentInvoices

	// 7. Get payment history
	if err := a.requirePermission("payments:view"); err == nil {
		profile.PaymentHistory = a.linkedPaymentHistoryForCustomer(linkIndex, customer, 6)
	}

	// 8. Get notes
	var notes []EntityNote
	a.db.Where("entity_type = ? AND entity_id = ?", "customer", customerID).Order("created_at DESC").Find(&notes)
	profile.Notes = notes

	log.Printf("📋 Customer Full Profile: %s (%s) - %d contacts, %d notes",
		customer.BusinessName, customerID, len(contacts), len(notes))

	return profile, nil
}

// AddCustomerNote adds a note to a customer
func (a *App) AddCustomerNote(customerID string, noteType string, content string) error {
	if err := a.requirePermission("customers:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// P1 FIX: Validate note content
	if GlobalValidator != nil {
		if err := GlobalValidator.ValidateNoteInput(content); err != nil {
			return fmt.Errorf("invalid note content: %w", err)
		}
	}

	note := EntityNote{
		EntityType: "customer",
		EntityID:   customerID,
		NoteType:   noteType,
		Content:    content,
	}

	if err := a.db.Create(&note).Error; err != nil {
		return fmt.Errorf("failed to create note: %v", err)
	}

	log.Printf("📝 Added customer note: %s (%s)", noteType, customerID)
	return nil
}

// SupplierFullProfile represents complete supplier profile
type SupplierFullProfile struct {
	// Basic Info
	ID           string `json:"id"`
	SupplierCode string `json:"supplier_code"`
	SupplierName string `json:"supplier_name"`
	SupplierType string `json:"supplier_type"`
	TaxID        string `json:"tax_id"`
	Country      string `json:"country"`

	// Address
	Address string `json:"address"`

	// Contact
	PrimaryContact string `json:"primary_contact"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`

	// Offerings
	BrandsHandled []string `json:"brands_handled"`
	ProductTypes  []string `json:"product_types"`

	// Bank Details
	BankName      string `json:"bank_name"`
	AccountNumber string `json:"account_number"`
	IBAN          string `json:"iban"`
	SwiftCode     string `json:"swift_code"`

	// Performance
	Rating            int     `json:"rating"`
	LeadTimeDays      int     `json:"lead_time_days"`
	OnTimeDeliveryPct float64 `json:"on_time_delivery_pct"`

	// Metrics
	TotalPurchases float64 `json:"total_purchases"`
	TotalPOs       int     `json:"total_pos"`
	AvgPOValue     float64 `json:"avg_po_value"`

	// Outstanding
	OutstandingBHD float64 `json:"outstanding_bhd"`
	OverdueBHD     float64 `json:"overdue_bhd"`

	// History
	RecentPOs      []POSummary              `json:"recent_pos"`
	RecentInvoices []SupplierInvoiceSummary `json:"recent_invoices"`

	// Issue Tracking
	Issues     []SupplierIssue `json:"issues"`
	OpenIssues int             `json:"open_issues"`
	IssueCost  float64         `json:"issue_cost"`

	// Notes
	Notes []EntityNote `json:"notes"`
}

type POSummary struct {
	ID       string    `json:"id"`
	PONumber string    `json:"po_number"`
	PODate   time.Time `json:"po_date"`
	TotalBHD float64   `json:"total_bhd"`
	Status   string    `json:"status"`
}

type SupplierInvoiceSummary struct {
	ID            string    `json:"id"`
	InvoiceNumber string    `json:"invoice_number"`
	InvoiceDate   time.Time `json:"invoice_date"`
	TotalBHD      float64   `json:"total_bhd"`
	Status        string    `json:"status"`
}

// GetSupplierFullProfile retrieves complete supplier profile
func (a *App) GetSupplierFullProfile(supplierID string) (SupplierFullProfile, error) {
	if err := a.requirePermission("suppliers:view"); err != nil {
		return SupplierFullProfile{}, err
	}
	profile := SupplierFullProfile{}

	if a.db == nil {
		return profile, fmt.Errorf("database not initialized")
	}

	// 1. Get supplier master
	var supplier SupplierMaster
	if err := a.db.Where("id = ?", supplierID).First(&supplier).Error; err != nil {
		return profile, fmt.Errorf("supplier not found: %v", err)
	}

	profile.ID = supplier.ID
	profile.SupplierCode = supplier.SupplierCode
	profile.SupplierName = supplier.SupplierName
	profile.SupplierType = supplier.SupplierType
	profile.TaxID = supplier.TaxID
	profile.Country = supplier.Country
	profile.Address = supplier.Address
	profile.PrimaryContact = supplier.PrimaryContact
	profile.Email = supplier.Email
	profile.Phone = supplier.Phone
	profile.BankName = supplier.BankName
	profile.AccountNumber = supplier.AccountNumber
	profile.IBAN = supplier.IBAN
	profile.SwiftCode = supplier.SwiftCode
	profile.Rating = supplier.Rating
	profile.LeadTimeDays = supplier.LeadTimeDays

	// Parse JSON arrays
	if supplier.BrandsHandled != "" {
		json.Unmarshal([]byte(supplier.BrandsHandled), &profile.BrandsHandled)
	}
	if supplier.ProductTypes != "" {
		json.Unmarshal([]byte(supplier.ProductTypes), &profile.ProductTypes)
	}

	// 2. Get PO metrics
	var pos []PurchaseOrder
	a.db.Where("supplier_id = ?", supplierID).Find(&pos)
	profile.TotalPOs = len(pos)
	var totalPurchases float64
	for _, po := range pos {
		totalPurchases += po.TotalBHD
	}
	profile.TotalPurchases = totalPurchases
	if profile.TotalPOs > 0 {
		profile.AvgPOValue = totalPurchases / float64(profile.TotalPOs)
	}

	// 3. Get recent POs
	var recentPOs []PurchaseOrder
	a.db.Where("supplier_id = ?", supplierID).Order("po_date DESC").Limit(5).Find(&recentPOs)
	poSummaries := make([]POSummary, 0, len(recentPOs))
	for _, po := range recentPOs {
		poSummaries = append(poSummaries, POSummary{
			ID:       po.ID,
			PONumber: po.PONumber,
			PODate:   po.PODate,
			TotalBHD: po.TotalBHD,
			Status:   po.Status,
		})
	}
	profile.RecentPOs = poSummaries

	// 4. Get supplier invoices
	var invoices []SupplierInvoice
	a.db.Where("supplier_id = ?", supplierID).Order("invoice_date DESC").Limit(5).Find(&invoices)
	invSummaries := make([]SupplierInvoiceSummary, 0, len(invoices))
	var outstanding, overdue float64
	for _, inv := range invoices {
		invSummaries = append(invSummaries, SupplierInvoiceSummary{
			ID:            inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			InvoiceDate:   inv.InvoiceDate,
			TotalBHD:      inv.TotalBHD,
			Status:        inv.Status,
		})
		if inv.Status != "Paid" {
			outstanding += inv.TotalBHD
			if time.Since(inv.DueDate).Hours()/24 > 30 {
				overdue += inv.TotalBHD
			}
		}
	}
	profile.RecentInvoices = invSummaries
	profile.OutstandingBHD = outstanding
	profile.OverdueBHD = overdue

	// 5. Get issues
	var issues []SupplierIssue
	a.db.Where("supplier_id = ?", supplierID).Order("created_at DESC").Find(&issues)
	profile.Issues = issues
	var openCount int
	var issueCost float64
	for _, issue := range issues {
		if issue.Status == "open" || issue.Status == "pending" {
			openCount++
		}
		issueCost += issue.CostBHD
	}
	profile.OpenIssues = openCount
	profile.IssueCost = issueCost

	// 6. Get notes
	var notes []EntityNote
	a.db.Where("entity_type = ? AND entity_id = ?", "supplier", supplierID).Order("created_at DESC").Find(&notes)
	profile.Notes = notes

	log.Printf("📦 Supplier Full Profile: %s (%s) - %d POs, %d issues",
		supplier.SupplierName, supplierID, profile.TotalPOs, len(issues))

	return profile, nil
}

// AddSupplierNote adds a note to a supplier
func (a *App) AddSupplierNote(supplierID string, noteType string, content string) error {
	if err := a.requirePermission("suppliers:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// P1 FIX: Validate note content
	if GlobalValidator != nil {
		if err := GlobalValidator.ValidateNoteInput(content); err != nil {
			return fmt.Errorf("invalid note content: %w", err)
		}
	}

	note := EntityNote{
		EntityType: "supplier",
		EntityID:   supplierID,
		NoteType:   noteType,
		Content:    content,
	}

	if err := a.db.Create(&note).Error; err != nil {
		return fmt.Errorf("failed to create note: %v", err)
	}

	log.Printf("📝 Added supplier note: %s (%s)", noteType, supplierID)
	return nil
}

// AddSupplierIssue adds an issue to a supplier
func (a *App) AddSupplierIssue(supplierID string, orderRef string, description string, costBHD float64) error {
	if err := a.requirePermission("suppliers:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	issue := SupplierIssue{
		SupplierID:  supplierID,
		OrderRef:    orderRef,
		Description: description,
		Status:      "open",
		CostBHD:     costBHD,
	}

	if err := a.db.Create(&issue).Error; err != nil {
		return fmt.Errorf("failed to create issue: %v", err)
	}

	log.Printf("⚠️ Added supplier issue: %s - %.3f BHD", supplierID, costBHD)
	return nil
}

// ResolveSupplierIssue marks an issue as resolved
func (a *App) ResolveSupplierIssue(issueID string, resolution string) error {
	if err := a.requirePermission("suppliers:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()
	if err := a.db.Model(&SupplierIssue{}).Where("id = ?", issueID).Updates(map[string]any{
		"status":      "resolved",
		"resolution":  resolution,
		"resolved_at": now,
	}).Error; err != nil {
		return fmt.Errorf("failed to resolve issue: %v", err)
	}

	log.Printf("✅ Resolved supplier issue: %s", issueID)
	return nil
}

// ============================================================================
// RFQ MANAGEMENT
// ============================================================================

// RFQData represents a simple RFQ for frontend submission
type RFQData struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RFQNumber string    `json:"rfq_number" gorm:"uniqueIndex;size:50"` // Business-friendly number like "1-26"
	RFQRef    string    `json:"rfq_ref" gorm:"size:100"`               // Customer/supplier reference entered by the sales team
	Client    string    `json:"client"`
	Project   string    `json:"project"`
	Value     float64   `json:"value"`
	Notes     string    `json:"notes"`
	Status    string    `json:"status" gorm:"default:'pending'"`
	Stage     string    `json:"stage" gorm:"index;size:50;default:'RFQ Received'"` // 9-stage pipeline
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// New fields for enhanced opportunity management
	DocumentHash   string `json:"document_hash" gorm:"size:64"`     // SHA-256 hash of source document for duplicate detection
	VisitLocations string `json:"visit_locations" gorm:"type:text"` // JSON array of site visit locations
	ProductDetails string `json:"product_details" gorm:"type:text"` // JSON array of product specifications
	SourceDocPath  string `json:"source_doc_path" gorm:"size:500"`  // Path to original document
}

// RFQComment stores comments/notes history for an RFQ (append-only log)
type RFQComment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RFQID     uint      `json:"rfq_id" gorm:"index"`
	Comment   string    `json:"comment" gorm:"type:text"`
	CreatedBy string    `json:"created_by" gorm:"size:100"`
	CreatedAt time.Time `json:"created_at"`
}

func (RFQComment) TableName() string { return "rfq_comments" }

// OpportunityComment stores the shared comment thread for canonical opportunities.
type OpportunityComment struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	OpportunityID string    `json:"opportunity_id" gorm:"index;size:36"`
	Comment       string    `json:"comment" gorm:"type:text"`
	CreatedBy     string    `json:"created_by" gorm:"size:100"`
	CreatedAt     time.Time `json:"created_at"`
}

func (OpportunityComment) TableName() string { return "opportunity_comments" }

// RFQUpdateRequest contains updatable fields for an RFQ
type RFQUpdateRequest struct {
	Client         string  `json:"client"`
	Project        string  `json:"project"`
	RFQRef         string  `json:"rfq_ref"`
	Value          float64 `json:"value"`
	Notes          string  `json:"notes"`
	Status         string  `json:"status"`
	VisitLocations string  `json:"visit_locations"`
	ProductDetails string  `json:"product_details"`
	DocumentHash   string  `json:"document_hash"`
	SourceDocPath  string  `json:"source_doc_path"`
}

// CreateRFQ creates a new RFQ from frontend form
