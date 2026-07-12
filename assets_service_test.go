package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitializeDefaultAssetsSeedsLetterheadsWithoutActiveUser(t *testing.T) {
	app := setupTestApp(t)

	require.NoError(t, app.EnsureAssetsTable())
	require.NoError(t, app.db.Where("1 = 1").Delete(&Asset{}).Error)

	app.currentUser = nil
	app.currentUserID = ""

	app.InitializeDefaultAssets()

	require.True(t, app.HasAsset(AssetLetterhead), "PH letterhead should seed without an active user")
	require.True(t, app.HasAsset(AssetLetterheadAHS), "AHS letterhead should seed without an active user")
}
