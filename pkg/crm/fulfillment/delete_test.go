package fulfillment

import (
	"strings"
	"testing"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/crm"
)

func crmDeliveryNote(t *testing.T, db *gorm.DB, number, status string) crm.DeliveryNote {
	t.Helper()
	dn := crm.DeliveryNote{DNNumber: number, Status: status}
	if err := db.Create(&dn).Error; err != nil {
		t.Fatalf("seed delivery note: %v", err)
	}
	return dn
}

func crmSerial(t *testing.T, db *gorm.DB, productID, serialNo, status, dnNumber string) crm.SerialNumber {
	t.Helper()
	serial := crm.SerialNumber{ProductID: productID, SerialNo: serialNo, Status: status, DNNumber: dnNumber}
	if err := db.Create(&serial).Error; err != nil {
		t.Fatalf("seed serial: %v", err)
	}
	return serial
}

func TestDeleteDeliveryNote_StatusGuardAndSerialRelease(t *testing.T) {
	db := serialsTestDB(t)

	dispatched := crmDeliveryNote(t, db, "DN-2026-0001", "Dispatched")
	err := DeleteDeliveryNote(db, dispatched.ID)
	if err == nil || !strings.Contains(err.Error(), "[DN_INVALID_STATUS]") {
		t.Fatalf("dispatched DN must be refused, got %v", err)
	}

	prepared := crmDeliveryNote(t, db, "DN-2026-0002", "Prepared")
	product := seedProduct(t, db)
	serial := crmSerial(t, db, product.ID, "SER-1", "Reserved", prepared.DNNumber)

	if err := DeleteDeliveryNote(db, prepared.ID); err != nil {
		t.Fatalf("prepared DN must delete, got %v", err)
	}

	var reloaded struct {
		Status   string
		DNNumber string
	}
	if err := db.Table("serial_numbers").Select("status, dn_number").
		Where("id = ?", serial.ID).Scan(&reloaded).Error; err != nil {
		t.Fatalf("reload serial: %v", err)
	}
	if reloaded.Status != "Available" || reloaded.DNNumber != "" {
		t.Fatalf("reserved serial must be released to Available, got %+v", reloaded)
	}
}
