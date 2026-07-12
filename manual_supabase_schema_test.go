//go:build manual

package main

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

func TestManualMigrateSupabaseSchema(t *testing.T) {
	if os.Getenv("APPLY_SUPABASE_SCHEMA") != "1" {
		t.Skip("set APPLY_SUPABASE_SCHEMA=1 to migrate the configured Supabase schema")
	}

	require.NoError(t, godotenv.Overload("deploy_package/.env"))

	cfg := LoadSupabaseConfig()
	require.True(t, cfg.Enabled, "Supabase config must be enabled and complete")

	manager := NewDBManager(nil, cfg)
	require.NoError(t, manager.ConnectRemote())
	t.Cleanup(manager.Disconnect)

	// 1-SYNC (PH ffbe9c7): allow targeted pushes against an already-migrated
	// remote without re-running DDL.
	if os.Getenv("SKIP_REMOTE_MIGRATION") == "1" {
		t.Log("Skipping remote migration (schema already exists)")
	} else {
		require.NoError(t, manager.MigrateRemote())
	}
}

func TestManualCheckSupabaseConnection(t *testing.T) {
	if os.Getenv("CHECK_SUPABASE_CONNECTION") != "1" {
		t.Skip("set CHECK_SUPABASE_CONNECTION=1 to ping the configured Supabase database")
	}

	require.NoError(t, godotenv.Overload("deploy_package/.env"))

	cfg := LoadSupabaseConfig()
	require.True(t, cfg.Enabled, "Supabase config must be enabled and complete")

	manager := NewDBManager(nil, cfg)
	require.NoError(t, manager.ConnectRemote())
	t.Cleanup(manager.Disconnect)
	require.True(t, manager.CheckConnectivity())
}
