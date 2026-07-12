package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Wave 9.5 B10: GetOrderDeliveryStatusBatch replaces an N+1 loop over
// GetOrderDeliveryStatus with two batched queries. These tests check the
// output shape against multiple orders, the empty-input short-circuit, and
// that a nonexistent order id is handled gracefully rather than panicking.

func seedBatchOrder(t *testing.T, a *App, qty, shipQty float64) (Order, OrderItem) {
	t.Helper()

	customer := CustomerMaster{CustomerID: uuid.New().String(), CustomerCode: "CUST-" + uuid.New().String()[:8], BusinessName: "Batch Test Customer"}
	require.NoError(t, a.db.Create(&customer).Error)

	order := Order{
		OrderNumber: "ORD-" + uuid.New().String()[:8], CustomerID: customer.ID, CustomerName: customer.BusinessName,
		OrderDate: time.Now(), Status: "Processing",
	}
	require.NoError(t, a.db.Create(&order).Error)

	item := OrderItem{OrderID: order.ID, ProductCode: "PRD-BATCH", Quantity: qty, UnitPrice: 50}
	require.NoError(t, a.db.Create(&item).Error)

	if shipQty > 0 {
		dnNumber := "DN-BATCH-" + uuid.New().String()[:8]
		dn := DeliveryNote{OrderID: order.ID, CustomerID: customer.ID, DNNumber: dnNumber, DeliveryDate: time.Now(), Status: "Prepared"}
		require.NoError(t, a.db.Create(&dn).Error)
		require.NoError(t, a.db.Create(&DeliveryNoteItem{
			DeliveryNoteID: dn.ID, OrderItemID: item.ID, ProductCode: "PRD-BATCH",
			QuantityOrdered: qty, QuantityDelivered: shipQty, QuantityRemaining: qty - shipQty,
		}).Error)
	}

	return order, item
}

func TestGetOrderDeliveryStatusBatch_ShapeAndAgreementWithSingularMethod(t *testing.T) {
	a := setupFullTestApp(t)

	orderA, itemA := seedBatchOrder(t, a, 10, 4) // 6 remaining
	orderB, itemB := seedBatchOrder(t, a, 5, 5)  // 0 remaining
	orderC, itemC := seedBatchOrder(t, a, 8, 0)  // 8 remaining (no DN at all)

	batch, err := a.GetOrderDeliveryStatusBatch([]string{orderA.ID, orderB.ID, orderC.ID})
	require.NoError(t, err)
	require.Len(t, batch, 3, "one entry per requested order")

	require.InDelta(t, 6.0, batch[orderA.ID][itemA.ID], 0.001)
	require.InDelta(t, 0.0, batch[orderB.ID][itemB.ID], 0.001)
	require.InDelta(t, 8.0, batch[orderC.ID][itemC.ID], 0.001)

	// Cross-check against the singular (non-batch) method for each order.
	for _, tc := range []struct {
		order Order
		item  OrderItem
	}{{orderA, itemA}, {orderB, itemB}, {orderC, itemC}} {
		single, err := a.GetOrderDeliveryStatus(tc.order.ID)
		require.NoError(t, err)
		require.InDelta(t, single[tc.item.ID], batch[tc.order.ID][tc.item.ID], 0.001,
			"batch result must agree with the singular GetOrderDeliveryStatus for order %s", tc.order.ID)
	}
}

func TestGetOrderDeliveryStatusBatch_EmptyInputReturnsEmptyResultNoError(t *testing.T) {
	a := setupFullTestApp(t)

	result, err := a.GetOrderDeliveryStatusBatch([]string{})
	require.NoError(t, err)
	require.Empty(t, result)

	result, err = a.GetOrderDeliveryStatusBatch(nil)
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestGetOrderDeliveryStatusBatch_NonexistentOrderIDHandledGracefully(t *testing.T) {
	a := setupFullTestApp(t)

	orderA, itemA := seedBatchOrder(t, a, 3, 1) // 2 remaining

	ghostID := uuid.New().String()
	result, err := a.GetOrderDeliveryStatusBatch([]string{orderA.ID, ghostID})
	require.NoError(t, err, "an unknown order id must not error the batch call")
	require.Len(t, result, 2, "an entry is still returned for the requested-but-missing id")

	require.InDelta(t, 2.0, result[orderA.ID][itemA.ID], 0.001)

	ghostEntry, ok := result[ghostID]
	require.True(t, ok, "the ghost id must have a (empty) entry, not be silently dropped")
	require.Empty(t, ghostEntry, "an order with no items/DNs resolves to an empty per-item map")
}
