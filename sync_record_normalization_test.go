package main

import (
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNormalizeRecordForRemoteSyncCoercesSQLiteBooleanIntegers(t *testing.T) {
	record := map[string]any{
		"id":                  "customer-1",
		"is_credit_blocked":   int64(0),
		"requires_prepayment": int64(1),
		"has_abb_competition": []byte("0"),
		"is_emergency_only":   "1",
		"total_orders_count":  int64(1),
		"business_name":       "NPC",
	}

	normalized := normalizeRecordForRemoteSync(record)
	require.Equal(t, false, normalized["is_credit_blocked"])
	require.Equal(t, true, normalized["requires_prepayment"])
	require.Equal(t, false, normalized["has_abb_competition"])
	require.Equal(t, true, normalized["is_emergency_only"])
	require.Equal(t, int64(1), normalized["total_orders_count"])
	require.Equal(t, "NPC", normalized["business_name"])
}

func TestSyncUpsertColumnsExcludesPrimaryKeyAndIsStable(t *testing.T) {
	columns := syncUpsertColumns([]map[string]any{
		{"id": "a", "updated_at": "now", "is_active": true},
		{"id": "b", "created_at": "then", "is_primary": false},
	})

	require.Equal(t, []string{"created_at", "is_active", "is_primary", "updated_at"}, columns)
}

func TestNormalizeRecordForRemoteSyncSchemaHonorsRemoteTypesAndColumns(t *testing.T) {
	record := map[string]any{
		"id":                   "supplier-1",
		"is_active":            int64(1),
		"supplier_part_number": "remote-does-not-have-this-column",
		"supplier_name":        "PetroSpan",
	}

	normalized := normalizeRecordForRemoteSyncSchema(record, map[string]string{
		"id":            "text",
		"is_active":     "int8",
		"supplier_name": "text",
	})

	require.Equal(t, int64(1), normalized["is_active"])
	require.Equal(t, "PetroSpan", normalized["supplier_name"])
	require.NotContains(t, normalized, "supplier_part_number")
}

func TestNormalizeRecordForRemoteSyncSchemaCoercesRemoteBooleans(t *testing.T) {
	normalized := normalizeRecordForRemoteSyncSchema(map[string]any{
		"id":        "employee-1",
		"is_active": int64(1),
	}, map[string]string{
		"id":        "text",
		"is_active": "bool",
	})

	require.Equal(t, true, normalized["is_active"])
}

func TestSyncNaturalKeyValueUsesSupplierCode(t *testing.T) {
	column, value, ok := syncNaturalKeyValue("suppliers", map[string]any{
		"id":            "local-id",
		"supplier_code": "PESPAN",
	}, map[string]string{
		"id":            "text",
		"supplier_code": "text",
	})

	require.True(t, ok)
	require.Equal(t, "supplier_code", column)
	require.Equal(t, "PESPAN", value)
	require.NotContains(t, syncRecordWithoutID(map[string]any{"id": "local-id", "supplier_code": "PESPAN"}), "id")
}

func TestSyncTableSkipsMissingRemoteTable(t *testing.T) {
	local, err := gorm.Open(sqlite.Open("file:sync_local?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	remote, err := gorm.Open(sqlite.Open("file:sync_remote?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, local.Exec(`CREATE TABLE sync_widgets (id TEXT PRIMARY KEY, name TEXT, updated_at DATETIME)`).Error)
	require.NoError(t, local.Exec(`INSERT INTO sync_widgets (id, name, updated_at) VALUES (?, ?, ?)`, "w1", "Widget", time.Now()).Error)

	manager := &DBManager{local: local, remote: remote}
	synced, err := manager.syncTable("sync_widgets", time.Time{}, "push")
	require.NoError(t, err)
	require.Equal(t, 0, synced)
}

func TestSyncTableSkipsSourceWithoutUpdatedAt(t *testing.T) {
	local, err := gorm.Open(sqlite.Open("file:sync_local_no_updated_at?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	remote, err := gorm.Open(sqlite.Open("file:sync_remote_no_updated_at?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, local.Exec(`CREATE TABLE no_timestamp_widgets (id TEXT PRIMARY KEY, name TEXT)`).Error)
	require.NoError(t, remote.Exec(`CREATE TABLE no_timestamp_widgets (id TEXT PRIMARY KEY, name TEXT)`).Error)
	require.NoError(t, local.Exec(`INSERT INTO no_timestamp_widgets (id, name) VALUES (?, ?)`, "w1", "Widget").Error)

	manager := &DBManager{local: local, remote: remote}
	synced, err := manager.syncTable("no_timestamp_widgets", time.Time{}, "push")
	require.NoError(t, err)
	require.Equal(t, 0, synced)
}

func TestSyncTablePullFiltersColumnsMissingFromLocalSchema(t *testing.T) {
	local, err := gorm.Open(sqlite.Open("file:sync_local_extra_remote_column?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	remote, err := gorm.Open(sqlite.Open("file:sync_remote_extra_remote_column?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, local.Exec(`CREATE TABLE order_items (id TEXT PRIMARY KEY, total_price REAL, updated_at DATETIME)`).Error)
	require.NoError(t, remote.Exec(`CREATE TABLE order_items (id TEXT PRIMARY KEY, total_price REAL, total_price_bhd REAL, updated_at DATETIME)`).Error)
	require.NoError(t, remote.Exec(`INSERT INTO order_items (id, total_price, total_price_bhd, updated_at) VALUES (?, ?, ?, ?)`, "oi-1", 125.5, 125.5, time.Now()).Error)

	manager := &DBManager{local: local, remote: remote}
	synced, err := manager.syncTable("order_items", time.Time{}, "pull")
	require.NoError(t, err)
	require.Equal(t, 1, synced)

	var total float64
	require.NoError(t, local.Raw(`SELECT total_price FROM order_items WHERE id = ?`, "oi-1").Scan(&total).Error)
	require.Equal(t, 125.5, total)
}

func TestFullDatabaseDownloadFiltersColumnsMissingFromLocalSchema(t *testing.T) {
	local, err := gorm.Open(sqlite.Open("file:download_local_extra_remote_column?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	remote, err := gorm.Open(sqlite.Open("file:download_remote_extra_remote_column?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, local.Exec(`CREATE TABLE order_items (id TEXT PRIMARY KEY, total_price REAL, updated_at DATETIME)`).Error)
	require.NoError(t, remote.Exec(`CREATE TABLE order_items (id TEXT PRIMARY KEY, total_price REAL, total_price_bhd REAL, updated_at DATETIME)`).Error)
	require.NoError(t, remote.Exec(`INSERT INTO order_items (id, total_price, total_price_bhd, updated_at) VALUES (?, ?, ?, ?)`, "oi-1", 225.75, 225.75, time.Now()).Error)

	manager := &DBManager{
		local:                     local,
		remote:                    remote,
		canPullActivityMonitoring: func() bool { return false },
	}
	downloaded, err := manager.DownloadFullDatabase()
	require.NoError(t, err)
	require.Equal(t, 1, downloaded)

	var total float64
	require.NoError(t, local.Raw(`SELECT total_price FROM order_items WHERE id = ?`, "oi-1").Scan(&total).Error)
	require.Equal(t, 225.75, total)
}
