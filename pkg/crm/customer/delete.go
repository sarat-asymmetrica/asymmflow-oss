// Relationship deletions (customers, suppliers, and their contacts),
// moved from the trading root in Wave 6 (Mission A.2). The host keeps
// the delete guard and RBAC checks; existence verification and the
// soft deletes live here.
package customer

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/kernel/apperr"
)

// liveCount counts non-soft-deleted rows referencing a party. Finance-owned
// tables are counted by name so this package does not import pkg/finance;
// the deleted_at filter mirrors what gorm.Model applies automatically.
func liveCount(db *gorm.DB, table, fkColumn, id string) int64 {
	var n int64
	db.Table(table).Where(fkColumn+" = ? AND deleted_at IS NULL", id).Count(&n)
	return n
}

// DeleteCustomer soft-deletes a customer after verifying it exists and has
// no transactional or pipeline children (PC-D1: the integrity guard lives in
// the engine so it protects every caller, including approved delete requests).
func DeleteCustomer(db *gorm.DB, id string) error {
	if db == nil {
		return apperr.New("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Verify customer exists before deletion
	var customer crm.CustomerMaster
	if err := db.First(&customer, "id = ?", id).Error; err != nil {
		return apperr.New("CUSTOMER_NOT_FOUND", "Customer not found", err.Error())
	}

	// CHILD-RECORD SAFETY GUARD: only redundant/empty customers may be deleted.
	// Contacts-only prospects are intentionally NOT counted — they stay deletable.
	var nOrders, nInvoices, nOffers, nOpps int64
	db.Model(&crm.Order{}).Where("customer_id = ?", id).Count(&nOrders)
	nInvoices = liveCount(db, "invoices", "customer_id", id)
	db.Model(&crm.Offer{}).Where("customer_id = ?", id).Count(&nOffers)
	db.Model(&crm.Opportunity{}).Where("customer_id = ?", id).Count(&nOpps)
	if nOrders+nInvoices+nOffers+nOpps > 0 {
		return apperr.New("CUSTOMER_HAS_LINKED_RECORDS", fmt.Sprintf("Cannot delete %s: %d order(s), %d invoice(s), %d offer(s), %d opportunit(ies) linked. Only redundant/empty records can be deleted — merge or reassign first.", customer.BusinessName, nOrders, nInvoices, nOffers, nOpps), "")
	}

	if err := db.Delete(&crm.CustomerMaster{}, "id = ?", id).Error; err != nil {
		return apperr.New("DB_DELETE_FAILED", "Failed to delete customer", err.Error())
	}

	log.Printf("🗑️ Customer deleted: %s (%s)", customer.BusinessName, id)
	return nil
}

// DeleteSupplier soft-deletes a supplier after verifying it exists.
func DeleteSupplier(db *gorm.DB, id string) error {
	if db == nil {
		return apperr.New("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Verify supplier exists before deletion
	var supplier crm.SupplierMaster
	if err := db.First(&supplier, "id = ?", id).Error; err != nil {
		return apperr.New("SUPPLIER_NOT_FOUND", "Supplier not found", err.Error())
	}

	// CHILD-RECORD SAFETY GUARD: only redundant/empty suppliers may be deleted.
	var nPOs, nSInv, nSPay int64
	db.Model(&crm.PurchaseOrder{}).Where("supplier_id = ?", id).Count(&nPOs)
	nSInv = liveCount(db, "supplier_invoices", "supplier_id", id)
	nSPay = liveCount(db, "supplier_payments", "supplier_id", id)
	if nPOs+nSInv+nSPay > 0 {
		return apperr.New("SUPPLIER_HAS_LINKED_RECORDS", fmt.Sprintf("Cannot delete %s: %d purchase order(s), %d supplier invoice(s), %d payment(s) linked. Only redundant/empty records can be deleted — merge or reassign first.", supplier.SupplierName, nPOs, nSInv, nSPay), "")
	}

	if err := db.Delete(&crm.SupplierMaster{}, "id = ?", id).Error; err != nil {
		return apperr.New("DB_DELETE_FAILED", "Failed to delete supplier", err.Error())
	}

	log.Printf("🗑️ Supplier deleted: %s (%s)", supplier.SupplierName, id)
	return nil
}

// DeleteCustomerContact removes a customer contact.
func DeleteCustomerContact(db *gorm.DB, contactID string) error {
	if db == nil {
		return apperr.New("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	return db.Delete(&crm.CustomerContact{}, "id = ?", contactID).Error
}

// DeleteSupplierContact removes a supplier contact.
func DeleteSupplierContact(db *gorm.DB, contactID string) error {
	if db == nil {
		return apperr.New("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	return db.Delete(&crm.SupplierContact{}, "id = ?", contactID).Error
}
