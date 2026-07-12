package main

import (
	"testing"
	"time"
)

func TestDedupeOrdersForList_PrefersCanonicalPaddedNumber(t *testing.T) {
	older := time.Date(2026, 3, 31, 9, 7, 3, 0, time.FixedZone("IST", 19800))
	newer := older.Add(24 * time.Hour)

	orders := []Order{
		{
			Base:          Base{ID: "canon", CreatedAt: older},
			OrderNumber:   "EH-01-26",
			TotalValueBHD: 2970,
			GrandTotalBHD: 3267,
			OrderDate:     older,
		},
		{
			Base:          Base{ID: "dup", CreatedAt: newer},
			OrderNumber:   "EH-1-26",
			TotalValueBHD: 2970,
			GrandTotalBHD: 3267,
			OrderDate:     newer,
		},
	}

	deduped := dedupeOrdersForList(orders)
	if len(deduped) != 1 {
		t.Fatalf("expected 1 order after dedupe, got %d", len(deduped))
	}
	if deduped[0].OrderNumber != "EH-01-26" {
		t.Fatalf("expected canonical padded order to remain, got %s", deduped[0].OrderNumber)
	}
}
