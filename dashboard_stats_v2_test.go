package main

import "testing"

func TestDashboardStatsProtoFields(t *testing.T) {
	stats := DashboardStats{
		ActiveRFQs:       12,
		ActiveOrders:     7,
		WinRate:          42.5,
		TotalRevenue:     123.456,
		SystemHealth:     "Robust",
		ActivityYear:     2026,
		PipelineValueBHD: 77.7,
		FreshStartDate:   "2026-01-01",
	}

	fields, err := dashboardStatsProtoFields(stats)
	if err != nil {
		t.Fatalf("dashboardStatsProtoFields: %v", err)
	}
	if len(fields) != 21 {
		t.Fatalf("expected 21 fields, got %d", len(fields))
	}

	found := map[string]string{}
	for _, field := range fields {
		found[field.Key] = field.Value
	}
	if found["active_rfqs"] != "12" || found["total_revenue"] != "123.456" || found["system_health"] != "Robust" {
		t.Fatalf("unexpected proto projection: %#v", found)
	}
}
