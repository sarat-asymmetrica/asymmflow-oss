package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestButlerIntentClarification_CapabilitiesQuestionReturnsCommandChoices(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Conversation{}, &ChatMessage{}))

	resp, err := app.ChatWithButlerPersistent("", "hello buddy! I would love it if you could let me understand your capabilities")
	require.NoError(t, err)
	require.Contains(t, resp.Response, "Pick the view")
	require.NotEmpty(t, resp.Actions)
	require.Equal(t, "clarify", resp.Actions[0].Type)
	require.Equal(t, "butler_prompt", resp.Actions[0].Target)

	data, ok := resp.Actions[0].Data.(map[string]any)
	require.True(t, ok)
	require.Contains(t, data["prompt"], "capabilities")

	var assistant ChatMessage
	require.NoError(t, app.db.Where("conversation_id = ? AND role = ?", resp.ConversationID, "assistant").First(&assistant).Error)
	require.Equal(t, "assistant_actionable", assistant.MessageType)
	require.Equal(t, "clarify", assistant.ActionType)
	require.Equal(t, "butler_prompt", assistant.ActionTarget)
	require.Contains(t, assistant.ActionMetadata, "butler_prompt")
}

func TestButlerIntentClarification_ARProjectionAsksForBasis(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Conversation{}, &ChatMessage{}))

	resp, err := app.ChatWithButlerPersistent("", "Yes Sir please get me the AR projections for the next month please!")
	require.NoError(t, err)
	require.Contains(t, resp.Response, "choice matters")
	require.Len(t, resp.Actions, 4)
	require.Equal(t, "Invoices only", resp.Actions[0].Label)
	require.Equal(t, "Include confirmed orders", resp.Actions[1].Label)
	require.Equal(t, "Add weighted offers", resp.Actions[2].Label)
}

func TestGroundedARProjection_IncludesConfirmedUninvoicedOrders(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Conversation{}, &ChatMessage{}))

	customerID := seedTestCustomer(t, app.db, "NPC")
	now := time.Now()
	nextMonthStart, _, _ := nextCalendarMonthWindow(now)
	orderDate := time.Date(now.Year(), time.January, 15, 0, 0, 0, 0, time.Local)
	if orderDate.After(now) {
		orderDate = now.AddDate(0, 0, -1)
	}

	order := Order{
		Base:          Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		OrderNumber:   "ORD-AR-001",
		CustomerID:    customerID,
		CustomerName:  "NPC",
		OrderDate:     orderDate,
		Status:        "Processing",
		GrandTotalBHD: 1000,
	}
	require.NoError(t, app.db.Create(&order).Error)

	invoice := Invoice{
		Base:           Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceNumber:  "INV-AR-001",
		CustomerID:     customerID,
		CustomerName:   "NPC",
		InvoiceDate:    now,
		DueDate:        nextMonthStart.AddDate(0, 0, 5),
		Status:         "Sent",
		GrandTotalBHD:  250,
		OutstandingBHD: 250,
	}
	require.NoError(t, app.db.Create(&invoice).Error)

	resp, err := app.ChatWithButlerPersistent("", "but we have 21 orders in the pipeline! will we not have any receivable from them?")
	require.NoError(t, err)
	require.Contains(t, resp.Response, "Confirmed order receivable path")
	require.Contains(t, resp.Response, "1 orders")
	require.Contains(t, resp.Response, "1000.000 BHD")
	require.Contains(t, resp.Response, "Orders are now explicitly counted")
	require.NotEmpty(t, resp.Actions)
	require.Equal(t, "clarify", resp.Actions[0].Type)
}

func TestButlerIntentClarification_VaguePromptReturnsSalesSafeChoices(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Conversation{}, &ChatMessage{}))

	resp, err := app.ChatWithButlerPersistent("", "do the needful")
	require.NoError(t, err)
	require.Contains(t, resp.Response, "Pick the closest route")
	require.GreaterOrEqual(t, len(resp.Actions), 4)

	labels := make([]string, 0, len(resp.Actions))
	for _, action := range resp.Actions {
		require.Equal(t, "clarify", action.Type)
		require.Equal(t, "butler_prompt", action.Target)
		labels = append(labels, action.Label)
	}
	require.Contains(t, labels, "Find offer/order")
	require.Contains(t, labels, "Draft customer email")
	require.Contains(t, labels, "Search item/model")
}

func TestButlerFinancePrivilegeBlocksSensitiveSalesPipelineTerms(t *testing.T) {
	require.True(t, requiresFinancePrivilege(
		classifyIntent("show pipeline revenue and profit margin for open offers"),
		"show pipeline revenue and profit margin for open offers",
	))
	require.False(t, requiresFinancePrivilege(
		classifyIntent("find offer OFF-2026-001 and list its items"),
		"find offer OFF-2026-001 and list its items",
	))
}

func TestButlerSalesRoleFinancialQueryBlockedBeforeModel(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Conversation{}, &ChatMessage{}))
	app.currentUserID = "sales-user"
	app.currentUser = &User{
		Base:     Base{ID: "sales-user"},
		Username: "sales-user",
		RoleName: "sales",
		Role: Role{
			Name:        "sales",
			DisplayName: "Sales",
			Permissions: `["intelligence:chat","offers:view","orders:view","customers:view"]`,
		},
	}

	resp, err := app.ChatWithButlerPersistent("", "show pipeline revenue and profit margin for open offers")
	require.NoError(t, err)
	require.Contains(t, resp.Response, "Financial data access requires manager or admin privileges")
	require.Contains(t, resp.Response, "Sales pipeline status without revenue")
	require.Empty(t, resp.Actions)
	require.Equal(t, "blocked", resp.Metadata.ContextMode)
}

func TestButlerSalesContextRedactsPipelineAmounts(t *testing.T) {
	app := setupTestApp(t)
	app.currentUserID = "sales-user"
	app.currentUser = &User{
		Base:     Base{ID: "sales-user"},
		Username: "sales-user",
		RoleName: "sales",
		Role: Role{
			Name:        "sales",
			DisplayName: "Sales",
			Permissions: `["intelligence:chat","offers:view","orders:view","customers:view"]`,
		},
	}

	customerID := seedTestCustomer(t, app.db, "NPC")
	now := time.Now()
	offer := Offer{
		Base:            Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		OfferNumber:     "OFF-SALES-001",
		CustomerID:      customerID,
		CustomerName:    "NPC",
		QuotationDate:   now,
		ValidityDate:    now.AddDate(0, 0, 30),
		Stage:           "Quoted",
		TotalValueBHD:   9999,
		EstimatedMargin: 37,
	}
	require.NoError(t, app.db.Create(&offer).Error)
	require.NoError(t, app.db.Create(&OfferItem{
		Base:          Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		OfferID:       offer.ID,
		LineNumber:    1,
		Equipment:     "Analyzer",
		Model:         "MODEL-SAFE",
		Quantity:      2,
		UnitPrice:     4999.5,
		TotalPrice:    9999,
		MarginPercent: 37,
		Currency:      "BHD",
	}).Error)
	require.NoError(t, app.db.Create(&Order{
		Base:          Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		OrderNumber:   "ORD-SALES-001",
		CustomerID:    customerID,
		CustomerName:  "NPC",
		OrderDate:     now,
		Status:        "Processing",
		GrandTotalBHD: 8888,
	}).Error)
	require.NoError(t, app.db.Create(&Opportunity{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		FolderNumber: "OPP-SALES-001",
		CustomerID:   customerID,
		CustomerName: "NPC",
		Stage:        "Qualified",
		RevenueBHD:   7777,
		Confidence:   0.8,
		Division:     "Acme Instrumentation",
	}).Error)

	context := app.buildFullContext(classifyIntent("show sales pipeline status"))
	require.NotContains(t, context, "financial_data")
	require.NotContains(t, context, "forecast_intelligence")

	ops, ok := context["operations_data"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, ops, "active_offer_pipeline_bhd")
	require.NotContains(t, ops, "active_order_value_bhd")

	payload, err := json.Marshal(context)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "unit_price_bhd")
	require.NotContains(t, string(payload), "margin_pct")
	require.NotContains(t, string(payload), "9999")
	require.NotContains(t, string(payload), "8888")
	require.NotContains(t, string(payload), "7777")
}
