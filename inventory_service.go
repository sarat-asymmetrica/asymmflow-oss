package main

import (
	"log"
	"time"

	"ph_holdings_app/pkg/crm/supplierlink"
)

// =============================================================================
// P2 FIX: INVENTORY ALERTS & MONITORING
// =============================================================================

// InventoryAlert represents a stock alert condition
type InventoryAlert struct {
	ProductID         string     `json:"product_id"`
	ProductCode       string     `json:"product_code"`
	ProductName       string     `json:"product_name"`
	QuantityOnHand    float64    `json:"quantity_on_hand"`
	ReorderPoint      float64    `json:"reorder_point"`
	MinimumStock      float64    `json:"minimum_stock"`
	AlertType         string     `json:"alert_type"`       // "low_stock", "out_of_stock", "slow_moving", "overstock"
	AlertSeverity     string     `json:"alert_severity"`   // "critical", "high", "medium", "low"
	DaysUntilStock    int        `json:"days_until_stock"` // Estimated days until out of stock
	LastMovementAt    *time.Time `json:"last_movement_at"`
	DaysSinceMovement int        `json:"days_since_movement"`
	SupplierID        string     `json:"supplier_id"`
	SupplierName      string     `json:"supplier_name"`
	LeadTimeDays      int        `json:"lead_time_days"`
}

// GetInventoryAlertsLowStock retrieves items below reorder point or custom threshold
func (a *App) GetInventoryAlertsLowStock(threshold float64) ([]InventoryAlert, error) {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Query inventory items below threshold
	var inventoryItems []InventoryItem
	query := a.db.Where("is_active = ?", true)

	if threshold > 0 {
		// Use custom threshold
		query = query.Where("quantity_available <= ?", threshold)
	} else {
		// Use reorder point
		query = query.Where("quantity_available <= reorder_point OR quantity_available <= minimum_stock")
	}

	if err := query.Find(&inventoryItems).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve low stock items", err.Error())
	}

	// Enrich with alert details
	alerts := make([]InventoryAlert, 0)
	for _, item := range inventoryItems {
		// Get product details
		var product ProductMaster
		if err := a.db.First(&product, "id = ?", item.ProductID).Error; err != nil {
			continue
		}

		// Get supplier details through the resolution chain (Band-2): a stale
		// or placeholder product.supplier_id no longer degrades the alert to
		// "Unknown" when the supplier is findable by code or commercial token.
		supplierName := "Unknown"
		leadTimeDays := 30 // Default
		alertSupplierID := product.SupplierID
		if supplier, err := supplierlink.ResolveSupplierForProduct(a.db, product, supplierLinkAliases()); err == nil {
			supplierName = supplier.SupplierName
			leadTimeDays = supplier.LeadTimeDays
			alertSupplierID = supplier.ID
		}

		// Determine alert type and severity
		alertType := "low_stock"
		alertSeverity := "medium"

		if item.QuantityAvailable <= 0 {
			alertType = "out_of_stock"
			alertSeverity = "critical"
		} else if item.QuantityAvailable <= item.MinimumStock {
			alertType = "low_stock"
			alertSeverity = "high"
		} else if item.QuantityAvailable <= item.ReorderPoint {
			alertType = "low_stock"
			alertSeverity = "medium"
		}

		// Calculate days until stock out (rough estimate)
		daysUntilStock := 0
		if item.LastMovementAt != nil {
			daysSinceMovement := int(time.Since(*item.LastMovementAt).Hours() / 24)
			if daysSinceMovement > 0 && item.QuantityAvailable > 0 {
				// Estimate consumption rate
				// avgDailyConsumption := item.QuantityAvailable / float64(daysSinceMovement)
				// daysUntilStock = int(item.QuantityAvailable / avgDailyConsumption)
				daysUntilStock = 7 // Placeholder - needs actual consumption rate calculation
			}
		}

		daysSinceMovement := 0
		if item.LastMovementAt != nil {
			daysSinceMovement = int(time.Since(*item.LastMovementAt).Hours() / 24)
		}

		alert := InventoryAlert{
			ProductID:         item.ProductID,
			ProductCode:       item.ProductCode,
			ProductName:       product.ProductName,
			QuantityOnHand:    item.QuantityOnHand,
			ReorderPoint:      item.ReorderPoint,
			MinimumStock:      item.MinimumStock,
			AlertType:         alertType,
			AlertSeverity:     alertSeverity,
			DaysUntilStock:    daysUntilStock,
			LastMovementAt:    item.LastMovementAt,
			DaysSinceMovement: daysSinceMovement,
			SupplierID:        alertSupplierID,
			SupplierName:      supplierName,
			LeadTimeDays:      leadTimeDays,
		}

		alerts = append(alerts, alert)
	}

	log.Printf("⚠️ Found %d low stock alerts (threshold: %.2f)", len(alerts), threshold)
	return alerts, nil
}

// GetInventoryAlertsSlowMoving retrieves items not sold/moved in N days
func (a *App) GetInventoryAlertsSlowMoving(days int) ([]InventoryAlert, error) {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -days)

	// Query inventory items with no recent movement
	var inventoryItems []InventoryItem
	query := a.db.Where("is_active = ?", true).
		Where("quantity_on_hand > 0").
		Where("last_movement_at < ? OR last_movement_at IS NULL", cutoffDate)

	if err := query.Find(&inventoryItems).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve slow moving items", err.Error())
	}

	// Enrich with alert details
	alerts := make([]InventoryAlert, 0)
	for _, item := range inventoryItems {
		// Get product details
		var product ProductMaster
		if err := a.db.First(&product, "id = ?", item.ProductID).Error; err != nil {
			continue
		}

		// Get supplier details through the resolution chain (Band-2).
		supplierName := "Unknown"
		alertSupplierID := product.SupplierID
		if supplier, err := supplierlink.ResolveSupplierForProduct(a.db, product, supplierLinkAliases()); err == nil {
			supplierName = supplier.SupplierName
			alertSupplierID = supplier.ID
		}

		daysSinceMovement := 0
		if item.LastMovementAt != nil {
			daysSinceMovement = int(time.Since(*item.LastMovementAt).Hours() / 24)
		} else {
			daysSinceMovement = 999 // Never moved
		}

		// Determine severity based on holding value and age
		alertSeverity := "low"
		if item.TotalValue > 5000.0 && daysSinceMovement > 180 {
			alertSeverity = "high" // High value, very old stock
		} else if daysSinceMovement > 365 {
			alertSeverity = "medium" // Over 1 year
		}

		alert := InventoryAlert{
			ProductID:         item.ProductID,
			ProductCode:       item.ProductCode,
			ProductName:       product.ProductName,
			QuantityOnHand:    item.QuantityOnHand,
			ReorderPoint:      item.ReorderPoint,
			MinimumStock:      item.MinimumStock,
			AlertType:         "slow_moving",
			AlertSeverity:     alertSeverity,
			LastMovementAt:    item.LastMovementAt,
			DaysSinceMovement: daysSinceMovement,
			SupplierID:        alertSupplierID,
			SupplierName:      supplierName,
		}

		alerts = append(alerts, alert)
	}

	log.Printf("📉 Found %d slow moving items (>%d days no movement)", len(alerts), days)
	return alerts, nil
}

// GetInventoryAlertsSummary returns counts by alert type
func (a *App) GetInventoryAlertsSummary() (map[string]int, error) {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	summary := make(map[string]int)

	// Get low stock count
	lowStock, err := a.GetInventoryAlertsLowStock(0)
	if err == nil {
		summary["low_stock"] = len(lowStock)

		// Count critical (out of stock)
		criticalCount := 0
		for _, alert := range lowStock {
			if alert.AlertSeverity == "critical" {
				criticalCount++
			}
		}
		summary["out_of_stock"] = criticalCount
	}

	// Get slow moving count
	slowMoving, err := a.GetInventoryAlertsSlowMoving(90)
	if err == nil {
		summary["slow_moving"] = len(slowMoving)
	}

	log.Printf("📊 Inventory alerts summary: %d total alerts", summary["low_stock"]+summary["slow_moving"])
	return summary, nil
}

// =============================================================================
// P2 FIX: AUTOMATIC REORDER SUGGESTIONS
// =============================================================================

// ReorderSuggestion represents an automated reorder recommendation
type ReorderSuggestion struct {
	ProductID      string  `json:"product_id"`
	ProductCode    string  `json:"product_code"`
	ProductName    string  `json:"product_name"`
	QuantityOnHand float64 `json:"quantity_on_hand"`
	ReorderPoint   float64 `json:"reorder_point"`
	ReorderQty     float64 `json:"reorder_qty"` // Suggested quantity to order
	SupplierID     string  `json:"supplier_id"`
	SupplierName   string  `json:"supplier_name"`
	EstimatedCost  float64 `json:"estimated_cost_bhd"`
	LeadTimeDays   int     `json:"lead_time_days"`
	UrgencyLevel   string  `json:"urgency_level"` // "urgent", "normal", "can_wait"
	Reason         string  `json:"reason"`
}

// GetReorderSuggestions generates automatic reorder recommendations
func (a *App) GetReorderSuggestions() ([]ReorderSuggestion, error) {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get low stock items
	alerts, err := a.GetInventoryAlertsLowStock(0)
	if err != nil {
		return nil, err
	}

	// Generate reorder suggestions
	suggestions := make([]ReorderSuggestion, 0)
	for _, alert := range alerts {
		// Get product to calculate reorder quantity
		var product ProductMaster
		if err := a.db.First(&product, "id = ?", alert.ProductID).Error; err != nil {
			continue
		}

		// Calculate suggested reorder quantity
		// Simple logic: bring stock to maximum level
		var inventoryItem InventoryItem
		reorderQty := 0.0
		if err := a.db.Where("product_id = ?", alert.ProductID).First(&inventoryItem).Error; err == nil {
			if inventoryItem.MaximumStock > 0 {
				reorderQty = inventoryItem.MaximumStock - alert.QuantityOnHand
			} else {
				// Default: order enough for 3 months (placeholder)
				reorderQty = alert.ReorderPoint * 3
			}
		}

		// Ensure minimum order quantity
		if reorderQty < 1 {
			reorderQty = 1
		}

		// Estimate cost
		estimatedCost := reorderQty * product.StandardCostBHD

		// Determine urgency
		urgencyLevel := "normal"
		if alert.AlertSeverity == "critical" {
			urgencyLevel = "urgent"
		} else if alert.DaysUntilStock <= alert.LeadTimeDays {
			urgencyLevel = "urgent"
		} else if alert.DaysUntilStock > alert.LeadTimeDays*2 {
			urgencyLevel = "can_wait"
		}

		suggestion := ReorderSuggestion{
			ProductID:      alert.ProductID,
			ProductCode:    alert.ProductCode,
			ProductName:    alert.ProductName,
			QuantityOnHand: alert.QuantityOnHand,
			ReorderPoint:   alert.ReorderPoint,
			ReorderQty:     reorderQty,
			SupplierID:     alert.SupplierID,
			SupplierName:   alert.SupplierName,
			EstimatedCost:  estimatedCost,
			LeadTimeDays:   alert.LeadTimeDays,
			UrgencyLevel:   urgencyLevel,
			Reason:         alert.AlertType,
		}

		suggestions = append(suggestions, suggestion)
	}

	log.Printf("💡 Generated %d reorder suggestions", len(suggestions))
	return suggestions, nil
}
