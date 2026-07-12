package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateInvoiceWithOptions_DeliveryNoteUsesDeliveredQuantity(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&DeliveryNote{}, &DeliveryNoteItem{}))

	now := time.Now()
	customerID := seedTestCustomer(t, app.db, "Invoice DN Customer")
	orderID := uuid.New().String()
	orderItemID := uuid.New().String()

	order := Order{
		Base:             Base{ID: orderID, CreatedAt: now, UpdatedAt: now},
		OrderNumber:      "ORD-INV-DN-001",
		CustomerID:       customerID,
		CustomerName:     "Invoice DN Customer",
		CustomerPONumber: "PO-123",
		OrderDate:        now,
		RequiredDate:     now.AddDate(0, 0, 30),
		TotalValueBHD:    50,
		GrandTotalBHD:    55,
		Status:           "Delivered",
		PaymentTerms:     "30 Days",
		DeliveryTerms:    "Direct Delivery",
		Division:         "Acme Instrumentation",
		Items: []OrderItem{{
			Base:        Base{ID: orderItemID, CreatedAt: now, UpdatedAt: now},
			OrderID:     orderID,
			LineNumber:  1,
			Description: "Pressure transmitter",
			Quantity:    10,
			UnitPrice:   5,
			TotalPrice:  50,
		}},
	}
	require.NoError(t, app.db.Create(&order).Error)

	dn := DeliveryNote{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		OrderID:      orderID,
		CustomerID:   customerID,
		DNNumber:     "DN-INV-DN-001",
		DeliveryDate: now,
		Status:       "Delivered",
		Items: []DeliveryNoteItem{{
			Base:              Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
			OrderItemID:       orderItemID,
			Description:       "Pressure transmitter",
			QuantityOrdered:   10,
			QuantityDelivered: 4,
			QuantityRemaining: 6,
		}},
	}
	require.NoError(t, app.db.Create(&dn).Error)

	invoice, err := app.CreateInvoiceWithOptions(orderID, dn.ID, defaultInvoiceFieldVisibilityJSON())
	require.NoError(t, err)
	require.Len(t, invoice.Items, 1)
	require.InDelta(t, 4, invoice.Items[0].Quantity, 0.001)
	require.InDelta(t, 20, invoice.Items[0].TotalBHD, 0.001)
	require.InDelta(t, 20, invoice.SubtotalBHD, 0.001)
	require.InDelta(t, 2, invoice.VATBHD, 0.001)
	require.InDelta(t, 22, invoice.GrandTotalBHD, 0.001)
	require.Equal(t, "DN-INV-DN-001", invoice.DeliveryNoteNumber)
	require.Equal(t, "DN-INV-DN-001", invoice.DespatchDocumentNo)
	require.NotNil(t, invoice.DeliveryNoteDate)

	var updatedItem OrderItem
	require.NoError(t, app.db.First(&updatedItem, "id = ?", orderItemID).Error)
	require.InDelta(t, 4, updatedItem.QuantityInvoiced, 0.001)
}
