package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// PH convergence Band-2 rows 15-16 (PH product_supplier_linkage_test.go,
// 55654d3/7034098): products must carry REAL supplier links, and inference
// must survive stale links via the code/canonical/commercial fallback chain.
func TestSeedProductDatabase_UsesCanonicalSupplierLinks(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}))

	srvx := SupplierMaster{SupplierCode: "SRVX", SupplierName: "Oxan Analytics"}
	eh := SupplierMaster{SupplierCode: "EH", SupplierName: "Rhine Instruments"}
	require.NoError(t, a.db.Create(&srvx).Error)
	require.NoError(t, a.db.Create(&eh).Error)

	require.NoError(t, a.seedProductDatabaseInternal())

	var svxProducts []ProductMaster
	require.NoError(t, a.db.Where("supplier_code = ?", "SRVX").Find(&svxProducts).Error)
	require.NotEmpty(t, svxProducts, "SVX seed products should canonicalize to SRVX")
	for _, p := range svxProducts {
		require.Equal(t, srvx.ID, p.SupplierID, "product %s must link the real SRVX supplier", p.ProductCode)
	}

	var ehProduct ProductMaster
	require.NoError(t, a.db.Where("product_code = ?", "8E3B50-1").First(&ehProduct).Error)
	require.Equal(t, eh.ID, ehProduct.SupplierID)

	// No supplier for GIC in this DB: the products stay honestly unlinked —
	// never the old fabricated "sup_gic" placeholder.
	var gicProducts []ProductMaster
	require.NoError(t, a.db.Where("product_code LIKE ?", "GIC-%").Find(&gicProducts).Error)
	require.NotEmpty(t, gicProducts)
	for _, p := range gicProducts {
		require.Empty(t, p.SupplierID, "unresolvable seed link must stay empty, got %q", p.SupplierID)
		require.False(t, strings.HasPrefix(p.SupplierID, "sup_"))
	}
}

func TestInferSupplierForOrderItems_FallsBackThroughChain(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}))

	srvx := SupplierMaster{SupplierCode: "SRVX", SupplierName: "Oxan Analytics"}
	require.NoError(t, a.db.Create(&srvx).Error)

	// Product with a stale placeholder supplier_id but a resolvable alias code.
	product := ProductMaster{ProductCode: "SVX-2200", ProductName: "Oxan Analytics 2200", SupplierID: "sup_svx", SupplierCode: "SVX"}
	require.NoError(t, a.db.Create(&product).Error)

	order := Order{OrderNumber: "ORD-26-5001", CustomerName: "Nimbus Controls"}
	order.Items = []OrderItem{{OrderID: order.ID, ProductID: product.ID, Description: "O2 analyzer", Quantity: 1, UnitPrice: 2500, TotalPrice: 2500}}

	supplier, err := a.inferSupplierForOrderItems(order, nil)
	require.NoError(t, err)
	require.Equal(t, srvx.ID, supplier.ID, "stale placeholder link must recover via canonical code")

	// Item with no product row at all: free-text token from the product code.
	orphanOrder := Order{OrderNumber: "ORD-26-5002", CustomerName: "Atlas Traders"}
	orphanOrder.Items = []OrderItem{{Description: "Spare analyzer cell", ProductCode: "SRVX-CELL-9", Quantity: 1, UnitPrice: 120, TotalPrice: 120}}

	supplier, err = a.inferSupplierForOrderItems(orphanOrder, nil)
	require.NoError(t, err)
	require.Equal(t, srvx.ID, supplier.ID, "commercial token from the item code must resolve")
}
