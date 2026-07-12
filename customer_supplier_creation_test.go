package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func makeCreationTestApp(t *testing.T) *App {
	t.Helper()

	app := setupTestApp(t)
	app.startupImporting = false
	app.currentUser = &User{
		Base:     Base{ID: "test-user"},
		Username: "test-admin",
		RoleName: "admin",
		Role: Role{
			Name:        "admin",
			DisplayName: "Admin",
			Permissions: `["customers:create","customers:view","suppliers:create","suppliers:view"]`,
		},
	}
	return app
}

func TestCustomerCreation_WithValidRole(t *testing.T) {
	app := makeCreationTestApp(t)

	customer := CustomerMaster{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		BusinessName:     "Creation Check Customer",
		CustomerType:     "Corporate",
		Country:          "Bahrain",
		PaymentGrade:     "B",
		CustomerGrade:    "B",
		PaymentTermsDays: 30,
	}

	created, err := app.CreateCustomer(customer)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEmpty(t, created.CustomerCode)
	require.NotEmpty(t, created.CustomerID)
	require.Equal(t, "Creation Check Customer", created.BusinessName)
}

func TestSupplierCreation_WithValidRole(t *testing.T) {
	app := makeCreationTestApp(t)

	supplier := SupplierMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierName: "Creation Check Supplier",
		SupplierType: "Manufacturer",
		Country:      "Bahrain",
		LeadTimeDays: 14,
	}

	created, err := app.CreateSupplier(supplier)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEmpty(t, created.SupplierCode)
	require.Equal(t, "Creation Check Supplier", created.SupplierName)
	require.Equal(t, 3, created.Rating)
}

func TestCustomerCreation_RequiresCreatePermission(t *testing.T) {
	app := makeCreationTestApp(t)
	app.currentUser = &User{
		Base:     Base{ID: "limited-user"},
		Username: "limited-sales",
		RoleName: "sales",
		Role: Role{
			Name:        "sales",
			DisplayName: "Sales",
			Permissions: `["customers:view"]`,
		},
	}

	customer := CustomerMaster{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		BusinessName:     "Blocked Customer",
		CustomerType:     "Corporate",
		Country:          "Bahrain",
		PaymentGrade:     "B",
		CustomerGrade:    "B",
		PaymentTermsDays: 30,
	}

	created, err := app.CreateCustomer(customer)
	require.Error(t, err)
	require.Nil(t, created)
	require.Contains(t, err.Error(), "customers:create")
}

func TestSupplierCreation_RequiresCreatePermission(t *testing.T) {
	app := makeCreationTestApp(t)
	app.currentUser = &User{
		Base:     Base{ID: "limited-user"},
		Username: "limited-sales",
		RoleName: "sales",
		Role: Role{
			Name:        "sales",
			DisplayName: "Sales",
			Permissions: `["suppliers:view"]`,
		},
	}

	supplier := SupplierMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierName: "Blocked Supplier",
		SupplierType: "Manufacturer",
		Country:      "Bahrain",
		LeadTimeDays: 14,
	}

	created, err := app.CreateSupplier(supplier)
	require.Error(t, err)
	require.Nil(t, created)
	require.Contains(t, err.Error(), "suppliers:create")
}
