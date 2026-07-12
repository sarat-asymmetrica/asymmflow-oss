package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func migrateWorkflowRegressionTables(t *testing.T, app *App) {
	t.Helper()
	require.NoError(t, app.db.AutoMigrate(
		&Role{},
		&User{},
		&LicenseKey{},
		&CustomerMaster{},
		&EntityNote{},
		&RFQData{},
		&CostingSheetData{},
		&Opportunity{},
		&Offer{},
		&OfferItem{},
		&OfferNote{},
		&Order{},
		&OrderItem{},
		&Invoice{},
		&DBInvoiceItem{},
		&Payment{},
		&SupplierMaster{},
		&SupplierInvoice{},
		&SupplierInvoiceItem{},
		&SupplierPayment{},
		&BankStatement{},
		&BankStatementLine{},
		&BankReconciliationAuditLog{},
		&Employee{},
		&TaskItem{},
		&TaskComment{},
		&Notification{},
	))
}

func TestCreateRFQWithReferenceUsesUserReference(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	rfq, err := app.CreateRFQWithReference("National Petroleum Co.", "Analyzer skid", "EH-REF-2026-001", 1250, "Urgent enquiry")
	require.NoError(t, err)
	require.Equal(t, "EH-REF-2026-001", rfq.RFQNumber)
	require.Equal(t, "EH-REF-2026-001", rfq.RFQRef)

	var stored RFQData
	require.NoError(t, app.db.First(&stored, "id = ?", rfq.ID).Error)
	require.Equal(t, "EH-REF-2026-001", stored.RFQNumber)
	require.Equal(t, "EH-REF-2026-001", stored.RFQRef)
}

func TestSaveCostingAsOfferPropagatesReferenceToLinkedOpportunity(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	opportunity := Opportunity{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		FolderNumber: "2-26",
		CustomerName: "National Petroleum Co.",
		Title:        "Analyzer skid",
		Stage:        "Qualified",
	}
	require.NoError(t, app.db.Create(&opportunity).Error)

	costingDate := time.Now().Format("2006-01-02")
	offer, err := app.SaveCostingAsOffer(CostingExportData{
		Date:                costingDate,
		PreparedBy:          "Jamie",
		CustomerName:        "National Petroleum Co.",
		RfqReference:        "EH-REF-2026-777",
		FolderNumber:        "2-26",
		CostingId:           "CS-EH-REF-2026-777",
		Subject:             "Analyzer skid",
		ProjectName:         "Analyzer skid",
		OpportunityRecordID: opportunity.ID,
		QuoteType:           "Quotation",
		VatRate:             10,
		Subtotal:            100,
		NetAmount:           100,
		VAT:                 10,
		GrandTotal:          110,
		TotalCost:           80,
		Profit:              30,
		LineItems: []CostingExportLineItem{
			{
				SlNo:           1,
				Equipment:      "Analyzer",
				Quantity:       1,
				Currency:       "BHD",
				SuggestedPrice: 100,
				TotalPrice:     100,
				TotalCost:      80,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "EH-REF-2026-777", offer.CustomerReference)

	var storedOffer Offer
	require.NoError(t, app.db.First(&storedOffer, "id = ?", offer.ID).Error)
	require.Equal(t, "EH-REF-2026-777", storedOffer.CustomerReference)

	var linked Opportunity
	require.NoError(t, app.db.First(&linked, "id = ?", opportunity.ID).Error)
	require.Equal(t, offer.ID, linked.OfferID)
	require.Equal(t, "Quoted", linked.Stage)
	require.Equal(t, "EH-REF-2026-777", linked.EHRef)
	require.Equal(t, "Analyzer skid", linked.Title)
}

func TestSaveCostingAsOfferUsesFolderNumberAsOfferNumber(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	costingDate := time.Now().Format("2006-01-02")
	offer, err := app.SaveCostingAsOffer(CostingExportData{
		Date:         costingDate,
		PreparedBy:   "Jamie",
		CustomerName: "National Petroleum Co.",
		RfqReference: "EH-REF-2026-888",
		FolderNumber: "26-26",
		CostingId:    "CS-EH-REF-2026-888",
		Subject:      "Analyzer skid",
		QuoteType:    "Quotation",
		VatRate:      10,
		Subtotal:     100,
		NetAmount:    100,
		VAT:          10,
		GrandTotal:   110,
		TotalCost:    80,
		Profit:       30,
		LineItems: []CostingExportLineItem{
			{
				SlNo:           1,
				Equipment:      "Analyzer",
				Quantity:       1,
				Currency:       "BHD",
				SuggestedPrice: 100,
				TotalPrice:     100,
				TotalCost:      80,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "26-26", offer.OfferNumber)
	require.Equal(t, 0, offer.RevisionNumber)
}

func TestUpdateOfferFullSavesEditableOfferNumberAndRevisionOne(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	now := time.Now()
	offer := Offer{
		Base:           Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		OfferNumber:    "TEMP-26",
		RevisionNumber: 0,
		CustomerName:   "National Petroleum Co.",
		QuotationDate:  now.AddDate(0, 0, -1),
		ValidityDate:   now.AddDate(0, 0, 30),
		Stage:          "Quoted",
		TotalValueBHD:  100,
	}
	require.NoError(t, app.db.Create(&offer).Error)

	updated, err := app.UpdateOfferFull(offer.ID, OfferUpdateData{
		OfferNumber:   "26-26",
		CustomerName:  offer.CustomerName,
		QuotationDate: offer.QuotationDate.Format("2006-01-02"),
		ValidityDate:  offer.ValidityDate.Format("2006-01-02"),
		PaymentTerms:  "Net 30",
		Items:         []OfferUpdateItem{},
	})
	require.NoError(t, err)
	require.Equal(t, "26-26", updated.OfferNumber)
	require.Equal(t, 1, updated.RevisionNumber)
}

func TestGetOpportunityLineItems_FallsBackToProductDetails(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	seedItems := []OfferItem{
		{
			LineNumber:    1,
			Description:   "Flow transmitter",
			Quantity:      2,
			UnitPrice:     540.758,
			TotalPrice:    1081.516,
			Equipment:     "Coriolis Flow",
			ProductCode:   "CORIOLIS-F",
			Specification: "Coriolis flow meter",
		},
	}

	opportunity := Opportunity{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		FolderNumber:   "2026-501",
		CustomerName:   "National Petroleum Co.",
		Title:          "Fallback item coverage",
		Stage:          "Qualified",
		ProductDetails: serializeOpportunityProductDetailsFromOfferItems(seedItems),
	}
	require.NoError(t, app.db.Create(&opportunity).Error)

	items, err := app.GetOpportunityLineItems(opportunity.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Flow transmitter", items[0].Description)
	require.Equal(t, 2.0, items[0].Quantity)
	require.Equal(t, 540.758, items[0].UnitPrice)
	require.Equal(t, 1081.516, items[0].TotalPrice)
}

func TestNormalizeOpportunityLineItems_DoesNotStoreCurrencyAsUnit(t *testing.T) {
	rawItems := []any{
		map[string]any{
			"description": "Bray 12 inch",
			"quantity":    8,
			"unit":        "EUR",
			"unit_price":  2737,
			"total_price": 21895,
		},
	}

	normalized, count := normalizeOpportunityLineItemsJSON(rawItems)
	require.Equal(t, 1, count)

	var rows []map[string]any
	require.NoError(t, json.Unmarshal([]byte(normalized), &rows))
	require.Len(t, rows, 1)
	require.NotContains(t, rows[0], "unit")
	require.Equal(t, "EUR", rows[0]["currency"])
	require.Equal(t, float64(8), rows[0]["quantity"])
}

func TestNormalizeOpportunityLineItems_ParsesCurrencyPrefixedMoney(t *testing.T) {
	rawItems := []any{
		map[string]any{
			"description": "Bray 20 inch",
			"quantity":    "5",
			"unit_price":  "EUR 3,176.00",
			"total_price": "EUR 15,880.00",
		},
	}

	normalized, count := normalizeOpportunityLineItemsJSON(rawItems)
	require.Equal(t, 1, count)

	var rows []map[string]any
	require.NoError(t, json.Unmarshal([]byte(normalized), &rows))
	require.Len(t, rows, 1)
	require.Equal(t, float64(3176), rows[0]["unit_price"])
	require.Equal(t, float64(15880), rows[0]["total_price"])
}

func TestNormalizeOpportunityLineItems_RejectsPriceOnlyRows(t *testing.T) {
	rawItems := []any{
		map[string]any{
			"unit_price":  2737,
			"total_price": 21895,
		},
	}

	normalized, count := normalizeOpportunityLineItemsJSON(rawItems)
	require.Equal(t, 0, count)
	require.Empty(t, normalized)
}

func TestGroundedCapabilitiesFastPath_IncludesDatabaseCoverage(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	reply, handled := app.tryGroundedCapabilitiesFastPath(Intent{}, "hey buddy! Can you tell us what you can do?", true)
	require.True(t, handled)
	require.Contains(t, reply, "Database tables indexed for guarded querying")
	require.Contains(t, reply, "controlled ERP access paths")
}

func TestBuildFullContext_IncludesCrossModuleDatabaseAndWorkAccess(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	adminRole := Role{
		Base:        Base{ID: uuid.New().String()},
		Name:        "admin",
		DisplayName: "Admin",
		Permissions: `["*","finance:view","intelligence:chat","tasks:view","notifications:view"]`,
		IsActive:    true,
	}
	require.NoError(t, app.db.Create(&adminRole).Error)
	app.currentUser = &User{
		Base:     Base{ID: "butler-admin"},
		Username: "butler-admin",
		RoleName: "admin",
		RoleID:   adminRole.ID,
		Role:     adminRole,
	}
	app.currentUserID = "butler-admin"

	customer := CustomerMaster{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerID:    "C-National Petroleum Co.-001",
		BusinessName:  "National Petroleum Co.",
		CustomerGrade: "A",
		PaymentGrade:  "A",
		Industry:      "Energy",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	require.NoError(t, app.db.Create(&EntityNote{
		Base:       Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EntityType: "customer",
		EntityID:   customer.ID,
		NoteType:   "general",
		Content:    "National Petroleum Co. prefers calibration packs and consolidated follow-up notes.",
	}).Error)

	offer := Offer{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferNumber:   "OFF-CTX-001",
		CustomerID:    customer.CustomerID,
		CustomerName:  customer.BusinessName,
		QuotationDate: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
		ValidityDate:  time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		Stage:         "Quoted",
		TotalValueBHD: 1500,
		Division:      "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&offer).Error)
	require.NoError(t, app.db.Create(&OfferItem{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferID:       offer.ID,
		LineNumber:    1,
		Description:   "Calibration service visit",
		Equipment:     "Flowmeter Calibration",
		Model:         "CAL-PACK",
		Quantity:      2,
		UnitPrice:     750,
		TotalPrice:    1500,
		MarginPercent: 32,
	}).Error)
	require.NoError(t, app.db.Create(&OfferNote{
		Base:     Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferID:  offer.ID,
		NoteDate: time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC),
		Content:  "Customer requested split pricing and service mobilization note.",
	}).Error)

	statement := BankStatement{
		Base:            Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		BankAccountID:   "bank-1",
		StatementNumber: "BS-CTX-001",
		StatementDate:   time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
		PeriodStart:     time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:       time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
		Status:          "Imported",
		Notes:           "National Petroleum Co. statement contains Jamie follow-up marker for unmatched service receipts.",
		Division:        "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&statement).Error)

	jamie := Employee{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EmployeeCode:     "EMP-002",
		FullName:         "Jamie",
		PreferredName:    "Jamie",
		Department:       "Operations",
		JobTitle:         "Coordinator",
		EmploymentStatus: "active",
		IsActive:         true,
		Notes:            "Handles service coordination and follow-up tasks.",
	}
	require.NoError(t, app.db.Create(&jamie).Error)

	task := TaskItem{
		Base:               Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Title:              "Review National Petroleum Co. calibration follow-up",
		Description:        "Check statement note and confirm the service offer line items.",
		Status:             "open",
		Priority:           "high",
		TaskType:           "service",
		AssigneeEmployeeID: &jamie.ID,
		CreatorEmployeeID:  jamie.ID,
		CustomerID:         &customer.ID,
	}
	require.NoError(t, app.db.Create(&task).Error)
	require.NoError(t, app.db.Create(&TaskComment{
		Base:       Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		TaskID:     task.ID,
		EmployeeID: jamie.ID,
		Body:       "Jamie noted that the bank statement and offer notes need a single client response.",
	}).Error)

	ctx := app.buildFullContext(Intent{
		RawQuery:   "How many tasks are assigned to Jamie and what notes or bank statements do we have for National Petroleum Co.?",
		Domain:     "general",
		EntityName: "National Petroleum Co.",
		PersonName: "Jamie",
	})

	workData, ok := ctx["work_data"].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, workData["tasks"])

	databaseAccess, ok := ctx["database_access"].(map[string]any)
	require.True(t, ok)

	databaseCatalog, ok := ctx["database_catalog"].(map[string]any)
	require.True(t, ok)
	require.NotZero(t, databaseCatalog["total_tables"])
	require.NotEmpty(t, databaseCatalog["tables"])

	notesLedger, ok := databaseAccess["notes_ledger"].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, notesLedger["bank_statement_notes"])
	require.NotEmpty(t, notesLedger["offer_notes"])

	commercialRecords, ok := databaseAccess["commercial_records"].(map[string]any)
	require.True(t, ok)
	offers, ok := commercialRecords["offers"].([]map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, offers)
	require.NotEmpty(t, offers[0]["line_items"])
}

func TestGetOpportunityLineItems_FallsBackWhenLinkedOfferHasNoItems(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	offer := Offer{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferNumber:   "PHO-TEST-001",
		CustomerName:  "National Petroleum Co.",
		QuotationDate: time.Now(),
		ValidityDate:  time.Now().AddDate(0, 0, 30),
		Stage:         "Quoted",
	}
	require.NoError(t, app.db.Create(&offer).Error)

	seedItems := []OfferItem{
		{
			LineNumber:  1,
			Description: "Pressure switch",
			Quantity:    1,
			UnitPrice:   321.123,
			TotalPrice:  321.123,
		},
	}

	opportunity := Opportunity{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		FolderNumber:   "2026-502",
		OfferID:        offer.ID,
		CustomerName:   "National Petroleum Co.",
		Title:          "Linked offer fallback",
		Stage:          "Quoted",
		ProductDetails: serializeOpportunityProductDetailsFromOfferItems(seedItems),
	}
	require.NoError(t, app.db.Create(&opportunity).Error)

	items, err := app.GetOpportunityLineItems(opportunity.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Pressure switch", items[0].Description)
	require.Equal(t, 321.123, items[0].UnitPrice)
}

func TestUpdateOpportunityDetails_PreservesOwnerNotesForNonManagement(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	app.currentUser = &User{
		Base:     Base{ID: "sales-user"},
		Username: "riley",
		RoleName: "sales",
		Role: Role{
			Name:        "sales",
			DisplayName: "Sales",
			Permissions: `["offers:edit"]`,
		},
	}
	app.currentUserID = "sales-user"

	opportunity := Opportunity{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		FolderNumber: "2026-503",
		CustomerName: "Vantage PG",
		Title:        "Editable note coverage",
		Comment:      "Original note",
		OwnerNotes:   "Management only",
		Stage:        "Qualified",
	}
	require.NoError(t, app.db.Create(&opportunity).Error)

	updated, err := app.UpdateOpportunityDetails(opportunity.ID, "  refreshed commercial note  ", "should not replace")
	require.NoError(t, err)
	require.Equal(t, "refreshed commercial note", updated.Comment)
	require.Equal(t, "Management only", updated.OwnerNotes)

	var stored Opportunity
	require.NoError(t, app.db.First(&stored, "id = ?", opportunity.ID).Error)
	require.Equal(t, "refreshed commercial note", stored.Comment)
	require.Equal(t, "Management only", stored.OwnerNotes)
}

func TestUpdateRFQNotes_AllowsClearingNotes(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	rfq := RFQData{
		RFQNumber: "RFQ-2026-777",
		Client:    "National Petroleum Co.",
		Project:   "Clear note coverage",
		Value:     2500,
		Notes:     "legacy note",
		Status:    "New",
		Stage:     "RFQ Received",
	}
	require.NoError(t, app.db.Create(&rfq).Error)

	updated, err := app.UpdateRFQNotes(rfq.ID, "   ")
	require.NoError(t, err)
	require.Equal(t, "", updated.Notes)

	var stored RFQData
	require.NoError(t, app.db.First(&stored, "id = ?", rfq.ID).Error)
	require.Equal(t, "", stored.Notes)
}

func TestSeedDefaultRoles_SalesCanCreatePurchaseOrders(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	require.NoError(t, app.SeedDefaultRoles())
	require.True(t, app.CheckPermissionByRole("sales", "po:create"))
	require.True(t, app.CheckPermissionByRole("sales", "suppliers:create"))
	require.True(t, app.CheckPermissionByRole("sales", "orders:update"))
}

func TestGetBusinessYearSummary_CapturesCommercialCoverage(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	customerID := seedTestCustomer(t, app.db, "National Petroleum Co.")
	baseTime := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)

	invoice := Invoice{
		Base:           Base{ID: uuid.New().String(), CreatedAt: baseTime, UpdatedAt: baseTime},
		InvoiceNumber:  "INV-2026-001",
		CustomerID:     customerID,
		CustomerName:   "National Petroleum Co.",
		InvoiceDate:    baseTime,
		Status:         "Sent",
		GrandTotalBHD:  505.120,
		OutstandingBHD: 505.120,
	}
	require.NoError(t, app.db.Create(&invoice).Error)

	order := Order{
		Base:          Base{ID: uuid.New().String(), CreatedAt: baseTime, UpdatedAt: baseTime},
		OrderNumber:   "ORD-2026-001",
		CustomerID:    customerID,
		CustomerName:  "National Petroleum Co.",
		OrderDate:     baseTime,
		Status:        "Confirmed",
		GrandTotalBHD: 111668.459,
	}
	require.NoError(t, app.db.Create(&order).Error)

	offer := Offer{
		Base:          Base{ID: uuid.New().String(), CreatedAt: baseTime, UpdatedAt: baseTime},
		OfferNumber:   "OFF-2026-001",
		CustomerID:    customerID,
		CustomerName:  "National Petroleum Co.",
		QuotationDate: baseTime,
		ValidityDate:  baseTime.AddDate(0, 0, 30),
		TotalValueBHD: 396609.242,
		Stage:         "Quoted",
	}
	require.NoError(t, app.db.Create(&offer).Error)

	opportunity := Opportunity{
		Base:         Base{ID: uuid.New().String(), CreatedAt: baseTime, UpdatedAt: baseTime},
		FolderNumber: "2026-001",
		CustomerID:   customerID,
		CustomerName: "National Petroleum Co.",
		Title:        "2026 coverage check",
		Stage:        "Qualified",
		RevenueBHD:   1711321.860,
		Year:         2026,
		OfferDate:    baseTime,
	}
	require.NoError(t, app.db.Create(&opportunity).Error)

	summary := app.getBusinessYearSummary("do you have access to 2026 data from this?")
	require.Equal(t, "available", summary["status"])
	require.Equal(t, 2026, summary["year"])

	invoices := summary["invoices"].(map[string]any)
	require.Equal(t, 1, invoices["count"])
	require.Equal(t, 505.12, invoices["total_bhd"])
	require.Equal(t, "2026-03-31", invoices["latest_date"])

	orders := summary["orders"].(map[string]any)
	require.Equal(t, 1, orders["count"])
	require.Equal(t, 111668.459, orders["total_bhd"])

	offers := summary["offers"].(map[string]any)
	require.Equal(t, 1, offers["count"])
	require.Equal(t, 396609.242, offers["total_bhd"])

	opps := summary["opportunities"].(map[string]any)
	require.Equal(t, 1, opps["count"])
	require.Equal(t, 1, opps["open_count"])
	require.Equal(t, 1711321.86, opps["total_value_bhd"])
	require.Equal(t, 1711321.86, opps["open_pipeline_bhd"])
	require.Equal(t, "2026-03-31", opps["latest_activity_date"])

	require.ElementsMatch(t, []string{"invoices", "orders", "offers", "opportunities"}, summary["available_data_types"].([]string))
}

func TestBuildFullContext_IncludesBusinessYearSummaryForYearQueries(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	opportunity := Opportunity{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		FolderNumber: "2026-PIPE-01",
		CustomerName: "National Petroleum Co.",
		Title:        "Pipeline visibility",
		Stage:        "Qualified",
		RevenueBHD:   1000,
		Year:         2026,
		OfferDate:    time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, app.db.Create(&opportunity).Error)

	ctx := app.buildFullContext(Intent{RawQuery: "do you have access to 2026 data?", Domain: "general"})
	yearSummary, ok := ctx["business_year_summary"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, 2026, yearSummary["year"])

	opps := yearSummary["opportunities"].(map[string]any)
	require.Equal(t, 1, opps["count"])
	require.Equal(t, 1000.0, opps["total_value_bhd"])
}

func TestManualMatchLine_SupportsSupplierPaymentMatches(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	statement := BankStatement{
		Base:            Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		BankAccountID:   "acct-1",
		StatementNumber: "STMT-2026-001",
		StatementDate:   time.Now(),
		PeriodStart:     time.Now().AddDate(0, 0, -7),
		PeriodEnd:       time.Now(),
		Status:          "Imported",
	}
	require.NoError(t, app.db.Create(&statement).Error)

	line := BankStatementLine{
		Base:            Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		BankStatementID: statement.ID,
		LineNumber:      1,
		TransactionDate: time.Now(),
		ValueDate:       time.Now(),
		Description:     "Supplier transfer",
		Debit:           110,
		Credit:          0,
		Balance:         890,
	}
	require.NoError(t, app.db.Create(&line).Error)

	payment := SupplierPayment{
		Base:              Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierInvoiceID: "supp-inv-1",
		SupplierID:        "sup-1",
		SupplierName:      "Rhine Instruments",
		InvoiceNumber:     "SUPP-2026-001",
		AmountForeign:     110,
		Currency:          "BHD",
		ExchangeRate:      1,
		AmountBHD:         110,
		PaymentDate:       time.Now(),
		PaymentMethod:     "Bank Transfer",
		Reference:         "TXN-1001",
	}
	require.NoError(t, app.db.Create(&payment).Error)

	require.NoError(t, app.ManualMatchLine(line.ID, "SUPPLIER_PAYMENT", payment.ID, "admin"))

	var updatedLine BankStatementLine
	require.NoError(t, app.db.First(&updatedLine, "id = ?", line.ID).Error)
	require.True(t, updatedLine.IsMatched)
	require.Equal(t, "Manual", updatedLine.MatchType)
	require.Equal(t, payment.ID, updatedLine.MatchedPaymentID)

	var auditCount int64
	require.NoError(t, app.db.Model(&BankReconciliationAuditLog{}).Where("bank_statement_id = ?", statement.ID).Count(&auditCount).Error)
	require.Equal(t, int64(1), auditCount)
}

func TestRepairImportedCommercialDocuments_RemovesSyntheticRowsAndRenamesImportedOrders(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	offer := Offer{
		Base:            Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferNumber:     "EH-64-26-R0",
		CustomerName:    "Harbor Dairy Company W.L.L.",
		QuotationDate:   time.Now(),
		ValidityDate:    time.Now().AddDate(0, 0, 30),
		Stage:           "Quoted",
		PaymentTerms:    "Net 30",
		DeliveryTerms:   "DAP Bahrain",
		VatRate:         10,
		TotalValueBHD:   858,
		EstimatedMargin: 20,
	}
	require.NoError(t, app.db.Create(&offer).Error)

	require.NoError(t, app.db.Create(&[]OfferItem{
		{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OfferID:     offer.ID,
			LineNumber:  1,
			ProductCode: "TT411",
			Model:       "TT411-4EVR5/101",
			Description: "ModuLine Temp Transmitter TT411",
			Quantity:    2,
			UnitPrice:   390,
			TotalPrice:  780,
		},
		{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OfferID:     offer.ID,
			LineNumber:  2,
			ProductCode: "TOTAL",
			Description: "Total for Order",
			Quantity:    1,
			UnitPrice:   780,
			TotalPrice:  780,
		},
		{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OfferID:     offer.ID,
			LineNumber:  3,
			Description: "Line Item 3 -",
			Equipment:   "Line Item 3",
			Quantity:    1,
			UnitPrice:   1,
			TotalPrice:  1,
		},
	}).Error)

	order := Order{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrderNumber:   "IMP-EH-64-26",
		OfferID:       offer.ID,
		OfferNumber:   offer.OfferNumber,
		CustomerName:  offer.CustomerName,
		Status:        "Confirmed",
		OrderDate:     time.Now(),
		TotalValueBHD: 1560,
		GrandTotalBHD: 1560,
	}
	require.NoError(t, app.db.Create(&order).Error)

	require.NoError(t, app.db.Create(&[]OrderItem{
		{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			LineNumber:  1,
			ProductCode: "TT411",
			Model:       "TT411-4EVR5/101",
			Description: "ModuLine Temp Transmitter TT411",
			Quantity:    2,
			UnitPrice:   390,
			TotalPrice:  780,
		},
		{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			LineNumber:  2,
			ProductCode: "TOTAL",
			Description: "Total for Order",
			Quantity:    1,
			UnitPrice:   780,
			TotalPrice:  780,
		},
		{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			LineNumber:  3,
			Description: "Line Item 3 -",
			Equipment:   "Line Item 3",
			Quantity:    1,
			UnitPrice:   1,
			TotalPrice:  1,
		},
	}).Error)

	opportunity := Opportunity{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		FolderNumber:   "2026-64",
		CustomerName:   offer.CustomerName,
		Year:           2026,
		Source:         "2026_onedrive",
		ProductDetails: `[{"description":"ModuLine Temp Transmitter TT411","quantity":2,"unit_price":390,"total_price":780,"part_number":"TT411-4EVR5/101"},{"description":"Line Item 2 -","quantity":1,"unit_price":1,"total_price":1}]`,
	}
	require.NoError(t, app.db.Create(&opportunity).Error)

	require.NoError(t, repairImportedCommercialDocuments(app.db))

	var repairedOffer Offer
	require.NoError(t, app.db.Preload("Items").First(&repairedOffer, "id = ?", offer.ID).Error)
	require.Len(t, repairedOffer.Items, 1)
	require.Equal(t, 1, repairedOffer.Items[0].LineNumber)
	require.Equal(t, 858.0, repairedOffer.TotalValueBHD)

	var repairedOrder Order
	require.NoError(t, app.db.Preload("Items").First(&repairedOrder, "id = ?", order.ID).Error)
	require.Equal(t, offer.OfferNumber, repairedOrder.OrderNumber)
	require.Len(t, repairedOrder.Items, 1)
	require.Equal(t, 780.0, repairedOrder.TotalValueBHD)
	require.Equal(t, 780.0, repairedOrder.GrandTotalBHD)

	var repairedOpportunity Opportunity
	require.NoError(t, app.db.First(&repairedOpportunity, "id = ?", opportunity.ID).Error)
	require.Contains(t, repairedOpportunity.ProductDetails, "ModuLine Temp Transmitter TT411")
	require.NotContains(t, repairedOpportunity.ProductDetails, "Line Item")
}

func TestUpdateOfferFull_PersistsMetadataAndClearsHeaderFields(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	customerID := seedTestCustomer(t, app.db, "Harbor Dairy Company W.L.L.")

	offer := Offer{
		Base:              Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferNumber:       "43-26",
		RevisionNumber:    1,
		CustomerID:        customerID,
		CustomerName:      "Harbor Dairy Company W.L.L.",
		QuotationDate:     time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC),
		ValidityDate:      time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC),
		Stage:             "Quoted",
		PaymentTerms:      "Net 30",
		DeliveryTerms:     "DAP Bahrain",
		DeliveryWeeks:     "4-6 weeks",
		CountryOfOrigin:   "DE",
		IssuedBy:          "Jamie Wong",
		ContactPhone:      "+973-1700-0000",
		CustomerReference: "EH-64-26-R0",
		AttentionPerson:   "Mr. Hassan",
		AttentionCompany:  "HARBOR DAIRY COMPANY W.L.L.",
		AttentionPhone:    "+973-1700-0001",
		AttentionAddress:  "Manama, Bahrain",
		VatRate:           10,
		TotalValueBHD:     844.8,
	}
	require.NoError(t, app.db.Create(&offer).Error)
	require.NoError(t, app.db.Create(&OfferItem{
		Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferID:     offer.ID,
		LineNumber:  1,
		ProductCode: "TT411",
		Description: "Old item",
		Quantity:    1,
		UnitPrice:   844.8,
		TotalPrice:  844.8,
	}).Error)

	opportunity := Opportunity{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferID:      offer.ID,
		FolderNumber: "1-26",
		CustomerID:   customerID,
		CustomerName: offer.CustomerName,
		Title:        "Legacy title",
		Stage:        "Quoted",
	}
	require.NoError(t, app.db.Create(&opportunity).Error)

	validityDate := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	updated, err := app.UpdateOfferFull(offer.ID, OfferUpdateData{
		CustomerName:      offer.CustomerName,
		ProjectName:       "Supply of Rhine TEMP. TRANSMITTER",
		FolderNumber:      "64-26",
		QuotationDate:     "2026-03-30",
		ValidityDate:      validityDate,
		PaymentTerms:      "",
		DeliveryTerms:     "",
		DeliveryWeeks:     "",
		CountryOfOrigin:   "",
		IssuedBy:          "",
		ContactPhone:      "",
		CustomerReference: "",
		AttentionPerson:   "",
		AttentionCompany:  "",
		AttentionPhone:    "",
		AttentionAddress:  "",
		VatRate:           0,
		Items: []OfferUpdateItem{
			{
				Description: "ModuLine Temp Transmitter TT411",
				ProductCode: "TT411",
				Model:       "TT411-4EVR5/101",
				Quantity:    2,
				UnitPrice:   384,
				TotalPrice:  768,
			},
			{
				Description: "Total for Order",
				ProductCode: "TOTAL",
				Quantity:    1,
				UnitPrice:   768,
				TotalPrice:  768,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 0.0, updated.VatRate)
	require.Equal(t, 768.0, updated.TotalValueBHD)

	var stored Offer
	require.NoError(t, app.db.Preload("Items").First(&stored, "id = ?", offer.ID).Error)
	require.Equal(t, "", stored.PaymentTerms)
	require.Equal(t, "", stored.DeliveryTerms)
	require.Equal(t, "", stored.DeliveryWeeks)
	require.Equal(t, "", stored.CountryOfOrigin)
	require.Equal(t, "", stored.IssuedBy)
	require.Equal(t, "", stored.ContactPhone)
	require.Equal(t, "", stored.CustomerReference)
	require.Equal(t, "", stored.AttentionPerson)
	require.Equal(t, "", stored.AttentionCompany)
	require.Equal(t, "", stored.AttentionPhone)
	require.Equal(t, "", stored.AttentionAddress)
	require.Equal(t, 0.0, stored.VatRate)
	require.Len(t, stored.Items, 1)
	require.Equal(t, 2, stored.RevisionNumber)

	var linked Opportunity
	require.NoError(t, app.db.First(&linked, "id = ?", opportunity.ID).Error)
	require.Equal(t, "64-26", linked.FolderNumber)
	require.Equal(t, "Supply of Rhine TEMP. TRANSMITTER", linked.Title)
	require.Equal(t, offer.CustomerName, linked.CustomerName)
}

func TestUpdateOfferFull_HeaderOnlySavePreservesExistingItems(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	validityDate := time.Now().AddDate(0, 0, 30)
	offer := Offer{
		Base:            Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferNumber:     "43-26-R1",
		RevisionNumber:  1,
		CustomerName:    "Harbor Dairy Company W.L.L.",
		QuotationDate:   time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC),
		ValidityDate:    validityDate,
		Stage:           "Quoted",
		PaymentTerms:    "Net 30",
		DeliveryTerms:   "DAP Bahrain",
		TotalValueBHD:   844.8,
		AttentionPerson: "Existing contact",
	}
	require.NoError(t, app.db.Create(&offer).Error)
	require.NoError(t, app.db.Create(&[]OfferItem{
		{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OfferID:     offer.ID,
			LineNumber:  1,
			ProductCode: "TT411",
			Description: "Temperature transmitter",
			Quantity:    1,
			UnitPrice:   420,
			TotalPrice:  420,
		},
		{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OfferID:     offer.ID,
			LineNumber:  2,
			ProductCode: "TT131",
			Description: "Thermowell",
			Quantity:    2,
			UnitPrice:   212.4,
			TotalPrice:  424.8,
		},
	}).Error)

	updated, err := app.UpdateOfferFull(offer.ID, OfferUpdateData{
		CustomerName:    offer.CustomerName,
		QuotationDate:   "2026-03-30",
		ValidityDate:    validityDate.Format("2006-01-02"),
		PaymentTerms:    "Net 45",
		DeliveryTerms:   "CIF Bahrain",
		AttentionPerson: "Updated contact",
		Items:           []OfferUpdateItem{},
	})
	require.NoError(t, err)
	require.Equal(t, 844.8, updated.TotalValueBHD)

	var stored Offer
	require.NoError(t, app.db.Preload("Items").First(&stored, "id = ?", offer.ID).Error)
	require.Equal(t, "Net 45", stored.PaymentTerms)
	require.Equal(t, "CIF Bahrain", stored.DeliveryTerms)
	require.Equal(t, "Updated contact", stored.AttentionPerson)
	require.Len(t, stored.Items, 2)
	require.Equal(t, 420.0, stored.Items[0].TotalPrice)
	require.Equal(t, 424.8, stored.Items[1].TotalPrice)
}

func TestUpdateOfferFull_AllowsSameDayValidityOnHeaderSave(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	offer := Offer{
		Base:           Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		OfferNumber:    "44-26-R1",
		RevisionNumber: 1,
		CustomerName:   "Forge Steel Company B.S.C Closed",
		QuotationDate:  today.AddDate(0, 0, -30),
		ValidityDate:   today,
		Stage:          "Quoted",
		PaymentTerms:   "Net 30",
		DeliveryTerms:  "DAP Bahrain",
		TotalValueBHD:  6500,
	}
	require.NoError(t, app.db.Create(&offer).Error)

	updated, err := app.UpdateOfferFull(offer.ID, OfferUpdateData{
		CustomerName:  offer.CustomerName,
		QuotationDate: offer.QuotationDate.Format("2006-01-02"),
		ValidityDate:  today.Format("2006-01-02"),
		PaymentTerms:  "30 days from Date of Delivery",
		Items:         []OfferUpdateItem{},
	})
	require.NoError(t, err)
	require.Equal(t, "30 days from Date of Delivery", updated.PaymentTerms)
}

func TestOfferPDFExportUsesSavedSubjectAndBody(t *testing.T) {
	offer := Offer{
		OfferNumber:        "44-26",
		RevisionNumber:     1,
		CustomerName:       "Harbor Dairy Company W.L.L.",
		QuotationDate:      time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		VatRate:            10,
		Subject:            "Sub: Revised pump package",
		Body:               "Custom commercial opening body for this quote.",
		PaymentTerms:       "Net 30",
		DeliveryTerms:      "DAP Bahrain",
		TermsAndConditions: "Custom terms",
		Items: []OfferItem{
			{
				LineNumber: 1,
				Equipment:  "Flowmeter",
				Model:      "FMR20B",
				Quantity:   2,
				UnitPrice:  3250,
				TotalPrice: 6500,
			},
		},
	}

	exportData := buildCostingExportDataFromOffer(offer, CustomerMaster{BusinessName: offer.CustomerName}, CustomerContact{})

	require.Equal(t, "Sub: Revised pump package", exportData.Subject)
	require.Equal(t, "Custom commercial opening body for this quote.", exportData.Body)
	require.Equal(t, "44-26-R1", exportData.CostingId)
	require.Equal(t, 7150.0, exportData.GrandTotal)
}

func TestSaveCostingAsOfferUpdatesExistingOfferFromCostingSheet(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	customer := CustomerMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		BusinessName: "Harbor Dairy Company W.L.L.",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	offer := Offer{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferNumber:    "43-26",
		RevisionNumber: 1,
		CustomerID:     customer.ID,
		CustomerName:   customer.BusinessName,
		QuotationDate:  time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		ValidityDate:   time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC),
		Stage:          "Quoted",
		TotalValueBHD:  495,
		Subject:        "Sub: Old subject",
		Body:           "Old body",
		VatRate:        10,
		Division:       "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&offer).Error)
	require.NoError(t, app.db.Create(&OfferItem{
		Base:       Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferID:    offer.ID,
		LineNumber: 1,
		Equipment:  "Legacy",
		Quantity:   1,
		UnitPrice:  450,
		TotalPrice: 450,
	}).Error)

	updated, err := app.SaveCostingAsOffer(CostingExportData{
		OfferID:       offer.ID,
		OfferNumber:   offer.OfferNumber,
		Division:      "Acme Instrumentation",
		Date:          "2026-04-27",
		CustomerID:    customer.ID,
		CustomerName:  customer.BusinessName,
		Subject:       "Sub: Revised subject",
		Body:          "Revised customer-facing body",
		QuoteType:     "Quotation",
		VatRate:       10,
		Subtotal:      1990,
		NetAmount:     1990,
		VAT:           199,
		GrandTotal:    2189,
		TotalCost:     1000,
		Profit:        990,
		ProfitPercent: 49.7487437186,
		LineItems: []CostingExportLineItem{
			{
				SlNo:           1,
				Equipment:      "FlowMeter",
				Model:          "9Xm1-s1982",
				Currency:       "EUR",
				Quantity:       5,
				SuggestedPrice: 195,
				TotalPrice:     975,
				ExchangeRate:   0.45,
				TotalCost:      353.9025,
			},
			{
				SlNo:           2,
				Equipment:      "Oilwellgas",
				Model:          "NY1230as-0",
				Currency:       "BHD",
				Quantity:       7,
				SuggestedPrice: 145,
				TotalPrice:     1015,
				ExchangeRate:   1,
				TotalCost:      120.1725,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "Sub: Revised subject", updated.Subject)
	require.Equal(t, "Revised customer-facing body", updated.Body)
	require.Equal(t, 2189.0, updated.TotalValueBHD)
	require.Equal(t, 2, updated.RevisionNumber)

	var stored Offer
	require.NoError(t, app.db.Preload("Items").First(&stored, "id = ?", offer.ID).Error)
	require.Len(t, stored.Items, 2)
	require.Equal(t, 195.0, stored.Items[0].UnitPrice)
	require.Equal(t, 975.0, stored.Items[0].TotalPrice)
	require.Equal(t, 0.45, stored.Items[0].ExchangeRate)
}

func TestRepairImportedCommercialDocumentsRestoresFXMultipliedOfferItems(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	offer := Offer{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferNumber:    "SA-06-26",
		RevisionNumber: 1,
		CustomerName:   "Marvale",
		QuotationDate:  time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		ValidityDate:   time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC),
		Stage:          "Quoted",
		TotalValueBHD:  9360.945,
		VatRate:        10,
	}
	require.NoError(t, app.db.Create(&offer).Error)

	rawItem := OfferItem{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferID:      offer.ID,
		LineNumber:   1,
		Equipment:    "LOT",
		Currency:     "",
		Quantity:     1,
		UnitPrice:    18911,
		TotalPrice:   18911,
		ExchangeRate: 0,
		TotalCost:    15759.765,
	}
	require.NoError(t, app.db.Create(&rawItem).Error)
	require.NoError(t, app.db.Delete(&rawItem).Error)

	require.NoError(t, app.db.Create(&OfferItem{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferID:      offer.ID,
		LineNumber:   1,
		Equipment:    "LOT",
		Currency:     "EUR",
		Quantity:     1,
		UnitPrice:    8509.95,
		TotalPrice:   8509.95,
		ExchangeRate: 0.45,
		TotalCost:    7176.67066116,
	}).Error)

	require.NoError(t, repairImportedCommercialDocuments(app.db))

	var stored Offer
	require.NoError(t, app.db.Preload("Items").First(&stored, "id = ?", offer.ID).Error)
	require.Len(t, stored.Items, 1)
	require.Equal(t, 18911.0, stored.Items[0].UnitPrice)
	require.Equal(t, 18911.0, stored.Items[0].TotalPrice)
	require.Equal(t, 20802.1, stored.TotalValueBHD)
}

func TestUpdateOrder_SkipsSyntheticSummaryRowsAndRecomputesTotals(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	customerID := seedTestCustomer(t, app.db, "National Petroleum Co.")
	order := Order{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrderNumber:   "ORD-2026-001",
		CustomerID:    customerID,
		CustomerName:  "National Petroleum Co.",
		Status:        "Confirmed",
		OrderDate:     time.Now(),
		TotalValueBHD: 100,
		GrandTotalBHD: 100,
	}
	require.NoError(t, app.db.Create(&order).Error)

	updated, err := app.UpdateOrder(order.ID, Order{
		OrderNumber:  order.OrderNumber,
		CustomerName: order.CustomerName,
		Status:       order.Status,
		OrderDate:    order.OrderDate,
		Items: []OrderItem{
			{
				Description: "Pressure transmitter",
				ProductCode: "PT-100",
				Model:       "PT-100",
				Quantity:    2,
				UnitPrice:   390,
				TotalPrice:  780,
			},
			{
				Description: "Total for Order",
				ProductCode: "TOTAL",
				Quantity:    1,
				UnitPrice:   780,
				TotalPrice:  780,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 780.0, updated.TotalValueBHD)
	require.Equal(t, 780.0, updated.GrandTotalBHD)

	var stored Order
	require.NoError(t, app.db.Preload("Items").First(&stored, "id = ?", order.ID).Error)
	require.Len(t, stored.Items, 1)
	require.Equal(t, 1, stored.Items[0].LineNumber)
	require.Equal(t, 780.0, stored.Items[0].TotalPrice)
}

func TestUpdateSupplier_PreservesExistingIdentityFieldsOnPartialPayload(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	supplier := SupplierMaster{
		Base:           Base{CreatedBy: "tester", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierCode:   "SUP-EH001",
		SupplierName:   "MERIDIAN METERING ME FZCO",
		PrimaryContact: "Legacy Contact",
		Email:          "legacy@example.com",
		Country:        "UAE",
		PaymentTerms:   "Net 30",
		LeadTimeDays:   14,
		Rating:         3,
	}
	require.NoError(t, app.db.Create(&supplier).Error)

	updated, err := app.UpdateSupplier(SupplierMaster{
		Base:           Base{ID: supplier.ID},
		SupplierCode:   "",
		SupplierName:   "",
		PrimaryContact: "Updated Contact",
		Email:          "updated@example.com",
		Country:        "International",
		PaymentTerms:   "Net 60",
		LeadTimeDays:   21,
		Rating:         4,
	})
	require.NoError(t, err)
	require.Equal(t, "SUP-EH001", updated.SupplierCode)
	require.Equal(t, "MERIDIAN METERING ME FZCO", updated.SupplierName)
	require.Equal(t, "Updated Contact", updated.PrimaryContact)
	require.Equal(t, "updated@example.com", updated.Email)

	var stored SupplierMaster
	require.NoError(t, app.db.First(&stored, "id = ?", supplier.ID).Error)
	require.Equal(t, "SUP-EH001", stored.SupplierCode)
	require.Equal(t, "MERIDIAN METERING ME FZCO", stored.SupplierName)
	require.Equal(t, "Updated Contact", stored.PrimaryContact)
	require.Equal(t, "Net 60", stored.PaymentTerms)
}

func TestCreateCostingSheet_PersistsCommercialPayloadFields(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	rfq := RFQData{
		RFQNumber: "RFQ-2026-901",
		Client:    "Harbor Dairy Company W.L.L.",
		Project:   "Temperature transmitter package",
		Value:     1250,
		Status:    "New",
		Stage:     "RFQ Received",
	}
	require.NoError(t, app.db.Create(&rfq).Error)

	payload := persistedCostingPayload{
		CustomerName:  "Harbor Dairy Company W.L.L.",
		Subject:       "Subject: Supply of Rhine temperature transmitter package",
		Body:          "We thank you for the opportunity and submit our techno-commercial offer.",
		QuoteType:     "Quotation",
		PreparedBy:    "Jamie",
		PaymentTerms:  "Net 30",
		DeliveryTerms: "DAP Bahrain",
		ProjectName:   "Supply of Rhine TEMP. TRANSMITTER",
		LineItems: []CostingExportLineItem{
			{
				SlNo:           1,
				Equipment:      "Temperature transmitter",
				Model:          "TT411",
				Quantity:       2,
				FOB:            390,
				FreightPercent: 7.5,
				ExchangeRate:   0.377,
				MarkupPercent:  18,
				TotalCost:      314.145,
				SuggestedPrice: 370.6911,
				TotalPrice:     741.3822,
			},
		},
	}
	itemsJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	costing, err := app.CreateCostingSheet(rfq.ID, string(itemsJSON), "Jamie")
	require.NoError(t, err)

	stored, err := app.GetCostingSheet(costing.ID)
	require.NoError(t, err)

	decoded, err := parsePersistedCosting(stored.Items)
	require.NoError(t, err)
	require.Equal(t, payload.Subject, decoded.Subject)
	require.Equal(t, payload.Body, decoded.Body)
	require.Equal(t, payload.PaymentTerms, decoded.PaymentTerms)
	require.Equal(t, payload.DeliveryTerms, decoded.DeliveryTerms)
	require.Len(t, decoded.LineItems, 1)
	require.InDelta(t, 7.5, decoded.LineItems[0].FreightPercent, 0.0001)
	require.InDelta(t, 0.377, decoded.LineItems[0].ExchangeRate, 0.0001)
	require.InDelta(t, 18, decoded.LineItems[0].MarkupPercent, 0.0001)
}
