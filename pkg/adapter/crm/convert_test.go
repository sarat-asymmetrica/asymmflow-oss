package crm

import (
	"testing"
	"time"

	gormcrm "ph_holdings_app/pkg/crm"
	shareddomain "ph_holdings_app/pkg/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerMasterRoundtrip(t *testing.T) {
	original := gormcrm.CustomerMaster{
		Base:             shareddomain.Base{ID: "customer-1", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), CreatedBy: "codex"},
		CustomerID:       "CUST-1",
		CustomerCode:     "GSC",
		BusinessName:     "Gulf Smelting Co.",
		PaymentGrade:     "A",
		CustomerGrade:    "B",
		OutstandingBHD:   2500.125,
		CreditLimitBHD:   50000,
		IsCreditBlocked:  true,
		MobileNumber:     "+973000",
		PrimaryEmail:     "finance@example.com",
		PaymentTermsDays: 30,
	}

	p, err := CustomerMasterToProto(original)
	require.NoError(t, err)
	back, err := CustomerMasterFromProto(*p)
	require.NoError(t, err)

	assert.Equal(t, original.ID, back.ID)
	assert.Equal(t, original.CustomerID, back.CustomerID)
	assert.Equal(t, original.CustomerCode, back.CustomerCode)
	assert.Equal(t, original.BusinessName, back.BusinessName)
	assert.Equal(t, original.PaymentGrade, back.PaymentGrade)
	assert.Equal(t, original.CustomerGrade, back.CustomerGrade)
	assert.Equal(t, original.OutstandingBHD, back.OutstandingBHD)
	assert.Equal(t, original.CreditLimitBHD, back.CreditLimitBHD)
	assert.Equal(t, original.IsCreditBlocked, back.IsCreditBlocked)
	assert.Equal(t, original.MobileNumber, back.MobileNumber)
}

func TestOrderRoundtrip(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 15, 0, 0, time.UTC)
	original := gormcrm.Order{
		Base:            shareddomain.Base{ID: "order-1", CreatedAt: now, UpdatedAt: now, CreatedBy: "codex"},
		OrderNumber:     "ORD-2026-001",
		CustomerID:      "customer-1",
		CustomerName:    "PH Test Customer",
		OrderDate:       now,
		RequiredDate:    now.AddDate(0, 0, 14),
		TotalValueBHD:   1000,
		GrandTotalBHD:   1100,
		Status:          "Sent",
		PaymentTerms:    "30 days",
		DeliveryTerms:   "DAP Bahrain",
		OfferID:         "offer-1",
		OfferNumber:     "OFF-1",
		Division:        "Acme Instrumentation",
		DiscountPercent: 2.5,
	}

	p, err := OrderToProto(original)
	require.NoError(t, err)
	back, err := OrderFromProto(*p)
	require.NoError(t, err)

	assert.Equal(t, original.ID, back.ID)
	assert.Equal(t, original.OrderNumber, back.OrderNumber)
	assert.Equal(t, original.CustomerID, back.CustomerID)
	assert.Equal(t, original.CustomerName, back.CustomerName)
	assert.Equal(t, original.TotalValueBHD, back.TotalValueBHD)
	assert.Equal(t, original.GrandTotalBHD, back.GrandTotalBHD)
	assert.Equal(t, original.Status, back.Status)
	assert.Equal(t, original.PaymentTerms, back.PaymentTerms)
}

func TestOfferRoundtrip(t *testing.T) {
	now := time.Date(2026, 5, 6, 11, 0, 0, 0, time.UTC)
	original := gormcrm.Offer{
		Base:            shareddomain.Base{ID: "offer-1", CreatedAt: now, UpdatedAt: now, CreatedBy: "codex"},
		OfferNumber:     "OFF-2026-001",
		CustomerID:      "customer-1",
		CustomerName:    "PH Test Customer",
		QuotationDate:   now,
		ValidityDate:    now.AddDate(0, 1, 0),
		TotalValueBHD:   1200,
		EstimatedMargin: 22.5,
		Stage:           "Won",
		PaymentTerms:    "30 days",
		DeliveryTerms:   "DAP Bahrain",
		VatRate:         10,
	}

	p, err := OfferToProto(original)
	require.NoError(t, err)
	back, err := OfferFromProto(*p)
	require.NoError(t, err)

	assert.Equal(t, original.ID, back.ID)
	assert.Equal(t, original.OfferNumber, back.OfferNumber)
	assert.Equal(t, original.CustomerID, back.CustomerID)
	assert.Equal(t, original.CustomerName, back.CustomerName)
	assert.Equal(t, original.TotalValueBHD, back.TotalValueBHD)
	assert.Equal(t, original.EstimatedMargin, back.EstimatedMargin)
	assert.Equal(t, original.Stage, back.Stage)
	assert.Equal(t, original.PaymentTerms, back.PaymentTerms)
	assert.Equal(t, original.DeliveryTerms, back.DeliveryTerms)
	assert.Equal(t, original.VatRate, back.VatRate)
}
