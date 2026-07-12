package main

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func makeDashboardTestApp(t *testing.T) *App {
	t.Helper()

	app := setupTestApp(t)
	app.startupImporting = false
	app.currentUser = &User{
		Base:     Base{ID: "dashboard-test-user"},
		Username: "dashboard-admin",
		RoleName: "admin",
		Role: Role{
			Name:        "admin",
			DisplayName: "Admin",
			Permissions: `["dashboard:view"]`,
		},
	}
	return app
}

func TestDashboardStats_PrefersConfirmedOrdersOverDraftInvoices(t *testing.T) {
	app := makeDashboardTestApp(t)

	orderDate := operationalMetricStartForYear(2026).AddDate(0, 0, 1)
	dueDate := orderDate.AddDate(0, 0, 30)

	order := Order{
		Base:          Base{ID: uuid.New().String(), CreatedAt: orderDate, UpdatedAt: orderDate},
		OrderNumber:   "ORD-2026-001",
		CustomerID:    uuid.New().String(),
		CustomerName:  "PH Test Customer",
		OrderDate:     orderDate,
		RequiredDate:  dueDate,
		GrandTotalBHD: 116128.811,
		Status:        "Confirmed",
	}
	require.NoError(t, app.db.Create(&order).Error)

	draftInvoice := Invoice{
		Base:           Base{ID: uuid.New().String(), CreatedAt: orderDate, UpdatedAt: orderDate},
		InvoiceNumber:  "PH2610-2026",
		InvoiceDate:    orderDate,
		CustomerID:     order.CustomerID,
		CustomerName:   order.CustomerName,
		OrderID:        order.ID,
		GrandTotalBHD:  505.12,
		Status:         "Draft",
		OutstandingBHD: 505.12,
		SubtotalBHD:    459.20,
		DueDate:        dueDate,
	}
	require.NoError(t, app.db.Create(&draftInvoice).Error)

	stats, err := app.GetDashboardStats()
	require.NoError(t, err)
	require.Equal(t, 2026, stats.ActivityYear)
	require.Equal(t, "2026-01-01", stats.FreshStartDate)
	require.Equal(t, 116128.811, stats.TotalRevenue)
	require.Equal(t, "Fresh orders from 1 Jan 2026", stats.RevenueMeta)
	require.Equal(t, 1, stats.PendingInvoices)
	require.Equal(t, 116128.811, stats.OutstandingAR)
}
