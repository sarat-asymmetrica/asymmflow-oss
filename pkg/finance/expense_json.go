package finance

import (
	"encoding/json"
	"fmt"

	"ph_holdings_app/pkg/kernel/jsondate"
)

// UnmarshalJSON parses an ExpenseEntry from frontend JSON, accepting date-only
// strings for expense_date/due_date while leaving every other field to the
// default decoder via the embedded alias. An empty/absent due_date leaves the
// nullable pointer nil.
func (e *ExpenseEntry) UnmarshalJSON(data []byte) error {
	type Alias ExpenseEntry
	aux := &struct {
		ExpenseDate json.RawMessage `json:"expense_date"`
		DueDate     json.RawMessage `json:"due_date"`
		*Alias
	}{Alias: (*Alias)(e)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if t, ok, err := jsondate.ParseFlexible(aux.ExpenseDate); err != nil {
		return fmt.Errorf("expense_date: %w", err)
	} else if ok {
		e.ExpenseDate = t
	}
	if t, ok, err := jsondate.ParseFlexible(aux.DueDate); err != nil {
		return fmt.Errorf("due_date: %w", err)
	} else if ok {
		e.DueDate = &t
	}
	return nil
}
