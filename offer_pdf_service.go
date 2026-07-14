package main

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// GenerateOfferPDF renders a saved offer through the same quotation PDF
// pipeline used by the costing sheet export, so page order and styling cannot drift.
func (a *App) GenerateOfferPDF(offerID string) (string, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return "", err
	}
	if strings.TrimSpace(offerID) == "" {
		return "", fmt.Errorf("offer ID is required")
	}
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	log.Printf("📄 Generating canonical offer PDF: offerID=%s", offerID)

	var offer Offer
	if err := a.db.Preload("Items").First(&offer, "id = ?", offerID).Error; err != nil {
		return "", fmt.Errorf("failed to fetch offer: %w", err)
	}

	if len(offer.Items) == 0 {
		var rawItemCount int64
		a.db.Model(&OfferItem{}).Where("offer_id = ?", offerID).Count(&rawItemCount)
		if rawItemCount > 0 {
			log.Printf("⚠️ Preload failed for offer %s (raw count: %d). Loading items manually.", offerID, rawItemCount)
			var items []OfferItem
			a.db.Where("offer_id = ?", offerID).Find(&items)
			offer.Items = items
		}
		if len(offer.Items) == 0 {
			return "", fmt.Errorf("this offer has no line items; add items via the costing sheet before generating a PDF")
		}
	}

	if offer.DiscountPercent < 0 || offer.DiscountPercent > 100 {
		return "", fmt.Errorf("invalid discount percentage: %.2f (must be 0-100)", offer.DiscountPercent)
	}

	var customer CustomerMaster
	if strings.TrimSpace(offer.CustomerID) != "" {
		if err := a.db.First(&customer, "id = ?", offer.CustomerID).Error; err != nil {
			log.Printf("⚠️ Could not fetch customer details for offer %s: %v", offer.OfferNumber, err)
		}
	}

	var customerContact CustomerContact
	if strings.TrimSpace(offer.CustomerID) != "" {
		a.db.Where("customer_id = ? AND is_primary_contact = ?", offer.CustomerID, true).First(&customerContact)
		if customerContact.ID == "" {
			a.db.Where("customer_id = ?", offer.CustomerID).First(&customerContact)
		}
	}

	var linkedOpportunity Opportunity
	if err := a.db.Select("folder_number, title, eh_ref").Where("offer_id = ?", offer.ID).First(&linkedOpportunity).Error; err == nil {
		offer.FolderNumber = linkedOpportunity.FolderNumber
		offer.ProjectName = linkedOpportunity.Title
		if strings.TrimSpace(offer.CustomerReference) == "" {
			offer.CustomerReference = linkedOpportunity.EHRef
		}
	}

	exportData := buildCostingExportDataFromOffer(offer, customer, customerContact)
	// I-25: bundle the offer's technical datasheets onto the generated PDF.
	if scope := normaliseCostingAttachmentScope(offer.AttachmentScopeID); scope != "" {
		attachments, err := a.listCostingSheetAttachmentsByScope(scope)
		if err != nil {
			return "", err
		}
		exportData.AttachmentScopeID = scope
		exportData.Attachments = attachments
	}
	filePath, err := a.exportCostingToPDF(exportData, "Quotation")
	if err != nil {
		return "", err
	}

	log.Printf("✅ Canonical offer PDF generated: %s", filePath)
	return filePath, nil
}

func buildCostingExportDataFromOffer(offer Offer, customer CustomerMaster, customerContact CustomerContact) CostingExportData {
	quotationDate := offer.QuotationDate
	if quotationDate.IsZero() {
		quotationDate = time.Now()
	}

	subtotal := 0.0
	lineItems := make([]CostingExportLineItem, 0, len(offer.Items))
	for i, item := range offer.Items {
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}
		unitPrice := item.UnitPrice
		lineTotal := item.TotalPrice
		if lineTotal <= 0 {
			lineTotal = unitPrice * qty
		}
		if unitPrice <= 0 && qty > 0 {
			unitPrice = lineTotal / qty
		}
		subtotal += lineTotal

		equipment := strings.TrimSpace(item.Equipment)
		if equipment == "" {
			equipment = strings.TrimSpace(item.Description)
		}
		model := firstNonEmptyString(strings.TrimSpace(item.Model), strings.TrimSpace(item.ProductCode))

		lineItems = append(lineItems, CostingExportLineItem{
			SlNo:                firstPositiveInt(item.LineNumber, i+1),
			Equipment:           equipment,
			Model:               model,
			LongCode:            item.LongCode,
			Specification:       item.Specification,
			DetailedDescription: item.DetailedDescription,
			Currency:            firstNonEmptyString(item.Currency, "BHD"),
			Quantity:            int(math.Round(qty)),
			FOB:                 item.FOB,
			Freight:             item.Freight,
			TotalCost:           item.TotalCost,
			MarginPercent:       item.MarginPercent,
			MarkupPercent:       item.MarginPercent,
			SuggestedPrice:      unitPrice,
			TotalPrice:          lineTotal,
			ExchangeRate:        item.ExchangeRate,
			FobBHD:              item.FobBHD,
			FreightBHD:          item.FreightBHD,
			Insurance:           item.Insurance,
			CustomsPercent:      item.CustomsPercent,
			CustomsBHD:          item.CustomsBHD,
			HandlingPercent:     item.HandlingPercent,
			HandlingBHD:         item.HandlingBHD,
			FinancePercent:      item.FinancePercent,
			FinanceBHD:          item.FinanceBHD,
			OtherCosts:          item.OtherCosts,
			UserPrice:           item.UserPrice,
			UserPriceSet:        item.UserPriceSet,
		})
	}

	discount := 0.0
	if offer.DiscountPercent > 0 {
		discount = subtotal * (offer.DiscountPercent / 100)
	}
	netAmount := subtotal - discount
	vatRate := offer.VatRate
	vat := netAmount * (vatRate / 100)
	grandTotal := netAmount + vat

	terms := strings.TrimSpace(offer.TermsAndConditions)
	if terms == "" {
		terms = defaultOfferTermsAndConditions(normalizeDivisionName(offer.Division), vatRate)
	}

	customerName := firstNonEmptyString(strings.TrimSpace(offer.CustomerName), strings.TrimSpace(customer.BusinessName), strings.TrimSpace(offer.AttentionCompany))
	contactPerson := firstNonEmptyString(strings.TrimSpace(offer.AttentionPerson), strings.TrimSpace(customerContact.ContactName))
	subject := firstNonEmptyString(strings.TrimSpace(offer.Subject), strings.TrimSpace(offer.ProjectName), strings.TrimSpace(offer.CustomerReference), strings.TrimSpace(offer.FolderNumber), customerName)
	body := firstNonEmptyString(
		strings.TrimSpace(offer.Body),
		"We thank you for the opportunity and are pleased to submit our techno-commercial offer for your review. Please find our pricing and scope below.",
	)
	quoteType := firstNonEmptyString(strings.TrimSpace(offer.QuoteType), "Quotation")

	return CostingExportData{
		Division: normalizeDivisionName(offer.Division),
		Date:     quotationDate.Format("2006-01-02"),
		// Canonicalise the issuer to the matched signature block's DisplayName
		// so an alias (e.g. "Sam") prints as its full name ("Sam Rivera") both
		// in the info table and in the signature block rendered by the shared
		// quotation pipeline. Unmatched names pass through trimmed.
		PreparedBy:         offerIssuerDisplayName(offer.IssuedBy),
		CustomerName:       customerName,
		ContactPerson:      contactPerson,
		RfqReference:       strings.TrimSpace(offer.CustomerReference),
		FolderNumber:       strings.TrimSpace(offer.FolderNumber),
		CostingId:          offerNumberWithRevision(offer.OfferNumber, offer.RevisionNumber),
		Subject:            subject,
		EstDelivery:        strings.TrimSpace(offer.DeliveryWeeks),
		DeliveryTerms:      strings.TrimSpace(offer.DeliveryTerms),
		PaymentTerms:       strings.TrimSpace(offer.PaymentTerms),
		CountryOfOrigin:    strings.TrimSpace(offer.CountryOfOrigin),
		CocCoo:             strings.TrimSpace(offer.CocCoo),
		TestCertificate:    strings.TrimSpace(offer.TestCertificate),
		Installation:       strings.TrimSpace(offer.Installation),
		Commissioning:      strings.TrimSpace(offer.Commissioning),
		Testing:            strings.TrimSpace(offer.Testing),
		QuoteType:          quoteType,
		VatRate:            vatRate,
		PlaceOfSupply:      "Kingdom of Bahrain",
		TaxCategory:        "Standard",
		CustomerTRN:        strings.TrimSpace(customer.TRN),
		Body:               body,
		LineItems:          lineItems,
		Subtotal:           subtotal,
		Discount:           discount,
		NetAmount:          netAmount,
		VAT:                vat,
		GrandTotal:         grandTotal,
		TotalCost:          sumOfferItemCost(offer.Items),
		Profit:             grandTotal - sumOfferItemCost(offer.Items),
		ProfitPercent:      percentageOrZero(grandTotal-sumOfferItemCost(offer.Items), grandTotal),
		ProjectName:        strings.TrimSpace(offer.ProjectName),
		TermsAndConditions: terms,
	}
}

func defaultOfferTermsAndConditions(division string, vatRate float64) string {
	// Wave 11 B1: the trading entity named in the T&C comes from the overlay so a
	// deployment's own name appears in generated quotations (no source edit).
	// Wave 12.5: the entity now resolves per the OFFER'S OWN division rather than
	// always the company-level/default name, so a non-default-division offer's
	// T&C correctly names that division (e.g. "Beacon Controls WLL").
	company := activeOverlay.DivisionDocumentDisplayName(division)
	return fmt.Sprintf(`1. QUOTATION VALIDITY
This quotation is valid for thirty (30) days from the date of issue.

2. PRICES
All prices are in Bahraini Dinars (BHD) unless otherwise stated. Prices are exclusive of VAT (%.0f%%) which will be added to the invoice.

3. PAYMENT TERMS
As per the payment terms specified in this quotation. Late payments may incur interest charges.

4. DELIVERY
Delivery times are estimates and subject to manufacturer's confirmation. %s shall not be liable for delays beyond our control.

5. WARRANTY
All products carry the manufacturer's standard warranty. Extended warranty options are available upon request.

6. INSTALLATION & COMMISSIONING
Installation and commissioning services are available at additional cost unless included in the quotation.

7. FORCE MAJEURE
%s shall not be liable for failure to perform due to causes beyond reasonable control.

8. GOVERNING LAW
This quotation is governed by the laws of the Kingdom of Bahrain.`, vatRate, company, company)
}

func firstPositiveInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func sumOfferItemCost(items []OfferItem) float64 {
	total := 0.0
	for _, item := range items {
		total += item.TotalCost
	}
	return total
}

func percentageOrZero(part, total float64) float64 {
	if total == 0 {
		return 0
	}
	return (part / total) * 100
}
