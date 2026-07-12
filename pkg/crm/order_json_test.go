package crm

import (
	"encoding/json"
	"testing"
	"time"
)

// PH convergence A1: <input type="date"> posts date-only strings; the binding
// layer must accept them, and empty/absent dates must stay zero so GORM's
// Updates(struct) skips the column.
func TestOrderUnmarshalJSON_FlexibleDates(t *testing.T) {
	var o Order
	payload := `{"order_number":"ORD-26-100","order_date":"2026-05-20","required_date":"2026-06-15T10:30:00Z"}`
	if err := json.Unmarshal([]byte(payload), &o); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if o.OrderNumber != "ORD-26-100" {
		t.Fatalf("other fields must decode normally, got %q", o.OrderNumber)
	}
	if got := o.OrderDate.Format("2006-01-02"); got != "2026-05-20" {
		t.Fatalf("date-only order_date, got %s", got)
	}
	if !o.RequiredDate.Equal(time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC)) {
		t.Fatalf("RFC3339 required_date, got %v", o.RequiredDate)
	}

	var empty Order
	if err := json.Unmarshal([]byte(`{"order_number":"ORD-26-101","order_date":"","required_date":null}`), &empty); err != nil {
		t.Fatalf("empty dates must not error: %v", err)
	}
	if !empty.OrderDate.IsZero() || !empty.RequiredDate.IsZero() {
		t.Fatal("empty/null dates must stay zero so GORM skips them")
	}

	if err := json.Unmarshal([]byte(`{"order_date":"20-05-2026x"}`), &Order{}); err == nil {
		t.Fatal("garbage date must be rejected")
	}
}
