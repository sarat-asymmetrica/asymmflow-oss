package main

import (
	"fmt"

	"gorm.io/gorm"
)

// QueryOptimizer provides optimized query patterns
type QueryOptimizer struct {
	db    *gorm.DB
	cache *Cache
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *gorm.DB, cache *Cache) *QueryOptimizer {
	return &QueryOptimizer{
		db:    db,
		cache: cache,
	}
}

// --- Optimized List Operations (with limits and selects) ---

// ListCustomersOptimized returns customers with optimized query (limit, select)
func (q *QueryOptimizer) ListCustomersOptimized(limit int) ([]CustomerMaster, error) {
	// Check cache first
	cacheKey := CacheKeyCustomerList
	if q.cache != nil {
		if cached, ok := q.cache.Get(cacheKey); ok {
			return cached.([]CustomerMaster), nil
		}
	}

	var customers []CustomerMaster
	if limit <= 0 {
		limit = 1000 // Default limit
	}

	// Select only necessary columns for list view
	err := q.db.
		Select("id, customer_id, customer_code, business_name, city, country, payment_grade, customer_grade, total_orders_value, total_orders_count, outstanding_bhd, is_credit_blocked").
		Limit(limit).
		Order("business_name ASC").
		Find(&customers).Error

	if err != nil {
		return nil, err
	}

	// Cache result
	if q.cache != nil {
		q.cache.Set(cacheKey, customers, CacheTTLMedium)
	}

	return customers, nil
}

// ListSuppliersOptimized returns suppliers with optimized query (limit, select)
func (q *QueryOptimizer) ListSuppliersOptimized(limit int) ([]SupplierMaster, error) {
	// Check cache first
	cacheKey := CacheKeySupplierList
	if q.cache != nil {
		if cached, ok := q.cache.Get(cacheKey); ok {
			return cached.([]SupplierMaster), nil
		}
	}

	var suppliers []SupplierMaster
	if limit <= 0 {
		limit = 1000 // Default limit
	}

	// Select only necessary columns for list view
	err := q.db.
		Select("id, supplier_code, supplier_name, country, lead_time_days, supplier_type, rating, payment_terms").
		Limit(limit).
		Order("supplier_name ASC").
		Find(&suppliers).Error

	if err != nil {
		return nil, err
	}

	// Cache result
	if q.cache != nil {
		q.cache.Set(cacheKey, suppliers, CacheTTLMedium)
	}

	return suppliers, nil
}

// ListProductsOptimized returns products with optimized query (limit, select)
func (q *QueryOptimizer) ListProductsOptimized(limit int, activeOnly bool) ([]ProductMaster, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:active=%t", CacheKeyProductCatalog, activeOnly)
	if q.cache != nil {
		if cached, ok := q.cache.Get(cacheKey); ok {
			return cached.([]ProductMaster), nil
		}
	}

	var products []ProductMaster
	if limit <= 0 {
		limit = 1000 // Default limit
	}

	query := q.db.
		Select("id, product_code, product_name, product_category, supplier_code, standard_cost_bhd, standard_price_bhd, is_active, stock_quantity").
		Limit(limit).
		Order("product_name ASC")

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	err := query.Find(&products).Error
	if err != nil {
		return nil, err
	}

	// Cache result
	if q.cache != nil {
		q.cache.Set(cacheKey, products, CacheTTLLong)
	}

	return products, nil
}

// ListOrdersOptimized returns orders with optimized query (limit, no items preload by default)
func (q *QueryOptimizer) ListOrdersOptimized(limit int, status string) ([]Order, error) {
	var orders []Order
	if limit <= 0 {
		limit = 1000 // Default limit
	}

	query := q.db.
		Select("id, order_number, customer_id, customer_name, order_date, required_date, total_value_bhd, grand_total_bhd, status").
		Limit(limit).
		Order("order_date DESC")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	return orders, query.Find(&orders).Error
}

// ListInvoicesOptimized returns invoices with optimized query (limit, select)
func (q *QueryOptimizer) ListInvoicesOptimized(limit int, status string) ([]Invoice, error) {
	var invoices []Invoice
	if limit <= 0 {
		limit = 1000 // Default limit
	}

	query := q.db.
		Select("id, invoice_number, invoice_date, customer_id, customer_name, grand_total_bhd, status, outstanding_bhd, due_date").
		Limit(limit).
		Order("invoice_date DESC")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	return invoices, query.Find(&invoices).Error
}

// --- N+1 Prevention with Strategic Preloading ---

// GetOrderWithItems returns order with items preloaded (avoids N+1)
func (q *QueryOptimizer) GetOrderWithItems(orderID string) (*Order, error) {
	var order Order
	err := q.db.
		Preload("Items"). // Single query for all items
		Where("id = ?", orderID).
		First(&order).Error

	return &order, err
}

// GetInvoiceWithItems returns invoice with items preloaded (avoids N+1)
func (q *QueryOptimizer) GetInvoiceWithItems(invoiceID string) (*Invoice, error) {
	var invoice Invoice
	err := q.db.
		Preload("Items"). // Single query for all items
		Where("id = ?", invoiceID).
		First(&invoice).Error

	return &invoice, err
}

// GetOfferWithItems returns offer with items preloaded (avoids N+1)
func (q *QueryOptimizer) GetOfferWithItems(offerID string) (*Offer, error) {
	var offer Offer
	err := q.db.
		Preload("Items"). // Single query for all items
		Where("id = ?", offerID).
		First(&offer).Error

	return &offer, err
}

// GetPurchaseOrderWithItems returns PO with items preloaded (avoids N+1)
func (q *QueryOptimizer) GetPurchaseOrderWithItems(poID string) (*PurchaseOrder, error) {
	var po PurchaseOrder
	err := q.db.
		Preload("Items"). // Single query for all items
		Where("id = ?", poID).
		First(&po).Error

	return &po, err
}

// GetCustomerWithContacts returns customer with contacts preloaded (avoids N+1)
func (q *QueryOptimizer) GetCustomerWithContacts(customerID string) (*CustomerMaster, []CustomerContact, error) {
	var customer CustomerMaster
	var contacts []CustomerContact

	// Get customer
	if err := q.db.Where("id = ?", customerID).First(&customer).Error; err != nil {
		return nil, nil, err
	}

	// Get contacts in single query
	if err := q.db.Where("customer_id = ?", customerID).Find(&contacts).Error; err != nil {
		return &customer, nil, err
	}

	return &customer, contacts, nil
}

// GetSupplierWithContacts returns supplier with contacts preloaded (avoids N+1)
func (q *QueryOptimizer) GetSupplierWithContacts(supplierID string) (*SupplierMaster, []SupplierContact, error) {
	var supplier SupplierMaster
	var contacts []SupplierContact

	// Get supplier
	if err := q.db.Where("id = ?", supplierID).First(&supplier).Error; err != nil {
		return nil, nil, err
	}

	// Get contacts in single query
	if err := q.db.Where("supplier_id = ?", supplierID).Find(&contacts).Error; err != nil {
		return &supplier, nil, err
	}

	return &supplier, contacts, nil
}

// --- Dashboard Query Optimizations ---

// GetDashboardStatsOptimized returns cached dashboard statistics
func (q *QueryOptimizer) GetDashboardStatsOptimized() (map[string]any, error) {
	// Check cache first
	cacheKey := CacheKeyDashboardStats
	if q.cache != nil {
		if cached, ok := q.cache.Get(cacheKey); ok {
			return cached.(map[string]any), nil
		}
	}

	stats := make(map[string]any)

	// Customer count
	var customerCount int64
	q.db.Model(&CustomerMaster{}).Count(&customerCount)
	stats["customer_count"] = customerCount

	// Supplier count
	var supplierCount int64
	q.db.Model(&SupplierMaster{}).Count(&supplierCount)
	stats["supplier_count"] = supplierCount

	// Active orders count
	var activeOrderCount int64
	q.db.Model(&Order{}).Where("status NOT IN ?", []string{"Delivered", "Cancelled"}).Count(&activeOrderCount)
	stats["active_order_count"] = activeOrderCount

	// Pending invoices count
	var pendingInvoiceCount int64
	q.db.Model(&Invoice{}).Where("status IN ?", []string{"Sent", "Overdue"}).Count(&pendingInvoiceCount)
	stats["pending_invoice_count"] = pendingInvoiceCount

	// Total outstanding AR
	var totalOutstanding float64
	q.db.Model(&Invoice{}).Where("status != ?", "Paid").Select("SUM(outstanding_bhd)").Scan(&totalOutstanding)
	stats["total_outstanding_bhd"] = totalOutstanding

	// Cache result
	if q.cache != nil {
		q.cache.Set(cacheKey, stats, CacheTTLShort)
	}

	return stats, nil
}

// --- App Integration Methods ---

// ListCustomersOptimized returns optimized customer list
func (a *App) ListCustomersOptimized(limit int) ([]CustomerMaster, error) {
	optimizer := NewQueryOptimizer(a.db, a.cache)
	return optimizer.ListCustomersOptimized(limit)
}

// ListSuppliersOptimized returns optimized supplier list
func (a *App) ListSuppliersOptimized(limit int) ([]SupplierMaster, error) {
	optimizer := NewQueryOptimizer(a.db, a.cache)
	return optimizer.ListSuppliersOptimized(limit)
}

// ListProductsOptimized returns optimized product list
func (a *App) ListProductsOptimized(limit int, activeOnly bool) ([]ProductMaster, error) {
	optimizer := NewQueryOptimizer(a.db, a.cache)
	return optimizer.ListProductsOptimized(limit, activeOnly)
}

// GetOrderWithItems returns order with items preloaded
func (a *App) GetOrderWithItems(orderID string) (*Order, error) {
	optimizer := NewQueryOptimizer(a.db, a.cache)
	return optimizer.GetOrderWithItems(orderID)
}

// GetInvoiceWithItems returns invoice with items preloaded
func (a *App) GetInvoiceWithItems(invoiceID string) (*Invoice, error) {
	// Wave 8 P0: unauthenticated financial-data read in deployed PH is gated invoices:view.
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	optimizer := NewQueryOptimizer(a.db, a.cache)
	return optimizer.GetInvoiceWithItems(invoiceID)
}

// GetDashboardStatsOptimized returns cached dashboard stats
func (a *App) GetDashboardStatsOptimized() (map[string]any, error) {
	optimizer := NewQueryOptimizer(a.db, a.cache)
	return optimizer.GetDashboardStatsOptimized()
}

// InvalidateCache invalidates cache entries by pattern
func (a *App) InvalidateCache(pattern string) {
	if a.cache != nil {
		if pattern == "" {
			a.cache.Clear()
		} else {
			a.cache.InvalidatePattern(pattern)
		}
	}
}

// GetCacheStats returns cache statistics
func (a *App) GetCacheStats() map[string]any {
	if a.cache == nil {
		return map[string]any{
			"enabled": false,
		}
	}

	stats := a.cache.Stats()
	stats["enabled"] = true
	return stats
}
