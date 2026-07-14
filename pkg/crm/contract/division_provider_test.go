package contract

import (
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/overlay"
)

// testServiceWithOrders mirrors testService (service_test.go) but also
// migrates crm.Order and crm.CustomerMaster, which the division-provider
// resolution needs (the read-only FK join from Contract.OrderID -> Order).
func testServiceWithOrders(t *testing.T) *Service {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "contract-division.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Template{}, &Clause{}, &Contract{}, &crm.Order{}, &crm.CustomerMaster{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return New(db)
}

// TestResolveContractProviderProfile_UsesLinkedOrderDivision proves the
// CHAIN-GAP fix: a contract with no Division field resolves its provider
// identity through the linked order's division, so a Beacon-order contract
// prints Beacon's legal entity, not the default division's.
func TestResolveContractProviderProfile_UsesLinkedOrderDivision(t *testing.T) {
	svc := testServiceWithOrders(t)
	defaults := overlay.BuiltinDefaults()
	beacon := defaults.Profile(defaults.NormalizeDivisionName("Beacon Controls"))

	customer := &crm.CustomerMaster{BusinessName: "Beacon Test Customer"}
	if err := svc.db.Create(customer).Error; err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	order := &crm.Order{
		OrderNumber:  "ORD-26-100",
		CustomerID:   customer.ID,
		CustomerName: customer.BusinessName,
		Division:     "Beacon Controls",
	}
	if err := svc.db.Create(order).Error; err != nil {
		t.Fatalf("seed order: %v", err)
	}

	contract := &Contract{
		ContractNo:   "CON26/900",
		CustomerID:   customer.ID,
		CustomerName: customer.BusinessName,
		OrderID:      order.ID,
	}
	if err := svc.db.Create(contract).Error; err != nil {
		t.Fatalf("seed contract: %v", err)
	}

	got := svc.resolveContractProviderProfile(contract)

	if got.LegalName != "BEACON CONTROLS W.L.L." {
		t.Errorf("LegalName = %q, want %q", got.LegalName, "BEACON CONTROLS W.L.L.")
	}
	if got.VATNumber != "990000000000001" {
		t.Errorf("VATNumber = %q, want %q", got.VATNumber, "990000000000001")
	}
	if got.LegalName != beacon.LegalName || got.VATNumber != beacon.VATNumber {
		t.Errorf("resolved profile %+v does not match overlay Beacon profile %+v", got, beacon)
	}
}

// TestResolveContractProviderProfile_NoOrderFallsBackToDefault proves the
// fix is byte-identical to the previous behavior when a contract has no
// linked order: it still resolves to the default (Acme) division.
func TestResolveContractProviderProfile_NoOrderFallsBackToDefault(t *testing.T) {
	svc := testServiceWithOrders(t)
	defaults := overlay.BuiltinDefaults()
	acme := defaults.Profile(defaults.NormalizeDivisionName(""))

	customer := &crm.CustomerMaster{BusinessName: "Acme Test Customer"}
	if err := svc.db.Create(customer).Error; err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	contract := &Contract{
		ContractNo:   "CON26/901",
		CustomerID:   customer.ID,
		CustomerName: customer.BusinessName,
		// OrderID intentionally left empty.
	}
	if err := svc.db.Create(contract).Error; err != nil {
		t.Fatalf("seed contract: %v", err)
	}

	got := svc.resolveContractProviderProfile(contract)

	if got.LegalName != "ACME INSTRUMENTATION W.L.L" {
		t.Errorf("LegalName = %q, want %q", got.LegalName, "ACME INSTRUMENTATION W.L.L")
	}
	if got.VATNumber != "990000000000000" {
		t.Errorf("VATNumber = %q, want %q", got.VATNumber, "990000000000000")
	}
	if got.LegalName != acme.LegalName || got.VATNumber != acme.VATNumber {
		t.Errorf("resolved profile %+v does not match overlay default profile %+v", got, acme)
	}
}
