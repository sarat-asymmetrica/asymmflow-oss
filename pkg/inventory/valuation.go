// Package inventory holds the stock-valuation engine (PH convergence Band-2
// rows 17-18, PH product_valuation_policy.go, e4aed66/5937958): the
// reference-cost fallback chain and fallback-aware weighted-average movement
// valuation. Averages are computed in raw float64 with NO intermediate
// rounding — only reporting edges round (BHD, 3 decimals). That is the PH
// golden behavior and must not change casually.
package inventory

import (
	"strings"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/crm"
)

// ReferenceCost is a resolved unit cost plus its provenance.
type ReferenceCost struct {
	UnitCostBHD float64
	Source      string
}

// ResolveProductReferenceCost resolves a product-level reference cost:
// the product's standard cost when positive, else zero.
func ResolveProductReferenceCost(db *gorm.DB, productID string) ReferenceCost {
	if db == nil || strings.TrimSpace(productID) == "" {
		return ReferenceCost{}
	}
	var product crm.ProductMaster
	if err := db.First(&product, "id = ?", productID).Error; err != nil {
		return ReferenceCost{}
	}
	if product.StandardCostBHD > 0 {
		return ReferenceCost{UnitCostBHD: product.StandardCostBHD, Source: "product_standard_cost"}
	}
	return ReferenceCost{}
}

// ResolvePurchaseOrderItemReferenceCost prefers the PO line's BHD unit price,
// falling back to the product standard cost.
func ResolvePurchaseOrderItemReferenceCost(db *gorm.DB, poItem crm.PurchaseOrderItem) ReferenceCost {
	if poItem.UnitPriceBHD > 0 {
		return ReferenceCost{UnitCostBHD: poItem.UnitPriceBHD, Source: "po_unit_price_bhd"}
	}
	return ResolveProductReferenceCost(db, poItem.ProductID)
}

// ResolveInventoryItemUnitCost is the main chain: stored unit cost → last
// purchase cost → product standard cost → zero.
func ResolveInventoryItemUnitCost(db *gorm.DB, item crm.InventoryItem) ReferenceCost {
	if item.UnitCost > 0 {
		return ReferenceCost{UnitCostBHD: item.UnitCost, Source: "inventory_unit_cost"}
	}
	if item.LastPurchaseCost > 0 {
		return ReferenceCost{UnitCostBHD: item.LastPurchaseCost, Source: "inventory_last_purchase_cost"}
	}
	return ResolveProductReferenceCost(db, item.ProductID)
}

// SupplierInvoiceItemUnitPriceBHD normalizes a supplier-invoice line price to
// BHD using the invoice's exchange rate; BHD/blank currency or a non-positive
// rate passes the price through unchanged.
func SupplierInvoiceItemUnitPriceBHD(currency string, exchangeRate, unitPrice float64) float64 {
	if unitPrice <= 0 {
		return 0
	}
	if currency == "BHD" || currency == "" || exchangeRate <= 0 {
		return unitPrice
	}
	return unitPrice * exchangeRate
}

// ApplyMovementValuation applies fallback-aware weighted-average costing to a
// stock movement whose quantity effect is ALREADY applied to the item
// (QuantityOnHand updated, movement.BalanceBefore captured). referenceCost is
// the pre-resolved chain result for the item (ResolveInventoryItemUnitCost) —
// so a zero-cost item backfills from last purchase / standard cost instead of
// silently averaging against zero, which is exactly what the pre-convergence
// inline version got wrong.
//
// IN: newAvg = (existingCost·balanceBefore + incomingCost·qty) / newOnHand,
// and LastPurchaseCost := incomingCost.
// OUT: a zero stored unit cost backfills from the chain.
// The movement is always valued (incoming cost, else the item's cost), and
// item.TotalValue = QuantityOnHand × UnitCost, unrounded.
func ApplyMovementValuation(item *crm.InventoryItem, movement *crm.StockMovement, referenceCost ReferenceCost) {
	existingUnitCost := referenceCost.UnitCostBHD

	incomingUnitCost := movement.UnitCost
	if incomingUnitCost <= 0 {
		incomingUnitCost = referenceCost.UnitCostBHD
	}

	if movement.Direction == "IN" {
		if incomingUnitCost > 0 && item.QuantityOnHand > 0 {
			totalValue := (existingUnitCost * movement.BalanceBefore) + (incomingUnitCost * movement.Quantity)
			item.UnitCost = totalValue / item.QuantityOnHand
			item.LastPurchaseCost = incomingUnitCost
		} else if incomingUnitCost > 0 {
			item.UnitCost = incomingUnitCost
			item.LastPurchaseCost = incomingUnitCost
		}
	} else if item.UnitCost <= 0 && existingUnitCost > 0 {
		item.UnitCost = existingUnitCost
	}

	if movement.UnitCost <= 0 {
		movement.UnitCost = incomingUnitCost
	}
	if movement.UnitCost > 0 {
		movement.TotalValue = movement.Quantity * movement.UnitCost
	} else {
		movement.TotalValue = movement.Quantity * item.UnitCost
	}

	item.TotalValue = item.QuantityOnHand * item.UnitCost
}

// ResolveInventoryItemTotalValue is the reporting fallback: the stored total
// when positive, else quantity × chain-resolved reference cost, else zero.
func ResolveInventoryItemTotalValue(db *gorm.DB, item crm.InventoryItem) float64 {
	if item.TotalValue > 0 {
		return item.TotalValue
	}
	referenceCost := ResolveInventoryItemUnitCost(db, item)
	if referenceCost.UnitCostBHD <= 0 {
		return 0
	}
	return item.QuantityOnHand * referenceCost.UnitCostBHD
}
