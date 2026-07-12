package finance

import (
	"testing"
	"time"

	shareddomain "ph_holdings_app/pkg/domain"
	gormfinance "ph_holdings_app/pkg/finance"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestInvoiceRoundtrip(t *testing.T) {
	now := time.Date(2026, 5, 6, 9, 30, 0, 0, time.UTC)
	buyerOrderDate := now.AddDate(0, 0, -2)
	deliveryDate := now.AddDate(0, 0, 1)
	original := gormfinance.Invoice{
		Base: shareddomain.Base{
			ID:        "invoice-1",
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-time.Hour),
			DeletedAt: gorm.DeletedAt{},
			CreatedBy: "codex",
		},
		InvoiceNumber:        "INV-2026-001",
		InvoiceDate:          now,
		CustomerID:           "customer-1",
		CustomerName:         "PH Test Customer",
		OrderID:              "order-1",
		CustomerPONumber:     "PO-77",
		GrandTotalBHD:        1234.567,
		Status:               "Sent",
		OutstandingBHD:       900,
		SubtotalBHD:          1122.333,
		DueDate:              now.AddDate(0, 0, 30),
		UpdatedBy:            "finance-user",
		RfqID:                "rfq-1",
		QuoteID:              "quote-1",
		OfferID:              "offer-1",
		OfferNumber:          "OFF-1",
		DeliveryNoteID:       "dn-1",
		DeliveryNoteNumber:   "DN-1",
		TotalSupplierCostBHD: 700,
		GrossMarginBHD:       422.333,
		GrossMarginPercent:   37.63,
		CustomerReference:    "REF-1",
		AttentionPerson:      "the maintainer",
		AttentionCompany:     "Asymmetrica",
		AttentionPhone:       "+973000",
		AttentionAddress:     "Bahrain",
		DeliveryWeeks:        "4-6 weeks",
		CountryOfOrigin:      "Germany",
		IssuedBy:             "Finance",
		ContactPhone:         "+973111",
		DiscountPercent:      3.5,
		PaymentTerms:         "30 days",
		DeliveryTerms:        "DAP Bahrain",
		Division:             "Acme Instrumentation",
		FieldVisibility:      `{"show_margin":false}`,
		DeliveryNoteRef:      "EH/253/25",
		ModeOfPayment:        "Bank Transfer",
		SuppliersRef:         "SUP-9",
		OtherReferences:      "Rhine Instruments",
		BuyersOrderNumber:    "LPS-11347",
		BuyersOrderDate:      &buyerOrderDate,
		DespatchDocumentNo:   "DES-1",
		DeliveryNoteDate:     &deliveryDate,
		DespatchedThrough:    "Direct",
		Destination:          "Bahrain",
		PlaceOfSupply:        "Kingdom of Bahrain",
		TermsOfDelivery:      "Direct Bank Transfer",
		VATBHD:               112.234,
		VATPercent:           10,
		JournalEntryID:       "je-1",
		InvoiceHash:          "hash",
		Items: []gormfinance.DBInvoiceItem{
			{
				Base:                shareddomain.Base{ID: "item-1", CreatedAt: now, UpdatedAt: now, CreatedBy: "codex"},
				InvoiceID:           "invoice-1",
				LineNumber:          1,
				Description:         "Pressure transmitter",
				Quantity:            2,
				Rate:                100,
				TotalBHD:            200,
				ProductID:           "product-1",
				ProductCode:         "P-1",
				Equipment:           "Instrument",
				Model:               "M1",
				Specification:       "Spec",
				DetailedDescription: "Detailed spec",
				Currency:            "USD",
				FOB:                 50,
				Freight:             5,
				TotalCost:           60,
				MarginPercent:       20,
				TotalPrice:          200,
			},
		},
	}

	protoInvoice, err := InvoiceToProto(original)
	require.NoError(t, err)
	back, err := InvoiceFromProto(*protoInvoice)
	require.NoError(t, err)

	assert.Equal(t, original.ID, back.ID)
	assert.Equal(t, original.InvoiceNumber, back.InvoiceNumber)
	assert.Equal(t, original.InvoiceDate, back.InvoiceDate)
	assert.Equal(t, original.CustomerID, back.CustomerID)
	assert.Equal(t, original.CustomerName, back.CustomerName)
	assert.Equal(t, original.OrderID, back.OrderID)
	assert.Equal(t, original.CustomerPONumber, back.CustomerPONumber)
	assert.Equal(t, original.GrandTotalBHD, back.GrandTotalBHD)
	assert.Equal(t, original.Status, back.Status)
	assert.Equal(t, original.OutstandingBHD, back.OutstandingBHD)
	assert.Equal(t, original.SubtotalBHD, back.SubtotalBHD)
	assert.Equal(t, original.DueDate, back.DueDate)
	assert.Equal(t, original.PaymentTerms, back.PaymentTerms)
	assert.Equal(t, original.DeliveryTerms, back.DeliveryTerms)
	assert.Equal(t, original.BuyersOrderDate, back.BuyersOrderDate)
	assert.Equal(t, original.DeliveryNoteDate, back.DeliveryNoteDate)
	assert.Equal(t, original.VATBHD, back.VATBHD)
	assert.Equal(t, original.VATPercent, back.VATPercent)
	require.Len(t, back.Items, 1)
	assert.Equal(t, original.Items[0].Description, back.Items[0].Description)
	assert.Equal(t, original.Items[0].Currency, back.Items[0].Currency)
	assert.Equal(t, original.Items[0].TotalPrice, back.Items[0].TotalPrice)
}
