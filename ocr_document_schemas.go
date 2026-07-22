package main

import "ph_holdings_app/pkg/ocr/mistralocr"

// Document AI (structured extraction) schemas passed to the Mistral OCR endpoint's
// document_annotation_format field. These replace the regex layer (extractFieldsFromTextLegacy)
// on the online path — the model reshapes its own output to this schema instead of a client-side
// regex scrape. Fields mirror what extractFieldsFromTextLegacy scrapes today (see that function's
// pattern map in ocr_service_simple.go) so downstream consumers (bank-statement merge, RFQ/invoice
// screens) see the same field names regardless of which engine produced them.
//
// The regex layer remains config-not-constant's "degraded mode": it runs only on the offline/
// tesseract fallback path (see ocrWithLocalEngine) and for local-parse formats (Excel/MSG/EML/
// DOCX/RTF) that never touch OCR at all.

func stringField(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

// schemaForDocType returns the Document AI JSON schema for a normalized document type hint.
// docType values match the hints already used by SimpleOCRService.ProcessDocument / the legacy
// regex classifier (detectDocumentTypeFromTextLegacy): "invoice", "supplier_invoice", "rfq",
// "quotation", "purchase_order", "bank_statement", "delivery_note", or anything else → generic.
func schemaForDocType(docType string) *mistralocr.DocumentSchema {
	switch normalizeSchemaDocType(docType) {
	case "invoice":
		return invoiceDocumentSchema()
	case "rfq":
		return rfqDocumentSchema()
	case "purchase_order":
		return purchaseOrderDocumentSchema()
	case "bank_statement":
		return bankStatementDocumentSchema()
	default:
		return genericDocumentSchema()
	}
}

func normalizeSchemaDocType(docType string) string {
	switch docType {
	case "invoice", "supplier_invoice":
		return "invoice"
	case "rfq", "quotation":
		return "rfq"
	case "purchase_order", "po":
		return "purchase_order"
	case "bank_statement":
		return "bank_statement"
	default:
		return "generic"
	}
}

func invoiceDocumentSchema() *mistralocr.DocumentSchema {
	return &mistralocr.DocumentSchema{
		Name: "invoice_extraction",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"invoice_number": stringField("Invoice / tax invoice number as printed on the document"),
				"invoice_date":   stringField("Invoice date, ISO 8601 (YYYY-MM-DD) if determinable"),
				"due_date":       stringField("Payment due date, ISO 8601 (YYYY-MM-DD) if determinable"),
				"po_number":      stringField("Referenced purchase order number, if present"),
				"customer_name":  stringField("Bill-to / customer name"),
				"supplier_name":  stringField("Issuing supplier / vendor name"),
				"subtotal":       stringField("Subtotal before tax, as printed (include currency symbol if shown)"),
				"vat":            stringField("VAT/tax amount, as printed"),
				"total":          stringField("Grand total / amount due, as printed"),
				"currency":       stringField("Currency code or symbol (e.g. BHD, USD, AED)"),
			},
		},
		Strict: false,
	}
}

func rfqDocumentSchema() *mistralocr.DocumentSchema {
	return &mistralocr.DocumentSchema{
		Name: "rfq_extraction",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rfq_number":       stringField("RFQ / enquiry / tender reference number"),
				"rfq_reference":    stringField("Customer's own reference, if separate from the RFQ number"),
				"quotation_number": stringField("Quotation/offer number, if this is a quotation rather than an inbound RFQ"),
				"project":          stringField("Project name or subject line"),
				"delivery_date":    stringField("Required/expected delivery date, ISO 8601 if determinable"),
				"validity":         stringField("Offer validity period as printed (e.g. '30 days')"),
				"bid_deadline":     stringField("Submission/closing deadline, ISO 8601 if determinable"),
				"customer_name":    stringField("Requesting customer / client name"),
				"contact_person":   stringField("Named contact person"),
				"contact_email":    stringField("Contact email address"),
				"contact_phone":    stringField("Contact phone number"),
			},
		},
		Strict: false,
	}
}

func purchaseOrderDocumentSchema() *mistralocr.DocumentSchema {
	return &mistralocr.DocumentSchema{
		Name: "purchase_order_extraction",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"po_number":     stringField("Purchase order number"),
				"po_date":       stringField("PO issue date, ISO 8601 if determinable"),
				"customer_name": stringField("Buyer / customer name"),
				"supplier_name": stringField("Supplier / vendor name"),
				"delivery_date": stringField("Required delivery date, ISO 8601 if determinable"),
				"part_number":   stringField("Primary part/item/model number referenced"),
				"quantity":      stringField("Quantity ordered, as printed"),
				"unit_price":    stringField("Unit price, as printed"),
				"total":         stringField("PO total value, as printed"),
				"currency":      stringField("Currency code or symbol"),
			},
		},
		Strict: false,
	}
}

func bankStatementDocumentSchema() *mistralocr.DocumentSchema {
	return &mistralocr.DocumentSchema{
		Name: "bank_statement_extraction",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"account_number":  stringField("Bank account number"),
				"iban":            stringField("IBAN, if printed"),
				"currency":        stringField("Statement currency"),
				"period_start":    stringField("Statement period start date, ISO 8601 if determinable"),
				"period_end":      stringField("Statement period end date, ISO 8601 if determinable"),
				"opening_balance": stringField("Opening balance, as printed"),
				"closing_balance": stringField("Closing balance, as printed"),
			},
		},
		Strict: false,
	}
}

func genericDocumentSchema() *mistralocr.DocumentSchema {
	return &mistralocr.DocumentSchema{
		Name: "generic_document_extraction",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"document_type_guess": stringField("Best guess at the document's type (invoice, rfq, purchase_order, delivery_note, bank_statement, report, other)"),
				"reference_number":    stringField("Any prominent reference/document number"),
				"date":                stringField("Any prominent date, ISO 8601 if determinable"),
				"customer_name":       stringField("Customer/client/recipient name, if present"),
				"supplier_name":       stringField("Supplier/vendor/sender name, if present"),
				"total":               stringField("Any prominent total/amount, as printed"),
				"currency":            stringField("Currency code or symbol, if present"),
			},
		},
		Strict: false,
	}
}
