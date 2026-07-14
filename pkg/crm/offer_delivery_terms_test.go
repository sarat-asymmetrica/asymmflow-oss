package crm

import (
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/ncruces/go-sqlite3/embed"
)

// deliveryTermsTestDB opens a pure-Go sqlite test DB and migrates Offer (mirrors
// the pattern in pkg/crm/supplierlink/supplierlink_test.go).
func deliveryTermsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "offer_delivery_terms.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Offer{}, &OfferItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

// TestOfferBeforeCreate_ComposesDivisionDeliveryTerms proves the DI seam: an
// offer created with an empty DeliveryTerms gets it composed from its own
// division (Wave 12.5 B3), not the hardcoded default-division column default.
// It also proves ID minting still fires through the shadowed Base.BeforeCreate.
func TestOfferBeforeCreate_ComposesDivisionDeliveryTerms(t *testing.T) {
	db := deliveryTermsTestDB(t)

	ComposeOfferDeliveryTerms = func(division string) string {
		return "DAP Bahrain at your store or " + division
	}
	t.Cleanup(func() { ComposeOfferDeliveryTerms = nil })

	offer := Offer{
		OfferNumber: "OFF-B3-001",
		Division:    "Beacon Controls",
		Stage:       "RFQ",
	}
	if err := db.Create(&offer).Error; err != nil {
		t.Fatalf("create offer: %v", err)
	}

	if offer.ID == "" {
		t.Fatal("expected ID to be minted by Base.BeforeCreate through the shadowed Offer.BeforeCreate hook")
	}
	if want := "DAP Bahrain at your store or Beacon Controls"; offer.DeliveryTerms != want {
		t.Fatalf("DeliveryTerms = %q, want %q", offer.DeliveryTerms, want)
	}
}

// TestOfferBeforeCreate_DoesNotClobberExplicitDeliveryTerms proves the hook only
// fills DeliveryTerms when empty — an explicit value passes through untouched.
func TestOfferBeforeCreate_DoesNotClobberExplicitDeliveryTerms(t *testing.T) {
	db := deliveryTermsTestDB(t)

	ComposeOfferDeliveryTerms = func(division string) string {
		return "DAP Bahrain at your store or " + division
	}
	t.Cleanup(func() { ComposeOfferDeliveryTerms = nil })

	offer := Offer{
		OfferNumber:   "OFF-B3-002",
		Division:      "Beacon Controls",
		Stage:         "RFQ",
		DeliveryTerms: "Ex-works Manama",
	}
	if err := db.Create(&offer).Error; err != nil {
		t.Fatalf("create offer: %v", err)
	}

	if offer.ID == "" {
		t.Fatal("expected ID to be minted")
	}
	if want := "Ex-works Manama"; offer.DeliveryTerms != want {
		t.Fatalf("explicit DeliveryTerms must not be clobbered: got %q, want %q", offer.DeliveryTerms, want)
	}
}

// TestOfferBeforeCreate_NilSeamFallsBackToColumnDefault proves that with no
// composer wired (e.g. a unit test that never calls the seam), creating an
// offer with empty DeliveryTerms does not panic and ID minting still fires.
// The GORM column default is expected to fill DeliveryTerms at the DB layer;
// this test tolerates either the pre-write empty Go value or the DB default
// after reload, since the point under test is "no panic, legacy behavior".
func TestOfferBeforeCreate_NilSeamFallsBackToColumnDefault(t *testing.T) {
	db := deliveryTermsTestDB(t)

	ComposeOfferDeliveryTerms = nil

	offer := Offer{
		OfferNumber: "OFF-B3-003",
		Division:    "Beacon Controls",
		Stage:       "RFQ",
	}
	if err := db.Create(&offer).Error; err != nil {
		t.Fatalf("create offer: %v", err)
	}

	if offer.ID == "" {
		t.Fatal("expected ID to be minted even with a nil seam")
	}

	var reloaded Offer
	if err := db.First(&reloaded, "id = ?", offer.ID).Error; err != nil {
		t.Fatalf("reload offer: %v", err)
	}
	if reloaded.DeliveryTerms != "" && reloaded.DeliveryTerms != "DAP Bahrain at your store or Acme Instrumentation" {
		t.Fatalf("unexpected DeliveryTerms with nil seam: got %q", reloaded.DeliveryTerms)
	}
}
