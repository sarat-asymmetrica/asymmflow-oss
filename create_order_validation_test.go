package main

// Wave 8 P2-3 (RES-004): CreateOrder must validate orderNumber (1-100 chars),
// customerName (1-255 chars) and status (<=50 chars) — the refactor dropped
// PH's validation, so empty/oversized inputs were written straight to the DB.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateOrderValidatesStringInputs(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CustomerMaster{}, &Order{}))

	// REJECT: empty order number
	_, err := app.CreateOrder("", "Acme Corp", 100.0, "2026-07-09", "Confirmed")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Order number is required")

	// REJECT: whitespace-only order number (trim collapses to empty)
	_, err = app.CreateOrder("   ", "Acme Corp", 100.0, "2026-07-09", "Confirmed")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Order number is required")

	// REJECT: order number over 100 chars
	_, err = app.CreateOrder(strings.Repeat("X", 101), "Acme Corp", 100.0, "2026-07-09", "Confirmed")
	require.Error(t, err)
	require.Contains(t, err.Error(), "at most 100 characters")

	// REJECT: empty customer name
	_, err = app.CreateOrder("ORD-001", "", 100.0, "2026-07-09", "Confirmed")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Customer name is required")

	// REJECT: customer name over 255 chars
	_, err = app.CreateOrder("ORD-001", strings.Repeat("Y", 256), 100.0, "2026-07-09", "Confirmed")
	require.Error(t, err)
	require.Contains(t, err.Error(), "at most 255 characters")

	// REJECT: status over 50 chars
	_, err = app.CreateOrder("ORD-001", "Acme Corp", 100.0, "2026-07-09", strings.Repeat("Z", 51))
	require.Error(t, err)
	require.Contains(t, err.Error(), "Order status must be at most 50 characters")

	// ALLOW: valid inputs create the order (and trimming is applied)
	order, err := app.CreateOrder("  ORD-100  ", "  Acme Corp  ", 100.0, "2026-07-09", "Confirmed")
	require.NoError(t, err)
	require.NotNil(t, order)
	require.Equal(t, "ORD-100", order.OrderNumber)    // trimmed
	require.Equal(t, "Acme Corp", order.CustomerName) // trimmed
}
