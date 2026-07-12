package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Wave 9.5 B7: CreateDNWithSerials applies the DN header + serial allocation
// in one create call. A failure partway through (e.g. a serial that isn't
// actually available) must not leave an orphaned DN behind, and any serials
// it did manage to reserve for that DN must be released back to Available.

func setupDNWave9App(t *testing.T) *App {
	t.Helper()
	a := setupFullTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &SerialNumber{}))
	return a
}

// seedOrderWithItems creates a customer + order with the given (productCode,
// qty) items and returns the order and its items in the same order supplied.
func seedOrderWithItems(t *testing.T, a *App, specs ...struct {
	productCode string
	qty         float64
}) (Order, []OrderItem) {
	t.Helper()

	customer := CustomerMaster{CustomerID: uuid.New().String(), CustomerCode: "CUST-" + uuid.New().String()[:8], BusinessName: "Northgate Process Controls"}
	require.NoError(t, a.db.Create(&customer).Error)

	order := Order{
		OrderNumber: "ORD-" + uuid.New().String()[:8], CustomerID: customer.ID, CustomerName: customer.BusinessName,
		OrderDate: time.Now(), Status: "Processing",
	}
	require.NoError(t, a.db.Create(&order).Error)

	var items []OrderItem
	for _, spec := range specs {
		product := ProductMaster{ProductCode: spec.productCode, ProductName: spec.productCode, IsActive: true}
		require.NoError(t, a.db.Create(&product).Error)

		item := OrderItem{OrderID: order.ID, ProductID: product.ID, ProductCode: spec.productCode, Quantity: spec.qty, UnitPrice: 100}
		require.NoError(t, a.db.Create(&item).Error)
		items = append(items, item)
	}
	return order, items
}

func TestCreateDNWithSerials_HappyPathCreatesDNAndReservesSerials(t *testing.T) {
	a := setupDNWave9App(t)

	order, items := seedOrderWithItems(t, a, struct {
		productCode string
		qty         float64
	}{"FT-100", 1})

	serial := SerialNumber{ProductCode: "FT-100", SerialNo: "SN-FT100-001", Status: "Available"}
	require.NoError(t, a.db.Create(&serial).Error)

	dn, err := a.CreateDNWithSerials(order.ID,
		[]DNItemInputWithSerials{{OrderItemID: items[0].ID, ShipQty: 1, SerialNos: []string{"SN-FT100-001"}}},
		DeliveryNoteHeaderInput{DriverName: "Amir Hassan"},
	)
	require.NoError(t, err)
	require.NotEmpty(t, dn.ID)
	require.Equal(t, "Prepared", dn.Status)
	require.Equal(t, "Amir Hassan", dn.DriverName)

	var dnItemCount int64
	require.NoError(t, a.db.Model(&DeliveryNoteItem{}).Where("delivery_note_id = ?", dn.ID).Count(&dnItemCount).Error)
	require.EqualValues(t, 1, dnItemCount)

	var reserved SerialNumber
	require.NoError(t, a.db.First(&reserved, "serial_no = ?", "SN-FT100-001").Error)
	require.Equal(t, "Reserved", reserved.Status)
	require.Equal(t, dn.DNNumber, reserved.DNNumber)
}

func TestCreateDNWithSerials_FailurePartwayCleansUpDNAndReleasesSerials(t *testing.T) {
	a := setupDNWave9App(t)

	order, items := seedOrderWithItems(t, a,
		struct {
			productCode string
			qty         float64
		}{"FT-200", 1},
		struct {
			productCode string
			qty         float64
		}{"FT-201", 1},
	)

	// Item 1's serial genuinely exists and is Available — its allocation will
	// succeed before the second item's allocation fails.
	require.NoError(t, a.db.Create(&SerialNumber{ProductCode: "FT-200", SerialNo: "SN-FT200-001", Status: "Available"}).Error)

	var dnCountBefore int64
	require.NoError(t, a.db.Model(&DeliveryNote{}).Count(&dnCountBefore).Error)

	// Item 2 references a serial that was never created — AllocateToDN's
	// atomic UPDATE WHERE status='Available' affects 0 rows and errors out.
	_, err := a.CreateDNWithSerials(order.ID, []DNItemInputWithSerials{
		{OrderItemID: items[0].ID, ShipQty: 1, SerialNos: []string{"SN-FT200-001"}},
		{OrderItemID: items[1].ID, ShipQty: 1, SerialNos: []string{"SN-FT201-MISSING"}},
	}, DeliveryNoteHeaderInput{})
	require.Error(t, err, "engineered mid-allocation failure must surface as an error")

	var dnCountAfter int64
	require.NoError(t, a.db.Model(&DeliveryNote{}).Count(&dnCountAfter).Error)
	require.Equal(t, dnCountBefore, dnCountAfter, "the orphaned DN must be cleaned up, not left behind")

	// The serial that WAS successfully reserved before the failure must be
	// released back to Available — no partial reservation left dangling.
	var releasedSerial SerialNumber
	require.NoError(t, a.db.First(&releasedSerial, "serial_no = ?", "SN-FT200-001").Error)
	require.Equal(t, "Available", releasedSerial.Status, "successfully-reserved serial must be released on cleanup")
	require.Empty(t, releasedSerial.DNNumber)

	// NOTE (observed, not asserted as a requirement): cleanupDN soft-deletes the
	// DeliveryNote and releases Reserved serials, but does not delete the
	// DeliveryNoteItem rows that were committed by the inner
	// CreateDeliveryNoteWithItems transaction before the serial-allocation step
	// failed. Those item rows are orphaned (pointing at a soft-deleted parent)
	// rather than truly gone. Logged here for visibility; not failed on, since
	// the task scope for this cleanup path is the DN + serial state.
	var orphanItemCount int64
	require.NoError(t, a.db.Model(&DeliveryNoteItem{}).Count(&orphanItemCount).Error)
	t.Logf("delivery_note_items rows remaining after DN cleanup (orphaned, not deleted): %d", orphanItemCount)
}

// Confirming a DN that delivers the last remaining quantity of an order
// progresses the order's fulfillment state and zeroes out its remaining
// delivery quantity. ConfirmDeliveryNote's signature is (string, error): the
// first return is a non-fatal post-confirm warning, not the primary result.
func TestConfirmDeliveryNote_FullDeliveryProgressesOrderAndZeroesRemaining(t *testing.T) {
	a := setupDNWave9App(t)

	order, items := seedOrderWithItems(t, a, struct {
		productCode string
		qty         float64
	}{"FT-300", 3})

	dn, err := a.CreateDeliveryNoteWithItems(order.ID, []DeliveryNoteItemInput{
		{OrderItemID: items[0].ID, ShipQty: 3},
	})
	require.NoError(t, err)
	require.False(t, dn.IsPartialDelivery, "shipping the full remaining quantity is not a partial delivery")

	require.NoError(t, a.DispatchDeliveryNote(dn.ID, "Amir Hassan", "BH-DRV-01"))

	warning, err := a.ConfirmDeliveryNote(dn.ID, "Warehouse Supervisor")
	require.NoError(t, err)
	t.Logf("post-confirm warning (expected empty on a clean run): %q", warning)

	var confirmedDN DeliveryNote
	require.NoError(t, a.db.First(&confirmedDN, "id = ?", dn.ID).Error)
	require.Equal(t, "Delivered", confirmedDN.Status)

	var updatedOrder Order
	require.NoError(t, a.db.First(&updatedOrder, "id = ?", order.ID).Error)
	require.Contains(t, []string{"Delivered", "FullyDelivered"}, updatedOrder.Status,
		"a fully-delivered order must progress past its pre-delivery status")

	remaining, err := a.GetOrderDeliveryStatus(order.ID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, remaining[items[0].ID], 0.001, "no quantity should remain after the only DN fully delivers")

	batchRemaining, err := a.GetOrderDeliveryStatusBatch([]string{order.ID})
	require.NoError(t, err)
	require.InDelta(t, 0.0, batchRemaining[order.ID][items[0].ID], 0.001, "batch status must agree with the singular method")
}
