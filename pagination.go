package main

import (
	"fmt"

	"gorm.io/gorm"
)

// PaginationResult contains paginated data with metadata
type PaginationResult struct {
	Data       any   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// PaginationParams contains pagination parameters
type PaginationParams struct {
	Page     int
	PageSize int
	Limit    int
	Offset   int
}

// NewPaginationParams creates pagination parameters with defaults
func NewPaginationParams(page, pageSize int) PaginationParams {
	// Apply defaults
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50 // Default page size
	}
	if pageSize > 200 {
		pageSize = 200 // Max page size (consistent with other listing functions)
	}

	offset := (page - 1) * pageSize

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Limit:    pageSize,
		Offset:   offset,
	}
}

// Paginate applies pagination to a GORM query
func Paginate(params PaginationParams) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(params.Offset).Limit(params.Limit)
	}
}

// BuildPaginationResult builds a pagination result with total count
func BuildPaginationResult(data any, total int64, params PaginationParams) PaginationResult {
	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))
	if totalPages < 1 {
		totalPages = 1
	}

	return PaginationResult{
		Data:       data,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}
}

// --- App Integration Methods ---

// ListCustomersPaginated returns paginated customer list
func (a *App) ListCustomersPaginated(page, pageSize int) (PaginationResult, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return PaginationResult{}, err
	}
	if a.db == nil {
		return PaginationResult{}, fmt.Errorf("database not initialized")
	}
	params := NewPaginationParams(page, pageSize)

	var customers []CustomerMaster
	var total int64

	// Count total
	if err := a.db.Model(&CustomerMaster{}).Count(&total).Error; err != nil {
		return PaginationResult{}, err
	}

	// Get page
	if err := a.db.Scopes(Paginate(params)).
		Order("business_name ASC").
		Find(&customers).Error; err != nil {
		return PaginationResult{}, err
	}

	return BuildPaginationResult(customers, total, params), nil
}

// ListSuppliersPaginated returns paginated supplier list
func (a *App) ListSuppliersPaginated(page, pageSize int) (PaginationResult, error) {
	if err := a.requirePermission("suppliers:view"); err != nil {
		return PaginationResult{}, err
	}
	if a.db == nil {
		return PaginationResult{}, fmt.Errorf("database not initialized")
	}
	params := NewPaginationParams(page, pageSize)

	var suppliers []SupplierMaster
	var total int64

	// Count total
	if err := a.db.Model(&SupplierMaster{}).Count(&total).Error; err != nil {
		return PaginationResult{}, err
	}

	// Get page
	if err := a.db.Scopes(Paginate(params)).
		Order("supplier_name ASC").
		Find(&suppliers).Error; err != nil {
		return PaginationResult{}, err
	}

	return BuildPaginationResult(suppliers, total, params), nil
}

// ListOrdersPaginated returns paginated order list
func (a *App) ListOrdersPaginated(page, pageSize int, status string) (PaginationResult, error) {
	if err := a.requirePermission("orders:view"); err != nil {
		return PaginationResult{}, err
	}
	if a.db == nil {
		return PaginationResult{}, fmt.Errorf("database not initialized")
	}
	params := NewPaginationParams(page, pageSize)

	var orders []Order
	var total int64

	query := a.db.Model(&Order{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return PaginationResult{}, err
	}

	// Get page with items preloaded
	if err := query.Scopes(Paginate(params)).
		Preload("Items").
		Order("order_date DESC").
		Find(&orders).Error; err != nil {
		return PaginationResult{}, err
	}

	return BuildPaginationResult(orders, total, params), nil
}

// ListInvoicesPaginated returns paginated invoice list
func (a *App) ListInvoicesPaginated(page, pageSize int, status string) (PaginationResult, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return PaginationResult{}, err
	}
	if a.db == nil {
		return PaginationResult{}, fmt.Errorf("database not initialized")
	}
	params := NewPaginationParams(page, pageSize)

	var invoices []Invoice
	var total int64

	query := a.db.Model(&Invoice{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return PaginationResult{}, err
	}

	// Get page
	if err := query.Scopes(Paginate(params)).
		Order("invoice_date DESC").
		Find(&invoices).Error; err != nil {
		return PaginationResult{}, err
	}

	return BuildPaginationResult(invoices, total, params), nil
}

// ListPurchaseOrdersPaginated returns paginated purchase order list
func (a *App) ListPurchaseOrdersPaginated(page, pageSize int, status string) (PaginationResult, error) {
	if err := a.requirePermission("po:view"); err != nil {
		return PaginationResult{}, err
	}
	if a.db == nil {
		return PaginationResult{}, fmt.Errorf("database not initialized")
	}
	params := NewPaginationParams(page, pageSize)

	var pos []PurchaseOrder
	var total int64

	query := a.db.Model(&PurchaseOrder{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return PaginationResult{}, err
	}

	// Get page with items preloaded
	if err := query.Scopes(Paginate(params)).
		Preload("Items").
		Order("po_date DESC").
		Find(&pos).Error; err != nil {
		return PaginationResult{}, err
	}

	return BuildPaginationResult(pos, total, params), nil
}

// ListOffersPaginated returns paginated offer list
func (a *App) ListOffersPaginated(page, pageSize int, stage string) (PaginationResult, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return PaginationResult{}, err
	}
	if a.db == nil {
		return PaginationResult{}, fmt.Errorf("database not initialized")
	}
	params := NewPaginationParams(page, pageSize)

	var offers []Offer
	var total int64

	query := a.db.Model(&Offer{})
	if stage != "" {
		query = query.Where("stage = ?", stage)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return PaginationResult{}, err
	}

	// Get page
	if err := query.Scopes(Paginate(params)).
		Order("quotation_date DESC").
		Find(&offers).Error; err != nil {
		return PaginationResult{}, err
	}

	return BuildPaginationResult(offers, total, params), nil
}

// ListProductsPaginated returns paginated product list
func (a *App) ListProductsPaginated(page, pageSize int, category string, activeOnly bool) (PaginationResult, error) {
	if err := a.requirePermission("products:view"); err != nil {
		return PaginationResult{}, err
	}
	if a.db == nil {
		return PaginationResult{}, fmt.Errorf("database not initialized")
	}
	params := NewPaginationParams(page, pageSize)

	var products []ProductMaster
	var total int64

	query := a.db.Model(&ProductMaster{})
	if category != "" {
		query = query.Where("product_category = ?", category)
	}
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return PaginationResult{}, err
	}

	// Get page
	if err := query.Scopes(Paginate(params)).
		Order("product_name ASC").
		Find(&products).Error; err != nil {
		return PaginationResult{}, err
	}

	return BuildPaginationResult(products, total, params), nil
}
