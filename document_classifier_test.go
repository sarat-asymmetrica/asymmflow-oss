package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFilesystemDocumentClassifier(t *testing.T) {
	service := NewFilesystemDocumentClassifierService()

	tests := []struct {
		name             string
		fileName         string
		folderPath       string
		expectedType     string
		expectedCustomer string
		expectedOffer    string
		expectedStage    string
		expectedProduct  string
	}{
		{
			name:             "GSC PO",
			fileName:         "45312345_GSC_PO.pdf",
			folderPath:       "101 GSC AIT/RFQ",
			expectedType:     "CUSTOMER_PO",
			expectedCustomer: "GSC",
			expectedOffer:    "101",
			expectedStage:    "RFQ",
			expectedProduct:  "AIT",
		},
		{
			name:             "NPC PO",
			fileName:         "PO_81_1234.pdf",
			folderPath:       "102 NPC FIT/OFFER",
			expectedType:     "CUSTOMER_PO",
			expectedCustomer: "NPC",
			expectedOffer:    "102",
			expectedStage:    "OFFER",
			expectedProduct:  "FIT",
		},
		{
			name:             "Rhine Instruments Order Confirmation",
			fileName:         "60112345678.pdf",
			folderPath:       "103 NGA LIT/EXECUTION",
			expectedType:     "SUPPLIER_PO_ACK",
			expectedCustomer: "NGA",
			expectedOffer:    "103",
			expectedStage:    "EXECUTION",
			expectedProduct:  "LIT",
		},
		{
			name:             "PH Internal PO",
			fileName:         "PH25-045.pdf",
			folderPath:       "104 VERTEX SP/OFFER",
			expectedType:     "INTERNAL_PO",
			expectedCustomer: "VERTEX",
			expectedOffer:    "104",
			expectedStage:    "OFFER",
			expectedProduct:  "SP",
		},
		{
			name:             "Invoice",
			fileName:         "PH-25 INV 123.pdf",
			folderPath:       "105 DPC VALVE/EXECUTION",
			expectedType:     "INVOICE",
			expectedCustomer: "DPC",
			expectedOffer:    "105",
			expectedStage:    "EXECUTION",
			expectedProduct:  "VALVE",
		},
		{
			name:             "RFQ Email",
			fileName:         "RFQ from customer.msg",
			folderPath:       "106 CGC TRANSMITTER/RFQ",
			expectedType:     "RFQ_EMAIL",
			expectedCustomer: "CGC",
			expectedOffer:    "106",
			expectedStage:    "RFQ",
			expectedProduct:  "TRANSMITTER",
		},
		{
			name:             "RFQ Document",
			fileName:         "RFQ_technical_specs.pdf",
			folderPath:       "107 HZP ANALYZER/RFQ",
			expectedType:     "RFQ_DOCUMENT",
			expectedCustomer: "HZP",
			expectedOffer:    "107",
			expectedStage:    "RFQ",
			expectedProduct:  "ANALYZER",
		},
		{
			name:             "Costing Sheet",
			fileName:         "Costing Sheet v2.xlsx",
			folderPath:       "108 GSC FLOWMETER/OFFER",
			expectedType:     "COSTING_SHEET",
			expectedCustomer: "GSC",
			expectedOffer:    "108",
			expectedStage:    "OFFER",
			expectedProduct:  "FLOWMETER",
		},
		{
			name:             "Commercial Offer",
			fileName:         "Commercial Offer Rev 3.pdf",
			folderPath:       "109 NPC PIT/OFFER",
			expectedType:     "COMMERCIAL_OFFER",
			expectedCustomer: "NPC",
			expectedOffer:    "109",
			expectedStage:    "OFFER",
			expectedProduct:  "PIT",
		},
		{
			name:             "Delivery Note",
			fileName:         "DN_12345.pdf",
			folderPath:       "110 NGA TIT/EXECUTION",
			expectedType:     "DELIVERY_NOTE",
			expectedCustomer: "NGA",
			expectedOffer:    "110",
			expectedStage:    "EXECUTION",
			expectedProduct:  "TIT",
		},
		{
			name:             "Shipping Document",
			fileName:         "Packing List.pdf",
			folderPath:       "111 VERTEX FT/EXECUTION/Shipment",
			expectedType:     "SHIPPING_DOC",
			expectedCustomer: "VERTEX",
			expectedOffer:    "111",
			expectedStage:    "EXECUTION",
			expectedProduct:  "FT",
		},
		{
			name:             "Technical Document",
			fileName:         "Technical Datasheet.pdf",
			folderPath:       "112 DPC LT/RFQ",
			expectedType:     "TECHNICAL_DOC",
			expectedCustomer: "DPC",
			expectedOffer:    "112",
			expectedStage:    "RFQ",
			expectedProduct:  "LT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock file info
			info := &mockFileInfo{
				name:    tt.fileName,
				size:    1024,
				modTime: time.Now(),
			}

			// Classify document
			doc := service.classifyDocument(
				filepath.Join("C:\\test", tt.folderPath, tt.fileName),
				info,
				"C:\\test",
			)

			// Verify document type
			if doc.DocumentType != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, doc.DocumentType)
			}

			// Verify customer name
			if tt.expectedCustomer != "" && doc.CustomerName != tt.expectedCustomer {
				t.Errorf("Expected customer %s, got %s", tt.expectedCustomer, doc.CustomerName)
			}

			// Verify offer number
			if tt.expectedOffer != "" && doc.OfferNumber != tt.expectedOffer {
				t.Errorf("Expected offer %s, got %s", tt.expectedOffer, doc.OfferNumber)
			}

			// Verify stage
			if tt.expectedStage != "" && doc.Stage != tt.expectedStage {
				t.Errorf("Expected stage %s, got %s", tt.expectedStage, doc.Stage)
			}

			// Verify product type
			if tt.expectedProduct != "" && doc.ProductType != tt.expectedProduct {
				t.Errorf("Expected product %s, got %s", tt.expectedProduct, doc.ProductType)
			}
		})
	}
}

func TestExtractOfferNumber(t *testing.T) {
	service := NewFilesystemDocumentClassifierService()

	tests := []struct {
		folderPath string
		expected   string
	}{
		{"101 GSC AIT", "101"},
		{"102 NPC FIT", "102"},
		{"subfolder/103 NGA LIT/RFQ", "103"},
		{"no-number-here", ""},
		{"999 DPC VALVE/OFFER/Documents", "999"},
	}

	for _, tt := range tests {
		result := service.extractOfferNumber(tt.folderPath)
		if result != tt.expected {
			t.Errorf("extractOfferNumber(%s) = %s, want %s", tt.folderPath, result, tt.expected)
		}
	}
}

func TestExtractCustomerName(t *testing.T) {
	service := NewFilesystemDocumentClassifierService()

	tests := []struct {
		folderPath string
		fileName   string
		expected   string
	}{
		{"101 GSC AIT", "invoice.pdf", "GSC"},
		{"102 NPC FIT", "quote.pdf", "NPC"},
		{"103 NGA LIT", "po.pdf", "NGA"},
		{"104 VERTEX SP", "rfq.pdf", "VERTEX"},
		{"105 DPC VALVE", "offer.pdf", "DPC"},
		{"106 CGC TRANSMITTER", "costing.xlsx", "CGC"},
		{"107 HZP ANALYZER", "dn.pdf", "HZP"},
		{"no-customer-folder", "GSC_invoice.pdf", "GSC"},
		{"generic-folder", "NPC_PO.pdf", "NPC"},
		{"Documents/AsymmFlow Exports/AquaPure_Energy_Gulf_WLL/RFQ", "Quotation_123.pdf", "AquaPure Energy Gulf WLL"},
		{`C:\Users\Demo\Documents\AsymmFlow Exports\The_National_Petroleum_Company\Delivery_Notes`, "DN.pdf", "The National Petroleum Company"},
		{"Documents/AsymmFlow Exports/Suppliers/Rhine_Instruments/Orders", "PO.pdf", ""},
	}

	for _, tt := range tests {
		result := service.extractCustomerName(tt.folderPath, tt.fileName)
		if result != tt.expected {
			t.Errorf("extractCustomerName(%s, %s) = %s, want %s",
				tt.folderPath, tt.fileName, result, tt.expected)
		}
	}
}

func TestExtractProductType(t *testing.T) {
	service := NewFilesystemDocumentClassifierService()

	tests := []struct {
		folderPath string
		fileName   string
		expected   string
	}{
		{"101 GSC AIT", "doc.pdf", "AIT"},
		{"102 NPC FIT", "doc.pdf", "FIT"},
		{"103 NGA LIT", "doc.pdf", "LIT"},
		{"104 VERTEX SP", "doc.pdf", "SP"},
		{"105 DPC VALVE", "doc.pdf", "VALVE"},
		{"106 CGC TRANSMITTER", "doc.pdf", "TRANSMITTER"},
		{"107 HZP ANALYZER", "doc.pdf", "ANALYZER"},
		{"108 GSC FLOWMETER", "doc.pdf", "FLOWMETER"},
		{"generic-folder", "AIT_datasheet.pdf", "AIT"},
		{"generic-folder", "flowmeter_specs.pdf", "FLOWMETER"},
	}

	for _, tt := range tests {
		result := service.extractProductType(tt.folderPath, tt.fileName)
		if result != tt.expected {
			t.Errorf("extractProductType(%s, %s) = %s, want %s",
				tt.folderPath, tt.fileName, result, tt.expected)
		}
	}
}

func TestExtractStage(t *testing.T) {
	service := NewFilesystemDocumentClassifierService()

	tests := []struct {
		folderPath string
		expected   string
	}{
		{"101 GSC AIT/RFQ", "RFQ"},
		{"102 NPC FIT/OFFER", "OFFER"},
		{"103 NGA LIT/EXECUTION", "EXECUTION"},
		{"104 VERTEX SP/QUOTATION", "OFFER"},
		{"105 DPC VALVE/ORDER", "EXECUTION"},
		{"106 CGC TRANSMITTER/SHIPMENT", "EXECUTION"},
		{"no-stage-folder", ""},
	}

	for _, tt := range tests {
		result := service.extractStage(tt.folderPath)
		if result != tt.expected {
			t.Errorf("extractStage(%s) = %s, want %s", tt.folderPath, result, tt.expected)
		}
	}
}

func TestDocumentClassifierBusinessTypes(t *testing.T) {
	classifier := NewDocumentClassifier()

	tests := []struct {
		name     string
		filename string
		text     string
		expected string
	}{
		{
			name:     "Bank Statement",
			filename: "March_2026_Bank_Statement.pdf",
			text:     "BANK STATEMENT\nAccount Number: 123456\nOpening Balance 1,000.000\nDebit Credit Balance\nClosing Balance 950.000",
			expected: "BankStatement",
		},
		{
			name:     "Supplier Invoice",
			filename: "oxan_invoice_2811.pdf",
			text:     "TAX INVOICE\nVendor: Oxan Analytics\nBill To: Acme Instrumentation WLL\nInvoice Number INV-2811\nPayment Terms Net 30",
			expected: "SupplierInvoice",
		},
		{
			name:     "Delivery Note",
			filename: "DN_12345.pdf",
			text:     "Delivery Note\nDN Number: 12345\nDispatch note for supplied items",
			expected: "DeliveryNote",
		},
		{
			name:     "Purchase Order",
			filename: "PO_81_1234.pdf",
			text:     "Purchase Order\nPO Number: 81-1234\nDelivery Date: 31/03/2026",
			expected: "PurchaseOrder",
		},
		{
			name:     "Report",
			filename: "Q1_2026_management_report.pdf",
			text:     "Management Report\nExecutive Summary\nKPI analysis report for Q1 2026",
			expected: "Report",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.text, tt.filename)
			if result == nil {
				t.Fatalf("expected classification result for %s", tt.name)
			}
			if result.DocumentType != tt.expected {
				t.Fatalf("expected %s, got %s (%s)", tt.expected, result.DocumentType, result.Explanation)
			}
		})
	}
}

func TestDetectDocumentTypeFromText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "Bank Statement Text",
			text:     "Account Statement\nOpening Balance 500.000\nDebit Credit Balance\nClosing Balance 750.000",
			expected: "bank_statement",
		},
		{
			name:     "Supplier Invoice Text",
			text:     "Invoice\nVendor invoice\nBill To: Acme Instrumentation WLL\nInvoice Number 12345",
			expected: "supplier_invoice",
		},
		{
			name:     "Delivery Note Text",
			text:     "Delivery Note\nPacking List\nConsignment Note",
			expected: "delivery_note",
		},
		{
			name:     "Report Text",
			text:     "Management Report\nExecutive Summary\nPerformance report for March 2026",
			expected: "report",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectDocumentTypeFromTextLegacy(tt.text); got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() any           { return nil }
