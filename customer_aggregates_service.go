package main

// Wave 8 Bucket G tail — computed-aggregate recompute, ported from deployed
// PH's sprint4_aggregates.go (Sprint 4 W5). Populates the CustomerMaster
// aggregate columns from live order/invoice/payment data. Idempotent
// overwrite of COMPUTED columns ONLY — it never touches user-editable fields.
//
// Sovereign divergence (same class as the P3 slice-4 division fix): PH's
// canonical active-customer count hardcodes its own internal entity markers
// (short_code 'PH', customer_type 'PH Trading'). Here both come from the
// active overlay — the license/company mnemonic and the default division —
// so a different vertical's internal entity is a config edit, not a code edit.

import (
	"fmt"
	"math"
	"strings"
	"time"

	"ph_holdings_app/pkg/overlay"

	"gorm.io/gorm"
)

type Sprint4AggregateResult struct {
	CustomersProcessed int     `json:"customers_processed"`
	CustomersUpdated   int     `json:"customers_updated"`
	WithOrders         int     `json:"with_orders"`
	TotalOrdersValue   float64 `json:"total_orders_value_sum"`
	TotalOutstanding   float64 `json:"total_outstanding_sum"`
	ActiveCanonical    int     `json:"active_customer_count_canonical"`
	ActiveRaw          int     `json:"active_customer_count_raw"`
}

// sprint4ARRiskTier derives the AR risk tier from outstanding balance and the
// worst overdue age. Documented, deterministic rule.
func sprint4ARRiskTier(outstanding float64, overdueDays int) string {
	if outstanding <= 0 {
		return "Low"
	}
	switch {
	case overdueDays >= 120:
		return "Critical"
	case overdueDays >= 90:
		return "High"
	case overdueDays >= 30:
		return "Medium"
	default:
		return "Low"
	}
}

// sprint4RecomputeAggregates recomputes every CustomerMaster aggregate column from
// the live order/invoice/payment data. Deterministic and idempotent.
func sprint4RecomputeAggregates(db *gorm.DB) (Sprint4AggregateResult, error) {
	var res Sprint4AggregateResult
	now := time.Now()

	// --- Order rollups per customer (count, value, last order date) ---
	type orderAgg struct {
		CustomerID    string
		Cnt           int
		Val           float64
		LastOrderDate string
	}
	var orderAggs []orderAgg
	if err := db.Raw(`SELECT customer_id AS customer_id,
		COUNT(*) AS cnt,
		COALESCE(SUM(CASE WHEN grand_total_bhd > 0 THEN grand_total_bhd ELSE total_value_bhd END), 0) AS val,
		MAX(order_date) AS last_order_date
		FROM orders
		WHERE deleted_at IS NULL AND customer_id IS NOT NULL AND TRIM(customer_id) <> ''
		GROUP BY customer_id`).Scan(&orderAggs).Error; err != nil {
		return res, err
	}
	orderByCustomer := make(map[string]orderAgg, len(orderAggs))
	for _, o := range orderAggs {
		orderByCustomer[o.CustomerID] = o
	}

	// --- Avg days-to-pay per customer (payments joined to invoices) ---
	type payAgg struct {
		CustomerID string
		AvgDays    float64
	}
	var payAggs []payAgg
	if err := db.Raw(`SELECT i.customer_id AS customer_id,
		AVG(CASE WHEN p.days_to_payment > 0 THEN p.days_to_payment
		         ELSE (julianday(p.payment_date) - julianday(i.invoice_date)) END) AS avg_days
		FROM payments p JOIN invoices i ON p.invoice_id = i.id
		WHERE p.deleted_at IS NULL AND i.deleted_at IS NULL
		  AND i.customer_id IS NOT NULL AND TRIM(i.customer_id) <> ''
		GROUP BY i.customer_id`).Scan(&payAggs).Error; err != nil {
		return res, err
	}
	avgPayByCustomer := make(map[string]float64, len(payAggs))
	for _, p := range payAggs {
		avgPayByCustomer[p.CustomerID] = p.AvgDays
	}

	// --- Outstanding + overdue per customer (collectibility-normalized in Go) ---
	var invoices []Invoice
	if err := db.Where("deleted_at IS NULL").Find(&invoices).Error; err != nil {
		return res, err
	}
	outstandingByCustomer := make(map[string]float64)
	overdueDaysByCustomer := make(map[string]int)
	disputeByCustomer := make(map[string]int)
	for _, inv := range invoices {
		state := customerInvoicePaymentStateFromInvoice(inv, now)
		if state.IsCollectible {
			outstandingByCustomer[inv.CustomerID] += state.OutstandingBHD
			if state.IsOverdue && !inv.DueDate.IsZero() {
				d := int(now.Sub(inv.DueDate).Hours() / 24)
				if d > overdueDaysByCustomer[inv.CustomerID] {
					overdueDaysByCustomer[inv.CustomerID] = d
				}
			}
		}
		if inv.Status == "Dispute" || inv.Status == "Disputed" {
			disputeByCustomer[inv.CustomerID]++
		}
	}

	// --- Apply to every customer (computed columns only) ---
	var customers []CustomerMaster
	if err := db.Where("deleted_at IS NULL").Find(&customers).Error; err != nil {
		return res, err
	}
	res.CustomersProcessed = len(customers)

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, c := range customers {
			ids := uniqueNonEmptyStrings(c.ID, c.CustomerID, c.CustomerCode)
			var oa orderAgg
			var haveOrders bool
			for _, id := range ids {
				if v, ok := orderByCustomer[id]; ok {
					oa.Cnt += v.Cnt
					oa.Val += v.Val
					if v.LastOrderDate > oa.LastOrderDate {
						oa.LastOrderDate = v.LastOrderDate
					}
					haveOrders = true
				}
			}
			var outstanding float64
			var overdue, dispute int
			var avgPay float64
			for _, id := range ids {
				outstanding += outstandingByCustomer[id]
				if overdueDaysByCustomer[id] > overdue {
					overdue = overdueDaysByCustomer[id]
				}
				dispute += disputeByCustomer[id]
				if v, ok := avgPayByCustomer[id]; ok && v > avgPay {
					avgPay = v
				}
			}

			avgOrder := 0.0
			if oa.Cnt > 0 {
				avgOrder = oa.Val / float64(oa.Cnt)
			}

			updates := map[string]any{
				"total_orders_count": oa.Cnt,
				"total_orders_value": roundTo3(oa.Val),
				"avg_order_value":    roundTo3(avgOrder),
				"outstanding_bhd":    roundTo3(outstanding),
				"overdue_days":       overdue,
				"dispute_count":      dispute,
				"avg_payment_days":   math.Round(avgPay*100) / 100,
				"ar_risk_tier":       sprint4ARRiskTier(outstanding, overdue),
			}
			if oa.LastOrderDate != "" {
				updates["last_order_date"] = oa.LastOrderDate
			}

			if err := tx.Model(&CustomerMaster{}).Where("id = ?", c.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("recompute aggregates for %s: %w", c.ID, err)
			}
			res.CustomersUpdated++
			if haveOrders {
				res.WithOrders++
			}
			res.TotalOrdersValue += oa.Val
			res.TotalOutstanding += outstanding
		}
		return nil
	})
	if err != nil {
		return res, err
	}

	res.TotalOrdersValue = roundTo3(res.TotalOrdersValue)
	res.TotalOutstanding = roundTo3(res.TotalOutstanding)
	res.ActiveRaw = sprint4CountActiveRaw(db)
	res.ActiveCanonical = sprint4CountActiveCanonical(db)
	return res, nil
}

// sprint4CountActiveRaw = non-deleted customers (the headline count).
func sprint4CountActiveRaw(db *gorm.DB) int {
	var n int
	db.Raw("SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL").Scan(&n)
	return n
}

// sprint4InternalEntityLiterals returns the configured internal-entity markers
// as escaped SQL string literals: the company mnemonic (used as the internal
// row's short_code, e.g. 'PH') and the default division (used as its
// customer_type, e.g. 'PH Trading' / 'Acme Instrumentation'). PH hardcodes
// both; here they are overlay configuration.
func sprint4InternalEntityLiterals() (shortCode, customerType string) {
	esc := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	return esc(overlay.Active().LicenseKeyPrefixOrDefault()), divisionDefaultSQLLiteral()
}

// sprint4CountActiveCanonical applies the agreed active-customer definition (W5.3):
// non-deleted ∧ Active status ∧ not an internal/company/test/demo entity.
func sprint4CountActiveCanonical(db *gorm.DB) int {
	shortCodeLit, customerTypeLit := sprint4InternalEntityLiterals()
	var n int
	db.Raw(`SELECT COUNT(*) FROM customers
		WHERE deleted_at IS NULL
		  AND (status = 'Active' OR status = '' OR status IS NULL)
		  AND COALESCE(short_code,'') <> ` + shortCodeLit + `
		  AND COALESCE(customer_type,'') <> ` + customerTypeLit + `
		  AND LOWER(COALESCE(business_name,'')) NOT LIKE '%demo%'
		  AND LOWER(COALESCE(business_name,'')) NOT LIKE '%test%'`).Scan(&n)
	return n
}

// RecomputeAllCustomerAggregates is the standalone, RBAC-gated handler that runs
// the W5 recompute against the live DB (so it can be scheduled later). Computed
// columns only — never touches user-editable customer fields.
func (a *App) RecomputeAllCustomerAggregates() (map[string]any, error) {
	if err := a.requirePermission("customers:edit"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	res, err := sprint4RecomputeAggregates(a.db)
	if err != nil {
		return nil, newError("RECOMPUTE_FAILED", "Failed to recompute customer aggregates", err.Error())
	}
	return map[string]any{
		"customers_processed": res.CustomersProcessed,
		"customers_updated":   res.CustomersUpdated,
		"with_orders":         res.WithOrders,
		"total_orders_value":  res.TotalOrdersValue,
		"total_outstanding":   res.TotalOutstanding,
		"active_canonical":    res.ActiveCanonical,
		"active_raw":          res.ActiveRaw,
	}, nil
}
