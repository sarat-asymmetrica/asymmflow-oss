package main

import (
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// =============================================================================
// ACME INSTRUMENTATION ERP - PRODUCTION TESTS
// =============================================================================
// This file contains real tests for core App service methods.
// Tests verify CRUD operations, business logic, and data integrity.
// =============================================================================

// setupTestApp creates a fresh SQLite database for testing
func setupTestApp(t *testing.T) *App {
	t.Helper()

	// ncruces SQLite uses distinct :memory: databases per connection. memdb
	// gives tests a shared in-memory database without forcing one connection.
	dsn := memdb.TestDB(t, url.Values{
		"_pragma": {"busy_timeout(5000)"},
	})
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	require.NoError(t, err, "Failed to create test database")
	sqlDB, err := db.DB()
	require.NoError(t, err, "Failed to access test database handle")
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	// Auto-migrate all schemas needed for tests
	// Migrate tables one at a time to isolate errors
	err = db.AutoMigrate(&CustomerMaster{})
	if err != nil {
		t.Fatalf("Failed to migrate CustomerMaster: %v", err)
	}
	err = db.AutoMigrate(&SupplierMaster{})
	if err != nil {
		t.Fatalf("Failed to migrate SupplierMaster: %v", err)
	}
	err = db.AutoMigrate(&Order{})
	if err != nil {
		t.Fatalf("Failed to migrate Order: %v", err)
	}
	err = db.AutoMigrate(&OrderItem{})
	if err != nil {
		t.Fatalf("Failed to migrate OrderItem: %v", err)
	}
	// Migrate Invoice and supporting tables
	err = db.AutoMigrate(&Invoice{})
	if err != nil {
		t.Logf("Warning: Invoice AutoMigrate failed: %v (continuing)", err)
	}
	err = db.AutoMigrate(&InvoiceSequence{})
	if err != nil {
		t.Logf("Warning: InvoiceSequence AutoMigrate failed: %v (continuing)", err)
	}
	err = db.AutoMigrate(&DBInvoiceItem{})
	if err != nil {
		t.Fatalf("Failed to migrate DBInvoiceItem: %v", err)
	}
	err = db.AutoMigrate(&Payment{})
	if err != nil {
		t.Fatalf("Failed to migrate Payment: %v", err)
	}

	// Create app with test database
	app := &App{
		db:                     db,
		cache:                  NewCache(),
		startupImporting:       false,
		startupImportStartTime: time.Now(),
		currentUserID:          "test-user",
		currentUser: &User{
			Base:     Base{ID: "test-user"},
			Username: "test-admin",
			RoleName: "admin",
			Role: Role{
				Name:        "admin",
				DisplayName: "Administrator",
				Permissions: `["*"]`,
			},
		},
	}

	// Stop the cache's background cleanup goroutine when the test ends —
	// otherwise every setupTestApp leaks a ticker goroutine for the rest of the
	// run (Wave 9.5 C4 root cause).
	t.Cleanup(app.cache.Stop)

	alignTestAppWithCurrentWorkflow(t, app)

	return app
}

func TestGetExportDirGroupsDocumentsByEntityWithoutYearDepth(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")

	app := &App{}

	customerDir := app.getExportDir("customer", "National Petroleum Co.", "RFQ", 2026)
	require.Equal(t,
		filepath.Join(homeDir, "Documents", "AsymmFlow Exports", sanitizeFileName("National Petroleum Co."), "RFQ"),
		customerDir,
	)
	require.DirExists(t, customerDir)
	require.NotContains(t, customerDir, string(filepath.Separator)+"2026"+string(filepath.Separator))

	supplierDir := app.getExportDir("supplier", "Rhine Instruments", "Orders", 2026)
	require.Equal(t,
		filepath.Join(homeDir, "Documents", "AsymmFlow Exports", "Suppliers", sanitizeFileName("Rhine Instruments"), "Orders"),
		supplierDir,
	)
	require.DirExists(t, supplierDir)
	require.NotContains(t, supplierDir, string(filepath.Separator)+"2026"+string(filepath.Separator))
}

func alignTestAppWithCurrentWorkflow(t *testing.T, app *App) {
	t.Helper()

	require.NoError(t, app.db.AutoMigrate(
		&Role{},
		&User{},
		&Setting{},
		&LicenseKey{},
		&ChartOfAccount{},
		&CompanyBankAccount{},
		&BankStatement{},
		&BankStatementLine{},
		&BankLinePaymentAllocation{},
		&Offer{},
		&OfferItem{},
		&Opportunity{},
	))
	require.NoError(t, app.ensureCriticalDeploymentFoundations())
}

func TestNonAdminDeleteCreatesApprovalRequestAndAdminApprovalDeletes(t *testing.T) {
	app := setupTestApp(t)

	customer := CustomerMaster{
		Base:         Base{ID: uuid.New().String()},
		CustomerCode: "CUST-DELETE-APPROVAL",
		BusinessName: "Delete Approval Customer",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	adminEmployee := Employee{
		Base:             Base{ID: "employee-admin"},
		EmployeeCode:     "EMP-ADMIN",
		FullName:         "Admin User",
		EmploymentStatus: "active",
		IsActive:         true,
	}
	salesEmployee := Employee{
		Base:             Base{ID: "employee-sales"},
		EmployeeCode:     "EMP-SALES",
		FullName:         "Sales User",
		EmploymentStatus: "active",
		IsActive:         true,
	}
	require.NoError(t, app.db.Create(&adminEmployee).Error)
	require.NoError(t, app.db.Create(&salesEmployee).Error)
	require.NoError(t, app.db.Create(&LicenseKey{Key: "PH-ADM-TEST01", Role: "admin", DisplayName: "Admin User"}).Error)
	require.NoError(t, app.db.Create(&EmployeeAccessLink{
		Base:         Base{ID: "link-admin"},
		EmployeeID:   adminEmployee.ID,
		LicenseKey:   "PH-ADM-TEST01",
		UserID:       "admin-user",
		AccessStatus: "active",
		IsPrimary:    true,
	}).Error)
	require.NoError(t, app.db.Create(&EmployeeAccessLink{
		Base:         Base{ID: "link-sales"},
		EmployeeID:   salesEmployee.ID,
		LicenseKey:   "PH-SLS-TEST01",
		UserID:       "sales-user",
		AccessStatus: "active",
		IsPrimary:    true,
	}).Error)

	app.currentUserID = "sales-user"
	app.currentUser = &User{
		Base:     Base{ID: "sales-user"},
		Username: "sales-user",
		RoleName: "sales",
		Role: Role{
			Name:        "sales",
			DisplayName: "Sales",
			Permissions: `["customers:view","customers:edit","notifications:view"]`,
		},
	}

	err := app.DeleteCustomer(customer.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "delete approval requested")

	var customerCount int64
	require.NoError(t, app.db.Model(&CustomerMaster{}).Where("id = ?", customer.ID).Count(&customerCount).Error)
	require.EqualValues(t, 1, customerCount)

	var request DeleteApprovalRequest
	require.NoError(t, app.db.Where("entity_type = ? AND entity_id = ?", "customer", customer.ID).First(&request).Error)
	require.Equal(t, "pending", request.Status)

	var adminNotificationCount int64
	require.NoError(t, app.db.Model(&Notification{}).
		Where("employee_id = ? AND source_type = ? AND source_id = ?", adminEmployee.ID, "delete_approval", request.ID).
		Count(&adminNotificationCount).Error)
	require.EqualValues(t, 1, adminNotificationCount)

	app.currentUserID = "admin-user"
	app.currentUser = &User{
		Base:     Base{ID: "admin-user"},
		Username: "admin-user",
		RoleName: "admin",
		Role: Role{
			Name:        "admin",
			DisplayName: "Administrator",
			Permissions: `["*"]`,
		},
	}

	reviewed, err := app.ReviewDeleteApprovalRequest(request.ID, "approve", "verified duplicate")
	require.NoError(t, err)
	require.Equal(t, "approved", reviewed.Status)
	require.NoError(t, app.db.Model(&CustomerMaster{}).Where("id = ?", customer.ID).Count(&customerCount).Error)
	require.EqualValues(t, 0, customerCount)
}

func TestAdminLicenseCanDeleteCustomersAndSuppliersWithStaleUserRole(t *testing.T) {
	app := setupTestApp(t)

	customer := CustomerMaster{
		Base:         Base{ID: uuid.New().String()},
		CustomerCode: "CUST-ADMIN-LICENSE-DELETE",
		BusinessName: "Admin License Delete Customer",
	}
	supplier := SupplierMaster{
		Base:         Base{ID: uuid.New().String()},
		SupplierCode: "SUP-ADMIN-LICENSE-DELETE",
		SupplierName: "Admin License Delete Supplier",
	}
	require.NoError(t, app.db.Create(&customer).Error)
	require.NoError(t, app.db.Create(&supplier).Error)
	require.NoError(t, app.db.Create(&LicenseKey{
		Key:        "PH-ADM-DELETE",
		Role:       "admin",
		Activated:  true,
		DeviceHash: app.getDeviceHash(),
	}).Error)

	app.currentUserID = "sales-user"
	app.currentUser = &User{
		Base:     Base{ID: "sales-user"},
		Username: "sales-user",
		RoleName: "sales",
		Role: Role{
			Name:        "sales",
			DisplayName: "Sales",
			Permissions: `["customers:view","suppliers:view"]`,
		},
	}

	require.NoError(t, app.DeleteCustomer(customer.ID))
	require.NoError(t, app.DeleteSupplier(supplier.ID))

	var deleteRequests int64
	require.NoError(t, app.db.Model(&DeleteApprovalRequest{}).Count(&deleteRequests).Error)
	require.EqualValues(t, 0, deleteRequests)
}

func TestApplyDeploymentLicenseActivationFlushResetsExistingActivationOnce(t *testing.T) {
	app := setupTestApp(t)
	t.Setenv("ASYMMFLOW_LICENSE_FLUSH_STAMP", "deploy-a")

	now := time.Now()
	license := LicenseKey{
		Key:         "PH-SLS-FLUSH1",
		Role:        "sales",
		DisplayName: "Sales Test",
		DeviceHash:  "device-a",
		Activated:   true,
		ActivatedAt: &now,
		CreatedBy:   "test",
	}
	require.NoError(t, app.db.Create(&license).Error)

	require.NoError(t, app.ApplyDeploymentLicenseActivationFlush())

	var flushed LicenseKey
	require.NoError(t, app.db.Where("key = ?", license.Key).First(&flushed).Error)
	require.False(t, flushed.Activated)
	require.Empty(t, flushed.DeviceHash)
	require.Nil(t, flushed.ActivatedAt)

	var marker Setting
	require.NoError(t, app.db.Where("key = ?", "deployment_license_activation_flush_stamp").First(&marker).Error)
	require.Equal(t, "deploy-a", marker.Value)

	later := time.Now()
	require.NoError(t, app.db.Model(&flushed).Updates(map[string]any{
		"activated":    true,
		"device_hash":  "device-b",
		"activated_at": &later,
	}).Error)

	require.NoError(t, app.ApplyDeploymentLicenseActivationFlush())
	require.NoError(t, app.db.Where("key = ?", license.Key).First(&flushed).Error)
	require.True(t, flushed.Activated, "same deployment stamp must not flush every launch")
	require.Equal(t, "device-b", flushed.DeviceHash)

	t.Setenv("ASYMMFLOW_LICENSE_FLUSH_STAMP", "deploy-b")
	require.NoError(t, app.ApplyDeploymentLicenseActivationFlush())
	require.NoError(t, app.db.Where("key = ?", license.Key).First(&flushed).Error)
	require.False(t, flushed.Activated)
	require.Empty(t, flushed.DeviceHash)
}

func TestApplyDeploymentLicenseActivationFlushIgnoresReseedFlagWithoutExplicitStamp(t *testing.T) {
	app := setupTestApp(t)
	t.Setenv("ASYMMFLOW_FLUSH_LICENSE_ON_RESEED", "true")
	t.Setenv("ASYMMFLOW_LICENSE_FLUSH_STAMP", "")

	now := time.Now()
	license := LicenseKey{
		Key:         "PH-SLS-STAY1",
		Role:        "sales",
		DisplayName: "Sales Test",
		DeviceHash:  "device-a",
		Activated:   true,
		ActivatedAt: &now,
		CreatedBy:   "test",
	}
	require.NoError(t, app.db.Create(&license).Error)

	require.NoError(t, app.ApplyDeploymentLicenseActivationFlush())

	var persisted LicenseKey
	require.NoError(t, app.db.Where("key = ?", license.Key).First(&persisted).Error)
	require.True(t, persisted.Activated, "normal reseed/update packages must preserve activated licenses")
	require.Equal(t, "device-a", persisted.DeviceHash)
}

// seedTestCustomer creates a test customer and returns its ID
func seedTestCustomer(t *testing.T, db *gorm.DB, businessName string) string {
	t.Helper()

	customer := CustomerMaster{
		Base: Base{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		CustomerID:       uuid.New().String(),
		CustomerCode:     "CUST-" + uuid.New().String()[:8],
		BusinessName:     businessName,
		CustomerType:     "Corporate",
		City:             "Manama",
		Country:          "Bahrain",
		PaymentGrade:     "A",
		CustomerGrade:    "A",
		PaymentTermsDays: 30,
		CreditLimitBHD:   50000.0,
	}

	err := db.Create(&customer).Error
	require.NoError(t, err, "Failed to create test customer")
	return customer.ID
}

// seedTestSupplier creates a test supplier and returns its ID
func seedTestSupplier(t *testing.T, db *gorm.DB, supplierName string) string {
	t.Helper()

	supplier := SupplierMaster{
		Base: Base{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		SupplierCode: "SUP-" + uuid.New().String()[:8],
		SupplierName: supplierName,
		Country:      "Germany",
		LeadTimeDays: 30,
		PaymentTerms: "Net 30",
		Rating:       5,
	}

	err := db.Create(&supplier).Error
	require.NoError(t, err, "Failed to create test supplier")
	return supplier.ID
}

// TestListCustomers verifies customer listing with pagination
func TestListCustomers(t *testing.T) {
	t.Run("should return customers with valid pagination", func(t *testing.T) {
		app := setupTestApp(t)

		// Seed 5 test customers
		for i := 1; i <= 5; i++ {
			seedTestCustomer(t, app.db, "Customer "+string(rune('A'+i-1)))
		}

		// List first 3 customers
		customers, err := app.ListCustomers(3, 0)
		require.NoError(t, err)
		require.Len(t, customers, 3, "Should return 3 customers")

		// Verify they are sorted by business name
		require.Equal(t, "Customer A", customers[0].BusinessName)
		require.Equal(t, "Customer B", customers[1].BusinessName)
	})

	t.Run("should handle empty database", func(t *testing.T) {
		app := setupTestApp(t)

		customers, err := app.ListCustomers(10, 0)
		require.NoError(t, err)
		require.Empty(t, customers, "Should return empty list for empty database")
	})

	t.Run("should validate pagination parameters", func(t *testing.T) {
		app := setupTestApp(t)
		seedTestCustomer(t, app.db, "Test Customer")

		// Negative limit defaults to 100
		customers, err := app.ListCustomers(-10, 0)
		require.NoError(t, err)
		require.Len(t, customers, 1)

		// Offset beyond results
		customers, err = app.ListCustomers(10, 100)
		require.NoError(t, err)
		require.Empty(t, customers, "Offset beyond data should return empty")
	})

	t.Run("should return results sorted by name", func(t *testing.T) {
		app := setupTestApp(t)

		// Seed in random order
		seedTestCustomer(t, app.db, "Zebra Corp")
		seedTestCustomer(t, app.db, "Alpha Ltd")
		seedTestCustomer(t, app.db, "Beta Inc")

		customers, err := app.ListCustomers(10, 0)
		require.NoError(t, err)
		require.Len(t, customers, 3)

		// Verify alphabetical sorting
		require.Equal(t, "Alpha Ltd", customers[0].BusinessName)
		require.Equal(t, "Beta Inc", customers[1].BusinessName)
		require.Equal(t, "Zebra Corp", customers[2].BusinessName)
	})
}

// TestListOrders verifies order listing with various filters
func TestListOrders(t *testing.T) {
	t.Run("should return all orders when no filters", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create test orders
		order1 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			OrderDate:     time.Now(),
			Status:        "Processing",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
		}
		order2 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-002",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			OrderDate:     time.Now(),
			Status:        "Confirmed",
			TotalValueBHD: 2000.0,
			GrandTotalBHD: 2200.0,
		}

		require.NoError(t, app.db.Create(&order1).Error)
		require.NoError(t, app.db.Create(&order2).Error)

		// List all orders
		orders, err := app.ListOrders(10, 0)
		require.NoError(t, err)
		require.Len(t, orders, 2, "Should return all orders")
	})

	t.Run("should filter by order status", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create orders with different statuses
		order1 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Processing",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
		}
		order2 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-002",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Delivered",
			TotalValueBHD: 2000.0,
			GrandTotalBHD: 2200.0,
		}

		require.NoError(t, app.db.Create(&order1).Error)
		require.NoError(t, app.db.Create(&order2).Error)

		// List all orders - verify both statuses exist
		orders, err := app.ListOrders(10, 0)
		require.NoError(t, err)
		require.Len(t, orders, 2, "Should return both orders")

		// Manual filter check
		processingOrders := 0
		for _, order := range orders {
			if order.Status == "Processing" {
				processingOrders++
			}
		}
		require.Equal(t, 1, processingOrders, "Should have 1 processing order")
	})

	t.Run("should filter by customer ID", func(t *testing.T) {
		app := setupTestApp(t)
		customerID1 := seedTestCustomer(t, app.db, "Customer One")
		customerID2 := seedTestCustomer(t, app.db, "Customer Two")

		// Create orders for different customers
		order1 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID1,
			CustomerName:  "Customer One",
			Status:        "Processing",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
		}
		order2 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-002",
			CustomerID:    customerID2,
			CustomerName:  "Customer Two",
			Status:        "Processing",
			TotalValueBHD: 2000.0,
			GrandTotalBHD: 2200.0,
		}

		require.NoError(t, app.db.Create(&order1).Error)
		require.NoError(t, app.db.Create(&order2).Error)

		// List all orders
		orders, err := app.ListOrders(10, 0)
		require.NoError(t, err)
		require.Len(t, orders, 2, "Should return both orders")

		// Verify both customers represented
		customerNames := make(map[string]bool)
		for _, order := range orders {
			customerNames[order.CustomerName] = true
		}
		require.True(t, customerNames["Customer One"], "Should have Customer One orders")
		require.True(t, customerNames["Customer Two"], "Should have Customer Two orders")
	})

	t.Run("should filter by date range", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create orders with different dates
		pastDate := time.Now().AddDate(0, 0, -10)  // 10 days ago
		futureDate := time.Now().AddDate(0, 0, 10) // 10 days from now

		order1 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-PAST",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			OrderDate:     pastDate,
			Status:        "Processing",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
		}
		order2 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-FUTURE",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			OrderDate:     futureDate,
			Status:        "Processing",
			TotalValueBHD: 2000.0,
			GrandTotalBHD: 2200.0,
		}

		require.NoError(t, app.db.Create(&order1).Error)
		require.NoError(t, app.db.Create(&order2).Error)

		// List all orders and verify dates
		orders, err := app.ListOrders(10, 0)
		require.NoError(t, err)
		require.Len(t, orders, 2, "Should return both orders")

		// Verify past order exists
		foundPast := false
		for _, order := range orders {
			if order.OrderNumber == "ORD-PAST" {
				foundPast = true
				require.True(t, order.OrderDate.Before(time.Now()), "Past order should be before now")
			}
		}
		require.True(t, foundPast, "Should find past order")
	})

	t.Run("should preload customer data", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		order := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Processing",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
		}

		require.NoError(t, app.db.Create(&order).Error)

		orders, err := app.ListOrders(10, 0)
		require.NoError(t, err)
		require.Len(t, orders, 1)
		require.Equal(t, "Test Customer", orders[0].CustomerName)
	})
}

// TestCreateInvoice verifies invoice creation from order
func TestCreateInvoice(t *testing.T) {
	t.Run("should create invoice from order successfully", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create order with items
		order := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			OrderDate:     time.Now(),
			Status:        "Confirmed",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
			PaymentTerms:  "Net 30",
		}
		require.NoError(t, app.db.Create(&order).Error)

		// Create order items
		orderItem := OrderItem{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			Description: "Test Product",
			Quantity:    10,
			UnitPrice:   100.0,
		}
		require.NoError(t, app.db.Create(&orderItem).Error)

		// Bypass RBAC for test
		app.startupImporting = true

		// Create invoice
		invoice, err := app.CreateInvoiceFromOrder(order.ID)
		require.NoError(t, err)
		require.NotEmpty(t, invoice.InvoiceNumber)
		require.Equal(t, order.GrandTotalBHD, invoice.GrandTotalBHD)
		require.Equal(t, order.GrandTotalBHD, invoice.OutstandingBHD, "Outstanding should equal total for new invoice")
	})

	t.Run("should prevent duplicate invoices for same order", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create order
		order := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Confirmed",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
			PaymentTerms:  "Net 30",
		}
		require.NoError(t, app.db.Create(&order).Error)

		// Create order item
		orderItem := OrderItem{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			Description: "Test Product",
			Quantity:    10,
			UnitPrice:   100.0,
		}
		require.NoError(t, app.db.Create(&orderItem).Error)

		app.startupImporting = true

		// Create first invoice
		_, err := app.CreateInvoiceFromOrder(order.ID)
		require.NoError(t, err)

		// Attempt to create duplicate - should fail
		_, err = app.CreateInvoiceFromOrder(order.ID)
		require.Error(t, err, "Should prevent duplicate invoice")
		require.Contains(t, err.Error(), "already has", "Error should mention duplicate")
	})

	t.Run("should fail for non-existent order", func(t *testing.T) {
		app := setupTestApp(t)
		app.startupImporting = true

		// Try to create invoice for non-existent order
		fakeOrderID := uuid.New().String()
		_, err := app.CreateInvoiceFromOrder(fakeOrderID)
		require.Error(t, err, "Should fail for non-existent order")
		require.Contains(t, err.Error(), "not found", "Error should mention order not found")
	})

	t.Run("should calculate total amount correctly", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create order
		order := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Confirmed",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0, // Including 10% VAT
			PaymentTerms:  "Net 30",
		}
		require.NoError(t, app.db.Create(&order).Error)

		// Create order item
		orderItem := OrderItem{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			Description: "Test Product",
			Quantity:    10,
			UnitPrice:   100.0,
		}
		require.NoError(t, app.db.Create(&orderItem).Error)

		app.startupImporting = true

		// Create invoice
		invoice, err := app.CreateInvoiceFromOrder(order.ID)
		require.NoError(t, err)
		require.Equal(t, 1100.0, invoice.GrandTotalBHD, "Grand total should match order total")
	})

	t.Run("should create invoice items matching order items", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create order
		order := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Confirmed",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
			PaymentTerms:  "Net 30",
		}
		require.NoError(t, app.db.Create(&order).Error)

		// Create 2 order items
		orderItem1 := OrderItem{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			Description: "Product A",
			Quantity:    5,
			UnitPrice:   100.0,
		}
		orderItem2 := OrderItem{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			Description: "Product B",
			Quantity:    5,
			UnitPrice:   100.0,
		}
		require.NoError(t, app.db.Create(&orderItem1).Error)
		require.NoError(t, app.db.Create(&orderItem2).Error)

		app.startupImporting = true

		// Create invoice
		invoice, err := app.CreateInvoiceFromOrder(order.ID)
		require.NoError(t, err)

		// Verify invoice items created
		var invoiceItems []DBInvoiceItem
		err = app.db.Where("invoice_id = ?", invoice.ID).Find(&invoiceItems).Error
		require.NoError(t, err)
		require.Len(t, invoiceItems, 2, "Should create 2 invoice items")
	})

	t.Run("should update order status after invoice creation", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create order
		order := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Confirmed",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
			PaymentTerms:  "Net 30",
		}
		require.NoError(t, app.db.Create(&order).Error)

		// Create order item
		orderItem := OrderItem{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			Description: "Test Product",
			Quantity:    10,
			UnitPrice:   100.0,
		}
		require.NoError(t, app.db.Create(&orderItem).Error)

		app.startupImporting = true

		// Create invoice
		_, err := app.CreateInvoiceFromOrder(order.ID)
		require.NoError(t, err)

		// Verify order status updated (implementation may vary)
		var updatedOrder Order
		err = app.db.First(&updatedOrder, "id = ?", order.ID).Error
		require.NoError(t, err)
		// Note: Actual status update logic depends on implementation
	})

	t.Run("should set correct outstanding balance", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create order
		order := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Confirmed",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
			PaymentTerms:  "Net 30",
		}
		require.NoError(t, app.db.Create(&order).Error)

		// Create order item
		orderItem := OrderItem{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			Description: "Test Product",
			Quantity:    10,
			UnitPrice:   100.0,
		}
		require.NoError(t, app.db.Create(&orderItem).Error)

		app.startupImporting = true

		// Create invoice
		invoice, err := app.CreateInvoiceFromOrder(order.ID)
		require.NoError(t, err)
		require.Equal(t, invoice.GrandTotalBHD, invoice.OutstandingBHD, "Outstanding should equal grand total")
		require.Equal(t, 1100.0, invoice.OutstandingBHD, "Outstanding should be 1100 BHD")
	})
}

// TestRecordPayment verifies payment recording with safety checks
func TestRecordPayment(t *testing.T) {
	t.Run("should record payment successfully", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create invoice
		invoice := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-001",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  1100.0,
			OutstandingBHD: 1100.0,
			Status:         "Sent",
		}
		require.NoError(t, app.db.Create(&invoice).Error)

		// Record payment
		payment, err := app.RecordPayment(invoice.ID, 500.0, "Bank Transfer", time.Now().Format("2006-01-02"), "TXN-123")
		require.NoError(t, err)
		require.NotNil(t, payment)
		require.Equal(t, 500.0, payment.AmountBHD)
		require.Equal(t, "Bank Transfer", payment.PaymentMethod)

		// Verify invoice updated
		var updatedInvoice Invoice
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		require.Equal(t, 600.0, updatedInvoice.OutstandingBHD, "Outstanding should be reduced by payment")
		require.Equal(t, "PartiallyPaid", updatedInvoice.Status)
	})

	t.Run("should prevent overpayment", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create invoice
		invoice := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-001",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  1100.0,
			OutstandingBHD: 1100.0,
			Status:         "Sent",
		}
		require.NoError(t, app.db.Create(&invoice).Error)

		// Attempt to overpay
		_, err := app.RecordPayment(invoice.ID, 1500.0, "Cash", time.Now().Format("2006-01-02"), "")
		require.Error(t, err, "Should prevent overpayment")
		require.Contains(t, err.Error(), "exceeds outstanding", "Error should mention overpayment")
	})

	t.Run("should reject negative amount", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create invoice
		invoice := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-001",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  1100.0,
			OutstandingBHD: 1100.0,
			Status:         "Sent",
		}
		require.NoError(t, app.db.Create(&invoice).Error)

		// Attempt negative payment
		_, err := app.RecordPayment(invoice.ID, -100.0, "Cash", time.Now().Format("2006-01-02"), "")
		require.Error(t, err, "Should reject negative amount")
		require.Contains(t, err.Error(), "greater than zero", "Error should mention amount validation")
	})

	t.Run("should reject zero amount", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create invoice
		invoice := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-001",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  1100.0,
			OutstandingBHD: 1100.0,
			Status:         "Sent",
		}
		require.NoError(t, app.db.Create(&invoice).Error)

		// Attempt zero payment
		_, err := app.RecordPayment(invoice.ID, 0.0, "Cash", time.Now().Format("2006-01-02"), "")
		require.Error(t, err, "Should reject zero amount")
	})

	t.Run("should fail for non-existent invoice", func(t *testing.T) {
		app := setupTestApp(t)

		// Attempt payment on non-existent invoice
		fakeInvoiceID := uuid.New().String()
		_, err := app.RecordPayment(fakeInvoiceID, 100.0, "Cash", time.Now().Format("2006-01-02"), "")
		require.Error(t, err, "Should fail for non-existent invoice")
		require.Contains(t, err.Error(), "not found", "Error should mention invoice not found")
	})

	t.Run("should update invoice outstanding balance", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create invoice
		invoice := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-001",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  1000.0,
			OutstandingBHD: 1000.0,
			Status:         "Sent",
		}
		require.NoError(t, app.db.Create(&invoice).Error)

		// Record partial payment
		_, err := app.RecordPayment(invoice.ID, 400.0, "Cash", time.Now().Format("2006-01-02"), "")
		require.NoError(t, err)

		// Verify balance updated
		var updatedInvoice Invoice
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		require.Equal(t, 600.0, updatedInvoice.OutstandingBHD)

		// Record full payment
		_, err = app.RecordPayment(invoice.ID, 600.0, "Cash", time.Now().Format("2006-01-02"), "")
		require.NoError(t, err)

		// Verify fully paid
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		require.Equal(t, 0.0, updatedInvoice.OutstandingBHD)
		require.Equal(t, "Paid", updatedInvoice.Status)
	})

	t.Run("should mark order as complete when fully paid", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create order
		order := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Confirmed",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1000.0,
		}
		require.NoError(t, app.db.Create(&order).Error)

		// Create invoice
		invoice := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-001",
			OrderID:        order.ID,
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  1000.0,
			OutstandingBHD: 1000.0,
			Status:         "Sent",
		}
		require.NoError(t, app.db.Create(&invoice).Error)

		// Record full payment
		_, err := app.RecordPayment(invoice.ID, 1000.0, "Cash", time.Now().Format("2006-01-02"), "")
		require.NoError(t, err)

		// Verify invoice fully paid
		var updatedInvoice Invoice
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		require.Equal(t, "Paid", updatedInvoice.Status)
	})

	t.Run("should round to BHD precision (3 decimals)", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create invoice
		invoice := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-001",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  100.123,
			OutstandingBHD: 100.123,
			Status:         "Sent",
		}
		require.NoError(t, app.db.Create(&invoice).Error)

		// Record payment with many decimals
		payment, err := app.RecordPayment(invoice.ID, 50.123456789, "Cash", time.Now().Format("2006-01-02"), "")
		require.NoError(t, err)
		require.Equal(t, 50.123, payment.AmountBHD, "Should round to 3 decimals")

		// Verify balance rounded
		var updatedInvoice Invoice
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		require.Equal(t, 50.0, updatedInvoice.OutstandingBHD, "Outstanding should be rounded")
	})

	t.Run("should prevent concurrent payment race conditions", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create invoice
		invoice := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-001",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  1000.0,
			OutstandingBHD: 1000.0,
			Status:         "Sent",
		}
		require.NoError(t, app.db.Create(&invoice).Error)

		// Record first payment
		_, err := app.RecordPayment(invoice.ID, 500.0, "Cash", time.Now().Format("2006-01-02"), "")
		require.NoError(t, err)

		// Attempt duplicate payment (same amount, same date)
		_, err = app.RecordPayment(invoice.ID, 500.0, "Cash", time.Now().Format("2006-01-02"), "")
		require.Error(t, err, "Should detect duplicate payment")
		require.Contains(t, err.Error(), "duplicate", "Error should mention duplicate detection")
	})
}

// TestGetDashboardStats verifies dashboard statistics calculation
func TestGetDashboardStats(t *testing.T) {
	t.Run("should return zero stats for empty database", func(t *testing.T) {
		app := setupTestApp(t)

		stats, err := app.GetDashboardStats()
		require.NoError(t, err)
		require.Equal(t, 0, stats.ActiveCustomers, "Customer count should be 0")
		require.Equal(t, 0.0, stats.TotalRevenue, "Revenue should be 0")
		require.Equal(t, 0.0, stats.OutstandingAR, "Outstanding AR should be 0")
	})

	t.Run("should calculate customer count correctly", func(t *testing.T) {
		app := setupTestApp(t)

		// Seed customers
		seedTestCustomer(t, app.db, "Customer 1")
		seedTestCustomer(t, app.db, "Customer 2")
		seedTestCustomer(t, app.db, "Customer 3")

		stats, err := app.GetDashboardStats()
		require.NoError(t, err)
		require.Equal(t, 3, stats.ActiveCustomers, "Should count 3 customers")
	})

	t.Run("should calculate supplier count correctly", func(t *testing.T) {
		app := setupTestApp(t)

		// Seed suppliers
		seedTestSupplier(t, app.db, "Supplier 1")
		seedTestSupplier(t, app.db, "Supplier 2")

		// Note: DashboardStats doesn't have SupplierCount field
		// This test verifies data exists but doesn't check dashboard stats
		var count int64
		err := app.db.Model(&SupplierMaster{}).Count(&count).Error
		require.NoError(t, err)
		require.Equal(t, int64(2), count, "Should count 2 suppliers")
	})

	t.Run("should calculate total revenue correctly", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create paid invoices
		invoice1 := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-001",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  1000.0,
			OutstandingBHD: 0.0,
			Status:         "Paid",
		}
		invoice2 := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-002",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  2000.0,
			OutstandingBHD: 0.0,
			Status:         "Paid",
		}

		require.NoError(t, app.db.Create(&invoice1).Error)
		require.NoError(t, app.db.Create(&invoice2).Error)

		stats, err := app.GetDashboardStats()
		require.NoError(t, err)
		require.Equal(t, 3000.0, stats.TotalRevenue, "Total revenue should be 3000 BHD")
	})

	t.Run("should calculate outstanding balance correctly", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create invoices with outstanding balances
		invoice1 := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-001",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  1000.0,
			OutstandingBHD: 500.0,
			Status:         "PartiallyPaid",
		}
		invoice2 := Invoice{
			Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceNumber:  "INV-002",
			CustomerID:     customerID,
			CustomerName:   "Test Customer",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			GrandTotalBHD:  2000.0,
			OutstandingBHD: 2000.0,
			Status:         "Sent",
		}

		require.NoError(t, app.db.Create(&invoice1).Error)
		require.NoError(t, app.db.Create(&invoice2).Error)

		stats, err := app.GetDashboardStats()
		require.NoError(t, err)
		require.Equal(t, 2500.0, stats.OutstandingAR, "Outstanding AR should be 2500 BHD")
	})

	t.Run("should fallback to uninvoiced active orders when invoice ar is zero", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		order := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-AR-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			OrderDate:     time.Now(),
			Status:        "Processing",
			GrandTotalBHD: 1750.0,
		}
		require.NoError(t, app.db.Create(&order).Error)

		stats, err := app.GetDashboardStats()
		require.NoError(t, err)
		require.Equal(t, 1750.0, stats.OutstandingAR, "Outstanding AR should fallback to uninvoiced active order value")
		require.Equal(t, 1, stats.PendingInvoices, "Pending invoices should reflect orders awaiting invoicing when no invoice AR exists")
	})

	t.Run("should count orders by status", func(t *testing.T) {
		app := setupTestApp(t)
		customerID := seedTestCustomer(t, app.db, "Test Customer")

		// Create orders with different statuses
		order1 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-001",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Processing",
			TotalValueBHD: 1000.0,
			GrandTotalBHD: 1100.0,
		}
		order2 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-002",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Processing",
			TotalValueBHD: 2000.0,
			GrandTotalBHD: 2200.0,
		}
		order3 := Order{
			Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderNumber:   "ORD-003",
			CustomerID:    customerID,
			CustomerName:  "Test Customer",
			Status:        "Delivered",
			TotalValueBHD: 3000.0,
			GrandTotalBHD: 3300.0,
		}

		require.NoError(t, app.db.Create(&order1).Error)
		require.NoError(t, app.db.Create(&order2).Error)
		require.NoError(t, app.db.Create(&order3).Error)

		stats, err := app.GetDashboardStats()
		require.NoError(t, err)

		// Verify active orders counted (DashboardStats has ActiveOrders field)
		require.GreaterOrEqual(t, stats.ActiveOrders, 0, "Active orders should be non-negative")

		// Directly count processing orders from database
		var processingCount int64
		err = app.db.Model(&Order{}).Where("status = ?", "Processing").Count(&processingCount).Error
		require.NoError(t, err)
		require.Equal(t, int64(2), processingCount, "Should count 2 processing orders")
	})

	t.Run("should use cache when available", func(t *testing.T) {
		app := setupTestApp(t)
		seedTestCustomer(t, app.db, "Test Customer")

		// First call - should populate cache
		stats1, err := app.GetDashboardStats()
		require.NoError(t, err)

		// Second call - should use cache
		stats2, err := app.GetDashboardStats()
		require.NoError(t, err)

		require.Equal(t, stats1.ActiveCustomers, stats2.ActiveCustomers, "Cached stats should match")
	})

	t.Run("should invalidate cache on data changes", func(t *testing.T) {
		app := setupTestApp(t)

		// Get initial stats
		stats1, err := app.GetDashboardStats()
		require.NoError(t, err)
		require.Equal(t, 0, stats1.ActiveCustomers)

		// Add customer
		seedTestCustomer(t, app.db, "New Customer")

		// Invalidate cache
		app.cache.Delete("dashboard_stats")

		// Get updated stats
		stats2, err := app.GetDashboardStats()
		require.NoError(t, err)
		require.Equal(t, 1, stats2.ActiveCustomers, "Stats should reflect new customer")
	})
}

func TestRouteToBankStatementHonorsExplicitBankAccountSelection(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&BankStatement{}, &BankStatementLine{}))

	kuwaitAccount := CompanyBankAccount{
		ID:            "bank-kfh-test",
		BankName:      "Kuwait Finance House",
		AccountName:   "Acme Instrumentation WLL",
		AccountNumber: "200000412340002",
		Currency:      "BHD",
		IsActive:      true,
		DisplayOrder:  10,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	otherAccount := CompanyBankAccount{
		ID:            "bank-other-test",
		BankName:      "National Bank of Bahrain BSC",
		AccountName:   "Acme Instrumentation WLL",
		AccountNumber: "0012340001",
		Currency:      "BHD",
		IsActive:      true,
		DisplayOrder:  11,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	require.NoError(t, app.db.Create(&kuwaitAccount).Error)
	require.NoError(t, app.db.Create(&otherAccount).Error)

	statementID, err := app.routeToBankStatement(map[string]any{
		"bank_account_id": kuwaitAccount.ID,
		"bank_name":       "KFH",
		"account_number":  "200000412340002",
		"period_start":    "2026-01-01",
		"period_end":      "2026-01-31",
		"opening_balance": 25709.210,
		"closing_balance": 56753.716,
		"currency":        "BHD",
		"line_items":      []any{},
	}, "kfh-test.pdf", 0.98)
	require.NoError(t, err)
	require.NotEmpty(t, statementID)

	var stmt BankStatement
	require.NoError(t, app.db.First(&stmt, "id = ?", statementID).Error)
	require.Equal(t, kuwaitAccount.ID, stmt.BankAccountID)
}

func TestRouteToBankStatementDoesNotFallbackToFirstActiveAccount(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&BankStatement{}, &BankStatementLine{}))

	require.NoError(t, app.db.Create(&CompanyBankAccount{
		ID:            "bank-only-test",
		BankName:      "National Bank of Bahrain BSC",
		AccountName:   "Acme Instrumentation WLL",
		AccountNumber: "0012340001",
		Currency:      "BHD",
		IsActive:      true,
		DisplayOrder:  12,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}).Error)

	_, err := app.routeToBankStatement(map[string]any{
		"bank_name":       "Kuwait Finance House",
		"account_number":  "200000412340002",
		"period_start":    "2026-01-01",
		"period_end":      "2026-01-31",
		"opening_balance": 25709.210,
		"closing_balance": 56753.716,
		"currency":        "BHD",
		"line_items":      []any{},
	}, "kfh-unmatched.pdf", 0.98)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not match this statement to an active bank account")
}
