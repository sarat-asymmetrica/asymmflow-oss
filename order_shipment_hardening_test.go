package main

// Wave 9.7 tight-ship-2: (a) CreateOrderWithItems replaces the two-call
// CreateOrder+UpdateOrder sequence the frontend used for manual order
// creation, which could leave a header-only "ghost" order behind if the
// second call failed. This file proves the new method is atomic — header
// and items land together, or neither does. (b) CreateShipment used to
// dereference a nil *time.Time when estimatedDelivery was empty or failed
// to parse; this file proves that path is now panic-safe.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreateOrderWithItems_Atomic(t *testing.T) {
	app := setupTestApp(t)

	order := Order{
		OrderNumber:  "ORD-ATOMIC-001",
		CustomerName: "Atomic Test Customer",
		OrderDate:    time.Now(),
		Status:       "Confirmed",
	}
	items := []OrderItem{
		{Description: "Pressure Transmitter", ProductCode: "PT-100", Quantity: 2, UnitPrice: 150},
		{Description: "Flow Meter", ProductCode: "FM-200", Quantity: 1, UnitPrice: 500},
	}

	created, err := app.CreateOrderWithItems(order, items)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "ORD-ATOMIC-001", created.OrderNumber)

	// Header persisted.
	var dbOrder Order
	require.NoError(t, app.db.Where("order_number = ?", "ORD-ATOMIC-001").First(&dbOrder).Error)

	// Both item rows persisted, linked to the header.
	var dbItems []OrderItem
	require.NoError(t, app.db.Where("order_id = ?", dbOrder.ID).Find(&dbItems).Error)
	require.Len(t, dbItems, 2)

	// Total is derived from the items, matching UpdateOrder's item-replacement math.
	require.InDelta(t, 800.0, dbOrder.TotalValueBHD, 0.001) // (2*150) + (1*500)
	require.InDelta(t, 800.0, dbOrder.GrandTotalBHD, 0.001)

	// GetOrdersWithNoItems must NOT surface this order as an itemless ghost.
	orphans, err := app.GetOrdersWithNoItems()
	require.NoError(t, err)
	for _, o := range orphans {
		require.NotEqual(t, "ORD-ATOMIC-001", o["order_number"])
	}
}

// TestCreateOrderWithItems_RollsBackOnItemFailure injects a primary-key
// collision on the second Create (the item batch insert) to prove the
// single-transaction design: if item persistence fails, the order header
// created moments earlier in the same transaction must not survive either.
func TestCreateOrderWithItems_RollsBackOnItemFailure(t *testing.T) {
	app := setupTestApp(t)

	// Pre-seed a row occupying the primary key our "new" item will also carry,
	// so the batch insert of items collides and fails.
	conflict := OrderItem{
		Base:        Base{ID: "conflict-item-id"},
		OrderID:     "some-other-order",
		Description: "Pre-existing row",
		Quantity:    1,
		UnitPrice:   1,
	}
	require.NoError(t, app.db.Create(&conflict).Error)

	order := Order{
		OrderNumber:  "ORD-ATOMIC-FAIL",
		CustomerName: "Atomic Failure Customer",
		OrderDate:    time.Now(),
		Status:       "Confirmed",
	}
	items := []OrderItem{
		{Base: Base{ID: "conflict-item-id"}, Description: "Colliding row", Quantity: 1, UnitPrice: 100},
	}

	_, err := app.CreateOrderWithItems(order, items)
	require.Error(t, err)

	// The order header must NOT have been left behind — this is the exact
	// "ghost order" bug the atomic transaction exists to prevent.
	var count int64
	require.NoError(t, app.db.Model(&Order{}).Where("order_number = ?", "ORD-ATOMIC-FAIL").Count(&count).Error)
	require.EqualValues(t, 0, count, "order header must not persist when item insert fails inside the transaction")
}

// TestCreateShipment_NoPanicOnEmptyDate covers the latent nil-deref: an empty
// (or unparseable) estimatedDelivery used to leave shipmentDate as a nil
// *time.Time that was then unconditionally dereferenced, panicking. The only
// caller that sent such input (DeliveryTrackingScreen.svelte) has been
// retired as an orphaned screen, but CreateShipment stays bound and callable,
// so it must be robust regardless of caller.
func TestCreateShipment_NoPanicOnEmptyDate(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Shipment{}))

	require.NotPanics(t, func() {
		err := app.CreateShipment([]string{"order-empty-date"}, "TRACK-EMPTY", "DHL", "", "")
		require.NoError(t, err)
	})

	var shipment Shipment
	require.NoError(t, app.db.Where("tracking_number = ?", "TRACK-EMPTY").First(&shipment).Error)
	require.True(t, shipment.ShipmentDate.IsZero(), "empty estimatedDelivery should leave a zero ShipmentDate, not panic")

	// An unparseable date string must also degrade gracefully, not panic.
	require.NotPanics(t, func() {
		err := app.CreateShipment([]string{"order-bad-date"}, "TRACK-BAD", "DHL", "not-a-date", "")
		require.NoError(t, err)
	})

	var badShipment Shipment
	require.NoError(t, app.db.Where("tracking_number = ?", "TRACK-BAD").First(&badShipment).Error)
	require.True(t, badShipment.ShipmentDate.IsZero())

	// A valid "2006-01-02" date should still parse and populate correctly.
	require.NotPanics(t, func() {
		err := app.CreateShipment([]string{"order-valid-date"}, "TRACK-VALID", "DHL", "2026-08-01", "")
		require.NoError(t, err)
	})

	var validShipment Shipment
	require.NoError(t, app.db.Where("tracking_number = ?", "TRACK-VALID").First(&validShipment).Error)
	require.False(t, validShipment.ShipmentDate.IsZero())
	require.Equal(t, 2026, validShipment.ShipmentDate.Year())
	require.Equal(t, time.Month(8), validShipment.ShipmentDate.Month())
	require.Equal(t, 1, validShipment.ShipmentDate.Day())
}
