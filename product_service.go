package main

import (
	"log"
	"time"

	"ph_holdings_app/pkg/crm/supplierlink"
)

// supplierLinkAliases is the commercial vocabulary handed to the
// pkg/crm/supplierlink engine, sourced from the active overlay (Mission D):
// a sovereign deployment ships its real principal alias catalogue in
// overlay.json; the built-in default carries only the one alias in this
// repo's synthetic seed canon (product seeds use code SVX, the seeded
// supplier row uses SRVX).
func supplierLinkAliases() supplierlink.AliasConfig {
	vocab := activeOverlay.SupplierAliasVocabulary()
	return supplierlink.AliasConfig{
		CanonicalCodes: vocab.CanonicalCodes,
		BrandAliases:   vocab.BrandAliases,
	}
}

// ============================================================================
// PRODUCT DATABASE ENGINE (VQC-Ready)
// ============================================================================

// SearchProducts performs a high-performance search for products
func (a *App) SearchProducts(query string) ([]ProductMaster, error) {
	log.Printf("🔍 Searching products for: '%s'", query)
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var products []ProductMaster

	if query == "" {
		if err := a.db.Limit(20).Find(&products).Error; err != nil {
			return nil, err
		}
		return products, nil
	}

	// Sanitize user input to prevent SQL injection via LIKE wildcards
	sanitized := sanitizeSearchQuery(query)
	if len(sanitized) < 2 {
		log.Printf("⚠️ Search query too short after sanitization: '%s' -> '%s'", query, sanitized)
		return products, nil // Return empty results for very short/invalid queries
	}

	// SECURITY: Escape LIKE wildcards to prevent LIKE injection
	escaped := escapeLikeWildcards(sanitized)
	searchPattern := "%" + escaped + "%"
	err := a.db.Where("product_code LIKE ? ESCAPE '\\' OR product_name LIKE ? ESCAPE '\\' OR description LIKE ? ESCAPE '\\'",
		searchPattern, searchPattern, searchPattern).
		Limit(100). // Limit results to prevent DOS
		Find(&products).Error

	if err != nil {
		log.Printf("❌ Search failed: %v", err)
		return nil, newError("SEARCH_FAILED", "Product search failed", err.Error())
	}

	log.Printf("✅ Search for '%s' returned %d results", query, len(products))
	return products, nil
}

// GetProductByCode retrieves a single product by exact code
func (a *App) GetProductByCode(code string) (*ProductMaster, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database not ready", "")
	}

	var product ProductMaster
	if err := a.db.Where("product_code = ?", code).First(&product).Error; err != nil {
		return nil, newError("PRODUCT_NOT_FOUND", "Product not found", err.Error())
	}

	return &product, nil
}

// SeedProductDatabase populates the DB with 50+ sample SKUs if empty
// This simulates the "50,000+ SKU" database for the prototype
// SeedProductDatabase is the RBAC-guarded entry point (RBAC-003): the seed
// can overwrite the product catalogue, so only admin sessions may invoke it
// from the frontend. Startup uses the unexported variant.
func (a *App) SeedProductDatabase() error {
	if err := a.requirePermission("*"); err != nil {
		return err
	}
	return a.seedProductDatabaseInternal()
}

func (a *App) seedProductDatabaseInternal() error {
	if a.db == nil {
		return nil
	}

	var count int64
	a.db.Model(&ProductMaster{}).Count(&count)
	if count > 0 {
		return nil // Already seeded
	}

	log.Println("🌱 Seeding Product Database with high-performance instrumentation SKUs...")

	products := []ProductMaster{
		// Flow Meters
		{ProductCode: "8E3B50-1", ProductName: "Promass 83E Coriolis Flow Meter", ProductCategory: "Flow", StandardCostBHD: 1200.00, StandardPriceBHD: 1650.00, SupplierCode: "EH", Description: "Coriolis mass flow meter, DN50, 2 inch, HART output"},
		{ProductCode: "8F3B50-2", ProductName: "Promass 83F Coriolis Flow Meter", ProductCategory: "Flow", StandardCostBHD: 1450.00, StandardPriceBHD: 1950.00, SupplierCode: "EH", Description: "High accuracy mass flow meter, DN50, High temp version"},
		{ProductCode: "50W4H-1", ProductName: "Promag 50W Electromagnetic Flow Meter", ProductCategory: "Flow", StandardCostBHD: 450.00, StandardPriceBHD: 680.00, SupplierCode: "EH", Description: "Wafer style magmeter, DN100, 4 inch, EPDM liner"},
		{ProductCode: "50P25-1", ProductName: "Promag 50P Chemical Flow Meter", ProductCategory: "Flow", StandardCostBHD: 650.00, StandardPriceBHD: 920.00, SupplierCode: "EH", Description: "Chemical grade magmeter, DN25, 1 inch, PTFE liner"},

		// Level Sensors
		{ProductCode: "FMR51-1", ProductName: "Micropilot FMR51 Radar", ProductCategory: "Level", StandardCostBHD: 850.00, StandardPriceBHD: 1250.00, SupplierCode: "EH", Description: "Free space radar level sensor, horn antenna, 26GHz"},
		{ProductCode: "FMR52-1", ProductName: "Micropilot FMR52 Radar", ProductCategory: "Level", StandardCostBHD: 1100.00, StandardPriceBHD: 1550.00, SupplierCode: "EH", Description: "Free space radar with flush mounted antenna for hygiene"},
		{ProductCode: "FMP51-1", ProductName: "Levelflex FMP51 Guided Wave", ProductCategory: "Level", StandardCostBHD: 750.00, StandardPriceBHD: 1050.00, SupplierCode: "EH", Description: "Guided wave radar, rod probe, 4-20mA HART"},
		{ProductCode: "FTM50-1", ProductName: "Soliphant FTM50 Point Level", ProductCategory: "Level", StandardCostBHD: 220.00, StandardPriceBHD: 340.00, SupplierCode: "EH", Description: "Vibration limit switch for bulk solids"},

		// Pressure
		{ProductCode: "PMP51-1", ProductName: "Cerabar M PMP51", ProductCategory: "Pressure", StandardCostBHD: 350.00, StandardPriceBHD: 520.00, SupplierCode: "EH", Description: "Digital pressure transmitter, metal diaphragm"},
		{ProductCode: "PMC51-1", ProductName: "Cerabar M PMC51", ProductCategory: "Pressure", StandardCostBHD: 320.00, StandardPriceBHD: 480.00, SupplierCode: "EH", Description: "Digital pressure transmitter, ceramic diaphragm"},
		{ProductCode: "PMD75-1", ProductName: "Deltabar S PMD75", ProductCategory: "Pressure", StandardCostBHD: 850.00, StandardPriceBHD: 1250.00, SupplierCode: "EH", Description: "Differential pressure transmitter, high precision"},

		// Analysis
		{ProductCode: "CPS11D-1", ProductName: "Orbisint CPS11D pH Sensor", ProductCategory: "Analysis", StandardCostBHD: 85.00, StandardPriceBHD: 145.00, SupplierCode: "EH", Description: "Digital pH sensor, Memosens technology, glass electrode"},
		{ProductCode: "COS61D-1", ProductName: "Oxymax COS61D Oxygen Sensor", ProductCategory: "Analysis", StandardCostBHD: 420.00, StandardPriceBHD: 650.00, SupplierCode: "EH", Description: "Optical dissolved oxygen sensor, Memosens"},
		{ProductCode: "CUS51D-1", ProductName: "Turbimax CUS51D Turbidity", ProductCategory: "Analysis", StandardCostBHD: 550.00, StandardPriceBHD: 820.00, SupplierCode: "EH", Description: "Turbidity and suspended solids sensor"},

		// Oxan Analytics (Gas Analysis)
		{ProductCode: "SVX-2200", ProductName: "Oxan Analytics 2200 Oxygen Analyzer", ProductCategory: "Gas Analysis", StandardCostBHD: 2500.00, StandardPriceBHD: 3800.00, SupplierCode: "SVX", Description: "Paramagnetic O2 analyzer for industrial gas"},
		{ProductCode: "SVX-1900", ProductName: "Oxan Analytics 1900 Gas Analyzer", ProductCategory: "Gas Analysis", StandardCostBHD: 3200.00, StandardPriceBHD: 4500.00, SupplierCode: "SVX", Description: "Infrared gas analyzer for CEMS applications"},
		{ProductCode: "SVX-LASER", ProductName: "Oxan Analytics Laser 3 Plus", ProductCategory: "Gas Analysis", StandardCostBHD: 4500.00, StandardPriceBHD: 6200.00, SupplierCode: "SVX", Description: "TDL laser gas analyzer, cross-stack"},

		// GIC (Gauges)
		{ProductCode: "GIC-PG100", ProductName: "GIC Pressure Gauge 100mm", ProductCategory: "Gauge", StandardCostBHD: 25.00, StandardPriceBHD: 45.00, SupplierCode: "GIC", Description: "Stainless steel pressure gauge, 0-10 bar"},
		{ProductCode: "GIC-TG100", ProductName: "GIC Temp Gauge 100mm", ProductCategory: "Gauge", StandardCostBHD: 35.00, StandardPriceBHD: 65.00, SupplierCode: "GIC", Description: "Bimetallic temperature gauge, 0-150C"},
	}

	for _, p := range products {
		p.IsActive = true
		p.StockQuantity = 10 // Default stock
		// Band-2 rows 15-16: resolve a real supplier link instead of the old
		// "sup_"+code placeholder, which fabricated foreign keys that never
		// existed in the suppliers table. If suppliers aren't seeded yet the
		// product stays unlinked (readers resolve lazily through the same
		// engine) — an empty link is honest, a dangling one is not.
		if normalized, err := supplierlink.NormalizeProductSupplierLink(a.db, p, supplierLinkAliases()); err == nil {
			p = normalized
		} else {
			p.SupplierID = ""
			log.Printf("⚠ Product %s: supplier link unresolved at seed: %v", p.ProductCode, err)
		}
		p.CreatedAt = time.Now()
		p.UpdatedAt = time.Now()

		if err := a.db.Create(&p).Error; err != nil {
			log.Printf("⚠ Failed to seed product %s: %v", p.ProductCode, err)
		}
	}

	return nil
}
