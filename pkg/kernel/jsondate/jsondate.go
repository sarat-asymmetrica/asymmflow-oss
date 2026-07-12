// Package jsondate parses JSON date fields that may arrive date-only.
// HTML <input type="date"> emits "2006-01-02"; normalized payloads emit
// RFC3339. Go's default time.Time unmarshal only accepts RFC3339, so
// date-only strings from an argument-binding layer would otherwise fail.
package jsondate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var flexibleLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02",
	"02-Jan-2006",
	"02/01/2006",
}

// ParseFlexible parses a json.RawMessage holding a JSON string that may be
// date-only or a full timestamp. ok=false means absent/null/empty — the
// caller MUST leave the existing field untouched in that case so GORM's
// Updates(struct) continues to skip the zero value.
func ParseFlexible(raw json.RawMessage) (time.Time, bool, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return time.Time{}, false, nil
	}
	var s string
	if err := json.Unmarshal(trimmed, &s); err != nil {
		// Not a JSON string — tolerate a raw JSON timestamp that time.Time
		// itself knows how to decode.
		var t time.Time
		if err2 := json.Unmarshal(trimmed, &t); err2 == nil {
			return t, true, nil
		}
		return time.Time{}, false, fmt.Errorf("invalid date value %s", string(trimmed))
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false, nil
	}
	for _, layout := range flexibleLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("unrecognized date format %q", s)
}
