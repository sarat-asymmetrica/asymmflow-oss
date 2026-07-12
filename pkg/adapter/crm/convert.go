// Package crm converts CRM GORM models to and from generated Proto messages.
package crm

import (
	"strings"

	"ph_holdings_app/pkg/adapter"
	gormcrm "ph_holdings_app/pkg/crm"
	commonproto "ph_holdings_app/schemas/go/common"
	protocrm "ph_holdings_app/schemas/go/crm"

	capnp "capnproto.org/go/capnp/v3"
)

func newMessage() (*capnp.Message, *capnp.Segment, error) {
	return capnp.NewMessage(capnp.SingleSegment(nil))
}

func setBase(seg *capnp.Segment, setter func(commonproto.Base) error, base gormcrm.Base) error {
	pb, err := adapter.BaseToProto(seg, base)
	if err != nil {
		return err
	}
	return setter(pb)
}

func recordStatus(status string) commonproto.RecordStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "inactive":
		return commonproto.RecordStatus_inactive
	case "archived":
		return commonproto.RecordStatus_archived
	case "deleted":
		return commonproto.RecordStatus_deleted
	case "draft":
		return commonproto.RecordStatus_draft
	default:
		return commonproto.RecordStatus_active
	}
}

func grade(value string) protocrm.CustomerGrade {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "A":
		return protocrm.CustomerGrade_a
	case "B":
		return protocrm.CustomerGrade_b
	case "C":
		return protocrm.CustomerGrade_c
	case "D":
		return protocrm.CustomerGrade_d
	default:
		return protocrm.CustomerGrade_unknown
	}
}

func gradeText(value protocrm.CustomerGrade) string {
	switch value {
	case protocrm.CustomerGrade_a:
		return "A"
	case protocrm.CustomerGrade_b:
		return "B"
	case protocrm.CustomerGrade_c:
		return "C"
	case protocrm.CustomerGrade_d:
		return "D"
	default:
		return ""
	}
}

func documentStatus(status string) commonproto.DocumentStatus {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(status), " ", "")) {
	case "prepared":
		return commonproto.DocumentStatus_prepared
	case "sent":
		return commonproto.DocumentStatus_sent
	case "acknowledged":
		return commonproto.DocumentStatus_acknowledged
	case "issued":
		return commonproto.DocumentStatus_issued
	case "partiallypaid", "partiallyreceived":
		return commonproto.DocumentStatus_partiallyPaid
	case "paid", "delivered", "completed", "received", "approved", "converted", "passed":
		return commonproto.DocumentStatus_paid
	case "overdue":
		return commonproto.DocumentStatus_overdue
	case "void", "voided":
		return commonproto.DocumentStatus_voided
	case "cancelled", "canceled", "rejected", "failed":
		return commonproto.DocumentStatus_cancelled
	case "closed":
		return commonproto.DocumentStatus_closed
	default:
		return commonproto.DocumentStatus_draft
	}
}

func statusText(status commonproto.DocumentStatus) string {
	switch status {
	case commonproto.DocumentStatus_prepared:
		return "Prepared"
	case commonproto.DocumentStatus_sent:
		return "Sent"
	case commonproto.DocumentStatus_acknowledged:
		return "Acknowledged"
	case commonproto.DocumentStatus_partiallyPaid:
		return "PartiallyPaid"
	case commonproto.DocumentStatus_paid:
		return "Completed"
	case commonproto.DocumentStatus_cancelled:
		return "Cancelled"
	case commonproto.DocumentStatus_closed:
		return "Closed"
	default:
		return "Draft"
	}
}

func opportunityStage(stage string) protocrm.OpportunityStage {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "lead", "rfq":
		return protocrm.OpportunityStage_lead
	case "qualified":
		return protocrm.OpportunityStage_qualified
	case "proposal":
		return protocrm.OpportunityStage_proposal
	case "quoted":
		return protocrm.OpportunityStage_quoted
	case "negotiation":
		return protocrm.OpportunityStage_negotiation
	case "won":
		return protocrm.OpportunityStage_won
	case "lost":
		return protocrm.OpportunityStage_lost
	case "expired":
		return protocrm.OpportunityStage_expired
	default:
		return protocrm.OpportunityStage_lead
	}
}

func stageText(stage protocrm.OpportunityStage) string {
	switch stage {
	case protocrm.OpportunityStage_qualified:
		return "Qualified"
	case protocrm.OpportunityStage_proposal:
		return "Proposal"
	case protocrm.OpportunityStage_quoted:
		return "Quoted"
	case protocrm.OpportunityStage_negotiation:
		return "Negotiation"
	case protocrm.OpportunityStage_won:
		return "Won"
	case protocrm.OpportunityStage_lost:
		return "Lost"
	case protocrm.OpportunityStage_expired:
		return "Expired"
	default:
		return "Lead"
	}
}

func currencyCode(currency string) commonproto.CurrencyCode {
	return commonproto.CurrencyCodeFromString(strings.ToLower(strings.TrimSpace(currency)))
}

func currencyText(currency commonproto.CurrencyCode) string {
	return strings.ToUpper(currency.String())
}

func CustomerMasterToProto(m gormcrm.CustomerMaster) (*protocrm.CustomerMaster, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewCustomerMaster(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetCustomerId(m.CustomerID), p.SetCustomerCode(m.CustomerCode), p.SetCustomerType(m.CustomerType), p.SetBusinessName(m.BusinessName), p.SetShortCode(m.ShortCode), p.SetTradingName(m.TradingName), p.SetCrNumber(m.CRNumber), p.SetPrimaryPhone(m.PrimaryPhone), p.SetPrimaryEmail(m.PrimaryEmail), p.SetWebsite(m.Website), p.SetAddressLine1(m.AddressLine1), p.SetCity(m.City), p.SetCountry(m.Country), p.SetTrn(m.TRN), p.SetMobileNumber(m.MobileNumber), p.SetIndustry(m.Industry), p.SetLastOrderDate(adapter.TimePtrToText(m.LastOrderDate))} {
		if err != nil {
			return nil, err
		}
	}
	p.SetStatus(recordStatus(m.Status))
	p.SetRelationYears(int64(m.RelationYears))
	p.SetPaymentGrade(grade(m.PaymentGrade))
	p.SetCustomerGrade(grade(m.CustomerGrade))
	p.SetPaymentTermsDays(int64(m.PaymentTermsDays))
	p.SetAvgPaymentDays(m.AvgPaymentDays)
	p.SetDisputeCount(int64(m.DisputeCount))
	p.SetTotalOrdersValue(m.TotalOrdersValue)
	p.SetTotalOrdersCount(int64(m.TotalOrdersCount))
	p.SetAvgOrderValue(m.AvgOrderValue)
	p.SetOutstandingBhd(m.OutstandingBHD)
	p.SetOverdueDays(int64(m.OverdueDays))
	p.SetCreditLimitBhd(m.CreditLimitBHD)
	p.SetIsCreditBlocked(m.IsCreditBlocked)
	p.SetRequiresPrepayment(m.RequiresPrepayment)
	p.SetHasAbbCompetition(m.HasABBCompetition)
	p.SetIsEmergencyOnly(m.IsEmergencyOnly)
	return &p, nil
}

func CustomerMasterFromProto(p protocrm.CustomerMaster) (gormcrm.CustomerMaster, error) {
	base, err := p.Base()
	if err != nil {
		return gormcrm.CustomerMaster{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormcrm.CustomerMaster{}, err
	}
	m := gormcrm.CustomerMaster{Base: sharedBase}
	m.CustomerID, _ = p.CustomerId()
	m.CustomerCode, _ = p.CustomerCode()
	m.BusinessName, _ = p.BusinessName()
	m.PaymentGrade = gradeText(p.PaymentGrade())
	m.CustomerGrade = gradeText(p.CustomerGrade())
	m.OutstandingBHD = p.OutstandingBhd()
	m.CreditLimitBHD = p.CreditLimitBhd()
	m.IsCreditBlocked = p.IsCreditBlocked()
	m.MobileNumber, _ = p.MobileNumber()
	return m, nil
}

func CustomerContactToProto(m gormcrm.CustomerContact) (*protocrm.CustomerContact, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewCustomerContact(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetCustomerId(m.CustomerID), p.SetContactName(m.ContactName), p.SetJobTitle(m.JobTitle), p.SetEmail(m.Email), p.SetPhone(m.Phone), p.SetAddress(m.Address)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetIsPrimaryContact(m.IsPrimaryContact)
	return &p, nil
}

func CustomerContactFromProto(p protocrm.CustomerContact) (gormcrm.CustomerContact, error) {
	m := gormcrm.CustomerContact{}
	m.CustomerID, _ = p.CustomerId()
	m.ContactName, _ = p.ContactName()
	m.Email, _ = p.Email()
	m.Phone, _ = p.Phone()
	m.IsPrimaryContact = p.IsPrimaryContact()
	return m, nil
}

func SupplierMasterToProto(m gormcrm.SupplierMaster) (*protocrm.SupplierMaster, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewSupplierMaster(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetSupplierCode(m.SupplierCode), p.SetSupplierName(m.SupplierName), p.SetCountry(m.Country), p.SetTaxId(m.TaxID), p.SetSupplierType(m.SupplierType), p.SetBrandsHandled(m.BrandsHandled), p.SetProductTypes(m.ProductTypes), p.SetPrimaryContact(m.PrimaryContact), p.SetEmail(m.Email), p.SetPhone(m.Phone), p.SetAddress(m.Address), p.SetBankName(m.BankName), p.SetAccountNumber(m.AccountNumber), p.SetIban(m.IBAN), p.SetSwiftCode(m.SwiftCode), p.SetPaymentTerms(m.PaymentTerms), p.SetNotes(m.Notes)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetLeadTimeDays(int64(m.LeadTimeDays))
	p.SetRating(int64(m.Rating))
	return &p, nil
}

func SupplierMasterFromProto(p protocrm.SupplierMaster) (gormcrm.SupplierMaster, error) {
	m := gormcrm.SupplierMaster{}
	m.SupplierCode, _ = p.SupplierCode()
	m.SupplierName, _ = p.SupplierName()
	m.Country, _ = p.Country()
	m.PaymentTerms, _ = p.PaymentTerms()
	return m, nil
}

func SupplierContactToProto(m gormcrm.SupplierContact) (*protocrm.SupplierContact, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewSupplierContact(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetSupplierId(m.SupplierID), p.SetContactName(m.ContactName), p.SetJobTitle(m.JobTitle), p.SetEmail(m.Email), p.SetPhone(m.Phone), p.SetAddress(m.Address)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetIsPrimaryContact(m.IsPrimaryContact)
	return &p, nil
}

func SupplierContactFromProto(p protocrm.SupplierContact) (gormcrm.SupplierContact, error) {
	m := gormcrm.SupplierContact{}
	m.SupplierID, _ = p.SupplierId()
	m.ContactName, _ = p.ContactName()
	m.Email, _ = p.Email()
	return m, nil
}

func ProductMasterToProto(m gormcrm.ProductMaster) (*protocrm.ProductMaster, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewProductMaster(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetProductCode(m.ProductCode), p.SetProductName(m.ProductName), p.SetProductCategory(m.ProductCategory), p.SetSupplierId(m.SupplierID), p.SetSupplierCode(m.SupplierCode), p.SetDescription(m.Description), p.SetSku(m.SKU), p.SetPartNumber(m.PartNumber), p.SetHsCode(m.HSCode), p.SetUnitOfMeasure(m.UnitOfMeasure), p.SetDatasheetUrl(m.DatasheetURL), p.SetSpecifications(m.Specifications)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetStandardCostBhd(m.StandardCostBHD)
	p.SetStandardPriceBhd(m.StandardPriceBHD)
	p.SetIsActive(m.IsActive)
	p.SetStockQuantity(int64(m.StockQuantity))
	p.SetRequiresSerialTracking(m.RequiresSerialTracking)
	return &p, nil
}

func ProductMasterFromProto(p protocrm.ProductMaster) (gormcrm.ProductMaster, error) {
	m := gormcrm.ProductMaster{}
	m.ProductCode, _ = p.ProductCode()
	m.ProductName, _ = p.ProductName()
	m.StandardCostBHD = p.StandardCostBhd()
	m.StandardPriceBHD = p.StandardPriceBhd()
	return m, nil
}

func OfferToProto(m gormcrm.Offer) (*protocrm.Offer, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewOffer(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetOfferNumber(m.OfferNumber), p.SetRfqId(m.RFQID), p.SetCustomerId(m.CustomerID), p.SetCustomerName(m.CustomerName), p.SetQuotationDate(adapter.TimeToText(m.QuotationDate)), p.SetValidityDate(adapter.TimeToText(m.ValidityDate)), p.SetLostReason(m.LostReason), p.SetPaymentTerms(m.PaymentTerms), p.SetDeliveryTerms(m.DeliveryTerms), p.SetDeliveryWeeks(m.DeliveryWeeks), p.SetCountryOfOrigin(m.CountryOfOrigin), p.SetIssuedBy(m.IssuedBy), p.SetContactPhone(m.ContactPhone), p.SetCustomerReference(m.CustomerReference), p.SetAttentionPerson(m.AttentionPerson), p.SetAttentionCompany(m.AttentionCompany), p.SetAttentionPhone(m.AttentionPhone), p.SetAttentionAddress(m.AttentionAddress), p.SetQuoteType(m.QuoteType), p.SetDivision(m.Division), p.SetTermsAndConditions(m.TermsAndConditions), p.SetSubject(m.Subject), p.SetBody(m.Body), p.SetCocCoo(m.CocCoo), p.SetTestCertificate(m.TestCertificate), p.SetInstallation(m.Installation), p.SetCommissioning(m.Commissioning), p.SetTesting(m.Testing), p.SetFolderNumber(m.FolderNumber), p.SetProjectName(m.ProjectName)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetRevisionNumber(int64(m.RevisionNumber))
	p.SetTotalValueBhd(m.TotalValueBHD)
	p.SetEstimatedMargin(m.EstimatedMargin)
	p.SetStage(opportunityStage(m.Stage))
	p.SetHasAbbCompetition(m.HasABBCompetition)
	p.SetDiscountPercent(m.DiscountPercent)
	p.SetVatRate(m.VatRate)
	return &p, nil
}

func OfferFromProto(p protocrm.Offer) (gormcrm.Offer, error) {
	base, err := p.Base()
	if err != nil {
		return gormcrm.Offer{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormcrm.Offer{}, err
	}
	m := gormcrm.Offer{Base: sharedBase}
	m.OfferNumber, _ = p.OfferNumber()
	m.CustomerID, _ = p.CustomerId()
	m.CustomerName, _ = p.CustomerName()
	m.TotalValueBHD = p.TotalValueBhd()
	m.EstimatedMargin = p.EstimatedMargin()
	m.Stage = stageText(p.Stage())
	m.PaymentTerms, _ = p.PaymentTerms()
	m.DeliveryTerms, _ = p.DeliveryTerms()
	m.VatRate = p.VatRate()
	return m, nil
}

func OfferItemToProto(m gormcrm.OfferItem) (*protocrm.OfferItem, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewOfferItem(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetOfferId(m.OfferID), p.SetProductId(m.ProductID), p.SetProductCode(m.ProductCode), p.SetModel(m.Model), p.SetDescription(m.Description), p.SetLongCode(m.LongCode), p.SetEquipment(m.Equipment), p.SetSpecification(m.Specification), p.SetDetailedDescription(m.DetailedDescription)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetLineNumber(int64(m.LineNumber))
	p.SetQuantity(m.Quantity)
	p.SetUnitPriceBhd(m.UnitPrice)
	p.SetCurrency(currencyCode(m.Currency))
	p.SetFob(m.FOB)
	p.SetFreight(m.Freight)
	p.SetTotalCost(m.TotalCost)
	p.SetMarginPercent(m.MarginPercent)
	p.SetTotalPrice(m.TotalPrice)
	return &p, nil
}

func OfferItemFromProto(p protocrm.OfferItem) (gormcrm.OfferItem, error) {
	m := gormcrm.OfferItem{}
	m.OfferID, _ = p.OfferId()
	m.LineNumber = int(p.LineNumber())
	m.ProductCode, _ = p.ProductCode()
	m.Description, _ = p.Description()
	m.Quantity = p.Quantity()
	m.UnitPrice = p.UnitPriceBhd()
	m.TotalPrice = p.TotalPrice()
	return m, nil
}

func OpportunityToProto(m gormcrm.Opportunity) (*protocrm.Opportunity, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewOpportunity(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetFolderNumber(m.FolderNumber), p.SetOfferId(m.OfferID), p.SetCustomerId(m.CustomerID), p.SetCustomerName(m.CustomerName), p.SetSalesperson(m.Salesperson), p.SetDivision(m.Division), p.SetFolderName(m.FolderName), p.SetTitle(m.Title), p.SetEhRef(m.EHRef), p.SetSource(m.Source), p.SetComment(m.Comment), p.SetOwnerNotes(m.OwnerNotes), p.SetProductDetails(m.ProductDetails), p.SetOfferDate(adapter.TimeToText(m.OfferDate)), p.SetOrderDate(adapter.TimePtrToText(m.OrderDate)), p.SetExpectedDate(adapter.TimePtrToText(m.ExpectedDate)), p.SetClosedDate(adapter.TimePtrToText(m.ClosedDate)), p.SetDeliveryTerms(m.DeliveryTerms), p.SetPaymentTerms(m.PaymentTerms), p.SetSpocStatus(m.SPOCStatus), p.SetWipStatus(m.WIPStatus), p.SetProductType(m.ProductType), p.SetWonReason(m.WonReason), p.SetLostReason(m.LostReason)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetCustomerGrade(grade(m.CustomerGrade))
	p.SetYear(int64(m.Year))
	p.SetOppNumber(int64(m.OppNumber))
	p.SetRevenueBhd(m.RevenueBHD)
	p.SetCostBhd(m.CostBHD)
	p.SetProfitBhd(m.ProfitBHD)
	p.SetStage(opportunityStage(m.Stage))
	p.SetRegime(int64(m.Regime))
	p.SetConfidence(m.Confidence)
	p.SetR1(m.R1)
	p.SetR2(m.R2)
	p.SetR3(m.R3)
	p.SetHasAbbCompetition(m.HasABBCompetition)
	return &p, nil
}

func OpportunityFromProto(p protocrm.Opportunity) (gormcrm.Opportunity, error) {
	m := gormcrm.Opportunity{}
	m.FolderNumber, _ = p.FolderNumber()
	m.CustomerID, _ = p.CustomerId()
	m.CustomerName, _ = p.CustomerName()
	m.Stage = stageText(p.Stage())
	m.RevenueBHD = p.RevenueBhd()
	m.ProfitBHD = p.ProfitBhd()
	return m, nil
}

func OrderToProto(m gormcrm.Order) (*protocrm.Order, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewOrder(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetOrderNumber(m.OrderNumber), p.SetCustomerPoNumber(m.CustomerPONumber), p.SetCustomerId(m.CustomerID), p.SetCustomerName(m.CustomerName), p.SetOrderDate(adapter.TimeToText(m.OrderDate)), p.SetRequiredDate(adapter.TimeToText(m.RequiredDate)), p.SetUpdatedBy(m.UpdatedBy), p.SetPaymentTerms(m.PaymentTerms), p.SetDeliveryTerms(m.DeliveryTerms), p.SetOfferId(m.OfferID), p.SetOfferNumber(m.OfferNumber), p.SetRfqId(m.RFQID), p.SetCustomerReference(m.CustomerReference), p.SetAttentionPerson(m.AttentionPerson), p.SetAttentionCompany(m.AttentionCompany), p.SetAttentionPhone(m.AttentionPhone), p.SetAttentionAddress(m.AttentionAddress), p.SetDeliveryWeeks(m.DeliveryWeeks), p.SetCountryOfOrigin(m.CountryOfOrigin), p.SetIssuedBy(m.IssuedBy), p.SetContactPhone(m.ContactPhone), p.SetDivision(m.Division)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetTotalValueBhd(m.TotalValueBHD)
	p.SetGrandTotalBhd(m.GrandTotalBHD)
	p.SetStatus(documentStatus(m.Status))
	p.SetDiscountPercent(m.DiscountPercent)
	return &p, nil
}

func OrderFromProto(p protocrm.Order) (gormcrm.Order, error) {
	base, err := p.Base()
	if err != nil {
		return gormcrm.Order{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormcrm.Order{}, err
	}
	m := gormcrm.Order{Base: sharedBase}
	m.OrderNumber, _ = p.OrderNumber()
	m.CustomerID, _ = p.CustomerId()
	m.CustomerName, _ = p.CustomerName()
	m.TotalValueBHD = p.TotalValueBhd()
	m.GrandTotalBHD = p.GrandTotalBhd()
	m.Status = statusText(p.Status())
	m.PaymentTerms, _ = p.PaymentTerms()
	return m, nil
}

func OrderItemToProto(m gormcrm.OrderItem) (*protocrm.OrderItem, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewOrderItem(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetOrderId(m.OrderID), p.SetProductId(m.ProductID), p.SetProductCode(m.ProductCode), p.SetDescription(m.Description), p.SetEquipment(m.Equipment), p.SetModel(m.Model), p.SetSpecification(m.Specification), p.SetDetailedDescription(m.DetailedDescription)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetLineNumber(int64(m.LineNumber))
	p.SetQuantity(m.Quantity)
	p.SetUnitPriceBhd(m.UnitPrice)
	p.SetQuantityShipped(m.QuantityShipped)
	p.SetQuantityInvoiced(m.QuantityInvoiced)
	p.SetCurrency(currencyCode(m.Currency))
	p.SetFob(m.FOB)
	p.SetFreight(m.Freight)
	p.SetTotalCost(m.TotalCost)
	p.SetMarginPercent(m.MarginPercent)
	p.SetTotalPrice(m.TotalPrice)
	return &p, nil
}

func OrderItemFromProto(p protocrm.OrderItem) (gormcrm.OrderItem, error) {
	m := gormcrm.OrderItem{}
	m.OrderID, _ = p.OrderId()
	m.LineNumber = int(p.LineNumber())
	m.ProductCode, _ = p.ProductCode()
	m.Description, _ = p.Description()
	m.Quantity = p.Quantity()
	m.UnitPrice = p.UnitPriceBhd()
	return m, nil
}

func DeliveryNoteToProto(m gormcrm.DeliveryNote) (*protocrm.DeliveryNote, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewDeliveryNote(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetOrderId(m.OrderID), p.SetCustomerId(m.CustomerID), p.SetDnNumber(m.DNNumber), p.SetDeliveryDate(adapter.TimeToText(m.DeliveryDate)), p.SetDeliveryAddress(m.DeliveryAddress), p.SetContactPerson(m.ContactPerson), p.SetContactPhone(m.ContactPhone), p.SetDriverName(m.DriverName), p.SetVehicleNumber(m.VehicleNumber), p.SetTransportMethod(m.TransportMethod), p.SetUpdatedBy(m.UpdatedBy), p.SetSignedBy(m.SignedBy), p.SetSignedAt(adapter.TimePtrToText(m.SignedAt)), p.SetSignatureImage(m.SignatureImage)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetStatus(deliveryStatus(m.Status))
	p.SetIsPartialDelivery(m.IsPartialDelivery)
	p.SetDeliverySequence(int64(m.DeliverySequence))
	p.SetTotalDeliveries(int64(m.TotalDeliveries))
	return &p, nil
}

func DeliveryNoteFromProto(p protocrm.DeliveryNote) (gormcrm.DeliveryNote, error) {
	m := gormcrm.DeliveryNote{}
	m.OrderID, _ = p.OrderId()
	m.CustomerID, _ = p.CustomerId()
	m.DNNumber, _ = p.DnNumber()
	m.Status = deliveryStatusText(p.Status())
	return m, nil
}

func deliveryStatus(status string) protocrm.DeliveryStatus {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(status), " ", "")) {
	case "dispatched":
		return protocrm.DeliveryStatus_dispatched
	case "intransit":
		return protocrm.DeliveryStatus_inTransit
	case "delivered":
		return protocrm.DeliveryStatus_delivered
	case "signed":
		return protocrm.DeliveryStatus_signed
	case "cancelled", "canceled":
		return protocrm.DeliveryStatus_cancelled
	default:
		return protocrm.DeliveryStatus_prepared
	}
}

func deliveryStatusText(status protocrm.DeliveryStatus) string {
	switch status {
	case protocrm.DeliveryStatus_dispatched:
		return "Dispatched"
	case protocrm.DeliveryStatus_inTransit:
		return "InTransit"
	case protocrm.DeliveryStatus_delivered:
		return "Delivered"
	case protocrm.DeliveryStatus_signed:
		return "Signed"
	case protocrm.DeliveryStatus_cancelled:
		return "Cancelled"
	default:
		return "Prepared"
	}
}

func DeliveryNoteItemToProto(m gormcrm.DeliveryNoteItem) (*protocrm.DeliveryNoteItem, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewDeliveryNoteItem(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetDeliveryNoteId(m.DeliveryNoteID), p.SetOrderItemId(m.OrderItemID), p.SetProductId(m.ProductID), p.SetProductCode(m.ProductCode), p.SetDescription(m.Description)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetQuantityOrdered(m.QuantityOrdered)
	p.SetQuantityDelivered(m.QuantityDelivered)
	p.SetQuantityRemaining(m.QuantityRemaining)
	return &p, nil
}

func DeliveryNoteItemFromProto(p protocrm.DeliveryNoteItem) (gormcrm.DeliveryNoteItem, error) {
	m := gormcrm.DeliveryNoteItem{}
	m.DeliveryNoteID, _ = p.DeliveryNoteId()
	m.ProductCode, _ = p.ProductCode()
	m.QuantityDelivered = p.QuantityDelivered()
	return m, nil
}

func SerialNumberToProto(m gormcrm.SerialNumber) (*protocrm.SerialNumber, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewSerialNumber(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetProductId(m.ProductID), p.SetProductCode(m.ProductCode), p.SetSerialNo(m.SerialNo), p.SetLotNumber(m.LotNumber), p.SetPoId(m.POID), p.SetPoNumber(m.PONumber), p.SetGrnItemId(m.GRNItemID), p.SetGrnNumber(m.GRNNumber), p.SetDnItemId(m.DNItemID), p.SetDnNumber(m.DNNumber), p.SetInvoiceId(m.InvoiceID), p.SetInvoiceNumber(m.InvoiceNumber), p.SetCustomerId(m.CustomerID), p.SetCustomerName(m.CustomerName), p.SetReceivedDate(adapter.TimePtrToText(m.ReceivedDate)), p.SetShippedDate(adapter.TimePtrToText(m.ShippedDate)), p.SetWarrantyStartDate(adapter.TimePtrToText(m.WarrantyStartDate)), p.SetWarrantyEndDate(adapter.TimePtrToText(m.WarrantyEndDate)), p.SetCalibrationDate(adapter.TimePtrToText(m.CalibrationDate)), p.SetCalibrationDueDate(adapter.TimePtrToText(m.CalibrationDueDate)), p.SetCalibrationCertPath(m.CalibrationCertPath), p.SetNotes(m.Notes)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetStatus(documentStatus(m.Status))
	p.SetWarrantyMonths(int64(m.WarrantyMonths))
	return &p, nil
}

func SerialNumberFromProto(p protocrm.SerialNumber) (gormcrm.SerialNumber, error) {
	m := gormcrm.SerialNumber{}
	m.ProductID, _ = p.ProductId()
	m.SerialNo, _ = p.SerialNo()
	m.Status = statusText(p.Status())
	return m, nil
}

func GoodsReceivedNoteToProto(m gormcrm.GoodsReceivedNote) (*protocrm.GoodsReceivedNote, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewGoodsReceivedNote(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetPurchaseOrderId(m.PurchaseOrderID), p.SetGrnNumber(m.GRNNumber), p.SetReceivedDate(adapter.TimeToText(m.ReceivedDate)), p.SetReceivedBy(m.ReceivedBy), p.SetWarehouseId(m.WarehouseID), p.SetSupplierDnNumber(m.SupplierDNNumber), p.SetQcNotes(m.QCNotes), p.SetQcDate(adapter.TimePtrToText(m.QCDate)), p.SetQcBy(m.QCBy), p.SetUpdatedBy(m.UpdatedBy)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetQcStatus(approvalStatus(m.QCStatus))
	return &p, nil
}

func GoodsReceivedNoteFromProto(p protocrm.GoodsReceivedNote) (gormcrm.GoodsReceivedNote, error) {
	m := gormcrm.GoodsReceivedNote{}
	m.PurchaseOrderID, _ = p.PurchaseOrderId()
	m.GRNNumber, _ = p.GrnNumber()
	m.QCStatus = approvalStatusText(p.QcStatus())
	return m, nil
}

func approvalStatus(status string) commonproto.ApprovalStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "passed", "matched":
		return commonproto.ApprovalStatus_approved
	case "rejected", "failed":
		return commonproto.ApprovalStatus_rejected
	case "cancelled", "canceled":
		return commonproto.ApprovalStatus_cancelled
	case "notrequired":
		return commonproto.ApprovalStatus_notRequired
	default:
		return commonproto.ApprovalStatus_pending
	}
}

func approvalStatusText(status commonproto.ApprovalStatus) string {
	switch status {
	case commonproto.ApprovalStatus_approved:
		return "Passed"
	case commonproto.ApprovalStatus_rejected:
		return "Failed"
	case commonproto.ApprovalStatus_cancelled:
		return "Cancelled"
	case commonproto.ApprovalStatus_notRequired:
		return "NotRequired"
	default:
		return "Pending"
	}
}

func GRNItemToProto(m gormcrm.GRNItem) (*protocrm.GRNItem, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protocrm.NewGRNItem(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetGrnId(m.GRNID), p.SetPoItemId(m.POItemID), p.SetProductId(m.ProductID), p.SetRejectionReason(m.RejectionReason)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetQuantityOrdered(m.QuantityOrdered)
	p.SetQuantityReceived(m.QuantityReceived)
	p.SetQuantityAccepted(m.QuantityAccepted)
	p.SetQuantityRejected(m.QuantityRejected)
	return &p, nil
}

func GRNItemFromProto(p protocrm.GRNItem) (gormcrm.GRNItem, error) {
	m := gormcrm.GRNItem{}
	m.GRNID, _ = p.GrnId()
	m.ProductID, _ = p.ProductId()
	m.QuantityReceived = p.QuantityReceived()
	return m, nil
}
