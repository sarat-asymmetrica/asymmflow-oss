package main

import (
	"fmt"
	"time"
)

// =============================================================================
// SERIAL NUMBER TRACKING SERVICE (Phase 23)
//
// Provides full lifecycle tracking for serialized process instrumentation:
//   PO Receipt (GRN) → Inventory → Dispatch (DN) → Delivery → Invoice
//
// Government clients (NPC, Gulf Smelting, NGA) require serial-level traceability.
//
// Wave 5 A.1: the lifecycle logic lives in pkg/crm/fulfillment (Serials).
// These delegates keep the Wails binding surface and the RBAC guards — auth
// is a hub and stays with the host (W4-D9).
// =============================================================================

func (a *App) serialsGuarded(permission string) (*App, error) {
	if err := a.requirePermission(permission); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return a, nil
}

// RegisterSerials creates serial number records for a product (manual registration)
func (a *App) RegisterSerials(productID string, serialNos []string) ([]SerialNumber, error) {
	if _, err := a.serialsGuarded("grn:create"); err != nil {
		return nil, err
	}
	return a.serialService().Register(productID, serialNos)
}

// GetSerialByNumber retrieves a serial number record by its serial number string
func (a *App) GetSerialByNumber(serialNo string) (SerialNumber, error) {
	if _, err := a.serialsGuarded("grn:view"); err != nil {
		return SerialNumber{}, err
	}
	return a.serialService().ByNumber(serialNo)
}

// SearchSerials searches serial numbers by partial match on serial_no, product_code, or customer_name
func (a *App) SearchSerials(query string, limit int) ([]SerialNumber, error) {
	if _, err := a.serialsGuarded("grn:view"); err != nil {
		return nil, err
	}
	return a.serialService().Search(query, limit)
}

// GetSerialsByProduct returns all serial numbers for a specific product
func (a *App) GetSerialsByProduct(productID string) ([]SerialNumber, error) {
	if _, err := a.serialsGuarded("grn:view"); err != nil {
		return nil, err
	}
	return a.serialService().ByProduct(productID)
}

// GetSerialsByCustomer returns all serial numbers delivered to a specific customer
func (a *App) GetSerialsByCustomer(customerID string) ([]SerialNumber, error) {
	if _, err := a.serialsGuarded("grn:view"); err != nil {
		return nil, err
	}
	return a.serialService().ByCustomer(customerID)
}

// GetAvailableSerials returns serial numbers with status "Available" for a product
func (a *App) GetAvailableSerials(productID string) ([]SerialNumber, error) {
	if _, err := a.serialsGuarded("grn:view"); err != nil {
		return nil, err
	}
	return a.serialService().Available(productID)
}

// UpdateSerialWarranty sets warranty duration and calculates end date
func (a *App) UpdateSerialWarranty(serialID string, warrantyMonths int) error {
	if _, err := a.serialsGuarded("grn:create"); err != nil {
		return err
	}
	return a.serialService().UpdateWarranty(serialID, warrantyMonths)
}

// AttachCalibrationCert sets calibration certificate path and date on a serial
func (a *App) AttachCalibrationCert(serialID, certPath string) error {
	if _, err := a.serialsGuarded("grn:create"); err != nil {
		return err
	}
	return a.serialService().AttachCalibrationCert(serialID, certPath)
}

// GetSerialsForInvoiceItem returns serials on a specific invoice for a specific product
func (a *App) GetSerialsForInvoiceItem(invoiceID, productID string) ([]SerialNumber, error) {
	if _, err := a.serialsGuarded("grn:view"); err != nil {
		return nil, err
	}
	return a.serialService().ForInvoiceItem(invoiceID, productID)
}

// GetRecentlyDeliveredSerials returns the most recently delivered serials
// (status Delivered/Signed), most recent first. B10-4: gives SerialTraceScreen
// a non-blank default view on mount instead of requiring a search first.
func (a *App) GetRecentlyDeliveredSerials(limit int) ([]SerialNumber, error) {
	app, err := a.serialsGuarded("grn:view")
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var serials []SerialNumber
	if err := app.db.Where("status IN ?", []string{"Delivered", "Signed"}).
		Order("warranty_start_date DESC, updated_at DESC").
		Limit(limit).
		Find(&serials).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve recently delivered serials: %w", err)
	}
	return serials, nil
}

// =============================================================================
// LIFECYCLE FUNCTIONS (called by GRN/DN services)
// =============================================================================

// assignSerialsToGRN creates serial records when goods are received against a PO
func (a *App) assignSerialsToGRN(grnItemID, grnNumber, poID, poNumber string, productID, productCode string, serialNos []string, receivedDate time.Time) error {
	return a.serialService().AssignToGRN(grnItemID, grnNumber, poID, poNumber, productID, productCode, serialNos, receivedDate)
}

// allocateSerialsToDN reserves serials for a delivery note
func (a *App) allocateSerialsToDN(dnItemID, dnNumber, customerID, customerName string, serialNos []string) error {
	return a.serialService().AllocateToDN(dnItemID, dnNumber, customerID, customerName, serialNos)
}

// markSerialsShipped updates serials on a DN to "Shipped" status
func (a *App) markSerialsShipped(dnNumber string) error {
	return a.serialService().MarkShipped(dnNumber)
}

// markSerialsDelivered updates serials on a DN to "Delivered" and sets warranty start
func (a *App) markSerialsDelivered(dnID string) error {
	return a.serialService().MarkDelivered(dnID)
}

// linkSerialsToInvoice stamps invoice ID on all serials from a delivery note
func (a *App) linkSerialsToInvoice(invoiceID, invoiceNumber, dnID string) error {
	return a.serialService().LinkToInvoice(invoiceID, invoiceNumber, dnID)
}
