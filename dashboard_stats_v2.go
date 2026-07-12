package main

import (
	"fmt"

	commonproto "ph_holdings_app/schemas/go/common"

	capnp "capnproto.org/go/capnp/v3"
)

// DashboardStatsV2Field is a frontend-safe projection of the Proto key-value pilot.
type DashboardStatsV2Field struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// DashboardStatsV2 keeps the existing dashboard payload and exposes the Proto
// bridge used by this pilot without changing GetDashboardStats.
type DashboardStatsV2 struct {
	Stats       DashboardStats          `json:"stats"`
	ProtoSchema string                  `json:"proto_schema"`
	Fields      []DashboardStatsV2Field `json:"fields"`
}

// GetDashboardStatsV2 is an additive pilot endpoint that roundtrips existing
// dashboard stats through generated common.KeyValue Proto messages.
func (a *App) GetDashboardStatsV2() (DashboardStatsV2, error) {
	stats, err := a.GetDashboardStats()
	if err != nil {
		return DashboardStatsV2{}, err
	}
	fields, err := dashboardStatsProtoFields(stats)
	if err != nil {
		return DashboardStatsV2{}, err
	}
	return DashboardStatsV2{
		Stats:       stats,
		ProtoSchema: "common.KeyValue_List",
		Fields:      fields,
	}, nil
}

func dashboardStatsProtoFields(stats DashboardStats) ([]DashboardStatsV2Field, error) {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}
	values := []DashboardStatsV2Field{
		{"active_rfqs", fmt.Sprintf("%d", stats.ActiveRFQs)},
		{"active_orders", fmt.Sprintf("%d", stats.ActiveOrders)},
		{"pending_review", fmt.Sprintf("%d", stats.PendingReview)},
		{"urgent_count", fmt.Sprintf("%d", stats.UrgentCount)},
		{"avg_velocity_days", fmt.Sprintf("%.2f", stats.AvgVelocityDays)},
		{"win_rate", fmt.Sprintf("%.2f", stats.WinRate)},
		{"total_revenue", fmt.Sprintf("%.3f", stats.TotalRevenue)},
		{"month_growth", fmt.Sprintf("%.2f", stats.MonthGrowth)},
		{"system_health", stats.SystemHealth},
		{"runway_months", fmt.Sprintf("%.2f", stats.Runway)},
		{"outstanding_ar", fmt.Sprintf("%.3f", stats.OutstandingAR)},
		{"ar_days_overdue", fmt.Sprintf("%d", stats.ARDaysOverdue)},
		{"pending_invoices", fmt.Sprintf("%d", stats.PendingInvoices)},
		{"active_customers", fmt.Sprintf("%d", stats.ActiveCustomers)},
		{"revenue_meta", stats.RevenueMeta},
		{"activity_year", fmt.Sprintf("%d", stats.ActivityYear)},
		{"pipeline_value_bhd", fmt.Sprintf("%.3f", stats.PipelineValueBHD)},
		{"collection_rate", fmt.Sprintf("%.2f", stats.CollectionRate)},
		{"cash_balance_bhd", fmt.Sprintf("%.3f", stats.CashBalanceBHD)},
		{"cash_position_note", stats.CashPositionNote},
		{"fresh_start_date", stats.FreshStartDate},
	}

	list, err := commonproto.NewKeyValue_List(seg, int32(len(values)))
	if err != nil {
		return nil, err
	}
	for i, field := range values {
		kv, err := commonproto.NewKeyValue(seg)
		if err != nil {
			return nil, err
		}
		if err := kv.SetKey(field.Key); err != nil {
			return nil, err
		}
		if err := kv.SetValue(field.Value); err != nil {
			return nil, err
		}
		if err := list.Set(i, kv); err != nil {
			return nil, err
		}
	}

	out := make([]DashboardStatsV2Field, 0, list.Len())
	for i := 0; i < list.Len(); i++ {
		kv := list.At(i)
		key, err := kv.Key()
		if err != nil {
			return nil, err
		}
		value, err := kv.Value()
		if err != nil {
			return nil, err
		}
		out = append(out, DashboardStatsV2Field{Key: key, Value: value})
	}
	return out, nil
}
