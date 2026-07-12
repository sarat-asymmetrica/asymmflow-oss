package main

import (
	"fmt"

	"gorm.io/gorm"
)

// BatchOperations provides efficient batch insert/update operations
type BatchOperations struct {
	db        *gorm.DB
	batchSize int
}

// NewBatchOperations creates a new batch operations helper
func NewBatchOperations(db *gorm.DB) *BatchOperations {
	return &BatchOperations{
		db:        db,
		batchSize: 100, // Default batch size
	}
}

// SetBatchSize configures the batch size for operations
func (b *BatchOperations) SetBatchSize(size int) {
	if size > 0 {
		b.batchSize = size
	}
}

// --- Batch Create Operations ---

// BatchCreateInvoiceItems creates invoice items in batches
func (b *BatchOperations) BatchCreateInvoiceItems(items []DBInvoiceItem) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) > maxBatchSize {
		return fmt.Errorf("batch create size %d exceeds maximum of %d", len(items), maxBatchSize)
	}
	return b.db.CreateInBatches(&items, b.batchSize).Error
}

// BatchCreateOfferItems creates offer items in batches
func (b *BatchOperations) BatchCreateOfferItems(items []OfferItem) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) > maxBatchSize {
		return fmt.Errorf("batch create size %d exceeds maximum of %d", len(items), maxBatchSize)
	}
	return b.db.CreateInBatches(&items, b.batchSize).Error
}

// BatchCreateOrderItems creates order items in batches
func (b *BatchOperations) BatchCreateOrderItems(items []OrderItem) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) > maxBatchSize {
		return fmt.Errorf("batch create size %d exceeds maximum of %d", len(items), maxBatchSize)
	}
	return b.db.CreateInBatches(&items, b.batchSize).Error
}

// BatchCreatePOItems creates purchase order items in batches
func (b *BatchOperations) BatchCreatePOItems(items []PurchaseOrderItem) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) > maxBatchSize {
		return fmt.Errorf("batch create size %d exceeds maximum of %d", len(items), maxBatchSize)
	}
	return b.db.CreateInBatches(&items, b.batchSize).Error
}

// BatchCreateGRNItems creates GRN items in batches
func (b *BatchOperations) BatchCreateGRNItems(items []GRNItem) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) > maxBatchSize {
		return fmt.Errorf("batch create size %d exceeds maximum of %d", len(items), maxBatchSize)
	}
	return b.db.CreateInBatches(&items, b.batchSize).Error
}

// BatchCreateDeliveryNoteItems creates delivery note items in batches
func (b *BatchOperations) BatchCreateDeliveryNoteItems(items []DeliveryNoteItem) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) > maxBatchSize {
		return fmt.Errorf("batch create size %d exceeds maximum of %d", len(items), maxBatchSize)
	}
	return b.db.CreateInBatches(&items, b.batchSize).Error
}

// BatchCreateCustomers creates customers in batches
func (b *BatchOperations) BatchCreateCustomers(customers []CustomerMaster) error {
	if len(customers) == 0 {
		return nil
	}
	if len(customers) > maxBatchSize {
		return fmt.Errorf("batch create size %d exceeds maximum of %d", len(customers), maxBatchSize)
	}
	return b.db.CreateInBatches(&customers, b.batchSize).Error
}

// BatchCreateSuppliers creates suppliers in batches
func (b *BatchOperations) BatchCreateSuppliers(suppliers []SupplierMaster) error {
	if len(suppliers) == 0 {
		return nil
	}
	if len(suppliers) > maxBatchSize {
		return fmt.Errorf("batch create size %d exceeds maximum of %d", len(suppliers), maxBatchSize)
	}
	return b.db.CreateInBatches(&suppliers, b.batchSize).Error
}

// BatchCreateProducts creates products in batches
func (b *BatchOperations) BatchCreateProducts(products []ProductMaster) error {
	if len(products) == 0 {
		return nil
	}
	if len(products) > maxBatchSize {
		return fmt.Errorf("batch create size %d exceeds maximum of %d", len(products), maxBatchSize)
	}
	return b.db.CreateInBatches(&products, b.batchSize).Error
}

// maxBatchSize is the safety cap for batch operations to prevent unbounded mass-mutations
const maxBatchSize = 500

// --- Batch Update Operations ---

// BatchUpdateOrderStatus updates order status for multiple orders
func (b *BatchOperations) BatchUpdateOrderStatus(orderIDs []string, status string) error {
	if len(orderIDs) == 0 {
		return nil
	}
	if len(orderIDs) > maxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum of %d", len(orderIDs), maxBatchSize)
	}

	return b.db.Model(&Order{}).
		Where("id IN ?", orderIDs).
		Update("status", status).Error
}

// BatchUpdateInvoiceStatus updates invoice status for multiple invoices
func (b *BatchOperations) BatchUpdateInvoiceStatus(invoiceIDs []string, status string) error {
	if len(invoiceIDs) == 0 {
		return nil
	}
	if len(invoiceIDs) > maxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum of %d", len(invoiceIDs), maxBatchSize)
	}

	return b.db.Model(&Invoice{}).
		Where("id IN ?", invoiceIDs).
		Update("status", status).Error
}

// BatchUpdateCustomerGrade updates customer grade for multiple customers
func (b *BatchOperations) BatchUpdateCustomerGrade(customerIDs []string, grade string) error {
	if len(customerIDs) == 0 {
		return nil
	}
	if len(customerIDs) > maxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum of %d", len(customerIDs), maxBatchSize)
	}

	return b.db.Model(&CustomerMaster{}).
		Where("id IN ?", customerIDs).
		Update("payment_grade", grade).Error
}

// BatchUpdateSupplierRating updates supplier rating for multiple suppliers
func (b *BatchOperations) BatchUpdateSupplierRating(supplierIDs []string, rating int) error {
	if len(supplierIDs) == 0 {
		return nil
	}
	if len(supplierIDs) > maxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum of %d", len(supplierIDs), maxBatchSize)
	}

	return b.db.Model(&SupplierMaster{}).
		Where("id IN ?", supplierIDs).
		Update("rating", rating).Error
}

// --- Batch Delete Operations (Soft Delete) ---

// BatchDeleteOrders soft deletes multiple orders
func (b *BatchOperations) BatchDeleteOrders(orderIDs []string) error {
	if len(orderIDs) == 0 {
		return nil
	}
	if len(orderIDs) > maxBatchSize {
		return fmt.Errorf("batch delete size %d exceeds maximum of %d", len(orderIDs), maxBatchSize)
	}
	return b.db.Where("id IN ?", orderIDs).Delete(&Order{}).Error
}

// BatchDeleteInvoices soft deletes multiple invoices
func (b *BatchOperations) BatchDeleteInvoices(invoiceIDs []string) error {
	if len(invoiceIDs) == 0 {
		return nil
	}
	if len(invoiceIDs) > maxBatchSize {
		return fmt.Errorf("batch delete size %d exceeds maximum of %d", len(invoiceIDs), maxBatchSize)
	}
	return b.db.Where("id IN ?", invoiceIDs).Delete(&Invoice{}).Error
}

// --- App Integration ---

// BatchCreateInvoiceItems creates invoice items in batches (app method)
func (a *App) BatchCreateInvoiceItems(items []DBInvoiceItem) error {
	if err := a.requirePermission(PermInvoicesCreate); err != nil {
		return err
	}

	if len(items) > maxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum of %d items", len(items), maxBatchSize)
	}

	batch := NewBatchOperations(a.db)
	if err := batch.BatchCreateInvoiceItems(items); err != nil {
		return fmt.Errorf("batch create invoice items failed: %w", err)
	}

	// Invalidate invoice cache
	if a.cache != nil {
		a.cache.InvalidatePattern(CacheKeyInvoicePrefix)
	}

	return nil
}

// BatchUpdateOrderStatus updates order status for multiple orders (app method)
func (a *App) BatchUpdateOrderStatus(orderIDs []string, status string) error {
	if err := a.requirePermission(PermOrdersCreate); err != nil {
		return err
	}

	if len(orderIDs) > maxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum of %d items", len(orderIDs), maxBatchSize)
	}

	batch := NewBatchOperations(a.db)
	if err := batch.BatchUpdateOrderStatus(orderIDs, status); err != nil {
		return fmt.Errorf("batch update order status failed: %w", err)
	}

	// Invalidate order cache
	if a.cache != nil {
		a.cache.InvalidatePattern(CacheKeyOrderPrefix)
	}

	return nil
}

// BatchUpdateCustomerGrade updates customer grade for multiple customers (app method)
func (a *App) BatchUpdateCustomerGrade(customerIDs []string, grade string) error {
	if err := a.requirePermission(PermCustomersEdit); err != nil {
		return err
	}

	if len(customerIDs) > maxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum of %d items", len(customerIDs), maxBatchSize)
	}

	batch := NewBatchOperations(a.db)
	if err := batch.BatchUpdateCustomerGrade(customerIDs, grade); err != nil {
		return fmt.Errorf("batch update customer grade failed: %w", err)
	}

	// Invalidate customer cache
	if a.cache != nil {
		a.cache.InvalidatePattern(CacheKeyCustomerPrefix)
		a.cache.Delete(CacheKeyCustomerList)
	}

	return nil
}
