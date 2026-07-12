package overlay

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The repo-shipped signature blocks must be SYNTHETIC (SYNTHETIC_IDENTITY.md):
// no real staff names, no real @platinumholdings emails, no real +973 17/3x
// phone numbers. Real identities arrive only via a sovereign overlay.json.
func TestBuiltinSignatureBlocksAreSynthetic(t *testing.T) {
	blocks := BuiltinDefaults().SignatureBlocks
	require.Len(t, blocks, 6, "expected six synthetic signature blocks")

	for _, b := range blocks {
		require.NotEmpty(t, b.DisplayName)
		assert.Equal(t, "ACME INSTRUMENTATION W.L.L", b.Company, "company must be the synthetic legal name")

		// Emails must be obviously fictional (.example TLD) and never the real domain.
		assert.True(t, strings.HasSuffix(b.Email, ".example"), "email %q must use the .example TLD", b.Email)
		assert.NotContains(t, strings.ToLower(b.Email), "platinumholdings")

		// Phones must be the obviously-fake +973-1700 range, never the real
		// PH office/fax lines (17587654 / 17564456) or real mobiles.
		for _, phone := range []string{b.Mobile, b.Office, b.Fax} {
			if phone == "" {
				continue
			}
			assert.True(t, strings.HasPrefix(phone, "+973-1700-"), "phone %q must be in the synthetic +973-1700 range", phone)
		}
		for _, line := range b.AddressLines {
			assert.NotContains(t, line, "815", "address must not carry the real PH PO box")
		}
	}
}

func TestSignatureBlockForMatchesNameAndAliases(t *testing.T) {
	o := BuiltinDefaults()

	byName, ok := o.SignatureBlockFor("Sam Rivera")
	require.True(t, ok)
	assert.Equal(t, "Business Development Manager", byName.Title)

	byAlias, ok := o.SignatureBlockFor("Sam")
	require.True(t, ok)
	assert.Equal(t, "Sam Rivera", byAlias.DisplayName, "alias must resolve to the canonical block")

	// "Support" is a role-word alias on the titleless block.
	support, ok := o.SignatureBlockFor("Support")
	require.True(t, ok)
	assert.Equal(t, "Casey Quinn", support.DisplayName)
	assert.Empty(t, support.Title, "the titleless block must stay titleless")

	_, ok = o.SignatureBlockFor("Nobody Special")
	assert.False(t, ok)

	_, ok = o.SignatureBlockFor("")
	assert.False(t, ok)
}

func TestSignatureBlockForIgnoresCaseAndPunctuation(t *testing.T) {
	o := BuiltinDefaults()
	for _, spelling := range []string{"sam rivera", "SAM  RIVERA", "Sam.Rivera", " sam  rivera "} {
		got, ok := o.SignatureBlockFor(spelling)
		require.True(t, ok, "spelling %q should match", spelling)
		assert.Equal(t, "Sam Rivera", got.DisplayName)
	}
}

func TestSignatureNamesInDeclarationOrder(t *testing.T) {
	assert.Equal(t,
		[]string{"Jordan Avery", "Alex Morgan", "Sam Rivera", "Casey Quinn", "Taylor Brooks", "Jamie Ellis"},
		BuiltinDefaults().SignatureNames(),
	)
}

func TestSignatureFallbackUsesConfiguredDefault(t *testing.T) {
	fb := BuiltinDefaults().SignatureFallback()
	assert.Empty(t, fb.DisplayName, "fallback carries no signer name; the caller stamps it")
	assert.Equal(t, "ACME INSTRUMENTATION W.L.L", fb.Company)
	assert.Equal(t, "sales@acme-instrumentation.example", fb.Email)
}

// With no SignatureDefault configured, the fallback is derived from the default
// division so a partial overlay.json still renders a coherent company block.
func TestSignatureFallbackDerivesFromDivisionWhenNil(t *testing.T) {
	o := BuiltinDefaults()
	o.SignatureDefault = nil
	fb := o.SignatureFallback()
	assert.Equal(t, "ACME INSTRUMENTATION W.L.L", fb.Company)
	assert.NotEmpty(t, fb.AddressLines)
}

// A sovereign overlay.json supplies real identities; this proves an override of
// the SignatureBlocks slice is honoured by resolution.
func TestOverlayOverrideHonored(t *testing.T) {
	o := BuiltinDefaults()
	o.SignatureBlocks = []SignatureBlockProfile{
		{
			DisplayName: "Real Person",
			Title:       "Managing Director",
			Company:     "REAL CO W.L.L",
			Email:       "real.person@realco.example",
			Aliases:     []string{"RP", "The MD"},
		},
	}

	got, ok := o.SignatureBlockFor("the md")
	require.True(t, ok)
	assert.Equal(t, "Real Person", got.DisplayName)
	assert.Equal(t, "Managing Director", got.Title)

	// The built-in synthetic names no longer resolve once overridden.
	_, ok = o.SignatureBlockFor("Sam Rivera")
	assert.False(t, ok)
}
