// Package toon provides a small Token-Oriented Object Notation encoder for
// Butler prompt context.
package toon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const indentStep = "  "

// Marshal converts JSON-compatible Go data to a compact TOON-style document.
func Marshal(v any) (string, error) {
	normalized, err := normalize(v)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	writeValue(&b, "", normalized, 0)
	return strings.TrimRight(b.String(), "\n"), nil
}

// EstimatedTokens returns a simple character-based token estimate suitable for
// relative JSON-vs-TOON comparisons in tests and logs.
func EstimatedTokens(s string) int {
	if s == "" {
		return 0
	}
	return (len(s) + 3) / 4
}

// Savings compares compact JSON with TOON output for the same value.
func Savings(v any) (jsonBytes, toonBytes int, err error) {
	compact, err := json.Marshal(v)
	if err != nil {
		return 0, 0, err
	}
	encoded, err := Marshal(v)
	if err != nil {
		return 0, 0, err
	}
	return len(compact), len(encoded), nil
}

func normalize(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch v.(type) {
	case map[string]any, []any, string, bool, json.Number, float64, float32, int, int64, int32, uint, uint64, uint32:
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(v); err != nil {
			return nil, err
		}
		dec := json.NewDecoder(&buf)
		dec.UseNumber()
		var out any
		if err := dec.Decode(&out); err != nil {
			return nil, err
		}
		return out, nil
	default:
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(v); err != nil {
			return nil, err
		}
		dec := json.NewDecoder(&buf)
		dec.UseNumber()
		var out any
		if err := dec.Decode(&out); err != nil {
			return nil, err
		}
		return out, nil
	}
}

func writeValue(b *strings.Builder, key string, v any, depth int) {
	prefix := strings.Repeat(indentStep, depth)
	if key != "" {
		switch typed := v.(type) {
		case map[string]any:
			b.WriteString(prefix)
			b.WriteString(key)
			b.WriteString(":\n")
			writeObject(b, typed, depth+1)
		case []any:
			writeArray(b, key, typed, depth)
		default:
			b.WriteString(prefix)
			b.WriteString(key)
			b.WriteString(": ")
			b.WriteString(formatScalar(typed))
			b.WriteByte('\n')
		}
		return
	}

	switch typed := v.(type) {
	case map[string]any:
		writeObject(b, typed, depth)
	case []any:
		writeArray(b, "items", typed, depth)
	default:
		b.WriteString(prefix)
		b.WriteString(formatScalar(typed))
		b.WriteByte('\n')
	}
}

func writeObject(b *strings.Builder, obj map[string]any, depth int) {
	keys := sortedKeys(obj)
	for _, key := range keys {
		writeValue(b, key, obj[key], depth)
	}
}

func writeArray(b *strings.Builder, key string, arr []any, depth int) {
	prefix := strings.Repeat(indentStep, depth)
	if len(arr) == 0 {
		b.WriteString(prefix)
		b.WriteString(key)
		b.WriteString("[0]:\n")
		return
	}
	if fields, ok := tabularFields(arr); ok {
		b.WriteString(prefix)
		b.WriteString(fmt.Sprintf("%s[%d]{%s}:\n", key, len(arr), strings.Join(fields, ",")))
		for _, item := range arr {
			row := item.(map[string]any)
			b.WriteString(prefix)
			b.WriteString(indentStep)
			for i, field := range fields {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteString(formatCell(row[field]))
			}
			b.WriteByte('\n')
		}
		return
	}
	if allScalar(arr) {
		b.WriteString(prefix)
		b.WriteString(fmt.Sprintf("%s[%d]: ", key, len(arr)))
		for i, item := range arr {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(formatCell(item))
		}
		b.WriteByte('\n')
		return
	}

	b.WriteString(prefix)
	b.WriteString(fmt.Sprintf("%s[%d]:\n", key, len(arr)))
	for _, item := range arr {
		switch typed := item.(type) {
		case map[string]any:
			b.WriteString(prefix)
			b.WriteString(indentStep)
			b.WriteString("-\n")
			writeObject(b, typed, depth+2)
		default:
			b.WriteString(prefix)
			b.WriteString(indentStep)
			b.WriteString("- ")
			b.WriteString(formatScalar(typed))
			b.WriteByte('\n')
		}
	}
}

func tabularFields(arr []any) ([]string, bool) {
	var fields []string
	for i, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok || len(obj) == 0 {
			return nil, false
		}
		keys := sortedKeys(obj)
		for _, key := range keys {
			if !isScalar(obj[key]) {
				return nil, false
			}
		}
		if i == 0 {
			fields = keys
			continue
		}
		if !reflect.DeepEqual(fields, keys) {
			return nil, false
		}
	}
	return fields, true
}

func allScalar(arr []any) bool {
	for _, item := range arr {
		if !isScalar(item) {
			return false
		}
	}
	return true
}

func isScalar(v any) bool {
	switch v.(type) {
	case nil, string, bool, json.Number, float64:
		return true
	default:
		return false
	}
}

func sortedKeys(obj map[string]any) []string {
	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatScalar(v any) string {
	switch typed := v.(type) {
	case nil:
		return "null"
	case string:
		return quoteIfNeeded(typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return quoteIfNeeded(fmt.Sprint(typed))
	}
}

func formatCell(v any) string {
	s := formatScalar(v)
	if strings.ContainsAny(s, ",\n\r") {
		return strconv.Quote(s)
	}
	return s
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	if strings.TrimSpace(s) != s || strings.ContainsAny(s, ":,{}[]\n\r\t\"") {
		return strconv.Quote(s)
	}
	lower := strings.ToLower(s)
	if lower == "null" || lower == "true" || lower == "false" {
		return strconv.Quote(s)
	}
	return s
}
