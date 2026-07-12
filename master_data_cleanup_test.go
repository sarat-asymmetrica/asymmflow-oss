package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMasterDataCleanupAuditDetectsLowRiskCustomerAndSupplierDuplicates(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CustomerContact{}, &SupplierContact{}, &Opportunity{}, &Offer{}, &PurchaseOrder{}, &SupplierInvoice{}, &SupplierPayment{}, &ProductMaster{}))

	primaryCustomer := CustomerMaster{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerID:       "CUST-GSC-001",
		CustomerCode:     "CUST-GSC-001",
		BusinessName:     "Gulf Smelting B.S.C",
		CustomerType:     "Corporate",
		City:             "Manama",
		Country:          "Bahrain",
		PaymentGrade:     "A",
		CustomerGrade:    "A",
		PaymentTermsDays: 30,
	}
	duplicateCustomer := CustomerMaster{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now().Add(time.Minute), UpdatedAt: time.Now().Add(time.Minute)},
		CustomerID:       "CUST-GSC-LEGACY",
		CustomerCode:     "CUST-GSC-LEGACY",
		BusinessName:     "Gulf Smelting B.S.C.(C)",
		CustomerType:     "Corporate",
		City:             "Manama",
		Country:          "Bahrain",
		PaymentGrade:     "A",
		CustomerGrade:    "A",
		PaymentTermsDays: 30,
	}
	require.NoError(t, app.db.Create(&primaryCustomer).Error)
	require.NoError(t, app.db.Create(&duplicateCustomer).Error)
	require.NoError(t, app.db.Create(&Invoice{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		InvoiceNumber:  "INV-CLEAN-001",
		InvoiceDate:    time.Now(),
		CustomerID:     primaryCustomer.ID,
		CustomerName:   primaryCustomer.BusinessName,
		GrandTotalBHD:  100,
		SubtotalBHD:    100,
		OutstandingBHD: 100,
		Status:         "Sent",
		DueDate:        time.Now().Add(72 * time.Hour),
	}).Error)
	require.NoError(t, app.db.Create(&CustomerContact{
		Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerID:  duplicateCustomer.ID,
		ContactName: "Legacy contact",
		Email:       "legacy@example.com",
	}).Error)

	primarySupplier := SupplierMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierCode: "SUP-EH-001",
		SupplierName: "Rhine Instruments",
		Country:      "Germany",
		SupplierType: "Manufacturer",
		PaymentTerms: "Net 30",
	}
	duplicateSupplier := SupplierMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now().Add(time.Minute), UpdatedAt: time.Now().Add(time.Minute)},
		SupplierCode: "SUP-EH-LEGACY",
		SupplierName: "Rhine Instruments W.L.L.",
		Country:      "Germany",
		SupplierType: "Manufacturer",
		PaymentTerms: "Net 30",
	}
	require.NoError(t, app.db.Create(&primarySupplier).Error)
	require.NoError(t, app.db.Create(&duplicateSupplier).Error)
	require.NoError(t, app.db.Create(&PurchaseOrder{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		PONumber:     "PO-CLEAN-001",
		SupplierID:   primarySupplier.ID,
		SupplierName: primarySupplier.SupplierName,
		PODate:       time.Now(),
		Status:       "Sent",
		Currency:     "BHD",
		TotalBHD:     55,
	}).Error)

	require.Equal(t, normalizeMasterDataName(primaryCustomer.BusinessName), normalizeMasterDataName(duplicateCustomer.BusinessName))
	require.Equal(t, normalizeMasterDataName(primarySupplier.SupplierName), normalizeMasterDataName(duplicateSupplier.SupplierName))

	audit, err := buildMasterDataCleanupAudit(app.db)
	require.NoError(t, err)
	require.NotEmpty(t, audit.CustomerCandidates)
	require.NotEmpty(t, audit.SupplierCandidates)

	require.Equal(t, "GULF SMELTING", audit.CustomerCandidates[0].NormalizedName)
	require.True(t, audit.CustomerCandidates[0].AutoMergeSafe)
	require.Equal(t, primaryCustomer.ID, audit.CustomerCandidates[0].PrimaryID)

	require.Equal(t, "RHINE INSTRUMENTS", audit.SupplierCandidates[0].NormalizedName)
	require.True(t, audit.SupplierCandidates[0].AutoMergeSafe)
	require.Equal(t, primarySupplier.ID, audit.SupplierCandidates[0].PrimaryID)
}

func TestValidateParsedStatementFlagsContradictions(t *testing.T) {
	parsed := &parsedStatement{
		OpeningBalance: 100,
		ClosingBalance: 160,
		Lines: []parsedLine{
			{LineNumber: 1, Date: time.Now(), Debit: 10, Credit: 5, Balance: 95},
			{LineNumber: 2, Date: time.Now(), Debit: 0, Credit: 20, Balance: 110},
		},
	}

	validation := validateParsedStatement(parsed)
	require.True(t, validation.Blocking)
	require.NotEmpty(t, validation.BlockingIssues)
}
