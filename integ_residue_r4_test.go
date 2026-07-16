package main

// INTEG residue campaign — Wave R4: AI-provider key encrypted at rest.
// The frontend wires the Butler/Mistral key WRITE through SetAPIKeys (persists
// via SetSetting encrypt=true) and READS it back MASKED via the new
// GetAIProviderKeyStatus binding — which reads the SAME encrypted settings-DB
// store SetAPIKeys writes to (GetSettings reads settings.json, a different
// store, so it can't reflect a SetAPIKeys write; owner ruling 2026-07-16 was to
// add this DB-backed masked read so the round-trip is honest). This proves:
// set → stored encrypted → masked read-back, plaintext never returned.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegR4_AIProviderKeyMaskedReadBack(t *testing.T) {
	app := setupFullTestApp(t)

	// Not set yet → honest "(not set)", never an error.
	before, err := app.GetAIProviderKeyStatus()
	require.NoError(t, err)
	require.False(t, before.IsSet)
	require.Equal(t, "(not set)", before.MaskedKey)

	const key = "sk-test-mistral-abcdefghijkl-9876"
	require.NoError(t, app.SetAPIKeys(map[string]string{"mistral_key": key}))

	// Stored encrypted — never plaintext at rest.
	var setting Setting
	require.NoError(t, app.db.Where("key = ?", "apiKeys.mistral_key").First(&setting).Error)
	require.True(t, setting.IsEncrypted, "the key is encrypted at rest")
	require.NotContains(t, setting.Value, "mistral", "ciphertext must not contain the plaintext")

	// GetAIProviderKeyStatus reads back the SAME store, MASKED to last-4.
	status, err := app.GetAIProviderKeyStatus()
	require.NoError(t, err)
	require.True(t, status.IsSet, "a saved key reads back as set")
	require.Equal(t, key[:4]+"****"+key[len(key)-4:], status.MaskedKey, "masked to last-4")
	require.NotEqual(t, key, status.MaskedKey, "the plaintext key is never returned")
}
