package main

// Customer win-rate aggregation (owner ruling G1.4). The legacy PricingScreen
// HARDCODED its per-customer win-rate sidebar list (`overallStats.customers`);
// that literal WAS the bug. This binding computes win-rate from the real offer
// won/lost history instead. Read-only — it posts nothing and mutates nothing;
// the pricing screen renders the result and feeds customer names to the
// separate SimulateMargin model.

import (
	"fmt"
	"sort"
	"strings"
)

// CustomerWinRate is one row of the aggregation: a customer's decided-offer
// outcomes (Won vs Lost) and the resulting win-rate + won-revenue. Offers still
// in flight (RFQ/Quoted/Expired) are excluded — win-rate is a ratio over
// DECIDED offers only.
type CustomerWinRate struct {
	CustomerID   string  `json:"customer_id"`
	CustomerName string  `json:"customer_name"`
	OffersWon    int     `json:"offers_won"`
	OffersLost   int     `json:"offers_lost"`
	OffersTotal  int     `json:"offers_total"` // won + lost (decided)
	WinRate      float64 `json:"win_rate"`     // won / (won + lost), 0..1
	WonValueBHD  float64 `json:"won_value_bhd"`
}

// GetCustomerWinRates aggregates decided offers (Stage Won/Lost) per customer.
// Grouping key is the customer id when present, else the (lower-cased) customer
// name, so offers that carry a name but no id still aggregate honestly rather
// than collapsing into one empty-id bucket. Ordered by won revenue descending.
func (a *App) GetCustomerWinRates() ([]CustomerWinRate, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var offers []Offer
	if err := a.db.Where("stage IN ?", []string{"Won", "Lost"}).Find(&offers).Error; err != nil {
		return nil, fmt.Errorf("failed to load offer history: %w", err)
	}

	type agg struct {
		id, name    string
		won, lost   int
		wonValueBHD float64
	}
	byKey := map[string]*agg{}
	order := make([]string, 0)

	for _, o := range offers {
		key := strings.TrimSpace(o.CustomerID)
		if key == "" {
			name := strings.ToLower(strings.TrimSpace(o.CustomerName))
			if name == "" {
				continue // no identity at all — cannot attribute this offer
			}
			key = "name:" + name
		}
		g := byKey[key]
		if g == nil {
			g = &agg{id: o.CustomerID, name: o.CustomerName}
			byKey[key] = g
			order = append(order, key)
		}
		if g.name == "" {
			g.name = o.CustomerName
		}
		if o.Stage == "Won" {
			g.won++
			g.wonValueBHD += o.TotalValueBHD
		} else {
			g.lost++
		}
	}

	result := make([]CustomerWinRate, 0, len(order))
	for _, key := range order {
		g := byKey[key]
		total := g.won + g.lost
		if total == 0 {
			continue
		}
		result = append(result, CustomerWinRate{
			CustomerID:   g.id,
			CustomerName: g.name,
			OffersWon:    g.won,
			OffersLost:   g.lost,
			OffersTotal:  total,
			WinRate:      float64(g.won) / float64(total),
			WonValueBHD:  roundBHD(g.wonValueBHD),
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].WonValueBHD > result[j].WonValueBHD
	})
	return result, nil
}
