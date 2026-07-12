package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

func normalizeRecordForRemoteSync(record map[string]any) map[string]any {
	if record == nil {
		return nil
	}

	normalized := make(map[string]any, len(record))
	for key, value := range record {
		if isSyncBooleanColumn(key) {
			if boolValue, ok := coerceSyncBoolean(value); ok {
				normalized[key] = boolValue
				continue
			}
		}
		normalized[key] = value
	}
	return normalized
}

func normalizeRecordsForRemoteSync(records []map[string]any) []map[string]any {
	normalized := make([]map[string]any, 0, len(records))
	for _, record := range records {
		normalized = append(normalized, normalizeRecordForRemoteSync(record))
	}
	return normalized
}

func normalizeRecordForRemoteSyncSchema(record map[string]any, columnTypes map[string]string) map[string]any {
	if record == nil {
		return nil
	}
	if len(columnTypes) == 0 {
		return normalizeRecordForRemoteSync(record)
	}

	normalized := make(map[string]any, len(record))
	for key, value := range record {
		dataType, ok := columnTypes[strings.ToLower(strings.TrimSpace(key))]
		if !ok {
			continue
		}
		if isSyncDatabaseBooleanType(dataType) {
			if boolValue, ok := coerceSyncBoolean(value); ok {
				normalized[key] = boolValue
				continue
			}
		}
		normalized[key] = value
	}
	return normalized
}

func normalizeRecordsForRemoteSyncSchema(records []map[string]any, columnTypes map[string]string) []map[string]any {
	normalized := make([]map[string]any, 0, len(records))
	for _, record := range records {
		normalized = append(normalized, normalizeRecordForRemoteSyncSchema(record, columnTypes))
	}
	return normalized
}

func syncUpsertColumns(records []map[string]any) []string {
	seen := make(map[string]struct{})
	for _, record := range records {
		for column := range record {
			if strings.EqualFold(column, "id") {
				continue
			}
			seen[column] = struct{}{}
		}
	}

	columns := make([]string, 0, len(seen))
	for column := range seen {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	return columns
}

func syncDBColumnTypes(db *gorm.DB, table string) (map[string]string, error) {
	if db == nil {
		return nil, nil
	}
	columnTypes, err := db.Migrator().ColumnTypes(table)
	if err != nil {
		return nil, err
	}

	types := make(map[string]string, len(columnTypes))
	for _, columnType := range columnTypes {
		types[strings.ToLower(columnType.Name())] = strings.ToLower(columnType.DatabaseTypeName())
	}
	return types, nil
}

func isSyncDatabaseBooleanType(dataType string) bool {
	t := strings.ToLower(strings.TrimSpace(dataType))
	return t == "bool" || t == "boolean" || strings.Contains(t, "bool")
}

func syncUpsertRecordByIDOrNaturalKey(db *gorm.DB, table string, record map[string]any, columnTypes map[string]string) error {
	if db == nil {
		return nil
	}
	id, hasID := record["id"]
	updateRecord := syncRecordWithoutID(record)
	if hasID {
		result := db.Table(table).Where("id = ?", id).Updates(updateRecord)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
	}
	if column, value, ok := syncNaturalKeyValue(table, record, columnTypes); ok {
		result := db.Table(table).Where(column+" = ?", value).Updates(updateRecord)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
	}
	return db.Table(table).Create(record).Error
}

func syncRecordWithoutID(record map[string]any) map[string]any {
	copied := make(map[string]any, len(record))
	for key, value := range record {
		if strings.EqualFold(key, "id") {
			continue
		}
		copied[key] = value
	}
	return copied
}

func syncNaturalKeyValue(table string, record map[string]any, columnTypes map[string]string) (string, any, bool) {
	column := syncNaturalKeyColumn(table, columnTypes)
	if column == "" {
		return "", nil, false
	}
	value, ok := record[column]
	if !ok || value == nil {
		return "", nil, false
	}
	if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
		return "", nil, false
	}
	return column, value, true
}

func syncNaturalKeyColumn(table string, columnTypes map[string]string) string {
	switch strings.ToLower(strings.TrimSpace(table)) {
	case "suppliers":
		if syncColumnExists(columnTypes, "supplier_code") {
			return "supplier_code"
		}
	default:
		return ""
	}
	return ""
}

func syncColumnExists(columnTypes map[string]string, column string) bool {
	if len(columnTypes) == 0 {
		return true
	}
	_, ok := columnTypes[strings.ToLower(strings.TrimSpace(column))]
	return ok
}

func isSyncBooleanColumn(column string) bool {
	c := strings.ToLower(strings.TrimSpace(column))
	if c == "" {
		return false
	}

	for _, prefix := range []string{"is_", "has_", "requires_", "was_", "auto_", "must_"} {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}

	for _, suffix := range []string{
		"_ok", "_verified", "_matched", "_posted", "_active", "_primary",
		"_acknowledged", "_stored", "_automatic", "_reversed", "_stale",
		"_reconciled", "_encrypted", "_blocked",
	} {
		if strings.HasSuffix(c, suffix) {
			return true
		}
	}

	switch c {
	case "user_price_set", "balance_verified", "po_match_ok", "grn_match_ok",
		"table_detected", "gpu_used", "dna_cache_hit":
		return true
	default:
		return false
	}
}

func coerceSyncBoolean(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case int:
		return coerceSyncBooleanNumber(float64(v))
	case int8:
		return coerceSyncBooleanNumber(float64(v))
	case int16:
		return coerceSyncBooleanNumber(float64(v))
	case int32:
		return coerceSyncBooleanNumber(float64(v))
	case int64:
		return coerceSyncBooleanNumber(float64(v))
	case uint:
		return coerceSyncBooleanNumber(float64(v))
	case uint8:
		return coerceSyncBooleanNumber(float64(v))
	case uint16:
		return coerceSyncBooleanNumber(float64(v))
	case uint32:
		return coerceSyncBooleanNumber(float64(v))
	case uint64:
		return coerceSyncBooleanNumber(float64(v))
	case float32:
		return coerceSyncBooleanNumber(float64(v))
	case float64:
		return coerceSyncBooleanNumber(v)
	case []byte:
		return coerceSyncBooleanString(string(v))
	case string:
		return coerceSyncBooleanString(v)
	default:
		return coerceSyncBooleanString(fmt.Sprint(v))
	}
}

func coerceSyncBooleanNumber(value float64) (bool, bool) {
	switch value {
	case 0:
		return false, true
	case 1:
		return true, true
	default:
		return false, false
	}
}

func coerceSyncBooleanString(value string) (bool, bool) {
	s := strings.TrimSpace(strings.ToLower(value))
	switch s {
	case "true", "t", "yes", "y":
		return true, true
	case "false", "f", "no", "n":
		return false, true
	case "0", "1":
		parsed, err := strconv.Atoi(s)
		if err != nil {
			return false, false
		}
		return parsed == 1, true
	default:
		return false, false
	}
}
