package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type butlerHarnessFixture struct {
	customer CustomerMaster
	supplier SupplierMaster
	employee Employee
}

func seedButlerBusinessHarnessFixture(t *testing.T) (*App, butlerHarnessFixture) {
	t.Helper()

	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)
	require.NoError(t, app.db.AutoMigrate(
		&CustomerContact{},
		&SupplierContact{},
		&SupplierIssue{},
		&FollowUpTask{},
		&PurchaseOrder{},
		&PurchaseOrderItem{},
		&GoodsReceivedNote{},
		&OfferFollowUp{},
	))

	adminRole := Role{
		Base:        Base{ID: uuid.New().String()},
		Name:        "admin",
		DisplayName: "Admin",
		Permissions: `["*","finance:view","intelligence:chat","tasks:view","notifications:view"]`,
		IsActive:    true,
	}
	require.NoError(t, app.db.Create(&adminRole).Error)
	app.currentUser = &User{
		Base:     Base{ID: "butler-harness-admin"},
		Username: "butler-harness-admin",
		RoleName: "admin",
		RoleID:   adminRole.ID,
		Role:     adminRole,
	}
	app.currentUserID = "butler-harness-admin"

	customer := CustomerMaster{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerID:       "C-NATPETRO-001",
		CustomerCode:     "NATPETRO-001",
		BusinessName:     "National Petroleum Co.",
		CustomerGrade:    "A",
		PaymentGrade:     "A",
		Industry:         "Energy",
		PaymentTermsDays: 45,
		AvgPaymentDays:   32,
		OutstandingBHD:   1250.500,
		OverdueDays:      7,
		TotalOrdersCount: 3,
		TotalOrdersValue: 14850.000,
		ARRiskTier:       "watch",
		IsCreditBlocked:  false,
		CreditLimitBHD:   50000,
		CustomerType:     "Corporate",
		City:             "Awali",
		Country:          "Bahrain",
		PrimaryPhone:     "+973-1700-0000",
		PrimaryEmail:     "pat.morgan@nationalpetroleum.example",
	}
	require.NoError(t, app.db.Create(&customer).Error)
	require.NoError(t, app.db.Create(&CustomerContact{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerID:       customer.ID,
		ContactName:      "Pat Morgan",
		JobTitle:         "Instrument Engineer",
		Email:            "pat.morgan@nationalpetroleum.example",
		Phone:            "+973-1700-0000",
		IsPrimaryContact: true,
	}).Error)
	require.NoError(t, app.db.Create(&EntityNote{
		Base:       Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EntityType: "customer",
		EntityID:   customer.ID,
		NoteType:   "general",
		Content:    "National Petroleum prefers split pricing and quick calibration turnaround.",
	}).Error)
	require.NoError(t, app.db.Create(&FollowUpTask{
		Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerID:  customer.ID,
		Title:       "Follow up on calibration budget",
		Description: "Check if commercial split is approved.",
		DueDate:     time.Now().AddDate(0, 0, 2),
		Status:      "pending",
		Priority:    "high",
		Type:        "commercial",
		Amount:      1500,
		Contact:     "Pat Morgan",
		Notes:       "Carry forward latest offer note.",
	}).Error)

	supplier := SupplierMaster{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierCode:   "SUP-EH-001",
		SupplierName:   "Rhine Instruments AG",
		Country:        "Switzerland",
		LeadTimeDays:   30,
		SupplierType:   "Manufacturer",
		BrandsHandled:  `["Rhine Instruments"]`,
		PrimaryContact: "Markus Vogel",
		Email:          "markus.vogel@rhine-instruments.example",
		PaymentTerms:   "Net 30",
		Rating:         5,
		Notes:          "Preferred OEM for instrumentation packages.",
	}
	require.NoError(t, app.db.Create(&supplier).Error)
	require.NoError(t, app.db.Create(&SupplierContact{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierID:       supplier.ID,
		ContactName:      "Markus Vogel",
		JobTitle:         "Regional Sales Manager",
		Email:            "markus.vogel@rhine-instruments.example",
		Phone:            "+41-555-0100",
		IsPrimaryContact: true,
	}).Error)
	require.NoError(t, app.db.Create(&EntityNote{
		Base:       Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EntityType: "supplier",
		EntityID:   supplier.ID,
		NoteType:   "commercial",
		Content:    "Rhine Instruments approved expedited lead time for urgent jobs.",
	}).Error)
	require.NoError(t, app.db.Create(&SupplierIssue{
		Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierID:  supplier.ID,
		OrderRef:    "PO-2026-1001",
		Description: "One transmitter arrived with incomplete certification.",
		Status:      "pending",
		CostBHD:     85,
	}).Error)

	po := PurchaseOrder{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierID:       supplier.ID,
		SupplierName:     supplier.SupplierName,
		PONumber:         "PO-2026-1001",
		PODate:           time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
		ExpectedDelivery: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Currency:         "EUR",
		ExchangeRate:     0.410,
		SubtotalForeign:  9500,
		SubtotalBHD:      3895,
		TotalForeign:     9500,
		TotalBHD:         3895,
		PaymentTerms:     "Net 30",
		PaymentDueDate:   time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Status:           "Sent",
		Division:         "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&po).Error)

	supplierInvoice := SupplierInvoice{
		Base:            Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierID:      supplier.ID,
		SupplierName:    supplier.SupplierName,
		PurchaseOrderID: po.ID,
		PONumber:        po.PONumber,
		InvoiceNumber:   "RH-INV-7781",
		InvoiceDate:     time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
		DueDate:         time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC),
		Currency:        "EUR",
		ExchangeRate:    0.410,
		SubtotalForeign: 9500,
		SubtotalBHD:     3895,
		TotalForeign:    9500,
		TotalBHD:        3895,
		Status:          "Approved",
		PaymentStatus:   "Paid",
		Division:        "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&supplierInvoice).Error)
	require.NoError(t, app.db.Create(&SupplierInvoiceItem{
		Base:              Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierInvoiceID: supplierInvoice.ID,
		LineNumber:        1,
		Description:       "Coriolis flow transmitter package",
		Quantity:          2,
		UnitPrice:         4750,
		TotalPrice:        9500,
		Currency:          "EUR",
	}).Error)
	require.NoError(t, app.db.Create(&SupplierPayment{
		Base:              Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierInvoiceID: supplierInvoice.ID,
		SupplierID:        supplier.ID,
		AmountForeign:     9500,
		Currency:          "EUR",
		ExchangeRate:      0.410,
		AmountBHD:         3895,
		PaymentDate:       time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		PaymentMethod:     "Bank Transfer",
		Reference:         "WIRE-RH-APR26",
		SupplierName:      supplier.SupplierName,
		InvoiceNumber:     supplierInvoice.InvoiceNumber,
		Division:          "Acme Instrumentation",
	}).Error)

	offer := Offer{
		Base:            Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferNumber:     "OFF-2026-0201",
		CustomerID:      customer.CustomerID,
		CustomerName:    customer.BusinessName,
		QuotationDate:   time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		ValidityDate:    time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC),
		Stage:           "Quoted",
		TotalValueBHD:   1500,
		EstimatedMargin: 32,
		QuoteType:       "Quotation",
		Division:        "Acme Instrumentation",
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
		NoteDate: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC),
		Content:  "Customer asked for mobilization note on the cover page.",
	}).Error)
	require.NoError(t, app.db.Create(&OfferFollowUp{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OfferID:      offer.ID,
		FollowUpDate: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
		Notes:        "Check if revised commercial split is approved.",
		Status:       "pending",
	}).Error)

	order := Order{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrderNumber:      "SO-2026-0104",
		CustomerID:       customer.CustomerID,
		CustomerName:     customer.BusinessName,
		OfferID:          offer.ID,
		OfferNumber:      offer.OfferNumber,
		OrderDate:        time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
		Status:           "Processing",
		GrandTotalBHD:    1500,
		CustomerPONumber: "NATPETRO-PO-778",
		PaymentTerms:     "45 days",
		Division:         "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&order).Error)
	require.NoError(t, app.db.Create(&OrderItem{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrderID:       order.ID,
		LineNumber:    1,
		Description:   "Calibration service visit",
		Equipment:     "Flowmeter Calibration",
		Model:         "CAL-PACK",
		Quantity:      2,
		UnitPrice:     750,
		TotalPrice:    1500,
		MarginPercent: 32,
	}).Error)

	invoice := Invoice{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		InvoiceNumber: "INV-2026-0055",
		CustomerID:    customer.CustomerID,
		CustomerName:  customer.BusinessName,
		OfferID:       offer.ID,
		OfferNumber:   offer.OfferNumber,
		OrderID:       order.ID,
		// Seeded "today" so "this quarter"/"this year" prompts stay grounded on
		// any run date (a fixed 2026-04-12 date broke the suite on July 1, and
		// any now-minus-N-days offset breaks in the first N days of a quarter).
		InvoiceDate:     time.Now().UTC(),
		DueDate:         time.Now().UTC().AddDate(0, 0, 45),
		Status:          "Sent",
		SubtotalBHD:     1500,
		GrandTotalBHD:   1500,
		OutstandingBHD:  250,
		IssuedBy:        "Alex Rivera",
		AttentionPerson: "Pat Morgan",
		Division:        "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&invoice).Error)
	require.NoError(t, app.db.Create(&DBInvoiceItem{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		InvoiceID:     invoice.ID,
		LineNumber:    1,
		Description:   "Calibration service visit",
		Equipment:     "Flowmeter Calibration",
		Model:         "CAL-PACK",
		Quantity:      2,
		Rate:          750,
		TotalBHD:      1500,
		MarginPercent: 32,
	}).Error)
	require.NoError(t, app.db.Create(&Payment{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		InvoiceID:      invoice.ID,
		InvoiceNumber:  invoice.InvoiceNumber,
		AmountBHD:      1250,
		PaymentDate:    time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
		PaymentMethod:  "Bank Transfer",
		DaysToPayment:  13,
		IdempotencyKey: uuid.New().String(),
		Reference:      "NATPETRO-TRF-APR26",
		Division:       "Acme Instrumentation",
	}).Error)

	require.NoError(t, app.db.Create(&Opportunity{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		FolderNumber: "2026-901",
		OfferID:      offer.ID,
		CustomerID:   customer.CustomerID,
		CustomerName: customer.BusinessName,
		Title:        "Calibration service package",
		Stage:        "Quoted",
		Salesperson:  "Alex Rivera",
		Division:     "Acme Instrumentation",
		Year:         time.Now().UTC().Year(),
		OfferDate:    time.Now().UTC(), // "offers this year" prompt must stay grounded on any run date
		RevenueBHD:   1500,
	}).Error)

	statement := BankStatement{
		Base:            Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		BankAccountID:   "bank-1",
		StatementNumber: "BS-2026-004",
		StatementDate:   time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC),
		PeriodStart:     time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:       time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC),
		OpeningBalance:  10000,
		ClosingBalance:  11250,
		Status:          "Imported",
		Notes:           "National Petroleum transfer received against calibration invoice.",
		Division:        "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&statement).Error)
	require.NoError(t, app.db.Create(&BankStatementLine{
		Base:              Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		BankStatementID:   statement.ID,
		LineNumber:        1,
		TransactionDate:   time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
		ValueDate:         time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
		Description:       "National Petroleum transfer INV-2026-0055",
		Reference:         "NATPETRO-TRF-APR26",
		Credit:            1250,
		Balance:           11250,
		ExtractedCustomer: customer.BusinessName,
		Notes:             "Matched to April service invoice.",
	}).Error)

	employee := Employee{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EmployeeCode:     "EMP-002",
		FullName:         "Jamie Wong",
		PreferredName:    "Jamie",
		Department:       "Operations",
		JobTitle:         "Coordinator",
		EmploymentStatus: "active",
		IsActive:         true,
		Notes:            "Handles service coordination and client follow-ups.",
	}
	require.NoError(t, app.db.Create(&employee).Error)
	overdue := time.Now().AddDate(0, 0, -2)
	primaryTaskID := uuid.New().String()
	require.NoError(t, app.db.Create(&TaskItem{
		Base:               Base{ID: primaryTaskID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Title:              "Review National Petroleum calibration follow-up",
		Description:        "Check statement note and confirm offer line items.",
		Status:             "open",
		Priority:           "high",
		TaskType:           "service",
		AssigneeEmployeeID: &employee.ID,
		CreatorEmployeeID:  employee.ID,
		CustomerID:         &customer.ID,
		DueDate:            &overdue,
	}).Error)
	require.NoError(t, app.db.Create(&TaskItem{
		Base:               Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Title:              "Confirm supplier paperwork",
		Description:        "Check Rhine Instruments certification status.",
		Status:             "blocked",
		Priority:           "medium",
		TaskType:           "procurement",
		AssigneeEmployeeID: &employee.ID,
		CreatorEmployeeID:  employee.ID,
	}).Error)
	require.NoError(t, app.db.Create(&TaskComment{
		Base:       Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		TaskID:     primaryTaskID,
		EmployeeID: employee.ID,
		Body:       "Cross-check bank statement note and customer offer note before replying.",
	}).Error)
	require.NoError(t, app.db.Create(&Notification{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EmployeeID:       employee.ID,
		NotificationType: "task",
		Title:            "Task reassigned",
		Message:          "A service task was assigned to Jamie.",
		Status:           "unread",
		SourceType:       "task",
		SourceID:         uuid.New().String(),
	}).Error)

	return app, butlerHarnessFixture{
		customer: customer,
		supplier: supplier,
		employee: employee,
	}
}

func TestButlerBusinessHarness_IntentRoutingQuestionBank(t *testing.T) {
	cases := []struct {
		name           string
		prompt         string
		expectedDomain string
		expectedRef    string
		expectedPerson string
	}{
		{"work_tasks", "How many tasks are assigned to Jamie right now?", "work", "employee", "Jamie"},
		{"work_notifications", "What notifications does Jamie have?", "work", "employee", "Jamie"},
		{"customer_invoices", "Show me invoices for National Petroleum Co. this quarter", "financial", "", ""},
		{"supplier_buying", "What did we buy from Rhine Instruments?", "supplier", "", ""},
		{"operations_delivery", "Which orders are pending delivery this week?", "operations", "", ""},
		{"risk_collections", "Which overdue customers need attention today?", "risk", "", ""},
		{"financial_cashflow", "Give me the cash flow projection for this year", "financial", "", ""},
		{"customer_dormant", "Who are our dormant customers?", "customer", "", ""},
		{"operations_pipeline", "What is our pipeline win rate?", "operations", "", ""},
		{"supplier_leadtime", "Which supplier has the worst lead time?", "supplier", "", ""},
		{"market_trend", "What is happening in the instrumentation market?", "market", "", ""},
		{"operations_quote", "Create a quotation for National Petroleum calibration service visit", "operations", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			intent := classifyIntent(tc.prompt)
			require.Equal(t, tc.expectedDomain, intent.Domain)
			if tc.expectedRef != "" {
				require.Equal(t, tc.expectedRef, intent.ReferenceKind)
			}
			if tc.expectedPerson != "" {
				require.Equal(t, tc.expectedPerson, intent.PersonName)
			}
		})
	}
}

func TestButlerBusinessHarness_GroundedQuestionBank(t *testing.T) {
	app, _ := seedButlerBusinessHarnessFixture(t)

	cases := []struct {
		name        string
		prompt      string
		run         func(Intent, string) (string, bool)
		mustContain []string
	}{
		{
			name:        "employee_tasks",
			prompt:      "How many tasks are assigned to Jamie right now?",
			run:         app.tryGroundedWorkFastPath,
			mustContain: []string{"Jamie", "active task(s)", "unread notification(s)"},
		},
		{
			name:        "employee_notifications",
			prompt:      "What notifications does Jamie have?",
			run:         app.tryGroundedWorkFastPath,
			mustContain: []string{"Jamie", "unread notification(s)", "Recent notifications"},
		},
		{
			name:        "customer_quarter_invoices",
			prompt:      "Show me National Petroleum Co. invoices this quarter",
			run:         app.tryGroundedCustomerFastPath,
			mustContain: []string{"invoice(s) for National Petroleum Co.", "Recent invoices"},
		},
		{
			name:        "customer_notes",
			prompt:      "What notes do we have for National Petroleum Co.?",
			run:         app.tryGroundedCustomerFastPath,
			mustContain: []string{"customer note(s) for National Petroleum Co.", "Recent notes"},
		},
		{
			name:        "customer_line_items",
			prompt:      "What have we sold to National Petroleum Co.?",
			run:         app.tryGroundedCustomerFastPath,
			mustContain: []string{"line-item visibility for National Petroleum Co.", "Top sold items"},
		},
		{
			name:        "customer_year_offers",
			prompt:      "Show me National Petroleum Co. offers this year",
			run:         app.tryGroundedCustomerFastPath,
			mustContain: []string{"offer(s) for National Petroleum Co.", "Recent offers"},
		},
		{
			name:        "customer_offer_overview",
			prompt:      "Show me National Petroleum Co. offers",
			run:         app.tryGroundedCustomerFastPath,
			mustContain: []string{"offer(s) for National Petroleum Co.", "Recent offers"},
		},
		{
			name:        "customer_invoice_overview",
			prompt:      "Show me National Petroleum Co. invoices",
			run:         app.tryGroundedCustomerFastPath,
			mustContain: []string{"latest invoice snapshot for National Petroleum Co.", "Payments received"},
		},
		{
			name:        "revenue_projection",
			prompt:      "Give me the revenue projection",
			run:         app.tryGroundedCustomerFastPath,
			mustContain: []string{"Latest revenue projection", "Projection scenarios"},
		},
		{
			name:        "supplier_payment_history",
			prompt:      "Tell me about Rhine Instruments payment history",
			run:         app.tryGroundedSupplierFastPath,
			mustContain: []string{"supplier payment history for Rhine Instruments AG", "Recent payments"},
		},
		{
			name:        "supplier_purchase_history",
			prompt:      "What did we buy from Rhine Instruments?",
			run:         app.tryGroundedSupplierFastPath,
			mustContain: []string{"what we have bought from Rhine Instruments AG", "Recent purchased line items"},
		},
		{
			name:        "supplier_issue_history",
			prompt:      "Are there any active supplier issues for Rhine Instruments?",
			run:         app.tryGroundedSupplierFastPath,
			mustContain: []string{"supplier issue record(s) for Rhine Instruments AG", "Recent issues"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			intent := classifyIntent(tc.prompt)
			reply, handled := tc.run(intent, tc.prompt)
			require.True(t, handled)
			for _, expected := range tc.mustContain {
				require.Contains(t, reply, expected)
			}
		})
	}
}

func TestButlerBusinessHarness_ContextCoverageQuestionBank(t *testing.T) {
	app, _ := seedButlerBusinessHarnessFixture(t)

	cases := []struct {
		name         string
		prompt       string
		expectedKeys []string
	}{
		{"customer_notes", "What notes do we have for National Petroleum Co.?", []string{"entity_resolution", "customer_data", "database_access"}},
		{"supplier_history", "Tell me about Rhine Instruments payment history", []string{"entity_resolution", "supplier_data", "database_access"}},
		{"employee_workload", "How many tasks are assigned to Jamie?", []string{"entity_resolution", "employee_context", "work_data", "database_access"}},
		{"quotation_readiness", "Prepare a quotation for National Petroleum calibration service visit", []string{"entity_resolution", "customer_data", "quotation_precheck"}},
		{"service_due", "What service follow-ups are due?", []string{"service_due_installations"}},
		{"closeable_work", "What can we close this month?", []string{"closeable_opportunities"}},
		{"year_access", "Do you have access to 2026 data?", []string{"business_year_summary"}},
		{"cash_outlook", "What is our cash outlook this year?", []string{"financial_data", "cashflow_projection"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			intent := classifyIntent(tc.prompt)
			context := app.buildFullContext(intent)
			coverage := collectContextCoverage(context)
			for _, expected := range tc.expectedKeys {
				require.Contains(t, coverage, expected)
			}
		})
	}
}
