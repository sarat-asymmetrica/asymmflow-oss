package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// PH convergence 1-POVAT (PH MON-004): VATAmount is a BHD figure — it feeds
// TotalBHD and any input-VAT read of the field — so it is computed on the BHD
// subtotal. The old SubtotalForeign*0.10 stored VAT in foreign units on a
// non-BHD PO. Identical for BHD POs where the rate is 1.
func TestNormalizePurchaseOrder_VATOnBHDSubtotal(t *testing.T) {
	a := setupTestApp(t)
	supplier := SupplierMaster{SupplierName: "Meridian Instruments GmbH"}
	require.NoError(t, a.db.Create(&supplier).Error)

	foreign := PurchaseOrder{
		SupplierID:   supplier.ID,
		Currency:     "EUR",
		ExchangeRate: 0.42,
		Items: []PurchaseOrderItem{
			{Description: "Flow transmitter", Quantity: 10, UnitPriceForeign: 100},
		},
	}
	require.NoError(t, a.normalizePurchaseOrder(&foreign))
	require.Equal(t, 1000.0, foreign.SubtotalForeign)
	require.Equal(t, 420.0, foreign.SubtotalBHD)
	require.Equal(t, 42.0, foreign.VATAmount, "VAT must be 10%% of the BHD subtotal, not the foreign one")
	require.Equal(t, 1100.0, foreign.TotalForeign)
	require.Equal(t, 462.0, foreign.TotalBHD)

	bhd := PurchaseOrder{
		SupplierID:   supplier.ID,
		Currency:     "BHD",
		ExchangeRate: 1,
		Items: []PurchaseOrderItem{
			{Description: "Pressure gauge", Quantity: 4, UnitPriceForeign: 250},
		},
	}
	require.NoError(t, a.normalizePurchaseOrder(&bhd))
	require.Equal(t, 1000.0, bhd.SubtotalBHD)
	require.Equal(t, 100.0, bhd.VATAmount)
	require.Equal(t, 1100.0, bhd.TotalBHD, "rate=1 totals unchanged by the basis fix")
}
