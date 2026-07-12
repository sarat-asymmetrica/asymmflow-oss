package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOCRPermissionsAssignedToDocumentCapableRoles(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.SeedDefaultRoles())

	for _, roleName := range []string{"manager", "sales", "operations", "staff"} {
		t.Run(roleName, func(t *testing.T) {
			for _, permission := range []string{"documents:view", "documents:create", "documents:classify"} {
				require.Contains(t, rolePermissions[roleName], permission)
				require.True(t, app.CheckPermissionByRole(roleName, permission), "%s should have %s", roleName, permission)
			}
		})
	}
}

func TestNormalizeOCRDocumentTypeCoversClassifierOutputs(t *testing.T) {
	tests := map[string]string{
		"RFQ":             "rfq",
		"Invoice":         "invoice",
		"CustomerInvoice": "invoice",
		"SupplierInvoice": "supplier_invoice",
		"PurchaseOrder":   "purchase_order",
		"PO":              "purchase_order",
		"Quotation":       "quotation",
		"Quote":           "quotation",
		"DeliveryNote":    "delivery_note",
		"BankStatement":   "bank_statement",
		"CostingSheet":    "costing",
		"ExcelData":       "excel_data",
		"contract":        "contract",
		"bank-statement":  "bank_statement",
		"":                "other",
	}

	for input, expected := range tests {
		require.Equal(t, expected, normalizeOCRDocumentType(input), input)
	}
}

func TestSaveDocumentToEntityNormalizesClassifierTypeAndRoutesOpportunity(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&OCRDocument{}, &Opportunity{}))

	payload, err := json.Marshal(map[string]any{
		"customer_name": "NPC",
		"project":       "Valve replacement RFQ",
		"rfq_number":    "OCR-RFQ-001",
		"division":      "Acme Instrumentation",
		"total":         1250.0,
	})
	require.NoError(t, err)

	result, err := app.SaveDocumentToEntity(
		"rfq.pdf",
		"/tmp/rfq.pdf",
		"RFQ",
		"Request for Quotation\nPlease quote valve replacement items.",
		0.91,
		12,
		"unit-test",
		string(payload),
	)
	require.NoError(t, err)
	require.Equal(t, true, result["routed"])
	require.Equal(t, "rfq", result["document_type"])
	require.Equal(t, "opportunities", result["entity_table"])

	var opportunity Opportunity
	require.NoError(t, app.db.Where("folder_number = ?", "OCR-RFQ-001").First(&opportunity).Error)
	require.Equal(t, "NPC", opportunity.CustomerName)
}

func TestDeploymentFoundationRepairsLegacyOpportunityDivisionBeforeOCRSave(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&OCRDocument{}))
	require.NoError(t, app.db.Exec("DROP TABLE IF EXISTS opportunities").Error)
	require.NoError(t, app.db.Exec(`
		CREATE TABLE opportunities (
			id TEXT PRIMARY KEY,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			version INTEGER DEFAULT 1,
			created_by TEXT,
			folder_number TEXT,
			offer_id TEXT,
			customer_id TEXT,
			customer_name TEXT,
			customer_grade TEXT,
			salesperson TEXT,
			year INTEGER DEFAULT 0,
			opp_number INTEGER DEFAULT 0,
			folder_name TEXT DEFAULT '',
			title TEXT DEFAULT '',
			eh_ref TEXT DEFAULT '',
			source TEXT DEFAULT '',
			comment TEXT DEFAULT '',
			owner_notes TEXT DEFAULT '',
			product_details TEXT,
			offer_date datetime,
			order_date datetime,
			expected_date datetime,
			closed_date datetime,
			delivery_terms TEXT DEFAULT '',
			payment_terms TEXT DEFAULT '',
			revenue_bhd REAL DEFAULT 0,
			cost_bhd REAL DEFAULT 0,
			profit_bhd REAL DEFAULT 0,
			spoc_status TEXT DEFAULT '',
			wip_status TEXT DEFAULT '',
			stage TEXT DEFAULT '',
			regime INTEGER DEFAULT 0,
			confidence REAL DEFAULT 0,
			r1 REAL DEFAULT 0,
			r2 REAL DEFAULT 0,
			r3 REAL DEFAULT 0,
			has_abb_competition BOOLEAN DEFAULT 0,
			product_type TEXT DEFAULT '',
			won_reason TEXT DEFAULT '',
			lost_reason TEXT DEFAULT ''
		)
	`).Error)
	require.False(t, app.hasColumn("opportunities", "division"))

	require.NoError(t, app.ensureCriticalDeploymentFoundations())
	require.True(t, app.hasColumn("opportunities", "division"))

	payload, err := json.Marshal(map[string]any{
		"customer_name": "NPC",
		"project":       "Analyzer OCR offer",
		"rfq_number":    "OCR-RFQ-LEGACY-001",
		"division":      "Acme Instrumentation",
		"total":         3250.0,
	})
	require.NoError(t, err)

	result, err := app.SaveDocumentToEntity(
		"ocr-offer.pdf",
		"/tmp/ocr-offer.pdf",
		"RFQ",
		"Request for Quotation\nAnalyzer line items from OCR.",
		0.94,
		16,
		"unit-test",
		string(payload),
	)
	require.NoError(t, err)
	require.Equal(t, "opportunities", result["entity_table"])

	var opportunity Opportunity
	require.NoError(t, app.db.Where("folder_number = ?", "OCR-RFQ-LEGACY-001").First(&opportunity).Error)
	require.Equal(t, "Acme Instrumentation", opportunity.Division)
	require.Equal(t, 3250.0, opportunity.RevenueBHD)
}

func TestSaveDocumentToEntityRequiresDownstreamWorkflowPermission(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&OCRDocument{}))

	app.currentUserID = "staff-user"
	app.currentUser = &User{
		Base:     Base{ID: "staff-user"},
		Username: "staff",
		RoleName: "staff",
		Role: Role{
			Name:        "staff",
			DisplayName: "Staff",
			Permissions: `["documents:view","documents:create","documents:classify"]`,
		},
	}

	payload, err := json.Marshal(map[string]any{
		"bank_name":       "National Bank of Bahrain",
		"account_number":  "123456",
		"opening_balance": 1000.0,
		"closing_balance": 950.0,
	})
	require.NoError(t, err)

	_, err = app.SaveDocumentToEntity(
		"statement.pdf",
		"/tmp/statement.pdf",
		"BankStatement",
		"Bank Statement\nOpening Balance\nDebit Credit Balance\nClosing Balance",
		0.88,
		25,
		"unit-test",
		string(payload),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "finance:create")

	var docs int64
	require.NoError(t, app.db.Model(&OCRDocument{}).Count(&docs).Error)
	require.Zero(t, docs)
}

func TestInternalOCRClassifierFallbackWorksWithoutClassifyPermission(t *testing.T) {
	app := setupTestApp(t)
	app.currentUserID = "document-uploader"
	app.currentUser = &User{
		Base:     Base{ID: "document-uploader"},
		Username: "document-uploader",
		RoleName: "limited",
		Role: Role{
			Name:        "limited",
			DisplayName: "Limited",
			Permissions: `["documents:create"]`,
		},
	}

	require.Nil(t, app.AIClassifyDocumentType("Request for Quotation\nPlease quote analyzer spares.", "rfq.pdf"))

	result := app.classifyDocumentForOCR("Request for Quotation\nPlease quote analyzer spares.", "rfq.pdf")
	require.NotNil(t, result)
	require.Equal(t, "RFQ", result.DocumentType)
}
