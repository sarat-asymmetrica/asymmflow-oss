package main

import (
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGenerateDeliveryNotePDFKeepsFiveCompactRowsOnOnePage(t *testing.T) {
	t.Chdir(t.TempDir())
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&DeliveryNote{}, &DeliveryNoteItem{}))

	now := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)
	customerID := seedTestCustomer(t, app.db, "RIVERSIDE POWER OPERATION AND MAINTENANCE COMPANY W.L.L")
	orderID := uuid.New().String()

	order := Order{
		Base:             Base{ID: orderID, CreatedAt: now, UpdatedAt: now},
		OrderNumber:      "ORD-INV-2025-0001",
		CustomerPONumber: "INV-2025-0001",
		CustomerID:       customerID,
		CustomerName:     "RIVERSIDE POWER OPERATION AND MAINTENANCE COMPANY W.L.L",
		OrderDate:        time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
		RequiredDate:     now,
		Status:           "Delivered",
		Division:         "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&order).Error)

	items := []DeliveryNoteItem{
		{Description: "Per Order ORD-INV-2025-0001"},
		{Description: "Promag P 300, 5P3B1H, DN100 4\" - 5P3B1H-24897/101"},
		{Description: "Promag, grounding disc/protection disc - DK5GD-14N3/0"},
		{Description: "Promag P 300, 5P3B1F, DN150 6\" - 5P3B1F-1NDD9/101"},
		{Description: "Promag, grounding disc/protection disc - DK5GD-11J4/0"},
	}
	for i := range items {
		items[i].Base = Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now}
		items[i].QuantityOrdered = 1
		items[i].QuantityDelivered = 1
	}

	dn := DeliveryNote{
		Base:            Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		DNNumber:        "DN-ORD-INV-2025-0001",
		OrderID:         orderID,
		CustomerID:      customerID,
		DeliveryDate:    now,
		DeliveryAddress: "RIVERSIDE POWER OPERATION AND MAINTENANCE COMPANY W.L.L\nManama",
		Status:          "Delivered",
		Items:           items,
	}
	require.NoError(t, app.db.Create(&dn).Error)

	path, err := app.GenerateDeliveryNotePDF(dn.ID)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	pageMarkers := regexp.MustCompile(`/Type\s*/Page\b`).FindAll(data, -1)
	require.Len(t, pageMarkers, 1)
}

func TestGenerateDeliveryNotePDFPaginatesManyRowsBeforeFooter(t *testing.T) {
	t.Chdir(t.TempDir())
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&DeliveryNote{}, &DeliveryNoteItem{}))

	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	customerID := seedTestCustomer(t, app.db, "AQUAPURE ENERGY GULF WLL")
	orderID := uuid.New().String()

	order := Order{
		Base:             Base{ID: orderID, CreatedAt: now, UpdatedAt: now},
		OrderNumber:      "037PONO2600011",
		CustomerPONumber: "037PONO2600011",
		CustomerID:       customerID,
		CustomerName:     "AQUAPURE ENERGY GULF WLL",
		OrderDate:        time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		RequiredDate:     now,
		Status:           "Delivered",
		Division:         "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&order).Error)

	items := make([]DeliveryNoteItem, 0, 28)
	for i := 0; i < 28; i++ {
		items = append(items, DeliveryNoteItem{
			Base:              Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
			Description:       "Micropilot FMR20B - FMR20B field instrument with calibration certificate",
			ProductCode:       "FMR20B",
			QuantityOrdered:   3,
			QuantityDelivered: 3,
		})
	}

	dn := DeliveryNote{
		Base:            Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		DNNumber:        "DN-037PONO2600011",
		OrderID:         orderID,
		CustomerID:      customerID,
		DeliveryDate:    now,
		DeliveryAddress: "AquaPure Energy Gulf WLL\nHello hello",
		Status:          "Delivered",
		Items:           items,
	}
	require.NoError(t, app.db.Create(&dn).Error)

	path, err := app.GenerateDeliveryNotePDF(dn.ID)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	pageMarkers := regexp.MustCompile(`/Type\s*/Page\b`).FindAll(data, -1)
	require.GreaterOrEqual(t, len(pageMarkers), 2)
}
