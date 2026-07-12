package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MISSION I — I-22: CUSTOMER RELATED PRODUCTS / SUPPLIERS
// =============================================================================
// GetCustomerRelatedProducts / GetCustomerRelatedSuppliers aggregate a customer's
// historical products (and the suppliers behind them) from order history. The
// rollup groups order-item lines by product, sums quantity, counts distinct
// orders, tracks last-ordered, and resolves the supplier off the OSS
// ProductMaster.SupplierID/SupplierCode link.
// =============================================================================

func seedRelatedEntitiesFixture(t *testing.T, app *App) {
	t.Helper()
	require.NoError(t, app.db.AutoMigrate(&ProductMaster{}))

	now := time.Now()
	jan := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	jun := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)

	suppliers := []SupplierMaster{
		{Base: Base{ID: "SUP-A"}, SupplierCode: "RHI", SupplierName: "Rhine Instruments", SupplierType: "Manufacturer"},
		{Base: Base{ID: "SUP-B"}, SupplierCode: "DEL", SupplierName: "Delta Controls", SupplierType: "Distributor"},
	}
	for i := range suppliers {
		require.NoError(t, app.db.Create(&suppliers[i]).Error)
	}

	products := []ProductMaster{
		{Base: Base{ID: "PROD-1"}, ProductCode: "P1", ProductName: "Flow Meter", ProductCategory: "Flow", SupplierID: "SUP-A"},
		{Base: Base{ID: "PROD-2"}, ProductCode: "P2", ProductName: "Level Transmitter", ProductCategory: "Level", SupplierID: "SUP-A"},
		{Base: Base{ID: "PROD-3"}, ProductCode: "P3", ProductName: "Control Valve", ProductCategory: "Valve", SupplierID: "SUP-B"},
	}
	for i := range products {
		require.NoError(t, app.db.Create(&products[i]).Error)
	}

	customer := CustomerMaster{
		Base:         Base{ID: "cust-rel-uuid", CreatedAt: now, UpdatedAt: now},
		CustomerID:   "CUST-REL-1",
		CustomerCode: "CUST-REL-1",
		BusinessName: "National Petroleum Co.",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	orders := []Order{
		{Base: Base{ID: "ORD-1"}, OrderNumber: "SO-1", CustomerID: "CUST-REL-1", OrderDate: jan},
		{Base: Base{ID: "ORD-2"}, OrderNumber: "SO-2", CustomerID: "CUST-REL-1", OrderDate: jun},
	}
	for i := range orders {
		require.NoError(t, app.db.Create(&orders[i]).Error)
	}

	items := []OrderItem{
		// P1 ordered in both orders → OrderCount 2, TotalQuantity 5, last = Jun.
		{Base: Base{ID: "OI-1"}, OrderID: "ORD-1", ProductID: "PROD-1", ProductCode: "P1", Quantity: 2},
		{Base: Base{ID: "OI-2"}, OrderID: "ORD-2", ProductID: "PROD-1", ProductCode: "P1", Quantity: 3},
		// P2 in Jun order only.
		{Base: Base{ID: "OI-3"}, OrderID: "ORD-2", ProductID: "PROD-2", ProductCode: "P2", Quantity: 4},
		// P3 in Jan order only (Supplier B).
		{Base: Base{ID: "OI-4"}, OrderID: "ORD-1", ProductID: "PROD-3", ProductCode: "P3", Quantity: 1},
		// Legacy free-text line: product code with no ProductMaster row → falls back
		// to the code as the name with no resolvable supplier.
		{Base: Base{ID: "OI-5"}, OrderID: "ORD-1", ProductID: "", ProductCode: "LEGACY-X", Quantity: 7},
	}
	for i := range items {
		require.NoError(t, app.db.Create(&items[i]).Error)
	}
}

func TestGetCustomerRelatedProducts_AggregatesOrderHistory(t *testing.T) {
	app := setupTestApp(t)
	seedRelatedEntitiesFixture(t, app)

	products, err := app.GetCustomerRelatedProducts("CUST-REL-1")
	require.NoError(t, err)

	byCode := make(map[string]CustomerRelatedProduct, len(products))
	for _, p := range products {
		byCode[p.ProductCode] = p
	}

	// Four distinct products (P1, P2, P3, plus the legacy free-text line).
	require.Len(t, products, 4)

	p1 := byCode["P1"]
	assert.Equal(t, "Flow Meter", p1.ProductName)
	assert.Equal(t, "Flow", p1.ProductCategory)
	assert.Equal(t, "Rhine Instruments", p1.SupplierName)
	assert.Equal(t, 5.0, p1.TotalQuantity, "P1 quantity must sum across both orders")
	assert.Equal(t, 2, p1.OrderCount, "P1 was ordered in two distinct orders")
	require.NotNil(t, p1.LastOrdered)
	assert.Equal(t, 2025, p1.LastOrdered.Year())
	assert.Equal(t, time.June, p1.LastOrdered.Month(), "last-ordered must be the most recent order")

	p3 := byCode["P3"]
	assert.Equal(t, "Delta Controls", p3.SupplierName)
	assert.Equal(t, 1, p3.OrderCount)

	// Legacy line: no catalog product → code becomes the name, no supplier.
	legacy := byCode["LEGACY-X"]
	assert.Equal(t, "LEGACY-X", legacy.ProductName)
	assert.Equal(t, "", legacy.SupplierName)
	assert.Equal(t, 7.0, legacy.TotalQuantity)

	// Sort contract: last_ordered desc, then order_count desc. P1 and P2 share the
	// June last-order date, so P1 (2 orders) must precede P2 (1 order).
	require.Equal(t, "P1", products[0].ProductCode)
	assert.Equal(t, "P2", products[1].ProductCode)
}

func TestGetCustomerRelatedSuppliers_DerivedThroughProducts(t *testing.T) {
	app := setupTestApp(t)
	seedRelatedEntitiesFixture(t, app)

	suppliers, err := app.GetCustomerRelatedSuppliers("CUST-REL-1")
	require.NoError(t, err)

	// Two resolvable suppliers (Rhine via P1+P2, Delta via P3). The legacy line has
	// no supplier and must not create a phantom supplier row.
	require.Len(t, suppliers, 2)

	byID := make(map[string]CustomerRelatedSupplier, len(suppliers))
	for _, s := range suppliers {
		byID[s.SupplierID] = s
	}

	rhine := byID["SUP-A"]
	assert.Equal(t, "Rhine Instruments", rhine.SupplierName)
	assert.Equal(t, "RHI", rhine.SupplierCode)
	assert.Equal(t, 2, rhine.ProductCount, "Rhine supplies two distinct ordered products")
	require.NotNil(t, rhine.LastOrdered)
	assert.Equal(t, time.June, rhine.LastOrdered.Month())

	delta := byID["SUP-B"]
	assert.Equal(t, 1, delta.ProductCount)

	// Sort contract: last_ordered desc. Rhine (June) precedes Delta (January).
	require.Equal(t, "SUP-A", suppliers[0].SupplierID)
}

func TestGetCustomerRelatedProducts_EmptyForUnknownCustomer(t *testing.T) {
	app := setupTestApp(t)
	seedRelatedEntitiesFixture(t, app)

	// A customer that exists but has no orders yields an empty (non-error) result.
	noOrders := CustomerMaster{Base: Base{ID: "cust-empty"}, CustomerID: "CUST-EMPTY", CustomerCode: "CUST-EMPTY", BusinessName: "Quiet Trading Co."}
	require.NoError(t, app.db.Create(&noOrders).Error)

	products, err := app.GetCustomerRelatedProducts("CUST-EMPTY")
	require.NoError(t, err)
	assert.Empty(t, products)

	suppliers, err := app.GetCustomerRelatedSuppliers("CUST-EMPTY")
	require.NoError(t, err)
	assert.Empty(t, suppliers)
}
