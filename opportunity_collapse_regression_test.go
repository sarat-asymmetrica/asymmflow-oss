package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOneDriveOpportunitiesDoNotCollapseToCustomerWord guards the PH Holdings
// production bug (ported from ph_holdings/opportunity_collapse_regression_test.go)
// where OneDrive-sourced opportunities collapsed onto a single canonical key per
// customer, hiding ~83 pipeline opportunities and cross-linking costings to the
// wrong opportunity.
//
// Root cause (two conspiring layers, both fixed in OSS):
//  1. parseOneDriveFolderMeta's loose fallback (splitLooseOneDriveFolderNumberToken)
//     returned a bare customer word like "BAPCO" as the folder number for inputs
//     such as "BAPCO LIT" — a real folder number always contains a digit.
//  2. normalizeOpportunityForList overwrote FolderNumber on EVERY metaCandidate for
//     OneDrive sources, and the LAST candidate is the bare Title, so it clobbered a
//     good "EH-103-26" with "BAPCO".
//
// canonicalOpportunityKey then keyed on the corrupted folder, so distinct
// opportunities collapsed (two Bapco deals both keyed "BAPCO", etc.).
//
// After the fix, the canonical keys of distinct opportunities must be pairwise
// distinct and must never be a bare customer word.
func TestOneDriveOpportunitiesDoNotCollapseToCustomerWord(t *testing.T) {
	// Representative OneDrive shapes from the triage evidence. Each carries a good
	// structured folder identity in FolderName, but a bare customer word in Title —
	// which the buggy loose fallback turned into the (clobbering) folder number.
	bapcoA := Opportunity{
		Source:       "2026_onedrive",
		FolderName:   "EH-103-26 BAPCO LIT",
		FolderNumber: "EH-103-26",
		Title:        "BAPCO LIT",
		Year:         2026,
		OppNumber:    103,
	}
	bapcoB := Opportunity{
		Source:       "2026_onedrive",
		FolderName:   "OTH-04-26 BAPCO PARTS",
		FolderNumber: "OTH-04-26",
		Title:        "BAPCO PARTS",
		Year:         2026,
		OppNumber:    4,
	}
	veolia := Opportunity{
		Source:       "2026_onedrive",
		FolderName:   "EH-22-26 VEOLIA TIT",
		FolderNumber: "EH-22-26",
		Title:        "VEOLIA TIT",
		Year:         2026,
		OppNumber:    22,
	}

	keyA := canonicalOpportunityKey(normalizeOpportunityForList(bapcoA))
	keyB := canonicalOpportunityKey(normalizeOpportunityForList(bapcoB))
	keyV := canonicalOpportunityKey(normalizeOpportunityForList(veolia))

	// The actual collapse: the two Bapco deals must NOT share a canonical key
	// (under the old behavior both became "BAPCO").
	require.NotEqualf(t, keyA, keyB,
		"two distinct Bapco opportunities collapsed to the same canonical key %q", keyA)
	require.NotEqual(t, keyA, keyV, "Bapco and Veolia opportunities collapsed to the same key")
	require.NotEqual(t, keyB, keyV, "Bapco and Veolia opportunities collapsed to the same key")

	// And no canonical key may be a bare customer word.
	for _, k := range []string{keyA, keyB, keyV} {
		require.NotEqual(t, "BAPCO", k, "canonical key must not be a bare customer word")
		require.NotEqual(t, "VEOLIA", k, "canonical key must not be a bare customer word")
		require.Truef(t, folderNumberHasDigit(k),
			"canonical key %q must be a real folder number (contain a digit)", k)
	}
}

// TestLooseOneDriveFolderHelpersRejectDigitlessTokens pins the D1 layer-1 fix: the
// loose folder-number helpers must reject purely alphabetic tokens (customer names)
// while still accepting real, digit-bearing folder numbers.
func TestLooseOneDriveFolderHelpersRejectDigitlessTokens(t *testing.T) {
	// Digit-less customer words are NOT folder numbers.
	require.Equal(t, "", cleanLooseOneDriveFolderNumberToken("BAPCO"),
		"a digit-less token must not be accepted as a folder number")
	require.Equal(t, "", cleanLooseOneDriveFolderNumberToken("VEOLIA"))

	folder, _ := splitLooseOneDriveFolderNumberToken("BAPCO")
	require.Equal(t, "", folder, "splitLoose must return an empty folder for a digit-less token")

	require.Equal(t, "", parseOneDriveFolderMeta("BAPCO LIT").FolderNumber,
		"a bare customer folder name must not yield a folder number")

	// Positive guard: digit-bearing loose tokens must still parse (no over-rejection).
	require.Equal(t, "22", cleanLooseOneDriveFolderNumberToken("22-"))
	folder, _ = splitLooseOneDriveFolderNumberToken("150-AIRMENCH")
	require.Equal(t, "150", folder, "a real numeric folder token must still be accepted")
}
