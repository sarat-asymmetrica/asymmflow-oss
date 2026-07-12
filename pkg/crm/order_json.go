package crm

import (
	"encoding/json"
	"fmt"

	"ph_holdings_app/pkg/kernel/jsondate"
)

// UnmarshalJSON parses an Order from frontend JSON, accepting date-only strings
// for order_date/required_date while leaving every other field to the default
// decoder via the embedded alias. An empty/absent date leaves the field at its
// zero value so GORM's Updates(struct) skips it.
func (o *Order) UnmarshalJSON(data []byte) error {
	type Alias Order
	aux := &struct {
		OrderDate    json.RawMessage `json:"order_date"`
		RequiredDate json.RawMessage `json:"required_date"`
		*Alias
	}{Alias: (*Alias)(o)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if t, ok, err := jsondate.ParseFlexible(aux.OrderDate); err != nil {
		return fmt.Errorf("order_date: %w", err)
	} else if ok {
		o.OrderDate = t
	}
	if t, ok, err := jsondate.ParseFlexible(aux.RequiredDate); err != nil {
		return fmt.Errorf("required_date: %w", err)
	} else if ok {
		o.RequiredDate = t
	}
	return nil
}
