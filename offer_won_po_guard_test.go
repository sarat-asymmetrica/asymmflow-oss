package main

// Wave 8 P2-2: MarkOfferWon must require a non-blank customer PO number before
// an offer can be marked won (the refactor dropped PH's trim+empty guard, so an
// offer could be won with a blank/whitespace PO).

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarkOfferWonRequiresCustomerPO(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Offer{}, &OfferItem{}))

	// REJECT: empty PO — guard returns before any offer lookup.
	_, err := app.MarkOfferWon("some-offer-id", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "customer PO number is required")

	// REJECT: whitespace-only PO (trim must collapse to empty).
	_, err = app.MarkOfferWon("some-offer-id", "   ")
	require.Error(t, err)
	require.Contains(t, err.Error(), "customer PO number is required")

	// ALLOW (guard passes): a non-blank PO gets past the guard and reaches the
	// offer lookup, which fails with a *different* error — proving the PO guard
	// did not reject valid input.
	_, err = app.MarkOfferWon("nonexistent-offer-id", "PO-12345")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "customer PO number is required")
	require.Contains(t, err.Error(), "offer not found")
}
