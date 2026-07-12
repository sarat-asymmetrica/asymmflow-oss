package finance

import (
	"encoding/json"
	"testing"
)

// PH convergence A1: date-only strings accepted; absent due_date stays nil.
func TestExpenseEntryUnmarshalJSON_FlexibleDates(t *testing.T) {
	var e ExpenseEntry
	payload := `{"entry_number":"EXP-26-01","expense_date":"2026-05-20","due_date":"2026-06-01"}`
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.EntryNumber != "EXP-26-01" {
		t.Fatalf("other fields must decode normally, got %q", e.EntryNumber)
	}
	if got := e.ExpenseDate.Format("2006-01-02"); got != "2026-05-20" {
		t.Fatalf("date-only expense_date, got %s", got)
	}
	if e.DueDate == nil || e.DueDate.Format("2006-01-02") != "2026-06-01" {
		t.Fatalf("date-only due_date, got %v", e.DueDate)
	}

	var empty ExpenseEntry
	if err := json.Unmarshal([]byte(`{"entry_number":"EXP-26-02","expense_date":"","due_date":null}`), &empty); err != nil {
		t.Fatalf("empty dates must not error: %v", err)
	}
	if !empty.ExpenseDate.IsZero() || empty.DueDate != nil {
		t.Fatal("empty expense_date stays zero; null due_date stays nil")
	}
}
