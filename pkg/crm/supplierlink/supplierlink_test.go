package supplierlink

import (
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/ncruces/go-sqlite3/embed"
	"ph_holdings_app/pkg/crm"
)

func linkTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "supplierlink.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&crm.SupplierMaster{}, &crm.ProductMaster{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func testAliases() AliasConfig {
	return AliasConfig{
		CanonicalCodes: map[string]string{"SVX": "SRVX"},
		BrandAliases:   map[string][]string{"OXAN": {"Oxan Analytics"}},
	}
}

func seedSupplier(t *testing.T, db *gorm.DB, code, name, brands string) crm.SupplierMaster {
	t.Helper()
	s := crm.SupplierMaster{SupplierCode: code, SupplierName: name, BrandsHandled: brands}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("seed supplier %s: %v", code, err)
	}
	return s
}

func TestResolveSupplierForProduct_ExactIDWins(t *testing.T) {
	db := linkTestDB(t)
	s := seedSupplier(t, db, "EH", "Rhine Instruments", "")
	seedSupplier(t, db, "SRVX", "Oxan Analytics", "")

	got, err := ResolveSupplierForProduct(db, crm.ProductMaster{SupplierID: s.ID, SupplierCode: "SRVX"}, testAliases())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != s.ID {
		t.Fatalf("exact supplier ID must win over code, got %s", got.SupplierCode)
	}
}

func TestResolveSupplierForProduct_FallsBackToSupplierCode(t *testing.T) {
	db := linkTestDB(t)
	s := seedSupplier(t, db, "EH", "Rhine Instruments", "")

	// Stale/placeholder supplier ID recovers via code (PH's exact regression).
	got, err := ResolveSupplierForProduct(db, crm.ProductMaster{SupplierID: "sup_eh", SupplierCode: "EH"}, testAliases())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != s.ID {
		t.Fatalf("stale ID should fall back to code, got %+v", got)
	}
}

func TestResolveSupplierForProduct_CanonicalCodeAlias(t *testing.T) {
	db := linkTestDB(t)
	s := seedSupplier(t, db, "SRVX", "Oxan Analytics", "")

	got, err := ResolveSupplierForProduct(db, crm.ProductMaster{SupplierCode: "SVX", ProductCode: "SVX-2200"}, testAliases())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != s.ID {
		t.Fatalf("SVX should canonicalize to SRVX, got %+v", got)
	}
}

func TestResolveSupplierForProduct_CommercialTokenFromName(t *testing.T) {
	db := linkTestDB(t)
	s := seedSupplier(t, db, "SUP-25", "Oxan Analytics", "")

	// No usable ID or code — the product name carries the brand token.
	p := crm.ProductMaster{ProductName: "Oxan Analytics 1900 Gas Analyzer", ProductCode: "GA-1900"}
	got, err := ResolveSupplierForProduct(db, p, testAliases())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != s.ID {
		t.Fatalf("commercial-token search should find the supplier, got %+v", got)
	}
}

func TestFindSupplierByCommercialToken_BrandAliasAndLike(t *testing.T) {
	db := linkTestDB(t)
	s := seedSupplier(t, db, "SUP-25", "Oxan Analytics", `["Oxan","Gasline"]`)

	// Alias expansion: OXAN -> "Oxan Analytics" (exact name pass).
	got, err := FindSupplierByCommercialToken(db, "OXAN", testAliases())
	if err != nil || got.ID != s.ID {
		t.Fatalf("brand alias should resolve, got %+v err=%v", got, err)
	}

	// LIKE pass over brands_handled.
	got, err = FindSupplierByCommercialToken(db, "Gasline", testAliases())
	if err != nil || got.ID != s.ID {
		t.Fatalf("brands_handled LIKE should resolve, got %+v err=%v", got, err)
	}

	// LIKE wildcards in the token must be escaped, not interpreted.
	if _, err := FindSupplierByCommercialToken(db, "%", testAliases()); err == nil {
		t.Fatal("a bare wildcard token must not match everything")
	}
}

func TestNormalizeProductSupplierLink_StampsCanonicalLink(t *testing.T) {
	db := linkTestDB(t)
	s := seedSupplier(t, db, "SRVX", "Oxan Analytics", "")

	p := crm.ProductMaster{ProductCode: "SVX-2200", SupplierCode: "SVX"}
	normalized, err := NormalizeProductSupplierLink(db, p, testAliases())
	if err != nil {
		t.Fatal(err)
	}
	if normalized.SupplierID != s.ID || normalized.SupplierCode != "SRVX" {
		t.Fatalf("normalize should stamp canonical ID+code, got %+v", normalized)
	}

	// Unresolvable: product returned unchanged with an error.
	orphan := crm.ProductMaster{ProductCode: "ZZ-1", SupplierCode: "ZZ"}
	back, err := NormalizeProductSupplierLink(db, orphan, testAliases())
	if err == nil {
		t.Fatal("unresolvable link must error")
	}
	if back.SupplierID != "" {
		t.Fatalf("failed resolution must not fabricate a link, got %q", back.SupplierID)
	}
}
